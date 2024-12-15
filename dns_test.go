package main

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"testing"

	"github.com/erikstmartin/go-testdb"
	"github.com/miekg/dns"
)

type resolver struct {
	server string
}

type testRecord struct {
	subDomain   string
	expTXT      []string
	getAnswer   bool
	validAnswer bool
}

func (r *resolver) lookup(host string, qtype uint16) (*dns.Msg, error) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{Name: dns.Fqdn(host), Qtype: qtype, Qclass: dns.ClassINET}
	in, err := dns.Exchange(msg, r.server)
	if err != nil {
		return in, fmt.Errorf("Error querying the server [%v]", err)
	}
	if in != nil && in.Rcode != dns.RcodeSuccess {
		return in, fmt.Errorf("Received error from the server [%s]", dns.RcodeToString[in.Rcode])
	}

	return in, nil
}

func hasExpectedTXTAnswer(answer []dns.RR, cmpTXT []string) error {
	matches := 0
	txts := 0

OUTER:
	for _, record := range answer {
		// Verify all expected answers are returned
		if rec, ok := record.(*dns.TXT); ok {
			for _, txtValue := range rec.Txt {
				txts++
				for _, cmpValue := range cmpTXT {
					if txtValue == cmpValue {
						matches++
						continue OUTER
					}
				}
			}
		} else {
			errmsg := fmt.Sprintf("Got answer of unexpected type [%q]", answer[0])
			return errors.New(errmsg)
		}
	}

	//Got too many results
	if txts > len(cmpTXT) {
		errmsg := fmt.Sprintf("Got too many answers [%d > %d]", txts, len(cmpTXT))
		return errors.New(errmsg)
	} else if txts < len(cmpTXT) {
		//Got too few results
		errmsg := fmt.Sprintf("Got too few answers [%d < %d]", txts, len(cmpTXT))
		return errors.New(errmsg)
	} else if matches > len(cmpTXT) {
		//Got too many matches
		errmsg := fmt.Sprintf("Got too many matches [%d > %d]", matches, len(cmpTXT))
		return errors.New(errmsg)
	} else if matches < len(cmpTXT) {
		//Got not enough matches
		errmsg := fmt.Sprintf("Got too few matches [%d < %d]", matches, len(cmpTXT))
		return errors.New(errmsg)
	} else if matches == len(cmpTXT) {
		//If they all matched we are ok
		return nil
	}

	return errors.New("Expected answer(s) not found")
}

func hasExpectedResolveTXTs(tests []testRecord) error {
	resolv := resolver{server: "127.0.0.1:15353"}

	for i, test := range tests {
		answer, err := resolv.lookup(test.subDomain+".auth.example.org", dns.TypeTXT)
		if err != nil {
			if test.getAnswer {
				return fmt.Errorf("%d: Expected answer but got: %v", i, err)
			}
		} else {
			if !test.getAnswer {
				return fmt.Errorf("%d: Expected no answer, but got one", i)
			}
		}

		if len(answer.Answer) > 0 {
			if !test.getAnswer && answer.Answer[0].Header().Rrtype != dns.TypeSOA {
				return fmt.Errorf("%d: Expected no answer, but got: [%q]", i, answer)
			}
			if test.getAnswer {
				err = hasExpectedTXTAnswer(answer.Answer, test.expTXT)
				if err != nil {
					if test.validAnswer {
						return fmt.Errorf("%d: %v", i, err)
					}
				} else {
					if !test.validAnswer {
						return fmt.Errorf("%d: Answer was not expected to be valid, answer [%q], compared to [%s]", i, answer, test.expTXT)
					}
				}
			}
		} else {
			if test.getAnswer {
				return fmt.Errorf("%d: Expected answer, but didn't get one", i)
			}
		}
	}

	return nil
}

