package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	hello_grpc "github.com/ljhhhhhh1224/mini_cmux/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:23456", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Println(err)
	}
	defer conn.Close()
	client := hello_grpc.NewHelloGRPCClient(conn)
	grpcSayHi(client)
	//grpcRequestStop(client)
	httpGet()
	//httpStop()
}

func httpGet() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:23456/get", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(body))
}

func httpStop() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "http://localhost:23456/stop", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(body))
}

func grpcRequestStop(client hello_grpc.HelloGRPCClient) {
	req, err := client.RequestStop(context.Background(), &hello_grpc.Req{Message: "Request stop from grpc client"})
	if err != nil {
		log.Println(err)
	}
	log.Println(req.GetMessage())
}

func grpcSayHi(client hello_grpc.HelloGRPCClient) {
	req, err := client.SayHi(context.Background(), &hello_grpc.Req{Message: "Say hi from grpc client"})
	if err != nil {
		log.Println(err)
	}
	log.Println(req.GetMessage())
}
