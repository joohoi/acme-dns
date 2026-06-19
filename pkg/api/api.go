package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/joohoi/acme-dns/pkg/acmedns"

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

func (a *AcmednsAPI) buildHSTSHeader() string {
	if !a.Config.API.HSTSEnabled {
		return ""
	}

	maxAge := a.Config.API.HSTSMaxAge
	if maxAge <= 0 {
		maxAge = 31536000
	}

	header := fmt.Sprintf("max-age=%d", maxAge)

	if a.Config.API.HSTSIncludeSubDom {
		header += "; includeSubDomains"
	}

	if a.Config.API.HSTSPreload {
		header += "; preload"
	}

	return header
}

func (a *AcmednsAPI) hstsMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		hstsHeader := a.buildHSTSHeader()
		if hstsHeader != "" {
			w.Header().Set("Strict-Transport-Security", hstsHeader)
		}
		next(w, r, ps)
	}
}

func (a *AcmednsAPI) Start(dnsservers []acmedns.AcmednsNS) {
	var err error
	//TODO: do we want to debug log the HTTP server?
	stderrorlog, err := zap.NewStdLogAt(a.Logger.Desugar(), zap.ErrorLevel)
	if err != nil {
		a.errChan <- err
		return
	}
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
		api.POST("/register", a.hstsMiddleware(a.webRegisterPost))
	}
	api.POST("/update", a.hstsMiddleware(a.Auth(a.webUpdatePost)))
	api.GET("/health", a.hstsMiddleware(a.healthCheck))

	host := a.Config.API.IP + ":" + a.Config.API.Port

	// TLS specific general settings
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	switch a.Config.API.TLS {
	case acmedns.ApiTlsProviderLetsEncrypt, acmedns.ApiTlsProviderLetsEncryptStaging, acmedns.ApiTlsProviderCustom:
		magic := a.setupTLS(dnsservers)
		err = magic.ManageAsync(context.Background(), []string{a.Config.General.Domain})
		if err != nil {
			a.errChan <- err
			return
		}
		cfg.GetCertificate = magic.GetCertificate
		srv := &http.Server{
			Addr:         host,
			Handler:      c.Handler(api),
			TLSConfig:    cfg,
			ErrorLog:     stderrorlog,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		}
		a.Logger.Infow("Listening HTTPS",
			"host", host,
			"domain", a.Config.General.Domain)
		err = srv.ListenAndServeTLS("", "")
	case acmedns.ApiTlsProviderCert:
		srv := &http.Server{
			Addr:         host,
			Handler:      c.Handler(api),
			TLSConfig:    cfg,
			ErrorLog:     stderrorlog,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
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
	switch a.Config.API.TLS {
	case acmedns.ApiTlsProviderLetsEncrypt:
		certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	case acmedns.ApiTlsProviderLetsEncryptStaging:
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	case acmedns.ApiTlsProviderCustom:
		certmagic.DefaultACME.CA = a.Config.API.ACMEDirURL
	default:
		a.Logger.Errorf("Setting up ACME certificate with invalid value %s", a.Config.API.TLS)
	}
	certmagic.DefaultACME.Email = a.Config.API.NotificationEmail

	certmagic.Default.Logger = a.Logger.Desugar()
	certmagic.Default.Storage = &storage
	certmagic.Default.DefaultServerName = a.Config.General.Domain

	magic := certmagic.NewDefault()
	return magic
}
