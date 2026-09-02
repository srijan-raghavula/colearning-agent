package controller

import "net/http"

type HealthController struct{}

func NewHealthController() HealthController {
	return HealthController{}
}

func (HealthController) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
