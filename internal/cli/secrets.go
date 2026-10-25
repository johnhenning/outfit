package cli

import (
	"fmt"
	"github.com/johnhenning/outfit/internal/credentials"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
)

func addSecrets(root *cobra.Command) {
	var env string
	sc := &cobra.Command{Use: "secrets", Short: "Use 1Password references without saving secret values"}
	run := &cobra.Command{Use: "run --env-file refs.env -- COMMAND [ARGS...]", Args: cobra.MinimumNArgs(1), RunE: func(c *cobra.Command, args []string) error {
		if env == "" {
			return fmt.Errorf("--env-file is required")
		}
		b, err := os.ReadFile(env)
		if err != nil {
			return err
		}
		if err = credentials.ValidateEnv(b); err != nil {
			return err
		}
		a := append([]string{"run", "--env-file", env, "--"}, args...)
		sub := exec.CommandContext(c.Context(), "op", a...)
		sub.Stdin = c.InOrStdin()
		sub.Stdout = c.OutOrStdout()
		sub.Stderr = c.ErrOrStderr()
		return sub.Run()
	}}
	run.Flags().StringVar(&env, "env-file", "", "File containing only op:// references")
	sc.AddCommand(run)
	root.AddCommand(sc)
}
