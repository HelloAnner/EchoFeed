# EchoFeed - 智能RSS订阅与推送系统

## 产品需求文档 (PRD)

### 1. 项目概述

#### 1.1 项目名称
**EchoFeed** (回声订阅)

命名理由：
- Echo（回声）：信息从世界各地回响到你身边
- Feed（订阅源）：RSS订阅的核心概念
- 寓意：让重要信息像回声一样及时传递给你

#### 1.2 项目定位
一个自托管的智能RSS聚合与推送系统，通过AI自动分析全量RSS内容，按任务需求筛选并推送给用户。

#### 1.3 核心价值
- **信息聚合**：统一管理多个RSS订阅源
- **任务驱动**：每个任务基于全量RSS内容进行AI分析筛选
- **智能推送**：AI判断是否有匹配内容，无则不推送
- **多机器人**：支持配置多个AI机器人，不同任务使用不同机器人
- **自托管**：数据完全掌控，TOML文件配置，易于备份迁移

---

### 2. 系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          EchoFeed System                                │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                         Web UI (配置界面)                         │  │
│  │  ┌─────────────┬─────────────┬─────────────┬─────────────┐       │  │
│  │  │  📋 任务    │  📡 RSS     │  🔔 通知    │  🤖 AI      │       │  │
│  │  │  列表      │  订阅管理    │  渠道配置   │  机器人     │       │  │
│  │  └─────────────┴─────────────┴─────────────┴─────────────┘       │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                    │                                    │
│                          ┌─────────▼─────────┐                         │
│                          │     REST API      │                         │
│                          └─────────┬─────────┘                         │
│                                    │                                    │
│  ┌─────────────────────────────────▼─────────────────────────────────┐ │
│  │                        Core Service                                │ │
│  │                                                                    │ │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐       │ │
│  │  │  RSS Fetcher   │  │  Task Engine   │  │   Notifier     │       │ │
│  │  │  (定时拉取)    │  │  (任务调度)    │  │  (多渠道推送)  │       │ │
│  │  └───────┬────────┘  └───────┬────────┘  └───────┬────────┘       │ │
│  │          │                   │                   │                │ │
│  │          │           ┌───────▼────────┐          │                │ │
│  │          │           │  AI Analyzer   │          │                │ │
│  │          │           │  (多机器人)    │          │                │ │
│  │          │           └────────────────┘          │                │ │
│  │          │                                       │                │ │
│  └──────────┼───────────────────────────────────────┼────────────────┘ │
│             │                                       │                  │
│  ┌──────────▼───────────────────────────────────────▼────────────────┐ │
│  │                     Storage Layer (TOML + Files)                  │ │
│  │  data/                                                            │ │
│  │  ├── rss.toml          # RSS订阅配置                              │ │
│  │  ├── tasks.toml        # 任务配置                                 │ │
│  │  ├── channels.toml     # 通知渠道配置                             │ │
│  │  ├── bots.toml         # AI机器人配置                             │ │
│  │  └── rss/              # RSS拉取的内容缓存                        │ │
│  │      ├── feed_xxx.json                                            │ │
│  │      └── ...                                                      │ │
│  └───────────────────────────────────────────────────────────────────┘ │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

### 3. 核心概念

#### 3.1 设计理念

**任务驱动模式**：
- 传统RSS阅读器：订阅 → 文章列表 → 阅读
- EchoFeed模式：订阅 → AI分析全量内容 → 按任务需求筛选推送

**任务示例**：
| 任务名称 | 描述 | AI提示词核心 |
|---------|------|-------------|
| Claude Code更新 | 推送Claude Code最新功能更新 | 筛选与Claude Code、Anthropic相关的更新公告 |
| 国内经济消息 | 推送国内经济相关重要新闻 | 筛选中国经济、财经、政策相关的重要消息 |
| 技术热点 | 推送技术圈热门话题 | 筛选GitHub trending、技术博客热门文章 |

#### 3.2 数据流

