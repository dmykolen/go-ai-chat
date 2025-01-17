package wvservice

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gookit/goutil"
	"github.com/gookit/goutil/errorx"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/structs"
	"github.com/gookit/slog"
	"github.com/mitchellh/copystructure"
	"github.com/samber/lo"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
	gh "gitlab.dev.ict/golang/libs/gohttp"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gologgers/applogger"
	"gitlab.dev.ict/golang/libs/utils"
)

const (
	DefaultClassKB      = "KnowledgeBase"
	DefaultClassKB_json = "../assets/knowledge_base.json"
	ch                  = "VectorDB"
	MAX_TOKEN_SUPPORT   = 8191
	// MAX_TOKEN_SUPPORT = 4096
)

//go:embed knowledge_base.json
var kb_json_file string

type WeaviateCfg struct {
	Host   string `json:"host"   env:"WEAVIATE_HOST"   default:"localhost" flag:"wv_h,   weaviate host (e.g.: localhost,dev-worker-7.dev.ict,etc...)"`
	Port   string `json:"port"   env:"WEAVIATE_PORT"   default:"8082"      flag:"wv_p,   weaviate port"`
	Scheme string `json:"scheme" env:"WEAVIATE_SCHEME" default:"http"      flag:"wv_sch, weaviate scheme"`
	Loglvl string `json:"loglvl" env:"WEAVIATE_LOGLVL" default:"error"     flag:"wv_lvl, weaviate log lvl"`
	Log    any    `json:"-"`
}

func (cfg *WeaviateCfg) String() string {
	return fmt.Sprintf("WeaviateCfg: host=%s; port=%s; scheme=%s; loglvl=%s", cfg.Host, cfg.Port, cfg.Scheme, cfg.Loglvl)
}

func NewWVClient(cfg *WeaviateCfg) *weaviate.Client {
	// var l any
	// if cfg.Log != nil {
	// 	l = cfg.Log
	// } else {
	// 	l = gh.WithLogCFG(applogger.NewLogCfgDef().WithLevel(cfg.Loglvl).WithColor(false).WithMaxLength(300))
	// }
	httpOptionLog := lo.IfF(cfg.Log != nil, func() gh.HttpClientOption { return gh.WithLog(cfg.Log) }).
		ElseF(func() gh.HttpClientOption {
			return gh.WithLogCFG(applogger.NewLogCfgDef().WithLevel(cfg.Loglvl).WithColor(false).WithMaxLength(300))
		})

	return Weaviate(
		fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		cfg.Scheme,
		gh.NewHttpClient(gh.WithPRX(nil), gh.WithTO(120), httpOptionLog).Client,
	)
}

func Weaviate(host, scheme string, httpClient *http.Client) *weaviate.Client {
	return lo.Must(weaviate.NewClient(weaviate.Config{Host: host, Scheme: scheme, ConnectionClient: httpClient}))
}

func WeaviateDefault(logLvl ...string) *weaviate.Client {
	cfg := &WeaviateCfg{}
	structs.InitDefaults(cfg)
	if len(logLvl) > 0 {
		cfg.Loglvl = logLvl[0]
	}
	return NewWVClient(cfg)
}

// KnowledgeItem is a document or part of document.
type KnowledgeItem struct {
	Title    string `json:"title"`
	ChunkNo  int    `json:"chunkNo,omitempty"`
	Content  string `json:"content,omitempty"`
	URL      string `json:"url,omitempty"`
	Category string `json:"category,omitempty"`
	Summary  string `json:"summary,omitempty"`
	KeyWords string `json:"keyWords,omitempty"`
	// Additional map[string]interface{} `json:"_additional,omitempty"`
	Additional AdditionalMap `json:"_additional,omitempty"`
}

func (ki *KnowledgeItem) ID() string {
	return ki.Additional.ID()
}

func (ki *KnowledgeItem) TimeCreationString() string {
	return ki.Additional.CreationTime().Format(time.DateTime)
}

func (ki *KnowledgeItem) TimeUpdString() string {
	return ki.Additional.LastUpdateTime().Format(time.DateTime)
}

