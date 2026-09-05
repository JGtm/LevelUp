# V3F — Les tirs du Ghost, de la Banshee, du Wraith et du Chopper

> Worktree `LevelUp-wt-vehicules`, branche `wt/vehicules-tourelles`. Lecture seule sur les
> donnees du jeu. AUCUN commit, AUCUN `git add`. Code ajoute : `cmd/weapon-sounds/tir_vehicules.go`
> (204 L, additif) + 6 lignes dans `main.go` (386 L). `gofmt` / `go vet` / `go test` / `go build`
> verts. WAV livres = transitoires non commites. GOCACHE dedie, avant-plan, un seul module en
> RAM a la fois. `apps/web/` non touche.

## 0. Verdict en une page

**Les tirs des quatre vehicules sont livres, et ils ne se cherchaient pas au bon endroit.**

La consigne de depart supposait que les evenements de tir restaient a trouver dans les banques
covenant / bannis. Ils n'etaient pas a trouver : **ils y etaient deja, rendus depuis la veille
sous les noms `deplacement/moteur_conduite_boucle8s.wav` et `deplacement/contact_*.wav`**. Le
verdict de l'utilisateur (« les events rendus comme moteur sont des sons d'armes ») n'est pas
une impression : il est maintenant demontre par le champ NOMME du tag d'arme, et il vaut pour
les **six** vehicules du lot V3E — Scorpion et Warthog compris.

Trois preuves independantes, aucune acoustique :

| # | preuve | ce qu'elle vaut |
|---|---|---|
| 1 | Le champ **« Weapon Fire Sound »** du tag `weap` designe un tag `lsnd`/`snd!` dont le corps porte **exactement deux** identifiants d'evenement de la banque du vehicule, sur 7 a 14 presents | probabilite de coincidence sur 94 a 120 mots u32 : **~4e-7** |
| 2 | Dans chacun de ces couples, un evenement n'a **que** des medias **mono**, l'autre **que** des medias **stereo** — la regle 3e / 1re personne etablie par le lot armes le 2026-08-15 | **10 evenements sur 10**, zero exception |
| 3 | Cadence declaree par le tag (`barrels` -> rounds per second) x nombre de canons = **cadence du conteneur** Wwise (`AkTransitionMode` 5) | Ghost 461 contre 450/min, Banshee 480 contre 480, Chopper 240 contre 240, temoin UNSC Falcon 779 contre 780 |

Et un temoin de methode qui n'a pas ete construit pour l'occasion : **la meme chaine, appliquee
aux cinq vehicules dont l'utilisateur a DEJA valide les tirs, retombe sur les fichiers deja
livres** (§4).

Quatrieme resultat, tombe en cours de route : le groupe de switch **2275666646** n'est pas un
regime moteur. Huit de ses neuf etats se cassent par hachage FNV-1 et s'appellent
`exterioropen`, `exteriorsmall`, `exteriormed`, `exteriorlarge`, `interiornarrow`,
`interiorsmall`, `interiormed`, `interiorlarge` : **c'est l'espace de l'auditeur**. Le lot V3E
rendait le « palier 4 » = `interiormed`, une reverberation d'interieur moyen. Ce lot rend
l'etat par DEFAUT declare par la banque, toujours exterieur.

## 1. Ce que la cle proposee dans le mandat donnait — et pourquoi elle ne pouvait pas suffire

Le mandat proposait de reconnaitre les evenements de tir en recoupant les identifiants `.wem`
des dossiers utilisateur (`Covenant_ghost`, `..._EMBARQUES`, ...) avec ceux que chaque
evenement reference. Fait d'abord, et voici ce que ca a rendu.

### 1.1 Le dossier de base recoupe TOUT — donc il ne separe rien

| vehicule | banque | wems atteints par les evenements de la banque | dossier de base | couverts | non couverts |
|---|---|---|---|---|---|
| Ghost | `ccd43fa8` | 289 | 248 | **248** | 0 |
| Wraith | `fda12da2` | 85 | 78 | **78** | 0 |
| Banshee | `c682f736` | 377 | 230 | **230** | 0 |
| Chopper | `1bb9f097` | 79 | 78 | **78** | 0 |

Les evenements de la banque couvrent **100 %** du `.pck` du vehicule. Le recoupement est donc
vrai pour tous les evenements a la fois : il dit « ce media appartient a ce vehicule », il ne
dit jamais « cet evenement est le tir ». Un critere qui vaut pour les 7 evenements du Ghost ne
peut pas en designer un.

### 1.2 Les dossiers `_EMBARQUES` ne recoupent RIEN, et on sait pourquoi

| vehicule | DIDX de la VRAIE banque | dossier `_EMBARQUES` | intersection |
|---|---|---|---|
| Ghost | 289 | 65 | **0** |
| Wraith | 85 | 52 | **0** |
| Banshee | 377 | 279 | **0** |
| Chopper | 79 | 40 | **0** |

