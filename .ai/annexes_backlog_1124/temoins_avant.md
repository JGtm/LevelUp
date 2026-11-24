# Témoins « avant » des pistes A et B (item 0.3)

Annexe de `.ai/PLAN_BACKLOG_2026-11-24.md`. Capturé le 2026-11-24 sur la tête de `release/v8.3.0` (commit
`e884790c6b`), sans lancer le serveur ni un navigateur.

## Ce que ce fichier contient, et ce qu'il ne contient pas

- **Capturé ici** : les comptes SQL, en lecture seule, sur les bases du poste. Ils se rejouent tels quels
  après les correctifs.
- **Non capturé** : les sorties des pages (fil, cartes Némésis, onglet d'équipe, section Objectifs) sur
  trois matchs, que l'item 0.3 du plan demandait. Elles demandent un serveur lancé, ce que l'étape 0
  interdisait. La section 3 dit comment les prendre sans navigateur, par les tests existants.
- Un match est désigné par les 8 premiers caractères de son identifiant. Aucun joueur n'est nommé.

## 1. Outil et bases

Outil : `apps/go-api/cmd/diag_q` (ouvre la base en lecture seule, sortie en colonnes).

```bash
export CGO_ENABLED=1 CC=C:/msys64/ucrt64/bin/gcc.exe PATH="/c/msys64/ucrt64/bin:$PATH" GOWORK=off \
  GOCACHE="$LOCALAPPDATA/go-build-bl24-etape0"
go -C apps/go-api build -o "$TEMP/diag_q_e0.exe" ./cmd/diag_q
"$TEMP/diag_q_e0.exe" <base> "<requête>"
```

Bases lues :
- `HI` = `data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb` (2 045 matchs au registre, dont
  1 462 avec des éliminations dans `highlight_events`) ;
- `H5` = `data/titles/halo_5/warehouse/shared_matches_v2.duckdb` (3 032 matchs).

Les deux se sont ouvertes sans conflit : aucun serveur ne les tenait.

## 2. Comptes capturés

### 2.1 Éliminations en double dans `highlight_events` (A.1) — base HI

Requête :

```sql
WITH g AS (
  SELECT match_id, xuid, time_ms, count(*) c
  FROM highlight_events WHERE event_type = 'kill' GROUP BY 1, 2, 3
), m AS (
  SELECT match_id, sum(c) tot, sum(c - 1) surplus,
         sum(CASE WHEN c > 1 THEN c ELSE 0 END) lignes_en_double,
         count(*) FILTER (WHERE c > 1) groupes_doubles, count(*) groupes
  FROM g GROUP BY 1
)
SELECT count(*) matchs_kill,
       count(*) FILTER (WHERE surplus > 0) matchs_avec_double,
       count(*) FILTER (WHERE groupes_doubles * 1.0 / groupes >= 0.9) matchs_tout_double,
       sum(tot) lignes, sum(lignes_en_double) lignes_en_groupes_doubles, sum(surplus) lignes_surplus
FROM m;
```

Résultat :

| matchs_kill | matchs_avec_double | matchs_tout_double | lignes | lignes_en_groupes_doubles | lignes_surplus |
|---|---|---|---|---|---|
| 1 462 | **172** | 172 | 161 147 | **30 372** | **15 186** |

Attendu après correctif : `matchs_avec_double` = 0, `lignes_surplus` = 0, `lignes` = 145 961
(161 147 − 15 186).

Par classe (aucun double partiel, jamais plus de deux copies) :

```sql
WITH g AS (SELECT match_id, xuid, time_ms, count(*) c FROM highlight_events
           WHERE event_type = 'kill' GROUP BY 1, 2, 3),
     m AS (SELECT match_id, sum(c) tot, sum(c - 1) surplus,
                  count(*) FILTER (WHERE c > 1) gd, count(*) groupes, max(c) mx FROM g GROUP BY 1)
SELECT CASE WHEN gd = 0 THEN 'a_aucun' WHEN gd * 1.0 / groupes >= 0.9 THEN 'c_tout_double'
            ELSE 'b_partiel' END classe,
       count(*) matchs, sum(tot) lignes, sum(surplus) surplus, max(mx) max_copies
FROM m GROUP BY 1 ORDER BY 1;
```

| classe | matchs | lignes | surplus | max_copies |
|---|---|---|---|---|
| a_aucun | 1 290 | 130 775 | 0 | 1 |
| c_tout_double | 172 | 30 372 | 15 186 | 2 |

Forme des deux copies (dans l'ordre d'écriture) :

```sql
WITH d AS (SELECT match_id, xuid, time_ms FROM highlight_events
           WHERE event_type = 'kill' GROUP BY 1, 2, 3 HAVING count(*) > 1),
     l AS (SELECT h.*, CASE WHEN h.raw_json IS NOT NULL THEN 'avec_json'
                            WHEN h.type_hint IS NULL THEN 'sans_code'
                            ELSE 'code_sans_json' END forme
           FROM highlight_events h JOIN d USING (match_id, xuid, time_ms)
           WHERE h.event_type = 'kill'),
     p AS (SELECT match_id, xuid, time_ms, string_agg(forme, '+' ORDER BY id) combo,
                  max(id) - min(id) ecart FROM l GROUP BY 1, 2, 3)
SELECT combo, count(*) groupes, count(DISTINCT match_id) matchs, min(ecart) emin, max(ecart) emax
FROM p GROUP BY 1 ORDER BY 2 DESC;
```

| combo | groupes | matchs | emin | emax |
|---|---|---|---|---|
| sans_code+code_sans_json | 12 853 | 145 | 215 | 24 201 |
| code_sans_json+code_sans_json | 2 333 | 27 | 6 303 | 6 303 |

Lecture : 145 matchs écrits deux fois par deux écrivains (première synchronisation les 25 et 26 mai 2026) ;
27 matchs recopiés en bloc, à 6 303 lignes d'écart (première synchronisation de février à mai 2026).

Mois de jeu des 172 matchs : janvier 2026 (1), février (13), mars (7), avril (75), mai (76).

Les autres types sont doublés dans les mêmes matchs : morts dans 172 matchs, médailles dans 166.

Aucun double depuis :

```sql
WITH d AS (SELECT DISTINCT match_id FROM (
             SELECT match_id FROM highlight_events WHERE event_type = 'kill'
             GROUP BY match_id, xuid, time_ms HAVING count(*) > 1))
SELECT count(*) matchs_doubles,
       (SELECT count(*) FROM match_registry
         WHERE first_sync_at >= TIMESTAMP '2026-05-27') matchs_sync_depuis_27_mai,
       (SELECT count(*) FROM d JOIN match_registry r USING (match_id)
         WHERE r.first_sync_at >= TIMESTAMP '2026-05-27') dont_doubles
FROM d;
```

| matchs_doubles | matchs_sync_depuis_27_mai | dont_doubles |
|---|---|---|
| 172 | 321 | 0 |

### 2.2 Temps forts « mode » en double et sans code (A.2) — base HI

Par code :

```sql
SELECT type_hint, count(*) n, count(DISTINCT match_id) m
FROM highlight_events WHERE event_type = 'mode' GROUP BY 1 ORDER BY 2 DESC;
```

| type_hint | n | m |
|---|---|---|
| 10 | 21 353 | 484 |
| NULL | 2 330 | 76 |

Doubles et forme seule :

```sql
WITH s AS (SELECT match_id, xuid, time_ms, count(*) c FROM highlight_events
           WHERE event_type = 'mode' AND type_hint IS NULL
             AND (raw_json IS NULL OR raw_json = '') GROUP BY 1, 2, 3),
     k AS (SELECT DISTINCT match_id, xuid, time_ms FROM highlight_events
           WHERE event_type = 'mode' AND type_hint IS NOT NULL),
     ms AS (SELECT DISTINCT match_id FROM highlight_events
            WHERE event_type = 'mode' AND type_hint IS NOT NULL)
SELECT count(DISTINCT s.match_id) matchs_sans_code,
       sum(s.c) lignes_sans_code,
       count(DISTINCT s.match_id) FILTER (WHERE s.match_id IN (SELECT match_id FROM ms)) matchs_avec_aussi_code,
       sum(s.c) FILTER (WHERE s.match_id IN (SELECT match_id FROM ms)) lignes_dans_ces_matchs,
       sum(s.c) FILTER (WHERE k.match_id IS NOT NULL) lignes_doubles_d_un_code,
       count(DISTINCT s.match_id) FILTER (WHERE s.match_id NOT IN (SELECT match_id FROM ms)) matchs_sans_code_seul,
       max(s.c) max_copies
FROM s LEFT JOIN k ON k.match_id = s.match_id AND k.xuid = s.xuid AND k.time_ms = s.time_ms;
```

| matchs_sans_code | lignes_sans_code | matchs_avec_aussi_code | lignes_dans_ces_matchs | lignes_doubles_d_un_code | matchs_sans_code_seul | max_copies |
|---|---|---|---|---|---|---|
| **76** | **2 330** | 33 | 1 284 | **1 284** | **43** | 1 |

Attendu après correctif : `lignes_doubles_d_un_code` = 0. Le sort des 43 matchs « sans code seul » dépend du
choix fait au lot (poser le code 10, ou laisser).

Dates : les 33 matchs doublés ont été synchronisés les 25 et 26 mai 2026 ; les 43 autres du 8 juin au
1er septembre 2026.

Étendue de la forme sans code, tous types (comptes par type, base HI) :

| forme | type | lignes | matchs |
|---|---|---|---|
| sans code ni `raw_json` | death | 30 901 | 346 |
| sans code ni `raw_json` | kill | 30 945 | 346 |
| sans code ni `raw_json` | medal | 5 263 | 145 |
| sans code ni `raw_json` | mode | 2 330 | 76 |
| code sans `raw_json` | kill | 29 424 | 300 |
| avec `raw_json` | kill | 100 778 | 961 |

Sur les 346 matchs à éliminations sans code, 201 n'ont que cette forme. Matchs à éliminations sans code,
par mois de première synchronisation : mai 2026 (145), juin (50), juillet (119), août (22), septembre (10).

### 2.3 Matchs marqués « film absent » dont le film est à l'archive (A.3) — base HI et disque

Liste des matchs marqués (bit 22 de `backfill_completed`) :

```sql
SELECT left(match_id, 8) FROM match_registry WHERE (backfill_completed & (1 << 22)) <> 0;
```

616 matchs. Croisée avec les dossiers de `data/cache/film_chunks/` (un dossier par match) et les rejeux de
`data/cache/replays/halo_infinite/` (commande `comm -12` sur les deux listes triées, après retrait des fins
de ligne Windows de la sortie de l'outil) :

| Mesure | Valeur |
|---|---|
| matchs marqués « film absent » | 616 sur 2 045 |
| dont un dossier de film existe à l'archive | **36** (34 avec des morceaux, de 14 à 63 ; 2 dossiers vides) |
| dont un rejeu généré existe | 33 |

Attendu après correctif : 0 match marqué dont le film est complet à l'archive.

### 2.4 Durée du registre plus courte que le match (A.3) — base HI

Les trois matchs cités par le backlog :

```sql
SELECT left(r.match_id, 8) m, duration_seconds dur,
       date_diff('second', start_time, end_time) fin_moins_debut,
       (SELECT max(time_ms) / 1000 FROM match_kill_events_latest k WHERE k.match_id = r.match_id) derniere_mort_s,
       (SELECT max(time_played_seconds) FROM match_participants p WHERE p.match_id = r.match_id) tp_max,
       (SELECT count(*) FROM match_participants p WHERE p.match_id = r.match_id) joueurs,
       real_start_source, (backfill_completed & (1 << 22)) <> 0 film_absent
FROM match_registry r WHERE left(match_id, 8) IN ('58e6f72a', 'ed3b6497', 'dd29957c');
```

| m | dur | fin_moins_debut | derniere_mort_s | tp_max | joueurs | source | film_absent |
|---|---|---|---|---|---|---|---|
| ed3b6497 | 21 | 21 | 460 | 462 | 14 | api | non |
| dd29957c | 24 | 24 | 674 | 677 | 30 | api | non |
| 58e6f72a | 592 | 592 | 589 | 592 | 25 | api | oui |

Portée :

```sql
SELECT count(*) FROM match_registry r
WHERE duration_seconds IS NOT NULL
  AND duration_seconds * 1000 + 5000 <
      (SELECT max(time_ms) FROM match_kill_events_latest k WHERE k.match_id = r.match_id);
```

Résultat : **23** matchs ont une mort lue plus de 5 s après la fin déclarée. Une liste de 30 matchs (limite
de la requête) a un temps de jeu d'un joueur supérieur de plus d'une minute à la durée ; les deux plus
courts sont les deux matchs cités.

### 2.5 Éliminations à victime ambiguë (B.1) — bases HI et H5

```sql
WITH k AS (
  SELECT match_id, feed_killer_xuid x, time_ms,
         count(DISTINCT coalesce(nullif(victim_xuid, ''), victim_gamertag)) v, count(*) n
  FROM match_kill_events_latest
  WHERE coalesce(feed_killer_xuid, '') <> '' GROUP BY 1, 2, 3
)
SELECT count(*) FILTER (WHERE v > 1) cles_ambigues,
       coalesce(sum(n) FILTER (WHERE v > 1), 0) eliminations_sans_victime_nommee,
       count(DISTINCT match_id) FILTER (WHERE v > 1) matchs
FROM k;
```

| base | cles_ambigues | eliminations_sans_victime_nommee | matchs |
|---|---|---|---|
| HI | 0 | 0 | 0 |
| H5 | 3 | 6 | 3 |

L'entrée du backlog (2026-11-22) donnait 4 éliminations dans 1 match pour Halo Infinite et 4 dans 2 matchs
pour Halo 5 ; la base Halo Infinite a été réécrite depuis par la régénération du parc. Les éliminations
retirées du fil parce qu'antérieures au coup d'envoi ne sont pas comptées ici.

### 2.6 Halo 5 : matchs sans ligne dans `highlight_events` (0.4, D.4) — base H5

```sql
SELECT count(*) matchs,
       count(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM match_participants p WHERE p.match_id = r.match_id)) sans_participants,
       count(*) FILTER (WHERE NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id)) sans_he
FROM match_registry r;
```

| matchs | sans_participants | sans_he |
|---|---|---|
| 3 032 | 0 | 303 |

Types présents dans `highlight_events` : médailles 272 159 lignes (2 698 matchs), assistances 103 072
(2 712), mode 39 891 (2 384). Aucune élimination : elles vivent dans d'autres tables.

Les 303, selon qu'une table dérivée de la chronologie est remplie (précision par arme ou paires
tueur-victime) :

```sql
WITH s AS (
  SELECT r.*,
    (EXISTS (SELECT 1 FROM weapon_accuracy k WHERE k.match_id = r.match_id)
     OR EXISTS (SELECT 1 FROM killer_victim_pairs k WHERE k.match_id = r.match_id)) derive,
    (SELECT coalesce(sum(kills), 0) FROM match_participants p WHERE p.match_id = r.match_id) frags,
    (SELECT coalesce(sum(assists), 0) FROM match_participants p WHERE p.match_id = r.match_id) aides,
    (SELECT count(*) FROM match_participants p WHERE p.match_id = r.match_id) joueurs
  FROM match_registry r
  WHERE NOT EXISTS (SELECT 1 FROM highlight_events h WHERE h.match_id = r.match_id))
SELECT derive, count(*) n,
       count(*) FILTER (WHERE frags = 0) zero_frag,
       count(*) FILTER (WHERE frags > 0) avec_frags,
       count(*) FILTER (WHERE frags > 0 AND duration_seconds >= 120) avec_frags_2min,
       count(*) FILTER (WHERE joueurs = 1) solo,
       count(*) FILTER (WHERE aides > 0) avec_aides
FROM s GROUP BY 1;
```

| derive | n | zero_frag | avec_frags | avec_frags_2min | solo | avec_aides |
|---|---|---|---|---|---|---|
| non | 171 | 105 | **66** | 65 | 121 | 41 |
| oui | 132 | 20 | 112 | 110 | 123 | 0 |

Années de jeu des 303 : 2015 (94), 2016 (144), 2017 (16), 2018 (35), 2019 (6), 2020 (5), 2021 (2),
2023 (1). Toute la base a été synchronisée le 23 juin 2026.

### 2.7 Échelle des unités du jeu (0.6, B.6) — base HI

Distance entre le tueur et la victime à l'instant de l'élimination, en unités du jeu, par catégorie de
dégât :

```sql
WITH j AS (
  SELECT k.source_category c,
         sqrt(pow(p.killer_x - p.victim_x, 2) + pow(p.killer_y - p.victim_y, 2)
              + pow(coalesce(p.killer_z, 0) - coalesce(p.victim_z, 0), 2)) d
  FROM match_kill_events_latest k
  JOIN kill_positions_latest p
    ON p.match_id = k.match_id AND p.killer_xuid = k.feed_killer_xuid AND p.time_ms = k.time_ms
  WHERE p.killer_x IS NOT NULL AND p.victim_x IS NOT NULL)
SELECT c, count(*) n, round(quantile_cont(d, 0.05), 2) p05, round(quantile_cont(d, 0.25), 2) p25,
       round(median(d), 2) med, round(quantile_cont(d, 0.75), 2) p75,
       round(quantile_cont(d, 0.95), 2) p95, round(quantile_cont(d, 0.99), 2) p99
FROM j GROUP BY 1 ORDER BY 2 DESC;
```

| catégorie | n | p05 | p25 | médiane | p75 | p95 | p99 |
|---|---|---|---|---|---|---|---|
| None | 90 521 | 0,24 | 1,52 | 3,84 | 7,50 | 17,90 | 30,76 |
| Headshot | 35 986 | 2,06 | 4,70 | 7,74 | 13,44 | 27,69 | 41,55 |
| AttachedDamage | 5 071 | 1,21 | 3,16 | 4,75 | 6,61 | 10,09 | 14,63 |
| HeadshotMultiplier | 3 038 | 1,74 | 4,03 | 6,57 | 10,04 | 20,13 | 30,65 |
| **SilentMelee** | 2 833 | 0,16 | 0,27 | **0,34** | 0,46 | 0,78 | 1,34 |
| VehicleTransferDamage | 1 723 | 1,61 | 5,01 | 10,52 | 20,37 | 51,45 | 264,82 |
| CollisionDamage | 632 | 0,62 | 0,83 | 0,96 | 1,17 | 13,67 | 67,90 |
| ChainedProjectile | 268 | 1,60 | 4,13 | 6,24 | 9,56 | 21,42 | 37,60 |

Lecture : une élimination au corps à corps par derrière se fait à 0,34 unité en médiane. Deux corps ne
peuvent pas être à 34 cm l'un de l'autre ; à 3,048 m par unité, cela donne 1,04 m. Ce témoin sert aussi de
contrôle après B.6 : les valeurs en base ne doivent pas changer, seul l'affichage change.

## 3. Témoins capturables par les tests existants (sans serveur)

Aucun de ces tests n'a été lancé à l'étape 0. Le pilote ou l'agent de la piste lance la commande AVANT son
premier changement, garde la sortie, et la rejoue après.

### Piste A

| Ce qui est tenu | Test ou commande | Limite |
|---|---|---|
| Écriture de la complétion (événements, paires, bits) | `go test -tags=integration ./internal/persist/ -run EventsCompletion` | aucun test n'écrit deux fois le même match : c'est le test à ajouter en A.1 |
| Rejeu des événements | `go test -tags=integration ./internal/sync/ -run 'Replay\|EventsReplay'` | idem, le cas « match déjà rempli » manque |
| Garde anti-ART et listes datées | `go test ./internal/sync/ -run 'ART\|Allowlist'` | doit rester vert sans entrée neuve |
| Coup d'envoi : résolution, rattrapage, publication | `go test -tags=integration ./internal/sync/kickoff/... ./internal/sync/replayartifacts/...` ; `go test ./internal/analysis/timeline/...` | |
| Écriture unique du coup d'envoi | `go test ./internal/archlint/ -run Kickoff` | |
| Médailles du bouclier | `go test -tags=integration ./internal/sync/replayartifacts/ -run Usage` ; `go test ./internal/migration/ -run Medals` | |
| Catalogue de cartes | `go test ./internal/mapcatalog/ -run Concurrent -count=50` | témoin « avant » = le nombre d'échecs sur 50 (l'entrée dit 1 sur 10) |
| Comptes en base | les requêtes de la section 2 | à rejouer sur une COPIE migrée, puis en production en lecture seule, le user prévenu |

