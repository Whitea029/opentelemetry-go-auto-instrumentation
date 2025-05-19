package main

import (
	"context"
	"time"

	"github.com/alibaba/opentelemetry-go-auto-instrumentation/test/verifier"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func main() {
	// starter server
	go setupDubbo()
	time.Sleep(3 * time.Second)
	// use a client to request to the server
	sendDubboReq(context.Background())
	// verify trace
	verifier.WaitAndAssertTraces(func(stubs []tracetest.SpanStubs) {
		verifier.VerifyRpcClientAttributes(stubs[0][0], "/GreetService/Greet", "apache_dubbo", "/GreetService", "Greet")
		verifier.VerifyRpcServerAttributes(stubs[0][1], "/GreetService/Greet", "apache_dubbo", "/GreetService", "Greet")
	}, 1)
}
