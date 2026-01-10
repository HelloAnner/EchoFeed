# EchoFeed 实现工作流

## 工作流概览

```
Phase 1: 项目基础 (MVP)
├── Step 1.1: 项目初始化
├── Step 1.2: 配置模块
├── Step 1.3: RSS拉取模块
├── Step 1.4: Web基础框架
├── Step 1.5: RSS管理页���
└── Step 1.6: Docker部署

Phase 2: 任务系统
├── Step 2.1: 任务模型与配置
├── Step 2.2: 任务调度引擎
├── Step 2.3: AI服务集成
└── Step 2.4: 任务管理页面

Phase 3: 通知推送
├── Step 3.1: 通知框架
├── Step 3.2: Telegram推送
├── Step 3.3: Webhook推送
└── Step 3.4: 通知管理页面

Phase 4: 增强功能
├── Step 4.1: 多AI机器人
├── Step 4.2: 执行历史
├── Step 4.3: 系统设置
└── Step 4.4: 认证安全
```

---

## Phase 1: 项目基础 (MVP)

### Step 1.1: 项目初始化

**目标**: 搭建Go项目基础结构

**任务清单**:
```
[ ] 1.1.1 初始化Go模块
    - go mod init github.com/xxx/echofeed
    - 配置Go版本 1.21+

[ ] 1.1.2 创建目录结构
    echofeed/
    ├── cmd/server/main.go
    ├── internal/
    │   ├── config/
    │   ├── model/
    │   ├── service/
    │   ├── handler/
    │   └── scheduler/
    ├── web/
    │   ├── static/
    │   └── templates/
    └── data/

[ ] 1.1.3 安装核心依赖
    go get github.com/gin-gonic/gin
    go get github.com/BurntSushi/toml
    go get github.com/mmcdole/gofeed
    go get github.com/robfig/cron/v3
    go get github.com/rs/zerolog
    go get github.com/go-resty/resty/v2

[ ] 1.1.4 创建入口文件
    - cmd/server/main.go
    - 基础HTTP服务启动
```

**产出文件**:
- `go.mod`, `go.sum`
- `cmd/server/main.go`
- 目录结构

**验收标准**:
- `go run ./cmd/server` 启动成功
- 访问 `http://localhost:8080` 返回响应

---

### Step 1.2: 配置模块

**目标**: 实现TOML配置读写

**任务清单**:
```
[ ] 1.2.1 定义配置模型
    internal/model/
    ├── feed.go      # Feed, FeedSettings
    ├── task.go      # Task, TaskPrompt
    ├── channel.go   # Channel, ChannelConfig
    └── bot.go       # Bot, BotConfig

[ ] 1.2.2 实现TOML读写工具
    internal/config/
    ├── config.go    # 配置管理器
    └── toml.go      # TOML读写辅助函数

[ ] 1.2.3 实现配置服务
    - LoadFeeds() / SaveFeeds()
    - LoadTasks() / SaveTasks()
    - LoadChannels() / SaveChannels()
    - LoadBots() / SaveBots()

[ ] 1.2.4 创建默认配置模板
    data/
    ├── rss.toml.example
    ├── tasks.toml.example
    ├── channels.toml.example
    └── bots.toml.example

[ ] 1.2.5 首次启动自动初始化
    - 检测配置文件是否存在
    - 不存在则生成默认配置
```

**产出文件**:
- `internal/model/*.go` (4个文件)
- `internal/config/config.go`
- `internal/config/toml.go`
- `data/*.toml.example` (4个文件)

**验收标准**:
- 配置文件读取正确
- 配置文件保存后格式正确
- 首次启动自动生成配置

---

### Step 1.3: RSS拉取模块

**目标**: 实现RSS定时拉取和缓存

**任务清单**:
```
[ ] 1.3.1 实现RSS拉取器
    internal/service/fetcher.go
    - FetchFeed(url string) (*gofeed.Feed, error)
    - 支持超时配置
    - 支持User-Agent配置

[ ] 1.3.2 实现RSS缓存存储
    internal/service/feed_service.go
    - SaveFeedCache(feedID string, items []Article)
    - LoadFeedCache(feedID string) []Article
    - 存储到 data/rss/{feed_id}.json

[ ] 1.3.3 实现定时拉取调度
    internal/scheduler/scheduler.go
    - 使用 robfig/cron
    - 根据每个Feed的interval配置调度
    - 支持动态添加/移除调度任务

[ ] 1.3.4 实现并发拉取控制
    - 使用 semaphore 控制并发数
    - max_concurrent 配置项

[ ] 1.3.5 实现拉取日志记录
    - 记录拉取时间、状态、错误信息
    - 更新Feed状态(active/error)
```

