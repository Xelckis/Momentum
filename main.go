package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"Momentum/internal/config"
	"Momentum/internal/database"
	"Momentum/internal/logger"
	"Momentum/internal/web"

	"github.com/gin-gonic/gin"
)

func main() {
	c := config.Config{}
	c.LoadConfig()
	conn := database.ConnectDB()
	defer conn.Close()

	logger.LogFileWriter = setupLogging()

	ginRouter := setupGin()
	serverHTTPS := config.LoadServerHTTPSConfig(c, ginRouter)

	registerRoutes(ginRouter, c)

	if c.Server.RedirectToHttps {
		go config.LoadHTTPServer(c)
	}

	err := serverHTTPS.ListenAndServeTLS(c.Security.CertFile, c.Security.KeyFile)
	if err != nil {
		log.Panicln(err)
	}

}

func setupLogging() io.Writer {
	gin.DisableConsoleColor()
	f, _ := os.OpenFile("var/log/gin.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	return gin.DefaultWriter
}

func setupGin() *gin.Engine {
	ginRouter := gin.New()
	ginRouter.LoadHTMLGlob("templates/*")
	ginRouter.Static("/uploads", "./uploads")

	return ginRouter
}

func registerRoutes(ginRouter *gin.Engine, c config.Config) {

	// --- Public Routes (Login/Logout) ---
	ginRouter.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})
	ginRouter.POST("/api/login", database.Login)
	ginRouter.GET("/logout", func(c *gin.Context) {
		c.SetCookie("token", "", -1, "/", "localhost", false, true)
		c.Redirect(http.StatusSeeOther, "/")
	})

	// --- Authenticated Routes (for logged-in users) ---
	auth := ginRouter.Group("/")
	auth.Use(web.AuthenticateMiddleware)
	{
		// User Profile
		profile := auth.Group("/profile")
		{
			profile.GET("/edit", database.UserProfile)
			profile.GET("/edit/password", func(c *gin.Context) {
				c.HTML(http.StatusOK, "changePasswordModal.html", nil)
			})
			profile.POST("/edit/password", database.EditPassword)
		}

		// Jobs (Web Pages and API)
		auth.GET("/jobs/type/:jobTypeId", database.Jobs)
		auth.GET("/jobs/new-form/:id", database.NewJobModal)
		auth.POST("/jobs/add/:jobTypeId", database.AddNewJob)
		auth.POST("/jobs/edit/:id", database.EditJob)
		auth.GET("/jobs/:id", database.JobView)
		auth.GET("/jobs/edit-form/:id", database.EditJobModal)
		auth.GET("/jobs/:id/updates/new", database.NewJobUpdateModal)

		auth.GET("/api/jobs/search", database.SearchJobFinances)
		auth.GET("/api/jobs/:id", database.JobsList)
		auth.GET("/api/jobs/:id/updates", database.JobUpdateHistory)
		auth.POST("/api/jobs/:id/updates", database.NewJobUpdate)
		auth.DELETE("/api/jobs/:id", database.DeleteJob)

		// Finance (Web Pages and API)
		auth.GET("/finance", func(c *gin.Context) {
			c.HTML(http.StatusOK, "finance.html", nil)
		})
		auth.GET("/finance/new", func(c *gin.Context) {
			formData := gin.H{
				"description":      "",
				"amount":           "",
				"type":             "",
				"transaction_date": "",
				"related_job_id":   "",
			}

			c.HTML(http.StatusOK, "newFinancialRecordForm.html", gin.H{
				"FormData":           formData,
				"SelectedJobDisplay": "",
				"Error":              nil,
			})
		})

		auth.GET("/api/finance/transactions", database.FinanceList)
		auth.POST("/api/finance/transactions", database.AddNewFinancialRecord)

		// Contacts (Web Pages and API)
		auth.GET("/contacts/new", database.HandleNewContactModal)
		auth.POST("/contacts", database.CreateContact)
		auth.GET("/api/contacts/search", database.SearchContact)
	}

	// --- Admin Routes (Web Pages) ---
	adminRoutes := ginRouter.Group("/admin")
	adminRoutes.Use(web.AuthenticateMiddleware, web.IsAdmin)
	{
		adminRoutes.GET("/register", func(c *gin.Context) {
			c.HTML(http.StatusOK, "register.html", nil)
		})
		adminRoutes.GET("/users", func(c *gin.Context) {
			c.HTML(http.StatusOK, "userList.html", nil)
		})
		adminRoutes.GET("/users/edit/:id", database.EditUserPage)

		// Job Types
		adminRoutes.GET("/job-types/:id", database.GetJobTypeHandler)
		adminRoutes.GET("/job-types", database.JobTypeList)
		adminRoutes.POST("/job-types", database.CreateJobType)
		adminRoutes.GET("/job-types/edit/:id", database.JobTypeEditForm)
		adminRoutes.PUT("/job-types/:id", database.EditJobTypeDB)
		adminRoutes.GET("/job-types/:id/fields", database.GetCustomFieldsHandler)
		adminRoutes.POST("/job-types/:id/fields", database.AddNewCustomFields)
		adminRoutes.DELETE("/job-types/:id/fields/:fieldName", database.DeleteCustomFields)
	}

	// --- API Admin Routes ---
	apiAdminRoutes := ginRouter.Group("/api/admin")
	apiAdminRoutes.Use(web.AuthenticateMiddleware, web.IsAdmin)
	{
		apiAdminRoutes.GET("/userslist", database.UserList)
		apiAdminRoutes.POST("/register", database.Register)
		apiAdminRoutes.DELETE("/users/:id", database.DeleteUser)
		apiAdminRoutes.PUT("users/:id", database.EditUserDB)
	}

	// --- Condicional Routes (Logs) ---
	if c.Server.LogEndpoint {
		ginRouter.GET("/log", web.AuthenticateMiddleware, web.IsAdmin, web.Log)
		ginRouter.GET("/ws/log", web.WsLog)
	}
}
