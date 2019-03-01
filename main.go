//+build !test

package main

import (
	"crypto/tls"
	"flag"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	// Created files are not world writable
	syscall.Umask(0077)
	configPtr := flag.String("c", "/etc/acme-dns/config.cfg", "config file location")
	flag.Parse()
	// Read global config
	var err error
	if fileIsAccessible(*configPtr) {
		log.WithFields(log.Fields{"file": *configPtr}).Info("Using config file")
		Config, err = readConfig(*configPtr)
	} else if fileIsAccessible("./config.cfg") {
		log.WithFields(log.Fields{"file": "./config.cfg"}).Info("Using config file")
		Config, err = readConfig("./config.cfg")
	} else {
		log.Errorf("Configuration file not found.")
		os.Exit(1)
	}
	if err != nil {
		log.Errorf("Encountered an error while trying to read configuration file:  %s", err)
		os.Exit(1)
	}

	setupLogging(Config.Logconfig.Format, Config.Logconfig.Level)

	// Open database
	newDB := new(acmedb)
	err = newDB.Init(Config.Database.Engine, Config.Database.Connection)
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	} else {
		log.Info("Connected to database")
	}
	DB = newDB
	defer DB.Close()

	// Error channel for servers
	errChan := make(chan error, 1)

	// DNS server
	if strings.HasPrefix(Config.General.Proto, "both") {
		// Handle the case where DNS server should be started for both udp and tcp
		udpProto := "udp"
		tcpProto := "tcp"
		if strings.HasSuffix(Config.General.Proto, "4") {
			udpProto += "4"
			tcpProto += "4"
		} else if strings.HasSuffix(Config.General.Proto, "6") {
			udpProto += "6"
			tcpProto += "6"
		}
		dnsServerUDP := NewDNSServer(DB, Config.General.Listen, udpProto)
		dnsServerUDP.ParseRecords(Config)
		dnsServerTCP := NewDNSServer(DB, Config.General.Listen, tcpProto)
		// No need to parse records from config again
		dnsServerTCP.Domains = dnsServerUDP.Domains
		dnsServerTCP.SOA = dnsServerUDP.SOA
		go dnsServerUDP.Start(errChan)
		go dnsServerTCP.Start(errChan)
	} else {
		dnsServer := NewDNSServer(DB, Config.General.Listen, Config.General.Proto)
		dnsServer.ParseRecords(Config)
		go dnsServer.Start(errChan)
	}

	// HTTP API
	go startHTTPAPI(errChan)

	// block waiting for error
	select {
	case err = <-errChan:
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Debugf("Shutting down...")
}

func startHTTPAPI(errChan chan error) {
	// Setup http logger
	logger := log.New()
	logwriter := logger.Writer()
	defer logwriter.Close()
	api := httprouter.New()
	c := cors.New(cors.Options{
		AllowedOrigins:     Config.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              Config.General.Debug,
	})
	if Config.General.Debug {
		// Logwriter for saner log output
		c.Log = stdlog.New(logwriter, "", 0)
	}
	if !Config.API.DisableRegistration {
		api.POST("/register", webRegisterPost)
	}
	api.POST("/update", Auth(webUpdatePost))
	api.GET("/health", healthCheck)

	host := Config.API.IP + ":" + Config.API.Port

	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	var err error
	switch Config.API.TLS {
	case "letsencrypt":
		m := autocert.Manager{
			Cache:      autocert.DirCache(Config.API.ACMECacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(Config.API.Domain),
		}
		autocerthost := Config.API.IP + ":" + Config.API.AutocertPort
		log.WithFields(log.Fields{"autocerthost": autocerthost, "domain": Config.API.Domain}).Debug("Opening HTTP port for autocert")
		go http.ListenAndServe(autocerthost, m.HTTPHandler(nil))
		cfg.GetCertificate = m.GetCertificate
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stdlog.New(logwriter, "", 0),
		}
		log.WithFields(log.Fields{"host": host, "domain": Config.API.Domain}).Info("Listening HTTPS, using certificate from autocert")
		err = srv.ListenAndServeTLS("", "")
	case "cert":
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stdlog.New(logwriter, "", 0),
		}
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTPS")
		err = srv.ListenAndServeTLS(Config.API.TLSCertFullchain, Config.API.TLSCertPrivkey)
	default:
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTP")
		err = http.ListenAndServe(host, c.Handler(api))
	}
	if err != nil {
		errChan <- err
	}
}
