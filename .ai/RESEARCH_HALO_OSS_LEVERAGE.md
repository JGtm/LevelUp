# RESEARCH_HALO_OSS_LEVERAGE.md — 8 projets Halo OSS + API CSR Leaderboard → ce qu'on reprend

> Workflow recherche fan-out (8 agents WebFetch + synthèse), 2026-06-24. Mappé sur nos gaps (a..g).

## TL;DR — le standout
**CSR Halo 5 par playlist (a) + leaderboards mondiaux (e)** = notre gap le plus chaud, **données déjà dispo + infra à réutiliser**. C'est LA cible.

## Leverage map par gap

### (a) CSR par playlist — **HIGH**, ~1-2 j
- **Source** : `ServiceRecordArena.ArenaPlaylistStats[]` (endpoint INTERNE `spartanstats.svc/h5/players/{gt}/servicerecords/arena`, SpartanToken) — déjà récupérable, **juste pas parsé** (cf. memory `reference_h5_csr_model`). Champs : PlaylistId, Csr, HighestCsr, MeasurementMatchesLeft, Games W/L.
- **Référence schéma** : Leafapp `h5_rankings` (playlist_id, season_id, csr, tier, rank, lastCsr…) — pattern upsert prouvé en prod.
- **Action** : table `h5_arena_playlist_stats` (PK account_id+playlist_id+season_id) + handler qui parse ArenaPlaylistStats + upsert + delta CSR.

### (e) Leaderboards mondiaux par playlist — **HIGH**, ~1 j
- **Source** : API OFFICIELLE `GET www.haloapi.com/stats/h5/leaderboards/csr/{seasonId}/{playlistId}?count=250` (clé Ocp-Apim) → Rank, Tier, SubTier, Rating (CSR), Gamertag, **XUID**. (Shape inféré du doc haloapi.com — à confirmer au 1er appel.)
- **Réutiliser l'existant** : on a déjà `cmd/snapshot-world-leaderboard` + table snapshots (pattern Infinite) → calquer pour H5.
- **seasonId/playlistId** : via `/metadata/h5/metadata/seasons` (on a déjà `cmd/h5-metadata-fetch` pour la metadata officielle).
- **Action** : job `fetch-h5-leaderboards` (boucle playlists ranked × season) + table snapshots + handler `GET /api/v1/halo5/leaderboard/{seasonId}/{playlistId}`.

