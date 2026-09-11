# Lot 6.13 — le film `b8a44fe8`, et la ZONE AVEUGLE des socles (2026-09-11)

> Worktree `LevelUp-wt-couverture-objectifs`, branche `wt/couverture-objectifs`, base `c5fd58417`.
> Preuves d'entree : `.ai/V7.5/RAPPORT_PASSAGES_DRAPEAU_2026-09-11.md` §6 (le residu nomme),
> `.ai/V7.5/AUDIT_COUVERTURE_OBJECTIFS_2026-09-10.md` §3.
> **Aucune base DuckDB n'a ete ouverte**, aucun artefact du parc n'a ete lu ni ecrit (le parc
> etait en recuisson par un autre processus). Tous les chiffres viennent de cuissons HORS LIGNE
> des chunks de film (`cmd/replay-build --facts`, racine de travail en scratchpad), confrontees a
> l'oracle API par `.ai/V7.5/outillage/couverture_objectifs`.

## 0. Verdict en six lignes

1. **Le canal objet n'est PAS muet sur `b8a44fe8`.** Il y replique 74 vies libres et en ferme
   deja 49. Il se tait sur **SEPT vies precises**, et les sept se taisent pour la MEME raison.
2. **La cause, nommee.** `flagSpawnAt` (`apps/go-api/internal/analysis/replay/flag_objects.go:437`
   avant le lot) decidait « cette naissance est-elle un drapeau qui RENTRE ? » avec le rayon du
   **LACHER** (`originDropMaxDist`, 1,5 m). Chaque socle devenait donc un **disque aveugle de
   1,5 m** : tout drapeau LACHE a portee d'un support y etait lu comme un drapeau rentre, donc
   ecarte de la seule chaine qui date un lacher volontaire.
3. **Ce n'est ni une carte, ni une variante, ni un trou de replication : c'est une SITUATION DE
   JEU** — lacher le drapeau adverse sur son propre point de livraison en attendant que son
   drapeau a soi revienne. **34 naissances du parc CTF tombent dans la zone aveugle, sur 10 des
   12 films**, dont **7 sur `b8a44fe8`, toutes au MEME socle** (celui de l'equipe 0, qui n'a
   jamais marque : 0-1 en 754 s).
4. **Le seuil ne se regle pas, il se constate.** Sur les 626 vies libres des 12 films de CTF :
   les RENTREES naissent a **0,008 m au plus** du point du catalogue, les LACHERS a **0,324 m au
   moins**. Rien entre les deux, un facteur 40.
5. **Gate ATTEINT.** `b8a44fe8` **1,234 -> 1,037**. Les 12 films CTF **1,066 -> 1,028**.
   **57 des 69 films sont identiques a l'octet** — les 12 qui changent sont exactement les 12
   films de CTF, **aucune action ne bouge**, aucune statistique publiee ne bouge.
6. **Un residu nomme et statue `[!]`** : `2533274823110022` reste a +7,8 s, et ce n'est PAS cette
   cause — c'est le SEUL span `carried_open` du parc (18,7 s), deja nomme par l'audit §3.4. Son
   chemin exact est au §6.4 et sa decouverte D1 au §10.

---

## 1. Etat des items

| # | item | statut | resultat |
|---|---|---|---|
| 1 | diagnostic differentiel `b8a44fe8` / `cde26226` / `a0c36016` / film a 1,03 | `[x]` | cause nommee `fichier:ligne`, condition et effectif (§3) |
| 2 | classe ou cas unique | `[x]` | **classe** : 34 naissances, 10 films sur 12, 5 cartes (§4) |
| 3 | correctif a la source | `[x]` | un seuil nomme + un seul predicat (§5), rouge -> vert + mutation |
| 4 | gate | `[x]` **sauf** deux sous-criteres `[!]` | detail et cause au §6 |

Commit unique : `a1948671b` `fix(6.13): la ZONE AVEUGLE des socles`.

## 2. Protocole de mesure

Deux parcs de **69 films** cuits HORS LIGNE avec les memes faits de match que B1/6.11
(`b1/facts/*.facts.json`), chunks du worktree partage lus en LECTURE SEULE :

