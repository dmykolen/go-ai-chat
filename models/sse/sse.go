package sse

import (
	"bufio"
	"fmt"
	"strconv"
	"time"

	"gitlab.dev.ict/golang/go-ai/helpers"
	"gitlab.dev.ict/golang/libs/gologgers"
)

const (
	FmtEvtFull        = "id: %s\nevent: %s\ndata: %s\nretry: %d\n\n"
	FmtEvtFullNoRetry = "id: %s\nevent: %s\ndata: %s\n\n"
	FmtEvt            = "event: %s\ndata: %s\n\n"
	FmtData           = "%sdata: %s\n\n"
	CookUID           = "userId"
	CookUName         = "username"
	ANON              = "anonymous"
)

type EventType int

const (
	EvtNul EventType = iota
	EvtChatGptResp
	EvtInfo
	EvtAnnounce
	EvtSQLResult
)

var (
	// mapping event type to string
	DictEvents = map[EventType]string{
		EvtNul:         "",
		EvtChatGptResp: "chatgpt_response",
		EvtInfo:        "info",
		EvtAnnounce:    "announce",
		EvtSQLResult:   "sql_table_as_json",
	}
)

type Event struct {
	ID    string
	Type  any    `validate:"required"`
	Msg   string `validate:"required"`
	Retry int
	TabId string
}

func (e Event) String() string {
	return fmt.Sprintf("EventMsg{IdEvent: %s, Type: %v, Msg: %s, Retry: %d, TabId: %s}", e.ID, e.Type, e.Msg, e.Retry, e.TabId)
}

func NewEventMsg(id string, t any, m string, r int) Event {
	return Event{ID: id, Type: t, Msg: m, Retry: r}
}

func (e Event) ET() EventType {
	switch tv := e.Type.(type) {
	case int:
		return EventType(tv)
	case string:
		for k, v := range DictEvents {
			if v == tv {
				return EventType(k)
			}
		}
	}
	panic("Unknown event type")
}

func (e Event) T() (t string) {
	switch tv := e.Type.(type) {
	case int:
		t = DictEvents[EventType(tv)]
	case EventType:
		t = DictEvents[tv]
	case string:
		t = tv
	}
	return
}

func (e Event) SendSSE(rec *gologgers.LogRec, uLogin, uId string, w *bufio.Writer) error {
	sseMsg, err := e.MakeMsgSSE(rec)
	if err != nil {
		rec.WithError(err).Errorf("Failed send event type[%d] to user=[%s:%s]", e.Type, uLogin, uId)
		return err
	}
	counBytesSent, err := w.WriteString(sseMsg)
	rec.WithError(err).Infof("Send event: bytesWasSent=[%d] type=[%d] content=[%s] to user=[%s:%s]", counBytesSent, e.Type, strconv.Quote(sseMsg), uLogin, uId)
	return err
}

func (e Event) MakeMsgSSE(log *gologgers.LogRec) (res string, err error) {
	if e.ID == "" {
		e.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if err = helpers.Validate(e); err != nil {
		log.Errorf("ERR: %#v\n", err)
		return
	}

	switch {
	case e.Retry != 0:
		res = fmt.Sprintf(FmtEvtFull, e.ID, e.T(), e.Msg, e.Retry)
	case e.Retry == 0:
		res = fmt.Sprintf(FmtEvtFullNoRetry, e.ID, e.T(), e.Msg)
	}
	log.Infof("Finish creating SSE[%s] msg => [%s]", e.T(), strconv.Quote(res))
	return
}
