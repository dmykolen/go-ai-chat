package config

import (
	_ "embed"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/artyom/autoflags"
	"github.com/caarlos0/env/v6"
	"github.com/gookit/goutil/structs"
	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
	"gitlab.dev.ict/golang/go-ai/db"
	us "gitlab.dev.ict/golang/go-ai/db/user_storage"
	"gitlab.dev.ict/golang/go-ai/db/user_storage/repos"
	"gitlab.dev.ict/golang/go-ai/handlers"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/llm"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	dbi "gitlab.dev.ict/golang/go-ai/logic/db_info_collector"
	"gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/go-ai/services"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	ad "gitlab.dev.ict/golang/libs/goldap"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gonet"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
	"gorm.io/gorm"
)

const (
	DefPathToDocx = "./assets/voip_ritm_docs"
)

type Config struct {
	Title             string                `json:"title" default:"Lifecell-AI" flag:"tit,app title"`
	IsDev             bool                  `json:"isDev" default:"false" env:"GO_AI_IS_DEV" flag:"dev,dev mode"`
	Port              string                `json:"port" default:"5555" env:"GO_AI_PORT" flag:"p,app port"`
	IsDebug           bool                  `json:"isDebug" default:"false" env:"GO_AI_IS_DEBUG" flag:"d,debug mode"`
	WithSSL           bool                  `json:"withSsl" default:"false" env:"GO_AI_WITH_SSL" flag:"app-ssl-on,enable HTTPS mode with self-signed SSL certificates"`
	DirAppSSL         string                `json:"dirAppSSL" default:"./data" env:"GO_AI_SSL_DIR" flag:"dir-app-ssl,dir for SSL certificates"`
	IsLocalFE         bool                  `json:"isLocalFe" flag:"localfe"`
	ConfRootPageID    string                `json:"confRootPageId" default:"75104258"`
	PathPermissions   string                `json:"pathPermissions" default:"assets/permissions.yml"`
	Layout            string                `json:"layout" default:"layouts/main" env:"GO_AI_LT" flag:"lt,layout template"`
	EnvFile           string                `json:"envFile" default:"dev" flag:"env-file,environment file (local,dev,prod)"`
	HttpClientTO      int                   `json:"httpClientTo" env:"HTTP_CLIENT_TO" envDefault:"300" flag:"hcto,HTTP client timeout"`
	IsProxyOFF        bool                  `json:"proxyOff" env:"IS_PROXY_OFF" flag:"prxoff,disable HTTP proxy"`
	ISV               bool                  `json:"isv" flag:"isv,skip server certificate verification"`
	SslPath           string                `json:"sslPath" default:"/etc/ssl/certs/ca-certificates.crt" env:"GO_AI_SSL_PATH" flag:"ssl,SSL cert path"`
	LLMModel          string                `json:"llmModel" default:"gpt-4" env:"GO_AI_DEF_LLM_MODEL" flag:"lm,default LLM model"`
	IsNeedPopulateVDB bool                  `json:"isNeedPopulateVDB" default:"false" flag:"vdb,initiate vector database population"`
	DBTmCimURL        string                `json:"dbURLTmCim" default:"" env:"DB_URL_TM_CIM" flag:"db-tm-cim,Oracle DB URL"`
	IsAiDbOFF         bool                  `json:"isAiDbOff" env:"IS_AI_DB_OFF" flag:"aidboff,disable AI DB"`
	IsForceInitRoles  bool                  `json:"isForceInitRoles" env:"IS_FORCE_INIT_ROLES" flag:"force-init-roles,force roles init"`
	LogOptions        *gologgers.LogOptions `json:"logOptions"`
	UserDbProps       *db.DBProperties      `json:"userDbProps"`
	AdClient          *ad.LDAPConf          `json:"adClient"`
	HandlerApp        *handlers.AppHandler
	WvCfg             *w.WeaviateCfg    `json:"wvCfg"`
	CimwsApiParams    *cimws.ApiParamas `json:"cimwsApiParams"`
	OmwsApiParams     *omws.ApiParamas  `json:"omwsApiParams"`

	// Runtime components
	Log                   *gologgers.Logger
	LLM                   llms.Model
	AI                    *goopenai.Client
	DbTmCim               dbi.DBSchemaInfoProvider
	SqliteChatsDB         *gorm.DB
	Cimws                 *cimws.CimClient
	Omws                  *omws.Client
	WsGetter              *biz.WSGetter
	Confluence            *services.ConfluenceProcessor // Processor for Confluence documents.
	docx                  *services.DocxPprocessor      // Processor for DOCX files.
	scrapperWebLifecellUA *services.WebPagesProcessor   // Web pages processor for Lifecell UA.
	scrapperWebOther      *services.WebPagesProcessor   // Web pages processor for other sources.
	Kb                    *w.KnowledgeBase
	Rag                   *services.RAGService
	UserStorage           us.UserStorage
	PermissionsConfig     *models.PermissionsConfig
	once                  sync.Once
	httpClient            *http.Client
}

