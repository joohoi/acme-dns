package main

import (
	"flag"
	"testing"
)

var (
	postgres = flag.Bool("postgres", false, "run integration tests against PostgreSQL")
)

func TestRegister(t *testing.T) {
	flag.Parse()
	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			t.Errorf("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			return
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}
	defer DB.DB.Close()

	// Register tests
	_, err := DB.Register()
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}
}

func TestGetByUsername(t *testing.T) {
	flag.Parse()
	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			t.Errorf("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			return
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}
	defer DB.DB.Close()

	// Create  reg to refer to
	reg, err := DB.Register()
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	regUser, err := DB.GetByUsername(reg.Username)
	if err != nil {
		t.Errorf("Could not get test user, got error [%v]", err)
	}

	if reg.Username != regUser.Username {
		t.Errorf("GetByUsername username [%q] did not match the original [%q]", regUser.Username, reg.Username)
	}

	if reg.Subdomain != regUser.Subdomain {
		t.Errorf("GetByUsername subdomain [%q] did not match the original [%q]", regUser.Subdomain, reg.Subdomain)
	}

	// regUser password already is a bcrypt hash
	if !correctPassword(reg.Password, regUser.Password) {
		t.Errorf("The password [%s] does not match the hash [%s]", reg.Password, regUser.Password)
	}
}

func TestGetByDomain(t *testing.T) {
	flag.Parse()
	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			t.Errorf("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			return
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}
	defer DB.DB.Close()

	var regDomain = ACMETxt{}

	// Create  reg to refer to
	reg, err := DB.Register()
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	regDomainSlice, err := DB.GetByDomain(reg.Subdomain)
	if err != nil {
		t.Errorf("Could not get test user, got error [%v]", err)
	}
	if len(regDomainSlice) == 0 {
		t.Errorf("No rows returned for GetByDomain [%s]", reg.Subdomain)
	} else {
		regDomain = regDomainSlice[0]
	}

	if reg.Username != regDomain.Username {
		t.Errorf("GetByUsername username [%q] did not match the original [%q]", regDomain.Username, reg.Username)
	}

	if reg.Subdomain != regDomain.Subdomain {
		t.Errorf("GetByUsername subdomain [%q] did not match the original [%q]", regDomain.Subdomain, reg.Subdomain)
	}

	// regDomain password already is a bcrypt hash
	if !correctPassword(reg.Password, regDomain.Password) {
		t.Errorf("The password [%s] does not match the hash [%s]", reg.Password, regDomain.Password)
	}

	// Not found
	regNotfound, _ := DB.GetByDomain("does-not-exist")
	if len(regNotfound) > 0 {
		t.Errorf("No records should be returned.")
	}
}

func TestUpdate(t *testing.T) {
	flag.Parse()
	if *postgres {
		DNSConf.Database.Engine = "postgres"
		err := DB.Init("postgres", "postgres://acmedns:acmedns@localhost/acmedns")
		if err != nil {
			t.Errorf("PostgreSQL integration tests expect database \"acmedns\" running in localhost, with username and password set to \"acmedns\"")
			return
		}
	} else {
		DNSConf.Database.Engine = "sqlite3"
		_ = DB.Init("sqlite3", ":memory:")
	}
	defer DB.DB.Close()

	// Create  reg to refer to
	reg, err := DB.Register()
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	regUser, err := DB.GetByUsername(reg.Username)
	if err != nil {
		t.Errorf("Could not get test user, got error [%v]", err)
	}

	// Set new values (only TXT should be updated) (matches by username and subdomain)

	validTXT := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	regUser.Password = "nevergonnagiveyouup"
	regUser.Value = validTXT

	err = DB.Update(regUser)
	if err != nil {
		t.Errorf("DB Update failed, got error: [%v]", err)
	}

	updUser, err := DB.GetByUsername(regUser.Username)
	if err != nil {
		t.Errorf("GetByUsername threw error [%v]", err)
	}
	if updUser.Value != validTXT {
		t.Errorf("Update failed, fetched value [%s] does not match the update value [%s]", updUser.Value, validTXT)
	}
}
