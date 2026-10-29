# Audit — découplage de trois briques : cartes, décodeur de film, API Halo — 2026-10-27

Point de départ : trois idées du user (sortir chacune sur son propre dépôt). Ce registre ne
tranche pas la question du dépôt ; il établit ce qui empêche aujourd'hui chaque brique d'être
autonome, et ce qu'il faut couper pour qu'elle le devienne. Une fois les coupes faites, le
passage en module Go séparé (même dépôt, `go.work`) ou en dépôt propre est une formalité à
décider à ce moment-là. Aucun code modifié : chaque axe deviendra un plan (`plan-review`),
exécuté sous `plan-execution`.

## Cadrage

- **Périmètre** : `internal/himap`, `himodule`, `hinavmesh`, `ooz`,
  `games/halo_infinite/film/**` (dont `replay/mapvar`), `platform/halo`, `sync/haloclient`,
  `platform/auth` ; état de `release/v8.1.0` au 2026-10-27 (`fb06cde01`).
- **Doctrine de référence** : skill `arch-rules` (couches, `platform/` = infrastructure,
  `analysis/` pur, code de titre sous `games/{slug}/`) ; ADR 0023, 0030, 0034, 0038 ; CLAUDE.md
  règle 6 (garde-rail à la factorisation).
- **Vérification** : mesures par `go list -deps` et relevé des imports fichier par fichier
  (fichiers hors `_test.go`), constats rouverts sur pièces par le pilote. **Vérification
  adverse par contexte frais NON faite.**
