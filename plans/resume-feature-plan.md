# CCB Resume Feature (`ccb.exe -r`) Implementation Plan

## Summary

Implement `ccb.exe -r codex,claude` to resume previous AI provider sessions instead of starting fresh. When WezTerm closes, session files are already persisted in `.ccb_config/` — this feature adds the ability to use them on next startup.

## Key Insight

**All 5 providers already have native resume support:**

| Provider | Resume Command | CWD-aware? |
|----------|---------------|------------|
| Claude   | `claude --continue` | Yes (auto per-project) |
| Codex    | `codex resume --last` | Yes (`--last` filters by CWD) |
| Gemini   | `gemini --resume latest` | Needs `cd` to resume dir |
| OpenCode | `opencode --continue` | Yes |
| Droid    | `droid -r` | Needs `cd` to resume dir |

CCB doesn't need to manage conversation history — providers handle that. CCB only needs to pass the right resume flag.

## Architecture Decision

**Approach: Minimal Flag Injection (Proposal B + enhancements)**

- No new structs, no new packages, no shutdown hooks
- Providers auto-save their own conversation state on exit
- Session files already persist across WezTerm restarts
- ~80 lines of changes across 2 files

## Implementation Details

### 1. CLI Flag Parsing — `cmd/ccb/main.go`

Add `-r` / `--resume` flag in `runLauncher()`:

```go
func runLauncher(args []string) {
    auto := false
    resume := false  // NEW

    for _, arg := range args {
        switch arg {
        case "-a", "--auto":
            auto = true
        case "-r", "--resume":  // NEW
            resume = true
        default:
            // ... existing logic
        }
    }

    results, err := launcher.Launch(launcher.LaunchConfig{
        Providers: providers,
        Auto:      auto,
        Resume:    resume,  // Already exists in struct, now actually used
        WorkDir:   cwd,
    })
    // ...
}
```

Also update help text in `buildRootCmd()` to document `-r`.

### 2. Resume Command Building — `internal/launcher/launcher.go`

#### 2a. Modify `BuildStartCommand` signature

```go
func BuildStartCommand(provider string, auto bool, resume bool) (string, error)
```

#### 2b. Add resume flag injection

Inside `BuildStartCommand`, after building base command parts:

```go
// Resume flags (injected before auto-approve flags)
if resume {
    switch provider {
    case "claude":
        parts = append(parts, "--continue")
    case "codex":
        // Insert "resume --last" as subcommand
        // Codex resume syntax: codex [base-flags] resume --last [auto-flags]
        parts = append(parts, "resume", "--last")
    case "gemini":
        parts = append(parts, "--resume", "latest")
    case "opencode":
        parts = append(parts, "--continue")
    case "droid":
        parts = append(parts, "-r")
    }
}
```

**Special consideration for Codex**: The `resume` subcommand must come before auto-approve flags. Current flow:

```
Fresh:  codex -c disable_paste_burst=true -c trust_level="trusted" ...
Resume: codex -c disable_paste_burst=true resume --last -c trust_level="trusted" ...
```

Actually, `codex resume` accepts its own `-c` flags, so we need to restructure:

```go
case "codex":
    if resume {
        // Codex resume command: codex resume --last [flags]
        parts = []string{exe, "resume", "--last"}
        parts = append(parts, "-c", "disable_paste_burst=true")
    } else {
        parts = append(parts, "-c", "disable_paste_burst=true")
    }
```

#### 2c. Update call sites

In `launchWithBackend()`:
```go
cmd, err := BuildStartCommand(provider, cfg.Auto, cfg.Resume)
```

In `launchFallback()`:
```go
cmd, err := BuildStartCommand(provider, cfg.Auto, cfg.Resume)
```

#### 2d. Add user feedback

```go
if resume {
    fmt.Printf("Resuming %s session...\n", provider)
}
```

### 3. Session File Persistence (Already Works!)

Current session files in `.ccb_config/.<provider>-session` already:
- Persist on disk after WezTerm closes
- Store `work_dir`, `pane_id`, `active` status
- Are NOT deleted when WezTerm exits

**No changes needed.** The session files are just metadata for CCB's internal use. The actual conversation history lives in provider directories (`~/.claude/`, `~/.codex/sessions/`, etc.) and is managed by the providers themselves.

### 4. Edge Cases

| Scenario | Behavior |
|----------|----------|
| No previous session exists | Provider starts fresh (e.g., `codex resume --last` shows empty picker or starts fresh) |
| Provider not installed | Same error as without `-r` |
| `-r` combined with `-a` | Both flags apply: `codex resume --last -a never` |
| Corrupted provider history | Provider handles its own error |
| Different work directory | `codex resume --last` auto-filters by CWD; Claude `--continue` auto-filters by project |

### 5. Combining `-r` and `-a`

The `-a` (auto-approve) and `-r` (resume) flags are orthogonal. Example:

```
ccb -a -r codex,claude
→ codex resume --last -c trust_level="trusted" -c approval_policy="never" -c sandbox_mode="danger-full-access"
→ claude --continue  (auto-approve handled via config file)
```

## Files to Modify

### `cmd/ccb/main.go` (~15 lines)
- Add `-r`/`--resume` flag parsing in `runLauncher()`
- Pass `Resume` to `LaunchConfig`
- Update CLI help text

### `internal/launcher/launcher.go` (~65 lines)
- Change `BuildStartCommand(provider, auto)` → `BuildStartCommand(provider, auto, resume)`
- Add resume flag injection logic per provider
- Handle Codex `resume --last` subcommand structure
- Update two call sites in `launchWithBackend()` and `launchFallback()`
- Add user feedback messages

## What This Does NOT Do (By Design)

1. ❌ No session lifecycle tracking — providers handle this
2. ❌ No conversation history management — providers handle this
3. ❌ No shutdown hooks — providers auto-save on exit
4. ❌ No new session metadata — existing files are sufficient
5. ❌ No session cleanup — existing 7-day TTL in pane registry handles this

## Testing

```powershell
# Test 1: Fresh start (baseline, should work as before)
.\ccb.exe -a codex,claude

# Test 2: Resume after closing WezTerm
.\ccb.exe -r codex,claude

# Test 3: Resume + auto-approve
.\ccb.exe -a -r codex,claude

# Test 4: Resume with no prior session (should start fresh)
cd C:\new-empty-project
.\ccb.exe -r codex

# Test 5: Only resume flag
.\ccb.exe -r codex
```

## Why This Approach

The user asked: "When I exit WezTerm, can the session be saved so next time it auto-loads?"

The answer is: **sessions are already saved** — each provider automatically persists conversation history to its own directory. The only thing missing is telling the provider CLI to "continue where you left off" instead of "start a new conversation." That's exactly what `-r` does.
