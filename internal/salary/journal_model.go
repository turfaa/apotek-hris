package salary

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/pkg/timex"
)

// JournalConfig configures posting salary snapshots to Vmedis journals.
type JournalConfig struct {
	Accounts JournalAccounts `mapstructure:"accounts"`
}

// JournalAccounts are the Vmedis account codes (akkode) salary journal
// entries use.
type JournalAccounts struct {
	// Bank pays the salaries and receives debt repayments, e.g. 111004 /
	// Bank OCBC.
	Bank string `mapstructure:"bank"`
	// SalaryExpense is debited with the salaries, e.g. 610002 / Gaji
	// Karyawan.
	SalaryExpense string `mapstructure:"salary_expense"`
	// EmployeeReceivable is credited with debt repayments, e.g. 171001 /
	// Piutang Pegawai.
	EmployeeReceivable string `mapstructure:"employee_receivable"`
}

func (a JournalAccounts) configured() bool {
	return a.Bank != "" && a.SalaryExpense != "" && a.EmployeeReceivable != ""
}

// Kinds of journal lines.
const (
	JournalKindSalary = "salary"
	JournalKindDebt   = "debt"
)

// Statuses of a recorded journal entry.
const (
	JournalEntryPending = "pending"
	JournalEntryPosted  = "posted"
	JournalEntryFailed  = "failed"
)

// Journal statuses of a snapshot.
const (
	// JournalStatusNone is a snapshot with nothing posted (or being posted).
	JournalStatusNone = "none"
	// JournalStatusPosted is a snapshot whose recorded entries are all
	// posted.
	JournalStatusPosted = "posted"
	// JournalStatusPartial is a snapshot with entries posted (or being
	// posted) and entries that failed or are still being posted.
	JournalStatusPartial = "partial"
)

var (
	// ErrSnapshotPosted is returned when deleting a snapshot that has journal
	// entries in Vmedis.
	ErrSnapshotPosted = errors.New("snapshot sudah diposting ke jurnal Vmedis dan tidak bisa dihapus; perbaiki jurnalnya di Vmedis")
	// ErrJournalNotConfigured is returned when the journal accounts or the
	// vmedis-proxy URL are not configured.
	ErrJournalNotConfigured = errors.New("posting jurnal Vmedis belum dikonfigurasi")
)

// JournalConflictError is returned when the snapshot can't be posted because
// of the state of its or another snapshot's journal entries.
type JournalConflictError struct {
	Message string
	// OtherSnapshotID is the snapshot of the same employee and month that was
	// posted, when that is the conflict.
	OtherSnapshotID *int64
}

func (e *JournalConflictError) Error() string {
	return e.Message
}

// JournalRequestError is an invalid posting request.
type JournalRequestError struct {
	Message string
}

func (e *JournalRequestError) Error() string {
	return e.Message
}

// SnapshotJournal is a snapshot's journal entries: the ones it can post, and
// the ones recorded.
type SnapshotJournal struct {
	SnapshotID int64  `json:"snapshotID"`
	Status     string `json:"status"`
	// PostedBy and PostedByName are of the admin who posted the snapshot,
	// PostedAt is when its first entry was posted. Nil until something is
	// posted.
	PostedBy     *string    `json:"postedBy"`
	PostedByName *string    `json:"postedByName"`
	PostedAt     *time.Time `json:"postedAt"`
	// JournalDate is the date every entry is dated: the last day of the
	// snapshot's month, YYYY-MM-DD.
	JournalDate       string    `json:"journalDate"`
	SnapshotCreatedAt time.Time `json:"snapshotCreatedAt"`
	// CanPost is whether a POST would send anything: true while nothing is
	// posted and posting is not blocked, or while recorded entries failed.
	CanPost bool `json:"canPost"`
	// BlockedReason says why the snapshot can't be posted, when it can't.
	BlockedReason *string `json:"blockedReason"`
	// OtherPostedSnapshotID is another snapshot of the same employee and
	// month that has entries posted, which blocks this one.
	OtherPostedSnapshotID *int64        `json:"otherPostedSnapshotID"`
	Lines                 []JournalLine `json:"lines"`
}

// JournalLine is a journal entry the snapshot posts, or can post.
type JournalLine struct {
	Kind string `json:"kind"`
	// ComponentIndex is the index of the utang component in the snapshot's
	// components, for debt lines.
	ComponentIndex *int `json:"componentIndex"`
	// ComponentDescription is the utang component's description, for debt
	// lines.
	ComponentDescription *string `json:"componentDescription"`
	// Description is the keterangan the entry has (or would have) in Vmedis.
	Description   string          `json:"description"`
	CreditAccount string          `json:"creditAccount"`
	DebitAccount  string          `json:"debitAccount"`
	Amount        decimal.Decimal `json:"amount"`
	// Postable is false for a line that is never posted, e.g. an utang
	// component with a positive total (a loan paid out, not a repayment).
	Postable bool `json:"postable"`
	// Warning explains why a line is not postable.
	Warning *string `json:"warning"`
	// Entry is the recorded entry, nil when the line was not posted.
	Entry *JournalLineEntry `json:"entry"`
}

// JournalLineEntry is the recorded outcome of a journal line.
type JournalLineEntry struct {
	Status    string     `json:"status"`
	VoucherNo *string    `json:"voucherNo"`
	Error     *string    `json:"error"`
	PostedAt  *time.Time `json:"postedAt"`
}

// PostSnapshotJournalRequest posts a snapshot's journal entries.
type PostSnapshotJournalRequest struct {
	// DebtComponentIndexes are the utang components to post repayments for.
	// Ignored once anything is posted: then only failed entries are sent
	// again.
	DebtComponentIndexes []int `json:"debtComponentIndexes"`
}

