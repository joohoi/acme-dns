package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"
	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/joohoi/acme-dns/pkg/acmedns"
)

type acmednsdb struct {
	DB     *sql.DB
	Mutex  sync.RWMutex
	Logger *zap.SugaredLogger
	Config *acmedns.AcmeDnsConfig
}

// DBVersion shows the database version this code uses. This is used for update checks.
var DBVersion = 1

// dbDefaultTimeout bounds every database operation. Without it, a single stalled
// query (e.g. a Postgres connection silently dropped by a stateful firewall, or a
// locked SQLite file) blocks forever while holding the shared lock, freezing the
// DNS hot path and leaking a goroutine per query until the process is OOM-killed.
const dbDefaultTimeout = 20 * time.Second

// dbReadTimeout bounds the DNS hot-path read (GetTXTForDomain). It is deliberately
// much shorter than dbDefaultTimeout: a DNS resolver abandons the query within a
// few seconds, so a 20s ceiling only lets a stalled connection pin a worker long
// after the client has given up. Failing fast frees the connection and prevents
// the goroutine pileup that bounded-but-still-slow reads would otherwise cause.
const dbReadTimeout = 5 * time.Second

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

// txtIndex backs the DNS hot-path lookup (SELECT ... FROM txt WHERE Subdomain=$1).
// Without it every DNS query is a full table scan whose latency grows with the
// number of registered accounts, eventually crossing the read timeout under load.
// CREATE INDEX IF NOT EXISTS is idempotent and valid for both SQLite and
// PostgreSQL, so existing deployments gain the index on the next startup with no
// explicit migration step.
var txtIndex = `
	CREATE INDEX IF NOT EXISTS idx_txt_subdomain ON txt (Subdomain);`

// getSQLiteStmt replaces all PostgreSQL prepared statement placeholders (eg. $1, $2) with SQLite variant "?"
func getSQLiteStmt(s string) string {
	re, _ := regexp.Compile(`\$[0-9]`)
	return re.ReplaceAllString(s, "?")
}

func Init(config *acmedns.AcmeDnsConfig, logger *zap.SugaredLogger) (acmedns.AcmednsDB, error) {
	var d = &acmednsdb{Config: config, Logger: logger}
	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	if config.Database.Engine == "sqlite3" {
		logger.Warn("DB: sqlite3 engine has been replaced by the sqlite engine. Please update your config")
		config.Database.Engine = "sqlite"
	}

	db, err := sql.Open(config.Database.Engine, config.Database.Connection)
	if err != nil {
		return d, err
	}
	d.DB = db
	// Bound the connection pool. SQLite (file or :memory:) is not safe for
	// concurrent writers, so pin it to a single connection — this serializes
	// access at the pool level, which is the job the global mutex used to do.
	// For other engines, recycle connections so a silently-dropped TCP
	// connection is never reused for a query that would then block forever.
	if config.Database.Engine == "sqlite" {
		d.DB.SetMaxOpenConns(1)
	} else {
		d.DB.SetMaxOpenConns(10)
		d.DB.SetMaxIdleConns(2)
		d.DB.SetConnMaxLifetime(5 * time.Minute)
		// Recycle idle connections quickly so a silently-dropped TCP connection is
		// retired within a minute instead of lingering until ConnMaxLifetime and
		// stalling the next query that borrows it.
		d.DB.SetConnMaxIdleTime(1 * time.Minute)
	}
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
	_, _ = d.DB.Exec(txtIndex)
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
	instr := fmt.Sprintf("INSERT INTO txt (Subdomain, LastUpdate) values('%s', 0)", subdomain)
	for i := 0; i < d.Config.General.Txtlimit; i++ {
		_, err := tx.Exec(instr)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *acmednsdb) Register(afrom acmedns.Cidrslice) (acmedns.ACMETxt, error) {
	d.Mutex.Lock()
	defer d.Mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), dbDefaultTimeout)
	defer cancel()
	var err error
	tx, err := d.DB.BeginTx(ctx, nil)
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
	sm, err := tx.PrepareContext(ctx, regSQL)
	if err != nil {
		d.Logger.Errorw("Database error in prepare",
			"error", err.Error())
		return a, fmt.Errorf("failed to prepare registration statement: %w", err)
	}
	defer sm.Close()
	_, err = sm.ExecContext(ctx, a.Username.String(), passwordHash, a.Subdomain, a.AllowFrom.JSON())
	if err == nil {
		err = d.NewTXTValuesInTransaction(tx, a.Subdomain)
	}
	return a, err
}

func (d *acmednsdb) GetByUsername(u uuid.UUID) (acmedns.ACMETxt, error) {
	d.Mutex.RLock()
	defer d.Mutex.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), dbDefaultTimeout)
	defer cancel()
	var results []acmedns.ACMETxt
	getSQL := `
	SELECT Username, Password, Subdomain, AllowFrom
	FROM records
	WHERE Username=$1 LIMIT 1
	`
	if d.Config.Database.Engine == "sqlite" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.PrepareContext(ctx, getSQL)
	if err != nil {
		return acmedns.ACMETxt{}, err
	}
	defer sm.Close()
	rows, err := sm.QueryContext(ctx, u.String())
	if err != nil {
		return acmedns.ACMETxt{}, fmt.Errorf("failed to query user: %w", err)
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
	return acmedns.ACMETxt{}, fmt.Errorf("user not found: %s", u.String())
}

func (d *acmednsdb) GetTXTForDomain(domain string) ([]string, error) {
	d.Mutex.RLock()
	defer d.Mutex.RUnlock()
	ctx, cancel := context.WithTimeout(context.Background(), dbReadTimeout)
	defer cancel()
	domain = acmedns.SanitizeString(domain)
	var txts []string

	getSQL := fmt.Sprintf(`
			SELECT Value FROM txt WHERE Subdomain=$1 LIMIT %d
	`, d.Config.General.Txtlimit)

	if d.Config.Database.Engine == "sqlite" {
		getSQL = getSQLiteStmt(getSQL)
	}

	sm, err := d.DB.PrepareContext(ctx, getSQL)
	if err != nil {
		return txts, err
	}
	defer sm.Close()
	rows, err := sm.QueryContext(ctx, domain)
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
	ctx, cancel := context.WithTimeout(context.Background(), dbDefaultTimeout)
	defer cancel()
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

	sm, err := d.DB.PrepareContext(ctx, updSQL)
	if err != nil {
		return err
	}
	defer sm.Close()
	_, err = sm.ExecContext(ctx, a.Value, timenow, a.Subdomain)
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
