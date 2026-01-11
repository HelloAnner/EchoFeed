// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

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
	taskState  *TaskStateService
	sentSvc    *SentService
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
		taskState:  NewTaskStateService(cfgMgr),
		sentSvc:    NewSentService(cfgMgr),
		aiSvc:      NewAIService(),
		notifier:   NewNotifierService(),
	}
}

// ExecuteTask 执行任务(两阶段AI分析)
func (e *TaskEngine) ExecuteTask(taskID string) error {
	return e.ExecuteTaskForDate(taskID, time.Now().In(time.Local).Format("2006-01-02"))
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

	var overviewHash string
	var detailParts []string

	if err := e.taskState.MarkTaskRun(task.ID, startTime); err != nil {
		log.Warn().Err(err).Str("task", task.Name).Msg("Failed to persist task last run time")
	}

	// 使用defer记录日志
	defer func() {
		execLog.EndTime = time.Now()
		execLog.Duration = execLog.EndTime.Sub(execLog.StartTime).Milliseconds()
		if e.logSvc != nil {
			if execLog.Details == "" && len(detailParts) > 0 {
				execLog.Details = strings.Join(detailParts, "; ")
			}
			e.logSvc.Create(execLog)
		}
		if execLog.Status != model.LogStatusFailed && overviewHash != "" {
			if err := e.taskState.MarkOverviewAnalyzed(task.ID, date, overviewHash); err != nil {
				log.Warn().Err(err).Str("task", task.Name).Msg("Failed to persist analyzed overview state")
			}
		}
	}()

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
		execLog.Details = "无 RSS 内容"
		return nil
	}

	sentIDs, err := e.sentSvc.GetSentIDs(date, task.ID)
	if err != nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("failed to load sended.json: %w", err)
	}

	filteredOverview := e.filterOverviewBySent(overview, sentIDs)
	detailParts = append(detailParts, fmt.Sprintf("概览文章数=%d", len(overview.Articles)))
	detailParts = append(detailParts, fmt.Sprintf("排除已发送=%d", len(overview.Articles)-len(filteredOverview.Articles)))
	if len(filteredOverview.Articles) == 0 {
		log.Info().Str("task", task.Name).Str("date", date).Msg("All articles already sent, skipping task execution")
		execLog.Status = model.LogStatusNoMatch
		detailParts = append(detailParts, "原因=今日文章已全部发送")
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}

	var articleIDs []string
	for _, a := range filteredOverview.Articles {
		articleIDs = append(articleIDs, a.ID)
	}
	overviewHash = computeOverviewHash(articleIDs)
	if overviewHash != "" && e.taskState.IsOverviewAnalyzed(task.ID, date, overviewHash) {
		log.Info().Str("task", task.Name).Str("date", date).Msg("No new articles since last analysis, skipping task execution")
		execLog.Status = model.LogStatusNoMatch
		detailParts = append(detailParts, "原因=与上次相比无新增文章")
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}

	execLog.ArticleCount = len(filteredOverview.Articles)
	detailParts = append(detailParts, fmt.Sprintf("分析池=%d", execLog.ArticleCount))

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

	filterCriteria := e.buildFilterCriteria(task)
	outputFormat := e.buildOutputFormat(task)

	// ========== 第一阶段：选择文章 ==========
	log.Info().Str("task", task.Name).Int("articles", len(filteredOverview.Articles)).Msg("Phase 1: Selecting articles from overview")

	overviewContent := e.buildOverviewContent(filteredOverview)
	phase1Prompt := e.buildPhase1Prompt(task, filterCriteria)

	selectionResult, err := provider.AnalyzeForSelection(phase1Prompt, overviewContent)
	if err != nil {
		log.Error().Err(err).Str("task", task.Name).Msg("Phase 1 analysis failed")
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return fmt.Errorf("phase 1 analysis failed: %w", err)
	}

	// 从分析结果中提取选中的文章ID
	selectedIDs := selectionResult.SelectedArticles
	if selectionResult.Reason != "" {
		detailParts = append(detailParts, "第一阶段原因="+truncateForLog(selectionResult.Reason, 200))
	}
	selectedIDs = e.filterSelectedIDsBySent(selectedIDs, sentIDs)
	if len(selectedIDs) == 0 {
		log.Info().Str("task", task.Name).Str("reason", selectionResult.Reason).Msg("No articles selected in phase 1")
		execLog.Status = model.LogStatusNoMatch
		detailParts = append(detailParts, "原因=第一阶段未选中任何文章："+truncateForLog(selectionResult.Reason, 200))
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}
	detailParts = append(detailParts, fmt.Sprintf("第一阶段选中=%d", len(selectedIDs)))

	log.Info().Str("task", task.Name).Int("selected", len(selectedIDs)).Msg("Articles selected for detailed analysis")

	// ========== 第二阶段：分析完整内容 ==========
	log.Info().Str("task", task.Name).Msg("Phase 2: Analyzing selected articles")

	// 加载选中文章的完整内容
	fullContent := e.loadSelectedArticlesContent(date, selectedIDs, filteredOverview)
	if fullContent == "" {
		log.Warn().Str("task", task.Name).Msg("Failed to load selected articles content")
		execLog.Status = model.LogStatusNoMatch
		detailParts = append(detailParts, "原因=加载文章内容失败")
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}

	phase2Prompt := e.buildPhase2Prompt(task, filterCriteria, outputFormat)

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
		detailParts = append(detailParts, "原因=第二阶段无匹配内容")
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}

	execLog.MatchCount = len(analysisResult.Items)
	log.Info().Str("task", task.Name).Int("items", len(analysisResult.Items)).Msg("Found matching content")
	detailParts = append(detailParts, fmt.Sprintf("第二阶段匹配=%d", execLog.MatchCount))

	// 去重 + 补齐通知字段（标题/来源/发布时间/关键词/摘要长度）
	prepared := e.prepareNotifyItems(task, analysisResult.Items, filteredOverview, sentIDs)
	if len(prepared) == 0 {
		log.Info().Str("task", task.Name).Msg("All matched content already processed or below importance")
		execLog.Status = model.LogStatusNoMatch
		detailParts = append(detailParts, "原因=匹配内容均已发送或重要性不足")
		execLog.Details = strings.Join(detailParts, "; ")
		return nil
	}
	analysisResult.Items = prepared
	execLog.MatchCount = len(analysisResult.Items)
	detailParts = append(detailParts, fmt.Sprintf("准备发送=%d", execLog.MatchCount))

	// 发送通知
	if err := e.sendNotifications(date, task, analysisResult); err != nil {
		execLog.Status = model.LogStatusFailed
		execLog.Error = err.Error()
		return err
	}
	execLog.Status = model.LogStatusSuccess
	detailParts = append(detailParts, fmt.Sprintf("已发送=%d", execLog.MatchCount))
	return nil
}

