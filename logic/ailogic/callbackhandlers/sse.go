package callbackhandlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/go-ai/models/sse"
	"gitlab.dev.ict/golang/libs/gologgers"
)

func CallbackSSEStreamEventMsg(ctx context.Context, log *gologgers.Logger, data string, sseChannel chan string, chunkSize int) error {
	rec := log.RecWithCtx(ctx, "STREAM")
	rec.Infof("Start callback! sseChanel_is_not_null=%t Data=[%s]", sseChannel == nil, data)
	chunks := strings.Split(data, "\n")
	rec.Infof("Chunks[size=%d]", len(chunks))

	if helpers.IsValidJSON(data) {
		msgSSE, err := sse.NewEventMsg("32332", sse.EvtSQLResult, data, 3000).MakeMsgSSE(rec)
		if err != nil {
			rec.Errorf("MakeMsgSSE failed! err: %#v", err)
			return err
		}

		sseChannel <- msgSSE
	} else {
		for idx, chunk := range chunks {
			chunk += " "
			msgSSE, err := sse.NewEventMsg(strconv.Itoa(idx), sse.EvtSQLResult, chunk, 3000).MakeMsgSSE(rec)
			rec.Infof("seqNumber=%d msgSSE: %s", idx, msgSSE)
			if err != nil {
				rec.Errorf("MakeMsgSSE failed! err: %#v", err)
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case sseChannel <- msgSSE:
				time.Sleep(100 * time.Millisecond) // Simulate latency
			}
		}
	}

	rec.Info("End callback!")
	return nil
}
