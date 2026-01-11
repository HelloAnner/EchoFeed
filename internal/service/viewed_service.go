// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 当日已浏览（已输入给第一阶段大模型）记录：按天落盘到 data/rss/YYYY-MM-DD/viewed.json，
// 用于在“第一阶段：选择文章”前过滤掉已经尝试过但未被采纳的文章，减少 token 消耗。
// @author Anner
// Created on 2026/1/11
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

type ViewedArticle struct {
	ArticleID   string `json:"article_id"`
	Title       string `json:"title"`
	Link        string `json:"link"`
	Source      string `json:"source"`
	PublishedAt string `json:"published_at"`
	ViewedAt    string `json:"viewed_at"`
	Phase       string `json:"phase"`
}

type TaskViewed struct {
	TaskID   string          `json:"task_id"`
	TaskName string          `json:"task_name"`
	Articles []ViewedArticle `json:"articles"`
}

type DailyViewed struct {
	Date      string                `json:"date"`
	UpdatedAt string                `json:"updated_at"`
	Tasks     map[string]TaskViewed `json:"tasks"`
}

// ViewedService 当日已浏览记录服务（用于第一阶段去重）
type ViewedService struct {
	cfgMgr *config.Manager
	mu     sync.Mutex
}

func NewViewedService(cfgMgr *config.Manager) *ViewedService {
	return &ViewedService{cfgMgr: cfgMgr}
}

func (s *ViewedService) viewedPath(date string) string {
	return filepath.Join(s.cfgMgr.RSSDateDir(date), "viewed.json")
}

func (s *ViewedService) Load(date string) (*DailyViewed, error) {
	path := s.viewedPath(date)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &DailyViewed{
				Date:      date,
				UpdatedAt: time.Now().In(time.Local).Format(time.RFC3339),
				Tasks:     make(map[string]TaskViewed),
			}, nil
		}
		return nil, err
	}

	var d DailyViewed
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if d.Date == "" {
		d.Date = date
	}
	if d.Tasks == nil {
		d.Tasks = make(map[string]TaskViewed)
	}
	return &d, nil
}

func (s *ViewedService) Save(date string, d *DailyViewed) error {
	path := s.viewedPath(date)
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

// GetViewedIDs 获取某任务在指定日期“已浏览”的文章ID集合（第一阶段已输入给大模型）
func (s *ViewedService) GetViewedIDs(date, taskID string) (map[string]bool, error) {
	d, err := s.Load(date)
	if err != nil {
		return nil, err
	}
	res := make(map[string]bool)
	tv, ok := d.Tasks[taskID]
	if !ok {
		return res, nil
	}
	for _, a := range tv.Articles {
		if a.ArticleID != "" {
			res[a.ArticleID] = true
		}
	}
	return res, nil
}

// MarkViewed 记录指定任务在当日已浏览的文章（幂等：同一 article_id 不重复追加）
func (s *ViewedService) MarkViewed(date string, task *model.Task, articles []model.ArticleSummary, phase string) error {
	if task == nil || task.ID == "" || date == "" || len(articles) == 0 {
		return nil
	}
	if phase == "" {
		phase = "phase1"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	d, err := s.Load(date)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	tv, ok := d.Tasks[task.ID]
	if ok {
		for _, a := range tv.Articles {
			if a.ArticleID != "" {
				existing[a.ArticleID] = true
			}
		}
	} else {
		tv = TaskViewed{TaskID: task.ID, TaskName: task.Name, Articles: []ViewedArticle{}}
	}

	now := time.Now().In(time.Local).Format(time.RFC3339)
	for _, a := range articles {
		articleID := a.ID
		if articleID == "" && a.Link != "" {
			articleID = generateArticleID(a.Link)
		}
		if articleID == "" || existing[articleID] {
			continue
		}
		existing[articleID] = true

		publishedAt := ""
		if !a.Published.IsZero() {
			publishedAt = a.Published.In(time.Local).Format(time.RFC3339)
		}
		tv.Articles = append(tv.Articles, ViewedArticle{
			ArticleID:   articleID,
			Title:       a.Title,
			Link:        a.Link,
			Source:      a.FeedName,
			PublishedAt: publishedAt,
			ViewedAt:    now,
			Phase:       phase,
		})
	}

	sort.Slice(tv.Articles, func(i, j int) bool { return tv.Articles[i].ViewedAt > tv.Articles[j].ViewedAt })
	if len(tv.Articles) > 5000 {
		tv.Articles = tv.Articles[:5000]
	}

	d.Tasks[task.ID] = tv
	if err := s.Save(date, d); err != nil {
		return err
	}
	log.Info().Str("task", task.Name).Str("date", date).Int("viewed_total", len(tv.Articles)).Msg("Updated viewed.json")
	return nil
}

// ClearTask 清理某个任务在所有 viewed.json 中的记录（任务配置变更后用于重新评估）
func (s *ViewedService) ClearTask(taskID string) error {
	if taskID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	root := filepath.Join(s.cfgMgr.DataDir, "rss")
	entries, err := os.ReadDir(root)
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
		date := ent.Name()
		path := filepath.Join(root, date, "viewed.json")
		b, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			log.Warn().Err(err).Str("path", path).Msg("Failed to read viewed.json")
			continue
		}

		var d DailyViewed
		if err := json.Unmarshal(b, &d); err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to parse viewed.json")
			continue
		}
		if d.Date == "" {
			d.Date = date
		}
		if d.Tasks == nil {
			continue
		}
		if _, ok := d.Tasks[taskID]; !ok {
			continue
		}
		delete(d.Tasks, taskID)
		if len(d.Tasks) == 0 {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				log.Warn().Err(err).Str("path", path).Msg("Failed to remove empty viewed.json")
			}
			continue
		}
		if err := s.Save(date, &d); err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to update viewed.json after clearing task")
		}
	}
	return nil
}
