package handlers

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
	"github.com/valyala/fasthttp"
	help "gitlab.dev.ict/golang/go-ai/helpers"
	m "gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/go-ai/services"
	wvservice "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/utils"
)

var searchOptsGetAllDocs = wvservice.DefaultSO().Limit(200).Fields(wvservice.FieldContent, wvservice.FieldAdditional2).SortOrder(wvservice.FieldTitle, false)

type VectorDBHandler struct {
	app     *fiber.App
	log     *gologgers.Logger
	rag     *services.RAGService
	isDebug bool
}

func NewVectorDBHandler(a *fiber.App, l *gologgers.Logger, r *services.RAGService, debug bool) *VectorDBHandler {
	r.Info()
	return &VectorDBHandler{app: a, log: l, rag: r, isDebug: debug}
}

func (h *VectorDBHandler) InitEndpoints(handlers ...func(*fiber.Ctx) error) {
	h.apiGroup(h.app, handlers...)

	// h.app.Get("/vectordb-admin", func(c *fiber.Ctx) error { return renderTemplate(c, "partials/htmx_weaviate", nil, "layouts/main") })
	h.app.Get("/vectordb-admin", append(handlers, h.webPageHandler)...)
	h.app.Get("/wdocs", h.WeaviateDocumntsHandler)
	h.app.Post("/upload", h.WeaviateDocumentUpload)
}

func (h *VectorDBHandler) webPageHandler(c *fiber.Ctx) error {
	return renderTemplate(c, "partials/htmx_weaviate", nil, "layouts/main")
}

// Group of api handlers
func (h *VectorDBHandler) apiGroup(app *fiber.App, handlers ...func(*fiber.Ctx) error) {
	api := app.Group("/api/vdb/v1", handlers...)
	api.Get("/objects", func(c *fiber.Ctx) error {
		ki, err := h.fetchDataFromWeaviate(c)
		if err != nil {
			return c.Status(500).JSON(&m.Response{Code: 500, Message: "Weaviate search error"})
		}
		return c.JSON(&m.Response{Code: 0, Message: "Get all objects", Data: ki})
	})
	api.Post("/objects", h.WeaviateDocumentUpload)
	api.Delete("/objects/:id", h.WeaviateDeleteObjectHandler)
	api.Get("/suggest", h.suggestHandler)
	app.Post("/search", h.WeaviateDocumntsHandler)

	api.Get("/objects/:id", func(c *fiber.Ctx) error {
		return c.SendString("Get object with id")
	})
	api.Put("/objects/:id", func(c *fiber.Ctx) error {
		return c.SendString("Update object with id")
	})
}

// Handler for search suggestions
func (h *VectorDBHandler) suggestHandler(c *fiber.Ctx) error {
	text := c.Query("searchText")
	limit := c.QueryInt("limit", 2)

	so := wvservice.DefaultSO().SearchTxt(text).SF(wvservice.FieldContent.String()).SetFields(wvservice.FieldTitle, wvservice.FieldAdditional3).Limit(limit)
	ki, err := h.fetchDataFromWeaviate(c, so)
	if err != nil {
		return c.Status(500).SendString("Error fetching suggestions")
	}
	return c.JSON(ki)
}

func (h *VectorDBHandler) fetchDataFromWeaviate(c *fiber.Ctx, so ...*wvservice.SearchOptions) ([]*wvservice.KnowledgeItem, error) {
	r := help.Log(c)
	r.Infof("Start vector db fetching documents... Search options: %s", utils.Json(so))

	gr, err := wvservice.WeaviateSearch(r, h.rag.GetKB().Client, wvservice.DefaultClassKB, utils.FirstOrDefault(searchOptsGetAllDocs, so...))
	if err != nil {
		r.Errorf("Error fetching from Weaviate: %v", err)
		return nil, err
	}
	r.Info("Finish vector db")
	ki := wvservice.GQLRespConvert[wvservice.KnowledgeItem](gr, wvservice.DefaultClassKB)
	r.Infof("VectorDB return %d objects", wvservice.KnowledgeItems(ki).Len())
	return ki, nil
}

