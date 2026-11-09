#!/bin/bash
# denominateur.sh <configs_r_comb2.tsv> <sortie> : le denominateur fixe CONSOLIDE de R-COMB-2,
# recalcule depuis les TSV bruts de la campagne, chaque (source, variante) classee.
CG=/c/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire/.ai/V7.5/film_re/campagne_grammaire_2026-10-01
RC2=$1; OUT=$2
{
  # source  fichier  colonne variante  colonne utiles_lus
  for x in "bis1 mesures_bis_tsv/mb_variantes.tsv 3 7" "bis3 mesures_bis3_tsv/mb3_variantes.tsv 3 8" \
           "bis3-ti3 mesures_bis3_tsv/mb3_ti3.tsv 3 7" "bis2-ti43 mesures_bis2_tsv/mb2_ti43.tsv 3 7" \
           "bis2-positions mesures_bis2_tsv/mb2_positions.tsv 3 7" "bis2-positions-prod mesures_bis2_tsv/mb2_positions_production.tsv 3 7" \
           "r-comb r_comb_tsv/r_comb_configs.tsv 4 14" "r-comb-deriv r_comb_tsv/r_comb_derivation.tsv 4 12" \
           "r-loc r_loc_tsv/rloc_variantes.tsv 3 10" "r-loc r_loc_tsv/rloc_variantes_passe2_ls2.tsv 3 10" \
           "r-nais r_nais_tsv/r_nais_variantes.tsv 3 10" "r-veh r_veh_tsv/r_veh_delta.tsv 3 7" \
           "r-comp-l3 r_comp_tsv/r_comp_l3_delta.tsv 3 9" "r-comp-p3 r_comp_tsv/r_comp_p3.tsv 3 9"; do
    set -- $x
    awk -F'\t' -v s=$1 -v cv=$3 -v cu=$4 'BEGIN{OFS="\t"} FNR>1 {print $1, s, $cv, $cu}' $CG/$2
  done
  awk -F'\t' 'BEGIN{OFS="\t"} FNR>1 {print $1, "r-comb-2", $5, $17}' $RC2
} > $OUT/denom_brut.tsv
# format par film (LM : formats 24-25 seulement)
awk -F'\t' 'FNR>1 && $5=="reference" {print $1"\t"$3"\t"$2}' $RC2 > $OUT/denom_formats.tsv
awk -F'\t' -v out=$OUT 'BEGIN{OFS="\t"}
  FNR==NR {fmt[$1]=$2; bld[$1]=$3; next}
  function classe(s, v, f,   m) {
    if (v ~ /^largeur-/ || v ~ /\(temoin\)/) return "exclu:temoin negatif"
    if (v ~ /^controle:/) return "exclu:controle"
    if (s == "r-loc" && (v == "libre" || v == "ls+libre")) return "exclu:repli libre, rejete (D-77)"
    if (s == "r-loc" && v == "ls+vueA") return "exclu:oracle de la vue A (L1b retire, D15 ; 45 % de gains factices)"
    if (s == "r-veh" && v ~ /mpp8\/3/ && f != 24 && f != 25) return "exclu:mpp 8/3 hors formats 24-25 (temoin)"
    if (s == "r-veh" && (v ~ /portee-neuf/ || v ~ /feuille4-brute/)) return "exclu:rejete par la mesure (R_VEH 3.2)"
    if (s == "r-nais" && v ~ /hors-evenements/) return "exclu:rejete par la mesure (R_NAIS 3.2)"
    if (s == "r-comp-p3" && v ~ /NEW a masque impossible|desaveu des deux/) return "exclu:rejete par la mesure (R_COMP 4.4)"
    if (s == "r-comp-l3" && v ~ /i4/) return "exclu:hypothese de format des vieux builds (L3b non discrimine)"
    if ((s == "bis2-positions" || s == "bis2-positions-prod") && v ~ /^jeu:/ && v != "jeu:flock-position" && v != "jeu:tacmap-displayasset" && v != "jeu:world-object-i0") return "exclu:site rejete ou melange de sites rejetes (BIS_2/BIS_4)"
    if (v ~ /P6/) return "exclu:oracle sans lot (P6, D3)"
    if (v ~ /\(iii/ || v ~ /^a-\(iii/) return "exclu:oracle de L1c (retire)"
    return "inclus"
  }
  { c = classe($2, $3, fmt[$1]); n[$2 SUBSEP $3 SUBSEP c]++
    if (c == "inclus" && $4 + 0 > mx[$1]) { mx[$1] = $4 + 0; src[$1] = $2 ": " $3 }
    if (c !~ /temoin|controle/ && !($2 == "r-loc" && c ~ /^exclu/) && $4 + 0 > lg[$1]) { lg[$1] = $4 + 0; slg[$1] = $2 ": " $3 }
  }
  END {
    print "film", "build", "fixe", "source_du_fixe", "fixe_large", "source_large" > (out "/r_comb2_denominateurs.tsv")
    for (f in mx) print f, bld[f], mx[f], src[f], lg[f], slg[f] > (out "/r_comb2_denominateurs.tsv")
    print "source", "variante", "classe", "lignes" > (out "/r_comb2_denominateurs_classes.tsv")
    for (k in n) { split(k, a, SUBSEP); print a[1], a[2], a[3], n[k] > (out "/r_comb2_denominateurs_classes.tsv") }
  }' $OUT/denom_formats.tsv $OUT/denom_brut.tsv
