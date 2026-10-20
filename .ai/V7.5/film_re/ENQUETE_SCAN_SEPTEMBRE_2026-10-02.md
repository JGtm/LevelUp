# Enquête : part de `scan` du killsource en septembre 2026 (2026-10-02)

Worktree `C:/Users/Guillaume/Projects/LevelUp-wt-suite-audit` (branche `feat/suite-audit-decodeur`,
tête `8b894a677`). Enquête seule : aucun code de production modifié, aucune écriture sous `data/`.

**Lecture de la base.** `shared_matches_v2.duckdb` du checkout principal (tenue par le serveur dev) a
été lue par `OpenReadForQuery` (`internal/platform/duckdb/db.go`). L'outil était un petit exécutable
jetable non commité (`cmd/zz_enquete_scan_q`), supprimé à la fin. Toutes les requêtes portent sur
`match_kill_events_latest` à `decoder_rev = 'killsource-2026-09-27'`, avec le temps canonique
`COALESCE(start_time_utc, start_time AT TIME ZONE 'UTC')`.

**Lecture des films.** Les films ont été lus en lecture seule (`data/cache/film_chunks` du principal),
un décodage à la fois. Outils :
- `cmd/killsource sante|json -carte <nom>` ;
- un test de recherche JETABLE dans le paquet `facts/killsource` (`zz_enquete_scan_local_test.go`,
  non commité, supprimé ; copie dans le scratchpad de la session). Il rejoue `prepare` + `run`, puis
  refait la marche paquet par paquet, à l'identique de `runWalk`, en la mesurant.

Pour chaque kill publié par le scan, ce test donne :
- le paquet, le bit du dead-state et le début localisé ;
- le premier record qui casse la marche ;
- un **essai** : le premier départ `s'` d'où la marche de production relit le dead-state AU MÊME BIT
  que le scan, et la chaîne de records lue depuis `s'` ;
- deux expériences de localisateur (§3).

## Verdict en une phrase

Le 19,1 % de septembre est un **effet de composition**. 19 matchs à **objectif porté unique** (18 en
Oddball classé, plus 1 variante `4640310c` de Squad Battle) portent **85,3 %** des kills `scan` du mois.

Dans ces modes, le localisateur de la marche cherche une signature figée sur le **slot 123** : un delta
de l'archétype « high-frequency » long de 35 bits. Or, dans ces modes, ce delta est porté par d'AUTRES
slots du même archétype (124, 126 à 129), et le slot 123 est absent du paquet. Les paquets à événements
ne sont donc pas localisés, et la marche ne lit pas leurs morts.

Le défaut relève d'un **correctif du décodeur, petit (S) en code** : accepter tout slot lié à
l'archétype high-frequency. Mesuré sur trois films, ce changement rend à la marche **157/167**,
**180/209** et **25/27** des kills servis par le scan.

Une seconde cause, propre à la carte **Banished Narrows** (≈ 49 % de scan sur 13 matchs, tous mois
confondus), est distincte. La marche y casse sur le delta d'un objet transitoire qu'aucune image-clé ne
déclare. Elle ne relève pas du localisateur, et son correctif est plus lourd (M/L, avec rétro-ingénierie).

Le build n'y est pour rien : tous les films de septembre sont en `HI_1_13_0`, le même build que de
décembre 2025 à août 2026, et un Oddball de 2024 (`HI_1_8_0`) montre le même défaut.

## 1. Concentration (question 1)

### 1.1 Septembre par classe de match

| classe | matchs | kills | scan | part |
|---|---|---|---|---|
| objectif porté unique (Ranked:Oddball ×18, variante `4640310c` ×1) | 19 | 3 528 | **2 246** | **63,7 %** |
| Banished Narrows (Slayer, Strongholds) | 2 | 153 | 76 | 49,7 % |
| origine `bot` (scan PAR CONSTRUCTION, cf. §5.b) | 7 | 51 | 51 | 100 % |
| reste | 101 | 10 145 | 259 | **2,6 %** |
| **total septembre** | 121 | 13 747 | 2 632 | 19,1 % |

