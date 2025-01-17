package ailogic

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"gitlab.dev.ict/golang/libs/gohttp"
	"gitlab.dev.ict/golang/libs/utils"
)

//go:embed prompts/sys_first_3_a.txt
var sysFirst3AContent string

type TestCase struct {
	Name, Conv, Description string
	Exp                     Expected
}

type Expected struct {
	Expected           string
	IsValidQuery       bool
	ClarifyingQuestion string
	NexAgent           bool
}

var (
	responseFormat *openai.ResponseFormat = &openai.ResponseFormat{
		Type: "json_schema",
		JSONSchema: &openai.ResponseFormatJSONSchema{
			Name:   "query_response",
			Strict: true,
			Schema: &openai.ResponseFormatJSONSchemaProperty{
				Type: "object",
				Required: []string{
					"response",
					"IsValidQuery",
					"isMsisdnRequired",
					"clarifyingQuestion",
					"missingInfo",
					"nextAgent",
					"reasoning",
					"self_criticism",
					"userIntent",
					"contextualized_query",
					"support_letter",
					"escalation_path",
					"queryType",
				},
				Properties: map[string]*openai.ResponseFormatJSONSchemaProperty{
					"queryType": {
						Type: "string",
					},
					"response": {
						Type:        "string",
						Description: "A Markdown-formatted message responding to the user's inquiry, or empty if `nextAgent` is true.",
					},
					"nextAgent": {
						Type:        "boolean",
						Description: "Indicates if the query requires escalation.",
					},
					"reasoning": {
						Type: "array",
						Items: &openai.ResponseFormatJSONSchemaProperty{
							Type: "string",
						},
						Description: "Step-by-step logical reasoning leading to the conclusion.",
					},
					"userIntent": {
						Type:        "string",
						Description: "The user's intent as interpreted from the query.",
					},
					"missingInfo": {
						Type:        "string",
						Description: "Details or information missing from the user's query.",
					},
					"IsValidQuery": {
						Type:        "boolean",
						Description: "Determines if the query is valid.",
					},
					"self_criticism": {
						Type:        "string",
						Description: "Critique of the process or the generated response.",
					},
					"support_letter": {
						Type:        "string",
						Description: "Non-empty only if the user requests review by a Lifecell employee (customer support). Create detailed support letters for escalations, summarizing the context clearly.",
					},
					"isMsisdnRequired": {
						Type:        "boolean",
						Description: "Specifies whether the MSISDN (mobile subscriber number) is required.",
					},
					"clarifyingQuestion": {
						Type:        "string",
						Description: "A clarifying question to refine the user's query.",
					},
					"contextualized_query": {
						Type:        "string",
						Description: "If `nextAgent` is false, this field is empty; otherwise, it provides a rewritten, independent query including all necessary key detailes from the current chat history for further investigation. (on behalf of the user and in the user's language)",
					},
					"escalation_path": {
						Type:        "string",
						Description: "Possible values: '', 'ai_agent_2', 'CS' (customer support).",
					},
				},
				AdditionalProperties: false,
			},
		},
	}
	conversationHistory1 = []llms.MessageContent{
		{
			Role:  llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{llms.TextPart(sysFirst3AContent)},
		},
	}
	callOptsTest = []llms.CallOption{llms.WithMaxTokens(800), llms.WithTemperature(0.1)}
	httpCl       = gohttp.NewHttpClient(gohttp.WithLog(logTrace), gohttp.WithPRX(gohttp.ProxyAstelit), gohttp.WithTO(300)).Client

	conv_ask_msisdn    = `[{"role":"human","text":"I want to know my balance"}]`
	conv_ask_409_error = `[{"role":"human","text":"Що означає помилка 409?"}]`
	// conv_how_balance_topup = `[{"role":"human","text":"Як поповнити баланс номера 380442334273?"}]`
	// conv_how_balance_topup = `[{"role":"human","text":"Дай мені урл на сайт Lifecell"}]`
	conv_how_balance_topup = `[{"role":"human","text":"Give me the URL to Lifecell"}]`
	// conv_how_balance_topup  = `[{"role":"human","text":"Як поповнити баланс номера 380442334273?"},{"role":"ai","text":"Щоб поповнити баланс номера Lifecell, ви можете скористатися одним з наступних способів:\n\n1. **Через банківську картку**: Використовуйте мобільний додаток вашого банку або інтернет-банкінг для поповнення рахунку.\n2. **Через термінали самообслуговування**: Знайдіть найближчий термінал, який підтримує поповнення мобільного зв'язку, і дотримуйтесь інструкцій на екрані.\n3. **Через офіційний сайт Lifecell**: Перейдіть на [офіційний сайт Lifecell](https://www.lifecell.ua/uk/mobilnij-zvyazok/oplata/) і скористайтеся сервісом поповнення рахунку.\n4. **Через SMS**: Відправте SMS з кодом поповнення на номер, вказаний на картці поповнення.\n\nЯкщо у вас виникнуть додаткові запитання, будь ласка, звертайтеся! 😊"},{"role":"human","text":"Звідки ти взяв цю інформацію? Посилання 'https://www.lifecell.ua/uk/mobilnij-zvyazok/oplata/' не працює! Ти дав не валідну інформацію"}]`
	conv_test_pass_to_agent = `[{"role":"human","text":"I want to know my balance"},{"role":"ai","text":"Could you please provide your MSISDN in the format 380XXXXXXXXX?"},{"role":"human","text":"380931234567"}]`
	conv_test_not_to_agent  = `[{"role":"human","text":"I want to know my balance"},{"role":"ai","text":"Could you please provide your MSISDN in the format 380XXXXXXXXX?"},{"role":"human","text":"I don't want to provide my MSISDN"}]`
	// Testing when the user wants to know their balance and then asks for more details in Ukrainian. Agent1 should add the MSISDN to contextualized_query.
	conv_test_2_agents = `[{"role":"human","text":"I want to know my balance"},{"role":"ai","text":"Could you please provide your MSISDN in the format 380XXXXXXXXX?"},{"role":"human","text":"380934256552"},{"role":"ai","text":"Your current balance is 0. Please let me know if you need any further assistance."},{"role":"human","text":"Дай мені детальну інформацію про всі мої баланси"}]`
	// Testing when the user wants to know their settings. Agent1 should add the MSISDN to contextualized_query. The user's query is in Ukrainian.
	conv_not_in_resposibilities = `[{"role":"human","text":"Як стати CEO Lifecell?"}]`
	conv_test_settings          = `[{"role":"human","text":"I want to know my balance"},{"role":"ai","text":"Could you please provide your MSISDN in the format 380XXXXXXXXX?"},{"role":"human","text":"380934256552"},{"role":"ai","text":"Your current balance is 0. Please let me know if you need any further assistance."},{"role":"human","text":"Дай мені мої налаштування"}]`
	conv_test_settings2         = `[{"role":"human","text":"Дай мені мої налаштування?"},{"role":"ai","text":"Будь ласка, надайте ваш MSISDN у форматі 380XXXXXXXXX, номер контракту або префікс, щоб я міг допомогти з вашими налаштуваннями."},{"role":"human","text":"ось мій номер контракту 30000894 та МСІСДН 380934256552"}]`
	conv_test_settings3         = `[{"role":"human","text":"Дай мені мої налаштування для контракту 30000894?"}]`
	// conv_test_settings2 = `[{"role":"human","text":"Дай мені баланси для контракту?"}]`
	// conv_test_settings2 = `[{"role":"human","text":"Дай мені баланси для контракту?"},{"role":"ai","text":"Будь ласка, надайте ваш MSISDN у форматі 380XXXXXXXXX?"},{"role":"human","text":"30000894"}]`

	cases = []TestCase{
		{
			Name: "TestNotInResponsibilities",
			Conv: conv_not_in_resposibilities,
			Exp: Expected{
				IsValidQuery:       false,
				ClarifyingQuestion: "",
				NexAgent:           false,
			},
		},
		{
			Name: "TestAskMsisdn",
			Conv: conv_ask_msisdn,
			Exp: Expected{
				IsValidQuery:       false,
				ClarifyingQuestion: "",
				NexAgent:           false,
			},
		},
		{
			Name: "TestPassToAgent",
			Conv: conv_test_pass_to_agent,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           true,
			},
		},
		{
			Name: "TestNotToAgent",
			Conv: conv_test_not_to_agent,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           false,
			},
		},
		{
			Name: "Test2Agents",
			Conv: conv_test_2_agents,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           true,
			},
		},
		{
			Name:        "TestSettings",
			Conv:        conv_test_settings,
			Description: "1. User ask for balance 2. Agent asks for MSISDN 3. User provides MSISDN 4. Agent provides balance 5. User asks for settings in Ukrainian",
			Exp: Expected{
				Expected:           "nextAgent=true and contextualized_query contains MSISDN from the conversation",
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           true,
			},
		},
		{
			Name: "TestSettings2",
			Conv: conv_test_settings2,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           true,
			},
		},
		{
			Name: "TestTopUpBalance",
			Conv: conv_how_balance_topup,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           false,
			},
		},
		{
			Name: "Test_err_409",
			Conv: conv_ask_409_error,
			Exp: Expected{
				IsValidQuery:       true,
				ClarifyingQuestion: "",
				NexAgent:           true,
			},
		},
	}
)

