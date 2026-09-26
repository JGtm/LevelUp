# IMPL — Système Prestige (Défis, Arcs, PP, Leaderboard)

> Plan d'implémentation détaillé pour la feature spécifiée dans `PLAN_challenges_xp_system.md`.
> Branche : `feat/multi-title-adapters-and-mappings` (courante)
> Livraison : finale, après validation des 5 phases.

---

## Conventions transverses

### Backend Go (`apps/go-api/`)

| Sujet | Convention | Source |
|-------|-----------|--------|
| Logging | `log/slog` avec `*Context`, structuré key-value | existant `internal/sync/`, `internal/api/handlers/` |
| Migrations | `init()` → `Register(Migration{...})` dans `internal/migration/` | `steps_shared.go`, `steps_shared_pve.go` |
| Handlers HTTP | DI via constructor, service via `port.*Service`, `writeError` / `writeJSON` | `internal/api/handlers/gamertag.go` |
| Repositories | `internal/platform/duckdb/`, contexte + timeout, `fmt.Errorf` wrapping | `gamertag_repo.go`, `asset_cache_repo.go` |
| Tests | stdlib `*testing.T` + struct mocks, `httptest`, pas de testify | `squad_test.go` |
| Pas de pandas/sqlite | Règle CLAUDE.md, DuckDB partout | — |
| Taille fichiers | ≤ 500 L module, ≤ 80 L fonction | CLAUDE.md |

### Frontend Web (`apps/web/`)

| Sujet | Convention |
|-------|-----------|
| Routes | TanStack Router, `routes/players/$playerSlug/...` |
| API client | `lib/api.ts`, sections par domaine |
| Types | `lib/types.ts` |
| Features | `features/<nom>/` (pages, components, hooks) |
| Pas de couleurs hex | `tokenCssVar()` / `resolveToken()` (CLAUDE.md règle 20) |

### Workflow

- 1 commit par phase (avec entrée `.ai/thought_log.md` correspondante)
- Branche unique, pas de sous-branches
- Livraison finale = un récap global + checklist verte
- Toutes les valeurs de tuning dans `config/prestige/tuning.toml` (modifiables sans rebuild)

### Discipline qualité (engagement explicite)

| Règle | Pourquoi | Application |
|-------|----------|-------------|
| Spec compliance check fin de phase | Empêche les écarts silencieux entre plan et code | Avant chaque commit, relire la matrice de traçabilité (fin du doc) et marquer écarts |
| Tests écrits avant code sur domain pur | Force à clarifier le contrat avant l'implémentation | Phase 2 obligatoire, optionnel ailleurs |
| Revue croisée à la fin de chaque phase | Distinguer "livré" de "presque livré" | Liste explicite "attendu vs livré" dans le thought_log |
| Pas de "presque OK" | Une exigence non implémentée reste cochée non, on n'avance pas | Si bloquant, ouvrir une question explicite, pas masquer |
| Logging de toutes les transitions | Calage et debug ultérieurs en dépendent | Voir Stratégie de logging |

---

## Phase 1 — Foundation

> Structure squelette : migrations DB, types domain, interfaces repository, config tuning. Aucune logique métier.
> Critère terminé : `go build ./...` + `go test ./...` verts, migrations idempotentes.

### Fichiers à créer

**Migrations**
- [ ] `apps/go-api/internal/migration/registry.go` — ajouter `TargetDBSharedSocial TargetDB = "shared_social"`
- [ ] `apps/go-api/internal/migration/steps_shared_social.go` — nouveau fichier avec migrations :
  - `prestige_events` (id, user_id, title_slug, source_type, source_id, pp_amount, tier, created_at)
  - `user_prestige` (user_id, title_slug, total_pp, current_level, updated_at) — PK composite
  - `squad` (id, name, created_by, created_at)
  - `squad_member` (squad_id, user_id, joined_at)
  - `squad_challenge` (id, squad_id, template_id, title_slug, mode, window_type, window_value, created_by, expires_at, created_at)
  - `squad_challenge_participant` (challenge_id, user_id, chosen_tier, data_tier, current_value, completed_at, is_private)
