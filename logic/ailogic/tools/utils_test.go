package tools

import (
	"context"
	"database/sql"
	"testing"

	"github.com/caarlos0/env/v6"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory/sqlite3"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"

	"gitlab.dev.ict/golang/go-ai/logic/biz"
)

func getDB(ds ...string) *sql.DB {
	dbCon, err := sql.Open("sqlite3", lo.TernaryF(len(ds) > 0 && ds[0] != "", func() string { return ds[0] }, func() string { return "hst.db" }))
	if err != nil {
		panic(err)
	}
	return dbCon
}

var log = gologgers.Defult()
var toolsToCall = []llms.ToolCall{
	{ID: "call_ZR7LogRI4n8bl69dCWhGWR5X", Type: "function", FunctionCall: &llms.FunctionCall{Name: "getAccountData", Arguments: "{\"msisdn\":\"380930164453\"}"}},
	{ID: "call_ZR7LogRI4n8bl69dCWhGWR5Y", Type: "function", FunctionCall: &llms.FunctionCall{Name: "getAccountData", Arguments: "{\"msisdn\":\"380632107489\"}"}},
}
var toolResponse = &llms.ContentResponse{Choices: []*llms.ContentChoice{{ToolCalls: toolsToCall}}}

func TestExecuteToolCalls(t *testing.T) {
	rec := log.RecWithCtx(utils.GenerateCtxWithRid(), "TOOLS")
	availableTools := AvailableVoipTools
	messageHistory := []llms.MessageContent{}
	resp := toolResponse
	writeHst := true
	// uuid := utils.UUID()
	uuid := "9a79e2ba-5a09-47a1-b759-6026f8c992f8"
	hst := sqlite3.NewSqliteChatMessageHistory(sqlite3.WithSession(uuid), sqlite3.WithDB(getDB()), sqlite3.WithContext(utils.GenerateCtxWithRid()), sqlite3.WithTableName("agent_2"))
	t.Log(hst.Messages(context.Background()))

	Help_init_wsGetter(t)

	messageHistory = updateMessageHistory(context.Background(), messageHistory, hst, resp)

	messageHistory, err := ExecuteToolCalls(rec, availableTools, messageHistory, resp, writeHst, hst)
	if err != nil {
		t.Errorf("ExecuteToolCalls() error = %v", err)
	}
	t.Log(utils.JsonPrettyStr(messageHistory))
	t.Log("###################################")
	// t.Log(utils.JsonPrettyStr(messageHistory))
}

func updateMessageHistory(ctx context.Context, messageHistory []llms.MessageContent, chatHistory schema.ChatMessageHistory, resp *llms.ContentResponse) []llms.MessageContent {
	respchoice := resp.Choices[0]
	if chatHistory != nil {
		chatHistory.AddMessage(ctx, llms.AIChatMessage{Content: respchoice.Content, ToolCalls: respchoice.ToolCalls})
	}

	assistantResponse := llms.TextParts(llms.ChatMessageTypeAI, respchoice.Content)
	for _, tc := range respchoice.ToolCalls {
		assistantResponse.Parts = append(assistantResponse.Parts, tc)
	}
	log.Infof("assistantResponse: %s", utils.JsonPrettyStr(assistantResponse))
	return append(messageHistory, assistantResponse)
}

func Help_init_wsGetter(t *testing.T) {
	t.Helper()
	cimwsApiParams := &cimws.ApiParamas{}
	omwsApiParams := &omws.ApiParamas{}
	env.Parse(cimwsApiParams)
	env.Parse(omwsApiParams)
	t.Log(utils.JsonPrettyStr(omwsApiParams))
	// t.Log(utils.JsonPrettyStr(cimwsApiParams.DebugOFF().URL("http://dev-main-tm-cim-1.dev.ict:8080")))
	t.Log(utils.JsonPrettyStr(cimwsApiParams.DebugOFF().URL("http://dev-test-tm-cim-1.dev.ict:8080")))
	cimCl := cimws.NewClient(cimws.WithParams(cimwsApiParams), cimws.WithLog(log))
	cimCl.SetProxy("http://proxy.astelit.ukr:3128")
	wsGetter := biz.NewWSGetter(log, cimCl, omws.NewClient(omws.WithParams(omwsApiParams), omws.WithLogger(log)))
	InitTools(wsGetter, nil)
	t.Log("wsGetter is initialized")
}