Cause identifiee pour le Ghost, sur pieces : `Covenant_ghost_EMBARQUES` est identique **fichier
pour fichier (65/65)** au dossier `bt_bank01862ab3_EMB` du meme poste, c'est-a-dire au chunk
`DIDX` de la banque **`01862ab3`** — la banque du **Warthog**, celle que V3D §10 avait
identifiee comme la mauvaise attribution du Ghost. Ces dossiers datent d'avant cette
correction. La cle « `_EMBARQUES` = perspective embarquee » ne tient donc pas ; c'est la regle
**mono / stereo** (§3.2) qui departage les perspectives, et elle, elle est mesuree.

### 1.3 Le seuil de 30 % du mandat : statue explicitement

Le mandat demandait « >= 30 % des wems du dossier, sinon dis-le ». **Trois evenements sur dix
le passent, sept ne le passent pas, et le seuil n'est pas atteignable** : chaque banque porte 7
a 16 evenements qui se partagent le meme dossier, donc aucun evenement ne peut en representer
30 % sauf a etre le plus gros de la banque. Les chiffres exacts :

| vehicule | mode | persp. | event | wems (tous etats) | dans le `.pck` | part du dossier | couverture de l'event |
|---|---|---|---|---|---|---|---|
| Ghost | M1 | 3P | `a91f9f78` | 47 | 47 | 18,9 % | **100 %** |
| Ghost | M1 | 1P | `603d9e29` | 121 | 100 | **40,2 %** | 83 % |
| Banshee | M1 | 3P | `bdb30da6` | 57 | 57 | 24,7 % | **100 %** |
| Banshee | M1 | 1P | `851558f7` | 131 | **0** | **0 %** | 0 % |
| Banshee | M2 | 3P | `f76415db` | 18 | 18 | 7,8 % | **100 %** |
| Banshee | M2 | 1P | `6ed9c3bc` | 31 | 30 | 13,0 % | 97 % |
| Wraith | M1 | 3P | `46b14f04` | 18 | 18 | 22,8 % | **100 %** |
| Wraith | M1 | 1P | `aa6215eb` | 31 | 30 | **38,0 %** | 97 % |
| Chopper | M1 | 3P | `66b341e8` | 18 | 18 | 22,8 % | **100 %** |
| Chopper | M1 | 1P | `1adf8067` | 31 | 30 | **38,0 %** | 97 % |

Le cas a signaler nommement est **la Banshee M1 en 1re personne (`851558f7`)** : **aucun** de
ses 131 medias n'est dans le `.pck`, ils vivent tous dans le `DIDX` de la banque. Son
identification ne repose donc sur AUCUN recoupement de dossier — uniquement sur les trois
preuves du §3. C'est dit, ce n'est pas cache.

## 2. Ou etait la piste, et comment elle a ete remontee

Le mandat prevoyait ce cas : « si un vehicule n'a pas d'event de tir recoupable dans sa banque,
REMONTE la piste ». La remontee n'a pas eu besoin d'aller chercher ailleurs — elle a eu besoin
d'un **outil qui n'existait pas**.

**Pourquoi il n'existait pas.** La chaine `lot` -> `lot-tir` qui a produit tous les tirs
d'armes ne balaie que `sb_010_wea_*`, `sb_010_tur_*` et `sb_010_whizby_*` (constante
`motifsPck`, `lot.go`). Les chassis `sb_010_veh_*` n'y entrent pas : **aucun `weap` de vehicule
n'avait jamais ete resolu par elle**.

**Mode ajoute : `tir-vehi`** (`tir_vehicules.go`, additif, 204 L) —
`vehi` --[refs `weap` INLINE]--> `weap` --[champ nomme « Weapon Fire Sound »]--> `lsnd`/`snd!`
--[table de dependances]--> `sbnk`, --[corps]--> mots candidats. Deux passes, deux modules,
jamais les deux en RAM.

```
ws -etroit -mode tir-vehi -module any/globals/globals-rtx-new.module        -json tirvehi.json
ws -etroit -mode tir-vehi -module any/globals/common-rtx-new.module         -json tirvehi_common.json
```

Sortie brute, module `any/globals/common` (81 `weap`, 4478 `snd!`+`lsnd`, 32 `weap` portes par
un `vehi`, 15 avec un son de tir) — les cinq lignes qui nous concernent :

```
  weap 0000aa68  vehis[000026ed 0001530a c6e79dcc]  1 mode(s)  cadence=240 cpm   banks[c682f736]
      mode 0  lsnd 572c0c57  banks[c682f736]  94 mots
  weap 0000aa69  vehis[000026ed 0001530a c6e79dcc]  1 mode(s)  cadence=1799 cpm  banks[c682f736]
      mode 0  snd! dc3d707b  banks[c682f736]  120 mots
  weap 00015435  vehis[0000d3dc 5b80c406 9af9e693]  1 mode(s)  cadence=225 cpm   banks[ccd43fa8]
      mode 0  lsnd 809481d1  banks[ccd43fa8]  94 mots
  weap 121b4009  vehis[233c877d]                    1 mode(s)  cadence=1799 cpm  banks[fda12da2]
      mode 0  snd! 5116c72e  banks[fda12da2]  120 mots
  weap b40e9618  vehis[002ba902 3d4a8a5a]           1 mode(s)  cadence=120 cpm   banks[1bb9f097]
      mode 0  lsnd f87b01a8  banks[1bb9f097]  94 mots
```

