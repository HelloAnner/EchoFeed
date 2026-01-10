// 文本清洗工具：用于清洗 AI 输出的概述文本，避免带标签、链接或省略号影响展示。
//
// @author Anner
// Created on 2026/1/10
package service

import (
	"regexp"
	"strings"
)

var (
	summaryPrefixRe = regexp.MustCompile(`^\s*(中文概述|中文摘要|摘要|概述)(（[^）]*）)?\s*[:：]\s*`)
	urlRe           = regexp.MustCompile(`https?://\\S+`)
)

func sanitizeSummaryText(summary string) string {
	s := strings.TrimSpace(summary)
	if s == "" {
		return ""
	}

	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	s = summaryPrefixRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(urlRe.ReplaceAllString(s, ""))
	s = strings.TrimSpace(strings.TrimPrefix(s, "-"))

	for _, suffix := range []string{"...", "…", "……", "。。。", "。。。。"} {
		for strings.HasSuffix(s, suffix) {
			s = strings.TrimSpace(strings.TrimSuffix(s, suffix))
		}
	}
	return s
}
