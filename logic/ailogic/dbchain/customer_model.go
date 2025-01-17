package dbchain

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"gitlab.dev.ict/golang/libs/gologgers"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic"
	lifetools "gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
	dic "gitlab.dev.ict/golang/go-ai/logic/db_info_collector"
)

var (
	sysPromptDB = prompts.NewSystemMessagePromptTemplate(defSysPrompt_db, []string{"db_type", "db_structure"})

	defPrompt = prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		// prompts.NewSystemMessagePromptTemplate(defSysPrompt_db, []string{"db_type", "db_structure"}),
		prompts.NewSystemMessagePromptTemplate(defSysPrompt_db, nil),
		prompts.MessagesPlaceholder{VariableName: KeyHistory},
		prompts.NewHumanMessagePromptTemplate(PromptTmplUser, []string{KeyInput}),
	})
)

var ErrorDbStruct = errors.New("error db struct is not provided! Please provide db struct or table pattern")

// type Options struct {
// 	log       *gologgers.Logger
// 	name      string
// 	tools     []llms.Tool
// 	mem       schema.Memory
// 	memHist   schema.ChatMessageHistory
// 	inputKey  string
// 	outKey    string
// 	callbacks callbacks.Handler
// 	outParser schema.OutputParser[any]
// 	optsChain []chains.ChainCallOption
// 	prompt    prompts.FormatPrompter
// }
// type OptFn func(*Options)
// func WithName(name string) OptFn                        { return func(o *Options) { o.name = name } }
// func WithMem(mem schema.Memory) OptFn                   { return func(o *Options) { o.mem = mem } }
// func WithMemHist(mem schema.ChatMessageHistory) OptFn   { return func(o *Options) { o.memHist = mem } }
// func WithTools(tools []llms.Tool) OptFn                 { return func(o *Options) { o.tools = tools } }
// func WithInputKey(inputKey string) OptFn                { return func(o *Options) { o.inputKey = inputKey } }
// func WithPrompt(prompt prompts.FormatPrompter) OptFn    { return func(o *Options) { o.prompt = prompt } }
// func WithOutputParse(op schema.OutputParser[any]) OptFn { return func(o *Options) { o.outParser = op } }
// func WithChOpts(opts ...chains.ChainCallOption) OptFn   { return func(o *Options) { o.optsChain = opts } }
// func DefaultOptions() Options {
// 	return Options{
// 		// mem:       memory.NewConversationBuffer(memory.WithOutputKey(defOutKey), memory.WithChatHistory(CreateSqliteMem(utils.GenerateCtxWithRid(), "default"))),
// 		inputKey:  defInputKey,
// 		outKey:    defOutKey,
// 		callbacks: callbackhandlers.NewLoggerHandler(gologgers.Defult()),
// 		outParser: outputparser.NewSimple(),
// 	}
// }
// func memCreate(o *Options) {
// 	switch {
// 	case o.mem != nil:
// 	case o.memHist != nil:
// 		o.mem = memory.NewConversationBuffer(memory.WithInputKey(o.inputKey), memory.WithOutputKey(o.outKey), memory.WithChatHistory(o.memHist))
// 	default:
// 		o.mem = memory.NewConversationBuffer(memory.WithInputKey(o.inputKey), memory.WithOutputKey(o.outKey), memory.WithChatHistory(CreateSqliteMem(utils.GenerateCtxWithRid(), o.name)))
// 	}
// }
// func CreateSqliteMem(ctx context.Context, chainName string) schema.ChatMessageHistory {
// 	return sqlite3.NewSqliteChatMessageHistory(
// 		sqlite3.WithSession(ailogic.GetUUIDAI(ctx)), sqlite3.WithDB(ailogic.DBMemory()), sqlite3.WithContext(ctx), sqlite3.WithTableName(chainName),
// 	)
// }

type DbChain struct {
	*chains.LLMChain
	ChainName  string
	DBProvider dic.DBSchemaInfoProvider
	log        *gologgers.Logger
	inputKey   string
	tools      lifetools.AvailableTools
	opts       []llms.CallOption
}

