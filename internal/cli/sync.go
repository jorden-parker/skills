package cli

import (
	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/links"
	"github.com/jorden-parker/skills/internal/settings"
	"github.com/jorden-parker/skills/internal/skills"
)

func syncCmd(find finder) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Repair links to an old checkout and drop links whose skill left the repo",
		Long: `Repair links that point at an old checkout and drop links whose skill
left the repo. Fill missing Claude links for enabled skills. Enables no new
skills. No prompts.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := find()
			if err != nil {
				return err
			}
			paths := skills.DefaultPaths()
			entries, err := skills.LoadStates(r, paths)
			if err != nil {
				return err
			}
			p := newPrinter(cmd.OutOrStdout())
			p.blank()
			p.bold("Syncing links in " + paths.LinkDir + " and " + paths.ClaudeLinkDir)
			for _, e := range entries {
				if e.State == skills.Off {
					continue
				}
				blocked, err := links.Link(r.Source(e.Name), e.Name, paths.LinkDirs(), p.warn)
				if err != nil {
					return err
				}
				if !blocked {
					p.printf("  synced %s\n", e.Name)
				}
			}
			removed, err := links.Prune(r.SrcDir, paths.LinkDirs())
			for _, target := range removed {
				p.printf("  removed %s (no longer in the repo)\n", target)
			}
			if err != nil {
				return err
			}
			if err := settings.WriteOverrides(paths.Settings, overrides(entries)); err != nil {
				return err
			}
			p.list(r, paths, entries)
			p.dim(sessionHint)
			return nil
		},
	}
}
