package nameserver

import (
	"strings"

	"github.com/acme-dns/acme-dns/pkg/acmedns"

	"github.com/miekg/dns"
	"go.uber.org/zap"
)

// Records is a slice of ResourceRecords
type Records struct {
	Records []dns.RR
}

type Nameserver struct {
	Config          *acmedns.AcmeDnsConfig
	DB              acmedns.AcmednsDB
	Logger          *zap.SugaredLogger
	Server          *dns.Server
	OwnDomain       string
	SOA             dns.RR
	personalAuthKey string
	Domains         map[string]Records
	errChan         chan error
}

func InitAndStart(config *acmedns.AcmeDnsConfig, db acmedns.AcmednsDB, logger *zap.SugaredLogger, errChan chan error) []acmedns.AcmednsNS {
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
		go dnsServerUDP.Start(errChan)
		go dnsServerTCP.Start(errChan)
	} else {
		dnsServer := NewDNSServer(config, db, logger, config.General.Proto)
		dnsservers = append(dnsservers, dnsServer)
		dnsServer.ParseRecords()
		go dnsServer.Start(errChan)
	}
	return dnsservers
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
	server.DB = db
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
	err := n.Server.ListenAndServe()
	if err != nil {
		errorChannel <- err
	}
}
