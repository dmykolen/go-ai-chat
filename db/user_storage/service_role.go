package userstorage

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/go-cache"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/gorm"
)

// UserRoleService handles role-related operations
type UserRoleService struct {
	db  *gorm.DB
	log *gologgers.Logger
}

// NewUserRoleService creates a new UserRoleService
func NewUserRoleService(db *gorm.DB, log *gologgers.Logger) *UserRoleService {
	return &UserRoleService{db: db, log: log}
}

func (s *UserRoleService) l(ctx context.Context) *gologgers.LogRec {
	return s.log.RecWithCtx(ctx, CH)
	// return s.log.Rec(CH)
}

// UserRolePermissions contains all role and permission information for a user
type UserRolePermissions struct {
	UserID      uint         `json:"user_id"`
	Roles       []Role       `json:"roles"`
	Permissions []Permission `json:"permissions"`
}

// GetRoles retrieves all roles with optional pagination
func (s *UserRoleService) GetRoles(ctx context.Context, offset, limit int) ([]Role, error) {
	var roles []Role
	query := s.db.WithContext(ctx).Model(&Role{})
	// query := s.db.WithContext(createCtx(ctx, s.log)).Model(&Role{})

	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Preload("Permissions").Find(&roles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get roles: %w", err)
	}

	return roles, nil
}

// CreateRole creates a new role with the given details
func (s *UserRoleService) CreateRole(ctx context.Context, role *Role) error {
	if role.Code == "" || role.Name == "" {
		return fmt.Errorf("role code and name are required")
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if role with same code exists
		var count int64
		if err := tx.Model(&Role{}).Where("code = ?", role.Code).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check role existence: %w", err)
		}

		if count > 0 {
			return fmt.Errorf("role with code %s already exists", role.Code)
		}

		// Create the role
		if err := tx.Create(role).Error; err != nil {
			return fmt.Errorf("failed to create role: %w", err)
		}

		return nil
	})
}

// UpdateRolePermissions updates the permissions for a role
func (s *UserRoleService) UpdateRolePermissions(ctx context.Context, roleID uint, permissionCodes []string) (*Role, error) {
	var role Role
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Verify role exists

		if err := tx.First(&role, roleID).Error; err != nil {
			return fmt.Errorf("role not found: %w", err)
		}
		s.l(ctx).Infof("Role found: %#v", role)

		// Get permissions by codes
		var permissions []Permission
		if len(permissionCodes) > 0 {
			if err := tx.Where("code IN ?", permissionCodes).Find(&permissions).Error; err != nil {
				return fmt.Errorf("failed to find permissions: %w", err)
			}

			// Verify all permissions were found
			if len(permissions) != len(permissionCodes) {
				return fmt.Errorf("some permission codes were not found")
			}
		}

		// Replace existing permissions with new ones
		if err := tx.Model(&role).Association("Permissions").Replace(permissions); err != nil {
			return fmt.Errorf("failed to update role permissions: %w", err)
		}

		s.l(ctx).Infof("Updated permissions for role ID=%d; Perms=[%v]; Role=[%v]", roleID, permissionCodes, role)
		return nil
	})
	return &role, err
}

// UpdateRole updates an existing role
func (s *UserRoleService) UpdateRole(ctx context.Context, role *Role) error {
	return s.db.WithContext(ctx).Save(role).Error
}

// DeleteRole deletes a role and its user/group associations
func (s *UserRoleService) DeleteRole(ctx context.Context, roleID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if role exists
		var role Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return fmt.Errorf("role not found: %w", err)
		}

		// Delete user role associations
		if err := tx.Where("role_id = ?", roleID).Delete(&UserRole{}).Error; err != nil {
			return fmt.Errorf("failed to delete user role associations: %w", err)
		}

		// Delete group role associations
		if err := tx.Where("role_id = ?", roleID).Delete(&GroupRole{}).Error; err != nil {
			return fmt.Errorf("failed to delete group role associations: %w", err)
		}

		// Delete the role
		if err := tx.Delete(&role).Error; err != nil {
			return fmt.Errorf("failed to delete role: %w", err)
		}

		return nil
	})
}

