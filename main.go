package main

import (
	log "github.com/Sirupsen/logrus"
	"os"
)

// DNSConf is global configuration struct
var DNSConf DNSConfig

// DB is used to access the database functions in acme-dns
var DB database

// RR holds the static DNS records
var RR Records

func main() {
	// Read global config
	configTmp := readConfig("config.cfg")
	DNSConf = configTmp

	setupLogging(DNSConf.Logconfig.Format, DNSConf.Logconfig.Level)

	// Read the default records in
	RR.Parse(DNSConf.General.StaticRecords)

	// Open database
	newDB := new(acmedb)
	err := newDB.Init(DNSConf.Database.Engine, DNSConf.Database.Connection)
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	DB = newDB
	defer DB.Close()

	// DNS server
	startDNS(DNSConf.General.Listen)

	// HTTP API
	startHTTPAPI()

	log.Debugf("Shutting down...")
}
