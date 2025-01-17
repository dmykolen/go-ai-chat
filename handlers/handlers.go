package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gookit/goutil"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic"
	"gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/go-ai/models/sse"
	"gitlab.dev.ict/golang/go-ai/services"
	ad "gitlab.dev.ict/golang/libs/goldap"
	"gitlab.dev.ict/golang/libs/gologgers"
	goai "gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
)

type callAiLogicFn func(ctx context.Context, query string, chat *goai.Chat, userChan chan string, sendToChan func(string)) (any, error)

// AppHandler - struct for handling requests
type AppHandler struct {
	log         *gologgers.Logger
	ai          *goai.Client
	rag         *services.RAGService
	uStorage    us.UserStorage
	adClient    *ad.LDAPConf
	rbacHandler *RBACHandler
	hbInterval  int
}

// optsFn - function for setting options for AppHandler
type optsFn func(*AppHandler)

// AppHandlerOptsFunc - function for setting options for AppHandler
func NewAppHandler(opts ...optsFn) *AppHandler {
	a := &AppHandler{hbInterval: 2}
	for _, opt := range opts {
		opt(a)
	}
	a.rbacHandler = NewRBACHandler(WithLog(a.log), WithStore(a.uStorage))
	return a
}

func WithLogger(l *gologgers.Logger) optsFn { return func(a *AppHandler) { a.log = l } }
func WithRAG(r *services.RAGService) optsFn { return func(a *AppHandler) { a.rag = r } }
func WithPGDB(ustor us.UserStorage) optsFn  { return func(a *AppHandler) { a.uStorage = ustor } }
func WithAI(ai *goai.Client) optsFn         { return func(a *AppHandler) { a.ai = ai } }
func WithAD(adClient *ad.LDAPConf) optsFn   { return func(a *AppHandler) { a.adClient = adClient } }

func (a *AppHandler) Store() us.UserStorage     { return a.uStorage }
func (a *AppHandler) RBACHandler() *RBACHandler { return a.rbacHandler }

// AskDB - Ask DB handler. Handle requests from client with question about database. Search related information in DB and return it to the client
// @Summary Ask DB
// @Description Handle requests from client with question about database
// @Tags AI
// @Accept json
// @Produce json
// @Param body body models.AIRequest true "Request body"
// @Success 200 {object} models.AIResponse
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /ask-db [post]
func (a *AppHandler) AskDB(c *fiber.Ctx) error {
	return a.processUserAIRequest(c, "DbCimChain", a.rag.CallAICimDBChanEventMsg)
	// return a.processUserAIRequest(c, "DbCimChain", a.rag.CallAICimDBChanEventMsg, getUser(c).ChanWithSSEMsg)
}

// AskAiVoIP_NEW - Ask AI VoIP handler. Handle requests from client with question about VoIP. Search related information in DB and send it to AI
// @Summary Ask AI VoIP
// @Description Handle requests from client with question about VoIP
// @Tags AI
// @Accept json
// @Produce json
// @Param body body models.AIRequest true "Request body"
// @Success 200 {object} models.AIResponse
// @Failure 400 {object} models.Response
// @Failure 500 {object} models.Response
// @Router /ask-ai-voip [post]
func (a *AppHandler) AskAiVoIP_NEW(c *fiber.Ctx) error {
	return a.processUserAIRequest(c, "VoipAgents", a.rag.CallAIAgents)
}

func sendToChanFN(c *fiber.Ctx, evtType EventType, tabId string) func(string) {
	user := getUser(c)
	if user == nil {
		help.Log(c).Error("User not found in context")
		return nil
	}
	help.Log(c).Infof("Create func [sendToChanFN] for user %s; tabId=%s; eventType=%v", user.Login, tabId, evtType)

	// Create a function that sends a message to the user.ChanEventMsg channel
	return func(msg string) {
		eventMsg := sse.Event{
			Type:  evtType,
			Msg:   msg,
			TabId: tabId,
		}
		user.ChanEventMsg <- eventMsg
	}
}

