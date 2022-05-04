package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	hello_grpc "example/pb"
)

func main(){
	conn,_ := grpc.Dial("localhost:23456",grpc.WithInsecure())
	client := hello_grpc.NewHelloGRPCClient(conn)
	res,_ := client.SayHi(context.Background(),&hello_grpc.Req{Message: "你好呀,我是客户端"})
	fmt.Println(res.GetMessage())
}
