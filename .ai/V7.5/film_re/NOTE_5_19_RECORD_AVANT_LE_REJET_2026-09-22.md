# NOTE 5.19 — LE RECORD AVANT LE REJET : LA DIFFERENTIELLE, PUIS LES ECRIVAINS

Lot 5.19, branche `feat/decfilm-69`, base `d976be4e0` (lots 5.1 a 5.18, grammar-2026-09-22.7,
schema 67). Les lots 5.15 a 5.18 ont instruit le residu de `bfecd02b` sans le reduire, chacun en
refutant le suspect du precedent. Ce lot change de methode : une DIFFERENTIELLE d abord, qui
LOCALISE sans conclure, puis l ecrivain des composants qu elle nomme.

---

## 1. L INSTRUMENT

`mouvement_5_19_differentielle_research_test.go` (collecte) et
`mouvement_5_19_tableaux_research_test.go` (publication, deplacement pur sous le seuil de
500 lignes), tag `research`, test `TestDiff519`. Une passe, un decodage par paquet (les deux
instruments du 5.15 en faisaient deux et muaient le monde deux fois).

```
MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=snowbound \
  go test -tags=research -count=1 -v -timeout 60m -run '^TestDiff519$' \
  ./internal/games/halo_infinite/film/internal/grammar/
```

**Le gate du 5.16/5.18 est reproduit AU PAQUET avant toute lecture**, donc l instrument est bien
celui qui a mesure le residu :

| | `dad793c7` | `bfecd02b` |
|---|---:|---:|
| paquets delta | 5 365 | 30 387 |
| non localises | 5 | 845 |
| FERMES a reste NUL | **5 354** | **2 884** |
| REJET de slot inconnu (trois rangs portes) | **1** | **23 325** |
| debordements | 2 | 32 |
| autres restes (terminateur / desynchronisation / rejet partiel) | 8 | 4 146 |

(Les 23 325 sont les rejets dont les TROIS rangs sont portes ; le 5.15 comptait 23 452 rejets
sans cette condition, et 282 de plus sortent ici en « reste hors bourrage · rejet ».)

---

## 2. CE QUE LA DIFFERENTIELLE MONTRE — ET SES DEUX CONFONDANTS

### 2.1 Le masque du dernier record ne distingue PAS les deux populations

Les masques les plus frequents du dernier record avant le rejet sont EXACTEMENT ceux des paquets
qui ferment : `ti=35 masque 0x2200003` est le premier des deux cotes (8 007 fautifs, 155 fermes),
`0x2000003` le troisieme des deux cotes, `ti=40 masque 0x200000f` le quatrieme chez l un et le
cinquieme chez l autre. **Un paquet ne faute pas parce qu il porte une forme de record que les
autres n ont pas.**

### 2.2 Les largeurs lues par composant sont les MEMES des deux cotes

`i25 unit-command-tick` 10 bits (20 997 fautifs / 397 fermes), `i0 object-position-dynamic-
precision` 54 bits (20 458 / 395), `i1 object-translational-velocity-dynamic-precision` 31 bits
(19 138 / 329), `i21 unit-desired-aiming-vector` 25 bits (13 488 / 216), `i5 object-shield-
vitality` 29 bits (5 807 / 22). Les seules largeurs « que seuls les fautifs portent » sont des
classes a 1 a 6 occurrences — la queue de bruit d un curseur deja faux, pas un signal.

**`i25` est re-confirme comme confondant** (5.15.1 (i)) : premier par volume des deux cotes, MEME
largeur des deux cotes.

### 2.3 Premier confondant : la DENSITE

La comparaison « paquets fautifs contre paquets fermes » est biaisee parce que les paquets qui
ferment sont d abord des paquets PAUVRES : `ti=35` est present dans 96,6 % des fautifs contre
30,4 % des fermes, et 61,1 % des paquets fermes n ont pour dernier record qu un `ti=4
high-frequency` d un seul composant. Une presence « 1 339 fois chez les fautifs, 0 fois chez les
fermes » (`i32 weapon-state-overheated`) se lit d abord comme « ce composant vit dans les paquets
denses ».

