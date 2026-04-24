param(
    [string]$Version = "",
    [string]$OutputDir = "dist",
    [string]$PackageName = "gocluster-windows-amd64"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Resolve-RepoRoot {
    $root = & git rev-parse --show-toplevel
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($root)) {
        throw "Unable to resolve repository root with git."
    }
    return $root.Trim()
}

function Copy-TrackedPayload {
    param(
        [string]$RepoRoot,
        [string]$StageRoot,
        [string[]]$AllowlistPrefixes
    )

    $tracked = & git -C $RepoRoot ls-files data
    if ($LASTEXITCODE -ne 0) {
        throw "git ls-files data failed."
    }

    foreach ($relativePath in $tracked) {
        $normalized = $relativePath -replace "\\", "/"
        $allowed = $false
        foreach ($prefix in $AllowlistPrefixes) {
            if ($normalized -eq $prefix -or $normalized.StartsWith($prefix + "/")) {
                $allowed = $true
                break
            }
        }
        if (-not $allowed) {
            continue
        }

        if ($normalized -eq "data/config/openai.yaml") {
            throw "Refusing to package secret-bearing data/config/openai.yaml."
        }

        $source = Join-Path $RepoRoot ($normalized -replace "/", [IO.Path]::DirectorySeparatorChar)
        $destination = Join-Path $StageRoot ($normalized -replace "/", [IO.Path]::DirectorySeparatorChar)
        $destinationDir = Split-Path -Parent $destination
        New-Item -ItemType Directory -Path $destinationDir -Force | Out-Null
        Copy-Item -LiteralPath $source -Destination $destination -Force
    }
}

function Assert-ForbiddenPayloadAbsent {
    param([string]$StageRoot)

    $forbiddenPatterns = @(
        "data/config/openai.yaml",
        "data/archive",
        "data/grids",
        "data/ipinfo",
        "data/scp",
        "data/logs",
        "logs",
        "data/users",
        "data/reputation",
        "data/fcc",
        "data/rbn"
    )

    foreach ($pattern in $forbiddenPatterns) {
        $path = Join-Path $StageRoot ($pattern -replace "/", [IO.Path]::DirectorySeparatorChar)
        if (Test-Path -LiteralPath $path) {
            throw "Forbidden release payload path found: $pattern"
        }
    }
}

function Copy-ReleaseDocument {
    param(
        [string]$RepoRoot,
        [string]$StageRoot,
        [string]$SourceRelativePath,
        [string]$DestinationRelativePath
    )

    $source = Join-Path $RepoRoot ($SourceRelativePath -replace "/", [IO.Path]::DirectorySeparatorChar)
    if (-not (Test-Path -LiteralPath $source)) {
        throw "Required release document is missing: $SourceRelativePath"
    }

    $destination = Join-Path $StageRoot ($DestinationRelativePath -replace "/", [IO.Path]::DirectorySeparatorChar)
    $destinationDir = Split-Path -Parent $destination
    if (-not [string]::IsNullOrWhiteSpace($destinationDir)) {
        New-Item -ItemType Directory -Path $destinationDir -Force | Out-Null
    }
    Copy-Item -LiteralPath $source -Destination $destination -Force
}

function Invoke-GoRunHost {
    param([string[]]$Arguments)

    $oldGOOS = $env:GOOS
    $oldGOARCH = $env:GOARCH
    try {
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        & go @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "go $($Arguments -join ' ') failed."
        }
    }
    finally {
        if ($null -eq $oldGOOS) {
            Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        }
        else {
            $env:GOOS = $oldGOOS
        }
        if ($null -eq $oldGOARCH) {
            Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        }
        else {
            $env:GOARCH = $oldGOARCH
        }
    }
}

function Render-ReleaseReadme {
    param(
        [string]$RepoRoot,
        [string]$StageRoot
    )

    $templatePath = Join-Path $RepoRoot "docs/release/README.md.template"
    $configDir = Join-Path $StageRoot "data/config"
    $outputPath = Join-Path $StageRoot "README.md"
    Invoke-GoRunHost -Arguments @(
        "run",
        "./cmd/release_readme",
        "-template",
        $templatePath,
        "-config-dir",
        $configDir,
        "-out",
        $outputPath
    )
}

function Read-StagedPayloadText {
    param(
        [string]$StageRoot,
        [string]$RelativePath
    )

    $path = Join-Path $StageRoot ($RelativePath -replace "/", [IO.Path]::DirectorySeparatorChar)
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Required release payload file is missing: $RelativePath"
    }
    return Get-Content -LiteralPath $path -Raw
}

function Assert-MatchingLinesAllowed {
    param(
        [string]$Text,
        [string]$LinePattern,
        [string]$AllowedPattern,
        [string]$Description
    )

    foreach ($line in ($Text -split "\r?\n")) {
        if ($line.TrimStart().StartsWith("#")) {
            continue
        }
        if ($line -match $LinePattern -and $line -notmatch $AllowedPattern) {
            throw "$Description contains a non-public release value."
        }
    }
}

function Assert-RequiredLinePresent {
    param(
        [string]$Text,
        [string]$AllowedPattern,
        [string]$Description
    )

    if ($Text -notmatch $AllowedPattern) {
        throw "$Description is missing the required public release value."
    }
}

