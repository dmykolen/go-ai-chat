package ailogic

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
)

var (
	prevMsgs = []llms.ChatMessage{
		llms.SystemChatMessage{Content: "Sys message"},
		llms.HumanChatMessage{Content: "Як стати CEO Lifecell?"},
		llms.AIChatMessage{Content: "Вибачте, але я можу допомогти лише з питаннями, що стосуються підтримки клієнтів, телекомунікацій або послуг Lifecell. Якщо у вас є питання щодо наших послуг або технічної підтримки, будь ласка, дайте знати! 😊"},
		llms.HumanChatMessage{Content: "Який тариф на номері 380632107489?"},
	}
	prevMsgs2 = []llms.ChatMessage{
		llms.HumanChatMessage{Content: uq[0]},
		llms.AIChatMessage{Content: airesp[0]},
	}
)

var promptWithPlaceHolder = prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
	prompts.NewSystemMessagePromptTemplate("You are a company branding design wizard.", nil),
	prompts.MessagesPlaceholder{VariableName: "history_chat"},
	prompts.NewHumanMessagePromptTemplate("{{.human_input}}", []string{"human_input"}),
})

var testVoipOptions = func(sseChannel chan string) []RunOptFn {
	return []RunOptFn{
		WithCBSse(CallbackSSEStream),
		WithSSEChan(sseChannel),
		WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(log))),
	}
}

func TestChainVoip_Run(t *testing.T) {
	tools.TestMode()
	sseChannel := make(chan string)
	go readFromChannelAndPrint(sseChannel)
	type args struct {
		ctx       context.Context
		input     string
		hst       []llms.ChatMessage
		optsFuncs []RunOptFn
	}
	tests := []struct {
		name string
		args args
	}{
		{"t2", args{ctx, uq[5], prevMsgs2, testVoipOptions(sseChannel)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			voip := NewChainVoip(tt.args.ctx, llm, log)
			resultMap, err := voip.Run(tt.args.ctx, map[string]any{voip.GetKeyIn(): tt.args.input, PlaceholderForHistory: tt.args.hst}, tt.args.optsFuncs...)

			assert.NoError(t, err)
			assert.NotNil(t, resultMap)

			help_prettyPrintStruct_T(t, resultMap)
		})
	}
}

// readFromChannelAndPrint reads from the given channel and prints each message to stdout
func readFromChannelAndPrint(sseChannel chan string) {
	for chunk := range sseChannel {
		fmt.Printf("Received chunk ===>[%s]\n", chunk)
	}
}
func TestNewChainVoipExt(t *testing.T) {
	ctx := context.Background()
	inputKey := "userPrompt"
	ws := &biz.WSGetter{}
	db := &w.KnowledgeBase{}

	voip := NewChainVoipExt(ctx, llm, log, ws, db)
	t.Log("inputKey =>", voip.GetKeyIn())
	t.Log("outKey =>", voip.outKey)

	assert.NotNil(t, voip)
	assert.Equal(t, log, voip.log)
	assert.Equal(t, inputKey, voip.GetKeyIn())
	assert.Equal(t, ws, voip.ws)
	assert.Equal(t, db, voip.db)
	assert.NotNil(t, voip.chainFirst)
	assert.NotNil(t, voip.agents[escalation_path_ag_2])

	t.Log("InputKey => ", voip.GetKeyIn())
	t.Log("KeyOut => ", voip.GetKeyOut())
	t.Log()
	voip.chainFirst.(*LifecellAgentFirst).LogInfoAboutChain(ctx)
	voip.agents[escalation_path_ag_2].(*LifecellChain).LogInfoAboutChain(ctx)
}

func TestFirstChain(t *testing.T) {
	ctx := AddToCtxUUIDAI(ctx, "a5584496-b231-48b3-9710-f8dfe7c36ee3")
	help_prettyPrintStruct_T(t, ctx)
	voip := NewChainVoip(ctx, llm, log)
	ag1 := voip.chainFirst.(*LifecellAgentFirst)
	ag1.GetBuffer().ReturnMessages = true

	inputs := MapAny{voip.GetKeyIn(): "Яке було моє останнє питання?", PlaceholderForHistory: prevMsgs}

	res, err := chains.Call(ctx, voip.chainFirst.(*LifecellAgentFirst), inputs)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	help_prettyPrintStruct_T(t, res)
}

func TestFirstChain3(t *testing.T) {
	chat, err := ChatHistoryAsString(prevMsgs)
	assert.NoError(t, err)
	t.Logf("\n<< CHAT_HISTORY w/o sys_prompt >>\n%s", chat)
	t.Log(sep)
	promptAg1 := promptTmplAg1(promptFromFS(PROMPT_PATH_AG_1))
	t.Logf("promptWithPlaceHolder.GetInputVariables => %v\n", promptAg1.GetInputVariables())
	t.Run("1", func(t *testing.T) {
		chatMsg, err := promptAg1.FormatMessages(map[string]any{"outputFormat": "json", PlaceholderForHistory: prevMsgs, PromptTmplUserInput: "Some neeeeeeeew Questions?????"})
		t.Error(err)
		t.Logf("\n<< CHAT_HISTORY full >>\n%s", lo.Must(ChatHistoryAsString(chatMsg, true)))
	})
	t.Run("2", func(t *testing.T) {
		chatMsg, _ := promptAg1.FormatMessages(map[string]any{"outputFormat": "json", PlaceholderForHistory: []llms.ChatMessage{}, PromptTmplUserInput: "Some neeeeeeeew Questions?????"})
		t.Logf("\n<< CHAT_HISTORY full >>\n%s", lo.Must(ChatHistoryAsString(chatMsg, true)))
	})
}

func TestCallbackSSEStream(t *testing.T) {
	data := "Ось SQL-запит, який поверне всі типи акаунтів з таблиці `CM_ACCOUNT_TYPE`:\n\n```sql\nSELECT CODE, PARENT_CODE, DESCRIPTION\nFROM CM_ACCOUNT_TYPE;\n```\n\nЦей запит поверне всі записи з таблиці `CM_ACCOUNT_TYPE`, включаючи код, батьківський код та опис кожного типу акаунту."
	sseChannel := make(chan string)

	go func() {
		for chunk := range sseChannel {
			t.Logf("Received chunk: %s", chunk)
		}
	}()
	err := _CallbackSSEStream(ctx, log, data, sseChannel, 10)
	if err != nil {
		t.Errorf("CallbackSSEStream returned an error: %v", err)
	}
}

func TestCallbackSSEStream2(t *testing.T) {
	data := "Ось SQL-запит, який поверне всі типи акаунтів з таблиці `CM_ACCOUNT_TYPE`:\n\n```sql\nSELECT CODE, PARENT_CODE, DESCRIPTION\nFROM CM_ACCOUNT_TYPE;\n```\n\nЦей запит поверне всі записи з таблиці `CM_ACCOUNT_TYPE`, включаючи код, батьківський код та опис кожного типу акаунту."
	for _, v := range strings.Split(data, "\n") {
		t.Logf("CHUNK=[%s]\n", v)
	}
}
