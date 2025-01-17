package services

import (
	"context"

	"gitlab.dev.ict/golang/go-ai/config"
	wv "gitlab.dev.ict/golang/go-ai/services/weaviate"
)

type KBService struct {
	BaseService
	cfg *config.Config
	kb  *wv.KnowledgeBase
}

func NewKBService(cfg *config.Config) *KBService {
	return &KBService{
		BaseService: NewBaseService("kb", cfg.Log),
		cfg:         cfg,
	}
}

func (s *KBService) Start(ctx context.Context) error {
	s.kb = wv.NewKnowledgeBase(
		wv.NewWVClient(s.cfg.WvCfg),
		s.logger,
		wv.DefaultClassKB,
		"assets/knowledge_base.json",
	)
	return nil
}

func (s *KBService) Stop(ctx context.Context) error {
	return nil
}

func (s *KBService) GetKB() *wv.KnowledgeBase {
	return s.kb
}
