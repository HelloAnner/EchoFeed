package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/model"
	"github.com/echofeed/echofeed/internal/service"
)

// ChannelHandler 通知渠道API处理器
type ChannelHandler struct {
	channelSvc *service.ChannelService
}

// NewChannelHandler 创建通知渠道API处理器
func NewChannelHandler(channelSvc *service.ChannelService) *ChannelHandler {
	return &ChannelHandler{channelSvc: channelSvc}
}

// List 获取渠道列表
func (h *ChannelHandler) List(c *gin.Context) {
	channels, err := h.channelSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, channels)
}

// Get 获取单个渠道
func (h *ChannelHandler) Get(c *gin.Context) {
	id := c.Param("id")
	channel, err := h.channelSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

// Create 创建渠道
func (h *ChannelHandler) Create(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.channelSvc.Create(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, channel)
}

// Update 更新渠道
func (h *ChannelHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	channel.ID = id

	if err := h.channelSvc.Update(&channel); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, channel)
}

// Delete 删除渠道
func (h *ChannelHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.channelSvc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Test 测试渠道
func (h *ChannelHandler) Test(c *gin.Context) {
	id := c.Param("id")
	channel, err := h.channelSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	// 发送测试通知
	notifier := service.NewNotifierService()
	if err := notifier.Test(channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Test notification sent"})
}
