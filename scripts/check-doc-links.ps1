# Production-maintenance documentation consistency checks.

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$passed = 0
$failed = 0

function Get-TrackedPaths {
    param([string[]]$Pathspecs)
    $paths = @(& git -C $repoRoot ls-files -- $Pathspecs)
    if ($LASTEXITCODE -ne 0) {
        throw "git ls-files failed while collecting documentation inputs"
    }
    return @($paths | Where-Object { $_ -and (Test-Path -LiteralPath (Join-Path $repoRoot $_)) })
}

function Add-Check {
    param([string]$Name, [bool]$Ok, [string]$Detail)
    if ($Ok) { $script:passed++ } else { $script:failed++ }
    $mark = if ($Ok) { "PASS" } else { "FAIL" }
    Write-Host ("  [{0}] {1} - {2}" -f $mark, $Name, $Detail)
}

Write-Host "Documentation consistency check..."

$readme = Get-Content (Join-Path $repoRoot "README.md") -Raw -Encoding UTF8
$readmeEn = Get-Content (Join-Path $repoRoot "README.en.md") -Raw -Encoding UTF8
Add-Check "README lifecycle" ($readme -match 'Production Maintenance' -and $readmeEn -match 'Production Maintenance') "production maintenance"
Add-Check "README automated regression" ($readme -match 'GitHub Actions' -and $readmeEn -match 'GitHub Actions') "workflow-owned"
Add-Check "README human acceptance" ($readme -match 'Human Sign-off' -and $readmeEn -match 'Human Sign-off') "manual product sign-off"
Add-Check "README has no deleted commands" ($readme -notmatch 'verify:demo|check:p4-r|test:p[789]|tests/gates|tests/load') "current pnpm commands only"

$wrongRouteHits = @()
$trackedRouteInputs = Get-TrackedPaths -Pathspecs @('*.md', '*.tsx', '*.ts')
$trackedRouteInputs |
    Where-Object { $_ -notmatch '^backend/' } |
    ForEach-Object {
        $relativePath = $_
        $fullPath = Join-Path $repoRoot $relativePath
        $lines = Get-Content -LiteralPath $fullPath -ErrorAction SilentlyContinue -Encoding UTF8
        for ($i = 0; $i -lt $lines.Count; $i++) {
            $line = $lines[$i]
            if ($line -match '`/task-center/failures`|"/task-center/failures"|''/task-center/failures''' -and $line -notmatch '/ops/task-center/failures|/api/v1/task-center') {
                $wrongRouteHits += "${relativePath}:$($i + 1)"
            }
        }
    }
Add-Check "Admin failure route" ($wrongRouteHits.Count -eq 0) $(if ($wrongRouteHits.Count -eq 0) { "ok" } else { ($wrongRouteHits | Select-Object -First 5) -join ", " })

$missingScriptReferences = @()
$scriptReferencePattern = '(?<![A-Za-z0-9_./-])(?:deploy/)?scripts/[A-Za-z0-9._/-]+\.(?:ps1|sh|mjs|ts|py)'
$trackedMarkdown = Get-TrackedPaths -Pathspecs @('*.md')
$trackedMarkdown |
    ForEach-Object {
        $relativeDocument = $_
        $content = Get-Content -LiteralPath (Join-Path $repoRoot $relativeDocument) -Raw -ErrorAction SilentlyContinue -Encoding UTF8
        foreach ($match in [regex]::Matches($content, $scriptReferencePattern)) {
            $relativePath = $match.Value.Replace('/', [IO.Path]::DirectorySeparatorChar)
            if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $relativePath))) {
                $missingScriptReferences += "$relativeDocument -> $($match.Value)"
            }
        }
    }
$missingScriptReferences = @($missingScriptReferences | Sort-Object -Unique)
Add-Check "Documented script paths" ($missingScriptReferences.Count -eq 0) $(if ($missingScriptReferences.Count -eq 0) { "all present" } else { ($missingScriptReferences | Select-Object -First 5) -join ", " })

$generatedDocReferencePatterns = @(
    '^docs/demo-[A-Za-z0-9._/-]+\.json$',
    '^docs/COPYWRITING_AUDIT\.auto\.md$'
)
$missingDocReferences = @()
$docReferencePattern = '(?<![A-Za-z0-9_./-])docs/[A-Za-z0-9._/-]+\.(?:md|json)'
$trackedReferenceInputs = Get-TrackedPaths -Pathspecs @('*.md', '*.go', '*.ts', '*.tsx', '*.mjs', '*.json', '*.yml', '*.yaml', '*.ps1', '*.sh')
foreach ($relativeDocument in $trackedReferenceInputs) {
    $content = Get-Content -LiteralPath (Join-Path $repoRoot $relativeDocument) -Raw -ErrorAction SilentlyContinue -Encoding UTF8
    foreach ($match in [regex]::Matches($content, $docReferencePattern)) {
        $isGeneratedOutput = $false
        foreach ($pattern in $generatedDocReferencePatterns) {
            if ($match.Value -match $pattern) {
                $isGeneratedOutput = $true
                break
            }
        }
        if ($isGeneratedOutput) { continue }
        if (-not (Test-Path -LiteralPath (Join-Path $repoRoot $match.Value))) {
            $missingDocReferences += "$relativeDocument -> $($match.Value)"
        }
    }
}
$missingDocReferences = @($missingDocReferences | Sort-Object -Unique)
Add-Check "Documented docs paths" ($missingDocReferences.Count -eq 0) $(if ($missingDocReferences.Count -eq 0) { "all stable references present" } else { ($missingDocReferences | Select-Object -First 5) -join ", " })

$apiContent = Get-Content (Join-Path $repoRoot "docs/api.md") -Raw -Encoding UTF8
Add-Check "Task-center API docs" ($apiContent -match '/api/v1/task-center/failures') "api.md"
Add-Check "Operation-workbench API docs" ($apiContent -match '/api/v1/ai/operation-workbench/summary') "api.md"

$operationalScripts = @(
    "scripts/p10-preproduction-contract.mjs",
    "scripts/p10-preproduction-preflight.mjs"
)
$missingOperational = @($operationalScripts | Where-Object { -not (Test-Path -LiteralPath (Join-Path $repoRoot $_)) })
Add-Check "Preproduction safety scripts" ($missingOperational.Count -eq 0) $(if ($missingOperational.Count -eq 0) { "present" } else { $missingOperational -join ", " })

Write-Host ""
Write-Host ("Result: {0} passed, {1} failed" -f $passed, $failed)
if ($failed -gt 0) { exit 1 }
exit 0
