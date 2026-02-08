package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var noteTags string

var noteCmd = &cobra.Command{
	Use:   "note [slug] <text>",
	Short: "Add a note with optional tags",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		a, err := loadApp(ctx)
		if err != nil {
			return err
		}
		defer a.close()

		var slug, text string
		if len(args) == 1 {
			slug, err = problemSlugFromArgOrCurrent(ctx, a, "")
			if err != nil {
				return err
			}
			text = args[0]
		} else {
			slug = args[0]
			text = args[1]
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("note text is required")
		}
		tags := parseTags(noteTags)
		if err := a.store.AddNote(ctx, slug, text, tags); err != nil {
			return err
		}

		notesPath := filepath.Join(a.cfg.Workspace.ProblemsDir, slug, "notes.md")
		line := fmt.Sprintf("- [%s] %s", time.Now().Format("2006-01-02 15:04"), text)
		if len(tags) > 0 {
			line += fmt.Sprintf(" (tags: %s)", strings.Join(tags, ","))
		}
		line += "\n"
		f, err := os.OpenFile(notesPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = f.WriteString(line)
			_ = f.Close()
		}
		_ = syncMeta(ctx, a, slug)
		fmt.Printf("Saved note for %s\n", slug)
		return nil
	},
}

func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func init() {
	noteCmd.Flags().StringVar(&noteTags, "tags", "", "comma-separated tags (edge-case,complexity,DS choice,bug,off-by-one)")
}
