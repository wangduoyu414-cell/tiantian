// weixin-key-gui is the desktop one-click export front end: directory input
// with memory, system folder picker, preflight (offline cache first), an
// explicit user-approved WeChat restart for material capture, then the same
// export pipeline the CLI uses. Only ever operate on your own account.
package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/io/system"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"

	"weixin-key/internal/config"
	"weixin-key/internal/guicore"
	"weixin-key/internal/wxkey"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("微信聊天记录一键导出"),
			app.Size(unit.Dp(880), unit.Dp(620)),
			app.MinSize(unit.Dp(640), unit.Dp(480)),
		)
		if err := run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

type ui struct {
	theme *material.Theme

	dirEditor     widget.Editor
	rootEditor    widget.Editor
	startBtn      widget.Clickable
	cancelBtn     widget.Clickable
	browseBtn     widget.Clickable
	browseRootBtn widget.Clickable
	detectBtn     widget.Clickable
	openBtn       widget.Clickable
	approveBtn    widget.Clickable
	cancelCap     widget.Clickable
	recoveryBtn   widget.Clickable
	accountEnum   widget.Enum

	job        *guicore.Job
	settings   *guicore.Settings
	dataRoot   string
	accounts   []guicore.AccountInfo
	statusLine string
	lastStage  guicore.Stage
	shutdown   <-chan struct{}
}

func run(w *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	loadCJKFonts(th)

	settings, settingsErr := guicore.LoadSettings()
	if settings == nil {
		settings = &guicore.Settings{}
	}
	u := &ui{theme: th, job: guicore.NewJob(), settings: settings, lastStage: guicore.StageIdle}
	if settingsErr != nil {
		u.statusLine = "设置读取失败：" + settingsErr.Error()
	}

	u.dirEditor.SingleLine = true
	u.dirEditor.Submit = false
	u.rootEditor.SingleLine = true
	u.rootEditor.Submit = true

	// Directory memory: last valid dir wins; otherwise suggest a Documents path.
	dir := settings.LastExportDir
	if dir == "" {
		if s, err := guicore.SuggestExportDir(); err == nil {
			dir = s
		}
	}
	u.dirEditor.SetText(dir)

	// Data-root memory + detection: remembered account dir, else config, else
	// platform auto-detect (which also scans fixed drives for xwechat_files).
	cfgDBRoot := ""
	if cfg, err := config.Load(); err == nil {
		cfgDBRoot = cfg.DBRoot
	}
	selected := ""
	u.dataRoot, selected = guicore.DetectDataRoot(cfgDBRoot, settings.DBRoot)
	u.rootEditor.SetText(u.dataRoot)
	u.refreshAccounts(selected)

	for {
		err := u.runWindow(w)
		u.shutdown = u.job.RequestClose()
		select {
		case <-u.shutdown:
			return err
		default:
		}
		if err != nil {
			// Do not spin creating windows after a renderer/driver failure.
			// The native fallback retains a visible retry path on Windows.
			waitForRecoveryWithoutWindow(u.job, u.shutdown)
			return err
		}
		// Gio's DestroyEvent arrives after destruction, so the old window
		// cannot veto close. Retain the job and show a recovery-only window.
		w = new(app.Window)
		w.Option(app.Title("微信导出 — 正在安全退出"),
			app.Size(unit.Dp(760), unit.Dp(500)), app.MinSize(unit.Dp(640), unit.Dp(400)))
	}
}

func (u *ui) runWindow(w *app.Window) error {
	// Repaint while the job runs so progress streams in.
	done := make(chan struct{})
	defer close(done)
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				w.Invalidate()
			}
		}
	}()

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if u.shutdown != nil {
				select {
				case <-u.shutdown:
					w.Perform(system.ActionClose)
				default:
				}
				u.layoutRecovery(gtx)
				e.Frame(gtx.Ops)
				continue
			}
			u.handle(gtx)
			u.layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (u *ui) layoutRecovery(gtx layout.Context) layout.Dimensions {
	if u.recoveryBtn.Clicked(gtx) {
		u.job.RetryRecovery()
	}
	stage, events, _, _, _ := u.job.State()
	message := "已请求取消，正在等待工作线程完成清理。此窗口会在安全退出后关闭。请勿在任务管理器中强制结束工具。"
	if stage == guicore.StageRecovering && len(events) > 0 {
		message += "\n\n" + events[len(events)-1].Message
	}
	return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(material.H6(u.theme, "正在安全退出").Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(material.Body1(u.theme, message).Layout),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if stage != guicore.StageRecovering {
					return layout.Dimensions{}
				}
				return material.Button(u.theme, &u.recoveryBtn, "重试恢复（不会重新采集）").Layout(gtx)
			}),
		)
	})
}

func hasDBStorage(dir string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "db_storage"))
	return err == nil
}

