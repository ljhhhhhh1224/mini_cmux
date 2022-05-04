package ginServer

import (
	"github.com/gin-gonic/gin"
	"example/logging"
	"example/syscallOperate"
	"net/http"
	"syscall"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("get", func(c *gin.Context){
		logging.Info("Receive Http /get request from ",c.ClientIP())
		c.JSON(http.StatusOK,gin.H{
			"message" : "success get message",
		})
	})

	router.GET("stop", func(c *gin.Context) {
		logging.Info("Receive Http /stop request from ",c.ClientIP() )
		c.JSON(http.StatusOK,gin.H{
			"message" : "success stop",
		})
		syscallOperate.GetSyscallChan() <- syscall.SIGINT
	})
	return router
}
