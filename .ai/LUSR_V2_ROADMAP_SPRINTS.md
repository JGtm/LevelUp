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

- [ ] **1.C.1** — Décider de la définition opérationnelle de "squad" :
  - Pair de XUIDs ayant joué ensemble ≥ N matchs dans les M dernières semaines ?
  - N=10, M=4 semaines → à valider sur prod : compter combien de paires éligibles pour les 4 joueurs trackés
- [ ] **1.C.2** — Migration `steps_shared_create_squad_offset.go` :
  - Table `player_squad_offset` append-only (PK technique `id`, clé logique `(xuid, partner_xuid, playlist_group, written_at)`)
  - Vue `player_squad_offset_latest`
- [ ] **1.C.3** — Repo `internal/platform/duckdb/squad_offset_repo.go` :
  - `LoadSquadOffsets(ctx, xuid, group) (map[string]float64, error)` — par partner_xuid
  - `UpsertSquadOffset(ctx, offset domain.SquadOffset) error`
- [ ] **1.C.4** — Algorithme pur `internal/analysis/skill_v2/squad.go` :
  - `ComputeSquadOffset(matchHistory []SquadMatch) float64`
  - Approche : moyenne de (perf_real - perf_predicted_solo) sur les matchs où la paire a joué ensemble
  - Borné à [-2.0, +2.0] (sécurité — éviter offsets délirants sur petit échantillon)
- [ ] **1.C.5** — CLI `cmd/lusr_v2_squad_estimate/` :
  - Scan match history, identifie paires éligibles, calcule l'offset, écrit en DB
  - Idempotent (UPSERT append-only)
- [ ] **1.C.6** — Application de l'offset dans `processOneShadowMatch` :
  - Avant `applyMatchToSkillV2`, charger les offsets pour chaque partenaire présent dans le match
  - Réduire le μ "effectif" de chaque joueur : `mu_effectif = mu_individuel + Σ(offset[partner] pour chaque partenaire présent)`
  - L'EP update bouge `mu_individuel`, pas `mu_effectif` (l'offset est constant pour ce match)
- [ ] **1.C.7** — Tests unitaires (`squad_test.go`) :
  - 0 historique → offset = 0
  - 10 matchs avec gain systématique +1.5 par-rapport-au-solo → offset proche de 1.5
  - 5 matchs avec gain +5 → offset clampé à 2.0
- [ ] **1.C.8** — Tests E2E :
  - `TestRunLUSRV2Shadow_AppliesSquadOffset` — 2 paires (haut squad, faible solo) jouant ensemble → leurs μ individuels baissent moins que si pas d'offset
- [ ] **1.C.9** — `slog.DebugContext` "squad offset appliqué" avec xuid, partner, value
- [ ] **1.C.10** — Mise à jour handoff doc (`LUSR_V2_HANDOFF.md`) section "Phase 2 (squad offset)"
- [ ] **1.C.11** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 1.C

- [ ] `go test ./internal/analysis/skill_v2/... ./internal/sync/ ./internal/platform/duckdb/` PASS
- [ ] CLI `lusr_v2_squad_estimate` tourne sur prod, produit des offsets dans [-2, +2] pour les 4 joueurs trackés (cohérents avec l'intuition : Madina/Choco/JGtm trio sûrement positif)
- [ ] Dry-run replay du LUSR v2 avec offsets actifs → comparer μ avant/après pour les 4 trackés : pas de drift > 1.5 point pour les solo, possible drift > 0.5 pour les multi-stack
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

---

## Sprint 2.A — Timeline du score au moment du quit

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

- [ ] **2.B.1** — Algorithme pur `internal/analysis/skill_v2/mode_correlation_matrix.go` :
  - `EstimateCouplingMatrix(playerStates map[xuid][]GroupState) map[string]map[string]float64`
  - Pour chaque paire (mode A, mode B), corrélation de Pearson entre μ_A et μ_B sur l'ensemble des joueurs ayant joué les 2
  - Cap à 0.4 (règle métier)
- [ ] **2.B.2** — Stockage : étendre `lusr_hyperparams_v2` avec rows
  `mode_coupling_<source>_<target>` (source datée "batch_YYYY_MM_DD")
- [ ] **2.B.3** — Modifier `propagateCrossModeLeak` pour utiliser la matrice
  au lieu du scalaire
- [ ] **2.B.4** — Étendre le CLI `lusr_v2_ttt_batch` pour calculer + écrire
  la matrice
- [ ] **2.B.5** — Tests unitaires : 3 modes parfaitement corrélés → matrice = 0.4 partout, 3 modes décorrélés → matrice = 0
- [ ] **2.B.6** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 2.B

- [ ] Tests PASS
- [ ] Batch tourne sur prod, matrice 4×4 cohérente (slayer↔objectif > slayer↔chaos)
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

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

- [ ] **3.A.1** — Étudier le paper TS2 §10 (Minka et al. 2018), section
  "Batch through-time inference"
- [ ] **3.A.2** — Prototype passe backward : à partir de l'état final de chaque
  joueur, propage l'information vers les états passés
- [ ] **3.A.3** — EM loop : forward → backward → re-estimate hyperparams →
  forward... jusqu'à convergence
- [ ] **3.A.4** — Métrique de convergence : delta entre 2 EM iterations
- [ ] **3.A.5** — Tests : converger en < 10 itérations sur un dataset synthétique
- [ ] **3.A.6** — Comparer ratings avant/après sur prod : doit donner mêmes
  tiers, μ légèrement raffinés
- [ ] **3.A.7** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 3.A

- [ ] Tests PASS, convergence atteinte
- [ ] Replay sur prod n'introduit pas de changement abrupt de tier sur les 4 trackés (Madina reste Diamant, etc.)
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

---

## Sprint 3.B — Delta de rating dans l'historique

### Objectif
Aujourd'hui quand on écrit une ligne `match_skill_rank`, le champ
`rating_delta` est nul. Pour afficher au joueur "vous avez gagné +12 LUSR
ce match", il faudrait fetcher le rating précédent. Petit confort UX.

### Étapes

- [ ] **3.B.1** — Modifier `writeCanonicalLUSRRow` :
  - Charger le rating LUSR le plus récent avant ce match (via `match_skill_rank_latest`)
  - Calculer `rating_delta = new_rating - previous_rating`
  - Si pas de précédent : delta = nil (premier match)
- [ ] **3.B.2** — Test E2E : 2 matchs successifs pour un joueur → 2ème row a `rating_delta != nil`
- [ ] **3.B.3** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 3.B

- [ ] Tests PASS
- [ ] Vérifier en prod après quelques matchs que `rating_delta` est populé
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

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