// AddRoleToUser assigns a role to a user
func (s *UserRoleService) AddRoleToUser(ctx context.Context, userID uint, roleCode string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role Role
		if err := tx.Where("code = ?", roleCode).First(&role).Error; err != nil {
			return fmt.Errorf("role not found: %w", err)
		}

		if err := tx.Model(&UserRole{}).
			Where("user_id = ? AND role_id = ?", userID, role.ID).
			FirstOrCreate(&UserRole{UserID: userID, RoleID: role.ID}).Error; err != nil {
			return fmt.Errorf("failed to assign role: %w", err)
		}

		return nil
	})
}

// RemoveRoleFromUser removes a role from a user
func (s *UserRoleService) RemoveRoleFromUser(ctx context.Context, userID uint, roleCode string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role Role
		if err := tx.Where("code = ?", roleCode).First(&role).Error; err != nil {
			return fmt.Errorf("role not found: %w", err)
		}

		result := tx.Where("user_id = ? AND role_id = ?", userID, role.ID).
			Delete(&UserRole{})

		if result.Error != nil {
			return fmt.Errorf("failed to remove role: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("user doesn't have role: %s", roleCode)
		}

		return nil
	})
}

// RemoveAllRolesFromUser removes all roles from a user
func (s *UserRoleService) RemoveAllRolesFromUser(ctx context.Context, userID uint) error {
	// Start transaction
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete all user role associations
		result := tx.Where("user_id = ?", userID).Delete(&UserRole{})
		if result.Error != nil {
			return fmt.Errorf("failed to remove user roles: %w", result.Error)
		}

		// Check if any rows were affected
		if result.RowsAffected == 0 {
			return fmt.Errorf("no roles found for user ID: %d", userID)
		}

		s.l(ctx).Infof("Removed all roles[%d] for user ID: %d", result.RowsAffected, userID)

		return nil
	})
}

// UpdateUserRoles replaces all roles for a user
func (s *UserRoleService) UpdateUserRoles(ctx context.Context, userID uint, roleCodes []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all specified roles
		var roles []Role
		if err := tx.Where("code IN ?", roleCodes).Find(&roles).Error; err != nil {
			return fmt.Errorf("failed to find roles: %w", err)
		}

		if len(roles) != len(roleCodes) {
			return fmt.Errorf("some roles were not found")
		}

		// Get the user
		var user User
		if err := tx.First(&user, userID).Error; err != nil {
			return fmt.Errorf("user not found: %w", err)
		}

		// Replace all roles
		if err := tx.Model(&user).Association("Roles").Replace(roles); err != nil {
			return fmt.Errorf("failed to update roles: %w", err)
		}

		return nil
	})
}

