package syscallOperate

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mini_cmux2 "github.com/ljhhhhhh1224/mini_cmux/mini_cmux"

	"github.com/ljhhhhhh1224/mini_cmux/logging"

	"google.golang.org/grpc"
)

var c chan os.Signal

func GetSyscallChan() chan os.Signal {
	return c
}

func CloseProcess(m mini_cmux2.CMux, g *grpc.Server, s *http.Server) {
	<-c
	logging.Info("------------------------开始平滑关闭服务器------------------------")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := s.Shutdown(ctx); err != nil {
		logging.Error("Server Shutdown : ", err)
	}
	defer cancel()
	logging.Info("------------------------http服务已平滑退出------------------------")
	g.GracefulStop()
	logging.Info("------------------------grpc服务已平滑退出------------------------")
	m.Close()
	logging.Info("------------------------关闭成功------------------------")
}

func init() {

	c = make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
}
