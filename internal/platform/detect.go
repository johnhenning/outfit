package platform

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type Target struct {
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	Family      string `json:"family"`
	Environment string `json:"environment"`
	Desktop     bool   `json:"desktop"`
}
type Facts struct {
	OS, Arch, Release, Kernel string
	Desktop                   bool
}

func Detect(f Facts) (Target, error) {
	t := Target{OS: f.OS, Arch: f.Arch, Environment: "native", Desktop: f.Desktop}
	if t.Arch != "arm64" && t.Arch != "amd64" {
		return t, fmt.Errorf("unsupported architecture %q: supported architectures are amd64 and arm64", t.Arch)
	}
	switch f.OS {
	case "darwin":
		t.Family = "brew"
		t.Desktop = true
	case "windows":
		t.Family = "winget"
		t.Desktop = true
	case "linux":
		values := map[string]string{}
		for _, line := range strings.Split(f.Release, "\n") {
			k, v, ok := strings.Cut(line, "=")
			if ok {
				values[k] = strings.Trim(v, "\"'")
			}
		}
		ids := " " + values["ID"] + " " + values["ID_LIKE"] + " "
		switch {
		case strings.Contains(ids, " debian ") || strings.Contains(ids, " ubuntu "):
			t.Family = "apt"
		case strings.Contains(ids, " fedora ") || strings.Contains(ids, " rhel "):
			t.Family = "dnf"
		default:
			return t, fmt.Errorf("unsupported Linux distribution %q: use Debian/Ubuntu or Fedora", values["ID"])
		}
		if strings.Contains(strings.ToLower(f.Kernel), "microsoft") {
			t.Environment = "wsl"
			t.Desktop = false
		}
	default:
		return t, fmt.Errorf("unsupported operating system %q", f.OS)
	}
	return t, nil
}
func Current() (Target, error) {
	release, _ := os.ReadFile("/etc/os-release")
	kernel, _ := os.ReadFile("/proc/sys/kernel/osrelease")
	return Detect(Facts{OS: runtime.GOOS, Arch: runtime.GOARCH, Release: string(release), Kernel: string(kernel), Desktop: os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""})
}
func (t Target) Key() string {
	switch t.Family {
	case "brew":
		return "macos"
	case "apt":
		return "debian"
	case "dnf":
		return "fedora"
	case "winget":
		return "windows"
	}
	return t.OS
}
