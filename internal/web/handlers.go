package web

import (
	"Momentum/internal/jwt"
	"Momentum/internal/logger"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func WsLog(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.LogToLogFile(c, fmt.Sprintf("Ws Log: Error while upgrading the HTTP Server to WebSocket protocol `%v`", err))
		return
	}

	var lastMod time.Time
	if n, err := strconv.ParseInt(c.Request.FormValue("lastMod"), 16, 64); err == nil {
		lastMod = time.Unix(0, n)
	}

	go writer(ws, lastMod)
	reader(ws)

}

func Log(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	p, lastMod, err := readFileIfModified(time.Time{})
	if err != nil {
		p = []byte(err.Error())
		lastMod = time.Unix(0, 0)
	}
	var v = struct {
		Host    string
		Data    string
		LastMod string
	}{
		c.Request.Host,
		string(p),
		strconv.FormatInt(lastMod.UnixNano(), 16),
	}
	homeTempl.Execute(c.Writer, &v)
}

func Hello(c *gin.Context) {

	c.String(200, "Hello, World!")
}

func RedirectToHTTPS(httpsPort string) func(http.ResponseWriter, *http.Request) {

	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}

		target := "https://" + host + httpsPort + r.URL.RequestURI()

		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	}
}

func AuthenticateMiddleware(c *gin.Context) {
	tokenString, err := c.Cookie("token")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
		return
	}

	claims, err := jwt.VerifyToken(tokenString)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/login")
		c.Abort()
		return
	}

	c.Set("username", claims["sub"])
	c.Set("role", claims["aud"])
	c.Set("userID", claims["id"])

	c.Next()

}

func IsAdmin(c *gin.Context) {
	role, ok := c.Get("role")
	if role != "admin" && !ok {
		c.Redirect(http.StatusForbidden, "/")
		c.Abort()
		return
	}

	c.Next()

}
