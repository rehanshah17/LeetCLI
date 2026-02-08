package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"leetcli/internal/tester"
	"leetcli/internal/workspace"
)

var testCmd = &cobra.Command{
	Use:   "test [slug]",
	Short: "Run local tests (examples + tests.json)",
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
		p, err := a.store.GetProblem(ctx, slug)
		if err != nil {
			return err
		}
		sPath := solutionPath(a.cfg.Workspace.ProblemsDir, slug)
		cases, err := tester.LoadUserCases(filepath.Join(a.cfg.Workspace.ProblemsDir, slug))
		if err != nil {
			return err
		}
		res, err := tester.RunPython(sPath, p.ExampleTests, cases)
		if err != nil {
			return err
		}
		_ = a.store.SaveTestRun(ctx, slug, res.Passed, res.FailedCount, res.Output)
		if !res.Passed {
			_ = workspace.AppendDebugLog(a.cfg.Workspace.ProblemsDir, slug, res.Output)
			fmt.Printf("Tests failed for %s (failed=%d)\n", slug, res.FailedCount)
			if res.Output != "" {
				fmt.Println(res.Output)
			}
			return fmt.Errorf("test failure")
		}
		fmt.Printf("Tests passed for %s\n", slug)
		return nil
	},
}
