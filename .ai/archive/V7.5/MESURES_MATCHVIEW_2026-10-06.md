# Mesures — maquette « Vue match », avant / après (2026-10-06)

Base : COPIE `<scratchpad>\db\shared_matches_v2.duckdb` (copiée le 2026-10-05 18:26), lue par `diag_q.exe` (lecture seule). Vues `_latest` seules (et `match_participants`, `match_registry`, `highlight_events`, tables sans version). Aucun artefact de rejeu lu.

Chaîne : `extract_m.sh <clé> <match_id>` → `ex/<clé>_*.tsv` → `compute_m.js` → `vm_m.json` → `build_m.js` (+ `head_m.part`, `body_m.part`) → `maquette_matchview.html` ; validation `validate_m.js` → `val_m.json`.

Matchs : `m2209` = `ab526724-3684-4335-b759-a18edcccc137` ; `m2407` = `4f77afc1-d9f4-443b-b38f-3628341bf7e9`. Joueur de la page : JGtm `2533274823110022`.

## 1. Requêtes

Préfixe commun (CTE) :

```sql
WITH sc AS (SELECT r.*, p.team_id AS my_team, p.outcome AS my_outcome FROM match_registry r JOIN match_participants p ON p.match_id=r.match_id AND p.xuid='2533274823110022' WHERE r.match_id='<match_id>')
```

### `matches`

```sql
SELECT match_id, strftime(start_time_utc AT TIME ZONE 'Europe/Paris','%d/%m/%Y') d, strftime(start_time_utc AT TIME ZONE 'Europe/Paris','%H:%M') h, map_name, map_name_fr, pair_name, playlist_name, regexp_replace(pair_name,' on .*','') AS md, my_outcome, my_team, team_0_score, team_1_score, rounds_total, team_0_rounds_won, team_1_rounds_won, duration_seconds, player_count FROM sc
```

### `parts`

```sql
SELECT p.match_id, p.xuid, COALESCE(p.gamertag,'') gt, COALESCE(p.team_id,-1) team, COALESCE(p.present_at_completion,false)::INT pac, COALESCE(p.power_weapon_kills,-1) pwk, p.kills, p.deaths, p.assists, p.rank, p.score, COALESCE(p.melee_kills,0) melee, COALESCE(p.grenade_kills,0) gren FROM match_participants p JOIN sc USING(match_id) ORDER BY team, p.rank
```

### `usage`

```sql
SELECT u.match_id, u.xuid, u.camo_episodes, u.camo_ms, u.camo_kills, u.overshield_episodes, u.overshield_ms, u.overshield_kills, u.grapple_pulls, u.deployed_json, u.dropped_objects, u.pad_pickups, u.pad_pickups_json, u.taken_json, u.spent_json, u.kept_json, u.dropped_json FROM match_usage_players_latest u JOIN sc USING(match_id)
```

### `films`

```sql
SELECT f.match_id, f.duration_ms, f.powerup_pickups_json, f.weapon_pads_json, f.pad_occupancies, f.pad_named, f.pad_unnamed FROM match_usage_films_latest f JOIN sc USING(match_id)
```

### `tiers`

```sql
SELECT t.match_id, t.xuid, t.tier, t.weapon_family, t.pickups, t.pads_total, t.pads_confirmed, t.random_starts::INT rs FROM match_pad_pickups_by_tier_latest t JOIN sc USING(match_id)
```

### `kills`

```sql
SELECT e.match_id, e.time_ms, COALESCE(e.feed_killer_xuid,'') killer, COALESCE(e.feed_killer_gamertag,'') kgt, COALESCE(e.victim_xuid,'') victim, COALESCE(e.victim_gamertag,'') vgt, COALESCE(e.assist_xuid,'') assist, COALESCE(e.assist_gamertag,'') agt, e.assist_known::INT ak, e.killer_damage_pct kdp, e.assist_damage_pct adp, printf('%08x', e.source_tag) tag, COALESCE(e.source_category,'') cat FROM match_kill_events_latest e JOIN sc USING(match_id) WHERE e.publishable ORDER BY e.time_ms
```

### `killcov`

```sql
SELECT e.match_id, count(*) n, sum(e.publishable::INT) pub FROM match_kill_events_latest e JOIN sc USING(match_id) GROUP BY 1
```

### `lives`

```sql
SELECT l.match_id, l.xuid, l.start_ms, l.end_ms, l.end_cause FROM match_lives_latest l JOIN sc USING(match_id) ORDER BY l.xuid, l.start_ms
```

### `dctx`

```sql
SELECT d.match_id, d.victim_xuid, d.time_ms, d.nearest_teammate_m, d.teammates_visible, d.teammates_total FROM match_death_context_latest d JOIN sc USING(match_id) ORDER BY d.victim_xuid, d.time_ms
```

### `kpos`

```sql
SELECT kp.match_id, kp.killer_xuid, kp.time_ms, round(sqrt(pow(killer_x-victim_x,2)+pow(killer_y-victim_y,2)+pow(killer_z-victim_z,2)),2) dist, round(killer_z-victim_z,2) dz FROM kill_positions_latest kp JOIN sc USING(match_id) WHERE killer_x IS NOT NULL AND victim_x IS NOT NULL ORDER BY kp.time_ms
```

### `veh`

```sql
SELECT v.match_id, v.row_kind, count(*) n FROM match_vehicle_takes_latest v JOIN sc USING(match_id) GROUP BY 1,2
```

### `hl`

```sql
SELECT h.match_id, h.event_type, h.time_ms, h.xuid, h.type_hint FROM highlight_events h JOIN sc USING(match_id) ORDER BY h.time_ms
```

### `names`

```sql
SELECT x, string_agg(DISTINCT g, ' | ') gts, count(*) n FROM (SELECT victim_xuid x, victim_gamertag g FROM match_kill_events_latest e JOIN sc USING(match_id) WHERE victim_xuid IS NOT NULL AND victim_gamertag IS NOT NULL UNION ALL SELECT feed_killer_xuid, feed_killer_gamertag FROM match_kill_events_latest e JOIN sc USING(match_id) WHERE feed_killer_xuid IS NOT NULL AND feed_killer_gamertag IS NOT NULL UNION ALL SELECT assist_xuid, assist_gamertag FROM match_kill_events_latest e JOIN sc USING(match_id) WHERE assist_xuid IS NOT NULL AND assist_gamertag IS NOT NULL) GROUP BY x
```

### `kvall`

```sql
SELECT e.match_id, e.time_ms, COALESCE(e.feed_killer_xuid,'') killer, COALESCE(e.feed_killer_gamertag,'') kgt, COALESCE(e.victim_xuid,'') victim, COALESCE(e.victim_gamertag,'') vgt, e.publishable::INT pub FROM match_kill_events_latest e JOIN sc USING(match_id) ORDER BY e.time_ms
```

## 2. Contrôles ponctuels

```
-- résumé d'usage : passe, révision, schéma
match_id	summary_pass	summary_rev	artifact_schema	written_at
ab526724-3684-4335-b759-a18edcccc137	4aa1356bd5b3013d	us6	71	2026-09-25 21:23:12.419961 +0000 UTC
4f77afc1-d9f4-443b-b38f-3628341bf7e9	f16abff2333831d4	us6	71	2026-09-25 17:29:21.026602 +0000 UTC
(2 rows)

-- journal des morts : passe, révision, lignes, publiables
match_id	decode_pass	decoder_rev	n	pub
4f77afc1-d9f4-443b-b38f-3628341bf7e9	53efc84b0c5af033	killsource-2026-09-24	294	0
ab526724-3684-4335-b759-a18edcccc137	98adb3883fb03726	killsource-2026-09-24	145	145
(2 rows)

-- prises nettes de drapeau (non affichées : objectif hors périmètre)
match_id	count_star()
ab526724-3684-4335-b759-a18edcccc137	8
(1 rows)

-- véhicules
match_id	count_star()
(0 rows)

-- participants : lignes, présents à la fin, gamertag renseigné
match_id	lignes	presents	avec_gamertag
4f77afc1-d9f4-443b-b38f-3628341bf7e9	36	24	1
ab526724-3684-4335-b759-a18edcccc137	8	8	1
(2 rows)

-- prises de bonus publiées (taken_json non vide)
match_id	lignes	avec_prises
4f77afc1-d9f4-443b-b38f-3628341bf7e9	25	0
ab526724-3684-4335-b759-a18edcccc137	8	7
(2 rows)
```

## 3. Chiffres dérivés par carte (compute_m.js, extrait de vm_m_report.json)

### 22/09 — Drapeau à Starboard

- Méta : {"id":"ab526724-3684-4335-b759-a18edcccc137","d":"22/09/2026","h":"21:23","map":"Starboard","mode":"Arena:CTF","playlist":"Quick Play","outcome":2,"ms":3,"es":0,"durS":670} ; présents à la fin 8 (4 contre 4) ; journal des morts 145 publiables sur 145
- C1 badges : first_blood joueur_403 @40586 ms ; first_group_death JGtm @40586 ms ; clutch_finisher JOUEUR_1012 @665662 ms ; last_group_kill JGtm @116315 ms ; top_gun JOUEUR_1012 @215198 ms ; top_killer JOUEUR_1012 ; kamikaze JOUEUR_1012 @665744 ms ; badges d’équipement : {"camo":[],"overshield":[],"undetermined":[]}
- A / A1 frags de JGtm par classe : {"shoulder":1,"sidearm":7,"melee":2,"grenade":1} (feuille 11, film 11)
- B / A2 outils : MK50 Sidekick 7 ; Mêlée 2 ; Grenade frag 1 ; VK78 Commando 1
- C / A3 distance : 140 frags mesurés ; MK50 Sidekick (69) : joueur_117 n=16 min 2.21 moy 8.2 max 14.86, joueur_403 n=14 min 1.14 moy 7.7 max 17.19, joueur_621 n=10 min 2.49 moy 7.5 max 15.32, JGtm n=7 min 2.77 moy 4.6 max 7.79, joueur_1011 n=7 min 3.27 moy 8.5 max 14.47, joueur_165 n=6 min 2.51 moy 6 max 11.74, joueur_1010 n=5 min 4.52 moy 7.2 max 11.49, JOUEUR_1012 n=4 min 4.68 moy 9.7 max 11.97 | MA40 AR (25) : JOUEUR_1012 n=16 min 0.93 moy 3.8 max 13.56, joueur_621 n=3 min 4.5 moy 6.2 max 8.9, joueur_165 n=2 min 1.51 moy 2.8 max 4, joueur_117 n=2 min 2.19 moy 3.2 max 4.17, joueur_1011 n=1 min 4 moy 4 max 4, joueur_1010 n=1 min 3.1 moy 3.1 max 3.1 | Mêlée (17) : joueur_117 n=4 min 0.15 moy 0.3 max 0.45, joueur_1011 n=3 min 0.14 moy 0.3 max 0.39, joueur_165 n=2 min 0.39 moy 0.4 max 0.39, JGtm n=2 min 0.21 moy 0.2 max 0.26, joueur_621 n=2 min 0.21 moy 0.4 max 0.66, JOUEUR_1012 n=2 min 0.39 moy 0.4 max 0.48, joueur_1010 n=1 min 0.21 moy 0.2 max 0.21, joueur_403 n=1 min 0.28 moy 0.3 max 0.28 | Grenade frag (11) : JOUEUR_1012 n=5 min 3 moy 5.8 max 8.73, joueur_1010 n=2 min 7.41 moy 8.2 max 8.94, JGtm n=1 min 15.16 moy 15.2 max 15.16, joueur_117 n=1 min 7.13 moy 7.1 max 7.13, joueur_621 n=1 min 12.43 moy 12.4 max 12.43, joueur_403 n=1 min 3.89 moy 3.9 max 3.89 | Fusil électrique (8) : JOUEUR_1012 n=6 min 1.82 moy 9.4 max 17.82, joueur_1010 n=2 min 11.27 moy 21.1 max 30.87 | Épée à énergie (4) : joueur_621 n=4 min 0.24 moy 0.4 max 0.51 | CQS48 Bulldog (3) : JOUEUR_1012 n=3 min 1.54 moy 2.2 max 2.82 | VK78 Commando (2) : JGtm n=1 min 4.06 moy 4.1 max 4.06, joueur_1010 n=1 min 22.15 moy 22.2 max 22.15 | Chute, environnement (1) : joueur_165 n=1 min 13.25 moy 13.3 max 13.25
- A4 hauteur : frags de JGtm mesurés 11/11
- A5 / F colonnes : Écran occultant, Surbouclier ; lignes : JGtm g=0 [[0,1,1],[2,0,0]] ; JOUEUR_1012 g=0 [[0,0,2],[0,0,0]] ; joueur_117 g=0 [[0,0,0],[1,0,0]] ; joueur_165 g=0 [[0,1,2],[2,0,0]] ; joueur_621 g=0 [[0,2,2],[0,0,1]] ; joueur_403 g=0 [[0,0,2],[0,0,1]] ; joueur_1010 g=0 [[0,1,0],[0,0,0]] ; joueur_1011 g=0 [[0,0,0],[0,0,0]]
- A6 part de chaque équipe : {"grapple":{"us":0,"them":0},"eq":[{"us":7,"them":7},{"us":5,"them":2}]}
- D contrôle : {"powerup":{"us":5,"them":2,"lostUs":0,"lostThem":2},"power":{"us":0,"them":2,"lostUs":0,"lostThem":0},"rack":{"us":4,"them":3,"lostUs":0,"lostThem":0},"base":{"us":0,"them":0,"lostUs":0,"lostThem":0}} ; canal des prises de bonus : true ; socles de bonus vidés {"camo":0,"overshield":10} ; emplacement non identifié {"us":0,"them":0}
- D objets : powerup = Surbouclier 5–2 | power = Épée à énergie 0–2 | rack = Fusil électrique 2–2, VK78 Commando 2–0, CQS48 Bulldog 0–1 | base = 
- A7 lignes de niveau (prises > 0) : terrain Fusil électrique 2806336078533919 2 ; puissance Épée à énergie 2749978711717053 2 ; terrain CQS48 Bulldog 2533906527081383 1 ; terrain VK78 Commando 2563513065873397 1 ; terrain Fusil électrique 2563513065873397 2 ; terrain VK78 Commando 2533274823110022 1 ; occupations {"occ":27,"named":9,"unnamed":8}
- E fiches (mon camp, humains présents) : JGtm, JOUEUR_1012, joueur_117, joueur_165 ; powerup : Surbouclier camp 5 [2,0,1,2] gardé [0,0,0,0] lâché [0,0,0,0] | power :  | rack : Fusil électrique camp 2 [0,2,0,0] gardé [0,0,0,0] lâché [0,0,0,0] ; VK78 Commando camp 2 [1,1,0,0] gardé [0,0,0,0] lâché [0,0,0,0]
- G effet : {"us":{"ms":71600,"kills":2,"eps":5},"them":{"ms":0,"kills":0,"eps":0}} ; frags aux armes spéciales (feuille) {"us":0,"them":4} ; lignes véhicules 0
- I vies (seuil 18 m, frags lus dans match_kill_events_latest publiables) : JGtm près 11/8 seul 2/0 sans contexte 0 sans coéquipier situé 0 (vies-mort 13, vies 16) ; JOUEUR_1012 près 9/30 seul 1/1 sans contexte 0 sans coéquipier situé 0 (vies-mort 10, vies 11) ; joueur_117 près 14/16 seul 3/6 sans contexte 0 sans coéquipier situé 0 (vies-mort 17, vies 20) ; joueur_165 près 13/10 seul 2/1 sans contexte 0 sans coéquipier situé 0 (vies-mort 15, vies 17)
- J3 assistances : morts mesurées 145 ; paires joueur_1011→joueur_403 4 (volées 3, part 65) ; joueur_1011→joueur_621 4 (volées 1, part 48) ; joueur_117→JOUEUR_1012 4 (volées 0, part 35) ; joueur_165→JOUEUR_1012 4 (volées 1, part 35) ; joueur_165→joueur_117 6 (volées 2, part 44) ; joueur_1010→joueur_1011 3 (volées 3, part 53) ; joueur_1010→joueur_403 3 (volées 2, part 42) ; joueur_621→joueur_403 3 (volées 1, part 37) ; joueur_403→joueur_621 3 (volées 1, part 37) ; JGtm→joueur_165 2 (volées 2, part 65) ; joueur_117→JGtm 4 (volées 1, part 50) ; JGtm→JOUEUR_1012 2 (volées 2, part 50) ; joueur_403→joueur_1011 1 (volées 1, part 81) ; joueur_1010→joueur_621 2 (volées 1, part 56) ; joueur_621→joueur_1010 1 (volées 1, part 87) ; JOUEUR_1012→joueur_117 6 (volées 4, part 62) ; joueur_165→JGtm 2 (volées 1, part 40) ; JOUEUR_1012→joueur_165 4 (volées 2, part 53) ; JGtm→joueur_117 1 (volées 1, part 46) ; joueur_403→joueur_1010 2 (volées 2, part 51) ; joueur_1011→joueur_1010 1 (volées 0, part 20)
- J4 riposte (avant) : journal 145 lignes dont 145 publiables ; vengées 38 ; mon camp JOUEUR_1012 1/10, joueur_117 7/4, JGtm 6/3, joueur_165 5/2 ; adversaire joueur_621 2/8, joueur_403 4/5, joueur_1010 7/3, joueur_1011 6/3

