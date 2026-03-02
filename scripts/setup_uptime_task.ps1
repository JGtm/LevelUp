# Enregistre (ou met à jour) la tâche planifiée Windows qui surveille
# la disponibilité du dashboard LevelUp via Tailscale Funnel.
#
# Le script monitor_uptime.py est exécuté toutes les minutes.
# En cas de changement d'état (online ↔ offline) il envoie une notification Discord.
#
# Prérequis :
#   - .venv configuré (python scripts/check_env.py)
#   - DISCORD_WEBHOOK_URL dans .env.local
#   - Tailscale installé et authentifié
#
# Usage (PowerShell en administrateur) :
#   .\scripts\setup_uptime_task.ps1
#
# Suppression de la tâche :
#   Unregister-ScheduledTask -TaskName "LevelUp-UptimeMonitor" -Confirm:$false

$ErrorActionPreference = "Stop"

$TaskName   = "LevelUp-UptimeMonitor"
$ScriptRoot = Split-Path -Parent $PSScriptRoot   # racine du repo (parent de scripts/)
$Python     = Join-Path $ScriptRoot ".venv\Scripts\python.exe"
$Script     = Join-Path $ScriptRoot "scripts\monitor_uptime.py"

# --- Vérifications préalables ---

if (-not (Test-Path $Python)) {
    Write-Error "Interpréteur introuvable : $Python`nActive d'abord l'environnement : .\scripts\setup_env.ps1"
    exit 1
}

if (-not (Test-Path $Script)) {
    Write-Error "Script introuvable : $Script"
    exit 1
}

# --- Définition de la tâche ---

$Action = New-ScheduledTaskAction `
    -Execute  $Python `
    -Argument $Script `
    -WorkingDirectory $ScriptRoot

# Déclencheur : toutes les 1 minute, indéfiniment, à partir de maintenant
$Trigger = New-ScheduledTaskTrigger `
    -Once `
    -At (Get-Date) `
    -RepetitionInterval (New-TimeSpan -Minutes 1)

$Settings = New-ScheduledTaskSettingsSet `
    -ExecutionTimeLimit (New-TimeSpan -Minutes 1) `
    -MultipleInstances IgnoreNew `
    -StartWhenAvailable

# --- Enregistrement (ou mise à jour si la tâche existe déjà) ---

$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existing) {
    Set-ScheduledTask `
        -TaskName $TaskName `
        -Action   $Action `
        -Trigger  $Trigger `
        -Settings $Settings | Out-Null
    Write-Host "Tâche mise à jour  : $TaskName"
} else {
    Register-ScheduledTask `
        -TaskName    $TaskName `
        -Action      $Action `
        -Trigger     $Trigger `
        -Settings    $Settings `
        -RunLevel    Highest `
        -Description "Surveille la disponibilité du dashboard LevelUp (Tailscale Funnel) et notifie Discord." | Out-Null
    Write-Host "Tâche enregistrée  : $TaskName"
}

Write-Host "Répertoire de travail : $ScriptRoot"
Write-Host "Python                : $Python"
Write-Host ""
Write-Host "Pour vérifier : Get-ScheduledTask -TaskName '$TaskName' | Select-Object State"
Write-Host "Pour supprimer : Unregister-ScheduledTask -TaskName '$TaskName' -Confirm:`$false"
