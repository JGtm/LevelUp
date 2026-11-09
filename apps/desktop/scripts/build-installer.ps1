<#
.SYNOPSIS
    Construit en local les installeurs NSIS de LevelUp et le paquet de ressources : les memes etapes que le job
    `desktop-build` de .github/workflows/release.yml, sans secret.

.DESCRIPTION
    Deux unites sont livrees a part (ADR 0045, amendement L24) : le MOTEUR (coquille, serveur, front compile,
    config) et le PAQUET DE RESSOURCES (images, sons, referentiel, ffmpeg). Trois fichiers en sortie :

      LevelUp_<version>_x64-setup.exe          installeur COMPLET : moteur + paquet, premiere installation
      LevelUp_<version>_x64-engine-setup.exe   installeur MOTEUR SEUL : celui de l'updater (latest.json)
      levelup-resources-<empreinte>.zip        le paquet, que l'application telecharge quand son empreinte change

    Etapes :
    1. front compile (apps/web, sauf -SkipWeb) ;
    2. ffmpeg et ffprobe LGPL (fetch-ffmpeg.ps1 : archive epinglee, somme de controle verifiee) ;
    3. moteur et paquet assembles (stage-bundle.ps1) : l'empreinte du paquet en sort ;
    4. serveur Go (CGO, DuckDB lie) pose sous le nom du sidecar, version ET empreinte du paquet dans -ldflags, sans
       symboles (-s -w : traces de panique intactes) ;
    5. `npx tauri build` deux fois : tauri.bundle.conf.json (moteur + paquet) puis tauri.engine.conf.json (moteur seul).

    Sans -SigningKey, aucune cle n'est necessaire. Avec -SigningKey (chemin du fichier de cle privee de l'updater),
    l'installeur moteur seul et le paquet sont aussi signes pour l'updater (fichiers .sig a cote) ; -SigningPassword
    donne le mot de passe de la cle, s'il en a un.

.PARAMETER SkipWeb
    Reutilise apps/web/dist tel quel.

.PARAMETER SigningKey
    Cle privee de l'updater : produit aussi les fichiers .sig.

.PARAMETER SigningPassword
    Mot de passe de la cle (defaut : aucun).

.PARAMETER OutDir
    Dossier des trois sorties (defaut : apps/desktop/src-tauri/target/release/installers).

.EXAMPLE
    powershell -File apps/desktop/scripts/build-installer.ps1
#>
[CmdletBinding()]
param(
    [switch]$SkipWeb,
    [string]$SigningKey = '',
    [string]$SigningPassword = '',
    [string]$OutDir = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$desktop = Join-Path $repo 'apps\desktop'
$tauri = Join-Path $desktop 'src-tauri'
$scripts = Join-Path $desktop 'scripts'
if (-not $OutDir) { $OutDir = Join-Path $tauri 'target\release\installers' }

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

$ffmpeg = Join-Path $env:TEMP 'levelup-ffmpeg-out'
& (Join-Path $scripts 'fetch-ffmpeg.ps1') -OutDir $ffmpeg
if (Test-Path $OutDir) { Remove-Item -Recurse -Force $OutDir }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
& (Join-Path $scripts 'stage-bundle.ps1') -WebDist (Join-Path $repo 'apps\web\dist') -FfmpegDir $ffmpeg -ArchiveDir $OutDir -Version $version
$fingerprint = (Get-Content -Raw (Join-Path $tauri 'pack-staging.fingerprint')).Trim()
if ($fingerprint -notmatch '^[0-9a-f]{64}$') { throw "empreinte du paquet illisible : $fingerprint" }

$env:CGO_ENABLED = '1'
$gcc = 'C:\msys64\ucrt64\bin'
if ((Test-Path $gcc) -and ($env:PATH -notlike "*$gcc*")) { $env:PATH = "$gcc;$env:PATH" }
$binaries = Join-Path $tauri 'binaries'
New-Item -ItemType Directory -Force -Path $binaries | Out-Null
$server = Join-Path $binaries "levelup-server-$triple.exe"
Push-Location (Join-Path $repo 'apps\go-api')
try { Invoke-Native 'go build du serveur' { go build -ldflags "-s -w -X main.version=$version -X main.resourcesFingerprint=$fingerprint" -o $server ./cmd/server } }
finally { Pop-Location }

function Build-Installer {
    param([string]$Name, [string[]]$Configs, [string]$Target)
    $tauriArgs = @()
    foreach ($config in $Configs) { $tauriArgs += @('--config', "src-tauri/$config") }
    Remove-Item -Recurse -Force (Join-Path $tauri 'target\release\bundle') -ErrorAction SilentlyContinue
    Invoke-Native "tauri build ($Name)" { npx tauri build @tauriArgs }
    $built = Join-Path $tauri "target\release\bundle\nsis\LevelUp_${version}_x64-setup.exe"
    if (-not (Test-Path $built)) { throw "installeur absent : $built" }
    Move-Item -Force -Path $built -Destination (Join-Path $OutDir $Target)
    if (Test-Path "$built.sig") { Move-Item -Force -Path "$built.sig" -Destination (Join-Path $OutDir "$Target.sig") }
}

Push-Location $desktop
try {
    if (-not (Test-Path 'node_modules')) { Invoke-Native 'npm ci' { npm ci } }
    Build-Installer 'complet' @('tauri.bundle.conf.json') "LevelUp_${version}_x64-setup.exe"
    $engineConfigs = @('tauri.engine.conf.json')
    if ($SigningKey) {
        # UNE seule voie, celle de la CI (release.yml) : le CONTENU de la cle dans TAURI_SIGNING_PRIVATE_KEY, lu par
        # `tauri build` ET par `tauri signer sign`. Passer aussi `-f <chemin>` a `signer sign` est refuse (la variable
        # d'environnement est deja `--private-key` : « --private-key-path cannot be used with --private-key »).
        $env:TAURI_SIGNING_PRIVATE_KEY = (Get-Content -Raw -Path (Resolve-Path $SigningKey).Path)
        $env:TAURI_SIGNING_PRIVATE_KEY_PASSWORD = $SigningPassword
        $engineConfigs += 'tauri.updater.conf.json'
    }
    Build-Installer 'moteur seul' $engineConfigs "LevelUp_${version}_x64-engine-setup.exe"
    if ($SigningKey) {
        $pack = Join-Path $OutDir "levelup-resources-$fingerprint.zip"
        Invoke-Native 'signature du paquet' { npx tauri signer sign $pack }
    }
}
finally { Pop-Location }

foreach ($file in Get-ChildItem -File $OutDir | Where-Object { $_.Extension -ne '.sig' }) {
    Write-Host ("{0} : {1:N1} Mo ({2} octets)" -f $file.Name, ($file.Length / 1MB), $file.Length)
}
Write-Host "Empreinte du paquet : $fingerprint ; sorties dans $OutDir"
