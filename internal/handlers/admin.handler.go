package handlers

import (
	"net/http"

	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
)

func HandlePing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "pong"})
	}
}

func HandleReady(store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		logger.Info("readyz: request received")

		if err := store.CheckDependencies(c.Request.Context(), logger); err != nil {
			logger.Error("readyz: dependency check failed", "error", err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": "service unavailable",
				"error":   "dependency check failed: ",
			})
			return
		}

		logger.Info("readyz: service ready")
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "ready"})
	}
}
