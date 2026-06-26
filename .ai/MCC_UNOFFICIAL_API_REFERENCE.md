# Halo: The Master Chief Collection — Référence API officieuse

> Reverse-engineering du backend utilisé par le site officiel `www.halowaypoint.com`
> (section `halo-the-master-chief-collection`). Découvert + validé le 2026-06-26.
> Méthode : capture XHR navigateur sous login waypoint + minage du bundle JS
> (`chunk 2303` = client API MCC, 12 fonctions `getHaloMcc*`) + validation **headless**
> avec un SpartanToken v4 de notre pool via `apps/go-api/cmd/probe-mcc`.
>
> Statut : **non officiel**. Aucune garantie de stabilité (343/Microsoft peut changer
> sans préavis). Slug interne 343 du titre = `hmcc`. Xbox title id = `1144039928`.

---

## 1. Authentification

Identique à Halo Infinite / Halo 5 — **rien de spécifique à MCC** :

| Élément | Valeur |
|---|---|
| Header | `X-343-Authorization-Spartan: v4=<spartan_token>` |
| 343-clearance | **non requis** (pas de header clearance, pas de `?auth=st`) |
| Token | SpartanToken v4 **account-level** du pool (partagé inter-titres, comme H5) |
| Acquisition | `auth.RefreshHaloTokensViaStoreFirst(...)` — pipeline existant inchangé |
| CORS | `access-control-allow-origin: *` |
| User-Agent | non gaté (Chrome **et** `cpprestsdk/2.4.0` acceptés) |
| `Accept` | `application/json` |

> Le token de N'IMPORTE QUEL joueur de notre pool suffit pour lire les données
> **publiques** de n'importe quel autre joueur (voir §5 ownership).

---

## 2. Registre des services (waypoint, `__NEXT_DATA__.props.settings`)

Host MCC = **`mccapi.svc.halowaypoint.com`** (`mccGatewayService`). Registre complet
(pour contexte / endpoints connexes) :

| Clé settings | Host |
|---|---|
| `mccGatewayService` | `https://mccapi.svc.halowaypoint.com` |
| `profileService` | `https://profile.svc.halowaypoint.com` |
| `commsService` | `https://comms.svc.halowaypoint.com` |
| `waypointContentService` | `https://wpcontent.svc.halowaypoint.com` |
| `gameCmsService` | `https://gamecms-hacs.svc.halowaypoint.com` |
| `settingsService` | `https://settings.svc.halowaypoint.com` |
| `legacyGatewayService` | `https://legacy.svc.halowaypoint.com` |
| `halo5GatewayService` | `https://halo5api.svc.halowaypoint.com` |
| `haloStatsService` | `https://halostats.svc.halowaypoint.com` (Infinite) |
| `spartanStatsService` | `https://spartanstats.svc.halowaypoint.com` (Halo 5) |
| `skillService` / `economyService` / `packsService` / `voucherService` / `flightingService` / `haloPlayerService` / `haloCmsService` / `fireteamRavenService` / `iUgcService` / `iUgcAuthoringService` / `iUgcDiscoveryService` | (autres titres / transverses) |

Hosts d'assets associés :
- `https://emblems.svc.halowaypoint.com/hmcc/emblems/{config}` — emblèmes
- `https://gamecms-hacs.svc.halowaypoint.com/branches/hmcc/waypoint/data/images/levelicons/T{prestige}_L{level}.png` — icônes de rang
- `https://mccapi.svc.halowaypoint.com/assets/{seasonId}/{file}.png` — récompenses de saison
- `https://content.halocdn.com/media/Default/games/Halo-Master-Chief-Collection/avatars/...` — avatars

---

## 3. Convention de chemin

Préfixe `/hmcc/`. L'identité joueur est encapsulée : **`users/gt(<gamertag>)`**
(wrapper `gt(...)`, encodé `gt%28...%29`). **PAS** `players/`, **PAS** `xuid(...)`
(c'est cette combinaison host+convention inattendue qui rendait l'endpoint indevinable).
La réponse renvoie toujours le `xuid` résolu.

---

## 4. Endpoints MCC (`mccapi.svc.halowaypoint.com`) — surface complète

12 endpoints, **tous `GET`** (extraits 1:1 du client officiel, chunk 2303).
`{u}` = `gt(<gamertag>)`. `lang` défaut `en`.

### 4.1 Carrière / stats joueur

