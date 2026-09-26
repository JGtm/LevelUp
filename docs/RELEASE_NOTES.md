## What's new

**v7.5 — 2D replay, the Theater film decoded & a Tactics tab**

The biggest release so far. Halo records a film of every match; until now the app never opened it. It does now — and a match can be watched again from above, second by second, with everything the film knows written around it: who killed whom with what, at what range, who held the flag, who took the power weapon, who wasted the equipment they were carrying when they died.

**Watch your match again, in 2D**
- **A replay page for every match** — all the players move on the real map background, under their own names, from kickoff to the final whistle; play, pause, jump 10 seconds back or forward, and pick a speed
- **A playback bar with tracks** — You, Allies, Dominance and Media on one timeline, with round pills and the score of the round in progress above the field
- **The kill feed follows the cursor** — killer, weapon, victim, medal, and the assist with its damage share whenever the film measures it; a death nobody is credited for says so instead of inventing a killer
- **One card per player** — weapons carried, grenades, shield, active camo and overshield, grappling hook, time spent dead, and a watermark while the player is carrying an objective
- **Layers you switch on and off** — aim cone, trail, shot and kill effects, heat map (time spent or eliminations), weapon spots, weapons left on the ground, deployed equipment, dropped objects, vehicles, named zones, and the live objectives: flag, skull, hill, strongholds A/B/C, bomb, VIP crown
- **The game's own sound** — 177 sounds taken from the game, muted by default, filtered by category
- **Zoom and framing** — wheel, keyboard or a directional pad; the canvas takes the whole block and zooming never crops the map
- **Capture and record** — a PNG in one click, or a video of the replay with its soundtrack, encoded away from real time so the file is clean whatever your machine was doing
- **You get in from everywhere** — the match view, the home tiles, the Explorer and the Tactics tab
- **The Spartan's movement, at the instant** — the player card says when a Spartan is crouched, sliding, clambering or sprinting, read from the film itself; the jump, which the film does not write, is shown as derived and labelled so
- **Vehicles, seat by seat** — who sits where is read from the film, a vehicle points where it is actually heading, a turret shot goes where its gunner aims, and the map shows when a vehicle comes back at its spot
- **Captures in the right colour** — a zone fills with the colour of the side actually pushing, neutral bases included, and a flag on the ground shows its return gauge ("Return N %")
- **Busy matches read more completely** — on the densest matches the replay now follows far more of the film, so movement that used to be lost shows up
- **A clearer field** — mode objectives stand on their floor like the players, with a "Mode objectives" switch and its help; vehicles are larger, with redrawn sprites and weapon flashes of their own; markers keep a readable size at every zoom level; base sounds follow the action, and the score tick keeps time with the game clock
- **Video export in standard formats** — 1080p or 720p, with the framing of your choice when the map is zoomed

**What the film knows, and the API never said**
- **The weapon behind every elimination** — read from the damage source in the film instead of being guessed from the scoreboard, including the deaths with no firearm at all: repulsor, fall, the environment
- **Where people die** — kill and death positions on the map, and "Where it plays out" on the match view
- **Engagement range** — low range (p10), median and high range (p90) per weapon, plus the signed height difference: the gun you win with up close is not the one you win with across the map
- **Weapon levels** — pad pickups split into starting weapons, map weapons, power weapons and power-up pads. What a spot is worth comes from the map, never from the weapon's name: the same Hydra is a map weapon on one map and a power weapon on another
- **Net flag grabs** — juggling folded into a 1.5-second window, so picking your own flag back up twice in a second counts once
- **Assault** — bomb plants, defuses, carriers killed and the rest of the mode's statistics
- **Rounds are counted as rounds** — on modes decided by rounds, the score shown is the number of rounds won and lost; the API's point score is kept alongside because on those modes it can hand the advantage to the side that lost
- **Payback and support** — when a teammate dies, does your side take down the killer within 5 seconds? One rule everywhere — Squad, Sessions, Timeseries and the match view — and support counts the assists between teammates
- **Range and height against your lobby** — your kill range and the height you fight from, read against the whole lobby of each match rather than against the playlist: range roles, height roles, and the engagement height of every kill on the match view
- **Distance records by weapon** — your longest measured kill with each weapon, on the Synthesis
- **Starting weapons read right** — a weapon picked up in the first seconds of a life no longer counts as a starting weapon

