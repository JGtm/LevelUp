# Plan — Leaderboard mondial enrichi (stats joueur par saison/playlist)

> **Créé le** : 2026-06-09 · **Mis à jour** : 2026-06-14 (enrichissement cron dé-gaté : flag `LEVELUP_WORLD_ENRICH` + token dédié supprimés → toujours actif via le pool db_profiles)
> **Statut** : ✅ Phases A→F LIVRÉES + enrichissement cron actif par défaut (pool multi-token, plus de flag). Reste : backfills **historiques** one-shot via CLI (la saison courante est tenue à jour par le cron).
> **Branche de travail** : `feat/world-leaderboard-enriched` — créée **depuis `feat/leaderboard-csr-followup`**, PAS `main`. ⚠️ Vérifié 2026-06-10 : `main` n'a PAS l'infra leaderboard (scraper/cron/repo/migrations absents ; branche courante 60 commits devant `main`) → brancher depuis `main` ne compilerait pas. L'infra vit sur `feat/leaderboard-csr-followup`.
> **Données** : les binaires `cmd/` (probe, backfill) tournent contre le `data/` du repo principal via flags `--shared-db`/`--tokens-dir` (les DB sont gitignored, donc absentes des worktrees).

---

## Addendum 2026-06-10 — revue de plan (raffinements validés)

Revue conjointe concept + code. Le cœur est sain (probe-first, compteurs bruts + ratios dérivés, réutilisation infra, piège `xuid(NNN)` intégré). Raffinements :

- **Volume / tokens** : le backfill **hérite du pool de tokens joueurs + du rate limiter app** (pas un seul compte) → charge répartie + survie aux rotations de token (le `RefreshLoop` du pool re-dérive). Run **non contraint à 1 jour** : checkpoint reprenable → étalé sur fenêtres creuses. Les ~400k appels deviennent acceptables dans cet envelope.
  - **À confirmer en Phase A (point clé)** : un binaire `cmd/` séparé **n'hérite PAS** du pool construit au boot serveur — il doit le **reconstruire** (charger `data/auth/watcher_tokens/*`, instancier pool + rate limiter comme `main.go`). Valider que le probe tape sur le pool, pas un token unique.
- **Service-record agrégé** : **rule-out explicite en Phase A** (5 min). Connaissance métier : ne fournit pas la granularité saison×playlist voulue → l'itération match-par-match reste nécessaire. Documenter le résultat.
- **Logging (manquait)** : `slog` structuré vers `logs/` sur tout le backfill — progression/saison, reprise checkpoint, `xuid_miss`, `coverage_gap`, rate-limit.
- **Tests** : (a) requête `LAG()` inter-saison → **test d'intégration DuckDB** avec dataset multi-saison **hétérogène** (playlists qui apparaissent/disparaissent) ; (b) accumulateur de compteurs → **fonction pure `internal/analysis/`** testée sans API (le service orchestre seulement).
- **DRY** : réutiliser le parsing match-stats existant (`transforms.go`) — ne pas ré-extraire le `map[string]any` à la main.
- **Écriture shared (ART)** : `world_player_season_stats` en UPSERT → soit adopter le pattern append-only + vue `_latest` (comme `match_skill_rank`/`csrs`), soit **justifier le single-writer** dans la migration + vérifier l'allowlist `no_art_patterns_test`.
- **Frontend / chemins** : nouvelles strings (Matchs, V%, K/D/A, Temps, K/min, Δ rang, podium, tooltips trend) → **i18n FR+EN** ; chemin checkpoint via **PathResolver**, pas `filepath.Join` direct sur `data/`.

---

## Phase A — résultats du probe (2026-06-10)

Probe `cmd/probe-world-stats` exécuté (token-bearer JGtm, échantillon csrseason13-2, serveur stoppé). **Concept validé** ; 1 finding bloquant pour Phase B.

**Validé :**
- **Auth PeopleHub** : `auth.RefreshUserXSTS(ctx, store, xuid)` → header `XBL3.0` (audience `http://xboxlive.com`) directement utilisable. **Zéro nouveau code auth.**
- **Résolution xuid** : 100% sur l'échantillon (Gun Uchiha, Wacki Rz) via PeopleHub, match exact case-insensitive.
- **Fetch joueur mondial (inconnu)** : `GetMatchHistory`/`GetMatchStats` OK avec les tokens d'un tiers (JGtm).
- **Chemins extraction (Phase B)** : `Players[]` ; cible par `PlayerId == "xuid(N)"` ; `Outcome` **numérique** (2=win/3=loss) ; stats dans `PlayerTeamStats[0].Stats.CoreStats` (`Kills`/`Deaths`/`Assists`/`PersonalScore`) ; `ParticipationInfo.TimePlayed` = **durée ISO-8601** (`PT10M39.203S` → parser en s) ; `MatchInfo.SeasonId` présent. → réutiliser `transforms.go::findCoreStats`.
- **Timing** : ~0.8–1.0 s / `GetMatchStats` → backfill ~390k ≈ **88–110 h mono-thread** → pool + étalement off-peak obligatoires.

**✅ Filtrage playlist — RÉSOLU (faux problème détecté en v1)** : les ids snapshot SONT des `Playlist.AssetId` (même espace de noms que les matchs), confirmé par `rankedplaylists.go` ET le blog den.dev (16 playlists actives, asset+version ids) : `edfef3ac`=Ranked Arena, `dcb2e24e`=Ranked Slayer, `c94cb508`=Ranked Legacy. Le « mismatch » initial venait du probe v1 (qui ne filtrait pas) + d'un échantillon dont les matchs récents étaient en Arena. **Le bucketing `match.Playlist.AssetId == snapshot.playlist_id` fonctionne.**

**✅ Extraction + bucketing validés end-to-end (probe v2, 2026-06-10)** : Gun Uchiha (classé Legacy) — 15 matchs récents = 8 Arena + 1 Tactical + 6 Legacy, correctement attribués/agrégés :
- Ranked Arena 23m 7W-13L-3T KDA 1.03 V%30 ; RANKED LEGACY 6m 4W-2L KDA 1.78 V%67 ; Ranked Tactical 1m.
- Chemins extraction confirmés en exécution : `Outcome` (2=W/3=L/1=T), `PlayerTeamStats[0].Stats.CoreStats` (Kills/Deaths/Assists/PersonalScore), `ParticipationInfo.TimePlayed` ISO→s.
- ℹ️ Nuance : matchs `Outcome=1` (tie) à 0/0/0 (joueur non engagé / forfait) — à filtrer ou compter à part en Phase B.

