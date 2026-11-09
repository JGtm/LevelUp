# d1.awk denominateurs.tsv base.tsv l8.tsv l3a.tsv l4a.tsv tete.tsv (fermeture_denominateurs.tsv)
# Indicateur D1 : records utiles sains, denominateur fixe consolide RECALCULE (max des marches) et variable.
BEGIN{FS=OFS="\t"}
FNR==1{fi++; next}
fi==1{fixe[$1]=$3; fx0[$1]=$3; b[$1]=$2; next}
{lus=$3; if(lus>fixe[$1]) {fixe[$1]=lus; src[$1]=fi}}
fi==2{bl[$1]=$3; bf[$1]=$5}
fi==6{tl[$1]=$3; tf[$1]=$5}
END{print "film","build","fixe_r_comb2","fixe_vague","source","utiles_sains_base","utiles_sains_tete","fixe_base","fixe_tete","var_base","var_tete"
 for(f in tl){if(!(f in b)) continue; s=(f in src)?("marche "src[f]):"r_comb2"; printf "%s\t%s\t%d\t%d\t%s\t%d\t%d\t%.1f%%\t%.1f%%\t%.1f%%\t%.1f%%\n", f,b[f],fx0[f],fixe[f],s,bf[f],tf[f],100*bf[f]/fixe[f],100*tf[f]/fixe[f],100*bf[f]/bl[f],100*tf[f]/tl[f]
  B=b[f]; FX[B]+=fixe[f]; BF[B]+=bf[f]; TF[B]+=tf[f]; BL[B]+=bl[f]; TL[B]+=tl[f]; FX["corpus"]+=fixe[f]; BF["corpus"]+=bf[f]; TF["corpus"]+=tf[f]; BL["corpus"]+=bl[f]; TL["corpus"]+=tl[f]}
 for(B in FX) printf "AGG\t%s\tfixe=%d\tsains %d -> %d\tfixe %.1f%% -> %.1f%%\tvariable %.1f%% -> %.1f%%\n", B, FX[B], BF[B], TF[B], 100*BF[B]/FX[B], 100*TF[B]/FX[B], 100*BF[B]/BL[B], 100*TF[B]/TL[B]}
