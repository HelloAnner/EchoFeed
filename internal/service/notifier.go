package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/model"
)

// NotifierService 通知服务
type NotifierService struct {
	client *http.Client
}

// NewNotifierService 创建通知服务
func NewNotifierService() *NotifierService {
	return &NotifierService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Send 发送通知
func (s *NotifierService) Send(channel *model.Channel, msg *model.NotifyMessage) error {
	switch channel.Type {
	case model.ChannelTypeTelegram:
		return s.sendTelegram(channel, msg)
	case model.ChannelTypeWebhook:
		return s.sendWebhook(channel, msg)
	case model.ChannelTypeBark:
		return s.sendBark(channel, msg)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// Test 测试通知渠道
func (s *NotifierService) Test(channel *model.Channel) error {
	testMsg := &model.NotifyMessage{
		TaskID:   "test",
		TaskName: "测试任务",
		Items: []model.AIAnalysisItem{
			{
				Title:      "测试消息",
				Summary:    "这是一条来自EchoFeed的测试通知",
				Importance: 5,
				Source:     "EchoFeed",
				Link:       "https://github.com/echofeed/echofeed",
			},
		},
	}
	return s.Send(channel, testMsg)
}

// sendTelegram 发送Telegram通知
func (s *NotifierService) sendTelegram(channel *model.Channel, msg *model.NotifyMessage) error {
	botToken := channel.Config["bot_token"]
	chatID := channel.Config["chat_id"]

	if botToken == "" || chatID == "" {
		return fmt.Errorf("missing telegram config: bot_token or chat_id")
	}

	// 构建消息文本
	text := s.formatMessage(msg)

	// 发送请求
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	reqBody := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API error: %s", string(body))
	}

	log.Info().Str("channel", channel.Name).Msg("Telegram notification sent")
	return nil
}

// sendWebhook 发送Webhook通知
func (s *NotifierService) sendWebhook(channel *model.Channel, msg *model.NotifyMessage) error {
	url := channel.Config["url"]
	method := channel.Config["method"]

	if url == "" {
		return fmt.Errorf("missing webhook url")
	}
	if method == "" {
		method = "POST"
	}

	// 构建请求体
	payload := map[string]interface{}{
		"task_id":   msg.TaskID,
		"task_name": msg.TaskName,
		"items":     msg.Items,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加自定义headers
	if headers := channel.Config["headers"]; headers != "" {
		var headerMap map[string]string
		if err := json.Unmarshal([]byte(headers), &headerMap); err == nil {
			for k, v := range headerMap {
				req.Header.Set(k, v)
			}
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook error: %d - %s", resp.StatusCode, string(body))
	}

	log.Info().Str("channel", channel.Name).Msg("Webhook notification sent")
	return nil
}

// sendBark 发送Bark通知
func (s *NotifierService) sendBark(channel *model.Channel, msg *model.NotifyMessage) error {
	serverURL := channel.Config["server_url"]
	deviceKey := channel.Config["device_key"]

	if serverURL == "" {
		serverURL = "https://api.day.app"
	}
	if deviceKey == "" {
		return fmt.Errorf("missing bark device_key")
	}

	// 构建标题和内容
	title := fmt.Sprintf("EchoFeed: %s", msg.TaskName)
	var bodyParts []string
	for _, item := range msg.Items {
		bodyParts = append(bodyParts, fmt.Sprintf("• %s", item.Title))
	}
	body := strings.Join(bodyParts, "\n")

	// 发送请求
	url := fmt.Sprintf("%s/%s/%s/%s", serverURL, deviceKey, title, body)
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bark API error: %s", string(respBody))
	}

	log.Info().Str("channel", channel.Name).Msg("Bark notification sent")
	return nil
}

// formatMessage 格式化通知消息
func (s *NotifierService) formatMessage(msg *model.NotifyMessage) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*📰 %s*\n\n", msg.TaskName))

	for i, item := range msg.Items {
		if i >= 5 {
			sb.WriteString(fmt.Sprintf("\n... 还有 %d 条内容", len(msg.Items)-5))
			break
		}

		sb.WriteString(fmt.Sprintf("*%s*\n", item.Title))
		sb.WriteString(fmt.Sprintf("%s\n", item.Summary))
		sb.WriteString(fmt.Sprintf("📍 %s | ⭐ %d/5\n", item.Source, item.Importance))
		sb.WriteString(fmt.Sprintf("[阅读原文](%s)\n\n", item.Link))
	}

	return sb.String()
}
