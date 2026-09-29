package handlers

import (
	"net/http"
	"time"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/models"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/ua-parser/uap-go/uaparser"
)

func HandleRedirect(cfg *configs.EnvData, presignedClient *s3.PresignClient, store *store.Store, parser *uaparser.Parser) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		media, ok := utils.GetMedia(c)
		if !ok {
			logger.Error("media not found in context")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		ua := c.GetHeader("user-agent")
		client := parser.Parse(ua)
		metric := models.Metrics{
			MediaId:        media.PublicKey,
			Browser:        client.UserAgent.Family,
			BrowserVersion: client.UserAgent.Major,
			Device:         client.Device.Family,
			DeviceBrand:    client.Device.Brand,
			DeviceModel:    client.Device.Model,
			Os:             client.Os.Family,
			OsVersion:      client.Os.Major,
			Ip:             c.ClientIP(),
		}
		preSignedUrl, err := store.GetPresigned(c.Request.Context(), logger, media.PublicKey)
		if err != nil {
			preSigned, err := presignedClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
				Bucket: aws.String(cfg.BUCKET_NAME),
				Key:    aws.String(media.InnerKey),
			}, s3.WithPresignExpires(30*time.Minute))
			if err != nil {
				logger.Error("failed to generate presigned url", "public_key", media.PublicKey, "error", err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
				return
			}
			store.SetPresigned(c.Request.Context(), logger, media.PublicKey, preSigned.URL, 28*time.Minute)
			preSignedUrl = preSigned.URL
		}
		err = store.SaveMetrics(c.Request.Context(), logger, &metric)
		if err != nil {
			logger.Error("failed to save access metrics", "media_id", media.PublicKey, "error", err.Error())
		}
		c.Redirect(http.StatusFound, preSignedUrl)
	}
}

type toggleAsset struct {
	Public bool `json:"public"`
}

func ToggleAssetVisibility(store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		var req toggleAsset
		publicKey := c.Param("assetId")
		err := c.ShouldBindJSON(&req)
		if err != nil {
			logger.Error("failed to parse asset visibility toggle request", "error", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": http.StatusText(http.StatusBadRequest)})
			return
		}
		m, err := store.ToggleAsset(c.Request.Context(), logger, publicKey, req.Public)
		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("asset not found during visibility toggle", "public_key", publicKey)
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Invalid publicKey"})
				return
			}
			logger.Error("failed to toggle asset visibility", "public_key", publicKey, "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		logger.Info("asset visibility toggled", "public_key", m.PublicKey, "public", m.Public)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "success", "media": m})
	}
}
