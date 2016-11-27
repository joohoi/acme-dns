package main

import (
	"database/sql"
	"github.com/miekg/dns"
	"github.com/satori/go.uuid"
	"sync"
)

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

// ACMETxt is the default structure for the user controlled record
type ACMETxt struct {
	Username uuid.UUID
	Password string
	ACMETxtPost
	LastActive int64
}

// ACMETxtPost holds the DNS part of the ACMETxt struct
type ACMETxtPost struct {
	Subdomain string `json:"subdomain"`
	Value     string `json:"txt"`
}

type acmedb struct {
	sync.Mutex
	DB *sql.DB
}

type database interface {
	Init(string, string) error
	Register() (ACMETxt, error)
	GetByUsername(uuid.UUID) (ACMETxt, error)
	GetByDomain(string) ([]ACMETxt, error)
	Update(ACMETxt) error
	GetBackend() *sql.DB
	SetBackend(*sql.DB)
	Close()
	Lock()
	Unlock()
}
