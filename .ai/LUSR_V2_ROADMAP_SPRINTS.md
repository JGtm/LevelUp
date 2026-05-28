# LUSR v2 — Plan de sprints détaillé

> Document de pilotage. Coche les cases au fur et à mesure. Mets à jour la
> date de complétion en bas de chaque sprint.

**Branche cible** : `feat/lusr-v2-phase0-metrics` (branche LUSR v2 active).
Création de sous-branches optionnelle si plusieurs sprints en parallèle.

**Rappel règles projet** :
- `.ai/thought_log.md` à mettre à jour à chaque sprint complété
- Tests + `go vet ./...` clean avant de cocher "Sprint terminé"
- Pas de commit sans demander d'abord
- Logs structurés `slog.*Context` (auto-routés vers `logs/sync.log` etc.)

---

## Vue d'ensemble

| Sprint | Contenu | Effort | Priorité |
|---|---|---|---|
| **1.A** | Probabilité de victoire prédite | 1/2 jour | Haute (quick win UX) |
| **1.B** | Brancher les paramètres ré-estimés | 2-3h | Haute (corrige un bug latent) |
| **1.C** | Détection escouades + correction skill | 2 jours | Haute (angle mort modèle) |
| **2.A** | Timeline du score au moment du quit | 1/2 jour | Moyenne |
| **2.B** | Matrice de corrélation entre modes | 1 jour | Moyenne |
| **3.A** | Recalibration auto périodique propre | 1-2 jours | Basse |
| **3.B** | Delta de rating dans l'historique | 2h | Basse (confort UX) |
| **Final** | Bascule canonical en prod | 1 jour | Critique |

**Total estimé : ~8 jours de dev** réparti sur N sessions.

---

## Sprint 1.A — Probabilité de victoire prédite

> Référence détaillée : `.ai/LUSR_V2_WIN_PROBABILITY.md`

### Objectif
Exposer, pour chaque match, "votre équipe avait X% de chance de gagner" avant
que le match commence. Permet de filtrer les défaites entre "matchs perdables"
et "matchs vraiment au-dessus", et de mettre en valeur les belles perfs sur
matchs donnés perdants.

### Architecture prévue
- `internal/analysis/skill_v2/predict.go` — fonction pure `PredictTwoTeamWinProb(teamA, teamB []Gaussian, p Priors) (probA, probDraw, probB float64)`
- `internal/sync/skill_v2_predict_loader.go` — `LoadPreMatchStates(ctx, sharedDB, matchID) (teamA, teamB []Gaussian, error)`
- Colonne `expected_win_prob FLOAT` ajoutée à `match_skill_rank` (migration), écrite par `writeCanonicalLUSRRow` au moment de la Stratégie C
- Optionnel pour Sprint 2 : endpoint HTTP `/api/skill-v2/match/{id}/prediction` si on veut le calcul à la volée

### Étapes

- [x] **1.A.1** — Créé `internal/analysis/skill_v2/predict.go` : `PredictTwoTeamWinProb(teamA, teamB, p) (probA, probDraw, probB)`, draw margin, 0 accès DB. Helper privé `matchSpread` partagé avec `PredictWinProbability`/`PredictDrawProbability` (migrées depuis trueskill.go, refacto DRY).
- [x] **1.A.2** — `predict_test.go` : équilibré (probA=probB), favori net μ30 vs μ20 → probA>0.9, σ haut → tiré vers 0.5, draw_probability↑ → probDraw↑, équipe vide → neutre, cohérence avec les helpers legacy.
- [~] **1.A.3** — **ABANDONNÉ (write-off)** : `LoadPreMatchStates` non créé. Les états pré-match sont déjà en mémoire dans `applyMatchToSkillV2` (teamAStates/teamBStates) AVANT l'update ; un re-query lirait l'état POST-persist (faux) + requête redondante. Cf. thought_log 2026-05-28.
- [x] **1.A.4** — Migration `steps_player_add_expected_win_prob.go` (additive `addColumnIfMissing`, FLOAT).
- [x] **1.A.5** — `LUSRRatingInsert.ExpectedWinProb *float64` + INSERT (10→11 placeholders).
- [x] **1.A.6** — Wire : `applyMatchToSkillV2` calcule la prob (in-memory) et la retourne ; `processOneShadowMatch` → `writeCanonicalLUSRRow` (nouveau param) → pose sur les 2 rows LUSR + LUSR_V2.
- [x] **1.A.7** — Tests E2E : `TestRunLUSRV2Shadow_Canonical_StoresExpectedWinProb` (∈ [0,1]), `TestRunLUSRV2Shadow_Canonical_FirstMatchFallback` (joueurs neufs → ≈ 0.5, pas de panic).
- [x] **1.A.8** — expvar `levelup.lusr_v2.predictions_total`.
- [x] **1.A.9** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 1.A