func (c *DbChain) String() string {
	return fmt.Sprintf("DbChain ==> name=%s, inputKey=%s, outputKey=%s, memKey=%s, loglvl=%s, tools=%v, opts=%v", c.ChainName, c.inputKey, c.OutputKey, c.GetMemory().GetMemoryKey(context.Background()), c.log.Options.LogLevel, c.tools, c.opts)
}

// DbChainNew is a constructor for DbChain
//
//	FOR CHAT_HISTORY for continue conversation:
//		a) opts.WithName("DbChain"), opts.WithUserLogin("login_user"), al.WithCtx(al.AddToCtxUUIDAI(ctx, "066fd096-a5fe-4a66-9c5c-1bb644e53545"))
//		b) opts.WithName("DbChain"),, al.WithCtx(al.AddToCtxUUIDAI(ctx, "066fd096-a5fe-4a66-9c5c-1bb644e53545"))
func DbChainNew(l *gologgers.Logger, llm llms.Model, dbProvider dic.DBSchemaInfoProvider, opts ...ailogic.OptFn) *DbChain {
	opts = append(opts, ailogic.WithLog(l))
	opt := ailogic.InitOptsChain(opts...)
	opt.M().(*memory.ConversationBuffer).ReturnMessages = true

	defCallOpts := []llms.CallOption{llms.WithMaxTokens(800), llms.WithTemperature(0.15)}

	c := &DbChain{
		LLMChain: &chains.LLMChain{
			LLM:              llm,
			CallbacksHandler: opt.Callbacks(),
			OutputParser:     opt.OP(),
			OutputKey:        opt.OK(),
			Memory:           opt.M(),
			Prompt:           opt.Prompt(),
		},
		log:        l,
		ChainName:  opt.Name(),
		DBProvider: dbProvider,
		tools:      opt.AvailableTools(),
		inputKey:   opt.IK(),
		opts: lo.TernaryF(len(opt.CallOpts()) > 0,
			func() []llms.CallOption { return opt.CallOpts() },
			func() []llms.CallOption { return defCallOpts }),
	}
	if opt.AvailableTools() != nil {
		l.Info("Tools are available!")
		c.opts = append(c.opts, llms.WithTools(opt.AvailableTools().GetTools()))
	}
	if opt.Prompt() == nil {
		c.Prompt = defPrompt
	}
	l.Info(c)
	return c
}

