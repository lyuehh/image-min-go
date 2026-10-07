package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNativeViewEmptyState(t *testing.T) {
	a := newImageApp()
	tester := ui.NewTester(a.view, 780, 640)

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
