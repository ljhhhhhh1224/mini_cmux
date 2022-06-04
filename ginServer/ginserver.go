package ginServer

import (
	"mini_cmux/logging"
	"mini_cmux/syscallOperate"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"
)

// SetupRouter 创建路由
func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("get", get)
	router.GET("stop", stop)
	return router
}

// get
func get(c *gin.Context) {
	logging.Info("Receive Http /get request from ", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message": "get message successfully",
	})
}

// stop 用于关闭服务器
func stop(c *gin.Context) {
	logging.Info("Receive Http /stop request from ", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message": "stop successfully",
	})
	syscallOperate.GetSyscallChan() <- syscall.SIGINT
}