```
┌─────────┐    定时拉取    ┌─────────┐    缓存    ┌─────────────┐
│ RSS源   │ ────────────► │ Fetcher │ ────────► │ data/rss/   │
└─────────┘               └─────────┘           └──────┬──────┘
                                                       │
                                                       ▼
┌─────────┐    推送结果    ┌─────────┐   AI分析   ┌─────────────┐
│ 用户    │ ◄──────────── │ Notifier│ ◄──────── │ Task Engine │
└─────────┘               └─────────┘           └─────────────┘
                                                       │
                                                 选择AI机器人
                                                       │
                                                       ▼
                                               ┌─────────────┐
                                               │ AI Bot Pool │
                                               └─────────────┘
```

---

### 4. 功能模块设计

#### 4.1 RSS订阅管理

**功能清单**：
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 添加订阅 | 支持RSS 2.0/Atom格式，带备注 | P0 |
| 订阅列表 | 展示所有订阅源及状态 | P0 |
| 编辑/删除 | 修改订阅信息或移除 | P0 |
| 手动刷新 | 立即拉取指定订阅 | P1 |
| 健康检测 | 显示订阅源可用性 | P1 |

**配置文件**: `data/rss.toml`
```toml
# RSS订阅配置

[settings]
default_interval = 30      # 默认拉取间隔(分钟)
timeout = 30               # 请求超时(秒)
user_agent = "EchoFeed/1.0"
max_concurrent = 5         # 最大并发拉取数

[[feeds]]
id = "anthropic-news"
name = "Anthropic News"
url = "https://www.anthropic.com/feed.xml"
remark = "Anthropic官方博客，关注Claude更新"
interval = 60              # 可选，覆盖默认间隔
enabled = true

[[feeds]]
id = "hackernews"
name = "Hacker News"
url = "https://hnrss.org/frontpage"
remark = "技术热点新闻"
enabled = true

[[feeds]]
id = "36kr"
name = "36氪"
url = "https://36kr.com/feed"
remark = "国内科技财经新闻"
enabled = true
```

#### 4.2 任务管理

**核心概念**：每个任务 = AI提示词 + 通知渠道 + 调度周期

**功能清单**：
| 功能 | 描述 | 优先级 |
|------|------|--------|
| 创建任务 | 配置名称、描述、提示词 | P0 |
| 选择AI机器人 | 指定使用哪个AI机器人 | P0 |
| 配置通知渠道 | 支持多个通知渠道 | P0 |
| 调度周期 | 配置扫描频率 | P0 |
| 手动执行 | 立即执行任务 | P1 |
| 执行历史 | 查看任务执行记录 | P1 |

**配置文件**: `data/tasks.toml`
```toml
# 任务配置

[[tasks]]
id = "claude-code-updates"
name = "Claude Code 更新推送"
description = "监控并推送Claude Code的最新功能更新和发布信息"
enabled = true
bot_id = "gpt4"            # 使用的AI机器人ID
channels = ["telegram-main", "email-work"]  # 通知渠道ID列表
schedule = "0 */2 * * *"   # Cron表达式：每2小时执行一次

[tasks.prompt]
template = "custom"        # default / custom
content = """
你是一个技术新闻编辑，请从以下RSS内容中筛选与Claude Code、Anthropic、AI编程助手相关的更新。

筛选标准：
1. Claude Code新功能发布
2. Anthropic官方公告
3. Claude API更新
4. 相关技术教程和最佳实践

如果没有符合条件的内容，直接返回空结果。

输出格式（JSON）：
{
  "has_content": true/false,
  "items": [
    {
      "title": "标题",
      "summary": "50字以内摘要",
      "importance": 1-5,
      "source": "来源名称",
      "link": "原文链接"
    }
  ]
}
"""

[[tasks]]
id = "china-economy"
name = "国内经济消息"
description = "推送国内经济、财经、政策相关的重要新闻"
enabled = true
bot_id = "claude"
channels = ["telegram-main"]
schedule = "0 8,12,18 * * *"  # 每天8点、12点、18点执行

[tasks.prompt]
template = "default"       # 使用默认提示词模板
keywords = ["经济", "财经", "央行", "政策", "GDP", "利率"]
min_importance = 3
```

