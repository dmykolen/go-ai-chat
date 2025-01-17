package dbchain

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	al "gitlab.dev.ict/golang/go-ai/logic/ailogic"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	experemental "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools/experemental"
	dic "gitlab.dev.ict/golang/go-ai/logic/db_info_collector"
)

var (
	sep           = strings.Repeat("#", 80)
	logi          = gl.New(gl.WithChannel(""), gl.WithLevel("info"), gl.WithOC(), gl.WithColor())
	logd          = gl.New(gl.WithChannel(""), gl.WithLevel("debug"), gl.WithOC(), gl.WithColor())
	logt          = gl.New(gl.WithChannel(""), gl.WithLevel("trace"), gl.WithOC(), gl.WithColor())
	ctx           = utils.GenerateCtxWithRid()
	llm_openai, _ = openai.New(
		openai.WithModel(al.GPT_4o),
		openai.WithHTTPClient(al.HttpCl),
		openai.WithCallback(callbackhandlers.NewLoggerHandler(logi)))

	llm_openai_JSON, _ = openai.New(
		openai.WithModel(al.GPT_4o),
		openai.WithHTTPClient(al.HttpCl),
		openai.WithCallback(callbackhandlers.NewLoggerHandler(logi)),
		openai.WithResponseFormat(openai.ResponseFormatJSON))

	db dic.DBSchemaInfoProvider = dic.NewDBOracleGorm(dic.DSN(os.Getenv("DB_URL_TM_CIM")), dic.Logger(logi), dic.DictsInclude(), dic.DictsFilters(dic.ScopeDictTablesFilterByRegExp(dic.RegExpDict_CustomerModel), dic.ScopeDictTablesFilterByNumRows(100), dic.ScopeExcludeBkpTmp))

	chatHistory = []llms.ChatMessage{
		llms.HumanChatMessage{Content: "How can you help me?"},
		llms.AIChatMessage{Content: "I am help you with any question about db structure that you provided"},

	}
)

const (
	tq1 = "I want to get all accounts that are students and entered after 2024/03/01"
	tq2 = "Get all accounts that are students"
	tq3 = "Change entered after 2023"
	tq4 = "Run your last query"
)

func TestDbChainNew(t *testing.T) {
	ctx = al.AddToCtxUUIDAI(ctx, "444-4444-3332-111")
	chain := DbChainNew(logi, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx))
	logInfoAboutChain(chain)
}

func TestDbChainNew_Prompt(t *testing.T) {
	ctx = al.AddToCtxUUIDAI(ctx, "444-4444-3332-111")
	chain := DbChainNew(logi, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx))
	// dbStruct := db.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)

	testCases := []struct {
		name, of, dbt, dbs, input string
		hst                       []llms.ChatMessage
	}{
		// {"1", outputFormatJSON, DB_TYPE_ORACLE, testDbStruct, tq1, []llms.ChatMessage{}},
		// {"1", outputFormatEmpty, DB_TYPE_ORACLE, testDbStruct, tq1, []llms.ChatMessage{}},
		{"1", outputFormatEmpty, DB_TYPE_ORACLE, testDbStruct, tq1, chatHistory},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pt, err := chain.Prompt.FormatPrompt(map[string]interface{}{KeyOutFmt: tc.of, KeyDbType: tc.dbt, KeyDbStruct: tc.dbs, KeyInput: tc.input, KeyHistory: tc.hst})
			// pt, err := chain.Prompt.FormatPrompt(map[string]interface{}{KeyInput: tc.input, KeyHistory: tc.hst})
			assert.NoError(t, err)
			t.Log(pt.String())
		})
	}
}

func TestDbChainNew_ChainCall(t *testing.T) {
	// ctx = al.AddToCtxUUIDAI(ctx, "066fd096-a5fe-4a66-9c5c-1bb644e53545")

	// chain := DbChainNew(logt, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx), al.WithLog(logt), al.WithUserLogin("dmykolen"))
	// chain := DbChainNew(logt, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx), al.WithLog(logt), al.WithUserLogin("dmykolen"), al.WithAvailableTools(experemental.NewToolSqlRunner(logd, db.G(), lifetools.ExecuteSQLQuery)))
	chain := DbChainNew(logt, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx), al.WithLog(logt), al.WithUserLogin("dmykolen"), al.WithAvailableTools(experemental.NewToolSqlRunner2(logd, db.G())))
	// dbStruct := db.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)

	testCases := []struct {
		name   string
		inputs map[string]interface{}
	}{
		// {"1", map[string]interface{}{KeyOutFmt: outputFormatEmpty, KeyDbType: DB_TYPE_ORACLE, KeyDbStruct: db.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel), KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		// {"1", map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyDbType: DB_TYPE_ORACLE, KeyDbStruct: testDbStruct, KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		{"1", map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyDbType: DB_TYPE_ORACLE, KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		// {"3", map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		// {"4", map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyInput: tq1}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := chains.Call(ctx, chain, tc.inputs)
			assert.NoError(t, err)
			t.Logf("%v", res)
		})
	}
}