func (h *VectorDBHandler) WeaviateDocumentUpload(c *fiber.Ctx) error {
	r := help.Log(c)
	r.Info("Start")
	file, err := c.FormFile("file-upload")
	if err != nil && err != fasthttp.ErrMissingFile {
		r.Error("File get error: ", err)
		return c.Status(fiber.StatusNotFound).SendString("File upload error")
	}

	if file != nil {
		filePath := fmt.Sprintf("/tmp/%s_%s", time.Now().Format("2006-01-02_15-04-05"), file.Filename)
		if err := c.SaveFile(file, filePath); err != nil {
			r.Errorf("File[%s] save to[%s] - FAIL! err=%v", err)
			return c.Status(400).SendString("File save error")
		}
		r.Infof("File[%s] save to[%s] - OK!", file.Filename, filePath)
		h.rag.GetLogic(m.LogicTypeDocx).WithExternalSource(filePath).Process(r.Ctx, m.ContentSaveToVectorDB(h.rag.GetKB()), m.ContentBackupLocal(r, "/tmp"))
	}

	urlInput := c.FormValue("url-input")
	if urlInput != "" {
		hostname, isValid := help.ValidateURLAndExtractDomain(urlInput)
		if !isValid {
			r.Errorf("URL input is not valid: %s", urlInput)
			return c.Status(400).SendString("URL input is not valid")
		}
		r.Infof("URL input: %s", urlInput)
		if lo.Contains([]string{"lifecell.ua", "lifecell.com.ua"}, hostname) {
			r.Debug("SCRAP from official lifecell web page")
			h.rag.GetLogic(m.LogicTypeWebLifecell).WithExternalSource(urlInput).Process(r.Ctx, m.ContentSaveToVectorDB(h.rag.GetKB()), m.ContentBackupLocal(r, "/tmp"))
		} else {
			h.rag.GetLogic(m.LogicTypeWebOther).WithExternalSource(urlInput).Process(r.Ctx, m.ContentSaveToVectorDB(h.rag.GetKB()), m.ContentBackupLocal(r, "/tmp"))
		}
	}

	if file != nil || urlInput != "" {
		batchResp, err := h.rag.GetKB().AddToWeaviateBatchWithAutoClear(r.Ctx)
		if err != nil {
			r.Errorf("Error adding to Weaviate: %v", err)
			return err
		}
		ft, _ := os.CreateTemp("", "wv-upload-data-*")
		ft.Write(lo.Must(batchResp[0].MarshalJSON()))
		r.Infof("Batch response: %s", utils.Json(batchResp))
	}
	return c.SendStatus(200)
}

func (h *VectorDBHandler) WeaviateDeleteObjectHandler(c *fiber.Ctx) error {
	log := help.Log(c)
	id := c.Params("id")

	log.Infof("Deleting object with ID: %s\n", id)

	if err := h.rag.GetKB().DeleteItemFromWeaviate(help.Log(c).Ctx, id); err != nil {
		log.Errorf("Delete object error: %v", err)
		return c.Status(500).SendString("Weaviate delete error")
	}

	return c.Status(200).Send(nil)
}

func (h *VectorDBHandler) WeaviateDocumntsHandler(c *fiber.Ctx) error {
	log := help.Log(c)
	log.Info("Start handle request")
	var ki []*wvservice.KnowledgeItem
	var err error
	if c.Method() == fiber.MethodPost {
		var req wvservice.SearchRequest
		if err := c.BodyParser(&req); err != nil {
			log.Error("Error parsing request body: ", err)
			return err
		}
		ki, err = h.fetchDataFromWeaviate(c, req.ToSearchOptions())
	} else {
		ki, err = h.fetchDataFromWeaviate(c)
	}

	if err != nil {
		log.Errorf("Error fetching from Weaviate: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching documents")
	}

	log.Infof("Fetched %d documents from Weaviate", len(ki))
	if h.isDebug {
		log.Info(utils.JsonPrettyStr(ki[0]))
		log.Infof("First document: TimeCreationString=%v; LastUpdateTime=%v", ki[0].TimeCreationString(), ki[0].Additional.LastUpdateTime())
	}
	return renderTemplate(c, "partials/fromWeaviate", ki)
}