// GetRoleByID
func (s *UserRoleService) GetRoleByID(ctx context.Context, roleID uint) (*Role, error) {
	var role Role
	err := s.db.WithContext(ctx).
		Preload("Permissions").
		Where("id = ?", roleID).
		First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetRole retrieve role witn permissions by code
func (s *UserRoleService) GetRole(ctx context.Context, roleCode string) (*Role, error) {
	var role Role
	err := s.db.WithContext(ctx).
		Preload("Permissions").
		Where("code = ?", roleCode).
		First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetUserRoles retrieves all roles for a user
func (s *UserRoleService) GetUserRoles(ctx context.Context, userID uint) ([]Role, error) {
	var user User
	err := s.db.WithContext(ctx).
		Preload("Roles").
		First(&user, userID).Error
	if err != nil {
		return nil, err
	}
	return user.Roles, nil
}

// GetUserPermissions fetches all permissions for a user, including those from groups
func (s *UserRoleService) GetUserPermissions(ctx context.Context, userID uint) ([]Permission, error) {
	var permissions []Permission

	err := s.db.WithContext(ctx).
		Model(&Permission{}).
		Distinct().
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("LEFT JOIN user_roles ON user_roles.role_id = roles.id AND user_roles.user_id = ?", userID).
		Joins("LEFT JOIN group_roles ON group_roles.role_id = roles.id").
		Joins("LEFT JOIN user_groups ON user_groups.group_id = group_roles.group_id AND user_groups.user_id = ?", userID).
		Where("user_roles.user_id = ? OR user_groups.user_id = ?", userID, userID).
		Find(&permissions).Error

	return permissions, err
}

func (s *UserRoleService) GetUserRolesAndPermissions(ctx context.Context, userID uint) (*UserRolePermissions, error) {
	var user User

	// Using nested preloading to get all relationships in one query
	err := s.db.WithContext(ctx).
		Preload("Roles.Permissions"). // Nested preload to get both roles and their permissions
		First(&user, userID).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get user with roles: %w", err)
	}

	result := &UserRolePermissions{
		UserID:      user.ID,
		Roles:       user.Roles,
		Permissions: make([]Permission, 0),
	}

	// Collect unique permissions from all roles
	permMap := make(map[uint]Permission)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permMap[perm.ID] = perm
		}
	}

	// Convert map to slice
	for _, perm := range permMap {
		result.Permissions = append(result.Permissions, perm)
	}

	return result, nil
}

// HasPermission checks if a user has a specific permission
func (s *UserRoleService) HasPermission(ctx context.Context, userID uint, permissionCode string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("permissions").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ? AND permissions.code = ?", userID, permissionCode).
		Count(&count).Error

	return count > 0, err
}

// HasPermission checks if a user has a specific permission, including group roles
func (s *UserRoleService) HasPermissionIncludingGroups(ctx context.Context, userID uint, permissionCode string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&Permission{}).
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN roles ON roles.id = role_permissions.role_id").
		Joins("LEFT JOIN user_roles ON user_roles.role_id = roles.id AND user_roles.user_id = ?", userID).
		Joins("LEFT JOIN group_roles ON group_roles.role_id = roles.id").
		Joins("LEFT JOIN user_groups ON user_groups.group_id = group_roles.group_id AND user_groups.user_id = ?", userID).
		Where("permissions.code = ? AND (user_roles.user_id = ? OR user_groups.user_id = ?)", permissionCode, userID, userID).
		Count(&count).Error

	return count > 0, err
}

// HasRole checks if a user has a specific role
func (s *UserRoleService) HasRole(ctx context.Context, userID uint, roleCode string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Table("roles").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.code = ?", userID, roleCode).
		Count(&count).Error

	return count > 0, err
}

// HasRoleIncludingGroups checks if a user has a specific role, including group roles
func (s *UserRoleService) HasRoleIncludingGroups(ctx context.Context, userID uint, roleCode string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&Role{}).
		Joins("LEFT JOIN user_roles ON user_roles.role_id = roles.id AND user_roles.user_id = ?", userID).
		Joins("LEFT JOIN group_roles ON group_roles.role_id = roles.id").
		Joins("LEFT JOIN user_groups ON user_groups.group_id = group_roles.group_id AND user_groups.user_id = ?", userID).
		Where("roles.code = ? AND (user_roles.user_id = ? OR user_groups.user_id = ?)", roleCode, userID, userID).
		Count(&count).Error

	return count > 0, err
}

// GetUserPermissionCodes returns just the permission codes for a user
func (s *UserRoleService) GetUserPermissionCodes(ctx context.Context, userID uint) ([]string, error) {
	var codes []string
	err := s.db.WithContext(ctx).
		Model(&Permission{}).
		Select("DISTINCT permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Pluck("code", &codes).Error

	return codes, err
}

// Example usage during AD authentication
func HandleADAuthentication(ctx context.Context, s *UserRoleService, username string, adGroups []string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Find or create user
		var user User
		if err := tx.Where(User{Username: username}).FirstOrCreate(&user).Error; err != nil {
			return err
		}

		// Sync AD groups
		adGroupService := NewGroupService(tx, s.log)
		if err := adGroupService.SyncUserADGroups(ctx, user.ID, adGroups); err != nil {
			return err
		}

		return nil
	})
}