func (c *DbChain) Run(ctx context.Context, inputs map[string]any, optsFuncs ...ailogic.RunOptFn) (map[string]any, error) {
	opts := ailogic.NewRunOptions(optsFuncs...)
	c.log.Info("Start Run")
	rec := c.log.RecWithCtx(ctx)

	// outMap, err := c.chainFirst.Call(ctx, values, opts.chainOptions...)
	outMap, err := chains.Call(ctx, c, inputs, opts.GetChainOpts()...)
	if err != nil {
		return nil, err
	}
	rec.Tracef("OUT_MAP from chain:\n%s", ailogic.PrettyPrintStruct(outMap))

	rec.Info("Start callbackToUserStreamChunkToChannel! Chunk by chunk send response to user via SSE channel!")
	if opts.GetCallbackToUserStreamChunkToChannel() != nil {
		opts.GetCallbackToUserStreamChunkToChannel()(ctx, c.log, outMap[c.OutputKey].(string), opts.GetSSEChan(), 10)
	}
	return outMap, nil
}
func (c *DbChain) Call(ctx context.Context, inputs map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
	rec := c.log.RecWithCtx(ctx, c.ChainName)
	rec.Infof("Start chain '%s'! memKey=%s output:[%v] expected_input:[%v]; Provided variables: %#v", c.ChainName, c.GetMemory().GetMemoryKey(ctx), c.GetOutputKeys(), c.Prompt.GetInputVariables(), ailogic.Keys(inputs))

	// rec.Trace(ailogic.PrettyPrintStruct(inputs))

	if _, ok := inputs[KeyOutFmt]; !ok {
		inputs[KeyOutFmt] = ""
	}

	var err error
	var promptValue llms.PromptValue
	var isHistoryExists bool

	if _, ok := inputs[c.Memory.GetMemoryKey(ctx)]; !ok {
		inputs[c.Memory.GetMemoryKey(ctx)] = []llms.ChatMessage{}
	}

	if hist := inputs[c.Memory.GetMemoryKey(ctx)].([]llms.ChatMessage); len(hist) > 0 {
		isHistoryExists = true
		hist = append(hist, llms.HumanChatMessage{Content: inputs[c.inputKey].(string)})
		promptValue = prompts.ChatPromptValue(hist)
	}

	rec.Infof("isHistoryExists=[%t] len(history)=%d input[TblPattern]=%v input[DbStruct]=%v", isHistoryExists, len(inputs[KeyHistory].([]llms.ChatMessage)), inputs[KeyTblPattern], inputs[KeyDbStruct])
	if !isHistoryExists {
		if _, ok := inputs[KeyDbStruct]; !ok {
			if _, ok := inputs[KeyTblPattern].(string); !ok {
				rec.Error(ErrorDbStruct)
				return nil, ErrorDbStruct
			}
			inputs[KeyDbStruct] = c.DBProvider.PrepareDBInfoForLLM(ctx, []string{inputs[KeyTblPattern].(string)}...)
		}

		if _, ok := inputs[KeyDbType]; !ok {
			inputs[KeyDbType] = DB_TYPE_ORACLE
		}

		if promptValue, err = c.Prompt.FormatPrompt(inputs); err != nil {
			rec.Errorf("incorrect inputs!", err)
			return nil, err
		}
	}
	// rec.Trace(promptValue)

	rec.Info("Call LLM GenerateContent...")
	callOpts := lo.TernaryF(len(c.opts) > 0, func() []llms.CallOption { return c.opts }, func() []llms.CallOption {
		return []llms.CallOption{llms.WithMaxTokens(800), llms.WithTemperature(0.15)}
	})
	resp, err := c.LLM.GenerateContent(ctx, ailogic.MessageContentFromChat(promptValue), callOpts...)
	// resp, err := &llms.ContentResponse{Choices: []*llms.ContentChoice{{Content: "XXX"}}}, err
	if err != nil {
		rec.Fatal(err)
	}

	if resp.Choices[0].StopReason == "tool_calls" {
		rec.Info("Tool calls detected!")
		// return ExecuteToolCalls(rec, c.tools, promptValue, resp, true, c.GetConversation().ChatHistory)
		messages, err := lifetools.ExecuteToolCalls(rec, c.tools.GetMapCallHandlers(), []llms.MessageContent{}, resp, true, c.GetConversation().ChatHistory)
		if err != nil {
			rec.Errorf("Error executing tool calls: %v", err)
			return nil, err
		}
		rec.Infof("Tool calls executed! messages=%v", messages[len(messages)-1].Parts[0])
		return map[string]any{c.OutputKey: messages[len(messages)-1].Parts[0].(llms.ToolCallResponse).Content}, nil
	}

	outMap := map[string]any{c.OutputKey: resp.Choices[0].Content}

	if !isHistoryExists {
		c.GetConversation().ChatHistory.AddMessage(ctx, llms.SystemChatMessage{Content: promptValue.Messages()[0].GetContent()})
	}

	return outMap, nil
}

func (c *DbChain) GetConversation() *memory.ConversationBuffer {
	return c.Memory.(*memory.ConversationBuffer)
}

func (c *DbChain) GetKeyIn() string {
	return c.inputKey
}

func PrepareInputsCM(userPrompt string) map[string]any {
	return map[string]interface{}{KeyTblPattern: regExpTblsCustomerModel, KeyInput: userPrompt, KeyHistory: []llms.ChatMessage{}}
}
