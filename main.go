//+build !test

package main

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	// Read global config
	if fileExists("/etc/acme-dns/config.cfg") {
		Config = readConfig("/etc/acme-dns/config.cfg")
		log.WithFields(log.Fields{"file": "/etc/acme-dns/config.cfg"}).Info("Using config file")

	} else {
		log.WithFields(log.Fields{"file": "./config.cfg"}).Info("Using config file")
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
	api := httprouter.New()
	//api.Use(cors.New(cors.Options{
	//	AllowedOrigins:     Config.API.CorsOrigins,
	//	AllowedMethods:     []string{"GET", "POST"},
	//	OptionsPassthrough: false,
	//	Debug:              Config.General.Debug,
	//}))
	api.POST("/register", webRegisterPost)
	api.POST("/update", Auth(webUpdatePost))

	host := Config.API.Domain + ":" + Config.API.Port

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	switch Config.API.TLS {
	case "letsencrypt":
		m := autocert.Manager{
			Cache:      autocert.DirCache("api-certs"),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(Config.API.Domain),
		}
		cfg.GetCertificate = m.GetCertificate
		srv := &http.Server{
			Addr:      host,
			Handler:   api,
			TLSConfig: cfg,
		}
		log.Fatal(srv.ListenAndServeTLS("", ""))
	case "cert":
		srv := &http.Server{
			Addr:      host,
			Handler:   api,
			TLSConfig: cfg,
		}
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTPS")
		log.Fatal(srv.ListenAndServeTLS(Config.API.TLSCertFullchain, Config.API.TLSCertPrivkey))
	default:
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTP")
		log.Fatal(http.ListenAndServe(host, api))
	}
}
