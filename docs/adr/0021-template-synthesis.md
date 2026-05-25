# ADR 0021 — Synthèse dynamique de Template et Arc ad-hoc

**Date** : 2026-05-25
**Status** : Proposed
**Branch** : `feat/coach-proactive-prestige-bridge`

## Context

L'ADR 0020 cadre le pont coach → Prestige : le coach détecte des signaux et
propose des challenges/arcs Prestige calibrés. La première version envisagée
consommait **uniquement le catalogue de templates existant** (`challenge_template`
table seedée TOML). Si aucun template ne couvrait un signal détecté, le coach
restait silencieux.

Le retour produit (2026-05-25) a écarté cette approche : la richesse du modèle
exige que le coach puisse **proposer même des objectifs hors catalogue**, sinon
:

- les signaux portant sur des métriques rares (ex. `kills_vs_expected`,
  `objective_carry_time`, `damage_dealt_pve_per_wave`) sont systématiquement
  perdus ;
- la combinatoire `(metric × window × eval_type)` explose si on tente de seed
  manuellement tous les cas (10+ métriques × 4 fenêtres × 2 cadences × 2 eval =
  160 templates minimum) ;
- les arcs thématiques composés par le coach (3 signaux convergents sur un
  même axis) sont coincés à la couverture du catalogue.

Mais autoriser la **génération libre** côté coach risque de diverger
philosophiquement de Prestige : tier non calibré, baseline ignorée, métriques
absurdes, pollution du catalog par des templates jetables.

## Decision

Autoriser **deux extensions productrices** dans `coach_advisor`, sous contraintes
strictes alignées sur les invariants I1-I5 de l'ADR 0020 :

1. **Synthèse de Template** (`coach_advisor/synthesizer.go`) — produit un
   nouveau `prestige.Template` à partir d'un `Signal`, sous allowlist stricte.
2. **Composition d'Arc ad-hoc** (`coach_advisor/arc_composer.go`) — compose
   un `Arc` dynamique multi-étapes à partir de signaux convergents sur un
   même `radar_axis`. Les étapes sont des templates catalogue ET/OU
   synthétisés.

### Synthèse de Template — contrat

#### Allowlist `synthesis_grammar.toml`

Le synthesizer **refuse** de produire un template hors d'une grammaire
explicite (`config/coach_advisor/synthesis_grammar.toml`). Cette grammaire
liste les `(metric, window_type, window_value, eval_type)` autorisés. Toute
combinaison hors allowlist retourne `ErrMetricNotSynthesizable`.

Exemple (extrait initial Halo Infinite, 10-15 entrées max pour démarrer) :

```toml
[[allow]]
metric    = "accuracy"
windows   = ["last_n_matches:10", "last_n_matches:20", "rolling_days:7", "rolling_days:14"]
eval_type = "threshold"

[[allow]]
metric    = "kills_vs_expected"
windows   = ["last_n_matches:10", "last_n_matches:20"]
eval_type = "threshold"

[[allow]]
metric    = "kda"
windows   = ["session", "last_n_matches:10", "rolling_days:7"]
eval_type = "threshold"

[[allow]]
metric    = "headshots_total"
windows   = ["rolling_days:7", "rolling_days:30"]
eval_type = "cumulative"
```

L'allowlist sert deux objectifs :
- **anti-explosion combinatoire** — borne le nombre de templates synthétisables
  possibles ;
- **garantie de cohérence métrique** — seules les métriques pour lesquelles
  Prestige sait calculer une baseline et un palier sont éligibles.

#### Targets vides — Prestige calcule

Le template synthétisé **ne contient pas** de valeurs de target absolues
(`NormalTarget`, `HeroicTarget`, etc.). Il fournit uniquement les
`stretch_factors_per_tier` standards (cf. tuning Prestige : 1.08 / 1.25 / 1.50 /
2.00). C'est `CalculatePalier()` qui matérialise les targets en valeurs absolues
à l'instanciation du challenge, à partir de la baseline personnelle du joueur.

Conformité **I1** (palier calculé par Prestige) + **I2** (baseline ancre).

#### Persistance avec flag origine

Tout template synthétisé est **persisté** dans la table `challenge_template`
(metadata.duckdb), avec deux colonnes ajoutées :

