package nameserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// ParseRecords parses a slice of DNS record string
func (n *Nameserver) ParseRecords() {
	for _, v := range n.Config.General.StaticRecords {
		rr, err := dns.NewRR(strings.ToLower(v))
		if err != nil {
			n.Logger.Errorw("Could not parse RR from config",
				"error", err.Error(),
				"rr", v)
			continue
		}
		// Add parsed RR
		n.appendRR(rr)
	}
	// Create serial
	serial := time.Now().Format("2006010215")
	// Add SOA
	SOAstring := fmt.Sprintf("%s. SOA %s. %s. %s 28800 7200 604800 86400", strings.ToLower(n.Config.General.Domain), strings.ToLower(n.Config.General.Nsname), strings.ToLower(n.Config.General.Nsadmin), serial)
	soarr, err := dns.NewRR(SOAstring)
	if err != nil {
		n.Logger.Errorw("Error while adding SOA record",
			"error", err.Error(),
			"soa", SOAstring)
	} else {
		n.appendRR(soarr)
		n.SOA = soarr
	}
}

func (n *Nameserver) appendRR(rr dns.RR) {
	addDomain := rr.Header().Name
	_, ok := n.Domains[addDomain]
	if !ok {
		n.Domains[addDomain] = Records{[]dns.RR{rr}}
	} else {
		drecs := n.Domains[addDomain]
		drecs.Records = append(drecs.Records, rr)
		n.Domains[addDomain] = drecs
	}
	n.Logger.Debugw("Adding new record to domain",
		"recordtype", dns.TypeToString[rr.Header().Rrtype],
		"domain", addDomain)
}
