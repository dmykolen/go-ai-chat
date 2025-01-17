package handlers

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic"
	"gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/libs/utils"
)

// GetUserByName gets a user by username.
// @Summary Get user by username
// @Description Retrieves a user by their username, optionally including their chats.
// @Tags users
// @Accept json
// @Produce json
// @Param username path string true "Username"
// @Param with_chats query bool false "Include Chats" default(false)
// @Success 200 {object} models.Response "Success response with user data."
// @Failure 400 {object} models.Response "Error: Bad request."
// @Failure 404 {object} models.Response "Error: User not found."
// @Failure 500 {object} models.Response "Error: Internal server error."
// @Router /users/{username} [get]
func (a *AppHandler) GetUserByName(c *fiber.Ctx) error {
	log := help.Log(c)
	un := c.Params("username")
	u, err := a.uStorage.FindUserByUsername(log.Ctx, un, c.QueryBool("with_chats"))
	if err != nil {
		log.Errorf("Error getting user by username[%s]: %v", un, err)
		return c.JSON(models.RespErrDB.WithData(err))
	}
	return c.JSON(models.RespOK.WithData(u))
}

// /users/chats/{username}
// /users/chats/{username}?type=UUID_ONLY
func (a *AppHandler) GetUserChats(c *fiber.Ctx) error {
	un := c.Params("username")
	u, err := a.uStorage.FindUserByUsername(help.Log(c).Ctx, un, true)
	if err != nil {
		help.Log(c).Errorf("Error getting user by username[%s]: %v", un, err)
		return c.JSON(models.RespErrDB.WithData(err))
	}
	if c.Query("type") == "UUID_ONLY" {
		return c.JSON(models.RespOK.WithData(u.ChatsUUID()))
	}
	return c.JSON(models.RespOK.WithData(u.ChatsOpenAI()))
}

// /users/:username/chats/:type
// /users/:username/chats/a7bbc185-92f9-45f7-9010-3196a8a12596
func (a *AppHandler) GetChats(c *fiber.Ctx) error {
	un, t := c.Params("username", GetUser(c).Login), c.Params("type")
	help.Log(c).Infof("search chats... input_params: username=%s; type=%s", un, t)
	u, err := a.uStorage.FindUserByUsername(help.Log(c).Ctx, un, true)
	if err != nil {
		help.Log(c).Errorf("Error getting user by username[%s]: %v", un, err)
		return c.JSON(models.RespErrDB.WithData(err))
	}

	switch {
	case t == "shortinfo":
		help.Log(c).Infof("getting chats by chainName[%s] and userId: %v", "VoipAgents", u.ID)
		chats, err := a.uStorage.GetChatsByChainAndUserId(help.Log(c).Ctx, "VoipAgents", int(u.ID))
		if err != nil {
			help.Log(c).Errorf("Error getting chats: %v", err)
			return c.JSON(models.RespErrDB.WithData(err))
		}
		return c.JSON(models.RespOK.WithData(us.NewChatsInfo(chats)))
		// return c.JSON(models.RespOK.WithData(u.ChatsInfo()))
	case t == "shortinfo-db-chain":
		users, err := ailogic.GetAllByLoginAndChain(help.Log(c).Ctx, un, "DbCimChain")
		help.Log(c).WithError(err).Infof("sqlite while fetching users by chainname and userName! len_chats=%d", len(users))
		return c.JSON(models.RespOK.WithData(users))
	case regexp.MustCompile(`[a-f0-9]{8}-([a-f0-9]{4}-){3}[a-f0-9]{12}`).MatchString(t):
		return c.JSON(models.RespOK.WithData(u.GetChatOpenAI(t)))
	default:
		return c.JSON(models.RespOK.WithData(u.ChatsOpenAI()))
	}
}

