#!/usr/bin/env bash
set -e
SP=$(curl -s --max-time 15 "http://127.0.0.1:8000/api/v1/players/Chocoboflor/pages/palmares/season-pass")

echo "=== 1. Taille réponse ==="
echo "$SP" | wc -c

echo "=== 2. Passes distincts ==="
echo "$SP" | grep -o '"reward_track_path":"[^"]*"' | sort -u | wc -l

echo "=== 3. Noms des passes ==="
echo "$SP" | grep -o '"reward_track_path":"[^"]*"' | sort -u

echo "=== 4. Total tiers ==="
echo "$SP" | grep -o '"tier":[0-9]*' | wc -l

echo "=== 5. Tiers avec image battlepass ==="
echo "$SP" | grep -o '"image_url":"[^"]*battlepass[^"]*"' | wc -l

echo "=== 6. Tiers sans image (vide) ==="
echo "$SP" | grep -o '"image_url":""' | wc -l

echo "=== 7. Tiers image null ==="
echo "$SP" | grep -o '"image_url":null' | wc -l

echo "=== 8. Items + tracks en DB (via SQL) ==="
find /c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/investigation/battlepass -name "*.json" | wc -l
find /c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/investigation/battlepass -name "*.json" -path "*/items/*" | wc -l
find /c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/investigation/battlepass -name "*.json" -path "*/tracks/*" | wc -l

echo "=== 9. Fichiers images cache battlepass ==="
find /c/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/battlepass_assets -type f 2>/dev/null | wc -l

echo "=== 10. Test HTTP image battlepass ==="
IMGPATH=$(echo "$SP" | grep -o '"image_url":"[^"]*battlepass[^"]*"' | head -1 | sed 's/"image_url":"//;s/"//')
echo "Path: $IMGPATH"
curl -s -o /dev/null -w "HTTP %{http_code} size=%{size_download}\n" "http://127.0.0.1:8000${IMGPATH}"
