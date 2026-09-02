package controller

import (
	"net/http"

	"github.com/srijan-raghavula/colearning-agent/src/usecase"
	"github.com/srijan-raghavula/colearning-agent/src/view"
)

type FacultyController struct {
	portal   usecase.FacultyPortal
	renderer *view.Renderer
}

func NewFacultyController(portal usecase.FacultyPortal, renderer *view.Renderer) FacultyController {
	return FacultyController{portal: portal, renderer: renderer}
}

func (c FacultyController) Page(w http.ResponseWriter, r *http.Request) {
	c.render(w, r, "faculty/page.html")
}

func (c FacultyController) SummaryPartial(w http.ResponseWriter, r *http.Request) {
	c.render(w, r, "faculty/summary.html")
}

func (c FacultyController) MissingPartial(w http.ResponseWriter, r *http.Request) {
	c.render(w, r, "faculty/missing.html")
}

func (c FacultyController) render(w http.ResponseWriter, r *http.Request, templateName string) {
	dashboard, err := c.portal.Dashboard(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.renderer.Render(w, templateName, map[string]any{
		"Role":      "faculty",
		"Dashboard": dashboard,
	})
}
