# gate2.awk avant.tsv apres.tsv (fermeture_paquets.tsv) : par film, le gate 2 de la campagne.
# sain = ferme (col 7, definition de L0 : au bit pres ET aucune regle de l ecrivain contredite).
BEGIN{FS=OFS="\t"}
FNR==1{next}
NR==FNR{k=$1 SUBSEP $2 SUBSEP $3; s[k]=($7=="true"); b[k]=($6=="true"); u[k]=$11; if(s[k]){sa[$1]++; ua[$1]+=$11}; next}
{k=$1 SUBSEP $2 SUBSEP $3; f=$1; F[f]=1; p=($7=="true"); pb=($6=="true")
 if(p){sp[f]++; up[f]+=$11}
 if(s[k] && !p){per[f]++; uper[f]+=u[k]; if(pb) pc[f]++; else pn[f]++; print "SAIN_PERDU", f, $2":"$3, (pb?"devenu_contredit":"devenu_non_ferme"), $8, $10 > "/dev/stderr"}
 if(!s[k] && p){g[f]++; ug[f]+=$11}
 if(!b[k] && pb){gb[f]++; if(!p) gbf[f]++}
 if(b[k] && !pb){pbit[f]++}
}
END{print "film","sains_avant","sains_apres","net","sains_perdus","dont_contredits","dont_non_fermes","gagnes_sains","gains_au_bit","dont_factices","part_factice","pertes_au_bit","utiles_sains_avant","utiles_sains_apres","net_utiles","utiles_sains_perdus_brut"
 for(f in F){pf=(gb[f]>0)?sprintf("%.1f%%",100*gbf[f]/gb[f]):"-"; print f,sa[f]+0,sp[f]+0,sp[f]-sa[f],per[f]+0,pc[f]+0,pn[f]+0,g[f]+0,gb[f]+0,gbf[f]+0,pf,pbit[f]+0,ua[f]+0,up[f]+0,up[f]-ua[f],uper[f]+0
  T[1]+=sa[f];T[2]+=sp[f];T[4]+=per[f];T[5]+=pc[f];T[6]+=pn[f];T[7]+=g[f];T[8]+=gb[f];T[9]+=gbf[f];T[11]+=pbit[f];T[12]+=ua[f];T[13]+=up[f];T[15]+=uper[f]}
 print "corpus",T[1],T[2],T[2]-T[1],T[4],T[5],T[6],T[7],T[8],T[9],sprintf("%.1f%%",100*T[9]/T[8]),T[11],T[12],T[13],T[13]-T[12],T[15]}
