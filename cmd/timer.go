package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var timerCmd = &cobra.Command{
	Use:   "timer",
	Short: "Manage solve timers",
}

var timerStartCmd = &cobra.Command{
	Use:   "start [slug]",
	Short: "Start timer",
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
		if err := a.store.StartTimer(ctx, slug, timerStartMinutes, true); err != nil {
			return err
		}
		fmt.Printf("Started %d-minute timer for %s\n", timerStartMinutes, slug)
		return nil
	},
}

var timerStopCmd = &cobra.Command{
	Use:   "stop [slug]",
	Short: "Stop active timer",
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
		dur, err := a.store.StopTimer(ctx, slug)
		if err != nil {
			return err
		}
		if dur == 0 {
			fmt.Printf("No active timer for %s\n", slug)
			return nil
		}
		_ = syncMeta(ctx, a, slug)
		fmt.Printf("Stopped timer for %s: +%dm%ds\n", slug, dur/60, dur%60)
		return nil
	},
}

var timerExtendCmd = &cobra.Command{
	Use:   "extend [slug]",
	Short: "Manually add minutes to time spent",
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
		if err := a.store.AddManualTime(ctx, slug, timerExtendMinutes); err != nil {
			return err
		}
		_ = syncMeta(ctx, a, slug)
		fmt.Printf("Added %d minutes to %s\n", timerExtendMinutes, slug)
		return nil
	},
}

var timerStartMinutes int
var timerExtendMinutes int

func init() {
	timerStartCmd.Flags().IntVar(&timerStartMinutes, "minutes", 30, "timer duration in minutes")
	timerExtendCmd.Flags().IntVar(&timerExtendMinutes, "minutes", 10, "minutes to add")
	timerCmd.AddCommand(timerStartCmd)
	timerCmd.AddCommand(timerStopCmd)
	timerCmd.AddCommand(timerExtendCmd)
}
