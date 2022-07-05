package grpcServer

import (
	"context"
	"syscall"

	"github.com/ljhhhhhh1224/mini_cmux/logging"
	hello_grpc "github.com/ljhhhhhh1224/mini_cmux/pb"
	"github.com/ljhhhhhh1224/mini_cmux/syscallOperate"
	"github.com/ljhhhhhh1224/mini_cmux/utils"
)

// Server 取出server
type Server struct {
	hello_grpc.UnimplementedHelloGRPCServer
}

// SayHi 挂载服务
func (s *Server) SayHi(ctx context.Context, req *hello_grpc.Req) (res *hello_grpc.Res, err error) {
	ip, err := utils.GetGrpcClientIP(ctx)
	if err != nil {
		logging.Error(err)
		return
	}
	logging.Info("Receive Grpc SayHi request : ", req.GetMessage(), " from ", ip)
	return &hello_grpc.Res{Message: "(GRPC)The server responds to the SayHi request"}, nil
}

func (s *Server) RequestStop(ctx context.Context, req *hello_grpc.Req) (res *hello_grpc.Res, err error) {
	ip, err := utils.GetGrpcClientIP(ctx)
	if err != nil {
		logging.Error(err)
		return
	}
	logging.Info("Receive Grpc Stop request : ", req.GetMessage(), " from ", ip)
	syscallOperate.GetSyscallChan() <- syscall.SIGINT
	return &hello_grpc.Res{Message: "Start shutting down the server"}, nil
}
