package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type AuthConfig struct {
	LeetCodeSession string `mapstructure:"leetcode_session"`
	CSRFToken       string `mapstructure:"csrftoken"`
}

type WorkspaceConfig struct {
	ProblemsDir string `mapstructure:"problems_dir"`
	DBPath      string `mapstructure:"db_path"`
}

type Config struct {
	Site      string          `mapstructure:"site"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Workspace WorkspaceConfig `mapstructure:"workspace"`
}

type Paths struct {
	XDGConfigFile   string
	LocalConfigFile string
}

type Loaded struct {
	Config Config
	Paths  Paths
}

func defaultConfig() Config {
	return Config{
		Site: "https://leetcode.com",
		Auth: AuthConfig{},
		Workspace: WorkspaceConfig{
			ProblemsDir: "problems",
			DBPath:      filepath.Join(".leetcli", "leetcli.db"),
		},
	}
}

func ResolvePaths() (Paths, error) {
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve home dir: %w", err)
		}
		xdgHome = filepath.Join(home, ".config")
	}

	return Paths{
		XDGConfigFile:   filepath.Join(xdgHome, "leetcli", "config.yaml"),
		LocalConfigFile: filepath.Join(".leetcli", "config.yaml"),
	}, nil
}

func Load() (Loaded, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return Loaded{}, err
	}

	v := viper.New()
	cfg := defaultConfig()
	v.SetDefault("site", cfg.Site)
	v.SetDefault("workspace.problems_dir", cfg.Workspace.ProblemsDir)
	v.SetDefault("workspace.db_path", cfg.Workspace.DBPath)

	if _, err := os.Stat(paths.XDGConfigFile); err == nil {
		v.SetConfigFile(paths.XDGConfigFile)
		if err := v.ReadInConfig(); err != nil {
			return Loaded{}, fmt.Errorf("read XDG config: %w", err)
		}
	}

	if _, err := os.Stat(paths.LocalConfigFile); err == nil {
		v.SetConfigFile(paths.LocalConfigFile)
		if err := v.MergeInConfig(); err != nil {
			return Loaded{}, fmt.Errorf("merge local config: %w", err)
		}
	}

	_ = v.BindEnv("auth.leetcode_session", "LEETCODE_SESSION")
	_ = v.BindEnv("auth.csrftoken", "CSRFTOKEN")
	_ = v.BindEnv("auth.csrftoken", "LEETCODE_CSRFTOKEN")
	_ = v.BindEnv("site", "LEETCODE_SITE")
	v.AutomaticEnv()

	var out Config
	if err := v.Unmarshal(&out); err != nil {
		return Loaded{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return Loaded{Config: out, Paths: paths}, nil
}

func Save(cfg Config, toLocal bool) (string, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return "", err
	}

	path := paths.XDGConfigFile
	if toLocal {
		path = paths.LocalConfigFile
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	content := fmt.Sprintf(`site: %q
auth:
  leetcode_session: %q
  csrftoken: %q
workspace:
  problems_dir: %q
  db_path: %q
`, cfg.Site, cfg.Auth.LeetCodeSession, cfg.Auth.CSRFToken, cfg.Workspace.ProblemsDir, cfg.Workspace.DBPath)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	return path, nil
}
