# PLAN — Rationalisation des notifications in-app (2026-07)

> Rédigé le 2026-07-07 après audit prod (VPS). Exécution par un agent ultérieur sous le
> contrat du skill `plan-execution` (OBLIGATOIRE : ordre strict, une phase à la fois,
> gate passé avant la suivante, aucun report d'item exécutable, statuts
> `[x]` / `[~]` (couvert ailleurs, référence) / `[!]` (non traité, justification écrite),
> zéro fix opportuniste hors périmètre — consigner dans « Découvertes »).
>
> Les numéros de ligne cités sont valides au commit `ccc950324`
> (branche `refactor/audits-2026-07`). RE-VÉRIFIER sur pièces avant chaque modification.

## Contexte et diagnostic (prouvé sur données prod)

JGtm : 57 notifications non lues, toutes du 2026-07-03 (16:05 → 21:39, une seule
soirée). Volume moyen faible (59 notifs en 6 semaines) : le problème est
**des bursts mécaniques + un cycle de vie inexistant**, pas le débit.

Décomposition des 57 (mesurée dans `shared_social.duckdb` prod, xuid 2533274823110022) :

| Cause | Notifs | Mécanisme code |
|---|---|---|
| Burst cold-start 16:05:32 (22 notifs / 1 s) | 22 | snapshot « before » à zéro → tout l'historique émis comme delta (`objective_completed count=3434`, `career_rank previous=0`, 6× `skill_tier previous_tier=""`) |
| Paire `objective_completed`+`objective_assigned` à chaque cycle de sync | 24 | les deux catégories branchées sur le MÊME champ `PersonalAwardCount` (`post_sync_deltas.go:81` et `:87`) |
| `skill_tier` sans hystérésis | 5 | flapping Or IV↔V arena_slayer : chaque franchissement (montée ET descente) notifie |
| `media_added` sans coalescence | 5 | 5 clips du même acteur en 5 min = 5 notifs |
| Records/near-miss coach multipliés par période | ~6 | même record émis 3× (30d/90d/all_time), near-miss en double |

Constat transverse : personne ne marque lu (JGtm 57/59 non-lues, Madina 32/32 depuis
avril, Chocoboflor 10/10). Le badge est un cumul, pas un signal.

Points d'appui code (vérifiés le 2026-07-07) :