func truncateForLog(s string, maxRunes int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	s = strings.Join(strings.Fields(s), " ")
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	if maxRunes == 1 {
		return "…"
	}
	return string(r[:maxRunes-1]) + "…"
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
		sb.WriteString(fmt.Sprintf("文章ID: %s\n", article.ID))
		sb.WriteString(fmt.Sprintf("来源: %s\n", article.FeedName))
		sb.WriteString(fmt.Sprintf("链接: %s\n", article.Link))
		sb.WriteString(fmt.Sprintf("时间: %s\n\n", article.Published.Format("2006-01-02 15:04")))
		sb.WriteString("内容:\n")
		sb.WriteString(content)
		sb.WriteString("\n\n---\n\n")
	}

	return sb.String()
}

func (e *TaskEngine) prepareNotifyItems(task *model.Task, items []model.AIAnalysisItem, overview *model.DailyOverview, sentIDs map[string]bool) []model.AIAnalysisItem {
	articleMap := make(map[string]model.ArticleSummary, len(overview.Articles))
	for _, a := range overview.Articles {
		articleMap[a.ID] = a
	}

	minImportance := task.Prompt.MinImportance
	if minImportance == 0 {
		minImportance = 3
	}

	seen := make(map[string]bool)
	var prepared []model.AIAnalysisItem
	for _, item := range items {
		item = e.enrichNotifyItem(task, item, articleMap)

		if item.ArticleID != "" && sentIDs[item.ArticleID] {
			continue
		}

		if item.Importance < 1 {
			item.Importance = 1
		}
		if item.Importance > 5 {
			item.Importance = 5
		}
		if item.Importance < minImportance {
			continue
		}

		if item.ArticleID != "" {
			if seen[item.ArticleID] {
				continue
			}
			seen[item.ArticleID] = true
		}

		articleID := item.ArticleID
		if articleID == "" && item.Link != "" {
			articleID = generateArticleID(item.Link)
			item.ArticleID = articleID
		}

		if articleID != "" && sentIDs[articleID] {
			continue
		}

		prepared = append(prepared, item)
	}

	return prepared
}

