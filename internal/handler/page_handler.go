package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
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
		"add": func(a, b int) int { return a + b },
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
	BotName      string
	ScheduleDesc string
	LastRunAt    string
	NextRunAt    string
}

type FeedListView struct {
	Items    []FeedView
	Total    int
	TotalAll int
	Page     int
	PageSize int
	Pages    int
	Query    string
}

type FeedView struct {
	model.Feed
	TodayCount int
	LastUpdate string
	NextUpdate string
}

type TaskListView struct {
	Items    []TaskView
	Total    int
	TotalAll int
	Page     int
	PageSize int
	Pages    int
	Query    string
}

type LogListView struct {
	Items    []model.ExecutionLog
	Total    int
	TotalAll int
	Page     int
	PageSize int
	Pages    int
	Query    string
	Status   string
	TaskID   string
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

	q := strings.TrimSpace(c.Query("q"))
	page := 1
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	pageSize := 8

	botNames := make(map[string]string, len(bots))
	for _, b := range bots {
		botNames[b.ID] = b.Name
	}

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
			BotName:      botNames[t.BotID],
			ScheduleDesc: desc,
			LastRunAt:    last,
			NextRunAt:    next,
		})
	}

	totalAll := len(views)
	filtered := filterTasksByQuery(views, q)
	total := len(filtered)
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	items := []TaskView{}
	if start < end {
		items = filtered[start:end]
	}

	h.render(c, "layout.html", PageData{
		Title:    "任务列表",
		Active:   "tasks",
		Tasks:    TaskListView{Items: items, Total: total, TotalAll: totalAll, Page: page, PageSize: pageSize, Pages: pages, Query: q},
		Bots:     bots,
		Channels: channels,
	})
}

func filterTasksByQuery(tasks []TaskView, q string) []TaskView {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return tasks
	}
	var out []TaskView
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.ID), q) ||
			strings.Contains(strings.ToLower(t.Name), q) ||
			strings.Contains(strings.ToLower(t.Description), q) ||
			strings.Contains(strings.ToLower(t.Remark), q) ||
			strings.Contains(strings.ToLower(t.BotID), q) ||
			strings.Contains(strings.ToLower(t.BotName), q) ||
			strings.Contains(strings.ToLower(t.Schedule), q) {
			out = append(out, t)
		}
	}
	return out
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
	// 按添加时间排序：配置文件中追加在末尾，因此倒序展示即为“最近添加在最前”。
	feedsSorted := make([]model.Feed, len(feeds))
	copy(feedsSorted, feeds)
	for i, j := 0, len(feedsSorted)-1; i < j; i, j = i+1, j-1 {
		feedsSorted[i], feedsSorted[j] = feedsSorted[j], feedsSorted[i]
	}

	now := time.Now().In(time.Local)
	today := now.Format("2006-01-02")

	// 统计：今日每个订阅的文章数、最新抓取时间（来自 overview.json）
	todayCount := make(map[string]int)
	todayLastFetched := make(map[string]time.Time)
	if overview, err := h.feedSvc.LoadOverview(today); err == nil && overview != nil {
		for _, a := range overview.Articles {
			todayCount[a.FeedID]++
			if prev, ok := todayLastFetched[a.FeedID]; !ok || a.Fetched.After(prev) {
				todayLastFetched[a.FeedID] = a.Fetched
			}
		}
	}

	// 下次更新时间：按系统默认拉取间隔推算（全局 RSS fetch）
	nextUpdate := "-"
	intervalMin := 5
	if settings, err := h.cfgMgr.LoadSettings(); err == nil && settings != nil && settings.Fetch.DefaultInterval > 0 {
		intervalMin = settings.Fetch.DefaultInterval
	}
	nextUpdate = nextTickByMinuteInterval(now, intervalMin).Format("01-02 15:04")

	feedState := service.NewFeedStateService(h.cfgMgr)

	q := strings.TrimSpace(c.Query("q"))
	page := 1
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	pageSize := 8

	totalAll := len(feedsSorted)
	filtered := filterFeedsByQuery(feedsSorted, q)
	total := len(filtered)
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	items := []FeedView{}
	if start < end {
		for _, f := range filtered[start:end] {
			lastUpdate := "-"
			if st, err := feedState.Load(f.ID); err == nil && st != nil && st.LastFetchAt != "" {
				if t, err := time.Parse(time.RFC3339, st.LastFetchAt); err == nil {
					lastUpdate = t.In(time.Local).Format("01-02 15:04")
				}
			} else if t, ok := todayLastFetched[f.ID]; ok && !t.IsZero() {
				lastUpdate = t.In(time.Local).Format("01-02 15:04")
			}

			items = append(items, FeedView{
				Feed:       f,
				TodayCount: todayCount[f.ID],
				LastUpdate: lastUpdate,
				NextUpdate: nextUpdate,
			})
		}
	}

	h.render(c, "layout.html", PageData{
		Title:  "RSS订阅",
		Active: "feeds",
		Feeds: FeedListView{
			Items:    items,
			Total:    total,
			TotalAll: totalAll,
			Page:     page,
			PageSize: pageSize,
			Pages:    pages,
			Query:    q,
		},
	})
}