Le compte de matchs de la ligne `bot` recoupe les autres lignes.

Avant septembre (tous mois confondus) :

| classe | part de scan |
|---|---|
| objectif porté unique (13 matchs) | 70,4 % |
| Banished Narrows (11 matchs) | 48,6 % |
| reste (1 075 matchs) | 4,2 % |

Le « reste » de septembre (2,6 %) est donc **meilleur** que l'historique.

### 1.2 Par mode (tous mois)

Les modes à objectif unique porté par un joueur sont tous en tête :

| mode | part de scan |
|---|---|
| Assault:One Bomb | 81,9 % |
| BTB:One Flag CTF | 74,6 % |
| Oddball:Arena | 71,3 % |
| variante `4640310c` | 69,3 % |
| Ranked:Oddball, septembre | 63,1 % |
| Ranked:Oddball, avant septembre | 47,9 % |

Les autres modes restent bas :

| mode | part de scan |
|---|---|
| CTF:Arena (deux drapeaux) | 9,9 % |
| Ranked:Strongholds | 2,6 % |
| Ranked:King of the Hill | 2,0 % |
| Ranked:Slayer | 0,4 % |

### 1.3 Par match

Les 18 Oddball classés de septembre se répartissent ainsi :
- **16** sont entre **27,7 % et 74,3 %**, par exemple :

| match | carte | part de scan |
|---|---|---|
| `8f7f5806` | Live Fire | 74,3 % |
| `a765f61d` | Live Fire | 71,8 % |
| `c46ef9d1` | Recharge | 69,6 % |
| `af3af607` | Lattice | 69,0 % |
| `9c0ec856` | Recharge | 27,7 % |

- les deux autres sont à 0 % sur 2 et 3 kills seulement (`4555ce28`, `5e2214a0`).

La variante `4640310c` (`6b0e6f0f`) est à 69,3 %. Elle se joue sur Refuge en Squad Battle, avec un score
de 3-2 en 21 min. Son nom de variante n'est pas résolu : `metadata.duckdb` est tenue en écriture par le
serveur.

Le défaut est **diffus à l'intérieur de la classe** : tous les Oddball sont touchés. Il est **concentré
par mode**.

### 1.4 Par carte

Ce n'est pas la carte. Sur les mêmes cartes, hors Oddball, la part de scan de septembre est basse :

| carte | hors Oddball | avec Oddball |
|---|---|---|
| Live Fire | 0,6 % | 35,3 % |
| Lattice | 1,0 % | 45,1 % |
| Recharge | 3,3 % | 28,0 % |

Toutes les cartes de septembre se résolvent au catalogue des bornes (carte obligatoire tenue). Seule
exception : **Banished Narrows** (§2.3), haute en Slayer comme en Strongholds. Elle n'est pas liée au
canevas Forge `fo05_desert`, que partagent aussi Sylvanus, Solution et Domicile :
- en septembre, Sylvanus et Solution ont 0 kill `scan` en `credit-concordant` au décodage local, et
  leurs 12 + 12 lignes `scan` en base sont des morts de bot ;
- tous mois confondus, ces trois cartes ne comptent que 6, 5 et 16 lignes `scan` en `credit-concordant`.

### 1.5 Par build

Tous les films de septembre sont en `HI_1_13_0` : la chaîne de build apparaît dans `chunk_00` des 121
films. C'est le build des films de décembre 2025 à août 2026, d'après la carte de fermeture J11.3 :
`c75f33b8`, `d9781168`, `51ebbc0f`, eux-mêmes des Oddball / One Bomb à 74 à 92 % de scan.

L'Oddball `60ae07c4` (`HI_1_8_0`, octobre 2024) est à 49 %. Il n'y a **aucune mise à jour du jeu en
cause**.

### 1.6 Par joueur suivi

