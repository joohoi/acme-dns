package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/op/go-logging"
	"github.com/satori/go.uuid"
	"math/big"
	"os"
	"regexp"
	"strings"
)

func readConfig(fname string) (DNSConfig, error) {
	var conf DNSConfig
	if _, err := toml.DecodeFile(fname, &conf); err != nil {
		return DNSConfig{}, errors.New("Malformed configuration file")
	}
	return conf, nil
}

func sanitizeString(s string) string {
	// URL safe base64 alphabet without padding as defined in ACME
	re, err := regexp.Compile("[^A-Za-z\\-\\_0-9]+")
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	return re.ReplaceAllString(s, "")
}

func generatePassword(length int) (string, error) {
	ret := make([]byte, length)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890-_"
	alphalen := big.NewInt(int64(len(alphabet)))
	for i := 0; i < length; i++ {
		c, err := rand.Int(rand.Reader, alphalen)
		if err != nil {
			return "", err
		}
		r := int(c.Int64())
		ret[i] = alphabet[r]
	}
	return string(ret), nil
}

func sanitizeDomainQuestion(d string) string {
	var dom string
	suffix := DNSConf.General.Domain + "."
	if strings.HasSuffix(d, suffix) {
		dom = d[0 : len(d)-len(suffix)]
	} else {
		dom = d
	}
	return dom
}

func newACMETxt() (ACMETxt, error) {
	var a = ACMETxt{}
	password, err := generatePassword(40)
	if err != nil {
		return a, err
	}
	a.Username = uuid.NewV4()
	a.Password = password
	a.Subdomain = uuid.NewV4().String()
	return a, nil
}

func setupLogging() {
	var logformat = logging.MustStringFormatter(DNSConf.Logconfig.Format)
	var logBackend *logging.LogBackend
	switch DNSConf.Logconfig.Logtype {
	default:
		// Setup logging - stdout
		logBackend = logging.NewLogBackend(os.Stdout, "", 0)
	case "file":
		// Logging to file
		logfh, err := os.OpenFile(DNSConf.Logconfig.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Printf("Could not open log file %s\n", DNSConf.Logconfig.File)
			os.Exit(1)
		}
		defer logfh.Close()
		logBackend = logging.NewLogBackend(logfh, "", 0)
	}
	logFormatter := logging.NewBackendFormatter(logBackend, logformat)
	logLevel := logging.AddModuleLevel(logFormatter)
	switch DNSConf.Logconfig.Level {
	default:
		logLevel.SetLevel(logging.DEBUG, "")
	case "warning":
		logLevel.SetLevel(logging.WARNING, "")
	case "error":
		logLevel.SetLevel(logging.ERROR, "")
	case "info":
		logLevel.SetLevel(logging.INFO, "")
	}
	logging.SetBackend(logFormatter)

}