### 2.4 Second confondant : la DOSE — et elle refute la loi geometrique

| deltas de bipede dans le paquet | fermes | fautifs | taux de fermeture |
|---:|---:|---:|---:|
| 0 | 2 007 | 786 | **71,9 %** |
| 1 | 3 | 52 | 5,5 % |
| 2 | 0 | 950 | 0,0 % |
| 3 | 4 | 3 227 | 0,1 % |
| 4 | 53 | 4 566 | 1,1 % |
| 5 | 303 | 7 417 | 3,9 % |
| 6 | 231 | 4 629 | 4,8 % |
| 7 | 208 | 1 225 | 14,5 % |
| 8 | 75 | 473 | 13,7 % |

Un paquet SANS delta de bipede ferme 7 fois sur 10 ; des qu il en porte un, il ne ferme plus. Et
la distribution des fautifs n est PAS geometrique (elle serait decroissante) : c est une cloche
centree sur 5, donc le nombre de bipedes lus avant la faute, tronque par elle.

---

## 3. LA DIFFERENTIELLE INTERNE — CELLE QUI N A AUCUN CONFONDANT

Les deux confondants disparaissent si l on compare, **DANS LE MEME PAQUET FAUTIF**, le DERNIER
record (celui apres lequel le curseur est faux) a TOUS CEUX QUI LE PRECEDENT (lus juste, puisque
le record suivant s est decode derriere eux). Meme film, meme trame, meme paquet, meme carte.

> **taux de faute d une classe = dernier / (dernier + precedents)**

23 325 derniers records contre 114 147 precedents. Voici la table des suspects, ordonnee.

### 3.1 Par TAUX de faute (les classes presque toujours terminales)

| classe | taux | dernier | precedents |
|---|---:|---:|---:|
| `ti=40` masque `0x200000a` (i1,i3,i25) | **100,0 %** | 45 | 0 |
| `ti=40` masque `0x200001f` (i0..i4,i25) | **98,8 %** | 82 | 1 |
| `ti=35` masque `0x40000002200023` (bit **54**) | **92,6 %** | 25 | 2 |
| `ti=40` masque `0x2000007` | **92,0 %** | 69 | 6 |
| `ti=42` masque `0xa` | **90,6 %** | 87 | 9 |
| `ti=40` masque `0x200000e` | **88,9 %** | 48 | 6 |
| `ti=40` masque `0x10` (i4 SEUL) | **87,2 %** | 238 | 35 |
| `ti=0` masque vide (record de type 2) | 86,2 % | 75 | 12 |
| `ti=35` masque `0x40000002200003` (bit **54**) | 83,3 % | 50 | 10 |
| `ti=32` masque `0xf` | 82,6 % | 76 | 16 |
| `ti=42` masque `0xe` | 81,4 % | 70 | 16 |
| `ti= 2` masque `0x1` | 80,3 % | 376 | 92 |
| `ti=40` masque `0x200000f` | **75,0 %** | **1 841** | 615 |
| `ti=37` masque `0xc` / `0xd` | 62,8 / 63,9 % | 98 / 69 | 58 / 39 |
| `ti=10` masque `0x4000000` | 53,6 % | 239 | 207 |
| `ti=42` masque `0xf` | 48,3 % | 143 | 153 |
| `ti=37` masque `0xf` | 43,3 % | 311 | 408 |
| `ti=35` masques usuels (`0x2200003`, `0x2200023`, `0x2000003`, …) | **10,7 a 25,1 %** | 18 856 | 85 889 |
| `ti= 4` masque `0x1` (`high-frequency`) | **1,1 %** | 244 | 22 604 |

### 3.2 Par composant ET par largeur lue

