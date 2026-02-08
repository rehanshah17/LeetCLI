package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/leetcode"
	"leetcli/internal/store"
	"leetcli/internal/workspace"
)

type app struct {
	cfg   config.Config
	paths config.Paths
	store *store.Store
}

func loadApp(ctx context.Context) (*app, error) {
	loaded, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := workspace.EnsureBaseDirs(loaded.Config.Workspace.ProblemsDir); err != nil {
		return nil, err
	}
	st, err := store.Open(loaded.Config.Workspace.DBPath)
	if err != nil {
		return nil, err
	}
	return &app{cfg: loaded.Config, paths: loaded.Paths, store: st}, nil
}

func (a *app) close() { _ = a.store.Close() }

func (a *app) client() *leetcode.Client {
	return leetcode.New(a.cfg.Site, a.cfg.Auth.LeetCodeSession, a.cfg.Auth.CSRFToken)
}

func problemSlugFromArgOrCurrent(ctx context.Context, a *app, slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug != "" {
		return slug, nil
	}
	cur, err := a.store.CurrentProblem(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", err
		}
		return "", fmt.Errorf("no current problem set; pass a slug")
	}
	if cur == "" {
		return "", fmt.Errorf("no current problem set; pass a slug")
	}
	return cur, nil
}

func syncMeta(ctx context.Context, a *app, slug string) error {
	p, err := a.store.GetProblem(ctx, slug)
	if err != nil {
		return err
	}
	if err := workspace.EnsureProblemFiles(a.cfg.Workspace.ProblemsDir, p); err != nil {
		return err
	}
	return workspace.WriteMetaJSON(a.cfg.Workspace.ProblemsDir, p)
}

func solutionPath(problemsDir, slug string) string {
	return filepath.Join(problemsDir, slug, "solution.py")
}