| parc | binaire | role |
|---|---|---|
| `parcH` | HEAD `c5fd58417` | **la reference AVANT** |
| `parcC` | HEAD + le correctif | **la reference APRES** |

**Calibrage, deux fois plutot qu'une :**

1. **`parcH` reproduit la reference 6.11 A L'IDENTIQUE** : sur les **69 films**, `flagCarries` et
   `coverage.flagCarries` sont identiques champ pour champ a `parcI3` (le parc APRES de 6.11).
   L'instrument de mesure rend exactement les chiffres committes du lot 6.11 — 511 periodes,
   2 538,9 s pour 2 381,6 s, ratio **1,066**.
2. **L'instrument de recherche n'a rien change a la sortie** : le binaire de diagnostic et le
   binaire final (instrument retire) rendent `b8a44fe8` et `cde26226` **identiques a l'octet**.

---

## 3. Item 1 — le diagnostic differentiel

### 3.1 Le canal objet n'est pas muet, et les compteurs le disent

Couverture du calque, AVANT, sur les quatre films du differentiel :

| compteur | `b8a44fe8` (1,234) | `a0c36016` (1,026, MEME CARTE) | `cde26226` (1,040) | `7fce3219` (1,055) |
|---|---|---|---|---|
| `spawns` | 2 | 2 | 2 | 2 |
| `objectLives` (vies libres) | **74** | 66 | 90 | 98 |
| `closedByObject` | **49** | 41 | 63 | 72 |
| `dropsRepositioned` | 54 | 44 | 69 | 78 |
| `dropsWithheld` | **5** | 0 | 4 | 0 |
| `homeByObject` | 8 | 8 | 3 | 5 |
| `noBridge` / `noSlot` / `noTrack` / `outOfWindow` | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 | 0 / 0 / 0 / 0 |
| `assignedByPlay` / `ownFlagRefused` | 2 / 2 | 0 / 3 | 0 / 2 | 0 / 0 |

**49 portages sur 63 sont deja fermes par l'objet sur `b8a44fe8`.** Le canal fonctionne. Ce qui
manque tient en sept vies.

### 3.2 Les hypotheses ECARTEES par la mesure, une par une

| hypothese | mesure | verdict |
|---|---|---|
| Forest est une carte Forge hors catalogue | `map_objectives.json` la porte : `flag_spawn` team 0 `(-5,388 ; -19,816)`, team 1 `(-24,300 ; 22,822)`, neutre `(-14,610 ; -0,079)` | **REFUTEE** |
| `Spawns = 0`, drapeau d'equipe -1 | `spawns = 2`, `neutralFlag = false`, `teamBirths = 20`, `carrierTeamUnknown` absent | **REFUTEE** |
| la carte : `a0c36016` est sur la MEME carte, MEMES socles, MEME disposition d'equipes | `a0c36016` est a **1,026** avec les memes socles aux memes coordonnees | **REFUTEE** |
| trou d'identite / pont manquant | `noBridge = 0`, `noSlot = 0`, `spans_portes_sans_xuid = 0`, **129 actions publiees pour 129 a l'oracle, ecart 0 pour CHAQUE joueur et CHAQUE action** | **REFUTEE** |
| trou de replication / images-cles manquantes | 258 images muettes sur 4 951 (5,2 %) — mais `a0c36016` en a **0** et partage la classe, `cde26226` en a 134 (2,7 %) et est a 1,040 : aucune correlation. Et sur le span decisif, le porteur A une position a l'image de la naissance (0,152 m) | **REFUTEE** |
| le jonglage | `cde26226` a un joueur a 35 prises pour 42,3 s et est a 1,040 ; `b8a44fe8` a 25 et 13 prises. Le jonglage est partout | **REFUTEE** |

### 3.3 Ou le canal se tait : SEPT vies, UN socle

Les 74 vies libres de `b8a44fe8` se rangent en trois familles par leur distance au socle le plus
proche :

| famille | effectif | distance au socle | echantillons de position | traitement AVANT |
|---|---|---|---|---|
| nee AU POINT du catalogue (le drapeau RENTRE) | 13 | 0,003 a 0,006 m | 1 a 4 | ecartee — **correct** |
| nee **A COTE** d'un socle (le drapeau est LACHE la) | **7** | **0,614 a 1,485 m** | 24 a 54 | ecartee — **FAUX** |
| nee ailleurs | 54 | > 1,5 m | — | traitee normalement |

