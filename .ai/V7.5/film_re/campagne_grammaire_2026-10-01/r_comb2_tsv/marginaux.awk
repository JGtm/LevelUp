# marginaux.awk r_comb2_par_build.tsv : seul (X - reference), marginal (full - full-X), interaction.
BEGIN { FS = OFS = "\t" }
FNR == 1 { next }
{ S[$1, $2] = $5; US[$1, $2] = $7; FX[$2] = $9; grp[$2] = 1; UF[$1,$2]=$6; FE[$1,$2]=$4 }
END {
  print "levier", "groupe", "seul_sains", "seul_utiles_sains", "marginal_sains", "marginal_utiles_sains", \
    "interaction_sains", "interaction_utiles_sains", "points_fixe_marginal", "points_fixe_seul", "marginal_fermes_bruts", "marginal_utiles_bruts"
  n = split("L8 L2 LM L3a L6a L6b L4a LS L1a LP L7 L9", L, " ")
  for (g in grp) for (i = 1; i <= n; i++) {
    x = L[i]
    ss = S[x, g] - S["reference", g]; su = US[x, g] - US["reference", g]
    ms = S["full", g] - S["full-" x, g]; mu = US["full", g] - US["full-" x, g]
    print x, g, ss, su, ms, mu, ms - ss, mu - su, sprintf("%.2f", 100 * mu / FX[g]), sprintf("%.2f", 100 * su / FX[g]), \
      FE["full", g] - FE["full-" x, g], UF["full", g] - UF["full-" x, g]
  }
}
