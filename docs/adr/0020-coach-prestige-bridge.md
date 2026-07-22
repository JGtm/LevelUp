# ADR 0020 — Coach proactif : pont vers Prestige

**Date** : 2026-05-25
**Status** : Proposed
**Branch** : `feat/coach-proactive-prestige-bridge` (worktree initial : `worktree-coach-prestige-bridge-adr`)

## Context

Deux systèmes de progression coexistent dans LevelUp depuis fin 2026 :

- **Coach proactif** (ADR 0014, V2 Ascension) — détecteurs purs `streaks`, `records`,
  `milestones`, `LOWESS`, `combat_patterns`, `comeback`. Le coach **observe** et **émet
  des alertes positives** via `player_notifications`. Il ne crée jamais d'objectif
  actionnable côté joueur.
- **Prestige** (ADR 0005) — système de défis personnels et collectifs avec catalogue
  de `Template` calibrés (baseline + stretch + palier), `Arc` libres ou preset,
  `SquadChallenge`, mode pilote, PP/niveaux. Tout est créé soit par le joueur
  (mode libre), soit par auto-attribution (mode pilote), soit par le scheduler
  d'escouade. **Le coach n'invoque rien.**

Le gap conceptuel : quand le coach détecte qu'un joueur a progressé sur l'accuracy
(LOWESS positive +12 % sur 14j), il sait **qu'il y a quelque chose à consolider**
mais ne propose **aucun objectif** au joueur. À l'inverse, Prestige peut proposer
des défis aléatoires depuis le catalogue mais n'a pas de signal pour cibler
**ce qui compte maintenant** pour ce joueur précis.

Le user product veut combler ce gap en V2 : une option utilisateur
`coach_proactive_mode` (défaut ON depuis 2026-07-22, cf. DEC-2 du plan
PLAN_ASCENSION_UX_2026-07 ; off à l'origine) qui, quand activée, fait que le
coach **propose** des défis et des arcs Prestige calibrés sur les signaux
détectés.
Le joueur reste maître de l'acceptation (pas d'auto-attribution silencieuse).

## Decision

Créer un **troisième acteur** entre coach et Prestige : le package
`internal/progression/coach_advisor/`. Il consomme les sorties du coach
(signaux LOWESS, patterns, near-miss records/milestones, comeback), interroge
le catalogue Prestige (`TemplateRepo.ListByTitle()`), et produit des
`Proposal` (matérialisations différées de challenges ou d'arcs).

### Architecture

```
post-sync hook (existant, ADR 0014)
   ├─ coach.Generate()         → []Alert (inchangé)
   ├─ coach_advisor.GenerateProposals()  → []Proposal (NOUVEAU)
   │     ↳ lit TemplateRepo + LUSRSnapshot + PatternReport + UserSettings
   │     ↳ filtre selon coach_proactive_mode
   │     ↳ persiste dans coach_proposal (status=pending)
   ├─ coach.emitter.Emit()     → player_notifications
   │     ↳ inclut désormais les alertes coach_proposal
   └─ prestige.EvaluateForUser() (inchangé)

HTTP — actions joueur
   ├─ GET    /coach/proposals?status=pending
   ├─ POST   /coach/proposals/{id}/accept   → prestige.CreateChallenge / CreateArc
   ├─ POST   /coach/proposals/{id}/dismiss
   └─ PATCH  /settings/coach-proactive       (toggle ON/OFF)
```

### Invariants (le contrat avec Prestige)

5 invariants gravés dans `coach_advisor`. Toute violation casse la cohérence
philosophique avec Prestige et doit faire échouer la PR.

**I1 — Tout challenge passe par `CalculatePalier()`**
Pas de bypass du palier system. Si stretch < seuil_normal → `RejectTooEasy` →
le coach ne matérialise pas. Le tier final est toujours calculé par Prestige
à partir de la baseline personnelle du joueur au moment de l'acceptance.

**I2 — La baseline reste l'ancre**
Le coach n'invente jamais une valeur cible absolue. Il invente une **direction**
(metric + delta narratif + window). Prestige matérialise la cible via
`baseline × stretch_ratio` au moment du `CreateChallenge`.

**I3 — Tout template synthétisé reste un `Template` valide**
Mêmes champs : `metric`, `window_type`, `window_value`, `cadence`, `eval_type`,
`mode_filter`, `lusr_components`, `radar_axes`, `targets_*`. Indistinguable
structurellement d'un template catalogue (sauf flag `source` pour analytics).
Validé par les mêmes routines de validation Prestige.

**I4 — Tout arc dynamique reste un `Arc` standard**
`IsPreset=false`, étapes stockées comme `Challenge` avec `arc_id` non nul.
Pas de nouvelle table arc. Pas de structure parallèle. Le frontend les consomme
via les mêmes endpoints `ListArcs` que les arcs libres existants.

**I5 — Aucune génération sans signal fort**
Seuil dur `signal.strength >= tuning.coach_advisor.synthesis_min_strength`
(défaut 0.6). En-dessous, fallback sur le catalogue uniquement (matcher score).
Sans signal du tout, le coach se tait — pas de proposition aléatoire.

### Le coach_advisor n'est pas un deuxième Prestige

Il ne calcule pas de palier, ne crée pas d'event PP, ne tient pas d'état de
progression d'un challenge. Il produit des `Template` et des `Arc` (objets
métier Prestige) et les **remet** à `prestige.Service` pour matérialisation.
Toute la calibration reste côté Prestige.

