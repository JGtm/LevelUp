# Plan — Assets Halo 5 (médailles, cartes, rangs CSR, rangs XP/SR, armes)

> Roadmap issue du workflow de scoping `h5-assets-scope` (6 agents recherche + synthèse,
> halopedia.org + wiki.halo.fr + den.dev + lecture codebase) du 2026-06-21. Branche
> `feat/multititre-peripherie`. Prérequis posé : **ROOT FIX isolation metadata h5** (commit
> `d19f340a0`) — la metadata.duckdb h5 a ses propres tables référentielles VIDES, isolées
> d'Infinite, prêtes à peupler.

## Constat clé (ce que le scoping a révélé)

Le « minimum prod » (médailles / cartes / rang CSR / rang XP / armes) se décompose en
**ce qui est déjà fait**, **ce qui est statique/débloqué**, et **ce qui est bloqué sur du live CMS**.

| Catégorie | Labels (texte) | Images | Bloquant |
|---|---|---|---|
| **CSR (rang classé)** | ✅ DÉJÀ FAIT (`mapping_servicerecord.go::h5Designations`, FR composé "Diamant 5") | ❌ pas de PNG sources + câblage badge hardcodé HINF | images sources + refactor câblage |
| **SR (rang XP 1-152)** | référentiel statique prêt (SR{n}) + **XP réels 152 niveaux** (`.ai/refs/h5_spartan_rank_xp.csv`, vérifié) | aucune image **by design** (le SR s'affiche en chiffre) | **le niveau SR du joueur n'est PAS dans la donnée récupérée** → fetch live requis |
| **Médailles** | noms FR connus (WikiHalo) | spritesheet CMS | **medal_name_id numériques = live CMS** |
| **Armes** | ~31 noms EN/FR prêts | bundle ou CDN | **weapon_id (StockId) = live (ingestion matchs)** |
| **Cartes** | ~15 noms Arena (name_fr=name_en) | CDN UGC | **map_asset_id GUID + image_url = live UGC** |

**Conclusion** : CSR texte = fait. Tout le reste exige soit une **sonde CMS live** (IDs/shapes/URLs),
soit des **images sources**, soit un **refactor de câblage badge**. Aucun gros « seed statique »
n'a de consommateur immédiat (cf. SR ci-dessous) → ne PAS créer de schéma mort.

## Pré-requis transverse : câblage badge title-aware (BLOQUANT visuel)

Le builder du badge CSR du Home (`internal/platform/duckdb/home_repo_skill_peak.go:436`) et le header
de match (`internal/service/match_view_builders_header.go:230-233`) instancient
`halo_infinite.NewAssetURLAdapter()` **EN DUR**. Tant que ce n'est pas résolu via le
`games.Resolver` (adapter par slug du titre courant, fallback HINF), enregistrer un adapter
h5 ne fait **rien** sur le badge. C'est le vrai travail de câblage, à faire avant tout affichage
d'image de rang h5. Risque : touche le chemin HINF (à tester sans régression).

## Track B — référentiels STATIQUES (zéro réseau)

- **B-CSR labels** : ✅ FAIT (rien à faire côté texte).
- **B-CSR designations TOML** (optionnel) : `config/titles/halo_5/mappings/assets.toml` kind
  `csr_designation` (Bronze..Onyx, designation_id 0..5 ; PAS de Champion = position leaderboard).
  *Faible valeur tant qu'aucun consommateur ne lit `Assets()` pour ce kind — différer.*
- **B-SR référentiel** : tables `career_ranks` + `career_rank_translations` dans la metadata h5
  (`internal/games/halo_5/migrations/metadata.go`, miroir DDL HINF) + seed 152 lignes
  (`SR{n}` EN / `SR {n}` FR) + `xp_required` depuis `.ai/refs/h5_spartan_rank_xp.csv` (colonne
  `xp_to_next`). Le loader `duckdb/halo_ranks_loader.go::LoadRankCatalog` est title-agnostique
  (à brancher : `server_titles_additional.go:82` passe `ranks=nil` → charger depuis la metaDB h5).
  **⚠ Pas de consommateur tant que le niveau SR du joueur n'est pas fetché** (le service record
  arena ne le contient pas). → **NE PAS livrer seul** ; le faire AVEC le fetch SR (Track A).
  Recommandation synthèse : seed direct en INSERT dans `ApplySchema` (évite le seam global
  mono-titre `SetCareerRankTranslationsProvider`, écrasé au boot par HINF).
- **B-armes noms** : `internal/games/halo_5/migrations/weapon_labels.go` (miroir HINF, quirk
  littéral décimal UBIGINT bit63) + CLI `cmd/seed-h5-weapon-labels`. Table de noms livrable,
  mais **`weapon_id` reste vide jusqu'à collecte des StockId** (Track A) → seed effectif différé.

