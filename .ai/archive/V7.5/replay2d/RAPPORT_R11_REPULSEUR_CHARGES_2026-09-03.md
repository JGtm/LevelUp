# RAPPORT R11 — Le REPULSEUR avec une ANCRE : trois charges au ramassage

Date : 2026-09-03. Lot R11, suite de R8 (propulseur trouve) et R9 (huit canaux fermes sans
ancre). Retro-ingenierie STATIQUE. Aucune DuckDB ouverte, aucun debogueur, aucun commit,
aucune ecriture sur la production, aucun serveur Ghidra lance (il n'a pas ete necessaire).

> Rapport ECRIT AU FIL DE L'EAU. **[ETABLI]** = mesure faite ; **[REFUTE]** = hypothese tuee
> par la mesure. Une interruption ne doit rien couter.

---

## Verdict en cinq phrases

**LE CANAL DES CHARGES EXISTE, IL EST TROUVE, ET IL NE PORTE PAS LE REPULSEUR.** **Le
composant bipede i56 `biped-spartan-ability-energy` transmet un COMPTEUR DE CHARGES ENTIERES
— le quartet haut de sa valeur 7 bits — et sa decroissance DATE chaque usage : sur
`1cd3848a`, la serie de JGtm vaut 4, 3, 2, 1, 0 aux instants 1:52, 1:55, 2:03, 2:05, 2:15,
c'est-a-dire EXACTEMENT les cinq usages de propulseur que l'utilisateur a releves au Theater
(1:51, 1:54, 2:03, 2:05, 2:14) — precision 5/5, rappel 5/5.** **Les deux temoins positifs
pre-inscrits passent tous les deux : 40 baisses sur 40 coincident avec une impulsion i57/i59
(le canal de R8), et 36 accroches de grappin sur 36 (canal `grappleLines[]`, totalement
independant) sont appariees a une baisse, contre 2 sur 36 pour le temoin decale de 5 s.**
**Sur le repulseur, le meme canal rend ZERO : 218 vies de repulseur, 111,7 minutes de port
mesurees sur onze films, aucune baisse de charge attribuable — et dans les SIX films dont la
palette ne porte ni grappin ni propulseur, i56 n'est JAMAIS arme (485 lectures, masque a 000
sur toutes), alors meme que ces films sont pleins de porteurs de repulseur.** **Le cas qui
tranche : sur `a6ae19fb`, un joueur porte le repulseur de 6:21 a 7:29 et le film lui-meme
annonce a 7:29 qu'il vient de consommer sa DERNIERE charge (i48 `spent` depuis le rang
repulseur) — pendant ces 68 secondes i56 n'a pas transmis un seul emplacement arme, alors
qu'il comptait les charges du grappin de trois autres joueurs dans le meme film.**

---

## 0. Ce qui change par rapport a R9 : L'ANCRE

Releve Theater de l'utilisateur (2026-09-03), film `72b0a25e`
(`72b0a25e-c94d-42c0-85ca-195a320c7b73`) :

> **JGtm ramasse un REPULSEUR sur son socle a 2:46, avec 3 CHARGES.**

C'est la premiere contrainte NUMERIQUE jamais disponible sur le repulseur. R9 jugeait huit
canaux contre un plancher de bruit, sans savoir ni quand ni combien. Ici on sait QUI (JGtm,
xuid `2533274823110022`), QUAND (2:46, horloge validee), et **COMBIEN (3)**.

Le meme releve valide l'autre moitie du chantier : sur `1cd3848a`, les 5 usages de propulseur
notes par l'utilisateur (1:51, 1:54, 2:03, 2:05, 2:14) correspondent aux 5 impulsions mesurees
(1:52, 1:55, 2:03, 2:05, 2:15) — **precision 5/5, rappel 5/5**. L'horloge et la methode sont
donc fiables ; c'est le canal du repulseur qui manquait.

### 0.1 [ETABLI] Ce que l'artefact dit de la fenetre de l'ancre — et une CORRECTION

`data/cache/replays/halo_infinite/72b0a25e.json` — `originMs` 38 551, `frameIntervalMs` 100.
Palette du film (`abilityLabels`) : **2 = mur de protection, 6 = REPULSEUR, 8 = camouflage
actif**, resolue PAR LA PALETTE et non par un numero fige (par. 0.2). Pistes de JGtm :
slots 512, 533, 554, 572, 581, 584, 593.

