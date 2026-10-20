BEGIN{FS=OFS="\t"}
FNR==1{next}
NR==FNR{k=$1 SUBSEP $2 SUBSEP $3; s=($7=="true" && $9=="-"); avS[k]=s; avF[k]=($7=="true"); avL[k]=$4; if(s){sa[$1]++; ua[$1]+=$11}; next}
{k=$1 SUBSEP $2 SUBSEP $3; p=($7=="true"); films[$1]=1
 if(p){sp[$1]++; up[$1]+=$11}
 if(avS[k] && !p){perdu[$1]++; if($6=="true") pc[$1]++; else pn[$1]++; print "SAIN_PERDU", $1, $2":"$3, avL[k]"->"$4, $10 > "/dev/stderr"}
 if(!avS[k] && p){gagne[$1]++}}
END{print "film","sains_avant","sains_apres","net","sains_perdus","dont_devenus_contredits","dont_devenus_non_fermes","gagnes","utiles_sains_avant","utiles_sains_apres","net_utiles"
 for(f in films) print f,sa[f]+0,sp[f]+0,sp[f]-sa[f],perdu[f]+0,pc[f]+0,pn[f]+0,gagne[f]+0,ua[f]+0,up[f]+0,up[f]-ua[f]}
