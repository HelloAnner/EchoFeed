# EchoFeed

一个自托管的 RSS 聚合与 AI 筛选推送工具：定时拉取 RSS → 任务按提示词筛选 → 通过通知渠道推送（支持企业微信机器人）。

English README: `README.md`

## 功能

- RSS 订阅管理与定时拉取
- 任务：按关键词/提示词筛选内容，控制最低重要性
- 推送：企业微信/Webhook/Telegram/Email/Bark
- 任务级去重：同一任务不会重复推送同一篇文章
- Web UI：配置与管理（任务/订阅/渠道/机器人/日志）

## 快速启动（Docker）

前置：已安装 Docker + Docker Compose。

```bash
make start
```

访问：
- `http://localhost:33333`

常用命令：
```bash
make restart
make stop
```

## Web UI 截图

以下截图来自本地启动后的 Web UI（`http://localhost:33333` / `http://127.0.0.1:33333`），按左侧边栏入口逐一展示，方便快速了解各功能页面。

### 仪表盘

![仪表盘](img/dashboard.png)

### 任务列表

![任务列表](img/tasks.png)

### RSS 订阅

![RSS 订阅](img/feeds.png)

### 通知渠道

![通知渠道](img/channels.png)

### AI配置

![AI配置](img/bots.png)

### 执行日志

![执行日志](img/logs.png)

### 系统设置

![系统设置](img/settings.png)

## 配置

配置文件默认在 `./data/`：
- `data/rss.toml`：RSS 源
- `data/tasks.toml`：任务
- `data/channels.toml`：通知渠道
- `data/bots.toml`：AI 机器人
- `data/settings.toml`：服务端口等

