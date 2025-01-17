package wvservice

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/goutil/strutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate/entities/models"
	gh "gitlab.dev.ict/golang/libs/gohttp"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gologgers/applogger"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/goparser"
	"gitlab.dev.ict/golang/libs/utils"
)

const CATEGORY_MN = "MEETING_NOTES"
const CATEGORY_FRD = "FRD"

const (
	FILE1 = "_testdata/docx/GiftsPoolFRD.docx"
	FILE2 = "_testdata/docx/lifecellCampusFRDV2.docx"
	FILE3 = "_testdata/docx/PrepaidRegistartion.docx"
	FILE4 = "_testdata/docx/Meeting_with_ISMET_about_AI.docx"

	FILE_FRD = "assets/frd.json"
	classFRD = "FRD"
	classA   = "Article"
)

var (
	kb         *KnowledgeBase
	logTestKB  = gologgers.New(gologgers.WithChannel("KB"), gologgers.WithLevel("debug"), gologgers.WithOC())
	parserDocx = goparser.NewParser(goparser.WithLogger(logTestKB.Logger))
	// ai         = goopenai.New().WithProxy(false).WithHttpCl(gohttp.New().WithLoggerDefault().WithProxy(nil).WithTimeout(150).Build().Client).WithLogger(logTestKB).Build()
	httpClient = gh.NewHttpClient(gh.WithPRX(nil), gh.WithTO(120), gh.WithLogCFG(applogger.NewLogCfgDef().WithMaxLength(10000))).Client
	ai         = goopenai.New().WithProxy(false).WithHttpCl(httpClient).WithLogger(logTestKB).Build()
	ctx        = utils.GenerateCtxWithRid()
	client     *weaviate.Client
)

func init() {
	// client = WeaviateDefault()
	client = NewWVClient(&WeaviateCfg{
		Host:   "ai.dev.ict",
		Port:   "8083",
		Scheme: "http",
		Loglvl: "debug",
	})
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)
	kb.log.Info("KB initialized:", kb.TotalItems())
}

func TestNewKnowledgeBase(t *testing.T) {
	t.Run("upload-test-1", func(t *testing.T) {
		// kb.AddItem(ctx, "TEST doc 1", "teeest first document", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 2", "teeest first document 222222", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 3", "teeest first document 333333", "", CATEGORY_FRD, "", "")
		err := kb.AddItemsToWeaviate(ctx)
		assert.NoError(t, err)
	})

	t.Run("upload-batch-test-1", func(t *testing.T) {
		// kb.AddItem(ctx, "TEST doc 1", "teeest first document", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 2", "teeest first document 222222", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 3", "teeest first document 333333", "", CATEGORY_FRD, "", "")
		resp, err := kb.AddToWeaviateBatch(ctx)
		assert.NoError(t, err)
		t.Log(utils.JsonPrettyStr(resp))
	})

	t.Run("upload-batch-test-2", func(t *testing.T) {
		// kb.AddItem(ctx, "TEST doc 1", "teeest first document", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 22", "teeest first document 222222", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 33", "teeest first document 333333", "", CATEGORY_FRD, "", "")
		kb.AddItem(ctx, "TEST doc 44", "teeest first document 333333", "", CATEGORY_FRD, "", "")
		resp, err := kb.AddToWeaviateBatchNew(ctx)
		assert.NoError(t, err)
		t.Log(utils.JsonPrettyStr(resp))
	})
}

func TestKnowledgeBase_DeleteFull(t *testing.T) {
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)
	t.Logf("IS_EXISTS class[KB]? => %t", kb.IsClassExists(ctx))
	t.Logf("IS_EXISTS class[FRD]? => %t", kb.IsClassExists(ctx, "FRD"))
	kb.DeleteFull(ctx)
	t.Logf("IS_EXISTS class[KB] after DELETE? => %t", kb.IsClassExists(ctx))
	t.Log(kb)
}

