# comparer.awk avant/fermeture_paquets.tsv apres/fermeture_paquets.tsv
# colonnes : 1 film 2 chunk 3 paquet 4 liste 5 sortie_vueB 6 ferme_au_bit 7 ferme 8 premiere_regle 9 regles 10 cause 11 utiles_lus
# sain = ferme (definition L0 : au bit ET aucune regle de l ecrivain contredite).
BEGIN { FS = OFS = "\t" }
FNR == 1 { next }
NR == FNR { k = $1 SUBSEP $2 SUBSEP $3; s = ($7 == "true" && $9 == "-"); aS[k] = s; aB[k] = ($6 == "true"); aU[k] = $11; aC[k] = $10
  if (s) { sa[$1]++; ua[$1] += $11 }; if ($6 == "true") ba[$1]++; la[$1] += $11; next }
{ k = $1 SUBSEP $2 SUBSEP $3; f = $1; films[f] = 1; s = ($7 == "true" && $9 == "-"); b = ($6 == "true")
  if (s) { sp[f]++; up[f] += $11 }; if (b) bp[f]++; lp[f] += $11
  if (!(k in aS)) absent[f]++
  if (aS[k] && !s) { perdu[f]++; uperdu[f] += aU[k]; if (b) pc[f]++; else pn[f]++; print "SAIN_PERDU", f, $2 ":" $3, "avant=" aC[k], "apres=" $10, "au_bit=" $6, "regles=" $9, "utiles=" aU[k] "->" $11 > "/dev/stderr" }
  if (!aS[k] && s) { gagne[f]++; ugagne[f] += $11 }
  if (!aB[k] && b && !s) { gfact[f]++; print "GAIN_FACTICE", f, $2 ":" $3, "regles=" $9 > "/dev/stderr" }
  if (!aB[k] && b) gbit[f]++
  if (aS[k] && s && aU[k] != $11) { uchg[f]++; duchg[f] += $11 - aU[k] }
}
END {
  print "film", "sains_avant", "sains_apres", "net", "gagnes", "perdus", "perdus_contredits", "perdus_non_fermes", "gains_au_bit", "gains_factices", "utiles_sains_avant", "utiles_sains_apres", "net_utiles", "utiles_perdus", "conserves_utiles_changes", "delta_utiles_conserves", "utiles_lus_avant", "utiles_lus_apres", "absents"
  for (f in films) print f, sa[f]+0, sp[f]+0, sp[f]-sa[f], gagne[f]+0, perdu[f]+0, pc[f]+0, pn[f]+0, gbit[f]+0, gfact[f]+0, ua[f]+0, up[f]+0, up[f]-ua[f], uperdu[f]+0, uchg[f]+0, duchg[f]+0, la[f]+0, lp[f]+0, absent[f]+0
}
