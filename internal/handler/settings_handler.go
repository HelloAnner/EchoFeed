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

	if err := h.cfgMgr.SaveSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	service.ConfigureAppLogger(h.cfgMgr.DataDir, settings.Log)
	if h.sched != nil {
		h.sched.ReloadRSSFetch()
	}

	c.JSON(http.StatusOK, settings)
}