### 24/07 — Drapeau à Flood Gulch (BTB)

- Méta : {"id":"4f77afc1-d9f4-443b-b38f-3628341bf7e9","d":"24/07/2026","h":"22:20","map":"Flood Gulch","mode":"BTB:CTF","playlist":"Big Team Battle","outcome":1,"ms":1,"es":1,"durS":1194} ; présents à la fin 24 (12 contre 12) ; journal des morts 0 publiables sur 294
- C1 badges : first_blood joueur_117 @136435 ms ; first_group_death joueur_582 @138868 ms ; last_group_kill joueur_297 @510822 ms ; top_gun joueur_257 @497980 ms ; top_killer joueur_317 ; kamikaze JOUEUR_409 @945318 ms ; badges d’équipement : {"camo":[],"overshield":[],"undetermined":[]}
- A / A1 frags de JGtm par classe : {"unattributed":10} (feuille 10, film 0)
- B / A2 outils : Non attribué 10
- C / A3 distance : 0 frags mesurés ; 
- A4 hauteur : frags de JGtm mesurés 0/10
- A5 / F colonnes : Grappin, Mur de protection, Écran occultant, Camouflage, Surbouclier ; lignes : JGtm g=0 [[0,0,0],[0,0,1],[0,0,0],[0,0,0]] ; joueur_811 g=0 [[2,0,0],[0,0,1],[0,0,0],[0,0,0]] ; joueur_317 g=0 [[0,0,0],[0,0,1],[0,0,0],[0,0,0]] ; joueur_257 g=1 [[1,0,0],[0,0,2],[1,0,0],[0,0,0]] ; joueur_117 g=0 [[0,0,0],[0,0,0],[0,0,0],[4,0,0]] ; JOUEUR_126 g=0 [[0,0,1],[0,0,1],[0,0,0],[0,0,0]] ; JOUEUR_409 g=8 [[0,0,1],[0,0,1],[0,0,0],[0,0,0]] ; joueur_427 g=0 [[1,0,3],[0,0,1],[0,0,0],[0,0,0]] ; joueur_582 g=0 [[0,0,0],[0,0,2],[0,0,0],[0,0,0]] ; joueur_706 g=4 [[0,0,0],[0,0,1],[1,0,0],[1,0,0]] ; Bot 59 — ; joueur_297 g=2 [[1,0,0],[0,0,0],[0,0,0],[0,0,0]] ; joueur_1379 (parti) g=0 [[0,0,0],[0,0,0],[0,0,0],[0,0,0]] ; joueur_728 g=6 [[1,0,0],[0,0,0],[4,0,0],[0,0,0]] ; joueur_580 g=0 [[1,0,1],[0,0,2],[0,0,0],[0,0,0]] ; joueur_615 g=0 [[0,0,1],[0,0,0],[0,0,0],[0,0,0]] ; joueur_655 g=7 [[0,0,0],[0,0,0],[0,0,0],[0,0,0]] ; JOUEUR_390 g=1 [[0,0,1],[0,0,0],[3,0,0],[0,0,0]] ; JOUEUR_221 g=2 [[0,0,0],[0,0,2],[0,0,0],[0,0,0]] ; JOUEUR_136 g=0 [[0,0,0],[0,0,1],[0,0,0],[0,0,0]] ; joueur_017 g=0 [[3,0,1],[0,0,0],[0,0,0],[0,0,0]] ; joueur_218 g=0 [[0,0,0],[0,0,0],[0,0,0],[0,0,0]] ; joueur_172 g=0 [[0,0,0],[0,0,1],[0,0,1],[0,0,0]] ; JOUEUR_469 (parti) g=0 [[0,0,0],[0,0,0],[0,0,0],[0,0,0]] ; joueur_430 g=0 [[1,0,0],[0,0,0],[0,0,0],[0,0,0]] ; Bot 23 — ; JOUEUR_456 (parti) g=0 [[0,0,0],[0,0,1],[0,0,0],[0,0,0]]
- A6 part de chaque équipe : {"grapple":{"us":15,"them":16},"eq":[{"us":10,"them":10},{"us":11,"them":7},{"us":2,"them":8},{"us":5,"them":0}]}
- D contrôle : {"powerup":{"us":0,"them":0,"lostUs":0,"lostThem":0},"power":{"us":19,"them":11,"lostUs":0,"lostThem":0},"rack":{"us":29,"them":40,"lostUs":0,"lostThem":0},"base":{"us":0,"them":0,"lostUs":0,"lostThem":0}} ; canal des prises de bonus : false ; socles de bonus vidés {"camo":18,"overshield":19} ; emplacement non identifié {"us":19,"them":12}
- D objets : powerup =  | power = S7 Sniper 11–8, SPNKr à combustible 8–3 | rack = BR75 8–17, Disrupteur 14–4, CQS48 Bulldog 2–12, MA5K Avenger 5–2, Carabine à impulsion 0–3, Pistolet à plasma 0–2 | base = 
- A7 lignes de niveau (prises > 0) : terrain Disrupteur 2710930876332218 1 ; terrain Carabine à impulsion 2710930876332218 1 ; non_classe M41 SPNKr 2710930876332218 2 ; terrain MA5K Avenger 2640661141794626 2 ; terrain CQS48 Bulldog 2640661141794626 1 ; terrain Disrupteur 2640661141794626 1 ; terrain BR75 2640661141794626 1 ; puissance SPNKr à combustible 2640661141794626 1 ; non_classe Ravageur 2640661141794626 1 ; non_classe Hydra 2640661141794626 1 ; terrain CQS48 Bulldog 2784870293334143 3 ; terrain Disrupteur 2784870293334143 1 ; terrain BR75 2784870293334143 4 ; puissance S7 Sniper 2779781155078810 4 ; non_classe Ravageur 2779781155078810 1 ; terrain Pistolet à plasma 2646046920107160 1 ; terrain BR75 2646046920107160 4 ; terrain MA5K Avenger 2769104854931374 1 ; terrain CQS48 Bulldog 2769104854931374 3 ; terrain Carabine à impulsion 2769104854931374 1 ; puissance S7 Sniper 2769104854931374 5 ; non_classe Ravageur 2769104854931374 1 ; terrain BR75 2753091070419728 2 ; non_classe Ravageur 2753091070419728 1 ; terrain MA5K Avenger 2690056610860898 1 ; terrain CQS48 Bulldog 2690056610860898 4 ; puissance SPNKr à combustible 2690056610860898 3 ; puissance S7 Sniper 2690056610860898 1 ; terrain CQS48 Bulldog 2615815077049243 1 ; terrain Disrupteur 2615815077049243 8 ; terrain BR75 2615815077049243 1 ; puissance S7 Sniper 2615815077049243 4 ; terrain MA5K Avenger 2634990399857364 1 ; terrain Disrupteur 2642222702413791 2 ; terrain BR75 2642222702413791 1 ; terrain Disrupteur 2632700264829909 2 ; non_classe Hydra 2632700264829909 2 ; terrain BR75 2575401242986491 2 ; non_classe Hydra 2575401242986491 1 ; terrain Disrupteur 2685352648935266 1 ; terrain BR75 2685352648935266 1 ; puissance SPNKr à combustible 2685352648935266 3 ; terrain MA5K Avenger 2550886075837404 1 ; puissance S7 Sniper 2550886075837404 3 ; non_classe Ravageur 2550886075837404 4 ; non_classe Hydra 2550886075837404 1 ; non_classe M41 SPNKr 2550886075837404 1 ; terrain CQS48 Bulldog 2563350602769669 1 ; terrain BR75 2563350602769669 1 ; terrain BR75 2793075222215887 2 ; puissance S7 Sniper 2793075222215887 2 ; non_classe Hydra 2793075222215887 3 ; non_classe M41 SPNKr 2793075222215887 3 ; terrain Disrupteur 2623439693412443 1 ; terrain BR75 2623439693412443 1 ; non_classe M41 SPNKr 2623439693412443 2 ; terrain MA5K Avenger 2542494231387188 1 ; terrain BR75 2542494231387188 2 ; puissance SPNKr à combustible 2542494231387188 1 ; non_classe Ravageur 2542494231387188 3 ; non_classe Hydra 2542494231387188 2 ; terrain Disrupteur 2533274823110022 1 ; puissance SPNKr à combustible 2533274823110022 3 ; terrain Pistolet à plasma 2573321465012002 1 ; terrain BR75 2573321465012002 3 ; non_classe Hydra 2573321465012002 2 ; terrain CQS48 Bulldog 2788951673476170 1 ; terrain Carabine à impulsion 2788951673476170 1 ; occupations {"occ":317,"named":130,"unnamed":150}
- E fiches (mon camp, humains présents) : JGtm, joueur_811, joueur_317, joueur_257, joueur_117, JOUEUR_126, JOUEUR_409, joueur_427, joueur_582, joueur_706, joueur_297 ; powerup :  | power : S7 Sniper camp 11 [0,0,4,0,0,0,3,4,0,0,0] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0] ; SPNKr à combustible camp 8 [3,0,0,3,0,0,0,0,1,0,1] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0] | rack : Disrupteur camp 14 [1,0,0,1,1,0,0,8,1,2,0] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0] ; BR75 camp 8 [0,2,0,1,1,0,0,1,1,0,2] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0] ; MA5K Avenger camp 5 [0,0,0,0,0,1,1,0,2,0,1] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0] ; CQS48 Bulldog camp 2 [0,0,0,0,0,0,0,1,1,0,0] gardé [0,0,0,0,0,0,0,0,0,0,0] lâché [0,0,0,0,0,0,0,0,0,0,0]
- G effet : {"us":{"ms":166700,"kills":0,"eps":7},"them":{"ms":216900,"kills":0,"eps":7}} ; frags aux armes spéciales (feuille) {"us":23,"them":38} ; lignes véhicules 0
- I vies (seuil 30 m, frags lus dans highlight_events) : JGtm près 10/8 seul 0/0 sans contexte 0 sans coéquipier situé 0 (vies-mort 10, vies 12) ; joueur_811 près 5/6 seul 2/4 sans contexte 0 sans coéquipier situé 0 (vies-mort 7, vies 8) ; joueur_317 près 2/2 seul 2/12 sans contexte 0 sans coéquipier situé 0 (vies-mort 4, vies 11) ; joueur_257 près 4/12 seul 3/7 sans contexte 0 sans coéquipier situé 0 (vies-mort 7, vies 9) ; joueur_117 près 11/9 seul 3/2 sans contexte 0 sans coéquipier situé 0 (vies-mort 14, vies 15) ; JOUEUR_126 près 4/3 seul 0/0 sans contexte 1 sans coéquipier situé 1 (vies-mort 6, vies 11) ; JOUEUR_409 près 6/7 seul 1/1 sans contexte 0 sans coéquipier situé 0 (vies-mort 7, vies 8) ; joueur_427 près 4/3 seul 1/2 sans contexte 0 sans coéquipier situé 1 (vies-mort 6, vies 9) ; joueur_582 près 13/6 seul 0/0 sans contexte 0 sans coéquipier situé 0 (vies-mort 13, vies 16) ; joueur_706 près 7/7 seul 0/0 sans contexte 0 sans coéquipier situé 0 (vies-mort 7, vies 8) ; joueur_297 près 9/1 seul 3/3 sans contexte 1 sans coéquipier situé 0 (vies-mort 13, vies 15)
- J3 assistances : morts mesurées 0 ; paires 
- J4 riposte (avant) : journal 294 lignes dont 0 publiables ; vengées 32 ; mon camp joueur_317 2/2, joueur_427 1/2, JOUEUR_409 1/2, joueur_706 1/2, joueur_811 0/2, joueur_257 0/2, joueur_297 0/1, JGtm 3/0, joueur_582 3/0, JOUEUR_126 1/0, joueur_117 1/0 ; adversaire joueur_580 5/6, joueur_728 1/5, joueur_655 2/2, joueur_615 2/1, joueur_017 1/1, JOUEUR_390 1/1, joueur_218 3/0, joueur_172 3/0, JOUEUR_221 1/0, JOUEUR_136 0/0, joueur_430 0/0

## 4. Résultats bruts

### m2209 · `matches` (1 lignes)

```
match_id	d	h	map_name	map_name_fr	pair_name	playlist_name	md	my_outcome	my_team	team_0_score	team_1_score	rounds_total	team_0_rounds_won	team_1_rounds_won	duration_seconds	player_count
ab526724-3684-4335-b759-a18edcccc137	22/09/2026	21:23	Starboard	NULL	Arena:CTF on Starboard	Quick Play	Arena:CTF	2	1	0	3	1	0	1	670	0
```

### m2209 · `parts` (8 lignes)

```
match_id	xuid	gt	team	pac	pwk	kills	deaths	assists	rank	score	melee	gren
ab526724-3684-4335-b759-a18edcccc137	2806336078533919		0	1	0	13	22	8	5	1920	1	2
ab526724-3684-4335-b759-a18edcccc137	2749978711717053		0	1	4	21	16	7	5	2470	3	2
ab526724-3684-4335-b759-a18edcccc137	2533906527081383		0	1	0	11	23	9	5	1660	3	0
ab526724-3684-4335-b759-a18edcccc137	2582035846398048		0	1	0	16	23	7	5	2025	1	1
ab526724-3684-4335-b759-a18edcccc137	2533274823110022		1	1	0	11	15	6	1	2390	2	1
ab526724-3684-4335-b759-a18edcccc137	2563513065873397		1	1	0	38	11	11	2	4475	2	4
ab526724-3684-4335-b759-a18edcccc137	2623439693412443		1	1	0	24	19	10	2	3210	4	1
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	joueur_165	1	1	0	11	16	12	2	1945	2	0
```

### m2209 · `usage` (8 lignes)

