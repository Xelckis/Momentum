package logger

import (
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var mu sync.Mutex
var LogFileWriter io.Writer

func LogToLogFile(c *gin.Context, errMsg string) {
	mu.Lock()
	_, err := fmt.Fprintf(LogFileWriter, "%s - [%s] \"[%s] %s?%s [%d]\" - \"%s\" \n", c.ClientIP(), time.Now().Format(time.RFC1123), c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery, c.Writer.Status(), errMsg)
	mu.Unlock()
	if err != nil {
		log.Printf("Error writing on log file: %v", err)
	}
}
