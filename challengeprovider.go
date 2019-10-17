package main

type ChallengeProvider struct {
	Servers []*DNSServer
}

func NewChallengeProvider(servers []*DNSServer) ChallengeProvider {
	c := &ChallengeProvider{Servers: servers}
	return c
}

func (c *ChallengeProvider) Present(_, _, keyAuth string) error {
	for i, s := range c.Servers {
		s.PersonalKeyAuth = keyAuth
	}
	return nil
}

func (c *ChallengeProvider) CleanUp(_, _, _ string) error {
	for i, s := range c.Servers {
		s.PersonalKeyAuth = ""
	}
	return nil
}