Canal i48 pour la vie concernee, slot **533** (frames 979..2013, soit 2:16 -> 3:59) :

| frame | mm:ss | i48 | i26 `unit-equipment` (canal independant) |
|---|---|---|---|
| 1285 | **2:47** | `taken` rang 6 (**repulseur**), compteur 5 | liste = {1370, **1460**} — l'entree 1460 APPARAIT |
| 1335 | **2:52** | `taken` rang 8 (camouflage), `from` 6, compteur 6 | liste = {1370, **1468**} — 1460 DISPARAIT |
| 1371 | **2:55** | `spent` (porte ouverte), compteur 7 | liste = {1370} |

**CORRECTION A LA FICHE DE CRENEAUX, ecrite et non gommee.**
`CRENEAUX_VERIFICATION_EQUIPEMENT_2026-09-03.md` par. 2.c annonce pour ce film une plage de
port « 2:47 -> 3:59 (73 s) puis vous prenez un camouflage ». **Deux canaux independants du
film disent autre chose** : i48 (le rang porte) et i26 (la liste des objets tenus) placent
tous deux l'echange contre le camouflage a **2:52**, et 3:59 n'est que la FIN DE LA VIE du
slot 533. La plage de 73 s de la fiche etait « du ramassage a la fin de la vie », pas « du
ramassage a l'echange ». **La fenetre de port du repulseur par JGtm est donc de 5 secondes,
pas de 73**, ce qui la rend trop courte pour conclure a elle seule — d'ou le poids donne au
par. 5 (un porteur qui, lui, consomme ses trois charges).

`padPickups` de JGtm dans ce film : frames 301 (1:08), 1904 (3:48), 3878 (7:06) — **aucun a
2:46**. Le canal des socles ne voit donc pas ce ramassage : son rappel est incomplet, et il
ne peut pas servir a dater.

### 0.2 Consigne de couverture (message du coordinateur, integree et tenue)

- Il existe **deux variantes d'objet propulseur** (GlobalID `0x430dda48`, classique, 23 films ;
  `0xeef5d48d`, dont `1cd3848a` — la variante de l'utilisateur). Le repulseur n'a qu'un seul
  GlobalID sur tout le parc (`0x7ca85adc`).
- **La palette ne nomme jamais deux rangs « propulseur » dans un meme film** : un rang par
  equipement, quelle que soit la variante. Le rang i48 est donc VARIANT-AGNOSTIQUE.
- **Tenu par tous les instruments de ce lot** : la resolution d'un equipement passe par le NOM
  dans la palette du film (`abilityLabels`, helper `r9RankOf`), jamais par un numero fige.
- **Preuve de couverture, mesuree ici** : la verite terrain de l'utilisateur porte sur
  `1cd3848a`, ou le propulseur est au **rang 21 (famille B)** et le grappin au rang 20 ; les
  films de R8/R9 le portent au **rang 5 (famille A)**, grappin rang 4. **Les deux familles
  emettent la baisse de charge** (40 usages en famille B sur `1cd3848a` ; 9 et 5 usages en
  famille A sur `00ba2e1c` et `084a804d`). Un litteral en dur aurait rendu la famille B muette
  — le piege exact qui avait rendu le translocateur invisible hors famille A.

---

## 1. Pre-inscription (ECRITE AVANT TOUTE MESURE, reproduite telle quelle)

### 1.1 Les trois gisements candidats

**(A) i56 `biped-spartan-ability-energy` — CANDIDAT PRINCIPAL.** Grammaire portee
(`ability_energy.go`, desassemblage relu le 2026-07-26) : `R(3)` MASQUE puis **7 bits par
emplacement arme**, soit **trois emplacements de charge**. Et l'en-tete d'`i56_drops_test.go`
consigne, du consommateur `FUN_140F8F300`, que la meme valeur 7 bits se lit de deux facons :
continu `v / 127.0f`, ou discret **`(v >> 4) & 0xF` charges ENTIERES** plus `(v & 0xF)` de
recharge fractionnaire. Trois emplacements, un quartet de charges entieres, une ancre a 3 :
c'est la forme exacte de ce qu'on cherche. R9 a ferme i56 sur un RECENSEMENT D'ANNONCES
(par. 7.5) en ecrivant lui-meme la limite de cet instrument : « un usage transmis comme une
VALEUR a l'interieur d'un composant [...] n'apparait pas dans un recensement d'ANNONCES ».
Les VALEURS d'i56 n'avaient jamais ete lues sur une vie ancree.