**Equipment: used, kept or wasted**
- **The three outcomes of every piece of equipment** — used, kept without ever using it, or dropped when you died, family by family
- **On three pages** — the Synthesis, the Squad and the Sessions, with your share of your team, the parity line and a match-by-match regularity band
- **Charges are visible** — a deployable shows how much of it is left before the first use

**A Tactics tab**
- **The maps you play** — one tile per map, its record, and a floor below which a map is not shown rather than shown wrong
- **An analysis view on the map plan** — where you spend your time, where you die, where you kill, where you die isolated, where the wins and the losses part ways, and the routes you take
- **A cell opens the replay at the exact moment** — click a square, land on the second where it happened
- **Team coordination** — how far apart your squad actually plays, read per match from the variant reference rather than from a threshold somebody picked

**Squad, sessions and progression**
- **The shapes you keep** — six ways of reading a squad across nineteen cards: equipment, special weapons and objectives, solo against squad, your share against the parity of your team
- **Per-match cadence** — every usage number is now a number per measured match, not per minute
- **The exchange in six cards** — assists as stacked bars, player-by-map with a single legend, medals last
- **First kill / first death, ranges and usages** on the Timeseries page, objectives trimmed down to what the mode actually offered
- **New badges** — "Thief" (the most kills stolen from a friend) in the Squad impact matrix, "Scuttled" and "Selfless" on objective modes when the score and the kills tell opposite stories, and "Lackey / Boss" roles on the assist charts
- **Assists exchanged with each relation** — a column in Relations, a "Duo partner" card and an "Inner circle" card
- **A "Usage" tab on the Timeseries and the Squad** — range, equipment and "the shapes you keep" in one place, and the cumulative damage balance on the Timeseries Summary
- **Sessions in four sections**, and a side-by-side comparison in shared rows that stays readable when the drawer opens
- **Squad intensity against your real team and the lobby**, and a filter bar that no longer replays every chart's animation on each click
- **Explorer and Head-to-head** — the briefing shows the favourite weapon of the selection and the kill range of the player you scout; the Head-to-head gets a weapon profile: kill share by weapon class, range by role, top 3 weapons
- **The home match tile** shows how many of your kills a teammate assisted

**Match view**
- **One card template for the whole page** — the three tables that were hard to compare became charts
- **Score over time, per mode** — a round-based mode no longer draws a points curve that contradicts its own result
- **Distance per weapon**, frag distribution in two levels, and special weapon control broken down by weapon level
- **Game changers first** — equipment and weapons that changed nothing fold away
- **A "Weapons and terrain" tab** — kills and weapons, distance, engagement height, equipment usage, pad control and film positions in one place; special weapon control in columns per pad, each team's share as horizontal tracks, and a larger field occupation map you can zoom
- **Payback per player**, one chart per side

**Maps, medals and media**
- **109 map backgrounds**, converted losslessly and served whatever the format, with Forge zones named 100 %
- **Named zones** — the official callouts of the map under the replay
- **The VIP medal "Clash of Kings" has a name and an image again**, and medal images are refreshed from the game's own catalog
- **Likes are per viewer** — your like is yours, not the account's
- **Media in the replay** — your screenshots and clips sit on their own track, at the second they were captured
- **Mode names read right** — playlists whose internal name is written the other way round now show their real mode
- **Clips play one after another** — opening the next clip in the viewer no longer breaks its neighbours

**Repairs**
- **The World ranking page works again** (shipped as v7.3.1) — the source pages it reads had moved, and the page was serving a stale ranking instead of saying so; it can no longer degrade in silence, and an empty ranking and a failed fetch are no longer shown the same way
- **LUSR** — a corrupted Halo 5 arena rating is fixed at the source, the reader corrected, and a purge tool added
- **The match view no longer dies for everybody** when a snapshot lags behind the query
- **A replay that cannot be read says so** — a film that has expired on Microsoft's side is filed as missing instead of being retried forever

