//+build !test

package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"os"
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
	api := iris.New()
	api.Config.DisableBanner = true
	crs := cors.New(cors.Options{
		AllowedOrigins:     DNSConf.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              DNSConf.General.Debug,
	})
	api.Use(crs)
	var ForceAuth = authMiddleware{}
	api.Get("/register", webRegisterGet)
	api.Post("/register", webRegisterPost)
	api.Post("/update", ForceAuth.Serve, webUpdatePost)
	switch DNSConf.API.TLS {
	case "letsencrypt":
		listener, err := iris.LETSENCRYPTPROD(DNSConf.API.Domain)
		err = api.Serve(listener)
		if err != nil {
			log.Errorf("Error in HTTP server [%v]", err)
		}
	case "cert":
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.ListenTLS(host, DNSConf.API.TLSCertFullchain, DNSConf.API.TLSCertPrivkey)
	default:
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.Listen(host)
	}
}
