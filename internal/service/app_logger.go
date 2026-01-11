// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 应用日志配置：根据 settings.toml 的 log 配置输出到控制台 + data/logs/app.log
// @author Anner
// Created on 2026/1/10
package service

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/echofeed/echofeed/internal/model"
)

var appLoggerMu sync.Mutex
var appLogFile io.WriteCloser

func ConfigureAppLogger(dataDir string, cfg model.LogSettings) {
	appLoggerMu.Lock()
	defer appLoggerMu.Unlock()

	if appLogFile != nil {
		_ = appLogFile.Close()
		appLogFile = nil
	}

	level := parseLogLevel(cfg.Level)
	zerolog.SetGlobalLevel(level)

	maxSize := cfg.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100
	}
	maxBackups := cfg.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 3
	}

	_ = os.MkdirAll(filepath.Join(dataDir, "logs"), 0755)
	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(dataDir, "logs", "app.log"),
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     30,
		Compress:   false,
	}
	appLogFile = fileWriter

	consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	multi := zerolog.MultiLevelWriter(consoleWriter, fileWriter)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}

func parseLogLevel(level string) zerolog.Level {
	s := strings.TrimSpace(strings.ToLower(level))
	switch s {
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "info", "":
		return zerolog.InfoLevel
	default:
		return zerolog.InfoLevel
	}
}
