package services

import (
	"context"

	"gitlab.dev.ict/golang/go-ai/config"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
)

type WSService struct {
	BaseService
	cfg      *config.Config
	cimws    *cimws.CimClient
	omws     *omws.Client
	wsGetter *biz.WSGetter
}

func NewWSService(cfg *config.Config) *WSService {
	return &WSService{
		BaseService: NewBaseService("ws", cfg.Log),
		cfg:         cfg,
	}
}

func (s *WSService) Initialize(ctx context.Context) error {
	s.cimws = cimws.NewClient(
		cimws.WithLogger(s.logger.Logger),
		cimws.WithParams(s.cfg.CimwsApiParams.DebugOFF()),
	)

	s.omws = omws.NewClient(
		omws.WithLogger(s.logger),
		omws.WithParams(s.cfg.OmwsApiParams.DebugOFF()),
	)

	s.wsGetter = biz.NewWSGetter(s.logger, s.cimws, s.omws)
	return nil
}

func (s *WSService) Stop(ctx context.Context) error {
	return nil
}

func (s *WSService) GetCIMWS() *cimws.CimClient {
	return s.cimws
}

func (s *WSService) GetOMWS() *omws.Client {
	return s.omws
}

func (s *WSService) GetGetter() *biz.WSGetter {
	return s.wsGetter
}
