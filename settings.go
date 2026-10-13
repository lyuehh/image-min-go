package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// 语言代码。const 分组声明常量；这些值也会被写进设置文件，改名会让旧文件失效。
const (
	langChinese = "zh"
	langEnglish = "en"
)

// 有损 JPEG 质量的取值范围与默认值。无损模式忽略它，固定以质量 100 重新编码。
const (
	minLossyQuality     = 40
	maxLossyQuality     = 100
	defaultLossyQuality = defaultJPEGQuality
	// losslessJPEGQuality 是无损模式下 JPEG 的质量。JPEG 本身是有损格式，这里以最高
	// 质量 100 重新编码，尽量减少二次压缩带来的画质损失；finalizeOutput 仍保证输出不会变大。
	losslessJPEGQuality = 100
)

// settingsFileName 是存放在 PathUserData 下的设置文件名。
const settingsFileName = "settings.json"

// Settings 是会被持久化到磁盘的用户偏好。
//
// json 标签决定文件里的字段名。AllowLossy 关闭时进行无损压缩；开启时 JPEG 按
// LossyQuality 重新编码。零值（AllowLossy=false）正好对应“默认无损”，符合需求。
type Settings struct {
	Language     string `json:"language"`
	AllowLossy   bool   `json:"allowLossy"`
	LossyQuality int    `json:"lossyQuality"`
}

// defaultSettings 返回首次启动或读档失败时使用的默认设置：中文、无损、默认质量。
func defaultSettings() Settings {
	return Settings{
		Language:     langChinese,
		AllowLossy:   false,
		LossyQuality: defaultLossyQuality,
	}
}

// normalize 把可能来自旧文件或损坏文件的字段修正到合法范围，避免界面出现非法值。
func (s *Settings) normalize() {
	if s.Language != langEnglish {
		s.Language = langChinese
	}
	if s.LossyQuality < minLossyQuality {
		s.LossyQuality = minLossyQuality
	}
	if s.LossyQuality > maxLossyQuality {
		s.LossyQuality = maxLossyQuality
	}
}

// jpegQuality 根据“是否允许有损”决定实际传给编码器的 JPEG 质量。
// 无损模式返回最高质量 100；有损模式返回用户设置的质量。
func (s Settings) jpegQuality() int {
	if !s.AllowLossy {
		return losslessJPEGQuality
	}
	return s.LossyQuality
}

// settingsPath 返回设置文件的完整路径（PathUserData/settings.json）。
func settingsPath() (string, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settingsFileName), nil
}

// loadSettings 读取磁盘上的设置；文件不存在或损坏时都安全回退到默认值。
func loadSettings() Settings {
	s := defaultSettings()
	path, err := settingsPath()
	if err != nil {
		return s
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || err != nil {
		return s
	}
	// 损坏的文件被忽略并沿用默认值，下一次保存会覆盖它。
	if json.Unmarshal(data, &s) != nil {
		return defaultSettings()
	}
	s.normalize()
	return s
}

// saveSettings 把设置原子地写回磁盘：先写 .tmp 再重命名，避免写一半的文件。
// 返回 error 供调用方决定是否提示；界面层目前尽力保存，失败不阻断操作。
func saveSettings(s Settings) error {
	s.normalize()
	path, err := settingsPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path+".tmp", data, 0o644); err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
