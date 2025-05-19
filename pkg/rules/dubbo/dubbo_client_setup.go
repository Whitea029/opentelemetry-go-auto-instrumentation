package dubbo

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/api"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

var dubboClientInstrumenter = BuildDubboClientInstrumenter()

func init() {
	extension.SetFilter(DubboClientOTelFilterKey, func() filter.Filter {
		return &DubboClientOTelFilter{
			Propagators: otel.GetTextMapPropagator(),
		}
	})
}

func dubboNewClientOnEnter(call api.CallContext, instance *dubbo.Instance, opts ...client.ClientOption) {
	if !dubboEnabler.Enable() {
		return
	}
	opts = append(opts, client.WithClientFilter(DubboClientOTelFilterKey))
	call.SetParam(0, opts)
}

type DubboClientOTelFilter struct {
	Propagators propagation.TextMapPropagator
}

func (f *DubboClientOTelFilter) OnResponse(ctx context.Context, result protocol.Result, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	return result
}

func (f *DubboClientOTelFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if !dubboEnabler.Enable() {
		return invoker.Invoke(ctx, invocation)
	}

	req := dubboRequest{
		methodName:    invocation.MethodName(),
		serviceKey:    invoker.GetURL().ServiceKey(),
		serverAddress: invoker.GetURL().Address(),
	}

	ctx = dubboClientInstrumenter.Start(ctx, req)

	attachments := invocation.Attachments()
	if attachments == nil {
		attachments = map[string]any{}
	}
	inject(ctx, attachments, f.Propagators)
	for k, v := range attachments {
		invocation.SetAttachment(k, v)
	}

	result := invoker.Invoke(ctx, invocation)

	resp := dubboResponse{
		hasError: result.Error() != nil,
	}
	if result.Error() != nil {
		resp.errorMsg = result.Error().Error()
	}

	dubboClientInstrumenter.End(ctx, req, resp, result.Error())

	return result
}
