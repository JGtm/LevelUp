# Mesures — maquette « Sessions », avant / après (2026-10-05)

Base : copie `scratchpad/db/shared_matches_v2.duckdb` (2026-10-05 18:26), lecture seule, outil `diag_q.exe`.
Vues `_latest` uniquement (`match_participants` / `match_registry` ne sont pas append-only). Rien lu sous `data/`.
Scripts : `extract_s.sh` (extraits bruts `ex/<clé>_*.tsv`), `qs.sh` (requête sur le CTE `sc` d'une session et
`side` = camp de chaque participant), `compute_s.js` (modèles de vue -> `vm_s.json`, `vm_s_report.json`),
`build_s.js`, `validate_s.js`, `mes.sh` (relevés ci-dessous).

## 0. Les trois sessions

Découpage de l'app : trou de 2 h ET changement de composition (`domain/sessions.go`, mode `group` par défaut).
Joueur JGtm `2533274823110022`. Madina97294 = `2533274858283686` (`match_participants.gamertag`).
NB : `match_participants.gamertag` n'est rempli que sur une ligne par match ; l'identification passe par le xuid.

| clé | matchs | composition | filmés |
|---|---|---|---|
| s2209 | 22/09 21:23 → 22:29 (Starboard CTF, Curfew, Origin CTF, Solution, Detachment, Shogun, Catalyst) | + Chocoboflor, Madina97294 (même camp sur les 7) | 6 (Detachment sans film) |
| s0709 | 07/09 21:26 → 22:14 (4 Strongholds, 3 CTF) | + XxDaemonGamerxX, Madina97294, Chocoboflor | 7 |
| solo | 22/09 19:27 → 20:19 (6 Super Fiesta Assassin) | aucun ami suivi | 6 |

Le 22/09 au soir compte 15 matchs avec un trou de 64 min : la session d'escouade (7) est découpée par la composition ;
les 6 premiers matchs forment la session solo. Le 07/09, le 8e match (22:24 Domicile, sans XxDaemonGamerxX) ouvre
une autre session.

## 1. Relevés SQL bruts

### s2209 — matchs

```sql
SELECT strftime(start_time_utc AT TIME ZONE 'Europe/Paris','%d/%m %H:%M') t, map_name, regexp_replace(pair_name,' on .*','') md, my_outcome, my_team, team_0_score, team_1_score, match_id IN (SELECT match_id FROM match_usage_films_latest) filme FROM sc ORDER BY start_time_utc
```
```
t	map_name	md	my_outcome	my_team	team_0_score	team_1_score	filme
22/09 21:23	Starboard	Arena:CTF	2	1	0	3	true
22/09 21:36	Curfew	Community:Team Slayer	2	0	50	48	true
22/09 21:46	Origin	Arena:CTF	3	0	1	3	true
22/09 21:59	Solution	Community:Team Slayer	2	1	17	50	true
22/09 22:08	Detachment	Arena:Team Slayer	3	0	45	50	false
22/09 22:19	Shogun	Community:Team Slayer	3	1	50	37	true
22/09 22:29	Catalyst	Arena:Team Slayer	3	0	33	50	true
(7 rows)
```

### s2209 — bonus par camp (témoins Emprise)

```sql
SELECT s, sum(COALESCE(json_extract(u.taken_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.taken_json,'$.powerup_overshield')::INT,0)) bonus_pris, sum(COALESCE(json_extract(u.kept_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.kept_json,'$.powerup_overshield')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_overshield')::INT,0)) bonus_perdus, sum(camo_ms+overshield_ms) effet_ms, sum(camo_kills+overshield_kills) frags_effet, sum(pad_pickups) pad_pickups FROM match_usage_players_latest u JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=u.match_id AND side.xuid=u.xuid GROUP BY 1 ORDER BY 1
```
```
s	bonus_pris	bonus_perdus	effet_ms	frags_effet	pad_pickups
them	8	2	112800	5	51
us	12	2	159100	8	44
(2 rows)
```

### s2209 — prises par niveau et par camp

```sql
SELECT t.tier, COALESCE(side.s,'sans_camp') s, sum(pickups) FROM match_pad_pickups_by_tier_latest t JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=t.match_id AND side.xuid=t.xuid WHERE pickups>0 GROUP BY 1,2 ORDER BY 1,2
```
```
tier	s	sum(pickups)
base	us	2
non_classe	us	5
puissance	them	21
puissance	us	13
terrain	them	30
terrain	us	24
(6 rows)
```

### s2209 — frags aux armes spéciales (feuille)

```sql
SELECT s, sum(power_weapon_kills) pwk, sum(kills) frags FROM match_participants p JOIN sc USING(match_id) JOIN side ON side.match_id=p.match_id AND side.xuid=p.xuid GROUP BY 1 ORDER BY 1
```
```
s	pwk	frags
them	54	342
us	47	362
(2 rows)
```

### s2209 — véhicules

```sql
SELECT count(*) FROM match_vehicle_takes_latest JOIN sc USING(match_id)
```
```
count_star()
0
(1 rows)
```

### s2209 — prises nettes de drapeau

```sql
SELECT s, sum(flag_grabs_raw) brut, sum(flag_grabs_net) net FROM match_flag_grabs_net_latest g JOIN sc USING(match_id) JOIN side ON side.match_id=g.match_id AND side.xuid=g.xuid GROUP BY 1 ORDER BY 1
```
```
s	brut	net
them	30	26
us	26	22
(2 rows)
```

### s2209 — vies (requête modèle, seuil 18 m)

```sql
, lv AS (SELECT l.match_id, l.start_ms, l.end_ms FROM match_lives_latest l WHERE l.xuid='2533274823110022' AND l.end_cause='death' AND l.match_id IN (SELECT match_id FROM sc)), j AS (SELECT lv.*, (SELECT d.nearest_teammate_m FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS near_m, (SELECT d.teammates_visible FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS vis, (SELECT count(*) FROM match_kill_events_latest k WHERE k.match_id=lv.match_id AND k.publishable AND k.feed_killer_xuid='2533274823110022' AND k.time_ms>=lv.start_ms AND k.time_ms<=lv.end_ms) AS frags FROM lv) SELECT CASE WHEN vis IS NULL THEN 'sans_contexte' WHEN vis=0 OR near_m IS NULL THEN 'aucun_coequipier_situe' WHEN near_m<18 THEN 'pres' ELSE 'seul' END cls, count(*) vies, sum(frags) frags FROM j GROUP BY 1 ORDER BY 1
```
```
cls	vies	frags
pres	56	37
seul	18	4
(2 rows)
```

### s0709 — matchs

```sql
SELECT strftime(start_time_utc AT TIME ZONE 'Europe/Paris','%d/%m %H:%M') t, map_name, regexp_replace(pair_name,' on .*','') md, my_outcome, my_team, team_0_score, team_1_score, match_id IN (SELECT match_id FROM match_usage_films_latest) filme FROM sc ORDER BY start_time_utc
```
```
t	map_name	md	my_outcome	my_team	team_0_score	team_1_score	filme
07/09 21:26	Banished Narrows	Arena:Strongholds	3	1	200	82	true
07/09 21:34	Isolation	Arena:Strongholds	3	0	31	200	true
07/09 21:42	Illusion	Arena:Strongholds	2	0	200	183	true
07/09 21:52	Fortress	Arena:Strongholds	3	1	200	89	true
07/09 22:00	Origin	Arena:CTF	3	0	1	3	true
07/09 22:10	Domicile	Arena:CTF	3	1	3	0	true
07/09 22:14	Absolution	Arena:CTF	3	1	3	0	true
(7 rows)
```

### s0709 — bonus par camp (témoins Emprise)

```sql
SELECT s, sum(COALESCE(json_extract(u.taken_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.taken_json,'$.powerup_overshield')::INT,0)) bonus_pris, sum(COALESCE(json_extract(u.kept_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.kept_json,'$.powerup_overshield')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_overshield')::INT,0)) bonus_perdus, sum(camo_ms+overshield_ms) effet_ms, sum(camo_kills+overshield_kills) frags_effet, sum(pad_pickups) pad_pickups FROM match_usage_players_latest u JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=u.match_id AND side.xuid=u.xuid GROUP BY 1 ORDER BY 1
```
```
s	bonus_pris	bonus_perdus	effet_ms	frags_effet	pad_pickups
them	9	1	398200	11	58
us	8	0	456500	13	40
(2 rows)
```

### s0709 — prises par niveau et par camp

```sql
SELECT t.tier, COALESCE(side.s,'sans_camp') s, sum(pickups) FROM match_pad_pickups_by_tier_latest t JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=t.match_id AND side.xuid=t.xuid WHERE pickups>0 GROUP BY 1,2 ORDER BY 1,2
```
```
tier	s	sum(pickups)
base	them	1
base	us	2
non_classe	us	1
puissance	them	29
puissance	us	18
terrain	them	28
terrain	us	19
(7 rows)
```

### s0709 — frags aux armes spéciales (feuille)

```sql
SELECT s, sum(power_weapon_kills) pwk, sum(kills) frags FROM match_participants p JOIN sc USING(match_id) JOIN side ON side.match_id=p.match_id AND side.xuid=p.xuid GROUP BY 1 ORDER BY 1
```
```
s	pwk	frags
them	48	252
us	21	185
(2 rows)
```

### s0709 — véhicules

```sql
SELECT count(*) FROM match_vehicle_takes_latest JOIN sc USING(match_id)
```
```
count_star()
0
(1 rows)
```

### s0709 — prises nettes de drapeau

```sql
SELECT s, sum(flag_grabs_raw) brut, sum(flag_grabs_net) net FROM match_flag_grabs_net_latest g JOIN sc USING(match_id) JOIN side ON side.match_id=g.match_id AND side.xuid=g.xuid GROUP BY 1 ORDER BY 1
```
```
s	brut	net
them	61	41
us	47	29
(2 rows)
```

### s0709 — vies (requête modèle, seuil 18 m)

```sql
, lv AS (SELECT l.match_id, l.start_ms, l.end_ms FROM match_lives_latest l WHERE l.xuid='2533274823110022' AND l.end_cause='death' AND l.match_id IN (SELECT match_id FROM sc)), j AS (SELECT lv.*, (SELECT d.nearest_teammate_m FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS near_m, (SELECT d.teammates_visible FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS vis, (SELECT count(*) FROM match_kill_events_latest k WHERE k.match_id=lv.match_id AND k.publishable AND k.feed_killer_xuid='2533274823110022' AND k.time_ms>=lv.start_ms AND k.time_ms<=lv.end_ms) AS frags FROM lv) SELECT CASE WHEN vis IS NULL THEN 'sans_contexte' WHEN vis=0 OR near_m IS NULL THEN 'aucun_coequipier_situe' WHEN near_m<18 THEN 'pres' ELSE 'seul' END cls, count(*) vies, sum(frags) frags FROM j GROUP BY 1 ORDER BY 1
```
```
cls	vies	frags
pres	53	35
sans_contexte	1	0
seul	10	11
(3 rows)
```

### solo — matchs

```sql
SELECT strftime(start_time_utc AT TIME ZONE 'Europe/Paris','%d/%m %H:%M') t, map_name, regexp_replace(pair_name,' on .*','') md, my_outcome, my_team, team_0_score, team_1_score, match_id IN (SELECT match_id FROM match_usage_films_latest) filme FROM sc ORDER BY start_time_utc
```
```
t	map_name	md	my_outcome	my_team	team_0_score	team_1_score	filme
22/09 19:27	Cliffhanger	Super Fiesta:Slayer	2	0	50	35	true
22/09 19:36	Recharge	Super Fiesta:Slayer	3	1	50	42	true
22/09 19:46	Catalyst	Super Fiesta:Slayer	3	0	40	50	true
22/09 19:57	Streets	Super Fiesta:Slayer	2	1	45	50	true
22/09 20:06	Streets	Super Fiesta:Slayer	2	1	43	50	true
22/09 20:19	Recharge	Super Fiesta:Slayer	2	1	33	50	true
(6 rows)
```

