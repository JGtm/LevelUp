# PLAN — Backfill highlight events propre + audit bitmasks menteurs

> **Date d'ouverture** : 2026-05-08
> **Branche cible** : `feat/token-pool-parallel-sync` (continuité des commits `64f6720b`, `34c7f646`)
> **Statut** : à valider — plan en attente d'exécution
> **Document de référence croisée** : ce plan est posé pour permettre la confrontation entre diagnostics indépendants (deux agents ont abouti à la même conclusion sur le périmètre).

---

## 1. Contexte

### 1.1 Symptôme initial

L'utilisateur a constaté que les highlight events n'étaient plus récupérés sur les matchs récents de JGtm. Diagnostic via sondes ad-hoc :

- API HTTP fonctionnelle (manifest 200, blob 200, zlib valide)
- Parser Go retournait **0 events** sur des chunks de 196 KB pourtant valides
- Sur le dernier match `b8c1b220-…`, scan byte-aligné trouvait 28 candidats XUID, **0 avec end-marker**. Scan bit-aligné équivalent : 275 candidats, 275 OK, 8 XUIDs distincts (= 4v4 humains).

### 1.2 Cause racine

Le port Go de `spnkr/film/highlight_events.py` scannait byte-par-byte alors que le format film Halo est **packé au bit** (Python `bitstring.findall` matche au bit près). 247 / 275 XUIDs sont à des bit-offsets non-multiples de 8.

### 1.3 Ce qui a été corrigé (commits `64f6720b` + `34c7f646`)

| Item | Commit | État |
|---|---|---|
| Parser bit-aligné (bit-reader hand-rolled, scan + parse) | 64f6720b | Livré |
| Robustesse multi-end-marker (itère vs. premier match aveugle Python) | 64f6720b | Livré |
| Fixture v41 réelle commitée (`testdata/v41_chunk_he.bin`) | 64f6720b | Livré |
| Tests bit-offset 0..7 + tests bit-reader | 64f6720b | Livré |
| Anomalie de parsing remontée en WARN + counter expvar `highlight_events_parse_anomaly_total` + `result.Warnings` | 64f6720b | Livré |
| `cmd/replay_highlight_events` (one-shot tool) | 34c7f646 | Livré (à supprimer en Phase 3 ci-dessous) |
| Fix `InsertKillerVictimPairsFromEvents` (DELETE+INSERT, 7 colonnes au lieu de 4 + dropping de OR IGNORE incompatible avec table sans PK) | 34c7f646 | Livré |
| Migration `steps_shared.go` alignée prod (9 colonnes, pas de PK) | 34c7f646 | Livré |
| Replay exécuté sur la prod : 27 matchs healed (6303 events), 72 no-film, 0 anomaly, 0 errors | — | Livré |

### 1.4 Ce qui reste à corriger (objet du présent plan)