**The project**
- **A project footer** — the source, the licence, the feedback issues and a support button, with the non-affiliation notice
- **A bilingual privacy page**
- **The public demo carries no credential at all** — the seed switched from excluding known secrets to publishing only an explicit list of harmless fields
- **Friends belong to a player**, invitations work without a group, and an admin can close the instance and see every identity in one place
- **Copy the Xbox sign-in code in one click**
- **An empty block says why** — every chart card and usage block names the reason when it has nothing to show
- **Backups include the friend lists and Xbox aliases**

**Halo 5** — everything read from a film is Halo Infinite only: Halo 5 keeps its own pages and never borrows another title's data. Where a feature cannot exist there, the app says so instead of showing an empty tab.

**v7.3 — Overtime, first kill / first death & a repaired Achievements page**

**Overtime on your matches**
- **A badge in the match header** — a match that went past its regulation time is now flagged "Overtime", with the extra time played shown on hover
- **A pill in the Explorer** — the same flag appears next to the result in your match list, so overtime matches can be spotted without opening them
- **Your whole history, straight away** — the flag is worked out as the page is read, so it covers every match you have ever played; nothing to re-sync and nothing to wait for
- **Measured, not guessed** — the game gives no overtime signal, so the flag compares the time actually played against the regulation time of the variant played, with a safety margin
- **Halo 5 deliberately left out** — no regulation times are declared for it yet, and an undeclared variant is never flagged rather than wrongly flagged

**First kill / first death — a new chart**
- **One lane per player** — the first kills of each match are plotted above the line and the first deaths below, so a fast starter and a fast dier are told apart at a glance
- **Medians and advance window** — each lane carries its median first kill, its median first death and the bar between the two: the head start you usually have
- **Sorted by how early you open** — lanes are ordered by median first kill
- **On three pages** — Squad ("Dynamics" tab, after Intensity), Timeseries ("Progression" tab) and session detail, where it also appears in the session comparison panel
- **Replaces the old histograms** — the 10-second and 15-second bucket charts were unreadable and disagreed with each other; both are gone
- **A scale that holds** — one endless match no longer flattens everybody else's lane

**Halo 5 — Achievements page repaired**
- **The page opens again** — it answered with an error instead of showing your milestones
- **Section title always present** — the Achievements grid keeps its title whether it is loading, empty or in error

**Reading in your language**
- **Commendations follow the language switch** — their labels came from the server in the language of the moment and stayed there until you reloaded the page
- **"Headshots" is translated** — the metric showed up under its raw field name in several places
- **A faster start** — the field reference is no longer fetched twice when the app boots

**Match view**
- **A missing objective statistic no longer breaks the scoreboard** — a match whose objective data could not be read was shown as incomplete in full; now only the Objectives section stays empty and everything else is served

**Demo**
- **Improvement campaigns visible** — the demo profile showed no campaign at all
- **Objective statistics included** — the demo now carries the objective data the real app has
- **Anonymization reinforced** — player identifiers are anonymized along with the rest

**Also shipped ahead**
- **Match view rearranged** — the Media block moves last, alone and full width, and the frag, chart and medal rows no longer strand a lonely card on a third of the width
- **Objectives participation reworked** — the Objectives axis of your profile is now an index of what you actually did against what the modes you played had to offer, instead of a raw score diluted by every match you ever played

**v7.2.1 — Objective modes, title isolation & instant player search**

**Three more objective modes — Stockpile, Extraction and VIP**
- **Stockpile** — power seeds deposited and stolen, enemy carriers killed, and your time carrying a seed
- **Extraction** — successful extractions, initiations completed, enemy beacons converted and conversions denied
- **VIP** — VIPs killed, times you were selected as VIP, kills while being the VIP, total VIP time and your longest run as VIP
- **Where you read them** — the "Objectives" section of the match view, exactly like Capture the Flag, Strongholds, King of the Hill and Oddball: one column per stat of the mode played, with a "Team total" line

