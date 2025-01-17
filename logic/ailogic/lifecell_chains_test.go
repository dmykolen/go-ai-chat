package ailogic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	"gitlab.dev.ict/golang/libs/utils"
)

// testConfig holds common test configuration and dependencies
type testConfig struct {
	ctx   context.Context
	chain *LifecellChain
	input string
}

// setupTest prepares a test environment
func setupTest(t *testing.T) *testConfig {
	t.Helper()
	Help_init_ALL_tools_for_test(t)

	return &testConfig{
		ctx: utils.GenerateCtxWithRid(),
	}
}

// setupChain creates a chain with specified configuration
func setupChain(t *testing.T, name string, promptPath string) *LifecellChain {
	t.Helper()
	return LifecellChainNew(logInfo, llm,
		WithName(name),
		WithTools(tools.ToolFuncs),
		WithPrompt(newPromptFromFS(promptPath)),
		WithOutputParse(NewOutputParserJSONSimple[any]()),
		WithThreshold(7),
		WithMemHist(CreateSqliteMem(ctx, name)),
		WithCallOpts(
			llms.WithModel(GPT_4o),
			llms.WithTools(tools.ToolFuncs),
			llms.WithMaxTokens(700),
			llms.WithTemperature(0.02),
			llms.WithJSONMode(),
		),
	)
}

func TestSolutionSearcherBasicFlow(t *testing.T) {
	cfg := setupTest(t)
	chain := setupChain(t, chainName2, "prompts/agent_tools_6a__3a.txt")

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:  "basic query",
			input: "What does error 409 mean?",
		},
		{
			name:  "tariff plan query",
			input: "який у мене тарифний план 380930164453",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := chains.Call(cfg.ctx, chain, MapAny{chain.inputKey: tt.input})

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, res[chain.OutputKey])

			// Verify response format
			var responseMap map[string]string
			utils.JsonToStructStr(res[defKeyOutLLM].(string), &responseMap)
			require.NoError(t, err)

			// Verify required fields exist
			assert.Contains(t, responseMap, "finalResponse")
			assert.Contains(t, responseMap, CoT)
			assert.Contains(t, responseMap, SC)
		})
	}
}

func TestCallIssueAgent(t *testing.T) {
	cfg := setupTest(t)
	agent := NewAgentCallIssue(cfg.ctx, logTrace)
	input := "У мене проблеми з дзвінками на номері 380933780687"

	t.Run("agent configuration", func(t *testing.T) {
		info := agent.LogInfoAboutChain(ctx)
		assert.NotEmpty(t, info)

		prompt, err := agent.Prompt.FormatPrompt(map[string]any{agent.inputKey: input})
		require.NoError(t, err)
		assert.NotEmpty(t, prompt.String())
	})

	t.Run("chain execution", func(t *testing.T) {
		res, err := chains.Call(cfg.ctx, agent, MapAny{agent.inputKey: input})
		require.NoError(t, err)

		assert.NotNil(t, res[agent.OutputKey])
		assert.IsType(t, "", res[agent.OutputKey])
	})
}
