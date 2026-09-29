package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/ajaka/redir/internal/models"
)

func (r *Repository) SaveMetrics(ctx context.Context, logger *slog.Logger, metric *models.Metrics) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
	INSERT INTO METRICS (media_id, browser, browser_version, device, device_brand, device_model, os, os_version, ip)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`
	_, err := r.pool.Exec(ctx, query, metric.MediaId, metric.Browser, metric.BrowserVersion, metric.Device, metric.DeviceBrand, metric.DeviceModel, metric.Os, metric.OsVersion, metric.Ip)
	if err != nil {
		logger.Error("failed to save metrics", "media_id", metric.MediaId, "error", err.Error())
	}
	return err
}
