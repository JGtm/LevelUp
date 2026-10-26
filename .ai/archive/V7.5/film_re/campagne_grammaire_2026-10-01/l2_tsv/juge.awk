# juge.awk avant apres : par film, au bit / sains / contredits, gagnes et perdus, part de gains factices.
# colonnes : 1 film 2 chunk 3 paquet 4 liste 5 sortie 6 ferme_au_bit 7 ferme 8 premiere 9 regles 10 cause 11 utiles_lus
BEGIN{FS=OFS="\t"}
FNR==1{next}
NR==FNR{k=$1 SUBSEP $2 SUBSEP $3; AB[k]=($6=="true"); AS[k]=($7=="true"&&$9=="-"); AU[k]=$11; next}
{k=$1 SUBSEP $2 SUBSEP $3; f=$1; F[f]=1; b=($6=="true"); s=($7=="true"&&$9=="-")
 if(AB[k])ab[f]++; if(b)pb[f]++
 if(AS[k])as[f]++; if(s)ps[f]++
 if(!AB[k]&&b){gb[f]++; if(!s){gf[f]++; print "GAIN_FACTICE",f,$2":"$3,$8 > "/dev/stderr"}}
 if(!AS[k]&&s){gs[f]++; ugs[f]+=$11}
 if(AB[k]&&!b){pbr[f]++; print "PERDU_AU_BIT",f,$2":"$3,(AS[k]?"sain":"contredit"),$10 > "/dev/stderr"}
 if(AS[k]&&!s){psn[f]++}
 if(AB[k]&&b&&!AS[k]&&s){req[f]++}
 if(AS[k]&&b&&!s){reqc[f]++}
}
END{print "film","au_bit_avant","au_bit_apres","gagnes_au_bit","dont_contredits(factices)","part_factice_%","perdus_au_bit","sains_avant","sains_apres","sains_gagnes","sains_perdus","sains_devenus_contredits","contredits_devenus_sains"
 for(f in F){g=gb[f]+0; printf "%s\t%d\t%d\t%d\t%d\t%.1f\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n",f,ab[f],pb[f],g,gf[f],(g?100*gf[f]/g:0),pbr[f],as[f],ps[f],gs[f],psn[f],reqc[f],req[f]}}
