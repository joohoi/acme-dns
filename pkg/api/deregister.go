package api

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/joohoi/acme-dns/pkg/acmedns"
)

// webDeregisterPost removes the account identified by the X-Api-User / X-Api-Key headers.
// Unlike /update the allowfrom list is not checked: the caller is the provisioning backend,
// not the host the account was registered for. The subdomain in the body must match the account.
func (a *AcmednsAPI) webDeregisterPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var status int
	var resp []byte
	postData := acmedns.ACMETxtPost{}
	user, err := a.getUserFromRequest(r)
	if err != nil {
		a.Logger.Errorw("Error while trying to get user",
			"error", err.Error())
		status = http.StatusUnauthorized
		resp = jsonError("forbidden")
	} else if err = json.NewDecoder(r.Body).Decode(&postData); err != nil {
		a.Logger.Errorw("Decoding error",
			"error", "json_error")
		status = http.StatusBadRequest
		resp = jsonError("malformed_json_payload")
	} else if postData.Subdomain != user.Subdomain {
		a.Logger.Errorw("Subdomain mismatch",
			"error", "subdomain_mismatch",
			"name", postData.Subdomain,
			"expected", user.Subdomain)
		status = http.StatusUnauthorized
		resp = jsonError("forbidden")
	} else if err = a.DB.Unregister(user.Username, user.Subdomain); err != nil {
		a.Logger.Errorw("Error while trying to deregister user",
			"error", err.Error())
		status = http.StatusInternalServerError
		resp = jsonError("db_error")
	} else {
		a.Logger.Infow("Deregistered user",
			"user", user.Username.String(),
			"subdomain", user.Subdomain)
		status = http.StatusOK
		resp = []byte("{\"deregistered\": \"" + user.Subdomain + "\"}")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(resp)
}
