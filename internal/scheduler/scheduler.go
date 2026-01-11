// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package scheduler

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"time"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
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
	logSvc     *service.LogService
	taskEngine *service.TaskEngine
	cleanupSvc *service.CleanupService
	backupSvc  *service.BackupService
	jobs       map[string]cron.EntryID
	rssFetchID cron.EntryID
	rssSpec    string
	backupID   cron.EntryID
	backupSpec string
}

// NewScheduler 创建调度器
func NewScheduler(
	cfgMgr *config.Manager,
	feedSvc *service.FeedService,
	taskSvc *service.TaskService,
	botSvc *service.BotService,
	channelSvc *service.ChannelService,
	logSvc *service.LogService,
) *Scheduler {
	taskEngine := service.NewTaskEngine(cfgMgr, feedSvc, taskSvc, botSvc, channelSvc, logSvc)
	cleanupSvc := service.NewCleanupService(cfgMgr, logSvc)
	backupSvc := service.NewBackupService(cfgMgr)

	return &Scheduler{
		cron:       cron.New(),
		cfgMgr:     cfgMgr,
		feedSvc:    feedSvc,
		taskSvc:    taskSvc,
		botSvc:     botSvc,
		channelSvc: channelSvc,
		logSvc:     logSvc,
		taskEngine: taskEngine,
		cleanupSvc: cleanupSvc,
		backupSvc:  backupSvc,
		jobs:       make(map[string]cron.EntryID),
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.ReloadRSSFetch()
	s.ReloadBackup()

	// 数据清理任务：每天凌晨 1 点执行，保留最近 7 天
	s.cron.AddFunc("0 1 * * *", func() {
		settings, err := s.cfgMgr.LoadSettings()
		if err != nil {
			log.Error().Err(err).Msg("Failed to load settings for cleanup")
			return
		}
		days := settings.Fetch.RetentionDays
		if days <= 0 {
			days = 7
		}

		log.Info().Int("retention_days", days).Msg("Cleanup job started")
		if err := s.cleanupSvc.CleanupBeforeToday(days); err != nil {
			log.Error().Err(err).Msg("Cleanup job failed")
			return
		}
		log.Info().Msg("Cleanup job finished")
	})

	// 加载任务调度
	s.ReloadTasks()

	s.cron.Start()
	log.Info().Msg("Scheduler started")
}

// ReloadRSSFetch 根据系统设置重新加载 RSS 拉取定时任务
func (s *Scheduler) ReloadRSSFetch() {
	if s.rssFetchID != 0 {
		s.cron.Remove(s.rssFetchID)
		s.rssFetchID = 0
	}

	interval := 5
	settings, err := s.cfgMgr.LoadSettings()
	if err == nil && settings != nil {
		if settings.Fetch.DefaultInterval > 0 {
			interval = settings.Fetch.DefaultInterval
		}
	}
	if interval < 1 {
		interval = 1
	}
	if interval > 60 {
		interval = 60
	}

	spec := fmt.Sprintf("*/%d * * * *", interval)
	entryID, err := s.cron.AddFunc(spec, func() {
		log.Info().Msg("Scheduled RSS fetch started")
		if err := s.feedSvc.FetchAll(); err != nil {
			log.Error().Err(err).Msg("Scheduled RSS fetch failed")
		}
	})
	if err != nil {
		log.Error().Err(err).Str("spec", spec).Msg("Failed to schedule RSS fetch")
		return
	}

	s.rssFetchID = entryID
	s.rssSpec = spec
	log.Info().Str("spec", spec).Msg("RSS fetch scheduled")
}

// ReloadBackup 根据系统设置重新加载备份定时任务
func (s *Scheduler) ReloadBackup() {
	if s.backupID != 0 {
		s.cron.Remove(s.backupID)
		s.backupID = 0
	}

	settings, err := s.cfgMgr.LoadSettings()
	if err != nil || settings == nil {
		log.Error().Err(err).Msg("Failed to load settings for backup")
		return
	}
	if !settings.Backup.Enabled {
		log.Info().Msg("Backup is disabled")
		return
	}

	spec := buildBackupSpec(settings.Backup)
	entryID, err := s.cron.AddFunc(spec, func() {
		log.Info().Str("spec", spec).Msg("Scheduled backup started")
		if err := s.backupSvc.RunOnce(settings.Backup); err != nil {
			log.Error().Err(err).Msg("Scheduled backup failed")
		}
	})
	if err != nil {
		log.Error().Err(err).Str("spec", spec).Msg("Failed to schedule backup")
		return
	}

	s.backupID = entryID
	s.backupSpec = spec
	log.Info().Str("spec", spec).Msg("Backup scheduled")
}

func buildBackupSpec(b model.BackupSettings) string {
	every := b.Every
	if every <= 0 {
		every = 1
	}
	if every > 30 {
		every = 30
	}

	h, m := parseHHMM(b.At, 3, 0)
	minField := strconv.Itoa(m)

	unit := strings.ToLower(strings.TrimSpace(b.Unit))
	switch unit {
	case "hour", "hours", "h":
		return fmt.Sprintf("%s */%d * * *", minField, every)
	default:
		return fmt.Sprintf("%s %d */%d * *", minField, h, every)
	}
}

func parseHHMM(s string, defH, defM int) (int, int) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return defH, defM
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil {
		return defH, defM
	}
	if h < 0 || h > 23 {
		h = defH
	}
	if m < 0 || m > 59 {
		m = defM
	}
	return h, m
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

// RunTaskForDate 执行任务(指定日期)
func (s *Scheduler) RunTaskForDate(taskID, date string) {
	if err := s.taskEngine.ExecuteTaskForDate(taskID, date); err != nil {
		log.Error().Err(err).Str("task_id", taskID).Str("date", date).Msg("Task execution failed")
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

// TriggerTaskRunForDate 手动触发任务执行(指定日期)
func (s *Scheduler) TriggerTaskRunForDate(taskID, date string) {
	if date == "" {
		go s.RunTask(taskID)
	} else {
		go s.RunTaskForDate(taskID, date)
	}
}

// GetNotifier 获取通知服务(用于测试)
func (s *Scheduler) GetNotifier() *service.NotifierService {
	return service.NewNotifierService()
}

// GetBackupNextRunAt 获取备份下次执行时间（用于页面展示/调试）
func (s *Scheduler) GetBackupNextRunAt() time.Time {
	if s.backupID == 0 {
		return time.Time{}
	}
	return s.cron.Entry(s.backupID).Next
}
