package services

import (
	"context"
	"path/filepath"

	"github.com/samber/lo"
	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/services"
	"gitlab.dev.ict/golang/libs/gonet"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	DefPathToDocx = "./assets/voip_ritm_docs"
)

type RAGService struct {
	BaseService
	cfg *config.Config
	ai  *AIService
	kb  *KBService
	ws  *WSService
	rag *services.RAGService
}

func NewRAGService(cfg *config.Config, ai *AIService, kb *KBService, ws *WSService) *RAGService {
	return &RAGService{
		BaseService: NewBaseService("rag", cfg.Log),
		cfg:         cfg,
		ai:          ai,
		kb:          kb,
		ws:          ws,
	}
}

func (s *RAGService) Initialize(ctx context.Context, confPID, pathToDocx string) {
	processorConfluence := services.NewConfluenceProcessor(s.cfg.Log, true).
		WithRun("", "", services.URL_CE, 10, s.cfg.IsDebug).
		WithPageID(confPID).
		Debug(s.cfg.IsDebug).
		EBT(services.ExpBV)

	processorScrapperWebLifecellUA := services.NewWPPLifecellUA(s.cfg.Log).
		WithWebURLs(services.Links_LifecellUA)

	processorScrapperWebOther := services.NewWPPOther(s.cfg.Log)

	if pathToDocx == "" {
		pathToDocx = DefPathToDocx
	}

	docxFiles := lo.Must(filepath.Glob(filepath.Join(utils.ExpandPath(pathToDocx), "*.docx")))
	processorDocx := services.NewDocxPprocessor(s.cfg.Log, gonet.NewRestyClient(s.cfg.Log)).
		WithFilePaths(docxFiles...)

	s.rag = services.NewRAGService(s.logger, s.kb.kb, s.ai.ai).
		SetWS(s.ws.wsGetter).
		LLM(s.ai.llm).
		DBCim(s.cfg.DbTmCim).
		AppendLogic(processorConfluence).
		AppendLogic(processorDocx).
		AppendLogic(processorScrapperWebLifecellUA).
		AppendLogic(processorScrapperWebOther)

}

func (s *RAGService) Start(ctx context.Context) error {
	return nil
}

func (s *RAGService) Stop(ctx context.Context) error {
	return nil
}

func (s *RAGService) GetKB() *KBService {
	return s.kb
}
