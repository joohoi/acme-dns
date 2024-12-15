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
	md, err := toml.DecodeFile(fname, &conf)
	if err != nil {
		// Return with config file parsing errors from toml package
		return conf, err
	}
	undecoded := md.Undecoded()
	if len(undecoded) > 0 {
		return conf, fmt.Errorf("Unexpected keys: %v", undecoded)
	}
	return prepareConfig(conf)
}

// prepareConfig checks that mandatory values exist, and can be used to set default values in the future
func prepareConfig(conf DNSConfig) (DNSConfig, error) {
	if conf.General.Domain == "" {
		return conf, errors.New("missing general configuration option \"domain\"")
	}
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

func setupLogging(format string, level string) {
	if format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}
	switch level {
	default:
		log.SetLevel(log.WarnLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}
	// TODO: file logging
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
