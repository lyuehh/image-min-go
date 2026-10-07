# Image Min：Go 初学者阅读指南

这份指南不要求你已经熟悉 Go。建议一边打开源码，一边按下面的顺序阅读和运行。

## 1. 先运行项目

```bash
go test ./...
go run .
```

- `go test ./...`：运行当前模块及所有子目录中的测试。
- `go run .`：编译并运行当前目录的 `package main`。
- 命令末尾的 `.` 表示“当前目录”；`./...` 表示“当前目录及其所有子包”。

打包 macOS 应用：

```bash
go tool mygo build
```

`tool` 指运行 `go.mod` 中声明的工具。本项目把 MyGO CLI 固定在与运行库相同的版本，避免不同电脑使用不同 CLI。

## 2. 文件和调用顺序

| 文件 | 职责 | 建议阅读顺序 |
| --- | --- | --- |
| `main.go` | 程序入口、窗口、菜单、系统打开文件事件 | 1 |
| `app.go` | MyGO 原生 UI、界面状态、异步任务调度 | 2 |
| `files.go` | 展开目录、筛选图片、读取文件信息 | 3 |
| `compressor.go` | 解码、压缩、安全写出文件 | 4 |
| `app_test.go` | 不打开真实窗口测试原生 UI | 5 |
| `compressor_test.go` | 用临时图片测试压缩与文件安全 | 6 |
| `mygo.json` | 应用名称、版本、文件关联和 macOS 打包配置 | 7 |

一次普通操作的数据流：

```text
用户拖放文件
  -> imageApp.dropZone
  -> imageApp.addPaths（启动后台 goroutine）
  -> inspectPaths / expandPaths
  -> Window.Update（回到 UI 主线程）
  -> imageApp.merge
  -> MyGO 再次调用 imageApp.view 绘制新状态
```

压缩按钮的数据流：

```text
imageApp.compress
  -> 标记 busy 和 Processing
  -> 后台逐张调用 compressImage
  -> 每张完成后通过 Window.Update 写回 Result
  -> view 根据 Result 显示节省体积或错误
```

## 3. 项目中常见的 Go 语法和符号

### `package main` 和 `func main()`

可执行程序必须属于 `package main`，并有一个无参数、无返回值的 `func main()`。同一目录里的 `main.go`、`app.go`、`files.go` 和 `compressor.go` 都属于同一个包，所以它们可以直接调用彼此的小写函数。

### 大写与小写名称

Go 用名称首字母控制可见性：

- `CompressionResult`、`OutputPath`：首字母大写，其他 package 可以访问。
- `compressImage`、`imageItem`：首字母小写，只能在当前 package 中访问。

这相当于其他语言中的 public/private，但 Go 不使用对应关键字。

### `var`、`const`、`:=` 和 `=`

```go
const defaultJPEGQuality = 82 // 常量，运行时不能改
var files []string            // 声明变量，得到该类型的零值
quality := options.JPEGQuality // 声明并由右侧推断类型
quality = 90                   // 给已经存在的变量赋值
```

注意：

- `:=` 只能在函数内部使用。
- `:=` 左边至少要有一个当前作用域中的新变量。
- `=` 不会创建新变量。
- 在较短作用域中重复使用 `:=` 可能遮蔽外层同名变量，尤其要小心 `err`。

### `{}`、`()` 和 `[]`

- `{}`：函数体、条件体，或结构体/切片/map 字面量。
- `()`：函数参数、函数调用、分组表达式。
- `[]T`：元素类型为 `T` 的切片，例如 `[]string`。
- `[N]T`：固定长度为 `N` 的数组。本项目业务数据主要使用切片。
- `value[index]`：按索引读取数组/切片，越界会 panic。
- `mapping[key]`：按键访问 map；键不存在时得到值类型的零值。

### `&` 和 `*`

```go
state := &imageApp{} // & 取得新结构体的地址，类型是 *imageApp
var win *mygo.Window // *T 表示“指向 T 的指针”
```

- 指针让多个函数操作同一个对象，而不是各自的副本。
- 指针的零值是 `nil`，解引用 nil 指针会 panic。
- Go 调用方法时通常会自动处理 `a.Field` 中的解引用，不必写 `(*a).Field`。
- `&a.quality` 把字段地址交给 MyGO 滑块，滑块才能修改真实状态。

### `.`、`,`、`;` 和 `...`

- `a.view`：`.` 访问字段或方法。
- `x, err := work()`：`,` 分隔多个返回值或参数。
- Go 通常不写分号；编译器会在合适位置自动插入。
- `append(dst, src...)`：`...` 把一个切片展开成多个参数。
- 函数参数中的 `values ...string` 则表示可变参数。

### `_` 空白标识符

