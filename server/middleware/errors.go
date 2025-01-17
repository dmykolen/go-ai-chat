package middleware

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	h "gitlab.dev.ict/golang/go-ai/helpers"
	m "gitlab.dev.ict/golang/go-ai/models"
)

func ErrorsMiddleware(c *fiber.Ctx) error {
	err := c.Next()
	if err == nil {
		return nil
	}

	l := h.Log(c)
	l.Warn("Start error handling")
	defer l.Warn("End error handling")

	var e *fiber.Error
	if errors.As(err, &e) {
		l.Errorf("Fiber Error: %s", e.Message)
		return c.Status(e.Code).JSON(m.RErr(e.Message, e))
	}

	var ie *m.InternalError
	if errors.As(err, &ie) {
		l.Errorf("Internal Error: %s", ie.Message)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	switch e := err.(type) {
	case *m.RenderingError:
		l.Errorf("Error during rendering: %v", e.OriginalError)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering page")
	case *m.NotFoundError:
		l.Errorf("Not Found Error: %s", e.Error())
		return c.Status(fiber.StatusNotFound).SendString("Page not found")
	default:
		if errors.Is(err, m.ErrTemplateNotFound) {
			l.Errorf("Template not found error: %v", err)
			return c.Status(fiber.StatusNotFound).SendString(err.Error())
		}
		l.Errorf("An unexpected error occurred: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("An unexpected error occurred")
	}
}
