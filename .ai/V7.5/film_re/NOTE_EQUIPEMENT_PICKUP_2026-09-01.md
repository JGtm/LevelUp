# Le ramassage d'EQUIPEMENT : nomme a moitie, classes ELUCIDEES (a l'envers), lien toujours refute

Date : 2026-09-01. Lot 4, RECHERCHE PURE (aucune publication, aucun fichier de production
touche). Instruments : `apps/go-api/internal/analysis/replay/equipment_pickup_{naming,classes,
link}_research_test.go`, sous gardes `BIPED_PICKUP_FILM` (etapes 1-2) et `PICKUP_FILM` +
`PICKUP_MAP` (etape 3). Sautes sans elles : aucun effet en CI.

## ETAPE 2 D'ABORD, PARCE QU'ELLE RENVERSE L'ENONCE

L'hypothese donnee etait « classe 2 = equipement, classe 3 = grenades ». **Elle est FAUSSE, et
c'est l'inverse qui se mesure.** Deux juges independants, seuils ecrits avant :

| juge | classe 2 | classe 3 | temoin decale |
|---|---|---|---|
| **J1 — rang de palette i48** (000d5950) | 3,9 % | **45,2 %** | 0,0 % |
| **J1 — rang de palette i48** (00502e52) | 6,2 % | **40,0 %** | 0,0 % |
| J2 — compteur de grenades en hausse (000d5950) | 15,7 % | 0,0 % | 0,0 % |
| J2 — compteur de grenades en hausse (00502e52) | 0,0 % | 0,0 % | 0,0 % |

**La classe 3 EST l'equipement** au sens d'i48 : elle porte un rang de palette dans 40 a 45 %
des cas contre 0,0 % au temoin, sur les deux films. C'est reproduit et net.

**La classe 2 n'est PAS identifiee.** J2 la designe faiblement sur un film (15,7 % contre 0,0 %
en classe 3) et pas du tout sur l'autre (0,0 %). Ce n'est donc pas « les grenades » : c'est
**non conclu**, et la faiblesse du juge est mesuree — i22 ne porte ses compteurs que sur 120 et
89 lectures, ce qui laisse peu de matiere.

Le seuil pre-enregistre C1 (separation >= 40 points) est atteint sur 000d5950 (-41,2) et **pas**
sur 00502e52 (-33,8). La DIRECTION est identique et large sur les deux, le barreau des 40 points
ne tombe que sur un des deux — publie tel quel, sans rebaisser le seuil.

## ETAPE 1 — NOMMER : la voie fonctionne, sa COUVERTURE ne suffit pas

Voie : chaque ramassage non-arme apparie a une transmission i48 du MEME slot a moins de 500 ms
recoit le rang de palette de cette transmission ; le manifeste (`replay_labels.toml`,
`ability_palettes`) nomme le rang. Les deux films sont de la palette `famille_b` (rangs 19-22).

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| ramassages non-arme | 82 | 36 |
| ETIQUETES (un seul rang dans la fenetre) | 16 (**19,5 %**) | 9 (**25,0 %**) |
| ambigus (plusieurs rangs) | 0 | 0 |
| TEMOIN decale (pire des 3) | **0 (0,0 %)** | **0 (0,0 %)** |
| identifiants distincts etiquetes | 6 | 4 |
| dont en collision intra-film | 1 (16,7 %) | 0 |

**Verdicts : N1 (>= 30 % etiquetes) NON TENU sur les deux films · N2 (temoin < 10 %) TENU, a
0,0 % · N3 (collisions <= 20 %) TENU.**

### La table, et ce qu'elle vaut

| identifiant | 000d5950 | 00502e52 | lecture croisee |
|---|---|---|---|
| `eef5d48d` | rang 21 x2 = **Thruster / propulseur** | rang 21 x1 | **COHERENT** sur deux films |
| `8e2dc574` | rang 19 x2 | rang 19 x2 | **COHERENT** (rang 19 non nomme par la palette) |
| `8c77ffe7` | rangs 19 x4 **et** 20 x3 | rang 20 x5 | collision intra-film ; majoritairement 20 = Grappleshot |
| `bcabbe43` | rang 22 x1 | rang 20 x1 | **COLLISION INTER-FILM** — non nommable |
| `72199cba` | rang 22 x3 | absent | un seul film |
| `caaadcb0` | rang 19 x1 | absent | un seul film |

**Deux identifiants seulement sont coherents sur les deux films** (`eef5d48d` -> Thruster,
`8e2dc574` -> rang 19). Un est en collision inter-film (`bcabbe43`). Le reste tient a un film.
La voie NE PERMET PAS de publier une table id -> nom : elle est juste, elle n'est pas assez
couvrante ni assez stable. Aucun vote majoritaire n'a ete applique — deux etiquettes pour une
valeur restent deux etiquettes.