**Les sept sont TOUTES au socle 1**, celui de l'equipe 0 — dont les joueurs promenent le drapeau
adverse jusqu'a leur propre point de livraison et l'y posent, faute de pouvoir marquer : le match
finit **0-1**, l'equipe 0 n'a capture **aucune** fois en 754 s.

| vie | instant (ms) | distance au socle | echantillons | portage qu'elle aurait du fermer | duree publiee |
|---|---|---|---|---|---|
| 34 | 416 883 | 0,614 m | 31 | — | — |
| 36 | 423 783 | 1,222 m | 50 | — | — |
| 56 | 564 983 | 1,027 m | 35 | 47 · `2533274858911298` | 34,6 s |
| 57 | 594 283 | 1,363 m | 32 | 48 · `2535455799302553` | 16,3 s |
| 59 | 609 483 | 1,021 m | 24 | 49 · `2535442462807197` | 6,1 s |
| 61 | 615 583 | 1,485 m | 24 | 51 · `2535442462807197` | 16,7 s |
| 66 | 662 683 | 1,233 m | 29 | **57 · `2535442462807197`** | **58,1 s** |

### 3.4 LE TEMOIN, ET IL TIENT EN DEUX CENTIMETRES

Le meme joueur, `2535442462807197`, lache le drapeau DEUX FOIS au meme endroit de sa base :

| lacher | instant | naissance de l'objet | distance au socle | verdict AVANT | portage publie |
|---|---|---|---|---|---|
| 1er | 662 683 ms | `(-6,376 ; -19,079)` | **1,233 m** | « le drapeau rentre » -> ecartee | **58,1 s** (frames 6 590 -> 7 169) |
| 2e | 720 883 ms | `(-6,646 ; -18,976)` | **1,513 m** | lacher -> retenue | **0,4 s** |

**Deux naissances a 0,28 m l'une de l'autre, et 0,013 m de part et d'autre du seuil.** Le porteur
est a **0,152 m** de la premiere naissance a cette image (piste publiee, frame 6 593) : rien ne
distingue physiquement les deux lachers, seul le seuil les separe.

Et ce lacher-la est bien le sien, sans ambiguite : l'autre porteur ouvert a cet instant
(`2533274858283686`, portage 56, drapeau 1) est a **46,0 m** de la naissance — trente fois la
distance du lacher. La condition de distance au porteur, deja dans le code, nomme le laisseur a
elle seule.

### 3.5 Les trois chaines de datation, revisitees

Le rapport 6.11 §6.2 concluait que les trois chaines etaient muettes sur ce span. **Deux
l'etaient a juste titre, la troisieme etait BAILLONNEE :**

| chaine | sur le span 662 404 -> 720 462 | pourquoi |
|---|---|---|
| prise d'un AUTRE joueur (`closeByHandoff`) | muette | personne d'autre ne prend ce drapeau — **juste** |
| drapeau qui RENTRE (`closeByHomecoming`) | muette | le drapeau ne rentre pas — **juste** |
| vie libre de l'objet (`closeByFreeLives`) | muette | **l'objet NAIT a 1,233 m du porteur et la regle le refuse comme « rentree »** — **FAUX** |
| marqueur d'image-cle (controle) | non consomme | reste le CONTROLE INDEPENDANT, decision 6.11 §6.3 — **non rediscutee, non utilisee** |

L'hypothese « reprise au sol par lui-meme que rien ne date » du rapport 6.11 est donc **REFUTEE
par la mesure** : il ne reprend pas au sol pendant 58 s, il a LACHE a 279 ms de sa prise, et la
chaine qui devait le dater l'a refuse.

---

## 4. Item 2 — classe ou cas unique : une CLASSE, et elle se chiffre

Population de la zone aveugle sur les **12 films de CTF a calque du parc** (sortie
`vague6_613_zone_aveugle.tsv`, mesuree sur `parcH`) :

