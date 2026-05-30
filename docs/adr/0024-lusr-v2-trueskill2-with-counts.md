# ADR 0024 — LUSR v2 : TrueSkill 2 avec observations kills/deaths (Halo Infinite)

**Statut** : Accepté (2026-05-27) — Phases 0, 1a-d, 2 (squadOffset), 3a-d livrées. **Bascule canonical effectuée en prod (2026-05-30)** : v2 est le writer du rating affiché (Stratégie C), sentinelle auto + capability `CapLUSR` en place. Phase 3e (damping EP matchs déséquilibrés) reste en backlog ; calibration tier validée empiriquement (pas de recalibration nécessaire). Cleanup du code v1 (`batchComputeLUSR`) volontairement différé tant que v2 n'a pas plusieurs jours stables (le rollback `LEVELUP_LUSR_CANONICAL=LUSR` reste donc possible).

**Branche source** : `feat/lusr-v2-phase0-metrics`

---

## Contexte

Le LUSR v1 (`internal/sync/skill_rating.go`) calcule un score Bayésien-like avec une formule composite ad-hoc (KvE, DvE, damage_efficiency, accuracy_delta, etc.) pondérée à la main. Validation Phase 0 sur 2513 observations a montré des biais structurels :

| Métrique Menke | Verdict baseline LUSR v1 |
|---|---|
| Squad effect (duo +3pp vs solo -2.6pp) | SIGNAL PRÉSENT |
| Experience effect (newbies surcôtés) | SIGNAL PRÉSENT sur 0-9 matchs |
| Kill rate prédictivité | SIGNAL FORT (+7.7pp entre kpm<0.8 et kpm≥2.0) |