func nextTickByMinuteInterval(now time.Time, intervalMin int) time.Time {
	if intervalMin <= 0 {
		intervalMin = 5
	}
	if intervalMin > 60 {
		intervalMin = 60
	}
	t := now.Truncate(time.Minute)
	m := t.Minute()
	nextMin := ((m / intervalMin) + 1) * intervalMin
	if nextMin >= 60 {
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location()).Add(time.Hour)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nextMin, 0, 0, t.Location())
}

func filterFeedsByQuery(feeds []model.Feed, q string) []model.Feed {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return feeds
	}
	var out []model.Feed
	for _, f := range feeds {
		if strings.Contains(strings.ToLower(f.Name), q) ||
			strings.Contains(strings.ToLower(f.URL), q) ||
			strings.Contains(strings.ToLower(f.Remark), q) ||
			strings.Contains(strings.ToLower(f.ID), q) {
			out = append(out, f)
		}
	}
	return out
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
		Title:  "AI配置",
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
	q := strings.TrimSpace(c.Query("q"))
	status := strings.TrimSpace(c.Query("status"))
	taskID := strings.TrimSpace(c.Query("task_id"))
	page := 1
	if p := strings.TrimSpace(c.Query("page")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	pageSize := 10

	logs, _ := h.logSvc.List(0)
	stats, _ := h.logSvc.GetStats()
	statsToday, _ := h.logSvc.GetStatsToday()
	tasks, _ := h.taskSvc.List()

	totalAll := len(logs)
	filtered := filterLogsByQuery(logs, q, status, taskID)
	total := len(filtered)
	pages := (total + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	items := []model.ExecutionLog{}
	if start < end {
		items = filtered[start:end]
	}

	h.render(c, "layout.html", PageData{
		Title:  "执行日志",
		Active: "logs",
		Logs: LogListView{
			Items:    items,
			Total:    total,
			TotalAll: totalAll,
			Page:     page,
			PageSize: pageSize,
			Pages:    pages,
			Query:    q,
			Status:   status,
			TaskID:   taskID,
		},
		Stats: map[string]interface{}{"all": stats, "today": statsToday},
		Tasks: tasks,
	})
}

func filterLogsByQuery(logs []model.ExecutionLog, q, status, taskID string) []model.ExecutionLog {
	q = strings.ToLower(strings.TrimSpace(q))
	status = strings.ToLower(strings.TrimSpace(status))
	taskID = strings.TrimSpace(taskID)

	var out []model.ExecutionLog
	for _, l := range logs {
		if taskID != "" && l.TaskID != taskID {
			continue
		}
		if status != "" && strings.ToLower(l.Status) != status {
			continue
		}
		if q == "" {
			out = append(out, l)
			continue
		}

		if strings.Contains(strings.ToLower(l.TaskID), q) ||
			strings.Contains(strings.ToLower(l.TaskName), q) ||
			strings.Contains(strings.ToLower(l.Status), q) ||
			strings.Contains(strings.ToLower(l.Details), q) ||
			strings.Contains(strings.ToLower(l.Error), q) {
			out = append(out, l)
		}
	}
	return out
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
