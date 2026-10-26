#!/bin/bash
# ks_cmp.sh : morts killsource base contre lot, appariees par (instant, victime) ; voie, tag, statut, credit.
L=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/v2-integ
J=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad/rloc_ks/extraire.jq
mkdir -p $L/ks_tsv
echo -e "film\tidentique\tmorts_b\tmorts_n\tmarche_b\tscan_b\tmarche_n\tscan_n\tscan_vers_marche\tmarche_vers_scan\tautre_voie\ttag\tstatut\tcredit\tdisparues\tapparues\torigine_credit_concordant_b\torigine_credit_concordant_n\tout_of_roster_b\tout_of_roster_n"
while IFS=$'\t' read -r id carte; do
  b=$L/ks_base/$id.json; n=$L/ks_tete/$id.json
  cmp -s $b $n && ident=oui || ident=non
  jq -r -f $J $b > $L/ks_tsv/${id}_b.tsv; jq -r -f $J $n > $L/ks_tsv/${id}_n.tsv
  ccb=$(awk -F'\t' '$4=="scan"&&$5 ~ /concordent/' $L/ks_tsv/${id}_b.tsv | wc -l)
  ccn=$(awk -F'\t' '$4=="scan"&&$5 ~ /concordent/' $L/ks_tsv/${id}_n.tsv | wc -l)
  orb=$(jq '[.sante.compteurs_expvar[]|select(.nom=="killsource_out_of_roster")|.valeur][0]' $b)
  orn=$(jq '[.sante.compteurs_expvar[]|select(.nom=="killsource_out_of_roster")|.valeur][0]' $n)
  awk -F'\t' -v id=$id -v ident=$ident -v ccb=$ccb -v ccn=$ccn -v orb=$orb -v orn=$orn 'FNR==NR{k=$2"|"$3; rv[k]=$4; rt[k]=$6; rs[k]=$7; rc[k]=$8; nb++; if($4=="marche")mb++; else if($4=="scan")sb++; next}
   {k=$2"|"$3; nn++; if($4=="marche")mn++; else if($4=="scan")sn++; if(!(k in rv)){app++;next} seen[k]=1;
    if(rv[k]=="scan"&&$4=="marche")sm++; else if(rv[k]=="marche"&&$4=="scan")ms++; else if(rv[k]!=$4)va++;
    if(rt[k]!=$6)tc++; if(rs[k]!=$7)sc++; if(rc[k]!=$8)cc++}
   END{for(k in rv) if(!(k in seen)) dis++; printf "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n", id,ident,nb,nn,mb,sb,mn,sn,sm,ms,va,tc,sc,cc,dis,app,ccb,ccn,orb,orn}' $L/ks_tsv/${id}_b.tsv $L/ks_tsv/${id}_n.tsv
done < $L/cartes.tsv