- **Critère d'autonomie d'une brique** : elle n'importe aucun paquet de LevelUp hors d'elle-même
  (ou d'une brique de rang inférieur), et c'est LevelUp qui l'importe, jamais l'inverse.
- **Gravité** (échelle `adversarial-audit`) : coût de la dette aujourd'hui, pas urgence produit.

## Synthèse

| # | Brique | Taille (hors tests) | État | Gravité | Effort estimé | Réécriture ? |
|---|---|---|---|---|---|---|
| D1 | Images de carte (`himap` + `himodule` + `hinavmesh` + `ooz`) | ~12 500 L | 1 seul lien sortant (`mapvar`) | P2 | ~1 jour | non |
| D2 | Décodeur de film (`film/**`) | ~155 000 L | propre en interne ; types de sortie possédés par LevelUp | P2 | 2 à 4 jours | non (déplacements) |
| D3 | API Halo (`platform/halo` + `sync/haloclient` + `platform/auth`) | ~6 500 L + auth | protocole et exploitation mélangés, deux clients | P1 | 1 à 2 semaines | oui, partielle |
| G | Garde-fous d'autonomie (liste fermée des imports par brique) | — | absents pour les trois | P1 | inclus dans D1-D3 | — |

Les efforts sont des estimations du pilote, pas des mesures.

Ordre recommandé : **D1** (le plus court, et il libère `mapvar` dont D2 a aussi besoin), puis
**D2**, puis **D3**. Chaque axe pose son garde-fou G **avant** le premier déplacement (méthode
du ratchet `archlint/film_layers_deps_test.go` : l'allowlist de départ EST la liste des coupes,
et elle se vide à mesure).

Brique cible de rang inférieur, commune à D1 et D2 : **« fichiers du jeu »** = `himodule`,
`ooz`, `hinavmesh` et le lecteur de variantes de carte (`mapvar`, aujourd'hui sous
`film/replay/`).

---

## Constats retenus

### D1 — Images de carte : `mapvar` rangé sous le film [P2]

- **Où** : `internal/himap/cuisson_forge.go:68,463,600,709,743,1002` et
  `internal/himap/zones_forge.go:6-7` utilisent `mapvar.Object` et `mapvar.ZonesNommeesForge`.
  `mapvar` vit sous `internal/games/halo_infinite/film/replay/mapvar/` alors qu'il lit les
  fichiers de variante de carte (`.mvar`) et n'importe aucun paquet de LevelUp.
- **Le reste est déjà autonome** : `himap` n'importe que `himodule`, `hinavmesh`, `ooz` et
  `mapvar` ; `hinavmesh` et `ooz` n'importent rien ; `himodule` n'importe que `ooz`.
- **Conséquence** : la brique « images de carte » dépend nominalement du décodeur de film (un
  paquet de 155 000 lignes) pour un lecteur de format qui n'a rien à voir avec le film.
- **Reproduction** : `cd apps/go-api && go list -deps ./internal/himap/... | grep '^levelup/go-api'`.
- **Traitement proposé** : déplacer `mapvar` dans la brique « fichiers du jeu » ; `film/replay`
  et `himap` l'importent tous les deux. Garde-fou : `himap`, `himodule`, `hinavmesh`, `ooz`,
  `mapvar` n'importent rien de `internal/` hors de cette liste.
- **Ce qui reste dans LevelUp** : les commandes `mapfond-build`, `mapstruct-build` (30 paquets
  internes chacune, parce qu'elles cadrent les fonds avec les positions lues dans les films),
  `mapcatalog`, `mapdecoupe`. Seule la bibliothèque d'extraction et de rendu est la brique.
- **Points d'attention** : `ooz` est un pont CGO (C++, mingw-w64) ; les tests qui lisent
  l'installation du jeu (tag `gamefiles`) suivent la brique. Aucun fichier extrait du jeu ne
  doit entrer dans le dépôt d'une brique publiée.
- **Tests** : `go test ./internal/himap/... ./internal/games/halo_infinite/film/replay/...`,
  `make go-api-test-gamefiles`, recuisson d'un fond déjà publié identique à l'octet.

### D2 — Décodeur de film : il ne possède pas ses types de sortie [P2]

- **Stabilité** : depuis la v8.0.0 (`c8eeea23a`), 167 fichiers touchés, +4 945 / −871 lignes
  sur ~155 000. Le user le juge stabilisé ; la mesure va dans le même sens.
- **Déjà tenu** : sens unique des cinq couches et interdit d'importer `internal/analysis`
  (`archlint/film_layers_deps_test.go`, R1-R4 stricts) ; surface publique par la façade
  `decfilm` (`archlint/film_facade_surface_test.go`) ; couches internes fermées par le
  compilateur (`film/internal/`). Aucune dépendance à DuckDB.
- **Imports sortants (fichiers hors tests, nombre de fichiers)** :

  | Paquet importé | Fichiers | Nature présumée (à confirmer au plan) |
  |---|---|---|
  | `games/weapons/filmshell` | 11 | connaissance du jeu (armes) |
  | `observability` | 9 | infrastructure de journal |
  | `games/canonical` | 7 | types inter-titres de LevelUp |
  | `domain/highlightevent` | 6 | type de sortie du décodeur |
  | `domain/playerposition` | 3 | type de sortie du décodeur |
  | `games/halo_infinite/equipenergy` | 2 | connaissance du jeu (équipement) |
  | `domain/objectiveevent` | 2 | type de sortie du décodeur |
  | `domain/equipmentusage` | 2 | type de sortie du décodeur |
  | `platform/atomicfile` | 1 | utilitaire d'écriture |
  | `filmproc` | 2 | seulement sous `film/research/cmd_*` (outils de recherche) |

  Les types de sortie sont partagés avec LevelUp : `replaydoc` 5 paquets hors film,
  `playerposition` 5, `highlightevent` 7, `objectiveevent` 5, `equipmentusage` 3.
- **Conséquence** : la flèche va dans le mauvais sens pour les types de sortie : le décodeur
  dépend du domaine de LevelUp au lieu que LevelUp dépende du contrat du décodeur.
- **Reproduction** : `grep -rhoE --include='*.go' --exclude='*_test.go'
  '"levelup/go-api/internal/[^"]+"' internal/games/halo_infinite/film | grep -v halo_infinite/film
  | sort | uniq -c`.
- **Traitement proposé** :
  1. Types de sortie (`highlightevent`, `playerposition`, `objectiveevent`, `equipmentusage`,
     `replaydoc`) déplacés sous le décodeur ; les 3 à 7 paquets lecteurs les importent de là.
  2. `filmshell` et `equipenergy` : s'ils sont de la connaissance du jeu, ils partent avec le
     décodeur ; sinon, inversion (le décodeur expose une interface, LevelUp la remplit).
  3. `canonical` : décider si le décodeur en a besoin (types inter-titres = produit LevelUp,
     donc plutôt une conversion côté LevelUp).
  4. `observability` remplacé par `log/slog` pur ou une interface injectée ; `atomicfile`
     copié ou déplacé dans la brique.
  5. `film/research/cmd_*` (outils de recherche qui utilisent `filmproc`) : sortir du décodeur
     ou accepter `filmproc` dans la brique.
  6. La génération des artefacts (`replaybuild`, `replay-worker`, cache, persistance des faits)
     reste dans LevelUp : ce n'est pas le décodeur.
- **Tests** : ratchets `film_*` existants, `replay-equiv` à révision constante (faits relus,
  `depuis_les_faits=false`), `go test ./internal/...`. Les révisions de couche ne doivent pas
  bouger (déplacements purs, empreinte insensible aux chemins à vérifier).
- **Effort** : surtout la mise à jour mécanique des imports (52 paquets importent le décodeur).

### D3 — API Halo : protocole et exploitation mélangés [P1]

**Critère de coupe.** *API* = ce dont un tiers aurait besoin pour parler aux services Halo avec
un jeton valide : sans état, sans politique, sans interprétation. *Exploitation* = ce qui décide
qui appelle, quand, combien, avec le jeton de qui, où l'on range le résultat et ce qu'il veut
dire pour LevelUp.

| API (brique) | Exploitation (reste dans LevelUp) |
|---|---|
| Chaîne d'authentification : `auth/xsts.go`, `sisu_provider.go`, `oauth_refresh.go`, `halo_exchange.go`, `xbox_device_code.go`, `auth_code.go`, `oauth_error.go`, `api_constants.go`, `refresh_user_xsts.go`, `xbox_profile_resolver.go`, `peoplehub_resolver.go` | Stockage des jetons par xuid et état watcher : `multi_user_token_store.go`, `token_store.go`, `user_token_store.go`, `watcher_*.go`, `attempt_*.go` ; mise en commun `pool/`, `pool_sharing.go` (ADR 0038) ; `refresh_loop.go`, `token_health.go`, `metrics.go`, `cli_refresh.go`, `access_token_store_first.go`, `link_strategy.go`, `cached_header_provider.go` |
| Clés d'adresse et hôtes par défaut, surchargeables par injection | Surcharges lues dans les manifests `config/titles/{slug}/` et injectées au boot |
| Transport HTTP : `http.Client` reçu, nouvel essai, 429 / `Retry-After`, erreurs typées (auth, film disparu) | Budget de requêtes, `netguard` (fourni sous forme de `http.Client`), cache de jetons de 50 min (`platform/halo/player_token_cache.go`) |
| Appels stats, skill/CSR, carrière, économie, CMS et découverte, défis, classements, confidentialité, plaques, UGC ; DTO JSON bruts | Traduction des DTO vers le domaine : `assets.Ref`, `assets.Resolver`, `metadata.MedalEntry`, `analysis.MMRKnown`, `halo_infinite.AssetFetcher` |
| Téléchargement du manifeste et des fragments de film : rend des octets | Décodage et prédicat de finalisation (`decfilm`, `finalise`), cache local des films (`filmcache`), choix de la variante de carte (`mapcatalog`) |

Constats sur pièces :

- **D3.1 — Le client HTTP décode et met en cache les films.**
  `sync/haloclient/halo_client_film.go:106` (`decfilm.Inflate`,
  `decfilm.FilmMajorVersionFromHeader`), `:143` et `:341` (`finalise.Finalise`) ;
  `sync/haloclient/local_film_cache.go:46,127` (`filmcache.ManifestsRoot`, `filmcache.LireChunk`).
  Conséquence : `haloclient` tire une vingtaine de paquets du décodeur (dont ses couches
  internes, par transitivité) ; une évolution du décodeur
  touche le client réseau.
- **D3.2 — Le client interprète le métier.** `sync/haloclient/halo_skill.go:203,253`
  (`analysis.MMRKnown`) ; `sync/haloclient/halo_client_mvar.go:30,41` (`mapcatalog.NewClient`,
  `mapcatalog.ChoisirFichierVariante`).
- **D3.3 — `platform/halo` dépend de l'adaptateur du titre.** `platform/halo/catalog_fetcher.go:5-30`
  adapte `HaloProvider` à `halo_infinite.AssetFetcher` ; `medal_provider.go:36,53`
  (`metadata.MedalEntry`) ; `provider.go:138,199,329` (`assets.Resolver`, `assets.Ref`). Par
  ricochet, un paquet de la couche `platform/` tire DuckDB, `analysis/*`, `progression/*`,
  `prestige`, `migration` (51 paquets internes au `go list -deps`). C'est une inversion de
  couche au sens du skill `arch-rules`.
- **D3.4 — Deux clients pour le même service.** `platform/halo` (« providers » : catalogue,
  saisons, médailles, défis, passe de combat, confidentialité, classements, comparaison) et
  `sync/haloclient` (matchs, carrière, skill, playlists, films, plaques) font chacun du HTTP et
  chacun leur résolution d'adresses (`platform/halo/endpoints.go:28` et
  `sync/haloclient/endpoint_resolver.go`, tous deux sur `games.DefaultEndpointResolver`, un
  singleton posé au boot par `games.SetDefaultEndpointResolver`).
- **D3.5 — L'authentification mêle protocole et politique dans un même paquet.**
  `platform/auth` contient à la fois la chaîne d'échange (sans état) et le stockage, la mise en
  commun et les boucles de rafraîchissement. Les fichiers de protocole n'importent que
  `domain.HaloTokens` et `domain/title` (`halo_exchange.go`, `sisu_provider.go`,
  `azure_credentials.go`, `oauth_refresh.go`) : la coupe est nette.
- **Reproduction** (depuis `apps/go-api`) :
  `grep -nE '"levelup/go-api/internal/' internal/platform/halo/*.go internal/sync/haloclient/*.go | grep -v _test`
  pour le relevé par fichier ; `go list -deps ./internal/platform/halo/... | grep '^levelup/go-api'`
  pour la dépendance transitive.
- **Traitement proposé** :
  1. Un seul paquet client (brique API) : transport, adresses (clés + défauts, surcharge
     injectée, plus de singleton global), un fichier par service, DTO bruts.
  2. Le film : le client rend manifeste et octets ; décodage, finalisation et cache passent dans
     un service d'exploitation de LevelUp (proche de `sync/replayartifacts` ou d'un
     `filmfetch` dédié).
  3. Traductions DTO → domaine (`assets`, `metadata`, MMR, `AssetFetcher`) dans un adaptateur
     côté LevelUp, sous `games/halo_infinite/`.
  4. `platform/auth` coupé en deux : protocole (dans la brique API, avec son propre type de
     jetons à la place de `domain.HaloTokens`) et exploitation (reste, garde les ratchets
     `auth/sentinel_test.go`, ADR 0023 et 0038 inchangés).
  5. Les 16 paquets qui importent `platform/halo` ou `sync/haloclient` (9 sous `internal/`,
     dont `api`, `api/handlers`, `service`, `sync`, `watcher` ; 7 commandes) migrent vers le client unifié
     ou l'adaptateur.
