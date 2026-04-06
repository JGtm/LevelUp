# LevelUp - Halo Infinite Dashboard

> **Analyze your Halo Infinite performance with advanced visualizations and an ultra-fast DuckDB architecture.**

[![Version](https://img.shields.io/badge/Version-6.3.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v6.3.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![Streamlit](https://img.shields.io/badge/Streamlit-1.28%2B-FF4B4B.svg)](https://streamlit.io/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## What's new

**v6.3 — Localized names, squad records & medal details**
- **Maps and modes in your language** — map names, playlists and game modes now appear in French (or English) across every page: sidebar filters, match tables, charts, and the win-rate histogram
- **Medal descriptions on hover** — hover any medal in the Last Match grid or the Citations section to see its description
- **Squad all-time records** — the Teammates page now shows the career bests for each squad member (K/D, kills, streaks…) with color-coded per-player highlights and a per-map record breakdown
- **Top Killer badge** 🔫 — shown on the Impact timeline for the first player on the team to reach 10 kills
- **Time-to-first-kill/death histogram revamped** — mirrored butterfly chart with 15-second bins and real gameplay time (pre-game countdown excluded)
- **Win rate chart** now shows total match count per map and translated map names
- **Settings page redesigned** — reorganized into clear sections (General, Sync, Performance, Display) for easier navigation
- **Last Match page polished** — map and mode merged into one KPI card; the MMR, Kills, and Deaths cards now show the opponent team's score alongside your own, with a color-coded gap; the performance score appears directly next to your rating; all cards share a uniform height
- **Medals and citations in a 4-column grid** — the scoreboard now lays out medals and citations in 4 columns for a cleaner overview; weapons in 2 columns with correct thumbnail sizing; citation tooltips include the medal description
- **Match ID search in Explorer** — start typing any part of a match ID (3+ characters) to instantly filter your match list with live feedback
- **Compact language selector** — the language switcher now shows flag emojis inline; the app version is always displayed below the logo
- **Bug fixes** — Spartan Carnage citation type corrected (now properly counts killing-spree medals); Discord notifications now display correct map and playlist names; hero page backdrop overflow fixed

**v6.2 — Comeback badges & unified squad view**
- **Remontada / Collapse / Contre-Remontada badges** — the app now detects comeback narratives from your match history: *Remontada* (you were losing at the midpoint but came back to win), *Collapse* (you were winning and blew it), *Contre-Remontada* (you stopped the enemy's comeback). Run `--comeback-badges` to compute them
- **Unified squad view** — the 1-on-1 teammate comparison view has been removed; the squad view now works for 1, 2, or 3 friends, giving you the same rich charts regardless of party size
- **Kills ↑ / Deaths ↓ chart** — separate kills and deaths charts replaced by a single mirrored chart per squad member, making K/D arcs easier to compare at a glance
- **Game mode names normalized** — mode labels now go through a single display resolver (`resolve_display_mode`), with translation delegated by `translate_pair_name` and 29 FR/EN overrides in `mode_pair_overrides`, so naming stays consistent across pages and charts

**v6.1 — Sync faster, bugs fixed**
- **Sync ~30–40 % faster** — each sync now completes significantly quicker
- **Career rank names corrected** — displays the actual rank name (e.g. "Lance Corporal Diamond 1") instead of an approximation
- Bug fixes: performance scores and materialized views now always update correctly

---

## Features

### Advanced stats
- **Interactive dashboard** - Explore your stats in real time
- **Detailed charts** - K/D trend, accuracy, time alive, kill streaks
- **Map analysis** - Per-map performance with heatmaps
- **Teammates** - Stats with your friends (same team or opponents)
- **Play sessions** - Automatic session detection with performance metrics

### Visualizations
- **Radar charts** - Per-minute stats and overall performance
- **Heatmaps** - Win rate by day/time of week
- **Distributions** - Histograms for accuracy, kills, scores
- **Correlations** - Scatter plots (time alive vs kills)
- **Top weapons** - Weapon stats with headshot rate

### Pages & navigation
- **Explorer** - Unified match search with cascade filters, fuzzy gamertag lookup, OS-style table, deep linking and encounter badges
- **Last match** - Full scoreboard for your latest game, searchable by match ID — with Encounter History panel
- **Session comparison** - Side-by-side analysis of two play sessions
- **Career progression** - Rank history, XP curve, Hero rank projections and **multi-player XP comparison** (all players with a refresh token overlaid on the same chart — detail level varies by sync history)
- **Commendations** - Track your commendations with medal distributions and grids
- **Media library** - Index and browse clips/screenshots linked to their matches
- **Discord notifications** - Automatic alerts after sync and backfill operations
- **Setup Wizard** - Guided first-time setup with Xbox Express or Azure manual paths
- **Xbox OAuth** - One-click Xbox login with automatic player provisioning

### v5.5 architecture — DuckDB Multi-DB
- **Shared Matches** — `shared_matches.duckdb` centralizes all matches (registry, participants, events, medals)
- **PvE Firefight** — `shared_pve.duckdb` isolates Firefight stats (waves, bosses, enemies by type)
- **Multi-DB ATTACH** — DuckDB `ATTACH` for seamless cross-DB reads
- **LUSR/CSR** — TrueSkill 2 ratings per group stored in `match_skill_rank` (player DB)
- **Performance** — DuckDB queries < 30ms (warm), native Polars DataFrames, materialized views
- **Device Code Flow (MSAL)** — Token acquisition via xbox.com/activate, no redirect URI or client secret; refresh token stored in player DB
- **Setup Wizard** — Guided configuration with auto-detection of missing credentials/players

---

## Screenshots

### Overview

![Main dashboard](docs/screenshots/main.png)

*Main dashboard: multi-page navigation and real-time interactive charts.*

![Sidebar, Time to First Kill & Performance](docs/screenshots/Sidebar-first-kill-performance.png)

*Advanced filters (type, playlist, mode, map,session/period), Time-to-First-Kill vs First Death distribution, and per-match performance score.*

---

### Performance & Combat

| KDA | Cumulative performance & trend |
|:-:|:-:|
| ![KDA](docs/screenshots/kda.png) | ![Cumulative performance & trend](docs/screenshots/cumulative-perf.png) |

![Average lifespan & Combat Skills](docs/screenshots/avg-lifespan-perfect-kills.png)


*K/D ratio with trend, cumulative performance score, average lifespan and combat skills.*

---

### Distributions & Correlations

| Distributions | Correlations |
|:-:|:-:|
| ![Distributions](docs/screenshots/distributions.png) | ![Correlations](docs/screenshots/correlations.png) |

*Histograms for accuracy/kills/scores with means and medians — scatter plots (time alive vs kills, etc.).*

---

### Activity by day & time

![Heatmap Top Week](docs/screenshots/heatmap-top-week.png)

*Win rate and activity heatmap by day of week and time slot.*

---

### Last match details

| Last match | Scoreboard |
|:-:|:-:|
| ![Summary](docs/screenshots/last-match.png) | ![Scoreboard Commendations](docs/screenshots/scoreboard.png) |
| Impact & Dominance | Antagonists |
| ![Impact & Dominance](docs/screenshots/impact-dominance.png) | ![Antagonists](docs/screenshots/antagonist.png) |

*Full scoreboard for your latest game (searchable by match ID) — and your most formidable rivals, MVP/LVP, scoreboard, commendations (Halo 5 inspired) grid and medal distributions.*

---

### Squad sessions & Teammates

| Squad overview | Session stats |
|:-:|:-:|
| ![Session history](docs/screenshots/history.png) | ![Squad complementarity](docs/screenshots/per-minute-complementarity.png) |
| **Teammates performance** | **Squad ranking** |
| ![Squad performance](docs/screenshots/performance-spree.png) | ![Squad ranking](docs/screenshots/teammate-heatmap.png) |

*Filter your sessions by squad: compare your stats when playing with friends and see how you and your teammates perform across shared matches.*

---

### Career progression, Ranks & Path to Hero

| Career | Ranks (LUSR/CSR) |
|:-:|:-:|
| ![Career](docs/screenshots/career.png) | ![Ranks](docs/screenshots/LUSRs.png) |
| ![Path to Hero](docs/screenshots/path-hero.png) | ![Memorable Matches](docs/screenshots/memorable-matches.png) |

*Rank history, progression to Hero, LUSR/CSR per playlist group*

---

### Explorer & Encounter History

| Explorer | Encounter History |
|:-:|:-:|
| ![Explorer](docs/screenshots/explorer.png) | ![Encounter History](docs/screenshots/encounters.png) |

*Browse and filter all your matches in detail with the Explorer, including search by player — track recurring opponents and cross-match encounter patterns with the Encounter History view.*

---

### Media library & Commendations

| Media library | Commendations |
|:-:|:-:|
| ![Media library](docs/screenshots/media-library.png) | ![commendations](docs/screenshots/commendations.png) |

*Browse and search your clips and screenshots linked to their matches (still in beta) — track your commendations with medal grids and distributions.*

---

## Quick start

**Prerequisites**: Python 3.12+ recommended (3.10 minimum).

### Windows (no technical knowledge required)

```
1. Download and extract the ZIP (or git clone)
2. Double-click LevelUp.bat
   → Python is installed automatically if missing
   → .venv created, dependencies installed
   → browser opens at http://localhost:8501
3. In the browser: enter your gamertag
4. Go to https://xbox.com/activate and enter the displayed code
   → LevelUp retrieves your profile and starts the initial sync
```

**No Azure configuration required** — the app bundles its own client ID.

### macOS / Linux

```bash
git clone https://github.com/JGtm/LevelUp.git
cd LevelUp
python3 -m venv .venv && source .venv/bin/activate
pip install -e ".[spnkr]"
python launcher.py run
```

Then follow the in-browser wizard (same 2-step flow).

**Detailed docs**: [docs/INSTALL.md](docs/INSTALL.md)

**French README**: [docs/FR/README.md](docs/FR/README.md)

---

## Configuration

**v6 — Zero configuration.** LevelUp bundles its own Azure client ID.
Just launch the app, enter your gamertag, and authenticate via Device Code Flow
(`https://xbox.com/activate`). No `.env.local` file or Azure account required.

### Refresh token (advanced / headless)

If you cannot use the interactive wizard (e.g. server/headless setup):

```bash
python scripts/spnkr_get_refresh_token.py --device-code
```

This prints a code to enter at `https://xbox.com/activate`, then saves the token
automatically to `.env.local`.

### Note for forks / developers

The bundled `LEVELUP_CLIENT_ID` is an Azure App Registration tied to this project.
**If you fork LevelUp**, please create your own free Azure App Registration
(see [docs/CONFIGURATION.md](docs/CONFIGURATION.md)) and set:

```env
# .env.local
SPNKR_AZURE_CLIENT_ID=your_own_client_id
```

This env var takes precedence over the bundled ID.

**Full configuration reference**: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

---

## Architecture

### Data layout (v6)

```
data/
├── warehouse/
│   ├── metadata.duckdb            # Shared reference data (maps, playlists, medals)
│   ├── shared_matches.duckdb      # All matches (registry, participants, events, medals)
│   └── shared_pve.duckdb          # PvE Firefight stats (pve_match_stats) — v5.2
├── players/                       # Per-player enrichments (~4 MB/player)
│   └── {gamertag}/
│       ├── stats.duckdb
│       │   ├── player_match_enrichment  # performance_score, session_id (ONLY match table)
│       │   ├── antagonists, match_citations, career_progression
│       │   └── match_skill_rank         # LUSR or CSR rating per match — v5.3
│       └── archive/               # Parquet archives
└── backups/                       # Parquet backups
```

### Main DuckDB tables

| Database | Table | Description |
|----------|-------|-------------|
| `shared_matches` | `match_registry` | Central registry (1 row per match) |
| `shared_matches` | `match_participants` | All player stats (31 cols, incl. MMR) |
| `shared_matches` | `medals_earned`, `highlight_events` | Medals and recorded highlight events |
| `shared_matches` | `weapon_kills` | Weapon kills per player per match (extracted from SPNKr films) |
| `shared_pve` | `pve_match_stats` | Firefight stats per player/match |
| player `stats` | `player_match_enrichment` | performance_score, session_id |
| player `stats` | `match_skill_rank` | LUSR/CSR rating per match |
| player `stats` | `mv_map_stats`, `mv_global_stats` | Materialized views |

**Technical docs**: [docs/ARCHITECTURE_V6.md](docs/ARCHITECTURE_V6.md)

---

## Documentation

| Document | Content |
|----------|---------|
| [INSTALL.md](docs/INSTALL.md) | Detailed installation guide |
| [CONFIGURATION.md](docs/CONFIGURATION.md) | Tokens and profiles configuration |
| [COMMANDS.md](docs/COMMANDS.md) | Common commands cheat sheet |
| [ARCHITECTURE_V6.md](docs/ARCHITECTURE_V6.md) | v6 architecture (shared matches + i18n assets) |
| [SYNC_GUIDE.md](docs/SYNC_GUIDE.md) | Sync guide |
| [BACKUP_RESTORE.md](docs/BACKUP_RESTORE.md) | Backup and restore |
| [TESTING_V5.md](docs/TESTING_V5.md) | v5 testing strategy |
| [FAQ.md](docs/FAQ.md) | Frequently asked questions |
| [COMMENDATIONS.md](docs/COMMENDATIONS.md) | Commendations system (architecture & usage) |
| [COMMENDATIONS_REFERENCE.md](docs/COMMENDATIONS_REFERENCE.md) | Full commendations reference |

French docs: [docs/FR/](docs/FR/)

Archived docs (not translated): [docs/archive/](docs/archive/)

---

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for guidelines.

---

## Tech stack

| Technology | Usage |
|------------|-------|
| **Python 3.12+** | Main language |
| **Streamlit** | UI |
| **DuckDB 1.4** | OLAP query engine |
| **Polars 1.38** | High-performance DataFrames |
| **PyArrow 23** | Data interoperability |
| **Pydantic v2** | Data validation |
| **Plotly** | Interactive charts |
| **SPNKr** | Halo Infinite API |

---

## Known limitations

- **Halo API**: Depends on SPNKr — some endpoints can be unstable or rate-limited. Weapon kills are extracted from match film binary data (SPNKr), not from the stats API; POV coverage ~87.5 %.

---

## License

This project is licensed under MIT. See [LICENSE](LICENSE) for details.

---

## Acknowledgements

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) for [SPNKr](https://github.com/acurtis166/SPNKr)
- **Den Delimarsky** ([dend](https://github.com/dend)) for [Grunt](https://github.com/dend/grunt) and [OpenSpartan](https://github.com/OpenSpartan)

See also [ACKNOWLEDGMENTS.md](docs/ACKNOWLEDGMENTS.md).

---

**Made with passion for the Halo community**