**Eight new objective citations**
- **Flag play** — Flag Captures, Flag Secures, Unstoppable Carrier and Aggressive Return
- **Zones and Oddball** — Zone Defense, Untouchable Carrier, Skull Carrier Takedown and Skull Grabs
- **Tiers set on real play** — every ladder is calibrated on actual match data instead of a generic scale, so the rare feats (killing while carrying the flag or the skull) stay within reach

**Explorer — every season, not just some**
- **"Matches by season" is complete** — the breakdown only covered the seasons the game happened to return; it now covers every season of the catalog
- **Nothing silently missing** — a season you never played and a season that could not be retrieved are no longer shown the same way

**Squad objectives & Prestige**
- **"Propose challenges" works again** — the button could fail for good until the server was restarted
- **No more challenges whose rule does not match** — challenges whose stated rule did not correspond to the way they were actually scored are no longer offered
- **Prestige without points** — opening your prestige before earning a single point returned an error instead of an empty summary

**Charts that read straight**
- **Real average lifespan** — the average-life chart uses the value measured in game instead of an estimate derived from time played and deaths, and the correlation cloud now tells the same story as the histogram
- **Win rate and MMR on their own axes** — the win-rate curve is no longer flattened against the bottom by the MMR scale it used to share
- **Dominance on the results tape** — a marker flags the matches your team dominated
- **Synergy radar** — the tooltip shows the raw value next to the normalized score

**Objective modes — Capture the Flag, Strongholds, King of the Hill, Oddball**
- **Objective stats collected for every match** — flag captures, returns, steals and time as flag carrier; zone captures, secures and time in zones; skull grabs, time and longest time as skull carrier
- **In the match view** — a new "Objectives" section per team, one column per stat of the mode played, with a "Team total" line
- **In your totals** — aggregated objective figures on Synthesis, Squad and Timeseries, shown only where the title provides them
- **Objective citations** — "Storm the Walls", "Got You!", "Head in the Game" and "Carrier Takedown" now progress on the real objective counters instead of medal awards

**No more Halo 5 data leaking into Halo Infinite**
- **Strict title gate** — the Spartan banner (nameplate, emblem, backdrop, service tag) no longer shows another title's assets while a page is loading or while you switch games
- **Title switch without crossed data** — every server response now states which title it was resolved for, and the app refuses anything that does not match the title you are on
- **Appearance per player and per title** — Halo 5 nameplates and emblems are no longer shared between players, and emblem and banner colours are kept separately

**Explorer — instant search and honest live data**
- **Instant player search** — gamertag suggestions come from your local data in about 200 ms; an explicit "Search on Xbox" button looks the player up on Xbox when you need it
- **Any player readable again** — career, identity, medals and seasons of a searched player are fetched through the pool of healthy credentials, so a target whose own credentials are dead is no longer a dead end
- **No more silent degradation** — when live data cannot be fetched, a discreet badge says so ("Live data unavailable (authentication)", "(error)", "Partial live data") instead of leaving an empty card
- **Smoother navigation** — selecting a player no longer stacks one history entry per click, the "Cumulative frag gap" chart gets its X axis back, and an explicit message replaces it when you have no match in common

**Halo 5 — medal categories**
- **215 medals in 11 categories** — Halo 5 medals are now grouped the way Halo Infinite's are (Strongholds, Warzone, Objective, Capture the Flag, Oddball, sprees, multikills, weapons, vehicles, infection, style) under the four usual super-sections
- **Ghost medals hidden** — three medals earned in game but absent from the official catalog no longer pollute the Medals page (their data is kept)

**Readability**
- **Column tooltips everywhere** — hover any table header to get the definition of the column
- **Legends at the foot of the block** — chart legends move out of the drawing area, which also restores the bar thickness of "Tools of destruction"
- **Grenades split by type** — the frag sunburst breaks grenades down into frag / plasma / dynamo / splinter, and a Halo 5 double count (melee kills while holding a weapon) is gone
- **One weapon name per title** — weapon names come from a single per-title reference: "Fuel Rod SPNKr", "Fragmentation grenade" or "Light Rifle" now read the same everywhere
- **"In placement" instead of a blank** — a performance score still being calibrated shows "In placement (8/10)" on the tables and the home tiles rather than a bare dash
- **Clearer labels** — the "OC / DR" axes become "Yield / Resistance", playlists and modes in the squad picker are localized, the French wording of the cumulative net-score chart and of the Meganaut medal description is repaired, and the Career blocks are height-aligned

