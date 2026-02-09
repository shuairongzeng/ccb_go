# CCB Go - Claude Code Bridge

Multi-model AI collaboration tool, rewritten in Go. Launch multiple AI providers (Claude, Codex, Gemini, OpenCode, Droid) in terminal panes and let them collaborate.

## Build

```bash
go build -o ccb.exe ./cmd/ccb/
```

## Usage

```powershell
# Start fresh sessions
ccb codex,claude

# Auto-approve mode
ccb -a codex,claude

# Resume previous sessions
ccb -r codex,claude

# Resume + auto-approve
ccb -a -r codex,claude
```

## Flags

| Flag | Description |
|------|-------------|
| `-a`, `--auto` | Auto-approve mode (skip permission prompts) |
| `-r`, `--resume` | Resume previous sessions instead of starting fresh |

## Providers

| Provider | CLI | Resume Flag |
|----------|-----|-------------|
| Claude | `claude` | `--continue` |
| Codex | `codex` | `resume --last` |
| Gemini | `gemini` | `--resume latest` |
| OpenCode | `opencode` | `--continue` |
| Droid | `droid` | `-r` |

## Prerequisites

- **WezTerm** (recommended): `winget install wez.wezterm`
- At least one AI provider CLI installed
- Go 1.21+ (for building from source)

## Project Structure

```
cmd/ccb/          - CLI entry point
internal/
  launcher/       - Provider launch and resume logic
  terminal/       - WezTerm, tmux, PowerShell backends
  daemon/         - Built-in askd daemon
  session/        - Session discovery and registry
  client/         - Daemon client (ask/ping/pend)
  comm/           - Provider communication modules
  config/         - Configuration and session files
  protocol/       - Provider specs and protocol
  runtime/        - Runtime utilities
  i18n/           - Internationalization
  output/         - Output formatting
  lock/           - Process locking
claude_skills/    - Claude slash command skills
codex_skills/     - Codex skills
droid_skills/     - Droid skills
bin/              - Windows helper scripts
```

## License

See original project: [claude_code_bridge](https://github.com/anthropics/claude_code_bridge)
