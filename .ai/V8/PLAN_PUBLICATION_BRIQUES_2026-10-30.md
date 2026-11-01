# Plan — Publication des briques : module `libs/haloapi` multi-jeu, README et tags, dépôts séparés sur critère — 2026-10-30

Source : décision de l'utilisateur du 2026-10-30 (sortir l'API Halo et le générateur de cartes, garder le
décodeur de film dans le dépôt), entrée `[archi]` de `.ai/BACKLOG.md`, audit
`.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (clos : D1, D2, D3), ADR 0041 (modules sous `libs/`, D4 : tag
`libs/<brique>/vX.Y.Z`), ADR 0042 (protocole vs exploitation). Exécution sous le contrat du skill
`plan-execution` : ordre strict, un lot à la fois, gate passé avant le lot suivant, items statués
`[x]` / `[~]` / `[!]`, zéro fix hors périmètre (section « Découvertes »). Pilotage par agents : Sonnet par
défaut, Opus pour les lots 2 et 3 (déplacements de clients HTTP avec parité), Haiku pour les
micro-tâches de documentation contrôlables.

Lancement prévu : dimanche 2026-11-02 (après le renouvellement du quota hebdomadaire). Les mesures de ce
plan datent du 2026-10-30 (`release/v8.1.0` à `799b1f648`) : les ROUVRIR au lot 0.

## Objectif et critère de succès

Trois briques publiables depuis n'importe quel dépôt Go par `go get`, sans séparation de dépôt :

| Brique | Module | Contenu à la clôture |
|---|---|---|
| fichiers du jeu | `libs/gamefiles` (existe) | `himodule`, `ooz` (GPLv3), `hinavmesh`, `mapvar` + README |
| images de carte | `libs/himap` (existe) | `himap` + README (« générateur de cartes ») |
| API Halo | `libs/haloapi` (À CRÉER depuis `apps/go-api/internal/haloapi`) | `xblauth`, `infinite`, `halo5` (nouveau : le client Halo 5), `xboxlive` (nouveau : client Xbox), les clients UGC et GameCMS repliés dans `infinite` + README |

Succès = (1) `go list -deps -test ./...` dans `libs/haloapi` sans `levelup/go-api` ; (2) les quatre clients
HTTP Halo/Xbox restés hors brique après D3 (`games/halo_5/client.go`, `mapcatalog/ugc.go`,
`assets/fetcher_gamecms.go`, `sync/xbox_client.go`) n'existent plus comme clients : leur protocole est dans
`libs/haloapi`, leur exploitation dans LevelUp ; (3) aucun changement de comportement observable (mêmes
URL octet pour octet pour des identifiants sains, mêmes politiques de reprise, mêmes surfaces `netguard`,
mêmes valeurs de domaine : témoins avant/après) ; (4) un README par brique (EN, la langue des ADR ; une
page : ce que c'est, ce que ce n'est pas, exemple de 20 lignes, licence, comment contribuer) et un tag
`libs/<brique>/v0.1.0` sur les trois, posés par l'utilisateur (geste de publication) ; (5) CI verte, ratchets
`briques_imports_test` (ligne `haloapi`) et `briques_internes_test` (vidée) cohérents ; (6) la décision « dépôt
séparé » reste OUVERTE, avec son critère écrit (lot 4), pas prise ici.

Effort estimé : 3 jours d'agent (lot 2 = le gros : quatre clients à couper en protocole / exploitation,
avec la même méthode que D3).

## Branches

- Lots 0 à 4 : `refactor/publication-briques` depuis `release/v8.1.0` (après fusion des PR « harnais » et
  « suites du découplage » du 2026-10-30), worktree dédié `../LevelUp-wt-publication`, un commit par lot,
  fusion squash par PR dans la release. Jamais `main`.
- Caches : `GOCACHE=%LOCALAPPDATA%\go-build-pub`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-pub`.
- Autorisation : à redemander au lancement (le feu vert du 2026-10-28 couvrait D1 à D3).

## Chiffres de départ (2026-10-30, à rouvrir au lot 0)

