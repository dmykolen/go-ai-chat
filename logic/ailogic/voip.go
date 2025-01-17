package ailogic

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	"gitlab.dev.ict/golang/go-ai/logic/biz"
	"gitlab.dev.ict/golang/go-ai/models/sse"
	w "gitlab.dev.ict/golang/go-ai/services/weaviate"
	"gitlab.dev.ict/golang/libs/gologgers"
)

const (
	rewrittenQuery   = "contextualized_query"
	nextAgent        = "nextAgent"
	letterToSupport  = "support_letter"
	Response         = "response"
	FinalResponse    = "finalResponse"
	ClarifyQuestions = "clarifyQuestions"
	ChainOfThoughts  = "chainOfThoughts"
	SelfCheck        = "selfCheck"

	chainName1  = "agent_first"
	chainName2  = "agent_second"
	chainNameCI = "agent_call_issue"

	defVoipKeyIN  = "voip_final_result"
	defVoipKeyOUT = "voip_final_result"

	escalation_path_ag_2  = "ai_agent_2"
	escalation_path_ag_ci = "ai_agent_ci"
)

type ChainVoip struct {
	llm        llms.Model
	chainFirst chains.Chain
	// chainSecond chains.Chain
	agents   map[string]chains.Chain
	log      *gologgers.Logger
	inputKey string
	outKey   string
	ws       *biz.WSGetter
	db       *w.KnowledgeBase
}

type runOpts struct {
	chainOpts                          []chains.ChainCallOption
	callbackToUser                     func(context.Context, string) error
	callbackToUserStreamChunkToChannel func(context.Context, *gologgers.Logger, string, chan string, int) error
	callbackToChannel                  func(context.Context, *gologgers.Logger, string, int, func(string)) error
	sseChannel                         chan string
	sendToChan                         func(string)
}

type RunOptFn func(*runOpts)

// WithOptions sets the options field
func WithChainOptions(options ...chains.ChainCallOption) RunOptFn {
	return func(vo *runOpts) { vo.chainOpts = options }
}

// WithCallbackToUser sets the callbackToUser field
func WithCallbackToUser(callback func(context.Context, string) error) RunOptFn {
	return func(vo *runOpts) { vo.callbackToUser = callback }
}

// WithCBSse sets the callbackToUserStreamChunkToChannel field
func WithCBSse(callback func(context.Context, *gologgers.Logger, string, chan string, int) error) RunOptFn {
	return func(vo *runOpts) { vo.callbackToUserStreamChunkToChannel = callback }
}

// WithSSEChan sets the sseChannel field
func WithSSEChan(channel chan string) RunOptFn {
	return func(vo *runOpts) { vo.sseChannel = channel }
}

func WithCallbackToChannel(callbackFN func(context.Context, *gologgers.Logger, string, int, func(string)) error) RunOptFn {
	return func(vo *runOpts) { vo.callbackToChannel = callbackFN }
}

// WithSendToChan sets the sendToChan field
func WithSendToChan(callback func(string)) RunOptFn {
	return func(vo *runOpts) { vo.sendToChan = callback }
}

func (ro *runOpts) GetChainOpts() []chains.ChainCallOption                 { return ro.chainOpts }
func (ro *runOpts) GetCallbackToUser() func(context.Context, string) error { return ro.callbackToUser }
func (ro *runOpts) GetSSEChan() chan string                                { return ro.sseChannel }
func (ro *runOpts) GetSendToChan() func(string)                            { return ro.sendToChan }
func (ro *runOpts) GetCallbackToUserStreamChunkToChannel() func(context.Context, *gologgers.Logger, string, chan string, int) error {
	return ro.callbackToUserStreamChunkToChannel
}

// NewRunOptions initializes a new voipOptions struct with the given options
func NewRunOptions(opts ...RunOptFn) *runOpts {
	vo := &runOpts{}
	for _, opt := range opts {
		opt(vo)
	}
	return vo
}

func NewChainVoipExt(ctx context.Context, llm llms.Model, log *gologgers.Logger, ws *biz.WSGetter, db *w.KnowledgeBase) *ChainVoip {
	ch := NewChainVoip(ctx, llm, log)
	ch.ws = ws
	ch.db = db

	tools.InitTools(ws, ch.db)
	return ch
}

func NewChainVoip(ctx context.Context, llm llms.Model, log *gologgers.Logger) *ChainVoip {
	ch := &ChainVoip{
		llm:    llm,
		log:    log,
		outKey: defVoipKeyOUT,
		// chainFirst:  LifecellAgentFirstNew(ctx, log, llm),
		chainFirst: Agent1(ctx, log, openaiOptsAg1),
		// chainSecond: LifecellAgentSolutionSearcher(ctx, log, llm),
		agents: map[string]chains.Chain{
			escalation_path_ag_2:  LifecellAgentSolutionSearcher(ctx, log, llm),
			escalation_path_ag_ci: NewAgentCallIssue(ctx, log),
		},
	}
	return ch
}

