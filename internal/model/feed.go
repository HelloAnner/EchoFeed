package model

import "time"

// FeedSettings RSS全局设置
type FeedSettings struct {
	DefaultInterval int    `toml:"default_interval"` // 默认拉取间隔(分钟)
	Timeout         int    `toml:"timeout"`          // 请求超时(秒)
	UserAgent       string `toml:"user_agent"`       // User-Agent
	MaxConcurrent   int    `toml:"max_concurrent"`   // 最大并发数
}

// Feed RSS订阅源
type Feed struct {
	ID       string `toml:"id" json:"id"`
	Name     string `toml:"name" json:"name"`
	URL      string `toml:"url" json:"url"`
	Remark   string `toml:"remark" json:"remark"`
	Interval int    `toml:"interval,omitempty" json:"interval"` // 可选，覆盖默认间隔
	Enabled  bool   `toml:"enabled" json:"enabled"`
}

// FeedConfig RSS配置文件结构
type FeedConfig struct {
	Settings FeedSettings `toml:"settings"`
	Feeds    []Feed       `toml:"feeds"`
}

// Article RSS文章
type Article struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Link      string    `json:"link"`
	Content   string    `json:"content"`
	Published time.Time `json:"published"`
	Fetched   time.Time `json:"fetched"`
}

// ArticleSummary 文章摘要(用于overview.json)
type ArticleSummary struct {
	ID          string    `json:"id"`
	FeedID      string    `json:"feed_id"`
	FeedName    string    `json:"feed_name"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Summary     string    `json:"summary"`      // 内容摘要(前200字)
	ContentFile string    `json:"content_file"` // 完整内容文件路径
	Published   time.Time `json:"published"`
	Fetched     time.Time `json:"fetched"`
}

// DailyOverview 每日摘要(overview.json结构)
type DailyOverview struct {
	Date       string           `json:"date"`
	UpdatedAt  time.Time        `json:"updated_at"`
	TotalCount int              `json:"total_count"`
	Articles   []ArticleSummary `json:"articles"`
}

// FeedCache RSS缓存
type FeedCache struct {
	FeedID    string    `json:"feed_id"`
	FeedName  string    `json:"feed_name"`
	LastFetch time.Time `json:"last_fetch"`
	Items     []Article `json:"items"`
}

// DefaultFeedSettings 默认设置
func DefaultFeedSettings() FeedSettings {
	return FeedSettings{
		DefaultInterval: 5,
		Timeout:         30,
		UserAgent:       "EchoFeed/1.0",
		MaxConcurrent:   5,
	}
}
