package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// TaskEngine 任务执行引擎
type TaskEngine struct {
	cfgMgr     *config.Manager
	feedSvc    *FeedService
	taskSvc    *TaskService
	botSvc     *BotService
	channelSvc *ChannelService
	logSvc     *LogService
	aiSvc      *AIService
	notifier   *NotifierService
}

// NewTaskEngine 创建任务执行引擎
func NewTaskEngine(
	cfgMgr *config.Manager,
	feedSvc *FeedService,
	taskSvc *TaskService,
	botSvc *BotService,
	channelSvc *ChannelService,
	logSvc *LogService,
) *TaskEngine {
	return &TaskEngine{
		cfgMgr:     cfgMgr,
		feedSvc:    feedSvc,
		taskSvc:    taskSvc,
		botSvc:     botSvc,
		channelSvc: channelSvc,
		logSvc:     logSvc,
		aiSvc:      NewAIService(),
		notifier:   NewNotifierService(),
	}
}

// ExecuteTask 执行任务(两阶段AI分析)
func (e *TaskEngine) ExecuteTask(taskID string) error {
	return e.ExecuteTaskForDate(taskID, time.Now().Format("2006-01-02"))
}

// ExecuteTaskForDate 执行任务(指定日期)
func (e *TaskEngine) ExecuteTaskForDate(taskID string, date string) error {
	startTime := time.Now()

	// 获取任务
	task, err := e.taskSvc.Get(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if !task.Enabled {
		log.Info().Str("task", task.Name).Msg("Task is disabled, skipping")
		return nil
	}

	log.Info().Str("task", task.Name).Str("date", date).Msg("Executing task")

	// 创建执行日志
	execLog := &model.ExecutionLog{
		TaskID:    task.ID,
		TaskName:  task.Name,
		StartTime: startTime,
		Status:    model.LogStatusSuccess,
	}

	// 使用defer记录日志
	defer func() {
		execLog.EndTime = time.Now()
		execLog.Duration = execLog.EndTime.Sub(execLog.StartTime).Milliseconds()
		if e.logSvc != nil {
			e.logSvc.Create(execLog)
		}
	}()

	// 获取AI机器人
	bot, err := e.botSvc.Get(task.BotID)
	if err != nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("failed to get bot: %w", err)
	}
	if bot == nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = "bot not found"
		return fmt.Errorf("bot not found: %s", task.BotID)
	}
	if !bot.Enabled {
		execLog.Status = model.LogStatusFailed
		execLog.Error = "bot is disabled"
		return fmt.Errorf("bot is disabled: %s", bot.Name)
	}

	// 创建AI Provider
	provider := e.aiSvc.CreateProviderFromBot(bot)
	if provider == nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = "unsupported provider"
		return fmt.Errorf("unsupported provider: %s", bot.Provider)
	}

	// 获取指定日期的overview
	overview, err := e.feedSvc.LoadOverview(date)
	if err != nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("failed to load overview for %s: %w", date, err)
	}

	if overview == nil || len(overview.Articles) == 0 {
		log.Warn().Str("task", task.Name).Str("date", date).Msg("No RSS content available")
		execLog.Status = model.LogStatusNoMatch
		execLog.Details = "No RSS content available"
		return nil
	}

	execLog.ArticleCount = len(overview.Articles)

	// 构建筛选提示词
	filterPrompt := e.buildFilterPrompt(task)

	// ========== 第一阶段：选择文章 ==========
	log.Info().Str("task", task.Name).Int("articles", len(overview.Articles)).Msg("Phase 1: Selecting articles from overview")

	overviewContent := e.buildOverviewContent(overview)
	phase1Prompt := fmt.Sprintf("%s\n\n%s", model.SystemPromptPhase1, filterPrompt)

	selectionResult, err := provider.AnalyzeForSelection(phase1Prompt, overviewContent)
	if err != nil {
		log.Error().Err(err).Str("task", task.Name).Msg("Phase 1 analysis failed")
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("phase 1 analysis failed: %w", err)
	}

	// 从分析结果中提取选中的文章ID
	selectedIDs := selectionResult.SelectedArticles
	if len(selectedIDs) == 0 {
		log.Info().Str("task", task.Name).Str("reason", selectionResult.Reason).Msg("No articles selected in phase 1")
		execLog.Status = model.LogStatusNoMatch
		execLog.Details = "No articles selected in phase 1: " + selectionResult.Reason
		return nil
	}

	log.Info().Str("task", task.Name).Int("selected", len(selectedIDs)).Msg("Articles selected for detailed analysis")

	// ========== 第二阶段：分析完整内容 ==========
	log.Info().Str("task", task.Name).Msg("Phase 2: Analyzing selected articles")

	// 加载选中文章的完整内容
	fullContent := e.loadSelectedArticlesContent(date, selectedIDs, overview)
	if fullContent == "" {
		log.Warn().Str("task", task.Name).Msg("Failed to load selected articles content")
		execLog.Status = model.LogStatusNoMatch
		execLog.Details = "Failed to load content"
		return nil
	}

	phase2Prompt := fmt.Sprintf("%s\n\n%s", model.SystemPromptPhase2, filterPrompt)

	analysisResult, err := provider.Analyze(phase2Prompt, fullContent)
	if err != nil {
		log.Error().Err(err).Str("task", task.Name).Msg("Phase 2 analysis failed")
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("phase 2 analysis failed: %w", err)
	}

	// 检查是否有匹配内容
	if !analysisResult.HasContent || len(analysisResult.Items) == 0 {
		log.Info().Str("task", task.Name).Msg("No matching content found after detailed analysis")
		execLog.Status = model.LogStatusNoMatch
		execLog.Details = "No matching content"
		return nil
	}

	execLog.MatchCount = len(analysisResult.Items)
	log.Info().Str("task", task.Name).Int("items", len(analysisResult.Items)).Msg("Found matching content")

	// 发送通知
	return e.sendNotifications(task, analysisResult)
}

