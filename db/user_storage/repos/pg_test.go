package repos

import (
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/go-faker/faker/v4"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	ddb "gitlab.dev.ict/golang/go-ai/db"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
)

var (
	logDebug = gologgers.New(gologgers.WithLevel(gologgers.LevelDebug), gologgers.WithOC())
	log      = gologgers.New(gologgers.WithLevel(gologgers.LevelInfo), gologgers.WithOC())
	ctx      = utils.GenerateCtxWithRid()
	dbProps  = &ddb.DBProperties{}
	storage  *us.StorageService
)

func init() {
	envLog := func() {
		list := os.Environ()
		sort.Strings(list)
		lo.ForEach(list, func(item string, index int) {
			if strings.HasPrefix(item, "POSTGRES") {
				log.Info(item)
			}
		})
	}
	// os.Setenv("POSTGRES_PORT", "")
	envLog()

	godotenv.Overload("../../../.env.dev")
	// godotenv.Overload("../../../.env.local.dev")
	// godotenv.Overload("../../../.env.test.local")
	env.Parse(dbProps)
	log.Infof("DB_PROPS: %#v", dbProps)
	envLog()

	log.Warn("IsExists >>>", utils.IsExists("../../../.env.local.dev"))

}

func Test_DROP(t *testing.T) {
	t.Log("Dropping tables")
	// dropTables(t)
	dropTablesGP(t)
}

// Test_ALL is a test function that runs multiple tests related to user storage.
//
//		-> BEFORE calling this function, make sure to run the DROP test function.(call Test_DROP)
//
//	 1. Test if validation of db properties are passed
//	 2. Test if the storage is initialized correctly
//	 3. Test if the user is created (3 times)
//	 4. Test if the chat is created (5 times)
//	 5. Test if the user is fetched with the chats
//		5.1. Test if the user has chats
//		5.2. Test if the chats is fetched with the user
func Test_ALL(t *testing.T) {
	if assert.NoError(t, helpers.ValidateWithLog(log.Rec(), dbProps)) {
		help_init_storage(t, true)
	}
	Test_Users_Create(t)
	Test_Chats_Create(t)
	test_User(t, 1)
}

// TestCreateUser tests the CreateUser method of UserStoragePG
func Test_Users(t *testing.T) {
	help_init_storage(t, true)
	var users []us.User
	t.Log("Start call DB!!")
	users, err := storage.GetUsers(ctx)
	assert.NoError(t, err)
	if assert.NotNil(t, users) {
		t.Log(utils.JsonPrettyStr(users))
	}
}

func Test_User(t *testing.T) {
	help_init_storage(t, false)
	test_User(t, 1)
}
func test_User(t *testing.T, id uint) (isExists bool) {
	t.Run("GetUser_by_id_"+strconv.Itoa(int(id)), func(t *testing.T) {
		u, err := storage.GetUser(ctx, id)
		assert.NoError(t, err)
		if assert.NotNil(t, u) {
			assert.NotNil(t, u.Chats)
			assert.NotEmpty(t, u.Chats)
			t.Log(utils.JsonPrettyStr(u))
			isExists = true
		}
	})
	return
}
func Test_Users_Create(t *testing.T) {
	help_init_storage(t, false)
	for _, i := range []int{1, 2, 3} {
		t.Run("Create_user_"+strconv.Itoa(i), func(t *testing.T) {
			assert.NoError(t, storage.CreateUser(ctx, help_create_fake_user(t)))
		})
	}
}

func Test_User_Create(t *testing.T) {
	help_init_storage(t, false)
	u := help_create_fake_user(t)
	assert.NoError(t, storage.CreateUser(ctx, u))
	user, err := storage.FindUserByUsername(ctx, u.Username)
	assert_users(t, err, user)
}