**✅ Dimension SAISON validée (depth-scan 2026-06-10)** : pagination de 500 matchs (20 pages, `GetMatchHistory` sans stats) sur Gun Uchiha → couvre **2025-09-14 → 2026-06-05 (~9 mois)**, et le **plus ancien match est en `CsrSeason12-1`** (≥2 saisons avant la courante). Donc : (1) `GetMatchHistory` **remonte loin** — les saisons passées sont atteignables, pas de limite de profondeur bloquante ; (2) **attribution saison via `MatchInfo.SeasonId`** (`"Csr/Seasons/CsrSeason12-1.json"`, présent dans chaque match) — robuste, self-describing.
- ⚠️ **`csr_season_calendars` VIDE** dans la metadata.duckdb dev (serveur stoppé). **Design Phase B/C** : attribuer la saison via `MatchInfo.SeasonId` (par match), **PAS** par fenêtrage de dates calendrier. Le calendrier (peuplé par le serveur depuis Waypoint au runtime) sert au plus à **borner la pagination** — dérivable aussi des snapshots + SeasonId.
- ✅ **Historique complet atteignable (deep-scan 250 pages, 2026-06-10)** : remonté à **2021-11-15 (lancement HI)**, **4870 matchs** pour Gun Uchiha → pas de limite de profondeur bloquante, la Saison 1 est paginable. **3 caveats** : (a) **429 rate-limit** dès ~`start=3775` en pagination rapide → throttle obligatoire au backfill ; (b) **`MatchInfo.SeasonId` varie selon l'époque** — récents ranked = `Csr/Seasons/CsrSeasonX-Y` (fiable), mais le match 2021 portait `Seasons/Season6.json` (format content-season, douteux) → attribution CSR fiable seulement pour les matchs récents ; (c) **volume ~5000 matchs/joueur actif** pour l'historique complet → backfill toutes-saisons = semaines off-peak. **Mitigation** : n'enrichir que les saisons présentes dans les snapshots (récentes, SeasonId fiable) ; remonter à 2021 est inutile (rien à enrichir avant le 1er snapshot).

**⚠️ Auth — leçon (incident probe 2026-06-10)** : NE PAS enchaîner deux refresh (ex. `RefreshUserXSTS` MSAL **+** `RefreshHaloTokensViaStoreFirst` RT-brut) — double rotation de RT à usage unique → churn. **Un seul** `access_token` (chemin adapté à la forme du token : MSAL silent OU `ExchangeRefreshTokenWithRotation` pour les RT bruts type JGtm), puis dériver XSTS RTA (PeopleHub) **et** Halo de ce même token. Persister le RT tourné. Critique pour Phase D (backfill = beaucoup de refresh).

**💡 Insight efficacité (Phase C/D)** : un joueur joue plusieurs playlists → **fetcher sa saison UNE fois et bucketer par `Playlist.AssetId`** couvre toutes ses playlists d'un coup. Charger l'**UNION** des joueurs d'une saison (pas le produit saison×playlist×joueur) → évite de re-fetcher le même joueur N fois.

**Enrichissement catalogue — Phase F OBLIGATOIRE (dernière phase du plan, décision user 2026-06-10)** : le blog den.dev documente `GetPlaylist` (`…/hi/playlists/{id}/versions/{ver}?clearanceId=`) → `PlaylistEntries[]` {`MapModePairAssetId`, `Weight`}. Remplace/enrichit les 16 entrées hardcodées de `rankedplaylists.go` par la liste live + poids map/mode (et découvre les nouvelles playlists comme Ranked Legacy, absente du blog 2023). **Pas optionnel** : étape finale obligatoire, mutualisée avec les autres sections de l'app qui consomment le catalogue playlists. Détail à spécifier en Phase F.

**Notes :** certains (saison, playlist) snapshot ont très peu d'entrées (ex. 2 gamertags) — complétude de scraping variable, à surveiller en Phase D.

---

## Objectif

Enrichir le classement CSR mondial (200 → 100 joueurs) avec des stats réelles par saison ET par playlist :
FDA, temps de jeu, matchs joués, taux de victoire (dérivé), K/D/A, évolution de rang vs saison précédente,
podium SVG flat top 3.

---

## Ce qui existe déjà (vérifié 2026-06-09)

| Composant | Fichier | État |
|-----------|---------|------|
| Domain types | `internal/domain/leaderboard.go` | ✅ Exists — `LeaderboardEntry`, `LeaderboardCategory` |
| World repo (lecture + snapshot) | `internal/platform/duckdb/leaderboard_world_repo.go` | ✅ Exists — `GetCSRWorldLeaderboard`, `InsertWorldCSRSnapshot`, `GetWorldLeaderboardCatalog` |
| Migration table CSR | `internal/migration/steps_world_csr_leaderboard.go` | ✅ Exists — `world_csr_leaderboard_snapshots` + vue `world_csr_leaderboard_latest` |
| Migration vue batch | `internal/migration/steps_world_csr_leaderboard_latest_by_batch.go` | ✅ Exists |
| Scraper Halo Waypoint | `internal/platform/halo/leaderboard_scraper.go` | ✅ Exists |
| Cron daily | `internal/scheduler/world_leaderboard_cron.go` | ✅ Exists |
| Service | `internal/service/leaderboard_service.go` | ✅ Exists |
| Handler HTTP | `internal/api/handlers/leaderboard.go` | ✅ Exists |
| Client `GetMatchHistory` | `internal/sync/halo_client.go:197` | ✅ Exists — signature connue |
| Client `GetMatchStats` | `internal/sync/halo_client.go:249` | ✅ Exists — retourne `map[string]any` brut |

**À créer (Phases B–E) :**
- `internal/migration/steps_shared_world_player_season_stats.go`
- `internal/platform/duckdb/world_player_stats_repo.go` (UpsertPlayerSeasonStats)
- `internal/service/world_player_stats_aggregator.go`
- `cmd/backfill-world-player-stats/main.go`
- Extensions frontend (Phase E)

