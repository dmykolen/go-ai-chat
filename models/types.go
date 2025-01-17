package models

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gookit/slog"
	h "gitlab.dev.ict/golang/go-ai/helpers"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/utils"
)

type Logic interface {
	Type() LogicType
	WithExternalSource(...string) Logic
	Process(context.Context, ...ContentSaverFunc)
}

type LogicType int

const (
	LogicTypeUnknown LogicType = iota
	LogicTypeDocx
	LogicTypeConfluence
	LogicTypePDF
	LogicTypeWebLifecell
	LogicTypeWebOther
)

type ContentSaverFunc func(ctx context.Context, doc *Doc)

func ContentSaveToVectorDB(db *w.KnowledgeBase) ContentSaverFunc {
	return func(ctx context.Context, d *Doc) {
		db.AddItem(ctx, d.Title, d.TextContent, d.Link, d.Category(), d.Summary(), d.Keywords())
	}
}

func ContentBackupLocal(rec *slog.Record, dir string) ContentSaverFunc {
	dir = utils.ExpandPath(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		rec.Errorf("Error while creating backup dir=%s err=%v", dir, err)
		panic(err)
	}
	rec.Infof("Conten backup to dir=%s", dir)

	return func(ctx context.Context, d *Doc) {
		rec.Debugf("Process doc [%s]", utils.JsonPretty(d))
		fileName := filepath.Join(utils.ExpandPath(dir), h.ToSnake(d.Title))

		// Handle original content for web documents
		if d.Category() == CategoryWEB && d.Original() != "" {
			os.WriteFile(fileName+".orig.html", []byte(d.Original()), 0644)
			fileName += ".html"
		}

		rec.Infof("PARSED backup to file=%s", fileName)
		os.WriteFile(fileName, []byte(d.TextContent), 0644)
	}
}
