package llm

import (
	"net/http"
	"net/url"
	"os"

	"github.com/samber/lo"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"gitlab.dev.ict/golang/libs/gohttp"
	"gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gologgers/applogger"
	"gitlab.dev.ict/golang/libs/utils"

	"gitlab.dev.ict/golang/go-ai/logic/ailogic/callbackhandlers"
)

const SslPathOracleLinux = "/etc/ssl/certs/ca-bundle.crt"
const SslPath = "/etc/ssl/certs/ca-certificates.crt"
const GPT_4o = "gpt-4o"
const GPT_4 = "gpt-4"
const TXT_EMB_L = "text-embedding-3-large"
const TXT_EMB_S = "text-embedding-3-small"

var respFormatText = openai.ResponseFormat{Type: "text"}

func HttpClient(log any, to int, certPath string, isv bool, withPrx ...bool) *http.Client {
	cl := gohttp.New().WithTimeout(lo.Ternary(to == 0, 360, to))
	switch l := log.(type) {
	case *gologgers.Logger:
		cl.WithLogger(l)
	case *applogger.LogCfg:
		cl.WithLogCfg(l)
	default:
		panic("unknown LOGGER")
	}

	if isv {
		cl = cl.WithISV()
	}
	if certPath != "" {
		cl = cl.WithSSLPath(certPath)
	}


	return cl.WithProxy(processProxy(withPrx...)).Build().Client
}

func processProxy(withPrx ...bool) func(*http.Request) (*url.URL, error) {
	if utils.FirstOrDefault(true, withPrx...) {
		return gohttp.ProxyAstelit
	}
	return nil
}

func GroqAI(log *gologgers.Logger, httpClient *http.Client) llms.Model {
	llm, e := openai.New(
		openai.WithToken(os.Getenv("GROQ_API_KEY")),
		openai.WithBaseURL("https://api.groq.com/openai/v1"),
		openai.WithModel("llama3-70b-8192"),
		openai.WithHTTPClient(httpClient),
		openai.WithCallback(callbackhandlers.NewLoggerHandler(log)),
		openai.WithResponseFormat(openai.ResponseFormatJSON),
	)
	if e != nil {
		panic(e)
	}
	return llm
}

func OpenAiDefault(log *gologgers.Logger, withPrx ...bool) llms.Model {
	return OpenAI(GPT_4o, log, HttpClient(log, 300, SslPath, false, withPrx...))
}

func OpenAI(model string, log *gologgers.Logger, httpClient *http.Client) llms.Model {
	llm, e := openai.New(
		openai.WithModel(model),
		openai.WithHTTPClient(httpClient),
		openai.WithCallback(callbackhandlers.NewLoggerHandler(log)),
	)
	if e != nil {
		panic(e)
	}
	return llm
}
