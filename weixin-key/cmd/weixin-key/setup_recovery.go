package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"weixin-key/internal/wxkey"
)

type setupCall func(context.Context, *wxkey.CaptureRecovery) (*wxkey.SetupResult, string, error)

func runSetupWithSignals(stderr io.Writer) (*wxkey.SetupResult, string, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals) // only AFTER cleanup and worker join
	return setupWithSignals(signals, stderr, wxkey.RunSetupWithRecovery)
}

func setupWithSignals(signals <-chan os.Signal, stderr io.Writer, setup setupCall) (*wxkey.SetupResult, string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recovery := wxkey.NewCaptureRecovery(nil)
	type outcome struct {
		result *wxkey.SetupResult
		text   string
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		r, text, err := setup(ctx, recovery)
		done <- outcome{r, text, err}
	}()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	lastMessage := ""
	for {
		select {
		case result := <-done:
			return result.result, result.text, result.err
		case _, ok := <-signals:
			if !ok {
				signals = nil
				continue
			}
			cancel()
			if waiting, _ := recovery.State(); waiting {
				if recovery.Retry() {
					fmt.Fprintln(stderr, "已请求重试恢复；不会启动新的采集。")
				}
			} else {
				fmt.Fprintln(stderr, "已请求取消；正在等待安全清理，请勿关闭终端或强制结束进程。")
			}
		case <-ticker.C:
			waiting, message := recovery.State()
			if waiting && message != lastMessage {
				fmt.Fprintln(stderr, message)
				fmt.Fprintln(stderr, "恢复期间可按 Ctrl+C 请求重试恢复；工具不会在恢复未完成时主动退出。")
			}
			lastMessage = message
		}
	}
}