```sql
ALTER TABLE challenge_template ADD COLUMN source VARCHAR DEFAULT 'catalog';
ALTER TABLE challenge_template ADD COLUMN synthesized_from_signal_kind VARCHAR;
ALTER TABLE challenge_template ADD COLUMN usage_count INTEGER DEFAULT 0;
```

Valeurs de `source` : `"catalog"` (seed TOML manuel), `"coach_synthesized"`.

Conformité **I3** (template synthétisé indistinguable structurellement).

#### Dédup par hash déterministe

L'`id` du template synthétisé est un hash déterministe de :

```
hash(metric, window_type, window_value, cadence, eval_type, mode_filter,
     sorted(lusr_components), sorted(radar_axes))
```

Si deux joueurs ont un signal identique, ils partagent le même template
synthétisé. Le `usage_count` s'incrémente. Pas de duplication.

`TemplateRepo.UpsertIfNew()` = INSERT ... ON CONFLICT DO NOTHING (pattern
sûr car écriture sur metadata.duckdb, peu concurrentielle).

#### Garbage collection

Job nocturne `coach_advisor_template_gc` (V2.1, optionnel à la livraison
initiale) : supprime les templates `source='coach_synthesized'` avec
`usage_count=0` ET `updated_at > 90 j`. Aucun challenge actif ne les
référence (FK check). Pas de cascade vers les challenges historiques car
`challenge.template_id` est dénormalisé à la création (les valeurs target
matérialisées sont copiées localement).

### Composition d'Arc ad-hoc — contrat

#### Conditions de composition

`arc_composer.TryCompose([]Signal)` retourne `(ArcSpec, ok)`. `ok = true`
uniquement si :

- `len(signals_partageant_un_axis) >= tuning.arc_composition.min_signals_for_arc`
  (défaut 2) ;
- `shared_axis_required = true` (tous les signaux retenus doivent partager
  au moins un `radar_axis`) ;
- Au moins un signal a `strength >= synthesis_min_strength` (sinon pas
  d'autorisation à mélanger synthèse et catalog).

Si `ok=false`, l'arc_composer ne propose rien (les signaux deviennent des
challenges individuels via `matcher`).

#### Structure de l'arc

`ArcSpec` :
- `Title`, `Description` (FR/EN) générés via templates i18n paramétrés par
  le `radar_axis` partagé ;
