# Lot F2 — `ti=47 i2 personal-ai-data` : la largeur, la valeur, et le verdict

> Perimetre : plan `.ai/V7.5/replay2d/PLAN_TI47_ANNONCES_ZONE.md`, phases 0 et 1 completes,
> arret au verdict 1.5. **LECTURE SEULE.** Aucun code de production modifie : ni `traverse.go`,
> ni un `components_*.go`, ni un hook, ni un schema d'artefact, ni le contrat OpenAPI, ni le
> rendu web. Cinq fichiers de TEST ajoutes, sous garde `TI47_FILM`.
> Mesures du 2026-08-24, branche `wt/ti47-annonces`, worktree `LevelUp-wt-ti47`.
> Corpus : les **11 films** du lot C (5 modes), chunks et manifestes du depot principal, en
> lecture seule. Journaux bruts : `lotF2/LOTF2_phase0.log`, `lotF2/LOTF2_phase1.log`.
> Histogrammes complets : `lotF2/<short8>_ti47_chainage.tsv`.

---

## 0. Verdict en une page

**NEGATIF, MESURE.** `ti=47 i2 personal-ai-data-component` **ne porte pas d'annonces de zone**,
ni d'annonces d'aucune sorte. Ce n'est pas un canal d'evenements : c'est une **valeur repliquee
en continu, une par JOUEUR, a 20 Hz exactement**.

Les cinq faits qui le disent, chacun chiffre plus bas :

1. **La largeur est etablie** : `i2` = `R(45)`, record singleton de 72 bits. Mesuree sur les
   octets (98,7 % et 98,4 % des distances entre records consecutifs, sur 4 795 et 3 771
   distances), avec le temoin positif du meme archetype (`i1` = `R(24)`, porte) retrouve a 100 %.
2. **La cadence est 50 ms, pile** : ecart median entre deux emissions d'un meme slot = 50 ms
   (p90 = 51 ms) sur **tous** les slots de **tous** les films ou le canal parle. Une annonce ne
   tombe pas 20 fois par seconde.
3. **L'archetype est par JOUEUR, pas par zone** : 8 slots pour 8 joueurs sur 9 films sur 11
   (9 et 10 sur les deux autres), **quel que soit le nombre de zones du mode** (3 en Bastion, 1
   colline en KOTH, 0 en Slayer). Un objet d'annonce de zone serait par zone ou unique.
4. **Il n'y a pas d'alphabet** : 12 760 valeurs distinctes pour 14 451 emissions (`7344d24f`).
   Aucune enumeration, donc aucune « classe de message » a apparier a un type d'evenement.
5. **Aucun alignement temporel** : les variations hors norme de la valeur tombent aussi souvent
   pres d'un evenement de zone que le **temoin « meme loi, decale »** — exces **1,02x** et
   **1,17x** sur les deux Bastion, **0,33x a 4,00x sur 1 a 4 evenements** en KOTH (compte trop
   faible pour conclure autre chose que « rien »).

Consequence produit, ecrite pour qu'on ne la re-cherche pas : **l'effet UI a la capture voulu par
l'utilisateur ne peut pas venir de ce canal.** L'oracle de capture existe deja et est meilleur
(`zoneStates`, schema 15+, et sa jauge en direct au schema 18) : l'effet se cable dessus, pas ici.

**Ce que ce lot laisse en acquis, et qui vaut plus que le negatif** : la largeur de `i2` est
desormais connue, donc ti=47 est **entierement traversable** (ses trois composants ont une
largeur), et la methode de mesure de largeur sans binaire est outillee et rejouable.

---

## 1. La largeur, sans le binaire (item 0.1)

### 1.1 La voie prescrite est fermee, et c'est ecrit

Le plan demandait les largeurs « depuis le descriptor +0x28, jamais devinees ». Ce chemin passe
par une instance Ghidra du `HaloInfinite.exe` de l'utilisateur. `mcp__ghidra__list_instances`
rend `{"instances": []}` : **aucune instance n'est ouverte**, et le lot est offline pur. La
largeur ne pouvait donc pas etre LUE.

