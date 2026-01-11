// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 内置应用信息：版本与作者，从 info.json 读取（embed 到二进制）。
// @author Anner
// Created on 2026/1/10
package appinfo

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed info.json
var fs embed.FS

// Info 应用信息
type Info struct {
	Version string `json:"version"`
	Author  string `json:"author"`
}

var (
	once sync.Once
	info Info
)

// Get 获取内置应用信息
func Get() Info {
	once.Do(func() {
		b, err := fs.ReadFile("info.json")
		if err != nil {
			info = Info{Version: "unknown", Author: "unknown"}
			return
		}
		var v Info
		if err := json.Unmarshal(b, &v); err != nil {
			info = Info{Version: "unknown", Author: "unknown"}
			return
		}
		if v.Version == "" {
			v.Version = "unknown"
		}
		if v.Author == "" {
			v.Author = "unknown"
		}
		info = v
	})
	return info
}
