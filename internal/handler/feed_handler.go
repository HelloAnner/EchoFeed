package handler

import (
	"net/http"

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
		c.JSON(http.StatusNotFound, gin.H{"error": "Feed not found"})
		return
	}
	c.JSON(http.StatusOK, feed)
}

// Create 创建订阅
func (h *FeedHandler) Create(c *gin.Context) {
	var feed model.Feed
	if err := c.ShouldBindJSON(&feed); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Refresh 手动刷新订阅
func (h *FeedHandler) Refresh(c *gin.Context) {
	id := c.Param("id")
	if err := h.sched.TriggerFeedRefresh(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Refresh triggered"})
}
