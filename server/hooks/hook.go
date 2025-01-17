package hooks

import (
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"gitlab.dev.ict/golang/libs/gologgers"
)

func InitFiberHooks(s *fiber.App, log *gologgers.Logger) {
	s.Hooks().OnListen(func(ld fiber.ListenData) error {
		log.Warnf("Server start listening on: %s://%s:%s/", lo.Ternary(ld.TLS, "https", "http"), ld.Host, ld.Port)
		return nil
	})

	s.Hooks().OnRoute(func(r fiber.Route) error {
		if r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "CONNECT" || r.Method == "USE" || r.Method == "TRACE" || r.Method == "PATCH" {
			return nil
		}
		if len(r.Params) > 0 {
			log.Debugf("Register route:[%-6s %s] with params=[%v]", r.Method, r.Path, r.Params)
		} else {
			log.Debugf("Register route:[%-6s %s]", r.Method, r.Path)
		}
		return nil
	})

}
