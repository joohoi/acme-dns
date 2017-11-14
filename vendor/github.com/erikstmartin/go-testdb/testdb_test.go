package testdb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"testing"
)

func TestSetOpenFunc(t *testing.T) {
	defer Reset()

	SetOpenFunc(func(dsn string) (driver.Conn, error) {
		return Conn(), errors.New("test error")
	})

	// err only returns from this if it's an unknown driver, we are stubbing opening a connection
	db, _ := sql.Open("testdb", "foo")
	conn, err := db.Driver().Open("foo")

	if db == nil {
		t.Fatal("driver.Open not properly set: db was nil")
	}

	if conn == nil {
		t.Fatal("driver.Open not properly set: didn't connection")
	}

	if err.Error() != "test error" {
		t.Fatal("driver.Open not properly set: err was not returned properly")
	}
}

func TestStubQuery(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from foo"
	columns := []string{"count"}
	result := `
  5
  `
	StubQuery(sql, RowsFromCSVString(columns, result))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	if res.Next() {
		var count int64
		err = res.Scan(&count)

		if err != nil {
			t.Fatal(err)
		}

		if count != 5 {
			t.Fatal("failed to return count")
		}
	}
}

func TestStubQueryAdditionalWhitespace(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sqlWhitespace := "select count(*) from              foo"
	sql := "select count(*) from foo"
	columns := []string{"count"}
	result := `
  5
  `
	StubQuery(sqlWhitespace, RowsFromCSVString(columns, result))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	if res.Next() {
		var count int64
		err = res.Scan(&count)

		if err != nil {
			t.Fatal(err)
		}

		if count != 5 {
			t.Fatal("failed to return count")
		}
	}
}

func TestStubQueryChangeCase(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sqlCase := "SELECT COUNT(*) FROM foo"
	sql := "select count(*) from foo"
	columns := []string{"count"}
	result := `
  5
  `
	StubQuery(sqlCase, RowsFromCSVString(columns, result))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	if res.Next() {
		var count int64
		err = res.Scan(&count)

		if err != nil {
			t.Fatal(err)
		}

		if count != 5 {
			t.Fatal("failed to return count")
		}
	}
}

func TestUnknownQuery(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from foobar"
	_, err := db.Query(sql)

	if err == nil {
		t.Fatal("Unknown queries should fail")
	}
}

func TestStubQueryError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from error"

	StubQueryError(sql, errors.New("test error"))

	res, err := db.Query(sql)

	if err == nil {
		t.Fatal("failed to return error from stubbed query")
	}

	if res != nil {
		t.Fatal("result should be nil on error")
	}
}

func TestStubQueryRowError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from error"

	StubQueryError(sql, errors.New("test error"))

	row := db.QueryRow(sql)
	var count int64
	err := row.Scan(&count)

	if err == nil {
		t.Fatal("error not returned")
	}
}

func TestStubQueryMultipleResult(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select id, name, age from users"
	columns := []string{"id", "name", "age", "created"}
	result := `
  1,tim,20,2012-10-01 01:00:01
  2,joe,25,2012-10-02 02:00:02
  3,bob,30,2012-10-03 03:00:03
  `
	StubQuery(sql, RowsFromCSVString(columns, result))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	i := 0

	for res.Next() {
		var u = user{}
		err = res.Scan(&u.id, &u.name, &u.age, &u.created)

		if err != nil {
			t.Fatal(err)
		}

		if u.id == 0 || u.name == "" || u.age == 0 || u.created == "" {
			t.Fatal("failed to populate object with result")
		}
		i++
	}

	if i != 3 {
		t.Fatal("failed to return proper number of results")
	}
}

func TestStubQueryMultipleResultWithCustomComma(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select id, name, age from users"
	columns := []string{"id", "name", "age", "data", "created"}
	result := `
  1|tim|20|part_1,part_2,part_3|2014-10-16 15:01:00
  2|joe|25|part_4,part_5,part_6|2014-10-17 15:01:01
  3|bob|30|part_7,part_8,part_9|2014-10-18 15:01:02
  `
	StubQuery(sql, RowsFromCSVString(columns, result, '|'))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	i := 0

	for res.Next() {
		var u = user{}
		err = res.Scan(&u.id, &u.name, &u.age, &u.data, &u.created)

		if err != nil {
			t.Fatal(err)
		}

		if u.id == 0 || u.name == "" || u.age == 0 || u.data == "" || u.created == "" {
			t.Fatal("failed to populate object with result")
		}
		i++
	}

	if i != 3 {
		t.Fatal("failed to return proper number of results")
	}
}