func TestGetObjects(t *testing.T) {
	help_getObjects(t, DefaultClassKB)
}
func TestGetObjects1(t *testing.T) {
	so := DefaultSO().Limit(2).Fields(FieldContent, FieldAdditional2).SortOrder(FieldTitle, false).SearchTxt("error 409")
	t.Log(utils.JsonPrettyStr(so))
	gr, err := kb.Search(ctx, so)
	assert.NoError(t, err)
	t.Log(utils.JsonPrettyStr(gr))

	ki := GQLRespConvert[KnowledgeItem](gr, DefaultClassKB)
	kb.log.Infof("VectorDB return %d objects", KnowledgeItems(ki).Len())
	// t.Log(KnowledgeItems(ki).Json())

	t.Log(utils.JsonPrettyStr(ki[0]))
}

func TestGetObjects3(t *testing.T) {
	so := DefaultSO().Limit(5).Fields(FieldAdditional1).SearchTxt("error 409")
	t.Log(utils.JsonPrettyStr(so))
	gr, err := kb.Search(ctx, so)
	assert.NoError(t, err)
	t.Log("RESULT>>>>\n", utils.JsonPrettyStr(gr))

	ki := GQLRespConvert[KnowledgeItem](gr, DefaultClassKB)
	kb.log.Infof("VectorDB return %d objects", KnowledgeItems(ki).Len())

	t.Log(utils.JsonPrettyStr(ki[0]))
}

func TestGetObjects2(t *testing.T) {
	id := "c77211ad-d0b7-4eae-87f9-ade4a95287b2"
	obj := lo.Must(kb.GetObjByID(ctx, id))
	t.Log("GetObjByID:", help_object_to_str(obj))
	t.Log(utils.JsonPrettyStr(obj))
}

func TestKnowledgeBase_GetObjectsFromWeaviate(t *testing.T) {
	t.Log("START TestKnowledgeBase_GetObjectsFromWeaviate! WD:", goutil.Must(os.Getwd()))
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, "../"+DefaultClassKB_json)
	objList := kb.GetObjectsFromWeaviate(ctx, false)
	assert.NotNil(t, objList)
}

func TestKnowledgeBase_GetObjByTitle(t *testing.T) {
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)
	resp := kb.GetObjByTitle(ctx, "*gift*")
	assert.NotNil(t, resp)
	t.Logf("gqlResp.Data: %v", utils.JsonPrettyStr(resp))
}

func TestKnowledgeBase_GetObjByID(t *testing.T) {
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)

	tests := []struct {
		name  string
		title string
	}{
		{"1", "gift"},
		{"2", "*gift*"},
		{"3", "GiftsPoolFRD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := kb.GetObjByTitle(ctx, tt.title)
			assert.NotNil(t, list)
			if !assert.NotEmpty(t, list, "result of GetObjByTitle(%s) is empty, can't get ID", tt.title) {
				t.FailNow()
			}
			t.Logf("Search obj's ID by title=[%s]. Result: %s", tt.title, list[0].Additional["id"])

			obj := lo.Must(kb.GetObjByID(ctx, list[0].Additional["id"].(string)))
			t.Log("GetObjByID:", obj)
		})
	}

}

func TestKnowledgeBase_SearchInContentsHybrid(t *testing.T) {
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)

	tests := []struct {
		name       string
		searchtext string
	}{
		{"1", "i want to know about student program"},
		{"2", "Is it possible to get a gift in lifecell?"},
		{"3", "Can i passportize on foreign passport?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("START Search by searchtext:", tt.searchtext)
			list := kb.SearchInContentsHybrid(ctx, tt.searchtext, 3)
			assert.NotNil(t, list)
			if !assert.NotEmpty(t, list, "result of SearchInContentsHybrid(%s) is empty, can't get ID", tt.searchtext) {
				t.FailNow()
			}
			t.Logf("Search obj's ID by searchtext=[%s]. Result: %s", tt.searchtext, utils.JsonPrettyStr(list))
		})
	}
}

