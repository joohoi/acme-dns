package nameserver

import "github.com/miekg/dns"

// SetOwnAuthKey sets the ACME challenge token for completing dns validation for acme-dns server itself
func (n *Nameserver) SetOwnAuthKey(key string) {
	n.personalAuthKey = key
}

// answerOwnChallenge answers to ACME challenge for acme-dns own certificate
func (n *Nameserver) answerOwnChallenge(q dns.Question) ([]dns.RR, error) {
	r := new(dns.TXT)
	r.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
	r.Txt = append(r.Txt, n.personalAuthKey)
	return []dns.RR{r}, nil
}
