package salary

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/internal/salary/payslip"
	"github.com/turfaa/apotek-hris/pkg/vmedisproxy"
)

// journalStaleAfter is how long an entry can stay pending without an error
// before it is taken for one whose request died while posting it. It is
// longer than a post to vmedis-proxy can take.
const journalStaleAfter = 3 * time.Minute

// JournalProxy creates and finds journal entries in Vmedis, through
// vmedis-proxy-api.
type JournalProxy interface {
	CreateJournalEntry(ctx context.Context, idempotencyKey, email string, req vmedisproxy.CreateJournalEntryRequest) (vmedisproxy.CreatedJournalEntry, error)
	FindJournalEntries(ctx context.Context, email, date, query string) ([]vmedisproxy.JournalEntry, error)
}

// GetSnapshotJournal returns the snapshot's journal entries: the ones it can
// post, with the keterangan they'd get when posted by poster, merged with the
// recorded ones.
func (s *Service) GetSnapshotJournal(ctx context.Context, snapshotID int64, poster JournalPoster) (SnapshotJournal, error) {
	snapshot, err := s.db.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return SnapshotJournal{}, fmt.Errorf("get snapshot %d: %w", snapshotID, err)
	}

	name, err := s.employeeName(ctx, snapshot.EmployeeID)
	if err != nil {
		return SnapshotJournal{}, err
	}

	records, err := s.db.GetJournalEntries(ctx, s.db.db, snapshotID)
	if err != nil {
		return SnapshotJournal{}, err
	}

	otherID, err := s.db.GetOtherPostedSnapshotID(ctx, s.db.db, snapshot.EmployeeID, snapshot.Month, snapshotID)
	if err != nil {
		return SnapshotJournal{}, err
	}

	return s.assembleJournal(snapshot, name, poster, records, otherID, time.Now())
}

// PostSnapshotJournal posts the snapshot's salary entry and the repayments of
// the chosen utang components to Vmedis, the first time; afterwards it sends
// again only the entries that failed (or whose outcome is unknown). Entries
// are sent one by one, and each one's outcome is recorded, so a partial
// failure never loses the entries that were posted.
func (s *Service) PostSnapshotJournal(ctx context.Context, snapshotID int64, req PostSnapshotJournalRequest, poster JournalPoster) (SnapshotJournal, error) {
	if !s.journalConfigured() {
		return SnapshotJournal{}, ErrJournalNotConfigured
	}
	if strings.TrimSpace(poster.Email) == "" {
		return SnapshotJournal{}, &JournalRequestError{Message: "header X-Email wajib diisi"}
	}

	// Once started, a post runs to completion even when the client goes
	// away, so every outcome is recorded.
	ctx = context.WithoutCancel(ctx)

	snapshot, err := s.db.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return SnapshotJournal{}, fmt.Errorf("get snapshot %d: %w", snapshotID, err)
	}

	name, err := s.employeeName(ctx, snapshot.EmployeeID)
	if err != nil {
		return SnapshotJournal{}, err
	}

	date, err := journalDate(snapshot.Month)
	if err != nil {
		return SnapshotJournal{}, err
	}

	var toSend []claimedJournalEntry
	err = s.db.WithLockedSnapshot(ctx, snapshotID, func(tx *sqlx.Tx) error {
		toSend, err = s.claimJournalEntries(ctx, tx, snapshot, name, date, req, poster)
		return err
	})
	if err != nil {
		if errors.Is(err, errUniqueViolation) {
			return SnapshotJournal{}, &JournalConflictError{Message: s.postedElsewhereMessage(name, snapshot)}
		}
		return SnapshotJournal{}, err
	}

	for _, entry := range toSend {
		s.sendJournalEntry(ctx, entry, poster)
	}

	return s.GetSnapshotJournal(ctx, snapshotID, poster)
}

// claimedJournalEntry is a recorded entry this request claimed to send.
type claimedJournalEntry struct {
	record JournalEntryRecord
	// mayExist is set when an earlier attempt may have saved the entry in
	// Vmedis, so it is looked up before being sent again.
	mayExist bool
}

