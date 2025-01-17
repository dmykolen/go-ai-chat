package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/sqlite3"
	"github.com/gofiber/template/html/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/handlers"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/server/hooks"
	"gitlab.dev.ict/golang/go-ai/server/middleware"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

var (
	log = gl.New(gl.WithChannel("TEST"), gl.WithLevel(gl.LevelInfo), gl.WithOC(true))
	cfg = config.New()
	ah  *handlers.AppHandler
)

func init() {
	cfg.Load()
	ah = handlers.NewAppHandler(
		handlers.WithLogger(log),
		handlers.WithPGDB(nil), // Mock DB for tests
	)
}

// testSuite holds common test dependencies
type testSuite struct {
	app    *fiber.App
	engine *html.Engine
	ctx    context.Context
}

// setupTestSuite creates a new test suite with all dependencies
func setupTestSuite(t *testing.T) *testSuite {
	t.Helper()

	ctx := utils.GenerateCtxWithRid()
	app, engine := prepareApp(t)

	return &testSuite{
		app:    app,
		engine: engine,
		ctx:    ctx,
	}
}

// prepareApp creates a test Fiber app with required middleware and routes
func prepareApp(t *testing.T) (*fiber.App, *html.Engine) {
	t.Helper()

	engine := html.New("./web/views", ".html")
	engine.Reload(true)
	engine.Debug(true)
	engine.AddFuncMap(helpers.FuncMap)

	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	storage := initTestStorage()
	setupMiddleware(app, storage)
	setupRoutes(app)

	return app, engine
}

func initTestStorage() fiber.Storage {
	return sqlite3.New(sqlite3.Config{
		Database: ":memory:",
		Table:    "fiber_storage",
	})
}

func setupMiddleware(app *fiber.App, storage fiber.Storage) {
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("storage", storage)
		return c.Next()
	})
}

func setupRoutes(app *fiber.App) {
	app.Static("/", "./web/static")
	app.Static("/web/files", "./web/files")

	app.Post("/login", ah.Login)
	app.Get("/logout", ah.Logout)

	api := app.Group("/api/v1")
	api.Use(ah.CheckAuth)
	setupAPIRoutes(api)
}

func setupAPIRoutes(api fiber.Router) {
	api.Post("/ask-db", ah.AskDB)
	api.Post("/ask-ai-voip", ah.AskAiVoIP_NEW)
	api.Get("/users/:username", ah.GetUserByName)
	api.Get("/users/chats/:username", ah.GetUserChats)
}

// TestRoutes verifies basic route functionality
func TestRoutes(t *testing.T) {
	ts := setupTestSuite(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Get Home",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Get API Status",
			method:         http.MethodGet,
			path:           "/api/v1/status",
			expectedStatus: fiber.StatusUnauthorized, // Should fail without auth
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			resp, err := ts.app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestAuthenticatedRoutes verifies routes requiring authentication
func TestAuthenticatedRoutes(t *testing.T) {
	ts := setupTestSuite(t)
	user := createTestUser(t, "testuser")

	tests := []struct {
		name           string
		path           string
		authenticated  bool
		expectedStatus int
	}{
		{
			name:           "API Unauthorized",
			path:           "/api/v1/users/testuser",
			authenticated:  false,
			expectedStatus: fiber.StatusUnauthorized,
		},
		{
			name:           "API Authorized",
			path:           "/api/v1/users/testuser",
			authenticated:  true,
			expectedStatus: fiber.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.authenticated {
				addAuthCookies(req, user)
			}
			resp, err := ts.app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestTemplateRendering verifies template rendering functionality
func TestTemplateRendering(t *testing.T) {
	ts := setupTestSuite(t)

	tests := []struct {
		name        string
		template    string
		data        fiber.Map
		expectError bool
	}{
		{
			name:     "Render Main Template",
			template: "ai_index",
			data:     fiber.Map{"Title": "Test Title"},
		},
		{
			name:     "Render Partial Template",
			template: "partials/header",
			data:     fiber.Map{"Title": "Test Header"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ts.engine.Render(io.Discard, tt.template, tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test utilities

func createTestUser(t *testing.T, login string) *handlers.User {
	t.Helper()
	user := handlers.NewUser(login, utils.UUID())
	handlers.GetAppStoreForUsers().Store(user.UUID, user)
	return user
}

func addAuthCookies(req *http.Request, user *handlers.User) {
	req.AddCookie(&http.Cookie{
		Name:  handlers.CookUID,
		Value: user.UUID,
	})
	req.AddCookie(&http.Cookie{
		Name:  handlers.CookUName,
		Value: user.Login,
	})
}

func serverBaseSetup() *fiber.App {
	server := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
			c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
			return c.Status(code).SendString(err.Error())
		},
	})
	hooks.InitFiberHooks(server, log)
	middleware.SetupMiddlewares(server, log, initStorage())
	return server
}

func createHtmlEngine(dir, ext string) (e *html.Engine) {
	e = html.New(dir, ext)
	e.Engine.Reload(true).Debug(true).Layout("embed")
	e.AddFuncMap(helpers.FuncMap)
	return
}

func RouteGroupApi(app *fiber.App, r fiber.Router, handlers ...fiber.Handler) {
	r.Get("/", func(c *fiber.Ctx) error { return c.SendString(fmt.Sprintf("Hello from %s!", c.Route().Path)) })

	r.Post("/ask-db", ah.AskDB)
	r.Post("/ask-ai-voip", ah.AskAiVoIP_NEW)
	r.Post("/stt", ah.HandleSTT)
	r.Post("/rate", ah.RateChat)
	r.Get("/users/:username", ah.GetUserByName)
	r.Get("/users/chats/:username", ah.GetUserChats)
	r.Get("/users/:username?/chats/:type", ah.GetChats)
	r.Get("/users/photo/:id", ah.GetUserPhoto)
}
