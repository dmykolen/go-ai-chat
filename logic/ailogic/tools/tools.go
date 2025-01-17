package tools

import (
	"encoding/json"
	"strings"

	"github.com/tmc/langchaingo/llms"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/gorm"

	"gitlab.dev.ict/golang/go-ai/logic/biz"
)

const (
	accData = `{"BillingID":"3833331","Status":"ACT/BAR","Tariff":"CRP_RMF_IP_PBX_FIX_30","Msisdn":"380930164453","ContractNo":"300070023","UseCommonMain":true,"IfEnoughMoney":false,"Balances":{"Line_CM_Usage":0,"Line_Main":0,"Line_SpendingLimit":-99}}`
	relDoc  = "ACT/BAR status means that the account is active but barred(one of the reasons could be that the account has reached the spending limit)."
	sqlRes  = `[{"ACCOUNT_CODE":391,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106999"},{"ACCOUNT_CODE":401,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106975"},{"ACCOUNT_CODE":414,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380636733910"},{"ACCOUNT_CODE":431,"ATTRIBUTE_TYPE_CODE":"MSISDN","VALUE":"380632106799"}]`
)

var (
	ToolFuncs = []llms.Tool{
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "getAccountData",
				Description: "API to fetch customer-specific data when needed to provide personalized responses and support. Return: billingAccountID,balances,active tariff,enabled/disabled services,contractNo if subscriber is contracted, etc.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"msisdn": map[string]interface{}{
							"type":        "string",
							"description": "Subscriber phone number(pattern for MSISDN: `380\\d{9}`)",
						},
					},
					"required": []string{"msisdn"},
				},
			},
		},
		{
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
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "get_FMC_VOIP_settings",
				Description: "Return FMC VoIP settings, which will helpful to handle complaints related to in/out calls or provide to user his on ask",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"msisdn": map[string]interface{}{
							"type":        "string",
							"description": "Mobile Station Integrated Services Digital Network number, optional and should have tariff if provided",
						},
						"tariff": map[string]interface{}{
							"type":        "string",
							"description": "Tariff associated with the provided MSISDN, should only be included if msisdn is provided",
						},
						"contractNo": map[string]interface{}{
							"type":        "string",
							"description": "Contract number",
						},
					},
					"required":             []string{"contractNo", "msisdn", "tariff"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "get_FMC_MOBILE_settings",
				Description: "Return FMC MOBILE settings",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"msisdn": map[string]interface{}{
							"type":        "string",
							"description": "Mobile Station Integrated Services Digital Network number(MSISDN), optional and should have tariff if provided",
						},
						"tariff": map[string]interface{}{
							"type":        "string",
							"description": "Tariff associated with the provided MSISDN, should only be included if msisdn is provided",
						},
						"contractNo": map[string]interface{}{
							"type":        "string",
							"description": "Contract number associated with the mobile service",
						},
					},
					"required":             []string{"contractNo", "msisdn", "tariff"},
					"additionalProperties": false,
				},
			},
		},
	}

	AvailableVoipTools = map[string]ToolCallHandler{
		"getAccountData":              GetAccountData,
		"getRelevantDocsFromVectorDB": GetRelevantDocsFromVectorDB,
		"get_FMC_VOIP_settings":       GetFMCVoipSettings,
		"get_FMC_MOBILE_settings":     GetFMCMobileSettings,
	}

	AvailableSqlTools = map[string]ToolCallHandler{
		"": GetAccountData,
	}
)

type ToolCallHandler func(rec *gologgers.LogRec, arguments json.RawMessage) (string, error)

var isTestMode = false
var ws *biz.WSGetter
var db *w.KnowledgeBase
var cimDB *gorm.DB

func TestMode(isEnable ...bool) {
	isTestMode = utils.FirstOrDefault(true, isEnable...)
}

func InitTools(wsGetter *biz.WSGetter, dbVector *w.KnowledgeBase) {
	once.Do(func() { ws = wsGetter; db = dbVector })
}

func InitCimDB(db *gorm.DB) {
	once.Do(func() { cimDB = db })
}

