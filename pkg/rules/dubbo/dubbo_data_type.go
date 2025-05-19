package dubbo

const (
	DubboServerOTelFilterKey = "DubboServerOTelFilter"
	DubboClientOTelFilterKey = "DubboClientOTelFilter"
)

type dubboRequest struct {
	methodName    string
	serviceKey    string
	serverAddress string
	attachments   map[string]any
}

// dubboResponse封装了Dubbo调用的响应信息
type dubboResponse struct {
	hasError bool
	errorMsg string
}
