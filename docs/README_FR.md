# README Français (archivé)

Le README en français a été déplacé ici : [FR/README.md](FR/README.md).
# LevelUp - Dashboard Halo Infinite

> **Analysez vos performances Halo Infinite avec des visualisations avancées et une architecture DuckDB v5 ultra-rapide.**

[![Version](https://img.shields.io/badge/Version-5.3.0-green.svg)](https://github.com/JGtm/LevelUp_with_SPNKr/releases/tag/v5.3.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.110%2B-009688.svg)](https://fastapi.tiangolo.com/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

- **v5.4 — Refactoring** — 72 sous-modules extraits (phases 0-6), architecture mixin MRO généralisée. **3693 tests**, 0 failure.
- **v5.3 — LUSR/CSR** — Système de rating TrueSkill 2 per-groupe (ranked/arena/btb/tactical/social/fun) avec calibration empirique. Notifications Discord post-sync/backfill.
- **v5.2 — Stats PvE** — Base dédiée `shared_pve.duckdb` pour les matchs Firefight. Filtres intent-based persistants. Scoreboard complet "Dernier match". Palette Okabe-Ito (accessibilité daltonisme).
- **v5.1 — Architecture optimisée** — Streamlit moderne (`@st.fragment`, `st.navigation`), éradication SQLite/Pandas, `SyncScope` centralisé, -75% temps de connexion DB.
- **v5.0 — Shared Matches** — `shared_matches.duckdb` centralise tous les matchs (-69% stockage, -72% appels API). **3323 tests**, 0 failure.

---

## Fonctionnalités

### Statistiques Avancées
- **Dashboard interactif** - Visualisez vos stats en temps réel
- **Graphiques détaillés** - Évolution K/D, précision, durée de vie, séries de frags
- **Analyse par carte** - Performance détaillée sur chaque map avec heatmaps
- **Coéquipiers** - Statistiques avec vos amis (même équipe ou adversaires)
- **Sessions de jeu** - Détection automatique avec métriques de performance

### Visualisations
- **Graphes radar** - Stats par minute et performance globale
- **Heatmaps** - Win rate par jour/heure de la semaine
- **Distributions** - Histogrammes précision, kills, scores
- **Corrélations** - Scatter plots durée de vie vs kills
- **Top armes** - Statistiques par arme avec headshot rate

### Architecture v5.3 - DuckDB Multi-DB
- **Shared Matches** — `shared_matches.duckdb` centralise tous les matchs (registry, participants, events, médailles)
- **PvE Firefight** — `shared_pve.duckdb` isole les stats Firefight (waves, boss, ennemis par type)
- **ATTACH multi-DB** — DuckDB ATTACH pour lecture transparente cross-DB
- **LUSR/CSR** — Ratings TrueSkill 2 per-groupe stockés dans `match_skill_rank` (player DB)
- **Performance** — Requêtes DuckDB < 30ms (warm), DataFrame Polars natifs, vues matérialisées

---

## Installation Rapide

**Prérequis** : Python 3.12+ recommandé (3.10 minimum). Note Windows : évitez Python 3.14 si vous constatez des crashes natifs pendant `pytest`.

```bash
# Cloner le projet
git clone https://github.com/JGtm/LevelUp_with_SPNKr.git
cd LevelUp_with_SPNKr

# Créer l'environnement virtuel
python -m venv .venv

# Activer (Windows)
.venv\Scripts\activate

# Activer (Linux/macOS)
source .venv/bin/activate

# Installer les dépendances
pip install -e .
```

**Documentation détaillée** : [INSTALL.md](INSTALL.md)

---

## Configuration

### 1. Copier le fichier d'environnement

```bash
cp .env.example .env.local
```

### 2. Configurer les tokens Azure

```env
SPNKR_AZURE_CLIENT_ID=votre_client_id
SPNKR_AZURE_CLIENT_SECRET=votre_secret
SPNKR_AZURE_REDIRECT_URI=https://localhost
SPNKR_OAUTH_REFRESH_TOKEN=votre_refresh_token
```

### 3. Récupérer le refresh token

```bash
python scripts/spnkr_get_refresh_token.py
```

**Documentation détaillée** : [CONFIGURATION.md](CONFIGURATION.md)

---

## Architecture

### Structure des Données (v5.3)

```
data/
├── warehouse/
│   ├── metadata.duckdb            # Référentiels partagés (maps, playlists, médailles)
│   ├── shared_matches.duckdb      # Tous les matchs (registry, participants, events, médailles)
│   └── shared_pve.duckdb          # Stats PvE Firefight (pve_match_stats) — v5.2
├── players/                       # Enrichissements personnels (~4 MB/joueur)
│   └── {gamertag}/
│       ├── stats.duckdb
│       │   ├── player_match_enrichment  # performance_score, session_id (SEULE table match)
│       │   ├── antagonists, match_citations, career_progression
│       │   └── match_skill_rank         # Rating LUSR ou CSR par match — v5.3
│       └── archive/               # Archives Parquet
└── backups/                       # Backups Parquet
```

### Tables DuckDB principales

| Base | Table | Description |
|------|-------|-------------|
| `shared_matches` | `match_registry` | Registre central (1 ligne/match) |
| `shared_matches` | `match_participants` | Stats de tous les joueurs (31 col, MMR) |
| `shared_matches` | `medals_earned`, `highlight_events` | Médailles et événements filmés |
| `shared_pve` | `pve_match_stats` | Stats Firefight par joueur/match |
| player `stats` | `player_match_enrichment` | performance_score, session_id |
| player `stats` | `match_skill_rank` | Rating LUSR/CSR par match |
| player `stats` | `mv_map_stats`, `mv_global_stats` | Vues matérialisées |

**Documentation technique** : [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md)

---

## Documentation

| Document | Contenu |
|----------|---------|
| [INSTALL.md](INSTALL.md) | Guide d'installation détaillé |
| [CONFIGURATION.md](CONFIGURATION.md) | Configuration des tokens et profils |
| [ARCHITECTURE_V6.md](ARCHITECTURE_V6.md) | Architecture v6 (shared matches + i18n assets) |
| [DATA_ARCHITECTURE.md](DATA_ARCHITECTURE.md) | Architecture des données |
| [SHARED_MATCHES_SCHEMA.md](SHARED_MATCHES_SCHEMA.md) | Schéma shared_matches.duckdb |
| [SQL_SCHEMA.md](SQL_SCHEMA.md) | Schémas DuckDB complets |
| [SYNC_GUIDE.md](SYNC_GUIDE.md) | Guide de synchronisation |
| [SYNC_OPTIMIZATIONS_V5.md](SYNC_OPTIMIZATIONS_V5.md) | Optimisations sync v5 |
| [BACKUP_RESTORE.md](BACKUP_RESTORE.md) | Backup et restauration |
| [TESTING_V5.md](TESTING_V5.md) | Stratégie de tests v5 |
| [FAQ.md](FAQ.md) | Questions fréquentes |

---

## Contribution

Les contributions sont les bienvenues ! Voir [CONTRIBUTING.md](CONTRIBUTING.md) pour les guidelines.

---

## Stack Technique

| Technologie | Usage |
|-------------|-------|
| **Python 3.12+** | Langage principal |
| **React 19 + Vite** | Interface utilisateur |
| **FastAPI** | API REST backend |
| **DuckDB 1.4** | Moteur de requêtes OLAP |
| **Polars 1.38** | DataFrames haute performance |
| **PyArrow 23** | Passerelle données |
| **Pydantic v2** | Validation des données |
| **Plotly** | Visualisations interactives |
| **SPNKr** | API Halo Infinite |

---

## Limitations connues

- **API Halo** : Dépend de l'API SPNKr — certains endpoints peuvent être instables ou limités en débit. Les stats par arme ne sont pas disponibles via l'API (vérifié 2026-02-02).

---

## Licence

Ce projet est sous licence MIT. Voir [LICENSE](../LICENSE) pour plus de détails.

---

## Remerciements

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr)
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt) et [OpenSpartan](https://github.com/OpenSpartan)

Voir aussi [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md).

---

**Fait avec passion pour la communauté Halo**
