package main

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var (
	state            = newImageApp()
	mainWindow       *mygo.Window
	windowMu         sync.Mutex
	inputMu          sync.Mutex
	pending          []string
	applicationReady atomic.Bool
)

func openWindow() {
	windowMu.Lock()
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
		Content:         ui.View(state.view),
	})
	state.window = mainWindow
	state.update = mainWindow.Update
}

func enqueueFiles(paths []string) {
	inputMu.Lock()
	pending = append(pending, paths...)
	inputMu.Unlock()
	if !applicationReady.Load() {
		return
	}
	openWindow()
	mainWindow.Update(func() { state.addPaths(takePending()) })
}

func takePending() []string {
	inputMu.Lock()
	defer inputMu.Unlock()
	paths := pending
	pending = nil
	return paths
}

func main() {
	app := mygo.App
	app.SetName("Image Min")
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
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
