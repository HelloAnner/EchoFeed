// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 基于 settings.toml 的 BasicAuth 中间件
// @author Anner
// Created on 2026/1/10
package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/echofeed/echofeed/internal/config"
)

func BasicAuthFromSettings(cfgMgr *config.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfgMgr == nil {
			c.Next()
			return
		}
		settings, err := cfgMgr.LoadSettings()
		if err != nil || settings == nil || !settings.Auth.Enabled {
			c.Next()
			return
		}

		username := strings.TrimSpace(settings.Auth.Username)
		passwordHash := strings.TrimSpace(settings.Auth.PasswordHash)
		if username == "" || passwordHash == "" {
			abortUnauthorized(c)
			return
		}

		u, p, ok := c.Request.BasicAuth()
		if !ok {
			abortUnauthorized(c)
			return
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(p))
		if subtle.ConstantTimeCompare([]byte(u), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(encoded), []byte(passwordHash)) != 1 {
			abortUnauthorized(c)
			return
		}

		c.Next()
	}
}

func abortUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="EchoFeed"`)
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}
	c.AbortWithStatus(http.StatusUnauthorized)
}
