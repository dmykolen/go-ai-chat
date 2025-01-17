package services

import (
	"context"
	"fmt"
	// w "w1/wvservice"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/dbchain"
	experemental "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools/experemental"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	dic "gitlab.dev.ict/golang/go-ai/logic/db_info_collector"
	"gitlab.dev.ict/golang/go-ai/models"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
	goai "gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
)

type RAGService struct {
	log                  *gologgers.Logger
	db                   *w.KnowledgeBase
	ws                   *biz.WSGetter
	ai                   *goai.Client
	dbCim                dic.DBSchemaInfoProvider
	llm                  llms.Model
	LogicsForDataSources []models.Logic
}

// NewRAGService - create new RAGService
func NewRAGService(log *gologgers.Logger, db *w.KnowledgeBase, ai *goai.Client) *RAGService {
	if db == nil {
		e := fmt.Errorf("(db *w.KnowledgeBase) - should not be null")
		log.Error(e)
		panic(e)
	}
	return &RAGService{log: log, db: db, ai: ai}
}

// InitWS - init WS
func (rag *RAGService) SetWS(ws *biz.WSGetter) *RAGService {
	rag.ws = ws
	return rag
}

func (rag *RAGService) LLM(llm llms.Model) *RAGService {
	rag.llm = llm
	return rag
}

func (rag *RAGService) DBCim(i dic.DBSchemaInfoProvider) *RAGService {
	rag.dbCim = i
	return rag
}

func (rag *RAGService) Info() {
	rag.log.Infof("RAGService:\n\tdb => %+v\n\tLogics count => %d", rag.db, len(rag.LogicsForDataSources))
}

// AppendLogic - append logic
func (rag *RAGService) AppendLogic(logic models.Logic) *RAGService {
	rag.LogicsForDataSources = append(rag.LogicsForDataSources, logic)
	return rag
}

// RunLogicsForDataSources - run logics for data sources
func (rag *RAGService) RunLogicsForDataSources(ctx context.Context, csf ...models.ContentSaverFunc) {
	for _, v := range rag.LogicsForDataSources {
		v.Process(ctx, csf...)
	}
}

// GetLogic - get logic
func (rag *RAGService) GetLogic(t models.LogicType) models.Logic {
	for _, v := range rag.LogicsForDataSources {
		if v.Type() == t {
			return v
		}
	}
	return nil
}

func (rag *RAGService) GetKB() *w.KnowledgeBase {
	return rag.db
}

func (rag *RAGService) CallAICimDBChanEventMsg(ctx context.Context, query string, chat *goai.Chat, userChan chan string, sendToChan func(string)) (any, error) {
	log := rag.log.RecWithCtx(ctx, "rag")
	log.Info("Start calling AI agents...")
	ctx = ailogic.AddToCtxUUIDAI(ctx, chat.ID)

	chain := dbchain.DbChainNew(rag.log, rag.llm, rag.dbCim,
		ailogic.WithName("DbCimChain"),
		// al.WithPrompt(defPrompt),
		ailogic.WithCtx(ctx),
		ailogic.WithLog(rag.log),
		ailogic.WithUserLogin("XXXXXX"),
		ailogic.WithAvailableTools(experemental.NewToolSqlRunner2(rag.log, rag.dbCim.G())))

	// mapResult, err := chain.Run(ctx, dbchain.PrepareInputsCM(query), ailogic.WithCBSse(ailogic.CallbackSSEStream), ailogic.WithSSEChan(userChan), ailogic.WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(rag.log))))
	mapResult, err := chain.Run(ctx, dbchain.PrepareInputsCM(query), ailogic.RunOptsInstantSSE(userChan, rag.log)...)
	if err != nil {
		log.Errorf("Error calling AI agents: %v", err)
		return "", err
	}

	log.Infof("AI agents mapResult[%s]: %v", chain.OutputKey, mapResult[chain.OutputKey])
	chat.AddUserMessage(query)
	chat.AddAssistantMessage(mapResult[chain.OutputKey].(string))

	return mapResult[chain.OutputKey], nil
}

// SearchInVectorAndAskAIStream - search in vector and ask ai stream
func (rag *RAGService) CallAICimDB(ctx context.Context, query string, chat *goai.Chat, userChan chan string) (any, error) {
	log := rag.log.RecWithCtx(ctx, "rag")
	log.Info("Start calling AI agents...")
	ctx = ailogic.AddToCtxUUIDAI(ctx, chat.ID)

	chain := dbchain.DbChainNew(rag.log, rag.llm, rag.dbCim,
		ailogic.WithName("DbCimChain"),
		// al.WithPrompt(defPrompt),
		ailogic.WithCtx(ctx),
		ailogic.WithLog(rag.log),
		ailogic.WithUserLogin("XXXXXX"),
		ailogic.WithAvailableTools(experemental.NewToolSqlRunner2(rag.log, rag.dbCim.G())))

	// mapResult, err := chain.Run(ctx, dbchain.PrepareInputsCM(query), ailogic.WithCBSse(ailogic.CallbackSSEStream), ailogic.WithSSEChan(userChan), ailogic.WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(rag.log))))
	mapResult, err := chain.Run(ctx, dbchain.PrepareInputsCM(query), ailogic.RunOpts(userChan, rag.log)...)
	if err != nil {
		log.Errorf("Error calling AI agents: %v", err)
		return "", err
	}

	log.Infof("AI agents mapResult[%s]: %v", chain.OutputKey, mapResult[chain.OutputKey])
	chat.AddUserMessage(query)
	chat.AddAssistantMessage(mapResult[chain.OutputKey].(string))

	return mapResult[chain.OutputKey], nil
}

