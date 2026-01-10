package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/model"
	"github.com/echofeed/echofeed/internal/scheduler"
	"github.com/echofeed/echofeed/internal/service"
)

// FeedHandler RSS订阅API处理器
type FeedHandler struct {
	feedSvc *service.FeedService
	sched   *scheduler.Scheduler
}

// NewFeedHandler 创建RSS订阅API处理器
func NewFeedHandler(feedSvc *service.FeedService, sched *scheduler.Scheduler) *FeedHandler {
	return &FeedHandler{feedSvc: feedSvc, sched: sched}
}

// List 获取订阅列表
func (h *FeedHandler) List(c *gin.Context) {
	feeds, err := h.feedSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, feeds)
}

// Get 获取单个订阅
func (h *FeedHandler) Get(c *gin.Context) {
	id := c.Param("id")
	feed, err := h.feedSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if feed == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到订阅"})
		return
	}
	c.JSON(http.StatusOK, feed)
}

func feedTestErrorToChinese(err error) (string, string) {
	raw := strings.TrimSpace(err.Error())
	details := raw

	switch {
	case strings.HasPrefix(raw, "build_request:"):
		return "URL 格式不正确", strings.TrimSpace(strings.TrimPrefix(raw, "build_request:"))
	case strings.HasPrefix(raw, "http_status:"):
		codeStr := strings.TrimSpace(strings.TrimPrefix(raw, "http_status:"))
		if code, parseErr := strconv.Atoi(codeStr); parseErr == nil && code > 0 {
			return fmt.Sprintf("RSS 订阅返回错误状态（HTTP %d）", code), ""
		}
		return "RSS 订阅返回错误状态", codeStr
	case strings.HasPrefix(raw, "request:"):
		return "RSS 订阅无法访问", strings.TrimSpace(strings.TrimPrefix(raw, "request:"))
	case strings.HasPrefix(raw, "parse:"):
		return "RSS 订阅解析失败", strings.TrimSpace(strings.TrimPrefix(raw, "parse:"))
	case strings.HasPrefix(raw, "read:"):
		return "读取 RSS 内容失败", strings.TrimSpace(strings.TrimPrefix(raw, "read:"))
	default:
		return "RSS 订阅不可用", details
	}
}

// Create 创建订阅
func (h *FeedHandler) Create(c *gin.Context) {
	var feed model.Feed
	if err := c.ShouldBindJSON(&feed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, _, err := h.feedSvc.TestFeedURL(feed.URL); err != nil {
		msg, details := feedTestErrorToChinese(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "保存失败：" + msg,
			"details": details,
		})
		return
	}

	if err := h.feedSvc.Create(&feed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, feed)
}

// Update 更新订阅
func (h *FeedHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var feed model.Feed
	if err := c.ShouldBindJSON(&feed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	feed.ID = id

	if _, _, err := h.feedSvc.TestFeedURL(feed.URL); err != nil {
		msg, details := feedTestErrorToChinese(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "保存失败：" + msg,
			"details": details,
		})
		return
	}

	if err := h.feedSvc.Update(&feed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, feed)
}

// Delete 删除订阅
func (h *FeedHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.feedSvc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

// Refresh 手动刷新订阅
func (h *FeedHandler) Refresh(c *gin.Context) {
	id := c.Param("id")
	if err := h.sched.TriggerFeedRefresh(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "刷新已触发"})
}

// RefreshAll 手动刷新全部订阅
func (h *FeedHandler) RefreshAll(c *gin.Context) {
	go func() {
		_ = h.feedSvc.FetchAll()
	}()
	c.JSON(http.StatusOK, gin.H{"message": "已触发更新全部订阅"})
}
