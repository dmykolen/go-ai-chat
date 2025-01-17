package services

import (
	"os"

	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	dirFrdVoip = "../assets/voip_ritm_docs/"
)

var (
	log      = gl.New(gl.WithChannel("SERVICES-TEST"), gl.WithLevel(gl.LevelInfo), gl.WithOC(true))
	logDebug = gl.New(gl.WithChannel("SERVICES-TEST"), gl.WithLevel(gl.LevelDebug), gl.WithOC(true))
	ctx      = utils.GenerateCtxWithRid()
	dir, _   = os.MkdirTemp("", "TestNewWebPagesProcessor_lifecellUA_*")

	docxFiles []string
	dirParsed string
)