**Notifications & alerts**
- **New-medal notification** — you are notified the first time you ever earn a medal
- **Discord alerts that behave** — disk alerts no longer burst after every restart, a return to normal is announced once, the release notification carries the real version, and the automatic sync cycle reports its new matches
- **Readable notifications** — career rank names are localized and figures are rounded

**Admin & sync**
- **Initial sync for a single player** — a new admin card re-imports one player's full history, with a legend explaining the scope of each of the four sync actions

**Quality of life**
- **"Dynamics" in the Squad menu** — the tab existed but was missing from the navigation
- **"See synergies"** — a shortcut from a squad session to the Synergies view
- **Career XP on Sessions** — the cumulative XP curve and XP per match, already on Timeseries, now also cover a single session
- **Match not synced yet** — opening a match absent from your data shows a dedicated screen instead of a generic error

**v7.1 — Squad reliability, Halo 5 combat data & career XP**

**Squad — more reliable**
- **Exact-composition history** — "Squad performance per session", "Map performance — session vs history" and "Win rate — session vs history" now compare each session against your history with that *exact* lineup, instead of a blurred all-teammates average
- **Fixed saved compositions** — a saved squad now shows the same members for everyone; no more duplicates or a teammate missing from another player's view
- **Chronological squad charts** — the map and win-rate comparison charts now read left-to-right in the order the maps first appeared, matching the intensity view
- **Clearer win-rate axis** — the "(n)" match count on the axis is now explained by a hover tooltip
- **New "Dynamics" tab** — intensity, yield/resistance and engagement charts are grouped in a dedicated Squad tab
- **Intensity as a median profile** — the match × phase heatmap is replaced by a median frag-share profile per phase, with an interquartile band that makes irregular play visible
- **Yield & Resistance** — split into two multi-player charts, one colour per player
- **Damage balance in lives** — cumulative damage dealt minus taken, expressed in the title's lives, on Sessions and Dynamics
- **Cumulative engagement gap** — a new cumulative engagement-gap curve on Timeseries, Dynamics and Sessions

**Squad objectives**
- **Squad challenge loop** — localized challenge labels, join feedback, live per-member progress, and full lifecycle (abandon, delete, expiration)
- **Renamed** — "Squad focus" becomes "Squad objectives"
- **No more silent failure** — the "Propose challenges" button now shows an explicit error instead of doing nothing

**Halo 5 — repaired combat data**
- **Vehicles destroyed & hijacks** — vehicle-destroyed and hijack counters on the Synthesis "Tools of destruction" card
- **Combat mechanics restored** — assassinations, ground pound and shoulder bash counts repaired for matches whose values had been stored as zeros
- **Cleaner scoreboard** — Halo 5 team names de-duplicated, and MVP/LVP is no longer decided by mechanic kills
- **Mechanic columns hidden off Halo 5** — the assassination / ground-pound / shoulder-bash columns no longer show zeros on Halo Infinite

**Citations repaired**
- **Firefight** — "Firefight eliminations" now counts your Firefight victories
- **Remaps & fixes** — grenade citations restored, "road trip" remapped to the Splatter medal, and "flag defender" cleanly disabled (no data source for it yet)

**Career**
- **Estimated career XP** — a cumulative XP curve and XP-per-match on Timeseries, calibrated on real data
- **Multi-title Path to Hero** — progression toward the title's own maximum rank (fixes Halo 5 aiming at the wrong ceiling)
- **Medals page** — a new Career sub-page listing the title's full medal catalog, including medals you have never earned, grouped by section, with all / earned / not-earned filters and sorting

**Home & Synthesis**
- **Peak date on rank cards** — your best LUSR / CSR cards now show the date the peak was reached
- **Longest loss streak** — a new KPI card next to your win streak

**Explorer**
- **Head-to-head panel** — the "Over XX matches together" section adds win-rate donuts (together / head-to-head) and a cumulative frag-gap-vs-target chart
- **Collapsible briefing** — the Matches-mode synthesis can be folded away, and the choice is remembered