**默认提示词模板**：
```toml
# data/prompt_templates.toml

[default]
content = """
你是一个专业的新闻筛选助手，请从以下RSS内容中筛选符合条件的新闻。

筛选关键词：{keywords}
最低重要性：{min_importance}

要求：
1. 只返回与关键词高度相关的内容
2. 评估每条内容的重要性（1-5分）
3. 生成简洁摘要（50字以内）
4. 如果没有符合条件的内容，返回空结果

输出格式（JSON）：
{
  "has_content": true/false,
  "items": [
    {
      "title": "标题",
      "summary": "摘要",
      "importance": 4,
      "source": "来源",
      "link": "链接"
    }
  ]
}
"""
```

#### 4.3 通知渠道配置

**支持渠道**：
| 渠道类型 | 描述 | 配置项 |
|---------|------|--------|
| telegram | Telegram机器人 | bot_token, chat_id |
| email | 邮件通知 | smtp_host, smtp_port, from, to |
| webhook | 自定义HTTP回调 | url, method, headers |
| bark | iOS Bark推送 | server_url, device_key |
| wecom | 企业微信机器人 | webhook_url |

**配置文件**: `data/channels.toml`
```toml
# 通知渠道配置

[[channels]]
id = "telegram-main"
name = "Telegram主群"
type = "telegram"
enabled = true
remark = "日常消息推送"

[channels.config]
bot_token = "YOUR_BOT_TOKEN"
chat_id = "YOUR_CHAT_ID"
parse_mode = "Markdown"

[[channels]]
id = "email-work"
name = "工作邮箱"
type = "email"
enabled = true
remark = "重要消息邮件通知"

[channels.config]
smtp_host = "smtp.gmail.com"
smtp_port = 587
smtp_user = "your@gmail.com"
smtp_pass = "YOUR_APP_PASSWORD"
from = "EchoFeed <your@gmail.com>"
to = ["work@example.com"]

[[channels]]
id = "webhook-custom"
name = "自定义Webhook"
type = "webhook"
enabled = true

[channels.config]
url = "https://your-server.com/webhook"
method = "POST"
headers = { "Authorization" = "Bearer xxx", "Content-Type" = "application/json" }
```

#### 4.4 AI机器人配置

**设计理念**：
- 支持多个AI机器人，不同任务可选择不同机器人
- 支持OpenAI、Claude、Ollama等多种Provider
- 每个机器人独立配置API Key、模型参数

**配置文件**: `data/bots.toml`
```toml
# AI机器人配置

[[bots]]
id = "gpt4"
name = "GPT-4o Mini"
remark = "OpenAI GPT-4o-mini，性价比高，适合日常任务"
provider = "openai"
enabled = true

[bots.config]
api_key = "sk-xxx"         # 或通过环境变量: ${OPENAI_API_KEY}
model = "gpt-4o-mini"
base_url = ""              # 可选，自定义API地址
max_tokens = 2000
temperature = 0.3

[[bots]]
id = "claude"
name = "Claude Sonnet"
remark = "Anthropic Claude，适合复杂分析任务"
provider = "claude"
enabled = true

[bots.config]
api_key = "sk-ant-xxx"
model = "claude-sonnet-4-20250514"
max_tokens = 2000
temperature = 0.3

[[bots]]
id = "ollama-local"
name = "本地Ollama"
remark = "本地部署的Ollama，免费但速度较慢"
provider = "ollama"
enabled = true

[bots.config]
base_url = "http://localhost:11434"
model = "llama3"
max_tokens = 2000
temperature = 0.3
```

---

### 5. Web界面设计

#### 5.1 整体布局

