package main

import (
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

// Records is a slice of ResourceRecords
type Records struct {
	Records []dns.RR
}

// DNSServer is the main struct for acme-dns DNS server
type DNSServer struct {
	DB              database
	Domain          string
	Server          *dns.Server
	SOA             dns.RR
	PersonalKeyAuth string
	Domains         map[string]Records
}

// NewDNSServer parses the DNS records from config and returns a new DNSServer struct
func NewDNSServer(db database, addr string, proto string, domain string) *DNSServer {
	var server DNSServer
	server.Server = &dns.Server{Addr: addr, Net: proto}
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}
	server.Domain = strings.ToLower(domain)
	server.DB = db
	server.PersonalKeyAuth = ""
	server.Domains = make(map[string]Records)
	return &server
}

// Start starts the DNSServer
func (d *DNSServer) Start(errorChannel chan error) {
	// DNS server part
	dns.HandleFunc(".", d.handleRequest)
	log.WithFields(log.Fields{"addr": d.Server.Addr, "proto": d.Server.Net}).Info("Listening DNS")
	err := d.Server.ListenAndServe()
	if err != nil {
		errorChannel <- err
	}
}

// ParseRecords parses a slice of DNS record string
func (d *DNSServer) ParseRecords(config DNSConfig) {
	for _, v := range config.General.StaticRecords {
		rr, err := dns.NewRR(strings.ToLower(v))
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error(), "rr": v}).Warning("Could not parse RR from config")
			continue
		}
		// Add parsed RR
		d.appendRR(rr)
	}
	// Create serial
	serial := time.Now().Format("2006010215")
	// Add SOA
	SOAstring := fmt.Sprintf("%s. SOA %s. %s. %s 28800 7200 604800 86400", strings.ToLower(config.General.Domain), strings.ToLower(config.General.Nsname), strings.ToLower(config.General.Nsadmin), serial)
	soarr, err := dns.NewRR(SOAstring)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "soa": SOAstring}).Error("Error while adding SOA record")
	} else {
		d.appendRR(soarr)
		d.SOA = soarr
	}
}

func (d *DNSServer) appendRR(rr dns.RR) {
	addDomain := rr.Header().Name
	_, ok := d.Domains[addDomain]
	if !ok {
		d.Domains[addDomain] = Records{[]dns.RR{rr}}
	} else {
		drecs := d.Domains[addDomain]
		drecs.Records = append(drecs.Records, rr)
		d.Domains[addDomain] = drecs
	}
	log.WithFields(log.Fields{"recordtype": dns.TypeToString[rr.Header().Rrtype], "domain": addDomain}).Debug("Adding new record to domain")
}

func (d *DNSServer) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	// handle edns0
	opt := r.IsEdns0()
	if opt != nil {
		if opt.Version() != 0 {
			// Only EDNS0 is standardized
			m.MsgHdr.Rcode = dns.RcodeBadVers
			m.SetEdns0(512, false)
		} else {
			// We can safely do this as we know that we're not setting other OPT RRs within acme-dns.
			m.SetEdns0(512, false)
			if r.Opcode == dns.OpcodeQuery {
				d.readQuery(m)
			}
		}
	} else {
		if r.Opcode == dns.OpcodeQuery {
			d.readQuery(m)
		}
	}
	_ = w.WriteMsg(m)
}

func (d *DNSServer) readQuery(m *dns.Msg) {
	var authoritative = false
	for _, que := range m.Question {
		if rr, rc, auth, err := d.answer(que); err == nil {
			if auth {
				authoritative = auth
			}
			m.MsgHdr.Rcode = rc
			m.Answer = append(m.Answer, rr...)
		}
	}
	m.MsgHdr.Authoritative = authoritative
	if authoritative {
		if m.MsgHdr.Rcode == dns.RcodeNameError ||
		   (m.MsgHdr.Rcode == dns.RcodeSuccess && len(m.Answer) == 0) {
			m.Ns = append(m.Ns, d.SOA)
		}
	}
}