**(B) le compteur `R(3)` d'i48** — trois bits, 0..7, jamais confrontes a une semantique.

**(C) `equipment-charges-remaining` (i27) de l'entite ti=37 PORTEE** — report 2 du registre R9.

### 1.2 Ce que j'attendrais si (A) etait le canal — seuils fixes AVANT la mesure

1. **Ancre de valeur** : au moins une lecture d'i56 du slot 533 dans **[2:46, 2:56]** dont un
   emplacement porte un quartet haut egal a **3** (`v` dans 48..63).
2. **Decroissance** : la serie de cet emplacement ne remonte pas et decroit par pas de 1.
3. **TEMOIN POSITIF n°1 — PROPULSEUR, verite terrain utilisateur** : sur `1cd3848a`, des
   baisses a moins de 1,5 s des 5 impulsions ; **au moins 3 des 5** appariees, contre **au
   plus 1** pour un temoin decale de +5 s.
4. **TEMOIN POSITIF n°2 — GRAPPIN** : appariement baisse <-> `grappleLines[]` (±1,5 s)
   ecrasant d'un facteur **>= 3** le meme taux sur un temoin decale de ±5 s.
5. **CONTROLE NEGATIF** : hors des fenetres de port du repulseur, pas de motif
   « 3 puis decroissance » pour le meme slot.
6. **REFUTATION ACTIVE** : toute decroissance trouvee sera confrontee a (i) est-ce vraiment un
   quartet 3 -> 2 -> 1 -> 0 ? (ii) tombe-t-elle dans la fenetre et nulle part ailleurs ?
   (iii) le meme emplacement fait-il la meme chose pour le grappin, qui a aussi des charges ?

**Si les temoins 3 et 4 echouent, aucun negatif ne sera publie sur le repulseur a partir
d'i56.**

### 1.3 Ce que j'attendrais si (B) etait le canal

Le compteur `R(3)` d'i48 vaut **3** a la lecture du 2:47 et decroit ensuite. Refutation
immediate s'il se comporte en compteur de CHANGEMENTS (increment a chaque `taken`, y compris
pour un equipement sans charge, et bouclage sur 8).

---

## 2. [ETABLI — LE CANAL EST TROUVE] i56 porte un COMPTEUR DE CHARGES

Instrument : `r11_journal_research_test.go`, `TestR11Journal`. Film `1cd3848a`, slot 525
(JGtm), journal brut, une ligne par lecture :

```
1:40  slot=525  i48  compteur=5 rang=21 (propulseur)
1:52  slot=525  i56  masque=001 e0[ 64=4/0]      1:52 i57 tag=1   1:52 i59 tag=1
1:55  slot=525  i56  masque=001 e0[ 48=3/0]      1:55 i57 tag=1   1:55 i59 tag=1
2:03  slot=525  i56  masque=001 e0[ 32=2/0]      2:03 i57 tag=1   2:03 i59 tag=1
2:05  slot=525  i56  masque=001 e0[ 16=1/0]      2:05 i57 tag=1   2:05 i59 tag=1
2:15  slot=525  i56  masque=001 e0[  0=0/0]      2:15 i57 tag=1   2:15 i59 tag=1
2:15  slot=525  i48  compteur=6 rang=-1 (porte ouverte : rien de porte)
2:15  slot=525  i56  masque=000 (plus aucun emplacement arme)
```

**La lecture est immediate et elle etait pre-inscrite** : le quartet haut vaut 4, 3, 2, 1, 0 ;
le quartet bas vaut 0 partout ; chaque valeur tombe sur une impulsion ; a zero, le film
declare que le joueur ne porte plus rien. **Les cinq instants sont ceux du releve Theater de
l'utilisateur** (1:51, 1:54, 2:03, 2:05, 2:14 — a une seconde). C'est le canal des charges.

Trois faits de cadrage, tous mesures et jamais publies ailleurs :

- **Le film ne transmet la valeur qu'apres le PREMIER usage.** Le bit de masque a 0 signifie
  « le moteur pose 0x7F », c'est-a-dire plein ; il n'y a donc aucune lecture au ramassage. La
  premiere valeur transmise est ce qui RESTE apres le premier usage (4 ici, donc cinq charges
  pour cette variante de propulseur).
- **Les trois emplacements du masque sont specialises** : sur les 16 films mesures, `e0` ne
  s'arme que pour le PROPULSEUR, `e2` pour le GRAPPIN (et trois fois pour le surbouclier), et
  **`e1` n'est arme nulle part**.