| matchs | part de scan |
|---|---|
| 31 matchs avec au moins un joueur suivi (JGtm, Chocoboflor, Madina97294, XxDaemonGamerxX) | 4,7 à 11,9 % |
| 90 matchs sans participant suivi (`first_sync_by = Nuzzles`, Ranked Arena), qui contiennent les 19 matchs à objectif porté | 21,8 % |

La concentration « par joueur » est un **effet de sélection** : les Ranked Arena de ce compte
contiennent l'Oddball.

## 2. Pourquoi la marche ne lit pas ces kills (question 2)

Rappel du mécanisme (`facts/killsource/walk.go`). Dans un paquet type-0 à liste d'événements, la boucle
de records ne commence pas au bit 2. La liste d'événements (vue A, 123 genres) n'est pas portée.
`locateRecords` cherche donc :
- d'abord la signature stricte : bit précédent à 0, puis un delta sur le **slot 123**, long de
  **35 bits** et à composant unique ;
- puis un repli à largeur libre, toujours sur le slot 123.

Un paquet non localisé est **sauté** par la marche. Ses morts ne peuvent venir que du scan.

Les keyframes des quatre films lisent le slot 123 comme l'archétype **ti 4 `high-frequency`** : un
composant, « rafraîchi à chaque image ».

### 2.1 `8f7f5806` — Ranked:Oddball, Live Fire, 2026-09-04 (74,3 %)

Calibration `LU axisW=[12 12 11] [CARTE]`. Le film publie 226 lignes :
- 56 par la marche ;
- **167 par le scan**, en `credit-concordant` ;
- 3 d'autres origines.

Paquets à événements localisés : **3 051 / 9 403 = 32,4 %**, contre 96,3 % sur le témoin.

| cause, pour les 167 kills `scan` | kills |
|---|---|
| A — paquet à événements **non localisé** (aucune signature slot 123) | **153** |
| B — mort AVANT le début localisé (le repli à largeur libre a pris un « slot 123 » plus loin que le dead-state, `libre=true`) | **14** |
| E / F / G — désynchronisation, vues épuisées, décalage | 0 |

L'**essai** porte sur 167 kills. La marche de production relit le dead-state **au bit du scan** depuis un
départ antérieur pour **165**. Pour 2 kills, aucun départ ne convient. La chaîne lue depuis ce départ dit
pourquoi la signature manque. Exemples (slot/ti/longueur) :

```
359473 ms  [3/ti6/267b 4/ti6/115b 10/ti6/115b 12/ti6/267b 53/ti5/48b 127/ti4/35b 580/ti35/152b ...]
363876 ms  [3/ti6/115b 4/ti6/339b 6/ti6/153b 7/ti6/283b 9/ti6/107b 58/ti5/48b 127/ti4/35b 580/ti35/154b ...]
391220 ms  [2/ti2/64b 3/ti6/153b 4/ti6/265b 6/ti6/321b 12/ti6/153b 59/ti5/48b 127/ti4/35b 588/ti35/154b ...]
```

La chaîne enchaîne, par slots croissants :
1. les compteurs de score (statborg, ti 6), qui changent sans cesse en Oddball ;
2. le joueur (ti 5) ;
3. **le delta high-frequency de 35 bits, sur le slot 127** (166 occurrences) **ou 128** (138) ;
4. les bipèdes.

**Le slot 123 n'apparaît dans aucune de ces chaînes.** Les keyframes du film déclarent trois objets
high-frequency : les slots 123, 127 et 128.

### 2.2 `6b0e6f0f` — variante `4640310c`, Squad Battle, Refuge, 2026-09-11 (69,3 %)

Le film publie 255 lignes :
- 44 par la marche ;
- **209 par le scan** ;
- 2 d'autres origines.

Paquets à événements localisés : **4 142 / 17 302 = 23,9 %**.