func TestQuestionDBError(t *testing.T) {
	testdb.SetQueryWithArgsFunc(func(query string, args []driver.Value) (result driver.Rows, err error) {
		columns := []string{"Username", "Password", "Subdomain", "Value", "LastActive"}
		return testdb.RowsFromSlice(columns, [][]driver.Value{}), errors.New("Prepared query error")
	})

	defer testdb.Reset()

	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()

	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)

	q := dns.Question{Name: dns.Fqdn("whatever.tld"), Qtype: dns.TypeTXT, Qclass: dns.ClassINET}
	_, err = dnsserver.answerTXT(q)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestParse(t *testing.T) {
	var testcfg = DNSConfig{
		General: general{
			Domain:        ")",
			Nsname:        "ns1.auth.example.org",
			Nsadmin:       "admin.example.org",
			StaticRecords: []string{},
			Debug:         false,
		},
	}
	dnsserver.ParseRecords(testcfg)
	if !loggerHasEntryWithMessage("Error while adding SOA record") {
		t.Errorf("Expected SOA parsing to return error, but did not find one")
	}
}

func TestResolveA(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	answer, err := resolv.lookup("auth.example.org", dns.TypeA)
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(answer.Answer) == 0 {
		t.Error("No answer for DNS query")
	}

	_, err = resolv.lookup("nonexistent.domain.tld", dns.TypeA)
	if err == nil {
		t.Errorf("Was expecting error because of NXDOMAIN but got none")
	}
}

func TestEDNS(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	answer, _ := resolv.lookup("auth.example.org", dns.TypeOPT)
	if answer.Rcode != dns.RcodeSuccess {
		t.Errorf("Was expecing NOERROR rcode for OPT query, but got [%s] instead.", dns.RcodeToString[answer.Rcode])
	}
}

func TestEDNSA(t *testing.T) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{Name: dns.Fqdn("auth.example.org"), Qtype: dns.TypeA, Qclass: dns.ClassINET}
	// Set EDNS0 with DO=1
	msg.SetEdns0(512, true)
	in, err := dns.Exchange(msg, "127.0.0.1:15353")
	if err != nil {
		t.Errorf("Error querying the server [%v]", err)
	}
	if in != nil && in.Rcode != dns.RcodeSuccess {
		t.Errorf("Received error from the server [%s]", dns.RcodeToString[in.Rcode])
	}
	opt := in.IsEdns0()
	if opt == nil {
		t.Errorf("Should have got OPT back")
	}
}

func TestEDNSBADVERS(t *testing.T) {
	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.Question = make([]dns.Question, 1)
	msg.Question[0] = dns.Question{Name: dns.Fqdn("auth.example.org"), Qtype: dns.TypeA, Qclass: dns.ClassINET}
	// Set EDNS0 with version 1
	o := new(dns.OPT)
	o.SetVersion(1)
	o.Hdr.Name = "."
	o.Hdr.Rrtype = dns.TypeOPT
	msg.Extra = append(msg.Extra, o)
	in, err := dns.Exchange(msg, "127.0.0.1:15353")
	if err != nil {
		t.Errorf("Error querying the server [%v]", err)
	}
	if in != nil && in.Rcode != dns.RcodeBadVers {
		t.Errorf("Received unexpected rcode from the server [%s]", dns.RcodeToString[in.Rcode])
	}
}

func TestResolveCNAME(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	expected := "cn.example.org.	3600	IN	CNAME	something.example.org."
	answer, err := resolv.lookup("cn.example.org", dns.TypeCNAME)
	if err != nil {
		t.Errorf("Got unexpected error: %s", err)
	}
	if len(answer.Answer) != 1 {
		t.Errorf("Expected exactly 1 RR in answer, but got %d instead.", len(answer.Answer))
	}
	if answer.Answer[0].Header().Rrtype != dns.TypeCNAME {
		t.Errorf("Expected a CNAME answer, but got [%s] instead.", dns.TypeToString[answer.Answer[0].Header().Rrtype])
	}
	if answer.Answer[0].String() != expected {
		t.Errorf("Expected CNAME answer [%s] but got [%s] instead.", expected, answer.Answer[0].String())
	}
}

