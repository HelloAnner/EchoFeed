// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 当日已发送记录：按天落盘到 data/rss/YYYY-MM-DD/sended.json，
// 用于确保同一任务重复执行时，AI 两阶段输入都不包含已发送文章。
// @author Anner
// Created on 2026/1/10
package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

type SendedArticle struct {
	ArticleID    string `json:"article_id"`
	Title        string `json:"title"`
	Link         string `json:"link"`
	Source       string `json:"source"`
	PublishedAt  string `json:"published_at"`
	SentAt       string `json:"sent_at"`
	Importance   int    `json:"importance"`
	Summary100CN string `json:"summary_100_cn"`
}

type TaskSended struct {
	TaskID   string          `json:"task_id"`
	TaskName string          `json:"task_name"`
	Articles []SendedArticle `json:"articles"`
}

type DailySended struct {
	Date      string                `json:"date"`
	UpdatedAt string                `json:"updated_at"`
	Tasks     map[string]TaskSended `json:"tasks"`
}

// SentService 当日已发送记录服务
type SentService struct {
	cfgMgr *config.Manager
	mu     sync.Mutex
}

func NewSentService(cfgMgr *config.Manager) *SentService {
	return &SentService{cfgMgr: cfgMgr}
}

func (s *SentService) sentPath(date string) string {
	return filepath.Join(s.cfgMgr.RSSDateDir(date), "sended.json")
}

func (s *SentService) Load(date string) (*DailySended, error) {
	path := s.sentPath(date)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DailySended{
				Date:      date,
				UpdatedAt: time.Now().In(time.Local).Format(time.RFC3339),
				Tasks:     make(map[string]TaskSended),
			}, nil
		}
		return nil, err
	}

	var d DailySended
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if d.Date == "" {
		d.Date = date
	}
	if d.Tasks == nil {
		d.Tasks = make(map[string]TaskSended)
	}
	return &d, nil
}

func (s *SentService) Save(date string, d *DailySended) error {
	path := s.sentPath(date)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	d.UpdatedAt = time.Now().In(time.Local).Format(time.RFC3339)
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// GetSentIDs 获取某任务在指定日期已发送的文章ID集合
func (s *SentService) GetSentIDs(date, taskID string) (map[string]bool, error) {
	d, err := s.Load(date)
	if err != nil {
		return nil, err
	}
	res := make(map[string]bool)
	ts, ok := d.Tasks[taskID]
	if !ok {
		return res, nil
	}
	for _, a := range ts.Articles {
		if a.ArticleID != "" {
			res[a.ArticleID] = true
		}
	}
	return res, nil
}

// MarkSent 记录指定任务在当日已发送的文章（幂等：同一 article_id 不重复追加）
func (s *SentService) MarkSent(date string, task *model.Task, items []model.AIAnalysisItem) error {
	if task == nil || task.ID == "" || date == "" || len(items) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	d, err := s.Load(date)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	ts, ok := d.Tasks[task.ID]
	if ok {
		for _, a := range ts.Articles {
			if a.ArticleID != "" {
				existing[a.ArticleID] = true
			}
		}
	} else {
		ts = TaskSended{TaskID: task.ID, TaskName: task.Name, Articles: []SendedArticle{}}
	}

	now := time.Now().In(time.Local).Format(time.RFC3339)
	for _, it := range items {
		articleID := it.ArticleID
		if articleID == "" && it.Link != "" {
			articleID = generateArticleID(it.Link)
		}
		if articleID == "" || existing[articleID] {
			continue
		}
		existing[articleID] = true

		ts.Articles = append(ts.Articles, SendedArticle{
			ArticleID:    articleID,
			Title:        it.Title,
			Link:         it.Link,
			Source:       it.Source,
			PublishedAt:  it.PublishedAt,
			SentAt:       now,
			Importance:   it.Importance,
			Summary100CN: it.Summary,
		})
	}

	sort.Slice(ts.Articles, func(i, j int) bool { return ts.Articles[i].SentAt > ts.Articles[j].SentAt })
	d.Tasks[task.ID] = ts

	if err := s.Save(date, d); err != nil {
		return err
	}
	log.Info().Str("task", task.Name).Str("date", date).Int("sent_total", len(ts.Articles)).Msg("Updated sended.json")
	return nil
}
