// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/echofeed/echofeed/internal/model"
)

// Manager 配置管理器
type Manager struct {
	DataDir string
}

// NewManager 创建配置管理器
func NewManager(dataDir string) *Manager {
	return &Manager{DataDir: dataDir}
}

// EnsureDataDir 确保数据目录存在
func (m *Manager) EnsureDataDir() error {
	dirs := []string{
		m.DataDir,
		filepath.Join(m.DataDir, "rss"),
		filepath.Join(m.DataDir, "logs"),
		filepath.Join(m.DataDir, "logs", "tasks"),
		filepath.Join(m.DataDir, "state"),
		filepath.Join(m.DataDir, "state", "tasks"),
		filepath.Join(m.DataDir, "state", "feeds"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

// feedsPath 获取RSS配置文件路径
func (m *Manager) feedsPath() string {
	return filepath.Join(m.DataDir, "rss.toml")
}

// tasksPath 获取任务配置文件路径
func (m *Manager) tasksPath() string {
	return filepath.Join(m.DataDir, "tasks.toml")
}

// channelsPath 获取通知渠道配置文件路径
func (m *Manager) channelsPath() string {
	return filepath.Join(m.DataDir, "channels.toml")
}

// botsPath 获取AI机器人配置文件路径
func (m *Manager) botsPath() string {
	return filepath.Join(m.DataDir, "bots.toml")
}

// settingsPath 获取系统设置文件路径
func (m *Manager) settingsPath() string {
	return filepath.Join(m.DataDir, "settings.toml")
}

// rssTagPath 获取RSS标签配置文件路径
func (m *Manager) rssTagPath() string {
	return filepath.Join(m.DataDir, "rss-tag.toml")
}

// LoadFeeds 加载RSS订阅配置
func (m *Manager) LoadFeeds() (*model.FeedConfig, error) {
	path := m.feedsPath()
	if !fileExists(path) {
		cfg := &model.FeedConfig{
			Settings: model.DefaultFeedSettings(),
			Feeds:    []model.Feed{},
		}
		if err := m.SaveFeeds(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg model.FeedConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveFeeds 保存RSS订阅配置
func (m *Manager) SaveFeeds(cfg *model.FeedConfig) error {
	return saveToml(m.feedsPath(), cfg)
}

// LoadTasks 加载任务配置
func (m *Manager) LoadTasks() (*model.TaskConfig, error) {
	path := m.tasksPath()
	if !fileExists(path) {
		cfg := &model.TaskConfig{Tasks: []model.Task{}}
		if err := m.SaveTasks(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg model.TaskConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// 兼容旧版任务配置：自动迁移到 filter_criteria / output_format
	if migrateTaskConfig(&cfg) {
		_ = m.SaveTasks(&cfg)
	}
	return &cfg, nil
}

func migrateTaskConfig(cfg *model.TaskConfig) bool {
	if cfg == nil || len(cfg.Tasks) == 0 {
		return false
	}

	changed := false
	for i := range cfg.Tasks {
		t := cfg.Tasks[i]

		// 默认最低重要度保持 3（与历史一致）
		if t.Prompt.MinImportance == 0 {
			t.Prompt.MinImportance = 3
			changed = true
		}

		// 迁移筛选条件：优先使用新字段；否则把旧 filter_prompt 迁移过来
		if strings.TrimSpace(t.Prompt.FilterCriteria) == "" {
			if strings.TrimSpace(t.Prompt.FilterPrompt) != "" {
				t.Prompt.FilterCriteria = t.Prompt.FilterPrompt
				t.Prompt.FilterPrompt = ""
				changed = true
			} else if len(t.Prompt.Keywords) > 0 {
				t.Prompt.FilterCriteria = "关注关键词：" + strings.Join(t.Prompt.Keywords, "、")
				changed = true
			}
		}

		// 迁移输出格式：为空则补默认模板
		if strings.TrimSpace(t.Prompt.OutputFormat) == "" {
			t.Prompt.OutputFormat = model.DefaultTaskOutputFormat
			changed = true
		}

		// 旧字段不再写回文件（保持 tasks.toml 干净）
		if t.Prompt.Template != "" {
			t.Prompt.Template = ""
			changed = true
		}

		cfg.Tasks[i] = t
	}
	return changed
}

// SaveTasks 保存任务配置
func (m *Manager) SaveTasks(cfg *model.TaskConfig) error {
	return saveToml(m.tasksPath(), cfg)
}

// LoadChannels 加载通知渠道配置
func (m *Manager) LoadChannels() (*model.ChannelConfig, error) {
	path := m.channelsPath()
	if !fileExists(path) {
		cfg := &model.ChannelConfig{Channels: []model.Channel{}}
		if err := m.SaveChannels(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg model.ChannelConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveChannels 保存通知渠道配置
func (m *Manager) SaveChannels(cfg *model.ChannelConfig) error {
	return saveToml(m.channelsPath(), cfg)
}

// LoadBots 加载AI机器人配置
func (m *Manager) LoadBots() (*model.BotConfig, error) {
	path := m.botsPath()
	if !fileExists(path) {
		cfg := &model.BotConfig{Bots: []model.Bot{}}
		if err := m.SaveBots(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg model.BotConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveBots 保存AI机器人配置
func (m *Manager) SaveBots(cfg *model.BotConfig) error {
	return saveToml(m.botsPath(), cfg)
}

// LoadSettings 加载系统设置
func (m *Manager) LoadSettings() (*model.Settings, error) {
	path := m.settingsPath()
	if !fileExists(path) {
		cfg := model.DefaultSettings()
		if err := m.SaveSettings(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	var cfg model.Settings
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}

	// 兼容旧版 settings.toml：补齐 backup 默认值
	if cfg.Backup.Unit == "" && cfg.Backup.At == "" && cfg.Backup.Every == 0 && cfg.Backup.Retain == 0 {
		cfg.Backup = model.DefaultSettings().Backup
	}
	return &cfg, nil
}

// SaveSettings 保存系统设置
func (m *Manager) SaveSettings(cfg *model.Settings) error {
	return saveToml(m.settingsPath(), cfg)
}

// LoadRSSTags 加载 RSS 标签配置
func (m *Manager) LoadRSSTags() (*model.RSSTagConfig, error) {
	path := m.rssTagPath()
	if !fileExists(path) {
		cfg := &model.RSSTagConfig{Tags: []string{}}
		if err := m.SaveRSSTags(cfg); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var cfg model.RSSTagConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	if cfg.Tags == nil {
		cfg.Tags = []string{}
	}
	return &cfg, nil
}

// SaveRSSTags 保存 RSS 标签配置
func (m *Manager) SaveRSSTags(cfg *model.RSSTagConfig) error {
	return saveToml(m.rssTagPath(), cfg)
}

// RSSCachePath 获取RSS缓存文件路径 (已废弃，保留兼容)
func (m *Manager) RSSCachePath(feedID string) string {
	return filepath.Join(m.DataDir, "rss", feedID+".json")
}

// RSSDateDir 获取指定日期的RSS目录 (格式: data/rss/2006-01-02/)
func (m *Manager) RSSDateDir(date string) string {
	return filepath.Join(m.DataDir, "rss", date)
}

// RSSOverviewPath 获取指定日期的overview.json路径
func (m *Manager) RSSOverviewPath(date string) string {
	return filepath.Join(m.RSSDateDir(date), "overview.json")
}

// RSSContentPath 获取文章内容文件路径
func (m *Manager) RSSContentPath(date, articleID string) string {
	return filepath.Join(m.RSSDateDir(date), articleID+".txt")
}

// EnsureRSSDateDir 确保指定日期的RSS目录存在
func (m *Manager) EnsureRSSDateDir(date string) error {
	return os.MkdirAll(m.RSSDateDir(date), 0755)
}