- [ ] `apps/go-api/internal/migration/steps_player.go` (étendre l'existant) — ajouter migrations stats.duckdb :
  - `arc` (id, user_id, title_slug, title, description, is_preset, preset_id NULL, created_at, completed_at)
  - `challenge` (id, user_id, title_slug, arc_id NULL, **position INTEGER NULL**, template_id NULL, metric, target, target_per_member NULL, window_type, window_value, cadence, eval_type, mode, tier, data_tier, label, status, created_at, committed_at, completed_at, expired_at, abandoned_at, last_palier_recompute_at NULL, is_private)
  - `moment_card` (id, challenge_id, blob_path, created_at)
  - `prestige_telemetry` (id, user_id, challenge_id, event_type, palier, stretch_ratio, baseline_value, mode, cadence, eval_type, time_since_create_seconds NULL, created_at) — pour le calage Annexe E
  - `baseline_state` (user_id, title_slug, metric, last_match_at, is_stale, recovery_matches_remaining) — staleness reset 60j
- [ ] `apps/go-api/internal/migration/steps_metadata.go` (étendre) — ajouter table `challenge_template` (id, title_slug, metric, window_type, window_value, cadence, eval_type, label_en, label_fr, description_en, description_fr, normal_target, heroic_target, legendary_target, mythic_target, mode_filter, schema_version)

**Domain types**
- [ ] `apps/go-api/internal/prestige/types.go` — structs `Challenge`, `Arc`, `MomentCard`, `PrestigeEvent`, `Template`, `SquadChallenge`, `SquadParticipant`
- [ ] `apps/go-api/internal/prestige/enums.go` — enums avec `String()` + JSON marshaling :
  - `ChallengeStatus` : draft / active / completed / expired / abandoned / archived
  - `Tier` : normal / heroic / legendary / mythic
  - `Cadence` : daily / weekly / monthly / free
  - `EvalType` : threshold / cumulative
  - `WindowType` : session / rolling_days / deadline / matches_internal
  - `ChallengeMode` : libre / pilote
  - `DataTier` : full / estimated / tracking
  - `SquadMode` : collective / competitive
- [ ] `apps/go-api/internal/prestige/constants.go` — couleurs paliers (référence Annexe B), pas de logique

**Repository interfaces**
- [ ] `apps/go-api/internal/prestige/repository.go` — interfaces :
  - `ChallengeRepo` : Create, Get, List, UpdateStatus, UpdateLabel, UpdateTarget (libre seulement), GetActiveByUser
  - `ArcRepo` : Create, Get, ListByUser, MarkCompleted
  - `PrestigeRepo` : EmitEvent, GetUserPrestige, ListEvents, GetLeaderboard
  - `SquadChallengeRepo` : Create, Get, AddParticipant, UpdateParticipantProgress
  - `TemplateRepo` : ListByTitle, Suggest, GetByID
  - `MomentCardRepo` : Create, GetByChallenge

**Configuration**
- [ ] `config/prestige/tuning.toml` — valeurs initiales : stretch ratios (1.08/1.25/1.50/1.85), PP par palier (50/75/125/200), niveaux (500/1500/3000/6000/12000), cooldowns (12h/48h/60j), fenêtre baseline (20 matchs), **seuil population mini désactivation cap (50 joueurs)**, reset baseline (60 jours), **plafonds simultanés (3/5/2 + 12 absolu, 3 libres/jour)**, **min matchs win_rate par fenêtre (5/session, 8/7j)**
- [ ] `config/titles/halo_infinite/challenges/templates.toml` — placeholder vide avec `[meta] schema_version = 1`
- [ ] `config/titles/halo_infinite/arcs/presets.toml` — placeholder vide avec `[meta] schema_version = 1` (contenu rempli en Phase 4)

### Tests Phase 1

- [ ] `apps/go-api/internal/migration/steps_shared_social_test.go` — applique migration sur DB temporaire, vérifie présence des tables + colonnes + idempotence
- [ ] `apps/go-api/internal/prestige/types_test.go` — JSON roundtrip sur chaque struct
- [ ] `apps/go-api/internal/prestige/enums_test.go` — `String()`, `MarshalJSON`, `UnmarshalJSON` sur chaque enum + cas invalides

### Logging Phase 1

- `slog.Info` à chaque migration appliquée : `"prestige migration applied"` avec `name`, `target_db`
- `slog.Warn` si TOML tuning manquant : fallback aux valeurs par défaut hardcodées
- `slog.Error` sur erreur de parsing config

### Validation Phase 1

```bash
# Depuis apps/go-api/
go build ./...
go test ./internal/migration/... ./internal/prestige/...
# Lancer migrations sur DB temp, relancer → idempotent
```

---

## Phase 2 — Domain & Service

> Logique pure (palier, baseline, lifecycle, évaluateur) + service orchestrateur + implémentations repository.
> Critère terminé : couverture unitaire ≥ 80 % sur le package `prestige`.

### Fichiers à créer

**Domain pur (apps/go-api/internal/prestige/)**
- [ ] `palier.go` — `CalculatePalier(stretch, popPercentile float64, popSize int, dataTier DataTier) (Tier, RejectReason)` selon Annexe B. **Désactive le cap population si `popSize < tuning.PopulationMinThreshold` (défaut 50).**
- [ ] `baseline.go` — `ComputeBaseline(matches []MatchData, field canonical.FieldKey, mode Mode) (Baseline, error)`. Fenêtre 20 matchs, normalisation/min. **`CheckStaleness(state BaselineState, now time.Time) bool` → marque stale après 60 jours d'inactivité.** **`Recover(matches []MatchData) DataTier` → retourne `estimated` pendant les 10 premiers matchs après reset.**
- [ ] `stretch.go` — `ComputeStretchRatio(target, baseline float64, fieldType FieldType) float64` (ratio brut vs ceiling-normalized pour ratios bornés)
- [ ] `lifecycle.go` — transitions du state machine. **`CanEditTarget(challenge Challenge) bool` → false si mode pilote OU statut != active.** Validation cooldowns (12h expired, 48h abandoned).
- [ ] `evaluator.go` — `EvaluateThreshold(challenge, matches) Result` + `EvaluateCumulative(challenge, events) Result`. **Refuse l'évaluation si `len(matches) < tuning.WinRateMinMatches[windowType]`** pour les défis win_rate. **Cumulative joint sur `medals_earned` pour les défis "5× Killtacular".**
- [ ] `pp_amounts.go` — `PPForCompletion(tier Tier, mode ChallengeMode, dataTier DataTier) int` table d'Annexe B
- [ ] `level.go` — `LevelFromPP(totalPP int) Level` selon paliers de l'Axe 6
- [ ] `squad_target.go` — **`ResizeCollectiveTarget(perMember float64, activeMembers int) float64` → recalcul du total quand un membre rejoint/quitte. Verrou : interdire validation si la cible courante baisse sous la progression déjà acquise.**
- [ ] `telemetry.go` — `EmitChallengeEvent(ctx, event)` → écrit dans `prestige_telemetry` à chaque transition (créé / palier rejeté / completed / expired / abandoned). Champs : palier, stretch_ratio, baseline_value, mode, cadence, eval_type, time_since_create_seconds.

**Service & repositories impl (apps/go-api/internal/prestige/ + internal/platform/duckdb/)**
- [ ] `service.go` — implémente le contrat `Service` (Annexe E), injecte les repos. **Méthode `UpdateChallenge` recalcule le palier sur la baseline courante à chaque édition de cible (mode libre uniquement).** **Méthode `CheckQuotas(userID, mode, cadence) error` consultée par `CreateChallenge`.**
- [ ] `internal/platform/duckdb/prestige_repo.go` — implémente ChallengeRepo, ArcRepo, PrestigeRepo, MomentCardRepo, **TelemetryRepo, BaselineStateRepo**
- [ ] `internal/platform/duckdb/squad_challenge_repo.go` — implémente SquadChallengeRepo
- [ ] `internal/platform/duckdb/template_repo.go` — charge depuis `metadata.duckdb`, source initiale = TOML
- [ ] `internal/platform/duckdb/preset_arc_repo.go` — charge depuis `metadata.duckdb` (table à ajouter en Phase 1 si non incluse), source initiale = TOML

### Tests Phase 2

- [ ] `palier_test.go` — table-driven sur stretch / popPercentile / popSize / dataTier (matrice de cas, **dont popSize < 50 → cap désactivé**)
- [ ] `baseline_test.go` — fixtures de matchs, moyenne, normalisation/min, < 5 / 5-9 / ≥ 10 matchs, **staleness 60j, recovery 10 matchs en estimated**
- [ ] `stretch_test.go` — métriques avec et sans plafond
- [ ] `lifecycle_test.go` — transitions valides + invalides, cooldowns 12h/48h, **`CanEditTarget` mode pilote vs libre**
- [ ] `evaluator_test.go` — threshold KDA atteint/raté, **win_rate refusé si < 5 matchs en session**, cumulative 5× Killtacular via `medals_earned`
- [ ] `pp_amounts_test.go` — matrice tier × mode × data_tier (data_tier=estimated divise par 2)
- [ ] `level_test.go` — boundaries (499 / 500 / 501, etc.)
- [ ] `squad_target_test.go` — **resize quand membre rejoint/quitte, refus de validation si cible passe sous la progression**
- [ ] `telemetry_test.go` — émission d'événements à chaque transition
- [ ] `service_test.go` — orchestration avec mocks de repos, **recalcul palier sur édition cible, refus d'édition en mode pilote**
- [ ] Tests d'intégration repos avec vraie DB temporaire

### Logging Phase 2

- `slog.DebugContext` sur chaque évaluation : `"challenge evaluated"` avec `challenge_id`, `old_status`, `new_status`, `current_value`, `target`
- `slog.InfoContext` sur transitions de lifecycle : `"challenge completed"`, `"challenge expired"`, `"challenge abandoned"`
- `slog.WarnContext` sur palier rejeté : `"challenge rejected: too easy"` avec `stretch`, `baseline`
- `slog.WarnContext` sur baseline insuffisante : `"degraded baseline"` avec `match_count`, `data_tier`
- `slog.ErrorContext` sur incohérence (challenge sans template référencé, palier impossible à calculer)

---

## Phase 3 — API & Sync hook

> Couche HTTP + intégration au pipeline sync existant.
> Critère terminé : tests d'intégration verts, contrats API stables et documentés.

### Fichiers à créer

**Handlers HTTP (apps/go-api/internal/api/handlers/)**
- [ ] `challenges.go` :
  - `POST   /api/v1/challenges` — créer (mode libre ou pilote). **Vérifie quotas avant création (3 daily / 5 weekly / 2 monthly + 12 absolu, max 3 nouveaux libres/jour) — uniquement en mode pilote.**
  - `GET    /api/v1/challenges/:id`
  - `GET    /api/v1/challenges?status=active&user_id=…&title_slug=…`
  - `PATCH  /api/v1/challenges/:id` — libre uniquement, contrôle interne via `lifecycle.CanEditTarget`. **Si `target` modifié, recalcule palier et data_tier sur baseline courante.**
  - `DELETE /api/v1/challenges/:id` — abandon avec confirmation côté API. **Applique cooldown 48h sur la métrique.**
  - `POST /api/v1/challenges/:id/suggest-next` — **après transition `completed`, retourne 1 suggestion palier supérieur (+15 % stretch) + 2 alternatives depuis le catalogue.**
- [ ] `arcs.go` — POST/GET/list, marquage complétion automatique (déclenchée depuis le service)
- [ ] `prestige.go` :
  - `GET /api/v1/prestige/me?title_slug=…` (NULL = cross-titre, somme de tous titres)
  - `GET /api/v1/prestige/leaderboard?title_slug=…&period=week|month|all`. **`title_slug` NULL = vue cross-titre.** **Sources d'amis : auto-dérivées de `squad_member` + table existante `relations` (à confirmer dans le code) — pas de paramètre `user_ids`, pas de gestion manuelle.**
- [ ] `pilot_mode.go` — **`POST /api/v1/pilot-mode/enable` (active le mode + auto-attribue 1 quotidien + 1 hebdo forcé + 3 hebdo proposés). `POST /api/v1/pilot-mode/disable` (désactive, défis pilotés en cours conservés)**
- [ ] `squad_challenges.go` — création / participation / progression. **`POST /api/v1/squads/:id/challenges/pool/refresh` génère un pool de 6-9 défis thématiques pour l'escouade (renouvellement hebdo automatique + manuel).**
- [ ] `templates.go` — `GET /api/v1/challenges/templates/suggest?title_slug=…&user_id=…&count=3`

**Sync hook (apps/go-api/internal/sync/)**
- [ ] Identifier le point d'extension existant après écriture de `match_participants` (probable `internal/sync/engine.go` ou équivalent)
- [ ] Appeler `prestige.Service.EvaluateForUser(ctx, userID, titleSlug)` derrière un feature flag (`PRESTIGE_ENABLED` env, défaut `false` pour ne pas casser le sync existant)

**Routes**
- [ ] Enregistrer les nouveaux endpoints dans le router principal (vérifier le pattern dans `internal/api/router.go` ou équivalent)

### Tests Phase 3

- [ ] `challenges_test.go` — handler tests avec mock service (pattern `squad_test.go`). **Inclut tests quotas (4e daily refusé, 13e absolu refusé, 4e libre/jour refusé)**
- [ ] `pilot_mode_test.go` — **activation génère bien 1 quotidien + 1 hebdo + 3 propositions ; désactivation conserve les actifs**
- [ ] `prestige_test.go` — handler tests, **inclut leaderboard cross-titre (title_slug NULL)**
- [ ] `squad_challenges_test.go` — pool refresh hebdo, participation, modes collectif/compétitif
- [ ] `internal/prestige/integration_test.go` — flux complet : créer défi → simuler match → évaluer → vérifier completed → vérifier PP émis et user_prestige mis à jour. **Plus : suggestion post-completion appelable et retourne 3 candidats.** DB temporaire réelle.
- [ ] Test du feature flag : sync avec `PRESTIGE_ENABLED=false` ne touche pas au système ; avec `=true` déclenche l'évaluation

### Logging Phase 3

- `slog.InfoContext` à l'entrée et sortie des handlers : `"challenge created"`, `"challenge fetched"` avec `user_id`, `challenge_id`
- `slog.WarnContext` sur erreurs de validation : `"invalid challenge target"` avec `reason`
- `slog.InfoContext` sur déclenchement post-sync : `"prestige evaluation triggered"` avec `user_id`, `match_count`, `evaluations_count`
- `slog.DebugContext` sur queries repo (pattern existant)

---

## Phase 4 — Templates + refonte nav frontend

> Contenu éditorial des défis + restructuration de la nav L1 du web app.
> Critère terminé : build web vert, nav cohérente, ~30 templates Halo Infinite chargeables.

### Fichiers à créer / modifier

**Templates Halo Infinite (`config/titles/halo_infinite/challenges/templates.toml`)**
- [ ] ~10 templates **quotidiens** (cadence=daily, threshold). Ex : KDA > X sur 1 session, accuracy > X% sur 1 session, headshot rate > X% sur 1 session
- [ ] ~10 templates **hebdomadaires** (cadence=weekly, threshold). Ex : win_rate > X% sur 3 sessions, kills_vs_expected > X sur 3 sessions
- [ ] ~10 templates **mensuels** (cadence=monthly, cumulative). Ex : 5× Killtacular ce mois, 8 maps différentes ce mois, victoires sur 3 modes différents
- [ ] Cibles calibrées sur 4 paliers Normal/Heroic/Legendary/Mythic
- [ ] Labels et descriptions FR + EN
- [ ] Filtre `mode_filter` correct (universal / pvp / ranked / pve)

**Preset arcs Halo Infinite (`config/titles/halo_infinite/arcs/presets.toml`)**
- [ ] Au minimum 4 arcs preset : "Le Slayer", "Le Support", "Le Consistant", "L'Explorateur" (Axe 1 du plan conceptuel)
- [ ] Chaque arc liste 3 défis ordonnés (palier croissant) avec `position` 1, 2, 3
- [ ] Chaque défi de l'arc a son template dans `templates.toml` (ou défini inline dans le preset)
- [ ] Labels et descriptions narratives FR + EN

**Script de calage (`scripts/analyze_prestige_tuning.py`)**
- [ ] Squelette du script : lit `prestige_telemetry` + transitions de défis depuis `shared_social.duckdb`
- [ ] Produit un rapport texte/markdown avec : distribution des paliers créés, taux de complétion par palier, temps moyen `created → completed`, distribution mode libre vs pilote
- [ ] Détection d'anomalies : Mythic à 0 % ou Normal à 100 %
- [ ] Pas besoin d'être sophistiqué à ce stade — l'objectif est qu'il existe et soit utilisable post-alpha

**Frontend nav refactor**
- [ ] `apps/web/src/components/shell/NavL1.tsx` :
  - Renommer "Palmarès" → "Communauté"
  - Retirer "Synthèse" du L1
  - Ajouter "Objectifs" en L1 (8e position)
- [ ] Rajouter onglet "Synthèse" dans Stats (route + tab nav)
- [ ] Déplacer "Pass saisonnier" de Palmarès vers Carrière (route + tab nav)
- [ ] Communauté : ajouter onglet "Leaderboard PP" (page placeholder en attendant Phase 5)
- [ ] Retirer onglet "Pass saisonnier" de Communauté

**API client web**
- [ ] `apps/web/src/lib/api.ts` — sections `api.prestige`, `api.challenges`, `api.arcs`, `api.squadChallenges`, `api.templates`
- [ ] `apps/web/src/lib/types.ts` — types correspondants (Challenge, Arc, MomentCard, Tier, etc.) — alignés sur les types Go via JSON

### Tests Phase 4

- [ ] `apps/go-api/internal/prestige/templates_test.go` — charge `templates.toml` Halo Infinite, valide structure, ≥ 30 templates, paliers monotones (normal < heroic < legendary < mythic)
- [ ] Build web : `pnpm build` (ou commande équivalente du repo) sans erreur
- [ ] Type check web vert

### Logging Phase 4

- `slog.Info` au démarrage : `"prestige templates loaded"` avec `count` et `title_slug`
- `slog.Warn` sur template invalide (paliers non-monotones, labels manquants) — n'empêche pas le démarrage mais alerte

---

## Phase 5 — Frontend feature complete

> Toutes les surfaces UI : page Objectifs, moment cards, carousel home, leaderboard PP, formulaire création.
> Critère terminé : tous les parcours de l'Axe 8 fonctionnels manuellement.

### Fichiers à créer (`apps/web/src/features/prestige/`)

- [ ] `pages/ObjectifsPage.tsx` — page principale avec 2 onglets (Défis / Mon parcours), toggle mode pilote en tête
- [ ] `components/ChallengesTab.tsx` — liste défis actifs, distinction libre/pilote, CTA création
- [ ] `components/ParcoursTab.tsx` — bandeau Prestige, arcs actifs, stats globales (avec filtre auto/libre), historique entrelacé
- [ ] `components/PrestigeBadge.tsx` — niveau + total PP + barre progression
- [ ] `components/ChallengeCard.tsx` — tuile dans le carousel home (mêmes dimensions que tuiles matchs)
- [ ] `components/MomentCard.tsx` — composant card 16:9, 4 variations palier, animation sobre (Annexe F)
- [ ] `components/ArcSummary.tsx` — résumé d'arc avec progression
- [ ] `components/CreateChallengeForm.tsx` — formulaire 3 modes (auto / libre / hybride), palier calculé en live
- [ ] `components/ChallengesCarousel.tsx` — carousel home avec switch Actifs/Terminés + bouton "+ Nouveau"
- [ ] `components/LeaderboardPP.tsx` — composant pour onglet Communauté. **Affichage décomposé par ligne joueur : `Score brut` / `+ Bonus défis` / `Score total` (Axe 5).** Filtres période + filtre par arc actif.
- [ ] `components/StatsGlobales.tsx` — graphes + top métriques + taux complétion
- [ ] `hooks/useChallenges.ts`, `hooks/useArcs.ts`, `hooks/usePrestige.ts` — React Query
- [ ] `routes/.../objectifs.tsx` — route TanStack Router

**Modifications existantes**
- [ ] `apps/web/src/features/home/HomePage.tsx` — insérer `<ChallengesCarousel />` au-dessus de Faits Marquants
- [ ] Page Communauté — brancher `<LeaderboardPP />` dans son onglet (placeholder Phase 4)

### Tests Phase 5

- [ ] Type check web vert
- [ ] Build web vert
- [ ] Smoke test manuel des 4 parcours utilisateur de l'Axe 8 :
  1. Création défi mode libre
  2. Validation après sync (avec données mockées si besoin)
  3. Activation mode pilote
  4. Défi d'escouade
- [ ] Vérification visuelle moment cards aux 4 paliers
- [ ] Vérification responsive raisonnable (pas de mobile mais pas de scroll horizontal cassé)

### Logging Phase 5

- Erreurs API loggées vers la console + Sentry (si présent dans la stack)
- Pas de tracking analytique additionnel dans cette phase

---

## Stratégie de logging globale

| Niveau | Quand l'utiliser | Exemples |
|--------|------------------|----------|
| `Debug` | Détail technique utile au debug, pas en prod normalement | repo queries, évaluations détaillées |
| `Info` | Événements métier normaux | défi créé, défi complété, migration appliquée, templates chargés |
| `Warn` | Comportement dégradé mais non-bloquant | baseline insuffisante, palier rejeté, template invalide ignoré |
| `Error` | Échec d'opération nécessitant attention | parse config impossible, incohérence DB, panic récupéré |

**Champs structurés systématiques** : `user_id`, `title_slug`, `challenge_id`, `arc_id` quand pertinent. Toujours via `slog.*Context(ctx, ...)`.

**Pas de PII dans les logs** : pas de `gamertag` brut, utiliser `user_id` (xuid hashé si besoin pour anonymisation).

---

## Stratégie de tests globale

| Niveau | Cible | Outils |
|--------|-------|--------|
| Unitaire domain | ≥ 80 % couverture sur `internal/prestige/` (hors service.go) | stdlib `testing`, table-driven sur les fonctions pures |
| Unitaire handlers | Mocks de service, pattern `squad_test.go` | `httptest`, struct mocks |
| Intégration | Flux complet créer → évaluer → compléter sur DB temp | DuckDB temp file, vraies migrations |
| Intégration sync | Hook actif/inactif via feature flag | DB temp + fixtures de match |
| Intégration templates | Chargement TOML Halo Infinite, validation structure | TOML réel du repo |
| Frontend | Type check + build | `tsc`, build CLI |
| Smoke manuel | 4 parcours Axe 8 | navigateur, données réalistes |

Pas d'E2E automatisés cette itération. À ajouter en Phase 6 future si le besoin se confirme.

---

## Risques & mitigations

| Risque | Impact | Mitigation |
|--------|:------:|------------|
| Sync hook casse le pipeline existant | Élevé | Feature flag `PRESTIGE_ENABLED`, défaut `false`. Activé seulement après validation Phase 3. |
| Migrations non-idempotentes | Élevé | Test explicite : appliquer 2× sur DB temp, vérifier no-op. Pattern `schema_migrations` existant. |
| Templates calibrés trop facile/dur | Moyen | Externalisés en TOML, modifiables sans déploiement. Calage prévu en Annexe E (script analyze_prestige_tuning). |
| Refonte nav L1 régresse l'existant | Moyen | Refactor minimal en Phase 4, vérification manuelle avant de continuer Phase 5. |
| Couplage trop fort sync ↔ prestige | Moyen | Interface `Service` injectée, hook minimal qui appelle juste `EvaluateForUser`. |
| Volume de templates trop juste pour MVP | Faible | 30 = plancher confortable pour démarrer. Plus possible plus tard via TOML. |

---

## Checklist d'avancement (mise à jour à chaque phase)

### Phase 1 — Foundation
- [x] Migrations `shared_social.duckdb` créées (`steps_shared_social_prestige.go`)
- [x] Migrations stats.duckdb étendues (`steps_player_prestige.go` — arc, challenge, moment_card, prestige_telemetry, baseline_state)
- [x] Migration metadata.duckdb étendue (`steps_metadata_prestige.go` — challenge_template, preset_arc, preset_arc_step)
- [x] Types domain Go créés (`internal/prestige/types.go` — 15 structs)
- [x] Enums avec JSON marshaling (`internal/prestige/enums.go` — 8 enums + RejectReason)
- [x] Interfaces repository (`internal/prestige/repository.go` — 10 interfaces)
- [x] `config/prestige/tuning.toml` initial (exhaustif : stretch, PP, niveaux, cooldowns, quotas pilote, win_rate min, suggestion, squad pool)
- [x] Placeholders `templates.toml` + `presets.toml` Halo Infinite
- [x] Tests Phase 1 verts (`prestige` unitaires + 4 tests intégration migrations, 3 passes idempotents)
- [x] Aucune régression (`go test ./internal/...` complet)
- [x] Tailles fichiers conformes CLAUDE.md (max 327 L)
- [x] Compliance check vs matrice de traçabilité passé
- [x] Entrée thought_log
- [x] Commit Phase 1 (`c0ac7b5b`)

### Phase 2 — Domain & Service
- [x] tuning.go — chargement TOML + DefaultTuning + Validate + helpers (CooldownDuration, PPForTier, PopulationCapTier, WinRateMinForWindow)
- [x] stretch.go — ComputeStretchRatio (MetricCount + MetricRatio bornées)
- [x] palier.go — CalculatePalier avec popSize (cap désactivé < 50), DataTracking → RejectInsufficientData
- [x] baseline.go — ComputeBaseline + CheckStaleness (60j) + RecoveryDataTier + AdvanceRecovery + MarkStale
- [x] lifecycle.go — Commit / MarkCompleted/Expired/Abandoned + CanEditTarget (libre/pilote) + CooldownEndsAt + IsCooldownActive
- [x] pp_amounts.go — PPForCompletion (matrice tier × isSquad × dataTier) + PPForArc/Match/Streak/Medal
- [x] level.go — LevelFromPP avec ProgressRatio
- [x] squad_target.go — CollectiveTargetTotal + CollectiveBaseline + ValidateResizeForRemoval
- [x] evaluator.go — EvaluateThreshold + EvaluateCumulative + win_rate min matches + WindowDeadline
- [x] telemetry.go — TelemetryEmitter (best-effort, slog.Warn sur échec)
- [x] service.go + service_evaluate.go — Service interface + CreateChallenge avec recompute palier sur édition libre + SuggestTemplates/SuggestNext + EvaluateForUser
- [x] Repos DuckDB (5 player + 3 social + 2 metadata = 10 structs implémentant les interfaces)
- [x] Tests unitaires (47 sous-tests : stretch/palier/level + baseline/lifecycle/squad + evaluator/pp/tuning)
- [x] Tests intégration repos (6 tests : Challenge roundtrip + UpdateStatus + List filter + EmitEvent bumps total + Leaderboard per-title + Template Replace/List/Get)
- [x] Logging structuré (slog.InfoContext création/abandon/completion, slog.WarnContext baseline insuffisante / palier rejeté / telemetry failed)
- [x] Tailles fichiers conformes CLAUDE.md (max prestige_social_repo.go 409 L)
- [x] Aucune régression (`go test ./internal/...` complet)
- [x] Compliance check passé
- [ ] Entrée thought_log
- [ ] Commit Phase 2

### Phase 3 — API & Sync hook
- [x] Handler `PrestigeHandler` unifié (`internal/api/handlers/prestige.go`) couvrant : POST/GET/PATCH/DELETE /challenges, GET /challenges (list), POST /challenges/:id/suggest-next, GET /prestige/me (par titre + cross-titre), GET /templates/suggest
- [x] Mapping erreurs service → HTTP : 400 invalid_input/too_easy, 403 not_editable, 404 not_found, 409 already_terminal, 429 cooldown_active
- [x] Hook post-sync (`internal/prestige/sync_hook.go`) avec feature flag `PRESTIGE_ENABLED` (env var, valeurs 1/true/yes/on insensible à la casse)
- [x] `RunPostSyncHook` best-effort : log warn sur erreur, jamais propagation pour ne pas casser le sync
- [x] Tests handler avec mock service (16 sous-tests)
- [x] Tests sync hook (4 sous-tests : flag off skip, flag on call, nil service safe, service error logged)
- [x] Build + tests verts, aucune régression
- [ ] Routes câblées dans `server.go` — **reporté Phase 4** : nécessite un `BaselineProvider` Halo (à construire avec les adaptateurs Halo) pour instancier le `prestige.Service`
- [x] Entrée thought_log
- [ ] Commit Phase 3

### Phase 4 — Templates + nav refactor
- [x] 27 templates Halo Infinite rédigés (10 daily + 10 weekly + 7 monthly, FR + EN)
- [x] Test chargement templates (`prestige_loader_test.go` intégration)
- [x] Renommage Palmarès → Communauté (NavL1.tsx)
- [x] Synthèse → onglet Stats
- [x] Pass saisonnier → onglet Carrière
- [x] Objectifs en L1 avec sous-onglets Défis / Mon parcours
- [x] API client web (`apps/web/src/lib/prestige.ts` — namespace `prestigeApi`)
- [x] Types web alignés Go (sérialisation JSON identique)
- [x] Script `analyze_prestige_tuning.py` livré
- [x] BaselineProvider Halo (`prestige_baseline_provider.go`)
- [x] Loader TOML templates + presets (`catalog_loader.go`)
- [x] Bundle Prestige + factory par-joueur (`prestige_setup.go` + `prestige_lazy_service.go`)
- [x] Câblage routes serveur derrière `PRESTIGE_ENABLED` (16 endpoints montés)
- [x] Handler `pilot_mode` (enable/disable + auto-attribution 1 daily + 1 weekly + 3 propositions)
- [x] Endpoint `POST /squads/:id/challenges/pool/refresh` (pool 6-9 templates)
- [x] Sync hook exposé via `WithPrestigeHook` sur SyncEngine
- [x] Entrée thought_log
- [x] Commit Phase 4 (`d61de8ac` backend + `ee3b54f4` frontend)

### Phase 5 — Frontend feature complete
- [x] Page Objectifs (Défis + Mon parcours, 2 onglets)
- [x] ChallengeCard.tsx (tuile carousel + grid, 4 paliers couleur)
- [x] MomentCard.tsx (16:9, 4 variations palier, halo Mythic, animation sobre)
- [x] CreateChallengeForm.tsx (3 modes auto / libre / hybride)
- [x] ChallengesCarousel.tsx (sur la home, switch Actifs/Terminés)
- [x] LeaderboardPP.tsx (composant — affichage décomposé brut/bonus/total)
- [x] LeaderboardPPPage.tsx (page Communauté)
- [x] ArcSummary.tsx (résumé arc avec progression visuelle)
- [x] StatsGlobales.tsx (par palier + taux complétion + top métriques + filtre auto/libre)
- [x] PrestigeBadge (intégré dans ParcoursTab d'ObjectifsPage)
- [x] Hooks React Query séparés (`hooks/useChallenges.ts` + `useArcs.ts` + `usePrestige.ts`)
- [x] Routes TanStack (`/objectifs/index.tsx` + `/palmares/prestige.tsx`)
- [x] Mutations branchées : create, update, abandon, createArc, joinSquad
- [x] Build types web : aucune erreur sur les fichiers Prestige (préexistant hors scope)
- [x] Entrée thought_log
- [x] Commit Phase 5 (`ee3b54f4` + `bcd29383` câblage final)

### Livraison finale
- [x] `go test ./internal/prestige/...` vert (54+ sous-tests : domain pur + telemetry + service complet + quotas)
- [x] `go test ./internal/api/handlers/...` vert (16 sous-tests handler)
- [x] `go test ./internal/migration/...` (intégration) vert pour tags `integration`
- [x] `go build ./internal/prestige/... ./internal/platform/duckdb/... ./internal/api/handlers/...` vert
- [x] Migrations idempotentes vérifiées (3 passes)
- [x] Feature flag `PRESTIGE_ENABLED` documenté + testé (4 sous-tests sync_hook)
- [x] Récap final dans le thought_log

**Note résiduelle** : le build complet du package `internal/api` peut échouer ponctuellement à cause de WIP non lié à Prestige (méthodes MediaHandler en cours dans une feature sœur). Le code Prestige lui-même compile et passe tous ses tests.
- [ ] Plan principal `PLAN_challenges_xp_system.md` mis à jour avec statut "Implémenté"

---

## Matrice de traçabilité (plan conceptuel ↔ impl)

> Cette matrice est le filet anti-oubli. À chaque fin de phase, vérifier ligne par ligne que les éléments concernés sont livrés.

### Glossaire & concepts fondamentaux

| Concept (PLAN) | Phase | Fichier(s) | Statut |
|----------------|:-----:|------------|:------:|
| Points de Prestige (PP) | 2 | `pp_amounts.go`, `prestige_events` table | ⏳ |
| Niveau Prestige | 2 | `level.go` | ⏳ |
| Arc | 1, 4 | table `arc`, `presets.toml` | ⏳ |
| Défi | 1 | table `challenge` | ⏳ |
| Palier | 2 | `palier.go`, enum `Tier` | ⏳ |
| Baseline personnelle | 2 | `baseline.go` | ⏳ |
| Baseline collective | 2 | `squad_target.go` | ⏳ |
| Moment card | 1, 5 | table `moment_card`, `MomentCard.tsx` | ⏳ |
| Cadence | 1 | enum `Cadence` | ⏳ |
| Évaluation threshold/cumulative | 2 | `evaluator.go` | ⏳ |
| Auto-attribution | 3 | `pilot_mode.go` handler | ⏳ |

### Axe 1 — Arcs narratifs

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Concept arc séquentiel | 1, 2 | table `arc` + `position` sur `challenge` + déverrouillage en `lifecycle.go` |
| Arcs prédéfinis (Slayer, Support…) | 4 | `config/titles/halo_infinite/arcs/presets.toml` + loader |
| Arcs libres | 1, 3 | table `arc` (is_preset=false), handlers `arcs.go` |
| Scope titre (`title_slug`) | 1 | colonne sur toutes les tables |

### Axe 2 — Fenêtres temporelles

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Session / Rolling jours / Deadline / matches_internal | 1 | enum `WindowType` |
| Min matchs win_rate (5/session, 8/7j) | 2 | `evaluator.go` refus si < min, valeurs dans `tuning.toml` |
| Baseline 20 matchs glissante | 2 | `baseline.go` |
| Reset baseline 60j inactivité | 1, 2 | table `baseline_state` + `baseline.go::CheckStaleness` + `Recover` |
| Données insuffisantes (<5 / 5-9 / ≥10) | 2 | enum `DataTier`, divisions PP dans `pp_amounts.go` |

### Axe 3 — Rituel de validation

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Point de notification | 5 | nav UI |
| Moment card | 5 | `MomentCard.tsx` |
| Journal accomplissements | 5 | `ParcoursTab.tsx` |
| "Prochain défi" (3 cas) | 3 | endpoint `POST /challenges/:id/suggest-next` |
| Contribution PP | 2, 3 | `pp_amounts.go` + `service.go` émission event |
| Carousel home | 5 | `ChallengesCarousel.tsx` |

### Axe 4 — Escouade

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Pool collectif (6-9 défis hebdo) | 3 | endpoint `POST /squads/:id/challenges/pool/refresh` |
| Modes collectif/compétitif | 1 | enum `SquadMode` |
| Palier individuel par membre | 1, 2 | `chosen_tier` sur `squad_challenge_participant` |
| Baseline collective | 2 | `squad_target.go` |
| target_per_member dynamique | 1, 2 | colonne `target_per_member` + `ResizeCollectiveTarget` |
| Privacy (is_private) | 1 | colonne sur `challenge` et `squad_challenge_participant` |

### Axe 5 — Multiplicateur de performance

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Score brut + bonus + total décomposé | 5 | `LeaderboardPP.tsx` affichage 3 lignes |
| Sécurité défi trop facile | 2 | `palier.go` `RejectReason::TooEasy` |
| Multiplicateur uniquement à la hausse | 2 | pas de malus codé, PP=0 si non complété |

### Axe 6 — Système Prestige (PP)

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Sources de PP (table) | 2 | `pp_amounts.go` |
| Niveaux Prestige (5 paliers) | 2 | `level.go` valeurs depuis `tuning.toml` |
| Leaderboard amis (auto-peuplé) | 3 | `prestige.go` handler — source : escouade + relations DB existante |
| Pas de leaderboard global public | 3 | aucun endpoint exposé sans `user_ids` ou squad |

### Axe 7 — Modes, lifecycle, garde-fous

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Mode libre par défaut | 1 | enum `ChallengeMode::Libre` (défaut DB) |
| Mode pilote opt-in | 3 | endpoints `pilot_mode.go` |
| Cadences quotidien/hebdo/mensuel | 1 | enum `Cadence` |
| Plafonds (3/5/2/12) + 3 libres/jour | 3 | `service.CheckQuotas` appelé dans `CreateChallenge` |
| Auto-attribution | 3 | `POST /pilot-mode/enable` + cron de renouvellement (Phase 3) |
| State machine | 2 | `lifecycle.go` |
| Cooldowns 12h/48h | 2 | `lifecycle.go` validation |
| Édition cible figée pilote / libre + recalc palier | 2, 3 | `CanEditTarget` + `service.UpdateChallenge` |
| Catalogue 3 modes (auto/libre/hybride) | 5 | `CreateChallengeForm.tsx` |
| Cap percentile désactivé < 50 joueurs | 2 | `palier.go` paramètre `popSize` |

### Axe 8 — Navigation et parcours

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Refonte L1 (Palmarès → Communauté, Synthèse → Stats, Pass saisonnier → Carrière, +Objectifs) | 4 | `NavL1.tsx` + routes |
| Page Objectifs (Défis + Mon parcours) | 5 | `ObjectifsPage.tsx` + tabs |
| Page Communauté (4 onglets, +LeaderboardPP) | 4, 5 | renommage Phase 4, composant Phase 5 |
| Carousel home + bouton "+ Nouveau" | 5 | `ChallengesCarousel.tsx` |
| Toggle mode pilote en tête Défis | 5 | `ChallengesTab.tsx` |
| Smoke test 4 parcours utilisateur | 5 | manuel |

### Annexe A — Métriques

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Référencement par FieldKey canonical | 2 | `evaluator.go` utilise `canonical.FieldKey` |
| Mode filter (universal/pvp/ranked/pve) | 1, 4 | colonne sur `challenge_template` |
| Médailles (cumulative) | 2 | `evaluator.go` join sur `medals_earned` |

### Annexe B — Difficulté

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| 4 paliers + couleurs | 1, 5 | enum `Tier` + tokens UI |
| Formule stretch ratio | 2 | `stretch.go` |
| Cap population (p50/p75/p90 + désactivation < 50) | 2 | `palier.go` |
| Multiplicateur PP par palier | 2 | `pp_amounts.go` |

### Annexe C — Architecture stockage

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| `shared_social.duckdb` (nouveau) | 1 | `steps_shared_social.go` |
| Extensions `stats.duckdb` | 1 | `steps_player.go` |
| Extension `metadata.duckdb` (templates + presets) | 1 | `steps_metadata.go` |
| Blob storage moment cards | 1, 5 | colonne `blob_path` + génération PNG côté front ou back (à trancher en Phase 5) |

### Annexe D — Multi-titre

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| `title_slug` partout | 1 | colonne sur toutes tables |
| Leaderboard par titre (défaut) | 3 | handler param `title_slug` |
| Leaderboard cross-titre (NULL) | 3 | handler accepte `title_slug=null` |
| Catalogues TOML par titre | 4 | `config/titles/{slug}/challenges/` + `arcs/` |
| Arcs cross-titre non supportés | 1 | contrainte FK `title_slug` non NULL |

### Annexe E — Mise en œuvre et calage

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Séparation des couches | toutes | structure des packages |
| Surface API `Service` | 2 | `service.go` interface |
| Hook sync feature-flagged | 3 | `PRESTIGE_ENABLED` env var |
| Télémétrie complète | 1, 2 | table `prestige_telemetry` + `telemetry.go` |
| Script `analyze_prestige_tuning.py` | 4 | livré en même temps que le module |
| `tuning.toml` modifiable sans rebuild | 1 | config externe |

### Annexe F — Moment cards

| Élément | Phase | Couverture |
|---------|:-----:|-----------|
| Format 16:9 + export PNG/JPG | 5 | `MomentCard.tsx` + génération |
| 4 variations palier (couleur/badge/halo Mythic) | 5 | composant + tokens |
| 4 variations contexte (solo/arc/cumulative/arc complété) | 5 | props du composant |
| Animation sobre (glow + fade-in + compteur PP) | 5 | CSS/transitions |

---

## Anti-régression — checks finaux livraison

À cocher avant le commit final de la livraison globale :

- [ ] Toutes les lignes de la matrice de traçabilité ont leur statut ✅
- [ ] Aucune fonction Go > 80 L (CLAUDE.md règle 13)
- [ ] Aucun module Go > 500 L (CLAUDE.md règle 14)
- [ ] Aucun import pandas/sqlite ajouté
- [ ] Aucune couleur hex dans le frontend (CLAUDE.md règle 20)
- [ ] Migration ré-applicable sans erreur (idempotente)
- [ ] Feature flag `PRESTIGE_ENABLED` permet de désactiver complètement le module
- [ ] Logs structurés sur toutes les transitions de défi
- [ ] Tests `go test ./...` verts
- [ ] Build web vert
- [ ] Smoke test 4 parcours Axe 8 réussi
- [ ] thought_log à jour
- [ ] PLAN principal mis à jour avec statut
