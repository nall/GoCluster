<#
.SYNOPSIS
	Check support-critical Go files for crawler-entry comments.

.DESCRIPTION
	Reports first-party non-test Go files in support-critical areas that do not
	start with a package comment or a file-level crawler entry comment. Use this
	as a mechanical review aid; source-aware review remains responsible for
	whether the comment explains useful intent/why instead of restating code.

.PARAMETER BaseRef
	Git ref used to find changed files when ChangedOnly is set. Defaults to HEAD.

.PARAMETER ChangedOnly
	Limit checks to Go files changed relative to BaseRef.

.PARAMETER FailOnMissing
	Return a failing exit code when candidate files are missing entry comments.

.NOTES
	Prerequisites: run from a Git worktree with git available.
	Side effects: reads tracked Go files and git diff state only.
	Safety: does not modify files, refs, generated artifacts, or source code.
#>

Param(
	[string]$BaseRef = "HEAD",
	[switch]$ChangedOnly,
	[switch]$FailOnMissing
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Invoke-Git {
	Param([string[]]$GitArgs)
	$output = & git @GitArgs
	if ($LASTEXITCODE -ne 0) {
		throw "git $($GitArgs -join ' ') failed."
	}
	return $output
}

function Convert-ToRepoPath {
	Param([string]$Path)
	return ($Path -replace "\\", "/")
}

function Test-SupportCriticalPath {
	Param([string]$Path)
	$p = Convert-ToRepoPath -Path $Path
	if ($p -match "_test\.go$" -or $p -match "^third_party/") {
		return $false
	}
	if ($p -match "^internal/cluster/") { return $true }
	if ($p -match "^telnet/") { return $true }
	if ($p -match "^pathreliability/") { return $true }
	if ($p -match "^spot/") { return $true }
	if ($p -match "^filter/user_record\.go$") { return $true }
	if ($p -match "^reputation/gate\.go$") { return $true }
	if ($p -match "^cmd/rbn_replay/") { return $true }
	if ($p -match "^cmd/callcorr_reveng_rebuilt/") { return $true }
	return $false
}

function Test-CrawlerEntryComment {
	Param([string]$Path)
	$lines = @(Get-Content -LiteralPath $Path -TotalCount 24)
	if ($lines.Count -eq 0) {
		return $false
	}
	$header = ($lines -join "`n")
	return (
		$header -match "(?m)^// Package " -or
		$header -match "(?m)^// File role: " -or
		$header -match "(?m)^// Crawler notes:" -or
		$header -match "(?m)^// Program "
	)
}

if ($ChangedOnly) {
	$files = @(Invoke-Git -GitArgs @("diff", "--name-only", $BaseRef, "--", "*.go"))
} else {
	$files = @(Invoke-Git -GitArgs @("ls-files", "*.go"))
}

$candidates = New-Object System.Collections.Generic.List[string]
foreach ($file in $files) {
	if ((Test-SupportCriticalPath -Path $file) -and (Test-Path -LiteralPath $file)) {
		$candidates.Add($file)
	}
}

$missing = New-Object System.Collections.Generic.List[string]
foreach ($file in $candidates) {
	if (-not (Test-CrawlerEntryComment -Path $file)) {
		$missing.Add($file)
	}
}

Write-Output "Go crawler-entry candidate check: scanned $($candidates.Count) support-critical Go file(s)."
if ($missing.Count -eq 0) {
	Write-Output "Go crawler-entry comments: pass."
	exit 0
}

Write-Output "Go crawler-entry comments missing:"
$missing | ForEach-Object { Write-Output "  $_" }

if ($FailOnMissing) {
	exit 1
}
