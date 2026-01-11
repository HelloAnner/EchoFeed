// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 任务去重状态存储：按任务记录已处理文章ID，避免重复推送。
// @author Anner
// Created on 2026/1/10
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
)

type taskProcessedState struct {
	TaskID      string            `json:"task_id"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ProcessedAt map[string]string `json:"processed_at"` // article_id -> RFC3339

	LastRunAt string `json:"last_run_at,omitempty"` // RFC3339

	LastAnalyzedDate string `json:"last_analyzed_date,omitempty"`
	LastAnalyzedHash string `json:"last_analyzed_hash,omitempty"`
}

// TaskStateService 任务去重状态服务
type TaskStateService struct {
	cfgMgr *config.Manager
}

// NewTaskStateService 创建任务去重状态服务
func NewTaskStateService(cfgMgr *config.Manager) *TaskStateService {
	return &TaskStateService{cfgMgr: cfgMgr}
}

// IsProcessed 判断文章是否已被该任务处理过
func (s *TaskStateService) IsProcessed(taskID, articleID string) bool {
	state, err := s.load(taskID)
	if err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("Failed to load task state, treat as unprocessed")
		return false
	}
	if state.ProcessedAt == nil {
		return false
	}
	return state.ProcessedAt[articleID] != ""
}

// MarkProcessed 标记文章已被该任务处理
func (s *TaskStateService) MarkProcessed(taskID, articleID string) error {
	if taskID == "" || articleID == "" {
		return nil
	}

	state, err := s.load(taskID)
	if err != nil {
		return err
	}
	if state.ProcessedAt == nil {
		state.ProcessedAt = make(map[string]string)
	}

	state.ProcessedAt[articleID] = time.Now().Format(time.RFC3339)
	state.UpdatedAt = time.Now()

	s.pruneIfNeeded(state, 5000)
	return s.save(taskID, state)
}

// IsOverviewAnalyzed 判断该任务对指定overview是否已分析过（用于节省token）
func (s *TaskStateService) IsOverviewAnalyzed(taskID, date, overviewHash string) bool {
	if taskID == "" || date == "" || overviewHash == "" {
		return false
	}
	state, err := s.load(taskID)
	if err != nil {
		log.Warn().Err(err).Str("task_id", taskID).Msg("Failed to load task state, treat overview as not analyzed")
		return false
	}
	return state.LastAnalyzedDate == date && state.LastAnalyzedHash == overviewHash
}

// MarkOverviewAnalyzed 标记该任务已分析过某个overview（按日期+hash）
func (s *TaskStateService) MarkOverviewAnalyzed(taskID, date, overviewHash string) error {
	if taskID == "" || date == "" || overviewHash == "" {
		return nil
	}
	state, err := s.load(taskID)
	if err != nil {
		return err
	}
	state.LastAnalyzedDate = date
	state.LastAnalyzedHash = overviewHash
	state.UpdatedAt = time.Now()
	return s.save(taskID, state)
}

// MarkTaskRun 记录任务上次执行时间（不区分成功/失败/跳过）
func (s *TaskStateService) MarkTaskRun(taskID string, at time.Time) error {
	if taskID == "" || at.IsZero() {
		return nil
	}
	state, err := s.load(taskID)
	if err != nil {
		return err
	}
	state.LastRunAt = at.Format(time.RFC3339)
	state.UpdatedAt = time.Now()
	return s.save(taskID, state)
}

// GetLastRunAt 获取任务上次执行时间
func (s *TaskStateService) GetLastRunAt(taskID string) (time.Time, bool) {
	if taskID == "" {
		return time.Time{}, false
	}
	state, err := s.load(taskID)
	if err != nil {
		return time.Time{}, false
	}
	if state.LastRunAt == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, state.LastRunAt)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// PurgeProcessedBefore 清理指定时间之前的“已处理文章ID”记录
func (s *TaskStateService) PurgeProcessedBefore(cutoff time.Time) error {
	root := filepath.Join(s.cfgMgr.DataDir, "state", "tasks")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".json") {
			continue
		}
		taskID := strings.TrimSuffix(ent.Name(), ".json")
		if err := s.purgeOne(taskID, cutoff); err != nil {
			log.Warn().Err(err).Str("task_id", taskID).Msg("Failed to purge task processed state")
		}
	}
	return nil
}

func (s *TaskStateService) purgeOne(taskID string, cutoff time.Time) error {
	state, err := s.load(taskID)
	if err != nil {
		return err
	}
	if state.ProcessedAt == nil || len(state.ProcessedAt) == 0 {
		return nil
	}

	kept := make(map[string]string, len(state.ProcessedAt))
	for id, ts := range state.ProcessedAt {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			kept[id] = ts
			continue
		}
		if t.Before(cutoff) {
			continue
		}
		kept[id] = ts
	}

	state.ProcessedAt = kept
	state.UpdatedAt = time.Now()
	return s.save(taskID, state)
}

func computeOverviewHash(articleIDs []string) string {
	if len(articleIDs) == 0 {
		return ""
	}
	ids := make([]string, 0, len(articleIDs))
	for _, id := range articleIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join(ids, ",")))
	return hex.EncodeToString(sum[:])
}

func (s *TaskStateService) statePath(taskID string) string {
	return filepath.Join(s.cfgMgr.DataDir, "state", "tasks", taskID+".json")
}

func (s *TaskStateService) load(taskID string) (*taskProcessedState, error) {
	path := s.statePath(taskID)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &taskProcessedState{TaskID: taskID, ProcessedAt: make(map[string]string)}, nil
		}
		return nil, err
	}

	var state taskProcessedState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	if state.TaskID == "" {
		state.TaskID = taskID
	}
	if state.ProcessedAt == nil {
		state.ProcessedAt = make(map[string]string)
	}

	return &state, nil
}

func (s *TaskStateService) save(taskID string, state *taskProcessedState) error {
	path := s.statePath(taskID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *TaskStateService) pruneIfNeeded(state *taskProcessedState, max int) {
	if state == nil || state.ProcessedAt == nil || len(state.ProcessedAt) <= max {
		return
	}

	type kv struct {
		ID string
		T  string
	}
	all := make([]kv, 0, len(state.ProcessedAt))
	for id, t := range state.ProcessedAt {
		all = append(all, kv{ID: id, T: t})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].T > all[j].T })

	keep := make(map[string]string, max)
	for i := 0; i < max && i < len(all); i++ {
		keep[all[i].ID] = all[i].T
	}
	state.ProcessedAt = keep
}
