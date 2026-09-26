# Lot C — phase 1b : les trois canaux portes, et la mesure d'etat contre G-C1

> Ouverte par `LOTC_ARBITRAGE_PHASE0.md` ; grammaire lue en phase 1a (`LOTC_PHASE1A.md`).
> Base : `797d635bc` (lot 0 fusionne). Perimetre : C.1b.1 (port), C.0.2, C.1b.3 (mesure), C.1b.4.
> **La phase 2 (publication `zoneStates`) n'est PAS faite** : elle est serialisee (D16) et attend
> l'item 6 phase 3. Mesures du 2026-08-18. Gates : `LOTC_gates.log`. Sorties : `lotC/*.tsv`.

## 1. C.1b.1 — le port

| ti | i | composant | statut table | deser_addr | code_source |
|---|---|---|---|---|---|
| 12 | 14 | `managed-navpoint-radial-progress` | `porte` | `FUN_140fc8d14` | `components_managed_object.go:187` |
| 10 | 1 | `managed-object-boundary-color-component` | `porte` | `FUN_142ed52b4` | `components_managed_object.go:93` |
| 10 | 26..29 | `managed-object-rtpc-component` (4 lignes) | `porte` | `FUN_140796d38` | `components_managed_object.go:119` |

`traverse.go` ne grossit que de six lignes : trois `case` de routage (855-866), corps dans le
nouveau fichier. Le bloc de hook de ti=10 (enum, hook, setter) a ete DEPLACE de
`components_walk_batch9.go` vers `components_managed_object.go`, ou il est etendu ; aucune ligne
de table existante n'a bouge (le deser d'i0 reste a `components_walk_batch9.go:27`).

