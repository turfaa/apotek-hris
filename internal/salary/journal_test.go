package salary

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/turfaa/apotek-hris/internal/hris"
	"github.com/turfaa/apotek-hris/internal/salary/payslip"
	"github.com/turfaa/apotek-hris/pkg/database"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"github.com/turfaa/apotek-hris/pkg/vmedisproxy"
)

var testAccounts = JournalAccounts{Bank: "111004", SalaryExpense: "610002", EmployeeReceivable: "171001"}

func TestJournalDate(t *testing.T) {
	for month, want := range map[timex.Month]string{
		timex.NewMonth(2026, 10): "2026-10-31",
		timex.NewMonth(2026, 2):  "2026-02-28",
		timex.NewMonth(2028, 2):  "2028-02-29",
		timex.NewMonth(2026, 11): "2026-11-30",
	} {
		got, err := journalDate(month)
		if err != nil || got != want {
			t.Errorf("journalDate(%s) = %q, %v; want %q", month, got, err, want)
		}
	}
}

func TestBuildJournalLines(t *testing.T) {
	snapshot := Snapshot{ID: 42, Salary: Salary{Components: []Component{
		{Description: "Gaji Pokok", Amount: decimal.NewFromInt(3_000_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Utang Kasbon", Amount: decimal.NewFromInt(-150_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Bonus", Amount: decimal.NewFromInt(200_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "utang kasbon", Amount: decimal.NewFromInt(-50_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Pinjaman Utang", Amount: decimal.NewFromInt(500_000), Multiplier: decimal.NewFromInt(1)},
	}}}

	lines := buildJournalLines(snapshot, "Budi", "Oktober 2026", JournalPoster{Email: "admin@example.com", Name: "Ani"}, testAccounts)
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4: %+v", len(lines), lines)
	}

	salary := lines[0]
	if salary.Description != "Gaji Budi Oktober 2026 (Snapshot #42, diposting oleh Ani)" ||
		salary.CreditAccount != "111004" || salary.DebitAccount != "610002" ||
		!salary.Amount.Equal(decimal.NewFromInt(3_200_000)) || !salary.Postable {
		t.Errorf("got salary line %+v", salary)
	}

	debt := lines[1]
	if debt.Description != "Pelunasan Utang Kasbon - Budi Oktober 2026 (Snapshot #42, diposting oleh Ani)" ||
		*debt.ComponentIndex != 1 || debt.CreditAccount != "171001" || debt.DebitAccount != "111004" ||
		!debt.Amount.Equal(decimal.NewFromInt(150_000)) || !debt.Postable {
		t.Errorf("got debt line %+v", debt)
	}

	// A repeated description gets a number, so each keterangan is unique.
	if lines[2].Description != "Pelunasan utang kasbon (2) - Budi Oktober 2026 (Snapshot #42, diposting oleh Ani)" {
		t.Errorf("got repeated debt line %q", lines[2].Description)
	}

	// A positive utang is a loan paid out: shown, never posted.
	if lines[3].Postable || lines[3].Warning == nil || *lines[3].ComponentIndex != 4 {
		t.Errorf("got positive utang line %+v", lines[3])
	}

	// Without a name, the poster is named by email.
	lines = buildJournalLines(snapshot, "Budi", "Oktober 2026", JournalPoster{Email: "admin@example.com"}, testAccounts)
	if !strings.HasSuffix(lines[0].Description, "(Snapshot #42, diposting oleh admin@example.com)") {
		t.Errorf("got %q", lines[0].Description)
	}
}

func TestJournalStatusOf(t *testing.T) {
	for _, tc := range []struct {
		statuses []string
		want     string
	}{
		{nil, JournalStatusNone},
		{[]string{"failed", "failed"}, JournalStatusNone},
		{[]string{"posted", "posted"}, JournalStatusPosted},
		{[]string{"posted", "failed"}, JournalStatusPartial},
		{[]string{"pending"}, JournalStatusPartial},
	} {
		if got := journalStatusOf(tc.statuses); got != tc.want {
			t.Errorf("journalStatusOf(%v) = %s, want %s", tc.statuses, got, tc.want)
		}
	}
}

// fakeProxy is a vmedis-proxy that keeps the created entries.
type fakeProxy struct {
	mu sync.Mutex

	entries map[string]vmedisproxy.JournalEntry // by idempotency key
	// fail returns an error for the request with the key, if set.
	fail func(key string) error
	// saveDespiteFail saves the entry even when fail returns an error.
	saveDespiteFail bool

	creates, finds int
}

func newFakeProxy() *fakeProxy {
	return &fakeProxy{entries: map[string]vmedisproxy.JournalEntry{}}
}

func (p *fakeProxy) CreateJournalEntry(_ context.Context, key, email string, req vmedisproxy.CreateJournalEntryRequest) (vmedisproxy.CreatedJournalEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.creates++

	var err error
	if p.fail != nil {
		err = p.fail(key)
	}
	if err != nil && !p.saveDespiteFail {
		return vmedisproxy.CreatedJournalEntry{}, err
	}

	if email == "" {
		return vmedisproxy.CreatedJournalEntry{}, &vmedisproxy.Error{Status: http.StatusForbidden, Message: "unauthorized"}
	}

	entry, ok := p.entries[key]
	if !ok {
		n := len(p.entries) + 1
		entry = vmedisproxy.JournalEntry{
			VmedisID:    int64(1000 + n),
			VoucherNo:   fmt.Sprintf("JRLM%s%04d", strings.ReplaceAll(req.Date[2:], "-", ""), n),
			Description: req.Description,
			Amount:      req.Amount,
		}
		p.entries[key] = entry
	}

	if err != nil {
		return vmedisproxy.CreatedJournalEntry{}, err
	}
	return vmedisproxy.CreatedJournalEntry{VoucherNo: entry.VoucherNo, VmedisID: entry.VmedisID, Replayed: ok}, nil
}

func (p *fakeProxy) FindJournalEntries(_ context.Context, _, _, query string) ([]vmedisproxy.JournalEntry, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finds++

	var res []vmedisproxy.JournalEntry
	for _, e := range p.entries {
		if strings.Contains(e.Description, query) {
			res = append(res, e)
		}
	}
	return res, nil
}

var (
	testDBOnce sync.Once
	testDB     *sqlx.DB
	testDBErr  error
)

// openJournalTestDB connects to the migrated postgres database of
// HRIS_TEST_POSTGRES_URL (e.g. postgres://postgres:test@localhost:55432/hris?sslmode=disable),
// skipping the test when it isn't set.
func openJournalTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dbURL := os.Getenv("HRIS_TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Skip("HRIS_TEST_POSTGRES_URL is not set")
	}

	testDBOnce.Do(func() {
		m, err := migrate.New("file://../../migrations", dbURL)
		if err != nil {
			testDBErr = fmt.Errorf("migrate.New: %w", err)
			return
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			testDBErr = fmt.Errorf("migrate up: %w", err)
			return
		}

		cfg := parseTestDBURL(dbURL)
		testDB, testDBErr = database.NewPostgresConnection(context.Background(), cfg)
	})
	if testDBErr != nil {
		t.Fatalf("open test database: %v", testDBErr)
	}
	return testDB
}

func parseTestDBURL(dbURL string) database.Config {
	// postgres://user:password@host:port/db?sslmode=mode
	rest := strings.TrimPrefix(dbURL, "postgres://")
	userinfo, rest, _ := strings.Cut(rest, "@")
	user, password, _ := strings.Cut(userinfo, ":")
	hostport, rest, _ := strings.Cut(rest, "/")
	host, portStr, _ := strings.Cut(hostport, ":")
	dbName, query, _ := strings.Cut(rest, "?")
	port, _ := strconv.Atoi(portStr)
	return database.Config{Host: host, Port: port, User: user, Password: password, DBName: dbName, SSLMode: strings.TrimPrefix(query, "sslmode=")}
}

type journalTestEnv struct {
	service    *Service
	proxy      *fakeProxy
	db         *sqlx.DB
	employeeID int64
	poster     JournalPoster
}

func newJournalTestEnv(t *testing.T) *journalTestEnv {
	t.Helper()

	db := openJournalTestDB(t)
	proxy := newFakeProxy()

	env := &journalTestEnv{
		service: NewService(db, hris.NewService(db), nil, payslip.Letterhead{}, JournalConfig{Accounts: testAccounts}, proxy),
		proxy:   proxy,
		db:      db,
		poster:  JournalPoster{Email: "admin@example.com", Name: "Admin Satu"},
	}

	if err := db.Get(&env.employeeID, `INSERT INTO employees (name, shift_fee) VALUES ($1, 0) RETURNING id`, "Budi "+t.Name()); err != nil {
		t.Fatalf("insert employee: %v", err)
	}
	return env
}

// snapshot creates a snapshot of the employee's October 2026 salary: 3.2M
// without debt, and two utang of 150k and 50k.
func (env *journalTestEnv) snapshot(t *testing.T) Snapshot {
	t.Helper()

	snapshot, err := env.service.db.CreateSnapshot(context.Background(), env.employeeID, timex.NewMonth(2026, 10), Salary{Components: []Component{
		{Description: "Gaji Pokok", Amount: decimal.NewFromInt(3_000_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Utang Kasbon", Amount: decimal.NewFromInt(-150_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Bonus", Amount: decimal.NewFromInt(200_000), Multiplier: decimal.NewFromInt(1)},
		{Description: "Utang HP", Amount: decimal.NewFromInt(-50_000), Multiplier: decimal.NewFromInt(1)},
	}})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	return snapshot
}

func (env *journalTestEnv) journal(t *testing.T, snapshotID int64) SnapshotJournal {
	t.Helper()

	journal, err := env.service.GetSnapshotJournal(context.Background(), snapshotID, env.poster)
	if err != nil {
		t.Fatalf("GetSnapshotJournal: %v", err)
	}
	return journal
}

func TestPostSnapshotJournal(t *testing.T) {
	env := newJournalTestEnv(t)
	snapshot := env.snapshot(t)
	ctx := context.Background()

	before := env.journal(t, snapshot.ID)
	if before.Status != JournalStatusNone || !before.CanPost || before.JournalDate != "2026-10-31" || len(before.Lines) != 3 {
		t.Fatalf("got journal before posting %+v", before)
	}

	journal, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{DebtComponentIndexes: []int{1}}, env.poster)
	if err != nil {
		t.Fatalf("PostSnapshotJournal: %v", err)
	}

	if journal.Status != JournalStatusPosted || journal.CanPost || *journal.PostedBy != "admin@example.com" || *journal.PostedByName != "Admin Satu" || journal.PostedAt == nil {
		t.Errorf("got journal %+v", journal)
	}

	salary, debt, unticked := journal.Lines[0], journal.Lines[1], journal.Lines[2]
	if salary.Entry == nil || salary.Entry.Status != JournalEntryPosted || *salary.Entry.VoucherNo == "" || !salary.Amount.Equal(decimal.NewFromInt(3_200_000)) {
		t.Errorf("got salary line %+v (entry %+v)", salary, salary.Entry)
	}
	if debt.Entry == nil || debt.Entry.Status != JournalEntryPosted || !debt.Amount.Equal(decimal.NewFromInt(150_000)) || debt.CreditAccount != "171001" {
		t.Errorf("got debt line %+v (entry %+v)", debt, debt.Entry)
	}
	if unticked.Entry != nil || *unticked.ComponentIndex != 3 {
		t.Errorf("got unticked debt line %+v", unticked)
	}

	entry := env.proxy.entries[fmt.Sprintf("hris-snapshot-%d-salary-0", snapshot.ID)]
	if entry.Amount != "3200000" || !strings.Contains(entry.Description, fmt.Sprintf("(Snapshot #%d, diposting oleh Admin Satu)", snapshot.ID)) {
		t.Errorf("got salary entry in vmedis %+v", entry)
	}

	// Listed as posted, can't be posted again or deleted.
	listed, err := env.service.GetSnapshots(ctx, GetSnapshotsRequest{EmployeeID: &env.employeeID})
	if err != nil || len(listed) != 1 || listed[0].JournalStatus != JournalStatusPosted {
		t.Errorf("got listed snapshots %+v, %v", listed, err)
	}

	var conflict *JournalConflictError
	if _, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, env.poster); !errors.As(err, &conflict) {
		t.Errorf("got %v posting again, want a JournalConflictError", err)
	}
	if err := env.service.DeleteSnapshot(ctx, snapshot.ID); !errors.Is(err, ErrSnapshotPosted) {
		t.Errorf("got %v deleting, want ErrSnapshotPosted", err)
	}
	if env.proxy.creates != 2 {
		t.Errorf("got %d creates, want 2", env.proxy.creates)
	}

	// Another snapshot of the month can't be posted.
	other := env.snapshot(t)
	otherJournal := env.journal(t, other.ID)
	if otherJournal.CanPost || otherJournal.OtherPostedSnapshotID == nil || *otherJournal.OtherPostedSnapshotID != snapshot.ID {
		t.Errorf("got other snapshot's journal %+v", otherJournal)
	}
	if _, err := env.service.PostSnapshotJournal(ctx, other.ID, PostSnapshotJournalRequest{}, env.poster); !errors.As(err, &conflict) || conflict.OtherSnapshotID == nil {
		t.Errorf("got %v posting another snapshot, want a JournalConflictError naming the posted one", err)
	}
}

func TestPostSnapshotJournal_PartialFailureRetriesOnlyFailed(t *testing.T) {
	env := newJournalTestEnv(t)
	snapshot := env.snapshot(t)
	ctx := context.Background()

	debtKey := fmt.Sprintf("hris-snapshot-%d-debt-3", snapshot.ID)
	env.proxy.fail = func(key string) error {
		if key == debtKey {
			return &vmedisproxy.Error{Status: http.StatusBadGateway, Message: "vmedis rejected the request"}
		}
		return nil
	}

	journal, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{DebtComponentIndexes: []int{1, 3}}, env.poster)
	if err != nil {
		t.Fatalf("PostSnapshotJournal: %v", err)
	}
	if journal.Status != JournalStatusPartial || !journal.CanPost {
		t.Fatalf("got journal %+v", journal)
	}
	if e := journal.Lines[2].Entry; e == nil || e.Status != JournalEntryFailed || e.Error == nil {
		t.Fatalf("got failed line entry %+v", e)
	}

	env.proxy.fail = nil
	journal, err = env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, env.poster)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if journal.Status != JournalStatusPosted {
		t.Errorf("got status %s after retry, want posted", journal.Status)
	}
	if env.proxy.creates != 4 || len(env.proxy.entries) != 3 {
		t.Errorf("got %d creates, %d entries; want 4 (3 + the retry), 3", env.proxy.creates, len(env.proxy.entries))
	}
}

