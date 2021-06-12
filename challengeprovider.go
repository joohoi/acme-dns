package main

import (
	"context"
	"strings"

	"github.com/libdns/libdns"
	log "github.com/sirupsen/logrus"
)

// ChallengeProvider implements go-acme/lego Provider interface which is used for ACME DNS challenge handling
type ChallengeProvider struct {
	servers []*DNSServer
}

// NewChallengeProvider creates a new instance of ChallengeProvider
func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

func (c *ChallengeProvider) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	var token string
	for _, item := range recs {

		log.WithFields(log.Fields{"name": item.Name, "value": item.Value, "type": item.Type}).Info("Attempting to set dns record")
		if strings.Contains(item.Name, "acme-challenge") {
			token = item.Value
			break
		}
	}

	for _, s := range c.servers {
		s.PersonalKeyAuth = token
	}
	return recs, nil
}

func (c *ChallengeProvider) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	for _, item := range recs {
		log.WithFields(log.Fields{"name": item.Name, "value": item.Value, "type": item.Type}).Info("Attempting to unset dns record")
	}
	for _, s := range c.servers {
		s.PersonalKeyAuth = ""
	}
	return recs, nil
}
