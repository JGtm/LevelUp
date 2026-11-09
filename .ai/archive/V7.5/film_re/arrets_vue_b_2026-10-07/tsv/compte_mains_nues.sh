#!/usr/bin/env bash
# compte_mains_nues.sh — compte, artefact par artefact, les changements d arme publies qui portent
# l objet « mains nues » (famille 00007ca9), en LECTURE SEULE (jq). Correctif D9.
#
# Usage : compte_mains_nues.sh <artefact.json>...
# Sortie TSV : id, lacher_depuis_mn, echange_depuis_mn, echange_vers_mn, prise_mn, autre_champ_mn,
#              lachers, echanges, prises, unarmedGrants, schemaVersion
# `autre_champ_mn` : toute autre chaine du document egale a 00007ca9 (hors weaponChanges).
set -u
for f in "$@"; do
	id=$(basename "$f"); id=${id%%[-.]*}
	jq -r --arg id "$id" '
	  def u: . == "00007ca9";
	  (.weaponChanges // []) as $w |
	  [$id,
	   ($w | map(select(.kind == "dropped" and ((.from | u) or (.w | u)))) | length),
	   ($w | map(select(.kind == "swapped" and (.from | u))) | length),
	   ($w | map(select(.kind == "swapped" and (.w | u))) | length),
	   ($w | map(select(.kind == "taken" and ((.w | u) or (.from | u)))) | length),
	   ([del(.weaponChanges) | .. | strings | select(ascii_downcase | test("^(0x)?00007ca9"))] | length),
	   ($w | map(select(.kind == "dropped")) | length),
	   ($w | map(select(.kind == "swapped")) | length),
	   ($w | map(select(.kind == "taken")) | length),
	   (.coverage.weaponChanges.unarmedGrants // 0),
	   .schemaVersion] | @tsv' "$f"
done
