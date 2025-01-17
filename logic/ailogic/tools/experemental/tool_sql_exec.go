package agents

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/gorm"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	lifetools "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
)

const sqlRes = `[{"ACCOUNT_CODE":391,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106999"},{"ACCOUNT_CODE":401,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106975"},{"ACCOUNT_CODE":414,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380636733910"},{"ACCOUNT_CODE":431,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106799"}]`

type ToolSqlRunner struct {
	log              *gologgers.Logger
	CallbacksHandler callbacks.Handler
	openAiTool       *llms.Tool
	db               *gorm.DB
	callHandler      lifetools.ToolCallHandler
	isTestMode       bool
}

var _ tools.Tool = ToolSqlRunner{}
var _ lifetools.AvailableTool = ToolSqlRunner{}

func NewToolSqlRunner(l *gologgers.Logger, d *gorm.DB, ch lifetools.ToolCallHandler, isTest ...bool) *ToolSqlRunner {
	if d == nil {
		panic("db is nil")
	}

	// lifetools.InitCimDB(d)
	return &ToolSqlRunner{
		log:              l,
		CallbacksHandler: callbackhandlers.NewLoggerHandler(l),
		callHandler:      ch,
		db:               d,
		isTestMode:       utils.FirstOrDefault(false, isTest...),
		openAiTool: &llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name: "SqlRunner",
				Description: `SqlRunner is a tool designed to execute SQL queries on an Oracle database. It takes a JSON object as input, which must contain a single required field: sqlQuery. The sqlQuery field should hold a valid SQL query string.

Requirements:
- Input Format: JSON object
- Required Field: sqlQuery (string) // valid SQL query

Example Input: {"sqlQuery": "valid SQL query that conforms to the syntax rules of Oracle SQL"}`,
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"sqlQuery": map[string]interface{}{
							"type":        "string",
							"description": "valid SQL query that conforms to the syntax rules of Oracle SQL",
						},
					},
					"required": []string{"sqlQuery"},
				},
			},
		},
	}
}

func NewToolSqlRunner2(l *gologgers.Logger, d *gorm.DB, isTest ...bool) *ToolSqlRunner {
	if d == nil {
		panic("db is nil")
	}

	// lifetools.InitCimDB(d)
	tsr := &ToolSqlRunner{
		log:              l,
		CallbacksHandler: callbackhandlers.NewLoggerHandler(l),
		db:               d,
		isTestMode:       utils.FirstOrDefault(false, isTest...),
		openAiTool: &llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name: "SqlRunner",
				Description: `SqlRunner is a tool designed to execute SQL queries on an Oracle database. It takes a JSON object as input, which must contain a single required field: sqlQuery. The sqlQuery field should hold a valid SQL query string.

Requirements:
- Input Format: JSON object
- Required Field: sqlQuery (string) // valid SQL query

Example Input: {"sqlQuery": "valid SQL query that conforms to the syntax rules of Oracle SQL"}`,
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"sqlQuery": map[string]interface{}{
							"type":        "string",
							"description": "valid SQL query that conforms to the syntax rules of Oracle SQL",
						},
					},
					"required": []string{"sqlQuery"},
				},
			},
		},
	}
	tsr.callHandler = tsr.ExecuteSQLQuery
	return tsr
}

func (c ToolSqlRunner) GetTool() *llms.Tool {
	return c.openAiTool
}

// Description returns a string describing the calculator tool.
func (c ToolSqlRunner) Description() string {
	return c.openAiTool.Function.Description
}

// Name returns the name of the tool.
func (c ToolSqlRunner) Name() string {
	return c.openAiTool.Function.Name
}

func (c ToolSqlRunner) GetCallHandler() lifetools.ToolCallHandler {
	return c.callHandler
}

func (c ToolSqlRunner) Call(ctx context.Context, input string) (string, error) {
	rec := c.log.RecWithCtx(ctx, "sql-tool")
	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolStart(ctx, input)
	}

	// result, err := c.getRelevantDocsFromVectorDB2(ctx, []byte(input))
	// r, err := lifetools.ExecuteSQLQuery(rec, []byte(input))
	r, err := c.callHandler(rec, []byte(input))
	if err != nil {
		rec.Errorf("error from vactrodb: %s", err.Error())
		return "", err
	}

	// r := utils.JsonStr(result)
	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolEnd(ctx, r)
	}

	return r, nil
}

func (c ToolSqlRunner) ExecuteSQLQuery(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	var query struct {
		ClarifyQuestion string `json:"clarifyQuestion"`
		SQLQuery        string `json:"sqlQuery"`
		OtherResponse   string `json:"otherResponse"`
	}

	if err := json.Unmarshal(arguments, &query); err != nil {
		query.SQLQuery = string(arguments)
		if !strings.HasPrefix(query.SQLQuery, "SELECT") {
			rec.Errorf("Error unmarshal arguments: %v; ARGs=[%s]", err, arguments)
			return "", err
		}
	} else {
		rec.Info("Unmarshall success!")
	}

	query.SQLQuery = strings.Trim(query.SQLQuery, " ;")

	rec.Infof("Call tool 'ExecuteSQLQuery' for query: [%s]", query.SQLQuery)

	if c.isTestMode {
		rec.Debug("Test mode is enabled. Return test data")
		return sqlRes, nil
	}

	res, err := runSQLQuery(rec.Ctx, c.db, query.SQLQuery)
	if err != nil {
		rec.Errorf("Error db for running sql: %v", err)
		return "", err
	}
	return utils.JsonStr(res), nil

}

func runSQLQuery(ctx context.Context, db *gorm.DB, query string) (res []map[string]interface{}, err error) {
	err = db.WithContext(ctx).Raw(query).Scan(&res).Error
	return
}
