# Specs reproductibles des graphiques (migration Python → Go)

> Un dossier par page Streamlit, un fichier YAML par chart/tableau, une `_index.md` par page pour la composition.

## Pourquoi

`.ai/CHARTS_AND_TABLES.md` est un **inventaire** descriptif (~80 visuels listés, type/axes/source). Il ne suffit pas pour **reproduire à l'identique** un chart en Go : il manque la requête de données, les paramètres de calcul (lissage, KDE, IQR), le layout Plotly détaillé, les hovertemplates, les états vides, la composition de page.

Ce dossier complète l'inventaire avec une **spec reproductible** par chart, calibrée pour qu'un développeur Go puisse rebâtir le visuel sans relire le code Python.

## Méthodologie hybride (B + C)

- **B — manuel ciblé** : un fichier YAML par chart, un `_index.md` par page.
- **C — extraction semi-automatique** : `scripts/specs/extract_chart_spec.py`
  1. **AST** sur le code source (lu via `git show origin/v7/cockpit:src/...`) → squelette : signature, traces littérales, layout littéral, clés `viz_t`.
  2. **Runtime** sur fixture (`fig.to_dict()`) → valeurs dynamiques : hauteurs calculées, customdata, hovertemplate résolus, ranges auto.
  3. **Merge** : YAML final structure stable (AST) + valeurs réelles (runtime).

Les YAML sont édités à la main pour ce que l'AST/runtime ne capture pas : SQL upstream, conditions d'affichage, composition de page.

## Périmètre

- **Branche source du Python** : `origin/v7/cockpit` (lu via `git show`, pas de checkout).
- **Branche de travail des specs** : la branche courante (`docs/playlists-catalog-design` au démarrage).
- **Pilote** : page Synthèse (`src/ui/pages/synthesis.py`) — voir `synthesis/_index.md`.
- **Gamertag fixture** : `JGtm` (`data/titles/halo_infinite/players/JGtm/stats.duckdb`).

## Arborescence

```
.ai/charts_specs/
├── README.md                       # ce fichier
├── _schema.yaml                    # schéma YAML par chart (champs autorisés)
├── _theme_default.yaml             # thème Halo commun (template, couleurs, axes, légendes)
├── synthesis/                      # pilote
│   ├── _index.md                   # composition de page (ordre, période, fragments, controls page-level)
│   ├── 01_outcomes_by_map.yaml
│   ├── 02_outcomes_by_mode.yaml
│   ├── 03_winrate_heatmap.yaml
│   ├── 04_top_matches_by_week.yaml
│   └── 05_solo_squad_duel.yaml
└── _generated/                     # sortie brute fig.to_dict() (vérification)
    └── synthesis/
        └── *.json
```

## Thème commun

`_theme_default.yaml` contient les valeurs par défaut héritées par toutes les charts (template `plotly_dark`, couleurs Waypoint Halo, palette HALO_COLORS, hoverlabel, axes grid, légendes horizontales). Les YAML par chart **n'écrivent que les overrides** — quand un champ est absent, le thème par défaut s'applique.

## Schéma YAML d'un chart

Voir `_schema.yaml` pour la liste exacte des champs. Sections principales :

| Section | Contenu | Source |
|---|---|---|
| `id`, `title`, `page`, `chart_kind` | Identifiants + type de chart | manuel |
| `source_function`, `source_helpers` | `path::function` sur origin/v7/cockpit | manuel |
| `data` | `computed_by`, service amont, SQL/Polars, filtres, transformations, `bucket_logic`, **`timezone`** | manuel |
| `traces[]` | Type Plotly, couleurs, `customdata`, `hovertemplate`, `data_transform`, `text_data`, `show_when`, `clip` | AST + runtime + manuel |
| `heatmap` | `colorscale`, `z_range`, `nan_treatment`, `cell_label`, `colorbar` | manuel |
| `layout` | `height` (avec branches if-else), margin, barmode, axes (incl. `autorange:"reversed"`, `yaxis2`), `shapes`, `annotations` | AST + runtime |
| `display` | `shown_when`, `empty_state`, config Plotly, `fragment_isolation`, `preceded_by` | manuel |
| **`controls`** | Widgets pilotant le chart : selectbox, toggle, segmented_control, slider… avec `position`, `scope`, `effect`, `affects_charts`, `requires_refetch` | manuel |
| `i18n_keys` | Liste des clés `viz_t`/`t` utilisées | AST |
| `interactivity` | Click target, drilldown, `session_state_writes` | manuel |
| `fingerprint` | Commit, généré-le, dataset fixture, champs AST/runtime/manuel | runtime |

## Contrôles (widgets) liés à un chart

Un chart peut être piloté par des contrôles UI placés **au-dessus, en dessous, ou à côté** : selectbox, toggle, segmented_control, slider, multiselect, date_range. Ces contrôles modifient le rendu en :
- **filtrant le dataset** (`filter_dataset`) — ex : sélecteur de période sur Synthèse
- **swappant la source de données** (`swap_dataset`) — ex : "Tous les matchs" / "Session courante"
- **changeant un axe** (`change_axis`, `change_grouping`) — ex : "Trier par win rate vs total"
- **changeant la métrique** (`change_metric`) — ex : "Par minute" vs "Absolu"
- **changeant la normalisation** (`change_normalization`) — ex : "%" vs "Compte"
- **modifiant la visibilité d'une trace** (`toggle_trace_visibility`)

Chaque contrôle a un **scope** :
- `page` — affecte tous les charts de la page (documenté dans `_index.md` et référencé par id)
- `section` — affecte un groupe de charts cohérent
- `chart` — local à un seul chart, documenté dans son YAML

Le champ `requires_refetch: true|false` indique si le control déclenche un appel API supplémentaire (Go API) ou si le re-mapping est local au client (ECharts). Cette info est cruciale pour la migration : un control `requires_refetch:true` impose un endpoint paramétrable côté Go ; un control client-only se résout en JS uniquement.

## Règles d'édition

- Une spec représente **une instance** : si la même fonction est appelée deux fois avec des paramètres différents (cas §1 et §2 sur Synthèse — `plot_stacked_outcomes_by_category` map vs mode), créer **deux YAML distincts**.
- Les paramètres effectifs de l'appel (max_categories, top_n_ranks, etc.) doivent figurer dans `data.call_args`.
- Un chart partagé entre plusieurs pages se documente **par page**, en référençant la même `source_function`.

## Validation

```bash
python scripts/specs/validate_specs.py .ai/charts_specs/synthesis/
```

(Validation à écrire en même temps que la phase runtime — voir thought_log pour l'avancement.)