```
match_id	xuid	camo_episodes	camo_ms	camo_kills	overshield_episodes	overshield_ms	overshield_kills	grapple_pulls	deployed_json	dropped_objects	pad_pickups	pad_pickups_json	taken_json	spent_json	kept_json	dropped_json
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	0	0	0	2	43800	1	0	{}	1	1	{"fd98554c":1}	{"powerup_overshield":2,"shroud_screen":2}	{"powerup_overshield":1}	{"powerup_overshield":0,"shroud_screen":1}	{"shroud_screen":1}
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	0	0	0	0	0	0	0	{}	3	3	{"9387a8b9":2,"fd98554c":1}	{"shroud_screen":2}	{}	{"shroud_screen":0}	{"repulsor":1,"shroud_screen":2}
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	0	0	0	1	5500	0	0	{}	1	0	{}	{"powerup_overshield":1}	{"powerup_overshield":1}	{"powerup_overshield":0}	{"repulsor":1}
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	0	0	0	0	0	0	0	{}	4	0	{}	{"powerup_overshield":1,"shroud_screen":1}	{}	{"powerup_overshield":0,"shroud_screen":0}	{"powerup_overshield":1,"repulsor":1,"shroud_screen":2}
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	0	0	0	0	0	0	0	{}	1	1	{"b619d84a":1}	{}	{}	{}	{"repulsor":1}
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	0	0	0	0	0	0	0	{}	3	2	{"4ff3937e":2}	{"powerup_overshield":1,"shroud_screen":4}	{}	{"powerup_overshield":0,"shroud_screen":2}	{"powerup_overshield":1,"shroud_screen":2}
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	0	0	0	0	0	0	0	{}	0	2	{"9387a8b9":2}	{"shroud_screen":1}	{}	{"shroud_screen":1}	{}
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	0	0	0	2	22300	1	0	{}	2	0	{}	{"powerup_overshield":2,"shroud_screen":3}	{}	{"powerup_overshield":0,"shroud_screen":1}	{"shroud_screen":2}
```

### m2209 · `films` (1 lignes)

```
match_id	duration_ms	powerup_pickups_json	weapon_pads_json	pad_occupancies	pad_named	pad_unnamed
ab526724-3684-4335-b759-a18edcccc137	645800	{"powerup_overshield":10}	[{"weapon":"b619d84a","occupations":3,"named":1},{"weapon":"4ff3937e","occupations":5,"named":2},{"weapon":"9387a8b9","occupations":4,"named":4},{"weapon":"fd98554c","occupations":3,"named":1},{"weapon":"fd98554c","occupations":2,"named":1}]	27	9	8
```

### m2209 · `tiers` (9 lignes)

```
match_id	xuid	tier	weapon_family	pickups	pads_total	pads_confirmed	rs
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	aucune_prise		0	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	terrain	9387a8b9	2	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	puissance	4ff3937e	2	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	terrain	b619d84a	1	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	aucune_prise		0	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	aucune_prise		0	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	terrain	fd98554c	1	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	terrain	9387a8b9	2	6	6	0
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	terrain	fd98554c	1	6	6	0
```

### m2209 · `kills` (145 lignes)

```
match_id	time_ms	killer	kgt	victim	vgt	assist	agt	ak	kdp	adp	tag	cat
ab526724-3684-4335-b759-a18edcccc137	40586	2582035846398048	joueur_403	2533274823110022	JGtm	2533906527081383	joueur_1011	1	69	30	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	41671	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	100	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	51681	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	100	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	56469	2749978711717053	joueur_621	2623439693412443	joueur_117	2533906527081383	joueur_1011	1	123	44	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	57786	2749978711717053	joueur_621	2547408409520575	joueur_165			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	59071	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	2623439693412443	joueur_117	1	59	40	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	61756	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	2547408409520575	joueur_165	1	30	60	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	72201	2623439693412443	joueur_117	2806336078533919	joueur_1010	2547408409520575	joueur_165	1	59	40	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	77540	2533906527081383	joueur_1011	2623439693412443	joueur_117	2806336078533919	joueur_1010	1	28	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	80610	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	2547408409520575	joueur_165	1	79	20	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	90470	2547408409520575	joueur_165	2533906527081383	joueur_1011			1	93	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	94023	2582035846398048	joueur_403	2533274823110022	JGtm	2806336078533919	joueur_1010	1	39	45	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	101766	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1	59	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	110276	2582035846398048	joueur_403	2623439693412443	joueur_117	2749978711717053	joueur_621	1	49	50	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	112243	2806336078533919	joueur_1010	2547408409520575	joueur_165			1	99	NULL	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	116315	2533274823110022	JGtm	2533906527081383	joueur_1011			1	99	NULL	1f11df6b	Headshot
ab526724-3684-4335-b759-a18edcccc137	118066	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	122153	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1	69	30	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	134900	2623439693412443	joueur_117	2582035846398048	joueur_403			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	137502	2749978711717053	joueur_621	2547408409520575	joueur_165			1	100	NULL	698aacdb	SilentMelee
ab526724-3684-4335-b759-a18edcccc137	140956	2749978711717053	joueur_621	2533274823110022	JGtm			1	100	NULL	698aacdb	None
ab526724-3684-4335-b759-a18edcccc137	141074	2806336078533919	joueur_1010	2623439693412443	joueur_117			1	100	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	148079	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	150566	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1	79	20	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	156455	2547408409520575	joueur_165	2582035846398048	joueur_403	2533274823110022	JGtm	1	31	58	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	157206	2806336078533919	joueur_1010	2547408409520575	joueur_165			1	100	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	172138	2623439693412443	joueur_117	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	172905	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012			1	100	NULL	698aacdb	None
ab526724-3684-4335-b759-a18edcccc137	177877	2533274823110022	JGtm	2582035846398048	joueur_403	2623439693412443	joueur_117	1	59	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	181847	2749978711717053	joueur_621	2547408409520575	joueur_165			1	100	NULL	698aacdb	None
ab526724-3684-4335-b759-a18edcccc137	188554	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	191823	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	2533274823110022	JGtm	1	39	60	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	194895	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1	4	89	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	198064	2749978711717053	joueur_621	2533274823110022	JGtm	2582035846398048	joueur_403	1	79	20	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	214949	2806336078533919	joueur_1010	2623439693412443	joueur_117			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	215198	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	100	NULL	daa03c35	SilentMelee
ab526724-3684-4335-b759-a18edcccc137	215864	2547408409520575	joueur_165	2582035846398048	joueur_403	2533274823110022	JGtm	1	20	71	00403594	None
ab526724-3684-4335-b759-a18edcccc137	217018	2533274823110022	JGtm	2749978711717053	joueur_621			1	79	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	217049	2533906527081383	joueur_1011	2533274823110022	JGtm	2582035846398048	joueur_403	1	18	81	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	222956	2533906527081383	joueur_1011	2547408409520575	joueur_165			1	99	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	224890	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	2547408409520575	joueur_165	1	69	30	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	230931	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	89	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	231247	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1	18	81	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	242777	2749978711717053	joueur_621	2623439693412443	joueur_117			1	100	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	255706	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	261494	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012			1	84	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	265014	2623439693412443	joueur_117	2582035846398048	joueur_403	2547408409520575	joueur_165	1	18	71	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	266182	2749978711717053	joueur_621	2547408409520575	joueur_165	2806336078533919	joueur_1010	1	69	30	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	268486	2533274823110022	JGtm	2806336078533919	joueur_1010			1	99	NULL	daa03c35	SilentMelee
ab526724-3684-4335-b759-a18edcccc137	270955	2749978711717053	joueur_621	2533274823110022	JGtm	2533906527081383	joueur_1011	1	59	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	274992	2623439693412443	joueur_117	2533906527081383	joueur_1011			1	100	NULL	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	278813	2582035846398048	joueur_403	2623439693412443	joueur_117	2533906527081383	joueur_1011	1	33	66	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	279695	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	59	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	283816	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	99	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	284552	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	294913	2547408409520575	joueur_165	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	298683	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1	12	87	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	299367	2582035846398048	joueur_403	2547408409520575	joueur_165	2533906527081383	joueur_1011	1	5	122	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	299518	2623439693412443	joueur_117	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1	7	92	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	304389	2806336078533919	joueur_1010	2533274823110022	JGtm			1	43	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	305673	2623439693412443	joueur_117	2806336078533919	joueur_1010			1	89	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	309093	2582035846398048	joueur_403	2623439693412443	joueur_117			1	89	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	309096	2623439693412443	joueur_117	2582035846398048	joueur_403			1	100	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	324391	2623439693412443	joueur_117	2749978711717053	joueur_621			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	326612	2533274823110022	JGtm	2806336078533919	joueur_1010	2547408409520575	joueur_165	1	49	50	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	328330	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	338840	2547408409520575	joueur_165	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1	4	63	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	348517	2623439693412443	joueur_117	2806336078533919	joueur_1010			1	100	NULL	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	349251	2547408409520575	joueur_165	2533906527081383	joueur_1011	2563513065873397	JOUEUR_1012	1	53	46	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	351019	2749978711717053	joueur_621	2623439693412443	joueur_117			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	352187	2533274823110022	JGtm	2749978711717053	joueur_621	2623439693412443	joueur_117	1	54	30	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	364851	2547408409520575	joueur_165	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	1	53	46	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	367269	2623439693412443	joueur_117	2582035846398048	joueur_403	2533274823110022	JGtm	1	18	46	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	367869	2533906527081383	joueur_1011	2547408409520575	joueur_165			1	100	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	371023	2749978711717053	joueur_621	2623439693412443	joueur_117			1	100	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	374993	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	89	NULL	daa03c35	SilentMelee
ab526724-3684-4335-b759-a18edcccc137	386038	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	386437	2623439693412443	joueur_117	2582035846398048	joueur_403	2547408409520575	joueur_165	1	69	30	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	390593	2806336078533919	joueur_1010	2547408409520575	joueur_165			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	390692	2749978711717053	joueur_621	2533274823110022	JGtm			1	99	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	393261	2623439693412443	joueur_117	2749978711717053	joueur_621			1	89	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	398434	2533906527081383	joueur_1011	2623439693412443	joueur_117	2806336078533919	joueur_1010	1	39	60	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	399368	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	2623439693412443	joueur_117	1	39	30	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	408227	2547408409520575	joueur_165	2533906527081383	joueur_1011			1	100	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	412947	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012			1	100	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	414868	2623439693412443	joueur_117	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1	18	51	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	416936	2623439693412443	joueur_117	2806336078533919	joueur_1010	2547408409520575	joueur_165	1	59	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	420673	2623439693412443	joueur_117	2749978711717053	joueur_621	2547408409520575	joueur_165	1	79	20	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	425328	2533906527081383	joueur_1011	2533274823110022	JGtm			1	59	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	428915	2533906527081383	joueur_1011	2547408409520575	joueur_165			1	100	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	428917	2547408409520575	joueur_165	2533906527081383	joueur_1011			1	89	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	434268	2582035846398048	joueur_403	2623439693412443	joueur_117			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	436122	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	99	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	438358	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	100	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	443529	2533906527081383	joueur_1011	2533274823110022	JGtm			1	100	NULL	daa03c35	None
ab526724-3684-4335-b759-a18edcccc137	449500	2623439693412443	joueur_117	2533906527081383	joueur_1011	2563513065873397	JOUEUR_1012	1	12	87	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	451938	2806336078533919	joueur_1010	2623439693412443	joueur_117	2582035846398048	joueur_403	1	39	60	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	455926	2582035846398048	joueur_403	2547408409520575	joueur_165	2806336078533919	joueur_1010	1	37	62	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	458312	2582035846398048	joueur_403	2533274823110022	JGtm	2806336078533919	joueur_1010	1	79	20	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	466321	2623439693412443	joueur_117	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1	26	73	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	466587	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	99	NULL	038ad14c	None
ab526724-3684-4335-b759-a18edcccc137	482036	2533906527081383	joueur_1011	2623439693412443	joueur_117			1	90	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	482820	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	485371	2533274823110022	JGtm	2533906527081383	joueur_1011	2623439693412443	joueur_117	1	8	91	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	488159	2533274823110022	JGtm	2582035846398048	joueur_403	2547408409520575	joueur_165	1	69	30	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	494698	2806336078533919	joueur_1010	2533274823110022	JGtm			1	92	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	497318	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	82	NULL	038ad14c	None
ab526724-3684-4335-b759-a18edcccc137	515534	2623439693412443	joueur_117	2533906527081383	joueur_1011	2563513065873397	JOUEUR_1012	1	63	36	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	516389	2582035846398048	joueur_403	2547408409520575	joueur_165			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	519858	2623439693412443	joueur_117	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	1	69	30	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	521193	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	99	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	524228	2582035846398048	joueur_403	2623439693412443	joueur_117			1	89	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	527899	2533274823110022	JGtm	2582035846398048	joueur_403	2623439693412443	joueur_117	1	59	40	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	538475	2806336078533919	joueur_1010	2547408409520575	joueur_165			1	100	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	540262	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	2547408409520575	joueur_165	1	69	30	038ad14c	None
ab526724-3684-4335-b759-a18edcccc137	542381	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	2623439693412443	joueur_117	1	59	40	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	553759	2582035846398048	joueur_403	2533274823110022	JGtm			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	555311	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	2533274823110022	JGtm	1	39	40	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	561033	2533906527081383	joueur_1011	2623439693412443	joueur_117			1	87	NULL	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	567257	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	100	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	573296	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	575932	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1	59	20	1f11df6b	Headshot
ab526724-3684-4335-b759-a18edcccc137	585942	2582035846398048	joueur_403	2547408409520575	joueur_165	2749978711717053	joueur_621	1	79	20	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	588328	2623439693412443	joueur_117	2582035846398048	joueur_403			1	89	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	592315	2749978711717053	joueur_621	2623439693412443	joueur_117	2582035846398048	joueur_403	1	48	61	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	596135	2533274823110022	JGtm	2806336078533919	joueur_1010			1	103	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	600306	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	605710	2749978711717053	joueur_621	2533274823110022	JGtm			1	54	NULL	da3b5ba4	None
ab526724-3684-4335-b759-a18edcccc137	607898	2547408409520575	joueur_165	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1	43	56	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	614838	2623439693412443	joueur_117	2806336078533919	joueur_1010			1	100	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	616656	2533906527081383	joueur_1011	2623439693412443	joueur_117	2806336078533919	joueur_1010	1	39	60	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	616957	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	625865	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	630236	2533274823110022	JGtm	2749978711717053	joueur_621			1	100	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	630239	2749978711717053	joueur_621	2533274823110022	JGtm			1	100	NULL	2ff92041	None
ab526724-3684-4335-b759-a18edcccc137	632022	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403			1	100	NULL	7ff0849c	Headshot
ab526724-3684-4335-b759-a18edcccc137	637944	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010			1	100	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	645202	2547408409520575	joueur_165	2749978711717053	joueur_621			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	648028	2623439693412443	joueur_117	2533906527081383	joueur_1011	2547408409520575	joueur_165	1	38	61	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	651415	2623439693412443	joueur_117	2582035846398048	joueur_403			1	99	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	654468	2806336078533919	joueur_1010	2547408409520575	joueur_165	2582035846398048	joueur_403	1	39	42	bb1a095b	None
ab526724-3684-4335-b759-a18edcccc137	655568	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	2623439693412443	joueur_117	1	69	30	7ff0849c	None
ab526724-3684-4335-b759-a18edcccc137	662943	2749978711717053	joueur_621	2623439693412443	joueur_117			1	36	NULL	bb1a095b	Headshot
ab526724-3684-4335-b759-a18edcccc137	665662	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621			1	89	NULL	0000b238	None
ab526724-3684-4335-b759-a18edcccc137	665744	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1	38	40	bb1a095b	Headshot
```

### m2209 · `killcov` (1 lignes)

```
match_id	n	pub
ab526724-3684-4335-b759-a18edcccc137	145	145
```

### m2209 · `lives` (150 lignes)

