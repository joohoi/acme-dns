package main

import (
	"database/sql"
	//"encoding/json"
	//"github.com/boltdb/bolt"
	_ "github.com/mattn/go-sqlite3"
	//"strings"
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
	a := NewACMETxt()
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
	_, err = sm.Exec(a.Username, a.Password, a.Subdomain, a.Value)
	if err != nil {
		return a, err
	}
	// Do an insert check
	/*
		id, err := status.LastInsertId()
		if err != nil {
			return a, err
		}*/

	return a, nil
}

func (d *Database) GetByUsername(u string) ([]ACMETxt, error) {
	u = NormalizeString(u, 36)
	log.Debugf("Trying to select by user [%s] from table", u)
	var results []ACMETxt
	get_sql := `
	SELECT Username, Password, Subdomain, Value
	FROM records
	WHERE Username=? LIMIT 1
	`
	sm, err := d.DB.Prepare(get_sql)
	if err != nil {
		return nil, err
	}
	defer sm.Close()
	rows, err := sm.Query(u)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// It will only be one row though
	for rows.Next() {
		var a ACMETxt = ACMETxt{}
		err = rows.Scan(&a.Username, &a.Password, &a.Subdomain, &a.Value)
		if err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, nil
}

func (d *Database) GetByDomain(domain string) ([]ACMETxt, error) {
	domain = NormalizeString(domain, 36)
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

/*
func addTXT(txt ACMETxt) error {

	err := db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("domains"))
		if err != nil {
			return err
		}
		jtxt, err := json.Marshal(txt)
		if err != nil {
			return err
		}

		// put returns nil if successful, nil return commits db.Update
		return bucket.Put([]byte(strings.ToLower(txt.Domain)), jtxt)
	})
	return err

}

func getTXT(domain string) (ACMETxt, error) {
	var atxt ACMETxt
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("domains"))
		value := bucket.Get([]byte(strings.ToLower(domain)))
		if len(value) == 0 {
			// Not found
			log.Debugf("Record for [%s] not found", domain)
			atxt = ACMETxt{}
		} else {
			if err := json.Unmarshal(value, &atxt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ACMETxt{}, err
	}
	return atxt, err
}
*/
