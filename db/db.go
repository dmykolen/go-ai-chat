package db

import (
	"context"
	"reflect"
	"time"

	"github.com/gookit/slog"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/gorm"
)

// Define a context key to avoid collisions
type contextKey string

const (
	KeyStartTime = contextKey("startTime")
	KeyLog       = contextKey("log")
	RoleAdmin    = "ADMIN"
	RoleU        = "USUAL"

	RoleCodeSuperAdmin    = "SUPERADMIN"
	RoleCodeAdminVoIP     = "VOIP_ADMIN"
	RoleCodeVoIPUser      = "VOIP_USER"
	RoleCodeVectorDBAdmin = "VECTORDB_ADMIN"
)

func Log(db *gorm.DB, records ...*slog.Record) *slog.Record {
	switch {
	case db.Statement.Context != nil && db.Statement.Context.Value(KeyLog) != nil:
		return db.Statement.Context.Value(KeyLog).(*slog.Record)
	case len(records) > 0:
		return records[0]
	case db.Statement.Context != nil && db.Statement.Context.Value(utils.RID) != nil:
		return gologgers.Defult().RecWithRid(db.Statement.Context.Value(utils.RID).(string))
	default:
		return gologgers.Defult().Rec("no-rec-in-context")
	}
}

type DBProperties struct {
	Host     string `json:"host" env:"POSTGRES_HOST" validate:"required,hostname" default:"localhost"`
	Port     int    `json:"port" env:"POSTGRES_PORT" validate:"required,min=1,max=65535"`
	User     string `json:"user" env:"POSTGRES_USER" validate:"required"`
	Password string `json:"password" env:"POSTGRES_PASSWORD" validate:"required"`
	DBName   string `json:"dbname" env:"POSTGRES_DB_NAME" validate:"required"`
	SSLMode  string `json:"sslmode" env:"POSTGRES_SSL_MODE" validate:"required,oneof=disable require verify-ca verify-full"`
}

func CallbackBeforeFind(db *gorm.DB) {
	db.Statement.Context = context.WithValue(db.Statement.Context, KeyStartTime, time.Now())
}

func CallbackAfterFind(db *gorm.DB) {
	log := Log(db)
	if startTime, exists := db.Statement.Context.Value(KeyStartTime).(time.Time); exists {
		log.AddValue("elapsed", time.Since(startTime).String())
	}

	switch db.Statement.ReflectValue.Kind() {
	case reflect.Slice:
		log.AddValue("rows", db.Statement.ReflectValue.Len())
	case reflect.Struct:
		if db.Statement.ReflectValue.IsZero() {
			log.AddValue("rows", 0)
		} else {
			log.AddValue("rows", 1)
		}
	}
	log.Infof("Finish query=[%s] result_type=[%T] PARAMS: %v", db.Statement.SQL.String(), db.Statement.Model, db.Statement.Vars)
}

func CallbackBeforeQuery(name string, r *slog.Record) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		Log(db, r).Debugf("%s => %s", name, db.Statement.SQL.String())
	}
}

func CallbackAfterQuery(name string, r *slog.Record) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		Log(db, r).Debugf("%s => %v", name, db.Statement.Model)
	}
}

func ConfigureDBPool(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get database object")
	}

	sqlDB.SetMaxIdleConns(10)           // SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxOpenConns(100)          // SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetConnMaxLifetime(time.Hour) // SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
}