func (e *TaskEngine) enrichNotifyItem(task *model.Task, item model.AIAnalysisItem, articleMap map[string]model.ArticleSummary) model.AIAnalysisItem {
	articleID := strings.TrimSpace(item.ArticleID)
	if articleID == "" && item.Link != "" {
		articleID = generateArticleID(item.Link)
	}
	if articleID != "" {
		item.ArticleID = articleID
	}

	if articleID != "" {
		if a, ok := articleMap[articleID]; ok {
			item.Title = a.Title
			item.Source = a.FeedName
			item.Link = a.Link
			item.PublishedAt = a.Published.Format("2006-01-02 15:04")
		}
	}

	item.SummaryFull = e.normalizeCnSummaryFull(item.Summary, item.Keywords, task.Prompt.Keywords)
	item.Summary = e.normalizeCnSummary(item.Summary, item.Keywords, task.Prompt.Keywords, 100)
	return item
}

func (e *TaskEngine) normalizeCnSummaryFull(summary string, keywords []string, fallbackKeywords []string) string {
	s := sanitizeSummaryText(summary)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")

	kws := keywords
	if len(kws) == 0 {
		kws = fallbackKeywords
	}
	kws = uniqueNonEmpty(kws)

	if len(kws) > 0 && !strings.Contains(s, "关键词") {
		s = s + " 关键词：" + strings.Join(kws[:minInt(5, len(kws))], "、")
	}
	return s
}

func (e *TaskEngine) normalizeCnSummary(summary string, keywords []string, fallbackKeywords []string, maxRunes int) string {
	s := sanitizeSummaryText(summary)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")

	kws := keywords
	if len(kws) == 0 {
		kws = fallbackKeywords
	}
	kws = uniqueNonEmpty(kws)

	// 先保证主体不超过限制（尽量在分隔符处截断，不使用省略号）
	s = cutAtSeparator(s, maxRunes)

	// 再尽量把关键词塞进100字内：summary + " 关键词：a、b、c"
	if len(kws) > 0 && maxRunes > 0 {
		for keep := minInt(5, len(kws)); keep >= 1; keep-- {
			tail := " 关键词：" + strings.Join(kws[:keep], "、")
			if runeLen(s)+runeLen(tail) <= maxRunes {
				s = s + tail
				return s
			}
		}
	}

	return s
}

