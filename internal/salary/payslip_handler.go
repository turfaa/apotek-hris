package salary

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/turfaa/apotek-hris/internal/salary/payslip"
	"github.com/turfaa/apotek-hris/pkg/httpx"
	"github.com/turfaa/apotek-hris/pkg/timex"
)

// GetSnapshotSlipPDF downloads the pay slip PDF of a snapshot.
func (h *Handler) GetSnapshotSlipPDF(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	file, err := h.service.RenderSlip(r.Context(), id)
	if err != nil {
		slipError(w, err)
		return
	}

	writeFile(w, "application/pdf", file)
}

// GetSnapshotSlipsZip downloads a ZIP of the pay slips of a month's
// snapshots, optionally of one employee.
func (h *Handler) GetSnapshotSlipsZip(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	monthStr := query.Get("month")
	if monthStr == "" {
		httpx.Error(w, errors.New("month is required"), http.StatusBadRequest)
		return
	}

	month, err := timex.NewMonthFromString(monthStr)
	if err != nil {
		httpx.Error(w, fmt.Errorf("parse month: %w", err), http.StatusBadRequest)
		return
	}

	var employeeID *int64
	if employeeIDStr := query.Get("employeeID"); employeeIDStr != "" {
		id, err := strconv.ParseInt(employeeIDStr, 10, 64)
		if err != nil {
			httpx.Error(w, fmt.Errorf("parse employee ID: %w", err), http.StatusBadRequest)
			return
		}
		employeeID = &id
	}

	file, err := h.service.RenderSlipsZip(r.Context(), month, employeeID)
	if err != nil {
		slipError(w, err)
		return
	}

	writeFile(w, "application/zip", file)
}

func writeFile(w http.ResponseWriter, contentType string, file payslip.File) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", payslip.ContentDisposition(file.Filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Content)))
	// Slips carry salaries: never keep them in a cache.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}

func slipError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payslip.ErrNotConfigured):
		httpx.Error(w, payslip.ErrNotConfigured, http.StatusServiceUnavailable)
	case errors.Is(err, sql.ErrNoRows):
		httpx.Error(w, errors.New("snapshot not found"), http.StatusNotFound)
	case errors.Is(err, ErrNoSnapshots):
		httpx.Error(w, ErrNoSnapshots, http.StatusNotFound)
	default:
		httpx.Error(w, err, http.StatusInternalServerError)
	}
}
