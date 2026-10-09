# V3E — Reconstruction COMPLETE : l'evenement tel que le jeu le joue

> Worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`. Lecture seule sur les
> donnees du jeu. AUCUN commit, AUCUN `git add`. Code ajoute : uniquement sous
> `apps/go-api/cmd/weapon-sounds/` (8 fichiers, additifs, `gofmt` / `go vet` / `go test` /
> `go build` verts, tous sous 500 lignes). WAV livres = transitoires non commites.
>
> **Mandat**, reproche utilisateur verbatim : « comme d'habitude pour la millieme fois, tu
> n'as rendu que les fichiers isoles, pas reconstitues avec leur format wwise et reglages de
> package ». Ce lot rend l'EVENEMENT, pas la piece.

## 1. Verdict en une page

Le reproche est **fonde, et il est maintenant mesure**. Les lots V3B / V3C / V3D sommaient
bien des couches, mais avec quatre reglages du paquet Wwise **jamais lus** :

| Reglage | Etat avant V3E | Mesure V3E |
|---|---|---|
| Gain des PARENTS actor-mixer (`DirectParentID`) | jamais somme | jusqu'a **-14,5 dB** de chemin amont (Ghost), **+7 dB** (lointaine Wraith) |
| `MakeUpGain` (AkPropID **6**) | jamais lu | 15 occurrences sur les 12 banques ouvertes |
| Offset de couche (`InitialDelay`, AkPropID **59**) | lu au **mauvais identifiant** (17) | 25 occurrences : 0,05 a 0,25 s reellement appliques |
| Mode de boucle du conteneur (`AkTransitionMode` **5**, CADENCE) | ignore | Ghost : un grain redeclenche **toutes les 0,130 s**, avec chevauchement |

Consequences directes, toutes corrigees dans les fichiers livres :

- le **lit sub** de l'explosion etait a **-2 dB** (Mongoose, Ghost) ou **+3 dB** (Warthog,
  Wasp, Chopper) de son niveau reel — son ActorMixer parent porte un gain non nul ;
- l'explosion du **Ghost** pose ses trois couches proches **100 ms APRES** le lit sub ;
  rev10 les empilait toutes a t = 0 ;
- l'explosion lointaine du **Warthog / Wasp** est jouee **+380 cents plus haut** (3,97 s ->
  3,19 s) ; rev10 la rendait a hauteur nominale ;
- le **moteur du Ghost** n'est pas une boucle, c'est un **nuage de grains** : 24 medias de
  0,12 a 0,68 s redeclenches toutes les 0,130 s, quatre a cinq superposes en permanence. En
  repeter un seul pendant 8 s donne un hachage periodique, pas un souffle.

Et la piste rouverte par V3D est fermee : **Ghost, Wraith, Banshee, Chopper** sont refaits
depuis **leurs** banques (`pc/globals/common`), pas depuis celle du Warthog.

## 2. La table `AkPropID`, mesuree et non postulee

`proprietes.go` ne lit que quatre proprietes et nomme la **17** « InitialDelay ». Le releve
des identifiants reellement presents dans les 12 banques ouvertes tranche : **la 17
n'apparait sur aucun noeud** ; c'est la **59** qui porte le delai, et ses valeurs (0,05 a
0,25 s) sont exactement celles d'un offset de couche. La table des versions de bank 113+ est
donc la bonne, et le « zero delai partout » des lots precedents etait un **artefact de
lecture**, pas une propriete des donnees.

Releve, module `pc/globals` (5 banques) puis `pc/globals/common` (7 banques) :

| id | nom | occurrences UNSC | occurrences covenant |
|---|---|---|---|
| 7 | Priority | 177 | 738 |
| 8 | PriorityDistanceOffset | 135 | 675 |
| 19 | UserAuxSendVolume0 | 90 | 387 |
| 0 | Volume | 87 | 214 |
| 70 | AttenuationID | 73 | 115 |
| 23 | GameAuxSendVolume | 67 | 232 |
| 2 | Pitch (RANGED) | 4 | 30 |
| 59 | **InitialDelay** | 18 | 7 |
| 4 | HPF | 4 | 6 |
| 6 | **MakeUpGain** | 11 | 4 |
| 0 | Volume (RANGED) | 0 | 6 |
| 13 | PAN_FR | 2 | 4 |
| 58 | Loop | 2 | 0 |
| 2 | Pitch | 2 | 1 |
| **71** | **NON DECODEE** | 2 | 2 |

**Propriete non decodee, octets bruts** (regle du mandat : on le dit, on ne le tait pas) :

```
sbnk 969f4dc6 noeud 37a7fbe6 (ActorMixer) prop 71 : octets [00 00 5c 42] = 55 en flottant, 1113325568 en u32
sbnk c468fb55 noeud 1d1f0aa9 (ActorMixer) prop 71 : octets [00 00 5c 42] = 55 en flottant, 1113325568 en u32
sbnk 3fdc61a7 noeud 1d1f0aa9 (ActorMixer) prop 71 : octets [00 00 5c 42] = 55 en flottant, 1113325568 en u32
sbnk b1f8608b noeud 37a7fbe6 (ActorMixer) prop 71 : octets [00 00 5c 42] = 55 en flottant, 1113325568 en u32
```

Toujours la meme valeur (55,0), toujours sur un ActorMixer de la chaine **lointaine** (bus
`ff2ef7d6`). Elle n'est sur le chemin d'AUCUN fichier livre : elle ne change rien aux rendus,
mais elle existe et elle n'est pas nommee.

**Deux autres reglages sont LUS mais NON APPLIQUES, et c'est un choix declare :**

- `HPF` (AkPropID 4) : valeurs 10, 16, 20 et 35 sur quatre a six noeuds — dont la **couche 2
  de chaque explosion** (`36c9b286` du Scorpion, HPF = 10, octets `[00 00 20 41]`). Wwise
  exprime ce filtre sur une echelle 0..100 dont la correspondance en frequence n'est **pas
  dans la banque**. L'appliquer demanderait de deviner la courbe : on ne le fait pas, on le
  consigne.
- `UserAuxSendVolume0` (-12 dB partout) et `GameAuxSendVolume` (-30 a +6 dB) : ce sont des
  **envois vers les bus auxiliaires** (reverberation), pas le trajet direct. Un rendu hors
  jeu n'a pas de reverberation de niveau : ils ne s'appliquent pas.

## 3. Le gain de chemin COMPLET

**Definition retenue** : somme de `Volume` + `MakeUpGain` sur la chaine
`Sound -> conteneurs -> parents actor-mixer (DirectParentID) -> bus`.

**Ce qui manquait** : `arbre.go` ne somme que la partie DESCENDANTE (evenement ->
conteneurs -> Sound). Tant que toutes les couches d'un evenement ont le meme parent, les deux
coincident. Des qu'elles n'ont pas le meme — c'est le cas du **lit sub**, pose par une action
distincte donc par un ActorMixer distinct — le melange est faux.

**Les bus ne sont PAS resolus, et c'est mesure** : **zero** objet `Bus` / `AuxBus` dans les
12 banques ouvertes. La hierarchie de bus vit dans une banque d'initialisation qui n'est pas
un tag `sbnk`. Les huit bus rencontres, avec leur role deduit de qui les emprunte :

| bus | ce qui y passe |
|---|---|
| `a47c42cf` | explosion, couches principales et lit sub |
| `395ebec8` | explosion, **une** couche par banque routee a part |
| `ff2ef7d6` | explosion lointaine (relais `stai`) |
| `5a880943` | moteur, couche de regime (le `Switch`) |
| `1f17314c` | moteur, lit sub |
| `8165b6c5` | contact avec le sol (chenilles, roues, antigrav) |
| `0f233096` | les cinq paliers de regime joues **hors** switch (§5.3) |
| `fd3d66ca` | Wraith seul, evenement `e0302bdf` |

**Portee exacte de la reserve** : a l'interieur d'un meme bus les gains sont exacts au
dixieme de dB ; entre deux bus il manque un decalage constant inconnu. C'est une reserve
beaucoup plus etroite que le « calibre par cible de RMS » de rev9.

## 4. MISSION 1 — les destructions, refaites

### 4.1 Structure HIRC de l'evenement

Les six banques ont la meme forme : **2 actions `Play`** (une pour le trio de couches
proches groupees sous un `Blend`, une pour le lit sub), **4 couches simultanees**, conteneurs
`RandomSequence` en mode aleatoire **non continu, une seule lecture**
(`sequence=false continu=false`) — le one-shot de V3D est confirme sur pieces.

Dump integral, `sb_008_exp_vehicle_large_unsc` (Scorpion), evenement `dcc7bca1` :

```
action 39a28030  Play (0x0403) -> 05aa04fe  delai=0.000s
action 3e33f3fb  Play (0x0403) -> 35278286  delai=0.000s

