package main

import "github.com/go-acme/lego/challenge/dns01"

type ChallengeProvider struct {
	servers []*DNSServer
}

func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

func (c *ChallengeProvider) Present(_, _, keyAuth string) error {
	_, token := dns01.GetRecord("whatever", keyAuth)
	for _, s := range c.servers {
		s.PersonalKeyAuth = token
	}
	return nil
}

func (c *ChallengeProvider) CleanUp(_, _, _ string) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = ""
	}
	return nil
}
