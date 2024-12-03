package httpx

import (
	"html/template"
	"log"
	"net/http"
)

func Template(w http.ResponseWriter, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("error writing template:", err)
	}
}
