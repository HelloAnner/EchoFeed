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

// AIProvider AI服务提供者接口
type AIProvider interface {
	Analyze(prompt string, content string) (*model.AIAnalysisResult, error)
}

// AIService AI服务
type AIService struct {
	providers map[string]AIProvider
}

// NewAIService 创建AI服务
func NewAIService() *AIService {
	return &AIService{
		providers: make(map[string]AIProvider),
	}
}

// RegisterProvider 注册AI提供者
func (s *AIService) RegisterProvider(name string, provider AIProvider) {
	s.providers[name] = provider
}

// GetProvider 获取AI提供者
func (s *AIService) GetProvider(name string) AIProvider {
	return s.providers[name]
}

// CreateProviderFromBot 根据Bot配置创建Provider
func (s *AIService) CreateProviderFromBot(bot *model.Bot) AIProvider {
	switch bot.Provider {
	case model.ProviderOpenAI:
		return NewOpenAIProvider(
			bot.Config["api_key"],
			bot.Config["model"],
			bot.Config["base_url"],
		)
	case model.ProviderClaude:
		return NewClaudeProvider(
			bot.Config["api_key"],
			bot.Config["model"],
			bot.Config["base_url"],
		)
	case model.ProviderOllama:
		return NewOllamaProvider(
			bot.Config["base_url"],
			bot.Config["model"],
		)
	default:
		return nil
	}
}

// OpenAIProvider OpenAI提供者
type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
}

// NewOpenAIProvider 创建OpenAI提供者
func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// Analyze 分析内容
func (p *OpenAIProvider) Analyze(prompt string, content string) (*model.AIAnalysisResult, error) {
	fullPrompt := fmt.Sprintf("%s\n\n以下是RSS内容:\n%s", prompt, content)

	reqBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{"role": "user", "content": fullPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// 解析AI返回的JSON
	return parseAIResponse(result.Choices[0].Message.Content)
}

// ClaudeProvider Claude提供者
type ClaudeProvider struct {
	apiKey  string
	model   string
	baseURL string
}

// NewClaudeProvider 创建Claude提供者
func NewClaudeProvider(apiKey, model, baseURL string) *ClaudeProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &ClaudeProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

// Analyze 分析内容
func (p *ClaudeProvider) Analyze(prompt string, content string) (*model.AIAnalysisResult, error) {
	fullPrompt := fmt.Sprintf("%s\n\n以下是RSS内容:\n%s", prompt, content)

	reqBody := map[string]interface{}{
		"model":      p.model,
		"max_tokens": 2000,
		"messages": []map[string]string{
			{"role": "user", "content": fullPrompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API error: %s", string(body))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("no response from Claude")
	}

	return parseAIResponse(result.Content[0].Text)
}

// OllamaProvider Ollama提供者
type OllamaProvider struct {
	baseURL string
	model   string
}

// NewOllamaProvider 创建Ollama提供者
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama3"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
	}
}

// Analyze 分析内容
func (p *OllamaProvider) Analyze(prompt string, content string) (*model.AIAnalysisResult, error) {
	fullPrompt := fmt.Sprintf("%s\n\n以下是RSS内容:\n%s", prompt, content)

	reqBody := map[string]interface{}{
		"model":  p.model,
		"prompt": fullPrompt,
		"stream": false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API error: %s", string(body))
	}

	var result struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return parseAIResponse(result.Response)
}

// parseAIResponse 解析AI返回的JSON响应
func parseAIResponse(content string) (*model.AIAnalysisResult, error) {
	// 尝试从内容中提取JSON
	content = strings.TrimSpace(content)

	// 查找JSON开始和结束位置
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 || end <= start {
		log.Warn().Str("content", content).Msg("No valid JSON found in AI response")
		return &model.AIAnalysisResult{HasContent: false}, nil
	}

	jsonStr := content[start : end+1]

	var result model.AIAnalysisResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Warn().Err(err).Str("json", jsonStr).Msg("Failed to parse AI response JSON")
		return &model.AIAnalysisResult{HasContent: false}, nil
	}

	return &result, nil
}

// parseSelectionResponse 解析AI选择阶段的响应
func parseSelectionResponse(content string) (*model.AISelectionResult, error) {
	content = strings.TrimSpace(content)

	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")

	if start == -1 || end == -1 || end <= start {
		log.Warn().Str("content", content).Msg("No valid JSON found in selection response")
		return &model.AISelectionResult{SelectedArticles: []string{}}, nil
	}

	jsonStr := content[start : end+1]

	var result model.AISelectionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		log.Warn().Err(err).Str("json", jsonStr).Msg("Failed to parse selection response JSON")
		return &model.AISelectionResult{SelectedArticles: []string{}}, nil
	}

	return &result, nil
}

// AnalyzeRaw 原始分析，返回字符串内容
func AnalyzeRaw(provider AIProvider, systemPrompt, filterPrompt, content string) (string, error) {
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, filterPrompt)
	result, err := provider.Analyze(fullPrompt, content)
	if err != nil {
		return "", err
	}
	// 这里返回原始内容，需要调整Provider接口
	// 暂时通过结果判断
	return "", fmt.Errorf("use AnalyzeForSelection or Analyze instead")
}
