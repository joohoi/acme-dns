package testdb

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strconv"
)

type user struct {
	id      int64
	name    string
	age     int64
	created string
	data    string
}

func ExampleSetOpenFunc() {
	defer Reset()

	SetOpenFunc(func(dsn string) (driver.Conn, error) {
		// Conn() will return the same internal driver.Conn being used by the driver
		return Conn(), errors.New("test error")
	})

	// err only returns from this if it's an unknown driver, we are stubbing opening a connection
	db, _ := sql.Open("testdb", "foo")
	_, err := db.Driver().Open("foo")

	if err != nil {
		fmt.Println("Stubbed error returned as expected: " + err.Error())
	}

	// Output:
	// Stubbed error returned as expected: test error
}

func ExampleRowsFromCSVString() {
	columns := []string{"id", "name", "age", "created"}
	result := `
  1,tim,20,2012-10-01 01:00:01
  2,joe,25,2012-10-02 02:00:02
  3,bob,30,2012-10-03 03:00:03
  `
	rows := RowsFromCSVString(columns, result)

	fmt.Println(rows.Columns())

	// Output:
	// [id name age created]
}

func ExampleStubQuery() {
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

	res, _ := db.Query(sql)

	for res.Next() {
		var u = new(user)
		res.Scan(&u.id, &u.name, &u.age, &u.created)

		fmt.Println(u.name + " - " + strconv.FormatInt(u.age, 10))
	}

	// Output:
	// tim - 20
	// joe - 25
	// bob - 30
}

func ExampleStubQuery_queryRow() {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select id, name, age from users"
	columns := []string{"id", "name", "age", "created"}
	result := `
  1,tim,20,2012-10-01 01:00:01
  `
	StubQuery(sql, RowsFromCSVString(columns, result))

	row := db.QueryRow(sql)

	u := new(user)
	row.Scan(&u.id, &u.name, &u.age, &u.created)

	fmt.Println(u.name + " - " + strconv.FormatInt(u.age, 10))

	// Output:
	// tim - 20
}

func ExampleStubQueryError() {
	defer Reset()

	db, _ := sql.Open("testdb", "")

	sql := "select count(*) from error"

	StubQueryError(sql, errors.New("test error"))

	_, err := db.Query(sql)

	if err != nil {
		fmt.Println("Error returned: " + err.Error())
	}

	// Output:
	// Error returned: test error
}

func ExampleSetQueryFunc() {
	defer Reset()

	columns := []string{"id", "name", "age", "created"}
	rows := "1,tim,20,2012-10-01 01:00:01\n2,joe,25,2012-10-02 02:00:02\n3,bob,30,2012-10-03 03:00:03"

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return RowsFromCSVString(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")

	res, _ := db.Query("SELECT foo FROM bar")

	for res.Next() {
		var u = new(user)
		res.Scan(&u.id, &u.name, &u.age, &u.created)

		fmt.Println(u.name + " - " + strconv.FormatInt(u.age, 10))
	}

	// Output:
	// tim - 20
	// joe - 25
	// bob - 30
}

func ExampleSetQueryFunc_queryRow() {
	defer Reset()

	columns := []string{"id", "name", "age", "created"}
	rows := "1,tim,20,2012-10-01 01:00:01"

	SetQueryFunc(func(query string) (result driver.Rows, err error) {
		return RowsFromCSVString(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")

	row := db.QueryRow("SELECT foo FROM bar")

	var u = new(user)
	row.Scan(&u.id, &u.name, &u.age, &u.created)

	fmt.Println(u.name + " - " + strconv.FormatInt(u.age, 10))

	// Output:
	// tim - 20
}

func ExampleSetQueryWithArgsFunc() {
	defer Reset()

	SetQueryWithArgsFunc(func(query string, args []driver.Value) (result driver.Rows, err error) {
		columns := []string{"id", "name", "age", "created"}

		rows := ""
		if args[0] == "joe" {
			rows = "2,joe,25,2012-10-02 02:00:02"
		}
		return RowsFromCSVString(columns, rows), nil
	})

	db, _ := sql.Open("testdb", "")

	res, _ := db.Query("SELECT foo FROM bar WHERE name = $1", "joe")

	for res.Next() {
		var u = new(user)
		res.Scan(&u.id, &u.name, &u.age, &u.created)

		fmt.Println(u.name + " - " + strconv.FormatInt(u.age, 10))
	}

	// Output:
	// joe - 25
}

type testResult struct {
	lastId       int64
	affectedRows int64
}

func (r testResult) LastInsertId() (int64, error) {
	return r.lastId, nil
}

func (r testResult) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}

func ExampleSetExecWithArgsFunc() {
	defer Reset()

	SetExecWithArgsFunc(func(query string, args []driver.Value) (result driver.Result, err error) {
		if args[0] == "joe" {
			return testResult{1, 1}, nil
		}
		return testResult{1, 0}, nil
	})

	db, _ := sql.Open("testdb", "")

	res, _ := db.Exec("UPDATE bar SET name = 'foo' WHERE name = ?", "joe")

	rowsAffected, _ := res.RowsAffected()
	fmt.Println("RowsAffected =", rowsAffected)

	// Output:
	// RowsAffected = 1
}

func ExampleSetBeginFunc() {
	defer Reset()

	commitCalled := false
	rollbackCalled := false
	SetBeginFunc(func() (txn driver.Tx, err error) {
		t := &Tx{}
		t.SetCommitFunc(func() error {
			commitCalled = true
			return nil
		})
		t.SetRollbackFunc(func() error {
			rollbackCalled = true
			return nil
		})
		return t, nil
	})

	db, _ := sql.Open("testdb", "")
	tx, _ := db.Begin()
	tx.Commit()

	fmt.Println("CommitCalled =", commitCalled)
	fmt.Println("RollbackCalled =", rollbackCalled)

	// Output:
	// CommitCalled = true
	// RollbackCalled = false
}

func ExampleSetCommitFunc() {
	defer Reset()

	SetCommitFunc(func() error {
		return errors.New("commit failed")
	})

	db, _ := sql.Open("testdb", "")
	tx, _ := db.Begin()

	fmt.Println("CommitResult =", tx.Commit())

	// Output:
	// CommitResult = commit failed
}

func ExampleSetRollbackFunc() {
	defer Reset()

	SetRollbackFunc(func() error {
		return errors.New("rollback failed")
	})

	db, _ := sql.Open("testdb", "")
	tx, _ := db.Begin()

	fmt.Println("RollbackResult =", tx.Rollback())

	// Output:
	// RollbackResult = rollback failed
}
