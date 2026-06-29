# LevelUp - Halo Infinite Dashboard

> **Analyze your Halo Infinite stats match by match, track your progress over time, and compare your performance with your squad.**

[![Version](https://img.shields.io/badge/Version-7.0.0-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.0.0)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.4%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Feedback issues](https://img.shields.io/github/issues-search/JGtm/LevelUp?query=label%3Afeedback%20is%3Aopen&label=feedback&color=0e8a16)](https://github.com/JGtm/LevelUp/issues?q=is%3Aissue+is%3Aopen+label%3Afeedback)

---

## What's new

**v7.0 — New React app, Mission Control & multi-title support**

Major overhaul. LevelUp leaves Streamlit behind for a **React 19 + Go API** app with a **fully redesigned UX/UI** and **fully automated synchronization**. Close to 400 commits of work — here's what it changes for you:

**A completely rebuilt app — new UX/UI**
- **React 19 + Tailwind frontend** — instant navigation between pages, shareable URLs for every match/session/player, dark/light theme, full FR/EN i18n
- **Unified design system** — palette, typography, spacing and components (cards, buttons, modals, tooltips, carousels, lightbox) harmonized across the whole app; no more patchwork Streamlit pages
- **Two-level navigation** — L1 bar (Home / Synthesis / Explorer / Squad / Communauté / Ascension / Media / Help / Settings) plus contextual L2 tabs; everything is one click away, no more scrolling sidebar
- **Modern interactions** — page transitions, deep links (`?session=`, prev/next match), hovers, side drawers, visual feedback on every action, responsive UI
- **Go API backend** — Python → Go switch for a lighter server, faster startup, smaller memory footprint
- **Multi-title support** — the app now handles multiple Halo games (Halo 5 : Guardians, Infinite and beyond) via a **TitleSwitcher** in the nav bar; `levelup add-title` CLI command to register a new title
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

**Synthèse — career analytics hub**
- **Overview dashboard** — KDA, accuracy, damage dealt/taken, headshots, perfect kills and kill streaks in one card grid, each color-coded against your all-time average; local filters for experience (ranked / unranked), period, season, playlist and mode
- **Per-weapon kills** — full kill breakdown by weapon with count and percentage share
- **Solo vs squad** — bipolar chart comparing your key metrics when playing solo vs with your squad
- **Top performing weeks** — identify your peak play periods at a glance
- **Per-map & per-mode outcomes** — win / loss / tie breakdown for every map and mode you have played
- **Activity heatmap** — session frequency and kill density by day of week and time slot
- **Combat profile** — Offensive Conversion (OC) and Defensive Resistance (DR) evolution over time
- **Relations preview** — top teammates and most frequent opponents surfaced from your match history

**Media V2 — likes, Discord notifications, upload**
- **Persistent likes** — like your screenshots and clips directly from the grid, state kept across reloads
- **Smart grouping** — by favorites, session, or solo/squad context
- **Lighter grid** — native thumbnails, shared lightbox, heart icons for liked / unliked
- **Drag-and-drop upload** — add your manual captures straight from the Media page
- **Non-destructive scanning** — automatic background re-indexing with a dedicated `--captures-dir` option
- **Discord notifications for new media** — embed with a GIF or screenshot thumbnail whenever a new capture is indexed; anti-spam (each file notified once); toggle `discord_notify_new_media` in Settings
- **Manual reassociation with match suggestions** — built-in modal listing your matches in a ±15 / ±60 / ±180 min window around the capture, with map thumbnail, map · mode · playlist, local time + delta, outcome badge and full lobby per team; one click + confirm to fix any media linked to the wrong match

**Dedicated Match page & richer visualizations**
- **Clean URL per match** — `/players/{gamertag}/matches/{id}`, shareable, with prev/next match navigation
- **Tug-of-War timeline** — dynamic curve of team score swings
- **KD Timeline** — kills/deaths evolution by phase with a moving average
- **Impact Badges** — narrative badges (Top Killer, Silent Hero, False Brother, Comeback Champion…) computed per match
- **Encounters panel** — list of players you've crossed paths with in earlier matches
- **Combat Yield & Perfect Kills** — new metrics in the match view
- **V7 scoreboard** — higher info density: expected stats, skill rank, linked media, citations
- **Session comparison** — dedicated A/B page: pick any two sessions and compare KDA, performance score, Offensive Conversion / Defensive Resistance, outcome distribution and dominant playlist side by side

**Authentication**
- **Xbox sign-in (standard)** — the standard way to use LevelUp: browser-based Xbox SSO (`/auth/xbox/login` → Microsoft → callback), using SISU/Proof-of-Possession for stable sessions. Device Code is available as an alternative transport, and the redirect URI is configurable via `LEVELUP_OAUTH_REDIRECT_URI`. No registration step — your Xbox account is your identity.
- **Admin login (password)** — the instance administrator has a username/password account, created with the `admin` CLI (`create-admin` / `reset-password`). In Xbox mode, password login is reserved for admins.
- **Optional per-player local login** — non-standard: a specific player can be granted a username/password account (invitation-based registration via `/register`, or an opt-in password on an existing SSO account) for deployments that need it. This is not part of the standard flow.

**Xbox achievements & match events**
- **Xbox achievements sync** — your Xbox achievements are pulled automatically from the Halo API on every sync
- **Achievement tracker** — browse your full achievement list on the Career page: filter by unlocked / in-progress / not-started, track your Gamerscore (earned vs total), and filter by game for multi-title support
- **Highlight events** — binary parser of match films to extract all major events (medals, clutches, spawns)
- **Weapon kills backfill** — weapon used per frag reconstructed from the film (POV ~87 %)
- **Comeback badges for teammates** — Remontada / Collapse / Counter-Remontada computed for squad members synced alongside you

**Communauté — Hall of Fame, Relations & Face-à-face**
- **Multilingual Season Pass** — Battle Pass translations in 26 languages, tier images from GameCMS
- **Relations** — track all players you have crossed paths with: per-player stats, shared match history, alliance and rivalry patterns, career micro-leaderboard
- **Face-à-face** — dedicated 1v1 (or 1v1v1 mirror) comparison page: pit any two or three players head-to-head across Combat, Precision and Bilan metrics; encounter badges mark alliances and rivals from your shared match history

**Objectives & Prestige**
- **Objectives** — individual and squad challenge system: set personal goals or create squad challenges (collective or competitive) on any Halo metric with configurable windows, tiers, and narrative arcs; earn Prestige Points (PP) on completion; two evaluation modes (threshold / cumulative) and two creation modes (free / guided)
- **Prestige leaderboard** — PP ranking in Palmares comparing your score against your squad and relations; four tiers: Normal / Heroic / Legendary / Mythic

**Ascension — progression tracking & game profile**
- **Streak dashboard** — win streaks, loss streaks and kill streaks tracked over time, with your all-time personal records highlighted
- **Records & milestones** — all-time personal bests (best KDA, most kills in a match, longest win streak…) plus a milestone grid showing how close you are to the next target
- **6-axis game profile radar** — strengths and weaknesses mapped across Lethality, Precision, Resilience, Team Impact, Survivability and Consistency
- **Style badge** — computed play-style classification (Fragger, Support, Sniper…) derived from your match history
- **LUSR component breakdown** — each component of your rating visualized and explained so you know exactly what to improve to climb
- **Behavioral pattern detection** — automatic detection of tilt, fatigue, engagement plateaus and skill ceilings from your recent matches
- **Contextual patterns** — how your stats shift by mode, map and squad composition
- **Solo vs squad card** — side-by-side comparison of your playstyle when playing alone vs with teammates
- **Proactive coach** — a background engine analyses your progress after every sync and fires positive-only alerts in the notification center: new personal records, near-misses, LUSR tier approaching, milestone unlocks, sustained stat improvements and contextual strengths by map, mode and squad type

**In-app notifications**
- **Notification center** — per-player feed with unread badge in the nav bar, category filters, day-grouped timeline, bulk actions, and 60-second live refresh; preferences configurable per player in Settings

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

**Full release history**: [docs/RELEASE_NOTES.md](docs/RELEASE_NOTES.md)

---

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
