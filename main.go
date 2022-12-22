package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/acme-dns/acme-dns/pkg/api"
	"github.com/acme-dns/acme-dns/pkg/nameserver"
	"os"
	"syscall"

	"github.com/acme-dns/acme-dns/pkg/acmedns"
	"github.com/acme-dns/acme-dns/pkg/database"

	"go.uber.org/zap"
)

func setupLogging(config acmedns.AcmeDnsConfig) (*zap.Logger, error) {
	var logger *zap.Logger
	logformat := "console"
	if config.Logconfig.Format == "json" {
		logformat = "json"
	}
	outputPath := "stdout"
	if config.Logconfig.Logtype == "file" {
		outputPath = config.Logconfig.File
	}
	errorPath := "stderr"
	if config.Logconfig.Logtype == "file" {
		errorPath = config.Logconfig.File
	}
	zapConfigJson := fmt.Sprintf(`{
   "level": "%s",
   "encoding": "%s",
   "outputPaths": ["%s"],
   "errorOutputPaths": ["%s"],
   "encoderConfig": {
	 "timeKey": "time",
     "messageKey": "msg",
     "levelKey": "level",
     "levelEncoder": "lowercase",
	 "timeEncoder": "iso8601"
   }
 }`, config.Logconfig.Level, logformat, outputPath, errorPath)
	var zapCfg zap.Config
	if err := json.Unmarshal([]byte(zapConfigJson), &zapCfg); err != nil {
		return logger, err
	}
	logger, err := zapCfg.Build()
	return logger, err
}

func readConfig(configFile string) (acmedns.AcmeDnsConfig, string, error) {
	var usedConfigFile string
	var config acmedns.AcmeDnsConfig
	var err error
	if acmedns.FileIsAccessible(configFile) {
		usedConfigFile = configFile
		config, err = acmedns.ReadConfig(configFile)
	} else if acmedns.FileIsAccessible("./config.cfg") {
		usedConfigFile = "./config.cfg"
		config, err = acmedns.ReadConfig("./config.cfg")
	} else {
		err = fmt.Errorf("configuration file not found")
	}
	if err != nil {
		err = fmt.Errorf("encountered an error while trying to read configuration file:  %s\n", err)
	}
	return config, usedConfigFile, err
}

func main() {
	syscall.Umask(0077)
	configPtr := flag.String("c", "/etc/acme-dns/config.cfg", "config file location")
	flag.Parse()
	// Read global config
	var err error
	var logger *zap.Logger
	config, usedConfigFile, err := readConfig(*configPtr)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	logger, err = setupLogging(config)
	if err != nil {
		fmt.Printf("Could not set up logging: %s\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
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