- `internal/haloapi` : 4 103 L prod (`doer.go`, `xblauth/`, `infinite/`), tiers `golang.org/x/sync/errgroup`,
  `golang.org/x/time/rate` ; importeurs : `games/halo_infinite/haloadapter`, `platform/auth`, `sync/filmfetch`
  (+ `cmd/diag_api_lecture` par l'adaptateur).
- Clients HTTP hors brique (lignes, imports internes) : `games/halo_5/client.go` (462 L ; `platform/netguard`
  seulement — le plus net), `mapcatalog/ugc.go` (185 L ; `domain`, `netguard`), `assets/fetcher_gamecms.go`
  (356 L ; `ctxkeys`, `domain`, `games` (singleton d'adresses), `netguard`), `sync/xbox_client.go` (271 L ;
  `platform/auth`, `netguard`).
- Aucun README sous `libs/`, aucun tag `libs/...`.
- Singleton `games.DefaultEndpointResolver` : lu par huit fichiers de `games/`, `assets/fetcher_gamecms.go`,
  `haloadapter/endpoints.go` (D3-F : il reste côté exploitation ; la brique reçoit ses adresses).

## Relevé du lot 0 (2026-11-01, HEAD `8f0b512c58`) — FAIT FOI pour les coupes du lot 2

Relevé complet symbole par symbole, URL, en-têtes, reprises, appelants :
`.ai/V8/RELEVE_PUBLICATION_BRIQUES_LOT0_2026-11-01.md` (agent Sonnet, lecture seule ; points clés vérifiés
sur pièces par le pilote). Les chiffres de départ du 2026-10-30 (tailles, imports internes) sont exacts.

| Client | Protocole / exploitation | Reprise actuelle | Surface netguard | Destination | Appelants prod | Tests HTTP existants |
|---|---|---|---|---|---|---|
| `games/halo_5/client.go` (462 L + DTO `dto.go` 154 L, `dto_carnage.go` 103 L, `events_dto.go` 65 L) | ~85 % protocole (URL `/h5/...` + `?auth=st`, en-têtes `X-343-Authorization-Spartan`, `User-Agent: cpprestsdk/2.4.0`, pas de clearance, reprise, `HTTPError`) ; exploitation = `TitleSlug` (~90 usages, RESTE), `NewClient` (client HTTP 20 s, limiteur 10 rps), jeton stocké | 4 essais, tout non-200 hors 401/403/404/410 retenté (429 compris), `Retry-After` secondes plafonné 10 s, limiteur avant chaque essai, garde APRÈS le limiteur ; = aucune des 3 politiques d'`infinite`, exprimable en `Retry` (Retente toujours, HonoreRetryAfter, AttenteFinale, erreur du limiteur rendue) | `halo5_api.get`, `halo5_api.get_binary` | `haloapi/halo5` (client + DTO exportés, `h5MatchEventsResponse` à exporter) ; adaptateur dans `games/halo_5` (Doer gardé, limiteur, alias `HTTPError`) | `games/halo_5/adapter_data.go:76` (seul `NewClient`), sources `capture.go` → `wire/server_titles_additional.go`, `cmd/server`, 9 CLI `h5-*` | 8 tests (`client_test.go`, `client_appearance_test.go`) ; aucun test de reprise ni de `GetMatchCarnage`/`GetMatchEvents` |
| `mapcatalog/ugc.go` (185 L) | ~55 % protocole (URL `{discovery}/hi/Maps/{id}[/versions/{v}]`, en-têtes Spartan/clearance/`Origin`/`User-Agent` LevelUp, GET blob sans jeton, lecture bornée 64 MiB) ; exploitation = `MvarPaths`, validation `Files` vide, logs, client HTTP 30 s | aucune (un GET) | `mapcatalog_ugc.get` (deux hôtes) | service UGC-cartes dans `haloapi/infinite` : PAS un doublon de `FetchAsset` (`Maps` majuscule, préfixe `hi` en dur, version optionnelle, DTO différent, `Origin`+UA, blob sans jeton) ; `mapcatalog` garde façade, choix et logs | `haloadapter/client_mvar.go`, `cmd/mapobj-build`, `cmd/mapcallouts-build` (blob seul) | AUCUN |
| `assets/fetcher_gamecms.go` (356 L) | ~35 % protocole (URL `Progression/file/...`, `waypoint/file/...`, `buildGameCMSImageFetchURL`, `doGet`) ; exploitation = hôte par titre via singleton + `ctxkeys`, dispatch `Kind`, `Payload`, `Err*`, `TokenProvider` ; `defaultGameCMSBase`, `gamecmsHostFor/PrefixFor` partagés avec `fetcher_chain.go` | aucune ; `http.DefaultClient` sans délai en prod | `gamecms_assets.get` | service GameCMS-actifs dans `infinite` (GET binaire + JSON SANS `Accept` ni UA : le transport `get` d'`infinite` impose `Accept: application/json` et une reprise, il ne convient pas tel quel) ; `assets` garde résolution, cache, erreurs ; aucun recouvrement d'URL avec `infinite/gamecms.go` | `assets/wire.go:72`, `cmd/refresh-career-ranks`, `scripts/warm_bp_assets` | 18 tests `fetcher_gamecms_test.go` + `demo_offline_no_persist_test.go` |
| `sync/xbox_client.go` (271 L) | ~60 % protocole (URL `achievements.xboxlive.com/users/xuid({xuid})/achievements?skipItems&titleId`, en-têtes `Authorization` XBL3.0, `x-xbl-contract-version: 2`, `Accept-Language`, `Accept`, pagination par `continuationToken`, DTO `xboxAchievementItem`) ; exploitation = `PlayerAchievementRaw`, `parseAchievementItem`, interface `XboxAchievementsClient`, client HTTP 30 s | aucune (un GET par page, corps entier dans l'erreur) | `xbox_achievements.get` | `haloapi/xboxlive` (SUCCÈS Xbox seulement : le plan disait « profils, présence », c'est faux) ; l'en-tête d'autorisation en `string` par appel (`XSTSResult.AuthHeader()` est déjà dans `xblauth`) ; `sync` garde conversion et interface | `sync/engine_postsync_csr.go:197` | AUCUN du chemin HTTP (6 tests de `sync` construisent `xboxAchievementItem` en littéral) |

Conséquences pour le plan (amendements, pas de nouvelle décision produit) :
- P-B UGC et GameCMS : « rejoint `FetchAsset` » devient « nouveau service à côté » ; `infinite` reçoit un chemin GET
  qui n'impose ni `Accept` ni reprise pour servir des octets (les en-têtes produits restent ceux d'aujourd'hui, P-C).
- P-B Xbox : sous-paquet `xboxlive` = succès (achievements v2) ; rien d'autre n'existe dans ce fichier.
- P-A : le tiers `golang.org/x/time/rate` doit être permis sous `haloapi/halo5` (`tiersHaloapi`) ou remplacé par
  l'interface `Limiter` ; `argumentsDoerDesConstructeurs` (`haloapi_doer_garde_test`) reçoit les constructeurs
  `halo5.New` et `xboxlive.New` ; `no_halowaypoint_literal_test` : 3 entrées d'allowlist à déplacer avec les fichiers.
- T4 : aucun des quatre fichiers ne déclare de compteur expvar ; le témoin T4 devient « les cinq surfaces
  netguard » (`halo5_api.get`, `halo5_api.get_binary`, `mapcatalog_ugc.get`, `gamecms_assets.get`,
  `xbox_achievements.get`) : mêmes noms, consultées aux mêmes appels.
- T2 « URL figées » couvre, par client : chaque URL du relevé (§1.E, 2.E, 3.E, 4.E) pour des identifiants sains,
  chaque en-tête posé ET l'absence des en-têtes non posés (GameCMS : ni `Accept` ni `User-Agent` ; Halo 5 : pas de
  `343-clearance`), l'échappement tel qu'il est (`xuid(...)` non échappé, `relPath` non échappé), la pagination Xbox
  (ordre `skipItems` puis `titleId`), la redirection 302 de Halo 5 avec le jeton réémis.
- T3 : sous-commandes nouvelles de `diag_api_lecture` : `halo5-servicerecord` (compte Halo 5 suivi), `ugc-carte
  <asset_id>` (DTO Discovery de la carte, sans `.mvar`), `gamecms-actif <chemin>` (octets d'un actif public :
  longueur + SHA-256, pas les octets), `xbox-succes` (première page, DTO brut).

## Décisions tranchées avant exécution (pilote, 2026-10-30 ; confirmées par le go de l'utilisateur du 2026-11-01)

- **P-A — Un module `libs/haloapi`**, chemin `github.com/JGtm/LevelUp/libs/haloapi`, paquets au premier
  niveau : `xblauth`, `infinite`, `halo5`, `xboxlive`, plus `doer.go` à la racine (paquet `haloapi` :
  `Doer`, `ErrDefinitif`, `Garde`). Rang : n'importe aucune autre brique. `go.mod` à la main (ADR 0041 D2 ;
  `go mod tidy` possible ICI puisque le module n'importe pas de brique voisine — vérifier). Dockerfile, CI
  (`cache-dependency-path`, pas vet/test/lint/govulncheck, `go list -deps -test`), `go-shards.txt` (lots du
  module), baseline par module, dependabot, `briques_imports_test` (ligne `haloapi`, permis : rien),
  `briques_internes_test` (ligne retirée), ratchets rerootés (`haloapi_sans_client_http`, invariants
  `xblauth`, `TestBriqueInfiniteRecoitLeDoerGarde`, netguard par préfixe) : même recette que D2 lot 3.
- **P-B — Quatre coupes, même méthode que D3** (tableau protocole / exploitation du lot 0 FAIT FOI) :
  - `games/halo_5/client.go` → `libs/haloapi/halo5` : transport, adresses, DTO bruts ; `netguard.Check` →
    `Doer` gardé fourni par un adaptateur `games/halo_5/haloadapter` (ou le paquet `halo_5` lui-même) ;
    politique de reprise = celle d'aujourd'hui, en valeur.
  - `mapcatalog/ugc.go` (Discovery UGC) → service de `libs/haloapi/infinite` (le troisième client UGC
    rejoint `FetchAsset`/`GetPlaylistConfig`) ; `mapcatalog` garde le CHOIX de la variante et appelle la brique.
  - `assets/fetcher_gamecms.go` (GameCMS) → service de `infinite` ; `assets` garde la résolution
    (`Resolver`, cache, `ctxkeys`, singleton d'adresses → `EndpointsDepuisResolveur`).
  - `sync/xbox_client.go` (Xbox : profils, présence… lire sur pièces) → `libs/haloapi/xboxlive` ; le jeton
    XSTS vient de `platform/auth` par paramètre (jamais le store dans la brique).
- **P-C — Aucun changement de comportement** : témoins = URL produites octet pour octet (test « URL figées »
  AVANT la coupe, vert APRÈS), politiques de reprise par chemin, surfaces netguard, valeurs de domaine à
  partir des mêmes fixtures ; T3 LIVE lecture seule par `cmd/diag_api_lecture` étendu d'une sous-commande
  par client nouveau (Halo 5 : service record du compte ; UGC : une variante ; GameCMS : un actif ; Xbox :
  le profil) — jetons par le magasin, jamais de re-capture.
- **P-D — README et tags** : README EN par brique (modèle unique, 1 page : objet, non-objet, exemple,
  licence — `gamefiles` : GPLv3 à cause de `ooz`, les deux autres : la licence du dépôt —, statut
  « expérimental, API non stable avant v1 », comment contribuer, lien vers l'ADR) ; les tags
  `libs/gamefiles/v0.1.0`, `libs/himap/v0.1.0`, `libs/haloapi/v0.1.0` sont POSÉS PAR L'UTILISATEUR (un tag
  publie ; le pilote prépare la commande et vérifie `go get` depuis un module jetable hors dépôt).
- **P-E — Dépôt séparé : décision différée, critère écrit** (lot 4) : on sépare une brique quand (a) un
  contributeur externe a ouvert une PR ou une issue dessus, ou (b) sa cadence de changement est devenue
  indépendante de LevelUp (moins d'une PR transverse par mois), ou (c) sa CI pèse sur celle de LevelUp. Mode :
  `git subtree split` (historique conservé), dépôt `JGtm/<brique>`, le module garde son chemin ou change
  avec un `replace` transitoire ; la couche exploitation reste dans LevelUp.
- **P-F — Hors périmètre** : convergence des politiques de reprise (comportement) ; retrait du singleton
  d'adresses ; garde netguard sur les surfaces d'authentification non gardées ; refonte du générateur de
  cartes lui-même.

## Lot 0 — Préalable : branche, relevé, témoins

- [x] 0.1 Worktree, branche, CI de la base verte ; PR « harnais » et « suites » fusionnées (sinon attendre).
  (2026-11-01 : `../LevelUp-wt-publication`, `refactor/publication-briques` depuis `release/v8.1.0` à
  `8f0b512c58` ; PR #116 et #117 fusionnées ; trois derniers runs CI de la release verts.)
- [x] 0.2 Relevé fichier par fichier des quatre clients (protocole / exploitation / mixte, symboles de
  couplage, destination) → tableau dans le plan. Relevé des appelants de chacun.
  (Section « Relevé du lot 0 » ci-dessus + `RELEVE_PUBLICATION_BRIQUES_LOT0_2026-11-01.md`.)
- [x] 0.3 Témoins : T1 (`go test -json ./...` → paquet/test/verdict) ; T2 « URL figées » : pour chaque client,
  un test qui enregistre les URL et en-têtes produits pour des identifiants sains (écrit AVANT, sur le code
  actuel, commité au lot 1) ; T3 LIVE lecture seule (sous-commandes existantes de `diag_api_lecture` +
  les quatre nouvelles écrites au lot 1 sur les clients ACTUELS) ; T4 noms de compteurs expvar.
  (T1 capturé : `go test -json ./... -count=1 -timeout 1200s` dans `apps/go-api`, code 0 en 206 s, 336 paquets
  dont 195 `ok`, 16 469 pass / 68 skip / 0 fail ; verdicts triés dans
  `%TEMP%\claude\...\scratchpad\temoins_pub_lot0_avant\T1_goapi.verdicts.txt`. T2, T3, T4 : périmètre fixé dans
  la section « Relevé du lot 0 », écriture au lot 1 comme prévu.)
- [x] 0.4 Gate : témoins écrits, thought_log. (Commit « lot 0 ».)

## Lot 1 — Garde-fou et outil témoin

- [x] 1.1 `briques_internes_test` : les quatre fichiers clients entrent dans le périmètre `haloapi` avec des
  tolérances datées par import visé (comptes mesurés), à zéro à la fin du lot 2.
  (2026-11-01 : champ `fichiers` (fichiers nommés jugés comme le périmètre) + auto-preuve ; tolérances
  mesurées = attendu : `platform/netguard` 4, `domain` 2, `ctxkeys` 1, `games` 1, `platform/auth` 1,
  `golang.org/x/time/rate` 1 ; mutation -1 → rouge vérifiée sur deux entrées.)
- [x] 1.2 Tests « URL figées » (T2) et sous-commandes `diag_api_lecture` (T3) sur les clients actuels ;
  capture « avant ».
  (T2 : quatre `*_url_figees_test.go` dans le paquet de chaque client, 19 tests / 29 sous-tests, attendus en
  littéral ; enregistreur UNIQUE `internal/haloapi/haloapitest` (stdlib seule) + garde-rail
  `archlint/enregistreur_http_unique_test.go` (allowlist datée des 9 copies préexistantes, ne descend
  que). Non figé : le délai de base de 800 ms (contexte annulé au premier essai) ; `Retry-After` mesuré 1 s.
  T3 : `halo5-servicerecord`, `ugc-carte <asset_id> [version_id]`, `gamecms-actif <chemin>`, `xbox-succes`
  dans `cmd/diag_api_lecture` (`titleseams.RegisterAll` exigé par le ratchet des binaires liés à `sync`).
  Captures « avant » (compte Chocoboflor) dans `%TEMP%\claude\...\scratchpad\temoins_pub_lot0_avant\` avec
  `T3_md5s.txt` : `servicerecord` 4 565 o, `halo5-servicerecord` 955 o, `ugc-carte` (Absolution
  `78da545f-a168-4a5e-9c8d-dd379067c352`) 659 o, `gamecms-actif`
  (`Waypoint/file/images/nameplates/104-001-olympus-campa-2ddbe23b_n809699482.png`) 103 o, `xbox-succes`
  103 415 o ; `calendrier` 403 (état connu depuis D3 : repli = `servicerecord`).)
- [x] 1.3 Gate : archlint vert, mutations (tolérance -1 → rouge), témoins capturés ; commit « lot 1 ».
  (Rejoué par le pilote : `go test` archlint + haloapi/... + halo_5 + mapcatalog + assets + sync, 9 paquets `ok`,
  code 0 ; `go vet` 0 ; `golangci-lint --new-from-rev=HEAD` 0 issue.)

## Lot 2 — Les quatre coupes (protocole dans `internal/haloapi`, exploitation dans LevelUp)

Dans le module go-api d'abord (comme D2 lot 2 et D3 lots 2-4 : couper AVANT de déplacer le module).

- [x] 2.1 `halo5` : `internal/haloapi/halo5` + adaptateur côté `games/halo_5` ; politique en valeur ; `Doer`
  gardé ; URL figées vertes ; tests portés sous leur nom.
  (2026-11-01, agent Opus, CR vérifié et gates rejoués par le pilote. Décisions appliquées : D-2.1-a `Retry`,
  `Limiter`, `ParseRetryAfter` montés à la racine `haloapi` (`retry.go`), alias dans `infinite`, méthode
  `RetenteStatut` (un champ `Retente` existe) ; D-2.1-b paquet `halo5` (`New(doerJSON, doerBinaire, retry)`,
  `Endpoints`, `Auth` par appel, `RetryHalo5()` = politique d'aujourd'hui, une seule boucle `faire`) ; DTO
  déménagés par `git mv`, `H5MatchEventsResponse`/`H5EventPlayer`/`H5WorldLocation`/`H5GameEvent` exportés ;
  `GetMatchDetail` NON porté (PUB-0-a) ; D-2.1-c `netguard.Doer` centralisé (`platform/netguard/doer.go`),
  `haloadapter` perd sa copie (4 sites), ratchet `haloapi_doer_garde_test` par paquet (`infinite.New` 1,
  `infinite.NewClassement` 2, `halo5.New` 2) ; D-2.1-d adaptateur `games/halo_5/client.go` 462 → 140 L, API des
  appelants inchangée, 27 fichiers d'appelants réécrits sur les DTO de la brique (`gofmt -r`), aucun alias de type ;
  D-2.1-e 8 tests existants et T2 sous leurs noms, aucune URL attendue modifiée (seul le cas `GetMatchDetail`
  retiré) ; D-2.1-f tolérances `platform/netguard` 3, `rate` retirée. Deux différences de forme documentées
  dans `halo5/doc.go` : garde consultée avant le limiteur ; `Retry-After` en date HTTP désormais lu. Gates :
  vet 0, `go list -deps -test` de la brique sans exploitation, 14 paquets `ok` + livesync `-tags=integration`,
  archlint vert avec deux mutations rouges, lint 0 issue.)
- [x] 2.2 UGC : service dans `infinite` ; `mapcatalog` appelle la brique ; URL figées vertes.
  (2026-11-01, agent Sonnet, gates rejoués par le pilote. `infinite/ugcmaps.go` : `CarteUGC` (DTO brut),
  `FetchCarteUGC(ctx, auth, assetID, versionID, Identite)`, `TelechargerFichierUGC(ctx, prefix, relPath, Identite)` ;
  `Identite{UserAgent, Origin}` = valeur de l'appelant ; GET unique paramétré `getUnique(ctx, requeteUnique)` dans
  `transport.go` (accept, en-têtes, échéance, borne de lecture ; `nameplate.go` l'appelle avec ses en-têtes et
  `DelaiNameplate` inchangés) ; `RetryAucune()` ; erreurs `ErreurTransport`, `ErreurLecture`, `HTTPError.Body`
  = 512 premiers octets. `mapcatalog/ugc.go` 185 → 137 L : façade `NewClient(tokens)` + `WithHTTPClient`,
  `MvarPaths(a *infinite.CarteUGC)`, messages `GET %s: HTTP %d (%s)` / `GET %s: %w` reformés depuis les erreurs
  typées ; `Asset` supprimé sans alias (2 appelants réécrits). Tolérances `platform/netguard` 2, `domain` 1.
  Non figé / différences de forme : le `WarnContext` de fermeture du corps disparaît ; dans le résolveur de
  nameplate, un corps coupé donne `lecture <url>: ...` au lieu de l'erreur brute (chemin d'erreur, `%w` conservé).
  Report au 2.3 : `getUnique` ne rend pas encore `Content-Type` (lint `unparam` sans lecteur) ; complexité ~11.
  T2 : 4 tests, aucune URL attendue modifiée. Gates : vet 0, brique sans exploitation, tests `ok`, lint 0.)
- [x] 2.3 GameCMS : service dans `infinite` ; `assets` garde la résolution ; URL figées vertes.
  (2026-11-01, agent Opus, gates rejoués par le pilote. `infinite/gamecmsfichiers.go` (76 L) : `URLFichierGameCMS(e,
  chemin)`, `URLImageGameCMS(e, brut)` (= l'ancien `buildGameCMSImageFetchURL`, 7 formes), `TelechargerFichierGameCMS(ctx,
  auth, url)` = GET unique SANS `Accept`, garde consultée d'abord (ordre d'avant rétabli sur décision pilote, test de
  mutation rouge). `assets/fetcher_gamecms.go` 356 → 242 L : `infinite.New(netguard.Doer{"gamecms_assets.get"},
  RetryAucune())`, `hoteGameCMSParDefaut()` remplace `defaultGameCMSBase` (aussi dans `fetcher_chain.go`),
  `gamecmsHostFor/PrefixFor` restent, classement des erreurs et messages inchangés, `tokens, _ :=` (PUB-0-b) et
  `http.DefaultClient` (PUB-0-c) conservés. D-2.3-a : pas de `Content-Type` rendu (aucun lecteur) — report du 2.2 clos.
  Tolérances : `platform/netguard` 1, `platform/auth` 1 ; `domain`, `ctxkeys`, `games` retirées ; allowlist
  `halowaypoint` : `fetcher_gamecms.go` retiré. Non figé (cas dégénérés) : Spartan vide avec jetons non nil → plus
  d'en-tête vide ; hôte vide → hôte par défaut au lieu d'une URL relative. T2 : 5 tests, aucune URL attendue modifiée ;
  18 tests existants et test démo inchangés. Gates : vet 0, brique sans exploitation, tests `ok`, lint 0.)
- [x] 2.4 Xbox : `internal/haloapi/xboxlive` ; `sync` appelle la brique avec le jeton en paramètre ; URL
  figées vertes.
  (2026-11-01, agent Sonnet, gates rejoués par le pilote. Paquet `xboxlive` (6 fichiers, 430 L dont 143 de
  test) : `Endpoints{Succes}`, `Auth{Autorisation}` (valeur complète de l'en-tête, masquée), `New(doer)`,
  `GetSuccesPage(ctx, auth, xuid, lang, Page{TitleID, SkipItems})` → `SuccesReponse`/`Succes` (DTO bruts exportés) ;
  URL construite comme avant (`xuid(...)` non échappé, `url.Values.Encode`), 4 en-têtes, garde d'abord, corps lu en
  entier puis statut (`HTTPError.Body` entier), `ErreurTransport`. `sync/xbox_client.go` 271 → 184 L : interface,
  `PlayerAchievementRaw`, `parseAchievementItem(xboxlive.Succes)`, pagination et ses trois journaux inchangés,
  `NewXboxHTTPClient` (signature inchangée) sur `netguard.Doer{"xbox_achievements.get"}`, messages recomposés à
  l'octet (`xbox_client: HTTP %d: %s`, `xbox_client: HTTP do: %w`) prouvés par `xbox_client_erreurs_test.go`.
  Tolérances : `fichiers` et table VIDES (mécanisme laissé pour le 3.3). Ratchet Doer gardé : `xboxlive.New` 1.
  T2 : 3 témoins, aucune URL attendue modifiée ; 14 tests de succès aux mêmes noms (6 constructions renommées).
  Différence de forme : garde avant la construction d'URL (un `xuid` invalide en mode démo rend `ErrOffline`).
  Gates : vet 0 (+ `-tags=integration`), brique sans exploitation, `haloapi/...` + `sync` + archlint `ok`, lint 0.)
- [x] 2.5 Tolérances à zéro ; `go list -deps -test ./internal/haloapi/...` propre ; T1 (aucun nom disparu),
  T3 identique, T4 identique ; revue adversariale 1 ronde (deux relecteurs : parité de sortie ; L6) ;
  corrections ; commit « lot 2 » ; CI verte.
  (2026-11-01. Tolérances et `fichiers` vides ; brique sans import d'exploitation. T1 : 16 612 verdicts contre 16 537,
  0 nom disparu hors les 14 cas de `TestParseRetryAfter` déplacés d'`infinite` à `haloapi` (retrouvés), 1 rouge =
  `mapcatalog.TestAddOverlayEntryConcurrentDossierAbsentNePerdRien` (verrou de fichier, « Accès refusé » sous charge,
  vert seul 3 fois : flake Windows, PUB-2.5-a). T3 : les 6 surfaces identiques à l'octet (SHA-256 égaux ;
  `calendrier` 403 des deux côtés). T4 : cinq surfaces netguard, mêmes noms. Revue A (parité, lecture seule, Sonnet) :
  0 bloquant, 2 à corriger (quatre copies de `masque` ; mode démo plus prouvé pour Halo 5 / UGC / Xbox), 7 mineurs ;
  corrigés (commit `a049bf3d5c`) : `haloapi.Masque` + garde-rail `haloapi_masque_unique_test`, trois tests de mode
  démo sur les constructeurs de production, garde en tête du blob UGC, `NewClient(int)` rétabli, commentaire
  `HTTPError.Body`, doc des différences de forme (4), cas de date future dans `LectureDeRetryAfter`. Revue B (L6,
  54 mutations, Sonnet) : 29 rouges, 25 vertes dont 12 observables → 12 tests + 2 compléments écrits sans toucher la
  production (`reprise_test.go`, `get_unique_test.go`, `netguard/doer_test.go`, `client_debit_test.go`, …), chacun
  prouvé rouge sous sa mutation ; aucun écart code / annonce. Gates pilote : vet 0, 11 paquets `ok`, lint 0.)

## Lot 3 — Module `libs/haloapi` + infrastructure

- [x] 3.1 `git mv apps/go-api/internal/haloapi libs/haloapi` ; `go.mod` ; `go.work` ; imports réécrits
  (`levelup/go-api/internal/haloapi/` → `github.com/JGtm/LevelUp/libs/haloapi/`) ; tiers.
  (2026-11-01, agent Opus. 90 fichiers déplacés (`testdata` compris) ; `go.mod` à la main (`x/sync v0.24.0`,
  `x/time v0.16.0`, `go.sum` 4 lignes copiées) ; `go.work` `use ./libs/haloapi` ; 123 imports réécrits par outil
  `go/parser` dans 109 fichiers (84 go-api, 25 brique), multiset ôté = ajouté ; `go mod tidy -diff` vide, `go mod verify`
  ok, compile aussi `GOWORK=off`.)
- [x] 3.2 Infrastructure (recette D2 lot 3) : Dockerfile, `ci.yml` (pas du module, cache paths, lots
  `go-shards.txt`, baseline par module, race si utile), lint, govulncheck, dependabot, Makefile
  (`libs-test` boucle déjà sur `libs/*`), `.air.toml`.
  (Dockerfile `COPY` ; `ci.yml` : pas go-build sans CGO sur les deux OS (vet, test, autonomie), 5e passage de lint,
  govulncheck, lot `haloapi` dans la matrice, artefact JSONL et couverture affichée ; `cache-dependency-path` ×10
  (ci, release, shared-social-gate) ; `go-shards.txt` : `module libs/haloapi` + `lot haloapi - @reste` ; baseline
  `.ai/baselines/tests_libs_haloapi.jsonl` (216 tests, exigée par la CI pour tout module présent dans les lots ; la
  baseline go-api ne portait aucune ligne `haloapi`) + entrée datée dans `check_test_baseline.sh` ; dependabot,
  `.air.toml`, `.gitleaks.toml` ; Makefile et lefthook : rien à changer. Docker absent du poste : `go mod download` hors
  ligne simulé ; la CI « Docker Build » fait foi.)
- [x] 3.3 Ratchets rerootés : `briques_imports_test` (ligne `haloapi`, rien de permis), `briques_internes_test`
  (ligne retirée ; si la table devient vide, retirer le mécanisme avec sa doc : règle 7),
  `haloapi_sans_client_http`, invariants `xblauth`, `TestBriqueInfiniteRecoitLeDoerGarde`, netguard préfixe,
  `racinesEspaceDeTravail`, `racinesOctetsBruts`, licence GPL (`cmd/server` ne doit pas lier `ooz` : sans
  changement), `no_halowaypoint_literal` (clés), `.gitleaks.toml`.
  (`briques_imports_test` : ligne `haloapi` + champ `tiers` par sous-arbre (errgroup/rate sous `infinite` seulement,
  porté depuis la table interne) ; `briques_internes_test.go` et `_garde_test.go` SUPPRIMÉS ;
  `briques_internes_fermeture_test.go` → `haloapi_fermeture_domaine_test.go` (contrôle distinct conservé) ;
  `haloapi_*`, `racinesOctetsBruts`, `racinesEspaceDeTravail` rerootés avec compteurs « aucun fichier → rouge » ;
  `enregistreur_http_unique` et `no_halowaypoint_literal` à double repère ; exemption netguard par préfixe retirée (le
  parcours ne balaie que go-api) ; trois tests go-api qui lisaient la brique par une échelle de `..` passent par
  `testutil.RepoRoot` (deux seraient devenus muets). Lint : règle `.golangci.yml` bornée aux trois fichiers DTO Halo 5 et
  au motif `Id` (dette gelée que le renommage inter-module faisait réapparaître ; justifiée et datée). Docs du même
  commit : CLAUDE.md (tableau, ligne auth, résumés ADR 0041/0042), amendement daté ADR 0042, `ARCHITECTURE_V6` et
  `COMMANDS` EN+FR.)
- [x] 3.4 Gate : build go-api + libs/haloapi ; `go list -deps -test` vide ; T1 (modulo préfixe), T3, T4 ;
  archlint ; lint des cinq modules ; `make libs-test` ; CI verte ; commit « lot 3 ».
  (Rejoué par le pilote : vet des deux modules 0 (+ `-tags=integration`), `cmd/server` compilé, brique sans
  `levelup/go-api`, 6 paquets de la brique `ok`, archlint `ok` (100 s), lint brique avec la config du dépôt 0 issue, lint
  go-api `--new-from-rev` 0, `make libs-test` vert (4 modules), `ci_go_shard.sh --verifier` : 373 paquets, 3 modules,
  8 lots. T1 (agent) : 16 648 verdicts, 0 fail ; 6 disparus = les tests de la table interne supprimée ; T3 : 6 SHA-256
  identiques au lot 2 ; T2 : seules les lignes d'import. CI : au push.)

## Lot 4 — README, tags, critère de séparation, clôture

- [x] 4.1 README EN pour `libs/gamefiles`, `libs/himap`, `libs/haloapi` (modèle unique, P-D) ; vérification
  que chaque exemple compile (`go vet` d'un `example_test.go` par brique).
  (2026-11-01, agent Sonnet, vérifié par le pilote : trois README (70 à 79 L, modèle en 10 sections : statut,
  objet, non-objet, install, exemple, prérequis, licence, contribution, non-affiliation) ; exemples copiés de
  `libs/haloapi/infinite/example_test.go`, `libs/gamefiles/himodule/example_test.go`, `libs/himap/himap/example_test.go`
  (`go vet` 0 dans les trois modules, avec et sans `-tags gamefiles` ; lint 0). Licences écrites : `gamefiles` GPL-3.0
  (`ooz/NOTICE.md`, importé par `himodule` seul), `himap` et `haloapi` AGPL-3.0-only (NOTICE racine). Forme
  d'installation : `go get <module>@v0.1.0` (Go résout le tag `libs/<brique>/v0.1.0`). `haloapitest` public (PUB-3-e
  tranché, ADR 0043 D-7). Constats du relecteur : `hinavmesh`/`mapvar` sans `doc.go` ; origine des fixtures
  `mapvar/testdata`, `hinavmesh/testdata` non documentée (PUB-4-a).)
- [x] 4.2 Vérification `go get` depuis un module jetable HORS dépôt (`go mod init essai && go get
  github.com/JGtm/LevelUp/libs/haloapi@<sha ou tag>`), consignée ; commande de tag préparée pour l'utilisateur
  (`git tag libs/haloapi/v0.1.0 <sha> && git push origin --tags`) — le pilote ne pousse pas de tag.
  (2026-11-01, pilote, module `essai` dans le scratchpad, `GOWORK=off`, `GOPROXY=direct`, `GONOSUMDB=github.com/JGtm/LevelUp`,
  sha `f1a5bfbf82` (lot 3 poussé) : `go get github.com/JGtm/LevelUp/libs/haloapi@f1a5bfbf82` →
  `v0.0.0-20261101134051-f1a5bfbf82ab`, `go.mod` de l'essai ne requiert que la brique et `x/sync` indirect, `go run` d'un
  `main` qui appelle `infinite.DefaultEndpoints().PrefixeJeu()` imprime `hi` ; `go get` de `libs/gamefiles` et `libs/himap`
  au même sha résolu (pseudo-versions). Note : Go télécharge aussi la racine `github.com/JGtm/LevelUp v8.0.8-…+incompatible`
  pour résoudre (tags `vX.Y.Z` du dépôt sans `go.mod` racine) : sans effet, mais un tag `libs/<brique>/v0.1.0` évitera la
  pseudo-version. **Commandes de publication, à exécuter par l'utilisateur après fusion dans la release puis `main`**
  (un tag publie ; depuis le checkout principal, sur le sha de `main` qui porte la fusion) :
  `git tag libs/gamefiles/v0.1.0 <sha> ; git tag libs/himap/v0.1.0 <sha> ; git tag libs/haloapi/v0.1.0 <sha> ;
  git push origin libs/gamefiles/v0.1.0 libs/himap/v0.1.0 libs/haloapi/v0.1.0` — puis vérification :
  `go get github.com/JGtm/LevelUp/libs/haloapi@v0.1.0` depuis le module `essai` (attendu : `v0.1.0` ; Go résout
  lui-même le tag `libs/haloapi/v0.1.0`, la forme `@libs/haloapi/v0.1.0` serait lue comme une révision).
  **`himap` hors espace de travail** (constaté par essai `essai_himap` au sha `f1a5bfbf82`) : `libs/himap/go.mod` ne
  requiert pas `libs/gamefiles` qu'il importe (ADR 0041 : `go.work` sans `require`) ; un consommateur doit donc faire
  `go get` des DEUX modules (`himap` puis `gamefiles`, puis `go mod tidy`) : vérifié, `go build` et `go run` passent
  alors (README de `himap` le dit). La forme propre (ADR 0041, amendement 2026-11-01) : après le tag de `gamefiles`,
  ajouter `require github.com/JGtm/LevelUp/libs/gamefiles v0.1.0` à `libs/himap/go.mod` (+ `go.sum` par
  `GOWORK=off go mod tidy` dans `libs/himap`), fusionner, PUIS taguer `himap` sur ce commit — deux temps, à faire avec
  l'utilisateur au moment des tags. Constat annexe : `go mod tidy` d'un consommateur qui doit sonder la RACINE
  `github.com/JGtm/LevelUp` échoue sur des chemins `static/maps/halo_infinite/<nom>'<...>.jpg` (apostrophe interdite
  dans un zip de module) : sans effet quand les `require` sont complets, à garder en tête (PUB-4-b).)
- [x] 4.3 ADR 0043 « Publication des briques » (EN) : modules, README, politique de version (`v0.x` sans
  stabilité promise), critère de séparation P-E, ce qui reste dans LevelUp ; CLAUDE.md (tableau) ; ADR 0041
  (quatrième module) ; ADR 0042 (périmètre élargi : Halo 5, Xbox, UGC, GameCMS).
  (2026-11-01, agent Sonnet, relu par le pilote : `docs/adr/0043-publishing-the-bricks.md` (212 L, D-1 à D-8 :
  trois modules sans séparation, tags `libs/<brique>/vX.Y.Z` par le mainteneur et `v0.x`, README modèle, licences
  GPL-3.0 / AGPL-3.0-only, critère P-E et mode `subtree split` décidé en dernier et exécuté ensemble, ce qui reste dans
  LevelUp, `haloapitest` public, différences de forme acceptées ; garde-rails cités par nom, les tests de licence
  vivant dans `cmd/server/licence_gplv3_test.go`) ; ADR 0041 mis à jour en place (D1, D3 `tiers`, D7) + amendement
  daté (4e module, table interne retirée, ordre des `require` avant tag) ; ADR 0042 : amendement du lot 3 étendu
  (périmètre, `getUnique`, racine, `netguard.Doer`, 13 surfaces) ; CLAUDE.md : entrée 0043, résumé 0042, tableau, et
  trois affirmations fausses corrigées (`tokencache.InvalidateCachedPlayerTokens`, 34 fichiers `*_gamefiles_test.go`,
  résumé 0042). Hors périmètre noté : `docs/CONFIGURATION.md:150` cite encore `halo.InvalidateCachedPlayerTokens`
  (PUB-4-c).)
- [x] 4.4 Registre des reports, addendum au backlog (entrée `[archi]` → faite / reste : tags par
  l'utilisateur, séparation sur critère), thought_log, mémoire ; caches `-Lot pub` ; PR, CI verte, fusion
  squash, nettoyage.
  (2026-11-01 : `.ai/REGISTRE_REPORTS.md` (section « Publication des briques », ligne D3-L6-e soldée),
  `.ai/BACKLOG.md` (addendum à l'entrée `[archi]`), thought_log, mémoire du pilote ; PR ouverte vers
  `release/v8.1.0` ; fusion squash et nettoyage du worktree et des caches `-Lot pub` APRÈS l'accord de l'utilisateur
  et la CI verte — voir le journal de clôture.)

## Témoins avant/après

| Témoin | Capture | Comparaison |
|---|---|---|
| T1 tests Go | `go test -json ./...` → (paquet, test, verdict) | mêmes noms et verdicts modulo paquet, 0 fail |
| T2 URL figées | tests écrits au lot 1 sur les clients actuels (URL + en-têtes pour identifiants sains) | verts après chaque coupe sans modification |
| T3 LIVE lecture seule | `diag_api_lecture` : surfaces existantes + Halo 5 / UGC / GameCMS / Xbox | mêmes octets (horodatages ignorés) |
| T4 compteurs | constantes expvar des quatre clients | noms inchangés |
| ratchets auth | sentinelles `platform/auth`, netguard | verts, allowlists des guards 2 et 3 VIDES |

## Protocole de reprise de session

1. `git branch --show-current` ; `git log --oneline -10` dans `../LevelUp-wt-publication`.
2. Premier item sans statut du premier lot non clos = point de reprise.
3. Vérifier sur pièces avant de coder (les chiffres datent du 2026-10-30).
4. Journal : `.ai/thought_log.md`, entrées « Publication des briques — lot N ».

## Découvertes (notées, non traitées)

- PUB-0-a (2026-11-01, relevé) : `halo_5.(*Client).GetMatchDetail` (client.go:187-205) n'a aucun appelant dans le
  dépôt, ni prod ni test (règle 7 « 0 code mort » : à trancher au lot 2.1 — porter ou supprimer ; le pilote propose
  de ne pas le porter).
- PUB-0-b : `assets/fetcher_gamecms.go:170` ignore l'erreur du `TokenProvider` sans journal (`tokens, _ :=`) —
  anti-pattern n° 10 ; comportement conservé par P-C, à corriger hors de ce chantier.
- PUB-0-c : `GameCMSFetcher` tourne en prod sur `http.DefaultClient` (sans délai) : `AssetConfig.HTTPClient` jamais
  renseigné (`server_apiv1.go:1450`). Conservé par P-C ; à traiter séparément.
- PUB-0-d : `sync/xbox_client.go:160` insère le `xuid` non échappé dans le chemin ; `mapcatalog/ugc.go:132` idem pour
  `relPath`. Conservé tel quel (témoin T2 fige la forme actuelle).
- PUB-0-e : les cinq surfaces netguard de ces clients ne figurent dans aucune table ni dans la liste D-6 de l'ADR
  0042 (huit surfaces, toutes Infinite) : la liste est à compléter au lot 4.3 (ADR 0042, périmètre élargi).
- PUB-0-f : `mapcatalog/ugc.go` et le chemin HTTP de `sync/xbox_client.go` n'ont aucun test : T2 les couvre pour la
  première fois (lot 1.2).
- PUB-1-a (lot 1) : GameCMS `/hi/Progression/file/...` (médailles, fond, sprite CSR) répond 403 au compte de
  diagnostic alors que `/hi/Waypoint/file/...` répond 200 ; les chemins `images/file/Progression/...` des
  fixtures de tests répondent 404. À instruire hors chantier (droits du jeton ? chemins morts ?).
- PUB-1-b : `diag_api_lecture xbox-succes` fait deux rotations de refresh token par exécution (l'`ExchangeResult`
  n'expose pas l'access_token) et ne consulte pas la porte des jetons morts de la passe de succès. Accepté
  pour un outil de lecture ; documenté dans l'en-tête de `main.go`.
- PUB-1-c : neuf `_test.go` déclaraient déjà leur propre `RoundTrip` enregistreur (allowlist datée du
  garde-rail `enregistreur_http_unique_test.go`) ; migration vers `haloapitest.Enregistreur` hors chantier.
- PUB-2.1-a : `netguard_coverage_test` cherche la CHAÎNE `netguard.Check` dans un fichier qui fait `.Do(` ;
  `platform/netguard/doer.go` passe grâce à son doc-commentaire, pas à un appel détecté. Ratchet à durcir (AST).
- PUB-2.1-b : `platform/auth/xbl_doer.go` porte sa propre copie du Doer gardé (2e copie à côté de
  `netguard.Doer`) ; une 3e déclenche la centralisation (règle 6).
- PUB-2.1-c : `games/halo_5` a deux commentaires de paquet (`doc.go` et `client.go`), préexistant.
- PUB-2.3-a : `infinite/appearance.go` `buildCustomizationImageURL` est un quasi-doublon plus pauvre de
  `URLImageGameCMS` (ne reconnaît que `images/file/`, ne refuse pas `.json`) ; non fusionné (P-F).
- PUB-2.3-b : `scripts/warm_bp_assets/main.go` porte deux constantes `gameCMSBase` locales (lignes ~359 et ~428) en
  doublon de l'hôte de la brique.
- PUB-2.3-c : l'en-tête de `assets/demo_offline_no_persist_test.go` (lignes 9-12) cite encore `doGet` et
  `netguard.Check`, disparus de `fetcher_gamecms.go`.
- PUB-2.5-a : `mapcatalog.TestAddOverlayEntryConcurrentDossierAbsentNePerdRien` flake sous charge sur Windows
  (« verrou d'ecriture impossible a poser : Accès refusé », une carte perdue) ; vert seul. Test de verrou à durcir.
- PUB-2.5-b (revue A) : lecture du corps d'un non-200 (512 octets puis drainage 64 Kio) pour UGC et GameCMS, là où
  l'ancien GameCMS ne lisait rien ; avec `http.DefaultClient` sans délai (PUB-0-c) un corps suspendu peut bloquer.
- PUB-2.5-c (revue A) : `mapcatalog.Client` copie les jetons à la construction (l'ancien relisait le pointeur à chaque
  requête) ; `WithHTTPClient` exporté avec des appelants de test seulement (comme `games/halo_5`).
- PUB-2.5-d (revue A) : alias d'import de `haloapi/halo5` hétérogènes (`halo5` dans `games/halo_5`, `h5api` dans
  `cmd/*`) ; le T2 Xbox remplace le Doer gardé par un client nu (le test de mode démo couvre le constructeur).
- PUB-2.5-e (revue A, N3) : la redirection 302 des rendus Halo 5 réémet le jeton Spartan vers `image.halocdn.com`
  (comportement préexistant, figé par T2) : à garder en tête pour la publication de la brique.
- PUB-2.5-f (revue B) : `Retry.Delai/Attendre/RetenteStatut` ne sont testés que depuis `infinite` et `halo5` ; le
  paquet racine `haloapi` ne porte que `TestParseRetryAfter`, `TestNonRetryable`, et le test de `Masque`.
- PUB-3-a : la baseline de la brique n'est vérifiée qu'en CI (en local `make libs-test` joue ses tests sans la confronter).
- PUB-3-b : la couverture de go-api change mécaniquement (paquets de la brique hors du dénominateur) ; CI fait foi.
- PUB-3-c : `leaderboard_scraper_test` garde un `t.Skipf` sur fixture absente ; devrait être un échec.
- PUB-3-d : dix ratchets archlint ne balaient que go-api (sujets étrangers à la brique) ; non étendus.
- PUB-3-e : `haloapitest` (enregistreur de tests) sera importable par les utilisateurs du module une fois le tag posé :
  à trancher au lot 4 (le garder public comme `net/http/httptest`, ou le documenter « usage interne »).
- PUB-3-f : ADR 0041 affirme encore que la troisième brique « reste dans apps/go-api » → lot 4.3.
- PUB-4-a : origine des fixtures `libs/gamefiles/{mapvar,hinavmesh}/testdata` non documentée ; ADR 0041 interdit tout
  fichier extrait du jeu dans une brique : à vérifier et à documenter (NOTICE ou commentaire) avant le tag.
- PUB-4-b : la racine `github.com/JGtm/LevelUp` n'est pas un module zip valide (`static/maps/halo_infinite/*'*.jpg`,
  apostrophe interdite) ; sans effet pour les modules en sous-répertoire tant que leurs `require` sont complets.
- PUB-4-c : `docs/CONFIGURATION.md:150` cite `halo.InvalidateCachedPlayerTokens` (paquet disparu ; la fonction est
  `tokencache.InvalidateCachedPlayerTokens`).
- PUB-4-d : `hinavmesh`, `mapvar`, `himap` n'ont pas de `doc.go` (en-têtes de fichier seulement : `go doc` sans synthèse).
- PUB-0-g : première capture T1 rouge sur ce poste : `gcc` absent du PATH du PowerShell (lien DuckDB) ; recette =
  `C:\msys64\ucrt64\bin` en tête du PATH + `CGO_ENABLED=1` (mémoire `reference_air_rebuild_fails_gcc_missing_path`).
