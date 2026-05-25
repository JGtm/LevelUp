## What's new

**v7.0 — New React app, Mission Control & multi-title support**

Major overhaul. LevelUp leaves Streamlit behind for a **React 19 + Go API** app with a **fully redesigned UX/UI** and **fully automated synchronization**. Close to 400 commits of work — here's what it changes for you:

**A completely rebuilt app — new UX/UI**
- **React 19 + Tailwind frontend** — instant navigation between pages, shareable URLs for every match/session/player, dark/light theme, full FR/EN i18n
- **Unified design system** — palette, typography, spacing and components (cards, buttons, modals, tooltips, carousels, lightbox) harmonized across the whole app; no more patchwork Streamlit pages
- **Two-level navigation** — L1 bar (Home / Synthesis / Explorer / Squad / Communauté / Ascension / Media / Help / Settings) plus contextual L2 tabs; everything is one click away, no more scrolling sidebar
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

**Rebuilt authentication**
- **SISU/PoP provider** — new Xbox authentication with Proof-of-Possession for more stable sessions and fewer reconnects
- **OAuth redirect flow** — browser-based Xbox login (`/auth/xbox/login` → Microsoft → callback) as an alternative to Device Code; configurable via `LEVELUP_OAUTH_REDIRECT_URI`
- **Local auth** — username/password mode for single-user / LAN deployments
- **Invitation-based registration** — new `/register` page; account creation requires a server-issued invitation code (`?code=` query param); invalid or expired codes are rejected before the account is written

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
