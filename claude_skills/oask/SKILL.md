---
name: oask
description: Async via oask, end turn immediately; use only when user explicitly delegates to OpenCode (ask/@opencode/let opencode/review); NOT for questions about OpenCode itself.
metadata:
  short-description: Ask OpenCode asynchronously via oask
  managed-by: ccb-installer
  template-variant: bash
---

# Ask OpenCode (Async)

Send the user's request to OpenCode asynchronously.

## Execution (MANDATORY)

For simple one-line messages:
```
Bash(oask "your message here", run_in_background=true)
```

For multi-line or complex messages:
```
Bash(oask "$ARGUMENTS", run_in_background=true)
```

## CRITICAL Rules

- Always use `run_in_background=true`.
- After running `oask`, say "OpenCode processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.

## Notes

- CCB uses a unified `askd` daemon for all providers (codex, gemini, opencode, etc.)
- If it fails, check backend health with `oping`
- The daemon state is stored in `$CCB_RUN_DIR/askd.json`
- For a more complete pattern (including multiline): `../docs/async-ask-pattern.md`

## Troubleshooting

If `oask` fails with "daemon required but not available":
1. Check if askd is running: `oping`
2. The daemon should be auto-started by CCB session
3. State file location: `$CCB_RUN_DIR/askd.json`
