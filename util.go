package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

func jsonError(message string) []byte {
	return []byte(fmt.Sprintf("{\"error\": \"%s\"}", message))
}

func fileIsAccessible(fname string) bool {
	_, err := os.Stat(fname)
	if err != nil {
		return false
	}
	f, err := os.Open(fname)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func readConfig(fname string) (DNSConfig, error) {
	var conf DNSConfig
	_, err := toml.DecodeFile(fname, &conf)
	if err != nil {
		// Return with config file parsing errors from toml package
		return conf, err
	}
	return prepareConfig(conf)
}

// prepareConfig checks that mandatory values exist, and can be used to set default values in the future
func prepareConfig(conf DNSConfig) (DNSConfig, error) {
	if conf.Database.Engine == "" {
		return conf, errors.New("missing database configuration option \"engine\"")
	}
	if conf.Database.Connection == "" {
		return conf, errors.New("missing database configuration option \"connection\"")
	}

	// Default values for options added to config to keep backwards compatibility with old config
	if conf.API.ACMECacheDir == "" {
		conf.API.ACMECacheDir = "api-certs"
	}

	return conf, nil
}

func sanitizeString(s string) string {
	// URL safe base64 alphabet without padding as defined in ACME
	re, _ := regexp.Compile(`[^A-Za-z\-\_0-9]+`)
	return re.ReplaceAllString(s, "")
}

func sanitizeIPv6addr(s string) string {
	// Remove brackets from IPv6 addresses, net.ParseCIDR needs this
	re, _ := regexp.Compile(`[\[\]]+`)
	return re.ReplaceAllString(s, "")
}

func generatePassword(length int) string {
	ret := make([]byte, length)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890-_"
	alphalen := big.NewInt(int64(len(alphabet)))
	for i := 0; i < length; i++ {
		c, _ := rand.Int(rand.Reader, alphalen)
		r := int(c.Int64())
		ret[i] = alphabet[r]
	}
	return string(ret)
}

func sanitizeDomainQuestion(d string) string {
	dom := strings.ToLower(d)
	firstDot := strings.Index(d, ".")
	if firstDot > 0 {
		dom = dom[0:firstDot]
	}
	return dom
}

func setupLogging(logconfig logconfig, readConfigLog string) {
	switch logconfig.Logtype {
	default:
		log.SetOutput(os.Stdout)
		logconfig.Logtype = "stdout"
	case "file":
		file, err := os.OpenFile(logconfig.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.SetOutput(os.Stdout)
		}
	}
	switch logconfig.Level {
	default:
		log.SetLevel(log.WarnLevel)
		logconfig.Level = "warning"
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}
	if logconfig.Format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		logconfig.Format = "text"
	}
	log.WithFields(log.Fields{"file": readConfigLog}).Info("Using config file")
	log.WithFields(log.Fields{"type": logconfig.Logtype}).Info("Set log type")
	log.WithFields(log.Fields{"level": logconfig.Level}).Info("Set log level")
	log.WithFields(log.Fields{"format": logconfig.Format}).Info("Set log format")
	if logconfig.Logtype == "file" {
		log.WithFields(log.Fields{"file": logconfig.File}).Info("Set log file")
	}
}

func setupHTTPLogging(logger *log.Logger, logconfig logconfig) {
	switch logconfig.Logtype {
	default:
		logger.SetOutput(os.Stdout)
	case "file":
		file, err := os.OpenFile(logconfig.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.SetOutput(os.Stdout)
		}
	}
	switch logconfig.Level {
	default:
		logger.SetLevel(log.WarnLevel)
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "info":
		logger.SetLevel(log.InfoLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	}
	if logconfig.Format == "json" {
		logger.SetFormatter(&log.JSONFormatter{})
	}
	logger.WithFields(log.Fields{"type": logconfig.Logtype}).Info("HTTP logger set log type")
	logger.WithFields(log.Fields{"level": logconfig.Level}).Info("HTTP logger Set log level")
	logger.WithFields(log.Fields{"format": logconfig.Format}).Info("HTTP logger Set log format")
	if logconfig.Logtype == "file" {
		logger.WithFields(log.Fields{"file": logconfig.File}).Info("HTTP logger Set log file")
	}
}

func getIPListFromHeader(header string) []string {
	iplist := []string{}
	for _, v := range strings.Split(header, ",") {
		if len(v) > 0 {
			// Ignore empty values
			iplist = append(iplist, strings.TrimSpace(v))
		}
	}
	return iplist
}
