# Big Team Battle 2025 — pourquoi le decodeur n'attribuait presque aucune source de degat

Date : 2026-09-12 · Branche : `wt/btb-2025-abstention` (base `feat/v75` 2f5d165be)
Perimetre : une session, borne. Correctif livre (la cause est simple et prouvee).
Mise a jour du 2026-09-12 (lot G.2bis) : le correctif retenu N EST PLUS la resolution par
mesure decrite en section 4 — voir la section 6, qui fait foi.

## 1. Verdict

**Cause (a), et une seule : le format du film change sur la periode, et le decodeur ne le voit
pas.** Precisement : les films Theater de mars a novembre 2025 portent une version de bloc
d'evenement (`FilmMajorVersion` 39-40) ou le gamertag vit a l'octet 12 du bloc de 60 octets, et
non a l'octet 0. `killsource.loadKillFeed` appelle `analysis.ParseHighlightEvents(chunk, 0)` —
0 = version inconnue, traitee jusqu'ici comme « gamertag en tete ». Sur ces films la lecture
ramenait donc du rembourrage : **2 gamertags distincts pour 24 a 27 joueurs**.

La chaine de consequences est entierement mecanique :

| etape | effet |
|---|---|
| `buildFeed` | tous les XUID dont un event porte le rembourrage recoivent le MEME nom |
| `rosterNames` | 10 a 12 noms distincts au lieu de 24 a 27 |
| `buildRoster` | `nPlay = len(noms)` — la borne de l'espace des indices s'effondre |
| `scanFilm(nPlay)` | portes T1/T2 « indice < nPlay » : 3 candidats sur 4 rejetes |
| `selectCredible` | meme filtre sur la marche : 268 refus « victime hors roster » sur 345 |
| resultat | la mort existe, personne ne peut la nommer -> aucune source de degat publiee |

Hypotheses ecartees, sur pieces :

- **(b) vehicules** — le refus se produit AVANT toute lecture de tag ; les tags refuses sont de
  toutes familles, et les temoins 2026 (memes vehicules, meme playlist) sont a 92 %.
- **(c) cartes hors catalogue de bornes** — les MEMES cartes sont bonnes ailleurs : Oasis 0.5 %
  en 2024-10 contre 85 % en 2025-03 ; Refuge 7.9 % en 2026-01 contre 88.8 % en 2025-03.
- **(d) chunks tronques** — les 8 films examines ont un manifeste complet et autant de fichiers
  que d'entrees, chunk HIGHLIGHT (type 3) compris. Les 92 films du parc BTB sont tous en cache.
- **(e) autre** — la correction du seul decoupage du gamertag rend la couverture, sans toucher
  une ligne du decodeur de dead-state.

Note : l'empreinte du registre ECS (`filmdec.RegistryFingerprint`) sonne bien sur ces films,
mais elle sonne AUSSI sur le temoin 2024 qui est a 99.5 % de couverture. Elle n'est donc pas le
discriminant, et le signal qu'elle porte etait ici une fausse piste.

## 2. Preuve — le parc BTB entier, 92 films sur 92

Pour chaque film : nombre de gamertags DISTINCTS que rend le chunk HIGHLIGHT sous chacun des
deux decoupages. Le decoupage gagnant date le film ; il est croise avec le taux de morts sans
source mesure par le pilote en base.

| mois | decoupage gagnant | films | % moyen de morts sans source |
|---|---|---:|---:|
| 2023-04 | en tete (<=38 / >=41) | 2 | 3.6 |
| 2023-05 | en tete | 1 | 0.0 |
| 2023-07 | en tete | 2 | 1.1 |
| 2023-09 | en tete | 1 | 2.0 |
| 2024-10 | en tete | 4 | 6.8 |
| 2024-11 | en tete | 2 | 3.6 |
| **2025-03** | **decale (39-40)** | **14** | **87.6** |
| **2025-04** | **decale** | **9** | **87.5** |
| **2025-05** | **decale** | **4** | **86.1** |
| **2025-06** | **decale** | **2** | **88.6** |
| **2025-07** | **decale** | **23** | **90.4** |
| **2025-08** | **decale** | **13** | **90.7** |
| **2025-10** | **decale** | **5** | **79.1** |
| **2025-11** | **decale** | **4** | **74.7** |
| 2026-01 | en tete | 2 | 9.6 |
| 2026-05 | en tete | 1 | 0.0 |
| 2026-07 | en tete | 3 | 11.8 |