### solo — bonus par camp (témoins Emprise)

```sql
SELECT s, sum(COALESCE(json_extract(u.taken_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.taken_json,'$.powerup_overshield')::INT,0)) bonus_pris, sum(COALESCE(json_extract(u.kept_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.kept_json,'$.powerup_overshield')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_camo')::INT,0)+COALESCE(json_extract(u.dropped_json,'$.powerup_overshield')::INT,0)) bonus_perdus, sum(camo_ms+overshield_ms) effet_ms, sum(camo_kills+overshield_kills) frags_effet, sum(pad_pickups) pad_pickups FROM match_usage_players_latest u JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=u.match_id AND side.xuid=u.xuid GROUP BY 1 ORDER BY 1
```
```
s	bonus_pris	bonus_perdus	effet_ms	frags_effet	pad_pickups
them	0	0	705000	36	0
us	0	0	804900	50	0
(2 rows)
```

### solo — prises par niveau et par camp

```sql
SELECT t.tier, COALESCE(side.s,'sans_camp') s, sum(pickups) FROM match_pad_pickups_by_tier_latest t JOIN sc USING(match_id) LEFT JOIN side ON side.match_id=t.match_id AND side.xuid=t.xuid WHERE pickups>0 GROUP BY 1,2 ORDER BY 1,2
```
```
tier	s	sum(pickups)
(0 rows)
```

### solo — frags aux armes spéciales (feuille)

```sql
SELECT s, sum(power_weapon_kills) pwk, sum(kills) frags FROM match_participants p JOIN sc USING(match_id) JOIN side ON side.match_id=p.match_id AND side.xuid=p.xuid GROUP BY 1 ORDER BY 1
```
```
s	pwk	frags
them	101	256
us	91	282
(2 rows)
```

### solo — véhicules

```sql
SELECT count(*) FROM match_vehicle_takes_latest JOIN sc USING(match_id)
```
```
count_star()
0
(1 rows)
```

### solo — prises nettes de drapeau

```sql
SELECT s, sum(flag_grabs_raw) brut, sum(flag_grabs_net) net FROM match_flag_grabs_net_latest g JOIN sc USING(match_id) JOIN side ON side.match_id=g.match_id AND side.xuid=g.xuid GROUP BY 1 ORDER BY 1
```
```
s	brut	net
(0 rows)
```

### solo — vies (requête modèle, seuil 18 m)

```sql
, lv AS (SELECT l.match_id, l.start_ms, l.end_ms FROM match_lives_latest l WHERE l.xuid='2533274823110022' AND l.end_cause='death' AND l.match_id IN (SELECT match_id FROM sc)), j AS (SELECT lv.*, (SELECT d.nearest_teammate_m FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS near_m, (SELECT d.teammates_visible FROM match_death_context_latest d WHERE d.match_id=lv.match_id AND d.victim_xuid='2533274823110022' AND abs(d.time_ms-lv.end_ms)<=1500 ORDER BY abs(d.time_ms-lv.end_ms) LIMIT 1) AS vis, (SELECT count(*) FROM match_kill_events_latest k WHERE k.match_id=lv.match_id AND k.publishable AND k.feed_killer_xuid='2533274823110022' AND k.time_ms>=lv.start_ms AND k.time_ms<=lv.end_ms) AS frags FROM lv) SELECT CASE WHEN vis IS NULL THEN 'sans_contexte' WHEN vis=0 OR near_m IS NULL THEN 'aucun_coequipier_situe' WHEN near_m<18 THEN 'pres' ELSE 'seul' END cls, count(*) vies, sum(frags) frags FROM j GROUP BY 1 ORDER BY 1
```
```
cls	vies	frags
pres	53	46
sans_contexte	1	3
seul	5	11
(3 rows)
```

## 2. Calculs de `compute_s.js` (depuis les extraits)

Règles reprises : riposte et appui = `analysis/coordination` (fenêtre 5 s, morts vengeables, appuis
`publishable AND assist_known`, parité pondérée 100/n, n = présents à la fin dans mon camp) ; usages d'avant =
`analysis/sessionusage` (grandeurs, parts croisées, `equipmentMetrics`) ; ressources d'après = `squademprise`
(bonus `taken_json`, armes spéciales = niveau `puissance`, râteliers = `terrain`, matchs à niveaux mesurés) ;
arme de chaque frag = `source_tag` de `match_kill_events_latest` traduit par la table embarquée
`damagetag/data/labels.tsv` (copie `damagetag_labels.tsv`) puis `games/weapons/registry.go` (classe, rôle) ;
une alternative désignant deux armes différentes va en « Non attribué » ; reliquat feuille − film en « Non attribué ».
Portée : série par match de la maquette précédente (`ex/illu_range.tsv`, 189 matchs depuis le 1er juillet).

