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

	ginRouter.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	adminRoutes := ginRouter.Group("/admin")
	adminRoutes.Use(web.AuthenticateMiddleware, web.IsAdmin)
	{
		adminRoutes.GET("/register", func(c *gin.Context) {
			c.HTML(http.StatusOK, "register.html", nil)
		})

		adminRoutes.GET("/users", func(c *gin.Context) {
			c.HTML(http.StatusOK, "userList.html", nil)
		})
	}

	ginRouter.GET("/logout", func(c *gin.Context) {
		c.SetCookie("token", "", -1, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/")
	})

	apiAdminRoutes := ginRouter.Group("/api/admin")
	apiAdminRoutes.Use(web.AuthenticateMiddleware, web.IsAdmin)
	{
		apiAdminRoutes.GET("/userslist", database.UserList)
		apiAdminRoutes.POST("/register", database.Register)
		apiAdminRoutes.DELETE("/users/:id", database.DeleteUser)
	}
	ginRouter.POST("/api/login", database.Login)
	if c.Server.LogEndpoint {
		ginRouter.GET("/log", web.AuthenticateMiddleware, web.IsAdmin, web.Log)
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