couche 1  20d47cce RandomSequence  amont=+0.00 dB  bus=a47c42cf HORS BANQUE
  action 39a28030(+0.000s) <- 0aa8e7f7(ActorMixer,+0.0dB) <- 1b645ed9(ActorMixer,+0.0dB)
    | bus a47c42cf -> 05aa04fe(Blend,+0.0dB) -> 20d47cce(RandomSequence,+17.0dB) -> Sound(+0.0dB)
  6 variantes, gain de chemin +17,00 dB, offset 0,000 s, pitch +0
couche 2  36c9b286 RandomSequence  amont=+0.00 dB  bus=395ebec8 HORS BANQUE
  ... -> 36c9b286(RandomSequence,+20.0dB) -> Sound(+0.0dB)      6 variantes, +20,00 dB
couche 3  3c95cab9 RandomSequence  amont=+0.00 dB  bus=a47c42cf HORS BANQUE
  ... -> 3c95cab9(RandomSequence,+10.0dB) -> Sound(+0.0dB)      6 variantes, +10,00 dB
couche 4  35278286 RandomSequence  amont=+0.00 dB  bus=a47c42cf HORS BANQUE
  action 3e33f3fb(+0.000s) <- 3d0ee8f0(ActorMixer,+0.0dB) <- 1b645ed9(ActorMixer,+0.0dB)
    | bus a47c42cf -> 35278286(RandomSequence,+8.0dB) -> 37355e5a(Sound,-6.0dB)
  1 variante (686882988), gain de chemin +2,00 dB   <- 8 - 6 : le SOUND porte -6 dB