```
┌────────────────────────────────────────────────────────────────────────┐
│  🔔 EchoFeed                                              [Settings]   │
├──────────────┬─────────────────────────────────────────────────────────┤
│              │                                                         │
│  📋 任务列表 │   ┌─────────────────────────────────────────────────┐   │
│  ─────────── │   │              任务详情 / 编辑区域               │   │
│  ► 任务1     │   │                                                 │   │
│    任务2     │   │   名称: [________________]                      │   │
│    任务3     │   │   描述: [________________]                      │   │
│              │   │                                                 │   │
│  + 新建任务  │   │   AI机器人: [▼ GPT-4o Mini    ]                │   │
│              │   │                                                 │   │
│  ─────────── │   │   通知渠道: ☑ Telegram ☑ Email ☐ Webhook       │   │
│  📡 RSS订阅  │   │                                                 │   │
│  ─────────── │   │   调度周期: [0 */2 * * *    ] (每2小时)        │   │
│  ► 订阅管理  │   │                                                 │   │
│              │   │   提示词:                                       │   │
│  ─────────── │   │   ┌─────────────────────────────────────────┐   │   │
│  🔔 通知渠道 │   │   │ 你是一个技术新闻编辑...                │   │   │
│  ─────────── │   │   │                                         │   │   │
│  ► 渠道管理  │   │   └─────────────────────────────────────────┘   │   │
│              │   │                                                 │   │
│  ─────────── │   │   [保存]  [立即执行]  [删除]                    │   │
│  🤖 AI机器人 │   │                                                 │   │
│  ─────────── │   └─────────────────────────────────────────────────┘   │
│  ► 机器人管理│                                                         │
│              │                                                         │
└──────────────┴─────────────────────────────────────────────────────────┘
```

#### 5.2 页面路由

```
/                        # 仪表盘 - 概览统计、最近执行记录
/tasks                   # 任务列表
/tasks/:id               # 任务详情/编辑
/tasks/new               # 新建任务
/feeds                   # RSS订阅管理
/feeds/new               # 添加订阅
/channels                # 通知渠道管理
/channels/:id            # 渠道详情/编辑
/bots                    # AI机器人管理
/bots/:id                # 机器人详情/编辑
/logs                    # 执行日志
/settings                # 系统设置
```

#### 5.3 技术选型

**推荐方案：Go模板 + HTMX + Tailwind CSS**

理由：
- 无需Node.js构建环境，部署简单
- 单二进制文件，所有资源内嵌
- HTMX提供足够的交互能力
- Tailwind CSS保证UI美观

```
web/
├── static/
│   ├── css/
│   │   └── tailwind.min.css
│   └── js/
│       └── htmx.min.js
└── templates/
    ├── layout.html        # 基础布局
    ├── sidebar.html       # 侧边栏组件
    ├── index.html         # 仪表盘
    ├── tasks/
    │   ├── list.html      # 任务列表
    │   ├── detail.html    # 任务详情
    │   └── form.html      # 任务表单组件
    ├── feeds/
    │   ├── list.html
    │   └── form.html
    ├── channels/
    │   ├── list.html
    │   └── form.html
    └── bots/
        ├── list.html
        └── form.html
```

---

### 6. API设计

#### 6.1 任务API
```
GET    /api/tasks              # 获取任务列表
POST   /api/tasks              # 创建任务
GET    /api/tasks/:id          # 获取任务详情
PUT    /api/tasks/:id          # 更新任务
DELETE /api/tasks/:id          # 删除任务
POST   /api/tasks/:id/run      # 立即执行任务
GET    /api/tasks/:id/logs     # 获取任务执行日志
```

#### 6.2 RSS订阅API
```
GET    /api/feeds              # 获取订阅列表
POST   /api/feeds              # 添加订阅
GET    /api/feeds/:id          # 获取订阅详情
PUT    /api/feeds/:id          # 更新订阅
DELETE /api/feeds/:id          # 删除订阅
POST   /api/feeds/:id/refresh  # 手动刷新订阅
GET    /api/feeds/:id/articles # 获取订阅的文章列表
```

