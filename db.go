package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// DBVersion shows the database version this code uses. This is used for update checks.
var DBVersion = 2

var acmeTable = `
	CREATE TABLE IF NOT EXISTS acmedns(
		Name TEXT,
		Value TEXT
	);`

var recordsTable = `
	CREATE TABLE IF NOT EXISTS records(
        Username TEXT UNIQUE NOT NULL PRIMARY KEY,
        Password TEXT UNIQUE NOT NULL,
        Subdomain TEXT UNIQUE NOT NULL,
	AllowFrom TEXT,
	LastUsed, INT
    );`

var txtTable = `
    CREATE TABLE IF NOT EXISTS txt(
		Subdomain TEXT NOT NULL,
		Value   TEXT NOT NULL DEFAULT '',
	        InsertDate, INT
	);`

// getSQLiteStmt replaces all PostgreSQL prepared statement placeholders (eg. $1, $2) with SQLite variant "?"
func getSQLiteStmt(s string) string {
	re, _ := regexp.Compile(`\$[0-9]`)
	return re.ReplaceAllString(s, "?")
}

func (d *acmedb) Init(engine string, connection string) error {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	db, err := sql.Open(engine, connection)
	if err != nil {
		return err
	}
	d.DB = db
	// create tables if they don't exist
	_, err = d.DB.Exec("SELECT COUNT(*) from txt")
	if err != nil && (err.Error() == `no such table: txt` || err.Error() == `relation "txt" does not exist`) {
		log.Info("Creating tables")
		_, _ = d.DB.Exec(acmeTable)
		_, _ = d.DB.Exec(recordsTable)
		_, _ = d.DB.Exec(txtTable)
		insversion := fmt.Sprintf("INSERT INTO acmedns (Name, Value) values('db_version', '%d')", DBVersion)
		_, err = db.Exec(insversion)
	}

	var versionString string
	_ = d.DB.QueryRow("SELECT Value FROM acmedns WHERE Name='db_version'").Scan(&versionString)
	if versionString == "" {
		versionString = "0"
	}
	// If everything is fine, handle db upgrade tasks
	if err == nil {
		err = d.checkDBUpgrades(versionString)
	}
	if err == nil {
		if versionString == "0" {
			// No errors so we should now be in version 1
			insversion := fmt.Sprintf("INSERT INTO acmedns (Name, Value) values('db_version', '%d')", DBVersion)
			_, err = db.Exec(insversion)
		}
	}
	return err
}

func (d *acmedb) checkDBUpgrades(versionString string) error {
	var err error
	version, err := strconv.Atoi(versionString)
	if err != nil {
		return err
	}
	if version != DBVersion {
		return d.handleDBUpgrades(version)
	}
	return nil

}

func (d *acmedb) handleDBUpgrades(version int) error {
	var err error
	if version < 1 {
		err = d.handleDBUpgradeTo1()
		if err != nil {
			return err
		}
	}
	if version < 2 {
		return d.handleDBUpgradeTo2()
	}
	return nil
}

func (d *acmedb) handleDBUpgradeTo1() error {
	var err error
	log.Info("Upgrading db to version 1")
	tx, err := d.DB.Begin()
	// Rollback if errored, commit if not
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()
	_, _ = tx.Exec("DELETE FROM txt")
	// SQLite doesn't support dropping columns
	if Config.Database.Engine != "sqlite3" {
		_, _ = tx.Exec("ALTER TABLE records DROP COLUMN IF EXISTS Value")
		_, _ = tx.Exec("ALTER TABLE records DROP COLUMN IF EXISTS LastActive")
	}
	_, err = tx.Exec("UPDATE acmedns SET Value='1' WHERE Name='db_version'")
	return err
}
func (d *acmedb) handleDBUpgradeTo2() error {
	var err error
	log.Info("Upgrading db to version 2")
	tx, err := d.DB.Begin()
	// Rollback if errored, commit if not
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()
	_, _ = tx.Exec("ALTER TABLE records ADD COLUMN LastUsed INT");
	if err != nil {
		return err
	}
	_, err = tx.Exec("ALTER TABLE txt RENAME COLUMN LastUpdate to InsertDate")
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE acmedns SET Value='2' WHERE Name='db_version'")
	return err
}

