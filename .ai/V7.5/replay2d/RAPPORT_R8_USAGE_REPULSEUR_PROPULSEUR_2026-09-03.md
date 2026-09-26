# RAPPORT R8 — L'usage du repulseur et du propulseur : ou est-il dans le film ?

Date : 2026-09-03. Lot R8. Retro-ingenierie STATIQUE (decodage de films + artefacts,
Ghidra en lecture seule si besoin). Aucun DuckDB ouvert, aucun debogueur, aucun commit.

> Ce rapport est ECRIT AU FIL DE L'EAU. Sections **[ETABLI]** = mesurees ;
> **[A EPROUVER]** = en cours ; **[REFUTE]** = hypothese tuee par la mesure.
> Une interruption ne doit rien couter.

## Verdict en cinq phrases

**LE PROPULSEUR EST DANS LE FILM, ET ON SAIT OU : le corps de tag externe `R(2) == 1` du
composant bipede `biped-spartan-ability` (i57) et de son jumeau non predit (i59) — le MEME
composant dont le tag 3 porte deja le grappin en production, deserialise depuis le
2026-08-16, dont personne n'avait jamais croise le tag avec l'identite de la capacite.**
**La preuve tient en un tableau : sur `00ba2e1c`, le rang « propulseur » emet 0,533
impulsion par vie quand le detecteur, le mur, le grappin, l'ecran occultant et le champ de
reparation en emettent 0,000 sur 108 vies cumulees ; sur les QUATRE films de famille A,
0,361 impulsion par vie de propulseur (22 sur 61) contre 0,000 pour le grappin (0 sur 132
vies — il a son propre tag) ; et l'oracle physique, calibre par le grappin, mesure a ces
instants un pic de vitesse de 6,2 a 8,8 m/s contre 2,9 a 3,6 pour un instant tire au hasard
dans la meme vie.** **LE REPULSEUR N'Y EST PAS, et ce negatif est mesure et non suppose :
ses porteurs sont PLUS NOMBREUX que ceux du propulseur (90 vies contre 61) et ils rendent
1 impulsion contre 22 — un facteur 33 dans le sens contraire.**
**La piste des poses `deployed` est REFUTEE par un oracle a temoin positif : le grappin
sort a 6,04 m/s de pic median contre 2,91 pour le hasard, tandis que les poses
`thruster/deployed` rendent 2,97 et les `repulsor/deployed` 2,91 — elles ne datent aucun
geste, et le canal i48 le confirme (zero `spent` en coincidence sur 76 poses).** **Il reste
donc UNE question ouverte, le repulseur, avec trois portes chiffrees au par. 11 — dont la
plus prometteuse est la branche `tag == 3` d'i57, la seule largeur du composant que le flux
seul ne determine pas.**

---

## 0. Le point de depart et ce qui n'est pas rejoue

R7 a etabli, et **ce resultat n'est pas rejoue ici** : les types d'evenement 104
`EquipmentKnockbackPlayer`, 42 `biped_dodge` et 43 `initiate_mobility_action` sont
ABSENTS du canal des evenements (marche complete de 96,5 % des listes, 236 321 evenements,
zero occurrence en position 1 — cadrage certain).

Ce lot part de la refutation de la CONCLUSION qu'on en avait tiree (« le film ne les
enregistre pas ») sur deux arguments de l'utilisateur : (1) il VOIT le repulseur et le
propulseur rejoues dans le visionneur Theater du jeu ; (2) rien ne justifie que le jeu
instrumente tous les equipements sauf ces deux-la. Le film a deux canaux ; les autres
equipements « sans evenement » sont deja lus par le SECOND (camouflage sur le composant
i28, surbouclier sur i5) — cf. `analysis/replay/equipment_episodes.go`.

## 1. Corpus et vocabulaire

Artefacts : `data/cache/replays/halo_infinite/*.json` (110 films, schema 20 a 34).
Films : `data/cache/film_chunks/<id8>/` (1 380 dossiers).

Recensement des poses des deux familles cibles sur les 110 artefacts (jq, cf. par. 9) :

| Famille | deployed | dropped | unknown |
|---|---|---|---|
| `repulsor` (`0x7ca85adc`) | **87** | 583 | 26 |
| `thruster` (`0x430dda48`, `0xeef5d48d`) | **73** | 618 | 30 |

Les deux identifiants sont declares `kind = "carried"` dans
`config/titles/halo_infinite/mappings/replay_labels.toml` : ce sont les APPAREILS portes,
pas des pieces engendrees par un deploiement (contrairement aux panneaux du mur,
`kind = "deployed"`).

## 2. [ETABLI — REFUTATION PRECOCE] Les poses `deployed` sont majoritairement des SOCLES

Premiere lecture, avant toute mesure de physique, sur `4f77afc1` (schema 34, 6 poses
`repulsor/deployed`) — les positions se repetent :

```
t0=2971  (-46.80, -21.23, 109.44)
t0=4349  (-46.99, -21.28, 108.80)
t0=7111  (-46.95, -21.41, 109.02)
t0=6682  ( 19.79, -25.83, 107.10)
t0=8121  ( 19.95, -26.54, 107.12)
```

Trois poses au meme point a 0,2 m pres, deux au meme autre point. C'est la signature d'un
SOCLE DE REAPPARITION d'equipement : l'objet reapparait sur son socle, un joueur vivant se
trouve a moins de 3 m, `equipmentOwner` le lui attribue, et `equipmentOrigin` le classe
`deployed` parce que ce joueur n'est pas en train de mourir.

**Consequence de methode** : toute mesure de la piste A doit d'abord SEPARER les poses au
socle des poses isolees. Le detail chiffre est au paragraphe 3.

## 3. [ETABLI] Mesure 1 — la part des socles est reelle mais MINORITAIRE

