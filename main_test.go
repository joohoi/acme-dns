package main

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	server, resolv = startDNSServer("0.0.0.0:15353")
	flag.Parse()
	exitval := m.Run()
	server.Shutdown()
	DB.DB.Close()
	os.Exit(exitval)
}