func ConvertOpenAIMessageToChatMessage(openaiMsg *goai.Chat) (chatMessages []llms.ChatMessage) {
	for _, v := range openaiMsg.Messages {
		switch v.Role {
		case openai.RoleSystem:
			chatMessages = append(chatMessages, llms.SystemChatMessage{Content: v.Content})
		case openai.RoleUser:
			chatMessages = append(chatMessages, llms.HumanChatMessage{Content: v.Content})
		case openai.RoleAssistant:
			chatMessages = append(chatMessages, llms.AIChatMessage{Content: v.Content})
		}
	}
	return
}

// SearchInVectorAndAskAIStream - search in vector and ask ai stream
func (rag *RAGService) CallAIAgents(ctx context.Context, query string, chat *goai.Chat, userChan chan string, sendToChan func(string)) (any, error) {
	log := rag.log.RecWithCtx(ctx, "rag")

	log.Info("Start calling AI agents...")
	ctx = ailogic.AddToCtxUUIDAI(ctx, chat.ID)
	voipChain := ailogic.NewChainVoipExt(ctx, rag.llm, rag.log, rag.ws, rag.db)

	values := map[string]any{
		voipChain.GetKeyIn():          query,
		ailogic.PlaceholderForHistory: ConvertOpenAIMessageToChatMessage(chat),
	}

	mapResult, err := voipChain.Run(ctx, values,
		ailogic.WithCBSse(ailogic.CallbackSSEStream),
		ailogic.WithSSEChan(userChan),
		ailogic.WithCallbackToChannel(ailogic.CallbackSSEStreamWithFN),
		ailogic.WithSendToChan(sendToChan),
		ailogic.WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(rag.log)), chains.WithTemperature(0.2)))
	if err != nil {
		log.Errorf("Error calling AI agents: %v", err)
		return "", err
	}
	log.Debugf("AI agents mapResult: %s", utils.JsonPretty(mapResult))
	log.Infof("AI agents mapResult[%s]: %v", voipChain.GetKeyOut(), mapResult[voipChain.GetKeyOut()])
	chat.WithName(mapResult["userIntent"].(string)).AddUserMessage(query).AddAssistantMessage(mapResult[voipChain.GetKeyOut()].(string))

	return mapResult[voipChain.GetKeyOut()], nil
}

// SearchInVectorAndAskAIStream - search in vector and ask ai stream
func (rag *RAGService) SearchInVectorAndAskAIStream(ctx context.Context, query string, chat *goai.Chat, userChan chan string, fn ...goai.PromptGenFN) (string, error) {
	rid, ctx := utils.GetRidOrAdd(ctx)
	log := rag.log.WithField("rid", rid)
	list := rag.db.SearchInContentsHybrid(ctx, query, 1, w.FieldTitle, w.FieldContent)

	if len(list) == 0 {
		log.Warnf("Result searching in VectorDB => NOT_FOUND! q=[%s]", query)
		return "I'm so sorry :( I can't find anything for you.", nil
	}

	obj, err := rag.db.GetObjByID(ctx, w.IdOfFirstEl(list))
	if err != nil {
		log.Errorf("Error getting obj by id[%s]: %v", w.IdOfFirstEl(list), err)
		return "I'm so sorry :( I can't find anything for you.", err
	}

	if len(fn) == 0 {
		fn = append(fn, goai.CreatePromptGen(goai.UserTemplate1))
	}

	stream, e := rag.ai.AskAIStream(ctx, fn[0](w.ObjContent(obj), query), chat)
	if e != nil {
		log.Errorf("Error asking ai stream: %v", e)
		return "", e
	}
	log.Info("AI streamming...")

	// Process the stream
	dataCh, errCh := goai.ProcessStreamParallel(log, stream)

	// Handle data and errors
	for {
		select {
		case data, ok := <-dataCh:
			if !ok {
				log.Debug(">>> DATA_CH - NOT_OK")
				dataCh = nil
			} else {
				userChan <- data.Choices[0].Delta.Content
				chat.AddAssistantMessage(data.Choices[0].Delta.Content)
			}

			log.Tracef("%s", utils.Json(data))
		case err, ok := <-errCh:
			if !ok {
				log.Debug(">>> ERR_CH - NOT_OK")
				errCh = nil
			}
			log.Errorf("Error: %v", err)
		}
		// fmt.Printf(">>> dataCh == nil? -> [%t]; errCh == nil? -> [%t]\n", dataCh == nil, errCh == nil)
		if dataCh == nil && errCh == nil {
			log.Info(">>> dataCh == nil && errCh == nil")
			break
		}
	}

	return "", nil
}