**Statut `porte` et non `partiel` pour les rtpc, et c'est motive.** Leur largeur depend de la
donnee (32 bits si l'identifiant est nul, 54 sinon), mais le branchement porte sur une valeur LUE
DANS LE FLUX, les deux branches sont integralement consommees, et le cas rend toujours `true` :
aucun desync n'est possible. `partiel` est reserve aux cas ou la traversee peut lacher.

**Les quatre rtpc partagent un lecteur, et `consumeByName` ne recoit pas l'index du composant.**
Le jeu le lit dans le descripteur (`param_1 + 8`). Ce n'est pas une perte : l'IDENTIFIANT de
32 bits est la vraie identite du canal, et la mesure le confirme (section 5). Publier
l'identifiant plutot qu'un index de registre respecte en outre la regle du lot 0 — le decoupage
du registre change avec le build.

### Vecteurs : 14 cas sur octets reels, verts du premier coup

`components_managed_object_test.go` — les octets viennent des `*_vecteurs.tsv` de la phase 1a
(records a masque SINGLETON, seul cas ou la position de la charge utile est connue sans porter
les voisins). Aucun film n'est lu : les octets sont recopies, les tests tournent en CI.

| test | cas | ce qu'il contraint |
|---|---|---|
| `TestNavpointRadialProgressVecteurs` | 3 | quantum, 8 bits consommes, dequantification |
| `TestManagedObjectBoundaryColorVecteurs` | 3 | 4 quanta RGBA, 32 bits consommes |
| `TestManagedObjectRTPCVecteurs` | 4 | id + valeur, 54 bits ; branche « id nul » a 32 bits (vecteur SYNTHETIQUE, signale : aucun record singleton du corpus ne la porte) |
| `TestManagedObjectRTPCIdentifiantConstant` | 4 | l'identifiant est constant et la valeur CROIT — le controle qui a valide le cadrage |
| `TestZoneHooksConsommentLesMemesBitsSansHook` | 4 | poser un hook ne change pas la consommation |
| `TestZoneDequantification` | 5 | la convention retenue, figee |

### Gate de portage : `DesyncAt` sur les 12 films, avant / apres

Instrument du lot 0 (`delta_walk_witness_test.go`, 12 premiers chunks). **Aucun film ne recule ;
sept progressent.**

| film | records avant -> apres | traversee ABOUTIE avant -> apres | `ported=false` |
|---|---|---|---|
| 7344d24f | 33 025 -> 33 029 | 25 016 -> **25 021** (75,749 -> 75,755 %) | 8 009 -> 8 008 |
| 696a9d7c | 32 713 -> 32 713 | 24 652 -> **24 653** | 8 061 -> 8 060 |
| 0a247154 | 33 226 | 24 468 (inchange) | 8 758 |
| 01e1f945 | 37 966 -> 37 967 | 29 138 -> **29 140** | 8 828 -> 8 827 |
| 606d9844 | 34 512 | 26 642 (inchange) | 7 870 |
| 8076f97f | 32 970 | 24 889 (inchange) | 8 081 |
| 64e8adfa | 39 776 | 31 934 -> **31 935** | 7 842 -> 7 841 |
| 530820e5 | 35 539 -> 35 542 | 26 238 -> **26 241** | 9 301 |
| 53ce4390 | 38 007 -> 38 008 | 28 584 -> **28 586** | 9 423 -> 9 422 |
| 24dbb67d | 39 634 | 30 917 (inchange) | 8 717 |
| 000d5950 | 38 860 -> 38 862 | 30 058 -> **30 060** | 8 802 |
| 06dfe6d9 | 10 607 | 8 494 (inchange) | 2 113 |

Le gate demandait « non aggrave » : il est tenu, et strictement ameliore sur 7 films. La table
figee de l'instrument a ete mise a jour pour `000d5950` et `64e8adfa` avec sa justification
ecrite sur place, comme son contrat l'exige.

**Le gain est PETIT, et il faut dire pourquoi** : dans la fenetre de douze chunks les records
ti=10/ti=12 sont une petite part du trafic (le bipede domine), et une traversee n'aboutit que si
TOUS ses composants annonces sont portes — ti=10 en compte encore 26 non portes. Le port ne
« repare » donc pas la traversee de ti=10 ; il retire trois murs sur vingt-neuf.

## 2. Les temoins d'ancrage, publies a chaque passe (exigence du registre)

| film | bande fantome (vide) | purete `ti=4` (records / hors grammaire) |
|---|---|---|
| 7344d24f | 26 899 records | 35 698 / **0,68 %** |
| 696a9d7c | 26 217 | 33 874 / **1,10 %** |
| 0a247154 | 31 539 | 46 682 / **1,05 %** |
| 01e1f945 | 14 236 | 32 534 / **0,73 %** |
| 606d9844 | 4 139 | 14 656 / **0,70 %** |
| 8076f97f | 7 911 | 21 196 / **0,41 %** |

L'ancrage est pur a 98,9-99,6 % sur une bande d'un seul slot. Le bruit reste un effet de LARGEUR
DE BANDE, comme la phase 1a l'avait etabli.

## 3. C.0.2 — les 32 drapeaux de `boundary-visibility` (ti=10 i0)

Mesure sur `7344d24f` : **1 347 records, 1 162 valeurs distinctes, les 32 bits utilises**, chacun
leve dans 22,0 % a 60,1 % des records (b12 le plus rare a 22,0 %, b14 le plus frequent a 60,1 %).
Valeur la plus frequente : 1,6 % des records. 1 182 transitions, dont 371 (31,4 %) a moins de 2 s
d'un evenement d'objectif.

**Les trois hypotheses du plan sont departagees, et aucune ne tient :**

| hypothese | ce qu'elle predit | mesure |
|---|---|---|
| par JOUEUR | 8 bits utiles, 24 morts | 32 bits utiles — REFUTEE |
| par EQUIPE | 2 bits utiles | 32 bits utiles — REFUTEE |
| par ETAT | peu de valeurs distinctes | 1 162 sur 1 347 — REFUTEE |

**Ce que la mesure suggere a la place** : une distribution quasi uniforme sur les 32 bits, sans
valeur dominante, est la signature d'un champ a granularite FINE — par exemple une visibilite par
SEGMENT de bordure ou par point de navigation (ti=10 declare 16 `navpoint`), ou une visibilite
par observateur. Ce n'est pas un masque de propriete.

**Reserve honnete sur ce chiffre** : 1 347 records, contre un plancher de bruit de 317 annonces
par index sur la bande ti=10 de ce film (phase 0) — le rapport signal/bruit est d'environ 4,2x,
donc une part non negligeable de l'echantillon peut etre du bruit d'ancrage. Une lecture
definitive de ce composant demande une bande plus propre, ce qui est un probleme d'ancrage et non
de grammaire.

## 4. C.1b.3 (a) — `radial-progress` : les rampes existent, le gate ne passe pas

| film | mode | valeurs | slots | rampes | amplitude (min/med/max, sur 256) | captures couvertes | temoin +20 s | **niveau du hasard** | concurrence (clause KOTH) |
|---|---|---|---|---|---|---|---|---|---|
| 7344d24f | Strongholds | 16 679 | 15 | 62 | 16 / 126 / 253 | **68/71 = 95,8 %** | 36,6 % | **46,1 %** | 9,6 % |
| 696a9d7c | Strongholds | 17 298 | 8 | 50 | 48 / 126 / 200 | **70/77 = 90,9 %** | 28,6 % | **39,9 %** | 1,8 % |
| 0a247154 | KOTH | 355 | 34 | 5 | 28 / 112 / 158 | pas d'oracle | — | 3,3 % | 40,5 % |
| 01e1f945 | KOTH | 7 180 | 10 | 62 | 21 / 124 / 187 | pas d'oracle | — | 51,9 % | 79,1 % |
| 606d9844 | KOTH | 1 833 | 6 | 9 | 120 / 124 / 127 | pas d'oracle | — | 23,5 % | **100,0 %** |
| 8076f97f | KOTH | 2 954 | 7 | 26 | 16 / 122 / 243 | pas d'oracle | — | 37,2 % | 45,0 % |

**Clause principale : TENUE, et largement.** 95,8 % et 90,9 % des captures ont un sommet de rampe
dans +/- 2 s, pour un seuil de 80 %. Les rampes sont franches : amplitude mediane 126 quanta sur
256, soit la moitie de la plage, et jusqu'a 253. Dans l'autre sens, 52 des 62 sommets (83,9 %)
tombent a moins de 2 s d'un evenement d'objectif.

**Clause du temoin : NON TENUE.** 36,6 % et 28,6 % pour un seuil de <= 20 %.

**Ce que le niveau du hasard ajoute, et pourquoi il est publie.** Avec N sommets de rampe et une
fenetre de +/- 2 s sur une duree T, une capture tombe pres d'un sommet par pur hasard avec une
probabilite d'environ `N x 4 s / T` : **46,1 %** sur `7344d24f`, **39,9 %** sur `696a9d7c`. Le
temoin decale (36,6 % et 28,6 %) est donc SOUS le hasard, et le signal reel (95,8 % et 90,9 %)
est a plus du DOUBLE. La discrimination est reelle et forte ; c'est le seuil de 20 % qui est
inatteignable pour un canal produisant 50 a 62 rampes en neuf minutes.

**Clause KOTH : NON TENUE.** 9,6 %, 1,8 %, 40,5 %, 79,1 %, 100,0 %, 45,0 % pour un seuil de 90 %.
Plusieurs points de navigation rampent EN PARALLELE. Le seul film a 100 % (`606d9844`) est aussi
celui qui porte le moins de rampes (9). La lecture « une seule colline se remplit a la fois » est
donc fausse au niveau du NAVPOINT : ti=12 porte plusieurs marqueurs simultanement (plusieurs
zones candidates, et vraisemblablement un marqueur par equipe).

> **VERDICT G-C1 (a) : NON TENU** — clause principale tenue, clause du temoin et clause KOTH non
> tenues. Seuils ni rebaisses ni reinterpretes.

## 5. C.1b.3 (b) — `boundary-color` : REFUTE sur le fond

| film | records | quadruplets distincts (seuil <= 8) | changements | captures couvertes (seuil 80 %) |
|---|---|---|---|---|
| 7344d24f | 1 194 | **996** | 1 023 | 67/71 = 94,4 % |
| 696a9d7c | 697 | **390** | — | 57/77 = 74,0 % |
| 0a247154 | 1 095 | 451 | — | pas d'oracle |
| 01e1f945 | 822 | 458 | — | pas d'oracle |
| 606d9844 | 41 | 30 | — | pas d'oracle |
| 8076f97f | 191 | 101 | — | pas d'oracle |

**La clause « <= 8 quadruplets distincts » est violee de deux ordres de grandeur.**
`boundary-color` n'est PAS un enumere d'etat : c'est une couleur CONTINUE, vraisemblablement
animee (pulsation, fondu). La clause de changement pres des captures est tenue sur un film
(94,4 %) et pas sur l'autre (74,0 %), ce qui est attendu d'un canal qui change tout le temps.

> **VERDICT G-C1 (b) : NON TENU**, et ici le negatif porte sur la NATURE du canal, pas sur un
> seuil : une couleur continue ne peut pas servir d'etat de zone enumerable.

**CORRECTION D'UNE LECTURE DE LA PHASE 1a.** J'avais releve quatre niveaux dominants sur le
premier octet (55, 119, 183, 247, espaces de 64) et laisse ouverte l'hypothese d'une palette a
quatre niveaux. Sur l'echantillon COMPLET (et non plus les seuls records a masque singleton) le
premier octet prend beaucoup de valeurs : 119 a 7,5 %, 183 a 4,9 %, 55 a 4,2 %, **16 a 3,9 %**,
247 a 3,7 %, 117 a 2,8 %, 116 a 1,8 %, 0 a 1,4 %. **Les quatre niveaux etaient un artefact de
l'echantillon singleton, pas une propriete du canal.** Reponse a la question « que sont les
4 niveaux, equipe / neutre / conteste ? » : ils ne sont rien de tout cela — ils n'existent pas
hors de ce sous-echantillon.

## 6. C.1b.3 (c) — `rtpc` : ce que suit la valeur

Sur `7344d24f` : 23 470 records, **811 identifiants distincts**, dont deux couvrent 95,3 % —
`0x7CBF0066` (69,7 %) et `0x06854540` (25,6 %), exactement les deux constantes relevees en
phase 1a. **100,0 % des records portent une valeur** (l'identifiant n'est jamais nul). La queue de
809 identifiants rares est coherente avec les 22,56 % de records hors grammaire de la bande ti=10
mesures en phase 0 : c'est du bruit d'ancrage, pas 811 canaux sonores.

La valeur de 22 bits produit **124 rampes monotones sur 78 slots**, de meme forme que celles de
`radial-progress`. Le canal suit donc bien une PROGRESSION et non un volume : un volume sonore
oscillerait autour d'un niveau, il ne monterait pas par paliers reguliers jusqu'a un sommet. La
correlation avec (a) est de forme, pas encore d'appariement slot a slot — l'appariement demande
le pont objet ti=10 -> navpoint ti=12, qui n'est pas etabli.

## 7. Statut des items

- [x] **C.1b.1** — port des 3 desers, 6 lignes de table, 14 vecteurs verts, `DesyncAt` non
  aggrave (ameliore sur 7 films sur 12).
- [x] **C.0.2** — les 32 drapeaux sont lus et distribues ; les trois hypotheses du plan sont
  REFUTEES ; une quatrieme (granularite fine, par segment ou par navpoint) est proposee avec sa
  reserve de bruit.
- [x] **C.1b.3** — mesure jouee sur 6 films (2 Strongholds + 4 KOTH), verdict par canal ci-dessus.
- [!] **G-C1** — NON TENU sur les trois canaux. Details et chiffres sections 4 a 6.
- [ ] **Phase 2** — NON FAITE, et volontairement : serialisee par D16 derriere l'item 6 phase 3.

## 8. Ce que le lot C a etabli au total, pour la decision du superviseur

1. `ti=23` est ABSENT (11 films) — definitif.
2. `ti=10` et `ti=12` sont la machinerie vivante du mode, et trois de leurs canaux sont
   maintenant PORTES, testes sur octets reels, sans regression de traversee.
3. `radial-progress` est bien la jauge de capture : 90,9 % et 95,8 % des captures ont un sommet de
   rampe a moins de 2 s, pour un hasard a 40-46 %. **C'est le resultat exploitable du lot.**
4. Mais AUCUN des trois canaux ne donne un ETAT DE ZONE ENUMERABLE : la couleur est continue
   (996 quadruplets), la visibilite est a granularite fine (32 bits actifs), et plusieurs
   navpoints rampent en parallele (concurrence 1,8 a 79 %).
5. Il manque le **pont objet -> zone du catalogue**. Le gate le prevoyait via `AttributeZones` et
   les positions de joueur ; sans un etat de zone enumerable a apparier, ce pont n'a pas d'objet a
   porter. C'est la vraie condition d'une publication `zoneStates`, et elle n'est pas remplie.

## 9. Decouvertes (hors perimetre — notees, NON traitees)

1. **Les seuils de G-C1 (a) reproduisent le defaut de la clause 2 du gate 0** : un seuil de temoin
   a 20 % suppose un canal RARE, alors que le canal produit 50 a 62 rampes par match. Le niveau du
   hasard (40-46 %) devrait etre le denominateur de tout temoin futur sur ce type de canal.
2. **La clause KOTH « une seule rampe active » est fausse au niveau du navpoint** : ti=12 porte 6
   a 34 slots actifs. Une mesure « une seule colline » devrait porter sur un objet ti=10 apparie a
   une zone, pas sur les marqueurs.
3. **`boundary-color` est une couleur ANIMEE** — utile pour un rendu fidele (le lot C phase 2
   aurait pu la publier telle quelle pour teinter une zone), inutilisable comme etat.
4. **Les 811 identifiants rtpc** confirment quantitativement le bruit d'ancrage de la bande ti=10
   (22,56 % hors grammaire) : deux identifiants couvrent 95,3 %, le reste est du faux positif.
   C'est un second temoin de bruit, gratuit, que d'autres lots pourraient reutiliser.
5. **`606d9844` a une concurrence de 100 %** avec 9 rampes seulement : sur les petits films la
   clause KOTH passe pour une raison de volume, pas de semantique. Un seuil sur un compte doit
   toujours publier son denominateur.