func (d *acmedb) Register(afrom cidrslice) (ACMETxt, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var err error
	tx, err := d.DB.Begin()
	// Rollback if errored, commit if not
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()
	a := newACMETxt()
	a.AllowFrom = cidrslice(afrom.ValidEntries())
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(a.Password), 10)
	regSQL := `
    INSERT INTO records(
        Username,
        Password,
        Subdomain,
		AllowFrom) 
        values($1, $2, $3, $4)`
	if Config.Database.Engine == "sqlite3" {
		regSQL = getSQLiteStmt(regSQL)
	}
	sm, err := tx.Prepare(regSQL)
	if err != nil {
		log.WithFields(log.Fields{"error": err.Error()}).Error("Database error in prepare")
		return a, errors.New("SQL error")
	}
	defer sm.Close()
	_, err = sm.Exec(a.Username.String(), passwordHash, a.Subdomain, a.AllowFrom.JSON())
	return a, err
}

func (d *acmedb) GetByUsername(u uuid.UUID) (ACMETxt, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var results []ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, AllowFrom
	FROM records
	WHERE Username=$1 LIMIT 1
	`
	if Config.Database.Engine == "sqlite3" {
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

func (d *acmedb) GetTXTForDomain(domain string) ([]string, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	domain = sanitizeString(domain)
	var txts []string
	// limit records, because a acme-dns cert may not contain more than 100 domains
	getSQL := `
	SELECT Value FROM txt WHERE Subdomain=$1 ORDER BY InsertDate DESC LIMIT 100
	`
	if Config.Database.Engine == "sqlite3" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.Prepare(getSQL)
	if err != nil {
		return txts, err
	}
	defer sm.Close()
	rows, err := sm.Query(domain)
	if err != nil {
		return txts, err
	}
	defer rows.Close()

	for rows.Next() {
		var rtxt string
		err = rows.Scan(&rtxt)
		if err != nil {
			return txts, err
		}
		txts = append(txts, rtxt)
	}
	return txts, nil
}

func (d *acmedb) Update(a ACMETxtPost) error {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var err error
	// Data in a is already sanitized
	timenow := time.Now().Unix()

	updSQL := `INSERT INTO txt (Subdomain, Value, InsertDate) values($1, $2, $3)`
	if Config.Database.Engine == "sqlite3" {
		updSQL = getSQLiteStmt(updSQL)
	}

	updStmt, err := d.DB.Prepare(updSQL)
	if err != nil {
		return err
	}
	defer updStmt.Close()
	_, err = updStmt.Exec(a.Subdomain, a.Value, timenow)
	if err != nil {
		return err
	}

	lastUsedSQL := `UPDATE records SET LastUsed = $1 WHERE Subdomain = $2`
	lastUsedStmt, err := d.DB.Prepare(lastUsedSQL)
	if err != nil {
		return err
	}
	defer lastUsedStmt.Close()
	if err != nil {
		return err
	}
	_, err = lastUsedStmt.Exec(timenow, a.Subdomain)
	return err
}

func (d *acmedb) Delete(a ACMETxtPost) error {
	d.Lock()
	defer d.Unlock()
	var err error
	// Data in a is already sanitized

	delSQL := `DELETE FROM txt WHERE Subdomain=$1 AND Value=$2`
	if Config.Database.Engine == "sqlite3" {
		delSQL = getSQLiteStmt(delSQL)
	}

	sm, err := d.DB.Prepare(delSQL)
	if err != nil {
		return err
	}
	defer sm.Close()
	_, err = sm.Exec(a.Subdomain, a.Value)
	return err
}

func getModelFromRow(r *sql.Rows) (ACMETxt, error) {
	txt := ACMETxt{}
	afrom := ""
	err := r.Scan(
		&txt.Username,
		&txt.Password,
		&txt.Subdomain,
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
