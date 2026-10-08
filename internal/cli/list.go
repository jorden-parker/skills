package cli

import (
	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/skills"
)

func listCmd(find finder) *cobra.Command {
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
			paths := skills.DefaultPaths()
			entries, err := skills.LoadStates(r, paths)
			if err != nil {
				return err
			}
			newPrinter(cmd.OutOrStdout()).list(r, paths, entries)
			return nil
		},
	}
}