### Piste B

| Ce qui est tenu | Test ou commande | Limite |
|---|---|---|
| Fil des éliminations (arme, victime, bots) | `go test ./internal/service/ -run 'KillFeed\|Killfeed'` (`match_view_killfeed_weapon_test.go`, `match_view_killfeed_bots_test.go`) | |
| Vue du match de bout en bout, chronologie | `go test ./internal/service/ -run 'MatchView'` (dont `match_view_t0_golden_test.go`, `match_view_chronologie_test.go`) | |
| Rencontres de l'onglet d'équipe | `go test ./internal/service/ -run Encounters` ; `go test ./internal/platform/duckdb/ -run Encounter` | l'ordre à égalité n'est pas testé : témoin à ajouter en B.2 |
| Cartes Némésis | `npx vitest run src/features/match-view/MatchNemesisCards --maxWorkers=2` | |
| Section Objectifs | `npx vitest run src/features/match-view/objectivesChart src/features/match-view/MatchObjectivesSection --maxWorkers=2` | |
| Impact des coéquipiers | `go test ./internal/analysis/squadimpact/ ./internal/service/teammates/ -run Impact` ; `npx vitest run src/features/squad/impact-sessions --maxWorkers=2` | |
| Contrat d'API | `make generate-types` puis `git diff --stat apps/go-api/api/openapi.yaml apps/web/src/lib/api/generated.ts` | témoin « avant » = aucun écart |

