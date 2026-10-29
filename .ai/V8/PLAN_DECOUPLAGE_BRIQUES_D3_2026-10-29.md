# Plan D3 — Découplage de la brique « API Halo » : protocole et exploitation — 2026-10-29

Source : `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (axe D3 [P1], garde-fou G, décision 4 de
l'utilisateur du 2026-10-27 : noms `internal/haloapi/xblauth`, `internal/haloapi/infinite`,
`internal/sync/filmfetch`, adaptateur sous `games/halo_infinite/`, `platform/auth` garde l'exploitation
des jetons ; addendum du 2026-10-28 point 4 ; addenda « D1 exécuté » et « D2 exécuté »). Exécution sous
le contrat du skill `plan-execution` : ordre strict, un lot à la fois, gate passé avant le lot suivant,
chaque item statué `[x]` / `[~]` / `[!]`, zéro fix hors périmètre (section « Découvertes »).

## Objectif et critère de succès

Le code qui PARLE aux services Halo (chaîne d'authentification Xbox Live → XSTS → Spartan, transport
HTTP, adresses, un fichier par service, DTO JSON bruts, octets des films) vit dans deux paquets sans
état ni politique, `internal/haloapi/xblauth` et `internal/haloapi/infinite`, qui n'importent RIEN de
LevelUp hors d'eux-mêmes (ni `domain`, ni `ctxkeys`, ni `games`, ni `observability`, ni `netguard`, ni
`platform/auth`) : ce que la brique a besoin de savoir (client HTTP, adresses, jetons, slug de titre,
langue) lui est PASSÉ. Tout ce qui décide (qui appelle, quand, avec le jeton de qui, où ranger, ce que
cela veut dire pour LevelUp) reste dans LevelUp : `platform/auth` (stockage, mise en commun, boucles),
`sync/filmfetch` (téléchargement → finalisation → cache local), `games/halo_infinite/haloadapter`
(DTO → domaine), les appelants.

Succès = (1) `go list -deps ./internal/haloapi/...` ne montre aucun paquet `levelup/go-api/internal/`
hors `haloapi/` ; (2) `archlint` tient la liste fermée des imports de `haloapi` (table des briques
INTERNES, allowlist vide à la clôture) et `platform/halo` + `sync/haloclient` n'existent plus ; (3)
les ratchets d'authentification sont INCHANGÉS et verts (`platform/auth/sentinel_test.go` allowlists
VIDES, `sync/no_legacy_source_used_test.go`, `no_pool_hasplayer_gate_test`, `no_request_scoped_halo_provider_test`,
`no_halowaypoint_literal_test`, `netguard` `TestOutboundCallsAreNetguarded`) ; (4) mêmes tests, mêmes
verdicts (modulo paquet) ; les fixtures HTTP enregistrées (`haloclient/testdata/halo/*.json`, tests
`httptest`) rendent les mêmes valeurs de domaine ; (5) témoin LIVE en lecture seule, avant et après :
pour le compte de l'utilisateur, un rafraîchissement de jetons par le chemin canonique, puis l'historique
récent (1 page), les stats d'un match, le skill d'un match, le manifeste d'un film, un actif du
catalogue, le calendrier de saison : JSON bruts identiques (hors horodatages et jetons) et valeurs de
domaine identiques ; (6) CI verte, revue adversariale à DEUX relecteurs sur l'authentification (lot à
risque, skill `adversarial-review` §1).

Effort estimé : 4 à 6 jours d'agent (l'audit disait 1 à 2 semaines humaines). Réécriture PARTIELLE :
les DTO bruts n'existent pas aujourd'hui (les providers rendent directement des types `domain.*`), il
faut les écrire et écrire les conversions ; le reste est déplacement et injection.

## Branches

- Lots 0 à 6 : `refactor/decouplage-d3-api-halo` depuis `release/v8.1.0` (après fusion de la PR #113,
  D2), worktree dédié `../LevelUp-wt-decouplage-d3`, un commit par lot, fusion squash par PR dans la
  release. Jamais `main`.
- Caches : `GOCACHE=%LOCALAPPDATA%\go-build-d3`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-d3`.
- Autorisation : message de l'utilisateur du 2026-10-28 (« tu enchaînes quand t'as fini D2 sur D3 ») =
  feu vert pour commits, PR, fusion squash dans la release. Les appels LIVE du témoin (5) sont des
  lectures (GET) et un rafraîchissement de jeton par le chemin canonique, jamais une re-capture (ADR 0023).

## Chiffres de départ (relevé du 2026-10-29 sur la branche D2 `bd97fe9b3`, à rouvrir au lot 0)

- `platform/halo` : 15 fichiers, 3 043 L prod, 2 970 L tests, sous-paquet `duration` (lu aussi par
  `games/halo_5` : 3 fichiers) ; `go list -deps` : 50 paquets LevelUp (inversion de couche D3.3).
  Imports prod : `domain` 10, `ctxkeys` 8 (`HaloTokens`, `HaloXUID`, `TitleSlug`, `Locale`,
  `WithHaloAuth`), `games` 5 (résolveur d'adresses : `DefaultEndpointResolver`, `Endpoint*`, `GamePrefix`),
  `netguard` 3 (`Check`), `assets` 3, `i18n` 2, `observability` 2, `metadata` 1,
  `games/halo_infinite` 1 (`AssetFetcher`, `AssetType*`), `domain/title` 1, `assetnames` 1,
  `duckdb.PersistSink` (provider.go:114,121). Méthodes de `HaloProvider` : FetchAndWarmTrack,
  FetchRemoteStats, FetchServiceRecord, FetchSeasonServiceRecord, FetchAsset, FetchMatchStats,
  FetchMedalsMetadata, GetMatchPrivacy, GetBattlePass(+WithRaw), GetChallenges(+WithRaw),
  FetchSeasonCalendar, FetchCSRSeasonCalendar, With* (RateLimit, Tokens, AssetResolver,
  TrackDefPersister, ItemDefPersister, TitleSlug). Autres : `LeaderboardScraper` (HTML),
  `CompareProvider`, `PrivacyProvider`, `DiscoveryClient`, `player_token_cache.go` (+ `auth_retry.go`,
  `InvalidateCachedPlayerTokens` 7 appelants), `catalog_fetcher.go`, `medal_provider.go`,
  `asset_name_fetcher.go`, `endpoints.go`. Importeurs prod : 15 paquets (`api/wire` 4, `games/halo_5` 3
  via `duration`, `watcher`, `sync`, `service`, `openspartan/mapper`, `api/handlers`, `api`, 7 `cmd/`).
- `sync/haloclient` : 16 fichiers, 3 226 L prod, 3 961 L tests ; deps 30 paquets. Imports prod :
  `games` 8, `domain` 3, `netguard` 2, `observability` 2, `ctxkeys` 2, `libs/film/decfilm` 2
  (`Inflate`, `FilmMajorVersionUnknown`, `Decompresser` dans `halo_client_http.go:135`),
  `libs/film/finalise` 1 (`Finalise`, `EstTempsForts`, `ChunkTypeTempsForts`), `libs/film/filmcache` 1
  (`local_film_cache.go`), `mapcatalog` 1 (`halo_client_mvar.go`), `analysis` 1 (`MMRKnown`,
  `halo_skill.go:203,243,253`), `domain/title` 1. Méthodes de `HaloAPIClient` : GetMatchHistory,
  GetMatchStats, GetCareerProgress, GetSpartanCustomization, GetCareerRank, GetMatchFilm,
  GetFilmChunks, GetFilmChunkURLs, GetHighlightEventsChunk, FetchMvarForMap, GetPlaylistConfig,
  GetMatchSkill, GetPlayerCSRs, GetPlaylistCsr, FetchAppearanceInputs, DiagnoseCustomizationImage,
  With* (LocalFilmCache, HTTPClient, Endpoints, Limiter). Importeurs prod : 6 paquets
  (`sync/killcollector` 6 fichiers, `cmd/levelup` 4, `service` 2, `sync/replayartifacts`,
  `sync/historyretry`, `sync`). Symboles les plus consommés : `FilmChunk` 17, `LocalFilmCache` 10,
  `NewLocalFilmCache` 5, `AppearanceDiagnosis` 5.
- `platform/auth` : 28 fichiers prod, 4 387 L (+ `pool/`, `capturecli/`), 6 084 L tests ; deps 9
  paquets seulement. Fichiers PROTOCOLE (audit D3.5, imports vérifiés) : `xsts.go` (→
  `observability/logging.WithEvent`), `sisu_provider.go` (→ `title.DefaultHaloAuthDescriptor`),
  `oauth_refresh.go` (→ `title`, `logging`, `netguard.Check`), `halo_exchange.go` (→ `domain.HaloTokens`,
  `title.AuthDescriptor`), `xbox_device_code.go` (→ `netguard`), `azure_credentials.go` (→ `title`),
  `auth_code.go`, `oauth_error.go`, `api_constants.go`, `refresh_user_xsts.go`, `xbox_profile_resolver.go`,
  `peoplehub_resolver.go` (aucun import du dépôt). Importeurs prod de `platform/auth` : 55 paquets
  (dont 44 `cmd/`), symboles les plus appelés : `NewMultiUserTokenStore` 41, `NewSISUProvider` 30,
  `RefreshHaloTokensViaStoreFirst` 29, `TokenProvider` 20 — TOUS d'exploitation : ils ne bougent pas.
  `domain.HaloTokens` : 42 fichiers prod.
- Résolveur d'adresses : singleton `games.DefaultEndpointResolver` / `SetDefaultEndpointResolver` lu par
  `platform/halo/endpoints.go`, `haloclient/endpoint_resolver.go` ET par sept fichiers de `games/`
  (`damage_model`, `career_xp`, `live_service_source`, `objective_stats_source`, `game_prefix`,
  `engagement_calibration`, `endpoint_default`) : le singleton est de l'EXPLOITATION et sert ailleurs
  que l'API ; D3 ne le retire pas, la brique reçoit ses adresses.
- `netguard.Check(ctx, surface)` : 11 appels dans les trois paquets ; `netguard` est de l'exploitation
  (bascule hors-ligne, allowlist `TestOutboundCallsAreNetguarded`) : la brique l'obtient par le
  `*http.Client` injecté (un `RoundTripper` qui appelle `Check`) — le ratchet netguard doit continuer à
  voir chaque appel sortant : à vérifier sur pièces au lot 1.
- Ratchets qui citent ces paquets : `no_film_reread_test`, `no_halowaypoint_literal_test`,
  `no_hardcoded_film_cache_dirs_test`, `no_pool_hasplayer_gate_test`,
  `no_request_scoped_halo_provider_test`, `sync_root_freeze_test`, `platform/auth/sentinel_test.go`,
  `sync/pooled_client_test`, `sync/assetnames_wiring_test`, netguard.

## Décisions tranchées avant exécution

- **D3-A — Forme : paquets INTERNES à `apps/go-api`, pas un module `libs/`** (décision 4 de
  l'utilisateur : `internal/haloapi/...` ; l'audit juge la publication à part sans pertinence). Le
  garde-fou G prend la forme d'une table des briques INTERNES dans `archlint` (même fichier
  `briques_imports_test.go`, seconde table : brique → dossier(s) → imports permis = stdlib + ses propres
  paquets + tiers listés), allowlist datée qui finit vide. Si un jour la brique part en module, c'est
  une décision neuve.
- **D3-B — `xblauth` (chaîne d'authentification)** : les 12 fichiers protocole de `platform/auth`
  (liste ci-dessus) déménagent dans `internal/haloapi/xblauth`. Coupes : (a) `domain.HaloTokens` →
  type `xblauth.Tokens` (mêmes champs) ; `platform/auth` convertit (`HaloTokensDepuisXBL`,
  `VersXBL`) et GARDE `domain.HaloTokens` pour ses 42 lecteurs ; (b) `title.AuthDescriptor` /
  `DefaultHaloAuthDescriptor()` → la brique reçoit un `xblauth.Descriptor` (identifiants d'application,
  scopes, audience/relying party) passé par l'appelant ; `platform/auth` garde les fonctions de
  commodité à descripteur par défaut (`ExchangeAccessToken`, `ExchangeXSTSForHaloTokens`, ...) comme
  façades d'exploitation qui appellent la brique avec `title.DefaultHaloAuthDescriptor()` converti ;
  (c) `observability/logging.WithEvent` → `slog` avec `ctx` (règle 3 CLAUDE.md) ; (d) `netguard.Check`
  → `*http.Client` injecté dont le `RoundTripper` porte la garde (fourni par `platform/auth`), et le
  ratchet netguard apprend que les appels sortants de `haloapi/` passent par ce client (allowlist ou
  règle, à poser au lot 1, jamais affaiblie). Les ratchets `sentinel_test.go` (guards 2 et 3 à
  allowlists VIDES) restent dans `platform/auth` et verts : la brique n'a AUCUNE lecture d'env ni de
  `sync_meta`.
- **D3-C — `infinite` (services Halo Infinite)** : UN client (`infinite.Client`) : transport
  (`halo_client_http.go` : nouvel essai, 429/`Retry-After`, erreurs typées `HTTPError`,
  `ErrFilmGone`/`IsFilmGoneErr`, `IsAuthError`), adresses (`infinite.Endpoints` : clés + défauts, reçues à
  la construction ; LevelUp les construit depuis `games.DefaultEndpointResolver` dans l'adaptateur),
  jetons passés par paramètre (`xblauth.Tokens` ou en-têtes) et JAMAIS lus dans le contexte
  (`ctxkeys.HaloTokens` reste côté LevelUp : l'adaptateur les sort du `ctx`), un fichier par service
  (stats/historique, skill/CSR, carrière, économie/apparence, CMS/découverte, défis, passe de combat,
  confidentialité, classements (le scraper HTML aussi : c'est du protocole), plaques, UGC/variantes,
  films : manifeste + fragments en OCTETS), DTO JSON BRUTS (`infinite.MatchStatsDTO`, ...). Les types
  `domain.*` que les providers rendent aujourd'hui (`ChallengesResponse`, `BattlePassResponse`,
  `MatchPrivacyInfo`, `RemoteServiceRecord`, `SeasonCalendar`, `LeaderboardEntry`, ...) RESTENT dans
  `domain` et sont produits par l'adaptateur à partir des DTO : là où le type `domain` EST déjà le
  JSON brut (vérifier champ par champ), le DTO est le même struct recopié dans la brique et la
  conversion est une copie champ à champ (test de parité par réflexion, comme `film_highlight_events`).
- **D3-D — Exploitation (reste ou naît dans LevelUp)** : `sync/filmfetch` (ex `halo_client_film.go`
  partie décodage/finalisation + `local_film_cache.go` : prend les octets de `infinite`, applique
  `decfilm.Inflate`/`finalise.Finalise`, écrit `filmcache`) ; `games/halo_infinite/haloadapter`
  (conversions DTO → `domain`, `assets.Ref`, `metadata.MedalEntry`, `analysis.MMRKnown`,
  `AssetFetcher`, choix de variante `mapcatalog`, résolution de langue `ctxkeys.Locale`/`i18n`) ;
  `platform/auth/tokencache` (ex `platform/halo/player_token_cache.go` + `auth_retry.go` :
  `ResolveFreshPlayerTokens`, `RetryWithFreshTokens`, `InvalidateCachedPlayerTokens`, 7 appelants
  suivis) ; `platform/halo/duration` → `games/halo_infinite/duration` ou `domain/duration` (lu par
  `halo_5` aussi : donc `internal/duration`, paquet feuille) ; `DefaultHaloProvider` /
  `HaloProvider` (10 + 14 usages) : l'adaptateur expose un type de même nom et mêmes méthodes
  (`haloadapter.Provider`) pour que les 15 importeurs ne changent que leur import — PAS de réécriture
  des appelants au-delà de l'import et du constructeur. `platform/halo` et `sync/haloclient`
  SUPPRIMÉS (règle 7 : 0 code mort).
- **D3-E — Tests** : les tests `httptest` des deux clients deviennent les tests de `infinite` (ils ne
  dépendent que du JSON) ; les tests qui vérifient une valeur de domaine deviennent des tests de
  l'adaptateur ; les fixtures `testdata/halo/*.json` suivent le paquet qui les lit. Aucun test
  disparu (T1).
- **D3-F — Ce qui NE bouge PAS** : `platform/auth` exploitation (stores, pool, boucles, `cli_refresh`,
  `access_token_store_first`, `link_strategy`, `cached_header_provider`, `capturecli`), ses ratchets et
  ADR 0023/0038 ; le singleton `games.DefaultEndpointResolver` (découverte : à retirer un jour, hors
  D3) ; `netguard` ; `ratebudget` ; `sync/v2`, `TitleSyncRunner` (ADR 0031 D-1 « per-title client
  packages + shared HTTP infra » : D3 le RÉALISE pour Halo Infinite — amender l'ADR 0031 d'un
  paragraphe daté, pas le contredire) ; `games/halo_5` (son client reste tel quel : un second
  `haloapi/halo5` est une décision neuve).
- **D3-G — Témoins avant/après** : T1 tests (listes), T2 fixtures enregistrées (les tests `httptest`
  prouvent la même valeur de domaine à partir du même JSON : AVANT = tests existants verts sur la base
  ; APRÈS = les mêmes assertions dans les nouveaux paquets), T3 LIVE lecture seule (commande
  `levelup` existante ou `cmd/diag_*` existants : relever au lot 0 lesquels couvrent historique, stats,
  skill, manifeste de film, actif, calendrier ; capturer leurs sorties JSON dans
  `temoins_d3_lot0_avant/`, rejouer au lot 6 ; horodatages et jetons masqués), T4 `/debug/vars` :
  noms de compteurs inchangés.
- **D3-H — Revue** : lot à risque (auth, tokens) → `adversarial-review` OBLIGATOIRE à DEUX relecteurs
  sur `xblauth`/`platform/auth` (lentilles : L3 + « protocole vs politique » ; L6) + un relecteur sur
  `infinite`/adaptateur (parité DTO → domaine). Une ronde.
- **D3-I — Docs** : ADR neuf `0042-halo-api-protocol-vs-exploitation.md` (EN) : critère de coupe du
  tableau de l'audit, les deux paquets, ce qui est injecté, ce qui reste ; ADR 0031 amendé ; ADR 0023
  : une ligne (la chaîne d'échange vit dans `haloapi/xblauth`, le store et la politique dans
  `platform/auth`) ; CLAUDE.md (tableau composants : ligne « API Halo (protocole) » ; section « Règle
  auth tokens » : chemin du helper canonique inchangé) ; `docs/SYNC_GUIDE.md` + FR si les chemins y
  sont ; `docs/ARCHITECTURE_V6.md` + FR.


## Décisions du pilote après le lot 0 (2026-10-29) — AMENDENT les décisions D3-A à D3-I ci-dessus

Prises sur le relevé du lot 0 (découvertes D3-L0-a à q). Le TABLEAU DU RELEVÉ (section « Exécution du lot 0 »)
FAIT FOI pour la destination de chaque fichier ; là où une décision D3-B/C/D ci-dessus le contredit, c'est le
relevé qui gagne (D3-L0-m : `asset_name_fetcher`, `catalog_fetcher`, `challenges_details`, `battlepass_details`,
`FetchMvarForMap`, `ServedFromCarry` -> `haloadapter`).

- **D3-J — Transport unique, politiques de reprise CONSERVÉES (D3-L0-g).** `infinite` a UN transport
  (`transport.go`) dont la politique de reprise est une VALEUR passée à la construction du client
  (`infinite.Retry{Essais, Retente func(status) bool, HonoreRetryAfter bool}`), et des erreurs typées
  (`HTTPError` avec `RetryAfter`, `ErrFilmGone`, `IsAuthError`). L'adaptateur construit UN client par ancien
  consommateur avec la politique qui reproduit sa boucle d'aujourd'hui (`provider.doGet` : 3 essais, 429 retenté ;
  `discovery.doGetWithLang` : 5xx seul ; `haloclient.doGet` : 4 essais, 429 rendu avec `RetryAfter`). AUCUN
  comportement ne change dans D3 ; la convergence des politiques est une décision future (découverte). Les deux
  hôtes de `FetchMatchStats` (`:443` ou non, PMT-1) restent deux clés d'adresse ; chaque appelant garde le sien.
  Tests `httptest` par politique (séquences 429 / `Retry-After` / 5xx) : ils prouvent la reproduction.
- **D3-K — Garde `netguard` par `Doer` injecté, erreur définitive (D3-L0-h).** La brique ne construit JAMAIS de
  client HTTP : elle reçoit un `infinite.Doer` (`Do(*http.Request) (*http.Response, error)`). LevelUp fournit, dans
  l'adaptateur, un `Doer` par SURFACE (`netguardDoer{surface, next}`) qui appelle `netguard.Check(ctx, surface)`
  avant `next.Do` et rend `ErrOffline` enveloppé dans `infinite.ErrDefinitif` ; la boucle de reprise de la brique
  s'arrête sur `errors.Is(err, ErrDefinitif)` (aucune attente). Le journal « une fois par surface » est conservé
  (une surface = un `Doer`). Délais : le `*http.Client` du `Doer` porte le délai d'aujourd'hui (15 s / 20 s /
  `xblExchangeTimeout`, 8 s pour le mapping d'emblèmes : D3-L0-i), les `ctx` à échéance de la brique sont conservés.
  Ratchet netguard (lot 1) : règle NEUVE pour `internal/haloapi/**` — interdit textuel de `http.Client{`,
  `http.DefaultClient`, `http.Get(`, `http.Post(`, `http.Head(` (la brique n'a pas le droit de posséder un client) ;
  les clés de chemin existantes de l'allowlist sont RENOMMÉES une pour une, datées, quand un fichier part (D3-L0-e) ;
  `netguard_demo_test` et `xbox_device_code_netguard_test` deviennent des tests de la façade `platform/auth`.
- **D3-L — Coupe de `platform/auth` révisée (D3-L0-d).** Vont dans `xblauth` : `halo_exchange.go` (scindé sous 500 L),
  la partie SISU de `sisu_provider.go`, `oauth_refresh.go` (le client Azure — id, secret, scopes, autorité — est une
  VALEUR `xblauth.ClientAzure` passée par l'appelant ; les compteurs expvar de `metrics.go` sont tenus par
  `platform/auth` à partir des issues typées que rend la brique), `xbox_device_code.go`, `auth_code.go`,
  `oauth_error.go`, `api_constants.go`, `xbox_profile_resolver.go`, `peoplehub_resolver.go`, la moitié protocole de
  `xsts.go` (échange XSTS). RESTENT dans `platform/auth` : `refresh_user_xsts.go`, `azure_credentials.go` (lecture
  d'environnement = exploitation ; il construit le `ClientAzure`), la moitié magasin de `xsts.go`
  (`AcquireXSTSForRTAWithProvenance`), les interfaces `TokenProvider` / `DeviceFlow` / `FlowExchanger` (satisfaites par
  une façade qui appelle la brique), `TokenFamily*`, `XboxScopes` / `xboxScopes` (deviennent des champs du
  descripteur). Les ratchets `sentinel_test.go` : guards 2 et 3 à allowlists VIDES inchangées ;
  `allowedClientSecretReaders` : clés renommées si un fichier bouge, AUCUN ajout.
- **D3-M — Périmètre « un client » borné aux trois paquets (D3-L0-k).** `internal/mapcatalog/ugc.go`,
  `internal/assets/fetcher_gamecms.go`, `internal/games/halo_5/client.go`, `internal/sync/xbox_client.go` ne bougent
  PAS dans D3 (découverte : les replier sur `infinite` est un chantier suivant). `infinite` = l'union des appels HTTP
  de `platform/halo` et `sync/haloclient`, rien de plus.
- **D3-N — Témoin T3 complété par un outil de diagnostic (D3-L0-b).** Lot 1 : `cmd/diag_api_lecture`, LECTURE SEULE,
  une sous-commande par surface (historique 1 page, stats d'un match, skill d'un match, manifeste d'un film, un actif
  de découverte, calendrier de saison / service record), sortie = JSON BRUT tel que reçu (en-têtes et jetons masqués,
  horodatages laissés : la comparaison les ignore), jetons par `RefreshHaloTokensViaStoreFirst` seulement. Il est
  écrit sur les clients d'AUJOURD'HUI (`platform/halo`, `haloclient`) et capture le témoin « avant » au lot 1 ; au
  lot 4 il est rebranché sur `haloadapter` / `infinite` (import et constructeur) et rejoue le témoin au lot 6. Il reste
  dans le dépôt (outil de diagnostic, comme les autres `cmd/diag_*`). Les rotations de jeton qu'il provoque pendant
  que le serveur tourne sont celles de D3-L0-f : au plus une par surface, consignées.
- **D3-O — Ratchets et scripts à clés de chemin (D3-L0-e, n).** Succès (3) reformulé : « aucun ajout ni élargissement
  d'allowlist ; renommage de clés UNE POUR UNE, daté, dans le commit qui déplace le fichier ; guards 2 et 3 du
  sentinel VIDES ». Concernés : `sentinel_test.go`, `netguard_coverage_test.go`, `no_halowaypoint_literal_test.go`,
  `auth_retry_guard_test.go`, `.gitleaks.toml:172`, `scripts/coverage_filter.sh`, `no_request_scoped_halo_provider_test`,
  `sync_root_freeze_test`, `no_film_reread_test`, `no_hardcoded_film_cache_dirs_test`. `racinesOctetsBruts` de
  `no_raw_film_bytes_outside_source_test.go` += `internal/haloapi` AU LOT 1.
- **D3-P — Tiers permis pour `haloapi` (D3-L0-o)** : `golang.org/x/sync/errgroup` et `golang.org/x/time/rate` ;
  `singleflight`, caches, singletons (`DefaultHaloProvider`, `emblemMapping*`, `challengesFetchSFGroup`,
  `playerTokenStore`, cache de confidentialité, `retryBaseDelay`) vont à l'adaptateur / `tokencache` ou deviennent
  des paramètres (D3-L0-i) ; `testexports.go` (seams de test en production) disparaît : les tests de la brique
  passent par la politique injectée.
- **D3-Q — Funnel `sync/haloclient_reexport.go` (D3-L0-j)** : re-pointé vers `haloadapter` / `filmfetch` en gardant
  chaque alias ; les 29 fichiers de 21 paquets ne changent pas. `CareerRankData` défini deux fois : découverte, hors D3.
- **D3-R — Secret dans `.env.local` (D3-L0-q)** : signalé à l'utilisateur dans le point d'étape du lot 0 ; fichier
  local ignoré par git, hors du dépôt ; aucune action de code.

## Lot 0 — Préalable : branche, témoins, état

- [x] 0.1 Worktree `../LevelUp-wt-decouplage-d3`, branche `refactor/decouplage-d3-api-halo` depuis
  `release/v8.1.0` (après #113) ; CI de la base verte.
  Statut : fait (2026-10-29). Worktree propre, HEAD `b92916328`, CI du HEAD verte (run #113, 13 min 53 s, vérifié en fin de lot ; il était encore en cours au début du relevé, D3-L0-c).
- [x] 0.2 T1 : `go test -json ./... -count=1` (go-api ; `libs/film` non concerné) → TSV.
  Statut : fait. T1 code 0 : 331 paquets, 16 422 tests, 0 fail (voir « Exécution du lot 0 »).
- [x] 0.3 T3 : relever les commandes existantes qui lisent l'API en lecture seule (candidats :
  `cmd/levelup` sous-commandes `diag`, `cmd/diag_matchstats_dump`, `cmd/diag_film_avail`,
  `cmd/diag_live_economy`, `cmd/get-token`, `cmd/probe-world-stats`) ; en choisir une par surface
  (historique, stats, skill, manifeste de film, actif/catalogue, calendrier/saison) ; les jouer avec le
  compte de l'utilisateur (jeton du store, rafraîchi par le chemin canonique s'il est expiré ; JAMAIS
  de re-capture) ; sorties JSON capturées, jetons/horodatages masqués, MD5.
  Statut : fait avec réserve consignée. Couvertes : stats (b), manifeste/chunk (d), historique (a) partiellement ; skill (c), actif (e), calendrier/service record (f) sans commande existante en lecture seule, rien de codé (D3-L0-b). Quatre rafraîchissements canoniques, aucune re-capture.
- [x] 0.4 Chiffres de départ rouverts ; relevé fichier par fichier de `platform/halo` et
  `sync/haloclient` : pour chaque fichier, « protocole » / « exploitation » / « mixte » (à couper) et
  la destination (D3-B/C/D), tableau dans le plan — c'est le relevé qui pilote les lots 2 à 4.
  Statut : fait. Écarts consignés (tableau du 0.4, D3-L0-l) ; relevé par fichier : 15 + 16 + 12 fichiers.
- [x] 0.5 Vérification sur pièces des points ouverts : (a) comment `netguard.Check` est tenu par le
  ratchet (`TestOutboundCallsAreNetguarded` : liste de fichiers ? motif ?) et comment un `RoundTripper`
  le satisferait ; (b) ce que `ctxkeys.WithHaloAuth` pose et qui le lit ; (c) `duckdb.PersistSink` dans
  `provider.go:114,121` (quel sens : un provider qui ÉCRIT en base ? → exploitation, à sortir vers
  l'appelant) ; (d) `halo.LeaderboardScraper` (HTML) : protocole (reste du HTML brut) ; (e) les
  constructeurs `With*` : lesquels sont de la politique (RateLimit, Tokens, TitleSlug, persisters) →
  adaptateur.
  Statut : fait. Réponses (a) à (g) dans « Exécution du lot 0 » ; (a) et (f) infirment deux hypothèses du plan (D3-L0-h, D3-L0-n).
- [x] 0.6 Gate : témoins écrits, thought_log « Découplage D3 — lot 0 ».
  Statut : fait. MD5 des témoins dans `md5s.txt`, plan mis à jour, entrée `.ai/thought_log.md`.

## Exécution du lot 0 (2026-10-29)

Relevé fait sur `b92916328` (HEAD de `refactor/decouplage-d3-api-halo` = `release/v8.1.0` après la fusion
de D2, #113), worktree `../LevelUp-wt-decouplage-d3`, `GOCACHE=%LOCALAPPDATA%\go-build-d3`. Témoins :
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d3_lot0_avant\`
(`md5s.txt` ; `commandes_T3.md` pour les commandes exactes). Aucun fichier de code touché ; aucun jeton dans un
témoin, un journal ou ce plan.

**0.1 État de départ.** Worktree propre (seul ce plan est non suivi), branche `refactor/decouplage-d3-api-halo`,
HEAD `b92916328`. CI de la base : le run du HEAD (#113, D2) était encore `in_progress` à 9 min 32 s au début du relevé
(le dernier run terminé, #112, était vert) ; il s'est terminé VERT en 13 min 53 s (vérifié en fin de lot, D3-L0-c).

**0.2 T1.** `go test -json ./... -count=1 -timeout 1200s` depuis `apps/go-api` (redirection binaire par `cmd`),
code 0, stderr vide : 331 paquets (191 ok, 140 sans fichier de test, 0 rouge), 16 422 tests avec sous-tests
(16 354 pass, 68 skip, 0 fail). Les trois paquets du chantier : `platform/auth` 284 lignes (+ `capturecli`,
`pool`), `platform/halo` 193 (+ `duration` 8), `sync/haloclient` 177. Les 4 skip de ces paquets :
`TestExchangeAuthorizationCode_MockServer_Skipped`, `TestMultiUserTokenStore_FilePermissions`,
`TestMultiUserTokenStore_FilePermissions_StrictMode`, `TestResolveNameplateURL_NegativeCfgResolvesPositive`.
Fichiers : `tests_T1.tsv` (paquet, test, verdict, trié), `paquets_ok.txt`, `resume_T1.txt`, `tests_go_api.jsonl`.

**0.3 T3 LIVE.** Voir `commandes_T3.md`. Surfaces couvertes : (b) stats d'un match (`cmd/diag_matchstats_dump`,
JSON brut de 31 710 octets) ; (d) manifeste et chunk des temps forts (`cmd/refresh_golden_fixture` redirigé vers
le dossier des témoins : version de film 41, 566 410 octets) et `cmd/diag_film_avail` (`GetFilmChunkURLs` :
1 film DISPO à 5 chunks) ; (a) historique : couverture PARTIELLE (l'outil n'imprime que le `match_id` du dernier
match, `fbd2f425-688d-420f-92ee-4331c7ed895c`). Surfaces SANS commande existante en lecture seule : (c) skill,
(e) actif de catalogue, (f) calendrier de saison, service record et carrière (D3-L0-b). Quatre rafraîchissements de
jetons du compte JGtm par `RefreshHaloTokensViaStoreFirst`, aucune re-capture (`updated_at` du fichier de jetons :
07:11:48Z avant, 07:18:45Z après ; valeurs secrètes jamais lues).

**0.4 Chiffres de départ rouverts.** Les chiffres du plan se retrouvent, aux écarts suivants près (D3-L0-l) :

| Point | Plan | Mesuré le 2026-10-29 |
|---|---|---|
| `platform/halo` prod / tests / deps LevelUp | 15 f / 3 043 L / 2 970 L / 50 | identique (sous-paquet `duration` : 185 L prod + 180 L test) |
| imports prod de `platform/halo` | domain 10, ctxkeys 8, games 5, netguard 3, assets 3, i18n 2, observability 2, metadata 1, halo_infinite 1, title 1, assetnames 1 | identique ; `duckdb.PersistSink` n'est PAS importé : le nom n'apparaît que dans deux commentaires (`provider.go:114,121`) |
| importeurs prod de `platform/halo` | 15 paquets | 13 directs + 2 via `duration` (`games/halo_5` : `events`, `mapping`, `mapping_carnage` ; `openspartan/mapper/iso8601.go`, non nommé au plan) = 15 |
| `sync/haloclient` prod / tests / deps | 16 f / 3 226 L / 3 961 L / 30 | identique (`testexports.go`, 18 L, compte en prod) |
| imports prod de `haloclient` | games 8, domain 3, netguard 2, observability 2, ctxkeys 2, decfilm 2, finalise 1, filmcache 1, mapcatalog 1, analysis 1, title 1 | identique |
| importeurs de `haloclient` | 6 paquets | 6 paquets directs (killcollector 6 fichiers, cmd/levelup 4, service 2, replayartifacts 1, historyretry 1, sync 1) ; MAIS 29 fichiers prod de 21 paquets le consomment par les alias de `sync/haloclient_reexport.go` (D3-L0-j) |
| symboles `haloclient.X` (prod) | FilmChunk 17, LocalFilmCache 10, NewLocalFilmCache 5, AppearanceDiagnosis 5 | identique |
| `platform/auth` prod / tests / deps | 28 f / 4 387 L / 6 084 L / 9 | identique (36 fichiers de test) |
| importeurs prod de `platform/auth` | 55 paquets dont 44 `cmd/` | 55 répertoires dont 40 `cmd/` (+ `scripts/warm_bp_assets`, `auth/pool`, `auth/capturecli`) ; `go list` n'en compte que 54 (balises de build) |
| symboles `auth.X` appelés hors du paquet (prod) | NewMultiUserTokenStore 41, NewSISUProvider 30, RefreshHaloTokensViaStoreFirst 29, TokenProvider 20 | 42, 32, 27, 16 (grep des alias : l'ordre de grandeur tient) |
| fichiers prod qui portent `domain.HaloTokens` | 42 | 43 |
| `netguard.Check` dans les trois paquets | 11 appels | 11 (halo 4, haloclient 4, auth 3) |
| singleton `games.DefaultEndpointResolver` | sept fichiers de `games/` | huit (`career_progression_source.go` en plus) ; lu aussi par `assets/fetcher_gamecms.go` et `haloclient/spartan_nameplate_resolver.go` |
| appelants prod d'`InvalidateCachedPlayerTokens` | 7 | 7 fichiers dont 2 dans `platform/halo` lui-même ; 5 appelants externes (`cmd/server`, `cmd/token-capture`, `cmd/token-import`, `api/handlers/auth_xbox_oauth`, `auth/capturecli`) |
| usages de `DefaultHaloProvider` / `HaloProvider` | 10 / 14 | 10 / 14 |

### Relevé fichier par fichier (pilote des lots 2 à 4)

Légende : P = protocole, E = exploitation, M = mixte (à couper). Destinations : `xblauth`, `infinite`, `filmfetch`
(`internal/sync/filmfetch`), `haloadapter` (`games/halo_infinite/haloadapter`), `tokencache`
(`platform/auth/tokencache`), `duration` (`internal/duration`), `reste` (reste dans `platform/auth`), `supprimé`.

**`internal/platform/halo/*.go` (15 fichiers, 3 043 L)**

| Fichier | L | Nature | Ce qui le couple | Destination |
|---|---|---|---|---|
| `asset_name_fetcher.go` | 57 | E | `assetnames.Fetcher`, `domain.HaloTokens`, `os.Getenv("LEVELUP_SYNC_RESOLVE_ASSETS")`, `NewHaloProvider().WithRateLimit().WithTokens()` | haloadapter |
| `auth_retry.go` | 62 | E | `ctxkeys.HaloXUID`, `ctxkeys.WithHaloAuth`, `InvalidateCachedPlayerTokens` ; la sentinelle `errHaloAuthFailure` / `IsAuthFailure` est du protocole | tokencache (la sentinelle devient l'erreur typée d'auth d'`infinite`) |
| `battlepass_details.go` | 223 | E (aucun HTTP) | `assets.Ref/Resolver/JSONPayload/Kind*`, `domain.HaloTokens`, appels des persisters, goroutines `context.Background()` | haloadapter |
| `catalog_fetcher.go` | 64 | E | `games/halo_infinite.AssetFetcher/DiscoveryAssetRaw/AssetType*` | haloadapter |
| `challenges_details.go` | 468 | E (aucun HTTP) | `domain.ChallengeItem/HaloTokens`, `assets.Ref/JSONPayload`, `i18n.HaloTags/Parse/LocaleEN`, `ctxkeys.Locale` | haloadapter |
| `compare_provider.go` | 190 | M | GET service record + DTO `serviceRecordResponse` (P) ; calcul `domain.NormalizedPlayerStats` / `RemoteServiceRecord` (KDA net, KDR, ratios : ADR 0006), `duration.SecondsInt64`, `ctxkeys.HaloTokens`, `defaultStatsHost` | infinite (GET, DTO, `buildSeasonServiceRecordURL`) + haloadapter (calculs, `domain.*`) |
| `discovery_client.go` | 278 | P (jetons et limiteur reçus) | `FetchAsset` (GET + décodage `PublicName` + URL d'image), `FetchMatchStats` (URL en dur, xuid « 0 »), 3e boucle de reprise `doGetWithLang`, `staticTokens`, `netguard.Check`, `games.EndpointDiscoveryUGC`, `domain.HaloTokens` en paramètre | infinite |
| `discovery_types.go` | 53 | P | aucun import LevelUp ; `AssetType`, tables segment d'URL / clé `MatchInfo` | infinite |
| `endpoints.go` | 45 | E | `games.DefaultEndpointResolver`, `ctxkeys.TitleSlug`, `games.GamePrefix` | haloadapter (construit les `infinite.Endpoints`) |
| `leaderboard_scraper.go` | 455 | M | HTML Waypoint : GET, `__NEXT_DATA__`, `parseLeaderboardPage`, `WaypointRef`, redirection de saison (P) ; `domain.LeaderboardEntry/WorldSeasonRef/WorldPlaylistRef/DeriveCSRTier`, rang recalculé, `i18n.FromColumns`, `FrenchName()` (« fr-FR » en dur), `observability.IncCounter("leaderboard_empty_page1")`, `netguard.Check` x2, `logModule` | infinite (DTO de page, fetch, parse) + haloadapter (entrées `domain.*`, noms, compteur) |
| `medal_provider.go` | 68 | M | GET `/medals/metadata.json` + `medalMetadataRaw` (P) ; `metadata.MedalEntry` (E), `ctxkeys.HaloTokens`, `games.EndpointGameCMS` | infinite + haloadapter |
| `player_token_cache.go` | 198 | E | `domain.HaloTokens` x11, `singleflight`, état global (`playerTokenStore`, `playerTokenRefresher`), TTL 50 min / marge 5 min | tokencache |
| `privacy_provider.go` | 186 | M | GET `matches-privacy` + `privacyResponse` (P) ; cache TTL 30 min / échec 5 min (état), `domain.MatchPrivacyInfo`, `timing.FromContext`, `ctxkeys.HaloTokens` | infinite (GET, DTO) + haloadapter (cache, `parsePrivacyResponse`, repli) |
| `provider.go` | 555 | M | struct `HaloProvider`, `With*` x6, `rateLimiter` (politique), `doGet` (boucle de reprise, `errHaloAuthFailure`), `fetchBattlePass`/`fetchChallenges` (GET + DTO + `domain.BattlePassResponse/ChallengesResponse` + `assetResolver.Warm` en goroutine), `DefaultHaloProvider` (singleton global), `singleflight` des défis, interfaces de persisters, `ctxkeys` x8, `title.DefaultSlug`, `netguard.Check` ; 555 L > 500 | infinite (`doGet`, DTO des opérations et des decks, URLs) + haloadapter (`Provider`, `With*`, `domain.*`, singleflight, persisters, ctx, singleton, limiteur : interface injectée côté brique) |
| `season_provider.go` | 141 | M | GET calendriers + `seasonCalendarRaw`/`csrSeasonRaw` (P) ; `domain.SeasonCalendar/CSRSeasonCalendar` (ids `s%d`, `time.Now()`), `ContentHash` exporté (appelé par `cmd/refresh-metadata` x3) | infinite (GET, DTO) + haloadapter (conversion, `ContentHash`) |
| `duration/duration.go` | 185 | P (feuille, aucun import LevelUp) | lu par `compare_provider`, `games/halo_5` (3 fichiers), `openspartan/mapper` | duration |

**`internal/sync/haloclient/*.go` (16 fichiers, 3 226 L)**

| Fichier | L | Nature | Ce qui le couple | Destination |
|---|---|---|---|---|
| `appearance_inputs.go` | 64 | P | extraction de champs d'un DTO ; aucun import LevelUp | infinite |
| `dto_aliases.go` | 7 | E | alias `CareerRankData = domain.CareerRankSnapshot` (défini aussi dans `sync/career.go`) | supprimé |
| `endpoint_resolver.go` | 63 | E | `ctxkeys.TitleSlug`, `games.DefaultEndpointResolver/GamePrefix*` | haloadapter |
| `halo_client.go` | 300 | M | hôtes par défaut, `HTTPError`/`IsAuthError`, `GetMatchHistory`/`GetMatchStats` (P) ; struct `HaloAPIClient` (jetons d'instance, limiteur, `LocalFilmCache`, `games.EndpointResolver`), `With*`, interface `HaloClient` (contrat lu par le pool de `sync`), `retryBaseDelay` variable mutable | infinite (transport, erreurs, historique, stats) + haloadapter (`HaloClient`, `With*` de politique) |
| `halo_client_career.go` | 468 | M | GET économie (carrière, personnalisation, apparence) et parse des payloads (P) ; `domain.SpartanCustomizationData/CareerRankSnapshot` via alias, `doPlayerGatedGet` (401 -> « absent » : politique), `games.EndpointEconomy/GameCMS` | infinite (GET, DTO, parse) + haloadapter (`domain.*`, politique du 401) |
| `halo_client_film.go` | 495 | M | manifeste HTTP + téléchargement parallèle `errgroup` (P) ; cache local d'abord, `finalise.Finalise/EstTempsForts/ErrFilmNonFinalise`, `decfilm.Inflate/FilmMajorVersionFromHeader/Unknown`, sélection par type, constantes `FilmChunkType*`, `FilmChunk`/`FilmChunkData`, `games.EndpointUGCFilm` | infinite (manifeste brut, `buildChunkURL`, octets, `GetFilmChunkURLs`) + filmfetch (cache, finalisation, version, sélection, `FilmChunk`) |
| `halo_client_film_taille.go` | 70 | M | `ErrChunkIncomplet`, `blobRecu` (contrôle de taille sur le brut PUIS `inflateBlob`), compteurs `observability.IncCounter/AddInt` | infinite (contrôle de taille, compteurs par crochet) + filmfetch (`inflateBlob`) |
| `halo_client_http.go` | 390 | M | `doGet` (reprises, 429 rendu tel quel avec `Retry-After`, `HTTPError` typé), `downloadBlob` (statuts du CDN, `BlobHTTPError`), `parseRetryAfter`, `backoff`, `rateWait` (P) ; `netguard.Check` x2, `observability.AddInt/AllowThrottledLog/NetworkFloodWindow`, `decfilm.Decompresser`, alias `SpartanCustomizationData` | infinite sauf la décompression (filmfetch) ; netguard -> `RoundTripper` injecté ; compteurs -> crochets |
| `halo_client_mvar.go` | 47 | E | `mapcatalog.NewClient/ChoisirFichierVariante` (3e client UGC), `domain.HaloTokens` | haloadapter |
| `halo_client_playlist.go` | 112 | P | GET playlist + DTO ; `games.EndpointDiscoveryUGC` | infinite |
| `halo_skill.go` | 407 | M | GET skill + `matchSkillResponse`/`flexFloat` (P) ; `transformMatchSkillResponse`, `computeEnemyMMR`, `analysis.MMRKnown` x3, `MatchSkillData`, `GetPlayerCSRs`, `ErrSkillStatsUnavailable` | infinite (GET, DTO) + haloadapter (transformation, `MMRKnown`, types) |
| `halo_skill_csr.go` | 94 | M | GET CSR de playlist + `parsePlaylistCSR` | infinite (GET, DTO) + haloadapter (`PlayerPlaylistCSR`) |
| `local_film_cache.go` | 149 | E | `filmcache.ManifestsRoot/LireChunk/ErrChunkTronque/ManifestPath`, `title.FilmShortMatchID`, `chunkDuCache` | filmfetch |
| `spartan_nameplate_diagnosis.go` | 169 | E | vocabulaire `Verdict/ServedFrom/Detail`, `AppearanceDiagnosis` (5 usages prod) | haloadapter |
| `spartan_nameplate_resolver.go` | 373 | M | GET du mapping d'emblèmes par `http.DefaultClient` (hors client injecté) + `netguard.Check` direct (P) ; cache global muable (`emblemMapping*`, TTL 6 h : état), `games.DefaultEndpointResolver`, `ctxkeys.TitleSlug/HaloXUID`, verdicts | infinite (GET + DTO du mapping) + haloadapter (cache, résolution, verdicts) |
| `testexports.go` | 18 | E (seams de test en fichier prod) | `rate.Limiter` | supprimé (seams dans un `_test.go` d'`infinite`) |

**`internal/platform/auth` : les 12 fichiers « protocole » du plan (D3-B), imports vérifiés**

| Fichier | L | Nature | Ce qui le couple | Destination |
|---|---|---|---|---|
| `xsts.go` | 171 | M | `logging.WithEvent` ; `AcquireXSTSForRTA` (P) mais `AcquireXSTSForRTAWithProvenance` prend un `*MultiUserTokenStore` (`Load`, `UpdateTokenClientFamily`) ; `&http.Client{Timeout: 20s}` local | xblauth (`AcquireXSTSForRTA`, `XSTSResult`) ; `...WithProvenance` reste |
| `sisu_provider.go` | 194 | M | `title.DefaultHaloAuthDescriptor` ; `SISUProvider` implémente `TokenProvider`, `DeviceFlow`, `FlowExchanger` (interfaces d'exploitation de `provider.go`) ; `TokenFamilyXboxNative` (défini dans `multi_user_token_store.go`) | xblauth (flux appareil SISU, échange) ; `SISUProvider` (32 appelants) reste en façade |
| `oauth_refresh.go` | 246 | M | `title`, `logging.WithEvent`, `netguard.Check`, compteurs expvar de `metrics.go` (`oauthRefreshRetryMSATotal`, `oauthRefreshRetryPublicTotal`, `recordOAuthRefreshOutcome`), `TokenFamily*` du fichier du magasin, `ResolveAzureOAuthClient()` (env) | xblauth avec client Azure, crochets de comptage et familles passés |
| `halo_exchange.go` | 515 | M | `domain.HaloTokens` (dans `ExchangeResult`), `title.AuthDescriptor` + 2 façades `DefaultHaloAuthDescriptor`, `TokenFamily*`, contexte (`WithTokenClientFamily`, observateur), `postJSON` ; 515 L > 500 | xblauth (à scinder) ; façades à descripteur par défaut : reste |
| `xbox_device_code.go` | 229 | P | `netguard.Check` x2 ; `http.DefaultClient` si client nul | xblauth |
| `azure_credentials.go` | 94 | E (lecture d'env) | `os.Getenv("LEVELUP_OAUTH_CLIENT_ID")`, `os.Getenv("SPNKR_AZURE_CLIENT_SECRET")` ; `XboxScopes` initialisé par `title.DefaultHaloAuthDescriptor()` ; garde-rail 5 du sentinel nomme ce chemin | reste (la brique reçoit un `AzureOAuthClient` passé) |
| `auth_code.go` | 162 | P après injection | aucun import LevelUp, mais appelle `ResolveAzureOAuthClient()` (env) et `xboxScopes` ; allowlist netguard | xblauth (client Azure passé) |
| `oauth_error.go` | 67 | P | aucun | xblauth (alias de ré-export dans `platform/auth` : `ClassifyAuthError`, `AuthErrorConfig`, `AuthErrorRevoked` ont des appelants hors paquet) |
| `api_constants.go` | 24 | P | aucun | xblauth |
| `refresh_user_xsts.go` | 139 | E (PAS du protocole) | `MultiUserTokenStore.Load/Upsert`, `persistRotatedRT`, `UserTokens` ; appelle `ExchangeRefreshTokenWithRotation` et `AcquireXSTSForRTA` | reste (consomme `xblauth`) |
| `xbox_profile_resolver.go` | 91 | P | aucun import LevelUp (`CachedHeaderProvider` cité en commentaire seulement) ; allowlist netguard | xblauth |
| `peoplehub_resolver.go` | 77 | P | aucun ; allowlist netguard | xblauth |

**Synthèse par destination (fichiers).**
- `platform/halo` (15) : haloadapter seul 5 (`asset_name_fetcher`, `battlepass_details`, `catalog_fetcher`, `challenges_details`,
  `endpoints`) ; tokencache 2 (`auth_retry`, `player_token_cache`) ; infinite seul 2 (`discovery_client`, `discovery_types`) ;
  coupés infinite + haloadapter 6 (`compare_provider`, `leaderboard_scraper`, `medal_provider`, `privacy_provider`,
  `provider`, `season_provider`) ; `duration/` -> `internal/duration`.
- `sync/haloclient` (16) : infinite seul 2 (`appearance_inputs`, `halo_client_playlist`) ; coupés infinite + haloadapter 5
  (`halo_client`, `halo_client_career`, `halo_skill`, `halo_skill_csr`, `spartan_nameplate_resolver`) ; coupés infinite +
  filmfetch 3 (`halo_client_film`, `halo_client_film_taille`, `halo_client_http`) ; haloadapter seul 3 (`endpoint_resolver`,
  `halo_client_mvar`, `spartan_nameplate_diagnosis`) ; filmfetch seul 1 (`local_film_cache`) ; supprimés 2 (`dto_aliases`,
  `testexports`).
- `platform/auth` (12) : xblauth pur 5 (`xbox_device_code`, `oauth_error`, `api_constants`, `xbox_profile_resolver`,
  `peoplehub_resolver`) + `auth_code` après injection ; coupés 4 (`xsts`, `sisu_provider`, `oauth_refresh`, `halo_exchange`) ;
  restent 2 (`azure_credentials`, `refresh_user_xsts`).
- Totaux : 43 fichiers relevés (+ `duration/duration.go`) ; 18 contribuent à `infinite`, 19 à `haloadapter`, 4 à
  `filmfetch`, 2 à `tokencache`, 10 à `xblauth` (+ 2 qui restent dans `platform/auth` sans passer par la brique), 1 à `duration`, 2 supprimés (un
  fichier coupé compte pour chacune de ses destinations).

### Réponses aux vérifications sur pièces (0.5)

- **(a) netguard.** `netguard.go` : drapeau de processus `offline` (atomique), `Check(ctx, surface)` rend `ErrOffline` et
  logge une fois par surface (11 noms de surface aujourd'hui). `TestOutboundCallsAreNetguarded` ne regarde pas les appels :
  il parcourt TOUS les `.go` non test de `internal/` et exige, par FICHIER, que tout fichier contenant un des cinq
  marqueurs textuels (`.Do(req)`, `.Do(httpReq)`, `http.Get(`, `http.Post(`, `http.PostForm(`) contienne la chaîne
  `netguard.Check`, sinon il doit figurer dans une allowlist à clés de CHEMIN (dont 4 fichiers de `platform/auth` à
  déplacer). Un `RoundTripper` qui appelle `Check` ne satisfait PAS ce test (les fichiers de la brique auront `.Do(req)`
  sans la chaîne) : règle neuve à poser au 1.4 (voir D3-L0-h).
- **(b) `ctxkeys.WithHaloAuth`.** Pose `*domain.HaloTokens` + xuid dans le `ctx` (`WithHaloXUID` : le xuid seul). Poseurs
  prod : middleware de session, `api/wire/registry_auth.go` (3 sites), `api/handlers/sync_handler`, `scheduler`, `service`,
  `watcher`, `games/halo_5/livesync`, `platform/halo/auth_retry.go` et 11 `cmd/`. `ctxkeys.HaloTokens` est lu par 5 fichiers de
  `platform/halo`, 2 d'`api/wire`, 5 de `service`, 1 de `sync`, 1 de `games/halo_5`, 1 `cmd` : seuls les 5 de `platform/halo`
  sont dans le périmètre (l'adaptateur les sort du `ctx`).
- **(c) `duckdb.PersistSink`.** Aucun import : `provider.go:114,121` définit les interfaces `TrackDefinitionPersister` /
  `ItemDefinitionPersister` « implémentées par `PersistSink` ». Le provider n'écrit PAS lui-même : il appelle le callback
  après une résolution `assets.Resolver.Get` (`battlepass_details.go`, dont un appel en goroutine `context.Background()`),
  câblé par `api/wire/registry_auth.go:35,122`. Exploitation pure : les interfaces et les `With*Persister` vont à `haloadapter`.
- **(d) `LeaderboardScraper`.** Page HTML Waypoint dont le JSON `__NEXT_DATA__` est extrait par expression régulière
  (`nextDataRe`) : fetch + extraction + DTO (`waypointPageProps`) = protocole, sans `x/net/html` (le tiers « à mesurer » du 1.2
  est donc inutile). La conversion en `domain.LeaderboardEntry` (rang recalculé, `DeriveCSRTier`), les noms (`i18n.FromColumns`,
  `FrenchName`) et le compteur `leaderboard_empty_page1` = exploitation. Le scraper n'envoie aucun jeton (site public).
- **(e) `With*`.** `HaloProvider` : `WithRateLimit`, `WithTokens`, `WithTitleSlug`, `WithAssetResolver`,
  `WithTrackDefPersister`, `WithItemDefPersister` = tous politique/adaptateur (limiteur et jetons deviennent des paramètres
  d'`infinite`). `HaloAPIClient` : `WithHTTPClient`, `WithEndpoints`, `WithLimiter` = protocole (injections de la brique,
  restent) ; `WithLocalFilmCache` = exploitation (-> `filmfetch`).
- **(f) `decfilm.Decompresser`.** Un flux zlib appliqué au CORPS du blob (aucun `Content-Encoding`), strict sur l'en-tête :
  c'est le format de stockage des films, pas l'en-tête HTTP standard. Le ratchet
  `archlint/no_raw_film_bytes_outside_source_test.go` interdit `compress/zlib` dans ses racines (`libs/film`,
  `internal/analysis`, `internal/sync`, `internal/replaybuild`, `cmd`) : `infinite` rend donc les octets COMPRIMÉS (le
  contrôle de taille annoncée porte sur le brut : il reste dans `infinite`) et `filmfetch` décompresse par
  `decfilm.Decompresser`. `internal/haloapi` n'est pas dans ces racines : à ajouter (D3-L0-n).
- **(g) Tests des trois paquets** (N = nombre de fonctions `Test`).
  - `platform/halo` (94) : MIXTES (HTTP + assertions de domaine, à couper) `provider_test` (14), `leaderboard_scraper_test` (8,
    fixture `testdata/leaderboard_sample.html`), `privacy_provider_test` (10), `season_provider_test` (2) ; protocole pur
    `discovery_image_test` (1) ; neutre `content_hash_test` (3) ; adaptateur (valeurs de domaine) `battlepass_details_test`
    (14), `challenges_details_helpers_test` (14), `compare_provider_test` (2), `endpoints_test` (1),
    `privacy_provider_failure_test` (3), `provider_limiter_shared_test` (1) ; tokencache `auth_retry_test` (4),
    `auth_retry_guard_test` (2, garde statique par NOMS de fichiers), `player_token_cache_test` (15).
  - `sync/haloclient` (111) : `httptest` purs sans import LevelUp, déplaçables dans `infinite` : `halo_client_extra_test` (35),
    `halo_client_blob_retry_test` (15, compteurs), `halo_client_film_parallel_test` (6), `halo_client_contract_test` (1, fixture
    `testdata/halo/match_history_response.json`), `halo_client_film_taille_test` (1), `isautherr`, `isfilmgone`, `retryafter`,
    `url_dryrun`, `playlist` (1+1+1+2+3) ; filmfetch `halo_client_film_finalise_test` (6), `halo_client_film_version_test` (2),
    `local_film_cache_test` (1) ; adaptateur `halo_skill_test` (8), `halo_skill_playlist_csr_test` (3) (valeurs
    `MatchSkillData`), `halo_client_endpoints_test` (3), `spartan_nameplate_diagnosis_test` (11),
    `spartan_nameplate_resolver_test` (8) ; garde-rails `netguard_demo_test` (2, passe par un client par défaut : devient un
    test de façade), `no_text_predicate_test` (1, suit `infinite`).
  - `platform/auth` : les tests des fichiers déplacés sont `httptest` purs SAUF `xsts_test` (12 usages du magasin),
    `xsts_provenance_test`, `refresh_user_xsts_test` (magasin, reste), `azure_credentials_test` et `oauth_refresh_internal_test`
    (`t.Setenv`, nommés par l'allowlist du sentinel) ; `xbox_device_code_netguard_test` passe un client arbitraire (devient un
    test de façade).


## Lot 1 — Garde-fou G avant tout déplacement

- [x] 1.1 Table des briques INTERNES dans `archlint` (fichier frère `briques_internes_test.go`, même
  helper de parcours que `briques_imports_test.go`) : au lot 1 la ligne `haloapi` pointe les dossiers
  ACTUELS des coupes (`apps/go-api/internal/platform/halo`, `apps/go-api/internal/sync/haloclient`, et
  pour `platform/auth` la LISTE des 12 fichiers protocole) avec les tolérances datées par import visé
  (comptes mesurés, qui ne font que descendre) ; au lot 2 la ligne devient `apps/go-api/internal/haloapi`
  (récursif) et les anciens dossiers en sortent à mesure qu'ils se vident.
  Statut : fait (2026-10-29). `briques_internes_test.go` (+ `briques_internes_garde_test.go`) ; 20 tolérances mesurées (tableau de « Exécution du lot 1 »). Le dossier cible `internal/haloapi` est déjà dans la ligne (arbre) et porte `doc.go`.
- [x] 1.2 Imports permis pour `haloapi` : stdlib, `internal/haloapi/...`, tiers `golang.org/x/sync/errgroup`
  et `golang.org/x/time/rate` (D3-P) ; tout `levelup/go-api/internal/...` hors `haloapi` = violation.
  Statut : fait. Les dossiers du périmètre s'importent entre eux ; un fichier de `platform/auth` n'a que son propre paquet. `x/sync/singleflight` (2 fichiers de `platform/halo`) est hors liste : tolérance datée, lot 4 (D3-L1-e).
- [x] 1.3 `domain`, `analysis`, `canonical` n'importent pas `haloapi` (test de fermeture, comme pour
  `libs/film`).
  Statut : fait. `TestFermetureDeDomainEtAnalysisSansHaloapi` (`briques_internes_fermeture_test.go`) : fermeture MESURÉE à 299 paquets, aucun paquet de `haloapi`, `platform/halo` (arbre) ni `sync/haloclient` : pas d'allowlist.
- [x] 1.4 netguard : règle NEUVE pour `internal/haloapi/**` (D3-K : interdit textuel de `http.Client{`,
  `http.DefaultClient`, `http.Get(`, `http.Post(`, `http.Head(`), posée AVANT le déplacement, prouvée par
  mutation sur un extrait synthétique présenté comme `internal/haloapi/infinite/x.go` ; `racinesOctetsBruts` de
  `no_raw_film_bytes_outside_source_test.go` += `internal/haloapi` (D3-O).
  Statut : fait, avec deux écarts de forme consignés. Règle posée dans `archlint/haloapi_sans_client_http_test.go` (test frère, pas dans `netguard_coverage_test.go` : il lit l'AST et réutilise `parseGoSnippet`, voir D3-L1-b) et jugée sur l'AST, pas au texte (un commentaire qui cite `http.DefaultClient` ne rougit pas) ; `racinesOctetsBruts` += `internal/haloapi` et `internal/haloapi/doc.go` créé (le ratchet échoue sur une racine absente, D3-L1-d).
- [x] 1.5 `cmd/diag_api_lecture` (D3-N) écrit sur les clients d'aujourd'hui ; témoin T3 « avant » capturé
  pour les six surfaces (JSON bruts, jetons masqués, MD5) dans `temoins_d3_lot1_avant/`.
  Statut : fait avec réserve consignée. Cinq surfaces sur six lues (code 0) ; `calendrier` rend HTTP 403 (D3-L1-a), repli sur une septième sous-commande `servicerecord` (code 0).
- [x] 1.6 Gate : archlint vert, mutations (tolérance -1 → rouge ; import `domain` depuis un fichier
  présenté comme `internal/haloapi/x.go` → rouge ; `http.Get(` dans un extrait `haloapi` → rouge).
  Commit « D3 lot 1 » avec ce plan ; CI verte.
  Statut : fait (build, vet, archlint, netguard, gofmt, lint 0 issue ; mutations dans « Exécution du lot 1 »). CI : voir le journal.

## Exécution du lot 1 (2026-10-29)

Code : branche `refactor/decouplage-d3-api-halo` depuis `b92916328`, deux commits (garde-fou 1.1-1.4 ; outil et
témoin 1.5). Témoins : `%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d3_lot1_avant\`
(`commandes.md`, `md5s.txt`).

**Fichiers.** `internal/archlint/briques_internes_test.go` (table `briquesInternes`, parcours, confrontation des
tolérances), `briques_internes_garde_test.go` (auto-preuves par extrait `parseGoSnippet`),
`briques_internes_fermeture_test.go` (1.3), `haloapi_sans_client_http_test.go` (1.4), `internal/haloapi/doc.go`
(paquet parent sans code), `cmd/diag_api_lecture/{main,surfaces}.go` (1.5). `briques_imports_test.go` : les deux
parcours à `t.Fatalf` (`sousDossiersGo`, `importsDuDossier`) délèguent désormais à des formes à erreur
(`sousDossiersGoErr`, `importsDuDossierErr`) que la nouvelle table réutilise pour REFUSER un dossier déclaré qui
n'existe pas ; comportement inchangé.

**Ligne `haloapi`.** Arbres : `internal/haloapi`, `internal/platform/halo` (`duration/` compris),
`internal/sync/haloclient` ; `platform/auth` par la LISTE de 10 fichiers prod + 15 `_test.go` jumeaux (`halo_exchange`,
`sisu_provider`, `oauth_refresh`, `xbox_device_code`, `auth_code`, `oauth_error`, `api_constants`,
`xbox_profile_resolver`, `peoplehub_resolver`, `xsts`). Permis : stdlib, `levelup/go-api/internal/haloapi[/...]`, les
arbres ci-dessus entre eux, le paquet propre d'un fichier de `platform/auth`, `x/sync/errgroup`, `x/time/rate`.

**Tolérances mesurées (fichiers, tests compris), datées 2026-10-29.** Clé = import sans `levelup/go-api/internal/`.

| Import visé | Fichiers | Origine (auth / halo / haloclient) | Lot qui le vide |
|---|---|---|---|
| `domain` | 22 | 1 / 18 / 3 | 2 (1), 4 (21) |
| `games` | 16 | 0 / 7 / 9 | 4 |
| `ctxkeys` | 14 | 0 / 11 / 3 | 4 |
| `domain/title` | 9 | 6 / 1 / 2 | 2 (6), 4 (3) |
| `platform/netguard` | 9 | 3 / 3 / 3 | 2 (3), 4 (6) |
| `observability` | 6 | 0 / 2 / 4 | 4 |
| `assets` | 5 | 0 / 5 / 0 | 4 |
| `libs/film/filmcache` | 4 | 0 / 0 / 4 | 4 |
| `libs/film/finalise` | 3 | 0 / 0 / 3 | 4 |
| `i18n` | 3 | 0 / 3 / 0 | 4 |
| `libs/film/decfilm` | 2 | 0 / 0 / 2 | 4 |
| `observability/logging` | 2 | 2 / 0 / 0 | 2 |
| `observability/timing` | 2 | 0 / 2 / 0 | 4 |
| `golang.org/x/sync/singleflight` | 2 | 0 / 2 / 0 | 4 |
| `analysis`, `assetnames`, `games/halo_infinite`, `games/mappings`, `mapcatalog`, `metadata` | 1 chacun | halo ou haloclient | 4 |

Total : 20 imports visés, 105 franchissements. `platform/halo` ne tire pas `platform/auth` ; `haloclient` n'importe rien de
`sync`. Règles de confrontation (mêmes que D2) : compte > tolérance, compte < tolérance (« baisse à N »), tolérance 0
(« retirer l'entrée »), import absent de la table : tous rouges, chacun exercé par `TestBriquesInternes_AutoPreuve_Tolerances`.
Le lot qui vide chaque entrée est porté par la tolérance elle-même (champ `lot`, obligatoire, contrôlé).

**Fermeture (1.3).** `go list -deps -test ./internal/domain/... ./internal/analysis/... ./internal/games/canonical/...` :
299 paquets, AUCUN sous `internal/haloapi`, `internal/platform/halo` ni `internal/sync/haloclient` (`platform/auth` non
plus). Pas d'allowlist ; le test interdit les trois arbres.

**Mutations (1.6), chacune lancée sur le disque puis annulée.**

| Mutation | Test rouge | Message |
|---|---|---|
| tolérance `assets` 5 -> 4 | `TestBriquesInternes_Imports` | `assets : 5 fichier(s) > tolérance 4` |
| tolérance `assets` 5 -> 6 | idem | `assets : baisse à 5 (tolérance 6) : abaisser la tolérance, le ratchet ne fait que descendre` |
| `internal/haloapi/mut_x.go` important `internal/domain` | idem | `domain : 23 fichier(s) > tolérance 22` |
| `internal/haloapi/infinite/x.go` appelant `http.Get(` | `TestHaloapiNePossedeAucunClientHTTP` | `x.go:5 : appelle http.Get (client par défaut)` |
| `internal/haloapi/mut_zlib.go` important `compress/zlib` | `TestAucuneLectureDOctetsBrutsHorsDeLaSource` | `internal/haloapi/mut_zlib.go \| decompression   vu : compress/zlib` |
| `internal/analysis/zz_mut_fermeture.go` important `platform/halo/duration` | `TestFermetureDeDomainEtAnalysisSansHaloapi` | `... contient l'API Halo : platform/halo/duration` |
| dossier `sync/haloclient` renommé dans la table | `TestBriquesInternes_Imports` | `... n'existe pas : la table décrit une arborescence disparue` |

Les mêmes cas sont aussi prouvés sans toucher au disque par les auto-preuves sur extraits (un extrait présenté comme
`internal/haloapi/x.go` qui importe `domain` ; un extrait `internal/haloapi/infinite/x.go` qui appelle `http.Get`, `Post`,
`Head`, `PostForm`, construit `http.Client{}` / `&http.Client{}` / `new(http.Client)` ou lit `http.DefaultClient`, aussi
sous un alias d'import ; le même `http.Client{}` passe en `_test.go`, `http.DefaultClient` et `http.Get` non).

**Outil `cmd/diag_api_lecture` (D3-N).** Sous-commandes : `historique` (25 matchs), `stats <id>`, `skill <id>`,
`manifeste <id>`, `actif <type> <asset> <version>`, `calendrier`, `servicerecord`. `-xuid` optionnel (défaut : premier joueur de
`db_profiles.json`), racine par `config.Load()` (pattern des `cmd/diag_*`), jetons par `RefreshHaloTokensViaStoreFirst` seulement,
arguments validés AVANT l'authentification (une faute de frappe ne fait pas tourner de jeton). Sortie : `calendrier` = corps
JSON reçu, octet pour octet ; les autres = `json.MarshalIndent(valeur, "", "  ")` du type Go que le client rend (les clients
ne donnent pas les octets), sans saut de ligne final ; adresses de blobs sans requête. `manifeste` = références de fragments
(`GetFilmChunkURLs`), le client n'exposant pas le manifeste brut. Aucun jeton ni en-tête écrit ; journal `slog` sur stderr.

**Témoin « avant » (lot 1), compte JGtm, match `fbd2f425-688d-420f-92ee-4331c7ed895c`, actif = variante de carte de ce
match (« Cliffhanger »).**

| Surface | Code | Octets | MD5 |
|---|---|---|---|
| `historique` | 0 | 2 698 | `60a2591c305729478c7223f8f481c604` |
| `stats` | 0 | 31 710 | `697ff1f0149d3e8f633cf2e9ceb73e8f` (IDENTIQUE au témoin du lot 0) |
| `skill` | 0 | 615 | `3abcd07bdf2ced7b82789f5838e6c229` |
| `manifeste` | 0 | 7 643 | `4eaf85bc852d67bb4f7c989bfe474d93` |
| `actif` | 0 | 373 | `3195a6077ee5bdd094daad591a7d25c3` |
| `calendrier` | 1 | 0 | échec : HTTP 403 (D3-L1-a) |
| `servicerecord` | 0 | 5 987 | `4dfdef90e9208e547125db378c08289b` |

Sept lancements = sept rotations de la chaîne de JGtm en 51 s (08:47:04 à 08:47:55, heure locale), serveur local en
marche, aucune re-capture (D3-L0-f). Lot 6 : mêmes sept lancements. Comparaison : stats, skill, manifeste, actif
identiques octet pour octet ; `historique` (tête de page) et `servicerecord` (cumuls) bougent si le joueur joue.

## Lot 2 — `haloapi/xblauth` (chaîne d'authentification) — lot à risque

Numérotation ajustée à l'exécution (brief du pilote du 2026-10-29, six étapes au lieu de cinq) ; les items d'origine
sont repris dans les étapes ci-dessous (ancien 2.1 -> 2.1 + 2.2, ancien 2.2 -> 2.3, ancien 2.3 -> 2.4, ancien 2.4 -> 2.5,
ancien 2.5 -> 2.6). 2.2, 2.3 et 2.4 sont dans UN commit : un `git mv` sans façade ne compile pas, et D3-O impose le
renommage des clés de ratchet dans le commit qui déplace le fichier.

- [x] 2.1 Squelette : `internal/haloapi/doer.go` (`Doer`, `ErrDefinitif`, `NonRetryable`), `xblauth/doc.go` (contrat),
  `Tokens`, `Descriptor`, `ClientAzure`, provenances, `Chaine` ; parité par réflexion `Tokens` <-> `domain.HaloTokens`
  dans `platform/auth`.
  Statut : fait, commit `7de69eaf3`.
- [x] 2.2 Déplacements `git mv` (relevé du lot 0, D3-L) : voir « Exécution du lot 2 ». Aucune variable de paquet mutable
  dans la brique (le sémaphore XBL, la provenance du `ctx` et les adresses surchargeables des tests restent dans
  `platform/auth`) ; `gochecknoglobals` n'est pas actif, la brique n'a aucune variable de paquet (vérifié au grep).
  Statut : fait, commit `61795015b`.
- [x] 2.3 Façades `platform/auth` aux noms inchangés, conversion des jetons, `Doer` gardé par surface, compteurs tenus
  depuis l'issue typée.
  Statut : fait, commit `61795015b`. 239 symboles exportés d'avant (types, fonctions, valeurs, champs, méthodes)
  vérifiés à la compilation.
- [x] 2.4 Ratchets (D3-O) : netguard (exemption par préfixe + 4 clés renommées), `no_halowaypoint_literal` (1 entrée
  retirée), `briques_internes` (liste `platform/auth` retirée, part « auth » à zéro), `sentinel_test.go` et
  `no_legacy_source_used_test.go` INCHANGÉS et verts ; `.gitleaks.toml` et `scripts/coverage_filter.sh` non concernés.
  Statut : fait, commit `61795015b`.
- [x] 2.5 Témoins : tests (aucun nom disparu, aucun pass->skip), intégration, T3 partiel LIVE (`servicerecord`).
  Statut : fait (voir « Exécution du lot 2 »).
- [x] 2.6 Gate local : build, vet, archlint, `go list -deps` de la brique, gofmt, lint 0 issue, seuils de taille.
  Statut : fait. CI : au pilote (aucun push dans ce lot).

## Exécution du lot 2 (2026-10-29)

Branche `refactor/decouplage-d3-api-halo`, depuis `c1024ef01`. Commits : `7de69eaf3` (2.1), `61795015b` (2.2 à 2.4),
puis le commit de clôture (ce plan, journal). Caches `go-build-d3` / `golangci-d3`. Témoins :
`%TEMP%\claude\c--Users-Guillaume-Downloads-Scripts-LevelUp\77a8b90a-21a6-40a0-ac64-eb152f616862\scratchpad\temoins_d3_lot2\`
(`md5s.txt`). Aucun jeton ni secret dans un fichier, un témoin, un journal ou ce plan.

**Fichiers déplacés (`git mv`) vers `internal/haloapi/xblauth`.** `xsts.go` -> `xsts_rta.go` (moitié protocole :
`XSTSResult`, échange XSTS Xbox Live) ; `oauth_refresh.go` ; `xbox_device_code.go` ; `auth_code.go` ; `oauth_error.go` ;
`api_constants.go` ; `xbox_profile_resolver.go` ; `peoplehub_resolver.go` ; tests `halo_exchange_test.go` ->
`exchange_test.go`, `halo_exchange_descriptor_test.go` -> `exchange_descriptor_test.go`, `xbox_device_code_test.go`,
`auth_code_test.go`, `peoplehub_resolver_test.go`, `xbox_profile_resolver_test.go`. `halo_exchange.go` (515 L) est SCINDÉ :
`exchange_access.go` (chaîne, jambe User Token, préfixes RpsTicket, 197 L) + `exchange_xsts.go` (XSTS, Spartan, Clearance,
`postJSON`, 194 L) ; sa part d'exploitation (sémaphore, provenance du `ctx`, observateur, `ExchangeResult` à
`domain.HaloTokens`) va à la façade `platform/auth/xbl_exchange.go` ; git ne le voit pas comme un renommage (contenu réparti
sur trois fichiers : `git log --follow -M30%` ou `git blame -C -C`). Nouveaux : `sisu.go` (client Xbox natif : identifiant,
scope MSA, deux adresses login.live.com ; la « partie SISU » de `sisu_provider.go`), `xsts_rta_test.go` (10 tests du
protocole coupés de `platform/auth/xsts_test.go`), `testdata_test.go` (`testGamertagInternal` y migre :
`platform/auth/testdata_test.go` supprimé, plus d'usage), `oauth_refresh_test.go` (issue typée, 3 tests neufs). Plus grand
fichier de production de la brique : 227 L (`oauth_refresh.go`) ; plus longue fonction : 74 L
(`OAuth.ExchangeAuthorizationCode`).

**Restés dans `platform/auth`, et pourquoi.** `azure_credentials.go` (lecture d'environnement ; il construit
`ClientAzure` par `versXBL()`, clé du sentinel inchangée) ; `refresh_user_xsts.go` (magasin) ; `sisu_provider.go`
(`SISUProvider`/`sisuDeviceFlow` implémentent `TokenProvider`/`DeviceFlow`/`FlowExchanger` : façade, `SISUDefaultAppID =
xblauth.IDAppXboxNatif`) ; `AcquireXSTSForRTAWithProvenance` (magasin) ; `TokenFamily*` ; `XboxScopes` ; le sémaphore XBL
process-wide (`acquireXBLUserAuthSlot`, passé à la brique par `Chaine.Creneau`) et son test `halo_exchange_throttle_test.go` ;
la provenance portée par le `ctx` (`WithTokenClientFamily`, `TokenFamilyObserver`, `RecordObservedTokenFamily`, passées par
`Chaine.Famille` / `Chaine.Observe`) ; les tests qui lisent le magasin ou l'environnement, ou qui testent la façade
(`xsts_provenance_test`, `oauth_refresh_internal_test` (clé du sentinel inchangée), `azure_credentials_test`, `sisu_*_test`,
`xbox_device_code_netguard_test`, `xbox_device_code_reachability_integration_test`, `oauth_refresh_test`, reste de
`xsts_test`). Façades neuves : `xbl_exchange.go`, `xbl_xsts_rta.go`, `xbl_oauth.go`, `xbl_device_code.go`,
`xbl_resolvers.go`, `xbl_doer.go` ; alias de type pour `XSTSResult`, `XboxDeviceCodeResult`, `AuthCodeResult`,
`OAuthExchangeError`, `AuthErrorClass`, `XboxProfileResolver`, `PeopleHubResolver` ; `ExchangeResult` reste une struct de
`platform/auth` (champ `Tokens *domain.HaloTokens`, 29 lecteurs). Conversion des jetons : sens brique -> `domain` seulement
(`haloTokensDepuisXBL`) : aucun chemin ne passe de jetons Halo À la brique (le sens inverse serait du code mort, règle 7).

**Table des délais (même valeur qu'avant, par fonction de façade).**

| Façade | Doer passé à la brique | Délai | Garde netguard |
|---|---|---|---|
| `ExchangeAccessToken(WithDescriptor)`, `ExchangeXSTSForHaloTokens(WithDescriptor)`, `AcquireXSTSForRTA(WithProvenance)` | `*http.Client` | 20 s (`xblExchangeTimeout`) | aucune (comme avant) |
| `sisuDeviceFlow.ExchangeFlow` | client injecté, sinon `*http.Client` | 20 s | aucune |
| `ExchangeRefreshToken(WithRotation)` (Azure, retry public, repli MSA) | `netguardDoer{"microsoft_oauth.refresh"}` | 15 s (`oauthTimeout`) | oui, à chaque requête |
| `ExchangeAuthorizationCode` | `*http.Client` | 15 s | aucune |
| `StartXboxDeviceCode` / `PollXboxDeviceCode` | `netguardDoer{"xbox_device_code.start"/".poll"}` sur `http.DefaultClient` | aucun | oui |
| `SISUProvider.InitDeviceFlow` (client nil) | `netguardDoer{"xbox_device_code.start"}` | 20 s | oui |
| `NewXboxProfileResolver` / `NewPeopleHubResolver` | client de l'appelant, sinon `http.DefaultClient` | celui de l'appelant | aucune |

Une sortie refusée (`netguard.ErrOffline`) est rendue par le `Doer` marquée `haloapi.NonRetryable` : la brique la rend
telle quelle (rafraîchissement : même erreur, même message qu'avant, aucun repli MSA tenté) ou avec le préfixe d'avant
(device code : « xbox_device_code: start: » / « poll: »).

**Issues typées et compteurs.** `xblauth.RefreshOutcome{AccessToken, RotatedRefreshToken, Famille, RetentePublique,
RetenteNatif}` ; `platform/auth/metrics.go` (`recordOAuthRefreshIssue`) incrémente `retry_public` si `RetentePublique`,
`retry_msa` si `RetenteNatif`, puis `oauth_refresh_total` / `fail_total` / `fail_by_class` comme avant ; refresh_token vide :
aucun appel, aucun compteur (comme avant). Noms expvar inchangés. Identifiant d'événement (`logging.WithEvent`) posé par la
façade dans le `ctx` avant l'appel : les journaux `slog` de la brique le portent ; messages et clés de journal inchangés.

**Symboles exportés vérifiés.** Relevé AVANT (`go/parser` sur `7de69eaf3`) : 239 symboles exportés de `platform/auth`
(32 types, 40 fonctions, 30 valeurs, 81 champs, 56 méthodes) ; un fichier généré qui référence chacun compile sur le HEAD
du lot (`EXPORTS_OK: 239`). `go build ./...` et `go vet ./...` verts : les 55 importeurs ne changent pas.

**Ratchets touchés (une clé pour une, datées 2026-10-29).** `netguard_coverage_test.go` : exemption par préfixe
`haloapi/` (constante `prefixeBriqueHaloapi`) ; `platform/auth/auth_code.go` -> `haloapi/xblauth/auth_code.go`,
`platform/auth/halo_exchange.go` -> `haloapi/xblauth/exchange_xsts.go` (porte `postJSON`), `platform/auth/peoplehub_resolver.go`
-> `haloapi/xblauth/peoplehub_resolver.go`, `platform/auth/xbox_profile_resolver.go` -> `haloapi/xblauth/xbox_profile_resolver.go`
(justifications gardées, « ex ... » ajouté) ; le site `Check` neuf `platform/auth/xbl_doer.go` est VU par le ratchet (il émet
`next.Do(req)` et appelle `netguard.Check`). `no_halowaypoint_literal_test.go` : entrée `internal/platform/auth/halo_exchange.go`
RETIRÉE (la brique ne nomme plus aucun hôte halowaypoint ; les défauts vivent dans `domain/title`, déjà allowlisté) :
l'allowlist décroît. `briques_internes_test.go` : liste `platform/auth` retirée de la ligne `haloapi` ; `domain` 22 -> 21,
`domain/title` 9 -> 3, `platform/netguard` 9 -> 6, `observability/logging` retirée. `sentinel_test.go` (guards 2 et 3 VIDES),
`no_legacy_source_used_test.go` : non modifiés, verts. `.gitleaks.toml` (aucun chemin déplacé), `scripts/coverage_filter.sh`
(absent du dépôt) : non concernés. Mutation : la façade du rafraîchissement privée de son `netguardDoer` rougit
`TestExchangeRefreshToken_DemoOffline_NEmetRien` (« erreur = <nil>, attendu netguard.ErrOffline »), puis rétablie.

**Témoins.** (1) `go test -json ./internal/platform/auth/... ./internal/haloapi/... ./internal/platform/netguard/...
./internal/sync/... ./internal/api/... ./internal/archlint/... -count=1` : tout vert après relance d'un test (le walk de
`TestNoMutationOnAppendOnlyStateTables` a croisé un dossier temporaire de vérification des exports, créé puis supprimé
pendant le run : relancé seul, vert). Comparaison au témoin T1 pour ces paquets (4 530 lignes avant, 4 557 après) : AUCUN
nom disparu, aucun pass->skip ; 53 tests (69 lignes avec sous-tests) passés de `platform/auth` à `haloapi/xblauth` sous le
même nom et le même verdict (68 pass, 1 skip `TestExchangeAuthorizationCode_MockServer_Skipped`, déjà skip) ; nouveaux :
9 tests d'archlint du lot 1, `TestNonRetryable`, `TestClientAzure_*` (2), `TestRefreshWithRotation_*` (3), parités
`platform/auth` (4), `TestNetguardDoer_DemoOffline`, `TestExchangeRefreshToken_DemoOffline_NEmetRien`,
`TestExchangeRefreshToken_CompteursDepuisIssue`. (2) `go test -tags=integration ./internal/platform/auth/...
./internal/haloapi/... -count=1` : vert ; `TestXboxDeviceCodeEndpointReachable` SKIP (garde opt-in
`LEVELUP_DEVICE_ENDPOINT_LIVE_CHECK` non posée, comme avant). (3) T3 partiel LIVE, depuis le worktree avec
`LEVELUP_REPO_ROOT` = dépôt principal, compte JGtm : `diag_api_lecture -xuid <JGtm> servicerecord` (une surface, une rotation
de la chaîne par le chemin canonique, aucune re-capture) : code 0, 5 987 octets, MD5 `4dfdef90e9208e547125db378c08289b`
IDENTIQUE au témoin du lot 1 ; journal identique au lot 1 (« invalid_grant côté Azure — retry refresh MSA natif » puis
« échange OK rotated=true », D3-L1-f) ; fichier de jetons du compte : mêmes 16 clés JSON avant et après (comparaison des
CHEMINS de clés seulement, valeurs jamais lues), `updated_at` avancé (réécrit par la rotation).

**Gate (2.6).** `go build ./...`, `go vet ./...` verts ; `go test ./internal/archlint/ -count=1` vert ; `go list -deps -test
./...` depuis `internal/haloapi` : aucun paquet `levelup/go-api/internal` hors `haloapi` ; `gofmt -l` vide ;
`golangci-lint run --new-from-merge-base=origin/release/v8.1.0 ./...` : 0 issue (2 `goconst` « grant_type » apparus au
déplacement, corrigés par la constante `oauthFieldGrantType`) ; brique : fichiers de production <= 227 L, fonctions <= 74 L.

## Lot 3 — `haloapi/infinite` (services, transport, adresses, DTO bruts)

- [ ] 3.1 Transport : `halo_client_http.go` → `infinite/transport.go` (sans `decfilm.Decompresser` :
  la décompression d'une réponse de film est du protocole HTTP (`Content-Encoding`) ou du décodage ? À
  trancher sur pièces au lot 0 : si c'est l'en-tête HTTP standard, `compress/*` stdlib dans la brique ;
  si c'est le format du film, l'appelant `filmfetch` décompresse).
- [ ] 3.2 Adresses : `infinite/endpoints.go` (clés + défauts + `WithEndpoints`), construit côté LevelUp
  depuis `games.DefaultEndpointResolver` (adaptateur) ; plus de `games.*` dans la brique.
- [ ] 3.3 Un fichier par service, DTO bruts : historique/stats (`haloclient/halo_client.go`,
  `halo.FetchMatchStats`, `FetchRemoteStats`, `FetchServiceRecord`), skill/CSR (`halo_skill*.go` sans
  `analysis.MMRKnown`), carrière (`halo_client_career.go`), économie/apparence
  (`appearance_inputs.go`, `GetSpartanCustomization`), CMS/découverte (`discovery_client.go`,
  `catalog_fetcher.go` sans `AssetFetcher`, `asset_name_fetcher.go`, `FetchAsset`), défis
  (`challenges_details.go`), passe de combat (`battlepass_details.go`), saisons (`season_provider.go`),
  confidentialité (`privacy_provider.go`), classements (`leaderboard_scraper.go`), comparaison
  (`compare_provider.go`), plaques (`spartan_nameplate_*.go`), UGC/variantes (`FetchMvarForMap` : octets
  seulement), films (`GetMatchFilm` manifeste, `GetFilmChunks`/`GetFilmChunkURLs`/`GetHighlightEventsChunk` :
  OCTETS, sans `finalise`), médailles (`medal_provider.go` sans `metadata.MedalEntry`).
- [ ] 3.4 Tests `httptest` déplacés avec leurs fixtures ; tolérances `infinite` à zéro ; `go list -deps
  ./internal/haloapi/...` propre.
- [ ] 3.5 Gate + commit « D3 lot 3 » (le code LevelUp n'est pas encore rebranché : `platform/halo` et
  `haloclient` existent encore, la brique est compilée et testée seule).

## Lot 4 — Exploitation : `filmfetch`, `haloadapter`, `tokencache`, rebranchement, suppression

- [ ] 4.1 `sync/filmfetch` : décodage/finalisation/cache (ex `halo_client_film.go` partie
  exploitation, `local_film_cache.go`) sur les octets de `infinite` ; `FilmChunk`, `LocalFilmCache`,
  `CachedChunk`, `CachedManifest`, `IsFilmGoneErr`, `ServedFromCarry` : mêmes noms exportés ; les
  consommateurs (`sync/replayartifacts`, `sync/killcollector`, `cmd/levelup`, `service`) changent
  d'import seulement.
- [ ] 4.2 `games/halo_infinite/haloadapter` : `Provider` (mêmes méthodes que `HaloProvider`, mêmes
  `With*` de politique), conversions DTO → `domain` (tests de parité par réflexion quand le DTO est le
  miroir du type domaine ; tests valeur à valeur sur les fixtures sinon), `assets.Ref`,
  `metadata.MedalEntry`, `analysis.MMRKnown`, `AssetFetcher`, `mapcatalog` (variante), `ctxkeys`
  (jetons, xuid, slug, langue sortis du `ctx` ICI), `duckdb.PersistSink` (si c'est une écriture :
  reste chez l'appelant qui la faisait).
- [ ] 4.3 `platform/auth/tokencache` (ex `player_token_cache.go`, `auth_retry.go`) ; 7 appelants
  d'`InvalidateCachedPlayerTokens` ; `internal/duration` (ex `platform/halo/duration`, 3 fichiers
  `halo_5` + lecteurs).
- [ ] 4.4 Rebranchement des 15 + 6 importeurs (import et constructeur seulement) ; suppression de
  `platform/halo` et `sync/haloclient` (tests, fixtures, entrées de baseline JSONL par un bloc
  RETRAIT daté pointant les nouveaux paquets, ratchets qui les citent : `no_request_scoped_halo_provider_test`
  (vise désormais `haloadapter.Provider`), `sync_root_freeze_test`, `no_film_reread_test`,
  `no_hardcoded_film_cache_dirs_test`, `no_halowaypoint_literal_test`, `sync/pooled_client_test`,
  `sync/assetnames_wiring_test`).
- [ ] 4.5 Gate : `go build ./...` ; T1 (mêmes tests modulo paquet, 0 fail) ; archlint vert ;
  `-tags=integration -p 1 ./internal/sync/...` vert ; lint 0 issue neuve ; `/debug/vars` : noms de
  compteurs inchangés (grep des constantes). Commit « D3 lot 4 » ; CI verte ; point d'étape.

## Lot 5 — Revue adversariale (une ronde, trois relecteurs)

- [ ] 5.1 Contrat ; relecteurs : (A) `xblauth` + `platform/auth` : protocole vs politique, ADR 0023/0038,
  sentinelles, netguard, rotation du RT (lentille L3 + lentille auth) ; (B) idem auth, lentille L6 ;
  (C) `infinite` + `haloadapter` + `filmfetch` : parité DTO → domaine, 429/`Retry-After`, erreurs
  typées, octets des films, cache local.
- [ ] 5.2 Triage P0–P3 ; corrections P0/P1 avec test ; P2 consignés ; relecture des seules corrections
  s'il y a eu P0/P1.
- [ ] 5.3 Gate lot 4 rejoué ; commit « D3 lot 5 » ; CI verte.

## Lot 6 — Preuve et clôture

- [ ] 6.1 T3 LIVE rejoué (mêmes commandes qu'au lot 0) : JSON bruts et valeurs de domaine identiques
  (hors horodatages/jetons) ; T1 ; T4.
- [ ] 6.2 Docs (D3-I) ; registre des reports ; addendum « D3 exécuté » dans l'audit (clôture des trois
  axes) ; thought_log ; mémoire.
- [ ] 6.3 Caches `-Lot d3` ; PR, CI verte, fusion squash, branche et worktree supprimés, dépôt
  principal remis au niveau.

## Témoins avant/après

| Témoin | Capture | Comparaison |
|---|---|---|
| T1 tests Go | `go test -json ./... -count=1` → (paquet, test, verdict) | mêmes noms et verdicts modulo paquet, 0 fail |
| T2 fixtures HTTP | tests `httptest` des deux clients (base) | mêmes assertions, mêmes valeurs dans les nouveaux paquets |
| T3 LIVE lecture seule | une commande par surface (historique, stats, skill, manifeste, actif, calendrier), jetons/horodatages masqués, MD5 | identiques |
| T4 compteurs | constantes de compteurs des trois paquets | noms inchangés |
| ratchets auth | `sentinel_test`, `no_legacy_source_used_test`, netguard, pool | verts SANS modification d'allowlist |

## Protocole de reprise de session

1. `git branch --show-current` ; `git log --oneline -10` dans `../LevelUp-wt-decouplage-d3`.
2. Premier item sans statut du premier lot non clos = point de reprise.
3. Vérifier sur pièces avant de coder.
4. Journal : `.ai/thought_log.md`, entrées « Découplage D3 — lot N ».

## Découvertes (notées, non traitées)

Lot 0 (2026-10-29). Notées, NON traitées ; le lot indiqué décide.

- **D3-L0-a (T3 lancé depuis le worktree, pas depuis le dépôt principal).** Le brief demandait les commandes LIVE depuis le
  dépôt principal. Elles ont été lancées depuis `apps/go-api` du worktree avec
  `LEVELUP_REPO_ROOT='C:\Users\Guillaume\Downloads\Scripts\LevelUp'` (`config.Load` lit alors `db_profiles.json`, `.env.local` et
  `data/` du dépôt principal ; même commit `b92916328`). Raison : au lot 6 le code testé vivra dans le worktree ; l'AVANT
  doit se lancer de la même façon. Lot 6 : même mode de lancement.
- **D3-L0-b (surfaces sans commande de témoin existante).** Aucune commande existante en lecture seule ne couvre : le
  skill d'un match (c), un actif de catalogue (e), le calendrier de saison / le service record / la carrière (f) ; l'historique
  (a) n'est couvert que par le `match_id` du dernier match. Cause : `levelup populate-assets` ouvre `metadata.duckdb` en
  écriture même en `--dry-run` ; `refresh-metadata` écrit `metadata.duckdb` ; `diag_live_economy` migre la base joueur en
  écriture (échec constaté, tenue par `server.exe`, code 1, avant toute authentification) ; `get-token` imprime les jetons en
  clair. Lot 1 ou 6 : décider entre un outil de témoin jetable (ajouté à `cmd/`, donc à porter par le plan) ou des fixtures
  `httptest` enregistrées (T2) pour ces surfaces.
- **D3-L0-c (CI de la base).** Au début du relevé, le run CI du HEAD `b92916328` (#113) était `in_progress` ; il s'est
  terminé vert (13 min 53 s) avant la fin du lot. Rien à faire ; noté parce que le critère 0.1 (« CI de la base verte ») n'était pas
  tenu à l'instant du premier relevé.
- **D3-L0-d (la coupe de `platform/auth` n'est pas « nette », audit D3.5 et plan D3-B).** Sur les 12 fichiers listés comme
  protocole : `refresh_user_xsts.go` est de l'EXPLOITATION (il lit et écrit `MultiUserTokenStore`, `persistRotatedRT`) ;
  `xsts.go` est MIXTE (`AcquireXSTSForRTAWithProvenance` prend le magasin) ; `azure_credentials.go` LIT L'ENVIRONNEMENT
  (`LEVELUP_OAUTH_CLIENT_ID`, `SPNKR_AZURE_CLIENT_SECRET`), ce qui contredit « la brique n'a AUCUNE lecture d'env » ;
  `auth_code.go` et `oauth_refresh.go` appellent `ResolveAzureOAuthClient()` en interne ; `sisu_provider.go` implémente
  `TokenProvider`/`DeviceFlow`/`FlowExchanger` (interfaces d'exploitation de `provider.go`) et `TokenFamily*` vit dans le
  fichier du magasin ; `oauth_refresh.go` incrémente des compteurs expvar de `metrics.go` ; `halo_exchange.go` fait 515 L
  (> 500) ; chaque fonction construit son propre `&http.Client{Timeout: 15 s | 20 s | xblExchangeTimeout}` ; `XboxScopes` et
  `xboxScopes` sont des variables de paquet initialisées par `title.DefaultHaloAuthDescriptor()`. Conséquence : le lot 2 est plus
  large que « déplacer 12 fichiers » (injection du client Azure, des crochets de comptage, des familles de jeton, des
  descripteurs ; interfaces `TokenProvider` & co. déclarées côté `platform/auth` et satisfaites par une façade). Le tableau du
  relevé fait foi pour le lot 2.
- **D3-L0-e (ratchets à clés de CHEMIN : « sans modification d'allowlist » est inapplicable).** Le plan (succès (3), tableau
  « ratchets auth », 2.3) exige des allowlists inchangées. Or plusieurs sont des tables de chemins qui échouent à la
  moindre absence : `sentinel_test.go` `allowedClientSecretReaders` (3 chemins `azure_credentials*.go`,
  `oauth_refresh_internal_test.go`) et `TestSentinel_AllowlistEntriesPointToExistingFiles` ; `netguard_coverage_test.go`
  `allowlist` (4 fichiers de `platform/auth`) et `TestAllowlistHasNoStaleEntry` ; `no_halowaypoint_literal_test.go`
  `halowaypointAllowlist` (une dizaine de chemins de `haloclient/`, `platform/halo/`, `platform/auth/`) ;
  `auth_retry_guard_test.go` (noms de fichiers `provider.go`, `season_provider.go`, `compare_provider.go`...) ;
  `.gitleaks.toml:172` (chemin de la fixture HTML du classement) ; `scripts/coverage_filter.sh` (ligne
  `internal/platform/halo/`). Critère à reformuler : « aucun ajout, aucun élargissement ; renommage de clés daté,
  une pour une ». Les allowlists des guards 2 et 3 du sentinel restent VIDES (aucun chemin à renommer).
- **D3-L0-f (rotations de jeton pendant que le serveur tourne).** Quatre lancements d'outils = quatre rotations de la chaîne
  de JGtm en 48 s (voir `commandes_T3.md`), alors que le serveur local (`server.exe`) tournait ; ces outils rechargent le
  magasin à chaque lancement mais le cache de 50 minutes du serveur n'est pas invalidé (processus différent). Aucun
  dommage constaté pendant le lot ; à surveiller si le serveur signale `AADSTS70000`/rotation perdue sur JGtm (ADR 0023 :
  diagnostiquer, ne JAMAIS re-capturer). Lot 6 : prévoir les mêmes quatre rotations.
- **D3-L0-g (trois boucles de reprise HTTP aux sémantiques différentes ; deux `FetchMatchStats`).** `provider.doGet` : 3 essais,
  429 retenté comme tout non-200, erreurs non typées, sentinelle `errHaloAuthFailure`. `discovery_client.doGetWithLang` : 5xx
  retenté, 429 NON retenté, pas de `Retry-After`. `haloclient.doGet` : 4 essais, 429 rendu immédiatement avec `RetryAfter`,
  `HTTPError` typé (consommé par le pool de `sync`). Les deux `FetchMatchStats` visent des hôtes différents
  (`halostats.svc.halowaypoint.com` sans `:443` côté provider, avec `:443` côté `haloclient`; PMT-1 documente la
  différence « byte-distincte »). « UN client » (D3-C) change donc le comportement d'au moins un chemin : décision à prendre
  au lot 3 (transport à stratégie de reprise injectée, ou comportements conservés par service), avant d'écrire `infinite`.
- **D3-L0-h (la garde `netguard` par `RoundTripper` demande plus qu'un client injecté).** (1) Le ratchet est textuel et par
  fichier : les fichiers de la brique contiendront `.Do(req)` sans `netguard.Check` -> faux rouge ; règle neuve au 1.4 (par ex.
  « tout `*http.Client` de `haloapi` est reçu, jamais construit » + test de comportement par façade). (2) Aujourd'hui `Check`
  est posé AVANT la boucle de reprise ; un `RoundTripper` qui rend `ErrOffline` est vu comme une erreur de transport RETENTÉE
  (3-4 essais avec attente = les ~12 s par appel de l'incident qui a motivé `netguard`) : `infinite` doit traiter
  `netguard.ErrOffline` (ou une erreur typée « non retentable ») comme définitive. (3) Le nom de surface (11 noms) vient du
  site d'appel ; un `RoundTripper` ne le connaît pas : le conserver par un client par service ou une valeur de contexte, sinon
  le log « une fois par surface » change. (4) Les marqueurs du ratchet ne couvrent que les variables nommées `req` ou `httpReq` :
  `client.Do(r)` lui échappe. (5) `netguard_demo_test` et `xbox_device_code_netguard_test` passent un client par défaut ou
  arbitraire : ils deviennent des tests de façade.
- **D3-L0-i (état de paquet à ne pas faire entrer dans une brique « sans état »).** `haloclient` : `retryBaseDelay`
  (variable mutable pour les tests), cache global du mapping d'emblèmes (`emblemMapping*`, TTL 6 h), GET du mapping par
  `http.DefaultClient` (hors client injecté). `platform/halo` : `DefaultHaloProvider` (singleton), `challengesFetchSFGroup`,
  `playerTokenStore`/`playerTokenRefresher`, cache de confidentialité. `platform/auth` : `XboxScopes`, `xboxScopes`. Tout
  cela va à l'adaptateur / `tokencache` / est passé en paramètre ; le GET du mapping devra passer par le client injecté
  (changement de délai : aujourd'hui 8 s de contexte, aucun délai de client).
- **D3-L0-j (le funnel `sync/haloclient_reexport.go`).** 36 L d'alias (`HaloAPIClient`, `HaloClient`, `FilmChunk*`,
  `MatchSkillData`, `LocalFilmCache`, `HTTPError`, `NewHaloAPIClient`, `IsAuthError`...) : 29 fichiers prod de 21 paquets
  (`api/wire`, `sync/v2`, `service`, `watcher`, `worldenrich`, `games/halo_5/livesync`, 15 `cmd/`) consomment le client par
  `sync.X`, non comptés dans « 6 importeurs ». Le lot 4.4 ne « rebranche » pas 6 importeurs : il re-pointe ce fichier
  vers `haloadapter`/`filmfetch` (les alias gardent les noms), plus les 6 imports directs. `CareerRankData` est défini deux
  fois (`sync/career.go`, `haloclient/dto_aliases.go`).
- **D3-L0-k (autres clients HTTP Halo hors des trois paquets).** `internal/mapcatalog/ugc.go` (client Discovery UGC,
  appelé par `haloclient.FetchMvarForMap` : troisième client UGC, avec `platform/halo` `FetchAsset` et `haloclient.GetPlaylistConfig`),
  `internal/assets/fetcher_gamecms.go` (GameCMS, lit aussi le singleton d'adresses), `internal/games/halo_5/client.go` (hors D3 par
  D3-F), `internal/sync/xbox_client.go` (Xbox, hors Halo). D3-C promet « UN client » : périmètre à confirmer (`mapcatalog` et
  `assets` sortent-ils ou restent-ils ?) avant le lot 3.
- **D3-L0-l (écarts de chiffres, tableau du 0.4).** Importeurs de `platform/auth` : 40 `cmd/` et non 44 ; `games/` lisant le
  singleton : huit et non sept ; importeurs de `duration` : `openspartan/mapper` non nommé ; `duckdb.PersistSink` = commentaire
  (voir 0.5 (c)) ; symboles `auth.X` : 42/32/27/16 contre 41/30/29/20. Sans conséquence sur le découpage.
- **D3-L0-m (incohérences entre le plan et le relevé).** 3.3 range `asset_name_fetcher.go`, `catalog_fetcher.go`,
  `challenges_details.go`, `battlepass_details.go` dans `infinite` : le relevé montre que ce sont des fichiers
  d'EXPLOITATION (aucun HTTP pour les deux derniers ; adaptation vers `assetnames` / `halo_infinite.AssetFetcher` pour les
  deux premiers) -> `haloadapter`. 4.1 range `ServedFromCarry` dans `filmfetch` : il est défini dans
  `spartan_nameplate_diagnosis.go` (apparence, `haloadapter`). 3.3 place `FetchMvarForMap` dans `infinite` : le fichier appelle
  `mapcatalog` (exploitation) -> `haloadapter`.
- **D3-L0-n (ratchet des octets de film).** `archlint/no_raw_film_bytes_outside_source_test.go` surveille
  `libs/film`, `internal/analysis`, `internal/sync`, `internal/replaybuild`, `cmd` : `internal/haloapi/infinite` rendra des octets
  de film (compressés) hors de ces racines, donc invisibles du ratchet. À ajouter à `racinesOctetsBruts` dans le même commit
  que la création du paquet (lot 1 ou 3), et `internal/sync/filmfetch` y est déjà (sous `internal/sync`).
- **D3-L0-o (tiers permis pour `haloapi`, 1.2).** Mesure : `golang.org/x/time/rate` (`HaloAPIClient`, `testexports.go`) et
  `golang.org/x/sync/errgroup` (téléchargement parallèle) ; `singleflight` va à l'adaptateur / `tokencache` ; `x/net/html` n'est
  pas nécessaire (le scraper n'utilise qu'une expression régulière et `encoding/json`). Table du 1.2 : `x/time/rate` et
  `x/sync/errgroup` au plus ; ou injecter un limiteur d'interface et ne garder que `errgroup`.
- **D3-L0-p (anti-motifs rencontrés, non traités).** `medal_provider.go` : `rowJSON, _ := json.Marshal(m)` (erreur avalée) ;
  `battlepass_details.go` : persisters lancés en goroutine sur `context.Background()` ; `discovery_client.go` : `slog.Debug` sans
  contexte (2 sites) ; `leaderboard_scraper.go` : `FrenchName()` compare à « fr-FR » en dur (ADR 0040, garde-rails de
  littéral de langue) ; `asset_name_fetcher.go` : `os.Getenv` dans `platform/halo` ; `testexports.go` : seams de test dans un
  fichier de production.
- **D3-L0-q (secret affiché par mégarde, signalé à l'utilisateur).** En listant les noms de variables de `.env.local`
  (`cut -d= -f1`), la sortie a montré des lignes de COMMENTAIRE de ce fichier qui contiennent en clair un secret client Azure
  (ligne « Value : ... » en tête du fichier, d'une ancienne application). Rien n'a été écrit dans un fichier, un témoin, ce plan ni
  un journal ; la valeur n'est reproduite nulle part. À supprimer du fichier et, si l'application existe encore, à renouveler
  côté Azure (décision de l'utilisateur).

Lot 1 (2026-10-29). Notées, NON traitées.

- **D3-L1-a (le calendrier de saisons répond 403).** `GET gamecms-hacs.svc.halowaypoint.com/hi/Progression/file/SeasonCalendar.json`
  avec des jetons fraîchement rafraîchis (les autres lectures passent avec les mêmes) : HTTP 403. Le filet 401/403 du
  provider tente un re-mint, sans rafraîchisseur câblé hors serveur (`halo: aucun refresher token câblé`). Le seul appelant
  hors serveur, `cmd/refresh-metadata seasons`, appelle avec un `context.Background()` sans jetons (`tokens absents du contexte`) : il ne
  peut pas réussir seul. Appelant de production : `service/seasons_catalog.go`. Cause non investiguée (consigne : ne pas insister) ;
  à reprendre si la surface sert un jour de preuve. Lot 6 : l'échec attendu est identique.
- **D3-L1-b (netguard : la règle du 1.4 n'exempte pas encore les fichiers de la brique).** `TestOutboundCallsAreNetguarded` reste
  inchangé : un futur fichier de `internal/haloapi/` contenant `.Do(req)` sans la chaîne `netguard.Check` le fera rougir (D3-L0-h(1)).
  Le contrepoids (la brique ne possède aucun client HTTP, donc tout passe par le `Doer` gardé) est posé et prouvé ; l'exemption par
  préfixe `haloapi/` dans le ratchet de couverture (et dans `TestAllowlistHasNoStaleEntry`) est à poser au lot 2 ou 3, dans le commit
  qui met le premier fichier concerné sous `haloapi/`, avec le renommage une pour une des quatre clés `platform/auth/*` (D3-L0-e).
  Le test neuf vit dans `archlint` et non dans `netguard_coverage_test.go` : il lit l'AST, réutilise `parseGoSnippet` et le
  parcours de l'espace de travail, et ne dépend pas du paquet qu'il surveille ; un commentaire en tête du ratchet de couverture y renvoie.
- **D3-L1-c (les témoins sont des valeurs, pas des octets, pour cinq surfaces).** Historique, stats, skill, actif et manifeste
  sont des sérialisations de types Go. La parité des DTO bruts de `infinite` (lot 3) n'est donc prouvée par le LIVE qu'au niveau des
  valeurs que l'adaptateur en tire ; la parité champ à champ reste portée par les tests `httptest`/fixtures et par la réflexion
  prévue en 4.2. Seul `calendrier` (en échec) donnait des octets.
- **D3-L1-d (`racinesOctetsBruts` échoue sur une racine absente).** Le ratchet des octets de film ne tolère pas un dossier
  inexistant ; d'où `internal/haloapi/doc.go` dès le lot 1 (paquet `haloapi` sans code, contrat de la brique en 10 lignes). Il
  se supprime avec la ligne de la table si D3 est abandonné.
- **D3-L1-e (`x/sync/singleflight` hors liste).** Deux fichiers de `platform/halo` l'importent (`player_token_cache.go`,
  `provider.go`) ; la décision D3-P les envoie à `tokencache` / `haloadapter`. Tolérance datée, lot 4.
- **D3-L1-f (chaque exécution loggue `invalid_grant côté Azure — retry refresh MSA natif`).** Même constat qu'au lot 0 : le client
  Azure public échoue puis le chemin natif MSA réussit (`rotated=true`). ADR 0023 : `AADSTS70000` = ancienne application, pas une
  raison de re-capturer ; rien à faire dans D3, mais le bruit sera dans les journaux des témoins du lot 6.
- **D3-L1-g (`tolerances` agrège les trois origines).** Une entrée porte le total des trois groupes (auth, halo, haloclient) :
  au lot 2 elle baisse de la seule part d'`auth`, au lot 4 le reste part avec la suppression des dossiers. Un lot ne peut pas
  « emprunter » la baisse d'un autre tant que les groupes sont vidés dans l'ordre 2 puis 4 ; sinon la table devra compter par groupe.

Lot 2 (2026-10-29). Notées, NON traitées.

- **D3-L2-a (test placeholder désormais activable).** `TestExchangeAuthorizationCode_MockServer_Skipped` (déplacé dans
  `haloapi/xblauth`) se saute parce que « msalTokenURL n'est pas paramétrable » : avec `OAuth.URLAzure` et le `Doer` reçus,
  il peut devenir un vrai test `httptest`. Laissé en skip (même verdict qu'avant, aucun fix hors périmètre).
- **D3-L2-b (historique de `halo_exchange.go`).** Scindé en trois (`exchange_access.go`, `exchange_xsts.go`, façade
  `platform/auth/xbl_exchange.go`), il n'est pas vu comme un renommage par git (même `-M30%`) ; `oauth_refresh.go` (30 %) et
  `xbox_device_code.go` (46 %) ne le sont qu'en dessous du seuil par défaut. Remonter l'historique : `git log --follow -M30%`,
  `git blame -C -C`.
- **D3-L2-c (surfaces d'authentification toujours sans garde `netguard`).** Échange XBL/XSTS/Spartan/Clearance, XSTS RTA,
  code d'autorisation, PeopleHub, profil Xbox : client nu, comme avant (allowlist, clés renommées). Avec le `Doer`, les garder
  ne coûterait qu'une ligne de façade chacun ; c'est un changement de comportement en mode démo : décision à prendre.
- **D3-L2-d (texte de justification du sentinel).** `sentinel_test.go`, guard 5 : la justification de
  `azure_credentials.go` dit « auth_code.go / oauth_refresh.go délèguent désormais à ResolveAzureOAuthClient » ; ces deux
  fichiers sont dans la brique et ne lisent plus rien, c'est la façade `xbl_oauth.go` qui délègue. Clé et comportement exacts ;
  texte périmé non touché (consigne : ne pas modifier le sentinel). Même historique dans l'en-tête de `azure_credentials.go`.
- **D3-L2-e (erreur avalée préexistante).** `xblauth/xbox_device_code.go`, polling : `raw, _ := io.ReadAll(resp.Body)`
  (erreur de lecture avalée, l'erreur JSON qui suit la masque) ; recopié tel quel.
- **D3-L2-f (champs du descripteur non lus par la chaîne).** `title.AuthDescriptor.XboxLiveRelyingParty`, `SISUAppID`,
  `SISUTitleID` : la chaîne ne les lisait pas avant le déplacement et ne les lit pas après (constantes `xboxLiveRelyingParty`
  et `IDAppXboxNatif` dans la brique, `SISUDefaultTitleID` dans `platform/auth`). `xblauth.Descriptor` ne porte que ce que la
  chaîne lit ; la double source est préexistante.
- **D3-L2-g (mécanisme des dossiers à liste de `briques_internes`).** Plus aucune ligne de la table n'utilise `fichiers` ;
  le mécanisme reste exercé par ses auto-preuves (`TestBriquesInternes_AutoPreuve_DossierAbsent`) et part avec la table à la
  clôture de D3 (comme prévu au lot 1). La constante `dossierPlatAuth` ne sert plus qu'à cette auto-preuve.
- **D3-L2-h (tests réseau de `platform/auth`).** `xsts_provenance_test` attend l'ÉCHEC d'un appel réel à Xbox Live (aucun
  `Doer` injecté par `AcquireXSTSForRTAWithProvenance`) : la durée du paquet varie de 4 à 18 s selon le réseau.
  Préexistant ; avec la brique, la façade pourrait accepter un `Doer` de test.
- **D3-L2-i (texte d'`oauth_error.go`).** L'en-tête dit « refresh_token révoqué (action user : re-capture) » ; l'ADR 0023
  parle de ré-authentification interactive et interdit la re-capture pour « réparer ». Texte déplacé tel quel.