**Separation parfaite.** Les 74 films « decale » sont exactement ceux dont le taux de morts sans
source va de 68.1 a 96.5 % ; les 18 films « en tete » vont de 0.0 a 23.5 %. Aucun recouvrement.

Reproduction : `KS_BTB2025_ROOT=<cache>/film_chunks KS_BTB2025_IDS=<id8,...> go test
./internal/games/halo_infinite/film/killsource/ -run TestBTB2025KillFeedNoms -v`.

## 3. Preuve — huit films instrumentes, avant / apres

Trois films 2025 a plus de 85 % sans source (mars, juillet, octobre), deux films 2025
supplementaires, et trois temoins attribues (2024, 2026-01, 2026-07). Couverture =
`Coverage.Covered / Coverage.RealPairs`, telle que `killsource sante` la publie.

| film | date | carte | roster humain av. -> ap. | candidats du scan av. -> ap. | couverture av. -> ap. |
|---|---|---|---|---|---|
| a26dbcdb | 2024-10 | Oasis (temoin) | 27 -> 27 | 237 -> 237 | 99.5 % -> **99.5 %** |
| 16c3cdbd | 2025-03 | Oasis | 11 -> 25 | 35 -> 157 | 15.0 % -> **97.1 %** |
| 443426df | 2025-03 | Refuge | 12 -> 26 | 69 -> 232 | 11.2 % -> **100.0 %** |
| 111fa685 | 2025-03 | Command | 10 -> 24 | 47 -> 207 | 6.8 % -> **94.8 %** |
| e5adf7b2 | 2025-07 | Fragmentation | 11 -> 26 | 92 -> 263 | 5.1 % -> **97.0 %** |
| b81f6415 | 2025-10 | Command | 12 -> 27 | 332 -> 332 | 17.3 % -> **82.4 %** |
| 58d09c44 | 2026-01 | Refuge (temoin) | 24 -> 24 | 435 -> 435 | 92.1 % -> **92.1 %** |
| 5676a9ba | 2026-07 | Insolence (temoin) | 26 -> 26 | 247 -> 247 | 92.5 % -> **92.5 %** |

Les trois temoins sont inchanges au dixieme de point, roster et candidats compris : la
resolution ne se declenche pas sur eux.

Detail d'un film, avant correctif (111fa685, 2025-03 Command, 191 morts au kill-feed) :

```
ROSTER   humains_kill_feed=10 nPlay=10
MARCHE   morts_brutes=345 credibles=32
REFUS    victime_hors_roster=268  tueur_hors_roster=44
INDICES  victime : 0..24 tous representes (7 a 11 occurrences chacun)
SCAN     candidats=47 (porte T1/T2 : indice < nPlay=10)
```

Les indices de victime couvrent bien 0..24 : les morts sont LA, lisibles, et c'est la seule
borne `nPlay=10` qui les jette.

## 4. Correctif

`internal/analysis/highlight_event_parser.go` — quand `filmMajorVersion` vaut 0 (version
inconnue : le manifeste en cache ne la porte pas, cf. `halo_client_film.go`), le scan decode les
deux decoupages du gamertag sur le MEME bloc de 60 octets deja retenu et garde celui qui rend le
plus de gamertags DISTINCTS. Une version DECLAREE fait toujours foi : la resolution ne s'applique
qu'a l'inconnu, et elle ne coute aucune seconde passe sur le chunk.

Le contraste mesure est sans zone grise (2 contre 24-27), ce qui rend le critere sur.

