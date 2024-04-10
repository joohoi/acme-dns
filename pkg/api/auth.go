package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/joohoi/acme-dns/pkg/acmedns"
)

type key int

// ACMETxtKey is a context key for ACMETxt struct
const ACMETxtKey key = 0

// Auth middleware for update request
func (a *AcmednsAPI) Auth(update httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		postData := acmedns.ACMETxt{}
		userOK := false
		user, err := a.getUserFromRequest(r)
		if err == nil {
			if a.updateAllowedFromIP(r, user) {
				dec := json.NewDecoder(r.Body)
				err = dec.Decode(&postData)
				if err != nil {
					a.Logger.Errorw("Decoding error",
						"error", "json_error")
				}
				if user.Subdomain == postData.Subdomain {
					userOK = true
				} else {
					a.Logger.Errorw("Subdomain mismatch",
						"error", "subdomain_mismatch",
						"name", postData.Subdomain,
						"expected", user.Subdomain)
				}
			} else {
				a.Logger.Errorw("Update not allowed from IP",
					"error", "ip_unauthorized")
			}
		} else {
			a.Logger.Errorw("Error while trying to get user",
				"error", err.Error())
		}
		if userOK {
			// Set user info to the decoded ACMETxt object
			postData.Username = user.Username
			postData.Password = user.Password
			// Set the ACMETxt struct to context to pull in from update function
			ctx := context.WithValue(r.Context(), ACMETxtKey, postData)
			update(w, r.WithContext(ctx), p)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write(jsonError("forbidden"))
		}
	}
}

func (a *AcmednsAPI) getUserFromRequest(r *http.Request) (acmedns.ACMETxt, error) {
	uname := r.Header.Get("X-Api-User")
	passwd := r.Header.Get("X-Api-Key")
	username, err := getValidUsername(uname)
	if err != nil {
		return acmedns.ACMETxt{}, fmt.Errorf("invalid username: %s: %s", uname, err.Error())
	}
	if validKey(passwd) {
		dbuser, err := a.DB.GetByUsername(username)
		if err != nil {
			a.Logger.Errorw("Error while trying to get user",
				"error", err.Error())
			// To protect against timed side channel (never gonna give you up)
			acmedns.CorrectPassword(passwd, "$2a$10$8JEFVNYYhLoBysjAxe2yBuXrkDojBQBkVpXEQgyQyjn43SvJ4vL36")

			return acmedns.ACMETxt{}, fmt.Errorf("invalid username: %s", uname)
		}
		if acmedns.CorrectPassword(passwd, dbuser.Password) {
			return dbuser, nil
		}
		return acmedns.ACMETxt{}, fmt.Errorf("invalid password for user %s", uname)
	}
	return acmedns.ACMETxt{}, fmt.Errorf("invalid key for user %s", uname)
}

func (a *AcmednsAPI) updateAllowedFromIP(r *http.Request, user acmedns.ACMETxt) bool {
	if a.Config.API.UseHeader {
		ips := getIPListFromHeader(r.Header.Get(a.Config.API.HeaderName))
		return user.AllowedFromList(ips)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		a.Logger.Errorw("Error while parsing remote address",
			"error", err.Error(),
			"remoteaddr", r.RemoteAddr)
		host = ""
	}
	return user.AllowedFrom(host)
}
