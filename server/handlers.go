package server

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"gitlab.dev.ict/golang/go-ai/handlers"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models"
)

func (s *Server) healthCheck(c *fiber.Ctx) error {
	return c.Status(200).SendString("OK")
}

func (s *Server) homeHandler(c *fiber.Ctx) error {
	return c.Render("start", models.NewBpp("START PAGE").Dev(s.config.IsLocalFE), s.config.Layout)
}

func (s *Server) accessDeniedHandler(c *fiber.Ctx) error {
	return c.Status(fiber.StatusForbidden).Render("errors/error_page", fiber.Map{
		"Title":        "Access Denied",
		"ErrorMessage": "Sorry, you do not have permission to access this page.",
		"ErrorCode":    "403",
	}, "layouts/main")
}

func (s *Server) loginFormHandler(c *fiber.Ctx) error {
	return c.Render("login_form", fiber.Map{
		"Title":  "Login",
		"EnvDev": s.config.IsLocalFE,
	}, s.config.Layout)
}

func (s *Server) loginFormOnlyHandler(c *fiber.Ctx) error {
	return c.Render("login_form", fiber.Map{
		"Title":  "Login",
		"EnvDev": s.config.IsLocalFE,
	})
}

func (s *Server) createRouteFE2(path string, bpp *models.BasicPageProps) {
	s.app.Get(path, s.ah.CheckAuth, s.ah.CheckUserRole, func(c *fiber.Ctx) error {
		pageProps := bpp.
			Auth(c.Locals("isAuth").(bool)).
			ULogin(handlers.GetUser(c).Login).
			Dev(s.config.IsLocalFE).
			Eval()
		return c.Render("ai_index", pageProps, s.config.Layout)
	}).Name(bpp.Title)
}

func (s *Server) handleWSAccount(c *fiber.Ctx) error {
	msisdn := c.Params("msisdn", c.Query("msisdn", ""))
	if err := helpers.ValidateMSISDN(msisdn); err != nil {
		return c.Status(400).JSON(&models.Response{Code: 400, Message: err.Error()})
	}

	acc, err := s.config.WsGetter.GetAccount(helpers.Log(c).Ctx, msisdn)
	if err != nil {
		helpers.Log(c).Errorf("wsGetter.GetAccount return err: %v", err)
		return c.Status(404).JSON(&models.Response{Code: 404, Message: "ws errors"})
	}

	helpers.Log(c).Debugf("ACC_INFO=%s", acc.String())
	return c.JSON(acc)
}

func (s *Server) ErrorsHandler(c *fiber.Ctx, err error) error {
	// Custom error handler logic
	s.log.Warn("ERRROOOOOOR HAAAAANDLER")
	code := fiber.StatusInternalServerError
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	return c.Status(code).SendString(err.Error())
}
