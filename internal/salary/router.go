package salary

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/salary", h.registerSalaryRoutes)
}

func (h *Handler) registerSalaryRoutes(r chi.Router) {
	r.Get("/{month}/{employeeID}", h.GetSalary)
}
