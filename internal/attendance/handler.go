package attendance

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"github.com/turfaa/apotek-hris/pkg/httpx"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"github.com/turfaa/apotek-hris/pkg/validatorx"
	"github.com/turfaa/go-date"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetAttendancesBetweenDates(w http.ResponseWriter, r *http.Request) {
	from, to, err := timex.GetMonthDateRangeFromQuery(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	attendances, err := h.service.GetAttendancesBetweenDates(r.Context(), from, to)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	dailyAttendances := CreateListAtDate(from, to, attendances)
	employeeSummaries := CreateEmployeeSummaries(attendances)

	httpx.Ok(w, map[string]any{
		"dailyAttendances":  dailyAttendances,
		"employeeSummaries": employeeSummaries,
	})
}

func (h *Handler) UpsertAttendance(w http.ResponseWriter, r *http.Request) {
	employeeIDStr := chi.URLParam(r, "employeeID")
	if employeeIDStr == "" {
		httpx.Error(w, errors.New("employeeID is required"), http.StatusBadRequest)
		return
	}

	employeeID, err := strconv.ParseInt(employeeIDStr, 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	dateStr := chi.URLParam(r, "date")
	if dateStr == "" {
		httpx.Error(w, errors.New("date is required"), http.StatusBadRequest)
		return
	}

	dt, err := date.NewFromString(dateStr)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var req UpsertAttendanceRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	req.EmployeeID = employeeID
	req.Date = dt

	attendance, err := h.service.UpsertAttendance(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, attendance)
}

func (h *Handler) GetAttendanceTypes(w http.ResponseWriter, r *http.Request) {
	attendanceTypes, err := h.service.GetAttendanceTypes(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, attendanceTypes)
}

func (h *Handler) CreateAttendanceType(w http.ResponseWriter, r *http.Request) {
	var req CreateAttendanceTypeRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	attendanceType, err := h.service.CreateAttendanceType(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, attendanceType)
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
