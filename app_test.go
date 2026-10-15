package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestNativeViewEmptyState 用 MyGO 的内存渲染器验证初始界面。
// 测试函数必须以 Test 开头，并接收 *testing.T；go test 会自动发现它。
func TestNativeViewEmptyState(t *testing.T) {
	a := newImageApp()
	tester := ui.NewTester(a.view, 780, 640)

	// 表驱动思路：把多个期望文本放进切片，用同一段断言逐个验证。
	for _, text := range []string{"Image Min", "把图片拖到这里", "选择图片…", "还没有图片", "添加图片后将在这里显示", "开始压缩"} {
		if !tester.HasText(text) {
			t.Errorf("native view does not show %q; texts: %q", text, tester.Texts())
		}
	}
	tester.SetDark(true)
	if !tester.HasText("把图片拖到这里") {
		t.Error("native view disappeared in dark mode")
	}
}

// TestNativeViewEnglishLocale 验证切换语言后界面以英文渲染。
func TestNativeViewEnglishLocale(t *testing.T) {
	a := newImageApp()
	tester := ui.NewTester(a.view, 780, 640)
	tester.SetLocale(langEnglish)
	for _, text := range []string{"Drop images here", "Choose images\u2026", "No images yet", "Compress", "Settings"} {
		if !tester.HasText(text) {
			t.Errorf("english view does not show %q; texts: %q", text, tester.Texts())
		}
	}
}

// TestSettingsDialogShowsControls 验证点开设置后弹窗展示语言和有损压缩选项。
func TestSettingsDialogShowsControls(t *testing.T) {
	a := newImageApp()
	tester := ui.NewTester(a.view, 780, 640)
	if err := tester.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if !a.settingsOpen {
		t.Fatal("settings dialog did not open")
	}
	tester.Frame()
	for _, text := range []string{"设置", "语言", "允许有损压缩", "完成"} {
		if !tester.HasText(text) {
			t.Errorf("settings dialog missing %q; texts: %q", text, tester.Texts())
		}
	}
}

// TestNativeViewRemovesAndClearsItems 模拟真实点击，验证事件会修改持久状态。
func TestNativeViewRemovesAndClearsItems(t *testing.T) {
	a := newImageApp()
	a.merge([]ImageInfo{
		{Path: "/pictures/one.png", Name: "one.png", Size: 2048},
		{Path: "/pictures/two.jpg", Name: "two.jpg", Size: 4096},
	})
	tester := ui.NewTester(a.view, 780, 640)

	if !tester.HasText("2 张图片") || !tester.HasText("one.png") || !tester.HasText("two.jpg") {
		t.Fatalf("items are not shown: %q", tester.Texts())
	}
	// Click 返回 error，因为找不到对应控件时测试本身也应失败。
	if err := tester.Click("移除"); err != nil {
		t.Fatal(err)
	}
	if len(a.items) != 1 {
		t.Fatalf("remove left %d items, want 1", len(a.items))
	}
	if err := tester.Click("清空"); err != nil {
		t.Fatal(err)
	}
	if len(a.items) != 0 || !tester.HasText("还没有图片") {
		t.Fatalf("clear left items or missed empty state: %#v, %q", a.items, tester.Texts())
	}
}

// TestNativeViewShowsCompressionSummary 验证业务结果正确映射为界面文字。
func TestNativeViewShowsCompressionSummary(t *testing.T) {
	a := newImageApp()
	a.items = []imageItem{{
		ImageInfo: ImageInfo{Path: "/pictures/photo.jpg", Name: "photo.jpg", Size: 10 * 1024},
		Result:    &CompressionResult{OutputPath: "/pictures/photo-min.jpg", OutputSize: 6 * 1024},
	}}
	tester := ui.NewTester(a.view, 780, 640)

	if !tester.HasText("−4 KB · 40%") {
		t.Errorf("saving is not shown: %q", tester.Texts())
	}
	if !tester.HasText("已完成 1 张，共节省 4 KB") {
		t.Errorf("summary is not shown: %q", tester.Texts())
	}
}

// TestMergeDeduplicatesAndFormatSize 覆盖不依赖 UI 的小型纯逻辑。
func TestMergeDeduplicatesAndFormatSize(t *testing.T) {
	a := newImageApp()
	item := ImageInfo{Path: "/pictures/a.png", Name: "a.png", Size: 1536}
	a.merge([]ImageInfo{item, item})
	a.merge([]ImageInfo{item})
	if len(a.items) != 1 {
		t.Fatalf("merge produced %d items, want 1", len(a.items))
	}
	if got := formatSize(1536); got != "1.5 KB" {
		t.Fatalf("formatSize(1536) = %q", got)
	}
}

// TestSettingsDialogEnglishLocale 验证当应用语言为英文时，设置弹窗以英文渲染，
// 且语言分段控件正确落在 English 一侧。语言现由 MyGO 原生 i18n 托管，测试用
// Tester.SetLocale 模拟 mygo.App.SetLocale 后的效果。
func TestSettingsDialogEnglishLocale(t *testing.T) {
	a := newImageApp()
	tester := ui.NewTester(a.view, 780, 640)
	tester.SetLocale(langEnglish)
	if err := tester.Click("Settings"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.HasText("Allow lossy compression") {
		t.Fatalf("settings dialog not in english: %q", tester.Texts())
	}
}

// TestLanguageIndexCodeRoundTrip 验证语言标签与 Segmented 索引的双向映射。
func TestLanguageIndexCodeRoundTrip(t *testing.T) {
	if languageIndex(langChinese) != 0 || languageIndex(langEnglish) != 1 {
		t.Fatalf("languageIndex mapping wrong: zh=%d en=%d", languageIndex(langChinese), languageIndex(langEnglish))
	}
	// 未知/空语言回退到中文（索引 0）。
	if languageIndex("") != 0 || languageIndex("fr") != 0 {
		t.Fatalf("unknown locale should map to chinese index 0")
	}
	if languageCode(0) != langChinese || languageCode(1) != langEnglish {
		t.Fatalf("languageCode mapping wrong: 0=%q 1=%q", languageCode(0), languageCode(1))
	}
}
