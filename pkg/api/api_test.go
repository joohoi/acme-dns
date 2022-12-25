package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/acme-dns/acme-dns/pkg/acmedns"
	"github.com/acme-dns/acme-dns/pkg/database"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gavv/httpexpect"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

func fakeConfigAndLogger() (acmedns.AcmeDnsConfig, *zap.SugaredLogger) {
	c := acmedns.AcmeDnsConfig{}
	c.Database.Engine = "sqlite"
	c.Database.Connection = ":memory:"
	l := zap.NewNop().Sugar()
	return c, l
}

// noAuth function to write ACMETxt model to context while not preforming any validation
func noAuth(update httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		postData := acmedns.ACMETxt{}
		uname := r.Header.Get("X-Api-User")
		passwd := r.Header.Get("X-Api-Key")

		dec := json.NewDecoder(r.Body)
		_ = dec.Decode(&postData)
		// Set user info to the decoded ACMETxt object
		postData.Username, _ = uuid.Parse(uname)
		postData.Password = passwd
		// Set the ACMETxt struct to context to pull in from update function
		ctx := r.Context()
		ctx = context.WithValue(ctx, ACMETxtKey, postData)
		r = r.WithContext(ctx)
		update(w, r, p)
	}
}

func getExpect(t *testing.T, server *httptest.Server) *httpexpect.Expect {
	return httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  server.URL,
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewCurlPrinter(t),
			httpexpect.NewDebugPrinter(t, true),
		},
	})
}

func setupRouter(debug bool, noauth bool) (http.Handler, AcmednsAPI, acmedns.AcmednsDB) {
	api := httprouter.New()
	config, logger := fakeConfigAndLogger()
	config.API.Domain = ""
	config.API.Port = "8080"
	config.API.TLS = "none"
	config.API.CorsOrigins = []string{"*"}
	config.API.UseHeader = true
	config.API.HeaderName = "X-Forwarded-For"

	db, _ := database.Init(&config, logger)
	errChan := make(chan error, 1)
	adnsapi := Init(&config, db, logger, errChan)
	c := cors.New(cors.Options{
		AllowedOrigins:     config.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              config.General.Debug,
	})
	api.POST("/register", adnsapi.webRegisterPost)
	api.GET("/health", adnsapi.healthCheck)
	if noauth {
		api.POST("/update", noAuth(adnsapi.webUpdatePost))
	} else {
		api.POST("/update", adnsapi.Auth(adnsapi.webUpdatePost))
	}
	return c.Handler(api), adnsapi, db
}

func TestApiRegister(t *testing.T) {
	router, _, _ := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	e.POST("/register").Expect().
		Status(http.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		NotContainsKey("error")

	allowfrom := map[string][]interface{}{
		"allowfrom": []interface{}{"123.123.123.123/32",
			"2001:db8:a0b:12f0::1/32",
			"[::1]/64",
		},
	}

	response := e.POST("/register").
		WithJSON(allowfrom).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().
		ContainsKey("fulldomain").
		ContainsKey("subdomain").
		ContainsKey("username").
		ContainsKey("password").
		ContainsKey("allowfrom").
		NotContainsKey("error")

	response.Value("allowfrom").Array().Elements("123.123.123.123/32", "2001:db8:a0b:12f0::1/32", "::1/64")
}

func TestApiRegisterBadAllowFrom(t *testing.T) {
	router, _, _ := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	invalidVals := []string{
		"invalid",
		"1.2.3.4/33",
		"1.2/24",
		"1.2.3.4",
		"12345:db8:a0b:12f0::1/32",
		"1234::123::123::1/32",
	}

	for _, v := range invalidVals {

		allowfrom := map[string][]interface{}{
			"allowfrom": []interface{}{v}}

		response := e.POST("/register").
			WithJSON(allowfrom).
			Expect().
			Status(http.StatusBadRequest).
			JSON().Object().
			ContainsKey("error")

		response.Value("error").Equal("invalid_allowfrom_cidr")
	}
}

func TestApiRegisterMalformedJSON(t *testing.T) {
	router, _, _ := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)

	malPayloads := []string{
		"{\"allowfrom': '1.1.1.1/32'}",
		"\"allowfrom\": \"1.1.1.1/32\"",
		"{\"allowfrom\": \"[1.1.1.1/32]\"",
		"\"allowfrom\": \"1.1.1.1/32\"}",
		"{allowfrom: \"1.2.3.4\"}",
		"{allowfrom: [1.2.3.4]}",
		"whatever that's not a json payload",
	}
	for _, test := range malPayloads {
		e.POST("/register").
			WithBytes([]byte(test)).
			Expect().
			Status(http.StatusBadRequest).
			JSON().Object().
			ContainsKey("error").
			NotContainsKey("subdomain").
			NotContainsKey("username")
	}
}