- **Risques** : chemin d'authentification et de mise en commun des jetons (lot à risque :
  `adversarial-review` obligatoire) ; le cache de jetons (`halo.InvalidateCachedPlayerTokens`)
  change de paquet, tous ses appelants à suivre.
- **Pertinence d'une publication à part** : faible. Des bibliothèques communautaires existent
  (Grunt en C#, SPNKr en Python) et la valeur propre à LevelUp est du côté exploitation. Le
  découplage vaut pour l'architecture, pas pour la publication.

### G — Garde-fous d'autonomie absents [P1]

- **Où** : `internal/archlint/` tient le sens interne du décodeur et l'interdit `analysis`
  (`film_layers_deps_test.go`), sa surface publique (`film_facade_surface_test.go`), mais
  **aucune liste fermée de ce qu'une brique a le droit d'importer hors d'elle**, pour aucune
  des trois.
- **Conséquence** : c'est par là que les dérives D1, D3.1-D3.3 sont entrées : chaque import
  ajouté était local et raisonnable, rien ne l'a refusé.
- **Traitement proposé** : un ratchet par brique (`archlint/brique_<nom>_imports_test.go` ou
  une table unique brique → imports permis, plus conforme à la règle 6), posé AVANT les
  déplacements avec une allowlist datée égale aux coupes restantes, qui doit finir vide.

---

## Constats écartés

- **« Le décodeur touche DuckDB »** (affirmé en conversation puis vérifié) : faux. Les deux
  occurrences (`film/internal/facts/fallback/registre_killsource_carte.go:90,157`) sont des
  mentions dans des commentaires ; aucun import DuckDB ni `database/sql` sous `film/`.
- **Publier l'API Halo comme dépôt** : non retenu (voir D3, pertinence). Le découplage, lui, est
  retenu.

## Décisions (user, 2026-10-27)

Prises sur recommandation du pilote, après vérification sur pièces des trois paquets de D2.

1. **Forme cible : modules Go dans le même dépôt.** Un `go.mod` par brique sous `libs/<brique>/`,
   reliés par `go.work` ; importables de l'extérieur par tag `libs/<brique>/vX.Y.Z`. Une
   modification qui touche brique et application reste un seul commit et une seule CI. Un
   dépôt propre reste possible plus tard, brique par brique, sur décision nouvelle.
2. **D2, catalogues : `filmshell` et `equipenergy` partent avec le décodeur**, comme paquets
   publics de sa brique ; LevelUp les importe de là. Constat : les deux sont des feuilles
   (aucun import du dépôt) ; `filmshell` (catalogue des armes telles que le film les nomme) est
   lu aussi par `platform/duckdb`, `persist` (via `games/weapons`), `sync/killcollector`,
   `games/halo_infinite`, `cmd/diag_film` ; `equipenergy` (énergie des équipements, extraite des
   fichiers du jeu par `cmd/equipenergy-build` via `himap`) par `replaybuild`. Le producteur
   d'`equipenergy` écrit ce format sans que la brique « fichiers du jeu » en dépende.
3. **D2, identité des entités du film : au décodeur.** `games/canonical/film_identity.go`
   (364 L : `Link`, `LinkMethod`, `LinkSource`, compteurs) devient le contrat de sortie du
   décodeur. Constat : un seul producteur (Halo Infinite) et un seul lecteur hors décodeur
   (`service/replayview`). Si un deuxième titre a un jour un décodeur, la conversion vers un
   type commun se fera côté LevelUp, au point d'entrée. Amende la promesse « inter-titres » de
   l'en-tête de ce fichier : à consigner dans l'ADR 0034 au plan D2.
4. **D3, noms : `internal/haloapi` + `internal/sync/filmfetch`.**
   - `internal/haloapi/xblauth` : chaîne Xbox Live → XSTS → Spartan (commune aux titres).
   - `internal/haloapi/infinite` : services Halo Infinite, transport, adresses, DTO bruts.
   - `internal/sync/filmfetch` : exploitation des films (téléchargement par `haloapi` →
     finalisation → cache local).
   - Traductions DTO → domaine : adaptateur sous `games/halo_infinite/`.
   - `platform/auth` garde l'exploitation des jetons (stockage, mise en commun, boucles).

## Découvertes hors périmètre (notées, non traitées)

- **Ratchet `archlint/doc_chemins_ai_test.go` rouge** (`TestCheminsAiCitesDansLeCodeExistent`,
  714 citations dans 561 fichiers) : le commit `324847a05` (« Archivage », 2026-10-26) a déplacé
  `.ai/V7.5/` sous `.ai/archive/V7.5/` sans mettre à jour les chemins cités dans le code. Les
  citations vérifiées pointent toutes vers un fichier présent sous `.ai/archive/V7.5/`.
  CLAUDE.md cite aussi `.ai/V7.5/` (README, référence des canaux d'équipement). Sans lien avec
  cet audit.

## Suite

Aucun calendrier : implémentation reportée par le user (2026-10-27). Le moment venu : un plan
par axe, dans l'ordre D1, D2, D3, chacun sur sa branche depuis la release en cours, pas sur la
release elle-même. Les décisions ci-dessus s'appliquent ; seule une décision neuve (dépôt
propre pour une brique) les rouvre.

## Addendum — vérification sur pièces du 2026-10-28 (avant le plan D1)

Relevé par `go list -deps`, grep des imports (production et tests) et lecture des ratchets
`archlint`, de `ci.yml`, du `Dockerfile` et du `Makefile`. Le fond de l'audit tient (trois
briques, ordre D1 → D2 → D3, coupes identifiées). Corrections et compléments :