**Dettes existantes à corriger avant/pendant Phase B (multi-titre + "Toutes") :**
- Migration `add_title_slug_to_world_csr_leaderboard` : ADD COLUMN `title_slug VARCHAR NOT NULL DEFAULT 'halo_infinite'`, recréer vue avec `PARTITION BY title_slug, season_id, playlist_id, rank`, update index
- Toutes les queries du repo (`GetCSRWorldLeaderboard`, `GetStatLeaderboard`, `WorldCSRSnapshotAge`, `GetWorldLeaderboardCatalog`) : ajouter `WHERE title_slug = ?`
- `InsertWorldCSRSnapshot` : ajouter `title_slug` dans l'INSERT
- `GetWorldLeaderboardCatalog` : ajouter param `titleSlug` + retourner `{ID:"", DisplayName:"Toutes"}` en tête de `Playlists`
- `GetCSRWorldLeaderboard` : quand `playlist == ""` → query sans filtre playlist, MAX(csr_value) par gamertag, re-rank ROW_NUMBER()
- `playlistDisplayName()` : remplacer import `rankedplaylists` Halo-only par un callback injecté `PlaylistNameResolver func(string) string`
- Frontend `LeaderboardBlock.tsx` : état initial `playlist = ""`, `""` est une valeur valide (pas un fallback), label "Toutes" vient du catalogue API

---

## Contraintes API critiques (vérifiées dans le code)

### `GetMatchHistory` — signature réelle
```go
func (c *HaloAPIClient) GetMatchHistory(
    ctx context.Context,
    gamertag, matchType string,  // gamertag DOIT être fmt.Sprintf("xuid(%s)", xuid)
    start, count int,            // count max = 25
) ([]MatchHistoryEntry, error)

type MatchHistoryEntry struct {
    MatchID   string
    StartTime string   // ISO 8601
}
```

**⚠️ CRITIQUE** : le param `gamertag` doit être `xuid(NNN)` — pas un gamertag textuel.
L'API retourne une réponse stale figée (pas 404) si on passe un gamertag texte.
→ Toujours résoudre gamertag → xuid **avant** d'appeler GetMatchHistory.

**`GetMatchHistory` ne retourne que `MatchID + StartTime`** — aucune info playlist.
Le pré-filtrage par playlist avant `GetMatchStats` est impossible.

### `GetMatchStats` — retourne `map[string]any` brut
```go
func (c *HaloAPIClient) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error)
```
Il faut extraire manuellement : kills, deaths, assists, outcome, playtime, playlist, medals.
Les chemins exacts dans la map doivent être vérifiés sur fixture ou appel live avant implémentation.

### Rate limiting
Transparent — géré par `rate.Limiter` dans le client + pooling de tokens existant. Rien à configurer.

---

## Schéma de la table `world_player_season_stats`

**Dans `shared_matches_v2.duckdb`.** UPSERT safe — writer unique (cron OU script, jamais en même temps).
PK naturelle : `(gamertag, season_id, playlist_id)`.

**Décision win_rate (2026-06-09)** : ne pas stocker `win_rate` comme colonne.
Stocker `match_count` + `win_count` (+ optionnellement `tie_count`, `dnf_count`) et **dériver win_rate**
au moment de la lecture (query layer ou API). Cela vaut aussi pour KDA et les stats par minute —
stocker les compteurs bruts, dériver les ratios à la lecture.

```sql
CREATE TABLE world_player_season_stats (
    title_slug      TEXT      NOT NULL DEFAULT 'halo_infinite',
    gamertag        TEXT      NOT NULL,
    season_id       TEXT      NOT NULL,
    playlist_id     TEXT      NOT NULL,  -- '' = agrégat toutes playlists
    match_count     INTEGER   NOT NULL DEFAULT 0,
    win_count       INTEGER   NOT NULL DEFAULT 0,
    loss_count      INTEGER   NOT NULL DEFAULT 0,
    tie_count       INTEGER   NOT NULL DEFAULT 0,
    dnf_count       INTEGER   NOT NULL DEFAULT 0,
    kills           BIGINT    NOT NULL DEFAULT 0,
    deaths          BIGINT    NOT NULL DEFAULT 0,
    assists         BIGINT    NOT NULL DEFAULT 0,
    playtime_s      BIGINT    NOT NULL DEFAULT 0,
    medal_count     BIGINT    NOT NULL DEFAULT 0,
    computed_at     TIMESTAMP NOT NULL,
    PRIMARY KEY (title_slug, gamertag, season_id, playlist_id)
);
```

**Colonnes dérivées à calculer au moment de la lecture (jamais stockées) :**
- `win_rate = win_count::DOUBLE / NULLIF(match_count, 0)`
- `kda = (kills + assists::DOUBLE / 3) / NULLIF(deaths, 0)`
- `kills_per_min = kills::DOUBLE / NULLIF(playtime_s / 60.0, 0)`
- `deaths_per_min = deaths::DOUBLE / NULLIF(playtime_s / 60.0, 0)`
- `assists_per_min = assists::DOUBLE / NULLIF(playtime_s / 60.0, 0)`

## Indicateur de progression inter-saison

**Principe** : pour chaque joueur × playlist, comparer les stats de la saison courante avec
la saison précédente **où la même playlist existait**. Si la playlist n'existait pas en saison N-1
on remonte automatiquement à N-2, N-3, etc. Si aucune saison antérieure trouvée → pas d'indicateur.

**Pas de nouvelle colonne** — tout se calcule au moment de la lecture via `LAG()`.
`PARTITION BY title_slug, gamertag, playlist_id ORDER BY season_id` saute naturellement les saisons manquantes.
Quand `playlist_id = ""` (toutes) : fenêtre `PARTITION BY title_slug, gamertag ORDER BY season_id`.

