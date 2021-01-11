package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
)

type key int

// ACMETxtKey is a context key for ACMETxt struct
const ACMETxtKey key = 0

// Auth middleware for update request
func Auth(update httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		postData := ACMETxt{}
		userOK := false
		user, err := getUserFromRequest(r)
		if err == nil {
			if updateAllowedFromIP(r, user) {
				dec := json.NewDecoder(r.Body)
				err = dec.Decode(&postData)
				if err != nil {
					log.WithFields(log.Fields{"error": "json_error", "string": err.Error()}).Error("Decode error")
				}
				if user.Subdomain == postData.Subdomain {
					userOK = true
				} else {
					log.WithFields(log.Fields{"error": "subdomain_mismatch", "name": postData.Subdomain, "expected": user.Subdomain}).Error("Subdomain mismatch")
				}
			} else {
				log.WithFields(log.Fields{"error": "ip_unauthorized"}).Error("Update not allowed from IP")
			}
		} else {
			log.WithFields(log.Fields{"error": err.Error()}).Error("Error while trying to get user")
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

func getUserFromRequest(r *http.Request) (ACMETxt, error) {
	uname := r.Header.Get("X-Api-User")
	passwd := r.Header.Get("X-Api-Key")
	username, err := getValidUsername(uname)
	if err != nil {
		return ACMETxt{}, fmt.Errorf("Invalid username: %s: %s", uname, err.Error())
	}
	if validKey(passwd) {
		dbuser, err := DB.GetByUsername(username)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Error("Error while trying to get user")
			// To protect against timed side channel (never gonna give you up)
			correctPassword(passwd, "$2a$10$8JEFVNYYhLoBysjAxe2yBuXrkDojBQBkVpXEQgyQyjn43SvJ4vL36")

			return ACMETxt{}, fmt.Errorf("Invalid username: %s", uname)
		}
		if correctPassword(passwd, dbuser.Password) {
			return dbuser, nil
		}
		return ACMETxt{}, fmt.Errorf("Invalid password for user %s", uname)
	}
	return ACMETxt{}, fmt.Errorf("Invalid key for user %s", uname)
}

func updateAllowedFromIP(r *http.Request, user ACMETxt) bool {
	if Config.API.UseHeader {
		ips := getIPListFromHeader(r.Header.Get(Config.API.HeaderName))
		return user.allowedFromList(ips)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "remoteaddr": r.RemoteAddr}).Error("Error while parsing remote address")
		host = ""
	}
	return user.allowedFrom(host)
}
