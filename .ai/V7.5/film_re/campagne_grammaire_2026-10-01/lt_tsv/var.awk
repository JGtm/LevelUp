# var.awk base_variante autre_variante < lt_paquets.tsv : par film, sains et utiles sains, pertes et gains
BEGIN{FS=OFS="\t"}
NR==1{next}
$2==A{k=$1 SUBSEP $3 SUBSEP $4; sa[k]=($6=="true"); ua[k]=$7; if(sa[k]){SA[$1]++;UA[$1]+=$7}; F[$1]=1}
$2==B{k=$1 SUBSEP $3 SUBSEP $4; sb[k]=($6=="true"); ub[k]=$7; bb[k]=($5=="true"); if(sb[k]){SB[$1]++;UB[$1]+=$7}}
END{print "film","sains_"A,"sains_"B,"net","perdus","dont_contredits","gagnes","utiles_"A,"utiles_"B,"net_utiles"
 for(k in sa){split(k,p,SUBSEP); f=p[1]; if(sa[k]&&!sb[k]){per[f]++; if(bb[k])pc[f]++; print "PERDU",f,p[2]":"p[3],(bb[k]?"contredit":"non_ferme") > "/dev/stderr"}; if(!sa[k]&&sb[k])g[f]++}
 for(f in F){print f,SA[f]+0,SB[f]+0,SB[f]-SA[f],per[f]+0,pc[f]+0,g[f]+0,UA[f]+0,UB[f]+0,UB[f]-UA[f]; T1+=SA[f];T2+=SB[f];T3+=per[f];T4+=pc[f];T5+=g[f];T6+=UA[f];T7+=UB[f]}
 print "corpus",T1,T2,T2-T1,T3,T4,T5,T6,T7,T7-T6}
