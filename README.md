# LevelUp - Halo Infinite Dashboard

> **Analyze your Halo Infinite stats match by match, track your progress over time, and compare your performance with your squad.**

[![Version](https://img.shields.io/badge/Version-7.0.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.0.0)
[![Python 3.12+](https://img.shields.io/badge/Python-3.12%2B-blue.svg)](https://www.python.org/downloads/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![FastAPI](https://img.shields.io/badge/FastAPI-0.110%2B-009688.svg)](https://fastapi.tiangolo.com/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![Polars](https://img.shields.io/badge/Polars-1.38%2B-blue.svg)](https://pola.rs/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## What's new

**v7.0 — New React app, Mission Control & multi-title support**

Major overhaul. LevelUp leaves Streamlit behind for a **React 19 + Go API** app with a **fully redesigned UX/UI** and **fully automated synchronization**. Close to 400 commits of work — here's what it changes for you:

**A completely rebuilt app — new UX/UI**
- **React 19 + Tailwind frontend** — instant navigation between pages, shareable URLs for every match/session/player, dark/light theme, full FR/EN i18n
- **Unified design system** — palette, typography, spacing and components (cards, buttons, modals, tooltips, carousels, lightbox) harmonized across the whole app; no more patchwork Streamlit pages
- **Two-level navigation** — L1 bar (Home / Synthesis / Explorer / Squad / Hall of Fame / Media / Help / Settings) plus contextual L2 tabs; everything is one click away, no more scrolling sidebar
- **Modern interactions** — page transitions, deep links (`?session=`, prev/next match), hovers, side drawers, visual feedback on every action, responsive UI
- **Go API backend** — Python → Go switch for a lighter server, faster startup, smaller memory footprint
- **Multi-title support** — the app now handles multiple Halo games (Infinite and beyond) via a **TitleSwitcher** in the nav bar; `levelup add-title` CLI command to register a new title
- **Built-in Help page** — release notes (in-app changelog) plus a glossary of Halo terms, with a local 24 h cache for offline reading

**Home "Mission Control" — fully redesigned**
- **Multi-title hero banner** — dynamic per-game banner with dedicated artwork
- **Live Battle Pass panel** — dedicated card with operation artwork, tier progression and upcoming reward, refreshed on every visit
- **Active challenges restored** — Mission Control challenge cards with real deck expiry, badge, localized title/description and your actual `x/y` progress in real time
- **Multi-language challenge catalog** — titles and descriptions stored in **26 languages** (BCP-47), with `en-US` fallback when the requested locale is missing
- **Clean historization** — your challenges are snapshotted into your player DB (`challenge_snapshots`), shared definitions live in `metadata.duckdb`
- **Live-first, failsafe** — if the metadata DB is locked, the Home still renders challenges live and simply skips persistence (no blocking)
- **Enriched match tiles** — KDA, squad, rank, headshots, citations, perf score and both team scores right on each tile
- **Sessions carousel** — color-coded FDA, dominant playlist/mode, hover tooltip
- **Liked-media tab** + recent matches carousel on the Home
- **Synthesis page** — new L1 section with weekly highlights, top rivalries and career summary stats

**Media V2 — likes, Discord notifications, upload**
- **Persistent likes** — like your screenshots and clips directly from the grid, state kept across reloads
- **Smart grouping** — by favorites, session, or solo/squad context
- **Lighter grid** — native thumbnails, shared lightbox, heart icons for liked / unliked
- **Drag-and-drop upload** — add your manual captures straight from the Media page
- **Non-destructive scanning** — automatic background re-indexing with a dedicated `--captures-dir` option
- **Discord notifications for new media** — embed with a GIF or screenshot thumbnail whenever a new capture is indexed; anti-spam (each file notified once); toggle `discord_notify_new_media` in Settings

**Dedicated Match page & richer visualizations**
- **Clean URL per match** — `/players/{gamertag}/matches/{id}`, shareable, with prev/next match navigation
- **Tug-of-War timeline** — dynamic curve of team score swings
- **KD Timeline** — kills/deaths evolution by phase with a moving average
- **Impact Badges** — narrative badges (Top Killer, Silent Hero, False Brother, Comeback Champion…) computed per match
- **Encounters panel** — list of players you've crossed paths with in earlier matches
- **Combat Yield & Perfect Kills** — new metrics in the match view
- **V7 scoreboard** — higher info density: expected stats, skill rank, linked media, citations

**Rebuilt authentication**
- **SISU/PoP provider** — new Xbox authentication with Proof-of-Possession for more stable sessions and fewer reconnects
- **Local auth** — username/password mode for single-user / LAN deployments
- **Invitation-based registration** — new `/register` page; account creation requires a server-issued invitation code (`?code=` query param); invalid or expired codes are rejected before the account is written

**Xbox achievements & match events**
- **Xbox achievements sync** — your Xbox achievements are pulled automatically from the Halo API on every sync
- **Highlight events** — binary parser of match films to extract all major events (medals, clutches, spawns)
- **Weapon kills backfill** — weapon used per frag reconstructed from the film (POV ~87 %)
- **Comeback badges for teammates** — Remontada / Collapse / Counter-Remontada computed for squad members synced alongside you

**Hall of Fame & Season Pass**
- **Multilingual Season Pass** — Battle Pass translations in 26 languages, tier images from GameCMS
- **Relations / Leaderboard** — new Hall of Fame pages: crossed players, per-player stats, career micro-leaderboard
- **Compare drawer** — redesigned session/player comparison UI

**Automated sync & real-time presence**
- **100 % automatic sync** — no more manual `python scripts/sync.py`: the app syncs your matches on its own, continuously in the background, as soon as a new game is played
- **Immediate end-of-match trigger** — as soon as a player finishes a match, the watcher fetches the stats without waiting for the next tick
- **Xbox RTA presence + Steam polling** — real-time online detection to know who's playing and sync at the right moment
- **Smart scheduler** — sync cadence adapts to player activity; no useless requests when nobody's playing
- **Autonomous token refresh** — no more interruptions: Halo tokens renew themselves in the background
- **Proactive reconnection** — status=3 handling with on-demand XSTS refresh, automatic reconnect on startup

**Settings & admin**
- **Settings auto-save** — settings persist immediately with an ephemeral visual indicator
- **Admin page** — supervision UI (auth provider, job status, privacy)
- **Browser preferences** — selected player, language and filters remembered across sessions
- **Configurable API endpoints** — Halo Stats, SPNKr, CMS… all configurable from Settings

**Assets & maps**
- **Cache-aside map images** — map artwork downloaded and cached locally, no more repeated external requests
- **`populate-assets` CLI** — Go command to pre-download every asset (maps, medals, Battle Pass tiers) ahead of offline use

**Colour accessibility**
- **Colour-blind safe palette** — a new Okabe-Ito palette (designed in 2008, universally recommended) is available in Settings → Accessibility; it replaces every colour in the app — charts, performance indicators, match outcomes, K/D ratings — with tones that remain distinguishable under deuteranopia, protanopia and tritanopia
- **Live preview** — the palette switches instantly across the whole app without a page reload; a swatch preview lets you compare before committing
- **Persistent preference** — your choice is saved in the browser and restored automatically on every visit

**v6.5 — Squad heatmap & hardened settings**
- **Per-player intensity heatmap** (Teammates) — new visualization: match × phase (early/mid/late) heatmap for every squad member. See who strikes early, who ramps up late. Toggle between "all together" and "player by player"
- **Separate Discord notifications** — sync and backfill alerts now have their own toggle each; disabling one doesn't affect the other
- **More robust settings** — preferences are written safely (atomic write + automatic backup); the file can no longer be corrupted on crash or forced shutdown
- **Fixes** — records (best performance) no longer appear by default when they had been disabled

**v6.4 — Media filters, squad CSR & reading aids**
- **Media library filters** — filter your clips and screenshots by owner (my captures / teammates / no linked match), map, mode, outcome (win / loss…) and solo vs squad context. Sort by date, map, mode or outcome in one click
- **Reading aids** — a sidebar checkbox shows or hides the ~45 explanatory callouts across the pages; disable it for a cleaner UI
- **Redesigned career summary** — 8 compact cards side by side: Matches, Total time, Frags, Deaths, Assists, Accuracy, Time alive, Outcomes. Each card compares your value to your all-time average (green/gold/red color code at ±8 %)
- **Win/Loss folded into Timeseries** — the Win/Loss page becomes a tab inside Timeseries; tabs renamed: Summary · Maps & Modes · Progression · Advanced
- **Automatic teammate CSR** — when syncing a ranked match, the rank of every registered co-player is pulled and distributed automatically — no need for each one to sync their own account anymore
- **Remontada/Collapse badges for teammates** — computed for squad members synced alongside you
- **Sticky legend panel** (Teammates) — a floating panel displays each squad member's color for the whole scroll through the squad section
- **Cross-session preferences** — selected player and language are remembered in the browser; filters survive updates

**v6.3 — Localized names, squad records & medal details**
- **Maps and modes in your language** — map, playlist and game mode names in French (or English) across every page: filters, tables, charts and winrate histogram
- **Medal descriptions on hover** — hover a medal in the scoreboard or the Citations section to read its description
- **All-time squad records** — the Teammates page shows career bests for each member (K/D, kills, streaks…) with per-player colored annotations and a per-map detail view
- **Top Killer badge** — shown on the Impact timeline for the first player to reach 10 kills in a match
- **Reworked first-kill/first-death histogram** — mirrored butterfly chart with 15-second buckets and real in-game time (pre-match countdown subtracted)
- **Improved Last Match view** — map and mode merged; MMR, Kills and Deaths cards also display the opposing team score with a color-coded gap; the performance score appears right next to the rating
- **Medals and citations in a 4-column grid** — clearer scoreboard; weapons in a 2-column grid with well-proportioned thumbnails
- **Partial match ID search** (Explorer) — type 3+ characters of a match ID to instantly filter the list
- **Kill cadence histogram** — new chart (Combat tab): kills by 15-second buckets for you and the enemy team, with per-team moving average
- **Match intensity heatmap** — visualize kill density per phase across all your matches
- **Auto-indexing media library** — your clips are re-scanned automatically in the background after each sync
- **Fixes** — Spartan Carnage citation corrected; correct names in Discord notifications; filter calendar with free navigation across years

**v6.2 — Comeback badges & unified squad view**
- **Remontada / Collapse / Counter-Remontada badges** — the app detects comeback scenarios in your history: *Remontada* (you were losing at mid-match and won), *Collapse* (you were winning and lost), *Counter-Remontada* (you stopped the enemy comeback)
- **Unified squad view** — 1-vs-1 and squad views merge; you get the same rich charts for 1, 2 or 3 friends
- **Kills ↑ / Deaths ↓ chart** — kills and deaths merged into a single mirrored chart per member, to compare K/D arcs at a glance
- **Consistent mode names** — game mode labels are now homogeneous across every page and every chart

**v6.1 — Faster sync, bugs fixed**
- **Sync ~30–40 % faster** — every synchronization completes noticeably faster
- **Correct rank names** — the displayed rank now matches the actual tier (e.g. "Lance Corporal Diamond 1")
- Fixes: performance scores and materialized views always up to date after sync

---

## Features

### Track your career
- **Rank history** — LUSR and CSR rating per playlist over time, with your rank name at each step
- **Path to Hero** — projection chart showing how close you are to the Hero rank
- **Career KPI cards** — 8 cards at a glance: matches played, total time, frags, deaths, assists, accuracy, time alive, W/L/T/DNF bar — each color-coded against your all-time average
- **Commendations** — monitor your Halo commendations with medal grids and per-medal distributions
- **XP progression** — XP curve with multi-player comparison overlay

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

### Notifications & Setup
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
| **React 19 + Vite** | Frontend UI |
| **FastAPI** | REST API backend |
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