Trois appelants passaient 0 et beneficient du correctif : `killsource/feed.go`,
`analysis/replay/deaths_source.go`, `ops/medal_feed_backfill.go`. `sync/collect.go` et
`sync/engine_highlight_events.go` passent la version du manifeste : ils sont inchanges.

`KillSourceDecoderRev` : `killsource-2026-09-05` -> `killsource-2026-09-12`, pour que les lignes
deja en base redeviennent candidates au backlog.

### Tests

- `internal/analysis/highlight_event_parser_version_test.go` — trois tests CI, sans fixture : un
  flux 39-40 lu sans version est resolu ; un flux >= 41 lu sans version est identique a la
  lecture versionnee ; une version declaree n'est jamais « corrigee ».
- `internal/games/halo_infinite/film/killsource/btb2025_abstention_research_test.go` — banc de
  mesure garde par `KS_BTB2025_*` (il n'asserte pas), plus `TestBTB2025NonRegression` qui, lui,
  asserte roster >= 20 et couverture >= 80 % sur un film 2025.

### Gates joues

```
go vet ./...                                                              OK
go test ./internal/...                                                    OK
go test -tags=integration -p 1 ./internal/sync/... ./internal/persist/...  OK
```

## 5. Ce qui reste ouvert

1. **Les 136 autres films de version 39-40.** Le parc en cache compte 1 351 films, dont **210 de
   version 39-40** ; 74 sont les BTB de ce rapport. Les 136 autres (arene et divers) sont
   atteints par le MEME defaut — le mecanisme ne connait pas la playlist. La mesure « arene a
   ~0-10 % » du pilote agrege probablement toutes les annees : **prediction a verifier en base,
   par mois et par playlist**. Aucune requete n'a ete faite ici (consigne : ne pas ouvrir les
   DuckDB).
2. **Le redecodage n'est PAS lance** (consigne). La montee de `KillSourceDecoderRev` suffit a
   remettre les lignes concernees dans le backlog de `postsync`. Ordre de grandeur : 210 films a
   8-50 s = **30 min a 3 h** de decodage, en serie (contrainte d'un decodage a la fois dans le
   process).
3. **L'empreinte du decodeur ne couvre pas son amont.** `killSourceDecoderFingerprint` ne hache
   que `games/halo_infinite/film/killsource/` : ce correctif, qui change les lignes produites,
   vit dans `internal/analysis/` et n'aurait pas fait sonner le gate. Decouverte notee, non
   traitee (hors perimetre).
4. **La version du film n'est pas conservee dans le cache.** `film_manifests/*.json` ne porte que
   la liste des chunks. La resolution par mesure est donc la bonne reponse aujourd'hui ; ecrire
   `FilmMajorVersion` dans le manifeste au telechargement la rendrait inutile et serait plus
   direct — a arbitrer separement.

## 6. Correctif retenu : la version est LUE dans l en-tete, l heuristique est retiree

Decision utilisateur du 2026-09-12 : **un outil bien developpe cherche l indicateur cle qui lui
dit comment le film est construit, il ne le devine pas.** La resolution par mesure de `2a6265ac4`
(section 4) est REFUSEE et supprimee.

### 6.1 L indicateur, et il etait sous nos yeux

Les **quatre premiers octets de `chunk_00.bin`** — le registre du film — sont le
`FilmMajorVersion`, en u32 little-endian. C est la MEME valeur que l API publie dans
`CustomData.FilmMajorVersion` de son manifeste de spectate. Le second u32 suit (25 / 24 / 27 sur
les films de version 40 / 37 / 41), puis la chaine `game-engine-team-mapping-component` ouvre le
premier bloc d archetype.

Verifie sur pieces, en-tete brut :

```
e5adf7b2 (2025-07)  28 00 00 00 | 19 00 00 00 | "game-engine-team-mapping..."  -> v40
a26dbcdb (2024-10)  25 00 00 00 | 18 00 00 00 | idem                           -> v37
5676a9ba (2026-07)  29 00 00 00 | 1b 00 00 00 | idem                           -> v41
```

