// SPDX-License-Identifier: LicenseRef-PolyForm-Noncommercial-1.0.0

package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// saveToml 保存TOML文件
func saveToml(path string, v interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(v)
}