| cause, pour les 209 kills `scan` | kills |
|---|---|
| A — paquet à événements non localisé | **196** |
| B — mort avant le début localisé | **11** |
| E — désynchronisation avant la mort : `ti=0 i18 forge-engine-player-roles-component`, non porté | 1 |
| G — marche passée sans lire le dead-state | 1 |

Même mécanisme que `8f7f5806`. Les deltas high-frequency de 35 bits sont portés par les slots **126,
127, 128 et 129** (43, 43, 35 et 37 occurrences sur les chaînes des morts), contre 2 seulement pour le
slot 123. Les keyframes déclarent cinq objets high-frequency : les slots 123 et 126 à 129.

### 2.3 `72b0a25e` — Slayer:Arena, Banished Narrows, 2026-09-01 (45,7 %)

Le film publie 81 lignes :
- 43 par la marche ;
- **36 par le scan** ;
- 2 d'autres origines.

La localisation est **normale** : 3 993 / 4 122 = 96,9 %. Les **36 kills** relèvent tous de la
**cause E** : la marche se désynchronise AVANT la mort.

| t (ms) | bit du dead-state | début localisé | slot qui casse | gén. | départ `s'` de l'essai | 1er record depuis `s'` |
|---|---|---|---|---|---|---|
| 63266 | 10704 | 9632 | 260 | 1 | 9879 | 512 (ti 35) |
| 89725 | 7236 | 6260 | 320 | 1 | 6401 | 512 (ti 35) |
| 114066 | 7065 | 6179 | 374 | 1 | 6293 | 1 (ti 34) |
| 136522 | 7955 | 7208 | 422 | 1 | 7349 | 514 (ti 35) |
| 138357 | 7192 | 6599 | 426 | 1 | 6740 | 514 (ti 35) |
| 143314 | 6090 | 5425 | 440 | 1 | 5566 | 514 (ti 35) |
| 144835 | 7572 | 7309 | 442 | 1 | 7450 | 514 (ti 35) |
| 156542 | 7147 | 6884 | 466 | 1 | 6998 | 2 (ti 2) |
| 178398 | 7249 | 6509 | 126 | 2 | 6650 | 533 (ti 35) |
| 196549 | 8260 | 7816 | 170 | 2 | 7957 | 533 (ti 35) |
| 203656 | 6943 | 6253 | 182 | 2 | 6394 | 533 (ti 35) |
| 212313 | 7379 | 6417 | 204 | 2 | 5092 | 35 (ti 6) |
| 212682 | 6374 | 5946 | 204 | 2 | 6087 | 533 (ti 35) |
| 217037 | 8279 | 7266 | 214 | 2 | 7407 | 533 (ti 35) |
| 221108 | 6939 | 6177 | 224 | 2 | 6318 | 533 (ti 35) |
| 224494 | 6351 | 5572 | 230 | 2 | 5713 | 533 (ti 35) |
| 234502 | 8258 | 7306 | 252 | 2 | 7447 | 533 (ti 35) |
| 240191 | 7108 | 6344 | 266 | 2 | 6485 | 533 (ti 35) |
| 254459 | 7632 | 7301 | 296 | 2 | 7442 | 541 (ti 35) |
| 280801 | 6588 | 5553 | 354 | 2 | 5694 | 541 (ti 35) |
| 287075 | 8454 | 7581 | 368 | 2 | 7722 | 541 (ti 35) |
| 303942 | 7534 | 6525 | 408 | 2 | 6666 | 551 (ti 35) |
| 305258 | 7288 | 7025 | 410 | 2 | 7166 | 551 (ti 35) |
| 317756 | 7085 | 6441 | 436 | 2 | 6555 | 5 (ti 6) |
| 330885 | 7295 | 6710 | 464 | 2 | 6957 | 565 (ti 35) |
| 362017 | 7455 | 7027 | 148 | 3 | 6796 | 2304 (ti 37) |
| 373646 | 6866 | 5969 | 172 | 3 | 6110 | 571 (ti 35) |
| 379718 | 7891 | 6678 | 184 | 3 | 6819 | 571 (ti 35) |
| 386958 | 8022 | 7759 | 202 | 3 | 7873 | 3 (ti 6) |
| 400423 | 7444 | 6098 | 232 | 3 | 6345 | 580 (ti 35) |
| 426081 | 6899 | 6337 | 290 | 3 | 6478 | 584 (ti 35) |
| 427833 | 6721 | 6458 | 292 | 3 | 6572 | 7 (ti 6) |
| 433640 | 7860 | 7491 | 304 | 3 | 7690 | 1656 (ti 37) |
| 436240 | 6355 | 6092 | 312 | 3 | 6206 | 1 (ti 34) |
| 440161 | 6978 | 6183 | 320 | 3 | 6324 | 591 (ti 35) |
| 446752 | 12217 | 10907 | 334 | 3 | 11048 | 591 (ti 35) |

**Lecture.**
- La signature du slot 123 est **juste**. Le record suivant commence exactement 35 bits plus loin
  (`hdr = s + 35`).
- Ce record suivant est un **delta** (`type=3`) sur un slot de **l'anneau des objets transitoires**
  (126 à 466). Sa génération monte de 1 à 3 à chaque tour de l'anneau.
- Ce slot n'est **lié par aucune image-clé**, ni avant (le monde) ni après (table anticipée du
  lot 5.23 : `ArchetypeApres` ne rend rien). L'objet naît et meurt entre deux images-clés.
- La marche du killsource ne retient jamais une liaison NEW. `walkPacket` restaure le monde après chaque
  paquet, et le localisateur saute les records NEW de tête (constat M4b). Le delta casse donc
  (`DesyncAt 0`, slot non lié).
- La marche s'arrête **avant les bipèdes**. Depuis `s'`, typiquement `s + 141` (soit 35 + 106 bits) ou
  `s + 114`, elle relit les 36 dead-states au bit du scan.

