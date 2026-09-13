# Lot H — la version majeure du film, calque par calque (2026-09-13)

> Branche `wt/mesure-versions`, base `feat/v75` `e528e347f` (lot E fusionne : le decodeur vit
> sous `internal/games/halo_infinite/film/{filmdec,replay,killsource}`).
> **C'est une MESURE, pas un correctif.** Aucun code de production n'a ete modifie ; les seuls
> fichiers ajoutes sont un instrument de mesure garde par variable d'environnement et les
> temoins de version ajoutes a `config/replay_corpus.toml` (item H.3).
> Aucune base DuckDB n'a ete ouverte (serveur de dev en RW sur :8000) ; aucune ecriture dans le
> parc : toutes les cuissons ont eu lieu dans une **racine jetable** (methode du lot B-bis).

## Question

Depuis le lot G, la version majeure du film — u32 little-endian en tete de `chunk_00`,
`filmdec.FilmMajorVersion` — est lue et publiee (`coverage.filmMajorVersion`). Un seul calque
y branche : le decoupage du gamertag du pied de film (gamertag en tete si version <= 38 ou
>= 41, a l'octet 12 si 39-40). Tout le reste du decodeur est ecrit comme si « tout etait en
41 ». La question du lot H : **pour chacun des autres calques, le format diverge-t-il par
version ?**

Le residu qui l'a fait poser : apres le lot G, les morts sans source de degat tombent a 5,9 %
sur le parc, mais les Big Team Battle de 2025 (versions 39-40) restent a 27 % contre 4 % en
2024 et 10,6 % en 2026, et les BTB de 2023 (versions 31-33) a 22 %.

---

## H.1 — Le corpus par version

### Le parc, par version et par build (mesure exhaustive)

Les 1 351 films du cache local, version lue a l'octet 0 de `chunk_00.bin` (le cache n'en porte
aucun compresse), build lu en clair dans la section d'identification du meme chunk :

