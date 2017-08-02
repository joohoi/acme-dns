package main

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"testing"
)

func TestSetupLogging(t *testing.T) {
	for i, test := range []struct {
		format   string
		level    string
		expected string
	}{
		{"text", "warning", "warning"},
		{"json", "debug", "debug"},
		{"text", "info", "info"},
		{"json", "error", "error"},
		{"text", "something", "warning"},
	} {
		setupLogging(test.format, test.level)
		if log.GetLevel().String() != test.expected {
			t.Errorf("Test %d: Expected loglevel %s but got %s", i, test.expected, log.GetLevel().String())
		}
	}
}

func TestReadConfig(t *testing.T) {
	for i, test := range []struct {
		inFile []byte
		output DNSConfig
	}{
		{
			[]byte("[general]\nlisten = \":53\"\ndebug = true\n[api]\napi_domain = \"something.strange\""),
			DNSConfig{
				General: general{
					Listen: ":53",
					Debug:  true,
				},
				API: httpapi{
					Domain: "something.strange",
				},
			},
		},

		{
			[]byte("[\x00[[[[[[[[[de\nlisten =]"),
			DNSConfig{},
		},
	} {
		tmpfile, err := ioutil.TempFile("", "acmedns")
		if err != nil {
			t.Error("Could not create temporary file")
		}
		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(test.inFile); err != nil {
			t.Error("Could not write to temporary file")
		}

		if err := tmpfile.Close(); err != nil {
			t.Error("Could not close temporary file")
		}
		ret := readConfig(tmpfile.Name())
		if ret.General.Listen != test.output.General.Listen {
			t.Errorf("Test %d: Expected listen value %s, but got %s", i, test.output.General.Listen, ret.General.Listen)
		}
		if ret.API.Domain != test.output.API.Domain {
			t.Errorf("Test %d: Expected HTTP API domain %s, but got %s", i, test.output.API.Domain, ret.API.Domain)
		}
	}
}

func TestGetIPListFromHeader(t *testing.T) {
	for i, test := range []struct {
		input  string
		output []string
	}{
		{"1.1.1.1, 2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
		{" 1.1.1.1 , 2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
		{",1.1.1.1 ,2.2.2.2", []string{"1.1.1.1", "2.2.2.2"}},
	} {
		res := getIPListFromHeader(test.input)
		if len(res) != len(test.output) {
			t.Errorf("Test %d: Expected [%d] items in return list, but got [%d]", i, len(test.output), len(res))
		} else {

			for j, vv := range test.output {
				if res[j] != vv {
					t.Errorf("Test %d: Expected return value [%v] but got [%v]", j, test.output, res)
				}

			}
		}
	}
}
