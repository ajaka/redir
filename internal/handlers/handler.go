package handlers

import (
	"net/http"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/models"
	"github.com/gin-gonic/gin"
)

func HydrateMedias(cfg *configs.EnvData, m []models.Media) {
	for i := range m {
		m[i].PublicKey = cfg.DATA_GET_PATH + m[i].PublicKey
	}
}

func setCookieFromHttpCookie(c *gin.Context, cookie *http.Cookie) {
	maxAge := int(time.Until(cookie.Expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.SetCookie(cookie.Name, cookie.Value, maxAge, cookie.Path, "", cookie.Secure, cookie.HttpOnly)
}
