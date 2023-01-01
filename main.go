package main

import (
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/acme-dns/acme-dns/pkg/acmedns"
	"github.com/acme-dns/acme-dns/pkg/api"
	"github.com/acme-dns/acme-dns/pkg/database"
	"github.com/acme-dns/acme-dns/pkg/nameserver"

	"go.uber.org/zap"
)

func main() {
	syscall.Umask(0077)
	configPtr := flag.String("c", "/etc/acme-dns/config.cfg", "config file location")
	flag.Parse()
	// Read global config
	var err error
	var logger *zap.Logger
	config, usedConfigFile, err := acmedns.ReadConfig(*configPtr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	logger, err = acmedns.SetupLogging(config)
	if err != nil {
		fmt.Printf("Could not set up logging: %s\n", err)
		os.Exit(1)
	}
	// Make sure to flush the zap logger buffer before exiting
	defer logger.Sync() //nolint:all
	sugar := logger.Sugar()

	sugar.Infow("Using config file",
		"file", usedConfigFile)
	sugar.Info("Starting up")
	db, err := database.Init(&config, sugar)
	// Error channel for servers
	errChan := make(chan error, 1)
	api := api.Init(&config, db, sugar, errChan)
	dnsservers := nameserver.InitAndStart(&config, db, sugar, errChan)
	go api.Start(dnsservers)
	if err != nil {
		sugar.Error(err)
	}
	for {
		err = <-errChan
		if err != nil {
			sugar.Fatal(err)
		}
	}
}
