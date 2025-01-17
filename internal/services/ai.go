package services

import (
	"context"

	"github.com/tmc/langchaingo/llms"
	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/llm"
	"gitlab.dev.ict/golang/libs/goopenai"
)

type AIService struct {
	BaseService
	cfg *config.Config
	llm llms.Model
	ai  *goopenai.Client
}

func NewAIService(cfg *config.Config) *AIService {
	return &AIService{
		BaseService: NewBaseService("ai", cfg.Log),
		cfg:         cfg,
	}
}

func (s *AIService) Initialize(ctx context.Context) error {
	s.llm = llm.OpenAI(s.cfg.LLMModel, s.logger, s.cfg.GetHTTPClient())
	s.ai = goopenai.New().
		WithProxy(true, "").
		WithLogger(s.logger).
		Build()
	return nil
}

func (s *AIService) Stop(ctx context.Context) error {
	return nil
}

func (s *AIService) GetLLM() llms.Model {
	return s.llm
}

func (s *AIService) GetAI() *goopenai.Client {
	return s.ai
}