```
match_id	xuid	start_ms	end_ms	end_cause
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	30893	40720	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	50780	94158	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	104219	141089	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	151149	198199	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	208259	217186	film_end
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	227245	271088	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	281149	304522	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	314582	390827	film_end
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	400886	425461	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	435522	443663	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	453724	458445	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	468506	494832	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	504893	553893	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	563953	605846	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	615906	630370	death
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	640464	675974	film_end
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	30359	101899	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	111960	122287	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	132347	150699	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	160760	173039	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	183099	195028	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	205089	231382	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	241444	261629	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	271690	298817	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	308877	413082	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	423142	576065	death
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	586126	665881	film_end
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	30308	56603	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	57154	57154	cut
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	66662	77674	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	87734	110409	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	120468	141207	film_end
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	151267	214715	film_end
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	225142	242911	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	252970	278947	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	289007	309227	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	319287	351153	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	361213	371157	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	381217	398567	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	408628	434403	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	444464	452073	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	462133	482170	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	492229	524361	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	534423	561167	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	571227	592449	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	602509	616791	death
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	626850	663076	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	30174	51815	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	61874	80744	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	90804	118200	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	128310	135034	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	145077	156588	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	166650	178011	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	188104	216001	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	226060	265150	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	275209	284686	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	294746	309227	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	319321	338974	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	349034	367403	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	377496	386573	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	396632	415001	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	425061	436256	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	446316	466454	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	476513	488292	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	499054	528033	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	538093	555444	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	565504	588461	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	598521	617090	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	627150	632156	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	642215	651548	death
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	661608	670467	film_end
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	30675	59205	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	69265	90604	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	100665	116447	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	126508	148213	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	158274	172271	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	182331	191958	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	202636	225025	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	235086	255840	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	266501	275126	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	285186	295047	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	305106	328464	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	339057	349384	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	359444	386172	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	396231	408361	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	418421	429049	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	439142	449637	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	459696	485506	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	496117	515670	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	525730	542515	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	553225	573430	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	583490	600440	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	611117	625999	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	636843	648161	death
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	658221	675872	film_end
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	38684	61891	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	71952	217152	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	227211	279831	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	289892	299651	film_end
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	309711	324527	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	334720	352321	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	362381	375127	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	385188	393395	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	403606	420523	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	430867	431050	film_end
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	493014	493181	film_end
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	531386	567390	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	577450	608031	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	618241	630370	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	640431	645336	death
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	655402	665796	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	32427	41804	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	51864	72336	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	82395	188689	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	198866	215333	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	225393	231065	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	241125	268619	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	278680	283952	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	294012	305807	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	315868	326745	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	336805	348651	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	358710	364983	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	375044	399502	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	409563	417070	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	427130	438491	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	448553	466721	film_end
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	485640	497452	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	507512	519991	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	530051	540395	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	550456	596269	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	606496	614971	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	625548	638078	death
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	648146	655702	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	30492	57920	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	67981	112377	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	122437	137637	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	147679	157340	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	167400	181981	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	192042	223090	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	233151	265887	film_end
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	276377	299501	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	309560	368003	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	378063	390727	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	400786	429049	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	439108	456060	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	466120	516521	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	526581	538611	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	548671	586076	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	596136	654601	death
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	664662	673503	film_end
```

### m2209 · `dctx` (143 lignes)

```
match_id	victim_xuid	time_ms	nearest_teammate_m	teammates_visible	teammates_total
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	40586	12.18	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	94023	2.64	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	140956	7.56	2	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	198064	6.74	2	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	217049	8.34	2	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	270955	0.61	1	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	304389	9.84	1	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	390692	7.6	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	425328	2.1	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	443529	18.04	2	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	458312	20.64	1	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	494698	8.86	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	553759	0.53	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	605710	10.62	3	3
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	630239	2.84	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	101766	13.9	2	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	122153	18.93	2	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	150566	7.81	1	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	172905	7.57	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	194895	15.92	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	231247	15.7	2	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	261494	4.93	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	298683	4.35	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	412947	3.51	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	575932	2.38	3	3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	665744	10.04	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	56469	2.1	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	77540	0.4	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	110276	13.87	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	141074	8.02	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	214949	2.41	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	242777	17.82	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	278813	14.38	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	309093	23.41	1	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	351019	2.28	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	371023	4.14	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	398434	5.53	1	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	434268	18.96	1	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	451938	3.36	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	482036	11.9	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	524228	11.07	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	561033	24.04	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	592315	16.37	2	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	616656	6.82	3	3
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	662943	4.02	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	51681	13.21	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	80610	11.69	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	118066	10.62	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	134900	16.53	3	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	156455	12.36	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	177877	26.43	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	215864	3.7	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	265014	4.76	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	284552	NULL	0	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	309096	12.35	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	338840	10.34	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	367269	11.12	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	386437	9.27	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	414868	3.59	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	436122	7.97	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	466321	5.9	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	488159	19.7	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	527899	19.48	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	555311	26.13	2	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	588328	8.23	3	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	616957	10.39	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	632022	15.24	1	3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	651415	14.08	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	59071	3.02	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	90470	4.89	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	116315	14.08	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	148079	8.1	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	172138	6.05	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	191823	5.54	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	224890	NULL	0	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	255706	7.66	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	274992	9.4	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	294913	3.54	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	328330	3.84	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	349251	3.75	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	386038	12.36	3	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	408227	13.72	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	428917	12.37	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	449500	5.1	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	485371	5.7	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	515534	17.08	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	542381	21.86	1	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	573296	11	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	600306	3.68	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	625865	8.51	2	3
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	648028	3.55	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	61756	23.29	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	217018	13.69	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	279695	8.92	2	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	299518	5.05	2	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	324391	13.14	3	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	352187	24.85	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	374993	13.46	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	393261	12.33	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	420673	20.15	1	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	567257	16.48	3	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	607898	11.56	2	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	630236	8.25	2	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	645202	1.08	2	3
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	665662	13.06	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	41671	10.62	3	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	72201	17.18	3	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	188554	3.83	3	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	215198	10.41	3	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	230931	8.57	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	268486	11.19	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	283816	0.61	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	305673	8.93	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	326612	3.29	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	348517	5.17	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	364851	6.9	3	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	399368	14.99	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	416936	5.82	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	438358	NULL	0	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	466587	26.18	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	497318	19.36	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	519858	4.98	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	540262	6.31	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	596135	25.86	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	614838	7.86	2	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	637944	18.86	1	3
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	655568	9.58	1	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	57786	4.17	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	112243	20.45	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	137502	4.04	3	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	157206	5.06	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	181847	28.25	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	222956	12.7	1	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	266182	14.42	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	299367	3.51	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	367869	7.03	3	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	390593	6.41	3	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	428915	8.46	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	455926	3.82	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	516389	13.02	3	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	538475	3.98	3	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	585942	10.1	2	3
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	654468	1.72	3	3
```

### m2209 · `kpos` (140 lignes)

```
match_id	killer_xuid	time_ms	dist	dz
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	40586	7.88	-0.44
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	41671	11.58	0.11
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	51681	11.97	0
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	56469	4.5	1.05
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	57786	8.9	1.05
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	59071	4.18	0.86
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	61756	3.96	-0.83
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	72201	0.45	0.28
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	77540	10.13	-0.53
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	80610	6.56	-0.12
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	90470	4.52	-0.43
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	94023	13.91	-0.66
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	101766	6.18	-0.76
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	110276	1.14	-0.69
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	112243	0.21	-0.01
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	116315	4.06	0.72
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	118066	5.32	1.22
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	122153	0.66	-0.65
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	134900	6.12	0
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	137502	0.37	-0.06
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	140956	0.51	0.08
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	141074	30.87	2
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	148079	2.32	0
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	150566	6.7	0
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	156455	6.34	1.4
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	157206	11.27	1.22
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	172138	2.19	0
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	172905	0.38	-0.06
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	177877	3.99	1.05
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	181847	0.24	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	188554	2.15	0.11
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	191823	3	1.71
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	194895	7.38	0.15
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	198064	9.58	1.38
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	215198	0.39	-0.11
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	215864	13.25	11.04
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	217018	2.77	-0.03
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	217049	14.47	0.77
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	222956	4	-0.32
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	224890	8.72	-0.05
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	230931	7.11	0.87
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	231247	15.32	0.91
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	242777	7.43	1.16
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	255706	1.81	0.1
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	261494	8.94	-2.02
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	265014	14.3	-0.84
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	268486	0.26	-0.03
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	270955	9.22	2.72
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	274992	0.15	0.01
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	278813	7.72	0.1
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	279695	8.73	-0.31
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	283816	4.73	0.87
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	284552	0.93	0
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	294913	4	-0.23
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	298683	4.52	-1.76
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	299367	11.25	-1.13
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	299518	4.17	0.17
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	304389	7.36	1.91
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	305673	2.21	-0.01
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	309093	0.28	0.07
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	309096	0.28	-0.07
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	324391	14.16	1.05
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	326612	7.79	2.02
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	328330	4.68	3.38
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	338840	0.39	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	348517	0.3	-0.01
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	349251	1.51	-0.38
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	351019	2.8	-0.31
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	352187	3.08	0.46
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	364851	5.22	-0.76
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	367269	10.16	-2.19
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	367869	0.14	-0.01
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	371023	7.14	3.37
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	374993	0.48	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	386038	2.15	1.04
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	386437	13.03	1.39
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	390593	4.58	1.54
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	390692	5.05	1.45
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	393261	6.73	0.43
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	398434	10.71	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	399368	4.68	-0.48
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	408227	2.51	-0.3
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	412947	3.89	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	414868	7.64	-0.94
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	416936	6.72	0.59
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	425328	7.84	-0.58
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	428915	0.39	0
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	428917	0.39	0
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	434268	17.19	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	436122	1.16	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	438358	10.68	0
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	443529	0.26	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	449500	7.13	0
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	451938	11.49	0.38
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	455926	6.02	-0.81
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	458312	5.22	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	466321	7.18	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	466587	2.82	0
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	482036	3.27	-0.21
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	485371	15.16	2.18
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	488159	5.03	2.71
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	494698	3.1	-1.15
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	497318	1.54	0.44
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	515534	14.86	-0.58
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	516389	3.66	-0.9
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	519858	7.07	-0.15
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	524228	3.69	0.83
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	527899	5.22	0
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	538475	7.41	1.38
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	540262	2.12	0
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	542381	1.82	0.77
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	553759	7.5	0.98
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	555311	7.88	-0.12
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	561033	6.38	-1.88
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	567257	6.76	0.05
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	573296	17.82	0.73
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	575932	22.15	-1.67
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	585942	3.17	1.11
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	588328	3.44	1.21
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	592315	2.49	-0.66
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	596135	4.32	-0.3
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	600306	13.56	1.4
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	605710	12.43	0.24
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	607898	11.74	-1.68
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	614838	4.37	0.82
ab526724-3684-4335-b759-a18edcccc137	2533906527081383	616656	6.93	0.58
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	616957	1.35	-0.09
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	625865	2.76	0.31
ab526724-3684-4335-b759-a18edcccc137	2533274823110022	630236	0.21	0.01
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	630239	0.21	-0.01
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	632022	15.02	1.9
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	637944	3.11	0.9
ab526724-3684-4335-b759-a18edcccc137	2547408409520575	645202	5.86	0
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	648028	7.7	-0.48
ab526724-3684-4335-b759-a18edcccc137	2623439693412443	651415	5.41	0.8
ab526724-3684-4335-b759-a18edcccc137	2806336078533919	654468	8.21	1.4
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	655568	7.32	2.26
ab526724-3684-4335-b759-a18edcccc137	2749978711717053	662943	7.15	1.9
ab526724-3684-4335-b759-a18edcccc137	2563513065873397	665662	1.93	0
ab526724-3684-4335-b759-a18edcccc137	2582035846398048	665744	13.8	1.39
```

### m2209 · `veh` (0 lignes)

```
match_id	row_kind	n
```

### m2209 · `hl` (394 lignes)

```
match_id	event_type	time_ms	xuid	type_hint
ab526724-3684-4335-b759-a18edcccc137	kill	40586	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	40586	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	41671	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	41671	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	51681	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	51681	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	56469	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	56469	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	57786	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	57786	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	medal	57787	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	kill	59071	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	59071	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	61756	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	61756	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	61756	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	mode	65228	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	72201	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	72201	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	mode	76822	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	mode	76823	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	77540	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	77540	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	80610	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	80610	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	80610	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	kill	90470	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	90470	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	94023	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	94023	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	medal	101765	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	kill	101766	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	101766	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	110276	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	110276	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	mode	110375	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	112243	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	112243	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	112246	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	116315	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	116315	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	118066	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	118066	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	118466	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	122153	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	122153	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	134900	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	134900	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	136501	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	137502	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	137502	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	140956	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	140956	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	medal	140956	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	kill	141074	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	141074	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	141074	2806336078533919	100
ab526724-3684-4335-b759-a18edcccc137	mode	141078	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	148079	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	148079	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	148080	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	medal	150565	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	kill	150566	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	150566	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	156455	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	156455	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	157206	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	157206	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	161510	2533906527081383	10
ab526724-3684-4335-b759-a18edcccc137	kill	172138	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	172138	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	172905	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	172905	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	medal	172905	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	kill	177877	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	177877	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	179545	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	181847	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	181847	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	183883	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	188554	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	188554	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	191823	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	191823	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	191824	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	kill	194895	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	194895	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	198064	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	198064	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	medal	198064	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	mode	200984	2749978711717053	10
ab526724-3684-4335-b759-a18edcccc137	mode	211162	2749978711717053	10
ab526724-3684-4335-b759-a18edcccc137	kill	214949	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	214949	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	214949	2806336078533919	100
ab526724-3684-4335-b759-a18edcccc137	mode	214952	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	215198	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	215198	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	215199	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	kill	215864	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	215864	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	217018	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	kill	217018	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	217018	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	217049	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	217049	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	222956	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	222956	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	224890	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	224890	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	230181	2563513065873397	10
ab526724-3684-4335-b759-a18edcccc137	mode	230298	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	230931	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	230931	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	231247	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	231247	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	mode	232132	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	mode	232133	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	242777	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	242777	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	242777	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	kill	255706	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	255706	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	255940	2582035846398048	10
ab526724-3684-4335-b759-a18edcccc137	kill	261494	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	261494	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	265014	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	265014	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	265019	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	266182	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	266182	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	266867	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	268486	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	268486	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	268486	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	kill	270955	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	270955	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	medal	270955	2749978711717053	100
ab526724-3684-4335-b759-a18edcccc137	mode	272340	2533906527081383	10
ab526724-3684-4335-b759-a18edcccc137	kill	274992	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	274992	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	274995	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	278813	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	278813	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	278813	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	kill	279695	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	279695	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	mode	283468	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	283816	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	283817	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	283817	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	mode	283822	2563513065873397	10
ab526724-3684-4335-b759-a18edcccc137	kill	284552	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	284552	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	284553	2563513065873397	150
ab526724-3684-4335-b759-a18edcccc137	mode	291342	2563513065873397	10
ab526724-3684-4335-b759-a18edcccc137	mode	291342	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	294913	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	294913	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	296381	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	298683	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	298683	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	299367	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	299367	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	299370	2582035846398048	10
ab526724-3684-4335-b759-a18edcccc137	kill	299518	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	299518	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	mode	301737	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	304389	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	304389	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	mode	304393	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	kill	305673	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	305674	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	309093	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	309094	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	309096	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	309096	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	316668	2533906527081383	10
ab526724-3684-4335-b759-a18edcccc137	kill	324391	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	324391	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	326612	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	326612	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	328330	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	328330	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	338840	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	338840	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	348517	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	348517	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	349251	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	349251	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	351019	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	351019	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	352187	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	352187	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	mode	361914	2582035846398048	10
ab526724-3684-4335-b759-a18edcccc137	kill	364851	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	364851	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	367269	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	367269	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	367270	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	mode	367273	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	367869	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	367870	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	371023	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	371023	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	374993	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	374993	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	374993	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	mode	375127	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	386038	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	386038	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	386437	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	386438	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	390593	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	390593	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	390692	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	390692	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	393261	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	393261	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	398434	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	398434	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	399368	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	399368	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	408227	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	408227	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	408227	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	kill	412947	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	412947	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	414868	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	414868	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	416936	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	416936	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	416937	2623439693412443	100
ab526724-3684-4335-b759-a18edcccc137	kill	420673	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	420673	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	420673	2623439693412443	150
ab526724-3684-4335-b759-a18edcccc137	mode	424594	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	kill	425328	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	425328	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	428915	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	428915	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	medal	428915	2533906527081383	100
ab526724-3684-4335-b759-a18edcccc137	kill	428917	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	428917	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	434268	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	434269	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	436122	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	436122	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	438358	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	438358	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	438358	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	mode	443296	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	443529	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	443529	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	mode	443533	2533906527081383	10
ab526724-3684-4335-b759-a18edcccc137	kill	449500	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	449500	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	450887	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	451938	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	451938	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	451939	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	kill	455926	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	455926	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	458312	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	458312	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	medal	458312	2582035846398048	100
ab526724-3684-4335-b759-a18edcccc137	kill	466321	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	466321	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	466587	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	466587	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	482036	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	482036	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	medal	482036	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	kill	482820	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	482820	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	482820	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	medal	482820	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	kill	485371	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	485371	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	488158	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	kill	488159	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	488159	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	488159	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	medal	488159	2533274823110022	100
ab526724-3684-4335-b759-a18edcccc137	kill	494698	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	494698	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	497318	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	497318	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	497318	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	kill	515534	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	515535	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	516389	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	516389	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	medal	519857	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	kill	519858	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	519858	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	519859	2623439693412443	100
ab526724-3684-4335-b759-a18edcccc137	kill	521193	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	521193	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	524228	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	524228	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	527899	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	527899	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	mode	528783	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	mode	536692	2533906527081383	10
ab526724-3684-4335-b759-a18edcccc137	kill	538475	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	538475	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	kill	540262	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	540262	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	542381	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	542381	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	542381	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	mode	542384	2563513065873397	10
ab526724-3684-4335-b759-a18edcccc137	mode	547802	2563513065873397	10
ab526724-3684-4335-b759-a18edcccc137	mode	547802	2623439693412443	10
ab526724-3684-4335-b759-a18edcccc137	mode	548453	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	553759	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	553760	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	555311	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	555311	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	561032	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	kill	561033	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	561034	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	567257	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	567257	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	567257	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	medal	567257	2563513065873397	150
ab526724-3684-4335-b759-a18edcccc137	kill	573296	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	573296	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	medal	573296	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	medal	575931	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	kill	575932	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	575932	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	kill	585942	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	585942	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	medal	588327	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	kill	588328	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	588328	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	592315	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	592315	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	596135	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	596135	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	600306	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	600306	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	605710	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	605710	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	607898	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	607898	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	614838	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	614838	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	kill	616656	2533906527081383	50
ab526724-3684-4335-b759-a18edcccc137	death	616656	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	616957	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	616957	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	kill	625865	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	625865	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	kill	630236	2533274823110022	50
ab526724-3684-4335-b759-a18edcccc137	death	630236	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	630239	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	630239	2533274823110022	20
ab526724-3684-4335-b759-a18edcccc137	kill	632022	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	632022	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	632023	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	kill	637944	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	637944	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	medal	637944	2563513065873397	100
ab526724-3684-4335-b759-a18edcccc137	kill	645202	2547408409520575	50
ab526724-3684-4335-b759-a18edcccc137	death	645202	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	kill	648028	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	648028	2533906527081383	20
ab526724-3684-4335-b759-a18edcccc137	mode	649112	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	kill	651415	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	death	651415	2582035846398048	20
ab526724-3684-4335-b759-a18edcccc137	medal	651416	2623439693412443	100
ab526724-3684-4335-b759-a18edcccc137	kill	654468	2806336078533919	50
ab526724-3684-4335-b759-a18edcccc137	death	654468	2547408409520575	20
ab526724-3684-4335-b759-a18edcccc137	mode	654471	2806336078533919	10
ab526724-3684-4335-b759-a18edcccc137	medal	655568	2623439693412443	50
ab526724-3684-4335-b759-a18edcccc137	kill	655568	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	655568	2806336078533919	20
ab526724-3684-4335-b759-a18edcccc137	mode	658038	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	kill	662943	2749978711717053	50
ab526724-3684-4335-b759-a18edcccc137	death	662943	2623439693412443	20
ab526724-3684-4335-b759-a18edcccc137	kill	665662	2563513065873397	50
ab526724-3684-4335-b759-a18edcccc137	death	665663	2749978711717053	20
ab526724-3684-4335-b759-a18edcccc137	medal	665743	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	kill	665744	2582035846398048	50
ab526724-3684-4335-b759-a18edcccc137	death	665744	2563513065873397	20
ab526724-3684-4335-b759-a18edcccc137	mode	668281	2547408409520575	10
ab526724-3684-4335-b759-a18edcccc137	mode	668282	2533274823110022	10
ab526724-3684-4335-b759-a18edcccc137	medal	668295	2533274823110022	150
ab526724-3684-4335-b759-a18edcccc137	medal	668295	2623439693412443	150
ab526724-3684-4335-b759-a18edcccc137	medal	668295	2563513065873397	150
ab526724-3684-4335-b759-a18edcccc137	medal	668295	2547408409520575	150
```

