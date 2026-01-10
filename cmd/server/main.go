package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/handler"
	"github.com/echofeed/echofeed/internal/middleware"
	"github.com/echofeed/echofeed/internal/scheduler"
	"github.com/echofeed/echofeed/internal/service"
)

func main() {
	// 初始化日志
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// 时区：默认上海时区（可通过环境变量 TZ 覆盖）
	setTimezone(getEnv("TZ", "Asia/Shanghai"))

	// 数据目录
	dataDir := getEnv("ECHOFEED_DATA_DIR", "./data")

	// 初始化配置管理器
	cfgMgr := config.NewManager(dataDir)
	if err := cfgMgr.EnsureDataDir(); err != nil {
		log.Fatal().Err(err).Msg("Failed to create data directory")
	}

	// 加载系统设置
	settings, err := cfgMgr.LoadSettings()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings")
	}

	// 初始化服务
	feedSvc := service.NewFeedService(cfgMgr)
	taskSvc := service.NewTaskService(cfgMgr)
	channelSvc := service.NewChannelService(cfgMgr)
	botSvc := service.NewBotService(cfgMgr)
	logSvc := service.NewLogService(cfgMgr)

	// 应用日志配置（level/rotation）
	service.ConfigureAppLogger(dataDir, settings.Log)

	// 初始化调度器
	sched := scheduler.NewScheduler(cfgMgr, feedSvc, taskSvc, botSvc, channelSvc, logSvc)
	sched.Start()
	defer sched.Stop()

	// 初始化Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.BasicAuthFromSettings(cfgMgr))

	// 静态资源
	r.Static("/static", "./web/static")

	// 初始化Handler
	pageHandler := handler.NewPageHandler(cfgMgr, feedSvc, taskSvc, channelSvc, botSvc, logSvc)
	feedHandler := handler.NewFeedHandler(feedSvc, sched)
	taskHandler := handler.NewTaskHandler(taskSvc, sched)
	channelHandler := handler.NewChannelHandler(channelSvc)
	botHandler := handler.NewBotHandler(botSvc)
	logHandler := handler.NewLogHandler(logSvc)
	settingsHandler := handler.NewSettingsHandler(cfgMgr, sched)

	// 页面路由
	r.GET("/", pageHandler.Index)
	r.GET("/tasks", pageHandler.Tasks)
	r.GET("/tasks/new", pageHandler.TaskNew)
	r.GET("/tasks/:id", pageHandler.TaskDetail)
	r.GET("/feeds", pageHandler.Feeds)
	r.GET("/feeds/new", pageHandler.FeedNew)
	r.GET("/channels", pageHandler.Channels)
	r.GET("/channels/new", pageHandler.ChannelNew)
	r.GET("/channels/:id", pageHandler.ChannelDetail)
	r.GET("/bots", pageHandler.Bots)
	r.GET("/bots/new", pageHandler.BotNew)
	r.GET("/bots/:id", pageHandler.BotDetail)
	r.GET("/logs", pageHandler.Logs)
	r.GET("/settings", pageHandler.Settings)

	// API路由
	api := r.Group("/api")
	{
		// Feed API
		api.GET("/feeds", feedHandler.List)
		api.POST("/feeds", feedHandler.Create)
		api.POST("/feeds/refresh_all", feedHandler.RefreshAll)
		api.GET("/feeds/:id", feedHandler.Get)
		api.PUT("/feeds/:id", feedHandler.Update)
		api.DELETE("/feeds/:id", feedHandler.Delete)
		api.POST("/feeds/:id/refresh", feedHandler.Refresh)

		// Task API
		api.GET("/tasks", taskHandler.List)
		api.POST("/tasks", taskHandler.Create)
		api.GET("/tasks/:id", taskHandler.Get)
		api.PUT("/tasks/:id", taskHandler.Update)
		api.DELETE("/tasks/:id", taskHandler.Delete)
		api.POST("/tasks/:id/run", taskHandler.Run)

		// Channel API
		api.GET("/channels", channelHandler.List)
		api.POST("/channels", channelHandler.Create)
		api.GET("/channels/:id", channelHandler.Get)
		api.PUT("/channels/:id", channelHandler.Update)
		api.DELETE("/channels/:id", channelHandler.Delete)
		api.POST("/channels/:id/test", channelHandler.Test)

		// Bot API
		api.GET("/bots", botHandler.List)
		api.POST("/bots", botHandler.Create)
		api.GET("/bots/:id", botHandler.Get)
		api.PUT("/bots/:id", botHandler.Update)
		api.DELETE("/bots/:id", botHandler.Delete)
		api.POST("/bots/:id/test", botHandler.Test)

		// Log API
		api.GET("/logs", logHandler.List)
		api.GET("/logs/stats", logHandler.GetStats)
		api.GET("/logs/stats/today", logHandler.GetStatsToday)
		api.POST("/logs/clear", logHandler.ClearAll)

		// Settings API
		api.GET("/settings", settingsHandler.Get)
		api.PUT("/settings", settingsHandler.Update)
	}

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", settings.Server.Host, settings.Server.Port)
	log.Info().Str("addr", addr).Msg("Starting EchoFeed server")
	if err := r.Run(addr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

func setTimezone(tz string) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Warn().Err(err).Str("tz", tz).Msg("Failed to load timezone, keep default")
		return
	}
	time.Local = loc
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
