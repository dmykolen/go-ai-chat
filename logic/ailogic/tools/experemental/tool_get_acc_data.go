package agents

import (
	"context"
	"encoding/json"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	"gitlab.dev.ict/golang/libs/gologgers"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	lifetools "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
)

type ToolGetAccData struct {
	log              *gologgers.Logger
	CallbacksHandler callbacks.Handler
	openAiTool       *llms.Tool
}

var _ tools.Tool = ToolGetAccData{}

func NewToolGetAccData(l *gologgers.Logger) *ToolGetAccData {
	return &ToolGetAccData{
		log:              l,
		CallbacksHandler: callbackhandlers.NewLoggerHandler(l),
		openAiTool: &llms.Tool{
			Type: "function",
			Function: &llms.FunctionDefinition{
				Name:        "getAccountData",
				Description: "API to fetch customer-specific data when needed to provide personalized responses and support. Return: billingAccountID,balances,active tariff,enabled/disabled services,contractNo if subscriber is contracted, etc.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"msisdn": map[string]interface{}{
							"type":        "string",
							"description": "Subscriber phone number",
						},
					},
					"required": []string{"msisdn"},
				},
			},
		},
	}
}

// Description returns a string describing the calculator tool.
func (c ToolGetAccData) Description() string {
	return `
	"A wrapper around Lifecell's API for retrieving subscriber data, such as: "
	"billingAccountID, balances, active tariff, enabled/disabled services, contractNo if subscriber is contracted, etc. "
	"Useful for when you need to fetch customer-specific data when needed to provide personalized responses and support. "
	"Input should be a search query as JSON object format"`
}

// Name returns the name of the tool.
func (c ToolGetAccData) Name() string {
	return "ToolGetAccData"
}

func (c ToolGetAccData) Call(ctx context.Context, input string) (string, error) {
	rec := c.log.RecWithCtx(ctx, "acc-tool")
	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolStart(ctx, input)
	}

	result, err := lifetools.GetAccountData(rec, []byte(input))
	if err != nil {
		rec.Errorf("error search acc data: %s", err.Error())
		return "", err
	}

	if c.CallbacksHandler != nil {
		c.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	return result, nil
}

func (c ToolGetAccData) getAccountData2(ctx context.Context, arguments json.RawMessage) (string, error) {
	var msisdn struct {
		Msisdn string `json:"msisdn"`
	}

	if err := json.Unmarshal(arguments, &msisdn); err != nil {
		return "", err
	}
	c.log.RecWithCtx(ctx, "tool").Infof("Call tool 'getAccountData2' for query: %s", msisdn.Msisdn)
	// Call the API to fetch the account data
	return `{"BillingID":"3833331","Status":"ACT/BAR","Tariff":"CRP_RMF_IP_PBX_FIX_30","Msisdn":"380930164453","ContractNo":"300070023","UseCommonMain":true,"IfEnoughMoney":false,"Balances":{"Line_CM_Usage":0,"Line_Main":0,"Line_SpendingLimit":-99}}`, nil
}
