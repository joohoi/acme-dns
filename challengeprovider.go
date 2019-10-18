package main

type ChallengeProvider struct {
	servers []*DNSServer
}

func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	return ChallengeProvider{servers: servers}
}

func (c *ChallengeProvider) Present(_, _, keyAuth string) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = keyAuth
	}
	return nil
}

func (c *ChallengeProvider) CleanUp(_, _, _ string) error {
	for _, s := range c.servers {
		s.PersonalKeyAuth = ""
	}
	return nil
}