func Test_User_Create_With_RoleADM(t *testing.T) {
	help_init_storage(t, false)
	userId := lo.Must(storage.FindUserByUsername(ctx, os.Getenv("USER"))).ID
	var totalChats int64
	assert.NoError(t, storage.Db().Model(&us.Chat{}).Count(&totalChats).Error)
	t.Log("USER_ID =>", userId, "; TOTAL_CHATS:", totalChats)
	if userId != 0 {
		t.Logf("\n\n\n%s\n\n\n", "DELETE USER! !!")
		if !assert.NoError(t, storage.DeleteUser(ctx, userId)) {
			t.FailNow()
		}
	}
	assert.NoError(t, storage.Db().Model(&us.Chat{}).Count(&totalChats).Error)
	t.Log("USER_ID =>", "; TOTAL_CHATS:", totalChats)

	u := &us.User{Username: os.Getenv("USER"), Password: "0003330000999999"}
	if assert.NoError(t, storage.CreateUser(ctx, u)) {
		t.Logf("User was created! User: %s", u)
		storage.CreateChat(ctx, us.NewChat(u.ID, goopenai.NewChat().AddUserMessage("Hello!").AddAssistantMessage("Hi!")))
		storage.CreateChat(ctx, us.NewChat(u.ID, goopenai.NewChat().AddUserMessage("Hello!").AddAssistantMessage("Hi!")))
	}
	assert.NoError(t, storage.Db().Model(&us.Chat{}).Count(&totalChats).Error)
	t.Log("USER_ID =>", u.ID, "; TOTAL_CHATS:", totalChats)
}

func Test_User_Create_ifDuplicate(t *testing.T) {
	help_init_storage(t, false)
	u := help_create_fake_user(t)
	assert.NoError(t, storage.CreateUser(ctx, u))
	user, err := storage.FindUserByUsername(ctx, u.Username)
	assert_users(t, err, user)

	u.Email = lo.ToPtr(faker.Email())
	assert.NoError(t, storage.CreateUser(ctx, u))
	user, err = storage.FindUserByUsername(ctx, u.Username)
	assert_users(t, err, user)
}

func Test_User_Update(t *testing.T) {
	help_init_storage(t, false)
	u := help_create_fake_user(t)
	assert.NoError(t, storage.Db().Create(u).Error)
	t.Log(u)
	u.Email = lo.ToPtr("__333333___@gmail.com")
	storage.Db().Save(u)
	t.Log(u)
	t.Log(storage.GetUser(ctx, u.ID))
	t.Logf("\n\n\nU - %s\n\n", "22222@ffff.ua")
	storage.Db().Model(u).Update("email", "22222@ffff.ua")
	t.Log(storage.GetUser(ctx, u.ID))
	t.Logf("\n\n\nU - %s\n\n", "55555@ffff.ua")
	storage.Db().Where(&us.User{Model: gorm.Model{ID: u.ID}}).Update("email", "55555@ffff.ua")
	t.Log(storage.GetUser(ctx, u.ID))

}

func Test_Chats2(t *testing.T) {
	help_init_storage(t, false)
	t.Log("Start call DB")
	t.Run("get_chat_info", func(t *testing.T) {
		chats, err := storage.GetChatsByChainAndUserId(ctx, "DbCimChain", 1)
		assert_chats(t, err, chats...)

		var ci []*us.ChatInfo
		for _, v := range chats {
			ci = append(ci, us.OpenAIChatToChatInfo(v.Data()))
		}
		t.Log(utils.JsonPrettyStr(ci))

		chats, err = storage.GetChatsByChainAndUserId(ctx, "VoipAgents", 1)
		assert_chats(t, err, chats...)
		t.Log(utils.JsonPrettyStr(us.NewChatsInfo(chats)))
	})

	t.Run("test_select_by_name_and_id", func(t *testing.T) {
		chats, err := storage.GetChatsByChainAndUserId(ctx, "VoipAgents", 1)
		assert_chats(t, err, chats...)
	})

}
func Test_Chats(t *testing.T) {
	help_init_storage(t, false)
	t.Log("Start call DB")
	chats, err := storage.GetChats(ctx, 1)
	assert_chats(t, err, chats...)
}