**Expected FDA**
- **Gap-to-expected curves** — new "gap to expected FDA" charts on Timeseries and Sessions, plus a per-member cumulative gap on Squad Synergies, with an average-gap-per-match KPI
- **Expected-FDA overlay** — a thin per-match expected-FDA line added on the FDA-gap chart, sharing the same axis

**Media**
- **Per-player audio-track roles** — declare the voice / game / other role of each audio track of your clips, per player, from a gear modal in the gallery (manual, or automatic)

**Quality of life**
- **Language in the URL** — the active language is now part of the address, so shared links keep their language
- **Halo Waypoint column** — an optional column to open any match on Halo Waypoint
- **Every table sortable** — click any column header to sort, across all tables in the app
- **Stable browser-tab titles** — page- and language-aware tab titles instead of a bare "LevelUp"
- **Scoreboard identity** — per-team identity colours and team logos, plus a consistent player-column width
- **French wording** — a sweep of anglicisms replaced with proper French terms
- **Chart legends & percentages** — the "Tools of destruction" legend is centered below the chart, with percentage labels on segments and legends and aligned chart heights across Synthesis, Sessions and Squad

**Admin**
- **Spartan appearance diagnostic** — a per-player panel in the admin Data tab explains why a nameplate, emblem, backdrop or service tag failed to load, with a re-authentication shortcut

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
- **Multi-player demo mode** — a public read-only demo with an anonymized multi-player corpus (a Spartan and two teammates), real HLS clips and prestige fixtures, with client-side language switching

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
- **Combat Yield band** — Offensive Conversion / Defensive Resistance shown as a unified band across Synthesis, Sessions and Squad
- **Real gameplay time** — durations now subtract the pre-match countdown, so time-played and per-minute stats reflect actual combat time

**Media V2 — likes, Discord notifications, upload**
- **Persistent likes** — like your screenshots and clips directly from the grid, state kept across reloads
- **Smart grouping** — by favorites, session, or solo/squad context
- **Lighter grid** — native thumbnails, shared lightbox, heart icons for liked / unliked
- **Drag-and-drop upload** — add your manual captures straight from the Media page
- **Non-destructive scanning** — automatic background re-indexing with a dedicated `--captures-dir` option
- **Discord notifications for new media** — embed with a GIF or screenshot thumbnail whenever a new capture is indexed; anti-spam (each file notified once); toggle `discord_notify_new_media` in Settings
- **Manual reassociation with match suggestions** — built-in modal listing your matches in a ±15 / ±60 / ±180 min window around the capture, with map thumbnail, map · mode · playlist, local time + delta, outcome badge and full lobby per team; one click + confirm to fix any media linked to the wrong match
- **HLS video player** — in-browser clip playback with an audio-track selector (game / voice / full mix) and multi-track transcoding at ingestion (HEVC clips remuxed automatically)

**Dedicated Match page & richer visualizations**
- **Clean URL per match** — `/players/{gamertag}/matches/{id}`, shareable, with prev/next match navigation
- **Tug-of-War timeline** — dynamic curve of team score swings
- **KD Timeline** — kills/deaths evolution by phase with a moving average
- **Impact Badges** — narrative badges (Top Killer, Silent Hero, False Brother, Comeback Champion…) computed per match
- **Encounters panel** — list of players you've crossed paths with in earlier matches
- **Combat Yield & Perfect Kills** — new metrics in the match view
- **V7 scoreboard** — higher info density: expected stats, skill rank, linked media, citations
- **Session comparison** — dedicated A/B page: pick any two sessions and compare KDA, performance score, Offensive Conversion / Defensive Resistance, outcome distribution and dominant playlist side by side
- **Exclude a match** — drop a match from your stats with a full cascade recompute (sessions, performance score, citations) and a guard for ranked matches
- **Rank badge always visible** — LUSR/CSR tier shown on the scoreboard, including shared CSRs for untracked players

