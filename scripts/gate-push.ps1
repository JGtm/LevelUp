# scripts/gate-push.ps1
#
# Wrapper PowerShell natif du filet local `make gate-push`.
#
# Pourquoi : sur ce poste, l'environnement git-bash casse le lien des binaires
# de test Go qui embarquent `libduckdb_static` (erreur deterministe
# "undefined reference __emutls_v._ZSt11__once_call", reproduite le
# 2026-08-03, 4 gate-push rouges) -- PowerShell natif lie correctement.
# Contournement valide, documente dans .ai/HANDOFF_POST_LOT2_V73.md
# ("Pieges operationnels du poste", #1) : produire le JSONL `go test -json`
# depuis PowerShell natif, puis faire verifier ce JSONL par
# scripts/check_test_baseline.sh (bash) en mode consommateur (--from-jsonl),
# qui ne relance PAS la suite -- il ne fait que parser le fichier.
#
# La CI reste l'autorite : ce script est un filet LOCAL, pas un remplacement.
#
# 4 maillons, dans l'ordre du Makefile `gate-push` :
#   1. Go lint      (golangci-lint, ratchet --new-from-merge-base=origin/main)
#   2. Go tests      (integration, JSONL PowerShell -> check_test_baseline.sh)
#   3. Web typecheck (make check-types)
#   4. Web lint      (npm run lint)
#
# Tous les maillons s'executent meme si un maillon precedent echoue (vision
# complete en un seul passage, comme la CI) ; le code de sortie global reflete
# fidelement l'etat (0 = tout vert, 1 = au moins un maillon rouge).
#
# Usage : pwsh -File scripts/gate-push.ps1
#         powershell -File scripts/gate-push.ps1

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$goApiDir = Join-Path $repoRoot 'apps\go-api'
$webDir = Join-Path $repoRoot 'apps\web'
$jsonlPath = Join-Path $goApiDir 'baseline_current.jsonl'
$jsonlRelative = 'apps/go-api/baseline_current.jsonl' # chemin POSIX pour bash, relatif a repoRoot
$stderrLogPath = Join-Path $goApiDir 'baseline_current.stderr.log'

$results = New-Object System.Collections.Generic.List[object]

function Write-StepHeader {
    param([string]$Text)
    Write-Host ''
    Write-Host ('=== {0} ===' -f $Text) -ForegroundColor Cyan
}

function Add-Result {
    param([string]$Name, [bool]$Ok, [string]$Detail = '')
    $results.Add([pscustomobject]@{ Name = $Name; Ok = $Ok; Detail = $Detail })
    if ($Ok) {
        Write-Host ('[OK]   {0}' -f $Name) -ForegroundColor Green
    } else {
        Write-Host ('[FAIL] {0}{1}' -f $Name, $(if ($Detail) { " -- $Detail" } else { '' })) -ForegroundColor Red
    }
}