**La banque designee par le champ de tir EST celle du chassis.** Le tir n'etait pas ailleurs :
il etait dans la banque du vehicule, sous un nom que le lot precedent avait lu comme un moteur.

`cadence=1799 cpm` = 29,98 rounds/s : c'est la valeur que le tag porte quand il ne declare pas
de cadence (elle apparait a l'identique sur le Scorpion, dont le manifeste rev9 note
`cadence_cpm: 0`). Elle est traitee comme **absente**, pas comme une cadence.

Recoupement independant, hors de cette chaine : le journal de retro-ingenierie de l'arme du
kill (`.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md`) porte, pour les tags de classe VEHICULE,
`00015438 ARME DE VEHICULE (vehi 5b80c406, sb_010_veh_cv_ghost, classe fixed)` et
`0001535b ... sb_010_veh_cv_banshee`. Meme `vehi` (5b80c406) que celui trouve ici pour le
`weap 00015435` du Ghost, meme banque de chassis.

## 3. Les trois preuves, sur pieces

### 3.1 Le champ nomme

Intersection entre les mots u32 du corps du tag de son et les identifiants d'evenement declares
par la banque :

```
Ghost    weap 00015435  lsnd 809481d1  bank ccd43fa8  -> a91f9f78, 603d9e29   (7 events dans la banque)
Banshee  weap 0000aa68  lsnd 572c0c57  bank c682f736  -> bdb30da6, 851558f7   (14 events)
Banshee  weap 0000aa69  snd! dc3d707b  bank c682f736  -> f76415db, 6ed9c3bc   (14 events)
Wraith   weap 121b4009  snd! 5116c72e  bank fda12da2  -> 46b14f04, aa6215eb   (9 events)
Chopper  weap b40e9618  lsnd f87b01a8  bank 1bb9f097  -> 66b341e8, 1adf8067   (7 events)
```

**Toujours deux, jamais un, jamais trois.** Les cinq autres evenements de la banque du Ghost —
`100acbe4`, `9db16175`, `ccaf1444`, `d02b3690`, `d8b036c6`, tous sur le bus `0f233096` — ne
sont designes par aucun champ de tir et **ne sont pas rendus par ce lot**.

### 3.2 Mono / stereo = 3e / 1re personne

Regle etablie par le lot armes, verbatim (`PLAN_EXTRACTION_SONS_ARMES.md`, 2026-08-15) : « les
evenements vont par paires **mono/stereo** : troisieme et premiere personne ». Canaux mesures
dans les en-tetes WAV des medias decodes :

```
vehicule  event     nwem  mono  stereo  hors pck  duree moy   -> perspective
Ghost     a91f9f78    47    47       0         0     0,24 s      3P
Ghost     603d9e29   121     0     100        21     0,21 s      1P
Banshee   bdb30da6    57    57       0         0     0,13 s      3P
Banshee   851558f7   131     0       0       131        -        1P (medias 100 % embarques)
Banshee   f76415db    18    18       0         0     1,26 s      3P
Banshee   6ed9c3bc    31     0      30         1     4,55 s      1P
Wraith    46b14f04    18    18       0         0     2,45 s      3P
Wraith    aa6215eb    31     0      30         1     5,95 s      1P
Chopper   66b341e8    18    18       0         0     1,30 s      3P
Chopper   1adf8067    31     0      30         1     3,17 s      1P
```

Zero melange : un evenement est **integralement** mono ou **integralement** stereo. Et la 1P est
systematiquement la plus longue — coherent avec « en 1re personne on entend la mecanique et la
queue » (meme plan, meme date).

### 3.3 La cadence

| vehicule | cadence du tag (par canon) | canons | attendu | cadence du conteneur (`AkTransitionMode` 5) | mesure |
|---|---|---|---|---|---|
| Ghost | 225 cpm | 2 | 450/min | **0,130 s** | 461/min |
| Banshee M1 | 240 cpm | 2 | 480/min | **0,125 s** | 480/min |
| Chopper | 120 cpm | 2 | 240/min | **0,250 s** | 240/min |
| Falcon LMG (temoin UNSC) | 780 cpm | 1 | 780/min | **0,077 s** | 779/min |
| Wraith, Banshee M2 | non declaree | — | — | **aucune** (mode 0) | one-shot |

Le temoin Falcon est le plus net : 780 cpm declares, `transition = 0,077 s` lue dans le
conteneur, soit 779 coups/min. Reserve honnete : le facteur **x2** des canons jumeles est une
connaissance du jeu, pas une lecture de tag — le tag ne declare que la cadence par canon.

## 4. Le temoin de methode : les cinq vehicules DEJA valides