func (d *DNSServer) getRecord(q dns.Question) ([]dns.RR, error) {
	var rr []dns.RR
	var cnames []dns.RR
	domain, ok := d.Domains[strings.ToLower(q.Name)]
	if !ok {
		return rr, fmt.Errorf("No records for domain %s", q.Name)
	}
	for _, ri := range domain.Records {
		if ri.Header().Rrtype == q.Qtype {
			rr = append(rr, ri)
		}
		if ri.Header().Rrtype == dns.TypeCNAME {
			cnames = append(cnames, ri)
		}
	}
	if len(rr) == 0 {
		return cnames, nil
	}
	return rr, nil
}

// answeringForDomain checks if we have any records for a domain
func (d *DNSServer) answeringForDomain(name string) bool {
	if d.Domain == strings.ToLower(name) {
		return true
	}
	_, ok := d.Domains[strings.ToLower(name)]
	return ok
}
// hasTxtForDomain checks if we have txt records for a domain
func (d *DNSServer) hasTxtForDomain(q dns.Question) bool {
	subdomain := sanitizeDomainQuestion(q.Name)
	txts, err := d.DB.GetTXTForDomain(subdomain)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to get record")
		return false
	}
	return len(txts) > 0
}


func (d *DNSServer) isAuthoritative(q dns.Question) bool {
	if d.answeringForDomain(q.Name) {
		return true
	}
	domainParts := strings.Split(strings.ToLower(q.Name), ".")
	for i := range domainParts {
		if d.answeringForDomain(strings.Join(domainParts[i:], ".")) {
			return true
		}
	}
	return false
}

// isOwnChallenge checks if the query is for the domain of this acme-dns instance. Used for answering its own ACME challenges
func (d *DNSServer) isOwnChallenge(name string) bool {
	domainParts := strings.SplitN(name, ".", 2)
	if len(domainParts) == 2 {
		if strings.ToLower(domainParts[0]) == "_acme-challenge" {
			domain := strings.ToLower(domainParts[1])
			if !strings.HasSuffix(domain, ".") {
				domain = domain + "."
			}
			if domain == d.Domain {
				return true
			}
		}
	}
	return false
}

func (d *DNSServer) answer(q dns.Question) ([]dns.RR, int, bool, error) {
	var rcode int
	var err error
	var txtRRs []dns.RR
	var authoritative = d.isAuthoritative(q)
	if !d.isOwnChallenge(q.Name) && !d.answeringForDomain(q.Name) && !d.hasTxtForDomain(q) {
		rcode = dns.RcodeNameError
	}
	r, _ := d.getRecord(q)
	if q.Qtype == dns.TypeTXT {
		if d.isOwnChallenge(q.Name) {
			txtRRs, err = d.answerOwnChallenge(q)
		} else {
			txtRRs, err = d.answerTXT(q)
		}
		if err == nil {
			r = append(r, txtRRs...)
		}
	}
	if len(r) > 0 {
		// Make sure that we return NOERROR if there were dynamic records for the domain
		rcode = dns.RcodeSuccess
	}
	log.WithFields(log.Fields{"qtype": dns.TypeToString[q.Qtype], "domain": q.Name, "rcode": dns.RcodeToString[rcode]}).Debug("Answering question for domain")
	return r, rcode, authoritative, nil
}

func (d *DNSServer) answerTXT(q dns.Question) ([]dns.RR, error) {
	var ra []dns.RR
	subdomain := sanitizeDomainQuestion(q.Name)
	atxt, err := d.DB.GetTXTForDomain(subdomain)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to get record")
		return ra, err
	}
	for _, v := range atxt {
		if len(v) > 0 {
			r := new(dns.TXT)
			r.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
			r.Txt = append(r.Txt, v)
			ra = append(ra, r)
		}
	}
	return ra, nil
}

// answerOwnChallenge answers to ACME challenge for acme-dns own certificate
func (d *DNSServer) answerOwnChallenge(q dns.Question) ([]dns.RR, error) {
	r := new(dns.TXT)
	r.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
	r.Txt = append(r.Txt, d.PersonalKeyAuth)
	return []dns.RR{r}, nil
}
