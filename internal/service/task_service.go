package service

import (
	"sync"

	"github.com/google/uuid"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// TaskService 任务服务
type TaskService struct {
	cfgMgr *config.Manager
	mu     sync.RWMutex
}

// NewTaskService 创建任务服务
func NewTaskService(cfgMgr *config.Manager) *TaskService {
	return &TaskService{cfgMgr: cfgMgr}
}

// List 获取所有任务
func (s *TaskService) List() ([]model.Task, error) {
	cfg, err := s.cfgMgr.LoadTasks()
	if err != nil {
		return nil, err
	}
	return cfg.Tasks, nil
}

// Get 获取单个任务
func (s *TaskService) Get(id string) (*model.Task, error) {
	cfg, err := s.cfgMgr.LoadTasks()
	if err != nil {
		return nil, err
	}
	for _, t := range cfg.Tasks {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

// Create 创建任务
func (s *TaskService) Create(task *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadTasks()
	if err != nil {
		return err
	}

	if task.ID == "" {
		task.ID = uuid.New().String()[:8]
	}
	cfg.Tasks = append(cfg.Tasks, *task)
	return s.cfgMgr.SaveTasks(cfg)
}

// Update 更新任务
func (s *TaskService) Update(task *model.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadTasks()
	if err != nil {
		return err
	}

	for i, t := range cfg.Tasks {
		if t.ID == task.ID {
			cfg.Tasks[i] = *task
			return s.cfgMgr.SaveTasks(cfg)
		}
	}
	return nil
}

// Delete 删除任务
func (s *TaskService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadTasks()
	if err != nil {
		return err
	}

	for i, t := range cfg.Tasks {
		if t.ID == id {
			cfg.Tasks = append(cfg.Tasks[:i], cfg.Tasks[i+1:]...)
			return s.cfgMgr.SaveTasks(cfg)
		}
	}
	return nil
}