La meme chaine a ete passee sur les banques dont l'utilisateur a accepte les tirs le
2026-08-31. Elle n'a pas ete calibree sur elles : elle a ete **verifiee** contre elles.

```
FalconLMG  weap 00015cd3  lsnd 68c0807f  cadence=780   -> d36e3c93 (mono, 57 wems) , 2b145170 (stereo, 120, switch)
Scorpion   weap 00015cfa  snd! b3adf402  cadence=1799  -> 251190f3 (mono, 18)      , 951f76c0 (stereo, 31, switch)
FalconLMG  weap 0c6fd911  lsnd 9931f9dd  cadence=1080  -> 01de6d3a (mono, 57)      , a490890b (stereo, 122, switch)
Wasp       weap 11725dc4  snd! ce8e2f81  cadence=450   -> e22a0d32 (mono, 18)      , b0554fc9 (stereo, 31, switch)
Warthog    weap c7d50912  snd! 155f1354  cadence=125   -> 38b83eb8 (mono, 18)      , 68b1a949 (stereo, 31, switch)
Wasp       weap d3c407ed  lsnd 8ba95ae0  cadence=600   -> 5baca8ee (mono, 57)      , ad6b4715 (stereo, 121, switch)
```

Rapprochement avec les fichiers deja livres et acceptes (durees du manifeste rev9) :

| vehicule | event 3P designe (duree moy. des medias) | `tir_M1_3p_1.wav` livre | event 1P designe | `tir_M1_1p_1.wav` livre |
|---|---|---|---|---|
| Scorpion | `251190f3` 2,25 s | **2,95 s** | `951f76c0` 6,34 s | **7,51 s** |
| Warthog | `38b83eb8` 1,67 s | **2,94 s** | `68b1a949` 4,03 s | **4,56 s** |
| Wasp | `e22a0d32` 2,18 s | **3,13 s** | `b0554fc9` 3,85 s | **4,26 s** |
| Falcon LMG | `d36e3c93` 0,27 s | **0,71 s** | `2b145170` 0,28 s | **3,83 s** |

**Et c'est la que la contradiction eclate** : `951f76c0` est l'evenement que V3E a rendu sous
`Scorpion/deplacement/moteur_conduite_boucle8s.wav`, et c'est le meme que
`Scorpion/tir/tir_M1_1p_1.wav` que l'utilisateur a valide comme un TIR. Le meme evenement a ete
livre deux fois, sous deux noms qui se contredisent. Le champ nomme tranche : c'est le tir.

## 5. Le switch 2275666646 n'est pas un regime moteur

Les neuf etats sont des hachages **FNV-1 32 bits** de noms en minuscules, sans separateur.
Balayage de listes de noms candidats (~30 M d'essais, temoin negatif : `fnv1("")` = 2166136261
retrouve) :