// refreshAccounts rescans the data root and rebuilds the account selector.
// With exactly one account it is auto-selected; with several, the user's
// existing selection is kept when still present; nothing is guessed.
func (u *ui) refreshAccounts(prefer string) {
	u.accounts = nil
	u.accountEnum.Value = ""
	root := strings.TrimSpace(u.rootEditor.Text())
	if root == "" {
		return
	}
	accts, err := guicore.ListAccounts(root)
	if err != nil || len(accts) == 0 {
		return
	}
	u.accounts = accts
	if len(accts) == 1 {
		u.accountEnum.Value = accts[0].Dir
		return
	}
	for _, a := range accts {
		if a.Dir == prefer {
			u.accountEnum.Value = a.Dir
			return
		}
	}
}

// selectedAccount returns the explicitly selected account dir, or "".
func (u *ui) selectedAccount() string {
	for _, a := range u.accounts {
		if a.Dir == u.accountEnum.Value {
			return a.Dir
		}
	}
	return ""
}

func (u *ui) handle(gtx layout.Context) {
	if u.browseBtn.Clicked(gtx) {
		if picked, err := pickDirectory(u.dirEditor.Text()); err == nil && picked != "" {
			u.dirEditor.SetText(picked)
		}
	}
	if u.browseRootBtn.Clicked(gtx) {
		if picked, err := pickDirectory(u.rootEditor.Text()); err == nil && picked != "" {
			u.rootEditor.SetText(picked)
			u.refreshAccounts("")
		}
	}
	if u.detectBtn.Clicked(gtx) {
		u.refreshAccounts("")
		if len(u.accounts) == 0 {
			u.statusLine = "该目录下未找到含 db_storage 的账号目录"
		}
	}

	stage, _, _, _, _ := u.job.State()
	running := u.job.Running()

	if u.recoveryBtn.Clicked(gtx) && stage == guicore.StageRecovering {
		u.job.RetryRecovery()
	}
	if u.startBtn.Clicked(gtx) && !running {
		u.startExport()
	}
	if u.cancelBtn.Clicked(gtx) && running {
		u.job.Cancel()
	}
	if u.approveBtn.Clicked(gtx) && stage == guicore.StageNeedsCapture {
		u.job.ApproveCapture()
	}
	if u.cancelCap.Clicked(gtx) && stage == guicore.StageNeedsCapture {
		u.job.Cancel()
	}
	if u.openBtn.Clicked(gtx) {
		openFolder(u.dirEditor.Text())
	}
}

func (u *ui) startExport() {
	dir := strings.TrimSpace(u.dirEditor.Text())
	acct := u.selectedAccount()
	if acct == "" {
		u.statusLine = "请先选择账号（数据根目录下有多个账号时必须明确选择）"
		return
	}
	if err := guicore.ValidateExportDir(dir, acct); err != nil {
		u.statusLine = "目录无效：" + err.Error()
		return
	}
	// Remember the valid directory + account (non-secret settings, separate file).
	u.settings.LastExportDir = dir
	u.settings.DBRoot = acct
	if err := guicore.SaveSettings(u.settings); err != nil {
		u.statusLine = "设置保存失败：" + err.Error()
		return
	}
	u.statusLine = ""
	u.lastStage = guicore.StageIdle
	err := u.job.Start(guicore.JobOptions{
		DBRoot:  acct,
		OutDir:  dir,
		Resolve: wxkey.ResolveOptions{},
		Capture: captureFlow,
	})
	if err != nil {
		u.statusLine = err.Error()
	}
}

var (
	pad    = unit.Dp(12)
	gap    = unit.Dp(8)
	red    = color.NRGBA{R: 0xC0, G: 0x2C, B: 0x2C, A: 0xFF}
	green  = color.NRGBA{R: 0x1E, G: 0x8A, B: 0x3C, A: 0xFF}
	blue   = color.NRGBA{R: 0x1B, G: 0x5E, B: 0xC4, A: 0xFF}
	grayBg = color.NRGBA{R: 0xF6, G: 0xF6, B: 0xF6, A: 0xFF}
)

