<#
.SYNOPSIS
    Test de bout en bout de l'installeur NSIS (ADR 0045 D-21, plan L17.3) : installation silencieuse,
    deux lancements, fermeture par WM_CLOSE, desinstallation qui conserve les donnees.

.DESCRIPTION
    Sequence, chaque etape est une assertion (la premiere qui echoue arrete le script, code 1) :

      1. installation silencieuse (/S) sous %LOCALAPPDATA%\Programs\LevelUp ; programme, serveur,
         ffmpeg, front et mentions tierces presents ; le dossier du programme n'est PAS la racine de
         donnees par defaut (%LOCALAPPDATA%\LevelUp) ;
      2. lancement avec LEVELUP_DATA_DIR sur une racine JETABLE (les donnees de l'utilisateur ne
         sont jamais touchees), attente de <DataRoot>\desktop\port, GET /health = 200 (le routeur
         est ouvert : le serveur a annonce `ready`) ;
      3. fermeture par WM_CLOSE (CloseMainWindow), attente de la sortie de l'application ; le
         journal porte une ligne `arret: termine` de plus avec clean=true, et AUCUN fichier *.wal
         ne reste sous la racine de donnees (hors backups) : l'observable d'un arret propre
         (contrat coquille/serveur, plan section 6, point 5) ;
      4. relancement (meme port memorise, /health 200) puis nouvelle fermeture propre ;
      5. desinstallation silencieuse : le programme disparait ; la racine de donnees par defaut
         (%LOCALAPPDATA%\LevelUp), ou le script a pose une sentinelle avant l'installation, existe
         encore avec sa sentinelle (le desinstalleur ne la supprime jamais), comme la racine jetable ;
         le desinstalleur a lance `levelup-server.exe -mode uninstall-cleanup` (crochet NSIS, ADR 0045
         D-28) : le serveur ecrit son bilan dans le fichier temoin LEVELUP_UNINSTALL_WITNESS (racine
         jetable, dossier du programme, sortie 0). Le test ne pose aucune entree de demarrage ni tache
         planifiee (trop risque sur le poste) : le bilan attendu est "absent".

    Verifications prealables, AVANT tout lancement et sans aucun nettoyage : le script refuse (code 1)
    si LevelUp est deja installe (dossier du programme, cle de registre NSIS du produit ou de
    desinstallation) ou si levelup-desktop.exe / levelup-server.exe tourne. Il ne tue que les
    processus qu'il a lui-meme lances et ne desinstalle que ce qu'il a lui-meme installe. Il rend la
    main avec le code 0 ou 1 ; en cas d'echec il copie les journaux sous -ArtifactsDir.

.PARAMETER Installer
    L'installeur NSIS (LevelUp_X.Y.Z_x64-setup.exe).

.PARAMETER DataDir
    Racine de donnees jetable (defaut : un dossier neuf sous %TEMP%).

.PARAMETER ArtifactsDir
    Ou copier les journaux en cas d'echec (defaut : %TEMP%\levelup-e2e-artifacts).

.PARAMETER TimeoutSec
    Delai de chaque attente (port, /health), en secondes. Le premier demarrage migre des bases
    vides : defaut 240.

.EXAMPLE
    powershell -File apps/desktop/scripts/e2e-install.ps1 -Installer .\LevelUp_8.1.0_x64-setup.exe
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Installer,
    [string]$DataDir = '',
    [string]$ArtifactsDir = '',
    [int]$TimeoutSec = 240
)

$ErrorActionPreference = 'Stop'
$InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\LevelUp'
$DefaultDataRoot = Join-Path $env:LOCALAPPDATA 'LevelUp'
$Sentinel = Join-Path $DefaultDataRoot 'e2e-sentinelle.txt'
# Editeur de la cle de registre NSIS (HKCU:\Software\<editeur>\LevelUp) : deduit par Tauri de `identifier`
# (info.lvelup.desktop) dans tauri.conf.json.
$Manufacturer = 'lvelup'
$MainExe = Join-Path $InstallDir 'levelup-desktop.exe'
# Ligne de journal du serveur a l'arret, construite sans caracteres non ASCII dans ce fichier.
$StopMessage = 'arr' + [char]0xEA + 't: termin' + [char]0xE9
$CloseBudgetSec = 90
# Fichier temoin du nettoyage de desinstallation (LEVELUP_UNINSTALL_WITNESS), nom aleatoire sous %TEMP%.
$Witness = Join-Path $env:TEMP ('levelup-e2e-uninstall-' + [Guid]::NewGuid().ToString('N') + '.txt')

