package ailogic

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"fmt"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"golang.org/x/exp/maps"
)

const (
	defInputKey  = PromptTmplUserInput
	defOutKey    = "output"
	defKeyOutLLM = "output_llm"
	OF           = "outputFormat" // the key used to specify the output format

	CoT = "chainOfThoughts"
	SC  = "selfCheck"

	PROMPT_FILE_TC_3          = "prompts/sys_agent_toolcaller_3.txt"
	PROMPT_FILE_TC            = "prompts/agent_tools_caller.txt"
	PROMPT_FILE_TC_4          = "prompts/agent_tools_4.txt"
	PROMPT_FILE_TC_6a         = "prompts/agent_tools_6a.txt"
	PROMPT_FILE_TC_6a__3      = "prompts/agent_tools_6a__3.txt"
	PROMPT_FILE_TC_6a__3a     = "prompts/agent_tools_6a__3a.txt"
	PROMPT_FILE_AG_CALL_ISSUE = "prompts/ag_call_issue.txt"
)

var (
	//go:embed prompts/*
	promptFS embed.FS

	_ chains.Chain           = &LifecellChain{}
	_ callbacks.HandlerHaver = &LifecellChain{}
)

type LifecellChain struct {
	*chains.LLMChain
	ChainName      string
	log            *gologgers.Logger
	tools          []llms.Tool
	toolsAvailable map[string]tools.ToolCallHandler
	inputKey       string
	optsChain      []chains.ChainCallOption
	tresholdTools  int
	callOpts       []llms.CallOption
}

func LifecellChainNew(l *gologgers.Logger, llm llms.Model, opts ...OptFn) *LifecellChain {
	opt := DefaultOptions()
	for _, o := range opts {
		o(&opt)
	}
	if opt.name == "" {
		panic("chain name is required")
	}
	if l != nil {
		opt.callbacks = callbackhandlers.NewLoggerHandler(l)
	}
	memCreate(&opt)
	return &LifecellChain{
		LLMChain:      NewLLMChainBuilder().WithLLM(llm).WithCH(opt.callbacks).WithOP(opt.outParser).WithOK(opt.outKey).WithMem(opt.mem).WithPrompt(opt.prompt).Build(),
		log:           l,
		ChainName:     opt.name,
		tools:         opt.tools,
		inputKey:      opt.inputKey,
		tresholdTools: opt.thresholdTools,
		callOpts:      opt.optsCall,
	}
}

func LifecellAgentSolutionSearcher(ctx context.Context, l *gologgers.Logger, llm llms.Model, opts ...OptFn) *LifecellChain {
	return LifecellChainNew(l, llm,
		WithName(chainName2),
		WithTools(tools.ToolFuncs),
		WithPrompt(newPromptFromFS(PROMPT_FILE_TC_6a__3a)),
		WithOutputParse(NewOutputParserJSONSimple[any]()),
		WithThreshold(6),
		// WithMem(CreateMemoryDefault(ctx)),
		WithMemHist(CreateSqliteMem(ctx, chainName2)),
		WithCallOpts(llms.WithModel(GPT_4o), llms.WithTools(tools.ToolFuncs), llms.WithMaxTokens(700), llms.WithTemperature(0.08), llms.WithJSONMode()), //llms.WithJSONMode()))
	)
}

func NewAgentCallIssue(ctx context.Context, l *gologgers.Logger) *LifecellChain {
	llm, err := openai.New(openai.WithModel(GPT_4o), openai.WithHTTPClient(HttpCl), openai.WithCallback(callbackhandlers.NewLoggerHandler(l)), openai.WithResponseFormat(responseFormatCallIssue))
	if err != nil {
		panic(err)
	}
	return LifecellChainNew(l, llm,
		WithName(chainNameCI),
		WithTools(tools.ToolFuncs),
		WithPrompt(newPromptFromFS(PROMPT_FILE_AG_CALL_ISSUE)),
		WithOutputParse(NewOutputParserJSONSimple[any]()),
		WithThreshold(6),
		WithMemHist(CreateSqliteMem(ctx, chainNameCI)),
		WithCallOpts(llms.WithModel(GPT_4o), llms.WithTools(tools.ToolFuncs), llms.WithMaxTokens(700), llms.WithTemperature(0.00)),
	)
}