- `apps/go-api/internal/api/wire/post_sync_deltas.go` — hook + émissions delta.
  `BuildPostSyncDeltaHook` (l.93) : si `SnapshotPlayerState` « before » échoue, log warn
  et **continue avec un snapshot zéro** (l.101-104). Table `postSyncCounterDeltas`
  (l.80-88). Émissions bespoke : career_rank (l.230), skill_tier (l.251), threshold
  (l.282, l.295), personal_record (l.308 — possède DÉJÀ la garde baseline
  `oldRec.Loaded`, c'est le modèle à suivre).
- `apps/go-api/internal/api/wire/post_sync_deltas_snapshot.go` — `SnapshotPlayerState`
  (l.61) dégrade en zéros sur toute erreur de lecture (Debug log) ; `pdb == nil ||
  pdb.Player == nil` → snapshot vide sans erreur (l.66-68). Un « before » à zéro est
  donc indistinguable d'un état réellement vide.
- `apps/go-api/internal/progression/coach/dedup.go` — `FilterRecent` (l.26) +
  `AnnotateDedupKey` (l.53) : le pattern de dédup 24 h existe déjà (fenêtre
  `ProgressionDedupWindow = 24h`, `post_sync_progression.go:57`, appel l.361).
- `apps/go-api/internal/progression/coach/generator.go` — RecordBroken (DedupKey
  `metric|period`, l.167), RecordNearMiss (l.182) : une alerte PAR période.
- `apps/go-api/internal/progression/coach/generator.go` (suite) — `buildRecordAlerts`
  (l.149-187) : émet `RecordBroken` même quand `PreviousValue == nil` (premier record
  = seed) — source des 4 `personal_record` du burst. `buildLUSRTierApproachAlert`
  (l.230-249) et `buildLOWESSAlerts`/`SoftNegative` (l.253-301) : alertes d'ÉTAT
  (re-émises tant que la condition dure, seul FilterRecent 24 h les espace).
  Sain par contraste : `buildStreakAlerts` (l.120-144) ne notifie que la transition
  exacte (`CurrentLength == palier`) — c'est le modèle événementiel à imiter.
- Constantes de seuils : `records/types.go:45` (`NearMissRatio = 0.05`),
  `milestones/types.go:17` (`NearMissRatio = 0.10`), `coach/types.go:83-97`
  (`MaxConcurrentUnread = 3` et `AutoDismissAfter = 7j` — MORTES, aucun consommateur ;
  `ComebackPauseThreshold = 5j` — sain ; `LUSRTierApproachDelta = 10.0`).
- `apps/go-api/internal/progression/records/detector.go` — `IsNearMiss` (l.226-231) :
  garde stricte `current < target` DÉJÀ en place (commentaire l.218-222 documentant le
  spam « X vs X »), mais insuffisante : l'incident prod porte `target=73.333336,
  value=73.33` — strictement inférieur en float, identique à l'affichage (2 décimales)
  → notif « 73.33 vs 73.33 » absurde. Constantes : `records/types.go` (`NearMissRatio
  = 0.05` l.45, `MinMatchesForRecord` l.41). `IsNearMiss` est le point de passage
  unique (détecteur + coach).
- `apps/go-api/internal/api/handlers/media.go` — `emitMediaAdded` (l.201-239) : un
  `Emit` par upload, aucune coalescence.
- `apps/go-api/internal/notifications/service.go` — `Emit`/`emitInner` (l.118-180),
  `CapAndSweep(500)` best-effort (l.178). `emitter.go:8` : interface `Emitter`.
- `apps/go-api/internal/notifications/types.go` — catégories (l.15-68), précédent de
  dépréciation `CategorySeasonPassLevel` (l.24-27), `AllCategories` (l.71+),
  `EmitInput` (l.152), struct `UnreadCount` (même fichier — localiser).
- `apps/go-api/internal/platform/duckdb/notifications_repo.go` — `Insert` (l.73, utilise
  `n.ID` fourni), `UnreadCount` (l.222), `MarkAllRead` (l.354 — modèle SQL pour le
  sweep D3), `CapAndSweep` (l.463). Persister : `social_persister_iface.go`.
- Front : `apps/web/src/features/notifications/NotificationsBell.tsx` (badge l.29-30,
  dropdown, `DROPDOWN_LIMIT=12`), `queries.ts`, `mutations.ts`,
  `apps/web/src/lib/query/keys.ts:159-167`.
- OpenAPI manuel : `apps/go-api/api/openapi.yaml` (+ `make generate-types`).

## Objectif et critère de succès

Réduire ~80 % du bruit à comportement joueur identique, et rendre le badge signifiant.

Critère mesurable (rejouer le scénario du 2026-07-03 dans les tests) :

1. Cold-start (before vide, after riche) → **0 émission** (au lieu de 22).
2. Cycle de sync avec N objectifs complétés → **1 notif** (au lieu de 2).
3. Flapping Or IV↔V sur 5 cycles en < 24 h → **≤ 1 notif** (au lieu de 5).
4. 5 uploads du même acteur en < 1 h → **1 notif coalescée count=5** (au lieu de 5).
5. Record KDA battu sur 30d+90d+all_time → **1 notif** (all_time) (au lieu de 3).
6. Badge cloche = non-lues `severity != info` ; ouverture puis fermeture du dropdown →
   les notifs affichées passent lues.
7. Near-miss avec écart < 2 % du PB (ex. 73.33 vs 73.333336) → **0 émission** (au lieu
   d'une notif « 73.33 vs 73.33 ») ; écart entre 2 % et 5 % → 1 émission.
8. Première évaluation records (`player_records` vide) → **0 notif** `personal_record`
   (au lieu de 4 : seed silencieux).
9. Nudge d'état (near-miss/approche/tendance) dont la condition persiste →
   **max 1 émission par 30 jours** par cible (au lieu d'une par jour).
10. Sync en panne durable (échecs répétés < 6 h d'intervalle) → **1 notif**
    `sync_error` coalescée (au lieu d'une par tentative).

## Décisions produit — TRANCHÉES (ne pas rouvrir en cours d'exécution)

Préfixe `DP` (décision produit) pour ne pas confondre avec les items `D1..D9` de la
phase D.

**Principe directeur (toutes catégories)** : une notification rapporte un ÉVÉNEMENT
(quelque chose vient de se passer, attribuable à l'activité récente), jamais un ÉTAT
(quelque chose est vrai depuis longtemps). Un écart doit être significatif pour un
joueur, pas seulement non-nul en float. Toute nouvelle catégorie devra passer ce test.

| # | Décision |
|---|---|
| DP1 | Anti-burst = **gardes** (snapshot froid + cap de vraisemblance), PAS de baseline persistée en table. Justification : le « before » est relu à chaque cycle, une anomalie ne dure qu'un cycle ; on évite la machinerie ADR 0026 d'une nouvelle table append-only. Rejeté : persister le snapshot. |
| DP2 | `objective_assigned` : émission post-sync **supprimée définitivement** (doublon exact de `objective_completed`). La catégorie reste déclarée (rétro-compat historique + prefs), commentaire de dépréciation daté, précédent `CategorySeasonPassLevel`. |
| DP3 | Records/near-miss coach : **une seule période, la plus large** (`all_time` > `90d` > `30d`), par métrique. |
| DP4 | `skill_tier` : émissions **montées uniquement** + dédup 24 h par (playlist, valeur cible). Démotions silencieuses. Tier inconnu (multi-titre) → fail-open : émettre sur changement. |
| DP5 | `media_added` : coalescence fenêtre **1 h** par (catégorie, acteur), uniquement si la notif candidate est **non lue**. Une notif lue n'est jamais ressuscitée. |
| DP6 | Badge = non-lues avec `severity != 'info'` (`badge_count` serveur). Le compteur complet reste exposé (`count`). |
| DP7 | Auto-read à la **fermeture** du dropdown (pas à l'ouverture — évite le flip visuel « Non lues »→« Anciennes » pendant la consultation). Le bouton « tout marquer lu » est conservé. |
| DP8 | Expiry douce : les `info` non lues depuis > **7 jours** passent lues (sweep best-effort à l'émission, à côté de `CapAndSweep`). Réalise l'intention de la constante morte `coach.AutoDismissAfter` (jamais câblée — item D9). |
| DP9 | `maxPlausibleCounterDelta = 20` par cycle de sync (le max légitime observé est 6). |
| DP10 | Aucune purge manuelle du stock prod existant : DP7 + DP8 le résorbent naturellement. |
| DP11 | Near-miss records : l'écart doit être **significatif pour un joueur** — bande d'émission `PB×(1-NearMissRatio) <= value <= PB×(1-NearMissMinGapRatio)` avec `NearMissMinGapRatio = 0.02` (2 %). À l'échelle KDA ~5.0 → écart >= 0.1 ; accuracy ~73 → écart >= ~1.5 pt. Relatif (pas d'absolu 0.01 : insignifiant sur toutes les échelles). Remplace la première version de cette décision (écart absolu 0.01, trop fin — un joueur se fiche d'un écart de deux décimales). |
| DP12 | **Premier record = seed silencieux** : `RecordBroken` avec `PreviousValue == nil` (première évaluation, `player_records` vide) n'émet PAS d'alerte — le PB est persisté sans notification. « Ton record est X » sans référence antérieure n'est pas une nouvelle (4 des 10 notifs coach du burst du 2026-07-03). Même principe que la garde `oldRec.Loaded` de post_sync. |
| DP13 | Nudges d'ÉTAT (`record_near_miss`, `milestone_near_miss`, `lusr_tier_approach`, `threshold_crossed` source coach/LOWESS, `trend_consolidate`) : fenêtre de dédup **30 jours** par (catégorie, dedup_key) au lieu de 24 h. Une condition qui persiste (μ proche d'un palier, progression à 95 % d'un milestone, tendance positive) ne doit pas re-notifier chaque jour — max 1 nudge par cible par mois. Les catégories ÉVÉNEMENT (records battus, streaks, unlocks) restent à 24 h. |
| DP14 | `milestones.NearMissRatio` : **0.10 → 0.02**. « Proche » à 90 % de 10 000 kills = 1 000 kills restants (des semaines) — pas une approche. À 98 % (200 kills) le dénouement est imminent, le nudge a du sens. |
| DP15 | `match_synced` : severity `success` → **`info`** (plomberie, pas un accomplissement — sort du badge via DP6, reste visible en liste/toast). `sync_error` : coalescence **6 h** (une panne durable = UNE notif dont le count s'incrémente, pas une par cycle de sync). |

## Branche Git

`refactor/notifications-rationalization`, à créer depuis `refactor/audits-2026-07`
si celle-ci n'est pas encore mergée dans `main`, sinon depuis `main` (le fichier
`post_sync_deltas.go` porte le refactor K1a présent sur la branche audits — vérifier
avec `git log --oneline -3 -- apps/go-api/internal/api/wire/post_sync_deltas.go`).
JAMAIS de travail sur `main` (push main = deploy prod auto).

## Effort estimé

Phase A : rapide. Phase B : moyen. Phase C : moyen. Phase D : moyen (back+front).
Phase E : rapide. Total : 1 session agent.

---

## Phase A — Anti-burst cold-start (`post_sync_deltas.go`)

- [ ] A1. Helper `snapshotLooksCold(s *PlayerSnapshot) bool` dans
      `post_sync_deltas.go` : vrai si TOUS les compteurs (`PersonalAwardCount`,
      `CitationsCount`, `ChallengePathsCount`, `ChallengeCompletedCount`,
      `BattlepassCompletedTracks`, `CitationTotalEarnedTiers`, `CitationMasteryCount`,
      `CurrentRank`) sont à 0 ET `len(SkillTierByPlaylist) == 0` ET `KDRatio == 0`.
- [ ] A2. Dans `EmitPostSyncDeltas` (l.191), tout en haut après le nil-check : si
      `snapshotLooksCold(before) && !snapshotLooksCold(after)` →
      `slog.WarnContext(ctx, "post_sync: snapshot before froid — émissions supprimées (cold-start)", "slug", slug)`,
      exécuter UNIQUEMENT le sous-bloc de persistance du record best_kda
      (l.329-335, seed silencieux — il a déjà sa garde `!oldRec.Loaded`), puis `return`.
- [ ] A3. Cap de vraisemblance dans la boucle counter deltas (l.205-225) : constante
      `maxPlausibleCounterDelta = 20` (commentée : max légitime observé = 6, incident
      prod 2026-07-03 = 3434) ; si `newV-oldV > maxPlausibleCounterDelta` →
      `slog.WarnContext` avec le delta + `continue` (pas d'émission).
- [ ] A4. Garde career_rank (l.230) : ne pas émettre si `before.CurrentRank == 0`
      (rang inconnu/non initialisé — l'incident montrait `previous:0`).
      `slog.DebugContext` en trace.
- [ ] A5. Tests `post_sync_deltas_test.go` (compléter l'existant, suivre ses
      helpers/patterns) : (a) before froid + after riche → 0 émission ;
      (b) delta objectifs = 25 → supprimé, les autres deltas du même cycle passent ;
      (c) delta = 5 → émis ; (d) career_rank previous=0 → supprimé ;
      (e) career_rank 190→192 → émis ; (f) before froid ET after froid (nouveau
      joueur vide) → 0 émission, pas de warn cold-start.

**Gate A** : `cd apps/go-api && go test ./internal/api/wire/`. Vert = phase close.

## Phase B — Dédoublonnage sémantique

- [ ] B1. Supprimer l'entrée `objective_assigned` de `postSyncCounterDeltas`
      (`post_sync_deltas.go:87`).
- [ ] B2. `types.go:20` : commentaire de dépréciation daté sur
      `CategoryObjectiveAssigned` (modèle : `CategorySeasonPassLevel` l.24-27 —
      « conservée pour rétro-compat des notifs déjà en DB + seed prefs, plus émise
      depuis 2026-07 »). NE PAS la retirer de `AllCategories`.
- [ ] B3. Garde-rail : test dans `post_sync_deltas_test.go` affirmant qu'aucun
      scénario post-sync n'émet `CategoryObjectiveAssigned` (émission autorisée
      uniquement = aucune ; le test échoue si quelqu'un rebranche la catégorie).
- [ ] B4. Adapter les tests existants qui attendent la paire assigned+completed
      (chercher : `grep -rn "objective_assigned" apps/go-api/internal/`).
- [ ] B5. Records coach — `generator.go` : collapse par métrique sur la période la
      plus large. Helper pur dans le package coach (ex. `keepWidestPeriod`) appliqué
      aux listes RecordBroken (l.155-170) ET RecordNearMiss (l.175-185) avant
      construction des alertes. Ordre : `all_time` > `90d` > `30d` (vérifier les
      valeurs exactes du type Period dans `internal/progression/records/` avant de
      coder — RE-VÉRIFIER).
- [ ] B6. Near-miss écart significatif (DP11) : constante
      `NearMissMinGapRatio = 0.02` dans `records/types.go` (commentaire : incident
      prod 2026-07-03 « 73.33 vs 73.333336 » rendu « 73.33 vs 73.33 » ; 2 % ≈ 0.1 de
      KDA sur un PB à 5.0) ; dans `IsNearMiss` (`detector.go:226-231`), remplacer
      `current < target` par `current <= target*(1-NearMissMinGapRatio)` et compléter
      le commentaire existant (l.215-225) qui documente déjà le cas d'égalité
      stricte — même rationale étendu aux écarts insignifiants. Point de passage
      unique : détecteur ET coach passent par `IsNearMiss`, aucun autre site à
      modifier (vérifier : `grep -rn "IsNearMiss" apps/go-api/internal/`).
- [ ] B7. Tests `records/detector_test.go` : écart 1 % sous le PB → pas de
      near-miss ; écart 3 % → near-miss ; écart 6 % (hors bande des 5 %) → pas de
      near-miss ; égalité stricte → pas de near-miss (cas existant à conserver) ;
      PB à 0 → pas de near-miss (garde existante).
- [ ] B8. Tests coach (`generator_test.go` existant à compléter) : record battu sur
      3 périodes → 1 alerte (all_time) ; battu sur 30d seul → 1 alerte (30d) ;
      near-miss 90d + all_time → 1 alerte (all_time) ; broken 30d + near-miss
      all_time (métriques différentes de cas) → les deux passent ; near-miss avec
      écart sous `NearMissMinDelta` → 0 alerte.
- [ ] B9. skill_tier montées uniquement : helper `skillTierRank(tier string,
      subTier int) int` dans `post_sync_deltas.go` (map insensible à la casse :
      Bronze=1, Silver=2, Gold=3, Platinum=4, Diamond=5, Onyx=6, Champion=7 — H5 ;
      tier inconnu → -1). Ne pas émettre si `rank(after) <= rank(before)` quand les
      deux rangs sont connus (≥ 0). Rang inconnu d'un côté → fail-open (émettre sur
      changement, comportement actuel). Placement dans une playlist nouvelle
      (`oldVal == ""` hors cold-start) → émettre.
      NB : politique de notification, pas un algo d'analyse — sa place est dans wire ;
      si un ordre de tiers canonique existe déjà (`grep -rn "Onyx" apps/go-api/internal/
      --include=*.go -l` puis inspection), le réutiliser au lieu de créer la map
      (skill `go-features`).
- [ ] B10. skill_tier dédup 24 h : suivre le câblage de `post_sync_progression.go`
      (qui charge les notifs récentes pour `FilterRecent`, appel l.361 — remonter à la
      source du paramètre `recent` pour réutiliser le même fetch). Dans
      `BuildPostSyncDeltaHook`, charger les notifs récentes catégorie `skill_tier`
      (limite 50) et les passer à `EmitPostSyncDeltas` (nouveau paramètre). Avant
      d'émettre : skip si une notif < 24 h porte le même `playlist_group` ET la même
      valeur cible `rating_type|tier|sub_tier` dans ses params. Ajouter
      `playlist_group` + valeur cible en clair dans les params émis si pas déjà le cas
      (c'est déjà le cas : l.264-272).
- [ ] B11. Tests skill_tier : séquence IV→V→IV→V→IV→V en < 24 h → 1 émission ;
      démotion V→IV → 0 ; montée Gold→Platinum → 1 ; tier inconnu « Mythril » →
      émet sur changement ; placement nouvelle playlist (before non froid) → 1.
- [ ] B12. Seed silencieux des records (DP12) : dans `buildRecordAlerts`
      (`generator.go:149-187`), le cas `r.NewPB` n'émet une alerte QUE si
      `r.PreviousValue != nil` (le détecteur a déjà persisté le PB — seed sans
      notification, `detector.go:126-167` inchangé). Commentaire renvoyant à la
      garde jumelle `oldRec.Loaded` de `post_sync_deltas.go:313`.
- [ ] B13. Fenêtre de dédup par catégorie (DP13) : remplacer l'unique
      `ProgressionDedupWindow = 24h` (`post_sync_progression.go:57`) par une
      résolution par catégorie — 30 jours pour les nudges d'état
      (`record_near_miss`, `milestone_near_miss`, `lusr_tier_approach`,
      `threshold_crossed`, `trend_consolidate`), 24 h pour le reste. Passer la
      fenêtre à `coach.FilterRecent` par alerte (signature à adapter,
      `dedup.go:26`). Vérifier que le fetch des notifs récentes passées à
      FilterRecent couvre bien 30 jours (remonter la construction du paramètre
      `recent` depuis l'appel `post_sync_progression.go:361` ; élargir la limite si
      elle est temporelle et non en nombre). NB : les `threshold_crossed` émis par
      post_sync (KD/winrate) ne passent pas par FilterRecent — hors sujet ici,
      pas de collision (dedup_key différent).
- [ ] B14. `milestones.NearMissRatio` 0.10 → 0.02 (`milestones/types.go:17`,
      DP14) + mise à jour du commentaire et des tests
      `milestones/detector_test.go` (cas : progress 0.95 → pas de near-miss ;
      0.985 → near-miss ; 1.0 → earned, pas de near-miss).
- [ ] B15. Tests dédup 30 j (`coach/dedup_test.go` ou équivalent existant) : une
      notif `lusr_tier_approach` émise il y a 5 jours avec le même dedup_key →
      alerte filtrée ; il y a 35 jours → alerte passe ; catégorie événement
      (`personal_record`) émise il y a 5 jours → passe (fenêtre 24 h).

**Gate B** : `cd apps/go-api && go test ./internal/api/wire/ ./internal/progression/... ./internal/notifications/`.

## Phase C — Coalescence `media_added`

- [ ] C1. Interface `Emitter` (`emitter.go:8`) : ajouter
      `EmitCoalesced(ctx context.Context, in EmitInput, window time.Duration) error`.
      Recenser et mettre à jour toutes les implémentations/fakes :
      `grep -rn "notifications.Emitter" apps/go-api/internal/ --include=*.go`.
- [ ] C2. `Service.EmitCoalesced` (`service.go`) : sous `withWriterBestEffort`,
      lister les ~20 dernières notifs de la catégorie (`repo.List`,
      `ListFilter{Category, Limit: 20}`) ; candidate = même catégorie, **non lue**
      (`ReadAt == nil`), même acteur (`Actor.Name`/params `actor_name`),
      `CreatedAt` dans la fenêtre. Trouvée → réémettre via `repo.Insert` avec le
      MÊME `ID`, `CreatedAt = now`, `params.count = ancien + nouveau` (le modèle
      append-only fait le reste : nouvel event même (xuid,id) → la vue `_latest`
      sert la version à jour, la notif remonte en tête). Pas trouvée → fallback
      émission normale. Extraire le code commun avec `emitInner` (seuil 80 L).
- [ ] C3. `media.go` `emitMediaAdded` (l.228) : remplacer `em.Emit(...)` par
      `em.EmitCoalesced(..., mediaCoalesceWindow)` avec
      `const mediaCoalesceWindow = time.Hour` (commentaire : incident 2026-07-03,
      5 notifs en 5 min pour 5 clips du même acteur).
- [ ] C4. Tests service (`service_test.go`, fakeRepo) : 2 émissions même acteur
      < 1 h → même ID, count sommé ; acteurs différents → 2 IDs ; > 1 h → 2 IDs ;
      candidate lue → nouvelle notif (jamais ressusciter une lue).
- [ ] C5. Test e2e DuckDB (`notifications_service_e2e_test.go`, package
      platform/duckdb) : 2 `EmitCoalesced` → `player_notifications_latest` contient
      1 ligne, count=2, non lue, `player_notifications_history` contient 2 events.
- [ ] C6. Vérifier le rendu i18n : `notif.media_added.body` consomme déjà
      `{count}` (`apps/web/src/features/notifications/i18n.ts`) — contrôler le
      pluriel FR/EN, corriger si « 5 clip » s'affiche.
- [ ] C7. `sync_error` coalescé (DP15) : `emitSyncError`
      (`sync_handler.go:274-294`) → `EmitCoalesced` fenêtre
      `syncErrorCoalesceWindow = 6h`. Préciser dans C2 la règle de matching :
      même catégorie + même acteur SI acteur présent, catégorie seule sinon
      (sync_error n'a pas d'acteur) ; params de la candidate mis à jour avec le
      dernier message, count incrémenté. Test : 3 échecs en 1 h → 1 notif
      count=3 avec le dernier message.

**Gate C** : `cd apps/go-api && go test ./internal/notifications/ ./internal/api/handlers/`
puis `go test -tags=integration -p 1 -run '^TestNotifications' ./internal/platform/duckdb/`
(filtre ANCRÉ `^`, sérialisé `-p 1` — cf. incident LOT B, faux verts sinon).

## Phase D — Cycle de vie du badge (back léger + front)

- [ ] D1. Backend `badge_count` : struct `UnreadCount` (`types.go`) — champ
      `BadgeCount int` json `badge_count` (« non-lues severity != 'info' »).
      Requête `NotificationsRepo.UnreadCount` (`notifications_repo.go:232-237`) :
      ajouter `COUNT(*) FILTER (severity <> 'info')` par catégorie et sommer dans
      la boucle de scan.
- [ ] D2. OpenAPI : ajouter `badge_count` au schéma unread-count dans
      `apps/go-api/api/openapi.yaml`, puis `make generate-types` (le garde-fou
      TestNoJSONRouteBypassesHuma existe — le laisser guider si autre chose est requis).
- [ ] D3. Front badge : `NotificationsBell.tsx:30` →
      `const unreadCount = countData?.badge_count ?? countData?.count ?? 0`.
      La page Notifications continue d'afficher le compteur complet.
- [ ] D4. Front auto-read à la fermeture : dans `NotificationsBell`, accumuler dans
      un `useRef<Set<number>>` les ids non lus rendus pendant l'ouverture ; sur
      transition open→false (click-outside, Esc, navigation), si le set est non vide
      → `markRead.mutate([...set])` puis vider. Vérifier que `mutations.ts` invalide
      bien le préfixe `notificationsAll` (keys.ts:161) pour rafraîchir badge + liste.
      Le bouton « tout marquer lu » reste.
- [ ] D5. Sweep expiry douce (DP8) : méthode persister
      `SweepStaleInfoNotificationsRead(ctx, xuid string, cutoff time.Time) error`
      (iface `social_persister_iface.go` + implémentation à côté de
      `MarkAllNotificationsRead` — la retrouver :
      `grep -rn "MarkAllNotificationsRead" apps/go-api/internal/platform/`) :
      INSERT d'events `read_at = now` pour `severity='info' AND read_at IS NULL AND
      created_at < cutoff` (SQL modèle : `MarkAllRead`, `notifications_repo.go:380-389`,
      + filtre severity/cutoff). Exposée via `Repository` (port.go) + repo, appelée
      best-effort dans `emitInner` à côté de `CapAndSweep` (l.178) avec
      `const staleInfoMaxAge = 7 * 24 * time.Hour`. Erreur → log warn, jamais propagée
      (même contrat que CapAndSweep, écriture idempotente).
- [ ] D6. Tests : repo/e2e — une info vieille de 8 j non lue + une émission → l'info
      passe lue, une `success` vieille de 8 j reste non lue ; UnreadCount →
      `badge_count` exclut les info. Front — test composant Bell (vitest, patterns
      des tests existants du dossier) : ouverture avec liste mockée (2 non lues) →
      fermeture → `markRead` appelé avec les 2 ids ; badge affiche `badge_count`.
      NB : vitest se lance HORS sandbox (`dangerouslyDisableSandbox`) ; typecheck OK
      en sandbox.
- [ ] D7. i18n : si de nouvelles strings UI apparaissent (a priori aucune), parité
      FR **et** EN dans `i18n.ts` (`Record<Locale, T>`), FR sans anglicisme.
- [ ] D8. `match_synced` severity `success` → `info` (DP15) : `emitMatchSynced`
      (`sync_handler.go:252-272`). Sort du badge via DP6, reste visible en
      liste. Adapter les tests qui vérifient la severity.
- [ ] D9. Code mort coach (règle CLAUDE.md n°7) : `MaxConcurrentUnread` et
      `AutoDismissAfter` (`coach/types.go:83-89`) ne sont consommées nulle part
      (vérifier : `grep -rn "AutoDismissAfter\|MaxConcurrentUnread"
      apps/go-api/`). Les SUPPRIMER — l'intention d'`AutoDismissAfter` (purge
      passive 7 j) est réalisée par l'item D5. Si le grep révèle un consommateur,
      statuer `[~]` avec référence et ne pas supprimer.

**Gate D** : `cd apps/go-api && go test ./...` ; `make check-types` ; `make test-web` ;
`make generate-types` puis `git diff --exit-code apps/web/src/lib/api/generated.ts`
(le commit doit contenir les types régénérés).

## Phase E — Gate final et clôture

- [ ] E1. Suite complète : `cd apps/go-api && go test ./...` puis
      `go test -tags=integration -p 1 ./internal/platform/duckdb/ ./internal/sync/`
      (OBLIGATOIRE : les écritures shared_social sont touchées — C2/D5).
- [ ] E2. `make go-api-lint` (dette baseline gelée : ne pas l'accroître),
      `make check-types`, `make test-web`.
- [ ] E3. Relire le diff complet : pas d'emoji, pas de `fmt.Println`, pas de couleur
      en dur côté web, seuils 500 L / 80 L respectés (`EmitPostSyncDeltas` porte déjà
      un nolint funlen documenté — ne pas l'aggraver, extraire si besoin).
- [ ] E4. Entrée `thought_log.md` (date, statut, décision, résultats, prochaine étape).
- [ ] E5. Skill `delivery-checklist` avant commit final. Commits par phase
      (`refactor(notif-A): ...` etc.) sur la branche unique. Demander à l'utilisateur
      avant merge — **push main = deploy prod automatique, prévenir**.
- [ ] E6. Statuer tous les items du plan ; consigner les découvertes hors périmètre
      ci-dessous sans les traiter.

**Critère de clôture global** : les 10 points du critère de succès sont couverts par
des tests nommés, tous les gates sont verts, aucun item sans statut.

---

## Hors périmètre (explicitement)

- Coalescence générique « par session » des counter deltas (fusionner les salves de
  cycles successifs en une notif récap) — à réévaluer si le bruit résiduel gêne encore.
- Écran de préférences par catégorie (existe côté API, UI non prioritaire).
- Notifs push/Discord (`internal/notify/`) — pipeline distinct, non concerné.
- Purge/backfill du stock prod existant (décision DP10).
- Fix de la formule quotient `best_kda` (DETTE K1a documentée dans
  `post_sync_deltas_snapshot.go:233-241` — re-backfill coordonné requis, NE PAS toucher).

## Découvertes en cours d'exécution (à consigner, ne pas traiter)

- (vide — à remplir par l'agent exécutant)

## Protocole de reprise de session

1. Lire cette section + les cases cochées ; reprendre à la première case non statuée
   de la phase la plus basse non close.
2. `git log --oneline -10` sur `refactor/notifications-rationalization` (un commit par
   phase attendu).
3. RE-VÉRIFIER les numéros de ligne cités (le code bouge) avant toute édition.
