package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/sqlite3"
	"github.com/gofiber/template/html/v2"
	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/handlers"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/server/hooks"
	"gitlab.dev.ict/golang/go-ai/server/middleware"
	"gitlab.dev.ict/golang/libs/gologgers"
)

type Server struct {
	app     *fiber.App
	config  *config.Config
	ah      *handlers.AppHandler
	storage *sqlite3.Storage
	log     *gologgers.Logger
}

func New(cfg *config.Config, h *handlers.AppHandler, storage *sqlite3.Storage) *Server {
	server := &Server{
		config:  cfg,
		ah:      h,
		storage: storage,
	}

	server.initialize()
	return server
}

func (s *Server) initialize() {
	engine := html.New("./web/views", ".html")
	engine.Reload(s.config.IsDev)
	engine.Debug(s.config.IsDebug)

	s.app = fiber.New(fiber.Config{
		AppName:           s.config.Title,
		Prefork:           false,
		EnablePrintRoutes: s.config.IsDebug,
		Views:             engine,
		ErrorHandler:      s.ErrorsHandler,
		PassLocalsToViews: true,
	})

	hooks.InitFiberHooks(s.app, s.config.Log)
	middleware.SetupMiddlewares(s.app, s.config.Log, s.storage)
	s.setupRoutes()
}

func (s *Server) Start() error {
	go s.gracefulShutdown()

	address := fmt.Sprintf(":%s", s.config.Port)

	if s.config.WithSSL {
		cert, err := s.setupSSL()
		if err != nil {
			return fmt.Errorf("SSL setup failed: %w", err)
		}
		return s.app.ListenTLSWithCertificate(address, cert)
	}

	return s.app.Listen(address)
}

func (s *Server) setupSSL() (tls.Certificate, error) {
	if err := os.MkdirAll(s.config.DirAppSSL, 0755); err != nil {
		return tls.Certificate{}, err
	}

	certPath := filepath.Join(s.config.DirAppSSL, helpers.DefFileCert)
	keyPath := filepath.Join(s.config.DirAppSSL, helpers.DefFileKey)

	return helpers.GenerateSSLCert(certPath, keyPath)
}

func (s *Server) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(ctx); err != nil {
		s.config.Log.Errorf("Server shutdown failed: %v", err)
	}

	s.config.Log.Info("Server gracefully stopped")
}

func (s *Server) errorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	s.config.Log.Errorf("Request error: %v", err)
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.Status(code).SendString(err.Error())
}

func (s *Server) permissionMiddleware() fiber.Handler {
	return middleware.PermissionCheck(
		s.ah,
		s.config.PermissionsConfig,
		"/access_denied",
		"/login_form",
	)
}
