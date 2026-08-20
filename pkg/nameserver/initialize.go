package nameserver

import (
	"fmt"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"go.uber.org/zap"

	"github.com/joohoi/acme-dns/pkg/acmedns"
)

// Records is a slice of ResourceRecords
type Records struct {
	Records []dns.RR
}

type Nameserver struct {
	Config            *acmedns.AcmeDnsConfig
	DB                acmedns.AcmednsDB
	Logger            *zap.SugaredLogger
	Server            *dns.Server
	OwnDomain         string
	NotifyStartedFunc func()
	SOA               dns.RR
	mu                sync.RWMutex
	personalAuthKey   string
	Domains           map[string]Records
	errChan           chan error
}

func InitAndStart(config *acmedns.AcmeDnsConfig, db acmedns.AcmednsDB, logger *zap.SugaredLogger, errChan chan error) ([]acmedns.AcmednsNS, error) {
	dnsservers := make([]acmedns.AcmednsNS, 0)
	if strings.HasPrefix(config.General.Proto, "both") {

		// Handle the case where DNS server should be started for both udp and tcp
		udpProto := "udp"
		tcpProto := "tcp"
		if strings.HasSuffix(config.General.Proto, "4") {
			udpProto += "4"
			tcpProto += "4"
		} else if strings.HasSuffix(config.General.Proto, "6") {
			udpProto += "6"
			tcpProto += "6"
		}
		dnsServerUDP := NewDNSServer(config, db, logger, udpProto)
		dnsservers = append(dnsservers, dnsServerUDP)
		dnsServerUDP.ParseRecords()
		dnsServerTCP := NewDNSServer(config, db, logger, tcpProto)
		dnsservers = append(dnsservers, dnsServerTCP)
		dnsServerTCP.ParseRecords()
		// wait for the servers to get started to proceed
		if err := startAndWait(dnsServerUDP, errChan); err != nil {
			return dnsservers, err
		}
		if err := startAndWait(dnsServerTCP, errChan); err != nil {
			return dnsservers, err
		}
	} else {
		dnsServer := NewDNSServer(config, db, logger, config.General.Proto)
		dnsservers = append(dnsservers, dnsServer)
		dnsServer.ParseRecords()
		if err := startAndWait(dnsServer, errChan); err != nil {
			return dnsservers, err
		}
	}
	return dnsservers, nil
}

// startAndWait starts a DNS server and blocks until it has either bound its
// listener or failed to do so. A server that never binds also never calls its
// NotifyStartedFunc, so waiting for the start signal alone would block forever
// on an unusable listener - an unset or misspelled general.protocol, a port
// that is already taken, or missing privileges for port 53 - leaving acme-dns
// hung with no indication of what went wrong.
func startAndWait(server acmedns.AcmednsNS, errChan chan error) error {
	started := make(chan struct{})
	server.SetNotifyStartedFunc(func() { close(started) })
	go server.Start(errChan)
	select {
	case <-started:
		return nil
	case err := <-errChan:
		return err
	}
}

// NewDNSServer parses the DNS records from config and returns a new DNSServer struct
func NewDNSServer(config *acmedns.AcmeDnsConfig, db acmedns.AcmednsDB, logger *zap.SugaredLogger, proto string) acmedns.AcmednsNS {
	//		dnsServerTCP := NewDNSServer(DB, Config.General.Listen, tcpProto, Config.General.Domain)
	server := Nameserver{Config: config, DB: db, Logger: logger}
	server.Server = &dns.Server{Addr: config.General.Listen, Net: proto}
	domain := config.General.Domain
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}
	server.OwnDomain = strings.ToLower(domain)
	server.personalAuthKey = ""
	server.Domains = make(map[string]Records)
	return &server
}

func (n *Nameserver) Start(errorChannel chan error) {
	n.errChan = errorChannel
	dns.HandleFunc(".", n.handleRequest)
	n.Logger.Infow("Starting DNS listener",
		"addr", n.Server.Addr,
		"proto", n.Server.Net)
	if n.NotifyStartedFunc != nil {
		n.Server.NotifyStartedFunc = n.NotifyStartedFunc
	}
	err := n.Server.ListenAndServe()
	if err != nil {
		errorChannel <- fmt.Errorf("DNS server %s failed: %w", n.Server.Net, err)
	}
}

func (n *Nameserver) SetNotifyStartedFunc(fun func()) {
	n.Server.NotifyStartedFunc = fun
}
