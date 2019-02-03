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
	DB      database
	Server  *dns.Server
	SOA     dns.RR
	Domains map[string]Records
}

// NewDNSServer parses the DNS records from config and returns a new DNSServer struct
func NewDNSServer(db database, addr string, proto string) *DNSServer {
	var server DNSServer
	server.Server = &dns.Server{Addr: addr, Net: proto}
	server.DB = db
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

	if r.Opcode == dns.OpcodeQuery {
		d.readQuery(m)
	} else if r.Opcode == dns.OpcodeUpdate {
		log.Debug("Refusing DNS Dynamic update request")
		m.MsgHdr.Rcode = dns.RcodeRefused
	}

	w.WriteMsg(m)
}

func (d *DNSServer) readQuery(m *dns.Msg) {
	for _, que := range m.Question {
		if rr, rc, err := d.answer(que); err == nil {
			m.MsgHdr.Rcode = rc
			for _, r := range rr {
				m.Answer = append(m.Answer, r)
			}
		}
	}
}

func (d *DNSServer) getRecord(q dns.Question) ([]dns.RR, error) {
	var rr []dns.RR
	var cnames []dns.RR
	domain, ok := d.Domains[q.Name]
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
func (d *DNSServer) answeringForDomain(q dns.Question) bool {
	_, ok := d.Domains[q.Name]
	return ok
}

func (d *DNSServer) answer(q dns.Question) ([]dns.RR, int, error) {
	var rcode int
	if !d.answeringForDomain(q) {
		rcode = dns.RcodeNameError
	}
	r, _ := d.getRecord(q)
	if q.Qtype == dns.TypeTXT {
		txtRRs, err := d.answerTXT(q)
		if err == nil {
			for _, txtRR := range txtRRs {
				r = append(r, txtRR)
			}
		}
	}
	if len(r) > 0 {
		// Make sure that we return NOERROR if there were dynamic records for the domain
		rcode = dns.RcodeSuccess
	}
	// Handle EDNS (no support at the moment)
	if q.Qtype == dns.TypeOPT {
		return []dns.RR{}, dns.RcodeFormatError, nil
	}
	log.WithFields(log.Fields{"qtype": dns.TypeToString[q.Qtype], "domain": q.Name, "rcode": dns.RcodeToString[rcode]}).Debug("Answering question for domain")
	return r, rcode, nil
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