Elle n'a pas ete devinee pour autant. Elle a ete **mesuree sur les octets**, par le protocole du
lot C-bis (`LOTCBIS_PHASE0` §3.2 : « un record correctement dimensionne se termine la ou le
suivant commence »), etendu de « confirmer une largeur connue » a « decouvrir une largeur
inconnue » — donc avec un histogramme sur TOUS les decalages, et non un test sur celui qu'on
esperait.

### 1.2 Quatre mesures, et pourquoi il en fallait quatre

Une seule mesure se serait fait piéger. La charge utile de `i2` contient un motif qui **ressemble
a un en-tete de record** : un « en-tete valide » demarre au bit **+1** dans **70,7 %** des cas sur
`7344d24f`, et jusqu'a **99,5 %** sur `606d9844`. Un instrument naif aurait conclu « largeur 1 ».

| mesure | ce qu'elle voit | son piege |
|---|---|---|
| chainage, cible = TOUS les slots recenses | tous les decalages ou un en-tete valide demarre | voit le faux pic a +1 |
| chainage, cible = les SEULS slots de ti=47 | idem, mais l'en-tete doit designer un objet de CET archetype | demande deux objets de l'archetype d'affilee |
| distance entre debuts de records consecutifs | la taille reelle : en-tete(21) + index(6) + largeur | demande deux records dans le meme paquet |
| longueur de chaine par largeur candidate | refutation : une fausse largeur ne tient pas deux pas | — |

### 1.3 Le temoin positif est dans le meme archetype

`ti=47` n'a que trois composants : `i0 splash-message-static`, `i1 splash-message-dynamic`
(porte, `R(24)`), `i2 personal-ai-data`. **`i1` sert de temoin positif sur chaque film** — si les
quatre mesures ne le retrouvent pas a 24, rien de ce qu'elles disent de `i2` n'a de valeur.

| film | mode | i1 : distance dominante | i1 : chainage cible BANDE | i2 : distance dominante |
|---|---|---|---|---|
| `7344d24f` | Bastion | 51 bits (W=24) — **100,0 %** (2 117) | d=24 : 87,1 % | **72 bits (W=45) — 98,7 %** (4 795) |
| `696a9d7c` | Bastion | 51 bits (W=24) — **100,0 %** (2 140) | d=24 : 87,2 % | **72 bits (W=45) — 98,4 %** (3 771) |
| `01e1f945` | KOTH | 51 bits (W=24) — 94,4 % (270) | d=24 : 74,1 % | 63 distances seulement (voir 1.5) |
| `0a247154` | KOTH | 51 bits (W=24) — 99,6 % (274) | d=24 : 88,3 % | 64 distances seulement |
| `606d9844` | KOTH | 51 bits (W=24) — 99,6 % (262) | d=24 : 78,1 % | 10 distances seulement |
| `8076f97f` | KOTH | 51 bits (W=24) — 97,0 % (264) | d=24 : 71,7 % | 21 distances seulement |
| `24dbb67d` | Oddball | 51 bits (W=24) — 99,7 % (4 263) | d=24 : 87,4 % | 3 distances |
| `530820e5` | CTF | 51 bits (W=24) — 99,8 % (4 892) | d=24 : 87,4 % | 4 distances |
| `53ce4390` | CTF | 51 bits (W=24) — 99,9 % (7 207) | d=24 : 87,4 % | 0 distance |
| `64e8adfa` | CTF | 51 bits (W=24) — 99,8 % (11 609) | d=24 : 87,4 % | 36 distances, sans mode |
| `000d5950` | Slayer | 51 bits (W=24) — 100,0 % (273) | d=24 : 84,0 % | 0 distance |

