package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

// RegResponse is a struct for registration response JSON
type RegResponse struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Fulldomain string   `json:"fulldomain"`
	Subdomain  string   `json:"subdomain"`
	Allowfrom  []string `json:"allowfrom"`
}

// UnregRequest is a struct providing the data to unregister
type UnregRequest struct {
	Username  uuid.UUID `json:"username"`
	Subdomain string    `json:"subdomain"`
}

func webRegisterPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var regStatus int
	var reg []byte
	var err error
	aTXT := ACMETxt{}
	bdata, _ := io.ReadAll(r.Body)
	if len(bdata) > 0 {
		err = json.Unmarshal(bdata, &aTXT)
		if err != nil {
			regStatus = http.StatusBadRequest
			reg = jsonError("malformed_json_payload")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(regStatus)
			_, _ = w.Write(reg)
			return
		}
	}

	// Fail with malformed CIDR mask in allowfrom
	err = aTXT.AllowFrom.isValid()
	if err != nil {
		regStatus = http.StatusBadRequest
		reg = jsonError("invalid_allowfrom_cidr")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(regStatus)
		_, _ = w.Write(reg)
		return
	}

	// Create new user
	nu, err := DB.Register(aTXT.AllowFrom)
	if err != nil {
		errstr := fmt.Sprintf("%v", err)
		reg = jsonError(errstr)
		regStatus = http.StatusInternalServerError
		log.WithFields(log.Fields{"error": err.Error()}).Debug("Error in registration")
	} else {
		log.WithFields(log.Fields{"user": nu.Username.String()}).Debug("Created new user")
		regStruct := RegResponse{nu.Username.String(), nu.Password, nu.Subdomain + "." + Config.General.Domain, nu.Subdomain, nu.AllowFrom.ValidEntries()}
		regStatus = http.StatusCreated
		reg, err = json.Marshal(regStruct)
		if err != nil {
			regStatus = http.StatusInternalServerError
			reg = jsonError("json_error")
			log.WithFields(log.Fields{"error": "json"}).Debug("Could not marshal JSON")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(regStatus)
	_, _ = w.Write(reg)
}

func webUnregisterPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var unregStatus int
	var err error
	var upd []byte

	// Get user data
	unregData, ok := r.Context().Value(ACMETxtKey).(UnregRequest)
	if !ok {
		log.WithFields(log.Fields{"error": "context"}).Error("Context error")
	}

	// Delete user
	err = DB.Unregister(unregData.Username)
	if err != nil {
		unregStatus = http.StatusInternalServerError
		upd = jsonError(fmt.Sprintf("%s (%v)", "delete_error", err))
	} else {
		log.WithFields(log.Fields{"user": unregData.Username.String()}).Debug("Deleted user")
		upd = []byte("{\"unregister\": \"" + unregData.Username.String() + "\"}")
		unregStatus = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(unregStatus)
	w.Write(upd)
}

func webUpdatePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var updStatus int
	var upd []byte
	// Get user
	a, ok := r.Context().Value(ACMETxtKey).(ACMETxt)
	if !ok {
		log.WithFields(log.Fields{"error": "context"}).Error("Context error")
	}
	// NOTE: An invalid subdomain should not happen - the auth handler should
	// reject POSTs with an invalid subdomain before this handler. Reject any
	// invalid subdomains anyway as a matter of caution.
	if !validSubdomain(a.Subdomain) {
		log.WithFields(log.Fields{"error": "subdomain", "subdomain": a.Subdomain, "txt": a.Value}).Debug("Bad update data")
		updStatus = http.StatusBadRequest
		upd = jsonError("bad_subdomain")
	} else if !validTXT(a.Value) {
		log.WithFields(log.Fields{"error": "txt", "subdomain": a.Subdomain, "txt": a.Value}).Debug("Bad update data")
		updStatus = http.StatusBadRequest
		upd = jsonError("bad_txt")
	} else if validSubdomain(a.Subdomain) && validTXT(a.Value) {
		err := DB.Update(a.ACMETxtPost)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to update record")
			updStatus = http.StatusInternalServerError
			upd = jsonError("db_error")
		} else {
			log.WithFields(log.Fields{"subdomain": a.Subdomain, "txt": a.Value}).Debug("TXT updated")
			updStatus = http.StatusOK
			upd = []byte("{\"txt\": \"" + a.Value + "\"}")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(updStatus)
	_, _ = w.Write(upd)
}

// Endpoint used to check the readiness and/or liveness (health) of the server.
func healthCheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.WriteHeader(http.StatusOK)
}
