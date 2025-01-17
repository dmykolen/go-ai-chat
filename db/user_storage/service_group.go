package userstorage

import (
	"context"
	"fmt"

	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/gorm"
)

// GroupService handles group-related operations
type GroupService struct {
	db  *gorm.DB
	log *gologgers.Logger
}

func NewGroupService(db *gorm.DB, logger *gologgers.Logger) *GroupService {
	return &GroupService{
		db:  db,
		log: logger,
	}
}

func (s *GroupService) CreateOrUpdate(ctx context.Context, g *Group) error {
	var roles []string
	for _, role := range g.Roles {
		roles = append(roles, role.Code)
	}
	gr, err := s.CreateOrUpdateGroup(ctx, g.Name, g.Description, roles...)
	*g = *gr
	return err
	// return s.db.WithContext(ctx).Save(group).Error
}

// DeleteGroup deletes a group and its associations
func (s *GroupService) DeleteGroup(ctx context.Context, groupID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if group exists
		var group Group
		if err := tx.First(&group, groupID).Error; err != nil {
			return fmt.Errorf("group not found: %w", err)
		}

		// Delete group roles associations
		if err := tx.Where("group_id = ?", groupID).Delete(&GroupRole{}).Error; err != nil {
			return fmt.Errorf("failed to delete group role associations: %w", err)
		}

		// Delete user group associations
		if err := tx.Where("group_id = ?", groupID).Delete(&UserGroup{}).Error; err != nil {
			return fmt.Errorf("failed to delete user group associations: %w", err)
		}

		// Delete the group
		if err := tx.Delete(&group).Error; err != nil {
			return fmt.Errorf("failed to delete group: %w", err)
		}

		return nil
	})
}

// CreateOrUpdateGroup creates or updates a group in the database
// CreateOrUpdateGroup creates or updates a group and assigns roles
func (s *GroupService) CreateOrUpdateGroup(ctx context.Context, name string, description string, roleCodes ...string) (*Group, error) {
	var group *Group

	err := s.db.WithContext(createCtx(ctx, s.log)).Transaction(func(tx *gorm.DB) error {
		// Find or create group
		group = &Group{
			Name:        name,
			Description: description,
		}
		if err := tx.Where(Group{Name: name}).FirstOrCreate(group).Error; err != nil {
			return fmt.Errorf("failed to create/update group: %w", err)
		}

		// If role codes provided, update roles
		if len(roleCodes) > 0 {
			// Find roles by codes
			var roles []Role
			if err := tx.Where("code IN ?", roleCodes).Find(&roles).Error; err != nil {
				return fmt.Errorf("failed to find roles: %w", err)
			}

			// Verify all roles were found
			if len(roles) != len(roleCodes) {
				return fmt.Errorf("some role codes were not found")
			}

			// Replace group's roles
			if err := tx.Model(group).Association("Roles").Replace(roles); err != nil {
				return fmt.Errorf("failed to update group roles: %w", err)
			}
		}

		return nil
	})

	return group, err
}

func (s *GroupService) _CreateOrUpdateGroup(ctx context.Context, name string, description string) (*Group, error) {
	group := &Group{
		Name:        name,
		Description: description,
	}

	err := s.db.WithContext(createCtx(ctx, s.log)).
		Where(Group{Name: name}).
		FirstOrCreate(group).Error

	return group, err
}

func (s *GroupService) GetGroupsSimple(ctx context.Context) ([]Group, error) {
	var groups []Group
	err := s.db.WithContext(ctx).Find(&groups).Error

	return groups, err
}

// GetGroups retrieves all groups from the database
func (s *GroupService) GetGroups(ctx context.Context) ([]Group, error) {
	var groups []Group
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Preload("Roles").
		// Preload("Users").
		Find(&groups).Error

	return groups, err
}

// AssignRoleToGroup assigns a role to a group
func (s *GroupService) AssignRoleToGroup(ctx context.Context, groupID uint, roleCode string) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Transaction(func(tx *gorm.DB) error {
		var role Role
		if err := tx.Where("code = ?", roleCode).First(&role).Error; err != nil {
			return fmt.Errorf("role not found: %w", err)
		}

		var group Group
		if err := tx.First(&group, groupID).Error; err != nil {
			return fmt.Errorf("group not found: %w", err)
		}

		return tx.Model(&group).Association("Roles").Append(&role)
	})
}

func (s *GroupService) SyncUserADGroups(ctx context.Context, userID uint, groupNames []string) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Transaction(func(tx *gorm.DB) error {
		// Get or create groups
		var groups []Group
		for _, name := range groupNames {
			var group Group
			if err := tx.Where(Group{Name: name}).
				FirstOrCreate(&group, Group{Name: name}).Error; err != nil {
				return err
			}
			groups = append(groups, group)
		}

		// Get the user
		var user User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}

		// Replace user's groups
		if err := tx.Model(&user).Association("Groups").Replace(groups); err != nil {
			return err
		}

		return nil
	})
}

// Add method to get user's groups
func (s *GroupService) GetUserGroups(ctx context.Context, userID uint) ([]string, error) {
	var groupNames []string
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Table("groups").
		Joins("JOIN user_groups ON groups.id = user_groups.group_id").
		Where("user_groups.user_id = ?", userID).
		Pluck("groups.name", &groupNames).Error

	return groupNames, err
}

// Add method to check if user belongs to a specific group
func (s *GroupService) IsUserInGroup(ctx context.Context, userID uint, groupName string) (bool, error) {
	var count int64
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Table("user_groups").
		Joins("JOIN ad_groups ON user_groups.group_id = ad_groups.id").
		Where("user_groups.user_id = ? AND ad_groups.name = ?", userID, groupName).
		Count(&count).Error

	return count > 0, err
}
