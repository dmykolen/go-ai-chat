package ailogic

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"gitlab.dev.ict/golang/libs/utils"
)

var llm4o = lo.Must1(openai.New(openai.WithModel(GPT_4o), openai.WithHTTPClient(HttpCl), openai.WithCallback(callbackH)))

func Test_Json_to_chat(t *testing.T) {
	chat := help_gen_msgChats(t)
	chatMessages := MessageContentToChatMessages(chat)

	t.Log(chatMessages)
	t.Log(llms.GetBufferString(chatMessages, "Human", "AI"))

	promptTempl := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("You are an AI customer support assistant specialized in FMC VoIP", nil),
		prompts.MessagesPlaceholder{VariableName: "history"},
		prompts.NewHumanMessagePromptTemplate("{{.human_input}}", []string{"human_input"}),
	})

	promptValue, e := promptTempl.FormatPrompt(map[string]interface{}{"history": chatMessages, "human_input": "What is my status and what does it mean error 409?"})
	assert.NoError(t, e)
	t.Log(promptValue.Messages())

	t.Log("========================================")
	t.Log(utils.JsonPrettyStr(MessageContentFromChat(promptValue)))
}

func TestCallChainWithJsonAsHistory(t *testing.T) {
	ch := &chains.LLMChain{
		LLM:              llm4o,
		CallbacksHandler: callbackH,
		OutputParser:     outputparser.NewSimple(),
		Memory: memory.NewConversationBuffer(
			memory.WithReturnMessages(true),
			memory.WithMemoryKey("history_chat"),
			memory.WithChatHistory(CreateSqliteMem(ctx, "test_llm_chain")),
		),
		Prompt: prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
			prompts.NewSystemMessagePromptTemplate("You are an AI customer support assistant specialized in FMC VoIP", nil),
			prompts.MessagesPlaceholder{VariableName: "history_chat"},
			prompts.NewHumanMessagePromptTemplate("{{.human_input}}", []string{"human_input"}),
		}),
	}

	r, e := chains.Run(ctx, ch, "What would be a good company name a company that makes colorful socks?")
	assert.NoError(t, e)
	t.Log(r)

}

func help_gen_msgChats(t *testing.T) []llms.MessageContent {
	t.Helper()
	// js := `[{"role":"system","content":[{"type":"text","text":"You are an AI customer support assistant specialized in FMC VoIP"}]},{"role":"human","content":[{"type":"text","text":"У мене проблеми з дзвінками на номері 380933780678"}]}]`
	// js := `[{"role":"human","content":[{"type":"text","text":"У мене проблеми з дзвінками на номері 380933780678"}]}]`
	js := `[{"role":"human","text":"У мене проблеми з дзвінками на номері 380933780678"}]`

	var chat []llms.MessageContent
	// var chat []llms.PromptValue
	e := json.Unmarshal([]byte(js), &chat)
	assert.NoError(t, e)
	// append Tool
	chat = append(chat, llms.TextParts(llms.ChatMessageTypeAI, "Blaaaa blaaaaa ....."))
	chat = append(chat, llms.TextParts(llms.ChatMessageTypeHuman, "What is my status and what does it mean error 409?"))
	chat = append(chat, llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{
		llms.ToolCall{ID: "11212123", Type: string(llms.ChatMessageTypeFunction), FunctionCall: &llms.FunctionCall{Name: "getAccountData", Arguments: "{\"msisdn\":\"380933780678\"}"}},
		llms.ToolCall{ID: "54548945", Type: string(llms.ChatMessageTypeFunction), FunctionCall: &llms.FunctionCall{Name: "getRelevantDocuments", Arguments: "{\"query\":\"error 409\"}"}},
	}})
	chat = append(chat, llms.MessageContent{Role: llms.ChatMessageTypeTool, Parts: []llms.ContentPart{llms.ToolCallResponse{ToolCallID: "11212123", Content: "{\"msisdn\":\"380933780678\", \"status\":\"ACT\\STD\"}"}}})
	t.Log(chat)
	jsb, err := json.MarshalIndent(chat, "", "  ")
	assert.NoError(t, err)
	t.Log(string(jsb))
	return chat
}

