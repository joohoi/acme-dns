package main

import (
	"database/sql"
	"errors"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"regexp"
	"time"
)

type database struct {
	DB *sql.DB
}

var recordsTable = `
	CREATE TABLE IF NOT EXISTS records(
        Username TEXT UNIQUE NOT NULL PRIMARY KEY,
        Password TEXT UNIQUE NOT NULL,
        Subdomain TEXT UNIQUE NOT NULL,
        Value   TEXT,
        LastActive INT
    );`

// getSQLiteStmt replaces all PostgreSQL prepared statement placeholders (eg. $1, $2) with SQLite variant "?"
func getSQLiteStmt(s string) string {
	re, err := regexp.Compile("\\$[0-9]")
	if err != nil {
		log.Errorf("%v", err)
		return s
	}
	return re.ReplaceAllString(s, "?")
}

func (d *database) Init(engine string, connection string) error {
	db, err := sql.Open(engine, connection)
	if err != nil {
		return err
	}
	d.DB = db
	d.DB.SetMaxOpenConns(1)
	_, err = d.DB.Exec(recordsTable)
	if err != nil {
		return err
	}
	return nil
}

func (d *database) Register() (ACMETxt, error) {
	a, err := newACMETxt()
	if err != nil {
		return ACMETxt{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(a.Password), 10)
	timenow := time.Now().Unix()
	regSQL := `
    INSERT INTO records(
        Username,
        Password,
        Subdomain,
		Value,
        LastActive) 
        values($1, $2, $3, '', $4)`
	if DNSConf.Database.Engine == "sqlite3" {
		regSQL = getSQLiteStmt(regSQL)
	}
	sm, err := d.DB.Prepare(regSQL)
	if err != nil {
		return a, errors.New("SQL error")
	}
	defer sm.Close()
	_, err = sm.Exec(a.Username.String(), passwordHash, a.Subdomain, timenow)
	if err != nil {
		return a, err
	}
	return a, nil
}

func (d *database) GetByUsername(u uuid.UUID) (ACMETxt, error) {
	var results []ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, Value, LastActive
	FROM records
	WHERE Username=$1 LIMIT 1
	`
	if DNSConf.Database.Engine == "sqlite3" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.Prepare(getSQL)
	if err != nil {
		return ACMETxt{}, err
	}
	defer sm.Close()
	rows, err := sm.Query(u.String())
	if err != nil {
		return ACMETxt{}, err
	}
	defer rows.Close()

	// It will only be one row though
	for rows.Next() {
		a := ACMETxt{}
		var uname string
		err = rows.Scan(&uname, &a.Password, &a.Subdomain, &a.Value, &a.LastActive)
		if err != nil {
			return ACMETxt{}, err
		}
		a.Username, err = uuid.FromString(uname)
		if err != nil {
			return ACMETxt{}, err
		}
		results = append(results, a)
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return ACMETxt{}, errors.New("no user")
}

func (d *database) GetByDomain(domain string) ([]ACMETxt, error) {
	domain = sanitizeString(domain)
	log.Debugf("Trying to select domain [%s] from table", domain)
	var a []ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, Value
	FROM records
	WHERE Subdomain=$1 LIMIT 1
	`
	if DNSConf.Database.Engine == "sqlite3" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.Prepare(getSQL)
	if err != nil {
		return a, err
	}
	defer sm.Close()
	rows, err := sm.Query(domain)
	if err != nil {
		return a, err
	}
	defer rows.Close()

	for rows.Next() {
		txt := ACMETxt{}
		err = rows.Scan(&txt.Username, &txt.Password, &txt.Subdomain, &txt.Value)
		if err != nil {
			return a, err
		}
		a = append(a, txt)
	}
	return a, nil
}

func (d *database) Update(a ACMETxt) error {
	// Data in a is already sanitized
	log.Debugf("Trying to update domain [%s] with TXT data [%s]", a.Subdomain, a.Value)
	timenow := time.Now().Unix()
	updSQL := `
	UPDATE records SET Value=$1, LastActive=$2
	WHERE Username=$3 AND Subdomain=$4
	`
	if DNSConf.Database.Engine == "sqlite3" {
		updSQL = getSQLiteStmt(updSQL)
	}

	sm, err := d.DB.Prepare(updSQL)
	if err != nil {
		return err
	}
	defer sm.Close()
	_, err = sm.Exec(a.Value, timenow, a.Username, a.Subdomain)
	if err != nil {
		return err
	}
	return nil
}
