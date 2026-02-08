package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit [slug]",
	Short: "Submit solution.py to LeetCode (Python3)",
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
		if p.QuestionID == "" {
			q, qErr := a.client().Question(ctx, slug)
			if qErr != nil {
				return qErr
			}
			p.QuestionID = q.QuestionID
		}

		code, err := os.ReadFile(solutionPath(a.cfg.Workspace.ProblemsDir, slug))
		if err != nil {
			return err
		}
		res, err := a.client().Submit(ctx, slug, p.QuestionID, string(code))
		if err != nil {
			return err
		}
		if err := a.store.SaveSubmissionResult(ctx, slug, res.Status, res.Runtime, res.Memory); err != nil {
			return err
		}
		_ = syncMeta(ctx, a, slug)
		fmt.Printf("Submission %d: %s\n", res.SubmissionID, res.Status)
		if res.Runtime != "" || res.Memory != "" {
			fmt.Printf("Runtime: %s  Memory: %s\n", res.Runtime, res.Memory)
		}
		return nil
	},
}
