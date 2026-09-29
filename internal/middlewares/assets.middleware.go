package middlewares

import (
	"net/http"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func ValidatePublicKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		publicKey := c.Param("assetId")
		if publicKey == "" {
			logger.Warn("asset id not provided in request")
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Could not get the public key"})
			c.Abort()
			return
		}
		if _, ok := utils.ValidatePublicKey(publicKey); !ok {
			logger.Warn("invalid public key provided", "public_key", publicKey)
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid public key"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func CheckIfAssetIsPublic(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		publicKey := c.Param("assetId")
		media, err := store.GetMedia(c.Request.Context(), logger, publicKey)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("media not found", "public_key", publicKey)
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": http.StatusText(http.StatusNotFound)})
				c.Abort()
				return
			}
			logger.Error("failed to retrieve media", "public_key", publicKey, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			c.Abort()
			return
		}
		if !media.Public {
			logger.Warn("access denied to private media", "public_key", publicKey)
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": http.StatusText(http.StatusForbidden)})
			c.Abort()
			return
		}
		c.Set("publicKey", publicKey)
		c.Set("media", media)
		c.Next()
	}
}