51 = 21 (en-tete) + 6 (un index) + 24. Le temoin est retrouve **sur les onze films**, jusqu'a la
serie complete : `i1` chaine aussi a d = 75, 126, 177 (pas de 51 bits — les records suivants du
meme bloc) a des taux **87,4 / 74,9 / 62,4 / 49,9 %** identiques a la troisieme decimale sur
quatre films, soit exactement 7/8, 6/8, 5/8, 4/8 : les objets de ti=47 emettent **par blocs de
huit**, et c'est une signature, pas une coincidence.

### 1.4 Ce que les quatre mesures disent de `i2`

Sur les deux Bastion, ou le canal a de quoi se mesurer :

| mesure | `7344d24f` | `696a9d7c` | lecture |
|---|---|---|---|
| distance dominante entre records | **72 bits (W=45) : 98,7 %** | **72 bits (W=45) : 98,4 %** | la taille reelle |
| 2e distance | 59 bits (W=32) : 0,5 % | 59 bits (W=32) : 0,5 % | reserve, cf. 1.6 |
| chainage cible TOUS, d=45 | 84,4 % (plancher median 0,31 % -> **270x**) | 86,5 % (0,27 % -> **321x**) | confirme |
| chainage cible BANDE, d=45 | **32,9 %** (+ echo a d=117 = 45+72 : 5,6 %) | **30,3 %** (echo 2,0 %) | confirme |
| chainage cible BANDE, d=1 | **0,0 %** | **0,0 %** | le faux pic est refute |
| longueur de chaine W=45 | 5,6 % atteignent 2 pas, max 2 | 2,0 %, max 2 | seule largeur qui tienne |
| longueur de chaine W=1 / 20 / 73 | **0,0 %** partout | **0,0 %** partout | refutees |

Sur les quatre KOTH, le chainage cible TOUS place le meme pic a **d=45** (54,7 % a 84,3 %, contre
un plancher median de 0,08 a 0,36 %, soit **152x a 1 122x**), et aucun autre decalage ne s'en
approche hors du faux pic a +1.

**Conclusion 0.1 : `ti=47 i2 personal-ai-data-component` = `R(45)`.** Record singleton = 72 bits.

### 1.5 Pourquoi la mesure directe manque en KOTH (et pourquoi ce n'est pas une contradiction)

En KOTH, `i2` ne rend que 10 a 64 distances entre records consecutifs, contre 4 795 en Bastion —
alors que le canal y emet 2 661 a 9 794 fois. La raison est mecanique : la mesure de distance
exige **deux records de la bande dans le MEME paquet delta**, et en KOTH `i2` emet 0,18 a 0,21
fois par paquet contre 0,36 a 0,40 en Bastion. Les records y sont isoles. La cible restreinte
tombe pour la meme raison. Ce qui reste — le chainage cible TOUS — donne le meme d=45, a 152x le
plancher.

### 1.6 Reserves honnetes

- **La sous-largeur `W=32` (59 bits) a 0,5 %** sur les deux Bastion n'est pas expliquee. Elle
  peut etre une seconde branche rare de la grammaire, ou du bruit d'ancrage. 0,5 % ne permet pas
  de trancher, et ce lot ne le tranche pas.
- **`24dbb67d` (Oddball) place son pic a d=20**, pas a 45 (65,9 %, 670 records, 9,49 % hors
  grammaire). Le canal y est quasi muet (10,9 % des records ti=47) et sa bande est bruitee : la
  mesure n'y est pas fiable. Consigne, non explique.
- La largeur est **mesuree, pas lue dans le binaire**. Tant qu'une instance Ghidra n'aura pas
  montre le deser au descripteur `+0x28`, le decoupage interne des 45 bits (§2.2) reste une
  lecture de la donnee, pas une lecture du code.

---

## 2. Ce que le canal porte (items 1.1 et 0.3)

### 2.1 Gate 0 : le curseur est au meme endroit qu'au lot C

Part des records `ti=47` dont le masque annonce `i2`, mesure de ce lot contre le lot C
(`lotC/LOTC_table_C03.tsv`) :

