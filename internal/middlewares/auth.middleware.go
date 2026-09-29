package middlewares

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func AuthMiddleware(st store.AuthStore, cfg *configs.EnvData) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		sessionId, err := c.Cookie("sessionId")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}
		user, ok := st.GetUser(c.Request.Context(), logger, sessionId)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid or expired sessionId"})
			c.Abort()
			return
		}
		c.Set("sessionId", sessionId)
		c.Set("user", user)
		c.Next()
	}
}

func CheckAndValidateClientKeys(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		pId := c.GetHeader("X-Product")
		pKey := c.GetHeader("Authorization")
		k, ok := strings.CutPrefix(pKey, "Bearer ")
		if !ok || pId == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid Auth token or clientId"})
			c.Abort()
			return
		}
		pIdI, err := strconv.Atoi(pId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Product ID"})
			c.Abort()
			return
		}
		product, err := store.GetProductById(c.Request.Context(), logger, pIdI)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("product not found for key validation", "product_id", pIdI)
				c.JSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("Product with id of %d not found", pIdI)})
				c.Abort()
				return
			}
			logger.Error("failed to get product for key validation", "product_id", pIdI, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
			c.Abort()
			return
		}
		err = utils.VerifyMultipStepHash(k, product.PrivateKey)
		if err != nil {
			logger.Warn("invalid key provided for product", "product_id", pIdI)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid key given"})
			c.Abort()
			return
		}
		c.Set("product", product)
		c.Next()
	}
}

func CanThisUserAlterThisProduct(cfg *configs.EnvData, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := utils.GetUser(c)
		logger := utils.GetLogger(c)
		if !ok || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": http.StatusText(http.StatusUnauthorized)})
			c.Abort()
			return
		}
		pId := c.Param("id")
		pIdI, err := strconv.Atoi(pId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid Product ID"})
			c.Abort()
			return
		}
		product, err := store.GetProductById(c.Request.Context(), logger, pIdI)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("product not found for user permission check", "product_id", pIdI, "user_id", user.Id.String())
				c.JSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("Product with id of %d not found", pIdI)})
				c.Abort()
				return
			}
			logger.Error("failed to get product for permission check", "product_id", pIdI, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
			c.Abort()
			return
		}
		if product.UserId != user.Id {
			logger.Warn("user attempted to access unauthorized product", "product_id", pIdI, "user_id", user.Id.String(), "product_owner", product.UserId.String())
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "You are forbidden from performing this operation"})
			c.Abort()
			return
		}
		c.Set("product", product)
		c.Set("id", pIdI)
		c.Next()
	}
}

func ValidateToken(store store.AuthStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Query("token")
		logger := utils.GetLogger(c)
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Token not found"})
			c.Abort()
			return
		}
		email, err := store.GetVerificationUser(c.Request.Context(), logger, token)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid token"})
			c.Abort()
			return
		}
		c.Set("email", email)
		c.Next()
	}
}
