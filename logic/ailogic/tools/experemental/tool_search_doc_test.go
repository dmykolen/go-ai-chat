package agents

import (
	"encoding/json"
	"testing"

	"gitlab.dev.ict/golang/libs/gologgers"
)

func TestNewVectorDBSearchTool(t *testing.T) {
	toolSearchVDB := NewVectorDBSearchTool(gologgers.Defult())
	b, _ := json.Marshal(toolSearchVDB.GetTool().Function.Parameters)
	t.Logf("%s", b)
}
