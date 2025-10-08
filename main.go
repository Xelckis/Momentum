package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"Momentum/internal/config"
	"Momentum/internal/database"
	"Momentum/internal/web"

	"github.com/gin-gonic/gin"
)

func main() {
	c := config.Config{}
	c.LoadConfig()
	conn := database.ConnectDB()
	defer conn.Close()

	gin.DisableConsoleColor()
	f, _ := os.OpenFile("var/log/gin.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	ginRouter := gin.New()
	ginRouter.LoadHTMLGlob("templates/*")
	serverHTTPS := config.LoadServerHTTPSConfig(c, ginRouter)

	ginRouter.GET("/hello", web.Hello)
	ginRouter.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	ginRouter.POST("/api/login", database.Login)
	if c.Server.LogEndpoint {
		ginRouter.GET("/log", web.AuthenticateMiddleware, web.Log)
		ginRouter.GET("/ws/log", web.WsLog)
	}

	if c.Server.RedirectToHttps {
		go config.LoadHTTPServer(c)
	}

	err := serverHTTPS.ListenAndServeTLS(c.Security.CertFile, c.Security.KeyFile)
	if err != nil {
		log.Panicln(err)
	}

}