- `RadarAxis` (l'axis pivot) ;
- `Steps []ChallengeSpec` (2-4 étapes, cappé par `tuning.arc_composition.max_arc_steps`,
  défaut 4) ;
- Chaque `ChallengeSpec` référence soit un `template_id` catalog, soit un
  `template_id` synthétisé (déjà persisté).

#### Progression de tier suggérée

Les étapes 1..N reçoivent un `suggested_tier` croissant :
- 2 étapes : Normal, Heroic
- 3 étapes : Normal, Heroic, Legendary
- 4 étapes : Normal, Heroic, Legendary, Mythic

**Suggestion UI uniquement** (cf. I1). À l'acceptance, `prestige.CreateChallenge`
recalcule le tier réel via baseline. Si le joueur n'a pas la baseline pour
soutenir le tier suggéré, Prestige descend automatiquement (ou rejette avec
`RejectInsufficientData`). L'arc reste créé avec autant d'étapes que
matérialisables.

#### Cohérence narrative

Le titre et description de l'arc sont **paramétrés par l'axis pivot**, pas par
les métriques individuelles :

| Axis | Titre FR | Titre EN |
|---|---|---|
| `combat` | "Domination en combat" | "Combat Mastery" |
| `survival` | "Reprise solide" | "Resilient Comeback" |
| `support` | "Pilier d'équipe" | "Team Pillar" |
| `objective` | "Objectif d'abord" | "Objective First" |
| `score` | "Performance d'élite" | "Elite Performance" |
| `impact` | "Joueur d'impact" | "Impact Player" |

Templates statiques (lookup dans `coach_advisor/labels.go`), pas de génération
LLM. Les params i18n peuvent enrichir avec le nombre d'étapes ou la métrique
dominante.

Conformité **I4** (Arc standard, IsPreset=false).

### Garde-fous globaux

| Garde-fou | Mécanisme |
|---|---|
| Pas de synthèse sans signal fort | I5 : `strength >= synthesis_min_strength` (défaut 0.6) |
| Pas de combinaison métrique × window absurde | `synthesis_grammar.toml` allowlist stricte |
| Pas de tier non calibré | I1 : Prestige recalcule via `CalculatePalier()` à l'acceptance |
| Pas de baseline ignorée | I2 : targets matérialisés à partir de la baseline du joueur |
| Pas de duplication dans le catalog | dédup hash + `usage_count` |
| Pas d'arc sans cohérence narrative | `shared_axis_required = true` |
| Pas d'arc à 1 étape | `min_signals_for_arc = 2` minimum |
| Pas d'arc trop ambitieux | `max_arc_steps = 4`, progression Normal→Mythic |
| Pas de pollution permanente du catalog | job GC sur `usage_count=0` + 90j |

## Consequences

### Positives

- Couverture extensive sans seed TOML manuel : 10-15 entrées d'allowlist
  produisent ~50-100 templates virtuels combinables, suffisant pour couvrir
  tous les signaux coach actuels.
- Réutilisation cross-joueurs : un template synthétisé pour un joueur est
  immédiatement disponible pour les autres (dédup hash). Effet réseau.
- Arcs dynamiques riches : compositions thématiques (3+ signaux sur axis
  combat → arc "Combat Excellence") sans pré-seed de PresetArc spécifiques.
- Cohérence préservée : tout passe par les routines Prestige existantes
  (CalculatePalier, baseline, palier). Aucun bypass.
- Observabilité : `prestige_telemetry.source='coach'` +
  `challenge_template.source='coach_synthesized'` permettent d'isoler l'impact
  du coach proactif vs le mode user/pilot dans les analytics (taux acceptance,
  abandon, complétion).

### Négatives / dette

- **Complexité du synthesizer** : la logique
  `signal → window/cadence/eval_type/lusr/axes` doit être correcte pour tous
  les `SignalKind`. Couvert par tests unitaires exhaustifs (1 test minimum
  par kind). Reste un point d'attention si on ajoute de nouveaux kinds.
- **Labels synthétisés peuvent être moins évocateurs** que ceux écrits à la
  main : templates i18n paramétrés ("Improve your {metric} over the last
  {window}") sont neutres. Si le product juge le ton insuffisant, possibilité
  d'ajouter une couche de labels override par `(metric, window)` dans le
  manifest i18n.
- **Allowlist à maintenir** : ajouter une nouvelle métrique requiert un edit
  TOML + déploiement. Volontaire : c'est la barrière qui empêche la dérive.
- **Job GC à implémenter** : si non livré en V2 initial, le catalog peut
  enfler. Mitigation : `usage_count` permet de détecter sans intervenir, et
  le GC reste optionnel (V2.1).

### Risques mitigés

| Risque | Mitigation |
|---|---|
| Synthèse produit un template absurde (metric+window incompatibles) | Allowlist stricte + validation Prestige.Template.Validate() au save |
| 2 joueurs reçoivent des templates différents pour le même signal | Hash déterministe → même `template_id` |
| Synthèse contourne le palier | I1 enforced par tests d'intégration (`prestige_telemetry.tier != null`) |
| Arc trop punitif (3 défis Legendary d'affilée) | Progression Normal→Mythic + Prestige recalcule via baseline + télémétrie d'abandon |
| Pollution catalog par templates jetables | Dédup hash + GC futur sur `usage_count=0` |

## Implementation phases

| Phase | Livrable | Référence |
|---|---|---|
| 5 (ADR 0020) | `synthesizer.go` + tests + `synthesis_grammar.toml` | Cf. plan §5 |
| 6 (ADR 0020) | `arc_composer.go` + tests + `labels.go` | Cf. plan §6 |
| 7 (ADR 0020) | Orchestration dans `service.GenerateProposals()` | Cf. plan §7 |
| V2.1 | Job GC `coach_advisor_template_gc` | Si volume justifie |

## References

- ADR 0020 — Coach proactif : pont vers Prestige (invariants I1-I5)
- ADR 0005 — Prestige Phased Activation (`Template`, `Arc`, `CalculatePalier`)
- ADR 0019 — Collect / Persist (pattern d'écriture)
- `config/coach_advisor/tuning.toml` (seuils synthèse + composition)
- `config/coach_advisor/synthesis_grammar.toml` (allowlist)