Le depot avait deja MESURE cette valeur sans la nommer : le commentaire de `looksZlib`
(`filmdec/registry.go`) la comptait sur les 1 378 `chunk_00` du cache et la lisait comme « le
`kind` u32 du premier slot ». Le comptage etait juste, l interpretation non. Les deux commentaires
concernes sont corriges.

### 6.2 Ce qui a ete livre

| item | etat |
|---|---|
| G.2bis-1 helper canonique `filmdec.FilmMajorVersionFromHeader` / `FilmMajorVersion` | `[x]` |
| G.2bis-2 les trois appelants passent la version lue | `[x]` |
| G.2bis-3 heuristique supprimee (0 code mort) | `[x]` |
| G.2bis-4 la version survit au cache disque (`fetchFilmManifest`) | `[x]` |
| G.2bis-4bis champ `film_major_version` dans le manifeste serialise | `[!]` justifie, cf. 6.5 |
| G.2bis-5 `SchemaVersion` 53 -> 54 + empreinte killsource recopiee | `[x]` |
| G.2 etendu (plan, 2026-09-12) : la version portee par le film charge et publiee dans la couverture | `[x]` |

**Le helper vit dans `internal/analysis/filmdec`** (`film_major_version.go`), et pas dans
`filmsource` : `filmsource` est volontairement aveugle au contenu (« les chunks bruts, et rien
d autre ») et `archlint/filmsource_leaf_test.go` le maintient feuille ; `filmdec` est le paquet
qui porte la semantique de `chunk_00` — `registry.go` l analyse, `FilmRegistryChunk` le localise.

**Les trois appelants** qui passaient `0` :

- `killsource/feed.go` — `loadFilm` pose `majorVersion` / `versionLue` sur le film, `prepare`
  consigne un `slog.WarnContext` quand le registre manque, `loadKillFeed` passe la version.
- `analysis/replay/deaths_source.go` — `ScanDeaths` lit la version du film deja charge, WARN idem.
- `ops/medal_feed_backfill.go` — `SourceChunkHighlight` rend desormais un
  `FilmHighlight{Chunk, MajorVersion}` ; l implantation CLI charge en plus le seul `chunk_00`
  (~2 Mio) pour lire l en-tete.

Aucune degradation silencieuse : registre illisible = WARN + version 0 = comportement historique.

**Ce qui a ete supprime** de `2a6265ac4` : `gamertagLayoutAlternatif`, la tranche `alternatif`, le
drapeau `auto`, les constantes `versionUnknown` / `versionGamertagEnTete` / `versionGamertagDecale`,
le troisieme retour `brut` de `parseEventAtBit`, et l enveloppe `gamertagPourVersion` (un seul
appelant restant, et sa justification ecrite portait sur la resolution retiree).
`highlight_event_parser.go` est redevenu identique a son etat d avant le correctif, a son
commentaire de contrat pres.

### 6.3 Le parc entier, par la version LUE

`TestBTB2025ParcVersions` (banc garde par `KS_BTB2025_PARC`) remplace `TestBTB2025ParcDecoupage` :
il classe par la version lue au lieu de mesurer.

```
PARC     films=1351 sans_registre_lisible=0
VERSIONS 31:3 33:3 37:10 38:1 39:26 40:185 41:1123
```

Exactement la distribution annoncee par le pilote. **211 films de version 39-40** sont concernes.

Croisement mesure contre version (`KS_BTB2025_PARC_CROISE=1`) : **1 divergence sur 1 351**, le film
`007d53a4` (v40). Inspection : ce film ne porte **aucun** highlight event — 0 sous les deux
decoupages. La mesure y est indefinie et retombe par defaut sur « en tete » ; la version, elle,
repond. C est exactement l angle mort d une heuristique par mesure, et une raison de plus de ne
pas en garder une.

### 6.4 Le contenu cuit du rejeu change — d ou le bump 53 -> 54