func (ki *KnowledgeItem) GenerateURL() string {
	if ki.Category == "confluence" {
		return fmt.Sprintf("https://docs-bizcell.lifecell.ua%s", ki.URL)
	}
	return ki.URL
}

type KnowledgeItems []*KnowledgeItem

func (ki KnowledgeItems) String() string {
	return fmt.Sprintf("KnowledgeItems Total=%d Dupls=%d", ki.Len(), len(ki.FindDuplicates()))
}

func (ki KnowledgeItems) Json() string {
	return fmt.Sprintf("%s", utils.JsonPretty(ki))
}

func (ki KnowledgeItems) Len() int { return len(ki) }

func (ki KnowledgeItems) FindDuplicates() (dupls []*KnowledgeItem) {
	var unique []*KnowledgeItem
	var isUnique = func(item1, item2 *KnowledgeItem) bool {
		return item1.Title == item2.Title && item1.ChunkNo == item2.ChunkNo
	}

	for _, obj := range ki {
		if lo.ContainsBy(unique, func(i *KnowledgeItem) bool { return isUnique(i, obj) }) {
			dupls = append(dupls, obj)
		} else {
			unique = append(unique, obj)
		}
	}

	return
}

func NewKI(title, content, url, category, summary, keyWords string) *KnowledgeItem {
	return &KnowledgeItem{
		Title:    title,
		Content:  content,
		URL:      url,
		Category: category,
		Summary:  summary,
		KeyWords: keyWords,
	}
}

func (k *KnowledgeItem) String() string {
	return fmt.Sprintf("KnowledgeItem=[title=%s; chunk=%d; url=%s; category=%s; additional=%v; content=%s; keyWords=%s; summary=%s]", k.Title, k.ChunkNo, k.URL, k.Category, k.Additional, strings.ReplaceAll(utils.StrCut(k.Content, 30), "\n", ""), k.KeyWords[:mathutil.Min(len(k.KeyWords), 50)], utils.StrCut(strconv.Quote(k.Summary), 30))
}

// ToWeaviate converts KnowledgeItem to Weaviate Object.
func (k *KnowledgeItem) ToWeaviate(clsName string) *models.Object {
	return &models.Object{
		Class: clsName,
		Properties: map[string]interface{}{
			"title":    k.Title,
			"chunkNo":  k.ChunkNo,
			"content":  k.Content,
			"url":      k.URL,
			"category": k.Category,
			"summary":  k.Summary,
			"keywords": k.KeyWords,
		},
	}
}
func (k *KnowledgeItem) _ToWeaviate(clsName string) *models.Object {
	// dataObjs := []models.PropertySchema{}
	return nil
}

// KnowledgeBase is a collection of KnowledgeItem.
type KnowledgeBase struct {
	*weaviate.Client
	Items   []*KnowledgeItem `json:"Items"`
	Class   string           `json:"WV_ClassName"`
	log     *gl.Logger
	IsDebug bool
}

// NewKnowledgeBase creates a new knowledgebase
func NewKnowledgeBase(client *weaviate.Client, log *gl.Logger, className, classConfigFile string) *KnowledgeBase {
	log.Infof("Start KnowledgeBase creation... for class=%s with file=%s", className, classConfigFile)
	kb := &KnowledgeBase{
		Client: client,
		log:    log,
		Class:  className,
	}

	if classConfigFile == "" {
		kb.log.Warn("Init KnowledgeBase without class creation from file!", kb.String())
		return kb
	}

	log.Info("JSON_FILE_CONTENT:", kb_json_file)

	if !kb.IsClassExists(context.Background()) {
		log.Infof("Class=%s exists=%t", className, false)
		if kb_json_file != "" {
			kb.CreateClassFromVar(context.Background(), kb_json_file)
		} else {
			kb.CreateClassFromJson(context.Background(), classConfigFile)
		}
	}
	return kb
}

func (kb *KnowledgeBase) Search(ctx context.Context, so *SearchOptions) (*models.GraphQLResponse, error) {
	return WeaviateSearch(kb.log.RecWithCtx(ctx), kb.Client, kb.Class, so)
}