| film | mode | ce lot | lot C | ecart |
|---|---|---|---|---|
| `7344d24f` | Bastion | 81,15 % | 80,53 % | +0,6 |
| `696a9d7c` | Bastion | 76,29 % | 77,23 % | −0,9 |
| `01e1f945` | KOTH | 74,93 % | 74,00 % | +0,9 |
| `0a247154` | KOTH | 78,89 % | 80,08 % | −1,2 |
| `606d9844` | KOTH | 81,12 % | 80,97 % | +0,2 |
| `8076f97f` | KOTH | 77,70 % | 79,90 % | −2,2 |
| `24dbb67d` | Oddball | 10,93 % | 11,00 % | −0,1 |
| `530820e5` | CTF | 0,54 % | 0,55 % | 0,0 |
| `53ce4390` | CTF | 1,71 % | 1,72 % | 0,0 |
| `64e8adfa` | CTF | 0,50 % | 0,42 % | +0,1 |
| `000d5950` | Slayer | 1,16 % | 1,16 % | 0,0 |

**GATE 0 PASSE.** Le filtre reel/fantome tient aussi : rapport reel/fantome de 3,6x a 21,2x, et
la bande fantome est a **74,9 a 99,2 % hors grammaire** (elle ne peut pas produire de records de
cet archetype) contre 5,0 a 23,5 % pour la bande reelle — sauf sur le temoin Slayer (85,25 %),
ou la reserve du lot C sur la contamination de la bande ti=47 est confirmee telle quelle.

### 2.2 Le decoupage des 45 bits, lu dans la donnee

Profil des bits sur `7344d24f` (14 451 emissions ; chiffre = dizaine de la part a 1) :

```
position :  0 1 2..19             20..44
profil   :  0 9 000000000000000000 6365554445559455454555554
```

- bit 0 : quasi toujours 0 ;
- bit 1 : quasi toujours 1 (c'est LUI qui fabrique le faux en-tete a d=1 : la mesure de largeur
  et le profil de bits se confirment mutuellement) ;
- bits 2 a 19 : quasi toujours 0 ;
- **bits 20 a 44 : 25 bits qui varient**, c'est la donnee.

Toute la suite porte sur ce sous-champ de 25 bits (`TI47_CHAMP=20:45`). Lu comme fraction de
2^25, il vit dans `[0,3 ; 0,8]` sur les six films ou le canal parle, avec **1 % d'echantillons au
plafond** (>= 0,999) :

| film | repartition (fraction de 2^25) | au plafond |
|---|---|---|
| `7344d24f` | 0,4-0,5 : 36,2 % · 0,6-0,7 : 27,1 % · 0,5-0,6 : 22,6 % | 1,1 % |
| `696a9d7c` | 0,6-0,7 : 30,2 % · 0,5-0,6 : 28,5 % · 0,4-0,5 : 25,9 % | 1,1 % |
| `01e1f945` | 0,5-0,6 : 36,1 % · 0,4-0,5 : 28,3 % · 0,6-0,7 : 19,1 % | 0,9 % |
| `0a247154` | 0,5-0,6 : 32,6 % · 0,4-0,5 : 28,9 % · 0,6-0,7 : 26,6 % | 1,1 % |
| `606d9844` | 0,4-0,5 : 67,1 % · 0,3-0,4 : 32,4 % | 0,4 % |
| `8076f97f` | 0,4-0,5 : 58,3 % · 0,5-0,6 : 32,7 % | 0,6 % |

### 2.3 Item 1.1 — ce n'est pas une enumeration

| film | emissions | valeurs distinctes | part distincte | correlation valeur/instant |
|---|---|---|---|---|
| `7344d24f` | 14 451 | 12 760 | 88 % | r = 0,913 |
| `696a9d7c` | 12 287 | 10 928 | 89 % | r = 0,900 |
| `01e1f945` | 6 475 | 5 115 | 79 % | r = 0,851 |
| `0a247154` | 9 795 | 8 936 | 91 % | r = 0,885 |
| `606d9844` | 2 672 | 2 106 | 79 % | r = 0,646 |
| `8076f97f` | 4 486 | 4 027 | 90 % | r = 0,740 |

