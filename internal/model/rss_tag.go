// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// RSS 标签配置：用于维护系统内的全部 RSS 标签列表，便于前端模糊搜索与新增。
//
// @author Anner
// Created on 2026/1/10
package model

// RSSTagConfig RSS 标签配置文件结构（data/rss-tag.toml）
type RSSTagConfig struct {
	Tags []string `toml:"tags"`
}