func (kb *KnowledgeBase) TotalItems() int {
	r, e := WeaviateSearch(kb.log.RecWithCtx(context.Background()), kb.Client, kb.Class, NewSO().Limit(10000))
	if e != nil {
		kb.log.Errorf("Error in search: %v", e)
		return 0
	}
	return len(GQLRespConvert[KnowledgeItem](r, kb.Class))
}

func (kb *KnowledgeBase) IsClassExists(ctx context.Context, className ...string) bool {
	if len(className) == 0 {
		className = append(className, kb.Class)
	}
	isExists, err := kb.Client.Schema().ClassExistenceChecker().WithClassName(className[0]).Do(ctx)
	if err != nil {
		kb.log.WithCtx(ctx).Errorf("Check class exists error: %v", err)
	}
	return isExists
}

func (kb *KnowledgeBase) CreateClassFromVar(ctx context.Context, contentJson string) error {
	log := kb.log.RecWithCtx(ctx, ch)
	var class = &models.Class{}
	utils.JsonToStruct([]byte(contentJson), class)

	if kb.IsClassExists(ctx) {
		log.Warnf("Class=%s alredy exists... Exit creation...", class.Class)
		return nil
	}

	if err := kb.Client.Schema().ClassCreator().WithClass(class).Do(ctx); err != nil {
		log.Errorf("Class[%s] create failed, cause error: %v", kb.Class, err)
		return err
	}
	log.Infof("Created class - %s", class.Class)
	return nil
}

func (kb *KnowledgeBase) CreateClassFromJson(ctx context.Context, fileName string) error {
	log := kb.log.RecWithCtx(ctx, ch)
	var class = &models.Class{}
	utils.JsonToStructFileRead(fileName, class)
	log.Infof("Read scheme from file=[%s] => %#v", fileName, class)

	if kb.IsClassExists(ctx) {
		log.Warnf("Class=%s alredy exists... Exit creation...", class.Class)
		return nil
	}

	if err := kb.Client.Schema().ClassCreator().WithClass(class).Do(ctx); err != nil {
		log.Errorf("Class[%s] create failed, cause error: %v", kb.Class, err)
		return err
	}
	log.Infof("Created class - %s", class.Class)
	return nil
}

func (kb *KnowledgeBase) String() string {
	return fmt.Sprintf("KnowledgeBase: wv_client=%v; is_log_not_null=%t; size_items=%d", lo.Must(kb.Client.Misc().MetaGetter().Do(context.Background())), kb.log != nil, len(kb.Items))
}

func (kb *KnowledgeBase) Size() int     { return len(kb.Items) }
func (kb *KnowledgeBase) IsEmpty() bool { return kb.Size() == 0 }
func (kb *KnowledgeBase) ClearItems()   { kb.Items = []*KnowledgeItem{} }

// AddItem adds a new knowledge item to the knowledge base.
func (kb *KnowledgeBase) AddItem(ctx context.Context, title, content, url, category, summary, keyWords string) {
	kb.AddItemSimple(ctx, NewKI(title, content, url, category, summary, keyWords))
}

// AddItemSimple adds a knowledge item to the knowledge base.
// If the content of the item exceeds the maximum token support, it will be divided into chunks.
// Each chunk will be assigned a chunk number and added to the knowledge base.
// Parameters:
//   - item: The knowledge item to be added.
func (kb *KnowledgeBase) AddItemSimple(ctx context.Context, item *KnowledgeItem) {
	log := kb.log.RecWithCtx(ctx, ch)
	log.Infof("AddItemSimple: %s", item)
	if cntTokens := countTokens(item.Content); cntTokens > MAX_TOKEN_SUPPORT {
		log.Warnf("Content size=%d is too large then %d. Divide into chunks", len(item.Content), MAX_TOKEN_SUPPORT)
		chunks := (cntTokens / MAX_TOKEN_SUPPORT) + 1
		chunkLen := (len(item.Content) / chunks) + chunks
		log.Infof("chunks=%d; chunkLen=%d", chunks, chunkLen)
		for i, v := range lo.ChunkString(item.Content, chunkLen) {
			newItem, err := copystructure.Copy(item)
			log.Infof("newItem_TYPE=%T", newItem)
			if err != nil {
				panic(err)
			}
			newItem.(*KnowledgeItem).Content = v
			newItem.(*KnowledgeItem).ChunkNo = i + 1
			log.Infof("Item chunk-%d: %s", (i + 1), newItem)
			kb.Items = append(kb.Items, newItem.(*KnowledgeItem))
		}
		log.Infof("Items %v", kb.Items)
	} else {
		kb.Items = append(kb.Items, item)
	}
}

