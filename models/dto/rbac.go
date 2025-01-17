package dto

import (
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	"gorm.io/gorm"
)

type UserDTO struct {
	ID       uint     `json:"id"`
	Username string   `json:"username"`
	Email    *string  `json:"email"`
	Status   string   `json:"status"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
}

type UserDTOArr []UserDTO

func UsersTransform(user []us.User) (u []UserDTO) {
	for _, user := range user {
		ur := UserDTO{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Status:   user.Status,
		}
		for _, role := range user.Roles {
			ur.Roles = append(ur.Roles, role.Code)
		}
		for _, group := range user.Groups {
			ur.Groups = append(ur.Groups, group.Name)
		}
		u = append(u, ur)
	}
	return
}

func (userDTO *UserDTO) ToUser() *us.User {
	user := us.User{
		Model:    gorm.Model{ID: userDTO.ID},
		Username: userDTO.Username,
		Email:    userDTO.Email,
		Status:   userDTO.Status,
	}

	// Convert role codes to Role objects
	for _, roleCode := range userDTO.Roles {
		user.Roles = append(user.Roles, us.Role{Code: roleCode})
	}

	// Convert group names to Group objects
	for _, groupName := range userDTO.Groups {
		user.Groups = append(user.Groups, us.Group{Name: groupName})
	}

	return &user
}

type RoleDTO struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Add a method to Role struct to convert to RoleResponse
func RoleTransform(roles []us.Role) (r []RoleDTO) {
	for _, role := range roles {
		rr := RoleDTO{
			ID:          role.ID,
			Name:        role.Name,
			Code:        role.Code,
			Description: role.Description,
		}
		for _, permission := range role.Permissions {
			rr.Permissions = append(rr.Permissions, permission.Code)
		}
		r = append(r, rr)
	}
	return
}

func (roleDTO *RoleDTO) ToRole() *us.Role {
	role := us.Role{
		ID:          roleDTO.ID,
		Name:        roleDTO.Name,
		Code:        roleDTO.Code,
		Description: roleDTO.Description,
	}

	// Convert permission codes to Permission objects
	for _, permCode := range roleDTO.Permissions {
		role.Permissions = append(role.Permissions, us.Permission{Code: permCode})
	}

	return &role
}

// Add these new DTO structs to rbac.go
type PermissionDTO struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

// Add these transform functions
func PermissionTransform(permissions []us.Permission) (p []PermissionDTO) {
	for _, perm := range permissions {
		p = append(p, PermissionDTO{
			ID:          perm.ID,
			Name:        perm.Name,
			Code:        perm.Code,
			Description: perm.Description,
		})
	}
	return
}

func (permissionDTO *PermissionDTO) ToPermission() *us.Permission {
	return &us.Permission{
		ID:          permissionDTO.ID,
		Name:        permissionDTO.Name,
		Code:        permissionDTO.Code,
		Description: permissionDTO.Description,
	}
}

type GroupDTO struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Roles       []string `json:"roles"`
}

func GroupTransform(groups []us.Group) (g []GroupDTO) {
	for _, group := range groups {
		gd := GroupDTO{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Roles:       make([]string, 0),
		}

		// Extract role codes for each group
		for _, role := range group.Roles {
			gd.Roles = append(gd.Roles, role.Code)
		}

		g = append(g, gd)
	}
	return
}

func (groupDTO *GroupDTO) ToGroup() *us.Group {
	group := us.Group{
		Model:       gorm.Model{ID: groupDTO.ID},
		Name:        groupDTO.Name,
		Description: groupDTO.Description,
	}

	// Convert role codes to Role objects
	for _, roleCode := range groupDTO.Roles {
		group.Roles = append(group.Roles, us.Role{Code: roleCode})
	}

	return &group
}