func getUserIDFromContext(c *gin.Context) uint {
	// Implement this based on your auth system
	return 123
}

// Example middleware for permission checking
func PermissionMiddleware(service *UserRoleService, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := getUserIDFromContext(c) // Implement this based on your auth system

		hasPermission, err := service.HasPermission(c.Request.Context(), userID, requiredPermission)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking permissions"})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Usage examples:
func ExampleUsage(db *gorm.DB, userID uint) {
	roleService := NewUserRoleService(db, nil)

	// Example 1: Get all roles for a user
	roles, err := roleService.GetUserRoles(context.Background(), userID)
	if err != nil {
		fmt.Printf("Error getting roles: %v\n", err)
	} else {
		fmt.Printf("Roles for user %d: %+v\n", userID, roles)
	}

	// Example 2: Check if user has specific permission
	hasPermission, err := roleService.HasPermission(context.Background(), userID, "chat:write")
	if err != nil {
		fmt.Printf("Error checking permission: %v\n", err)
	} else {
		fmt.Printf("User %d has 'chat:write' permission: %v\n", userID, hasPermission)
	}

	// Example 3: Get all permission codes
	permissionCodes, err := roleService.GetUserPermissionCodes(context.Background(), userID)
	if err != nil {
		fmt.Printf("Error getting permission codes: %v\n", err)
	} else {
		fmt.Printf("Permission codes for user %d: %+v\n", userID, permissionCodes)
	}

	// Example 4: Get complete role and permission information
	rolePerms, err := roleService.GetUserRolesAndPermissions(context.Background(), userID)
	if err != nil {
		fmt.Printf("Error getting roles and permissions: %v\n", err)
	} else {
		fmt.Printf("Roles and permissions for user %d: %+v\n", userID, rolePerms)
	}
}

// Cache implementation for better performance
type RolePermissionCache struct {
	cache    *cache.Cache
	service  *UserRoleService
	duration time.Duration
}

func NewRolePermissionCache(service *UserRoleService, duration time.Duration) *RolePermissionCache {
	return &RolePermissionCache{
		cache:    cache.New(duration, duration*2),
		service:  service,
		duration: duration,
	}
}

func (c *RolePermissionCache) GetUserPermissions(ctx context.Context, userID uint) ([]Permission, error) {
	cacheKey := fmt.Sprintf("user_permissions_%d", userID)

	if cached, found := c.cache.Get(cacheKey); found {
		return cached.([]Permission), nil
	}

	permissions, err := c.service.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, permissions, c.duration)
	return permissions, nil
}

// Legacy role constants
const (
	RoleUsual = "USUAL"
	RoleAdmin = "ADMIN"
)

// MigrateUserRoles helps migrate users from the legacy role system to the new RBAC system
func MigrateUserRoles(db *gorm.DB) error {
	// First, ensure we have the basic roles created
	basicRoles := map[string]Role{
		RoleUsual: {
			Name:        "Regular User",
			Code:        "user",
			Description: "Regular user with basic access",
			Level:       1,
		},
		RoleAdmin: {
			Name:        "Administrator",
			Code:        "admin",
			Description: "Full system administrator",
			Level:       100,
		},
	}

	// Create or update basic roles
	for _, role := range basicRoles {
		if err := db.Where(Role{Code: role.Code}).FirstOrCreate(&role).Error; err != nil {
			return err
		}
	}

	// Migrate users
	var users []User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		var roleToAssign Role
		if user.LegacyRole == RoleAdmin {
			if err := db.Where("code = ?", "admin").First(&roleToAssign).Error; err != nil {
				return err
			}
		} else {
			if err := db.Where("code = ?", "user").First(&roleToAssign).Error; err != nil {
				return err
			}
		}

		// Assign new role if not already assigned
		if err := db.Model(&user).Association("Roles").Append(&roleToAssign); err != nil {
			return err
		}
	}

	return nil
}
