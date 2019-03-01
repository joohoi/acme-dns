package main

import (
	"encoding/json"
	"net"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// ACMETxt is the default structure for the user controlled record
type ACMETxt struct {
	Username uuid.UUID
	Password string
	ACMETxtPost
	AllowFrom cidrslice
}

// ACMETxtPost holds the DNS part of the ACMETxt struct
type ACMETxtPost struct {
	Subdomain string `json:"subdomain"`
	Value     string `json:"txt"`
}

// cidrslice is a list of allowed cidr ranges
type cidrslice []string

func (c *cidrslice) JSON() string {
	ret, _ := json.Marshal(c.ValidEntries())
	return string(ret)
}

func (c *cidrslice) isValid() error {
	for _, v := range *c {
		_, _, err := net.ParseCIDR(sanitizeIPv6addr(v))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *cidrslice) ValidEntries() []string {
	valid := []string{}
	for _, v := range *c {
		_, _, err := net.ParseCIDR(sanitizeIPv6addr(v))
		if err == nil {
			valid = append(valid, sanitizeIPv6addr(v))
		}
	}
	return valid
}

// Check if IP belongs to an allowed net
func (a ACMETxt) allowedFrom(ip string) bool {
	remoteIP := net.ParseIP(ip)
	// Range not limited
	if len(a.AllowFrom.ValidEntries()) == 0 {
		return true
	}
	log.WithFields(log.Fields{"ip": remoteIP}).Debug("Checking if update is permitted from IP")
	for _, v := range a.AllowFrom.ValidEntries() {
		_, vnet, _ := net.ParseCIDR(v)
		if vnet.Contains(remoteIP) {
			return true
		}
	}
	return false
}

// Go through list (most likely from headers) to check for the IP.
// Reason for this is that some setups use reverse proxy in front of acme-dns
func (a ACMETxt) allowedFromList(ips []string) bool {
	if len(ips) == 0 {
		// If no IP provided, check if no whitelist present (everyone has access)
		return a.allowedFrom("")
	}
	for _, v := range ips {
		if a.allowedFrom(v) {
			return true
		}
	}
	return false
}

func newACMETxt() ACMETxt {
	var a = ACMETxt{}
	password := generatePassword(40)
	a.Username = uuid.New()
	a.Password = password
	a.Subdomain = uuid.New().String()
	return a
}
