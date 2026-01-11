// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
	"github.com/echofeed/echofeed/internal/scheduler"
	"github.com/echofeed/echofeed/internal/service"
)

// SettingsHandler 系统设置处理器
type SettingsHandler struct {
	cfgMgr *config.Manager
	sched  *scheduler.Scheduler
}

// NewSettingsHandler 创建系统设置处理器
func NewSettingsHandler(cfgMgr *config.Manager, sched *scheduler.Scheduler) *SettingsHandler {
	return &SettingsHandler{cfgMgr: cfgMgr, sched: sched}
}

// Get 获取系统设置
func (h *SettingsHandler) Get(c *gin.Context) {
	settings, err := h.cfgMgr.LoadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, settings)
}

// Update 更新系统设置
func (h *SettingsHandler) Update(c *gin.Context) {
	var settings model.Settings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 保留一些敏感字段不被覆盖
	current, err := h.cfgMgr.LoadSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保留密码哈希（如果前端没有提供）
	if settings.Auth.PasswordHash == "" {
		settings.Auth.PasswordHash = current.Auth.PasswordHash
	}

	// 兼容旧前端未携带 backup 字段：保持原配置
	if settings.Backup.At == "" && settings.Backup.Every == 0 && settings.Backup.Unit == "" && settings.Backup.Retain == 0 && !settings.Backup.Enabled {
		settings.Backup = current.Backup
	}

	// 兜底：备份配置参数修正
	if settings.Backup.Every <= 0 {
		settings.Backup.Every = 1
	}
	if settings.Backup.Retain <= 0 {
		settings.Backup.Retain = 7
	}
	if settings.Backup.At == "" {
		settings.Backup.At = "03:00"
	}
	if settings.Backup.Unit == "" {
		settings.Backup.Unit = "day"
	}

	if err := h.cfgMgr.SaveSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	service.ConfigureAppLogger(h.cfgMgr.DataDir, settings.Log)
	if h.sched != nil {
		h.sched.ReloadRSSFetch()
		h.sched.ReloadBackup()
	}

	c.JSON(http.StatusOK, settings)
}
