# Requêtes de reproduction — diagnostic 2026-09-08

Toutes les commandes se lancent depuis la racine du worktree. `tmp_diag_q.exe` est le lecteur
DuckDB read-only du dépôt (`apps/go-api/cmd/diag_q`).

`SM` = `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
`SS` = `data/titles/halo_infinite/warehouse/shared_social.duckdb`
`REP` = `data/cache/replays/halo_infinite`

Matchs de référence (7 septembre 2026) :

| clé | match_id |
|---|---|
| Origin CTF | `8bc6074f-d001-428b-8d6a-a755f0925572` |
| Isolation Zones | `81c02726-3f03-4a9e-9c05-3e286460752c` |
| Behemoth (défaites) | `7b0d89c4-a068-4ea2-82b8-8f13ba0311d6`, `f2966f08-f15a-4d9c-aa14-938d733173b7` |

---

## Cause commune — `publishable`

```bash
# Part du parc où AUCUNE ligne n'est publiable
./tmp_diag_q.exe "$SM" "
WITH m AS (SELECT match_id, max(CASE WHEN publishable THEN 1 ELSE 0 END) AS pub
           FROM match_kill_events_latest GROUP BY 1)
SELECT pub, count(*) AS matchs FROM m GROUP BY 1 ORDER BY 1"

# Les 20 derniers matchs, avec leur compte de lignes publiables
./tmp_diag_q.exe "$SM" "
SELECT r.map_name, r.game_variant_name,
       COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS t,
       count(k.id) AS lignes, count(k.id) FILTER (WHERE k.publishable) AS publiables
FROM match_registry r LEFT JOIN match_kill_events_latest k USING(match_id)
GROUP BY 1,2,3, r.match_id ORDER BY t DESC LIMIT 20"
```

## Point 1 — médailles de la frise

```bash
./tmp_diag_q.exe "$SM" "
SELECT xuid, count(*) AS n, min(raw_json) AS ex FROM highlight_events
WHERE match_id='8bc6074f-d001-428b-8d6a-a755f0925572' AND event_type='medal'
GROUP BY 1 ORDER BY 2 DESC"
```

Dans le navigateur, sur la page de rejeu (console) :

```js
[...document.querySelectorAll('span.pointer-events-none.absolute')]
  .filter(s => s.className.includes('-translate-x-1/2'))
  .map(m => ({ anneau: m.className.includes('ring-1'), title: m.title,
               w: m.getBoundingClientRect().width, h: m.getBoundingClientRect().height }))
```

## Point 3 — fiches rognées (mutation)

Console, page de rejeu :

```js
const grid = [...document.querySelectorAll('div')]
  .find(d => d.className.toString().includes('grid h-full min-h-0 gap-2.5'));
const avant = { tpl: getComputedStyle(grid).gridTemplateColumns, cw: grid.clientWidth, sw: grid.scrollWidth };
grid.style.gridTemplateColumns = 'repeat(2, minmax(0, 1fr))';
const apres = { tpl: getComputedStyle(grid).gridTemplateColumns, cw: grid.clientWidth, sw: grid.scrollWidth };
grid.style.gridTemplateColumns = '';
({ avant, apres })
// avant : 279.688px 273.094px  cw 480  sw 563
// apres : 235px 235px          cw 480  sw 480
```

## Point 4 — identités interverties (LA requête décisive)

Origin est une carte CTF symétrique : base `t0` à `x ≈ −22`, base `t1` à `x ≈ +22`. Chaque camp
spawne groupé. Une identité mal attribuée se voit donc à la position de départ de sa vie.

```bash
# Départ de chaque vie sur les 90 premières secondes, trié chronologiquement
jq -r '[.tracks[]? | select(.xuid != null)
        | {xuid, slot, t0:(.points[0].t), x0:(.points[0].x)}]
       | map(select(.t0 <= 900)) | sort_by(.t0)' $REP/8bc6074f.json

# Les deux joueurs intervertis, vie par vie
jq -r '[.tracks[]? | select(.xuid=="2535469190789936" or .xuid=="2535452521259564")
        | {xuid, slot, t0:(.points[0].t), x0:(.points[0].x), endFrame}] | sort_by(.t0)' \
  $REP/8bc6074f.json
# 2535452521259564 StevenW5318 (t1) : slot 516 @ x=-22.51  puis slot 525 @ x=+13.74
# 2535469190789936 Chocoboflor  (t0) : slot 518 @ x=+22.62  puis slot 526 @ x=-13.09
# -> vies 1 et 2 du mauvais cote, vie 3 correcte. Bascule vers l'image ~874.
```

Le camp de référence, à comparer :

```bash
./tmp_diag_q.exe "$SM" "
SELECT xuid, team_id, kills, deaths, present_at_beginning, joined_in_progress
FROM match_participants WHERE match_id='8bc6074f-d001-428b-8d6a-a755f0925572'
ORDER BY team_id, kills DESC"
```

Le décodeur le signale — c'est la condition même de `publishable = false` :

```bash
./tmp_diag_q.exe "$SM" "
SELECT count(*) AS lignes, count(*) FILTER (WHERE publishable) AS publiables
FROM match_kill_events_latest WHERE match_id='8bc6074f-d001-428b-8d6a-a755f0925572'"
# 81 / 0 -> BijectionMargin == 0 : « au moins deux joueurs sont interchangeables »

