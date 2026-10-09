#!/bin/bash
# replis_cmp.sh : replis par film base contre tete (journaux replay-equiv) ; repli_debut_de_liste_ferme_au_bit en tete.
V=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ
$V/replis.sh $V/re_base.log > $V/replis_base.tsv; $V/replis.sh $V/re_tete.log > $V/replis_tete.tsv
awk -F'\t' 'NR==FNR{b[$1"\t"$2]=$3; k[$1"\t"$2]=1; next}{t[$1"\t"$2]=$3; k[$1"\t"$2]=1}
 END{print "film\trepli\tbase\ttete"; for(x in k) if(b[x]+0!=t[x]+0) print x"\t"b[x]+0"\t"t[x]+0}' $V/replis_base.tsv $V/replis_tete.tsv | sort > $V/replis_changes.tsv
awk -F'\t' 'NR==FNR{if($2=="repli_debut_de_liste_ferme_au_bit")b[$1]=$3; f[$1]=1; next}{if($2=="repli_debut_de_liste_ferme_au_bit")t[$1]=$3; f[$1]=1}
 END{print "film\tbase\ttete"; for(x in f){print x"\t"b[x]+0"\t"t[x]+0; B+=b[x];T+=t[x]} print "total\t"B"\t"T}' $V/replis_base.tsv $V/replis_tete.tsv | sort > $V/repli_ferme_au_bit.tsv
