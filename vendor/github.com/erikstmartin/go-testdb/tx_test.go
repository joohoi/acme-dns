package testdb

import (
	"errors"
	"testing"
)

func TestTxSetCommitFunc(t *testing.T) {
	tx := &Tx{}

	tx.SetCommitFunc(func() error {
		return errors.New("commit failed")
	})

	err := tx.Commit()

	if err == nil || err.Error() != "commit failed" {
		t.Fatal("stubbed commit did not return expected error")
	}
}

func TestTxStubCommitError(t *testing.T) {
	tx := &Tx{}

	tx.StubCommitError(errors.New("commit failed"))

	err := tx.Commit()

	if err == nil || err.Error() != "commit failed" {
		t.Fatal("stubbed commit did not return expected error")
	}
}

func TestTxSetRollbackFunc(t *testing.T) {
	tx := &Tx{}

	tx.SetRollbackFunc(func() error {
		return errors.New("rollback failed")
	})

	err := tx.Rollback()

	if err == nil || err.Error() != "rollback failed" {
		t.Fatal("stubbed rollback did not return expected error")
	}
}

func TestTxStubRollbackError(t *testing.T) {
	tx := &Tx{}

	tx.StubRollbackError(errors.New("rollback failed"))

	err := tx.Rollback()

	if err == nil || err.Error() != "rollback failed" {
		t.Fatal("stubbed rollback did not return expected error")
	}
}
