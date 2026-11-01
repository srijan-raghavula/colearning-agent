package view

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
	"unicode"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(pattern string) (*Renderer, error) {
	templates, err := template.New("").Funcs(template.FuncMap{
		"initials": initials,
	}).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: templates}, nil
}

func initials(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "?"
	}
	first := []rune(words[0])
	result := string(unicode.ToUpper(first[0]))
	if len(words) > 1 {
		last := []rune(words[len(words)-1])
		result += string(unicode.ToUpper(last[0]))
	}
	return result
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