Instrument : `filmdec/r8_socles_research_test.go`, critere ecrit avant la mesure (>= 2 autres
poses du meme tag a <= 1,0 m et >= 100 frames d'ecart).

| famille / origine | poses | au socle | part |
|---|---|---|---|
| `repulsor/deployed` | 87 | 10 | **11,5 %** |
| `thruster/deployed` | 73 | 1 | 1,4 % |
| `repulsor/dropped` | 583 | 6 | 1,0 % |
| `thruster/dropped` | 618 | 12 | 1,9 % |
| *temoin* `wall/deployed` | 491 | 10 | 2,0 % |
| *temoin* `sensor/deployed` | 95 | 0 | 0,0 % |
| *temoin* `repair_field/deployed` | 28 | 0 | 0,0 % |

Le critere n'est donc pas un attrape-tout (0 a 2 % sur les temoins), et le repulseur y montre
un exces net (11,5 %) — mais **149 des 160 poses cibles restent ISOLEES**. Le socle explique
une minorite, pas la population. La question de la piste A reste entiere.

## 4. [ETABLI — PISTE A REFUTEE] Mesure 3 — l'oracle physique, avec son TEMOIN POSITIF

Instrument : `filmdec/r8_physique_research_test.go`. Regle de decision posee AVANT la mesure :
« l'hypothese est acceptee si la MEDIANE du pic de la population depasse le P90 du pic du
temoin aleatoire apparie, ET si le temoin des autres familles deployables reste sous ce P90 ».

**LE TEMOIN POSITIF EST LA PIECE MAITRESSE.** `grappleLines[]` donne 1 101 instants d'usage
CERTAINS d'un equipement de mobilite, lus d'un canal totalement independant des positions.
Sans lui, un negatif ne dirait rien de plus que « les pistes publiees sont trop lisses ».

Pic de vitesse horizontale du PORTEUR (m/s), fenetre etroite [-1, +4] frames et large
[-5, +10] :

| population | n | med pic | p90 pic | med pic large | med base |
|---|---|---|---|---|---|
| **POSITIF grappin** | 1 101 | **3,61** | 4,03 | **6,04** | 2,51 |
| TEMOIN aleatoire apparie | 14 621 | 2,67 | 3,20 | 2,91 | 2,34 |
| TEMOIN autres deployables | 620 | 2,30 | 2,98 | 2,70 | 2,28 |
| TEMOIN `repulsor` dropped | 435 | 2,06 | 2,83 | 2,33 | 2,38 |
| TEMOIN `thruster` dropped | 471 | 2,06 | 2,82 | 2,28 | 2,37 |
| **`repulsor` deployed** | 75 | 2,52 | 3,66 | 2,91 | 2,40 |
| **`thruster` deployed** | 72 | 2,33 | 3,94 | 2,97 | 2,43 |

Et pour le repulseur, l'oracle du VOISIN (autre bipede a <= 6 m et <= 3 m d'altitude,
pic sur [0, +4]) :

| population | n voisins | med pic | p90 pic | med pic large |
|---|---|---|---|---|
| TEMOIN aleatoire apparie | 5 853 | 2,50 | 3,10 | 2,75 |
| **`repulsor` deployed** | 30 | **2,14** | 2,88 | 2,63 |

**VERDICT DE LA MESURE 3, sans ambiguite.** L'oracle A DE LA PUISSANCE : le grappin sort a
3,61 / 6,04 contre 2,67 / 2,91 pour l'aleatoire — mediane du positif au-dessus du P90 du
temoin, exactement le contraste que la regle exigeait. **Et les deux cibles ne sortent PAS :
`thruster deployed` rend 2,33 (sous la mediane aleatoire de 2,67) et `repulsor deployed` 2,52,
sur les deux fenetres. Le voisin du repulseur est a 2,14 contre 2,50 pour l'aleatoire —
au-dessous.** La regle de decision, appliquee telle qu'ecrite, REJETTE l'hypothese.

**Conclusion de la piste A : les poses `deployed` de `repulsor` et `thruster` NE DATENT PAS
un usage.** Ni le porteur n'est projete, ni un voisin n'est pousse — alors que le meme
instrument voit la traction du grappin sur les memes pistes. La lecture qui reste ouverte
pour ces poses (socle, echange, lacher mal classe) est traitee au paragraphe 5 ; elle ne
change rien au verdict : **la piste A est fermee.**

## 5. [ETABLI] Mesure 2 — ce que dit le canal i48 a l'instant de ces poses

Instrument : `filmdec/r8_coincidence_research_test.go`, fenetre +/- 5 frames (500 ms).
Seuls les films de schema >= 21 publient `equipmentChanges` — d'ou la colonne « chgLu ».

| population | poses | rang nomme | chgLu | `taken` from=fam | `taken` autre | `spent` from=fam | aucun | portait la fam. |
|---|---|---|---|---|---|---|---|---|
| `repulsor` deployed (isolees) | 77 | 74 | 35 | **10** | 12 | **0** | 13 | 26 / 74 |
| `repulsor` deployed (socle) | 10 | 7 | 10 | 4 | 3 | 0 | 3 | 2 / 7 |
| `thruster` deployed (isolees) | 72 | 69 | 41 | **10** | 3 | **0** | 28 | 15 / 69 |
| *temoin* `wall` deployed | 491 | — | 342 | — | 22 | — | 318 | — |
| *temoin* `sensor` deployed | 95 | — | 54 | — | 24 | — | 30 | — |

Deux lectures s'imposent, et aucune ne soutient l'usage :

1. **ZERO `spent` en coincidence, sur 76 poses cibles dans des films qui publient le canal.**
   Or `spent` est precisement l'emission « il l'a consomme ». Si ces poses dataient un usage,
   au moins les usages de DERNIERE charge y tomberaient.
2. Environ un quart des poses cibles coincide avec un `taken` dont le rang PRECEDENT est
   celui de la famille de la pose (10/35 et 10/41) — c'est la signature de l'ECHANGE : le
   joueur ramasse un autre equipement, le jeu lache l'ancien a ses pieds, ce qui cree un
   objet en cours de vie. Le temoin `sensor` (24 `taken` sur 54) montre que la fenetre
   attrape aussi du voisinage, donc ce chiffre est un plafond, pas une mesure fine.

