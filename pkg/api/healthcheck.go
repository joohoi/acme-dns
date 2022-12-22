package api

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// Endpoint used to check the readiness and/or liveness (health) of the server.
func (a *AcmednsAPI) healthCheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}