1. **D1, faux** : `himap` importe aussi `internal/i18n` (`uslg.go:51`, `LexiqueLieux` rend des
   `i18n.Text`). Deuxième coupe de production, à côté de `mapvar`.
2. **D2, manque** : `film/replay` importe `internal/i18n` dans 5 fichiers (`MarshalFlat`,
   `UnmarshalFlat`, `FlatLayout`, `Text` : libellés du document de rejeu). Absent du tableau.
3. **D2, conflit** : le traitement 1 (types de sortie sous le décodeur) ferait importer un paquet
   de titre par `internal/analysis` (`highlightevent` lu par `analysis`, `equipmentusage` par
   `analysis/{sessionusage,squademprise,trends}`) — interdit par
   `archlint/no_title_package_in_analysis_test.go` (ADR 0012). À trancher au plan D2.
4. **D3.5, nuance** : `oauth_refresh.go` importe `observability/logging` + `platform/netguard`,
   `xsts.go` `observability/logging`, `xbox_device_code.go` `platform/netguard`. La coupe du
   protocole demande une injection (client HTTP, journal), pas un simple déplacement.
5. **Décision 1, coût non compté** : un module Go séparé ne peut PAS importer
   `levelup/go-api/internal/...`, tests compris. Dix tests `gamefiles` de `himap` lisent des
   données de LevelUp (`replay.LoadMapObjectives`, `data/cache/mvar`, `testfixtures`, `testutil`)
   et doivent passer côté LevelUp. L'infrastructure suit : `Dockerfile` (copie `apps/go-api`
   seul), `ci.yml` (`working-directory: apps/go-api` partout, filtre `paths` sans `libs/**`,
   énumération `go list ./...` des shards et de la baseline, lint), `Makefile`, `release.yml` ;
   les ratchets `archlint` sont racinés sur `apps/go-api` (`apiModuleRoot`) et perdraient de vue
   les briques (dont `gamefiles_tag_test`, 60+ tests concernés). Estimation D1 revue : ~3 jours.