## Résultats sonde A1 (2026-06-21, `cmd/probe-h5` Phase 3, host title-agnostic)

Live JGtm, host résolu via EndpointResolver (`gamecms=content-hacs.svc`, `prefix=h5`) :
- ✅ **SR_MANIFEST : HTTP 200** — `content-hacs.svc/contents/SpartanRankManifest`. Shape =
  wrapper `{Paging, ContentItems[{Id,Type,View:{...,SpartanRankManifest:{SpartanRanks[]}}}]}`
  (le 1er ContentItem est un dummy vide ; le réel suit). **Données déjà en main** (CSV den.dev
  vérifié) → le fetch live n'est pas nécessaire pour le référentiel SR.
- ❌ **Médailles** (`/h5/Progression/file/medals/metadata.json`, `/contents/Medals`,
  `/h5/metadata/medals`) : **403** → mauvais noms/paths (content-hacs renvoie 403 sur contenu
  inconnu, pas 404).
- ❌ **CSR-designations** (3 variantes) + **commendations** (`/contents/CommendationManifest`) :
  **403** → idem, noms de contenu à trouver.
- ❌ **UGC maps** (`ugc.svc/h5/maps`) : **404** → path à corriger.

**Conclusion A1** : le mécanisme `content-hacs/contents/{Name}` fonctionne (SR=200) mais les
NOMS de contenu médailles/CSR/commendations sont à récupérer de la **référence cryptum/HaloDotAPI**
(ou des posts den.dev équivalents au SR manifest). Le path UGC maps n'est pas `/h5/maps`.

## DÉCOUVERTE (réf cryptum Alexis-Bize/cryptum-halodotapi + re-sonde, 2026-06-21)

**cryptum H5 ContentHacs** n'expose que REQ/Emblem/Hopper/WeaponSkin/GameVariantDefinition/
GameBaseVariant/MOTD via `/contents/{type}` (+ `CUSTOM_TYPE` = type arbitraire). **AUCUN endpoint
médaille/CSR/SR-joueur** dans cryptum. `HALO_PLAYER.SPARTAN` (`/h5/profiles/{p}/spartan`) = un
**PNG** (rendu spartan), pas le SR.

**⭐ B-SR DÉBLOQUÉ — le niveau SR du joueur est DÉJÀ dans la carnage** : `CARNAGE_DETAIL`
`PlayerStats[].XpInfo` porte, PAR MATCH et PAR JOUEUR :
`{PrevSpartanRank, SpartanRank, PrevTotalXP, TotalXP, SpartanRankMatchXPScalar, ...}`
(ex. JGtm : SpartanRank=111, TotalXP=3 908 120). La carnage est DÉJÀ fetchée à l'ingestion.
⇒ le SR du joueur (niveau courant = XpInfo du match le plus récent) + son XP cumulé sont
disponibles SANS appel supplémentaire. **Plus aucun blocage live pour le SR.**

**Implémentation B-SR (end-to-end, statique + données déjà fetchées)** :
1. Référentiel : seed `career_ranks`(rank_id, xp_required) + `career_rank_translations`(SR{n}
   EN/FR) dans la migration metadata h5, XP depuis `.ai/refs/h5_spartan_rank_xp.csv`.
2. Ingestion : extraire `XpInfo.SpartanRank` + `TotalXP` du joueur (viewer) depuis la carnage
   (`mapping_carnage`) → stocker (player_match_enrichment ou match_participants).
3. Lecture carrière : exposer le SR courant (TotalXP le plus récent → SpartanRank) dans
   `CareerSnapshot` (CurrentRank=rank_id, CurrentXP=TotalXP, NextRank+XPForNextRank via le
   RankCatalog) + brancher le RankCatalog h5 (`server_titles_additional.go:82`, charger depuis
   la metaDB h5 au lieu de `ranks=nil`).