| version | build | films | periode connue |
|---|---|---|---|
| 31 | (sans section d'identification) | 3 | 2023 |
| 33 | (sans section) x2, `HI_1_4_1` x1 | 3 | 2023 |
| 37 | `HI_1_8_0` | 10 | 2024 |
| 38 | `HI_1_9_0` | 1 | 2024-2025 |
| **39** | `HI_1_10_0` | **26** | mars 2025 |
| **40** | `HI_1_11_0` (39) + `HI_1_12_0` (146) | **185** | juillet a novembre 2025 |
| 41 | `HI_1_13_0` | 1 123 | 2026 |

**Zero film incomplet** : sur les 1 351, le nombre de `chunk_NN.bin` presents egale le nombre
d'entrees du manifeste `film_manifests/<id8>.json` (controle exhaustif). Le corpus ci-dessous
est donc choisi sans contrainte de completude.

**La version est une dimension a DEUX etages** : la version majeure (31..41) et le build
(`HI_1_x_y`). La version 40 porte DEUX builds (`HI_1_11_0` et `HI_1_12_0`) ; le corpus les
couvre tous les deux.

### Comment la carte et le mode ont ete obtenus sans ouvrir de base

Le film **ne nomme pas sa carte** : verifie sur pieces (recherche ASCII et UTF-16LE des 79
modules du catalogue de bornes dans les 25 chunks de `0797ce72`, carte connue Live Fire /
`sgh_interlock` : zero occurrence ; seule la chaine `scenario-intro-component` sort, c'est un
nom de composant du registre). C'est coherent avec « le mur » des notes de retro-ingenierie :
les bornes et les largeurs sont calculees par le moteur au chargement du tag scenario, le film
n'en porte rien.

Les cartes et modes du corpus viennent donc de **trois sources du depot, toutes hors base** :

1. `config/replay_corpus.toml` (8 temoins, carte + mode) ;
2. `internal/games/halo_infinite/film/replay/testdata/equivalence/CORPUS.txt` et les 13
   `<id8>.facts.json` qui l'accompagnent (carte, variante, scores, roster) ;
3. `.ai/RAPPORT_BTB_2025_ABSTENTION_2026-09-12.md` § 3 (les huit films instrumentes du lot G :
   date et carte).

**Question ouverte (base)** : pour tout film hors de ces trois listes, la carte n'existe qu'en
base (`match_registry.map_id` + registre des cartes). Une sonde par decodage a ete essayee
(cuisson d'un film reduit sous les 21 signatures de bornes distinctes du catalogue) : elle
separe nettement les FAMILLES de largeurs d'axe (les six cartes a 12-13 bits rendent
2 805 points et 0 vol tronque sur `0797ce72`, les quinze a 15-18 bits 2 631 a 2 683 points et
12 vols tronques), mais **ne separe pas les cartes d'une meme famille** — Live Fire, Aquarius,
Breaker, Cliffhanger, Forbidden et Forest y sont indiscernables. La carte exacte d'un film
inconnu reste donc une verite de base.

### Le corpus retenu

Huit films mesures (3 en version 39, 3 en version 40, 2 en versions 31-38), chacun appareille
a un temoin de version 41 de meme carte ou de meme mode, plus un temoin BTB de version 37 qui
sert de charniere entre les deux mondes.

| # | film | ver | build | carte | mode | Mo | chunks | temoin apparie | appariement |
|---|---|---|---|---|---|---|---|---|---|
| 1 | `111fa685` | 39 | HI_1_10_0 | Command | BTB (2025-03) | 31 | 31 | `b81f6415` (v40 Command) + `58d09c44` (v41 BTB) | **meme carte** entre 39 et 40 |
| 2 | `443426df` | 39 | HI_1_10_0 | Refuge | BTB (2025-03) | 50 | 38 | **`58d09c44`** (v41, Refuge, BTB) | **meme carte ET meme mode** |
| 3 | `084a804d` | 39 | HI_1_10_0 | Fortitude Heavies | BTB Heavies:CTF | 63 | 57 | `5676a9ba` (v41, Insolence, BTB) | meme famille de mode (BTB, 24-27 joueurs) |
| 4 | `e5adf7b2` | 40 | HI_1_11_0 | Fragmentation | BTB (2025-07) | 33 | 31 | `a349fea8` (v33, Fragmentation Heavies) | **meme carte** entre 33 et 40 |
| 5 | `b81f6415` | 40 | HI_1_12_0 | Command | BTB (2025-10) | 54 | 51 | `111fa685` (v39 Command) | **meme carte** entre 39 et 40 |
| 6 | `bcb6d393` | 40 | HI_1_12_0 | Cliffhanger | CTF:Arena | 18 | 21 | **`000d5950`** (v41, Cliffhanger) + `64e8adfa`/`fb1a1a72` (v41 CTF:Arena) | **meme carte** et **meme mode** |
| 7 | `60ae07c4` | 37 | HI_1_8_0 | Live Fire | Ranked:Oddball | 35 | 44 | **`0797ce72`** (v41, Live Fire) + `d9781168` (v41 Oddball:Arena) | **meme carte** et **meme mode** |
| 8 | `a349fea8` | 33 | (sans build) | Fragmentation Heavies | BTB Heavies:Total Control | 70 | 51 | `e5adf7b2` (v40 Fragmentation) + `5676a9ba` (v41 BTB) | **meme carte** entre 33 et 40 |

Temoins de version 41 cuits dans la meme passe : `58d09c44` (Refuge, BTB 2026-01, 81 Mo),
`5676a9ba` (Insolence, BTB 2026-07, 42 Mo), `000d5950` (Cliffhanger, Slayer Fiesta, 23 Mo),
`0797ce72` (Live Fire, Slayer Fiesta, 21 Mo), `d9781168` (Dredge, Oddball:Arena, 33 Mo),
`64e8adfa` (Catalyst, CTF:Arena, 37 Mo), `fb1a1a72` (Banished Narrows, CTF:Arena, 38 Mo).
Charniere : `a26dbcdb` (v37, Oasis, BTB 2024-10, 40 Mo) — le temoin BTB « avant 2025 » du
lot G, a 99,5 % de couverture kill feed.

**Cinq appariements a carte identique** traversent les versions : Command 39 contre 40,
Fragmentation 33 contre 40, Cliffhanger 40 contre 41, Live Fire 37 contre 41, Refuge 39 contre
41. Sur ceux-la, toute difference de calque est imputable a la version (ou a la partie), pas
aux bornes de la carte.

### Methode de cuisson

`cmd/replay-build` compile une fois (`CGO_ENABLED=0`), puis une invocation par film avec
`LEVELUP_REPO_ROOT` pointant sur une **racine jetable** du scratchpad (elle ne porte que
`config/` et `data/titles/halo_infinite/reference/map_quant_bounds.json`) et le repertoire de
chunks du parc passe en argument positionnel, donc **lu et jamais ecrit**. Le verrou
`filmproc.AcquireSolo` se pose dans la racine jetable : aucune interference avec le parc ni
avec le serveur de dev.

**Deux passes** :

- **passe 1 — sans faits** (les 16 films) : c'est le FILM SEUL, donc la mesure du decodeur.
  Controle de fidelite : `0797ce72` cuit sans faits est identique a son artefact du parc
  (cuit avec faits) sur tous les calques mesures — 103 pistes, 717 tirs, 102 grenades,
  299 projectiles, 217 armes au sol, 227 poses, 8 joueurs identifies — a l'exception de
  `scoreTimeline` (1 ligne contre 3). Les faits ne nourrissent que le score et les actions
  d'objectif attribuees.
- **passe 2 — avec faits** (les 6 films du corpus qui ont un `.facts.json` versionne :
  `084a804d` v39, `a349fea8` v33, `60ae07c4` v37, `000d5950`, `64e8adfa`, `d9781168` v41) :
  elle seule mesure le calque score et manches.

---

## H.2 — La mesure, calque par calque

### Le tableau calque x version

`decode` = le calque publie des lignes coherentes ; `VIDE` = zero ligne la ou un temoin
comparable en publie ; `degrade` = publie, mais nettement en dessous du temoin. Les versions 31
et 33 sont scindees selon la SECTION D'IDENTIFICATION de `chunk_00` (chaine `HI_1_x_y`) : c'est
elle, et non la version majeure, qui separe les comportements — cinq films du cache n'en portent
pas (les 3 de version 31 et 2 des 3 de version 33).

| calque | v31 + v33 sans identification (5 films) | v33 `HI_1_4_1` | v37 `HI_1_8_0` | v38 `HI_1_9_0` | v39 `HI_1_10_0` | v40 `HI_1_11_0` | v40 `HI_1_12_0` | v41 `HI_1_13_0` |
|---|---|---|---|---|---|---|---|---|
| vies et identite (registre, index joueur) | **VIDE** | decode | decode | decode | decode | decode | decode | decode |
| tirs | **VIDE** (0 publie sur 1 921-4 137) | decode | decode | decode | decode | decode | decode | decode |
| **lancers de grenade** | **VIDE** | **VIDE** | **VIDE** | **VIDE** | **VIDE** | **VIDE** | decode | decode |
| grenades portees (compteurs) | decode | decode | decode | decode | decode | decode | decode | decode |
| trajectoires bipedes | decode | decode | decode | decode | decode | decode | decode | decode |
| projectiles | decode | decode | decode | decode | decode | decode | decode | decode |
| vehicules (vies, trajets) | decode, trajets NON NOMMES | decode | decode | decode | decode | decode | decode | decode |
| armes au sol, socles, power-ups | decode | decode | decode | decode | decode | decode | decode | decode |
| equipement (poses, lachers, ramassages, charges) | decode | decode | decode | decode | decode | decode | decode | decode |
| inventaire et changements d'arme | decode | decode | decode | decode | decode | decode | decode | decode |
| objectifs — drapeaux | n. m. | n. m. | n. m. | n. m. | decode | n. m. | decode | decode |
| objectifs — crane | n. m. | n. m. | decode (48) | n. m. | n. m. | n. m. | n. m. | decode (37) |
| objectifs — zones, bombe | **non mesure** | non mesure | non mesure | non mesure | **non mesure** | **non mesure** | non mesure | non mesure |
| medailles (pied de film) | decode | decode | decode | decode | decode | decode | decode | decode |
| kill feed et sources de degat | **degrade** (4 a 11 % de morts credibles) | 8 a 41 % | 5 a 46 % | 51 % | 33 a 50 % | 63 % | 55 a 81 % | 28 a 64 % |
| sons et evenements de mode | decode | decode | decode | decode | decode | decode | decode | decode |
| score et manches | decode | n. m. | decode | n. m. | decode | n. m. | n. m. | decode |
| **empreinte du registre ECS** | **INCONNUE** | **INCONNUE** | **INCONNUE** | **INCONNUE** | **INCONNUE** | **INCONNUE** | **INCONNUE** | connue |

`n. m.` = mode absent du corpus pour cette version, donc non mesurable — ce n'est pas un vide.

### Les chiffres, film par film (passe sans faits, schema 54)

| film | ver / build | pistes | points | pistes avec xuid | joueurs | tirs publ. / dispo | **lancers de grenade** | projectiles publ. / pistes / tronques | armes au sol | socles | poses | vehicules vies / publ. |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `bcb6d393` | 40 / 1_12 | 56 | 18 935 | 54 | 11 | 924 / 1 078 | **47** | 56 / 56 / 2 | 125 | 11 | 101 | 0 / 0 |
| `000d5950` | 41 / 1_13 | 104 | 29 220 | 104 | 8 | 504 / 519 | **70** | 436 / 580 / 3 | 220 | 0 | 295 | 0 / 0 |
| `60ae07c4` | 37 / 1_8 | 173 | 45 133 | 173 | 8 | 1 632 / 1 635 | **0** | 417 / 647 / 3 | 188 | 7 | 190 | 4 / 0 |
| `0797ce72` | 41 / 1_13 | 103 | 24 973 | 103 | 8 | 717 / 718 | **102** | 299 / 555 / 4 | 217 | 0 | 227 | 2 / 0 |
| `111fa685` | 39 / 1_10 | 243 | 90 436 | 243 | 24 | 1 771 / 1 996 | **0** | 658 / 672 / 10 | 436 | 0 | 591 | 27 / 15 |
| `b81f6415` | 40 / 1_12 | 319 | 106 253 | 317 | 29 | 1 890 / 4 726 | **292** | 612 / 623 / 4 | 896 | 24 | 668 | 75 / 36 |
| `443426df` | 39 / 1_10 | 332 | 89 422 | 332 | 28 | 3 678 / 4 856 | **0** | 358 / 374 / 15 | 577 | 17 | 428 | 45 / 22 |
| `58d09c44` | 41 / 1_13 | 305 | 85 619 | 305 | 25 | 3 411 / 7 842 | **395** | 515 / 563 / 27 | 658 | 18 | 443 | 86 / 32 |
| `e5adf7b2` | 40 / **1_11** | 254 | 85 472 | 253 | 26 | 1 492 / 1 786 | **0** | 551 / 579 / 20 | 487 | 0 | 620 | 96 / 53 |
| `a349fea8` | 33 / **aucun** | 334 | 113 963 | **0** | **0** | **0 / 4 137** | **0** | 738 / 771 / 27 | 827 | 48 | 729 | 265 / 109 |
| `084a804d` | 39 / 1_10 | 344 | 111 947 | 344 | 26 | 3 744 / 6 014 | **0** | 873 / 914 / 28 | 913 | 30 | 808 | 180 / 97 |
| `5676a9ba` | 41 / 1_13 | 304 | 96 161 | 302 | 26 | 2 928 / 3 449 | **217** | 591 / 598 / 8 | 648 | 40 | 490 | 94 / 41 |
| `a26dbcdb` | 37 / 1_8 | 269 | 92 529 | 269 | 27 | 2 712 / 3 233 | **0** | 565 / 575 / 12 | 541 | 35 | 508 | 107 / 66 |
| `d9781168` | 41 / 1_13 | 174 | 36 579 | 174 | 8 | 3 337 / 3 401 | **170** | 191 / 269 / 4 | 344 | 8 | 295 | 0 / 0 |
| `64e8adfa` | 41 / 1_13 | 139 | 50 142 | 139 | 8 | 2 865 / 2 879 | **154** | 235 / 303 / 2 | 293 | 10 | 229 | 0 / 0 |
| `fb1a1a72` | 41 / 1_13 | 147 | 44 697 | 147 | 8 | 3 275 / 3 289 | **165** | 258 / 533 / 32 | 326 | 12 | 341 | 0 / 0 |

---

## Les divergences NOMMEES

### D1 — Les lancers de grenade s'eteignent au BUILD, pas a la version majeure

**Le fait.** `coverage.grenades.available` vaut zero sur toute la lignee ancienne et redevient
non nul a partir du build `HI_1_12_0`. Le controle decisif est un couple de films de LA MEME
CARTE (Command) : `111fa685` (version 39, build `HI_1_10_0`) publie **0** lancer, `b81f6415`
(version 40, build `HI_1_12_0`) en publie **292**. Second couple, meme carte (Live Fire) :
`60ae07c4` (version 37) **0**, `0797ce72` (version 41) **102**. La carte est hors de cause.

**Ou exactement.** `filmdec.ScanGrenadeThrows` (`grenade_events.go`) est un balayage d'octets
PUR : il cherche le marqueur de 24 bits `grenadeMarker = 0x4C0C00` dans les paquets delta, lit
l'identifiant de 32 bits qui suit et le compare a `GrenadeTypeIDsByRank` — quatre valeurs, les
identifiants globaux de tag du groupe `proj` des quatre grenades, decales d'un bit a gauche.
L'instrument `version_grenade_tags_research_test.go` refait ce meme balayage SANS la liste
blanche, et histogramme ce qui suit le marqueur.

**La mesure, 38 films probes (aucune carte, aucun catalogue, aucune base) :**

| version / build | films probes | marqueurs trouves (min-max) | **lancers reconnus** |
|---|---|---|---|
| 31 (sans identification) | 3 | 13 - 23 | **0 sur 3** |
| 33 | 3 | 160 - 763 | **0 sur 3** |
| 37 `HI_1_8_0` | 4 | 100 - 257 | **0 sur 4** |
| 38 `HI_1_9_0` | 1 | 1 221 | **0** |
| 39 `HI_1_10_0` | 4 | 484 - 1 807 | **0 sur 4** |
| 40 **`HI_1_11_0`** | 12 | 84 - 1 207 | **0 sur 12** |
| 40 **`HI_1_12_0`** | 10 | 3 - 974 | **49 a 215 sur 8 films** (les 2 restants ne portent que 3 et 4 marqueurs) |
| 41 `HI_1_13_0` | 4 | 380 - 1 435 | **68 a 152 sur 4** |

**Le verdict.** Le marqueur est present a TOUTES les versions : la grammaire du record de lancer
n'a pas bouge. Ce qui a bouge, ce sont les IDENTIFIANTS DE TAG — la liste blanche du decodeur est
datee d'un seul build. Et les identifiants les plus frequents derriere le marqueur sur la lignee
ancienne forment eux-memes une famille stable, partagee par les versions 33, 37, 39 et
40/`HI_1_11_0` : `0xB2A8143A`, `0x31C3FD40`, `0xEB68DED2`, `0xC2471A0C`, `0x99F8AB50`,
`0x764ACFA8`, `0x00F39CF0`, `0xF290BD74`. Ils sont donc identifiables a leur tour, par la methode
qui a etabli les quatre actuels (appariement aux decrements du compteur d'inventaire porte, et
table `grenade_types` du binaire du build correspondant).

**Ce que cela coute.** Les films concernes sont ceux des builds anterieurs a `HI_1_12_0` :
3 + 3 + 10 + 1 + 26 (versions 31, 33, 37, 38, 39) plus les 39 de `HI_1_11_0`, soit **82 films
sur 1 351**. Sur eux, `verdict.grenades` vaut `aucune donnee` et le calque est vide. Estimation
du gain : les films de build recent rendent 47 a 395 lancers chacun ; un profil par build en
restituerait le meme ordre de grandeur sur ces 82 films, soit ~5 000 a 10 000 lancers.

### D2 — L'identite entiere s'eteint sur les films sans section d'identification

**Le fait.** Cinq films du cache — `13b00e35`, `47d20b5d`, `50247b26` (version 31), `03af54c3` et
`a349fea8` (version 33) — sont exactement ceux dont `chunk_00` ne porte pas de chaine
`HI_1_x_y`. Sur eux, l'artefact sort avec 242 a 334 PISTES et 85 000 a 114 000 points de
trajectoire, mais **zero piste porteuse de xuid, zero joueur identifie, roster vide, et zero tir
publie** (1 921 a 4 137 tirs tous classes `noSlot`). Le sixieme film de version 33, `a521164d`,
qui porte la chaine `HI_1_4_1`, est NOMINAL : 26 joueurs, 178 pistes sur 179 avec xuid,
19 lectures d'index concordantes. La version majeure ne separe pas ces deux cas ; la section
d'identification, si.

**Ou exactement.** Le pont d'identite part de `replay.ScanPlayerIndices`, qui cherche le XUID de
chaque joueur du roster comme un motif de **64 bits petit-boutiste aligne au bit** dans chaque
chunk de replication, puis lit les **5 bits** qui le precedent (`weaponv3.ResolveXuidToPI`,
`PIBits = 5`). L'instrument `TestVersionIndexJoueur` prend le roster DANS LE FILM (les XUID
distincts du pied de film, lisibles sur toutes les versions) et compte les motifs trouves :

