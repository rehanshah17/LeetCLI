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

		maize := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCB05")).Bold(true)
		blue := lipgloss.NewStyle().Foreground(lipgloss.Color("#00274C")).Bold(true)
		subtle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
		plus := "+"
		improvement := st.SolvedLast7Days - st.SolvedPrev7Days
		impText := fmt.Sprintf("%s%d vs previous 7d", plus, improvement)
		if improvement < 0 {
			impText = fmt.Sprintf("%d vs previous 7d", improvement)
		}

		asciiLines := []string{
			"    __              __  ________    __",
			"   / /   ___  ___  / /_/ ____/ /   / /",
			"  / /   / _ \\/ _ \\/ __/ /   / /   / / ",
			" / /___/  __/  __/ /_/ /___/ /___/ /  ",
			"/_____/\\___/\\___/\\__/\\____/_____/_/   ",
		}
		var renderedASCII []string
		for i, line := range asciiLines {
			if i%2 == 0 {
				renderedASCII = append(renderedASCII, blue.Render(line))
				continue
			}
			renderedASCII = append(renderedASCII, maize.Render(line))
		}
		ascii := strings.Join(renderedASCII, "\n")

		fmt.Println(ascii)
		fmt.Println(subtle.Render("terminal-first leetcode practice"))
		fmt.Printf("\n%s solved: Easy=%d Medium=%d Hard=%d\n", maize.Render("+"), st.SolvedEasy, st.SolvedMedium, st.SolvedHard)
		fmt.Printf("%s total cached problems: %d\n", maize.Render("+"), st.TotalProblems)
		fmt.Printf("%s topic coverage: %d\n", maize.Render("+"), st.TopicCoverage)
		fmt.Printf("%s avg solve time: %.1f min\n", maize.Render("+"), st.AvgSolveSec/60.0)
		fmt.Printf("%s momentum: %s\n", maize.Render("+"), maize.Render(impText))
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
