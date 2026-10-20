#!/bin/bash
# analyser.sh <film> : sonde C4, base contre tete.
C4=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/L2/c4
f=$1; D=$C4/dump; O=$C4/out; mkdir -p $O
for v in base head; do
  awk -F'\t' '$1=="P"' $D/$v.$f.tsv | sort -u > $O/$v.$f.P
  awk -F'\t' '$1=="R"' $D/$v.$f.tsv | sort -u > $O/$v.$f.R
  awk -F'\t' '$1=="B"' $D/$v.$f.tsv | sort -u > $O/$v.$f.B
done
echo "## $f : classes de liste (paquets a evenements), base -> tete"
join -t$'\t' -a1 -a2 -e0 -o 0,1.2,2.2 <(awk -F'\t' '{k=$5; if($5!="sans_liste" && $5!="non_localisee") k=k"/"($9=="true"?"ferme":($8=="true"?"au_bit_contredit":"non_ferme")); print k}' $O/base.$f.P | sort | uniq -c | awk '{print $2"\t"$1}' | sort) <(awk -F'\t' '{k=$5; if($5!="sans_liste" && $5!="non_localisee") k=k"/"($9=="true"?"ferme":($8=="true"?"au_bit_contredit":"non_ferme")); print k}' $O/head.$f.P | sort | uniq -c | awk '{print $2"\t"$1}' | sort)
echo "## $f : lectures d etat retenues : base $(wc -l < $O/base.$f.R), tete $(wc -l < $O/head.$f.R)"
# lectures propres a un cote, avec la classe de leur paquet des deux cotes
awk -F'\t' 'FNR==NR{k=$2" "$3; cl[k]=$5"/"$9"/"$10; next} {print}' /dev/null /dev/null
for side in base head; do
  other=head; [ $side = head ] && other=base
  comm -23 $O/$side.$f.R $O/$other.$f.R > $O/seul_$side.$f.R
  echo "## $f : lectures seulement en $side : $(wc -l < $O/seul_$side.$f.R) ; classe du paquet (base -> tete) :"
  awk -F'\t' -v B=$O/base.$f.P -v H=$O/head.$f.P 'BEGIN{while((getline l < B)>0){split(l,a,"\t"); cb[a[2]" "a[3]]=a[5]"/"a[9]"/"a[10]} while((getline l < H)>0){split(l,a,"\t"); ch[a[2]" "a[3]]=a[5]"/"a[9]"/"a[10]}} {k=$2" "$3; print (k in cb?cb[k]:"absent")"  ->  "(k in ch?ch[k]:"absent")}' $O/seul_$side.$f.R | sort | uniq -c | sort -rn
done
echo "## $f : dotations de naissance (B), lignes qui different"
diff $O/base.$f.B $O/head.$f.B
