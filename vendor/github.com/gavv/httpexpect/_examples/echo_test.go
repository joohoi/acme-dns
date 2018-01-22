package examples

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect"
)

// Echo JWT token authentication tests.
//
// This test is executed for the EchoHandler() in two modes:
//  - via http client
//  - via http.Handler
func testEcho(e *httpexpect.Expect) {
	type Login struct {
		Username string `form:"username"`
		Password string `form:"password"`
	}

	e.POST("/login").WithForm(Login{"ford", "<bad password>"}).
		Expect().
		Status(http.StatusUnauthorized)

	r := e.POST("/login").WithForm(Login{"ford", "betelgeuse7"}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	r.Keys().ContainsOnly("token")

	token := r.Value("token").String().Raw()

	e.GET("/restricted/hello").
		Expect().
		Status(http.StatusBadRequest)

	e.GET("/restricted/hello").WithHeader("Authorization", "Bearer <bad token>").
		Expect().
		Status(http.StatusUnauthorized)

	e.GET("/restricted/hello").WithHeader("Authorization", "Bearer "+token).
		Expect().
		Status(http.StatusOK).Body().Equal("hello, world!")

	auth := e.Builder(func(req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	auth.GET("/restricted/hello").
		Expect().
		Status(http.StatusOK).Body().Equal("hello, world!")
}

func TestEchoClient(t *testing.T) {
	handler := EchoHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	e := httpexpect.WithConfig(httpexpect.Config{
		BaseURL:  server.URL,
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})

	testEcho(e)
}

func TestEchoHandler(t *testing.T) {
	handler := EchoHandler()

	e := httpexpect.WithConfig(httpexpect.Config{
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
		Printers: []httpexpect.Printer{
			httpexpect.NewDebugPrinter(t, true),
		},
	})

	testEcho(e)
}
