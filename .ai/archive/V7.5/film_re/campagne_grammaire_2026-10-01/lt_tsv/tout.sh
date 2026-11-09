#!/bin/bash
# tout.sh <base|neuf> : construction puis carte, killsource et replay-equiv, en serie.
L=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT
$L/construire.sh $1 > $L/construire_$1.log 2>&1 || { echo ECHEC_CONSTRUCTION > $L/tout_$1.done; exit 1; }
$L/carte.sh $1
$L/ks.sh $1
$L/re.sh $1
echo FINI > $L/tout_$1.done