func TestStubQueryMultipleResultNewline(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select id, name, age from users"
	columns := []string{"id", "name", "age", "created"}
	result := "1,tim,20,2012-10-01 01:00:01\n2,joe,25,2012-10-02 02:00:02\n3,bob,30,2012-10-03 03:00:03"

	StubQuery(sql, RowsFromCSVString(columns, result))

	res, err := db.Query(sql)

	if err != nil {
		t.Fatal("stubbed query should not return error")
	}

	i := 0

	for res.Next() {
		var u = user{}
		err = res.Scan(&u.id, &u.name, &u.age, &u.created)

		if err != nil {
			t.Fatal(err)
		}

		if u.id == 0 || u.name == "" || u.age == 0 || u.created == "" {
			t.Fatal("failed to populate object with result")
		}
		i++
	}

	if i != 3 {
		t.Fatal("failed to return proper number of results")
	}
}

func TestSetQueryFunc(t *testing.T) {
	defer Reset()

	columns := []string{"id", "name", "age", "created"}
	rows := "1,tim,20,2012-10-01 01:00:01\n2,joe,25,2012-10-02 02:00:02\n3,bob,30,2012-10-03 03:00:03"

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return RowsFromCSVString(columns, rows), nil
	})

	if Conn().(*conn).queryFunc == nil {
		t.Fatal("query function not stubbed")
	}

	db, _ := sql.Open("testdb", "")

	res, err := db.Query("SELECT foo FROM bar")

	if err != nil {
		t.Fatal(err)
	}

	i := 0

	for res.Next() {
		var u = user{}
		err = res.Scan(&u.id, &u.name, &u.age, &u.created)

		if err != nil {
			t.Fatal(err)
		}

		if u.id == 0 || u.name == "" || u.age == 0 || u.created == "" {
			t.Fatal("failed to populate object with result")
		}
		i++
	}

	if i != 3 {
		t.Fatal("failed to return proper number of results")
	}
}

func TestSetQueryFuncError(t *testing.T) {
	defer Reset()

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return nil, errors.New("stubbed error")
	})

	db, _ := sql.Open("testdb", "")

	_, err := db.Query("SELECT foo FROM bar")

	if err == nil {
		t.Fatal("failed to return error from QueryFunc")
	}
}

func TestReset(t *testing.T) {
	sql.Open("testdb", "")

	sql := "select count(*) from error"
	StubQueryError(sql, errors.New("test error"))

	Reset()

	if len(d.conn.queries) > 0 {
		t.Fatal("failed to reset connection")
	}
}

func TestStubQueryRow(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from foo"
	columns := []string{"count"}
	result := `
  5
  `
	StubQuery(sql, RowsFromCSVString(columns, result))

	row := db.QueryRow(sql)

	if row == nil {
		t.Fatal("stub query should have returned row")
	}

	var count int64
	err := row.Scan(&count)

	if err != nil {
		t.Fatal(err)
	}

	if count != 5 {
		t.Fatal("failed to return count")
	}
}

func TestStubQueryRowReuse(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from foo"
	columns := []string{"count"}
	result := `
  5
  `
	StubQuery(sql, RowsFromCSVString(columns, result))

	i := 0
	rows, _ := db.Query(sql)
	for rows.Next() {
		i++
	}
	if i != 1 {
		t.Fatal("stub query should have returned one row")
	}

	j := i
	moreRows, _ := db.Query(sql)
	for moreRows.Next() {
		j++
	}

	if i == j {
		t.Fatal("stub query did not return another set of rows")
	}
}

func TestSetQueryFuncRow(t *testing.T) {
	defer Reset()

	columns := []string{"id", "name", "age", "created"}
	rows := "1,tim,20,2012-10-01 01:00:01"

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return RowsFromCSVString(columns, rows), nil
	})

	if Conn().(*conn).queryFunc == nil {
		t.Fatal("query function not stubbed")
	}

	db, _ := sql.Open("testdb", "")

	row := db.QueryRow("SELECT foo FROM bar")

	var u = user{}
	err := row.Scan(&u.id, &u.name, &u.age, &u.created)

	if err != nil {
		t.Fatal(err)
	}

	if u.id == 0 || u.name == "" || u.age == 0 || u.created == "" {
		t.Fatal("failed to populate object with result")
	}
}

func TestSetQueryFuncRowError(t *testing.T) {
	defer Reset()

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return nil, errors.New("Stubbed error")
	})

	if Conn().(*conn).queryFunc == nil {
		t.Fatal("query function not stubbed")
	}

	db, _ := sql.Open("testdb", "")

	row := db.QueryRow("SELECT foo FROM bar")

	var u = user{}
	err := row.Scan(&u.id, &u.name, &u.age, &u.created)

	if err == nil {
		t.Fatal("Did not return error")
	}
}