| film | version / build | roster du pied de film | **XUID trouves dans la trame** | index lus |
|---|---|---|---|---|
| `13b00e35` | 31 / aucun | 24 | **4** | 0, 22, 23, 24 |
| `47d20b5d` | 31 / aucun | 26 | **2** | 0, 24 |
| `50247b26` | 31 / aucun | 27 | **4** | 0, 22, 23, 24 |
| `03af54c3` | 33 / aucun | 27 | **3** | 0, 23, 24 |
| `a349fea8` | 33 / aucun | 25 | **2** | 0, 24 |
| `a521164d` | 33 / **`HI_1_4_1`** | 27 | **27 sur 27** | 0..27 |
| `11de8353` | 38 / `HI_1_9_0` | 26 | **26 sur 26** | 0..26 |
| `60ae07c4` | 37 / `HI_1_8_0` | 8 | **8 sur 8** | 0..7 |
| `a26dbcdb` | 37 / `HI_1_8_0` | 27 | **27 sur 27** | 0..26 |
| `111fa685` | 39 / `HI_1_10_0` | 24 | **24 sur 24** | 0..24 |
| `e5adf7b2` | 40 / `HI_1_11_0` | 26 | **26 sur 26** | 0..27 |
| `b81f6415` | 40 / `HI_1_12_0` | 27 | **27 sur 27** | 0..29 |
| `bcb6d393` | 40 / `HI_1_12_0` | 11 | **11 sur 11** | 0..11 |
| `000d5950` | 41 / `HI_1_13_0` | 8 | **8 sur 8** | 0..7 |

