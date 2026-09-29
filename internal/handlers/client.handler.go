package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/ajaka/redir/internal/configs"
	"github.com/ajaka/redir/internal/models"
	"github.com/ajaka/redir/internal/store"
	"github.com/ajaka/redir/internal/utils"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func HandleClientPing() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		logger.Info("ping attempt from client")
		client, ok := utils.GetProduct(c)
		if !ok {
			logger.Error("product not found in context for ping")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Something went wrong"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Client %v recognised, Proxy is ready for you.", client.ProductId)})
		logger.Info("ping acknowledged", "product_id", client.ProductId)
	}
}

func HandleUpload(cfg *configs.EnvData, tm *transfermanager.Client, store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		var mediaBatch []models.Media
		valid := false
		product, ok := utils.GetProduct(c)
		if !ok {
			logger.Error("product not found in context for upload")
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		batchId := c.GetHeader("X-Batch-ID")
		batchIdUUID, err := utils.ValidateAndReturnUUID(batchId)
		if batchId == "" || err != nil {
			logger.Warn("invalid or missing batch id header")
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "'X-Batch-ID' header is either missing or not valid uuid"})
			return
		}
		reader, err := c.Request.MultipartReader()
		if err != nil {
			logger.Error("failed to get multipart reader", "error", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Error getting multipart reader"})
			return
		}
		medias, err := store.RetriveBatch(c.Request.Context(), logger, batchIdUUID)
		if err != nil && err != pgx.ErrNoRows {
			logger.Error("failed to retrieve batch", "batch_id", batchIdUUID.String(), "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		existing := map[int]struct{}{}
		if medias != nil {
			for _, m := range *medias {
				existing[m.SeqId] = struct{}{}
			}
		}
		uploadCount := 0
		for {
			part, err := reader.NextPart()
			if err != nil {
				if err == io.EOF {
					if !valid {
						logger.Warn("upload received with no valid parts")
						c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Nothing was received"})
						return
					}
					break
				}
				logger.Error("multipart read error", "error", err.Error())
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "multipart read error"})
				return
			}
			valid = true
			seqId := part.Header.Get("X-Sequential-ID")
			seqIdI, _ := strconv.Atoi(seqId)
			if _, ok := existing[seqIdI]; ok {
				continue
			}

			fileName := part.FileName()
			contentType := part.Header.Get("Content-Type")
			innerKey, publicKey := utils.GenerateKeyForUpload(cfg, product.ProductId)
			_, err = tm.UploadObject(c.Request.Context(), &transfermanager.UploadObjectInput{
				Bucket: &cfg.BUCKET_NAME,
				Key:    &innerKey,
				Body:   part,
				Metadata: map[string]string{
					"original_name": fileName,
					"content_type":  contentType,
				},
			})

			if err != nil {
				logger.Error("failed to upload file to storage", "file_name", fileName, "seq_id", seqIdI, "product_id", product.ProductId, "error", err.Error())
				continue

			}
			uploadCount++
			mediaBatch = append(mediaBatch, models.Media{
				PublicKey: publicKey,
				InnerKey:  innerKey,
				FileName:  fileName,
				MimeType:  contentType,
				UserId:    product.UserId,
				BatchId:   batchIdUUID,
				SeqId:     seqIdI,
				Public:    product.Public,
			})
		}
		if len(mediaBatch) == 0 {
			logger.Warn("all files failed to upload", "batch_id", batchIdUUID.String(), "product_id", product.ProductId)
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "They all failed"})
			return
		}
		logger.Info("media batch created", "batch_id", batchIdUUID.String(), "batch_size", len(mediaBatch), "product_id", product.ProductId)
		media := store.CreateMediaBatch(c.Request.Context(), logger, &mediaBatch)
		HydrateMedias(cfg, *media)
		c.JSON(http.StatusCreated, gin.H{"success": true, "message": "file uploaded successfully", "media": media})
	}
}

func HandleBatchCommit(store *store.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := utils.GetLogger(c)
		batchId := c.Param("batchId")
		batchIdUUID, err := utils.ValidateAndReturnUUID(batchId)
		if err != nil {
			logger.Warn("invalid batch id provided to commit", "batch_id", batchId)
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid id given"})
			return
		}
		err = store.HandleBatchCommits(c.Request.Context(), logger, batchIdUUID)

		if err != nil {
			if err == pgx.ErrNoRows {
				logger.Warn("batch commit requested for non-existent batch", "batch_id", batchIdUUID.String())
				c.JSON(http.StatusNotFound, gin.H{"success": false, "message": fmt.Sprintf("Could not find batch with Id of '%s'", batchIdUUID)})
				return
			}
			logger.Error("failed to commit batch", "batch_id", batchIdUUID.String(), "error", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Something went wrong"})
			return
		}
		logger.Info("batch committed successfully", "batch_id", batchIdUUID.String())
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Batch commited successfully"})
	}
}
