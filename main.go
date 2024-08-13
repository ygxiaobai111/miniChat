package main

import (
	gin "github.com/gin-gonic/gin"
	"miniChat/service"
)

func main() {
	g := gin.Default()
	go func() {
		service.Manager.HandleWebSocketEvents()
	}()
	g.GET("ws", service.ChatHandler)
	g.Run(":8080")
}
