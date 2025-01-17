package callbackhandlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

type HandlerExt interface {
	callbacks.Handler
	HandlePrettyAsJS(ctx context.Context, msg string, data interface{})
}

// LoggerHandler is a callback handler that prints to the standard output.
type LoggerHandler struct {
	log *gl.Logger
}

func (l *LoggerHandler) Log() *gl.Logger {
	return l.log
}

func NewLoggerHandler(log *gl.Logger) LoggerHandler {
	return LoggerHandler{log: log}
}

// HandleCustom implements HandlerExt.
func (l LoggerHandler) HandlePrettyAsJS(ctx context.Context, msg string, data interface{}) {
	l.rec(ctx).Infof(msg, utils.JsonPretty(data))
}

var _ callbacks.Handler = LoggerHandler{}
var _ HandlerExt = LoggerHandler{}

func DefaultLoggerCallback() LoggerHandler {
	return LoggerHandler{log: gl.New(gl.WithLevel("debug"), gl.WithColor(), gl.WithOC())}

}

func (l LoggerHandler) rec(ctx context.Context) *gl.LogRec {
	return l.log.RecWithCtx(ctx)
}

func (l LoggerHandler) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	strBuf := strings.Builder{}

	if l.log.Options.LogLevel == "debug" || l.log.Options.LogLevel == "trace" {
		llms.ShowMessageContents(&strBuf, ms)
		l.rec(ctx).Debugf(">>> Entering LLM with messages[len=%d]:\n%s", len(ms), strBuf.String())
	} else {
		if ms[0].Role == "system" {
			ms = ms[1:]
		}
		llms.ShowMessageContents(&strBuf, ms)
		l.rec(ctx).Infof(">>> Entering LLM with messages[len=%d]. Show messages W/O SYS_PROMPT:\n%s", len(ms), strBuf.String())
	}
}

func (l LoggerHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	for idx, c := range res.Choices {
		l.rec(ctx).Infof("AI respond! Choice[%d]: tokens=[request:%d, response:%d] sr=%s responseContent=[%s]", idx, c.GenerationInfo["PromptTokens"], c.GenerationInfo["CompletionTokens"], c.StopReason, c.Content)
		if c.FuncCall != nil {
			l.rec(ctx).Info("AI FuncCall: ", c.FuncCall.Name, c.FuncCall.Arguments)
		}
	}
}

func (l LoggerHandler) HandleStreamingFunc(_ context.Context, chunk []byte) {
	fmt.Printf("%s", string(chunk))
}

func (l LoggerHandler) HandleText(ctx context.Context, text string) {
	l.rec(ctx).Info(">>> handle text:", text)
}

func (l LoggerHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	l.rec(ctx).Info("Entering LLM with prompts:", prompts)
}

func (l LoggerHandler) HandleLLMError(ctx context.Context, err error) {
	l.rec(ctx).Info("Exiting LLM with error:", err)
}

func (l LoggerHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	l.rec(ctx).Info("Entering chain with inputs:", formatChainValues(inputs))
}

func (l LoggerHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	if l.log.Options.LogLevel == "debug" || l.log.Options.LogLevel == "info" {
		if v, ok := outputs["full_history"].([]llms.MessageContent); ok {
			outputs["full_history"] = len(v)
		}
	}
	l.rec(ctx).Info("Exiting chain with outputs:", utils.JsonPrettyStr(outputs))
}

func (l LoggerHandler) HandleChainError(ctx context.Context, err error) {
	l.rec(ctx).Info("Exiting chain with error:", err)
}

func (l LoggerHandler) HandleToolStart(ctx context.Context, input string) {
	l.rec(ctx).Info("Entering tool with input:", removeNewLines(input))
}

func (l LoggerHandler) HandleToolEnd(ctx context.Context, output string) {
	l.rec(ctx).Info("Exiting tool with output:", removeNewLines(output))
}

func (l LoggerHandler) HandleToolError(ctx context.Context, err error) {
	l.rec(ctx).Info("Exiting tool with error:", err)
}

func (l LoggerHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	l.rec(ctx).Info("Agent selected action:", formatAgentAction(action))
}

func (l LoggerHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	fmt.Printf("Agent finish: %v \n", finish)
}

func (l LoggerHandler) HandleRetrieverStart(ctx context.Context, query string) {
	l.rec(ctx).Info("Entering retriever with query:", removeNewLines(query))
}

func (l LoggerHandler) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	l.rec(ctx).Info("Exiting retriever with documents for query:", documents, query)
}

func formatChainValues(values map[string]any) string {
	output := ""
	for key, value := range values {
		output += fmt.Sprintf("\"%s\" : \"%s\", ", removeNewLines(key), removeNewLines(value))
	}

	return output
}

func formatAgentAction(action schema.AgentAction) string {
	return fmt.Sprintf("\"%s\" with input \"%s\"", removeNewLines(action.Tool), removeNewLines(action.ToolInput))
}

func removeNewLines(s any) string {
	return strings.ReplaceAll(fmt.Sprint(s), "\n", " ")
}
