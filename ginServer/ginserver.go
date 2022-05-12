package ginServer

import (
	"mini_cmux/logging"
	"mini_cmux/syscallOperate"
	"net/http"
	"syscall"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.GET("get", get)
	router.GET("stop", stop)
	return router
}

func get(c *gin.Context) {
	logging.Info("Receive Http /get request from ", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message": "success get message",
	})
}

func stop(c *gin.Context) {
	logging.Info("Receive Http /stop request from ", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{
		"message": "success stop",
	})
	syscallOperate.GetSyscallChan() <- syscall.SIGINT
}
