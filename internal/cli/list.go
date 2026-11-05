package cli

import (
	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/skills"
)

func listCmd(find finder, pathsFor pathsFinder) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show each skill and its current state",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := find()
			if err != nil {
				return err
			}
			paths, err := pathsFor()
			if err != nil {
				return err
			}
			entries, err := skills.LoadStates(r, paths)
			if err != nil {
				return err
			}
			newPrinter(cmd.OutOrStdout()).list(r, paths, entries)
			return nil
		},
	}
}
