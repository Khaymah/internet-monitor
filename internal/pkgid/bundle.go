package pkgid

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const bundleMarker = ".app/Contents/MacOS/"

func InBundle(exe string) bool {
	return strings.Contains(filepath.ToSlash(exe), bundleMarker)
}

func Bundled() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return InBundle(exe)
}
