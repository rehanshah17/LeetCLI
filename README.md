# LeetCLI

```text
    __              __  ________    __
   / /   ___  ___  / /_/ ____/ /   / /
  / /   / _ \/ _ \/ __/ /   / /   / /
 / /___/  __/  __/ /_/ /___/ /___/ /
/_____/\___/\___/\__/\____/_____/_/
```

Terminal-first LeetCode workflow focused on local productivity.

## Build

- `make build` -> `bin/leet`

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

- `leet init [--project]`
- `leet auth --cookie "<COOKIE_HEADER>" [--project]`
- `leet auth --session <LEETCODE_SESSION> --csrf <CSRFTOKEN> [--project]`
- `leet auth guide`
- `leet solve [--slug two-sum | --random] [--difficulty Easy] [--topic Array] [--count 50] [--timer 30] [--no-timer]`
- `leet browse`
- `leet open [slug] [--dir]`
- `leet test [slug]`
- `leet submit [slug]`
- `leet note [slug] "<text>" [--tags edge-case,bug]`
- `leet timer start [slug] [--minutes 30]`
- `leet timer stop [slug]`
- `leet timer extend [slug] [--minutes 10]`
- `leet fetch` (neofetch-style dashboard)
- `leet stats [--json]`

## Notes

- Config default: XDG (`$XDG_CONFIG_HOME/leetcli/config.yaml` or `~/.config/leetcli/config.yaml`)
- Project-local override: `.leetcli/config.yaml`
- Env vars override config values.
- `leet fetch` uses a blue/maize terminal theme.