// ToWeaviate converts the KnowledgeBase to a list of Weaviate objects.
func (kb *KnowledgeBase) ToWeaviate() []*models.Object {
	var objs []*models.Object
	for _, item := range kb.Items {
		objs = append(objs, item.ToWeaviate(kb.Class))
	}
	return objs
}

// UpdateItemInWeaviate updates the items in the Weaviate knowledge base.
func (kb *KnowledgeBase) UpdateItemInWeaviate(ctx context.Context) error {
	for _, item := range kb.Items {
		if item.ID() == "" {
			item.Additional["id"] = kb.GetObjByTitleEQ(ctx, item.Title, FieldAdditional1)
		}
		err := kb.Client.Data().Updater().WithClassName(kb.Class).WithID(item.ID()).WithProperties(item.ToWeaviate(kb.Class).Properties).Do(ctx)
		kb.log.WithCtx(ctx).Errorf("Weaviate update object with id=%s failed. Error: %v", item.ID(), err)
	}
	return nil
}

// AddItemsToWeaviate adds items to the Weaviate knowledge base.
// It iterates over the items in the KnowledgeBase and sends them to the Weaviate server.
// If an error occurs while adding an item, it is added to the error map.
// Returns an error map containing any errors that occurred during the process.
func (kb *KnowledgeBase) AddItemsToWeaviate(ctx context.Context) error {
	r := kb.log.RecWithCtx(ctx, ch)
	errMap := errorx.ErrMap{}
	for _, item := range kb.Items {
		_, err := item.AddItemToWeaviate(r, kb.Client, kb.Class)
		if err != nil {
			errMap[item.Title] = err
		}
	}
	return errMap.ErrorOrNil()
}

func (item *KnowledgeItem) AddItemToWeaviate(r *slog.Record, w *weaviate.Client, class string) (id string, e error) {
	o, e := w.Data().Creator().WithClassName(class).WithProperties(item.ToWeaviate(class).Properties).Do(r.Ctx)
	if e != nil {
		r.Errorf("Object[%s] creation failed: %v", item.Title, e)
		return "", e
	}
	r.Infof("Created object with id=%s", o.Object.ID)
	r.Debugf("%s", utils.JsonPretty(o))

	return string(o.Object.ID), nil
}

func (kb *KnowledgeBase) AddToWeaviateBatchNew(ctx context.Context) ([]models.ObjectsGetResponse, error) {
	batcher := kb.Client.Batch().ObjectsBatcher()

	for _, item := range kb.Items {
		batcher.WithObjects(item.ToWeaviate(kb.Class))
	}

	batchRes, err := batcher.Do(ctx)

	if err != nil {
		kb.log.Infof("BATCH_RESP: %v", batchRes)
		kb.log.Infof("BATCH_RESP_ERR: %w", err)
		return batchRes, err
	}
	// Check the response for any errors.
	for _, res := range batchRes {
		if res.Result.Errors != nil {
			return batchRes, errorx.E(res.Result.Errors.Error[0].Message)
		}
	}
	return batchRes, err
}

// AddToWeaviateBatch adds the knowledge base to weaviate using the batch API.
func (kb *KnowledgeBase) AddToWeaviateBatch(ctx context.Context) ([]models.ObjectsGetResponse, error) {
	batchRes, err := kb.Client.Batch().ObjectsBatcher().WithObjects(kb.ToWeaviate()...).Do(ctx)
	if err != nil {
		return batchRes, err
	}
	// Check the response for any errors.
	for _, res := range batchRes {
		if res.Result.Errors != nil {
			return batchRes, errorx.E(res.Result.Errors.Error[0].Message)
		}
	}
	return batchRes, err
}

func (kb *KnowledgeBase) AddToWeaviateBatchWithAutoClear(ctx context.Context) ([]models.ObjectsGetResponse, error) {
	defer kb.ClearItems()
	return kb.AddToWeaviateBatch(ctx)
}

