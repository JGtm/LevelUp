# MESURE — les trous de la fiche d'inventaire du rejeu 2D (2026-08-24)

> Branche `feat/v75`, aucun commit. Périmètre : **MESURER, pas corriger**. Aucune ligne de
> production n'est modifiée ; le test de recherche laissé au dépôt est gaté par variable
> d'environnement et sauté en CI.
>
> Corpus : 24 films du cache `data/cache/film_chunks/`, échantillonnés à pas constant sur les
> 951 préfixes triés, plus le film de vérité terrain `000d5950` (Cliffhanger, 8 joueurs).
> Total mesuré : **698 images-clés, 6 721 records de bipède (ti=35)**.
>
> Outil : `apps/go-api/internal/analysis/replay/inventory_trous_mesure_test.go`
> (+ `inventory_trous_rapport_test.go`), `TestInventaireTrousMesure`.
>
> Documents frères du même jour, à lire avec celui-ci : `AUDIT_AVAL_INVENTAIRE_2026-08-24.md`
> (ce qui jette une lecture DÉJÀ décodée) et `FAISABILITE_SUIVI_DELTA_INVENTAIRE_2026-08-24.md`
> (le suivi entre deux images-clés). Le présent document traite l'AMONT : ce que le décodeur
> ne lit pas.

---

## 0. Ce que la mesure a trouvé, en trois phrases

1. **L'ancre 28 bits de la capacité (`0x8CAC57A`) n'est pas un invariant du format.** 80,9 %
   des records de bipède n'en portent AUCUNE occurrence, et **11 films sur 24 n'en portent
   aucune sur la totalité du film**. Comme R2 (les grenades) part de la position de cette
   ancre, la capacité ET les grenades tombent ensemble : sur ces films, la fiche ne montre
   jamais que des munitions.
2. **La cause n'est PAS la fenêtre borgne 16..23**, contrairement à ce que
   `inventory_decode.go` suppose. Trois élargissements ont été testés et **les trois sont
   réfutés par la mesure** (§4) — dont un par un témoin de hasard qui dépasse le signal.
3. **Le symptôme « la fiche ne donne RIEN par moments » a une cause AVAL distincte et
   directe** : un record sans arme produit un inventaire VIDE qui est publié comme une
   lecture, et le client, qui retient la dernière lecture ≤ T, fait disparaître la ligne
   entière pendant ~20 s. 17,4 % des lectures publiées sont dans ce cas.

---

## 1. La taxonomie chiffrée — catégories exclusives (première règle qui tombe)

Sur 6 721 records de bipède, 698 images-clés, 24 films.

| catégorie | définition | n | % des records |
|---|---|---:|---:|
| (b1) ancre absente | aucune occurrence de l'ancre 28 bits dans le record | **5 440** | **80,9 %** |
| (b2) motif absent | ancre présente, motif 20 bits hors des 60 bits suivants | 0 | 0,0 % |
| (c) ancre non unique | deux lectures d'ancre, non départagées | 5 | 0,07 % |
| (d) R2 échoue | pas de motif i22 de somme > 0 après l'ancre | 104 | 1,5 % |
| (e) R3 échoue | aucune famille d'arme reconnue | 0 | 0,0 % |
| (f) R4 échoue | aucun début de bloc de munitions n'atterrit | 5 | 0,07 % |
| (g) succès partiel | (aucun : tout échec est déjà classé plus haut) | 0 | 0,0 % |
| (h) succès complet | capacité + grenades + munitions + dégainé | **1 167** | **17,4 %** |

Hors taxonomie exclusive, le détail de ce qui MANQUE par record :

| champs manquants | n | lecture |
|---|---:|---|
| `capacité + grenades` | 4 278 | record ARMÉ, munitions et dégainé lus — le trou est R1/R2 seuls |
| `capacité + grenades + munitions + dégainé` | 1 167 | **fiche entièrement vide** |
| aucun (lecture complète) | 1 167 | |
| `grenades` seules | 102 | R1 passe, R2 tombe |
| `munitions + dégainé` | 5 | R4 tombe |
| `grenades + munitions + dégainé` | 2 | |