func TestKnowledgeBase_SearchInContentsHybrid_AI(t *testing.T) {
	const SysPrompt1 = "You are a helpful assistant in the Lifecell chatbot. Lifecell is one of the biggest mobile telecom operators in Ukraine. You should be providing answers and info based on context, which will be provided also with questions."
	const SysPrompt2 = "You are a helpful assistant in the Lifecell chatbot"
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)

	tests := []struct {
		name       string
		searchtext string
		sysroles   []string
	}{
		// {"1", "i want to know about student program", []string{SysPrompt1, SysPrompt2}},
		// {"2", "Is it possible to get a gift in lifecell?", []string{SysPrompt1, SysPrompt2}},
		{"3", "Can i passportize on foreign passport?", []string{SysPrompt1, SysPrompt2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log("START Search by searchtext:", tt.searchtext)
			list := kb.SearchInContentsHybrid(ctx, tt.searchtext, 3)
			assert.NotNil(t, list)
			if !assert.NotEmpty(t, list, "result of SearchInContentsHybrid(%s) is empty, can't get ID", tt.searchtext) {
				t.FailNow()
			}
			t.Logf("Search obj's ID by searchtext=[%s]. Result: %s", tt.searchtext, utils.JsonPrettyStr(list))

			obj := lo.Must(kb.GetObjByID(ctx, IdOfFirstEl(list)))
			t.Log("GetObjByID:", help_object_to_str(obj))

			aiAnswers := &sync.Map{}
			wg := &sync.WaitGroup{}
			for _, sysmsg := range tt.sysroles {
				wg.Add(1)
				go func(msg string) {
					defer wg.Done()
					r := ai.AskAI(ctx, ai.PromptQuestionWithCont(ObjContent(obj), tt.searchtext), goopenai.NewChat(msg))
					aiAnswers.Store(fmt.Sprintf("> SYS: %s\n> Q: %s", msg, tt.searchtext), r)
				}(sysmsg)
			}
			wg.Wait()

			t.Log("##############################################################")
			aiAnswers.Range(func(key, value any) bool {
				t.Logf("\n%s\n> AI_ANSWER: %s\n", key, value)
				return true
			})
			t.Log("##############################################################")
			time.Sleep(10 * time.Second)
		})
	}
}

func TestKnowledgeBase_AddItem(t *testing.T) {
	// help_deleteClass(t, DefaultClassKB)
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)
	kb.DeleteFull(ctx)

	t.Run("0", func(t *testing.T) {
		fn, content := parseFile(t, FILE1)
		summary, keywords := aiSummarizeAndKeywords(t, content)
		kb.AddItem(ctx, fn, content, "", CATEGORY_FRD, summary, keywords)
		t.Log("ITEM[0]>>>", kb.Items[0])
	})
	assert.NotZero(t, kb.Items)
	t.Log("#########################################################", kb.Size())
	t.Run("1", func(t *testing.T) {
		fn, content := parseFile(t, FILE2)
		summary, keywords := aiSummarizeAndKeywords(t, content)
		kb.AddItem(ctx, fn, content, "", CATEGORY_FRD, summary, keywords)
		t.Log("ITEM[1]>>>", kb.Items[1])
	})
	assert.NotZero(t, kb.Items)
	t.Log("#########################################################", kb.Size())
	t.Run("2", func(t *testing.T) {
		fn, content := parseFile(t, FILE3)
		summary, keywords := aiSummarizeAndKeywords(t, content)
		kb.AddItem(ctx, fn, content, "", CATEGORY_FRD, summary, keywords)
	})
	assert.NotZero(t, kb.Items)
	t.Log("#########################################################", kb.Size())

	_, err := kb.AddToWeaviateBatch(ctx)
	assert.NoError(t, err)
	help_getObjects(t, DefaultClassKB)
}