| Fonction client | Méthode + path | Accès |
|---|---|---|
| `getHaloMccServiceRecord` | `GET /hmcc/users/{u}/service-record` | public |
| `getHaloMccCampaignSummary` | `GET /hmcc/users/{u}/service-record/campaign-summary` | public |
| `getHaloMccCampaign` | `GET /hmcc/users/{u}/service-record/{campaign}/campaign` | public |
| `getHaloMccMatchHistory` | `GET /hmcc/users/{u}/matches?page=&pageSize=&categoryId=&title=` | public |
| `getHaloMccAchievements` | `GET /hmcc/users/{u}/achievements?lang=` | public |
| `getHaloMccSkillRanks` | `GET /hmcc/users/{u}/skill-ranks?platform=Xbox&hoppers=<id,id>` | public |

`{campaign}` ∈ `h1` (Halo CE), `h2` (Halo 2), `h3` (Halo 3), `h4` (Halo 4),
`odst` (Halo 3: ODST), `reach` (Halo: Reach).

### 4.2 Personnalisation / inventaire

| Fonction client | Méthode + path | Accès |
|---|---|---|
| `getHaloMccInventory` | `GET /hmcc/users/{u}/inventory` | **owner-only (403 sinon)** |

### 4.3 Métadonnées / catalogues (pas de joueur)

| Fonction client | Méthode + path |
|---|---|
| `getHaloMccRanks` | `GET /hmcc/ranks?lang=` |
| `getHaloMccMaps` | `GET /hmcc/maps?lang=` |
| `getHaloMccSeasons` | `GET /hmcc/seasons?lang=` |
| `getHaloMccSeason` | `GET /hmcc/seasons/{seasonId}?lang=` |
| `getHaloMccPlaylists` | `GET /hmcc/playlists?lang=` ⚠️ renvoie **500** (côté serveur, idem sur le site) |

---

## 5. Modèle d'accès (ownership) — validé

