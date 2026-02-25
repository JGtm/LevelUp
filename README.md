# LevelUp - Dashboard Halo Infinite

> **Analysez vos performances Halo Infinite avec des visualisations avancées et une architecture DuckDB v5 ultra-rapide.**

[![Version](https://img.shields.io/badge/Version-5.3.0-green.svg)](https://github.com/JGtm/LevelUp_with_SPNKr/releases/tag/v5.3.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![Streamlit](https://img.shields.io/badge/Streamlit-1.28%2B-FF4B4B.svg)](https://streamlit.io/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

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
- **Zéro legacy** — 0 SQLite, 0 Pandas dans le code métier, Backup Parquet Zstd

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

**Documentation détaillée** : [docs/INSTALL.md](docs/INSTALL.md)

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

**Documentation détaillée** : [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

---

## Utilisation

### Lancer le Dashboard

```bash
# Mode interactif
python launcher.py

# Lancer directement
python launcher.py run

# Avec synchronisation
python launcher.py run+refresh --player MonGamertag --delta
```

### Synchronisation des Données

```bash
# Sync incrémentale (nouveaux matchs uniquement)
python scripts/sync.py --delta --gamertag MonGamertag

# Sync complète
python scripts/sync.py --full --gamertag MonGamertag --max-matches 500
```

### Backup et Restore

```bash
# Backup d'un joueur
python scripts/backup_player.py --gamertag MonGamertag

# Restauration
python scripts/restore_player.py --gamertag MonGamertag --backup ./backups/MonGamertag
```

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

**Documentation technique** : [docs/ARCHITECTURE_V5.md](docs/ARCHITECTURE_V5.md)

---

## Documentation

| Document | Contenu |
|----------|---------|
| [INSTALL.md](docs/INSTALL.md) | Guide d'installation détaillé |
| [CONFIGURATION.md](docs/CONFIGURATION.md) | Configuration des tokens et profils |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Architecture technique (v4 legacy) |
| [ARCHITECTURE_V5.md](docs/ARCHITECTURE_V5.md) | Architecture v5 (shared matches) |
| [DATA_ARCHITECTURE.md](docs/DATA_ARCHITECTURE.md) | Architecture des données |
| [SHARED_MATCHES_SCHEMA.md](docs/SHARED_MATCHES_SCHEMA.md) | Schéma shared_matches.duckdb |
| [SQL_SCHEMA.md](docs/SQL_SCHEMA.md) | Schémas DuckDB complets |
| [SYNC_GUIDE.md](docs/SYNC_GUIDE.md) | Guide de synchronisation |
| [SYNC_OPTIMIZATIONS_V5.md](docs/SYNC_OPTIMIZATIONS_V5.md) | Optimisations sync v5 |
| [MIGRATION_V4_TO_V5.md](docs/MIGRATION_V4_TO_V5.md) | Guide de migration v4→v5 |
| [CLEANUP_V5.md](docs/CLEANUP_V5.md) | Nettoyage post-migration v5 |
| [BACKUP_RESTORE.md](docs/BACKUP_RESTORE.md) | Backup et restauration |
| [TESTING_V5.md](docs/TESTING_V5.md) | Stratégie de tests v5 |
| [FAQ.md](docs/FAQ.md) | Questions fréquentes |

---

## Tests

```bash
# Suite complète (inclut les tests smoke pages/filtres/visualisations)
python -m pytest

# Suite stable hors intégration (recommandé au quotidien)
python -m pytest -q --ignore=tests/integration

# Avec couverture
python -m pytest --cov=src --cov-report=html

# Tests spécifiques
python -m pytest tests/test_duckdb_repository.py -v

# E2E navigateur réel (optionnel, Playwright)
# Désactivé par défaut ; activation explicite avec --run-e2e-browser
python -m pytest tests/e2e/test_streamlit_browser_e2e.py -v --run-e2e-browser
```

---

## Docker

```bash
# Construire et démarrer
docker compose up --build

# En arrière-plan
docker compose up -d

# Arrêter
docker compose down
```

Le dashboard est accessible sur `http://localhost:8501`.

L'image installe toutes les dépendances via `pyproject.toml` (y compris SPNKr pour la synchronisation API). Au runtime, `docker-compose.yml` monte :
- `./data` → `/app/data` — données DuckDB v4 (lecture/écriture)
- `./db_profiles.json` → `/app/db_profiles.json` — profils joueurs
- `./app_settings.json` → `/app/app_settings.json` — paramètres

Pour forcer une base précise, décommentez dans `docker-compose.yml` :

```yaml
environment:
  - OPENSPARTAN_DB=/app/data/players/MonGamertag/stats.duckdb
```

**Documentation Docker détaillée** : [docs/INSTALL.md](docs/INSTALL.md#installation-docker)

---

## Contribution

Les contributions sont les bienvenues ! Voir [CONTRIBUTING.md](CONTRIBUTING.md) pour les guidelines.

```bash
# Format du code
ruff check --fix .
black .
isort .

# Avant de commiter
pytest
```

---

## Stack Technique

| Technologie | Usage |
|-------------|-------|
| **Python 3.12+** | Langage principal |
| **Streamlit** | Interface utilisateur |
| **DuckDB 1.4** | Moteur de requêtes OLAP |
| **Polars 1.38** | DataFrames haute performance |
| **PyArrow 23** | Passerelle données |
| **Pydantic v2** | Validation des données |
| **Plotly** | Visualisations interactives |
| **SPNKr** | API Halo Infinite |

---

## Limitations connues

- **Pandas résiduel** : Pandas conservé uniquement aux frontières Plotly/Streamlit (conversion `.to_pandas()` au dernier moment). Polars est le standard pour tout le code métier.
- **Couverture tests** : ~43% global — les modules UI Streamlit tirent la moyenne vers le bas. Les modules métier (sync, repositories, analysis) dépassent individuellement 70%.
- **API Halo** : Dépend de l'API SPNKr — certains endpoints peuvent être instables ou limités en débit. Les stats par arme ne sont pas disponibles via l'API (vérifié 2026-02-02).

---

## Licence

Ce projet est sous licence MIT. Voir [LICENSE](LICENSE) pour plus de détails.

---

## Remerciements

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr)
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt) et [OpenSpartan](https://github.com/OpenSpartan)

Voir aussi [ACKNOWLEDGMENTS.md](ACKNOWLEDGMENTS.md).

---

**Fait avec passion pour la communauté Halo**
