#!/bin/bash
# carte.sh <base|neuf> : carte de fermeture v2 sur les 20 films (recette integ2/integ3).
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LT/env.sh
cd $L
$L/bin/$1/fermeture.exe -racine $RACINE -films $FILMS -sortie $L/carte_$1 -table $(cygpath -m $L/ecs_table_$1.tsv) -mode v2 -denominateur-fixe $I2/denominateurs.tsv -paquets -plafond-gib 4 > $L/carte_$1.log 2>&1
echo "rc=$?" >> $L/carte_$1.log
