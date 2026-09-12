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
