package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/gorm"

	"gitlab.dev.ict/golang/go-ai/logic/biz"
)

// ToolContext encapsulates tool dependencies.
type ToolContext struct {
	WS     *biz.WSGetter
	DB     *w.KnowledgeBase
	CimDB  *gorm.DB
	IsTest bool
}

// Global instance for tool context
var toolContext ToolContext
var once sync.Once

// InitToolContext initializes the global tool context.
func InitToolContext(ws *biz.WSGetter, db *w.KnowledgeBase, cimDB *gorm.DB, isTest bool) {
	once.Do(func() {
		toolContext = ToolContext{
			WS:     ws,
			DB:     db,
			CimDB:  cimDB,
			IsTest: isTest,
		}
	})
}

// ValidateContext ensures the necessary dependencies are initialized.
func (tc *ToolContext) ValidateContext() error {
	if tc.WS == nil {
		return fmt.Errorf("web service (WS) is not initialized")
	}
	if tc.DB == nil {
		return fmt.Errorf("vector database is not initialized")
	}
	if tc.CimDB == nil {
		return fmt.Errorf("SQL database is not initialized")
	}
	return nil
}

func (tc *ToolContext) ValidateVectorDB() {
	if tc.DB == nil && !tc.IsTest {
		panic("VectorDB is nil! Please init `db *w.KnowledgeBase` in 'logic/ailogic/tools/tools.go' before using tools!")
	}
}

func (tc *ToolContext) ValidateWS() {
	if tc.WS == nil && !tc.IsTest {
		panic("WSGetter is nil! Please init `ws *biz.WSGetter` in 'logic/ailogic/tools/tools.go' before using tools!")
	}
}

func (tc *ToolContext) ValidateCimDB() {
	if tc.CimDB == nil && !tc.IsTest {
		panic("CIM database is nil! Please init `cimDB *gorm.DB` in 'logic/ailogic/tools/tools.go' before using tools!")
	}
}

func runSQLQuery(ctx context.Context, db *gorm.DB, query string) (res []map[string]interface{}, err error) {
	err = db.WithContext(ctx).Raw(query).Scan(&res).Error
	return
}

func MakeToolCallMessageContent(toolCall llms.ToolCall, content string) llms.MessageContent {
	return llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    content,
			},
		},
	}
}

func validateWS() {
	if ws == nil && !isTestMode {
		panic("WSGetter is nil! Please init `ws *biz.WSGetter` in 'logic/ailogic/tools/tools.go' before using tools!")
	}
}

func validateDB() {
	if db == nil && !isTestMode {
		panic("VectorDB is nil! Please init `db *w.KnowledgeBase` in 'logic/ailogic/tools/tools.go' before using tools!")
	}
}

func makeErrResp(err string, tc llms.ToolCall) string {
	var resp struct {
		Error    string `json:"error"`
		ToolName string `json:"tool_name"`
		Args     string `json:"args"`
	}
	resp.Error = err
	resp.ToolName = tc.FunctionCall.Name
	resp.Args = tc.FunctionCall.Arguments
	return utils.JsonStr(&resp)
}

func ExecuteToolCalls(rec *gologgers.LogRec, availableTools map[string]ToolCallHandler, messageHistory []llms.MessageContent, resp *llms.ContentResponse, writeHst bool, hst schema.ChatMessageHistory) ([]llms.MessageContent, error) {
	rec.Info("Executing tool calls...")
	for _, tc := range resp.Choices[0].ToolCalls {
		rec.Infof("Call tool[%s] '%s' with args: %s", tc.ID, tc.FunctionCall.Name, tc.FunctionCall.Arguments)

		var err error
		var response string
		handler, ok := availableTools[tc.FunctionCall.Name]
		if !ok {
			err = fmt.Errorf("unsupported tool: %s", tc.FunctionCall.Name)
			rec.Error(err)
			response = makeErrResp(err.Error(), tc)
			// continue
		} else {
			response, err = handler(rec, []byte(tc.FunctionCall.Arguments))
			if err != nil {
				// e = err
				rec.Errorf("Error executing tool call %s: %v", tc.FunctionCall.Name, err)
				response = makeErrResp(err.Error(), tc)
				// continue
			}
		}

		rec.Infof("Is_writeHst=%t is_hst_not_nil=%t hst_type=%T", writeHst, hst != nil, hst)
		if writeHst && hst != nil {
			hst.AddMessage(rec.Ctx, llms.ToolChatMessage{ID: tc.ID, Content: response})
		}
		messageHistory = append(messageHistory, MakeToolCallMessageContent(tc, response))
		rec.WithData(gologgers.M{"isSuccess": err == nil}).Infof("FINISH call tool[%s]!", tc.ID)
	}
	return messageHistory, nil
}

// ExecuteToolCalls processes all tool calls in parallel.
func ExecuteToolCallsInParallel(rec *gologgers.LogRec, availableTools map[string]ToolCallHandler, messageHistory []llms.MessageContent, resp *llms.ContentResponse, writeHst bool, hst schema.ChatMessageHistory) ([]llms.MessageContent, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)

	rec.Info("Executing tool calls...")
	for _, tc := range resp.Choices[0].ToolCalls {
		wg.Add(1)
		go func(tc llms.ToolCall) {
			defer wg.Done()
			rec.Infof("Processing tool[%s]: %s", tc.ID, tc.FunctionCall.Name)

			handler, exists := availableTools[tc.FunctionCall.Name]
			if !exists {
				mu.Lock()
				errs = append(errs, fmt.Errorf("unsupported tool: %s", tc.FunctionCall.Name))
				mu.Unlock()
				return
			}

			response, err := handler(rec, []byte(tc.FunctionCall.Arguments))
			if err != nil {
				rec.Errorf("Error executing tool [%s]: %v", tc.FunctionCall.Name, err)
				response = makeErrResp(err.Error(), tc)
			}

			if writeHst && hst != nil {
				hst.AddMessage(rec.Ctx, llms.ToolChatMessage{ID: tc.ID, Content: response})
			}

			mu.Lock()
			messageHistory = append(messageHistory, MakeToolCallMessageContent(tc, response))
			mu.Unlock()
		}(tc)
	}

	wg.Wait()

	if len(errs) > 0 {
		return messageHistory, fmt.Errorf("errors occurred during tool execution: %v", errs)
	}

	return messageHistory, nil
}
