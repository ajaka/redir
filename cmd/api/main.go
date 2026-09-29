package main

import (
	"log/slog"

	"github.com/ajaka/redir/internal"
)

func main() {
	logger := slog.Default()
	if err := internal.Listen(); err != nil {
		logger.Error("failed to start server", "error", err.Error())
	}
}
