<#
.SYNOPSIS
    Prepare les deux arbres de ressources de l'installeur : le MOTEUR et le PAQUET DE RESSOURCES.

.DESCRIPTION
    Sous Windows, le dossier des ressources de Tauri EST le dossier d'installation : il devient la
    racine de ressources du serveur (LEVELUP_RESOURCES_DIR, ADR 0008 amendee). Deux unites sont
    livrees a part (ADR 0045, amendement L24) :

    LE MOTEUR, change a chaque version, dans apps/desktop/src-tauri/bundle-staging (ignore par git) :

      web/                    le front compile SANS titles/ (LEVELUP_WEB_DIST par defaut en desktop)
      config/                 les manifestes de titre et les mappings (couples au code)
      THIRD_PARTY_NOTICES.md, THIRD_PARTY_CARGO.md, NOTICE, LICENSE

    LE PAQUET, qui change rarement, dans apps/desktop/src-tauri/pack-staging (ignore par git) :

      static/                 images et sons
      web/titles/             les images des titres (plaques, embleme, equipes)
      data/titles/...         le referentiel versionne (cartes, profils de film, base metadata
                              pre-construite) : UNIQUEMENT les fichiers suivis par git, jamais
                              les donnees locales du poste
      ffmpeg.exe, ffprobe.exe, *.dll, LICENSE.LGPL   (fetch-ffmpeg.ps1)
      resources.manifest.json le manifeste, dont l'empreinte est l'identite du paquet

    L'empreinte (SHA-256 du manifeste trie) est ecrite dans pack-staging.fingerprint, a cote : le
    serveur la recoit a la compilation (-X main.resourcesFingerprint=<empreinte>), il doit donc etre
    compile APRES ce script. Avec -ArchiveDir, l'archive levelup-resources-<empreinte>.zip est aussi
    produite. L'installeur COMPLET embarque les deux arbres ; l'installeur "moteur seul" de
    l'updater n'embarque que bundle-staging.

    Le dossier data/cache/career-rank-image n'est PAS livre : le serveur les cherche sous la racine
    de DONNEES, que l'installeur ne remplit pas, et les retelecharge a la demande ; 17 Mo de plus
    a chaque mise a jour (D-9) pour une image qui revient du reseau de toute facon.

.PARAMETER WebDist
    Le front compile (apps/web/dist).

.PARAMETER FfmpegDir
    La sortie de fetch-ffmpeg.ps1.

.PARAMETER Out
    Dossier du moteur (defaut : apps/desktop/src-tauri/bundle-staging), recree a chaque appel.

.PARAMETER PackOut
    Dossier du paquet (defaut : apps/desktop/src-tauri/pack-staging), recree a chaque appel.

.PARAMETER ArchiveDir
    Si donne, y ecrit levelup-resources-<empreinte>.zip (cree le dossier).

.PARAMETER Version
    Version documentaire inscrite au manifeste (jamais comparee).
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$WebDist,
    [Parameter(Mandatory = $true)][string]$FfmpegDir,
    [string]$Out = '',
    [string]$PackOut = '',
    [string]$ArchiveDir = '',
    [string]$Version = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$tauri = Join-Path $repo 'apps\desktop\src-tauri'
if (-not $Out) { $Out = Join-Path $tauri 'bundle-staging' }
if (-not $PackOut) { $PackOut = Join-Path $tauri 'pack-staging' }

foreach ($required in @(
        (Join-Path $WebDist 'index.html'),
        (Join-Path $FfmpegDir 'ffmpeg.exe'),
        (Join-Path $FfmpegDir 'ffprobe.exe'))) {
    if (-not (Test-Path $required)) { throw "Entree absente : $required" }
}

foreach ($dir in $Out, $PackOut) {
    if (Test-Path $dir) { Remove-Item -Recurse -Force $dir }
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}

function Copy-Tree {
    param([string]$From, [string]$To)
    New-Item -ItemType Directory -Force -Path $To | Out-Null
    Copy-Item -Path (Join-Path $From '*') -Destination $To -Recurse -Force
}

Copy-Tree $WebDist (Join-Path $Out 'web')
Copy-Tree (Join-Path $repo 'config') (Join-Path $Out 'config')
Copy-Tree (Join-Path $repo 'static') (Join-Path $PackOut 'static')
# Les images des titres sortent du front : elles appartiennent au paquet, pas au moteur.
$titles = Join-Path $Out 'web\titles'
if (-not (Test-Path $titles)) { throw "web/titles absent du front compile : $titles" }
New-Item -ItemType Directory -Force -Path (Join-Path $PackOut 'web') | Out-Null
Move-Item -Path $titles -Destination (Join-Path $PackOut 'web\titles')

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
    $dest = Join-Path $PackOut ($rel -replace '/', '\')
    New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
    Copy-Item -Path (Join-Path $repo ($rel -replace '/', '\')) -Destination $dest
}

Copy-Item -Path (Join-Path $FfmpegDir '*') -Destination $PackOut -Recurse -Force

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

# Le manifeste du paquet : un seul generateur (cmd/resources-manifest), le meme que celui du Dockerfile.
$tool = Join-Path ([System.IO.Path]::GetTempPath()) ('resources-manifest-' + [Guid]::NewGuid().ToString('N') + '.exe')
Push-Location (Join-Path $repo 'apps\go-api')
try {
    $env:CGO_ENABLED = '0'
    & go build -o $tool ./cmd/resources-manifest
    if ($LASTEXITCODE -ne 0) { throw 'go build de resources-manifest a echoue' }
}
finally { Pop-Location }
$toolArgs = @('-dir', $PackOut)
if ($Version) { $toolArgs += @('-version', $Version) }
if ($ArchiveDir) {
    New-Item -ItemType Directory -Force -Path $ArchiveDir | Out-Null
    $archiveTmp = Join-Path $ArchiveDir 'levelup-resources.zip.tmp'
    $toolArgs += @('-zip', $archiveTmp)
}
$fingerprint = ((& $tool @toolArgs) | Select-Object -Last 1).Trim()
if ($LASTEXITCODE -ne 0 -or $fingerprint -notmatch '^[0-9a-f]{64}$') { throw "resources-manifest a echoue (sortie : $fingerprint)" }
Remove-Item -Force $tool
if ($ArchiveDir) {
    Move-Item -Force -Path $archiveTmp -Destination (Join-Path $ArchiveDir "levelup-resources-$fingerprint.zip")
}
$fingerprintFile = ($PackOut.TrimEnd('\', '/')) + '.fingerprint'
[System.IO.File]::WriteAllText($fingerprintFile, $fingerprint, (New-Object System.Text.UTF8Encoding($false)))

$engineSize = (Get-ChildItem -Recurse -File $Out | Measure-Object Length -Sum).Sum
$packSize = (Get-ChildItem -Recurse -File $PackOut | Measure-Object Length -Sum).Sum
Write-Host ("Moteur assemble dans {0} ({1:N1} Mo)" -f $Out, ($engineSize / 1MB))
Write-Host ("Paquet de ressources assemble dans {0} ({1:N1} Mo), empreinte {2}" -f $PackOut, ($packSize / 1MB), $fingerprint)