#### 6.3 通知渠道API
```
GET    /api/channels           # 获取渠道列表
POST   /api/channels           # 添加渠道
GET    /api/channels/:id       # 获取渠道详情
PUT    /api/channels/:id       # 更新渠道
DELETE /api/channels/:id       # 删除渠道
POST   /api/channels/:id/test  # 测试渠道
```

#### 6.4 AI机器人API
```
GET    /api/bots               # 获取机器人列表
POST   /api/bots               # 添加机器人
GET    /api/bots/:id           # 获取机器人详情
PUT    /api/bots/:id           # 更新机器人
DELETE /api/bots/:id           # 删除机器人
POST   /api/bots/:id/test      # 测试机器人
```

#### 6.5 系统API
```
GET    /api/stats              # 获取统计数据
GET    /api/logs               # 获取系统日志
GET    /api/settings           # 获取系统设置
PUT    /api/settings           # 更新系统设置
```

---

### 7. 数据存储设计

#### 7.1 目录结构

```
data/                          # Docker挂载目录
├── rss.toml                   # RSS订阅配置
├── tasks.toml                 # 任务配置
├── channels.toml              # 通知渠道配置
├── bots.toml                  # AI机器人配置
├── settings.toml              # 系统设置
├── rss/                       # RSS内容缓存
│   ├── anthropic-news.json    # 每个订阅一个文件
│   ├── hackernews.json
│   └── 36kr.json
├── logs/                      # 执行日志
│   ├── tasks/                 # 任务执行日志
│   │   ├── 2025-01-10.log
│   │   └── ...
│   └── system.log             # 系统日志
└── cache/                     # 临时缓存
    └── ...
```

#### 7.2 RSS内容缓存格式

`data/rss/{feed_id}.json`:
```json
{
  "feed_id": "anthropic-news",
  "feed_name": "Anthropic News",
  "last_fetch": "2025-01-10T10:30:00Z",
  "items": [
    {
      "id": "guid-xxx",
      "title": "Introducing Claude 3.5",
      "link": "https://www.anthropic.com/news/claude-3-5",
      "content": "Full content here...",
      "published": "2025-01-10T08:00:00Z",
      "fetched": "2025-01-10T10:30:00Z"
    }
  ]
}
```

#### 7.3 系统设置

`data/settings.toml`:
```toml
# 系统设置

[server]
port = 8080
host = "0.0.0.0"

[auth]
enabled = true
username = "admin"
password_hash = "bcrypt_hash_here"

[fetch]
default_interval = 30     # 默认RSS拉取间隔(分钟)
timeout = 30              # 请求超时(秒)
max_concurrent = 5        # 最大并发数
retention_days = 7        # RSS内容保留天数

[log]
level = "info"
max_size_mb = 100
max_backups = 3
```

---

### 8. 项目结构

```
echofeed/
├── cmd/
│   └── server/
│       └── main.go                 # 入口文件
├── internal/
│   ├── config/
│   │   ├── config.go               # 配置加载
│   │   └── toml.go                 # TOML读写工具
│   ├── model/
│   │   ├── feed.go                 # RSS订阅模型
│   │   ├── task.go                 # 任务模型
│   │   ├── channel.go              # 通知渠道模型
│   │   └── bot.go                  # AI机器人模型
│   ├── service/
│   │   ├── feed_service.go         # RSS订阅服务
│   │   ├── fetcher.go              # RSS拉取器
│   │   ├── task_service.go         # 任务服务
│   │   ├── task_engine.go          # 任务执行引擎
│   │   ├── ai_service.go           # AI服务（多机器人）
│   │   └── notifier.go             # 通知推送服务
│   ├── handler/
│   │   ├── feed_handler.go         # RSS API
│   │   ├── task_handler.go         # 任务API
│   │   ├── channel_handler.go      # 渠道API
│   │   ├── bot_handler.go          # 机器人API
│   │   └── page_handler.go         # 页面渲染
│   └── scheduler/
│       └── scheduler.go            # 定时任务调度
├── web/
│   ├── static/                     # 静态资源
│   │   ├── css/
│   │   └── js/
│   └── templates/                  # Go模板
│       ├── layout.html
│       ├── index.html
│       ├── tasks/
│       ├── feeds/
│       ├── channels/
│       └── bots/
├── data/                           # 数据目录（Git忽略）
│   ├── rss.toml
│   ├── tasks.toml
│   ├── channels.toml
│   ├── bots.toml
│   └── rss/
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

---

### 9. Docker部署

#### 9.1 Dockerfile
```dockerfile
# 构建阶段
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o echofeed ./cmd/server

