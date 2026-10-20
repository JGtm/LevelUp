# HANDOFF — Fiabilisation sync : données de combat + citations vides (onglet Détails)

> Fichier de continuité anti-perte-de-contexte. **Mettre à jour à chaque étape.**
> Plan complet approuvé : `C:\Users\Guillaume\.claude\plans\les-matchs-d-hier-a-whimsical-lark.md`
> Audit cause racine : `.ai/AUDIT_SYNC_COMBAT_RO_2026-05-31.md`

## ✅ PHASE 1b SLICE 1 LIVRÉE (2026-06-01, non commité) — lire en premier
- **Créé** `internal/persist/events_completion_persister.go` : `EventsCompletionPersister` avec
  `Persist(ctx, EventsCompletionInput) (eventsInserted int, err error)` (1 TX : INSERT highlight_events
  INSERT-only + DELETE/INSERT killer_victim_pairs par-kill + UPDATE match_registry bits/events_loaded)
  ET `MarkNoFilmDefinitive(ctx, matchID, eventsBit)` (UPDATE registry no-film). Types `HLEventCompletion`
  + `KVPairCompletion`. ART-safety : highlight_events INSERT-only (ART-indexée), kv DELETE+INSERT (sans index).
- **Recâblé** `engine_highlight_events.go` : `ProcessHighlightEvents` + `insertHighlightEventsFromData`
  passent par le helper `persistCombatCompletion(ctx, sharedDB, matchID, events)` → le persister. Branche
  no-film → `MarkNoFilmDefinitive`. Plus aucun `db.Exec` shared dans le chemin de complétion.
- **Test** `internal/persist/events_completion_persister_test.go` (5 tests, opener dédié schéma kv par-kill).
- **Vérifs** : build+vet clean ; persist+sync verts (cgo ET cgo+integration). NON commité.
- **Conséquence** : `InsertHighlightEvents`, `InsertKillerVictimPairsFromEvents`, `MarkEventsLoaded`,
  `MarkKillerVictimLoaded` (writes.go) ne sont plus appelées qu'en tests (writes_test.go) → à SUPPRIMER
  avec leurs tests dans le commit garde-fou (item 8), en même temps que le test interdisant db.Exec shared
  hors persist (allowlist pour slice-2 skill/weapon encore legacy).
## ✅ PHASE 1b SLICE 2 LIVRÉE (2026-06-01, non commité)
- **Créé** `internal/persist/skill_completion_persister.go` : `SkillCompletionPersister.Persist` (UPDATE
  match_participants colonnes skill + UPDATE match_registry skill_loaded/MBitSkill en 1 TX). Types
  `ParticipantSkillUpdate` + `SkillCompletionInput`.
- **Recâblé** `internal/sync/skill_heal_apply.go::applySkillToParticipants` → délègue au persister (plus
  de tx.ExecContext direct). Imports nettoyés (fmt/slog retirés, persist/strconv ajoutés).
- **Test** `internal/persist/skill_completion_persister_test.go` (2 tests). Build+vet clean, persist+sync verts.
- **Phase 1b TERMINÉE** : events + killer_victim + skill de complétion passent tous par persist.

## ✅ ÉTAT après Phase 1b (récap pour item 8 = garde-fou + cleanup)
Écritures shared encore en `db.Exec`/`tx.Exec` direct DANS internal/sync (à traiter item 8) :
1. `writes.go` : `InsertHighlightEvents`, `InsertKillerVictimPairsFromEvents`, `MarkEventsLoaded`,
   `MarkKillerVictimLoaded` — désormais appelées UNIQUEMENT par writes_test.go → SUPPRIMER (fonctions +
   tests). Le comportement est couvert par events_completion_persister_test.go.
2. `backfill_weapons.go` : `InsertWeaponKills` (weapon_kills heal) — pas encore migré. Auditer + router
   via un persister weapon (slice 3 optionnelle) OU allowlister explicitement.
3. `writes.go` : `InsertParticipants` (utilisé par le chemin primaire legacy insertFetchedMatch ?) — à
   vérifier ; si primaire legacy, allowlister (le primaire batch passe déjà par SharedPersister).
