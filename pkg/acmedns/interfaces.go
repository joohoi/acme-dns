package acmedns

import (
	"database/sql"
	"github.com/google/uuid"
)

type AcmednsDB interface {
	Register(cidrslice Cidrslice) (ACMETxt, error)
	GetByUsername(uuid.UUID) (ACMETxt, error)
	GetTXTForDomain(string) ([]string, error)
	Update(ACMETxtPost) error
	GetBackend() *sql.DB
	SetBackend(*sql.DB)
	Close()
}

type AcmednsNS interface {
	Start(errorChannel chan error)
	SetOwnAuthKey(key string)
	ParseRecords()
}
