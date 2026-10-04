#!/bin/bash
# carte.sh <base|tete> : carte de fermeture v2 sur les 20 films, une ligne par paquet.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/env.sh
cd $V
$V/bin/$1/fermeture.exe -racine $RACINE -films $FILMS -sortie $(cygpath -m $V/carte_$1) -table $(cygpath -m $V/ecs_table.tsv) -mode v2 -denominateur-fixe $(cygpath -m $I2/denominateurs.tsv) -paquets -plafond-gib 4 > $V/carte_$1.log 2>&1
echo "rc=$?" >> $V/carte_$1.log
