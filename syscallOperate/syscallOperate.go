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
	logging.Info("------------------------Start a graceful server shutdown------------------------")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := s.Shutdown(ctx); err != nil {
		logging.Error("Server Shutdown : ", err)
	}
	defer cancel()
	logging.Info("------------------------The http service has been exited smoothly------------------------")
	g.GracefulStop()
	logging.Info("------------------------grpc service has exited smoothly------------------------")
	m.Close()
	logging.Info("------------------------Closed successfully------------------------")
}

func init() {
	c = make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
}