func TestDbChainNew_Call(t *testing.T) {
	ctx = al.AddToCtxUUIDAI(ctx, "444-4444-3332-111")
	chain := DbChainNew(logt, llm_openai, db, al.WithName("DbChain"), al.WithPrompt(defPrompt), al.WithCtx(ctx), al.WithLog(logt), al.WithAvailableTools())
	// dbStruct := db.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel)
	chain.GetCallbackHandler().HandleText(ctx, ">>>>>>> Start chain 'DbChain'! memKey=444-4444-3332-111 output:[map[output:]] expected_input:[map[input:]]")

	testCases := []struct {
		name   string
		inputs map[string]interface{}
	}{
		// {"1", map[string]interface{}{KeyOutFmt: outputFormatEmpty, KeyDbType: DB_TYPE_ORACLE, KeyDbStruct: db.PrepareDBInfoForLLM(ctx, regExpTblsCustomerModel), KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		// {"1", map[string]interface{}{KeyOutFmt: outputFormatEmpty, KeyDbType: DB_TYPE_ORACLE, KeyDbStruct: testDbStruct, KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
		{"2", map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyInput: tq1, KeyHistory: []llms.ChatMessage{}}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := chain.Call(ctx, tc.inputs)
			assert.NoError(t, err)
			t.Logf("%v", res)
		})
	}
}

func Test_sqlite(t *testing.T) {
	type User struct {
		ID    uint   `gorm:"primaryKey"`
		Name  string `gorm:"size:255"`
		Email string `gorm:"size:255"`
	}

	// Connect to SQLite database
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&User{})

	// Insert a new record
	newUser := User{Name: "John Doe", Email: "johndoe@example.com"}
	db.Create(&newUser)
	t.Log("Inserted user:", newUser)

	// Update the record
	db.Model(&newUser).Update("Email", "newemail@example.com")
	t.Log("Updated user:", newUser)

	// Select the record
	var user User
	db.First(&user, newUser.ID) // find user with primary key
	t.Log("Selected user:", user)

	// Select with conditions
	var users []User
	db.Where("name = ?", "John Doe").Find(&users)
	t.Log("Selected users with name 'John Doe':", users)

}

func logInfoAboutChain(c *DbChain) {
	c.log.Info(sep)
	c.log.Infof("ChainName: %s", c.ChainName)
	c.log.Infof("Prompt expected variables: %v", c.Prompt.GetInputVariables())
	c.log.Infof("OutputKey: %s", c.OutputKey)
	c.log.Infof("MemoryKey: %#v", c.Memory.GetMemoryKey(ctx))
	c.log.Infof("GetInputKeys: %#v", c.GetInputKeys())
	c.log.Infof("GetOutputKeys: %#v", c.GetOutputKeys())
	// c.log.Info(c.Prompt.FormatPrompt(map[string]any{c.inputKey: "json", defKeyOutFmt: c.OutputParser.GetFormatInstructions()}))
	// c.log.Infof("PROMPT: %#v", c.Prompt)
	c.log.Info(sep)
}

const testDbStruct = `Table: CM_DICT_SEGMENT_RULES (6 rows)
Columns:
 - CODE NUMBER(22) NOT NULL
 - NUMBER_OF_EMPLOYEES NUMBER(22)
 - ANNUAL_TURNOVER NUMBER(22)
Constraints:
 - Foreign Keys:
   - CODE (Constraint: CM_D_S_RULES_SEG_CODE_FK, Referenced Table: CM_DICT_SEGMENT, Referenced Column: CODE)

Table: CM_DICT_SEGMENT_TYPE (2 rows)
Columns:
 - CODE NUMBER(22) NOT NULL
 - CODE_NAME VARCHAR2(32) NOT NULL
 - DESCRIPTION VARCHAR2(512)
Constraints:
 - Primary Keys:
   - CODE (Position: 1)`