func TestPostSnapshotJournal_UnknownOutcomeIsLookedUp(t *testing.T) {
	env := newJournalTestEnv(t)
	snapshot := env.snapshot(t)
	ctx := context.Background()

	// The salary is saved, but the answer is lost.
	env.proxy.fail = func(key string) error { return errors.New("context deadline exceeded") }
	env.proxy.saveDespiteFail = true

	journal, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, env.poster)
	if err != nil {
		t.Fatalf("PostSnapshotJournal: %v", err)
	}
	if e := journal.Lines[0].Entry; journal.Status != JournalStatusPartial || e == nil || e.Status != JournalEntryPending || e.Error == nil {
		t.Fatalf("got journal %+v (entry %+v), want the salary pending with an error", journal, journal.Lines[0].Entry)
	}
	if !journal.CanPost {
		t.Error("got canPost false, want a retry allowed")
	}
	if err := env.service.DeleteSnapshot(ctx, snapshot.ID); !errors.Is(err, ErrSnapshotPosted) {
		t.Errorf("got %v deleting, want ErrSnapshotPosted while the outcome is unknown", err)
	}

	env.proxy.fail = nil
	env.proxy.saveDespiteFail = false
	journal, err = env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, env.poster)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if journal.Status != JournalStatusPosted || *journal.Lines[0].Entry.VoucherNo == "" {
		t.Errorf("got journal %+v after retry", journal)
	}
	if env.proxy.finds != 1 || env.proxy.creates != 1 {
		t.Errorf("got %d finds, %d creates; want the retry to find the entry instead of sending it", env.proxy.finds, env.proxy.creates)
	}
}

