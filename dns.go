package main

import (
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func readQuery(m *dns.Msg) {
	for _, que := range m.Question {
		if rr, rc, err := answer(que); err == nil {
			m.MsgHdr.Rcode = rc
			for _, r := range rr {
				m.Answer = append(m.Answer, r)
			}
		}
	}
}

func answerTXT(q dns.Question) ([]dns.RR, int, error) {
	var ra []dns.RR
	rcode := dns.RcodeNameError
	subdomain := sanitizeDomainQuestion(q.Name)
	atxt, err := DB.GetTXTForDomain(subdomain)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Debug("Error while trying to get record")
		return ra, dns.RcodeNameError, err
	}
	for _, v := range atxt {
		if len(v) > 0 {
			r := new(dns.TXT)
			r.Hdr = dns.RR_Header{Name: q.Name, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
			r.Txt = append(r.Txt, v)
			ra = append(ra, r)
			rcode = dns.RcodeSuccess
		}
	}

	log.WithFields(log.Fields{"domain": q.Name}).Info("Answering TXT question for domain")
	return ra, rcode, nil
}

func answer(q dns.Question) ([]dns.RR, int, error) {
	if q.Qtype == dns.TypeTXT {
		return answerTXT(q)
	}
	var r []dns.RR
	var rcode = dns.RcodeSuccess
	var domain = strings.ToLower(q.Name)
	var rtype = q.Qtype
	r, ok := RR.Records[rtype][domain]
	if !ok {
		rcode = dns.RcodeNameError
	}
	log.WithFields(log.Fields{"qtype": dns.TypeToString[rtype], "domain": domain, "rcode": dns.RcodeToString[rcode]}).Debug("Answering question for domain")
	return r, rcode, nil
}

func handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	if r.Opcode == dns.OpcodeQuery {
		readQuery(m)
	}

	w.WriteMsg(m)
}

// Parse config records
func (r *Records) Parse(config general) {
	rrmap := make(map[uint16]map[string][]dns.RR)
	for _, v := range config.StaticRecords {
		rr, err := dns.NewRR(strings.ToLower(v))
		if err != nil {
			log.WithFields(log.Fields{"error": err.Error(), "rr": v}).Warning("Could not parse RR from config")
			continue
		}
		// Add parsed RR to the list
		rrmap = appendRR(rrmap, rr)
	}
	// Create serial
	serial := time.Now().Format("2006010215")
	// Add SOA
	SOAstring := fmt.Sprintf("%s. SOA %s. %s. %s 28800 7200 604800 86400", strings.ToLower(config.Domain), strings.ToLower(config.Nsname), strings.ToLower(config.Nsadmin), serial)
	soarr, err := dns.NewRR(SOAstring)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error(), "soa": SOAstring}).Error("Error while adding SOA record")
	} else {
		rrmap = appendRR(rrmap, soarr)
	}
	r.Records = rrmap
}

func appendRR(rrmap map[uint16]map[string][]dns.RR, rr dns.RR) map[uint16]map[string][]dns.RR {
	_, ok := rrmap[rr.Header().Rrtype]
	if !ok {
		newrr := make(map[string][]dns.RR)
		rrmap[rr.Header().Rrtype] = newrr
	}
	rrmap[rr.Header().Rrtype][rr.Header().Name] = append(rrmap[rr.Header().Rrtype][rr.Header().Name], rr)
	log.WithFields(log.Fields{"recordtype": dns.TypeToString[rr.Header().Rrtype], "domain": rr.Header().Name}).Debug("Adding new record type to domain")
	return rrmap
}
