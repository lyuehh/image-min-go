# Image Min

一个使用 [MyGO](https://github.com/egoist/mygo) 原生 UI 构建的轻量级 macOS 图片压缩工具。界面完全由 Go 绘制，不使用 HTML、JavaScript 或 WebView。把 PNG、JPEG、GIF、WebP、TIFF 或 SVG 图片拖进窗口，Image Min 会在原文件旁生成一个带 `-min` 后缀的压缩副本，不会覆盖原图。

## 功能

- 拖放图片或通过原生文件选择器批量添加
- 中英文界面，可在设置页面切换语言（默认跟随系统语言，未匹配时回退中文）
- 默认无损压缩；可在设置页面开启有损压缩并调节压缩质量
- 可选转换为现代格式：在设置页面开启“转换为现代格式”后，才能把位图（PNG/JPEG/WebP/TIFF）另存为 AVIF 或 JPEG XL，默认保持原格式
- 转换使用纯 Go（CGo-free）编码器（AVIF/JXL 的 libavif/libjxl 以 WASM 运行），不依赖任何外部二进制，保持自包含 `.app`
- JPEG 无损模式以最高质量重新编码，有损模式按设置的质量（默认 82）编码
- PNG 使用最佳无损压缩
- GIF 重新编码并保留全部动画帧
- WebP 以无损 VP8L 重新编码
- TIFF 以无损 Deflate（带差分预测）重新编码
- SVG 做文本级压缩（去除注释和多余空白）
- GIF 和 SVG 不参与格式转换，始终保持原格式
- 自动跳过重复文件，并逐项显示压缩结果
- 只有结果更小时才写入重新编码的数据，否则复制原图，保证输出不会变大
- 批量压缩时显示实时进度（n/total）和整体节省百分比
- 一键在 Finder 中显示输出文件
- 浅色、深色模式和 macOS 窗口状态保存
- 界面语言由 MyGO 原生 i18n 托管（文案在 `locales/`），压缩偏好保存在用户配置目录，重启后仍然生效

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

## Go 初学者阅读指南

建议先阅读 [`docs/GO_BEGINNER_GUIDE.md`](docs/GO_BEGINNER_GUIDE.md)。它按程序启动顺序介绍各文件职责，并解释本项目出现的 Go 语法、常用符号、指针、切片、错误处理、goroutine、MyGO 原生 UI 状态更新和文件安全注意事项。核心源码也包含对应的逐段中文注释。

## 当前范围

支持 PNG、JPG、JPEG、GIF、WebP、TIFF（.tif/.tiff）和 SVG。位图重新编码会移除 EXIF 等元数据；JPEG 的 EXIF 方向会先应用到像素，避免输出方向发生变化。GIF 会保留动画帧，WebP 以无损 VP8L 重新编码，TIFF 以无损 Deflate 重新编码，SVG 只做文本级压缩。AVIF 暂未包含。