### m2209 · `names` (8 lignes)

```
x	gts	n
2623439693412443	joueur_117	51
2533906527081383	joueur_1011	43
2582035846398048	joueur_403	45
2547408409520575	joueur_165	39
2533274823110022	JGtm	31
2563513065873397	JOUEUR_1012	59
2806336078533919	joueur_1010	43
2749978711717053	joueur_621	41
```

### m2209 · `kvall` (145 lignes)

```
match_id	time_ms	killer	kgt	victim	vgt	pub
ab526724-3684-4335-b759-a18edcccc137	40586	2582035846398048	joueur_403	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	41671	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	51681	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	56469	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	57786	2749978711717053	joueur_621	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	59071	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	61756	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	72201	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	77540	2533906527081383	joueur_1011	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	80610	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	90470	2547408409520575	joueur_165	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	94023	2582035846398048	joueur_403	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	101766	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	110276	2582035846398048	joueur_403	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	112243	2806336078533919	joueur_1010	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	116315	2533274823110022	JGtm	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	118066	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	122153	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	134900	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	137502	2749978711717053	joueur_621	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	140956	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	141074	2806336078533919	joueur_1010	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	148079	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	150566	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	156455	2547408409520575	joueur_165	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	157206	2806336078533919	joueur_1010	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	172138	2623439693412443	joueur_117	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	172905	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	177877	2533274823110022	JGtm	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	181847	2749978711717053	joueur_621	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	188554	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	191823	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	194895	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	198064	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	214949	2806336078533919	joueur_1010	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	215198	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	215864	2547408409520575	joueur_165	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	217018	2533274823110022	JGtm	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	217049	2533906527081383	joueur_1011	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	222956	2533906527081383	joueur_1011	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	224890	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	230931	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	231247	2749978711717053	joueur_621	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	242777	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	255706	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	261494	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	265014	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	266182	2749978711717053	joueur_621	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	268486	2533274823110022	JGtm	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	270955	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	274992	2623439693412443	joueur_117	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	278813	2582035846398048	joueur_403	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	279695	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	283816	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	284552	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	294913	2547408409520575	joueur_165	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	298683	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	299367	2582035846398048	joueur_403	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	299518	2623439693412443	joueur_117	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	304389	2806336078533919	joueur_1010	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	305673	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	309093	2582035846398048	joueur_403	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	309096	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	324391	2623439693412443	joueur_117	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	326612	2533274823110022	JGtm	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	328330	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	338840	2547408409520575	joueur_165	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	348517	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	349251	2547408409520575	joueur_165	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	351019	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	352187	2533274823110022	JGtm	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	364851	2547408409520575	joueur_165	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	367269	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	367869	2533906527081383	joueur_1011	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	371023	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	374993	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	386038	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	386437	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	390593	2806336078533919	joueur_1010	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	390692	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	393261	2623439693412443	joueur_117	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	398434	2533906527081383	joueur_1011	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	399368	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	408227	2547408409520575	joueur_165	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	412947	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	414868	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	416936	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	420673	2623439693412443	joueur_117	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	425328	2533906527081383	joueur_1011	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	428915	2533906527081383	joueur_1011	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	428917	2547408409520575	joueur_165	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	434268	2582035846398048	joueur_403	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	436122	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	438358	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	443529	2533906527081383	joueur_1011	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	449500	2623439693412443	joueur_117	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	451938	2806336078533919	joueur_1010	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	455926	2582035846398048	joueur_403	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	458312	2582035846398048	joueur_403	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	466321	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	466587	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	482036	2533906527081383	joueur_1011	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	482820	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	485371	2533274823110022	JGtm	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	488159	2533274823110022	JGtm	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	494698	2806336078533919	joueur_1010	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	497318	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	515534	2623439693412443	joueur_117	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	516389	2582035846398048	joueur_403	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	519858	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	521193	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	524228	2582035846398048	joueur_403	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	527899	2533274823110022	JGtm	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	538475	2806336078533919	joueur_1010	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	540262	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	542381	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	553759	2582035846398048	joueur_403	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	555311	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	561033	2533906527081383	joueur_1011	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	567257	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	573296	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	575932	2806336078533919	joueur_1010	2563513065873397	JOUEUR_1012	1
ab526724-3684-4335-b759-a18edcccc137	585942	2582035846398048	joueur_403	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	588328	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	592315	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	596135	2533274823110022	JGtm	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	600306	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	605710	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	607898	2547408409520575	joueur_165	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	614838	2623439693412443	joueur_117	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	616656	2533906527081383	joueur_1011	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	616957	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	625865	2563513065873397	JOUEUR_1012	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	630236	2533274823110022	JGtm	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	630239	2749978711717053	joueur_621	2533274823110022	JGtm	1
ab526724-3684-4335-b759-a18edcccc137	632022	2563513065873397	JOUEUR_1012	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	637944	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	645202	2547408409520575	joueur_165	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	648028	2623439693412443	joueur_117	2533906527081383	joueur_1011	1
ab526724-3684-4335-b759-a18edcccc137	651415	2623439693412443	joueur_117	2582035846398048	joueur_403	1
ab526724-3684-4335-b759-a18edcccc137	654468	2806336078533919	joueur_1010	2547408409520575	joueur_165	1
ab526724-3684-4335-b759-a18edcccc137	655568	2563513065873397	JOUEUR_1012	2806336078533919	joueur_1010	1
ab526724-3684-4335-b759-a18edcccc137	662943	2749978711717053	joueur_621	2623439693412443	joueur_117	1
ab526724-3684-4335-b759-a18edcccc137	665662	2563513065873397	JOUEUR_1012	2749978711717053	joueur_621	1
ab526724-3684-4335-b759-a18edcccc137	665744	2582035846398048	joueur_403	2563513065873397	JOUEUR_1012	1
```

### m2407 · `matches` (1 lignes)

```
match_id	d	h	map_name	map_name_fr	pair_name	playlist_name	md	my_outcome	my_team	team_0_score	team_1_score	rounds_total	team_0_rounds_won	team_1_rounds_won	duration_seconds	player_count
4f77afc1-d9f4-443b-b38f-3628341bf7e9	24/07/2026	22:20	Flood Gulch	NULL	BTB:CTF on Flood Gulch	Big Team Battle	BTB:CTF	1	1	1	1	NULL	NULL	NULL	1194	0
```

### m2407 · `parts` (36 lignes)

```
match_id	xuid	gt	team	pac	pwk	kills	deaths	assists	rank	score	melee	gren
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450		0	1	0	0	18	0	1	335	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374		0	1	1	1	3	1	3	150	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898		0	1	8	34	13	8	3	4300	1	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887		0	1	20	54	9	4	3	6400	1	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002		0	1	5	20	8	5	3	2350	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160		0	1	1	12	10	10	3	1900	3	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143		0	1	0	3	12	4	3	575	1	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218		0	1	3	16	16	6	3	2050	0	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945		0	1	0	1	19	2	3	225	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669		0	1	0	7	16	3	3	875	0	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791		0	1	0	4	12	5	3	650	1	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(23.0)		0	1	0	0	1	1	3	50	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491		0	0	0	2	4	0	25	200	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(38.0)		0	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170		0	0	0	1	8	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2622264960958390		0	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(11.0)		0	0	0	0	0	2	25	100	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(57.0)		0	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728		1	1	1	19	11	10	1	2930	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404		1	1	1	9	11	6	3	1360	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626		1	1	4	10	18	5	3	1325	1	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909		1	1	4	10	11	5	3	1325	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810		1	1	2	24	13	7	3	2750	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188		1	1	0	5	17	1	3	575	0	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364		1	1	0	7	11	14	3	1475	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266		1	1	7	22	11	7	3	2675	1	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443		1	1	1	16	19	7	3	2120	1	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	JGtm	1	1	0	10	15	1	3	1225	0	3
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(59.0)		1	1	2	5	3	3	3	700	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243		1	1	1	9	11	5	3	1325	2	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(6.0)		1	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(42.0)		1	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622		1	0	0	2	4	0	25	200	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2813308502433074		1	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	bid(22.0)		1	0	0	0	0	0	25	0	0	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2721080764965416		1	0	0	0	0	0	25	0	0	0
```

### m2407 · `usage` (25 lignes)

```
match_id	xuid	camo_episodes	camo_ms	camo_kills	overshield_episodes	overshield_ms	overshield_kills	grapple_pulls	deployed_json	dropped_objects	pad_pickups	pad_pickups_json	taken_json	spent_json	kept_json	dropped_json
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	0	0	0	0	0	0	0	{}	3	0	{}	{}	{}	{}	{"powerup_camo":1,"repulsor":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	0	0	0	0	0	0	0	{}	1	2	{"30484ea6":1,"b619d84a":1}	{}	{}	{}	{"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	0	0	0	0	0	0	0	{}	4	6	{"2b1824d5":3,"767db96d":2,"c3542946":1}	{}	{}	{}	{"grapple":1,"repulsor":2,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	0	0	0	0	0	0	0	{}	2	4	{"84bd29ed":1,"9d6aaed2":3}	{}	{}	{}	{"repulsor":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	0	0	0	0	0	0	2	{"wall":1}	8	9	{"2b1824d5":2,"767db96d":2,"9d6aaed2":1,"c30d87c7":3,"f5c335df":1}	{}	{}	{}	{"grapple":7,"repulsor":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	0	0	0	4	95400	0	0	{}	5	4	{"2b1824d5":1,"71ab0a2c":2,"84bd29ed":1}	{}	{}	{}	{"grapple":1,"repulsor":4}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	4	173600	0	0	0	0	6	{"wall":1}	8	10	{"0a1992bc":2,"2b1824d5":2,"71ab0a2c":3,"767db96d":3}	{}	{}	{}	{"grapple":3,"repulsor":5}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	0	0	0	0	0	0	2	{}	4	2	{"2b1824d5":1,"b619d84a":1}	{}	{}	{}	{"grapple":1,"repulsor":1,"shroud_screen":2}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	0	0	0	0	0	0	0	{}	0	0	{}	{}	{}	{}	{}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	0	0	0	0	0	0	8	{}	4	10	{"0a1992bc":3,"71ab0a2c":1,"767db96d":1,"c30d87c7":4,"f5c335df":1}	{}	{}	{}	{"grapple":1,"repulsor":1,"shroud_screen":1,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	1	34300	0	0	0	0	1	{"wall":1}	4	5	{"2b1824d5":1,"84bd29ed":1,"9d6aaed2":3}	{}	{}	{}	{"grapple":1,"repulsor":1,"shroud_screen":2}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	0	0	0	0	0	0	0	{}	0	3	{"2b1824d5":2,"767db96d":1}	{}	{}	{}	{}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	1	24700	0	1	12300	0	4	{}	2	4	{"767db96d":2,"84bd29ed":2}	{}	{}	{}	{"grapple":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	0	0	0	0	0	0	0	{}	2	3	{"2b1824d5":1,"84bd29ed":2}	{}	{}	{}	{"repulsor":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	0	0	0	0	0	0	0	{}	2	1	{"f5c335df":1}	{}	{}	{}	{"shroud_screen":1,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	0	0	0	0	0	0	0	{"wall":1}	4	14	{"0a1992bc":4,"2b1824d5":1,"84bd29ed":8,"b619d84a":1}	{}	{}	{}	{"shroud_screen":1,"wall":3}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	0	0	0	0	0	0	0	{"wall":1}	5	9	{"0a1992bc":1,"9d6aaed2":3,"b619d84a":4,"f5c335df":1}	{}	{}	{}	{"grapple":1,"repulsor":1,"shroud_screen":2,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	0	0	0	0	0	0	0	{"wall":2}	2	3	{"2b1824d5":2,"c30d87c7":1}	{}	{}	{}	{"repulsor":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	0	0	0	0	0	0	0	{"wall":1}	0	11	{"0a1992bc":5,"30484ea6":1,"b619d84a":3,"c30d87c7":1,"f5c335df":1}	{}	{}	{}	{}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	3	43300	0	0	0	0	1	{}	3	5	{"2b1824d5":4,"c3542946":1}	{}	{}	{}	{"grapple":1,"repulsor":1,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	0	0	0	0	0	0	0	{}	2	5	{"0a1992bc":4,"c30d87c7":1}	{}	{}	{}	{"repulsor":1,"shroud_screen":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	0	0	0	0	0	0	0	{"wall":3}	2	8	{"2b1824d5":4,"84bd29ed":1,"b619d84a":3}	{}	{}	{}	{"repulsor":1,"wall":1}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	0	0	0	0	0	0	0	{}	3	8	{"2b1824d5":1,"767db96d":1,"84bd29ed":1,"9d6aaed2":1,"b619d84a":1,"c30d87c7":1,"f5c335df":2}	{}	{}	{}	{"grapple":1,"shroud_screen":2}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	0	0	0	0	0	0	0	{}	0	0	{}	{}	{}	{}	{}
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	0	0	0	0	0	0	7	{}	2	4	{"30484ea6":1,"71ab0a2c":2,"84bd29ed":1}	{}	{}	{}	{"grapple":1,"repulsor":1}
```

