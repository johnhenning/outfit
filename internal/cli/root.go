package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/credentials"
	"github.com/johnhenning/outfit/internal/execute"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"github.com/johnhenning/outfit/internal/source"
	"github.com/johnhenning/outfit/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultSource and DefaultSHA256 are set only by a release with a pinned public bundle.
var DefaultSource, DefaultSHA256 string

type options struct {
	source, sha, personal, profile, role, target, config string
	presets, include, exclude, dotfiles                  []string
	json, yes, plain, replace                            bool
	home, state                                          string
}

func stateDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ".outfit"
	}
	return filepath.Join(d, "outfit")
}
func (o *options) load(ctx context.Context) (catalog.Catalog, func(), error) {
	src := o.source
	if src == "" {
		src = DefaultSource
	}
	sha := o.sha
	if sha == "" {
		sha = DefaultSHA256
	}
	if src == "" {
		return catalog.Catalog{}, nil, fmt.Errorf("choose a profile source with --source PATH or --source HTTPS_BUNDLE --sha256 DIGEST")
	}
	root, clean, err := (source.Loader{}).Open(ctx, src, sha)
	if err != nil {
		return catalog.Catalog{}, nil, err
	}
	c, err := catalog.Load(root)
	if err != nil {
		clean()
		return c, nil, err
	}
	if o.personal != "" {
		overlay, e := catalog.Load(o.personal)
		if e == nil {
			c, e = catalog.Merge(c, overlay)
		}
		if e != nil {
			clean()
			return c, nil, e
		}
	}
	if err = c.Validate(true); err != nil {
		clean()
		return c, nil, err
	}
	return c, clean, nil
}
func (o *options) selection() (plan.Config, error) {
	cfg := plan.Config{Profile: o.profile}
	if o.config != "" {
		b, err := os.ReadFile(o.config)
		if err != nil {
			return cfg, err
		}
		cfg, err = plan.ParseConfig(b)
		if err != nil {
			return cfg, err
		}
		if o.profile != "" {
			cfg.Profile = o.profile
		}
	}
	if cfg.Profile == "" {
		cfg.Profile = "workstation"
	}
	if o.role != "" {
		cfg.Role = o.role
	}
	cfg.Presets = append(cfg.Presets, o.presets...)
	cfg.Include = append(cfg.Include, o.include...)
	cfg.Exclude = append(cfg.Exclude, o.exclude...)
	cfg.Dotfiles = append(cfg.Dotfiles, o.dotfiles...)
	return cfg, nil
}
func (o *options) platform() (platform.Target, error) {
	if o.target == "" {
		return platform.Current()
	}
	parts := strings.Split(o.target, "-")
	if len(parts) != 2 {
		return platform.Target{}, fmt.Errorf("target must be macos|debian|fedora|windows followed by -amd64 or -arm64")
	}
	facts := platform.Facts{Arch: parts[1]}
	switch parts[0] {
	case "macos":
		facts.OS = "darwin"
	case "debian", "fedora":
		facts.OS = "linux"
		facts.Release = "ID=" + parts[0]
	case "windows":
		facts.OS = "windows"
	default:
		return platform.Target{}, fmt.Errorf("unknown target %s", parts[0])
	}
	return platform.Detect(facts)
}
func printPlan(w io.Writer, p plan.Plan, c catalog.Catalog) {
	fmt.Fprintf(w, "\nOutfit · %s · %s/%s\nSource: %s\nDigest: %s\n\n", p.Config.Profile, p.Target.Key(), p.Target.Arch, c.Name, p.BundleDigest)
	for _, a := range p.Actions {
		flag := a.Provider
		if a.Privileged {
			flag += "; privilege required"
		}
		if a.ManualReason != "" {
			flag = "manual: " + a.ManualReason
		}
		fmt.Fprintf(w, "  %-28s %s [%s]\n", a.ID, a.Description, flag)
	}
	for _, note := range p.Notes {
		fmt.Fprintf(w, "  note: %s\n", note)
	}
	fmt.Fprintln(w)
}
func encode(w io.Writer, v any) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(v)
}
func actualTarget(p plan.Plan) error {
	t, err := platform.Current()
	if err != nil {
		return err
	}
	if t.OS != p.Target.OS || t.Arch != p.Target.Arch || t.Family != p.Target.Family || t.Environment != p.Target.Environment {
		return fmt.Errorf("apply target differs from this machine; simulated targets are preview-only")
	}
	return nil
}
func confirmed(cmd *cobra.Command, o *options) bool {
	if o.yes {
		return true
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}
	fmt.Fprint(cmd.OutOrStdout(), "Apply this plan? [y/N] ")
	s, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(s), "y")
}
func New(version string) *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{Use: "outfit", Short: "Equip a machine from reusable profiles", SilenceUsage: true, SilenceErrors: true}
	f := cmd.PersistentFlags()
	f.StringVar(&o.source, "source", os.Getenv("OUTFIT_SOURCE"), "Profile bundle directory or HTTPS .tar.gz URL")
	f.StringVar(&o.sha, "sha256", os.Getenv("OUTFIT_SHA256"), "SHA-256 of remote profile archive")
	f.StringVar(&o.personal, "personal", "", "Authenticated personal overlay directory")
	f.StringVar(&o.profile, "profile", "", "Named profile")
	f.StringVar(&o.role, "role", "", "Override machine role")
	f.StringSliceVar(&o.presets, "preset", nil, "Additional software presets")
	f.StringSliceVar(&o.include, "include", nil, "Additional package IDs")
	f.StringSliceVar(&o.exclude, "exclude", nil, "Excluded package IDs")
	f.StringSliceVar(&o.dotfiles, "dotfile", nil, "Additional dotfile IDs")
	f.StringVar(&o.target, "target", "", "Simulate platform for preview, e.g. debian-amd64")
	f.StringVar(&o.config, "config", "", "Saved selection JSON")
	f.BoolVar(&o.json, "json", false, "Machine-readable output")
	f.BoolVarP(&o.yes, "yes", "y", false, "Explicitly approve apply (for unattended runs)")
	f.BoolVar(&o.plain, "plain", false, "Use plain terminal prompts")
	f.StringVar(&o.state, "state-dir", stateDir(), "Private state/journal directory")
	home, _ := os.UserHomeDir()
	f.StringVar(&o.home, "home", home, "Dotfile destination home (use a disposable path for tests)")
	cmd.AddCommand(&cobra.Command{Use: "version", Args: cobra.NoArgs, Run: func(c *cobra.Command, _ []string) {
		fmt.Fprintln(c.OutOrStdout(), "outfit "+version+" · profile schema 1 · "+runtime.GOOS+"/"+runtime.GOARCH)
	}})
	var output string
	pc := &cobra.Command{Use: "plan", Short: "Preview actions without changing the machine", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		cat, clean, err := o.load(c.Context())
		if err != nil {
			return err
		}
		defer clean()
		cfg, err := o.selection()
		if err != nil {
			return err
		}
		target, err := o.platform()
		if err != nil {
			return err
		}
		p, err := plan.Resolve(cat, target, cfg)
		if err != nil {
			return err
		}
		if output != "" {
			return execute.Save(output, p)
		}
		if o.json {
			return encode(c.OutOrStdout(), p)
		}
		printPlan(c.OutOrStdout(), p, cat)
		return nil
	}}
	pc.Flags().StringVarP(&output, "output", "o", "", "Write a reusable plan to this file")
	cmd.AddCommand(pc)
	var planFile string
	ac := &cobra.Command{Use: "apply", Short: "Apply an explicitly approved plan", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if !o.yes {
			return fmt.Errorf("apply requires --yes; use outfit setup for interactive review")
		}
		cat, clean, err := o.load(c.Context())
		if err != nil {
			return err
		}
		defer clean()
		target, err := o.platform()
		if err != nil {
			return err
		}
		var p plan.Plan
		if planFile != "" {
			b, e := os.ReadFile(planFile)
			if e != nil {
				return e
			}
			p, err = plan.Parse(b)
		} else {
			var cfg plan.Config
			cfg, err = o.selection()
			if err == nil {
				p, err = plan.Resolve(cat, target, cfg)
			}
		}
		if err != nil {
			return err
		}
		if err = plan.Validate(p, cat, target); err != nil {
			return err
		}
		if err = actualTarget(p); err != nil {
			return err
		}
		return applyPackages(c, o, cat, p)
	}}
	ac.Flags().StringVar(&planFile, "plan", "", "Saved plan JSON")
	cmd.AddCommand(ac)
	sc := &cobra.Command{Use: "setup", Short: "Choose a profile, review changes and equip this machine", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		cat, clean, err := o.load(c.Context())
		if err != nil {
			return err
		}
		defer clean()
		target, err := o.platform()
		if err != nil {
			return err
		}
		cfg, err := o.selection()
		if err != nil {
			return err
		}
		approved := o.yes
		if !o.yes && !o.plain && term.IsTerminal(int(os.Stdin.Fd())) {
			cfg, approved, err = tui.Run(cat, target, cfg)
			if err != nil {
				return err
			}
			if !approved {
				return nil
			}
		} else if !o.yes && !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("interactive setup needs a terminal; use plan for preview or --yes with explicit selections")
		}
		p, err := plan.Resolve(cat, target, cfg)
		if err != nil {
			return err
		}
		printPlan(c.OutOrStdout(), p, cat)
		if !approved && !confirmed(c, o) {
			return nil
		}
		if err = actualTarget(p); err != nil {
			return err
		}
		if err = execute.Save(filepath.Join(o.state, "selection.json"), cfg); err != nil {
			return err
		}
		return applyPackages(c, o, cat, p)
	}}
	cmd.AddCommand(sc)
	cmd.AddCommand(&cobra.Command{Use: "catalog", Short: "List available profiles and software", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		cat, clean, err := o.load(c.Context())
		if err != nil {
			return err
		}
		defer clean()
		if o.json {
			return encode(c.OutOrStdout(), cat)
		}
		fmt.Fprintln(c.OutOrStdout(), "Profiles:", strings.Join(catalog.SortedKeys(cat.Profiles), ", "))
		for _, p := range cat.Packages {
			fmt.Fprintf(c.OutOrStdout(), "%-18s %-20s %s\n", p.ID, p.Category, p.Name)
		}
		return nil
	}})
	var exportPath string
	ec := &cobra.Command{Use: "export", Short: "Save nonsecret selections", RunE: func(c *cobra.Command, _ []string) error {
		cfg, err := o.selection()
		if err != nil {
			return err
		}
		if exportPath == "" {
			return encode(c.OutOrStdout(), cfg)
		}
		return execute.Save(exportPath, cfg)
	}}
	ec.Flags().StringVarP(&exportPath, "output", "o", "", "Selection file")
	cmd.AddCommand(ec)
	cmd.AddCommand(&cobra.Command{Use: "config", Short: "Show the last saved selection", RunE: func(c *cobra.Command, _ []string) error {
		b, err := os.ReadFile(filepath.Join(o.state, "selection.json"))
		if err != nil {
			return err
		}
		cfg, err := plan.ParseConfig(b)
		if err != nil {
			return err
		}
		return encode(c.OutOrStdout(), cfg)
	}})
	cmd.AddCommand(&cobra.Command{Use: "update", Short: "Reconcile the saved selection against an explicitly chosen source", RunE: func(c *cobra.Command, _ []string) error {
		if o.config == "" {
			o.config = filepath.Join(o.state, "selection.json")
		}
		return sc.RunE(c, nil)
	}})
	cmd.AddCommand(&cobra.Command{Use: "doctor", Short: "Report prerequisites and credential setup steps", RunE: func(c *cobra.Command, _ []string) error {
		t, err := platform.Current()
		if err != nil {
			return err
		}
		for _, name := range []string{t.Family, "git", "gh", "uv", "op"} {
			_, err := exec.LookPath(name)
			status := "available"
			if err != nil {
				status = "missing"
			}
			fmt.Fprintf(c.OutOrStdout(), "%-14s %s\n", name, status)
		}
		for _, s := range credentials.Guidance(t) {
			fmt.Fprintln(c.OutOrStdout(), "- "+s)
		}
		if t.OS == "windows" {
			fmt.Fprintln(c.OutOrStdout(), "WSL: use wsl --list --verbose to choose a distro; run the Unix bootstrap inside that distro. Enable WSL explicitly with wsl --install if needed, then restart.")
		}
		return nil
	}})
	addDotfiles(cmd, o)
	addSecrets(cmd)
	addPersonal(cmd, o)
	return cmd
}
func applyPackages(c *cobra.Command, o *options, cat catalog.Catalog, p plan.Plan) error {
	notify := func(r execute.Result) {
		if !o.json {
			fmt.Fprintf(c.OutOrStdout(), "%-18s %-28s %s\n", r.Status, r.ActionID, r.Message)
		}
	}
	results, err := execute.Apply(c.Context(), p, cat, execute.RealRunner{Interactive: !o.json}, filepath.Join(o.state, "runs"), notify)
	if o.json {
		if e := encode(c.OutOrStdout(), results); e != nil {
			return e
		}
	}
	return err
}