1. **`/backfill/start` HTTP** : `events` toujours dans `warnUnimplemented` ([`backfill.go:278`](../apps/go-api/internal/api/handlers/backfill.go#L278)). Le front qui passerait par cette API obtient un warning silencieux et zéro effet.
2. **CLI `levelup`** : pas de sous-commande pour relancer un backfill events. `cmd/replay_highlight_events` est un binaire orphelin non listé dans `levelup --help`.
3. **Bitmasks menteurs** : plusieurs `Mark*Loaded` dans le sync sont appelés inconditionnellement après un insert, peu importe son résultat. Le bit prétend "c'est fait" alors que la donnée n'est pas là. Le fix parser n'a réparé que le mensonge sur `events_loaded` au cas particulier "0 events parsés" ; le mensonge structurel reste.
4. **Code à nettoyer** : `cmd/replay_highlight_events/main.go` contient ~280 lignes mélangeant détection SQL, orchestration, auth, glue. Non testable, non réutilisable. À extraire en helper dans `internal/sync/`.

---

## 2. Objectifs et critères de succès

| Critère | Mesure |
|---|---|
| `POST /backfill/start {events:true}` traite réellement les events | Test handler vert + grep `"events"` dans `warnUnimplemented` retourne vide |
| `levelup replay-events --gamertag X` listé dans `--help` et fonctionnel | `levelup --help` mentionne la sous-commande + dry-run sur prod retourne `0 cassés` |
| Bitmasks honnêtes (events / kvp) | Tests unitaires `TestNoLyingBitmasks_*` qui valident qu'un insert raté ne déclenche pas le `Mark*Loaded` |
| Audit lecture seule des autres bits (PBit*, MBit*) | Tableau verdict ajouté à `thought_log.md` |
| Code de replay réutilisable | Helper `internal/sync/events_replay.go` testé en isolation, consommé par handler HTTP + CLI |
| `cmd/replay_highlight_events/` supprimé | git status |
| Aucune régression | `go test ./...` 100% pass + `go vet` clean |

---

## 3. Phases

### Phase 1 — Refactor : extraire le replay du `cmd/` vers `internal/sync/`

**Effort** : ~1 h
**Livrable indépendant** : oui (l'ancien CLI continue de fonctionner via l'helper)

**Fichiers nouveaux** :

- `internal/sync/events_replay.go` (~150 L) :
  - `FindBrokenHighlightEventMatches(ctx, db, limit) ([]string, error)` — détection via SQL `events_loaded=TRUE AND (NOT EXISTS he OR (kills>0 AND NOT EXISTS kvp))`
  - `ReplayResult` struct (Total, Healed, NoFilm, ParseAnomaly, Errors, EventsInserted)
  - `ReplayHighlightEventsForMatches(ctx, client, sharedDB, globalDB, matchIDs, progressFn) (ReplayResult, error)` — boucle clear+process avec callback de progression
  - `clearEventsLoaded(db, matchID) error` — privé

- `internal/sync/events_replay_test.go` (build tag `integration`, ~250 L) :
  - `TestFindBrokenHighlightEventMatches_DetectsBothCases` — table-driven sur 4 cas
  - `TestReplayHighlightEventsForMatches_HealedNoFilmAnomaly` — 3 réponses différentes via mock
  - `TestReplayHighlightEventsForMatches_ProgressCallback` — callback appelé 1× par match
  - `TestReplayHighlightEventsForMatches_ContextCancelled` — annulation propre

**Fichier modifié** :

- `internal/sync/halo_client_mock_test.go` : étendre `mockHaloClient` pour mapper `matchID → (data, version, found, err)` (actuellement : une seule réponse globale)

**Critère de complétion** : `go test -tags integration ./internal/sync/... -run "TestFindBroken|TestReplayHighlight"` vert.

---

### Phase 1bis — Fix des bitmasks menteurs (events + kvp)

**Effort** : ~30-45 min
**Livrable indépendant** : oui
**Pourquoi avant Phase 2** : Phase 2 industrialise un canal qui ment. Inverser livrerait Phase 2 sur un sol assaini.

**Audit ciblé sur ce périmètre** :

| Call-site | État actuel | Verdict |
|---|---|---|
| `engine.go::processHighlightEvents` MarkEventsLoaded | conditionnel `if n > 0` | OK |
| `engine.go::processHighlightEvents` MarkKillerVictimLoaded | conditionnel `if pairsErr == nil` | OK |
| `engine.go::insertHighlightEventsFromData` MarkEventsLoaded | conditionnel `if n > 0` | OK |
| **`engine.go::insertHighlightEventsFromData` MarkKillerVictimLoaded** | **inconditionnel** (`_ = InsertKVP(); _ = MarkKVLoaded()`) | **MENTEUR** — fix 1.bis.a |
| **`events_heal.go:75` MarkEventsLoaded** | **inconditionnel** sur tout retour de `ProcessHighlightEvents`, y compris parse_anomaly | **MENTEUR sur anomaly** — fix 1.bis.b |

**Fixes** :

- **1.bis.a** — `engine.go::insertHighlightEventsFromData` : aligner sur le pattern de `processHighlightEvents` :
  ```go
  pairsErr := InsertKillerVictimPairsFromEvents(sharedDB, matchID, events)
  if pairsErr != nil {
      slog.WarnContext(ctx, "InsertKillerVictimPairs échoué", "match_id", matchID, "err", pairsErr)
  } else {
      _ = MarkKillerVictimLoaded(sharedDB, matchID)
  }
  ```

- **1.bis.b** — `events_heal.go::healEventsForRecentMatches` : distinguer 4 cas via le résultat de `ProcessHighlightEvents` :
  - `success` (events insérés) → mark
  - `no_film` (404 définitif côté CDN) → mark
  - `parse_anomaly` (warnings non vide) → **NE PAS** mark (sera retenté au prochain sync ou via `--force-events`)
  - `network error` (err != nil) → **NE PAS** mark (idem)

**Tests** :

- `TestInsertHighlightEventsFromData_DoesNotMarkKVLoadedOnInsertFailure` — mock DB qui fait échouer `InsertKillerVictimPairsFromEvents`, vérifie que `MBitKillerVictim` n'est PAS positionné
- `TestHealEventsForRecentMatches_DoesNotMarkOnParseAnomaly` — mock client qui retourne un chunk 0-events-parsé, vérifie que `events_loaded` reste FALSE

**Critère de complétion** : tests verts + audit ré-exécuté sur events/kvp donne 5/5 OK.

---

### Phase 1ter — Audit lecture seule des autres bitmasks

**Effort** : ~20-30 min (audit seul, aucun code modifié)
**Livrable indépendant** : oui (ne bloque rien)

**Périmètre d'audit** :

Bits de match (`match_registry.backfill_completed`) :
- `MBitEvents` (1<<16) — déjà audité Phase 1bis
- `MBitAssets` (1<<17)
- `MBitAliases` (1<<18)
- `MBitKillerVictim` (1<<19) — déjà audité Phase 1bis
- `MBitPVEStats` (1<<20)
- `MBitWeaponKills` (1<<21)
- `MBitWeaponKillsNoFilm` (1<<22)

Bits de participants (`match_participants.backfill_bits`) :
- `PBitSkill`, `PBitMedals`, `PBitAccuracy`, `PBitShots`, `PBitDamage`, `PBitAvgLife`, `PBitGrenadeKills`, `PBitMeleeKills`, `PBitPowerWeapon`, `PBitPersonalScore`, `PBitHeadshotKills`, `PBitMaxSpree`, `PBitKDA`, `PBitTimePlayed`

**Méthode** :

1. `grep -rn "Mark.*Loaded\|backfill_completed.*|=\|backfill_bits.*|=" internal/sync/`
2. Pour chaque hit : remonter le call-site et vérifier la condition d'appel
3. Classer en : `OK` (conditionnel sur insert success), `MENTEUR` (inconditionnel), `INTENTIONNEL` (cap-de-retry assumé, à documenter)

**Livrable** : tableau dans `thought_log.md` :

```markdown
| Bit | Mark function | Call-site | Verdict | Action |
|---|---|---|---|---|
| MBitAssets | MarkAssetsLoaded | … | OK / MENTEUR / INTENTIONNEL | (issue à ouvrir) ou rien |
```

**Aucun code modifié dans cette phase.** Si des menteurs sont détectés, ils sont notés comme "à fixer dans un plan dédié" pour ne pas exploser le scope. Le présent plan livre les 2 fixes events+kvp (Phase 1bis) + l'audit (cette phase).

**Critère de complétion** : tableau complet ajouté à `thought_log.md`, tous les bits audités, verdicts justifiés.

---

### Phase 2 — Wire `events` first-class dans `/backfill/start`

**Effort** : ~45 min
**Livrable indépendant** : oui (dépend de Phase 1 et 1bis pour les helpers et l'honnêteté)

**Fichier modifié** : `internal/api/handlers/backfill.go`

- Retirer `"events"` de `warnUnimplemented` (l. 278-279)
- Ajouter "Phase 2.7 highlight events" dans la goroutine du job, après "Phase 2.6 engagement coefficients" :
  ```go
  eventsHealed := 0
  if scope.Events {
      if tokens == nil {
          j.Warnings = append(j.Warnings, "WARN: highlight events ignorés — tokens absents")
      } else {
          // Construire la liste : missing (events_loaded=FALSE) + broken si ForceEvents
          ids := missing
          if scope.ForceEvents {
              broken, _ := go_sync.FindBrokenHighlightEventMatches(ctx, sharedDB, 5000)
              ids = unionMatchIDs(ids, broken)
          }
          client := buildAuthedHaloClient(tokens, rps)
          res, err := go_sync.ReplayHighlightEventsForMatches(
              ctx, client, sharedDB, globalDB, ids, progressFn(jobID))
          if err != nil {
              j.Warnings = append(j.Warnings, fmt.Sprintf("WARN events: %v", err))
          }
          eventsHealed = res.Healed
      }
  }
  ```
- Le résumé final mentionne `events: %d`

**Risque identifié** : le handler n'ouvre pas explicitement la `globalDB` aujourd'hui. À résoudre via `PathResolver` + `duckdbpkg.OpenReadWrite` best-effort (comme dans le tool actuel).

**Tests** : `internal/api/handlers/backfill_test.go`

- `TestBackfillStart_Events_NotInWarnUnimplemented` (régression)
- `TestBackfillStart_Events_HealsBrokenMatches` — handler avec mock client + DuckDB in-memory : `POST /backfill/start {events:true,force_events:true}` → vérifie que la job_status finale a `events_healed > 0`

**Critère de complétion** : tests verts + grep "events" dans `warnUnimplemented` vide + manuel `curl POST /backfill/start` retourne un job qui termine en `succeeded`.

---

### Phase 3 — CLI : sous-commande `levelup replay-events` + suppression du binaire orphelin

**Effort** : ~30 min
**Livrable indépendant** : oui (dépend de Phase 1 pour les helpers)

**Fichiers nouveaux** :

- `cmd/levelup/cmd_replay_events.go` (~80 L) :
  - Flags : `--gamertag`, `--limit`, `--dry-run`, `--rps`
  - Reproduit le pattern d'auth de `cmd_sync.go::runSyncDelta`
  - Appelle `go_sync.FindBrokenHighlightEventMatches` puis `go_sync.ReplayHighlightEventsForMatches`
  - Imprime `ReplayResult` + delta du counter expvar
  - `--dry-run` n'appelle que la détection

**Fichier modifié** : `cmd/levelup/main.go`
- Ajouter `case "replay-events": runReplayEvents(cfg, args)`
- Liste dans `printUsage()`

**Fichiers supprimés** : `cmd/replay_highlight_events/` (code mort, remplacé par la sous-commande `levelup`)

**Tests** : la fonction `runReplayEvents` étant un thin wrapper, les tests Phase 1 couvrent la logique. Pas de test CLI dédié (alignement avec `cmd_sync.go`).

**Critère de complétion** : `go run ./cmd/levelup replay-events --gamertag JGtm --dry-run --limit 10` affiche `0 cassés` (puisque la prod est déjà nettoyée par les commits précédents) + `cmd/replay_highlight_events/` absent du repo.

---

### Phase 4 — Test E2E sur fixture canonique (golden match)

**Effort** : ~2 h
**Livrable indépendant** : oui (dépend de Phase 1 pour `mockHaloClient` étendu, sinon autonome)
**Pourquoi cette phase existe** : les mocks unitaires de Phase 1 vérifient *la logique* avec des données fabriquées à la main. Ils n'auraient PAS capté le bug parser (les fixtures synthétiques étaient byte-alignées) ni le bug `InsertKillerVictimPairs` (la fonction n'était jamais atteinte avec `events=0`). Une fixture E2E rejoue une réponse API authentique de bout en bout : un changement subtil de format ou un mensonge dans le pipeline est détecté immédiatement.

**Périmètre** : un seul match canonique, choisi pour sa diversité (4v4 PvP avec film disponible, kills, deaths, medals, mode events). Capturé une fois, commité, ré-utilisable indéfiniment.

**Capture des fixtures** :

Nouveau outil `cmd/refresh_golden_fixture/main.go` (~120 L) :
- Flags : `--gamertag`, `--match-id`
- Auth via `.env.local` (réutilise le pattern de `cmd/get-token`)
- Appelle l'API et persiste sous `internal/sync/testdata/golden_match/` :
  - `manifest.go` — constantes (matchID, XUIDs attendus, kills attendus, etc.)
  - `stats.json` — réponse `GetMatchStats(matchID)` brute (~50 KB)
  - `skill.json` — réponse `GetMatchSkill(matchID, xuids)` brute (~5 KB)
  - `film_manifest.json` — réponse `/spectate` brute (~6 KB)
  - `film_chunk_he.bin` — chunk ChunkType=3 zlib brut (~200 KB)
  - `film_chunks_replication.tar` — chunks ChunkType=2 packés (REPLICATION_DATA pour weapon_kills, ~5 MB)

L'outil n'est exécuté que ponctuellement (à la première capture, puis si l'API change). Il est ajouté à `.gitignore` côté binaire mais le `main.go` est commité pour traçabilité.

**Fichiers nouveaux** :

- `internal/sync/testdata/golden_match/` — fixtures (cf. ci-dessus)

- `internal/sync/golden_mock_client.go` (test-only, ~80 L) — implémente `HaloClient` en lisant les fichiers `testdata/golden_match/*`. Distinct de `mockHaloClient` : ce dernier reste pour les tests unitaires synthétiques, le golden mock est dédié au scénario E2E.

- `internal/sync/sync_pipeline_e2e_test.go` (build tag `integration`, ~250 L) :

  ```go
  func TestSyncPipeline_GoldenMatch_AllTablesPopulated(t *testing.T) {
      sharedDB := openInMemoryShared(t)  // schéma prod via migration.RunForDB
      globalDB := openInMemoryGlobal(t)
      client := NewGoldenMockClient(t)

      engine := NewSyncEngineForTesting(sharedDB, globalDB, client, golden.MatchID)
      err := engine.SyncSingleMatch(ctx, golden.MatchID)
      require.NoError(t, err)

      // — Comptes ligne par table —
      require.Equal(t, 1, countRows(sharedDB, "match_registry"))
      require.Equal(t, golden.ParticipantsCount, countRows(sharedDB, "match_participants"))
      require.GreaterOrEqual(t, countRows(sharedDB, "highlight_events"), golden.MinHighlightEvents)
      require.GreaterOrEqual(t, countRows(sharedDB, "killer_victim_pairs"), golden.MinKVP)
      require.GreaterOrEqual(t, countRows(sharedDB, "medals_earned"), golden.MinMedals)

      // — Honnêteté des bitmasks (Phase 1bis) —
      var bf int64
      sharedDB.QueryRow(`SELECT backfill_completed FROM match_registry`).Scan(&bf)
      require.NotZero(t, bf & MBitEvents, "MBitEvents doit être set")
      require.NotZero(t, bf & MBitKillerVictim, "MBitKillerVictim doit être set")

      // — Pas de NULL critiques —
      require.Zero(t, countWhere(sharedDB, "match_participants", "gamertag IS NULL OR gamertag = ''"),
          "tous les participants doivent avoir un gamertag")
      require.Zero(t, countWhere(sharedDB, "killer_victim_pairs", "time_ms IS NULL"),
          "killer_victim_pairs.time_ms doit toujours être set (régression InsertKVP fix mai 2026)")

      // — Couverture xuid_aliases —
      humanXUIDs := golden.HumanXUIDs
      coveredAliases := countAliasCoverage(globalDB, humanXUIDs)
      require.Equal(t, len(humanXUIDs), coveredAliases,
          "tous les xuid humains doivent avoir un alias dans xuid_aliases")
  }

  func TestSyncPipeline_GoldenMatch_NoFilm_CascadeRespected(t *testing.T) {
      // Variante : client renvoie found=false sur GetHighlightEventsChunk
      // Vérifie que MBitEvents reste 0 (cascade respectée), kvp reste vide.
  }
  ```

- `internal/sync/sync_helpers_for_testing.go` (test-only, ~50 L) — helpers `openInMemoryShared`, `openInMemoryGlobal`, `countRows`, `countWhere`, `NewSyncEngineForTesting`. Évite la duplication entre tests E2E et tests existants.

**Coordination avec Phase 1 et 1bis** : la fonction `NewSyncEngineForTesting` doit pouvoir injecter un `HaloClient` arbitraire (étendre l'API publique de `SyncEngine` si nécessaire — déjà partiellement présent via `SetCustomClient`).

**Quand le golden test échoue** :
- Si Halo change le format de l'API → re-capturer via `cmd/refresh_golden_fixture` puis ajuster `manifest.go`. Le test devient l'oracle de "le format API d'aujourd'hui est rétro-compatible".
- Si une régression de code → le test indique exactement quelle invariant cassée (count, bitmask, NULL, alias).

**Ce que cette phase aurait capté** sur les bugs récents :
- Parser bit-aligné cassé → `highlight_events < golden.MinHighlightEvents` (FAIL immédiat)
- `InsertKillerVictimPairs` `OR IGNORE` rejeté → `killer_victim_pairs` vide (FAIL)
- Schéma `killer_victim_pairs` désaligné (cols manquantes) → `time_ms IS NULL` (FAIL)
- `MarkKillerVictimLoaded` inconditionnel → couvert par Phase 1bis tests, le golden assure aussi `bf & MBitKillerVictim != 0` cohérent avec les rows réellement insérées

**Critère de complétion** :
- `internal/sync/testdata/golden_match/` peuplé et commité (~5-6 MB compressés)
- `cmd/refresh_golden_fixture` buildable
- `go test -tags integration ./internal/sync/ -run TestSyncPipeline_GoldenMatch` vert
- Documentation dans le test : commentaire en tête expliquant comment re-capturer si l'API change

---

### Phase 5 — Vérification finale + thought_log

**Effort** : ~15 min

| Check | Méthode |
|---|---|
| `go build ./...` | passe |
| `go test ./...` | 100% |
| `go test -tags integration ./internal/sync/` | passe (Phase 1 + Phase 4) |
| `go vet ./...` | clean |
| Pas de `fmt.Println` introduit | grep |
| `events` retiré de `warnUnimplemented` | grep |
| `levelup --help` liste `replay-events` | manuel |
| `cmd/replay_highlight_events/` supprimé | git status |
| Phase 1ter — tableau audit ajouté à thought_log | manuel |
| Phase 4 — fixture commitée + test E2E vert | manuel |
| Entrée thought_log `[2026-05-08]` ajoutée pour ce plan | obligatoire |

---

## 4. Découpage en commits (1 branche, N commits ordonnés)

| # | Phase(s) | Message |
|---|---|---|
| 1 | Phase 1 | `refactor(sync): extract events_replay helpers from cmd/replay_highlight_events` |
| 2 | Phase 1bis | `fix(sync): bitmasks menteurs — InsertKVPairs unconditional + heal anomaly` |
| 3 | Phase 1ter | `docs(ai): audit lecture seule des bitmasks restants (PBit* + MBit*)` |
| 4 | Phase 2 | `feat(api): wire highlight events backfill in /backfill/start` |
| 5 | Phase 3 | `feat(cli): add levelup replay-events subcommand + remove standalone binary` |
| 6 | Phase 4 | `test(sync): golden match fixture E2E — guard contre régressions sync` |

Chaque commit livrable indépendamment ; `go test ./...` reste vert entre chaque.

---

## 5. Architecture — checks plan-review

| Check | État |
|---|---|
| Algos purs dans `internal/analysis/` | N/A — pas de nouvel algo, le parser bit-aligné est déjà livré |
| Types résultat dans `internal/domain/` ou `canonical/` | `ReplayResult` dans `internal/sync/` (couche orchestration, pas de dimension cross-titre) |
| Orchestration dans `internal/service/` | Non applicable — la logique est sync-spécifique, déjà conventionnellement dans `internal/sync/` (cf. `engagement_recompute.go`, `events_heal.go`) |
| Handlers HTTP dans `internal/api/handlers/` | Oui — Phase 2 |
| Aucun SQL dans handler / service | Oui — la SQL de détection est dans `events_replay.go` (couche sync), le handler ne fait qu'appeler |
| Multi-titres : `PathResolver` | Oui — Phase 3 CLI utilise `PathResolver` pour shared/global DB |
| Multi-titres : `HasCapability()` | Le replay est sync-only et ne touche pas aux capabilities (la feature "highlight events" est implicite à Halo Infinite — pas de gating capability nécessaire pour cette itération, à reconsidérer si un autre titre est ajouté) |
| Tests à chaque couche | Oui — sync (Phase 1), handler (Phase 2), CLI (couvert par tests sync), E2E pipeline (Phase 4) |
| Fixture API capturée | Oui — Phase 4, `internal/sync/testdata/golden_match/` (stats + skill + film manifest + chunks réels) |
| Logging via `slog` | Oui — `slog.WarnContext` partout, pas de `fmt.Println` introduit |
| Frontend impacté | Non — Phase 2 ouvre la voie pour un futur bouton "rescan events" mais le plan ne touche pas `apps/web/` |

---

## 6. Risques et hors-scope

### Risques identifiés

| Risque | Mitigation |
|---|---|
| `globalDB` non ouvert dans le handler HTTP | Best-effort via `PathResolver` + `OpenReadWrite`, dégradation silencieuse si lockée (cohérent avec le tool replay) |
| Concurrence — un sync simultané sur le même joueur peut interférer avec le replay | Le pipeline existant utilise déjà des `dblease.Acquire*` ; le replay doit faire pareil. À expliciter dans l'implémentation Phase 1 |
| Remontée du `parse_anomaly` dans le warning de job HTTP — peut polluer la liste des warnings sur des matchs anormaux légitimes (très courts, lobby quitté) | Le compteur expvar reste la source de vérité ; un warning par match sur un job de 100 matchs n'est pas problématique |
| Phase 1ter peut révéler 5+ bits menteurs supplémentaires | Si découvert : ouvrir un plan séparé `PLAN_BITMASKS_HONESTY_AUDIT.md`, ne PAS étendre ce plan |

### Hors-scope explicite

- Fix des autres bits menteurs détectés en Phase 1ter (réservé à un plan dédié)
- CLI générique `levelup audit-bitmasks` qui détecterait les rows incohérentes (bit set mais data absente) — utile mais autre task
- Ajout d'un flag `--force-events` à `levelup sync-delta` (sémantique floue : sync-delta = nouveaux matchs, pas re-traitement ; la voie d'entrée pour le re-traitement reste `replay-events`)
- Front : aucun composant `apps/web/` n'est touché. Une éventuelle tuile "rescan highlight events" est laissée à un futur plan UX
- **RC4 / RC5 / RC6 du plan jumeau `PLAN_RECENT_MATCH_REGRESSION_FIX.md`** : ces sujets (asymétrie i18n match-view, xuid_aliases backfill cross-match, handler graceful 404→200+is_partial) restent dans leur plan d'origine. Ce plan-ci ne les absorbe pas — ils sont indépendants du périmètre highlight events
- **Fixture multi-titres** : Phase 4 capture un match Halo Infinite uniquement. Quand un autre titre arrivera, ajouter `internal/sync/testdata/golden_match_<title>/` (extension naturelle, pas un nouveau plan)

---

## 7. Effort total estimé

| Phase | Effort |
|---|---|
| Phase 1 — extraction helpers | 1 h |
| Phase 1bis — fix bitmasks events+kvp | 30-45 min |
| Phase 1ter — audit lecture seule | 20-30 min |
| Phase 2 — handler HTTP | 45 min |
| Phase 3 — CLI + suppression binaire | 30 min |
| Phase 4 — golden fixture E2E + test | 2 h |
| Phase 5 — vérifs + thought_log | 15 min |
| **Total** | **~5 h 15** |

---

## 8. Done definition globale

- [ ] Phase 1 : helper `internal/sync/events_replay.go` + tests intégration verts + `mockHaloClient` étendu
- [ ] Phase 1bis : 2 fixes bitmasks + 2 nouveaux tests + audit events/kvp = 5/5 OK
- [ ] Phase 1ter : tableau audit complet ajouté à `thought_log.md`, tous les bits classés
- [ ] Phase 2 : handler câblé + `events` retiré de `warnUnimplemented` + tests handler verts
- [ ] Phase 3 : sous-commande `levelup replay-events` listée dans `--help` + `cmd/replay_highlight_events/` supprimé
- [ ] Phase 4 : fixture golden capturée + commitée + test E2E vert + `cmd/refresh_golden_fixture` opérationnel
- [ ] Phase 5 : `go test ./...` + `go vet ./...` clean + entrée thought_log `[2026-05-08]` pour ce plan

---

## 9. Référence croisée

- Commits déjà livrés : `64f6720b` (parser bit-aligné), `34c7f646` (replay tool + InsertKVP fix)
- Diagnostic original : entrées thought_log `[2026-05-07]` et `[2026-05-08]`
- Code touché : `internal/analysis/highlight_event_parser.go`, `internal/sync/{engine.go, events_heal.go, writes.go, halo_client_mock_test.go, highlight_events_test.go, highlight_events_orchestration_test.go}`, `internal/migration/steps_shared.go`
- Tool actuel à supprimer : `cmd/replay_highlight_events/main.go`
- Document jumeau : `.ai/PLAN_RECENT_MATCH_REGRESSION_FIX.md` (RC1 à RC6 + Phase A/B/C/D)

### Articulation avec le plan jumeau

| Item du plan jumeau | Dans ce plan | Hors-scope (reste dans le jumeau) |
|---|---|---|
| RC1 — parser bit-aligné | Déjà livré (commit `64f6720b`) | — |
| RC2 — lying bits structuraux | **Phase 1bis** (events + kvp) + **Phase 1ter** (audit du reste) | Fix des autres bits si Phase 1ter en révèle (plan dédié) |
| RC3 — cascade dependency weapons | Implicite (résolu par RC1 + Phase 1bis) | — |
| RC4 — i18n cascade match-view UUID→nom | — | **Reste dans le jumeau (Phase A)** — sujet match-view, pas highlight events |
| RC5 — xuid_aliases backfill cross-match | Vérifié par golden test (Phase 4) | **Backfill cross-match reste dans le jumeau (Phase B)** — l'invariant est testé ici, l'implémentation est ailleurs |
| RC6 — handler 404 sur partial data | — | **Reste dans le jumeau (Phase C)** — sujet API/front, pas highlight events |
| Phase D invariant 1 (pipeline complète) | **Phase 4** (golden test couvre cet invariant et plus) | Devient redondant côté jumeau |
| Phase D invariant 2 (no-film cascade) | **Phase 4** (variante `TestSyncPipeline_GoldenMatch_NoFilm_CascadeRespected`) | Devient redondant |
| Phase D invariant 3 (lying bits detector) | **Phase 1bis** tests | Redondant si Phase 1bis livrée |
| Phase D invariant 4 (home/match-view consistency) | — | **Reste dans le jumeau** — sujet RC4 |
| Phase D invariant 5 (alias coverage post-sync) | Partiellement couvert par golden test | **Backfill cross-match reste dans le jumeau (RC5)** ; invariant hot-path testé ici |

**Recommandation** : exécuter ce plan en premier (les Phases 1+1bis+4 produisent les guarantees structurelles). Le plan jumeau peut ensuite se concentrer sur les Phases A/B/C (UX/data) sans avoir à re-scoper les invariants Phase D — ils auront été couverts.
