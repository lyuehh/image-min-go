package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	loc "github.com/lyuehh/image-min-go/locales"
)

// 本文件实现设置页面（以弹窗形式呈现）以及设置的打开、读取和持久化辅助逻辑。
// 设置内容：切换界面语言（中文 / English）、是否允许有损压缩、有损压缩质量。
//
// 界面语言由 MyGO 原生 i18n 托管：选择语言调用 mygo.App.SetLocale，MyGO 负责持久化
// 并通知各窗口重绘和菜单重建；本应用的设置文件只保存压缩相关偏好。

// openSettings 打开设置弹窗。打开前把当前有损质量同步到滑块绑定的 quality 字段。
func (a *imageApp) openSettings() {
	a.quality = float64(a.settings.LossyQuality)
	a.settingsOpen = true
}

// persistSettings 把当前设置写回磁盘。它是“尽力保存”：失败不会打断界面操作，
// 下一次修改仍会再尝试写入。保存发生在 UI 主线程、数据量极小，直接同步调用即可。
func (a *imageApp) persistSettings() {
	_ = saveSettings(a.settings)
}

// settingsDialog 绘制设置弹窗。只有 settingsOpen 为真时 Modal 才会渲染内容。
func (a *imageApp) settingsDialog(c *ui.Context) {
	if !a.settingsOpen {
		return
	}
	t, s := c.Theme(), loc.In(c.Locale())
	ui.Modal(c, &a.settingsOpen, func() {
		ui.Column(c).Width(360).Gap(18).Children(func() {
			ui.Text(c, s.SettingsTitle()).FontSize(17).FontWeight(700)

			// 语言选择：Segmented 用 *int 选择，0=中文、1=English。
			// 必须通过 .Changed() 判断是否切换——它会先把本帧的点击输入应用到 lang，
			// 再返回结果；直接读取 lang 会拿到应用点击前的旧值，导致切换“不生效”。
			ui.Column(c).Gap(7).Children(func() {
				ui.Text(c, s.Language()).FontSize(12).FontWeight(600).TextColor(t.TextMuted)
				lang := languageIndex(c.Locale())
				seg := ui.Segmented(c, &lang, s.LangChinese(), s.LangEnglish()).Label(s.Language())
				if seg.Changed() {
					// SetLocale 立即让窗口以新语言重绘，并持久化到下次启动；
					// 它还会触发 OnLocaleChanged，由 main.go 重建应用菜单。
					mygo.App.SetLocale(languageCode(lang))
				}
			})

			ui.Divider(c)

			// 有损压缩开关：关闭=无损（默认），开启后才显示并启用质量滑块。
			ui.Column(c).Gap(7).Children(func() {
				ui.Row(c).Gap(10).Children(func() {
					ui.Text(c, s.AllowLossy()).FontSize(13).FontWeight(600).Grow(1)
					sw := ui.Switch(c, &a.settings.AllowLossy).Label(s.AllowLossy())
					if sw.Changed() {
						a.persistSettings()
					}
				})
				if a.settings.AllowLossy {
					ui.Text(c, s.AllowLossyHint()).FontSize(11).TextColor(t.TextMuted)
					ui.Row(c).Gap(10).Children(func() {
						ui.Text(c, s.LossyQuality()).FontSize(12).TextColor(t.TextMuted).Grow(1)
						slider := ui.StepSlider(c, &a.quality, minLossyQuality, maxLossyQuality, 1).Width(150).Label(s.LossyQuality())
						ui.Textf(c, "%.0f", a.quality).Width(28).FontSize(12).FontFeatures("tnum")
						if slider.Changed() {
							a.settings.LossyQuality = int(a.quality)
							a.persistSettings()
						}
					})
				} else {
					ui.Text(c, s.LosslessHint()).FontSize(11).TextColor(t.TextMuted)
				}
			})

			ui.Divider(c)

			// 转换格式开关：关闭=保持原格式（默认），开启后才显示并启用目标格式选择。
			ui.Column(c).Gap(7).Children(func() {
				ui.Row(c).Gap(10).Children(func() {
					ui.Text(c, s.ConvertFormat()).FontSize(13).FontWeight(600).Grow(1)
					sw := ui.Switch(c, &a.settings.ConvertFormat).Label(s.ConvertFormat())
					if sw.Changed() {
						a.persistSettings()
					}
				})
				if a.settings.ConvertFormat {
					ui.Text(c, s.ConvertFormatHint()).FontSize(11).TextColor(t.TextMuted)
					// 目标格式：Segmented 用 *int 选择，0=AVIF、1=JPEG XL。通过 .Changed() 判断是否切换。
					target := targetFormatIndex(a.settings.TargetFormat)
					seg := ui.Segmented(c, &target, s.FormatAvif(), s.FormatJxl()).Label(s.TargetFormat())
					if seg.Changed() {
						a.settings.TargetFormat = targetFormatCode(target)
						a.persistSettings()
					}
				} else {
					ui.Text(c, s.KeepFormatHint()).FontSize(11).TextColor(t.TextMuted)
				}
			})

			ui.Row(c).Children(func() {
				ui.Spacer(c)
				if ui.PrimaryButton(c, s.Done()).Clicked() {
					a.settingsOpen = false
				}
			})
		})
	})
}

// languageIndex 把当前界面语言映射为 Segmented 的索引（0=中文、1=English）。
// 入参是 mygo.App.Locale() 解析出的 BCP 47 标签，如 "en" 或 "zh-Hans"。
func languageIndex(locale string) int {
	if locale == langEnglish {
		return 1
	}
	return 0
}

// languageCode 把 Segmented 的索引映射回语言标签，交给 mygo.App.SetLocale。
func languageCode(index int) string {
	if index == 1 {
		return langEnglish
	}
	return langChinese
}

// targetFormatIndex 把目标格式名映射为 Segmented 的索引（0=AVIF、1=JPEG XL）。
func targetFormatIndex(format string) int {
	if format == formatJXL {
		return 1
	}
	return 0
}

// targetFormatCode 把 Segmented 的索引映射回目标格式名。
func targetFormatCode(index int) string {
	if index == 1 {
		return formatJXL
	}
	return formatAVIF
}

// localizedMenu 按当前语言构建应用菜单。main.go 在启动时及语言变化时调用它。
func (a *imageApp) localizedMenu() *mygo.Menu {
	s := loc.In(mygo.App.Locale())
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: s.MenuFile(), Submenu: []*mygo.MenuItem{
			{Label: s.MenuAddImages(), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.pickImages() }},
			mygo.Separator(),
			{Role: mygo.RoleClose},
		}},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleViewMenu},
		{Role: mygo.RoleWindowMenu},
	})
}
