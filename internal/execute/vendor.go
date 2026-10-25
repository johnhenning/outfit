package execute

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Vendor struct {
	URL         string `json:"url"`
	SHA256      string `json:"sha256"`
	Version     string `json:"version"`
	Interpreter string `json:"interpreter"`
}

//go:embed vendors.json
var vendorData []byte

func FetchInstaller(ctx context.Context, client *http.Client, v Vendor, dir string) (string, error) {
	if !strings.HasPrefix(v.URL, "https://") {
		return "", fmt.Errorf("vendor installer requires HTTPS")
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	cc := *client
	cc.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || len(via) > 5 {
			return fmt.Errorf("unsafe vendor redirect")
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", v.URL, nil)
	if err != nil {
		return "", err
	}
	res, err := cc.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("vendor returned HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, (4<<20)+1))
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	if len(b) > 4<<20 || hex.EncodeToString(h[:]) != v.SHA256 {
		return "", fmt.Errorf("pinned vendor installer checksum mismatch; update Outfit before retrying")
	}
	f, err := os.CreateTemp(dir, "outfit-installer-")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err = f.Write(b); err == nil {
		err = f.Chmod(0600)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}
func installVendor(ctx context.Context, id string, r Runner) error {
	vendors := map[string]Vendor{}
	if err := json.Unmarshal(vendorData, &vendors); err != nil {
		return err
	}
	v, ok := vendors[id]
	if !ok {
		return fmt.Errorf("unknown built-in vendor installer %s", id)
	}
	dir, err := os.MkdirTemp("", "outfit-vendor-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	script, err := FetchInstaller(ctx, nil, v, dir)
	if err != nil {
		return err
	}
	args := []string{script}
	if id == "rust" {
		args = append(args, "-y", "--no-modify-path", "--profile", "minimal")
	}
	if id == "uv" {
		args = append(args, "--no-modify-path")
	}
	_, err = r.Run(ctx, Command{v.Interpreter, args, false})
	if err == nil {
		RefreshPath()
	}
	return err
}

func ToolPaths(home, localAppData, system string) []string {
	p := []string{filepath.Join(home, ".local", "bin"), filepath.Join(home, ".cargo", "bin")}
	if system == "windows" {
		p = append(p, filepath.Join(localAppData, "Microsoft", "WinGet", "Links"))
	} else if system == "darwin" {
		p = append(p, "/opt/homebrew/bin", "/usr/local/bin")
	}
	return p
}
func RefreshPath() {
	home, _ := os.UserHomeDir()
	paths := ToolPaths(home, os.Getenv("LOCALAPPDATA"), runtime.GOOS)
	os.Setenv("PATH", strings.Join(paths, string(os.PathListSeparator))+string(os.PathListSeparator)+os.Getenv("PATH"))
}
