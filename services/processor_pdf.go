package services

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"gitlab.dev.ict/golang/go-ai/helpers/pdf"
	"gitlab.dev.ict/golang/go-ai/models"
	gl "gitlab.dev.ict/golang/libs/gologgers"
)

type PDFPprocessor struct {
	log        *gl.Logger
	filePaths  []string
	fileUrls   []string
	httpClient *resty.Client
}

func NewPDFPprocessor(log *gl.Logger, httpClient *resty.Client) *PDFPprocessor {
	return &PDFPprocessor{log: log, httpClient: httpClient}
}

func (pp *PDFPprocessor) WithFilePaths(paths ...string) *PDFPprocessor {
	pp.filePaths = append(pp.filePaths, paths...)
	return pp
}

func (pp *PDFPprocessor) WithFileUrls(urls ...string) *PDFPprocessor {
	pp.fileUrls = append(pp.fileUrls, urls...)
	return pp
}

func (pp *PDFPprocessor) Process(ctx context.Context, csf ...models.ContentSaverFunc) {

	var docs []*models.Doc

	pdfloader := func(filepath string) {
		c, t, e := pdf.PDFLoad(filepath)
		if e != nil {
			pp.log.Errorf("Error while loading PDF file %s. Error: %v", filepath, e)
		}
		d := models.NewDoc(t, c, filepath).WithErrorLoading(e)
		pp.log.Debugf("Process file - %s. IsContentExists=%t; IsErrorLoading=%t", filepath, d.TextContent != "", d.IsErrorLoading())
		docs = append(docs, d)
	}

	for _, v := range pp.filePaths {
		pdfloader(v)
	}
	for _, v := range pp.fileUrls {
		randomName := fmt.Sprintf("./%d.pdf", time.Now().UnixNano())
		pp.httpClient.R().SetContext(ctx).SetOutput(randomName).Get(v)
		pdfloader(randomName)
	}

	pp.log.Infof("Total number of PDF files was processed: %d", len(docs))

	for _, dd := range docs {
		if dd.IsErrorLoading() {
			continue
		}
		for _, f := range csf {
			f(ctx, dd)
		}
	}
}

func (wpp *PDFPprocessor) Type() models.LogicType {
	return models.LogicTypePDF
}