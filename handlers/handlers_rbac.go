package handlers

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	m "gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/go-ai/models/dto"
	"gitlab.dev.ict/golang/libs/gologgers"
)

// User represents a user in the system
type UserFE struct {
	ID       int      `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Status   string   `json:"status"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
}

func newUserFE(user []us.User) []UserFE {
	uarr := make([]UserFE, len(user))
	for i, user := range user {
		uarr[i] = UserFE{
			ID:       int(user.ID),
			Username: user.Username,
			Email:    *user.Email,
			Status:   user.Status,
		}

		for _, role := range user.Roles {
			uarr[i].Roles = append(uarr[i].Roles, role.Code)
		}
		for _, group := range user.Groups {
			uarr[i].Groups = append(uarr[i].Groups, group.Name)
		}
	}
	return uarr
}

// Role represents a role in the system
type Role struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Group represents a group in the system
type Group struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

// Permission represents a permission in the system
type Permission struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type Option func(*RBACHandler)

// WithLogger sets the logger for the RBAC handler
func WithLog(log *gologgers.Logger) Option {
	return func(h *RBACHandler) {
		h.log = log
	}
}

// WithUserStorage sets the user storage service
func WithStore(ss us.UserStorage) Option {
	return func(h *RBACHandler) {
		h.ss = ss
	}
}

type RBACHandler struct {
	log *gologgers.Logger
	ss  us.UserStorage
	rs  *us.UserRoleService
	gs  *us.GroupService
	ps  *us.PermissionService
}

func NewRBACHandler(opts ...Option) *RBACHandler {
	h := &RBACHandler{}
	for _, opt := range opts {
		opt(h)
	}
	if h.log == nil || h.ss == nil {
		panic("logger and user storage service are required")
	}

	h.rs = h.ss.RoleService()
	h.gs = h.ss.GroupService()
	h.ps = h.ss.PermService()
	return h
}

func (h *RBACHandler) handleError(c *fiber.Ctx, e string, err error) error {
	h.log.Errorf("%s: %v", e, err)
	return c.Status(fiber.StatusBadRequest).JSON(m.RErr(e, err))
}

func (h *RBACHandler) GetUsers(c *fiber.Ctx) error {
	users, err := h.ss.GetUsers(c.Context())
	if err != nil {
		return h.handleError(c, "Failed to get users", err)
	}
	return c.JSON(dto.UsersTransform(users))
}

func (h *RBACHandler) CreateUser(c *fiber.Ctx) error {
	// var input CreateUserRequest
	var input dto.UserDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse user input", err)
	}
	h.log.Infof("Create user: %v", input)

	// Create user
	user := &us.User{
		Username: input.Username,
		Email:    input.Email,
		Password: "",
	}

	if err := h.ss.CreateUser(c.Context(), user); err != nil {
		return h.handleError(c, "Failed to create user", err)
	}

	// Assign roles
	for _, roleCode := range input.Roles {
		if err := h.rs.AddRoleToUser(c.Context(), user.ID, roleCode); err != nil {
			h.log.Errorf("Failed to assign role %s: %v", roleCode, err)
			continue
		}
	}

	// Assign groups
	if len(input.Groups) > 0 {
		if err := h.ss.UpdateUserGroups(c.Context(), user.ID, input.Groups); err != nil {
			return h.handleError(c, "Failed to assign groups", err)
		}
	}

	return c.JSON(fiber.Map{"status": "success", "message": "user success created", "user": user})
}

