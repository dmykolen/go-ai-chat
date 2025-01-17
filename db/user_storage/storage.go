package userstorage

import (
	"context"
)

// UserStorage is an interface that defines the methods for user and chat operations
type UserStorage interface {
	RoleService() *UserRoleService
	GroupService() *GroupService
	PermService() *PermissionService

	UpdateUserGroups(ctx context.Context, userID uint, groups []string) error

	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, id uint) (*User, error)
	GetUserFull(ctx context.Context, id uint) (*User, error)
	GetUsersAM(ctx context.Context) ([]UserResponse, error)
	GetUserWithChatsByUserName(ctx context.Context, userName string) (*User, error)
	GetUserPhoto(ctx context.Context, userID uint) (*UserPhoto, error)
	ProcessAndUpdateUserPhoto(ctx context.Context, userID uint, photoData []byte) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uint) error
	GetUsers(ctx context.Context) ([]User, error)
	UpdateUserEmail(ctx context.Context, userID uint, newEmail string) error
	FindUserByUsername(ctx context.Context, username string, withChats ...bool) (*User, error)

	CreateChat(ctx context.Context, chat *Chat) error
	CreateChatForUsername(ctx context.Context, username string, chat *Chat) error
	GetAllChats(ctx context.Context) ([]Chat, error)
	GetChats(ctx context.Context, userID uint) ([]Chat, error)
	UpdateChat(ctx context.Context, chat *Chat) error
	DeleteChat(ctx context.Context, chatID uint) error
	DeleteAllChatsForUser(ctx context.Context, userID uint) error
	GetChatByUUID(ctx context.Context, uuid string) (*Chat, error)
	UpdateChatFieldInChat(ctx context.Context, chatID uint, field string, newValue interface{}) error
	UpdateMessageFieldInChat(ctx context.Context, chatID uint, messageIndex int, field string, newValue interface{}) error
	AddMessageToChat(ctx context.Context, chatID uint, newMessage interface{}) error
	GetChatsByChainAndUserId(ctx context.Context, chainName string, userId int) (chats []Chat, err error)
}
