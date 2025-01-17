package dto

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
)

func TestRoleTransform(t *testing.T) {
	tests := []struct {
		name  string
		roles []us.Role
		want  []RoleDTO
	}{
		{
			name: "single role with permissions",
			roles: []us.Role{
				{
					ID:          1,
					Name:        "Admin",
					Code:        "admin",
					Description: "Administrator role",
					Permissions: []us.Permission{
						{Code: "read"},
						{Code: "write"},
					},
				},
			},
			want: []RoleDTO{
				{
					ID:          1,
					Name:        "Admin",
					Code:        "admin",
					Description: "Administrator role",
					Permissions: []string{"read", "write"},
				},
			},
		},
		{
			name: "multiple roles",
			roles: []us.Role{
				{
					ID:          1,
					Name:        "Admin",
					Code:        "admin",
					Description: "Administrator role",
					Permissions: []us.Permission{
						{Code: "read"},
						{Code: "write"},
					},
				},
				{
					ID:          2,
					Name:        "User",
					Code:        "user",
					Description: "User role",
					Permissions: []us.Permission{
						{Code: "read"},
					},
				},
			},
			want: []RoleDTO{
				{
					ID:          1,
					Name:        "Admin",
					Code:        "admin",
					Description: "Administrator role",
					Permissions: []string{"read", "write"},
				},
				{
					ID:          2,
					Name:        "User",
					Code:        "user",
					Description: "User role",
					Permissions: []string{"read"},
				},
			},
		},
		{
			name:  "empty roles",
			roles: []us.Role{},
			want:  []RoleDTO{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoleTransform(tt.roles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoleDTO_ToRole(t *testing.T) {
	tests := []struct {
		name     string
		roleDTO  RoleDTO
		expected us.Role
	}{
		{
			name: "Admin role",
			roleDTO: RoleDTO{
				ID:          1,
				Name:        "Admin",
				Code:        "admin",
				Description: "Administrator role",
				Permissions: []string{"read", "write"},
			},
			expected: us.Role{
				ID:          1,
				Name:        "Admin",
				Code:        "admin",
				Description: "Administrator role",
				Permissions: []us.Permission{
					{Code: "read"},
					{Code: "write"},
				},
			},
		},
		{
			name: "User role",
			roleDTO: RoleDTO{
				ID:          2,
				Name:        "User",
				Code:        "user",
				Description: "User role",
				Permissions: []string{},
			},
			expected: us.Role{
				ID:          2,
				Name:        "User",
				Code:        "user",
				Description: "User role",
				Permissions: []us.Permission{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.roleDTO.ToRole()
			assert.Equal(t, tt.expected, *got)
		})
	}
}

func TestGroupDTO_ToGroup(t *testing.T) {
	tests := []struct {
		name     string
		groupDTO GroupDTO
		expected us.Group
	}{
		{
			name: "Admin group",
			groupDTO: GroupDTO{
				ID:          1,
				Name:        "Admin",
				Description: "Administrator group",
				Roles:       []string{"admin"},
			},
			expected: us.Group{
				Name:        "Admin",
				Description: "Administrator group",
				Roles:       []us.Role{{Code: "admin"}},
			},
		},
		{
			name: "User group",
			groupDTO: GroupDTO{
				ID:          2,
				Name:        "User",
				Description: "User group",
				Roles:       []string{},
			},
			expected: us.Group{
				Name:        "User",
				Description: "User group",
				Roles:       []us.Role{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.groupDTO.ToGroup()
			assert.Equal(t, tt.expected, *got)
		})
	}
}

func TestUserDTO_ToUser(t *testing.T) {
	tests := []struct {
		name     string
		userDTO  UserDTO
		expected us.User
	}{
		{
			name: "Admin user",
			userDTO: UserDTO{
				ID:       1,
				Username: "admin",
				Email:    lo.ToPtr(""),
				Status:   "act",
				Roles:    []string{"admin"},
				Groups:   []string{"admin"},
			},
			expected: us.User{
				Username: "admin",
				Email:    lo.ToPtr(""),
				Status:   "act",
				Roles:    []us.Role{{Code: "admin"}},
				Groups:   []us.Group{{Name: "admin"}},
			},
		},
		{
			name: "User",
			userDTO: UserDTO{
				ID:       2,
				Username: "user",
				Email:    lo.ToPtr(""),
				Status:   "act",
				Roles:    []string{},
				Groups:   []string{},
			},
			expected: us.User{
				Username: "user",
				Email:    lo.ToPtr(""),
				Status:   "act",
				Roles:    []us.Role{},
				Groups:   []us.Group{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.userDTO.ToUser()
			assert.Equal(t, tt.expected, *got)
		})
	}
}

func TestPermissionDTO_ToPermission(t *testing.T) {
	tests := []struct {
		name          string
		permissionDTO PermissionDTO
		expected      us.Permission
	}{
		{
			name: "Read permission",
			permissionDTO: PermissionDTO{
				ID:          1,
				Code:        "read",
				Description: "Read permission",
			},
			expected: us.Permission{
				ID:          1,
				Code:        "read",
				Description: "Read permission",
			},
		},
		{
			name: "Write permission",
			permissionDTO: PermissionDTO{
				ID:          2,
				Code:        "write",
				Description: "Write permission",
			},
			expected: us.Permission{
				ID:          2,
				Code:        "write",
				Description: "Write permission",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.permissionDTO.ToPermission()
			assert.Equal(t, tt.expected, *got)
		})
	}
}