func TestFirstAgent(t *testing.T) {
	t.Log("####################")
	testCases := []TestCase{}
	testCases = append(testCases, cases...)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			llm := newTestClient(t, openai.WithModel(GPT_4o), openai.WithHTTPClient(httpCl), openai.WithResponseFormat(responseFormat))

			r, e := llm.GenerateContent(ctx, help_create_conversation(t, tc.Conv), callOptsTest...)
			assert.NoError(t, e)
			mapRes := help_parse_resp(t, r)

			assert.Equal(t, tc.Exp.IsValidQuery, mapRes["IsValidQuery"])
			assert.Equal(t, tc.Exp.NexAgent, mapRes["nextAgent"])
		})
	}

}

func newTestClient(t *testing.T, opts ...openai.Option) llms.Model {
	t.Helper()
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
		return nil
	}

	llm, err := openai.New(opts...)
	require.NoError(t, err)
	return llm
}

func help_parse_resp(t *testing.T, resp *llms.ContentResponse) map[string]any {
	t.Helper()
	var mapRes map[string]any
	err = json.Unmarshal([]byte(resp.Choices[0].Content), &mapRes)
	assert.NoError(t, err)

	t.Log(utils.JsonPrettyStr(mapRes))

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Field", "Type", "Value"})
	table.SetColMinWidth(2, 150)
	table.SetColWidth(80)
	table.SetReflowDuringAutoWrap(true)

	// Add rows to table
	table.Rich([]string{"response", fmt.Sprintf("%T", mapRes["response"]), fmt.Sprintf("%s", mapRes["response"])}, []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.BgRedColor},
		{tablewriter.Bold, tablewriter.BgGreenColor},
		{tablewriter.Bold, tablewriter.FgHiGreenColor},
	})
	table.Append([]string{"hypotheticalAnswer", fmt.Sprintf("%T", mapRes["hypotheticalAnswer"]), fmt.Sprintf("%s", mapRes["hypotheticalAnswer"])})
	table.Append([]string{"IsValidQuery", fmt.Sprintf("%T", mapRes["IsValidQuery"]), fmt.Sprintf("%v", mapRes["IsValidQuery"])})
	table.Append([]string{"isMsisdnRequired", fmt.Sprintf("%T", mapRes["isMsisdnRequired"]), fmt.Sprintf("%v", mapRes["isMsisdnRequired"])})
	table.Rich([]string{"clarifyingQuestion", fmt.Sprintf("%T", mapRes["clarifyingQuestion"]), fmt.Sprintf("%s", mapRes["clarifyingQuestion"])}, []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.BgRedColor},
		{tablewriter.Bold, tablewriter.BgGreenColor},
		{tablewriter.Bold, tablewriter.FgHiGreenColor},
	})
	table.Append([]string{"decisionMaking", fmt.Sprintf("%T", mapRes["decisionMaking"]), fmt.Sprintf("%s", mapRes["decisionMaking"])})
	table.Append([]string{"missingInfo", fmt.Sprintf("%T", mapRes["missingInfo"]), fmt.Sprintf("%s", mapRes["missingInfo"])})
	table.Append([]string{"nextAgent", fmt.Sprintf("%T", mapRes["nextAgent"]), fmt.Sprintf("%v", mapRes["nextAgent"])})
	table.Rich([]string{"nextAgent", fmt.Sprintf("%T", mapRes["nextAgent"]), fmt.Sprintf("%v", mapRes["nextAgent"])}, []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.BgRedColor},
		{tablewriter.Bold, tablewriter.BgGreenColor},
		{tablewriter.Bold, tablewriter.FgHiGreenColor},
	})
	table.Append([]string{"reasoning", fmt.Sprintf("%T", mapRes["reasoning"]), fmt.Sprintf("%v", mapRes["reasoning"])})
	table.Append([]string{"self_criticism", fmt.Sprintf("%T", mapRes["self_criticism"]), fmt.Sprintf("%s", mapRes["self_criticism"])})
	table.Append([]string{"userIntent", fmt.Sprintf("%T", mapRes["userIntent"]), fmt.Sprintf("%s", mapRes["userIntent"])})
	table.Rich([]string{"contextualized_query", fmt.Sprintf("%T", mapRes["contextualized_query"]), fmt.Sprintf("%s", mapRes["contextualized_query"])}, []tablewriter.Colors{
		{tablewriter.Bold, tablewriter.BgRedColor},
		{tablewriter.Bold, tablewriter.BgGreenColor},
		{tablewriter.Bold, tablewriter.FgHiGreenColor},
	})
	table.Append([]string{"support_letter", fmt.Sprintf("%T", mapRes["support_letter"]), fmt.Sprintf("%s", mapRes["support_letter"])})

	// Render table
	table.Render()
	return mapRes
}

