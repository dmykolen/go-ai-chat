package ailogic

import (
	"context"
	_ "embed"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/memory/sqlite3"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
)

const (
	PromptTemplSys1     = DefaultPrefixSystemPrompt + "\n\n## OutputFormat\n{{.outputFormat}}\n\n {{.suffix}}"
	PromptTmplUser      = "{{.userPrompt}}"
	PromptTmplUserInput = "userPrompt"
	defaultName         = "agent_first"
)

func CreateMemoryDefault(ctx context.Context) *memory.ConversationBuffer {
	return memory.NewConversationBuffer(
		memory.WithInputKey("userPrompt"),
		memory.WithOutputKey("llm_output"),
		memory.WithChatHistory(sqlite3.NewSqliteChatMessageHistory(
			sqlite3.WithSession(GetUUIDAI(ctx)), sqlite3.WithDB(db), sqlite3.WithContext(ctx), sqlite3.WithTableName(defaultName)),
		),
	)
}

type LLMChainBuilder struct {
	*chains.LLMChain
}

func NewLLMChainBuilder() *LLMChainBuilder {
	return &LLMChainBuilder{
		LLMChain: &chains.LLMChain{},
	}
}

func (b *LLMChainBuilder) WithLLM(llm llms.Model) *LLMChainBuilder {
	b.LLM = llm
	return b
}

func (b *LLMChainBuilder) WithCH(handler callbacks.Handler) *LLMChainBuilder {
	b.CallbacksHandler = handler
	return b
}

func (b *LLMChainBuilder) WithOP(parser schema.OutputParser[any]) *LLMChainBuilder {
	b.OutputParser = parser
	return b
}

func (b *LLMChainBuilder) WithOK(key string) *LLMChainBuilder {
	b.OutputKey = key
	return b
}

func (b *LLMChainBuilder) WithMem(mem schema.Memory) *LLMChainBuilder {
	b.Memory = mem
	return b
}

func (b *LLMChainBuilder) WithPrompt(prompt prompts.FormatPrompter) *LLMChainBuilder {
	b.Prompt = prompt
	return b
}

func (b *LLMChainBuilder) Build() *chains.LLMChain {
	return b.LLMChain
}
