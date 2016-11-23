package main

import (
	"errors"
	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/op/go-logging"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"
)

func SetupIris(t *testing.T, debug bool, noauth bool) *httpexpect.Expect {
	iris.ResetDefault()
	var dbcfg = dbsettings{
		Engine:     "sqlite3",
		Connection: ":memory:"}
	var httpapicfg = httpapi{
		Domain:      "",
		Port:        "8080",
		TLS:         "none",
		CorsOrigins: []string{"*"},
	}
	var dnscfg = DNSConfig{
		API:      httpapicfg,
		Database: dbcfg,
	}
	DNSConf = dnscfg
	// In memory logger
	logging.InitForTesting(logging.DEBUG)
	err := DB.Init(DNSConf.Database.Engine, DNSConf.Database.Connection)
	if err != nil {
		panic(err)
	}
	var ForceAuth = authMiddleware{}
	iris.Get("/register", webRegisterGet)
	iris.Post("/register", webRegisterPost)
	if noauth {
		iris.Post("/update", webUpdatePost)
	} else {
		iris.Post("/update", ForceAuth.Serve, webUpdatePost)
	}
	httptestcfg := httptest.DefaultConfiguration()
	httptestcfg.Debug = debug
	return httptest.New(iris.Default, t, httptestcfg)
}

func TestApiRegister(t *testing.T) {
	e := SetupIris(t, false, false)
	defer DB.DB.Close()
	e.GET("/register").Expect().
		Status(iris.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		NotContainsKey("error")
	e.POST("/register").Expect().
		Status(iris.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		NotContainsKey("error")
}

func TestApiRegisterWithMockDB(t *testing.T) {
	e := SetupIris(t, false, false)
	DB.DB.Close()
	db, mock, _ := sqlmock.New()
	DB.DB = db
	defer DB.DB.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO records").WillReturnError(errors.New("error"))
	e.GET("/register").Expect().
		Status(iris.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
}

func TestApiUpdateWithoutCredentials(t *testing.T) {
	e := SetupIris(t, false, false)
	defer DB.DB.Close()
	e.POST("/update").Expect().
		Status(iris.StatusUnauthorized).
		JSON().Object().
		ContainsKey("error").
		NotContainsKey("txt")
}

func TestApiUpdateWithCredentials(t *testing.T) {
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	e := SetupIris(t, false, false)
	defer DB.DB.Close()
	newUser, err := DB.Register()
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}
	// Valid data
	updateJSON["subdomain"] = newUser.Subdomain
	updateJSON["txt"] = validTxtData

	e.POST("/update").
		WithJSON(updateJSON).
		WithHeader("X-Api-User", newUser.Username.String()).
		WithHeader("X-Api-Key", newUser.Password).
		Expect().
		Status(iris.StatusOK).
		JSON().Object().
		ContainsKey("txt").
		NotContainsKey("error").
		ValueEqual("txt", validTxtData)
}

func TestApiUpdateWithCredentialsMockDB(t *testing.T) {
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	// Valid data
	updateJSON["subdomain"] = "a097455b-52cc-4569-90c8-7a4b97c6eba8"
	updateJSON["txt"] = validTxtData

	e := SetupIris(t, false, true)
	DB.DB.Close()
	db, mock, _ := sqlmock.New()
	DB.DB = db
	defer DB.DB.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("UPDATE records").WillReturnError(errors.New("error"))
	e.POST("/update").
		WithJSON(updateJSON).
		Expect().
		Status(iris.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
}

func TestApiManyUpdateWithCredentials(t *testing.T) {
	// TODO: transfer to using httpexpect builder
	// If test fails and more debuf info is needed, use SetupIris(t, true, false)
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	e := SetupIris(t, true, false)
	defer DB.DB.Close()
	newUser, err := DB.Register()
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}
	for _, test := range []struct {
		user      string
		pass      string
		subdomain string
		txt       interface{}
		status    int
	}{
		{"non-uuid-user", "tooshortpass", "non-uuid-subdomain", validTxtData, 401},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", "tooshortpass", "bb97455b-52cc-4569-90c8-7a4b97c6eba8", validTxtData, 401},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", "LongEnoughPassButNoUserExists___________", "bb97455b-52cc-4569-90c8-7a4b97c6eba8", validTxtData, 401},
		{newUser.Username.String(), newUser.Password, "a097455b-52cc-4569-90c8-7a4b97c6eba8", validTxtData, 401},
		{newUser.Username.String(), newUser.Password, newUser.Subdomain, "tooshortfortxt", 400},
		{newUser.Username.String(), newUser.Password, newUser.Subdomain, 1234567890, 400},
		{newUser.Username.String(), newUser.Password, newUser.Subdomain, validTxtData, 200},
	} {
		updateJSON = map[string]interface{}{
			"subdomain": test.subdomain,
			"txt":       test.txt}
		e.POST("/update").
			WithJSON(updateJSON).
			WithHeader("X-Api-User", test.user).
			WithHeader("X-Api-Key", test.pass).
			Expect().
			Status(test.status)
	}
}
