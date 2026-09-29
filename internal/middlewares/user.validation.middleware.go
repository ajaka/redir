package middlewares

import (
	"net/http"
	"strings"

	"github.com/ajaka/redir/internal/domain"
	"github.com/ajaka/redir/internal/utils"
	"github.com/gin-gonic/gin"
)

func RegisterValidationMiddleware(c *gin.Context) {
	logger := utils.GetLogger(c)
	logger.Info("Validating user registration details")
	var RegisterRequestBody domain.
		CreateUserDetails
	var errors []string
	err := c.ShouldBindJSON(&RegisterRequestBody)
	if err != nil {
		logger.Warn("User provided bad data on registration")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		c.Abort()
		return
	}
	RegisterRequestBody.FullName = strings.TrimSpace(RegisterRequestBody.FullName)
	names := strings.Fields(RegisterRequestBody.FullName)
	if len(names) < 2 {
		errors = append(errors, "Fullname should contain at least two names")
	}
	if len(RegisterRequestBody.Password) < 8 {
		errors = append(errors, "Password should be at least 8 characters")
	}
	if len(errors) > 0 {
		logger.Warn("Validating user registration details failed")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errors": errors})
		c.Abort()
		return
	}
	c.Set("reg", &RegisterRequestBody)
	c.Next()
}

func LoginValidationMiddleware(c *gin.Context) {
	logger := utils.GetLogger(c)
	logger.Info("Validating user login details")
	var LoginRequestBody domain.
		LoginUserDetails
	var errors []string
	err := c.ShouldBindJSON(&LoginRequestBody)
	if err != nil {
		logger.Warn("User provided bad data on login")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		c.Abort()
		return
	}
	if len(LoginRequestBody.Password) < 8 {
		errors = append(errors, "Password should be at least 8 characters")
	}
	if len(errors) > 0 {
		logger.Warn("Validating user login details failed")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errors": errors})
		c.Abort()
		return
	}
	c.Set("login", &LoginRequestBody)
	c.Next()
}

func ProductValidationMiddleware(c *gin.Context) {
	var request domain.CreateProductDetails
	logger := utils.GetLogger(c)
	logger.Info("Validating user product creation details")
	err := c.ShouldBindJSON(&request)
	if err != nil {
		logger.Warn("User provided bad data on product creation ")
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		c.Abort()
		return
	}
	c.Set("product", &request)
	c.Next()
}
