#!/bin/bash
# replis.sh <log> : par film, declenchements de chaque repli (journal « rejeu : repli declenche »).
grep 'repli declenche' "$1" | sed -E 's/.*match_id=([0-9a-f]{8})[^ ]* repli=([^ ]+) declenchements=([0-9]+).*/\1\t\2\t\3/' | sort -u