**Aucun alphabet.** Les 16 valeurs les plus frequentes couvrent 1,42 % des emissions sur
`7344d24f`. La question 1.3 du plan — « chaque valeur corrèle-t-elle a UN type d'evenement » —
**n'a pas de sujet** : il n'y a pas de valeurs, il y a un continuum.

### 2.4 Item 1.1 (suite) — la cadence, qui tranche le lot

Par slot, sur `7344d24f` :

| slot | emissions | ecart median | p90 | valeurs distinctes | variation mediane | variation p99 |
|---|---|---|---|---|---|---|
| 1308 | 3 103 | **50 ms** | 51 ms | 2 893 | 19 | 14 196 689 |
| 1304 | 3 035 | **50 ms** | 51 ms | 2 505 | 8 | 14 251 477 |
| 1306 | 2 687 | **50 ms** | 51 ms | 2 443 | 12 | 14 366 584 |
| 1300 | 2 662 | **50 ms** | 51 ms | 2 324 | 13 | 13 203 182 |
| 1302 | 1 165 | **50 ms** | 51 ms | 961 | 10 | 17 781 733 |
| 1298 | 985 | **50 ms** | 51 ms | 921 | 17 | 17 537 911 |
| 1310 | 518 | **50 ms** | 51 ms | 476 | 16 | 16 441 461 |
| 1312 | 296 | **50 ms** | 51 ms | 273 | 15 | 19 626 834 |

**20 Hz, sur tous les slots, sur tous les films.** La valeur derive imperceptiblement — 8 a 19
quanta sur 33 554 432, soit 4 x 10^-7 de pleine echelle par pas de 50 ms — et fait, dans 1 % des
cas, un aller-retour au plafond (variation p99 : 13 a 20 millions, soit 0,4 a 0,6 de pleine
echelle).

**Ce n'est pas un canal d'annonces.** Le mot « annonces » du lot C designait les annonces DU
MASQUE (le composant est annonce dans le masque d'un record delta), pas des annonces de jeu — ce
lot le mesure et le dit, pour que la confusion ne se rejoue pas.

### 2.5 L'archetype est par JOUEUR

| film | mode | zones du mode | slots ti=47 | joueurs |
|---|---|---|---|---|
| `7344d24f` / `696a9d7c` | Bastion | 3 | 8 / 8 | 8 / 8 |
| `01e1f945` / `0a247154` | KOTH | 1 colline | 9 / 8 | 8 / 8 |
| `606d9844` / `8076f97f` | KOTH | 1 colline | 9 / 10 | 8 / 8 |
| `24dbb67d` | Oddball | 0 | 8 | 8 |
| `530820e5` / `53ce4390` / `64e8adfa` | CTF | 0 | 8 / 8 / 8 | 8 / 8 / 8 |
| `000d5950` | Slayer | 0 | 8 | 8 |

Le nombre de slots suit le nombre de JOUEURS (8, plus 0 a 2 entites supplementaires), jamais le
nombre de zones. Le nom du composant dit la meme chose : la « personal AI » de Halo Infinite est
un objet par joueur.

---

## 3. La confrontation aux oracles (items 1.2 a 1.4)

### 3.1 Le montage

- **Oracle** : l'artefact de rejeu (`data/cache/replays/halo_infinite/<short8>.json`, schema 18),
  cale sur l'horloge du film par son propre champ `originMs` (`instantMS = originMs + frame x 100`,
  la formule de `scoreClock.frameOf` en sens inverse). Quatre familles derivees de `zoneStates` :
  `prise` (bascule de proprietaire), `perte` (neutralisation), `colline_debut` / `colline_fin`
  (l'intervalle `active` de KOTH), `jauge_debut` / `jauge_fin` (les bornes de chaque rampe de la
  jauge en direct — le seul oracle du corpus qui approche la « contestation »).