6. **Décision 1, chemin de module** : « importable de l'extérieur par tag » suppose un chemin de
   module résolvable (`github.com/<dépôt>/libs/<brique>`), pas `levelup/...`.
7. **CI rouge** : (a) `archlint/doc_chemins_ai_test.go` échoue dans l'arbre de travail local :
   `.ai/PLAN_I18N_MULTILANGUE.md` a été déplacé sous `.ai/V8/` sans commit, deux citations du
   code pointent l'ancien chemin ; (b) la CI de `release/v8.1.0` est rouge côté web (garde-rails
   i18n `no-locale-literal-comparison` et `no-intl-tag-literal`, dépassements de 5 s) — chantier
   i18n, pas D1, mais préalable à toute PR D1 verte.
8. **Confirmé** : `himodule`, `hinavmesh`, `ooz`, `mapvar` n'importent rien du dépôt ;
   `canonical` n'est utilisé par le film qu'au travers des symboles de `film_identity.go`
   (décision 3 tenable) ; les types de sortie `domain/*` n'importent rien (feuilles) ; la
   sensibilité des révisions de couche aux chemins reste À VÉRIFIER au plan D2
   (`grammar/registry_fingerprint.go`).

Plan D1 : `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`.

## D1 exécuté le 2026-10-28

Plan : `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md` (lots 0 à 5) ; décision : `docs/adr/0041-go-modules-under-libs.md`.

- **Deux modules** : `libs/gamefiles` (`github.com/JGtm/LevelUp/libs/gamefiles` : `himodule`, `ooz`, `hinavmesh`, `mapvar`, rien du dépôt) et `libs/himap` (`github.com/JGtm/LevelUp/libs/himap`, paquet `libs/himap/himap`, importe `libs/gamefiles` seulement). Reliés à `apps/go-api` par un `go.work` racine (`go.work.sum` versionné), SANS `require` ni `replace` de module voisin (ADR 0041 D2) : `go mod tidy` est inutilisable dans `apps/go-api` et dans `libs/himap`.
- **Ratchets posés** : `archlint/briques_imports_test` (table brique -> paquets + imports permis, allowlist vide, entrée morte refusée), `deploy_root_copie_unique_test` (chemin du jeu en une seule copie, porté de la brique), `dockerfile_go_version_test` (même ligne `go` partout), garde-rail de licence GPLv3 de `cmd/server` (suit tous les modules de `libs/`), plus `TestModulesLibsSurveillesParDependabot` et `TestCibleMakefileGamefilesCouvreLeCorpus`.
- **Preuve du lot 5** : cinq fonds recuits identiques à l'octet (PNG aux témoins, WebP aux fichiers publiés), corpus `gamefiles` entier identique test par test à la base (un seul rouge connu, `TestRenduForgeVagabond`, poste sans `.ai/re_dump`), mêmes 212 paquets `ok` de `go test ./...`, `-tags=integration -p 1 ./internal/sync/...` vert, `gate-push` joué commande par commande (le `$(MAKE)` GnuWin32 du poste échoue : découvertes 28 et 33).
- **Pour D2 (décodeur de film)** : 20 fichiers de `film/internal/grammar` et `film/replay` citent encore `C:\Users\Guillaume\Projects\LevelUp` (commandes `ZONE_FILM`, chemins par défaut de `vehicules_v2*` ; découverte 9) ; les mêmes contraintes de module que D1 valent (pas d'import de `go-api/internal`, ratchets à racine d'espace de travail, une seule boucle `libs/*/go.mod` pour Makefile et CI).
- **Pour D2 et D3, découvertes de D1 qui les concernent** : 18 (retraits de baseline JSONL à poser en même temps que le déplacement d'un paquet) ; 19 et 27 (Dependabot `gomod` charge un module à voisin sans `go.work` : décision neuve si ça échoue) ; 20 (`.air.toml` ne surveille pas `libs/`) ; 22 (hooks `go-vet.sh`, `govulncheck.sh` limités à `apps/go-api`) ; 23 (deux copies de remontée de chemin `hinavmesh`/`mapvar`, une 3e impose helper + garde-rail) ; 24 (artefact CI périmé après `gh run rerun --failed`) ; 28 (`$(MAKE)` GnuWin32) ; 29 (commentaires de `himap` qui citent l'hôte) ; 30 (`golang.org/x/sys` tenu à la main dans trois `go.mod`, sans garde-rail de version).