func (h *RBACHandler) UpdateUser(c *fiber.Ctx) error {
	// Get user ID from params
	id, err := c.ParamsInt("id")
	if err != nil {
		h.handleError(c, "Invalid user ID", err)
	}

	// Parse request body
	var input dto.UserDTO
	if err := c.BodyParser(&input); err != nil {
		h.handleError(c, "Failed to parse user input", err)
	}

	user, err := h.ss.GetUser(c.Context(), uint(id))
	if err != nil {
		return err
	}

	user.Username = input.Username
	user.Email = input.Email
	user.Status = input.Status

	// Update user
	if err := h.ss.UpdateUser(c.Context(), user); err != nil {
		h.log.Errorf("Failed to update user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	// Update roles if provided
	if len(input.Roles) > 0 {
		if err := h.rs.UpdateUserRoles(c.Context(), user.ID, input.Roles); err != nil {
			return h.handleError(c, "Failed to remove existing roles", err)
		}
	}

	if len(input.Groups) > 0 {
		if err := h.ss.UpdateUserGroups(c.Context(), user.ID, input.Groups); err != nil {
			return h.handleError(c, "Failed to update user groups", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "User updated successfully",
		"user":    user,
	})
}

func (h *RBACHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		h.log.Errorf("Failed to parse user ID: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if err := h.ss.DeleteUser(c.Context(), uint(id)); err != nil {
		h.log.Errorf("Failed to delete user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete user"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *RBACHandler) DeleteRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.handleError(c, "Failed to parse user ID", err)
	}
	if err := h.rs.DeleteRole(c.Context(), uint(id)); err != nil {
		return h.handleError(c, "Failed to delete role", err)
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *RBACHandler) DeleteGroup(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		h.log.Errorf("Failed to parse user ID: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if err := h.gs.DeleteGroup(c.Context(), uint(id)); err != nil {
		h.log.Errorf("Failed to delete group: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete group"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *RBACHandler) DeletePermission(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		h.log.Errorf("Failed to parse user ID: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	if err := h.ps.DeletePermission(c.Context(), uint(id)); err != nil {
		h.log.Errorf("Failed to delete permission: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete permission"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *RBACHandler) GetRoles(c *fiber.Ctx) error {
	roles, err := h.rs.GetRoles(c.Context(), 0, 0)
	if err != nil {
		return h.handleError(c, "Failed to get roles", err)
	}
	return c.JSON(dto.RoleTransform(roles))
}

func (h *RBACHandler) CreateRole(c *fiber.Ctx) error {
	var input dto.RoleDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse role input", err)
	}

	// Validation
	if input.Name == "" || input.Code == "" {
		return h.handleError(c, "Name and Code are required", errors.New("missing required fields"))
	}

	// Convert DTO to role model
	role := input.ToRole()

	// Create role
	err := h.rs.CreateRole(c.Context(), role)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return h.handleError(c, "Role with this code already exists", err)
		}
		return h.handleError(c, "Failed to create role", err)
	}

	var updatedRole *us.Role

	// If permissions were specified, assign them
	if len(input.Permissions) > 0 {
		if updatedRole, err = h.rs.UpdateRolePermissions(c.Context(), role.ID, input.Permissions); err != nil {
			h.log.Warnf("Failed to assign permissions to role: %v", err)
		}
	}

	// Get updated role with permissions
	// updatedRole, err := h.rs.GetRoleByID(c.Context(), role.ID)
	// if err != nil {
	// 	return h.handleError(c, "Failed to get created role", err)
	// }

	return c.Status(fiber.StatusCreated).JSON(dto.RoleTransform([]us.Role{*updatedRole}))
}

func (h *RBACHandler) _CreateRole(c *fiber.Ctx) error {
	var input dto.RoleDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse role input", err)
	}
	h.log.Infof("Create role: %v", input)

	role := input.ToRole()

	if err := h.rs.CreateRole(c.Context(), role); err != nil {
		return h.handleError(c, "Failed to create role", err)
	}

	return c.JSON(fiber.Map{"status": "success", "message": "role created successfully", "role": role})
}

func (h *RBACHandler) UpdateRole(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return h.handleError(c, "Invalid role ID", err)
	}

	var input dto.RoleDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse role input", err)
	}

	role, err := h.rs.GetRoleByID(c.Context(), uint(id))
	if err != nil {
		return h.handleError(c, "Failed to get role", err)
	}

	role.Name = input.Name
	role.Code = input.Code
	role.Description = input.Description

	if err := h.rs.UpdateRole(c.Context(), input.ToRole()); err != nil {
		return h.handleError(c, "Failed to update role", err)
	}

	return c.JSON(fiber.Map{"status": "success", "message": "role updated successfully", "role": role})
}

func (h *RBACHandler) GetGroups(c *fiber.Ctx) error {
	groups, err := h.gs.GetGroups(c.Context())
	if err != nil {
		return h.handleError(c, "Failed to get groups", err)
	}
	h.log.Warnf("Groups: %v", groups)
	result := dto.GroupTransform(groups)
	if result == nil {
		result = []dto.GroupDTO{} // Return empty array instead of null
	}
	return c.JSON(result)
}

func (h *RBACHandler) CreateGroup(c *fiber.Ctx) error {
	var input dto.GroupDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse group input", err)
	}
	h.log.Infof("Create group: %v", input)

	group := input.ToGroup()

	if err := h.gs.CreateOrUpdate(c.Context(), group); err != nil {
		return h.handleError(c, "Failed to create group", err)
	}
	return c.Status(fiber.StatusCreated).JSON(dto.GroupTransform([]us.Group{*group}))
}

func (h *RBACHandler) GetPermissions(c *fiber.Ctx) error {
	permissions, err := h.ps.GetPermissions(c.Context())
	if err != nil {
		return h.handleError(c, "Failed to get permissions", err)
	}
	return c.JSON(dto.PermissionTransform(permissions))
}

func (h *RBACHandler) CreatePermission(c *fiber.Ctx) error {
	var input dto.PermissionDTO
	if err := c.BodyParser(&input); err != nil {
		return h.handleError(c, "Failed to parse permission input", err)
	}
	h.log.Infof("Create permission: %v", input)

	permission := &us.Permission{
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
	}

	if err := h.ps.CreatePermission(c.Context(), permission); err != nil {
		return h.handleError(c, "Failed to create permission", err)
	}

	// return c.JSON(fiber.Map{"status": "success", "message": "permission created successfully", "permission": permission})
	return c.Status(fiber.StatusCreated).JSON([]us.Permission{*permission})

}
