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

## Décisions tranchées avant exécution (pilote, 2026-10-30 ; à confirmer par l'utilisateur au lancement)

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

- [ ] 0.1 Worktree, branche, CI de la base verte ; PR « harnais » et « suites » fusionnées (sinon attendre).
- [ ] 0.2 Relevé fichier par fichier des quatre clients (protocole / exploitation / mixte, symboles de
  couplage, destination) → tableau dans le plan. Relevé des appelants de chacun.
- [ ] 0.3 Témoins : T1 (`go test -json ./...` → paquet/test/verdict) ; T2 « URL figées » : pour chaque client,
  un test qui enregistre les URL et en-têtes produits pour des identifiants sains (écrit AVANT, sur le code
  actuel, commité au lot 1) ; T3 LIVE lecture seule (sous-commandes existantes de `diag_api_lecture` +
  les quatre nouvelles écrites au lot 1 sur les clients ACTUELS) ; T4 noms de compteurs expvar.
- [ ] 0.4 Gate : témoins écrits, thought_log.

## Lot 1 — Garde-fou et outil témoin

- [ ] 1.1 `briques_internes_test` : les quatre fichiers clients entrent dans le périmètre `haloapi` avec des
  tolérances datées par import visé (comptes mesurés), à zéro à la fin du lot 2.
- [ ] 1.2 Tests « URL figées » (T2) et sous-commandes `diag_api_lecture` (T3) sur les clients actuels ;
  capture « avant ».
- [ ] 1.3 Gate : archlint vert, mutations (tolérance -1 → rouge), témoins capturés ; commit « lot 1 ».

## Lot 2 — Les quatre coupes (protocole dans `internal/haloapi`, exploitation dans LevelUp)

Dans le module go-api d'abord (comme D2 lot 2 et D3 lots 2-4 : couper AVANT de déplacer le module).

- [ ] 2.1 `halo5` : `internal/haloapi/halo5` + adaptateur côté `games/halo_5` ; politique en valeur ; `Doer`
  gardé ; URL figées vertes ; tests portés sous leur nom.
- [ ] 2.2 UGC : service dans `infinite` ; `mapcatalog` appelle la brique ; URL figées vertes.
- [ ] 2.3 GameCMS : service dans `infinite` ; `assets` garde la résolution ; URL figées vertes.
- [ ] 2.4 Xbox : `internal/haloapi/xboxlive` ; `sync` appelle la brique avec le jeton en paramètre ; URL
  figées vertes.
- [ ] 2.5 Tolérances à zéro ; `go list -deps -test ./internal/haloapi/...` propre ; T1 (aucun nom disparu),
  T3 identique, T4 identique ; revue adversariale 1 ronde (deux relecteurs : parité de sortie ; L6) ;
  corrections ; commit « lot 2 » ; CI verte.

## Lot 3 — Module `libs/haloapi` + infrastructure

- [ ] 3.1 `git mv apps/go-api/internal/haloapi libs/haloapi` ; `go.mod` ; `go.work` ; imports réécrits
  (`levelup/go-api/internal/haloapi/` → `github.com/JGtm/LevelUp/libs/haloapi/`) ; tiers.
- [ ] 3.2 Infrastructure (recette D2 lot 3) : Dockerfile, `ci.yml` (pas du module, cache paths, lots
  `go-shards.txt`, baseline par module, race si utile), lint, govulncheck, dependabot, Makefile
  (`libs-test` boucle déjà sur `libs/*`), `.air.toml`.
- [ ] 3.3 Ratchets rerootés : `briques_imports_test` (ligne `haloapi`, rien de permis), `briques_internes_test`
  (ligne retirée ; si la table devient vide, retirer le mécanisme avec sa doc : règle 7),
  `haloapi_sans_client_http`, invariants `xblauth`, `TestBriqueInfiniteRecoitLeDoerGarde`, netguard préfixe,
  `racinesEspaceDeTravail`, `racinesOctetsBruts`, licence GPL (`cmd/server` ne doit pas lier `ooz` : sans
  changement), `no_halowaypoint_literal` (clés), `.gitleaks.toml`.
- [ ] 3.4 Gate : build go-api + libs/haloapi ; `go list -deps -test` vide ; T1 (modulo préfixe), T3, T4 ;
  archlint ; lint des cinq modules ; `make libs-test` ; CI verte ; commit « lot 3 ».

## Lot 4 — README, tags, critère de séparation, clôture

- [ ] 4.1 README EN pour `libs/gamefiles`, `libs/himap`, `libs/haloapi` (modèle unique, P-D) ; vérification
  que chaque exemple compile (`go vet` d'un `example_test.go` par brique).
- [ ] 4.2 Vérification `go get` depuis un module jetable HORS dépôt (`go mod init essai && go get
  github.com/JGtm/LevelUp/libs/haloapi@<sha ou tag>`), consignée ; commande de tag préparée pour l'utilisateur
  (`git tag libs/haloapi/v0.1.0 <sha> && git push origin --tags`) — le pilote ne pousse pas de tag.
- [ ] 4.3 ADR 0043 « Publication des briques » (EN) : modules, README, politique de version (`v0.x` sans
  stabilité promise), critère de séparation P-E, ce qui reste dans LevelUp ; CLAUDE.md (tableau) ; ADR 0041
  (quatrième module) ; ADR 0042 (périmètre élargi : Halo 5, Xbox, UGC, GameCMS).
- [ ] 4.4 Registre des reports, addendum au backlog (entrée `[archi]` → faite / reste : tags par
  l'utilisateur, séparation sur critère), thought_log, mémoire ; caches `-Lot pub` ; PR, CI verte, fusion
  squash, nettoyage.

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

(vide au 2026-10-30)
