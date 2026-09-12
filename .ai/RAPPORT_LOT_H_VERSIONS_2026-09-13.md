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

**Trois appariements a carte identique** traversent les versions : Command 39 contre 40,
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
