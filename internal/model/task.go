package model

// TaskPrompt 任务提示词配置
type TaskPrompt struct {
	Template       string `toml:"template,omitempty" json:"template"`             // 旧字段：default / custom
	FilterPrompt   string `toml:"filter_prompt,omitempty" json:"filter_prompt"`   // 旧字段：用户自定义筛选提示词
	FilterCriteria string `toml:"filter_criteria,omitempty" json:"filter_criteria"` // 新字段：第一阶段筛选条件
	OutputFormat   string `toml:"output_format,omitempty" json:"output_format"`   // 新字段：第二阶段输出格式（单篇文章）

	Keywords      []string `toml:"keywords,omitempty" json:"keywords"`             // 关键词(可选，用于摘要补齐关键词)
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
const SystemPromptPhase1 = `你是一个专业的 RSS 信息筛选助手，目标是“省 token、宁缺毋滥、只挑值得进一步阅读的文章”。

你收到的是今日 RSS 文章 overview（可理解为 overview.json 的内容摘要），每篇包含：文章ID、标题、来源、链接、发布时间、摘要。

你的任务：
1) 严格根据“筛选条件”判断哪些文章值得进入第二阶段（将会读取本地 txt 正文）
2) 只返回需要进入第二阶段的文章ID
3) 为了节省 token：最多选择 10 篇，宁缺毋滥；如果不确定价值，就不要选

判断建议（保持克制）：
- 优先选择：强相关、信息密度高、可落地、影响面大、包含明确的新功能/新版本/关键结论/可复现细节
- 过滤：纯搬运、营销、重复信息、与筛选条件弱相关、只有标题党没有有效信息

输出格式（必须是 JSON，且只能输出 JSON）：
{
  "selected_articles": ["article_id_1", "article_id_2"],
  "reason": "为什么选择这些文章（中文，简洁，1-3句）"
}

如果没有符合条件的文章，返回空数组：
{
  "selected_articles": [],
  "reason": "没有找到符合条件的文章（中文，简洁）"
}`

// SystemPromptPhase2 第二阶段系统提示词(分析内容)
const SystemPromptPhase2 = `你是一个专业的文章分析与摘要助手。你会收到多篇文章的“本地正文 txt”（已按 newsreader 风格拼接）。

你的任务：
1) 逐篇阅读正文，重新核对是否满足“筛选条件”（第二阶段仍可剔除不匹配的）
2) 对匹配的文章逐篇给出：中文概述、重要性评分、关键词等
3) 为了节省 token：最终最多输出 5 篇最相关的文章，按相关性/重要性排序；其余忽略

重要性评分（1-5）务必克制：
- 5：重大进展/强相关/高影响，且信息明确
- 4：明显重要，有实质细节或明确改动
- 3：有价值但影响较小或细节一般
- 1-2：相关性弱或价值有限（一般不建议输出）

中文概述要求：
- 不要出现“中文概述（100字内）”这类字样
- 不要用“...”省略号；需要把关键信息讲完整
- 建议 150-300 字，包含关键结论与关键点（邮件会展示完整概述；其他渠道会自动裁剪到更短）

输出格式（必须是 JSON，且只能输出 JSON）：
{
  "has_content": true/false,
  "items": [
    {
      "title": "文章原始标题(不要改写)",
      "summary": "中文概述（完整版本，用于邮件等展示）",
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

// DefaultTaskOutputFormat 默认“输出格式”（供 UI/迁移使用）
const DefaultTaskOutputFormat = `请对每篇符合条件的文章输出一个 JSON item，字段要求：
- title：原始标题
- summary：中文概述（完整版本，不要省略号）
- importance：1-5（克制评分）
- keywords：3-5个关键词
- source：信息来源
- link：原文链接
- article_id：文章ID（如果已知）
- published_at：发布时间（上海时区，格式 2006-01-02 15:04）`
