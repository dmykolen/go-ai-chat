package tools

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

type AvailableTool interface {
	tools.Tool
	GetTool() *llms.Tool
	GetCallHandler() ToolCallHandler
}

type AvailableTools []AvailableTool

func NewAvailableTools(tools ...AvailableTool) AvailableTools {
	return tools
}

func (t AvailableTools) GetTools() []llms.Tool {
	tools := make([]llms.Tool, len(t))
	for i, tool := range t {
		tools[i] = *tool.GetTool()
	}
	return tools
}

func (t AvailableTools) GetMapCallHandlers() map[string]ToolCallHandler {
	callHandlers := make(map[string]ToolCallHandler)
	for _, tool := range t {
		callHandlers[tool.Name()] = tool.GetCallHandler()
	}
	return callHandlers
}