func (kb *KnowledgeBase) GetObjectsFromWeaviate(ctx context.Context, withVector bool) []*models.Object {
	log := kb.log.RecWithCtx(ctx, ch)
	dataGetter := kb.Client.Data().ObjectsGetter().WithClassName(kb.Class).WithAdditional("classification")

	if withVector {
		dataGetter = dataGetter.WithVector()
	}
	res, err := dataGetter.WithLimit(2000).Do(ctx)
	if err != nil {
		log.Errorf("Error getting objects from Weaviate: %v", err)
		return nil
	}
	log.Infof("Returned %d objects for class=%s", len(res), kb.Class)

	if kb.IsDebug {
		for idx, obj := range res {
			log.Debugf("obj[%d]> %s", idx, help_object_to_str(obj))
		}
	}
	return res
}

func (kb *KnowledgeBase) GetAllObjectsFromWeaviateGQL(ctx context.Context, sort []graphql.Sort, fields ...Field) []*KnowledgeItem {
	log := kb.log.RecWithCtx(ctx, ch)
	fieldsDef := []graphql.Field{f("title"), f("category"), f("chunkNo"), f(Additioanl2)}
	if len(fields) > 0 {
		fieldsDef = fieldsList(fields...)
	}
	res, err := kb.GraphQL().Get().WithClassName(kb.Class).
		// WithFields(f("title"), f("category"), f("chunkNo"), f("_additional {id}")).
		WithFields(fieldsDef...).
		WithLimit(1000).
		// WithSort(graphql.Sort{Path: []string{"title"}, Order: graphql.Desc}).
		// WithSort(SortBy(FieldTitle, false)).
		WithSort(sort...).
		// WithHybrid(kb.GraphQL().HybridArgumentBuilder().WithQuery("")).
		Do(ctx)

	log.Debugf("err: %v; res: %v", err, res)

	if err == nil && res.Errors == nil {
		return kb.JqlRespToListKI(res)
	}
	return nil
}

func (kb *KnowledgeBase) GetObjByID(ctx context.Context, id string) (*models.Object, error) {
	res, err := kb.Client.Data().ObjectsGetter().WithClassName(kb.Class).WithID(id).Do(ctx)
	if err != nil {
		return nil, err
	}
	kb.log.RecWithCtx(ctx, ch).Infof("Returned %d objects for class=%s", len(res), kb.Class)
	return res[0], nil
}

func (kb *KnowledgeBase) GetObjByTitle(ctx context.Context, title string) []*KnowledgeItem {
	log := kb.log.RecWithCtx(ctx, ch)
	res, err := kb.GraphQL().Get().WithClassName(kb.Class).
		WithFields(f("title"), f("category"), f("chunkNo"), f("_additional {id}")).
		WithWhere(FilterWhereTitleLike(title)).
		Do(ctx)
	goutil.PanicErr(err)
	log.Infof("Returned objects for class=%s", kb.Class)
	if res.Errors == nil {
		return kb.JqlRespToListKI(res)
	}
	for _, e := range res.Errors {
		log.Errorf("error in Path=%v ERR_message: %s", e.Path, e.Message)
	}
	return nil
}

func (kb *KnowledgeBase) GetObjByTitleEQ(ctx context.Context, title string, fields ...Field) []*KnowledgeItem {
	log := kb.log.RecWithCtx(ctx, ch)
	fieldsDef := []graphql.Field{f("title"), f("url"), f("category"), f("chunkNo"), f("_additional {id}")}
	if len(fields) > 0 {
		fieldsDef = fieldsList(fields...)
	}

	res, err := kb.GraphQL().Get().
		WithClassName(kb.Class).
		WithFields(fieldsDef...).
		WithWhere(FilterWhereTitleEQ(title)).
		Do(ctx)
	goutil.PanicErr(err)
	log.Infof("Returned objects for class=%s", kb.Class)
	if res.Errors == nil {
		return kb.JqlRespToListKI(res)
	}
	for _, e := range res.Errors {
		log.Errorf("error in Path=%v ERR_message: %s", e.Path, e.Message)
	}
	return nil
}

