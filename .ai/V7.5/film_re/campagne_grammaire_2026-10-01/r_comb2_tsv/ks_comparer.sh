#!/bin/bash
# ks_comparer.sh : compare chaque variante killsource a la production, mort par mort.
# Sorties : ks_resume.tsv (variante x film), ks_morts_changees.tsv (une ligne par mort changee).
SP=/c/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp/f46f71fc-4042-4343-b5c8-623c9fca58c2/scratchpad
cd $SP/rcomb2/ks
morts() { jq -r '.morts[] | [.temps_ms, .victime, .lecture.voie_identifiant, .source_du_degat_fatal.tag, .source_du_degat_fatal.statut, .source_du_degat_fatal.nature, .credit_du_jeu.joueur, (.assistant.joueur // "-"), (.parts_de_degats.part_du_tueur_pct|tostring), (.divergence|tostring)] | @tsv' "$1" | awk -F'\t' 'BEGIN{OFS="\t"} {k=$1"|"$2; n[k]++; $1=k"#"n[k]; $2=""; print}'; }
sante() { jq -r '[.sante.verdict, (.sante.alertes|length), .couverture.morts_couvertes] | @tsv' "$1"; }
echo -e "variante\tfilm\tmorts_ref\tmorts_var\tapparues\tdisparues\tscan_vers_marche\tmarche_vers_scan\ttag_change\tstatut_change\tnature_change\tcredit_change\tassistant_change\tpart_tueur_change\tdivergence_change\tmarche_ref\tmarche_var\tverdict_ref\tverdict_var\talertes_ref\talertes_var\tcouvertes_ref\tcouvertes_var\tjson_identique" > ../ks_resume.tsv
echo -e "variante\tfilm\tmort\tchamp\tref\tvar" > ../ks_morts_changees.tsv
for d in */; do v=${d%/}; [ "$v" = prod ] && continue
  for f in prod/*.json; do id=$(basename $f .json); g=$v/$id.json; [ -s "$g" ] || continue
    ident=non; cmp -s $f $g && ident=oui
    paste <(sante $f) <(sante $g) > /tmp/rc2_s.$$
    awk -F'\t' -v v=$v -v id=$id -v ident=$ident -v sfile=/tmp/rc2_s.$$ -v det=../ks_morts_changees.tsv '
      BEGIN{OFS="\t"; getline s < sfile; split(s, S, "\t"); champs[3]="voie";champs[4]="tag";champs[5]="statut";champs[6]="nature";champs[7]="credit";champs[8]="assistant";champs[9]="part_tueur";champs[10]="divergence"}
      FNR==NR {R[$1]=$0; nr++; if($3=="marche") mr++; next}
      {nv++; if($3=="marche") mv++; if(!($1 in R)){app++; print v,id,$1,"mort","absente",$3 >> det; next}
       split(R[$1], a, "\t"); vu[$1]=1
       if(a[3]!=$3){ if(a[3]=="scan"&&$3=="marche") sm++; else if(a[3]=="marche"&&$3=="scan") ms++; print v,id,$1,"voie",a[3],$3 >> det }
       for(i=4;i<=10;i++) if(a[i]!=$i){c[i]++; print v,id,$1,champs[i],a[i],$i >> det}}
      END{for(k in R) if(!(k in vu)){dis++; split(R[k],a,"\t"); print v,id,k,"mort",a[3],"absente" >> det}
        print v,id,nr+0,nv+0,app+0,dis+0,sm+0,ms+0,c[4]+0,c[5]+0,c[6]+0,c[7]+0,c[8]+0,c[9]+0,c[10]+0,mr+0,mv+0,S[1],S[4],S[2],S[5],S[3],S[6],ident}' <(morts $f) <(morts $g) >> ../ks_resume.tsv
    rm -f /tmp/rc2_s.$$
  done
done
# champs hors morts qui changent (chemin sans valeur), par variante et film
plat(){ jq -r 'del(.morts) | paths(scalars) as $p | "\($p|map(tostring)|join("."))\t\(getpath($p))"' $1 | sort; }
echo -e "variante\tfilm\tchemin" > ../ks_champs_hors_morts.tsv
for d in */; do v=${d%/}; [ "$v" = prod ] && continue
  for f in prod/*.json; do id=$(basename $f .json); g=$v/$id.json; [ -s "$g" ] || continue
    cmp -s $f $g && continue
    diff <(plat $f) <(plat $g) | grep '^[<>]' | cut -c3- | cut -f1 | sed -E 's/\.[0-9]+(\.|$)/.N\1/g' | sort -u | awk -v v=$v -v id=$id '{print v"\t"id"\t"$0}' >> ../ks_champs_hors_morts.tsv
  done
done
