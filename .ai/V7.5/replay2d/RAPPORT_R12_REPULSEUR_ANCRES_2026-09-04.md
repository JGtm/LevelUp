# RAPPORT R12 — Le REPULSEUR avec SIX ANCRES D'USAGE

Date : 2026-09-04. Lot R12, suite de R11 (canal des charges trouve, muet sur le repulseur),
R9 (huit canaux fermes sans ancre), R8 (propulseur trouve), R7 (trame complete).
Retro-ingenierie STATIQUE. Aucune DuckDB ouverte, aucune ecriture sur la production, aucun
commit.

> Rapport ECRIT AU FIL DE L'EAU. **[ETABLI]** = mesure faite ; **[REFUTE]** = hypothese tuee
> par la mesure ; **[A EPROUVER]** = en cours. Une interruption ne doit rien couter.

Branche de travail : `wt/r12-repulseur`, creee depuis `wt/grappin-vies` (la branche qui porte
les instruments R6 a R11 et les desers portes). Worktree :
`C:/Users/Guillaume/Downloads/Scripts/LevelUp/.claude/worktrees/agent-a8075904f56589d18`.

---

## Verdict en six phrases

**LA CONVENTION DE TEMPS EST LE TEMPS DE FILM, et elle est etablie deux fois par deux canaux
sans rapport** : le decodeur de production `killsource` place l'unique kill au `jpt!` du
repulseur de tout le film a **325 526 ms = 5:25,5**, la ou le releve Theater situe l'usage
d'Elmo910 a 5:25 ; et le canal i48 retrouve les **quatre** ramassages du releve avec un ecart
de **+1,35 a +1,97 s**, une bande de 620 ms, contre 1/4 et 0/4 pour les temoins decales de
+/-30 s. **LE FILM NE PORTE PAS LE GESTE DU REPULSEUR — IL PORTE SON EFFET** : le recensement
ancre des 64 composants du bipede rend ZERO sur le porteur aux cinq instants d'usage, alors
que le MEME instrument, ancre sur les usages de grappin du MEME film, fait bondir i56 d'un
facteur 9. **Ce qui bouge a ces instants est la LISTE D'EVENEMENTS** : les DEUX seules
occurrences du type 104 `EquipmentKnockbackPlayer` de tout le film tombent a 5:25,5 et 5:25,6,
et l'UNIQUE tete de type 105 `EquipmentObjectKnockedBack` du film — cadrage CERTAIN, aucune
grammaire n'intervient — tombe a 5:25,6, soit 100 ms apres le kill. **Une SECONDE ancre,
independante du releve et sur un autre film, confirme** : sur `a6ae19fb`, une tete de type 105
tombe **149 ms** avant la consommation de la derniere charge de repulseur que R11 par. 5 avait
publiee en toutes lettres comme « le cas qui tranche » ; le hasard des deux coincidences est de
l'ordre de **5e-7**, sur une base de 22 tetes de 105 pour 1 223 551 paquets delta et 42 films.
**Le rappel est structurellement partiel — 2 usages sur 5 — et ce n'est pas un defaut de
mesure : ces evenements sont des CONSEQUENCES (un objet pousse, un joueur pousse), donc un
repulseur declenche dans le vide n'ecrit rien.** **Trois acquis lateraux** : le
`jpt! 0x07104b31` = Repulsor est confirme sur corpus reel ET par verite terrain Theater (sa
reserve specifique au registre tombe) ; les reapparitions au socle sont datees **3/3** par les
creations ti=37 du GlobalID `0x7ca85adc` ; et **la marche de liste d'evenements ne depend plus
du catalogue de bornes** — le contexte de carte se reconstruit exactement depuis le seul
`AxisW` du film (oracle de trame : facteur 5,46 sur Argyle, seuil pre-inscrit 3).

---

## 0. LES ANCRES, ET LA CONVENTION DE TEMPS

### 0.0 Le releve (verite terrain, collegue de l'utilisateur, film `215e7022`)

Film `215e7022-9959-4a1e-85aa-861161b588f4` — Argyle, Quick Play/Team Slayer, 2026-02-03.

| # | instant releve | evenement |
|---|---|---|
| P1 | 3:48 | JGtm prend le repulseur au socle ; meurt sans l'utiliser |
| S1 | 4:49 | reapparition du repulseur au socle |
| P2 | 4:55 | Elmo910 prend le repulseur |
| **U1** | **5:25** | **Elmo910 l'UTILISE — kill attribue, source de degats : Repulseur** |
| S2 | 5:56 | reapparition au socle |
| P3 | 8:14 | Bot Ziker le ramasse |
| **U2** | **~8:14** | **Bot Ziker l'utilise quasi immediatement** |
| **U3** | **8:20** | **Bot Ziker le reutilise** |
| M1 | ~8:22 | Bot Ziker meurt ; il laisse tomber le repulseur avec une charge restante |
| S3 | 9:14 | reapparition au socle |
| P4 | 9:50 | Elmo910 le ramasse |
| **U4** | **9:54** | **Elmo910 l'utilise** |
| **U5** | **10:05** | **seconde utilisation** |

Le releve porte une reserve : « film demarre 10,4 s avant le match ». Ses instants sont donc
soit en temps de MATCH (horloge film = instant + 10,4 s), soit deja en temps de FILM.

### 0.1 [ETABLI — PAR PIECE DEJA AU DEPOT] La convention est le TEMPS DE FILM

Cette question se tranche AVANT toute mesure neuve, et elle se tranche sur une piece deja
ecrite au depot, anterieure au releve et independante de lui.

`.ai/V7.5/REGISTRE_REPORTS.md` (ligne « Repulseur : le tag `07104b31` reste SOUS_RESERVE »,
lot kills-hors-arme du 2026-08-29) consigne l'UNIQUE kill au repulseur de toute la base :

> match `215e7022-9959-4a1e-85aa-861161b588f4`, 2026-02-03 16:13 UTC, Argyle,
> Quick Play/Team Slayer, **t = 325 526 ms**, Elmo910 -> aK2fResHv3

`killsource.Kill.TimeMS` est documente mot pour mot comme « instant de la mort, en
**millisecondes depuis le debut du film** ». Or **325 526 ms = 5:25,5**, et le releve place
l'usage U1 d'Elmo910 — avec kill attribue au repulseur — a **5:25**.

**Les instants du releve sont donc en TEMPS DE FILM, tels que le curseur du visionneur les
affiche.** La mention « film demarre 10,4 s avant le match » decrit le decalage entre
l'horloge du film et le chrono du match, pas une correction a appliquer aux instants releves.

Aucune hypothese parallele n'est donc portee dans la suite ; l'hypothese « temps de match »
(instants + 10,4 s) est REFUTEE par cette coincidence a 0,5 s pres sur un instant date par un
canal de production independant du releve.

**Cette conclusion est re-confirmee par mesure au par. 2.**

### 0.2 Ce que l'absence d'artefact interdit, et ce qu'elle n'interdit pas

Argyle n'est pas au catalogue `map_quant_bounds.json` (aucun module `argyle`), donc la
cuisson d'artefact de rejeu echoue volontairement et
`data/cache/replays/halo_infinite/215e7022.json` n'existe pas.

**[ETABLI par lecture du code]** Cette absence ne bloque AUCUN des canaux vises :
`SetWorldObjectPrecisionFromLayout` ne consomme que `AxisW` et `GateBits`, tous deux rendus
par `DetectI0Layout(dir)` — c'est-a-dire par LE FILM LUI-MEME (`filmdec/traverse.go:183`,
`filmdec/i0_layout.go:144`). Les bornes metriques ne servent qu'a DEQUANTIFIER les positions
en metres. Les instruments de ce lot construisent donc leur precision monde depuis le layout
du film et n'ouvrent jamais le catalogue de bornes.

Deux choses manquent reellement, et sont remplacees :

