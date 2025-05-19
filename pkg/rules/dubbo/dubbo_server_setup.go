package dubbo

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	extension.SetFilter(DubboServerOTelFilterKey, func() filter.Filter {
		return &DubboServerOTelFilter{
			Propagators: otel.GetTextMapPropagator(),
		}
	})
}

var dubboServerInstrumenter = BuildDubboServerInstrumenter()

func dubboNewServerOnEnter(call api.CallContext, instance *dubbo.Instance, opts ...server.ServerOption) {
	if !dubboEnabler.Enable() {
		return
	}
	opts = append(opts, server.WithServerFilter(DubboServerOTelFilterKey))
	call.SetParam(0, opts)
}

type DubboServerOTelFilter struct {
	Propagators propagation.TextMapPropagator
}

func (f *DubboServerOTelFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if !dubboEnabler.Enable() {
		return invoker.Invoke(ctx, invocation)
	}

	attachments := invocation.Attachments()
	bags, spanCtx := extract(ctx, attachments, f.Propagators)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	if spanCtx.IsValid() {
		ctx = trace.ContextWithRemoteSpanContext(ctx, spanCtx)
	}

	req := dubboRequest{
		methodName:    invocation.MethodName(),
		serviceKey:    invoker.GetURL().ServiceKey(),
		serverAddress: invoker.GetURL().Address(),
		attachments:   attachments,
	}

	ctx = dubboServerInstrumenter.Start(ctx, req)

	result := invoker.Invoke(ctx, invocation)

	resp := dubboResponse{
		hasError: result.Error() != nil,
	}
	if result.Error() != nil {
		resp.errorMsg = result.Error().Error()
	}

	dubboServerInstrumenter.End(ctx, req, resp, result.Error())

	return result
}

func (f *DubboServerOTelFilter) OnResponse(ctx context.Context, res protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return res
}
