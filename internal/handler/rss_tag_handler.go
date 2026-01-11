// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// RSS 标签 API：用于给前端提供标签列表（data/rss-tag.toml）。
//
// @author Anner
// Created on 2026/1/11
package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/service"
)

// RSSTagHandler RSS 标签 API 处理器
type RSSTagHandler struct {
	cfgMgr *config.Manager
}

// NewRSSTagHandler 创建 RSS 标签 API 处理器
func NewRSSTagHandler(cfgMgr *config.Manager) *RSSTagHandler {
	return &RSSTagHandler{cfgMgr: cfgMgr}
}

// List 获取 RSS 标签列表（支持 q 模糊搜索）
func (h *RSSTagHandler) List(c *gin.Context) {
	cfg, err := h.cfgMgr.LoadRSSTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	tags := service.NormalizeTags(cfg.Tags)

	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if q != "" {
		var filtered []string
		for _, t := range tags {
			if strings.Contains(strings.ToLower(t), q) {
				filtered = append(filtered, t)
			}
		}
		tags = filtered
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
