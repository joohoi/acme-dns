package main

import (
	"errors"
	"fmt"
	"github.com/miekg/dns"
	"github.com/op/go-logging"
	"os"
	"strings"
	"testing"
)

var testAddr = ":15353"

var records = []string{
	"auth.example.org. A 192.168.1.100",
	"ns1.auth.example.org. A 192.168.1.101",
	"ns2.auth.example.org. A 192.168.1.102",
}

type resolver struct {
	server string
}

func (r *resolver) lookup(host string, qtype uint16) (string, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{dns.Fqdn(host), qtype, dns.ClassINET}
	in, err := dns.Exchange(msg, r.server)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error querying the server [%v]", err))
	}
	if in != nil && in.Rcode != dns.RcodeSuccess {
		return "", errors.New(fmt.Sprintf("Recieved error from the server [%s]", dns.RcodeToString[in.Rcode]))
	}

	if len(in.Answer) > 0 {
		return in.Answer[0].String(), nil
	}
	return "", errors.New("No answer")
}

func findRecord(rrstr string, host string, qtype uint16) error {
	var errmsg = "No record found"
	arr, _ := dns.NewRR(strings.ToLower(rrstr))
	if arr_qt, ok := RR.Records[qtype]; ok {
		if arr_hst, ok := arr_qt[host]; ok {
			for _, v := range arr_hst {
				if arr.String() == v.String() {
					return nil
				}
			}
		} else {
			errmsg = "No records for domain"
		}
	} else {
		errmsg = "No records for this type in DB"
	}
	return errors.New(errmsg)
}

func startDNSServer(addr string) (*dns.Server, resolver) {
	logging.InitForTesting(logging.DEBUG)
	// DNS server part
	dns.HandleFunc(".", handleRequest)
	server := &dns.Server{Addr: addr, Net: "udp"}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Errorf("%v", err)
			os.Exit(1)
		}
	}()
	return server, resolver{server: addr}
}

func TestResolveA(t *testing.T) {
	server, resolver := startDNSServer(testAddr)
	RR.Parse(records)
	a, err := resolver.lookup("auth.example.org", dns.TypeA)
	if err != nil {
		t.Errorf("%v", err)
	}
	err = findRecord(a, "auth.example.org.", dns.TypeA)
	if err != nil {
		t.Errorf("Answer [%s] did not match the expected, got error: [%s], debug: [%q]", a, err, RR.Records)
	}
	server.Shutdown()
}