- **Ce qu'on confronte** : pas l'instant d'emission (a 20 Hz il couvre tout le match et ne
  mesurerait que la densite des oracles), mais le **SAUT** — une variation au-dela du percentile
  99 des variations DU SLOT, donc un seuil mesure sur le canal et non ecrit d'avance.
- **Temoin** : la meme suite de sauts rejouee avec un decalage d'un tiers de match modulo la
  duree. Meme nombre, meme loi d'espacement, alignement detruit. C'est le temoin « meme loi hors
  evenements » que le plan demande.

### 3.2 Item 1.2 — resultats

| film | oracles | sauts | sauts dans +/-2 s | temoin decale | **exces** |
|---|---|---|---|---|---|
| `7344d24f` | 247 (104+104 jauge, 39 prises) | 148 | 58,8 % | 57,4 % | **1,02x** |
| `696a9d7c` | 237 (100+100 jauge, 37 prises) | 127 | 64,6 % | 55,1 % | **1,17x** |
| `01e1f945` | 10 (5 collines) | 68 | 1,5 % | 4,4 % | **0,33x** |
| `0a247154` | 12 (6 collines) | 102 | 3,9 % | 1,0 % | **4,00x** (4 sauts contre 1) |
| `606d9844` | 6 (3 collines) | 30 | 16,7 % | 6,7 % | **2,50x** (5 sauts contre 2) |
| `8076f97f` | 6 (3 collines) | 48 | 0,0 % | 0,0 % | **0 / 0** |

Distances medianes a l'oracle le plus proche : **1 388 ms** (p90 9 399) sur `7344d24f`,
**1 288 ms** (p90 5 857) sur `696a9d7c` — mais le temoin decale rend 1 567 et 1 856 ms. En KOTH,
24 563 / 38 019 / 15 247 / 30 045 ms, temoins 23 050 / 39 066 / 22 286 / 25 475 ms.

Les emissions elles-memes (et non les sauts) : exces **1,11x** / **1,04x** en Bastion,
0,91x a 1,90x en KOTH.

**Aucun de ces chiffres ne sort du hasard.** Les deux exces > 2 reposent sur 4 et 5 sauts.

Note de lecture, pour ne pas contredire le lot C par accident : le lot C mesurait une densite
**en fenetre / hors fenetre** et trouvait 1,60-1,61x pour ce canal en modes a zones. Ce n'est pas
la meme statistique que celle d'ici (part des sauts a moins de 2 s d'un evenement, contre la meme
part sur la suite decalee). Le lot C n'avait pas de temoin de meme loi ; avec ce temoin, l'exces
retombe a 1,02-1,17x. Les deux mesures ne s'opposent pas : la seconde explique la premiere par la
densite des evenements de zone, qui couvrent deja 55 a 63 % du match en fenetre +/-2 s.

### 3.3 Items 1.3 et 1.4 — la matrice, et les classes sans oracle

Faute d'enumeration (§2.3), la seule partition que la donnee autorise est l'**amplitude du saut
par decade**. Part des sauts de la classe a moins de 2 s d'un evenement du type :

| film | classe | sauts | jauge_debut | jauge_fin | prise | colline | **sans oracle (1.4)** |
|---|---|---|---|---|---|---|---|
| `7344d24f` | 10^7 | 148 | 57 % | 45 % | 25 % | — | **41 %** |
| `696a9d7c` | 10^7 | 127 | 62 % | 42 % | 21 % | — | **35 %** |
| `01e1f945` | 10^7 | 68 | — | — | — | 1 % | **99 %** |
| `0a247154` | 10^7 | 102 | — | — | — | 4 % | **96 %** |
| `606d9844` | 10^2 / 10^3 / 10^6 / 10^7 | 7 / 6 / 2 / 15 | — | — | — | 57 / 0 / 0 / 7 % | **43 / 100 / 100 / 93 %** |
| `8076f97f` | 10^3 / 10^5 / 10^6 / 10^7 | 3 / 1 / 2 / 42 | — | — | — | 0 % | **100 %** |

