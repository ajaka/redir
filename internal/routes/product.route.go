package routes

import (
	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/handlers"
	"github.com/ajaka/redir/internal/middlewares"
	"github.com/ajaka/redir/internal/store"
	"github.com/gin-gonic/gin"
)

func ProductRoutes(rg *gin.RouterGroup, cfg *configs.EnvData, store *store.Store) {
	product := rg.Group("/product")
	product.Use(middlewares.RL.GetLimiterForProductAndUser(10))
	product.Use(middlewares.AuthMiddleware(store, cfg))
	product.POST("", middlewares.ProductValidationMiddleware, handlers.CreateProduct(cfg, store))
	product.POST("/:id", middlewares.CanThisUserAlterThisProduct(cfg, store), handlers.GenerateKey(cfg, store))
	product.PUT("/:id", middlewares.CanThisUserAlterThisProduct(cfg, store), handlers.ToggleProductVisibility(cfg, store))
	product.PUT("/:id/assets/:assetId", middlewares.ValidatePublicKey(), middlewares.CanThisUserAlterThisProduct(cfg, store), handlers.ToggleAssetVisibility(store))
}
