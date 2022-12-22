package acmedns

import (
	"github.com/google/uuid"
	"net"
)

// Check if IP belongs to an allowed net
func (a ACMETxt) AllowedFrom(ip string) bool {
	remoteIP := net.ParseIP(ip)
	// Range not limited
	if len(a.AllowFrom.ValidEntries()) == 0 {
		return true
	}
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
func (a ACMETxt) AllowedFromList(ips []string) bool {
	if len(ips) == 0 {
		// If no IP provided, check if no whitelist present (everyone has access)
		return a.AllowedFrom("")
	}
	for _, v := range ips {
		if a.AllowedFrom(v) {
			return true
		}
	}
	return false
}

func NewACMETxt() ACMETxt {
	var a = ACMETxt{}
	password := generatePassword(40)
	a.Username = uuid.New()
	a.Password = password
	a.Subdomain = uuid.New().String()
	return a
}
