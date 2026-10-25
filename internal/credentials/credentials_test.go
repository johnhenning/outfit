package credentials

import (
	"github.com/johnhenning/outfit/internal/platform"
	"strings"
	"testing"
)

func TestHeadlessDefaults(t *testing.T) {
	steps := Guidance(platform.Target{OS: "linux", Desktop: false})
	if len(steps) == 0 || !strings.Contains(strings.Join(steps, " "), "headless") {
		t.Fatal("missing headless path")
	}
}
func TestSecretReferences(t *testing.T) {
	for _, raw := range []string{"API_TOKEN=op://dev/service/token\n", "# ignored\nTOKEN=op://dev/service/key\n"} {
		if ValidateEnv([]byte(raw)) != nil {
			t.Fatal("valid reference rejected")
		}
	}
	for _, raw := range []string{"TOKEN=plain-secret\n", "TOKEN=$(echo bad)\n", "TOKEN=op://\n"} {
		if ValidateEnv([]byte(raw)) == nil {
			t.Fatal("plaintext or invalid secret accepted")
		}
	}
}