| etat | nom casse | ce que V3E en disait |
|---|---|---|
| `3561860439` | **exterioropen** | « palier 1 ralenti, DEFAUT covenant » |
| `356702912` | **exteriorsmall** | « palier 1 ralenti, DEFAUT UNSC » |
| `2311227951` | **exteriormed** | « palier 1, alias » |
| `1975887784` | **exteriorlarge** | « palier 1, alias » |
| `1136871302` | **interiorlarge** | « palier 3 intermediaire » |
| `1248419637` | **interiormed** | « palier 4 EN CONDUITE » (l'etat rendu par rev11) |
| `3707760930` | **interiorsmall** | « palier 4, alias » |
| `163696720` | **interiornarrow** | « palier 5 plein regime / boost » |
| `1093928064` | **NON CASSE** | « palier 2 bas » |

Deux des huit noms (`interiormed`, `exteriormed`) sont tombes sur un balayage generique AVANT
qu'aucune hypothese « interieur / exterieur » ne soit formulee ; les six autres ont suivi sur un
balayage cible. L'esperance de faux positifs du premier balayage etait de **0,06** : la paire
n'est pas un hasard.

Ce que ca change : les quatre etats `exterior*` pointent tous, sur le Ghost, vers **le meme**
enfant `3b92b1b4`, tandis que les `interior*` en ont quatre distincts — c'est la signature
d'une **variation de reverberation**, pas d'une echelle de puissance. Et l'« echelle monotone »
de V3E (niveau qui monte, duree qui baisse) se relit sans effort : un espace etroit rend un son
plus court et plus dense qu'un exterieur ouvert.

**Consequence appliquee** : ce lot rend, pour chaque evenement, **l'etat par DEFAUT que la
banque declare** — `exterioropen` pour le Ghost et la Banshee M1, `exteriorsmall` pour le
Wraith, le Chopper et la Banshee M2. Toujours un exterieur, ce qui est la bonne lecture pour
une carte multijoueur.

## 6. Donnees brutes, evenement par evenement

Chaine HIRC complete telle que `hirc-event` la publie (gains de chemin COMPLETS : parents
actor-mixer + `MakeUpGain` + gain de noeud).

### 6.1 Ghost — `weap 00015435`, `lsnd 809481d1`, canons a plasma jumeles

```
=== sbnk ccd43fa8 / event a91f9f78 : 1 action(s), 3 couche(s) ===      [3e personne, mono]
  action 1b6aa392  Play (0x0403) -> 08db6152  delai=0.000s
  couche 1 : 0a7965fc RandomSequence  amont=-12.00 dB  bus=8165b6c5 HORS BANQUE
      chemin : action 1b6aa392(+0.000s) <- 1fb0270a(ActorMixer,-5.0dB) <- 2274a969(ActorMixer,+0.0dB)
        <- 21352ab8(ActorMixer,-7.0dB) <- 0af1684d(ActorMixer,+0.0dB) <- 280705ab(ActorMixer,+0.0dB)
        | bus 8165b6c5 -> 08db6152(Blend,+0.0dB) -> 0a7965fc(RandomSequence,-1.0dB)
        -> 18ef53aa(RandomSequence,+2.0dB) -> 253f5514(Sound,+0.0dB)
      RANGED volume : -3.00 .. +0.00 dB     RANGED hauteur : -80 .. +80 cents
      repetitions : 0 (boucle infinie)  mode_transition=5 (CADENCE)  duree=0.130s
      16 variantes, gain de chemin -11,00 dB
  couche 2 : 31e0da9b  15 variantes, gain -2,00 dB, RANGED hauteur -85 .. +80 cents
  couche 3 : 3b58773c  16 variantes, gain -12,50 dB, RANGED hauteur -40 .. +40 cents

=== sbnk ccd43fa8 / event 603d9e29 / etat 3561860439 (exterioropen) : 2 couche(s) ===  [1P, stereo]
  action 04247f19  Play -> 37f07dd4   |   action 255ddd90  Play -> 2199fb1d
  couche 1 : 37f07dd4 RandomSequence  amont=-7.00 dB  bus=5a880943
      chemin : ... <- 10f589ad(-2.0dB) <- 138d1e40(-8.0dB) <- 2774363f(+3.0dB) <- 0af1684d <- 280705ab
        | bus 5a880943 -> 37f07dd4(+1.0dB) -> 107d73ac(Switch) -> 3b92b1b4 -> 2480e075(-2.0dB) -> Sound
      repetitions : 0  mode_transition=5  duree=0.130s   24 variantes, -6 a -13 dB selon la variante
  couche 2 : 2199fb1d RandomSequence  amont=-4.00 dB  bus=1f17314c
      repetitions : 0  mode_transition=5  duree=0.100s   1 variante (wem 87187708), gain -4,00 dB
```

### 6.2 Banshee — deux armes

```
M1 canons a plasma (weap 0000aa68, lsnd 572c0c57, 240 cpm x2, cadence 0,125 s)
  3P bdb30da6 : 3 couches, 19+19+19 variantes, bus 8165b6c5, amont -7 dB,
                gains -1 / -11 / -5 dB, RANGED volume -3..0 dB sur les trois,
                RANGED hauteur -85..80 (c1) et -80..80 (c3), medias 0,12-0,16 s
  1P 851558f7 / etat 3561860439 (exterioropen) : 2 couches
                c1 282e786e  26 variantes  bus 5a880943  amont -3 dB  gain -17 dB  cadence 0,125 s
                c2 272977ff   1 variante   bus 1f17314c  amont -4 dB  gain -15 dB
                             RANGED hauteur 0 .. +800 cents (le lit sub covenant, 31 ms)

M2 tir lourd unique (weap 0000aa69, snd! dc3d707b, aucune cadence)
  3P f76415db : 3 couches, 6+6+6, bus 8165b6c5, amont -7 dB, gains +5 / +4 / +11 dB,
                mode 0 (pas de boucle), medias 1,20-1,30 s
  1P 6ed9c3bc / etat 356702912 (exteriorsmall) : c1 279390c1 6 variantes bus 5a880943 amont -3 dB
                gain -5 dB ; c2 3f98b1bf Sound bus 1f17314c gain +3 dB ; medias 4,87-6,55 s
```

### 6.3 Wraith — mortier a plasma, one-shot

```
weap 121b4009, snd! 5116c72e, aucune cadence declaree, conteneurs en mode 0
  3P 46b14f04 : 3 couches, 6+6+6, bus 8165b6c5, amont -8 dB, gains +2 / +3 / +9 dB,
                medias 1,89-3,60 s
  1P aa6215eb / etat 356702912 (exteriorsmall) : c1 18a1ef6e 6 variantes bus 5a880943 amont -2 dB
                gain -8 dB ; c2 3c133923 Sound bus 1f17314c amont 0 dB gain +3 dB ; medias 6,85 s
```

### 6.4 Chopper — canons jumeles

```
weap b40e9618, lsnd f87b01a8, 120 cpm x2, cadence conteneur 0,250 s, delai d'action 0,038 s
  3P 66b341e8 : 3 couches, 6+6+6, bus 8165b6c5, amont -8 dB, gains -8 / -5 / -8,5 dB,
                mode 5 cadence 0,250 s, RANGED hauteur -48..+43 cents (c3), medias 0,99-1,54 s
  1P 1adf8067 / etat 356702912 (exteriorsmall) : c1 073c179e 6 variantes bus 5a880943 amont -10,5 dB
                gain -10 dB ; c2 160b118a 1 variante bus 1f17314c amont -4 dB gain -19 dB,
                RANGED hauteur 0..+800 cents ; delai 0,038 s sur les deux couches
```

## 7. Ce qui est livre

`sons_v3_reconstruits/<Vehicule>/tir/` — convention de nommage reprise a l'identique des cinq
dossiers `tir/` valides le 2026-08-31.

| fichier | contenu |
|---|---|
| `tir_M1_3p_1.wav`, `tir_M1_3p_2.wav` | un coup, 3e personne, deux tirages (1 = indice 0 partout, 2 = tirage aleatoire reel) |
| `tir_M1_1p_1.wav`, `tir_M1_1p_2.wav` | idem, 1re personne |
| `tir_M2_*` (Banshee) | le second mode de tir |
| `tir_RAFALE_M1_3p.wav`, `tir_RAFALE_M1_1p.wav` | 3,00 s de tir soutenu a la cadence reelle du conteneur |
| `tir_RAFALE.wav` | copie de la rafale 3P (la 3e personne est la perspective du rejeu 2D) |
| `stems/<nom>_coucheN_<noeud>.wav` | les couches isolees du tirage 1, au MEME gain de normalisation que le melange |

Pas de rafale pour le **Wraith** ni pour la **Banshee M2** : leurs conteneurs sont en mode 0 et
leur tag ne declare aucune cadence — fabriquer une rafale la serait une invention.

### 7.1 Mesures des fichiers livres

dBFS sauf le centroide (Hz). Instrument : `ws -mode mesurer-wav` (FFT), le meme qu'aux lots
V3B / V3E.

| vehicule | fichier | duree | crete | RMS | crest | bas<200 Hz | haut 3-8 k | centroide |
|---|---|---|---|---|---|---|---|---|
| Ghost | `tir_M1_3p_1` | 0,128 | -1,00 | -12,76 | 11,76 | -14,05 | -36,21 | 1788 |
| Ghost | `tir_M1_3p_2` | 0,645 | -4,30 | -20,00 | 15,70 | -21,52 | -40,01 | 2068 |
| Ghost | `tir_M1_1p_1` | 0,678 | -1,00 | -19,50 | 18,50 | -21,56 | -36,05 | **4996** |
| Ghost | `tir_M1_1p_2` | 0,125 | -2,12 | -15,59 | 13,47 | -18,27 | -32,75 | 3258 |
| Ghost | `tir_RAFALE_M1_3p` | 3,000 | -1,00 | -14,53 | 13,53 | -15,76 | -33,90 | 2083 |
| Ghost | `tir_RAFALE_M1_1p` | 3,000 | -1,00 | -12,22 | 11,22 | -12,56 | -34,33 | 2908 |
| Banshee | `tir_M1_3p_1` | 0,125 | -1,00 | -15,61 | 14,61 | -16,54 | -36,71 | 2113 |
| Banshee | `tir_M1_3p_2` | 0,125 | -1,06 | -16,30 | 15,23 | -17,77 | -36,43 | 2227 |
| Banshee | `tir_M1_1p_1` | 0,125 | -1,02 | -12,52 | 11,50 | -15,12 | -32,16 | 2861 |
| Banshee | `tir_M1_1p_2` | 0,125 | -1,00 | -12,51 | 11,51 | -15,15 | -30,44 | 3013 |
| Banshee | `tir_M2_3p_1` | 1,283 | -2,28 | -17,77 | 15,49 | -18,27 | -41,62 | 1498 |
| Banshee | `tir_M2_3p_2` | 1,260 | -1,00 | -17,19 | 16,19 | -17,44 | -41,41 | 1475 |
| Banshee | `tir_M2_1p_1` | 5,574 | -1,00 | -21,12 | 20,12 | -23,59 | -40,35 | 1791 |
| Banshee | `tir_M2_1p_2` | 6,554 | -1,25 | -22,11 | 20,86 | -24,74 | -41,22 | 1847 |
| Banshee | `tir_RAFALE_M1_3p` | 3,000 | -1,00 | -16,29 | 15,29 | -17,46 | -32,66 | 2711 |
| Banshee | `tir_RAFALE_M1_1p` | 3,000 | -1,00 | -12,59 | 11,59 | -13,64 | -29,11 | 2804 |
| Wraith | `tir_M1_3p_1` | 3,200 | -1,00 | -20,04 | 19,04 | -20,67 | -40,12 | 1973 |
| Wraith | `tir_M1_3p_2` | 3,200 | -2,30 | -21,10 | 18,80 | -21,65 | -40,80 | 2080 |
| Wraith | `tir_M1_1p_1` | 6,845 | -1,00 | -22,01 | 21,01 | -23,92 | -41,48 | 2102 |
| Wraith | `tir_M1_1p_2` | 6,845 | -1,00 | -21,75 | 20,75 | -23,36 | -41,11 | 2015 |
| Chopper | `tir_M1_3p_1` | 1,464 | -1,00 | -25,29 | 24,29 | -26,69 | -45,12 | 2628 |
| Chopper | `tir_M1_3p_2` | 1,444 | -2,10 | -25,73 | 23,63 | -27,16 | -45,20 | 2646 |
| Chopper | `tir_M1_1p_1` | 4,398 | -1,66 | -21,82 | 20,16 | -23,50 | -37,16 | 2724 |
| Chopper | `tir_M1_1p_2` | 4,366 | -1,00 | -22,13 | 21,13 | -23,73 | -37,30 | 2634 |
| Chopper | `tir_RAFALE_M1_3p` | 3,038 | -1,00 | -18,15 | 17,15 | -19,13 | -38,55 | 2880 |
| Chopper | `tir_RAFALE_M1_1p` | 3,038 | -1,00 | -12,56 | 11,56 | -14,06 | -28,91 | 3416 |

Lecture : les quatre timbres sont distincts et coherents avec les armes. Le **Wraith** est le
plus grave et le plus etale (crest 19-21, bande 3-8 kHz a -40 dB) — un mortier. La **Banshee M1**
et le **Ghost** ont les coups les plus brefs (0,125 s) et les centroides les plus hauts en 1re
personne (2861 et 4996 Hz) — des canons a plasma rapides. Le **Chopper** est le plus faible en
RMS a coup unique (-25,3 dB) et remonte le plus en rafale (-18,2) : ses grains de 1,0-1,5 s se
chevauchent quatre a six fois a 0,250 s de cadence.

## 8. La chaine de rendu, exactement

```
sbnk (module) --hirc-event--> plan JSON (structure + gains complets + offsets + RANGED + un dump PAR ETAT)
pck AKPK      --pck-dump----> wem complets --vgmstream--> wav
banque DIDX   --eqip-arbre -emb--> wem embarques --vgmstream--> wav   (medias absents du pack)
plan + wav    --rendu-event--> somme en float64 -> UNE normalisation a -1 dBFS -> WAV 24 bits
```

Reglages appliques par couche, dans cet ordre : hauteur (`Pitch` + centre de la fourchette
RANGED) -> montee mono vers stereo a gain unitaire -> gain de chemin COMPLET (`Volume` +
`MakeUpGain` de tout le chemin + centre de la fourchette RANGED de volume) -> remplissage de la
duree pour les rafales (cadence reelle, lectures qui se chevauchent) -> offset (`InitialDelay`,
AkPropID 59) -> somme. **Aucune egalisation, aucune compression, aucun limiteur.**

Priorite des medias : le `.wem` du `.pck` (complet) quand il existe, le `.wem` embarque du
`DIDX` sinon (regle V3E : un media present dans un pack n'a qu'un prefetch tronque dans la
banque).

**Controle de non-regression du decodage** : les 248 / 78 / 230 / 78 fichiers communs avec le
dossier utilisateur ont un ecart de duree maximal de **0,0000 s** et un format (canaux,
frequence) identique a 100 %. Le decodeur est le meme, la chaine aussi.

Commandes exactes :

```bash
# structure HIRC complete, un module a la fois
ws -mode hirc-event -module pc/globals/common-rtx-new.module \
   -banks ccd43fa8,fda12da2,c682f736,1bb9f097 -out cv4.json
ws -mode hirc-event -banks 05a51e0a,a52af042,4993b379,38167604,bd807a77 -out unsc5.json   # temoin

# designation du tir par le champ nomme
ws -etroit -mode tir-vehi -module any/globals/common-rtx-new.module -json tirvehi_common.json

# medias
ws -etroit -mode pck-dump -pck "<SFX>/sb_010_veh_cv_ghost.pck" -out wem/ghost
ws -etroit -mode eqip-arbre -module pc/globals/common-rtx-new.module \
   -banks ccd43fa8,fda12da2,c682f736,1bb9f097 -emb emb -out emb_cv.json
vgmstream-cli -o wav/ghost/?f.wav <wem>

# rendu (exemple Ghost)
ws -etroit -mode rendu-event -json cv4.json -events a91f9f78 -wav wav/ghost \
   -dest "<...>/Ghost/tir" -nom tir_M1_3p -tirages 2 -graine 7
ws -etroit -mode rendu-event -json cv4.json -events 603d9e29 -etats 3561860439 -wav wav/ghost \
   -dest "<...>/Ghost/tir" -nom tir_RAFALE_M1_1p -duree 3 -tirages 1 -graine 7
```

## 9. Compte rendu honnete — ce qui est prouve, ce qui est choisi

**Prouve sur pieces :**

1. Les dix evenements retenus sont ceux que le champ nomme « Weapon Fire Sound » designe. Ce
   n'est ni une lecture de spectre ni une heuristique.
2. La repartition 3P / 1P : mono contre stereo, 10 sur 10, sans exception.
3. La cadence des conteneurs correspond a la cadence declaree par le tag (au facteur canons
   pres, §3.3).
4. Huit des neuf etats du switch 2275666646 sont des noms d'espace d'auditeur, casses par
   FNV-1.
5. Les fichiers `deplacement/moteur_*.wav` et `deplacement/contact_*.wav` du lot V3E sont ces
   memes evenements de tir, sur les six vehicules.
6. Les dossiers `_EMBARQUES` du poste utilisateur ne recoupent pas les banques des vehicules ;
   celui du Ghost est le `DIDX` de la banque du Warthog.

**Choisi, et declare comme tel :**

1. **La duree de rafale : 3,00 s.** La recette exacte de la rafale du lot du 2026-08-31 n'est
   pas reproductible depuis les rapports (les durees livrees vont de 1,75 a 6,71 s sans regle
   retrouvable). 3,00 s est un nombre rond, pas une mesure.
2. **L'etat de switch rendu = le DEFAUT declare par la banque.** C'est le choix le moins
   inventif, mais c'est un choix : un autre etat exterieur (`exteriorlarge`) serait defendable
   sur une grande carte.
3. **`tir_RAFALE.wav` = la rafale 3P**, par application de la decision produit du lot armes
   (« pour le rejeu 2D, la camera n'est pas a la premiere personne »).
4. **La graine du 2e tirage** : 7 par defaut, portee a 11 quand 7 retombait sur l'indice 0 et
   livrait deux fichiers identiques (Chopper 1P, Wraith 1P, Banshee M2 1P). Le tirage 1 reste
   l'indice 0 partout, convention V3E.
5. **M1 / M2 pour la Banshee** : M1 = les canons a plasma (arme principale, cadence declaree),
   M2 = le tir lourd unique. L'ordre est une lecture, le tag ne numerote pas les armes.

**Non resolu, et pas masque :**

1. **`851558f7` (Banshee M1 1P) n'a aucun recoupement de dossier** : ses 131 medias sont
   uniquement dans le `DIDX`. Il tient sur les preuves 1, 2 et 3, pas sur la 4e.
2. **L'etat `1093928064`** du groupe 2275666646 n'est pas casse.
3. **Les gains de bus restent inconnus** : zero objet `Bus` dans les banques ouvertes. Les
   gains sont exacts a l'interieur d'un bus, decales d'une constante inconnue entre deux bus.
   Consequence directe : le rapport de niveau entre la couche 1P principale (`5a880943`) et son
   lit sub (`1f17314c`) porte cette incertitude.
4. **Le HPF declare n'est pas applique** (pas de correspondance 0..100 -> Hz dans la banque),
   comme au lot V3E.
5. **Aucun de ces sons n'a ete compare a une capture en jeu.** Le juge reste l'utilisateur.
6. **Les cinq evenements par banque sur le bus `0f233096`** (16 pour le Falcon) ne sont
   designes par aucun champ de tir. Ils portent la meme cadence que le tir du meme vehicule
   (Falcon : 0,077 s sur les 16), ce qui suggere des variantes de distance du meme tir — c'est
   une **lecture, pas une mesure**, et rien n'a ete rendu depuis eux.
7. **Le nom des bus n'est pas casse** : le meme balayage FNV-1 qui a rendu huit etats de switch
   n'a rien donne sur `8165b6c5`, `5a880943`, `1f17314c`, `0f233096`. « bus du tir 3P / 1P »
   reste une deduction par les evenements qui les empruntent, pas un nom lu.

**Hors perimetre, signale et NON traite** (regle du chantier : noter, ne pas corriger) :

- Les dossiers `<Vehicule>/deplacement/` des six vehicules contiennent des sons de tir sous des
  noms de moteur. Rien n'a ete supprime ni renomme : c'est a l'utilisateur de decider. Le champ
  `alerte_v3f_deplacement` du manifeste rev12 porte la liste exacte des correspondances.
- Le vrai son de deplacement reste, comme le disait deja le lot V3 rev7, le `lsnd` PARTAGE
  `06ba1096` / banque `e793c135` (`Mouvement_generique_partage/`). La piste « moteur par
  vehicule dans les banques » est fermee ; le protocole de capture en jeu reste la voie.

## 10. Outil ajoute

| fichier | lignes | mode | ce qu'il debloque |
|---|---|---|---|
| `cmd/weapon-sounds/tir_vehicules.go` | 204 | `tir-vehi` | `vehi` -> `weap` -> champ nomme « Weapon Fire Sound » -> tags de son -> `sbnk` + mots. Ferme le trou de `lot`/`lot-tir`, qui ne balaie que les packs d'armes et de tourelles et n'a donc jamais resolu un `weap` de chassis. |

`main.go` : +6 lignes (un `case` et son entree de documentation), 386 lignes au total.
`gofmt` : rien a reformater. `go vet` : propre. `go test ./cmd/weapon-sounds/` : ok.
`go build` : ok.