- [x] `go test ./internal/analysis/skill_v2/... ./internal/sync/ ./internal/persist/` PASS (+ migration)
- [x] `go vet ./...` clean
- [ ] Au moins 1 match prod vérifié : la row `match_skill_rank` post-bascule contient un `expected_win_prob` plausible (proche de 0.5 ± 0.3 pour la plupart des matchs) — **différé : non exécutable hors prod**
- [x] Pas de panic en cas de match avec joueur jamais vu (fallback testé)
- [x] Entrée `.ai/thought_log.md` ajoutée
- [x] Commit unique sur la branche, autorisé par l'utilisateur

**Date complétion** : 2026-05-28

---

## Sprint 1.B — Brancher les paramètres ré-estimés

### Objectif
Le CLI `cmd/lusr_v2_ttt_batch` écrit déjà en base les stats empiriques par
mode (draw_probability, mean kills, mean deaths). Mais le code qui tourne au
sync utilise toujours les valeurs par défaut hardcodées. Conséquence :
re-estimation = écriture morte. Cette tâche connecte les deux.

### Architecture prévue
- `internal/analysis/skill_v2/hyperparams_load.go` — `LoadPriorsFromHyperparams(map[string]float64, defaultP Priors) Priors` (pure)
- Modification `RunLUSRV2Shadow` pour charger via `repo.LoadHyperparams(ctx, group)` au début de la boucle par groupe
- Cache LRU optionnel pour éviter une requête par match (pas nécessaire vu volume)

### Étapes