| film | carte | vies libres | nees AU POINT (max) | **zone aveugle** (min) | portages touches | secondes fantomes |
|---|---|---|---|---|---|---|
| `b8a44fe8` | Forest | 74 | 13 (0,006 m) | **7** (0,614 m) | 6 | 194,7 |
| `bc60b4d9` | Illusion | 41 | 13 (0,005 m) | **6** (0,324 m) | 6 | 4,7 |
| `cde26226` | Critical Dewpoint | 90 | 15 (0,005 m) | **5** (0,527 m) | 1 | 70,1 |
| `8bc6074f` | Origin | 63 | 12 (0,004 m) | **3** (0,338 m) | 0 | 0,0 |
| `bf5ced1b` | Illusion | 13 | 4 (0,005 m) | **3** (0,572 m) | 2 | 2,3 |
| `fb1a1a72` | — | 47 | 14 (0,004 m) | **3** (0,627 m) | 2 | 4,0 |
| `7fce3219` | Takamanohara | 98 | 18 (0,006 m) | **2** (0,744 m) | 0 | 0,0 |
| `a0c36016` | Forest | 66 | 18 (0,006 m) | **2** (0,856 m) | 0 | 0,0 |
| `16ea3668` | Aquarius | 37 | 12 (0,005 m) | **1** (1,258 m) | 0 | 0,0 |
| `58864b3c` | Domicile | 24 | 7 (0,004 m) | **1** (0,917 m) | 0 | 0,0 |
| `f8efc5ca` | Absolution | 46 | 14 (0,008 m) | **1** (1,170 m) | 1 | 1,4 |
| `4ecdf3e7` | High Ground (neutre) | 27 | 5 (0,004 m) | **0** | 0 | 0,0 |
| **TOTAL** | **5 cartes touchees** | **626** | **145 (0,008 m)** | **34 (0,324 m)** | **18** | **277,1** |

**Ce que la table etablit :**

- **La classe est une SITUATION DE JEU, pas une carte.** Elle frappe **10 films sur 12** et
  **5 cartes** — Forest, Illusion, Critical Dewpoint, Origin, Absolution — toutes AU catalogue,
  toutes a 2 ou 3 socles. Aucune carte n'y echappe par construction, et aucune n'est hors
  catalogue.
- **Elle n'a rien a voir avec la variante.** Le seul film a DRAPEAU NEUTRE (`4ecdf3e7`) est aussi
  le seul a zero naissance en zone aveugle — son socle unique est au centre, loin des livraisons.
- **`b8a44fe8` n'est pas un cas unique : c'est le cas EXTREME.** Sept naissances (contre 1 a 6
  ailleurs), toutes au MEME socle, et surtout des tails LONGS parce que ce match est un siege :
  0-1 en 754 s, une seule capture, l'equipe 0 campant son point de livraison le drapeau adverse
  a la main. 194,7 s des 277,1 s de la classe y sont.
- **Sur un film NEUF de la meme carte, le defaut se rejouerait a l'identique** : il ne depend ni
  du film, ni du catalogue, ni du decodage — seulement de la distance a laquelle un porteur
  lache. `a0c36016`, MEME carte, n'a que 2 naissances en zone aveugle parce que ses joueurs
  lachent ailleurs (0 seconde fantome) : c'est le jeu qui varie, pas le code.