noeud 20d47cce :
  prop 0  Volume                 = 17    octets [00 00 88 41]
  prop 7  Priority               = 70    octets [00 00 8c 42]
  prop 8  PriorityDistanceOffset = 0     octets [00 00 00 00]
  prop 19 UserAuxSendVolume0     = -12   octets [00 00 40 c1]
  prop 70 AttenuationID          = u32 245331775   octets [3f a5 e3 0e]
noeud 36c9b286 :
  prop 0  Volume = 20  |  prop 4 HPF = 10  |  prop 23 GameAuxSendVolume = +6  |  bus propre 395ebec8
noeud 35278286 :
  prop 0  Volume = 8   |  prop 19 UserAuxSendVolume0 = -12  |  prop 23 GameAuxSendVolume = -21
noeud 37355e5a (Sound du lit sub) :
  prop 0  Volume = -6   octets [00 00 c0 c0]  |  prop 7 Priority = 100
noeud 05aa04fe (Blend) :
  prop 7 Priority = 90  |  prop 8 PriorityDistanceOffset = -30  |  prop 19 UserAuxSendVolume0 = -12
```

### 4.2 Ce qui CHANGE, gain par gain

| banque | L1 | L2 | L3 | lit sub rev10 | **lit sub rev11** | ecart | offsets rev11 |
|---|---|---|---|---|---|---|---|
| `969f4dc6` small_unsc | +7 | +10 | +15 | -8 | **-10** (amont **-2**, `1f009c9e`) | **-2 dB** | 0 |
| `c468fb55` med_unsc | +10 | +15 | +13 | 0 | **+3** (amont **+3**, `33a0f093`) | **+3 dB** | 0 |
| `94d43e95` large_unsc | +17 | +20 | +10 | +2 | +2 | 0 | 0 |
| `b1f8608b` small_cv | +17 | +14 | +12 | -8 | **-10** (amont **-2**) | **-2 dB** | **L1/L2/L3 a +0,100 s** |
| `3fdc61a7` med_cv | +12 | +9 | +8 | 0 | **+3** (amont **+3**) | **+3 dB** | 0 |
| `2eaae6d7` large_cv | +12 | +8 | +5 | +2 | +2 | 0 | 0 |

Etats lointains (`stai`) :

| banque | etats | gain rev10 | **gain rev11** | offset rev11 | **pitch rev11** |
|---|---|---|---|---|---|
| `969f4dc6` | `16c57452` / `daaf2744` | 6 / 9 | 6 / 9 | **0,050 / 0,150 s** | 0 |
| `c468fb55` | `50f7b5c5` / `29082103` | 2 / 8 | 2 / 8 | **0,250 / 0,150 s** | **+380 / 0 cents** |
| `94d43e95` | `7b1d99fe` / `651b5570` | 6 / 9 | 6 / 9 | **0,200 / 0,150 s** | 0 |
| `b1f8608b` | `ab214f3f` / `65fe08f9` | 2 / 6 | 2 / 6 | 0 / 0 | 0 |
| `3fdc61a7` | `7d52bdd4` / `3a818492` | 2 / 6 | 2 / 6 | **0,200 / 0,150 s** | 0 |
| `2eaae6d7` | `df41ac8b` / `f97fc935` | 3 / 6 | **10 / 13** (amont **+7**) | **0,150 s** | 0 |

### 4.3 Mesures, rev10 contre rev11

`explosion.wav`, meme tirage (variante d'indice 0 sur chaque couche) pour que la comparaison
isole le changement de gains. rev10 est reconstruit a l'identique depuis son melange brut.

| vehicule | rev | RMS | bas < 200 Hz | haut 3-8 kHz | RMS moins haut |
|---|---|---|---|---|---|
| Gungoose | rev10 | -17,61 | -18,56 | -35,13 | 17,52 |
| Gungoose | **rev11** | **-19,71** | **-20,66** | **-37,64** | **17,93** |
| Warthog / Wasp | rev10 | -21,64 | -22,93 | -37,48 | 15,84 |
| Warthog / Wasp | **rev11** | **-22,37** | **-23,52** | **-39,13** | **16,76** |
| Scorpion | rev10 | -18,47 | -19,37 | -35,07 | 16,60 |
| Scorpion | **rev11** | **-20,01** | **-20,86** | **-36,87** | **16,86** |
| Ghost | rev10 | -19,55 | -20,94 | -34,90 | 15,35 |
| Ghost | **rev11** | **-20,35** | **-21,78** | **-37,92** | **17,57** |
| Chopper | rev10 | -19,60 | -20,73 | -35,82 | 16,22 |
| Chopper | **rev11** | **-20,76** | **-22,16** | **-40,67** | **19,91** |
| Wraith / Banshee | rev10 | -18,29 | -19,48 | -32,57 | 14,28 |
| Wraith / Banshee | **rev11** | **-19,04** | **-20,48** | **-35,38** | **16,34** |

Lecture : le niveau absolu baisse de 0,7 a 2,1 dB parce que la normalisation vise desormais
le **plus fort des trois tirages** (aucun ne depasse -1 dBFS ; rev10 calibrait sur le seul
tirage 1 et laissait passer +0,13 dBFS sur le tirage 2 du Gungoose). La **forme** bouge peu
la ou le lit sub etait deja juste (Scorpion : 0,26 dB d'ecart sur `RMS moins haut`) et
nettement la ou il ne l'etait pas (Chopper 3,7 dB, Ghost 2,2 dB), plus les 100 ms qui
degagent le transitoire du Ghost.

### 4.4 Ce qui est livre

`sons_v3_reconstruits/<Vehicule>/destruction/` — **la racine ne porte que des evenements
complets** :

| fichier | contenu |
|---|---|
| `explosion.wav` | 4 couches sommees, tirage 1 (variante d'indice 0 partout, comparable a rev10) |
| `explosion_v2.wav`, `explosion_v3.wav` | tirages 2 et 3, **une variante tiree par couche, independamment** |
| `explosion_lointaine_A.wav`, `_B.wav` | les deux etats `stai`, evenement complet (1 couche), offset et hauteur appliques |
| `stems/explosion_coucheN_<noeud>.wav` | les couches isolees du tirage 1, **au meme gain de normalisation** que le melange |

Les quatre fichiers `couche_1..3.wav` et `lit_sub_partage.wav` de rev10 ont quitte la racine :
ils sont remplaces, aux gains corriges, par les `stems/` (leur contenu rev10 etait faux de
2 a 3 dB sur le lit). Huit vehicules : Gungoose, Warthog_roquettes, Wasp, Scorpion, Ghost,
Chopper, Wraith, Banshee.

## 5. MISSION 2 — les moteurs depuis les VRAIES banques

### 5.1 L'attribution corrigee, verifiee une seconde fois

Les quatre medias que rev9 servait pour le Ghost (`192653757`, `68830349`, `835658180`,
`52024965`) sont **absents du pack du Ghost (248 medias) COMME du pack du Warthog (78)** :
ils n'existent que dans la banque `01862ab3`, celle qu'atteint le `hlmt` du Warthog. Un
balayage des **841 packs** du jeu ne les trouve nulle part ailleurs. La correction de V3D §10
tient.

| vehicule | banque | module | evenement de moteur |
|---|---|---|---|
| Ghost | `ccd43fa8` | `pc/globals/common` | `603d9e29` |
| Wraith | `fda12da2` | `pc/globals/common` | `aa6215eb` |
| Banshee | `c682f736` | `pc/globals/common` | `6ed9c3bc` (+ `851558f7`, couche de grain) |
| Chopper | `1bb9f097` | `pc/globals/common` | `1adf8067` |
| Scorpion | `05a51e0a` | `pc/globals` | `951f76c0` |
| Warthog | `a52af042` | `pc/globals` | `68b1a949` |

**L'identification de l'evenement de moteur n'est plus une lecture de spectre : c'est une
mesure de structure.** Un seul evenement par banque traverse le conteneur `Switch` du groupe
**2275666646** — le groupe de regime, identique sur les six vehicules. Aucune ambiguite.

### 5.2 L'echelle de regime : 9 etats, 5 paliers, monotone

Le groupe `2275666646` porte **neuf** etats (rev9 en voyait cinq) qui pointent sur **cinq**
reservoirs distincts. Moyennes des medias de la couche de regime, RMS en dBFS / duree en s :

| palier | etat(s) | Scorpion | Warthog | Ghost | Wraith | Banshee | Chopper |
|---|---|---|---|---|---|---|---|
| 1 ralenti | `356702912`, `1975887784`, `2311227951`, `3561860439` | -21,31 / 7,42 | -20,93 / 4,32 | -15,39 / 0,82 | -15,62 / 6,84 | -16,28 / 5,32 | -19,28 / 4,29 |
| 2 bas | `1093928064` | -21,33 / 8,56 | -21,08 / 4,44 | -15,30 / 0,76 | -15,36 / 6,95 | -15,87 / 4,50 | -17,11 / 3,61 |
| 3 intermediaire | `1136871302` | -20,77 / 6,18 | -20,88 / 4,27 | -14,40 / 0,75 | -15,39 / 6,30 | -16,20 / 4,79 | -18,29 / 3,34 |
| **4 EN CONDUITE** | `1248419637`, **`3707760930`** | **-18,08 / 5,94** | **-20,90 / 4,14** | **-14,35 / 0,59** | **-15,10 / 5,41** | **-15,40 / 4,21** | **-17,62 / 2,71** |
| 5 plein regime | `163696720` | -16,87 / 3,61 | -18,15 / 3,00 | -13,91 / 0,54 | -13,62 / 4,24 | -15,26 / 3,92 | -16,07 / 1,88 |

L'echelle est **monotone sur les six vehicules** : le niveau monte et la duree de boucle
baisse quand le regime monte. Le DEFAUT declare est le palier 1 (`356702912` cote UNSC,
`3561860439` cote covenant) — le piege de V3C est confirme et evite.

**Correction de rev9** : rev9 rendait le palier 4 pour le Scorpion mais le palier **5** pour
le Warthog — deux vehicules cote a cote ne tournaient pas au meme regime. rev11 rend le
palier 4 pour tous (`moteur_conduite_boucle8s.wav`) et le palier 5 dans un fichier separe.

### 5.3 Ce que porte chaque bus (nommage par structure, pas par oreille)

- `5a880943` — la couche de regime, pilotee par le `Switch` : **le moteur proche** ;
- `1f17314c` — le lit sub du moteur, une seule variante, partage entre vehicules ;
- `8165b6c5` — un evenement a **3 couches simultanees** : le **contact avec le sol**
  (Scorpion `13beda14` / `d5c7daeb` = chenilles, centroide 4615 Hz ; Warthog `38b83eb8` =
  roues et combustion ; Ghost `a91f9f78`, Wraith `46b14f04`, Banshee `f76415db`, Chopper
  `66b341e8` = le contact antigrav) ;
- `0f233096` — **cinq evenements a une couche par banque**, plus faibles et plus sombres de
  5 a 8 dB : un par palier de regime, joues **hors** switch. Cinq evenements pour cinq
  paliers, sur les six banques : la coincidence numerique est totale. Lecture proposee (et
  declaree comme lecture, pas comme nom lu) : le moteur des AUTRES vehicules / a distance.

### 5.4 Le moteur du Ghost est un NUAGE DE GRAINS

C'est la decouverte qui change le rendu. Couche de regime du Ghost, etat `3707760930` :

```
couche 1 : 37f07dd4 RandomSequence  amont=-7.00 dB  bus=5a880943 HORS BANQUE
  action 04247f19(+0.000s) <- 10f589ad(ActorMixer,-2.0dB) <- 138d1e40(ActorMixer,-8.0dB)
    <- 2774363f(ActorMixer,+3.0dB) <- 0af1684d(ActorMixer,+0.0dB) <- 280705ab(ActorMixer,+0.0dB)
    | bus 5a880943 -> 37f07dd4(RandomSequence,+1.0dB) -> 107d73ac(Switch,+0.0dB)
    -> 150677c5(RandomSequence,+0.0dB) -> 3bd460c3(RandomSequence,-3.0dB) -> Sound(+0.0dB)
  repetitions : 0 (boucle infinie)   mode_transition = 5 (CADENCE)   duree = 0,130 s
  conteneur : sequence=true continu=true
  24 variantes, gain de chemin -6,00 a -9,00 dB selon la variante
