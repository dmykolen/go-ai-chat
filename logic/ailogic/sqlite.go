package ailogic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var dbSqlite *gorm.DB

// Define a User struct to map to the users table
type User struct {
	ID         uint      `gorm:"primaryKey" json:"-"`
	Login      string    `gorm:"size:255;not null" json:"-"`
	ChainName  string    `gorm:"size:255;not null" json:"name"`
	ChatUUID   string    `gorm:"size:255;uniqueIndex" json:"id"`
	Date       time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"createdTime"`
	DateUpdate time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"lastUpdateTime"`
}

func (u *User) String() string {
	return utils.JsonStr(u)
}

func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	if user.ChatUUID == "" {
		user.ChatUUID = uuid.New().String()
	}
	return
}

func GetSqliteDB(dbName ...string) *gorm.DB {
	if dbSqlite == nil {
		dbSqlite, err = gorm.Open(sqlite.Dialector{DriverName: "sqlite3", Conn: DBMemory()}, &gorm.Config{})
		// dbSqlite, err = gorm.Open(sqlite.Open(utils.FirstOrDefault("test.db", dbName...)), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}

		dbSqlite.AutoMigrate(&User{})
		dbSqlite.Exec("CREATE INDEX idx_login_chain_name ON users(login, chain_name);")
	}
	return dbSqlite
}

func ChatForChainCtx(ctx context.Context) context.Context {
	login, chainName, chatUUID := GetLCCFromCtx(ctx)
	chatUUID, err = ChatForChain(ctx, chainName, login, chatUUID)
	ctx = AddToCtxUUIDAI(ctx, chatUUID)
	return ctx
}
func ChatForChain(ctx context.Context, chainName, login string, chatid ...string) (chatUUID string, err error) {
	if chainName == "" {
		return "", fmt.Errorf("chainName or login cannot be empty")
	}

	u := User{Login: login, ChainName: chainName, ChatUUID: utils.FirstOrDefault("", chatid...)}
	tx := GetSqliteDB().WithContext(ctx).Create(&u)
	chatUUID, err = u.ChatUUID, tx.Error
	return
}

func GetChainChatFromDB(ctx context.Context, chainName, login string) (chats []string, err error) {
	if tx := GetSqliteDB().WithContext(ctx).Model(&User{}).Where("chain_name = ? AND login = ?", chainName, login).Pluck("chat_uuid", &chats); tx.Error != nil {
		return nil, tx.Error
	}
	return
}

// Is exists chat_uuid by login and chain_name
func IsExistsChatUUID(ctx context.Context, chainName, login, chatUUID string) (exists bool, err error) {
	var count int64
	tx := GetSqliteDB().WithContext(ctx).Model(&User{}).Where("chain_name = ? AND login = ? AND chat_uuid = ?", chainName, login, chatUUID).Count(&count)
	if tx.Error != nil {
		return false, tx.Error
	}
	return count > 0, nil
}

// select all users
func SelectAllUsers(ctx context.Context) (users []User, err error) {
	if tx := GetSqliteDB().WithContext(ctx).Find(&users); tx.Error != nil {
		return nil, tx.Error
	}
	return
}

// select all users by login
func GetAllByLoginAndChain(ctx context.Context, login, chainName string) (users []User, err error) {
	if tx := GetSqliteDB().WithContext(ctx).Where("login = ? AND chain_name = ?", login, chainName).Find(&users); tx.Error != nil {
		return nil, tx.Error
	}
	return
}
