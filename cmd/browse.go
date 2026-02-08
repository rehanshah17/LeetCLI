package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Interactive problem browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		a, err := loadApp(ctx)
		if err != nil {
			return err
		}
		defer a.close()

		m, err := newBrowseModel(ctx, a)
		if err != nil {
			return err
		}
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

type browseModel struct {
	ctx        context.Context
	a          *app
	items      []browseItem
	cursor     int
	query      textinput.Model
	difficulty string
	status     string
	msg        string
}

type browseItem struct {
	slug       string
	title      string
	difficulty string
	status     string
}

type submitDoneMsg struct {
	text string
	err  error
}

func newBrowseModel(ctx context.Context, a *app) (browseModel, error) {
	q := textinput.New()
	q.Placeholder = "search slug/title"
	q.Focus()
	q.CharLimit = 120
	q.Width = 40

	m := browseModel{ctx: ctx, a: a, query: q}
	if err := m.reload(); err != nil {
		return m, err
	}
	return m, nil
}

func (m *browseModel) reload() error {
	rows, err := m.a.store.ListProblems(m.ctx, m.difficulty, m.status, m.query.Value())
	if err != nil {
		return err
	}
	m.items = m.items[:0]
	for _, r := range rows {
		m.items = append(m.items, browseItem{slug: r.Slug, title: r.Title, difficulty: r.Difficulty, status: r.Status})
	}
	if m.cursor >= len(m.items) {
		m.cursor = len(m.items) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return nil
}

func (m browseModel) Init() tea.Cmd { return nil }

func (m browseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case tea.KeyMsg:
		switch t.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "ctrl+u":
			m.query.SetValue("")
			_ = m.reload()
		case "tab":
			m.difficulty = nextDifficulty(m.difficulty)
			_ = m.reload()
		case "shift+tab":
			m.status = nextStatus(m.status)
			_ = m.reload()
		case "s":
			if it, ok := m.selected(); ok {
				next := cycleProblemStatus(it.status)
				_ = m.a.store.SetProblemStatus(m.ctx, it.slug, next)
				_ = syncMeta(m.ctx, m.a, it.slug)
				m.msg = fmt.Sprintf("status: %s -> %s", it.slug, next)
				_ = m.reload()
			}
		case "t":
			if it, ok := m.selected(); ok {
				_ = m.a.store.StartTimer(m.ctx, it.slug, 30, true)
				m.msg = "timer started: " + it.slug
			}
		case "T":
			if it, ok := m.selected(); ok {
				d, _ := m.a.store.StopTimer(m.ctx, it.slug)
				_ = syncMeta(m.ctx, m.a, it.slug)
				m.msg = fmt.Sprintf("timer stopped: %s (+%ds)", it.slug, d)
			}
		case "n":
			if it, ok := m.selected(); ok {
				path := filepath.Join(m.a.cfg.Workspace.ProblemsDir, it.slug, "notes.md")
				return m, tea.ExecProcess(editorCmd(path), nil)
			}
		case "enter", "o":
			if it, ok := m.selected(); ok {
				path := solutionPath(m.a.cfg.Workspace.ProblemsDir, it.slug)
				_ = m.a.store.SetCurrentProblem(m.ctx, it.slug)
				return m, tea.ExecProcess(editorCmd(path), nil)
			}
		case "u":
			if it, ok := m.selected(); ok {
				m.msg = "submitting " + it.slug + "..."
				return m, m.submitCmd(it.slug)
			}
		default:
			var cmd tea.Cmd
			m.query, cmd = m.query.Update(t)
			if keyLikelyChangesQuery(t) {
				_ = m.reload()
			}
			return m, cmd
		}
	case submitDoneMsg:
		if t.err != nil {
			m.msg = "submit error: " + t.err.Error()
		} else {
			m.msg = t.text
		}
		_ = m.reload()
	}
	return m, nil
}

func (m browseModel) View() string {
	header := lipgloss.NewStyle().Bold(true).Render("LeetCLI Browse")
	sub := fmt.Sprintf("search=%q  difficulty=%s(tab)  status=%s(shift+tab)", m.query.Value(), blankAsAll(m.difficulty), blankAsAll(m.status))
	legend := "enter/o open  s mark status  t/T timer start/stop  n note  u submit  q quit"
	var b strings.Builder
	b.WriteString(header + "\n")
	b.WriteString(sub + "\n")
	b.WriteString(legend + "\n\n")
	if len(m.items) == 0 {
		b.WriteString("No cached problems. Run `leet solve --random` first.\n")
	} else {
		for i, it := range m.items {
			cursor := " "
			if i == m.cursor {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %-6s %-7s %-11s %s\n", cursor, it.slug, it.difficulty, it.status, it.title))
		}
	}
	if m.msg != "" {
		b.WriteString("\n" + m.msg + "\n")
	}
	return b.String()
}

func (m browseModel) selected() (browseItem, bool) {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return browseItem{}, false
	}
	return m.items[m.cursor], true
}

func (m browseModel) submitCmd(slug string) tea.Cmd {
	return func() tea.Msg {
		p, err := m.a.store.GetProblem(m.ctx, slug)
		if err != nil {
			return submitDoneMsg{err: err}
		}
		if p.QuestionID == "" {
			q, qErr := m.a.client().Question(m.ctx, slug)
			if qErr != nil {
				return submitDoneMsg{err: qErr}
			}
			p.QuestionID = q.QuestionID
		}
		code, err := os.ReadFile(solutionPath(m.a.cfg.Workspace.ProblemsDir, slug))
		if err != nil {
			return submitDoneMsg{err: err}
		}
		res, err := m.a.client().Submit(m.ctx, slug, p.QuestionID, string(code))
		if err != nil {
			return submitDoneMsg{err: err}
		}
		_ = m.a.store.SaveSubmissionResult(m.ctx, slug, res.Status, res.Runtime, res.Memory)
		_ = syncMeta(m.ctx, m.a, slug)
		return submitDoneMsg{text: fmt.Sprintf("submission %d: %s", res.SubmissionID, res.Status)}
	}
}

func editorCmd(path string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	return exec.Command(editor, path)
}

func blankAsAll(v string) string {
	if v == "" {
		return "all"
	}
	return v
}

func nextDifficulty(v string) string {
	order := []string{"", "Easy", "Medium", "Hard"}
	for i := range order {
		if order[i] == v {
			return order[(i+1)%len(order)]
		}
	}
	return ""
}

func nextStatus(v string) string {
	order := []string{"", "todo", "in_progress", "solved"}
	for i := range order {
		if order[i] == v {
			return order[(i+1)%len(order)]
		}
	}
	return ""
}

func cycleProblemStatus(v string) string {
	order := []string{"todo", "in_progress", "solved"}
	for i := range order {
		if order[i] == v {
			return order[(i+1)%len(order)]
		}
	}
	return "todo"
}

func keyLikelyChangesQuery(k tea.KeyMsg) bool {
	s := k.String()
	if len(s) == 1 {
		return true
	}
	return s == "backspace" || s == "delete" || s == "space"
}
