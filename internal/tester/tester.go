package tester

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type UserTestCase struct {
	Input    any `json:"input"`
	Expected any `json:"expected"`
}

type Result struct {
	Passed      bool
	FailedCount int
	Output      string
}

func RunPython(solutionPath, exampleTests string, userCases []UserTestCase) (Result, error) {
	method := detectMethod(solutionPath)
	payload := map[string]any{
		"method":  method,
		"example": splitExampleCases(exampleTests),
		"user":    userCases,
	}
	pb, _ := json.Marshal(payload)

	tmpRunner := filepath.Join(filepath.Dir(solutionPath), ".leetcli_runner.py")
	if err := os.WriteFile(tmpRunner, []byte(runnerScript), 0o644); err != nil {
		return Result{}, err
	}
	defer os.Remove(tmpRunner)

	cmd := exec.Command("python3", tmpRunner, solutionPath, string(pb))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String() + stderr.String()
	if err != nil {
		return Result{Passed: false, FailedCount: 1, Output: out}, nil
	}

	var parsed struct {
		Passed bool `json:"passed"`
		Failed int  `json:"failed"`
	}
	if jErr := json.Unmarshal(stdout.Bytes(), &parsed); jErr != nil {
		return Result{Passed: false, FailedCount: 1, Output: out}, nil
	}
	return Result{Passed: parsed.Passed, FailedCount: parsed.Failed, Output: out}, nil
}

func LoadUserCases(problemDir string) ([]UserTestCase, error) {
	path := filepath.Join(problemDir, "tests.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tc []UserTestCase
	if err := json.Unmarshal(b, &tc); err != nil {
		return nil, fmt.Errorf("parse tests.json: %w", err)
	}
	return tc, nil
}

func detectMethod(solutionPath string) string {
	b, err := os.ReadFile(solutionPath)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	m := re.FindSubmatch(b)
	if len(m) < 2 {
		return ""
	}
	if string(m[1]) == "__init__" {
		all := re.FindAllSubmatch(b, -1)
		for _, mm := range all {
			name := string(mm[1])
			if name != "__init__" {
				return name
			}
		}
	}
	return string(m[1])
}

func splitExampleCases(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{s}
	}
	return out
}

const runnerScript = `
import importlib.util
import json
import sys
import traceback


def load_module(path):
    spec = importlib.util.spec_from_file_location("solution_module", path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def parse_args(raw):
    lines = [l.strip() for l in raw.splitlines() if l.strip()]
    vals = []
    for line in lines:
        if line.startswith("Input:"):
            line = line[len("Input:"):].strip()
        if "=" in line and not line.startswith("[") and not line.startswith("{") and not line.startswith("("):
            chunks = [x.strip() for x in line.split(",") if x.strip()]
            for c in chunks:
                if "=" in c:
                    vals.append(eval(c.split("=", 1)[1].strip()))
                else:
                    vals.append(eval(c))
        else:
            vals.append(eval(line))
    return vals


def main():
    solution_path = sys.argv[1]
    payload = json.loads(sys.argv[2])
    method_name = payload.get("method")

    mod = load_module(solution_path)
    sol = mod.Solution()
    if not method_name:
      methods = [m for m in dir(sol) if not m.startswith("_") and callable(getattr(sol, m))]
      method_name = methods[0] if methods else None
    if not method_name:
      print(json.dumps({"passed": False, "failed": 1}))
      return

    fn = getattr(sol, method_name)
    failed = 0

    for raw in payload.get("example", []):
      try:
        args = parse_args(raw)
        if len(args) == 1 and isinstance(args[0], tuple):
          args = list(args[0])
        fn(*args)
      except Exception:
        failed += 1
        traceback.print_exc()

    for case in payload.get("user", []):
      try:
        args = case.get("input")
        if not isinstance(args, list):
          args = [args]
        got = fn(*args)
        if "expected" in case and case.get("expected") != got:
          failed += 1
          print(f"expected={case.get('expected')} got={got}")
      except Exception:
        failed += 1
        traceback.print_exc()

    print(json.dumps({"passed": failed == 0, "failed": failed}))


if __name__ == "__main__":
    main()
`
