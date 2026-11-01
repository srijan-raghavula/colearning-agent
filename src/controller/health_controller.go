package controller

import "net/http"

type HealthController struct{}

func NewHealthController() HealthController { return HealthController{} }

func (c HealthController) Handle(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
