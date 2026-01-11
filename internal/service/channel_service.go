// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package service

import (
	"sync"

	"github.com/google/uuid"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// ChannelService 通知渠道服务
type ChannelService struct {
	cfgMgr *config.Manager
	mu     sync.RWMutex
}

// NewChannelService 创建通知渠道服务
func NewChannelService(cfgMgr *config.Manager) *ChannelService {
	return &ChannelService{cfgMgr: cfgMgr}
}

// List 获取所有渠道
func (s *ChannelService) List() ([]model.Channel, error) {
	cfg, err := s.cfgMgr.LoadChannels()
	if err != nil {
		return nil, err
	}
	return cfg.Channels, nil
}

// Get 获取单个渠道
func (s *ChannelService) Get(id string) (*model.Channel, error) {
	cfg, err := s.cfgMgr.LoadChannels()
	if err != nil {
		return nil, err
	}
	for _, c := range cfg.Channels {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

// Create 创建渠道
func (s *ChannelService) Create(channel *model.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadChannels()
	if err != nil {
		return err
	}

	if channel.ID == "" {
		channel.ID = uuid.New().String()[:8]
	}
	if channel.Config == nil {
		channel.Config = make(map[string]string)
	}
	cfg.Channels = append(cfg.Channels, *channel)
	return s.cfgMgr.SaveChannels(cfg)
}

// Update 更新渠道
func (s *ChannelService) Update(channel *model.Channel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadChannels()
	if err != nil {
		return err
	}

	for i, c := range cfg.Channels {
		if c.ID == channel.ID {
			cfg.Channels[i] = *channel
			return s.cfgMgr.SaveChannels(cfg)
		}
	}
	return nil
}

// Delete 删除渠道
func (s *ChannelService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.cfgMgr.LoadChannels()
	if err != nil {
		return err
	}

	for i, c := range cfg.Channels {
		if c.ID == id {
			cfg.Channels = append(cfg.Channels[:i], cfg.Channels[i+1:]...)
			return s.cfgMgr.SaveChannels(cfg)
		}
	}
	return nil
}
