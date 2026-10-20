# scripts/disk-hygiene.ps1
#
# Hygiene du disque du poste de dev : rend la place que les caches de compilation Go dedies,
# les dossiers de travail des sessions Claude et les TMP courts des gates ne rendent jamais.
#
# Pourquoi : Go ne retire de son cache que les entrees inutilisees depuis 5 jours, et seulement
# quand une commande `go` reutilise CE cache. Un GOCACHE dedie a un lot (`go-build-<lot>`)
# abandonne a la cloture du lot ne retrecit donc jamais ; les racines de gate (`--keep-work`) et
# les copies du parc des scratchpads non plus. Mesure du 2026-10-08 : 308 Go a eux deux.
#
# SIMULATION PAR DEFAUT : sans -Apply, le script mesure et journalise ce qu'il supprimerait.
#
# Regles de securite (aucune n'est desactivable) :
#   1. Racines autorisees seulement : %LOCALAPPDATA%\go-build-* et golangci-* (hors le cache
#      par defaut golangci-lint), %USERPROFILE%\gocache-*, C:\*-gocache, les dossiers de session
#      sous %TEMP%\claude, et C:\t\*. Jamais data\, un worktree, .git ni le cache des modules.
#   2. Si un processus go, compile, link, *.test, golangci-lint, gate, replay-* ou levelup*
#      tourne, la passe globale ne supprime rien (rapport seul). -Lot n'est pas concerne : il
#      refuse a la place un cache qui a recu une ecriture dans les 10 dernieres minutes.
#   3. L'age d'un arbre est celui de son ecriture la plus recente (fichiers et dossiers), jamais
#      la date du dossier racine, qui ne bouge pas quand Go ecrit dans ses sous-dossiers.
#   4. Le parcours ne descend JAMAIS dans un point de reparation, et un arbre qui en contient un
#      (jonction, lien) n'est PAS supprime : il est signale. `Remove-Item -Recurse` de Windows
#      PowerShell 5.1 suit les jonctions ; un `git worktree remove` les a deja suivies et a vide
#      le cache des films (incident du 2026-09-16).
#   5. Les worktrees ne sont jamais supprimes : le script en fait seulement le rapport.
#   6. Chaque passe journalise l'espace libre avant et apres, et ce qui a ete supprime, dans
#      %LOCALAPPDATA%\levelup-disk-hygiene\disk-hygiene.log ; sous 60 Go libres, ALERTE.
#
# Seuils : cache dedie inactif depuis 48 h supprime ; cache partage vide par `go clean -cache`
# au-dela de 25 Go ; dossier de session supprime quand son transcript et son contenu n'ont pas
# bouge depuis 72 h ; dans une session active, sous-dossiers gate_work*, gocache*, golangci*,
# parc* supprimes apres 24 h d'inactivite ; C:\t\* apres 72 h.
#
# Usage :
#   powershell -File scripts/disk-hygiene.ps1                 # simulation, rapport complet
#   powershell -File scripts/disk-hygiene.ps1 -Apply          # passe reelle (tache planifiee)
#   powershell -File scripts/disk-hygiene.ps1 -Lot cg3-vuea -Apply
#                      # cloture d'un lot : supprime go-build-cg3-vuea et golangci-cg3-vuea
#
# Encodage : ASCII (Windows PowerShell 5.1 lit un .ps1 sans BOM en ANSI).

[CmdletBinding()]
param(
  [switch]$Apply,
  [ValidatePattern('^[A-Za-z0-9][A-Za-z0-9._-]*$')][string]$Lot,
  [ValidateSet('All', 'Caches', 'Sessions', 'Tmp', 'Worktrees')][string]$Scope = 'All',
  [string]$SessionRoot = (Join-Path $env:TEMP 'claude'),
  [string]$TranscriptRoot = (Join-Path $env:USERPROFILE '.claude\projects'),
  [string]$LogPath = (Join-Path $env:LOCALAPPDATA 'levelup-disk-hygiene\disk-hygiene.log')
)

$ErrorActionPreference = 'Stop'

$CacheIdleHours = 48
$SessionIdleHours = 72
$GateIdleHours = 24
$TmpIdleHours = 72
$SharedCacheMaxGB = 25
$AlertFreeGB = 60
$LotBusyMinutes = 10
$ShortTmpRoot = 'C:\t'
$GatePattern = '^(gate_work|gocache|golangci|parc)'
$BlockingPattern = '^(go|compile|link|golangci-lint|gate|asm|cgo)$|\.test$|^replay-|^levelup'
$SessionPattern = '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
$ReparseFlag = [IO.FileAttributes]::ReparsePoint

$script:DoDelete = [bool]$Apply
$script:Freed = [int64]0
$script:Refused = 0

