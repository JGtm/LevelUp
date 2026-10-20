# PLAN_H5_LEADERBOARD.md — CSR Halo 5 (par playlist + leaderboard mondial), en réutilisant l'existant

> Basé sur la cartographie du leaderboard CSR mondial **Infinite** existant (workflow 2026-06-24).
> Directive user : NE PAS réinventer le refresh (délicat). Réutiliser au max.

## ⚠ VERDICT SONDE (2026-06-24) — (e) leaderboard mondial = BLOQUÉ
Sonde read-only de l'API officielle (clé Ocp-Apim) :
- `GET /metadata/h5/metadata/seasons` + `/playlists` → **200** (35 saisons, playlists avec isRanked/isActive, ids GUID). Metadata OK.
- `GET /stats/h5/leaderboards/csr/{seasonId}/{playlistId}` → **404 sur TOUS les combos**, y compris la saison ACTIVE (« Evergreen Season », isActive=true) × ses 7 playlists classées, et plusieurs saisons passées.
→ **L'endpoint leaderboard CSR officiel H5 est mort** (343 a retiré le service stats compétitif H5). L'agent de recherche l'avait INFÉRÉ du doc sans l'appeler. **(e) world leaderboard H5 n'est PAS faisable** (ni officiel, ni scraping halowaypoint qui n'a plus de pages H5).

**PIVOT : on se concentre sur (a) CSR par playlist (par-joueur)** — source = `ServiceRecordArena.ArenaPlaylistStats[]` (endpoint INTERNE spartanstats.svc, SpartanToken, qu'on appelle DÉJÀ pour le SR/XP). Indépendant de l'endpoint mort. La réutilisation de la stack leaderboard Infinite (§2-§3) ne s'applique qu'à (e) → **désormais hors scope**.

---

## 0. Stratégie d'écriture = APPEND-ONLY (confirmé)
Le leaderboard Infinite écrit **append-only** : `world_csr_leaderboard_snapshots` (PK séquence + `fetched_at` + `written_at`), **zéro DELETE/UPDATE**, lecture via vue **`world_csr_leaderboard_latest`** (max `fetched_at` par season/playlist → snapshot cohérent, pas de « Frankenstein »). Stats d'enrichissement : `world_player_season_stats` (a **déjà** une colonne `title_slug`, vue `_latest` partitionnée par titre).
→ **Le delete/replace de Leafapp est PROSCRIT chez nous.** On garde l'append-only.

## 1. Deux sources H5 DISTINCTES
- **(e) Leaderboard mondial** : classement CSR top-N par playlist/saison. **Source = API OFFICIELLE** `GET www.haloapi.com/stats/h5/leaderboards/csr/{seasonId}/{playlistId}?count=N` (clé `Ocp-Apim-Subscription-Key`). **Décision : API officielle, PAS scraping** (l'Infinite scrape halowaypoint.com mais H5 n'y a plus de pages). Le fetch produit le MÊME `[]LeaderboardEntry` → réutilise toute la stack downstream.
- **(a) CSR par playlist (par joueur)** : CSR courant du joueur par playlist. **Source = `ServiceRecordArena.ArenaPlaylistStats[]`** (endpoint INTERNE spartanstats.svc, SpartanToken — qu'on appelle déjà, juste pas parsé). Surface = page carrière H5. **Indépendant de (e).**

## 2. Réutilisation : déjà title-agnostic vs à généraliser
**Déjà agnostique (réutiliser tel quel)** : domain types (`LeaderboardEntry/Request/Response.TitleSlug`), repo read (`GetCSRWorldLeaderboard`, lit la vue `_latest`), handler HTTP (lit `title_slug` en query param), `world_player_season_stats` (colonne title_slug), vues `_latest`, catalogue API, **front `LeaderboardBlock.tsx`** (passe title_slug, catalogue via API).

**HINF-hardcodé à généraliser/dupliquer** :
1. `defaultLeaderboardTitleSlug = "halo_infinite"` (world_player_stats_repo.go) → threader le titre.
2. `rankedplaylists.Active()` (Infinite only) → créer `internal/games/halo_5/rankedplaylists/` (asset_ids classés H5).
3. La SOURCE de fetch (scraper Infinite) → pour H5 = adapter API officielle (nouveau, produit `[]LeaderboardEntry`).
4. Cascade de résolution des noms de playlist → param titre.

## 3. Étapes (réutilisation maximale)
- **P0 (de-risk)** : SONDER l'API officielle leaderboard (confirmer qu'elle répond + le shape réel). Toute la feature (e) en dépend.
- **P1** : `internal/games/halo_5/rankedplaylists/` (asset_ids + saisons H5, depuis `/metadata/h5/metadata/seasons`).
- **P2** : fetch adapter H5 (`internal/games/halo_5/leaderboard_fetch.go`) → appelle l'API officielle → `[]domain.LeaderboardEntry` (même type). PAS de scraping.
- **P3** : généraliser le repo read (threader title_slug, retirer le default hardcodé).
- **P4** : job/cron H5 (clone `cmd/snapshot-world-leaderboard` ou param `-title`) → INSERT append-only dans la table partagée.
- **P5** : front — valider que le sélecteur titre passe `title_slug` jusqu'au bout (probablement déjà OK).
- **(a) séparé** : parser `ArenaPlaylistStats[]` + table per-joueur + carte carrière H5.

## 4. Risques / pièges
- **API officielle leaderboard inconfirmée** → P0 obligatoire avant tout.
- **Collision season_id/playlist_id** entre titres dans la table partagée → vérifier que les asset_ids H5 ≠ Infinite (UUID uniques) ; sinon ajouter `title_slug` à `world_csr_leaderboard_snapshots`.
- **Enrichissement stats H5** peut être vide (tokens) → CSR/rang/tier restent affichables, stats best-effort.
- **Append-only = gonflement disque** (accepté, doctrine ART ; lecture via `_latest`, pas de full scan).
- **Rate-limit officiel** (~10 req/10s, 600/600s) → backoff + cadence quotidienne.
