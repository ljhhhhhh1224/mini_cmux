package grpcServer

import (
	"context"
	"fmt"
	"mini_cmux/logging"
	hello_grpc "mini_cmux/pb"
	"mini_cmux/syscallOperate"
	"mini_cmux/utils"
	"syscall"

	"google.golang.org/grpc"

	metadata2 "google.golang.org/grpc/metadata"
)

// Server 取出server
type Server struct {
	hello_grpc.UnimplementedHelloGRPCServer
}

// SayHi 挂载服务
func (s *Server) SayHi(ctx context.Context, req *hello_grpc.Req) (res *hello_grpc.Res, err error) {
	ip, err := utils.GetGrpcClientIP(ctx)
	md, _ := metadata2.FromIncomingContext(ctx)
	fmt.Println(md)
	if err != nil {
		logging.Error(err)
	}
	header := metadata2.Pairs("key1", "value1")
	grpc.SendHeader(ctx, header)
	logging.Info("Receive Grpc SayHi request : ", req.GetMessage(), " from ", ip)
	return &hello_grpc.Res{Message: "(GRPC)服务端响应SayHi请求"}, nil
}

func (s *Server) RequestStop(ctx context.Context, req *hello_grpc.Req) (res *hello_grpc.Res, err error) {
	ip, err := utils.GetGrpcClientIP(ctx)
	if err != nil {
		logging.Error(err)
	}
	logging.Info("Receive Grpc Stop request : ", req.GetMessage(), " from ", ip)
	syscallOperate.GetSyscallChan() <- syscall.SIGINT
	return &hello_grpc.Res{Message: "开始关闭服务器"}, nil
}
