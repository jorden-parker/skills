// Package cli defines the skills commands.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/jorden-parker/skills/internal/skills"
	"github.com/jorden-parker/skills/internal/tui"
)

// exitError carries an exit status for a command that has already written
// its own output. fang prints nothing for it.
type exitError struct{ code int }

func (exitError) Error() string { return "" }

// Main runs the command and exits with its status.
func Main(program, version string) {
	err := fang.Execute(context.Background(), Root(program), fang.WithVersion(version), fang.WithErrorHandler(handleError))
	if err == nil {
		return
	}
	var ex exitError
	if errors.As(err, &ex) {
		os.Exit(ex.code)
	}
	os.Exit(1)
}

// handleError keeps the plain `error: …` line the shell script printed when
// stderr is not a terminal, and stays silent for an exitError.
func handleError(w io.Writer, styles fang.Styles, err error) {
	var ex exitError
	if errors.As(err, &ex) {
		return
	}
	// fang hands over a colour-profile wrapper, so ask the real stderr.
	if !term.IsTerminal(os.Stderr.Fd()) {
		fmt.Fprintf(w, "error: %v\n", err)
		return
	}
	fang.DefaultErrorHandler(w, styles, err)
}

// Root returns the root command. program is the name shown in help.
func Root(program string) *cobra.Command {
	var repoFlag string
	root := &cobra.Command{
		Use:   program,
		Short: "Tend the skill garden: link skills from this checkout into your agents",
		Long: program + ` links the skills in a checkout of this repository into
~/.agents/skills and Claude's skills directory, and sets how visible each one is.

Run with no arguments to open the garden, an interactive picker.
The subcommands do the same work without prompts, for scripts and agents.

States: on, off, name-only, user-invocable-only
  on                    Linked. Fully visible to you and to Claude.
  off                   Not linked. Hidden everywhere.
  name-only             Linked. Claude sees the name, not the description.
  user-invocable-only   Linked. You can /invoke it; Claude cannot reach for it itself.

The checkout is found from --repo, then $SKILLS_REPO, then the path baked in
at build time, then ~/src/skills.

Adding or removing a skill needs a NEW Claude Code session to take effect.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := skills.Find(repoFlag)
			if err != nil {
				return err
			}
			if f, ok := cmd.InOrStdin().(*os.File); !ok || !term.IsTerminal(f.Fd()) {
				return errors.New("not a terminal - use add, remove, set or sync instead")
			}
			return tui.Run(r, skills.DefaultPaths())
		},
	}
	root.PersistentFlags().StringVarP(&repoFlag, "repo", "C", "", "path to the skills checkout (default: $SKILLS_REPO, the built-in path, or ~/src/skills)")
	find := func() (skills.Repo, error) { return skills.Find(repoFlag) }
	root.AddCommand(listCmd(find), addCmd(find), removeCmd(find), setCmd(find), syncCmd(find), refreshCmd(find))
	return root
}

type finder func() (skills.Repo, error)