```json
{
 "s2209": {
  "key": "s2209",
  "tag": "2209",
  "label": "22/09 — escouade",
  "title": "soirée du 22 septembre 2026",
  "squad": [
   {
    "x": "2535469190789936",
    "gt": "Chocoboflor",
    "tok": "squad-player-2"
   },
   {
    "x": "2533274858283686",
    "gt": "Madina97294",
    "tok": "squad-player-3"
   }
  ],
  "ctx": "escouade",
  "n": 7,
  "filmed": 6,
  "tiersMatches": 6,
  "wins": 3,
  "losses": 4,
  "span": "21:23 → 22:29",
  "modes": [
   [
    "Arena:CTF",
    2
   ],
   [
    "Community:Team Slayer",
    3
   ],
   [
    "Arena:Team Slayer",
    2
   ]
  ],
  "matches": [
   {
    "d": "22/09",
    "h": "21:23",
    "map": "Starboard",
    "mode": "Arena:CTF",
    "outcome": 2,
    "ms": 3,
    "es": 0,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "21:36",
    "map": "Curfew",
    "mode": "Community:Team Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 48,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "21:46",
    "map": "Origin",
    "mode": "Arena:CTF",
    "outcome": 3,
    "ms": 1,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "21:59",
    "map": "Solution",
    "mode": "Community:Team Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 17,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "22:08",
    "map": "Detachment",
    "mode": "Arena:Team Slayer",
    "outcome": 3,
    "ms": 45,
    "es": 50,
    "filmed": false,
    "tiers": false,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "22:19",
    "map": "Shogun",
    "mode": "Community:Team Slayer",
    "outcome": 3,
    "ms": 37,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   },
   {
    "d": "22/09",
    "h": "22:29",
    "map": "Catalyst",
    "mode": "Arena:Team Slayer",
    "outcome": 3,
    "ms": 33,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 2
   }
  ],
  "coord": {
   "measured": 7,
   "total": 7,
   "delaiMs": 2569,
   "couvert": [
    21,
    86
   ],
   "riposte": [
    16,
    338
   ],
   "teamAvenged": 73,
   "parityR": 25,
   "prep": [
    31,
    64
   ],
   "part": [
    31,
    137
   ],
   "parityA": 25,
   "perMatch": [
    [
     0,
     61,
     3,
     35,
     6,
     4
    ],
    [
     1,
     48,
     4,
     20,
     6,
     4
    ],
    [
     2,
     66,
     2,
     30,
     4,
     4
    ],
    [
     3,
     17,
     1,
     12,
     4,
     4
    ],
    [
     4,
     50,
     2,
     17,
     5,
     4
    ],
    [
     5,
     46,
     2,
     11,
     3,
     4
    ],
    [
     6,
     50,
     2,
     12,
     3,
     4
    ]
   ]
  },
  "sunburst": {
   "shoulder": {
    "automatic": 2,
    "precision": 25,
    "shotgun": 1
   },
   "sidearm": {
    "sidearm": 13
   },
   "melee": {
    "melee": 6
   },
   "grenade": {
    "grenade_frag": 2,
    "grenade_plasma": 2,
    "grenade_dynamo": 1
   },
   "heavy": {
    "power": 11
   },
   "environmental": {
    "explosive_object": 2
   }
  },
  "fragBy": {
   "2533274823110022": {
    "shoulder": 28,
    "sidearm": 13,
    "melee": 6,
    "grenade": 5,
    "heavy": 11,
    "environmental": 2
   },
   "2535469190789936": {
    "sidearm": 19,
    "environmental": 2,
    "shoulder": 24,
    "melee": 12,
    "grenade": 2,
    "heavy": 4
   },
   "2533274858283686": {
    "melee": 15,
    "sidearm": 37,
    "shoulder": 45,
    "grenade": 4,
    "heavy": 16,
    "environmental": 3,
    "unattributed": 1
   }
  },
  "sheet": {
   "2533274823110022": 65,
   "2535469190789936": 63,
   "2533274858283686": 121
  },
  "filmKills": {
   "2533274823110022": 65,
   "2535469190789936": 63,
   "2533274858283686": 120
  },
  "tools": [
   {
    "key": "w:BR75",
    "label": "BR75",
    "cls": "shoulder",
    "by": {
     "2535469190789936": 22,
     "2533274858283686": 35,
     "2533274823110022": 22
    },
    "total": 79
   },
   {
    "key": "w:MK50 Sidekick",
    "label": "MK50 Sidekick",
    "cls": "sidearm",
    "by": {
     "2535469190789936": 17,
     "2533274858283686": 34,
     "2533274823110022": 13
    },
    "total": 64
   },
   {
    "key": "melee",
    "label": "Mêlée",
    "cls": "melee",
    "by": {
     "2533274858283686": 15,
     "2533274823110022": 6,
     "2535469190789936": 12
    },
    "total": 33
   },
   {
    "key": "w:M41 SPNKr",
    "label": "M41 SPNKr",
    "cls": "heavy",
    "by": {
     "2533274823110022": 11,
     "2533274858283686": 5,
     "2535469190789936": 4
    },
    "total": 20
   },
   {
    "key": "w:S7 Sniper",
    "label": "S7 Sniper",
    "cls": "heavy",
    "by": {
     "2533274858283686": 11
    },
    "total": 11
   },
   {
    "key": "w:MA40 AR",
    "label": "MA40 AR",
    "cls": "shoulder",
    "by": {
     "2533274858283686": 6,
     "2535469190789936": 2,
     "2533274823110022": 1
    },
    "total": 9
   },
   {
    "key": "g:grenade_frag",
    "label": "Grenade frag",
    "cls": "grenade",
    "by": {
     "2533274858283686": 4,
     "2533274823110022": 2,
     "2535469190789936": 1
    },
    "total": 7
   },
   {
    "key": "w:Bandit EVO",
    "label": "Bandit EVO",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 2,
     "2533274858283686": 4
    },
    "total": 6
   },
   {
    "key": "w:Déchiqueteur",
    "label": "Déchiqueteur",
    "cls": "sidearm",
    "by": {
     "2533274858283686": 3,
     "2535469190789936": 2
    },
    "total": 5
   },
   {
    "key": "explosive_object",
    "label": "Objet explosif (bidon)",
    "cls": "environmental",
    "by": {
     "2535469190789936": 1,
     "2533274858283686": 2,
     "2533274823110022": 2
    },
    "total": 5
   },
   {
    "key": "g:grenade_plasma",
    "label": "Grenade plasma",
    "cls": "grenade",
    "by": {
     "2535469190789936": 1,
     "2533274823110022": 2
    },
    "total": 3
   },
   {
    "key": "environment",
    "label": "Chute, environnement",
    "cls": "environmental",
    "by": {
     "2535469190789936": 1,
     "2533274858283686": 1
    },
    "total": 2
   },
   {
    "key": "w:Carabine Vestige",
    "label": "Carabine Vestige",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "g:grenade_dynamo",
    "label": "Grenade dynamo",
    "cls": "grenade",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:Mutilateur",
    "label": "Mutilateur",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:VK78 Commando",
    "label": "VK78 Commando",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "unattributed",
    "label": "Non attribué",
    "cls": "unattributed",
    "by": {
     "2533274858283686": 1
    },
    "total": 1
   }
  ],
  "usage": {
   "measured": 6,
   "total": 7,
   "teamSizeAvg": 4,
   "lobbySizeAvg": 8,
   "teamParity": 25,
   "lobbyParity": 12.5,
   "equip": [
    {
     "key": "equipment_powerup_camo",
     "kind": "equipment",
     "label": "Camouflage",
     "player": 1,
     "team": 7,
     "lobby": 30,
     "cad": {
      "me": 0.17,
      "sq": {
       "2533274858283686": 0.83,
       "2535469190789936": 0.17
      },
      "team": 1.17,
      "lobby": 5
     },
     "shareTeam": 14.29,
     "shareLobby": 3.33,
     "teamOfLobby": 23.33,
     "aboveTeam": 0,
     "outcomes": {
      "used": 1,
      "kept": 0,
      "dropped": 0,
      "tmRate": 66.67,
      "opRate": 100,
      "tm": [
       4,
       1,
       1
      ],
      "op": [
       23,
       0,
       0
      ]
     }
    },
    {
     "key": "equipment_powerup_overshield",
     "kind": "equipment",
     "label": "Surbouclier",
     "player": 2,
     "team": 6,
     "lobby": 11,
     "cad": {
      "me": 0.33,
      "sq": {
       "2533274858283686": 0.17,
       "2535469190789936": 0.33
      },
      "team": 1,
      "lobby": 1.83
     },
     "shareTeam": 33.33,
     "shareLobby": 18.18,
     "teamOfLobby": 54.55,
     "aboveTeam": 1,
     "outcomes": {
      "used": 2,
      "kept": 0,
      "dropped": 0,
      "tmRate": 100,
      "opRate": 60,
      "tm": [
       4,
       0,
       0
      ],
      "op": [
       3,
       0,
       2
      ]
     }
    },
    {
     "key": "equipment_shroud_screen",
     "kind": "equipment",
     "label": "Écran occultant",
     "player": 5,
     "team": 14,
     "lobby": 32,
     "cad": {
      "me": 0.83,
      "sq": {
       "2533274858283686": 0.33,
       "2535469190789936": 0.67
      },
      "team": 2.33,
      "lobby": 5.33
     },
     "shareTeam": 35.71,
     "shareLobby": 15.63,
     "teamOfLobby": 43.75,
     "aboveTeam": 3,
     "outcomes": {
      "used": 1,
      "kept": 1,
      "dropped": 3,
      "tmRate": 0,
      "opRate": 11.11,
      "tm": [
       0,
       2,
       7
      ],
      "op": [
       2,
       5,
       11
      ]
     }
    },
    {
     "key": "equipment_wall",
     "kind": "equipment",
     "label": "Mur de protection",
     "player": 4,
     "team": 15,
     "lobby": 31,
     "cad": {
      "me": 0.67,
      "sq": {
       "2533274858283686": 1.33,
       "2535469190789936": 0.17
      },
      "team": 2.5,
      "lobby": 5.17
     },
     "shareTeam": 26.67,
     "shareLobby": 12.9,
     "teamOfLobby": 48.39,
     "aboveTeam": 1,
     "outcomes": {
      "used": 3,
      "kept": 0,
      "dropped": 1,
      "tmRate": 45.45,
      "opRate": 25,
      "tm": [
       5,
       0,
       6
      ],
      "op": [
       4,
       1,
       11
      ]
     }
    },
    {
     "key": "grapple_pulls",
     "kind": "grapple",
     "label": "Grappin",
     "player": 2,
     "team": 8,
     "lobby": 17,
     "cad": {
      "me": 0.33,
      "sq": {
       "2533274858283686": 0.5,
       "2535469190789936": 0
      },
      "team": 1.33,
      "lobby": 2.83
     },
     "shareTeam": 25,
     "shareLobby": 11.76,
     "teamOfLobby": 47.06,
     "aboveTeam": 1
    }
   ],
   "pad": {
    "key": "pad_pickups",
    "kind": "pads",
    "label": "Toutes armes spéciales",
    "player": 13,
    "team": 44,
    "lobby": 95,
    "cad": {
     "me": 2.17,
     "sq": {
      "2533274858283686": 1.33,
      "2535469190789936": 1
     },
     "team": 7.33,
     "lobby": 15.83
    },
    "shareTeam": 29.55,
    "shareLobby": 13.68,
    "teamOfLobby": 46.32,
    "aboveTeam": 2
   },
   "padFamilies": [
    {
     "key": "71ab0a2c",
     "label": "M41 SPNKr",
     "player": 5,
     "team": 9,
     "lobby": 22,
     "shareTeam": 55.56,
     "shareLobby": 22.73,
     "teamOfLobby": 40.91
    },
    {
     "key": "80977ba5",
     "label": "Déchiqueteur",
     "player": 0,
     "team": 7,
     "lobby": 9,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 77.78
    },
    {
     "key": "0a1992bc",
     "label": "S7 Sniper",
     "player": 2,
     "team": 5,
     "lobby": 8,
     "shareTeam": 40,
     "shareLobby": 25,
     "teamOfLobby": 62.5
    },
    {
     "key": "4ff3937e",
     "label": "Épée à énergie",
     "player": 1,
     "team": 2,
     "lobby": 7,
     "shareTeam": 50,
     "shareLobby": 14.29,
     "teamOfLobby": 28.57
    },
    {
     "key": "30484ea6",
     "label": "Carabine à impulsion",
     "player": 0,
     "team": 3,
     "lobby": 6,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "fd98554c",
     "label": "VK78 Commando",
     "player": 1,
     "team": 5,
     "lobby": 6,
     "shareTeam": 20,
     "shareLobby": 16.67,
     "teamOfLobby": 83.33
    },
    {
     "key": "b533957e",
     "label": "Needler",
     "player": 1,
     "team": 1,
     "lobby": 5,
     "shareTeam": 100,
     "shareLobby": 20,
     "teamOfLobby": 20
    },
    {
     "key": "9387a8b9",
     "label": "Fusil électrique",
     "player": 0,
     "team": 2,
     "lobby": 4,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "a0955e9e",
     "label": "Rayon de Sentinelle",
     "player": 0,
     "team": 2,
     "lobby": 4,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "3e070217",
     "label": "Carabine Vestige",
     "player": 1,
     "team": 1,
     "lobby": 3,
     "shareTeam": 100,
     "shareLobby": 33.33,
     "teamOfLobby": 33.33
    },
    {
     "key": "f408190f",
     "label": "MK50 Sidekick",
     "player": 2,
     "team": 3,
     "lobby": 3,
     "shareTeam": 66.67,
     "shareLobby": 66.67,
     "teamOfLobby": 100
    },
    {
     "key": "2ac9c2ff",
     "label": "Calcineur",
     "player": 0,
     "team": 1,
     "lobby": 2,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "0d20c469",
     "label": "Empaleur",
     "player": 0,
     "team": 2,
     "lobby": 2,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 100
    },
    {
     "key": "2fb21c87",
     "label": "Bandit EVO",
     "player": 0,
     "team": 1,
     "lobby": 1,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 100
    }
   ],
   "track": {
    "me": 13,
    "sq": {
     "2533274858283686": 8,
     "2535469190789936": 6
    },
    "rest": 17,
    "them": 51
   },
   "tiers": {
    "terrain": {
     "tier": "terrain",
     "player": 3,
     "team": 24,
     "lobby": 54,
     "w": {
      "VK78 Commando": 1,
      "Carabine Vestige": 1,
      "Needler": 1
     },
     "shareTeam": 12.5,
     "shareLobby": 5.56,
     "teamOfLobby": 44.44
    },
    "puissance": {
     "tier": "puissance",
     "player": 7,
     "team": 13,
     "lobby": 34,
     "w": {
      "M41 SPNKr": 4,
      "S7 Sniper": 2,
      "Épée à énergie": 1
     },
     "shareTeam": 53.85,
     "shareLobby": 20.59,
     "teamOfLobby": 38.24
    },
    "base": {
     "tier": "base",
     "player": 2,
     "team": 2,
     "lobby": 2,
     "w": {
      "MK50 Sidekick": 2
     },
     "shareTeam": 100,
     "shareLobby": 100,
     "teamOfLobby": 100
    }
   },
   "tierNotes": {
    "measured": 6,
    "noPads": 0,
    "unest": 0,
    "unclassified": 1
   }
  },
  "objAvant": {
   "roleAgg": [
    {
     "role": "take",
     "me": 7,
     "team": 22,
     "lobby": 43,
     "sq": {
      "2535469190789936": 5,
      "2533274858283686": 7
     },
     "shareTeam": 31.82,
     "shareLobby": 16.28,
     "teamOfLobby": 51.16,
     "isDur": false
    },
    {
     "role": "defend",
     "me": 10,
     "team": 33,
     "lobby": 66,
     "sq": {
      "2535469190789936": 6,
      "2533274858283686": 11
     },
     "shareTeam": 30.3,
     "shareLobby": 15.15,
     "teamOfLobby": 50,
     "isDur": false
    },
    {
     "role": "hold",
     "me": 64.9,
     "team": 137.5,
     "lobby": 248.8,
     "sq": {
      "2535469190789936": 19.7,
      "2533274858283686": 44.8
     },
     "shareTeam": 47.2,
     "shareLobby": 26.09,
     "teamOfLobby": 55.27,
     "isDur": true
    }
   ],
   "teamParity": 25,
   "lobbyParity": 12.5,
   "teamOfLobbyParity": 50,
   "famGrid": [
    {
     "family": "ctf",
     "label": "Drapeau",
     "matches": 2,
     "roles": [
      31.82,
      30.3,
      47.2
     ]
    }
   ],
   "grabsLobbyRaw": 56,
   "withObj": 2
  },
  "balance": [
   {
    "family": "ctf",
    "label": "Drapeau",
    "matches": 2,
    "roles": [
     {
      "role": "take",
      "lines": [
       {
        "key": "flag_captures",
        "label": "Drapeaux capturés",
        "duration": false,
        "us": 4,
        "them": 3,
        "me": 4,
        "rest": 0,
        "sq": {
         "2535469190789936": 0,
         "2533274858283686": 0
        }
       },
       {
        "key": "flag_capture_assists",
        "label": "Aides à la capture",
        "duration": false,
        "us": 2,
        "them": 3,
        "me": 0,
        "rest": 2,
        "sq": {
         "2535469190789936": 1,
         "2533274858283686": 1
        }
       },
       {
        "key": "flag_steals",
        "label": "Drapeaux volés",
        "duration": false,
        "us": 14,
        "them": 12,
        "me": 3,
        "rest": 11,
        "sq": {
         "2535469190789936": 4,
         "2533274858283686": 5
        }
       },
       {
        "key": "flag_returners_killed",
        "label": "Rapatrieurs abattus",
        "duration": false,
        "us": 2,
        "them": 3,
        "me": 0,
        "rest": 2,
        "sq": {
         "2535469190789936": 0,
         "2533274858283686": 1
        }
       }
      ]
     },
     {
      "role": "defend",
      "lines": [
       {
        "key": "flag_returns",
        "label": "Retours",
        "duration": false,
        "us": 14,
        "them": 9,
        "me": 3,
        "rest": 11,
        "sq": {
         "2535469190789936": 4,
         "2533274858283686": 3
        }
       },
       {
        "key": "flag_secures",
        "label": "Drapeaux sécurisés",
        "duration": false,
        "us": 11,
        "them": 12,
        "me": 5,
        "rest": 6,
        "sq": {
         "2535469190789936": 2,
         "2533274858283686": 4
        }
       },
       {
        "key": "flag_carriers_killed",
        "label": "Porteurs abattus",
        "duration": false,
        "us": 8,
        "them": 12,
        "me": 2,
        "rest": 6,
        "sq": {
         "2535469190789936": 0,
         "2533274858283686": 4
        }
       }
      ]
     },
     {
      "role": "hold",
      "lines": [
       {
        "key": "time_as_flag_carrier_seconds",
        "label": "Temps de portage",
        "duration": true,
        "us": 137.5,
        "them": 111.3,
        "me": 64.9,
        "rest": 72.6,
        "sq": {
         "2535469190789936": 19.7,
         "2533274858283686": 44.8
        }
       }
      ]
     }
    ],
    "grabs": {
     "us": 22,
     "them": 26,
     "me": 7,
     "rawUs": 26,
     "rawThem": 30,
     "rows": 16
    }
   }
  ],
  "roleTot": {
   "me": [
    7,
    10,
    64.9
   ],
   "rest": [
    15,
    23,
    72.6
   ]
  },
  "dominant": {
   "me": "hold",
   "rest": "defend"
  },
  "control": {
   "powerup": {
    "us": 12,
    "them": 8,
    "me": 3,
    "lostUs": 2,
    "lostThem": 2
   },
   "power": {
    "us": 13,
    "them": 21,
    "me": 7,
    "lostUs": 0,
    "lostThem": 0
   },
   "rack": {
    "us": 24,
    "them": 30,
    "me": 3,
    "lostUs": 0,
    "lostThem": 0
   }
  },
  "objects": {
   "powerup": [
    {
     "key": "powerup_overshield",
     "name": "Surbouclier",
     "us": 6,
     "them": 5,
     "me": 2,
     "sq": {
      "2533274858283686": 1,
      "2535469190789936": 2
     }
    },
    {
     "key": "powerup_camo",
     "name": "Camouflage",
     "us": 6,
     "them": 3,
     "me": 1,
     "sq": {
      "2533274858283686": 4,
      "2535469190789936": 1
     }
    }
   ],
   "power": [
    {
     "key": "71ab0a2c",
     "name": "M41 SPNKr",
     "us": 6,
     "them": 13,
     "me": 4,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "0a1992bc",
     "name": "S7 Sniper",
     "us": 5,
     "them": 3,
     "me": 2,
     "sq": {
      "2533274858283686": 3
     }
    },
    {
     "key": "4ff3937e",
     "name": "Épée à énergie",
     "us": 2,
     "them": 5,
     "me": 1,
     "sq": {}
    }
   ],
   "rack": [
    {
     "key": "80977ba5",
     "name": "Déchiqueteur",
     "us": 7,
     "them": 2,
     "me": 0,
     "sq": {
      "2533274858283686": 4,
      "2535469190789936": 3
     }
    },
    {
     "key": "2b1824d5",
     "name": "BR75",
     "us": 0,
     "them": 7,
     "me": 0,
     "sq": {}
    },
    {
     "key": "30484ea6",
     "name": "Carabine à impulsion",
     "us": 3,
     "them": 3,
     "me": 0,
     "sq": {}
    },
    {
     "key": "fd98554c",
     "name": "VK78 Commando",
     "us": 5,
     "them": 1,
     "me": 1,
     "sq": {}
    },
    {
     "key": "b533957e",
     "name": "Needler",
     "us": 1,
     "them": 4,
     "me": 1,
     "sq": {}
    },
    {
     "key": "9387a8b9",
     "name": "Fusil électrique",
     "us": 2,
     "them": 2,
     "me": 0,
     "sq": {}
    },
    {
     "key": "a0955e9e",
     "name": "Rayon de Sentinelle",
     "us": 2,
     "them": 2,
     "me": 0,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "3e070217",
     "name": "Carabine Vestige",
     "us": 1,
     "them": 2,
     "me": 1,
     "sq": {}
    },
    {
     "key": "2ac9c2ff",
     "name": "Calcineur",
     "us": 1,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "c30d87c7",
     "name": "Ravageur",
     "us": 0,
     "them": 2,
     "me": 0,
     "sq": {}
    },
    {
     "key": "2fb21c87",
     "name": "Bandit EVO",
     "us": 1,
     "them": 0,
     "me": 0,
     "sq": {}
    },
    {
     "key": "b619d84a",
     "name": "CQS48 Bulldog",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "84bd29ed",
     "name": "Disrupteur",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "daf193c7",
     "name": "Fusil traqueur",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "f5c335df",
     "name": "MA5K Avenger",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "f408190f",
     "name": "MK50 Sidekick",
     "us": 1,
     "them": 0,
     "me": 0,
     "sq": {
      "2533274858283686": 1
     }
    }
   ]
  },
  "padsEmptied": 33,
  "gridCols": [
   {
    "d": "22/09",
    "h": "21:23",
    "map": "Starboard",
    "mode": "CTF",
    "outcome": 2,
    "ms": 3,
    "es": 0,
    "filmed": true,
    "tiers": true,
    "pwk": [
     0,
     4
    ],
    "o": {
     "powerup|powerup_overshield": [
      5,
      2,
      2
     ],
     "rack|9387a8b9": [
      2,
      2,
      0
     ],
     "power|4ff3937e": [
      0,
      2,
      0
     ],
     "rack|b619d84a": [
      0,
      1,
      0
     ],
     "rack|fd98554c": [
      2,
      0,
      1
     ]
    }
   },
   {
    "d": "22/09",
    "h": "21:36",
    "map": "Curfew",
    "mode": "Team Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 48,
    "filmed": true,
    "tiers": true,
    "pwk": [
     12,
     4
    ],
    "o": {
     "powerup|powerup_camo": [
      4,
      0,
      1
     ],
     "rack|3e070217": [
      1,
      1,
      1
     ],
     "rack|30484ea6": [
      1,
      3,
      0
     ],
     "rack|a0955e9e": [
      1,
      2,
      0
     ],
     "rack|80977ba5": [
      2,
      1,
      0
     ],
     "rack|fd98554c": [
      3,
      0,
      0
     ]
    }
   },
   {
    "d": "22/09",
    "h": "21:46",
    "map": "Origin",
    "mode": "CTF",
    "outcome": 3,
    "ms": 1,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "pwk": [
     8,
     11
    ],
    "o": {
     "power|0a1992bc": [
      3,
      1,
      2
     ],
     "power|71ab0a2c": [
      2,
      6,
      2
     ],
     "rack|2b1824d5": [
      0,
      7,
      0
     ]
    }
   },
   {
    "d": "22/09",
    "h": "21:59",
    "map": "Solution",
    "mode": "Team Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 17,
    "filmed": true,
    "tiers": true,
    "pwk": [
     5,
     0
    ],
    "o": {
     "rack|80977ba5": [
      4,
      1,
      0
     ],
     "rack|c30d87c7": [
      0,
      1,
      0
     ],
     "rack|84bd29ed": [
      0,
      1,
      0
     ],
     "power|0a1992bc": [
      2,
      0,
      0
     ],
     "rack|2fb21c87": [
      1,
      0,
      0
     ]
    }
   },
   {
    "d": "22/09",
    "h": "22:08",
    "map": "Detachment",
    "mode": "Team Slayer",
    "outcome": 3,
    "ms": 45,
    "es": 50,
    "filmed": false,
    "tiers": false,
    "pwk": [
     9,
     13
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "22:19",
    "map": "Shogun",
    "mode": "Team Slayer",
    "outcome": 3,
    "ms": 37,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "pwk": [
     4,
     10
    ],
    "o": {
     "powerup|powerup_camo": [
      2,
      3,
      0
     ],
     "rack|a0955e9e": [
      1,
      0,
      0
     ],
     "rack|80977ba5": [
      1,
      0,
      0
     ],
     "rack|fd98554c": [
      0,
      1,
      0
     ],
     "rack|f5c335df": [
      0,
      1,
      0
     ],
     "rack|c30d87c7": [
      0,
      1,
      0
     ],
     "rack|2ac9c2ff": [
      1,
      1,
      0
     ],
     "power|71ab0a2c": [
      1,
      3,
      0
     ],
     "power|4ff3937e": [
      1,
      2,
      0
     ],
     "rack|daf193c7": [
      0,
      1,
      0
     ],
     "rack|30484ea6": [
      2,
      0,
      0
     ]
    }
   },
   {
    "d": "22/09",
    "h": "22:29",
    "map": "Catalyst",
    "mode": "Team Slayer",
    "outcome": 3,
    "ms": 33,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "pwk": [
     9,
     12
    ],
    "o": {
     "powerup|powerup_overshield": [
      1,
      3,
      0
     ],
     "power|71ab0a2c": [
      3,
      4,
      2
     ],
     "rack|b533957e": [
      1,
      4,
      1
     ],
     "power|4ff3937e": [
      1,
      1,
      1
     ],
     "rack|3e070217": [
      0,
      1,
      0
     ],
     "power|0a1992bc": [
      0,
      2,
      0
     ],
     "rack|f408190f": [
      1,
      0,
      0
     ]
    }
   }
  ],
  "effect": {
   "us": {
    "ms": 159100,
    "kills": 8
   },
   "them": {
    "ms": 112800,
    "kills": 5
   }
  },
  "pwkAll": {
   "us": 47,
   "them": 54
  },
  "pwkTiers": {
   "us": 38,
   "them": 41
  },
  "equip": [
   {
    "key": "grapple",
    "label": "Grappin",
    "me": null,
    "dropsMe": 4
   },
   {
    "key": "wall",
    "label": "Mur de protection",
    "me": [
     3,
     0,
     1
    ],
    "rest": [
     5,
     0,
     6
    ],
    "takenMe": 4
   },
   {
    "key": "sensor",
    "label": "Capteur de menaces",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "translocator_beacon",
    "label": "Translocateur",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "shroud_screen",
    "label": "Écran occultant",
    "me": [
     1,
     1,
     3
    ],
    "rest": [
     0,
     2,
     7
    ],
    "takenMe": 4
   },
   {
    "key": "threat_seeker",
    "label": "Traqueur de menaces",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "repair_field",
    "label": "Champ de réparation",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "thruster",
    "label": "Propulseur",
    "me": null,
    "dropsMe": 0
   }
  ],
  "vies": {
   "near": [
    56,
    37
   ],
   "alone": [
    18,
    4
   ],
   "noctx": 0,
   "nomate": 0
  },
  "vehRows": 0,
  "sessKills": 65,
  "sessMeasured": 49
 },
 "s0709": {
  "key": "s0709",
  "tag": "0709",
  "label": "07/09 — escouade, objectifs",
  "title": "soirée du 7 septembre 2026",
  "squad": [
   {
    "x": "2533274833178266",
    "gt": "XxDaemonGamerxX",
    "tok": "squad-player-2"
   },
   {
    "x": "2533274858283686",
    "gt": "Madina97294",
    "tok": "squad-player-3"
   },
   {
    "x": "2535469190789936",
    "gt": "Chocoboflor",
    "tok": "squad-player-4"
   }
  ],
  "ctx": "escouade",
  "n": 7,
  "filmed": 7,
  "tiersMatches": 7,
  "wins": 1,
  "losses": 6,
  "span": "21:26 → 22:14",
  "modes": [
   [
    "Arena:Strongholds",
    4
   ],
   [
    "Arena:CTF",
    3
   ]
  ],
  "matches": [
   {
    "d": "07/09",
    "h": "21:26",
    "map": "Banished Narrows",
    "mode": "Arena:Strongholds",
    "outcome": 3,
    "ms": 82,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "21:34",
    "map": "Isolation",
    "mode": "Arena:Strongholds",
    "outcome": 3,
    "ms": 31,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "21:42",
    "map": "Illusion",
    "mode": "Arena:Strongholds",
    "outcome": 2,
    "ms": 200,
    "es": 183,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "21:52",
    "map": "Fortress",
    "mode": "Arena:Strongholds",
    "outcome": 3,
    "ms": 89,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "22:00",
    "map": "Origin",
    "mode": "Arena:CTF",
    "outcome": 3,
    "ms": 1,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "22:10",
    "map": "Domicile",
    "mode": "Arena:CTF",
    "outcome": 3,
    "ms": 0,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   },
   {
    "d": "07/09",
    "h": "22:14",
    "map": "Absolution",
    "mode": "Arena:CTF",
    "outcome": 3,
    "ms": 0,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 3
   }
  ],
  "coord": {
   "measured": 7,
   "total": 7,
   "delaiMs": 1968,
   "couvert": [
    10,
    65
   ],
   "riposte": [
    11,
    252
   ],
   "teamAvenged": 37,
   "parityR": 25,
   "prep": [
    22,
    50
   ],
   "part": [
    22,
    82
   ],
   "parityA": 25,
   "perMatch": [
    [
     0,
     40,
     1,
     17,
     4,
     4
    ],
    [
     1,
     29,
     1,
     6,
     1,
     4
    ],
    [
     2,
     37,
     0,
     7,
     2,
     4
    ],
    [
     3,
     44,
     2,
     12,
     5,
     4
    ],
    [
     4,
     42,
     2,
     18,
     3,
     4
    ],
    [
     5,
     16,
     0,
     3,
     1,
     4
    ],
    [
     6,
     44,
     5,
     19,
     6,
     4
    ]
   ]
  },
  "sunburst": {
   "melee": {
    "melee": 8
   },
   "shoulder": {
    "automatic": 3,
    "shotgun": 1,
    "special": 1
   },
   "sidearm": {
    "sidearm": 28
   },
   "heavy": {
    "power": 4,
    "sniper": 1
   },
   "grenade": {
    "grenade_frag": 1,
    "grenade_dynamo": 2
   },
   "environmental": {
    "explosive_object": 1
   }
  },
  "fragBy": {
   "2533274823110022": {
    "melee": 8,
    "shoulder": 5,
    "sidearm": 28,
    "heavy": 5,
    "grenade": 3,
    "environmental": 1
   },
   "2533274833178266": {
    "grenade": 2,
    "heavy": 2,
    "sidearm": 4,
    "shoulder": 4
   },
   "2533274858283686": {
    "sidearm": 39,
    "heavy": 13,
    "shoulder": 7,
    "melee": 9,
    "grenade": 3,
    "environmental": 3
   },
   "2535469190789936": {
    "shoulder": 13,
    "melee": 8,
    "sidearm": 24,
    "grenade": 2,
    "heavy": 1,
    "environmental": 1
   }
  },
  "sheet": {
   "2533274823110022": 50,
   "2533274833178266": 12,
   "2533274858283686": 74,
   "2535469190789936": 49
  },
  "filmKills": {
   "2533274823110022": 50,
   "2533274833178266": 12,
   "2533274858283686": 74,
   "2535469190789936": 49
  },
  "tools": [
   {
    "key": "w:MK50 Sidekick",
    "label": "MK50 Sidekick",
    "cls": "sidearm",
    "by": {
     "2533274858283686": 38,
     "2535469190789936": 24,
     "2533274823110022": 28,
     "2533274833178266": 4
    },
    "total": 94
   },
   {
    "key": "melee",
    "label": "Mêlée",
    "cls": "melee",
    "by": {
     "2535469190789936": 8,
     "2533274823110022": 8,
     "2533274858283686": 9
    },
    "total": 25
   },
   {
    "key": "w:MA40 AR",
    "label": "MA40 AR",
    "cls": "shoulder",
    "by": {
     "2535469190789936": 12,
     "2533274823110022": 1,
     "2533274858283686": 3,
     "2533274833178266": 4
    },
    "total": 20
   },
   {
    "key": "w:S7 Sniper",
    "label": "S7 Sniper",
    "cls": "heavy",
    "by": {
     "2533274858283686": 9,
     "2533274823110022": 1,
     "2533274833178266": 2
    },
    "total": 12
   },
   {
    "key": "g:grenade_frag",
    "label": "Grenade frag",
    "cls": "grenade",
    "by": {
     "2535469190789936": 1,
     "2533274833178266": 2,
     "2533274858283686": 3,
     "2533274823110022": 1
    },
    "total": 7
   },
   {
    "key": "w:M41 SPNKr",
    "label": "M41 SPNKr",
    "cls": "heavy",
    "by": {
     "2533274823110022": 3,
     "2533274858283686": 4
    },
    "total": 7
   },
   {
    "key": "explosive_object",
    "label": "Objet explosif (bidon)",
    "cls": "environmental",
    "by": {
     "2533274823110022": 1,
     "2533274858283686": 3,
     "2535469190789936": 1
    },
    "total": 5
   },
   {
    "key": "w:Bandit EVO",
    "label": "Bandit EVO",
    "cls": "shoulder",
    "by": {
     "2533274858283686": 3
    },
    "total": 3
   },
   {
    "key": "g:grenade_dynamo",
    "label": "Grenade dynamo",
    "cls": "grenade",
    "by": {
     "2533274823110022": 2
    },
    "total": 2
   },
   {
    "key": "w:MA5K Avenger",
    "label": "MA5K Avenger",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 2
    },
    "total": 2
   },
   {
    "key": "w:Needler",
    "label": "Needler",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1,
     "2535469190789936": 1
    },
    "total": 2
   },
   {
    "key": "w:BR75",
    "label": "BR75",
    "cls": "shoulder",
    "by": {
     "2533274858283686": 1
    },
    "total": 1
   },
   {
    "key": "w:Déchiqueteur",
    "label": "Déchiqueteur",
    "cls": "sidearm",
    "by": {
     "2533274858283686": 1
    },
    "total": 1
   },
   {
    "key": "g:grenade_splinter",
    "label": "Grenade splinter",
    "cls": "grenade",
    "by": {
     "2535469190789936": 1
    },
    "total": 1
   },
   {
    "key": "w:Hydra",
    "label": "Hydra",
    "cls": "heavy",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:Marteau antigravité",
    "label": "Marteau antigravité",
    "cls": "heavy",
    "by": {
     "2535469190789936": 1
    },
    "total": 1
   },
   {
    "key": "w:Mutilateur",
    "label": "Mutilateur",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   }
  ],
  "usage": {
   "measured": 7,
   "total": 7,
   "teamSizeAvg": 4,
   "lobbySizeAvg": 8,
   "teamParity": 25,
   "lobbyParity": 12.5,
   "equip": [
    {
     "key": "equipment_powerup_camo",
     "kind": "equipment",
     "label": "Camouflage",
     "player": 7,
     "team": 28,
     "lobby": 50,
     "cad": {
      "me": 1,
      "sq": {
       "2533274833178266": 0.57,
       "2533274858283686": 1.71,
       "2535469190789936": 0.71
      },
      "team": 4,
      "lobby": 7.14
     },
     "shareTeam": 25,
     "shareLobby": 14,
     "teamOfLobby": 56,
     "aboveTeam": 1,
     "outcomes": {
      "used": 7,
      "kept": 0,
      "dropped": 0,
      "tmRate": 100,
      "opRate": 100,
      "tm": [
       21,
       0,
       0
      ],
      "op": [
       22,
       0,
       0
      ]
     }
    },
    {
     "key": "equipment_powerup_overshield",
     "kind": "equipment",
     "label": "Surbouclier",
     "player": 4,
     "team": 6,
     "lobby": 11,
     "cad": {
      "me": 0.57,
      "sq": {
       "2533274833178266": 0,
       "2533274858283686": 0.14,
       "2535469190789936": 0.14
      },
      "team": 0.86,
      "lobby": 1.57
     },
     "shareTeam": 66.67,
     "shareLobby": 36.36,
     "teamOfLobby": 54.55,
     "aboveTeam": 3,
     "outcomes": {
      "used": 4,
      "kept": 0,
      "dropped": 0,
      "tmRate": 100,
      "opRate": 80,
      "tm": [
       2,
       0,
       0
      ],
      "op": [
       4,
       0,
       1
      ]
     }
    },
    {
     "key": "equipment_threat_seeker",
     "kind": "equipment",
     "label": "Traqueur de menaces",
     "player": 1,
     "team": 3,
     "lobby": 13,
     "cad": {
      "me": 0.14,
      "sq": {
       "2533274833178266": 0.14,
       "2533274858283686": 0,
       "2535469190789936": 0.14
      },
      "team": 0.43,
      "lobby": 1.86
     },
     "shareTeam": 33.33,
     "shareLobby": 7.69,
     "teamOfLobby": 23.08,
     "aboveTeam": 1,
     "outcomes": {
      "used": 0,
      "kept": 0,
      "dropped": 1,
      "tmRate": 0,
      "opRate": 30,
      "tm": [
       0,
       1,
       1
      ],
      "op": [
       3,
       2,
       5
      ]
     }
    },
    {
     "key": "deployed_wall",
     "kind": "wall",
     "label": "Mur de protection",
     "player": 0,
     "team": 0,
     "lobby": 6,
     "cad": {
      "me": 0,
      "sq": {
       "2533274833178266": 0,
       "2533274858283686": 0,
       "2535469190789936": 0
      },
      "team": 0,
      "lobby": 0.86
     },
     "shareTeam": null,
     "shareLobby": 0,
     "teamOfLobby": 0,
     "aboveTeam": 0
    },
    {
     "key": "grapple_pulls",
     "kind": "grapple",
     "label": "Grappin",
     "player": 3,
     "team": 5,
     "lobby": 5,
     "cad": {
      "me": 0.43,
      "sq": {
       "2533274833178266": 0.29,
       "2533274858283686": 0,
       "2535469190789936": 0
      },
      "team": 0.71,
      "lobby": 0.71
     },
     "shareTeam": 60,
     "shareLobby": 60,
     "teamOfLobby": 100,
     "aboveTeam": 2
    }
   ],
   "pad": {
    "key": "pad_pickups",
    "kind": "pads",
    "label": "Toutes armes spéciales",
    "player": 10,
    "team": 40,
    "lobby": 98,
    "cad": {
     "me": 1.43,
     "sq": {
      "2533274833178266": 0.71,
      "2533274858283686": 2.14,
      "2535469190789936": 1.43
     },
     "team": 5.71,
     "lobby": 14
    },
    "shareTeam": 25,
    "shareLobby": 10.2,
    "teamOfLobby": 40.82,
    "aboveTeam": 3
   },
   "padFamilies": [
    {
     "key": "0a1992bc",
     "label": "S7 Sniper",
     "player": 4,
     "team": 12,
     "lobby": 22,
     "shareTeam": 33.33,
     "shareLobby": 18.18,
     "teamOfLobby": 54.55
    },
    {
     "key": "71ab0a2c",
     "label": "M41 SPNKr",
     "player": 1,
     "team": 3,
     "lobby": 18,
     "shareTeam": 33.33,
     "shareLobby": 5.56,
     "teamOfLobby": 16.67
    },
    {
     "key": "2b1824d5",
     "label": "BR75",
     "player": 0,
     "team": 5,
     "lobby": 17,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 29.41
    },
    {
     "key": "80977ba5",
     "label": "Déchiqueteur",
     "player": 0,
     "team": 5,
     "lobby": 7,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 71.43
    },
    {
     "key": "b619d84a",
     "label": "CQS48 Bulldog",
     "player": 1,
     "team": 1,
     "lobby": 6,
     "shareTeam": 100,
     "shareLobby": 16.67,
     "teamOfLobby": 16.67
    },
    {
     "key": "d7915565",
     "label": "Mutilateur",
     "player": 1,
     "team": 2,
     "lobby": 5,
     "shareTeam": 50,
     "shareLobby": 20,
     "teamOfLobby": 40
    },
    {
     "key": "2fb21c87",
     "label": "Bandit EVO",
     "player": 0,
     "team": 1,
     "lobby": 3,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 33.33
    },
    {
     "key": "f408190f",
     "label": "MK50 Sidekick",
     "player": 1,
     "team": 2,
     "lobby": 3,
     "shareTeam": 50,
     "shareLobby": 33.33,
     "teamOfLobby": 66.67
    },
    {
     "key": "fd98554c",
     "label": "VK78 Commando",
     "player": 0,
     "team": 2,
     "lobby": 3,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 66.67
    },
    {
     "key": "3e070217",
     "label": "Carabine Vestige",
     "player": 0,
     "team": 1,
     "lobby": 2,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "841ac5e5",
     "label": "Marteau antigravité",
     "player": 0,
     "team": 1,
     "lobby": 2,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 50
    },
    {
     "key": "b533957e",
     "label": "Needler",
     "player": 1,
     "team": 2,
     "lobby": 2,
     "shareTeam": 50,
     "shareLobby": 50,
     "teamOfLobby": 100
    },
    {
     "key": "0d20c469",
     "label": "Empaleur",
     "player": 0,
     "team": 1,
     "lobby": 1,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 100
    },
    {
     "key": "daf193c7",
     "label": "Fusil traqueur",
     "player": 0,
     "team": 1,
     "lobby": 1,
     "shareTeam": 0,
     "shareLobby": 0,
     "teamOfLobby": 100
    },
    {
     "key": "f5c335df",
     "label": "MA5K Avenger",
     "player": 1,
     "team": 1,
     "lobby": 1,
     "shareTeam": 100,
     "shareLobby": 100,
     "teamOfLobby": 100
    }
   ],
   "track": {
    "me": 10,
    "sq": {
     "2533274833178266": 5,
     "2533274858283686": 15,
     "2535469190789936": 10
    },
    "rest": 0,
    "them": 58
   },
   "tiers": {
    "terrain": {
     "tier": "terrain",
     "player": 3,
     "team": 19,
     "lobby": 47,
     "w": {
      "Needler": 1,
      "MA5K Avenger": 1,
      "CQS48 Bulldog": 1
     },
     "shareTeam": 15.79,
     "shareLobby": 6.38,
     "teamOfLobby": 40.43
    },
    "puissance": {
     "tier": "puissance",
     "player": 6,
     "team": 18,
     "lobby": 47,
     "w": {
      "Mutilateur": 1,
      "S7 Sniper": 4,
      "M41 SPNKr": 1
     },
     "shareTeam": 33.33,
     "shareLobby": 12.77,
     "teamOfLobby": 38.3
    },
    "base": {
     "tier": "base",
     "player": 1,
     "team": 2,
     "lobby": 3,
     "w": {
      "MK50 Sidekick": 1
     },
     "shareTeam": 50,
     "shareLobby": 33.33,
     "teamOfLobby": 66.67
    }
   },
   "tierNotes": {
    "measured": 7,
    "noPads": 0,
    "unest": 0,
    "unclassified": 0
   }
  },
  "objAvant": {
   "roleAgg": [
    {
     "role": "take",
     "me": 28,
     "team": 110,
     "lobby": 257,
     "sq": {
      "2533274858283686": 36,
      "2533274833178266": 10,
      "2535469190789936": 36
     },
     "shareTeam": 25.45,
     "shareLobby": 10.89,
     "teamOfLobby": 42.8,
     "isDur": false
    },
    {
     "role": "defend",
     "me": 32,
     "team": 71,
     "lobby": 183,
     "sq": {
      "2533274858283686": 27,
      "2533274833178266": 6,
      "2535469190789936": 6
     },
     "shareTeam": 45.07,
     "shareLobby": 17.49,
     "teamOfLobby": 38.8,
     "isDur": false
    },
    {
     "role": "hold",
     "me": 342.3,
     "team": 1238.3,
     "lobby": 2509.3,
     "sq": {
      "2533274858283686": 348.6,
      "2533274833178266": 214.8,
      "2535469190789936": 332.6
     },
     "shareTeam": 27.64,
     "shareLobby": 13.64,
     "teamOfLobby": 49.35,
     "isDur": true
    }
   ],
   "teamParity": 25,
   "lobbyParity": 12.5,
   "teamOfLobbyParity": 50,
   "famGrid": [
    {
     "family": "zones_strongholds",
     "label": "Bases",
     "matches": 4,
     "roles": [
      24.73,
      46.67,
      25.41
     ]
    },
    {
     "family": "ctf",
     "label": "Drapeau",
     "matches": 3,
     "roles": [
      29.41,
      43.9,
      41.64
     ]
    }
   ],
   "grabsLobbyRaw": 108,
   "withObj": 7
  },
  "balance": [
   {
    "family": "zones_strongholds",
    "label": "Bases",
    "matches": 4,
    "roles": [
     {
      "role": "take",
      "lines": [
       {
        "key": "zone_captures",
        "label": "Zones capturées",
        "duration": false,
        "us": 71,
        "them": 79,
        "me": 18,
        "rest": 53,
        "sq": {
         "2533274833178266": 7,
         "2533274858283686": 20,
         "2535469190789936": 26
        }
       },
       {
        "key": "zone_offensive_kills",
        "label": "Frags offensifs de zone",
        "duration": false,
        "us": 22,
        "them": 30,
        "me": 5,
        "rest": 17,
        "sq": {
         "2533274833178266": 0,
         "2533274858283686": 11,
         "2535469190789936": 6
        }
       }
      ]
     },
     {
      "role": "defend",
      "lines": [
       {
        "key": "zone_secures",
        "label": "Zones sécurisées",
        "duration": false,
        "us": 10,
        "them": 27,
        "me": 7,
        "rest": 3,
        "sq": {
         "2533274833178266": 0,
         "2533274858283686": 2,
         "2535469190789936": 1
        }
       },
       {
        "key": "zone_defensive_kills",
        "label": "Frags défensifs de zone",
        "duration": false,
        "us": 20,
        "them": 33,
        "me": 7,
        "rest": 13,
        "sq": {
         "2533274833178266": 2,
         "2533274858283686": 10,
         "2535469190789936": 1
        }
       }
      ]
     },
     {
      "role": "hold",
      "lines": [
       {
        "key": "time_in_zones_seconds",
        "label": "Temps en zone",
        "duration": true,
        "us": 1067.8,
        "them": 926.1,
        "me": 271.3,
        "rest": 796.5,
        "sq": {
         "2533274833178266": 202,
         "2533274858283686": 308.5,
         "2535469190789936": 286
        }
       }
      ]
     }
    ],
    "grabs": null
   },
   {
    "family": "ctf",
    "label": "Drapeau",
    "matches": 3,
    "roles": [
     {
      "role": "take",
      "lines": [
       {
        "key": "flag_captures",
        "label": "Drapeaux capturés",
        "duration": false,
        "us": 1,
        "them": 9,
        "me": 0,
        "rest": 1,
        "sq": {
         "2533274833178266": 0,
         "2533274858283686": 0,
         "2535469190789936": 1
        }
       },
       {
        "key": "flag_capture_assists",
        "label": "Aides à la capture",
        "duration": false,
        "us": 1,
        "them": 7,
        "me": 0,
        "rest": 1,
        "sq": {
         "2533274833178266": 1,
         "2533274858283686": 0,
         "2535469190789936": 0
        }
       },
       {
        "key": "flag_steals",
        "label": "Drapeaux volés",
        "duration": false,
        "us": 15,
        "them": 16,
        "me": 5,
        "rest": 10,
        "sq": {
         "2533274833178266": 2,
         "2533274858283686": 5,
         "2535469190789936": 3
        }
       },
       {
        "key": "flag_returners_killed",
        "label": "Rapatrieurs abattus",
        "duration": false,
        "us": 0,
        "them": 6,
        "me": 0,
        "rest": 0,
        "sq": {
         "2533274833178266": 0,
         "2533274858283686": 0,
         "2535469190789936": 0
        }
       }
      ]
     },
     {
      "role": "defend",
      "lines": [
       {
        "key": "flag_returns",
        "label": "Retours",
        "duration": false,
        "us": 7,
        "them": 15,
        "me": 4,
        "rest": 3,
        "sq": {
         "2533274833178266": 1,
         "2533274858283686": 2,
         "2535469190789936": 0
        }
       },
       {
        "key": "flag_secures",
        "label": "Drapeaux sécurisés",
        "duration": false,
        "us": 24,
        "them": 30,
        "me": 12,
        "rest": 12,
        "sq": {
         "2533274833178266": 1,
         "2533274858283686": 7,
         "2535469190789936": 4
        }
       },
       {
        "key": "flag_carriers_killed",
        "label": "Porteurs abattus",
        "duration": false,
        "us": 10,
        "them": 7,
        "me": 2,
        "rest": 8,
        "sq": {
         "2533274833178266": 2,
         "2533274858283686": 6,
         "2535469190789936": 0
        }
       }
      ]
     },
     {
      "role": "hold",
      "lines": [
       {
        "key": "time_as_flag_carrier_seconds",
        "label": "Temps de portage",
        "duration": true,
        "us": 170.5,
        "them": 344.9,
        "me": 71,
        "rest": 99.5,
        "sq": {
         "2533274833178266": 12.8,
         "2533274858283686": 40.1,
         "2535469190789936": 46.6
        }
       }
      ]
     }
    ],
    "grabs": {
     "us": 29,
     "them": 41,
     "me": 12,
     "rawUs": 47,
     "rawThem": 61,
     "rows": 25
    }
   }
  ],
  "roleTot": {
   "me": [
    28,
    32,
    342.3
   ],
   "rest": [
    82,
    39,
    896
   ]
  },
  "dominant": {
   "me": "defend",
   "rest": "take"
  },
  "control": {
   "powerup": {
    "us": 8,
    "them": 9,
    "me": 5,
    "lostUs": 0,
    "lostThem": 1
   },
   "power": {
    "us": 18,
    "them": 29,
    "me": 6,
    "lostUs": 0,
    "lostThem": 0
   },
   "rack": {
    "us": 19,
    "them": 28,
    "me": 3,
    "lostUs": 0,
    "lostThem": 0
   }
  },
  "objects": {
   "powerup": [
    {
     "key": "powerup_overshield",
     "name": "Surbouclier",
     "us": 6,
     "them": 5,
     "me": 4,
     "sq": {
      "2533274833178266": 0,
      "2533274858283686": 1,
      "2535469190789936": 1
     }
    },
    {
     "key": "powerup_camo",
     "name": "Camouflage",
     "us": 2,
     "them": 4,
     "me": 1,
     "sq": {
      "2533274833178266": 0,
      "2533274858283686": 1,
      "2535469190789936": 0
     }
    }
   ],
   "power": [
    {
     "key": "0a1992bc",
     "name": "S7 Sniper",
     "us": 11,
     "them": 10,
     "me": 4,
     "sq": {
      "2533274858283686": 5,
      "2535469190789936": 1,
      "2533274833178266": 1
     }
    },
    {
     "key": "71ab0a2c",
     "name": "M41 SPNKr",
     "us": 3,
     "them": 15,
     "me": 1,
     "sq": {
      "2533274858283686": 2
     }
    },
    {
     "key": "d7915565",
     "name": "Mutilateur",
     "us": 2,
     "them": 3,
     "me": 1,
     "sq": {
      "2533274858283686": 1
     }
    },
    {
     "key": "841ac5e5",
     "name": "Marteau antigravité",
     "us": 1,
     "them": 1,
     "me": 0,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "0d20c469",
     "name": "Empaleur",
     "us": 1,
     "them": 0,
     "me": 0,
     "sq": {
      "2533274858283686": 1
     }
    }
   ],
   "rack": [
    {
     "key": "2b1824d5",
     "name": "BR75",
     "us": 5,
     "them": 12,
     "me": 0,
     "sq": {
      "2533274833178266": 2,
      "2535469190789936": 2,
      "2533274858283686": 1
     }
    },
    {
     "key": "80977ba5",
     "name": "Déchiqueteur",
     "us": 5,
     "them": 2,
     "me": 0,
     "sq": {
      "2535469190789936": 2,
      "2533274858283686": 2,
      "2533274833178266": 1
     }
    },
    {
     "key": "b619d84a",
     "name": "CQS48 Bulldog",
     "us": 1,
     "them": 5,
     "me": 1,
     "sq": {}
    },
    {
     "key": "2fb21c87",
     "name": "Bandit EVO",
     "us": 1,
     "them": 2,
     "me": 0,
     "sq": {
      "2533274858283686": 1
     }
    },
    {
     "key": "fd98554c",
     "name": "VK78 Commando",
     "us": 2,
     "them": 1,
     "me": 0,
     "sq": {
      "2533274833178266": 1,
      "2533274858283686": 1
     }
    },
    {
     "key": "3e070217",
     "name": "Carabine Vestige",
     "us": 1,
     "them": 1,
     "me": 0,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "b533957e",
     "name": "Needler",
     "us": 2,
     "them": 0,
     "me": 1,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "c30d87c7",
     "name": "Ravageur",
     "us": 0,
     "them": 2,
     "me": 0,
     "sq": {}
    },
    {
     "key": "30484ea6",
     "name": "Carabine à impulsion",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "84bd29ed",
     "name": "Disrupteur",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    },
    {
     "key": "daf193c7",
     "name": "Fusil traqueur",
     "us": 1,
     "them": 0,
     "me": 0,
     "sq": {
      "2535469190789936": 1
     }
    },
    {
     "key": "f5c335df",
     "name": "MA5K Avenger",
     "us": 1,
     "them": 0,
     "me": 1,
     "sq": {}
    },
    {
     "key": "a0955e9e",
     "name": "Rayon de Sentinelle",
     "us": 0,
     "them": 1,
     "me": 0,
     "sq": {}
    }
   ]
  },
  "padsEmptied": 29,
  "gridCols": [
   {
    "d": "07/09",
    "h": "21:26",
    "map": "Banished Narrows",
    "mode": "Strongholds",
    "outcome": 3,
    "ms": 82,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "pwk": [
     2,
     7
    ],
    "o": {
     "powerup|powerup_camo": [
      2,
      4,
      1
     ],
     "rack|c30d87c7": [
      0,
      1,
      0
     ],
     "rack|2b1824d5": [
      0,
      2,
      0
     ],
     "power|71ab0a2c": [
      0,
      4,
      0
     ],
     "power|0a1992bc": [
      1,
      3,
      0
     ]
    }
   },
   {
    "d": "07/09",
    "h": "21:34",
    "map": "Isolation",
    "mode": "Strongholds",
    "outcome": 3,
    "ms": 31,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "pwk": [
     1,
     8
    ],
    "o": {
     "rack|2b1824d5": [
      1,
      2,
      0
     ],
     "power|71ab0a2c": [
      2,
      2,
      0
     ],
     "rack|2fb21c87": [
      1,
      2,
      0
     ],
     "power|d7915565": [
      1,
      1,
      0
     ],
     "power|0a1992bc": [
      2,
      0,
      0
     ],
     "rack|fd98554c": [
      1,
      0,
      0
     ]
    }
   },
   {
    "d": "07/09",
    "h": "21:42",
    "map": "Illusion",
    "mode": "Strongholds",
    "outcome": 2,
    "ms": 200,
    "es": 183,
    "filmed": true,
    "tiers": true,
    "pwk": [
     5,
     5
    ],
    "o": {
     "rack|80977ba5": [
      3,
      2,
      0
     ],
     "rack|a0955e9e": [
      0,
      1,
      0
     ],
     "rack|84bd29ed": [
      0,
      1,
      0
     ],
     "rack|3e070217": [
      0,
      1,
      0
     ],
     "rack|30484ea6": [
      0,
      1,
      0
     ],
     "power|0a1992bc": [
      4,
      4,
      4
     ],
     "power|d7915565": [
      1,
      2,
      1
     ]
    }
   },
   {
    "d": "07/09",
    "h": "21:52",
    "map": "Fortress",
    "mode": "Strongholds",
    "outcome": 3,
    "ms": 89,
    "es": 200,
    "filmed": true,
    "tiers": true,
    "pwk": [
     4,
     7
    ],
    "o": {
     "powerup|powerup_overshield": [
      2,
      3,
      2
     ],
     "power|0a1992bc": [
      1,
      3,
      0
     ],
     "rack|2b1824d5": [
      1,
      4,
      0
     ],
     "rack|fd98554c": [
      1,
      0,
      0
     ]
    }
   },
   {
    "d": "07/09",
    "h": "22:00",
    "map": "Origin",
    "mode": "CTF",
    "outcome": 3,
    "ms": 1,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "pwk": [
     7,
     5
    ],
    "o": {
     "rack|2b1824d5": [
      1,
      1,
      0
     ],
     "power|71ab0a2c": [
      1,
      4,
      1
     ],
     "rack|fd98554c": [
      0,
      1,
      0
     ],
     "power|0a1992bc": [
      3,
      0,
      0
     ]
    }
   },
   {
    "d": "07/09",
    "h": "22:10",
    "map": "Domicile",
    "mode": "CTF",
    "outcome": 3,
    "ms": 0,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "pwk": [
     2,
     4
    ],
    "o": {
     "powerup|powerup_overshield": [
      1,
      1,
      1
     ],
     "rack|daf193c7": [
      1,
      0,
      0
     ],
     "rack|b533957e": [
      2,
      0,
      1
     ],
     "rack|3e070217": [
      1,
      0,
      0
     ],
     "power|841ac5e5": [
      1,
      1,
      0
     ],
     "power|0d20c469": [
      1,
      0,
      0
     ],
     "rack|c30d87c7": [
      0,
      1,
      0
     ]
    }
   },
   {
    "d": "07/09",
    "h": "22:14",
    "map": "Absolution",
    "mode": "CTF",
    "outcome": 3,
    "ms": 0,
    "es": 3,
    "filmed": true,
    "tiers": true,
    "pwk": [
     0,
     12
    ],
    "o": {
     "powerup|powerup_overshield": [
      3,
      1,
      1
     ],
     "rack|80977ba5": [
      2,
      0,
      0
     ],
     "rack|2b1824d5": [
      2,
      3,
      0
     ],
     "rack|b619d84a": [
      1,
      5,
      1
     ],
     "power|71ab0a2c": [
      0,
      5,
      0
     ],
     "rack|f5c335df": [
      1,
      0,
      1
     ]
    }
   }
  ],
  "effect": {
   "us": {
    "ms": 456500,
    "kills": 13
   },
   "them": {
    "ms": 398200,
    "kills": 11
   }
  },
  "pwkAll": {
   "us": 21,
   "them": 48
  },
  "pwkTiers": {
   "us": 21,
   "them": 48
  },
  "equip": [
   {
    "key": "grapple",
    "label": "Grappin",
    "me": null,
    "dropsMe": 4
   },
   {
    "key": "wall",
    "label": "Mur de protection",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     4
    ],
    "takenMe": 0
   },
   {
    "key": "sensor",
    "label": "Capteur de menaces",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "translocator_beacon",
    "label": "Translocateur",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "shroud_screen",
    "label": "Écran occultant",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     1,
     1
    ],
    "takenMe": 0
   },
   {
    "key": "threat_seeker",
    "label": "Traqueur de menaces",
    "me": [
     0,
     0,
     1
    ],
    "rest": [
     0,
     1,
     1
    ],
    "takenMe": 1
   },
   {
    "key": "repair_field",
    "label": "Champ de réparation",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "thruster",
    "label": "Propulseur",
    "me": null,
    "dropsMe": 1
   }
  ],
  "vies": {
   "near": [
    53,
    35
   ],
   "alone": [
    10,
    11
   ],
   "noctx": 1,
   "nomate": 0
  },
  "vehRows": 0,
  "sessKills": 50,
  "sessMeasured": 50
 },
 "solo": {
  "key": "solo",
  "tag": "solo",
  "label": "Solo",
  "title": "soirée du 22 septembre 2026, début de soirée",
  "squad": [],
  "ctx": "solo",
  "n": 6,
  "filmed": 6,
  "tiersMatches": 6,
  "wins": 4,
  "losses": 2,
  "span": "19:27 → 20:19",
  "modes": [
   [
    "Super Fiesta:Slayer",
    6
   ]
  ],
  "matches": [
   {
    "d": "22/09",
    "h": "19:27",
    "map": "Cliffhanger",
    "mode": "Super Fiesta:Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 35,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 0
   },
   {
    "d": "22/09",
    "h": "19:36",
    "map": "Recharge",
    "mode": "Super Fiesta:Slayer",
    "outcome": 3,
    "ms": 42,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 0
   },
   {
    "d": "22/09",
    "h": "19:46",
    "map": "Catalyst",
    "mode": "Super Fiesta:Slayer",
    "outcome": 3,
    "ms": 40,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 0
   },
   {
    "d": "22/09",
    "h": "19:57",
    "map": "Streets",
    "mode": "Super Fiesta:Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 45,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 7,
    "friends": 0
   },
   {
    "d": "22/09",
    "h": "20:06",
    "map": "Streets",
    "mode": "Super Fiesta:Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 43,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 7,
    "friends": 0
   },
   {
    "d": "22/09",
    "h": "20:19",
    "map": "Recharge",
    "mode": "Super Fiesta:Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 33,
    "filmed": true,
    "tiers": true,
    "teamSize": 4,
    "lobbySize": 8,
    "friends": 0
   }
  ],
  "coord": {
   "measured": 6,
   "total": 6,
   "delaiMs": 2411,
   "couvert": [
    10,
    58
   ],
   "riposte": [
    17,
    255
   ],
   "teamAvenged": 48,
   "parityR": 25,
   "prep": [
    18,
    69
   ],
   "part": [
    18,
    51
   ],
   "parityA": 25,
   "perMatch": [
    [
     0,
     35,
     2,
     7,
     3,
     4
    ],
    [
     1,
     50,
     4,
     8,
     4,
     4
    ],
    [
     2,
     50,
     0,
     3,
     1,
     4
    ],
    [
     3,
     45,
     5,
     9,
     4,
     4
    ],
    [
     4,
     43,
     6,
     14,
     5,
     4
    ],
    [
     5,
     32,
     0,
     10,
     1,
     4
    ]
   ]
  },
  "sunburst": {
   "shoulder": {
    "special": 13,
    "precision": 8,
    "automatic": 13,
    "shotgun": 2
   },
   "melee": {
    "melee": 5
   },
   "heavy": {
    "special": 10,
    "power": 6
   },
   "sidearm": {
    "sidearm": 11
   },
   "grenade": {
    "grenade_dynamo": 3
   },
   "unattributed": {
    "unattributed": 1
   }
  },
  "fragBy": {
   "2533274823110022": {
    "shoulder": 36,
    "melee": 5,
    "heavy": 16,
    "sidearm": 11,
    "grenade": 3,
    "unattributed": 1
   }
  },
  "sheet": {
   "2533274823110022": 72
  },
  "filmKills": {
   "2533274823110022": 71
  },
  "tools": [
   {
    "key": "w:Needler",
    "label": "Needler",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 13
    },
    "total": 13
   },
   {
    "key": "w:Rayon de Sentinelle",
    "label": "Rayon de Sentinelle",
    "cls": "heavy",
    "by": {
     "2533274823110022": 10
    },
    "total": 10
   },
   {
    "key": "w:Ravageur",
    "label": "Ravageur",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 8
    },
    "total": 8
   },
   {
    "key": "w:Fusil traqueur",
    "label": "Fusil traqueur",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 7
    },
    "total": 7
   },
   {
    "key": "w:Disrupteur",
    "label": "Disrupteur",
    "cls": "sidearm",
    "by": {
     "2533274823110022": 5
    },
    "total": 5
   },
   {
    "key": "melee",
    "label": "Mêlée",
    "cls": "melee",
    "by": {
     "2533274823110022": 5
    },
    "total": 5
   },
   {
    "key": "w:MK50 Sidekick",
    "label": "MK50 Sidekick",
    "cls": "sidearm",
    "by": {
     "2533274823110022": 5
    },
    "total": 5
   },
   {
    "key": "w:VK78 Commando",
    "label": "VK78 Commando",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 4
    },
    "total": 4
   },
   {
    "key": "g:grenade_dynamo",
    "label": "Grenade dynamo",
    "cls": "grenade",
    "by": {
     "2533274823110022": 3
    },
    "total": 3
   },
   {
    "key": "w:M41 SPNKr",
    "label": "M41 SPNKr",
    "cls": "heavy",
    "by": {
     "2533274823110022": 3
    },
    "total": 3
   },
   {
    "key": "w:CQS48 Bulldog",
    "label": "CQS48 Bulldog",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 2
    },
    "total": 2
   },
   {
    "key": "w:Épée à énergie",
    "label": "Épée à énergie",
    "cls": "heavy",
    "by": {
     "2533274823110022": 2
    },
    "total": 2
   },
   {
    "key": "w:BR75",
    "label": "BR75",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:Carabine à impulsion",
    "label": "Carabine à impulsion",
    "cls": "shoulder",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:Marteau antigravité",
    "label": "Marteau antigravité",
    "cls": "heavy",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "w:Pistolet à plasma",
    "label": "Pistolet à plasma",
    "cls": "sidearm",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   },
   {
    "key": "unattributed",
    "label": "Non attribué",
    "cls": "unattributed",
    "by": {
     "2533274823110022": 1
    },
    "total": 1
   }
  ],
  "usage": {
   "measured": 6,
   "total": 6,
   "teamSizeAvg": 4,
   "lobbySizeAvg": 7.67,
   "teamParity": 25,
   "lobbyParity": 13.04,
   "equip": [
    {
     "key": "overshield_episodes",
     "kind": "overshield",
     "label": "Surbouclier",
     "player": 0,
     "team": 0,
     "lobby": 0,
     "cad": {
      "me": 0,
      "sq": {},
      "team": 0,
      "lobby": 0
     },
     "shareTeam": null,
     "shareLobby": null,
     "teamOfLobby": null,
     "aboveTeam": 0
    },
    {
     "key": "equipment_powerup_camo",
     "kind": "equipment",
     "label": "Camouflage",
     "player": 26,
     "team": 87,
     "lobby": 183,
     "cad": {
      "me": 4.33,
      "sq": {},
      "team": 14.5,
      "lobby": 30.5
     },
     "shareTeam": 29.89,
     "shareLobby": 14.21,
     "teamOfLobby": 47.54,
     "aboveTeam": 5,
     "outcomes": {
      "used": 26,
      "kept": 0,
      "dropped": 0,
      "tmRate": 100,
      "opRate": 100,
      "tm": [
       61,
       0,
       0
      ],
      "op": [
       96,
       0,
       0
      ]
     }
    },
    {
     "key": "equipment_sensor",
     "kind": "equipment",
     "label": "Capteur de menaces",
     "player": 16,
     "team": 63,
     "lobby": 130,
     "cad": {
      "me": 2.67,
      "sq": {},
      "team": 10.5,
      "lobby": 21.67
     },
     "shareTeam": 25.4,
     "shareLobby": 12.31,
     "teamOfLobby": 48.46,
     "aboveTeam": 3,
     "outcomes": {
      "used": 0,
      "kept": 0,
      "dropped": 16,
      "tmRate": 8.51,
      "opRate": 4.48,
      "tm": [
       4,
       0,
       43
      ],
      "op": [
       3,
       0,
       64
      ]
     }
    },
    {
     "key": "equipment_wall",
     "kind": "equipment",
     "label": "Mur de protection",
     "player": 15,
     "team": 68,
     "lobby": 142,
     "cad": {
      "me": 2.5,
      "sq": {},
      "team": 11.33,
      "lobby": 23.67
     },
     "shareTeam": 22.06,
     "shareLobby": 10.56,
     "teamOfLobby": 47.89,
     "aboveTeam": 2,
     "outcomes": {
      "used": 9,
      "kept": 0,
      "dropped": 6,
      "tmRate": 60.38,
      "opRate": 43.24,
      "tm": [
       32,
       0,
       21
      ],
      "op": [
       32,
       0,
       42
      ]
     }
    },
    {
     "key": "grapple_pulls",
     "kind": "grapple",
     "label": "Grappin",
     "player": 13,
     "team": 72,
     "lobby": 147,
     "cad": {
      "me": 2.17,
      "sq": {},
      "team": 12,
      "lobby": 24.5
     },
     "shareTeam": 18.06,
     "shareLobby": 8.84,
     "teamOfLobby": 48.98,
     "aboveTeam": 2
    }
   ],
   "pad": {
    "key": "pad_pickups",
    "kind": "pads",
    "label": "Toutes armes spéciales",
    "player": 0,
    "team": 0,
    "lobby": 0,
    "cad": {
     "me": 0,
     "sq": {},
     "team": 0,
     "lobby": 0
    },
    "shareTeam": null,
    "shareLobby": null,
    "teamOfLobby": null,
    "aboveTeam": 0
   },
   "padFamilies": [],
   "track": {
    "me": 0,
    "sq": {},
    "rest": 0,
    "them": 0
   },
   "tiers": {},
   "tierNotes": {
    "measured": 6,
    "noPads": 6,
    "unest": 0,
    "unclassified": 0
   }
  },
  "objAvant": null,
  "balance": [],
  "roleTot": {
   "me": [
    0,
    0,
    0
   ],
   "rest": [
    0,
    0,
    0
   ]
  },
  "dominant": {
   "me": null,
   "rest": null
  },
  "control": {
   "powerup": {
    "us": 0,
    "them": 0,
    "me": 0,
    "lostUs": 0,
    "lostThem": 0
   },
   "power": {
    "us": 0,
    "them": 0,
    "me": 0,
    "lostUs": 0,
    "lostThem": 0
   },
   "rack": {
    "us": 0,
    "them": 0,
    "me": 0,
    "lostUs": 0,
    "lostThem": 0
   }
  },
  "objects": {
   "powerup": [],
   "power": [],
   "rack": []
  },
  "padsEmptied": 0,
  "gridCols": [
   {
    "d": "22/09",
    "h": "19:27",
    "map": "Cliffhanger",
    "mode": "Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 35,
    "filmed": true,
    "tiers": true,
    "pwk": [
     15,
     10
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "19:36",
    "map": "Recharge",
    "mode": "Slayer",
    "outcome": 3,
    "ms": 42,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "pwk": [
     13,
     22
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "19:46",
    "map": "Catalyst",
    "mode": "Slayer",
    "outcome": 3,
    "ms": 40,
    "es": 50,
    "filmed": true,
    "tiers": true,
    "pwk": [
     9,
     20
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "19:57",
    "map": "Streets",
    "mode": "Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 45,
    "filmed": true,
    "tiers": true,
    "pwk": [
     19,
     13
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "20:06",
    "map": "Streets",
    "mode": "Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 43,
    "filmed": true,
    "tiers": true,
    "pwk": [
     20,
     21
    ],
    "o": {}
   },
   {
    "d": "22/09",
    "h": "20:19",
    "map": "Recharge",
    "mode": "Slayer",
    "outcome": 2,
    "ms": 50,
    "es": 33,
    "filmed": true,
    "tiers": true,
    "pwk": [
     15,
     15
    ],
    "o": {}
   }
  ],
  "effect": {
   "us": {
    "ms": 804900,
    "kills": 50
   },
   "them": {
    "ms": 705000,
    "kills": 36
   }
  },
  "pwkAll": {
   "us": 91,
   "them": 101
  },
  "pwkTiers": {
   "us": 91,
   "them": 101
  },
  "equip": [
   {
    "key": "grapple",
    "label": "Grappin",
    "me": null,
    "dropsMe": 17
   },
   {
    "key": "wall",
    "label": "Mur de protection",
    "me": [
     9,
     0,
     6
    ],
    "rest": [
     32,
     0,
     21
    ],
    "takenMe": 0
   },
   {
    "key": "sensor",
    "label": "Capteur de menaces",
    "me": [
     0,
     0,
     16
    ],
    "rest": [
     4,
     0,
     43
    ],
    "takenMe": 2
   },
   {
    "key": "translocator_beacon",
    "label": "Translocateur",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "shroud_screen",
    "label": "Écran occultant",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "threat_seeker",
    "label": "Traqueur de menaces",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "repair_field",
    "label": "Champ de réparation",
    "me": [
     0,
     0,
     0
    ],
    "rest": [
     0,
     0,
     0
    ],
    "takenMe": 0
   },
   {
    "key": "thruster",
    "label": "Propulseur",
    "me": null,
    "dropsMe": 13
   }
  ],
  "vies": {
   "near": [
    53,
    46
   ],
   "alone": [
    5,
    11
   ],
   "noctx": 1,
   "nomate": 0
  },
  "vehRows": 0,
  "sessKills": 72,
  "sessMeasured": 65
 }
}```

