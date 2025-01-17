package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	h "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models"
	cf "gitlab.dev.ict/golang/libs/goconfluence"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	ExpBV  = cf.ExpBodyView
	URL_CE = cf.UrlConfExternal
)

// ConfluenceProcessor - confluence processor. It is implementation of <Logic> interface
type ConfluenceProcessor struct {
	*cf.Run
	log         *gologgers.Logger
	isParallel  bool
	rootPageID  string
	IsDebug     bool
	expBodyType string
	psFuncs     []h.ProcessSelectionFN
	seFuncs     []h.SkipElementFN
}

// String
func (cp *ConfluenceProcessor) String() string {
	return fmt.Sprintf("ConfluenceProcessor{isParallel=%t, rootPageId=%s, isDebug=%t}", cp.isParallel, cp.rootPageID, cp.IsDebug)
}

func NewConfluenceProcessor(log *gologgers.Logger, isParallel bool) *ConfluenceProcessor {
	return &ConfluenceProcessor{log: log, isParallel: isParallel}
}

func (cp *ConfluenceProcessor) WithPageID(id string) *ConfluenceProcessor {
	cp.rootPageID = id
	return cp
}

func (cp *ConfluenceProcessor) WithRunObj(run *cf.Run, isDebug bool) *ConfluenceProcessor {
	cp.Run = run
	return cp
}

func (cp *ConfluenceProcessor) WithRun(u, p, baseURL string, to int, isDebug bool) *ConfluenceProcessor {
	cp.Run = cf.NewRunWithHttpCl(u, p, baseURL, to, isDebug, cp.log)
	return cp
}

func (cp *ConfluenceProcessor) WithPSFuncs(psFuncs ...h.ProcessSelectionFN) *ConfluenceProcessor {
	cp.psFuncs = psFuncs
	return cp
}

func (cp *ConfluenceProcessor) WithSEFuncs(seFuncs ...h.SkipElementFN) *ConfluenceProcessor {
	cp.seFuncs = seFuncs
	return cp
}

func (cp *ConfluenceProcessor) Debug(d bool) *ConfluenceProcessor {
	cp.IsDebug = d
	return cp
}

func (cp *ConfluenceProcessor) EBT(t string) *ConfluenceProcessor {
	cp.expBodyType = t
	return cp
}

// Process - process confluence pages.
func (cp *ConfluenceProcessor) Process(ctx context.Context, csf ...models.ContentSaverFunc) {
	log := cp.log.RecWithCtx(ctx, "confl-scrap")
	log.Infof("Start processing confluence pages! Type of body expanding: %s", cp.expBodyType)
	var summary, keywords string
	if !cp.isParallel {
		cp.ProcessPages(ctx, cp.rootPageID, func(p *cf.Page) {
			log.Infof("Process page: ID=%s Type=%s Title=[%s]", p.ID, p.Type, p.Title)
			for _, f := range csf {
				f(ctx, models.NewDoc(p.Title, cp.parseContent(ctx, p.Body.GetBodyStorage().Value), p.Links.Webui).WithSummary(summary).WithKeywords(keywords))
			}
		}, cp.expBodyType)
	} else {
		cp.ProcessPagesInParallel(ctx, cp.rootPageID, func(p *cf.Page, wg *sync.WaitGroup) {
			defer wg.Done()
			log.Infof("[PARALLEL]Process page: ID=%s Type=%s Title=[%s]", p.ID, p.Type, p.Title)
			doc := models.NewDoc(p.Title, cp.parseContent(ctx, p.Body.GetBodyStorage().Value), p.Links.Webui).WithCategory(models.CategoryCONF).WithSummary(summary).WithKeywords(keywords)
			log.Infof("Scrapped doc: %s", utils.Json(doc))
			for _, f := range csf {
				f(ctx, doc)
			}
		}, cp.expBodyType)
	}
}

func (cp *ConfluenceProcessor) parseContent(ctx context.Context, s string) string {
	log := cp.log.RecWithCtx(ctx, "web-scrap")
	log.Debugf("Start parsing page content: %s", s)
	d, _ := goquery.NewDocumentFromReader(strings.NewReader(s))
	selection := d.Selection
	for _, fn := range cp.psFuncs {
		selection = fn(selection)
	}
	parsedHtml := h.ParseWebPage(ctx, selection, cp.IsDebug, cp.seFuncs...)
	parsedHtml = h.RemoveExcessEmptyLines(h.TrimLines(strings.NewReader(parsedHtml)))
	return parsedHtml
}

func (cp *ConfluenceProcessor) Type() models.LogicType {
	return models.LogicTypeConfluence
}

func (cp *ConfluenceProcessor) WithExternalSource(sources ...string) models.Logic {
	return cp.WithPageID(sources[0])
}
