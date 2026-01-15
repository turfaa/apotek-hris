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

func (h *Handler) GetEmployeeStaticComponents(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeID"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	staticComponents, err := h.service.GetEmployeeStaticComponents(r.Context(), employeeID)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, staticComponents)
}

func (h *Handler) CreateStaticComponent(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeID"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var req Component
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	createdComponent, err := h.service.CreateStaticComponent(r.Context(), employeeID, req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, createdComponent)
}

func (h *Handler) DeleteStaticComponent(w http.ResponseWriter, r *http.Request) {
	employeeID, err := strconv.ParseInt(chi.URLParam(r, "employeeID"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteStaticComponent(r.Context(), employeeID, id); err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, map[string]string{"message": "successfully deleted the static component"})
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

func (h *Handler) BulkCreateAdditionalComponents(w http.ResponseWriter, r *http.Request) {
	var req BulkCreateAdditionalComponentRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	created, err := h.service.BulkCreateAdditionalComponents(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, created)
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

func (h *Handler) GetEmployeeExtraInfos(w http.ResponseWriter, r *http.Request) {
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	extraInfos, err := h.service.GetEmployeeExtraInfos(r.Context(), employeeID, month)
	if err != nil {
		httpServiceError(w, err)
		return
	}
	httpx.Ok(w, extraInfos)

}

func (h *Handler) CreateExtraInfo(w http.ResponseWriter, r *http.Request) {
	employeeID, month, err := h.parseEmployeeIDAndMonth(r)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	var req CreateExtraInfoRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	extraInfo, err := h.service.CreateExtraInfo(r.Context(), employeeID, month, req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, extraInfo)
}

func (h *Handler) DeleteExtraInfo(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.DeleteExtraInfo(r.Context(), employeeID, month, idToDelete); err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, map[string]string{"message": "successfully deleted the extra info"})
}

func (h *Handler) GetSnapshots(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()

	var req GetSnapshotsRequest
	if employeeIDStr := queries.Get("employeeID"); employeeIDStr != "" {
		employeeID, err := strconv.ParseInt(employeeIDStr, 10, 64)
		if err != nil {
			httpx.Error(w, err, http.StatusBadRequest)
			return
		}

		req.EmployeeID = &employeeID
	}

	if monthStr := queries.Get("month"); monthStr != "" {
		month, err := timex.NewMonthFromString(monthStr)
		if err != nil {
			httpx.Error(w, err, http.StatusBadRequest)
			return
		}

		req.Month = &month
	}

	snapshots, err := h.service.GetSnapshots(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, snapshots)
}

func (h *Handler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	snapshot, err := h.service.GetSnapshot(r.Context(), id)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, snapshot)
}

func (h *Handler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req CreateSnapshotRequest
	if err := json.UnmarshalRead(r.Body, &req); err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	snapshot, err := h.service.CreateSnapshot(r.Context(), req)
	if err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, snapshot)
}

func (h *Handler) DeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	idToDelete, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpx.Error(w, err, http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteSnapshot(r.Context(), idToDelete); err != nil {
		httpServiceError(w, err)
		return
	}

	httpx.Ok(w, map[string]string{"message": "successfully deleted the snapshot"})
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
