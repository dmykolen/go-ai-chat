package ailogic

import (
	"os"
	"testing"

	"github.com/caarlos0/env/v6"
	"github.com/olekukonko/tablewriter"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/memory/sqlite3"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gitlab.dev.ict/golang/libs/ws/cimws"
	"gitlab.dev.ict/golang/libs/ws/omws"
)

var (
	ctx       = utils.GenerateCtxWithRid()
	sessId    = utils.UUID()
	callbackH = callbackhandlers.NewLoggerHandler(log)
	logTrace  = gl.New(gl.WithChannel("ai"), gl.WithLevel("trace"), gl.WithOC(), gl.WithColor())
	logInfo   = gl.New(gl.WithChannel("ai"), gl.WithLevel("info"), gl.WithOC(), gl.WithColor())
	log       = logTrace

	uq = []string{
		"Є проблеми з номером 380930164453, при вхідних говорить, що номер поза зоною.",
		"Що означає помилка 409?",
		"Є проблеми з номером, при вхідних говорить, що номер поза зоною.",
		`Звертаємось щодо спільного клієнта "ТОВ АДАПТИС "У клієнта спостерігається складність в роботі SIM 0930164453 fmc mobile, а саме при дзвінку на даний номер звучить голосове повідомлення що даний номер не обслуговується. Прохання уточнити інформацію, що до цього питання, оскільки клієнт не може ним користуватися, та надати відповідь з виправлення цієї ситуації.`,
		"Як стати CEO Lifecell?",
		"Який тариф на номері 380632107489?",
	}
	airesp = []string{
		"Статус лінії - Вихідні дзвінки заблоковано, що може впливати на прийом вхідних дзвінків. Рекомендуємо звернутись до свого менеджера або до кол центру для вирішення цієї проблеми. 📞",
		"",
		"",
		"",
		"Вибачте, але я можу допомогти лише з питаннями, що стосуються підтримки клієнтів, телекомунікацій або послуг Lifecell. Якщо у вас є питання щодо наших послуг або проблем з вашим номером, будь ласка, дайте знати! 😊",
	}

	llm_groq, _ = openai.New(
		openai.WithToken(os.Getenv("GROQ_API_KEY")),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithModel("llama3-70b-8192"),
		openai.WithHTTPClient(HttpCl),
		openai.WithCallback(callbackH),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)

	llm_openai, _ = openai.New(
		openai.WithModel(GPT_4o),
		openai.WithHTTPClient(HttpCl),
		openai.WithCallback(callbackH),
		// openai.WithResponseFormat(openai.ResponseFormatJSON),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)

	llm = llm_openai
	// llm = llm_groq

	chainOpts   = []chains.ChainCallOption{chains.WithCallback(callbackH), chains.WithMaxTokens(1000), chains.WithTemperature(0.25)}
	chatHistory = []llms.ChatMessage{
		llms.SystemChatMessage{Content: PP3},
		llms.HumanChatMessage{Content: uq[3]},
		llms.AIChatMessage{Content: "Please provide MSISDN"},
	}
)

func Help_table_setup(t *testing.T, tw *tablewriter.Table) {
	t.Helper()
	tw.SetHeader([]string{"#", "Input", "Output", "CoT", "SelfCheck"})
	tw.SetAlignment(tablewriter.ALIGN_CENTER)
	tw.SetColWidth(57)
	tw.SetTablePadding("\t\t--->")
	tw.SetRowSeparator("*")
	// tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	// tw.SetCenterSeparator("|")
	tw.SetColumnColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiRedColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgBlackColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
	)
	t.Log("Table was set up!")
}

func Help_init_VECTORDB(t *testing.T) *w.KnowledgeBase {
	t.Helper()
	return w.NewKnowledgeBase(w.NewWVClient(&w.WeaviateCfg{Host: "localhost", Port: "8083", Scheme: "http", Log: log}), log, w.DefaultClassKB, "")
}

func iHelp_init_ws_for_test(t *testing.T) *biz.WSGetter {
	cimwsApiParams := &cimws.ApiParamas{}
	omwsApiParams := &omws.ApiParamas{}
	env.Parse(cimwsApiParams)
	env.Parse(omwsApiParams)
	t.Log(utils.JsonPrettyStr(omwsApiParams))
	t.Log(utils.JsonPrettyStr(cimwsApiParams.DebugOFF().URL("http://dev-test-tm-cim-1.dev.ict:8080")))
	cimCl := cimws.NewClient(cimws.WithParams(cimwsApiParams), cimws.WithLog(log))
	cimCl.SetProxy("http://proxy.astelit.ukr:3128")
	return biz.NewWSGetter(log, cimCl, omws.NewClient(omws.WithParams(omwsApiParams), omws.WithLogger(log)))
}

func Help_init_ALL_tools_for_test(t *testing.T) {
	t.Helper()
	tools.InitTools(iHelp_init_ws_for_test(t), Help_init_VECTORDB(t))
}

func help_prettyPrintStruct_T(t *testing.T, v interface{}) {
	t.Helper()
	// t.Logf("%s\n%# v\n%s\n", sep, pretty.Formatter(v), sep)
	t.Logf(PrettyPrintStruct(v))
}

func help_get_sqliteConvHst() *sqlite3.SqliteChatMessageHistory {
	return sqlite3.NewSqliteChatMessageHistory(sqlite3.WithSession(sessId), sqlite3.WithDB(db))
}

func help_print_mem(t *testing.T, mem schema.Memory) {
	t.Helper()
	t.Log(sep)
	t.Log("MemoryVariables===>", mem.MemoryVariables(ctx))
	mem.(*memory.ConversationBuffer).ReturnMessages = true
	m, e := mem.LoadMemoryVariables(ctx, nil)
	assert.NoError(t, e)
	t.Logf("LoadMemoryVariables with[ReturnMessages=true]===> %#v", m)
	mem.(*memory.ConversationBuffer).ReturnMessages = false
	m, e = mem.LoadMemoryVariables(ctx, nil)
	assert.NoError(t, e)
	t.Logf("LoadMemoryVariables with[ReturnMessages=false]===> %#v", m)
	// err := llmChain.Memory.SaveContext(ctx, MapAny{"human_input": "xxxxxx", "42244": "XXXXXXX"}, MapAny{llmChain.OutputKey: "yyyyyy", "42244": "YYYYYYY"})
}

func Test_call_llm(t *testing.T) {
	llm.CallbacksHandler = callbackhandlers.DefaultLoggerCallback()
	res, err := llm.Call(ctx, "Напиши список своїх най надзвичайніших можливостей! І напиши як я повинен до тебе писати щоб скористатися цими можливостями. Навчи з тобою спілкуватися, щоб досягти найкращих результатів", llms.WithMaxTokens(1024))
	assert.NoError(t, err)
	t.Log(res)

}

func Test2(t *testing.T) {
	q := uq[0]
	t.Logf("User question: %s", q)
	// resp, e := _llm.Call(ctx, q, llms.WithTemperature(0.3))
	resp, e := llms.GenerateFromSinglePrompt(ctx, llm, q, llms.WithTemperature(0.3))
	if e != nil {
		t.Error(e)
	}
	t.Logf("Response: %s", resp)
}

func Test1(t *testing.T) {
	for _, v := range uq {
		t.Logf("User question: %s", v)
		resp, e := llm.Call(ctx, v, llms.WithTemperature(0.3))
		if e != nil {
			t.Error(e)
		}
		t.Logf("Response: %s", resp)

	}
}
