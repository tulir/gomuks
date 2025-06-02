package tui

import (
	"context"
	"os"

	"github.com/rs/zerolog"
)

// InitLogger initializes a background logger in the tempdir. Returns the modified context.
func InitLogger(ctx context.Context) context.Context {
	tempfile, err := os.CreateTemp(os.TempDir(), "gomuks-tui-*.log")
	if err != nil {
		panic("failed to create temp log file: " + err.Error())
	}
	println("Logging to", tempfile.Name())
	logger := zerolog.New(tempfile)
	logger = logger.With().Timestamp().Logger()
	return logger.WithContext(ctx)
}
