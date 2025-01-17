package repos

import (
	"fmt"

	"github.com/samber/lo"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/driver/postgres"

	"gitlab.dev.ict/golang/go-ai/db"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
)

const (
	// PGConnStr = "host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC"
	PGConnStr = "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s"
)

// NewUserStoragePG creates a new instance of UserStoragePG
func NewUserStoragePG(props *db.DBProperties, log *gologgers.Logger, optFuncs ...OptFunc) (*us.StorageService, error) {
	log.Info("Creating new StorageService instance with Postgres db connection[%s:%d]", props.Host, props.Port)
	connStr := fmt.Sprintf(PGConnStr, props.Host, props.Port, props.User, props.Password, props.DBName, lo.Ternary(props.SSLMode == "", "disable", props.SSLMode))
	return newStorageService(postgres.Open(connStr), log, optFuncs...)
}
