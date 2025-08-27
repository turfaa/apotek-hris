package attendance

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/attendances", h.registerAttendanceRoutes)
}

func (h *Handler) registerAttendanceRoutes(r chi.Router) {
	r.Get("/", h.GetAttendancesBetweenDates)
	r.Get("/types", h.GetAttendanceTypes)
	r.Post("/types", h.CreateAttendanceType)
	r.Put("/{employeeID}/{date}", h.UpsertAttendance)
}
