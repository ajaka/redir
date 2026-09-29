package middlewares

import (
	"log/slog"
	"strings"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
)

func GenAndAttachRequestIdMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if id == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n") {
			id = utils.GenUUID()
		}
		c.Writer.Header().Set("X-Request-ID", id)
		c.Set("requestId", id)
		c.Next()
	}
}

func AttachLoggerToContext(cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqId := c.GetString("requestId")
		reqLogger := slog.Default().With(
			slog.String("request_id", reqId),
			slog.String("environment", cfg.ENVIRONMENT),
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
		)
		c.Set("logger", reqLogger)
		c.Next()
	}
}

func PerformBasicRequestCycleCalculations() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		logger := utils.GetLogger(c)

		logger.Info("request started")
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		msg := "request completed"
		attrs := []any{
			"status", status,
			"latency", latency,
		}
		if status >= 500 {
			logger.Error(msg, attrs...)
		} else if status >= 400 {
			logger.Warn(msg, attrs...)
		} else {
			logger.Info(msg, attrs...)
		}
	}
}
