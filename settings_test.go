package main

import (
	"testing"

	"github.com/egoist/mygo"
)

// TestJPEGQualityPolicy 验证“无损默认、可选有损”的质量决策逻辑。
func TestJPEGQualityPolicy(t *testing.T) {
	lossless := Settings{AllowLossy: false, LossyQuality: 60}
	if got := lossless.jpegQuality(); got != losslessJPEGQuality {
		t.Fatalf("lossless quality = %d, want %d", got, losslessJPEGQuality)
	}
	lossy := Settings{AllowLossy: true, LossyQuality: 60}
	if got := lossy.jpegQuality(); got != 60 {
		t.Fatalf("lossy quality = %d, want 60", got)
	}
}

// TestSettingsNormalizeClamps 验证损坏/越界字段被修正到合法范围。
func TestSettingsNormalizeClamps(t *testing.T) {
	s := Settings{Language: "fr", LossyQuality: 5}
	s.normalize()
	if s.Language != langChinese {
		t.Fatalf("unknown language not reset: %q", s.Language)
	}
	if s.LossyQuality != minLossyQuality {
		t.Fatalf("low quality not clamped: %d", s.LossyQuality)
	}
	s = Settings{Language: langEnglish, LossyQuality: 999}
	s.normalize()
	if s.Language != langEnglish || s.LossyQuality != maxLossyQuality {
		t.Fatalf("normalize changed valid language or missed clamp: %#v", s)
	}
}

// TestSettingsSaveLoadRoundTrip 验证设置能写盘并原样读回。
// SetPath 把 PathUserData 指到临时目录，避免污染真实用户配置。
func TestSettingsSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, dir)

	want := Settings{Language: langEnglish, AllowLossy: true, LossyQuality: 70}
	if err := saveSettings(want); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}
	got := loadSettings()
	if got != want {
		t.Fatalf("loadSettings = %#v, want %#v", got, want)
	}
}

// TestLoadSettingsDefaultsWhenMissing 验证文件不存在时回退到默认设置。
func TestLoadSettingsDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, dir)

	if got := loadSettings(); got != defaultSettings() {
		t.Fatalf("loadSettings without file = %#v, want defaults %#v", got, defaultSettings())
	}
}