### Modification minimale de Prestige

L'unique extension du Service Prestige est l'ajout d'un champ `Source string`
dans `CreateChallengeRequest` et `CreateArcRequest` (valeurs : `"user"`,
`"pilot_mode"`, `"coach"`). Tracé dans `prestige_telemetry.source`. Aucun
nouveau endpoint, aucun nouveau type, aucun bypass du flow standard.

### Settings utilisateur (révisé Phase 1 — 2026-05-25)

Toggle `coach_proactive_mode` (booléen, default `true` depuis 2026-07-22 —
DEC-2 ; `false` à l'origine) persisté dans `app_settings.json` (global
app-level, pattern identique à `ShowProgression`).
LevelUp est une app locale single-user, pas multi-user — pas besoin d'une table
`user_preferences` per-user comme initialement envisagé dans l'esquisse.

Lu **avant toute logique advisor** dans le post-sync hook — si off,
short-circuit total. Modifiable via `PATCH /api/v1/settings` (endpoint
existant, extension purement additive de `domain.UpdateSettingsRequest`).

Cette simplification ne change pas les invariants I1-I5 ni l'architecture
`coach_advisor` — uniquement le mécanisme de stockage du toggle.

### Pas d'expiration des proposals

Décision produit : les proposals pending ne s'expirent pas par âge. Le cleanup
est piloté par :

- **Supersession** : si un signal plus fort (>= 10 % uplift) arrive sur la même
  `(metric, radar_axis)`, l'ancienne proposal devient `superseded`.
- **Obsolescence sur completion** : quand le joueur complète un challenge issu
  d'une acceptance coach, toutes les proposals pending sur le même axis sont
  marquées `obsoleted` (le coach se tait sur cet axe le temps que la baseline
  bouge).
- **Job de garbage collection léger** : marquer `stale` les pending > 60 j sans
  aucun nouveau signal sur l'axis (job nocturne, optionnel V2.1).

Le joueur peut toujours `dismiss` manuellement.

### Limites volumétriques

| Setting | Default | Justification |
|---|---|---|
| `max_proposals_per_sync` | 3 | éviter de noyer le centre de notif |
| `max_active_coach_arcs` | 1 | sémantique pilot_mode `total_active_max` analogique |
| `synthesis_min_strength` | 0.6 | I5 — signaux faibles fallback catalogue uniquement |
| `min_catalog_match_score` | 0.4 | scoring `matcher.go` (lusr+axis+metric) |

