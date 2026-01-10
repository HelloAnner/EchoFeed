package scheduler

import (
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/service"
)

// Scheduler 定时任务调度器
type Scheduler struct {
	cron       *cron.Cron
	cfgMgr     *config.Manager
	feedSvc    *service.FeedService
	taskSvc    *service.TaskService
	botSvc     *service.BotService
	channelSvc *service.ChannelService
	taskEngine *service.TaskEngine
	jobs       map[string]cron.EntryID
}

// NewScheduler 创建调度器
func NewScheduler(
	cfgMgr *config.Manager,
	feedSvc *service.FeedService,
	taskSvc *service.TaskService,
	botSvc *service.BotService,
	channelSvc *service.ChannelService,
) *Scheduler {
	taskEngine := service.NewTaskEngine(cfgMgr, feedSvc, taskSvc, botSvc, channelSvc)

	return &Scheduler{
		cron:       cron.New(),
		cfgMgr:     cfgMgr,
		feedSvc:    feedSvc,
		taskSvc:    taskSvc,
		botSvc:     botSvc,
		channelSvc: channelSvc,
		taskEngine: taskEngine,
		jobs:       make(map[string]cron.EntryID),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	// 添加RSS定时拉取任务(每5分钟)
	s.cron.AddFunc("*/5 * * * *", func() {
		log.Info().Msg("Scheduled RSS fetch started")
		if err := s.feedSvc.FetchAll(); err != nil {
			log.Error().Err(err).Msg("Scheduled RSS fetch failed")
		}
	})

	// 加载任务调度
	s.ReloadTasks()

	s.cron.Start()
	log.Info().Msg("Scheduler started")
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
	log.Info().Msg("Scheduler stopped")
}

// ReloadTasks 重新加载任务调度
func (s *Scheduler) ReloadTasks() {
	// 移除旧的任务
	for id, entryID := range s.jobs {
		s.cron.Remove(entryID)
		delete(s.jobs, id)
	}

	// 加载新的任务
	tasks, err := s.taskSvc.List()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load tasks")
		return
	}

	for _, task := range tasks {
		if !task.Enabled || task.Schedule == "" {
			continue
		}

		taskID := task.ID
		entryID, err := s.cron.AddFunc(task.Schedule, func() {
			s.RunTask(taskID)
		})
		if err != nil {
			log.Error().Err(err).Str("task", task.Name).Msg("Failed to schedule task")
			continue
		}
		s.jobs[taskID] = entryID
		log.Info().Str("task", task.Name).Str("schedule", task.Schedule).Msg("Task scheduled")
	}
}

// RunTask 执行任务
func (s *Scheduler) RunTask(taskID string) {
	if err := s.taskEngine.ExecuteTask(taskID); err != nil {
		log.Error().Err(err).Str("task_id", taskID).Msg("Task execution failed")
	}
}

// TriggerFeedRefresh 手动触发RSS刷新
func (s *Scheduler) TriggerFeedRefresh(feedID string) error {
	feed, err := s.feedSvc.Get(feedID)
	if err != nil {
		return err
	}
	if feed == nil {
		return nil
	}
	return s.feedSvc.Fetch(feed)
}

// TriggerTaskRun 手动触发任务执行
func (s *Scheduler) TriggerTaskRun(taskID string) {
	go s.RunTask(taskID)
}

// GetNotifier 获取通知服务(用于测试)
func (s *Scheduler) GetNotifier() *service.NotifierService {
	return service.NewNotifierService()
}