# 运行阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/echofeed .
COPY --from=builder /app/web ./web

EXPOSE 8080
VOLUME ["/app/data"]

CMD ["./echofeed"]
```

#### 9.2 docker-compose.yml
```yaml
version: '3.8'

services:
  echofeed:
    build: .
    container_name: echofeed
    restart: unless-stopped
    ports:
      - "33333:8080"
    volumes:
      - ./data:/app/data
    environment:
      - TZ=Asia/Shanghai
```

#### 9.3 初始化配置

首次启动时，如果data目录下不存在配置文件，系统会自动生成默认配置。

---

### 10. 技术选型

| 组件 | 选型 | 理由 |
|------|------|------|
| 语言 | Go 1.21+ | 高性能、单二进制部署 |
| Web框架 | Gin | 轻量、性能好、生态完善 |
| 配置格式 | TOML | 人类可读、易于编辑 |
| 模板引擎 | html/template | Go标准库、无依赖 |
| 前端交互 | HTMX | 轻量、无需构建 |
| CSS框架 | Tailwind CSS | 美观、响应式 |
| 定时任务 | robfig/cron | Go标准cron库 |
| RSS解析 | gofeed | 功能完整 |
| TOML解析 | BurntSushi/toml | 性能好、功能全 |
| 日志 | zerolog | 高性能结构化日志 |
| HTTP客户端 | resty | 简洁易用 |

---

### 11. 开发计划

#### Phase 1 - MVP (核心功能)
- [ ] 项目初始化，目录结构搭建
- [ ] TOML配置读写模块
- [ ] RSS拉取和解析
- [ ] 基础Web界面（侧边栏布局）
- [ ] RSS订阅管理页面
- [ ] Docker部署

#### Phase 2 - 任务系统
- [ ] 任务配置和管理
- [ ] 任务调度引擎
- [ ] AI机器人集成（OpenAI）
- [ ] 任务执行和日志

#### Phase 3 - 通知推送
- [ ] Telegram推送
- [ ] Webhook推送
- [ ] 邮件通知
- [ ] 通知渠道测试

#### Phase 4 - 增强功能
- [ ] 多AI机器人支持（Claude、Ollama）
- [ ] 执行历史和统计
- [ ] 系统设置页面
- [ ] 认证和安全

---

### 12. 非功能需求

#### 12.1 性能要求
- 支持50+订阅源稳定运行
- RSS拉取超时 < 30s
- AI分析响应 < 30s
- Web界面响应 < 200ms

#### 12.2 可靠性
- RSS拉取失败自动重试
- 任务执行失败记录日志
- 推送失败重试机制
- 配置文件自动备份

#### 12.3 安全性
- 支持Basic Auth认证
- API Key不明文显示
- HTTPS支持（反向代理）

---

### 13. 参考项目

- [Miniflux](https://github.com/miniflux/v2) - Go RSS阅读器
- [RSSHub](https://github.com/DIYgod/RSSHub) - RSS生成工具
- [Shaarli](https://github.com/shaarli/Shaarli) - 自托管书签服务

---

*文档版本: v2.0*
*更新日期: 2025-01-10*
*作者: EchoFeed Team*
