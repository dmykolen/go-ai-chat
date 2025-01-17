package ailogic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/libs/utils"
)

type testFixture struct {
	ctx     context.Context
	agent   *LifecellAgentFirst
	history schema.ChatMessageHistory
}

func setupTest1(t *testing.T) *testFixture {
	t.Helper()
	ctx := utils.GenerateCtxWithRid()
	agent := Agent1(ctx, logTrace, openaiOptsAg1)
	return &testFixture{
		ctx:     ctx,
		agent:   agent,
		history: CreateSqliteMem(ctx, chainName1),
	}
}

func TestAgentInitialization(t *testing.T) {
	t.Run("basic initialization", func(t *testing.T) {
		fix := setupTest1(t)
		assert.NotNil(t, fix.agent)
		assert.NotNil(t, fix.agent.LLM)
		assert.NotNil(t, fix.agent.Prompt)
	})

	t.Run("with custom history", func(t *testing.T) {
		uuid := "test-uuid"
		ctx := context.WithValue(context.Background(), _ctx_u_cu, uuid)
		hist := CreateSqliteMem(ctx, chainName1)
		agent := Agent1(ctx, logTrace, openaiOptsAg1, hist)
		assert.NotNil(t, agent)
		assert.Equal(t, hist, agent.chatHistory())
	})
}

func TestAgentQueries(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		setup   func(*testFixture)
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "balance query",
			input: "Який у мене баланс?",
			want:  map[string]string{"response": "", "IsValidQuery": "true"},
		},
		{
			name:  "tariff query with history",
			input: "дай мені VOIP налаштування",
			setup: func(f *testFixture) {
				f.history.AddUserMessage(f.ctx, "який у мене тариф 380930164453")
				f.history.AddAIMessage(f.ctx, "Ваш тарифний план - Єдина Мережа Преміум 30")
			},
			want: map[string]string{"response": "", "nextAgent": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := setupTest1(t)
			if tt.setup != nil {
				tt.setup(fix)
			}

			result, err := chains.Call(fix.ctx, fix.agent, MapAny{
				PromptTmplUserInput:   tt.input,
				PlaceholderForHistory: []llms.ChatMessage{},
			}, chainOpts...)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result[fix.agent.OutputKey])

			var response map[string]string
			utils.JsonToStructStr(result[defKeyOutLLM].(string), &response)

			for k, v := range tt.want {
				assert.Contains(t, response, k)
				if v != "" {
					assert.Equal(t, v, response[k])
				}
			}
		})
	}
}

func TestAgentMemoryPersistence(t *testing.T) {
	fix := setupTest1(t)
	messages := []struct {
		input  string
		aiResp string
	}{
		{
			input:  uq[4],
			aiResp: airesp[4],
		},
		{
			input:  "Який тариф на номері 380632107489?",
			aiResp: "Тариф на номері 380632107489 - Лайт 2021",
		},
	}

	for _, msg := range messages {
		fix.history.AddUserMessage(fix.ctx, msg.input)
		fix.history.AddAIMessage(fix.ctx, msg.aiResp)
	}

	stored, err := fix.history.Messages(fix.ctx)
	require.NoError(t, err)
	assert.Len(t, stored, len(messages)*2)
}

func TestAgentErrorHandling(t *testing.T) {
	fix := setupTest1(t)

	tests := []struct {
		name    string
		input   string
		mutate  func(*LifecellAgentFirst)
		wantErr string
	}{
		{
			name:    "empty input",
			input:   "",
			wantErr: "empty input",
		},
		{
			name:  "nil LLM",
			input: "test",
			mutate: func(a *LifecellAgentFirst) {
				a.LLM = nil
			},
			wantErr: "LLM not initialized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mutate != nil {
				tt.mutate(fix.agent)
			}

			_, err := chains.Call(fix.ctx, fix.agent, MapAny{
				PromptTmplUserInput: tt.input,
			})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
