package attendance

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"github.com/turfaa/apotek-hris/internal/hris"
	"github.com/turfaa/apotek-hris/pkg/httpx"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"github.com/turfaa/apotek-hris/pkg/validatorx"
	"github.com/turfaa/go-date"
)

type Handler struct {
	service     *Service
	hrisService *hris.Service
}

func NewHandler(service *Service, hrisService *hris.Service) *Handler {
	return &Handler{service: service, hrisService: hrisService}
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

func (h *Handler) EnableAttendanceTypeQuota(w http.ResponseWriter, r *http.Request) {
	typeIDStr := chi.URLParam(r, "typeID")
	if typeIDStr == "" {
		httpx.Error(w, errors.New("typeID is required"), http.StatusBadRequest)
		return
	}

	typeID, err := strconv.ParseInt(typeIDStr, 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	attendanceType, err := h.service.EnableAttendanceTypeQuota(r.Context(), typeID)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, attendanceType)
}

func (h *Handler) GetAllQuotas(w http.ResponseWriter, r *http.Request) {
	quotas, err := h.service.GetAllQuotas(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	quotaEnabledTypes, err := h.service.GetQuotaEnabledAttendanceTypes(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	employees, err := h.hrisService.GetEmployees(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	employeeIDs := make([]int64, len(employees))
	for i, e := range employees {
		employeeIDs[i] = e.ID
	}

	pages := GroupQuotasByAttendanceType(quotas, quotaEnabledTypes, employeeIDs)
	httpx.Ok(w, pages)
}

func (h *Handler) GetEmployeeQuotas(w http.ResponseWriter, r *http.Request) {
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

	quotas, err := h.service.GetEmployeeQuotas(r.Context(), employeeID)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, quotas)
}

func (h *Handler) SetEmployeeQuota(w http.ResponseWriter, r *http.Request) {
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

	typeIDStr := chi.URLParam(r, "typeID")
	if typeIDStr == "" {
		httpx.Error(w, errors.New("typeID is required"), http.StatusBadRequest)
		return
	}

	typeID, err := strconv.ParseInt(typeIDStr, 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var req SetEmployeeAttendanceQuotaRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	req.EmployeeID = employeeID
	req.AttendanceTypeID = typeID

	quota, err := h.service.SetEmployeeQuota(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, quota)
}

func (h *Handler) GetQuotaAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.service.GetQuotaAuditLogs(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, logs)
}

func (h *Handler) GetEmployeeQuotaAuditLogs(w http.ResponseWriter, r *http.Request) {
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

	logs, err := h.service.GetEmployeeQuotaAuditLogs(r.Context(), employeeID)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, logs)
}

func httpServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		httpx.Error(w, err, http.StatusNotFound)
	case errors.Is(err, ErrQuotaExhausted):
		httpx.Error(w, err, http.StatusBadRequest)
	case errors.Is(err, ErrAlreadyHasQuota):
		httpx.Error(w, err, http.StatusBadRequest)
	case errors.As(err, &validatorx.ValidationErrors{}):
		httpx.Error(w, err, http.StatusBadRequest)
	default:
		httpx.Error(w, err, http.StatusInternalServerError)
	}
}
