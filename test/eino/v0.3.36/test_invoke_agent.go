package main

import (
	"context"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/test/verifier"
	"github.com/cloudwego/eino/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()
	a, err := BuildEinoAgent(ctx)
	if err != nil {
		panic(err)
	}

	_, err = a.Invoke(ctx, &UserMessage{
		ID:    "2",
		Query: "搜索阿里巴巴详细信息",
		History: []*schema.Message{
			{
				Role:    schema.User,
				Content: "你好",
			},
			{
				Role:    schema.Assistant,
				Content: "你好！😊 很高兴见到你～有什么我可以帮你的吗",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	verifier.WaitAndAssertTraces(func(stubs []tracetest.SpanStubs) {
		verifier.VerifyLLMCommonAttributes(stubs[0][0], "graph", "eino", trace.SpanKindClient)
		verifier.VerifyLLMCommonAttributes(stubs[0][1], "lambda", "eino", trace.SpanKindClient)
		verifier.VerifyLLMCommonAttributes(stubs[0][3], "retriever", "eino", trace.SpanKindClient)
		verifier.VerifyLLMCommonAttributes(stubs[0][4], "prompt", "eino", trace.SpanKindClient)
		verifier.VerifyLLMAttributes(stubs[0][7], "chat", "eino", "deepseek-chat")
		verifier.VerifyLLMCommonAttributes(stubs[0][9], "tool_node", "eino", trace.SpanKindClient)
		verifier.VerifyLLMCommonAttributes(stubs[0][10], "execute_tool", "eino", trace.SpanKindClient)
		verifier.VerifyLLMAttributes(stubs[0][15], "chat", "eino", "deepseek-chat")
	}, 1)
}
