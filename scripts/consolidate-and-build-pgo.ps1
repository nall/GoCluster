<#
.SYNOPSIS
	Merge captured CPU profiles and build a PGO-enabled GoCluster binary.

.DESCRIPTION
	Scans the repo logs directory for cpu-*.pprof files, merges them into
	logs/pgo-merged.pprof with go tool pprof, and builds gocluster_pgo.exe with
	Go PGO enabled. The script sets its working directory to the configured repo
	root before reading profiles or building.

.NOTES
	Prerequisites: Go toolchain, git, logs/cpu-*.pprof, and the matching
	gocluster.exe used to capture those profiles.
	Side effects: writes logs/pgo-merged.pprof and gocluster_pgo.exe.
	Safety: verify profiles came from the same source binary before using the
	PGO build for performance conclusions.
#>

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repoRoot = "C:\src\gocluster"
$logsDir = Join-Path $repoRoot "logs"
$mergedProfile = Join-Path $logsDir "pgo-merged.pprof"
$outputExe = Join-Path $repoRoot "gocluster_pgo.exe"
$exePath = Join-Path $repoRoot "gocluster.exe"

Set-Location $repoRoot

if (-not (Test-Path $logsDir)) {
    Write-Error "Logs directory not found: $logsDir"
    exit 1
}

$profiles = Get-ChildItem -Path $logsDir -Filter "cpu-*.pprof" | Sort-Object LastWriteTime
if ($profiles.Count -eq 0) {
    Write-Error "No cpu-*.pprof files found in $logsDir"
    exit 1
}

# Merge profiles into a single proto pprof
Write-Host "Merging $($profiles.Count) profiles into $mergedProfile ..."
$profilePaths = $profiles | ForEach-Object { $_.FullName }
if (-not (Test-Path $exePath)) {
    Write-Error "Source binary for profiles not found: $exePath (expected same binary used to generate cpu-*.pprof)"
    exit 1
}
& go tool pprof -proto "-output=$mergedProfile" $exePath @profilePaths
if ($LASTEXITCODE -ne 0) {
    Write-Error "pprof merge failed"
    exit $LASTEXITCODE
}

function Get-GitValue {
    param(
        [Parameter(Mandatory = $true)][scriptblock]$Probe,
        [Parameter(Mandatory = $true)][string]$Default
    )
    try {
        $output = & $Probe 2>$null
        if ($LASTEXITCODE -eq 0 -and $output) {
            $trimmed = $output.ToString().Trim()
            if ($trimmed.Length -gt 0) {
                return $trimmed
            }
        }
    } catch {
        # Default is used when git metadata is unavailable.
    }
    return $Default
}

$commit = Get-GitValue -Probe { git rev-parse --short=12 HEAD } -Default "unknown"
$dirtySuffix = ""
$gitStatus = & git status --porcelain
if ($LASTEXITCODE -eq 0 -and $gitStatus) {
    $dirtySuffix = "+dirty"
}
$buildUtc = (Get-Date).ToUniversalTime()
$version = "v$($buildUtc.ToString("yy.dd.MM"))-$commit$dirtySuffix"
$buildTime = $buildUtc.ToString("yyyy-MM-ddTHH:mm:ssZ")
$ldflags = "-X main.Version=$version -X main.Commit=$commit -X main.BuildTime=$buildTime"

# Build with PGO
Write-Host "Building PGO binary -> $outputExe ..."
Write-Host "Stamping build metadata: version=$version commit=$commit built=$buildTime"
& go build "-pgo=$mergedProfile" "-ldflags=$ldflags" "-o=$outputExe" .
if ($LASTEXITCODE -ne 0) {
    Write-Error "go build failed"
    exit $LASTEXITCODE
}

Write-Host "Done. PGO profile: $mergedProfile"
Write-Host "Binary: $outputExe"
