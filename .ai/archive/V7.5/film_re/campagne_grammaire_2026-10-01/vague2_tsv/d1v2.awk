# d1v2.awk denominateurs.tsv base tete LS LP LN (fermeture_denominateurs.tsv) : indicateur D1.
# fixe_ret = max(fixe consolide, lus base, LS, tete) ; fixe_large = max(fixe_ret, lus LP, LN).
BEGIN{FS=OFS="\t"}
FNR==1{fi++; next}
fi==1{fx0[$1]=$3; fr[$1]=$3; fl[$1]=$3; b[$1]=$2; next}
{lus=$3}
fi<=4{if(lus>fr[$1]){fr[$1]=lus; sr[$1]=fi}}
{if(lus>fl[$1]){fl[$1]=lus; sl[$1]=fi}}
fi==2{bl[$1]=$3; bf[$1]=$5}
fi==3{tl[$1]=$3; tf[$1]=$5}
END{nm[2]="base";nm[3]="tete";nm[4]="LS";nm[5]="LP";nm[6]="LN"
 print "film","build","fixe_consolide","fixe_vague","source","fixe_large","source_large","utiles_sains_base","utiles_sains_tete","fixe_base","fixe_tete","large_base","large_tete","var_base","var_tete"
 for(f in tl){ if(!(f in b)) continue
  printf "%s\t%s\t%d\t%d\t%s\t%d\t%s\t%d\t%d\t%.1f%%\t%.1f%%\t%.1f%%\t%.1f%%\t%.1f%%\t%.1f%%\n", f,b[f],fx0[f],fr[f],(f in sr)?nm[sr[f]]:"consolide",fl[f],(f in sl)?nm[sl[f]]:"consolide",bf[f],tf[f],100*bf[f]/fr[f],100*tf[f]/fr[f],100*bf[f]/fl[f],100*tf[f]/fl[f],100*bf[f]/bl[f],100*tf[f]/tl[f]
  n=split(b[f]" corpus",ks," "); for(i=1;i<=n;i++){B=ks[i]; F0[B]+=fx0[f]; FR[B]+=fr[f]; FL[B]+=fl[f]; BF[B]+=bf[f]; TF[B]+=tf[f]; BL[B]+=bl[f]; TL[B]+=tl[f]}}
 for(B in FR) printf "AGG\t%s\tfixe_consolide=%d\tfixe_vague=%d\tfixe_large=%d\tsains %d -> %d\tfixe %.1f%% -> %.1f%%\tlarge %.1f%% -> %.1f%%\tvariable %.1f%% -> %.1f%%\n", B, F0[B], FR[B], FL[B], BF[B], TF[B], 100*BF[B]/FR[B], 100*TF[B]/FR[B], 100*BF[B]/FL[B], 100*TF[B]/FL[B], 100*BF[B]/BL[B], 100*TF[B]/TL[B]}
