package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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

	today := time.Now().Format("2006-01-02")
	stats := &model.ExecutionStats{}

	var totalDuration int64
	for _, l := range logs {
		if l.StartTime.Format("2006-01-02") != today {
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

	// 按时间降序排序
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].StartTime.After(logs[j].StartTime)
	})

	return logs, nil
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
