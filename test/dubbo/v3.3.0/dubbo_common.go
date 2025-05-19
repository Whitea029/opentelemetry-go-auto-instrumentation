package main

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/dubbogo/gost/log/logger"
)

type GreetTripleServer struct {
}

func (srv *GreetTripleServer) Greet(ctx context.Context, req *GreetRequest) (*GreetResponse, error) {
	resp := &GreetResponse{Greeting: req.Name}
	return resp, nil
}

func setupDubbo() {
	ins, err := dubbo.NewInstance(
		dubbo.WithName("dubbo_test_server"),
		dubbo.WithProtocol(
			protocol.WithTriple(),
			protocol.WithPort(20000),
		),
	)
	if err != nil {
		panic(err)
	}
	srv, err := ins.NewServer()
	if err != nil {
		panic(err)
	}
	if err := RegisterGreetServiceHandler(srv, &GreetTripleServer{}); err != nil {
		panic(err)
	}

	if err := srv.Serve(); err != nil {
		logger.Error(err)
	}
}

func sendDubboReq(ctx context.Context) {
	instance, err := dubbo.NewInstance(
		dubbo.WithName("dubbo_test_client"),
		dubbo.WithRegistry(
			registry.WithNacos(),
			registry.WithAddress("127.0.0.1:8848"),
		),
	)
	if err != nil {
		panic(err)
	}

	cli, err := instance.NewClient(
		client.WithClientURL("tri://127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}

	svc, err := NewGreetService(cli)
	if err != nil {
		panic(err)
	}

	resp, err := svc.Greet(ctx, &GreetRequest{Name: "hello world"})
	if err != nil {
		panic(err)
	}
	logger.Infof("Greet response: %s", resp)
}
