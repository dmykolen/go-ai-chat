package services

import (
	"context"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	h "gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gonet"
)

var Links_LifecellUA = []string{
	"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/gsm-shliuz/",
	"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/ip-telefoniia/",
	"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/mobilnii-marketing/",
	"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/virtualna-ats/",
	"https://lifecell.ua/uk/malii-biznes-lifecell/servisi/yedina-merezha-fmc/",
	"https://lifecell.ua/uk/velikii-biznes-lifecell/servisi/fiksovanij-zvyazok/",
	"https://lifecell.ua/uk/pidtrimka/pitannya-v-kategoriyi/?category=121",
}

type WebPagesProcessor struct {
	log        *gl.Logger
	WebURLs    []string
	psFuncs    []h.ProcessSelectionFN
	seFuncs    []h.SkipElementFN
	httpClient *resty.Client
	IsDebug    bool
	typeLogic  models.LogicType
}

var processSelectionFNLifecellUa = func(s *goquery.Selection) *goquery.Selection {
	// return s.Find("div.page-content > div").Children().Not(".iot_widget").Not(".breadcrumb")
	ss := s.Find("div.page-content > div")
	ss.Find(".iot_widget").Remove()
	ss.Find(".breadcrumb").Remove()
	return ss
}

func NewWebPagesProcessor(log *gl.Logger, urls []string, psFuncs []h.ProcessSelectionFN, seFuncs []h.SkipElementFN) *WebPagesProcessor {
	return &WebPagesProcessor{log: log, WebURLs: urls, psFuncs: psFuncs, seFuncs: seFuncs, httpClient: gonet.NewRestyClient(log)}
}

func NewWebPagesProcessorWithDefaultSkipElems(log *gl.Logger, urls []string, psFuncs []h.ProcessSelectionFN) *WebPagesProcessor {
	return NewWebPagesProcessor(log, urls, psFuncs, []h.SkipElementFN{h.SkipElemWithClass("check-answer"), h.SkipElemWithNodeNames("form")})
}

func (wpp *WebPagesProcessor) Debug(d bool) *WebPagesProcessor {
	wpp.IsDebug = d
	return wpp
}

func (wpp *WebPagesProcessor) WithWebURLs(urls []string) *WebPagesProcessor {
	wpp.WebURLs = urls
	return wpp
}

func (wpp WebPagesProcessor) Process(ctx context.Context, csf ...models.ContentSaverFunc) {
	log := wpp.log.RecWithCtx(ctx, "web-scrap")
	log.WithData(gl.M{"isDebug": wpp.IsDebug}).Infof("Start scrap web pages")
	for _, v := range wpp.WebURLs {
		pageParsed := h.ScrapAndParse(log, wpp.httpClient.SetDisableWarn(true), v, wpp.IsDebug, wpp.psFuncs, wpp.seFuncs...)
		log.Infof("pageParsed: %s", pageParsed)
		if pageParsed.Err != nil {
			log.Errorf("Error while process page [%s] >> %s", v, pageParsed.Err)
			continue
		}
		pageParsed.TextContent = h.CleanEmptyLine(pageParsed.TextContent)
		log.Infof("Process page [%s] >> %s", v, pageParsed)
		for _, f := range csf {
			f(ctx, models.NewDoc(pageParsed.Title, pageParsed.TextContent, v).WithCategory(models.CategoryWEB).WithOriginal(pageParsed.OriginalPage))
		}
	}
}

func (wpp *WebPagesProcessor) Type() models.LogicType {
	return wpp.typeLogic
}

func (wpp *WebPagesProcessor) WithExternalSource(sources ...string) models.Logic {
	return wpp.WithWebURLs(sources)
}

func (wpp *WebPagesProcessor) WithType(t models.LogicType) *WebPagesProcessor {
	wpp.typeLogic = t
	return wpp
}

func (wpp *WebPagesProcessor) WithSelectionProcess(functions ...h.ProcessSelectionFN) *WebPagesProcessor {
	wpp.psFuncs = functions
	return wpp
}

func (wpp *WebPagesProcessor) WithSkipElems(functions ...h.SkipElementFN) *WebPagesProcessor {
	wpp.seFuncs = functions
	return wpp
}

func (wpp *WebPagesProcessor) WithHttpCl(client *resty.Client) *WebPagesProcessor {
	wpp.httpClient = client
	return wpp
}

func NewWPP(log *gl.Logger) *WebPagesProcessor {
	return &WebPagesProcessor{log: log}
}

func NewWPPLifecellUA(log *gl.Logger) *WebPagesProcessor {
	return NewWPP(log).
		Debug(false).
		WithType(models.LogicTypeWebLifecell).
		WithHttpCl(gonet.NewRestyClient(log)).
		WithSelectionProcess(h.FindBodyAndExcludeScript, h.ExcludeScripts, processSelectionFNLifecellUa).
		WithSkipElems(h.SkipElemWithClass("check-answer"), h.SkipElemWithNodeNames("form"))
}

func NewWPPOther(log *gl.Logger) *WebPagesProcessor {
	return NewWPP(log).
		Debug(false).
		WithType(models.LogicTypeWebOther).
		WithHttpCl(gonet.NewRestyClient(log)).
		WithSelectionProcess(h.FindBodyAndExcludeScript, h.ExcludeScripts)
}