**产出文件**:
- `internal/service/fetcher.go`
- `internal/service/feed_service.go`
- `internal/scheduler/scheduler.go`

**验收标准**:
- 手动触发拉取成功
- RSS内容正确存储到JSON
- 定时任务按配置执行
- 并发数控制有效

---

### Step 1.4: Web基础框架

**目标**: 搭建Web界面基础结构

**任务清单**:
```
[ ] 1.4.1 下载前端资源
    web/static/
    ├── css/tailwind.min.css   # Tailwind CSS CDN
    └── js/htmx.min.js         # HTMX

[ ] 1.4.2 创建基础布局模板
    web/templates/
    ├── layout.html            # 基础布局(头部、侧边栏、内容区)
    ├── sidebar.html           # 侧边栏组件
    └── components/
        ├── header.html
        ├── toast.html         # 提示消息组件
        └── modal.html         # 模态框组件

[ ] 1.4.3 实现模板渲染器
    internal/handler/page_handler.go
    - 模板加载和缓存
    - 公共数据注入
    - HTMX片段渲染

[ ] 1.4.4 创建仪表盘页面
    web/templates/index.html
    - 统计概览卡片
    - 最近执行记录

[ ] 1.4.5 配置静态资源服务
    - /static/* 路由
    - 静态资源缓存头
```

**产出文件**:
- `web/static/css/tailwind.min.css`
- `web/static/js/htmx.min.js`
- `web/templates/*.html` (约8个文件)
- `internal/handler/page_handler.go`

**验收标准**:
- 访问首页显示侧边栏布局
- 样式正确加载
- HTMX交互正常

---

### Step 1.5: RSS管理页面

**目标**: 实现RSS订阅的CRUD界面

**任务清单**:
```
[ ] 1.5.1 实现Feed API
    internal/handler/feed_handler.go
    - GET    /api/feeds          # 列表
    - POST   /api/feeds          # 添加
    - GET    /api/feeds/:id      # 详情
    - PUT    /api/feeds/:id      # 更新
    - DELETE /api/feeds/:id      # 删除
    - POST   /api/feeds/:id/refresh  # 刷新

[ ] 1.5.2 创建RSS列表页面
    web/templates/feeds/list.html
    - 订阅源列表表格
    - 状态指示(active/error)
    - 操作按钮(编辑/删除/刷新)

[ ] 1.5.3 创建RSS表单组件
    web/templates/feeds/form.html
    - 添加/编辑表单
    - 字段: name, url, remark, interval, enabled
    - HTMX表单提交

[ ] 1.5.4 实现HTMX交互
    - 添加订阅后刷新列表
    - 删除确认对话框
    - 手动刷新反馈

[ ] 1.5.5 URL验证
    - 检测URL格式
    - 测试RSS可访问性
```

**产出文件**:
- `internal/handler/feed_handler.go`
- `web/templates/feeds/list.html`
- `web/templates/feeds/form.html`

**验收标准**:
- 添加订阅成功并保存到TOML
- 列表正确显示所有订阅
- 编辑/删除操作正常
- 手动刷新触发拉取

---

### Step 1.6: Docker部署

**目标**: 实现Docker容器化部署

**任务清单**:
```
[ ] 1.6.1 创建Dockerfile
    - 多阶段构建
    - 静态编译Go二进制
    - 内嵌web资源(或复制)

[ ] 1.6.2 创建docker-compose.yml
    - 端口映射 33333:8080
    - 数据卷挂载 ./data:/app/data
    - 时区配置 TZ=Asia/Shanghai

[ ] 1.6.3 创建.dockerignore
    - 忽略不必要文件

[ ] 1.6.4 创建.gitignore
    - 忽略data目录
    - 忽略二进制文件

[ ] 1.6.5 测试Docker构建和运行
    - docker-compose build
    - docker-compose up -d
    - 验证功能正常
```

**产出文件**:
- `Dockerfile`
- `docker-compose.yml`
- `.dockerignore`
- `.gitignore`

**验收标准**:
- Docker构建成功
- 容器启动正常
- 数据持久化到宿主机

---

## Phase 2: 任务系统

### Step 2.1: 任务模型与配置

**目标**: 实现任务配置管理

