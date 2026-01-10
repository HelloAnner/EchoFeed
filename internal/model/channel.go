package model

// Channel 通知渠道
type Channel struct {
	ID      string            `toml:"id" json:"id"`
	Name    string            `toml:"name" json:"name"`
	Type    string            `toml:"type" json:"type"` // telegram/email/webhook/bark/wecom
	Enabled bool              `toml:"enabled" json:"enabled"`
	Remark  string            `toml:"remark" json:"remark"`
	Config  map[string]string `toml:"config" json:"config"`
}

// ChannelConfig 通知渠道配置文件结构
type ChannelConfig struct {
	Channels []Channel `toml:"channels"`
}

// 通知渠道类型常量
const (
	ChannelTypeTelegram = "telegram"
	ChannelTypeEmail    = "email"
	ChannelTypeWebhook  = "webhook"
	ChannelTypeBark     = "bark"
	ChannelTypeWecom    = "wecom"
)

// NotifyMessage 通知消息
type NotifyMessage struct {
	TaskID   string            `json:"task_id"`
	TaskName string            `json:"task_name"`
	Items    []AIAnalysisItem  `json:"items"`
}
