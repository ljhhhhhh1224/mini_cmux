package grpcServer

import (
	"context"
	"log"
	"net"
	"testing"

	hello_grpc "github.com/ljhhhhhh1224/mini_cmux/pb"

	. "github.com/smartystreets/goconvey/convey"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	grpcS := grpc.NewServer()
	hello_grpc.RegisterHelloGRPCServer(grpcS, &Server{})
	go func() {
		if err := grpcS.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestServer_RequestStop(t *testing.T) {
	Convey("Test Grpc RequestStop Handler", t, func() {
		ctx := context.Background()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := hello_grpc.NewHelloGRPCClient(conn)
		resp, err := client.RequestStop(ctx, &hello_grpc.Req{Message: "Request Stop from client(UT)"})
		So(resp.GetMessage(), ShouldEqual, "Start shutting down the server")
		So(err, ShouldBeNil)
	})
}

func TestServer_SayHi(t *testing.T) {
	Convey("Test Grpc SayHi Handler", t, func() {
		ctx := context.Background()
		conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("Failed to dial bufnet: %v", err)
		}
		defer conn.Close()
		client := hello_grpc.NewHelloGRPCClient(conn)
		resp, err := client.SayHi(ctx, &hello_grpc.Req{Message: "SayHi from client(UT)"})
		So(resp.GetMessage(), ShouldEqual, "(GRPC)The server responds to the SayHi request")
		So(err, ShouldBeNil)
	})
}
