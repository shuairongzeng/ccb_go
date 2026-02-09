---
name: lask
description: Async via lask, end turn immediately; use only when user explicitly delegates to Claude (ask/@claude/let claude/review); NOT for questions about Claude itself.
metadata:
  short-description: Ask Claude asynchronously via lask
  managed-by: ccb-installer
  template-variant: bash
---

# Ask Claude (Async)

Send the user's request to Claude asynchronously.

## Execution (MANDATORY)

For simple one-line messages:
```
Bash(lask "your message here", run_in_background=true)
```

For multi-line or complex messages:
```
Bash(lask "$ARGUMENTS", run_in_background=true)
```

## CRITICAL Rules

- Always use `run_in_background=true`.
- After running `lask`, say "Claude processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.

## Notes

- CCB uses a unified `askd` daemon for all providers (codex, gemini, opencode, claude, droid)
- If it fails, check backend health with `lping`
- The daemon state is stored in `$CCB_RUN_DIR/askd.json`
- For a more complete pattern (including multiline): `../docs/async-ask-pattern.md`

## Troubleshooting

If `lask` fails with "daemon required but not available":
1. Check if askd is running: `lping`
2. The daemon should be auto-started by CCB session
3. State file location: `$CCB_RUN_DIR/askd.json`
