#!/bin/bash
# r_comp_hom_recension.sh — chantier comp, R-HOM (2026-10-02). Recensement STATIQUE, lecture seule Ghidra :
# nom -> chaine ASCII -> accesseur (xref DATA) -> tables (xrefs DATA vers l accesseur) -> lecteur.
# Entree : un nom de composant par ligne (stdin). Sortie TSV : nom, chaine, accesseur, table, vtable[0x00],
# vtable[0x18] (ecrivain), vtable[0x28], lecteur. Usage : bash r_comp_hom_recension.sh < noms.txt > tables.tsv
G=http://127.0.0.1:8089
le64() { # hex string (16 chars) little-endian -> 0x...
  local h=$1 out=""; for i in 14 12 10 8 6 4 2 0; do out="$out${h:$i:2}"; done; echo "0x$(echo $out | sed 's/^0*//')"; }
lire() { curl -s -m 60 "$G/read_memory?address=$1&length=$2" | sed -n 's/.*"hex":"\([0-9a-f]*\)".*/\1/p'; }
while read -r nom; do
  enc=$(printf '%s' "$nom" | sed 's/[.]/\./g' | jq -sRr @uri 2>/dev/null)
  [ -z "$enc" ] && enc=$(printf '%s' "$nom")
  res=$(curl -s -m 120 "$G/search_strings?search_term=%5E${enc}%24&min_length=4&limit=20")
  addrs=$(echo "$res" | grep -o '"address":"[0-9a-f]*","value":"[^"]*"' | awk -F'"' -v n="$nom" '$8==n{print $4}')
  [ -z "$addrs" ] && { printf '%s\t-\t-\t-\t-\t-\t-\t-\n' "$nom"; continue; }
  for sa in $addrs; do
    accs=$(curl -s -m 60 "$G/get_xrefs_to?address=0x$sa&limit=50" | awk '/^From/{print $2"|"$NF}')
    [ -z "$accs" ] && { printf '%s\t%s\t-\t-\t-\t-\t-\t-\n' "$nom" "$sa"; continue; }
    for ax in $accs; do
      acc=${ax%%|*}; kind=${ax##*|}
      tabs=$(curl -s -m 60 "$G/get_xrefs_to?address=0x$acc&limit=50" | awk '/^From/{print $2}')
      [ -z "$tabs" ] && { printf '%s\t%s\t%s%s\t-\t-\t-\t-\t-\n' "$nom" "$sa" "$acc" "$kind" ; continue; }
      for T in $tabs; do
        Tm8=$(printf '0x%x' $((0x$T - 8)))
        h=$(lire $Tm8 56)
        lvl=$(le64 ${h:0:16}); q3=$(le64 ${h:32:16}); q5=$(le64 ${h:48:16}); q6=$(le64 ${h:96:16})
        q5b=$(le64 ${h:80:16})
        # h: q0=0..15 (T-8), q1=16..31 (T), q2=32..47 (T+8), q3=48..63 (T+0x10), q4=64..79, q5=80..95 (T+0x20), q6=96..111 (T+0x28)
        lvl=$(le64 ${h:0:16}); ecr=$(le64 ${h:48:16}); v28=$(le64 ${h:80:16}); v30=$(le64 ${h:96:16})
        lect=$v28; [ "$v28" = "0x14076ce9c" ] && lect=$v30
        printf '%s\t%s\t%s%s\t0x%s\t%s\t%s\t%s\t%s\n' "$nom" "$sa" "$acc" "$kind" "$T" "$lvl" "$ecr" "$v28" "$lect"
      done
    done
  done
done
