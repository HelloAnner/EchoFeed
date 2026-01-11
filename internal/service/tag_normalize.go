// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 标签清洗：用于 RSS 订阅标签的统一处理（去空格、去重、限制长度）。
//
// @author Anner
// Created on 2026/1/10
package service

import (
	"sort"
	"strings"
)

// NormalizeTags 标签清洗（去空格、去重、排序、限制长度）
func NormalizeTags(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]bool)
	var out []string
	for _, t := range in {
		t = strings.TrimSpace(t)
		t = strings.ReplaceAll(t, "\n", " ")
		t = strings.Join(strings.Fields(t), " ")
		if t == "" {
			continue
		}
		if len([]rune(t)) > 30 {
			r := []rune(t)
			t = string(r[:30])
		}
		if seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