**Query pattern (lecture enrichie) :**
```sql
WITH hist AS (
    SELECT
        gamertag, season_id, playlist_id,
        match_count, win_count, kills, deaths, assists, playtime_s, medal_count,
        -- saison précédente avec la même playlist (saute les saisons manquantes)
        LAG(season_id)   OVER w AS prev_season_id,
        LAG(match_count) OVER w AS prev_match_count,
        LAG(win_count)   OVER w AS prev_win_count,
        LAG(kills)       OVER w AS prev_kills,
        LAG(deaths)      OVER w AS prev_deaths,
        LAG(assists)     OVER w AS prev_assists
    FROM world_player_season_stats
    WINDOW w AS (PARTITION BY gamertag, playlist_id ORDER BY season_id)
)
SELECT
    *,
    -- ratios courants
    win_count::DOUBLE / NULLIF(match_count, 0)                        AS win_rate,
    (kills + assists / 3.0) / NULLIF(deaths, 0)                       AS kda,
    -- ratios saison précédente
    prev_win_count::DOUBLE / NULLIF(prev_match_count, 0)              AS prev_win_rate,
    (prev_kills + prev_assists / 3.0) / NULLIF(prev_deaths, 0)        AS prev_kda,
    -- indicateur directionnel (NULL si pas de saison précédente)
    CASE
        WHEN prev_season_id IS NULL THEN NULL
        WHEN (kills + assists/3.0)/NULLIF(deaths,0)
           > (prev_kills + prev_assists/3.0)/NULLIF(prev_deaths,0) THEN 'up'
        WHEN (kills + assists/3.0)/NULLIF(deaths,0)
           < (prev_kills + prev_assists/3.0)/NULLIF(prev_deaths,0) THEN 'down'
        ELSE 'stable'
    END AS kda_trend,
    CASE
        WHEN prev_season_id IS NULL THEN NULL
        WHEN win_count::DOUBLE/NULLIF(match_count,0)
           > prev_win_count::DOUBLE/NULLIF(prev_match_count,0) THEN 'up'
        WHEN win_count::DOUBLE/NULLIF(match_count,0)
           < prev_win_count::DOUBLE/NULLIF(prev_match_count,0) THEN 'down'
        ELSE 'stable'
    END AS win_rate_trend
FROM hist
WHERE season_id = ?
  AND gamertag  = ?
  AND playlist_id = ?
```

**Types Go (extension `WorldPlayerSeasonStats`) :**
```go
// Courant
WinRate      *float64
KDA          *float64
KillsPerMin  *float64
// Comparaison inter-saison (nil = pas de saison précédente avec cette playlist)
PrevSeasonID  *string  // ex : "csrseason11-2" — permet à l'UI d'afficher "vs S11"
PrevWinRate   *float64
PrevKDA       *float64
KDATrend      *string  // "up" | "down" | "stable" | nil
WinRateTrend  *string  // idem
```

**Types TypeScript (extension `LeaderboardEntry`) :**
```typescript
prev_season_id?:  string        // saison de référence pour l'indicateur
prev_kda?:        number        // KDA saison précédente
prev_win_rate?:   number        // win_rate saison précédente
kda_trend?:       'up'|'down'|'stable'   // null = pas d'historique
win_rate_trend?:  'up'|'down'|'stable'
```

**Rendu frontend** :
- `↑` (token `success`) / `↓` (token `outcome-loss`) / `=` (token `info`) / absent si null
- Tooltip : "vs [prev_season_id]" (ex : "vs Saison 11")
- Affiché sur les colonnes K/D/A et V% uniquement (pas sur les compteurs bruts)

---

## Volume d'appels API

### Backfill historique (one-shot)

Hypothèse : ~300 matchs/joueur/saison en moyenne (joueurs top, très actifs).

| Étape | Calcul | Appels |
|-------|--------|--------|
| Match history pages | 100 joueurs × 13 saisons × 300/25 = 12 pages | ~15 600 |
| GetMatchStats | 100 joueurs × 13 saisons × 300 matchs | ~390 000 |
| **Total** | | **~400 000** |

Le backfill tournera plusieurs heures voire une journée.

### Cron daily (delta, saison courante uniquement)

| Étape | Calcul | Appels |
|-------|--------|--------|
| GetMatchStats | 100 joueurs × ~5 nouveaux matchs | ~500 |

---

## Playlists par saison — source de vérité

`rankedplaylists.go` est une liste **statique** (16 entrées, `Active bool`) sans mapping saison → playlists.
Il n'existe **pas** de catalogue "playlist X active à partir de saison Y".

**Source de vérité réelle** : requêter `world_csr_leaderboard_latest` dans la shared DB :
```sql
SELECT DISTINCT season_id, playlist_id
FROM world_csr_leaderboard_latest
WHERE season_id <> '' AND playlist_id <> ''
ORDER BY season_id DESC, playlist_id
```
Ce résultat reflète ce qui a réellement été scrapé → c'est ça que le backfill doit couvrir.

---

## Phases d'implémentation

### Phase A — Probe E2E sur échantillon ⬅️ PROCHAINE ÉTAPE

**Nature** : script CLI Go de diagnostic, one-shot, **aucun INSERT**. Valide le process complet
et calibre le backfill avant d'écrire le moindre code de production.

**Fichier** : `cmd/probe-world-stats/main.go` (éphémère — peut être supprimé post-Phase A)

**Ce que Phase A doit valider :**
1. **XUID resolution via PeopleHub** : taux de succès sur les gamertags du top-100
2. **Chemins dans `Players[]` de `GetMatchStats`** : comment localiser le joueur par xuid et extraire kills/deaths/assists/outcome/playtime/medals (structure à confirmer sur données réelles)
3. **Format outcome** : numérique (WIN=2/LOSS=3/TIE=1/DNF=4) ou string ?
4. **Complétude des playlist IDs** : comparer les IDs rencontrés pendant le fetch avec `rankedplaylists.All()` → documenter les éventuels gaps
5. **Timing réel** : calibrer le volume et la durée du backfill historique
6. **Couverture** : si un joueur du snapshot n'a pas de match pour la playlist cible dans la fenêtre saison → rotation sur le joueur suivant du snapshot

**Chemins JSON déjà connus** (vérifiés dans `transforms.go`) :
- `GetMatchStats` → `MatchInfo.Playlist.AssetId` (playlist_id), `MatchInfo.Playlist.PublicName`
- Les chemins dans `Players[]` restent à confirmer sur données réelles

**Résolution XUID — via PeopleHub Xbox Live**

Endpoint : `GET https://peoplehub.xboxlive.com/users/me/people/search/decoration/detail,preferredColor?q={gamertag}&maxItems=25`

Headers :
```
x-xbl-contract-version: 3
Authorization: XBL3.0 x={user_hash};{xsts_token}
Accept-Language: en-us
```

Réponse : `{"people": [{"gamertag": "...", "xuid": "2535462389823105"}, ...]}`

Auth : chaîne MSAL → Xbox user token → XSTS (`RelyingParty: "http://xboxlive.com"`) → header `XBL3.0`.
**L'app fait déjà cette chaîne** pour les tokens Halo (même MSAL, relying party différent).
Point à confirmer en Phase A : est-ce que le XSTS générique (`http://xboxlive.com`) est déjà
exposé dans l'infra auth, ou faut-il en dériver un nouveau ? Si nouveau → ajouter une méthode
dans le package `auth` (pas de logique métier dans `auth`, cf. ADR 0023).