## 3. Agrégats ajoutés en v2 (vue Comparaison, calculés dans la page depuis vm_s.json)

Parts du camp : C = mon camp / (mon camp + adversaire) ; F = moi / mon camp ; J = somme des actions d’un rôle (Tenir en secondes) ; K = moi / mon camp ; B = frags d’un outil / mes frags (feuille de match).

### s2209

| ressource | mon camp–adversaire | part de mon camp | moi / mon camp | ma part |
|---|---|---|---|---|
| powerup | 12–8 | 60 % | 3 / 12 | 25 % |
| power | 13–21 | 38.2 % | 7 / 13 | 53.8 % |
| rack | 24–30 | 44.4 % | 3 / 24 | 12.5 % |

Bonus perdus : mon camp 16.7 % (2/12), adversaire 25 % (2/8).

B, 6 premiers outils (part de mes 65 frags) : BR75 22 (33.8 %) · MK50 Sidekick 13 (20 %) · M41 SPNKr 11 (16.9 %) · Mêlée 6 (9.2 %) · Grenade frag 2 (3.1 %) · Bandit EVO 2 (3.1 %)

J, par rôle et par famille (mon camp–adversaire, part) :
- Drapeau : take 22–21 (51.2 %) ; defend 33–33 (50 %) ; hold 137.5–111.3 (55.3 %) ; prises nettes 22–26 (45.8 %), moi 7

