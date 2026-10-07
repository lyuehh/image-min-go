package main

import (
	_ "embed"
	"log"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo"
)

//go:embed index.html
var page string

var (
	filesAdded  = mygo.NewEvent[[]string]("files-added")
	inputsReady = mygo.NewEvent[bool]("inputs-ready")
)

var (
	mainWindow   *mygo.Window
	windowMu     sync.Mutex
	inputMu      sync.Mutex
	pendingFiles []string
	appReady     atomic.Bool
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
		TitleBarHeight:  64,
		BackgroundColor: "light-dark(#f4f4f6, #18181b)",
	})
	mainWindow.Page().LoadHTML(page, "")
	mainWindow.OnFileDrop(func(event *mygo.FileDropEvent) {
		_ = filesAdded.Emit(mainWindow, event.Paths)
	})
}

func chooseFromMenu(window *mygo.Window) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:   window,
			Title:    "选择要压缩的图片",
			Filters:  []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg"}}},
			Multiple: true,
		})
		if err == nil && len(paths) > 0 {
			enqueueFiles(paths)
		}
	}()
}

func enqueueFiles(paths []string) {
	inputMu.Lock()
	pendingFiles = append(pendingFiles, paths...)
	inputMu.Unlock()
	if !appReady.Load() {
		return
	}
	openWindow()
	_ = inputsReady.Emit(mainWindow, true)
}

func main() {
	mygo.Bind(ImageService{})
	app := mygo.App
	app.SetName("Image Min")
	app.OnOpenFile(func(path string) { enqueueFiles([]string{path}) })
	app.WhenReady(func() {
		appReady.Store(true)
		app.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "添加图片…", Accelerator: "CmdOrCtrl+O", Click: func(_ *mygo.MenuItem, w *mygo.Window) { chooseFromMenu(w) }},
				mygo.Separator(),
				{Role: mygo.RoleClose},
			}},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleViewMenu},
			{Role: mygo.RoleWindowMenu},
		}))
		openWindow()
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