La recherche est fuzzy → filtrer le résultat sur `gamertag == cible` (exact, case-insensitive).

**Flags du script :**
- `--token-gamertag <gt>` — gamertag dont les tokens sont utilisés pour les appels API
- `--seasons <all|id1,id2>` — saisons à couvrir (défaut : toutes présentes dans le snapshot)
- `--playlists-per-season <n>` — max playlists par saison à tester (défaut : 3, production : toutes)
- `--matches <n>` — matchs à fetcher par joueur sélectionné (défaut : 10)
- `--max-candidates <n>` — max joueurs à essayer par (saison, playlist) avant SKIP (défaut : 5)
- `--dump-raw` — dump JSON brut du 1er GetMatchStats de la session

**Algorithme :**

```
ÉTAPE 0 — Matrice saison × playlist depuis le snapshot
  SELECT DISTINCT season_id, playlist_id
  FROM world_csr_leaderboard_latest
  WHERE season_id <> '' AND playlist_id <> ''
  ORDER BY season_id DESC, playlist_id
  → matrice complète des combinaisons à couvrir

ÉTAPE 1 — Pour chaque saison (du plus récent au plus ancien)
  Pour chaque playlist de cette saison (jusqu'à --playlists-per-season) :

    Lire les gamertags du snapshot pour (season_id, playlist_id) — ordre rank ASC (top d'abord)

    Pour chaque gamertag (jusqu'à --max-candidates sans résultat) :

      [XUID resolution]
      GET peoplehub.xboxlive.com/...?q={gamertag}
      → Trouver l'entrée dont gamertag correspond exactement (case-insensitive)
      → Extraire xuid
      Si échec (rate limit / non trouvé / réseau) → logger XUID_MISS, candidat suivant

      offset = 0
      accumulator = zéro pour (kills, deaths, assists, wins, losses, ties, dnf, playtime_s, medals)

      BOUCLE GetMatchHistory :
        GetMatchHistory("xuid("+xuid+")", "matchmaking", offset, 10)
        Pour chaque {MatchID, StartTime} :
          Si StartTime < saison.Start → STOP (fin de la fenêtre, matchs trop anciens)
          Si StartTime > saison.End   → SKIP (trop récent, hors fenêtre)

          GetMatchStats(matchID)
          → Extraire MatchInfo.Playlist.AssetId
          Si AssetId != playlist_id cible → SKIP (autre playlist)

          → Localiser xuid du joueur dans Players[]
          → Extraire kills, deaths, assists, outcome, playtime, medals
          → Accumuler

          Si --dump-raw ET 1er GetMatchStats de la session → logger map brute complète
          Logger tout AssetId rencontré → détection passive de gaps rankedplaylists

        Si len(résultats) < 10 → fin de l'historique du joueur, STOP boucle
        offset += 10

      Si accumulator.match_count > 0 → STOP rotation (on a des données pour ce joueur)

    Si aucun candidat n'a produit de données → logger COVERAGE_GAP (season, playlist)

ÉTAPE 2 — Rapport final
  Pour chaque (season_id, playlist_id) :
    gamertag, xuid_resolved, matches_found, source (api | gap)
    kills, deaths, assists, wins, losses, ties, dnf, playtime_s, medals
    win_rate (dérivé), kda (dérivé)
  Résumé global :
    timing total, timing moyen par match
    playlist IDs nouveaux vs rankedplaylists.All()
    (season, playlist) en COVERAGE_GAP
```

**Phase A terminée quand :**
- XUID resolution fonctionne (méthode PeopleHub validée ou alternative trouvée)
- Chemins kills/deaths/assists/outcome/playtime/medals confirmés sur données réelles
- Timing réel mesuré → calibrage durée backfill
- Gaps rankedplaylists documentés (ou confirmés vides)

### Phase B — Backend : migration + types + repo

**Migration** `internal/migration/steps_shared_world_player_season_stats.go`
- CREATE TABLE ci-dessus

**Domain** `internal/domain/leaderboard.go`
- Nouveau type `WorldPlayerSeasonStats` avec :
  - compteurs bruts : `MatchCount`, `WinCount`, `LossCount`, `TieCount`, `DnfCount int`, `Kills`, `Deaths`, `Assists`, `PlaytimeSec`, `MedalCount int64`
  - ratios courants (dérivés, nil si dénominateur nul) : `WinRate`, `KDA`, `KillsPerMin *float64`
  - comparaison inter-saison (nil si aucune saison précédente avec cette playlist) :
    `PrevSeasonID *string`, `PrevWinRate`, `PrevKDA *float64`, `KDATrend`, `WinRateTrend *string`
- Extension de `LeaderboardEntry` avec champs optionnels (pointeurs, `nil` = non enrichi) :
  `MatchCount *int`, `WinCount *int`, `LossCount *int`, `TieCount *int`, `DnfCount *int`,
  `Kills *int64`, `Deaths *int64`, `Assists *int64`, `PlaytimeSec *int64`, `MedalCount *int64`,
  `WinRate *float64`, `KDA *float64`, `KillsPerMin *float64`,
  `PrevSeasonID *string`, `PrevWinRate *float64`, `PrevKDA *float64`,
  `KDATrend *string`, `WinRateTrend *string`, `RankDelta *int`
  
