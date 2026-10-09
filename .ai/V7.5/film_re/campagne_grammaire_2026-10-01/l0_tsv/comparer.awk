# comparer.awk : avant (carte_A, marche de reference, ferme = ferme au bit) contre apres (definition L0).
# colonnes : 1 film 2 chunk 3 paquet 4 liste 5 sortie 6 ferme_au_bit 7 ferme 8 premiere 9 regles 10 cause 11 utiles_lus
BEGIN { FS = OFS = "\t" }
FNR == 1 { next }
NR == FNR { k = $1 SUBSEP $2 SUBSEP $3; av[k] = $7; avU[k] = $11; avL[k] = $4; next }
{
  k = $1 SUBSEP $2 SUBSEP $3; f = $1; films[f] = 1
  a = (av[k] == "true"); p = ($7 == "true"); pab = ($6 == "true")
  if (!(k in av)) { absent[f]++ }
  if (a) { fa[f]++; ua[f] += avU[k] }
  if (p) { fp[f]++; up[f] += $11 }
  if (a && p) { cons[f]++ }
  if (a && !p && pab) { fact[f]++; ufact[f] += avU[k]; regle[$8]++ ; uregle[$8] += avU[k] }
  if (a && !p && !pab) { autre[f]++; uautre[f] += avU[k]; cause[f, $10]++ }
  if (!a && p) { gain[f]++; ugain[f] += $11 }
  if (avL[k] != $4) { liste[f]++ }
  if (a && p && avU[k] != $11) { uchg[f]++ }
}
END {
  print "film", "fermes_avant", "fermes_apres", "conserves", "retires_invariant", "autres_baisses", "gains", "utiles_fermes_avant", "utiles_fermes_apres", "utiles_retires_invariant", "utiles_autres_baisses", "utiles_gains", "paquets_absents", "liste_changee", "utiles_changes_sur_conserves"
  for (f in films) print f, fa[f]+0, fp[f]+0, cons[f]+0, fact[f]+0, autre[f]+0, gain[f]+0, ua[f]+0, up[f]+0, ufact[f]+0, uautre[f]+0, ugain[f]+0, absent[f]+0, liste[f]+0, uchg[f]+0
  for (r in regle) print "REGLE", r, regle[r], uregle[r] > "/dev/stderr"
  for (x in cause) { split(x, c, SUBSEP); print "AUTRE_BAISSE", c[1], c[2], cause[x] > "/dev/stderr" }
}
