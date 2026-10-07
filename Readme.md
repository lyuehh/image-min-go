# Image Min

一个使用 [MyGO](https://github.com/egoist/mygo) 原生 UI 构建的轻量级 macOS 图片压缩工具。界面完全由 Go 绘制，不使用 HTML、JavaScript 或 WebView。把 PNG 或 JPEG 图片拖进窗口，Image Min 会在原文件旁生成一个带 `-min` 后缀的压缩副本，不会覆盖原图。

## 功能

- 拖放图片或通过原生文件选择器批量添加
- JPEG 可调压缩质量（默认 82）
- PNG 使用最佳无损压缩
- 自动跳过重复文件，并逐项显示压缩结果
- 只有结果更小时才写入重新编码的数据，否则复制原图，保证输出不会变大
- 一键在 Finder 中显示输出文件
- 浅色、深色模式和 macOS 窗口状态保存

## 开发

需要 Go 1.27 或更高版本。

```bash
go test ./...
go run .
```

使用 MyGO 的开发模式运行真实 `.app` 包：

```bash
go tool mygo dev
```

构建 macOS 应用和 DMG：

```bash
go tool mygo build
```

产物默认写入 `build/`。

## 当前范围

首个版本支持 PNG、JPG 和 JPEG。重新编码会移除 EXIF 等元数据；JPEG 的 EXIF 方向会先应用到像素，避免输出方向发生变化。动画 GIF、WebP、AVIF 和 SVG 暂未包含在 MVP 中。
