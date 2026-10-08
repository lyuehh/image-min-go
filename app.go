package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// SVG 在包加载时解析一次，之后每一帧复用。MustParseSVG 在数据写错时会 panic，
// 适合这种由开发者写死、启动前就应该保证正确的资源；用户文件不能用 Must... 处理。
var (
	imageIcon = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="3" fill="none" stroke="currentColor" stroke-width="1.7"/><circle cx="16" cy="9" r="1.6" fill="currentColor"/><path d="m6 17 4-4 3 2.8 2.2-2.2L18 17" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	appIcon   = ui.MustParseSVG([]byte(`<svg viewBox="0 0 24 24"><rect x="2" y="2" width="20" height="20" rx="6" fill="currentColor"/><path d="m6.5 16.5 3.8-4 2.8 2.6 2.1-2.2 2.8 3.6" fill="none" stroke="white" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/><circle cx="16" cy="8.5" r="1.5" fill="white"/></svg>`))
)

// imageItem 把磁盘元数据和 UI 运行状态组合在一起。
//
// 单独写一行 ImageInfo 是“匿名嵌入”：imageItem 可以直接使用 Path、Name、Size 等
// 字段，就像它们声明在 imageItem 中一样。*CompressionResult 是指针；nil 表示还没有结果。
type imageItem struct {
	ImageInfo
	Result     *CompressionResult
	Processing bool
}

// imageApp 是单窗口应用的全部界面状态。
//
// UI 每次收到输入或状态更新时都会重新调用 view；因此状态不能放在 view 的局部变量里，
// 而应保存在这个长期存在的结构体中。list 保存虚拟列表的滚动位置。
type imageApp struct {
	items      []imageItem
	quality    float64
	busy       bool
	loading    int
	dialogOpen bool
	list       ui.ListState
	window     *mygo.Window
	// func(func()) 是“接收一个无参数函数、无返回值”的函数类型。生产环境把
	// Window.Update 方法赋给它，后台任务通过它把状态修改排回 UI 主线程。
	update func(func())
}

// newImageApp 是构造函数。Go 没有 constructor 关键字，惯例用 NewXxx/newXxx 函数。
// &imageApp{...} 返回结构体地址；小写函数名表示只在当前 package 内使用。
func newImageApp() *imageApp {
	return &imageApp{quality: defaultJPEGQuality}
}

// view 构建一帧原生界面。
//
// (a *imageApp) 是方法接收者：方法可通过 a 修改持久状态。c 只在当前帧有效，不能存到
// goroutine 中。Children 接收闭包，在父元素作用域中声明子元素，形成 UI 树。
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

