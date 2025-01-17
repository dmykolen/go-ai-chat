package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	m "gitlab.dev.ict/golang/go-ai/models"
	"gitlab.dev.ict/golang/go-ai/services"
	wvservice "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/goopenai"
	"gitlab.dev.ict/golang/libs/utils"
)

var (
	AIClient = goopenai.New().WithProxy(false).WithLogger(gologgers.New(gologgers.WithOC())).Build()
	WClient  = wvservice.WeaviateDefault()
	log      = gologgers.New(gologgers.WithOC())
	kb       = wvservice.NewKnowledgeBase(WClient, log, wvservice.DefaultClassKB, "")
	dp       = services.NewDocxPprocessor(log, resty.New()).WithTableStyle("pretty")
)

type VectorDBHandler struct {
	app *fiber.App
	kb  *wvservice.KnowledgeBase
	log *gologgers.Logger
}

func NewVectorDBHandler(app *fiber.App, kb *wvservice.KnowledgeBase, log *gologgers.Logger) *VectorDBHandler {
	return &VectorDBHandler{app: app, kb: kb, log: log}
}

func (h *VectorDBHandler) InitEndpoints() {
	h.apiGroup(h.app)

	h.app.Get("/hw", func(c *fiber.Ctx) error { return c.SendFile("web/static/htmx_test/htmx_weaviate.html") })
	h.app.Get("/wdocs", h.WeaviateDocumntsHandler)
	h.app.Post("/upload", h.WeaviateDocumentUpload)
	h.app.Delete("/object/:id", h.WeaviateDeleteObjectHandler)
}

