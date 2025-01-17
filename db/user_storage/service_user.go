package userstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.dev.ict/golang/go-ai/helpers/tools"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"gitlab.dev.ict/golang/go-ai/db"
)

const CH = "userDB"

// StorageService is a Postgres implementation of the StorageService interface
type StorageService struct {
	db      *gorm.DB
	log     *gologgers.Logger
	isDebug bool
}

// NewStorageService creates a new instance of StorageService
func NewStorageService(db *gorm.DB, log *gologgers.Logger, isDebug bool) *StorageService {
	return &StorageService{
		db:      db,
		log:     log,
		isDebug: isDebug,
	}
}

func createCtx(ctx context.Context, log *gologgers.Logger) context.Context {
	return context.WithValue(ctx, db.KeyLog, log.RecWithCtx(ctx, CH))
}

func (s *StorageService) Db() *gorm.DB                            { return s.db }
func (s *StorageService) RoleService() *UserRoleService           { return NewUserRoleService(s.db, s.log) }
func (s *StorageService) GroupService() *GroupService             { return NewGroupService(s.db, s.log) }
func (s *StorageService) PermService() *PermissionService         { return NewPermService(s.db, s.log) }
func (s *StorageService) l(ctx context.Context) *gologgers.LogRec { return s.log.RecWithCtx(ctx, CH) }
func (s *StorageService) ctx(ctx context.Context) context.Context { return createCtx(ctx, s.log) }

// CreateUser implements the CreateUser method of the UserStorage interface
func (s *StorageService) CreateUser(ctx context.Context, user *User) error {
	return s.db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_login_time"}),
		}).
		WithContext(createCtx(ctx, s.log)).Create(user).Error
}

// Updated CreateUserWithPhoto
func (s *StorageService) CreateUserWithPhoto(ctx context.Context, user *User, photoData []byte) error {
	// Create user first
	err := s.db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoUpdates: clause.AssignmentColumns([]string{"last_login_time"}),
		}).
		WithContext(createCtx(ctx, s.log)).Create(user).Error

	if err != nil {
		return err
	}

	// Process and update photo if provided
	return s.ProcessAndUpdateUserPhoto(ctx, user.ID, photoData)
}

// ProcessAndUpdateUserPhoto handles photo processing and updating for a user
func (s *StorageService) ProcessAndUpdateUserPhoto(ctx context.Context, userID uint, photoData []byte) error {
	if len(photoData) == 0 {
		return nil
	}

	l := s.log.RecWithCtx(ctx, CH)

	// Detect image type
	mimeType, err := tools.DetectImageType(photoData)
	if err != nil {
		return fmt.Errorf("failed to detect image type: %w", err)
	}
	l.Debugf("Detected image type: %s; Size: %d", mimeType, len(photoData))

	// Compress image
	photoData, err = tools.CompressImage(photoData)
	if err != nil {
		l.Errorf("failed to compress image: %w", err)
		return fmt.Errorf("failed to compress image: %w", err)
	}
	l.Debugf("Compressed image size: %d", len(photoData))

	// Update user's photo
	return s.UpdateUserPhoto(ctx, userID, photoData, mimeType)
}

// GetUser implements the GetUser method of the UserStorage interface
func (s *StorageService) GetUser(ctx context.Context, id uint) (*User, error) {
	var user User
	err := s.db.WithContext(createCtx(ctx, s.log)).Preload("Chats").First(&user, id).Error
	return &user, err
}

func (s *StorageService) GetSoftDeletedUser(ctx context.Context, id uint) (*User, error) {
	var user User
	// Use Unscoped() to include soft-deleted records in the query
	err := s.db.WithContext(createCtx(ctx, s.log)).Unscoped().Preload("Chats").First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUser implements the GetUser method of the UserStorage interface
func (s *StorageService) GetUserFull(ctx context.Context, id uint) (*User, error) {
	var user User
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Preload("Roles.Permissions").
		Preload("Chats").
		Preload("Groups.Roles").
		Preload("Photo").
		First(&user, id).Error
	return &user, err
}

// GetUsers implements the GetUsers method of the UserStorage interface
func (s *StorageService) _GetUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := s.db.WithContext(createCtx(ctx, s.log)).Preload("Chats").Find(&users).Error
	return users, err
}

func (s *StorageService) GetUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := s.db.WithContext(createCtx(ctx, s.log)).Preload("Roles").Preload("Groups").Find(&users).Error
	return users, err
}

func (s *StorageService) GetUsersAM(ctx context.Context) ([]UserResponse, error) {
	var users []User
	if err := s.db.Preload("Roles").Preload("Groups").Find(&users).Error; err != nil {
		return nil, err
	}

	response := make([]UserResponse, len(users))
	for i, user := range users {
		response[i] = user.ToResponse()
	}
	return response, nil
}

func (s *StorageService) GetUserPhoto(ctx context.Context, userID uint) (*UserPhoto, error) {
	var photo UserPhoto
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Where("user_id = ?", userID).
		First(&photo).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if photo not found
		}
		return nil, fmt.Errorf("failed to get user photo: %w", err)
	}

	return &photo, nil
}

