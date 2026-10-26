# run_corpus.ps1 -- lance run_one.ps1 sur une LISTE de films, UN PROCESSUS PAR FILM.
#
# La boucle est ICI, dans le pilote, et jamais dans le processus `go test` : c'est la regle
# D17(a), et c'est elle qui borne la memoire (un film decode, un processus qui meurt, la
# memoire rendue). Un `go test` qui boucle sur le corpus a deja fait exploser un instrument a
# 3,3 Go sur cette machine.
#
# FICHIER STRICTEMENT ASCII (cf. run_one.ps1).
param(
  [Parameter(Mandatory = $true)][string[]]$Films,
  [string]$EnvVar = "GAME_FILM",
  [string]$Run = "^TestGameEntitiesPhase0$",
  [string]$Pkg = "./internal/analysis/filmdec/"
)
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
foreach ($f in $Films) {
  & (Join-Path $here "run_one.ps1") -Film $f -EnvVar $EnvVar -Run $Run -Pkg $Pkg
}
