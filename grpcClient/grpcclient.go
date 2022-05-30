package main

import (
	"context"
	"fmt"
	hello_grpc "mini_cmux/pb"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:23456", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	//md := metadata2.Pairs(
	//	"content-type", "application/grpc",
	//	"accept-encoding", "identity",
	//	"grpc-accept-encoding", "identity,deflate,gzip",
	//)
	//ctx := metadata2.NewOutgoingContext(context.Background(), md)
	client := hello_grpc.NewHelloGRPCClient(conn)
	req, err := client.SayHi(context.Background(), &hello_grpc.Req{Message: "Say hi from grpc client"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(req.GetMessage())
}