couche 2 : 2199fb1d RandomSequence  amont=-4.00 dB  bus=1f17314c HORS BANQUE
  repetitions : 0   mode_transition = 5   duree = 0,100 s
  1 variante : wem 87187708, gain -4,00 dB, RANGED hauteur 0 .. +800 cents

le Switch 107d73ac, groupe 2275666646, defaut 3561860439 :
  etat 163696720   -> 29f66aaf  20 wems
  etat 356702912   -> 3b92b1b4  20 wems
  etat 1093928064  -> 0ffedb8b  20 wems
  etat 1136871302  -> 2ad4b24a  20 wems
  etat 1248419637  -> 150677c5  20 wems
  etat 1975887784  -> 3b92b1b4  (alias du palier 1)
  etat 2311227951  -> 3b92b1b4  (alias du palier 1)
  etat 3561860439  -> 3b92b1b4  (DEFAUT, palier 1)
  etat 3707760930  -> 150677c5  (alias du palier 4)
```

Le chemin amont vaut **-7 dB** (cinq ActorMixer traverses) : c'est exactement ce que les lots
precedents ignoraient. Et le mode **5 = CADENCE** dit qu'une lecture demarre toutes les
**0,130 s** alors que les grains durent 0,12 a 0,68 s : **quatre a cinq grains sont superposes
en permanence**. Le rendu rev11 le reproduit (`tenirLaDuree`). Cadences mesurees : Ghost
0,130 s, Banshee 0,125 s, Chopper 0,250 s. Scorpion, Warthog, Wraith et la couche principale
de la Banshee n'ont pas de boucle declaree au conteneur : leurs medias sont des corps de
boucle longs (4 a 8 s), repetes tels quels.

**Reserve honnete sur le lit sub covenant** : le media `87187708` ne fait que **0,031 s** et
n'existe dans **aucun** des 841 packs — il n'est que dans le chunk `DIDX` de la banque. Regle
de lecture retenue : un media present dans un pack a une version tronquee (prefetch) dans la
banque — mesure : `41940380` fait 0,410 s embarque contre 0,677 s dans le pack ; un media
**absent de tout pack** est complet dans la banque. `87187708` est donc un grain de 31 ms,
redeclenche toutes les 100 ms, avec une randomisation de hauteur de 0 a +800 cents. C'est une
lecture defendable, pas une certitude : si le lit sonne comme un bourdonnement periodique,
c'est cette regle-la qu'il faut refuter.

### 5.5 Verite terrain : ce n'est PAS une jeep

Mesures des fichiers livres (`deplacement/`), dBFS sauf le centroide :

| vehicule | fichier | RMS | crest | bas < 200 Hz | haut 3-8 kHz | centroide |
|---|---|---|---|---|---|---|
| Ghost | `moteur_conduite_boucle8s` | -13,77 | 11,80 | -14,14 | -36,23 | **2870 Hz** |
| Ghost | `moteur_boost_boucle8s` | -13,81 | 12,81 | -14,19 | -35,47 | **3215 Hz** |
| Ghost | `contact_antigrav_boucle8s` | -15,39 | 14,39 | -16,74 | -34,30 | 2131 Hz |
| Wraith | `moteur_conduite_boucle8s` | -18,36 | 17,36 | -18,75 | -41,97 | 1719 Hz |
| Wraith | `moteur_boost_boucle8s` | -17,19 | 16,19 | -17,62 | -41,34 | 1562 Hz |
| Wraith | `contact_antigrav_boucle8s` | -17,53 | 16,53 | -18,27 | -38,44 | 1871 Hz |
| Banshee | `moteur_conduite_boucle8s` | -17,95 | 16,59 | -19,71 | -37,99 | 1706 Hz |
| Banshee | `moteur_grain_boucle8s` | -13,69 | 12,69 | -14,91 | -30,09 | 2965 Hz |
| Chopper | `moteur_conduite_boucle8s` | -13,39 | 12,31 | -15,10 | -29,45 | 3514 Hz |
| Scorpion | `moteur_conduite_boucle8s` | -19,16 | 18,02 | -20,85 | -44,16 | 1474 Hz |
| Scorpion | `chenilles_boucle8s` | -10,69 | 9,69 | -14,85 | **-18,98** | **4237 Hz** |
| Scorpion | `moteur_amorce` | -21,35 | 20,35 | -25,62 | -36,47 | 1640 Hz |
| Warthog | `moteur_conduite_boucle8s` | -21,62 | 20,59 | -23,76 | -35,21 | 2864 Hz |
| Warthog | `roues_combustion_boucle8s` | -22,02 | 21,02 | -23,30 | -39,09 | 2697 Hz |

- **Le Ghost n'est pas une combustion** : crest 11,8 (texture continue, pas de cycle de
  detonations), centroide 2870 Hz, corps grave a -14 dB. La signature « jeep » citee par V3B
  (1472 Hz a l'epoque, 1805 Hz avec le centroide de ce lot, cycles de rev de 1,15 s)
  n'apparait sur **aucun** fichier covenant.
- **Les chenilles du Scorpion sont intactes** : centroide 4237 Hz et bande 3-8 kHz a
  -18,98 dB, 25 dB au-dessus de celle de son moteur. C'est bien le cliquetis de maillons.
- **Wraith et Banshee** sont plus graves et plus soutenus que le Ghost (centroide ~1700 Hz,
  crest 16-17) : coherent avec deux appareils lourds contre un vehicule leger.

Temoin de non-regression du centroide, pour que ces chiffres soient comparables a ceux de
V3B : le wem de chenilles `1033065922` est mesure a **4747 Hz** ici contre **4687 Hz** dans
V3B (ffmpeg `aspectralstats`), soit 1,3 % d'ecart.

### 5.6 Ce qui est livre

`sons_v3_reconstruits/<Vehicule>/deplacement/` :

| fichier | contenu | present pour |
|---|---|---|
| `moteur_conduite_boucle8s.wav` (+ `_v2`) | evenement de moteur complet, palier 4, 8 s | les 6 |
| `moteur_boost_boucle8s.wav` | le meme, palier 5 (plein regime = poussee) | Ghost, Wraith, Banshee |
| `moteur_plein_regime_boucle8s.wav` | le meme, palier 5 | Chopper, Scorpion, Warthog |
| `moteur_grain_boucle8s.wav` | 2e evenement de moteur (26 grains, cadence 0,125 s) | Banshee |
| `contact_antigrav_boucle8s.wav` | evenement de contact, 3 couches | Ghost, Wraith, Banshee |
| `contact_sol_boucle8s.wav` | idem | Chopper |
| `chenilles_boucle8s.wav` | evenement `d5c7daeb` | Scorpion |
| `roues_combustion_boucle8s.wav` | evenement `38b83eb8`, 3 couches | Warthog |
| `moteur_amorce.wav` | **vrai evenement** d'allumage `0134da4e` (2 couches) | Scorpion |
| `moteur_queue.wav` | **vrai evenement** d'extinction `7f6a7624` | Scorpion |
| `stems/<nom>_coucheN_<noeud>.wav` | couches isolees, meme gain de normalisation | tous |

**`moteur_amorce` / `moteur_queue` n'existent que pour le Scorpion** : c'est la seule banque
des six qui porte des evenements d'allumage et d'extinction sur le bus du moteur. Les cinq
autres n'en ont pas, et **aucun fichier n'a ete fabrique pour combler le trou** : un fondu
d'entree pose sur une boucle n'est pas un evenement du jeu, et le livrer sous ce nom serait
exactement le genre d'approximation que ce lot corrige.

## 6. La chaine de rendu, en clair

```
sbnk (module) --hirc-event--> plan JSON (structure + gains complets + offsets + RANGED)
pck AKPK      --pck-dump---->  wem --vgmstream--> wav
banque DIDX   --eqip-arbre -emb--> wem embarques --vgmstream--> wav  (medias absents des packs)
plan + wav    --rendu-event--> somme en float64 -> UNE normalisation a -1 dBFS -> WAV 24 bits
```

Reglages appliques par couche, dans cet ordre : hauteur (`Pitch` + centre de la fourchette
RANGED de hauteur) -> montee mono vers stereo a gain unitaire -> gain de chemin complet
(`Volume` + `MakeUpGain` de tout le chemin + centre de la fourchette RANGED de volume) ->
remplissage de la duree (cadence ou repetition) -> offset (`InitialDelay`) -> somme.

**Fourchettes RANGED** : leur valeur **centrale** est appliquee et publiee au manifeste. Que
les deux composantes soient des offsets **signes** autour du nominal est mesure, pas postule :
les couches de contact du Ghost declarent (-80, +80) et (-85, +80) cents — une paire
symetrique ne peut etre qu'un couple min/max signe. Le centre est donc nul pour la grande
majorite des couches, et non nul dans deux cas seulement : (-3, 0) dB sur le contact du Ghost
(-1,5 dB applique) et (0, +800) cents sur le lit sub covenant (+400 cents appliques).

**Aucune egalisation, aucune compression, aucun limiteur.** La seule operation apres la somme
est une multiplication scalaire. Les boucles recoivent un fondu de 10 ms a chaque extremite,
pour l'audition seulement.

## 7. Un defaut de rendu trouve et corrige DANS CE LOT

Le premier jet du melangeur montait les sources mono en stereo en faisant pointer les deux
canaux sur **le meme tableau** : le gain de chemin etait alors applique **deux fois**
(+7 dB rendus a +14 dB), sur les seules couches mono. Mesure qui l'a revele : la couche 3 du
Gungoose (source stereo) recevait -10,27 dB comme prevu, la couche 2 (source mono) -5,27 au
lieu de -15,27. Corrige, et **garde par un test** (`wav_io_test.go`,
`TestVersStereoNePartagePasLesCanaux`).

## 8. Limites assumees

1. **Le juge final est l'utilisateur.** Toute la validation est indirecte (structure de tags
   + mesures). Aucun de ces sons n'a ete compare a une capture en jeu.
2. **Les gains de bus sont inconnus** (§3) : exacts a l'interieur d'un bus, decales d'une
   constante inconnue entre deux bus.
3. **Le HPF declare n'est pas applique** (§2), faute de correspondance 0..100 -> Hz.
4. **Le lit sub covenant de 31 ms** (§5.4) : regle de lecture defendable, pas certitude.
5. **Le reechantillonnage de hauteur est lineaire**, pas a bande limitee. Sur les +-400 cents
   rencontres l'effet est negligeable, mais ce n'est pas un reechantillonneur de production.
6. **La montee mono vers stereo est a gain unitaire sur les deux canaux** — le meme choix que
   rev10, pour que les deux revisions restent comparables. Le panoramique 3D du jeu n'est pas
   reproductible hors ligne.
7. **`moteur_amorce` / `moteur_queue` n'existent que pour le Scorpion** (§5.6).
8. **Le palier 5 lu comme « boost »** sur Ghost / Wraith / Banshee est une lecture de
   l'echelle de regime, pas un nom lu : les etats du groupe 2275666646 sont des hachages et
   aucun balayage de noms courants ne les a casses.
9. **Le cas « eau peu profonde »** de la destruction (V3D §9) n'est toujours pas rendu.

## 9. Outils ajoutes (`cmd/weapon-sounds`, additifs)

| fichier | lignes | mode | ce qu'il debloque |
|---|---|---|---|
| `hirc_noeuds.go` | 341 | — | `NodeBaseParams` COMPLET : bus, parent, toutes les proprietes, octets bruts des non decodees |
| `hirc_types.go` | 148 | — | contrat JSON entre le dump et le rendu |
| `hirc_event.go` | 473 | `hirc-event` | dump d'un evenement : chaine, gains complets, offsets, RANGED, un dump par etat de `Switch` |
| `hirc_texte.go` | 126 | — | sortie lisible, recopiable telle quelle dans un rapport |
| `wav_io.go` | 259 | — | lecture / ecriture WAV et melange en `float64` |
| `rendu_event.go` | 471 | `rendu-event` | rendu d'un evenement entier : tirages, cadence, offsets, une seule normalisation |
| `mesure_wav.go` | 216 | `mesurer-wav` | fiche spectrale (duree, crete, RMS, crest, bandes, centroide) par FFT |
| `wav_io_test.go` | 50 | — | garde-rails du melangeur (aliasing stereo, boucle, offset) |

### Commandes de reproduction

```bash
# structure HIRC complete (un module a la fois : 7,24 Go puis 2,86 Go)
ws -mode hirc-event -banks 969f4dc6,c468fb55,94d43e95,05a51e0a,a52af042 -out unsc.json
ws -mode hirc-event -module pc/globals/common-rtx-new.module \
   -banks b1f8608b,3fdc61a7,2eaae6d7,ccd43fa8,fda12da2,c682f736,1bb9f097 -out cv.json