**Repository** `internal/platform/duckdb/world_player_stats_repo.go` (nouveau fichier)
- `UpsertPlayerSeasonStats(ctx, db, []WorldPlayerSeasonStats) error` — INSERT OR REPLACE
- Ratios dérivés calculés dans le SELECT (pas dans l'INSERT)

**Extension `GetCSRWorldLeaderboard`** dans `leaderboard_world_repo.go`
- LEFT JOIN `world_player_season_stats` pour enrichir les entries
- `computeRankDelta` : compare rang saison N vs N-1 depuis `world_csr_leaderboard_snapshots`

### Phase C — Fetch : agrégateur + cron enrichi — ✅ CŒUR LIVRÉ (2026-06-10)

**Cœur pur** `internal/analysis/world_stats.go` (✅ + tests `world_stats_test.go`)
- `NormalizeSeasonID("Csr/Seasons/CsrSeason13-2.json") -> "csrseason13-2"` — **clé du design** : `MatchInfo.SeasonId` est un chemin, les snapshots stockent l'id court ; les matchs hors-CSR (`Seasons/SeasonN.json` → `seasonN`) ne matchent aucun snapshot CSR (ignorés, attendu).
- `ExtractPlayerMatchStat(matchJSON, xuid) (PlayerMatchStat, bool)` — chemins Phase A (`Players[]`/`PlayerId=="xuid(N)"`/`Outcome`/`PlayerTeamStats[0].Stats.CoreStats`/`ParticipationInfo.TimePlayed` ISO-8601).
- `AccumulateWorldStats(gamertag, []PlayerMatchStat) []WorldPlayerSeasonStats` — bucket par (saison, playlist), somme W/L/T/DNF + K/D/A + playtime. **0 API, 0 DB → unit-testable.**

**Agrégateur MULTI-TOKENS** `internal/service/world_player_stats_aggregator.go` (✅ + tests)
- `worldMatchSource` = sous-ensemble satisfait par **`*syncpkg.PooledHaloClient`** (assertion compile-time). `PooledHaloClient.GetMatchHistory/GetMatchStats` utilisent **`PolicyAnyPublic` (round-robin sur tous les tokens du pool)** → parallélisme natif sans code custom (directive « multi tokens one-shot »).
- `worldXUIDResolver` = `*auth.PeopleHubResolver` (résolution gamertag→xuid, single-token RTA, bas volume).
- `AggregatePlayer(ctx, gamertag)` : résout xuid → pagine `GetMatchHistory(xuid(N), "matchmaking", page*25, 25)` → `GetMatchStats` par match → extraction → accumulation.
- **Pas de fenêtre par dates** (calendrier vide en dev) : filtre `TargetSeasons` (saisons normalisées) + `StopAfterNonTarget` (arrêt anticipé une fois sous les saisons cibles, historique chronologique décroissant) + `MaxPages` (plafond dur).
- `Run(ctx, gamertags)` : fan-out parallèle borné par `Concurrency` (errgroup `SetLimit`), best-effort par joueur (un KO n'interrompt pas le batch ; erreurs collectées). Le RPS global reste plafonné par le pool.

**Résolveur** `internal/platform/auth/peoplehub_resolver.go` (✅ + tests httptest)
- `NewPeopleHubResolver(httpClient, headerFn)` ; `headerFn` fournit un header `XBL3.0 x=<hash>;<token>` RTA frais (le caller mémoïse/rafraîchit). `ResolveXUID` filtre sur correspondance **exacte** case-insensitive (pas de faux positif fuzzy).

**Enricher + cron (✅ livrés)** :
- `service/world_stats_enricher.go` (+ test) — `WorldStatsEnricher.EnrichSeason(ctx, season, gamertags)` : construit un agrégateur ciblé sur LA saison (TargetSeasons normalisé) à chaque cycle. Interfaces `WorldMatchSource`/`WorldXUIDResolver` exportées pour le wiring.
- `auth/cached_header_provider.go` (+ test) — `CachedHeaderProvider` : mémoïse le header RTA, rebuild après TTL (défaut 3h), thread-safe. Sa méthode `Header` = le `headerFn` du résolveur.
- `duckdb.WorldSeasonGamertags(ctx, db, season)` — gamertags distincts d'une saison (union toutes playlists, cf. insight Phase A : 1 joueur fetché couvre toutes ses playlists).
- `scheduler/world_leaderboard_cron.go` étendu — `WithStatsEnricher(e)` (optionnel, nil-safe) ; phase 5 `enrich()` après le snapshot CSR : lecture gamertags (RO) → `EnrichSeason` → `InsertPlayerSeasonStats` (fenêtre RW minimale, même discipline que le scrape). Best-effort.

**Wiring boot `cmd/server` — ✅ LIVRÉ (2026-06-11, gaté)** :
- Glue extraite dans `internal/worldenrich` (package partagé CLI + serveur, **une seule** implémentation de la résolution token store-first ADR 0023) : `BuildHaloSource`/`BuildMultiHaloSource` (param `eager` : fail-fast CLI vs lazy serveur), `BuildResolver` (PeopleHub via `CachedHeaderProvider` + `AcquireXSTSForRTA`), `BuildEnricher` (compose source + résolveur + `RankedPlaylistSet`).
- `cmd/server` : cron wire via `worldenrich.BuildEnricher(cfg, …)` **gaté par `LEVELUP_WORLD_ENRICH`** (OFF par défaut → scrape-only, comportement prod inchangé). Build **lazy** (`Eager:false`) : zéro résolution token au boot, différée au 1er tick (déjà dans la goroutine du cron, hors chemin de démarrage). Compte du header PeopleHub : `LEVELUP_WORLD_ENRICH_TOKEN` (sinon 1er compte `db_profiles`). `worldLbCron.WithStatsEnricher(enr)` avant le `go worldLbCron.Run`.
- **Activation prod délibérée** : poser `LEVELUP_WORLD_ENRICH=1` après validation du backfill (⚠️ nouvelle charge API Halo quotidienne à l'enrichissement).

**WorldLeaderboardCron étendu** `world_leaderboard_cron.go`
- Après `scrapeAll`, pour la **saison courante uniquement** :
  1. Résoudre gamertag → xuid (via `xuid_aliases`)
  2. `aggregator.FetchAndAggregate(ctx, xuid, currentSeason, lastCheckpoint)` — delta only
  3. `UpsertPlayerSeasonStats` dans la fenêtre RW minimale
- Timeout global cycle étendu : 30 min

### Phase D — Script backfill one-shot — ✅ LIVRÉ (2026-06-10)

**`cmd/backfill-world-player-stats/main.go`** (`//go:build cgo`) — reprenant, idempotent, MULTI-TOKENS. Build + vet OK.

**Multi-tokens** : pool construit comme `cmd/levelup/cmd_sync` (Discovery + Resolver + NewPool) → `NewPooledHaloClient(pool, "", "", rps)` (round-robin). Résolution xuid PeopleHub via UN compte (`-token-gamertag`) : header RTA mémoïsé (`CachedHeaderProvider`, both-shapes single-refresh comme le probe). Pilote `AggregatePlayer` par joueur dans un pool de workers (`-concurrency`) pour garder la granularité checkpoint/progression.

**Flags livrés :**
- `-token-gamertag <gt>` — **requis** (compte résolvant les xuid)
- `-season <id|all>` — défaut `all` (toutes les saisons des snapshots, récentes d'abord)
- `-limit <n>` — nb max joueurs/saison (0 = tous)
- `-concurrency <n>` (défaut 6) · `-rps <n>` (défaut 5/token) · `-max-pages <n>` (défaut 80 ; ↑ pour vieilles saisons)
- `-flush-every <n>` (défaut 20 : persiste + checkpoint tous les N joueurs)
- `-checkpoint <path>` (défaut `data/world_backfill_checkpoint.json`) · `-force` (ignore checkpoint) · `-dry-run`

**Arrêt / reprise / progression :**
- **Arrêt** : Ctrl-C (SIGINT/SIGTERM) → `signal.NotifyContext` annule le ctx → workers stoppent, lot en cours flushé + checkpoint sauvegardé, sortie propre.
- **Reprise** : relancer la MÊME commande → lit le checkpoint, skip les gamertags faits + les saisons complètes. Idempotent (append-only + vue _latest).
- **Progression** : ligne réécrite en place (`\r`) `[saison] done/total joueurs · lignes · err · elapsed`, + bilan par saison.

**Limite connue** : `medal_count` non extrait (reste 0) — `AccumulateWorldStats` somme match/win/k/d/a/playtime ; les médailles sont un suivi ultérieur (chemin `CoreStats`/`Medals` à confirmer).

**Backfill profond** : `StopAfterNonTarget=-1` (désactivé) → scanne jusqu'à `-max-pages`. Pour Season 1 (~4870 matchs ≈ 195 pages), passer `-max-pages 240+` et lancer off-peak.

~~**Flags (plan initial)** :~~ (remplacé par les flags livrés ci-dessus)

**Algorithme :**
1. Charger le catalogue de saisons via `GetWorldLeaderboardCatalog` (liste saisons + dates)
2. Pour chaque saison :
   a. Si données déjà présentes et pas `--force` → skip
   b. Charger checkpoint JSON si existant
   c. Lire `world_csr_leaderboard_latest` → liste gamertags (≤ 100)
   d. Pour chaque gamertag non complété dans checkpoint :
      - Résoudre gamertag → xuid (via `xuid_aliases` OU lookup API si absent)
      - Paginer `GetMatchHistory` depuis offset checkpointé
      - `GetMatchStats` pour chaque match dans la fenêtre saison
      - Agréger par playlist en mémoire
      - Upsert dans `world_player_season_stats`
      - Marquer joueur complet dans checkpoint
3. Marquer saison complète dans checkpoint

**Checkpoint** (1 JSON par saison, dans `data/checkpoints/backfill-world-{seasonID}.json`) :
```json
{
  "season_id": "csrseason12-1",
  "completed_gamertags": ["Gamertag1", "Gamertag2"],
  "in_progress": {
    "Gamertag3": { "last_match_offset": 75 }
  },
  "started_at": "2026-06-09T10:00:00Z",
  "season_completed": false
}
```

### Phase E — Frontend — ✅ LIVRÉ + POLISH (2026-06-11)

**Livré** (colonnes enrichies dans le tableau existant, approche incrémentale plutôt que podium/composants séparés — ces derniers reportés en polish) :
- `apps/web/src/lib/api/types.ts` — `LeaderboardEntry` étendu (19 champs d'enrichissement optionnels `?: T | null`).
- `apps/web/src/lib/i18n/manifests/common.toml` (+ régénération `generated/common.ts`) — clés `col_win_rate` / `col_kda` / `col_rank_delta` (FR + EN). `col_matches` réutilisé pour le nb de matchs.
- `apps/web/src/features/leaderboard/LeaderboardBlock.tsx` — colonnes **Parties / Victoires(%) + tendance / KDA + tendance / Δ rang** ajoutées à la branche CSR mondial, **affichées uniquement si `hasEnrichment`** (au moins un joueur backfillé → table CSR historique inchangée avant backfill). Tri client sur matchs/win_rate/kda. Tendances via `--narrative-trend-*` (pattern KPIStrip), Δ rang via `tokenCssVar(skillDeltaScale(delta))` — **zéro hex** (règle 20). Fallback `—` pour les entrées non enrichies (colonnes alignées).
- Test : `LeaderboardBlock.test.tsx` +2 cas (colonnes masquées sans enrichissement / valeurs affichées avec). **9/9 vitest verts, typecheck + lint OK.**

**Polish livré (2026-06-11)** :
- **Masquage colonnes mobile** — `COL_HIDE_SM`/`COL_HIDE_LG` (mêmes classes en-tête + cellules) : #, joueur, CSR toujours visibles ; victoires/KDA dès `sm` ; tier/parties/précision/rendement/Δrang dès `lg`. La table 10 colonnes ne déborde plus sur petit écran.
- **Accent podium top-3** — rang en gras + couleur pleine (vs muted) sur la cellule rang, **sans nouvelle couleur** (tokens `foreground`/`muted` existants — pas de `PodiumRow` séparé, choix éditorial flat aligné sur la pref data-viz). 
- **Tooltips "vs saison précédente"** — sur les flèches de tendance (victoires, KDA) et le Δrang. Clés i18n `trend_tooltip` / `rank_delta_tooltip` (FR + EN), manifests régénérés.
- **9/9 vitest verts, typecheck + lint OK.**

**Reporté** (vision riche, optionnel) : composants séparés `PodiumRow`/`RankDeltaBadge`/`RichLeaderboardRow` (l'accent top-3 inline couvre le besoin sans sur-composant).

#### Spécification initiale (référence)

**TypeScript types** `apps/web/src/lib/api/types.ts` — extension de `LeaderboardEntry` :
```typescript
// Compteurs bruts
match_count?:    number
win_count?:      number
loss_count?:     number
kills?:          number
deaths?:         number
assists?:        number
playtime_seconds?: number
medal_count?:    number
rank_delta?:     number | null
// Ratios courants (calculés côté API, nil si dénominateur nul)
win_rate?:       number
kda?:            number
kills_per_min?:  number
// Indicateur inter-saison (absent si aucune saison précédente avec cette playlist)
prev_season_id?:  string
prev_kda?:        number
prev_win_rate?:   number
kda_trend?:       'up' | 'down' | 'stable'
win_rate_trend?:  'up' | 'down' | 'stable'
```

**LeaderboardBlock.tsx**
- Limit CSR world : 200 → 100
- `PodiumRow` : top 3 en cartes (or/argent/bronze), gamertag, CSR, medals
- Colonnes table : `# | Joueur | Matchs | V% | K/D/A | Temps | K/min | Δ rang`
- `RankDeltaBadge` : ↑N (token `success`) / ↓N (token `outcome-loss`) / — (gris)
- `TrendBadge` : ↑ / ↓ / = avec tooltip "vs [prev_season_id]" — affiché sur V% et K/D/A
  - `null` (pas d'historique) → badge absent, pas de placeholder
  - Token `success` pour 'up', token `outcome-loss` pour 'down', token `info` pour 'stable'
  - **Jamais de rouge vif** — respecter la règle couleur sémantique (`tokenCssVar`)
- Colonnes stats/min masquées sur mobile (`hidden sm:table-cell`)
- Tri client sur toutes les colonnes

**Nouveaux composants** `apps/web/src/features/leaderboard/` :
- `PodiumRow.tsx`, `RichLeaderboardRow.tsx`, `RankDeltaBadge.tsx`

---

## Réutilisation de l'existant

| Existant | Réutilisé pour |
|----------|----------------|
| `GetMatchHistory` + `GetMatchStats` | Fetch stats par match |
| `world_csr_leaderboard_snapshots` | Calcul rank delta |
| `GetWorldLeaderboardCatalog` | Piloter le backfill (liste saisons + dates) |
| `InsertWorldCSRSnapshot` pattern (tx atomique) | Modèle pour `UpsertPlayerSeasonStats` |
| `xuid_aliases` (shared DB) | Résolution gamertag → xuid sans appel API |
| Rate limiter + pooling existants | Transparent |

---

## Statut des phases

| Phase | Statut | Notes |
|-------|--------|-------|
| A — Probe E2E échantillon | ✅ VALIDÉE (2026-06-10) | auth PeopleHub + xuid 100% + extraction + bucketing **playlist** + **dimension saison** (pagination 9 mois → CsrSeason12-1, attribution via `MatchInfo.SeasonId`) + timing ~0.9s/match. Design Phase B/C : attribuer par SeasonId (pas dates), auth single-token |
| B — Migration + types + repo | ✅ FAITE (2026-06-10) | Table append-only `world_player_season_stats` + vue `_latest` ; types `WorldPlayerSeasonStats` + `LeaderboardEntry` étendu ; repo (`InsertPlayerSeasonStats`, `GetWorldPlayerSeasonStats` LAG inter-saison, `loadPrevSeasonRanks`) ; `GetCSRWorldLeaderboard` enrichi (merge + RankDelta, best-effort). Tests :memory: verts |
| C — Agrégateur + cron | ✅ LIVRÉ (wiring boot 2026-06-11, gaté) | Cœur (2026-06-10, 11 tests) : `analysis/world_stats.go` (pur), `service/world_player_stats_aggregator.go` (multi-tokens, **dédup match-centric singleflight + ranked-only**), `world_stats_enricher.go`, `auth/peoplehub_resolver.go`, cron `WithStatsEnricher` + phase enrich. Wiring (2026-06-11) : `internal/worldenrich` (glue partagée CLI+serveur) + `cmd/server` gaté `LEVELUP_WORLD_ENRICH` (OFF par défaut, build lazy). Activation prod = poser le flag |
| D — Script backfill | ✅ LIVRÉ (2026-06-10) | `cmd/backfill-world-player-stats` multi-tokens (round-robin) + Ctrl-C→checkpoint + reprise idempotente + dédup match-centric + ranked-only + early-stop (`-deep` pour vieilles saisons). Bascule sur `internal/worldenrich` (2026-06-11). Build+vet OK. Limite : medal_count=0 (suivi). Lancé manuellement (off-peak) |
| E — Frontend | ✅ LIVRÉ + POLISH (2026-06-11) | Colonnes enrichies (Parties/Victoires+trend/KDA+trend/**Précision**/**Rendement-Résistance**/Δrang) conditionnelles à `hasEnrichment`. Polish : masquage colonnes mobile (sm/lg), accent podium top-3 (token-safe), tooltips "vs saison précédente" (i18n FR/EN). 9/9 vitest + typecheck/lint OK |
| F — Catalogue playlists | ✅ LIVRÉ (noms mutualisés, 2026-06-11) | `GetWorldLeaderboardCatalog` résout les noms via le **catalogue metadata partagé** (cascade `asset_translations[fr]` > rankedplaylists FR > `playlists_catalog` EN > id), nil-safe (fallback rankedplaylists) ; front préfère `display_name`. Test cascade vert. **Suivi optionnel** : `GetPlaylist` live (poids map/mode + auto-découverte) — séparé, le catalogue se peuple déjà via `populate-playlists-catalog` |

---

## Checklist de lancement

- [x] Créer branche `feat/world-leaderboard-enriched` depuis `feat/leaderboard-csr-followup` (worktree, 2026-06-10 — `main` n'a pas l'infra)
- [ ] Phase A : écrire `cmd/probe-world-stats/main.go` (flags `--shared-db`/`--tokens-dir` → data du repo principal)
- [ ] Phase A : confirmer que le CLI reconstruit bien le pool de tokens (pas un token unique)
- [ ] Phase A : rule-out service-record agrégé (granularité saison×playlist ?) — documenter
- [ ] Phase A : lancer avec `--dump-raw --playlists-per-season 3 --matches 10 --max-candidates 5`
- [ ] Phase A : confirmer les chemins kills/deaths/assists/outcome/playtime/medals dans GetMatchStats live
- [ ] Phase A : documenter les gaps rankedplaylists.go (IDs en match_registry absents de la liste statique)
- [ ] Phase A : noter le taux de résolution xuid via xuid_aliases (objectif > 80%)
- [ ] Phase A : noter le timing réel par match (base calibrage backfill)
- [ ] Phase A : documenter les (saison, playlist) sans couverture après rotation 5 candidats
- [ ] Phase B : migration + types + repo (schéma final basé sur Phase A)
- [x] Phase C : agrégateur + cron étendu (wiring boot gaté `LEVELUP_WORLD_ENRICH`, 2026-06-11)
- [x] Phase D : script backfill avec checkpoint
- [x] Phase D : test `--dry-run` sur 1 saison, vérifier reprise après kill
- [x] Phase E : frontend + polish (responsive, accent podium, tooltips saison)
- [x] Phase F : noms playlists mutualisés via catalogue metadata (cascade FR/EN, fallback rankedplaylists)
- [ ] `go test ./apps/go-api/internal/...` vert
- [ ] Boot serveur → migration appliquée
