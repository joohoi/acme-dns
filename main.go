//+build !test

package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/cors"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	// Read global config
	configTmp := readConfig("config.cfg")
	DNSConf = configTmp

	setupLogging(DNSConf.Logconfig.Format, DNSConf.Logconfig.Level)

	// Read the default records in
	RR.Parse(DNSConf.General)

	// Open database
	newDB := new(acmedb)
	err := newDB.Init(DNSConf.Database.Engine, DNSConf.Database.Connection)
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	DB = newDB
	defer DB.Close()

	// DNS server
	startDNS(DNSConf.General.Listen, DNSConf.General.Proto)

	// HTTP API
	startHTTPAPI()

	log.Debugf("Shutting down...")
}

func startHTTPAPI() {
	api := iris.New(iris.Configuration{DisableBodyConsumptionOnUnmarshal: true})
	api.Adapt(httprouter.New())
	api.Adapt(cors.New(cors.Options{
		AllowedOrigins:     DNSConf.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              DNSConf.General.Debug,
	}))
	var ForceAuth = authMiddleware{}
	api.Post("/register", webRegisterPost)
	api.Post("/update", ForceAuth.Serve, webUpdatePost)
	switch DNSConf.API.TLS {
	/*case "letsencrypt":
	listener, err := iris.LETSENCRYPT(DNSConf.API.Domain)
	err = api.Serve(listener)
	if err != nil {
		log.Errorf("Error in HTTP server [%v]", err)
	}*/
	case "cert":
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.ListenTLS(host, DNSConf.API.TLSCertFullchain, DNSConf.API.TLSCertPrivkey)
	default:
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.Listen(host)
	}
}
