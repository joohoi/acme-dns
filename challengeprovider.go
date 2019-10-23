package main

import "github.com/go-acme/lego/challenge/dns01"

// ChallengeProvider implements go-acme/lego Provider interface which is used for ACME DNS challenge handling
type ChallengeProvider struct {
	servers []*DNSServer
}

// NewChallengeProvider creates a new instance of ChallengeProvider
func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

// Present is used for making the ACME DNS challenge token available for DNS
func (c *ChallengeProvider) Present(_, _, keyAuth string) error {
	_, token := dns01.GetRecord("whatever", keyAuth)
	for _, s := range c.servers {
		s.PersonalKeyAuth = token
	}
	return nil
}

// CleanUp is called after the run to remove the ACME DNS challenge tokens from DNS records
func (c *ChallengeProvider) CleanUp(_, _, _ string) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = ""
	}
	return nil
}
