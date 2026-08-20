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
| **1.A** | Probabilité de victoire prédite (backend uniquement) | 1/2 jour | Haute (quick win UX) |
| **1.B** | Brancher les paramètres ré-estimés | 2-3h | Haute (corrige un bug latent) |
| **1.C** | Détection escouades + correction skill | 2 jours | Haute (angle mort modèle) |
| **1.D** | Préparer le wiring API + types frontend (placement UI reporté) | 1/2 jour | Haute (sans ça 1.A et 3.B sont des données mortes) |
| **2.A** | Timeline du score au moment du quit | 1/2 jour | Moyenne |
| **2.B** | Matrice de corrélation entre modes | 1 jour | Moyenne |
| **2.C** | Sentinelle dual-row appelée après chaque sync + logs Error | 2-3h | Haute (sinon les incohérences ne sont jamais détectées) |
| **3.A** | Recalibration auto périodique propre | 1-2 jours | Basse |
| **3.B** | Delta de rating dans l'historique | 2h | Basse (confort UX) |
| **3.C** | Capability multi-titres `CapLUSR` (obligatoire) | 1/2 jour | Haute (règle projet — pas de slug-coupling) |
| **Final** | ADR consolidée + bascule canonical en prod | 1 jour | Critique |

**Total estimé : ~9 jours de dev** réparti sur N sessions.

### Pourquoi ces ajouts (2026-05-28)

- **1.D — wiring API/types frontend** : les sprints 1.A et 3.B ajoutent des colonnes (`expected_win_prob`, `rating_delta`) mais sans exposition côté API + sans types TypeScript / query hooks, rien n'est visible. Le **placement UI final est reporté** (décision produit en attente : encart sur l'historique ? overlay sur la page LUSR ? section dédiée ?), mais le wiring backend → frontend peut être préparé maintenant.
- **2.C — sentinelle automatique** : `RunDualRowSentinel` existait en CLI manuel, sans utilité prod. Hook post-sync (appel à la fin du pipeline `engine_postsync`), pas de cron séparé. Toute incohérence détectée → `slog.ErrorContext` vers `logs/sync.log` (auto-routé), pas de système de notification externe.
- **3.C — capability `CapLUSR`** : règle projet (cf. skill arch-rules). Le code actuel branche implicitement sur `halo_infinite` sans passer par `HasCapability`. Doit être corrigé **avant la bascule canonical** sinon dette qui ne se résoudra jamais.

---

## ✅ CHECKLIST UTILISATEUR (état au 2026-05-28)

> **État code** : Sprints 1.A, 1.B, 1.C, 2.A, 2.B, 3.A (wiring complet), 3.B = **livrés,
> testés, committés** sur `feat/lusr-v2-phase0-metrics` (10 commits). Build + `go vet`
> + tests verts. Rien n'est poussé ni en prod. Ce qui suit est ce qu'il TE reste.

### 🔧 À FAIRE par toi — sur le serveur (je ne peux pas le faire d'ici)

Pour basculer en prod cette semaine (Sprint Final), dans l'ordre :

1. **Peupler les hyperparams + la matrice cross-mode** : lancer `cmd/lusr_v2_ttt_batch`
   (sans `--dry-run`). Sans ça, le cross-mode utilise le scalaire 0.3 par défaut.
2. **(Optionnel mais recommandé) Activer la correction d'escouade** : lancer
   `cmd/lusr_v2_squad_estimate --dry-run` → vérifier les offsets ∈ [-2,+2] cohérents
   pour tes 4 trackés → si OK, relancer **sans** `--dry-run`. (Le flag est déjà ON par
   défaut, mais sans ce passage la correction ne fait rien.)