func (c *LifecellChain) Call(ctx context.Context, inputs map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
	rec := c.log.RecWithCtx(ctx, c.ChainName)
	rec.Infof("Start chain '%s'! memKey=%s output:[%v] expected_input:[%v]; Provided variables: %#v", c.ChainName, c.GetMemory().GetMemoryKey(ctx), c.GetOutputKeys(), c.Prompt.GetInputVariables(), maps.Keys(inputs))

	var promptValue llms.PromptValue
	if promptValue, err = c.Prompt.FormatPrompt(inputs); err != nil {
		rec.Errorf("incorrect inputs!", err)
		return nil, err
	}

	messages := MessageContentFromChat(promptValue)
	for toolCallCount := 0; toolCallCount < c.tresholdTools; toolCallCount++ {
		rec.Info("Call LLM GenerateContent...")
		resp, err := c.LLM.GenerateContent(ctx, messages, c.callOpts...)
		if err != nil {
			rec.Errorf("LLM GenerateContent error: %v", err)
			return nil, err
		}

		messages = updateMessageHistory(ctx, messages, nil, resp)
		if resp.Choices[0].StopReason != "tool_calls" {
			break
		}

		rec.Infof("Tool counter = %d. Treshold=%d", toolCallCount, c.tresholdTools)
		messages, err = c.executeToolCalls(rec, messages, resp, false)
		if err != nil {
			rec.Errorf("executeToolCalls error: %v", err)
			return nil, err
		}
	}

	finalOutput, err := c.OutputParser.ParseWithPrompt(messages[len(messages)-1].Parts[0].(llms.TextContent).Text, promptValue)
	if err != nil {
		rec.Errorf("OutputParser.ParseWithPrompt error: %v ORIGINAL_MESSAGE=[%v]", err, messages[len(messages)-1].Parts[0].(llms.TextContent).Text)

		// var mapRes MapAny
		// finalOutput = utils.JsonToStructStr(messages[len(messages)-1].Parts[0].(llms.TextContent).Text, &mapRes)
		// rec.Infof("Try parse in another way. mapRes1=%v \n-------\n\tfinalOutput=%v", mapRes, finalOutput)
		finalOutput = MapAny{FinalResponse: messages[len(messages)-1].Parts[0].(llms.TextContent).Text}
	}
	rec.Infof("finalOutput TYPE=[%T] IsNil=%t", finalOutput, finalOutput == nil)

	mapOfResJson, ok := finalOutput.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("finalOutput is not a map[string]any")
	}
	// mapOfResJson[c.OutputKey] = getString(mapOfResJson, FinalResponse)
	mapOfResJson[c.OutputKey] = getStringIf(mapOfResJson, FinalResponse, ClarifyQuestions)
	rec.Infof("outputKey-name=%s value=%s", c.OutputKey, mapOfResJson[c.OutputKey])
	mapOfResJson[c.OutputKey+"_llm"] = messages[len(messages)-1].Parts[0].(llms.TextContent).Text
	mapOfResJson["full_history"] = messages
	return mapOfResJson, nil
}

func (c *LifecellChain) ChatHistory(ctx context.Context, withSystem ...bool) []llms.ChatMessage {
	hst, err := c.chatHistory().Messages(ctx)
	if err != nil || len(hst) == 0 {
		return []llms.ChatMessage{}
	}
	if hst[0].GetType() == llms.ChatMessageTypeSystem && !utils.FirstOrDefault(false, withSystem...) {
		return hst[1:]
	}
	return hst
}

func (c *LifecellChain) chatHistory() schema.ChatMessageHistory {
	return c.GetMemory().(*memory.ConversationBuffer).ChatHistory
}

func (c *LifecellChain) executeToolCalls(rec *gologgers.LogRec, messageHistory []llms.MessageContent, resp *llms.ContentResponse, writeHst bool) ([]llms.MessageContent, error) {
	return tools.ExecuteToolCalls(rec, tools.AvailableVoipTools, messageHistory, resp, writeHst, c.chatHistory())
}

func (c *LifecellChain) GetBuffer() *memory.ConversationBuffer {
	return c.GetMemory().(*memory.ConversationBuffer)
}

func (c *LifecellChain) LogInfoAboutChain(ctx context.Context) string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("ChainName: %s\n", c.ChainName))
	buffer.WriteString(fmt.Sprintf("Prompt expected variables: %v\n", c.Prompt.GetInputVariables()))
	buffer.WriteString(fmt.Sprintf("OutputKey: %s\n", c.OutputKey))
	buffer.WriteString(fmt.Sprintf("MemoryKey: %#v\n", c.Memory.GetMemoryKey(ctx)))
	buffer.WriteString(fmt.Sprintf("GetInputKeys: %#v\n", c.GetInputKeys()))
	buffer.WriteString(fmt.Sprintf("GetOutputKeys: %#v\n", c.GetOutputKeys()))

	c.log.Infof("\n%s\n%s\n%s\n", sep, buffer.String(), sep)
	return buffer.String()
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
	return append(messageHistory, assistantResponse)
}

// NewPromptFromFS creates a ChatPromptTemplate from an embedded file
func newPromptFromFS(fileName string) prompts.ChatPromptTemplate {
	// Read system prompt from embedded file
	content, err := promptFS.ReadFile(fileName)
	if err != nil {
		panic(fmt.Errorf("failed to read prompt file %s: %w", fileName, err))
	}

	// Create and return chat prompt template
	return prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate(string(content), nil),
		prompts.NewHumanMessagePromptTemplate(PromptTmplUser, []string{PromptTmplUserInput}),
	})
}
