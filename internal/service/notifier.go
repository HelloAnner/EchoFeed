package service

import (
	"bytes"
	"crypto/tls"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/model"
)

// NotifierService 通知服务
type NotifierService struct {
	client *http.Client
}

type PartialSendError struct {
	Err            error
	SentArticleIDs []string
}

func (e *PartialSendError) Error() string {
	if e == nil || e.Err == nil {
		return "partial send failed"
	}
	return e.Err.Error()
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
	case model.ChannelTypeWecom:
		return s.sendWecom(channel, msg)
	case model.ChannelTypeBark:
		return s.sendBark(channel, msg)
	case model.ChannelTypeEmail:
		return s.sendEmail(channel, msg)
	default:
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// sendWecom 发送企业微信机器人通知（单篇文章一条消息）
func (s *NotifierService) sendWecom(channel *model.Channel, msg *model.NotifyMessage) error {
	url := channel.Config["webhook_url"]
	if url == "" {
		url = channel.Config["url"]
	}
	method := channel.Config["method"]
	if method == "" {
		method = "POST"
	}
	template := channel.Config["body_template"]

	if url == "" {
		return fmt.Errorf("missing wecom webhook url")
	}
	return s.sendWecomMessages(url, method, template, msg, channel)
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

// sendWebhook 发送Webhook通知（支持企业微信和自定义模板）
func (s *NotifierService) sendWebhook(channel *model.Channel, msg *model.NotifyMessage) error {
	url := channel.Config["url"]
	method := channel.Config["method"]
	bodyTemplate := channel.Config["body_template"]
	webhookType := channel.Config["webhook_type"] // wecom/custom

	if url == "" {
		return fmt.Errorf("missing webhook url")
	}
	if method == "" {
		method = "POST"
	}

	// 自定义Webhook：按原有逻辑一次性发送整包
	if webhookType != "wecom" && webhookType != "" {
		jsonData, err := s.buildCustomPayload(msg, bodyTemplate)
		if err != nil {
			return err
		}
		return s.sendWebhookRequest(url, method, jsonData, channel)
	}

	// 企业微信：单篇文章一条消息
	return s.sendWecomMessages(url, method, bodyTemplate, msg, channel)
}

func (s *NotifierService) sendWecomMessages(url, method, template string, msg *model.NotifyMessage, channel *model.Channel) error {
	if msg == nil || len(msg.Items) == 0 {
		return nil
	}

	var firstErr error
	var sentIDs []string
	for _, item := range msg.Items {
		single := &model.NotifyMessage{
			TaskID:   msg.TaskID,
			TaskName: msg.TaskName,
			Items:    []model.AIAnalysisItem{item},
		}

		jsonData, err := s.buildWecomPayload(single, template)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		if err := s.sendWebhookRequest(url, method, jsonData, channel); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if item.ArticleID != "" {
			sentIDs = append(sentIDs, item.ArticleID)
		}
	}

	if firstErr != nil {
		if len(sentIDs) > 0 {
			return &PartialSendError{Err: firstErr, SentArticleIDs: sentIDs}
		}
		return firstErr
	}
	return nil
}

func (s *NotifierService) sendWebhookRequest(url, method string, jsonData []byte, channel *model.Channel) error {
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

// buildWecomPayload 构建企业微信消息体
func (s *NotifierService) buildWecomPayload(msg *model.NotifyMessage, template string) ([]byte, error) {
	var content string

	if template != "" {
		// 使用用户自定义模板
		content = s.applyTemplate(template, msg)
	} else {
		// 使用默认企业微信Markdown模板
		content = s.formatWecomMarkdown(msg)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	return json.Marshal(payload)
}

// buildCustomPayload 构建自定义模板消息体
func (s *NotifierService) buildCustomPayload(msg *model.NotifyMessage, template string) ([]byte, error) {
	if template == "" {
		// 默认JSON格式
		payload := map[string]interface{}{
			"task_id":   msg.TaskID,
			"task_name": msg.TaskName,
			"items":     msg.Items,
			"timestamp": time.Now().Format(time.RFC3339),
		}
		return json.Marshal(payload)
	}

	// 应用模板替换
	content := s.applyTemplate(template, msg)
	return []byte(content), nil
}

// applyTemplate 应用模板参数替换
func (s *NotifierService) applyTemplate(template string, msg *model.NotifyMessage) string {
	// 替换基础参数
	result := template
	result = strings.ReplaceAll(result, "{{task_name}}", msg.TaskName)
	result = strings.ReplaceAll(result, "{{task_id}}", msg.TaskID)
	result = strings.ReplaceAll(result, "{{timestamp}}", time.Now().Format("2006-01-02 15:04:05"))
	result = strings.ReplaceAll(result, "{{date}}", time.Now().Format("2006-01-02"))
	result = strings.ReplaceAll(result, "{{count}}", fmt.Sprintf("%d", len(msg.Items)))

	// 替换AI输出（完整列表）
	var aiOutput strings.Builder
	for i, item := range msg.Items {
		if i > 0 {
			aiOutput.WriteString("\n\n")
		}
		aiOutput.WriteString(fmt.Sprintf("**%s**\n", item.Title))
		aiOutput.WriteString(fmt.Sprintf("%s\n", item.Summary))
		aiOutput.WriteString(fmt.Sprintf("> 来源: %s | 重要度: %d/5\n", item.Source, item.Importance))
		aiOutput.WriteString(fmt.Sprintf("[阅读原文](%s)", item.Link))
	}
	result = strings.ReplaceAll(result, "{{ai_output}}", aiOutput.String())

	// 替换简洁列表
	var titleList strings.Builder
	for _, item := range msg.Items {
		titleList.WriteString(fmt.Sprintf("- [%s](%s)\n", item.Title, item.Link))
	}
	result = strings.ReplaceAll(result, "{{title_list}}", titleList.String())

	// 替换第一条内容（摘要）
	if len(msg.Items) > 0 {
		first := msg.Items[0]
		result = strings.ReplaceAll(result, "{{first_title}}", first.Title)
		result = strings.ReplaceAll(result, "{{first_summary}}", first.Summary)
		result = strings.ReplaceAll(result, "{{first_link}}", first.Link)
	}

	return result
}

// formatWecomMarkdown 格式化企业微信Markdown消息
func (s *NotifierService) formatWecomMarkdown(msg *model.NotifyMessage) string {
	if msg == nil || len(msg.Items) == 0 {
		return "## EchoFeed\n\n> 无可推送内容"
	}

	item := msg.Items[0]

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 📰 %s\n\n", msg.TaskName))
	sb.WriteString(fmt.Sprintf("### %s\n\n", item.Title))

	meta := fmt.Sprintf("信息来源：%s", item.Source)
	if item.PublishedAt != "" {
		meta = meta + " ｜ 发布时间：" + item.PublishedAt
	}
	meta = meta + fmt.Sprintf(" ｜ 重要性：%d/5", item.Importance)
	sb.WriteString(fmt.Sprintf("> <font color=\"comment\">%s</font>\n\n", meta))

	sb.WriteString(fmt.Sprintf("%s\n\n", item.Summary))
	if item.Link != "" {
		sb.WriteString(fmt.Sprintf("[原文链接](%s)\n", item.Link))
	}
	return sb.String()
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

// sendEmail 发送邮件通知
func (s *NotifierService) sendEmail(channel *model.Channel, msg *model.NotifyMessage) error {
	smtpHost := channel.Config["smtp_host"]
	smtpPort := channel.Config["smtp_port"]
	username := channel.Config["username"]
	password := channel.Config["password"]
	from := channel.Config["from"]
	to := channel.Config["to"]
	useTLS := channel.Config["use_tls"]

	if smtpHost == "" || smtpPort == "" || to == "" {
		return fmt.Errorf("missing email config: smtp_host, smtp_port or to")
	}

	if from == "" {
		from = username
	}

	// 构建邮件内容
	subject := fmt.Sprintf("EchoFeed: %s", msg.TaskName)
	body := s.formatEmailBody(msg)

	// 构建邮件消息
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	// 根据配置选择TLS或普通连接
	if useTLS == "true" || smtpPort == "465" {
		return s.sendEmailWithTLS(addr, username, password, from, to, message)
	}

	// 使用STARTTLS
	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, smtpHost)
	}

	err := smtp.SendMail(addr, auth, from, []string{to}, []byte(message))
	if err != nil {
		// 某些 SMTP 服务器会在发送完成后立即断开连接，导致客户端认为失败（但邮件已投递）
		if isIgnorableSMTPTerminationError(err) {
			log.Warn().Err(err).Str("channel", channel.Name).Str("to", to).Msg("SMTP termination error ignored after send")
			return nil
		}
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Info().Str("channel", channel.Name).Str("to", to).Msg("Email notification sent")
	return nil
}

func isIgnorableSMTPTerminationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "short response") {
		return true
	}
	if strings.Contains(s, "unexpected eof") {
		return true
	}
	if strings.Contains(s, "connection reset by peer") {
		return true
	}
	if strings.Contains(s, "broken pipe") {
		return true
	}
	return false
}

// sendEmailWithTLS 使用TLS发送邮件
func (s *NotifierService) sendEmailWithTLS(addr, username, password, from, to, message string) error {
	host := strings.Split(addr, ":")[0]

	tlsConfig := &tls.Config{
		ServerName: host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// 认证
	if username != "" && password != "" {
		auth := smtp.PlainAuth("", username, password, host)
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	// 发送邮件
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("smtp_mail: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp_rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp_data: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("smtp_write: %w", err)
	}

	err = w.Close()
	if err != nil {
		// 某些服务器在接受完 DATA 后会提前断开连接，但邮件仍会投递；这里避免误判失败。
		if isIgnorableSMTPTerminationError(err) {
			log.Warn().Err(err).Str("to", to).Msg("SMTP termination error ignored after data close")
			return nil
		}
		return fmt.Errorf("smtp_close: %w", err)
	}

	if err := client.Quit(); err != nil {
		if isIgnorableSMTPTerminationError(err) {
			log.Warn().Err(err).Str("to", to).Msg("SMTP termination error ignored on quit")
			return nil
		}
		return fmt.Errorf("smtp_quit: %w", err)
	}
	return nil
}

// formatEmailBody 格式化邮件正文(HTML格式)
func (s *NotifierService) formatEmailBody(msg *model.NotifyMessage) string {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html><html><head><meta charset="UTF-8"></head><body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 20px;">`)
	sb.WriteString(fmt.Sprintf(`<h2 style="color: #333;">📰 %s</h2>`, msg.TaskName))
	sb.WriteString(`<div style="margin-top: 20px;">`)

	for i, item := range msg.Items {
		if i >= 10 {
			sb.WriteString(fmt.Sprintf(`<p style="color: #666;">... 还有 %d 条内容</p>`, len(msg.Items)-10))
			break
		}

		summary := item.SummaryFull
		if strings.TrimSpace(summary) == "" {
			summary = item.Summary
		}

		published := strings.TrimSpace(item.PublishedAt)
		if published == "" {
			published = "-"
		}

		sb.WriteString(`<div style="margin-bottom: 20px; padding: 15px; border: 1px solid #eee; border-radius: 8px;">`)
		sb.WriteString(fmt.Sprintf(`<h3 style="margin: 0 0 10px 0;"><a href="%s" style="color: #333; text-decoration: none;">%s</a></h3>`, item.Link, item.Title))
		sb.WriteString(fmt.Sprintf(`<p style="color: #666; margin: 0 0 10px 0;">%s</p>`, summary))
		sb.WriteString(fmt.Sprintf(`<p style="color: #999; font-size: 12px; margin: 0;">📍 %s | 🕒 %s | ⭐ 重要度: %d/5</p>`, item.Source, published, item.Importance))
		if strings.TrimSpace(item.Link) != "" {
			sb.WriteString(fmt.Sprintf(`<p style="margin: 10px 0 0 0; font-size: 12px;"><a href="%s" style="color: #2563eb; text-decoration: none;">原文链接</a></p>`, item.Link))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)
	sb.WriteString(`<p style="color: #999; font-size: 12px; margin-top: 30px;">—— Powered by EchoFeed</p>`)
	sb.WriteString(`</body></html>`)

	return sb.String()
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
