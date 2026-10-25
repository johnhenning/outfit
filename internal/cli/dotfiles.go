package cli

import (
	"fmt"
	"github.com/johnhenning/outfit/internal/dotfiles"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/spf13/cobra"
	"path/filepath"
	"time"
)

func addDotfiles(root *cobra.Command, o *options) {
	var apply, replace bool
	dc := &cobra.Command{Use: "dotfiles", Short: "Review or apply explicitly managed configuration files", RunE: func(c *cobra.Command, _ []string) error {
		cat, clean, err := o.load(c.Context())
		if err != nil {
			return err
		}
		defer clean()
		cfg, err := o.selection()
		if err != nil {
			return err
		}
		t, err := o.platform()
		if err != nil {
			return err
		}
		p, err := plan.Resolve(cat, t, cfg)
		if err != nil {
			return err
		}
		changes, err := dotfiles.Preview(cat, p, o.home)
		if err != nil {
			return err
		}
		for _, ch := range changes {
			status := "new"
			if ch.Same {
				status = "unchanged"
			} else if ch.Conflict {
				status = "conflict"
			}
			fmt.Fprintf(c.OutOrStdout(), "%s [%s]\n", ch.Target, status)
			if ch.Private {
				fmt.Fprintln(c.OutOrStdout(), "  Private configuration: content hidden; existing changes require a manual merge.")
			} else {
				fmt.Fprintf(c.OutOrStdout(), "--- existing\n%s\n+++ proposed\n%s\n", ch.Before, ch.After)
			}
		}
		if !apply {
			return nil
		}
		if !o.yes {
			return fmt.Errorf("dotfiles --apply requires --yes after reviewing changes")
		}
		if err = actualTarget(p); err != nil {
			return err
		}
		backup := filepath.Join(o.state, "backups", fmt.Sprint(time.Now().UnixNano()))
		results, err := dotfiles.Apply(changes, o.home, backup, replace)
		if err != nil {
			return err
		}
		for _, r := range results {
			fmt.Fprintf(c.OutOrStdout(), "%s %s\n", r.Status, r.ActionID)
		}
		return nil
	}}
	dc.Flags().BoolVar(&apply, "apply", false, "Apply after reviewing the proposed files")
	dc.Flags().BoolVar(&replace, "replace", false, "Back up and replace conflicting nonprivate configs")
	root.AddCommand(dc)
}