// UpdateUser implements the UpdateUser method of the UserStorage interface
func (s *StorageService) UpdateUser(ctx context.Context, user *User) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Save(user).Error
}

func (gs *StorageService) UpdateUserGroups(ctx context.Context, userID uint, groupsName []string) error {
	return gs.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all specified groups
		var groups []Group
		if err := tx.Where("name IN (?)", groupsName).Find(&groups).Error; err != nil {
			return fmt.Errorf("failed to find groups: %w", err)
		}

		if len(groups) != len(groupsName) {
			return fmt.Errorf("some groups were not found")
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

func (s *StorageService) DeleteUserWithOption(ctx context.Context, userID uint, isHard ...bool) error {
	// Start a transaction for consistent deletion
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("user not found: %w", err)
			}
			return err
		}

		// Log deletion action for auditing
		s.l(ctx).Infof("Deleting user: %d (%s)", user.ID, user.Username)

		// tx = tx.WithContext(s.ctx(ctx))
		// Hard delete user
		if len(isHard) > 0 && isHard[0] {
			tx = tx.Unscoped()
		}

		// Soft delete user
		if err := tx.Delete(&user).Error; err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})
}

// DeleteUser implements the DeleteUser method of the UserStorage interface
func (s *StorageService) DeleteUser(ctx context.Context, id uint) error {
	// return s.db.WithContext(createCtx(ctx, s.log)).Delete(&User{}, id).Error
	// return s.db.Select(clause.Associations).WithContext(createCtx(ctx, s.log)).Delete(&User{}, id).Error
	// return s.db.Unscoped().WithContext(createCtx(ctx, s.log)).Delete(&User{}, id).Error
	res := s.db.WithContext(s.ctx(ctx)).Select("Chats").Delete(&User{Model: gorm.Model{ID: id}})

	return res.Error
}

func (s *StorageService) DeleteUser2(ctx context.Context, id uint) error {
	// Start a transaction
	return s.db.WithContext(createCtx(ctx, s.log)).Transaction(func(tx *gorm.DB) error {
		// Delete all related records from other tables
		if err := tx.Where("user_id = ?", id).Delete(&Chat{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&UserPhoto{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", id).Delete(&UserGroup{}).Error; err != nil {
			return err
		}

		// Finally, delete the user
		if err := tx.Delete(&User{}, id).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *StorageService) UpdateUserPhoto(ctx context.Context, userID uint, photoData []byte, mimeType string) error {
	photo := UserPhoto{
		UserID:   userID,
		Data:     photoData,
		MimeType: mimeType,
	}
	return s.db.WithContext(createCtx(ctx, s.log)).Save(&photo).Error
}

// updateUserEmail updates an existing user's email
func (s *StorageService) UpdateUserEmail(ctx context.Context, userID uint, newEmail string) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Model(&User{}).Where("id = ?", userID).Update("email", newEmail).Error
}

// findUserByUsername finds a user by their username
func (s *StorageService) FindUserByUsername(ctx context.Context, username string, withChats ...bool) (*User, error) {
	var user User
	tx := s.db.WithContext(createCtx(ctx, s.log)).Where("username = ?", username)
	if len(withChats) > 0 && withChats[0] {
		tx = tx.Preload("Chats").First(&user)
	} else {
		tx = tx.First(&user)
	}
	return &user, tx.Error
}

// CreateChat implements the CreateChat method of the UserStorage interface
func (s *StorageService) CreateChat(ctx context.Context, chat *Chat) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Create(chat).Error
}

// GetChats implements the GetChats method of the UserStorage interface
func (s *StorageService) GetChats(ctx context.Context, userID uint) ([]Chat, error) {
	var chats []Chat
	err := s.db.WithContext(createCtx(ctx, s.log)).Where("user_id = ?", userID).Find(&chats).Error
	return chats, err
}

func (s *StorageService) GetAllChats(ctx context.Context) (c []Chat, err error) {
	err = s.db.WithContext(createCtx(ctx, s.log)).Find(&c).Error
	return
}

func (s *StorageService) GetUserWithChatsByUserName(ctx context.Context, userName string) (*User, error) {
	var user User
	err := s.db.WithContext(createCtx(ctx, s.log)).Where("username = ?", userName).Preload("Chats").First(&user)
	return &user, err.Error
}

// UpdateChat implements the UpdateChat method of the UserStorage interface
func (s *StorageService) UpdateChat(ctx context.Context, chat *Chat) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Save(chat).Error
}

// DeleteChat implements the DeleteChat method of the UserStorage interface
func (s *StorageService) DeleteChat(ctx context.Context, chatID uint) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Delete(&Chat{}, chatID).Error
}

// DeleteAllChatsForUser deletes all chats for a given user
func (s *StorageService) DeleteAllChatsForUser(ctx context.Context, userID uint) error {
	return s.db.WithContext(createCtx(ctx, s.log)).Where("user_id = ?", userID).Delete(&Chat{}).Error
}

