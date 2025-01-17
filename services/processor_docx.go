package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dmykolen/docx2txt"
	"github.com/go-resty/resty/v2"
	"github.com/samber/lo"
	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models"
	gl "gitlab.dev.ict/golang/libs/gologgers"
)

const (
	TblStyleCSV    = "csv"
	TblStyleMD     = "md"
	TblStylePretty = "pretty"
)

type DocxPprocessor struct {
	log        *gl.Logger
	tblStyle   string
	filePaths  []string
	fileUrls   []string
	httpClient *resty.Client
}

func (dp *DocxPprocessor) String() string {
	return fmt.Sprintf("DocxPprocessor={tblStyle=%s, filePaths=%v, fileUrls=%v}", dp.tblStyle, strings.Join(dp.filePaths, ";"), dp.fileUrls)
}

func NewDocxPprocessor(log *gl.Logger, httpClient *resty.Client) *DocxPprocessor {
	return &DocxPprocessor{log: log, httpClient: httpClient}
}

func (dp *DocxPprocessor) WithFilePaths(paths ...string) *DocxPprocessor {
	// dp.filePaths = append(dp.filePaths, paths...)
	dp.filePaths = paths
	return dp
}

func (dp *DocxPprocessor) WithFileUrls(urls ...string) *DocxPprocessor {
	dp.fileUrls = append(dp.fileUrls, urls...)
	return dp
}

func (dp *DocxPprocessor) WithTableStyle(style string) *DocxPprocessor {
	dp.tblStyle = style
	return dp
}

func (dp *DocxPprocessor) Process(ctx context.Context, csf ...models.ContentSaverFunc) {
	log := dp.log.RecWithCtx(ctx, "docx-scrap")
	log.Infof("Start DocxPprocessor! %s", dp)
	var docs []*models.Doc

	docxloader := func(filepath string) {
		if dp.tblStyle == "" {
			dp.tblStyle = "csv"
		}
		c, t, e := dp.docxLoad(filepath)
		if e != nil {
			log.Errorf("Error while loading DOCX file %s. Error: %v", filepath, e)
		} else {
			c = helpers.RemoveExcessEmptyLines(helpers.TrimLines(strings.NewReader(c)))
		}
		d := models.NewDoc(t, c, filepath).WithCategory(models.CategoryFRD).WithErrorLoading(e)
		log.Infof("Process file - %s. IsContentExists=%t; IsErrorLoading=%t", filepath, d.TextContent != "", d.IsErrorLoading())
		docs = append(docs, d)
	}

	for _, v := range dp.filePaths {
		docxloader(v)
	}
	for _, v := range dp.fileUrls {
		randomName := fmt.Sprintf("./%d.docx", time.Now().UnixNano())
		dp.httpClient.R().SetContext(ctx).SetOutput(randomName).Get(v)
		docxloader(randomName)
	}

	log.Infof("Total number of DOCX files was processed: %d", len(docs))

	for _, dd := range docs {
		if dd.IsErrorLoading() {
			continue
		}
		for _, f := range csf {
			f(ctx, dd)
		}
	}
}

func (dp *DocxPprocessor) docxLoad(filePath string) (string, string, error) {
	st := lo.TernaryF(dp.tblStyle != "", func() string { return dp.tblStyle }, func() string { return TblStyleCSV })
	c, e := docx2txt.Docx2txt(filePath, false, docx2txt.StyleTbls(st), docx2txt.WithDebug(false), docx2txt.WithLogger(dp.log))
	if e != nil {
		return "", "", e
	}
	t := filepath.Base(filePath)
	return c.String(), t, nil
}

func (dp *DocxPprocessor) Type() models.LogicType {
	return models.LogicTypeDocx
}

func (dp *DocxPprocessor) WithExternalSource(sources ...string) models.Logic {
	return dp.WithFilePaths(sources...)
}