func main() {
	log.Info(os.Getwd())
	engine := html.New("/Users/dmykolen/go/src/go-ai/web/static/htmx_test", ".html")
	app := fiber.New(fiber.Config{Views: engine})
	app.Static("/", ".", fiber.Static{Browse: true})

	app.Use(func(c *fiber.Ctx) error {
		fmt.Printf("Request: %s\n", c.Path())
		return c.Next()
	})

	app.Use(ErrorsMiddleware)

	vdbHandler := NewVectorDBHandler(app, kb, log)
	vdbHandler.InitEndpoints()

	app.Get("/rec", func(c *fiber.Ctx) error { return c.SendFile("web/static/audio_record2.html") })
	// Add routes
	// app.Post("/search", h.WeaviateDocumntsHandler)
	// app.Get("/suggest", h.suggestHandler)
	// app.Post("/stt", HandlerAudio)

	// Start the server
	if err := app.Listen("0.0.0.0:5556"); err != nil {
		panic(err)
	}
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Group of api handlers
func (h *VectorDBHandler) apiGroup(app *fiber.App) {
	api := app.Group("/api/vdb/v1")
	api.Get("/objects", func(c *fiber.Ctx) error {
		ki, err := h.fetchDataFromWeaviate()
		if err != nil {
			return c.Status(500).JSON(&Response{Code: 500, Message: "Weaviate search error"})
		}
		return c.JSON(&Response{Code: 0, Message: "Get all objects", Data: ki})
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

	so := wvservice.DefaultSO().SearchTxt(text).SF(wvservice.FieldContent.String()).SetFields(wvservice.FieldTitle, wvservice.FieldAdditional3)
	ki, err := h.fetchDataFromWeaviate(so)
	if err != nil {
		return c.Status(500).SendString("Error fetching suggestions")
	}
	return c.JSON(ki)
}

func (h *VectorDBHandler) fetchDataFromWeaviate(so ...*wvservice.SearchOptions) ([]*wvservice.KnowledgeItem, error) {
	r := log.WithCtx(utils.GenerateCtxWithRid())
	r.Infof("Search options: %s", utils.JsonPretty(so))

	searchOpts := wvservice.DefaultSO().Limit(200).Fields(wvservice.FieldContent, wvservice.FieldAdditional2).SortOrder(wvservice.FieldTitle, false)
	if len(so) > 0 {
		searchOpts = so[0]
	}

	gr, err := wvservice.WeaviateSearch(r, kb.Client, wvservice.DefaultClassKB, searchOpts)
	if err != nil {
		return nil, err
	}
	r.Tracef("Weaviate search response. GraphQL resp: %s", utils.JsonPrettyStr(gr))
	ki := wvservice.GQLRespConvert[wvservice.KnowledgeItem](gr, wvservice.DefaultClassKB)
	r.Info("Weaviate search response. KnowledgeItems: ", wvservice.KnowledgeItems(ki).Len())
	return ki, nil
}

func (h *VectorDBHandler) WeaviateDocumentUpload(c *fiber.Ctx) error {
	r := log.WithCtx(utils.GenerateCtxWithRid())
	file, err := c.FormFile("file-upload")
	if err != nil {
		return c.Status(400).SendString("File upload error")
	}

	filePath := fmt.Sprintf("/tmp/%s_%s", time.Now().Format("2006-01-02_15-04-05"), file.Filename)
	if err := c.SaveFile(file, filePath); err != nil {
		r.Errorf("File[%s] save to[%s] - FAIL! err=%v", err)
		return c.Status(400).SendString("File save error")
	}
	r.Infof("File[%s] save to[%s] - OK!", file.Filename, filePath)

	dp.WithFilePaths(filePath).Process(r.Ctx, m.ContentSaveToVectorDB(kb))
	kb.AddToWeaviateBatch(r.Ctx)

	return c.SendStatus(200)
}

func (h *VectorDBHandler) WeaviateDeleteObjectHandler(c *fiber.Ctx) error {
	r := log.WithCtx(utils.GenerateCtxWithRid())
	id := c.Params("id")

	log.Infof("Deleting object with ID: %s\n", id)

	if err := kb.DeleteItemFromWeaviate(r.Ctx, id); err != nil {
		return c.Status(500).SendString("Weaviate delete error")
	}

	// return c.SendStatus(200)
	// return c.SendStatus(fiber.StatusNoContent)
	return c.Status(200).Send(nil)
}

func (h *VectorDBHandler) WeaviateDocumntsHandler(c *fiber.Ctx) error {
	log.Info("Start handle request")
	var ki []*wvservice.KnowledgeItem
	var err error
	if c.Method() == "POST" {
		var req wvservice.SearchRequest
		if err := c.BodyParser(&req); err != nil {
			log.Error("Error parsing request body: ", err)
			return err
		}
		ki, err = h.fetchDataFromWeaviate(req.ToSearchOptions())
	} else {
		ki, err = h.fetchDataFromWeaviate()
	}

	if err != nil {
		log.Errorf("Error fetching from Weaviate: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error fetching documents")
	}

	log.Infof("Fetched %d documents from Weaviate", len(ki))
	log.Debug(utils.JsonPrettyStr(ki[0]))
	log.Infof("First document: TimeCreationString=%v; LastUpdateTime=%v", ki[0].TimeCreationString(), ki[0].Additional.LastUpdateTime())

	// Assuming 'ki' is the data to be rendered and "fromWeaviate" is your template
	return renderTemplate(c, "fromWeaviate", ki)
}

func renderTemplate(c *fiber.Ctx, templateName string, data interface{}, layouts ...string) error {
	err := c.Render(templateName, data, layouts...)
	if err != nil {
		return m.NewRenderingError(err)
	}
	return nil
}

func ErrorsMiddleware(c *fiber.Ctx) error {
	err := c.Next()

	var ie *m.InternalError
	if err != nil && errors.As(err, &ie) {
		log.Printf("Internal Error: %s", ie.Message)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	}

	if err != nil {
		switch e := err.(type) {
		case *m.RenderingError:
			log.Errorf("Error during rendering: %v", e.OriginalError)
			return c.Status(fiber.StatusInternalServerError).SendString("Error rendering page")
		case *m.NotFoundError:
			log.Printf("Not Found Error: %s", e.Error())
			return c.Status(fiber.StatusNotFound).SendString("Page not found")
		default:
			if errors.Is(err, m.ErrTemplateNotFound) {
				log.Printf("Template not found error: %v", err)
				return c.Status(fiber.StatusNotFound).SendString(err.Error())
			}
			log.Errorf("An unexpected error occurred: %v", err)
			return c.Status(fiber.StatusInternalServerError).SendString("An unexpected error occurred")
		}
	}

	return nil
}

func HandlerAudio(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).SendString("File upload error")
	}
	mf, err := file.Open()
	if err != nil {
		return c.Status(400).SendString("File open error")
	}
	defer mf.Close()

	// Copy the file content to the buffer
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, mf); err != nil {
		return c.Status(400).SendString("File copy error")
	}

	// Save the file
	// buf.WriteTo(os.Stdout)

	// Store the file
	c.SaveFile(file, fmt.Sprintf("./uploads/%s_%s", time.Now().Format("2006-01-02_15-04-05"), file.Filename))
	fmt.Printf("BUFFER length=%d", buf.Len())

	// Send the file to the AI
	text := AIClient.AudioTranscript(context.Background(), "audio.wav", buf)
	// text := "TEEEXT"

	// For now, just respond with a placeholder text
	// return c.SendString("Voice file received. Text conversion not implemented yet.")
	return c.SendString(text)
}

func testmain() {
	r := log.WithCtx(utils.GenerateCtxWithRid())

	gr, err := wvservice.WeaviateSearch(r, kb.Client, wvservice.DefaultClassKB, wvservice.DefaultSO().Limit(1).Fields(wvservice.FieldTitle, wvservice.FieldUrl, wvservice.FieldAdditional1))
	if err != nil {
		r.Errorf("Weaviate search error: %v", err)
		return
	}
	r.Info("Weaviate search response. GraphQL resp: ", utils.JsonPrettyStr(gr))
	ki := wvservice.GQLRespConvert[wvservice.KnowledgeItem](gr, wvservice.DefaultClassKB)
	r.Info("Weaviate search response. KnowledgeItems resp: ", utils.JsonPrettyStr(ki))
	r.Infof("Weaviate search response. KnowledgeItems resp: %#v", ki[0])
	i := ki[0]
	r.Infof("Title=%s, Url=%s, Content=%s", i.Title, i.URL, i.Content, i.Category, i.Additional["id"], i.Additional["creationTimeUnix"])

}