func TestDbChain_Run(t *testing.T) {
	var testVoipOptions = func(sseChannel chan string) []al.RunOptFn {
		return []al.RunOptFn{
			al.WithCBSse(al.CallbackSSEStream),
			al.WithSSEChan(sseChannel),
			al.WithChainOptions(chains.WithCallback(callbackhandlers.NewLoggerHandler(logd))),
		}
	}
	var testInputs = func(userPrompt string) map[string]any {
		return map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyInput: userPrompt, KeyHistory: []llms.ChatMessage{}}
	}
	// lifetools.TestMode()
	sseChannel := make(chan string)
	go readFromChannelAndPrint(sseChannel)

	// ctx := al.AddToCtxUUIDAI(ctx, "116b3463-6c4e-4245-9a08-6dfb9de91fd4")
	chain := DbChainNew(logt, llm_openai, db,
		al.WithName("DbChain"),
		al.WithPrompt(defPrompt),
		al.WithCtx(ctx),
		al.WithLog(logt),
		al.WithUserLogin("dmykolen"),
		al.WithAvailableTools(experemental.NewToolSqlRunner2(logd, db.G())))
	// resultMap, err := chain.Run(context.Background(), testInputs(tq1), testVoipOptions(sseChannel)...)
	resultMap, err := chain.Run(context.Background(), testInputs(tq4), testVoipOptions(sseChannel)...)
	assert.NoError(t, err)
	assert.NotNil(t, resultMap)
	t.Logf("ResultMap: %#v", resultMap)
}

func Test4(t *testing.T) {
	chain := DbChainNew(logt, llm_openai, db,
		al.WithName("DbChain"),
		al.WithCtx(ctx),
		al.WithLog(logt),
		al.WithUserLogin("dmykolen"),
		al.WithAvailableTools(experemental.NewToolSqlRunner2(logd, db.G())))
	t.Log(chain.Prompt)
}
func Test3(t *testing.T) {
	sql1 := `SELECT ca.CODE AS ACCOUNT_CODE, ca.CUSTOMER_CODE, ca.ACCOUNT_TYPE_CODE, ca.REFERENCE, ca.ACCOUNT_CREATION_DATE, ca.LINK_CHANNEL, ca.LINK_AGENT, ca.ACCOUNT_EXPIRY_DATE
FROM CM_CUSTOMER_ACCOUNT ca
JOIN CM_ACCOUNT_ATTRIBUTE aa ON ca.CODE = aa.ACCOUNT_CODE
JOIN CM_CUSTOMER c ON ca.CUSTOMER_CODE = c.CODE
WHERE aa.ATTRIBUTE_TYPE_CODE = 'STUDENT_STATE'
  AND aa.VALUE = 'ACT_STUDENT'
  AND c.ENTRY_DATE > TO_DATE('2023-01-01', 'YYYY-MM-DD')
FETCH FIRST 30 ROWS ONLY    ;    +   `

	t.Logf("[%s]\n=====\n", strings.Trim(sql1, " ;"))
}

func Test2(t *testing.T) {
	sql1 := `SELECT ca.CODE AS ACCOUNT_CODE, ca.CUSTOMER_CODE, ca.ACCOUNT_TYPE_CODE, ca.REFERENCE, ca.ACCOUNT_CREATION_DATE, ca.LINK_CHANNEL, ca.LINK_AGENT, ca.ACCOUNT_EXPIRY_DATE
FROM CM_CUSTOMER_ACCOUNT ca
JOIN CM_ACCOUNT_ATTRIBUTE aa ON ca.CODE = aa.ACCOUNT_CODE
JOIN CM_CUSTOMER c ON ca.CUSTOMER_CODE = c.CODE
WHERE aa.ATTRIBUTE_TYPE_CODE = 'STUDENT_STATE'
  AND aa.VALUE = 'ACT_STUDENT'
  AND c.ENTRY_DATE > TO_DATE('2023-01-01', 'YYYY-MM-DD')
FETCH FIRST 30 ROWS ONLY`
	toolSql := experemental.NewToolSqlRunner2(logd, db.G())
	// r, e := toolSql.GetCallHandler()(logd.RecWithCtx(ctx), []byte("SELECT * FROM CM_DICT_SEGMENT_RULES"))
	r, e := toolSql.GetCallHandler()(logd.RecWithCtx(ctx), []byte(sql1))
	t.Logf("Result: %v, Error: %v", r, e)
}

func readFromChannelAndPrint(sseChannel chan string) {
	for chunk := range sseChannel {
		fmt.Printf("Received chunk ===>[%s]\n", chunk)
	}
}