func Test_Chats_AddMessage(t *testing.T) {
	help_init_storage(t, false)
	// chats, err := storage.GetChats(ctx, 4)
	// assert_chats(t, err, chats...)
	chat, err := storage.GetChatByUUID(ctx, "13963cf4-33f6-4cbd-b432-32729f652c9a")
	assert_chats(t, err, *chat)

	cc := goopenai.NewChatEmpty().AddUserMessage("Today is FRIDAY!!!")
	t.Log("NEEEEW", utils.JsonPrettyStr(cc.GetLastMessage()))

	assert.NoError(t, storage.AddMessageToChat(ctx, chat.ID, cc.GetLastMessage()))
	assert.NoError(t, storage.AddMessageToChat(ctx, chat.ID, cc.AddAssistantMessage("I am glaaaaaaad to see you :)").GetLastMessage()))

	chat, err = storage.GetChatByUUID(ctx, "13963cf4-33f6-4cbd-b432-32729f652c9a")
	assert_chats(t, err, *chat)
}

func Test_ChatsByUserName(t *testing.T) {
	help_init_storage(t, false)
	t.Run("GetUser", func(t *testing.T) {
		u, err := storage.FindUserByUsername(ctx, "dmykolen", true)
		assert.NoError(t, err)
		if assert.NotNil(t, u) {
			t.Log(utils.JsonPrettyStr(u))
		}

	})
	t.Run("DeleteAllChatsForUser", func(t *testing.T) {
		err := storage.DeleteAllChatsForUser(ctx, 4)
		assert.NoError(t, err)
		users, err := storage.FindUserByUsername(ctx, "dmykolen", true)
		assert.NoError(t, err)
		t.Log("UserChatsTOTAL =>", len(users.Chats))
		assert.Empty(t, users.Chats)

	})
	t.Run("AddChat_", func(t *testing.T) {
		aiChat := goopenai.NewChat().AddUserMessage("Hello!").AddAssistantMessage("Hi!")
		t.Log("CHAAAT =>", utils.JsonPrettyStr(aiChat))
		err := storage.CreateChatForUsername(ctx, "dmykolen", us.NewChatSimple(aiChat))
		assert.NoError(t, err)
		users, err := storage.FindUserByUsername(ctx, "dmykolen", true)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
		t.Log("UserChatsTOTAL =>", len(users.Chats))
		assert_chats(t, err, users.Chats...)
	})
	t.Run("GetChats_by_user_name", func(t *testing.T) {
		users, err := storage.GetUserWithChatsByUserName(ctx, "Emelia")
		assert_chats(t, err, users.Chats...)
	})
	t.Run("GetChats_by_user_name_22", func(t *testing.T) {
		user, err := storage.FindUserByUsername(ctx, os.Getenv("USER"), true)
		assert_chats(t, err, user.Chats...)
		assert.NotEmpty(t, user.ChatsOpenAI())
		assert.NotEmpty(t, user.ChatsUUID())
		t.Log(user.ChatsUUID())
		t.Log(utils.JsonPrettyStr(user.ChatsUUID()))
		t.Log(user.ChatsOpenAI()[0].Json())

	})
	t.Run("GetChats_by_user_name", func(t *testing.T) {
		aiChat := goopenai.NewChat().AddUserMessage("Hello!").AddAssistantMessage("Hi!")
		t.Log(utils.JsonPrettyStr(aiChat))
		err := storage.CreateChatForUsername(ctx, "dmykolen", us.NewChatSimple(aiChat))
		assert.NoError(t, err)
		users, err := storage.FindUserByUsername(ctx, "dmykolen", true)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
		t.Log(utils.JsonPrettyStr(users))
	})
}

func Test_GetChats(t *testing.T) {
	help_init_storage(t, false)
	// chats, err := storage.GetChats(ctx, 4)
	// assert_chats(t, err, chats...)
	chat, err := storage.GetChatByUUID(ctx, "5c4428e7-8e50-474e-bdcd-4cde04971526")
	assert_chats(t, err, *chat)
}

