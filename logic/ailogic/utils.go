package ailogic

import (
	"context"
	"encoding/json"
	"fmt"

	// "github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/memory/sqlite3"
	"github.com/tmc/langchaingo/schema"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
)

type MapAny = map[string]any
type CtxKey string

const (
	defChunkSize    = 256
	defChunkOverlap = 50
)

const (
	_ctx_u_login CtxKey = "login"
	_ctx_u_cn    CtxKey = "chain_name"
	_ctx_u_cu    CtxKey = "uuidAI"
)

func memCreate(o *Options) {
	switch {
	case o.mem != nil:
		return
	case o.memHist != nil:
	case o.ctx != nil:
		o.memHist = CreateSqliteMem(o.ctx, o.name)
	default:
		o.mem = memory.NewConversationBuffer(memory.WithInputKey(o.inputKey), memory.WithOutputKey(o.outKey), memory.WithChatHistory(CreateSqliteMem(utils.GenerateCtxWithRid(), o.name)))
	}
	o.mem = memory.NewConversationBuffer(memory.WithInputKey(o.inputKey), memory.WithOutputKey(o.outKey), memory.WithChatHistory(o.memHist))
}

func CreateSqliteMem(ctx context.Context, chainName string) schema.ChatMessageHistory {
	return sqlite3.NewSqliteChatMessageHistory(
		sqlite3.WithSession(GetUUIDAI(ctx)), sqlite3.WithDB(DBMemory()), sqlite3.WithContext(ctx), sqlite3.WithTableName(chainName),
	)
}

func RunOptsInstantSSE(sseChannel chan string, log *gl.Logger) []RunOptFn {
	return []RunOptFn{
		WithCBSse(callbackhandlers.CallbackSSEStreamEventMsg),
		WithSSEChan(sseChannel),
		WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(log))),
	}
}

func RunOpts(sseChannel chan string, log *gl.Logger) []RunOptFn {
	return []RunOptFn{
		WithCBSse(CallbackSSEStream),
		WithSSEChan(sseChannel),
		WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(log))),
	}
}

func AddToCtxIfNotExists(ctx context.Context, key CtxKey, value string) context.Context {
	if ctx.Value(key) == nil && value != "" {
		return context.WithValue(ctx, key, value)
	}
	return ctx
}

func GetUUIDAI(ctx context.Context) string {
	uuid := utils.GetCtxRid(ctx, _ctx_u_cu)
	if uuid == "" {
		uuid = utils.UUID()
	}
	return uuid
}

func AddToCtxUUIDAI(ctx context.Context, uuid string) context.Context {
	return AddToCtxIfNotExists(ctx, _ctx_u_cu, uuid)
}

func AddToCtxCN(ctx context.Context, chainName string) context.Context {
	return AddToCtxIfNotExists(ctx, _ctx_u_cn, chainName)
}

func AddToCtxLogin(ctx context.Context, login string) context.Context {
	return AddToCtxIfNotExists(ctx, _ctx_u_login, login)
}

func AddToCtxLoginCn(ctx context.Context, login, chainName string) context.Context {
	// return AddToCtxIfNotExists(AddToCtxIfNotExists(ctx, _ctx_u_login, login), _ctx_u_cn, chainName)
	return AddToCtxCN(AddToCtxLogin(ctx, login), chainName)
}

func AddToCtxLoginCnCu(ctx context.Context, login, chainName, chatUUID string) context.Context {
	return context.WithValue(AddToCtxLoginCn(ctx, login, chainName), _ctx_u_cu, chatUUID)
}

func GetLCCFromCtx(ctx context.Context) (login, chainName, chatUUID string) {
	var ok bool
	if login, ok = ctx.Value(_ctx_u_login).(string); !ok {
		login = ""
	}
	if chainName, ok = ctx.Value(_ctx_u_cn).(string); !ok {
		chainName = ""
	}
	if chatUUID, ok = ctx.Value(_ctx_u_cu).(string); !ok {
		chatUUID = ""
	}
	return
}