func (a *AppHandler) GetChatHTML3(c *fiber.Ctx) map[string]interface{} {
	un, t := c.Params("username", GetUser(c).Login), c.Params("type")
	help.Log(c).Infof("search chats... input_params: username=%s; type=%s", un, t)
	chat, err := a.uStorage.GetChatByUUID(help.Log(c).Ctx, t)
	if err != nil {
		help.Log(c).Errorf("Error getting user by username[%s]: %v", un, err)
		return nil
	}
	help.Log(c).Infof("Chat: %s", chat.Chat)
	var m map[string]interface{}
	e := json.Unmarshal(chat.Chat, &m)
	if e != nil {
		help.Log(c).Errorf("unmarshalling chat failed! Err: %v", e)
	}
	help.Log(c).Debugf("CHAT_AS_MAP=> %#v", m)
	return m
}

func (a *AppHandler) RateChat(c *fiber.Ctx) error {
	// {"chatID":"937bcc28-abcc-42e1-92a1-e0a5c5d5ddb1","chatRating":"3","chatIdx":3}
	var body struct {
		ChatID     string      `json:"chatID"`
		ChatRating interface{} `json:"chatRating"`
		ChatIdx    int         `json:"chatIdx"`
	}

	log := help.Log(c)

	log.Infof("User want rate chat! INPUT Body=[%s]", c.Body())
	if err := c.BodyParser(&body); err != nil {
		log.Errorf("Error while parsing request body: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error while parsing request body", err))
	} else {
		log.Info("Marshalled input request:", utils.JsonStr(body))
	}

	chat, err := a.uStorage.GetChatByUUID(log.Ctx, body.ChatID)
	if err != nil {
		log.Errorf("Error while getting chat by UUID: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error while getting chat by UUID", err))
	}

	log.Infof("chat id=%d", chat.ID)
	r, err := help.ConvertToInt(body.ChatRating)
	if err != nil {
		log.Errorf("Error while converting rating to int. ERR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error while converting rating to int", err))
	}
	log.Infof("Start update message with id=%d and rating=%v", chat.ID, r)
	if err := a.uStorage.UpdateMessageFieldInChat(log.Ctx, chat.ID, body.ChatIdx, "rating", r); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error updateing message rating", err))
	}

	return c.JSON(models.RespOK)
}

func (a *AppHandler) GetChatHTML(c *fiber.Ctx) error {
	un, t := c.Params("username"), c.Params("type")
	help.Log(c).Infof("search chats... input_params: username=%s; type=%s", un, t)
	u, err := a.uStorage.FindUserByUsername(help.Log(c).Ctx, un, true)
	if err != nil {
		help.Log(c).Errorf("Error getting user by username[%s]: %v", un, err)
		return c.JSON(models.RespErrDB.WithData(err))
	}
	help.Log(c).Infof("Chat: %s", utils.JsonPrettyStr(u.GetChatOpenAI(t)))
	return c.Render("chat", u.GetChatOpenAI(t), "layouts/main")
}

func (a *AppHandler) GetUsersFromAppStoreForUsers(c *fiber.Ctx) error {
	userMap := map[string]interface{}{}
	appStoreForUsers.Range(func(key, value any) bool {
		u := *value.(*User)
		u.Password = strings.Repeat("*", len(u.Password))
		userMap[key.(string)] = u
		return true
	})
	return c.JSON(models.RespOK.WithData(userMap))
}

// GetUserPhoto retrieves and serves a user's photo
func (ah *AppHandler) GetUserPhoto(c *fiber.Ctx) error {
	help.Log(c).Infof("Get user photo by id: %s", c.Params("id"))
	userID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.RespErrDB.WithData(err))
	}

	userPhoto, err := ah.Store().GetUserPhoto(help.Log(c).Ctx, uint(userID))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.R(400, "DB error while getting photo", err))
	}

	if userPhoto == nil || len(userPhoto.Data) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(models.RespErrDB.WithData(err))
	}

	c.Set("Content-Type", "image/"+userPhoto.MimeType)
	return c.Send(userPhoto.Data)
}

func (ah *AppHandler) GetChatByUUID(c *fiber.Ctx) error {
	uid := c.Params("uuid")
	var data any
	var err error
	if uid == "" {
		data, err = ah.Store().GetAllChats(help.Log(c).Ctx)
	} else {
		data, err = ah.Store().GetChatByUUID(help.Log(c).Ctx, uid)
	}

	if err != nil {
		return c.JSON(models.RespErrDB.WithData(err))
	}
	return c.JSON(models.RespOK.WithData(data))
}
