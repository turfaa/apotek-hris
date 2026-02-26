package attendance

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/attendances", h.registerAttendanceRoutes)
}

func (h *Handler) registerAttendanceRoutes(r chi.Router) {
	r.Get("/", h.GetAttendancesBetweenDates)
	r.Get("/types", h.GetAttendanceTypes)
	r.Post("/types", h.CreateAttendanceType)
	r.Post("/types/{typeID}/enable-quota", h.EnableAttendanceTypeQuota)
	r.Get("/quotas", h.GetAllQuotas)
	r.Get("/quotas/{employeeID}", h.GetEmployeeQuotas)
	r.Put("/quotas/{employeeID}/{typeID}", h.SetEmployeeQuota)
	r.Put("/{employeeID}/{date}", h.UpsertAttendance)
}
