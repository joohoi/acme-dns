package main

import (
	"github.com/miekg/dns"
	"github.com/satori/go.uuid"
	"time"
)

// Static records
type Records struct {
	Records map[uint16]map[string][]dns.RR
}

// Config file main struct
type DnsConfig struct {
	General   general
	Api       httpapi
	Logconfig logconfig
}

// Auth middleware
type AuthMiddleware struct{}

// Config file general section
type general struct {
	Domain        string
	Nsname        string
	Nsadmin       string
	StaticRecords []string `toml:"records"`
}

// API config
type httpapi struct {
	Domain             string
	Port               string
	Tls                string
	Tls_cert_privkey   string
	Tls_cert_fullchain string
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
	LastActive time.Time
}

type ACMETxtPost struct {
	Subdomain string `json:"subdomain"`
	Value     string `json:"txt"`
}