**任务清单**:
```
[ ] 2.1.1 完善Task模型
    internal/model/task.go
    - Task结构体
    - TaskPrompt结构体
    - 默认提示词模板

[ ] 2.1.2 实现Task配置服务
    internal/service/task_service.go
    - CRUD操作
    - 配置验证

[ ] 2.1.3 实现Task API
    internal/handler/task_handler.go
    - 完整CRUD API
    - 立即执行API
    - 执行日志API
```

**产出文件**:
- `internal/model/task.go` (完善)
- `internal/service/task_service.go`
- `internal/handler/task_handler.go`

---

### Step 2.2: 任务调度引擎

**目标**: 实现任务定时执行

**任务清单**:
```
[ ] 2.2.1 实现任务执行引擎
    internal/service/task_engine.go
    - ExecuteTask(taskID string)
    - 加载全量RSS内容
    - 调用AI分析
    - 触发通知推送

[ ] 2.2.2 集成到调度器
    internal/scheduler/scheduler.go
    - 添加任务调度
    - 支持Cron表达式
    - 动态更新调度

[ ] 2.2.3 实现执行日志
    - 记录执行时间、结果
    - 存储到 data/logs/tasks/
    - 日志轮转

[ ] 2.2.4 实现手动执行
    - POST /api/tasks/:id/run
    - 异步执行
    - 返回执行状态
```

**产出文件**:
- `internal/service/task_engine.go`
- `internal/scheduler/scheduler.go` (更新)

---

### Step 2.3: AI服务集成

**目标**: 集成OpenAI API

**任务清单**:
```
[ ] 2.3.1 实现AI服务接口
    internal/service/ai_service.go
    - AIProvider接口定义
    - Analyze(prompt string, content string) (AIResponse, error)

[ ] 2.3.2 实现OpenAI Provider
    internal/service/ai_openai.go
    - 调用OpenAI Chat API
    - 支持base_url配置
    - 错误处理和重试

[ ] 2.3.3 实现提示词模板
    - 默认模板加载
    - 变量替换 {keywords}, {min_importance}
    - 自定义模板支持

[ ] 2.3.4 实现AI响应解析
    - JSON响应解析
    - 结果验证
    - 错误处理
```

**产出文件**:
- `internal/service/ai_service.go`
- `internal/service/ai_openai.go`

---

### Step 2.4: 任务管理页面

**目标**: 实现任务配置界面

**任务清单**:
```
[ ] 2.4.1 创建任务列表页面
    web/templates/tasks/list.html
    - 任务列表
    - 状态显示(enabled/disabled)
    - 下次执行时间
    - 操作按钮

[ ] 2.4.2 创建任务表单页面
    web/templates/tasks/form.html
    - 基本信息(name, description)
    - AI机器人选择(下拉)
    - 通知渠道选择(多选)
    - 调度配置(Cron输入+预览)
    - 提示词编辑器(textarea)

[ ] 2.4.3 创建任务详情页面
    web/templates/tasks/detail.html
    - 任务信息展示
    - 执行历史
    - 手动执行按钮

[ ] 2.4.4 实现Cron表达式预览
    - 解析Cron显示人类可读描述
    - 显示下次N次执行时间
```

**产出文件**:
- `web/templates/tasks/list.html`
- `web/templates/tasks/form.html`
- `web/templates/tasks/detail.html`

---

## Phase 3: 通知推送

### Step 3.1: 通知框架

**目标**: 设计通知推送架构

**任务清单**:
```
[ ] 3.1.1 定义通知接口
    internal/service/notifier.go
    - Notifier接口
    - Send(message NotifyMessage) error
    - Test() error

[ ] 3.1.2 实现通知服务
    - 根据渠道类型路由
    - 多渠道并发发送
    - 失败重试机制

[ ] 3.1.3 定义消息格式
    - NotifyMessage结构体
    - Markdown格式化
    - 多语言支持
```

**产出文件**:
- `internal/service/notifier.go`
- `internal/model/notify.go`

---

### Step 3.2: Telegram推送

**目标**: 实现Telegram机器人推送

**任务清单**:
```
[ ] 3.2.1 实现Telegram Notifier
    internal/service/notify_telegram.go
    - 调用Telegram Bot API
    - 支持Markdown格式
    - 支持长消息分割

[ ] 3.2.2 实现消息格式化
    - 标题、摘要、链接
    - 来源和重要性标识
    - 美化排版

[ ] 3.2.3 实现测试功能
    - 发送测试消息
    - 验证bot_token和chat_id
```

