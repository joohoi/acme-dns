package api

import (
	"net/http"

	"github.com/joohoi/acme-dns/pkg/acmedns"

	"github.com/julienschmidt/httprouter"
)

func (a *AcmednsAPI) webUpdatePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var updStatus int
	var upd []byte
	// Get user
	atxt, ok := r.Context().Value(ACMETxtKey).(acmedns.ACMETxt)
	if !ok {
		a.Logger.Errorw("Context error",
			"error", "context")
	}
	// NOTE: An invalid subdomain should not happen - the auth handler should
	// reject POSTs with an invalid subdomain before this handler. Reject any
	// invalid subdomains anyway as a matter of caution.
	if !validSubdomain(atxt.Subdomain) {
		a.Logger.Errorw("Bad update data",
			"error", "subdomain",
			"subdomain", atxt.Subdomain,
			"txt", atxt.Value)
		updStatus = http.StatusBadRequest
		upd = jsonError("bad_subdomain")
	} else if !validTXT(atxt.Value) {
		a.Logger.Errorw("Bad update data",
			"error", "txt",
			"subdomain", atxt.Subdomain,
			"txt", atxt.Value)
		updStatus = http.StatusBadRequest
		upd = jsonError("bad_txt")
	} else if validSubdomain(atxt.Subdomain) && validTXT(atxt.Value) {
		err := a.DB.Update(atxt.ACMETxtPost)
		if err != nil {
			a.Logger.Errorw("Error while trying to update record",
				"error", err.Error())
			updStatus = http.StatusInternalServerError
			upd = jsonError("db_error")
		} else {
			a.Logger.Debugw("TXT record updated",
				"subdomain", atxt.Subdomain,
				"txt", atxt.Value)
			updStatus = http.StatusOK
			upd = []byte("{\"txt\": \"" + atxt.Value + "\"}")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(updStatus)
	_, _ = w.Write(upd)
}
