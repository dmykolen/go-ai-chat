package helpers

import (
	"bytes"
	"fmt"
	"html/template"
	"sync"
)

var tmpl *template.Template

func init() {
	// InitTemplatesGlob("../assets/prompt_templates/*.tmpl")
}

type DocumentsData struct {
	Name      string `json:"name"`
	Content   string `json:"data"`
	Reference string `json:"reference"`
}

type PromptData struct {
	Documents    []DocumentsData `json:"documents"`
	UserQuestion string          `json:"question"`
}

type PromptWithDocs struct {
	Documents    []any  `json:"documents"`
	UserQuestion string `json:"question"`
}

func NewPromptData(question string, documents ...DocumentsData) *PromptData {
	return &PromptData{documents, question}
}

func InitTemplates(f string) {
	tmpl = template.Must(template.New("template").Option("missingkey=zero").ParseFiles(f))
	for _, t := range tmpl.Templates() {
		fmt.Println("t: ", t.Name())
	}
}

func InitTemplatesGlob(f string) {
	tmpl = template.Must(template.New("template").Option("missingkey=zero").ParseGlob(f))
	for _, t := range tmpl.Templates() {
		fmt.Println("t: ", t.Name())
	}
}

func EvalPromptSimple(templateName, question string, data ...any) (string, error) {
	var buffer bytes.Buffer
	err := tmpl.ExecuteTemplate(&buffer, templateName, &PromptWithDocs{data, question})
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func EvalPrompt(tmpl *template.Template, templateName string, data any) (string, error) {
	var buffer bytes.Buffer
	err := tmpl.ExecuteTemplate(&buffer, templateName, data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

type TemplateData struct {
	sync.Mutex
	templates       *template.Template
	filePathToTmpls string
}

func NewTemplateData(filePathToTmpls string) *TemplateData {
	return &TemplateData{filePathToTmpls: filePathToTmpls}
}

func (t *TemplateData) ParseGlob(pattern string) *TemplateData {
	t.Lock()
	defer t.Unlock()

	t.templates = template.Must(template.New("template").Option("missingkey=zero").ParseGlob(pattern))
	return t
}

func (t *TemplateData) ParseFiles(files ...string) *TemplateData {
	t.Lock()
	defer t.Unlock()

	if files == nil {
		files = []string{t.filePathToTmpls}
	}

	t.templates = template.Must(template.New("template").Option("missingkey=zero").ParseFiles(files...))
	return t
}

func (t *TemplateData) GetTemplate(name string) *template.Template {
	t.Lock()
	defer t.Unlock()

	return t.templates.Lookup(name)
}

func (t *TemplateData) ExecuteTemplate(name string, data any) (string, error) {
	var buffer bytes.Buffer
	err := t.templates.ExecuteTemplate(&buffer, name, data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (t *TemplateData) ExecuteTemplateSafe(name string, data any) string {
	result, _ := t.ExecuteTemplate(name, data)
	return result
}

func (t *TemplateData) CountTempls() int {
	return len(t.templates.Templates())
}
