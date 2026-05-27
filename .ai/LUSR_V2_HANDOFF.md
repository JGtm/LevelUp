# LUSR v2 — Handoff Phase 3e+ → Phase 5 (state au 2026-05-27)

> Document de reprise. Lis ce fichier avant de toucher au LUSR v2.

## TL;DR

5 chantiers autonomes livrés sur la branche `refactor/split-god-files-1000plus` :

| # | Chantier | Status | Tests |
|---|---|---|---|
| 1 | Bascule prod Stratégie C (canonical writer LUSR_V2) | LIVRÉ | `TestRunLUSRV2Shadow_Canonical_*` |
| 2 | Sentinelle dual-row + métriques expvar | LIVRÉ | `TestRunDualRowSentinel_DetectsInconsistencies` |
| 3 | Quit penalty (post-EP shift, NOT proper TS2 §9) | LIVRÉ | `TestPhase3quit_*` |
| 4 | Mode correlation (leak post-EP, w_d cap 0.4) | LIVRÉ | `TestRunLUSRV2Shadow_Phase4_*` |
| 5 | TTT batch CLI (stats empiriques par groupe) | LIVRÉ partiel | dry-run OK sur data réelle |

Aucun commit n'a encore été poussé — l'utilisateur a la consigne explicite **"demander avant tout commit"**. Voir la section "Commits suggérés" plus bas pour découper en 5-6 commits cohérents.

---

## Décisions importantes (à NE PAS contredire sans relire)

### Stratégie C, dual-row obligatoire
- L'env `LEVELUP_LUSR_CANONICAL=LUSR_V2` désactive `batchComputeLUSR` (v1) et fait écrire v2 dans `match_skill_rank` AVEC `rating_type='LUSR'` ET `rating_type='LUSR_V2'`.
- Les readers UI continuent de lire `LUSR` sans modification. Le slot `LUSR_V2` sert d'audit trail.
- Atomicité garantie par `AppendOnlyLUSRPersister.Persist` (1 transaction pour les 2 rows).
- La sentinelle (`RunDualRowSentinel`) doit être tournée périodiquement (cron + endpoint debug) pour détecter l'invariant cassé.

### Quit penalty : interprétation "communauté gaming", pas TS2 §9 littéral
- TS2 §9 propose une variable `u_p` (under-performance) qui ABSORBE la sous-perf des quitters et donc REMONTE leur skill. C'est l'inverse de ce que veut la communauté Halo.
- J'ai prototypé l'EP factor de TS2 §9 → vérifié que ça remonte effectivement le skill du quitter → REJETÉ, fichier supprimé.
- Implémentation actuelle : `applyQuitPenaltyPost` retire `QuitPenaltyDelta` à μ après EP. Constantes :
  - `DefaultQuitDeltaRelated = 1.0` (team perdait → modéré)
  - `DefaultQuitDeltaUnrelated = 2.5` (team gagnait/égalisait → fort)
- Source heuristique : `outcome` final du match approxime "related" (loss) vs "unrelated" (win/draw). Si on récupère un jour la timeline du score, on pourra utiliser le score AU MOMENT DU QUIT — meilleur signal.
- Discrimination quit vs late-join via `isQuitter()` (`left_in_progress=TRUE` OU `(present_at_beginning && !present_at_completion)`). Les late-joiners ne sont pas pénalisés (ils ont déjà des counts plus bas).