**Reserve de methode** : le croisement inter-film est fait A LA MAIN, en lisant deux sorties
d'instrument. La regle « un film par process » interdit de le faire dans un seul test ; c'est
une limite assumee, pas un oubli.

## ETAPE 3 — LE LIEN OBJET-AU-SOL, RE-MESURE : REFUTATION NI LEVEE NI CONFIRMEE

**D'abord une correction de l'enonce.** On m'a presente « l'instant natif exact » comme l'idee
neuve autorisant de retenter. Ce n'en est qu'une a moitie : les emissions i48 sont DEJA datees a
la milliseconde, et la refutation ecrite dans `equipment_placements.go` impute l'echec a la
DENSITE d'objets, pas a un flou temporel. Ce que le canal natif apporte reellement, et qui est
mesure, c'est la POPULATION : 70 a 72 % des ramassages non-arme n'ont aucune emission i48 dans
la fenetre — la mesure D ne les voyait pas.

Preambule respecte : `glResolve` (verrou de process + largeurs d'axe de la carte), vies d'objets
par la chaine de PRODUCTION (`decodeFilmPlacements` pour la calibration MPP, puis
`decodeFilmPadScans`), jamais une copie de chaine.

000d5950, 82 ramassages non-arme mesures, 477 vies ti=37, calibration MPP 9/5 :

| population | n | mediane | part < 1 m |
|---|---|---|---|
| **REEL** — ramasseur -> objet ti=37 vivant, a l'instant natif | 82 | **1,33 m** | 46,3 % |
| TEMOIN — autre bipede vivant, meme instant | 429 | 9,57 m | 4,0 % |
| TEMOIN — meme ramasseur, instants decales | 27 | 15,10 m | 3,7 % |

**Verdicts : L1 (mediane < 1,0 m ET part > 60 %) NON TENU · L2 (temoin autre bipede >= 3x)
TENU, 7,2x · L3 (temoin decale >= 3x) TENU, 11,4x · L4 (refutation confirmee, mediane >= 1,4 m)
NON TENU.**

**Lecture honnete : ni l'un ni l'autre.** Le lien est REEL — le ramasseur est 7 a 11 fois plus
proche d'un objet que n'importe quel autre bipede au meme instant, et que lui-meme a un autre
instant. Mais 1,33 m ne permet pas d'attribuer UN objet : c'est mieux que les 1,4-1,7 m de la
mesure D, moins bon que les 0,61-0,75 m des armes, et la part sous le metre plafonne a 46 %.
La refutation n'est donc pas levee (on ne peut pas publier le lien) et pas confirmee non plus
(le signal existe, il n'est pas noye). **Elle est DEPLACEE** : ce n'est plus « le lien n'existe
pas », c'est « le lien existe et la resolution spatiale ne suffit pas a le rendre injectif ».

### Controle d'instrument, et il valide le preambule

Rejoue avec une carte VOLONTAIREMENT FAUSSE (`aquarius` au lieu de `Cliffhanger`) : reel
**3,86 m**, temoin autre bipede 7,01 m — le rapport tombe de **7,2x a 1,8x** et la part sous le
metre de 46,3 % a **0,0 %**. Les mauvaises largeurs d'axe detruisent le signal, exactement comme
la lecon ecrite le prevoyait. Le resultat du bon cadrage n'est donc pas un artefact de mesure.

**Limite : ETAPE 3 EST MONO-FILM.** La carte de `00502e52` ne m'est pas connue, et le catalogue
en compte plus de cent. La deviner en essayant celles qui donnent le meilleur resultat serait
exactement l'ajustement que cette note s'interdit. Mesure a un film, dit comme tel.

## CE QUI DEVIENT PUBLIABLE, ET CE QUI NE L'EST PAS

- **PUBLIABLE** : rien de ce lot. Aucune des trois etapes n'atteint son seuil de publication.
- **ACQUIS COMME CONNAISSANCE** : la classe 3 est l'equipement au sens d'i48 (40-45 % contre
  temoin 0,0 %, deux films) — l'enonce initial etait inverse. Deux identifiants sont nommes de
  facon coherente sur deux films (`eef5d48d` = Thruster, `8e2dc574` = rang 19).
- **NON CONCLU** : ce qu'est la classe 2. Le juge grenades ne la designe pas de facon
  reproductible.
- **REFUTATION DEPLACEE, PAS LEVEE** : le lien ramasseur -> objet au sol pour l'equipement.

## VOLET B (2026-09-01) — L'ETAT DES IMAGES-CLES : la couverture esperee N'EST PAS AU RENDEZ-VOUS

Question utilisateur : elargir la correlation d'inventaire au-dela du delta i48. Le levier non
tente etait l'ETAT COMPLET des images-cles, diffe entre deux releves — le patron qui a servi
d'oracle aux armes.

**Recensement d'abord, sans rien reimplementer.** `ScanFilmKeyframeInventory` rend deja, par
bipede et par image-cle : `Grenades [4]uint32` + `GrenadesRead` (etat complet des compteurs) et
`AbilityRank` (rang de palette). RESERVE ECRITE AVANT LA MESURE : `AbilityRank` ne se lit que
dans la fenetre 16..23 de la palette. Mes deux films sont de la palette `famille_b` (rangs
19-22), donc ENTIEREMENT dedans — la limitation ne mord pas ici, elle mordrait sur un film de
famille A (rangs 1-12).

| mesure | 000d5950 | 00502e52 |
|---|---|---|
| releves d'inventaire · rang lu · grenades lues | 184 · 132 · 150 | 209 · 152 · 170 |
| ramassages non-arme | 82 | 36 |
| **ETIQUETES (une seule etiquette)** | **24 (29,3 %)** | **7 (19,4 %)** |
| ambigus (plusieurs changements dans la fenetre) | 15 | 8 |
| sans changement | 13 | 10 |
| sans paire d'images-cles | 30 | 11 |
| **TEMOIN decale (pire des 3)** | **1 (1,2 %)** | **0 (0,0 %)** |
| identifiants etiquetes · dont en COLLISION | 6 · **4 (66,7 %)** | 4 · 1 (25,0 %) |

**VERDICT B1 (>= 50 % etiquetes) : NON TENU** — 29,3 % et 19,4 %, c'est-a-dire **pas mieux que
la voie delta** (19,5 % et 25,0 %). La couverture qui justifiait la voie n'existe pas : entre
les ramassages sans paire d'images-cles (30 et 11), les ambigus (15 et 8) et ceux sans aucun
changement (13 et 10), la fenetre de vingt secondes perd les deux tiers de la population.

**VERDICT B2 (temoin < 25 %) : TENU, et tres largement** — 1,2 % et 0,0 %. Le risque structurel
que je redoutais (« il se passe toujours quelque chose en 20 s ») ne se materialise PAS.
L'etiquetage mesure bien quelque chose.

**VERDICT B3 (concordance) : TENU la ou les deux voies se recouvrent.** `eef5d48d` recoit
**rang 21 (Thruster)** par la voie delta ET par la voie images-cles, sur les DEUX films.
`8e2dc574` recoit rang 19 par les deux, mais la voie images-cles y ajoute une etiquette
parasite « grenade rang 1 » — collision.

**Le defaut propre de cette voie est le BRUIT** : 66,7 % de collisions sur 000d5950 contre
16,7 % pour la voie delta. Vingt secondes melangent les gestes, et l'etiquetage attribue au
ramassage ce qui s'est passe a cote.

### Un signal qualitatif, publie comme tel

Dans la table, les identifiants que le lot 4 avait classes en **classe 2** recoivent des
etiquettes GRENADE (`bcabbe43` : grenade rang 0 x3 · `caaadcb0` : grenade rang 1 x4), tandis
que ceux de **classe 3** recoivent des etiquettes RANG (`eef5d48d` : rang 21). Cela CONVERGE
avec l'hypothese « classe 2 = grenades » que le juge J2 du lot 4 n'avait pas su trancher. Ce
n'est PAS une mesure — je n'ai pas croise classe x type d'etiquette avec un temoin — c'est une
observation de table, et c'est la piste la plus prometteuse pour elucider la classe 2.

### Conclusion du volet B

La voie des images-cles **ne remplace pas** la voie delta : meme couverture, plus de bruit.
Elle apporte en revanche une **corroboration croisee** (B3) qui vaut mieux qu'une couverture :
deux voies independantes disent rang 21 pour `eef5d48d` sur deux films. Aucune table n'est
publiable pour autant.

## Ce qu'il faudrait pour aller plus loin

1. Un film dont la CARTE est connue pour rejouer l'etape 3 en croise.
2. Elargir la fenetre d'appariement de l'etape 1 au-dela de 500 ms et mesurer ce que la
   couverture gagne contre ce que le temoin perd — 19,5 % vient peut-etre de la fenetre, pas de
   la voie.
3. Pour la classe 2 : le pool Lua en clair des tags `hsc*` comme VOCABULAIRE (jamais un
   negatif), et les lancers `biped_throw_initiate` du chantier trame comme troisieme juge.

## Reproduire

```
cd apps/go-api
BIPED_PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 \
  go test ./internal/analysis/replay/ -run 'EquipmentPickupNaming|EquipmentPickupClassSemantics' -v
PICKUP_FILM=<depot>/data/cache/film_chunks/000d5950 PICKUP_MAP=Cliffhanger \
  go test ./internal/analysis/replay/ -run EquipmentPickupGroundLinkAtNativeInstant -v
```

Un film par process, lecture seule, aucune cuisson.