func GetAccountData(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	validateWS()
	var msisdn struct {
		Msisdn string `json:"msisdn"`
	}

	if err := json.Unmarshal(arguments, &msisdn); err != nil {
		return "", err
	}
	rec.Infof("Call tool 'getAccountData' for query: %s", msisdn.Msisdn)
	if isTestMode {
		return accData, nil
	}

	acc, err := ws.GetAccount(rec.Ctx, msisdn.Msisdn)
	if err != nil {
		return "", err
	}
	return acc.String(), nil
}

func GetRelevantDocsFromVectorDB(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	validateDB()
	var query struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal(arguments, &query); err != nil {
		return "", err
	}

	rec.Infof("Call tool 'getRelevantDocsFromVectorDB' for query: %s", query.Query)

	if isTestMode {
		return relDoc, nil
	}

	gqlResp, err := w.WeaviateSearch(rec, db.Client, db.Class, w.DefaultSO().Limit(2).Fields(w.FieldContent).SearchTxt(query.Query))
	if err != nil {
		return "", err
	}

	rec.Info("Finish vector db")
	ki := w.GQLRespConvert[w.KnowledgeItem](gqlResp, db.Class)
	knowItems := w.KnowledgeItems(ki)
	rec.Infof("VectorDB return %d objects", knowItems.Len())
	rec.Debugf("Documents from VectorDB:\n%s", knowItems.Json())
	return knowItems.Json(), nil
}

func ExecuteSQLQuery(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	var query struct {
		ClarifyQuestion string `json:"clarifyQuestion"`
		SQLQuery        string `json:"sqlQuery"`
		OtherResponse   string `json:"otherResponse"`
	}

	if err := json.Unmarshal(arguments, &query); err != nil {
		query.SQLQuery = strings.Trim(string(arguments), " ;")
		if !strings.HasPrefix(query.SQLQuery, "SELECT") {
			rec.Errorf("Error unmarshal arguments: %v; ARGs=[%s]", err, arguments)
			return "", err
		}
	}

	rec.Infof("Call tool 'ExecuteSQLQuery' for query: %s", query.SQLQuery)

	if isTestMode {
		rec.Debug("Test mode is enabled. Return test data")
		return sqlRes, nil
	}

	res, err := runSQLQuery(rec.Ctx, cimDB, query.SQLQuery)
	if err != nil {
		rec.Errorf("Error db for running sql: %v", err)
		return "", err
	}
	return utils.JsonStr(res), nil

}

func GetFMCVoipSettings(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	validateWS()

	var params struct {
		Msisdn     string `json:"msisdn"`
		Tariff     string `json:"tariff"`
		ContractNo string `json:"contractNo"`
	}

	if err := json.Unmarshal(arguments, &params); err != nil {
		return "", err
	}

	rec.Infof("Call tool 'get_FMC_VOIP_settings' with contractNo=%s msisdn=%s tariff=%s", params.ContractNo, params.Msisdn, params.Tariff)

	if isTestMode {
		return `{"settings": "test settings"}`, nil
	}

	response, err := ws.GetFMC_VOIP(rec.Ctx, params.ContractNo, params.Msisdn)
	if err != nil {
		return "", err
	}

	return utils.JsonStr(response), nil
}

func GetFMCMobileSettings(rec *gologgers.LogRec, arguments json.RawMessage) (string, error) {
	validateWS()

	var params struct {
		Msisdn     string `json:"msisdn"`
		Tariff     string `json:"tariff"`
		ContractNo string `json:"contractNo"`
	}

	if err := json.Unmarshal(arguments, &params); err != nil {
		return "", err
	}

	rec.Infof("Call tool 'get_FMC_MOBILE_settings' with contractNo=%s msisdn=%s tariff=%s",
		params.ContractNo, params.Msisdn, params.Tariff)

	if isTestMode {
		return `{"settings": "test mobile settings"}`, nil
	}

	response, err := ws.GetFMC_MOBILE(rec.Ctx, params.ContractNo, params.Msisdn)
	if err != nil {
		return "", err
	}

	return utils.JsonStr(response), nil
}
