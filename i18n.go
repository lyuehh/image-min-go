package main

// 本文件集中存放界面上所有可见文案的中英文两份翻译。
//
// 做法说明（给 Go 初学者）：把每条文案抽成结构体 L 的一个字段，再分别用
// zhStrings、enStrings 两个变量填好中文、英文。界面代码不再写死字符串，而是通过
// 当前语言取出对应的 L，再读取字段。这样新增一种语言只需再填一张表，界面代码不动。
//
// 字段里带 %s、%d、%.0f 的是 fmt 格式模板，供 Textf / fmt.Sprintf 使用；中英文模板的
// 占位符数量和顺序必须一致，否则格式化会出错。

// L 是一张完整的界面文案表。首字母大写表示字段可被其他文件读取。
type L struct {
	// 拖放区
	DropHere     string
	DropHint     string
	ChooseImages string

	// 工具栏与列表
	NoImages      string
	ImageCountFmt string // "%d 张图片"
	LoadingImages string
	Clear         string
	Settings      string
	EmptyHint     string
	Show          string
	Remove        string

	// 单条状态
	Processing     string
	Waiting        string
	AlreadyOptimal string
	SavedItemFmt   string // "−%s · %.0f%%"

	// 底部汇总
	DoneWithPercentFmt string // "已完成 %d 张，共节省 %s（%.0f%%）"
	DoneSummaryFmt     string // "已完成 %d 张，共节省 %s"
	OutputHint         string
	Compress           string
	Compressing        string
	ProgressSpinner    string

	// 文件选择对话框
	PickDialogTitle string
	ImagesFilter    string

	// 应用菜单
	MenuFile      string
	MenuAddImages string

	// 设置页面
	SettingsTitle  string
	Language       string
	LangChinese    string
	LangEnglish    string
	AllowLossy     string
	AllowLossyHint string
	LossyQuality   string
	LosslessHint   string
	Done           string
}

// zhStrings 是简体中文文案，也是应用的默认语言，保持与改造前完全一致的措辞。
var zhStrings = L{
	DropHere:     "把图片拖到这里",
	DropHint:     "支持 PNG、JPEG、GIF、WebP、TIFF 和 SVG，也可以拖入整个文件夹",
	ChooseImages: "选择图片…",

	NoImages:      "还没有图片",
	ImageCountFmt: "%d 张图片",
	LoadingImages: "正在读取图片",
	Clear:         "清空",
	Settings:      "设置",
	EmptyHint:     "添加图片后将在这里显示",
	Show:          "显示",
	Remove:        "移除",

	Processing:     "处理中",
	Waiting:        "等待压缩",
	AlreadyOptimal: "已是最优",
	SavedItemFmt:   "−%s · %.0f%%",

	DoneWithPercentFmt: "已完成 %d 张，共节省 %s（%.0f%%）",
	DoneSummaryFmt:     "已完成 %d 张，共节省 %s",
	OutputHint:         "输出保存在原文件旁，不会覆盖原图",
	Compress:           "开始压缩",
	Compressing:        "压缩中",
	ProgressSpinner:    "正在压缩",

	PickDialogTitle: "选择要压缩的图片",
	ImagesFilter:    "图片",

	MenuFile:      "文件",
	MenuAddImages: "添加图片…",

	SettingsTitle:  "设置",
	Language:       "语言",
	LangChinese:    "中文",
	LangEnglish:    "English",
	AllowLossy:     "允许有损压缩",
	AllowLossyHint: "开启后 JPEG 会按下面的质量重新编码，体积更小但可能损失画质。",
	LossyQuality:   "有损压缩质量",
	LosslessHint:   "默认进行无损压缩：JPEG 以最高质量重新编码，其它格式本就无损。",
	Done:           "完成",
}

// enStrings 是英文文案。
var enStrings = L{
	DropHere:     "Drop images here",
	DropHint:     "Supports PNG, JPEG, GIF, WebP, TIFF and SVG — whole folders too",
	ChooseImages: "Choose images…",

	NoImages:      "No images yet",
	ImageCountFmt: "%d images",
	LoadingImages: "Reading images",
	Clear:         "Clear",
	Settings:      "Settings",
	EmptyHint:     "Added images will show up here",
	Show:          "Show",
	Remove:        "Remove",

	Processing:     "Processing",
	Waiting:        "Waiting",
	AlreadyOptimal: "Already optimal",
	SavedItemFmt:   "−%s · %.0f%%",

	DoneWithPercentFmt: "Done %d, saved %s (%.0f%%)",
	DoneSummaryFmt:     "Done %d, saved %s",
	OutputHint:         "Output is saved next to the original; the original is never overwritten",
	Compress:           "Compress",
	Compressing:        "Compressing",
	ProgressSpinner:    "Compressing",

	PickDialogTitle: "Choose images to compress",
	ImagesFilter:    "Images",

	MenuFile:      "File",
	MenuAddImages: "Add images…",

	SettingsTitle:  "Settings",
	Language:       "Language",
	LangChinese:    "中文",
	LangEnglish:    "English",
	AllowLossy:     "Allow lossy compression",
	AllowLossyHint: "When on, JPEG is re-encoded at the quality below: smaller files, but some quality may be lost.",
	LossyQuality:   "Lossy quality",
	LosslessHint:   "Lossless by default: JPEG is re-encoded at top quality, and the other formats are already lossless.",
	Done:           "Done",
}

// stringsFor 按语言代码返回对应文案表，未知语言回退到中文。
func stringsFor(lang string) *L {
	if lang == langEnglish {
		return &enStrings
	}
	return &zhStrings
}