# ---------------------------------------------------------------------------
# Maillon 1/4 -- Go lint (golangci-lint, meme ratchet que la CI)
# ---------------------------------------------------------------------------
Write-StepHeader '1/4 -- Go lint (golangci-lint, ratchet origin/main)'
$golangciLint = Get-Command golangci-lint -ErrorAction SilentlyContinue
if ($null -eq $golangciLint) {
    Write-Host 'golangci-lint absent du PATH -- gate impossible a reproduire.' -ForegroundColor Yellow
    Write-Host '  Installation : https://golangci-lint.run/usage/install/ (version CI : v2.12.2)'
    Add-Result -Name 'Go lint' -Ok $false -Detail 'golangci-lint absent du PATH'
} else {
    Push-Location $goApiDir
    try {
        & golangci-lint run --timeout 5m --new-from-merge-base=origin/main
        $lintExit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    Add-Result -Name 'Go lint' -Ok ($lintExit -eq 0) -Detail "exit=$lintExit"
}

# ---------------------------------------------------------------------------
# Maillon 2/4 -- Go tests (integration) : JSONL produit par PowerShell natif,
# verifie par check_test_baseline.sh (bash, mode consommateur --from-jsonl).
# ---------------------------------------------------------------------------
Write-StepHeader '2/4 -- Go tests (integration, JSONL PowerShell natif)'
Push-Location $goApiDir
try {
    $env:CGO_ENABLED = '1'
    Write-Host '  Lancement de la suite (peut prendre plusieurs minutes)...'
    # -timeout=600s (defaut Go) : 300s faisait paniquer internal/sync (~173s isole,
    # davantage sous la charge du run serialise complet) depuis les tests
    # d'autorite de schema qui rejouent des chaines de migrations completes.
    $testOutput = & go test -tags=integration -count=1 -timeout=600s -p 1 -json ./... 2> $stderrLogPath
    $testExit = $LASTEXITCODE
} finally {
    Pop-Location
}
# Encodage UTF-8 SANS BOM explicite : un BOM en tete de fichier corromprait la
# toute premiere ligne JSON lue par check_test_baseline.sh (piege encodage
# PowerShell 5.1 documente au handoff, point 2).
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllLines($jsonlPath, [string[]]$testOutput, $utf8NoBom)
Write-Host ("  go test exit={0} -- {1} ligne(s) JSONL ecrite(s) dans {2}" -f $testExit, $testOutput.Count, $jsonlRelative)

if ($testOutput.Count -eq 0) {
    # Rien n'a ete emis : echec d'infrastructure (le meme piege de lien que sur
    # git-bash, un go.mod introuvable, etc.) -- pas la peine d'invoquer
    # check_test_baseline.sh, il n'y a rien a analyser.
    Add-Result -Name 'Go tests (integration)' -Ok $false -Detail "go test exit=$testExit, JSONL vide (echec avant tout test -- voir $stderrLogPath)"
} else {
    Push-Location $repoRoot
    try {
        & bash scripts/check_test_baseline.sh tests --from-jsonl $jsonlRelative
        $baselineExit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    Add-Result -Name 'Go tests (integration)' -Ok ($baselineExit -eq 0) -Detail "check_test_baseline exit=$baselineExit (go test exit=$testExit)"
}

# ---------------------------------------------------------------------------
# Maillon 3/4 -- Web typecheck
# ---------------------------------------------------------------------------
Write-StepHeader '3/4 -- Web typecheck (make check-types)'
Push-Location $repoRoot
try {
    & make check-types
    $typecheckExit = $LASTEXITCODE
} finally {
    Pop-Location
}
Add-Result -Name 'Web typecheck' -Ok ($typecheckExit -eq 0) -Detail "exit=$typecheckExit"

# ---------------------------------------------------------------------------
# Maillon 4/4 -- Web lint
# ---------------------------------------------------------------------------
Write-StepHeader '4/4 -- Web lint (eslint)'
Push-Location $webDir
try {
    & npm run lint
    $webLintExit = $LASTEXITCODE
} finally {
    Pop-Location
}
Add-Result -Name 'Web lint' -Ok ($webLintExit -eq 0) -Detail "exit=$webLintExit"

# ---------------------------------------------------------------------------
# Recapitulatif -- code de sortie global fidele a l'etat reel des 4 maillons.
# ---------------------------------------------------------------------------
Write-Host ''
Write-Host '=== Recapitulatif gate-push ===' -ForegroundColor Cyan
foreach ($r in $results) {
    $mark = if ($r.Ok) { '[OK]  ' } else { '[FAIL]' }
    $color = if ($r.Ok) { 'Green' } else { 'Red' }
    Write-Host ('  {0} {1}{2}' -f $mark, $r.Name, $(if ($r.Detail) { " ($($r.Detail))" } else { '' })) -ForegroundColor $color
}

$failed = @($results | Where-Object { -not $_.Ok })
if ($failed.Count -gt 0) {
    Write-Host ''
    Write-Host ("gate-push : {0}/{1} maillon(s) en echec. CI reste l'autorite finale." -f $failed.Count, $results.Count) -ForegroundColor Red
    exit 1
}

Write-Host ''
Write-Host 'gate-push : 4/4 maillons verts.' -ForegroundColor Green
exit 0