func help_create_conversation(t *testing.T, conv string) []llms.MessageContent {
	t.Helper()
	var mc []llms.MessageContent
	err := json.Unmarshal([]byte(conv), &mc)
	assert.NoError(t, err)
	conversationHistory1 = append(conversationHistory1, mc...)
	t.Logf("len(conversationHistory1) ==> %d", len(conversationHistory1))
	if !assert.Equal(t, 0, len(conversationHistory1)%2, "len of conversation is odd") {
		t.FailNow()
	}
	return conversationHistory1
}

func TestF1(t *testing.T) {
	s := `{"Choices":[{"Content":"{\"IsValidQuery\":true,\"clarifyingQuestion\":\"Could you please provide your MSISDN in the format 380XXXXXXXXX?\",\"contextualized_query\":\"\",\"escalation_path\":\"\",\"hypotheticalAnswer\":\"To check your balance, you can dial *111# from your Lifecell number.\",\"isMsisdnRequired\":true,\"missingInfo\":\"MSISDN\",\"nextAgent\":false,\"reasoning\":[\"The user is asking for their balance, which is account-specific information.\",\"To provide this information, the user's MSISDN is required.\",\"Once the MSISDN is provided, the query can be escalated to retrieve the balance information.\"],\"response\":\"Could you please provide your MSISDN in the format 380XXXXXXXXX?\",\"self_criticism\":\"The response correctly identifies the need for the user's MSISDN to proceed with the balance inquiry.\",\"support_letter\":\"\",\"userIntent\":\"The user wants to check their balance.\"}","StopReason":"stop","GenerationInfo":{"CompletionTokens":191,"PromptTokens":1499,"ReasoningTokens":0,"TotalTokens":1690},"FuncCall":null,"ToolCalls":null}]}`
	t.Log(s)

	var response llms.ContentResponse
	err := json.Unmarshal([]byte(s), &response)
	assert.NoError(t, err)

	t.Log(utils.JsonPrettyStr(response))

	mapRes := help_parse_resp(t, &response)
	// Convert reasoning to a slice of strings
	reasoningInterfaces := mapRes["reasoning"].([]interface{})
	reasoning := make([]string, len(reasoningInterfaces))
	for i, v := range reasoningInterfaces {
		reasoning[i] = v.(string)
	}
	t.Logf("reasoning[0] ==> %s", reasoning[0])

	t.Log(utils.JsonPrettyStr(help_create_conversation(t, conv_test_pass_to_agent)))
}
