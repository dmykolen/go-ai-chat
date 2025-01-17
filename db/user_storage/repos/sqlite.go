package repos

import (
	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/driver/sqlite"

	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
)

func NewUserStorageSQLite(pathToDB string, log *gologgers.Logger, optFuncs ...OptFunc) (*us.StorageService, error) {
	log.Info("Creating new StorageService instance with SQLite db connection[%s]", pathToDB)
	return newStorageService(sqlite.Open(pathToDB), log, optFuncs...)
}
