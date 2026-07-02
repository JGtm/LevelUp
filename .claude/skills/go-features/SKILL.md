# Skill : go-features — Features Go existantes (internal/analysis)

## Règle fondamentale

**Toujours vérifier `apps/go-api/internal/analysis/` (et ses sous-packages `temporal/`,
`breakdown/`, `narrative/`, `timeline/` — READMEs catalogues) avant de réimplémenter.**
La quasi-totalité des algos du produit existe déjà quelque part.

## Inventaire des fichiers

### Modes
| Fichier | Exports clés |
|---|---|
| `mode_label.go` | `NormalizeModeLabel(raw, mapLabels...)` |
| `mode_category.go` | `InferModeCategoryFromPairName(pairName)`, `PairNamePrefixesForCategory()`, `AllKnownPairNamePrefixes()` |

### Scores et performances
| Fichier | Exports clés |
|---|---|
| `performance_score.go` | Calcul du performance_score par match |
| `skill_rating.go` | LUSR / CSR rating par match |
| `combat_yield.go` | Rendement combat (kills/damage ratio) |
| `match_impact.go` | Impact individuel sur l'issue du match |
| `squad_score.go` | Score agrégé de squad |
| `squad_profiles.go` | Profils de jeu des membres du squad |
| `squad_breakdown.go` | Décomposition des contributions squad |
| `squad_impact.go` | Impact du squad |
| `squad_timeseries.go` | Évolution temporelle du squad |

### Armes (pipeline film — 100% Halo Infinite)
| Fichier | Exports clés |
|---|---|
| `weapon_parser.go` | Parsing des weapon_id / weapon data |
| `weapon_scanner.go` | Scan et détection des armes |
| `weapon_reconciliation.go` | Réconciliation weapon_id↔labels |
| `weapon_correlation.go` | Corrélations entre armes |
| `weapon_data.go` | Données statiques armes |

> Ce pipeline (+ `highlight_event_parser.go`, `spawn_detection.go`, `kill_attribution.go`)
> est title-specific et destiné à migrer vers `internal/games/halo_infinite/film/`
> (plan audits 2026-07, item F12). Vérifier son emplacement actuel avant de le référencer.

### Évènements et timeline
| Fichier | Exports clés |
|---|---|
| `highlight_event_parser.go` | Parsing des highlight_events JSON |
| `killer_victim.go` | Paires killer→victim, chaînes de kills |
| `kd_timeline.go` | Timeline K/D dans le match |
| `tug_of_war.go` | Momentum match (tug-of-war) |
| `comeback.go` | Détection des comebacks |
| `spawn_detection.go` | Détection des respawns |

### Médailles et citations
| Fichier | Exports clés |
|---|---|
| `medals_earned` | (via DB, pas d'algo dédié) |
| `medal_exploit.go` | Exploitation des médailles pour insights |
| `citations.go` | Calcul des citations par match |
| `citation_snippets.go` | Snippets textuels de citations |

### Historique et sessions
| Fichier | Exports clés |
|---|---|
| `sessions.go` | Groupement en sessions de jeu |
| `match_history_avg.go` | Moyennes glissantes historique |
| `home.go` | Agrégats page d'accueil |

### Médias
| Fichier | Exports clés |
|---|---|
| `media.go` | Gestion des fichiers médias |

### Multi-titres (`internal/domain/title/`)
| Fichier | Exports clés |
|---|---|
| `registry.go` | `TitleRegistry`, `TitleDescriptor`, `PathResolver`, `NewRegistry()`, `NewPathResolver()` |
| `matcher.go` | Matching de présence Xbox/Steam → titre |

## Pattern de vérification

```bash
# Chercher si une fonction existe déjà
grep -r "func.*Score\|func.*Rating\|func.*Normalize" apps/go-api/internal/analysis/

# Lister tous les exports d'un fichier
grep "^func [A-Z]" apps/go-api/internal/analysis/squad_score.go
```

## Règle avant toute implémentation

Si une fonctionnalité ne semble pas exister dans `analysis/` :
1. Vérifier les autres sous-packages de `apps/go-api/internal/` (service/, progression/,
   games/, platform/duckdb/ — la logique peut vivre côté repo)
2. Vérifier le thought_log et les plans `.ai/V7/` (elle peut être en cours ou décidée)
3. Seulement alors implémenter — dans `internal/analysis/` pour les algos purs
   title-agnostic, `internal/games/{slug}/` pour le title-specific, `service/` pour
   l'orchestration
