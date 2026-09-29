package handlers

import (
	"fmt"
	"net/http"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetUser(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		u, ok := utils.GetUser(c)
		if !ok {
			logger.Error("user not found in context for get user request")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Couldn't identify user"})
			return
		}
		user, err := store.GetUserById(c.Request.Context(), logger, cfg, u.Id)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("requested user not found", "user_id", u.Id.String())
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": fmt.Sprintf("Couldn't find user with id of %v", u.Id)})
				return
			}
			logger.Error("failed to get user", "user_id", u.Id.String(), "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		logger.Info("user retrieved", "user_id", u.Id.String())
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "User retrieved successfully", "user": *user})
	}
}
