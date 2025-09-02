package salary

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/turfaa/apotek-hris/pkg/httpx"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"github.com/turfaa/apotek-hris/pkg/validatorx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetSalary(w http.ResponseWriter, r *http.Request) {
	monthStr := chi.URLParam(r, "month")
	if monthStr == "" {
		httpx.Error(w, errors.New("month is required"), http.StatusBadRequest)
		return
	}

	month, err := timex.NewMonthFromString(monthStr)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeID"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	salary, err := h.service.GetSalary(r.Context(), employeeID, month)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, salary)
}

func httpServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.Error(w, err, http.StatusNotFound)
	case errors.As(err, &validatorx.ValidationErrors{}):
		httpx.Error(w, err, http.StatusBadRequest)
	default:
		httpx.Error(w, err, http.StatusInternalServerError)
	}
}
