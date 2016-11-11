package main

import (
	"github.com/miekg/dns"
)

// Static records
type Records struct {
	Records map[uint16]map[string][]dns.RR
}

// Config file main struct
type DnsConfig struct {
	General general
}

// Config file general section
type general struct {
	Domain             string
	Nsname             string
	Nsadmin            string
	Tls                string
	Tls_cert_privkey   string
	Tls_cert_fullchain string
	StaticRecords      []string `toml:"records"`
}
