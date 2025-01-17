package wvservice

import (
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)

type SearchOptions struct {
	FieldsReturn   []Field           `json:"fields"`       // The fields to return
	SearchText     string            `json:"searchText"`   // The text to search
	SearchFields   []string          `json:"searchFields"` // The fields to search in
	SortBy         []graphql.Sort    `json:"sort"`         // The sort options
	filterWhere    []FilterWhereFunc // The filter where functions
	LimitItems     int               `json:"limit"` // The number of records to return
	whereCondition *filters.WhereBuilder
}

func NewSO() *SearchOptions {
	return &SearchOptions{}
}

/*
SearchRequest
  - `Fields` this is a list of fields that the user wants to return.
  - `SearchFields` this is a list of fields that the user wants to search on.
  - `SearchText` this is the text that the user wants to search for.
  - `SortFields` this is a list of fields that the user wants to sort on.
  - > SortDirection` this is the direction that the user wants to sort the field on.This can be `Ascending` or`Descending`.
  - `Limit` this is the number of records that the user wants to return.
  - `Filter` this is a list of filter functions that the user wants to apply to the data.
  - > Filter:Field - this is the field that the user wants to filter on.
  - > Filter:Operator - this is the operator that the user wants to use to filter the data. This can be `Equals`, `NotEquals`, `GreaterThan`, `LessThan`, `GreaterThanOrEquals`, `LessThanOrEquals`, `Contains`, `NotContains`, `StartsWith`, `EndsWith`.
  - > Filter:Value - this is the value that the user wants to filter on.
*/
type SearchRequest struct {
	Fields       []Field  `json:"fields"`       // The fields to return
	SearchText   string   `json:"searchText"`   // The text to search
	SearchFields []string `json:"searchFields"` // The fields to search in
	Sort         struct {
		Field     string `json:"field"`
		SortOrder string `json:"sortOrder"`
	} `json:"sort"` // The sort options
	Limit int `json:"limit"` // The number of records to return
	Where struct {
		Field    string `json:"field"`
		Operator string `json:"operator"`
		Values   string `json:"value"`
	} `json:"where"` // The filter where functions
}

func (sr *SearchRequest) ToSearchOptions() *SearchOptions {
	so := NewSO().
		SetFields(sr.Fields...).
		SearchTxt(sr.SearchText).
		SF(sr.SearchFields...).
		Limit(sr.Limit).
		Where(sr.Where.Field, sr.Where.Values, sr.Where.Operator)
	if sr.Sort.Field != "" {
		so.Sort(sr.Sort.Field, sr.Sort.SortOrder)
	}
	return so
}

func sortT() graphql.Sort {
	return graphql.Sort{
		Path:  []string{"title"},
		Order: "asc",
	}
}

func DefaultSO() *SearchOptions {
	return &SearchOptions{
		// Fields: []Field{FieldTitle, FieldCategory, FieldChunkNo, FieldContent, FieldUrl, FieldKeywords, FieldSummary},
		FieldsReturn: []Field{FieldTitle, FieldCategory, FieldChunkNo, FieldUrl},
		LimitItems:   3,
	}
}

func (so *SearchOptions) FilterWhere(f FilterWhereFunc) *SearchOptions {
	so.filterWhere = append(so.filterWhere, f)
	return so
}

func (so *SearchOptions) SetFields(fields ...Field) *SearchOptions {
	so.FieldsReturn = fields
	return so
}

func (so *SearchOptions) Fields(fields ...Field) *SearchOptions {
	so.FieldsReturn = append(so.FieldsReturn, fields...)
	return so
}

func (so *SearchOptions) SearchTxt(q string) *SearchOptions {
	so.SearchText = q
	return so
}

func (so *SearchOptions) SF(fields ...string) *SearchOptions {
	so.SearchFields = fields
	return so
}

func (so *SearchOptions) Sort(f, o string) *SearchOptions {
	so.SortBy = SortGQL(Sort(f, o))
	return so
}
func (so *SearchOptions) SortOrder(f Field, isDesc bool) *SearchOptions {
	so.SortBy = append(so.SortBy, SortBy(f, isDesc)...)
	return so
}

func (so *SearchOptions) Where(f, value, op string) *SearchOptions {
	so.whereCondition = filters.Where().WithPath([]string{f}).WithOperator(filters.WhereOperator(op)).WithValueText(value)
	return so
}

func (so *SearchOptions) Limit(limit int) *SearchOptions {
	so.LimitItems = limit
	return so
}

func (so *SearchOptions) GetFields() []graphql.Field {
	return fieldsList(so.FieldsReturn...)
}

func (so *SearchOptions) GetSort() []graphql.Sort {
	return so.SortBy
}

func (so *SearchOptions) GetFilterWhere() *filters.WhereBuilder {
	var where *filters.WhereBuilder
	for _, f := range so.filterWhere {
		return f("")
	}
	return where
}
