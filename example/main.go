package main

import (
	"example/ginServer"
	"example/grpcServer"
	"example/logging"
	hello_grpc "example/pb"
	"example/syscallOperate"
	"google.golang.org/grpc"
	"mini_cmux"
	"net"
	"net/http"
	"time"
)

func main() {
	l, err := net.Listen("tcp", ":23456")
	if err != nil {
		logging.Error(err)
	}

	m := mini_cmux.New(l)
	//路由
	grpcL := m.Match(mini_cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(mini_cmux.HTTP1HeaderField("content-type", "application/json"))

	//grpc
	grpcS := grpc.NewServer()
	hello_grpc.RegisterHelloGRPCServer(grpcS, &grpcServer.Server{})
	go grpcS.Serve(grpcL)

	//http
	router := ginServer.SetupRouter()
	httpS := &http.Server{
		Handler: router,
	}
	go httpS.Serve(httpL)

	//监听关闭信号
	go syscallOperate.CloseProcess(m, grpcS, httpS)

	logging.Info("------------------------服务器启动成功------------------------")
	err = m.Serve()
	if err != nil {
		logging.Error(err)
	}

	time.Sleep(10 * time.Second)

}