## D2 exécuté le 2026-10-29

Plan : `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D2_2026-10-28.md` (lots 0 à 5) ; décisions : `docs/adr/0041-go-modules-under-libs.md` (troisième module, rang, D7, D8) et `docs/adr/0034-film-decoder-profile-and-layers.md` (amendements du 2026-10-29).

- **Un module** : `libs/film` (`github.com/JGtm/LevelUp/libs/film`, `go 1.26.10`). Paquets au premier niveau : `decfilm`, `replay`, `filmcache`, `finalise`, `types`, `damagetag`, `killicon`, `medalname`, `revision`, `filmproc`, `filmshell`, `equipenergy`, `identity`, `playerposition`, `objectiveevent`, `highlightevent`, `filmobs`, `digest`, `research/...` ; fermés par `internal/` : `source`, `profile`, `grammar/...`, `facts/...`, `constat`, `atomicfile`, `filelock`, `flatjson`. Importe `libs/gamefiles` (`mapvar`) et `libs/himap`, rien du dépôt (`go list -deps -test ./...` : aucun `levelup/go-api`). Reliée à `apps/go-api` par `go.work` sans `require` ni `replace` ; dépendances tierces écrites à la main aux versions d'`apps/go-api` (`go-toml/v2`, `x/sys`).
- **Décisions appliquées** : D2-A (un module, l'ancien dossier `film/` devient sa racine) ; D2-B (types de sortie : `playerposition`, `objectiveevent` déplacés dans la brique ; `domain/highlightevent` reste à LevelUp, la brique porte son propre type dans la feuille `highlightevent` et l'adaptateur de titre convertit ; familles d'équipement copiées dans `replay` avec un test de LevelUp qui refuse la divergence ; `film_identity.go` déplacé dans `identity`) ; D2-C (`filmshell`, `equipenergy` dans la brique) ; D2-D (compteurs `filmobs` + puits `SetSink` branché dans `wire`, noms de compteurs inchangés ; `atomicfile`/`filelock` déplacés ou copiés avec test d'identité ; `filmproc` dans la brique ; libellés `replay.Text` + codec `flatjson` qui écrit le même fil JSON que `i18n.MarshalFlat`, conversions dans `replaylabels`) ; D2-E (148 fichiers de test couplés : R0 réécriture d'import, R1 `internal/filmcorpus/`, R2 helpers locaux `racineDepot` ; les 19 chemins `Projects\LevelUp` retirés) ; D2-G (garde-fou posé AVANT tout déplacement, allowlist VIDE à la clôture du lot 2) ; D2-H (rien d'autre ne bouge) ; D2-I (CI à lots PAR MODULE, baseline `tests_libs_film.jsonl`, lint, vulncheck, Film race, Dockerfile, Dependabot, Makefile, `.golangci.yml`) ; D2-J (ADR 0041, ADR 0034, CLAUDE.md, ARCHITECTURE_V6 EN/FR, COMMANDS EN/FR).
- **D2-F AMENDÉE** (découverte D2-L2-a) : les révisions des cinq couches n'ont pas bougé (`source-2026-09-16.2`, `profile-2026-10-16`, `grammar-2026-10-28`, `killsource-2026-10-18`, `objectives-2026-10-17` : aucun fait persisté périmé, aucune recuisson). Le cadre historique de `revision` rend l'empreinte insensible au chemin du module (le déplacement du lot 3 n'a rien changé) ; mais les paquets qui ont changé de dossier au lot 2 (`playerposition`, `objectiveevent`, `filmshell`) et le type d'événement propre à la brique ont changé légitimement les jetons de `grammar`, `killsource` et `objectives` : leurs goldens ont été régénérés UNE FOIS à révision constante (porte à deux verrous, entrée de chronique par couche) au 2.12. L'ORDRE des imports est dans le flux haché et hors du cadre : 15 sources hachées gardent l'ordre historique en groupes (D2-L3-a).
- **Ratchets posés ou rerootés** : table `archlint/briques_imports_test` (ligne `film`, allowlist vide, tolérances datées retirées au lot 2), exhaustivité de `libs/**` dans la table, `go list -deps -test` en CI et dans `make libs-test`, interdit direct et fermeture `go list -deps` d'`internal/analysis` (aucun paquet du décodeur), `TestCadreHistoriqueDesImports` et deux voisins dans `revision`, tests d'égalité des deux listes d'équipement et des deux codecs de libellés, test du branchement du puits des compteurs (`wire`), `archlint/copies_identiques_test` (copie d'`atomicfile` identique à l'original, `filelock` déplacé), règle 220 du lint prouvée active sur le module, une quinzaine de ratchets muets par le déplacement rebranchés sur la brique (mutations jouées).
- **Preuve (lot 5)** : T4 (`replay-corpus-gate --reference=base --base=a5bae50b4`, celui du lot 0) : 18 témoins sur 19 identiques (0 gain, 0 perte, 0 changement, banc de vérité `ok`), code 0 avec `-allow-missing` ; le 19e, `f75e7053` (`koth_collines`), est ABSENT du parc local et le gate sans `-allow-missing` sort en code 4 (propriété du poste, la base ne peut pas non plus le cuire) ; T1 `go test -json ./... -count=1` dans `apps/go-api` et `libs/film` (codes 0 et 0) : 20 264 verdicts, 0 fail, 218 paquets `ok` ; contre le lot 3 un seul couple de plus (le test du branchement du puits des compteurs) ; contre le lot 0, traduit, 70 couples disparus (la liste consignée au 2.12 : tests déménagés vers `filmcorpus`, `digest`, `filmproc`, `canonical`, `equipenergy`, `filmshell`, `filelock`), 0 nom de test disparu, 0 changement de verdict ; T2 202 fichiers de témoin (goldens de révision et de périmètre, `equivalence`, faits) identiques au lot 3 ; contre le lot 0, seuls diffèrent les 6 goldens régénérés au 2.12 ; `revision` 46 PASS, empreintes identiques ; T3 `replay-equiv` : « 20 different » comme à la base (références périmées, D2-L0-a), MAIS les 20 fichiers de digests obtenus ont les MD5 du lot 0 (1 360 lignes) ; intégration `-tags=integration -p 1 ./internal/sync/... ./internal/persist/...` code 0 (16 paquets `ok`) ; web typecheck et lint code 0 (27 avertissements préexistants), Vitest 10 570 tests passés (4 tests de lecture de sources du décodeur reracinés au lot 3, hypothèse « web non touché » tombée) ; `gate-push` joué commande par commande lint des quatre modules à 0 issue, `make libs-test` code 0, baseline de présence jugée pour `apps/go-api` (8 359 tests de baseline, tous présents) et pour `libs/film` (877, tous présents) ; piège de méthode : la capture PowerShell 5.1 du JSONL abîme les noms non ASCII (D2-L5-a) ; CI : voir le journal.
- **Pour D3 (API Halo), découvertes de D2 qui le concernent** : D2-L3-b et D2-L3-c (ratchets qui sautent au lieu d'échouer quand le dossier visé bouge ; ratchets propres à `apps/go-api` qui ne couvrent plus un module sorti : inventaire à faire, règle générale = espace de travail) ; D2-L3-h et découverte 29 de D1 (chemins de l'hôte dans les commentaires de la brique, avant tout tag) ; D2-L3-f (registre des replis qui nomme des fichiers de LevelUp) ; D2-L3-e (le lint `--new-from-merge-base` ne voit pas un renommage qui franchit la racine d'un module : prévoir l'exemption au même commit) ; D2-L3-a (l'ordre des imports entre dans une empreinte : toute brique qui porte des empreintes de sources doit le dire) ; D2-L3-g (`-timeout 60s` de `go-api-test`) ; D2-L2-d (`slog` sans contexte dans `filmproc`) ; et, héritées de D1, 19 et 27 (Dependabot sans `go.work`), 20 (`.air.toml` ne surveille pas `libs/`), 22 (hooks `go-vet.sh` et `govulncheck.sh`), 28 (`$(MAKE)` GnuWin32 : `make libs-test` appelé depuis `gate-push` échoue sur le poste), 30 (`x/sys` tenu à la main dans quatre `go.mod` maintenant). Décision 4 de l'audit : la conversion DTO -> domaine se fera sous `games/halo_infinite/`, comme la conversion `highlightevent` de D2. Le détail des reports est au `.ai/REGISTRE_REPORTS.md` (section « découplage D2 »).

## D3 exécuté le 2026-10-29

Plan : `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D3_2026-10-29.md` (lots 0 à 6) ; décision : `docs/adr/0042-halo-api-protocol-vs-exploitation.md` (amende l'ADR 0031 D-1, précise l'ADR 0023).

- **Paquets, pas de module** (décision 4 de l'utilisateur, D3-A) : la brique « API Halo » est `apps/go-api/internal/haloapi` (parent : `Doer`, `Garde`, `ErrDefinitif`, `NonRetryable`), `haloapi/xblauth` (la chaîne d'échange, 13 fichiers de production, 1 528 L) et `haloapi/infinite` (adresses, transport, un fichier par service, DTO JSON bruts, octets de film compressés : 18 fichiers, 2 517 L). Exploitation : `games/halo_infinite/haloadapter` (23 fichiers, 3 206 L : `Provider` et `Client`, conversions DTO -> `domain`, caches, `netguardDoer`), `sync/filmfetch` (6 fichiers : décompression, finalisation, cache local, version de film), `platform/auth/tokencache` (cache de jetons de 50 minutes, `RetryWithFreshTokens`), `internal/duration` (feuille lue aussi par `games/halo_5`). `platform/halo` et `sync/haloclient` n'existent plus ; `platform/auth` garde tous ses noms exportés (239 symboles vérifiés à la compilation) et délègue à la brique par des façades `xbl_*.go`.
- **Décisions appliquées** : D3-A (forme interne ; le garde-fou G prend la forme d'une table des briques INTERNES, `archlint/briques_internes_test.go`, allowlist vide) ; D3-B à D3-D réécrites par le relevé du lot 0 (D3-L : `xblauth` reçoit le client Azure, le descripteur de titre, le créneau XBL, la provenance et les compteurs sous forme de paramètres et d'issues typées ; `refresh_user_xsts.go` et `azure_credentials.go` restent, ce sont de l'exploitation) ; D3-E (tests : `httptest` purs dans la brique, valeurs de domaine dans l'adaptateur, aucun nom disparu) ; D3-F (rien d'autre ne bouge : stores, pool, `netguard`, `games.DefaultEndpointResolver`, `games/halo_5`) ; D3-G (témoins T1 à T4) ; D3-H (revue) ; D3-I (docs : ADR 0042, 0023, 0031, `CLAUDE.md`, `ARCHITECTURE_V6` EN et FR).
- **Décisions AMENDÉES en cours de route** : D3-J (transport unique, les trois politiques de reprise conservées comme VALEURS `infinite.Retry`, aucune convergence) ; D3-K (garde `netguard` par un `Doer` injecté par surface, erreur définitive `NonRetryable` ; amendée au lot 5 : interface optionnelle `haloapi.Garde` consultée AVANT le limiteur, pour les trois politiques et les blobs) ; D3-M (périmètre borné aux trois paquets d'origine : `mapcatalog/ugc.go`, `assets/fetcher_gamecms.go`, `games/halo_5/client.go`, `sync/xbox_client.go` ne bougent pas) ; D3-N (un outil de témoin `cmd/diag_api_lecture`, lecture seule, sept sous-commandes) ; D3-O (« aucun ajout ni élargissement d'allowlist, renommage de clés une pour une » à la place de « allowlists inchangées ») ; D3-P (tiers permis : `errgroup`, et `rate` en test) ; D3-Q (le funnel `sync/haloclient_reexport.go` re-pointé, 29 fichiers de 21 paquets inchangés). Les décisions « DTO <-> domaine par réflexion » (4.2) n'ont pas d'objet : aucun client ne décodait dans un type `domain` (D3-L3-a).
- **Ratchets posés** : table des briques internes (imports permis par sous-arbre, tolérances datées vidées, auto-preuves) ; fermeture de `domain`, `analysis` et `games/canonical` sans `haloapi` ; `haloapi_sans_client_http` (aucun `http.Client`, `Transport`, `DefaultClient`, `Get/Post/Head` ni `RoundTrip` dans la production de la brique, jugé sur l'AST) ; `TestBriqueInfiniteRecoitLeDoerGarde` (tout `Doer` donné à `infinite.New` / `NewClassement` est un littéral `netguardDoer{...}`) ; `TestXblauthSansVariableDePaquet`, `TestHaloapiSansEnvNiFichier`, `TestHaloapiCinqParametresAuPlus` ; `racinesOctetsBruts` += `internal/haloapi` (la brique ne décompresse pas un film) ; exemption par préfixe `haloapi/` du ratchet de couverture netguard, quatre clés d'allowlist retirées, huit surfaces prouvées muettes en démo (`TestNetguard_HuitSurfacesGardees`). Les ratchets d'authentification n'ont pas bougé : `sentinel_test.go` (guards 2 et 3, allowlists VIDES), `sync/no_legacy_source_used_test.go`, `no_pool_hasplayer_gate`, `no_request_scoped_halo_provider` (vise `haloadapter`), `no_halowaypoint_literal` (une seule entrée pour la brique, dix retirées).
- **Preuve** : T1 `go test -json ./... -count=1` dans `apps/go-api` : 336 paquets, 16 531 tests et sous-tests ; contre le lot 0, 447 paires portées sous le même nom dans `xblauth` (69), `infinite` (35), `haloadapter` (258), `filmfetch` (60), `tokencache` (16), `duration` (9), aucun nom disparu, aucun pass -> skip ; un échec isolé (`mapcatalog`, verrou de fichier Windows, passe en relance, paquet non touché : D3-L6-d). T3 LIVE lecture seule, compte JGtm, une exécution par surface par le chemin canonique de rafraîchissement (jamais de re-capture) : historique, stats, skill, manifeste, actif de découverte et service record rendent exactement les mêmes octets (MD5) que le témoin pris avant tout déplacement ; le calendrier répond 403 avant comme après (D3-L1-a). T4 : dix noms de compteurs (cinq `levelup.auth.oauth_refresh_*`, `halo_api_blob_retry_success`, `halo_api_blob_retry_exhausted`, `film_fetch_chunk_incomplet`, `leaderboard_empty_page1`, `log_throttle_halo_api_download_blob`) identiques, relevés par grep avant et après. Intégration `-tags=integration -p 1 ./...` : code 0, 337 paquets (197 ok, 140 sans test), 18 454 tests et sous-tests (18 355 pass, 99 skip, 0 fail) ; la baseline de présence (`check_test_baseline.sh tests --from-jsonl`) retrouve ses 8 114 tests, aucun échec de test ni de paquet. Revue adversariale : une ronde à trois relecteurs aveugles (0 P0, 4 P1, 11 P2 recevables, 74 conditions vérifiées qui tiennent), corrections avec test vu rouge ; seconde ronde sur les seules corrections : 0 P0, 0 P1, 3 P2 (documentation de la garde, seuil de `compterReprise` testé, effet de course consigné). Lint des quatre modules à 0 issue, `make libs-test` code 0, `go list -deps` de la brique sans paquet LevelUp hors d'elle.
- **Hors périmètre, consigné** (`.ai/REGISTRE_REPORTS.md`, section « découplage D3 ») : convergence des trois politiques de reprise ; les quatre autres clients HTTP Halo ; le singleton `games.DefaultEndpointResolver` ; la garde `netguard` des surfaces d'authentification ; le calendrier de saisons en 403 ; le client Azure public qui échoue à chaque rafraîchissement de JGtm (repli MSA natif) ; un secret en commentaire de `.env.local` (décision du user) ; les anti-motifs recopiés dans l'adaptateur et les tests qui sortent sur le réseau.
- **Clôture de l'audit** : les trois axes D1, D2, D3 et le garde-fou G sont exécutés : audit CLOS. Le garde-fou G existe sous deux formes, la table des briques-modules (`briques_imports_test.go`, ADR 0041) et la table des briques internes (`briques_internes_test.go`, ADR 0042). Pour la suite (une brique qui devient module, un `haloapi/halo5`, une politique de reprise unique), c'est une décision neuve.