### m2407 · `films` (1 lignes)

```
match_id	duration_ms	powerup_pickups_json	weapon_pads_json	pad_occupancies	pad_named	pad_unnamed
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1111100	{"powerup_camo":18,"powerup_overshield":19}	[{"weapon":"2b1824d5","occupations":11,"named":4},{"weapon":"2b1824d5","occupations":8,"named":3},{"weapon":"2b1824d5","occupations":24,"named":3},{"weapon":"2b1824d5","occupations":15,"named":9},{"weapon":"2b1824d5","occupations":14,"named":6},{"weapon":"b619d84a","occupations":12,"named":7},{"weapon":"b619d84a","occupations":15,"named":7},{"weapon":"84bd29ed","occupations":20,"named":15},{"weapon":"84bd29ed","occupations":6,"named":3},{"weapon":"71ab0a2c","occupations":17,"named":8},{"weapon":"9d6aaed2","occupations":18,"named":5},{"weapon":"9d6aaed2","occupations":17,"named":6},{"weapon":"f5c335df","occupations":8,"named":6},{"weapon":"f5c335df","occupations":5,"named":1},{"weapon":"767db96d","occupations":22,"named":12},{"weapon":"c3542946","occupations":3,"named":2},{"weapon":"30484ea6","occupations":2,"named":0},{"weapon":"30484ea6","occupations":10,"named":3},{"weapon":"c30d87c7","occupations":15,"named":11},{"weapon":"0a1992bc","occupations":19,"named":9},{"weapon":"0a1992bc","occupations":19,"named":10}]	317	130	150
```

### m2407 · `tiers` (75 lignes)

```
match_id	xuid	tier	weapon_family	pickups	pads_total	pads_confirmed	rs
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	terrain	30484ea6	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	non_classe	71ab0a2c	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	terrain	f5c335df	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	terrain	b619d84a	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	puissance	9d6aaed2	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	non_classe	c30d87c7	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	non_classe	767db96d	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	terrain	b619d84a	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	terrain	2b1824d5	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	puissance	0a1992bc	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	non_classe	c30d87c7	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	terrain	c3542946	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	terrain	2b1824d5	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	terrain	f5c335df	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	terrain	b619d84a	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	terrain	30484ea6	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	puissance	0a1992bc	5	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	non_classe	c30d87c7	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	terrain	2b1824d5	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	non_classe	c30d87c7	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	terrain	f5c335df	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	terrain	b619d84a	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	puissance	9d6aaed2	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	puissance	0a1992bc	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	terrain	b619d84a	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	terrain	84bd29ed	8	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	puissance	0a1992bc	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	terrain	f5c335df	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	terrain	84bd29ed	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	terrain	84bd29ed	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	non_classe	767db96d	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	terrain	2b1824d5	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	non_classe	767db96d	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	puissance	9d6aaed2	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2721080764965416	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	terrain	f5c335df	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	puissance	0a1992bc	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	non_classe	c30d87c7	4	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	non_classe	767db96d	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	non_classe	71ab0a2c	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2813308502433074	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	terrain	b619d84a	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2622264960958390	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	terrain	2b1824d5	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	puissance	0a1992bc	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	non_classe	767db96d	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	non_classe	71ab0a2c	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2801749445162953	aucune_prise		0	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	terrain	2b1824d5	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	non_classe	71ab0a2c	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	terrain	f5c335df	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	terrain	2b1824d5	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	puissance	9d6aaed2	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	non_classe	c30d87c7	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	non_classe	767db96d	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	terrain	84bd29ed	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	puissance	9d6aaed2	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	terrain	c3542946	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	terrain	2b1824d5	3	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	non_classe	767db96d	2	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	terrain	b619d84a	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	terrain	30484ea6	1	23	20	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	aucune_prise		0	23	20	0
```

### m2407 · `kills` (0 lignes)

```
match_id	time_ms	killer	kgt	victim	vgt	assist	agt	ak	kdp	adp	tag	cat
```

### m2407 · `killcov` (1 lignes)

```
match_id	n	pub
4f77afc1-d9f4-443b-b38f-3628341bf7e9	294	0
```

### m2407 · `lives` (253 lignes)

```
match_id	xuid	start_ms	end_ms	end_cause
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	157514	219077	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	229162	250808	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	260885	280872	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	301661	526905	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	537011	572840	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	582916	594795	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	604872	651519	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	661595	699934	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	710014	749649	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	759726	796730	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	806809	825259	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	835337	887656	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	897731	915517	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	944584	952122	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	269998	291517	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	301632	385725	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	395803	540886	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	550984	884855	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	894930	900501	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	910610	984286	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	358397	489299	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	499375	580179	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	590258	616418	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	626492	791757	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	801869	849051	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	859129	864465	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	905072	921222	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	931332	1048783	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	86687	223847	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	233922	316178	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	326265	409685	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	419795	432573	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	443859	483994	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	494072	538380	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	548490	569868	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	579945	659159	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	669267	715986	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	726062	765631	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	775710	798498	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	826895	967704	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	86719	192152	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	134358	134358	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	202233	252778	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	262854	277904	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	309101	410786	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	420996	466541	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	478154	578080	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	588154	625124	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	635201	690058	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	700167	734300	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	744411	780415	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	790523	855424	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	865502	903070	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	913147	972342	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	1144481	1144481	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	86824	174667	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	184748	224949	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	235057	250508	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	260586	326800	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	336877	404611	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	415155	437010	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	447121	502679	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	512758	546691	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	556788	657188	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	667299	716351	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	726459	763663	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	773742	838673	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	848783	894162	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	904238	934440	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	944551	983386	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	86687	295891	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	305964	452061	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	462171	604235	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	614316	747845	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	782386	836603	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	846681	887155	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	897231	957325	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	109363	120259	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	168494	197688	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	207764	278337	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	288413	317046	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	355027	361135	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	386724	450427	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	460562	510890	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	520963	590191	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	628962	711949	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	722022	786821	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	796898	830834	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	922057	940840	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	87844	227050	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	237126	264757	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	274869	355095	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	365207	409751	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	419861	478486	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	488602	522434	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	540519	598832	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	608914	624960	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	635033	673239	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	683317	741707	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	751818	796561	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	807608	889060	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	899134	920555	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	944622	954991	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	87645	174706	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	184773	229624	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	239730	388963	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	399038	428235	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	438313	738442	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	748513	791825	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	822722	911946	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	922030	945380	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2721080764965416	635634	638772	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	168126	235424	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	245503	270162	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	280238	332105	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	342215	351159	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	402184	413288	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	431104	499076	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	509188	631764	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	641841	677045	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	687189	936870	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	86687	123515	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	146603	166692	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	176768	216608	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	226683	244237	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	280043	289752	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	86921	181038	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	191114	259852	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	269930	601466	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	611544	746611	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	756690	789222	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	799301	852321	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	862430	893528	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	903637	909077	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	108995	169793	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	179875	187244	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	212270	276637	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	286712	299826	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	355001	379487	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	389563	450191	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	460300	509521	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	519594	578746	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	592560	598297	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	726436	787054	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	797130	915953	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	926029	989592	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	87710	135593	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	167959	229756	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	239830	329035	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	339144	349891	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	366942	375184	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	402209	409048	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	443885	489933	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	500009	509463	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	553517	579843	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	746044	823125	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	833201	1079950	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	86687	248840	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	292454	392068	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	402151	528739	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	538814	552818	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	733534	772206	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	782319	832203	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	868237	891395	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	901470	915883	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	925960	1056061	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	86687	146603	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	163919	169661	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	183573	201859	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	211936	220911	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	231024	358732	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	368809	436743	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	446822	659024	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	669101	702169	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	712247	779380	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	791356	894461	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	904538	944310	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	108995	243135	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	253211	323256	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	333339	425300	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	435376	530040	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	540151	598832	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	608941	740109	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	750217	819988	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	830069	847284	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	86946	139097	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	149176	406645	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	416758	509420	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	519496	1197714	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	86687	139097	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	149205	163586	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	179904	226416	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	236493	245873	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	342190	392902	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	402977	588654	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	598731	654658	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	664730	816119	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	826226	1006242	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	110230	121079	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	161182	204962	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	215040	254812	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	366916	380586	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	424566	439281	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	460635	587052	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	597129	601098	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	674708	679945	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	741307	747479	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	767601	919521	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	929597	1031134	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	87743	174901	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	184974	228751	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	238829	351592	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	361669	478486	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	488631	561292	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	571405	617652	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	627728	668802	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	678887	738145	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	748212	782451	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	792526	854322	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	905043	984888	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	87146	138928	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	149004	177134	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	187210	246005	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	256085	297290	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	307367	343015	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	353093	418958	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	429036	431671	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	460611	471081	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	537746	555154	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	565230	596128	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	606205	642679	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	652753	668802	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	678912	720289	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	730363	775343	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	785419	934571	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	1144349	1144415	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	206195	254047	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	264123	284112	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	294188	339679	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	349758	353661	cut
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	367108	373983	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	384092	421965	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	86790	113369	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	202126	215874	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	225983	294356	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	304429	367009	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	377119	464505	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	474581	498044	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	508119	518760	film_end
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	557423	597263	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	607339	739541	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	749615	806107	death
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	816186	959697	death
```

### m2407 · `dctx` (191 lignes)

```
match_id	victim_xuid	time_ms	nearest_teammate_m	teammates_visible	teammates_total
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	219017	12.36	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	250748	3.8	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	526844	33.24	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	572774	18.84	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	594733	12.47	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	651453	25.49	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	699873	2.3	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	749587	40.43	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	796669	25.26	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	825198	19.45	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	887594	7.19	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	291459	22.74	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	385660	12.02	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	540822	38.83	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	884792	12.79	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	900443	11.88	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2788951673476170	984227	12.67	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	489237	1.36	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	580117	23.6	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	616354	13.51	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	791698	14.2	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	848988	28.64	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	1048718	71.5	1	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	223787	5.34	11	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	316116	19.74	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	409625	1.56	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	483932	14.94	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	538322	3.81	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	569807	16.25	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	659099	20.22	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	715924	23.49	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	765567	2.74	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	967644	13.77	3	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	192089	13.06	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	252717	15.28	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	410726	42.51	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	466481	11.87	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	578013	51.25	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	625063	8.96	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	690000	15.98	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	734242	2.06	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	780355	8.32	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	855361	5	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	903008	11.68	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	972278	34.15	2	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	174602	4.62	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	224890	21.78	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	250447	23.99	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	326738	5.58	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	404987	9.54	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	436952	4.94	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	502618	25	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	546630	25.83	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	657131	42.63	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	716296	16.16	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	763599	5.63	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	838613	18.87	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	894100	38.64	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	934378	19.28	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	983327	51.16	1	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	295828	17.57	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	452002	19.11	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	604175	16.04	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	836542	20.01	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	887093	48.31	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	957260	15.71	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	197627	20.91	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	278277	34.86	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	450369	6.84	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	510822	14.48	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	711884	46.3	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	786759	6.22	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	940783	6.52	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	226990	16.05	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	264701	13.84	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	355035	12.64	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	409691	9.05	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	478428	1.74	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	598766	16.16	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	624897	6.28	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	673177	18.33	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	741649	37.11	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	797470	26.86	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2682452436637450	888992	12.4	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	174638	29.81	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	229564	1.7	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	388898	39.76	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	428170	15.97	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	738377	16.59	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	911883	5.71	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	945318	28.59	3	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	235365	33.31	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	270101	33.67	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	332048	31.71	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	499016	24.03	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	631703	25.9	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	676982	27.99	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	936806	20.53	3	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	166628	21.03	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	216547	7.05	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	180974	27.72	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	259791	21.64	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	601405	13.22	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	746552	20.19	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	789160	14.58	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	852262	7.51	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	893470	2.78	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	169730	22.82	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	276575	12.24	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	379425	29.52	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	450137	13.33	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	509454	0.93	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	786989	15.03	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	915886	34.45	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	989532	21.98	3	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	229691	2.26	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	328977	15.58	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	489873	29.7	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	823062	26.4	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	1079892	NULL	0	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	392001	43.44	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	528677	20.58	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	772147	7.79	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	891332	5.67	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	915822	11.04	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	1056003	NULL	0	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	201797	4.1	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	220856	11.13	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	358666	22.42	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	436683	16.61	11	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	658960	38.47	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	702108	27.74	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	894398	20.79	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	944253	7.42	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	243073	20.64	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	323191	5.38	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	425239	30.91	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	529982	27.32	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	598774	6.96	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	740049	43.25	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	819925	6.85	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	139032	0.78	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	406588	10.23	11	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2769104854931374	509356	2.29	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	139035	0.78	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	226355	5.71	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	392839	50.23	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	588594	21.63	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	654591	19.96	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	816064	25.76	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	1006180	42.16	2	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	204902	10.21	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	586991	12.32	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	919459	37.67	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	1031072	96.96	1	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	174838	46.54	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	228691	3	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	351530	17.31	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	478430	1.74	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	561235	23.6	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	617586	57.97	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	668739	20.12	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	738074	16.88	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	782389	2.57	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	984822	6.06	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	138868	20.46	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	177070	6.31	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	245944	24.03	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	297229	20.73	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	342954	20.49	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	418894	20.07	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	555092	5.93	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	596066	9.31	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	642615	5.84	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	668742	24.28	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	720222	22.23	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	775278	14.32	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	934509	23.31	4	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	253986	45.1	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	284050	7.89	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	339618	4.82	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2750551653702622	373924	7.09	9	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	215818	16.34	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	294289	11.32	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	366949	18.07	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	464444	35.23	10	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	497980	27.05	6	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	597198	9.27	7	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	739479	18.87	5	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	806045	13.61	8	13
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	959636	37.75	5	13
```

### m2407 · `kpos` (135 lignes)