**Le verdict.** Sur les cinq films sans identification, 2 a 4 XUID sur 24 a 27 se trouvent, et les
index lus devant eux (0, 22, 23, 24) sont NON INJECTIFS : ce sont des coincidences de motif, pas
la table. La porte d'injectivite de `ScanPlayerIndices` vide alors la table, a bon droit. **Le
XUID de 64 bits n'est pas ecrit en clair dans les chunks de replication de ces films** : le pont
d'identite du rejeu, tel qu'il est ecrit, ne peut pas exister sur eux. Toute reparation passe par
une AUTRE voie — la table des joueurs de la section 3 de `chunk_00` (percee le 2026-09-12, qui
porte le XUID en clair sur 64 bits a 85 bits du debut de chaque enregistrement de slot), ou le
pont par les morts.

**Ce que cela coute** : 5 films sur 1 351 (0,4 %). Mais sur eux, c'est la moitie du document qui
tombe : identite, noms de piste, attribution des tirs, nommage des trajets de vehicule, actions
d'objectif. Estimation du gain : 1 900 a 4 100 tirs et 24 a 27 joueurs par film.

### D3 — Le registre ECS change avec le build, et la production le sait sans en tenir compte

**Le fait.** `filmdec.RegistryFingerprint` compare l'empreinte FNV-1a du registre du film a
`KnownRegistryFingerprint` (`0x61e492dd4de7fd4e`) et journalise un WARN quand elle differe. Sur
les 16 cuissons du corpus, la separation est **parfaite** :

