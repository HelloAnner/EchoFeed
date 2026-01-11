// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// RSS 订阅抓取状态：用于记录每个订阅的上次更新时间（最后一次抓取尝试时间），供页面展示。
//
// @author Anner
// Created on 2026/1/10
package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
)

type FeedState struct {
	FeedID      string `json:"feed_id"`
	LastFetchAt string `json:"last_fetch_at,omitempty"` // RFC3339
	LastError   string `json:"last_error,omitempty"`
}

// FeedStateService 订阅抓取状态服务
type FeedStateService struct {
	cfgMgr *config.Manager
	mu     sync.Mutex
}

func NewFeedStateService(cfgMgr *config.Manager) *FeedStateService {
	return &FeedStateService{cfgMgr: cfgMgr}
}

func (s *FeedStateService) statePath(feedID string) string {
	return filepath.Join(s.cfgMgr.DataDir, "state", "feeds", feedID+".json")
}

func (s *FeedStateService) Load(feedID string) (*FeedState, error) {
	path := s.statePath(feedID)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &FeedState{FeedID: feedID}, nil
		}
		return nil, err
	}

	var st FeedState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	if st.FeedID == "" {
		st.FeedID = feedID
	}
	return &st, nil
}

func (s *FeedStateService) Save(feedID string, st *FeedState) error {
	if st == nil || feedID == "" {
		return nil
	}
	path := s.statePath(feedID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	st.FeedID = feedID
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// MarkFetch 记录一次抓取尝试（成功/失败都会更新上次时间；失败会记录错误）
func (s *FeedStateService) MarkFetch(feedID string, at time.Time, fetchErr error) error {
	if feedID == "" || at.IsZero() {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.Load(feedID)
	if err != nil {
		log.Warn().Err(err).Str("feed_id", feedID).Msg("Failed to load feed state")
		st = &FeedState{FeedID: feedID}
	}

	st.LastFetchAt = at.In(time.Local).Format(time.RFC3339)
	if fetchErr != nil {
		st.LastError = strings.TrimSpace(fetchErr.Error())
	} else {
		st.LastError = ""
	}
	return s.Save(feedID, st)
}
