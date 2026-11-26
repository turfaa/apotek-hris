package vmedisproxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateJournalEntry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/vmedis/journal-entries" ||
			r.Header.Get("Idempotency-Key") != "hris-snapshot-1-salary-0" || r.Header.Get("X-Email") != "admin@example.com" ||
			!strings.Contains(string(body), `"amount":"3200000"`) {
			t.Errorf("got %s %s %v %s", r.Method, r.URL, r.Header, body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"voucherNo":"JRLM2610310001","vmedisId":1000001,"replayed":false}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL + "/api/"})
	got, err := c.CreateJournalEntry(context.Background(), "hris-snapshot-1-salary-0", "admin@example.com", CreateJournalEntryRequest{Date: "2026-10-31", Amount: "3200000"})
	if err != nil {
		t.Fatalf("CreateJournalEntry: %v", err)
	}
	if got != (CreatedJournalEntry{VoucherNo: "JRLM2610310001", VmedisID: 1000001}) {
		t.Errorf("got %+v", got)
	}
}

func TestCreateJournalEntry_Errors(t *testing.T) {
	for name, tc := range map[string]struct {
		status       int
		body         string
		outcomeKnown bool
	}{
		"bad request":      {400, `{"error":"nominal harus lebih dari 0"}`, true},
		"forbidden":        {403, `{"error":"unauthorized"}`, true},
		"in progress":      {409, `{"error":"sedang diproses"}`, true},
		"vmedis rejected":  {502, `{"error":"vmedis rejected","outcomeUnknown":false}`, true},
		"vmedis timed out": {502, `{"error":"timeout","outcomeUnknown":true}`, false},
		"proxy broke":      {500, `{"error":"db down"}`, false},
		"gateway not json": {504, `upstream timed out`, false},
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			_, err := New(Config{BaseURL: srv.URL}).CreateJournalEntry(context.Background(), "k", "e", CreateJournalEntryRequest{})
			var proxyErr *Error
			if !errors.As(err, &proxyErr) || proxyErr.Status != tc.status {
				t.Fatalf("got %v", err)
			}
			if OutcomeKnown(err) != tc.outcomeKnown {
				t.Errorf("OutcomeKnown = %v, want %v", OutcomeKnown(err), tc.outcomeKnown)
			}
		})
	}

	// A connection failure has an unknown outcome.
	_, err := New(Config{BaseURL: "http://127.0.0.1:1"}).CreateJournalEntry(context.Background(), "k", "e", CreateJournalEntryRequest{})
	if err == nil || OutcomeKnown(err) {
		t.Errorf("got %v, want an error with an unknown outcome", err)
	}
}

func TestFindJournalEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("date") != "2026-10-31" || r.URL.Query().Get("q") != "(Snapshot #1," {
			t.Errorf("got query %v", r.URL.Query())
		}
		_, _ = w.Write([]byte(`{"entries":[{"vmedisId":1,"voucherNo":"A","time":"2026-10-31T10:00:00+07:00","description":"Gaji","amount":"10","username":"bot"}]}`))
	}))
	defer srv.Close()

	got, err := New(Config{BaseURL: srv.URL}).FindJournalEntries(context.Background(), "e", "2026-10-31", "(Snapshot #1,")
	if err != nil || len(got) != 1 || got[0].VoucherNo != "A" || got[0].Amount != "10" {
		t.Errorf("got %+v, %v", got, err)
	}
}
