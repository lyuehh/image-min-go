package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ImageInfo 是 UI 在压缩前显示的文件元数据。
// Error 非空表示该条目无法处理；保留条目可以让用户看到具体失败原因。
type ImageInfo struct {
	Path  string
	Name  string
	Size  int64
	Error string
}

// inspectPaths 展开文件/目录输入，并读取每个候选文件的基本信息。
// []string 表示 string 切片（可变长度序列），[]ImageInfo 是返回的图片信息切片。
func inspectPaths(paths []string) []ImageInfo {
	files := expandPaths(paths)
	// make([]T, length, capacity) 创建切片。这里长度为 0、容量预留为文件数，
	// append 时通常不必反复扩容复制。
	result := make([]ImageInfo, 0, len(files))
	// range 遍历切片；_ 丢弃当前索引，只保留 path 值。
	for _, path := range files {
		item := ImageInfo{Path: path, Name: filepath.Base(path)}
		info, err := os.Stat(path)
		if err != nil {
			item.Error = err.Error()
		} else if _, ok := supportedFormat(path); !ok {
			// if 初始化语句声明的 ok 只在这条 if/else 链中可见。
			item.Error = "仅支持 PNG 和 JPEG"
		} else {
			item.Size = info.Size()
		}
		result = append(result, item)
	}
	return result
}

// expandPaths 接受文件与目录的混合输入，递归找出支持的图片并去重。
func expandPaths(paths []string) []string {
	// map[string]bool 是键为路径、值为布尔值的哈希表，用于 O(1) 去重。
	seen := make(map[string]bool)
	// nil 切片可以直接 append，不需要手动初始化。
	var files []string
	// add := func(...) 声明闭包。闭包可以读取并修改外层的 seen 和 files。
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
		// WalkDir 会递归访问目录。回调返回 nil 表示继续；这里忽略单个无法访问的节点，
		// 让同一批中的其他图片仍可加入。隐藏目录（路径片段以 . 开头）会被跳过。
		_ = filepath.WalkDir(path, func(child string, entry os.DirEntry, walkErr error) error {
			if walkErr == nil && !entry.IsDir() {
				if _, ok := supportedFormat(child); ok && !strings.Contains(child, string(filepath.Separator)+".") {
					add(child)
				}
			}
			return nil
		})
	}
	// 固定排序让 UI 和测试结果稳定，不受文件系统枚举顺序影响。
	sort.Strings(files)
	return files
}
