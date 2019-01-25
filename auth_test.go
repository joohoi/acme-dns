package main

import (
	"net/http"
	"testing"
)

func TestUpdateAllowedFromIP(t *testing.T) {
	Config.API.UseHeader = false
	userWithAllow := newACMETxt()
	userWithAllow.AllowFrom = cidrslice{"192.168.1.2/32", "[::1]/128"}
	userWithoutAllow := newACMETxt()

	for i, test := range []struct {
		remoteaddr string
		expected   bool
	}{
		{"192.168.1.2:1234", true},
		{"192.168.1.1:1234", false},
		{"invalid", false},
		{"[::1]:4567", true},
	} {
		newreq, _ := http.NewRequest("GET", "/whatever", nil)
		newreq.RemoteAddr = test.remoteaddr
		ret := updateAllowedFromIP(newreq, userWithAllow)
		if test.expected != ret {
			t.Errorf("Test %d: Unexpected result for user with allowForm set", i)
		}

		if !updateAllowedFromIP(newreq, userWithoutAllow) {
			t.Errorf("Test %d: Unexpected result for user without allowForm set", i)
		}
	}
}