// JournalPoster is the admin posting a snapshot.
type JournalPoster struct {
	Email string
	// Name is the admin's name, or empty.
	Name string
}

func (p JournalPoster) displayName() string {
	if name := strings.TrimSpace(p.Name); name != "" {
		return name
	}
	return p.Email
}

// JournalEntryRecord is a row of salary_snapshot_journal_entries.
type JournalEntryRecord struct {
	ID                int64           `db:"id"`
	SnapshotID        int64           `db:"snapshot_id"`
	EmployeeID        int64           `db:"employee_id"`
	Month             timex.Month     `db:"month"`
	Kind              string          `db:"kind"`
	ComponentIndex    *int            `db:"component_index"`
	Description       string          `db:"description"`
	CreditAccountCode string          `db:"credit_account_code"`
	DebitAccountCode  string          `db:"debit_account_code"`
	Amount            decimal.Decimal `db:"amount"`
	JournalDate       time.Time       `db:"journal_date"`
	Status            string          `db:"status"`
	Attempts          int             `db:"attempts"`
	IdempotencyKey    string          `db:"idempotency_key"`
	Payload           []byte          `db:"payload"`
	VoucherNo         *string         `db:"vmedis_voucher_no"`
	VmedisID          *int64          `db:"vmedis_id"`
	Error             *string         `db:"error"`
	PostedBy          string          `db:"posted_by"`
	PostedByName      string          `db:"posted_by_name"`
	PostedAt          *time.Time      `db:"posted_at"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
}

func (r JournalEntryRecord) lineKey() string {
	return lineKey(r.Kind, r.ComponentIndex)
}

func (r JournalEntryRecord) journalDateString() string {
	return r.JournalDate.Format(time.DateOnly)
}

func lineKey(kind string, componentIndex *int) string {
	if componentIndex == nil {
		return kind
	}
	return fmt.Sprintf("%s-%d", kind, *componentIndex)
}

// journalIdempotencyKey is the key a line is sent to vmedis-proxy with, the
// same on every attempt.
func journalIdempotencyKey(snapshotID int64, kind string, componentIndex *int) string {
	index := 0
	if componentIndex != nil {
		index = *componentIndex
	}
	return fmt.Sprintf("hris-snapshot-%d-%s-%d", snapshotID, kind, index)
}

// isDebtComponent is the rule TotalWithoutDebt leaves components out by.
func isDebtComponent(c Component) bool {
	return strings.Contains(strings.ToLower(c.Description), "utang")
}

// journalDate is the last day of the month.
func journalDate(month timex.Month) (string, error) {
	_, to, err := month.DateRange()
	if err != nil {
		return "", fmt.Errorf("month date range of %s: %w", month, err)
	}
	return to.String(), nil
}

// buildJournalLines returns the lines a snapshot can post, with the
// keterangan they get when posted by poster.
func buildJournalLines(snapshot Snapshot, employeeName, periodName string, poster JournalPoster, accounts JournalAccounts) []JournalLine {
	reference := fmt.Sprintf("(Snapshot #%d, diposting oleh %s)", snapshot.ID, poster.displayName())

	salaryAmount := snapshot.Salary.TotalWithoutDebt()
	salary := JournalLine{
		Kind:          JournalKindSalary,
		Description:   fmt.Sprintf("Gaji %s %s %s", employeeName, periodName, reference),
		CreditAccount: accounts.Bank,
		DebitAccount:  accounts.SalaryExpense,
		Amount:        salaryAmount,
		Postable:      salaryAmount.IsPositive(),
	}
	if !salary.Postable {
		salary.Warning = new("Total Tanpa Utang tidak lebih dari 0, tidak ada gaji yang bisa dijurnal")
	}

	lines := []JournalLine{salary}

	// Each entry's keterangan must be unique, so it can be found in Vmedis by
	// it: a repeated utang description gets a number.
	seen := map[string]int{}
	for i, c := range snapshot.Salary.Components {
		if !isDebtComponent(c) {
			continue
		}

		description := strings.TrimSpace(c.Description)
		seen[strings.ToLower(description)]++
		label := description
		if n := seen[strings.ToLower(description)]; n > 1 {
			label = fmt.Sprintf("%s (%d)", description, n)
		}

		total := c.Total()
		line := JournalLine{
			Kind:                 JournalKindDebt,
			ComponentIndex:       new(i),
			ComponentDescription: new(c.Description),
			Description:          fmt.Sprintf("Pelunasan %s - %s %s %s", label, employeeName, periodName, reference),
			CreditAccount:        accounts.EmployeeReceivable,
			DebitAccount:         accounts.Bank,
			Amount:               total.Abs(),
			Postable:             total.IsNegative(),
		}

		switch {
		case total.IsPositive():
			line.Warning = new("Totalnya positif (pinjaman yang dibayarkan lewat gaji), bukan pelunasan utang, jadi tidak diposting")
		case total.IsZero():
			line.Warning = new("Totalnya 0, tidak ada yang dijurnal")
		}

		lines = append(lines, line)
	}

	return lines
}

// journalStatusOf returns a snapshot's journal status from its recorded
// entries' statuses.
func journalStatusOf(statuses []string) string {
	var inVmedis, notPosted int
	for _, status := range statuses {
		if status == JournalEntryPosted || status == JournalEntryPending {
			inVmedis++
		}
		if status != JournalEntryPosted {
			notPosted++
		}
	}

	switch {
	case inVmedis == 0:
		return JournalStatusNone
	case notPosted == 0:
		return JournalStatusPosted
	default:
		return JournalStatusPartial
	}
}