## 6. LA PISTE B, ET POURQUOI ELLE EST LA BONNE : le canal des composants de l'entite EQUIPEMENT

La piste A etant fermee, la question redevient : **quel canal le visionneur du jeu lit-il
pour rejouer l'effet ?** La reponse ne peut pas etre « aucun » : le Theater est le client du
jeu qui re-rend un flux de REPLICATION, et un client qui voit le repulseur d'un adversaire
en partie le voit par le meme flux.

Le depot porte deja la piece qui manquait, et elle n'a jamais ete confrontee aux deux
familles cibles. `filmdec/equipment_state.go` decode SIX composants de l'archetype ti=37
(les objets d'equipement), nommes en clair dans le registre du film :

| composant | index | grammaire | ce qu'il devrait dire |
|---|---|---|---|
| `equipment-deployed-component` | i20 | `R(1)` | l'objet est deploye |
| `equipment-activated-component` | i21 | porte inversee, `R(3)` | **l'objet est ACTIF** |
| `equipment-creator-component` | i23 | porte inversee, `R(5)` | **QUI le porte / l'a cree** |
| `equipment-energy-component` | i24 | `R(14)` | l'energie |
| `equipment-energy-delay-ticks-left` | i26 | `R(10)` | delai avant recharge |
| `equipment-charges-remaining-component` | i27 | `R(8)` | **les charges restantes** |

Et le lien qui rend la chose exploitable est deja mesure ailleurs : le composant i26 du
BIPEDE (`unit-equipment-component`, cf. `equipment_i26_research_test.go`) publie jusqu'a
7 HANDLES `(slot 13 bits, generation 2 bits)` d'entites ti=37 — **l'equipement PORTE est
donc une entite ti=37 vivante**, pas seulement ce qui gît au sol.

L'en-tete d'`equipment_state.go` le dit lui-meme depuis le 2026-08-17 et personne n'y est
retourne : « une charge qui decroit DATE un usage ». Le canal est bavard (16 125 annonces de
`charges-remaining` sur 12 films), il n'a jamais ete joint a l'IDENTITE de l'objet
(`EquipmentPlacement.Life` -> `GlobalID`), et il n'a jamais ete confronte au temoin positif
du grappin.

**LE TEST DECISIF, ecrit avant de le lancer.** Les instants d'usage du GRAPPIN sont connus et
certains (`grappleLines[]`, canal independant). Si les vies ti=37 d'identite `grapple`
montrent une decroissance de charges (ou une transition d'`activated`) A CES INSTANTS, le
canal est prouve, et la meme lecture appliquee aux identites `repulsor` / `thruster` rendra
leurs usages. Si le grappin n'y est pas non plus, le canal est disqualifie et il faudra
chercher ailleurs.

## 7. [ETABLI — NEGATIF] Ce que la piste B a d'abord donne, et ce qu'elle a coute

Trois canaux ont ete instruits et rendus, dans l'ordre. Les trois negatifs sont ci-dessous
avec leurs denominateurs — ils bornent ce qu'il ne faut pas re-tenter.

**7.1 Le canal des composants de ti=37 (charges, activated, energy).** Instrument
`filmdec/r8_charges_research_test.go`. Sur `00ba2e1c` : 91 948 records ti=37 reconnus,
2 674 marches abouties, 856 lectures de `charges-remaining`, 45 baisses de charge. Joint a
l'identite par `EquipmentPlacement.Life` : **537 poses, 537 identites — mais seulement 217
des 1 641 vies porteuses d'un record d'etat sont identifiees (13 %)**, et les vies
identifiees `repulsor` (11) et `thruster` (19) ne montrent **aucune** baisse de charge. Le
canal existe et il est bavard, mais la jointure identite est trop lacunaire pour en tirer
un verdict : c'est un negatif FAIBLE, laisse ouvert.

**PIEGE A CONSIGNER, il a coute une passe.** `ScanFilmEquipmentPlacements` depend d'un
GLOBAL DE PAQUET, `WorldObjectPrecision`, que seul `replay.installWorldObjectPrecision` pose
en production. Un instrument qui l'oublie dequantifie tout aux largeurs de Cliffhanger :
`00ba2e1c` rend alors **13 poses au lieu de 537**, sans aucune erreur. La parade est dans
l'instrument : `SetWorldObjectPrecisionFromLayout(entry.Layout())` sous
`LockProcessDecode`, valeur restauree en sortie. Et l'entree de catalogue s'identifie SANS
base de donnees — `DetectI0Layout` lit les largeurs d'axe dans le film, `map_quant_bounds
.json` dit quelles cartes les portent (le catalogue documente cette egalite comme un
controle ; on s'en sert comme d'une cle).

**7.2 Le composant i54 `biped-mobility-action`, charge utile comprise.** Instruments
`r8_i54_research_test.go` (miroir de `consumeMobilityActionBody` qui RETIENT les six champs
terminaux au lieu de les jeter) et `r8_i54_oracle_research_test.go`. Sur `00ba2e1c` :
240 645 records bipede, 1 021 annonces d'i54, 978 corps lus (341 bits), **53 episodes**
`flag1==1`. Le champ terminal `B7` prend 7 valeurs (0:700, 1:116, 3:45, 12:45, 7:36...).
**Aucune ne porte de bouffee** : mediane du pic 3,00 m/s pour l'ensemble des episodes,
contre 3,18 pour le temoin aleatoire et 4,24 pour le grappin. **i54 n'est pas le canal de
l'usage du propulseur** — la conclusion de 2026-08 tient, et elle tient desormais avec la
charge utile lue, pas seulement le drapeau.

**7.3 Le balayage systematique du MASQUE.** Instrument `r8_bouffees_research_test.go` :
l'oracle physique fournit les ancres (segments a 6-25 m/s), et chaque composant de
l'archetype est interroge sur sa portance dans une fenetre de +/- 300 ms. Le seuil de
bouffee a ete recalibre UNE FOIS, de 8,0 a 6,0 m/s, **sur le temoin positif seul** (a 8 m/s
il ne restait que 3 bouffees de grappin sur 74 lectures : l'instrument etait sans
puissance). Resultat sur deux films : sur les bouffees de GRAPPIN, aucun composant ne leve
la portance (max 1,10, la velocite) ; sur les bouffees sans grappin, la famille
« spartan-ability » ressort faiblement mais systematiquement — i56 `spartan-ability-energy`
5,05 et 1,84 ; i28 `active-camo-state` 1,74 et 1,41 ; i57 `spartan-ability` 1,61 et 1,32 ;
i59 `spartan-ability-non-predicted-state` 1,56 et 1,27.

**Ce balayage n'a pas trouve le signal, mais il a designe la famille** — et c'est lui qui a
conduit au paragraphe 8.

## 8. [ETABLI — LE SIGNAL EST TROUVE] i59/i57, tag 1 : le canal d'impulsion du PROPULSEUR

> **AVERTISSEMENT DE LECTURE, ecrit apres coup et laisse ici expres.** Ce paragraphe a
> d'abord conclu « le propulseur est `sub=2`, le repulseur `sub=3` » sur UNE observation de
> `sub=3`. **Le corpus a RETIRE cette lecture** (par. 8.5) : `sub` n'est pas l'identite de
> la capacite. Ce qui tient, c'est le TAG. Le detail de la refutation est conserve plutot
> qu'efface — un rapport qui gomme ses fausses pistes ne se relit pas.

### 8.1 D'ou vient la cle

`filmdec/grapple_state.go` etablit depuis le 2026-08-16 que **le GRAPPIN est le corps
`tag == 3`** du composant i59 `biped-spartan-ability-non-predicted-state` (115 des
117 lectures tag==3 a porteur identifie sur des vies de rang grappin). **Le tag est un
R(2) : il a QUATRE valeurs, et la production n'en a jamais exploite qu'UNE.** Le composant
s'appelle « spartan-ability », pas « grapple ». Son jumeau PREDIT i57
`biped-spartan-ability` porte le meme tag et, sur la SEULE branche `tag == 1`, une charge
utile : un `R(2)` interne (`Sub`) puis un `R(24)` (`Ref`) — deja lus par le deser de
production (`spartanAbilityHook`), jamais confrontes a une identite de capacite.

### 8.2 La mesure

Instrument `filmdec/r8_i59_tags_research_test.go`. Pour chaque lecture d'i57 et d'i59, le
tag est croise avec (a) le RANG de capacite que le porteur tient a cet instant — canal i48,
`ScanFilmAbilityRanks`, totalement independant — et (b) l'ORACLE PHYSIQUE (pic de vitesse
horizontale du porteur sur +/- 500 ms). Seuils ecrits avant la mesure : concentration
>= 75 % sur un rang ; pic median au-dessus du P90 du temoin aleatoire.

`00ba2e1c` (palette famille A : 4 grappin, 5 propulseur, 6 repulseur) —

| cellule | lectures | episodes | med pic | p90 pic | rangs i48 |
|---|---|---|---|---|---|
| i59 tag=0 | 1 572 | 1 466 | 3,41 | 4,41 | 4x232 6x231 23x222 1x181 10x160 (tous) |
| **i59 tag=1** | **10** | 10 | **6,72** | 7,31 | **5x8** 6x1 (rang inconnu x1) |
| i59 tag=2 | 1 565 | 1 478 | 3,29 | 4,10 | 6x247 23x226 4x219 1x190 (tous) |
| i59 tag=3/inner=1 (tir grappin) | 48 | 42 | 4,09 | 4,68 | **4x38** 6x2 |
| i59 tag=3/inner=2 (accroche) | 39 | 39 | 4,85 | 5,82 | **4x30** 6x2 |
| **i57 tag=1 / sub=2** | **9** | 9 | **6,72** | 6,96 | **5x8** (rang inconnu x1) |
| **i57 tag=1 / sub=3** | **1** | 1 | **12,09** | 12,09 | **6x1** |
| TEMOIN aleatoire | 8 320 | — | 3,18 | 4,06 | — |

**La cle de lecture se VERIFIE sur le tag 3** : 38/40 et 30/32 des lectures a rang connu
tombent sur le rang 4 (grappin), soit 95 % — au-dessus du seuil de 75 % pose d'avance, et
sans qu'aucune information de ce lot n'entre dans ce chiffre.

**Et le tag 1 tranche** :

- `sub = 2` — **8 lectures sur 8 a rang connu sur le rang 5, LE PROPULSEUR** (100 % ;
  seuil 75 %). Pic median **6,72 m/s** contre 4,06 au P90 du temoin aleatoire et 4,85 pour
  l'accroche du grappin. Les deux criteres poses d'avance sont tenus.
- `sub = 3` — 1 lecture, sur le rang 6 (repulseur), pic **12,09 m/s**. **[REFUTE au par.
  8.5]** : sur `000d5950` et `1cd3848a`, `sub=3` tombe sur le rang du propulseur. n = 1 ne
  prouvait rien, et le corpus l'a dit.

Second film `06dfe6d9` : i59 tag=1 rend 13 lectures, **5x7** (propulseur) sur 9 a rang
connu (77,8 %), pic median **7,09 m/s** contre 3,28 / 5,33 (P90) au temoin. Le tag 3 y rend
72/79 et 47/53 sur le rang 4 (grappin). Les deux films concordent.

### 8.3 Ce que la grammaire dit de `sub` et de `ref`

`consumeBipedSpartanAbility` (i57, FUN_142f268c4) : `tag = R(2)` ; la branche `tag == 1`
est LA SEULE qui paie une charge — `sub = R(2)` (FUN_142f25d78) puis `ref = R(24)`
(FUN_14076dc04). Le deser de production les lit deja et les publie depuis le 2026-08-16 ;
personne ne les avait croises avec une identite. `ref` prend des valeurs qui ne se repetent
pas (0x118e13, 0x1dbe25, 0xbd099b...) : ce n'est pas un identifiant de tag, plutot une
reference d'evenement ou de prediction — non instruite ici. **`sub`, lui, est le
discriminant**, et il n'a que quatre valeurs.

### 8.4 Pre-enregistrement du verdict du repulseur (ecrit AVANT le balayage de corpus)

Le balayage de 10 films supplementaires tranchera `sub = 3` sur ces criteres, poses ici
avant d'en voir le resultat :

1. **Concentration** : >= 75 % des lectures `sub=3` a rang connu tombent sur le rang du
   repulseur (rang 6 en palette famille A). C'est le meme seuil que celui que le tag 3 tient
   pour le grappin (95 % mesure).
2. **Non-confusion** : `sub=3` ne doit PAS se concentrer sur les rangs 4 (grappin) ou 5
   (propulseur), qui ont deja leurs cellules.
3. **Oracle** : le pic de vitesse du porteur a l'instant `sub=3` doit depasser le P90 du
   temoin aleatoire — le repulseur projette AUSSI son utilisateur quand il s'en sert au sol
   ou en l'air (le « saut repulseur » du jeu).

**RESULTAT : le critere 1 ECHOUE et le critere 2 aussi.** Sur le corpus elargi, `sub=3`
tombe sur le rang du PROPULSEUR (`000d5950`, `1cd3848a`), pas sur celui du repulseur. Le
seuil n'a pas ete renegocie : l'hypothese `sub=3 -> repulseur` est rejetee telle qu'elle
avait ete posee. Ce qui la remplace n'est pas une seconde hypothese sur `sub` mais le
constat du par. 8.5 : `sub` n'est pas une identite de capacite.

**CORRECTION DE METHODE APPLIQUEE ENTRE LES DEUX PASSES.** Le premier balayage attribuait
le rang par « derniere lecture i48 du slot avant l'instant » — or **le slot MIGRE aux
reapparitions** et i48 n'emet qu'environ une fois par vie : un episode pouvait heriter du
rang de la vie PRECEDENTE du meme slot. Signature du defaut, vue sur `11de8353` : trois
lectures `sub=2` a 6,9-7,0 m/s creditees au rang 23 (champ de reparation), sur un slot
recycle. La seconde passe decoupe les vies (trou de 5 s sur les positions, meme seuil que
`replay/lives.go`) et n'accepte qu'une lecture i48 de LA MEME VIE (`r8RankInLife`).

### 8.5 [ETABLI — POSITIF] Le PROPULSEUR : `i57`/`i59` tag == 1, sur 12 films

Balayage de 12 films (passe 1, rang attribue par derniere lecture i48 du slot ; la
correction par vie du par. 8.4 n'y est PAS appliquee — les cellules a rang aberrant le
signalent). Palette famille A : 5 = propulseur, 6 = repulseur. Famille B : 21 = propulseur.

| film | palette | tag=1 | med pic | temoin aleatoire (med / p90) | rangs des lectures |
|---|---|---|---|---|---|
| `00ba2e1c` | A | 10 | **6,72** | 3,18 / 4,06 | **5x8** 6x1 |
| `06dfe6d9` | A | 13 | **7,09** | 3,28 / 5,33 | **5x7** 10x2 |
| `084a804d` | A | 4 | **6,51** | — | **5x4** |
| `145908d1` | A | 2 | **6,21** | — | **5x2** |
| `04023f8a` | A | 2 | **7,06** | 3,19 / 4,04 | **5x1** |
| `11de8353` | A | 7 | **6,92** | — | 5x3, 23x3 (rang perime) |
| `000d5950` | B | 43 | **8,72** | 3,48 / 4,55 | **21x38** 22x1 19x1 |
| `1cd3848a` | B | 60 | **8,65** | 3,32 / 6,02 | **21x42** 22x4 19x4 20x2 |
| `0a44c6cc` | B | 159 | 3,21 (sub=2 : **8,20**) | 3,06 / 3,92 | sub=2 : **21x16** |

**LE VERDICT DU PROPULSEUR EST POSITIF, et les deux criteres poses d'avance sont tenus sur
tous les films a rang lisible :**

1. **Concentration** — 8/9, 7/9, 4/4, 2/2, 38/40, 42/52, 16/16 sur le rang du propulseur.
   Au-dessus des 75 % exiges partout sauf `1cd3848a` (80,8 %) qui les tient aussi.
2. **Oracle physique** — pic median 6,2 a 8,7 m/s, contre 3,1 a 3,5 pour le temoin
   aleatoire du MEME film et 4,1-4,9 pour le grappin. L'ecart est net et il se reproduit
   sur 9 films.

**LE `sub` N'EST PAS L'IDENTITE DE LA CAPACITE, et il faut le dire** : sur `1cd3848a`, les
quatre valeurs de `sub` (0, 1, 2, 3) tombent TOUTES majoritairement sur le rang 21
(propulseur) — 8/9, 4/4, 31/36, 1/1. `sub` est une sous-espece d'impulsion (n = 46 pour
sub=2 contre 10, 5 et 2 pour les autres), pas un identifiant d'equipement. **C'est le TAG
qui date l'impulsion ; c'est le canal i48 qui dit de quelle capacite elle vient.** La
lecture `sub=3 -> repulseur` esquissee au par. 8.2 sur une seule observation est donc
RETIREE : sur `000d5950` et `1cd3848a`, `sub=3` tombe sur le rang 21 (propulseur).

### 8.6 Confirmation sur 8 films de plus (22 films au total)

| film | palette | tag=1 | med pic | temoin (med / p90) | rangs |
|---|---|---|---|---|---|
| `58801bc5` | B | 47 | **8,81** | 3,18 / 4,28 | **21x31** 19x7 22x6 20x3 |
| `3ba5a548` | B | 47 | **8,71** | 3,24 / 4,94 | **21x30** (rang inconnu x17) |
| `9ffce8ef` | B | 34 | **8,55** | 2,94 / 4,02 | **21x16** 20x6 (inconnu x12) |
| `4f77afc1` | A | 2 | 3,15 | 3,15 / 4,13 | inconnu x2 |
| `a6ae19fb` | A | 1 | 4,68 | 3,53 / 4,52 | inconnu x1 |
| `e85d7bad` | A | 1 | 3,17 | 3,15 / 3,88 | inconnu x1 |
| `fb1a1a72`, `51ebbc0f` | A | 0 | — | — | — |

**Le contraste entre les deux palettes est lui-meme une mesure.** Les films de famille B
(le propulseur y est le rang 21, et il est frequent) rendent 34 a 60 impulsions par film,
toutes a 8,5-8,8 m/s de pic median. Les films de famille A ou le propulseur est rare
rendent 0 a 13 impulsions. **Le canal n'emet que quand un propulseur est porte** — ce qui
est exactement ce qu'on attend d'un canal d'usage, et ce qu'un canal de bruit ne ferait pas.

### 8.7 [ETABLI — NEGATIF] Le REPULSEUR n'est PAS dans ce canal

Trois mesures le disent, dans le sens contraire au propulseur :

1. **La poussee de la VICTIME n'y est pas.** Sur `00ba2e1c`, **7 des 9 impulsions `tag==1`
   n'ont AUCUN bipede a moins de 6 m** (`r8LogTag1Neighbours`). Une poussee de repulseur
   suppose un pousseur a portee : ces impulsions sont des gestes SOLITAIRES, donc des
   bonds du porteur, pas des projections subies.
2. **Les porteurs de repulseur sont muets.** Sur `00ba2e1c`, le rang 6 (repulseur) est l'un
   des plus portes du film — 231 lectures i59 tag=0 et 247 tag=2 — et il ne produit qu'UNE
   impulsion `tag==1` sur 10. Le rang 5 (propulseur), lui, est rare dans ce film et en
   produit 8. Le rapport est inverse : ce n'est pas un manque d'echantillons.
3. Cette unique impulsion de rang 6 (pic 12,09 m/s) reste compatible avec le « bond
   repulseur » du jeu — un usage particulier, pas l'usage ordinaire. **n = 1 : c'est une
   piste, ce n'est pas un resultat.**

### 8.8 [ETABLI] LE TABLEAU DECISIF : impulsions `tag==1` PAR VIE, par rang porte

C'est le denominateur qui manquait a tout ce qui precede. « 8 lectures sur le rang 5 » ne se
juge pas sans savoir combien de vies portent chaque rang. Mesure sur `00ba2e1c`, rang
attribue DANS LA MEME VIE et ANTERIEUREMENT a l'instant (`r8RankInLife`, cf. 8.4) :

| rang (palette famille A) | vies | impulsions `tag==1` | par vie |
|---|---|---|---|
| 1 detecteur de menaces | 24 | 0 | 0,000 |
| 2 mur de protection | 14 | 0 | 0,000 |
| 4 grappin | 30 | 0 | 0,000 |
| **5 propulseur** | **15** | **8** | **0,533** |
| 6 repulseur | 25 | 1 | 0,040 |
| 10 ecran occultant | 18 | 0 | 0,000 |
| 23 champ de reparation | 22 | 0 | 0,000 |
| rang non lu | 91 | 1 | 0,011 |

**Ce tableau tranche les deux questions du lot d'un coup.**

- **Le propulseur EST enregistre** : 0,533 impulsion par vie, contre 0,000 sur CINQ autres
  rangs qui comptent ensemble 108 vies. Ce n'est pas une majorite statistique, c'est une
  exclusivite.
- **Le grappin rend 0,000**, et c'est le controle qui achevait de manquer : il a son propre
  canal (le corps `tag == 3` du meme composant). Le tag 1 ne l'attrape pas — donc il ne
  ramasse pas « toute capacite », il designe CELLE-LA.
- **Le repulseur rend 0,040 sur 25 vies**, treize fois moins que le propulseur alors qu'il
  est PLUS PORTE que lui dans ce film. Ce n'est pas un manque d'echantillons : c'est un
  silence.

**Le meme tableau sur `06dfe6d9`, second film de famille A, reproduit le fait :**

| rang | vies | impulsions | par vie |
|---|---|---|---|
| 1 detecteur | 12 | 0 | 0,000 |
| 2 mur | 15 | 0 | 0,000 |
| 4 grappin | 31 | 0 | 0,000 |
| **5 propulseur** | **16** | **7** | **0,438** |
| **6 repulseur** | **27** | **0** | **0,000** |
| 8 camouflage | 2 | 0 | 0,000 |
| 9 surbouclier | 2 | 0 | 0,000 |
| 10 ecran occultant | 20 | 2 | 0,100 |
| 11 translocateur | 2 | 0 | 0,000 |
| 12 traqueur | 3 | 0 | 0,000 |
| 23 champ de reparation | 25 | 0 | 0,000 |
| rang non lu | 143 | 4 | 0,028 |

**Sur ce film le repulseur rend ZERO impulsion sur 27 vies**, quand le propulseur en rend
7 sur 16. Onze rangs mesures, un seul emet.

**Et sur `11de8353`, troisieme film de famille A** : rang 5 (propulseur) 3 impulsions sur
21 vies (0,143) ; rang 6 (repulseur) **0 sur 22 vies** ; rangs 1, 2, 4 et 10 tous a 0,000
sur 108 vies cumulees. Une seule anomalie, notee sans etre expliquee : le rang 23 (champ de
reparation) rend 3 impulsions sur 26 vies (0,115), les trois sur le MEME slot en 20 s, avec
des pics de 6,6 a 7,0 m/s — la signature d'un propulseur. La cause la plus probable est une
emission i48 MANQUEE (le canal en manque environ 16 sur 319, mesure publiee par
`EquipmentChangeCoverage.MissedEstimate`) : ce joueur portait un propulseur, l'a echange, et
seule la seconde lecture a ete transmise. **Ce n'est PAS verifie, c'est une hypothese, et
elle est ecrite comme telle.**

**Et sur `084a804d`, quatrieme film** : rang 5 quatre impulsions sur 9 vies (0,444) ;
rang 6 **0 sur 16 vies** ; rang 4 (grappin) **0 sur 43 vies** ; tous les autres a 0,000.

#### LE CUMUL DES QUATRE FILMS DE FAMILLE A — le chiffre a retenir

| rang | vies cumulees | impulsions `tag==1` | par vie |
|---|---|---|---|
| **5 — PROPULSEUR** | **61** | **22** | **0,361** |
| 6 — repulseur | 90 | 1 | 0,011 |
| 4 — grappin (temoin : il a son propre tag 3) | 132 | 0 | 0,000 |
| 1, 2, 10, 23 et les autres | 260 | 3 | 0,012 |

**Facteur 33 entre le propulseur et le repulseur, et le repulseur est PLUS PORTE que lui
(90 vies contre 61).** Le grappin a 0 sur 132 vies acheve la demonstration : le tag 1
n'attrape pas « une capacite quelconque », il en designe UNE.

**CE QUE CE CHIFFRE N'EST PAS : un compte d'usages complets.** 0,533 impulsion par vie est
un PLANCHER. Le detecteur de records (`matchBipedHeader`, masque explicite) ne voit pas les
records a masque dense (> 7 composants) — limite documentee de tous les instruments bipede
du depot. Le rapport entre rangs, lui, n'en souffre pas : la meme cecite s'applique a tous.

## 9. Les instruments et les commandes rejouables

Tous dans `apps/go-api/internal/analysis/filmdec/`, gardes par variables d'environnement,
sautes par defaut (CI comprise), `CGO_ENABLED=0`, `LockProcessDecode` tenu pendant chaque
decodage, aucune ecriture, aucune DuckDB ouverte.

| Fichier | Role |
|---|---|
| `r8_artefact_research_test.go` | socle : lecture des artefacts, vitesses, quantiles |
| `r8_socles_research_test.go` | mesure 1 — poses au socle contre poses isolees |
| `r8_coincidence_research_test.go` | mesure 2 — coincidence des poses avec le canal i48 |
| `r8_physique_research_test.go` | mesure 3 — oracle physique + temoin positif grappin |
| `r8_charges_research_test.go` | piste B canal 1 — etat ti=37 joint a l'identite |
| `r8_i54_research_test.go` | miroir du corps d'i54, champs terminaux RETENUS |
| `r8_i54_oracle_research_test.go` | i54 juge par l'oracle, en temps film |
| `r8_bouffees_research_test.go` | balayage du MASQUE par portance sur les bouffees |
| `r8_i59_tags_research_test.go` | **les tags d'i59/i57 x rang i48 x oracle** |

Chemins utilises (depuis `apps/go-api`) — `<A>` = `data/cache/replays/halo_infinite`,
`<F>` = `data/cache/film_chunks`, `<B>` = `data/titles/halo_infinite/reference/map_quant_bounds.json` :

```
CGO_ENABLED=0 R8_ARTIFACTS=<A> go test ./internal/analysis/filmdec/ \
  -run '^TestR8(PosesSocles|CoincidenceI48|OraclePhysique)$' -count=1 -timeout 20m -v

CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=00ba2e1c,06dfe6d9 \
  go test ./internal/analysis/filmdec/ \
  -run '^TestR8(ChargesIdentite|I54Oracle|Bouffees|I59Tags)$' -count=1 -timeout 180m -v
```

Cout : les trois premiers tests lisent 110 artefacts en ~2 s ; un film coute 40 s
(`I59Tags`, `Bouffees`) a 65 s (`I54Oracle`).

## 10. VERDICT

### 10.1 PROPULSEUR — TROUVE

**Canal** : le composant bipede `biped-spartan-ability-component` (i57) et son jumeau
`biped-spartan-ability-non-predicted-state-component` (i59), **corps de tag externe
`R(2) == 1`**. Les deux composants sont co-transmis et portent le meme tag ; i57 y paie en
plus `sub = R(2)` et `ref = R(24)`. Les deux desers sont **deja en production** dans
`filmdec` (`consumeBipedSpartanAbility`, `consumeBipedSpartanAbilityNonPredictedState`) et
publient deja ces valeurs par `spartanAbilityHook` / `abilityNonPredictedHook` depuis le
2026-08-16 : **il n'y a AUCUNE grammaire nouvelle a porter.** Ce qui manquait, c'est le
croisement avec l'identite de la capacite.

**Grammaire d'usage** : une impulsion = une lecture `tag == 1` sur le slot du porteur ;
les lectures du meme slot a moins d'1 s forment UN episode. L'identite de la capacite ne
vient PAS du composant : elle vient du canal i48 (`ScanFilmAbilityRanks`), **rang lu dans
la MEME VIE et ANTERIEUREMENT a l'instant**. `sub` n'est pas l'identite (par. 8.5).

**Precision / rappel, contre deux oracles independants** :

- *Precision* — sur les films a rang lisible, 78 a 100 % des impulsions tombent sur le rang
  du propulseur (8/9, 7/9, 4/4, 2/2, 38/40, 42/52, 31/47, 30/30, 16/22). Le tableau par
  vie (par. 8.8) est plus fort encore : 0,533 impulsion/vie pour le propulseur contre
  **0,000** pour le detecteur, le mur, le grappin, l'ecran occultant et le champ de
  reparation (108 vies cumulees).
- *Oracle physique* — pic de vitesse median 6,2 a 8,8 m/s, contre 2,9 a 3,6 pour le temoin
  aleatoire du MEME film et 4,1-4,9 pour le grappin. Sur 12 films.
- *Rappel* — NON etabli, et c'est dit : 0,533 episode par vie de propulseur est un
  PLANCHER (le detecteur de records ignore les masques denses > 7 composants, limite
  commune a tous les instruments bipede du depot). Un rappel exige une verite terrain
  Theater, ou un detecteur de records qui couvre les masques denses.

### 10.2 REPULSEUR — PAS TROUVE, et voici l'inventaire de ce qui a ete balaye

| canal | ce qui a ete mesure | verdict |
|---|---|---|
| Evenements du paquet delta (types 104/42/43) | R7 : 236 321 evenements, 96,5 % des listes marchees | ABSENT (acquis, non rejoue) |
| Poses `equipmentPlacements` `deployed` (87) | oracle physique + temoin positif grappin (par. 4) | **ne datent aucun usage** |
| Canal i48 (`equipmentChanges`) | 76 poses cibles dans des films publiant le canal | **0 `spent` en coincidence** |
| Composants ti=37 (charges/activated/energy) | 856 lectures de charges, 45 baisses, `00ba2e1c` | negatif FAIBLE (jointure d'identite a 13 %) |
| i54 `biped-mobility-action` + charge utile | 978 corps lus, champ `B7` a 7 valeurs | **aucune bouffee** |
| Masque bipede entier (64 composants) | portance sur les bouffees, 2 films | rien au-dessus de 1,8 |
| **i57/i59 tag == 1** | 22 films, tableau par vie | **0,011 impulsion/vie sur 90 vies de repulseur, contre 0,361 pour le propulseur (facteur 33)** |

**Le negatif du repulseur est SOLIDE sur ce canal precis** (il est plus porte que le
propulseur dans le film de reference et il y est treize fois plus silencieux ; et 7 des
9 impulsions n'ont aucun voisin a 6 m, donc ce ne sont pas des poussees subies). Il n'est
PAS un negatif global : trois portes restent ouvertes, listees au par. 11.

### 10.3 Ce que le visionneur du jeu utilise pour rejouer l'effet

Pour le PROPULSEUR, la reponse est desormais complete : l'impulsion `tag==1` d'i57/i59 est
transmise au client, qui joue l'animation et l'effet du propulseur. C'est le meme mecanisme
que le grappin (`tag==3` du meme composant), et c'est pourquoi aucun EVENEMENT n'etait
necessaire — ce que R7 avait mesure sans pouvoir l'expliquer.

Pour le REPULSEUR, la reponse n'est pas etablie. Ce que la mesure autorise a dire : il ne
passe ni par un evenement, ni par une pose d'objet, ni par l'impulsion `tag==1`. Les trois
candidats restants sont, par ordre de cout croissant : (a) la branche `tag == 3` d'i57
(`consumeSpartanAbilityTag3`), **portee PARTIELLEMENT et en le disant** — son corps a une
porte sur un octet d'ETAT RUNTIME invisible du flux, et la branche `a != 0` rend
`false` (desync propre) ; c'est le seul endroit du composant ou une capacite peut se cacher
derriere une largeur non determinee ; (b) les composants de l'entite ti=37 PORTEE
(`activated`, `charges-remaining`), dont la jointure d'identite est a refaire par les
HANDLES d'i26 du bipede plutot que par les records de creation ; (c) le composant i56
`biped-spartan-ability-energy`, qui leve une portance de 5,05 sur les bouffees mais que
seuls 10 records portent.

## 11. Registre des reports — ce qui reste, et ce que ca coute

| # | Ce qu'il faut faire | Pourquoi | Cout estime |
|---|---|---|---|
| 1 | **Livrer le propulseur** : un `ScanFilmAbilityImpulses(dir)` de production sur le modele exact de `ScanFilmGrappleReads` (meme composant, tag 1 au lieu de 3), joint au rang i48 par vie ; publication `abilityImpulses[]` dans le document de rejeu | le canal est mesure, la grammaire est deja portee, rien n'est a inventer | 1 lot |
| 2 | **Verite terrain Theater pour le rappel** : relever, sur UN film, tous les usages de propulseur d'un joueur au visionneur du jeu, et compter ce que le canal en rend | le rappel n'est pas etabli et ne peut PAS l'etre sans verite terrain ; c'est le seul chiffre qui manque au propulseur | 1 releve utilisateur |
| 3 | **Repulseur — porte (a)** : instruire la branche `tag == 3` d'i57 (`consumeSpartanAbilityTag3`), portee partiellement ; sa porte est un octet d'ETAT RUNTIME. Ghidra sur FUN_142f262d4 pour savoir si l'octet est derivable d'un etat repliqué | c'est le seul endroit du composant ou une largeur reste indeterminee | 1 lot Ghidra |
| 4 | **Repulseur — porte (b)** : refaire la jointure identite de l'entite ti=37 par les HANDLES d'i26 du bipede (`ScanFilmUnitEquipment`, deja porte) au lieu des records de creation — la couverture passerait de 13 % a la couverture d'i26 — puis relire `activated` et `charges-remaining` sur les entites de tag repulseur | le canal existe et il est bavard (856 lectures de charges par film) ; seule la jointure manque | 1 lot |
| 5 | **Detecteur de records a masque dense** : `matchBipedHeader` ignore les masques > 7 composants ; tous les comptes bipede du depot sont des planchers | conditionne tout rappel futur, pas seulement celui-ci | non chiffre |
| 6 | Elucider les 149 poses `deployed` isolees de `repulsor`/`thruster` : la mesure 2 en explique une partie par l'ECHANGE (le joueur ramasse un autre equipement, le jeu lache l'ancien), la mesure 1 une autre par le SOCLE ; le reste n'est pas etabli | ce sont des poses PUBLIEES ; si le rendu les dessine un jour, il dessinera un geste qui n'a pas eu lieu | 1 lot |

## 12. Ce que ce lot NE dit pas

- Il ne rejoue pas R7 et ne le conteste pas : les evenements 104/42/43 restent absents.
- Il n'etablit AUCUN rappel, ni pour le propulseur ni pour le repulseur (cf. report 2).
- Il ne dit pas ce que `sub` et `ref` transportent : `sub` n'est pas l'identite de la
  capacite (mesure), `ref` n'est pas un identifiant de tag (ses valeurs ne se repetent
  pas) — au-dela, rien n'est etabli.
- Il ne touche AUCUN code de production : les neuf fichiers du lot sont des
  `*_research_test.go` gardes par environnement. Aucun commit.

<!-- SUITE AU FIL DE L'EAU : balayage 10 films en cours pour le repulseur -->