L'identité de l'objet **n'est pas établie**. La recherche des en-têtes NEW/DEL du slot dans les 30 s
précédentes est trop bruitée pour conclure.

Le même tableau se retrouve sur `e60aaf06` (Strongholds, Banished Narrows, 54,2 %) : **39/39** kills
`scan` en cause E, slots 222, 232, 274, 286, 320…, génération 1, non liés.

### 2.4 Témoin ancien : `d1a18847` — Slayer:Arena, Live Fire, 2026-02-19

Le film publie 91 lignes :
- 88 par la marche ;
- **0** `scan` en `credit-concordant` : la ligne `scan` en base est d'une autre origine.

Localisation : **3 491 / 3 624 = 96,3 %**. La carte (Live Fire) est celle de `8f7f5806` : l'écart
vient du mode.

### 2.5 Complément : `9c0ec856` — Ranked:Oddball, Recharge (27,7 %)

| cause, pour les 27 kills `scan` | kills |
|---|---|
| A | 26 |
| B | 1 |

Ici, le delta high-frequency est porté par le slot **124**. Les keyframes déclarent deux objets
high-frequency : les slots 123 et 124.

## 3. Expériences de localisateur (sur les mêmes paquets)

### 3.1 HF : signature sur tout slot high-frequency

La signature stricte est inchangée (35 bits, composant unique, génération stricte). Elle est acceptée
sur **tout slot dont l'archétype lié est ti 4 (high-frequency)**. La marche de production, inchangée,
repart de là.

Kills `scan` relus au bit du scan :

| film | relus par la marche HF | avec marche à inférence (HFW) |
|---|---|---|
| `8f7f5806` | **157 / 167** | – |
| `6b0e6f0f` | **180 / 209** | 192 / 209 |
| `9c0ec856` | **25 / 27** | – |
| `72b0a25e` | 0 / 36 | 0 / 36 |

HFW est la marche à inférence de la cuisson, `DecodeFrameViewsCurseur`, avec la table anticipée. Elle
ne rend rien à `72b0a25e` : la cause y est autre (§2.3).

Kills `marche` gardés par HF utilisé **seul** :

