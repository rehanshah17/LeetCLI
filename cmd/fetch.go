package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Neofetch-inspired LeetCLI overview",
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

		green := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
		cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
		subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		plus := "+"
		improvement := st.SolvedLast7Days - st.SolvedPrev7Days
		impText := fmt.Sprintf("%s%d vs previous 7d", plus, improvement)
		if improvement < 0 {
			impText = fmt.Sprintf("%d vs previous 7d", improvement)
		}

		ascii := cyan.Render(strings.Join([]string{
			"    __              __  ________    __",
			"   / /   ___  ___  / /_/ ____/ /   / /",
			"  / /   / _ \\/ _ \\/ __/ /   / /   / / ",
			" / /___/  __/  __/ /_/ /___/ /___/ /  ",
			"/_____/\\___/\\___/\\__/\\____/_____/_/   ",
		}, "\n"))

		fmt.Println(ascii)
		fmt.Println(subtle.Render("terminal-first leetcode practice"))
		fmt.Printf("\n%s solved: Easy=%d Medium=%d Hard=%d\n", green.Render("+"), st.SolvedEasy, st.SolvedMedium, st.SolvedHard)
		fmt.Printf("%s total cached problems: %d\n", green.Render("+"), st.TotalProblems)
		fmt.Printf("%s topic coverage: %d\n", green.Render("+"), st.TopicCoverage)
		fmt.Printf("%s avg solve time: %.1f min\n", green.Render("+"), st.AvgSolveSec/60.0)
		fmt.Printf("%s momentum: %s\n", green.Render("+"), green.Render(impText))
		fmt.Println("\nRecent activity:")
		if len(st.RecentActivity) == 0 {
			fmt.Println("  (none yet)")
			return nil
		}
		for _, a := range st.RecentActivity {
			fmt.Printf("  %s %s [%s]\n", subtle.Render(a.CreatedAt), a.Kind, a.Slug)
		}
		return nil
	},
}