function Get-FullPath([string]$p) {
  if (Test-Path -LiteralPath $p) { return (Get-Item -LiteralPath $p -Force).FullName.TrimEnd('\') }
  return [IO.Path]::GetFullPath($p).TrimEnd('\')
}

$LocalAppData = Get-FullPath $env:LOCALAPPDATA
$UserProfile = Get-FullPath $env:USERPROFILE
$SessionRoot = Get-FullPath $SessionRoot
$TranscriptRoot = Get-FullPath $TranscriptRoot
$RepoRoot = Split-Path -Parent $PSScriptRoot

function Write-Log([string]$msg) {
  $line = '{0} {1}' -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $msg
  $dir = Split-Path -Parent $LogPath
  if (-not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
  Add-Content -LiteralPath $LogPath -Value $line -Encoding ASCII
  # Write-Host et non Write-Output : les fonctions qui journalisent rendent aussi une valeur.
  Write-Host $line
}

function Get-FreeGB { [math]::Round((Get-PSDrive -Name C).Free / 1GB, 2) }

function Format-GB([int64]$bytes) { '{0:N2} Go' -f ($bytes / 1GB) }

# Parcours iteratif qui ne descend jamais dans un point de reparation : taille, ecriture la
# plus recente, et liste des points de reparation rencontres.
function Get-TreeInfo([string]$root) {
  $info = @{ Bytes = [int64]0; NewestUtc = [datetime]::MinValue; Reparse = New-Object System.Collections.Generic.List[string] }
  $rootItem = New-Object IO.DirectoryInfo($root)
  $info.NewestUtc = $rootItem.LastWriteTimeUtc
  $stack = New-Object System.Collections.Generic.Stack[string]
  $stack.Push($root)
  while ($stack.Count -gt 0) {
    $dir = New-Object IO.DirectoryInfo($stack.Pop())
    foreach ($e in $dir.EnumerateFileSystemInfos()) {
      if ($e.Attributes -band $ReparseFlag) { $info.Reparse.Add($e.FullName); continue }
      if ($e.LastWriteTimeUtc -gt $info.NewestUtc) { $info.NewestUtc = $e.LastWriteTimeUtc }
      if ($e -is [IO.DirectoryInfo]) { $stack.Push($e.FullName) } else { $info.Bytes += $e.Length }
    }
  }
  return $info
}

function Get-IdleHours([datetime]$newestUtc) {
  return ((Get-Date).ToUniversalTime() - $newestUtc).TotalHours
}

function Test-StrictChild([string]$path, [string]$root) {
  return $path.StartsWith($root + '\', [StringComparison]::OrdinalIgnoreCase)
}

# Regle 1 : une suppression n'est permise que dans les racines autorisees.
function Test-Deletable([string]$path) {
  $parent = Split-Path -Parent $path
  $name = Split-Path -Leaf $path
  $eq = { param($a, $b) [string]::Equals($a, $b, [StringComparison]::OrdinalIgnoreCase) }
  if ((& $eq $parent $LocalAppData) -and ($name -like 'go-build-*')) { return $true }
  if ((& $eq $parent $LocalAppData) -and ($name -like 'golangci-*') -and ($name -ne 'golangci-lint')) { return $true }
  if ((& $eq $parent $UserProfile) -and ($name -like 'gocache-*')) { return $true }
  if ((& $eq $parent 'C:\') -and ($name -like '*-gocache')) { return $true }
  if (& $eq $parent $ShortTmpRoot) { return $true }
  if (Test-StrictChild $path $SessionRoot) { return $true }
  return $false
}

# Supprime un arbre si toutes les regles le permettent ; rend le nombre d'octets liberes
# (ou qui le seraient, en simulation).
function Remove-Tree([string]$path, [double]$minIdleHours, [string]$label) {
  $path = Get-FullPath $path
  if (-not (Test-Deletable $path)) { Write-Log "REFUS hors des racines autorisees : $path"; $script:Refused++; return [int64]0 }
  $item = Get-Item -LiteralPath $path -Force
  if ($item.Attributes -band $ReparseFlag) { Write-Log "REFUS la racine est un point de reparation : $path"; $script:Refused++; return [int64]0 }
  try { $info = Get-TreeInfo $path } catch {
    Write-Log ("REFUS {0} : parcours impossible ({1}) : {2}" -f $label, $_.Exception.Message, $path)
    $script:Refused++
    return [int64]0
  }
  $idle = Get-IdleHours $info.NewestUtc
  if ($info.Reparse.Count -gt 0) {
    Write-Log ("REFUS {0} : {1} point(s) de reparation, arbre conserve : {2}" -f $label, $info.Reparse.Count, $path)
    foreach ($r in $info.Reparse) { Write-Log "  point de reparation : $r" }
    $script:Refused++
    return [int64]0
  }
  if ($idle -lt $minIdleHours) { return [int64]0 }
  $desc = '{0} {1} ({2}, inactif depuis {3:N0} h)' -f $label, $path, (Format-GB $info.Bytes), $idle
  if (-not $script:DoDelete) { Write-Log "SIMULATION supprimerait $desc"; return $info.Bytes }
  Remove-Item -LiteralPath $path -Recurse -Force -ErrorAction SilentlyContinue
  if (Test-Path -LiteralPath $path) { cmd /c "rmdir /s /q `"\\?\$path`"" 2>&1 | Out-Null }
  if (Test-Path -LiteralPath $path) { Write-Log "INCOMPLET $desc"; return [int64]0 }
  Write-Log "SUPPRIME $desc"
  return $info.Bytes
}

function Get-BlockingProcess {
  $procs = Get-Process | Where-Object { $_.ProcessName -match $BlockingPattern }
  return @($procs | ForEach-Object { '{0}({1})' -f $_.ProcessName, $_.Id })
}

function Invoke-LotClosure([string]$lot) {
  $minIdle = $LotBusyMinutes / 60.0
  foreach ($name in @("go-build-$lot", "golangci-$lot")) {
    $path = Join-Path $LocalAppData $name
    if (-not (Test-Path -LiteralPath $path)) { Write-Log "lot $lot : $name absent"; continue }
    $info = Get-TreeInfo $path
    if ((Get-IdleHours $info.NewestUtc) -lt $minIdle) {
      Write-Log "REFUS lot $lot : $name a recu une ecriture il y a moins de $LotBusyMinutes min (en usage ?)"
      $script:Refused++
      continue
    }
    $script:Freed += Remove-Tree $path $minIdle "cache du lot $lot"
  }
}

function Invoke-SharedCacheTrim {
  $go = Get-Command go -ErrorAction SilentlyContinue
  if (-not $go) { Write-Log 'cache partage : go absent du PATH, ignore'; return }
  $shared = Get-FullPath (& go env GOCACHE)
  $expected = Join-Path $LocalAppData 'go-build'
  if (-not [string]::Equals($shared, $expected, [StringComparison]::OrdinalIgnoreCase)) {
    Write-Log "cache partage : GOCACHE=$shared n'est pas le cache par defaut, ignore"
    return
  }
  if (-not (Test-Path -LiteralPath $shared)) { return }
  $info = Get-TreeInfo $shared
  $msg = 'cache partage {0} : {1}' -f $shared, (Format-GB $info.Bytes)
  if ($info.Bytes -lt ($SharedCacheMaxGB * 1GB)) { Write-Log "$msg, sous le seuil de $SharedCacheMaxGB Go"; return }
  if (-not $script:DoDelete) { Write-Log "SIMULATION $msg, go clean -cache"; $script:Freed += $info.Bytes; return }
  & go clean -cache
  Write-Log "VIDE $msg (go clean -cache, code $LASTEXITCODE)"
  $script:Freed += $info.Bytes
}

function Invoke-CacheSweep {
  $dedicated = @(Get-ChildItem -LiteralPath $LocalAppData -Directory -Force -Filter 'go-build-*')
  $dedicated += @(Get-ChildItem -LiteralPath $LocalAppData -Directory -Force -Filter 'golangci-*' | Where-Object { $_.Name -ne 'golangci-lint' })
  $dedicated += @(Get-ChildItem -LiteralPath $UserProfile -Directory -Force -Filter 'gocache-*')
  $dedicated += @(Get-ChildItem -LiteralPath 'C:\' -Directory -Force -Filter '*-gocache' -ErrorAction SilentlyContinue)
  foreach ($d in $dedicated) { $script:Freed += Remove-Tree $d.FullName $CacheIdleHours 'cache dedie' }
  Invoke-SharedCacheTrim
}

# Sous-dossiers de gate d'une session active : parcours qui s'arrete sur le premier dossier
# dont le nom correspond, et ne descend jamais dans un point de reparation.
function Find-GateDirs([string]$root) {
  $found = New-Object System.Collections.Generic.List[string]
  $stack = New-Object System.Collections.Generic.Stack[string]
  $stack.Push($root)
  while ($stack.Count -gt 0) {
    $dir = New-Object IO.DirectoryInfo($stack.Pop())
    foreach ($c in $dir.EnumerateDirectories()) {
      if ($c.Attributes -band $ReparseFlag) { continue }
      if ($c.Name -match $GatePattern) { $found.Add($c.FullName) } else { $stack.Push($c.FullName) }
    }
  }
  return $found
}

function Get-SessionActivityUtc([string]$sessionDir) {
  $project = Split-Path -Leaf (Split-Path -Parent $sessionDir)
  $id = Split-Path -Leaf $sessionDir
  $newest = (Get-TreeInfo $sessionDir).NewestUtc
  $transcript = Join-Path (Join-Path $TranscriptRoot $project) "$id.jsonl"
  if (Test-Path -LiteralPath $transcript) {
    $t = (Get-Item -LiteralPath $transcript -Force).LastWriteTimeUtc
    if ($t -gt $newest) { $newest = $t }
  }
  return $newest
}

function Invoke-SessionSweep {
  if (-not (Test-Path -LiteralPath $SessionRoot)) { Write-Log "sessions : $SessionRoot absent"; return }
  foreach ($project in (Get-ChildItem -LiteralPath $SessionRoot -Directory -Force)) {
    if ($project.Attributes -band $ReparseFlag) { continue }
    foreach ($s in (Get-ChildItem -LiteralPath $project.FullName -Directory -Force)) {
      if ($s.Name -notmatch $SessionPattern -or ($s.Attributes -band $ReparseFlag)) { continue }
      try {
        $idle = Get-IdleHours (Get-SessionActivityUtc $s.FullName)
        if ($idle -ge $SessionIdleHours) {
          $script:Freed += Remove-Tree $s.FullName $SessionIdleHours 'session inactive'
          continue
        }
        foreach ($g in (Find-GateDirs $s.FullName)) {
          $script:Freed += Remove-Tree $g $GateIdleHours 'dossier de gate'
        }
      } catch {
        Write-Log ("ERREUR session {0} : {1} ; ignoree" -f $s.FullName, $_.Exception.Message)
        $script:Refused++
      }
    }
  }
}

function Invoke-TmpSweep {
  if (-not (Test-Path -LiteralPath $ShortTmpRoot)) { return }
  foreach ($d in (Get-ChildItem -LiteralPath $ShortTmpRoot -Directory -Force)) {
    $script:Freed += Remove-Tree $d.FullName $TmpIdleHours 'TMP court'
  }
}

# Regle 5 : rapport seul. Jonctions cherchees aux emplacements ou les sessions en posent.
function Write-WorktreeReport {
  $lines = & git -C $RepoRoot worktree list --porcelain 2>$null
  if ($LASTEXITCODE -ne 0) { Write-Log "worktrees : git indisponible sous $RepoRoot"; return }
  $paths = @($lines | Where-Object { $_ -like 'worktree *' } | ForEach-Object { $_.Substring(9) }) | Select-Object -Skip 1
  $junctionSpots = @('apps\web\node_modules', 'data\cache\film_chunks', 'data\cache\film_manifests', 'node_modules')
  foreach ($wt in $paths) {
    $head = & git -C $wt rev-parse HEAD 2>$null
    & git -C $wt merge-base --is-ancestor $head origin/feat/v75 2>$null
    $merged = ($LASTEXITCODE -eq 0)
    $dirty = @(& git -C $wt status --short 2>$null).Count
    $last = & git -C $wt log -1 --format=%ci 2>$null
    $junctions = @($junctionSpots | Where-Object {
        $p = Join-Path $wt $_
        (Test-Path -LiteralPath $p) -and ((Get-Item -LiteralPath $p -Force).Attributes -band $ReparseFlag)
      })
    Write-Log ("worktree {0} : dernier commit {1}, fusionne dans origin/feat/v75 {2}, non commites {3}, jonctions [{4}]" -f $wt, $last, $merged, $dirty, ($junctions -join ', '))
  }
}

$mode = if ($Apply) { 'APPLY' } else { 'SIMULATION' }
Write-Log ("=== disk-hygiene {0} scope={1} lot={2} : libre {3} Go" -f $mode, $Scope, $Lot, (Get-FreeGB))

if ($Lot) {
  Invoke-LotClosure $Lot
} else {
  $blocking = @(Get-BlockingProcess)
  if ($blocking.Count -gt 0 -and $script:DoDelete) {
    Write-Log ("processus actifs : {0} ; aucune suppression, rapport seul" -f ($blocking -join ' '))
    $script:DoDelete = $false
  }
  if ($Scope -in 'All', 'Caches') { Invoke-CacheSweep }
  if ($Scope -in 'All', 'Sessions') { Invoke-SessionSweep }
  if ($Scope -in 'All', 'Tmp') { Invoke-TmpSweep }
  if ($Scope -in 'All', 'Worktrees') { Write-WorktreeReport }
}

$verb = if ($script:DoDelete) { 'libere' } else { 'liberable' }
$free = Get-FreeGB
Write-Log ("=== fin : {0} {1}, refus {2}, libre {3} Go" -f $verb, (Format-GB $script:Freed), $script:Refused, $free)
if ($free -lt $AlertFreeGB) { Write-Log "ALERTE moins de $AlertFreeGB Go libres sur C:" }
