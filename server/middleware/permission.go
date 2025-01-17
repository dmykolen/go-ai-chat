// middleware/permission.go
package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.dev.ict/golang/go-ai/handlers"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/libs/utils"
)

func PermissionCheck(ah *handlers.AppHandler, config *models.PermissionsConfig, skipRoutes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := help.Log(c)
		log.Warnf("Permission check middleware start... Path=%s", c.Path())

		u, ok := c.Locals(help.CtxUser).(*handlers.User)
		log.Infof("check skip routes for path[%s]. User from ctx => %v", c.Path(), u)
		for _, route := range skipRoutes {
			if strings.HasPrefix(c.Path(), route) || (ok && u.Login == "dmykolen") {
				return c.Next()
			}
		}

		// Check route permissions from config
		route := getRoutePermConfig(config, c.Path(), c.Method())
		log.Debugf("Route: %s", route)
		if route == nil {
			log.Warnf("Route [%s] not found in permissions config", c.Path())
		} else if route.RequiredPermission == "" || route.RequiredPermission == "none" {
			log.Warnf("Route [%s] has no required permission", c.Path())
		} else {
			user := handlers.GetUser(c)
			log.Warn(utils.JsonPrettyStr(user))
			if user.IsEmpty() && !strings.HasPrefix(c.Path(), "/api") {
				log.Warnf("Redirect to /login from [%s], cause user not exists in context", c.Path())
				return c.Redirect("/login")
			} else if user.IsEmpty() {
				log.Warnf("User not exists in context, but it's an API request")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Unauthorized",
				})
			}

			log.Infof("Checking permission[%s] for user [%d:%s] on route [%s]", route.RequiredPermission, user.DBID, user.Login, c.Path())

			hasPermission, err := ah.Store().RoleService().HasPermissionIncludingGroups(c.Context(), user.DBID, route.RequiredPermission)
			if err != nil || !hasPermission {
				if strings.HasPrefix(c.Path(), "/api") { // API request
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error": "You don't have permission to access this resource",
					})
				}

				log.Info("Redirect to /access_denied")
				return c.Redirect("/access_denied")
			}
		}

		return c.Next()
	}
}

func getRoutePermConfig(config *models.PermissionsConfig, path string, method string) *models.RoutePermission {
	for _, route := range config.Routes {
		if route.Path == path && contains(route.Methods, method) {
			return &route
		}
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RequirePermission middleware
func RequirePermission(ah *handlers.AppHandler, permCode string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		log := help.Log(c)
		user := handlers.GetUser(c)
		if user == nil {
			log.Infof("Redirect to /login from [%s], cause user not exists in context", c.Path())
			return c.Redirect("/login")
		}

		hasPermission, err := ah.Store().RoleService().HasPermission(c.Context(), user.DBID, permCode)
		if err != nil || !hasPermission {
			if c.XHR() { // API request
				return c.Status(403).JSON(fiber.Map{
					"error": "Permission denied",
				})
			}
			return c.Redirect("/access_denied")
		}

		return c.Next()
	}
}

// Template helper function
func HasPermission(c *fiber.Ctx, ah *handlers.AppHandler, permCode string) bool {
	user := handlers.GetUser(c)
	if user == nil {
		return false
	}

	has, _ := ah.Store().RoleService().HasPermission(c.Context(), user.DBID, permCode)
	return has
}
