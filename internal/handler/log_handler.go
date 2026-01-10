package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/service"
)

// LogHandler 日志处理器
type LogHandler struct {
	logSvc *service.LogService
}

// NewLogHandler 创建日志处理器
func NewLogHandler(logSvc *service.LogService) *LogHandler {
	return &LogHandler{logSvc: logSvc}
}

// List 获取日志列表
func (h *LogHandler) List(c *gin.Context) {
	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	taskID := c.Query("task_id")

	var logs interface{}
	var err error

	if taskID != "" {
		logs, err = h.logSvc.ListByTask(taskID, limit)
	} else {
		logs, err = h.logSvc.List(limit)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// GetStats 获取统计信息
func (h *LogHandler) GetStats(c *gin.Context) {
	stats, err := h.logSvc.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetStatsToday 获取今日统计
func (h *LogHandler) GetStatsToday(c *gin.Context) {
	stats, err := h.logSvc.GetStatsToday()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ClearAll 一键清理执行日志
func (h *LogHandler) ClearAll(c *gin.Context) {
	if err := h.logSvc.ClearAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Cleared"})
}
