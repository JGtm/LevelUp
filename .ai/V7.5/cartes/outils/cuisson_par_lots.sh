#!/bin/bash
# CUISSON DES FONDS FORGE PAR LOTS — REPRENABLE, ET EXCLUSIVE.
#
# Quatre problemes rencontres le 2026-08-26/27, quatre reponses :
#
# 1. LA MEMOIRE. En une seule passe, le temps par carte monte de 137 s a 236 s et le process
#    tient 15 Go ; la machine tombe a 0,1 Go libre et pagine. L'index des modules s'accumule
#    d'une carte a l'autre (chaque carte Forge ouvre forge_objects, les globals et son canevas,
#    jusqu'a 600 Mo). Preuve : a l'arret du process, la RAM libre repasse a 16,2 Go.
#    -> UN PROCESS PAR LOT DE 3 CARTES, memoire rendue a chaque lot.
#
# 2. LA REPRISE. Une interruption ne doit rien faire recommencer.
#    -> L'etat est LU SUR LE DISQUE a chaque tour, jamais tenu par un compteur. Il survit a
#       l'arret du script, a celui de la machine et a la perte du journal.
#
# 3. LE CRITERE EST LE SIDECAR, PAS L'IMAGE. mapfond-build ecrit le PNG, puis seulement s'il a
#    reussi le .json a cote (cmd/mapfond-build/cuisson.go : ecritPNG puis ecritSidecar). Un
#    process tue entre les deux laisse un PNG frais et un json perime : la carte est REPRISE.
#    Un critere sur le PNG aurait publie une image tronquee en la croyant faite.
#
# 4. L'EXCLUSION MUTUELLE. Le pire incident de la nuit : arreter la tache de fond tuait le
#    processus enveloppe, PAS la boucle. Trois boucles ont tourne en meme temps sur la meme
#    file, se battant pour la memoire (0,4 Go libre) et ecrivant le meme PNG a deux. Une heure
#    pour une seule carte. -> VERROU par mkdir (atomique) + refus si un mapfond-build tourne
#    deja + trap qui tue l'enfant et libere le verrou en sortant.
#
# Usage : bash cuisson_par_lots.sh <dossier_scratchpad>
# Relancer la meme commande apres une interruption reprend exactement ou on s'etait arrete.
set -u
SP="$1"
RACINE=/c/Users/Guillaume/Projects/LevelUp
FONDS="$RACINE/data/titles/halo_infinite/reference/map_backgrounds"
JALON="$SP/jalon_campagne"
VERROU="$SP/cuisson.verrou"
# Taille du lot : 3 est la valeur mesuree (memoire rendue a chaque lot). Abaissable par
# l environnement quand la machine est chargee — une campagne de 55 cartes se paie en RAM.
TAILLE_LOT=${TAILLE_LOT:-3}
cd "$RACINE" || exit 1

[ -f "$JALON" ] || { echo "jalon de campagne absent : $JALON"; exit 1; }
[ -f "$SP/tous_forge.txt" ] || { echo "liste des cartes absente"; exit 1; }
touch "$SP/echecs.txt"

# VERROU. mkdir echoue si le dossier existe : c'est atomique, contrairement a un test de
# presence suivi d'une creation.
if ! mkdir "$VERROU" 2>/dev/null; then
  pid=$(cat "$VERROU/pid" 2>/dev/null || echo "?")
  if kill -0 "$pid" 2>/dev/null; then
    echo "!!! une cuisson tourne deja (pid $pid) — rien a faire, elle reprendra seule"
    exit 3
  fi
  echo "verrou orphelin (pid $pid mort), on le reprend"
fi
echo $$ > "$VERROU/pid"

enfant=""
menage() {
  [ -n "$enfant" ] && kill "$enfant" 2>/dev/null
  rm -rf "$VERROU"
}
trap menage EXIT INT TERM

if pgrep -f 'mapfond-build' >/dev/null 2>&1; then
  echo "!!! un mapfond-build tourne hors de ce script — arret pour ne pas se battre avec lui"
  exit 4
fi

restantes() {
  find "$FONDS" -name '*.json' -newer "$JALON" \
    | sed 's#.*/##' | sed 's#\.json$##' | sort > "$SP/faites.txt"
  cat "$SP/faites.txt" "$SP/echecs.txt" | sort -u > "$SP/hors_file.txt"
  grep -vxF -f "$SP/hors_file.txt" "$SP/tous_forge.txt"
}

while :; do
  mapfile -t RESTE < <(restantes)
  n=${#RESTE[@]}
  if [ "$n" -eq 0 ]; then
    echo "=== $(date +%H:%M:%S) TERMINE — $(wc -l < "$SP/faites.txt") fonds cuits, $(wc -l < "$SP/echecs.txt") en echec"
    break
  fi
  paquet=$(printf "%s," "${RESTE[@]:0:$TAILLE_LOT}"); paquet=${paquet%,}
  echo "=== $(date +%H:%M:%S) reste $n — lot : $paquet"
  "$SP/mapfond-build.exe" --natives=false --forge --style encre --maps "$paquet" > "$SP/lot_courant.log" 2>&1 &
  enfant=$!
  wait "$enfant"; code=$?
  enfant=""
  grep -aE 'fond de carte|Forge cuite|err=|ERROR' "$SP/lot_courant.log"
  # FILET ANTI-BOUCLE — COMPTEUR DE TENTATIVES, PAS CODE DE SORTIE.
  #
  # Version precedente : n'ecarter un lot que si le binaire sortait avec le code 0, pour ne
  # pas classer en echec un process tue de l'exterieur. Or mapfond-build sort en ERREUR quand
  # une carte echoue : le filet ne se declenchait donc JAMAIS sur le seul cas qu'il devait
  # couvrir — une carte definitivement incuisable — et ne couvrait que le cas « le binaire
  # reussit sans rien faire », qui n'arrive pas. Le script a rejoue le meme lot pendant SEPT
  # HEURES la nuit du 26 au 27/08 : 135 000 lignes d'erreur, 32 Mo de journal, zero carte.
  #
  # Desormais le critere ne regarde plus le code de sortie mais le NOMBRE DE PASSAGES : trois
  # tentatives sans production et la carte sort de la file. Un process tue de l'exterieur
  # coute au pire deux tentatives de plus, ce qui etait le risque a couvrir.
  mapfile -t APRES < <(restantes)
  if [ "${#APRES[@]}" -eq "$n" ]; then
    echo "!!! $(date +%H:%M:%S) lot sans production (code $code) : $paquet"
    for c in "${RESTE[@]:0:$TAILLE_LOT}"; do
      echo "$c" >> "$SP/tentatives.txt"
      essais=$(grep -cxF "$c" "$SP/tentatives.txt")
      if [ "$essais" -ge 3 ]; then
        echo "!!! $c ecartee apres $essais tentatives"
        echo "$c" >> "$SP/echecs.txt"
      fi
    done
  fi
done