**Rebuilt authentication**
- **SISU/PoP provider** — new Xbox authentication with Proof-of-Possession for more stable sessions and fewer reconnects
- **OAuth redirect flow** — browser-based Xbox login (`/auth/xbox/login` → Microsoft → callback) as an alternative to Device Code; configurable via `LEVELUP_OAUTH_REDIRECT_URI`
- **Local auth** — username/password mode for single-user / LAN deployments
- **Invitation-based registration** — new `/register` page; account creation requires a server-issued invitation code (`?code=` query param); invalid or expired codes are rejected before the account is written
- **SISU by default & instance lockdown** — the SISU provider is now the default Xbox auth; the instance can be sealed and per-player ownership checks isolate each player's data (no cross-account access)
- **Opt-in password for fast re-login** — set a password to re-open your SSO session quickly without going through the full Xbox flow every time
- **Reconnection banner & dead-token detection** — an in-app banner detects a dead Xbox refresh token and walks you through re-linking before sync breaks; the token store is the single source of truth for credentials

**Xbox achievements & match events**
- **Xbox achievements sync** — your Xbox achievements are pulled automatically from the Halo API on every sync
- **Achievement tracker** — browse your full achievement list on the Career page: filter by unlocked / in-progress / not-started, track your Gamerscore (earned vs total), and filter by game for multi-title support
- **Highlight events** — binary parser of match films to extract all major events (medals, clutches, spawns)
- **Weapon kills backfill** — weapon used per frag reconstructed from the film (POV ~87 %)
- **Comeback badges for teammates** — Remontada / Collapse / Counter-Remontada computed for squad members synced alongside you
- **Achievement category filter** — filter your Xbox achievements by Multiplayer / Campaign / Other, with Multiplayer shown by default

**Communauté — Hall of Fame, Relations & Face-à-face**
- **Multilingual Season Pass** — Battle Pass translations in 26 languages, tier images from GameCMS
- **Relations** — track all players you have crossed paths with: per-player stats, shared match history, alliance and rivalry patterns, career micro-leaderboard
- **Face-à-face** — dedicated 1v1 (or 1v1v1 mirror) comparison page: pit any two or three players head-to-head across Combat, Precision and Bilan metrics; encounter badges mark alliances and rivals from your shared match history

**Objectives & Prestige**
- **Objectives** — individual and squad challenge system: set personal goals or create squad challenges (collective or competitive) on any Halo metric with configurable windows, tiers, and narrative arcs; earn Prestige Points (PP) on completion; two evaluation modes (threshold / cumulative) and two creation modes (free / guided)
- **Prestige leaderboard** — PP ranking in Palmares comparing your score against your squad and relations; four tiers: Normal / Heroic / Legendary / Mythic
- **Narrative arcs** — group challenges into arcs with free creation, ready-made presets and deletion; an arc-completion bonus is credited and shown on the final step
- **Coach-driven challenges** — guided mode proposes challenges auto-calibrated on your weaker performance axes

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
- **Campaign tracker** — start an improvement campaign on a chosen goal and follow your progress toward it over time, with a dedicated tracker and start modal

**In-app notifications**
- **Notification center** — per-player feed with unread badge in the nav bar, category filters, day-grouped timeline, bulk actions, and 60-second live refresh; preferences configurable per player in Settings

**Automated sync & real-time presence**
- **100 % automatic sync** — no more manual `python scripts/sync.py`: the app syncs your matches on its own, continuously in the background, as soon as a new game is played
- **Immediate end-of-match trigger** — as soon as a player finishes a match, the watcher fetches the stats without waiting for the next tick
- **Xbox RTA presence + Steam polling** — real-time online detection to know who's playing and sync at the right moment
- **Smart scheduler** — sync cadence adapts to player activity; no useless requests when nobody's playing
- **Autonomous token refresh** — no more interruptions: Halo tokens renew themselves in the background
- **Proactive reconnection** — status=3 handling with on-demand XSTS refresh, automatic reconnect on startup
- **Convergent sync** — asset names (maps, modes, playlists) resolve themselves during sync, with a weekly catalog-refresh safety net for stragglers
- **Cross-source dedup** — concurrent syncs of the same match are de-duplicated so nothing is fetched or written twice

