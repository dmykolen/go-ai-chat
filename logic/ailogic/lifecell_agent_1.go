package ailogic

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/libs/gologgers"
	"golang.org/x/exp/maps"
)

const (
	PlaceholderForHistory = "history_chat"
	PROMPT_PATH_AG_1      = "prompts/sys_first_3_a.txt"
	PROMPT_PATH_AG_1_4    = "prompts/sys_first_4.txt"
)

var (
	// Note: The `Sys_first_3` prompt template includes a hidden input variable `outputFormat`.
	// If specified here, it must also be provided in the `inputs` when calling `Call` or `Execute`, which is inconvenient as it is auto-determined.
	// `Sys_first_3` refers to the content of the file `prompts/sys_first_3.txt`.
	promptTmplAg1 = func(systemPrompt string) prompts.ChatPromptTemplate {
		return prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
			prompts.NewSystemMessagePromptTemplate(systemPrompt, []string{}),
			prompts.MessagesPlaceholder{VariableName: PlaceholderForHistory},
			prompts.NewHumanMessagePromptTemplate(PromptTmplUser, []string{PromptTmplUserInput}),
		})
	}
	openaiOptsAg1 = []openai.Option{
		openai.WithModel(GPT_4o),
		openai.WithHTTPClient(HttpCl),
		openai.WithResponseFormat(responseFormatSchemaAg1),
	}
)

type LifecellAgentFirst struct {
	*LifecellChain
}

var (
	_ chains.Chain           = &LifecellAgentFirst{}
	_ callbacks.HandlerHaver = &LifecellAgentFirst{}
)

func promptFromFS(fileName string) string {
	content, err := promptFS.ReadFile(fileName)
	if err != nil {
		panic(fmt.Errorf("failed to read prompt file %s: %w", fileName, err))
	}

	return string(content)
}

func Agent1(ctx context.Context, l *gologgers.Logger, openaiOpts []openai.Option, chatHst ...schema.ChatMessageHistory) *LifecellAgentFirst {
	llm, err := openai.New(openaiOpts...)
	if err != nil {
		panic(err)
	}
	return &LifecellAgentFirst{
		LifecellChain: LifecellChainNew(
			l, llm,
			WithName(chainName1),
			WithPrompt(promptTmplAg1(promptFromFS(PROMPT_PATH_AG_1_4))),
			WithOutputParse(NewOutputParserJSONSimple[any]()),
			WithMemHist(
				lo.TernaryF(chatHst == nil,
					func() schema.ChatMessageHistory { return CreateSqliteMem(ctx, chainName1) },
					func() schema.ChatMessageHistory { return chatHst[0] },
				)),
			WithCallOpts(defCallOpts...),
		),
	}
}

func (c *LifecellAgentFirst) Execute(ctx context.Context, inputs map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
	panic("not implemented")
}

func (c *LifecellAgentFirst) Call(ctx context.Context, inputs map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
	rec := c.log.RecWithCtx(ctx, c.ChainName)
	rec.Infof("Prompt expected variables: %v; Provided values: %#v", c.Prompt.GetInputVariables(), maps.Keys(inputs))
	msgs, err := c.chatHistory().Messages(ctx)
	rec.WithError(err).Infof("Currently memory size=%d", len(msgs))

	var promptValue llms.PromptValue
	if _, ok := inputs[OF]; !ok || inputs[OF] == "" {
		inputs[OF] = c.OutputParser.GetFormatInstructions()
	}
	if promptValue, err = c.Prompt.FormatPrompt(inputs); err != nil {
		return nil, err
	}

	c.chatHistory().AddMessage(ctx, promptValue.Messages()[0])
	rec.Infof("chat messages before send to AI full_size(with sysprompt)=%d\n [W/O sys_msg] =>\n%s", len(promptValue.Messages()), ChatHistoryAsStringSafe(promptValue.Messages()))

	result, err := c.LLM.GenerateContent(ctx, MessageContentFromChat(promptValue), append(c.callOpts, llms.WithModel(GPT_4o))...)

	/* <<<<<< FOR TEST >>>>>>*/
	// result, err := &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: respContent}}}, nil
	/* <<<<<<<<<<<>>>>>>>>>>>*/
	if err != nil {
		return nil, err
	}

	finalOutput, err := c.OutputParser.ParseWithPrompt(result.Choices[0].Content, promptValue)
	if err != nil {
		return nil, err
	}

	mapOfResJson, ok := finalOutput.(map[string]any)
	if !ok {
		return nil, err
	}
	mapOfResJson[c.OutputKey] = result.Choices[0].Content
	return mapOfResJson, nil
}