`ScanDeaths` alimente `gamertagsOf`, la table qui **nomme `roster[]`** dans l artefact de rejeu
(`build.go`, `identity_registry_section.go`) et qui sert aussi de table nom -> xuid a
`replaybuild/kills.go`. Mesure sur les memes octets, version passee `0` puis version lue :

| film | version lue | identites nommees | noms distincts |
|---|---:|---|---|
| `e5adf7b2` | 40 | 17 -> **26** | 2 -> **26** |
| `111fa685` | 39 | 16 -> **24** | 2 -> **24** |
| `000d5950` (temoin) | 41 | 8 -> 8 | 8 -> 8 |
| `5676a9ba` (temoin) | 41 | 26 -> 26 | 26 -> 26 |

Le contenu cuit change : la regle de `document.go` exige la montee, et c est elle qui rendra ces
artefacts candidats a `backfill-replay --only-existing`. Chronique datee posee dans
`document_chronicle.go` et dans le garde de `structure_test.go`, selon la convention.

Le golden d assemblage `000d5950` change d **une seule ligne** (`schema 53` -> `schema 54`) :
609 lignes figees, 609 obtenues, un seul ecart — ses entrees etant figees dans
`inputs_000d5950.bin.gz`.

`KillSourceDecoderRev` reste `killsource-2026-09-12` (deja monte) ; `killSourceDecoderFingerprint`
est recopiee, `loadFilm` / `loadKillFeed` / `prepare` ayant bouge et les lignes produites aussi.

### 6.5 Le manifeste en cache : ce qui a ete fait, et ce qui ne l a pas ete

**Fait.** `fetchFilmManifest` posait `CustomData.FilmMajorVersion = 0` des que le manifeste venait
du cache local. Consequence non vue jusqu ici : `GetHighlightEventsChunk` servait `0` a **tout le
pipeline de synchronisation** pour ces matchs — le chemin en direct compris, qu on croyait
epargne. La version est desormais lue dans l en-tete du registre, ce qui vaut pour les 1 351 films
deja en cache, **sans migration**. Deux tests sans reseau le figent.

**`[!]` non fait, et c est delibere :** aucun champ `film_major_version` n est ajoute au manifeste
serialise. Trois raisons : il serait redondant avec l en-tete du registre (seconde source de
verite pour le meme fait) ; il ne couvrirait **aucun** des 1 351 films deja en cache, `filmcache.Write`
ne reecrivant jamais un manifeste existant ; et l objectif fonctionnel vise — « `GetFilmChunks`
rend la vraie version aussi depuis le cache » — est atteint sans lui.

### 6.6 La version voyage avec l artefact (Lot G etendu, ligne du plan du 2026-09-12)

Constat du plan : seul le parseur des temps forts consomme `FilmMajorVersion` ; `filmdec` et le
constructeur de rejeu travaillent sous l hypothese implicite « tout est en 41 », alors que le
cache porte SEPT versions. Un artefact ne disait donc pas sous quelle grammaire il avait ete cuit,
et la mesure par version (lot H) aurait exige de relire les films.

`Options.FilmMajorVersion` (pointeur) est pose par `BuildFromFilm` depuis
`filmdec.FilmMajorVersion(film)`, republie tel quel dans `Coverage.FilmMajorVersion`
(`coverage.filmMajorVersion` dans le JSON). Pointeur et non zero : l absence dit « film sans
registre, ou artefact anterieur a ce lot », jamais « version 0 ».

Telemetrie pure, champ OPTIONNEL : il n aurait pas exige de montee a lui seul, il part avec le
bump 54 deja acquis. Miroir `replaydoc` + projection `replayview` (garde-rail de parite), contrat
regenere par `make openapi-gen` et types web regeneres par `openapi-typescript` : un champ
optionnel de plus des deux cotes, rien d autre.

### 6.7 Portes jouees