K, pied (ma part de mon camp) : Prendre 31.8 % (7/22), Défendre 30.3 % (10/33), Tenir 47.2 % (65 s / 138 s).

### s0709

| ressource | mon camp–adversaire | part de mon camp | moi / mon camp | ma part |
|---|---|---|---|---|
| powerup | 8–9 | 47.1 % | 5 / 8 | 62.5 % |
| power | 18–29 | 38.3 % | 6 / 18 | 33.3 % |
| rack | 19–28 | 40.4 % | 3 / 19 | 15.8 % |

Bonus perdus : mon camp 0 % (0/8), adversaire 11.1 % (1/9).

B, 6 premiers outils (part de mes 50 frags) : MK50 Sidekick 28 (56 %) · Mêlée 8 (16 %) · M41 SPNKr 3 (6 %) · Grenade dynamo 2 (4 %) · MA5K Avenger 2 (4 %) · MA40 AR 1 (2 %)

J, par rôle et par famille (mon camp–adversaire, part) :
- Bases : take 93–109 (46 %) ; defend 30–60 (33.3 %) ; hold 1067.8–926.1 (53.6 %)
- Drapeau : take 17–38 (30.9 %) ; defend 41–52 (44.1 %) ; hold 170.5–344.9 (33.1 %) ; prises nettes 29–41 (41.4 %), moi 12

K, pied (ma part de mon camp) : Prendre 25.5 % (28/110), Défendre 45.1 % (32/71), Tenir 27.6 % (342 s / 1238 s).

### solo

| ressource | mon camp–adversaire | part de mon camp | moi / mon camp | ma part |
|---|---|---|---|---|
| powerup | 0–0 | — | 0 / 0 | — |
| power | 0–0 | — | 0 / 0 | — |
| rack | 0–0 | — | 0 / 0 | — |

Bonus perdus : mon camp — (0/0), adversaire — (0/0).

B, 6 premiers outils (part de mes 72 frags) : Needler 13 (18.1 %) · Rayon de Sentinelle 10 (13.9 %) · Ravageur 8 (11.1 %) · Fusil traqueur 7 (9.7 %) · Disrupteur 5 (6.9 %) · Mêlée 5 (6.9 %)

J, K : aucun match à objectif.
