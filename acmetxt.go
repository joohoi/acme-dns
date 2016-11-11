package main

import (
	"github.com/satori/go.uuid"
	"time"
)

// The default database object
type ACMETxt struct {
	Username string `json:"username"`
	Password string `json:"password"`
	ACMETxtPost
	LastActive time.Time
}

type ACMETxtPost struct {
	Subdomain string `json:"subdomain"`
	Value     string `json:"txt"`
}

func NewACMETxt() ACMETxt {
	var a ACMETxt = ACMETxt{}
	a.Username = uuid.NewV4().String()
	a.Password = uuid.NewV4().String()
	a.Subdomain = uuid.NewV4().String()
	return a
}