func DefaultLoggerCallback() callbackhandlers.LoggerHandler {
	return callbackhandlers.NewLoggerHandler(gl.New(gl.WithLevel("debug"), gl.WithColor(), gl.WithOC()))
}

// func TextToChunks(dirFile string, chunkSize, chunkOverlap *int) ([]schema.Document, error) {
// 	file, err := os.Open(dirFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	docLoaded := documentloaders.NewText(file)
// 	split := textsplitter.NewRecursiveCharacter()
// 	split.ChunkSize = lo.Ternary(chunkSize == nil, defChunkSize, *chunkSize)
// 	split.ChunkOverlap = lo.Ternary(chunkOverlap == nil, defChunkOverlap, *chunkSize)
// 	docs, err := docLoaded.LoadAndSplit(context.Background(), split)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return docs, nil
// }

func ChatHistoryAsOpenai(prompt []llms.ChatMessage) (messages []openai.ChatMessage) {
	for _, v := range prompt {
		var buf openai.ChatMessage
		switch p := v.(type) {
		case llms.SystemChatMessage:
			buf = openai.ChatMessage{Role: "system", Content: p.Content}
		case llms.ToolChatMessage:
			buf = openai.ChatMessage{Role: "tool", Content: p.Content}
		case llms.AIChatMessage:
			buf = openai.ChatMessage{Role: "assistant", Content: p.Content}
		default:
			buf = openai.ChatMessage{Role: "user", Content: v.GetContent()}
		}
		messages = append(messages, buf)
	}
	return
}

func ChatHistoryAsStringSafe(prompt []llms.ChatMessage, withSystem ...bool) string {
	chat, err := ChatHistoryAsString(prompt, withSystem...)
	if err != nil {
		return ""
	}
	return chat
}
func ChatHistoryAsString(prompt []llms.ChatMessage, withSystem ...bool) (buf string, e error) {
	if utils.FirstOrDefault(false, withSystem...) {
		return llms.GetBufferString(prompt, "human", "ai")
	}
	if prompt[0].GetType() == llms.ChatMessageTypeSystem {
		buf, e = llms.GetBufferString(prompt[1:], "human", "ai")
	} else {
		buf, e = llms.GetBufferString(prompt, "human", "ai")
	}
	return
}

func MessageContentFromChat(prompt llms.PromptValue) (mcList []llms.MessageContent) {
	return MessageContentFromChatArray(prompt.Messages())
}

func MessageContentFromChatArray(prompt []llms.ChatMessage) (mcList []llms.MessageContent) {
	for _, msg := range prompt {
		role := msg.GetType()
		text := msg.GetContent()

		var mc llms.MessageContent

		switch p := msg.(type) {
		case llms.ToolChatMessage:
			mc = llms.MessageContent{
				Role: role,
				Parts: []llms.ContentPart{llms.ToolCallResponse{
					ToolCallID: p.ID,
					Content:    p.Content,
				}},
			}
		case llms.AIChatMessage:
			mc = llms.MessageContent{Role: role, Parts: []llms.ContentPart{}}
			if p.Content != "" {
				mc.Parts = append(mc.Parts, llms.TextContent{Text: p.Content})
			}

			for _, tc := range p.ToolCalls {
				mc.Parts = append(mc.Parts, llms.ToolCall{
					ID:           tc.ID,
					Type:         tc.Type,
					FunctionCall: tc.FunctionCall,
				})
			}
		default:
			mc = llms.MessageContent{
				Role:  role,
				Parts: []llms.ContentPart{llms.TextContent{Text: text}},
			}
		}
		// mcList[i] = mc
		mcList = append(mcList, mc)
	}
	return
}

