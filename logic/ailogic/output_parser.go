package ailogic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/schema"
)

const _structuredLineTemplate = "\"%s\": // %s\n"
const _structuredFormatInstructionTemplate = "The output should be a JSON object following this schema: \n{\n%s}" // nolint

type OutputParserJSON[T any] struct {
	outputparser.Structured
	prefixFormatInstruction string
}

var _ schema.OutputParser[any] = OutputParserJSON[any]{}

func NewOutputParserJSONSimple[T any]() OutputParserJSON[T] {
	return OutputParserJSON[T]{}
}
func NewOutputParserJSON[T any](schema []outputparser.ResponseSchema) OutputParserJSON[T] {
	return OutputParserJSON[T]{Structured: outputparser.NewStructured(schema)}
}

func (p OutputParserJSON[T]) Parse(text string) (any, error) {
	return p.parse(text)
}

func (p OutputParserJSON[T]) ParseWithPrompt(text string, prompt llms.PromptValue) (any, error) {
	return p.parse(text)
}

func (p OutputParserJSON[T]) GetFormatInstructions() string {
	jsonLines := ""
	for _, rs := range p.ResponseSchemas {
		jsonLines += "\t" + fmt.Sprintf(_structuredLineTemplate, rs.Name, rs.Description)
	}
	return fmt.Sprintf(_structuredFormatInstructionTemplate, jsonLines)
}

func (p OutputParserJSON[T]) parse(text string) (parsed map[string]T, err error) {
	var indexEval = func(arr []string) int { return lo.Ternary(len(arr) > 1, 1, 0) }

	// Remove the ```json that can be at the start of the text, and the ```that should be at the end of the text.
	withoutJSONStart := strings.Split(text, "```json")
	jsonString := withoutJSONStart[indexEval(withoutJSONStart)]

	withoutJSONEnd := strings.Split(jsonString, "```")
	jsonString = withoutJSONEnd[indexEval(withoutJSONEnd)]

	// var parsed map[string]string
	err = json.Unmarshal([]byte(jsonString), &parsed)
	if err != nil {
		return
	}

	if len(p.ResponseSchemas) == 0 {
		return
	}

	// Validate that the parsed map contains all fields specified in the response schemas.
	missingKeys := make([]string, 0)
	for _, rs := range p.ResponseSchemas {
		if _, ok := parsed[rs.Name]; !ok {
			missingKeys = append(missingKeys, rs.Name)
		}
	}

	if len(missingKeys) > 0 {
		err = outputparser.ParseError{Text: text, Reason: fmt.Sprintf("output is missing the following fields %v", missingKeys)}
	}
	return
}
