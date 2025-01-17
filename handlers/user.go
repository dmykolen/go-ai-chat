package handlers

import (
	"bufio"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/slog"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models/sse"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
)

const sessionDuration = 6 * time.Hour

var (
	appStoreForUsers = sync.Map{} // sync map for storing users
)

func GetAppStoreForUsers() *sync.Map {
	return &appStoreForUsers
}

// User - user struct, describes user, who connected to the server. Contains user's uuid, login, connection time, channel for sending messages to the user
type User struct {
	UUID           string
	Login          string
	Password       string
	ConnTime       time.Time
	ChanMessages   chan string              `json:"-"`
	ChanEventMsg   chan sse.Event           `json:"-"`
	ChanMsgBB      chan []byte              `json:"-"`
	ChanWithSSEMsg chan string              `json:"-"`
	ActiveConns    map[string]*bufio.Writer `json:"-"`
	chats          []*goopenai.Chat
	DBID           uint
	storeDB        us.UserStorage `json:"-"`
	mu             sync.Mutex
}

func (u *User) IsEmpty() bool {
	return u == nil || u.UUID == ""
}

func (u *User) WithStoreDB(storage us.UserStorage) *User {
	u.storeDB = storage
	return u
}

func (u *User) String() string {
	return fmt.Sprintf("UUID=[%s] login=%s passwd=[%s] conTime=[%s] id_in_db=%d", u.UUID, u.Login, strings.Repeat("*", len(u.Password)), u.ConnTime.Format(time.RFC3339), u.DBID)
}

func (u *User) ChatsOpenAI() []*goopenai.Chat {
	return u.chats
}

func (u *User) AddChatOpenAI(log *slog.Record, chat *goopenai.Chat) {
	if chat == nil {
		return
	}
	if u.chats == nil {
		u.chats = make([]*goopenai.Chat, 8)
	}
	u.chats = append(u.chats, chat)
	u.storeDB.CreateChat(log.Ctx, us.NewChatSimple(chat))
}

func (u *User) GetChatOpenAI(uuid string) *goopenai.Chat {
	for _, chat := range u.chats {
		if chat.ID == uuid {
			return chat
		}
	}
	return nil
}

func (u *User) ChatsUUID() (res []string) {
	for _, chat := range u.chats {
		res = append(res, chat.ID)
	}
	return
}

// GetConnection retrieves a connection from ActiveConns by ID.
// If the ID is empty, it returns the first connection from the map.
// If the ID is not found in the map, it returns nil.
func (u *User) GetConnection(id string) *bufio.Writer {
	u.mu.Lock()
	defer u.mu.Unlock()

	if id == "" {
		for _, conn := range u.ActiveConns {
			return conn
		}
		return nil
	}

	if conn, exists := u.ActiveConns[id]; exists {
		return conn
	}

	return nil
}

// StoreConnection - store an active SSE connection for a user, replacing an old one if it exists.
func (u *User) StoreConnection(log *gologgers.LogRec, connectionId string, w *bufio.Writer) {
	u.mu.Lock()
	defer u.mu.Unlock()

	// If a connection already exists for this connectionId, close it.
	if existingWriter, exists := u.ActiveConns[connectionId]; exists {
		log.Infof("Closing existing connectionId=%s for userUUID=%s", connectionId, u.UUID)
		existingWriter.Flush() // Optionally attempt a flush before closing.
		delete(u.ActiveConns, connectionId)
	}

	// Store the new connection.
	u.ActiveConns[connectionId] = w
	log.Infof("Stored connectionId=%s for userUUID=%s", connectionId, u.UUID)
}

// CleanupConnection - cleanup a disconnected SSE connection for a user, and remove it from ActiveConns.
func (u *User) CleanupConnection(log *gologgers.LogRec, connectionId string) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if _, exists := u.ActiveConns[connectionId]; exists {
		log.Infof("Cleaning up connectionId=%s for userUUID=%s", connectionId, u.UUID)
		delete(u.ActiveConns, connectionId)
	}

	// If no more connections are active, remove the user from the store.
	if len(u.ActiveConns) == 0 {
		log.Infof("No active connections left for userUUID=%s", u.UUID)
	}
}

// NewUser - constructor for User
func NewUser(login string, pwd ...string) *User {
	return &User{
		UUID:           utils.UUID(),
		Login:          login,
		Password:       utils.FirstOrDefault[string]("", pwd...),
		ConnTime:       time.Now(),
		ChanMessages:   make(chan string),
		ChanEventMsg:   make(chan sse.Event),
		ChanMsgBB:      make(chan []byte),
		ChanWithSSEMsg: make(chan string),
		ActiveConns:    make(map[string]*bufio.Writer),
	}
}

func addUser(c *fiber.Ctx, user ...*User) (u *User, err error) {
	log := help.Log(c)
	log.Infof("Add user! IN=>[cookie_userid=%s; user=[%s]; existing_users=%s", c.Cookies(CookUID), utils.FirstOrDefault(nil, user), usersUUIDsAsStr())

	if u = getUser(c); u != nil {
		log.Infof("User found in app cache store: %s", u)
		return
	}

	if len(user) > 0 {
		log.Infof("User found in input params: %s", user[0])
		u = user[0]
	} else {
		log.Infof("User not found in params and in app cache store")
		c.ClearCookie(CookUID, CookUName)
		c.Context().Error("You are UNAUTHORIZE user!", 401)
		err = fmt.Errorf("you are UNAUTHORIZE user")
		return
	}

	userDB := &us.User{Username: u.Login, Password: u.Password}
	if err = u.storeDB.CreateUser(log.Ctx, userDB); err != nil {
		log.Errorf("Error creating user in DataBase: %v", err)
		return
	}
	u.DBID = userDB.ID

	log.Infof("Update cookies and add user to cache. USER[%d]: %s", userDB.ID, user)
	cookiesUserUpdate(c, u, sessionDuration)
	appStoreForUsers.Store(u.UUID, u)
	return
}

// getUser - return a pointer to the User struct if found.
// If the cookie does not exist or the user is not found in the cache, it returns nil.
func getUser(c *fiber.Ctx) (user *User) {
	if c.Cookies(CookUID) == "" {
		help.Log(c).Warnf("User not found, cause Cookie [%s] not exists!", CookUID)
		return
	}

	if u, ok := appStoreForUsers.Load(c.Cookies(CookUID)); ok {
		user = u.(*User)
		help.Log(c).Debug("User found in app cache store: ", user)
	} else {
		help.Log(c).Warn("User not found in app cache store!")
	}
	return
}

func GetUser(c *fiber.Ctx) (user *User) {
	user = getUser(c)
	if user == nil {
		return &User{}
	}
	return
}

func usersCount() int {
	var count int
	appStoreForUsers.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

func usersUUIDsAsStr() string { return arrutil.AnyToString(usersUUIDs()) }

func usersUUIDs() (list []string) {
	appStoreForUsers.Range(func(key, v interface{}) bool {
		list = append(list, v.(*User).UUID)
		return true
	})
	return
}