func Test_ChatsByUserName2(t *testing.T) {
	help_init_storage(t, false)
	t.Log("Start call DB")
	user, err := storage.GetUserWithChatsByUserName(ctx, "Emelia")
	assert.NoError(t, err)
	if assert.NotNil(t, user) {
		t.Log(utils.JsonPrettyStr(user))
		t.Log(utils.JsonPrettyStr(user.ChatsUUID()))
		t.Log(utils.JsonPrettyStr(user.GetChatOpenAI("6f3d189b-f4ba-4d88-ad8d-9a658065d659")))
	}
}

func Test_ChatsByUserName3(t *testing.T) {
	help_init_storage(t, false)
	t.Log("Start call DB!")
	user, err := storage.GetUserWithChatsByUserName(ctx, "dmykolen")
	assert.NoError(t, err)
	if assert.NotNil(t, user) {
		t.Log(utils.JsonPrettyStr(user))
		t.Log(utils.JsonPrettyStr(user.ChatsUUID()))
		// t.Log(utils.JsonPrettyStr(user.GetChatOpenAI("6f3d189b-f4ba-4d88-ad8d-9a658065d659")))
	}
}

func Test_Chat_Create(t *testing.T) {
	help_init_storage(t, false)
	assert.NoError(t, storage.CreateChat(ctx, help_create_fake_chat(t, 1)))
}
func Test_Chats_Create(t *testing.T) {
	help_init_storage(t, false)
	for _, i := range []int{1, 1, 1, 2, 3} {
		t.Run("Create_chat_for_user-"+strconv.Itoa(i), func(t *testing.T) {
			assert.NoError(t, storage.CreateChat(ctx, help_create_fake_chat(t, uint(i))))
		})
	}
}

func TestInsertChat(t *testing.T) {
	help_init_storage(t, false)
	chat := us.Chat{
		UserID: 1, // Assuming a user with ID=1 exists
		// Chat:   datatypes.JSON([]byte(`{"message": "Hello, World!"}`)),
		Chat: datatypes.JSON(goopenai.NewChat().Json()),
	}

	err := storage.Db().Create(&chat).Error
	assert.NoError(t, err, "InsertChat should not return an error")
}

func help_init_storage(t *testing.T, isDebug bool) {
	var err error
	storage, err = NewUserStoragePG(dbProps, lo.Ternary(isDebug, logDebug, log), WithDebug(true))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NotNil(t, storage)
}

func help_create_fake_user(t *testing.T) *us.User {
	user := &us.User{
		Username:      faker.FirstName(),
		Email:         lo.ToPtr(faker.Email()),
		Password:      faker.Password(),
		LastLoginTime: lo.ToPtr(time.Unix(faker.RandomUnixTime(), 0)),
	}
	t.Logf("FakeUser: %s", user)
	return user
}

func help_create_fake_chat(t *testing.T, uid uint) *us.Chat {
	// ch := goopenai.NewChat().AddUserMessage("Hey! What is your best feature?").AddAssistantMessage("I am a language model AI. I can generate human-like text based on the input I receive.")
	// ch := goopenai.NewChat(prompt).AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
	chat := &us.Chat{
		UserID: uid,
		Chat:   datatypes.JSON(populateChatFakeData(goopenai.NewChat(prompt)).Json()),
		// Chat: datatypes.NewJSONType(populateChatFakeData(goopenai.NewChat(prompt))),
	}
	t.Logf("FakeChat: %s", utils.Json(chat))
	return chat
}

func populateChatFakeData(ch *goopenai.Chat) *goopenai.Chat {
	for _, v := range []int{0, 1, 2, 3, 45, 6, 7, 8, 9} {
		_ = v
		ch.AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
		ch.AddUserMessage(faker.Sentence()).AddAssistantMessage(faker.Sentence())
	}
	return ch
}

func help_list_env(t *testing.T) {
	t.Log(strings.Join(os.Environ(), "\n"))
}

func dropTables(t *testing.T) error {
	help_init_storage(t, false)
	// Dropping the Chat table. It's usually safer to start with the dependent tables first.
	if err := storage.Db().Migrator().DropTable(&us.Chat{}); err != nil {
		t.Error("Failed to drop Chat table:", err)
		return err
	}
	// Dropping the User table.
	if err := storage.Db().Migrator().DropTable(&us.User{}); err != nil {
		t.Error("Failed to drop User table:", err)
		return err
	}
	return nil
}