```
go build ./...                                         OK
go vet ./...                                           OK
go test ./...                                          171 paquets OK, 0 echec
go test -tags=integration -p 1 ./internal/sync/...      OK
make go-api-lint (golangci-lint, ratchet CI)            0 issues
make openapi-gen                                        api/openapi.yaml : +3 lignes
openapi-typescript -> apps/web/.../generated.ts          +2 lignes (champ optionnel)
golden assembly_000d5950                                regenere, 1 ligne (schema)
replay-corpus-gate --reference=base --base=feat/v75     8 temoins / 8 ok, 0 perte, exit 0
```

`make generate-types` n a PAS pu tourner dans ce worktree (`apps/web/node_modules` absent) : les
types ont ete regeneres par le meme binaire `openapi-typescript` 7.13.0 emprunte a l arbre
principal, sur le MEME `openapi.yaml`. `make check-types` (tsc) et `make test-web` n ont pas ete
joues pour la meme raison — le changement cote web est un champ OPTIONNEL de plus dans un fichier
genere, et la CI reste le gate d autorite.

Gate corpus, detail (`base(feat/v75) 53 -> HEAD 54`, gains / pertes) :

```
bcb6d393  ctf_mono_manche      5 / 0     (film v40 — affecte)
fb1a1a72  ctf_multi_manche     3 / 0
d9781168  oddball              3 / 0
c75f33b8  assaut_bombe         3 / 0
bf15f7ab  slayer               3 / 0
51ebbc0f  deux_manches         3 / 0
084a804d  vehicules           17 / 0     (film v39 — affecte)
0797ce72  region_index_2_bits  3 / 0
```

Les deux temoins de version 39-40 (`bcb6d393` v40, `084a804d` v39) sont exactement ceux qui
gagnent plus que le socle commun ; les six temoins de version 41 portent le socle et rien d autre.
Avant l ajout de `coverage.filmMajorVersion`, le meme gate donnait 3 / 1 / 1 / 1 / 1 / 1 / 15 / 1 :
chaque temoin a gagne exactement deux differences de plus — la ligne de schema et ce champ.
**Zero perte, aux deux passages.**

### 6.8 Ce qui reste ouvert

1. **Le redecodage n est pas lance** (consigne du lot) : `backfill-replay --only-existing` pour
   les artefacts de rejeu, et le backlog `killsource` par `KillSourceDecoderRev`. Ordre de
   grandeur : 211 films a 8-50 s, soit 30 min a 3 h en serie.
2. **Les 136 films 39-40 hors BTB** (arene et divers) restent a verifier en base, par mois et par
   playlist. Aucune requete DuckDB n a ete faite ici (consigne).
3. **L empreinte du decodeur ne couvre pas son amont.** `killSourceDecoderFingerprint` ne hache
   que `killsource/` : ce correctif a commence dans `analysis/` et `filmdec` et n aurait pas fait
   sonner le gate si `loadFilm` / `loadKillFeed` n avaient pas bouge aussi. Decouverte notee sur
   place dans `collector.go`, non traitee (hors perimetre).

## 7. Revue adversariale, ronde 1

Relecture par contexte frais du 2026-09-12. Neuf constats, tous verifies PAR MUTATION par le
relecteur : sur les quatre constats P1, la mutation ne faisait rougir AUCUN test de la CI. Les neuf
sont corriges ci-dessous, et chaque mutation a ete rejouee ici (rouge, puis vert au retour).

### 7.1 Les trois appelants n avaient aucun temoin — P1-1, P1-2, P1-3

**La cause est la meme pour les trois, et elle est mecanique.** La seule bobine versionnee du depot,
`minibobine_000d5950`, vient d un film de version 41. Or `decodeEventBytes` decoupe le gamertag a
l octet 0 pour `version <= 38 || version >= 41` : sur un film v41, passer 0 ou passer 41 donne LE
MEME decoupage. Le correctif « la version est lue » n avait donc, par construction, aucun temoin
capable de le defaire bruyamment.

