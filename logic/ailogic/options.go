package ailogic

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"gitlab.dev.ict/golang/libs/gologgers"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
	"gitlab.dev.ict/golang/go-ai/logic/ailogic/tools"
)

var (
	defCallOpts = []llms.CallOption{llms.WithMaxTokens(800), llms.WithTemperature(0.3)}
)

type Options struct {
	log            *gologgers.Logger
	name           string
	prompt         prompts.FormatPrompter
	ctx            context.Context
	tools          []llms.Tool
	toolsHandlers  map[string]tools.ToolCallHandler
	mem            schema.Memory
	memHist        schema.ChatMessageHistory
	inputKey       string
	outKey         string
	callbacks      callbacks.Handler
	outParser      schema.OutputParser[any]
	optsChain      []chains.ChainCallOption
	optsCall       []llms.CallOption
	userLogin      string
	availableTools tools.AvailableTools
	thresholdTools int
}

func (o *Options) String() string {
	return PrettyPrintStruct(o)
}

func DefaultOptions() Options {
	return Options{
		inputKey:       defInputKey,
		outKey:         defOutKey,
		callbacks:      callbackhandlers.NewLoggerHandler(gologgers.Defult()),
		outParser:      outputparser.NewSimple(),
		ctx:            context.Background(),
		thresholdTools: 3,
		optsCall:       defCallOpts,
	}
}

type OptFn func(*Options)

func WithLog(l *gologgers.Logger) OptFn                 { return func(o *Options) { o.log = l } }
func WithName(name string) OptFn                        { return func(o *Options) { o.name = name } }
func WithMem(mem schema.Memory) OptFn                   { return func(o *Options) { o.mem = mem } }
func WithMemHist(mem schema.ChatMessageHistory) OptFn   { return func(o *Options) { o.memHist = mem } }
func WithTools(tools []llms.Tool) OptFn                 { return func(o *Options) { o.tools = tools } }
func WithInputKey(inputKey string) OptFn                { return func(o *Options) { o.inputKey = inputKey } }
func WithPrompt(prompt prompts.FormatPrompter) OptFn    { return func(o *Options) { o.prompt = prompt } }
func WithOutputParse(op schema.OutputParser[any]) OptFn { return func(o *Options) { o.outParser = op } }
func WithChOpts(opts ...chains.ChainCallOption) OptFn   { return func(o *Options) { o.optsChain = opts } }
func WithCtx(ctx context.Context) OptFn                 { return func(o *Options) { o.ctx = ctx } }
func WithCallOpts(opts ...llms.CallOption) OptFn        { return func(o *Options) { o.optsCall = opts } }
func WithUserLogin(login string) OptFn                  { return func(o *Options) { o.userLogin = login } }
func WithThreshold(threshold int) OptFn                 { return func(o *Options) { o.thresholdTools = threshold } }
func WithAvailableTools(t ...tools.AvailableTool) OptFn {
	return func(o *Options) { o.availableTools = tools.NewAvailableTools(t...) }
}

func (o *Options) Log() *gologgers.Logger               { return o.log }
func (o *Options) Name() string                         { return o.name }
func (o *Options) Tools() []llms.Tool                   { return o.tools }
func (o *Options) M() schema.Memory                     { return o.mem }
func (o *Options) MHist() schema.ChatMessageHistory     { return o.memHist }
func (o *Options) IK() string                           { return o.inputKey }
func (o *Options) OK() string                           { return o.outKey }
func (o *Options) Callbacks() callbacks.Handler         { return o.callbacks }
func (o *Options) OP() schema.OutputParser[any]         { return o.outParser }
func (o *Options) ChainOpts() []chains.ChainCallOption  { return o.optsChain }
func (o *Options) Prompt() prompts.FormatPrompter       { return o.prompt }
func (o *Options) CallOpts() []llms.CallOption          { return o.optsCall }
func (o *Options) UserLogin() string                    { return o.userLogin }
func (o *Options) AvailableTools() tools.AvailableTools { return o.availableTools }
func (o *Options) TresholdTools() int                   { return o.thresholdTools }

func InitOptsChain(opts ...OptFn) *Options {
	opt := DefaultOptions()
	for _, o := range opts {
		o(&opt)
	}
	if opt.name == "" {
		panic("chain name is required")
	}
	if opt.log != nil {
		opt.callbacks = callbackhandlers.NewLoggerHandler(opt.log)
	}
	opt.ctx = AddToCtxLogin(AddToCtxCN(opt.ctx, opt.name), opt.UserLogin())
	opt.ctx = ChatForChainCtx(opt.ctx)
	opt.log.Tracef(PrettyPrintStruct(opt.ctx))
	opt.callbacks.HandleText(opt.ctx, fmt.Sprintf("Init options for chain [%s]! uuid=%v", opt.name, opt.ctx.Value(_ctx_u_cu)))

	memCreate(&opt)
	return &opt
}