- **Une baisse peut valoir plusieurs usages** (7 -> 3, 4 -> 2, 2 -> 0 sont observes) : le film
  ne transmet pas toutes les valeurs intermediaires. Le compte de BAISSES est donc un plancher
  du compte d'USAGES ; la somme des decroissances en est une meilleure estimation.

---

## 3. [ETABLI] Les deux temoins positifs pre-inscrits PASSENT

Instruments : `r11_charges_research_test.go` (`TestR11Charges`) et
`r11_grappin_research_test.go` (`TestR11Grappin`).

### 3.1 Temoin n°1 — le PROPULSEUR contre le canal d'impulsion de R8

| film | vies de propulseur | baisses de charge | dont appariees a une impulsion (±1,5 s) |
|---|---|---|---|
| `1cd3848a` (famille B, rang 21) | 29 | **40** | **40 / 40** |
| `00ba2e1c` (famille A, rang 5) | 20 | 9 | 8 / 9 |
| `084a804d` (famille A, rang 5) | 11 | 5 | 4 / 5 |

Deux canaux qui n'ont rien en commun dans le film — un compteur de charges et une impulsion
d'etat — datent le meme geste a la meme seconde, **52 fois sur 54**.

### 3.2 Temoin n°2 — le GRAPPIN contre `grappleLines[]` (canal totalement independant)

Appariement a ±1,5 s, puis LA MEME mesure avec les accroches decalees de +5 s :

| film | accroches | **appariees** | temoin decale +5 s |
|---|---|---|---|
| `53ce4390` | 25 | **25 / 25** | 2 / 25 |
| `f2966f08` | 6 | **6 / 6** | 0 / 6 |
| `a6ae19fb` | 4 | **4 / 4** | 0 / 4 |
| `3d58eb37` | 1 | **1 / 1** | 0 / 1 |
| **total** | **36** | **36 / 36 (100 %)** | **2 / 36 (5,6 %)** |

Le facteur exige etait 3 ; il est de 18. **Les deux temoins eliminatoires passent : ce que ce
canal ne montre pas, il ne le montre pas parce qu'il n'y est pas.**

### 3.3 Sous-produit pour le chantier : i56 voit PLUS que le canal du grappin en production

Sur `53ce4390`, 42 usages de grappin attribues avec un rang frais contre 25 accroches dans
l'artefact ; sur `084a804d`, 57 contre 5. Le canal des charges a donc un MEILLEUR RAPPEL que
`grappleLines[]` (qui ne garde que les accroches dont la trajectoire est exploitable). C'est
un acquis utilisable tel quel : **dater les usages de grappin et de propulseur, tous joueurs,
sur tout film**.

---

## 4. [ETABLI — NEGATIF] Le REPULSEUR : zero, sur onze films

