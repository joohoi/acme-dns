package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/acme-dns/acme-dns/pkg/acmedns"
)

// RegResponse is a struct for registration response JSON
type RegResponse struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Fulldomain string   `json:"fulldomain"`
	Subdomain  string   `json:"subdomain"`
	Allowfrom  []string `json:"allowfrom"`
}

func (a *AcmednsAPI) webRegisterPost(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var regStatus int
	var reg []byte
	var err error
	aTXT := acmedns.ACMETxt{}
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
	err = aTXT.AllowFrom.IsValid()
	if err != nil {
		regStatus = http.StatusBadRequest
		reg = jsonError("invalid_allowfrom_cidr")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(regStatus)
		_, _ = w.Write(reg)
		return
	}

	// Create new user
	nu, err := a.DB.Register(aTXT.AllowFrom)
	if err != nil {
		errstr := fmt.Sprintf("%v", err)
		reg = jsonError(errstr)
		regStatus = http.StatusInternalServerError
		a.Logger.Errorw("Error in registration",
			"error", err.Error())
	} else {
		a.Logger.Debugw("Created new user",
			"user", nu.Username.String())
		regStruct := RegResponse{nu.Username.String(), nu.Password, nu.Subdomain + "." + a.Config.General.Domain, nu.Subdomain, nu.AllowFrom.ValidEntries()}
		regStatus = http.StatusCreated
		reg, err = json.Marshal(regStruct)
		if err != nil {
			regStatus = http.StatusInternalServerError
			reg = jsonError("json_error")
			a.Logger.Errorw("Could not marshal JSON",
				"error", "json")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(regStatus)
	_, _ = w.Write(reg)
}