func parseFile(t *testing.T, path string) (fileName, content string) {
	t.Helper()
	fileName = strings.Split(filepath.Base(path), ".")[0]
	content = parserDocx.ReadAndParse(path)
	t.Logf("Read and parse file=%s. Result size: %d. GPT-3-Tokens: %d", fileName, len(content), countTokens(content))

	content = normalize(content)
	t.Logf("Read and parse file=%s. Result size: %d. GPT-3-Tokens: %d", fileName, len(content), countTokens(content))
	return
}

func normalize(in string) string {
	in = strings.Join(strutil.SplitTrimmed(in, "\n"), "\n")
	return in
}

func Test_ParseFile(t *testing.T) {
	n, c := parseFile(t, "../../" + FILE1)
	os.WriteFile(n+".txt", []byte(c), 0666)
}

func aiSummarizeAndKeywords(t *testing.T, content string) (s, keywords string) {
	t.Helper()
	chat := goopenai.NewChat()
	s = ai.AskAI(ctx, ai.PromptSummarize(content), chat)
	t.Log("SUMMARIZE RESPONSE:", s)
	keywords = ai.AskAI(ctx, "generate top 10 keywords of my document above. One line, splitted by comma", chat)
	t.Log("KEYWORDS RESPONSE:", keywords)
	return
}

func TestDeleteClass(t *testing.T) {
	kb = NewKnowledgeBase(client, logTestKB, DefaultClassKB, DefaultClassKB_json)
	kb.DeleteFull(ctx)
	// help_deleteClass(t, "Knowledge_base")
}

func help_getObjects(t *testing.T, className string) []*models.Object {
	objList, err := client.Data().ObjectsGetter().WithClassName(className).WithVector().Do(ctx)
	goutil.PanicErr(err)
	for i, obj := range objList {
		t.Logf("obj[%d]> %s\n", i, help_object_to_str(obj))
	}
	return objList
}

func TestKnowledgeItem_String(t *testing.T) {
	content := `
	Three months from period. It means that an each particular month. You will have two extra customers. Instead of a double so you would have one. Yeah.
	And you will have to kind of double bubble here. This is what you will get. You know, we can actually get by the way. Very nice game. So we will reach 10 million is really, but yeah, our pool, not sure.
	Based on their official statistics, they lost during from Q4 2022, till Q2 2023, they lost about 700,000 customers called 3 months, a few days.
	So like it seems like they lost during last 12 months about 1 million. \n\nI don’t know the way which reminded come how it works.
	I mean this 180 value 110. This is what you should remember this. Let me touch base with him, but I thought that they all had quarter has such a big say, in Ukraine business. No, no, it’s been the three things here is the following.
	So actually if you do it the exclusive channel or in any control channel you may expect, you know I don’t mean at least you can have yourself guys with long tone trying to explain everyone that this would create a huge amount of good customers.
	When you do it in Supermarket, this is definitely. I mean, it’s will never create a good customers on the taxi drivers who come to every month but all takes it right?
	Mm-hmm, the better part I think is important in this, you know, and it turned precise. \n\nOkay. That’s an occasion. Yes, yes, he’s okay. If you like give some more support corporate, we will go to revenue. That’s switch to revenue. I have detailed corporate business review.
	Annuals because we should have everything inside the DC in output called outgoing subscribers on. It’s predictable. This previously and changes. If you open the charger and you see that, this is accumulation campaign. Also give us zero point, eight percent in absolute figures around eight, nine, eight million extra resume which converts to partially in this month.
	And we see one billion charge plus, seven million months over month, and on financial side, off`
	tests := []struct {
		name string
		ki   *KnowledgeItem
		want string
	}{
		{"1", NewKI("title string", content, "url string", "category string", "summary string", "keyWords string"), "KnowledgeItem: title=title string; content=content string; url=url string; category=category string; summary=summary string; keyWords=keyWords string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.ki.String())
		})
	}
}