// CreateChat creates a new chat for the given user
func (s *StorageService) CreateChatForUsername(ctx context.Context, username string, chat *Chat) error {
	var user User
	if err := s.db.WithContext(createCtx(ctx, s.log)).Where("username = ?", username).First(&user).Error; err != nil {
		return err
	}

	chat.UserID = user.ID
	return s.db.WithContext(createCtx(ctx, s.log)).Create(chat).Error
}

// GetChatByUUID retrieves a chat by its UUID
func (s *StorageService) _GetChatByUUID(userID uint, uuid string) (*Chat, error) {
	var chat Chat

	// Use GORM's Preload and Where with JSON query to find the chat
	err := s.db.
		// Preload(clause.Associations).
		Where("user_id = ?", userID).
		Where("chat->>'id' = ?", uuid).
		First(&chat).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil if chat not found
		}
		return nil, err
	}

	return &chat, nil
}

func (s *StorageService) GetChatByUUID(ctx context.Context, uuid string) (chat *Chat, err error) {
	err = s.db.WithContext(createCtx(ctx, s.log)).
		Where(datatypes.JSONQuery("chat").Equals(uuid, "id")).
		First(&chat).
		Error
	return
}

func (s *StorageService) UpdateChatFieldInChat(ctx context.Context, chatID uint, field string, newValue interface{}) error {
	path := fmt.Sprintf("{%s}", field)
	_, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("failed to marshal new value: %w", err)
	}
	s.log.Warnf("JSONSet: %#v", datatypes.JSONSet("chat").Set(path, newValue))
	return s.db.WithContext(createCtx(ctx, s.log)).
		Model(&Chat{}).
		Where("id = ?", chatID).
		UpdateColumn("chat", datatypes.JSONSet("chat").Set(path, newValue)).
		UpdateColumn("chat", datatypes.JSONSet("chat").Set("{lastUpdateTime}", time.Now().UnixMilli())).
		Error
}

func (s *StorageService) UpdateMessageFieldInChat(ctx context.Context, chatID uint, messageIndex int, field string, newValue interface{}) error {
	if messageIndex < 0 {
		return fmt.Errorf("invalid message index: %d", messageIndex)
	}

	if field == "" {
		return fmt.Errorf("field cannot be empty")
	}

	jsonValue, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("failed to marshal new value: %w", err)
	}

	// Construct the JSON path
	jsonPath := fmt.Sprintf("{messages,%d,%s}", messageIndex, field)
	s.log.Infof("Update in json by json_path=[%s] and value=[%s]", jsonPath, string(jsonValue))

	return s.db.WithContext(createCtx(ctx, s.log)).
		Model(&Chat{}).
		Where("id = ?", chatID).
		Updates(map[string]interface{}{
			"chat": gorm.Expr("jsonb_set(chat, ?, ?::jsonb)", jsonPath, string(jsonValue)),
		}).
		UpdateColumn("chat", datatypes.JSONSet("chat").Set("{lastUpdateTime}", time.Now().UnixMilli())).
		Error
}

func (s *StorageService) AddMessageToChat(ctx context.Context, chatID uint, newMessage interface{}) error {
	// Convert the new message to JSON
	messageJSON, err := json.Marshal(newMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal new message: %w", err)
	}
	messageJSONStr := string(messageJSON)
	tools.AddFieldToJson(&messageJSONStr, "time", time.Now().Format("2006-01-02T15:04:05.000"))

	// Create the update expression
	update := gorm.Expr("jsonb_set(chat, '{messages}', coalesce(chat->'messages', '[]'::jsonb) || ?::jsonb)", messageJSONStr)

	result := s.db.WithContext(createCtx(ctx, s.log)).
		Model(&Chat{}).
		Where("id = ?", chatID).
		UpdateColumn("chat", update)

	if result.Error != nil {
		s.log.Errorf("failed to update chat: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		s.log.Warn("chat not found or no changes made")
	}

	return result.Error
}

// GetChatsByChainAndUser retrieves chats by chain name and user name
func (s *StorageService) GetChatsByChainAndUser(ctx context.Context, chainName string, userName string) ([]Chat, error) {
	var chats []Chat
	err := s.db.WithContext(createCtx(ctx, s.log)).
		Joins("JOIN users ON users.id = chats.user_id").
		Joins("JOIN chains ON chains.id = users.chain_id").
		Where("chains.name = ?", chainName).
		Where("users.username = ?", userName).
		Order("id desc").
		Find(&chats).Error
	return chats, err
}

func (s *StorageService) GetChatsByChainAndUserId(ctx context.Context, chainName string, userId int) (chats []Chat, err error) {
	err = s.db.WithContext(createCtx(ctx, s.log)).
		Where("user_id = ?", userId).
		Where("chain_name = ?", chainName).
		Order("id desc").
		Find(&chats).Error
	return
}