# medias : streames (pack) et embarques (banque)
ws -etroit -mode pck-dump -pck "<SFX>/sb_010_veh_cv_ghost.pck" -out wem/ghost
ws -etroit -mode eqip-arbre -module pc/globals/common-rtx-new.module \
   -banks ccd43fa8,fda12da2,c682f736,1bb9f097 -emb emb -out emb_cv.json
vgmstream-cli -o wav/<veh>/<id>.wav wem/<veh>/<id>.wem

# rendu d'un evenement entier
ws -etroit -mode rendu-event -json cv.json -events 8b33a81a -wav wav/small_covenant \
   -dest "<dossier>/Ghost/destruction" -nom explosion -tirages 3 -graine 7
ws -etroit -mode rendu-event -json cv.json -events 603d9e29 -etats 3707760930 \
   -wav wav_mix/ghost -dest "<dossier>/Ghost/deplacement" \
   -nom moteur_conduite_boucle8s -duree 8 -tirages 2 -graine 11

# fiche spectrale d'un dossier livre
ws -etroit -mode mesurer-wav -wav "<dossier>/Ghost/deplacement"
```

Manifeste : `sons_v3_reconstruits/manifeste_v3.json`, revision **11**, champ
`reconstruction_v3e` — 27 evenements decrits noeud par noeud (chaine HIRC, gain amont, gain
de chemin complet, offset, hauteur, fourchettes RANGED, mode de boucle, bus) et 45 lots de
rendu mesures.