func MessageContentToChatMessages(mcList []llms.MessageContent) (prompt []llms.ChatMessage) {
	for _, msg := range mcList {

		var chatMsg llms.ChatMessage

		switch msg.Role {
		case llms.ChatMessageTypeSystem:
			chatMsg = llms.HumanChatMessage{
				Content: msg.Parts[0].(llms.TextContent).Text,
			}
			prompt = append(prompt, chatMsg)
		case llms.ChatMessageTypeHuman:
			chatMsg = llms.HumanChatMessage{
				Content: msg.Parts[0].(llms.TextContent).Text,
			}
			prompt = append(prompt, chatMsg)
		case llms.ChatMessageTypeTool:
			chatMsg = llms.ToolChatMessage{
				ID:      msg.Parts[0].(llms.ToolCallResponse).ToolCallID,
				Content: msg.Parts[0].(llms.ToolCallResponse).Content,
			}
			prompt = append(prompt, chatMsg)
		case llms.ChatMessageTypeAI:

			for _, p := range msg.Parts {
				switch p := p.(type) {
				case llms.TextContent:
					chatMsg = llms.AIChatMessage{
						Content: p.Text,
					}
				case llms.ToolCall:
					chatMsg = llms.AIChatMessage{
						FunctionCall: p.FunctionCall,
						ToolCalls:    []llms.ToolCall{p},
					}

					// case llms.ToolCall:
					// 	chatMsg = llms.AIChatMessage{
					// 		FunctionCall: p.(llms.ToolCall).FunctionCall,
					// 		ToolCalls: []llms.ToolCall{
					// 			msg.Parts[0].(llms.ToolCall),
					// 		},
					// 	}
				}
				prompt = append(prompt, chatMsg)
			}

		}

	}
	return
}

func AnyToMap[T any](a any) map[string]T {
	v, ok := a.(map[string]T)
	if !ok {
		return nil
	}
	return v
}

func _valueFromMap[T any](m map[string]any, key string) (res T, b bool) {
	v, ok := m[key]
	if !ok {
		b = false
		return
	}
	return v.(T), true
}

// Add this helper to convert various types to string
func interfaceToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []interface{}:
		if len(val) > 0 {
			return fmt.Sprint(val[0])
		}
	case map[string]interface{}:
		if jsonStr, err := json.Marshal(val); err == nil {
			return string(jsonStr)
		}
	}
	return fmt.Sprint(v)
}

// Update valueFromMap to handle string type safely
func valueFromMap[T any](m map[string]any, key string) (res T, b bool) {
	v, ok := m[key]
	if !ok {
		return res, false
	}

	// Special handling for string type
	if _, isString := any(res).(string); isString {
		str := interfaceToString(v)
		return any(str).(T), true
	}

	// Handle other types
	if converted, ok := v.(T); ok {
		return converted, true
	}

	return res, false
}

func valueFromMapAny[T any](mapAny any, key string) (res T, b bool) {
	m := AnyToMap[any](mapAny)
	v, ok := m[key]
	if !ok {
		b = false
		return
	}
	return v.(T), true
}

func getString(m map[string]any, key string) (v string) {
	if vv, ok := valueFromMap[string](m, key); ok {
		v = vv
	}
	return
}

func getStringIf(m map[string]any, key string, key2 ...string) (v string) {
	if vv, ok := valueFromMap[string](m, key); ok {
		v = vv
		return
	}
	for _, k := range key2 {
		if vv, ok := valueFromMap[string](m, k); ok {
			v = vv
			return
		}
	}
	return
}

func mapAsStringMap(m map[string]any) map[string]string {
	res := make(map[string]string)
	for k, v := range m {
		res[k] = v.(string)
	}
	return res
}

// chunkString divides a string into chunks of a given size
func chunkString(s string, chunkSize int) []string {
	var chunks []string
	for len(s) > 0 {
		if len(s) < chunkSize {
			chunkSize = len(s)
		}
		chunks = append(chunks, s[:chunkSize])
		s = s[chunkSize:]
	}
	return chunks
}

// Helper function to split string into chunks
func _chunkString(s string, chunkSize int) []string {
	if chunkSize <= 0 {
		return []string{s}
	}
	var chunks []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

func Keys(m map[string]any) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}
