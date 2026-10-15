package main

import (
	"os"
	"testing"

	"github.com/egoist/mygo"
)

// testUserDataDir 是 TestMain 为整个测试包设定的默认 PathUserData。
// 单独覆盖过 PathUserData 的测试在结束时要恢复到它，避免影响后续测试。
var testUserDataDir string

// TestMain 把 PathUserData 指向一个临时目录，确保任何会写 settings.json 的测试
// （以及点击设置控件触发的持久化）都不会污染真实用户配置。
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "image-min-test-userdata-")
	if err != nil {
		panic(err)
	}
	testUserDataDir = dir
	mygo.App.SetPath(mygo.PathUserData, dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

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
	s := Settings{LossyQuality: 5}
	s.normalize()
	if s.LossyQuality != minLossyQuality {
		t.Fatalf("low quality not clamped: %d", s.LossyQuality)
	}
	s = Settings{LossyQuality: 999}
	s.normalize()
	if s.LossyQuality != maxLossyQuality {
		t.Fatalf("high quality not clamped: %#v", s)
	}
	// 未知或空的目标格式回退到 AVIF。
	s = Settings{LossyQuality: 60, TargetFormat: "bogus"}
	s.normalize()
	if s.TargetFormat != formatAVIF {
		t.Fatalf("invalid target format not reset: %q", s.TargetFormat)
	}
}

// TestTargetFormatPolicy 验证“仅开启转换后才返回目标格式”的决策逻辑。
func TestTargetFormatPolicy(t *testing.T) {
	off := Settings{ConvertFormat: false, TargetFormat: formatJXL}
	if got := off.targetFormat(); got != "" {
		t.Fatalf("convert off should keep original format, got %q", got)
	}
	on := Settings{ConvertFormat: true, TargetFormat: formatJXL}
	if got := on.targetFormat(); got != formatJXL {
		t.Fatalf("convert on target = %q, want %q", got, formatJXL)
	}
}

// TestSettingsSaveLoadRoundTrip 验证设置能写盘并原样读回。
// 用独立子目录覆盖 PathUserData，与其它测试隔离。
func TestSettingsSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	mygo.App.SetPath(mygo.PathUserData, dir)
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, testUserDataDir) })

	want := Settings{AllowLossy: true, LossyQuality: 70, ConvertFormat: true, TargetFormat: formatJXL}
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
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, testUserDataDir) })

	if got := loadSettings(); got != defaultSettings() {
		t.Fatalf("loadSettings without file = %#v, want defaults %#v", got, defaultSettings())
	}
}
