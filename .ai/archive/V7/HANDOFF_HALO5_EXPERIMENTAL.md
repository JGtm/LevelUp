# HANDOFF — Halo 5: Guardians comme 2e titre EXPERIMENTAL

> **Créé** : 2026-06-18 (session multi-titre péripherie).
> **CORRIGÉ EN PROFONDEUR** : 2026-06-19 — comparaison au repo `cryptum-halodotapi` (la prémisse « données limitées / pas d'events » de la v1 était FAUSSE, cf. §0-bis).
> **Statut** : faisabilité PROUVÉE (surface API riche), pas commencé.
> **Branche** : `feat/multititre-peripherie`. **User de test** : **JGtm** (l'utilisateur).
> Mémoire liée : `project_halo5_experimental_direction`.

> **REPRISE 2026-06-19** : Phase 1a (adapter read-only) LIVRÉE + reviewée + active-ready. Avant activation, 2 pré-requis user (placement par titre = fait ; sélection par titre = à faire) + l'activation elle-même sont cadrés dans **`.ai/HANDOFF_MULTITITRE_ACTIVATION.md`** — c'est le handoff de reprise courant.

## 0. TL;DR pour reprendre

Halo 5 = premier VRAI 2e titre, en mode **experimental**. **Toute l'infra d'accueil est déjà prête** (registre config, switcher câblé, provisioning DB au boot, gating capability, drift-detector de contrat). Ce qui reste = le **title-specific** : hosts + client + adapter de données + mappings + **feature-matrix** (déterminer quoi est affichable). **L'auth est quasi-gratuite** (réutilisation du SpartanToken v4 du pool Infinite, cf. §1).

## 0-bis. CORRECTION MAJEURE 2026-06-19 — Halo 5 est un titre RICHE, pas dégradé

La v1 (2026-06-18) affirmait que Halo 5 n'a « RIEN au niveau des highlight events ni des films » et qu'il faut tout dégrader. **C'est faux.** La lecture du repo `cryptum-halodotapi` (autorité SpartanStats), recoupée par la doc communautaire `glitch100/Halo-API` et le portail officiel 343, montre que Halo 5 expose :

| Endpoint (autorité SpartanStats / HaloPlayer) | Donnée | Granularité |
|---|---|---|
| `/h5[platform]/players/{player}/matches` | historique de matchs | liste paginée, profondeur historique |
| `/h5[platform]/{mode}/matches/{id}` | carnage report (scoreboard complet par joueur, dont CSR Arena) | **par match** |
| `/h5/matches/{id}/events` (⚠️ **SANS segment de mode** — le `/h5/{mode}/matches/{id}/events` renvoie 404) | **events / timeline typés** (Death, Medal, PlayerSpawn, WeaponPickup/Drop, RoundStart/End) | **par match, horodatés** — CONFIRMÉ LIVE 2026-06-19 (§0-quater) |
| `/h5/servicerecords/{mode}?players=` | service record agrégé (dont Spartan Rank) | agrégé par mode |
| `/h5/players/{player}/commendations` | **citations natives** | agrégé (lifetime) |
| `/h5/players/{player}/credits` | crédits / REQ | agrégé |
| HaloPlayer `/h5/profiles/{p}/{spartan,appearance,emblem}` | bannière, emblème, service tag, armure | par joueur |
| UGC | variants, forge groups, **films** | — |

**Conséquences :**
- La divergence Halo 5 ⟷ Infinite est de **vocabulaire / échelle / natif-vs-calculé**, **pas** un manque de données. (Pas de season pass, pas de défis, Spartan Rank ≠ rang carrière HINF, CSR ≠ HINF, citations natives au lieu du moteur local, REQ packs au lieu du battlepass.)
- **Le kill-feed Halo 5 est STRUCTURÉ** : les death events JSON portent tueur / victime / arme. **Tout le travail film-decoder / dead-state d'Infinite est INUTILE pour Halo 5** — l'API le sert propre.
- **Citations natives** → la **décision B** (découpler la capability `citations.engine` en surface-vs-mécanisme) devient un **vrai bloquant Phase 1** : sans elle on ne PEUT PAS exposer les citations Halo 5 (le moteur local ne tourne pas, mais la surface existe et est plus complète qu'Infinite).

**Ce que ça NE périme PAS** : la machinerie de **dégradation fine par capability** (statuts `degraded`/`not_exposed`, cascade feature-matrix) reste pleinement pertinente. Un **futur titre (ex. Halo 7)** pourra réellement ne pas exposer d'events-per-film ni de highlight events — et c'est exactement ce que le système gère. Le travail de dégradation n'est pas perdu ; il ne s'applique simplement pas à Halo 5.

**LE caveat à lever en premier (Phase 1)** : l'API **officielle** `haloapi.com` est dégradée (sonde 2026-06-18 : Snip3down tout à zéro). cryptum **définit** la surface riche ci-dessus via les endpoints **internes** (SpartanToken), mais **est-ce que le backend Halo 5 de 343 sert encore tout ça en 2026** (surtout les events d'un jeu de 2015) ? Indéterminé tant qu'on n'a pas **sondé le endpoint interne live avec un vrai SpartanToken**. → **Phase 1, étape 0 = sonde live** avant de figer la matrice.

## 0-ter. SONDE LIVE — CONFIRMÉ 2026-06-19 (le caveat est LEVÉ)

`cmd/probe-h5` (réutilise `auth.RefreshHaloTokensViaStoreFirst` → SpartanToken v4 du pool Infinite ; hosts/headers calqués cryptum). Sondé JGtm sur 7 endpoints internes h5. **Résultat : tout HTTP 200, données réelles.** Les deux inconnues critiques sont tranchées :

1. **343 sert ENCORE Halo 5 en 2026** — matches (dont un de 2023-04-05), servicerecords arena+warzone, commendations, credits, profiles : tous 200 avec du contenu réel. Pas un jeu mort côté backend.
2. **Le SpartanToken v4 (pool Infinite) est ACCEPTÉ par Halo 5** — `spartan_preamble="v4="`, 200 partout. L'hypothèse §1 est **prouvée**. (cryptum valide `v[2-3]=` côté client, mais le **service** accepte v4.) L'auth Halo 5 ≈ réutilisation pure, **zéro audience séparée**.

**Recette de requête Halo 5 confirmée** (≠ Infinite) :
- Host : autorités cryptum **réelles** (gravées dans `config/titles/halo_5/constants.toml`) — `spartanstats.svc.halowaypoint.com` (matches/servicerecords/commendations/credits), `haloplayer.svc.halowaypoint.com` (profiles/appearance/spartan render), `content-hacs` (CMS), `ugc` (films/variants), `packs` (REQ).
- Header auth : `X-343-Authorization-Spartan: <v4>` ; **`User-Agent: cpprestsdk/2.4.0`** ; **`?auth=st`** en query sur les hosts `*.svc.halowaypoint.com` ; **PAS de `343-clearance`** (les réponses portent `ClearanceAware:false`).
- **Identité = GAMERTAG brut** (`/players/{gamertag}/…`), PAS `xuid(N)`. Confirmé : `Players[].Player.Xuid` = **null**, seul `Gamertag` est rempli. **Divergence structurante vs Infinite** (xuid-keyé) → l'adapter h5 doit indexer par gamertag.

**Shapes réels capturés** (pour designer l'adapter) :
- **MATCHES** `/h5/players/{gt}/matches` : `{Start,Count,Results:[{Id.MatchId, Id.GameMode(1=arena), HopperId(playlist), MapId, GameBaseVariantId, MatchDuration(ISO8601 PT..), MatchCompletedDate.ISO8601Date, Teams:[{Id,Score,Rank}], Players:[{Player.Gamertag, TeamId, Rank, Result(2=win?/3=loss), TotalKills/Deaths/Assists, Pre/PostMatchRatings(null en liste, CSR dans le carnage)}], IsTeamGame, Links.StatsMatchDetails(→carnage h5/{mode}/matches/{id}), Links.UgcFilmManifest(→film)}]}`.
- **SERVICE_RECORDS** `/h5/servicerecords/{mode}?players={gt}` : CSR **natif** (`HighestCsrAttained:{Tier,DesignationId,Csr,PercentToNextTier}`), `ArenaPlaylistStats[]`/`WarzoneStat.ScenarioStats[]` avec kills/HS/deaths/assists/games, `MedalAwards:[{MedalId,Count}]`, `WeaponWithMostKills.WeaponId.StockId`, `TotalTimePlayed`.
- **COMMENDATIONS** `/h5/players/{gt}/commendations` : `{ProgressiveCommendations:[{Id,Progress,CompletedLevels:[{Id,CompletedDateUtc.ISO8601Date}]}]}` — **citations natives** datées (décision B : exposables direct, le moteur local ne tourne pas mais la surface est plus riche).
- **CREDITS** `{CurrentBalance}` ; **APPEARANCE** identité complète (ServiceTag, Company, emblème, armure, weapon skins) ; **SPARTAN** = PNG render.

**Conclusion** : la **matrice optimiste §2 est CONFIRMÉE par la donnée réelle** (history/detail/scoreboard/skill/citations supported ; warzone=PvE-like ; films présents mais inutiles car kill-feed structuré dans carnage/events). On peut passer aux étapes 1-3 de la Phase 1 (auth reuse → client → adapter) sans risque de construire sur du vide. `cmd/probe-h5` est conservé comme outil de re-sonde.

## 0-quater. SONDE EVENTS + CARNAGE — CONFIRMÉ 2026-06-19 (la timeline est SERVIE, riche)

Sonde `cmd/probe-h5` étendue (carnage detail + variantes events + film) sur un vrai match arena de JGtm. **Résultat décisif** :

- **`/h5/matches/{id}/events` → HTTP 200, 220 Ko** de `GameEvents` typés. **GOTCHA** : le chemin avec segment de mode (`/h5/arena/matches/{id}/events`) renvoie **404** ; le bon chemin est **SANS mode**. (Mon premier rapport « events 404 » était une erreur de chemin.)
- **Types d'events (1 match arena)** : `Death`×107, `Medal`×126, `WeaponPickup`×269, `WeaponDrop`×305, `PlayerSpawn`×113, `RoundStart`/`RoundEnd`×1. Tous portent `TimeSinceStart` (ISO8601 PT..).
- **Shape de l'event `Death` (= kill-feed + arme-par-kill NATIFS)** : `{IsHeadshot, IsMelee, IsGroundPound, IsShoulderBash, IsWeapon, Killer:{Gamertag}, KillerAgent, KillerWeaponStockId, KillerWeaponAttachmentIds[], KillerWorldLocation:{x,y,z}, Victim:{Gamertag}, VictimWorldLocation:{x,y,z}, TimeSinceStart, EventName:"Death"}`. → **tueur·victime·ARME·type·POSITION·instant**, en clair.
- **Event `Medal`** : `{MedalId, Player:{Gamertag}, TimeSinceStart}` — timeline de médailles.
- **Carnage detail `/h5/{mode}/matches/{id}` → 200, 37 Ko** : scoreboard par-joueur étendu + **matrice tueur↔victime agrégée** (`KilledOpponentDetails`/`KilledByOpponentDetails`) + `XpInfo` (Spartan Rank pré/post) + `CreditsEarned` + `ProgressiveCommendationDeltas` (par match) + `WeaponWithMostKills`. Les tableaux fins `EnemyKills`/`WeaponStats`/`Impulses` sont VIDES dans le carnage (le détail per-kill est dans `/events`, pas dans le carnage).
- **Film manifest UGC** (`ugc.svc/h5/films/{id}?view=film-manifest`) → **403** (clearance/auth films ≠) — **non nécessaire** : `/events` donne déjà le per-kill propre.

**Conséquence stratégique (validée par la donnée)** : Halo 5 expose NATIVEMENT, en JSON propre, exactement ce que le décodage film d'Infinite cherche à reconstruire (arme-par-kill, kill-feed) — **plus** les positions monde (qu'Infinite n'a pas). Direction adoptée (user 2026-06-19) : **canoniser le modèle d'events à partir de la shape Halo 5** (la plus propre/complète) et **faire converger Infinite dessus** (inverse de la décision citations). Caveat honnête : confirmé sur arena ; warzone non testé (aucun match non-arena dans les 25 derniers de JGtm) — à re-sonder le jour où on a un match warzone.

## 1. Source de données + AUTH (TRANCHÉ + CORRIGÉ)

**Source = endpoints internes façon `cryptum-halodotapi`, PAS l'API officielle** (officielle = dégradée). cryptum = doc des URLs/hosts/shapes par autorité (SpartanStats, HaloPlayer, ContentHacs, UGC, Packs, BanProcessor).

**AUTH — CORRECTION 2026-06-19** : la v1 disait « SpartanToken Halo 5 ≠ Halo Infinite (audience) — c'est le point d'effort auth ». **Faux/sur-estimé.** D'après l'user : **le SpartanToken v4 qu'on utilise déjà pour Infinite fonctionne pour Halo 5** ; cryptum référence des tokens v2/v3 (plus anciens). Le SpartanToken est title-agnostique au niveau des services Xbox/343 ; la différence Halo 5 est dans les **hosts d'endpoints**, pas dans le token. → **`auth.toml` Halo 5 ≈ mirroir d'Infinite (mêmes audiences)** ; le vrai travail title-specific est `constants.toml [endpoints]` (hosts h5) + le client + l'adapter. **Auth ≈ réutilisation — CONFIRMÉ par la sonde live 2026-06-19 (v4 accepté, 200 partout, cf. §0-ter).**

**Clé API officielle** (subscription key haloapi.com) : fournie par l'user 2026-06-18, **en clair dans le chat → À RÉGÉNÉRER**. Inutile dans le chemin retenu (interne) ; si jamais utilisée en fallback : env/`.env.local` gitignoré, JAMAIS versionnée. Repo cryptum : https://github.com/Alexis-Bize/cryptum-halodotapi

## 2. Feature-matrix par titre — fine, dirigée par les DONNÉES réelles du titre

L'enjeu n'est pas « Halo 5 est pauvre » (faux), mais « chaque titre expose un sous-ensemble + un vocabulaire propres ». Le système de **capabilities fines** (cascade → feature-matrix) gère ça. Briques :
- **Capability gating front** : `apps/web/src/lib/capabilities/` (`useCapability`/`FeatureGate`/`RouteCapabilityGate`). NO-OP pour halo_infinite (l'app sert tout en legacy) ; gating réel pour un titre servi via l'adapter.
- **Capabilities fines backend** : `internal/games/adapter.go` (clés `CapabilityKey`) + `config/titles/{slug}/mappings/capabilities.toml` + cascade `internal/games/feature.go`. Statuts `supported`/`degraded`/`not_exposed`.
- **Sémantique à NE PAS confondre** (source du malentendu corrigé le 2026-06-19) : les statuts décrivent ce que **l'adapter canonique** expose, **pas** le plafond de données ni les features de l'app. Pour halo_infinite, plusieurs clés sont `not_exposed`/`degraded` parce que la migration de l'adapter (Phase 1.7a/B/C) ne les a pas encore câblées — **l'app sert ces surfaces en LEGACY**, ce n'est PAS une absence de données. Pour un titre servi **uniquement** par l'adapter (Halo 5), ces statuts **== son plafond réel** : à régler d'après ce que SA source fournit.

**Clés fines ajoutées le 2026-06-19** (décision A — vocabulaire BP/défis explicite) : `battlepass.progression`, `challenges.surface`. Déclarées `supported` pour halo_infinite/synthetic, `not_exposed` pour Halo 5.

### Matrice de capabilities Halo 5 (intention `coming_soon`, OPTIMISTE, sous réserve sonde live Phase 1)

| Clé fine | Halo 5 | Raison (donnée cryptum) |
|---|---|---|
| `match.history` | **supported** | `/players/{p}/matches` |
| `match.detail.core` | **supported** | carnage report par match |
| `match.scoreboard.extra` | **supported** | carnage report riche (par-joueur étendu) |
| `match.skill.snapshot` | **supported** | CSR pré/post natif dans le carnage Arena |
| `career.progression` | **supported** | Spartan Rank via service record |
| `analytics.timeseries` | **degraded** | buildable depuis l'historique (kills/KDA dans le temps) |
| `engagement.score` | **degraded** | events présents, mais coefficients à RECALIBRER pour Halo 5 |
| `citations.engine` | **bloqué par B** | commendations NATIVES → exposable une fois la clé découplée surface/mécanisme |
| `pve.firefight_stats` | **degraded / à sonder** | Warzone Firefight existe, modèle ≠ Firefight HINF |
| `battlepass.progression` | **not_exposed** | pas de season pass (REQ packs à la place) |
| `challenges.surface` | **not_exposed** | pas de défis |

**Coarse caps** (`title.toml capabilities=[]`) : `matchmaking`, `ranked`, `career`, `asset.images`, `achievements` (Xbox via xbox_title_id), `lusr` (degraded, cf. §LUSR), `engagement`. À sonder/Phase 2 : `forge`/`media` (UGC + films existent mais modèle HINF-shaped), `firefight` (Warzone). Exclus : `world.leaderboard` (scrape Waypoint HINF-spécifique).

**LUSR Halo 5 = `degraded`** (PAS `not_exposed`) : le cœur du rating (`skill_v2_service.go` `UpdateTwoTeam`) ne consomme que outcomes + rosters + priors Mu/Sigma — zéro event requis. La pondération temps-joué dépend de `real_start_time` (calculé via `FirstJoinedTime` de l'API de participation, pas des events) ; absente → poids uniformes `wᵢ=1`, rating valide. Le chemin de dégradation existe déjà (zéro code). NB : Halo 5 a même un CSR natif par match (carnage Arena) — on peut afficher CSR direct ET calculer LUSR.

## 3. Infra d'accueil PRÊTE (ne pas reconstruire)

- **Registre piloté par config** : déposer `config/titles/halo_5/` (`title.toml` + `mappings/{capabilities,fields,assets,outcomes}.toml` + `constants.toml` + `auth.toml`) → découvert au boot par `title.LoadTitlesIntoRegistry` (`internal/domain/title/config_loader.go`). Modèle = `config/titles/synthetic_title_b/`.
- **Switcher UI** : `apps/web/src/components/shell/TitleSwitcher.tsx`. `coming_soon` → « Bientôt disponible » ; `active` → sélectionnable.
- **Provisioning DB au boot** : `cmd/server/main.go` `provisionAdditionalActiveTitles` (itère `reg.Active()`). Ne provisionne PAS les `coming_soon`.
- **Watcher** : title-threadé (le daemon reçoit `titleReg`, chemins via `PathResolver(slug)`, `PlayerWatcher.SetTitleSlug`). MAIS le boot n'instancie **qu'UN daemon** sur `title.DefaultSlug` (`cmd/server/main.go:1879`). Pour qu'un 2e titre ACTIF soit réellement surveillé → faire boucler le boot sur `reg.Active()` (un daemon par titre actif). Inutile en Phase 0 (`coming_soon` non surveillé). L'xbox_title_id Halo 5 sert au watcher (présence) ET aux achievements Xbox.
- **Pattern asset « kinds » universel** : `internal/assetnames/resolver.go` (local → live discovery → persist), paramétré par titre via `EndpointResolver` (lit `constants.toml [endpoints]`). Déjà dans la branche (commit `3c8a005c4`). Halo 5 = `asset.images` supported, résolu à la demande, **zéro bundle local** (on n'a aucun asset Halo 5 à héberger).
- **Adapters** (le vrai travail title-specific) : `games.TitleDataAdapter` (Load*) + `TitleSemanticAdapter` dans `internal/games/halo_5/`. Types de retour = **canonical** (`internal/games/canonical/`).
- **Validateur boot** : `internal/games/mappings/validate.go`. N'`os.Exit` que sur titres ACTIFS.

## 4. Plan staged

### Phase 0 — Skeleton — FAIT 2026-06-19 (commit `fc989ca5c`, status `coming_soon`, skeleton_test vert)
`config/titles/halo_5/` complet, **status `coming_soon`**, **xbox_title_id `219630713`**, matrice de capabilities §2 (intention optimiste), mappings calqués sur synthetic_title_b mais métadonnées Halo 5 réelles + vocabulaire propre (Spartan Rank, CSR Halo 5, modes/playlists Halo 5). Aucun adapter requis (pas servi).
- **Pré-requis livré** : extension vocabulaire capabilities BP/défis (`battlepass.progression` + `challenges.surface`) + sémantique capabilities.toml clarifiée.
- **Oracle** : `internal/games/halo_5/skeleton_test.go` (registre découvre coming_soon + capabilities + endpoints distincts) ; `go test ./internal/games/mappings/`.

### Phase 1 — Experimental read-only (≈1-2 sessions, risque moyen)
0. **SONDE LIVE D'ABORD** : avec le SpartanToken v4 existant (pool Infinite), taper les endpoints internes h5 (matches, carnage, **events**, servicerecords, commendations) pour un xuid réel (JGtm) → **confirmer que 343 sert encore Halo 5 en 2026** et capturer les shapes réels. C'est ce qui valide/ajuste la matrice §2.
1. **Auth** : réutiliser le SpartanToken v4 (`auth.toml` ≈ mirroir Infinite). PAS d'audience séparée a priori — confirmer à la sonde.
2. **Client interne** Halo 5 : `internal/games/halo_5/client.go` (hosts via `constants.toml [endpoints]`, réf cryptum pour URLs h5).
3. **Adapter data** : 1 surface d'abord (recommandé **historique + carnage** = scoreboard réel). Mapper JSON Halo 5 → canonical `MatchSummary`/`MatchDetail`/`PlayerStats`. Le kill-feed est STRUCTURÉ (events JSON) → pas de décodage film.
4. **Décision B** (citations) : découpler `citations.engine` (mécanisme) en clé de surface → exposer les commendations natives Halo 5.
5. **Feature-matrix** : ajuster d'après la sonde live (ce qui répond réellement).
6. **Statut → `active`** (servi + provisionné) + faire boucler le watcher sur `reg.Active()`.
- **Oracle** : page front avec données réelles JGtm, surfaces non-dispo masquées (pas vides).

### Phase 2+ — Complet (multi-sessions)
Sync/ingestion DB, recalibration coefficients engagement Halo 5, REQ packs (surface Halo-5-spécifique nouvelle), Warzone Firefight, UGC/films, nettoyage vocab cosmétique.

## 5. Pièges connus / notes
- **Sonder live AVANT de croire la matrice** : cryptum documente la surface ; seul un appel réel dit ce que 343 sert encore (2015 → 2026).
- **Auth** : SpartanToken v4 (pool Infinite) attendu OK pour Halo 5 ; cryptum v2/v3 = obsolète. Ne PAS sur-investir une « audience séparée » sans preuve de la sonde.
- **Citations** : décision B est un BLOQUANT (natives, plus complètes qu'Infinite) — pas cosmétique.
- **Ne JAMAIS comparer `slug == "halo_5"`** (archlint `no_slug_comparison`) — tout par capability.
- **capabilities.toml** : statut = surface ADAPTER, pas plafond de données (cf. §2). Pour Halo 5 (pur-adapter) les deux coïncident ; pour halo_infinite non (legacy).
- **Spartan ID** : bannière / emblème / service-tag **portables** (chemin metadata title-agnostic, persistance générique `career_progression`) ; seul l'**adornment** (overlay nameplate, `metadata.career_ranks.adornment_icon_path`) potentiellement absent pour Halo 5 — le front dégrade DÉJÀ sur null (`HomeSpartanIdentityBanner.tsx:80` rend l'img sous condition). Pas un bloquant.
- **Vocab cosmétique** (admin « API Halo », Lab « Waypoint », `HINF-CSR`) : Halo 5 l'exerce (CSR ≠ HINF) → à neutraliser en Phase 1/2.
- Réfs : `.ai/PLAN_MULTITITRE_INDEX.md`, `.ai/PLAN_TITLE_AGNOSTIC_TRACKER.md`, mémoire `project_halo5_experimental_direction`.
- Sources cryptum : https://github.com/Alexis-Bize/cryptum-halodotapi · https://github.com/glitch100/Halo-API/blob/master/docs/Halo5.md · portail 343 developer.haloapi.com.
