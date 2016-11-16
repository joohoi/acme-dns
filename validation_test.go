package main

import (
	"github.com/satori/go.uuid"
	"testing"
)

func TestGetValidUsername(t *testing.T) {
	v1, _ := uuid.FromString("a097455b-52cc-4569-90c8-7a4b97c6eba8")
	for i, test := range []struct {
		uname     string
		output    uuid.UUID
		shouldErr bool
	}{
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", v1, false},
		{"a-97455b-52cc-4569-90c8-7a4b97c6eba8", uuid.UUID{}, true},
		{"", uuid.UUID{}, true},
		{"&!#!25123!%!'%", uuid.UUID{}, true},
	} {
		ret, err := GetValidUsername(test.uname)
		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error, but there was none", i)
		}
		if !test.shouldErr && err != nil {
			t.Errorf("Test %d: Expected no error, but got [%v]", i, err)
		}
		if ret != test.output {
			t.Errorf("Test %d: Expected return value %v, but got %v", i, test.output, ret)
		}
	}
}

func TestValidKey(t *testing.T) {
	for i, test := range []struct {
		key    string
		output bool
	}{
		{"", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"aaaaaaaa-aaa-aaaaaa-aaaaaaaa-aaa_aacaaaa", true},
		{"aaaaaaaa-aaa-aaaaaa#aaaaaaaa-aaa_aacaaaa", false},
		{"aaaaaaaa-aaa-aaaaaa-aaaaaaaa-aaa_aacaaaaa", false},
	} {
		ret := ValidKey(test.key)
		if ret != test.output {
			t.Errorf("Test %d: Expected return value %t, but got %t", i, test.output, ret)
		}
	}
}

func TestGetValidSubdomain(t *testing.T) {
	for i, test := range []struct {
		subdomain string
		output    bool
	}{
		{"a097455b-52cc-4569-90c8-7a4b97c6eba8", true},
		{"a-97455b-52cc-4569-90c8-7a4b97c6eba8", false},
		{"", false},
		{"&!#!25123!%!'%", false},
	} {
		ret := ValidSubdomain(test.subdomain)
		if ret != test.output {
			t.Errorf("Test %d: Expected return value %t, but got %t", i, test.output, ret)
		}
	}
}

func TestValidTXT(t *testing.T) {
	for i, test := range []struct {
		txt    string
		output bool
	}{
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaa#aaaaaaaaaaaaaa", false},
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},
		{"", false},
	} {
		ret := ValidTXT(test.txt)
		if ret != test.output {
			t.Errorf("Test %d: Expected return value %t, but got %t", i, test.output, ret)
		}
	}
}

func TestCorrectPassword(t *testing.T) {
	for i, test := range []struct {
		pw     string
		hash   string
		output bool
	}{
		{"PUrNTjU24JYNEOCeS2JcjaJGv1sinT80oV9--dpX",
			"$2a$10$ldVoGU5yrdlbPzuPUbUfleVovGjaRelP9tql0IltVUJk778gf.2tu",
			true},
		{"PUrNTjU24JYNEOCeS2JcjaJGv1sinT80oV9--dpX",
			"$2a$10$ldVoGU5yrdlbPzuPUbUfleVovGjaRelP9tql0IltVUJk778gf.2t",
			false},
		{"PUrNTjU24JYNEOCeS2JcjaJGv1sinT80oV9--dp",
			"$2a$10$ldVoGU5yrdlbPzuPUbUfleVovGjaRelP9tql0IltVUJk778gf.2tu",
			false},
		{"", "", false},
	} {
		ret := CorrectPassword(test.pw, test.hash)
		if ret != test.output {
			t.Errorf("Test %d: Expected return value %t, but got %t", i, test.output, ret)
		}
	}
}