```
match_id	killer_xuid	time_ms	dist	dz
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	138868	2.17	0.75
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	169730	14.5	0.92
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	174638	20.47	-0.38
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	174838	0.06	-0.04
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	192089	3.89	1.28
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	197627	7.36	0.65
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	201797	1.69	-0.78
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2575401242986491	204902	7.75	-1.37
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	216547	3.29	0.27
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	219017	0.57	0.55
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	223787	5.84	-1.5
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	224890	3.08	0.21
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	226355	7.17	1.04
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	228691	10.28	0.69
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	229564	18.33	-2.58
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	229691	14.08	-1.68
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	235365	1.68	0.29
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	243073	3.12	-2.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	245944	5.18	0.68
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	250447	4.58	0.69
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2563350602769669	252717	8.4	1.12
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	259791	11.54	3.19
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	270101	0.26	-0.01
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	276575	6.9	0.77
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	278277	8.58	1.38
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	295828	2.74	-0.98
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	297229	12.97	1.18
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	316116	15.74	-0.61
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	326738	19.95	-0.22
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	328977	26	0.93
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	332048	10.37	1.57
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	342954	19.28	2.87
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	351530	14.35	1.47
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	379425	10.89	1.56
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	388898	7.03	-0.89
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	392839	6.07	-3.39
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	406588	31.8	3.26
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	409625	7.22	0.86
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	409691	29.36	0.69
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	410726	7.21	1.22
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	418894	2.54	1.59
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	425239	4.94	0.75
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	428170	2.66	1.52
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	436683	21.55	-2.1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	450137	3.23	2
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	450369	24.95	-1.7
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	452002	3.14	1.43
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	464444	4.99	-0.28
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	466481	1.52	0.19
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	478428	3.89	1.22
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	478430	2.52	0.89
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	483932	26	2.08
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2634990399857364	489237	3.09	-0.39
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	489873	3.92	0.6
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	497980	9.81	0.05
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	502618	3.84	-1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	510822	21.77	-0.89
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	526844	1.6	0.69
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	528677	7.02	-0.92
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	529982	34.05	-2.55
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	540822	2.89	-0.41
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	546630	4.66	-1.15
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	555092	13.17	-0.84
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2798225113631945	569807	9.3	0.67
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	580117	4.83	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	586991	9.1	-1.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	588594	22.86	-0.64
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	594733	12.25	1.86
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	596066	0.26	0.05
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	598766	3.25	-0.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	598774	4.88	1.27
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	601405	12.02	-1.9
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	604175	3.86	-0.73
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	616354	6.81	0.02
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	617586	8.51	1.86
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	624897	11.25	2.01
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	625063	10.99	7.68
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	631703	2.67	-0.03
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	642615	22.84	-0.56
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	654591	3.66	1.5
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	657131	4.04	2.65
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	659099	24.37	-0.12
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2640661141794626	668739	0.23	-0.02
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2784870293334143	668742	0.23	0.02
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	673177	4.84	-0.83
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	676982	10.04	-3.59
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	690000	26.33	-0.48
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	699873	6.38	-0.07
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	702108	8.45	0.34
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	711884	10.13	2.95
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	715924	11.8	4.76
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	734242	14.58	-4.73
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	738074	4.56	0.91
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	738377	22.31	-1.05
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	739479	8.4	2.54
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	740049	26.55	-0.46
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	741649	32.08	5.58
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	746552	2.32	-1.19
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	749587	10.09	0.04
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	763599	38.7	2.51
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2646046920107160	772147	27.83	-0.23
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	775278	10.37	-1.61
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	780355	26.45	0.87
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	782389	16.64	-1.33
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2753091070419728	786759	9.75	1.15
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2533274823110022	786989	13.2	-1.95
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	789160	8.21	1.98
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	791698	36.21	4.2
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	796669	4.25	-0.56
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	806045	17.9	1.68
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2685352648935266	816064	4.15	1.32
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	819925	19.23	1.66
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	823062	15.58	-1.77
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	825198	16.06	-0.07
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	836542	4.62	1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	838613	30.44	1.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2632700264829909	848988	2.67	0.38
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	852262	34.19	-0.67
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2623439693412443	884792	13.92	-1.25
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	887093	8.07	-0.73
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	887594	0.31	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	891332	14.07	-2.78
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	894100	6.92	-1.38
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2542494231387188	894398	7.02	1.17
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2779781155078810	900443	25.72	0.64
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	903008	10.45	1.16
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2642222702413791	915822	8.44	0.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2615815077049243	915886	8.47	-0.24
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	919459	3.34	-1.04
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	934378	53.92	2.1
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2690056610860898	934509	12.61	1.12
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2710930876332218	936806	46.17	2.43
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2550886075837404	944253	5.47	-1.27
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2793075222215887	945318	5.14	-0.6
4f77afc1-d9f4-443b-b38f-3628341bf7e9	2573321465012002	967644	37.61	0.4
```

### m2407 · `veh` (0 lignes)

```
match_id	row_kind	n
```

### m2407 · `hl` (844 lignes)

```
match_id	event_type	time_ms	xuid	type_hint
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	136435	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	136435	2575401242986491	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	136442	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	138868	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	138868	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	138868	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	139032	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	139032	2769104854931374	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	139035	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	139035	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	139035	2640661141794626	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	140677	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	150219	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	151046	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	151047	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	151047	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	153781	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	153781	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	153795	2634990399857364	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	157820	2575401242986491	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	157820	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	158324	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	158324	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	158325	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	166628	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	166628	2575401242986491	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	168705	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	169730	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	169730	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	169733	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	169733	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	169733	2640661141794626	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	169746	2640661141794626	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	173435	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	173435	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	173435	2640661141794626	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	173436	2640661141794626	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	173449	2640661141794626	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	174601	2710930876332218	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	174602	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	174602	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	174638	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	174638	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	174838	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	174838	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	176715	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	177069	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	177070	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	177070	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	177070	2710930876332218	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	180974	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	180974	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	180975	2710930876332218	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	183618	2682452436637450	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	191125	2615815077049243	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	191958	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	191958	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	192088	2563350602769669	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	192089	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	192089	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	193362	2634990399857364	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	196832	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	196833	2682452436637450	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	197627	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	197627	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	199236	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	201797	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	201798	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	202132	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	202132	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	202133	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	204902	2575401242986491	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	204902	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	215818	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	215818	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	215818	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	216547	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	216547	2575401242986491	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	219015	2685352648935266	200
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	219017	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	219017	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	219017	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	220856	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	220856	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	221691	2550886075837404	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	223787	2784870293334143	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	223787	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	223995	2550886075837404	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	224890	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	224890	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	226355	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	226355	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	226355	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	226355	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	226990	2750551653702622	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	226990	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	228691	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	228691	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	228691	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	228691	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	229564	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	229565	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	229570	2563350602769669	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	229691	2642222702413791	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	229691	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	235364	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	235365	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	235365	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	243073	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	243073	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	245944	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	245944	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	250447	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	250447	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	250447	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	250748	2750551653702622	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	250748	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	252717	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	252717	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	253986	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	253987	2750551653702622	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	259791	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	259791	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	264701	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	264701	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	264701	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	268971	2533274823110022	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	269876	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	269876	2575401242986491	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	270101	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	270101	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	276575	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	276575	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	276575	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	278277	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	278277	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	278277	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	282284	2769104854931374	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	282284	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	282285	2769104854931374	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	284050	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	284050	2750551653702622	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	289293	2634990399857364	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	289293	2634990399857364	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	291459	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	291459	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	291521	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	291521	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	293498	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	294289	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	294289	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	294290	2533274823110022	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	295827	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	295828	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	295828	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	297229	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	297229	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	298936	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	298936	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	298944	2646046920107160	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	316116	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	316116	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	318424	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	323191	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	323191	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	323204	2646046920107160	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	325309	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	326738	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	326738	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	328977	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	328977	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	328977	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	330450	2685352648935266	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	332045	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	332045	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	332048	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	332048	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	332048	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	332049	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	339618	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	339618	2750551653702622	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	339618	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	341458	2710930876332218	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	342954	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	342954	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	342955	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	342955	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	344827	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	344827	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	344831	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	344831	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	344832	2533274823110022	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	344832	2533274823110022	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	344838	2533274823110022	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	347834	2784870293334143	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	351530	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	351530	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	351541	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	355035	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	355035	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	356766	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	356766	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	356779	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	356798	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	356798	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	356798	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	358666	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	358666	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	358666	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	366949	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	366949	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	373923	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	373924	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	373924	2750551653702622	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	373924	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	376586	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	376586	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	379425	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	379425	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	385660	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	385660	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	388608	2550886075837404	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	388897	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	388898	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	388898	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	388912	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	388913	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	390041	2542494231387188	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	392001	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	392001	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	392004	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	392004	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	392004	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	392008	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	392008	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	392008	2573321465012002	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	392839	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	392839	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	403088	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	404987	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	404987	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	404995	2710930876332218	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	406588	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	406588	2769104854931374	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	406588	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	409625	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	409625	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	409625	2710930876332218	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	409691	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	409691	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	409692	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	410726	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	410726	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	410726	2710930876332218	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	412631	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	414424	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	414424	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	418894	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	418894	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	418894	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	420808	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	420939	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	420939	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	425239	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	425239	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	428170	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	428170	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	428170	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	433486	2710930876332218	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	433712	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	433712	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	433715	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	433715	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	433715	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	436682	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	436683	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	436683	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	436952	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	436952	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	449603	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	449603	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	450137	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	450137	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	450140	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	450369	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	450369	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	450436	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	450436	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	450436	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	450439	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	450439	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	450439	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	452002	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	452002	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	452003	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	464443	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	464444	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	464444	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	466481	2784870293334143	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	466481	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	478428	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	478428	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	478430	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	478430	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	478431	2550886075837404	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	483932	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	483932	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	483932	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	489237	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	489237	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	489238	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	489873	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	489873	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	497980	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	497980	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	499016	2788951673476170	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	499016	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	502618	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	502618	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	509356	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	509356	2769104854931374	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	509454	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	509454	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	510822	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	510822	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	510835	2542494231387188	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	526844	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	526844	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	527579	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	527579	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	528677	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	528677	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	529982	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	529982	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	529983	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	529983	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	530379	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	530379	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	530379	2779781155078810	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	538322	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	538322	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	540822	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	540823	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	542965	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	543361	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	543361	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	543369	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	546630	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	546630	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	546631	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	546631	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	546631	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	547267	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	547267	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	555092	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	555092	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	555093	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	561235	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	561235	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	569807	2798225113631945	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	569807	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	572774	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	572774	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	578013	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	578013	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	580117	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	580117	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	582391	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	582391	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	582397	2632700264829909	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	586991	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	586992	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	588594	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	588594	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	594733	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	594733	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	594733	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	596066	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	596066	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	596067	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	597198	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	597198	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	598109	2682452436637450	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	598766	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	598766	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	598774	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	598774	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	598774	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	598783	2632700264829909	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	601405	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	601405	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	604174	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	604175	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	604175	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	614359	2615815077049243	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	616354	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	616354	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	617586	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	617586	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	618793	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	618793	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	622668	2533274823110022	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	624897	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	624897	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	625063	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	625063	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	625073	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	631703	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	631703	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	642615	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	642615	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	642615	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	651453	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	651453	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	654591	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	654591	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	654591	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	657131	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	657131	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	657131	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	658959	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	658960	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	658960	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	659099	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	659099	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	664539	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	664539	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	664539	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	664539	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	664539	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	668739	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	668739	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	668742	2784870293334143	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	668742	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	673177	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	673177	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	673178	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	676982	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	676982	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	690000	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	690000	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	699873	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	699873	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	702108	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	702108	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	702108	2623439693412443	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	711884	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	711884	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	715924	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	715924	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	716285	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	716285	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	716286	2550886075837404	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	716296	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	716296	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	720222	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	720222	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	720222	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	723392	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	723392	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	723393	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	726402	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	726402	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	731139	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	731139	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	731139	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	734241	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	734242	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	734242	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	734478	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	735909	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	735909	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	735910	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	738074	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	738074	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	738075	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	738377	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	738377	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	738377	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	738378	2690056610860898	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	739479	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	739479	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	740049	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	740049	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	741649	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	741649	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	741649	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	746552	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	746552	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	749587	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	749588	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	749588	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	757432	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	757433	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	757433	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	757440	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	763599	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	763599	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	765567	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	765567	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	765568	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	772147	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	772147	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	772241	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	772242	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	772242	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	775278	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	775278	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	775278	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	775278	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	777725	2784870293334143	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	780355	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	780355	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	781194	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	782389	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	782389	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	782400	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	786759	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	786759	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	786989	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	786989	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	789160	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	789161	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	789161	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	791698	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	791698	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	791699	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	791699	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	796669	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	796669	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	797470	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	797470	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	797470	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	797471	2623439693412443	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	800046	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	806045	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	806045	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	806045	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	812585	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	812585	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	812585	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	814760	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	816064	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	816064	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	816753	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	816753	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	819925	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	819925	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	819925	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	821935	2634990399857364	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	823062	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	823062	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	823072	2710930876332218	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	825198	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	825198	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	836542	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	836542	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	838613	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	838613	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	848988	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	848988	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	852262	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	852263	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	852263	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	852269	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	855361	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	855361	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	855361	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	858069	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	858070	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	858070	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	884792	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	884793	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	887093	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	887093	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	887336	2798225113631945	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	887594	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	887594	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	887605	2615815077049243	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	888991	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	888992	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	888992	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	891332	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	891332	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	893470	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	893470	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	894100	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	894100	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	894397	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	894398	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	894398	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	894870	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	894871	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	894871	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	894875	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	894875	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	894875	2632700264829909	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	894875	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	900443	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	900443	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	902448	2685352648935266	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	903008	2642222702413791	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	903008	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	909788	2550886075837404	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	911883	2563350602769669	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	911883	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	911887	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	911887	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	915822	2642222702413791	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	915822	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	915886	2615815077049243	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	915886	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	919459	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	919459	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	921164	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	929535	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	934378	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	934378	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	934379	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	934379	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	934414	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	934414	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	934417	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	934417	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	934418	2533274823110022	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	934424	2533274823110022	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	934509	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	934509	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	936806	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	936806	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	938049	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	938978	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	938978	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	938979	2793075222215887	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	940783	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	940783	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	944253	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	944253	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	945317	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	945318	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	945318	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	945318	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	957259	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	957260	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	957260	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	957471	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	959636	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	959636	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	959636	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	965168	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	965168	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	967379	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	967379	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	967386	2623439693412443	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	967644	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	967644	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	967644	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	971219	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	972276	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	972276	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	972278	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	972278	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	972279	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	974556	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	983327	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	983327	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	983498	2784870293334143	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	984227	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	984227	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	984821	2634990399857364	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	984822	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	984822	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	984822	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	984839	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	986492	2642222702413791	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	986492	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	987062	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	987062	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	987363	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	987363	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	987363	2779781155078810	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	987424	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	987424	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	989532	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	989532	2642222702413791	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	989533	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	989533	2753091070419728	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	993497	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	993563	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	993563	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	993564	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	993564	2779781155078810	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	997640	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	997642	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	997642	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	997642	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1003681	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1003681	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1003684	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1003684	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1003685	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1003685	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1003685	2690056610860898	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1003692	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1003692	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1005584	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1005584	2685352648935266	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1005585	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1006180	2623439693412443	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1006180	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1006312	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1006312	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1009353	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1009587	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1009588	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1009588	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1011321	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1011321	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1016187	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1016187	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1016187	2779781155078810	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1018428	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1018428	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1018428	2779781155078810	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1019793	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1019794	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1019794	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1025930	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1025930	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1026472	2646046920107160	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1028041	2573321465012002	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1028967	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1028967	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1031071	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1031072	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1031072	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1048718	2640661141794626	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1048718	2573321465012002	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1049258	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1049258	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1056003	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1056003	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1057339	2793075222215887	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1059332	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1059332	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1059340	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1059341	2632700264829909	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1059341	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1059536	2710930876332218	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1059536	2632700264829909	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1059542	2710930876332218	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1068108	2542494231387188	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1068109	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1071151	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1071151	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1075182	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1075182	2788951673476170	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1077284	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1079892	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1079892	2634990399857364	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1079892	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1080223	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1080223	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1082796	2753091070419728	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1084790	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1084791	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1084791	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1084791	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1084800	2573321465012002	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1092667	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1092667	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1100444	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1100444	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1100474	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1100474	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1100474	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1100474	2646046920107160	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1103377	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1103377	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1103377	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1103377	2685352648935266	150
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1104642	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1104642	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1104643	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1104643	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1107248	2753091070419728	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1107248	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1107549	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1107549	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1108648	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1108649	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1108649	2685352648935266	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1108649	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1111523	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1111523	2690056610860898	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1112589	2646046920107160	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1112589	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1114632	2563350602769669	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1114966	2646046920107160	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1115559	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1115559	2784870293334143	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1115559	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1115559	2779781155078810	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1116459	2563350602769669	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1124832	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1124832	2779781155078810	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1124846	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1129478	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1129478	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1129478	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1135216	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1135216	2640661141794626	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1137882	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1137882	2615815077049243	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1137882	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1137883	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1150496	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1150496	2542494231387188	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1153497	2533274823110022	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1153497	2710930876332218	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1155432	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1155433	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1155433	2793075222215887	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1158265	2634990399857364	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	mode	1166420	2690056610860898	10
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1169145	2690056610860898	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1169145	2753091070419728	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1171282	2779781155078810	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1171282	2682452436637450	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1171283	2779781155078810	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1176454	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1176455	2550886075837404	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1176455	2646046920107160	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1179227	2685352648935266	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1179227	2798225113631945	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1184533	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1184533	2623439693412443	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1187060	2573321465012002	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1187060	2550886075837404	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1187060	2573321465012002	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	kill	1188100	2793075222215887	50
4f77afc1-d9f4-443b-b38f-3628341bf7e9	death	1188100	2533274823110022	20
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1188100	2793075222215887	100
4f77afc1-d9f4-443b-b38f-3628341bf7e9	medal	1188100	2793075222215887	100
```