func dropTablesGP(t *testing.T) error {
	help_init_storage(t, false)
	if err := storage.Db().Migrator().DropTable(&us.RolePermission{}); err != nil {
		t.Error("Failed to drop Chat table:", err)
		return err
	}
	if err := storage.Db().Migrator().DropTable(&us.Group{}); err != nil {
		t.Error("Failed to drop Chat table:", err)
		return err
	}

	if err := storage.Db().Migrator().DropTable(&us.UserGroup{}); err != nil {
		t.Error("Failed to drop Chat table:", err)
		return err
	}
	// Dropping the User table.
	if err := storage.Db().Migrator().DropTable(&us.Permission{}); err != nil {
		t.Error("Failed to drop User table:", err)
		return err
	}
	return nil
}

func assert_chats(t *testing.T, err error, chats ...us.Chat) {
	assert.NoError(t, err)
	if assert.NotNil(t, chats) && assert.NotEmpty(t, chats) {
		t.Log(utils.JsonPrettyStr(chats))
	}
}

func assert_users(t *testing.T, err error, users ...*us.User) {
	assert.NoError(t, err)
	if assert.NotNil(t, users) && assert.NotEmpty(t, users) {
		t.Log(utils.JsonPrettyStr(users))
	}
}

func Test_db_props(t *testing.T) {
	props := &ddb.DBProperties{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "your_password",
		DBName:   "your_database",
		SSLMode:  "disable",
	}
	assert.NoError(t, helpers.ValidateWithLog(log.Rec(), props))
	props.Host = "000000"
	assert.Error(t, helpers.ValidateWithLog(log.Rec(), props))
}

func Test_IsTypeCorrect(t *testing.T) {
	help_init_storage(t, false)
	u, e := storage.GetUser(ctx, 1)
	assert.NoError(t, e)
	chat1 := u.Chats[0]
	chatOpenAI := chat1.Chat
	t.Logf("TYPE=%T", chatOpenAI)
	assert.IsType(t, &goopenai.Chat{}, chat1.Data())
	assert.IsType(t, []goopenai.Chat{}, u.ChatsOpenAI())
}

var prompt = `You are a customer support chatbot for the Ukrainian telecommunication operator "Lifecell".
Your primary goal is to provide answers to user queries using the information provided within triple quotes in the question. If this information is missing, you must determine the category of the query and respond with a structured answer indicating the lack of information.
Always maintain a polite and helpful tone, ensuring that subscribers feel valued and supported.
When provided with text within triple quotes, extract relevant details and formulate a response that directly addresses the user's query.
If the necessary information is missing, analyze the query to identify its category (e.g., Billing, Network Issues, Plan Details, Technical Support) and respond in the format: {"answer": "Insufficient information", "category": $category}.
Avoid making assumptions about missing information. Instead, guide users to provide the required details or direct them to the appropriate resources or Lifecell support channels.
Personalize your responses when possible, addressing users by name if provided, and tailoring advice or instructions to their specific circumstances.
Be efficient in providing accurate and relevant information, aiming to resolve queries in a single interaction when possible.You are a customer support chatbot for the Ukrainian telecommunication operator "Lifecell".
Your primary goal is to provide answers to user queries using the information provided within triple quotes in the question. If this information is missing, you must determine the category of the query and respond with a structured answer indicating the lack of information.
Always maintain a polite and helpful tone, ensuring that subscribers feel valued and supported.
When provided with text within triple quotes, extract relevant details and formulate a response that directly addresses the user's query.
If the necessary information is missing, analyze the query to identify its category (e.g., Billing, Network Issues, Plan Details, Technical Support) and respond in the format: {"answer": "Insufficient information", "category": $category}.
Avoid making assumptions about missing information. Instead, guide users to provide the required details or direct them to the appropriate resources or Lifecell support channels.
Personalize your responses when possible, addressing users by name if provided, and tailoring advice or instructions to their specific circumstances.
Be efficient in providing accurate and relevant information, aiming to resolve queries in a single interaction when possible.`
