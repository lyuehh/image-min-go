package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

const defaultJPEGQuality = 82

type CompressionOptions struct {
	JPEGQuality int `json:"jpegQuality"`
}

type CompressionResult struct {
	Path         string `json:"path"`
	OutputPath   string `json:"outputPath,omitempty"`
	OriginalSize int64  `json:"originalSize"`
	OutputSize   int64  `json:"outputSize,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Error        string `json:"error,omitempty"`
}

func compressImage(path string, options CompressionOptions) CompressionResult {
	result := CompressionResult{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.OriginalSize = info.Size()
	if !info.Mode().IsRegular() {
		result.Error = "不是普通文件"
		return result
	}

	format, ok := supportedFormat(path)
	if !ok {
		result.Error = "仅支持 PNG 和 JPEG"
		return result
	}

	quality := options.JPEGQuality
	if quality == 0 {
		quality = defaultJPEGQuality
	}
	if quality < 1 || quality > 100 {
		result.Error = "JPEG 质量必须在 1 到 100 之间"
		return result
	}

	file, err := os.Open(path)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer file.Close()

	var img image.Image
	if format == "jpeg" {
		img, err = imaging.Decode(file, imaging.AutoOrientation(true))
	} else {
		img, err = png.Decode(file)
	}
	if err != nil {
		result.Error = "无法解码图片: " + err.Error()
		return result
	}
	bounds := img.Bounds()
	result.Width, result.Height = bounds.Dx(), bounds.Dy()

	var encoded bytes.Buffer
	if format == "jpeg" {
		err = jpeg.Encode(&encoded, img, &jpeg.Options{Quality: quality})
	} else {
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(&encoded, img)
	}
	if err != nil {
		result.Error = "压缩失败: " + err.Error()
		return result
	}

	outputPath, err := nextOutputPath(path)
	if err != nil {
		result.Error = "创建输出路径失败: " + err.Error()
		return result
	}
	result.OutputPath = outputPath
	if int64(encoded.Len()) < info.Size() {
		err = writeAtomic(outputPath, bytes.NewReader(encoded.Bytes()), info.Mode().Perm())
		result.OutputSize = int64(encoded.Len())
	} else {
		if _, err = file.Seek(0, io.SeekStart); err == nil {
			err = writeAtomic(outputPath, file, info.Mode().Perm())
		}
		result.OutputSize = info.Size()
	}
	if err != nil {
		result.Error = "写入失败: " + err.Error()
		result.OutputPath = ""
		result.OutputSize = 0
	}
	return result
}

func supportedFormat(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "jpeg", true
	case ".png":
		return "png", true
	default:
		return "", false
	}
}

func nextOutputPath(path string) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	candidate := base + "-min" + ext
	for n := 2; ; n++ {
		if _, err := os.Stat(candidate); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return candidate, nil
			}
			return "", err
		}
		candidate = fmt.Sprintf("%s-min-%d%s", base, n, ext)
	}
}

func writeAtomic(path string, src io.Reader, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".image-min-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err = io.Copy(tmp, src); err != nil {
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
