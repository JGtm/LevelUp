#!/bin/bash
# re_cmp.sh <dirA> <dirB> : etapes de replay-equiv qui different, par film et par etape (lignes non commentees).
for a in $1/*.tsv; do n=$(basename $a); b=$2/$n; [ -f $b ] || { echo "ABSENT $n"; continue; }
  awk -F'\t' -v f=${n%.tsv} 'FNR==NR{if($0!~/^#/)A[$1]=$2"\t"$3; next} $0!~/^#/{if(A[$1]!=$2"\t"$3) print f"\t"$1}' $a $b
done