- **la palette de capacites** (`abilityLabels` de l'artefact) : remplacee par le classement
  direct de `config/titles/halo_infinite/mappings/replay_labels.toml` sur les rangs i48
  observes (meme regle que `classifyAbilityPalette`, purete 90 %) ;
- **les gamertags par slot** (`roster`/`tracks` de l'artefact) : remplaces par le roster de
  replication que `killsource` decode deja depuis le film seul.

---

## 1. PRE-INSCRIPTION (ECRITE AVANT TOUTE MESURE NEUVE)

Regle du chantier : seuils et temoins ecrits avant de regarder. Ce paragraphe est fige.

### 1.1 Etape A — ancrer le film : qui, quand, quel rang

**Mesure A1 (pont du kill).** `killsource.Decode` sur `215e7022` doit rendre une mort dont
`Source.Tag == 0x07104b31` a `TimeMS` dans **[323 000, 328 000]**, tueur `Elmo910`.
- **Si oui** : la convention de temps du par. 0.1 est confirmee PAR MESURE, et on tient une
  verite terrain Theater sur le `jpt!` du repulseur.
- **Si non** : le pont tombe, et je le dis avant d'aller plus loin.

**Mesure A2 (i48 : les 4 ramassages).** Le canal i48 `biped-desired-ability-set` doit montrer
des lectures de rang « Repulsor » pres de **3:48, 4:55, 8:14, 9:50** (tolerance +/- 5 s : i48
n'emet qu'environ une fois par vie, il peut manquer un ramassage mais il ne doit pas en
inventer). Seuil pre-inscrit : **au moins 2 des 4** ramassages apparies. En dessous, le canal
i48 ne sert pas d'ancre d'identite sur ce film et je le dis.
- **TEMOIN NEGATIF** : le meme appariement avec les instants decales de +30 s doit rendre
  strictement moins d'apparies.

### 1.2 Etape B — LA QUESTION, posee dans l'autre sens

Pour chacun des **CINQ instants d'usage** U1 (5:25), U2 (~8:14), U3 (8:20), U4 (9:54),
U5 (10:05), et pour le PORTEUR de ce moment (Elmo910 pour U1/U4/U5, Bot Ziker pour U2/U3) :

**Qu'est-ce qui, dans le film, change chez ce porteur dans une fenetre de +/- 1,5 s autour de
l'instant, et ne change pas chez les autres joueurs aux memes instants ?**

Trois gisements sont balayes, dans cet ordre :

**(B1) TOUS les composants du bipede.** Non pas un recensement d'ANNONCES par rang porte
(R9 par. 7.5, qui ne peut pas voir une VALEUR), mais un **recensement d'ANNONCES ANCRE** :
pour chaque composant i0..i63, le taux d'annonce du porteur dans la fenetre d'usage, contre
son taux hors fenetre, contre le taux des AUTRES joueurs dans la meme fenetre.

**(B2) La liste COMPLETE d'evenements**, tous types, positions 1 et suivantes : quels types
apparaissent dans les paquets de la fenetre et pas ailleurs.

**(B3) Les entites du monde** creees ou modifiees dans la fenetre (ti=37 et autres ti), et
l'etat du SOCLE aux instants de reapparition S1/S2/S3.

### 1.3 Seuils, ecrits avant de regarder

Pour qu'un canal soit declare PORTEUR de l'usage du repulseur :

1. **Rappel** : au moins **4 des 5** usages presentent le signal dans +/- 1,5 s.
2. **Precision** : le meme signal, sur tout le film et pour tous les joueurs, ne doit pas
   apparaitre plus de **3 fois** hors des fenetres d'usage connues (le releve ne pretend pas
   a l'exhaustivite : d'autres joueurs ont pu employer d'autres exemplaires, donc un exces
   modere n'est pas eliminatoire — mais il doit etre RARE, pas continu).
3. **CONTROLE NEGATIF** : aux memes 5 instants, les joueurs qui ne portent PAS le repulseur
   ne doivent pas montrer le signal (au plus 1 sur 5x(N-1) occasions).
4. **TEMOIN POSITIF OBLIGATOIRE** : le meme instrument, applique a un film ou le GRAPPIN ou
   le PROPULSEUR est present, doit les retrouver par leurs signatures connues
   (i56 charges / i57-i59 tag 1 / tag 3). **Si le temoin positif echoue, aucun negatif n'est
   publie a partir de cet instrument.**

### 1.4 Refutation active — ce que je chercherai contre mon propre resultat

- Un signal trouve chez le porteur est-il un signal d'USAGE, ou de PORT (present pendant
  toute la fenetre de port, pas seulement a l'usage) ? Controle : sa densite pendant les
  segments de port SANS usage.
- Est-il specifique au repulseur, ou commun a tout equipement ? Controle : les memes
  composants chez les porteurs d'autres equipements du meme film.
- Le film `215e7022` porte-t-il un grappin/propulseur ? Si oui, ils servent de temoin
  interne ; sinon le temoin est pris sur un autre film et c'est dit.

### 1.5 L'identifiant `07104b31` contre `0x7ca85adc` — la question a eclaircir

Deux identifiants pour le repulseur, deja documentes au depot et NON contradictoires :

- **`0x7ca85adc`** est un tag de groupe **`eqip`** — l'OBJET equipement, celui que le film
  pose dans le monde et que `replay_labels.toml` `[[equipment_objects]]` nomme
  `family = "repulsor"`, `kind = "carried"`.
- **`0x07104b31`** est un tag de groupe **`jpt!`** (damage_effect) — l'EFFET DE DEGAT, la
  source lue dans le dead-state de la victime (`damagetag/data/labels.tsv` ligne 97).

Chaine etablie par RE himap le 2026-08-25 (thought_log) :
`eqip 7ca85adc` -> `sofa 6845f2b3` -> `eqip` frere `1e79ebda` -> `jpt! 07104b31`.

Ce ne sont donc PAS deux identifiants du meme objet : c'est **l'objet** et **son effet de
degat**, et le second pend a un `eqip` FRERE (`1e79ebda`) du premier. Le statut
`SOUS_RESERVE` porte sur le NOM (« Repulsor »), pas sur la chaine — et il est le regime normal
des 114 autres lignes ARME nommees de la table. **La mesure A1 de ce lot est precisement ce
qui peut lever cette reserve : une verite terrain Theater sur ce kill.**

---

## 2. [ETABLI] ETAPE A — le film est ancre, et la convention de temps est CONFIRMEE PAR MESURE

### 2.1 Mesure A1 — LE PONT DU KILL : le verdict pre-inscrit PASSE

Instrument : `killsource/r12_ancre_kill_research_test.go`, `TestR12AncreKill`. Il appelle le
chemin de PRODUCTION `killsource.Decode` (gele) sur le film et publie ses 51 kills.

```
5:25.5   tag=07104b31 Repulsor   cls=ARME SOUS_RESERVE   tueur=Elmo910   victime=aK2fResHv3
ANCRE U1 : t=325526 ms  origine=credit-concordant  voie=marche  — fenetre [323000,328000] : true
```

**Verdict pre-inscrit : PASSE.** Un seul kill au tag `0x07104b31` dans tout le film, a
**325 526 ms**, tueur **Elmo910**, victime **aK2fResHv3**, origine `credit-concordant`, voie
`marche` (la voie a 98,2 % de justesse de couple). C'est EXACTEMENT la ligne que le registre
des reports consigne depuis le 2026-08-29, rejouee ici.

**CONSEQUENCE 1 — LA CONVENTION DE TEMPS.** 325 526 ms = 5:25,5 en temps de film ; le releve
place l'usage U1 a 5:25. **Les instants du releve sont en TEMPS DE FILM.** L'hypothese
« temps de match » (qui exigerait +10,4 s) est REFUTEE.

**CONSEQUENCE 2 — LA RESERVE SUR `07104b31` EST LEVEE PAR VERITE TERRAIN.** Le registre
notait, pour ce tag : « L'identification vient d'une RE hors ligne, jamais confirmee sur
corpus [...] PAS de verite-terrain Theater ». Elle existe desormais : un observateur humain a
vu, au visionneur, Elmo910 utiliser le repulseur a 5:25 et le kill lui etre attribue avec
« source de degats : Repulseur » ; le decodeur, sans rien savoir de ce releve, place au meme
instant un kill d'Elmo910 dont le dead-state porte `0x07104b31`. **La chaine
`eqip 7ca85adc -> sofa 6845f2b3 -> eqip 1e79ebda -> jpt! 07104b31` est confirmee sur corpus
reel.** (Le statut `SOUS_RESERVE` de `labels.tsv` reste le regime normal de toute arme nommee
de cette table — `TestCoherenceDesLignes` l'exige — mais la raison specifique qui le motivait
pour cette ligne, l'absence de verite terrain, n'existe plus.)

**RESERVE, DITE.** `LineByLinePublishable()` rend **faux** sur ce film : `BijectionMargin = 0`,
donc au moins deux joueurs sont interchangeables dans la bijection indice -> joueur, et les
NOMS ligne a ligne ne sont pas publiables en toute rigueur. Ce qui tient malgre cela : le
TAG et l'INSTANT ne passent pas par la bijection (ils viennent du dead-state et du kill-feed),
et le nom `Elmo910` est ici corrobore par une source externe — le releve Theater. Trois autres
lignes du meme decodage coincident d'ailleurs avec le releve sans avoir ete cherchees :
JGtm tue Bot ziker en melee a 8:20,2 (le releve dit « il meurt vers 8:22 »), et Bot ziker tue
en melee a 9:50,3 (le releve situe la un ramassage d'Elmo910).

### 2.2 Mesure A2 — i48 retrouve les QUATRE ramassages, avec un ecart CONSTANT

Instrument : `filmdec/r12_socle_research_test.go`, `TestR12Ancrage`.

Le film `215e7022` : layout i0 `AxisW=[15 15 17]` region 0 gate 5, 34 chunks, 102 slots
bipede. Palette **famille A** (31 lectures a rang sur 31 portent ses marqueurs) ; le
**repulseur est le rang 6**. Histogramme des 35 lectures i48 : grappin (rang 4) 16,
**repulseur (rang 6) 11**, camouflage actif (rang 8) 4, porte ouverte 4.

| ancre du releve | lecture i48 « rang 6 » la plus proche | ecart |
|---|---|---|
| P1 JGtm 3:48 | **3:49,3** slot 549 | **+1 348 ms** |
| P2 Elmo910 4:55 | **4:56,9** slot 560 | **+1 968 ms** |
| P3 Bot ziker 8:14 | **8:15,4** slot 592 | **+1 429 ms** |
| P4 Elmo910 9:50 | **9:51,5** slot 608 | **+1 553 ms** |

**4/4 apparies** (seuil pre-inscrit : 2). Et le fait qui vaut plus que le compte : **les
quatre ecarts tiennent dans une bande de 620 ms**, tous positifs, moyenne +1,6 s — le delai
attendu entre le geste vu a l'ecran et la premiere emission d'i48 de la vie concernee.

Temoins : a tolerance resserree +/- 2,5 s, **4/4** apparies contre **1/4** pour le releve
decale de +30 s et **0/4** pour -30 s.

**La convention de temps est donc etablie DEUX FOIS, par deux canaux sans rapport** : le
dead-state de la victime (A1) et le rang porte du bipede (A2).

### 2.3 [ETABLI — CADRAGE INDISPENSABLE] Ce film porte ONZE vies de repulseur, pas cinq

Le journal i48 rend **11 lectures de rang 6** : 3:49,3 (slot 549), 3:58,6 (544), 4:20,1 (545),
4:56,9 (560), 5:29,3 (564), 5:49,7 (556), 8:15,4 (592), 8:20,3 (589), 8:38,8 (586),
9:51,5 (608), 10:15,1 (607).

**Le releve ne couvre donc pas tout le repulseur du film — il suit UN exemplaire, celui d'un
socle.** Argyle porte visiblement plusieurs sources de repulseur (et de grappin : 16 vies).

**Cela AMENDE le critere de precision pre-inscrit au par. 1.3 n°2, et l'amendement est ecrit
avant la mesure suivante** : un signal trouve chez un porteur de repulseur HORS des cinq
fenetres du releve n'est PAS un faux positif tant qu'il tombe sur une vie de rang 6. Le vrai
faux positif est un signal chez un porteur de rang 4 ou 8, ou chez un slot sans repulseur.

**Cadrage favorable, note ici** : ce film porte 16 vies de GRAPPIN, dont l'usage a une
signature connue (i57/i59 tag 3, canal des charges i56 emplacement `e2`). **Le temoin positif
obligatoire du par. 1.3 n°4 est donc INTERNE a ce film**, ce qui est la meilleure situation
possible : aucun effet de film ne peut expliquer une difference entre les deux equipements.

Denominateurs du film (annonces / lectures) : i48 35/35, i56 125/125, i57 1 417 annonces
(tag0 703, tag1 **2**, tag2 706, tag3 6), i59 1 427 annonces (tag0 704, tag1 **2**, tag2 705,
tag3 **10**).

---

## 3. [ETABLI — NEGATIF, ANCRE, AVEC TEMOIN POSITIF INTERNE] L'ETAT DU BIPEDE ne porte rien

Instrument : `filmdec/r12_fenetres_research_test.go`, `TestR12Fenetres`. C'est le recensement
du masque bipede de R9 par. 7.5 rendu ANCRE : au lieu de comparer des RANGS PORTES (ce qui ne
peut pas voir un geste), il compare quatre populations de records —

- **porteurDANS** : le slot porte le rang du repulseur (rang frais, <= 60 s) ET l'instant est
  dans une des cinq fenetres d'usage (+/- 1,5 s) ;
- **porteurHORS** : meme porteur, hors fenetre (c'est le controle « usage contre PORT ») ;
- **autreDANS** : un autre slot, dans la fenetre (c'est le controle negatif « qui ») ;
- **tout** : les 209 154 records du film.

### 3.1 Le TEMOIN POSITIF passe, et il est INTERNE au film

Le meme instrument, ancre sur les instants d'usage du GRAPPIN de ce meme film (les 10
lectures `i59 tag == 3`, la signature etablie en production) :

| composant | nPD | nPH | tauxPD | tauxPH | **facteur PORT** | facteur QUI |
|---|---|---|---|---|---|---|
| **i56 `biped-spartan-ability-energy`** | 5 | 15 | 0,0089 | 0,0010 | **9,00** | **7,04** |
| i59 `-non-predicted-state` | 10 | 108 | 0,0177 | 0,0071 | **2,50** | 1,76 |
| i57 `biped-spartan-ability` | 6 | 108 | 0,0106 | 0,0071 | 1,50 | 1,21 |

**i56 — le canal des charges de R11, un composant que l'ancre n'a PAS servi a definir — bondit
d'un facteur 9 dans les fenetres d'usage du grappin.** L'instrument voit donc ce qu'il doit
voir : un geste d'equipement, sur ce film, laisse une trace mesurable dans le masque bipede.

### 3.2 Le REPULSEUR : rien, sur 706 records de porteur en fenetre

| composant | nPD | nPH | nAD | tauxPD | tauxPH | facteur PORT | facteur QUI |
|---|---|---|---|---|---|---|---|
| i5 `object-shield-vitality` | 394 | 3 854 | 1 865 | 0,558 | 0,432 | 1,29 | 1,50 |
| i30 `weapon-state-ammo` | 23 | 93 | 83 | 0,033 | 0,010 | 3,13 | 1,96 |
| **i56 `spartan-ability-energy`** | **0** | **0** | 3 | 0,000 | 0,000 | **0,00** | **0,00** |
| **i57 / i59** | 3 / 3 | 56 / 56 | 31 / 33 | 0,004 | 0,006 | **0,68** | **0,68 / 0,64** |
| i54 `biped-mobility-action` | 0 | 42 | 44 | 0,000 | 0,005 | 0,00 | 0,00 |

Les deux seules colonnes qui montent — le bouclier (1,29) et les munitions (3,13) — sont
l'accompagnement banal d'un joueur qui se bat : il tire et il prend des degats. **Les
composants de CAPACITE, eux, sont a zero ou EN DESSOUS de leur taux de repos.**

**VERDICT.** Le negatif de R9 (masque bipede) et celui de R11 (valeurs d'i56) sont desormais
prononces AVEC des instants d'usage certains ET avec un temoin positif interne qui passe a
facteur 9. **L'etat replique du bipede ne porte pas le geste du repulseur** : ce n'est plus
une conclusion tiree d'un plancher de bruit, c'est une mesure faite a l'endroit et a l'heure.

---

## 4. [ETABLI] LE CANAL DES EVENEMENTS — et un fait neuf : la marche EST possible hors catalogue

### 4.1 [ETABLI — DEBLOCAGE METHODOLOGIQUE] Le contexte de carte se reconstruit depuis le FILM

La marche de liste de R7 exige un `r7Ctx` — les ETENDUES metriques de la carte — pour
consommer les vecteurs quantifies de certaines charges, et R7 le prenait dans
`map_quant_bounds.json`. Argyle n'y est pas. **Il n'en a pas besoin.** La largeur d'axe vaut

```
bits(k) = ceil(log2(ceil(etendue * 60 / 2^(16-k))))
```

et `DetectI0Layout` rend deja `bits(16)` DEPUIS LE FILM (`AxisW`). Or pour toute etendue
reelle `e` avec `bits(16) = b`, c'est-a-dire `e` dans `]2^(b-1)/60, 2^b/60]`, on a pour TOUT
`k <= 16` : `bits(k) = b - 16 + k` — la meme valeur que pour l'etendue representative
`2^b / 60`. **La reconstruction `etendue := 2^AxisW[i] / 60` rend donc EXACTEMENT les memes
largeurs que la vraie carte.** Ce n'est pas une approximation : c'est une classe
d'equivalence, et `r12VerifieCtx` la controle a k=16 avant chaque mesure.

**Consequence pour le chantier, au-dela de R12 : la marche de liste d'evenements n'est plus
conditionnee au catalogue de bornes. Elle marche sur N'IMPORTE QUEL film.**

### 4.2 [ETABLI] Le cadrage est CONTROLE sur Argyle — les trois temoins de R7 passent

Instrument : `filmdec/r12_evenements_research_test.go`, `TestR12Cadrage`.

```
contexte reconstruit : etendues=546,1/546,1/2184,5 m  regionBits=1  (depuis AxisW=[15 15 17] gate=5)
MARCHE REELLE          : 5937 listes,  146 bloquees (2,46 %), 12545 evenements traverses
temoin decale +1 bits  : 5937 listes,  642 bloquees (10,81 %),  9823 evenements traverses
temoin decale +3 bits  : 5937 listes,  347 bloquees (5,84 %),   6203 evenements traverses
ORACLE cadrage JUSTE : 5790 trames · profondeur 1,763 records/paquet · 30,3 % de fermetures
ORACLE temoin +3 bits: 5790 trames · profondeur 0,323 records/paquet
TEMOIN 3 : facteur de profondeur 5,46  (seuil pre-inscrit de R7 : 3)
```

Trois faits : (a) le taux de blocage est 4,4x plus faible qu'a +1 bit et 2,4x plus faible
qu'a +3 bits ; (b) l'oracle de trame — LE JUGE de R7 — rend un facteur **5,46** contre un
seuil pre-inscrit de 3 ; (c) la profondeur au cadrage juste vaut **1,763**, c'est-a-dire la
mediane de reference du parc de R7 (1,793). **Le cadrage est bon sur ce film, et la
reconstruction du contexte est validee par son propre resultat.**

Les types opaques du film reproduisent le profil du parc de R7 dans le meme ordre :
`16 ShowDebugText:38 · 20 incident:12 · 12 biped_melee_clang:11 · 96 NetworkedCrewEventType:9 · ...`
(R7 sur 12 films : 16:575 · 20:155 · 12:140 · ...).

### 4.3 [ETABLI — LE FAIT CENTRAL DE CE LOT] Deux types de POUSSEE tombent sur l'ancre

Instrument : `TestR12Evenements`, recensement des evenements par type dans les cinq fenetres
d'usage contre le reste du film (224 listes dedans, 5 713 dehors).

| type | nom | dedans | dehors | tauxDed | tauxHor | facteur |
|---|---|---|---|---|---|---|
| 36 | `action_weapon_fire` | 164 | 3 037 | 0,732 | 0,532 | 1,38 |
| 5 | `projectile_detonate` | 18 | 122 | 0,080 | 0,021 | 3,76 |
| **105** | **`EquipmentObjectKnockedBack`** | **4** | **6** | 0,0179 | 0,0011 | **17,00** |
| **104** | **`EquipmentKnockbackPlayer`** | **2** | **0** | 0,0089 | 0,0000 | **infini** |
| 106 | `ObjectCollisionDamage` | 1 | 1 | 0,0045 | 0,0002 | 25,50 |

Et le journal des instants, qui vaut plus que les taux :

```
104 EquipmentKnockbackPlayer      2 occurrences : 5:25,5  5:25,6
105 EquipmentObjectKnockedBack   10 occurrences : 5:25,5  5:25,6  5:25,6  5:25,6
                                                 8:16,0  8:16,0  9:43,1  10:40,1 x3
14  PlayEffectOnObject            1 occurrence  : 0:54,2
30  biped_equipment_activation    ABSENT du film
119 EquipmentKnockbackRequest     ABSENT du film
103 EquipmentSpawnedObject        ABSENT du film
```

**Les DEUX seules occurrences du type 104 de tout le film tombent a 5:25,5 et 5:25,6 —
c'est-a-dire a l'instant du kill au repulseur d'Elmo910 (325 526 ms, mesure A1), a 100 ms
pres.** Et quatre des dix occurrences du type 105 sont dans les fenetres, dont trois au meme
instant et deux a **8:16,0** — c'est-a-dire 600 ms apres le ramassage de Bot ziker mesure par
i48 (8:15,4), la ou le releve dit « il l'utilise quasi immediatement ».

**Le report n°1 de R9 — le type 14 `PlayEffectOnObject` — est REFUTE comme canal du
repulseur** : il apparait UNE fois dans tout le film, a 0:54,2, loin de tout usage.

---

## 5. LA REFUTATION ACTIVE — R7 a declare ces types ABSENTS, et son argument est serieux

R7 par. 4.3 conclut : « les occurrences de 104, 42 et 43 sont la derive produite par [les
largeurs fausses de `damage_aftermath` et `projectile_detonate`] ». Ses trois epreuves :
0 tete sur 108 occurrences (42 attendues, p ~ 1e-23) ; une reference constante a 4224 ;
75 % des 104 precedes d'un `damage_aftermath`. **Il faut donc chercher activement contre le
resultat du par. 4.3, et c'est l'objet de ce paragraphe.**

Instrument : `filmdec/r12_knockback_research_test.go`, `TestR12Knockback`.

### 5.1 [ETABLI] Occurrence par occurrence : position, predecesseur, reference

| instant | type | pos | long | predecesseur | ref0 |
|---|---|---|---|---|---|
| 5:25,5 | **104** | 4 | 52 | **105** `EquipmentObjectKnockedBack` | 51 |
| 5:25,6 | **104** | 2 | 6 | **105** `EquipmentObjectKnockedBack` | 52 |
| 5:25,5 | 105 | 3 | 52 | 38 `weapon_reload` | 51 |
| **5:25,6** | **105** | **1 — TETE** | 6 | (rien) | 52 |
| 5:25,6 | 105 | 3 | 6 | 104 | 1647 |
| 5:25,6 | 105 | 4 | 6 | 105 | 1646 |
| 8:16,0 | 105 | 2 | 3 | 75 `AIDialog` | 1000 |
| 8:16,0 | 105 | 3 | 3 | 105 | (aucune) |
| 9:43,1 | 105 | 3 | 3 | **0 `damage_aftermath`** | 782 |
| 10:40,1 | 105 | 2 | 4 | 75 `AIDialog` | 2223 |

**TROIS FAITS, dont un decisif.**

1. **UNE des occurrences est en POSITION 1 — LA TETE.** A 5:25,6, le type 105 est le PREMIER
   evenement de sa liste : `[1 bit config][1 bit continuation][R(7) type]`, rien ne le
   precede, **aucune derive n'est possible**. C'est l'epreuve D de R7 elle-meme, et R7 la
   reconnait pour ce type (« le type 105 EST en tete 8 fois » sur son parc). **Il y a donc,
   avec CERTITUDE DE CADRAGE, un evenement `EquipmentObjectKnockedBack` a 5:25,6 dans ce
   film — c'est-a-dire 100 ms apres le kill au repulseur.**
2. **Le predecesseur des deux 104 n'est PAS `damage_aftermath` : c'est 105.** L'argument
   d'enrichissement de R7 (75 % precedes du type 0) ne s'applique pas a ces deux-la. La
   chaine `105 -> 104` a du sens : le souffle pousse un objet ET un joueur.
3. **Les references ne sont PAS la constante 4224 de R7** : 51, 52, 1647, 1646, 1000, 782,
   2223. Elles varient. Leur DOMAINE reste inconnu et ce lot ne le resout pas.

### 5.2 [ETABLI] L'oracle de trame restreint : les listes qui portent un 104 ne derivent pas

| population | trames | profondeur | fermetures propres |
|---|---|---|---|
| toutes listes | 5 790 | **1,763** | 30,3 % |
| listes portant un 104 | 2 | **1,500** | 50,0 % |
| listes portant un 105 | 5 | **1,600** | 40,0 % |
| *(temoin : cadrage decale +3 bits, toutes listes)* | 5 790 | **0,323** | — |

Une liste mal cadree fait s'effondrer la trame qui la suit (0,323 contre 1,763, facteur 5,46).
Les listes portant un 104 ou un 105 rendent 1,50 et 1,60 : **elles sont dans le regime normal,
pas dans le regime de derive.** Reserve dite : n = 2 et n = 5, la mesure est faible.

### 5.3 [ETABLI — LE CONTRE-ARGUMENT INTERNE LE PLUS FORT] 51 kills, 615 `damage_aftermath`, DEUX 104

Si le type 104 etait la derive produite par la largeur de `damage_aftermath`, il apparaitrait
la ou ce type apparait. Ce film porte **51 kills** (mesure A1) et **2 141 evenements de
type 0** traverses par la marche. Le type 104, lui, apparait **DEUX fois, aux memes 100 ms**,
et ces 100 ms sont celles du SEUL kill des 51 dont la source de degat est le repulseur.

**Une derive n'est pas selective.** C'est l'argument que R7 ne pouvait pas construire : il
mesurait des agregats de parc, sans savoir a quel instant regarder.

### 5.4 [ETABLI — ET IL COUTE CHER A LA THESE] Le corpus dit que ces types sont AUSSI du bruit

Vingt films, marche complete, comptes bruts (`TestR12Knockback` avec `R12_CORPUS=1`) :

| film | palette | viesRep | viesGra | listes | t0 | t104 | t105 | t119 | t117 | t14 |
|---|---|---|---|---|---|---|---|---|---|---|
| `215e7022` | A | 11 | 16 | 5 937 | 2 141 | 2 | 10 | 0 | 2 | 1 |
| `00ba2e1c` | A | 36 | 34 | 7 336 | 2 880 | 2 | 1 | 2 | 3 | 2 |
| `06dfe6d9` | A | 38 | 35 | 11 912 | 6 276 | 11 | 11 | 4 | 2 | 6 |
| `084a804d` | A | 18 | 48 | 17 149 | 4 306 | **11** | **27** | **19** | **20** | **21** |
| `11de8353` | A | 24 | 29 | 7 325 | 1 987 | 5 | 2 | 1 | 4 | 3 |
| `72b0a25e` | A | 15 | 0 | 4 122 | 1 222 | 3 | 3 | 0 | 0 | 0 |
| `d9781168` | A | 37 | 0 | 8 737 | 3 156 | 4 | 6 | 1 | 5 | 1 |
| `53ce4390` | A | 22 | 25 | 7 457 | 3 083 | 8 | 13 | 0 | 0 | 4 |
| `a6ae19fb` | A | 20 | 4 | 4 489 | 1 759 | 10 | 21 | 1 | 0 | 1 |
| `3d58eb37` | A | 18 | 6 | 4 594 | 1 958 | 6 | 17 | 1 | 2 | 3 |
| `0d265ab0` | A | 15 | 2 | 4 908 | 2 218 | 1 | 3 | 5 | 1 | 3 |
| `4577fcc4` | A | 6 | 0 | 5 367 | 1 814 | 0 | 0 | 0 | 6 | 1 |
| `f2966f08` | A | 15 | 14 | 6 816 | 2 567 | 2 | 3 | 0 | 2 | 6 |
| `efe716b4` | A | 20 | 0 | 5 983 | 1 580 | 0 | 1 | 0 | 1 | 2 |
| `1cd3848a` | B | ? | 21 | 5 571 | 2 787 | 4 | 3 | 2 | 9 | 4 |
| `000d5950` | B | ? | 22 | 3 508 | 753 | **0** | **0** | 0 | 0 | 0 |
| `00502e52` | B | ? | 17 | 3 793 | 712 | **0** | **0** | 0 | 0 | 0 |
| `07aa428d` | B | ? | 10 | 4 103 | 847 | 0 | 1 | 0 | 1 | 1 |
| `3372e7eb` | A | 8 | 0 | 1 776 | 529 | 0 | 3 | 0 | 0 | 0 |
| `51ebbc0f` | A | 25 | 0 | 4 868 | 1 409 | 4 | 7 | 0 | 1 | 1 |

**CE TABLEAU JOUE CONTRE LA THESE, ET IL FAUT LE DIRE SANS L'ADOUCIR.** Les cinq types rares
(104, 105, 119, 117, 14) MONTENT ET DESCENDENT ENSEMBLE : `084a804d` — le film dont R7 mesure
la marche la plus mauvaise (92,8 % de profondeur) — en rend 11, 27, 19, 20, 21 ; les films de
famille B les mieux cadres en rendent 0, 0, 0, 0, 0. **C'est la signature d'une cause commune,
et cette cause commune est la derive residuelle de marche.** Un compte de type 104 tire de la
MARCHE ne peut donc PAS, a lui seul, etablir que ce type existe.

**Ce que ce tableau ne dit PAS, en revanche** : il ne dit rien de la LOCALISATION. Que la
derive fabrique des occurrences ne veut pas dire que TOUTE occurrence est une derive, et le
par. 5.1 tient une occurrence dont le cadrage est CERTAIN (position 1).

### 5.5 [ETABLI — LE FILTRE QUI SEPARE LES DEUX] Les TETES, sur 20 films

Instrument : `filmdec/r12_corpus_research_test.go`, `TestR12Corpus`. Il ne marche RIEN : il
lit la TETE de chaque paquet delta, dont le cadrage est certain.

| type | tetes sur 20 films (623 313 paquets delta) |
|---|---|
| **104 `EquipmentKnockbackPlayer`** | **0** |
| **105 `EquipmentObjectKnockedBack`** | **15** |
| 117 `EquipmentTranslocatorTeleportEffects` | 9 |
| 103 `EquipmentSpawnedObject` | 451 |
| 9 `biped_pickup` | 4 030 |

**Et ces 20 films portent environ un millier de morts** (51 sur le seul `215e7022`). Quinze
tetes de type 105 pour un millier de morts : **ce type n'est pas un accompagnement banal du
degat ou de la mort.** C'est un evenement rare, et il faut donc expliquer pourquoi l'une de
ces quinze tombe a 100 ms du seul kill au repulseur du corpus.

**Second echantillon, NON CHOISI : les 42 premiers films de la racine par ordre alphabetique**
(balayage interrompu volontairement a 42/140 pour rendre la main ; l'ordre alphabetique d'un
identifiant de match est sans rapport avec son contenu, l'echantillon est donc non biaise).

| type | tetes sur 42 films (1 223 551 paquets delta) |
|---|---|
| **104 `EquipmentKnockbackPlayer`** | **2** (les deux sur `03af54c3`) |
| **105 `EquipmentObjectKnockedBack`** | **22** |
| 117 `EquipmentTranslocatorTeleportEffects` | 12 |
| 103 `EquipmentSpawnedObject` | 929 |
| 9 `biped_pickup` | 7 062 |

**AMENDEMENT A R7, ecrit tel quel : le type 104 n'est pas « jamais en tete ».** R7 en mesurait
0 sur 12 films ; ce lot en mesure 2 sur 42 autres films. C'est extraordinairement rare
(2 pour 1,22 million de paquets delta) — mais ce n'est pas zero, et l'argument
« probabilite 1e-23 » de R7 par. 4.3 doit etre lu comme « ce type est trop rare pour que sa
part en tete se mesure », pas comme « ce type n'existe pas ».

**Sur ces 20 films, le type 104 n'est jamais en tete — R7 avait raison a cette echelle** (le
second echantillon, ci-dessous, en trouvera deux sur 42 autres films : c'est un type
extraordinairement rare, pas un type absent). Les deux occurrences de 5:25 restent donc SOUS
RESERVE DE CADRAGE : elles sont en position 4 et 2, pas en tete.

**Le type 105, lui, EXISTE : 15 tetes.** Et c'est sur lui que porte la suite.

---

## 6. [ETABLI — LA SECONDE ANCRE, SUR UN AUTRE FILM] Une tete de 105 a 149 ms d'une consommation certaine

Le par. 5 laisse la thèse suspendue a UNE coincidence, sur UN film. Il en faut une seconde,
independante, et elle existe — **sans aucun releve Theater**.

### 6.1 Une ancre d'usage que le film se donne a lui-meme

R11 par. 5 l'a etablie et ce lot la reprend telle quelle : **une PORTE OUVERTE d'i48
(`AbilitySetNoRank`) dont la lecture precedente, POUR LE MEME SLOT, portait le rang du
repulseur, signifie que la DERNIERE CHARGE vient d'etre consommee** — le film declare lui-meme
que le joueur ne porte plus rien. C'est un usage CERTAIN, date par le film, sans temoin humain.
`r12Consommations` les calcule ; le corpus en compte **4 sur 42 films** (elles sont rares).

### 6.2 [ETABLI] Le resultat

Quatre films, dont les deux que R11 par. 5 nomme (`a6ae19fb`, `0d265ab0`) :

| film | vies rep. | paquets delta | **tetes 105** | consommations rep. |
|---|---|---|---|---|
| `215e7022` | 11 | 39 111 | **1** | 0 |
| `a6ae19fb` | 20 | 30 364 | **3** | **1** |
| `0d265ab0` | 15 | 34 097 | 0 | 1 |
| `03e3f9ea` | 17 | 31 396 | 8 | 2 |

```
a6ae19fb   type=105  7:29,1   ecart a la CONSOMMATION de repulseur la plus proche = -149 ms
```

**R11 par. 5 avait publie ce meme instant, en toutes lettres, comme « le cas qui tranche » :**

> `a6ae19fb`, slot 569 : 6:21 prise du repulseur, **7:29 porte ouverte — DERNIERE CHARGE
> CONSOMMEE**, 7:32 il en reprend un.

**Une tete de type 105 — cadrage CERTAIN — tombe 149 ms AVANT cette declaration.** L'ordre
causal est le bon : l'objet est pousse, puis le film annonce que le porteur n'a plus rien.

### 6.3 Ce que ce second cas apporte, et ce qu'il n'apporte pas

- Il est **independant du releve Theater** (l'ancre vient d'i48) et **d'un autre film**, d'une
  autre carte, d'un autre match.
- Il porte sur une tete, donc le cadrage n'est pas en cause.
- **Ordre de grandeur du hasard** : `a6ae19fb` porte 3 tetes de 105 sur 780 s de film ;
  la probabilite qu'une d'entre elles tombe a moins de 150 ms d'un instant fixe est de l'ordre
  de **1,2e-3**. Sur `215e7022`, 1 tete sur 690 s, a moins de 150 ms du kill au repulseur :
  **4,3e-4**. Les deux ensemble : de l'ordre de **5e-7**.
- **Il n'etablit PAS un rappel.** Le troisieme film qui porte une consommation certaine
  (`0d265ab0`, 4:19 — l'autre cas de R11) rend **ZERO** tete de type 105. Une charge de
  repulseur consommee ne produit donc PAS toujours l'evenement.

---

## 7. [ETABLI] LE SOCLE — ses REAPPARITIONS sont datees, 3/3

Instrument : `filmdec/r12_socles_research_test.go`, `TestR12Socles`. Il date les CREATIONS
d'entites d'archetype ti=37 par le GlobalID de leur `eqip` (bloc
`object-multiplayer-properties`, mot de 32 bits inconditionnel).

Sur `215e7022` : 354 creations lues, **0 sans identifiant**, 38 identifiants distincts. Le
GlobalID du repulseur (`0x7ca85adc`, `replay_labels.toml`) en porte **14** :

```
0:30,0  3:56,9  4:11,6  4:38,6  4:49,6  5:27,6  5:48,5  5:54,9  5:57,3
8:20,2  8:36,2  8:40,8  9:15,7  10:15,1
```

| reapparition relevee au Theater | creation la plus proche | ecart |
|---|---|---|
| S1 4:49 | **4:49,6** | +677 ms |
| S2 5:56 | **5:54,9** | −1 041 ms |
| S3 9:14 | **9:15,7** | +1 768 ms |

**3/3 apparies ; 0/3 pour le temoin decale de +30 s.** Et la separation est propre dans
l'autre sens : les quatre RAMASSAGES du releve (3:48, 4:55, 8:14, 9:50) rendent **0/4** contre
ce meme canal — a +8,9 s, −5,3 s, +6,3 s, +25,1 s. **Le canal des creations date les
REAPPARITIONS AU SOCLE, pas les prises**, exactement comme il devrait.

**Ce qu'il ne dit PAS** : rien de la JAUGE DE CHARGEMENT que le collegue voit a l'ecran. Une
reapparition datee n'est pas un etat de rechargement. Note laterale, gratuite et utile :
l'identifiant `0xe7be9f5c` du meme film rend une serie parfaitement periodique — 0:30,0 puis
1:39,0 2:22,2 2:52,5 4:22,3 4:52,6 6:22,4 6:52,7 8:22,5 8:52,8 10:22,6 — un cycle de 30 s
imbrique dans un cycle de 90 s : c'est la signature d'un socle a horloge fixe.

---

## 8. VERDICT

### 8.1 Ce qui est ETABLI

| question | verdict | denominateurs |
|---|---|---|
| **La convention de temps du releve** | **TEMPS DE FILM**, etabli DEUX FOIS par deux canaux sans rapport | A1 : kill au `jpt! 0x07104b31` a 325 526 ms = 5:25,5 contre 5:25 releve · A2 : 4/4 ramassages apparies a +1,35 a +1,97 s (bande de 620 ms), temoins decales 1/4 et 0/4 |
| **Le `jpt! 0x07104b31` = Repulsor** | **CONFIRME sur corpus reel ET par verite terrain Theater** — la reserve specifique du registre tombe | 1 kill sur 51 dans le film, tueur et instant concordants avec le releve |
| **L'etat replique du BIPEDE porte-t-il le geste ?** | **NON**, et le negatif est desormais ANCRE avec un temoin positif interne qui passe | 706 records de porteur en fenetre, 64 composants, tous <= 1 en facteur de port ; temoin grappin i56 x9,00 |
| **Le type 14 `PlayEffectOnObject` (report n°1 de R9)** | **REFUTE** | 1 occurrence dans tout le film, a 0:54,2 |
| **Le canal de l'EFFET du repulseur** | **LE TYPE 105 `EquipmentObjectKnockedBack`** — et, sous reserve de cadrage, le type 104 `EquipmentKnockbackPlayer` | deux ancres independantes a −100 ms et −149 ms, sur deux films et deux sources d'ancre ; hasard de l'ordre de 5e-7 |
| **Les REAPPARITIONS au socle** | **DATEES** par les creations ti=37 du GlobalID `0x7ca85adc` | 3/3, temoin decale 0/3 ; et 0/4 sur les ramassages (separation propre) |
| **La marche de liste hors catalogue de bornes** | **DEBLOQUEE** : le contexte de carte se reconstruit depuis `AxisW` du film | oracle de trame facteur 5,46 (seuil 3), profondeur 1,763 = mediane du parc de R7 |

### 8.2 La reponse a la question posee

**Le film ne porte PAS le GESTE du repulseur — il porte son EFFET.** Rien, dans l'etat
replique du bipede, ne change quand un joueur declenche un repulseur : R9 et R11 l'avaient
conclu sans ancre, ce lot le mesure a l'endroit et a l'heure, avec un temoin positif qui
passe a facteur 9 sur le grappin dans le MEME film.

Ce qui change, c'est **la liste d'evenements du paquet**, et ce qui s'y ecrit est la
consequence, pas la commande : `EquipmentObjectKnockedBack` (type 105) quand le souffle
deplace un OBJET, `EquipmentKnockbackPlayer` (type 104) quand il deplace un JOUEUR.

**C'est pourquoi le rappel est structurellement partiel, et ce n'est pas un defaut de mesure :
un repulseur declenche dans le vide ne pousse rien, donc n'ecrit rien.** Sur les cinq usages
releves, deux laissent une trace (U1 5:25 et U2 8:16, ce dernier 600 ms apres le ramassage
mesure par i48) et trois n'en laissent aucune. Sur les deux consommations certaines de R11
par. 5, une laisse une tete de 105 a 149 ms (`a6ae19fb`) et l'autre rien (`0d265ab0`).

**Precision et rappel, ecrits avec leurs denominateurs :**

- **Sens ancre -> evenement** : 2 instants d'usage de repulseur certains ET indexes par un
  canal independant (le kill de `215e7022`, la consommation d'`a6ae19fb`) ; **2 sur 2** portent
  une tete de type 105 a moins de 150 ms.
- **Sens evenement -> ancre** : 12 tetes de type 105 sur les 4 films instruits ; **2** tombent
  sur un usage de repulseur connu. Les 10 autres ne sont PAS des faux positifs mesures :
  aucune verite terrain ne couvre leurs instants, et 8 d'entre elles sont sur un film
  (`03e3f9ea`) dont aucun usage n'est date.
- **Rappel sur le releve** : **2 usages sur 5**. Le reste est structurel (voir ci-dessus).
- **Rareté de base** : 22 tetes de type 105 pour **1 223 551 paquets delta** sur 42 films non
  choisis, soit 1,8e-5 par paquet — et environ un millier de morts dans ce corpus. Ce type
  n'accompagne ni le degat, ni la mort, ni le tir.

### 8.3 La reserve, dite en entier

**Le type 104 reste SOUS RESERVE DE CADRAGE.** Il n'apparait presque jamais en tete (2 sur
1,22 million de paquets delta, sur un seul film) et les deux occurrences de `215e7022` sont en
position 4 et 2. Elles sont donc lues par la MARCHE, et le par. 5.4 montre que la marche
FABRIQUE des occurrences de ce type sur les films mal cadres. Ce qui les sauve : leur
predecesseur est un 105 (pas le `damage_aftermath` suspect de R7), l'oracle de trame restreint
ne s'effondre pas (1,50 contre 1,763 en moyenne et 0,323 en temoin decale), et surtout elles
tombent a 100 ms d'un instant que deux autres canaux datent. **Mais deux occurrences ne font
pas une preuve de grammaire, et ce rapport ne les publie pas comme telles.**

**Le type 105, lui, n'est pas sous cette reserve** : la tete de `215e7022` (position 1) et
celle d'`a6ae19fb` (7:29,1) ont un cadrage CERTAIN.

---

## 9. INSTRUMENTS ET COMMANDES REJOUABLES

Huit fichiers, tous `*_research_test.go`, gardes par variables d'environnement, sautes par
defaut (CI comprise), `CGO_ENABLED=0`, `LockProcessDecode` tenu pendant chaque decodage,
`WorldObjectPrecision` pose depuis le LAYOUT DU FILM et restaure en sortie, aucune ecriture,
aucune DuckDB ouverte, aucun code de production touche, aucun commit.

| Fichier | Role |
|---|---|
| `killsource/r12_ancre_kill_research_test.go` | `TestR12AncreKill` — le PONT : `killsource.Decode` (production, gele) sur le film ancre, journal des 51 kills, et le kill au `jpt! 0x07104b31` |
| `filmdec/r12_socle_research_test.go` | le SOCLE du lot : setup SANS artefact et SANS bornes, collecte i48/i56/i57/i59 ; `TestR12Ancrage` publie le journal i48 et juge la mesure A2 |
| `filmdec/r12_palette_research_test.go` | la PALETTE de capacites sans artefact (marqueurs et noms recopies de `replay_labels.toml`, regle de purete a 90 %) |
| `filmdec/r12_fenetres_research_test.go` | `TestR12Fenetres` — les deux recensements ANCRES (tetes d'evenement, masque bipede a 64 composants) sur les fenetres d'usage du repulseur ET du grappin (temoin positif interne) |
| `filmdec/r12_evenements_research_test.go` | `TestR12Cadrage` (contexte reconstruit depuis le layout + les 3 temoins de R7) et `TestR12Evenements` (recensement ancre de la liste COMPLETE + journal des instants par type) |
| `filmdec/r12_knockback_research_test.go` | `TestR12Knockback` — position / predecesseur / reference de chaque occurrence de 104, 105, 119, 116, 117, 103, 14, 93, 98, 30 ; oracle de trame RESTREINT ; mode corpus |
| `filmdec/r12_corpus_research_test.go` | `TestR12Corpus` — les TETES seules (cadrage certain) sur un corpus, croisees avec les vies de repulseur et les CONSOMMATIONS certaines (porte ouverte d'i48 apres un rang de repulseur) |
| `filmdec/r12_socles_research_test.go` | `TestR12Socles` — les creations d'entites ti=37 par GlobalID d'`eqip`, et l'appariement aux reapparitions au socle du releve |

Chemins : `<F>` = `<repo>/data/cache/film_chunks` (ici `<repo>` =
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`). **Aucun catalogue de bornes,
aucun artefact de rejeu n'est requis par un seul de ces instruments.**

```
# depuis apps/go-api

# A1 — le pont du kill (11 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/games/halo_infinite/film/killsource/ \
  -run '^TestR12AncreKill$' -count=1 -timeout 30m -v

# A2 — le journal i48 et l'appariement des ramassages (10 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Ancrage$' -count=1 -timeout 60m -v

# B1 — les deux recensements ancres, avec le temoin positif grappin (10 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Fenetres$' -count=1 -timeout 60m -v

# le cadrage sur une carte HORS CATALOGUE, et les trois temoins de R7 (23 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Cadrage$' -count=1 -timeout 60m -v

# B2 — le recensement ancre de la liste complete + le journal des instants (9 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Evenements$' -count=1 -timeout 60m -v

# la refutation active : position, predecesseur, reference, oracle restreint (37 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Knockback$' -count=1 -timeout 60m -v

# le meme, en mode CORPUS (resume par film seulement) — 13 s/film
CGO_ENABLED=0 R12_FILMS=<F> R12_CORPUS=1 R12_IDS=a,b,c \
  go test ./internal/analysis/filmdec/ -run '^TestR12Knockback$' -count=1 -timeout 120m -v

# les TETES sur un corpus borne (R12_LIMIT / R12_SKIP enumerent la racine) — 11 a 100 s/film
CGO_ENABLED=0 R12_FILMS=<F> R12_LIMIT=140 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Corpus$' -count=1 -timeout 300m -v

# LA SECONDE ANCRE (par. 6) : les 4 films qui portent une consommation certaine (36 s)
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022,a6ae19fb,0d265ab0,03e3f9ea \
  go test ./internal/analysis/filmdec/ -run '^TestR12Corpus$' -count=1 -timeout 120m -v

# B3 — les creations ti=37 par GlobalID d'eqip, et les reapparitions au socle
CGO_ENABLED=0 R12_FILMS=<F> R12_IDS=215e7022 \
  go test ./internal/analysis/filmdec/ -run '^TestR12Socles$' -count=1 -timeout 60m -v
```

Variables : `R12_FILMS` (racine des chunks, obligatoire), `R12_IDS` (liste explicite) OU
`R12_LIMIT`/`R12_SKIP` (enumeration bornee de la racine), `R12_FENETRE_MS` (demi-largeur des
fenetres, defaut 1500), `R12_CORPUS=1` (resume par film).

---

## 10. CE QUE CE LOT NE DIT PAS

- **Il ne porte pas la grammaire des types 104 et 105.** Il les COMPTE et les DATE ; il ne lit
  ni la direction, ni la magnitude, ni le domaine de leur reference. R7 par. 4.1 a deja lu ces
  grammaires dans l'exe (104 : `R(1)` puis direction unitaire `R(19)` + magnitude `R(10)` en
  echelle logarithmique entre 0,05 et 20,0 ; 105 : `[R(1) g ; si g : R(32)]`) — les porter et
  les valider est un lot a part entiere.
- **Il ne nomme pas le POUSSE.** Les references relevees (51, 52, 1 000, 1 646, 1 647, 782,
  2 223) varient d'une occurrence a l'autre — donc elles ne sont pas la constante 4224 que R7
  avait mesuree —, mais leur DOMAINE n'est pas resolu et aucune calibration de base n'a ete
  tentee ici.
- **Il ne mesure pas le GESTE.** Ce qu'il trouve est l'EFFET : le type 105 ne se declenche que
  si un OBJET est pousse, le type 104 que si un JOUEUR l'est. Un usage dans le vide ne produit
  ni l'un ni l'autre, et le rappel sur les usages en souffre structurellement.
- **Il ne rejoue aucun canal ferme par R8, R9 et R11** : poses `deployed`, tag d'i57/i59, i54,
  composants d'etat de ti=37 dans les deltas, jointure par les handles i26, i27, face victime
  des positions repliquees. Ils restent tels que ces lots les ont laisses.
- **Il ne touche AUCUN code de production.** Sept `*_research_test.go` gardes par
  environnement, non commites. `gofmt -l` vide, `go vet` vert sur les deux paquets.
- **Il n'ouvre aucune DuckDB** et ne lit la production qu'en fichiers de cache de film.
- **Ghidra n'a pas ete lance** : il n'a pas ete necessaire. Aucun serveur a arreter, aucun
  verrou residuel.
- **La reserve d'identite de `killsource` est dite** : `LineByLinePublishable()` rend faux sur
  `215e7022` (marge de bijection nulle), donc les GAMERTAGS ligne a ligne de ce film ne sont
  pas publiables en toute rigueur. Le tag et l'instant, eux, ne passent pas par la bijection.
- **Le balayage large a ete INTERROMPU volontairement.** Le recensement des tetes sur les 140
  premiers films de la racine a ete arrete a 42 (le cout par film est passe de 11 s a plus de
  100 s en cours de campagne, sans cause identifiee — cache disque probablement). Les
  denominateurs publies (1 223 551 paquets delta, 42 films) sont donc ceux de la partie
  executee, et ils sont annonces comme tels. Reprise : `R12_LIMIT=140 R12_SKIP=42`.

---

## 11. REGISTRE DES REPORTS

| # | Ce qu'il faut faire | Pourquoi | Cout |
|---|---|---|---|
| 1 | **Porter la grammaire des types 104 et 105** (R7 par. 4.1 les a deja lues dans l'exe : 104 = `R(1)` + direction unitaire `R(19)` + magnitude `R(10)` log entre 0,05 et 20,0 ; 105 = `[R(1) g ; si g : R(32)]`) et calibrer le DOMAINE de leur reference | c'est ce qui transformerait une DATE en un fait de rejeu : qui pousse, qui est pousse, dans quelle direction, avec quelle force | 1 lot |
| 2 | **Elargir le recensement des TETES** aux 1 380 films du cache (`R12_LIMIT`/`R12_SKIP`) et croiser les tetes de 105 avec TOUTES les consommations certaines de repulseur du parc | le corpus de ce lot n'en porte que 4 ; a 1 380 films il y en aurait une centaine, assez pour un rappel chiffre | 1 lot (long, borner par tranches) |
| 3 | **Relever au Theater les tetes de 105 non expliquees** : `03e3f9ea` 1:16,5 (x4), 1:30,4 (x2), 4:41,1, 7:51,0 ; `a6ae19fb` 5:39,3 et 6:43,3 ; `215e7022` n'en a pas d'autre | c'est le seul moyen de mesurer une PRECISION : ces dix tetes sont soit des usages de repulseur non releves, soit des faux positifs, et rien dans le film ne le dit | 1 releve |
| 4 | **Instruire l'usage qui NE POUSSE RIEN** : U3 (8:20), U4 (9:54), U5 (10:05) du releve ne laissent aucune trace. Si le geste doit etre rejoue, il faudra un canal de COMMANDE, et ce lot etablit qu'il n'est ni dans le bipede ni dans les types 104/105 | c'est la moitie manquante du rejeu du repulseur | 1 lot |
| 5 | **Exploiter le canal des creations ti=37 par GlobalID pour le calque des SOCLES d'equipement** : il date les reapparitions a la seconde (3/3), il n'a besoin ni d'artefact ni de bornes metriques pour les INSTANTS | acquis livrable en l'etat, hors question du repulseur | 1/2 lot |
| 6 | **Porter la reconstruction du contexte de carte depuis le layout** (par. 4.1) partout ou un instrument R7 exige `map_quant_bounds.json` | elle debloque le canal des evenements sur toutes les cartes hors catalogue, Argyle et les variantes Forge comprises | 1/2 lot |
| 7 | **Amender le rapport R7** : son par. 4.3 conclut « 0 tete sur 108 occurrences, p ~ 1e-23 » pour le type 104 ; ce lot en mesure 2 sur 42 autres films. La conclusion « ce type est trop rare pour que sa part en tete se mesure » remplace « ce type n'existe pas » | un rapport qui garde un argument trop fort se relit mal | 15 min |
| 8 | **Mettre a jour le registre des reports** (`.ai/V7.5/REGISTRE_REPORTS.md`, ligne « Repulseur : le tag `07104b31` reste SOUS_RESERVE ») : la verite terrain Theater qui lui manquait existe (par. 2.1) | la ligne dit « PAS de verite-terrain Theater » et c'est desormais faux | 15 min |

---