```go
for _, path := range paths { ... }
_, ok := supportedFormat(path)
```

Go 不允许声明后不使用变量。`_` 明确表示“这个值我知道存在，但不需要”。它不会保存数据。

### `nil` 和零值

未显式初始化时，Go 会提供零值：

- 数字为 `0`
- 布尔值为 `false`
- 字符串为 `""`
- 指针、切片、map、函数、接口为 `nil`

nil 切片可以读取长度、遍历和 `append`，但 nil map 不能写入，所以 map 通常先用 `make` 创建。

## 4. 结构体、方法、嵌入和接口

### 结构体

```go
type ImageInfo struct {
    Path string
    Size int64
}
```

结构体把相关字段组成一个类型。`ImageInfo{Path: path}` 是带字段名的结构体字面量，未写字段自动取零值。

### 方法接收者

```go
func (a *imageApp) merge(infos []ImageInfo) { ... }
```

`(a *imageApp)` 叫接收者。使用指针接收者是因为 `merge` 要修改 `a.items`。如果写成 `(a imageApp)`，方法只会修改结构体副本。

### 匿名嵌入

```go
type imageItem struct {
    ImageInfo
    Result *CompressionResult
}
```

`ImageInfo` 没有单独字段名，称为嵌入。可以直接写 `item.Path`，也可以完整写 `item.ImageInfo.Path`。嵌入是字段提升，不是传统面向对象语言中的继承。

### 接口

`image.Image` 和 `io.Reader` 都是接口。类型只要实现接口要求的方法就自动满足接口，不需要 `implements` 声明。

例如 `writeAtomic` 接收 `io.Reader`，所以它既能读取 `*os.File`，也能读取 `*bytes.Reader`。这让文件写入逻辑不必关心数据来自哪里。

## 5. 切片和 map

切片可以理解为“对底层数组的一段视图”，包含指针、长度和容量。

```go
items := make([]ImageInfo, 0, len(files))
items = append(items, item)
```

- `len(items)`：当前元素数。
- `cap(items)`：不用重新分配时最多能容纳多少元素。
- `items[:i]`：从开头到 i，但不包含 i。
- `items[i+1:]`：从 i+1 到结尾。
- `append(items[:i], items[i+1:]...)`：删除第 i 项的常见写法。

注意：子切片通常共享底层数组。修改一个切片的元素，另一个切片可能也会看到变化。

map 示例：

```go
seen := make(map[string]bool)
if !seen[path] {
    seen[path] = true
}
```

并发读写普通 map 会产生 data race，严重时程序会崩溃。本项目只在 UI 主线程中修改界面 map/切片。

## 6. 多返回值和错误处理

Go 通常显式返回错误：

```go
info, err := os.Stat(path)
if err != nil {
    return result
}
```

- `error` 是内建接口。
- `nil` error 表示成功。
- 不要因为“不知道怎么处理”就忽略错误。
- `errors.Is(err, os.ErrNotExist)` 能识别被包装过的“文件不存在”错误，比字符串比较可靠。

“comma ok” 模式：

```go
format, ok := supportedFormat(path)
if !ok { ... }
```

`ok` 明确表示第一个返回值是否有效。map 查询和类型断言也经常使用同样模式。

## 7. `defer`、闭包和命名返回值

### `defer`

```go
file, err := os.Open(path)
if err != nil { ... }
defer file.Close()
```

`defer` 的函数会在当前函数返回前执行，多个 defer 按后进先出顺序执行。资源成功打开后应尽快安排关闭。

注意：不要在一个处理成千上万项的大循环里随意 `defer`，因为它们要等整个外层函数返回才执行。本项目的一次 `compressImage` 只打开一个输入文件，因此适合使用。

### 闭包

```go
add := func(path string) {
    files = append(files, path)
}
```

匿名函数捕获外层的 `files`，这就是闭包。闭包很方便，但被 goroutine 使用时要确认捕获变量不会同时被别处修改。

### 命名返回值

```go
func writeAtomic(...) (err error) { ... }
```

返回值 `err` 在函数体中也是变量，因此 defer 可以检查最终错误并决定是否删除临时文件。命名返回值应谨慎使用；过长函数中的裸 `return` 容易让读者不知道返回什么。

## 8. goroutine 和 UI 主线程

`go func() { ... }()` 会并发启动一个 goroutine。它不是操作系统线程的一一对应物，创建成本较低，但仍需管理共享状态。

本项目遵守两条规则：

1. 文件扫描、对话框等待、图片编解码在 goroutine 中运行。
2. `imageApp` 的 UI 状态只通过 `Window.Update` 在主线程修改。

```go
go func() {
    items := inspectPaths(paths) // 后台工作
    a.update(func() {            // 排回主线程
        a.items = append(a.items, items...)
    })
}()
```

