package api

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/acme-dns/acme-dns/pkg/acmedns"

	"github.com/caddyserver/certmagic"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
	"go.uber.org/zap"
)

type AcmednsAPI struct {
	Config  *acmedns.AcmeDnsConfig
	DB      acmedns.AcmednsDB
	Logger  *zap.SugaredLogger
	errChan chan error
}

func Init(config *acmedns.AcmeDnsConfig, db acmedns.AcmednsDB, logger *zap.SugaredLogger, errChan chan error) AcmednsAPI {
	a := AcmednsAPI{Config: config, DB: db, Logger: logger, errChan: errChan}
	return a
}

func (a *AcmednsAPI) Start(dnsservers []acmedns.AcmednsNS) {
	var err error
	//TODO: do we want to debug log the HTTP server?
	stderrorlog, err := zap.NewStdLogAt(a.Logger.Desugar(), zap.ErrorLevel)
	if err != nil {
		a.errChan <- err
		return
	}
	//legolog.Logger = stderrorlog
	api := httprouter.New()
	c := cors.New(cors.Options{
		AllowedOrigins:     a.Config.API.CorsOrigins,
		AllowedMethods:     []string{"GET", "POST"},
		OptionsPassthrough: false,
		Debug:              a.Config.General.Debug,
	})
	if a.Config.General.Debug {
		// Logwriter for saner log output
		c.Log = stderrorlog
	}
	if !a.Config.API.DisableRegistration {
		api.POST("/register", a.webRegisterPost)
	}
	api.POST("/update", a.Auth(a.webUpdatePost))
	api.GET("/health", a.healthCheck)

	host := a.Config.API.IP + ":" + a.Config.API.Port

	// TLS specific general settings
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	switch a.Config.API.TLS {
	case "letsencryptstaging":
		magic := a.setupTLS(dnsservers)
		err = magic.ManageAsync(context.Background(), []string{a.Config.General.Domain})
		if err != nil {
			a.errChan <- err
			return
		}
		cfg.GetCertificate = magic.GetCertificate

		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stderrorlog,
		}
		a.Logger.Infow("Listening HTTPS",
			"host", host,
			"domain", a.Config.General.Domain)
		err = srv.ListenAndServeTLS("", "")
	case "letsencrypt":
		magic := a.setupTLS(dnsservers)
		err = magic.ManageAsync(context.Background(), []string{a.Config.General.Domain})
		if err != nil {
			a.errChan <- err
			return
		}
		cfg.GetCertificate = magic.GetCertificate
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stderrorlog,
		}
		a.Logger.Infow("Listening HTTPS",
			"host", host,
			"domain", a.Config.General.Domain)
		err = srv.ListenAndServeTLS("", "")
	case "cert":
		srv := &http.Server{
			Addr:      host,
			Handler:   c.Handler(api),
			TLSConfig: cfg,
			ErrorLog:  stderrorlog,
		}
		a.Logger.Infow("Listening HTTPS",
			"host", host,
			"domain", a.Config.General.Domain)
		err = srv.ListenAndServeTLS(a.Config.API.TLSCertFullchain, a.Config.API.TLSCertPrivkey)
	default:
		a.Logger.Infow("Listening HTTP",
			"host", host)
		err = http.ListenAndServe(host, c.Handler(api))
	}
	if err != nil {
		a.errChan <- err
	}
}

func (a *AcmednsAPI) setupTLS(dnsservers []acmedns.AcmednsNS) *certmagic.Config {
	provider := NewChallengeProvider(dnsservers)
	certmagic.Default.Logger = a.Logger.Desugar()
	storage := certmagic.FileStorage{Path: a.Config.API.ACMECacheDir}

	// Set up certmagic for getting certificate for acme-dns api
	certmagic.DefaultACME.DNS01Solver = &provider
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Logger = a.Logger.Desugar()
	if a.Config.API.TLS == "letsencrypt" {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	} else {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}
	certmagic.DefaultACME.Email = a.Config.API.NotificationEmail

	magicConf := certmagic.Default
	magicConf.Logger = a.Logger.Desugar()
	magicConf.Storage = &storage
	magicConf.DefaultServerName = a.Config.General.Domain
	magicCache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			return &magicConf, nil
		},
		Logger: a.Logger.Desugar(),
	})
	magic := certmagic.New(magicCache, magicConf)
	return magic
}