| film | gardés |
|---|---|
| `8f7f5806` | 55 / 56 |
| `6b0e6f0f` | 44 / 44 |
| `72b0a25e` | 39 / 43 |
| `d1a18847` (témoin) | 85 / 88 |

Les pertes sont des paquets que seul le repli à largeur libre du slot 123 localise. HF doit donc
**s'insérer** entre la signature stricte du slot 123 et le repli, sans les remplacer.

Résidu après HF :
- **`8f7f5806`, 10 kills** :
  - 3 non localisés ;
  - 5 désynchronisés sur un slot non lié (5288, 3053, 756, 7936) ;
  - 2 fins de marche avant la mort.
- **`6b0e6f0f`, 29 kills** :
  - 13 non localisés ;
  - 12 sur une **génération de bipède périmée** (slot lié en ti 35, `genOK=false` : réapparition entre
    deux images-clés) ;
  - 3 sur d'autres objets : génération périmée en ti 37 et en ti 17, et 1 slot non lié ;
  - 1 sur le composant `i59 biped-spartan-ability-non-predicted-state`, non porté.

### 3.2 Fermeture : premier départ qui ferme le paquet

Deux variantes ont été essayées :
- le premier départ d'où la vue C ferme le paquet (oracle `vueCFermee`) ;
- le premier départ d'où la marche finit sans désynchronisation.

Les deux sont **rejetées**. Le départ trouvé n'égale **jamais** le début localisé des kills de la marche
(0 sur 56 à 88). Il ne relit que 13 des 56 kills de la marche sur `8f7f5806`, et 0 en variante
« chaîne propre ». Ce ne sont pas des localisateurs.

## 4. Verdict (question 3)

### 4.1 Cause racine n° 1 — signature du localisateur figée sur le slot 123

Elle porte **85 % du scan de septembre**. Avant septembre, 70 % des kills des modes à objectif porté passaient déjà par le scan.

**Le défaut.** La boucle de records d'un paquet à événements commence bien par un delta de 35 bits
d'un objet **high-frequency**. Mais le slot 123 n'est que **l'un** de ces objets. Le film en déclare
plusieurs dès qu'il y a un objectif unique porté :
- 123, 127 et 128 en Oddball sur Live Fire ;
- 123 et 124 sur Recharge ;
- 123 et 126 à 129 dans la variante `4640310c`.

Dans ces modes, le delta de tête passe par un autre slot, et le slot 123 est absent du paquet. Ce n'est
ni une grammaire inconnue, ni un build non profilé, ni une carte : la **valeur lue** (l'archétype du
slot, déclaré par les images-clés) suffit. C'est un littéral (`123`) qui la remplace.

**Correctif proposé, de taille S en code et M avec ses gates.** Dans `facts/killsource/walk.go`
(`signature123`, `locateFallback`), accepter un slot dont l'archétype lié au monde est
« high-frequency ». L'index de cet archétype se lit **par son nom** dans le registre du film, jamais
`ti == 4` en dur : le registre est par build. L'ordre devient :
1. signature stricte du slot 123 ;
2. signature stricte sur un autre slot high-frequency ;
3. repli à largeur libre.

Le même littéral vit dans `grammar/object_deaths_march.go` (`marchSignature123`, `marchLocateFallback`).
Il est utilisé par la cuisson du rejeu : `debutDeLaListe`, états de mouvement, positions. Le rejeu perd
très probablement les mêmes paquets dans ces modes ; ce n'est **pas mesuré ici**. Les deux sites
doivent changer ensemble, sinon une troisième copie de la règle apparaît.

Ensuite :
- montée de `killsource.Rev` (et de la révision de grammaire si `grammar` change) ;
- G-corpus et banc de vérité ;
- goldens régénérés et relus ;
- vague `backfill-killsource`.

Le gain attendu est mesuré au §3 : **362 des 403 kills `scan`** des trois films (90 %) repassent par la marche, et 374 (93 %) avec la marche à inférence.