### m2407 · `names` (25 lignes)

```
x	gts	n
2642222702413791	JOUEUR_136	19
2533274823110022	JGtm	26
2750551653702622	joueur_1379	10
2682452436637450	joueur_218	18
2784870293334143	joueur_017	17
2769104854931374	joueur_430	5
2623439693412443	joueur_117	42
2634990399857364	JOUEUR_126	19
2753091070419728	joueur_811	38
2710930876332218	joueur_655	35
2542494231387188	joueur_297	23
2632700264829909	joueur_706	25
2690056610860898	joueur_580	54
2779781155078810	joueur_317	43
2575401242986491	JOUEUR_469	6
2550886075837404	JOUEUR_409	24
2573321465012002	joueur_615	31
2646046920107160	JOUEUR_390	28
2788951673476170	JOUEUR_456	9
2793075222215887	joueur_728	64
2563350602769669	JOUEUR_221	23
2798225113631945	joueur_172	22
2615815077049243	joueur_427	22
2685352648935266	joueur_257	37
2640661141794626	joueur_582	32
```

### m2407 · `kvall` (294 lignes)

```
match_id	time_ms	killer	kgt	victim	vgt	pub
4f77afc1-d9f4-443b-b38f-3628341bf7e9	136435	2623439693412443	joueur_117	2575401242986491	JOUEUR_469	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	138868	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	139032	2640661141794626	joueur_582	2769104854931374	joueur_430	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	139035	2640661141794626	joueur_582	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	151047	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	153781	2634990399857364	JOUEUR_126	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	157820	2575401242986491	JOUEUR_469	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	158324	2634990399857364	JOUEUR_126	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	166628	2623439693412443	joueur_117	2575401242986491	JOUEUR_469	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	169730	2640661141794626	joueur_582	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	169733	2640661141794626	joueur_582	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	173435	2640661141794626	joueur_582	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	174602	2710930876332218	joueur_655	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	174638	2793075222215887	joueur_728	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	174838	2615815077049243	joueur_427	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	177070	2710930876332218	joueur_655	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	180974	2710930876332218	joueur_655	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	191958	2615815077049243	joueur_427	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	192089	2563350602769669	JOUEUR_221	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	197627	2685352648935266	joueur_257	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	201797	2779781155078810	joueur_317	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	202132	2685352648935266	joueur_257	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	204902	2575401242986491	JOUEUR_469	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	215818	2634990399857364	JOUEUR_126	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	216547	2685352648935266	joueur_257	2575401242986491	JOUEUR_469	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	219017	2685352648935266	joueur_257	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	220856	2533274823110022	JGtm	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	223787	2784870293334143	joueur_017	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	224890	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	226355	2685352648935266	joueur_257	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	226990	2750551653702622	joueur_1379	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	228691	2685352648935266	joueur_257	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	229564	2563350602769669	JOUEUR_221	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	229691	2642222702413791	JOUEUR_136	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	235365	2563350602769669	JOUEUR_221	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	243073	2793075222215887	joueur_728	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	245944	2563350602769669	JOUEUR_221	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	250447	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	250748	2750551653702622	joueur_1379	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	252717	2563350602769669	JOUEUR_221	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	253986	2690056610860898	joueur_580	2750551653702622	joueur_1379	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	259791	2793075222215887	joueur_728	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	264701	2634990399857364	JOUEUR_126	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	269876	2533274823110022	JGtm	2575401242986491	JOUEUR_469	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	270101	2793075222215887	joueur_728	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	276575	2753091070419728	joueur_811	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	278277	2753091070419728	joueur_811	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	282284	2769104854931374	joueur_430	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	284050	2690056610860898	joueur_580	2750551653702622	joueur_1379	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	291459	2779781155078810	joueur_317	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	291521	2533274823110022	JGtm	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	294289	2533274823110022	JGtm	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	295828	2640661141794626	joueur_582	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	297229	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	298936	2646046920107160	JOUEUR_390	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	316116	2793075222215887	joueur_728	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	323191	2646046920107160	JOUEUR_390	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	326738	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	328977	2793075222215887	joueur_728	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	332045	2685352648935266	joueur_257	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	332048	2793075222215887	joueur_728	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	339618	2690056610860898	joueur_580	2750551653702622	joueur_1379	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	342954	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	344827	2533274823110022	JGtm	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	344831	2533274823110022	JGtm	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	351530	2623439693412443	joueur_117	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	355035	2779781155078810	joueur_317	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	356766	2690056610860898	joueur_580	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	356798	2793075222215887	joueur_728	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	358666	2550886075837404	JOUEUR_409	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	366949	2623439693412443	joueur_117	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	373924	2793075222215887	joueur_728	2750551653702622	joueur_1379	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	376586	2615815077049243	joueur_427	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	379425	2623439693412443	joueur_117	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	385660	2615815077049243	joueur_427	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	388898	2690056610860898	joueur_580	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	392001	2573321465012002	joueur_615	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	392004	2573321465012002	joueur_615	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	392008	2573321465012002	joueur_615	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	392839	2623439693412443	joueur_117	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	404987	2710930876332218	joueur_655	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	406588	2753091070419728	joueur_811	2769104854931374	joueur_430	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	409625	2710930876332218	joueur_655	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	409691	2753091070419728	joueur_811	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	410726	2710930876332218	joueur_655	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	414424	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	418894	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	420939	2710930876332218	joueur_655	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	425239	2690056610860898	joueur_580	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	428170	2793075222215887	joueur_728	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	433712	2690056610860898	joueur_580	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	433715	2690056610860898	joueur_580	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	436683	2615815077049243	joueur_427	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	436952	2646046920107160	JOUEUR_390	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	450137	2685352648935266	joueur_257	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	450369	2615815077049243	joueur_427	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	450436	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	450439	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	452002	2632700264829909	joueur_706	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	464444	2685352648935266	joueur_257	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	466481	2784870293334143	joueur_017	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	478428	2550886075837404	JOUEUR_409	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	478430	2550886075837404	JOUEUR_409	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	483932	2793075222215887	joueur_728	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	489237	2634990399857364	JOUEUR_126	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	489873	2646046920107160	JOUEUR_390	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	497980	2685352648935266	joueur_257	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	499016	2788951673476170	JOUEUR_456	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	502618	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	509356	2640661141794626	joueur_582	2769104854931374	joueur_430	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	509454	2632700264829909	joueur_706	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	510822	2542494231387188	joueur_297	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	526844	2542494231387188	joueur_297	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	527579	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	528677	2690056610860898	joueur_580	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	529982	2793075222215887	joueur_728	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	530379	2779781155078810	joueur_317	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	538322	2710930876332218	joueur_655	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	540822	2542494231387188	joueur_297	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	543361	2573321465012002	joueur_615	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	546630	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	547267	2550886075837404	JOUEUR_409	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	555092	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	561235	2753091070419728	joueur_811	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	569807	2798225113631945	joueur_172	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	572774	2550886075837404	JOUEUR_409	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	578013	2710930876332218	joueur_655	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	580117	2685352648935266	joueur_257	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	582391	2632700264829909	joueur_706	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	586991	2690056610860898	joueur_580	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	588594	2753091070419728	joueur_811	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	594733	2753091070419728	joueur_811	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	596066	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	597198	2634990399857364	JOUEUR_126	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	598766	2632700264829909	joueur_706	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	598774	2793075222215887	joueur_728	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	601405	2690056610860898	joueur_580	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	604175	2685352648935266	joueur_257	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	616354	2685352648935266	joueur_257	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	617586	2640661141794626	joueur_582	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	618793	2632700264829909	joueur_706	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	624897	2685352648935266	joueur_257	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	625063	2793075222215887	joueur_728	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	631703	2793075222215887	joueur_728	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	642615	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	651453	2779781155078810	joueur_317	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	654591	2632700264829909	joueur_706	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	657131	2690056610860898	joueur_580	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	658960	2779781155078810	joueur_317	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	659099	2793075222215887	joueur_728	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	664539	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	668739	2640661141794626	joueur_582	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	668742	2784870293334143	joueur_017	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	673177	2685352648935266	joueur_257	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	676982	2793075222215887	joueur_728	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	690000	2573321465012002	joueur_615	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	699873	2623439693412443	joueur_117	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	702108	2623439693412443	joueur_117	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	711884	2753091070419728	joueur_811	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	715924	2710930876332218	joueur_655	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	716285	2550886075837404	JOUEUR_409	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	716296	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	720222	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	723392	2793075222215887	joueur_728	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	731139	2690056610860898	joueur_580	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	734242	2710930876332218	joueur_655	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	735909	2690056610860898	joueur_580	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	738074	2550886075837404	JOUEUR_409	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	738377	2690056610860898	joueur_580	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	739479	2753091070419728	joueur_811	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	740049	2573321465012002	joueur_615	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	741649	2685352648935266	joueur_257	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	746552	2690056610860898	joueur_580	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	749587	2623439693412443	joueur_117	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	757432	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	763599	2573321465012002	joueur_615	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	765567	2793075222215887	joueur_728	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	772147	2646046920107160	JOUEUR_390	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	772242	2615815077049243	joueur_427	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	775278	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	780355	2573321465012002	joueur_615	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	782389	2623439693412443	joueur_117	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	786759	2753091070419728	joueur_811	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	786989	2533274823110022	JGtm	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	789160	2573321465012002	joueur_615	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	791698	2685352648935266	joueur_257	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	796669	2623439693412443	joueur_117	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	797470	2623439693412443	joueur_117	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	806045	2685352648935266	joueur_257	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	812585	2563350602769669	JOUEUR_221	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	816064	2685352648935266	joueur_257	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	816753	2690056610860898	joueur_580	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	819925	2793075222215887	joueur_728	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	823062	2710930876332218	joueur_655	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	825198	2623439693412443	joueur_117	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	836542	2779781155078810	joueur_317	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	838613	2573321465012002	joueur_615	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	848988	2632700264829909	joueur_706	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	852262	2793075222215887	joueur_728	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	855361	2690056610860898	joueur_580	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	858069	2793075222215887	joueur_728	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	884792	2623439693412443	joueur_117	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	887093	2779781155078810	joueur_317	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	887594	2615815077049243	joueur_427	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	888992	2753091070419728	joueur_811	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	891332	2690056610860898	joueur_580	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	893470	2573321465012002	joueur_615	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	894100	2710930876332218	joueur_655	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	894398	2542494231387188	joueur_297	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	894870	2632700264829909	joueur_706	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	894875	2632700264829909	joueur_706	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	900443	2779781155078810	joueur_317	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	903008	2642222702413791	JOUEUR_136	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	911883	2563350602769669	JOUEUR_221	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	911887	2753091070419728	joueur_811	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	915822	2642222702413791	JOUEUR_136	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	915886	2615815077049243	joueur_427	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	919459	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	934378	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	934414	2533274823110022	JGtm	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	934417	2533274823110022	JGtm	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	934509	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	936806	2710930876332218	joueur_655	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	938978	2793075222215887	joueur_728	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	940783	2753091070419728	joueur_811	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	944253	2550886075837404	JOUEUR_409	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	945318	2793075222215887	joueur_728	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	957260	2779781155078810	joueur_317	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	959636	2753091070419728	joueur_811	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	965168	2779781155078810	joueur_317	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	967379	2623439693412443	joueur_117	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	967644	2573321465012002	joueur_615	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	972276	2690056610860898	joueur_580	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	972278	2690056610860898	joueur_580	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	983327	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	984227	2779781155078810	joueur_317	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	984822	2753091070419728	joueur_811	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	986492	2642222702413791	JOUEUR_136	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	987062	2646046920107160	JOUEUR_390	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	987363	2779781155078810	joueur_317	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	987424	2690056610860898	joueur_580	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	989532	2753091070419728	joueur_811	2642222702413791	JOUEUR_136	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	993563	2779781155078810	joueur_317	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	997642	2573321465012002	joueur_615	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1003681	2690056610860898	joueur_580	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1003684	2690056610860898	joueur_580	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1005584	2573321465012002	joueur_615	2685352648935266	joueur_257	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1006180	2623439693412443	joueur_117	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1006312	2646046920107160	JOUEUR_390	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1009587	2690056610860898	joueur_580	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1011321	2779781155078810	joueur_317	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1016187	2779781155078810	joueur_317	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1018428	2779781155078810	joueur_317	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1019794	2753091070419728	joueur_811	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1025930	2779781155078810	joueur_317	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1028967	2573321465012002	joueur_615	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1031072	2646046920107160	JOUEUR_390	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1048718	2640661141794626	joueur_582	2573321465012002	joueur_615	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1049258	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1056003	2646046920107160	JOUEUR_390	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1059332	2793075222215887	joueur_728	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1059341	2632700264829909	joueur_706	2563350602769669	JOUEUR_221	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1059536	2710930876332218	joueur_655	2632700264829909	joueur_706	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1068108	2542494231387188	joueur_297	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1071151	2753091070419728	joueur_811	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1075182	2779781155078810	joueur_317	2788951673476170	JOUEUR_456	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1079892	2573321465012002	joueur_615	2634990399857364	JOUEUR_126	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1080223	2779781155078810	joueur_317	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1084791	2573321465012002	joueur_615	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1092667	2690056610860898	joueur_580	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1100444	2793075222215887	joueur_728	2533274823110022	JGtm	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1100474	2646046920107160	JOUEUR_390	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1103377	2685352648935266	joueur_257	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1104642	2573321465012002	joueur_615	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1107248	2753091070419728	joueur_811	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1107549	2690056610860898	joueur_580	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1108648	2685352648935266	joueur_257	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1111523	2779781155078810	joueur_317	2690056610860898	joueur_580	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1112589	2646046920107160	JOUEUR_390	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1115559	2779781155078810	joueur_317	2784870293334143	joueur_017	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1124832	2793075222215887	joueur_728	2779781155078810	joueur_317	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1129478	2793075222215887	joueur_728	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1135216	2793075222215887	joueur_728	2640661141794626	joueur_582	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1137882	2793075222215887	joueur_728	2615815077049243	joueur_427	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1150496	2573321465012002	joueur_615	2542494231387188	joueur_297	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1153497	2533274823110022	JGtm	2710930876332218	joueur_655	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1155433	2779781155078810	joueur_317	2793075222215887	joueur_728	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1169145	2690056610860898	joueur_580	2753091070419728	joueur_811	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1171282	2779781155078810	joueur_317	2682452436637450	joueur_218	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1176455	2550886075837404	JOUEUR_409	2646046920107160	JOUEUR_390	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1179227	2685352648935266	joueur_257	2798225113631945	joueur_172	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1184533	2793075222215887	joueur_728	2623439693412443	joueur_117	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1187060	2573321465012002	joueur_615	2550886075837404	JOUEUR_409	0
4f77afc1-d9f4-443b-b38f-3628341bf7e9	1188100	2793075222215887	joueur_728	2533274823110022	JGtm	0
```
