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

- [ ] **1.A.1** — Créer `internal/analysis/skill_v2/predict.go` avec :
  - `PredictTwoTeamWinProb(teamA, teamB []Gaussian, p Priors) (probA, probDraw, probB float64)`
  - Utilise la formule TrueSkill standard avec draw margin
  - 0 accès DB, 0 dépendance externe
- [ ] **1.A.2** — Créer `internal/analysis/skill_v2/predict_test.go` avec :
  - Cas équilibré (teams identiques) → probA ≈ probB ≈ 50%
  - Cas asymétrique fort (team A μ=30 vs team B μ=20) → probA > 90%
  - Cas avec gros σ (incertitude haute) → probabilités tirées vers 50% même si μ différent
  - Cas avec draw_probability > 0 → probDraw > 0
- [ ] **1.A.3** — Créer `internal/sync/skill_v2_predict_loader.go` avec :
  - `LoadPreMatchStates(ctx, repo *duckdb.SkillV2Repo, matchID string, startTime time.Time, group string) (teamA, teamB []Gaussian, error)`
  - Pour chaque participant : trouve la row `player_skill_state_v2` la plus récente avec `last_match_at < startTime`
  - Fallback `priors.NewPlayerState()` si jamais joué
  - `slog.DebugContext` pour signaler les fallbacks
- [ ] **1.A.4** — Migration `steps_player_add_expected_win_prob.go` :
  - `ALTER TABLE match_skill_rank ADD COLUMN IF NOT EXISTS expected_win_prob FLOAT`
- [ ] **1.A.5** — Étendre `domain.LUSRRatingInsert` (`internal/persist/lusr_append_only_persister.go`) :
  - Ajout champ `ExpectedWinProb *float64`
  - INSERT statement updated (10→11 placeholders)
- [ ] **1.A.6** — Wire dans `writeCanonicalLUSRRow` (`internal/sync/skill_v2_canonical.go`) :
  - Avant l'écriture, charger les pré-match states + calculer la prob
  - Stocker dans les 2 rows (LUSR + LUSR_V2)
- [ ] **1.A.7** — Tests E2E :
  - `TestPredictWinProb_StoresInCanonicalRow` — match 2v2, vérifier que la row LUSR contient un `expected_win_prob` dans [0,1]
  - `TestPredictWinProb_FallbackForFirstMatch` — joueur sans historique → utilise priors initiaux
- [ ] **1.A.8** — Métrique expvar : `levelup.lusr_v2.predictions_total` (compteur)
- [ ] **1.A.9** — Mise à jour `.ai/thought_log.md` (entrée datée)

### Definition of Done — Sprint 1.A

- [ ] `go test ./internal/analysis/skill_v2/... ./internal/sync/ ./internal/persist/` PASS
- [ ] `go vet ./...` clean
- [ ] Au moins 1 match prod vérifié : la row `match_skill_rank` post-bascule contient un `expected_win_prob` plausible (proche de 0.5 ± 0.3 pour la plupart des matchs)
- [ ] Pas de panic en cas de match avec joueur jamais vu (fallback testé)
- [ ] Entrée `.ai/thought_log.md` ajoutée
- [ ] Commit unique sur la branche, autorisé par l'utilisateur

**Date complétion** : _______________

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

- [ ] **1.B.1** — Créer `internal/analysis/skill_v2/hyperparams_load.go` avec :
  - `LoadPriorsFromHyperparams(params map[string]float64, defaultP Priors) Priors`
  - Override `DrawProbability` si `draw_probability_empirical` présent
  - Reste passé tel quel (les autres sont CountHyperparams)
  - Logique pure, testable
- [ ] **1.B.2** — Créer `LoadCountHyperparamsFromDB(params map[string]float64, defaults map[CountType]CountHyperparams) map[CountType]CountHyperparams`
  - Override `Bias` pour kill / death depuis `kill_mean_empirical` / `death_mean_empirical`
- [ ] **1.B.3** — Tests unitaires :
  - Map vide → defaults retournés intacts
  - Map avec uniquement draw_prob → seul DrawProbability change
  - Map complète → tous les overrides appliqués
- [ ] **1.B.4** — Modifier `processOneShadowMatch` (skill_v2_shadow.go) :
  - Avant `applyMatchToSkillV2`, charger les hyperparams pour `group`
  - Passer les Priors override à l'appel
  - `slog.DebugContext` "hyperparams ré-estimés appliqués" avec compte de overrides
- [ ] **1.B.5** — Refactor `applyMatchToSkillV2` pour accepter `Priors` au lieu de `priors skillv2.Priors` du shadowRunContext (qui restent les defaults)
- [ ] **1.B.6** — Test E2E :
  - `TestRunLUSRV2Shadow_UsesEmpiricalDrawProb` — seed une row hyperparam avec `draw_probability_empirical=0.5` (artificiellement haut), vérifier qu'un draw est moins surprenant pour le modèle
- [ ] **1.B.7** — Mise à jour `.ai/thought_log.md`

### Definition of Done — Sprint 1.B

- [ ] `go test ./internal/analysis/skill_v2/... ./internal/sync/` PASS
- [ ] Run manuel : `LEVELUP_LUSR_V2_ENABLED=1 ./server` → vérifier dans `logs/sync.log` qu'on voit le log "hyperparams ré-estimés appliqués" avec un count > 0 pour chaque groupe (après que TTT batch ait écrit)
- [ ] Entrée `.ai/thought_log.md`
- [ ] Commit autorisé

**Date complétion** : _______________

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