**Settings & admin**
- **Settings auto-save** — settings persist immediately with an ephemeral visual indicator
- **Admin page** — supervision UI (auth provider, job status, privacy)
- **Browser preferences** — selected player, language and filters remembered across sessions
- **Configurable API endpoints** — Halo Stats, SPNKr, CMS… all configurable from Settings
- **Automatic backup & restore** — restic-based snapshots of every database, with a dedicated Settings tab (manual trigger, per-database status, informational integrity check) and point-in-time restore
- **Monitoring dashboard** — full admin supervision: sync cycles and trend sparklines, convergence, data-integrity invariants, token health (MSAL/XSTS/Refresh), per-player Halo API-call attribution, recurring-error collector, logs and performance

**Assets & maps**
- **Cache-aside map images** — map artwork downloaded and cached locally, no more repeated external requests
- **`populate-assets` CLI** — Go command to pre-download every asset (maps, medals, Battle Pass tiers) ahead of offline use

**Colour accessibility**
- **Colour-blind safe palette** — a new Okabe-Ito palette (designed in 2008, universally recommended) is available in Settings → Accessibility; it replaces every colour in the app — charts, performance indicators, match outcomes, K/D ratings — with tones that remain distinguishable under deuteranopia, protanopia and tritanopia
- **Live preview** — the palette switches instantly across the whole app without a page reload; a swatch preview lets you compare before committing
- **Persistent preference** — your choice is saved in the browser and restored automatically on every visit

**Sessions — rebuilt page**
- **Richer charts** — F/D/A per match and per minute, performance score by tier, F/D/A radar, OC/DR cloud and per-match engagement, with explicit axes and skill-tier bands
- **A/B compare drawer** — pick two sessions and compare them side by side, with shared scales across all charts
- **Single-session metrics** — Win %, KDR, kills/match, average precision and rank delta surfaced directly in the single view
- **Rank delta per match** — LUSR/CSR movement shown match by match, with an adaptive session window

**Explorer — combat profiles & rivalries**
- **Live combat profile of any player** — read a non-tracked player's recent combat profile live (read-only, short-lived cache), with career rank and Spartan grade
- **Dominance & encounters** — dominance metrics plus shared-history encounters (ally / rival / opponent) surfaced per player
- **CSV export** — export the filtered match table in one click
- **Cascade-aware filters** — five filter dimensions whose available options update as you narrow the others down
- **Matches by season** — per-season match bars with a CSR rank badge
- **Search briefing** — a compact recap sits above the results: matches, win rate, FDA and performance (shown as lowest · average · highest), total time and peak stats; per-map / mode / selection / context breakdowns; ranking progression per playlist; best streaks and standout moments — each block with a legend tooltip
- **Sortable table & extremes highlighting** — click any numeric column header to sort the whole result set; the top and bottom 10% of each key column are highlighted so your best and worst matches stand out at a glance

**Ranked (CSR)**
- **Per-match & per-playlist CSR** — CSR captured for every ranked match and for each active ranked playlist
- **CSR season selector** — switch between available CSR seasons; past seasons can be backfilled
- **Dynamic placement thresholds** — the placement-match count is resolved per season
- **Automatic teammate CSR** — every registered co-player's CSR is pulled and distributed on a ranked sync
- **Authoritative ranked-playlist reference** — ranked status is read from a stable reference instead of being guessed from matches

**World leaderboard**
- **Global CSR ranking** — a world CSR leaderboard scraped from Halo Waypoint, enriched with native per-player stats (KDA, accuracy, damage) across multiple seasons
- **Cross-season trend** — a colored indicator shows each player's movement versus the previous season
- **Local players first** — your tracked players are always surfaced at the top

**Squad coach**
- **Squad orientation** — the coach surfaces the squad's current focus and biases the challenge pool toward your weaker performance axes
- **"Focus of the moment" card** — a CoachFocusCard highlights the single most useful thing to work on next, with a soft-negative signal that stays encouraging

**Rating accuracy (LUSR v2)**
- **Reworked rating engine** — a new TrueSkill2 model (factor graph + expectation propagation) with time-played weighting, quit handling, pre-match win probability and anti-volatility display safeguards, so your rating moves for the right reasons

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