func (kb *KnowledgeBase) SearchInContentsHybrid(ctx context.Context, searchText string, limit int, fields ...Field) []*KnowledgeItem {
	log := kb.log.RecWithCtx(ctx, ch)
	h := kb.GraphQL().HybridArgumentBuilder().
		WithQuery(searchText).
		WithFusionType(graphql.Ranked).
		WithProperties([]string{"content"})

	res, err := kb.GraphQL().Get().
		WithClassName(kb.Class).
		// WithFields(f("title"), f("content"), f("category"), f("chunkNo"), f("_additional {id creationTimeUnix score explainScore certainty distance}")).
		WithFields(fieldsList(fields...)...).
		WithLimit(limit).
		WithHybrid(h).
		Do(ctx)

	if err != nil {
		log.Errorf("Error in search: %v", err)
		return nil
	}
	log.Infof("Hybrid search finish in class=%s", kb.Class)

	if res.Errors == nil {
		return kb.JqlRespToListKI(res)
	}
	for _, e := range res.Errors {
		log.Errorf("error in Path=%v LOC=[%+v] ERR_message: %s", e.Path, e.Locations, e.Message)
	}

	return nil

}

func (kb *KnowledgeBase) GetContentsForItems(ctx context.Context, items []*KnowledgeItem) []string {
	var res []string
	for idx, item := range items {
		obj, err := kb.GetObjByID(ctx, item.ID())
		if err != nil {
			kb.log.Errorf("Error getting obj by id[%s]: %v", item.ID(), err)
			continue
		}
		items[idx].Content = ObjContent(obj)
		res = append(res, ObjContent(obj))
	}
	return res
}

// DeleteItemsFromWeaviate delete the item in the Weaviate knowledge base.
func (kb *KnowledgeBase) DeleteItemFromWeaviate(ctx context.Context, uuid string) error {
	return kb.Client.Data().Deleter().WithClassName(kb.Class).WithID(uuid).Do(ctx)
}

func (kb *KnowledgeBase) RemoveDuplicates(ctx context.Context, items []*KnowledgeItem, class ...string) (dupls []*KnowledgeItem) {
	ki := KnowledgeItems(items)
	if ki.Len() == 0 {
		ki = KnowledgeItems(kb.Items)
	}
	dupls = ki.FindDuplicates()
	kb.Class = utils.FirstOrDefault(kb.Class, class...)
	kb.log.WithCtx(ctx).Infof("Found %d duplicates! Start removing dupls from class=%s", len(dupls), kb.Class)

	for _, v := range dupls {
		if err := kb.DeleteItemFromWeaviate(ctx, v.ID()); err != nil {
			kb.log.WithCtx(ctx).Errorf("Delete item[%s] error: %v", v.ID(), err)
		}
	}
	return
}

// DeleteFull deletes the knowledge base class
func (kb *KnowledgeBase) DeleteFull(ctx context.Context) {
	if !kb.IsClassExists(ctx) {
		kb.log.WithCtx(ctx).Infof("class[%s] already NOT exists!", kb.Class)
		return
	}
	if err := kb.Client.Schema().ClassDeleter().WithClassName(kb.Class).Do(ctx); err != nil {
		kb.log.WithCtx(ctx).Errorf("Class[%s] delete failed, cause error: %v", kb.Class, err)
	}
	kb.log.WithCtx(ctx).Infof("Class[%s] delete success!", kb.Class)
}

func (kb *KnowledgeBase) JqlRespToListKI(gqlResp *models.GraphQLResponse) []*KnowledgeItem {
	v := gqlResp.Data["Get"].(map[string]interface{})[kb.Class].([]interface{})
	var items2 []*KnowledgeItem
	utils.JsonToStruct(utils.Json(v), &items2)
	return items2
}

// result, err := client.Backup().Creator().
// WithIncludeClassNames("Article", "Publication").
// WithBackend(backup.BACKEND_FILESYSTEM).
// WithBackupID("my-very-first-backup").
// WithWaitForCompletion(true).
// WithCompressionConfig(backup.Compression{
//   Level:         backup.BackupConfigCompressionLevelBestSpeed,
//   ChunkSize:     512,
//   CPUPercentage: 80,
// }).
// Do(context.Background())
