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
	ctx.JSON(reg_status, reg_json)
}

func WebRegisterGet(ctx *iris.Context) {
	// This is placeholder for now
	WebRegisterPost(ctx)
}

func WebUpdatePost(ctx *iris.Context) {
	var username, password string
	var a ACMETxtPost = ACMETxtPost{}
	username = ctx.RequestHeader("X-API-User")
	password = ctx.RequestHeader("X-API-Key")
	if err := ctx.ReadJSON(&a); err != nil {
		// Handle bad post data
		WebUpdatePostError(ctx, err, iris.StatusBadRequest)
		return
	}
	// Sanitized by db function
	euser, err := DB.GetByUsername(username)
	if err != nil {
		// DB error
		WebUpdatePostError(ctx, err, iris.StatusInternalServerError)
		return
	}
	if len(euser) == 0 {
		// User not found
		// TODO: do bcrypt to avoid side channel
		WebUpdatePostError(ctx, errors.New("invalid user or api key"), iris.StatusUnauthorized)
		return
	}
	// Get first (and the only) user
	upduser := euser[0]
	// Validate password
	if upduser.Password != password {
		// Invalid password
		WebUpdatePostError(ctx, errors.New("invalid user or api key"), iris.StatusUnauthorized)
		return
	} else {
		// Do update
		if len(a.Value) == 0 {
			WebUpdatePostError(ctx, errors.New("missing txt value"), iris.StatusBadRequest)
			return
		} else {
			upduser.Value = a.Value
			err = DB.Update(upduser)
			if err != nil {
				WebUpdatePostError(ctx, err, iris.StatusInternalServerError)
				return
			}
			// All ok
			ctx.JSON(iris.StatusOK, iris.Map{"txt": upduser.Value})
		}
	}

}

func WebUpdatePostError(ctx *iris.Context, err error, status int) {
	err_str := fmt.Sprintf("%v", err)
	upd_json := iris.Map{"error": err_str}
	ctx.JSON(status, upd_json)
}
