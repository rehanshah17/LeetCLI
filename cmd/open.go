package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"leetcli/internal/workspace"
)

var openDir bool

var openCmd = &cobra.Command{
	Use:   "open [slug]",
	Short: "Open solution.py in $EDITOR (or full problem directory)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		a, err := loadApp(ctx)
		if err != nil {
			return err
		}
		defer a.close()

		slug := ""
		if len(args) == 1 {
			slug = args[0]
		}
		slug, err = problemSlugFromArgOrCurrent(ctx, a, slug)
		if err != nil {
			return err
		}
		if err := syncMeta(ctx, a, slug); err != nil {
			return err
		}
		path := solutionPath(a.cfg.Workspace.ProblemsDir, slug)
		if openDir {
			path = filepath.Join(a.cfg.Workspace.ProblemsDir, slug)
		}
		fmt.Printf("Opening %s\n", path)
		return workspace.OpenInEditor(path)
	},
}

func init() {
	openCmd.Flags().BoolVar(&openDir, "dir", false, "open full problem directory instead of solution.py")
}