func New() *Config {
	return &Config{
		LogOptions:     &gologgers.LogOptions{},
		UserDbProps:    &db.DBProperties{},
		AdClient:       &ad.LDAPConf{InsecSkipVerify: true},
		WvCfg:          &w.WeaviateCfg{},
		CimwsApiParams: &cimws.ApiParamas{IsDebug: false},
		OmwsApiParams:  &omws.ApiParamas{},
		IsAiDbOFF:      true,
	}
}

func (c *Config) Load() error {
	structs.InitDefaults(c)
	structs.InitDefaults(c.LogOptions)
	structs.InitDefaults(c.WvCfg)

	if err := env.Parse(c); err != nil {
		return fmt.Errorf("parsing env vars: %w", err)
	}

	// Parse all other configurations
	env.Parse(c.UserDbProps)
	env.Parse(c.AdClient)
	env.Parse(c.WvCfg)
	env.Parse(c.CimwsApiParams)
	env.Parse(c.OmwsApiParams)

	autoflags.Define(c)
	autoflags.Define(c.LogOptions)
	autoflags.Define(c.CimwsApiParams)
	autoflags.Define(c.OmwsApiParams)
	autoflags.Define(c.WvCfg)
	flag.Parse()

	return c.validate()
}

func (c *Config) validate() error {
	if c.Port == "" {
		return fmt.Errorf("port must be specified")
	}

	if c.WithSSL && c.DirAppSSL == "" {
		return fmt.Errorf("SSL directory must be specified when SSL is enabled")
	}

	return nil
}

func (c *Config) GetHTTPClient() *http.Client {
	c.once.Do(func() {
		c.httpClient = helpers.HttpClient(c.Log, c.HttpClientTO, c.SslPath, c.ISV, !c.IsProxyOFF)
	})
	return c.httpClient
}

func (c *Config) GetUploadDir() string {
	return filepath.Join(".", "uploads")
}

func (c *Config) GetTemplatesDir() string {
	return filepath.Join(".", "web", "views")
}

func (c *Config) GetStaticDir() string {
	return filepath.Join(".", "web", "static")
}

func (c *Config) IsProduction() bool {
	return !c.IsDev
}