Garde-fou : test scannant internal/sync/** interdisant Exec/Prepare INSERT|UPDATE|DELETE sur tables
shared hors persist, avec allowlist explicite (2 + 3 ci-dessus) documentée, vide à terme. Modèle :
internal/sync/no_art_patterns_test.go.

## 🔧 PHASE 1b — design d'origine (slice 1 = FAIT ci-dessus, garder pour slice 2/contexte)

**Objectif** : la complétion events/killer_victim (et skill) ne doit plus écrire en `db.Exec` direct
dans le package `sync` ; tout passe par le package `persist`, en UNE transaction atomique, sur le writer
RW (le `sharedDB` reçu par le post-sync EST déjà le writer provider RW — cf. correction §CORRECTION).

**Slice 1 (events + killer_victim) — design validé par lecture code :**
- Cible à remplacer : `ProcessHighlightEvents` (engine_highlight_events.go) fait aujourd'hui 3 écritures
  séparées NON atomiques : `InsertHighlightEvents` (INSERT OR IGNORE), `InsertKillerVictimPairsFromEvents`
  (DELETE+INSERT), `MarkEventsLoaded`/`MarkKillerVictimLoaded` (UPDATE match_registry bits). Partial-write
  possible + db.Exec direct = exactement ce que le cadrage interdit.
- `SharedPersister.Persist` INUTILISABLE ici : no-op si `batch.Shared.Match==nil` (shared_persister.go:67)
  + idempotence-skip si match_registry existe. La complétion écrit events-only sur un match déjà inséré.
- DTO réutilisables : `persist.HighlightEventInsert{MatchID,EventType,TimeMS,XUID,DetailsJSON}` (DetailsJSON
  = colonne type_hint) et `persist.KillerVictimInsert{MatchID,KillerXUID,VictimXUID,Count}` (rows.go:49/58).
  Helpers tx EXISTANTS (unexported, shared_persister.go:293 persistHighlightEvents = INSERT OR IGNORE ;
  :273 persistKillerVictim = INSERT pur). ⚠️ kv en complétion doit rester DELETE-then-INSERT (idempotent
  sur re-run) → le nouveau persister fait son propre DELETE WHERE match_id puis INSERT (ne PAS réutiliser
  persistKillerVictim nu qui n'a pas le DELETE).
- **À CRÉER** : `internal/persist/events_completion_persister.go` :
  `type EventsCompletionPersister struct { db txBeginner }` (txBeginner = interface BeginTx, déjà dans
  shared_persister.go:34). Méthode `Persist(ctx, in EventsCompletionInput) error` en 1 TX :
  (1) INSERT OR IGNORE highlight_events (si len>0) ; (2) DELETE killer_victim_pairs WHERE match_id +
  INSERT kv ; (3) UPDATE match_registry SET events_loaded=TRUE, backfill_completed |= MBitEvents (si events
  insérés) ; (4) backfill_completed |= MBitKillerVictim (si kv ok). Bits : valeurs dans
  internal/sync/backfill_flags.go (MBitEvents, MBitKillerVictim) — les passer en params (le package persist
  ne doit pas importer sync : passer les int64 depuis le caller).
- **Câblage** : dans `engine_highlight_events.go::ProcessHighlightEvents`, après parse, construire les
  `[]HighlightEventInsert` + calculer les pairs (réutiliser `analysis.ComputeKillerVictimPairs` comme le
  fait `InsertKillerVictimPairsFromEvents`), puis `persist.NewEventsCompletionPersister(sharedDB).Persist(...)`.
  Garder l'upsert xuid_aliases vers globalDB (DB séparée, hors TX shared — best-effort inchangé).
  Conserver le marquage events_loaded SOUS la politique film-retardé (isNoFilmDefinitive) déjà en place.
- **Tests** : nouveau `events_completion_persister_test.go` (persist) : insert events+kv atomique,
  idempotence re-run (kv pas dupliqués), bits marqués. + golden/bitmask/film_retry restent verts (ils
  passent par ProcessHighlightEvents → couvrent le câblage).
- **Garde-fou** (item suivant) : test scannant `internal/sync/**` pour interdire `db.Exec`/`Prepare` direct
  d'INSERT/UPDATE/DELETE sur tables shared hors persist (modèle no_art_patterns_test.go). Allowlist pour
  les sites legacy pas encore migrés (writes.go primaire) avec TODO, vide à terme.
- ⚠️ **GOTCHA SCHÉMA killer_victim_pairs (critique, découvert 2026-05-31)** : DEUX schémas divergents !
  - Legacy completion (`InsertKillerVictimPairsFromEvents`, writes.go:472) écrit **par-kill** :
    `(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, time_ms, created_at)`,
    kill_count laissé à DEFAULT 1. C'est CE schéma que lit le match-view (tug-of-war, KD timeline →
    time_ms requis, antagonistes → gamertags).
  - persist.`persistKillerVictim` (shared_persister.go:273) écrit **agrégé** :
    `(match_id, killer_xuid, victim_xuid, kill_count, created_at)` — PAS de gamertags, PAS de time_ms.
    → LOSSY. NE PAS le réutiliser pour la complétion (régresserait les graphes).
  - Table réelle (schema.go) : match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag,
    kill_count DEFAULT 1, time_ms, is_validated DEFAULT FALSE, created_at.
  - DÉCISION : `EventsCompletionPersister` écrit le schéma LEGACY par-kill (avec gamertags + time_ms).
    Input dédié `KVPairCompletion{KillerXUID,KillerGamertag,VictimXUID,VictimGamertag,TimeMS}` (NE PAS
    réutiliser persist.KillerVictimInsert qui est la forme agrégée lossy). Le caller construit ces rows
    depuis `analysis.ComputeKillerVictimPairs` (retourne .KillerXUID/.KillerGT/.VictimXUID/.VictimGT/.TimeMS,
    tolérance 5ms) — exactement comme le legacy.
  - MBit values (à passer en params depuis sync, persist n'importe pas sync) : MBitEvents=1<<16 (65536),
    MBitKillerVictim=1<<19 (524288).
- **Construction `[]HighlightEventInsert` (mirror EXACT du primaire collect.go:97-107)** :
  `persist.HighlightEventInsert{MatchID: matchID, XUID: &xuidStr (strconv.FormatUint(ev.XUID,10)),
  EventType: ev.EventType, TimeMS: ev.TimeMS, DetailsJSON: nil}`. NB : le primaire met `DetailsJSON: nil`
  (type_hint NULL) — le legacy completion écrivait ev.TypeHint mais la plupart des matchs existants ont
  type_hint NULL (écrits via primaire). Mirror primaire = `nil` (le DTO ne peut pas porter un int de toute
  façon). EventsInserted = somme des RowsAffected>0 (INSERT OR IGNORE) pour décider du marquage events_loaded.
- **Construction Pairs** : `raw := []analysis.RawEvent` filtré sur EventType kill/death (XUID=FormatUint,
  Gamertag, TimeMS=int64(ev.TimeMS)) ; `pairs := analysis.ComputeKillerVictimPairs(raw, 5)` ;
  map → `KVPairCompletion{p.KillerXUID, p.KillerGT, p.VictimXUID, p.VictimGT, p.TimeMS}`.
- **API persister cible** : `NewEventsCompletionPersister(db txBeginner)` ;
  `Persist(ctx, EventsCompletionInput) (eventsInserted int, err error)` en 1 TX :
  (1) INSERT OR IGNORE highlight_events ; (2) si len(Pairs)>0 : DELETE kv WHERE match_id + INSERT par-kill
  (match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag, time_ms, created_at) ;
  (3) UPDATE match_registry une seule fois : `backfill_completed |= mask`, `events_loaded = events_loaded OR ?`
  où mask = (EventsBit si eventsInserted>0) | (KillerVictimBit si MarkKV). Input porte EventsBit/KillerVictimBit
  (int64, passés par le caller sync) + MarkKV bool. events_loaded passe TRUE seulement si eventsInserted>0.
- **Slice 2 (skill heal)** : `healSkill`→`InsertParticipants` (db.Exec) idem → persister participants
  completion. Plus petit, faire après slice 1 validée.
- **NE PAS** toucher au chemin PRIMAIRE batch (collect.go→SharedPersister) qui marche déjà.
- Vérif obligatoire AVANT commit : `cd apps/go-api` (le shell perd son CWD!) puis
  `CGO_ENABLED=1 PATH=/c/msys64/ucrt64/bin:$PATH go test -tags cgo ./internal/sync/ ./internal/persist/ -count=1`
  + `go vet`. Relire le EXIT/tail AVANT de committer (cf. leçon test rouge 67742cd14).

## ÉTAT COMMITS (branche fix/sync-combat-completion-persist)
- `7c584c158` honnêteté compteurs heal (failed≠no_film) + tests.
- `1672f34b2` fail-fast shared-RO (assertSharedWritable) + tests.
- `67742cd14` no-film match récent → pas de marquage events_loaded (anti-perte film retardé) + tests.
- `7e9ce46dc` fix test golden no-film (start_time ancien) — RÉPARE une régression que `67742cd14` avait
  commitée ROUGE (CWD shell perdu pendant la vérif → test non relancé avant commit). LEÇON : toujours
  `cd apps/go-api` explicite avant `go test` (le shell perd son CWD après corruption), et relire le
  résultat AVANT de committer.
- Suite `./internal/sync/` verte en `-tags cgo` ET `-tags "cgo integration"` (14-15s).
- Working tree restant = uniquement fichiers utilisateur (PLAN_explorer untracked, PLAN_MEDIA_HLS rename).

## ✅ INCRÉMENT 3 — pas de marquage prématuré events_loaded (commits 67742cd14 + 7e9ce46dc)
- `engine_highlight_events.go` : sur film 404 (`!found`), `ProcessHighlightEvents` ne marque
  `events_loaded=TRUE` que si `isNoFilmDefinitive` → match plus vieux que `filmRetryWindow` (48h, const)
  OU `start_time` NULL (legacy). Match récent + 404 → reste FALSE → réessayé (anti-perte d'un film retardé).
- Tests `film_retry_policy_test.go` (3) : récent→non marqué / ancien→marqué / NULL→marqué. + golden &
  bitmask no-film restent verts (insèrent start_time NULL → marqués). Suite ./internal/sync/ verte
  (-tags cgo ET -tags "cgo integration"), go vet clean. **NON commité.**
- Fichiers : `engine_highlight_events.go` (+const +helper +branche), `film_retry_policy_test.go` (nouveau),
  thought_log.md (point 4), ce HANDOFF.

## ✅ INCRÉMENT 2 — fail-fast shared-RO — commit `1672f34b2`
- **Nouveau** `internal/sync/shared_rw_guard.go` : `assertSharedWritable(ctx, db)` →
  `SELECT bool_or(readonly) FROM duckdb_databases() WHERE database_name NOT IN ('system','temp')`.
  Robuste au basename (ne filtre PAS sur "shared_matches_v2"). Retourne `ErrSharedReadOnly` si RO,
  nil si writable OU si le probe échoue (best-effort, ne bloque pas les tests minimaux).
- **Câblé** en tête de `healEventsForRecentMatches` ET `healWeaponKillsForRecentMatches` (après le
  `len(matchIDs)==0` check, avant la boucle d'écriture) : shared RO → `slog.ErrorContext` + retour
  `fmt.Errorf("healX: %w", ErrSharedReadOnly)`. Plus de N écritures vouées à l'échec.
- **Tests** `internal/sync/shared_rw_guard_test.go` (4) : RW ok, RO détecté, nil ok, heal RO → fail-fast.
  Probe runtime confirmé : handle `?access_mode=read_only` → `duckdb_databases().readonly=true`.
- **Vérifs** : `go vet ./internal/sync/` clean ; `go test -tags cgo ./internal/sync/` vert (14s, 6 tests
  ajoutés au total verts). **PAS commité — demander autorisation.**
- Fichiers modifiés depuis commit `7c584c158` : `events_heal.go` (+wiring fail-fast), nouveaux
  `shared_rw_guard.go` + `shared_rw_guard_test.go`, `thought_log.md` (point 3), ce HANDOFF.
- **NOTE périmètre** : ceci est le volet "fail-fast" de Phase 1. Le volet "router la complétion via
  persist sur le writer provider" (Option A) reste à faire — c'est le gros refactor multi-fichiers.
  Le fail-fast garantit déjà "jamais silencieux" ; le routage persist garantira "toujours sur writer RW".

## ✅ COMMIT FAIT — commit `7d8a09a39` (branche fix/sync-combat-completion-persist)
- Contenu : events_heal.go (honnêteté failed≠no_film) + events_heal_honesty_test.go + thought_log.md
  + HANDOFF + AUDIT (force-add car gitignore `*_RO_*.md` l'excluait). 5 fichiers, 532 insertions.
  Pre-commit hooks (gofmt/vet/merge-check) verts. **Non poussé.**
- GOTCHA résolu : un 1er commit (8bf0d2a62, abandonné) avait par erreur aspiré
  `.ai/PLAN_explorer_combat_profile_charts.md` (fichier STAGÉ par l'utilisateur). Corrigé via
  `git reset --mixed 29fb9637b` + restage explicite de mes 5 fichiers + recommit → `7d8a09a39`.
  EFFET DE BORD à signaler : le reset --mixed a DÉSTAGÉ `PLAN_explorer_combat_profile_charts.md`
  (contenu intact sur disque, redevenu untracked) — l'utilisateur doit `git add` pour le re-stager.
- L'utilisateur travaille en parallèle dans le repo (V7/ untracked, PLAN_explorer + rename
  PLAN_MEDIA_HLS lui appartiennent) : NE committer QUE mes fichiers explicitement, jamais `git commit -a`.
- ⚠️ `.git/config.lock` intermittent pendant la session (process concurrent) → git échouait par
  moments ; retry suffit, ne pas supprimer le lock à la légère.
- **PROCHAIN PAS (reprendre ici)** : item plan #5 (test Go déterministe mécanisme RO) PUIS #6 (Phase 1 :
  router events/killer_victim/skill via persist sur writer provider + fail-fast si RO, Option A direct).
  Phase 1 = gros refactor multi-fichiers du cœur sync → démarrer sur contexte frais, vertical slice
  (events d'abord) + test, pas les 3 writers d'un coup.

## ✅ LIVRÉ session 2026-05-31 (Phase 2 partielle)
- **Honnêteté des compteurs heal** faite dans `internal/sync/events_heal.go` :
  - `healEventsForRecentMatches` : nouveau compteur `failed` ; un `procErr != nil` (write RO / réseau /
    parse) est compté `failed`, PLUS jamais `no_film`. WARN agrégé si `failed>0`.
  - `healWeaponKillsForRecentMatches` : idem (avant, l'échec disparaissait de tous les compteurs).
  - Signatures INCHANGÉES (`failed` reste local + WARN) → aucun caller modifié. Volontaire pour limiter
    le diff ; si on veut propager `failed` jusqu'au SyncResult/expvar, ce sera un incrément séparé.
- **Test garde-fou** : `internal/sync/events_heal_honesty_test.go` (2 tests, cgo) — verts.
  `TestHealEvents_FetchError_NotCountedAsNoFilm` (échoue AVANT le fix) + contrôle genuine-no-film.
- **Vérifs** : `go test -tags cgo ./internal/sync/ -count=1` = **ok (14.5s, vert)** ; 2 tests honnêteté = PASS ;
  `go vet ./internal/sync/` = clean.
  ⚠️ GOTCHA : un panic `duplicate fixup for FuncInfo` peut apparaître sur un run = **stale testcache**
  (pas un vrai bug) ; `go clean -testcache` puis re-run → vert.
- **Branche** : `fix/sync-combat-completion-persist`. Working tree : events_heal.go modifié,
  events_heal_honesty_test.go ajouté, thought_log.md + HANDOFF + AUDIT. writes.go = propre (instru retirée).
  **Pas encore commité** (demander avant commit).
- **PAS encore fait (reste du plan)** : fail-fast/RW-check réel dans la complétion (Phase 1 — le gros :
  router events/killer_victim/skill via persist sur writer provider), garde-fou "zéro db.Exec shared hors
  persist", pas-de-marquage-prématuré events_loaded, bitmasks batch path, citations recompute forcé,
  metadata names, statut front, backfill remédiation, rebuild player DBs.

## 🟥 CORRECTION + ÉTAT RÉEL (2026-05-31 ~20:35)

**CORRECTION d'une erreur d'analyse précédente** : le `SharedDBProvider` (B-swap) **EST** câblé en prod.
- `auto_sync.go:219 BuildEngine` → `engine.WithSharedProvider(s.cfg.SharedProvider)` si non-nil.
  Idem `sync_handler.go:110`, `backfill.go:113`, `sync_v2_wiring.go:226`.
- `main.go:348` `useSharedProvider := os.Getenv("LEVELUP_USE_SHARED_PROVIDER") != "0"` (DEFAULT ON),
  `main.go:391 cfg.SharedProvider = provider`. `.env.local` ne le désactive pas.
- DONC `e.sharedProvider != nil` en prod, et `acquireSharedWriter` passe par `provider.AcquireWriter`
  → `swapToRW` → `OpenReadWrite(path)` = shared en MAIN **RW**. (Mon précédent point "jamais câblé"
  était FAUX — ne pas s'y fier.)

**Donc pourquoi `shared_matches_v2(ro=true)` à l'INSERT de complétion ?** → PAS encore tranché à 100%.
Hypothèses restantes (à départager proprement, idéalement par un test Go déterministe et NON via le
serveur live qui a été pollué par mes kill/relaunch successifs — plusieurs instances ont pu se
chevaucher pendant le churn, créant un faux positif RO) :
  (H1) FATAL "database invalidated" (suite corruption ART) laisse le wrapper `*sql.DB` "rw:"+path du
       cache (`platform/duckdb/db.go openCachedDB`) dans un état dégradé/RO jusqu'au restart. La cascade
       FATAL observée (achievements/citations/seedPlaylists) vient de là.
  (H2) Collision de cache : `OpenReadWrite` ET `OpenReadWriteShared` partagent la clé `"rw:"+path` ;
       `OpenReadOnly` utilise `"ro:"+path` (conn séparée). Si le provider, après un swap raté, retombe
       sur un handle issu d'un autre chemin d'ouverture, la conn peut être RO.
  (H3) Multi-instance transitoire pendant mes manips (le plus probable pour les logs 20:09).

**Ce qui EST certain et actionnable (indépendant du mécanisme exact) :**
1. La complétion (events/killer_victim via `events_heal.go`→`InsertHighlightEvents`/`InsertKillerVictimPairsFromEvents`,
   skill via `healSkill`→`InsertParticipants`) écrit en **`db.Exec` direct hors persist** sur le
   `sharedDB` reçu, **sans vérifier qu'il est RW** et en **avalant** l'échec (compté `no_film`). C'est
   la faille qui a transformé un incident transitoire en **31h de panne silencieuse**.
2. Le fix demandé (RÈGLE ABSOLUE) reste correct quel que soit H1/H2/H3 : toute écriture shared via
   persist sur le writer provider + **fail-fast/alerte si RO** (jamais silencieux) + métriques honnêtes.

**DÉCISION UTILISATEUR (prise)** : **Option A — câbler le provider partout + router la complétion par le
writer provider unique**, en **DIRECT sans flag**. (NB : le provider étant déjà câblé dans les engines,
"câbler partout" = (a) s'assurer que la COMPLÉTION écrit via le writer provider/persist et non en
db.Exec direct ; (b) ajouter une assertion RW + fail-fast ; (c) garde-fou test "zéro db.Exec shared hors
persist" ; (d) métriques honnêtes failed≠no_film.)

**⚠️ AVANT de coder Phase 1** : trancher H1 vs H2 par un **test Go déterministe** (pas le serveur live)
qui reproduit : provider AcquireWriter après une invalidation FATAL simulée → l'INSERT part-il en RO ?
Ce test devient aussi le garde-fou de non-régression. C'est le point de départ propre de la session
suivante (le serveur live n'est pas une source fiable ici).

## 🎯 DIAGNOSTIC COMPLET — CHAÎNE CAUSALE CONFIRMÉE (2026-05-31 ~20:15)

**Chaîne causale (prouvée, pas hypothèse) :**
1. **RC-E** : corruption ART sur `shared.match_participants` (détectée `art_guard` au boot).
2. → FATAL `database has been invalidated` en cascade (metadata: achievements, citations, seedPlaylists).
3. → le `SharedDBProvider` sert un **handle RO** au post-sync (prouvé par instrumentation
   `duckdb_databases()` : `shared_matches_v2(ro=true)`).
4. → **RC-A** : la complétion combat (`healEvents`→`InsertHighlightEvents`, `healSkill`→`InsertParticipants`)
   échoue `read-only mode` à CHAQUE cycle. Métrique ment (`no_film` au lieu de `failed`).
5. → `highlight_events`/`killer_victim_pairs` jamais écrits pour les matchs de la fenêtre du bug →
   **graphiques onglet Détails vides + citations vides** (citations dépendent de highlight_events).

**Après rebuild ART (`force_rebuild_art --db ...`, 26068 rows, 0 perte) + restart propre :**
- ✅ **0** `read-only mode`, **0** `invalidated`, **0** `art_guard corruption` dans le log frais.
- ✅ `weapon self-heal healed=4` → **le chemin d'écriture shared RW fonctionne de nouveau**.
- ❌ Les **2 matchs du 2026-05-30** (`3e9967f6`, `4cb4a8d0`) restent `he=0 wk=0 kv=0 bf=0
  events_loaded=false` ; `events self-heal healed=0 no_film=2`. Tous les matchs ≤ 2026-05-26 sont
  COMPLETS (he=200-340, bf=2686976). Gap : rien entre 2026-05-26 23:03 et 2026-05-30 13:22.
- **Interprétation** : `GetHighlightEventsChunk` renvoie maintenant `found=false` pour ces 2 matchs
  (avant rebuild il renvoyait found=true → on atteignait l'INSERT qui échouait RO). Le chunk highlight a
  vraisemblablement **EXPIRÉ** pendant les ~31h où le bug RO a empêché la capture. → **data-loss
  probable pour ces 2 matchs** (à confirmer par un probe token : `cmd/diag_highlight_match <id>` qui lit
  l'API sans écrire, ou tenter backfill force_events). C'est la conséquence directe du bug : il faut le
  fix structurel pour qu'aucun futur match ne subisse ça.

**CE QUI RESTE (le vrai livrable, indépendant de la récupérabilité des 2 matchs) :**
- Phase 1 : migrer TOUTE écriture shared de complétion vers persist sur writer RW **vérifié** +
  **fail-fast si RO** (au lieu d'échouer silencieusement 31h). Règle absolue : zéro db.Exec shared hors persist.
- Garde-fou test anti-régression (db.Exec shared hors persist interdit).
- Phase 2 : métriques honnêtes (`completed`/`film_absent`/`failed`) + pas de marquage prématuré
  `events_loaded` + ALERTE si `failed>0` (aurait rendu ce bug visible en 1 cycle au lieu de 31h).
- Rebuild ART des **player DBs** pas encore fait (`--player-db <path>` un par un ; `--all` cassé).
- Boucle ouverte : POURQUOI l'ART s'est corrompu ? (probablement écriture concurrente / 2e instance
  historique — cf. les multiples server.exe orphelins trouvés). Le fix persist INSERT-only réduit ce risque.

## ✅ ART REBUILD FAIT (2026-05-31 20:07) — lire en premier
- `force_rebuild_art` sur shared = **succès** : `rows_before=rows_after=26068, ART corruption defeated`,
  0 perte. Commande qui marche (depuis `apps/go-api`, serveur arrêté) :
  `go run -tags cgo ./cmd/force_rebuild_art --db ../../data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
  ⚠️ `--all true` est CASSÉ (résout `data/...` relatif au cwd apps/go-api → "DB introuvable"). Le flag
  helper custom (`flag()`) prend la valeur après `--db`. Pour les player DBs : `--player-db <path>` un par un.
  Player DBs PAS encore rebuild (séparé ; shared était le critique pour combat).
- Serveur relancé propre (Air, 1 instance, port 8000) après rebuild. Sync post-rebuild déclenché
  (`POST /_diag/auto-sync/run` Origin localhost:5173) — monitor armé sur RO/ART/invalidated/self-heal.
- **À vérifier au prochain event** : est-ce que `read-only mode` + `database has been invalidated`
  ont DISPARU après rebuild ? Si OUI → la corruption ART invalidait shared → provider servait un handle RO
  (RC-A causé par RC-E). Si NON (RO persiste sans corruption) → RC-A est structurel indépendant et le fix
  Phase 1 (persist + writer RW + fail-fast) reste requis tel quel. Dans les DEUX cas Phase 1 est à faire
  (honnêteté architecturale : la complétion ne doit jamais écrire hors persist/RW).

## ✅ PHASE 0 TRANCHÉE (2026-05-31 ~20:00) — RÉSULTAT DÉFINITIF, lire en premier

**Preuve runtime captée** (logs/sync.log 19:58:07, instrumentation `duckdb_databases()`) :
```
INSTRUMENT InsertHighlightEvents match_id=4cb4a8d0...
  state="shared_matches_v2(ro=true,duckdb) system(ro=false,duckdb) temp(ro=false,duckdb)"
```
→ La connexion utilisée par la complétion post-sync a **`shared_matches_v2` ouverte READ-ONLY comme base
PRINCIPALE** (ce n'est PAS un ATTACH RO — c'est la main DB elle-même en RO ; `system`+`temp` sont les DB
internes DuckDB toujours présentes). Le même symptôme touche `healSkill`→`InsertParticipants` (RO) dans
le même pipeline. Donc le `sharedDB` passé à `runConditionalPostSync`/`healEventsForRecentMatches` est un
handle RO, alors qu'il est censé provenir du writer RW (`acquireSharedWriter`→provider.AcquireWriter→
swapToRW→OpenReadWrite). **Conclusion : la complétion écrit en `db.Exec` direct sur un handle non-RW —
exactement la divergence orchestrateur vs legacy. Le fix Phase 1 (tout via persist sur writer RW vérifié
+ fail-fast si RO) est confirmé comme correct.**

**2e FACTEUR MAJEUR découvert (à traiter, RC-E)** — mêmes logs, cascade :
- `art_guard: corruption ART détectée db=shared table=match_participants` (count_indexed mismatch, au boot).
- `FATAL Error: database has been invalidated because of a previous fatal error. The database must be
  reopened` sur metadata → casse **achievements**, **citations** (`BackfillMatchCitations: mappings`),
  `seedPlaylistsCatalog`. Cf. mémoire `reference_metadata_fatal_invalidated_multi_server` +
  ADR 0019 (anti-corruption ART, INSERT-only). 
- Hypothèse de liaison RC-A↔RC-E : une invalidation FATAL de shared peut laisser le provider en état
  dégradé servant un handle RO (au lieu d'échouer franchement). À vérifier en Phase 1, mais le fix
  d'écriture-via-persist + fail-fast RO rend le système honnête quoi qu'il arrive. La corruption ART
  elle-même doit être remédiée (rebuild ART : `cmd/force_rebuild_art --all true` ; vérifier 1 seule
  instance — confirmé 1 seule maintenant).
- NB observé aussi : `engagement coefs: save failed ... conflict target not referenced by UNIQUE/PK`
  (player DB) — bug annexe (ON CONFLICT sur table sans contrainte), cf. mémoire
  `reference_legacy_player_db_no_constraints`. Hors scope combat mais à noter.

**Instrumentation RETIRÉE** de writes.go (bloc + imports temp) — rebuild sync OK. État propre pour Phase 1.

## ⚡ DERNIER POINT (session 2026-05-31, restart contrôlé)
- **Env live était wedgé** : un Air pré-existant (PID 15976) lançait server.exe (PID 27276, build
  19:28:00) depuis le MAIN tree (`apps/go-api/tmp/server.exe`). Air ne rebuildait PAS sur mes edits
  (touch mtime-only ignoré par fsnotify Windows). Le scheduler auto-sync ne tournait pas
  (`/snapshot` → last_cycle_at=0001, players=[]) — DBs verrouillées au boot (`spartan_cron` ERROR).
  `/health` renvoie `uptime:"33s"` FIXE (champ non fiable, ne pas s'y fier).
- **Action faite** : kill de TOUS air.exe + server.exe (confirmé 0/0), `rm tmp/server.exe`, puis **j'ai
  relancé Air moi-même** depuis `apps/go-api` :
  `CGO_ENABLED=1 PATH=/c/msys64/ucrt64/bin:$PATH /c/Users/Guillaume/go/bin/air > /c/Users/Guillaume/air2.log`
  (tâche bg bm4rslqu5). Health-poll bg bdrf45oef → notifiera "SERVER-READY".
- **Triggers /run testés** : `POST /api/v1/_diag/auto-sync/run` avec `Origin: http://localhost:5173`
  (origins autorisés par défaut : localhost/127.0.0.1 :5173 et :5174, cf. config.go parseCORSOrigins).
  → renvoyait exit 52 (empty reply) AVANT le restart propre — à re-tester sur le serveur frais. Sans
  Origin = csrf_rejected. `/players/{slug}/sync` = auth_required (tokens dans la requête).
- **PROCHAINES ÉTAPES une fois SERVER-READY** :
  1. Confirmer 1 seule instance + build frais (server.exe mtime récent).
  2. `POST /_diag/auto-sync/run` (Origin localhost:5173). Si empty-reply persiste → le handler crashe
     peut-être ; vérifier `logs/server.crash.log` + air2.log.
  3. Sinon attendre/laisser le cycle, puis `Select-String logs/sync.log INSTRUMENT InsertHighlightEvents`
     → lire le champ `state` = `db(ro=bool,type)` pour CHAQUE base attachée à la conn de l'INSERT.
  4. **RETIRER l'instrumentation** (bloc dans writes.go InsertHighlightEvents + imports temp
     `log/slog`, `strings`) avant tout commit.
- **PLAN B si l'env reste hostile** : écrire directement le test Go déterministe (Phase 8) qui monte un
  vrai `SharedDBProvider`, `AcquireWriter`, et tente un INSERT highlight_events via le chemin post-sync
  → prouve/épingle le RO sans dépendre du serveur live. C'est de toute façon le garde-fou de régression.
- **Conclusion wiring déjà établie (lecture code)** : `healEventsForRecentMatches(ctx, sharedDB,...)`
  (engine_postsync.go:210) reçoit LE MÊME `sharedDB` que `acquireSharedWriter` a retourné
  (engine.go:526/535) → `ProcessHighlightEvents` → `InsertHighlightEvents(ctx, sharedDB,...)`. Donc si
  RO, c'est CE handle (censé RW via provider.AcquireWriter→swapToRW→OpenReadWrite) qui est RO au moment
  de l'INSERT. L'instrumentation tranche : handle lui-même RO, ou base `shared` attachée RO en plus.

## ⚡ DERNIER POINT (session 2026-05-31, suite)
- **Serveur redémarré** avec instrumentation (Air relancé manuellement :
  `cd apps/go-api && CGO_ENABLED=1 PATH=/c/msys64/ucrt64/bin:$PATH /c/Users/Guillaume/go/bin/air`,
  log de boot dans `C:\Users\Guillaume\air_run.log`). 1 seule instance server.exe confirmée (pas de
  multi-instance). Health OK :8000.
- **Au boot, 2 signaux notables** : `art_guard: corruption ART détectée db=shared table=match_participants`
  (divergences_count=1) ET `spartan_cron: player DB verrouillée par un autre process`. À investiguer mais
  pas le cœur du bug combat.
- **Trigger sync** : `POST http://127.0.0.1:8000/api/v1/_diag/auto-sync/run` (loopback-only, sans auth,
  utilise le pool de tokens serveur — c'est LE moyen de déclencher un sync complet à la demande). Les
  routes `/players/{slug}/sync` et `/admin/sync` exigent des tokens dans la requête (auth_required) ou
  rejettent l'origin (csrf). NE PAS réutiliser celles-là.
- **Fait confirmé** : `WriterHandle.DB()` (provider) retourne bien le `*sql.DB` RW issu de
  `swapToRW`→`OpenReadWrite(path)` = shared en MAIN read-write. Donc le `sharedDB` post-sync passé à
  `runConditionalPostSync`→`healEventsForRecentMatches` DEVRAIT être RW. La contradiction (events-heal
  voit shared "attached read-only") sera tranchée par l'instrumentation `duckdb_databases()`.
- **Cache DuckDB** (`platform/duckdb/db.go`) : clés distinctes `ro:`+path et `rw:`+path → RO et RW sont
  des pools sql.DB séparés. `OpenReadWrite` = maxConns 1. Auto-attach DuckDB possible.
- **EN ATTENTE** : event Monitor sur `INSTRUMENT InsertHighlightEvents` (sync déclenché). Lire le champ
  `state` = liste `database_name(ro=bool,type)`. Puis **RETIRER l'instrumentation** (bloc + imports
  `log/slog`/`strings` dans `writes.go`) avant tout commit.

## ⚡ DERNIER POINT (fin de session 2026-05-31) — lire en premier
- **Instrumentation APPLIQUÉE et COMPILE** (`go build ./internal/sync/` = OK) dans
  `apps/go-api/internal/sync/writes.go` : bloc `INSTRUMENT InsertHighlightEvents` + imports temporaires
  `log/slog` et `strings` (marqués « TEMP INSTRUMENT Phase 0 — retirer »). **À RETIRER après diagnostic.**
- **MAIS pas encore déployée** : `server.exe` (PID 6696) tourne depuis **hier** (2026-05-30). `air.exe`
  (PID 2336) tourne mais le serveur n'a PAS redémarré après mon edit → binaire en cours = ancien.
  → Pour capter l'instrumentation : soit forcer le rebuild/restart (Air devrait le faire ; sinon
  `taskkill` server.exe pour qu'Air relance), soit attendre/déclencher un cycle après restart, puis
  `POST /admin/sync/{gamertag}` (route `routes.go:106`) et lire `logs/sync.log`.
- **NOUVELLE HYPOTHÈSE FORTE (à vérifier en priorité)** : possible **plusieurs instances server.exe**
  simultanées (vérif `Get-CimInstance Win32_Process -Filter "name='server.exe'"` non concluante car
  sorties d'outils corrompues en fin de session). Cf. mémoire
  `reference_metadata_fatal_invalidated_multi_server` : des instances multiples ouvrent les DB en RO →
  expliquerait DIRECTEMENT « attached in read-only mode » (la 2e instance / le pool ouvre `shared` en RO
  pendant que le sync veut écrire). **Si c'est ça, la cause racine est opérationnelle (process en double)
  ET architecturale (écritures de complétion pas sur le writer RW).** Vérifier le nombre d'instances
  AVANT le refactor : `Get-CimInstance Win32_Process -Filter "name='server.exe'" | Select ProcessId,CreationDate`.
- **Décision en attente utilisateur** : autoriser le redémarrage du serveur pour déployer
  l'instrumentation (il l'utilise activement).
- Outils de session instables : sorties parfois corrompues → relancer en isolant chaque commande,
  écrire dans un fichier temp + `Read`, ou PowerShell `Set-Content` puis `Read`.

## État actuel (2026-05-31)
- **Branche de travail** : `fix/sync-combat-completion-persist` (créée depuis
  `fix/metadata-art-catalog-upsert-invalidation`). ⚠️ Des changements non-commités metadata-art sont
  présents dans le working tree (M sur `.ai/thought_log.md`, `cmd/diag_lusr_displayed/main.go`,
  `apps/web/src/features/session-detail/*`). **NE PAS les committer** — ne stager que mes fichiers.
- **Pas de commit fait. Demander avant tout commit.** Pas de Python. Pas de git stash.
- Serveur en cours (Air) PID ~6696 ; logs : `logs/sync.log` (racine, le plus récent) ET
  `apps/go-api/logs/sync.log`. Air recompile à chaque modif `.go`.
- Étape en cours : **Phase 0 — Option 1 = micro-instrumentation runtime** pour épingler à 100% la couche
  qui rend `shared_matches_v2` read-only au moment de l'INSERT events.

## Symptôme
Matchs d'hier (2026-05-30) onglet Détails : tableaux OK, **graphiques de combat vides + citations
absentes**. Matchs concernés confirmés : `3e9967f6-9cc7-452d-9f55-8db0b2351484`,
`4cb4a8d0-962b-436e-a6fc-98c1a5e36ca9`. DB : `highlight_events=0`, `killer_victim_pairs=0`,
`weapon_kills=0`, `events_loaded=false`, `backfill_completed=0`, `map_name`=GUID.
Outil de vérif : `cd apps/go-api && go run ./cmd/diag_recent_match_sync --summary 25`
(et `--recent N` pour le détail). CGO requis : `CGO_ENABLED=1 PATH=/c/msys64/ucrt64/bin:$PATH`.

## Cause racine (confirmée logs + code)
Deux régimes d'écriture shared :
- **PRIMAIRE (orchestré, RW-safe)** : `collect.go` → `persist.BatchBuilder` → `SharedPersister` via writer
  RW du `SharedDBProvider`. Écrit highlight_events/killer_victim quand film dispo au sync. **Marche.**
- **COMPLÉTION POST-SYNC (legacy, cassée)** : `events_heal.go` → `ProcessHighlightEvents` →
  `InsertHighlightEvents` (`writes.go:406`) en `db.Exec` direct. Film téléchargé+parsé OK mais INSERT
  échoue **à chaque cycle** :
  `Cannot execute statement of type "INSERT" on database "shared_matches_v2" which is attached in read-only mode!`
  Idem LUSR v2 `SkillV2Repo.UpsertState`. → combat data jamais persistée → graphiques vides.
- **Citations** : `writeCitations` (`citations.go:411`, écrit player DB `match_citations`) dépend de
  `highlight_events` (vides) → 0 delta + sentinel `_processed` → ne recalcule jamais seul →
  **recompute forcé requis** (`force=true`, `citations_backfill.go:331` / `selectMatchesForCitations`).

### Réponse à la question archi de l'utilisateur
« On a un orchestrateur + collect/persist » est vrai pour le PRIMAIRE ; la complétion post-sync le
**contourne** (écritures legacy directes sur un handle shared non garanti RW). → **RÈGLE FERME ET ABSOLUE
du plan** : toute écriture shared (primaire ET complétion/heal/backfill) DOIT passer par collect/persist
sur le writer RW provider. Zéro `db.Exec` shared hors package `persist`. Rien laissé de côté + garde-fou.

## Énigme à trancher par instrumentation (Phase 0, EN COURS)
Dans le MÊME run JGtm : weapon-heal réussit (`healed:4`) mais events-heal échoue RO — les deux reçoivent
le même `sharedDB`. Le post-sync ré-acquiert pourtant un writer RW provider après le drain async
(`engine.go:526` `postSDB, postRls, _ := e.acquireSharedWriter(ctx)`; `sharedDB = postSDB`).
`OpenReadWrite` ouvre shared en MAIN read-write (maxConns=1, `platform/duckdb/db.go:157-167`). Donc
théoriquement RW. À CONFIRMER : pourquoi l'INSERT events voit shared **attaché RO** (auto-attach DuckDB
côté connexion player ? handle différent ? swap RO concurrent ?). Le fix ne dépend pas du verdict mais on
veut réparer la bonne couche.

### Instrumentation à appliquer (Option 1)
Dans `apps/go-api/internal/sync/writes.go`, fonction `InsertHighlightEvents` (commence ~ligne 406), juste
après le `if len(events) == 0 { return 0, nil }` et AVANT le `db.PrepareContext`, insérer (imports déjà
présents : `fmt`, `log/slog`, `strings`) :
```go
	// TEMP INSTRUMENT (Phase 0 fix/sync-combat-completion-persist) — À RETIRER après diagnostic.
	if dr, derr := db.QueryContext(ctx, `SELECT database_name, readonly, type FROM duckdb_databases()`); derr == nil {
		var st []string
		for dr.Next() {
			var name, typ string
			var ro bool
			_ = dr.Scan(&name, &ro, &typ)
			st = append(st, fmt.Sprintf("%s(ro=%v,%s)", name, ro, typ))
		}
		dr.Close()
		slog.WarnContext(ctx, "INSTRUMENT InsertHighlightEvents conn databases",
			"match_id", matchID, "state", strings.Join(st, " "))
	} else {
		slog.WarnContext(ctx, "INSTRUMENT InsertHighlightEvents duckdb_databases() failed",
			"match_id", matchID, "err", derr)
	}
```
Optionnel : même bloc dans `BackfillWeaponKillsForMatch` (`backfill_weapons.go:34`) juste avant
`InsertWeaponKills` (~ligne 108) pour comparer l'état RO/RW weapon vs events dans le même run.

### Procédure d'observation
1. Appliquer l'instrumentation (Air recompile tout seul) OU build manuel.
2. Attendre un cycle auto-sync (~15 min) OU forcer un sync.
3. `grep "INSTRUMENT InsertHighlightEvents" logs/sync.log | tail` (gérer encodage : via PowerShell
   `Select-String` si grep bash garble).
4. Lire `state` : si `shared_matches_v2(ro=true,...)` → le handle est bien RO (confirme la couche).
   Comparer avec le bloc weapon si ajouté.
5. **RETIRER l'instrumentation** avant tout commit.

## Design Phase 1 (confirmé en lecture code)
- `SharedPersister.Persist` est **no-op si `batch.Shared.Match == nil`** (shared_persister.go:67) → on ne
  peut PAS le réutiliser pour la complétion events-only (le match_registry existe déjà). Il faut un
  **persister de complétion dédié** (ex. `EventsCompletionPersister`) dans le package `persist`, qui
  réutilise les helpers existants `persistHighlightEvents` + `persistKillerVictim` (déjà présents,
  shared_persister.go:293 & :273) en TX, sur le writer RW. Pattern de référence :
  `comeback_postsync_persist.go` → `persist.NewPostSyncEnrichmentPersister(playerDB).BatchUpdateColumn`.
- Côté sync, `events_heal.go`/`ProcessHighlightEvents` doivent appeler ce persister (writer RW) au lieu de
  `InsertHighlightEvents`/`InsertKillerVictimPairsFromEvents` directs. Idem `MarkEventsLoaded` (UPDATE
  match_registry) → passer par un persister/TX RW.
- Le writer RW est déjà tenu par le post-sync (`engine.go:526 acquireSharedWriter`) ; NE PAS ré-acquérir
  (dblease non-réentrant). Passer le `sharedDB` RW (ou un persister construit dessus) aux fonctions de
  complétion.

## Fix prévu (Phases, voir plan complet)
- **P1** : migrer events/killer_victim/LUSR v2 completion vers `persist` sur le writer RW effectivement
  tenu par le post-sync (NE PAS ré-acquérir : dblease non-réentrant, cf. `engine_acquire.go` godoc).
  Supprimer les `db.Exec` shared directs de la complétion.
- **P1bis** : garde-fou test (modèle `no_art_patterns_test.go`) interdisant `db.Exec`/`Prepare` shared
  hors `persist` (allowlist). Inventaire complet dans `.ai/AUDIT_SYNC_COMBAT_RO_2026-05-31.md`.
- **P2** : métriques honnêtes heal (`completed`/`film_absent`/`failed`, plus de `no_film` menteur,
  `events_heal.go:120-128`) + pas de marquage prématuré `events_loaded=TRUE` sur 404 transitoire
  (`engine_highlight_events.go:125-135`, garder TRUE seulement si film expiré via start_time).
- **P3** : bitmasks batch path à parité legacy (`engine_batch_path.go` vs `engine_fetch.go:179-192`) +
  log `fm.SkillError`.
- **P4** : citations recompute forcé après backfill events.
- **P5** : résolution noms metadata registry (réutiliser `cmd/backfill_registry_names` / service assets ;
  mémoire `reference_asset_translations_fr`).
- **P6** : statut sync visible (compteurs + bandeau front `is_partial`/`events_empty`,
  `MatchViewPage.tsx:168-201`).
- **P7** : remédiation = backfill events/kv des matchs affectés (writer RW) + recompute citations.
- **P8** : tests E2E complétude (échoue si write shared RO) + unitaires. `go test ./... -race && go vet`
  (CGO+msys64). Front `npm run typecheck && lint && test` (vitest hors sandbox,
  `reference_vitest_outside_sandbox`).

## Fichiers clés
- `internal/sync/events_heal.go` (cœur RC), `writes.go` (Insert*), `engine_highlight_events.go`
- `internal/sync/engine.go` (~478-560 drain+post-sync), `engine_acquire.go`, `engine_batch_path.go`
- `internal/platform/duckdb/sharedprovider/provider_writer.go`, `platform/duckdb/db.go`
- `internal/sync/citations.go`, `citations_backfill.go`, `skill_v2_shadow.go`, `backfill_weapons.go`
- `internal/persist/{shared_persister.go,batch.go,rows.go}` (cible orchestrée)
- Front : `apps/web/src/features/match-view/MatchViewPage.tsx`
- Diag : `cmd/diag_recent_match_sync`

## Rappels process
- Répondre en FR. Pas d'emojis dans les fichiers versionnés. Thought_log obligatoire avant de rendre la
  main / commit. Skills à invoquer avant commit : delivery-checklist, plan-review.
- Outils de cette session instables (sorties parfois corrompues) → privilégier lectures ciblées, fichiers
  temp + Read, PowerShell Select-String pour les logs.
