package main

import (
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"local/gintest/controllers/user"
	"local/gintest/controllers/ws"
	"local/gintest/middleware/jwt"
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

	r.GET(
		"/ws",
		jwt.GetQInstance().MiddlewareFunc(),
		ws.ServeWs,
	)

	r.POST("/register", func(c *gin.Context) {
		user.Register(c.Writer, c.Request)
	})

	r.POST("/login", jwt.GetHInstance().LoginHandler)

	auth := r.Group("/auth")

	auth.Use(jwt.GetHInstance().MiddlewareFunc())
	{
		auth.GET("/hello", jwt.HelloHandler)
		auth.GET("/refresh_token", jwt.GetHInstance().RefreshHandler)
	}

	r.Run("localhost:2021")
}