// claimJournalEntries records the entries to post (the first time) or claims
// the ones to send again, in tx, which holds the snapshot's lock.
func (s *Service) claimJournalEntries(
	ctx context.Context,
	tx *sqlx.Tx,
	snapshot Snapshot,
	employeeName string,
	date string,
	req PostSnapshotJournalRequest,
	poster JournalPoster,
) ([]claimedJournalEntry, error) {
	records, err := s.db.GetJournalEntries(ctx, tx, snapshot.ID)
	if err != nil {
		return nil, err
	}

	statuses := make([]string, len(records))
	for i, r := range records {
		statuses[i] = r.Status
	}

	now := time.Now()

	if journalStatusOf(statuses) != JournalStatusNone {
		// Something is in Vmedis: send again only what failed.
		var claimed []claimedJournalEntry
		for _, record := range records {
			retry, mayExist := retryable(record, now)
			if !retry {
				continue
			}

			ok, err := s.db.ClaimJournalEntry(ctx, tx, &record, now)
			if err != nil {
				return nil, err
			}
			if ok {
				claimed = append(claimed, claimedJournalEntry{record: record, mayExist: mayExist})
			}
		}

		if len(claimed) == 0 {
			return nil, &JournalConflictError{Message: "tidak ada jurnal yang perlu diposting ulang: semuanya sudah diposting atau sedang diproses"}
		}
		return claimed, nil
	}

	otherID, err := s.db.GetOtherPostedSnapshotID(ctx, tx, snapshot.EmployeeID, snapshot.Month, snapshot.ID)
	if err != nil {
		return nil, err
	}
	if otherID != nil {
		return nil, &JournalConflictError{Message: s.postedElsewhereMessage(employeeName, snapshot), OtherSnapshotID: otherID}
	}

	lines := buildJournalLines(snapshot, employeeName, payslip.PeriodName(snapshot.Month.Year, snapshot.Month.Month), poster, s.journal.Accounts)

	selected := map[int]bool{}
	for _, index := range req.DebtComponentIndexes {
		selected[index] = true
	}

	var toRecord []JournalLine
	for _, line := range lines {
		switch line.Kind {
		case JournalKindSalary:
			if !line.Postable {
				return nil, &JournalRequestError{Message: *line.Warning}
			}
			toRecord = append(toRecord, line)

		case JournalKindDebt:
			if !selected[*line.ComponentIndex] {
				continue
			}
			delete(selected, *line.ComponentIndex)

			if !line.Postable {
				return nil, &JournalRequestError{Message: fmt.Sprintf("utang %q tidak bisa diposting: %s", *line.ComponentDescription, *line.Warning)}
			}
			toRecord = append(toRecord, line)
		}
	}
	for index := range selected {
		return nil, &JournalRequestError{Message: fmt.Sprintf("komponen #%d bukan utang di snapshot ini", index)}
	}

	// The earlier attempts failed without saving anything; start over with
	// the entries chosen now.
	if err := s.db.DeleteFailedJournalEntries(ctx, tx, snapshot.ID); err != nil {
		return nil, err
	}

	journalDay, err := time.ParseInLocation(time.DateOnly, date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse journal date %q: %w", date, err)
	}

	claimed := make([]claimedJournalEntry, len(toRecord))
	for i, line := range toRecord {
		record := JournalEntryRecord{
			SnapshotID:        snapshot.ID,
			EmployeeID:        snapshot.EmployeeID,
			Month:             snapshot.Month,
			Kind:              line.Kind,
			ComponentIndex:    line.ComponentIndex,
			Description:       line.Description,
			CreditAccountCode: line.CreditAccount,
			DebitAccountCode:  line.DebitAccount,
			Amount:            line.Amount,
			JournalDate:       journalDay,
			Status:            JournalEntryPending,
			Attempts:          1,
			IdempotencyKey:    journalIdempotencyKey(snapshot.ID, line.Kind, line.ComponentIndex),
			PostedBy:          poster.Email,
			PostedByName:      strings.TrimSpace(poster.Name),
		}
		if err := s.db.InsertJournalEntry(ctx, tx, &record); err != nil {
			return nil, err
		}

		claimed[i] = claimedJournalEntry{record: record}
	}

	return claimed, nil
}

