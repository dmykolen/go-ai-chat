package agents

import (
	"context"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"gitlab.dev.ict/golang/libs/gologgers"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	lifetools "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
)

type VectorDBSearchTool struct {
	log              *gologgers.Logger
	CallbacksHandler callbacks.Handler
	openAiTool       *llms.Tool
}

var _ tools.Tool = VectorDBSearchTool{}

func NewVectorDBSearchTool(l *gologgers.Logger) *VectorDBSearchTool {
	return &VectorDBSearchTool{
		log:              l,
		CallbacksHandler: callbackhandlers.NewLoggerHandler(l),
		openAiTool: &llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "getRelevantDocsFromVectorDB",
				Description: "API to retrieve documents pieces from Vector DB, that answer customer queries or provide necessary information regarding services and troubleshooting. Documents types: technical documentation for tariffs, products, services; troubleshooting guides, etc.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "User query",
						},
					},
					"required": []string{"query"},
				},
			},
		},
	}
}

func (c VectorDBSearchTool) GetTool() *llms.Tool {
	return c.openAiTool
}

// Description returns a string describing the calculator tool.
func (c VectorDBSearchTool) Description() string {
	return `
	"A wrapper around Lifecell's knowledge base Search. "
	"Lifecell's knowledge base is the Vector DB. "
	"Useful for when you need Lifecell's technical documentations or info about tariffs, products, services or troubleshooting guides, etc. "
	"Always one of the first options when you need to find specific info related to Lifecell"
	"Input should be as JSON object in following format: {"query":"Optomized query for searching in VectorDB"}"`
}

// Name returns the name of the tool.
func (c VectorDBSearchTool) Name() string {
	return c.openAiTool.Function.Name
}

func (c VectorDBSearchTool) Call(ctx context.Context, input string) (string, error) {
	rec := c.log.RecWithCtx(ctx, "vector-tool")
	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolStart(ctx, input)
	}

	// result, err := c.getRelevantDocsFromVectorDB2(ctx, []byte(input))
	result, err := lifetools.GetRelevantDocsFromVectorDB(rec, []byte(input))
	if err != nil {
		rec.Errorf("error from vactrodb: %s", err.Error())
		return "", err
	}

	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	return result, nil
}
