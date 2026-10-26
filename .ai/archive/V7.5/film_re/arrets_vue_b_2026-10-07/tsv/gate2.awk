# usage: awk -F'\t' -v PERDUS=fichier -f gate2.awk base/fermeture_paquets.tsv tete/fermeture_paquets.tsv
FNR==1 {f++; next}
{ k=$1 SUBSEP $2 SUBSEP $3; film[$1]=1
  if (f==1) { bf[k]=$7; bu[k]=$11; if ($7=="true") {sb[$1]++; ub[$1]+=$11} }
  else { tf[k]=$7; tu[k]=$11; tc[k]=$10; if ($7=="true") {st[$1]++; ut[$1]+=$11}
         if ($7=="true" && bf[k]!="true") gagn[$1]++
         if ($7!="true" && bf[k]=="true") { perd[$1]++; uperd[$1]+=bu[k]; if (PERDUS!="") print $1"\t"$2"\t"$3"\t"bu[k]"\t"$10 > PERDUS }
  } }
END { printf "film\tsains_base\tsains_tete\tnet\tperdus\tgagnes\tutiles_base\tutiles_tete\tnet_utiles\tutiles_perdus\tverdict\n"
  bas=0
  for (x in film) { n=st[x]-sb[x]; nu=ut[x]-ub[x]; v=(n<0||nu<0)?"BAISSE":"ok"; if (v=="BAISSE") bas++
    printf "%s\t%d\t%d\t%+d\t%d\t%d\t%d\t%d\t%+d\t%d\t%s\n", x, sb[x], st[x], n, perd[x], gagn[x], ub[x], ut[x], nu, uperd[x], v
    T1+=sb[x];T2+=st[x];T3+=perd[x];T4+=gagn[x];T5+=ub[x];T6+=ut[x];T7+=uperd[x] }
  printf "CORPUS\t%d\t%d\t%+d\t%d\t%d\t%d\t%d\t%+d\t%d\tfilms_en_baisse=%d\n", T1,T2,T2-T1,T3,T4,T5,T6,T6-T5,T7,bas }