# … mais la santé du pont, elle, se déclare saine
jq -r '.coverage.bridge' $REP/8bc6074f.json
# indexDisagreements: 0, slotCollisions: 0
```

**Contre-preuves qui NE valent PAS** (elles m'ont d'abord induit en erreur) :

```bash
# 1) « aucun frag allié » : se lit sur des lignes que le produit ne publie jamais ici
./tmp_diag_q.exe "$SM" "
WITH p AS (SELECT xuid, team_id FROM match_participants
           WHERE match_id='8bc6074f-d001-428b-8d6a-a755f0925572')
SELECT pk.team_id AS kt, pv.team_id AS vt, count(*) AS n
FROM match_kill_events_latest k
LEFT JOIN p pk ON pk.xuid=k.feed_killer_xuid
LEFT JOIN p pv ON pv.xuid=k.victim_xuid
WHERE k.match_id='8bc6074f-d001-428b-8d6a-a755f0925572' GROUP BY 1,2"

# 2) « le scoreboard API est correct » : il vient de l'API Halo, pas du film — il ne peut pas
#    révéler un défaut du film
curl -s "http://127.0.0.1:8000/api/v1/players/JGtm/matches/8bc6074f-d001-428b-8d6a-a755f0925572" \
  | jq -c '[.team_tab.scoreboard[] | {gamertag, team_side}]'
