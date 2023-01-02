package acmedns

import (
	"encoding/json"
	"net"
)

// cidrslice is a list of allowed cidr ranges
type Cidrslice []string

func (c *Cidrslice) JSON() string {
	ret, _ := json.Marshal(c.ValidEntries())
	return string(ret)
}

func (c *Cidrslice) IsValid() error {
	for _, v := range *c {
		_, _, err := net.ParseCIDR(sanitizeIPv6addr(v))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cidrslice) ValidEntries() []string {
	valid := []string{}
	for _, v := range *c {
		_, _, err := net.ParseCIDR(sanitizeIPv6addr(v))
		if err == nil {
			valid = append(valid, sanitizeIPv6addr(v))
		}
	}
	return valid
}
