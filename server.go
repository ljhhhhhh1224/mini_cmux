package main

import (
	"net"
	"net/http"
	"time"

	"github.com/ljhhhhhh1224/mini_cmux/utils"

	mini_cmux2 "github.com/ljhhhhhh1224/mini_cmux/mini_cmux"

	"github.com/ljhhhhhh1224/mini_cmux/ginServer"
	"github.com/ljhhhhhh1224/mini_cmux/grpcServer"
	"github.com/ljhhhhhh1224/mini_cmux/logging"
	hello_grpc "github.com/ljhhhhhh1224/mini_cmux/pb"
	"github.com/ljhhhhhh1224/mini_cmux/syscallOperate"

	"google.golang.org/grpc"
)

func main() {
	l, err := net.Listen(utils.Config().Server.Network, utils.Config().Server.Port)
	if err != nil {
		logging.Error(err)
	}

	m := mini_cmux2.New(l)

	//匹配
	grpcL := m.Match(mini_cmux2.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(mini_cmux2.HTTP1HeaderField("content-type", "application/json"))

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

	logging.Info("------------------------Server started successfully------------------------")
	m.Serve()
	time.Sleep(10 * time.Second)
}
