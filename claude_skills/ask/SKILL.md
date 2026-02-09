---
name: ask
description: Async via ask, end turn immediately; use when user explicitly delegates to any AI provider (gemini/codex/opencode/droid); NOT for questions about the providers themselves.
metadata:
  short-description: Ask AI provider asynchronously
---

# Ask AI Provider (Async)

Send the user's request to specified AI provider asynchronously.

## Usage

The first argument must be the provider name, followed by the message:
- `gemini` - Send to Gemini
- `codex` - Send to Codex
- `opencode` - Send to OpenCode
- `droid` - Send to Droid

## Execution (MANDATORY)

**Windows Native (PowerShell/WezTerm) - USE THIS:**
```
Bash(ask $PROVIDER "$MESSAGE")
```

**Linux/macOS/WSL only:**
```
Bash(nohup sh -c 'CCB_CALLER=claude ask $PROVIDER <<EOF
$MESSAGE
EOF
' > /dev/null 2>&1 &)
```

IMPORTANT: On Windows, just use `ask` directly. Do NOT use nohup/sh - they don't exist on native Windows!

## Rules

- After running the command, say "[Provider] processing..." and immediately end your turn.
- Do not wait for results or check status in the same turn.
- The task ID and log file path will be displayed for tracking.

## Examples

- `/ask gemini What is 12+12?`
- `/ask codex Refactor this code`
- `/ask opencode Analyze this bug`
- `/ask droid Execute this task`

## Notes

- If it fails, check backend health with the corresponding ping command (`ping <provider>` (e.g., `ping gemini`)).
