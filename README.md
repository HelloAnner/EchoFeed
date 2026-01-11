# EchoFeed

Self-hosted RSS aggregation + AI filtering + notifications: pull RSS on schedule → filter with tasks/prompts → push via notification channels (WeCom bot supported).

中文文档：`README.zh-CN.md`

## Features

- RSS subscription management and scheduled fetching
- Tasks: filter by keywords/prompts with minimum importance threshold
- Notifications: WeCom / Webhook / Telegram / Email / Bark
- Task-level deduplication: no duplicate pushes for the same post in the same task
- Web UI: manage tasks/subscriptions/channels/bots/logs

## Quick Start (Docker)

Prerequisites: Docker + Docker Compose installed.

```bash
make start
```

Access:
- `http://localhost:33333`

Common commands:
```bash
make restart
make stop
```

## Web UI Screenshots

Screenshots are taken from the Web UI after starting locally (`http://localhost:33333` / `http://127.0.0.1:33333`), one per left sidebar entry.

### Dashboard

![Dashboard](img/dashboard.png)

### Tasks

![Tasks](img/tasks.png)

### RSS Feeds

![RSS Feeds](img/feeds.png)

### Notification Channels

![Notification Channels](img/channels.png)

### AI Bots

![AI Bots](img/bots.png)

### Execution Logs

![Execution Logs](img/logs.png)

### Settings

![Settings](img/settings.png)

## Configuration

Config files are stored in `./data/` by default:
- `data/rss.toml`: RSS sources
- `data/tasks.toml`: tasks
- `data/channels.toml`: notification channels
- `data/bots.toml`: AI bots
- `data/settings.toml`: server port and settings

