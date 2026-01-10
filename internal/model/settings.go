package model

// Settings 系统设置
type Settings struct {
	Server ServerSettings `toml:"server"`
	Auth   AuthSettings   `toml:"auth"`
	Fetch  FetchSettings  `toml:"fetch"`
	Log    LogSettings    `toml:"log"`
}

// ServerSettings 服务器设置
type ServerSettings struct {
	Port int    `toml:"port"`
	Host string `toml:"host"`
}

// AuthSettings 认证设置
type AuthSettings struct {
	Enabled      bool   `toml:"enabled"`
	Username     string `toml:"username"`
	PasswordHash string `toml:"password_hash"`
}

// FetchSettings 拉取设置
type FetchSettings struct {
	DefaultInterval int `toml:"default_interval"` // 默认RSS拉取间隔(分钟)
	Timeout         int `toml:"timeout"`          // 请求超时(秒)
	MaxConcurrent   int `toml:"max_concurrent"`   // 最大并发数
	RetentionDays   int `toml:"retention_days"`   // RSS内容保留天数
}

// LogSettings 日志设置
type LogSettings struct {
	Level      string `toml:"level"`
	MaxSizeMB  int    `toml:"max_size_mb"`
	MaxBackups int    `toml:"max_backups"`
}

// DefaultSettings 默认系统设置
func DefaultSettings() Settings {
	return Settings{
		Server: ServerSettings{
			Port: 8080,
			Host: "0.0.0.0",
		},
		Auth: AuthSettings{
			Enabled:  false,
			Username: "admin",
		},
		Fetch: FetchSettings{
			DefaultInterval: 5,
			Timeout:         30,
			MaxConcurrent:   5,
			RetentionDays:   7,
		},
		Log: LogSettings{
			Level:      "info",
			MaxSizeMB:  100,
			MaxBackups: 3,
		},
	}
}
