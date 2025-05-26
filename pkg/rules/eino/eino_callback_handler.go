package eino

import (
	"context"
	"fmt"
	"io"
	"log"
	"runtime/debug"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	utilscallbacks "github.com/cloudwego/eino/utils/callbacks"
)

func einoPromptCallbackHandler() *utilscallbacks.PromptCallbackHandler {
	return &utilscallbacks.PromptCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *prompt.CallbackInput) context.Context {
			request := einoRequest{operationName: "prompt"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *prompt.CallbackOutput) context.Context {
			request := einoRequest{operationName: "prompt"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "prompt"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoModelCallHandler(config *openai.ChatModelConfig) *utilscallbacks.ModelCallbackHandler {
	return &utilscallbacks.ModelCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			request := einoLLMRequest{
				operationName: "chat",
				stopSequences: config.Stop,
				serverAddress: config.BaseURL,
			}
			if config.FrequencyPenalty != nil {
				request.frequencyPenalty = float64(*config.FrequencyPenalty)
			}
			if config.PresencePenalty != nil {
				request.presencePenalty = float64(*config.PresencePenalty)
			}
			if config.MaxTokens != nil {
				request.maxTokens = int64(*config.MaxTokens)
			}
			if config.Seed != nil {
				request.seed = int64(*config.Seed)
			}
			if config.BaseURL != "" {
			}
			if config.Temperature != nil {
				request.temperature = float64(*config.Temperature)
			} else {
				request.temperature = 1
			}
			if config.TopP != nil {
				request.topP = float64(*config.TopP)
			} else {
				request.topP = 1
			}
			if config.Model != "" {
				request.modelName = config.Model
			} else {
				request.modelName = "UnknownModel"
			}
			ctx = einoLLMInstrument.Start(ctx, request)
			return context.WithValue(ctx, EinoLLMRequestKey, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			request := ctx.Value(EinoLLMRequestKey).(einoLLMRequest)
			response := einoLLMResponse{
				usageOutputTokens:     int64(output.TokenUsage.TotalTokens),
				responseModel:         output.Config.Model,
				responseFinishReasons: []string{output.Message.ResponseMeta.FinishReason},
			}
			einoLLMInstrument.End(ctx, request, response, nil)
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, runInfo *callbacks.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
			go func() {
				defer func() {
					err := recover()
					if err != nil {
						log.Printf("recover update langfuse span panic: %v, runinfo: %+v, stack: %s", err, runInfo, string(debug.Stack()))
					}
					output.Close()
				}()
				request := ctx.Value(EinoLLMRequestKey).(einoLLMRequest)
				response := einoLLMResponse{}
				var outs []*model.CallbackOutput
				for {
					chunk, err := output.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						log.Printf("read stream output error: %v, runinfo: %+v", err, runInfo)
					}
					outs = append(outs, chunk)
				}

				usage, outMessage, _, err := extractModelOutput(outs)
				if err != nil {
					log.Printf("extract model output error: %v, runinfo: %+v", err, runInfo)
					einoLLMInstrument.End(ctx, request, response, err)
					return
				}
				response.usageOutputTokens = int64(usage.TotalTokens)
				response.responseModel = request.modelName
				response.responseFinishReasons = []string{outMessage.ResponseMeta.FinishReason}
				einoLLMInstrument.End(ctx, request, response, nil)
			}()
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := ctx.Value(EinoLLMRequestKey).(einoLLMRequest)
			response := einoLLMResponse{}
			einoLLMInstrument.End(ctx, request, response, nil)
			return ctx
		},
	}
}

func extractModelOutput(outs []*model.CallbackOutput) (usage *model.TokenUsage, message *schema.Message, extra map[string]interface{}, err error) {
	var mas []*schema.Message
	for _, out := range outs {
		if out == nil {
			continue
		}
		if out.TokenUsage != nil {
			usage = out.TokenUsage
		}
		if out.Message != nil {
			mas = append(mas, out.Message)
		}
		if out.Extra != nil {
			extra = out.Extra
		}
	}
	if len(mas) == 0 {
		return usage, &schema.Message{}, extra, nil
	}
	message, err = schema.ConcatMessages(mas)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("concat message failed: %v", err)
	}
	return usage, message, extra, nil
}

