package repos

import (
	"time"

	"github.com/samber/lo"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gitlab.dev.ict/golang/go-ai/db"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
)

// Options holds configuration parameters for the storage service
type Options struct {
	isDebug          bool
	isForceinitRoles bool
}

// OptFunc is a function that modifies Options
type OptFunc func(*Options)

// WithDebug sets the isDebug option
func WithDebug(isDebug bool) OptFunc {
	return func(opts *Options) {
		opts.isDebug = isDebug
	}
}

// WithForceInitRoles sets the isForceinitRoles option
func WithForceInitRoles(isForceinitRoles bool) OptFunc {
	return func(opts *Options) {
		opts.isForceinitRoles = isForceinitRoles
	}
}

// NewUserStoragePG creates a new instance of UserStoragePG
func newStorageService(dialector gorm.Dialector, log *gologgers.Logger, optFuncs ...OptFunc) (*us.StorageService, error) {
	// Initialize default options
	opts := &Options{}
	// Apply option functions
	for _, optFunc := range optFuncs {
		optFunc(opts)
	}

	r := log.Rec(us.CH)
	isLogDebug := lo.Ternary(opts.isDebug, logger.Info, logger.Error)
	r.Infof("Init DBPool with log_mode=%d", isLogDebug)

	// Open the database connection
	dataBase, err := gorm.Open(dialector, newGormConfig(r, opts.isDebug))
	if err != nil {
		r.Errorf("userStorage init failed! err: %v", err)
		return nil, err
	}

	if opts.isDebug {
		dataBase = dataBase.Debug()
	}

	// Attempt to drop conflicting indexes/constraints
	// tx := dataBase.Exec("DROP INDEX IF EXISTS idx_users_username")
	// dataBase.Exec("DROP INDEX IF EXISTS users_username_key")
	// r.Trace("drop users_username_key! err:", tx.Error)
	// Migrate the schema - ensures that the necessary tables are created or updated in the database.
	// err = dataBase.AutoMigrate(&us.User{}, &us.Chat{})
	err = us.AutoMigrate(dataBase)
	if err != nil {
		r.Errorf("userStorage AutoMigrate failed! err: %v", err)
		// return nil, err
	}

	err = us.InitializeRolesAndPermissions(dataBase, r, opts.isForceinitRoles)
	if err != nil {
		r.Errorf("userStorage InitializeRolesAndPermissions failed! err: %v", err)
	}

	r.Info("add callbacks")
	dataBase.Callback().Query().Before("gorm:query").Register("log_before_find", db.CallbackBeforeFind)
	dataBase.Callback().Query().After("gorm:after_query").Register("process_after_find", db.CallbackAfterFind)
	dataBase.Callback().Raw().Before("gorm:raw_query").Register("log_before_raw_query", db.CallbackBeforeQuery("rAw_query:BEFORE", r))
	dataBase.Callback().Row().Before("gorm:row_query").Register("log_before_row_query", db.CallbackBeforeQuery("rOw_query:BEFORE", r))
	dataBase.Callback().Row().After("gorm:row_query").Register("close_row_after_query", db.CallbackAfterQuery("rOw_query:AFTER", r))

	r.Info("configure DBPool")
	db.ConfigureDBPool(dataBase)
	return us.NewStorageService(dataBase, log, opts.isDebug), nil
}

// newGormConfig returns a comprehensive gorm.Config
func newGormConfig(r *gologgers.LogRec, isDebug bool) *gorm.Config {
	return &gorm.Config{
		// Logger configuration
		Logger: logger.New(r, logger.Config{
			SlowThreshold:        2 * time.Second,                                // Log queries that take longer than 2 seconds
			ParameterizedQueries: true,                                           // Use parameterized queries to prevent SQL injection
			Colorful:             isDebug,                                        // Enable colorful logging if in debug mode
			LogLevel:             lo.Ternary(isDebug, logger.Info, logger.Error), // Set log level based on debug mode
		}),
		DisableForeignKeyConstraintWhenMigrating: false, // Disable foreign key constraints when migrating
		PrepareStmt:                              true,  // Enable prepared statements for performance
		SkipDefaultTransaction:                   true,  // Skip default transactions for performance
	}
}
