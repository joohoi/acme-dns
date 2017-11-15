package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

func webRegisterPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var regStatus int
	var reg []byte
	aTXT := ACMETxt{}
	bdata, _ := ioutil.ReadAll(r.Body)
	if bdata != nil && len(bdata) > 0 {
		err := json.Unmarshal(bdata, &aTXT)
		if err != nil {
			regStatus = http.StatusBadRequest
			reg = jsonError("malformed_json_payload")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(regStatus)
			w.Write(reg)
			return
		}
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
	w.Write(reg)
}

func webUpdatePost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var updStatus int
	var upd []byte
	// Get user
	a, ok := r.Context().Value(ACMETxtKey).(ACMETxt)
	if !ok {
		log.WithFields(log.Fields{"error": "context"}).Error("Context error")
	}
	if validSubdomain(a.Subdomain) && validTXT(a.Value) {
		err := DB.Update(a)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to update record")
			updStatus = http.StatusInternalServerError
			upd = jsonError("db_error")
		} else {
			log.WithFields(log.Fields{"subdomain": a.Subdomain, "txt": a.Value}).Debug("TXT updated")
			updStatus = http.StatusOK
			upd = []byte("{\"txt\": \"" + a.Value + "\"}")
		}
	} else {
		log.WithFields(log.Fields{"error": "subdomain", "subdomain": a.Subdomain, "txt": a.Value}).Debug("Bad update data")
		updStatus = http.StatusBadRequest
		upd = jsonError("bad_subdomain")
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(updStatus)
	w.Write(upd)
}
