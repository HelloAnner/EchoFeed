// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0
// 备份服务：按配置将 data/ 下的全部文件进行打包备份（包含 rss、logs 等目录）。
//
// @author Anner
// Created on 2026/1/11
package service

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/echofeed/echofeed/internal/config"
	"github.com/echofeed/echofeed/internal/model"
)

// BackupService 备份服务
type BackupService struct {
	cfgMgr *config.Manager
}

// NewBackupService 创建备份服务
func NewBackupService(cfgMgr *config.Manager) *BackupService {
	return &BackupService{cfgMgr: cfgMgr}
}

// RunOnce 执行一次备份
func (s *BackupService) RunOnce(settings model.BackupSettings) error {
	if !settings.Enabled {
		return nil
	}

	now := time.Now().In(time.Local)
	bakDir := siblingDir(s.cfgMgr.DataDir, "bak")
	if err := os.MkdirAll(bakDir, 0755); err != nil {
		return err
	}

	name := now.Format("2006-01-02_15-04") + ".zip"
	zipPath := filepath.Join(bakDir, name)

	files, err := s.collectBackupFiles()
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no_files_to_backup")
	}

	if err := writeZip(zipPath, s.cfgMgr.DataDir, files); err != nil {
		return err
	}

	if settings.Retain > 0 {
		if err := cleanupOldBackups(bakDir, settings.Retain); err != nil {
			log.Warn().Err(err).Str("bak_dir", bakDir).Msg("Backup retention cleanup failed")
		}
	}

	log.Info().Str("zip", zipPath).Int("files", len(files)).Msg("Backup finished")
	return nil
}

func (s *BackupService) collectBackupFiles() ([]string, error) {
	var files []string
	dataDir := s.cfgMgr.DataDir

	// 备份 data/ 下全部文件（含 rss、logs、state 等）
	err := filepath.WalkDir(dataDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

func siblingDir(dataDir, name string) string {
	parent := filepath.Dir(dataDir)
	return filepath.Join(parent, name)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func writeZip(zipPath string, dataDir string, absFiles []string) error {
	tmpPath := zipPath + ".tmp"
	_ = os.Remove(tmpPath)

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()

	for _, abs := range absFiles {
		rel, err := filepath.Rel(filepath.Dir(dataDir), abs)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if err := addFileToZip(zw, abs, rel); err != nil {
			return err
		}
	}

	if err := zw.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, zipPath)
}

func addFileToZip(zw *zip.Writer, absPath, nameInZip string) error {
	src, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	h, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	h.Name = nameInZip
	h.Method = zip.Deflate

	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}

func cleanupOldBackups(bakDir string, retain int) error {
	if retain <= 0 {
		return nil
	}

	entries, err := os.ReadDir(bakDir)
	if err != nil {
		return err
	}

	var zips []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".zip") {
			zips = append(zips, filepath.Join(bakDir, name))
		}
	}
	sort.Strings(zips)

	if len(zips) <= retain {
		return nil
	}
	toDelete := zips[:len(zips)-retain]
	for _, p := range toDelete {
		_ = os.Remove(p)
	}
	return nil
}
