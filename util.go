package main

import (
	"crypto/rand"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
	"github.com/satori/go.uuid"
	"math/big"
	"regexp"
	"strings"
)

func readConfig(fname string) DNSConfig {
	var conf DNSConfig
	// Practically never errors
	_, _ = toml.DecodeFile(fname, &conf)
	return conf
}

func sanitizeString(s string) string {
	// URL safe base64 alphabet without padding as defined in ACME
	re, _ := regexp.Compile("[^A-Za-z\\-\\_0-9]+")
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

func newACMETxt() ACMETxt {
	var a = ACMETxt{}
	password := generatePassword(40)
	a.Username = uuid.NewV4()
	a.Password = password
	a.Subdomain = uuid.NewV4().String()
	return a
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

func startDNS(listen string, proto string) *dns.Server {
	// DNS server part
	dns.HandleFunc(".", handleRequest)
	server := &dns.Server{Addr: listen, Net: proto}
	go server.ListenAndServe()
	return server
}
