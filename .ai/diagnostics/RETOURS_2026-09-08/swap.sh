#!/usr/bin/env bash
# Pour chaque artefact : compare le SPAWN de la premiere vie de chaque joueur
# au centroide (mediane) de son camp, tel que le donne le scoreboard.
REP=data/cache/replays/halo_infinite
TEAMS="$1"
printf "match\tjoueurs\tmalplaces\tsepX\tsepY\n"
for f in $REP/*.json; do
  case "$f" in *derived*) continue;; esac
  b=$(basename "$f" .json)
  full=$(awk -F'\t' -v p="$b" '$1 ~ "^"p {print $2"\t"$3}' "$TEAMS")
  [ -z "$full" ] && continue
  echo "$full" > /tmp/_t.tsv
  # spawn de la PREMIERE vie de chaque xuid
  jq -r '[.tracks[]? | select(.xuid != null)] | group_by(.xuid)
         | map({xuid: .[0].xuid, t0: (map(.points[0].t) | min)} + ( . as $g | {p: ($g | sort_by(.points[0].t) | .[0].points[0])} ))
         | .[] | select(.t0 <= 120) | [.xuid, (.p.x|tostring), (.p.y|tostring)] | @tsv' "$f" 2>/dev/null > /tmp/_s.tsv
  [ ! -s /tmp/_s.tsv ] && continue
  join -t$'\t' -1 1 -2 1 <(sort -k1,1 /tmp/_s.tsv) <(sort -k1,1 /tmp/_t.tsv) 2>/dev/null > /tmp/_j.tsv
  awk -F'\t' -v m="$b" '
    { x[NR]=$2; y[NR]=$3; t[NR]=$4; n++ }
    END{
      if (n < 6) exit
      # medianes par camp
      for (k=0;k<2;k++){ c=0; delete ax; delete ay
        for(i=1;i<=n;i++) if(t[i]==k){ c++; ax[c]=x[i]; ay[c]=y[i] }
        if(c<2) exit
        asort(ax); asort(ay)
        mx[k]=ax[int((c+1)/2)]; my[k]=ay[int((c+1)/2)]; cnt[k]=c
      }
      sepx = mx[0]-mx[1]; if (sepx<0) sepx=-sepx
      sepy = my[0]-my[1]; if (sepy<0) sepy=-sepy
      # separation trop faible : test non concluant sur cette carte
      if (sepx < 8 && sepy < 8) exit
      bad=0
      for(i=1;i<=n;i++){
        own=t[i]; oth=1-own
        d_own=(x[i]-mx[own])^2+(y[i]-my[own])^2
        d_oth=(x[i]-mx[oth])^2+(y[i]-my[oth])^2
        if (d_oth < d_own) bad++
      }
      printf "%s\t%d\t%d\t%.1f\t%.1f\n", m, n, bad, sepx, sepy
    }' /tmp/_j.tsv
done
