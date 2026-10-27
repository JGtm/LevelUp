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