Et surtout : verdict Phase 1d (replay shadow d'un TS classique propre sur les 4 joueurs trackés) — le modèle TS sans signal individuel **ne discrimine pas les coéquipiers récurrents**. Madina (forte, win rate 47.8%, mais carry stats individuelles) finit en bas du classement vs Choco/JGtm (moyens, win rate 50%) qui finissent au-dessus — l'ordre est inversé parce que les 3 jouent ensemble 40-75% du temps et leurs μ bougent en parallèle.

**Conclusion** : il faut un modèle Bayésien qui intègre le signal individuel (kills/deaths) **dans le même framework** que le skill, pas en post-traitement (Moving Target Problem de Menke).

---

## Décision

> **Implémenter TrueSkill 2 (Minka/Cleven/Zaykov, MSR 2018), Section 8 (kills/deaths comme observations Bayésiennes), via un factor graph + Expectation Propagation.**

Le LUSR v1 reste actif (rating_type='LUSR' dans match_skill_rank). LUSR v2 cohabite dans des tables séparées (`player_skill_state_v2`, `lusr_hyperparams_v2`) et un sous-package isolé (`internal/analysis/skill_v2/`), gated par le flag d'env `LEVELUP_LUSR_V2_ENABLED`. Bascule prod future via **Stratégie C (write-through aliasing)** : quand v2 deviendra canonical, son writer remplira `rating_type='LUSR'` (slot historique inchangé) — pas de migration de readers UI.

### Couches livrées

```
internal/analysis/skill_v2/                — math pure, 0 dépendance externe
  ├── gaussian.go, math.go, trueskill.go    Closed-form TS classique (Phase 1a)
  ├── trueskill_ep.go                       Wrapper EP (Phase 3b)
  └── ep/                                   Factor graph EP (Phase 3a-c)
        ├── gaussian.go, variable.go,
        ├── factor.go (Runner)
        ├── prior_factor.go, likelihood_factor.go,
        ├── sum_factor.go, greater_than_factor.go,
        ├── within_factor.go (draws),
        ├── match2team.go (orchestrateur match),
        └── count_obs.go (TS2 §8 observations)

internal/domain/skill_v2.go                 — SkillV2State, SkillV2Hyperparam
internal/migration/                         — 2 nouvelles migrations idempotentes
  ├── steps_shared_skill_v2.go              player_skill_state_v2 + lusr_hyperparams_v2
  └── steps_shared_add_participation_info.go    4 booleans ParticipationInfo

internal/platform/duckdb/skill_v2_repo.go   — Load/Upsert state + hyperparams
internal/service/skill_v2_service.go         — Orchestration externe (tests, futurs handlers)
internal/sync/skill_v2_shadow.go             — Pipeline shadow (gated env flag)

cmd/lusr_v2_phase0/                          — Métriques Menke baseline LUSR v1
cmd/lusr_v2_replay/                          — Replay offline sur les joueurs trackés
cmd/restore_one_player/                      — Outil ops (corruption DuckDB après kill)
```

### Modèle probabiliste

Pour un match à 2 équipes A/B avec N et M joueurs, et observations counts (kills, deaths) optionnelles par joueur :

```
skill_i ~ N(prior.μ, prior.σ²)                        (prior par joueur × playlist_group)
perf_i  ~ N(skill_i, β²)                              (bruit du jeu)
team_perf_A = Σ perf_{A,i}
team_perf_B = Σ perf_{B,j}
diff = team_perf_A − team_perf_B

observed:
  TeamWin   :  diff > ε    (GreaterThanFactor)
  TeamLoss  :  diff < -ε   (encodé via swap interne A↔B → diff > ε)
  TeamDraw  :  |diff| < ε  (WithinFactor)

Pour chaque (player_i, count_type) avec obs :
  expected_count_i = bias + w_p · perf_i + (w_o / M_opp) · Σ perf_opp_j
  observed_count_i ~ N(expected_count_i, v)            (PriorFactor sur expected_count_i)
```

Inférence : Expectation Propagation, runner round-robin, convergence sur max-delta < tolerance, MaxIters = 200 (bumped pour absorber les contradictions count-vs-outcome).

Après convergence, τ² ajouté aux variances posterieures pour modéliser le random walk dynamique (parité avec le closed-form).

### Hyperparamètres (Halo Infinite, valeurs initiales)

| Param | Valeur | Origine |
|---|---|---|
| μ_0 | 25 | Convention TrueSkill (Herbrich 2007) |
| σ_0 | 25/3 ≈ 8.33 | idem |
| β | σ_0/2 ≈ 4.17 | idem |
| τ | σ_0/100 ≈ 0.083 | idem |
| DrawProbability | 0.10 | empirique Halo |
| kill.Bias | 0 | calibré pour DefaultPriors |
| kill.w_p | +1.0 | TS2 paper §8 (signe positif) |
| kill.w_o | -0.5 | TS2 paper §8 (signe opposé) |
| kill.v | 25 | σ_count ≈ 5, ordre de grandeur des kill counts |
| death.Bias | +25 | **calibré pour μ_0 = 25 (compense le scale shift vs Halo 5 paper où μ_0=3)** |
| death.w_p | -1.0 | TS2 paper §8 |
| death.w_o | +0.5 | TS2 paper §8 |
| death.v | 25 | idem kill |

À ré-estimer en Phase 5 (TrueSkill Through Time batch) sur des données représentatives.

### Pourquoi un intercept (bias) pour deaths ?

Sur l'échelle native TrueSkill (μ_0 = 25, perf typique ≈ 25), la formule paper `expected_death = w_p · perf + w_o · perf_opp` donne **-12.5** typiquement (perf positives + w_p négatif). L'observation étant toujours ≥ 0, la résiduelle est systématiquement positive et pousse perf vers le bas — biais aveugle.

Le paper Halo 5 utilise μ_0 = 3 (échelle 8× plus serrée) ; la formule sans bias produit alors `expected_death ≈ 0` qui colle bien à la zone des observations (0-15). Le `max(0, ...)` du paper sert de plancher mais n'a pas besoin de jouer souvent.

Sur notre échelle, le bias = 25 ramène l'expected_death à ~12.5 pour des perfs typiques, ce qui matche la moyenne observée. Le `max(0, ...)` devient inutile pour les counts > 0.

### Stratégie multi-mode (séparation par playlist_group)

Le pair_name de chaque match est mappé vers 1 des 4 chaînes via `internal/sync/skill_config.go:GetLUSRChain` :
- `arena_slayer` (Slayer sous-modes Arena/Tactical/Assault/Community + Rumble Pit + fallback)
- `arena_objectif` (CTF, Strongholds, KotH, Total Control, etc.)
- `btb` (Big Team Battle, BTB Heavies)
- `chaos` (Fiesta, Super Fiesta, Husky Raid, Action Sack, Infection, etc.)

Matchs Ranked → CSR (calculé séparément). Matchs Firefight → PvE (pas de skill rating).

Chaque (xuid, playlist_group) a son posterior indépendant — le modèle ne corrèle PAS les groupes (Phase 4 du paper §11 non implémentée). Un joueur top arena_slayer démarre depuis priors en chaos s'il n'a jamais joué chaos.

---

## Conséquences

### Positives

- **Verdict Phase 1d résolu** : sur les 4 joueurs trackés, ordre relatif correct (Madina > Choco > JGtm > XxDaemon) après replay Phase 3d.
- **Architecture isolée** : 0 modification du code LUSR v1, cohabitation triviale, rollback = env flag.
- **Math testée** : 62 tests dans `skill_v2/*` (25 closed-form Phase 1a + 22 EP factors Phase 3a + 10 régression EP vs closed-form Phase 3b + 5 discrimination intra-squad Phase 3c).
- **Pipeline shadow** : `LEVELUP_LUSR_V2_ENABLED=1` active le calcul en parallèle de v1, sans aucun reader UI à ce stade.

### Limites et dette

1. **Calibration tier non faite** (Phase 3e backlog) — les seuils μ → Bronze/.../Onyx du cmd replay sont à l'œil. Doit être figé dans `lusr_hyperparams_v2.tier_boundary_*` après calibration empirique sur population réelle.
2. **~10% des matchs ne convergent pas** en 200 itérations EP — équipes très déséquilibrées (4v6, 5v8+). Skippés silencieusement, retentés au prochain run. Solutions possibles : damping EP, MaxIters plus grand, ou skip explicite ratio > 2:1.
3. **Truncated Gaussian non-impl** : MVP utilise Gaussien pur sur les count observations (pas le `max(0, ...)` du paper). Pour count = 0, le bias compense. Pour matchs très courts où count = 0 est fréquent : à monitorer.
4. **Pas de mode correlation (TS2 §11)** : un nouveau joueur démarre à priors par groupe, pas avec l'info inférée des autres groupes.
5. **Pas de squadOffset (TS2 §6)** : déprioritisé après Phase 3d (les counts règlent l'essentiel du carry intra-squad). Toujours utile pour la prédiction squad-vs-solo mais bloqué par l'absence de `party_id` dans l'API Halo publique — seul proxy disponible = `is_with_friends` + coéquipiers trackés (bruité).
6. **Pas de quit penalty (TS2 §9)** : la capture `ParticipationInfo` (Mini-Phase 0.5) est en place mais le facteur n'est pas implémenté. Nécessaire surtout pour Ranked, moins pour LUSR (qui filtre les DNF en amont).
7. **Cycle import sync↔service contourné par duplication** : `applyMatchToSkillV2` dans le package sync duplique la logique de `SkillV2Service.UpdateAfterMatch` parce que sync ne peut pas importer service (cycle existant via career_live_cache). À refactorer plus tard si on stabilise davantage.

---

## Migration prod (Stratégie C — write-through aliasing)

Quand le LUSR v2 sera validé pour bascule (Phase 3e + calibration + monitoring shadow N semaines), la bascule se fera en :

1. **Stopper la compute v1** : skip de l'étape 2 LUSR du `runPostSyncPipeline` si `LEVELUP_LUSR_CANONICAL=LUSR_V2`.
2. **Routage du writer v2** : le service v2 écrit dans **deux rows** de `match_skill_rank` :
   - `rating_type='LUSR'` avec la valeur v2 mappée sur la grille legacy [1000..2000] (cf. mapping_func à calibrer en Phase 3e)
   - `rating_type='LUSR_V2'` avec μ/σ natifs (audit/transparence)
3. **Readers actuels** : aucune modification — ils lisent toujours `rating_type='LUSR'` via la vue `match_skill_rank_latest`.
4. **Sentinelle** : test E2E qui vérifie qu'après un sync, chaque match récent a au moins une row `rating_type='LUSR'`. Métrique expvar `levelup.lusr_canonical_missing` pour alerting.
5. **Rollback** : `LEVELUP_LUSR_CANONICAL=LUSR` + cmd `lusr_v1_recompute_all` qui repasse `BatchComputeLUSR` sur l'historique pour écraser les rows v2-en-déguisement. Append-only → last-writer-wins via la vue `_latest`.

Cleanup final (optionnel) : `DELETE FROM match_skill_rank WHERE rating_type='LUSR_V2'` post-bascule.

---

## État as-built post-bascule (2026-05-30)

La bascule décrite ci-dessus est **effectuée**. Différences vs le plan initial :

1. **Défaut code, pas seulement env (Fix B 2026-05-30)** : `sync.DefaultLUSRModeIfUnset` (appelée au boot dans `cmd/server`) pose `LEVELUP_LUSR_V2_ENABLED=1` + `LEVELUP_LUSR_CANONICAL=LUSR_V2` si les flags sont absents du process. v2 canonical survit donc à un reset `.air.toml`/`.env.local`. Rollback explicite préservé : `LEVELUP_LUSR_CANONICAL=LUSR`.

2. **Sentinelle dual-row automatique (Sprint 2.C)** : `RunDualRowSentinel` est appelée à l'étape 2.6 de `runPostSyncPipeline` (engine_postsync.go), uniquement en mode canonical, avec timeout 30s, read-only. Toute incohérence (`OnlyLUSRV2 > 0` = match avec row audit mais sans slot LUSR) émet un `slog.ErrorContext` auto-routé vers `logs/sync.log`. Pas de notif externe (cohérent ADR 0009). Remplace l'appel CLI manuel qui n'était jamais déclenché en prod.

3. **Capability `CapLUSR` (Sprint 3.C)** : le LUSR (v1 + v2 + sentinelle) est gardé par `title.CapLUSR` (déclaré par halo_infinite dans `title.NewRegistry`), plus aucun couplage `slug == "halo_infinite"`. Helpers `slugHasLUSR` / `titleHasLUSR` / `skipIfNoLUSRCapability` (skill_v2_capability.go). `RunLUSRV2Shadow` self-gate ; `runPostSyncPipeline` gate sur `e.titleSlug`. Un titre futur sans CapLUSR → LUSR no-op silencieux pour ce titre.

4. **Readers UI → vue `_latest`** : `playerMatchesSkillRankTpl` (table Explorer/Session) et `Q22aMatchSkillRankPlayer` (carte rang) lisent `match_skill_rank_latest` (et non la table brute) — sinon la row audit `LUSR_V2` ou une row LUSR périmée fuite à l'affichage de façon non-déterministe.

5. **Réconciliation historique** : `cmd/lusr_v2_canonical_backfill --commit` lancé **une fois par joueur** (le shadow avance le watermark des coéquipiers → un run multi-joueurs ne traite que le 1er). Résiduel v1 attendu = matchs v2-inéligibles (BTB déséquilibrés `|nA-nB|>1` + FFA, cf. `isTeamImbalanceTooHigh`) ; ils gardent leur slot LUSR v1 (label jamais `LUSR_V2`).

**Reste backlog** (non bloquant) : Sprint 1.D (wiring frontend `expected_win_prob`/`rating_delta`, calculés+stockés mais non affichés) ; cleanup code v1 `batchComputeLUSR` (différé pour préserver le rollback) ; Phase 3e damping EP (couvrirait les matchs déséquilibrés).

---

## Références

- Herbrich, Minka, Graepel — TrueSkill: A Bayesian Skill Rating System (NIPS 2006)
- Minka, Cleven, Zaykov — TrueSkill 2: An improved Bayesian skill rating system (MSR 2018) ([texte extrait](.ai/trueskill2.txt))
- Menke (GDC) — Significantly Improving your Skill System with TrueSkill Through Time ([texte extrait](.ai/menke.txt))
- Moserware/Skills (C#, BSD) — implémentation de référence des facteurs EP utilisée pour valider les formules
- Rapport Phase 0 : [.ai/lusr_v2_phase0_metrics.md](../../.ai/lusr_v2_phase0_metrics.md)
- Rapport Phase 1d : [.ai/lusr_v2_phase1d_replay.md](../../.ai/lusr_v2_phase1d_replay.md)
- Rapport Phase 3d : [.ai/lusr_v2_phase3d_replay.md](../../.ai/lusr_v2_phase3d_replay.md)
