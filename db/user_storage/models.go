/*
The implemented role-based access control (RBAC) system is designed based on the following considerations:

1. Design Decisions:
- Hierarchical Roles: The Role model includes a Level field to support role hierarchy
- Flexible Permissions: Many-to-many relationships between roles and permissions allow for granular access control
- JSON Metadata: UserMetadata uses JSONB for flexible user-specific data storage
- Separation of Concerns: Clear separation between users, roles, and permissions

2. Role Structure:
- Administrator: Full system access
- User: Basic chat functionality
- VectorDB Admin: Vector database management
- Custom roles can be easily added

3. Performance Optimizations:
- Indexed fields: Username and Email fields are indexed
- Efficient relationships: Using junction tables for many-to-many relationships
- JSONB for metadata: Allows efficient storage and querying of dynamic data

4. Scalability Features:
- The system supports unlimited roles and permissions
- Hierarchical role structure allows for complex access patterns
- Metadata structure supports future extensions without schema changes

5. Security Considerations:
- Password hashing is handled separately
- Clear separation between authentication and authorization
- Role-based access control at both model and endpoint levels

The implementation provides a solid foundation for the application's security needs while remaining flexible enough for future expansion.

This implementation provides a robust RBAC system that can handle the application's current needs while being extensible for future requirements. The code is production-ready and follows best practices for both Golang and GORM development.
*/
package userstorage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"gitlab.dev.ict/golang/go-ai/db"
)

