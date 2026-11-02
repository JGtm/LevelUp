<#
.SYNOPSIS
    Prepare le dossier des ressources que l'installeur depose a cote de l'application.

.DESCRIPTION
    Sous Windows, le dossier des ressources de Tauri EST le dossier d'installation : il devient la
    racine de ressources du serveur (LEVELUP_RESOURCES_DIR, ADR 0008 amendee). Ce script en
    assemble le contenu dans apps/desktop/src-tauri/bundle-staging (ignore par git), que
    tauri.bundle.conf.json designe :

      web/                    le front compile (LEVELUP_WEB_DIST par defaut en desktop)
      config/                 les manifestes de titre et les mappings
      static/                 images et sons
      data/titles/...         le referentiel versionne (cartes, profils de film, base metadata
                              pre-construite) : UNIQUEMENT les fichiers suivis par git, jamais
                              les donnees locales du poste
      ffmpeg.exe, ffprobe.exe, *.dll, LICENSE.LGPL   (fetch-ffmpeg.ps1)
      THIRD_PARTY_NOTICES.md, THIRD_PARTY_CARGO.md, NOTICE, LICENSE

    Le dossier data/cache/career-rank-image n'est PAS livre : le serveur les cherche sous la racine
    de DONNEES, que l'installeur ne remplit pas, et les retelecharge a la demande ; 17 Mo de plus
    a chaque mise a jour (D-9) pour une image qui revient du reseau de toute facon.

.PARAMETER WebDist
    Le front compile (apps/web/dist).

.PARAMETER FfmpegDir
    La sortie de fetch-ffmpeg.ps1.

.PARAMETER Out
    Dossier de sortie (defaut : apps/desktop/src-tauri/bundle-staging), recree a chaque appel.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$WebDist,
    [Parameter(Mandatory = $true)][string]$FfmpegDir,
    [string]$Out = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$tauri = Join-Path $repo 'apps\desktop\src-tauri'
if (-not $Out) { $Out = Join-Path $tauri 'bundle-staging' }

foreach ($required in @(
        (Join-Path $WebDist 'index.html'),
        (Join-Path $FfmpegDir 'ffmpeg.exe'),
        (Join-Path $FfmpegDir 'ffprobe.exe'))) {
    if (-not (Test-Path $required)) { throw "Entree absente : $required" }
}

if (Test-Path $Out) { Remove-Item -Recurse -Force $Out }
New-Item -ItemType Directory -Force -Path $Out | Out-Null

function Copy-Tree {
    param([string]$From, [string]$To)
    New-Item -ItemType Directory -Force -Path $To | Out-Null
    Copy-Item -Path (Join-Path $From '*') -Destination $To -Recurse -Force
}

Copy-Tree $WebDist (Join-Path $Out 'web')
Copy-Tree (Join-Path $repo 'config') (Join-Path $Out 'config')
Copy-Tree (Join-Path $repo 'static') (Join-Path $Out 'static')

# Referentiel versionne : la liste vient de git, pas du disque (le poste de dev porte des Go de donnees).
Push-Location $repo
try {
    $files = & git ls-files -z -- 'data/titles/*/reference/*' 'data/titles/*/warehouse/metadata-prebuilt.zip'
    if ($LASTEXITCODE -ne 0) { throw 'git ls-files a echoue' }
}
finally { Pop-Location }
$tracked = ($files -join '') -split "`0" | Where-Object { $_ }
if (-not $tracked) { throw 'Aucun fichier de referentiel suivi par git' }
foreach ($rel in $tracked) {
    $dest = Join-Path $Out ($rel -replace '/', '\')
    New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
    Copy-Item -Path (Join-Path $repo ($rel -replace '/', '\')) -Destination $dest
}

Copy-Item -Path (Join-Path $FfmpegDir '*') -Destination $Out -Recurse -Force

foreach ($doc in 'NOTICE', 'LICENSE') {
    Copy-Item -Path (Join-Path $repo $doc) -Destination (Join-Path $Out $doc)
}
Copy-Item -Path (Join-Path $repo 'apps\desktop\THIRD_PARTY_NOTICES.md') -Destination $Out

# Dependances Rust de la coquille : nom, version, licence, depuis le verrou (aucun outil a installer).
Push-Location $tauri
try {
    $meta = (& cargo metadata --format-version 1 --locked --filter-platform x86_64-pc-windows-msvc) -join "`n" | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) { throw 'cargo metadata a echoue' }
}
finally { Pop-Location }
$workspace = [System.Collections.Generic.HashSet[string]]($meta.workspace_members)
$lines = @(
    '# Rust dependencies of the desktop shell',
    '',
    'Generated from `Cargo.lock` by `apps/desktop/scripts/stage-bundle.ps1` (cargo metadata, Windows target).',
    '',
    '| Crate | Version | Licence |',
    '|---|---|---|'
)
$meta.packages | Where-Object { -not $workspace.Contains($_.id) } | Sort-Object name, version | ForEach-Object {
    $license = if ($_.license) { $_.license } elseif ($_.license_file) { "see $($_.license_file)" } else { 'unspecified' }
    $lines += "| $($_.name) | $($_.version) | $license |"
}
[System.IO.File]::WriteAllLines((Join-Path $Out 'THIRD_PARTY_CARGO.md'), $lines, (New-Object System.Text.UTF8Encoding($false)))

$size = (Get-ChildItem -Recurse -File $Out | Measure-Object Length -Sum).Sum
Write-Host ("Ressources assemblees dans {0} ({1:N1} Mo)" -f $Out, ($size / 1MB))
