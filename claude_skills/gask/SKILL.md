---
name: gask
description: Async via gask, end turn immediately; use only when user explicitly delegates to Gemini (ask/@gemini/let gemini/review); NOT for questions about Gemini itself.
metadata:
  short-description: Ask Gemini asynchronously via gask
  managed-by: ccb-installer
  template-variant: bash
---

# Ask Gemini (Async)

Send the user's request to Gemini asynchronously.

## Execution (MANDATORY)

For simple one-line messages:
```
Bash(gask "your message here", run_in_background=true)
```

For multi-line or complex messages:
```
Bash(gask "$ARGUMENTS", run_in_background=true)
```

## CRITICAL Rules

- Always use `run_in_background=true`.
- After running `gask`, say "Gemini processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.

## Notes

- CCB uses a unified `askd` daemon for all providers (codex, gemini, opencode, etc.)
- If it fails, check backend health with `gping`
- The daemon state is stored in `$CCB_RUN_DIR/askd.json`
- For a more complete pattern (including multiline): `../docs/async-ask-pattern.md`

## Troubleshooting

If `gask` fails with "daemon required but not available":
1. Check if askd is running: `gping`
2. The daemon should be auto-started by CCB session
3. State file location: `$CCB_RUN_DIR/askd.json`