### Priority quitter : 1er = delta plein, suivants = 50% (`QuitSecondaryFactor = 0.5`)
- Décision produit 2026-05-27 : "le 1er joueur qui quitte porte la responsabilité, les suivants subissent la cascade et voient leur malus réduit de 50%".
- Signal idéal : **`last_leave_time`** (TIMESTAMPTZ absolu de l'API `ParticipationInfo.LastLeaveTime`). Backfillé via `cmd/backfill_quit_timestamps` (2026-05-27).
- Fallback : `time_played_seconds` (proxy : less play time = quit earlier) si pas de timestamp.
- Stratégie `identifyPrimaryQuitter` (par ordre de précision) :
  1. **Préférence 1** : si AU MOINS un quitter a `last_leave_time` → choisir le min parmi CEUX qui l'ont. Les autres (sans timestamp) sont automatiquement secondaires — on ne mélange jamais timestamp et time_played dans la comparaison.
  2. **Préférence 2** : sinon, min sur `time_played_seconds` parmi ceux qui l'ont.
  3. **Si aucun n'a NI timestamp NI time_played** → primary = "" → tous traités comme primaires (matchs ancien backfill participation booleans seul).
- `scaledQuitDelta(xuid, primary, base)` : `base` si xuid==primary, sinon `base * 0.5`. `primary=""` → tous primaires (fallback safe).
- Si on change `QuitSecondaryFactor`, garder le ratio plein/réduit ≥ 2× pour préserver l'incitation à NE PAS être le premier quitter.

### ParticipationInfo timestamps capturés (2026-05-27)
- 2 nouvelles colonnes shared `match_participants.{first_joined_time, last_leave_time}` (TIMESTAMPTZ).
- Mapper + transforms parsent `ParticipationInfo.FirstJoinedTime` et `LastLeaveTime` (already in `openspartan.models.go` — étaient jetés avant).
- `isQuitter()` utilise `last_leave_time IS NOT NULL` comme signal direct (top priority), avant les booleans heuristics.
- Backfill historique : `cmd/backfill_quit_timestamps` re-fetche les JSON via Halo API et UPDATE les rows NULL. ~1700 matchs en ~10 min (3 rps).
- Matchs > 6 mois pourront retourner HTTP 404/410 (Halo API expire les stats anciennes) — ces matchs gardent `first_joined_time = NULL` et tombent en fallback `time_played_seconds`.

### Mode correlation cap w_d ≤ 0.4 — contrainte produit
- Reformulation user 2026-05-27 : "Chaos c'est les modes où on meurt beaucoup mais c'est ultra fun, slayer c'est plus engageant et team play. Je suis ok pour la correlation mais pas d'influence abusive."
- Implémentation : `ApplyCrossModeLeak(oldMuOther, oldMuPrimary, newMuPrimary, weight)` clampe `weight` à `[0, Phase4ModeCouplingMaxWeight=0.4]`.
- Valeur runtime par défaut : `DefaultModeCouplingWeight = 0.3`.
- **NE JAMAIS** monter le cap au-dessus de 0.4 sans validation explicite. Si Phase 5 batch suggère plus, garder le cap et logger un warn.
- Activation gated par `LEVELUP_LUSR_V2_MODE_COUPLING=1` (off par défaut pour éviter régression silencieuse).

---

## Fichiers livrés (résumé)

### Modifs analysis/skill_v2/
- `legacy_mapping.go` + `legacy_mapping_test.go` — mapping μ v2 → rating v1 (Stratégie C)
- `mode_correlation.go` + `mode_correlation_test.go` — leak Phase 4
- `trueskill_ep.go` — ajout `PlayerCounts.Quit`/`QuitPenaltyDelta`, `applyQuitPenaltyPost`, `DefaultQuitDelta{Related,Unrelated}`
- `trueskill_ep_quit_test.go` — 2 tests Phase 3-quit

### Modifs sync/
- `skill_v2_shadow.go` — gros patch :
  - `IsLUSRV2Canonical()` + `IsLUSRV2ModeCouplingEnabled()`
  - `RunLUSRV2Shadow(ctx, playerDB, sharedDB, xuid)` (nouvelle signature)
  - `writeCanonicalLUSRRow` (dual-row LUSR + LUSR_V2)
  - `propagateCrossModeLeak` + `findOwnerPrior`
  - `buildTwoTeamRosters` (lit présent_at_beginning, etc.)
  - `buildCountInputs(teamA, teamB, outcomeA)` (signature changée)
  - `isQuitter`, `quitDeltaForTeam`, `invertOutcome`
- `skill_v2_metrics.go` (NEW) — 4 compteurs expvar + `RunDualRowSentinel` + `SentinelReport`
- `skill_v2_shadow_test.go` — 4 tests E2E (Canonical, Sentinel, Phase4_CrossModeLeak, Phase4_OffByDefault)
- `engine_postsync.go` — skip `batchComputeLUSR` si `IsLUSRV2Canonical()`

### Modifs persist/
- `lusr_append_only_persister.go` — ajout `RatingType` field (défaut "LUSR" si vide) au lieu du hardcode

### Nouveau cmd
- `cmd/lusr_v2_ttt_batch/main.go` — CLI stats empiriques par playlist_group, écrit dans `lusr_hyperparams_v2`

### Modifs platform/duckdb/
- `skill_v2_repo.go` — ajout `LoadAllStates(ctx, xuid)` pour Phase 4

### Modifs cmd
- `cmd/lusr_v2_replay/main.go` — adapté à la nouvelle signature `RunLUSRV2Shadow(ctx, nil, db, xuid)`

---

## Tests à lancer avant tout merge

```bash
cd apps/go-api
go test ./internal/sync/ ./internal/persist/ ./internal/analysis/skill_v2/...
go vet ./...
go build ./...
```

Au 2026-05-27 22:55, toutes les suites ci-dessus passent.

---

## Phase 5.B — Wiring restant côté shadow runner

Le `cmd/lusr_v2_ttt_batch` écrit les hyperparams empiriques dans `lusr_hyperparams_v2`, mais `RunLUSRV2Shadow` utilise toujours `skillv2.DefaultPriors()` hardcodé. Pour que la re-estimation soit EFFECTIVE :

1. Charger `lusr_hyperparams_v2_latest` au début de `RunLUSRV2Shadow` (par playlist_group).
2. Override `Priors.DrawProbability` avec `draw_probability_empirical`.
3. Override `CountHyperparams.Bias` (kill/death) avec `kill_mean_empirical` / `death_mean_empirical`.
4. Tester qu'un Mu0 ré-estimé proche de la moyenne kills donne un `expected_count_death` réaliste (cf. trace Phase 3c initiale qui avait montré le bug "expected_death < 0").

Pas urgent — les defaults Phase 3e v2 sont raisonnables. À ouvrir quand la prochaine session veut affiner.

---

## Phase 6 candidates (non implémentées)

- **TTT proper (forward + backward smoothing)** : pour les hyperparams sigma_skill / sigma_perf / sigma_dynamic. Le batch actuel ne fait QU'une passe forward agrégée. La version complète demande un solveur EM sur factor graph sériel — non trivial, ~1-2 jours.
- **Quit penalty avec timeline du score** : remplacer l'heuristique outcome-final par un check du score au moment du quit. Nécessite parser les `highlight_events` ou `match_progression` (à vérifier la source de vérité dans `internal/openspartan/`).
- **Mode correlation à coefficients ré-estimés par paire de modes** : remplacer le scalaire `DefaultModeCouplingWeight=0.3` par une matrice 4×4 `w_d[i][j]` calibrée empiriquement (corrélation observée entre μ_groupes pour les players multi-modes). Reste capé à 0.4.
- **UX delta dans match_skill_rank.rating_delta** : actuellement nul en canonical (cf. `writeCanonicalLUSRRow`). Pour le calculer il faudrait fetch le previous rating_value sur le même playlist_group.
- **Migration backfill `lusr_v2_replay` avec canonical=ON** : aujourd'hui le replay tourne en mode shadow uniquement. Pour repeupler historiquement les `rating_type='LUSR_V2'` rows, ajouter un flag `--canonical` au replay et lui passer un playerDB.

---

## Commits suggérés (à demander à l'user avant d'exécuter)

Pour faire des commits cohérents par chantier :

1. `feat(persist): parameterize rating_type in LUSRRatingInsert (Strategie C)` — `lusr_append_only_persister.go`
2. `feat(skill_v2): legacy_mapping mu v2 -> rating v1 LUSR` — `legacy_mapping.go` + test
3. `feat(sync): canonical writer Strategie C avec flag LEVELUP_LUSR_CANONICAL` — `skill_v2_shadow.go` (dual-row write) + `engine_postsync.go` skip + cmd lusr_v2_replay adapt + tests
4. `feat(sync): sentinelle dual-row + metriques expvar` — `skill_v2_metrics.go` + test
5. `feat(skill_v2): quit penalty post-EP avec related/unrelated` — `trueskill_ep.go` (PlayerCounts.Quit) + sync isQuitter + tests + migration backfill participation
6. `feat(skill_v2): mode correlation leak w_d cap 0.4 (Phase 4)` — `mode_correlation.go` + test + sync propagateCrossModeLeak + LoadAllStates repo + tests
7. `feat(skill_v2): TTT batch CLI hyperparams empiriques (Phase 5 MVP)` — `cmd/lusr_v2_ttt_batch/main.go`

Chaque commit doit avoir une entrée correspondante dans `.ai/thought_log.md` (règle CLAUDE.md projet).

---

## Activation prod — ordre recommandé

1. **Étape 1 (déjà possible)** : laisser les flags off → comportement = avant (v1 canonical, v2 shadow). Aucune régression possible.
2. **Étape 2** : activer `LEVELUP_LUSR_V2_ENABLED=1` (déjà le cas en prod actuellement). Le shadow runner produit des rows v2 dans `player_skill_state_v2`.
3. **Étape 3** : run `cmd/lusr_v2_replay` une fois pour rattraper l'historique (déjà fait selon thought_log).
4. **Étape 4** : activer `LEVELUP_LUSR_CANONICAL=LUSR_V2` en staging d'abord. Vérifier sur 3-5 syncs que :
   - Les rows `rating_type='LUSR'` du shadow runner ressemblent à ce que `batchComputeLUSR` aurait écrit (sanity vs grille v1).
   - La sentinelle ne lève aucune inconsistance.
   - Les expvar `levelup.lusr_v2.canonical_writes_total` montent.
5. **Étape 5** : prod. Garder `batchComputeLUSR` court-circuité.
6. **Étape 6 (optionnel)** : activer `LEVELUP_LUSR_V2_MODE_COUPLING=1` pour la Phase 4. Plus risqué — observer 1 semaine en staging avant prod.

---

## Anti-bug rappels rapides

- **Pas de `git stash`** — utiliser commit WIP si besoin.
- **Pas de Python** dans le projet — DuckDB CLI ou Go uniquement.
- **DEMANDER avant tout commit** — même dans une série planifiée.
- **Sentinelle false-positive si on cleanup les anciennes rows v1** — vérifier OnlyLUSR vs nombre de matchs syncés avant Stratégie C bascule.
- **Quit penalty constants** : si on ajoute un 3e cas (e.g., AFK détecté), garder le cap à `< 2 * DefaultQuitDeltaUnrelated` pour éviter qu'un AFK seul détruise un rating.
- **Mode coupling cap = 0.4 strict** — `ApplyCrossModeLeak` clampe automatiquement, mais si on ajoute du code qui multiplie w_d ailleurs, re-clampper.

---

Bonne reprise.
