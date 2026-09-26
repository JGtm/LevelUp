# run_replay_build.ps1 -- UN film par processus, en AVANT-PLAN, sous plafond memoire surveille
# (D17), adapte du gabarit lotBP/run_one.ps1 pour cmd/replay-build --facts (lot C-ter, fusion).
#
# Le pic est releve par echantillonnage (PeakWorkingSet64 du processus, rafraichi toutes les
# 250 ms) et le processus est TUE au-dela du plafond memoire OU du plafond de duree -- le rapport
# le dit alors, il ne l'efface pas.
#
# FICHIER STRICTEMENT ASCII : Windows PowerShell 5.1 lit un .ps1 sans BOM en codepage ANSI, et un
# seul caractere accentue casse l'analyse syntaxique du script entier.
#
# Usage :
#   .\run_replay_build.ps1 -MatchID 7344d24f -MapName Vagabond -Facts <chemin faits.json>
param(
  [Parameter(Mandatory = $true)][string]$MatchID,
  [Parameter(Mandatory = $true)][string]$MapName,
  [string]$Facts = "",
  [int]$CapMB = 3072,
  [int]$CapSec = 900,
  [string]$Out = "C:\Users\Guillaume\Projects\LevelUp-wt-cter-fusion\.ai\V7.5\replay2d\registre_film\lotCter"
)

$wt = "C:\Users\Guillaume\Projects\LevelUp-wt-cter-fusion"
$api = Join-Path $wt "apps\go-api"
$env:GOCACHE = Join-Path $wt ".gocache"
$env:CGO_ENABLED = "0"
$env:LEVELUP_REPO_ROOT = $wt
$filmDir = "C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/$MatchID"

# Start-Process joint les elements de -ArgumentList par un simple espace, SANS les
# re-quoter : un nom de carte a espace (« Solitude - Ranked ») doit donc porter ses
# propres guillemets dans l'element du tableau, sans quoi il se scinde en plusieurs
# arguments positionnels (constate sur 0a247154 : match=- filmDir=Ranked).
function QuoteArg($s) { if ($s -match '[\s]') { return '"' + $s + '"' } else { return $s } }
$argList = @("run", "./cmd/replay-build", "--map", (QuoteArg $MapName))
if ($Facts -ne "") { $argList += @("--facts", (QuoteArg $Facts)) }
$argList += @((QuoteArg $MatchID), (QuoteArg $filmDir))

$log = Join-Path $Out "cuisson_$MatchID.log"
$t0 = Get-Date
$p = Start-Process -FilePath "go" -ArgumentList $argList -WorkingDirectory $api -NoNewWindow `
  -PassThru -RedirectStandardOutput $log -RedirectStandardError "$log.err"

$peak = 0
$killed = $false
$reason = ""
while (-not $p.HasExited) {
  Start-Sleep -Milliseconds 250
  try { $p.Refresh(); if ($p.PeakWorkingSet64 -gt $peak) { $peak = $p.PeakWorkingSet64 } } catch {}
  $elapsed = ((Get-Date) - $t0).TotalSeconds
  if ($peak -gt ($CapMB * 1MB)) {
    try { $p.Kill() } catch {}
    $killed = $true
    $reason = "memoire"
    break
  }
  if ($elapsed -gt $CapSec) {
    try { $p.Kill() } catch {}
    $killed = $true
    $reason = "duree"
    break
  }
}
try { $p.Refresh(); if ($p.PeakWorkingSet64 -gt $peak) { $peak = $p.PeakWorkingSet64 } } catch {}
$mb = [math]::Round($peak / 1MB)
$sec = [math]::Round(((Get-Date) - $t0).TotalSeconds, 1)
$code = $p.ExitCode
if ($killed) { Write-Output ("PLAFOND DEPASSE (" + $reason + ") : " + $mb + " Mo / " + $sec + " s -- processus tue") }
Write-Output ("COUT " + $MatchID + " : " + $sec + " s - pic " + $mb + " Mo - EXIT_REPLAYBUILD_" + $MatchID + "=" + $code)
$tab = [char]9
$line = $MatchID + $tab + $MapName + $tab + $sec + $tab + $mb + $tab + $code
Add-Content -Path (Join-Path $Out "cout_machine.tsv") -Value $line -Encoding utf8
