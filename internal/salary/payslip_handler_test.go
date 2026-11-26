package salary

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Without the pharmacy letterhead, both slip routes answer 503 before
// touching the database, and bad requests are refused first.
func TestSlipRoutesWithoutLetterhead(t *testing.T) {
	r := chi.NewRouter()
	NewHandler(&Service{}).RegisterRoutes(r)

	cases := []struct {
		path string
		want int
	}{
		{"/salary/snapshots/12/pdf", http.StatusServiceUnavailable},
		{"/salary/snapshots/slips.zip?month=2026-10", http.StatusServiceUnavailable},
		{"/salary/snapshots/slips.zip?month=2026-10&employeeID=3", http.StatusServiceUnavailable},
		{"/salary/snapshots/slips.zip", http.StatusBadRequest},
		{"/salary/snapshots/slips.zip?month=oktober", http.StatusBadRequest},
		{"/salary/snapshots/slips.zip?month=2026-10&employeeID=x", http.StatusBadRequest},
		{"/salary/snapshots/abc/pdf", http.StatusNotFound},
	}

	for _, c := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("GET %s = %d, want %d (%s)", c.path, rec.Code, c.want, rec.Body.String())
		}
	}
}