| version du film | films cuits | WARN « empreinte du registre ECS du film INCONNUE » |
|---|---|---|
| 33, 37, 39, 40 | 9 | **9 sur 9** |
| 41 | 7 | **0 sur 7** |

Et le registre n'y est pas seulement different, il est plus PETIT : sur les films anciens il porte
**49 blocs et 1 029 a 1 033 slots non vides**, contre **50 blocs et 1 067 slots** pour le registre
de reference (`HI_1_13_0`, celui que decrit `filmdec/testdata/ecs_table.tsv`). Huit empreintes
distinctes ont ete relevees sur le corpus pour une seule connue, dont une a 50 blocs et
1 067 slots — meme cardinal que la reference, contenu different.

**Le verdict.** La table (archetype, composant) sur laquelle repose TOUT le dispatch du decodeur —
l'ordre des composants EST l'index de bit du masque de presence — n'est valable que pour le build
`HI_1_13_0`. Les 228 films anterieurs se decodent sous une grammaire dont le film declare
lui-meme qu'elle n'est pas la sienne. Le code le dit deja, en WARN, et n'en tire aucune
consequence. C'est le point ou le profil de dechiffrage doit brancher EN PREMIER : tous les
autres en dependent, et l'en-tete de `registry_fingerprint.go` l'avait deja ecrit sur un seul
film (`06dfe6d9`) sans que la portee du constat soit mesuree.

### D4 — La marche des morts se calibre a vide sur les films anciens

**Le fait.** `TestBTB2025Abstention` publie, par film, la calibration du decodeur de source de
degat et la ventilation des refus. Sur les trois films de version 31, les trois de version 33 et
`60ae07c4` (version 37), la ligne de calibration porte **`PROFIL PLAT ... : valeurs par defaut
conservees`** — l'auto-calibration ne trouve aucun signal et retombe sur ses defauts
(`axisW = 14`, `indexW = 1`). Sur `11de8353` (38), `a26dbcdb` (37) et tous les films des versions
39 a 41, elle trouve un profil net (`axisW` 12 a 16, `indexW` 2 ou 3, score 284 a 8 052).

| film | version | morts brutes | **credibles** | part | refus dominant |
|---|---|---|---|---|---|
| `13b00e35` | 31 | 346 | 23 | **6,6 %** | hors_plage_bipede = 300 |
| `47d20b5d` | 31 | 212 | 24 | 11,3 % | hors_plage_bipede = 162 |
| `50247b26` | 31 | 190 | 20 | 10,5 % | hors_plage_bipede = 158 |
| `a349fea8` | 33 | 709 | 30 | **4,2 %** | hors_plage_bipede = 651 |
| `a521164d` | 33 | 135 | 11 | 8,1 % | hors_plage_bipede = 118 |
| `03af54c3` | 33 | 307 | 126 | 41,0 % | hors_plage_bipede = 170 |
| `60ae07c4` | 37 | 549 | 26 | **4,7 %** | hors_plage_bipede = 482 |
| `a26dbcdb` | 37 | 434 | 199 | 45,9 % | hors_plage_bipede = 220 |
| `11de8353` | 38 | 285 | 146 | 51,2 % | victime_hors_roster = 128 |
| `111fa685` | 39 | 345 | 171 | 49,6 % | victime_hors_roster = 161 |
| `443426df` | 39 | 280 | 92 | 32,9 % | hors_plage_bipede = 154 |
| `084a804d` | 39 | 587 | 282 | 48,0 % | hors_plage_bipede = 272 |
| `e5adf7b2` | 40 | 273 | 173 | 63,4 % | victime_hors_roster = 91 |
| `b81f6415` | 40 | 486 | 269 | 55,3 % | hors_plage_bipede = 177 |
| `bcb6d393` | 40 | 54 | 44 | 81,5 % | hors_plage_bipede = 5 |
| `58d09c44` | 41 | 405 | 118 | 29,1 % | hors_plage_bipede = 272 |
| `5676a9ba` | 41 | 361 | 196 | 54,3 % | hors_plage_bipede = 137 |
| `000d5950` | 41 | 136 | 87 | 64,0 % | hors_plage_bipede = 48 |
| `0797ce72` | 41 | 144 | 86 | 59,7 % | hors_plage_bipede = 46 |
| `d9781168` | 41 | 155 | 43 | 27,7 % | hors_plage_bipede = 109 |