func TestApiRegisterWithMockDB(t *testing.T) {
	router, _, db := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	oldDb := db.GetBackend()
	mdb, mock, _ := sqlmock.New()
	db.SetBackend(mdb)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO records").WillReturnError(errors.New("error"))
	e.POST("/register").Expect().
		Status(http.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
	db.SetBackend(oldDb)
}

func TestApiUpdateWithInvalidSubdomain(t *testing.T) {
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	router, _, db := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	newUser, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}
	// Invalid subdomain data
	updateJSON["subdomain"] = "example.com"
	updateJSON["txt"] = validTxtData
	e.POST("/update").
		WithJSON(updateJSON).
		WithHeader("X-Api-User", newUser.Username.String()).
		WithHeader("X-Api-Key", newUser.Password).
		Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		ContainsKey("error").
		NotContainsKey("txt").
		ValueEqual("error", "forbidden")
}

func TestApiUpdateWithInvalidTxt(t *testing.T) {
	invalidTXTData := "idk m8 bbl lmao"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	router, _, db := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	newUser, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}
	updateJSON["subdomain"] = newUser.Subdomain
	// Invalid txt data
	updateJSON["txt"] = invalidTXTData
	e.POST("/update").
		WithJSON(updateJSON).
		WithHeader("X-Api-User", newUser.Username.String()).
		WithHeader("X-Api-Key", newUser.Password).
		Expect().
		Status(http.StatusBadRequest).
		JSON().Object().
		ContainsKey("error").
		NotContainsKey("txt").
		ValueEqual("error", "bad_txt")
}

func TestApiUpdateWithoutCredentials(t *testing.T) {
	router, _, _ := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	e.POST("/update").Expect().
		Status(http.StatusUnauthorized).
		JSON().Object().
		ContainsKey("error").
		NotContainsKey("txt")
}

func TestApiUpdateWithCredentials(t *testing.T) {
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	updateJSON := map[string]interface{}{
		"subdomain": "",
		"txt":       ""}

	router, _, db := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	newUser, err := db.Register(acmedns.Cidrslice{})
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
		Status(http.StatusOK).
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

	router, _, db := setupRouter(false, true)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	oldDb := db.GetBackend()
	mdb, mock, _ := sqlmock.New()
	db.SetBackend(mdb)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectPrepare("UPDATE records").WillReturnError(errors.New("error"))
	e.POST("/update").
		WithJSON(updateJSON).
		Expect().
		Status(http.StatusInternalServerError).
		JSON().Object().
		ContainsKey("error")
	db.SetBackend(oldDb)
}

func TestApiManyUpdateWithCredentials(t *testing.T) {
	validTxtData := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	router, _, db := setupRouter(true, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	// User without defined CIDR masks
	newUser, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}

	// User with defined allow from - CIDR masks, all invalid
	// (httpexpect doesn't provide a way to mock remote ip)
	newUserWithCIDR, err := db.Register(acmedns.Cidrslice{"192.168.1.1/32", "invalid"})
	if err != nil {
		t.Errorf("Could not create new user with CIDR, got error [%v]", err)
	}

	// Another user with valid CIDR mask to match the httpexpect default
	newUserWithValidCIDR, err := db.Register(acmedns.Cidrslice{"10.1.2.3/32", "invalid"})
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
		updateJSON := map[string]interface{}{
			"subdomain": test.subdomain,
			"txt":       test.txt}
		e.POST("/update").
			WithJSON(updateJSON).
			WithHeader("X-Api-User", test.user).
			WithHeader("X-Api-Key", test.pass).
			WithHeader("X-Forwarded-For", "10.1.2.3").
			Expect().
			Status(test.status)
	}
}

