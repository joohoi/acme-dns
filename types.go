package main

import (
	"database/sql"
	"github.com/miekg/dns"
	"github.com/satori/go.uuid"
	"sync"
)

// Config is global configuration struct
var Config DNSConfig

// DB is used to access the database functions in acme-dns
var DB database

// RR holds the static DNS records
var RR Records

// Records is for static records
type Records struct {
	Records map[uint16]map[string][]dns.RR
}

// DNSConfig holds the config structure
type DNSConfig struct {
	General   general
	Database  dbsettings
	API       httpapi
	Logconfig logconfig
}

// Auth middleware
type authMiddleware struct{}

// Config file general section
type general struct {
	Listen        string
	Proto         string `toml:"protocol"`
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
	Domain           string `toml:"api_domain"`
	LEmail           string `toml:"le_email"`
	IP               string
	Port             string
	TLS              string
	TLSCertPrivkey   string `toml:"tls_cert_privkey"`
	TLSCertFullchain string `toml:"tls_cert_fullchain"`
	CorsOrigins      []string
	UseHeader        bool   `toml:"use_header"`
	HeaderName       string `toml:"header_name"`
}

// Logging config
type logconfig struct {
	Level   string `toml:"loglevel"`
	Logtype string `toml:"logtype"`
	File    string `toml:"logfile"`
	Format  string `toml:"logformat"`
}

type acmedb struct {
	sync.Mutex
	DB *sql.DB
}

type database interface {
	Init(string, string) error
	Register(cidrslice) (ACMETxt, error)
	GetByUsername(uuid.UUID) (ACMETxt, error)
	GetByDomain(string) ([]ACMETxt, error)
	Update(ACMETxt) error
	GetBackend() *sql.DB
	SetBackend(*sql.DB)
	Close()
	Lock()
	Unlock()
}
