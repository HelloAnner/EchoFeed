package model

// TaskPrompt 任务提示词配置
type TaskPrompt struct {
	Template      string   `toml:"template" json:"template"`                       // default / custom
	FilterPrompt  string   `toml:"filter_prompt,omitempty" json:"filter_prompt"`   // 用户自定义筛选提示词
	Keywords      []string `toml:"keywords,omitempty" json:"keywords"`             // 关键词(default模板用)
	MinImportance int      `toml:"min_importance,omitempty" json:"min_importance"` // 最低重要度(1-5)
}

// Task 任务配置
type Task struct {
	ID          string     `toml:"id" json:"id"`
	Name        string     `toml:"name" json:"name"`
	Description string     `toml:"description" json:"description"`
	Remark      string     `toml:"remark,omitempty" json:"remark"`
	Enabled     bool       `toml:"enabled" json:"enabled"`
	BotID       string     `toml:"bot_id" json:"bot_id"`
	Channels    []string   `toml:"channels" json:"channels"`
	Schedule    string     `toml:"schedule" json:"schedule"` // Cron表达式
	Prompt      TaskPrompt `toml:"prompt" json:"prompt"`
}

// TaskConfig 任务配置文件结构
type TaskConfig struct {
	Tasks []Task `toml:"tasks"`
}

// AIAnalysisResult AI分析结果
type AIAnalysisResult struct {
	HasContent bool             `json:"has_content"`
	Items      []AIAnalysisItem `json:"items"`
}

// AIAnalysisItem AI分析结果项
type AIAnalysisItem struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Importance  int      `json:"importance"`
	Source      string   `json:"source"`
	Link        string   `json:"link"`
	ArticleID   string   `json:"article_id,omitempty"`   // 用于去重/补齐字段(优先来自RSS内容)
	PublishedAt string   `json:"published_at,omitempty"` // 用于推送展示(格式: 2006-01-02 15:04)
	Keywords    []string `json:"keywords,omitempty"`     // 3-5个关键词(可选)
	SummaryFull string   `json:"-"`                      // 邮件等场景使用的完整中文概述（不做100字裁剪）
}

// AISelectionResult AI第一阶段选择结果
type AISelectionResult struct {
	SelectedArticles []string `json:"selected_articles"` // 选中的文章ID列表
	Reason           string   `json:"reason"`            // 选择理由
}

// SystemPromptPhase1 第一阶段系统提示词(选择文章)
const SystemPromptPhase1 = `你是一个专业的新闻筛选助手。你的任务是从RSS文章摘要列表中选择值得深入阅读的文章。

当前你收到的是今日RSS文章的overview(摘要)，包含每篇文章的标题、来源、简短摘要和文章ID。

请根据用户的筛选要求，选择需要进一步阅读完整内容的文章。
为了节省token：最多选择10篇，宁缺毋滥，只选最相关/最有价值的。

输出格式（JSON）：
{
  "selected_articles": ["article_id_1", "article_id_2"],
  "reason": "选择这些文章的理由"
}

如果没有符合条件的文章，返回空数组：
{
  "selected_articles": [],
  "reason": "没有找到符合条件的文章"
}`

// SystemPromptPhase2 第二阶段系统提示词(分析内容)
const SystemPromptPhase2 = `你是一个专业的新闻分析助手。你的任务是分析文章的完整内容，并生成结构化的摘要报告。

请根据用户的筛选要求，分析文章内容并评估其重要性。
重要性评分请保持克制：只有“确实重大/强相关/高影响”的内容才给5分。
为了节省token：最终最多输出5篇最相关的文章，按相关性/重要性排序；其他忽略。

输出格式（JSON）：
{
  "has_content": true/false,
  "items": [
    {
      "title": "文章原始标题(不要改写)",
      "summary": "中文概述(100字以内，包含重点与关键词)",
      "importance": 1-5,
      "keywords": ["关键词1", "关键词2", "关键词3"],
      "source": "来源(原始信息源名称)",
      "link": "原文链接(必须来自内容中出现的链接)",
      "article_id": "文章ID(如果内容中包含)",
      "published_at": "发布时间(如果内容中包含，格式: 2006-01-02 15:04)"
    }
  ]
}

如果文章内容不符合筛选要求，设置has_content为false并返回空items数组。`

// DefaultFilterPrompt 默认筛选提示词模板
const DefaultFilterPrompt = `筛选要求：
- 关键词：{keywords}
- 最低重要性：{min_importance}分(1-5分)

请选择与关键词高度相关且重要性达标的内容。`