// retryable reports whether the recorded entry is sent again by a post, and
// whether an earlier attempt may have saved it in Vmedis.
func retryable(record JournalEntryRecord, now time.Time) (retry bool, mayExist bool) {
	switch record.Status {
	case JournalEntryFailed:
		return true, false
	case JournalEntryPending:
		// A pending entry with an error is one whose outcome is unknown; one
		// without has a request posting it, unless that request died.
		if record.Error != nil || now.Sub(record.UpdatedAt) >= journalStaleAfter {
			return true, true
		}
	}
	return false, false
}

// sendJournalEntry posts one claimed entry to Vmedis and records the outcome.
func (s *Service) sendJournalEntry(ctx context.Context, entry claimedJournalEntry, poster JournalPoster) {
	record := entry.record
	date := record.journalDateString()

	if entry.mayExist {
		existing, err := s.findPostedJournalEntry(ctx, record, poster)
		switch {
		case err != nil:
			s.finishJournalEntry(ctx, record, JournalEntryPending, fmt.Sprintf("gagal mengecek apakah jurnal sudah ada di Vmedis: %s", err), nil)
			return
		case existing != nil:
			log.Printf("Journal entry %s was posted to Vmedis as %s without an answer", record.IdempotencyKey, existing.VoucherNo)
			s.finishJournalEntry(ctx, record, JournalEntryPosted, "", &vmedisproxy.CreatedJournalEntry{VoucherNo: existing.VoucherNo, VmedisID: existing.VmedisID})
			return
		}
	}

	req := vmedisproxy.CreateJournalEntryRequest{
		Date:              date,
		Description:       record.Description,
		CreditAccountCode: record.CreditAccountCode,
		DebitAccountCode:  record.DebitAccountCode,
		Amount:            record.Amount.String(),
	}

	payload, err := json.Marshal(struct {
		IdempotencyKey string                                `json:"idempotencyKey"`
		Email          string                                `json:"email"`
		Request        vmedisproxy.CreateJournalEntryRequest `json:"request"`
	}{record.IdempotencyKey, poster.Email, req})
	if err == nil {
		err = s.db.RecordJournalEntryPayload(ctx, record, payload, time.Now())
	}
	if err != nil {
		// Nothing was sent yet.
		s.finishJournalEntry(ctx, record, JournalEntryFailed, err.Error(), nil)
		return
	}

	created, err := s.journalProxy.CreateJournalEntry(ctx, record.IdempotencyKey, poster.Email, req)
	switch {
	case err == nil:
		s.finishJournalEntry(ctx, record, JournalEntryPosted, "", &created)

	case vmedisproxy.IsStatus(err, http.StatusConflict):
		// Another request is posting it right now: leave it to that one.
		s.finishJournalEntry(ctx, record, JournalEntryPending, fmt.Sprintf("masih diproses di Vmedis, cek lagi sebentar lagi: %s", err), nil)

	case vmedisproxy.OutcomeKnown(err):
		s.finishJournalEntry(ctx, record, JournalEntryFailed, err.Error(), nil)

	default:
		// It may have been saved: it stays pending, which keeps the snapshot
		// from being deleted, and is looked up before being sent again.
		s.finishJournalEntry(ctx, record, JournalEntryPending, fmt.Sprintf("belum pasti apakah jurnal tersimpan di Vmedis, coba posting ulang: %s", err), nil)
	}
}

// findPostedJournalEntry looks for the record's entry in Vmedis, by its date,
// keterangan (which names the snapshot) and amount.
func (s *Service) findPostedJournalEntry(ctx context.Context, record JournalEntryRecord, poster JournalPoster) (*vmedisproxy.JournalEntry, error) {
	entries, err := s.journalProxy.FindJournalEntries(ctx, poster.Email, record.journalDateString(), fmt.Sprintf("(Snapshot #%d,", record.SnapshotID))
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		amount, err := decimal.NewFromString(e.Amount)
		if err != nil {
			continue
		}
		if strings.TrimSpace(e.Description) == strings.TrimSpace(record.Description) && amount.Equal(record.Amount) {
			return &e, nil
		}
	}
	return nil, nil
}

