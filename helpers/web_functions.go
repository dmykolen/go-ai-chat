package helpers

import (
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/russross/blackfriday/v2"
	"gitlab.dev.ict/golang/libs/utils"
)

var FuncMap = template.FuncMap{
	"extractQuestion":     extractQuestion,
	"convertToJSON":       convertToJSON,
	"convertToJSONPretty": convertToJSONPretty,
	"convertMdToHTML":     convertMarkdownToHTML,
	"millisToTS":          millisToTS,
	"getCssAsset":         getCssAsset,
}

var questionRegex = regexp.MustCompile(`Question:\s*(.*)`)

func extractQuestion(input string) string {
	match := questionRegex.FindStringSubmatch(input)
	if len(match) > 1 {
		return match[1]
	}
	return input
}

func convertToJSON(input interface{}) string {
	return utils.JsonStr(input)
}

func convertToJSONPretty(input interface{}) string {
	return utils.JsonPrettyStr(input)
}

func convertMarkdownToHTML(input string) template.HTML {
	return template.HTML(blackfriday.Run([]byte(input)))
}

func millisToTS(ms int64) string {
	return time.UnixMilli(ms).Format(utils.TS_FMT1)
}

func getCssAsset(name string) (res template.HTML) {
	filepath.Walk("public/assets", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == name {
			res = template.HTML("<link rel=\"stylesheet\" href=\"/" + path + "\">")
		}
		return nil
	})
	return
}
