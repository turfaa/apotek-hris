package hris

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/turfaa/apotek-hris/internal/hris/templates"
	"github.com/turfaa/apotek-hris/pkg/httpx"
	"github.com/turfaa/apotek-hris/pkg/timex"
	"github.com/turfaa/apotek-hris/pkg/validatorx"

	"github.com/go-chi/chi/v5"
	"github.com/go-json-experiment/json"
	"github.com/jmoiron/sqlx"
)

type Handler struct {
	service *Service
}

func NewHandler(db *sqlx.DB) *Handler {
	return &Handler{service: NewService(db)}
}

func (h *Handler) GetEmployees(w http.ResponseWriter, r *http.Request) {
	employees, err := h.service.GetEmployees(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, employees)
}

func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var req CreateEmployeeRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	employee, err := h.service.CreateEmployee(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, employee)
}

func (h *Handler) GetWorkTypes(w http.ResponseWriter, r *http.Request) {
	workTypes, err := h.service.GetWorkTypes(r.Context())
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, workTypes)
}

func (h *Handler) CreateWorkType(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkTypeRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	workType, err := h.service.CreateWorkType(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, workType)
}

func (h *Handler) GetWorkLogs(w http.ResponseWriter, r *http.Request) {
	from, to, err := timex.GetTimeRangeFromQuery(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	workLogs, err := h.service.GetWorkLogsBetween(r.Context(), from, to)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, workLogs)
}

func (h *Handler) CreateWorkLog(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkLogRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	workLog, err := h.service.CreateWorkLog(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, workLog)
}

func (h *Handler) PrintWorkLogForPatient(w http.ResponseWriter, r *http.Request) {
	workLogIDStr := chi.URLParam(r, "workLogID")
	if workLogIDStr == "" {
		httpx.Error(w, errors.New("workLogID is required"), http.StatusBadRequest)
		return
	}

	workLogID, err := strconv.ParseInt(workLogIDStr, 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	workLog, err := h.service.GetWorkLog(r.Context(), workLogID)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	units := make([]templates.WorkLogUnitForPatient, len(workLog.Units))
	for i, unit := range workLog.Units {
		units[i] = templates.WorkLogUnitForPatient{
			WorkType:    unit.WorkType.Name,
			WorkOutcome: unit.WorkOutcome,
			OutcomeUnit: unit.WorkType.OutcomeUnit,
		}
	}

	httpx.Template(w, templates.WorkLogForPatient, templates.WorkLogForPatientData{
		PatientName:  workLog.PatientName,
		Place:        "Apotek Aulia Farma",
		EmployeeName: workLog.Employee.Name,
		Units:        units,
		Notes:        "Untuk hasil yang lebih akurat, silakan lakukan tes kembali di laboratorium terdekat.",
		CreatedAt:    timex.FormatDateTime(workLog.CreatedAt),
	})
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
