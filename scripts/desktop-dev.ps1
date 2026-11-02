<#
.SYNOPSIS
    Poste de developpement de la coquille desktop (Tauri) : compile le vrai serveur Go et le pose
    au nom que le bundler attend, puis (avec -Tauri) lance la coquille.

.DESCRIPTION
    La coquille lance le binaire serveur comme sidecar. Tauri le cherche sous
    apps/desktop/src-tauri/binaries/levelup-server-<triple de la cible>.exe et le copie a cote de
    l'executable sous le nom levelup-server.exe. Ce script compile apps/go-api/cmd/server (CGO,
    DuckDB lie : gcc de msys64) vers ce chemin.

    Avec -Tauri, il lance `npx tauri dev` sur la racine du depot (ressources) et un dossier de
    donnees DE DEVELOPPEMENT, jamais la racine de l'application installee : un test de la
    coquille ne touche pas les donnees de l'utilisateur.

.PARAMETER Tauri
    Lance `npx tauri dev` apres la compilation du serveur.

.PARAMETER BuildWeb
    Compile d'abord le front (`npm run build` dans apps/web) : le serveur sert apps/web/dist, que la
    coquille lui transmet (LEVELUP_WEB_DIST) quand ce dossier existe.

.PARAMETER DataDir
    Racine de donnees de la coquille en developpement (defaut : %LOCALAPPDATA%\LevelUp-dev).

.EXAMPLE
    powershell -File scripts/desktop-dev.ps1            # compile seulement le serveur
    powershell -File scripts/desktop-dev.ps1 -Tauri     # compile, puis lance la coquille
#>
[CmdletBinding()]
param(
    [switch]$Tauri,
    [switch]$BuildWeb,
    [string]$DataDir = ''
)

$ErrorActionPreference = 'Stop'
$repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$desktop = Join-Path $repo 'apps\desktop'
$binaries = Join-Path $desktop 'src-tauri\binaries'

# Triple de la cible : celui que tauri-build met en suffixe du sidecar.
$triple = ((& rustc -vV) | Where-Object { $_ -like 'host:*' }) -replace '^host:\s*', ''
if (-not $triple) { throw 'rustc introuvable : le triple de la cible ne peut pas etre lu.' }

# DuckDB est lie par CGO : gcc de msys64 (convention du poste, cf. CLAUDE.md).
$env:CGO_ENABLED = '1'
$gcc = 'C:\msys64\ucrt64\bin'
if ((Test-Path $gcc) -and ($env:PATH -notlike "*$gcc*")) { $env:PATH = "$gcc;$env:PATH" }

New-Item -ItemType Directory -Force -Path $binaries | Out-Null
$server = Join-Path $binaries "levelup-server-$triple.exe"
Write-Host "Compilation du serveur vers $server"
Push-Location (Join-Path $repo 'apps\go-api')
try {
    & go build -ldflags '-X main.version=dev' -o $server ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "go build a echoue (code $LASTEXITCODE)" }
}
finally { Pop-Location }

if ($BuildWeb) {
    Push-Location (Join-Path $repo 'apps/web')
    try {
        & npm run build
        if ($LASTEXITCODE -ne 0) { throw "npm run build a echoue (code $LASTEXITCODE)" }
    }
    finally { Pop-Location }
}

if (-not $Tauri) { return }

if (-not $DataDir) { $DataDir = Join-Path $env:LOCALAPPDATA 'LevelUp-dev' }
$env:LEVELUP_RESOURCES_DIR = $repo
$env:LEVELUP_DATA_DIR = $DataDir
Write-Host "Ressources : $repo"
Write-Host "Donnees    : $DataDir"
$webDist = Join-Path $repo 'apps/web/dist'
if (Test-Path (Join-Path $webDist 'index.html')) {
    $env:LEVELUP_WEB_DIST = $webDist
    Write-Host "Front      : $webDist"
}
else {
    Write-Host 'Front      : absent (utiliser -BuildWeb) : le serveur ne servira pas la page'
}

Push-Location $desktop
try {
    if (-not (Test-Path 'node_modules')) { & npm install; if ($LASTEXITCODE -ne 0) { throw 'npm install a echoue' } }
    & npx tauri dev
    if ($LASTEXITCODE -ne 0) { throw "tauri dev a echoue (code $LASTEXITCODE)" }
}
finally { Pop-Location }