Attribution stricte (rang lu par i48 le plus recent AVANT l'usage, dans la MEME vie) :

| film | vies de repulseur | expo (min) | i56 lu dans le port | dont ARME | baisses attribuables |
|---|---|---|---|---|---|
| `72b0a25e` (film de l'ancre) | 15 | 6,52 | 0 | 0 | **0** |
| `d9781168` | 37 | 11,54 | 0 | 0 | **0** |
| `53ce4390` | 22 | 6,56 | 0 | 0 | **0** |
| `a6ae19fb` | 19 | 11,99 | 1 | 0 | **0** |
| `3d58eb37` | 18 | 7,42 | 0 | 0 | **0** |
| `0d265ab0` | 15 | 8,11 | 0 | 0 | **0** |
| `4577fcc4` | 6 | 2,30 | 0 | 0 | **0** |
| `f2966f08` | 15 | 5,72 | 0 | 0 | **0** |
| `efe716b4` | 20 | 13,67 | 0 | 0 | **0** |
| `084a804d` | 16 | 18,77 | 10 | 9 | 6, dont **0** avec un rang frais (par. 6) |
| `00ba2e1c` | 35 | 19,11 | 36 | 3 | 2, dont **0** avec un rang frais (par. 6) |
| **total** | **218** | **111,7** | **47** | **12** | **0** |

Dans les memes films et par le meme instrument, le grappin rend 91, 51, 37, 19, 7, 7, 3
baisses et le propulseur 40, 9, 5.

**Le controle qui separe « le canal se tait » de « le canal parle sans jamais baisser ».**
Dans les **six films dont la palette ne porte NI grappin NI propulseur** — `72b0a25e` (74
lectures d'i56), `d9781168` (157), `4577fcc4` (99), `efe716b4` (54), `3372e7eb` (38),
`51ebbc0f` (63) —, **i56 est lu 485 fois et n'est JAMAIS arme** : masque 000 sur toutes les
lectures, dans des films entierement peuples de porteurs de repulseur, de mur de protection,
de detecteur et de camouflage. Le composant transmet donc bien (il annonce, il est lu), et ce
qu'il transmet est « aucun emplacement de charge arme ».

**Aucun autre equipement que le grappin et le propulseur n'arme i56** : mur de protection
(25 vies, 15,2 min), detecteur de menaces (30 vies, 18,3 min), champ de reparation (35 vies,
17,0 min), camouflage (10 vies) rendent tous 0 lecture armee. Seul le surbouclier apparait
trois fois sur `53ce4390`, sur l'emplacement du grappin, avec une attribution faible.

---

## 5. [ETABLI — LE CAS QUI TRANCHE] Un porteur qui consomme SES TROIS CHARGES, et i56 se tait

L'objection legitime au par. 4 est : « et si personne n'avait utilise son repulseur ? ».
Le film y repond lui-meme. Le canal i48 annonce un `spent` — la porte s'ouvre, le joueur ne
porte plus rien — depuis le rang repulseur dans deux des films mesures. Un `spent` depuis le
repulseur signifie que la DERNIERE charge vient d'etre consommee, donc que les trois l'ont
ete.

**`a6ae19fb`, slot 569** (journal complet, `TestR11Journal` avec `R11_ALL=1`) :

```
6:04  slot=569  i56  masque=000
6:21  slot=569  i48  compteur=5 rang=6 (repulseur)      <- prise
   ... 68 secondes de port ...
7:29  slot=569  i48  compteur=6 rang=-1 (porte ouverte) <- DERNIERE CHARGE CONSOMMEE
7:32  slot=569  i48  compteur=7 rang=6 (repulseur)      <- il en reprend un
7:11  slot=569  i56  masque=000
```

**Zero emplacement arme sur toute la duree.** Dans le meme film, au meme moment, i56 compte
les charges du grappin de trois autres joueurs (slots 524, 528, 514, 593 : `e2` a 3, puis 2,
puis 1). Le second cas, `0d265ab0` slot 521 (repulseur pris a 2:46, `spent` a 4:19, 93 s de
port), donne exactement le meme resultat : aucune lecture armee.

**C'est le controle negatif pre-inscrit n°5, dans sa forme la plus forte** : trois charges de
repulseur certainement consommees, et le compteur de charges du film n'a rien transmis.

---

## 6. [REFUTE] Mes propres faux positifs, abattus par le canal independant

L'attribution stricte rendait tout de meme 6 baisses « repulseur » sur `084a804d` et 2 sur
`00ba2e1c`. **Elles sont fausses, et c'est `grappleLines[]` qui le prouve** :

| film | baisses dites « repulseur » | accroches de grappin du MEME slot | verdict |
|---|---|---|---|
| `084a804d` slot 592 | 6:07, 6:11, 6:17, 6:30, 6:45, 6:52 | **6:07, 6:11, 6:17, 6:30, 6:45** | ce sont des accroches de grappin |
| `00ba2e1c` slot 559 | 3:27, 3:34 | **3:26, 3:34** | idem |

La cause est connue et mesuree : **i48 n'emet qu'environ une fois par vie**. Le rang qui
nommait ces baisses avait 117 a 162 secondes (084a804d) et 65 a 72 secondes (00ba2e1c) ; le
joueur avait echange son repulseur contre un grappin sans que le film le dise. L'instrument
publie desormais l'AGE du rang attributeur et une colonne `dont_<20s` ; **avec ce critere de
fraicheur, le repulseur tombe a 0 partout, et le grappin comme le propulseur restent**.

---

## 7. [REFUTE] Les deux autres gisements

### 7.1 Le compteur `R(3)` d'i48 n'est pas un compteur de charges

Mesure sur `72b0a25e`, slot 533 : compteur **5** a la prise du repulseur (2:47), **6** a la
prise du camouflage (2:52), **7** a la porte ouverte (2:55). Sur `a6ae19fb` slot 569 : 5, 6,
7 puis 1 (bouclage sur 8). **Il s'incremente a chaque CHANGEMENT, y compris pour un
equipement sans charge consommable, et il boucle** : c'est bien un compteur de rotation, comme
le port le supposait. L'attendu pre-inscrit (valoir 3 a la prise, decroitre) est refute.

### 7.2 L'entite ti=37 portee : elle n'emet rien, et le champ i27 ne porte pas de charges

Instrument : `r11_entite_research_test.go`, `TestR11Entite`, sur `72b0a25e`.

- **L'ancre donne le handle de l'objet** : a 2:47, la liste i26 de JGtm gagne l'entree
  **1460** ; a 2:52 elle la perd. C'est l'entite du repulseur, identifiee par la coincidence
  exacte avec le `taken` rang 6 d'i48 — deux canaux, un seul instant.
- **Cette entite n'emet AUCUN record d'etat** dans tout le film. Sur les douze handles tenus
  par JGtm, 26 records d'etat seulement, et aucun pour 1460. C'est la limite structurelle
  deja consignee par R9 (« le canal decrit des objets DANS LE MONDE, pas des objets EN MAIN »).
- **Le champ `equipment-charges-remaining` (i27) ne porte pas des charges** : 1 711 valeurs
  lues, 102 distinctes, dont **6,9 % seulement dans 0..8**. Les plus frequentes sont 243 (330
  fois), 251 (200), 247 (180), 255 (142), 248 (133). Une charge d'equipement vaut 0 a
  quelques unites. **Le soupcon du par. 3.3 de R9 est confirme et devient un fait mesure** :
  ce que ce champ rend, sur les records delta, n'est pas un compteur de charges.
- **La jointure d'identite par les creations reste inoperante** : sur 392 records de creation
  ti=37, aucun ne nomme l'entite 1460 (les GlobalID retrouves pour les autres handles sont
  `0xaada07f3`, `0xcaaadcb0`, `0xbcabbe43`, `0x07ae7392` — jamais `0x7ca85adc`, le repulseur).
  L'identification de 1460 repose donc sur la COINCIDENCE TEMPORELLE avec i48, pas sur un
  GlobalID, et c'est dit.

---

## 8. Ce que ce lot etablit, et ce qu'il laisse ouvert

### 8.1 Le negatif du repulseur passe de « structurel » a « structurel ET ancre »

R9 concluait « les tiroirs sont comptes et ils sont pleins » sans jamais disposer d'un usage
certain ni d'un canal valide. Ce lot ajoute les deux : **un canal de charges VALIDE par deux
verites terrain independantes, et des usages de repulseur CERTAINS (les `spent`) pendant
lesquels ce canal se tait**. L'inventaire de R9 s'enrichit donc d'une ligne, et c'est la plus
forte :

| canal | ce qui a ete mesure | verdict |
|---|---|---|
| **i56 `spartan-ability-energy`, ses VALEURS** | **R11 : quartet de charges ; 5/5 sur la verite terrain, 52/54 contre l'impulsion, 36/36 contre `grappleLines`, temoin decale 2/36 ; 218 vies de repulseur, 111,7 min, 0 baisse ; 485 lectures jamais armees dans 6 films sans grappin ni propulseur ; 2 porteurs consommant leurs 3 charges sans une seule lecture armee** | **LE CANAL EXISTE ET NE PORTE PAS LE REPULSEUR** |
| compteur `R(3)` d'i48 | R11 : 5, 6, 7, bouclage — compteur de changements | ABSENT |
| entite ti=37 portee (i27, ancree par le handle i26) | R11 : l'entite du repulseur n'emet aucun record ; 6,9 % des valeurs d'i27 dans 0..8 | ABSENT, et le champ ne porte pas de charges |

### 8.2 Pourquoi c'est coherent, et ce que ca dit du jeu

Le composant s'appelle `biped-spartan-ability-energy`, et la mesure lui donne raison a la
lettre : **il ne compte que les capacites qui deplacent LEUR PORTEUR** — le grappin (`e2`) et
le propulseur (`e0`). Ce sont exactement les deux dont le client PREDIT le mouvement, donc les
deux dont l'energie doit etre repliquee pour que la prediction tienne. Le repulseur agit sur
AUTRUI : son etat n'a pas besoin de voyager vers le client du porteur. R9 avait ecrit cette
intuition (par. 4) ; elle est desormais mesuree, et elle explique la forme du negatif.

### 8.3 Ce qui reste ouvert, dit tel quel

**Je ne peux toujours pas nommer le canal que le visionneur Theater emploie pour rejouer le
repulseur.** Ce lot ferme un gisement de plus — le plus plausible de tous, celui des charges —
et il le ferme avec une ancre. La piste a instruire en premier reste celle de R9 : le type
d'evenement **14 `PlayEffectOnObject`**, present dans les films, dont la grammaire n'est pas
portee. Ce lot lui ajoute une raison : si le geste du repulseur n'est ni dans l'etat du bipede
ni dans l'etat de l'objet, il ne reste que le canal des EVENEMENTS.

**L'ancre reste partielle, et il faut le redire** : l'utilisateur a donne le NOMBRE DE CHARGES
au ramassage, pas les INSTANTS d'usage. Sa fenetre propre ne dure que 5 secondes (par. 0.1) et
on ne sait pas s'il s'en est servi. Ce sont les `spent` du par. 5 qui apportent des usages
certains — deux, sur onze films.

---

## 9. Ce que ce lot NE dit pas

- Il ne dit pas ce que le visionneur Theater emploie (par. 8.3).
- Il ne dit pas combien de charges a le repulseur dans le film : aucune valeur n'est
  transmise. Les 3 charges viennent de l'ecran du jeu, pas de la mesure.
- Il ne mesure pas le rappel du canal des charges sur le grappin et le propulseur : il etablit
  qu'il ne RATE aucun des 36 usages de grappin connus et aucun des 5 usages de propulseur
  releves, mais un usage que ni `grappleLines` ni l'impulsion ne voient resterait invisible aux
  deux mesures.
- Il ne rejoue aucun canal de R7/R8/R9 : evenements 104/119/42/43, tag d'i57/i59, i54, poses
  `deployed`, face victime restent fermes tels que ces lots les ont laisses.
- Il ne touche AUCUN code de production : quatre `*_research_test.go` gardes par
  environnement, non commites.
- Une reserve d'environnement, dite : le checkout `LevelUp-wt-lecture-equipement` etait en
  cours d'edition par une session voisine pendant ce lot (le paquet `internal/games/mappings`
  n'y compilait plus a 21:12). **Les mesures ont donc ete faites sur une COPIE de
  `apps/go-api`** posee dans le scratchpad de la session, ou une rustine d'une ligne
  (conversion `abilityRankEntry.label()`) retablit la compilation de ce paquet — rustine
  LOCALE a la copie, jamais ecrite dans le depot. Le paquet `filmdec`, lui, est intact et
  identique dans les deux arbres.

---

## 10. Instruments et commandes rejouables

Les quatre instruments sont deposes dans le checkout qui porte la pile d'equipement —
`C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-lecture-equipement/apps/go-api/internal/analysis/filmdec/`
— en `*_research_test.go` NON COMMITES, gardes par variables d'environnement, sautes par
defaut (CI comprise), `CGO_ENABLED=0`, `LockProcessDecode` tenu pendant chaque decodage,
`WorldObjectPrecision` pose depuis le layout du film et restaure en sortie (le piege consigne
par R8), aucune ecriture, aucune DuckDB, aucun code de production touche. `gofmt -l` vide et
`go vet` vert sur le paquet.

| Fichier | Role |
|---|---|
| `r11_journal_research_test.go` | `TestR11Journal` — le JOURNAL d'une vie (i48 compteur/rang, i56 masque + 3 emplacements en quartets, i57/i59 tags), en mm:ss, joint au gamertag ; porte AUSSI la collecte partagee `r11Collect` (un seul balayage pour tout le lot) |
| `r11_charges_research_test.go` | `TestR11Charges` — les BAISSES de charge par equipement NOMME PAR LA PALETTE, avec vies, segments de port, minutes d'exposition, lectures i56 armees, age du rang attributeur et detail des instants |
| `r11_grappin_research_test.go` | `TestR11Grappin` — le temoin positif independant : appariement baisses <-> `grappleLines[]`, avec le temoin decale de 5 s |
| `r11_entite_research_test.go` | `TestR11Entite` — i26 (handles tenus), histogramme des valeurs d'i27, series d'etat des entites tenues, identite par les records de creation |

Chemins : `<F>` = `<repo>/data/cache/film_chunks`, `<A>` = `<repo>/data/cache/replays/halo_infinite`,
`<B>` = `<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json`
(ici `<repo>` = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`).

```
# depuis apps/go-api — le journal de l'ancre
CGO_ENABLED=0 R9_FILMS=<F> R9_ARTIFACTS=<A> R8_BOUNDS=<B> R11_IDS=72b0a25e \
  go test ./internal/analysis/filmdec/ -run '^TestR11Journal$' -count=1 -timeout 60m -v

# le journal complet d'un film (tous joueurs) — c'est lui qui montre la serie 4,3,2,1,0
CGO_ENABLED=0 R9_FILMS=<F> R9_ARTIFACTS=<A> R8_BOUNDS=<B> R11_IDS=1cd3848a R11_ALL=1 \
  go test ./internal/analysis/filmdec/ -run '^TestR11Journal$' -count=1 -timeout 60m -v

# les charges par equipement, avec temoins et denominateurs
CGO_ENABLED=0 R9_FILMS=<F> R9_ARTIFACTS=<A> R8_BOUNDS=<B> R11_DETAIL=Repulsor \
  R11_IDS=1cd3848a,72b0a25e,d9781168,53ce4390,a6ae19fb,3d58eb37,0d265ab0,4577fcc4,f2966f08,efe716b4 \
  go test ./internal/analysis/filmdec/ -run '^TestR11Charges$' -count=1 -timeout 180m -v

# le temoin positif independant (grappin)
CGO_ENABLED=0 R9_FILMS=<F> R9_ARTIFACTS=<A> R8_BOUNDS=<B> R11_IDS=53ce4390,f2966f08,a6ae19fb,3d58eb37 \
  go test ./internal/analysis/filmdec/ -run '^TestR11Grappin$' -count=1 -timeout 60m -v

# l'entite portee, ses charges et son identite
CGO_ENABLED=0 R9_FILMS=<F> R9_ARTIFACTS=<A> R8_BOUNDS=<B> R11_IDS=72b0a25e \
  go test ./internal/analysis/filmdec/ -run '^TestR11Entite$' -count=1 -timeout 120m -v
```

Cout mesure : `Journal` 6 a 10 s/film, `Charges` 12 a 25 s/film, `Grappin` 8 s/film,
`Entite` 18 a 40 s/film. Variables : `R11_IDS` (obligatoire), `R11_XUID` (defaut JGtm),
`R11_ALL=1` (tous les joueurs), `R11_DETAIL` (nom EN de l'equipement detaille).

---

## 11. Registre des reports

| # | Ce qu'il faut faire | Pourquoi | Cout |
|---|---|---|---|
| 1 | **Porter la grammaire du type 14 `PlayEffectOnObject`** et la croiser avec les porteurs de repulseur | seul canal present, non porte, dont la semantique correspond ; ce lot ferme le dernier gisement d'ETAT, il ne reste que les EVENEMENTS | 1 lot |
| 2 | **Exploiter le canal des charges pour le rejeu** : i56 date les usages de grappin et de propulseur, tous joueurs, avec un rappel superieur a `grappleLines[]` (42 contre 25 sur `53ce4390`) | acquis livrable en l'etat, hors question du repulseur | 1 lot |
| 3 | **Corriger la fiche de creneaux** (`CRENEAUX_VERIFICATION_EQUIPEMENT_2026-09-03.md` par. 2.c) : les plages de port y vont jusqu'a la fin de la vie et non jusqu'a l'echange — sur `72b0a25e` la fenetre est 2:47 -> 2:52, pas 2:47 -> 3:59 | l'utilisateur a regarde 73 s la ou le film en annonce 5 | 15 min |
| 4 | **Relever au Theater les deux `spent` du par. 5** (`a6ae19fb` 6:21 -> 7:29, joueur du slot 569 ; `0d265ab0` 2:46 -> 4:19, slot 521) : y voit-on trois usages de repulseur ? | ce serait la premiere verite terrain d'USAGE du repulseur, et elle est deja localisee | 1 releve |
| 5 | **Instruire les composants d'etat de ti=37 dans les IMAGES-CLES** (report 2 de R9, non traite ici) | ce lot confirme qu'ils ne sont pas dans les deltas ; les images-cles restent non regardees | 1 lot |
| 6 | **Elucider les valeurs d'i27** (report 4 de R9) : 243, 251, 247, 255 dominent — bruit de reconnaissance ou champ mal nomme ? | ce lot chiffre le probleme (6,9 % dans 0..8) sans le trancher | 1/2 lot |

Les reports 5, 6 et l'angle mort i57 tag 3 (report 5 de R9) restent tels que R9 les a laisses.