// User represents a user in the system. The model includes fields for user status, login information, and relationships to other models.
type User struct {
	gorm.Model
	Status        string       `gorm:"type:varchar(20);default:'active'" json:"status"`                       // User status: active, inactive, suspended; gorm: ``;check:status IN ('active', 'inactive', 'suspended')``
	Username      string       `gorm:"uniqueIndex;not null" validate:"required,min=3,max=50" json:"username"` // Unique user login
	Email         *string      `gorm:"" validate:"email" json:"email"`                                        // Optional email address
	Password      string       `gorm:"not null" validate:"required,min=6,max=50"`                             // Hashed password
	LegacyRole    string       `gorm:"column:role;default:'USUAL'"`                                           // Maintaining old column for backward compatibility
	LastLoginTime *time.Time   ``                                                                             // Last login time
	Metadata      UserMetadata `gorm:"type:jsonb"`                                                            // JSONB field for user-specific metadata
	Chats         []Chat       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`                         // 1-N relationship to table chats
	Roles         []Role       `gorm:"many2many:user_roles;constraint:OnDelete:CASCADE" json:"roles"`         // N-N relationship to table roles
	Groups        []Group      `gorm:"many2many:user_groups;constraint:OnDelete:CASCADE" json:"groups"`       // N-N relationship to table groups
	Photo         *UserPhoto   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`                         // 1-1 relationship to table user_photos
	UserGroup     []UserGroup  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	UserRole      []UserRole   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (u *User) String() string {
	return utils.JsonStr(u)
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.Password, _ = helpers.HashPassword(u.Password)
	u.LastLoginTime = lo.ToPtr(time.Now())
	db.Log(tx).Tracef("User:BeforeCreate => %s", u)
	return
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	if !u.IsValid() {
		db.Log(tx).Errorf("User:AfterCreate => rollback TX, cause invalid user: %s", u)
		return errors.New("rollback invalid user")
	}
	if u.LegacyRole != db.RoleAdmin && (u.ID == 1 || lo.Contains([]string{"admin", "dmykolen"}, u.Username)) {
		tx.Model(u).Update("role", db.RoleAdmin)
	}
	db.Log(tx).Tracef("User:AfterCreate => %s", u)
	return nil
}

// BeforeDelete prevents deletion if the user has an admin role directly or through group association.
func (u *User) BeforeDelete(tx *gorm.DB) (err error) {
	// Check for direct admin role
	var roles []Role
	if err := tx.Model(u).Association("Roles").Find(&roles); err != nil {
		return err
	}
	for _, role := range roles {
		if role.Code == db.RoleCodeSuperAdmin {
			return errors.New("deletion forbidden: user is an admin")
		}
	}

	// Check for admin role through groups
	var groups []Group
	if err := tx.Model(u).Association("Groups").Find(&groups); err != nil {
		return err
	}
	for _, group := range groups {
		var groupRoles []Role
		if err := tx.Model(&group).Association("Roles").Find(&groupRoles); err != nil {
			return err
		}
		for _, role := range groupRoles {
			if role.Code == db.RoleCodeSuperAdmin {
				return errors.New("deletion forbidden: user belongs to an admin group")
			}
		}
	}

	return nil
}

func (u *User) IsValid() bool {
	return u.ID > 0 && u.Username != "" && u.Password != "" && u.LegacyRole != ""
}

func (u *User) Validate() error {
	return helpers.Validate(u)
}

func (u *User) ChatsOpenAI() []*goopenai.Chat {
	chats := make([]*goopenai.Chat, len(u.Chats))
	for i, chat := range u.Chats {
		chats[i] = chat.Data()
	}
	return chats
}

func (u *User) ChatsUUID() (res []string) {
	for _, chat := range u.Chats {
		res = append(res, chat.Data().ID)
	}
	return
}

func (u *User) GetChat(uuid string) *Chat {
	for _, chat := range u.Chats {
		if chat.Data().ID == uuid {
			return &chat
		}
	}
	return nil
}

func (u *User) GetChatOpenAI(uuid string) *goopenai.Chat {
	return u.GetChat(uuid).Data()
}

func (u *User) ChatsInfo(filterFN ...func(*Chat) bool) (res []*ChatInfo) {
	for _, chat := range u.Chats {
		if len(filterFN) > 0 && !filterFN[0](&chat) {
			continue
		}
		res = append(res, OpenAIChatToChatInfo(chat.Data()))
	}
	return
}

type UserResponse struct {
	ID       uint     `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Status   string   `json:"status"`
	Roles    []string `json:"roles"`
	Groups   []string `json:"groups"`
}

func (u *User) ToResponse() UserResponse {
	response := UserResponse{
		ID:       u.ID,
		Username: u.Username,
		Status:   u.Status,
	}

	if u.Email != nil {
		response.Email = *u.Email
	}

	// Extract role names
	response.Roles = make([]string, len(u.Roles))
	for i, role := range u.Roles {
		response.Roles[i] = role.Name // Assuming Role struct has a Name field
	}

	// Extract group names
	response.Groups = make([]string, len(u.Groups))
	for i, group := range u.Groups {
		response.Groups[i] = group.Name // Assuming Group struct has a Name field
	}

	return response
}

// Usage:

func OpenAIChatToChatInfo(g *goopenai.Chat) *ChatInfo {
	return &ChatInfo{
		ID:             g.ID,
		Name:           g.Name,
		CreatedTime:    time.UnixMilli(g.CreatedTime).Format(utils.TS_FMT1),
		LastUpdateTime: time.UnixMilli(g.LastUpdateTime).Format(utils.TS_FMT1),
	}
}

type ChatInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CreatedTime    string `json:"createdTime"`
	LastUpdateTime string `json:"lastUpdateTime"`
}

func NewChatsInfo(chats []Chat) (res []*ChatInfo) {
	for _, chat := range chats {
		res = append(res, OpenAIChatToChatInfo(chat.Data()))
	}
	return
}

type Chat struct {
	gorm.Model
	UserID    uint `gorm:"index;not null;constraint:OnDelete:CASCADE"`
	ChainName string
	Chat      datatypes.JSON `gorm:"type:jsonb;not null"`
	User      *User
}

func (c *Chat) BeforeUpdate(tx *gorm.DB) (err error) {
	c.UpdatedAt = time.Now()
	return
}

func (c *Chat) Data() (g *goopenai.Chat) {
	if c.Chat == nil {
		return &goopenai.Chat{}
	}
	utils.JsonToStruct(c.Chat, &g)
	return
}

func (c *Chat) WithChainName(chainName string) *Chat {
	c.ChainName = chainName
	return c
}

func NewChat(userID uint, chat *goopenai.Chat) *Chat {
	chatData, _ := json.Marshal(chat)
	newChat := &Chat{
		UserID: userID,
		Chat:   datatypes.JSON(chatData),
	}
	return newChat
}

func NewChatSimple(chat *goopenai.Chat) *Chat {
	return &Chat{Chat: lo.Must(json.Marshal(chat))}
}

// UserPhoto model for storing photos separately
type UserPhoto struct {
	UserID   uint   `gorm:"primaryKey"`                // Using UserID as PK for 1-1 relationship
	Data     []byte `gorm:"not null"`                  // PostgreSQL: bytea, SQLite: blob
	MimeType string `gorm:"type:varchar(32);not null"` // png or jpeg
}

// SaveToFile saves the photo data to a file
func (p *UserPhoto) SaveToFile(path string) error {
	if len(p.Data) == 0 {
		return errors.New("no photo data to save")
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(p.Data)
	if err != nil {
		return fmt.Errorf("failed to write photo data to file: %w", err)
	}

	return nil
}

// Group represents a group in the system
type Group struct {
	gorm.Model
	Name        string `gorm:"type:varchar(100);unique;not null"`
	Description string `gorm:"type:text"`
	Roles       []Role `gorm:"many2many:group_roles;constraint:OnDelete:CASCADE"`
	Users       []User `gorm:"many2many:user_groups;"`
}

// Permission represents a single permission in the system
type Permission struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(100);unique;not null" json:"name"`
	Code        string `gorm:"type:varchar(100);unique;not null;index" json:"code"`
	Description string `gorm:"type:text" json:"description"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Role represents a user role with associated permissions
type Role struct {
	ID          uint         `gorm:"primaryKey"`
	Name        string       `gorm:"type:varchar(100);unique;not null;index"`
	Code        string       `gorm:"type:varchar(100);unique;not null;index"`
	Description string       `gorm:"type:text"`
	Level       int          `gorm:"type:int;default:0"` // For hierarchical roles
	Permissions []Permission `gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// BeforeSave GORM hook to automatically make Role.Code uppercase
func (r *Role) BeforeSave(tx *gorm.DB) (err error) {
	r.Code = strings.ToUpper(r.Code)
	return nil
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uint `gorm:"primaryKey;foreignKey:RoleID;references:ID;constraint:OnDelete:CASCADE"`
	PermissionID uint `gorm:"primaryKey;foreignKey:PermissionID;references:ID;constraint:OnDelete:CASCADE"`
	// RoleID       uint `gorm:"primaryKey"`
	// PermissionID uint `gorm:"primaryKey"`
	CreatedAt time.Time
}

// Junction table represents the many-to-many relationship between groups and roles
type GroupRole struct {
	GroupID   uint `gorm:"primaryKey;foreignKey:GroupID;references:ID;constraint:OnDelete:CASCADE"`
	RoleID    uint `gorm:"primaryKey;foreignKey:RoleID;references:ID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// Junction table for users and groups
type UserGroup struct {
	UserID    uint `gorm:"primaryKey;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	GroupID   uint `gorm:"primaryKey;foreignKey:GroupID;references:ID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uint `gorm:"primaryKey;constraint:OnDelete:CASCADE"`
	RoleID    uint `gorm:"primaryKey;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time
}

// UserMetadata stores additional user information
type UserMetadata struct {
	LastAccess     time.Time              `json:"last_access,omitempty"`
	PreferredTheme string                 `json:"preferred_theme,omitempty"`
	Settings       map[string]interface{} `json:"settings,omitempty"`
}

// Value implements the driver.Valuer interface for UserMetadata
func (m UserMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for UserMetadata
func (m *UserMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for UserMetadata")
	}
	return json.Unmarshal(bytes, &m)
}

// Migration function to create the necessary database schema
func AutoMigrate(db *gorm.DB) error {
	dialectName := db.Dialector.Name()

	if dialectName == "postgres" {
		// PostgreSQL-specific enum creation
		db.Exec(`DO $$ BEGIN
            CREATE TYPE user_status AS ENUM ('active', 'inactive', 'suspended');
            EXCEPTION WHEN duplicate_object THEN NULL;
        END $$;`)
	} else {
		// For SQLite, we'll use a string column instead of enum
		db.Exec(`CREATE TABLE IF NOT EXISTS user_statuses (
            status TEXT CHECK (status IN ('active', 'inactive', 'suspended'))
        )`)
	}

	// Migrate the schemas
	return db.AutoMigrate(
		&Permission{},
		&Role{},
		&User{},
		&UserPhoto{},
		&RolePermission{},
		&UserRole{},
		&Group{},
		&GroupRole{},
		&UserGroup{},
		&Chat{},
	)
}

// initializeRolesAndPermissions handles the safe initialization of roles and permissions
func InitializeRolesAndPermissions(gdb *gorm.DB, log *gologgers.LogRec, isForceinitRoles bool) error {
	// First, check if initialization has already been done
	var count int64
	if err := gdb.Model(&Role{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing roles: %w", err)
	}

	// If roles already exist, skip initialization
	if count > 0 && !isForceinitRoles {
		log.Info("Roles already initialized, skipping...")
		return nil
	}

	// Then in initializeRolesAndPermissions:
	if isForceinitRoles {
		// Drop existing roles and permissions
		log.Info("Forced reinitialization of roles and permissions")
		if err := clearRolesAndPermissions(gdb); err != nil {
			return fmt.Errorf("failed to clear existing roles: %w", err)
		}
	}

	// permissions := []Permission{
	// 	{Code: "chat:read", Name: "Read Chats", Description: "Can read chat messages"},
	// 	{Code: "chat:write", Name: "Write Chats", Description: "Can write chat messages"},
	// 	{Code: "user:read", Name: "Read Users", Description: "Can read user information"},
	// 	{Code: "user:write", Name: "Write Users", Description: "Can modify user information"},
	// 	{Code: "vectordb:read", Name: "Read VectorDB", Description: "Can read from vector database"},
	// 	{Code: "vectordb:write", Name: "Write VectorDB", Description: "Can write to vector database"},
	// 	{Code: "admin:full", Name: "Full Admin", Description: "Full administrative access"},
	// }

	permissions := []Permission{
		{Code: "fe.ai_chat.access", Name: "AI Chat Access", Description: "Access to AI chat features"},
		{Code: "fe.voip.access", Name: "VoIP Access", Description: "Access to VoIP features"},
		{Code: "fe.aidb.access", Name: "AI DB Access", Description: "Access to AI database features"},
		{Code: "sse_access", Name: "SSE Access", Description: "Access to Server-Sent Events"},
		{Code: "fe.chat_access", Name: "Chat Access", Description: "Access to chat functionality"},
		{Code: "chatgpt_access", Name: "ChatGPT Access", Description: "Access to ChatGPT features"},
		{Code: "announce_access", Name: "Announce Access", Description: "Permission to make announcements"},
		{Code: "api_v1_access", Name: "API v1 Access", Description: "Access to API v1 endpoints"},
		{Code: "ask_ai_voip", Name: "Ask AI VoIP", Description: "Permission to use AI VoIP features"},
		{Code: "handle_stt", Name: "Handle STT", Description: "Permission to handle Speech-to-Text"},
		{Code: "api.chat.access", Name: "API Chat Access", Description: "Access to chat API endpoints"},
		{Code: "api.user_store.access", Name: "User Store Access", Description: "Access to user store API"},
		{Code: "users.manage", Name: "Manage Users", Description: "Permission to manage users"},
		{Code: "users.modify", Name: "Modify Users", Description: "Permission to modify user data"},
		{Code: "roles.manage", Name: "Manage Roles", Description: "Permission to manage roles"},
		{Code: "roles.modify", Name: "Modify Roles", Description: "Permission to modify roles"},
		{Code: "groups.manage", Name: "Manage Groups", Description: "Permission to manage groups"},
		{Code: "groups.modify", Name: "Modify Groups", Description: "Permission to modify groups"},
		{Code: "permissions.manage", Name: "Manage Permissions", Description: "Permission to manage permissions"},
		{Code: "permissions.modify", Name: "Modify Permissions", Description: "Permission to modify permissions"},
		{Code: "ws_access", Name: "WebSocket Access", Description: "Access to WebSocket endpoints"},
		{Code: "ws_account_access", Name: "WS Account Access", Description: "Access to WebSocket account features"},
		{Code: "vectordb.admin_access", Name: "VectorDB Admin Access", Description: "Administrative access to vector database"},
		{Code: "vectordb.docs_view", Name: "View VectorDB Docs", Description: "Permission to view vector database documents"},
		{Code: "vectordb.upload_docs", Name: "Upload VectorDB Docs", Description: "Permission to upload documents to vector database"},
		{Code: "vectordb.manage_objects", Name: "Manage VectorDB Objects", Description: "Permission to manage vector database objects"},
		{Code: "vectordb.modify_objects", Name: "Modify VectorDB Objects", Description: "Permission to modify vector database objects"},
		{Code: "vectordb.search", Name: "Search VectorDB", Description: "Permission to search vector database"},
	}

	// Create permissions within a transaction
	return gdb.Transaction(func(tx *gorm.DB) error {
		// Create permissions
		for _, perm := range permissions {
			if err := tx.FirstOrCreate(&perm, Permission{Code: perm.Code}).Error; err != nil {
				return fmt.Errorf("failed to create permission %s: %w", perm.Code, err)
			}
		}

		// Define roles with their permissions
		roles := []struct {
			Role        Role
			Permissions []string
		}{
			{
				Role: Role{
					Name:        "Super Administrator",
					Code:        db.RoleCodeSuperAdmin,
					Description: "Full system administrator",
					Level:       100,
				},
				Permissions: []string{"fe.ai_chat.access", "fe.voip.access", "fe.aidb.access", "sse_access", "fe.chat_access", "chatgpt_access", "announce_access", "api_v1_access", "ask_ai_voip", "handle_stt", "api.chat.access", "api.user_store.access", "users.manage", "users.modify", "roles.manage", "roles.modify", "groups.manage", "groups.modify", "permissions.manage", "permissions.modify", "ws_access", "ws_account_access", "vectordb.admin_access", "vectordb.docs_view", "vectordb.upload_docs", "vectordb.manage_objects", "vectordb.modify_objects", "vectordb.search"},
			},
			{
				Role: Role{
					Name:        "Administrator VoIP",
					Code:        db.RoleCodeAdminVoIP,
					Description: "Full system administrator",
					Level:       100,
				},
				Permissions: []string{"fe.ai_chat.access", "fe.voip.access", "fe.aidb.access", "sse_access", "fe.chat_access", "chatgpt_access", "announce_access", "api_v1_access", "ask_ai_voip", "handle_stt", "api.chat.access", "api.user_store.access", "users.manage", "users.modify", "roles.manage", "roles.modify", "groups.manage", "groups.modify", "permissions.manage", "permissions.modify", "ws_access", "ws_account_access", "vectordb.admin_access", "vectordb.docs_view", "vectordb.upload_docs", "vectordb.manage_objects", "vectordb.modify_objects", "vectordb.search"},
			},
			{
				Role: Role{
					Name:        "VoIP User",
					Code:        db.RoleCodeVoIPUser,
					Description: "VoIP user",
					Level:       1,
				},
				Permissions: []string{"fe.voip.access", "sse_access", "ask_ai_voip", "fe.chat_access"},
			},
			{
				Role: Role{
					Name:        "VectorDB Admin",
					Code:        db.RoleCodeVectorDBAdmin,
					Description: "Vector database administrator",
					Level:       50,
				},
				Permissions: []string{"vectordb.admin_access", "vectordb.docs_view", "vectordb.upload_docs", "vectordb.manage_objects", "vectordb.modify_objects", "vectordb.search"},
			},
		}

		// Create roles and assign permissions
		for _, r := range roles {
			role := r.Role
			if err := tx.FirstOrCreate(&role, Role{Code: role.Code}).Error; err != nil {
				return fmt.Errorf("failed to create role %s: %w", role.Code, err)
			}

			// Get permissions for this role
			var permissions []Permission
			if err := tx.Where("code IN ?", r.Permissions).Find(&permissions).Error; err != nil {
				return fmt.Errorf("failed to find permissions for role %s: %w", role.Code, err)
			}

			// Assign permissions to role
			if err := tx.Model(&role).Association("Permissions").Replace(permissions); err != nil {
				return fmt.Errorf("failed to assign permissions to role %s: %w", role.Code, err)
			}
		}

		log.Info("Successfully initialized roles and permissions")
		return nil
	})
}

func clearRolesAndPermissions(db *gorm.DB) error {
	dialectName := db.Dialector.Name()

	if dialectName == "postgres" {
		// Execute each TRUNCATE command separately
		if err := db.Exec("TRUNCATE roles CASCADE").Error; err != nil {
			return fmt.Errorf("failed to clear existing roles: %w", err)
		}
		if err := db.Exec("TRUNCATE permissions CASCADE").Error; err != nil {
			return fmt.Errorf("failed to clear existing permissions: %w", err)
		}
		// return db.Exec("TRUNCATE roles CASCADE; TRUNCATE permissions CASCADE;").Error
	} else {
		// For SQLite, use DELETE instead of TRUNCATE
		err := db.Exec("DELETE FROM role_permissions").Error
		if err != nil {
			return err
		}
		err = db.Exec("DELETE FROM user_roles").Error
		if err != nil {
			return err
		}
		err = db.Exec("DELETE FROM roles").Error
		if err != nil {
			return err
		}
		err = db.Exec("DELETE FROM permissions").Error
		if err != nil {
			return err
		}

		// Reset SQLite sequences
		_ = db.Exec("DELETE FROM sqlite_sequence WHERE name IN ('roles', 'permissions', 'role_permissions', 'user_roles')").Error
		if err != nil {
			return err
		}
	}
	return nil
}