| composant | largeur | taux | dernier | precedents |
|---|---:|---:|---:|---:|
| `ti=40 i1 object-translational-velocity-dynamic-precision` | 2 | **92,7 %** | 191 | 15 |
| `ti=40 i3 object-angular-velocity-dynamic-precision` | 2 | **92,5 %** | 148 | 12 |
| `ti=35 i54 biped-mobility-action-component` | **344** | **90,3 %** | 102 | 11 |
| `ti=40 i2 object-forward-and-up-dynamic-precision` | 31 | **89,7 %** | 504 | 58 |
| `ti=40 i4 object-body-vitality-component` | 11 | **88,9 %** | 367 | 46 |
| `ti=32 i0 tacmap-areaofinterest` | 66 / 67 | 87,0 / 84,1 % | 20 / 90 | 3 / 17 |
| `ti= 2 i0 game-engine-team-mapping-component` | 58 / 62 | 85,7 / 77,8 % | 126 / 242 | 21 / 69 |
| `ti=42 i1 object-translational-velocity-component` | 1 | 78,6 % | 198 | 54 |
| `ti=40 i25 unit-command-tick-component` | 10 | **77,0 %** | **2 141** | 639 |
| `ti=40 i0 object-position-dynamic-precision` | 54 | **76,2 %** | **2 016** | 629 |
| `ti=40 i1 object-translational-velocity-dynamic-precision` | 31 | 75,7 % | 1 939 | 624 |
| `ti=40 i3 object-angular-velocity-dynamic-precision` | 29 | 75,6 % | 1 923 | 621 |
| `ti=40 i2 object-forward-and-up-dynamic-precision` | 64 | 73,0 % | 1 561 | 576 |
| `ti=42 / ti=37 object-*` (sans dyn.-prec.) | 28 / 30 / 52 | 41 a 57 % | ~2 400 | ~2 900 |
| `ti=10 i26 managed-object-rtpc-component` | 54 | 53,6 % | 239 | 207 |
| `ti=35 i1 object-translational-velocity-dynamic-precision` | 31 | 20,7 % | 17 199 | 66 018 |
| `ti=35 i0 object-position-dynamic-precision` | 54 | 20,0 % | 18 442 | 73 839 |
| `ti=35 i21 unit-desired-aiming-vector` | 25 | 19,1 % | 13 488 | 56 956 |
| `ti=35 i25 unit-command-tick` | 10 | 18,0 % | 18 856 | 85 889 |
| `ti=35 i5 object-shield-vitality` | 29 | 13,9 % | 5 807 | 35 992 |
| `ti= 4 i0 high-frequency` | 8 | **1,1 %** | 244 | 22 604 |

### 3.3 LES TROIS FAITS QUE CETTE TABLE ETABLIT

1. **Le taux de faute d un record de `ti=40` est de 73 a 89 % POUR CHACUN de ses composants,
   masque par masque.** Un record de vehicule est terminal trois fois sur quatre. Le taux est
   UNIFORME sur les six composants : ce n est donc pas « un composant de plus a porter », c est
   le record entier.
2. **Le meme composant, a la meme largeur, faute differemment selon l ARCHETYPE.**
   `object-body-vitality-component`, 11 bits : 88,9 % sur `ti=40`, 21,3 % sur `ti=35`. Le port
   lit les deux par le MEME deserialiseur et la MEME largeur (`FUN_140fb8978`). L archetype
   change donc la verite de la lecture sans changer ce que le port lit.
3. **`ti=4 high-frequency` a un taux de 1,1 % sur 22 848 lectures** : c est le seul temoin propre
   du film. Tout ce qui est au-dessus de 10 % est une lecture a instruire.

### 3.4 ET `dad793c7` NE PORTE AUCUNE DES CLASSES FAUTIVES

Sur le film a un joueur, les derniers records sont `ti=4` (97,8 %), `ti=35` (1,4 %), `ti=47`
(0,8 %), `ti=0` et `ti=5` (une fois chacun). **`ti=40`, `ti=2`, `ti=32`, `ti=42`, `ti=37` et
`ti=10` n y apparaissent jamais** — un seul rejet sur 5 365 paquets, sans record lu. Les
archetypes que la differentielle accuse sont exactement ceux que le film de calibration
n exerce pas.
