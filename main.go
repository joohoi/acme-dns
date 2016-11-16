package main

import (
	"fmt"
	"github.com/iris-contrib/middleware/cors"
	"github.com/kataras/iris"
	"github.com/miekg/dns"
	"github.com/op/go-logging"
	"os"
)

// Logging config
var log = logging.MustGetLogger("acme-dns")

// Global configuration struct
var DnsConf DnsConfig

var DB Database

// Static records
var RR Records

func main() {
	// Read global config
	config_tmp, err := ReadConfig("config.cfg")
	if err != nil {
		fmt.Printf("Got error %v\n", DnsConf.Logconfig.File)
		os.Exit(1)
	}
	DnsConf = config_tmp
	// Setup logging
	var logformat = logging.MustStringFormatter(DnsConf.Logconfig.Format)
	var logBackend *logging.LogBackend
	switch DnsConf.Logconfig.Logtype {
	default:
		// Setup logging - stdout
		logBackend = logging.NewLogBackend(os.Stdout, "", 0)
	case "file":
		// Logging to file
		logfh, err := os.OpenFile(DnsConf.Logconfig.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Could not open log file %s\n", DnsConf.Logconfig.File)
			os.Exit(1)
		}
		defer logfh.Close()
		logBackend = logging.NewLogBackend(logfh, "", 0)
	}

	logLevel := logging.AddModuleLevel(logBackend)
	switch DnsConf.Logconfig.Level {
	case "warning":
		logLevel.SetLevel(logging.WARNING, "")
	case "error":
		logLevel.SetLevel(logging.ERROR, "")
	case "info":
		logLevel.SetLevel(logging.INFO, "")
	}
	logFormatter := logging.NewBackendFormatter(logLevel, logformat)
	logging.SetBackend(logFormatter)

	// Read the default records in
	RR.Parse(DnsConf.General.StaticRecords)

	// Open database
	err = DB.Init("acme-dns.db")
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	defer DB.DB.Close()

	// DNS server part
	dns.HandleFunc(".", handleRequest)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			log.Errorf("%v", err)
			os.Exit(1)
		}
	}()

	// API server and endpoints
	api := iris.New()
	crs := cors.New(cors.Options{
		AllowedOrigins:     DnsConf.Api.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              DnsConf.General.Debug,
	})
	api.Use(crs)
	var ForceAuth AuthMiddleware = AuthMiddleware{}
	api.Get("/register", WebRegisterGet)
	api.Post("/register", WebRegisterPost)
	api.Post("/update", ForceAuth.Serve, WebUpdatePost)
	// TODO: migrate to api.Serve(iris.LETSENCRYPTPROD("mydomain.com"))
	switch DnsConf.Api.Tls {
	case "letsencrypt":
		host := DnsConf.Api.Domain + ":" + DnsConf.Api.Port
		api.Listen(host)
	case "cert":
		host := DnsConf.Api.Domain + ":" + DnsConf.Api.Port
		api.ListenTLS(host, DnsConf.Api.Tls_cert_fullchain, DnsConf.Api.Tls_cert_privkey)
	default:
		host := DnsConf.Api.Domain + ":" + DnsConf.Api.Port
		api.Listen(host)
	}
	if err != nil {
		log.Errorf("Error in HTTP server [%v]", err)
	}
	log.Debugf("Shutting down...")
}
