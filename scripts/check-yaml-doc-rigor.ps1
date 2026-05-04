<#
.SYNOPSIS
	Check first-party YAML files for the documented header/comment workflow gate.

.DESCRIPTION
	Verifies that tracked data/config/*.yaml files retain the exact five-line
	header documented in data/config/README.md. Optionally compares
	comment-stripped YAML tokens against a base ref for comment-only changes and
	warns about newly added inline comments on obvious boolean values.

.PARAMETER BaseRef
	Git ref used for diff-based checks. Defaults to HEAD.

.PARAMETER CommentOnlyCompare
	Compare comment-stripped YAML tokens against BaseRef and fail if non-comment
	YAML content changed.

.PARAMETER WarnBooleanComments
	Warn when the current diff adds an inline comment on a true/false YAML value.
	This is a review prompt because some booleans have non-obvious side effects.

.NOTES
	Prerequisites: run from a Git worktree with git available.
	Side effects: reads tracked YAML and git diff state only.
	Safety: does not modify files, refs, generated artifacts, or config values.
#>

Param(
	[string]$BaseRef = "HEAD",
	[switch]$CommentOnlyCompare,
	[switch]$WarnBooleanComments
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

function Remove-YamlCommentText {
	Param([string]$Line)
	$inSingle = $false
	$inDouble = $false
	for ($i = 0; $i -lt $Line.Length; $i++) {
		$ch = $Line[$i]
		if ($ch -eq "'" -and -not $inDouble) {
			$inSingle = -not $inSingle
			continue
		}
		if ($ch -eq '"' -and -not $inSingle) {
			$inDouble = -not $inDouble
			continue
		}
		if ($ch -eq "#" -and -not $inSingle -and -not $inDouble) {
			if ($i -eq 0 -or [char]::IsWhiteSpace($Line[$i - 1])) {
				return $Line.Substring(0, $i).TrimEnd()
			}
		}
	}
	return $Line.TrimEnd()
}

function ConvertTo-CommentlessYaml {
	Param([string[]]$Lines)
	$out = New-Object System.Collections.Generic.List[string]
	foreach ($line in $Lines) {
		$stripped = Remove-YamlCommentText -Line $line
		if ($stripped.Trim().Length -gt 0) {
			$out.Add($stripped)
		}
	}
	return $out.ToArray()
}

$yamlFiles = @(Invoke-Git -GitArgs @("ls-files", "data/config/*.yaml"))
if ($yamlFiles.Count -eq 0) {
	throw "No tracked data/config/*.yaml files found."
}

$badHeaders = New-Object System.Collections.Generic.List[string]
foreach ($file in $yamlFiles) {
	$lines = @(Get-Content -LiteralPath $file -TotalCount 5)
	if ($lines.Count -lt 5 -or
		$lines[0] -notmatch "^# Purpose: " -or
		$lines[1] -notmatch "^# Ownership: " -or
		$lines[2] -notmatch "^# Runtime behavior: " -or
		$lines[3] -notmatch "^# Safe edits: " -or
		$lines[4] -notmatch "^# Source: data/config/README\.md\.$") {
		$badHeaders.Add($file)
	}
}

if ($badHeaders.Count -gt 0) {
	Write-Error "YAML header check failed:`n$($badHeaders -join "`n")"
	exit 1
}

Write-Output "YAML header check: pass ($($yamlFiles.Count) files)."

if ($CommentOnlyCompare) {
	$changedTokens = New-Object System.Collections.Generic.List[string]
	foreach ($file in $yamlFiles) {
		$baseLines = @(& git show "$BaseRef`:$file" 2>$null)
		if ($LASTEXITCODE -ne 0) {
			$changedTokens.Add("$file (missing from $BaseRef)")
			continue
		}
		$workLines = @(Get-Content -LiteralPath $file)
		$baseCanon = (ConvertTo-CommentlessYaml -Lines $baseLines) -join "`n"
		$workCanon = (ConvertTo-CommentlessYaml -Lines $workLines) -join "`n"
		if ($baseCanon -ne $workCanon) {
			$changedTokens.Add($file)
		}
	}
	if ($changedTokens.Count -gt 0) {
		Write-Error "Comment-only YAML comparison failed against ${BaseRef}:`n$($changedTokens -join "`n")"
		exit 1
	}
	Write-Output "Comment-only YAML comparison: pass against $BaseRef."
}

if ($WarnBooleanComments) {
	$diff = @(& git diff --unified=0 -- "data/config/*.yaml")
	if ($LASTEXITCODE -ne 0) {
		throw "git diff for boolean-comment warnings failed."
	}
	$warnings = @($diff |
		Where-Object {
			$_ -match '^\+[^+].*:\s*(true|false)\s+#' -or
			$_ -match '^\+[^+].*#.*\b(true|false)\b'
		})
	if ($warnings.Count -gt 0) {
		Write-Warning "Added boolean inline comments need review under data/config/README.md:"
		$warnings | ForEach-Object { Write-Warning $_ }
	} else {
		Write-Output "Boolean inline comment warning scan: no added warnings."
	}
}