**产出文件**:
- `internal/service/notify_telegram.go`

---

### Step 3.3: Webhook推送

**目标**: 实现通用Webhook推送

**任务清单**:
```
[ ] 3.3.1 实现Webhook Notifier
    internal/service/notify_webhook.go
    - 支持自定义URL
    - 支持自定义Headers
    - 支持GET/POST方法

[ ] 3.3.2 定义Webhook Payload
    - JSON格式
    - 包含完整消息内容
    - 支持模板变量

[ ] 3.3.3 实现测试功能
    - 发送测试请求
    - 验证响应状态
```

**产出文件**:
- `internal/service/notify_webhook.go`

---

### Step 3.4: 通知管理页面

**目标**: 实现通知渠道配置界面

**任务清单**:
```
[ ] 3.4.1 实现Channel API
    internal/handler/channel_handler.go
    - CRUD API
    - 测试API

[ ] 3.4.2 创建渠道列表页面
    web/templates/channels/list.html
    - 渠道列表
    - 类型图标
    - 测试按钮

[ ] 3.4.3 创建渠道表单页面
    web/templates/channels/form.html
    - 渠道类型选择
    - 动态配置表单(根据类型)
    - Telegram: bot_token, chat_id
    - Webhook: url, method, headers

[ ] 3.4.4 实现测试功能
    - 发送测试消息
    - 显示测试结果
```

**产出文件**:
- `internal/handler/channel_handler.go`
- `web/templates/channels/list.html`
- `web/templates/channels/form.html`

---

## Phase 4: 增强功能

### Step 4.1: 多AI机器人

**目标**: 支持多种AI Provider

**任务清单**:
```
[ ] 4.1.1 实现Claude Provider
    internal/service/ai_claude.go
    - Anthropic API调用
    - 消息格式适配

[ ] 4.1.2 实现Ollama Provider
    internal/service/ai_ollama.go
    - 本地Ollama API
    - 模型选择

[ ] 4.1.3 实现Bot API
    internal/handler/bot_handler.go
    - CRUD API
    - 测试API

[ ] 4.1.4 创建机器人管理页面
    web/templates/bots/
    - list.html
    - form.html
    - 动态配置表单
```

**产出文件**:
- `internal/service/ai_claude.go`
- `internal/service/ai_ollama.go`
- `internal/handler/bot_handler.go`
- `web/templates/bots/*.html`

---

### Step 4.2: 执行历史

**目标**: 实现执行记录和统计

**任务清单**:
```
[ ] 4.2.1 实现执行记录存储
    - 任务执行记录
    - 推送记录
    - 按日期归档

[ ] 4.2.2 实现日志查询API
    - 分页查询
    - 按任务/日期过滤

[ ] 4.2.3 创建日志页面
    web/templates/logs/
    - 执行历史列表
    - 详情查看
    - 筛选功能

[ ] 4.2.4 更新仪表盘
    - 统计卡片
    - 最近执行记录
    - 图表(可选)
```

**产出文件**:
- `internal/service/log_service.go`
- `internal/handler/log_handler.go`
- `web/templates/logs/list.html`

---

### Step 4.3: 系统设置

**目标**: 实现系统配置界面

**任务清单**:
```
[ ] 4.3.1 实现Settings API
    - GET /api/settings
    - PUT /api/settings

[ ] 4.3.2 创建设置页面
    web/templates/settings.html
    - RSS拉取配置
    - 日志配置
    - 其他系统设置

[ ] 4.3.3 实现配置热更新
    - 保存后生效
    - 重启调度器
```

**产出文件**:
- `internal/handler/settings_handler.go`
- `web/templates/settings.html`

---

### Step 4.4: 认证安全

**目标**: 实现基础认证

**任务清单**:
```
[ ] 4.4.1 实现Basic Auth中间件
    - 用户名密码验证
    - bcrypt密码哈希

[ ] 4.4.2 创建登录页面
    web/templates/login.html
    - 登录表单
    - 错误提示

[ ] 4.4.3 实现会话管理
    - Cookie/Token
    - 登出功能

[ ] 4.4.4 API密钥保护
    - 配置文件中密钥不明文显示
    - 编辑时显示掩码
```

**产出文件**:
- `internal/middleware/auth.go`
- `web/templates/login.html`

---

## 文件清单汇总

### 后端文件 (Go)

