package nameserver

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

func (n *Nameserver) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	// handle edns0
	opt := r.IsEdns0()
	if opt != nil {
		if opt.Version() != 0 {
			// Only EDNS0 is standardized
			m.Rcode = dns.RcodeBadVers
			m.SetEdns0(512, false)
		} else {
			// We can safely do this as we know that we're not setting other OPT RRs within acme-dns.
			m.SetEdns0(512, false)
			if r.Opcode == dns.OpcodeQuery {
				n.readQuery(m)
			}
		}
	} else {
		if r.Opcode == dns.OpcodeQuery {
			n.readQuery(m)
		}
	}
	_ = w.WriteMsg(m)
}

func (n *Nameserver) readQuery(m *dns.Msg) {
	var authoritative = false
	for _, que := range m.Question {
		rr, rc, auth, err := n.answer(que)
		if auth {
			authoritative = true
		}
		if err != nil {
			m.Rcode = dns.RcodeServerFailure
			m.Answer = nil
			break
		}
		m.Rcode = rc
		m.Answer = append(m.Answer, rr...)
	}
	m.Authoritative = authoritative
	if authoritative && (m.Rcode == dns.RcodeNameError ||
		(m.Rcode == dns.RcodeSuccess && len(m.Answer) == 0)) {
		m.Ns = append(m.Ns, n.SOA)
	}
}

func (n *Nameserver) answer(q dns.Question) ([]dns.RR, int, bool, error) {
	var rcode int
	var err error
	var txtRRs []dns.RR
	loweredName := strings.ToLower(q.Name)
	var authoritative = n.isAuthoritative(loweredName)
	nameExists := n.isOwnChallenge(loweredName) || n.answeringForDomain(loweredName)
	if authoritative && !nameExists && q.Qtype != dns.TypeTXT {
		nameExists, err = n.hasDynamicTXTRecord(loweredName)
		if err != nil {
			return nil, dns.RcodeServerFailure, authoritative, err
		}
	}
	if !nameExists {
		rcode = dns.RcodeNameError
	}
	r, _ := n.getRecord(loweredName, q.Qtype)
	if q.Qtype == dns.TypeTXT {
		if n.isOwnChallenge(loweredName) {
			txtRRs, err = n.answerOwnChallenge(q)
		} else {
			txtRRs, err = n.answerTXT(q)
		}
		if err == nil {
			r = append(r, txtRRs...)
		}
	}
	if len(r) > 0 {
		// Make sure that we return NOERROR if there were dynamic records for the domain
		rcode = dns.RcodeSuccess
	}
	n.Logger.Debugw("Answering question for domain",
		"qtype", dns.TypeToString[q.Qtype],
		"domain", q.Name,
		"rcode", dns.RcodeToString[rcode])
	return r, rcode, authoritative, nil
}

func (n *Nameserver) hasDynamicTXTRecord(name string) (bool, error) {
	subdomain, ok := strings.CutSuffix(name, "."+n.OwnDomain)
	if !ok || subdomain == "" || strings.Contains(subdomain, ".") {
		return false, nil
	}

	txts, err := n.DB.GetTXTForDomain(subdomain)
	if err != nil {
		n.Logger.Errorw("Error while trying to get record",
			"error", err.Error())
		return false, err
	}
	for _, txt := range txts {
		if len(txt) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (n *Nameserver) answerTXT(q dns.Question) ([]dns.RR, error) {
	var ra []dns.RR
	subdomain := sanitizeDomainQuestion(q.Name)
	atxt, err := n.DB.GetTXTForDomain(subdomain)
	if err != nil {
		n.Logger.Errorw("Error while trying to get record",
			"error", err.Error())
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

func (n *Nameserver) isAuthoritative(name string) bool {
	if n.answeringForDomain(name) {
		return true
	}
	off := 0
	for {
		i, next := dns.NextLabel(name, off)
		if next {
			return false
		}
		off = i
		if n.answeringForDomain(name[off:]) {
			return true
		}
	}
}

func (n *Nameserver) isOwnChallenge(name string) bool {
	if strings.HasPrefix(name, "_acme-challenge.") {
		domain := name[16:]
		if domain == n.OwnDomain {
			return true
		}
	}
	return false
}

// answeringForDomain checks if we have any records for a domain
func (n *Nameserver) answeringForDomain(name string) bool {
	if n.OwnDomain == name {
		return true
	}
	_, ok := n.Domains[name]
	return ok
}

func (n *Nameserver) getRecord(name string, qtype uint16) ([]dns.RR, error) {
	var rr []dns.RR
	var cnames []dns.RR
	domain, ok := n.Domains[name]
	if !ok {
		return rr, fmt.Errorf("no records for domain %s", name)
	}
	for _, ri := range domain.Records {
		if ri.Header().Rrtype == qtype {
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
