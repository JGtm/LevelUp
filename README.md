# LevelUp - Halo Dashboard

> **Analyze your Halo 5: Guardians and Halo Infinite stats match by match, track your progress over time, and compare your performance with your squad.**

[![Version](https://img.shields.io/badge/Version-7.3.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.3.0)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Feedback issues](https://img.shields.io/github/issues-search/JGtm/LevelUp?query=label%3Afeedback%20is%3Aopen&label=feedback&color=0e8a16)](https://github.com/JGtm/LevelUp/issues?q=is%3Aissue+is%3Aopen+label%3Afeedback)

---

## What's new

**v7.3 — Overtime, first kill / first death & a repaired Achievements page**

A release focused on reading a match better: an overtime flag on every match that went past regulation time, a new first kill / first death chart on three pages, and a round of repairs on Halo 5, on the language switch and on the demo.

**Overtime**
- **A badge when the match went past regulation time** — in the match header and as a pill in the Explorer, with the extra time played
- **Your whole history at once** — the flag is computed as you read, so every past match is covered without waiting for a re-sync; Halo 5 stays deliberately unflagged until its regulation times are declared

**First kill / first death**
- **A new chart replacing the old histograms** — one lane per player, first kills above, first deaths below, with each median and the advance window between the two
- **On three pages** — Squad ("Dynamics" tab), Timeseries ("Progression" tab) and session detail, including the session comparison panel

**Halo 5**
- **Achievements page repaired** — the page returned an error instead of your milestones, and the section title now stays in place whether the grid is loading, empty or in error

**Reading in your language**
- **Commendations follow the language switch** — their labels used to stay in the previous language until you reloaded the page, and "Headshots" showed up under its raw field name

**Match view & demo**
- **A missing objective statistic no longer breaks the scoreboard** — the match no longer reads as incomplete; only the Objectives section stays empty
- **A more complete demo** — improvement campaigns and objective statistics are visible at last, and player identifiers are anonymized as well

## Features

### Track your career
- **Rank history** — LUSR and CSR rating per playlist over time, with your rank name at each step
- **Path to Hero** — projection chart showing how close you are to the Hero rank
- **Career KPI cards** — 8 cards at a glance: matches played, total time, frags, deaths, assists, accuracy, time alive, W/L/T/DNF bar — each color-coded against your all-time average
- **Commendations** — monitor your Halo commendations with medal grids and per-medal distributions
- **XP progression** — XP curve with multi-player comparison overlay
- **Objectives** — create individual or squad challenges (collective or competitive) on any Halo metric with configurable windows, tiers (Normal / Heroic / Legendary / Mythic), and narrative arcs; earn Prestige Points (PP) on completion
- **Prestige** — PP leaderboard in Palmares ranking you against your squad and relations; four tiers with color-coded badges

### Analyze your matches
- **Explorer** — browse all your matches with cascade filters (map, mode, playlist, outcome, date, session), partial match ID search, and encounter badges
- **Last Match** — full scoreboard with K/D, medals, weapons, performance score, impact badges, and encounter history panel for recurring opponents
- **Kill cadence** — kills by 15-second intervals for you and the enemy team, with moving-average overlay — see exactly when the pace shifted
- **Match intensity heatmap** — kill density per game phase (early/mid/late) across all your matches at a glance
- **Comeback badges** — *Remontada* (you were losing and came back), *Collapse* (you were winning and blew it), *Contre-Remontada* (you stopped the enemy comeback)
- **Session comparison** — side-by-side analysis of two play sessions
- **Activity heatmap** — win rate and activity by day of week and time slot

### Squad & Teammates
- **Unified squad view** — same rich charts for 1, 2, or 3 friends; works for all party sizes
- **Per-player intensity heatmap** — see each squad member's kill profile by game phase across shared matches
- **Squad records** — career bests for each member (K/D, kills, streaks…) with per-map breakdown
- **Synergy radar** — per-minute stats and complementarity across your squad
- **Kill cadence per player** — synchronized kill tempo across shared matches
- **Impact timeline** — narrative badges (Top Killer, Silent Hero, False Brother…) per match

### Clips & Media
- **Media library** — browse screenshots and video clips linked to their match; filter by owner, map, mode, outcome, or solo/squad context
- **Auto-indexing** — clips re-scanned automatically every few hours and after each sync
- **Manual reassociation** — fix a wrongly linked clip in one click: a built-in picker suggests matches around the capture timestamp (±15 / ±60 / ±180 min) with map thumbnails, outcome and full lobby

### Notifications & Setup
- **In-app notification center** — per-player feed with unread badge, category filters, day-grouped timeline, and bulk actions; 60-second live refresh; preferences per player
- **Discord alerts** — configurable notifications after sync and after backfill, independently
- **One-click setup** — Xbox Device Code login (`xbox.com/activate`) with automatic player provisioning; no Azure account required

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

**Prerequisites**: Go 1.26+, Node.js + npm, GNU Make, and Air for Go hot reload.

```bash
git clone https://github.com/JGtm/LevelUp.git
cd LevelUp
cd apps/web && npm install && cd ../..
go install github.com/air-verse/air@latest
make dev
```

Open http://localhost:5173 in your browser, then follow the in-app wizard.

Useful variants:

```bash
make go-api-dev
make web
```

**Detailed docs**: [docs/INSTALL.md](docs/INSTALL.md)

**French README**: [docs/FR/README.md](docs/FR/README.md)

---

## Configuration

**v6 — Zero configuration.** LevelUp bundles its own Azure client ID.
Just launch the app, enter your gamertag, and authenticate via Device Code Flow
(`https://xbox.com/activate`). No `.env.local` file or Azure account required.

### Refresh token (advanced / headless)

If you cannot use the interactive wizard (e.g. server/headless setup), open the login page
at `http://localhost:5173/auth/xbox/login` from any browser on the same network and follow the
Device Code Flow. Alternatively, configure a redirect URI via `LEVELUP_OAUTH_REDIRECT_URI`
for a fully browser-based flow.

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

## Documentation

| Document | Content |
|----------|---------|
| [INSTALL.md](docs/INSTALL.md) | Detailed installation guide |
| [CONFIGURATION.md](docs/CONFIGURATION.md) | Tokens and profiles configuration |
| [COMMANDS.md](docs/COMMANDS.md) | Common commands cheat sheet |
| [ARCHITECTURE_V6.md](docs/ARCHITECTURE_V6.md) | v6 architecture (shared matches + i18n assets) |
| [SYNC_GUIDE.md](docs/SYNC_GUIDE.md) | Sync guide |
| [BACKUP_RESTORE.md](docs/BACKUP_RESTORE.md) | Backup and restore |
| [testing.md](docs/testing.md) | Go testing strategy (CGO, coverage ratchet) |
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
| **Go 1.26+** | API backend |
| **React 19 + Vite** | Frontend UI |
| **TanStack Query / Router / Table** | Data fetching, routing, tables |
| **ECharts 5** | Interactive charts |
| **DuckDB 1.4+** | OLAP query engine |
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
