# agreger.awk : agregats de R-COMB-2.
#   awk -F'\t' -v sortie=<dir> -f agreger.awk denominateurs.tsv configs.tsv
# denominateurs.tsv : film, build, fixe (colonne 3 = fixe retenu) ; configs.tsv : r_comb2_configs.
# Colonnes de configs : 1 film 2 build 3 format 4 contexte 5 config ... 13 fermes 14 sains
# 15 utiles_fermes 16 utiles_sains 17 utiles_lus 18 hors_cadre 20 g_ref 21 p_ref 22 gs_ref 23 ps_ref
# 24 ps_ref_vers_contredit.
BEGIN { OFS = "\t"; HC = "81c02726" }
FNR == NR { if (FNR > 1) fixe[$1] = $3; next }
FNR == 1 { next }
$5 == "controle:reference-instruments" { next }
{
  f = $1; c = $5; b = $2
  films[f] = 1; build[f] = b; cfgs[c] = 1
  F[f, c] = $13; S[f, c] = $14; UF[f, c] = $15; US[f, c] = $16; UL[f, c] = $17; HCD[f, c] = $18
  GS[f, c] = $22; PS[f, c] = $23; PSC[f, c] = $24; G[f, c] = $20; P[f, c] = $21
}
function grp(f) { return f == HC ? "hors-corpus:" f : build[f] }
function pct(n, d) { return d > 0 ? sprintf("%.1f", 100 * n / d) : "-" }
END {
  # (a) par groupe et configuration
  print "config", "groupe", "films", "fermes", "fermes_sains", "utiles_fermes", "utiles_fermes_sains", "utiles_lus", \
    "fixe", "pct_fixe_sains", "pct_fixe_brut", "pct_variable_sains", "hors_cadre", "d_sains_vs_ref", "d_utiles_sains_vs_ref", \
    "films_en_baisse_paquets_sains", "films_en_baisse_utiles_sains", "gagnes", "perdus", "gagnes_sains", "perdus_sains", \
    "perdus_sains_vers_contredit" > (sortie "/r_comb2_par_build.tsv")
  for (c in cfgs) for (f in films) {
    if (!((f, c) in F)) continue
    for (k = 0; k < 2; k++) {
      g = (k == 0) ? grp(f) : (f == HC ? "" : "corpus-20")
      if (g == "") continue
      key = c SUBSEP g
      n[key]++; fe[key] += F[f, c]; sa[key] += S[f, c]; uf[key] += UF[f, c]; us[key] += US[f, c]; ul[key] += UL[f, c]
      fx[key] += fixe[f]; hcd[key] += HCD[f, c]; ds[key] += S[f, c] - S[f, "reference"]; du[key] += US[f, c] - US[f, "reference"]
      if (S[f, c] < S[f, "reference"]) bp[key]++
      if (US[f, c] < US[f, "reference"]) bu[key]++
      gg[key] += G[f, c]; pp[key] += P[f, c]; gs[key] += GS[f, c]; ps[key] += PS[f, c]; psc[key] += PSC[f, c]
      keys[key] = 1
    }
  }
  for (key in keys) {
    split(key, kk, SUBSEP)
    print kk[1], kk[2], n[key], fe[key], sa[key], uf[key], us[key], ul[key], fx[key], pct(us[key], fx[key]), pct(uf[key], fx[key]), \
      pct(us[key], ul[key]), hcd[key], ds[key], du[key], bp[key] + 0, bu[key] + 0, gg[key], pp[key], gs[key], ps[key], psc[key] \
      > (sortie "/r_comb2_par_build.tsv")
  }
  # (b) gate 2 par film : levier seul (X - reference) et marginal (full - full-X)
  print "levier", "film", "build", "seul_d_sains", "seul_d_utiles_sains", "seul_perdus_sains", "marginal_d_sains", \
    "marginal_d_utiles_sains", "verdict_seul", "verdict_marginal" > (sortie "/r_comb2_gate2_par_film.tsv")
  nl = split("L8 L2 LM L3a L6a L6b L4a LS L1a LP L7 L9 WO L6b-flock L1a-ref", L, " ")
  for (i = 1; i <= nl; i++) for (f in films) {
    x = L[i]
    if (!((f, x) in F)) continue
    sd = S[f, x] - S[f, "reference"]; su = US[f, x] - US[f, "reference"]
    md = "-"; mu = "-"; vm = "-"
    if ((f, "full-" x) in F) { md = S[f, "full"] - S[f, "full-" x]; mu = US[f, "full"] - US[f, "full-" x]; vm = (md < 0 || mu < 0) ? "BAISSE" : "tenu" }
    vs = (sd < 0 || su < 0) ? "BAISSE" : "tenu"
    print x, f, build[f], sd, su, PS[f, x], md, mu, vs, vm > (sortie "/r_comb2_gate2_par_film.tsv")
  }
  # (c) la combinaison complete par film
  print "film", "build", "ref_sains", "full_sains", "d_sains", "ref_utiles_sains", "full_utiles_sains", "d_utiles_sains", \
    "fixe", "pct_ref", "pct_full", "perdus_sains_bruts", "perdus_sains_vers_contredit", "verdict" > (sortie "/r_comb2_par_film.tsv")
  for (f in films) {
    print f, build[f], S[f, "reference"], S[f, "full"], S[f, "full"] - S[f, "reference"], US[f, "reference"], US[f, "full"], \
      US[f, "full"] - US[f, "reference"], fixe[f], pct(US[f, "reference"], fixe[f]), pct(US[f, "full"], fixe[f]), PS[f, "full"], \
      PSC[f, "full"], (S[f, "full"] < S[f, "reference"] || US[f, "full"] < US[f, "reference"]) ? "BAISSE" : "tenu" \
      > (sortie "/r_comb2_par_film.tsv")
  }
}