3. **Backfill historique** : relancer `cmd/lusr_v2_replay` (rattrape l'historique v2).
4. **Bascule** : suivre la checklist du **Sprint Final** ci-dessous (F.1→F.10) — staging
   d'abord (`LEVELUP_LUSR_CANONICAL=LUSR_V2`), 3-5 syncs, vérifs logs/sentinelle/UI,
   puis prod + surveillance 24h.

### 🤔 À DÉCIDER par toi

- **Affichage 1.A** (`expected_win_prob`, "X% de chance de gagner") : OÙ et COMMENT
  l'afficher ? (rien ne l'affiche aujourd'hui — juste en base). Cf. section 1.A.
- **Affichage 3.B** (`rating_delta`, "+12 LUSR ce match") : OÙ et COMMENT ? (idem). Cf. 3.B.
- **Squad + cross-mode ON par défaut** : ✅ **confirmé (2026-05-28)**. Les deux flags
  sont ON. Calibration à valider sur tes vraies données après la bascule (offsets squad
  via `--dry-run` d'abord, matrice cross-mode via `--smooth --dry-run`). Pour désactiver :
  `LEVELUP_LUSR_V2_SQUAD_OFFSET=0` ou `LEVELUP_LUSR_V2_MODE_COUPLING=0`.
- **3.A (TTT)** : ✅ **wiring complet livré (2026-05-28)**. Pour valider sur tes données :
  `go run -tags cgo ./apps/go-api/cmd/lusr_v2_ttt_batch --smooth --dry-run` → rapport τ estimés.
  Sans `--dry-run` pour écrire. `--write-smoothed` en option après bascule stable.

### 👤 À FAIRE par ton collègue (dev)

- **Adapter T0 (multi-titre)** : brancher `match_registry.real_start_time` au hook
  marqué en commentaire dans `quitOffsetMs` (`internal/sync/skill_v2_quit_penalty.go`).
  Aujourd'hui on utilise `start_time_utc` (début film) — correct pour Halo, à
  généraliser pour les autres titres.

### ⏭️ Reste à faire / différé (non bloquant pour la bascule)

- Wiring complet du TTT couplé (3.A) — follow-up dédié.
- Validations prod des features (offsets squad cohérents, matrice 4×4, deltas peuplés,
  win-prob plausible) — se font après la bascule en observant les vraies données.
- Nettoyage code v1 (`batchComputeLUSR`) — étape F.9, après ≥7j stables.

#### 🆕 Découvert pendant l'onboarding clone-frais (2026-05-29, post-prod)

> Surfacé en simulant un clone frais (`cmd/server/boot_dirs_test.go`). Non bloquant
> tant que LUSR v2 n'est pas en prod. À traiter quand on bascule.

- [ ] **Fix ordre migration `shared_seed_tier_boundaries_v2`** — sur DB vierge, le seed
  (statique : Bronze..Onyx hardcodés) s'exécute AVANT `shared_create_skill_v2_tables`
  car l'ordre = ordre alphabétique des fichiers et `..._seed_tier_boundaries_v2.go`
  ("seed") trie avant `..._skill_v2.go` ("skill"). Résultat : WARN
  `Table lusr_hyperparams_v2 does not exist` et seuils tier **non seedés** au boot.
  Ce seed **doit tourner immédiatement** (pas gated sur des matchs) — sinon l'affichage
  des tiers est cassé dès le 1er match. Fix recommandé : co-localiser le seed dans le
  `ApplyBackfill` de `shared_create_skill_v2_tables` (table + seed atomiques), ou renommer
  le fichier seed pour qu'il trie après `..._skill_v2.go`. Garde-rail : ajouter un cas au
  test de boot clone-frais vérifiant que `lusr_hyperparams_v2` contient les 6×4 seuils
  après migrations.

- [ ] **Ré-estimation empirique auto en post-sync** (≠ du seed statique ci-dessus) — les
  hyperparams *empiriques* (`ttt_tau_empirical`, `kill_mean_empirical`,
  `death_mean_empirical`, `draw_probability_empirical`), aujourd'hui calculés à la main via
  `cmd/lusr_v2_ttt_batch`, devraient se relancer automatiquement une fois un corpus de
  matchs suffisant atteint. Le runtime retombe proprement sur les défauts quand ils sont
  absents (`skill_v2/hyperparams_load.go`) → non bloquant. Pistes : extraire la logique
  de `cmd/lusr_v2_ttt_batch` vers `internal/` (un cmd n'est pas importable), brancher dans
  le `PostSyncRunner` (cf. `internal/api/post_sync_*.go`), déclencher au franchissement
  d'un seuil de matchs. ⚠️ Seuil : "10 matchs" (idée initiale) est **trop bas** pour une
  estimation bayésienne stable — viser plutôt quelques centaines, et décider une fois
  (au franchissement) vs périodique (ex. tous les N matchs).

---

## Sprint 1.A — Probabilité de victoire prédite

> Référence détaillée : `.ai/LUSR_V2_WIN_PROBABILITY.md`

### Objectif
Exposer, pour chaque match, "votre équipe avait X% de chance de gagner" avant
que le match commence. Permet de filtrer les défaites entre "matchs perdables"
et "matchs vraiment au-dessus", et de mettre en valeur les belles perfs sur
matchs donnés perdants.

> 📊 **AFFICHAGE À DÉCIDER (utilisateur)** : la donnée `expected_win_prob` est
> calculée et stockée par match, mais **rien ne l'affiche encore**. À toi de
> décider OÙ (page match-view ? historique ? badge ?) et COMMENT (ex. "Match
> attendu" / "Upset" / "Belle perf sur match perdable"). Tant que ce n'est pas
> décidé, c'est juste une colonne en base, invisible côté UI.

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
- [ ] Dry-run replay avec offsets actifs, pas de drift > 1.5 pt sur les solo — **différé : prod uniquement**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28. **MAJ 2026-05-28 : flag `LEVELUP_LUSR_V2_SQUAD_OFFSET` désormais ON par défaut** (décision produit). NB : aucun effet tant que `cmd/lusr_v2_squad_estimate` n'a pas peuplé d'offsets (LoadSquadOffsets vide → no-op). Donc l'activation réelle = lancer le CLI.

---

## Sprint 1.D — Wiring API + types frontend (placement UI reporté)

### Objectif
Préparer toute la plomberie pour que le frontend puisse afficher
`expected_win_prob` (Sprint 1.A) et `rating_delta` (Sprint 3.B) **sans
décider du placement UI final**. Une fois la décision produit prise plus tard,
l'intégration UI sera un patch isolé.

**Décision en attente** : où afficher la probabilité de victoire — encart
sur la timeline d'historique, overlay sur la page LUSR, badge sur la ligne
de match, ou section dédiée "match equity sur la session". À discuter
quand on aura vu plusieurs matchs en prod.

### Architecture prévue
- Côté Go : exposer les 2 champs dans le DTO existant qui sert les matchs (vraisemblablement `MatchSummary` ou `MatchHeader` dans `internal/domain/`)
- Côté TypeScript : étendre les types correspondants dans `apps/web/src/lib/api/types.ts`
- Côté MSW handler / test : maj des handlers de test pour renvoyer les nouvelles valeurs

### Étapes

- [ ] **1.D.1** — Identifier le DTO existant qui sert l'historique des matchs au frontend (probablement `MatchListEntry` ou similaire dans `internal/api/handlers/`)
- [ ] **1.D.2** — Ajouter `ExpectedWinProb *float64` et `RatingDelta *float64` au DTO Go (nullables — anciens matchs n'auront pas ces valeurs)
- [ ] **1.D.3** — Modifier le service qui construit ce DTO pour lire les valeurs depuis `match_skill_rank_latest`
- [ ] **1.D.4** — Étendre `apps/web/src/lib/api/types.ts` avec les 2 champs optionnels
- [ ] **1.D.5** — i18n FR + EN dans `apps/web/src/lib/i18n/manifests/` :
  - "Match attendu" / "Expected match"
  - "Surprise" / "Upset"
  - "Belle performance" / "Strong performance"
  - "Probabilité de victoire" / "Win probability"
  - "Variation" / "Rating change"
- [ ] **1.D.6** — Helper TypeScript `apps/web/src/features/skill/winProbCategory.ts` :
  - `categorizeWinProb(prob: number, won: boolean): 'expected' | 'upset' | 'strong-perf' | 'balanced'`
  - 1 helper pur, testable avec vitest
- [ ] **1.D.7** — Tests vitest pour les 4 catégories
- [ ] **1.D.8** — Maj handler MSW de test (`apps/web/src/test/handlers.ts`) pour renvoyer les nouvelles valeurs
- [ ] **1.D.9** — `slog.DebugContext` côté service si on charge un match sans `expected_win_prob` (= match pré-Sprint-1.A)
- [ ] **1.D.10** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 1.D

- [ ] `npm run typecheck && npm run lint && npm run test` (apps/web) PASS
- [ ] `go test ./internal/api/handlers/ ./internal/service/` PASS
- [ ] Helper `categorizeWinProb` testé sur 4 cas
- [ ] DTO API renvoie bien les 2 champs (vérifiable via curl sur un match avec ces données)
- [ ] **Note dans handoff doc** : "wiring frontend prêt, placement UI en attente de décision produit"
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

---

## Sprint 2.A — Timeline du score au moment du quit — ✅ LIVRÉ (2026-05-28)

> **Réouvert après correction utilisateur** : la donnée EST disponible via la
> timeline des frags (`killer_victim_pairs`, comme le graphe "tug of war" de la
> page match-view). Fiable à 100% sur modes non-objectifs (frags = score), signal
> suffisant sur modes à objectifs. On compte les frags cumulés par équipe et on
> regarde qui mène au moment du quit. Fallback sur l'outcome final si la timeline
> est absente (vieux matchs).
>
> ⚠️ **HOOK ADAPTER T0 (pour le collègue)** : le vrai T0 = `match_registry.real_start_time`
> (début réel après countdown, ~99% des cas) doit être obtenu via un **adapter
> titre-spécifique**. **NON implémenté ici** — on utilise `start_time_utc` (début
> film) comme repère, ce qui est correct pour Halo. Le point d'insertion est
> **marqué en commentaire** dans `quitOffsetMs` (`internal/sync/skill_v2_quit_penalty.go`) :
> c'est là que l'adapter `real_start_time` doit brancher.

### Objectif
Aujourd'hui on classe le quit "related" (équipe perdait) vs "unrelated"
(équipe gagnait) sur l'outcome FINAL du match. Si l'API Halo fournit
l'historique du score, on peut savoir SI le quitter est parti pendant que
son équipe était dominée ou pendant qu'elle dominait. C'est un signal plus
juste.

### Étapes

- [x] **2.A.1** — Source de vérité : `killer_victim_pairs` (killer_xuid + time_ms relatif film), même base que le graphe tug-of-war (`analysis.ComputeTugOfWar`).
- [x] **2.A.2** — `skill_v2/quit_context.go` (pur) : `InferQuitContext(frags []TeamFrag, quitMs, quitterTeamID) QuitContext` (Leading/Tied/Trailing) en comptant les frags cumulés ≤ quitMs.
- [x] **2.A.3** — `quitDeltaForContext(ctx)` (trailing→related modéré ; leading/tied→unrelated fort). `buildCountInputs` prend `quitTimeline` et applique le contexte par quitter (fallback `quitDeltaForTeam` si timeline absente). Loader `loadQuitTimeline` + `quitOffsetMs` (**hook adapter T0 commenté**).
- [x] **2.A.4** — Tests unitaires `quit_context_test.go` : menait / perdait / égalité / ne compte que ≤ quit / 0-0 / perspective équipe adverse.
- [x] **2.A.5** — E2E `TestRunLUSRV2Shadow_QuitContext_LeadingAtQuit` (quit en menant mais défaite → pénalité forte 2.5 vs fallback 1.0, écart 1.5). + thought_log + handoff.

### Definition of Done — Sprint 2.A

- [x] Tests PASS (skill_v2 + sync), `go vet` clean
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé
- [ ] **À FAIRE (collègue)** : brancher l'adapter `real_start_time` au hook marqué dans `quitOffsetMs` pour le multi-titre

**Date complétion** : 2026-05-28 (hook T0 adapter à compléter par le collègue)

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

**Date complétion** : 2026-05-28. **MAJ 2026-05-28 : le leak cross-mode (`LEVELUP_LUSR_V2_MODE_COUPLING`) est désormais ON par défaut** (décision produit). Tant que la matrice n'est pas calculée par le batch, le leak utilise le scalaire 0.3 (comportement Phase 4 historique).

---

## Sprint 2.C — Sentinelle dual-row après chaque sync + logs Error

### Objectif
`RunDualRowSentinel` existe en CLI manuel mais n'est appelée nulle part en
prod. Sans appel automatique, on découvrirait une incohérence
`OnlyLUSRV2 > 0` (= bug Stratégie C) uniquement en lançant la CLI à la main.
Cette tâche la hooke dans le pipeline post-sync.

**Décision** : pas de notif externe (pas de Slack/email/PagerDuty). Toute
incohérence détectée est inscrite via `slog.ErrorContext` qui sera
auto-routé vers `logs/sync.log` par le `MultiModuleHandler`. Le monitoring
prod consiste à grep `level=ERROR` dans `logs/sync.log` périodiquement.

### Étapes

- [ ] **2.C.1** — Identifier le point de hook : fin de `engine_postsync.go` après les étapes existantes (`batchComputeLUSR` skippé + `RunLUSRV2Shadow` exécuté). Doit s'exécuter SEULEMENT si `IsLUSRV2Canonical()` (sinon la table dual-row n'est pas censée exister)
- [ ] **2.C.2** — Modifier `engine_postsync.go` :
  ```go
  if IsLUSRV2Canonical() {
      report, err := RunDualRowSentinel(ctx, playerDB)
      if err != nil {
          slog.ErrorContext(ctx, "post-sync: sentinelle dual-row échouée",
              "err", err, "gamertag", e.gamertag)
      } else if report.OnlyLUSRV2 > 0 {
          slog.ErrorContext(ctx, "post-sync: sentinelle dual-row a détecté des incohérences",
              "gamertag", e.gamertag,
              "only_lusr_v2", report.OnlyLUSRV2,
              "sample", report.SampleInconsistent,
          )
      }
  }
  ```
- [ ] **2.C.3** — Vérifier que `RunDualRowSentinel` reste idempotent + read-only (déjà le cas, c'est juste un SELECT agrégé). Acceptable de l'appeler à chaque sync.
- [ ] **2.C.4** — Garde-fou perf : si le scan prend > 5s, c'est anormal. Ajouter un timeout 30s sur le context passé à `RunDualRowSentinel` pour ne pas bloquer le post-sync.
- [ ] **2.C.5** — Vérifier que les compteurs expvar utilisés par writeCanonicalLUSRRow (`canonicalWriteErrors`) sont aussi accompagnés systématiquement d'un `slog.ErrorContext` — grep dans le code, ajouter si manquant. **Règle : tout incrément de compteur "error" / "inconsistency" doit être doublé d'un slog.Error.**
- [ ] **2.C.6** — Test E2E `TestEnginePostSync_RunsDualRowSentinelWhenCanonical` :
  - Setup : sharedDB + playerDB, env canonical=on, seed 1 match avec dual-row valide
  - Exec engine_postsync
  - Vérifier dans le log capté qu'on a un INFO "sentinel dual-row terminé" (pas d'ERROR)
- [ ] **2.C.7** — Test E2E `TestEnginePostSync_LogsErrorWhenSentinelInconsistent` :
  - Setup : playerDB avec une row orpheline `LUSR_V2` sans `LUSR`
  - Exec engine_postsync
  - Vérifier qu'un ERROR a été émis avec les bons champs
- [ ] **2.C.8** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 2.C

- [ ] `go test ./internal/sync/` PASS
- [ ] Grep `canonicalWriteErrors\|dualRowInconsistencies` : chaque incrément est précédé/suivi d'un `slog.ErrorContext`
- [ ] Sync manuel en staging post-Sprint-1 (canonical=on) → `logs/sync.log` contient "sentinel dual-row terminé" mais 0 ligne "détecté des incohérences"
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

> **Livré en tant que PROTOTYPE pur** (`skill_v2/ttt.go`). Le couplage inter-joueurs
> complet (factor graph TS2 §10) + le câblage prod restent un follow-up (la partie
> 1-2j+ risquée ; comparaison prod 3.A.6 de toute façon différée). Cf. thought_log 2026-05-28.
>
> **Décisions utilisateur 2026-05-28** :
> - **Scope** : lisser uniquement le(s) joueur(s) ayant une BDD dans l'app (joueurs
>   actifs/trackés). S'il y a plusieurs joueurs avec BDD, on les fait tous (un par un).
> - **Fiabilité si limité aux actifs ?** Réponse : perte **négligeable**. Le lissage
>   travaille sur l'historique de CHAQUE joueur indépendamment ; les trackés ont le
>   plus de matchs = la meilleure matière. La seule chose perdue vs "lisser tout le
>   monde" est le partage d'info entre joueurs qui se sont affrontés (gain marginal
>   pour une poignée de joueurs, et c'est ça la grosse version coûteuse).
> - **Approche wiring restante (follow-up)** : pour chaque joueur tracké, construire
>   la série de ses μ post-match (depuis `player_skill_state_v2`) comme observations
>   `z_t`, appeler `EstimateTTT`, et décider où écrire les μ raffinés (réécriture
>   historique = à trancher ; risqué avant/juste après la bascule prod). **À planifier
>   séparément.**

- [x] **3.A.1** — Étude TS2 §10 : modèle état-espace linéaire-gaussien (random walk + observation), inférence forward (Kalman) + backward (RTS) + EM.
- [x] **3.A.2** — Passe backward : `rtsBackward` (lisseur RTS + covariance lag-one) propage l'info des états futurs vers les passés.
- [x] **3.A.3** — EM loop : `EstimateTTT` (kalmanForward → rtsBackward → ré-estime q=τ² et r) jusqu'à convergence.
- [x] **3.A.4** — Convergence sur `|Δq| + |Δr| < Tol`.
- [x] **3.A.5** — Tests synthétiques : convergence < 10 itérations ; log-vraisemblance croissante (propriété EM) ; lisseur ≤ filtre (RMSE) ; edge cases 0/1 obs.
- [x] **3.A.5b** — **Wiring complet (2026-05-28)** : `LoadStateHistory` (SkillV2Repo) + `runTTTSmoothing` (charge séries μ, appelle EstimateTTT, agrège q/r par groupe, écrit `ttt_tau_empirical` dans `lusr_hyperparams_v2`) + `LoadPriorsFromHyperparams` override Tau. Flags CLI : `--smooth` (τ-hyperparams) et `--write-smoothed` (μ lissé terminal → nouveau snapshot). Tests Tau : `TestLoadPriorsFromHyperparams_TauOverride` + `_TauInvalidIgnored` + 2 cas `AppliedHyperparamCount`.
- [ ] **3.A.6** — Comparaison ratings prod avant/après — **différé : valider τ estimé sur données réelles**
- [x] **3.A.7** — Entrée `.ai/thought_log.md` 2026-05-28.

### Definition of Done — Sprint 3.A

- [x] Tests PASS, convergence atteinte (prototype pur)
- [x] Wiring câblé : `--smooth` écrit `ttt_tau_empirical` par groupe ; `--write-smoothed` écrit μ lissé terminal ; `LoadPriorsFromHyperparams` override Tau au runtime
- [ ] Replay sur prod sans changement abrupt de tier — **différé : valider τ sur données réelles**
- [x] Entrée `.ai/thought_log.md`
- [x] Commit autorisé

**Date complétion** : 2026-05-28 (wiring complet livré ; validation prod = follow-up)

---

## Sprint 3.B — Delta de rating dans l'historique

### Objectif
Aujourd'hui quand on écrit une ligne `match_skill_rank`, le champ
`rating_delta` est nul. Pour afficher au joueur "vous avez gagné +12 LUSR
ce match", il faudrait fetcher le rating précédent. Petit confort UX.

> 📊 **AFFICHAGE À DÉCIDER (utilisateur)** : `rating_delta` est désormais peuplé
> en base, mais **rien ne l'affiche encore**. À toi de décider OÙ (ligne
> d'historique de match ? page match-view ?) et COMMENT (ex. "+12 LUSR" en vert /
> "−8 LUSR" en rouge). Tant que ce n'est pas décidé, c'est juste une colonne en base.

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

## Sprint 3.C — Capability multi-titres `CapLUSR` (obligatoire)

### Objectif
Faire passer le LUSR v2 par le système de capabilities du projet, au lieu
d'être implicitement spécifique à `halo_infinite`. Règle non négociable
(skill arch-rules) : aucun `if slug == "halo_infinite"` ne doit subsister.

C'est obligatoire AVANT la bascule canonical : si on bascule sans, on grave
dans le marbre une dette qui ne se résoudra jamais (le code marche, donc
personne ne reviendra le corriger).

### Architecture prévue
- Ajout d'une capability `CapLUSR` dans `internal/domain/title/registry.go`
- Tous les call sites du LUSR v2 vérifient `descriptor.HasCapability(title.CapLUSR)` avant de s'exécuter
- Si capability absente : dégradation gracieuse (pas de panic, pas d'erreur 500 — le LUSR v2 ne tourne juste pas pour ce titre)
- Halo Infinite déclare `CapLUSR` dans son TitleDescriptor

### Étapes

- [ ] **3.C.1** — Audit grep `halo_infinite` dans le code LUSR v2 :
  ```bash
  grep -rn "halo_infinite\|HaloInfinite" internal/sync/skill_v2_*.go internal/analysis/skill_v2/ cmd/lusr_v2_*/ cmd/backfill_quit_timestamps/
  ```
  Identifier tous les sites de couplage implicite.
- [ ] **3.C.2** — Ajouter `CapLUSR Capability = "lusr"` dans `internal/domain/title/registry.go` (ou wherever les capabilities sont déclarées)
- [ ] **3.C.3** — Le `TitleDescriptor` halo_infinite déclare `CapLUSR` dans sa liste de capabilities
- [ ] **3.C.4** — Hook dans `RunLUSRV2Shadow` :
  ```go
  desc, _ := registry.Get(ctxkeys.TitleSlug(ctx))
  if desc == nil || !desc.HasCapability(title.CapLUSR) {
      slog.DebugContext(ctx, "LUSR v2 shadow skip — capability absente",
          "title_slug", ctxkeys.TitleSlug(ctx))
      return 0, nil
  }
  ```
- [ ] **3.C.5** — Idem pour `engine_postsync` (`batchComputeLUSR` skip + canonical) : gate sur `CapLUSR`
- [ ] **3.C.6** — Idem pour les 3 CLIs (`cmd/lusr_v2_replay`, `cmd/backfill_quit_timestamps`, `cmd/lusr_v2_ttt_batch`) : flag `--title=halo_infinite` qui charge le descriptor + check capability avant de tourner
- [ ] **3.C.7** — Remplacer tous les `paths.SharedDBPath("halo_infinite")` hardcodés par la prise du titre courant via `ctxkeys.TitleSlug(ctx)`
- [ ] **3.C.8** — Test `TestRunLUSRV2Shadow_SkipsIfNoLUSRCapability` : registry avec descriptor sans CapLUSR → returns (0, nil) silencieusement
- [ ] **3.C.9** — Vérification grep : `grep -rn "halo_infinite\|HaloInfinite" internal/sync/skill_v2_*.go cmd/lusr_v2_*/` doit retourner 0 occurrence (sauf commentaires/strings d'i18n)
- [ ] **3.C.10** — Mise à jour `.ai/thought_log.md` + handoff doc

### Definition of Done — Sprint 3.C

- [ ] `grep -rn "halo_infinite" internal/sync/skill_v2_*.go internal/analysis/skill_v2/` → 0 occurrence dans le code Go (commentaires OK)
- [ ] `go test ./...` PASS, en particulier le nouveau test `TestRunLUSRV2Shadow_SkipsIfNoLUSRCapability`
- [ ] Manuel : enlever temporairement `CapLUSR` du descriptor halo_infinite → vérifier dans `logs/sync.log` que le shadow runner émet le DEBUG "skip — capability absente"
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

---

## Sprint Final — ADR consolidée + bascule canonical en prod

### Objectif
Activer `LEVELUP_LUSR_CANONICAL=LUSR_V2` en prod pour que le LUSR v2 devienne
le writer officiel du rating affiché à l'UI. Le LUSR v1 est désactivé en
même temps. À faire APRÈS Sprint 1 (les 3 quick wins) et idéalement APRÈS
Sprint 1.C (squad offset) pour ne pas montrer des ratings biaisés.

### Pré-requis avant de lancer
- [ ] Sprints **1.A + 1.B + 1.C + 1.D + 2.C + 3.C** terminés (les 3 quick wins + le wiring frontend + la sentinelle auto + la capability multi-titres). Les autres sprints (2.A / 2.B / 3.A / 3.B) peuvent attendre après bascule.
- [ ] Backfill v2 historique complet : `cmd/lusr_v2_replay` re-tourné post-Sprint-1.C (squad offsets appliqués sur l'historique)
- [ ] Vérifier `expvar /debug/vars` montre `levelup.lusr_v2.canonical_writes_total` qui monte avant la bascule (en mode shadow d'abord)
- [ ] `logs/sync.log` ne contient aucun ERROR `sentinel dual-row` sur les 24h précédant la bascule (sentinelle déjà active depuis Sprint 2.C)

### Étapes

#### Partie 1 — ADR consolidée
- [ ] **F.0a** — Créer `docs/adr/0025-lusr-v2-production-stable.md` (ou mettre à jour `0024-lusr-v2-trueskill2-with-counts.md`) qui consolide :
  - Approche TS classique + counts kills/deaths (Phase 3c)
  - Quit penalty post-EP avec primary/secondary 50% (last_leave_time-based)
  - Squad offset additif (Sprint 1.C)
  - Mode coupling cap 0.4 (scalaire ou matrice si Sprint 2.B fait)
  - Dual-row Stratégie C + sentinelle post-sync (Sprint 2.C)
  - Capability `CapLUSR` (Sprint 3.C)
  - Decision : pas de TS2 §9 littéral (cf. handoff)
- [ ] **F.0b** — Si nouvel ADR : noter dans `CLAUDE.md` la référence
- [ ] **F.0c** — Mise à jour `.ai/LUSR_V2_HANDOFF.md` → archivage `.ai/archive/` (ou suppression si l'ADR couvre tout)

#### Partie 2 — Bascule staging
- [ ] **F.1** — Staging : poser `LEVELUP_LUSR_CANONICAL=LUSR_V2` + redémarrer
- [ ] **F.2** — Trigger 3-5 syncs manuels sur joueurs trackés
- [ ] **F.3** — Vérifier dans `logs/sync.log` :
  - Pas de warn "fallback shadow" (= playerDB pas câblé)
  - Pas de warn "canonical write échoué"
  - Pas d'ERROR "sentinel dual-row a détecté des incohérences"
  - Le log "post-sync: LUSR v1 skippé" apparaît
  - Le log "sentinel dual-row terminé" apparaît à chaque sync (Sprint 2.C)
- [ ] **F.4** — Vérifier la table `match_skill_rank` du player DB :
  - Rows avec `rating_type='LUSR'` ET `rating_type='LUSR_V2'` pour les nouveaux matchs
  - Les anciens matchs ont juste `rating_type='LUSR'` (héritage v1, légitime)
- [ ] **F.5** — Vérifier dans l'UI (frontend wiring Sprint 1.D) : la page d'historique affiche les `expected_win_prob` et `rating_delta` sur les nouveaux matchs

#### Partie 3 — Bascule prod
- [ ] **F.6** — Si tout OK staging : passer en prod
- [ ] **F.7** — Monitoring 7j en prod (relire `logs/sync.log` quotidiennement) :
  - `levelup.lusr_v2.canonical_writes_total` monte
  - 0 ligne `level=ERROR` avec message contenant "canonical_write_errors" ou "dual-row a détecté des incohérences"
  - Rapports utilisateurs : pas de "mon rating a sauté de 200 points"

#### Partie 4 — Nettoyage post-bascule
- [ ] **F.8** — Une fois stable 7j : supprimer `batchComputeLUSR` v1 et le branchement `IsLUSRV2Canonical()` (le flag devient permanent ON)
- [ ] **F.9** — Supprimer aussi la partie shadow-fallback dans `RunLUSRV2Shadow` (le code "canonical demandé mais playerDB nil → fallback shadow" devient inutile)
- [ ] **F.10** — Mise à jour `.ai/thought_log.md`

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
1. Sprint 1.A (1/2j) — backend probabilité de victoire
2. Sprint 1.B (2-3h) — branche les paramètres ré-estimés
3. Sprint 1.C (2j) — squad offsets
4. Sprint 1.D (1/2j) — wiring API + types frontend (peut être en parallèle de 1.C)
5. Sprint 2.C (2-3h) — sentinelle auto post-sync
6. Sprint 3.C (1/2j) — capability `CapLUSR` obligatoire
7. Sprint Final (1j) — ADR consolidée + bascule
8. Sprints 2.A / 2.B / 3.A / 3.B au fil de l'eau, sans urgence

**Si on doit livrer "à minima"** pour bascule canonical : 1.A + 1.B + 1.C +
1.D + 2.C + 3.C + Final = **~5 jours de dev** pour avoir une bascule prod
propre et conforme aux règles projet (capability + sentinelle active +
frontend prêt). Les améliorations modèle (2.A, 2.B, 3.A) et le confort UX
(3.B) viennent ensuite.
