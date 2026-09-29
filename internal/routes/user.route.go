package routes

import (
	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/handlers"
	"github.com/ajaka/redir/internal/middlewares"
	"github.com/ajaka/redir/internal/store"
	"github.com/gin-gonic/gin"
)

func UserRoutes(rg *gin.RouterGroup, cfg *configs.EnvData, store *store.Store) {
	user := rg.Group("/users")
	user.Use(middlewares.RL.GetLimiterForProductAndUser(10))
	user.Use(middlewares.AuthMiddleware(store, cfg))
	user.GET("/me", handlers.GetUser(cfg, store))
}
