package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"os"
)

// DNSConf is global configuration struct
var DNSConf DNSConfig

// DB is used to access the database functions in acme-dns
var DB database

// RR holds the static DNS records
var RR Records

func main() {
	// Read global config
	configTmp, err := readConfig("config.cfg")
	if err != nil {
		fmt.Printf("Got error %v\n", err)
		os.Exit(1)
	}
	DNSConf = configTmp

	setupLogging(DNSConf.Logconfig.Format, DNSConf.Logconfig.Level)

	// Read the default records in
	RR.Parse(DNSConf.General.StaticRecords)

	// Open database
	err = DB.Init(DNSConf.Database.Engine, DNSConf.Database.Connection)
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	defer DB.DB.Close()

	// DNS server
	startDNS(DNSConf.General.Listen)

	// API server and endpoints
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
	// TODO: migrate to api.Serve(iris.LETSENCRYPTPROD("mydomain.com"))
	switch DNSConf.API.TLS {
	case "letsencrypt":
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.Listen(host)
	case "cert":
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.ListenTLS(host, DNSConf.API.TLSCertFullchain, DNSConf.API.TLSCertPrivkey)
	default:
		host := DNSConf.API.Domain + ":" + DNSConf.API.Port
		api.Listen(host)
	}
	if err != nil {
		log.Errorf("Error in HTTP server [%v]", err)
	}
	log.Debugf("Shutting down...")
}
