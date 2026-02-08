package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"leetcli/internal/store"
	"leetcli/internal/workspace"
)

var solveSlug string
var solveRandom bool
var solveDifficulty string
var solveTopic string
var solveTimer int
var solveNoTimer bool
var solveCount int

var solveCmd = &cobra.Command{
	Use:   "solve",
	Short: "Fetch a problem and prepare a local solving workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		rand.Seed(time.Now().UnixNano())
		ctx := context.Background()
		a, err := loadApp(ctx)
		if err != nil {
			return err
		}
		defer a.close()

		cli := a.client()
		slugs := make([]string, 0)

		chosenSlug := strings.TrimSpace(solveSlug)
		if chosenSlug == "" && !solveRandom {
			solveRandom = true
		}
		if chosenSlug != "" {
			slugs = append(slugs, chosenSlug)
		} else if solveCount > 1 {
			all, err := cli.ListSummaries(ctx)
			if err != nil {
				return err
			}
			rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
			for _, s := range all {
				if s.PaidOnly {
					continue
				}
				if solveDifficulty != "" && !strings.EqualFold(s.Difficulty, solveDifficulty) {
					continue
				}
				slugs = append(slugs, s.Slug)
				if len(slugs) >= solveCount {
					break
				}
			}
			if len(slugs) == 0 {
				return fmt.Errorf("no problems found for requested filters")
			}
		} else {
			s, err := cli.PickRandom(ctx, solveDifficulty)
			if err != nil {
				return err
			}
			slugs = append(slugs, s.Slug)
		}

		prepared := 0
		for _, slug := range slugs {
			q, err := cli.Question(ctx, slug)
			if err != nil {
				continue
			}
			if solveTopic != "" {
				matched := false
				for _, t := range q.Topics {
					if strings.EqualFold(t, solveTopic) {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
			if solveDifficulty != "" && !strings.EqualFold(q.Difficulty, solveDifficulty) {
				continue
			}

			p := store.Problem{
				FrontendID:    q.FrontendID,
				QuestionID:    q.QuestionID,
				Slug:          q.Slug,
				Title:         q.Title,
				Difficulty:    q.Difficulty,
				Topics:        q.Topics,
				StatementHTML: q.StatementHTML,
				ExampleTests:  q.ExampleTests,
				CodeStub:      q.PythonStub,
				Status:        "in_progress",
			}
			if err := a.store.UpsertProblem(ctx, p); err != nil {
				return err
			}
			row, err := a.store.GetProblem(ctx, q.Slug)
			if err != nil {
				return err
			}
			if row.Status == "todo" {
				_ = a.store.SetProblemStatus(ctx, q.Slug, "in_progress")
				row.Status = "in_progress"
			}
			if err := workspace.EnsureProblemFiles(a.cfg.Workspace.ProblemsDir, row); err != nil {
				return err
			}
			if err := workspace.WriteMetaJSON(a.cfg.Workspace.ProblemsDir, row); err != nil {
				return err
			}
			prepared++
			if prepared == 1 {
				_ = a.store.SetCurrentProblem(ctx, q.Slug)
				if !solveNoTimer {
					_ = a.store.StartTimer(ctx, q.Slug, solveTimer, false)
				}
			}
		}

		if prepared == 0 {
			return fmt.Errorf("no problems prepared; filters may be too restrictive")
		}

		current, _ := a.store.CurrentProblem(ctx)
		fmt.Printf("Prepared %d problem(s). Current: %s\n", prepared, current)
		fmt.Printf("Open: leetcli open %s\n", current)
		if !solveNoTimer {
			fmt.Printf("Timer started: %d minutes\n", solveTimer)
		}
		return nil
	},
}

func init() {
	solveCmd.Flags().StringVar(&solveSlug, "slug", "", "specific problem slug")
	solveCmd.Flags().BoolVar(&solveRandom, "random", false, "pick a random problem")
	solveCmd.Flags().StringVar(&solveDifficulty, "difficulty", "", "filter by difficulty (Easy/Medium/Hard)")
	solveCmd.Flags().StringVar(&solveTopic, "topic", "", "best-effort topic filter")
	solveCmd.Flags().IntVar(&solveCount, "count", 1, "number of problems to cache/prepare")
	solveCmd.Flags().IntVar(&solveTimer, "timer", 30, "default solve timer in minutes")
	solveCmd.Flags().BoolVar(&solveNoTimer, "no-timer", false, "do not auto-start timer")
}
