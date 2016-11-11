package main

import (
	"fmt"
	"github.com/miekg/dns"
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
	var rcode int = dns.RcodeNameError
	var domain string = q.Name

	atxt, err := DB.GetByDomain(domain)
	if err != nil {
		log.Errorf("Error while trying to get record [%v]", err)
		return ra, dns.RcodeNameError, err
	}
	for _, v := range atxt {
		if len(v.Value) > 0 {
			r := new(dns.TXT)
			r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 1}
			r.Txt = append(r.Txt, v.Value)
			ra = append(ra, r)
			rcode = dns.RcodeSuccess
		}
	}
	log.Debugf("Answering TXT question for domain [%s]", domain)
	return ra, rcode, nil
}

func answer(q dns.Question) ([]dns.RR, int, error) {
	if q.Qtype == dns.TypeTXT {
		return answerTXT(q)
	}
	var r []dns.RR
	var rcode int = dns.RcodeSuccess
	var domain string = q.Name
	var rtype uint16 = q.Qtype
	r, ok := RR.Records[rtype][domain]
	if !ok {
		rcode = dns.RcodeNameError
	}
	log.Debugf("Answering [%s] question for domain [%s] with rcode [%s]", dns.TypeToString[rtype], domain, dns.RcodeToString[rcode])
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
func (r *Records) Parse(recs []string) {
	rrmap := make(map[uint16]map[string][]dns.RR)
	for _, v := range recs {
		rr, err := dns.NewRR(v)
		if err != nil {
			log.Errorf("Could not parse RR from config: [%v] for RR: [%s]", err, v)
			continue
		}
		// Add parsed RR to the list
		rrmap = AppendRR(rrmap, rr)
	}
	// Create serial
	serial := time.Now().Format("2006010215")
	// Add SOA
	SOAstring := fmt.Sprintf("%s. SOA %s. %s. %s 28800 7200 604800 86400", DnsConf.General.Domain, DnsConf.General.Nsname, DnsConf.General.Nsadmin, serial)
	soarr, err := dns.NewRR(SOAstring)
	if err != nil {
		log.Errorf("Error [%v] while trying to add SOA record: [%s]", err, SOAstring)
	} else {
		rrmap = AppendRR(rrmap, soarr)
	}
	r.Records = rrmap
}

func AppendRR(rrmap map[uint16]map[string][]dns.RR, rr dns.RR) map[uint16]map[string][]dns.RR {
	_, ok := rrmap[rr.Header().Rrtype]
	if !ok {
		newrr := make(map[string][]dns.RR)
		rrmap[rr.Header().Rrtype] = newrr
	}
	rrmap[rr.Header().Rrtype][rr.Header().Name] = append(rrmap[rr.Header().Rrtype][rr.Header().Name], rr)
	log.Debugf("Adding new record of type [%s] for domain [%s]", dns.TypeToString[rr.Header().Rrtype], rr.Header().Name)
	return rrmap
}
