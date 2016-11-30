package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	log "github.com/Sirupsen/logrus"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

var recordsTable = `
	CREATE TABLE IF NOT EXISTS records(
        Username TEXT UNIQUE NOT NULL PRIMARY KEY,
        Password TEXT UNIQUE NOT NULL,
        Subdomain TEXT UNIQUE NOT NULL,
        Value   TEXT,
        LastActive INT,
		AllowFrom TEXT
    );`

// getSQLiteStmt replaces all PostgreSQL prepared statement placeholders (eg. $1, $2) with SQLite variant "?"
func getSQLiteStmt(s string) string {
	re, _ := regexp.Compile("\\$[0-9]")
	return re.ReplaceAllString(s, "?")
}

func (d *acmedb) Init(engine string, connection string) error {
	d.Lock()
	defer d.Unlock()
	db, err := sql.Open(engine, connection)
	if err != nil {
		return err
	}
	d.DB = db
	//d.DB.SetMaxOpenConns(1)
	_, err = d.DB.Exec(recordsTable)
	if err != nil {
		return err
	}
	return nil
}

func (d *acmedb) Register(afrom cidrslice) (ACMETxt, error) {
	d.Lock()
	defer d.Unlock()
	a := newACMETxt()
	a.AllowFrom = cidrslice(afrom.ValidEntries())
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(a.Password), 10)
	timenow := time.Now().Unix()
	regSQL := `
    INSERT INTO records(
        Username,
        Password,
        Subdomain,
		Value,
        LastActive,
		AllowFrom) 
        values($1, $2, $3, '', $4, $5)`
	if DNSConf.Database.Engine == "sqlite3" {
		regSQL = getSQLiteStmt(regSQL)
	}
	sm, err := d.DB.Prepare(regSQL)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Database error in prepare")
		return a, errors.New("SQL error")
	}
	defer sm.Close()
	_, err = sm.Exec(a.Username.String(), passwordHash, a.Subdomain, timenow, a.AllowFrom.JSON())
	if err != nil {
		return a, err
	}
	return a, nil
}

func (d *acmedb) GetByUsername(u uuid.UUID) (ACMETxt, error) {
	d.Lock()
	defer d.Unlock()
	var results []ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, Value, LastActive, AllowFrom
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
		txt, err := getModelFromRow(rows)
		if err != nil {
			return ACMETxt{}, err
		}
		results = append(results, txt)
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return ACMETxt{}, errors.New("no user")
}

func (d *acmedb) GetByDomain(domain string) ([]ACMETxt, error) {
	d.Lock()
	defer d.Unlock()
	domain = sanitizeString(domain)
	var a []ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, Value, LastActive, AllowFrom
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
		txt, err := getModelFromRow(rows)
		if err != nil {
			return a, err
		}
		a = append(a, txt)
	}
	return a, nil
}

func (d *acmedb) Update(a ACMETxt) error {
	d.Lock()
	defer d.Unlock()
	// Data in a is already sanitized
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

func getModelFromRow(r *sql.Rows) (ACMETxt, error) {
	txt := ACMETxt{}
	afrom := ""
	err := r.Scan(
		&txt.Username,
		&txt.Password,
		&txt.Subdomain,
		&txt.Value,
		&txt.LastActive,
		&afrom)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Row scan error")
	}

	cslice := cidrslice{}
	err = json.Unmarshal([]byte(afrom), &cslice)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("JSON unmarshall error")
	}
	txt.AllowFrom = cslice
	return txt, err
}

func (d *acmedb) Close() {
	d.DB.Close()
}

func (d *acmedb) GetBackend() *sql.DB {
	return d.DB
}

func (d *acmedb) SetBackend(backend *sql.DB) {
	d.DB = backend
}
