package main

import (
	"context"
	"encoding/json"
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
		userOK := false
		userSupplied := false
		user := ACMETxt{}
		postData, err := decodePostData(r)
		if err != nil {
			log.WithFields(log.Fields{"error": "json_error", "string": err.Error()}).Error("Decode error")
		} else {
			userSupplied = getCredsFromRequest(r, &postData)
		}
		if !validSubdomain(postData.Subdomain) {
			log.WithFields(log.Fields{"error": "invalid_subdomain"}).Error("Invalid subdomain UUID")
		} else {
			user, err = DB.GetBySubdomain(postData.Subdomain)
			if err != nil {
				log.WithFields(log.Fields{"error": "subdomain_not_found", "string": err.Error()}).Error("Subdomain is not registered")
			} else {
				if updateAllowedFromIP(r, user) {
					if correctPassword(postData.Password, user.Password) {
						if userSupplied {
							userOK = postData.Username == user.Username
						} else {
							userOK = true
						}
					} else {
						log.WithFields(log.Fields{"error": "invalid_password"}).Error("Password was not correct")
					}
				} else {
					log.WithFields(log.Fields{"error": "ip_unauthorized"}).Error("Update not allowed from IP")
				}
			}
		}
		if userOK {
			// Set user info to the decoded ACMETxt object
			postData.Username = user.Username
			postData.Password = user.Password
			postData.Subdomain = user.Subdomain
			// Set the ACMETxt struct to context to pull in from update function
			ctx := context.WithValue(r.Context(), ACMETxtKey, postData)
			update(w, r.WithContext(ctx), p)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(jsonError("forbidden"))
		}
	}
}

func decodePostData(r *http.Request) (ACMETxt, error) {
	postData := ACMETxt{}
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&postData)
	if err != nil {
		return ACMETxt{}, err
	}
	return postData, nil
}

func getCredsFromRequest(r *http.Request, postData *ACMETxt) bool {
	if !validKey(postData.Password) {
		key := r.Header.Get("X-Api-Key")
		if validKey(key) {
			postData.Password = key
		}
	}

	user := r.Header.Get("X-Api-User")
	uname, err := getValidUsername(user)
	if err != nil {
		return false
	}
	postData.Username = uname
	return true
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