**Correctif : une bobine de version 40, versionnee.** `killsource/testdata/minibobine_e5adf7b2` —
trois chunks, **876 Kio** :

| fichier | contenu | pourquoi celui-la |
|---|---|---|
| `chunk_00.bin` | le REGISTRE du film `e5adf7b2` | c est lui qui PORTE la version (u32 LE en tete) |
| `chunk_01.bin` | le premier chunk de DONNEES | `loadFilm` refuse un film sans paquet type-0 |
| `chunk_02.bin` | le chunk HIGHLIGHT (n30), trouve par son CONTENU | le kill-feed |

Origine : `data/cache/film_chunks/e5adf7b2` (2025-07, Fragmentation, Big Team Battle, version 40 —
l un des cinq films instrumentes en section 3). Recette executable, comme pour la bobine de rejeu :
`KILLSOURCE_FIXTURES=<racine> go test ./internal/games/halo_infinite/film/killsource/ -run
TestMiniBobineV40Regenerer -update`. **Un seul ecart avec la recette de `minibobine_000d5950`** :
les octets sont recompresses en zlib (le cache local les stocke desormais decompresses ; recopies
tels quels ils pesaient 3,0 Mio). Leur contenu DECOMPRESSE est celui du film, inchange, et
`filmsource` les inflate au chargement exactement comme il inflate ceux de la bobine v41.

Le contraste mesure sur cette bobine ne laisse aucune zone grise :

| appelant | test CI | version lue (40) | mutation « version 0 » |
|---|---|---|---|
| `killsource/feed.go` | `TestMiniBobineV40RosterSuitLaVersionLue` | roster 26, dont 26 gamertags | roster 11, dont 2 gamertags |
| `replay/ScanDeaths` | `TestScanDeathsSuitLaVersionDuFilm` | 199 morts, 26 noms distincts | 199 morts, 2 noms distincts |
| `ops/medal_feed_backfill` | `TestEventsDuFilmSuitLaVersionDeclaree` | gamertag intact | gamertag decale |

Les planchers des deux premiers tests sont a 20 : assez haut pour que la mutation les creve, assez
bas pour ne pas figer un compte exact qu une correction de parseur ferait bouger.

**Le troisieme appelant a demande une couture, et il faut dire pourquoi.** L appariement des
medailles se fait sur le couple (xuid, instant) — deux champs lus HORS du bloc de 60 octets — alors
que la version ne commande que le decoupage du GAMERTAG a l interieur du bloc. **Aucune correction
ecrite par cette passe ne depend de la version** : repasser 0 ne changeait donc rien d observable,
et c est la raison de fond pour laquelle le constat P1-3 existait. Une fonction d une ligne,
`eventsDuFilm(FilmHighlight)`, isole le geste « parser avec la version declaree » et lui donne un
point d observation ; deux tests l encadrent, dont une contre-epreuve (version 0 sur le meme bloc
39-40 ne rend PAS le nom) sans laquelle le premier ne prouverait rien.

### 7.2 Le gate de revision ne tenait qu un de ses deux gestes — P1-4

`TestKillSourceDecoderRevSuitLeDecodeur` comparait la seule EMPREINTE a la constante
`killSourceDecoderFingerprint`. Remettre `KillSourceDecoderRev` a sa valeur d avant, en gardant la
nouvelle empreinte, restait VERT — alors que le message d echec du test declare les deux gestes
« OBLIGATOIRES ».

Correctif : `killcollector/testdata/killsource_decoder_rev.golden` fige le couple
`revision<TAB>empreinte`, porte sa recette et l historique des mouvements ; la constante disparait
(une seule source de verite, pas deux copies a tenir en phase). Le test distingue desormais deux
echecs :

| observation | message | geste attendu |
|---|---|---|
| empreinte differente | « LE DECODEUR A CHANGE » | decider, bumper si les lignes bougent, regenerer |
| empreinte egale, revision non | « LA REVISION A CHANGE SANS QUE LE DECODEUR BOUGE » | regenerer si le changement vit en amont, sinon remettre la revision |

