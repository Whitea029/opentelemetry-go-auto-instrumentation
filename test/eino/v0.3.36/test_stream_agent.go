package main

import (
	"context"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/test/verifier"
	"github.com/cloudwego/eino/schema"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func main() {
	ctx := context.Background()
	a, err := BuildEinoAgent(ctx)
	if err != nil {
		panic(err)
	}

	_, err = a.Stream(ctx, &UserMessage{
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
		// todo
	}, 1)
}
