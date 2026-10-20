package log

import (
	"io"
	"os"
	"path"

	"kinstack/config"
	"kinstack/internal/pkg/assert"

	"gopkg.in/natefinch/lumberjack.v2"
)

func rollingFile(cfg config.Config) (io.Writer, error) {
	assert.Assert(cfg.Log.Dir != "", "log directory must be set")
	assert.Assert(cfg.Log.Name != "", "log name must be set")
	assert.NoError(os.MkdirAll(cfg.Log.Dir, 0o744))

	return &lumberjack.Logger{
		Filename:   path.Join(cfg.Log.Dir, cfg.Log.Name+".log"),
		MaxBackups: cfg.Log.MaxBackup,
		MaxSize:    cfg.Log.MaxSize,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   true,
	}, nil
}