**Item 1.3 : aucune separation de classes.** Une seule classe d'amplitude porte 100 % des sauts
sur cinq films sur six, et ses taux d'appariement sont ceux du temoin.

**Item 1.4 : les classes « sans oracle » ne sont pas des candidates « contestation ».** Sur les
Bastion, 35-41 % des sauts n'ont aucun oracle a moins de 2 s — mais le temoin en a autant, donc
ce residu ne designe rien. En KOTH, 93-100 % sans oracle : le canal ignore la colline. **Aucune
verification visuelle en Theater n'est donc proposee** : il n'y a pas de candidat a montrer, et
envoyer l'utilisateur regarder du bruit serait lui faire perdre son temps.

---

## 4. Statut de chaque item

| item | statut | justification |
|---|---|---|
| 0.1 porter la deser de `ti=47 i2` | `[x]` | `R(45)`, **mesuree** (§1) et non lue : le descripteur `+0x28` exige Ghidra, `list_instances` rend une liste vide. La deser vit dans l'instrument, pas dans `traverse.go` (§6). |
| 0.2 instruments gates `TI47_FILM` | `[x]` | 5 fichiers de test, `SKIP` verifie sans la variable, `CGO_ENABLED=0`, lecture seule, TSV en sortie. |
| 0.3 recensement >= 6 films / >= 3 modes | `[x]` | **11 films / 5 modes**, filtre reel/fantome et vies (slot,gen) publies (§2.1). |
| Gate 0 | **PASSE** | densites a 2,2 points au pire du recensement du lot C (§2.1). |
| 1.1 cardinalite | `[x]` | 79-91 % de valeurs distinctes, aucun alphabet ; profil de bits et echelle publies (§2.2-2.4). |
| 1.2 alignement temporel | `[x]` | medianes, p90, fenetres et temoin « meme loi decalee » pour six films (§3.2). |
| 1.3 separation des classes | `[x]` | matrice publiee ; aucune separation — et la question presuppose une enumeration que la donnee n'a pas (§3.3). |
| 1.4 classes sans oracle | `[x]` | 35-41 % (Bastion) et 93-100 % (KOTH) sans oracle, mais au niveau du temoin : aucun candidat date, donc aucune verification visuelle proposee (§3.3). |
| 1.5 verdict | `[x]` | **NEGATIF mesure** (§0). |
| Gate 1 | **PASSE** | chaque affirmation du §0 renvoie a une commande du §5. |

---

## 5. Reproduire

Depuis `apps/go-api`, dans le worktree, avec un GOCACHE prive
(`GOCACHE=<worktree>/.gocache`, jamais commite) :

```bash
# Phase 0 — largeur et recensement (un film ; TI47_RUN nomme les largeurs a departager)
CGO_ENABLED=0 \
  TI47_FILM=<principal>/data/cache/film_chunks/7344d24f \
  TI47_CACHE=<principal>/data/cache TI47_SHORT=7344d24f \
  TI47_RUN="1,20,45,73" \
  TI47_TSV=<worktree>/.ai/V7.5/replay2d/registre_film/lotF2 \
  go test ./internal/analysis/filmdec/ -run TestTI47Annonces -v -timeout 60m

# Phase 1 — valeurs et oracles (largeur du gate 0, sous-champ du profil de bits)
CGO_ENABLED=0 \
  TI47_FILM=<principal>/data/cache/film_chunks/7344d24f \
  TI47_CACHE=<principal>/data/cache TI47_SHORT=7344d24f \
  TI47_WIDTH=45 TI47_CHAMP="20:45" \
  go test ./internal/analysis/filmdec/ -run TestTI47Annonces -v -timeout 60m
```

Sans `TI47_FILM`, le test est `SKIP` — CI comprise. Sans `TI47_WIDTH`, la phase 1 ne s'execute
pas : le gate 0 doit trancher la largeur d'abord.

Cout machine (D17) : recensement des images-cle 4 a 17 s par film, passe delta 0,7 a 2,2 s.
Corpus complet des 11 films : environ 2 min par phase.

