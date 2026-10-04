#!/bin/bash
# carte.sh <nom> : carte de fermeture v2 sur les 20 films.
source /c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ/rev/env.sh
cd $R
$R/bin/$1/fermeture.exe -racine $RACINE -films $FILMS -sortie $(cygpath -m $R/carte_$1) -table $(cygpath -m $R/ecs_table_$1.tsv) -mode v2 -denominateur-fixe $(cygpath -m $I2/denominateurs.tsv) -paquets -plafond-gib 4 > $R/carte_$1.log 2>&1
echo "rc=$?" >> $R/carte_$1.log
