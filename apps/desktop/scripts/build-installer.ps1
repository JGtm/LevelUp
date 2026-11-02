<#
.SYNOPSIS
    Construit l'installeur NSIS de LevelUp en local : les memes etapes que le job `desktop-build` de
    .github/workflows/release.yml, sans secret.

.DESCRIPTION
    1. front compile (apps/web, sauf -SkipWeb) ;
    2. serveur Go (CGO, DuckDB lie) pose sous le nom du sidecar, version dans -ldflags ;
    3. ffmpeg et ffprobe LGPL (fetch-ffmpeg.ps1 : archive epinglee, somme de controle verifiee) ;
    4. ressources assemblees (stage-bundle.ps1) ;
    5. `npx tauri build` avec tauri.bundle.conf.json.

    Sans -SigningKey, aucune cle n'est necessaire : le build produit l'installeur seul
    (`createUpdaterArtifacts` reste faux). Avec -SigningKey (chemin du fichier de cle privee de
    l'updater), le CLI Tauri signe aussi l'installeur pour l'updater (fichier .sig a cote) ;
    -SigningPassword donne son mot de passe, s'il en a un.

    Resultat : apps/desktop/src-tauri/target/release/bundle/nsis/LevelUp_<version>_x64-setup.exe

.PARAMETER SkipWeb
    Reutilise apps/web/dist tel quel.

.PARAMETER SigningKey
    Cle privee de l'updater : produit aussi le fichier .sig.

.PARAMETER SigningPassword
    Mot de passe de la cle (defaut : aucun).

.EXAMPLE
    powershell -File apps/desktop/scripts/build-installer.ps1
#>
[CmdletBinding()]
param(
    [switch]$SkipWeb,
    [string]$SigningKey = '',
    [string]$SigningPassword = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$desktop = Join-Path $repo 'apps\desktop'
$tauri = Join-Path $desktop 'src-tauri'
$scripts = Join-Path $desktop 'scripts'

function Invoke-Native {
    param([string]$What, [scriptblock]$Command)
    & $Command
    if ($LASTEXITCODE -ne 0) { throw "$What a echoue (code $LASTEXITCODE)" }
}

# La version : celle de Cargo.toml (tauri.conf.json doit lui etre egale : check-version.sh).
$version = ((Select-String -Path (Join-Path $tauri 'Cargo.toml') -Pattern '^version\s*=\s*"([^"]+)"' | Select-Object -First 1).Matches[0].Groups[1].Value)
if (-not $version) { throw 'version introuvable dans Cargo.toml' }

$triple = ((& rustc -vV) | Where-Object { $_ -like 'host:*' }) -replace '^host:\s*', ''
if (-not $triple) { throw 'rustc introuvable : le triple de la cible ne peut pas etre lu.' }

if (-not $SkipWeb) {
    Push-Location (Join-Path $repo 'apps\web')
    try {
        if (-not (Test-Path 'node_modules')) { Invoke-Native 'npm ci' { npm ci --prefer-offline } }
        Invoke-Native 'npm run build' { npm run build }
    }
    finally { Pop-Location }
}

$env:CGO_ENABLED = '1'
$gcc = 'C:\msys64\ucrt64\bin'
if ((Test-Path $gcc) -and ($env:PATH -notlike "*$gcc*")) { $env:PATH = "$gcc;$env:PATH" }
$binaries = Join-Path $tauri 'binaries'
New-Item -ItemType Directory -Force -Path $binaries | Out-Null
$server = Join-Path $binaries "levelup-server-$triple.exe"
Push-Location (Join-Path $repo 'apps\go-api')
try { Invoke-Native 'go build du serveur' { go build -ldflags "-X main.version=$version" -o $server ./cmd/server } }
finally { Pop-Location }

$ffmpeg = Join-Path $env:TEMP 'levelup-ffmpeg-out'
& (Join-Path $scripts 'fetch-ffmpeg.ps1') -OutDir $ffmpeg
& (Join-Path $scripts 'stage-bundle.ps1') -WebDist (Join-Path $repo 'apps\web\dist') -FfmpegDir $ffmpeg

Push-Location $desktop
try {
    if (-not (Test-Path 'node_modules')) { Invoke-Native 'npm ci' { npm ci } }
    $tauriArgs = @('--config', 'src-tauri/tauri.bundle.conf.json')
    if ($SigningKey) {
        $env:TAURI_SIGNING_PRIVATE_KEY = (Resolve-Path $SigningKey).Path
        $env:TAURI_SIGNING_PRIVATE_KEY_PASSWORD = $SigningPassword
        $tauriArgs += @('--config', 'src-tauri/tauri.updater.conf.json')
    }
    Remove-Item -Recurse -Force (Join-Path $tauri 'target\release\bundle') -ErrorAction SilentlyContinue
    Invoke-Native 'tauri build' { npx tauri build @tauriArgs }
}
finally { Pop-Location }

$installer = Join-Path $tauri "target\release\bundle\nsis\LevelUp_${version}_x64-setup.exe"
if (-not (Test-Path $installer)) { throw "installeur absent : $installer" }
$mb = [math]::Round((Get-Item $installer).Length / 1MB, 1)
Write-Host "Installeur : $installer ($mb Mo)"
