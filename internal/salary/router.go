package salary

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/salary", h.registerSalaryRoutes)
}

func (h *Handler) registerSalaryRoutes(r chi.Router) {
	r.Delete(`/{employeeID:^\d+}/static-components/{id:^\d+}`, h.DeleteStaticComponent)
	r.Get(`/{employeeID:^\d+}/static-components`, h.GetEmployeeStaticComponents)
	r.Post(`/{employeeID:^\d+}/static-components`, h.CreateStaticComponent)

	r.Delete(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components/{id:^\d+}`, h.DeleteAdditionalComponent)
	r.Get(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components`, h.GetEmployeeAdditionalComponents)
	r.Post(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components`, h.CreateAdditionalComponent)

	r.Get(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}`, h.GetSalary)
}
