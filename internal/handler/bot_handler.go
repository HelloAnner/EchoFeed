package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/model"
	"github.com/echofeed/echofeed/internal/service"
)

// BotHandler AI机器人API处理器
type BotHandler struct {
	botSvc *service.BotService
}

// NewBotHandler 创建AI机器人API处理器
func NewBotHandler(botSvc *service.BotService) *BotHandler {
	return &BotHandler{botSvc: botSvc}
}

// List 获取机器人列表
func (h *BotHandler) List(c *gin.Context) {
	bots, err := h.botSvc.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bots)
}

// Get 获取单个机器人
func (h *BotHandler) Get(c *gin.Context) {
	id := c.Param("id")
	bot, err := h.botSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bot == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
		return
	}
	c.JSON(http.StatusOK, bot)
}

// Create 创建机器人
func (h *BotHandler) Create(c *gin.Context) {
	var bot model.Bot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.botSvc.Create(&bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bot)
}

// Update 更新机器人
func (h *BotHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var bot model.Bot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bot.ID = id

	if err := h.botSvc.Update(&bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bot)
}

// Delete 删除机器人
func (h *BotHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.botSvc.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// Test 测试机器人
func (h *BotHandler) Test(c *gin.Context) {
	id := c.Param("id")
	bot, err := h.botSvc.Get(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if bot == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bot not found"})
		return
	}

	// 测试AI服务
	aiSvc := service.NewAIService()
	provider := aiSvc.CreateProviderFromBot(bot)
	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider: " + bot.Provider})
		return
	}

	// 发送简单测试请求
	result, err := provider.Analyze("请用中文回复：你好", "测试内容")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AI test successful", "result": result})
}
