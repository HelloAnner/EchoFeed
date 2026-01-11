// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package model

// Settings 系统设置
type Settings struct {
	Server ServerSettings `toml:"server" json:"server"`
	Auth   AuthSettings   `toml:"auth" json:"auth"`
	Fetch  FetchSettings  `toml:"fetch" json:"fetch"`
	Log    LogSettings    `toml:"log" json:"log"`
	Backup BackupSettings `toml:"backup" json:"backup"`
}

// ServerSettings 服务器设置
type ServerSettings struct {
	Port int    `toml:"port" json:"port"`
	Host string `toml:"host" json:"host"`
}

// AuthSettings 认证设置
type AuthSettings struct {
	Enabled      bool   `toml:"enabled" json:"enabled"`
	Username     string `toml:"username" json:"username"`
	PasswordHash string `toml:"password_hash" json:"password_hash"`
}

// FetchSettings 拉取设置
type FetchSettings struct {
	DefaultInterval int `toml:"default_interval" json:"default_interval"` // 默认RSS拉取间隔(分钟)
	Timeout         int `toml:"timeout" json:"timeout"`                   // 请求超时(秒)
	MaxConcurrent   int `toml:"max_concurrent" json:"max_concurrent"`     // 最大并发数
	RetentionDays   int `toml:"retention_days" json:"retention_days"`     // RSS内容保留天数
}

// LogSettings 日志设置
type LogSettings struct {
	Level      string `toml:"level" json:"level"`
	MaxSizeMB  int    `toml:"max_size_mb" json:"max_size_mb"`
	MaxBackups int    `toml:"max_backups" json:"max_backups"`
}

// BackupSettings 备份设置
type BackupSettings struct {
	Enabled bool   `toml:"enabled" json:"enabled"` // 是否启用备份
	Every   int    `toml:"every" json:"every"`     // 周期数值（结合 unit）
	Unit    string `toml:"unit" json:"unit"`       // day/hour
	At      string `toml:"at" json:"at"`           // 时间点（HH:MM）
	Retain  int    `toml:"retain" json:"retain"`   // 保留最近 N 次备份
}

// DefaultSettings 默认系统设置
func DefaultSettings() Settings {
	return Settings{
		Server: ServerSettings{
			Port: 33333,
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
		Backup: BackupSettings{
			Enabled: true,
			Every:   1,
			Unit:    "day",
			At:      "03:00",
			Retain:  7,
		},
	}
}
