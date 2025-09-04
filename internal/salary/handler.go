package salary

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
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
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
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

func (h *Handler) GetEmployeeAdditionalComponents(w http.ResponseWriter, r *http.Request) {
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	additionalComponents, err := h.service.GetEmployeeAdditionalComponents(r.Context(), employeeID, month)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, additionalComponents)

}

func (h *Handler) CreateAdditionalComponent(w http.ResponseWriter, r *http.Request) {
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var component Component
	if err := json.UnmarshalRead(r.Body, &component); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	createdComponent, err := h.service.CreateAdditionalComponent(r.Context(), employeeID, month, component)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, createdComponent)
}

func (h *Handler) DeleteAdditionalComponent(w http.ResponseWriter, r *http.Request) {
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	idToDelete, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteAdditionalComponent(r.Context(), employeeID, month, idToDelete); err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, map[string]string{"message": "successfully deleted the additional component"})
}

func (h *Handler) parseEmployeeIDAndMonth(r *http.Request) (int64, timex.Month, error) {
	monthStr := chi.URLParam(r, "month")
	if monthStr == "" {
		return 0, timex.Month{}, errors.New("month is required")
	}

	month, err := timex.NewMonthFromString(monthStr)
	if err != nil {
		return 0, timex.Month{}, fmt.Errorf("parse month: %w", err)
	}

	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeID"), 10, 64)
	if err != nil {
		return 0, timex.Month{}, fmt.Errorf("parse employee ID: %w", err)
	}

	return employeeID, month, nil
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
