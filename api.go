package main

import (
	"errors"
	"fmt"

	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
)

// Serve is an authentication middlware function used to authenticate update requests
func (a authMiddleware) Serve(ctx iris.Context) {
	allowUpdate := false
	usernameStr := ctx.GetHeader("X-Api-User")
	password := ctx.GetHeader("X-Api-Key")
	postData := ACMETxt{}

	username, err := getValidUsername(usernameStr)
	if err == nil && validKey(password) {
		au, err := DB.GetByUsername(username)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Error("Error while trying to get user")
			// To protect against timed side channel (never gonna give you up)
			correctPassword(password, "$2a$10$8JEFVNYYhLoBysjAxe2yBuXrkDojBQBkVpXEQgyQyjn43SvJ4vL36")
		} else {
			if correctPassword(password, au.Password) {
				// Password ok

				// Now test for the possibly limited ranges
				if Config.API.UseHeader {
					ips := getIPListFromHeader(ctx.GetHeader(Config.API.HeaderName))
					allowUpdate = au.allowedFromList(ips)
				} else {
					allowUpdate = au.allowedFrom(ctx.RemoteAddr())
				}

				if allowUpdate {
					// Update is allowed from remote addr
					if err := ctx.ReadJSON(&postData); err == nil {
						if au.Subdomain == postData.Subdomain {
							ctx.Next()
							return
						}
					} else {
						// JSON error
						log.WithFields(log.Fields{"error": err.Error()}).Warning("Failed reading POST data")
						ctx.JSON(iris.Map{"error": "bad data"})
						ctx.StatusCode(iris.StatusBadRequest)
						return
					}
				}
			} else {
				// Wrong password
				log.WithFields(log.Fields{"username": username}).Warning("Failed password check")
			}
		}
	}
	ctx.JSON(iris.Map{"error": "unauthorized"})
	ctx.StatusCode(iris.StatusUnauthorized)
}

func webRegisterPost(ctx iris.Context) {
	var regJSON iris.Map
	var regStatus int
	aTXT := ACMETxt{}
	_ = ctx.ReadJSON(&aTXT)
	// Create new user
	nu, err := DB.Register(aTXT.AllowFrom)
	if err != nil {
		errstr := fmt.Sprintf("%v", err)
		regJSON = iris.Map{"error": errstr}
		regStatus = iris.StatusInternalServerError
		log.WithFields(log.Fields{"error": err.Error()}).Debug("Error in registration")
	} else {
		regJSON = iris.Map{"username": nu.Username, "password": nu.Password, "fulldomain": nu.Subdomain + "." + Config.General.Domain, "subdomain": nu.Subdomain, "allowfrom": nu.AllowFrom.ValidEntries()}
		regStatus = iris.StatusCreated

		log.WithFields(log.Fields{"user": nu.Username.String()}).Debug("Created new user")
	}
	ctx.JSON(regJSON)
	ctx.StatusCode(regStatus)
}

func webUpdatePost(ctx iris.Context) {
	// User auth done in middleware
	a := ACMETxt{}
	userStr := ctx.GetHeader("X-API-User")
	// Already checked in auth middlware
	username, _ := getValidUsername(userStr)
	// Already checked in auth middleware
	_ = ctx.ReadJSON(&a)
	a.Username = username
	// Do update
	if validSubdomain(a.Subdomain) && validTXT(a.Value) {
		err := DB.Update(a)
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to update record")
			webUpdatePostError(ctx, errors.New("internal error"), iris.StatusInternalServerError)
			return
		}
		ctx.JSON(iris.Map{"txt": a.Value})
		ctx.StatusCode(iris.StatusOK)
	} else {
		log.WithFields(log.Fields{"subdomain": a.Subdomain, "txt": a.Value}).Debug("Bad data for subdomain")
		webUpdatePostError(ctx, errors.New("bad data"), iris.StatusBadRequest)
		return
	}
}

func webUpdatePostError(ctx iris.Context, err error, status int) {
	errStr := fmt.Sprintf("%v", err)
	updJSON := iris.Map{"error": errStr}
	ctx.JSON(updJSON)
	ctx.StatusCode(status)
}