func TestAuthoritative(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	answer, _ := resolv.lookup("nonexistent.auth.example.org", dns.TypeA)
	if answer.Rcode != dns.RcodeNameError {
		t.Errorf("Was expecing NXDOMAIN rcode, but got [%s] instead.", dns.RcodeToString[answer.Rcode])
	}
	if len(answer.Ns) != 1 {
		t.Errorf("Was expecting exactly one answer (SOA) for invalid subdomain, but got %d", len(answer.Ns))
	}
	if answer.Ns[0].Header().Rrtype != dns.TypeSOA {
		t.Errorf("Was expecting SOA record as answer for NXDOMAIN but got [%s]", dns.TypeToString[answer.Ns[0].Header().Rrtype])
	}
	if !answer.MsgHdr.Authoritative {
		t.Errorf("Was expecting authoritative bit to be set")
	}
	nanswer, _ := resolv.lookup("nonexsitent.nonauth.tld", dns.TypeA)
	if len(nanswer.Answer) > 0 {
		t.Errorf("Didn't expect answers for non authotitative domain query")
	}
	if nanswer.MsgHdr.Authoritative {
		t.Errorf("Authoritative bit should not be set for non-authoritative domain.")
	}
}

func TestResolveTXT(t *testing.T) {
	validTXT := "______________valid_response_______________"
	validTXT2 := "_____________valid_response_2______________"

	atxt, err := DB.Register(cidrslice{})
	if err != nil {
		t.Errorf("Could not initiate db record: [%v]", err)
		return
	}
	atxt.Value = validTXT
	err = DB.Update(atxt.ACMETxtPost)
	if err != nil {
		t.Errorf("Could not update db record: [%v]", err)
		return
	}

	seq := 0
	err = hasExpectedResolveTXTs([]testRecord{
		{atxt.Subdomain, []string{validTXT}, true, true},
		{atxt.Subdomain, []string{"invalid"}, true, false},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", []string{validTXT}, false, false},
	})
	if err != nil {
		t.Fatalf("Test %d: %s", seq, err)
		return
	}
	seq++

	//Add 2nd record and verify it works as expected (both results)
	atxt.Value = validTXT2
	err = DB.Update(atxt.ACMETxtPost)
	if err != nil {
		t.Errorf("Could not update db record: [%v]", err)
		return
	}

	err = hasExpectedResolveTXTs([]testRecord{
		{atxt.Subdomain, []string{validTXT, validTXT2}, true, true},
		{atxt.Subdomain, []string{"invalid"}, true, false},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", []string{validTXT}, false, false},
	})
	if err != nil {
		t.Fatalf("Test %d: %s", seq, err)
		return
	}
	seq++

	//Delete the record and rerun the test, should see only first result again
	atxt.Value = validTXT2
	err = DB.Delete(atxt.ACMETxtPost)
	if err != nil {
		t.Errorf("Could not delete db record: [%v]", err)
		return
	}

	err = hasExpectedResolveTXTs([]testRecord{
		{atxt.Subdomain, []string{validTXT}, true, true},
		{atxt.Subdomain, []string{"invalid"}, true, false},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", []string{validTXT}, false, false},
	})
	if err != nil {
		t.Fatalf("Test %d: %s", seq, err)
		return
	}
	seq++

	//Delete the record and rerun the test, should see nothing
	atxt.Value = validTXT
	err = DB.Delete(atxt.ACMETxtPost)
	if err != nil {
		t.Errorf("Could not delete db record: [%v]", err)
		return
	}

	err = hasExpectedResolveTXTs([]testRecord{
		{atxt.Subdomain, []string{"empty"}, false, false},
		{atxt.Subdomain, []string{"invalid"}, false, false},
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", []string{validTXT}, false, false},
	})
	if err != nil {
		t.Fatalf("Test %d: %s", seq, err)
		return
	}
}

func TestCaseInsensitiveResolveA(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	answer, err := resolv.lookup("aUtH.eXAmpLe.org", dns.TypeA)
	if err != nil {
		t.Errorf("%v", err)
	}

	if len(answer.Answer) == 0 {
		t.Error("No answer for DNS query")
	}
}

func TestCaseInsensitiveResolveSOA(t *testing.T) {
	resolv := resolver{server: "127.0.0.1:15353"}
	answer, _ := resolv.lookup("doesnotexist.aUtH.eXAmpLe.org", dns.TypeSOA)
	if answer.Rcode != dns.RcodeNameError {
		t.Errorf("Was expecing NXDOMAIN rcode, but got [%s] instead.", dns.RcodeToString[answer.Rcode])
	}

	if len(answer.Ns) == 0 {
		t.Error("No SOA answer for DNS query")
	}
}
