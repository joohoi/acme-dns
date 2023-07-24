//go:build !test
// +build !test

package main

import (
	"context"
	"crypto/tls"
	"flag"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"syscall"

	"github.com/caddyserver/certmagic"
	legolog "github.com/go-acme/lego/v3/log"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
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
	dnsservers := make([]*DNSServer, 0)
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
		dnsServerUDP := NewDNSServer(DB, Config.General.Listen, udpProto, Config.General.Domain)
		dnsservers = append(dnsservers, dnsServerUDP)
		dnsServerUDP.ParseRecords(Config)
		dnsServerTCP := NewDNSServer(DB, Config.General.Listen, tcpProto, Config.General.Domain)
		dnsservers = append(dnsservers, dnsServerTCP)
		// No need to parse records from config again
		dnsServerTCP.Domains = dnsServerUDP.Domains
		dnsServerTCP.SOA = dnsServerUDP.SOA
		go dnsServerUDP.Start(errChan)
		go dnsServerTCP.Start(errChan)
	} else {
		dnsServer := NewDNSServer(DB, Config.General.Listen, Config.General.Proto, Config.General.Domain)
		dnsservers = append(dnsservers, dnsServer)
		dnsServer.ParseRecords(Config)
		go dnsServer.Start(errChan)
	}

	// HTTP API
	go startHTTPAPI(errChan, Config, dnsservers)

	// block waiting for error
	for {
		err = <-errChan
		if err != nil {
			log.Fatal(err)
		}
	}
}

func startHTTPAPI(errChan chan error, config DNSConfig, dnsservers []*DNSServer) {
	// Setup http logger
	logger := log.New()
	logwriter := logger.Writer()
	defer logwriter.Close()
	// Setup logging for different dependencies to log with logrus
	// Certmagic
	stdlog.SetOutput(logwriter)
	// Lego
	legolog.Logger = logger

	api := httprouter.New()
	c := cors.New(cors.Options{
		AllowedOrigins:     config.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              config.General.Debug,
	})
	if config.General.Debug {
		// Logwriter for saner log output
		c.Log = stdlog.New(logwriter, "", 0)
	}
	if !config.API.DisableRegistration {
		api.POST("/register", webRegisterPost)
	}
	api.POST("/update", Auth(webUpdatePost))
	api.GET("/health", healthCheck)

	host := config.API.IP + ":" + config.API.Port

	// TLS specific general settings
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	provider := NewChallengeProvider(dnsservers)
	storage := certmagic.FileStorage{Path: config.API.ACMECacheDir}

	// Set up certmagic for getting certificate for acme-dns api
	certmagic.DefaultACME.DNS01Solver = &provider
	certmagic.DefaultACME.Agreed = true
	switch config.API.TLS {
	case TlsTypeLetsEncrypt:
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	case TlsTypeAcmeCustom:
		certmagic.DefaultACME.CA = config.API.ACMEDir
	case TlsTypeLetsEncryptStaging:
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	default:
	}
	certmagic.DefaultACME.Email = config.API.ACMENotificationEmail
	magicConf := certmagic.NewDefault()
	magicConf.Storage = &storage
	magicConf.DefaultServerName = config.General.Domain

	magicCache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			return magicConf, nil
		},
	})

	magic := certmagic.New(magicCache, *magicConf)
	var err error
	switch config.API.TLS {
	case TlsTypeLetsEncrypt:
		fallthrough
	case TlsTypeLetsEncryptStaging:
		fallthrough
	case TlsTypeAcmeCustom:
		err = magic.ManageAsync(context.Background(), []string{config.General.Domain})
		if err != nil {
			errChan <- err
			return
		}
		cfg.GetCertificate = magic.GetCertificate
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stdlog.New(logwriter, "", 0),
		}
		log.WithFields(log.Fields{"host": host, "domain": config.General.Domain}).Info("Listening HTTPS")
		err = srv.ListenAndServeTLS("", "")
	case TlsTypeCert:
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stdlog.New(logwriter, "", 0),
		}
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTPS")
		err = srv.ListenAndServeTLS(config.API.TLSCertFullchain, config.API.TLSCertPrivkey)
	case TlsTypeNone:
		fallthrough
	default:
		log.WithFields(log.Fields{"host": host}).Info("Listening HTTP")
		err = http.ListenAndServe(host, c.Handler(api))
	}
	if err != nil {
		errChan <- err
	}
}
