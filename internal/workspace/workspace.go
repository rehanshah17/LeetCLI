package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"leetcli/internal/store"
)

type ProblemMeta struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Difficulty   string   `json:"difficulty"`
	Topics       []string `json:"topics"`
	Status       string   `json:"status"`
	TimeSpentSec int      `json:"time_spent_sec"`
	LastSubmit   string   `json:"last_submit"`
	Runtime      string   `json:"runtime"`
	Memory       string   `json:"memory"`
	UpdatedAt    string   `json:"updated_at"`
}

func EnsureBaseDirs(problemsDir string) error {
	if err := os.MkdirAll(problemsDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(".leetcli", 0o755); err != nil {
		return err
	}
	return nil
}

func ProblemDir(problemsDir, slug string) string {
	return filepath.Join(problemsDir, slug)
}

func EnsureProblemFiles(problemsDir string, p store.ProblemRow) error {
	dir := ProblemDir(problemsDir, p.Slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create problem dir: %w", err)
	}

	readmePath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(readmePath, []byte(renderReadme(p)), 0o644); err != nil {
		return fmt.Errorf("write README: %w", err)
	}

	solutionPath := filepath.Join(dir, "solution.py")
	if _, err := os.Stat(solutionPath); os.IsNotExist(err) {
		stub := strings.TrimSpace(p.CodeStub)
		if stub == "" {
			stub = "class Solution:\n    pass\n"
		}
		if err := os.WriteFile(solutionPath, []byte(stub+"\n"), 0o644); err != nil {
			return fmt.Errorf("write solution: %w", err)
		}
	}

	notesPath := filepath.Join(dir, "notes.md")
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		seed := "# Notes\n\n- Mistakes:\n- Insights:\n"
		if err := os.WriteFile(notesPath, []byte(seed), 0o644); err != nil {
			return fmt.Errorf("write notes: %w", err)
		}
	}

	return nil
}

func WriteMetaJSON(problemsDir string, p store.ProblemRow) error {
	m := ProblemMeta{
		Slug:         p.Slug,
		Title:        p.Title,
		Difficulty:   p.Difficulty,
		Topics:       p.Topics,
		Status:       p.Status,
		TimeSpentSec: p.TimeSpentSec,
		LastSubmit:   p.LastSubmit,
		Runtime:      p.Runtime,
		Memory:       p.Memory,
		UpdatedAt:    p.UpdatedAt,
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(ProblemDir(problemsDir, p.Slug), "meta.json")
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func AppendDebugLog(problemsDir, slug, out string) error {
	path := filepath.Join(ProblemDir(problemsDir, slug), "debug.log")
	entry := fmt.Sprintf("\n[%s]\n%s\n", time.Now().Format(time.RFC3339), out)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

func OpenInEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func renderReadme(p store.ProblemRow) string {
	topicLine := ""
	if len(p.Topics) > 0 {
		topicLine = strings.Join(p.Topics, ", ")
	}
	return fmt.Sprintf(`# %s

- Slug: %s
- Difficulty: %s
- Topics: %s

## Statement

%s

## Example Testcases (best effort)

`+"```text\n%s\n```\n", p.Title, p.Slug, p.Difficulty, topicLine, p.StatementHTML, p.ExampleTests)
}
