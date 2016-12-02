package main

import (
	"errors"
	"github.com/gavv/httpexpect"
	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
	"testing"
)

func setupIris(t *testing.T, debug bool, noauth bool) *httpexpect.Expect {
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
	var ForceAuth = authMiddleware{}
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
	e := setupIris(t, false, false)
	e.POST("/register").Expect().
		Status(iris.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		NotContainsKey("error")

	allowfrom := []interface{}{
		"123.123.123.123/32",
		"1010.10.10.10/24",
		"invalid",
	}

	response := e.POST("/register").
		WithJSON(allowfrom).
		Expect().
		Status(iris.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		ContainsKey("allowfrom").
		NotContainsKey("error")

	response.Value("allowfrom").String().Equal("[\"123.123.123.123/32\"]")

}

func TestApiRegisterWithMockDB(t *testing.T) {
	e := setupIris(t, false, false)
	oldDb := DB.GetBackend()
	db, mock, _ := sqlmock.New()
	DB.SetBackend(db)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO records").WillReturnError(errors.New("error"))
	e.POST("/register").Expect().
		Status(iris.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
	DB.SetBackend(oldDb)
}

func TestApiUpdateWithoutCredentials(t *testing.T) {
	e := setupIris(t, false, false)
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

	e := setupIris(t, false, false)
	newUser, err := DB.Register(cidrslice{})
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

	e := setupIris(t, false, true)
	oldDb := DB.GetBackend()
	db, mock, _ := sqlmock.New()
	DB.SetBackend(db)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("UPDATE records").WillReturnError(errors.New("error"))
	e.POST("/update").
		WithJSON(updateJSON).
		Expect().
		Status(iris.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
	DB.SetBackend(oldDb)
}

func TestApiManyUpdateWithCredentials(t *testing.T) {
	// TODO: transfer to using httpexpect builder
	// If test fails and more debug info is needed, use setupIris(t, true, false)
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	e := setupIris(t, false, false)
	// User without defined CIDR masks
	newUser, err := DB.Register(cidrslice{})
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}

	// User with defined allow from - CIDR masks, all invalid
	// (httpexpect doesn't provide a way to mock remote ip)
	newUserWithCIDR, err := DB.Register(cidrslice{"192.168.1.1/32", "invalid"})
	if err != nil {
		t.Errorf("Could not create new user with CIDR, got error [%v]", err)
	}

	// Another user with valid CIDR mask to match the httpexpect default
	newUserWithValidCIDR, err := DB.Register(cidrslice{"0.0.0.0/32", "invalid"})
	if err != nil {
		t.Errorf("Could not create new user with a valid CIDR, got error [%v]", err)
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
		{newUserWithCIDR.Username.String(), newUserWithCIDR.Password, newUserWithCIDR.Subdomain, validTxtData, 401},
		{newUserWithValidCIDR.Username.String(), newUserWithValidCIDR.Password, newUserWithValidCIDR.Subdomain, validTxtData, 200},
		{newUser.Username.String(), "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", newUser.Subdomain, validTxtData, 401},
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
