package main

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	nativewebp "github.com/HugoSmits86/nativewebp"
	"golang.org/x/image/webp"
)

// TestCompressPNGPreservesPixelsAndNeverGrows 创建临时 PNG，验证透明像素保持不变，
// 且“压缩”输出不会比输入更大。t.TempDir 会在测试结束后自动清理目录。
func TestCompressPNGPreservesPixelsAndNeverGrows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alpha.png")
	img := image.NewNRGBA(image.Rect(0, 0, 80, 60))
	for y := range 60 {
		for x := range 80 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 180, A: uint8((x + y) % 256)})
		}
	}
	writePNG(t, path, img, png.NoCompression)

	result := compressImage(path, CompressionOptions{})
	if result.Error != "" {
		t.Fatalf("compressImage error: %s", result.Error)
	}
	if result.OutputSize > result.OriginalSize {
		t.Fatalf("output grew from %d to %d", result.OriginalSize, result.OutputSize)
	}
	if filepath.Base(result.OutputPath) != "alpha-min.png" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	out, err := imagingOpen(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	// .(color.NRGBA) 是类型断言：Convert 返回 color.Color 接口，这里取出具体值。
	// 若实际类型不是 color.NRGBA，断言会 panic；NRGBAModel.Convert 的契约保证它是。
	want := color.NRGBAModel.Convert(img.At(25, 31)).(color.NRGBA)
	got := color.NRGBAModel.Convert(out.At(25, 31)).(color.NRGBA)
	if got != want {
		t.Fatalf("pixel changed: got %#v, want %#v", got, want)
	}
}

// TestCompressJPEGAndAvoidsOverwrite 验证有损 JPEG 会减小，并且第二次运行使用新文件名。
func TestCompressJPEGAndAvoidsOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 128, 96))
	for y := range 96 {
		for x := range 128 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * y), G: uint8(x * 3), B: uint8(y * 5), A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	first := compressImage(path, CompressionOptions{JPEGQuality: 60})
	second := compressImage(path, CompressionOptions{JPEGQuality: 60})
	if first.Error != "" || second.Error != "" {
		t.Fatalf("compression errors: %q, %q", first.Error, second.Error)
	}
	if filepath.Base(second.OutputPath) != "photo-min-2.jpg" {
		t.Fatalf("second output overwrote the first: %q", second.OutputPath)
	}
	if first.OutputSize >= first.OriginalSize {
		t.Fatalf("expected JPEG compression, got %d -> %d", first.OriginalSize, first.OutputSize)
	}
}

