package database

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/erikstmartin/go-testdb"
	"go.uber.org/zap"

	"github.com/joohoi/acme-dns/pkg/acmedns"
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

func fakeConfigAndLogger() (acmedns.AcmeDnsConfig, *zap.SugaredLogger) {
	c := acmedns.AcmeDnsConfig{}
	c.Database.Engine = "sqlite"
	c.Database.Connection = ":memory:"
	l := zap.NewNop().Sugar()
	return c, l
}

func fakeDB() acmedns.AcmednsDB {
	conf, logger := fakeConfigAndLogger()
	db, _ := Init(&conf, logger)
	return db
}

func TestRegisterNoCIDR(t *testing.T) {
	// Register tests
	DB := fakeDB()
	_, err := DB.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}
}

func TestRegisterMany(t *testing.T) {
	DB := fakeDB()
	for i, test := range []struct {
		input  acmedns.Cidrslice
		output acmedns.Cidrslice
	}{
		{acmedns.Cidrslice{"127.0.0.1/8", "8.8.8.8/32", "1.0.0.1/1"}, acmedns.Cidrslice{"127.0.0.1/8", "8.8.8.8/32", "1.0.0.1/1"}},
		{acmedns.Cidrslice{"1.1.1./32", "1922.168.42.42/8", "1.1.1.1/33", "1.2.3.4/"}, acmedns.Cidrslice{}},
		{acmedns.Cidrslice{"7.6.5.4/32", "invalid", "1.0.0.1/2"}, acmedns.Cidrslice{"7.6.5.4/32", "1.0.0.1/2"}},
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
			t.Errorf("Test %d: Expected to receive struct with [%d] entries in AllowFrom, but got [%d] records", i, len(test.output), len(user.AllowFrom))
		}
		if len(res.AllowFrom) != len(test.output) {
			t.Errorf("Test %d: Expected to receive struct with [%d] entries in AllowFrom, but got [%d] records", i, len(test.output), len(res.AllowFrom))
		}

	}
}

func TestGetByUsername(t *testing.T) {
	DB := fakeDB()
	// Create  reg to refer to
	reg, err := DB.Register(acmedns.Cidrslice{})
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
	if !acmedns.CorrectPassword(reg.Password, regUser.Password) {
		t.Errorf("The password [%s] does not match the hash [%s]", reg.Password, regUser.Password)
	}
}

func TestPrepareErrors(t *testing.T) {
	DB := fakeDB()
	reg, _ := DB.Register(acmedns.Cidrslice{})
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
	DB := fakeDB()
	reg, _ := DB.Register(acmedns.Cidrslice{})
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

	_, err = DB.Register(acmedns.Cidrslice{})
	if err == nil {
		t.Errorf("Expected error from exec in Register, but got none")
	}
	reg.Value = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	err = DB.Update(reg.ACMETxtPost)
	if err == nil {
		t.Errorf("Expected error from exec in Update, but got none")
	}

}

func TestQueryScanErrors(t *testing.T) {
	DB := fakeDB()
	reg, _ := DB.Register(acmedns.Cidrslice{})

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
	DB := fakeDB()
	reg, _ := DB.Register(acmedns.Cidrslice{})

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
	DB := fakeDB()
	// Create  reg to refer to
	reg, err := DB.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	txtval1 := "___validation_token_received_from_the_ca___"
	txtval2 := "___validation_token_received_YEAH_the_ca___"

	reg.Value = txtval1
	_ = DB.Update(reg.ACMETxtPost)

	reg.Value = txtval2
	_ = DB.Update(reg.ACMETxtPost)

	regDomainSlice, err := DB.GetTXTForDomain(reg.Subdomain)
	if err != nil {
		t.Errorf("Could not get test user, got error [%v]", err)
	}
	if len(regDomainSlice) == 0 {
		t.Errorf("No rows returned for GetTXTForDomain [%s]", reg.Subdomain)
	}

	var val1found = false
	var val2found = false
	for _, v := range regDomainSlice {
		if v == txtval1 {
			val1found = true
		}
		if v == txtval2 {
			val2found = true
		}
	}
	if !val1found {
		t.Errorf("No TXT value found for val1")
	}
	if !val2found {
		t.Errorf("No TXT value found for val2")
	}

	// Not found
	regNotfound, _ := DB.GetTXTForDomain("does-not-exist")
	if len(regNotfound) > 0 {
		t.Errorf("No records should be returned.")
	}
}

func TestUpdate(t *testing.T) {
	DB := fakeDB()
	// Create  reg to refer to
	reg, err := DB.Register(acmedns.Cidrslice{})
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

	err = DB.Update(regUser.ACMETxtPost)
	if err != nil {
		t.Errorf("DB Update failed, got error: [%v]", err)
	}
}

func TestUnregister(t *testing.T) {
	DB := fakeDB()
	// Create reg to refer to
	reg, err := DB.Register(acmedns.Cidrslice{})
	if err != nil {
		t.Errorf("Registration failed, got error [%v]", err)
	}

	// Add a TXT value so we can verify it gets removed too
	reg.Value = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err = DB.Update(reg.ACMETxtPost); err != nil {
		t.Errorf("DB Update failed, got error: [%v]", err)
	}

	if err = DB.Unregister(reg.Username, reg.Subdomain); err != nil {
		t.Errorf("Unregister failed, got error: [%v]", err)
	}

	// The user record must be gone
	if _, err = DB.GetByUsername(reg.Username); err == nil {
		t.Errorf("Expected error getting the deregistered user, but got none")
	}

	// The TXT rows must be gone
	txts, err := DB.GetTXTForDomain(reg.Subdomain)
	if err != nil {
		t.Errorf("GetTXTForDomain failed, got error: [%v]", err)
	}
	if len(txts) > 0 {
		t.Errorf("Expected no TXT records after Unregister, but got [%d]", len(txts))
	}
}

func TestUnregisterExecError(t *testing.T) {
	DB := fakeDB()
	reg, _ := DB.Register(acmedns.Cidrslice{})

	testdb.SetExecWithArgsFunc(func(query string, args []driver.Value) (result driver.Result, err error) {
		return testResult{}, errors.New("error")
	})
	defer testdb.Reset()
	tdb, err := sql.Open("testdb", "")
	if err != nil {
		t.Errorf("Got error: %v", err)
	}
	oldDb := DB.GetBackend()
	DB.SetBackend(tdb)
	defer DB.SetBackend(oldDb)

	if err = DB.Unregister(reg.Username, reg.Subdomain); err == nil {
		t.Errorf("Expected error from Unregister exec, but got none")
	}
}

func TestTXTSubdomainIndex(t *testing.T) {
	// The DNS hot path filters txt rows by Subdomain on every query; without this
	// index the lookup degrades to a full table scan as registrations grow.
	DB := fakeDB()
	backend := DB.GetBackend()
	var name string
	err := backend.QueryRow(
		"SELECT name FROM sqlite_master WHERE type='index' AND name='idx_txt_subdomain'").Scan(&name)
	if err != nil {
		t.Errorf("Expected idx_txt_subdomain index to exist after Init, got error [%v]", err)
	}
	if name != "idx_txt_subdomain" {
		t.Errorf("Expected index name [idx_txt_subdomain], got [%s]", name)
	}
}