Catégorie (a) — record de bipède ABSENT pour un slot pourtant vivant, « vivant » établi par
ENCADREMENT (le slot apparaît à une image-clé avant ET à une image-clé après) : **67 cas sur
698 images-clés**, soit 0,1 record manquant par image-clé. **Ce n'est pas la cause des fiches
vides.** À côté : **58 images-clés sur 698 (8,3 %) ne portent AUCUN record de bipède** — un
paquet d'image-clé sans monde reconstruit, à instruire séparément.

Distribution du nombre de bipèdes par image-clé (24 films) : `b8=401` domine (matchs à 8
joueurs), avec une seconde bosse `b21..b27` (matchs à grande équipe) et `b0=58`. **Aucun
film ne dépasse le nombre de joueurs attendu** : les records « sans arme » ne sont donc pas
des cadavres surnuméraires, ce sont les bipèdes courants des joueurs.

### Tableau catégorie x film

| film | kf | records | b1 | c | d | f | h | fiches vides | ammo multi-parse |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 000d5950 | 26 | 184 | 52 | 0 | 12 | 0 | 120 | 34 | 51 |
| 0873c469 | 30 | 227 | **227** | 0 | 0 | 0 | 0 | 26 | 25 |
| 121be2d6 | 27 | 111 | 29 | 0 | 3 | 0 | 79 | 25 | 21 |
| 1c5c10cc | 37 | 718 | 690 | 0 | 0 | 0 | 28 | 99 | 52 |
| 2a4bc093 | 22 | 163 | **163** | 0 | 0 | 0 | 0 | 37 | 9 |
| 34dac77d | 20 | 150 | 39 | 0 | 4 | 0 | 107 | 33 | 44 |
| 4013dc34 | 29 | 196 | **196** | 0 | 0 | 0 | 0 | 34 | 4 |
| 4c8d2287 | 31 | 231 | **231** | 0 | 0 | 0 | 0 | 34 | 7 |
| 53ce4390 | 39 | 300 | **300** | 0 | 0 | 0 | 0 | 70 | 11 |
| 5e121187 | 34 | 233 | 78 | 2 | 30 | 1 | 122 | 52 | 59 |
| 66aa2908 | 21 | 150 | 47 | 0 | 8 | 0 | 95 | 40 | 33 |
| 71ad4abd | 23 | 151 | **151** | 0 | 0 | 0 | 0 | 45 | 7 |
| 7da6e3f0 | 27 | 196 | **196** | 0 | 0 | 0 | 0 | 36 | 15 |
| 8778233d | 30 | 225 | **225** | 0 | 0 | 0 | 0 | 38 | 15 |
| 97b34406 | 37 | 780 | 693 | 0 | 6 | 1 | 80 | 111 | 194 |
| a26dbcdb | 34 | 704 | 672 | 1 | 3 | 0 | 28 | 113 | 59 |
| ac041ffa | 28 | 105 | 39 | 1 | 3 | 0 | 62 | 26 | 23 |
| b7b37365 | 31 | 232 | **232** | 0 | 0 | 0 | 0 | 39 | 19 |
| c1b4391a | 26 | 190 | 58 | 1 | 6 | 1 | 124 | 37 | 46 |
| cbd4f623 | 34 | 254 | **254** | 0 | 0 | 0 | 0 | 43 | 19 |
| d40afcfb | 26 | 192 | **192** | 0 | 0 | 0 | 0 | 34 | 10 |
| dd3095de | 24 | 163 | 36 | 0 | 5 | 1 | 121 | 28 | 46 |
| e67ad176 | 31 | 628 | 582 | 0 | 2 | 0 | 44 | 108 | 100 |
| f1e41f31 | 31 | 238 | 58 | 0 | 22 | 1 | 157 | 27 | 66 |
| **TOTAL** | **698** | **6 721** | **5 440** | **5** | **104** | **5** | **1 167** | **1 169** | **935** |

**Le régime est BIMODAL, et c'est le fait le plus parlant du tableau.** Onze films (en gras)
rendent `h_complet = 0` : sur eux, PAS UN SEUL record ne porte l'ancre, donc pas une seule
capacité ni une seule grenade de tout le match. Les treize autres tournent entre 60 % et
75 % de lectures complètes. Il n'y a pas de continuum : un film porte l'ancre, ou il ne la
porte pas. Ce n'est pas le comportement d'une règle qui « rate parfois », c'est celui d'une
constante qui n'en est pas une.

