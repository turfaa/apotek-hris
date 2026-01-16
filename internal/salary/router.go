package salary

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/salary", h.registerSalaryRoutes)
}

func (h *Handler) registerSalaryRoutes(r chi.Router) {
	r.Delete(`/{employeeID:^\d+}/static-components/{id:^\d+}`, h.DeleteStaticComponent)
	r.Get(`/{employeeID:^\d+}/static-components`, h.GetEmployeeStaticComponents)
	r.Post(`/{employeeID:^\d+}/static-components`, h.CreateStaticComponent)

	r.Post(`/{month:20\d{2}-\d{2}}/additional-components/bulk`, h.BulkCreateAdditionalComponents)
	r.Delete(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components/{id:^\d+}`, h.DeleteAdditionalComponent)
	r.Get(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components`, h.GetEmployeeAdditionalComponents)
	r.Post(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/additional-components`, h.CreateAdditionalComponent)

	r.Delete(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/extra-infos/{id:^\d+}`, h.DeleteExtraInfo)
	r.Get(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/extra-infos`, h.GetEmployeeExtraInfos)
	r.Post(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}/extra-infos`, h.CreateExtraInfo)

	r.Get(`/{month:20\d{2}-\d{2}}/{employeeID:^\d+}`, h.GetSalary)

	r.Get(`/snapshots/{id:^\d+}`, h.GetSnapshot)
	r.Get(`/snapshots`, h.GetSnapshots)
	r.Post(`/snapshots`, h.CreateSnapshot)
	r.Delete(`/snapshots/{id:^\d+}`, h.DeleteSnapshot)
}
