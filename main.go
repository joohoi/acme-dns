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
	var Config DNSConfig
	if fileExists("/etc/acme-dns/config.cfg") {
		Config = readConfig("/etc/acme-dns/config.cfg")
	} else {
		Config = readConfig("config.cfg")
	}

	setupLogging(Config.Logconfig.Format, Config.Logconfig.Level)

	// Read the default records in
	RR.Parse(Config.General)

	// Open database
	newDB := new(acmedb)
	err := newDB.Init(Config.Database.Engine, Config.Database.Connection)
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	DB = newDB
	defer DB.Close()

	// DNS server
	startDNS(Config.General.Listen, Config.General.Proto)

	// HTTP API
	startHTTPAPI()

	log.Debugf("Shutting down...")
}

func startHTTPAPI() {
	api := iris.New()
	api.Use(cors.New(cors.Options{
		AllowedOrigins:     Config.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              Config.General.Debug,
	}))
	var ForceAuth = authMiddleware{}
	api.Post("/register", webRegisterPost)
	api.Post("/update", ForceAuth.Serve, webUpdatePost)

	host := Config.API.Domain + ":" + Config.API.Port
	switch Config.API.TLS {
	case "letsencrypt":
		api.Run(iris.AutoTLS(host, Config.API.Domain, Config.API.LEmail), iris.WithoutBodyConsumptionOnUnmarshal)
	case "cert":
		api.Run(iris.TLS(host, Config.API.TLSCertFullchain, Config.API.TLSCertPrivkey), iris.WithoutBodyConsumptionOnUnmarshal)
	default:
		api.Run(iris.Addr(host), iris.WithoutBodyConsumptionOnUnmarshal)
	}
}
