package main

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"local/gintest/controllers/ws"
	"local/gintest/services/pid"
	"local/gintest/wslogic"
)

func main() {

	wslogic.Init()
	pid.Init()

	r := gin.Default()
	r.Use(static.Serve("/public", static.LocalFile("./public", true)))
	r.LoadHTMLFiles("index.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	r.GET("/ws", func(c *gin.Context) {
		//wshandler(c.Writer, c.Request)
		controllers.ServeWs(c.Writer, c.Request)
	})

	r.Run("localhost:2021")
}
