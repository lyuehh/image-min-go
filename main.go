package main

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// 包级 var 保存应用生命周期状态。括号形式 var (...) 可以集中声明多个变量。
// 指针类型 *mygo.Window 表示“窗口对象的地址”；nil 表示窗口尚未创建。
var (
	state      = newImageApp()
	mainWindow *mygo.Window
	windowMu   sync.Mutex
	inputMu    sync.Mutex
	pending    []string
	// atomic.Bool 可在不同 goroutine 间安全读写，避免数据竞争。
	applicationReady atomic.Bool
)

// openWindow 创建主窗口，或把已经存在的窗口带回前台。
//
// Mutex（互斥锁）保证两个系统事件同时到来时不会创建两扇主窗口。
func openWindow() {
	windowMu.Lock()
	// defer Unlock 能确保今后即使函数中间新增 return，也不会忘记释放锁。
	defer windowMu.Unlock()
	if mainWindow != nil && !mainWindow.IsDestroyed() {
		mainWindow.Show()
		mainWindow.Focus()
		return
	}
	mainWindow = mygo.NewWindow(mygo.WindowOptions{
		Title:           "Image Min",
		Width:           780,
		Height:          640,
		MinWidth:        600,
		MinHeight:       460,
		StateKey:        "main",
		TitleBarStyle:   mygo.TitleBarHidden,
		TitleBarHeight:  56,
		BackgroundColor: "light-dark(#f4f4f6, #18181b)",
		// 方法值 state.view 的类型是 func(*ui.Context)，ui.View 把它变成 MyGO
		// 原生内容。这里没有 URL、HTML 或 WebView。
		Content: ui.View(state.view),
	})
	// Window.Update 能把后台 goroutine 的结果安全送回 UI 主线程。
	state.window = mainWindow
	state.update = mainWindow.Update
}

// enqueueFiles 接收 Finder“打开方式”等系统事件传来的路径。
// App ready 前先放进 pending；ready 后再交给界面，防止过早操作未创建的窗口。
func enqueueFiles(paths []string) {
	inputMu.Lock()
	// paths... 把切片展开为 append 的多个参数。
	pending = append(pending, paths...)
	inputMu.Unlock()
	if !applicationReady.Load() {
		return
	}
	openWindow()
	mainWindow.Update(func() { state.addPaths(takePending()) })
}

// takePending 在锁内取走全部待处理路径。
// 把 pending 设为 nil 会释放对底层数组的引用，之后可由垃圾回收器回收。
func takePending() []string {
	inputMu.Lock()
	defer inputMu.Unlock()
	paths := pending
	pending = nil
	return paths
}

// main 是 Go 可执行程序的入口。它只负责应用生命周期和菜单；具体 UI 在 app.go。
func main() {
	app := mygo.App
	app.SetName("Image Min")
	// func(path string) { ... } 是匿名函数（闭包）。[]string{path} 是含一个元素的
	// 切片字面量，用它复用批量入口 enqueueFiles。
	app.OnOpenFile(func(path string) { enqueueFiles([]string{path}) })
	app.WhenReady(func() {
		applicationReady.Store(true)
		app.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "添加图片…", Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { state.pickImages() }},
				mygo.Separator(),
				{Role: mygo.RoleClose},
			}},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleViewMenu},
			{Role: mygo.RoleWindowMenu},
		}))
		openWindow()
		state.addPaths(takePending())
	})
	app.OnWindowAllClosed(func() {})
	app.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			openWindow()
		}
	})
	// if err := ...; err != nil 把 err 的作用域限制在 if 内，是 Go 常见错误处理写法。
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
