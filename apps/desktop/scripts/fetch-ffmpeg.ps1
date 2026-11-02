<#
.SYNOPSIS
    Telecharge, verifie et extrait ffmpeg et ffprobe (build LGPL) a livrer a cote du serveur.

.DESCRIPTION
    ADR 0045 D-17 : le serveur cherche ffmpeg et ffprobe d'abord dans le dossier de son
    executable (media.ResolveFFmpeg). L'archive est EPINGLEE par URL ET par somme de controle
    SHA-256 ; une somme differente arrete le script avant toute extraction.

    Source : BtbN/FFmpeg-Builds, release datee autobuild-2026-10-31-12-59 (jamais `latest`),
    archive ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip. Build LGPL v3 "shared"
    (sans libx264 ni libx265, qui sont GPL) : ffmpeg.exe, ffprobe.exe et ses DLL. La somme
    ci-dessous est celle du fichier `checksums.sha256` publie avec cette release.

    Sources correspondantes (obligation LGPL) : https://ffmpeg.org/releases/ffmpeg-8.1.3.tar.xz
    (etiquette n8.1.3) et les scripts de construction https://github.com/BtbN/FFmpeg-Builds
    a la release ci-dessus. Le lien est repris dans THIRD_PARTY_NOTICES.md.

    Pour changer de version : nouvelle release datee, nouvelle somme lue dans son
    `checksums.sha256`, les trois constantes ci-dessous et THIRD_PARTY_NOTICES.md ensemble.

.PARAMETER OutDir
    Dossier de sortie (cree, vide au depart) : ffmpeg.exe, ffprobe.exe, *.dll, LICENSE.LGPL.

.PARAMETER CacheDir
    Dossier ou l'archive telechargee est conservee (defaut : %TEMP%\levelup-ffmpeg-cache).

.EXAMPLE
    powershell -File apps/desktop/scripts/fetch-ffmpeg.ps1 -OutDir $env:TEMP\ffmpeg-out
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$OutDir,
    [string]$CacheDir = ''
)

$ErrorActionPreference = 'Stop'

$Release = 'autobuild-2026-10-31-12-59'
$Asset = 'ffmpeg-n8.1.3-27-g8dca1586f4-win64-lgpl-shared-8.1.zip'
$Sha256 = '5cef7bc48e513f3cee7c881e8b2999d835541edb1846c3e89443a67f8f8556ab'
$Url = "https://github.com/BtbN/FFmpeg-Builds/releases/download/$Release/$Asset"

if (-not $CacheDir) { $CacheDir = Join-Path $env:TEMP 'levelup-ffmpeg-cache' }
New-Item -ItemType Directory -Force -Path $CacheDir | Out-Null
$archive = Join-Path $CacheDir $Asset

function Test-Archive {
    param([string]$Path)
    if (-not (Test-Path $Path)) { return $false }
    return ((Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant() -eq $Sha256)
}

if (-not (Test-Archive $archive)) {
    Write-Host "Telechargement de $Url"
    $ProgressPreference = 'SilentlyContinue'
    Invoke-WebRequest -Uri $Url -OutFile $archive -UseBasicParsing
}
$actual = (Get-FileHash -Algorithm SHA256 -Path $archive).Hash.ToLowerInvariant()
if ($actual -ne $Sha256) {
    Remove-Item -Force $archive
    throw "Somme de controle SHA-256 differente pour $Asset : attendue $Sha256, obtenue $actual. Archive supprimee, rien n'est extrait."
}
Write-Host "SHA-256 verifiee : $actual"

$extract = Join-Path $CacheDir ("extract-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $extract | Out-Null
try {
    Expand-Archive -Path $archive -DestinationPath $extract -Force
    $root = Get-ChildItem -Path $extract -Directory | Select-Object -First 1
    if (-not $root) { throw "Archive inattendue : aucun dossier racine dans $Asset" }

    if (Test-Path $OutDir) { Remove-Item -Recurse -Force $OutDir }
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $bin = Join-Path $root.FullName 'bin'
    # ffplay (lecteur) est laisse : le serveur ne s'en sert pas, 19 Mo de moins par mise a jour.
    Get-ChildItem -Path $bin -File | Where-Object { $_.Name -ne 'ffplay.exe' } | Copy-Item -Destination $OutDir
    Copy-Item -Path (Join-Path $root.FullName 'LICENSE.txt') -Destination (Join-Path $OutDir 'LICENSE.LGPL')
    foreach ($required in 'ffmpeg.exe', 'ffprobe.exe', 'LICENSE.LGPL') {
        if (-not (Test-Path (Join-Path $OutDir $required))) { throw "$required absent de la sortie" }
    }
}
finally {
    Remove-Item -Recurse -Force $extract -ErrorAction SilentlyContinue
}
Write-Host "ffmpeg/ffprobe LGPL prets dans $OutDir"
