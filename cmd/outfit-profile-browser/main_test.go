package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWrongEngineHashFailsBeforeDownloadOrExecution(t *testing.T) {
	root := t.TempDir()
	engine := filepath.Join(root, "untrusted-engine")
	if err := os.WriteFile(engine, []byte("not a trusted executable"), 0700); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, "release-lock.json")
	data := map[string]any{"version": "v0.1.0", "profiles": map[string]string{"version": "v0.1.0", "url": "https://example.invalid/profiles.tar.gz", "sha256": strings.Repeat("0", 64)}, "engines": map[string]string{runtime.GOOS + "_" + runtime.GOARCH: strings.Repeat("0", 64)}}
	raw, _ := json.Marshal(data)
	if err := os.WriteFile(lock, raw, 0600); err != nil {
		t.Fatal(err)
	}
	err := run(lock, "unused-template", filepath.Join(root, "index.html"), engine)
	if err == nil || !strings.Contains(err.Error(), "CLI checksum mismatch") {
		t.Fatal("must reject the untrusted engine first", err)
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); !os.IsNotExist(err) {
		t.Fatal("invalid source must not publish a page")
	}
}

func TestLockCannotReadLocalSourceAsPublishedData(t *testing.T) {
	root := t.TempDir()
	lock := filepath.Join(root, "release-lock.json")
	if err := os.WriteFile(lock, []byte(`{"version":"v0.1.0","profiles":{"version":"v0.1.0","url":"/private/local/data","sha256":"abc"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	err := run(lock, "unused-template", filepath.Join(root, "index.html"), "unused-engine")
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatal("must reject local sources before loading them", err)
	}
}