---

## 2. La cause n°1 des FICHES VIDES est en aval, et elle est certaine

**1 169 records sur 6 721 (17,4 %) ne rendent RIEN** — ni grenade, ni munition, ni sélecteur.
De ces 1 169, **1 131 (96,7 %) sont des records SANS AUCUNE ARME** : R3 ne trouve aucune
famille d'arme connue, donc R4 n'a pas de borne droite et ne lit rien. Sur le film de vérité
terrain, ce chiffre est de 34/34 : **zéro fiche vide avec une arme au record**.

Interprétation la plus économique — un bipède qui ne porte aucune arme à cet instant est un
joueur MORT ou en attente de réapparition. Elle est **cohérente mais NON PROUVÉE ici** : le
croisement avec le fil des morts (`deaths_source.go`) n'a pas été fait, et il devrait l'être
avant tout correctif (§6, piste 2).

**Ce que ce record devient à l'écran, vérifié sur pièces :**

- `buildInventory` (`replay/inventory.go:92-121`) publie l'entrée **inconditionnellement** :
  `out = append(out, inv)` en fin de boucle, même quand `GrenadesRead`, `AmmoRead` et
  `DrawnSlot` sont tous absents. Le document sort donc un `{"t":N,"slot":S}` nu.
- `nearestReading` (`apps/web/src/features/match-replay/rosterLogic.ts:373-391`) retient **la
  lecture d'âge minimal ≤ frame**. L'entrée vide, étant la plus récente, **gagne contre la
  lecture pleine qui la précédait**.
- `ReplayInventoryRow.tsx:79` : `if (grenades.length === 0 && !ability && ammo.length === 0)
  return null`. **La ligne entière disparaît**, et elle reste disparue jusqu'à l'image-clé
  suivante, soit ~20 s.

C'est exactement le symptôme rapporté : « la fiche de certains joueurs ne donne rien par
moments ». Une lecture vide n'est pas une absence de lecture : **elle EFFACE**.

---

## 3. La cause n°1 des trous de CAPACITÉ et de GRENADES : l'ancre n'est pas un invariant

4 278 records (63,7 %) sont ARMÉS — munitions et dégainé correctement lus — mais rendent ni
capacité ni grenades. La fiche y montre des munitions et rien d'autre.

### 3.1 Le couplage R2 → R1 double la perte

`keyframeInventories` (`inventory_decode.go:231-238`) n'entre dans R2 **que si R1 a rendu
exactement une ancre** : la recherche de i22 démarre à `hits[0].anchorBit`. La conséquence
est mécanique et elle est le vrai multiplicateur du trou : **toute défaillance de R1 coûte
AUSSI les grenades**, alors que le champ i22 n'a, lui, rien à voir avec la capacité.