### Schéma DuckDB

Nouvelle table `coach_proposal` dans **stats.duckdb** (par joueur) :

```sql
CREATE TABLE coach_proposal (
    id              VARCHAR PRIMARY KEY,
    user_id         VARCHAR NOT NULL,
    title_slug      VARCHAR NOT NULL,
    kind            VARCHAR NOT NULL,         -- 'challenge' | 'arc'
    template_id     VARCHAR,                  -- NULL si kind=arc
    challenges_spec_json VARCHAR,             -- N templates ordonnés (arc only)
    suggested_tier  VARCHAR,                  -- indicatif UI uniquement (cf. I1)
    source_signal   VARCHAR NOT NULL,
    source_metric   VARCHAR,
    radar_axis      VARCHAR,
    strength        DOUBLE,
    origin          VARCHAR NOT NULL,         -- 'catalog' | 'synthesized'
    reason_key_en   VARCHAR,
    reason_key_fr   VARCHAR,
    reason_params   VARCHAR,                  -- JSON
    status          VARCHAR NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMP NOT NULL,
    expires_at      TIMESTAMP,                -- nullable (Q3)
    resolved_at     TIMESTAMP,
    resolved_ref    VARCHAR,                  -- challenge_id ou arc_id si accepté
    superseded_by   VARCHAR,
    superseded_at   TIMESTAMP,
    obsoleted_at    TIMESTAMP
);

CREATE INDEX idx_coach_proposal_user_status
    ON coach_proposal(user_id, title_slug, status);
CREATE INDEX idx_coach_proposal_metric_axis
    ON coach_proposal(user_id, title_slug, source_metric, radar_axis);
```

**Pattern d'écriture** : append + `SELECT-then-UPDATE-or-INSERT` pour les
transitions de status (cf. CLAUDE.md règle Phase 4 ART, ADR 0019). Pas de
`ON CONFLICT DO UPDATE`.

### Capabilities multi-titres

Le pont requiert la capability `prestige.templates` pour le titre. Si absente,
le coach_advisor retourne `nil, nil` sans erreur (log debug). Halo Infinite
déclare aujourd'hui cette capability ; futurs titres l'activeront quand leur
catalogue Prestige sera seed.

## Use cases couverts en V2

| # | Signal détecté | Action proposée | Couche |
|---|---|---|---|
| U1 | LOWESS positive accuracy 14j | Challenge tier Heroic, template accuracy_session | matcher (catalog) |
| U2 | combat_pattern_fragile (DR < P70) | Arc preset "Survival Mastery" 3 étapes | matcher (preset arc) |
| U3 | record_near_miss < 5 % du PB all-time | Challenge mode libre, target = PB + 1 % | matcher + signals |
| U4 | comeback_welcome après 7 j inactif | Challenge soft tier Normal cadence daily | matcher (catalog) |
| U5 | milestone_near_miss 90 % | Challenge cumulative ciblant ce milestone | matcher (catalog) |
| U6 | combat_pattern_actif + LOWESS combat axis | Arc dynamique "Combat Excellence" | arc_composer |
| U7 | Signal sur métrique non couverte par catalogue | Template synthétisé persisté + proposal | synthesizer (cf. ADR 0028) |
| U8 | 3 signaux convergents sur même radar_axis | Arc dynamique composé (catalog + synthèse mélangés) | arc_composer |
| U9 | Signal plus fort sur même metric 5 j après | Supersession de l'ancienne proposal pending | supersede.go |
| U10 | Acceptance d'une proposal sur axis "combat" | Obsolescence des autres pending sur cet axis | service.AcceptProposal |

## Consequences

### Positives

- Cohérence philosophique préservée : la calibration Prestige reste l'unique
  source de vérité du tier/PP/baseline (I1-I2).