// processUserAIRequest handles the AI request from the user.
// It parses the incoming request body to extract the user's question and chat ID,
// retrieves the user from the database, and manages chat sessions.
// Depending on whether a chat ID is provided, it either creates a new chat
// or retrieves an existing one from the database. The function then calls
// the AI logic asynchronously and updates the chat with the AI's response.
//
// Parameters:
//   - c: The Fiber context containing request and response information.
//   - chainName: A string representing the name of the AI processing chain.
//   - callAI: A function that encapsulates the AI logic to be executed.
//   - userChan: A channel for sending messages back to the user.
//
// Returns:
//   - An error if any issues occur during processing, otherwise returns a JSON response
//     indicating the status and the chat ID.
func (a *AppHandler) processUserAIRequest(c *fiber.Ctx, chainName string, callAI callAiLogicFn) error {
	log := help.Log(c)
	var body models.AIRequest
	log.Infof("Run ai_logic[%s] to process user request! req_body=[%s]", chainName, c.Body())
	if err := c.BodyParser(&body); err != nil {
		log.Errorf("Error while parsing request body: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error while parsing request body", err))
	}
	log.Infof("Marshalled input request: %s", utils.Json(body))

	u := getUser(c)
	ctx := ailogic.AddToCtxLogin(log.Ctx, u.Login)
	userFromDB, err := a.uStorage.GetUserWithChatsByUserName(ctx, u.Login)
	if err != nil {
		log.WithError(err).Errorf("Error fetching user from DB")
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(500, "Error fetching user from DB", err))
	}
	log.Infof("user from DB => Username=%s; role=%s id=%d total_chats=%d", userFromDB.Username, userFromDB.LegacyRole, userFromDB.ID, len(userFromDB.Chats))

	var chat *goai.Chat
	var chatDB *us.Chat

	switch {
	case body.ChatId == "":
		log.Warn("ChatId is empty. Create new chat")
		chat = goai.NewChatEmpty()
	case body.ChatId != "":
		log.Warn("ChatId is not empty. Get chat from DB")
		chatDB = userFromDB.GetChat(body.ChatId)
		log.Infof("ChatDB: %s", utils.Json(chatDB))
		chat = chatDB.Data()
	}
	log.Debugf("Chat BEFORE call AI logic: %s", utils.Json(chat))

	postToChannel := sendToChanFN(c, sse.EvtChatGptResp, body.TabId)

	go func() {
		resp, err := callAI(ctx, body.Question, chat, u.ChanMessages, postToChannel)
		log.WithError(err).Infof("Response from AI: %s", utils.JsonPretty(resp))
		if err != nil {
			log.Errorf("Error calling AI logic: %v", err)
			postToChannel("__Service temporarily unavailable. Please try again later...__")
			return
		}

		postToChannel("######")
		log.Infof("Finish call AI! countMessages=%d LastMsg=%s", len(chat.Messages), utils.JsonPretty(chat.GetLastMessage()))

		if chatDB != nil {
			if err = a.uStorage.AddMessageToChat(ctx, chatDB.ID, chat.GetMessages()[len(chat.GetMessages())-2]); err == nil {
				a.uStorage.AddMessageToChat(ctx, chatDB.ID, chat.GetLastMessage())
			}
		} else {
			a.uStorage.CreateChat(ctx, us.NewChat(userFromDB.ID, chat).WithChainName(chainName))
		}
	}()

	return c.JSON(fiber.Map{"status": "OK", "chatId": chat.ID})
}

func (a *AppHandler) AnnounceToUsers(c *fiber.Ctx) error {
	log := help.Log(c)
	var body struct {
		Announce string `json:"announce"`
	}
	log.Infof("User asked to announce to all users! Marshalling to struct Body=[%s]", c.Body())
	if err := c.BodyParser(&body); err != nil {
		log.Errorf("Error while parsing request body: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(models.R(400, "Error while parsing request body", err))
	}

	u := getUser(c)
	if u == nil {
		log.Error("User not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(models.R(401, "You are UNAUTHORIZED user"))
	}

	go func() {
		log.Info("Start announce to all users...")
		appStoreForUsers.Range(func(key, v interface{}) bool {
			user := v.(*User)
			user.mu.Lock()
			defer user.mu.Unlock()

			for connID, w := range user.ActiveConns {
				log.Infof("Send announce -> user=%s connectionId=%s", user.Login, connID)
				sse.NewEventMsg("", sse.EvtAnnounce, body.Announce, 0).SendSSE(log, "", "", w)
				w.Flush()
			}
			return true
		})
		log.Info("Finish announce to all users...")
	}()

	return c.JSON(fiber.Map{"status": "OK"})
}

// [DEPRECATED]
//
// ChatGPT - ChatGPT handler
// @Description Handle ChatGPT requests
// @Tags AI
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Router /chatgpt [post]
func (a *AppHandler) ChatGPT(c *fiber.Ctx) error {
	log := help.Log(c)
	var body struct {
		Question string `json:"userRequest"`
	}
	log.Infof("User asked for chatgpt! Body: %s; BodyJsonParse: %v", c.Body(), c.BodyParser(&body))
	u := getUser(c)

	go func() {
		ch := make(chan []byte)
		rp := "../go-ai/assets/text.txt"
		file := goutil.Must(filepath.Abs(rp))
		log.Warn("FILE:", file)
		go help.ReadFileBuffered(file, ch)

		for data := range ch {
			log.Debugf("Read from chan data: %s", utils.StrCut(string(data), 10))
			// u.ChanMessages <- string(data)
			u.ChanMsgBB <- data
		}
		u.ChanMsgBB <- []byte("######")
	}()

	return c.JSON(fiber.Map{"status": "OK"})
}

// [DEPRECATED]
func (a *AppHandler) HandleSTT(c *fiber.Ctx) error {
	log := help.Log(c)
	log.Warn("Start HandleSTT => process audio file and send it to AI for transcription")
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString("File upload error")
	}
	mf, err := file.Open()
	if err != nil {
		return c.Status(400).SendString("File open error")
	}
	defer mf.Close()

	// Copy the file content to the buffer
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, mf); err != nil {
		return c.Status(400).SendString("File copy error")
	}
	log.Infof("file copy to buffer. length=%d", buf.Len())

	// Store the file
	c.SaveFile(file, fmt.Sprintf("./uploads/%s_%s", time.Now().Format("2006-01-02_15-04-05"), file.Filename))

	// Send the file to the AI
	text := a.ai.AudioTranscript(context.Background(), "audio.wav", buf)
	return c.SendString(text)
}
