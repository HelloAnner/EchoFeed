package service

import (
	"sync"

	"github.com/google/uuid"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// BotService AI机器人服务
type BotService struct {
	cfgMgr *config.Manager
	mu     sync.RWMutex
}

// NewBotService 创建AI机器人服务
func NewBotService(cfgMgr *config.Manager) *BotService {
	return &BotService{cfgMgr: cfgMgr}
}

// List 获取所有机器人
func (s *BotService) List() ([]model.Bot, error) {
	cfg, err := s.cfgMgr.LoadBots()
	if err != nil {
		return nil, err
	}
	return cfg.Bots, nil
}

// Get 获取单个机器人
func (s *BotService) Get(id string) (*model.Bot, error) {
	cfg, err := s.cfgMgr.LoadBots()
	if err != nil {
		return nil, err
	}
	for _, b := range cfg.Bots {
		if b.ID == id {
			return &b, nil
		}
	}
	return nil, nil
}

// Create 创建机器人
func (s *BotService) Create(bot *model.Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadBots()
	if err != nil {
		return err
	}

	if bot.ID == "" {
		bot.ID = uuid.New().String()[:8]
	}
	if bot.Config == nil {
		bot.Config = make(map[string]string)
	}
	cfg.Bots = append(cfg.Bots, *bot)
	return s.cfgMgr.SaveBots(cfg)
}

// Update 更新机器人
func (s *BotService) Update(bot *model.Bot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadBots()
	if err != nil {
		return err
	}

	for i, b := range cfg.Bots {
		if b.ID == bot.ID {
			cfg.Bots[i] = *bot
			return s.cfgMgr.SaveBots(cfg)
		}
	}
	return nil
}

// Delete 删除机器人
func (s *BotService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadBots()
	if err != nil {
		return err
	}

	for i, b := range cfg.Bots {
		if b.ID == id {
			cfg.Bots = append(cfg.Bots[:i], cfg.Bots[i+1:]...)
			return s.cfgMgr.SaveBots(cfg)
		}
	}
	return nil
}
