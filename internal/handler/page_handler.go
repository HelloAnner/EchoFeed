package handler

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/service"
)

// PageHandler 页面处理器
type PageHandler struct {
	cfgMgr     *config.Manager
	feedSvc    *service.FeedService
	taskSvc    *service.TaskService
	channelSvc *service.ChannelService
	botSvc     *service.BotService
	logSvc     *service.LogService
	templates  *template.Template
}

// NewPageHandler 创建页面处理器
func NewPageHandler(cfgMgr *config.Manager, feedSvc *service.FeedService, taskSvc *service.TaskService, channelSvc *service.ChannelService, botSvc *service.BotService, logSvc *service.LogService) *PageHandler {
	// 加载所有模板文件（包含子目录）
	tmpl := template.Must(template.ParseGlob(filepath.Join("web", "templates", "*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join("web", "templates", "components", "*.html")))
	template.Must(tmpl.ParseGlob(filepath.Join("web", "templates", "pages", "*.html")))

	return &PageHandler{
		cfgMgr:     cfgMgr,
		feedSvc:    feedSvc,
		taskSvc:    taskSvc,
		channelSvc: channelSvc,
		botSvc:     botSvc,
		logSvc:     logSvc,
		templates:  tmpl,
	}
}

// PageData 页面数据
type PageData struct {
	Title    string
	Active   string
	Data     interface{}
	Feeds    interface{}
	Tasks    interface{}
	Channels interface{}
	Bots     interface{}
	Logs     interface{}
	Stats    interface{}
}

func (h *PageHandler) render(c *gin.Context, tmpl string, data PageData) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(c.Writer, tmpl, data); err != nil {
		c.String(http.StatusInternalServerError, "Template error: %v", err)
	}
}

// Index 首页/仪表盘
func (h *PageHandler) Index(c *gin.Context) {
	feeds, _ := h.feedSvc.List()
	tasks, _ := h.taskSvc.List()
	channels, _ := h.channelSvc.List()
	bots, _ := h.botSvc.List()

	h.render(c, "layout.html", PageData{
		Title:    "仪表盘",
		Active:   "index",
		Feeds:    feeds,
		Tasks:    tasks,
		Channels: channels,
		Bots:     bots,
	})
}

// Tasks 任务列表页
func (h *PageHandler) Tasks(c *gin.Context) {
	tasks, _ := h.taskSvc.List()
	bots, _ := h.botSvc.List()
	channels, _ := h.channelSvc.List()

	h.render(c, "layout.html", PageData{
		Title:    "任务列表",
		Active:   "tasks",
		Tasks:    tasks,
		Bots:     bots,
		Channels: channels,
	})
}

// TaskNew 新建任务页
func (h *PageHandler) TaskNew(c *gin.Context) {
	bots, _ := h.botSvc.List()
	channels, _ := h.channelSvc.List()

	h.render(c, "layout.html", PageData{
		Title:    "新建任务",
		Active:   "tasks",
		Bots:     bots,
		Channels: channels,
	})
}

// TaskDetail 任务详情页
func (h *PageHandler) TaskDetail(c *gin.Context) {
	id := c.Param("id")
	task, _ := h.taskSvc.Get(id)
	bots, _ := h.botSvc.List()
	channels, _ := h.channelSvc.List()

	h.render(c, "layout.html", PageData{
		Title:    "任务详情",
		Active:   "tasks",
		Data:     task,
		Bots:     bots,
		Channels: channels,
	})
}

// Feeds RSS订阅列表页
func (h *PageHandler) Feeds(c *gin.Context) {
	feeds, _ := h.feedSvc.List()

	h.render(c, "layout.html", PageData{
		Title:  "RSS订阅",
		Active: "feeds",
		Feeds:  feeds,
	})
}

// FeedNew 新建RSS订阅页
func (h *PageHandler) FeedNew(c *gin.Context) {
	h.render(c, "layout.html", PageData{
		Title:  "添加订阅",
		Active: "feeds",
	})
}

// Channels 通知渠道列表页
func (h *PageHandler) Channels(c *gin.Context) {
	channels, _ := h.channelSvc.List()

	h.render(c, "layout.html", PageData{
		Title:    "通知渠道",
		Active:   "channels",
		Channels: channels,
	})
}

// ChannelNew 新建通知渠道页
func (h *PageHandler) ChannelNew(c *gin.Context) {
	h.render(c, "layout.html", PageData{
		Title:  "添加渠道",
		Active: "channels",
	})
}

// ChannelDetail 通知渠道详情页
func (h *PageHandler) ChannelDetail(c *gin.Context) {
	id := c.Param("id")
	channel, _ := h.channelSvc.Get(id)

	h.render(c, "layout.html", PageData{
		Title:  "渠道详情",
		Active: "channels",
		Data:   channel,
	})
}

// Bots AI机器人列表页
func (h *PageHandler) Bots(c *gin.Context) {
	bots, _ := h.botSvc.List()

	h.render(c, "layout.html", PageData{
		Title:  "AI机器人",
		Active: "bots",
		Bots:   bots,
	})
}

// BotNew 新建AI机器人页
func (h *PageHandler) BotNew(c *gin.Context) {
	h.render(c, "layout.html", PageData{
		Title:  "添加机器人",
		Active: "bots",
	})
}

// BotDetail AI机器人详情页
func (h *PageHandler) BotDetail(c *gin.Context) {
	id := c.Param("id")
	bot, _ := h.botSvc.Get(id)

	h.render(c, "layout.html", PageData{
		Title:  "机器人详情",
		Active: "bots",
		Data:   bot,
	})
}

// Logs 日志页
func (h *PageHandler) Logs(c *gin.Context) {
	logs, _ := h.logSvc.List(100)
	stats, _ := h.logSvc.GetStats()
	statsToday, _ := h.logSvc.GetStatsToday()
	tasks, _ := h.taskSvc.List()

	h.render(c, "layout.html", PageData{
		Title:  "执行日志",
		Active: "logs",
		Logs:   logs,
		Stats:  map[string]interface{}{"all": stats, "today": statsToday},
		Tasks:  tasks,
	})
}

// Settings 设置页
func (h *PageHandler) Settings(c *gin.Context) {
	settings, _ := h.cfgMgr.LoadSettings()

	h.render(c, "layout.html", PageData{
		Title:  "系统设置",
		Active: "settings",
		Data:   settings,
	})
}
