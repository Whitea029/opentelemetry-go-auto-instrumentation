package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/wikipedia"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// FakeEmbedder
var t1 = []float64{-0.19121443, 0.34286803, 0.40902084 /* ... 其他向量值 ... */}
var t2 = []float64{-0.29103878, 0.23311868, 0.119307175 /* ... 其他向量值 ... */}

type FakeEmbedder struct{}

var _ embedding.Embedder = (*FakeEmbedder)(nil)

func (f *FakeEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, errors.New("empty texts")
	}

	switch texts[0] {
	case "君不见黄河之水天上来，奔流到海不复回。":
		return [][]float64{t1}, nil
	case "君不见高堂明镜悲白发，朝如青丝暮成雪。":
		return [][]float64{t2}, nil
	default:
		return [][]float64{}, errors.New("not match texts")
	}
}

// FakeRetriever
type FakeRetriever struct{}

var _ retriever.Retriever = (*FakeRetriever)(nil)

func NewFakeRetriever() *FakeRetriever {
	return &FakeRetriever{}
}

func (f *FakeRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	return []*schema.Document{
		{
			ID:       "doc_1",
			Content:  "test_content_2",
			MetaData: map[string]any{"key": "value", "source": "fake"},
		},
		{
			ID:       "doc_2",
			Content:  "test_content_2",
			MetaData: map[string]any{"key": "value", "source": "fake"},
		},
	}, nil
}

// FakeVectorDB
type FakeVectorDB struct{}

var _ indexer.Indexer = (*FakeVectorDB)(nil)

func (f *FakeVectorDB) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	ids := make([]string, len(docs))
	for i := range docs {
		ids[i] = fmt.Sprintf("doc_%d", i)
	}
	return ids, nil
}

// newLambda component initialization function of node 'InputToQuery' in graph 'EinoAgent'
func newLambda(ctx context.Context, input *UserMessage, opts ...any) (output string, err error) {
	return input.Query, nil
}

// newLambda component initialization function of node 'ReactAgent' in graph 'EinoAgent'
func newLambda1(ctx context.Context) (lba *compose.Lambda, err error) {
	chatModel, err := NewChatModel(ctx)
	if err != nil {
		return nil, err
	}

	timeTool, err := NewWikiPediaSearch(ctx)
	if err != nil {
		return nil, err
	}

	ins, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{},
		ToolCallingModel:   chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{timeTool},
		},
	})
	if err != nil {
		return nil, err
	}

	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}

// newLambda2 component initialization function of node 'InputToHistory' in graph 'EinoAgent'
func newLambda2(ctx context.Context, input *UserMessage, opts ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// NewChatModel
func NewChatModel(ctx context.Context) (cm model.ToolCallingChatModel, err error) {
	config := &openai.ChatModelConfig{
		APIKey:  "sk-3fad43649f6947d19317b3a75f81a3e6",
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat",
	}
	cm, err = openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// BuildEinoAgent
func BuildEinoAgent(ctx context.Context) (r compose.Runnable[*UserMessage, *schema.Message], err error) {
	const (
		InputToQuery   = "InputToQuery"
		ChatTemplate   = "ChatTemplate"
		ReactAgent     = "ReactAgent"
		FakeRetriever  = "FakeRetriever"
		InputToHistory = "InputToHistory"
	)
	g := compose.NewGraph[*UserMessage, *schema.Message]()

	_ = g.AddLambdaNode(InputToQuery, compose.InvokableLambdaWithOption(newLambda), compose.WithNodeName("UserMessageToQuery"))

	chatTemplateKeyOfChatTemplate, err := newChatTemplate(ctx)
	if err != nil {
		return nil, err
	}

	_ = g.AddChatTemplateNode(ChatTemplate, chatTemplateKeyOfChatTemplate)

	reactAgentKeyOfLambda, err := newLambda1(ctx)
	if err != nil {
		return nil, err
	}

	_ = g.AddLambdaNode(ReactAgent, reactAgentKeyOfLambda, compose.WithNodeName("ReAct Agent"))

	_ = g.AddRetrieverNode(FakeRetriever, NewFakeRetriever(), compose.WithOutputKey("documents"))

	_ = g.AddLambdaNode(InputToHistory, compose.InvokableLambdaWithOption(newLambda2), compose.WithNodeName("UserMessageToVariables"))

	_ = g.AddEdge(compose.START, InputToQuery)
	_ = g.AddEdge(InputToQuery, FakeRetriever)
	_ = g.AddEdge(FakeRetriever, ChatTemplate)

	_ = g.AddEdge(compose.START, InputToHistory)
	_ = g.AddEdge(InputToHistory, ChatTemplate)

	_ = g.AddEdge(ChatTemplate, ReactAgent)
	_ = g.AddEdge(ReactAgent, compose.END)

	r, err = g.Compile(ctx, compose.WithGraphName("EinoAgent"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}

	return r, err
}

var systemPrompt = `  
You are a helpful assistant with access to tools and documents.  
  
When users ask questions:  
1. If you need current information (time, weather, news), use the available tools  
2. Reference the provided documents when relevant  
3. Always try to be helpful and use your tools when appropriate  
  
Current Date: {date}  
Available Documents: {documents}  
`

type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

// newChatTemplate component initialization function of node 'ChatTemplate' in graph 'EinoAgent'
func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	config := &ChatTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(systemPrompt),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}

type UserMessage struct {
	ID      string            `json:"id"`
	Query   string            `json:"query"`
	History []*schema.Message `json:"history"`
}

func NewWikiPediaSearch(ctx context.Context) (tn tool.InvokableTool, err error) {
	tn, err = wikipedia.NewTool(ctx, &wikipedia.Config{})
	if err != nil {
		return nil, err
	}
	return tn, nil
}
