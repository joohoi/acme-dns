package main

import (
	"fmt"
	"github.com/kataras/iris"
	"github.com/miekg/dns"
	"github.com/op/go-logging"
	"os"
)

// Logging config
var logfile_path = "acme-dns.log"
var log = logging.MustGetLogger("acme-dns")

// Global configuration struct
var DnsConf DnsConfig

var DB Database

// Static records
var RR Records

func main() {
	// Setup logging
	var stdout_format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	var file_format = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{shortfunc} - %{level:.4s} %{id:03x} %{message}`,
	)
	// Setup logging - stdout
	logStdout := logging.NewLogBackend(os.Stdout, "", 0)
	logStdoutFormatter := logging.NewBackendFormatter(logStdout, stdout_format)
	// Setup logging - file
	// Logging to file
	logfh, err := os.OpenFile(logfile_path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Could not open log file %s\n", logfile_path)
		os.Exit(1)
	}
	defer logfh.Close()
	logFile := logging.NewLogBackend(logfh, "", 0)
	logFileFormatter := logging.NewBackendFormatter(logFile, file_format)
	/* To limit logging to a level
	logFileLeveled := logging.AddModuleLevel(logFile)
	logFileLeveled.SetLevel(logging.ERROR, "")
	*/

	// Start logging
	logging.SetBackend(logStdoutFormatter, logFileFormatter)
	log.Debug("Starting up...")

	// Read global config
	if DnsConf, err = ReadConfig("config.cfg"); err != nil {
		log.Errorf("Got error %v", err)
		os.Exit(1)
	}
	RR.Parse(DnsConf.General.StaticRecords)

	// Open database
	err = DB.Init("acme-dns.db")
	if err != nil {
		log.Errorf("Could not open database [%v]", err)
		os.Exit(1)
	}
	defer DB.DB.Close()

	// DNS server part
	dns.HandleFunc(".", handleRequest)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			log.Errorf("%v", err)
			os.Exit(1)
		}
	}()

	// API server
	api := iris.New()
	for path, handlerfunc := range GetHandlerMap() {
		api.Get(path, handlerfunc)
	}
	for path, handlerfunc := range PostHandlerMap() {
		api.Post(path, handlerfunc)
	}
	api.Listen(":8080")
	log.Debugf("Shutting down...")
}
