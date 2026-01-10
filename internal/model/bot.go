package model

// Bot AI机器人配置
type Bot struct {
	ID       string            `toml:"id" json:"id"`
	Name     string            `toml:"name" json:"name"`
	Remark   string            `toml:"remark" json:"remark"`
	Provider string            `toml:"provider" json:"provider"` // openai/claude/ollama
	Enabled  bool              `toml:"enabled" json:"enabled"`
	Config   map[string]string `toml:"config" json:"config"`
}

// BotConfig AI机器人配置文件结构
type BotConfig struct {
	Bots []Bot `toml:"bots"`
}

// AI Provider类型常量
const (
	ProviderOpenAI = "openai"
	ProviderClaude = "claude"
	ProviderOllama = "ollama"
)
