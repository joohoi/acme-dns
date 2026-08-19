package nameserver

import (
	"errors"
	"testing"

	"github.com/miekg/dns"

	"github.com/joohoi/acme-dns/pkg/acmedns"
)

func TestNODATAResponseIncludesSOA(t *testing.T) {
	ns, db, _ := setupDNS()
	t.Cleanup(db.Close)
	server := ns.(*Nameserver)

	registration, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Fatalf("registering dynamic name: %v", err)
	}
	if err := db.Update(acmedns.ACMETxtPost{
		Subdomain: registration.Subdomain,
		Value:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}); err != nil {
		t.Fatalf("adding dynamic TXT record: %v", err)
	}
	emptyRegistration, err := db.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Fatalf("registering empty dynamic name: %v", err)
	}

	tests := []struct {
		name      string
		qname     string
		wantRcode int
	}{
		{
			name:      "existing static name",
			qname:     "auth.example.org.",
			wantRcode: dns.RcodeSuccess,
		},
		{
			name:      "existing dynamic name",
			qname:     registration.Subdomain + ".auth.example.org.",
			wantRcode: dns.RcodeSuccess,
		},
		{
			name:      "registered name without TXT data",
			qname:     emptyRegistration.Subdomain + ".auth.example.org.",
			wantRcode: dns.RcodeNameError,
		},
		{
			name:      "nonexistent name",
			qname:     "nonexistent.auth.example.org.",
			wantRcode: dns.RcodeNameError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := new(dns.Msg)
			msg.SetQuestion(tc.qname, dns.TypeCAA)
			server.readQuery(msg)

			if msg.Rcode != tc.wantRcode {
				t.Fatalf("rcode = %s, want %s", dns.RcodeToString[msg.Rcode], dns.RcodeToString[tc.wantRcode])
			}
			if !msg.Authoritative {
				t.Fatal("authoritative bit is not set")
			}
			if len(msg.Answer) != 0 {
				t.Fatalf("answer count = %d, want 0", len(msg.Answer))
			}
			if len(msg.Ns) != 1 || msg.Ns[0].Header().Rrtype != dns.TypeSOA {
				t.Fatalf("authority = %v, want one SOA", msg.Ns)
			}
		})
	}
}

type failingTXTLookupDB struct {
	acmedns.AcmednsDB
}

func (failingTXTLookupDB) GetTXTForDomain(string) ([]string, error) {
	return nil, errors.New("lookup failed")
}

func TestNODATALookupScopeAndErrors(t *testing.T) {
	ns, db, _ := setupDNS()
	t.Cleanup(db.Close)
	server := ns.(*Nameserver)
	server.DB = failingTXTLookupDB{AcmednsDB: db}

	tests := []struct {
		name          string
		qname         string
		wantRcode     int
		wantAuthority bool
		wantSOA       bool
	}{
		{
			name:          "candidate dynamic name",
			qname:         "candidate.auth.example.org.",
			wantRcode:     dns.RcodeServerFailure,
			wantAuthority: true,
		},
		{
			name:          "existing static name skips lookup",
			qname:         "auth.example.org.",
			wantRcode:     dns.RcodeSuccess,
			wantAuthority: true,
			wantSOA:       true,
		},
		{
			name:          "nested name skips lookup",
			qname:         "candidate.nested.auth.example.org.",
			wantRcode:     dns.RcodeNameError,
			wantAuthority: true,
			wantSOA:       true,
		},
		{
			name:      "non-authoritative name skips lookup",
			qname:     "candidate.example.net.",
			wantRcode: dns.RcodeNameError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := new(dns.Msg)
			msg.SetQuestion(tc.qname, dns.TypeCAA)
			server.readQuery(msg)

			if msg.Rcode != tc.wantRcode {
				t.Fatalf("rcode = %s, want %s", dns.RcodeToString[msg.Rcode], dns.RcodeToString[tc.wantRcode])
			}
			if msg.Authoritative != tc.wantAuthority {
				t.Fatalf("authoritative = %v, want %v", msg.Authoritative, tc.wantAuthority)
			}
			if len(msg.Answer) != 0 {
				t.Fatalf("unexpected answer records: %v", msg.Answer)
			}
			if tc.wantSOA {
				if len(msg.Ns) != 1 || msg.Ns[0].Header().Rrtype != dns.TypeSOA {
					t.Fatalf("authority = %v, want one SOA", msg.Ns)
				}
			} else if len(msg.Ns) != 0 {
				t.Fatalf("unexpected authority records: %v", msg.Ns)
			}
		})
	}
}
