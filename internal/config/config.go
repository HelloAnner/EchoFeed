package config

import (
	"os"
	"path/filepath"

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
	return &cfg, nil
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
	return &cfg, nil
}

// SaveSettings 保存系统设置
func (m *Manager) SaveSettings(cfg *model.Settings) error {
	return saveToml(m.settingsPath(), cfg)
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
