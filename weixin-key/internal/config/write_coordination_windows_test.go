//go:build windows

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func init() {
	configWriterMutexPrefix = fmt.Sprintf(`Local\wxcli-config-test-%d-`, os.Getpid())
}

func TestInitializationBusyDoesNotPinDirectoriesOfActiveWriter(t *testing.T) {
	path, source := metadataInitFixture(t)
	writerReady, writerRelease := make(chan struct{}), make(chan struct{})
	writerResult := make(chan error, 1)
	defer close(writerRelease)
	go func() {
		writerResult <- withConfigWriteLock(path, func(root *os.Root, base string) error {
			if err := root.WriteFile("prior.tmp", []byte("synthetic prior metadata"), 0o600); err != nil {
				return err
			}
			close(writerReady)
			<-writerRelease
			return root.Rename("prior.tmp", "writer-finished.json")
		})
	}()
	select {
	case <-writerReady:
	case err := <-writerResult:
		t.Fatal("writer setup failed", err)
	case <-time.After(5 * time.Second):
		t.Fatal("writer did not start")
	}
	proof, err := InitializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic")
	if !errors.Is(err, ErrConfigBusy) || proof.Applied {
		t.Fatal("initialization did not fail before directory pins", err)
	}
	// Let the holder publish while another initialization is being rejected.
	writerRelease <- struct{}{}
	select {
	case err := <-writerResult:
		if err != nil {
			t.Fatal("initialization interfered with active writer publication", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("writer publication did not finish")
	}
}

func TestInitializationReleasesPinsBeforeNextWriter(t *testing.T) {
	path, source := metadataInitFixture(t)
	created, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	defer close(release)
	go func() {
		_, err := initializeMetadataAtPath(context.Background(), path, source, "wxid_synthetic",
			initializationHooks{beforeReadback: func(*os.Root, string) error {
				close(created)
				<-release
				return nil
			}})
		done <- err
	}()
	select {
	case <-created:
	case err := <-done:
		t.Fatal("initialization did not create metadata", err)
	case <-time.After(5 * time.Second):
		t.Fatal("initialization did not reach readback")
	}
	otherPath := filepath.Join(filepath.Dir(path), "other-config.json")
	writerDone := make(chan error, 1)
	go func() {
		writerDone <- withConfigWriteLock(otherPath, func(root *os.Root, base string) error {
			if err := root.WriteFile("next.tmp", []byte("synthetic next metadata"), 0o600); err != nil {
				return err
			}
			return root.Rename("next.tmp", base)
		})
	}()
	release <- struct{}{}
	if err := <-done; err != nil {
		t.Fatal("initialization failed", err)
	}
	select {
	case err := <-writerDone:
		if err != nil {
			t.Fatal("next writer entered before pins were released", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("next writer remained blocked")
	}
}
