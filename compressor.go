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

// defaultJPEGQuality 是用户没有指定质量时采用的默认值。
//
// const 表示编译期常量，它不能在程序运行时被重新赋值。JPEG 质量的合法范围是
// 1～100；数值越大通常画质越好、文件也越大。82 是体积和观感之间的折中值。
const defaultJPEGQuality = 82

// CompressionOptions 汇总一次压缩需要的可调参数。
//
// type ... struct 声明结构体类型。字段名首字母大写表示它对其他包可见（exported）。
// 反引号里的 json 标签说明序列化时使用 jpegQuality 这个名字；当前原生 UI 直接调用
// Go 方法并不依赖它，但保留标签可以让这个数据结构以后用于配置文件或 API。
type CompressionOptions struct {
	JPEGQuality int `json:"jpegQuality"`
}

// CompressionResult 是单张图片的处理结果。
//
// int64 用于文件大小，因为普通 int 的位数与平台有关。omitempty 表示 JSON 序列化时
// 可以省略零值字段。这里用 Error 字符串而不是直接返回 error，是为了让批量任务中
// 某一张失败时，其他图片仍能继续处理并各自展示结果。
type CompressionResult struct {
	Path         string `json:"path"`
	OutputPath   string `json:"outputPath,omitempty"`
	OriginalSize int64  `json:"originalSize"`
	OutputSize   int64  `json:"outputSize,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	Error        string `json:"error,omitempty"`
}

// compressImage 压缩一张图片，并始终返回描述成功或失败的结果。
//
// 参数写成 options CompressionOptions 是“按值传递”：函数得到一份小结构体副本，
// 不会修改调用者的选项。返回值没有写名称，因此通过 return result 显式返回。
func compressImage(path string, options CompressionOptions) CompressionResult {
	// := 是短变量声明：Go 会根据右侧值推断变量类型。它只能在函数内部使用。
	result := CompressionResult{Path: path}
	// 多返回值是 Go 的常见写法。os.Stat 同时返回文件信息和 error。
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
	// Go 中 int 的零值是 0。用 0 表示“未设置”，再替换为默认质量。
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
	// defer 会把 Close 安排在当前函数返回前执行。不论下面从哪个错误分支 return，
	// 文件都会关闭。这里忽略 Close 的错误，因为我们只读取该文件。
	defer file.Close()

	// image.Image 是接口：JPEG 和 PNG 解码出的不同具体图片类型都能赋给它。
	var img image.Image
	if format == "jpeg" {
		// 手机照片经常用 EXIF Orientation 表示方向。AutoOrientation 会先把方向真正
		// 应用到像素，否则重新编码、去掉 EXIF 后图片可能横过来或倒过来。
		img, err = imaging.Decode(file, imaging.AutoOrientation(true))
	} else {
		img, err = png.Decode(file)
	}
	if err != nil {
		result.Error = "无法解码图片: " + err.Error()
		return result
	}
	bounds := img.Bounds()
	// Go 支持并行赋值：先计算右边，再一起写入左边。
	result.Width, result.Height = bounds.Dx(), bounds.Dy()

	// bytes.Buffer 实现 io.Writer，把编码结果先放在内存中。这样可在落盘前比较大小，
	// 避免“压缩”后反而生成更大的文件。注意：超大图片会因此占用额外内存。
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
		// encoded.Len() 返回 int；文件大小是 int64，所以这里要显式转换类型。
		err = writeAtomic(outputPath, bytes.NewReader(encoded.Bytes()), info.Mode().Perm())
		result.OutputSize = int64(encoded.Len())
	} else {
		// 解码已经把 file 的读取位置移动到了后面。Seek(0, io.SeekStart) 把它重置到
		// 文件开头，之后才能完整复制原图。原图更小时直接复制，保证输出不会变大。
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

// supportedFormat 根据扩展名判断当前 MVP 支持的编码格式。
//
// 返回 (string, bool) 是 Go 常见的“值, 是否存在”模式。调用方应先检查 bool，
// 不能只使用字符串。ToLower 让 .JPG 这类大写扩展名也能识别。
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

// nextOutputPath 选择一个未占用的非破坏性输出名。
//
// 已有 photo-min.jpg 时继续尝试 photo-min-2.jpg、photo-min-3.jpg……。
// for n := 2; ; n++ 是没有条件的循环；只有 return 才会离开它。
func nextOutputPath(path string) (string, error) {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	candidate := base + "-min" + ext
	for n := 2; ; n++ {
		if _, err := os.Stat(candidate); err != nil {
			// 只有“文件不存在”说明名字可用。权限错误等其他问题必须返回，不能也当成
			// 不存在，否则后续写入会给出更难理解的错误，甚至形成无限循环。
			if errors.Is(err, os.ErrNotExist) {
				return candidate, nil
			}
			return "", err
		}
		candidate = fmt.Sprintf("%s-min-%d%s", base, n, ext)
	}
}

// writeAtomic 把 src 的数据安全地写到 path，并保留原文件权限。
//
// 命名返回值 (err error) 让 defer 中可以看到函数最终是否失败。流程是：在目标目录
// 写临时文件 -> 刷到磁盘 -> 关闭 -> 建立硬链接。目标文件绝不会只写了一半。
// 注意：临时文件必须与目标位于同一文件系统，硬链接才能成功。项目主要面向 macOS；
// 如果以后移植到不支持硬链接的文件系统，需要实现对应平台的“不覆盖原子重命名”。
func writeAtomic(path string, src io.Reader, mode os.FileMode) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".image-min-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	// func() { ... }() 是立即调用的匿名函数；前面的 defer 让它延迟到本函数结束。
	// 无论成功失败都删除临时名字。成功时正式路径已经是同一文件的另一个硬链接。
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if _, err = io.Copy(tmp, src); err != nil {
		return err
	}
	if err = tmp.Chmod(mode); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		// Sync 要求操作系统把缓冲数据提交给存储设备，降低断电时留下空文件的风险。
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	// os.Link 以原子方式创建正式路径；目标若已存在会失败，不会像 Unix 上的 Rename
	// 那样覆盖它。这也封住“检查文件名”和“写出文件”之间被其他进程抢占的竞态窗口。
	return os.Link(tmpPath, path)
}
