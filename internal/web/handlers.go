package web

import (
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
)

func WsLog(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("upgrade:", err)
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
