---
name: dask
description: Async via dask, end turn immediately; use only when user explicitly delegates to Droid (ask/@droid/let droid/review); NOT for questions about Droid itself.
metadata:
  short-description: Ask Droid asynchronously via dask
  managed-by: ccb-installer
  template-variant: bash
---

# Ask Droid (Async)

Send the user's request to Droid asynchronously.

## Execution (MANDATORY)

For simple one-line messages:
```
Bash(dask "your message here", run_in_background=true)
```

For multi-line or complex messages:
```
Bash(dask "$ARGUMENTS", run_in_background=true)
```

## CRITICAL Rules

- Always use `run_in_background=true`.
- After running `dask`, say "Droid processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.

## Notes

- CCB uses a unified `askd` daemon for all providers (codex, gemini, opencode, claude, droid)
- If it fails, check backend health with `dping`
- The daemon state is stored in `$CCB_RUN_DIR/askd.json`
- For a more complete pattern (including multiline): `../docs/async-ask-pattern.md`

## Troubleshooting

If `dask` fails with "daemon required but not available":
1. Check if askd is running: `dping`
2. The daemon should be auto-started by CCB session
3. State file location: `$CCB_RUN_DIR/askd.json`
