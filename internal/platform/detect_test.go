package platform

import "testing"

func TestDetectTargets(t *testing.T) {
	for _, tc := range []struct {
		os, arch, release, kernel, family, env string
		fail                                   bool
	}{
		{"darwin", "arm64", "", "", "brew", "native", false},
		{"darwin", "amd64", "", "", "brew", "native", false},
		{"linux", "amd64", "ID=ubuntu\nID_LIKE=debian", "", "apt", "native", false},
		{"linux", "arm64", "ID=debian", "", "apt", "native", false},
		{"linux", "amd64", "ID=fedora", "", "dnf", "native", false},
		{"linux", "amd64", "ID=ubuntu", "microsoft-standard-WSL2", "apt", "wsl", false},
		{"windows", "amd64", "", "", "winget", "native", false},
		{"linux", "amd64", "ID=unknown", "", "", "", true},
		{"darwin", "riscv64", "", "", "", "", true},
	} {
		t.Run(tc.os+tc.release+tc.arch+tc.kernel, func(t *testing.T) {
			got, err := Detect(Facts{OS: tc.os, Arch: tc.arch, Release: tc.release, Kernel: tc.kernel})
			if tc.fail {
				if err == nil {
					t.Fatal("unsupported target accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Family != tc.family || got.Environment != tc.env {
				t.Fatalf("unexpected target: %+v", got)
			}
		})
	}
}
