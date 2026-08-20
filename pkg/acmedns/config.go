package acmedns

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	ApiTlsProviderNone               = "none"
	ApiTlsProviderLetsEncrypt        = "letsencrypt"
	ApiTlsProviderLetsEncryptStaging = "letsencryptstaging"
	ApiTlsProviderCert               = "cert"
)

// validProtocols are the accepted values for the general.protocol config option
var validProtocols = []string{"both", "both4", "both6", "udp", "udp4", "udp6", "tcp", "tcp4", "tcp6"}

func FileIsAccessible(fname string) bool {
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

func readTomlConfig(fname string) (AcmeDnsConfig, error) {
	var conf AcmeDnsConfig
	_, err := toml.DecodeFile(fname, &conf)
	if err != nil {
		// Return with config file parsing errors from toml package
		return conf, err
	}
	return prepareConfig(conf)
}

// prepareConfig checks that mandatory values exist, and can be used to set default values in the future
func prepareConfig(conf AcmeDnsConfig) (AcmeDnsConfig, error) {
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
	if conf.General.Proto == "" {
		conf.General.Proto = "both"
	}

	if !slices.Contains(validProtocols, conf.General.Proto) {
		return conf, fmt.Errorf("invalid value for general.protocol, expected one of %s", strings.Join(validProtocols, ", "))
	}

	switch conf.API.TLS {
	case ApiTlsProviderCert, ApiTlsProviderLetsEncrypt, ApiTlsProviderLetsEncryptStaging, ApiTlsProviderNone:
		// we have a good value
	default:
		return conf, fmt.Errorf("invalid value for api.tls, expected one of [%s, %s, %s, %s]", ApiTlsProviderCert, ApiTlsProviderLetsEncrypt, ApiTlsProviderLetsEncryptStaging, ApiTlsProviderNone)
	}

	return conf, nil
}

func ReadConfig(configFile, fallback string) (AcmeDnsConfig, string, error) {
	var usedConfigFile string
	var config AcmeDnsConfig
	var err error
	if FileIsAccessible(configFile) {
		usedConfigFile = configFile
		config, err = readTomlConfig(configFile)
	} else if FileIsAccessible(fallback) {
		usedConfigFile = fallback
		config, err = readTomlConfig(fallback)
	} else {
		err = fmt.Errorf("configuration file not found")
	}
	if err != nil {
		err = fmt.Errorf("encountered an error while trying to read configuration file:  %w", err)
	}
	return config, usedConfigFile, err
}
