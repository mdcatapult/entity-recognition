package lib

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func JsonLogFormatter(params gin.LogFormatterParams) string {
	logline := map[string]interface{}{
		"time":    params.TimeStamp.UTC().Format("2006-01-02T15:04:05.999"),
		"status":  params.StatusCode,
		"latency": params.Latency.String(),
		"client":  params.ClientIP,
		"method":  params.Method,
		"path":    params.Path,
	}
	if params.ErrorMessage != "" {
		logline["error"] = params.ErrorMessage
	}
	if len(params.Keys) > 0 {
		logline["context"] = params.Keys
	}
	b, _ := json.Marshal(logline)
	return string(b) + "\n"
}
