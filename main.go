//+build !test

package main

import (
	"os"

	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	log "github.com/sirupsen/logrus"
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
	api.Use(cors.New(cors.Options{
		AllowedOrigins:     DNSConf.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              DNSConf.General.Debug,
	}))
	var ForceAuth = authMiddleware{}
	api.Post("/register", webRegisterPost)
	api.Post("/update", ForceAuth.Serve, webUpdatePost)

	host := DNSConf.API.Domain + ":" + DNSConf.API.Port
	switch DNSConf.API.TLS {
	case "letsencrypt":
		api.Run(iris.AutoTLS(host, DNSConf.API.Domain, DNSConf.API.LEmail), iris.WithoutBodyConsumptionOnUnmarshal)
	case "cert":
		api.Run(iris.TLS(host, DNSConf.API.TLSCertFullchain, DNSConf.API.TLSCertPrivkey), iris.WithoutBodyConsumptionOnUnmarshal)
	default:
		api.Run(iris.Addr(host), iris.WithoutBodyConsumptionOnUnmarshal)
	}
}
