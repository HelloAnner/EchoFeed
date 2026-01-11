// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 数据清理任务：保留最近 N 天的 RSS 文章与执行日志。
// @author Anner
// Created on 2026/1/10
package service

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
)

// CleanupService 数据清理服务
type CleanupService struct {
	cfgMgr    *config.Manager
	logSvc    *LogService
	taskState *TaskStateService
}

// NewCleanupService 创建数据清理服务
func NewCleanupService(cfgMgr *config.Manager, logSvc *LogService) *CleanupService {
	return &CleanupService{
		cfgMgr:    cfgMgr,
		logSvc:    logSvc,
		taskState: NewTaskStateService(cfgMgr),
	}
}

// CleanupBeforeToday 保留最近 retentionDays 天的数据
func (s *CleanupService) CleanupBeforeToday(retentionDays int) error {
	if retentionDays <= 0 {
		retentionDays = 7
	}

	now := time.Now()
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	keepFrom := today.AddDate(0, 0, -(retentionDays - 1))

	if err := s.deleteOldRSSDirs(keepFrom); err != nil {
		return err
	}

	if s.logSvc != nil {
		if err := s.logSvc.PurgeBefore(keepFrom); err != nil {
			return err
		}
	}

	if err := s.taskState.PurgeProcessedBefore(keepFrom); err != nil {
		return err
	}

	return nil
}

func (s *CleanupService) deleteOldRSSDirs(keepFrom time.Time) error {
	rssRoot := filepath.Join(s.cfgMgr.DataDir, "rss")
	entries, err := os.ReadDir(rssRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		date, ok := parseDateDir(ent.Name(), keepFrom.Location())
		if !ok || !date.Before(keepFrom) {
			continue
		}

		dir := filepath.Join(rssRoot, ent.Name())
		if err := os.RemoveAll(dir); err != nil {
			log.Warn().Err(err).Str("dir", dir).Msg("Failed to delete old rss dir")
			continue
		}
		log.Info().Str("dir", dir).Msg("Deleted old rss dir")
	}
	return nil
}

func parseDateDir(name string, loc *time.Location) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02", name, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
