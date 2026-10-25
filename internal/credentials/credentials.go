package credentials

import (
	"fmt"
	"github.com/johnhenning/outfit/internal/platform"
	"regexp"
	"strings"
)

func Guidance(t platform.Target) []string {
	if !t.Desktop {
		return []string{"On a headless node, use op signin interactively or an explicitly scoped service account. The desktop SSH agent is unavailable without the 1Password desktop app.", "Prefer AWS/Azure SSO. Do not copy cached sessions or private keys from another machine."}
	}
	steps := []string{"Sign in to the 1Password desktop app, then enable Settings > Developer > Integrate with 1Password CLI.", "Enable its SSH agent, import/generate an SSH Key in the app, and use public-key selectors per SSH host.", "Keep secrets in op:// references; use outfit secrets run --env-file refs.env -- command to inject only into that process."}
	if t.OS == "windows" {
		steps = append(steps, "For WSL's ssh.exe integration, manage the Windows host SSH config and enable Windows Hello in 1Password. Native tmux belongs in WSL.")
	} else {
		steps = append(steps, "Use 1Password's OS-specific agent socket in a private SSH include; validate with ssh -G before connecting.")
	}
	return steps
}

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidateEnv(b []byte) error {
	for n, line := range strings.Split(string(b), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		k, v, ok := strings.Cut(s, "=")
		if !ok || !envName.MatchString(k) || !strings.HasPrefix(v, "op://") || len(strings.Split(strings.TrimPrefix(v, "op://"), "/")) < 3 || strings.ContainsAny(v, " \t$`\"'") {
			return fmt.Errorf("line %d must be NAME=op://vault/item/field; plaintext secrets are not accepted", n+1)
		}
	}
	return nil
}