// finishJournalEntry records the outcome of the record's attempt. A non-empty
// errMessage is kept with a pending or failed status.
func (s *Service) finishJournalEntry(ctx context.Context, record JournalEntryRecord, status, errMessage string, created *vmedisproxy.CreatedJournalEntry) {
	now := time.Now()

	record.Status = status
	record.Error = nil
	if errMessage != "" {
		record.Error = &errMessage
	}
	if created != nil {
		record.VoucherNo = &created.VoucherNo
		record.VmedisID = &created.VmedisID
		record.PostedAt = &now
	}

	ok, err := s.db.FinishJournalEntry(ctx, record, now)
	switch {
	case err != nil:
		log.Printf("Recording journal entry %s as %s (%s): %s", record.IdempotencyKey, status, errMessage, err)
	case !ok:
		log.Printf("Journal entry %s was claimed by another request before its outcome (%s %s) was recorded", record.IdempotencyKey, status, errMessage)
	}
}

func (s *Service) assembleJournal(
	snapshot Snapshot,
	employeeName string,
	poster JournalPoster,
	records []JournalEntryRecord,
	otherPostedSnapshotID *int64,
	now time.Time,
) (SnapshotJournal, error) {
	date, err := journalDate(snapshot.Month)
	if err != nil {
		return SnapshotJournal{}, err
	}

	lines := buildJournalLines(snapshot, employeeName, payslip.PeriodName(snapshot.Month.Year, snapshot.Month.Month), poster, s.journal.Accounts)

	byKey := make(map[string]JournalEntryRecord, len(records))
	statuses := make([]string, len(records))
	for i, r := range records {
		byKey[r.lineKey()] = r
		statuses[i] = r.Status
	}

	journal := SnapshotJournal{
		SnapshotID:        snapshot.ID,
		Status:            journalStatusOf(statuses),
		JournalDate:       date,
		SnapshotCreatedAt: snapshot.CreatedAt,
		Lines:             lines,
	}

	canRetry := false
	for i := range journal.Lines {
		line := &journal.Lines[i]

		record, ok := byKey[lineKey(line.Kind, line.ComponentIndex)]
		if !ok {
			continue
		}

		// The entry as it was recorded, and as it is in Vmedis.
		line.Description = record.Description
		line.CreditAccount = record.CreditAccountCode
		line.DebitAccount = record.DebitAccountCode
		line.Amount = record.Amount
		line.Entry = &JournalLineEntry{
			Status:    record.Status,
			VoucherNo: record.VoucherNo,
			Error:     record.Error,
			PostedAt:  record.PostedAt,
		}

		if retry, _ := retryable(record, now); retry {
			canRetry = true
		}

		if record.Status == JournalEntryPosted && record.PostedAt != nil && (journal.PostedAt == nil || record.PostedAt.Before(*journal.PostedAt)) {
			journal.PostedAt = record.PostedAt
		}
	}

	// Who posted: the one who recorded the salary entry, which is recorded
	// with the first post.
	for _, r := range records {
		if r.Status == JournalEntryPosted || r.Status == JournalEntryPending {
			journal.PostedBy = new(r.PostedBy)
			if r.PostedByName != "" {
				journal.PostedByName = new(r.PostedByName)
			}
			break
		}
	}

	blocked := func(reason string) {
		journal.CanPost = false
		journal.BlockedReason = &reason
	}

	switch {
	case !s.journalConfigured():
		blocked(ErrJournalNotConfigured.Error())

	case journal.Status != JournalStatusNone:
		journal.CanPost = canRetry

	case otherPostedSnapshotID != nil:
		journal.OtherPostedSnapshotID = otherPostedSnapshotID
		blocked(s.postedElsewhereMessage(employeeName, snapshot))

	case !journal.Lines[0].Postable:
		blocked(*journal.Lines[0].Warning)

	default:
		journal.CanPost = true
	}

	return journal, nil
}

func (s *Service) postedElsewhereMessage(employeeName string, snapshot Snapshot) string {
	return fmt.Sprintf("Gaji %s %s sudah diposting dari snapshot lain", employeeName, payslip.PeriodName(snapshot.Month.Year, snapshot.Month.Month))
}

func (s *Service) journalConfigured() bool {
	return s.journalProxy != nil && s.journal.Accounts.configured()
}
