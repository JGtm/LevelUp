#!/bin/bash
# usage: gate2.sh <avant> <apres> : gate 2 paquet par paquet + causes
S=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/bfd187a4-ee4b-4656-9ab4-0654cbc1dfb8/scratchpad
W=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-grammaire-arrets-vue-b-2; A=$W/.ai/V7.5/film_re/arrets_vue_b_2026-10-07/tsv
awk -F'\t' -v PERDUS=$S/perdus_$2.tsv -f $A/gate2.awk $S/carte_$1/fermeture_paquets.tsv $S/carte_$2/fermeture_paquets.tsv > $S/gate2_$2_contre_$1.tsv
awk -F'\t' 'FNR==1{f++; next} $7!="true"{ if(f==1) a[$10]++; else b[$10]++; k[$10]=1 } END{ print "cause\tavant\tapres"; for (x in k) print x"\t"a[x]+0"\t"b[x]+0 }' $S/carte_$1/fermeture_paquets.tsv $S/carte_$2/fermeture_paquets.tsv | sort -t$'\t' -k3,3nr > $S/causes_$2.tsv