func (c *Config) initialize() error {
	// Initialize AI and LLM services
	c.LLM = llm.OpenAI(c.LLMModel, c.Log, c.GetHTTPClient())
	c.AI = goopenai.New().
		WithProxy(true, "").
		WithLogger(gologgers.New(
			gologgers.WithChannel("AI"),
			gologgers.WithLevel(c.LogOptions.LogLevel),
			gologgers.WithOC(c.LogOptions.IsOutConsole),
		)).Build()

	// Initialize database connections
	if !c.IsAiDbOFF {
		c.DbTmCim = dbi.NewDBOracleGorm(
			dbi.DSN(c.DBTmCimURL),
			dbi.Logger(c.Log),
			dbi.DictsInclude(),
			dbi.DictsFilters(
				dbi.ScopeDictTablesFilterByRegExp(dbi.RegExpDict_CustomerModel),
				dbi.ScopeDictTablesFilterByNumRows(100),
				dbi.ScopeExcludeBkpTmp,
			),
		)
	}

	// Initialize chat storage
	c.SqliteChatsDB = ailogic.GetSqliteDB()

	// Initialize web service clients
	c.Cimws = cimws.NewClient(
		cimws.WithLogger(c.Log.Logger),
		cimws.WithParams(c.CimwsApiParams.DebugOFF().URL(utils.GetBaseURL(c.CimwsApiParams.Url))),
	)
	c.Omws = omws.NewClient(
		omws.WithLogger(c.Log),
		omws.WithParams(c.OmwsApiParams.DebugOFF().URL(utils.GetBaseURL(c.OmwsApiParams.Url))),
	)
	c.WsGetter = biz.NewWSGetter(c.Log, c.Cimws, c.Omws)

	// Initialize vector database and knowledge base
	c.WvCfg.Log = c.Log
	c.Kb = w.NewKnowledgeBase(
		w.NewWVClient(c.WvCfg),
		gologgers.New(
			gologgers.WithChannel("VECTOR"),
			gologgers.WithLevel(c.LogOptions.LogLevel),
			gologgers.WithOC(c.LogOptions.IsOutConsole),
		),
		w.DefaultClassKB,
		"assets/knowledge_base.json",
	)

	// Initialize content processors
	processorConfluence := services.NewConfluenceProcessor(c.Log, true).
		WithRun("", "", services.URL_CE, 10, c.IsDebug).
		WithPageID(c.ConfRootPageID).
		Debug(c.IsDebug).
		EBT(services.ExpBV)

	processorScrapperWebLifecellUA := services.NewWPPLifecellUA(c.Log).
		WithWebURLs(services.Links_LifecellUA)

	processorScrapperWebOther := services.NewWPPOther(c.Log)
	processorDocx := services.NewDocxPprocessor(c.Log, gonet.NewRestyClient(c.Log)).
		WithFilePaths(lo.Must(filepath.Glob(filepath.Join(utils.ExpandPath(DefPathToDocx), "*.docx")))...)

	// Initialize RAG service
	c.Rag = services.NewRAGService(c.Log, c.Kb, c.AI).
		SetWS(c.WsGetter).
		LLM(c.LLM).
		DBCim(c.DbTmCim).
		AppendLogic(processorConfluence).
		AppendLogic(processorDocx).
		AppendLogic(processorScrapperWebLifecellUA).
		AppendLogic(processorScrapperWebOther)

	// Initialize user storage and authentication
	c.UserStorage = lo.Must(repos.NewUserStoragePG(
		c.UserDbProps,
		c.Log,
		repos.WithDebug(c.IsDebug),
		repos.WithForceInitRoles(c.IsForceInitRoles),
	))

	c.AdClient.Init(
		ad.WithLogger(c.Log),
		ad.WithDebug(c.IsDebug),
	)

	// Initialize application handler
	c.HandlerApp = handlers.NewAppHandler(
		handlers.WithLogger(c.Log),
		handlers.WithAI(c.AI),
		handlers.WithRAG(c.Rag),
		handlers.WithPGDB(c.UserStorage),
		handlers.WithAD(c.AdClient),
	)

	// Load permissions configuration
	var err error
	c.PermissionsConfig, err = models.LoadPermissionsFromBytes(lo.Must(os.ReadFile(c.PathPermissions)))
	if err != nil {
		return fmt.Errorf("failed to load permissions config: %w", err)
	}

	c.Log.Debugf("--- PermissionsConfig:\n%s", c.PermissionsConfig.String())
	c.Log.Infof("Total items in vector db => %d", c.Kb.TotalItems())

	return nil
}