function Assert-PublicReleaseConfig {
    param([string]$StageRoot)

    $app = Read-StagedPayloadText -StageRoot $StageRoot -RelativePath "data/config/app.yaml"
    $ingest = Read-StagedPayloadText -StageRoot $StageRoot -RelativePath "data/config/ingest.yaml"
    $peering = Read-StagedPayloadText -StageRoot $StageRoot -RelativePath "data/config/peering.yaml"
    $reputation = Read-StagedPayloadText -StageRoot $StageRoot -RelativePath "data/config/reputation.yaml"

    Assert-RequiredLinePresent -Text $app `
        -AllowedPattern '(?m)^\s*node_id:\s*"N0CALL-\d+"\s*(#.*)?$' `
        -Description "data/config/app.yaml server.node_id"

    Assert-MatchingLinesAllowed -Text $ingest `
        -LinePattern '^\s*callsign:\s*' `
        -AllowedPattern '^\s*callsign:\s*"N0CALL-\d+"\s*(#.*)?$' `
        -Description "data/config/ingest.yaml callsign"
    Assert-MatchingLinesAllowed -Text $ingest `
        -LinePattern '^\s*host:\s*' `
        -AllowedPattern '^\s*host:\s*"(telnet\.reversebeacon\.net|upstream\.example\.invalid)"\s*(#.*)?$' `
        -Description "data/config/ingest.yaml host"

    Assert-RequiredLinePresent -Text $peering `
        -AllowedPattern '(?m)^\s*local_callsign:\s*"N0CALL-\d+"\s*(#.*)?$' `
        -Description "data/config/peering.yaml local_callsign"
    Assert-MatchingLinesAllowed -Text $peering `
        -LinePattern '^\s*enabled:\s*' `
        -AllowedPattern '^\s*enabled:\s*false\s*(#.*)?$' `
        -Description "data/config/peering.yaml enabled"
    Assert-MatchingLinesAllowed -Text $peering `
        -LinePattern '^\s*host:\s*' `
        -AllowedPattern '^\s*host:\s*"peer\d+\.example\.invalid"\s*(#.*)?$' `
        -Description "data/config/peering.yaml host"
    Assert-MatchingLinesAllowed -Text $peering `
        -LinePattern '^\s*password:\s*' `
        -AllowedPattern '^\s*password:\s*""\s*(#.*)?$' `
        -Description "data/config/peering.yaml password"
    Assert-MatchingLinesAllowed -Text $peering `
        -LinePattern '^\s*login_callsign:\s*' `
        -AllowedPattern '^\s*login_callsign:\s*"N0CALL-\d+"\s*(#.*)?$' `
        -Description "data/config/peering.yaml login_callsign"
    Assert-MatchingLinesAllowed -Text $peering `
        -LinePattern '^\s*remote_callsign:\s*' `
        -AllowedPattern '^\s*remote_callsign:\s*"N0PEER-\d+"\s*(#.*)?$' `
        -Description "data/config/peering.yaml remote_callsign"

    Assert-RequiredLinePresent -Text $reputation `
        -AllowedPattern '(?m)^\s*ipinfo_download_enabled:\s*false\s*(#.*)?$' `
        -Description "data/config/reputation.yaml ipinfo_download_enabled"
    Assert-RequiredLinePresent -Text $reputation `
        -AllowedPattern '(?m)^\s*ipinfo_download_token:\s*"REPLACE_WITH_IPINFO_TOKEN"\s*(#.*)?$' `
        -Description "data/config/reputation.yaml ipinfo_download_token"
    Assert-RequiredLinePresent -Text $reputation `
        -AllowedPattern '(?m)^\s*ipinfo_api_enabled:\s*false\s*(#.*)?$' `
        -Description "data/config/reputation.yaml ipinfo_api_enabled"
    Assert-RequiredLinePresent -Text $reputation `
        -AllowedPattern '(?m)^\s*ipinfo_api_token:\s*""\s*(#.*)?$' `
        -Description "data/config/reputation.yaml ipinfo_api_token"
}

$repoRoot = Resolve-RepoRoot
Push-Location $repoRoot
try {
    $commit = (& git rev-parse --short=12 HEAD).Trim()
    if ([string]::IsNullOrWhiteSpace($Version)) {
        $Version = "dev-$commit"
    }
    $buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

    $outputRoot = Join-Path $repoRoot $OutputDir
    $stageRoot = Join-Path $outputRoot $PackageName
    $zipPath = Join-Path $outputRoot "$PackageName.zip"

    if (Test-Path -LiteralPath $stageRoot) {
        Remove-Item -LiteralPath $stageRoot -Recurse -Force
    }
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    New-Item -ItemType Directory -Path $stageRoot -Force | Out-Null

    Copy-TrackedPayload -RepoRoot $repoRoot -StageRoot $stageRoot -AllowlistPrefixes @(
        "data/config",
        "data/cty",
        "data/h3",
        "data/peers/topology.db",
        "data/skm_correction/rbnskew.json"
    )
    Render-ReleaseReadme -RepoRoot $repoRoot -StageRoot $stageRoot
    Copy-ReleaseDocument -RepoRoot $repoRoot -StageRoot $stageRoot `
        -SourceRelativePath "docs/OPERATOR_GUIDE.md" `
        -DestinationRelativePath "docs/OPERATOR_GUIDE.md"
    Assert-PublicReleaseConfig -StageRoot $stageRoot
    Assert-ForbiddenPayloadAbsent -StageRoot $stageRoot

    $exePath = Join-Path $stageRoot "gocluster.exe"
    $ldflags = "-X main.Version=$Version -X main.Commit=$commit -X main.BuildTime=$buildTime"

    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    & go build -trimpath -ldflags $ldflags -o $exePath .
    if ($LASTEXITCODE -ne 0) {
        throw "go build failed."
    }

    Push-Location $outputRoot
    try {
        Compress-Archive -Path $PackageName -DestinationPath $zipPath -Force
    }
    finally {
        Pop-Location
    }
    Write-Host "Release package: $zipPath"
}
finally {
    Pop-Location
}