func TestXxx(t *testing.T) {
	t.Log(utils.JsonPrettyStr(ToolFuncs))
}

func TestFMCSettings(t *testing.T) {
	// Setup
	rec := log.RecWithCtx(utils.GenerateCtxWithRid(), "TOOLS")
	Help_init_wsGetter(t)

	t.Run("Test FMC VOIP Settings", func(t *testing.T) {
		// Define test data
		toolCall := llms.ToolCall{
			ID:   "call_Z4dDIeHpoeEQwxyfrDtGX0Gz",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "get_FMC_VOIP_settings",
				Arguments: `{"msisdn":"380933780687","tariff":"CRP_PP_FMC_VOIP","contractNo":"300070023"}`,
			},
		}

		// Test normal operation
		response, err := GetFMCVoipSettings(rec, []byte(toolCall.FunctionCall.Arguments))
		assert.NoError(t, err)
		assert.NotEmpty(t, response)
		t.Logf("VOIP Settings Response: %s", response)

		// Test with invalid arguments
		invalidResponse, err := GetFMCVoipSettings(rec, []byte(`{invalid json}`))
		assert.Error(t, err)
		assert.Empty(t, invalidResponse)
	})

	t.Run("Test FMC MOBILE Settings", func(t *testing.T) {
		// Define test data
		toolCall := llms.ToolCall{
			ID:   "call_Jwjn061gOudYtx6crU8TuiEy",
			Type: "function",
			FunctionCall: &llms.FunctionCall{
				Name:      "get_FMC_MOBILE_settings",
				Arguments: `{"contractNo":"300070023","msisdn":"380933780678","tariff":"CRP_PP_FMC_MOBILE"}`,
			},
		}

		// Test normal operation
		response, err := GetFMCMobileSettings(rec, []byte(toolCall.FunctionCall.Arguments))
		assert.NoError(t, err)
		assert.NotEmpty(t, response)
		t.Logf("Mobile Settings Response: %s", response)

		// Test with invalid arguments
		invalidResponse, err := GetFMCMobileSettings(rec, []byte(`{invalid json}`))
		assert.Error(t, err)
		assert.Empty(t, invalidResponse)
	})

	t.Run("Test Integration with ExecuteToolCalls", func(t *testing.T) {
		// Define test response with both tool calls
		toolResponse := &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{
					ToolCalls: []llms.ToolCall{
						{
							ID:   "call_Z4dDIeHpoeEQwxyfrDtGX0Gz",
							Type: "function",
							FunctionCall: &llms.FunctionCall{
								Name:      "get_FMC_VOIP_settings",
								Arguments: `{"msisdn":"380933780687","tariff":"CRP_PP_FMC_VOIP","contractNo":"300070023"}`,
							},
						},
						{
							ID:   "call_Jwjn061gOudYtx6crU8TuiEy",
							Type: "function",
							FunctionCall: &llms.FunctionCall{
								Name:      "get_FMC_MOBILE_settings",
								Arguments: `{"contractNo":"300070023","msisdn":"380933780678","tariff":"CRP_PP_FMC_MOBILE"}`,
							},
						},
					},
				},
			},
		}

		messageHistory := []llms.MessageContent{}
		updatedHistory, err := ExecuteToolCalls(rec, AvailableVoipTools, messageHistory, toolResponse, false, nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, updatedHistory)
		assert.Len(t, updatedHistory, 2)

		// Log responses for inspection
		t.Logf("Updated Message History: %s", utils.JsonPrettyStr(updatedHistory))
	})
}