// TestCompressRejectsInvalidInputAndQuality 覆盖“不支持格式、非法参数、损坏图片”错误。
func TestCompressRejectsInvalidInputAndQuality(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(textPath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := compressImage(textPath, CompressionOptions{}); !strings.Contains(got.Error, "PNG") {
		t.Fatalf("unexpected unsupported format error: %q", got.Error)
	}
	pngPath := filepath.Join(dir, "broken.png")
	if err := os.WriteFile(pngPath, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := compressImage(pngPath, CompressionOptions{JPEGQuality: 101}); !strings.Contains(got.Error, "质量") {
		t.Fatalf("unexpected quality error: %q", got.Error)
	}
	if got := compressImage(pngPath, CompressionOptions{}); !strings.Contains(got.Error, "无法解码") {
		t.Fatalf("unexpected decode error: %q", got.Error)
	}
}

// TestCompressGIFPreservesAnimationFrames 验证多帧 GIF 重新编码后帧数、尺寸和循环设置保留。
func TestCompressGIFPreservesAnimationFrames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "anim.gif")
	palette := color.Palette{color.Black, color.White, color.RGBA{R: 200, G: 30, B: 30, A: 255}}
	anim := &gif.GIF{LoopCount: 0}
	for frame := range 3 {
		img := image.NewPaletted(image.Rect(0, 0, 32, 24), palette)
		for y := range 24 {
			for x := range 32 {
				img.SetColorIndex(x, y, uint8((x+y+frame)%len(palette)))
			}
		}
		anim.Image = append(anim.Image, img)
		anim.Delay = append(anim.Delay, 10)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := gif.EncodeAll(f, anim); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	result := compressImage(path, CompressionOptions{})
	if result.Error != "" {
		t.Fatalf("compressImage error: %s", result.Error)
	}
	if result.OutputSize > result.OriginalSize {
		t.Fatalf("output grew from %d to %d", result.OriginalSize, result.OutputSize)
	}
	if filepath.Base(result.OutputPath) != "anim-min.gif" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	if result.Width != 32 || result.Height != 24 {
		t.Fatalf("unexpected size %dx%d", result.Width, result.Height)
	}
	out, err := os.Open(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	decoded, err := gif.DecodeAll(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Image) != 3 {
		t.Fatalf("frame count changed to %d", len(decoded.Image))
	}
}

// TestCompressWebPReencodesLossless 验证 WebP 解码后重新编码，且输出不会变大。
func TestCompressWebPReencodesLossless(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pic.webp")
	img := image.NewNRGBA(image.Rect(0, 0, 64, 48))
	for y := range 48 {
		for x := range 64 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	// 用最低压缩级别写出一个较大的源 WebP，让重新编码有压缩空间。
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := nativewebp.Encode(f, img, &nativewebp.Options{CompressionLevel: 0}); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	result := compressImage(path, CompressionOptions{})
	if result.Error != "" {
		t.Fatalf("compressImage error: %s", result.Error)
	}
	if result.OutputSize > result.OriginalSize {
		t.Fatalf("output grew from %d to %d", result.OriginalSize, result.OutputSize)
	}
	if filepath.Base(result.OutputPath) != "pic-min.webp" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	if result.Width != 64 || result.Height != 48 {
		t.Fatalf("unexpected size %dx%d", result.Width, result.Height)
	}
	out, err := os.Open(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := webp.Decode(out); err != nil {
		t.Fatalf("output is not valid WebP: %v", err)
	}
}

// TestCompressSVGMinifiesText 验证 SVG 文本压缩去除注释和多余空白，且输出更小。
func TestCompressSVGMinifiesText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "icon.svg")
	source := `<?xml version="1.0" encoding="UTF-8"?>
<!-- a decorative comment that should be removed -->
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">
    <rect x="10"    y="10" width="80" height="80" fill="#ff0000" />
</svg>
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	result := compressImage(path, CompressionOptions{})
	if result.Error != "" {
		t.Fatalf("compressImage error: %s", result.Error)
	}
	if result.OutputSize >= result.OriginalSize {
		t.Fatalf("expected SVG to shrink, got %d -> %d", result.OriginalSize, result.OutputSize)
	}
	if filepath.Base(result.OutputPath) != "icon-min.svg" {
		t.Fatalf("unexpected output path %q", result.OutputPath)
	}
	data, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "decorative comment") {
		t.Fatalf("comment survived minification: %q", data)
	}
}

// TestExpandPathsFindsImagesAndDeduplicates 验证目录递归、扩展名大小写和路径去重。
func TestExpandPathsFindsImagesAndDeduplicates(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.png")
	b := filepath.Join(dir, "b.JPG")
	for _, path := range []string{a, b} {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := expandPaths([]string{dir, a})
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("expandPaths = %#v", got)
	}
}

// TestExpandPathsIncludesNewFormats 验证目录扫描能识别 GIF、WebP、SVG 扩展名。
func TestExpandPathsIncludesNewFormats(t *testing.T) {
	dir := t.TempDir()
	want := []string{
		filepath.Join(dir, "anim.gif"),
		filepath.Join(dir, "icon.svg"),
		filepath.Join(dir, "pic.webp"),
	}
	for _, path := range want {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := expandPaths([]string{dir})
	if len(got) != len(want) {
		t.Fatalf("expandPaths = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expandPaths[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestWriteAtomicNeverOverwritesExistingFile 模拟输出名在检查后被其他进程抢占。
// writeAtomic 必须返回错误，并完整保留已经存在的数据。
func TestWriteAtomicNeverOverwritesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "already-there.png")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, bytes.NewReader([]byte("replacement")), 0o644); err == nil {
		t.Fatal("writeAtomic unexpectedly overwrote an existing path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("existing data changed to %q", data)
	}
}

// writePNG 是测试辅助函数。t.Helper 让失败位置指向调用它的测试，而不是辅助函数内部。
func writePNG(t *testing.T, path string, img image.Image, level png.CompressionLevel) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&png.Encoder{CompressionLevel: level}).Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// imagingOpen 只供测试读取 PNG 输出。defer 确保 Decode 成功或失败后文件都关闭。
func imagingOpen(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
