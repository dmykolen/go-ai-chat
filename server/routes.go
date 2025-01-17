package server

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.dev.ict/golang/go-ai/handlers"
	"gitlab.dev.ict/golang/go-ai/models"
)

func (s *Server) setupRoutes() {
	// Public routes
	s.app.Get("/health", s.healthCheck)
	s.app.Get("/", s.homeHandler)
	s.app.Get("/login_form", s.loginFormHandler)
	s.app.Get("/login_form_only", s.loginFormOnlyHandler)

	// Protected routes
	protected := s.app.Group("", s.ah.CheckAuth)
	protected.Use(s.permissionMiddleware())

	s.setupCoreRoutes(protected)
    s.setupAPIRoutes(protected)
    s.setupWSRoutes(protected)
    s.setupVectorDBRoutes(protected)
    s.setupAccessManagementRoutes(protected)
    s.setupChatRoutes(protected)
}

func (s *Server) setupCoreRoutes(router fiber.Router) {
	router.Get("/access_denied", s.accessDeniedHandler)
	router.Post("/chatgpt", s.ah.ChatGPT)
	router.Get("/sse2", s.ah.SSE)
	router.Post("/announce", s.ah.AnnounceToUsers)
	router.Post("/login", s.ah.Login)
	router.Get("/logout", s.ah.Logout)

	s.createRouteFE2("/ai", models.NewBpp(models.T1))
	s.createRouteFE2("/voip", models.NewBpp(models.T2))
	s.createRouteFE2("/aidb", models.NewBpp(models.T3))
}

func (s *Server) setupWSRoutes(router fiber.Router) {
	ws := router.Group("/ws")
	ws.Get("/account/:msisdn?", s.handleWSAccount)
}

func (s *Server) setupAPIRoutes(router fiber.Router) {
	api := router.Group("/api")
	v1 := api.Group("/v1")
	v2 := api.Group("/v2")

	s.setupV1Routes(v1)
	s.setupV2Routes(v2)
}

func (s *Server) setupV1Routes(api fiber.Router) {
	api.Post("/ask-db", s.ah.AskDB)
	api.Post("/ask-ai-voip", s.ah.AskAiVoIP_NEW)
	api.Post("/stt", s.ah.HandleSTT)
	api.Post("/rate", s.ah.RateChat)
	api.Get("/users/:username", s.ah.GetUserByName)
	api.Get("/users/chats/:username", s.ah.GetUserChats)
	api.Get("/users/:username?/chats/:type", s.ah.GetChats)
	api.Get("/users/photo/:id", s.ah.GetUserPhoto)
}

func (s *Server) setupV2Routes(api fiber.Router) {
	api.Get("/users/:username?/chats/:type", func(c *fiber.Ctx) error {
		return c.JSON(models.RespOK.WithData(s.ah.GetChatHTML3(c)))
	})
	api.Get("/app/store/users", s.ah.GetUsersFromAppStoreForUsers)
	api.Get("/users/chats/:username", s.ah.GetUserChats)
	api.Get("/chats/:uuid?", s.ah.GetChatByUUID)
}

func (s *Server) setupAccessManagementRoutes(router fiber.Router) {
	rbac := router.Group("/access", s.ah.CheckUserRole)
	h := s.ah.RBACHandler()

	// User routes
	rbac.Get("/users", h.GetUsers)
	rbac.Post("/users", h.CreateUser)
	rbac.Put("/users/:id", h.UpdateUser)
	rbac.Delete("/users/:id", h.DeleteUser)

	// Role routes
	rbac.Get("/roles", h.GetRoles)
	rbac.Post("/roles", h.CreateRole)
	rbac.Put("/roles/:id", h.UpdateRole)
	rbac.Delete("/roles/:id", h.DeleteRole)

	// Group routes
	rbac.Get("/groups", h.GetGroups)
	rbac.Post("/groups", h.CreateGroup)
	rbac.Delete("/groups/:id", h.DeleteGroup)

	// Permission routes
	rbac.Get("/permissions", h.GetPermissions)
	rbac.Post("/permissions", h.CreatePermission)
	rbac.Delete("/permissions/:id", h.DeletePermission)
}

func (s *Server) setupChatRoutes(router fiber.Router) {
	// Frontend chat routes
	router.Get("/ffe/users/:username?/chats/:type", func(c *fiber.Ctx) error {
		return c.Render("chat2", s.ah.GetChatHTML3(c))
	})

	router.Get("/fe/users/:username?/chats/:type", func(c *fiber.Ctx) error {
		chats := s.ah.GetChatHTML3(c)
		chats["Login"] = handlers.GetUser(c).Login
		return c.Render("chat2", chats)
	})
}

func (s *Server) setupVectorDBRoutes(router fiber.Router) {
	vdb := handlers.NewVectorDBHandler(s.app, s.log, s.config.Rag, s.config.IsDebug)
	vdb.InitEndpoints(s.ah.CheckAuth, s.ah.CheckUserRole)
}
