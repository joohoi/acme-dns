package main

import (
	"context"
	"github.com/mholt/acmez/acme"
)

// ChallengeProvider implements go-acme/lego Provider interface which is used for ACME DNS challenge handling
type ChallengeProvider struct {
	servers []*DNSServer
}

// NewChallengeProvider creates a new instance of ChallengeProvider
func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

// Present is used for making the ACME DNS challenge token available for DNS
func (c *ChallengeProvider) Present(ctx context.Context, challenge acme.Challenge) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = challenge.DNS01KeyAuthorization()
	}
	return nil
}

// CleanUp is called after the run to remove the ACME DNS challenge tokens from DNS records
func (c *ChallengeProvider) CleanUp(ctx context.Context, _ acme.Challenge) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = ""
	}
	return nil
}

// Wait is a dummy function as we are just going to be ready to answer the challenge from the get-go
func (c *ChallengeProvider) Wait(_ context.Context, _ acme.Challenge) error {
	return nil
}
