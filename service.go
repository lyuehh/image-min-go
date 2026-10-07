package main

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/egoist/mygo"
)

type ImageService struct{}

type ImageInfo struct {
	Path  string `json:"path"`
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	Error string `json:"error,omitempty"`
}

func (ImageService) PickImages(ctx context.Context) ([]string, error) {
	return mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:   mygo.CallerWindow(ctx),
		Title:    "选择要压缩的图片",
		Filters:  []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg"}}},
		Multiple: true,
	})
}

func (ImageService) TakePendingFiles() []string {
	inputMu.Lock()
	defer inputMu.Unlock()
	paths := pendingFiles
	pendingFiles = nil
	return paths
}

func (ImageService) Inspect(paths []string) []ImageInfo {
	files := expandPaths(paths)
	result := make([]ImageInfo, 0, len(files))
	for _, path := range files {
		item := ImageInfo{Path: path, Name: filepath.Base(path)}
		info, err := os.Stat(path)
		if err != nil {
			item.Error = err.Error()
		} else if _, ok := supportedFormat(path); !ok {
			item.Error = "仅支持 PNG 和 JPEG"
		} else {
			item.Size = info.Size()
		}
		result = append(result, item)
	}
	return result
}

func (ImageService) Compress(paths []string, options CompressionOptions) []CompressionResult {
	results := make([]CompressionResult, len(paths))
	for i, path := range paths {
		results[i] = compressImage(path, options)
	}
	return results
}

func (ImageService) Reveal(path string) error {
	if path == "" {
		return nil
	}
	mygo.Shell.ShowItemInFolder(path)
	return nil
}

func expandPaths(paths []string) []string {
	seen := make(map[string]bool)
	var files []string
	add := func(path string) {
		path, err := filepath.Abs(path)
		if err == nil && !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			add(path)
			continue
		}
		_ = filepath.WalkDir(path, func(child string, entry os.DirEntry, walkErr error) error {
			if walkErr == nil && !entry.IsDir() {
				if _, ok := supportedFormat(child); ok && !strings.Contains(child, string(filepath.Separator)+".") {
					add(child)
				}
			}
			return nil
		})
	}
	sort.Strings(files)
	return files
}