**Le verdict.** La lecture n'est pas binaire : un profil plat ne vide pas le calque, il le
degrade. Les trois films de version 31, deux des trois de version 33 et un des deux films de
version 37 tombent sous 11 % de morts credibles, la ou la version 41 tient 28 a 64 %. Et
`03af54c3` (33, 41 %) comme `a26dbcdb` (37, 46 %) montrent que ce n'est pas une fatalite de la
version mais de la CALIBRATION : **la largeur d'axe et la largeur d'index de la marche des morts
sont devinees par mesure alors qu'elles sont une donnee de la carte et du build.** C'est le meme
defaut de fond que le lot B-bis a corrige du cote des projectiles (une porte qui lisait un
litteral au lieu du catalogue), a un autre endroit. Cela chiffre exactement le residu « BTB 2023
a 22 % de morts sans source » du controle `_latest`.

### D5 — La bande de slots bipede n'a pas la meme largeur d'un build a l'autre

Releve sur la meme sortie (`plage_bipede`) : `[512, 643..767]` sur les versions 31, 33, 37 et 41 ;
`[128, 757..767]` sur `a26dbcdb` (37), `443426df` et `084a804d` (39), `5676a9ba` (41) ;
`[512, 7808]` sur `11de8353` (38) et `e5adf7b2` (40) ; `[512, 8064]` sur `111fa685` (39). Quand la
borne haute vaut 7 808 ou 8 064, `hors_plage_bipede` tombe a 0 ou 1 et le refus bascule sur
`victime_hors_roster` : ce n'est pas une amelioration, c'est une porte qui ne filtre plus rien. La
bande est DEDUITE du film, avec des resultats sans commune mesure d'un build a l'autre. Point a
instruire, pas encore une cause etablie.

---

## Ce qui est IDENTIQUE d'une version a l'autre (avec la preuve)

1. **La grammaire du pied de film — kill feed, medailles, evenements de mode.**
   `TestVersionEvenements` analyse le chunk de temps forts de 20 films (versions 31 a 41) sous LES
   DEUX decoupages de gamertag. Sur les 20, **le nombre d'evenements reconnus, leur ventilation
   par nature et la liste des `type_hint` sont rigoureusement identiques sous les deux
   decoupages** (exemple, `13b00e35`, version 31 : 484 evenements, 196 kills, 202 morts,
   71 medailles, 15 evenements de mode, `type_hint` = {10, 20, 50, 100, 150, 205} dans les deux
   lectures). Seul le champ GAMERTAG bouge : sous le decoupage que la version impose, les blocs
   sans gamertag sont a **0 sur les 20 films** ; sous l'autre, ils montent a 26-463 et les
   gamertags distincts s'effondrent (2 pour 24 a 28 joueurs en versions 39-40, 0 pour 11 sur
   `bcb6d393`). **La version ne commande QUE l'implantation du gamertag** — ce que le lot G a
   corrige, et rien d'autre dans ce flux.
2. **Les medailles.** Decodees sur toutes les versions : 21 a 221 medailles par film, types de
   medaille distincts non vides partout, versions 31 et 33 comprises (71 et 90 medailles, 20 a
   30 types distincts).
3. **Les trajectoires bipedes et les points publies.** 56 a 344 pistes et 18 935 a 113 963 points
   sur les 16 films, sans trou par version. Les cinq films sans identification publient leurs
   trajectoires normalement : c'est le NOM qui manque, pas la position.
4. **Les projectiles.** 56 a 914 pistes par film, 2 a 32 vols tronques, sans ordre par version
   (32 tronques sur un film de version 41, 10 sur un de version 39).
5. **Les armes au sol, les socles et les power-ups.** 125 a 913 armes au sol par film ; socles
   presents des que le mode en porte (7 a 48) sur toutes les versions mesurees — dont 48 socles
   et 257 occupations datees sur `a349fea8` (version 33), le film par ailleurs le plus degrade.
6. **L'equipement** (poses, confirmations, deploiements, lachers, ramassages, changements,
   charges, grappin) : 101 a 808 poses et 92 a 504 lachers, sur toutes les versions.
7. **L'inventaire et les changements d'arme** : 100 a 918 lectures decodees, 21 a 229 changements,
   partout.
8. **Les vehicules** : 4 a 265 vies recensees des que le mode en porte, versions 33 a 41. Seul le
   NOMMAGE des trajets tombe, et seulement la ou l'identite est morte (D2).
9. **Le score et les manches** (passe avec faits) : 2 a 3 lignes de chronologie et 1 a 3 manches
   sur `a349fea8` (33), `60ae07c4` (37), `084a804d` (39), `000d5950`, `64e8adfa` et `d9781168`
   (41) — decode partout, y compris la ou l'identite est morte, parce qu'il vient des faits.
10. **La completude du cache** : sur les 1 351 films, le nombre de fichiers de chunk egale le
    nombre d'entrees du manifeste. Aucune version n'est sous-representee par une bobine tronquee.