func TestApiManyUpdateWithIpCheckHeaders(t *testing.T) {

	router, adnsapi, db := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	// Use header checks from default header (X-Forwarded-For)
	adnsapi.Config.API.UseHeader = true
	// User without defined CIDR masks
	newUser, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Could not create new user, got error [%v]", err)
	}

	newUserWithCIDR, err := db.Register(acmedns.Cidrslice{"192.168.1.2/32", "invalid"})
	if err != nil {
		t.Errorf("Could not create new user with CIDR, got error [%v]", err)
	}

	newUserWithIP6CIDR, err := db.Register(acmedns.Cidrslice{"2002:c0a8::0/32"})
	if err != nil {
		t.Errorf("Could not create a new user with IP6 CIDR, got error [%v]", err)
	}

	for _, test := range []struct {
		user        acmedns.ACMETxt
		headerValue string
		status      int
	}{
		{newUser, "whatever goes", 200},
		{newUser, "10.0.0.1, 1.2.3.4 ,3.4.5.6", 200},
		{newUserWithCIDR, "127.0.0.1", 401},
		{newUserWithCIDR, "10.0.0.1, 10.0.0.2, 192.168.1.3", 401},
		{newUserWithCIDR, "10.1.1.1 ,192.168.1.2, 8.8.8.8", 200},
		{newUserWithIP6CIDR, "2002:c0a8:b4dc:0d3::0", 200},
		{newUserWithIP6CIDR, "2002:c0a7:0ff::0", 401},
		{newUserWithIP6CIDR, "2002:c0a8:d3ad:b33f:c0ff:33b4:dc0d:3b4d", 200},
	} {
		updateJSON := map[string]interface{}{
			"subdomain": test.user.Subdomain,
			"txt":       "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
		e.POST("/update").
			WithJSON(updateJSON).
			WithHeader("X-Api-User", test.user.Username.String()).
			WithHeader("X-Api-Key", test.user.Password).
			WithHeader("X-Forwarded-For", test.headerValue).
			Expect().
			Status(test.status)
	}
	adnsapi.Config.API.UseHeader = false
}

func TestApiHealthCheck(t *testing.T) {
	router, _, _ := setupRouter(false, false)
	server := httptest.NewServer(router)
	defer server.Close()
	e := getExpect(t, server)
	e.GET("/health").Expect().Status(http.StatusOK)
}

func TestGetIPListFromHeader(t *testing.T) {
	for i, test := range []struct {
		input  string
		output []string
	}{
		{"1.1.1.1, 2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
		{" 1.1.1.1 , 2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
		{",1.1.1.1 ,2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
	} {
		res := getIPListFromHeader(test.input)
		if len(res) != len(test.output) {
			t.Errorf("Test %d: Expected [%d] items in return list, but got [%d]", i, len(test.output), len(res))
		} else {

			for j, vv := range test.output {
				if res[j] != vv {
					t.Errorf("Test %d: Expected return value [%v] but got [%v]", j, test.output, res)
				}

			}
		}
	}
}

func TestUpdateAllowedFromIP(t *testing.T) {
	_, adnsapi, _ := setupRouter(false, false)
	adnsapi.Config.API.UseHeader = false
	userWithAllow := acmedns.NewACMETxt()
	userWithAllow.AllowFrom = acmedns.Cidrslice{"192.168.1.2/32", "[::1]/128"}
	userWithoutAllow := acmedns.NewACMETxt()

	for i, test := range []struct {
		remoteaddr string
		expected   bool
	}{
		{"192.168.1.2:1234", true},
		{"192.168.1.1:1234", false},
		{"invalid", false},
		{"[::1]:4567", true},
	} {
		newreq, _ := http.NewRequest("GET", "/whatever", nil)
		newreq.RemoteAddr = test.remoteaddr
		ret := adnsapi.updateAllowedFromIP(newreq, userWithAllow)
		if test.expected != ret {
			t.Errorf("Test %d: Unexpected result for user with allowForm set", i)
		}

		if !adnsapi.updateAllowedFromIP(newreq, userWithoutAllow) {
			t.Errorf("Test %d: Unexpected result for user without allowForm set", i)
		}
	}
}
