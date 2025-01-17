package wvservice

import (
	"fmt"
	"time"

	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/slog"
	"github.com/mitchellh/mapstructure"
	tokenizer "github.com/samber/go-gpt-3-encoder"
	"github.com/samber/lo"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	Additioanl1 = "_additional {id creationTimeUnix score explainScore certainty distance}"
	Additioanl2 = "_additional {id creationTimeUnix lastUpdateTimeUnix}"
	Additioanl3 = "_additional {id}"
)

const (
	FieldTitle       Field = "title"
	FieldCategory    Field = "category"
	FieldChunkNo     Field = "chunkNo"
	FieldContent     Field = "content"
	FieldUrl         Field = "url"
	FieldKeywords    Field = "keywords"
	FieldSummary     Field = "summary"
	FieldAdditional1 Field = Additioanl1
	FieldAdditional2 Field = Additioanl2
	FieldAdditional3 Field = Additioanl3
)

type Field string

func (fi Field) String() string        { return string(fi) }
func (fi Field) Gf() graphql.Field     { return graphql.Field{Name: fi.String()} }
func f(fieldName string) graphql.Field { return graphql.Field{Name: fieldName} }

type FilterWhereFunc func(s string) *filters.WhereBuilder

func WhereTitleLike(s string) FilterWhereFunc {
	return func(s string) *filters.WhereBuilder { return FilterWhereTitleLike(s) }
}

var FilterWhereCategory = func(s string) *filters.WhereBuilder { return filterWhere(FieldCategory.String(), s, filters.Equal) }
var FilterWhereTitleLike = func(s string) *filters.WhereBuilder { return filterWhere(FieldTitle.String(), s, filters.Like) }
var FilterWhereTitleEQ = func(s string) *filters.WhereBuilder { return filterWhere(FieldTitle.String(), s, filters.Equal) }
var filterWhere = func(f, value string, op filters.WhereOperator) *filters.WhereBuilder {
	return filters.Where().WithPath([]string{f}).WithOperator(op).WithValueText(value)
}
var Where = func(f, value, op string) *filters.WhereBuilder {
	return filters.Where().WithPath([]string{f}).WithOperator(filters.WhereOperator(op)).WithValueText(value)
}

var SortBy = func(field Field, isDesc bool) []graphql.Sort {
	return []graphql.Sort{{Path: []string{field.String()}, Order: lo.Ternary(isDesc, graphql.Desc, graphql.Asc)}}
}

var SortByMultiple = func(isDesc bool, fields ...Field) (arr []graphql.Sort) {
	for _, v := range fields {
		arr = append(arr, graphql.Sort{Path: []string{v.String()}, Order: lo.Ternary(isDesc, graphql.Desc, graphql.Asc)})
	}
	return
}

func Sort(field string, order ...string) graphql.Sort {
	if len(order) == 0 {
		return graphql.Sort{Path: []string{field}, Order: graphql.Asc}
	}
	return graphql.Sort{Path: []string{field}, Order: graphql.SortOrder(order[0])}
}

func SortGQL(gql ...graphql.Sort) (arr []graphql.Sort) {
	arr = append(arr, gql...)
	return
}

func SortBy3() []graphql.Sort {
	return []graphql.Sort{
		{Path: []string{"title"}, Order: graphql.Asc},
		{Path: []string{"chunkNo"}, Order: graphql.Asc},
	}
}

func fieldsList(fields ...Field) (gf []graphql.Field) {
	for _, v := range fields {
		gf = append(gf, v.Gf())
	}
	return
}

func fieldsListStr(fields ...Field) (gf []string) {
	for _, v := range fields {
		gf = append(gf, v.String())
	}
	return
}

func ObjContent(obj *models.Object) string {
	return strutil.MustString(obj.Properties.(map[string]interface{})["content"])
}

func IdOfFirstEl(list []*KnowledgeItem) string {
	if len(list) > 0 {
		return list[0].ID()
	}
	return ""
}

func ParseAddResponse(log *slog.Record, resp []models.ObjectsGetResponse, err error) error {
	if err != nil {
		log.Errorf("Weaviate error response: %v", err)
		return err
	}
	log.Infof("Got %d objects", len(resp))
	return nil
}

func help_object_to_str(obj *models.Object) string {
	var i KnowledgeItem
	lo.Must0(mapstructure.Decode(obj.Properties, &i))
	return fmt.Sprintf("[id=%v; Created=%s; Upd=%s; Class=%s; Additional=%v; V=%v...; Props=%s]", obj.ID, time2str(obj.CreationTimeUnix), time2str(obj.LastUpdateTimeUnix), obj.Class, obj.Additional, obj.Vector[:mathutil.Min(len(obj.Vector), 5)], &i)
}

func time2str(timeUnix int64) string { return time.UnixMilli(timeUnix).Format(time.DateTime) }

func countTokens(content string) int {
	return len(lo.Must(lo.Must(tokenizer.NewEncoder()).Encode(content)))
}

func FindDuplicates(objarrr []*models.Object) (dupls []*models.Object) {
	var unique []string
	for _, obj := range objarrr {
		t := obj.Properties.(map[string]interface{})["title"].(string)
		if strutil.InArray(t, unique) {
			dupls = append(dupls, obj)
		} else {
			unique = append(unique, t)
		}
	}
	return
}

func WeaviateSearch(log *slog.Record, w *weaviate.Client, className string, so *SearchOptions) (*models.GraphQLResponse, error) {
	if so == nil {
		so = DefaultSO()
	}

	if so.LimitItems == 0 {
		so.LimitItems = 10
	}

	if so.FieldsReturn == nil {
		so.FieldsReturn = []Field{FieldTitle}
	}

	builder := w.GraphQL().Get().WithClassName(className).
		WithFields(so.GetFields()...).
		WithLimit(so.LimitItems)

	if so.SearchText != "" {
		h := w.GraphQL().HybridArgumentBuilder().
			WithQuery(so.SearchText).
			WithFusionType(graphql.Ranked).
			WithProperties(so.SearchFields)
		builder = builder.WithHybrid(h)
	}

	if len(so.SortBy) > 0 {
		builder = builder.WithSort(so.GetSort()...)
	}

	if so.whereCondition != nil {
		builder = builder.WithWhere(so.whereCondition)
	}

	r, e := builder.Do(log.Ctx)

	if e != nil {
		log.Errorf("Error searching in Weaviate: %v", e)
		return nil, e
	}
	if r.Errors != nil {
		for _, e := range r.Errors {
			log.Errorf("error in Path=%v ERR_message: %s", e.Path, e.Message)
		}
		return nil, errorx.Rawf("WW errors: %v", r.Errors)
	}

	log.Tracef("VectorDB documents RESPONSE: %s", utils.Json(r))
	return r, nil
}