4. Affichage : « SR 111 » en texte (pas d'image, by design).

**Médailles** : toujours bloqué — le nom de contenu content-hacs médailles H5 reste introuvable
(pas dans cryptum ; den.dev n'a documenté que le SR). Pistes : énumérer les contenus content-hacs,
ou les noms FR/IDs via les `MedalStatCounts` de la carnage (IDs) + manifest à localiser.

## DÉCOUVERTE MAJEURE (2026-06-21) — source des métadonnées = API officielle Halo 5

Sondes content-hacs (SpartanToken) + cryptum confirment : les **référentiels canoniques
(médailles, cartes, armes, csr-designations, spartan-ranks) ne sont PAS sur les endpoints
internes SpartanToken** — c'est pourquoi cryptum n'a aucune métadonnée et que mes sondes content-hacs
ont renvoyé 403. Ils ne vivent que sur l'**API Metadata OFFICIELLE** :

`https://www.haloapi.com/metadata/h5/metadata/{type}` — **VIVANTE en 2026** (401 = clé manquante,
pas 404). Types : `medals`, `maps`, `weapons`, `csr-designations`, `spartan-ranks`,
`game-base-variants`, `game-variants`, `playlists`, `vehicles`, `requisitions`, `commendations`,
`enemies`, `impulses`, `team-colors`, `flexible-stats`, `campaign-missions`, `map-variants`.
Auth : header `Ocp-Apim-Subscription-Key: {clé}` (≠ SpartanToken). API référencée
`developer.haloapi.com api=58ace18c21091812784ce8c5` (API Metadata Halo 5).

**BLOQUANT** : aucune clé d'abonnement dans le projet. Inscription gratuite sur
`developer.haloapi.com` requise (compte 343/Azure APIM) — non automatisable par l'agent.
**Une fois la clé fournie** : un fetcher unique (mirror de `medal_provider.go`, host+auth = clé)
peuple medals/maps/weapons/csr-designations/spartan-ranks dans la metadata h5 → débloque
TOUTES les catégories min-prod restantes d'un coup. C'est le chemin propre + autoritatif.

**Buildable SANS clé (content-hacs, SpartanToken, déjà 200)** : **modes** (`/contents/GameBaseVariant`,
22, nom+icône) + **playlists** (`/contents/Hopper`, 73, nom). Le match porte `GameBaseVariantId`
+ `HopperId` (GUIDs) → résolus en noms via ces contenus. Fetcher + seed `asset_translations`
(asset_type game_variant/playlist) + read.

**Images de rang CSR/SR** : l'API officielle `csr-designations` + `spartan-ranks` portent
probablement les `iconImageUrl` (à confirmer avec la clé) → résout aussi le gap images CSR.

## Track A — fetchers LIVE (après une sonde de confirmation)

**A1 — Sonde `cmd/probe-h5` étendue (PRÉ-REQUIS de tout A)** : réutiliser `halo_5/client.go`
(auth déjà résolue, SpartanToken v4, `X-343-Authorization-Spartan` + UA `cpprestsdk/2.4.0` +
`?auth=st`, PAS de 343-clearance). Confirmer host+path+shape JSON de :
- `csr-designations` (→ `iconImageUrl` des insignes CSR, champ exact à confirmer)
- `SpartanRankManifest` (`content-hacs.svc.halowaypoint.com/contents/SpartanRankManifest`,
  `View.SpartanRankManifest.SpartanRanks[].View.SpartanRank.StartXP` — déjà dumpé dans le CSV ref ;
  confirmer l'auth + le niveau SR **du joueur** via un endpoint profil/account)
- médailles `{gameCMSHost}/h5/Progression/file/medals/metadata.json` (host `gamecms-hacs` vs
  `content-hacs` à trancher ; clés `NameId/SpriteIndex` vs `Id/SpriteLocation`)
- maps UGC `ugc.svc.halowaypoint.com` (préfixe h5 ; pattern GUID→nom,image)

**A2 — Fetcher médailles** : plomberie déjà là (`medal_definitions`/`medal_translations` vides,
`platform/halo/medal_provider.go` title-agnostique, CLI `refresh-metadata medals --title-id
halo_5 --promote`). Icônes = spritesheet (`assets/wire.go::NewSpritesheetFallbackFetcher`).
Seed `medal_translations` fr-FR depuis WikiHalo si le CMS ne renvoie pas le FR.

**A3 — StockId armes** : joindre les StockId réels des events (`KillerWeaponStockId`/
`WeaponStockId`) + servicerecord (`WeaponWithMostKills.WeaponId.StockId`) aux noms statiques B,
puis remplir `weapon_id`.

**A4 — Catalogue maps** : créer `internal/games/halo_5/catalog_adapter.go`
(implémente `games.TitleCatalogAdapter`, miroir HINF) + `RunCatalogUGCDrain(titleSlug="halo_5")`
+ 2e `CatalogRefreshCron`. Sans cet adapter, `maps_catalog` h5 reste vide. `name_fr = name_en`.

**A5 — SR du joueur (débloque B-SR)** : confirmer l'endpoint exposant le niveau SR du compte,
le mapper dans `mapCareerSnapshot` (CurrentRank=rank_id SR, CurrentXP), ce qui donne enfin un
consommateur au référentiel B-SR. Affichage = « SR 152 » en **texte** (pas d'image, by design).

## Images de rang (CSR badges) — pipeline statique-bundle

Pattern HINF confirmé : PNG **committés** sous `apps/go-api/static/ranks/halo_infinite/120px-HINF-CSR_{Tier}{SubTier}.png`
(Bronze1..6 … Onyx, 37 fichiers). Couche pure title-agnostique `internal/assets/static/` déjà
h5-ready (slug paramétré). Pour h5 :
1. Obtenir les **PNG sources** des insignes CSR h5 (extraction jeu / wiki / CDN via `iconImageUrl`
   du JSON `csr-designations`) → committer sous `static/ranks/halo_5/H5-CSR_{Tier}{SubTier}.png`
   (format figé EN, ex. `H5-CSR_Diamond5`, `H5-CSR_Onyx` — distinct du format HINF).
2. Écrire `internal/games/halo_5/adapter_asset_urls.go` (miroir HINF, implémente
   `games.TitleAssetURLAdapter`) + enregistrer via `titleResolver.RegisterAssetURL` dans
   `server.go` (après ligne 345).
3. Faire le **refactor câblage badge title-aware** (cf. pré-requis transverse) — sinon (1)+(2)
   n'affichent rien.

## État final assets (2026-06-21) + reste précis

**LIVRÉ end-to-end** : rang XP (SR, texte) ; **référentiel (Asset Drawer) maps 49 + armes 68**
(noms + images/icônes officielles, title-aware front→back→DB). Données seedées : médailles 215,
maps 49, armes 68, CSR 42 tiers (`cmd/h5-metadata-fetch`).

**Reste (surfaces distinctes / efforts ciblés)** :
- **FR** : l'API Metadata officielle NE localise PAS (`?language=fr`/`lang`/`locale`/`Accept-Language`
  → tous EN). Noms maps (noms propres) + armes restent EN. CSR FR déjà en code (`h5Designations`).
  Médailles FR = seed WikiHalo (scraping ~215 noms, chantier à part).
- **Médailles dans leur surface** : médailles = icône SPRITE (spriteSheetUri + left/top/width/height,
  seedés dans medal_definitions). Affichage = soit un onglet médailles dans l'Asset Drawer (DTO
  enrichi sprite + tab front + CSS background-position), soit dans le détail de match (mais
  `match.detail` h5 = `not_exposed` → nécessite l'expansion de cette surface de lecture).
- **Image de rang CSR (carrière/home)** : `csr_designations(designation_name, tier_id) → icon_url`
  seedé. Câblage = (a) ajouter un champ badge CSR au `CareerSnapshot` canonical ; (b) injecter un
  résolveur csr_designations dans l'adapter h5 (mapCareerSnapshot ; mapping DesignationId→nom EN
  officiel + Tier→tier_id) ; (c) de-hardcoder le builder badge (`home_repo_skill_peak.go:436`
  instancie `halo_infinite.NewAssetURLAdapter()` EN DUR) ; (d) rendu front. Effort multi-couches.
- **PROD** : le fetcher seede la metadata h5 du clone deploy LOCAL. Pour la prod (VPS), lancer
  `LEVELUP_HALOAPI_KEY=<clé> cmd/h5-metadata-fetch` sur le serveur (la clé en env, jamais committée),
  après provisioning. À intégrer comme étape de déploiement (cf. RUNBOOK).

## Décisions produit en attente

- **D1** CSR designations : TOML `assets.toml` (recommandé) vs table metadata. Image via static-bundle.
- **D2** SR seam : seed statique direct dans la migration (recommandé) vs refactor provider par titre.
- **D3** Nom de fichier badge CSR h5 : figer `H5-CSR_{Tier}{SubTier}` EN.
- **D4** SR = texte « SR {n} » (recommandé, pas d'image by design).
- **D5** Champion ≠ designation (position leaderboard) — couche d'affichage au-dessus d'Onyx, pas un référentiel.
- **D6** Périmètre maps (Arena seules vs + Forge) — différable à A4.
- **D7** static-bundle (committer PNG) pour CSR/SR ; cache-aside live pour maps.

## Ordre recommandé d'exécution

1. **A1 sonde live** (débloque tout : shapes, IDs, URLs, auth SR/manifest). ← prochaine action.
2. **A2 médailles** (plomberie prête) + **A5 SR joueur** → qui débloque **B-SR** (seed + wiring).
3. **Refactor câblage badge** + adapter URL h5 + **PNG sources CSR** → badge CSR visible.
4. **A3 armes StockId** → remplit `weapon_id` du seed B-armes.
5. **A4 maps** (effort élevé, adapter catalogue + drain).

## Références

- Données SR XP vérifiées : `.ai/refs/h5_spartan_rank_xp.csv` (152 niveaux).
- Manifest SR : `https://content-hacs.svc.halowaypoint.com/contents/SpartanRankManifest`
  (`View.SpartanRankManifest.SpartanRanks[].View.SpartanRank.StartXP`).
- Pattern assets HINF à mirroir : `internal/assets/static/`, `internal/games/halo_infinite/adapter_asset_urls.go`,
  `internal/games/halo_infinite/migrations/weapon_labels.go`, `internal/platform/halo/medal_provider.go`.
