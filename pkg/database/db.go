package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/acme-dns/acme-dns/pkg/acmedns"
	_ "github.com/glebarez/go-sqlite"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type acmednsdb struct {
	DB     *sql.DB
	Mutex  sync.Mutex
	Logger *zap.SugaredLogger
	Config *acmedns.AcmeDnsConfig
}

// DBVersion shows the database version this code uses. This is used for update checks.
var DBVersion = 1

var acmeTable = `
	CREATE TABLE IF NOT EXISTS acmedns(
		Name TEXT,
		Value TEXT
	);`

var userTable = `
	CREATE TABLE IF NOT EXISTS records(
        Username TEXT UNIQUE NOT NULL PRIMARY KEY,
        Password TEXT UNIQUE NOT NULL,
        Subdomain TEXT UNIQUE NOT NULL,
		AllowFrom TEXT
    );`

var txtTable = `
    CREATE TABLE IF NOT EXISTS txt(
		Subdomain TEXT NOT NULL,
		Value   TEXT NOT NULL DEFAULT '',
		LastUpdate INT
	);`

var txtTablePG = `
    CREATE TABLE IF NOT EXISTS txt(
		rowid SERIAL,
		Subdomain TEXT NOT NULL,
		Value   TEXT NOT NULL DEFAULT '',
		LastUpdate INT
	);`

// getSQLiteStmt replaces all PostgreSQL prepared statement placeholders (eg. $1, $2) with SQLite variant "?"
func getSQLiteStmt(s string) string {
	re, _ := regexp.Compile(`\$[0-9]`)
	return re.ReplaceAllString(s, "?")
}