## Ce qui N'EST PAS une divergence de version, et qui pouvait le paraitre

- **Les tirs « sans slot » en Big Team Battle.** 2 836 sur 4 726 (60 %) sur `b81f6415`
  (version 40) — mais 4 431 sur 7 842 (56 %) sur `58d09c44`, VERSION 41, et 225 sur 1 996 (11 %)
  sur `111fa685`, version 39. Aucun ordre par version : c'est un effet de la densite de joueurs.
- **Les actions d'objectif non attribuees.** `084a804d` (version 39) : 216 actions disponibles,
  **0** rattachee ; `64e8adfa` (version 41) : 340 sur 340. Mais le refus vient du verdict de pont
  « non publiable : un slot change de porteur », et le temoin `58d09c44` — version 41, Big Team
  Battle — porte EXACTEMENT le meme verdict. Effet Big Team Battle, pas effet version.
- **Les socles a zero sur `000d5950` et `0797ce72`** (version 41) : deux parties Super Fiesta, un
  mode sans socle d'arme. Le mode, pas la version.

## Limites de cette mesure (a lire avant d'en tirer un correctif)

1. **Les modes a zones et le mode Assaut ne sont mesures sur AUCUNE version ancienne.** Le corpus
   n'a aucun film de version 39, 40 ou 31-38 dont la carte soit connue hors base ET dont le mode
   porte des zones (Roi de la colline, Bastions, Controle total) ou une bombe. Les zones de
   `a349fea8` (Controle total, version 33) sortent a zero, mais ce film a par ailleurs perdu toute
   identite : le resultat n'est pas interpretable seul. **Il manque un temoin, et il ne peut etre
   choisi qu'avec la base.**
