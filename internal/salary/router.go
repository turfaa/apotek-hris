package salary

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/salary", h.registerSalaryRoutes)
}

func (h *Handler) registerSalaryRoutes(r chi.Router) {
	r.Delete("/{month}/{employeeID}/additional-components/{id}", h.DeleteAdditionalComponent)
	r.Get("/{month}/{employeeID}/additional-components", h.GetEmployeeAdditionalComponents)
	r.Post("/{month}/{employeeID}/additional-components", h.CreateAdditionalComponent)

	r.Get("/{month}/{employeeID}", h.GetSalary)
}