func TestStubExec(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "INSERT INTO foo SET (foo) VALUES (bar)"
	StubExec(sql, NewResult(5, errors.New("last insert error"), 3, errors.New("rows affected error")))

	res, err := db.Exec(sql)

	if err != nil {
		t.Fatal("stubbed exec call returned unexpected error")
	}

	var insertId int64
	insertId, err = res.LastInsertId()
	if insertId != 5 || err.Error() != "last insert error" {
		t.Fatal("stubbed exec did not return expected result")
	}

	var affected int64
	affected, err = res.RowsAffected()

	if affected != 3 || err.Error() != "rows affected error" {
		t.Fatal("stubbed exec did not return expected result")
	}
}

func TestStubExecError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	query := "INSERT INTO foo SET (foo) VALUES (bar)"
	StubExecError(query, errors.New("request failed"))

	res, err := db.Exec(query)

	if reflect.Indirect(reflect.ValueOf(res)).CanAddr() {
		t.Fatal("stubbed exec returned unexpected result")
	}

	if err == nil || err.Error() != "request failed" {
		t.Fatal("stubbed exec call did not return expected error")
	}
}

func TestStubExecFunc(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	query := "INSERT INTO foo SET (foo) VALUES (bar)"
	result := NewResult(5, errors.New("last insert error"), 3, errors.New("rows affected error"))

	SetExecFunc(func(query string) (driver.Result, error) {
		return result, nil
	})

	res, err := db.Exec(query)

	if err != nil {
		t.Fatal("stubbed exec returned unexpected error")
	}

	var insertId int64
	insertId, err = res.LastInsertId()
	if insertId != 5 || err.Error() != "last insert error" {
		t.Fatal("stubbed exec did not return expected result")
	}

	var affected int64
	affected, err = res.RowsAffected()

	if affected != 3 || err.Error() != "rows affected error" {
		t.Fatal("stubbed exec did not return expected result")
	}
}

func TestStubExecFuncError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	query := "INSERT INTO foo SET (foo) VALUES (bar)"

	SetExecFunc(func(query string) (driver.Result, error) {
		return nil, errors.New("request failed")
	})

	res, err := db.Exec(query)

	if res != nil {
		t.Fatal("stubbed exec unexpected result")
	}

	if err == nil || err.Error() != "request failed" {
		t.Fatal("stubbed exec did not return expected error")
	}
}

func TestSetBeginFunc(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	SetBeginFunc(func() (driver.Tx, error) {
		return nil, errors.New("begin failed")
	})

	res, err := db.Begin()

	if res != nil {
		t.Fatal("stubbed begin unexpected result")
	}

	if err == nil || err.Error() != "begin failed" {
		t.Fatal("stubbed begin did not return expected error")
	}
}

func TestStubBegin(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	StubBegin(nil, errors.New("begin failed"))
	res, err := db.Begin()

	if res != nil {
		t.Fatal("stubbed begin unexpected result")
	}

	if err == nil || err.Error() != "begin failed" {
		t.Fatal("stubbed begin did not return expected error")
	}
}

func TestSetCommitFunc(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	SetCommitFunc(func() error {
		return errors.New("commit failed")
	})

	tx, err := db.Begin()

	if tx == nil {
		t.Fatal("begin expected result")
	}

	if err != nil {
		t.Fatal("begin returned unexpected error")
	}

	err = tx.Commit()

	if err == nil || err.Error() != "commit failed" {
		t.Fatal("stubbed commit did not return expected error")
	}
}

func TestStubCommitError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	StubCommitError(errors.New("commit failed"))

	tx, err := db.Begin()

	if tx == nil {
		t.Fatal("begin expected result")
	}

	if err != nil {
		t.Fatal("begin returned unexpected error")
	}

	err = tx.Commit()

	if err == nil || err.Error() != "commit failed" {
		t.Fatal("stubbed commit did not return expected error")
	}
}

func TestSetRollbackFunc(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	SetRollbackFunc(func() error {
		return errors.New("rollback failed")
	})

	tx, err := db.Begin()

	if tx == nil {
		t.Fatal("begin expected result")
	}

	if err != nil {
		t.Fatal("begin returned unexpected error")
	}

	err = tx.Rollback()

	if err == nil || err.Error() != "rollback failed" {
		t.Fatal("stubbed rollback did not return expected error")
	}
}

func TestStubRollbackError(t *testing.T) {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	StubRollbackError(errors.New("rollback failed"))

	tx, err := db.Begin()

	if tx == nil {
		t.Fatal("begin expected result")
	}

	if err != nil {
		t.Fatal("begin returned unexpected error")
	}

	err = tx.Rollback()

	if err == nil || err.Error() != "rollback failed" {
		t.Fatal("stubbed rollback did not return expected error")
	}
}
