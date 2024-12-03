package hris

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/employees", h.registerEmployeeRoutes)
	r.Route("/work-types", h.registerWorkTypeRoutes)
	r.Route("/work-logs", h.registerWorkLogRoutes)
}

func (h *Handler) registerEmployeeRoutes(r chi.Router) {
	r.Get("/", h.GetEmployees)
	r.Post("/", h.CreateEmployee)
}

func (h *Handler) registerWorkTypeRoutes(r chi.Router) {
	r.Get("/", h.GetWorkTypes)
	r.Post("/", h.CreateWorkType)
}

func (h *Handler) registerWorkLogRoutes(r chi.Router) {
	r.Get("/", h.GetWorkLogs)
	r.Post("/", h.CreateWorkLog)
	r.Get("/{workLogID}/for-patient", h.PrintWorkLogForPatient)
}
