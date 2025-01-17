//go:generate swag init --exclude _testdata/ -o "docs/swagger"
package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/storage/sqlite3"
	"github.com/gookit/goutil"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/server"
	"gitlab.dev.ict/golang/libs/utils"
)

//go:embed assets/permissions.yml
var permissionsYAML []byte

const (
	DIR_UPLOAD = "./uploads"
	ENV_DEV    = ".env.dev"
)

func init() {
	// Load environment variables
	if len(os.Args) > 1 {
		if strings.HasSuffix(os.Args[0], ".test") {
			lo.Must0(godotenv.Overload(ENV_DEV))
		}
		if strings.HasSuffix(os.Args[1], ".env") && utils.IsExists(os.Args[1]) {
			lo.Must0(godotenv.Overload(os.Args[1]))
		}
	}

	// Create required directories
	goutil.MustOK(os.MkdirAll(DIR_UPLOAD, os.FileMode(0775)))
}

// @title           LIFECELL AI VoIP customer support
// @description     This is app for customer support in VoIP area
// @contact.name    Dmytro Mykolenko
// @contact.email   dmytro.mykolenko@pe.lifecell.com.ua
// @host            ai.dev.ict:7557
// @BasePath        /
func main() {
	// Load configuration
	cfg := config.New()
	if err := cfg.Load(); err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// Create and start server
	srv := server.New(cfg, cfg.HandlerApp, initStorage())

	// Handle shutdown signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			cfg.Log.Fatalf("Server failed to start: %v", err)
		}
	}()

	cfg.Log.Infof("Server started on port %s", cfg.Port)

	// Wait for shutdown signal
	<-done
	cfg.Log.Info("Server shutting down...")
	cfg.Log.Info("Server stopped")
	os.Exit(0)
}

func initStorage() *sqlite3.Storage {
	return sqlite3.New(sqlite3.Config{
		Database:        "./fiber.sqlite3",
		Table:           "fiber_storage",
		Reset:           false,
		GCInterval:      time.Minute,
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
	})
}
