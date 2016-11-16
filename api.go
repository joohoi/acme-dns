package main

import (
	"errors"
	"fmt"
	"github.com/kataras/iris"
)

func GetHandlerMap() map[string]func(*iris.Context) {
	return map[string]func(*iris.Context){
		"/register": WebRegisterGet,
	}
}

func PostHandlerMap() map[string]func(*iris.Context) {
	return map[string]func(*iris.Context){
		"/register": WebRegisterPost,
		"/update":   WebUpdatePost,
	}
}

func (a AuthMiddleware) Serve(ctx *iris.Context) {
	usernameStr := ctx.RequestHeader("X-Api-User")
	password := ctx.RequestHeader("X-Api-Key")

	username, err := GetValidUsername(usernameStr)
	if err == nil && ValidKey(password) {
		au, err := DB.GetByUsername(username)
		if err == nil && CorrectPassword(password, au.Password) {
			log.Debugf("Accepted authentication from [%s]", usernameStr)
			ctx.Next()
			return
		}
		// To protect against timed side channel (never gonna give you up)
		CorrectPassword(password, "$2a$10$8JEFVNYYhLoBysjAxe2yBuXrkDojBQBkVpXEQgyQyjn43SvJ4vL36")
	}
	ctx.JSON(iris.StatusUnauthorized, iris.Map{"error": "unauthorized"})
}

func WebRegisterPost(ctx *iris.Context) {
	// Create new user
	nu, err := DB.Register()
	var regJSON iris.Map
	var regStatus int
	if err != nil {
		errstr := fmt.Sprintf("%v", err)
		regJSON = iris.Map{"username": "", "password": "", "domain": "", "error": errstr}
		regStatus = iris.StatusInternalServerError
	} else {
		regJSON = iris.Map{"username": nu.Username, "password": nu.Password, "fulldomain": nu.Subdomain + "." + DNSConf.General.Domain, "subdomain": nu.Subdomain}
		regStatus = iris.StatusCreated
	}
	log.Debugf("Successful registration, created user [%s]", nu.Username)
	ctx.JSON(regStatus, regJSON)
}

func WebRegisterGet(ctx *iris.Context) {
	// This is placeholder for now
	WebRegisterPost(ctx)
}

func WebUpdatePost(ctx *iris.Context) {
	// User auth done in middleware
	a := ACMETxt{}
	userStr := ctx.RequestHeader("X-API-User")
	username, err := GetValidUsername(userStr)
	if err != nil {
		log.Warningf("Error while getting username [%s]. This should never happen because of auth middlware.", userStr)
		WebUpdatePostError(ctx, err, iris.StatusUnauthorized)
		return
	}
	if err := ctx.ReadJSON(&a); err != nil {
		// Handle bad post data
		log.Warningf("Could not unmarshal: [%v]", err)
		WebUpdatePostError(ctx, err, iris.StatusBadRequest)
		return
	}
	a.Username = username
	// Do update
	if ValidSubdomain(a.Subdomain) && ValidTXT(a.Value) {
		err := DB.Update(a)
		if err != nil {
			log.Warningf("Error trying to update [%v]", err)
			WebUpdatePostError(ctx, errors.New("internal error"), iris.StatusInternalServerError)
			return
		}
		ctx.JSON(iris.StatusOK, iris.Map{"txt": a.Value})
	} else {
		log.Warningf("Bad data, subdomain: [%s], txt: [%s]", a.Subdomain, a.Value)
		WebUpdatePostError(ctx, errors.New("bad data"), iris.StatusBadRequest)
		return
	}
}

func WebUpdatePostError(ctx *iris.Context, err error, status int) {
	errStr := fmt.Sprintf("%v", err)
	updJSON := iris.Map{"error": errStr}
	ctx.JSON(status, updJSON)
}
