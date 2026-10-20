#!/bin/bash
# carte.sh <bin> <dir> : carte de fermeture v2 sur les 20 films, une ligne par paquet.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-LN/env.sh
cd $L
$1 -racine $RACINE -films $FILMS -sortie $2 -table $L/ecs_table_base.tsv -mode v2 -denominateur-fixe $I/denominateurs.tsv -paquets -plafond-gib 4 > $2.log 2>&1
echo "rc=$?" >> $2.log
