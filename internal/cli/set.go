package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/links"
	"github.com/jorden-parker/skills/internal/settings"
	"github.com/jorden-parker/skills/internal/skills"
)

func addCmd(find finder) *cobra.Command {
	return &cobra.Command{
		Use:   "add NAME...",
		Short: "Link skills into ~/.agents/skills and Claude's skills directory",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return applyStates(cmd, find, withState(args, skills.On))
		},
	}
}

func removeCmd(find finder) *cobra.Command {
	return &cobra.Command{
		Use:     "remove NAME...",
		Aliases: []string{"rm", "unlink"},
		Short:   "Remove skills' links from both directories",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return applyStates(cmd, find, withState(args, skills.Off))
		},
	}
}

func setCmd(find finder) *cobra.Command {
	return &cobra.Command{
		Use:   "set NAME STATE | NAME=STATE...",
		Short: "Set a skill's state directly: on, off, name-only or user-invocable-only",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var pairs []string
			if len(args) == 2 && !strings.Contains(args[0], "=") && !strings.Contains(args[1], "=") {
				pairs = []string{args[0] + "=" + args[1]}
			} else {
				pairs = args
			}
			return applyStates(cmd, find, pairs)
		},
	}
}

func withState(names []string, st skills.State) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n + "=" + string(st)
	}
	return out
}

// applyStates validates every NAME=STATE pair, then links, writes the
// overrides and prints the list with what changed.
func applyStates(cmd *cobra.Command, find finder, pairs []string) error {
	r, err := find()
	if err != nil {
		return err
	}
	paths := skills.DefaultPaths()
	before, err := skills.LoadStates(r, paths)
	if err != nil {
		return err
	}
	staged := append([]skills.Entry(nil), before...)
	for _, pair := range pairs {
		name, value, ok := strings.Cut(pair, "=")
		if !ok || name == "" {
			return fmt.Errorf("expected NAME=STATE, got: %s", pair)
		}
		if !r.Has(name) {
			return fmt.Errorf("no such skill: %s", name)
		}
		st, err := skills.ParseState(value)
		if err != nil {
			return err
		}
		for i := range staged {
			if staged[i].Name == name {
				staged[i].State = st
			}
		}
	}
	p := newPrinter(cmd.OutOrStdout())
	after, err := save(r, paths, staged, p.warn)
	if err != nil {
		return err
	}
	p.list(r, paths, after)
	p.changes(before, after)
	p.dim(sessionHint)
	return nil
}

// save applies the staged states to the filesystem and settings.json.
func save(r skills.Repo, paths skills.Paths, staged []skills.Entry, warn links.Warn) ([]skills.Entry, error) {
	after, err := links.Apply(r, paths, staged, warn)
	if err != nil {
		return nil, err
	}
	return after, settings.WriteOverrides(paths.Settings, overrides(after))
}

func overrides(entries []skills.Entry) map[string]string {
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.Name] = string(e.State)
	}
	return out
}
