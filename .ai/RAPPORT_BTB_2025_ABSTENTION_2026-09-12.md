# Big Team Battle 2025 — pourquoi le decodeur n'attribuait presque aucune source de degat

Date : 2026-09-12 · Branche : `wt/btb-2025-abstention` (base `feat/v75` 2f5d165be)
Perimetre : une session, borne. Correctif livre (la cause est simple et prouvee).

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
