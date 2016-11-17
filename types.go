package main

import (
	"github.com/miekg/dns"
	"github.com/satori/go.uuid"
)

// Static records
type Records struct {
	Records map[uint16]map[string][]dns.RR
}

// Config file main struct
type DNSConfig struct {
	General   general
	Database  dbsettings
	API       httpapi
	Logconfig logconfig
}

// Auth middleware
type AuthMiddleware struct{}

// Config file general section
type general struct {
	Domain        string
	Nsname        string
	Nsadmin       string
	Debug         bool
	StaticRecords []string `toml:"records"`
}

type dbsettings struct {
	Engine     string
	Connection string
}

// API config
type httpapi struct {
	Domain           string
	Port             string
	TLS              string
	TLSCertPrivkey   string `toml:"tls_cert_privkey"`
	TLSCertFullchain string `toml:"tls_cert_fullchain"`
	CorsOrigins      []string
}

// Logging config
type logconfig struct {
	Level   string `toml:"loglevel"`
	Logtype string `toml:"logtype"`
	File    string `toml:"logfile"`
	Format  string `toml:"logformat"`
}

// The default object
type ACMETxt struct {
	Username uuid.UUID
	Password string
	ACMETxtPost
	LastActive int64
}

type ACMETxtPost struct {
	Subdomain string `json:"subdomain"`
	Value     string `json:"txt"`
}
