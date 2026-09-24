package view

import (
	"bytes"
	"html/template"
	"net/http"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(pattern string) (*Renderer, error) {
	templates, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, name string, data any) {
	var body bytes.Buffer
	if err := r.templates.ExecuteTemplate(&body, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body.Bytes())
}
