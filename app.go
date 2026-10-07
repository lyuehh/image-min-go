package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	imageIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.7"/><circle cx="16" cy="9" r="1.6" fill="currentColor"/><path d="m6 17 4-4 3 2.8 2.2-2.2L18 17" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	appIcon   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24"><rect x="2" y="2" width="20" height="20" rx="6" fill="currentColor"/><path d="m6.5 16.5 3.8-4 2.8 2.6 2.1-2.2 2.8 3.6" fill="none" stroke="white" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/><circle cx="16" cy="8.5" r="1.5" fill="white"/></svg>`))
)

type imageItem struct {
	ImageInfo
	Result     *CompressionResult
	Processing bool
}

type imageApp struct {
	items      []imageItem
	quality    float64
	busy       bool
	loading    int
	dialogOpen bool
	list       ui.ListState
	window     *mygo.Window
	update     func(func())
}

func newImageApp() *imageApp {
	return &imageApp{quality: defaultJPEGQuality}
}

func (a *imageApp) view(c *ui.Context) {
	t := c.Theme()
	root := ui.Column(c).Fill().Background(t.Background)
	root.Children(func() {
		a.titleBar(c)
		ui.Column(c).Grow(1).MinHeight(0).Padding(18, 22, 22).Gap(14).Children(func() {
			a.dropZone(c)
			a.workspace(c)
		})
	})
}

func (a *imageApp) titleBar(c *ui.Context) {
	t, bar := c.Theme(), c.TitleBar()
	h := max(bar.Height, float32(56))
	ui.Row(c).Height(h).Shrink(0).Padding(0, max(bar.Right, 16), 0, max(bar.Left, 16)).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).DragWindow().Children(func() {
		ui.Spacer(c)
		ui.Row(c).Gap(8).Children(func() {
			ui.Icon(c, appIcon).Size(24, 24).TextColor(t.Accent)
			ui.Text(c, "Image Min").FontSize(15).FontWeight(650)
		})
		ui.Spacer(c)
	})
}

func (a *imageApp) dropZone(c *ui.Context) {
	t := c.Theme()
	zone := ui.Column(c).Height(150).Shrink(0).Center().Gap(7).Radius(16).Background(t.Surface).Border(1, t.Border)
	if zone.FileDragOver() {
		zone.Background(t.Accent.Alpha(0.08)).Border(2, t.Accent)
	}
	if paths := zone.DroppedFiles(); len(paths) > 0 {
		a.addPaths(paths)
	}
	zone.Children(func() {
		ui.Box(c).Size(42, 42).Radius(12).Center().Background(t.Accent.Alpha(0.12)).TextColor(t.Accent).Children(func() {
			ui.Icon(c, imageIcon).Size(23, 23)
		})
		ui.Text(c, "把图片拖到这里").FontSize(17).FontWeight(650)
		ui.Text(c, "支持 PNG 和 JPEG，也可以拖入整个文件夹").FontSize(12).TextColor(t.TextMuted)
		if ui.Button(c, "选择图片…").Disabled(a.busy || a.dialogOpen).Clicked() {
			a.pickImages()
		}
	})
}

func (a *imageApp) workspace(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinHeight(0).Radius(14).Background(t.Surface).Border(1, t.Border).Clip().Children(func() {
		ui.Row(c).Height(54).Shrink(0).PaddingX(14).Gap(10).Children(func() {
			count := "还没有图片"
			if len(a.items) > 0 {
				count = fmt.Sprintf("%d 张图片", len(a.items))
			}
			if a.loading > 0 {
				ui.Spinner(c).Label("正在读取图片")
			}
			ui.Text(c, count).FontSize(12).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, "JPEG 质量").FontSize(12).TextColor(t.TextMuted)
			ui.StepSlider(c, &a.quality, 40, 100, 1).Width(105).Disabled(a.busy).Label("JPEG 质量")
			ui.Textf(c, "%.0f", a.quality).Width(25).FontSize(12).FontFeatures("tnum")
			if ui.Button(c, "清空").Disabled(len(a.items) == 0 || a.busy).Clicked() {
				a.items = nil
			}
		})
		ui.Divider(c)
		a.fileList(c)
		ui.Divider(c)
		a.footer(c)
	})
}

func (a *imageApp) fileList(c *ui.Context) {
	t := c.Theme()
	remove := -1
	a.list.Key = func(row int) any { return a.items[row].Path }
	a.list.Label = func(row int) string { return a.items[row].Name }
	ui.List(c, &a.list, len(a.items), func(i int) {
		item := &a.items[i]
		ui.Row(c).Key(item.Path).MinHeight(64).Padding(9, 13).Gap(11).Children(func() {
			ui.Box(c).Size(34, 34).Shrink(0).Radius(9).Center().Background(t.Accent.Alpha(0.1)).TextColor(t.Accent).Children(func() {
				ui.Icon(c, imageIcon).Size(18, 18)
			})
			ui.Column(c).Grow(1).MinWidth(90).Gap(2).Children(func() {
				ui.Text(c, item.Name).FontSize(13).FontWeight(600).SingleLine()
				ui.Text(c, filepath.Dir(item.Path)).FontSize(10).TextColor(t.TextMuted).SingleLine()
			})
			ui.Text(c, formatSize(item.Size)).Width(82).TextAlign(ui.End).FontSize(11).TextColor(t.TextMuted).FontFeatures("tnum")
			a.itemStatus(c, item)
			if item.Result != nil && item.Result.Error == "" {
				if ui.Button(c, "显示").Clicked() {
					mygo.Shell.ShowItemInFolder(item.Result.OutputPath)
				}
			} else if ui.Button(c, "移除").Disabled(a.busy).Clicked() {
				remove = i
			}
		})
	}).Grow(1).Children(func() {
		if len(a.items) == 0 {
			ui.Column(c).Grow(1).Center().Children(func() {
				ui.Text(c, "添加图片后将在这里显示").FontSize(12).TextColor(t.TextMuted)
			})
		}
	})
	if remove >= 0 {
		a.items = append(a.items[:remove], a.items[remove+1:]...)
	}
}

func (a *imageApp) itemStatus(c *ui.Context, item *imageItem) {
	t := c.Theme()
	box := ui.Row(c).Width(130).Shrink(0).Justify(ui.End).Gap(5)
	box.Children(func() {
		switch {
		case item.Error != "":
			ui.Text(c, item.Error).FontSize(11).TextColor(t.Danger).SingleLine().Tooltip(item.Error)
		case item.Processing:
			ui.Spinner(c).Label("处理中")
			ui.Text(c, "处理中").FontSize(11).TextColor(t.Accent)
		case item.Result == nil:
			ui.Text(c, "等待压缩").FontSize(11).TextColor(t.TextMuted)
		case item.Result.Error != "":
			ui.Text(c, item.Result.Error).FontSize(11).TextColor(t.Danger).SingleLine().Tooltip(item.Result.Error)
		default:
			saved := item.Size - item.Result.OutputSize
			if saved <= 0 {
				ui.Text(c, "已是最优").FontSize(11).TextColor(t.TextMuted)
				return
			}
			percent := float64(saved) / float64(item.Size) * 100
			ui.Textf(c, "−%s · %.0f%%", formatSize(saved), percent).FontSize(11).FontWeight(650).TextColor(t.Success).FontFeatures("tnum")
		}
	})
}

func (a *imageApp) footer(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(52).Shrink(0).PaddingX(14).Gap(12).Children(func() {
		finished, saved := a.totals()
		if finished > 0 {
			ui.Textf(c, "已完成 %d 张，共节省 %s", finished, formatSize(saved)).FontSize(12).TextColor(t.Success).Grow(1)
		} else {
			ui.Text(c, "输出保存在原文件旁，不会覆盖原图").FontSize(12).TextColor(t.TextMuted).Grow(1)
		}
		label := "开始压缩"
		if a.busy {
			label = "压缩中"
			ui.Spinner(c).Label("正在压缩")
		}
		if ui.PrimaryButton(c, label).Disabled(!a.canCompress()).Clicked() {
			a.compress()
		}
	})
}

func (a *imageApp) addPaths(paths []string) {
	if len(paths) == 0 || a.busy || a.window == nil {
		return
	}
	a.loading++
	go func() {
		items := inspectPaths(paths)
		a.update(func() {
			a.loading--
			a.merge(items)
		})
	}()
}

func (a *imageApp) merge(infos []ImageInfo) {
	seen := make(map[string]bool, len(a.items)+len(infos))
	for _, item := range a.items {
		seen[item.Path] = true
	}
	for _, info := range infos {
		if !seen[info.Path] {
			a.items = append(a.items, imageItem{ImageInfo: info})
			seen[info.Path] = true
		}
	}
}

func (a *imageApp) pickImages() {
	if a.dialogOpen || a.busy || a.window == nil {
		return
	}
	a.dialogOpen = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   a.window,
			Title:    "选择要压缩的图片",
			Filters:  []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg"}}},
			Multiple: true,
		})
		a.update(func() {
			a.dialogOpen = false
			if err == nil {
				a.addPaths(paths)
			}
		})
	}()
}

func (a *imageApp) compress() {
	if !a.canCompress() || a.window == nil {
		return
	}
	a.busy = true
	quality := int(a.quality)
	var work []int
	for i := range a.items {
		if a.items[i].Error == "" {
			a.items[i].Processing = true
			a.items[i].Result = nil
			work = append(work, i)
		}
	}
	go func() {
		for _, index := range work {
			result := compressImage(a.items[index].Path, CompressionOptions{JPEGQuality: quality})
			a.update(func() {
				a.items[index].Processing = false
				a.items[index].Result = &result
			})
		}
		a.update(func() { a.busy = false })
	}()
}

func (a *imageApp) canCompress() bool {
	if a.busy || a.loading > 0 {
		return false
	}
	for _, item := range a.items {
		if item.Error == "" {
			return true
		}
	}
	return false
}

func (a *imageApp) totals() (finished int, saved int64) {
	for _, item := range a.items {
		if item.Result != nil && item.Result.Error == "" {
			finished++
			saved += max(int64(0), item.Size-item.Result.OutputSize)
		}
	}
	return finished, saved
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes) / 1024
	for _, unit := range []string{"KB", "MB", "GB"} {
		if value < 1024 || unit == "GB" {
			decimals := 2
			if value >= 10 {
				decimals = 1
			}
			return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.*f", decimals, value), "0"), ".") + " " + unit
		}
		value /= 1024
	}
	return ""
}
