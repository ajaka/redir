package handlers

import (
	"net/http"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GenerateKey(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		pIdI, ok := utils.GetId(c)
		if !ok {
			logger.Error("product id not found in context for key generation")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Couldn't get id"})
			return
		}
		pKey := utils.GeneratePrivateKey()
		hKey, err := utils.PerformMultiStepHash(pKey)
		if err != nil {
			logger.Error("failed to hash private key", "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
			return
		}
		product, err := store.CreatePrivateKey(c.Request.Context(), logger, pIdI, hKey)
		if err != nil {
			logger.Error("failed to create private key", "product_id", pIdI, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
			return
		}
		product.PrivateKey = pKey
		logger.Info("private key generated", "product_id", pIdI)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Private Key created successfully", "product": product})
	}
}

func CreateProduct(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		user, _ := utils.GetUser(c)
		val, _ := c.Get("product")
		request, ok := val.(*domain.CreateProductDetails)
		if !ok {
			logger.Error("could not parse product creation request from context")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		request.UserId = user.Id
		product, err := store.CreateProduct(c.Request.Context(), logger, request)
		if err != nil {
			logger.Error("failed to create product", "product_name", request.ProductName, "user_id", request.UserId, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		logger.Info("product created", "product_id", product.ProductId, "product_name", request.ProductName)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "Product created successfully", "product": product})
	}
}

type toggleProduct struct {
	Public bool `json:"public"`
}

func ToggleProductVisibility(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		var req toggleProduct
		p, ok := utils.GetProduct(c)
		if !ok {
			logger.Error("product not found in context for visibility toggle")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		err := c.ShouldBindJSON(&req)
		if err != nil {
			logger.Error("failed to parse product visibility toggle request", "error", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": http.StatusText(http.StatusBadRequest)})
			return
		}
		if p.Public == req.Public {
			logger.Info("product visibility toggle request has no state change", "product_id", p.ProductId, "public", req.Public)
			c.JSON(http.StatusBadRequest, gin.H{"success": false})
			return
		}
		product, err := store.ToggleProductVisibility(c.Request.Context(), logger, req.Public, p.ProductId)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("product not found during visibility toggle", "product_id", p.ProductId)
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Invalid productId"})
				return
			}
			logger.Error("failed to toggle product visibility", "product_id", p.ProductId, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		logger.Info("product visibility toggled", "product_id", product.ProductId, "public", product.Public)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "success", "product": product})
	}
}
