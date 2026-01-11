// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// LogService 执行日志服务
type LogService struct {
	cfgMgr *config.Manager
	mu     sync.RWMutex
}

// NewLogService 创建日志服务
func NewLogService(cfgMgr *config.Manager) *LogService {
	return &LogService{
		cfgMgr: cfgMgr,
	}
}

// Create 创建执行日志
func (s *LogService) Create(log *model.ExecutionLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if log.ID == "" {
		log.ID = uuid.New().String()[:8]
	}

	logs, err := s.loadLogs()
	if err != nil {
		logs = []model.ExecutionLog{}
	}

	logs = append([]model.ExecutionLog{*log}, logs...)

	// 只保留最近1000条日志
	if len(logs) > 1000 {
		logs = logs[:1000]
	}

	return s.saveLogs(logs)
}

// List 获取日志列表
func (s *LogService) List(limit int) ([]model.ExecutionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, err := s.loadLogs()
	if err != nil {
		return nil, err
	}

	if limit > 0 && len(logs) > limit {
		return logs[:limit], nil
	}
	return logs, nil
}

// ListByTask 获取指定任务的日志
func (s *LogService) ListByTask(taskID string, limit int) ([]model.ExecutionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, err := s.loadLogs()
	if err != nil {
		return nil, err
	}

	var result []model.ExecutionLog
	for _, l := range logs {
		if l.TaskID == taskID {
			result = append(result, l)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

// GetStats 获取统计信息
func (s *LogService) GetStats() (*model.ExecutionStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, err := s.loadLogs()
	if err != nil {
		return nil, err
	}

	stats := &model.ExecutionStats{}

	var totalDuration int64
	for _, l := range logs {
		stats.TotalExecutions++
		totalDuration += l.Duration
		stats.TotalArticles += l.ArticleCount
		stats.TotalMatches += l.MatchCount

		switch l.Status {
		case model.LogStatusSuccess:
			stats.SuccessCount++
		case model.LogStatusFailed:
			stats.FailedCount++
		case model.LogStatusNoMatch:
			stats.NoMatchCount++
		}
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
		stats.AvgDuration = totalDuration / int64(stats.TotalExecutions)
	}

	return stats, nil
}

// GetStatsToday 获取今日统计
func (s *LogService) GetStatsToday() (*model.ExecutionStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs, err := s.loadLogs()
	if err != nil {
		return nil, err
	}

	today := time.Now().In(time.Local).Format("2006-01-02")
	stats := &model.ExecutionStats{}

	var totalDuration int64
	for _, l := range logs {
		if l.StartTime.In(time.Local).Format("2006-01-02") != today {
			continue
		}

		stats.TotalExecutions++
		totalDuration += l.Duration
		stats.TotalArticles += l.ArticleCount
		stats.TotalMatches += l.MatchCount

		switch l.Status {
		case model.LogStatusSuccess:
			stats.SuccessCount++
		case model.LogStatusFailed:
			stats.FailedCount++
		case model.LogStatusNoMatch:
			stats.NoMatchCount++
		}
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions) * 100
		stats.AvgDuration = totalDuration / int64(stats.TotalExecutions)
	}

	return stats, nil
}

// ClearAll 清空所有执行日志
func (s *LogService) ClearAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.cfgMgr.DataDir, "logs.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// PurgeBefore 清理指定时间之前的日志
func (s *LogService) PurgeBefore(cutoff time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	logs, err := s.loadLogs()
	if err != nil {
		return err
	}

	var kept []model.ExecutionLog
	for _, l := range logs {
		if l.StartTime.Before(cutoff) {
			continue
		}
		kept = append(kept, l)
	}

	return s.saveLogs(kept)
}

// loadLogs 加载日志
func (s *LogService) loadLogs() ([]model.ExecutionLog, error) {
	path := filepath.Join(s.cfgMgr.DataDir, "logs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.ExecutionLog{}, nil
		}
		return nil, err
	}

	var logs []model.ExecutionLog
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, err
	}

	for i := range logs {
		logs[i].Details = localizeLogDetails(logs[i].Details)
		logs[i].Error = localizeLogError(logs[i].Error)
	}

	// 按时间降序排序
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].StartTime.After(logs[j].StartTime)
	})

	return logs, nil
}

var sentItemsRe = regexp.MustCompile(`^Sent\s+(\d+)\s+item\(s\)$`)

func localizeLogDetails(details string) string {
	d := strings.TrimSpace(details)
	if d == "" {
		return details
	}

	// 兼容旧版本英文详情
	if m := sentItemsRe.FindStringSubmatch(d); len(m) == 2 {
		return fmt.Sprintf("已发送 %s 条", m[1])
	}
	if d == "No RSS content available" {
		return "无 RSS 内容"
	}
	if d == "All articles already sent" {
		return "今日文章已全部发送"
	}
	if d == "No new articles since last analysis" {
		return "与上次相比无新增文章"
	}
	if strings.HasPrefix(d, "No articles selected in phase 1:") {
		return "第一阶段未选中任何文章：" + strings.TrimSpace(strings.TrimPrefix(d, "No articles selected in phase 1:"))
	}
	if d == "Failed to load content" {
		return "加载文章内容失败"
	}
	if d == "No matching content" {
		return "第二阶段无匹配内容"
	}
	if d == "All matched content already processed or below importance" {
		return "匹配内容均已发送或重要性不足"
	}

	return details
}

func localizeLogError(errMsg string) string {
	e := strings.TrimSpace(errMsg)
	if e == "" {
		return errMsg
	}
	if strings.HasPrefix(e, "notification failed:") {
		return "通知发送失败：" + strings.TrimSpace(strings.TrimPrefix(e, "notification failed:"))
	}
	return errMsg
}

// saveLogs 保存日志
func (s *LogService) saveLogs(logs []model.ExecutionLog) error {
	path := filepath.Join(s.cfgMgr.DataDir, "logs.json")
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
