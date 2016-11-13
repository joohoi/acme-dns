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
	username_str := ctx.RequestHeader("X-Api-User")
	password := ctx.RequestHeader("X-Api-Key")

	username, err := GetValidUsername(username_str)
	if err == nil && ValidKey(password) {
		au, err := DB.GetByUsername(username)
		if err == nil && CorrectPassword(password, au.Password) {
			log.Debugf("Accepted authentication from [%s]", username_str)
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
	var reg_json iris.Map
	var reg_status int
	if err != nil {
		errstr := fmt.Sprintf("%v", err)

		reg_json = iris.Map{"username": "", "password": "", "domain": "", "error": errstr}
		reg_status = iris.StatusInternalServerError
	} else {
		reg_json = iris.Map{"username": nu.Username, "password": nu.Password, "fulldomain": nu.Subdomain + "." + DnsConf.General.Domain, "subdomain": nu.Subdomain}
		reg_status = iris.StatusCreated
	}
	log.Debugf("Successful registration, created user [%s]", nu.Username)
	ctx.JSON(reg_status, reg_json)
}

func WebRegisterGet(ctx *iris.Context) {
	// This is placeholder for now
	WebRegisterPost(ctx)
}

func WebUpdatePost(ctx *iris.Context) {
	// User auth done in middleware
	var a ACMETxt = ACMETxt{}
	user_string := ctx.RequestHeader("X-API-User")
	username, err := GetValidUsername(user_string)
	if err != nil {
		log.Warningf("Error while getting username [%s]. This should never happen because of auth middlware.", user_string)
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
	err_str := fmt.Sprintf("%v", err)
	upd_json := iris.Map{"error": err_str}
	ctx.JSON(status, upd_json)
}
