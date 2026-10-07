package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	want := color.NRGBAModel.Convert(img.At(25, 31)).(color.NRGBA)
	got := color.NRGBAModel.Convert(out.At(25, 31)).(color.NRGBA)
	if got != want {
		t.Fatalf("pixel changed: got %#v, want %#v", got, want)
	}
}

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

func imagingOpen(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return png.Decode(f)
}
