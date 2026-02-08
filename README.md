# LeetCLI

Terminal-first LeetCode workflow focused on local productivity.

## v1 Features

- Cookie auth for `leetcode.com` via `LEETCODE_SESSION` + `csrftoken`
- Local problem cache with workspace files under `problems/<slug>/`
- SQLite-backed metadata, timers, notes, activity, test runs
- Offline cache-first once a problem is fetched
- Python-only solution flow for v1 (`solution.py` + Python3 submit)
- Full-screen keyboard-driven `browse` TUI

## Project Layout

- `problems/<slug>/README.md`
- `problems/<slug>/solution.py`
- `problems/<slug>/notes.md`
- `problems/<slug>/meta.json`
- `.leetcli/leetcli.db`

## Commands

- `leetcli init [--project]`
- `leetcli auth --session <...> --csrf <...> [--project]`
- `leetcli solve [--slug two-sum | --random] [--difficulty Easy] [--topic Array] [--count 50] [--timer 30]`
- `leetcli browse`
- `leetcli open [slug] [--dir]`
- `leetcli test [slug]`
- `leetcli submit [slug]`
- `leetcli note <slug> "text" [--tags edge-case,bug]`
- `leetcli timer start|stop|extend [slug]`
- `leetcli fetch` (neofetch-style dashboard)
- `leetcli stats [--json]`

## Notes

- Config default: XDG (`$XDG_CONFIG_HOME/leetcli/config.yaml` or `~/.config/leetcli/config.yaml`)
- Project-local override: `.leetcli/config.yaml`
- Env vars override config values.
