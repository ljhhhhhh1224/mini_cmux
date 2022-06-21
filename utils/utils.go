package utils

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"

	"golang.org/x/net/context"
	"google.golang.org/grpc/peer"
)

type config struct {
	Debug  bool
	Server struct {
		Port    string
		Network string
	}
	Client struct {
		IP   string
		Port string
	}
}

var (
	cfg  *config
	once sync.Once
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

func Config() *config {
	once.Do(func() {
		filePath, err := filepath.Abs("./conf/config.toml")
		if err != nil {
			panic(err)
		}
		if _, err := toml.DecodeFile(filePath, &cfg); err != nil {
			panic(err)
		}
	})
	return cfg
}