func (u *ui) layout(gtx layout.Context) {
	stage, events, rep, pf, jobErr := u.job.State()
	running := u.job.Running()

	layout.Inset{Top: pad, Bottom: pad, Left: pad, Right: pad}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.H5(u.theme, "微信聊天记录一键导出").Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: gap}.Layout),
			// Data root row (account dirs auto-detected beneath it).
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Body2(u.theme, "微信数据根目录（其下为各账号目录）").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(component.Rect{Color: grayBg}.Layout),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(8)).Layout(gtx, material.Editor(u.theme, &u.rootEditor, "例如 E:\\软件\\xwechat_files").Layout)
									}),
								)
							}),
							layout.Rigid(layout.Spacer{Width: gap}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Button(u.theme, &u.browseRootBtn, "浏览…").Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: gap}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Button(u.theme, &u.detectBtn, "检测账号").Layout(gtx)
							}),
						)
					}),
				)
			}),
			// Account selector: explicit choice when multiple; never a guess.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(u.accounts) == 0 {
					return material.Body2(u.theme, "未检测到账号：请确认数据根目录后点「检测账号」").Layout(gtx)
				}
				var rows []layout.FlexChild
				rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Body2(u.theme, fmt.Sprintf("检测到 %d 个账号：", len(u.accounts))).Layout(gtx)
				}))
				for i := range u.accounts {
					a := u.accounts[i]
					rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.RadioButton(u.theme, &u.accountEnum, a.Dir, a.WxID+"　"+a.Dir).Layout(gtx)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
			}),
			layout.Rigid(layout.Spacer{Height: gap}.Layout),
			// Export directory row.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Body2(u.theme, "导出目录（自动记忆上次有效路径）").Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return layout.Stack{}.Layout(gtx,
									layout.Expanded(component.Rect{Color: grayBg}.Layout),
									layout.Stacked(func(gtx layout.Context) layout.Dimensions {
										return layout.UniformInset(unit.Dp(8)).Layout(gtx, material.Editor(u.theme, &u.dirEditor, "选择或输入导出目录").Layout)
									}),
								)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Width: gap}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return material.Button(u.theme, &u.browseBtn, "浏览…").Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: gap}.Layout),
			// Action row.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if running {
							b := material.Button(u.theme, &u.startBtn, "导出中…")
							b.Background = color.NRGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xFF}
							return b.Layout(gtx)
						}
						return material.Button(u.theme, &u.startBtn, "一键导出").Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: gap}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !running {
							return layout.Dimensions{}
						}
						b := material.Button(u.theme, &u.cancelBtn, "取消")
						b.Background = red
						return b.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: gap}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if rep == nil || rep.OutDir == "" {
							return layout.Dimensions{}
						}
						return material.Button(u.theme, &u.openBtn, "打开导出文件夹").Layout(gtx)
					}),
				)
			}),
			layout.Rigid(layout.Spacer{Height: gap}.Layout),
			// Needs-capture approval banner.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if stage != guicore.StageNeedsCapture {
					return layout.Dimensions{}
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						l := material.Body1(u.theme, fmt.Sprintf("有 %d 个数据库缺少可用密钥材料。需要关闭并重新启动微信、完成登录后自动捕获；期间微信会短暂退出。", needsCount(pf)))
						l.Color = blue
						return l.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: gap}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								b := material.Button(u.theme, &u.approveBtn, "重启微信并获取材料")
								b.Background = blue
								return b.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: gap}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return material.Button(u.theme, &u.cancelCap, "取消").Layout(gtx)
							}),
						)
					}),
				)
			}),
			// Status line.
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if stage != guicore.StageRecovering {
					return layout.Dimensions{}
				}
				return material.Button(u.theme, &u.recoveryBtn, "重试恢复（不会重新采集）").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				msg := u.statusLine
				col := grayText
				if jobErr != nil {
					msg = "失败：" + jobErr.Error()
					col = red
				} else if stage == guicore.StageDone && rep != nil {
					msg = fmt.Sprintf("完成（%s）：%d 条消息，%d 个会话 → %s", rep.Status, rep.TotalMessages, rep.TotalConversations, rep.OutDir)
					if rep.Status == "complete" {
						col = green
					} else {
						col = blue
					}
				} else if running {
					msg = stageLabel(stage)
					col = blue
				}
				if msg == "" {
					return layout.Dimensions{}
				}
				l := material.Body1(u.theme, msg)
				l.Color = col
				return l.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: gap}.Layout),
			// Progress log.
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				if len(events) == 0 {
					return material.Body2(u.theme, "就绪。点击「一键导出」开始。").Layout(gtx)
				}
				var sb strings.Builder
				max := len(events)
				if max > 200 {
					events = events[max-200:]
				}
				for _, ev := range events {
					fmt.Fprintf(&sb, "%s  %s\n", ev.Time.Format("15:04:05"), ev.Message)
				}
				ed := material.Body2(u.theme, sb.String())
				return layout.Stack{}.Layout(gtx,
					layout.Expanded(component.Rect{Color: grayBg}.Layout),
					layout.Stacked(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(8)).Layout(gtx, ed.Layout)
					}),
				)
			}),
		)
	})
}

var grayText = color.NRGBA{R: 0x66, G: 0x66, B: 0x66, A: 0xFF}

func needsCount(pf *guicore.Preflight) int {
	if pf == nil {
		return 0
	}
	return len(pf.NeedsCapture)
}

func stageLabel(s guicore.Stage) string {
	switch s {
	case guicore.StageValidate:
		return "正在校验目录…"
	case guicore.StagePreflight:
		return "正在识别数据与材料…"
	case guicore.StageNeedsCapture:
		return "等待确认重启微信…"
	case guicore.StageCapturing:
		return "正在获取密钥材料…"
	case guicore.StageRecovering:
		return "恢复未完成：仍保留控制线程和句柄，请重试恢复，勿强制退出。"
	case guicore.StageExporting:
		return "正在导出…"
	default:
		return string(s)
	}
}

// openFolder opens the directory in the OS file manager.
func openFolder(dir string) {
	openFolderImpl(dir)
}