Le prix payé est celui d'une garantie d'indépendance que le fichier revendique explicitement
(« la position de l'ancre vient de R1, donc sans aucune information de grenade »). La
garantie est réelle ; son coût, mesuré, est de 4 278 records.

### 3.2 Le canal i48 ne comble que la moitié du trou

`ScanFilmAbilityRanks` (paquets delta) publie le rang complet et il est fiable : quand R1 et
i48 parlent tous les deux, **915 accords contre 74 désaccords (92,5 %)**. La capacité est
donc largement rattrapée à l'écran par `doc.Abilities`.

**Les grenades, elles, n'ont aucun second canal.** C'est pourquoi une fiche peut afficher une
capacité (venue d'i48) sans afficher une seule grenade.

---

## 4. Ce qui a été creusé au bit près — et ce que ça réfute

Quatre hypothèses ont été mises à l'épreuve. Aucune ne survit.

### 4.1 « La fenêtre de 60 bits est trop courte » — RÉFUTÉ

Le motif 20 bits a été recherché jusqu'à **400 bits** après chaque ancre trouvée. Sur les
6 721 records : **0 récupération**. Quand l'ancre est là, le motif est dans les 60 bits ou il
n'y est nulle part.

### 4.2 « L'ancre est trop stricte d'un bit » — le fait est réel, le correctif ne l'est pas

Sur les 5 440 records sans ancre, **5 065 (93,1 %) portent une ancre à distance de Hamming 1**,
et **le bit qui diffère est TOUJOURS le même : le rang 26** (l'avant-dernier des 28 ; l'ancre
`0x8CAC57A` y porte un `1`, la variante `0x8CAC578` un `0`). Histogramme complet sur 5 102
occurrences : `b26=5102`, **zéro à tout autre rang**.

Ce n'est pas du hasard, et le dénombrement le dit : 28 variantes d'un mot de 28 bits, c'est
une chance sur 9,6 millions par position ; un record de 5 000 bits en attend 0,0005. En
trouver une dans 93 % des records **est une structure**, pas une coïncidence. L'un des 28
bits de l'« ancre » n'est donc pas constant.

**Mais l'élargir ne rend rien.** Après l'ancre-variante, le motif 20 bits n'apparaît dans les
60 bits suivants **dans aucun record** (compteur « ancre H1 suivie d'un motif rendant un rang »
= 0 sur les 24 films). Les occurrences du motif présentes ailleurs dans ces records se
trouvent à des offsets NÉGATIFS et instables par rapport à la variante — mesuré sur
`000d5950` : `-518`, `-623`, `-667`, `-731`, `-836`, `-880`, avec des paires régulièrement
séparées de 213 bits (structure d'alias), et une stabilité par slot (slot 517 : ancre-variante
à +1465 du début de record et motif à +975, identique aux chunks 3, 4 et 5).

**Conclusion : le bit 26 est un champ, pas une signature — mais relâcher ce bit ne restitue
pas la capacité.** Ce qui suit la variante n'a pas la grammaire attendue.

### 4.3 « Le canal est borgne : hors des rangs 16..23 le motif diffère » — RÉFUTÉ

Le motif 20 bits vaut `[17 bits fixes][010]`, et ces trois derniers bits sont les bits de
poids fort du rang (`invAbilityRankHigh`). L'hypothèse naturelle est donc : chercher le seul
préfixe 17 bits, lire 6 bits (haut puis bas), obtenir le rang complet sur toute la palette.

- **Sur les 5 440 records sans ancre : 1 sauvetage.** Un seul record rend un rang unique par
  cette lecture, et i48 le **contredit** (accord 0, désaccord 1).
- **Le témoin de hasard tue l'hypothèse, sur tout le corpus.** Parmi les **2 992 records où R1
  tombe et où i48 donne un rang de référence** : **328** portent quelque part une occurrence
  du préfixe suivie du BON rang — mais le même test avec un rang décalé de 1 (rang+1, qui ne
  peut pas être la bonne réponse) en donne **443**. Le témoin de hasard est **plus fort que le
  signal**. Sur le seul film de vérité terrain, l'écart est du simple au double (15 contre 31
  sur 44).
- Les offsets où ces « bonnes » lectures tombent ne concentrent nulle part : le plus fréquent
  est `-852` avec **29 cas sur 328**, suivi de `-693:26`, `-1458:20`, `-639:19`, `-595:18`.
  Aucune position n'émerge — c'est la signature d'un bruit, pas d'un champ.

Le préfixe 17 bits (15 zéros puis `10`) est trop faible : il apparaît 9 à 20 fois par record.
**La capacité aux images-clés ne se rattrape pas par motif.**

### 4.4 « R2 rate parce que le joueur n'a pas de grenade » — CONFIRMÉ, 104 sur 104

R2 exige `somme > 0` sur les quatre compteurs i22. Or **dans 104 des 104 records classés (d),
un motif i22 de somme NULLE existe bel et bien après l'ancre** — `R(3)=4` suivi de quatre
`R(8)` tous bornés à 2, tous nuls.

Un Spartan qui a lancé toutes ses grenades produit exactement ce motif, et la règle le rejette.
« Zéro grenade » — une MESURE — devient indistinguable d'une non-lecture. Le film de vérité
terrain est net : 12/12. Sur les 24 films : 104/104.

Exemples relevés (`000d5950`, offsets relatifs au début du record) :

| chunk | slot | ancre à | rang R1 | i48 | i22 somme nulle | munitions |
|---|---|---|---|---|---|---|
| 5 | 523 | +1227 | 21 | 21 | oui | 1 candidat, dégainé 1 |
| 6 | 523 | +1227 | 21 | 21 | oui | 1 candidat, dégainé 1 |
| 6 | 526 | +1465 | 19 | 19 | oui | 1 candidat, dégainé 0 |
| 10 | 541 | +1394 | 22 | 22 | oui | 1 candidat, dégainé 0 |
| 14 | 561 | +1490 | 20 | 20 | oui | 3 candidats, dégainé 0 |

R1 lit bien, i48 confirme au rang près, les munitions passent : **seule la borne `somme > 0`
fait tomber la ligne des grenades.**

---

## 5. Ce que la mesure ne dit PAS

- **Que les records sans arme sont des morts.** C'est l'interprétation la plus économique, pas
  une mesure. Le croisement avec le fil des morts reste à faire.
- **Pourquoi le bit 26 varie.** On sait qu'il varie, toujours au même rang, et que le relâcher
  ne restitue rien. On ne sait pas ce qu'il porte.
- **Ce qui distingue les 11 films « sans ancre » des 13 autres.** Aucune corrélation n'a été
  cherchée avec la carte, le mode ou la version de réplication. C'est la mesure la plus
  rentable à faire ensuite.
- **Les 8,3 % d'images-clés sans aucun record de bipède** (58/698) : non instruits ici.
- **Le départage des 935 blocs de munitions multi-parses** (13,9 %) : non instruit ici.
- **Pourquoi un record sans ancre est DEUX FOIS plus court.** Largeur moyenne mesurée :
  **12 697 bits avec ancre (n=1 281) contre 6 322 bits sans (n=5 440)**. Le record sans ancre
  porte moitié moins d'état. C'est un indice fort — la même entité n'écrit pas la même chose
  selon les films — mais il n'a pas été instruit.

### Reproductibilité

Le corpus a été balayé DEUX FOIS, la seconde après ajout d'un compteur de chunks illisibles et
sans aucun autre processus concurrent. Les deux passes rendent des totaux **identiques au
record près** (6 721 records, 5 440 en b1, 104 en d, 1 167 complets, 1 169 fiches vides,
935 multi-parses), et **0 chunk illisible sur les 24 films**. La mesure est stable.

---

## 6. Pistes de correctif, classées par gain estimé (aucune n'est implémentée)

### Piste 1 — Découpler R2 de R1 : les grenades sans passer par l'ancre de capacité

- **Gain** : jusqu'à **4 278 records (63,7 %)** retrouveraient leur ligne de grenades — le plus
  gros gain unitaire mesuré.
- **Coût** : moyen. Il faut un nouveau critère de position pour i22, l'ancre de capacité ayant
  été jusqu'ici son seul garant d'unicité.
- **Risque, et il est sérieux** : R2 perdrait son indépendance revendiquée. Le remplacement
  doit être validé par un ORACLE extérieur — les lancers de grenade (`doc.Grenades`, décodés
  par un canal disjoint) donnent le type porté à l'instant du lancer : un compteur i22 qui
  décroît sur ce type, à cet instant, est une confirmation qui ne doit rien à R2.
- **Mesure préalable** : histogramme des offsets ancre → i22 sur les 1 167 records complets.
  S'il concentre, la position de i22 se borne par la structure du record plutôt que par
  l'ancre.

### Piste 2 — Ne pas laisser une lecture VIDE effacer la fiche

- **Gain** : **1 169 lectures (17,4 %)** cessent d'effacer ~20 s d'affichage. C'est le
  correctif qui vise DIRECTEMENT le symptôme rapporté par l'utilisateur.
- **Coût** : faible côté Go (`buildInventory`), faible côté web.
- **Ce n'est pas « ne rien publier »**, et c'est le point délicat : si ces records sont bien
  des bipèdes morts, la bonne réponse produit est de **dire « mort / sans équipement »**, pas
  de laisser survivre en silence un inventaire de vingt secondes d'âge. Prérequis : le
  croisement avec le fil des morts (§5). Tant qu'il n'est pas fait, le correctif honnête est
  de publier la lecture avec un marqueur explicite « aucun équipement porté », que le client
  affiche au lieu de disparaître.
- **À rapprocher** de `AUDIT_AVAL_INVENTAIRE_2026-08-24.md` point 6, qui décrit la disparition
  de la ligne par ABSENCE de lecture ; le présent point décrit la disparition par lecture
  VIDE. Les deux mènent au même `return null`.

### Piste 3 — R2 : accepter la somme nulle

- **Gain** : **104 records (1,5 %)**, dont 12 sur le film de vérité terrain. Petit volume,
  mais **100 % explicable et 100 % reproductible** — c'est la seule cause entièrement close
  de ce rapport.
- **Coût** : faible.
- **Risque** : le motif `R(3)=4` + quatre `R(8)` nuls est fréquent dans du remplissage nul ;
  accepter la somme nulle telle quelle ouvrirait des faux positifs. Il faut lui substituer une
  borne de POSITION (offset compatible avec ceux mesurés sur les records qui réussissent) et
  non une borne de VALEUR.
- **Effet de bord à ne pas manquer** : un tableau `G` de quatre zéros doit s'afficher comme
  « aucune grenade », pas comme « non lu » — `grenadesCarried` écarte déjà les compteurs nuls
  de l'affichage, la ligne resterait donc vide de puces. À trancher côté produit.

### Piste 4 — La marche ECS du record d'image-clé (chantier R7)

- **Gain** : potentiellement **total** — capacité, grenades, sélection, munitions lues par
  grammaire au lieu de motif, sur les 6 721 records.
- **Coût** : **très élevé, et déjà engagé**. `PLAN_R7A..R7E` mesurent l'atterrissage bit-exact
  du corps d'image-clé du bipède ; R7-d en est à 0,85 % (5 records sur 591). C'est la bonne
  réponse structurelle, ce n'est pas une réponse à court terme.
- **Ce que ce rapport lui apporte** : la preuve que les voies par motif sont épuisées (§4), et
  un dénominateur d'enjeu chiffré (63,7 % des records armés).

### Piste 5 — Comprendre le régime bimodal (mesure, pas correctif)

- **Gain** : indirect, mais c'est la mesure la moins chère du lot. Onze films sur 24 ne
  portent AUCUNE ancre. Corréler ce régime avec la carte, le mode, la date du film et la
  version de réplication (`filmdec.LastRepVersion`) dirait si l'ancre est propre à une
  version du jeu — auquel cas la « constante » est en réalité datée, et le décodeur doit la
  choisir, pas la supposer.
- **Coût** : très faible (le test de recherche est déjà écrit, il suffit d'y joindre le
  manifeste du film).

### Piste 6 — Le bit 26 de l'ancre (non prioritaire)

- **Gain mesuré : nul.** La variante existe dans 93 % des records sans ancre, toujours au même
  rang de bit, mais rien d'exploitable ne la suit. À conserver comme INDICE de structure pour
  le chantier R7, pas comme correctif.

---

## 7. Reproduire la mesure

```bash
# depuis apps/go-api — le film de vérité terrain, avec exemples creusés au bit près
CGO_ENABLED=0 INV_FILMS=<repo>/data/cache/film_chunks/000d5950 INV_DIG=8 \
  go test ./internal/analysis/replay/ -run '^TestInventaireTrousMesure$' -timeout 30m -v

# le corpus échantillonné (pas constant sur les préfixes triés), TSV en sortie
CGO_ENABLED=0 INV_CACHE=<repo>/data/cache/film_chunks INV_SAMPLE=24 INV_DIG=0 \
  INV_OUT=<dossier> \
  go test ./internal/analysis/replay/ -run '^TestInventaireTrousMesure$' -timeout 180m -v
```

`INV_I48=0` coupe le contrôle croisé par le canal i48 (il redécode tous les paquets delta et
coûte l'essentiel du temps de balayage). `INV_FILMS` accepte une liste séparée par des
virgules.

**UN SEUL balayage à la fois par machine.** Deux processus concurrents sur le même cache ont
produit, en cours d'étude, un film à zéro image-clé (`ReadFilmChunk` échoue et la boucle
continue). Le compteur `chunks illisibles` a été ajouté pour que cette panne ne puisse plus
se lire comme « aucun trou ».