func einoEmbeddingCallbackHandler() *utilscallbacks.EmbeddingCallbackHandler {
	return &utilscallbacks.EmbeddingCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *embedding.CallbackInput) context.Context {
			request := einoRequest{operationName: "embedding"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *embedding.CallbackOutput) context.Context {
			request := einoRequest{operationName: "embedding"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "embedding"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoIndexerCallbackHandler() *utilscallbacks.IndexerCallbackHandler {
	return &utilscallbacks.IndexerCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *indexer.CallbackInput) context.Context {
			request := einoRequest{operationName: "indexer"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *indexer.CallbackOutput) context.Context {
			request := einoRequest{operationName: "indexer"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "indexer"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoRetrieverCallbackHandler() *utilscallbacks.RetrieverCallbackHandler {
	return &utilscallbacks.RetrieverCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *retriever.CallbackInput) context.Context {
			request := einoRequest{operationName: "retriever"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *retriever.CallbackOutput) context.Context {
			request := einoRequest{operationName: "retriever"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "retriever"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoLoaderCallbackHandler() *utilscallbacks.LoaderCallbackHandler {
	return &utilscallbacks.LoaderCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *document.LoaderCallbackInput) context.Context {
			request := einoRequest{operationName: "loader"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *document.LoaderCallbackOutput) context.Context {
			request := einoRequest{operationName: "loader"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "loader"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoToolCallbackHandler() *utilscallbacks.ToolCallbackHandler {
	return &utilscallbacks.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			request := einoRequest{
				operationName: "execute_tool",
				input: map[string]interface{}{
					"tool":    info.Name,
					"tool-id": compose.GetToolCallID(ctx),
				},
			}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			request := einoRequest{operationName: "execute_tool"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*tool.CallbackOutput]) context.Context {
			request := einoRequest{operationName: "execute_tool"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "execute_tool"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoToolsNodeCallbackHandler() *utilscallbacks.ToolsNodeCallbackHandlers {
	return &utilscallbacks.ToolsNodeCallbackHandlers{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *schema.Message) context.Context {
			request := einoRequest{
				operationName: "tool_node",
				input: map[string]interface{}{
					"tool-node":    info.Name,
					"tool-node-id": input.ToolCallID,
				},
			}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, input []*schema.Message) context.Context {
			request := einoRequest{operationName: "tool_node"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[[]*schema.Message]) context.Context {
			request := einoRequest{operationName: "tool_node"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "tool_node"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

func einoTransformCallbackHandler() *utilscallbacks.TransformerCallbackHandler {
	return &utilscallbacks.TransformerCallbackHandler{
		OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *document.TransformerCallbackInput) context.Context {
			request := einoRequest{operationName: "transform"}
			return einoCommonInstrument.Start(ctx, request)
		},
		OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *document.TransformerCallbackOutput) context.Context {
			request := einoRequest{operationName: "transform"}
			einoCommonInstrument.End(ctx, request, nil, nil)
			return ctx
		},
		OnError: func(ctx context.Context, runInfo *callbacks.RunInfo, err error) context.Context {
			request := einoRequest{operationName: "transform"}
			einoCommonInstrument.End(ctx, request, nil, err)
			return ctx
		},
	}
}

type ComposeHandler struct {
	operationName string
}

var _ callbacks.Handler = ComposeHandler{}

func NewComposeHandler(operationName string) *ComposeHandler {
	return &ComposeHandler{
		operationName: operationName,
	}
}

func (c ComposeHandler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	request := einoRequest{operationName: c.operationName}
	return einoCommonInstrument.Start(ctx, request)
}

func (c ComposeHandler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	request := einoRequest{operationName: c.operationName}
	einoCommonInstrument.End(ctx, request, nil, nil)
	return ctx
}

func (c ComposeHandler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	request := einoRequest{operationName: c.operationName}
	einoCommonInstrument.End(ctx, request, nil, err)
	return ctx
}

// todo
func (c ComposeHandler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	request := einoRequest{operationName: c.operationName}
	return einoCommonInstrument.Start(ctx, request)
}

func (c ComposeHandler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	request := einoRequest{operationName: c.operationName}
	einoCommonInstrument.End(ctx, request, nil, nil)
	return ctx
}