```

## Point 6 — artefacts sans balayage véhicules

```bash
for f in $REP/*.json; do case "$f" in *derived*) continue;; esac
  c=$(jq -r 'if (.coverage|has("vehicles")) then 1 else 0 end' $f)
  [ "$c" = "0" ] && echo "$(basename $f .json) schema=$(jq -r .schemaVersion $f)"
done
# 15 artefacts, TOUS au schema 38 (les 49 autres sont au 48)

# Comparaison : un Behemoth recuit a bien le bloc
jq -r '.coverage.vehicles' $REP/1cd3848a.json
jq -r '.coverage.vehicles // "ABSENTE"' $REP/7b0d89c4.json
```

## Point 5 — fond de carte Isolation

```bash
# L'API sert bien le fond
curl -s "http://127.0.0.1:8000/api/v1/players/JGtm/matches/81c02726-3f03-4a9e-9c05-3e286460752c/replay/background" | jq .calibration
curl -s -o /dev/null -w "%{http_code} %{size_download}\n" \
  "http://127.0.0.1:8000/api/v1/players/JGtm/matches/81c02726-3f03-4a9e-9c05-3e286460752c/replay/background.png"

# Les bornes brutes du document débordent l'emprise de l'image
jq -r '.bounds' $REP/81c02726.json

# … alors que 99 % des positions tiennent dedans
jq -r '[.tracks[].points[].x] | sort
       | {p0:.[0], p1:.[(length*0.01|floor)], p50:.[(length*0.5|floor)],
          p99:.[(length*0.99|floor)], max:.[length-1]}' $REP/81c02726.json
jq -r '[.tracks[].points[].y] | sort
       | {p0:.[0], p1:.[(length*0.01|floor)], p50:.[(length*0.5|floor)],
          p99:.[(length*0.99|floor)], max:.[length-1]}' $REP/81c02726.json
```

## Point 8 — tirs de véhicule jamais publiés

```bash
# Balayage du parc : shots / shotsNoRide / shotsVehicleWeapon
for f in $REP/*.json; do case "$f" in *derived*) continue;; esac
  b=$(basename $f .json)
  s=$(jq -r '.coverage.vehicles.shots // "-"' $f)
  [ "$s" != "-" ] && echo "$b shots=$s noRide=$(jq -r '.coverage.vehicles.shotsNoRide' $f) vehWeapon=$(jq -r '.coverage.vehicles.shotsVehicleWeapon' $f)"
done

# Aucun tir ne porte la clé `v` (Shot.Vehicle)
jq -rc '[.shots[]? | keys] | add | unique' $REP/81c02726.json
```

## Point 10b — tirs pendant le portage du drapeau

```bash
jq -r '
 ([.tracks[]? | {slot, xuid}] | map(select(.xuid != null))
   | INDEX(.slot|tostring) | with_entries(.value = .value.xuid)) as $own
 | [.flagCarries[]?.spans[]? | select(.state=="carried" and .xuid != null)] as $carries
 | [ .shots[]? | . + {owner: ($own[(.slot|tostring)] // null)} ] as $shots
 | [ $shots[] | select(.owner != null) | . as $s
     | ($carries | map(select(.xuid == $s.owner and .t0 <= $s.t and $s.t <= .t1)) | .[0]) as $c
     | select($c != null)
     | {dansSpan: ($s.t - $c.t0), restant: ($c.t1 - $s.t)} ]
 | { n: length,
     medianeAvantFin: ([.[].restant]|sort|.[length/2|floor]),
     pct20PremieresFrames: (([.[]|select(.dansSpan<=20)]|length)*100/length|floor),
     pct20DernieresFrames: (([.[]|select(.restant<=20)]|length)*100/length|floor) }
' $REP/8bc6074f.json
# → n=124, medianeAvantFin=21, pct20Premieres=0 %, pct20Dernieres=48 %
```

## Point 11 — liaison lancer → projectile

```bash
for m in 8bc6074f 81c02726 7b0d89c4 b1ad85eb f8efc5ca; do
  echo -n "$m : "
  jq -rc '{grenades:(.grenades|length), avecProj:([.grenades[]?|select(.proj!=null)]|length),
           projectiles:(.projectiles|length)}' $REP/$m.json
done

# Taux global du parc
tot=0; lie=0
for f in $REP/*.json; do case "$f" in *derived*) continue;; esac
  r=$(jq -r '[(.grenades|length), ([.grenades[]?|select(.proj!=null)]|length)] | @tsv' $f)
  tot=$((tot+$(echo "$r"|cut -f1))); lie=$((lie+$(echo "$r"|cut -f2)))
done; echo "$lie / $tot"
```

## Point 12 — paires d'assistance

```bash
./tmp_diag_q.exe "$SM" "
SELECT match_id, count(*) AS lignes,
       count(*) FILTER (WHERE publishable) AS publiables,
       count(*) FILTER (WHERE publishable AND assist_known) AS assist_mesurees,
       count(*) FILTER (WHERE publishable AND assist_known
                        AND assist_xuid IS NOT NULL AND feed_killer_xuid IS NOT NULL) AS paires
FROM match_kill_events_latest
WHERE match_id IN ('8bc6074f-d001-428b-8d6a-a755f0925572',
                   '81c02726-3f03-4a9e-9c05-3e286460752c')
GROUP BY 1"
```

## Point 13 — médias sans match

```bash
./tmp_diag_q.exe "$SS" "
SELECT m.player_slug, count(*) AS total,
       count(a.media_file_id) AS avec_match,
       count(*)-count(a.media_file_id) AS sans_match
FROM media_files m
LEFT JOIN media_match_associations_latest a ON a.media_file_id = m.id
GROUP BY 1"

./tmp_diag_q.exe "$SS" "
SELECT date_part('year', m.capture_start_utc) AS annee, count(*) AS sans_match
FROM media_files m
LEFT JOIN media_match_associations_latest a ON a.media_file_id = m.id
WHERE a.media_file_id IS NULL GROUP BY 1 ORDER BY 1"

# Le registre de matchs commence à la sortie du jeu
./tmp_diag_q.exe "$SM" "
SELECT min(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')) AS premier,
       max(COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')) AS dernier
FROM match_registry"
```

## Point 14 — distances

```bash
./tmp_diag_q.exe "$SM" "
SELECT match_id, count(*) AS positions FROM kill_positions_latest
WHERE match_id IN ('8bc6074f-d001-428b-8d6a-a755f0925572',
                   '81c02726-3f03-4a9e-9c05-3e286460752c') GROUP BY 1"

# Tables sœurs vides sur tout le parc
./tmp_diag_q.exe "$SM" "SELECT count(*) FROM match_weapon_hit_distance_latest"
./tmp_diag_q.exe "$SM" "SELECT count(*) FROM weapon_accuracy"
```

## Point 16 — centrage de l'écran de fin

Console, à la fin d'un rejeu :

```js
const ov = [...document.querySelectorAll('div')]
  .find(d => d.className.toString().includes('absolute inset-0 z-10 flex items-center justify-center overflow-hidden'));
const cv = document.querySelector('canvas');
const c = e => Math.round(e.getBoundingClientRect().top + e.getBoundingClientRect().height/2);
({ centreOverlay: c(ov), centreCarte: c(cv) })   // 530 vs 441
```

## Point 21 — plan tactique vide

```bash
# Couverture des positions de kill sur le parc
./tmp_diag_q.exe "$SM" "SELECT count(DISTINCT match_id) FROM kill_positions_latest"   # 413 / 1967
```

Réponse serveur (onglet réseau) : `POST /api/v1/players/JGtm/tactical/{map_id}/raster`
→ `cellules: []`, `bornes.valide: false`, `evenements_localises: 115` sur `evenements_journal: 433`.

Constantes en cause : `analysis/tactical/merge.go:16` (`PlancherMatchsParCellule = 3`) et
`analysis/tactical/grid.go:41` (grille de 0,5 m).