| Endpoint | Joueur tiers (token d'un autre compte) |
|---|---|
| service-record, campaign(-summary), matches, achievements, skill-ranks, catalogues | **200 (public)** |
| inventory | **403** (uniquement le propriétaire du token) |

> Conséquence « roster » : il n'y a **aucune liste d'adversaires par match** ni endpoint
> match-detail-par-id (contrairement à H5 `GET /h5/{mode}/matches/{id}` et Infinite
> `GET /{t}/matches/{id}/stats`). Pour des stats « roster », il faut interroger
> **chaque gamertag séparément** via `service-record` / `matches` (lecture publique).

---

## 6. Pagination des matchs — limite importante

- Paramètres : `page` (1-based), `pageSize`, `categoryId` (filtre), `title` (filtre).
- `pageSize=20`/`25` OK ; `pageSize=100` **rejeté (400)** → plafond pageSize bas (≤ ~25–50, exact à confirmer).
- Réponse : `maxPage` observé = **4** @ pageSize 25 même pour un joueur à 12 272 matchs.
  → **seuls ~100 matchs récents** sont accessibles, PAS l'historique complet.
- `categoryId=1` : filtre (ex. 6 914 / 12 272 matchs renvoyés). `title=Halo2` → **400**
  (le filtre `title` n'accepte pas la valeur `haloTitleId` brute ; format réel à déterminer).

---

## 7. Shapes de réponse (échantillons réels)

### service-record — `GET /hmcc/users/{u}/service-record`
```json
{
  "xuid": "2533274823512881",
  "xp": 524122300,
  "avatarSrc": "https://content.halocdn.com/.../playeridavatar_032-....jpg",
  "emblemSrc": "https://emblems.svc.halowaypoint.com/hmcc/emblems/white_white_skullking-on-blank",
  "clanTag": "Loading...",
  "skillRank": 14,
  "timePlayedSeconds": 7170501,
  "multiplayer": { "gamesPlayed": 12272, "wins": 8778, "losses": 3494,
                   "kills": 324348, "deaths": 115925, "assists": 44802 },
  "campaign": { "missionsCompleted": 710, "missionKills": 55455, "missionDeaths": 3507,
                "playlistsCompleted": 268, "playlistKills": 19333, "playlistDeaths": 672 }
}
```
> `xp` se résout en rang via `/hmcc/ranks` (seuils). `skillRank` = entier 1..50 (cf. §8).

### matches — `GET /hmcc/users/{u}/matches`
```json
{
  "xuid": "2533274823512881", "totalMatches": 12272, "pageSize": 25, "maxPage": 4,
  "matches": [
    { "score": 11, "datePlayed": "2026-06-03T00:35:42.793Z", "durationSeconds": 450,
      "mapId": 205, "won": true, "standing": 4, "kills": 12, "deaths": 8, "assists": 0,
      "headshots": 8, "medals": 19, "haloTitleId": "HaloReach", "gameCategoryId": 1 }
  ]
}
```
> Ligne agrégée par match (pas de `matchId`, pas de roster, pas de détail per-arme).
> `mapId` → `/hmcc/maps`. `haloTitleId` ∈ `Halo`/`Halo2`/`Halo3`/`Halo4`/`HaloReach`(/ODST).
> `gameCategoryId` observés : `0`, `1`, `22` (mapping exact non résolu ; `1` ≈ Slayer).
> Dernière ligne parfois vide/null (sentinelle de pagination).

### inventory — `GET /hmcc/users/{u}/inventory` (owner-only)
```json
{ "inventory": [], "virtualCurrency": 4 }
```
> `virtualCurrency` = Spartan Points. `inventory` = items débloqués (vide ici).

### seasons — `GET /hmcc/seasons`
```json
{ "seasons": [ { "seasonId": "S1", "season": 1, "seasonName": "NOBLE", "startDate": "2019-12-03T00:00:00Z" }, ... "S8 MYTHIC" ] }
```
8 saisons : S1 NOBLE, S2 SPARK, S3 RECON, S4 RECLAIMER, S5 ANVIL, S6 RAVEN, S7 ELITE, S8 MYTHIC.

### season/{id} — `GET /hmcc/seasons/S8`
```json
{ "seasonId": "S8", "season": 8, "seasonName": "MYTHIC", "startDate": "...",
  "tiers": [ { "tierId": "S8T01", "seasonPoints": 1,
    "rewards": [ { "itemId": "CUSTOMIZATION_H3_Helmet_Belos_Base", "itemClass": "H3_Customization",
                   "displayName": "Helmet - BELOS HERAN", "description": "...",
                   "imageUrl": "https://mccapi.svc.halowaypoint.com/assets/S8/T_Im_H3_Armor_Helmet_BelosHERAN.png" } ] } ] }
```
> `itemClass` ∈ `H3_Customization`, `Nameplate`, … (battlepass-like par tier).

### ranks — `GET /hmcc/ranks`
```json
{ "ranks": [ { "id": "R1", "threshold": 0, "level": 1, "prestige": 0, "name": "Rookie" },
             { "id": "R2", "threshold": 3000, "level": 2, "prestige": 0, "name": "Recruit" }, ... ] }
```
> Échelle XP cumulée (`threshold`) → `level` (1..) + `prestige` (0..10). Max ≈ `xp >= 1282608999` → T10.
> Icône : `gamecms-hacs.svc.../branches/hmcc/waypoint/data/images/levelicons/T{prestige}_L{level}.png` (ou `T10.png`).

### maps — `GET /hmcc/maps`
```json
{ "maps": [ { "id": "_map_id_halo1_pillar_of_autumn", "mapId": 0, "name": "THE PILLAR OF AUTUMN" }, ... ] }
```
> `mapId` (entier) = clé jointe aux lignes `matches`.

### campaign-summary — `GET /hmcc/users/{u}/service-record/campaign-summary`
```json
{ "h1": [ { "mapId": 0, "highestDifficultySinglePlayer": "Legendary",
            "highestDifficultySinglePlayerImageUrl": "...legendary-shield.png",
            "highestDifficultyCoop": "None", "highestDifficultyCoopImageUrl": "...none-shield.png" }, ... ],
  "h2": [...], "h3": [...], "h4": [...], "odst": [...], "reach": [...] }
```

### campaign/{x} — `GET /hmcc/users/{u}/service-record/h1/campaign`
```json
{ "xuid": "...", "missions": [ { "mapId": 0,
    "highestDifficultySinglePlayer": "Legendary", "highestDifficultyCoop": "None",
    "easy":      { "bestScoreSinglePlayer": null, "bestTimeMsSinglePlayer": null, "bestScoreCoop": null, "bestTimeMsCoop": null },
    "normal":    {...}, "heroic": {...},
    "legendary": { "bestScoreSinglePlayer": 5491, "bestTimeMsSinglePlayer": 2158500, "bestScoreCoop": null, "bestTimeMsCoop": null },
    "laso":      {...} }, ... ] }
```

### achievements — `GET /hmcc/users/{u}/achievements?lang=en` (~1 Mo)
```json
{ "achievements": [ { "id": "109", "serviceConfigId": "77290100-...", "name": "Life Story",
    "titleAssociations": [ { "name": "Halo: The Master Chief Collection", "id": 1144039928 } ],
    "progressState": "NotStarted",
    "progression": { "requirements": [ { "current": null, "target": "1", "operationType": "SUM", "valueType": "Integer" } ],
                     "timeUnlocked": "0001-01-01T00:00:00Z" },
    "mediaAssets": [ { "type": "Icon", "url": "https://images-eds-ssl.xboxlive.com/image?url=..." } ],
    "platforms": ["XboxOne"], "isSecret": false, "description": "...",
    "rewards": [ { "value": "30", "type": "Gamerscore", "valueType": "Int" } ],
    "deeplink": "ms-xbl-4430a9f8://...", "isRevoked": false }, ... ] }
```
> IDs notables (skulls/terminals/audio logs) extraits du client (cf. §8) → débloquages campagne.

---

## 8. Enums & constantes (extraits du client `chunk 2303`)

**Campagnes** : `h1`=Halo: Combat Evolved, `h2`=Halo 2, `h3`=Halo 3, `h4`=Halo 4,
`odst`=Halo 3: ODST, `reach`=Halo: Reach.

**Difficultés** (ordre croissant) : `Easy`=1, `Normal`=2, `Heroic`=3, `Legendary`=4, `Laso`=5 (`None`=0).
Clés : `easy|normal|heroic|legendary|laso`.

**Skill rank** : entier 1..50 (sprite sheet positions définies côté client). `platform` défaut `Xbox`.
`skill-ranks` exige `hoppers=` (ids de playlists, source `/hmcc/playlists`, actuellement 500) → 400 si vide.

**Achievement IDs (débloquages campagne)** :
- Cross-game : audioLogs `842`, skulls `679`, terminals `148`
- HaloCE : skulls `209`, terminals `195`
- Halo2 : skulls `337`, terminals `321`
- Halo3 : skulls `743`, terminals `748`
- Halo4 : terminals `544`
- ODST : audioLogs `842`

**Career rank** : `threshold` (XP cumulée), `level`, `prestige` (0..10). Plafond `xp >= 1282608999`.

---

## 9. Endpoints connexes transverses (compte waypoint, hors `/hmcc/`)

Utilisés par le shell du site, partagés tous titres (`profile.svc` / `comms.svc`).
`{u}` ici = `me` (compte courant) ou un identifiant utilisateur.

| Service | Endpoints (GET sauf mention) |
|---|---|
| `profile.svc` | `/users/me`, `/users/{u}/profile`, `/users/{u}/settings`, `/users/{u}/rewards`, `/users/{u}/service-awards`, `/users/{u}/service-awards/featured-awards`, `/users/{u}/people?startIndex=&maxItems=` (amis), `/users/{u}/halo-insider`, `/users/{u}/twitch/drops`, `/users/{u}/settings/{steam,twitch,email-verification}` |
| `comms.svc` | `/users/me`, `/users/{u}/notifications?offset=&limit=`, `/users/{u}/notifications/count`, `/users/{u}/read-notifications` |
| `wpcontent.svc` | `/banners?lang=`, `/content-ratings/esrb/{rating}?lang=`, `/purchase-content/game-purchase-modal?lang=` |

> Contraste H5/Infinite (autres chunks, pour mémoire) :
> H5 `GET /h5/{mode}/matches/{id}?view=full`, `/h5/matches?matchIds=...` ;
> Infinite `GET /{t}/matches/{id}/stats`, `/{t}/players/xuid(...)/matches?...`.
> Ces deux titres exposent un **détail de match + roster** ; **MCC non**.

---

## 10. Reproduire / vérifier

```bash
# Sonde headless (réutilise le pool de tokens). owner = propriétaire du token ; subject = joueur lu.
go run ./apps/go-api/cmd/probe-mcc <owner_gamertag> [subject_gamertag]
```
Branche : `explore/mcc-endpoints-probe`. Source de vérité du client officiel :
`www.halowaypoint.com/_next/static/chunks/2303-*.js` (12 fonctions `getHaloMcc*`).

---

## 11. Implications pour une activation LevelUp (titre `halo_mcc`)

Faisable sur le pattern Halo 5 (cf. `docs/adr/0025` + `internal/games/halo_5/`) :
- `config/titles/halo_mcc/{title,auth,constants}.toml` : host `mccapi`, prefix `hmcc`,
  auth = descriptor Infinite réutilisé (pas de clearance), convention `users/gt(GT)`.
- Adapter `internal/games/halo_mcc/` (client + Load*).
- **Capabilities réalistes** : `career`/service-record, `matches` (≤100 récents, agrégé),
  `campaign`, `achievements`, `asset.images` (maps/ranks/seasons/emblems), perso=inventory (owner-only).
- **Hors capacité** (data inexistante côté API) : détail per-match, roster/adversaires,
  per-arme, médailles détaillées par match, MMR/CSR fin (seul `skillRank` 1..50 + `xp`).
- **Pièges** : `playlists` 500, `skill-ranks` exige hoppers, pagination matchs ~100,
  filtre `title` format inconnu, `inventory` 403 cross-joueur.
