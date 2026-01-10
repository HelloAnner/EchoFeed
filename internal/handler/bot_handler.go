package handler

import (
	"net/http"
	"strings"

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
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到机器人"})
		return
	}
	c.JSON(http.StatusOK, bot)
}

func ensureBotDefaults(bot *model.Bot) {
	if bot == nil {
		return
	}
	if bot.Config == nil {
		bot.Config = make(map[string]string)
	}
}

func botTestErrorToChinese(err error, provider string) (string, string) {
	if err == nil {
		return "未知错误", ""
	}
	raw := strings.TrimSpace(err.Error())

	if strings.Contains(raw, "Unsupported provider") || strings.Contains(raw, "unsupported provider") {
		return "不支持的提供商: " + provider, raw
	}
	if strings.Contains(raw, "missing") && strings.Contains(raw, "api_key") {
		return "API Key 未配置或不正确", raw
	}
	if strings.Contains(raw, "no response") {
		return "AI 服务无响应", raw
	}
	if strings.Contains(raw, "OpenAI API error") {
		return "AI 调用失败（OpenAI 兼容接口）", raw
	}
	if strings.Contains(raw, "Claude API error") || strings.Contains(raw, "anthropic") {
		return "AI 调用失败（Claude）", raw
	}
	if strings.Contains(raw, "connection refused") || strings.Contains(raw, "connect:") {
		return "AI 服务无法连接", raw
	}
	if strings.Contains(raw, "timeout") || strings.Contains(raw, "context deadline") {
		return "AI 服务请求超时", raw
	}
	return "AI 测试失败", raw
}

func testBotConfig(bot *model.Bot) (string, string, bool) {
	aiSvc := service.NewAIService()
	provider := aiSvc.CreateProviderFromBot(bot)
	if provider == nil {
		return "不支持的提供商: " + bot.Provider, "provider=" + bot.Provider, false
	}
	_, err := provider.Analyze("请用中文回复：你好", "测试内容")
	if err != nil {
		msg, details := botTestErrorToChinese(err, bot.Provider)
		return msg, details, false
	}
	return "", "", true
}

// Create 创建机器人
func (h *BotHandler) Create(c *gin.Context) {
	var bot model.Bot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ensureBotDefaults(&bot)
	if msg, details, ok := testBotConfig(&bot); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "保存失败：" + msg, "details": details})
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

	ensureBotDefaults(&bot)
	if msg, details, ok := testBotConfig(&bot); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "保存失败：" + msg, "details": details})
		return
	}

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
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到机器人"})
		return
	}

	// 测试AI服务
	ensureBotDefaults(bot)
	aiSvc := service.NewAIService()
	provider := aiSvc.CreateProviderFromBot(bot)
	if provider == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的提供商: " + bot.Provider})
		return
	}

	// 发送简单测试请求
	result, err := provider.Analyze("请用中文回复：你好", "测试内容")
	if err != nil {
		msg, details := botTestErrorToChinese(err, bot.Provider)
		c.JSON(http.StatusBadRequest, gin.H{"error": "测试失败：" + msg, "details": details})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AI 测试成功", "result": result})
}