// buildOverviewContent 构建overview内容供AI阅读
func (e *TaskEngine) buildOverviewContent(overview *model.DailyOverview) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("日期: %s\n", overview.Date))
	sb.WriteString(fmt.Sprintf("文章总数: %d\n\n", overview.TotalCount))
	sb.WriteString("=== 文章列表 ===\n\n")

	for _, article := range overview.Articles {
		sb.WriteString(fmt.Sprintf("【文章ID: %s】\n", article.ID))
		sb.WriteString(fmt.Sprintf("标题: %s\n", article.Title))
		sb.WriteString(fmt.Sprintf("来源: %s\n", article.FeedName))
		sb.WriteString(fmt.Sprintf("链接: %s\n", article.Link))
		sb.WriteString(fmt.Sprintf("时间: %s\n", article.Published.Format("2006-01-02 15:04")))
		sb.WriteString(fmt.Sprintf("摘要: %s\n", article.Summary))
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

// extractSelectedIDs 从AI分析结果中提取选中的文章ID
func (e *TaskEngine) extractSelectedIDs(result *model.AIAnalysisResult) []string {
	// 由于我们使用了通用的Analyze接口，需要从items中提取
	// 或者尝试从原始响应解析selected_articles
	// 这里采用一种兼容方式：如果有items，则使用items中的信息

	var ids []string

	// 尝试从items中提取(如果AI按照phase2格式返回)
	for _, item := range result.Items {
		// 使用Source字段存储ID（需要AI在响应中包含）
		if item.Source != "" {
			ids = append(ids, item.Source)
		}
	}

	// 如果没有从items获取到，说明AI可能返回的是selected_articles格式
	// 这种情况下HasContent为false但我们需要检查原始响应
	// 由于当前接口限制，我们采用另一种策略：让AI将选中的ID放在summary字段
	if len(ids) == 0 && len(result.Items) > 0 {
		for _, item := range result.Items {
			// 尝试将Title或Summary作为ID
			if item.Title != "" {
				ids = append(ids, item.Title)
			}
		}
	}

	return ids
}

// loadSelectedArticlesContent 加载选中文章的完整内容
func (e *TaskEngine) loadSelectedArticlesContent(date string, ids []string, overview *model.DailyOverview) string {
	var sb strings.Builder

	// 创建ID到文章的映射
	articleMap := make(map[string]model.ArticleSummary)
	for _, article := range overview.Articles {
		articleMap[article.ID] = article
	}

	for _, id := range ids {
		article, exists := articleMap[id]
		if !exists {
			log.Warn().Str("id", id).Msg("Article not found in overview")
			continue
		}

		// 读取完整内容
		content, err := e.feedSvc.LoadArticleContent(date, id)
		if err != nil {
			log.Warn().Err(err).Str("id", id).Msg("Failed to load article content")
			continue
		}

		sb.WriteString(fmt.Sprintf("=== %s ===\n", article.Title))
		sb.WriteString(fmt.Sprintf("来源: %s\n", article.FeedName))
		sb.WriteString(fmt.Sprintf("链接: %s\n", article.Link))
		sb.WriteString(fmt.Sprintf("时间: %s\n\n", article.Published.Format("2006-01-02 15:04")))
		sb.WriteString("内容:\n")
		sb.WriteString(content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

// buildFilterPrompt 构建筛选提示词
func (e *TaskEngine) buildFilterPrompt(task *model.Task) string {
	// 如果用户设置了自定义筛选提示词，使用自定义的
	if task.Prompt.Template == "custom" && task.Prompt.FilterPrompt != "" {
		return task.Prompt.FilterPrompt
	}

	// 使用默认筛选提示词模板
	prompt := model.DefaultFilterPrompt

	// 替换关键词变量
	if len(task.Prompt.Keywords) > 0 {
		keywords := strings.Join(task.Prompt.Keywords, ", ")
		prompt = strings.ReplaceAll(prompt, "{keywords}", keywords)
	} else {
		prompt = strings.ReplaceAll(prompt, "{keywords}", "无特定关键词，选择有价值的内容")
	}

	// 替换最低重要度变量
	minImportance := task.Prompt.MinImportance
	if minImportance == 0 {
		minImportance = 3
	}
	prompt = strings.ReplaceAll(prompt, "{min_importance}", fmt.Sprintf("%d", minImportance))

	return prompt
}

// sendNotifications 发送通知
func (e *TaskEngine) sendNotifications(task *model.Task, result *model.AIAnalysisResult) error {
	// 获取通知渠道
	channels, err := e.channelSvc.List()
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	// 构建消息
	msg := model.NotifyMessage{
		TaskID:   task.ID,
		TaskName: task.Name,
		Items:    result.Items,
	}

	// 发送到每个配置的渠道
	for _, channelID := range task.Channels {
		for _, channel := range channels {
			if channel.ID == channelID && channel.Enabled {
				log.Info().Str("task", task.Name).Str("channel", channel.Name).Msg("Sending notification")
				if err := e.notifier.Send(&channel, &msg); err != nil {
					log.Error().Err(err).Str("channel", channel.Name).Msg("Failed to send notification")
				}
			}
		}
	}

	return nil
}

// ---- 简化版本：单阶段分析(作为备选) ----

// ExecuteTaskSimple 执行任务(简化版，单阶段分析)
func (e *TaskEngine) ExecuteTaskSimple(taskID string) error {
	task, err := e.taskSvc.Get(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if !task.Enabled {
		log.Info().Str("task", task.Name).Msg("Task is disabled, skipping")
		return nil
	}

	bot, err := e.botSvc.Get(task.BotID)
	if err != nil {
		return fmt.Errorf("failed to get bot: %w", err)
	}
	if bot == nil || !bot.Enabled {
		return fmt.Errorf("bot not available: %s", task.BotID)
	}

	provider := e.aiSvc.CreateProviderFromBot(bot)
	if provider == nil {
		return fmt.Errorf("unsupported provider: %s", bot.Provider)
	}

	// 获取今日overview
	today := time.Now().Format("2006-01-02")
	overview, err := e.feedSvc.LoadOverview(today)
	if err != nil || overview == nil || len(overview.Articles) == 0 {
		log.Warn().Str("task", task.Name).Msg("No RSS content available")
		return nil
	}

	// 构建内容和提示词
	content := e.buildOverviewContent(overview)
	filterPrompt := e.buildFilterPrompt(task)
	prompt := fmt.Sprintf("%s\n\n%s", model.SystemPromptPhase2, filterPrompt)

	result, err := provider.Analyze(prompt, content)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	if !result.HasContent || len(result.Items) == 0 {
		log.Info().Str("task", task.Name).Msg("No matching content found")
		return nil
	}

	return e.sendNotifications(task, result)
}

// 添加缺失的import
var _ = json.Marshal