注意：

- 不要从 goroutine 保存或使用 `*ui.Context`；它只属于当前绘制帧。
- `Window.Update` 是异步安排，不会等待闭包执行结束。
- 压缩期间禁用添加/删除操作，是为了让后台保存的切片索引保持有效。
- `go test -race ./...` 可以发现很多并发读写错误，但不能证明程序绝对没有竞态。
- `sync.Mutex` 用 `Lock`/`Unlock` 保护复合状态；`atomic.Bool` 适合单个布尔标记。

## 9. MyGO 原生 UI 的状态模型

MyGO 的原生 UI 是“每帧根据状态重新声明界面”：

```go
func (a *imageApp) view(c *ui.Context) {
    if ui.Button(c, "清空").Clicked() {
        a.items = nil
    }
}
```

元素对象只属于当前帧，真正持久的数据放在 `imageApp` 中。事件发生后 MyGO 再调用 `view`，新状态自然产生新界面。

`Children(func() { ... })` 用闭包表达父子关系。链式调用：

```go
ui.Text(c, "标题").FontSize(17).FontWeight(650)
```

每个方法返回同一个 `*ui.Element`，所以能继续设置属性。这不是 Go 的特殊语法，只是普通方法连续调用。

虚拟列表的 `Key` 必须稳定且唯一。本项目用图片绝对路径；如果改用当前索引，删除前面的行后，焦点或内部状态可能错误地跳到另一张图片。

## 10. 图片压缩与文件安全

`compressImage` 的步骤：

1. `os.Stat` 确认输入存在且是普通文件。
2. 根据扩展名选择 PNG/JPEG 解码器。
3. JPEG 先应用 EXIF Orientation，避免去掉元数据后方向错误。
4. 编码到 `bytes.Buffer`，尚不修改磁盘。
5. 比较新旧大小；新数据没有更小时复制原图。
6. `nextOutputPath` 选择 `-min`、`-min-2` 等未占用名称。
7. `writeAtomic` 在同目录写临时文件，完成后用硬链接原子地发布到正式路径。

重要注意事项：

- 当前版本会移除 EXIF、GPS、相机型号等元数据。这有助于减小体积和保护隐私，但如果业务需要元数据，就必须另行保存。
- PNG 是无损压缩；JPEG 是有损压缩。
- 图片解码后占用的是像素内存，不是压缩文件大小。例如 12000×9000 RGBA 图片仅像素就约占 412 MiB。
- 不要把临时文件放到另一个文件系统再建立硬链接；跨文件系统链接会失败。
- 只先检查 `os.Stat` 再普通 `os.Rename` 存在竞态：其他进程可能在两步之间创建同名文件，并在 Unix 上被 Rename 覆盖。本项目最终用 `os.Link`，目标已存在时会安全失败。
- 不要直接覆盖原图。当前 `-min` 输出策略让失败可恢复。
- 不要用字符串判断具体 I/O 错误，应优先用 `errors.Is`/`errors.As`。

## 11. 如何阅读测试

`testing.T` 代表当前测试：

```go
func TestSomething(t *testing.T) {
    if got != want {
        t.Fatalf("got %v, want %v", got, want)
    }
}
```

- 测试函数名称必须以 `Test` 开头。
- `t.Fatal`/`t.Fatalf` 报错后立即停止当前测试。
- `t.Error`/`t.Errorf` 报错后继续，适合一次报告多个断言。
- `t.TempDir()` 创建测试专用临时目录，测试结束自动删除。
- `t.Helper()` 把辅助函数标记为 helper，失败行号会指向调用者。
- `ui.NewTester` 在内存中绘制原生 UI，不需要打开窗口，适合点击按钮和检查文本。

常用验证命令：

```bash
gofmt -w *.go          # 按 Go 官方规则格式化
go test ./...          # 普通测试
go test -race ./...    # 开启数据竞争检测
go vet ./...           # 检查常见可疑写法
go build ./...         # 编译所有包
go tool mygo build     # 构建 macOS .app 和 DMG
```

## 12. 适合初学者的小改动

可以按风险从低到高尝试：

1. 修改 `defaultJPEGQuality`，运行测试并观察默认滑块值。
2. 修改 `formatSize` 的小数位数，并补一个测试用例。
3. 在 `imageItem` 中增加“压缩耗时”，在 `compressImage` 前后记录时间并显示。
4. 为列表增加“重新压缩”按钮；注意压缩时仍要禁用会改变切片索引的操作。
5. 增加新的图片格式；先确认解码、编码、透明通道、动画和元数据策略，再修改 `supportedFormat`，不能只添加扩展名。

每次修改后至少运行 `gofmt -w *.go && go test ./...`。涉及 goroutine 或共享状态时，再运行 `go test -race ./...`。
