package handler

import (
	"fmt"
	"net/http"
	"strings"

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
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到渠道"})
		return
	}
	c.JSON(http.StatusOK, channel)
}

func channelTestErrorToChinese(err error) (string, string) {
	if err == nil {
		return "未知错误", ""
	}
	raw := strings.TrimSpace(err.Error())

	details := sanitizeDetails(raw)
	details = strings.ReplaceAll(details, "short response", "short response(响应过短)")
	switch {
	case strings.Contains(raw, "missing telegram config"):
		return "Telegram 配置不完整（需要 bot_token / chat_id）", details
	case strings.Contains(raw, "telegram API error"):
		return "Telegram 发送失败", details
	case strings.Contains(raw, "missing webhook url"):
		return "Webhook 配置不完整（缺少 URL）", details
	case strings.Contains(raw, "missing wecom webhook url"):
		return "企业微信 Webhook 配置不完整（缺少 URL）", details
	case strings.Contains(raw, "webhook error:"):
		return "Webhook 返回错误", details
	case strings.Contains(raw, "bark API error"):
		return "Bark 发送失败", details
	case strings.Contains(raw, "missing email config"):
		return "Email 配置不完整（需要 smtp_host / smtp_port / to）", details
	case strings.Contains(raw, "failed to connect to SMTP server"):
		return "连接 SMTP 服务器失败", details
	case strings.Contains(raw, "failed to create SMTP client"):
		return "SMTP 握手失败（服务器响应异常）", details
	case strings.Contains(raw, "SMTP auth failed"):
		return "SMTP 认证失败", details
	case strings.Contains(raw, "failed to send email"):
		return "发送邮件失败", details
	case strings.HasPrefix(raw, "smtp_mail:") || strings.HasPrefix(raw, "smtp_rcpt:") || strings.HasPrefix(raw, "smtp_data:") || strings.HasPrefix(raw, "smtp_write:") || strings.HasPrefix(raw, "smtp_close:") || strings.HasPrefix(raw, "smtp_quit:"):
		return "发送邮件失败（SMTP 交互异常）", details
	case strings.Contains(raw, "short response"):
		return "SMTP 服务器响应异常（响应过短）", details
	case strings.Contains(raw, "unsupported channel type"):
		return "不支持的渠道类型", details
	default:
		return "测试发送失败", details
	}
}

func sanitizeDetails(s string) string {
	if s == "" {
		return ""
	}
	// 将控制字符替换为可读的 \xNN
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteRune(r)
			continue
		}
		if r < 32 || r == 127 {
			b.WriteString(fmt.Sprintf("\\x%02X", r))
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	return strings.TrimSpace(out)
}

func ensureChannelDefaults(channel *model.Channel) {
	if channel == nil {
		return
	}
	if channel.Config == nil {
		channel.Config = make(map[string]string)
	}
}

// Create 创建渠道
func (h *ChannelHandler) Create(c *gin.Context) {
	var channel model.Channel
	if err := c.ShouldBindJSON(&channel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ensureChannelDefaults(&channel)
	notifier := service.NewNotifierService()
	if err := notifier.Test(&channel); err != nil {
		msg, details := channelTestErrorToChinese(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "保存失败：" + msg,
			"details": details,
		})
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

	ensureChannelDefaults(&channel)
	notifier := service.NewNotifierService()
	if err := notifier.Test(&channel); err != nil {
		msg, details := channelTestErrorToChinese(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "保存失败：" + msg,
			"details": details,
		})
		return
	}

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
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到渠道"})
		return
	}

	// 发送测试通知
	notifier := service.NewNotifierService()
	if err := notifier.Test(channel); err != nil {
		msg, details := channelTestErrorToChinese(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "测试失败：" + msg, "details": details})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "测试消息已发送"})
}
