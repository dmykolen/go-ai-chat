package models

import (
	"encoding/json"

	"gitlab.dev.ict/golang/libs/utils"
)

// DocumentAttribute represents document attribute keys
type DocumentAttribute string

const (
	AttrSummary  DocumentAttribute = "summary"
	AttrKeywords DocumentAttribute = "keywords"
	AttrCategory DocumentAttribute = "category"
	AttrOriginal DocumentAttribute = "original"

	CategoryFRD  = "FRD"
	CategoryWEB  = "WEB"
	CategoryCONF = "confluence"
)

// Attributes represents a collection of document attributes
type Attributes map[DocumentAttribute]any

// NewAttributes creates a new Attributes instance
func NewAttributes() Attributes {
	return make(Attributes)
}

// Set sets an attribute value
func (a Attributes) Set(key DocumentAttribute, value any) {
	a[key] = value
}

// Get retrieves an attribute value
func (a Attributes) Get(key DocumentAttribute) (any, bool) {
	val, ok := a[key]
	return val, ok
}

// MarshalJSON implements json.Marshaler
func (a Attributes) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	for k, v := range a {
		m[string(k)] = v
	}
	return json.Marshal(m)
}

type Doc struct {
	Title       string
	TextContent string
	Link        string
	Attrs       map[DocumentAttribute]any
	errLoading  error
}

// String returns a JSON string representation of the document
func (d *Doc) String() string {
	return utils.JsonStr(d)
}

func NewDoc(title, textContent, link string) *Doc {
	return &Doc{Title: title, TextContent: textContent, Link: link, Attrs: NewAttributes()}
}

func (d *Doc) WithAttr(key DocumentAttribute, val any) *Doc {
	d.Attrs[key] = val
	return d
}

func (d *Doc) WithErrorLoading(err error) *Doc {
	d.errLoading = err
	return d
}

func (d *Doc) IsErrorLoading() bool {
	return d.errLoading != nil
}

func (d *Doc) ErrorLoading() error {
	return d.errLoading
}

func (d *Doc) WithSummary(s string) *Doc {
	return d.WithAttr(AttrSummary, s)
}

func (d *Doc) Summary() string {
	if s, ok := d.Attrs[AttrSummary].(string); ok {
		return s
	}
	return ""
}

func (d *Doc) WithKeywords(s string) *Doc {
	return d.WithAttr(AttrKeywords, s)
}

func (d *Doc) Keywords() string {
	if s, ok := d.Attrs[AttrKeywords].(string); ok {
		return s
	}
	return ""
}

func (d *Doc) WithCategory(s string) *Doc {
	return d.WithAttr(AttrCategory, s)
}

func (d *Doc) Category() string {
	if s, ok := d.Attrs[AttrCategory].(string); ok {
		return s
	}
	return ""
}

func (d *Doc) WithOriginal(s string) *Doc {
	return d.WithAttr(AttrOriginal, s)
}

func (d *Doc) Original() string {
	if s, ok := d.Attrs[AttrOriginal].(string); ok {
		return s
	}
	return ""
}