func TestPostSnapshotJournal_AllFailedStartsOver(t *testing.T) {
	env := newJournalTestEnv(t)
	snapshot := env.snapshot(t)
	ctx := context.Background()

	env.proxy.fail = func(string) error { return &vmedisproxy.Error{Status: http.StatusForbidden, Message: "unauthorized"} }
	journal, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{DebtComponentIndexes: []int{1}}, env.poster)
	if err != nil {
		t.Fatalf("PostSnapshotJournal: %v", err)
	}
	if journal.Status != JournalStatusNone || !journal.CanPost {
		t.Fatalf("got journal %+v, want nothing posted and posting allowed", journal)
	}

	// The utang can be unticked when posting again.
	env.proxy.fail = nil
	journal, err = env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, env.poster)
	if err != nil {
		t.Fatalf("PostSnapshotJournal again: %v", err)
	}
	if journal.Status != JournalStatusPosted || journal.Lines[1].Entry != nil {
		t.Errorf("got journal %+v, want only the salary posted", journal)
	}
}

func TestPostSnapshotJournal_Validation(t *testing.T) {
	env := newJournalTestEnv(t)
	snapshot := env.snapshot(t)
	ctx := context.Background()

	var requestErr *JournalRequestError
	if _, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{DebtComponentIndexes: []int{0}}, env.poster); !errors.As(err, &requestErr) {
		t.Errorf("got %v for a non-utang component, want a JournalRequestError", err)
	}
	if _, err := env.service.PostSnapshotJournal(ctx, snapshot.ID, PostSnapshotJournalRequest{}, JournalPoster{}); !errors.As(err, &requestErr) {
		t.Errorf("got %v without an email, want a JournalRequestError", err)
	}
	if env.proxy.creates != 0 {
		t.Errorf("got %d creates, want none", env.proxy.creates)
	}

	// Unposted snapshots can still be deleted, and only once.
	if err := env.service.DeleteSnapshot(ctx, snapshot.ID); err != nil {
		t.Errorf("DeleteSnapshot: %v", err)
	}
	if err := env.service.DeleteSnapshot(ctx, snapshot.ID); err == nil {
		t.Error("got no error deleting a deleted snapshot")
	}
}

func TestGetSnapshotJournal_NonPositiveSalaryIsBlocked(t *testing.T) {
	env := newJournalTestEnv(t)

	snapshot, err := env.service.db.CreateSnapshot(context.Background(), env.employeeID, timex.NewMonth(2026, 9), Salary{Components: []Component{
		{Description: "Utang Kasbon", Amount: decimal.NewFromInt(-150_000), Multiplier: decimal.NewFromInt(1)},
	}})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	journal := env.journal(t, snapshot.ID)
	if journal.CanPost || journal.BlockedReason == nil {
		t.Errorf("got %+v, want posting blocked", journal)
	}
}

func TestJournalPosterFrom(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Email", " admin@example.com ")
	r.Header.Set("X-Name", "Admin%20%C3%9Cnal")

	got := journalPosterFrom(r)
	if got.Email != "admin@example.com" || got.Name != "Admin Ünal" {
		t.Errorf("got %+v", got)
	}
}