- [x] **1.B.1** — `internal/analysis/skill_v2/hyperparams_load.go` : `LoadPriorsFromHyperparams(params, defaultP)` override DrawProbability depuis `draw_probability_empirical` (guard [0,1[). + alias `CountType`/`CountHyperparams`, `DefaultCountHyperparamsMap`, `AppliedHyperparamCount`. Pur.
- [x] **1.B.2** — `LoadCountHyperparamsFromDB(params, mu0)`. **CORRECTION du plan** : le doc prescrivait `bias = kill_mean_empirical` — dimensionnellement faux pour `expected = bias + w_p·perf + w_o·avg_opp`. Formule correcte : `bias = mean − (w_p + w_o)·μ0` (réduit aux défauts pour mean ~12.5).
- [x] **1.B.3** — Tests unitaires : map vide → defaults ; draw_prob seul → seul DrawProbability change ; draw_prob invalide ignoré ; bias recalibré (12.5→defaults, 20/8→7.5/20.5).
- [x] **1.B.4** — `resolveGroupParams` (mémoïsé par groupe) charge les hyperparams via `repo.LoadHyperparams`, override Priors + CountHyperparams, log `slog.DebugContext "hyperparams ré-estimés appliqués"` (overrides + draw_probability). Best-effort (échec → fallback defaults + warn).
- [x] **1.B.5** — `applyMatchToSkillV2` prend `priors` (groupPriors résolu, pas c.priors) + nouveau param `countHyp` posé sur `counts.Hyperparams`. Threading via champ optionnel `CountInputs.Hyperparams` (nil → defaults) → 0 régression EP.
- [x] **1.B.6** — E2E `TestRunLUSRV2Shadow_UsesEmpiricalDrawProb` : match nul 2v2 symétrique sans counts, draw_prob=0.5 seedé vs default → σ owner plus grand (draw moins surprenant). DDL test étendu (lusr_hyperparams_v2).
- [x] **1.B.7** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 1.B

- [x] `go test ./internal/analysis/skill_v2/... ./internal/sync/` PASS
- [ ] Run manuel : `LEVELUP_LUSR_V2_ENABLED=1 ./server` → log "hyperparams ré-estimés appliqués" count > 0 — **différé : non exécutable hors prod (requiert un passage TTT batch préalable)**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28

---

## Sprint 1.C — Détection escouades + correction skill (TS2 §7)

### Objectif
Aujourd'hui le modèle traite 4 joueurs en escouade comme 4 indépendants. Or
un 4-stack régulier gagne plus que la somme de ses parties. Conséquence :
les ratings des joueurs en escouade sont un peu sur-estimés. C'est l'angle
mort le plus impactant en pratique (les 4 joueurs trackés jouent souvent
ensemble).

### Architecture prévue
- `internal/analysis/skill_v2/squad.go` — types et fonction pure de calcul d'offset par squad
- `internal/sync/skill_v2_squad.go` — détection des squads basée sur co-occurrence dans les matchs récents (par paire de XUIDs, fenêtre glissante)
- Migration : nouvelle table `player_squad_offset` (xuid, partner_xuid, playlist_group, offset_value, last_updated)
- L'offset est appliqué AVANT l'EP update (réduit le μ effectif du joueur dans le facteur graph)

### Étapes

- [x] **1.C.1** — Définition opérationnelle : paire de XUIDs sur la même équipe ≥ N matchs (défaut 10) dans les M semaines (défaut 4), par playlist_group. Paramétrable via flags CLI.
- [x] **1.C.2** — Migration `steps_shared_create_squad_offset.go` : `player_squad_offset` append-only (PK `id`) + vue `_latest`. TargetShared.
- [x] **1.C.3** — Repo `duckdb.SquadOffsetRepo` : `LoadSquadOffsets(xuid, group)` (cache run-scoped) + `UpsertSquadOffset(domain.SquadOffset)` (INSERT pur).
- [x] **1.C.4** — Algo pur `skill_v2/squad.go` : `ComputeSquadOffset([]SquadCoMatch, gain)` = `mean(Won − SoloWinProb) × gain`, clampé ±`SquadOffsetCap`(2.0). + `ApplySquadOffset`/`ClampSquadOffset`.
- [x] **1.C.5** — CLI `cmd/lusr_v2_squad_estimate` : scan fenêtre, paires éligibles, offset, écriture 2 sens (symétrique), `--dry-run`/`--weeks`/`--min-matches`/`--gain`. **MVP first-pass** : SoloWinProb = proxy ratings solo courants (biais conservateur, à raffiner Sprint 3.A).
- [x] **1.C.6** — Application dans `applyMatchToSkillV2` (gated `LEVELUP_LUSR_V2_SQUAD_OFFSET=1`) : μ effectif = μ + Σ(offsets partenaires présents) avant EP, retiré du posterior après. σ inchangé. Flag off → no-op exact.
- [x] **1.C.7** — Tests unitaires squad : 0 historique → 0 ; sur-perf → offset attendu ; clamp ±2.0 ; sous-perf → −2.0.
- [x] **1.C.8** — E2E `TestRunLUSRV2Shadow_AppliesSquadOffset` : offset +1.5, victoire squad → μ owner monte MOINS qu'sans offset (anti-inflation).
- [x] **1.C.9** — `slog.DebugContext "LUSR v2 squad offset appliqué"` (xuid, group, offset, partners_present).
- [x] **1.C.10** — Handoff doc : section "Phase 2 — Squad offset" ajoutée + Phase 5.B marquée livrée + étape 7 activation prod.
- [x] **1.C.11** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 1.C

- [x] `go test ./internal/analysis/skill_v2/... ./internal/sync/ ./internal/platform/duckdb/` PASS (+ migration), `go vet ./...` clean
- [ ] CLI `lusr_v2_squad_estimate` tourne sur prod, offsets ∈ [-2,+2] pour les 4 trackés — **différé : prod uniquement**
- [ ] Dry-run replay avec offsets actifs, pas de drift > 1.5 pt sur les solo — **différé : prod uniquement (flag OFF par défaut en attendant)**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28 (livré gated OFF ; activation prod après validation)

---

## Sprint 2.A — Timeline du score au moment du quit — ⛔ ABANDONNÉ (2026-05-28)

> **Donnée API absente** (issue prévue par 2.A.1). Le modèle openspartan n'expose
> que des scores FINAUX (`CoreStats.Score`, `RoundsWon/Lost/Tied`) ; `highlight_events`
> = highlights curés (kills/médailles) sans flux de scoring complet ni attribution
> d'équipe → impossible de reconstruire fiablement le score-au-quit (inutile aussi
> en modes objectifs où kills ≠ score). L'heuristique outcome-final (`quitDeltaForTeam`)
> reste en place. Réouverture si une source film/round-by-round horodatée apparaît.
> Cf. `.ai/thought_log.md` 2026-05-28.

### Objectif
Aujourd'hui on classe le quit "related" (équipe perdait) vs "unrelated"
(équipe gagnait) sur l'outcome FINAL du match. Si l'API Halo fournit
l'historique du score, on peut savoir SI le quitter est parti pendant que
son équipe était dominée ou pendant qu'elle dominait. C'est un signal plus
juste.

### Étapes

- [ ] **2.A.1** — Investiguer la source de vérité : grep `highlight_events`, `match_progression`, `events`, `score_timeline` dans `internal/openspartan/models.go`
  - Si la donnée existe : continuer
  - Si elle n'existe pas : abandonner ce sprint avec note dans handoff
- [ ] **2.A.2** — Si disponible : nouvelle fonction `inferQuitContextFromTimeline(events, leaveTime, ownerTeam) (related bool, scoreDiff int)`
- [ ] **2.A.3** — Modifier `quitDeltaForTeam(o)` en `quitDeltaForContext(scoreContext)` qui prend le contexte au moment du quit
- [ ] **2.A.4** — Tests unitaires sur 3 cas : équipe gagnait, équipe perdait, équipe à égalité
- [ ] **2.A.5** — Mise à jour `.ai/thought_log.md` + handoff

### Definition of Done — Sprint 2.A

- [ ] Tests PASS (ou sprint marqué abandonné avec justification si donnée API absente)
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

---

## Sprint 2.B — Matrice de corrélation entre modes

### Objectif
Le coefficient de leak cross-mode actuel est un scalaire unique (0.3) entre
tous les modes. En pratique slayer↔objectif (gunplay similaire) sont sûrement
plus corrélés que slayer↔chaos (chaos = mayhem). Une matrice
`coupling[mode_source][mode_target]` calibrée empiriquement reflète mieux.

### Étapes

- [x] **2.B.1** — `skill_v2/mode_correlation_matrix.go` : `EstimateCouplingMatrix` = Pearson des μ entre modes (joueurs ayant les 2), clampé [0, 0.4]. Symétrique ; négatif → 0 ; < 3 joueurs → pas d'entrée.
- [x] **2.B.2** — Stockage rows `mode_coupling_<source>_<target>` dans `lusr_hyperparams_v2` (playlist_group=source). Helpers `ModeCouplingHyperparamName` + `CouplingWeightFor`.
- [x] **2.B.3** — `propagateCrossModeLeak` charge `LoadHyperparams(source)` + `CouplingWeightFor(...)` par target, fallback `DefaultModeCouplingWeight` si pas d'entrée (comportement Phase 4 inchangé tant que matrice absente).
- [x] **2.B.4** — CLI `lusr_v2_ttt_batch` étendu (`mode_coupling.go`) : `loadPlayerStatesByXUID` + `EstimateCouplingMatrix` + `writeModeCoupling` (2 sens) + rapport.
- [x] **2.B.5** — Tests unitaires : corrélés parfaits → 0.4 ; décorrélés (orthogonaux) → 0 ; anti-corrélation → 0 ; sous-seuil → pas d'entrée ; CouplingWeightFor fallback.
- [x] **2.B.6** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 2.B

- [x] Tests PASS (skill_v2 + sync), `go vet` clean
- [ ] Batch tourne sur prod, matrice 4×4 cohérente (slayer↔objectif > slayer↔chaos) — **différé : prod (avec ~4 trackés, beaucoup de paires sous le seuil → fallback scalaire)**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28

---

## Sprint 3.A — Recalibration auto périodique propre (TTT complet)

### Objectif
Aujourd'hui le CLI `lusr_v2_ttt_batch` fait une simple agrégation forward
(moyennes par groupe). La version complète "Through-Time" fait un
forward + backward smoothing sur l'historique complet, ce qui ré-estime aussi
les paramètres internes du modèle (incertitude initiale, bruit de
performance, dynamique temporelle).

C'est du nice-to-have — la version simple suffit pour ajuster les biais
empiriques (draw rate, kill/death means). Cette tâche améliore la précision
long-terme.

### Étapes

> **Livré en tant que PROTOTYPE pur** (`skill_v2/ttt.go`). Le couplage inter-joueurs
> complet (factor graph TS2 §10) + le câblage prod restent un follow-up (la partie
> 1-2j+ risquée ; comparaison prod 3.A.6 de toute façon différée). Cf. thought_log 2026-05-28.

- [x] **3.A.1** — Étude TS2 §10 : modèle état-espace linéaire-gaussien (random walk + observation), inférence forward (Kalman) + backward (RTS) + EM.
- [x] **3.A.2** — Passe backward : `rtsBackward` (lisseur RTS + covariance lag-one) propage l'info des états futurs vers les passés.
- [x] **3.A.3** — EM loop : `EstimateTTT` (kalmanForward → rtsBackward → ré-estime q=τ² et r) jusqu'à convergence.
- [x] **3.A.4** — Convergence sur `|Δq| + |Δr| < Tol`.
- [x] **3.A.5** — Tests synthétiques : convergence < 10 itérations ; log-vraisemblance croissante (propriété EM) ; lisseur ≤ filtre (RMSE) ; edge cases 0/1 obs.
- [ ] **3.A.6** — Comparaison ratings prod avant/après — **différé : nécessite le TTT couplé câblé en prod (follow-up)**
- [x] **3.A.7** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 3.A

- [x] Tests PASS, convergence atteinte (prototype pur)
- [ ] Replay sur prod sans changement abrupt de tier — **différé : follow-up TTT couplé**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28 (prototype ; TTT couplé prod = follow-up)

---

## Sprint 3.B — Delta de rating dans l'historique

### Objectif
Aujourd'hui quand on écrit une ligne `match_skill_rank`, le champ
`rating_delta` est nul. Pour afficher au joueur "vous avez gagné +12 LUSR
ce match", il faudrait fetcher le rating précédent. Petit confort UX.

### Étapes

- [x] **3.B.1** — `writeCanonicalLUSRRow` : helper `loadPreviousLUSRRating` (rating LUSR le plus récemment écrit du groupe, ordre written_at DESC, id DESC), `rating_delta = rating - précédent`, nil au premier match. Best-effort (erreur → nil + warn).
- [x] **3.B.2** — E2E `TestRunLUSRV2Shadow_Canonical_RatingDelta` : 2 matchs successifs → m1 delta NULL, m2 delta non-nul.
- [x] **3.B.3** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 3.B

- [x] Tests PASS (sync), `go vet` + gofmt clean
- [ ] Vérifier en prod après quelques matchs que `rating_delta` est populé — **différé : prod**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28

---

## Sprint Final — Bascule canonical en prod

### Objectif
Activer `LEVELUP_LUSR_CANONICAL=LUSR_V2` en prod pour que le LUSR v2 devienne
le writer officiel du rating affiché à l'UI. Le LUSR v1 est désactivé en
même temps. À faire APRÈS Sprint 1 (les 3 quick wins) et idéalement APRÈS
Sprint 1.C (squad offset) pour ne pas montrer des ratings biaisés.

### Pré-requis avant de lancer
- [ ] Sprints 1.A + 1.B + 1.C terminés (les 3 sont sur la branche LUSR v2 actuelle)
- [ ] Backfill v2 historique complet : `cmd/lusr_v2_replay` re-tourné post-Sprint-1.C
- [ ] Vérifier `expvar /debug/vars` montre `levelup.lusr_v2.canonical_writes_total` qui monte avant la bascule (en mode shadow d'abord)

### Étapes

- [ ] **F.1** — Staging : poser `LEVELUP_LUSR_CANONICAL=LUSR_V2` + redémarrer
- [ ] **F.2** — Trigger 3-5 syncs manuels sur joueurs trackés
- [ ] **F.3** — Vérifier dans `logs/sync.log` :
  - Pas de warn "fallback shadow" (= playerDB pas câblé)
  - Pas de warn "canonical write échoué"
  - Le log "post-sync: LUSR v1 skippé" apparaît
- [ ] **F.4** — Vérifier la table `match_skill_rank` du player DB :
  - Rows avec `rating_type='LUSR'` ET `rating_type='LUSR_V2'` pour les nouveaux matchs
  - Les anciens matchs ont juste `rating_type='LUSR'` (héritage v1, légitime)
- [ ] **F.5** — Lancer `RunDualRowSentinel` manuellement → `OnlyLUSRV2` doit être à 0
- [ ] **F.6** — Vérifier dans l'UI : la page LUSR du joueur affiche un rating cohérent avec ses tiers tracked (Madina Diamant, Choco Or, etc.)
- [ ] **F.7** — Si tout OK staging : passer en prod
- [ ] **F.8** — Monitoring 24h en prod :
  - `levelup.lusr_v2.canonical_writes_total` monte
  - `levelup.lusr_v2.canonical_write_errors_total` reste à 0
  - `levelup.lusr_v2.dual_row_inconsistencies_total` reste à 0
  - Rapports utilisateurs : pas de "mon rating a saute de 200 points"
- [ ] **F.9** — Une fois stable : supprimer `batchComputeLUSR` v1 et le branchement `IsLUSRV2Canonical()` (le flag devient permanent ON)
- [ ] **F.10** — Mise à jour `.ai/thought_log.md` + suppression de `.ai/LUSR_V2_HANDOFF.md` (remplacé par la version stable dans `docs/`)

### Definition of Done — Sprint Final

- [ ] `LEVELUP_LUSR_CANONICAL=LUSR_V2` en prod depuis ≥ 7 jours
- [ ] 0 erreur dans expvar, 0 inconsistance sentinelle
- [ ] Aucun report utilisateur de rating bizarre
- [ ] Code v1 supprimé (batchComputeLUSR + flag canonical)
- [ ] Documentation finalisée

**Date complétion** : _______________

---

## Checklist transverse (à chaque sprint)

À cocher avant de marquer un sprint "Done" :

- [ ] Architecture respectée : analyse pure dans `internal/analysis/`, orchestration dans `internal/sync/` ou `internal/service/`, accès DB dans `internal/platform/duckdb/`
- [ ] Pas de SQL dans `analysis/`, pas d'algo dans handlers
- [ ] Tests à chaque couche modifiée (au moins 1 happy path + 1 cas dégradé)
- [ ] Logs structurés `slog.*Context(ctx, "...", "key", val)`
- [ ] Pas de `fmt.Println` ou `log.Printf` introduits
- [ ] `go test ./...` + `go vet ./...` clean
- [ ] Aucun fichier > 500L introduit (sinon split)
- [ ] Aucune fonction > 80L introduite (sinon extract)
- [ ] Pas de code title-specific (`if slug == "halo_infinite"`) — passer par capabilities
- [ ] Entrée `.ai/thought_log.md` ajoutée
- [ ] Commit autorisé par l'utilisateur (rule mémoire "demander avant tout commit")

---

## Notes de pilotage

**Bascule canonical à la fin OU avant si nécessaire** : le plan place le
Sprint Final à la fin (pour avoir le modèle le plus précis avant de l'exposer
en UI). Mais si à un moment l'utilisateur veut accélérer (par exemple parce
qu'il voit l'UI mais avec des ratings v1 inexacts), on peut basculer après
Sprint 1.B sans attendre 1.C. Trade-off : ratings v2 visibles mais sans
correction squad → un peu biaisés vers le haut pour les multi-stack.

**Ordre d'exécution recommandé** :
1. Sprint 1.A (1/2j) — débloque l'UX immédiatement
2. Sprint 1.B (2-3h) — corrige les ratings sans rien changer côté code visible
3. Sprint 1.C (2j) — corrige l'angle mort le plus impactant
4. Sprint Final si on veut passer en prod
5. Sprints 2.A / 2.B / 3.A / 3.B au fil de l'eau ensuite

**Si on doit livrer "à minima"** : juste les 3 sprints du Sprint 1 + Sprint
Final = ~3 jours de dev, et on a tout l'essentiel.
