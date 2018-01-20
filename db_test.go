package main

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/erikstmartin/go-testdb"
	"testing"
)

type testResult struct {
	lastID       int64
	affectedRows int64
}

func (r testResult) LastInsertId() (int64, error) {
	return r.lastID, nil
}

func (r testResult) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}

func TestDBInit(t *testing.T) {
	fakeDB := new(acmedb)
	err := fakeDB.Init("notarealegine", "connectionstring")
	if err == nil {
		t.Errorf("Was expecting error, didn't get one.")
	}

	testdb.SetExecWithArgsFunc(func(query string, args []driver.Value) (result driver.Result, err error) {
		return testResult{1, 0}, errors.New("Prepared query error")
	})
	defer testdb.Reset()

	errorDB := new(acmedb)
	err = errorDB.Init("testdb", "")
	if err == nil {
		t.Errorf("Was expecting DB initiation error but got none")
	}
	errorDB.Close()
}

func TestRegisterNoCIDR(t *testing.T) {
	// Register tests
	_, err := DB.Register(cidrslice{})
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}
}

func TestRegisterMany(t *testing.T) {
	for i, test := range []struct {
		input  cidrslice
		output cidrslice
	}{
		{cidrslice{"127.0.0.1/8", "8.8.8.8/32", "1.0.0.1/1"}, cidrslice{"127.0.0.1/8", "8.8.8.8/32", "1.0.0.1/1"}},
		{cidrslice{"1.1.1./32", "1922.168.42.42/8", "1.1.1.1/33", "1.2.3.4/"}, cidrslice{}},
		{cidrslice{"7.6.5.4/32", "invalid", "1.0.0.1/2"}, cidrslice{"7.6.5.4/32", "1.0.0.1/2"}},
	} {
		user, err := DB.Register(test.input)
		if err != nil {
			t.Errorf("Test %d: Got error from register method: [%v]", i, err)
		}
		res, err := DB.GetByUsername(user.Username)
		if err != nil {
			t.Errorf("Test %d: Got error when fetching username: [%v]", i, err)
		}
		if len(user.AllowFrom) != len(test.output) {
			t.Errorf("Test %d: Expected to recieve struct with [%d] entries in AllowFrom, but got [%d] records", i, len(test.output), len(user.AllowFrom))
		}
		if len(res.AllowFrom) != len(test.output) {
			t.Errorf("Test %d: Expected to recieve struct with [%d] entries in AllowFrom, but got [%d] records", i, len(test.output), len(res.AllowFrom))
		}

	}
}

func TestGetByUsername(t *testing.T) {
	// Create  reg to refer to
	reg, err := DB.Register(cidrslice{})
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

func TestPrepareErrors(t *testing.T) {
	reg, _ := DB.Register(cidrslice{})
	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()
	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)
	defer testdb.Reset()

	_, err = DB.GetByUsername(reg.Username)
	if err == nil {
		t.Errorf("Expected error, but didn't get one")
	}

	_, err = DB.GetTXTForDomain(reg.Subdomain)
	if err == nil {
		t.Errorf("Expected error, but didn't get one")
	}
}

func TestQueryExecErrors(t *testing.T) {
	reg, _ := DB.Register(cidrslice{})
	testdb.SetExecWithArgsFunc(func(query string, args []driver.Value) (result driver.Result, err error) {
		return testResult{1, 0}, errors.New("Prepared query error")
	})

	testdb.SetQueryWithArgsFunc(func(query string, args []driver.Value) (result driver.Rows, err error) {
		columns := []string{"Username", "Password", "Subdomain", "Value", "LastActive"}
		return testdb.RowsFromSlice(columns, [][]driver.Value{}), errors.New("Prepared query error")
	})

	defer testdb.Reset()

	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()

	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)

	_, err = DB.GetByUsername(reg.Username)
	if err == nil {
		t.Errorf("Expected error from exec, but got none")
	}

	_, err = DB.GetTXTForDomain(reg.Subdomain)
	if err == nil {
		t.Errorf("Expected error from exec in GetByDomain, but got none")
	}

	_, err = DB.Register(cidrslice{})
	if err == nil {
		t.Errorf("Expected error from exec in Register, but got none")
	}
	reg.Value = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	err = DB.Update(reg)
	if err == nil {
		t.Errorf("Expected error from exec in Update, but got none")
	}

}

func TestQueryScanErrors(t *testing.T) {
	reg, _ := DB.Register(cidrslice{})

	testdb.SetExecWithArgsFunc(func(query string, args []driver.Value) (result driver.Result, err error) {
		return testResult{1, 0}, errors.New("Prepared query error")
	})

	testdb.SetQueryWithArgsFunc(func(query string, args []driver.Value) (result driver.Rows, err error) {
		columns := []string{"Only one"}
		resultrows := "this value"
		return testdb.RowsFromCSVString(columns, resultrows), nil
	})

	defer testdb.Reset()
	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()

	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)

	_, err = DB.GetByUsername(reg.Username)
	if err == nil {
		t.Errorf("Expected error from scan in, but got none")
	}
}

func TestBadDBValues(t *testing.T) {
	reg, _ := DB.Register(cidrslice{})

	testdb.SetQueryWithArgsFunc(func(query string, args []driver.Value) (result driver.Rows, err error) {
		columns := []string{"Username", "Password", "Subdomain", "Value", "LastActive"}
		resultrows := "invalid,invalid,invalid,invalid,"
		return testdb.RowsFromCSVString(columns, resultrows), nil
	})

	defer testdb.Reset()
	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()

	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)

	_, err = DB.GetByUsername(reg.Username)
	if err == nil {
		t.Errorf("Expected error from scan in, but got none")
	}

	_, err = DB.GetTXTForDomain(reg.Subdomain)
	if err == nil {
		t.Errorf("Expected error from scan in GetByDomain, but got none")
	}
}

func TestGetTXTForDomain(t *testing.T) {
	// Create  reg to refer to
	reg, err := DB.Register(cidrslice{})
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	txtval1 := "___validation_token_received_from_the_ca___"
	txtval2 := "___validation_token_received_YEAH_the_ca___"

	reg.Value = txtval1
	_ = DB.Update(reg)

	reg.Value = txtval2
	_ = DB.Update(reg)

	regDomainSlice, err := DB.GetTXTForDomain(reg.Subdomain)
	if err != nil {
		t.Errorf("Could not get test user, got error [%v]", err)
	}
	if len(regDomainSlice) == 0 {
		t.Errorf("No rows returned for GetTXTForDomain [%s]", reg.Subdomain)
	}

	var val1_found bool = false
	var val2_found bool = false
	for _, v := range regDomainSlice {
		if v == txtval1 {
			val1_found = true
		}
		if v == txtval2 {
			val2_found = true
		}
	}
	if !val1_found {
		t.Errorf("No TXT value found for val1")
	}
	if !val2_found {
		t.Errorf("No TXT value found for val2")
	}

	// Not found
	regNotfound, _ := DB.GetTXTForDomain("does-not-exist")
	if len(regNotfound) > 0 {
		t.Errorf("No records should be returned.")
	}
}

func TestUpdate(t *testing.T) {
	// Create  reg to refer to
	reg, err := DB.Register(cidrslice{})
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
}