func uniqueNonEmpty(in []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func runeLen(s string) int {
	return len([]rune(s))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cutAtSeparator(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}

	seps := map[rune]bool{
		'。': true, '！': true, '？': true, '；': true, ';': true,
		'，': true, ',': true, '、': true, ':': true, '：': true,
	}

	cut := -1
	for i := maxRunes - 1; i >= 0; i-- {
		if seps[r[i]] {
			cut = i + 1
			break
		}
	}
	if cut == -1 {
		cut = maxRunes
	}

	out := strings.TrimSpace(string(r[:cut]))
	out = strings.TrimRight(out, "，,、:：")
	return out
}

func (e *TaskEngine) buildFilterCriteria(task *model.Task) string {
	if task == nil {
		return ""
	}
	if strings.TrimSpace(task.Prompt.FilterCriteria) != "" {
		return task.Prompt.FilterCriteria
	}
	// 兼容旧字段
	if strings.TrimSpace(task.Prompt.FilterPrompt) != "" {
		return task.Prompt.FilterPrompt
	}
	if len(task.Prompt.Keywords) > 0 {
		return "关注关键词：" + strings.Join(task.Prompt.Keywords, "、")
	}
	return "选择最有价值、信息密度高的文章（宁缺毋滥）。"
}

func (e *TaskEngine) buildOutputFormat(task *model.Task) string {
	if task == nil {
		return model.DefaultTaskOutputFormat
	}
	if strings.TrimSpace(task.Prompt.OutputFormat) != "" {
		return task.Prompt.OutputFormat
	}
	return model.DefaultTaskOutputFormat
}

func (e *TaskEngine) buildPhase1Prompt(task *model.Task, criteria string) string {
	minImportance := 3
	if task != nil && task.Prompt.MinImportance > 0 {
		minImportance = task.Prompt.MinImportance
	}

	desc := ""
	if task != nil && strings.TrimSpace(task.Description) != "" {
		desc = "\n\n=== 任务描述 ===\n" + strings.TrimSpace(task.Description) + "\n"
	}
	return fmt.Sprintf("%s%s\n\n=== 筛选条件 ===\n%s\n\n最低重要度：%d/5\n", model.SystemPromptPhase1, desc, strings.TrimSpace(criteria), minImportance)
}

func (e *TaskEngine) buildPhase2Prompt(task *model.Task, criteria, outputFormat string) string {
	minImportance := 3
	if task != nil && task.Prompt.MinImportance > 0 {
		minImportance = task.Prompt.MinImportance
	}

	desc := ""
	if task != nil && strings.TrimSpace(task.Description) != "" {
		desc = "\n\n=== 任务描述 ===\n" + strings.TrimSpace(task.Description) + "\n"
	}
	return fmt.Sprintf("%s%s\n\n=== 筛选条件 ===\n%s\n\n最低重要度：%d/5\n\n=== 输出格式（单篇文章）===\n%s\n\n请按系统要求输出 JSON（has_content + items），items 中每个对象按上面输出格式填充。\n",
		model.SystemPromptPhase2,
		desc,
		strings.TrimSpace(criteria),
		minImportance,
		strings.TrimSpace(outputFormat),
	)
}

// sendNotifications 发送通知
func (e *TaskEngine) sendNotifications(date string, task *model.Task, result *model.AIAnalysisResult) error {
	// 获取通知渠道
	channels, err := e.channelSvc.List()
	if err != nil {
		return fmt.Errorf("failed to get channels: %w", err)
	}

	channelMap := make(map[string]model.Channel, len(channels))
	for _, ch := range channels {
		channelMap[ch.ID] = ch
	}

	// 构建消息
	msg := model.NotifyMessage{
		TaskID:   task.ID,
		TaskName: task.Name,
		Items:    result.Items,
	}

	// 发送到每个配置的渠道
	var failed []string
	sentMap := make(map[string]model.AIAnalysisItem)
	for _, channelID := range task.Channels {
		channel, ok := channelMap[channelID]
		if !ok {
			failed = append(failed, fmt.Sprintf("%s: channel not found", channelID))
			continue
		}
		if !channel.Enabled {
			failed = append(failed, fmt.Sprintf("%s(%s): channel disabled", channel.Name, channel.ID))
			continue
		}

		log.Info().Str("task", task.Name).Str("channel", channel.Name).Msg("Sending notification")
		if err := e.notifier.Send(&channel, &msg); err != nil {
			log.Error().Err(err).Str("channel", channel.Name).Msg("Failed to send notification")
			failed = append(failed, fmt.Sprintf("%s(%s): %s", channel.Name, channel.ID, err.Error()))

			if pe, ok := err.(*PartialSendError); ok && len(pe.SentArticleIDs) > 0 {
				for _, id := range pe.SentArticleIDs {
					if it, ok := findItemByArticleID(msg.Items, id); ok {
						sentMap[id] = it
					}
				}
			}
			continue
		}

		for _, it := range msg.Items {
			if it.ArticleID != "" {
				sentMap[it.ArticleID] = it
			}
		}
	}

	if len(sentMap) > 0 {
		var sentItems []model.AIAnalysisItem
		for _, it := range sentMap {
			sentItems = append(sentItems, it)
		}
		if err := e.sentSvc.MarkSent(date, task, sentItems); err != nil {
			log.Warn().Err(err).Str("task", task.Name).Str("date", date).Msg("Failed to update sended.json")
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("通知发送失败：%s", strings.Join(failed, "；"))
	}
	return nil
}

func findItemByArticleID(items []model.AIAnalysisItem, id string) (model.AIAnalysisItem, bool) {
	for _, it := range items {
		if it.ArticleID == id {
			return it, true
		}
	}
	return model.AIAnalysisItem{}, false
}

func (e *TaskEngine) filterOverviewBySent(overview *model.DailyOverview, sentIDs map[string]bool) *model.DailyOverview {
	if overview == nil {
		return overview
	}
	if len(sentIDs) == 0 {
		return overview
	}
	var kept []model.ArticleSummary
	for _, a := range overview.Articles {
		if a.ID != "" && sentIDs[a.ID] {
			continue
		}
		kept = append(kept, a)
	}
	cp := *overview
	cp.Articles = kept
	cp.TotalCount = len(kept)
	return &cp
}

func (e *TaskEngine) filterSelectedIDsBySent(ids []string, sentIDs map[string]bool) []string {
	if len(ids) == 0 || len(sentIDs) == 0 {
		return ids
	}
	var kept []string
	for _, id := range ids {
		if id != "" && sentIDs[id] {
			continue
		}
		kept = append(kept, id)
	}
	return kept
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
	today := time.Now().In(time.Local).Format("2006-01-02")
	overview, err := e.feedSvc.LoadOverview(today)
	if err != nil || overview == nil || len(overview.Articles) == 0 {
		log.Warn().Str("task", task.Name).Msg("No RSS content available")
		return nil
	}

	// 构建内容和提示词
	content := e.buildOverviewContent(overview)
	filterCriteria := e.buildFilterCriteria(task)
	outputFormat := e.buildOutputFormat(task)
	prompt := e.buildPhase2Prompt(task, filterCriteria, outputFormat)

	result, err := provider.Analyze(prompt, content)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	if !result.HasContent || len(result.Items) == 0 {
		log.Info().Str("task", task.Name).Msg("No matching content found")
		return nil
	}

	return e.sendNotifications(today, task, result)
}

// 添加缺失的import
var _ = json.Marshal