Regeneration explicite, citee dans les deux messages :
`go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur -update`.

Trois mutations jouees : revision seule -> **rouge** (second message) ; decodeur seul (une ligne de
commentaire ajoutee a `feed.go`) -> **rouge** (premier message) ; les deux ensemble avec golden
regenere -> **vert**.

### 7.3 Documentation et journalisation — P2-1 a P2-5

| constat | correctif |
|---|---|
| P2-1 | le paragraphe du 2026-09-12 s etait colle au doc-comment de `fetchFilmManifest` et devenait celui de `filmMajorVersionDuCache`, laissant `fetchFilmManifest` sans doc. Les deux docs sont remises sur leur fonction. |
| P2-2 | `FilmRegistryChunk` n a plus un unique lecteur. La phrase dit maintenant ce que chacun lit : `FilmContext.Registry` ANALYSE le registre (archetypes et slots), `FilmMajorVersion` n en lit que l u32 de tete. |
| P2-3 | le godoc de `ParseHighlightEvents` attribuait la version au seul manifeste. Il nomme les DEUX sources de la meme valeur (u32 en tete du registre, `CustomData.FilmMajorVersion` de l API) et dit que 0 = inconnue = decoupage « en tete », a la charge de l appelant de consigner. |
| P2-4 | le WARN « registre absent » de `ScanDeaths` ne portait pas `match_id` et sortait DEUX fois par cuisson (`replaybuild.lireMorts` puis `BuildFromFilm`). Il monte chez `BuildFromFilm` — le seul appelant du chemin qui connaisse le match ET qui lise deja cette version pour la couverture. Aucune signature ne change, aucun log n est perdu : le chemin du collecteur (`killcollector/positions.go`) garde le sien, consigne par `killsource.prepare`. |
| P2-5 | `Coverage.FilmMajorVersion` etait publie sans qu aucun test n en asserte la valeur. Deux tests : version presente -> la couverture porte 40 ; version absente -> `nil`, jamais 0 (le pointeur distingue « on ne sait pas » de « version 0 », qui est signifiante ailleurs). Mutation sur la ligne de publication : rouge, puis vert. |

### 7.4 Portes jouees

```
go build ./...                                          OK
go vet ./...                                            OK
go test ./...                                           171 paquets ok, 0 echec, exit 0
go test -tags=integration -p 1 ./internal/sync/...      OK
make go-api-lint (golangci-lint, ratchet CI)            0 issue
golden assembly_000d5950                                INCHANGE (aucun -update)
golden minibobine_000d5950 et familles filmdec          INCHANGES
```

**Gate corpus NON rejoue, et c est justifie** : aucun changement de contenu cuit n est attendu de
cette ronde. Le seul fichier de production dont le comportement bouge est le WARN de P2-4 (une
ligne de journal, portee par un autre appelant) ; `eventsDuFilm` est une extraction sans changement
de valeur, et les cinq autres constats sont de la documentation, des tests ou un fichier de fixture.
`SchemaVersion` reste 54, `KillSourceDecoderRev` reste `killsource-2026-09-12`, et l empreinte du
decodeur est inchangee — le golden d assemblage `000d5950` le confirme sans une seule ligne d ecart.

### 7.5 Taille ajoutee

| element | octets |
|---|---|
| `minibobine_e5adf7b2/chunk_00.bin` | 445 742 |
| `minibobine_e5adf7b2/chunk_01.bin` | 116 319 |
| `minibobine_e5adf7b2/chunk_02.bin` | 334 509 |
| `minibobine_e5adf7b2/PROVENANCE.txt` | 948 |
| **total fixture** | **897 518 octets (876 Kio)** |

A comparer aux 3,8 Mio de `minibobine_000d5950` : la bobine v40 ne garde que ce qu il faut pour
lire un kill-feed, la bobine v41 garde un prefixe contigu long parce qu elle verrouille, elle, les
lignes publiees de source de degat.
