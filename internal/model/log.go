// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package model

import "time"

// ExecutionLog 任务执行日志
type ExecutionLog struct {
	ID           string    `json:"id"`
	TaskID       string    `json:"task_id"`
	TaskName     string    `json:"task_name"`
	Status       string    `json:"status"` // success/failed/no_match
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     int64     `json:"duration"` // 毫秒
	ArticleCount int       `json:"article_count"`
	MatchCount   int       `json:"match_count"`
	Error        string    `json:"error,omitempty"`
	Details      string    `json:"details,omitempty"`
}

// 执行状态常量
const (
	LogStatusSuccess = "success"
	LogStatusFailed  = "failed"
	LogStatusNoMatch = "no_match"
)

// ExecutionStats 执行统计
type ExecutionStats struct {
	TotalExecutions int     `json:"total_executions"`
	SuccessCount    int     `json:"success_count"`
	FailedCount     int     `json:"failed_count"`
	NoMatchCount    int     `json:"no_match_count"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDuration     int64   `json:"avg_duration"`
	TotalArticles   int     `json:"total_articles"`
	TotalMatches    int     `json:"total_matches"`
}