### (c) Medals h5 — **MEDIUM**, <1 j
- On a DÉJÀ la metadata (`h5-metadata-fetch` seede medals : name/classification/difficulty/**sprite sheet + offsets**). Reste l'**affichage** : rendu icône via spriteSheetUri + background-position (CSS), + agrégation par classification.
- Réf affichage : destefanis/halo-medals (sprite coords).

### (g) Weapon taxonomy (class/role/family/faction) — **validé**
- **Le research CONFIRME notre choix** : class/role/family/faction ne sont **dans aucune API** (officielle ni wrapper) → curation manuelle obligatoire. **C'est exactement le registre qu'on vient de construire.** Rien à reprendre, on était sur le bon chemin.

### (b) Carnage sans xuid — **NON résolu** (correction d'une erreur agent)
- L'agent a prétendu que l'API officielle carnage porte le xuid. **FAUX** : le doc officiel haloapi.com (collé par le user) dit `"Xuid": null` (« Internal use only. This will always be null »). Internal ET officiel = null. **PeopleHub reste le seul chemin** (cf. memory `reference_h5_carnage_no_xuid_peoplehub_only`).
- Seul apport : le leaderboard (e) renvoie gamertag↔xuid pour les joueurs classés (source xuid partielle).

### (d) REQ packs / cosmétiques — **LOW** (nice-to-have), ~1-2 j
- Halo5Reqs documente tout : `GET halo5api.svc.halowaypoint.com/en-us/reqs` (catalogue : rarity 0-4 + mythic, 5 catégories), `/h5/players/{gt}/packs`, `/h5/players/{gt}/cards`. Inventaire joueur = endpoints internes (SpartanToken).
- Catalogue cosmétiques aussi via metadata officielle (`/metadata/h5/metadata/requisitions`).

### (f) Film / kill-feed décodage — **ABANDON confirmé**
- Aucune source n'expose un décodage film / reconstruction kill-feed offline. Reste abandonné (notre décision tenait).

## Top 3 quick wins
1. **CSR par playlist** (a) — parse ArenaPlaylistStats (déjà dispo) → dashboard CSR H5. 1-2 j.
2. **Leaderboards mondiaux** (e) — endpoint officiel + réutiliser snapshot-world-leaderboard. 1 j.
3. **Affichage medals** (c) — metadata déjà seedée, juste le rendu sprite. <1 j.

## À ignorer
Spartan-Hub (archivé, scraping), 16807-Pious-Academic (2016, ref endpoints seulement), azz/haloapi.js (archivé ~2017, ref types seulement). tbenz9/go-halo5-api + Leafapp = bonnes **références de shapes/schemas**, pas des deps.

## Pièges
- **2 auths à ne pas mélanger** : officiel = `Ocp-Apim-Subscription-Key` (www.haloapi.com) ; interne = `X-343-Authorization-Spartan` (spartanstats.svc). On fait déjà les deux.
- **Tier vs DesignationId** : le leaderboard renvoie Tier+SubTier+Rating, pas DesignationId → mapper via `/metadata/h5/metadata/csr-designations` (déjà seedé par h5-metadata-fetch).
- **Saisons/playlists retirées** : `isActive`/`isRanked` togglent → filtrer via metadata seasons avant de poller.
- **(a) ≠ (e)** : ArenaPlaylistStats = CSR du joueur par playlist ; leaderboard = classement mondial. Deux sources distinctes, les deux utiles.

---

## Passe 2 — LEÇONS des repos « morts » (le user avait raison : archivé ≠ rien à apprendre)

### Modèle de données à copier (Leafapp, adapté à nous)
- **CSR snapshot par (joueur, playlist, saison)** : `rank` + `rank_previous` + `csr` + `csr_previous` + `tier` + `designation_id` + `percent_to_next_tier`. Le `*_previous` permet le **delta UI** (flèches ↑/↓, « passé de X à Y »). C'est LE schéma pour (a)+(e).
- **Stratégie de refresh leaderboard** : **delete-and-reinsert** par (saison, playlist) — pas d'update. Avant de delete, on lit l'ancien snapshot en dict → on remplit `rank_previous`/`csr_previous`. (= notre pattern snapshot existant.)
- **Heuristique saison close** : ne PAS rafraîchir une saison terminée (`end_date < now` ET déjà sync après). Économie d'appels API.
- **DÉCISION À FIGER UNE FOIS** : `upsert` (avec hooks) OU `delete-replace` (sans hooks) pour TOUS les snapshots (CSR/medals/SR). Leafapp a mélangé les deux = dette. On choisit un seul pattern.

### Médailles (parité HI) — pattern sprite-sheet
- Metadata = `{id, name, classification (Multi-kill/Spree/Style/Vehicle/Objective), difficulty, spriteSheetUri, left/top/width/height}`. **On a déjà seedé ça** (h5-metadata-fetch → medal_definitions). 
- Rendu = **CSS `background-position`** (zéro canvas, GPU-friendly) : `<MedalSprite>` avec le sprite_location.
- Affichage match : carnage `MedalAwards[]{medalId, count}` → grille d'icônes + badge count si >1. Carrière = SUM des counts.
- **Reste à faire = ingérer MedalAwards du carnage + l'affichage** (la metadata est faite).

### REQ packs — patterns + LE piège
- Taxonomie : **catégorie → sous-catégorie → rareté** (Common/Uncommon/Rare/UltraRare/Legendary + mythic) ; `use_type` (Consumable/Durable/Boost) ; `flair` (New/Hot/LeavingSoon/Featured). % collection = `owned/total` par rareté.
- **PIÈGE MAJEUR (Pious Academic)** : **les IDs de REQ packs ne sont PAS auto-découvrables** — aucun endpoint ne liste tous les packs. → **seed manuel obligatoire** (datamine / liste curée). C'est le vrai coût de la feature REQ-as-progression.

### Pépites que la 1re passe avait ratées
- **Routing par slug gamertag** (SEO-friendly) plutôt que xuid — on fait déjà ça côté front.
- **Rate limit API officielle stricte** : ~10 req/10s, 600/600s → token bucket + backoff (on a déjà du backoff, à documenter par endpoint).
- **Emblem profil = 302 redirect** : extraire le header `Location` (URL CDN), NE PAS suivre le redirect.
- **MatchEvents undocumented** mais porte kill-logs + positions (death recap / heatmaps) — bas priorité, mais réel.
- **Visibilité playlist par saison** : une playlist peut être ranked en S1, social en S2 → flag `is_visible` par (saison, playlist), pas une blacklist globale.
- **GameMode = int** (1=Arena,2=Campaign,3=Custom,4=Warzone) → enum Go typé, pas de magic number.

### Ce que Leafapp N'a PAS (limite)
Leafapp = **leaderboards/CSR uniquement** — PAS de match-detail, medals, weapons. Donc c'est une réf pour le schéma CSR/leaderboard, pas pour medals/match (qu'on a déjà, nous).