Fichiers : `internal/analysis/filmdec/ti47_annonces_{scan,largeur,valeurs,oracle,}_test.go`.
Journaux : `lotF2/LOTF2_phase0.log`, `lotF2/LOTF2_phase1.log`.
Histogrammes complets : `lotF2/<short8>_ti47_chainage.tsv` (une ligne par decalage, six taux).

---

## 6. Exploitation minimale — il n'y en a pas, et pourquoi

Le plan demandait, en cas de verdict positif, une proposition d'exploitation minimale. **Le
verdict est negatif : il n'y a rien a exploiter cote produit.** Ce qui suit remplace cette
section, parce qu'un lot qui s'arrete doit dire ce qu'il laisse.

**Ce qui est acquis et ne sera pas a refaire :**

1. **`ti=47` est entierement traversable.** Ses trois composants ont maintenant une largeur
   (`i0` porte, `i1` = R(24), `i2` = R(45)). Le jour ou un lot en aura besoin, le portage en
   production tient en une ligne de `traverse.go` plus un `consumeBits(45)`.
2. **La methode de mesure de largeur sans binaire est outillee.** Les quatre mesures et leur
   temoin positif intra-archetype se rejouent sur n'importe quel composant non porte en changeant
   une constante de nom. C'est la reponse au fait que la recette officielle exige Ghidra.

**Ce qui N'EST PAS fait, deliberement :**

- **Le portage en production n'est pas fait.** `traverse.go` n'est pas touche. Porter une largeur
  MESUREE (et non lue au descripteur) changerait le decodage de tous les films pour un composant
  dont personne n'a besoin aujourd'hui, et dont la sous-largeur `W=32` a 0,5 % (§1.6) reste
  inexpliquee. A faire le jour ou un lot en a l'usage, et de preference apres confirmation
  Ghidra.
- **Le sens physique des 25 bits n'est pas etabli.** Une valeur par joueur, a 20 Hz, sur
  `[0,3 ; 0,8]` de pleine echelle avec 1 % au plafond, qui derive de 4 x 10^-7 par pas : c'est le
  profil d'une grandeur normalisee, pas d'un identifiant. Le nommer demanderait le binaire.

---

## 7. Decouvertes (hors perimetre — notees, NON traitees)

1. **Les objets de `ti=47` emettent par BLOCS DE HUIT.** Le chainage de `i1` rend
   87,4 / 74,9 / 62,4 / 49,9 % aux decalages 24 / 75 / 126 / 177, soit exactement 7/8, 6/8, 5/8,
   4/8, a la troisieme decimale pres sur quatre films differents. C'est une propriete de
   serialisation par bloc, jamais consignee ; elle donnerait un ancrage tres sur pour n'importe
   quel balayage de cet archetype.
2. **Une charge utile peut contenir un motif qui passe le test d'en-tete de record.** Le bit 1 de
   `i2` fabrique un « en-tete valide » a +1 bit dans 70 a 99 % des cas. Tout futur balayage qui
   conclut une largeur sur un seul histogramme de chainage est expose au meme piege : la cible
   restreinte a la bande et la longueur de chaine sont les deux garde-fous qui l'ont refute.
3. **`24dbb67d` (Oddball) place son pic de chainage a d=20**, pas a 45. Non explique (670 records,
   bande a 9,5 % hors grammaire). Consigne.
4. **La sous-largeur `W=32` (59 bits) a 0,5 %** sur les deux Bastion : seconde branche de la
   grammaire ou bruit d'ancrage, non tranche.
5. **Le lot C-ter avait deja mesure que le canal de jauge de KOTH n'est pas la progression de
   garde** (`coverage.zones.gaugePoints = 0` en KOTH). Ce lot le retrouve par un autre chemin :
   les six films KOTH n'ont que 6 a 12 evenements de zone exploitables, ce qui limite
   structurellement toute mesure d'alignement en KOTH — a savoir avant de lancer un lot qui
   dependrait d'un oracle KOTH.
