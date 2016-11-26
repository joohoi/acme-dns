package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"github.com/op/go-logging"
	"os"
	"strings"
	"testing"
)

var testAddr = "0.0.0.0:15353"

var records = []string{
	"auth.example.org. A 192.168.1.100",
	"ns1.auth.example.org. A 192.168.1.101",
	"ns2.auth.example.org. A 192.168.1.102",
}

type resolver struct {
	server string
}

func (r *resolver) lookup(host string, qtype uint16) ([]dns.RR, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{Name: dns.Fqdn(host), Qtype: qtype, Qclass: dns.ClassINET}
	in, err := dns.Exchange(msg, r.server)
	if err != nil {
		return []dns.RR{}, fmt.Errorf("Error querying the server [%v]", err)
	}
	if in != nil && in.Rcode != dns.RcodeSuccess {
		return []dns.RR{}, fmt.Errorf("Recieved error from the server [%s]", dns.RcodeToString[in.Rcode])
	}

	return in.Answer, nil
}

func hasExpectedTXTAnswer(answer []dns.RR, cmpTXT string) error {
	for _, record := range answer {
		// We expect only one answer, so no need to loop through the answer slice
		if rec, ok := record.(*dns.TXT); ok {
			for _, txtValue := range rec.Txt {
				if txtValue == cmpTXT {
					return nil
				}
			}
		} else {
			errmsg := fmt.Sprintf("Got answer of unexpected type [%q]", answer[0])
			return errors.New(errmsg)
		}
	}
	return errors.New("Expected answer not found")
}

func findRecordFromMemory(rrstr string, host string, qtype uint16) error {
	var errmsg = "No record found"
	arr, _ := dns.NewRR(strings.ToLower(rrstr))
	if arrQt, ok := RR.Records[qtype]; ok {
		if arrHst, ok := arrQt[host]; ok {
			for _, v := range arrHst {
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

	var dbcfg = dbsettings{
		Engine:     "sqlite3",
		Connection: ":memory:",
	}

	var generalcfg = general{
		Domain:  "auth.example.org",
		Nsname:  "ns1.auth.example.org",
		Nsadmin: "admin.example.org",
		Debug:   false,
	}

	var dnscfg = DNSConfig{
		Database: dbcfg,
		General:  generalcfg,
	}

	DNSConf = dnscfg

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
	defer server.Shutdown()
	RR.Parse(records)
	answer, err := resolver.lookup("auth.example.org", dns.TypeA)
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(answer) > 0 {
		err = findRecordFromMemory(answer[0].String(), "auth.example.org.", dns.TypeA)
		if err != nil {
			t.Errorf("Answer [%s] did not match the expected, got error: [%s], debug: [%q]", answer[0].String(), err, RR.Records)
		}

	} else {
		t.Error("No answer for DNS query")
	}
}

func TestResolveTXT(t *testing.T) {
	flag.Parse()
	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			t.Errorf("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			return
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}
	defer DB.DB.Close()

	server, resolver := startDNSServer(testAddr)
	defer server.Shutdown()
	RR.Parse(records)

	validTXT := "______________valid_response_______________"

	atxt, err := DB.Register()
	if err != nil {
		t.Errorf("Could not initiate db record: [%v]", err)
		return
	}
	atxt.Value = validTXT
	err = DB.Update(atxt)
	if err != nil {
		t.Errorf("Could not update db record: [%v]", err)
		return
	}

	for i, test := range []struct {
		subDomain   string
		expTXT      string
		getAnswer   bool
		validAnswer bool
	}{
		{atxt.Subdomain, validTXT, true, true},
		{atxt.Subdomain, "invalid", true, false},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", validTXT, false, false},
	} {
		answer, err := resolver.lookup(test.subDomain+".auth.example.org", dns.TypeTXT)
		if err != nil {
			if test.getAnswer {
				t.Errorf("Test %d: Expected answer but got: %v", i, err)
			}
		} else {
			if !test.getAnswer {
				t.Errorf("Test %d: Expected no answer, but got one.", i)
			}
		}

		if len(answer) > 0 {
			if !test.getAnswer {
				t.Errorf("Test %d: Expected no answer, but got: [%q]", i, answer)
			}
			err = hasExpectedTXTAnswer(answer, test.expTXT)
			if err != nil {
				if test.validAnswer {
					t.Errorf("Test %d: %v", i, err)
				}
			} else {
				if !test.validAnswer {
					t.Errorf("Test %d: Answer was not expected to be valid, answer [%q], compared to [%s]", i, answer, test.expTXT)
				}
			}
		} else {
			if test.getAnswer {
				t.Errorf("Test %d: Expected answer, but didn't get one", i)
			}
		}
	}
}
