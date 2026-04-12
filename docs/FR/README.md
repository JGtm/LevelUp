# LevelUp - Dashboard Halo Infinite

> **Analysez vos performances Halo Infinite avec des visualisations avancées et une architecture DuckDB ultra-rapide.**

[![Version](https://img.shields.io/badge/Version-6.0.0-blue.svg)](https://github.com/JGtm/LevelUp_with_SPNKr/releases/tag/v6.0.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![Streamlit](https://img.shields.io/badge/Streamlit-1.28%2B-FF4B4B.svg)](https://streamlit.io/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Dernières nouveautés

- **v7.0 — Défis Mission Control, live + persistés** — la home V7 restaure les défis actifs via HaloStats `/decks`, avec badge Waypoint, titre/description localisés, vraie progression `x/y` et échéance du deck. Les états joueur sont historisés dans `challenge_snapshots`, les définitions partagées dans `metadata.duckdb`, avec stockage multi-langue normalisé BCP-47 et fallback `en-US`. Si `metadata.duckdb` est verrouillée, la home reste fonctionnelle en live et saute simplement la persistance sur ce refresh. **Nouveau — Notifications Discord pour les médias** : un embed Discord avec miniature GIF ou screenshot est envoyé à chaque nouveau fichier indexé ; chaque média n'est notifié qu'une fois (anti-spam via `discord_notified_at`) ; toggle `discord_notify_new_media` dans les Paramètres.

- **v6.2 — Badges narratifs, vue escouade unifiée & noms de modes normalisés** — badges Remontada/Débandade/Contre-remontada, vue coéquipiers unifiée (duo/trio/quatuor), graphe combiné Kills ↑ / Deaths ↓, et **normalisation des labels de modes** via un resolver d'affichage unique (`resolve_display_mode`) avec délégation dans `translate_pair_name` et 29 overrides FR/EN dans `mode_pair_overrides`.
- **v5.6 (bêta) — MSAL Device Code Flow & Extraction d’armes** — Le flux OAuth redirect est remplacé par le Device Code Flow MSAL : l’utilisateur entre un code sur xbox.com/activate, sans `client_secret` ni URI de redirection. **Kills par arme depuis les films SPNKr** *(bêta — couverture estimée 70–100 % selon les matchs, catalogue d’armes en cours)* ; kills par arme dans Match View et l’onglet Coéquipiers ; extraction automatique au sync. **Matrice d’Impact** — séparateurs verticaux entre matchs ; renommée depuis "Heatmap". **4 041 tests**.
- **v5.5 — Wizard & Xbox OAuth & macOS/Linux** — Wizard de configuration initiale avec deux parcours (Xbox Express / Azure manuel). Connexion Xbox en 1 clic. `LevelUp.sh` pour macOS/Linux (POSIX sh, Homebrew, APT…). Documentation réécrite. **75 nouveaux tests**. Page **Comparaison de sessions** entièrement revue : donuts résultats, courbe F/D + précision, highlights match, répartition modes/cartes, overlay LUSR/CSR sur le score cumulé. **Comparaison XP & rang Héros multi-joueurs** sur la page Carrière — précision variable selon l’historique de sync.
- **v5.4 — Explorer & Rencontres** — Page Explorer unifiée. Historique des rencontres. Refactoring massif (72 sous-modules). **3693 tests**, 0 failure.
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

### Architecture v6 - DuckDB Multi-DB
- **Shared Matches** — `shared_matches.duckdb` centralise tous les matchs (registry, participants, events, médailles)
- **PvE Firefight** — `shared_pve.duckdb` isole les stats Firefight (waves, boss, ennemis par type)
- **Couche de résolution v6** — vues SQL garanties : `v_gamertag_lookup`, `v_match_full`, `v_killer_victim_full`, `v_weapon_kills`
- **ATTACH multi-DB** — DuckDB ATTACH pour lecture transparente cross-DB
- **LUSR/CSR** — Ratings TrueSkill 2 per-groupe stockés dans `match_skill_rank` (player DB)
- **Performance** — Requêtes DuckDB < 30ms (warm), DataFrame Polars natifs, vues matérialisées
- **Zéro configuration** — client ID intégré ; authentification via xbox.com/activate en 2 étapes

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

### 2. Configurer votre Client ID Azure

```env
SPNKR_AZURE_CLIENT_ID=votre_client_id
```

> `client_secret` et `redirect_uri` ne sont plus nécessaires — l’authentification utilise le MSAL Device Code Flow.

### 3. Obtenir le refresh token (Device Code Flow)

```bash
python scripts/spnkr_get_refresh_token.py --device-code
# Ou utiliser le Wizard de configuration intégré (recommandé)
```

**Documentation détaillée** : [CONFIGURATION.md](CONFIGURATION.md)

---

## Architecture

### Structure des Données (v6)

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
| `shared_matches` | `weapon_kills` | Kills par arme par joueur par match (extraits des films SPNKr) |
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

Les contributions sont les bienvenues ! Voir [CONTRIBUTING.md](../CONTRIBUTING.md) pour les guidelines.

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

- **API Halo** : Dépend de l'API SPNKr — certains endpoints peuvent être instables ou limités en débit. Les stats par arme ne sont pas disponibles via l'API (vérifié 2026-02-02).

---

## Licence

Ce projet est sous licence MIT. Voir [LICENSE](../LICENSE) pour plus de détails.

---

## Remerciements

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) pour [SPNKr](https://github.com/acurtis166/SPNKr)
- **Den Delimarsky** ([dend](https://github.com/dend)) pour [Grunt](https://github.com/dend/grunt) et [OpenSpartan](https://github.com/OpenSpartan)

Voir aussi [ACKNOWLEDGMENTS.md](../ACKNOWLEDGMENTS.md).

---

**Fait avec passion pour la communauté Halo**