// titleBar 绘制可拖动的自定义标题栏。
func (a *imageApp) titleBar(c *ui.Context) {
	// 一条赋值语句可同时声明多个变量。c.TitleBar() 返回系统交通灯/窗口按钮占用区域。
	t, bar := c.Theme(), c.TitleBar()
	// max 是 Go 内建函数；这里至少保留 56 DIP 高度，同时尊重系统要求的标题栏高度。
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

// dropZone 构建文件拖放区，并直接读取 MyGO 原生文件拖放事件。
func (a *imageApp) dropZone(c *ui.Context) {
	t := c.Theme()
	zone := ui.Column(c).Height(150).Shrink(0).Center().Gap(7).Radius(16).Background(t.Surface).Border(1, t.Border)
	if zone.FileDragOver() {
		zone.Background(t.Accent.Alpha(0.08)).Border(2, t.Accent)
	}
	if paths := zone.DroppedFiles(); len(paths) > 0 {
		// if 的初始化部分让 paths 只在这个 if 中存在。磁盘扫描会在 addPaths 中转到
		// 后台 goroutine，避免目录较大时阻塞界面。
		a.addPaths(paths)
	}
	zone.Children(func() {
		ui.Box(c).Size(42, 42).Radius(12).Center().Background(t.Accent.Alpha(0.12)).TextColor(t.Accent).Children(func() {
			ui.Icon(c, imageIcon).Size(23, 23)
		})
		ui.Text(c, "把图片拖到这里").FontSize(17).FontWeight(650)
		ui.Text(c, "支持 PNG、JPEG、GIF、WebP 和 SVG，也可以拖入整个文件夹").FontSize(12).TextColor(t.TextMuted)
		if ui.Button(c, "选择图片…").Disabled(a.busy || a.dialogOpen).Clicked() {
			a.pickImages()
		}
	})
}

// workspace 绘制工具栏、虚拟列表和底部汇总栏。
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
			// &a.quality 取得字段地址。滑块通过这个指针直接更新数值；步长 1 保证质量为整数。
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

// fileList 使用虚拟列表显示文件。虚拟列表只构建当前可见行，大批图片也不会为每一行
// 同时创建绘制对象。
func (a *imageApp) fileList(c *ui.Context) {
	t := c.Theme()
	remove := -1
	// any 是 interface{} 的别名，可容纳任意类型。用稳定的绝对路径作为 key，删除行时
	// MyGO 仍能把焦点和滚动状态对应到正确图片，而不是错误地按旧索引对应。
	a.list.Key = func(row int) any { return a.items[row].Path }
	a.list.Label = func(row int) string { return a.items[row].Name }
	ui.List(c, &a.list, len(a.items), func(i int) {
		// 取得切片元素的指针，避免复制结构体，也让状态读取都指向当前条目。
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
	// 不要在 List 的 row 回调正在按旧长度遍历时立刻改变切片，否则后续索引可能越界。
	// 先记下索引，列表构建结束后再删除。append(a[:i], a[i+1:]...) 是常见切片删除法。
	if remove >= 0 {
		a.items = append(a.items[:remove], a.items[remove+1:]...)
	}
}

// itemStatus 根据条目的状态选择“等待、处理中、失败、完成”四种展示。
func (a *imageApp) itemStatus(c *ui.Context, item *imageItem) {
	t := c.Theme()
	box := ui.Row(c).Width(130).Shrink(0).Justify(ui.End).Gap(5)
	box.Children(func() {
		// 不带表达式的 switch 等价于 switch true，按顺序选择第一个成立条件。
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
			// 整数相除会丢掉小数，所以先显式转为 float64 再计算百分比。
			percent := float64(saved) / float64(item.Size) * 100
			ui.Textf(c, "−%s · %.0f%%", formatSize(saved), percent).FontSize(11).FontWeight(650).TextColor(t.Success).FontFeatures("tnum")
		}
	})
}

// footer 显示批量统计和主要操作按钮。
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

// addPaths 异步扫描文件，完成后在 UI 主线程合并结果。
func (a *imageApp) addPaths(paths []string) {
	if len(paths) == 0 || a.busy || a.window == nil {
		return
	}
	a.loading++
	// go 关键字启动 goroutine。goroutine 很轻量，但不能直接修改 UI 状态；否则会与
	// 主线程同时读写 a.items/a.loading，产生 data race。
	go func() {
		items := inspectPaths(paths)
		// update 把闭包安排到 UI 主线程执行，同时请求重绘。
		a.update(func() {
			a.loading--
			a.merge(items)
		})
	}()
}

// merge 把扫描结果加入列表，并按绝对路径去重。
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

// pickImages 在 goroutine 中打开系统文件选择器。
// Dialog.Open 会一直等到用户选择或取消；放在后台才能保持 UI 可响应。
func (a *imageApp) pickImages() {
	if a.dialogOpen || a.busy || a.window == nil {
		return
	}
	a.dialogOpen = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   a.window,
			Title:    "选择要压缩的图片",
			Filters:  []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "svg"}}},
			Multiple: true,
		})
		// err == nil 只说明对话框本身没有失败；用户取消时 paths 通常为空，addPaths
		// 会安全地直接返回。
		a.update(func() {
			a.dialogOpen = false
			if err == nil {
				a.addPaths(paths)
			}
		})
	}()
}

// compress 启动一次批量压缩。它先在 UI 线程冻结可变操作，再由一个 goroutine 顺序
// 压缩文件；顺序处理可避免多张大图同时解码造成内存峰值。
func (a *imageApp) compress() {
	if !a.canCompress() || a.window == nil {
		return
	}
	a.busy = true
	// 滑块使用 float64，而 JPEG 编码器需要 int，因此显式转换。
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
			// busy 期间添加、清空、移除按钮都被禁用，所以 items 的索引保持稳定。
			result := compressImage(a.items[index].Path, CompressionOptions{JPEGQuality: quality})
			// result 在本次循环中是独立变量；取 &result 后 Go 会进行逃逸分析，把需要的
			// 值安全放到堆上。不要把旧版 Go 中复用的 range 变量地址用于这种场景。
			a.update(func() {
				a.items[index].Processing = false
				a.items[index].Result = &result
			})
		}
		a.update(func() { a.busy = false })
	}()
}

// canCompress 集中定义按钮可用条件，避免 UI 与实际操作使用两套不一致的判断。
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

// totals 使用命名返回值累计成功数量和节省字节数。
// 裸 return 会返回当前的 finished、saved；这里末尾显式写出名称，更易读。
func (a *imageApp) totals() (finished int, saved int64) {
	for _, item := range a.items {
		if item.Result != nil && item.Result.Error == "" {
			finished++
			saved += max(int64(0), item.Size-item.Result.OutputSize)
		}
	}
	return finished, saved
}

// formatSize 把字节数格式化为易读文本，例如 1536 -> "1.5 KB"。
func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes) / 1024
	// []string{...} 是切片字面量；range 每轮给出索引和值，这里用 _ 忽略索引。
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