if (-not $DataDir) { $DataDir = Join-Path $env:TEMP ('levelup-e2e-data-' + [Guid]::NewGuid().ToString('N')) }
if (-not $ArtifactsDir) { $ArtifactsDir = Join-Path $env:TEMP 'levelup-e2e-artifacts' }
$Installer = (Resolve-Path $Installer).Path

function Step([string]$text) { Write-Host ('[e2e] ' + $text) }
function Fail([string]$text) { throw ('ECHEC : ' + $text) }

function Wait-Until {
    param([scriptblock]$Condition, [string]$What, [int]$Seconds = $TimeoutSec)
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        $value = & $Condition
        if ($value) { return $value }
        Start-Sleep -Milliseconds 250
    }
    Fail "delai depasse ($Seconds s) : $What"
}

function Get-Health([int]$Port) {
    try {
        $response = Invoke-WebRequest -Uri "http://127.0.0.1:$Port/health" -UseBasicParsing -TimeoutSec 5
        return [int]$response.StatusCode
    }
    catch {
        return 0
    }
}

function Get-StopLines {
    $logs = Join-Path $DataDir 'logs'
    if (-not (Test-Path $logs)) { return @() }
    # Seul le journal du serveur porte la ligne ; desktop.log relaie sa sortie d'erreur et la
    # dupliquerait : le compte se fait sur les journaux JSON, hors desktop.log.
    $lines = foreach ($file in Get-ChildItem -Path $logs -Filter '*.log' -File -Recurse | Where-Object { $_.Name -ne 'desktop.log' }) {
        Get-Content -Path $file.FullName -Encoding UTF8 | Where-Object { $_ -like "*$StopMessage*" }
    }
    return @($lines)
}

function Assert-NoWal {
    $backups = Join-Path $DataDir 'data\backups'
    $wal = Get-ChildItem -Path $DataDir -Recurse -File -Filter '*.wal' -ErrorAction SilentlyContinue |
        Where-Object { -not $_.FullName.StartsWith($backups, [StringComparison]::OrdinalIgnoreCase) }
    if ($wal) { Fail ("fichier WAL residuel apres l'arret : " + (($wal | ForEach-Object FullName) -join ', ')) }
}

function Invoke-Run([int]$Number) {
    $portFile = Join-Path $DataDir 'desktop\port'
    Step "lancement $Number"
    $process = Start-Process -FilePath $MainExe -PassThru
    $script:Launched += $process
    try {
        $port = Wait-Until { if (Test-Path $portFile) { $text = (Get-Content $portFile -Raw).Trim(); if ($text -match '^\d+$') { [int]$text } } } "le fichier $portFile"
        Step "port $port"
        Wait-Until { (Get-Health $port) -eq 200 } "GET /health = 200 sur le port $port" | Out-Null
        Step '/health = 200'
        if ($Number -gt 1 -and $port -ne $script:FirstPort) { Fail "le port n'est pas stable : $script:FirstPort puis $port" }
        if ($Number -eq 1) { $script:FirstPort = $port }

        Wait-Until { $process.Refresh(); $process.MainWindowHandle -ne [IntPtr]::Zero } 'la fenetre principale' 60 | Out-Null
        Step 'fermeture par WM_CLOSE'
        if (-not $process.CloseMainWindow()) { Fail 'CloseMainWindow a refuse le message' }
        if (-not $process.WaitForExit($CloseBudgetSec * 1000)) { Fail "l'application ne s'est pas fermee dans les $CloseBudgetSec s apres WM_CLOSE" }
    }
    catch {
        if (-not $process.HasExited) { $process.Kill() }
        throw
    }
    Step ('application sortie, code ' + $process.ExitCode)

    $stops = @(Get-StopLines)
    if ($stops.Count -ne $Number) { Fail ("$($stops.Count) ligne(s) '$StopMessage' dans les journaux, attendu $Number") }
    $last = $stops[-1]
    if ($last -notmatch '"clean"\s*:\s*true') { Fail "arret non propre : $last" }
    Assert-NoWal
    Step "arret propre ($StopMessage, clean=true), aucun *.wal"
}

function Copy-Failure {
    New-Item -ItemType Directory -Force -Path $ArtifactsDir | Out-Null
    $logs = Join-Path $DataDir 'logs'
    if (Test-Path $logs) { Copy-Item -Path $logs -Destination (Join-Path $ArtifactsDir 'logs') -Recurse -Force }
    Write-Host "[e2e] journaux copies sous $ArtifactsDir"
}

