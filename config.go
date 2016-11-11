package main

import (
	"errors"
	"github.com/BurntSushi/toml"
)

func ReadConfig(fname string) (DnsConfig, error) {
	var conf DnsConfig
	if _, err := toml.DecodeFile(fname, &conf); err != nil {
		return DnsConfig{}, errors.New("Malformed configuration file")
	}
	return conf, nil
}
