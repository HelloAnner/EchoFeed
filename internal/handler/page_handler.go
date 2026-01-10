package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"

	"github.com/echofeed/echofeed/internal/appinfo"
	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
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

type configKV struct {
	Key   string
	Value string
}

// NewPageHandler 创建页面处理器
func NewPageHandler(cfgMgr *config.Manager, feedSvc *service.FeedService, taskSvc *service.TaskService, channelSvc *service.ChannelService, botSvc *service.BotService, logSvc *service.LogService) *PageHandler {
	// 加载所有模板文件（包含子目录）
	funcs := template.FuncMap{
		"toJSON": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"configEntries": func(m map[string]string) []configKV {
			var entries []configKV
			for k, v := range m {
				entries = append(entries, configKV{Key: k, Value: v})
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
			return entries
		},
		"filterConfig": func(m map[string]string, exclude ...string) []configKV {
			excluded := make(map[string]bool, len(exclude))
			for _, k := range exclude {
				excluded[k] = true
			}
			var entries []configKV
			for k, v := range m {
				if excluded[k] {
					continue
				}
				entries = append(entries, configKV{Key: k, Value: v})
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
			return entries
		},
	}

	tmpl := template.Must(template.New("layout").Funcs(funcs).ParseGlob(filepath.Join("web", "templates", "*.html")))
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
	AppInfo  appinfo.Info
	Data     interface{}
	Feeds    interface{}
	Tasks    interface{}
	Channels interface{}
	Bots     interface{}
	Logs     interface{}
	Stats    interface{}
}

type TaskView struct {
	model.Task
	ScheduleDesc string
	LastRunAt    string
	NextRunAt    string
}

func (h *PageHandler) render(c *gin.Context, tmpl string, data PageData) {
	data.AppInfo = appinfo.Get()
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

	now := time.Now().In(time.Local)
	today := now.Format("2006-01-02")

	rssTodayCount := 0
	rssLastHourNew := 0
	if overview, err := h.feedSvc.LoadOverview(today); err == nil && overview != nil {
		rssTodayCount = len(overview.Articles)
		cutoff := now.Add(-1 * time.Hour)
		for _, a := range overview.Articles {
			if a.Fetched.In(time.Local).After(cutoff) {
				rssLastHourNew++
			}
		}
	}

	sentCount := 0
	sentSvc := service.NewSentService(h.cfgMgr)
	if daily, err := sentSvc.Load(today); err == nil && daily != nil {
		for _, t := range daily.Tasks {
			sentCount += len(t.Articles)
		}
	}

	execCountToday := 0
	if h.logSvc != nil {
		if statsToday, err := h.logSvc.GetStatsToday(); err == nil && statsToday != nil {
			execCountToday = statsToday.TotalExecutions
		}
	}

	h.render(c, "layout.html", PageData{
		Title:    "仪表盘",
		Active:   "index",
		Feeds:    feeds,
		Tasks:    tasks,
		Channels: channels,
		Bots:     bots,
		Stats: map[string]interface{}{
			"rss_last_hour_new": rssLastHourNew,
			"rss_today_count":   rssTodayCount,
			"sent_today_count":  sentCount,
			"exec_today_count":  execCountToday,
		},
	})
}

// Tasks 任务列表页
func (h *PageHandler) Tasks(c *gin.Context) {
	tasks, _ := h.taskSvc.List()
	bots, _ := h.botSvc.List()
	channels, _ := h.channelSvc.List()

	stateSvc := service.NewTaskStateService(h.cfgMgr)
	now := time.Now()

	var views []TaskView
	for _, t := range tasks {
		desc := describeCron(t.Schedule)
		last := "-"
		if ts, ok := stateSvc.GetLastRunAt(t.ID); ok {
			last = ts.In(time.Local).Format("01-02 15:04")
		}

		next := "-"
		if t.Schedule != "" {
			if sch, err := cron.ParseStandard(t.Schedule); err == nil {
				next = sch.Next(now).In(time.Local).Format("01-02 15:04")
			}
		}

		views = append(views, TaskView{
			Task:         t,
			ScheduleDesc: desc,
			LastRunAt:    last,
			NextRunAt:    next,
		})
	}

	h.render(c, "layout.html", PageData{
		Title:    "任务列表",
		Active:   "tasks",
		Tasks:    views,
		Bots:     bots,
		Channels: channels,
	})
}

func describeCron(expr string) string {
	s := strings.TrimSpace(expr)
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return "-"
	}

	// 每N分钟："*/N * * * *"
	if strings.HasPrefix(s, "*/") && strings.HasSuffix(s, " * * * *") {
		n := strings.TrimPrefix(strings.TrimSuffix(s, " * * * *"), "*/")
		return "每 " + n + " 分钟"
	}

	// M/N * * * *
	if parts := strings.Split(s, " "); len(parts) == 5 {
		minField, hourField := parts[0], parts[1]

		// 分钟：M/N
		if strings.Contains(minField, "/") && hourField == "*" {
			return "每 " + strings.Split(minField, "/")[1] + " 分钟（从第 " + strings.Split(minField, "/")[0] + " 分钟开始）"
		}

		// 小时：M */N
		if strings.Contains(hourField, "*/") && parts[2] == "*" && parts[3] == "*" && parts[4] == "*" {
			return "每 " + strings.TrimPrefix(hourField, "*/") + " 小时（第 " + minField + " 分钟）"
		}

		// 天：0 H */N * *
		if minField == "0" && strings.Contains(parts[2], "*/") && parts[3] == "*" && parts[4] == "*" {
			return "每 " + strings.TrimPrefix(parts[2], "*/") + " 天（" + pad2(hourField) + ":00）"
		}

		// 每天：0 H * * *
		if minField == "0" && parts[2] == "*" && parts[3] == "*" && parts[4] == "*" {
			if strings.Contains(hourField, ",") {
				var times []string
				for _, h := range strings.Split(hourField, ",") {
					h = strings.TrimSpace(h)
					if h == "" {
						continue
					}
					times = append(times, pad2(h)+":00")
				}
				if len(times) > 0 {
					return "每天 " + strings.Join(times, "、")
				}
			}
			return "每天 " + pad2(hourField) + ":00"
		}
	}

	return "Cron：" + s
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
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
