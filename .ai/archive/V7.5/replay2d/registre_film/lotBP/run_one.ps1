# run_one.ps1 -- UN film par processus, en AVANT-PLAN, sous plafond memoire surveille (D17).
#
# La machine de l'utilisateur a deja plante deux fois sur un balayage de corpus (memoire
# reference_statrecords_corpus_sweep_ram_bomb) : le plafond n'est pas une precaution de style,
# c'est la condition pour que l'instrument tourne du tout. Le pic est releve par
# echantillonnage (PeakWorkingSet64 du processus, rafraichi toutes les 250 ms) et le processus
# est TUE au-dela du plafond -- le rapport le dit alors, il ne l'efface pas.
#
# FICHIER STRICTEMENT ASCII : Windows PowerShell 5.1 lit un .ps1 sans BOM en codepage ANSI,
# et un seul caractere accentue casse l'analyse syntaxique du script entier.
#
# Usage :
#   .\run_one.ps1 -Film 000d5950 -EnvVar GAME_FILM -Run '^TestGameEntitiesPhase0$'
param(
  [Parameter(Mandatory = $true)][string]$Film,
  [string]$EnvVar = "GAME_FILM",
  [string]$Run = "^TestGameEntitiesPhase0$",
  [string]$Pkg = "./internal/analysis/filmdec/",
  [int]$CapMB = 3072,
  [string]$Out = "C:\Users\Guillaume\Projects\LevelUp-wt-joueur-moteur\.ai\V7.5\replay2d\registre_film\lotBP"
)

$wt = "C:\Users\Guillaume\Projects\LevelUp-wt-joueur-moteur"
$api = Join-Path $wt "apps\go-api"
$env:GOCACHE = Join-Path $wt ".gocache"
$env:CGO_ENABLED = "0"
$env:GAME_OUT = $Out
Set-Item -Path "env:$EnvVar" -Value "C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/$Film"

$log = Join-Path $Out "$Film.$EnvVar.log"
$t0 = Get-Date
$p = Start-Process -FilePath "go" `
  -ArgumentList @("test", $Pkg, "-run", $Run, "-count=1", "-timeout", "60m", "-v") `
  -WorkingDirectory $api -NoNewWindow -PassThru -RedirectStandardOutput $log `
  -RedirectStandardError "$log.err"

$peak = 0
$killed = $false
while (-not $p.HasExited) {
  Start-Sleep -Milliseconds 250
  try { $p.Refresh(); if ($p.PeakWorkingSet64 -gt $peak) { $peak = $p.PeakWorkingSet64 } } catch {}
  if ($peak -gt ($CapMB * 1MB)) {
    try { $p.Kill() } catch {}
    $killed = $true
    break
  }
}
try { $p.Refresh(); if ($p.PeakWorkingSet64 -gt $peak) { $peak = $p.PeakWorkingSet64 } } catch {}
$mb = [math]::Round($peak / 1MB)
$sec = [math]::Round(((Get-Date) - $t0).TotalSeconds, 1)
$code = $p.ExitCode
if ($killed) { Write-Output ("PLAFOND DEPASSE : " + $mb + " Mo > " + $CapMB + " Mo -- processus tue") }
Write-Output ("COUT " + $Film + " : " + $sec + " s - pic " + $mb + " Mo - EXIT_" + $EnvVar + "_" + $Film + "=" + $code)
$tab = [char]9
$line = $Film + $tab + $EnvVar + $tab + $sec + $tab + $mb + $tab + $code
Add-Content -Path (Join-Path $Out "cout_machine.tsv") -Value $line -Encoding utf8
