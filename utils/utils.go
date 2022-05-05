package utils

import (
	"fmt"
	"net"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
)

// GetGrpcClientIP 获得grpc客户端的ip方法
func GetGrpcClientIP(ctx context.Context) (string, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return "", fmt.Errorf("[getClinetIP] invoke FromContext() failed")
	}
	if pr.Addr == net.Addr(nil) {
		return "", fmt.Errorf("[getClientIP] peer.Addr is nil")
	}
	addSlice := strings.Split(pr.Addr.String(), ":")
	return addSlice[0], nil
}
