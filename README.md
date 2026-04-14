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

**v7.0 — Mission Control challenges, live + persisted**
- **Active challenges back on Home V7** — the Mission Control home now restores the active challenge card from HaloStats `/decks`, with the real deck expiry, badge, localized title/description and the player's actual `x/y` progress
- **Challenge data persisted cleanly** — current player challenge states are now historized in each player DB via `challenge_snapshots`, while shared challenge definitions live in `metadata.duckdb`
- **Multi-language challenge catalog** — challenge titles and descriptions are stored from all CMS-exposed languages, normalized to BCP-47, with `en-US` fallback when the requested locale is missing
- **Live-first, failsafe behavior** — if `metadata.duckdb` is locked by another process, the Home page still renders challenge data live and simply skips persistence for that refresh
- **Discord — new media notifications** — a Discord embed with GIF or screenshot thumbnail is sent whenever new captures are indexed; each file is notified once (anti-spam via `discord_notified_at`); toggle `discord_notify_new_media` in Settings
- **Media V2 — persistent likes, better grouping, lighter grid** — screenshots and videos can now be liked directly from the grid, grouped by liked state, session, or solo/squad context, and keep their like state across reloads through resilient UI preference storage. The grid also uses native thumbnails, a shared lightbox, and the local heart assets for liked / unliked states

**v6.5 — Heatmap escouade & paramètres fiabilisés**
- **Heatmap d'intensité par joueur** (Teammates) — nouvelle visualisation : heatmap match × phase (début/milieu/fin) pour chaque membre de l'escouade. Voir qui frappe tôt, qui accélère en fin de match. Toggle pour afficher tous ensemble ou joueur par joueur
- **Notifications Discord séparées** — les alertes sync et backfill ont maintenant chacune leur propre toggle ; désactiver l'une n'affecte pas l'autre
- **Paramètres plus robustes** — les réglages sont écrits de façon sécurisée (écriture atomique + sauvegarde automatique) ; le fichier ne peut plus se corrompre en cas de crash ou d'arrêt forcé
- **Corrections** — les records (meilleure performance) ne s'affichent plus par défaut quand ils avaient été désactivés

**v6.4 — Filtres médias, CSR escouade & aides à la lecture**
- **Filtres de la médiathèque** — filtrez vos clips et screenshots par propriétaire (mes captures / coéquipiers / sans match associé), carte, mode, résultat (victoire / défaite…) et contexte solo vs escouade. Triez par date, carte, mode ou résultat en un clic
- **Aides à la lecture** — une case à cocher dans la barre latérale affiche ou masque les ~45 cartouches d'explication sur chaque page ; désactivable pour une interface plus épurée
- **Récapitulatif carrière redessiné** — 8 cartes compactes côte à côte : Matchs, Durée totale, Frags, Morts, Assists, Précision, Temps en vie, Résultats. Chaque carte compare votre valeur à votre moyenne historique (code couleur vert/or/rouge à ±8 %)
- **Win/Loss intégré dans Timeseries** — la page Win/Loss devient un onglet dans Timeseries ; onglets renommés : Résumé · Cartes & Modes · Progression · Avancé
- **CSR des coéquipiers automatique** — lors d'un sync sur un match classé, le rang de tous les co-joueurs enregistrés est récupéré et distribué automatiquement — plus besoin que chacun synchronise son propre compte
- **Badges Remontada/Collapse pour les coéquipiers** — calculés pour vos co-joueurs synchronisés en même temps que vous
- **Panneau de légende fixe** (Teammates) — un panneau flottant affiche la couleur de chaque membre de l'escouade pendant tout votre défilement dans la section squad
- **Préférences mémorisées entre sessions** — le joueur sélectionné et la langue sont mémorisés dans le navigateur ; les filtres survivent aux mises à jour

**v6.3 — Noms localisés, records escouade & détails médailles**
- **Cartes et modes dans votre langue** — noms de cartes, playlists et modes de jeu en français (ou anglais) sur toutes les pages : filtres, tableaux, graphiques et histogramme de winrate
- **Description des médailles au survol** — survolez une médaille dans le scoreboard ou la section Citations pour lire sa description
- **Records all-time de l'escouade** — la page Teammates affiche les meilleurs records en carrière pour chaque membre (K/D, kills, séries…) avec annotations colorées par joueur et vue détaillée par carte
- **Badge Top Killer** 🔫 — affiché sur la timeline Impact pour le premier joueur à atteindre 10 kills dans le match
- **Histogramme temps de premier kill/mort revu** — graphique en papillon miroir avec intervalles de 15 secondes et temps réel de jeu (décompte pré-partie soustrait)
- **Dernier match amélioré** — carte et mode fusionnés ; les cartes MMR, Kills et Deaths affichent aussi le score équipe adverse avec un écart coloré ; le score de performance apparaît directement à côté du rating
- **Médailles et citations en grille 4 colonnes** — scoreboard plus lisible ; armes en grille 2 colonnes avec bonne proportion de miniatures
- **Recherche partielle par ID de match** (Explorer) — tapez 3+ caractères d'un ID de match pour filtrer instantanément la liste
- **Histogramme de cadence de kills** — nouveau graphique (onglet Combat) : kills par intervalles de 15 secondes pour vous et les ennemis, avec moyenne mobile par équipe
- **Heatmap d'intensité de match** — visualise la densité de kills par phase sur l'ensemble de vos matchs
- **Auto-indexation de la médiathèque** — vos clips sont re-scannés automatiquement en arrière-plan après chaque sync
- **Corrections** — citation Spartan Carnage corrigée ; noms corrects dans les notifications Discord ; calendar de filtres avec navigation libre entre les années

**v6.2 — Badges Comeback & vue escouade unifiée**
- **Badges Remontada / Collapse / Contre-Remontada** — l'app détecte les scénarios de comeback dans votre historique : *Remontada* (vous étiez perdants à mi-match et vous avez gagné), *Collapse* (vous meniez et avez perdu), *Contre-Remontada* (vous avez stoppé le comeback adverse)
- **Vue escouade unifiée** — les vues 1-vs-1 et escouade fusionnent ; vous obtenez les mêmes graphiques riches pour 1, 2 ou 3 amis
- **Graphique Kills ↑ / Morts ↓** — kills et morts fusionnés en un seul graphique miroir par membre, pour comparer les arcs K/D d'un coup d'œil
- **Noms de modes cohérents** — les libellés de modes de jeu sont désormais homogènes sur toutes les pages et tous les graphiques

**v6.1 — Sync plus rapide, bugs corrigés**
- **Sync ~30–40 % plus rapide** — chaque synchronisation se termine nettement plus vite
- **Noms de rangs corrects** — le rang affiché correspond désormais au vrai palier (ex. « Lance Corporal Diamond 1 »)
- Corrections : scores de performance et vues matérialisées toujours à jour après sync

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

**Prerequisites**: Python 3.12+ recommended (3.10 minimum).

### Windows (no technical knowledge required)

```
1. Download and extract the ZIP (or git clone)
2. Double-click LevelUp.bat
   → Python is installed automatically if missing
   → .venv created, dependencies installed
   → API starts on http://localhost:8000
   → browser opens at http://localhost:5173
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
pip install -e ".[spnkr,api]"
cd apps/web && npm install && cd ../..
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
