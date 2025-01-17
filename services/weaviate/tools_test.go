package wvservice

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
)

var (
	sr1 = &SearchRequest{
		Fields: []Field{FieldTitle, FieldUrl, FieldCategory},
		Limit:  3,
		Sort: struct {
			Field     string "json:\"field\""
			SortOrder string "json:\"sortOrder\""
		}{Field: "title", SortOrder: "desc"},
		Where: struct {
			Field    string "json:\"field\""
			Operator string "json:\"operator\""
			Values   string "json:\"value\""
		}{Field: string(FieldCategory), Operator: string(filters.Equal), Values: "FRD"},
	}

	sr2 = &SearchRequest{
		Fields:       []Field{FieldTitle, FieldUrl, FieldAdditional1},
		Limit:        3,
		SearchText:   "Чому помилка 409?",
		SearchFields: []string{"content"},
		Sort: struct {
			Field     string "json:\"field\""
			SortOrder string "json:\"sortOrder\""
		}{Field: "title", SortOrder: "asc"},
		Where: struct {
			Field    string "json:\"field\""
			Operator string "json:\"operator\""
			Values   string "json:\"value\""
		}{Field: "category", Operator: string(filters.Equal), Values: "FRD"},
	}

	sr3 = &SearchRequest{
		Fields:       []Field{FieldTitle, FieldUrl, FieldAdditional1},
		Limit:        3,
		SearchText:   "Чому помилка 409?",
		SearchFields: []string{"content", "title"},
		Where: struct {
			Field    string "json:\"field\""
			Operator string "json:\"operator\""
			Values   string "json:\"value\""
		}{Field: "category", Operator: string(filters.Equal), Values: "FRD"},
	}

	input = `{
    "certainty": null,
    "creationTimeUnix": "1706707535654",
    "distance": "0.95",
    "explainScore": "(bm25)\n(Result Set keyword) Document 657f6564-4fc4-464a-bc4b-3a8982bf9d7f contributed 0.0023584905660377358 to the score\n(Result Set vector) Document 657f6564-4fc4-464a-bc4b-3a8982bf9d7f contributed 0.012096774193548387 to the score",
    "id": "657f6564-4fc4-464a-bc4b-3a8982bf9d7f",
    "lastUpdateTimeUnix": "1706707535654",
    "score": "0.014455264"}`
)

func Test_countTokens(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"1", string(lo.Must(os.ReadFile("1.txt"))), 1},
		{"2", string(lo.Must(os.ReadFile("2.txt"))), 2},
		{"3", " random color             \n\n\n \n\n\n \n \n \n ", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countTokens(tt.content)
		})
	}
	r := ai.AskAI(ctx, " random color             \n\n\n \n\n\n \n \n \n ", goopenai.NewChat())
	t.Log("r:", r)
}
func TestWeaviateSearch(t *testing.T) {
	so := &SearchOptions{
		SearchText:   "YourSearchText",
		SearchFields: []string{"content"},
		FieldsReturn: []Field{FieldTitle},
		LimitItems:   10, // Replace with the actual limit
	}

	// Call the function being tested
	resp, err := WeaviateSearch(logTestKB.WithCtx(utils.GenerateCtxWithRid()), client, DefaultClassKB, so)
	assert.NoError(t, err)

	ki := GQLRespConvert[KnowledgeItem](resp, DefaultClassKB)
	t.Log(ki)

	kil := KnowledgeItems(ki)
	kil.FindDuplicates()
}

func TestWeaviateSearch2(t *testing.T) {
	record := logTestKB.WithCtx(utils.GenerateCtxWithRid())
	tests := []struct {
		name string
		so   *SearchOptions
	}{
		// {"test1", DefaultSO.SearchTxt("Чому помилка 409?").AddFields(FieldTitle, FieldUrl, FieldAdditional2).AddLimit(1).SF("content")},
		// {"test2", DefaultSO.SearchTxt("Що таке PBX?").AddFields(FieldTitle, FieldUrl, FieldAdditional1).AddLimit(1).SF("content")},
		{"test2", DefaultSO().SearchTxt("бібліотека golang для PostgresDB").SetFields(FieldTitle, FieldUrl, FieldAdditional1).Limit(1).SF("content")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := WeaviateSearch(record, client, DefaultClassKB, tt.so)
			assert.NoError(t, err)

			ki := GQLRespConvert[KnowledgeItem](resp, DefaultClassKB)
			assert.NotEmpty(t, ki)
			t.Log(ki)

			kis := KnowledgeItems(ki)
			t.Log(kis.Len())
			t.Log("SCORE:", ki[0].Additional.Score())

		})
	}
}

func Test_ToSearchOptions(t *testing.T) {
	record := logTestKB.WithCtx(utils.GenerateCtxWithRid())
	tests := []struct {
		name string
		sr   *SearchRequest
	}{
		// {"test1", sr1},
		{"test3", sr3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(utils.JsonPrettyStr(tt.sr))
			so := tt.sr.ToSearchOptions()
			t.Logf("so: %#v", so)

			resp, err := WeaviateSearch(record, client, DefaultClassKB, so)
			assert.NoError(t, err)

			ki := GQLRespConvert[KnowledgeItem](resp, DefaultClassKB)
			assert.NotEmpty(t, ki)
			t.Log(utils.JsonPrettyStr(ki))

		})
	}
}

func TestConv(t *testing.T) {
	var additional AdditionalMap
	json.Unmarshal([]byte(input), &additional)
	t.Logf("Additional: %#v", additional)
	t.Log("#########################")
	t.Log("CreationTimeUnix:", additional.CreationTime())
	t.Log("CreationTimeUnix:", additional.CreationTime().Year())
}

func TestXxx(t *testing.T) {
	so := NewSO().Limit(300).Fields(FieldTitle, FieldUrl, FieldAdditional1).SearchTxt("Чому помилка 409?").SF("content")
	t.Log(so)
}
