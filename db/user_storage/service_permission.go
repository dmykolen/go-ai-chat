package userstorage

import (
	"context"

	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/gorm"
)

// PermissionService handles permission-related operations
type PermissionService struct {
	db      *gorm.DB
	log     *gologgers.Logger
	isDebug bool
}

func NewPermService(db *gorm.DB, logger *gologgers.Logger) *PermissionService {
	return &PermissionService{
		db:      db,
		log:     logger,
		isDebug: false,
	}
}

// func NewPermService(db *gorm.DB, logger *gologgers.Logger, isDebug bool) *PermissionService {
// 	return &PermissionService{
// 		db:      db,
// 		log:     logger,
// 		isDebug: isDebug,
// 	}
// }

func (s *PermissionService) GetPermissions(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	err := s.db.WithContext(createCtx(ctx, s.log)).Find(&permissions).Error
	return permissions, err
}

func (s *PermissionService) CreatePermission(ctx context.Context, permission *Permission) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Create(permission).Error
}

func (s *PermissionService) UpdatePermission(ctx context.Context, permission *Permission) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Save(permission).Error
}

func (s *PermissionService) DeletePermission(ctx context.Context, id uint) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Delete(&Permission{}, id).Error
}