2. **La carte exacte d'un film hors des trois listes du depot est une verite de base.** La sonde
   par decodage separe les familles de largeurs d'axe, pas les cartes d'une meme famille (six
   cartes indiscernables sur l'essai). Les cuissons de `13b00e35`, `47d20b5d`, `50247b26`,
   `03af54c3`, `a521164d`, `11de8353`, `0a247154`, `036c102a`, `06dfe6d9` et `0f53c2da` ont donc
   ete faites avec une carte NEUTRE (Banished Narrows), et **seuls leurs compteurs independants de
   la carte** ont ete lus : identite, index de joueur, roster, tirs publies, lancers de grenade.
   Leurs comptes de projectiles, de vols tronques et de socles ne figurent nulle part ici.
3. **Une seule partie par couple (version, build) sur plusieurs lignes du tableau.** D1 repose sur
   38 films, D2 sur 14, D3 sur 16, D4 et D5 sur 20. Les lignes `n. m.` ne sont pas des vides
   mesures.
4. **Aucune base DuckDB n'a ete ouverte.** Les scores d'API, la table des scores d'equipe et les
   participants ne sont donc PAS confrontes aux compteurs ci-dessus. L'oracle `swap.sh`
   (placement des reapparitions par camp, qui exige la table des equipes) n'a pas ete joue :
   `identity` et `coverage.bridge` l'ont remplace la ou ils suffisent.

---

## H.3 — Ou le profil de dechiffrage doit brancher sur la version

Liste chiffree, par ordre de dependance (le premier conditionne les autres). « Cle » dit sur
QUOI brancher : la mesure montre que la version majeure ne suffit nulle part ou le build est
disponible.

| # | point de branchement | cle | preuve | films concernes | gain estime | effort |
|---|---|---|---|---|---|---|
| **P1** | **Table du registre ECS (archetype, composant)** — l'ordre des composants EST l'index de bit du masque de presence ; `ecs_table.tsv` ne decrit que `HI_1_13_0` | **build** (empreinte du registre lue dans le film) | D3 : 9 films sur 9 hors version 41 declarent une empreinte inconnue, 0 sur 7 en version 41 ; 49 blocs / 1 029-1 033 slots contre 50 / 1 067 ; 8 empreintes distinctes pour une seule connue | **228** (tout ce qui n'est pas version 41) | prerequis : sans lui, tout gain sur les autres points est fragile | L |
| **P2** | **Liste blanche des identifiants de tag `proj`** (`GrenadeTypeIDsByRank`) du balayage des lancers de grenade | **build** (frontiere a l'interieur de la version 40) | D1 : 0 lancer reconnu sur 0 + 3 + 3 + 4 + 1 + 4 + 12 = 27 films des builds anterieurs a `HI_1_12_0`, 49 a 215 sur 8 films de `HI_1_12_0` et 68 a 152 sur 4 de `HI_1_13_0` ; marqueur present partout (13 a 1 807) | **82** | 47 a 395 lancers par film, soit **~5 000 a 10 000 lancers** rendus au parc | M |
| **P3** | **Voie de resolution de l'index de joueur** — le motif « XUID 64 bits + 5 bits d'index » n'existe pas dans la trame des films sans section d'identification | **presence de la section d'identification** de `chunk_00` | D2 : 2 a 4 XUID trouves sur 24 a 27, index non injectifs (0, 22, 23, 24) sur 5 films ; 8 a 27 sur 8 a 27, index 0..n, sur les 9 autres | **5** | l'identite complete : 24 a 27 joueurs et **1 900 a 4 100 tirs publies par film** (aujourd'hui zero) | M |
| **P4** | **Calibration de la marche des morts** (`axisW`, `indexW`) — devinee par mesure au lieu d'etre lue dans la carte et le build | **carte + build** | D4 : `PROFIL PLAT : valeurs par defaut conservees` sur 7 films (3 en v31, 3 en v33, 1 en v37) ; morts credibles 4,2 a 11,3 % contre 28 a 64 % en version 41, refus dominant `hors_plage_bipede` (118 a 651) | 7 mesures, et par extension les **44** films d'avant 2025 | +30 a +50 points de couverture du kill feed sur ces films ; chiffre exactement le residu « BTB 2023 a 22 % sans source » | M |
| **P5** | **Bande de slots bipede** (`plage_bipede`), deduite du film, sans commune mesure d'un build a l'autre | **build** (a instruire) | D5 : `[512, 767]`, `[128, 767]`, `[512, 7808]`, `[512, 8064]` selon le film ; quand la borne haute vaut 7 808-8 064, la porte ne filtre plus rien | 20 mesures | non chiffre — cause non etablie, a instruire avant tout correctif | S (instruction) |

Et le point ou la version est **deja** branchee, et ou la mesure confirme qu'il n'y a rien de
plus a faire : l'implantation du gamertag du pied de film (lot G). Les comptes d'evenements sont
identiques sous les deux decoupages sur les 20 films mesures ; seul le champ de nom bouge, et le
decoupage que la version impose rend zero bloc sans gamertag sur les 20.

### Ordre recommande

P1 d'abord (c'est la grammaire elle-meme), puis P2 (le gain le plus large et le plus simple :
une table d'identifiants par build, sans changement de lecteur), puis P3 (une voie de
remplacement existe deja, la table des joueurs de la section 3), puis P4 et P5, qui demandent
l'un et l'autre une donnee derivee du jeu.

Chacun de ces cinq points change le CONTENU CUIT : il exige une montee de `SchemaVersion`, un
passage au gate corpus avec les temoins ci-dessous, et une recuisson. Aucun n'est un refactor.

### Temoins ajoutes a `config/replay_corpus.toml` (fait dans ce lot)

Le corpus temoin ne portait que deux films hors version 41 (`084a804d` en 39, `bcb6d393` en 40),
et aucun avant 2025 : c'est pourquoi le trou du lot G a traverse le gate. Quatre entrees sont
ajoutees, une par grammaire distincte mesuree.

| id | famille | version / build | carte | mode | ce qu'il garde |
|---|---|---|---|---|---|
| `111fa685` | `version_39` | 39 / `HI_1_10_0` | Command | BTB (2025-03) | la grammaire de mars 2025, et la MEME CARTE qu'un film de version 40 — l'appariement qui a prouve D1 |
| `e5adf7b2` | `version_40_build_1_11` | 40 / `HI_1_11_0` | Fragmentation | BTB (2025-07) | la seule grammaire ou la version majeure MENT : version 40 comme `bcb6d393`, mais zero lancer de grenade sur 1 082 marqueurs |
| `60ae07c4` | `version_37` | 37 / `HI_1_8_0` | Live Fire - Ranked | Ranked:Oddball | une vieille grammaire croisee avec la carte a index de region sur 2 bits, et le calque crane (48 portages) sur une version ancienne |
| `a349fea8` | `version_33_sans_identification` | 33 / aucun | Fragmentation Heavies | BTB Heavies:Total Control | la degradation la plus lourde connue (identite, tirs et lancers a zero) — et sa reparation future |

`bcb6d393` (deja au corpus, famille `ctf_mono_manche`) tient de fait le role de temoin
`version_40_build_1_12` : il est laisse en l'etat, la raison de la nouvelle famille le dit.

**Le gate corpus lui-meme n'est pas joue ici** : il exige `levelup replay-facts-export`, donc
l'ouverture en lecture de la base partagee, ce que la consigne de ce lot interdit. Il revient au
pilote, base libre, avec `--base=feat/v75`. Attendu : `12 temoins sur 12`, `54 -> 54`, zero
gain, zero perte — ces quatre films n'existaient pas au manifeste, leur premiere cuisson des
deux cotes doit etre identique.

### Ce qui manque encore au corpus (ne peut pas etre fait sans la base)

Un temoin de mode a ZONES (Roi de la colline, Bastions, Controle total) ou a BOMBE sur une
version anterieure a 41. Aucun film du cache ne remplit a la fois « version < 41 », « mode a
zones » et « carte connue hors base ». Le choisir demande une requete sur `match_registry`.

## Decouvertes (hors perimetre, non traitees)

1. **Le corpus d'equivalence versionne porte trois films non-41 sans que personne l'ait su** :
   `084a804d` (39), `1c4c63c2` (39), `60ae07c4` (37) et `a349fea8` (33) ont leurs
   `<id8>.facts.json` dans `replay/testdata/equivalence/`. `CORPUS.txt` les decrit par carte et
   par mode, jamais par version — alors que c'est la dimension qui les distingue le plus.
2. **`filmcache` ne porte pas la version dans son manifeste** (deja note au lot G) : chaque
   lecteur re-inflate `chunk_00` pour lire quatre octets. Sur 1 351 films, l'inventaire du lot H
   l'a fait en lisant directement les quatre premiers octets du fichier, sans decompression —
   le cache ne porte AUCUN `chunk_00` compresse (0 sur 1 351, deja mesure le 2026-09-12).
3. **L'en-tete de `registry_fingerprint.go` avait deja vu D3 sur UN film** (`06dfe6d9`,
   49 blocs / 1 031 slots) et l'avait consigne comme une curiosite. La mesure du lot H montre que
   c'est la regle sur 228 films, pas l'exception sur un.
4. **Deux films de build `HI_1_12_0` ne portent que 3 et 4 marqueurs de lancer de grenade**
   (`0014603f`, `05fffb2a`) la ou leurs pairs en portent 194 a 974 : bobines tres courtes ou
   parties sans grenade. Non instruit.
5. **`007d53a4` rend 951 identifiants distincts derriere le marqueur** contre 8 a 117 ailleurs.
   C'est le film que le lot G avait deja signale comme « sans aucun highlight event ». Non
   instruit.