func Init(config *acmedns.AcmeDnsConfig, logger *zap.SugaredLogger) (acmedns.AcmednsDB, error) {
	var d = &acmednsdb{Config: config, Logger: logger}
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	db, err := sql.Open(config.Database.Engine, config.Database.Connection)
	if err != nil {
		return d, err
	}
	d.DB = db
	// Check version first to try to catch old versions without version string
	var versionString string
	_ = d.DB.QueryRow("SELECT Value FROM acmedns WHERE Name='db_version'").Scan(&versionString)
	if versionString == "" {
		versionString = "0"
	}
	_, _ = d.DB.Exec(acmeTable)
	_, _ = d.DB.Exec(userTable)
	if config.Database.Engine == "sqlite" {
		_, _ = d.DB.Exec(txtTable)
	} else {
		_, _ = d.DB.Exec(txtTablePG)
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
	return d, err
}

func (d *acmednsdb) checkDBUpgrades(versionString string) error {
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

func (d *acmednsdb) handleDBUpgrades(version int) error {
	if version == 0 {
		return d.handleDBUpgradeTo1()
	}
	return nil
}

func (d *acmednsdb) handleDBUpgradeTo1() error {
	var err error
	var subdomains []string
	rows, err := d.DB.Query("SELECT Subdomain FROM records")
	if err != nil {
		d.Logger.Errorw("Error in DB upgrade",
			"error", err.Error())
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var subdomain string
		err = rows.Scan(&subdomain)
		if err != nil {
			d.Logger.Errorw("Error in DB upgrade while reading values",
				"error", err.Error())
			return err
		}
		subdomains = append(subdomains, subdomain)
	}
	err = rows.Err()
	if err != nil {
		d.Logger.Errorw("Error in DB upgrade while inserting values",
			"error", err.Error())
		return err
	}
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
	for _, subdomain := range subdomains {
		if subdomain != "" {
			// Insert two rows for each subdomain to txt table
			err = d.NewTXTValuesInTransaction(tx, subdomain)
			if err != nil {
				d.Logger.Errorw("Error in DB upgrade while inserting values",
					"error", err.Error())
				return err
			}
		}
	}
	// SQLite doesn't support dropping columns
	if d.Config.Database.Engine != "sqlite" {
		_, _ = tx.Exec("ALTER TABLE records DROP COLUMN IF EXISTS Value")
		_, _ = tx.Exec("ALTER TABLE records DROP COLUMN IF EXISTS LastActive")
	}
	_, err = tx.Exec("UPDATE acmedns SET Value='1' WHERE Name='db_version'")
	return err
}

// NewTXTValuesInTransaction creates two rows for subdomain to the txt table
func (d *acmednsdb) NewTXTValuesInTransaction(tx *sql.Tx, subdomain string) error {
	var err error
	instr := fmt.Sprintf("INSERT INTO txt (Subdomain, LastUpdate) values('%s', 0)", subdomain)
	_, _ = tx.Exec(instr)
	_, _ = tx.Exec(instr)
	return err
}

func (d *acmednsdb) Register(afrom acmedns.Cidrslice) (acmedns.ACMETxt, error) {
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
	a := acmedns.NewACMETxt()
	a.AllowFrom = acmedns.Cidrslice(afrom.ValidEntries())
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(a.Password), 10)
	regSQL := `
    INSERT INTO records(
        Username,
        Password,
        Subdomain,
		AllowFrom) 
        values($1, $2, $3, $4)`
	if d.Config.Database.Engine == "sqlite" {
		regSQL = getSQLiteStmt(regSQL)
	}
	sm, err := tx.Prepare(regSQL)
	if err != nil {
		d.Logger.Errorw("Database error in prepare",
			"error", err.Error())
		return a, errors.New("SQL error")
	}
	defer sm.Close()
	_, err = sm.Exec(a.Username.String(), passwordHash, a.Subdomain, a.AllowFrom.JSON())
	if err == nil {
		err = d.NewTXTValuesInTransaction(tx, a.Subdomain)
	}
	return a, err
}

func (d *acmednsdb) GetByUsername(u uuid.UUID) (acmedns.ACMETxt, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var results []acmedns.ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, AllowFrom
	FROM records
	WHERE Username=$1 LIMIT 1
	`
	if d.Config.Database.Engine == "sqlite" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.Prepare(getSQL)
	if err != nil {
		return acmedns.ACMETxt{}, err
	}
	defer sm.Close()
	rows, err := sm.Query(u.String())
	if err != nil {
		return acmedns.ACMETxt{}, err
	}
	defer rows.Close()

	// It will only be one row though
	for rows.Next() {
		txt, err := d.getModelFromRow(rows)
		if err != nil {
			return acmedns.ACMETxt{}, err
		}
		results = append(results, txt)
	}
	if len(results) > 0 {
		return results[0], nil
	}
	return acmedns.ACMETxt{}, errors.New("no user")
}

func (d *acmednsdb) GetTXTForDomain(domain string) ([]string, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	domain = acmedns.SanitizeString(domain)
	var txts []string
	getSQL := `
	SELECT Value FROM txt WHERE Subdomain=$1 LIMIT 2
	`
	if d.Config.Database.Engine == "sqlite" {
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

func (d *acmednsdb) Update(a acmedns.ACMETxtPost) error {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	var err error
	// Data in a is already sanitized
	timenow := time.Now().Unix()

	updSQL := `
	UPDATE txt SET Value=$1, LastUpdate=$2
	WHERE rowid=(
		SELECT rowid FROM txt WHERE Subdomain=$3 ORDER BY LastUpdate LIMIT 1)
	`
	if d.Config.Database.Engine == "sqlite" {
		updSQL = getSQLiteStmt(updSQL)
	}

	sm, err := d.DB.Prepare(updSQL)
	if err != nil {
		return err
	}
	defer sm.Close()
	_, err = sm.Exec(a.Value, timenow, a.Subdomain)
	if err != nil {
		return err
	}
	return nil
}

func (d *acmednsdb) getModelFromRow(r *sql.Rows) (acmedns.ACMETxt, error) {
	txt := acmedns.ACMETxt{}
	afrom := ""
	err := r.Scan(
		&txt.Username,
		&txt.Password,
		&txt.Subdomain,
		&afrom)
	if err != nil {
		d.Logger.Errorw("Row scan error",
			"error", err.Error())
	}

	cslice := acmedns.Cidrslice{}
	err = json.Unmarshal([]byte(afrom), &cslice)
	if err != nil {
		d.Logger.Errorw("JSON unmarshall error",
			"error", err.Error())
	}
	txt.AllowFrom = cslice
	return txt, err
}

func (d *acmednsdb) Close() {
	d.DB.Close()
}

func (d *acmednsdb) GetBackend() *sql.DB {
	return d.DB
}

func (d *acmednsdb) SetBackend(backend *sql.DB) {
	d.DB = backend
}
