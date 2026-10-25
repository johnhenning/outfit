package cli

import (
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func addPersonal(root *cobra.Command, o *options) {
	var repo, ref string
	pc := &cobra.Command{Use: "personalize", Short: "Authenticate and fetch an optional private profile overlay", RunE: func(c *cobra.Command, _ []string) error {
		parts := strings.Split(repo, "/")
		if len(parts) != 2 || strings.ContainsAny(repo, " :\\\t\n") || strings.HasPrefix(repo, "-") {
			return fmt.Errorf("repo must be OWNER/REPO")
		}
		if _, err := exec.LookPath("gh"); err != nil {
			return fmt.Errorf("install GitHub CLI using the public profile first, then rerun personalize")
		}
		auth := exec.CommandContext(c.Context(), "gh", "auth", "status", "--hostname", "github.com")
		if auth.Run() != nil {
			login := exec.CommandContext(c.Context(), "gh", "auth", "login", "--hostname", "github.com", "--web", "--git-protocol", "https")
			login.Stdin = c.InOrStdin()
			login.Stdout = c.OutOrStdout()
			login.Stderr = c.ErrOrStderr()
			if err := login.Run(); err != nil {
				return fmt.Errorf("GitHub sign-in did not complete")
			}
		}
		dest := filepath.Join(o.state, "personal", strings.ReplaceAll(repo, "/", "-"))
		if _, err := os.Stat(dest); err == nil {
			return fmt.Errorf("personal source exists at %s; update it explicitly before reuse", dest)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
			return err
		}
		args := []string{"repo", "clone", repo, dest, "--", "--depth", "1"}
		if ref != "" {
			args = append(args, "--branch", ref)
		}
		clone := exec.CommandContext(c.Context(), "gh", args...)
		clone.Stdout = c.ErrOrStderr()
		clone.Stderr = c.ErrOrStderr()
		if err := clone.Run(); err != nil {
			return fmt.Errorf("private profile clone failed")
		}
		if _, err := catalog.Load(dest); err != nil {
			return err
		}
		fmt.Fprintf(c.OutOrStdout(), "Personal overlay ready: %s\nUse --personal %q --profile YOUR_PROFILE with setup or plan.\n", dest, dest)
		return nil
	}}
	pc.Flags().StringVar(&repo, "repo", "johnhenning/outfit-personal", "Private GitHub repository")
	pc.Flags().StringVar(&ref, "ref", "main", "Branch or tag")
	root.AddCommand(pc)
}