- **Les deux populations ne se touchent nulle part** : 145 rentrees a 0,008 m au plus, 34 lachers
  a 0,324 m au moins, sur les 12 films. Les 34 lachers portent 16 a 157 echantillons de position
  (l'objet roule) ; **141 des 145 rentrees n'en portent qu'UN** — le moteur le pose et il ne
  bouge plus.

---

## 5. Item 3 — le correctif

### 5.1 UN seuil nomme, UN seul predicat

Le test « cette naissance est-elle celle du support ? » etait ecrit DEUX FOIS
(`flagFreeNearSpawn` et `flagSpawnAt`), avec la meme mauvaise constante. Le correctif le ramene a
un seul endroit :

- **`flagHomeExactDist = 0,10 m`** (`flag_objects.go`) — la distance sous laquelle une naissance
  EST le point du catalogue. Un decimetre : douze fois au-dessus de la plus grande rentree
  mesuree (0,008 m), trois fois au-dessous du plus petit lacher mesure (0,324 m). N'importe
  quelle valeur de `]0,008 ; 0,324[` rend le MEME classement sur tout le parc — meme regime que
  `originDropWindowUS`.
- **`flagSpawnAt`** porte desormais ce seuil, et il est **le seul endroit qui tranche**.
- **`flagFreeAtSpawn`** (ex-`flagFreeNearSpawn`) lui delegue : une seule ecriture, donc pas de
  divergence au prochain correctif — c'est exactement le defaut que ce lot repare.
- **`flag_neutral.go`** : le commentaire qui affirmait « c'est `originDropMaxDist` » est corrige
  (anti-patron n° 9). Le verdict de variante ne bouge sur AUCUN film (cf. §6.3).

`originDropMaxDist` reste ou il est sa reponse : la distance au PORTEUR (`flagFreeDropInside`,
`flagFreeAtDrop`) et la distance entre deux objets au repos (`flagOtherDroppedAt`).

### 5.2 Tests rouges -> verts, et leur mutation

| test | ce qu'il fige | mutation qui le rougit |
|---|---|---|
| `TestUnLacherAuPiedDuSocleFermeLePortage` | un lacher a **1,2 m** d'un support FERME le portage (le cas de `b8a44fe8`) | rendre `originDropMaxDist` a `flagSpawnAt` -> le portage se rouvre en `carried_open` |
| `TestUneRentreeResteUneRentreeAuPointDuSocle` | les DEUX bornes de l'intervalle vide : 0,03 m est le support, 0,32 m ne l'est plus | idem -> la seconde assertion tombe |
| `TestUneVieLibreNeeAUnSocleNeFermeRien` (existant) | **TEMOIN NEGATIF** : une naissance AU POINT reste ecartee, meme a 0,5 m du porteur | inchange — il reste vert avant comme apres, c'est ce qui prouve que le refus n'a pas ete supprime mais **resserre** |

Mutation jouee reellement : avec `originDropMaxDist` rendu a `flagSpawnAt`, les deux tests neufs
ECHOUENT et le temoin negatif reste VERT.

---

## 6. Item 4 — le gate

### 6.1 Les criteres, un par un

| critere | attendu | AVANT | APRES | verdict |
|---|---|---|---|---|
| ratio `b8a44fe8` | <= 1,05 | **1,234** | **1,037** | `[x]` |
| ratio des films CTF (12) | <= 1,03 | **1,066** | **1,028** | `[x]` |
| les quatre joueurs de `b8a44fe8` a +- 1,5 s | 4 / 4 | — | **3 / 4** (+0,2 · +1,2 · +1,7) | `[!]` §6.4 |
| aucun joueur au-dessus a 0,5 s pres | 0 | 5 / 7 | 7 / 7 | `[!]` §6.4 |
| les 10 autres films CTF ne bougent pas | 0 delta ou explique | — | **6 bougent, tous expliques ligne par ligne** (§6.3) | `[~]` |
| aucune action ne bouge | 0 | — | **0** (film x action x joueur, diff vide) | `[x]` |
| films non CTF identiques a l'octet | oui | — | **57 / 57** | `[x]` |
| aucun joueur ne passe sous 0,8 de son oracle | 0 nouveau | 1 | **1, LE MEME** (`4ecdf3e7`) | `[x]` |

### 6.2 `b8a44fe8`, joueur par joueur

| joueur | equipe | periodes av -> ap | publie av -> ap | oracle | ecart av -> ap |
|---|---|---|---|---|---|
| `2535442462807197` | 0 | 13 -> 16 | 140,4 -> **60,7** | 59,0 | **+81,4 -> +1,7** |
| `2533274858911298` | 0 | 2 -> 2 | 56,4 -> **28,6** | 28,4 | **+28,0 -> +0,2** |
| `2535455799302553` | 0 | 10 -> 10 | 30,9 -> **16,1** | 14,9 | **+16,0 -> +1,2** |
| `2533274823110022` | 1 | 2 -> 2 | 25,0 -> 25,0 | 17,2 | +7,8 -> **+7,8** (§6.4) |
| **`2533274858283686`** | 1 | 30 -> 30 | 232,4 -> **275,3** | 271,8 | **-39,4 -> +3,5** |
| `2533274897257135` | 0 | 2 -> 2 | 9,2 -> 9,2 | 9,0 | +0,2 -> +0,2 |
| `2535469190789936` | 1 | 1 -> 1 | 0,8 -> 0,9 | 0,8 | +0,0 -> +0,1 |

**Le cinquieme joueur est la seconde moitie de la MEME cause, et elle va dans l'autre sens.**
`2533274858283686` etait **39,4 s SOUS** son oracle : ses portages etaient TRONQUES par de
FAUSSES rentrees — des lachers nes a 1,0-1,5 m d'un support que `flagObjectHomecomings` nommait
« le drapeau est rentre », ce qui fermait son portage et le publiait `home` alors que le drapeau
gisait au sol. Le meme resserrement de seuil lui rend ses 42,9 s. Un seul seuil, deux erreurs de
signe oppose, corrigees ensemble.

**Sur tout le parc** : somme des ecarts absolus **537,3 s -> 366,9 s** (-31,7 %) ; joueurs hors
de +- 1,5 s **36 -> 32**.

### 6.3 Les six autres films qui bougent — chaque delta explique

Aucun n'a d'action qui change ; aucun ne perd de periode.

| film | ratio av -> ap | joueurs touches | ce qui bouge |
|---|---|---|---|
| `bc60b4d9` | 1,070 -> **1,030** | 4 | `closedByObject` 18 -> 24, `dropsRepositioned` 23 -> 28, `dropsWithheld` 2 -> 0. Illusion declare TROIS socles (decouverte D1 de 6.11, dont le socle CENTRAL etiquete equipe 0) : c'est ce socle central qui creait la plus grande zone aveugle du parc, en plein milieu de carte. |
| `bf5ced1b` | 1,094 -> **1,024** | 2 | idem Illusion : `closedByObject` 6 -> 9, `ownFlagRefused` 1 -> 0, `dropsWithheld` 1 -> 0 |
| `fb1a1a72` | 1,063 -> **1,047** | 2 | `closedByObject` 27 -> 30, `closedByHome` 3 -> 1 (deux fausses rentrees retirees) |
| `f8efc5ca` | 1,020 -> **1,013** | 1 | `closedByObject` 24 -> 25, `dropsWithheld` 1 -> 0 |
| `7fce3219` | 1,055 -> 1,055 | 1 (+0,1 s) | `closedByHome` 2 -> 1 : une fausse rentree devient un lacher, date a la MEME image a une frame pres |
| `a0c36016` | 1,026 -> 1,026 | 1 (+0,1 s) | idem, `closedByHome` 2 -> 1 |
| `16ea3668` | 1,031 -> 1,031 | 0 | **le calque est identique** ; seuls `closedByObject` 20 -> 21 et `teamBirths` 13 -> 12 bougent — une fermeture qui trouve desormais sa cause propre au meme instant |
| `4ecdf3e7` | 0,893 -> 0,893 | 0 | **le calque est identique** ; `teamBirths` 1 -> 0. Le verdict de VARIANTE NEUTRE est preserve (`neutralBirths = 5` inchange, toutes AU POINT du socle) |
| `58864b3c` / `8bc6074f` / `cde26226` | inchanges | 0 | compteurs seulement (`closedByObject` +1 a +5, `ownFlagRefused` -> 0) : les fermetures trouvent leur cause propre sans deplacer aucune borne |

**Le sens de TOUS les compteurs est le meme sur les douze films** : `teamBirths` baisse (moins de
naissances comptees « au support »), `closedByObject` monte (la chaine du lacher recupere sa
population), `dropsRepositioned` monte, `closedByHome` / `homeByObject` baissent (fausses
rentrees retirees), `dropsWithheld` tombe a **0 partout** (l'invariant de coherence n'a plus de
portages qui se chevauchent a proteger), `ownFlagRefused` et `assignedByPlay` baissent
(l'attribution geometrique n'a plus besoin de ses replis).

Sur `b8a44fe8` : `markerObserved` 15 -> 12 et `markerConfirmed` 15 -> 12 — **le controle
independant reste a 100 % de confirmation**, sur une population qui a retreci parce que les
portages fantomes ont disparu. Les images muettes du porteur, elles, restent a **258 exactement**
(5,2 % -> 6,2 % du fait du seul denominateur, qui tombe de 4 951 a 4 158 images portees) : elles
sont une population INDEPENDANTE de cette cause, et ce lot ne les touche pas.

### 6.4 Les deux sous-criteres `[!]`, et leur cause exacte

**(a) `2533274823110022` a +7,8 s — AUTRE CAUSE, hors perimetre.**
Ses deux portages sont `[337 528 ; 343 783]` (6,3 s, ferme par l'objet — correct) et
`[742 017 ; fin]`, publie **`carried_open` sur 187 images = 18,7 s**. C'est le **SEUL span
`carried_open` de tout le parc**, deja nomme par l'audit du 2026-09-10 (§3.4, « 18,7 s sur
`b8a44fe8` »). Il prend le drapeau a 742,0 s d'un match de 754 s et **le match se termine avec le
drapeau dans sa main** : aucun fait ne ferme le portage, la borne par defaut est la fin de l'AXE
du rejeu (frame 7 572, soit 757,2 s), pas la fin du match.
**Chemin exact du correctif** : `flagMatchEnd` (`flag_carries.go:433`) rend
`max(dernier fait, derniere image) + 1`. Il faudrait lui donner la fin de la partie JOUABLE
(`playable_duration_seconds` = 754 s aux faits de match, deja charges par `--facts`), ce qui
suppose de faire descendre cette duree jusqu'au calque — un champ neuf dans l'entree du calque,
donc hors du perimetre d'un lot dont la cause est le rayon des socles.
**Effectif : 1 span, 1 joueur, 1 film, 7,8 s** sur les 69 du parc.

**(b) « aucun joueur au-dessus a 0,5 s pres » n'est atteint par AUCUN film du parc**, ni avant ni
apres, y compris ceux a 1,013 : le grain de la mesure est la demi-image (0,05 s) multipliee par
le nombre de periodes, et une periode de portage publiee couvre `[t0, t1]` bornes INCLUSES. Un
joueur a 30 periodes accumule donc structurellement quelques dixiemes au-dessus. Le critere
n'est pas atteignable sans changer la DEFINITION du portage (decision produit, audit §3.4), et
il n'a pas ete contourne : il est mesure et rendu tel quel.

---

## 7. Films a recuire — la liste exacte

**12 des 69 films changent**, et ce sont EXACTEMENT les 12 films de CTF a calque. Les **57
autres sont identiques a l'octet**.

```
16ea3668  4ecdf3e7  58864b3c  7fce3219  8bc6074f  a0c36016
b8a44fe8  bc60b4d9  bf5ced1b  cde26226  f8efc5ca  fb1a1a72
```

**Aucune montee de `SchemaVersion`** : aucun champ n'est cree ni supprime, aucune cle ne bouge.
Seules changent des VALEURS de `flagCarries` et de `coverage.flagCarries`. Un artefact non recuit
n'est pas illisible, il est seulement trop long sur ses portages.

**Web** : non touche. Aucun champ neuf, donc ni `openapi.yaml` ni `generated.ts` ne bougent.

---

## 8. Gates techniques, joues reellement

| commande | resultat |
|---|---|
| `CGO_ENABLED=0 go test ./internal/analysis/replay/ ./internal/analysis/objectiveevents/ ./internal/replaybuild/` | **ok**, 3 paquets |
| `CGO_ENABLED=1 go test ./internal/service/...` | **ok**, 4 paquets |
| `golangci-lint run ./internal/analysis/replay/...` | **0 issues** |
| `go vet` (3 paquets) | propre |
| `filmdec` | **non touche** — aucun desassemblage necessaire, la cause est en amont du decodage |

**Goldens** : aucun golden de CI ne change (les goldens de rejeu portent sur `000d5950`, un
Slayer sans calque de drapeau). Aucune reference de test existante n'a ete modifiee ; le temoin
negatif historique (`TestUneVieLibreNeeAUnSocleNeFermeRien`) est conserve tel quel et reste vert.

**Corpus d'equivalence** : non joue ici, il appartient au superviseur (instruction du lot).

---

## 9. Reproduction et instruments

```bash
# 1. cuisson HORS LIGNE d'un parc (aucune base ouverte, chunks lus en lecture seule)
LEVELUP_REPO_ROOT=<racine de travail> \
  go run ./apps/go-api/cmd/replay-build --map <carte> --facts <short8>.facts.json <matchId> \
  <racine partagee>/data/cache/film_chunks/<short8>

# 2. confrontation a l'oracle API
cd .ai/V7.5/outillage/couverture_objectifs
CGO_ENABLED=0 go run . -parc <parc cuit> \
  -oracle <racine>/.ai/V7.5/replay2d/registre_film -out <sortie>
```

Les sorties `vague6_613_*.tsv|log` committees sont celles de cette commande sur `parcC`
(69 artefacts), plus deux tables propres au lot :

| sortie | contenu |
|---|---|
| `vague6_613_zone_aveugle.tsv` | la population des naissances d'objet drapeau par film : nees au point du catalogue, nees en zone aveugle, portages touches, secondes fantomes |
| `vague6_613_joueurs_avant_apres.tsv` | le portage publie AVANT et APRES, par film et par joueur, avec l'oracle |

**Instrument de recherche, supprime avant livraison** (recette pour le rejouer) :
`internal/analysis/replay/zz613_instrument.go`, ~130 lignes, sous garde `LOT613_DUMP=<fichier>`,
appele depuis `buildFlagCarries` a cinq points (apres le bornage, apres chaque chaine de
fermeture, et au final). Il ecrit, par film : les socles retenus, la table equipe par joueur,
les 74 vies libres avec leur distance au socle le plus proche et leur nombre d'echantillons, les
portages a CHAQUE etage de fermeture, et — la ligne qui a nomme la cause — pour chaque portage,
la RAISON de refus de chaque vie libre tombant dans sa fenetre (`SOCLE` / `LOIN_DU_PORTEUR` /
`SANS_POSITION` / `RETENUE`) avec la distance au porteur. **Controle** : les artefacts produits
par le binaire instrumente et par le binaire final sont identiques a l'octet.

---

## 10. Decouvertes, consignees et NON traitees

- **D1 — le SEUL `carried_open` du parc n'est pas ferme par la fin du MATCH.**
  `flagMatchEnd` (`replay/flag_carries.go:433`) borne a la fin de l'AXE du rejeu, qui depasse la
  duree jouable. Effectif : 1 span, `b8a44fe8` / `2533274823110022`, **7,8 s** — et c'est tout ce
  qui separe ce film de 1,018. Chemin : faire descendre `playable_duration_seconds` (deja aux
  faits de match) jusqu'a l'entree du calque.
- **D2 — Illusion declare TROIS `flag_spawn`, dont un CENTRAL etiquete equipe 0**
  (`(0 ; 0)`, catalogue `map_objectives.json`). C'est la decouverte D1 du lot 6.11, et ce lot en
  mesure le second effet : ce socle fantome creait une zone aveugle **au milieu de la carte**,
  la ou le drapeau tombe le plus souvent — 6 naissances sur `bc60b4d9`, 3 sur `bf5ced1b`. Le
  resserrement du seuil en annule l'essentiel, mais le socle reste faux au catalogue. **A
  verifier au catalogue, pas dans `replay/`.**
- **D3 — `cde26226` garde 70,1 s de tail candidates et ne bouge pas d'une seconde.** Ses cinq
  naissances en zone aveugle sont toutes au socle 0, mais le porteur n'est jamais a moins de
  1,5 m au moment ou elles naissent : la condition de distance au porteur les refuse, et c'est
  la bonne reponse. Son ratio de 1,040 a une AUTRE cause, non instruite ici.
- **D4 — les images muettes du porteur ne doivent RIEN aux portages fantomes, et la mesure le
  tranche dans l'autre sens que celui attendu.** Sur `b8a44fe8` leur compte est **rigoureusement
  le meme avant et apres : 258**, alors que 793 images de portage ont disparu. Le pourcentage de
  l'axe 5 de l'audit du 2026-09-10 MONTE donc (5,2 % -> 6,2 %) sans qu'aucune image muette ne
  s'ajoute : c'est un taux dont seul le denominateur bougeait. La population est reelle,
  independante, et reste a instruire.
- **D5 — le controle du marqueur d'image-cle n'a PAS ete consomme**, et la decision 6.11 §6.3
  n'a pas ete rediscutee. Il reste la seule chaine disjointe de celle des compteurs, et il donne
  desormais 12 / 12 sur `b8a44fe8`.
