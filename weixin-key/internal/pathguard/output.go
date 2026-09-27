// Package pathguard shares physical output-path validation between CLI and GUI.
package pathguard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveOutput resolves existing ancestors (including intermediate links)
// before adding nonexistent suffixes. It performs no writes. File identity
// checks also catch Windows short-name aliases of the source directory.
func ResolveOutput(output, source string) (string, error) {
	if strings.TrimSpace(output) == "" {
		return "", errors.New("empty export directory")
	}
	out, err := resolveFutureDir(output)
	if err != nil {
		return "", err
	}
	if source == "" {
		return out, nil
	}
	src, err := resolveFutureDir(source)
	if err != nil {
		return "", err
	}
	if within(out, src) {
		return "", errors.New("export directory must not be the source directory or inside it")
	}
	srcInfo, err := os.Stat(src)
	if err != nil {
		return "", fmt.Errorf("source directory: %w", err)
	}
	if !srcInfo.IsDir() {
		return "", errors.New("source is not a directory")
	}
	for p := out; ; p = filepath.Dir(p) {
		info, err := os.Stat(p)
		if err == nil && os.SameFile(srcInfo, info) {
			return "", errors.New("export directory aliases the source directory or lies inside it")
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return out, nil
}

func resolveFutureDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	p := abs
	var suffix []string
	for {
		info, err := os.Lstat(p)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(p)
			if err != nil {
				return "", err
			}
			if !info.IsDir() {
				info, err = os.Stat(resolved)
				if err != nil {
					return "", err
				}
				if !info.IsDir() {
					return "", errors.New("export path ancestor is not a directory")
				}
			}
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(p)
		if parent == p {
			return "", err
		}
		suffix = append(suffix, filepath.Base(p))
		p = parent
	}
}

func within(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && filepath.IsLocal(rel)
}
