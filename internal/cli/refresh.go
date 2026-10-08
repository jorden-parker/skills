package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/refresh"
	"github.com/jorden-parker/skills/internal/skills"
)

func refreshCmd(find finder) *cobra.Command {
	var check bool
	c := &cobra.Command{
		Use:   "refresh [NAME...]",
		Short: "Replace installed copies of skills with links to this checkout",
		Long: `Keep installed skills linked to this checkout.

Looks in ~/.agents/skills, $CODEX_HOME/skills and Claude's skills directory.
A copy is recognised when its SKILL.md matches the current source or a
version in this checkout's Git history. Recognised copies are backed up under
skill-backups beside the installation's skills directory, then replaced with
links. Unrecognised copies, including broken links, are left untouched and
reported for review.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := find()
			if err != nil {
				return err
			}
			if unknown := refresh.Unknown(r, args); len(unknown) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: unknown skills: %s\n", strings.Join(unknown, ", "))
				return exitError{2}
			}
			results, failed := refresh.Run(r, skills.DefaultPaths(), args, check, nil)
			out, errOut := cmd.OutOrStdout(), cmd.ErrOrStderr()
			for _, res := range results {
				switch res.Status {
				case refresh.Current:
					fmt.Fprintf(out, "current  %s\n", res.Target)
				case refresh.Refresh:
					fmt.Fprintf(out, "refresh  %s\n", res.Target)
				case refresh.Review:
					fmt.Fprintf(errOut, "review   %s: not a recognised copy; left untouched\n", res.Target)
				case refresh.Linked:
					fmt.Fprintf(out, "linked   %s -> %s\nbackup   %s\n", res.Target, res.Source, res.Backup)
				case refresh.Error:
					fmt.Fprintf(errOut, "error    %s: %v\n", res.Target, res.Err)
				}
			}
			if failed {
				return exitError{1}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&check, "check", false, "report without changing files; exit 1 if refresh or review is needed")
	return c
}
