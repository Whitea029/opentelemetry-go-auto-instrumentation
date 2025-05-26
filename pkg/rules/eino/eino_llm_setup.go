package eino

import (
	"context"
	_ "unsafe"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	utilscallbacks "github.com/cloudwego/eino/utils/callbacks"
)

//go:linkname openaiEinoNewChatModelOnEnter github.com/cloudwego/eino-ext/components/model/openai.openaiEinoNewChatModelOnEnter
func openaiEinoNewChatModelOnEnter(call api.CallContext, ctx context.Context, config *openai.ChatModelConfig) {
	handler := utilscallbacks.NewHandlerHelper().
		Graph(NewComposeHandler("graph")).Chain(NewComposeHandler("chain")).Lambda(NewComposeHandler("lambda")).
		Prompt(einoPromptCallbackHandler()).ChatModel(einoModelCallHandler(config)).Transformer(einoTransformCallbackHandler()).
		Embedding(einoEmbeddingCallbackHandler()).Indexer(einoIndexerCallbackHandler()).
		Retriever(einoRetrieverCallbackHandler()).Loader(einoLoaderCallbackHandler()).
		Tool(einoToolCallbackHandler()).ToolsNode(einoToolsNodeCallbackHandler()).
		Handler()
	callbacks.AppendGlobalHandlers(handler)
}
