<#
.SYNOPSIS
    CCB Windows Native Mode - Multi-AI collaboration without WezTerm

.DESCRIPTION
    This script launches CCB in Windows native mode using separate PowerShell
    windows instead of WezTerm panes. Use this if WezTerm crashes or is unstable.

.EXAMPLE
    .\ccb-windows.ps1 codex gemini claude
    .\ccb-windows.ps1 codex claude
    .\ccb-windows.ps1 -List
    .\ccb-windows.ps1 -Cleanup

.NOTES
    Each AI runs in its own PowerShell window.
    Use ask/ping/pend commands to communicate between them.
#>

param(
    [Parameter(Position = 0, ValueFromRemainingArguments = $true)]
    [string[]]$Providers,
    [switch]$List,
    [switch]$Cleanup,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

# UTF-8 support
try { $OutputEncoding = [System.Text.UTF8Encoding]::new($false) } catch {}
try { [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false) } catch {}
try { chcp 65001 | Out-Null } catch {}

function Show-Help {
    Write-Host @"

CCB Windows Native Mode
=======================
Multi-AI collaboration using separate PowerShell windows.
Use this if WezTerm is unstable on your system.

Usage:
  ccb-windows <providers...>    Start AI providers in separate windows
  ccb-windows -List             List running AI windows
  ccb-windows -Cleanup          Close all AI windows

Providers: codex, gemini, opencode, droid, claude

Examples:
  ccb-windows codex gemini claude    # Start all three
  ccb-windows codex claude           # Start just Codex + Claude

How it works:
  1. Each AI runs in its own PowerShell window
  2. Use 'ask <provider> <message>' to send messages
  3. Use 'pend <provider>' to view responses
  4. Use 'ping <provider>' to check connectivity

Note: The current window becomes your main control terminal.

"@
}

function Get-ProviderCommand {
    param([string]$Provider)

    switch ($Provider.ToLower()) {
        "codex" { return "codex" }
        "gemini" { return "gemini" }
        "opencode" { return "opencode" }
        "droid" { return "droid" }
        "claude" { return "claude" }
        default { return $null }
    }
}

function Start-ProviderWindow {
    param(
        [string]$Provider,
        [string]$WorkDir
    )

    $cmd = Get-ProviderCommand $Provider
    if (-not $cmd) {
        Write-Host "[ERROR] Unknown provider: $Provider" -ForegroundColor Red
        return $null
    }

    # Check if command exists
    $cmdPath = Get-Command $cmd -ErrorAction SilentlyContinue
    if (-not $cmdPath) {
        Write-Host "[ERROR] Command not found: $cmd" -ForegroundColor Red
        Write-Host "        Make sure $Provider is installed and in PATH" -ForegroundColor Yellow
        return $null
    }

    $title = "CCB-$Provider"

    Write-Host "[*] Starting $Provider..." -ForegroundColor Cyan

    # PowerShell command to run in new window
    $startupScript = @"
`$host.UI.RawUI.WindowTitle = '$title'
Set-Location '$WorkDir'
Write-Host '=== CCB $Provider Window ===' -ForegroundColor Cyan
Write-Host 'This window is managed by CCB.' -ForegroundColor Gray
Write-Host ''
$cmd
"@

    # Start new PowerShell window
    $proc = Start-Process pwsh -ArgumentList "-NoExit", "-Command", $startupScript -PassThru -ErrorAction SilentlyContinue
    if (-not $proc) {
        $proc = Start-Process powershell -ArgumentList "-NoExit", "-Command", $startupScript -PassThru
    }

    if ($proc) {
        Write-Host "[OK] $Provider started (PID: $($proc.Id), Window: $title)" -ForegroundColor Green

        # Save to registry
        Save-WindowToRegistry -Provider $Provider -Title $title -PID $proc.Id -WorkDir $WorkDir

        return @{
            Provider = $Provider
            Title = $title
            PID = $proc.Id
        }
    } else {
        Write-Host "[ERROR] Failed to start $Provider" -ForegroundColor Red
        return $null
    }
}

function Save-WindowToRegistry {
    param(
        [string]$Provider,
        [string]$Title,
        [int]$PID,
        [string]$WorkDir
    )

    $registryPath = Join-Path $env:USERPROFILE ".ccb\run\powershell-windows.json"
    $registryDir = Split-Path $registryPath -Parent

    if (-not (Test-Path $registryDir)) {
        New-Item -ItemType Directory -Path $registryDir -Force | Out-Null
    }

    $data = @{}
    if (Test-Path $registryPath) {
        try {
            $content = Get-Content $registryPath -Raw
            if ($content) {
                $data = $content | ConvertFrom-Json -AsHashtable
            }
        } catch {}
    }

    $data[$Provider] = @{
        provider = $Provider
        title = $Title
        pid = $PID
        cwd = $WorkDir
        created_at = [DateTimeOffset]::Now.ToUnixTimeSeconds()
    }

    $data | ConvertTo-Json -Depth 10 | Set-Content $registryPath -Encoding UTF8
}

function Get-RunningWindows {
    $registryPath = Join-Path $env:USERPROFILE ".ccb\run\powershell-windows.json"
    if (-not (Test-Path $registryPath)) {
        return @()
    }

    try {
        $data = Get-Content $registryPath -Raw | ConvertFrom-Json
        $result = @()
        foreach ($prop in $data.PSObject.Properties) {
            $info = $prop.Value
            # Check if process is still alive
            $proc = Get-Process -Id $info.pid -ErrorAction SilentlyContinue
            if ($proc) {
                $result += [PSCustomObject]@{
                    Provider = $info.provider
                    Title = $info.title
                    PID = $info.pid
                }
            }
        }
        return $result
    } catch {
        return @()
    }
}

function Stop-AllWindows {
    $windows = Get-RunningWindows
    if ($windows.Count -eq 0) {
        Write-Host "No CCB windows running." -ForegroundColor Yellow
        return
    }

    foreach ($w in $windows) {
        Write-Host "[*] Stopping $($w.Provider) (PID: $($w.PID))..." -ForegroundColor Cyan
        try {
            Stop-Process -Id $w.PID -Force -ErrorAction SilentlyContinue
            Write-Host "[OK] Stopped $($w.Provider)" -ForegroundColor Green
        } catch {
            Write-Host "[WARN] Could not stop $($w.Provider): $_" -ForegroundColor Yellow
        }
    }

    # Clean up registry
    $registryPath = Join-Path $env:USERPROFILE ".ccb\run\powershell-windows.json"
    if (Test-Path $registryPath) {
        Remove-Item $registryPath -Force
    }

    Write-Host ""
    Write-Host "All CCB windows stopped." -ForegroundColor Green
}

# Main logic
if ($Help) {
    Show-Help
    exit 0
}

if ($List) {
    Write-Host ""
    Write-Host "CCB Windows Status" -ForegroundColor Cyan
    Write-Host "==================" -ForegroundColor Cyan

    $windows = Get-RunningWindows
    if ($windows.Count -eq 0) {
        Write-Host "No CCB windows running." -ForegroundColor Yellow
    } else {
        foreach ($w in $windows) {
            Write-Host "  $($w.Provider): $($w.Title) (PID $($w.PID))" -ForegroundColor Green
        }
    }
    Write-Host ""
    exit 0
}

if ($Cleanup) {
    Write-Host ""
    Write-Host "CCB Cleanup" -ForegroundColor Cyan
    Write-Host "===========" -ForegroundColor Cyan
    Stop-AllWindows
    exit 0
}

if (-not $Providers -or $Providers.Count -eq 0) {
    Show-Help
    exit 0
}

# Set environment variable to use PowerShell backend
$env:CCB_BACKEND = "powershell"

$workDir = Get-Location

Write-Host ""
Write-Host "CCB Windows Native Mode" -ForegroundColor Cyan
Write-Host "=======================" -ForegroundColor Cyan
Write-Host "Starting $($Providers.Count) provider(s) in separate windows..."
Write-Host ""

$started = @()
foreach ($provider in $Providers) {
    # Don't start a window for the last provider - it runs in current terminal
    if ($provider -eq $Providers[-1]) {
        continue
    }

    $result = Start-ProviderWindow -Provider $provider -WorkDir $workDir
    if ($result) {
        $started += $result
    }
    Start-Sleep -Milliseconds 500
}

Write-Host ""
Write-Host "Windows created. Starting main AI in this terminal..." -ForegroundColor Cyan
Write-Host ""

# The last provider runs in the current terminal
$mainProvider = $Providers[-1]
$mainCmd = Get-ProviderCommand $mainProvider

if ($mainCmd) {
    $cmdPath = Get-Command $mainCmd -ErrorAction SilentlyContinue
    if ($cmdPath) {
        Write-Host "=== CCB Main Terminal: $mainProvider ===" -ForegroundColor Cyan
        Write-Host "Use 'ask <provider> <message>' to communicate with other AIs" -ForegroundColor Gray
        Write-Host "Use 'pend <provider>' to view responses" -ForegroundColor Gray
        Write-Host "Use 'ping <provider>' to check connectivity" -ForegroundColor Gray
        Write-Host ""

        # Run the main AI
        & $mainCmd
    } else {
        Write-Host "[ERROR] Command not found: $mainCmd" -ForegroundColor Red
    }
} else {
    Write-Host "[ERROR] Unknown provider: $mainProvider" -ForegroundColor Red
}
