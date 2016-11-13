package main

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

type Database struct {
	DB *sql.DB
}

var records_table string = `
	CREATE TABLE IF NOT EXISTS records(
        Username TEXT UNIQUE NOT NULL PRIMARY KEY,
        Password TEXT UNIQUE NOT NULL,
        Subdomain TEXT UNIQUE NOT NULL,
        Value   TEXT,
        LastActive DATETIME
    );`

func (d *Database) Init(filename string) error {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return err
	}
	d.DB = db
	_, err = d.DB.Exec(records_table)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) Register() (ACMETxt, error) {
	a, err := NewACMETxt()
	if err != nil {
		return ACMETxt{}, err
	}
	password_hash, err := bcrypt.GenerateFromPassword([]byte(a.Password), 10)
	reg_sql := `
    INSERT INTO records(
        Username,
        Password,
        Subdomain,
        Value,
        LastActive)
        values(?, ?, ?, ?, CURRENT_TIMESTAMP)`
	sm, err := d.DB.Prepare(reg_sql)
	if err != nil {
		return a, err
	}
	defer sm.Close()
	_, err = sm.Exec(a.Username, password_hash, a.Subdomain, a.Value)
	if err != nil {
		return a, err
	}
	return a, nil
}

func (d *Database) GetByUsername(u uuid.UUID) (ACMETxt, error) {
	var results []ACMETxt
	get_sql := `
	SELECT Username, Password, Subdomain, Value, LastActive
	FROM records
	WHERE Username=? LIMIT 1
	`
	sm, err := d.DB.Prepare(get_sql)
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
		var a ACMETxt = ACMETxt{}
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
	} else {
		return ACMETxt{}, errors.New("no user")
	}
}

func (d *Database) GetByDomain(domain string) ([]ACMETxt, error) {
	domain = SanitizeString(domain)
	log.Debugf("Trying to select domain [%s] from table", domain)
	var a []ACMETxt
	get_sql := `
	SELECT Username, Password, Subdomain, Value
	FROM records
	WHERE Subdomain=? LIMIT 1
	`
	sm, err := d.DB.Prepare(get_sql)
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

func (d *Database) Update(a ACMETxt) error {
	// Data in a is already sanitized
	log.Debugf("Trying to update domain [%s] with TXT data [%s]", a.Subdomain, a.Value)
	upd_sql := `
	UPDATE records SET Value=?
	WHERE Username=? AND Subdomain=?
	`
	sm, err := d.DB.Prepare(upd_sql)
	if err != nil {
		return err
	}
	defer sm.Close()
	_, err = sm.Exec(a.Value, a.Username, a.Subdomain)
	if err != nil {
		return err
	}
	return nil
}
