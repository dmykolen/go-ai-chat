package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"gitlab.dev.ict/golang/go-ai/helpers"
	m "gitlab.dev.ict/golang/go-ai/models"
)

func cookiesUserUpdate(c *fiber.Ctx, u *User, duration time.Duration) {
	c.Cookie(&fiber.Cookie{Name: CookUID, Value: u.UUID, Expires: time.Now().Add(duration)})
	c.Cookie(&fiber.Cookie{Name: CookUName, Value: u.Login, Expires: time.Now().Add(duration)})
}

func renderTemplate(c *fiber.Ctx, templateName string, data interface{}, layouts ...string) error {
	err := c.Render(templateName, data, layouts...)
	if err != nil {
		helpers.Log(c).Info("Error rendering template: ", templateName, err)
		return m.NewRenderingError(err)
	}
	return nil
}

func RenderTemplate(c *fiber.Ctx, templateName string, data interface{}, layouts ...string) error {
	return renderTemplate(c, templateName, data, layouts...)
}
