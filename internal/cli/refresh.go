package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/refresh"
	"github.com/jorden-parker/skills/internal/skills"
)

func refreshCmd(find finder, pathsFor pathsFinder) *cobra.Command {
	var check bool
	c := &cobra.Command{
		Use:   "refresh [NAME...]",
		Short: "Replace installed copies of skills with links to this checkout",
		Long: `Keep installed skills linked to this checkout.

Looks in ~/.agents/skills, $CODEX_HOME/skills and Claude's skills directory.
A copy is recognised when its SKILL.md matches the current source or a
version in this checkout's Git history. Recognised copies are backed up under
skill-backups beside the installation's skills directory, then replaced with
links. A link to the old flat skills/<name> path is relinked to the skill's
author folder. Unrecognised copies, including other broken links, are left
untouched and reported for review.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := find()
			if err != nil {
				return err
			}
			paths, err := pathsFor()
			if err != nil {
				return err
			}
			if paths.Scope == skills.ProjectScope {
				return errors.New("refresh works on global installs only; drop --scope project")
			}
			ids, ok := resolveAll(r, args, cmd.ErrOrStderr())
			if !ok {
				return exitError{2}
			}
			results, failed := refresh.Run(r, skills.DefaultPaths(), ids, check, nil)
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

// resolveAll turns each argument into a skill ID, reporting every one that
// does not resolve.
func resolveAll(r skills.Repo, args []string, errOut io.Writer) ([]string, bool) {
	ids := make([]string, 0, len(args))
	ok := true
	for _, arg := range args {
		id, err := r.Resolve(arg)
		if err != nil {
			fmt.Fprintf(errOut, "error: %v\n", err)
			ok = false
			continue
		}
		ids = append(ids, id)
	}
	return ids, ok
}
