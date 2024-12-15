package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

var loghook = new(logrustest.Hook)
var dnsserver *DNSServer

var (
	postgres = flag.Bool("postgres", false, "run integration tests against PostgreSQL")
)

var records = []string{
	"auth.example.org. A 192.168.1.100",
	"ns1.auth.example.org. A 192.168.1.101",
	"cn.example.org CNAME something.example.org.",
	"!''b', unparseable ",
	"ns2.auth.example.org. A 192.168.1.102",
}

func TestMain(m *testing.M) {
	setupTestLogger()
	setupConfig()
	flag.Parse()

	newDb := new(acmedb)
	if *postgres {
		Config.Database.Engine = "postgres"
		err := newDb.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			fmt.Println("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			os.Exit(1)
		}
	} else {
		Config.Database.Engine = "sqlite3"
		_ = newDb.Init("sqlite3", ":memory:")
	}
	DB = newDb
	dnsserver = NewDNSServer(DB, Config.General.Listen, Config.General.Proto, Config.General.Domain)
	dnsserver.ParseRecords(Config)

	// Make sure that we're not creating a race condition in tests
	var wg sync.WaitGroup
	wg.Add(1)
	dnsserver.Server.NotifyStartedFunc = func() {
		wg.Done()
	}
	go dnsserver.Start(make(chan error, 1))
	wg.Wait()
	exitval := m.Run()
	_ = dnsserver.Server.Shutdown()
	DB.Close()
	os.Exit(exitval)
}

func setupConfig() {
	var dbcfg = dbsettings{
		Engine:     "sqlite3",
		Connection: ":memory:",
	}

	var generalcfg = general{
		Domain:        "auth.example.org",
		Listen:        "127.0.0.1:15353",
		Proto:         "udp",
		Nsname:        "ns1.auth.example.org",
		Nsadmin:       "admin.example.org",
		StaticRecords: records,
		Debug:         false,
	}

	var httpapicfg = httpapi{
		Domain:      "",
		Port:        "8080",
		TLS:         "none",
		CorsOrigins: []string{"*"},
		UseHeader:   false,
		HeaderName:  "X-Forwarded-For",
	}

	var dnscfg = DNSConfig{
		Database: dbcfg,
		General:  generalcfg,
		API:      httpapicfg,
	}

	Config = dnscfg
}

func setupTestLogger() {
	log.SetOutput(io.Discard)
	log.AddHook(loghook)
}

func loggerHasEntryWithMessage(message string) bool {
	for _, v := range loghook.Entries {
		if v.Message == message {
			return true
		}
	}
	return false
}
