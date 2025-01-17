package ailogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.dev.ict/golang/libs/utils"
)

// var ctx = utils.GenerateCtxWithRid()

func TestInitNewChatForChain1(t *testing.T) {
	t.Run("Test with new chat ID", func(t *testing.T) {
		chatUUID, err := ChatForChain(ctx, "MainChain", "JohnDoe", "12345-67890")
		help_results(t, chatUUID, err)
		assert.NoError(t, err)
		assert.Equal(t, "12345-67890", chatUUID)
	})

	t.Run("Test with default chat ID", func(t *testing.T) {
		chatUUID, err := ChatForChain(ctx, "MainChain", "JaneDoe")

		help_results(t, chatUUID, err)
		assert.NoError(t, err)
		assert.NotEmpty(t, chatUUID)
	})

	t.Run("Test with existing chat ID", func(t *testing.T) {
		chatUUID, err := ChatForChain(ctx, "MainChain", "ExistingUser")
		help_results(t, chatUUID, err)
		chatUUID, err = ChatForChain(ctx, "NEWuser", "ExistingUser", chatUUID)
		help_results(t, chatUUID, err)
		assert.Error(t, err)
		assert.Equal(t, "", chatUUID)
	})

	t.Run("Test with empty chain name", func(t *testing.T) {
		chatUUID, err := ChatForChain(ctx, "", "NoChainUser")
		help_results(t, chatUUID, err)
		assert.Error(t, err)
		assert.Equal(t, "", chatUUID)
	})

	t.Run("Test with empty login", func(t *testing.T) {
		chatUUID, err := ChatForChain(ctx, "MainChain", "")

		help_results(t, chatUUID, err)
	})

	t.Run("Test with empty chain name and login", func(t *testing.T) {
		t.Log("CHAIN_UUID =>", ctx.Value(_ctx_u_cu))
		chatUUID, err := ChatForChain(ctx, "MainChain", "")
		help_results(t, chatUUID, err)

	})
}

func help_results(t *testing.T, chatId any, err error) {
	t.Helper()
	t.Logf("Chat ID: %v; ERR: %v", chatId, err)
}

func TestGetChainChatFromDB(t *testing.T) {
	chainName := "MainChain"
	login := "JaneDoe"

	t.Run("Test with existing chain and login", func(t *testing.T) {
		chats, err := GetChainChatFromDB(ctx, chainName, login)
		help_results(t, chats, err)
		assert.NoError(t, err)
		assert.NotEmpty(t, chats)
	})

	t.Run("Test with non-existing chain and login", func(t *testing.T) {
		chats, err := GetChainChatFromDB(ctx, "NonExistingChain", "NonExistingUser")
		help_results(t, chats, err)
		assert.NoError(t, err)
		assert.Empty(t, chats)
	})
}
func TestIsExistsChatUUID(t *testing.T) {
	chainName := "MainChain"
	login := "JohnDoe"
	chatUUID := "12345-67890"

	t.Run("Test with existing chat UUID", func(t *testing.T) {
		exists, err := IsExistsChatUUID(ctx, chainName, login, chatUUID)
		help_results(t, exists, err)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("Test with non-existing chat UUID", func(t *testing.T) {
		nonExistingChatUUID := "98765-43210"
		exists, err := IsExistsChatUUID(ctx, chainName, login, nonExistingChatUUID)
		help_results(t, exists, err)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestGetAllByLoginAndChain(t *testing.T) {
	login := "JaneDoe"
	chainName := "MainChain"

	t.Run("Test all", func(t *testing.T) {
		for i := range 10 {
			t.Log(i)
			t.Log(ChatForChain(ctx, "DBCimChain", "dmykolen"))
		}
		users, err := SelectAllUsers(ctx)
		help_results(t, users, err)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
		t.Log(utils.JsonPrettyStr(users))
	})
	t.Run("Test with existing login and chain name", func(t *testing.T) {
		users, err := GetAllByLoginAndChain(ctx, login, chainName)
		help_results(t, users, err)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
	})
	t.Run("Test with non-existing login and chain name", func(t *testing.T) {
		users, err := GetAllByLoginAndChain(ctx, "NonExistingUser", "NonExistingChain")
		help_results(t, users, err)
		assert.NoError(t, err)
		assert.Empty(t, users)
	})
}

func TestXxx2(t *testing.T) {
	tx := GetSqliteDB().Migrator().DropTable(&User{})
	t.Log("ERR:", tx.Error())
}
func TestXxx(t *testing.T) {
	ctx := AddToCtxLoginCnCu(ctx, "JohnDoe", "MainChain", "12345-67890")
	help_prettyPrintStruct_T(t, ctx)
	ctx = AddToCtxLoginCnCu(ctx, "JohnDoe2", "MainChain2", "12345-67890")
	help_prettyPrintStruct_T(t, ctx)

	login, chainName, chatUUID := GetLCCFromCtx(ctx)
	t.Logf("Login: %s; ChainName: %s; ChatUUID: %s", login, chainName, chatUUID)
}