```
cmd/
└── server/
    └── main.go

internal/
├── config/
│   ├── config.go
│   └── toml.go
├── model/
│   ├── feed.go
│   ├── task.go
│   ├── channel.go
│   ├── bot.go
│   └── notify.go
├── service/
│   ├── feed_service.go
│   ├── fetcher.go
│   ├── task_service.go
│   ├── task_engine.go
│   ├── ai_service.go
│   ├── ai_openai.go
│   ├── ai_claude.go
│   ├── ai_ollama.go
│   ├── notifier.go
│   ├── notify_telegram.go
│   ├── notify_webhook.go
│   └── log_service.go
├── handler/
│   ├── feed_handler.go
│   ├── task_handler.go
│   ├── channel_handler.go
│   ├── bot_handler.go
│   ├── log_handler.go
│   ├── settings_handler.go
│   └── page_handler.go
├── scheduler/
│   └── scheduler.go
└── middleware/
    └── auth.go
```

### 前端文件 (Templates)

```
web/
├── static/
│   ├── css/
│   │   └── tailwind.min.css
│   └── js/
│       └── htmx.min.js
└── templates/
    ├── layout.html
    ├── sidebar.html
    ├── index.html
    ├── login.html
    ├── settings.html
    ├── components/
    │   ├── header.html
    │   ├── toast.html
    │   └── modal.html
    ├── tasks/
    │   ├── list.html
    │   ├── form.html
    │   └── detail.html
    ├── feeds/
    │   ├── list.html
    │   └── form.html
    ├── channels/
    │   ├── list.html
    │   └── form.html
    ├── bots/
    │   ├── list.html
    │   └── form.html
    └── logs/
        └── list.html
```

### 配置文件

```
data/
├── rss.toml
├── tasks.toml
├── channels.toml
├── bots.toml
├── settings.toml
└── rss/
    └── *.json

Dockerfile
docker-compose.yml
.dockerignore
.gitignore
go.mod
go.sum
README.md
```

---

## 依赖关系图

```
Phase 1 (基础)
┌─────────┐    ┌─────────┐    ┌─────────┐
│ Step1.1 │───►│ Step1.2 │───►│ Step1.3 │
│ 项目初始化│    │ 配置模块 │    │ RSS5分钟定期拉取  │
└─────────┘    └─────────┘    └────┬────┘
                                   │
┌─────────┐    ┌─────────┐         │
│ Step1.6 │◄───│ Step1.5 │◄───┬────┘
│ Docker  │    │ RSS页面  │    │
└─────────┘    └─────────┘    │
                              │
               ┌─────────┐    │
               │ Step1.4 │────┘
               │ Web框架  │
               └─────────┘

Phase 2 (任务) - 依赖 Phase 1
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Step2.1 │───►│ Step2.2 │───►│ Step2.3 │───►│ Step2.4 │
│ 任务模型 │    │ 调度引擎 │    │ AI集成  │    │ 任务页面 │
└─────────┘    └─────────┘    └─────────┘    └─────────┘

Phase 3 (通知) - 依赖 Phase 2
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Step3.1 │───►│ Step3.2 │    │ Step3.3 │───►│ Step3.4 │
│ 通知框架 │    │ Telegram│    │ Webhook │    │ 通知页面 │
└─────────┘    └────┬────┘    └────┬────┘    └─────────┘
                    │              │
                    └──────┬───────┘
                           │
                    可并行开发

Phase 4 (增强) - 依赖 Phase 3
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Step4.1 │    │ Step4.2 │    │ Step4.3 │    │ Step4.4 │
│ 多AI机器人│    │ 执行历史 │    │ 系统设置 │    │ 认证安全 │
└─────────┘    └─────────┘    └─────────┘    └─────────┘
     │              │              │              │
     └──────────────┴──────────────┴──────────────┘
                    可并行开发
```

---

## 快速启动命令

### Phase 1 启动命令
```bash
# 1.1 项目初始化
mkdir -p cmd/server internal/{config,model,service,handler,scheduler} web/{static,templates}
go mod init github.com/xxx/echofeed

# 1.2 安装依赖
go get github.com/gin-gonic/gin
go get github.com/BurntSushi/toml
go get github.com/mmcdole/gofeed
go get github.com/robfig/cron/v3
go get github.com/rs/zerolog
go get github.com/go-resty/resty/v2

# 1.6 Docker构建
docker-compose build
docker-compose up -d
```

---

*工作流版本: v1.0*
*生成日期: 2025-01-10*
