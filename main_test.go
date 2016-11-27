package main

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

var (
	postgres = flag.Bool("postgres", false, "run integration tests against PostgreSQL")
)

func TestMain(m *testing.M) {
	setupConfig()
	RR.Parse(records)
	flag.Parse()

	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			fmt.Println("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			os.Exit(1)
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}

	server := startDNS("0.0.0.0:15353")
	exitval := m.Run()
	server.Shutdown()
	DB.DB.Close()
	os.Exit(exitval)
}

func setupConfig() {
	var dbcfg = dbsettings{
		Engine:     "sqlite3",
		Connection: ":memory:",
	}

	var generalcfg = general{
		Domain:  "auth.example.org",
		Nsname:  "ns1.auth.example.org",
		Nsadmin: "admin.example.org",
		Debug:   false,
	}

	var httpapicfg = httpapi{
		Domain:      "",
		Port:        "8080",
		TLS:         "none",
		CorsOrigins: []string{"*"},
	}

	var dnscfg = DNSConfig{
		Database: dbcfg,
		General:  generalcfg,
		API:      httpapicfg,
	}

	DNSConf = dnscfg
}
