package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ImageInfo struct {
	Path  string
	Name  string
	Size  int64
	Error string
}

func inspectPaths(paths []string) []ImageInfo {
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