### 4.2 Cause n° 2 — objet transitoire non lié avant les bipèdes (Banished Narrows)

Elle touche environ 49 % des kills de 13 matchs, et 76 kills `scan` en septembre.

Ce n'est pas le localisateur. La marche casse sur le delta d'un objet né et mort entre deux images-clés,
dont killsource ne lit ni ne retient le NEW. C'est la limite connue du constat M4b (« objet né en milieu
de chunk ») : elle n'appartient ni à la classe des 14 exceptions de J6/R3, ni à un build.

Il faudrait :
- identifier l'objet (rétro-ingénierie : propre à cette carte, sans doute un objet de décor ou de
  script récurrent à chaque kill) ;
- puis faire retenir à la marche les liaisons des records NEW lus proprement d'un paquet à l'autre.

Le commentaire de `walkPacket` rappelle une mesure ancienne : les politiques qui conservaient des
liaisons perdaient (330 → 328 → 315 morts sur 372), sur 4 films et avant M4b. Taille **M/L**, à rouvrir
avec une mesure neuve.

### 4.3 Résidus mineurs

- génération de bipède périmée après réapparition (Squad Battle) ;
- deux composants non portés (`biped-spartan-ability-non-predicted-state` i59,
  `forge-engine-player-roles-component` i18).

### 4.4 Ce que valent les lignes `scan` déjà publiées

Elles sont `credit-concordant`, donc couplées exactement au kill-feed. Dans les expériences, la marche
correctement amorcée relit le dead-state **au même bit** que le scan (essai : 165/167 sur `8f7f5806`,
36/36 sur `72b0a25e` ; HF : 157/167). Le même bit donne le même champ, donc la même étiquette.

Le correctif changera surtout la **voie déclarée** (provenance) et la confiance attachée. Les valeurs
publiées devraient rester celles de la vague J11.4 pour ces kills. C'est à confirmer par le diff de la
vague corrective.

## 5. Découvertes (non traitées)

**a. Le localisateur de la cuisson porte le même littéral.** `grammar.marchLocateStrict` /
`marchSignature123` sont utilisés par `debutDeLaListe`, les états de mouvement et le tir continu. La
carte de fermeture J11.3 classe « liste d'événements non localisée » en cause n° 2 (48 720 paquets), avec
un gain déclaré nul. Ce gain nul est à revoir pour les modes à objectif porté.

**b. Origine `bot` = `scan` par construction.** `apparierMortDeBot` ne cherche que dans `c.scanCands`, et
`runBots` étiquette la ligne `PathScan` (`match.go`, `hybrid.go`). Les 1 499 lignes `bot` (51 en
septembre) ne mesurent donc aucune lecture affaiblie, mais elles gonflent la « part de scan ». Une mesure
de la règle « flux seul » doit les exclure, ou l'étiquette doit dire la voie réelle (le scan rattrape
aussi les records que la marche a lus).

**c. Carte `a54808fb-…` (CTF 3 Captures classé).** Elle n'est **pas** au catalogue des bornes par son
nom, alors que la production a décodé ses deux matchs de septembre (22,2 % et 0 % de scan). La
résolution de production passe par autre chose que le nom brut (identifiant d'asset ?). Non instruit.

**d. Variante `4640310c`.** Son nom n'est pas résolu en base (`game_variant_name` est l'identifiant).

## 6. Ce qui n'a pas été fait

- Aucun code de production modifié, aucun correctif prototypé hors du test jetable.
- Pas de mesure de l'impact sur le rejeu (§5.a), pas de G-corpus.
- Identité de l'objet transitoire de Banished Narrows non établie.
- `.ai/thought_log.md` et le plan non mis à jour : le brief limite le commit à ce seul fichier.
- Les deux outils jetables (`cmd/zz_enquete_scan_q`, `zz_enquete_scan_local_test.go`) ont été supprimés
  du worktree. `git status` ne montre que `.bin-audit/`, présent avant l'enquête.
