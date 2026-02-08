package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show progress and activity statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		a, err := loadApp(ctx)
		if err != nil {
			return err
		}
		defer a.close()
		st, err := a.store.Stats(ctx)
		if err != nil {
			return err
		}
		if statsJSON {
			b, _ := json.MarshalIndent(st, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		fmt.Printf("Solved: Easy=%d Medium=%d Hard=%d\n", st.SolvedEasy, st.SolvedMedium, st.SolvedHard)
		fmt.Printf("Cached: %d\n", st.TotalProblems)
		fmt.Printf("Topic coverage: %d\n", st.TopicCoverage)
		fmt.Printf("Avg solve time: %.1f min\n", st.AvgSolveSec/60.0)
		fmt.Printf("7-day solved: %d (prev7=%d)\n", st.SolvedLast7Days, st.SolvedPrev7Days)
		return nil
	},
}

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output machine-readable JSON")
}
