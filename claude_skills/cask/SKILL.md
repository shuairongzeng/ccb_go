---
name: cask
description: Async via cask, end turn immediately; use only when user explicitly delegates to Codex (ask/@codex/let codex/review); NOT for questions about Codex itself.
metadata:
  short-description: Ask Codex asynchronously via cask
  managed-by: ccb-installer
  template-variant: bash
---

# Ask Codex (Async)

Send the user's request to Codex asynchronously.

## Execution (MANDATORY)

For simple one-line messages:
```
Bash(cask "your message here", run_in_background=true)
```

For multi-line or complex messages:
```
Bash(cask "$ARGUMENTS", run_in_background=true)
```

## CRITICAL Rules

- Always use `run_in_background=true`.
- After running `cask`, say "Codex processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.

## Notes

- CCB uses a unified `askd` daemon for all providers (codex, gemini, opencode, etc.)
- If it fails, check backend health with `cping`
- The daemon state is stored in `$CCB_RUN_DIR/askd.json`
- For a more complete pattern (including multiline): `../docs/async-ask-pattern.md`

## Troubleshooting

If `cask` fails with "daemon required but not available":
1. Check if askd is running: `cping`
2. The daemon should be auto-started by CCB session
3. State file location: `$CCB_RUN_DIR/askd.json`
