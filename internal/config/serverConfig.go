package config

import (
	"Momentum/internal/web"
	"crypto/tls"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"time"
)

func LoadHTTPServer(configFile Config) {
	httpServer := &http.Server{
		Addr:         configFile.Server.HttpPort,
		Handler:      http.HandlerFunc(web.RedirectToHTTPS(configFile.Server.HttpsPort)),
		ReadTimeout:  time.Duration(configFile.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(configFile.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(configFile.Server.IdleTimeout) * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error initiating the HTTP Server: %v", err)

	}

}

func LoadServerHTTPSConfig(configFile Config, ginRouter *gin.Engine) (serverHTTPS *http.Server) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
			tls.CurveP384,
		},
		PreferServerCipherSuites: configFile.Security.PreferServerCipherSuites,
	}

	ginRouter.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	serverHTTPS = &http.Server{
		Handler:      ginRouter,
		ReadTimeout:  time.Duration(configFile.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(configFile.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(configFile.Server.IdleTimeout) * time.Second,
		Addr:         configFile.Server.HttpsPort,
		TLSConfig:    tlsConfig,
	}

	return serverHTTPS

}
