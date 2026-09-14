# LevelUp - Halo Dashboard

> **Analyze your Halo 5: Guardians and Halo Infinite stats match by match, track your progress over time, and compare your performance with your squad.**

[![Version](https://img.shields.io/badge/Version-7.3.2-blue.svg)](https://github.com/JGtm/LevelUp/releases/tag/v7.3.2)
[![Go](https://img.shields.io/badge/Go-1.26%2B-00ADD8.svg)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://react.dev/)
[![DuckDB](https://img.shields.io/badge/DuckDB-1.5%2B-FEE14E.svg)](https://duckdb.org/)
[![ECharts](https://img.shields.io/badge/ECharts-5-AA344D.svg)](https://echarts.apache.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Feedback issues](https://img.shields.io/github/issues-search/JGtm/LevelUp?query=label%3Afeedback%20is%3Aopen&label=feedback&color=0e8a16)](https://github.com/JGtm/LevelUp/issues?q=is%3Aissue+is%3Aopen+label%3Afeedback)

---

## What's new

**v7.5 — 2D replay, the Theater film decoded & a Tactics tab**

The biggest release so far. Halo records a film of every match; until now the app never opened it. It does now — a match can be watched again from above, second by second, with everything the film knows written around it: who killed whom with what, at what range, who held the flag, who took the power weapon, who wasted the equipment they were carrying when they died.

**2D replay**
- **Every match, watched again from above** — all the players move on the real map background, under their own names, with a playback bar, four timeline tracks (You, Allies, Dominance, Media), speed control and 10-second jumps
- **A kill feed that follows the cursor** — killer, weapon, victim, medal and assist with its damage share; a death nobody is credited for says so rather than inventing a killer
- **Layers you switch on and off** — aim cone, trails, shot and kill effects, heat map, weapon spots, weapons on the ground, deployed equipment, vehicles, named zones, and the live objectives (flag, skull, hill, strongholds, bomb, VIP crown)
- **The game's own sound**, muted by default and filtered by category, plus a PNG capture and a video recording of the replay with its soundtrack

**What the film knows, and the API never said**
- **The weapon behind every elimination**, including the deaths with no firearm at all (repulsor, fall, the environment), and where on the map people kill and die
- **Engagement range** — low (p10), median and high (p90) per weapon, with the signed height difference
- **Weapon levels** — pad pickups split into starting, map and power weapons; what a spot is worth comes from the map, never from the weapon's name
- **Net flag grabs**, Assault bomb statistics, and equipment read as used, kept or dropped on death
- **Rounds counted as rounds** — on modes decided by rounds the score shown is the rounds won and lost, because the API's point score can hand the advantage to the side that lost

**Tactics tab**
- **The maps you play**, their record, and an analysis view on the map plan: where you spend your time, where you die, where you kill, where you die isolated, where wins and losses part ways, and the routes you take
- **A cell opens the replay at the exact moment** it happened

**Squad, sessions and match view**
- **"The shapes you keep"** — six ways of reading a squad across nineteen cards, plus the equipment used / kept / wasted block on the Synthesis, the Squad and the Sessions
- **Per-match cadence** everywhere, distance per weapon, frag distribution in two levels, and a score-over-time curve that respects the mode

**Maps, media and repairs**
- **109 map backgrounds** with Forge zones named 100 %, official callouts, medal images refreshed from the game catalog, likes per viewer
- **The World ranking page is repaired** and can no longer degrade in silence; a corrupted Halo 5 LUSR rating is fixed at the source

Everything read from a film is Halo Infinite only — Halo 5 keeps its own pages and never borrows another title's data.

## Features

> Everything read from the Theater film — the 2D replay, the Tactics tab, weapon per kill, engagement range, weapon levels, equipment usage, net flag grabs, Assault statistics — is **Halo Infinite only**, gated behind fine-grained capabilities. Halo 5 keeps its own pages and never borrows another title's data.

### Watch your match again — 2D replay
- **The whole match, from above** — every player moves on the real map background, under their own name, from kickoff to the final whistle, decoded from the Theater film the game records
- **Playback bar with four tracks** — You, Allies, Dominance and Media on one timeline, with round pills, inter-round messages, a play/pause, 10-second jumps, a speed menu and keyboard shortcuts
- **Kill feed synced to the cursor** — killer, weapon icon, victim, medal, and the assist with its damage share; a death nobody is credited for says so instead of inventing a killer, and says *what* you died of
- **One card per player** — weapons carried, grenades, shield, active camo and overshield, grappling hook, time spent dead, and a watermark while the player is carrying an objective
- **25 layers you switch on and off** — aim cone, trails, muzzle flashes, shot and kill effects, grenade flights and explosions, melee stars, heat map (time spent or eliminations), weapon spots and power-up pads, weapons left on the ground with their exact ammunition, deployed equipment (wall, rift, sensor, translocator…), dropped objects, vehicles, named zones, off-screen chevrons
- **Live objectives** — flag carries and the return zone, Oddball skull, King of the Hill ownership gauge, Strongholds A/B/C, the Assault bomb with its fuse countdown, the VIP crown
- **Score band** — the score of the round in progress, the victory target, and the score curve of the mode
- **The game's own sound** — 177 sounds taken from the game (weapons, grenades, melee, equipment, objectives, announcer, end-of-match fanfare), muted by default, filtered by category
- **Zoom, pan and framing** — steps, wheel, keyboard or a directional pad; the canvas takes the whole block and zooming never crops the map
- **Capture and record** — a PNG in one click, or a full video export of the replay with its mixed soundtrack, encoded away from real time
- **You get in from everywhere** — the match view, the home tiles, the Explorer and a Tactics cell, which opens the replay at the exact second

### Read a match
- **Full scoreboard** — K/D, medals, weapons, performance score, impact badges, and an encounter history panel for recurring opponents
- **Weapon behind every elimination** — read from the film's damage source instead of guessed from the scoreboard, including the deaths with no firearm at all: repulsor, explosion, fall, out of map
- **"Where it plays out"** — kill and death positions on the map, drawn as a plan when the map has no frozen image
- **Distance per weapon**, frag distribution in two levels, and special weapon control broken down by weapon level (starting / map / power weapons, power-up pads)
- **Objectives section** — one column per statistic of the mode played, with a team total, for Capture the Flag, Strongholds, King of the Hill, Oddball, Stockpile, Extraction, VIP and Assault
- **Rounds counted as rounds** — on modes decided by rounds the score shown is the rounds won and lost, with the API point score kept alongside because it can hand the advantage to the side that lost
- **Overtime badge** — a match that went past regulation time is flagged, with the extra time played
- **Kill cadence** — kills by 15-second intervals for you and the enemy team with a moving-average overlay, plus a tug-of-war curve and cumulative K/D
- **Comeback badges** — *Remontada*, *Collapse* and *Contre-Remontada*
- **Chronology and media tabs** — the match's events in order, and the clips and screenshots attached to it

### Track your career
- **Rank history** — LUSR and CSR rating per playlist over time, with your rank name at each step
- **Path to Hero** — projection chart showing how close you are to the Hero rank
- **Career KPI cards** — 8 cards at a glance: matches played, total time, frags, deaths, assists, accuracy, time alive, W/L/T/DNF bar — each color-coded against your all-time average
- **Commendations** — monitor your Halo commendations with medal grids and per-medal distributions
- **Medals** — the full medal catalog with your counts, images served from the game's own catalog
- **Season pass** — tier progression with the reward carousel and a content summary
- **XP progression** — XP curve with multi-player comparison overlay
- **Rivals and encounters** — who you meet, who you beat, and who beats you
- **Highlight matches** — your standout matches, filterable by ranked / unranked

### Analyze your matches
- **Explorer** — browse all your matches with cascade filters (map, mode, playlist, outcome, date, session), partial match ID search, encounter badges, a briefing strip and a combat profile; also works on **another player** to scout them before a match
- **Summary** — bipolar chart, outcomes by group, top weeks, weapon accuracy, activity heat map, and the **engagement range** section: low (p10) / median / high (p90) range per weapon with the signed height difference
- **Time series** — KDA trend, density and bars, distribution histograms, skill progression, engagement gap, and first kill / first death on one lane per player
- **Sessions** — per-session detail with frags, damage, intensity profile, mode breakdown, placement, net lives, participation, MMR dumbbell, career XP, and the equipment usage block
- **Session comparison** — side-by-side analysis of two play sessions
- **Activity heatmap** — win rate and activity by day of week and time slot

### Equipment, weapons and objectives — used, kept or wasted
- **The three outcomes of every piece of equipment** — used, kept without ever using it, or dropped when you died, family by family, with the remaining charges
- **On three pages** — Summary, Squad and Sessions, from one shared block: counts and shares variants, parity line, match-by-match regularity band and a lobby track showing who actually picks up the power weapons
- **Weapon levels** — pad pickups split into starting weapons, map weapons, power weapons and power-up pads; what a spot is worth comes from the map, never from the weapon's name
- **Net flag grabs** — juggling folded into a 1.5-second window, so picking your own flag back up twice in a second counts once
- **Objectives by role and family** — take, defend, hold, across Flag, King of the Hill, Strongholds and Oddball

### Squad & Teammates
- **Unified squad view** — same rich charts for 1, 2, or 3 friends; works for all party sizes
- **"The shapes you keep"** — six ways of reading a squad across nineteen cards and three blocks (equipment, special weapons, objectives), solo against squad, your share against the parity of your team
- **The exchange in six cards** — who gives and who receives, the delay, the matrix, the session rate and the count, with assists as stacked bars
- **Per-player intensity heatmap** — each squad member's kill profile by game phase across shared matches
- **Squad records** — career bests for each member (K/D, kills, streaks…) with per-map breakdown
- **Synergy radar** — per-minute stats and complementarity across your squad
- **Isolation cloud** — where and how often a member dies away from the group
- **Kill cadence per player** — synchronized kill tempo across shared matches
- **Impact timeline** — narrative badges (Top Killer, Silent Hero, False Brother…) per match
- **Objective panels** — the squad's objective statistics, per mode and per role

### Ascension — profile, objectives, coaching, achievements, tactics
- **Profile** — your combat axes, streaks, records timeline, behavior alerts, levers to pull, pattern context and improvement campaigns
- **Objectives** — create individual or squad challenges (collective or competitive) on any Halo metric with configurable windows, tiers (Normal / Heroic / Legendary / Mythic), and narrative arcs; earn Prestige Points (PP) on completion
- **Coaching** — proposals built from your own measured patterns, not from a generic checklist
- **Achievements** — milestone grid per title
- **Tactics** — a grid of the maps you play with their record, and an analysis view on the map plan answering six questions: where you spend your time, where you die, where you kill, where you die isolated, where wins and losses part ways, and the routes you take. Four KPI tiles (matches retained, coverage, exchange, isolated deaths), a team coordination card whose radius is read per match from the variant reference, and a cell click that opens the replay at the exact instant

### Community
- **Leaderboards** — the world ranking alongside the local one, with an honest empty state when a ranking could not be retrieved
- **Prestige** — PP leaderboard ranking you against your squad and relations; four tiers with color-coded badges
- **Relations** — who you play with and against, win rate donuts, rivalry cards, split bars and a moments heat map
- **Head-to-head** — a mirror comparison of two players, stat by stat

### Clips & Media
- **Media library** — browse screenshots and video clips linked to their match; filter by owner, map, mode, outcome, or solo/squad context
- **Auto-indexing** — clips re-scanned automatically every few hours and after each sync
- **Manual reassociation** — fix a wrongly linked clip in one click: a built-in picker suggests matches around the capture timestamp (±15 / ±60 / ±180 min) with map thumbnails, outcome and full lobby
- **Likes per viewer** — your like is yours, not the account's
- **In the replay** — your captures sit on their own timeline track, at the second they were taken

### Notifications & Setup
- **In-app notification center** — per-player feed with unread badge, category filters, day-grouped timeline, and bulk actions; 60-second live refresh; preferences per player
- **Discord alerts** — configurable notifications after sync, after backfill and when replays are ready, independently
- **One-click setup** — Xbox Device Code login (`xbox.com/activate`) with automatic player provisioning; no Azure account required
- **Spartan customizer** — your armour and colours, recoloured live
- **Multi-title** — Halo Infinite and Halo 5: Guardians side by side, each with its own storage, its own catalogs and its own capabilities

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

**Prerequisites**: Go 1.26+, Node.js + npm, GNU Make, Air for Go hot reload, and a **C toolchain** — the DuckDB driver is CGO, and on Windows a `gcc` missing from the `PATH` makes Air silently serve a stale binary (MinGW/UCRT64; see [docs/INSTALL.md](docs/INSTALL.md)).

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

**Zero configuration.** LevelUp bundles its own Azure client ID.
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
| [INSTALL.md](docs/INSTALL.md) | Detailed installation guide (including the C toolchain for CGO) |
| [CONFIGURATION.md](docs/CONFIGURATION.md) | Tokens and profiles configuration |
| [COMMANDS.md](docs/COMMANDS.md) | Common commands cheat sheet |
| [FAQ.md](docs/FAQ.md) | Frequently asked questions |
| [ARCHITECTURE_V6.md](docs/ARCHITECTURE_V6.md) | Current architecture (shared matches, per-title isolation, i18n assets) |
| [FOUNDATIONS_GUIDE.md](docs/FOUNDATIONS_GUIDE.md) | Frontend foundations: layout, charts, tokens |
| [adr/](docs/adr/) | 34 architecture decision records — the *why* behind the structural choices |
| [SYNC_GUIDE.md](docs/SYNC_GUIDE.md) | Sync guide |
| [ADD_TITLE.md](docs/ADD_TITLE.md) | Adding a new Halo title |
| [BACKUP_RESTORE.md](docs/BACKUP_RESTORE.md) | Backup and restore |
| [RUNBOOK_*.md](docs/) | Operations runbooks: deploy checklist, replay worker, restore test, DuckDB CLI tools |
| [testing.md](docs/testing.md) | Go testing strategy (CGO, coverage ratchet, `gamefiles` tag) |
| [WEAPONS.md](docs/WEAPONS.md) | Weapon reference (keys, families, icons) |
| [CITATIONS.md](docs/CITATIONS.md) · [reference](docs/CITATIONS_REFERENCE.md) | Citations system and full reference |
| [COMMENDATIONS.md](docs/COMMENDATIONS.md) · [reference](docs/COMMENDATIONS_REFERENCE.md) | Commendations system and full reference |
| [CHANGELOG.md](docs/CHANGELOG.md) · [RELEASE_NOTES.md](docs/RELEASE_NOTES.md) | Technical changelog · user-facing release notes |
| [ACKNOWLEDGMENTS.md](docs/ACKNOWLEDGMENTS.md) | Credits and third-party work |

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
| **DuckDB 1.5+** (embedded 1.5.5) | OLAP query engine |
| **In-house Go client** | Halo services: sync, catalogs, film download — no third-party Halo API library |
| **In-house film decoder** (Go) | Theater film → facts → replay artifact |
| **WebCodecs + mediabunny** | Out-of-real-time video export of the replay |

---

## Known limitations

- **Halo services**: the endpoints are undocumented and can change or rate-limit without notice; the client is ours, the ground under it is not.
- **The Theater film is the source of everything measured beyond the stats API** — the weapon behind a kill, positions, engagement range, weapon levels, equipment usage, live objectives and the 2D replay all come from the film, not from a stats field.
- **Films expire on Microsoft's side** — a match old enough to have lost its film keeps its API statistics but gets no replay and no film-derived measurement; the app files it as missing instead of retrying forever.
- **An unknown game build is a refusal, not a guess** — the decryption key of a film is its build; a film from a build the decoder has never seen is set aside with a typed error rather than decoded on an assumption.
- **Film-derived features are Halo Infinite only** — Halo 5 exposes no equivalent, and the app says so rather than showing an empty page.

---

## License

This project is licensed under MIT. See [LICENSE](LICENSE) for details.

---

## Acknowledgements

LevelUp talks to the Halo services through its own Go client — no third-party Halo API library. The work below is prior art we learned from, not code we ship:

- **Andy Curtis** ([acurtis166](https://github.com/acurtis166)) for [SPNKr](https://github.com/acurtis166/SPNKr) — endpoints and Theater film format
- **Den Delimarsky** ([dend](https://github.com/dend)) for [Grunt](https://github.com/dend/grunt) and [OpenSpartan](https://github.com/OpenSpartan) — community documentation and the OpenSpartan import path
- **Gravemind2401** ([Gravemind2401](https://github.com/Gravemind2401)) for [Reclaimer](https://github.com/Gravemind2401/Reclaimer) — Halo map file formats, the reference behind our geometry decoder

See also [ACKNOWLEDGMENTS.md](docs/ACKNOWLEDGMENTS.md).

---

**Made with passion for the Halo community**