- Découplage strict : `coach` reste un détecteur pur ; `prestige` reste agnostique
  du coach ; `coach_advisor` est le seul point de couplage (importé par aucun
  des deux, importe les deux).
- Suggestions sans write non sollicité : le coach **propose** uniquement
  (dédup 24h, supersession) ; aucune écriture Prestige n'est faite sans action
  du joueur. Le défaut du toggle est passé à ON le 2026-07-22 (DEC-2) —
  l'exposition par défaut reste sûre car ce sont des suggestions pures ; un
  opt-out explicite (`coach_proactive_mode=false`) reste respecté.
- Extensible : ajouter un nouveau signal demande 1 entry dans `signals.go` +
  1 mapping LUSR/radar dans `synthesis_grammar.toml`. Ajouter un nouveau type
  de proposal (squad, cross-title) ne demande pas de refonte du modèle.
- Télémétrie unifiée : `prestige_telemetry.source='coach'` permet de mesurer
  l'efficacité du coach proactif (taux d'acceptance, taux de complétion) sans
  schema dédié.

### Négatives / dette

- **Couche supplémentaire à maintenir** : un troisième acteur dans le pipeline
  post-sync. Si Prestige ou Coach refactorisent leurs API publiques,
  `coach_advisor` doit suivre.
- **Risque de pollution du catalog par synthèse** : mitigé par
  `synthesis_grammar.toml` (allowlist stricte) + dédup hash + job GC sur
  `usage_count=0` (cf. ADR 0028).
- **Bruit pour le joueur si calibration trop optimiste** : mitigé par le cap dur
  `max_proposals_per_sync=3` + supersession + opt-in explicite. Tuning itératif
  via télémétrie `prestige_telemetry` (taux d'abandon).
- **Le coach reste positif uniquement** : I5 + tuning excluent les signaux
  négatifs. Cohérent avec ADR 0014 §6.1 (pas de feedback culpabilisant). Si le
  product veut un "coach soft négatif" plus tard, c'est une décision explicite
  (V3 backlog).

### Observabilité

- `slog.InfoContext(ctx, "coach_advisor: proposals", "catalog", N, "synthesized",
  M, "arcs", A, "user", userID, "titleSlug", titleSlug)` à chaque hook.
- `slog.DebugContext` par signal traduit + par template matché.
- `slog.ErrorContext(ctx, "coach_advisor: failure", "err", err, ...)` sans
  arrêter le pipeline post-sync (best-effort, comme Prestige).
- `expvar` (ADR 0009) : compteurs optionnels V2.1 si volume justifie
  (`coach_proposals_generated_total{kind=challenge|arc, origin=catalog|synthesized}`).

## Decisions log

| Question | Décision |
|---|---|
| Default tier sur acceptance | Prestige recalcule via baseline. Le `suggested_tier` est indicatif UI. |
| Max arcs coach actifs | 1 par joueur (réutilise sémantique pilot_mode `total_active_max`). |
| Expiration proposals | Pas d'expiration fixe. Cleanup par supersession + obsolescence sur completion. |
| Squad coach (pool filtré par composition d'équipe) | V3 — backlog explicite (`.ai/V7/BACKLOG_COACH_PRESTIGE.md`). |
| Coach négatif soft | V3 — hors scope, décision produit séparée. |
| Création de templates ad-hoc | V2 — cf. ADR 0028 (synthèse). |
| Création d'arcs ad-hoc | V2 — `arc_composer.go`, étapes = templates catalog + synthétisés mélangés. |

## References

- ADR 0005 — Prestige Phased Activation (catalogue, palier system, baseline)
- ADR 0014 — Progression Tracking V2 (coach, streaks, records, milestones)
- ADR 0019 — Collect / Persist (pattern d'écriture append-only sur tables critiques)
- ADR 0028 — Template synthesis (synthèse dynamique cf. invariant I3)
- `.ai/V7/BACKLOG_COACH_PRESTIGE.md` — V3+ explicite (squad, négatif soft, cross-titre)