### Le fichier golden ne sert pas de témoin

`apps/go-api/tests/fixtures/golden_values/match_view_slayer.json` n'est contrôlé que sur la présence de deux
clés (`header`, `summary_tab`) par `apps/go-api/tests/golden/golden_test.go:133-137`. Il porte encore un
bloc `team_tab.nemesis` que le code ne produit plus : un changement de la vue du match ne le ferait pas
rougir. Ne pas s'appuyer dessus pour B.1, B.2 ni B.5.

### Pour avoir les sorties des pages sur trois matchs

Deux voies, sans navigateur :
1. Un test de service qui sérialise la vue du match de trois matchs réels amincis (fixtures pseudonymisées
   sous `apps/go-api/internal/testfixtures/`) et compare la sortie à un fichier figé. C'est un témoin
   durable, à écrire en tête de la piste B.
2. À défaut, le pilote lance le serveur de développement sur une copie des bases (chemins des réglages
   neutralisés avant) et range les réponses JSON de la vue du match. Hors du périmètre de l'étape 0.

Matchs proposés comme témoins, par ce qu'ils exercent :
- un match de la classe « tout doublé » (section 2.1) : son fil doit perdre exactement la moitié de ses
  lignes d'élimination après A.1 ;
- un match de Bases avec temps forts « mode » en double (section 2.2) : la citation des captures de zone
  doit baisser ;
- un match à drapeau ou à zones pour la section Objectifs (B.3) : colonnes neuves, totaux d'équipe
  inchangés pour les colonnes existantes.