# Les emplacements ou NSIS note une installation : la cle du produit (valeur par defaut = dossier,
# relue par RestorePreviousInstallLocation) et les cles de desinstallation (HKCU et HKLM).
function Get-Registrations {
    $sources = @(
        @{ Path = "HKCU:\Software\$Manufacturer\LevelUp"; Name = ''; Uninstall = $false },
        @{ Path = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\LevelUp'; Name = 'InstallLocation'; Uninstall = $true },
        @{ Path = 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\LevelUp'; Name = 'InstallLocation'; Uninstall = $true }
    )
    foreach ($source in $sources) {
        if (-not (Test-Path $source.Path)) { continue }
        $value = [string](Get-Item $source.Path).GetValue($source.Name)
        [pscustomobject]@{ Path = $source.Path; Location = $value.Trim().Trim('"'); Uninstall = $source.Uninstall }
    }
}

# Verifications prealables : tout ce qui ressemble a une installation ou a un LevelUp en cours refuse
# le test, SANS aucun nettoyage (ce script ne desinstalle et ne tue que ce qu'il a lui-meme cree).
function Test-Preconditions {
    $problems = @()
    if (Test-Path $InstallDir) { $problems += "$InstallDir existe deja" }
    foreach ($registration in @(Get-Registrations)) {
        $where = $registration.Location
        if ($registration.Uninstall) {
            $problems += "installation enregistree sous $($registration.Path) ($where)"
        }
        elseif ($where -and (Test-Path $where)) {
            $problems += "installation enregistree sous $($registration.Path) : $where existe"
        }
        elseif ($where -and $where -ne $InstallDir) {
            # Cle perimee vers un autre dossier : l'installeur la relirait et installerait ailleurs.
            $problems += "cle $($registration.Path) vers $where : l'installeur s'y installerait"
        }
    }
    $running = @(Get-Process -Name 'levelup-desktop', 'levelup-server' -ErrorAction SilentlyContinue)
    if ($running.Count -gt 0) { $problems += 'LevelUp est deja lance (pid ' + (($running | ForEach-Object Id) -join ', ') + ')' }
    return $problems
}

# Sentinelle posee dans la racine de donnees PAR DEFAUT : le desinstalleur ne doit jamais la supprimer.
function Remove-Sentinel {
    if (-not $script:SentinelPlaced) { return }
    Remove-Item -Path $Sentinel -Force -ErrorAction SilentlyContinue
    $script:SentinelPlaced = $false
    if ($script:CreatedDefaultRoot -and (Test-Path $DefaultDataRoot) -and -not (Get-ChildItem -Path $DefaultDataRoot -Force)) {
        Remove-Item -Path $DefaultDataRoot -Force -ErrorAction SilentlyContinue
    }
}

$script:FirstPort = 0
$script:Launched = @()
$script:SentinelPlaced = $false
$script:CreatedDefaultRoot = $false
$problems = @(Test-Preconditions)
if ($problems.Count -gt 0) {
    foreach ($problem in $problems) { Write-Host "[e2e] REFUS : $problem" }
    Write-Host "[e2e] le test ne touche pas a une installation reelle ni a un LevelUp en cours : rien n'a ete lance, installe ni retire"
    exit 1
}

$installedByScript = $false
$uninstalledByScript = $false
try {
    # Environnement du test : racine de donnees jetable, aucun reglage herite de la session.
    foreach ($name in (Get-ChildItem Env: | Where-Object { $_.Name -like 'LEVELUP_*' -or $_.Name -like 'SPNKR_*' }).Name) {
        Remove-Item "Env:$name"
    }
    New-Item -ItemType Directory -Force -Path $DataDir | Out-Null
    $env:LEVELUP_DATA_DIR = $DataDir

    if (-not (Test-Path $DefaultDataRoot)) {
        New-Item -ItemType Directory -Force -Path $DefaultDataRoot | Out-Null
        $script:CreatedDefaultRoot = $true
    }
    Set-Content -Path $Sentinel -Value 'sentinelle e2e : le desinstalleur ne doit jamais la supprimer' -Encoding ASCII
    $script:SentinelPlaced = $true

    Step "installation silencieuse de $Installer"
    # Des ici, ce qui se trouve sous $InstallDir est du fait de ce script (les verifications ont garanti qu'il n'y avait rien).
    $installedByScript = $true
    $setup = Start-Process -FilePath $Installer -ArgumentList '/S' -Wait -PassThru
    if ($setup.ExitCode -ne 0) { Fail "l'installeur a rendu le code $($setup.ExitCode)" }
    foreach ($relative in 'levelup-desktop.exe', 'levelup-server.exe', 'ffmpeg.exe', 'ffprobe.exe', 'LICENSE.LGPL',
        'THIRD_PARTY_NOTICES.md', 'web\index.html', 'config\locales.toml', 'uninstall.exe') {
        if (-not (Test-Path (Join-Path $InstallDir $relative))) { Fail "$relative absent de $InstallDir" }
    }
    if (Test-Path (Join-Path $DefaultDataRoot 'levelup-desktop.exe')) { Fail "le programme est installe dans la racine de donnees par defaut $DefaultDataRoot" }
    Step "installe sous $InstallDir"

    Invoke-Run 1
    Invoke-Run 2

    Step 'desinstallation silencieuse'
    $marker = Join-Path $DataDir 'desktop\port'
    $env:LEVELUP_UNINSTALL_WITNESS = $Witness
    $uninstaller = Start-Process -FilePath (Join-Path $InstallDir 'uninstall.exe') -ArgumentList '/S' -Wait -PassThru
    if ($uninstaller.ExitCode -ne 0) { Fail "le desinstalleur a rendu le code $($uninstaller.ExitCode)" }
    $uninstalledByScript = $true
    # Le desinstalleur NSIS se recopie et rend la main avant la fin : on attend la disparition du programme.
    Wait-Until { -not (Test-Path $MainExe) } "la disparition de $MainExe" 120 | Out-Null
    Wait-Until { -not (Test-Path $InstallDir) } "la disparition de $InstallDir" 60 | Out-Null
    # Le crochet de desinstallation a lance le nettoyage du serveur livre : son temoin porte le bilan (les
    # chemins y sont ecrits entre guillemets Go, antislash doubles).
    if (-not (Test-Path $Witness)) { Fail "le desinstalleur n'a pas lance -mode uninstall-cleanup : aucun fichier temoin $Witness" }
    $report = (Get-Content -Path $Witness -Raw -Encoding UTF8).Replace('\\', '\')
    if ($report -notmatch '^uninstall-cleanup:') { Fail "temoin inattendu : $report" }
    if ($report -notmatch 'demarrage=absent' -or $report -notmatch 'ok=true') { Fail "le nettoyage n'a pas rendu le bilan attendu (demarrage=absent, ok=true) : $report" }
    if (-not $report.Contains('data_root="' + $DataDir + '"')) { Fail "le nettoyage a vise une autre racine de donnees que $DataDir : $report" }
    if (-not $report.Contains('install_dir="' + $InstallDir + '"')) { Fail "le nettoyage a vise un autre dossier de programme que $InstallDir : $report" }
    Step ('desinstalleur : nettoyage lance (-mode uninstall-cleanup) : ' + $report.Trim())
    # L'invariant reel : le desinstalleur ne supprime JAMAIS la racine de donnees par defaut de l'utilisateur.
    if (-not (Test-Path $Sentinel)) { Fail "le desinstalleur a supprime la sentinelle de $DefaultDataRoot : la racine de donnees par defaut n'est pas conservee" }
    if (-not (Test-Path $marker)) { Fail "la racine de donnees $DataDir a perdu $marker a la desinstallation" }
    if (-not (Test-Path (Join-Path $DataDir 'logs'))) { Fail 'la racine de donnees a perdu ses journaux a la desinstallation' }
    Step 'programme retire, racine de donnees par defaut (sentinelle) et racine jetable conservees'

    Write-Host '[e2e] OK'
    $exitCode = 0
}
catch {
    Write-Host ('[e2e] ' + $_.Exception.Message)
    Copy-Failure
    $exitCode = 1
}
finally {
    # Seulement ce que ce script a cree : les processus qu'il a lances (arbre compris), jamais un LevelUp d'ailleurs.
    foreach ($launched in $script:Launched) {
        if (-not $launched.HasExited) {
            & taskkill.exe /PID $launched.Id /T /F | Out-Null
        }
    }
    # Un echec laisse l'installation du test en place : la retirer pour qu'un nouvel essai reparte propre.
    $leftover = Join-Path $InstallDir 'uninstall.exe'
    if ($installedByScript -and -not $uninstalledByScript -and (Test-Path $leftover)) {
        Start-Process -FilePath $leftover -ArgumentList '/S' -Wait | Out-Null
        Write-Host '[e2e] installation du test retiree apres l''echec'
    }
    Remove-Sentinel
    Remove-Item -Path $Witness -Force -ErrorAction SilentlyContinue
}
exit $exitCode
