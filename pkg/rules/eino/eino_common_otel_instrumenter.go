package eino

import (
	"context"
	"fmt"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api-semconv/instrumenter/ai"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api/instrumenter"
	"github.com/alibaba/opentelemetry-go-auto-instrumentation/pkg/inst-api/version"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
)

type einoCommonAttrsGetter struct {
}

var _ ai.CommonAttrsGetter[einoRequest, any] = einoCommonAttrsGetter{}

func (einoCommonAttrsGetter) GetAIOperationName(request einoRequest) string {
	return request.operationName
}
func (einoCommonAttrsGetter) GetAISystem(request einoRequest) string {
	return "eino"
}

type LExperimentalAttributeExtractor struct {
	Base ai.AICommonAttrsExtractor[einoRequest, any, einoCommonAttrsGetter]
}

// todo
func (l LExperimentalAttributeExtractor) OnStart(attributes []attribute.KeyValue, parentContext context.Context, request einoRequest) ([]attribute.KeyValue, context.Context) {
	attributes, parentContext = l.Base.OnStart(attributes, parentContext, request)
	if request.input != nil {
		var val attribute.Value
		for k, v := range request.input {
			switch v.(type) {
			case string:
				val = attribute.StringValue(v.(string))
			case int:
				val = attribute.IntValue(v.(int))
			case int64:
				val = attribute.Int64Value(v.(int64))
			case float64:
				val = attribute.Float64Value(v.(float64))
			case bool:
				val = attribute.BoolValue(v.(bool))
			default:
				val = attribute.StringValue(fmt.Sprintf("%#v", v))
			}
			if val.Type() > 0 {
				attributes = append(attributes, attribute.KeyValue{
					Key:   attribute.Key("gen_ai.other_input." + k),
					Value: val,
				})
			}
			val = attribute.Value{}
		}

	}
	return attributes, parentContext
}

func (l LExperimentalAttributeExtractor) OnEnd(attributes []attribute.KeyValue, context context.Context, request einoRequest, response any, err error) ([]attribute.KeyValue, context.Context) {
	attributes, context = l.Base.OnEnd(attributes, context, request, response, err)
	if request.output != nil {
		var val attribute.Value
		for k, v := range request.output {
			switch v.(type) {
			case string:
				val = attribute.StringValue(v.(string))
			case int:
				val = attribute.IntValue(v.(int))
			case int64:
				val = attribute.Int64Value(v.(int64))
			case float64:
				val = attribute.Float64Value(v.(float64))
			case bool:
				val = attribute.BoolValue(v.(bool))
			default:
				val = attribute.StringValue(fmt.Sprintf("%#v", v))
			}
			if val.Type() > 0 {
				attributes = append(attributes, attribute.KeyValue{
					Key:   attribute.Key("gen_ai.other_output." + k),
					Value: val,
				})
			}
			val = attribute.Value{}
		}

	}
	return attributes, context
}

func BuildEinoCommonInstrumenter() instrumenter.Instrumenter[einoRequest, any] {
	builder := instrumenter.Builder[einoRequest, any]{}
	return builder.Init().SetSpanNameExtractor(&ai.AISpanNameExtractor[einoRequest, any]{Getter: einoCommonAttrsGetter{}}).
		SetSpanKindExtractor(&instrumenter.AlwaysClientExtractor[einoRequest]{}).
		AddAttributesExtractor(&LExperimentalAttributeExtractor{}).
		SetInstrumentationScope(instrumentation.Scope{
			Name:    "pkg/rules/eino/setup.go", //todo
			Version: version.Tag,
		}).
		BuildInstrumenter()
}
