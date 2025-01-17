package middleware

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/storage/sqlite3"
	"github.com/gofiber/swagger"
	h "gitlab.dev.ict/golang/go-ai/handlers"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	PathLoginForm = "/login_form?referrer=%s"
)

var (
	patternNotLogged = regexp.MustCompile(`.*\.(js|css|woff2|png|ico)$`)
)

func SetupMiddlewares(s *fiber.App, log *gl.Logger, storage *sqlite3.Storage) {
	s.Get("/metrics", monitor.New())
	s.Use(limiter.New(limiter.Config{Storage: storage, Max: 300}))
	s.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
	s.Use(requestid.New())
	s.Use(AddLogger(log))
	s.Get("/swagger/*", AddSwagger())
	s.Use(Recover())

	s.Use(ErrorsMiddleware)
}

func Cache(exp time.Duration) fiber.Handler {
	return cache.New(cache.Config{
		Expiration:   exp,
		CacheControl: true,
	})
}

type responseWriter struct {
	bufio.Writer
	fiber.Response
}

func (w responseWriter) Write(b []byte) error {
	w.Writer.Write(b)                                   // write to the buffer
	return w.Response.Write(bufio.NewWriter(&w.Writer)) // write to the actual response
}

func AddLogger(l *gl.Logger) fiber.Handler {
	l.Info("Init middleware - logger. [add slog.Record to context]")
	return func(c *fiber.Ctx) error {
		ts := time.Now()
		r := l.RecWithCtx(utils.CtxWithRid(help.Rid(c)), "serv")
		c.Locals(help.CtxLogger, r)
		if patternNotLogged.MatchString(c.Path()) {
			return c.Next()
		}
		r.Infof("Begin process input request. [ctxId=%d;ConnID=%d;ConnNum=%d] %s", c.Context().ID(), c.Context().ConnID(), c.Context().ConnRequestNum(), c.Context().String())
		r.WithData(gl.M{"userId": c.Cookies(h.CookUID), "userName": c.Cookies(h.CookUName)}).Debugf("ContentType=%s Body=[%d]", c.Get(fiber.HeaderContentType), len(c.Body()))
		c.Next()
		r.AddData(gl.M{"status": c.Response().StatusCode(), "elapsed": time.Since(ts).Seconds(), "ctxId": c.Context().ID()}).Info("Finish process input request.")
		return nil
	}
}

func Recover() fiber.Handler {
	return recover.New(
		recover.Config{
			Next:              nil,
			EnableStackTrace:  true,
			StackTraceHandler: func(c *fiber.Ctx, e interface{}) { help.Log(c).Errorf("panic: %v\n%s\n", e, debug.Stack()) },
		},
	)
}

func AddSwagger() fiber.Handler {
	return swagger.New(swagger.Config{
		DisplayRequestDuration: true,
		Title:                  "AI Lifecell",
		RequestSnippetsEnabled: true,
		TryItOutEnabled:        true,
		SyntaxHighlight: &swagger.SyntaxHighlightConfig{
			Activate: true,
			Theme:    "tomorrow-night",
		},
	})
}
func AddFiberLogger() fiber.Handler {
	return logger.New(logger.Config{
		DisableColors: false,
		Output:        os.Stdout,
		TimeFormat:    gl.TS_FMT,
		Format:        "${locals:requestid} status=${status} elapsed=${latency} request_was=[${method} ${cyan}${path}] ${protocol}:${ip} ${red}ERR=${error}${reset}\n",
		Done:          func(c *fiber.Ctx, logString []byte) { help.Log(c).Infof("Finish %s", logString) },
	})
}

func Authenticate() fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := help.Log(c)
		log.Debug("→ Starting auth check for path:", c.Path())

		isHTMX := c.Get("HX-Request") != ""
		isNotAPI := !strings.HasPrefix(c.Path(), "/api")

		// Skip auth for public paths
		if isPublicPath(c.Path()) {
			log.Debug("Public path, skipping auth check")
			return c.Next()
		}
		if _, ok := checkAuth(c); !ok {
			log.Infof("Unauthorized access to [%s]", c.Path())

			switch {
			case isHTMX:
				var redirectTo string
				if isNotAPI {
					redirectTo = fmt.Sprintf(PathLoginForm, url.QueryEscape(c.OriginalURL()))
				} else {
					redirectTo = fmt.Sprintf(PathLoginForm, "/")
				}
				c.Set("HX-Location", redirectTo)
				log.Infof("Redirect HTMX request to %s", redirectTo)
				return c.SendStatus(fiber.StatusOK)
			case isNotAPI:
				log.Infof("Regular browser request, redirecting to %s", fmt.Sprintf(PathLoginForm, c.Path()))
				return c.Redirect(fmt.Sprintf(PathLoginForm, c.Path()), fiber.StatusFound)
			default:
				log.Infof("API request, returning 401 Unauthorized")
				return fiber.ErrUnauthorized
			}
		}
		log.Debug("✓ Auth check passed ✅")
		return c.Next()
	}
}

func checkAuth(c *fiber.Ctx) (u *h.User, ok bool) {
	if u = h.GetUser(c); u.IsEmpty() {
		c.ClearCookie(h.CookUID, h.CookUName)
		return
	}
	c.Locals(help.CtxIsAuth, true)
	c.Locals(help.CtxUser, u)
	return u, true
}

// Add helper function for public paths
func isPublicPath(path string) bool {
	publicPaths := []string{
		"/login",
		"/login_form",
		"/logout",
		"/health",
		"/metrics",
		"/web/files/",
		"/static/",
	}

	for _, pp := range publicPaths {
		if strings.HasPrefix(path, pp) {
			return true
		}
	}
	return false
}
