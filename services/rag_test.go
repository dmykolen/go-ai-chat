package services

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "gitlab.dev.ict/golang/go-ai/logic/ailogic"
    "gitlab.dev.ict/golang/go-ai/logic/ailogic/llm"
    w "gitlab.dev.ict/golang/go-ai/services/weaviate"
    gl "gitlab.dev.ict/golang/libs/gologgers"
    goai "gitlab.dev.ict/golang/libs/goopenai"
)

type testSetup struct {
    rag     *RAGService
    kb      *w.KnowledgeBase
    ai      *goai.Client
    ctx     context.Context
    cleanup func()
}

func setupTest(t *testing.T) *testSetup {
    t.Helper()

    logger := gl.New(gl.WithChannel("TEST"), gl.WithLevel(gl.LevelInfo))
    client := w.NewWVClient(&w.WeaviateCfg{
        Host:   "localhost",
        Port:   "8083",
        Scheme: "http",
        Loglvl: "debug",
    })
    kb := w.NewKnowledgeBase(client, logger, w.DefaultClassKB, w.DefaultClassKB_json)
    ai := goai.New().WithLogger(logger).Build()
    rag := NewRAGService(logger, kb, ai)

    return &testSetup{
        rag: rag,
        kb:  kb,
        ai:  ai,
        ctx: context.Background(),
        cleanup: func() {
            // Add cleanup code if needed
        },
    }
}

func TestKnowledgeBase_BasicOperations(t *testing.T) {
    ts := setupTest(t)
    defer ts.cleanup()

    t.Run("add items", func(t *testing.T) {
        err := populateTestData(ts.kb, ts.ctx)
        require.NoError(t, err)
    })

    t.Run("get all objects", func(t *testing.T) {
        objects := ts.kb.GetObjectsFromWeaviate(ts.ctx, false)
        assert.NotEmpty(t, objects)
    })

    t.Run("search by title", func(t *testing.T) {
        items := ts.kb.GetObjByTitle(ts.ctx, "test title")
        assert.NotNil(t, items)
    })
}

func TestRAGService_AIOperations(t *testing.T) {
    ts := setupTest(t)
    defer ts.cleanup()

    t.Run("call AI with streaming", func(t *testing.T) {
        llmModel := llm.OpenAiDefault(gl.Defult())
        ts.rag.LLM(llmModel)

        chat := goai.NewChat(goai.SysPromptForChatBot1)
        userChan := make(chan string)

        go func() {
            for msg := range userChan {
                t.Logf("Received: %s", msg)
            }
        }()

        ctx := ailogic.AddToCtxLogin(ts.ctx, "testuser")
        resp, err := ts.rag.CallAICimDB(ctx, "test query", chat, userChan)

        assert.NoError(t, err)
        assert.NotNil(t, resp)
    })
}

func TestRAGService_Search(t *testing.T) {
    ts := setupTest(t)
    defer ts.cleanup()

    testCases := []struct {
        name    string
        query   string
        wantErr bool
    }{
        {
            name:    "basic search",
            query:   "test query",
            wantErr: false,
        },
        {
            name:    "empty query",
            query:   "",
            wantErr: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            results := ts.kb.SearchInContentsHybrid(
                ts.ctx,
                tc.query,
                1,
                w.FieldTitle,
                w.FieldContent,
            )

            if tc.wantErr {
                assert.Empty(t, results)
            } else {
                assert.NotEmpty(t, results)
            }
        })
    }
}

// Helper functions
func populateTestData(kb *w.KnowledgeBase, ctx context.Context) error {
    testItems := []struct {
        title    string
        category string
    }{
        {"First Item", "TEST"},
        {"Second Item", "TEST"},
        {"Third Item", "TEST"},
    }

    for _, item := range testItems {
        kb.AddItem(ctx, item.title, "", "", item.category, "", "")
    }

    _, err := kb.AddToWeaviateBatch(ctx)
    return err
}