func CallbackSSEStreamEventMsg(ctx context.Context, log *gologgers.Logger, data string, sseChannel chan string) error {
	rec := log.RecWithCtx(ctx, "STREAM")
	rec.Info("Start callback! Data:", data)
	chunks := strings.Split(data, " ")
	rec.Infof("Chunks: %v", chunks)

	if len(chunks) == 1 {
		msgSSE, err := sse.NewEventMsg("32332", sse.EvtSQLResult, chunks[0], 3000).MakeMsgSSE(rec)
		if err != nil {
			rec.Errorf("MakeMsgSSE failed! err: %#v", err)
			return err
		}

		sseChannel <- msgSSE
	} else {
		for idx, chunk := range chunks {
			chunk += " "
			msgSSE, err := sse.NewEventMsg(strconv.Itoa(idx), sse.EvtSQLResult, chunk, 3000).MakeMsgSSE(rec)
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

// CallbackSSEStream splits a string into chunks and sends each chunk to the given SSE channel with a small delay
func _CallbackSSEStream(ctx context.Context, log *gologgers.Logger, data string, sseChannel chan string, chunkSize int) error {
	rec := log.RecWithCtx(ctx, "STREAM")
	rec.Info("Start callback! Data:", data)
	// chunks := chunkString(data, chunkSize)
	chunks := strings.Split(strconv.Quote(data), " ")
	rec.Infof("Chunks: %v", chunks)
	for _, chunk := range chunks {
		chunk += " "
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sseChannel <- chunk:
			time.Sleep(100 * time.Millisecond) // Simulate latency
		}
	}
	rec.Info("End callback!")
	return nil
}

func CallbackSSEStream(ctx context.Context, log *gologgers.Logger, data string, sseChannel chan string, chunkSize int) error {
	rec := log.RecWithCtx(ctx, "STREAM")
	rec.Info("Start callback! Data:", data)

	// Split the data into chunks of the specified size
	// chunks := _chunkString(strconv.Quote(data), chunkSize)
	chunks := strings.Split(data, "\n")
	rec.Infof("Chunks: %v", chunks)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sseChannel <- chunk:
			time.Sleep(100 * time.Millisecond) // Simulate latency
		}
	}

	rec.Info("End callback!")
	return nil
}

func CallbackSSEStreamWithFN(ctx context.Context, log *gologgers.Logger, data string, chunkSize int, sendToChan func(string)) error {
	rec := log.RecWithCtx(ctx, "STREAM")
	rec.Info("Start callback! Data:", data)

	// Split the data into chunks of the specified size
	chunks := strings.Split(data, "\n")
	rec.Infof("Chunks: %v", chunks)

	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			sendToChan(lo.If(chunk == "", " ").Else(chunk))
			time.Sleep(100 * time.Millisecond) // Simulate latency
		}
	}

	rec.Info("End callback!")
	return nil
}

func (c *ChainVoip) GetMemoryKey() string {
	return c.chainFirst.GetMemory().GetMemoryKey(context.Background())
}

func (c *ChainVoip) GetKeyIn() string {
	for _, key := range c.chainFirst.GetInputKeys() {
		if !strings.HasPrefix(key, "history") {
			return key
		}
	}
	panic("should be at least one key which will not have prefix 'history'")
}

func (c *ChainVoip) GetKeyOut() string {
	// return c.chainSecond.GetOutputKeys()[0]
	return c.outKey
}

func (c *ChainVoip) Run(ctx context.Context, values map[string]any, optsFuncs ...RunOptFn) (map[string]any, error) {
	opts := NewRunOptions(optsFuncs...)
	c.log.Info("Start Run")
	rec := c.log.RecWithCtx(ctx)

	outMap, err := chains.Call(ctx, c.chainFirst, values, opts.chainOpts...)
	if err != nil {
		return nil, err
	}
	rec.Tracef("OUT_MAP from first chain:\n%s", PrettyPrintStruct(outMap))

	if na, ok := outMap[nextAgent].(bool); ok {
		rec.Info("next_agent", na)

		isUserWantLifecellEmployee := false
		if letter, ok := outMap[nextAgent].(string); ok && letter != "" {
			isUserWantLifecellEmployee = true
		}
		rec.Infof("Explicitly call lifecell voip team => %t", isUserWantLifecellEmployee)

		if !na {
			rec.Info("Start callbackToUserStreamChunkToChannel! Chunk by chunk send response to user via SSE channel!")
			switch {
			case opts.callbackToChannel != nil:
				opts.callbackToChannel(ctx, c.log, outMap[Response].(string), 10, opts.sendToChan)
			}
			outMap[c.GetKeyOut()] = outMap[Response].(string)
			return outMap, nil
		}
	} else {
		rec.Error("next_agent not found")
	}

	escalationPath := outMap[EscalPath].(string)

	outMap[c.agents[escalationPath].GetInputKeys()[0]] = outMap[rewrittenQuery]

	resp, err := chains.Call(ctx, c.agents[escalationPath], outMap, opts.chainOpts...)
	if err != nil {
		return nil, err
	}
	rec.Tracef("OUT_MAP from second chain:\n%s", PrettyPrintStruct(resp))

	if opts.callbackToChannel != nil {
		opts.callbackToChannel(ctx, c.log, resp[c.agents[escalationPath].GetOutputKeys()[0]].(string), 10, opts.sendToChan)
	}

	// outMap[c.GetKeyOut()] = resp[FinalResponse].(string)
	outMap[c.GetKeyOut()] = resp[c.agents[escalationPath].GetOutputKeys()[0]].(string)
	outMap["agent2_result"] = resp

	return outMap, nil
}

func (c ChainVoip) Call(ctx context.Context, values map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	rec := c.log.RecWithCtx(ctx)
	outMap, err := c.chainFirst.Call(ctx, values, options...)
	if err != nil {
		return values, err
	}

	if na, ok := outMap[nextAgent].(bool); ok {
		rec.Info("next_agent", na)
		if !na {
			return outMap, nil
		}
	} else {
		rec.Error("next_agent not found")
	}

	outMap["input"] = values[c.inputKey]

	resp, err := c.agents[escalation_path_ag_2].Call(ctx, values, options...)
	if err != nil {
		return values, err
	}

	if resp == nil {
		rec.Error("resp is nil")
		return nil, nil
	}

	return resp, nil
}
