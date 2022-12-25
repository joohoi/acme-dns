package nameserver

import (
	"fmt"
	"github.com/miekg/dns"
	"strings"
)

func (n *Nameserver) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
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
		if rr, rc, auth, err := n.answer(que); err == nil {
			if auth {
				authoritative = auth
			}
			m.MsgHdr.Rcode = rc
			m.Answer = append(m.Answer, rr...)
		}
	}
	m.MsgHdr.Authoritative = authoritative
	if authoritative {
		if m.MsgHdr.Rcode == dns.RcodeNameError {
			m.Ns = append(m.Ns, n.SOA)
		}
	}
}

func (n *Nameserver) answer(q dns.Question) ([]dns.RR, int, bool, error) {
	var rcode int
	var err error
	var txtRRs []dns.RR
	var authoritative = n.isAuthoritative(q)
	if !n.isOwnChallenge(q.Name) && !n.answeringForDomain(q.Name) {
		rcode = dns.RcodeNameError
	}
	r, _ := n.getRecord(q)
	if q.Qtype == dns.TypeTXT {
		if n.isOwnChallenge(q.Name) {
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

func (n *Nameserver) isAuthoritative(q dns.Question) bool {
	if n.answeringForDomain(q.Name) {
		return true
	}
	domainParts := strings.Split(strings.ToLower(q.Name), ".")
	for i := range domainParts {
		if n.answeringForDomain(strings.Join(domainParts[i:], ".")) {
			return true
		}
	}
	return false
}

// isOwnChallenge checks if the query is for the domain of this acme-dns instance. Used for answering its own ACME challenges
func (n *Nameserver) isOwnChallenge(name string) bool {
	domainParts := strings.SplitN(name, ".", 2)
	if len(domainParts) == 2 {
		if strings.ToLower(domainParts[0]) == "_acme-challenge" {
			domain := strings.ToLower(domainParts[1])
			if !strings.HasSuffix(domain, ".") {
				domain = domain + "."
			}
			if domain == n.OwnDomain {
				return true
			}
		}
	}
	return false
}

// answeringForDomain checks if we have any records for a domain
func (n *Nameserver) answeringForDomain(name string) bool {
	if n.OwnDomain == strings.ToLower(name) {
		return true
	}
	_, ok := n.Domains[strings.ToLower(name)]
	return ok
}

func (n *Nameserver) getRecord(q dns.Question) ([]dns.RR, error) {
	var rr []dns.RR
	var cnames []dns.RR
	domain, ok := n.Domains[strings.ToLower(q.Name)]
	if !ok {
		return rr, fmt.Errorf("no records for domain %s", q.Name)
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
