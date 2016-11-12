package main

import (
	"regexp"
	"unicode/utf8"
)

func NormalizeString(s string, length int) string {
	var ret string
	re, err := regexp.Compile("[^A-Za-z\\-0-9]+")
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	ret = re.ReplaceAllString(s, "")
	if utf8.RuneCountInString(ret) > length {
		ret = ret[0:length]
	}
	return ret
}
