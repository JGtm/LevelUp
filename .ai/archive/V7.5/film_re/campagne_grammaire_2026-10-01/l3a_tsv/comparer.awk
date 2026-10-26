# comparer.awk avant/fermeture_paquets.tsv apres/fermeture_paquets.tsv
# colonnes : 1 film 2 chunk 3 paquet 4 liste 5 sortie 6 ferme_au_bit 7 ferme 8 premiere 9 regles 10 cause 11 utiles_lus
# sain = ferme (definition L0 : ferme au bit ET aucune regle de l ecrivain contredite) et regles = "-"
BEGIN { FS = OFS = "\t" }
FNR == 1 { next }
NR == FNR { k = $1 SUBSEP $2 SUBSEP $3; aS[k] = ($7 == "true" && $9 == "-"); aB[k] = ($6 == "true"); aU[k] = $11; aC[k] = $10; aR[k] = $8
  if (aS[k]) { sa[$1]++; usa[$1] += $11 }; if (aB[k]) ba[$1]++; ula[$1] += $11; vu[$1] = 1; next }
{ k = $1 SUBSEP $2 SUBSEP $3; f = $1; films[f] = 1
  s = ($7 == "true" && $9 == "-"); b = ($6 == "true")
  if (!(k in aS)) absent[f]++
  if (s) { sp[f]++; usp[f] += $11 }
  if (b) bp[f]++
  ulp[f] += $11
  if (aS[k] && !s) { perdu[f]++; if (b) { pc[f]++; print "SAIN_DEVENU_CONTREDIT", f, $2 ":" $3, $8, $10 > "/dev/stderr" } else { pn[f]++; print "SAIN_DEVENU_NON_FERME", f, $2 ":" $3, $10 > "/dev/stderr" } }
  if (!aB[k] && b) { gb[f]++; if (s) gs[f]++; else { gc[f]++; print "GAIN_CONTREDIT", f, $2 ":" $3, $8 > "/dev/stderr" } }
  if (aB[k] && !aS[k] && s) { requal[f]++ }
  if (aB[k] && !b) { pb[f]++; if (!aS[k]) print "FACTICE_PERDU", f, $2 ":" $3, aR[k], "->", $10 > "/dev/stderr" }
}
END {
  print "film", "sains_avant", "sains_apres", "net_sains", "sains_perdus", "dont_devenus_contredits", "dont_devenus_non_fermes", "gagnes_au_bit", "dont_sains", "dont_contredits", "contredits_devenus_sains", "fermes_au_bit_perdus", "utiles_sains_avant", "utiles_sains_apres", "net_utiles_sains", "utiles_lus_avant", "utiles_lus_apres", "paquets_absents"
  for (f in films) print f, sa[f]+0, sp[f]+0, sp[f]-sa[f], perdu[f]+0, pc[f]+0, pn[f]+0, gb[f]+0, gs[f]+0, gc[f]+0, requal[f]+0, pb[f]+0, usa[f]+0, usp[f]+0, usp[f]-usa[f], ula[f]+0, ulp[f]+0, absent[f]+0
}
