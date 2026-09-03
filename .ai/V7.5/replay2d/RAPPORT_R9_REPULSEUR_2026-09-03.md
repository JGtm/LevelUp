# RAPPORT R9 — Le REPULSEUR : la derniere question ouverte du chantier

Date : 2026-09-03. Lot R9, suite directe de R8. Retro-ingenierie STATIQUE (decodage de films
et d'artefacts, Ghidra en lecture seule). Aucun DuckDB ouvert, aucun debogueur, aucun commit,
aucune ecriture sur la production.

> Ce rapport est ECRIT AU FIL DE L'EAU. **[ETABLI]** = mesure faite ; **[A EPROUVER]** = en
> cours ; **[REFUTE]** = hypothese tuee par la mesure. Une interruption ne doit rien couter.

Branche de travail : `wt/r9-repulseur`, creee depuis `wt/lecture-equipement` (07ad449b1) pour
disposer des neuf instruments R8 et des desers portes. Instruments R9 ajoutes dans
`apps/go-api/internal/analysis/filmdec/`, tous `*_research_test.go` gardes par environnement.

---

## Verdict en cinq phrases

**LE REPULSEUR N'EST PAS DANS LE FILM, et ce lot fait passer ce negatif de circonstanciel a
STRUCTUREL : les tiroirs sont comptes et ils sont pleins.** **Les quatre valeurs du tag
d'i57/i59 sont maintenant TOUTES attribuees — 0 et 2 sont deux etats de repos qui portent la
meme distribution de rangs, 1 est le propulseur (R8), 3 est le grappin (production, et sa
branche predite `a == 1` d'i57 tombe a 90-100 % sur le rang du grappin sur quatre films) —
donc il n'y reste aucune place ; l'archetype bipede a exactement 64 composants, la borne dure
du masque, et aucun ne porte de knockback alors que le moteur en a un
(`KnockbackTargetComponent`, trouve dans l'exe) ; i54 est desormais ferme sur l'oracle
d'IDENTITE et plus seulement sur celui de la vitesse ; et le recensement du masque bipede par
rang porte — SANS ancre, la ou celui de R8 dependait d'une bouffee de vitesse que le repulseur
ne produit pas — DETECTE le surbouclier (2,6x) et le grappin (1,4-2,3x) mais ne donne au
repulseur, rang le mieux echantillonne du film, le maximum d'AUCUN des 64 composants.** **La porte (b) tombe deux fois : la
jointure par les handles d'i26 plafonne a 3,2 % (pire que les 13,2 % qu'elle devait remplacer,
i26 n'emettant que 170 fois par film), et surtout le RECENSEMENT DU MASQUE de ti=37 — qui
porte son propre etalon de bruit, les composants i31..i63 n'existant pas — montre que
`equipment-activated` (204 annonces) est AU plancher de faux positifs (~210) et
`equipment-control-signal` (126) SOUS ce plancher : le canal d'etat de l'equipement ne
transmet rien dans les deltas, et les « 45 baisses de charge » de R8 sont selon toute
vraisemblance du bruit.** **La sortie de secours — « le film porte l'EFFET sans le GESTE » —
a ete mesuree et n'est PAS validee : les voisins d'un porteur de repulseur ne sont pas pousses
plus que ceux d'un porteur temoin ; mais la mesure est SOUS-PUISSANTE (son temoin positif, le
propulseur, ne separe qu'a 1,4-1,8) et elle n'est donc pas refutee non plus.**
**Ce que le visionneur Theater emploie N'EST PAS ETABLI, et c'est ecrit tel quel : la piste a
instruire en premier est le type d'evenement 14 `PlayEffectOnObject`, PRESENT dans les films
et dont la grammaire n'est pas portee — et la piece qui manque a tout ce chantier est un
ORACLE : une verite terrain Theater du repulseur, sans laquelle chaque canal se juge a
l'aveugle.**

---

## 0. Ce qui n'est PAS rejoue

Acquis de R7 et R8, repris sans etre remesure :

- Les types d'evenement 104 `EquipmentKnockbackPlayer`, 119, 42 `biped_dodge` et 43
  `initiate_mobility_action` sont ABSENTS du canal des evenements (R7 par. 4.5, marche
  complete, 236 321 evenements). **Canal CLOS.**
- Le PROPULSEUR est l'impulsion `tag == 1` des composants bipede i57/i59 ; le GRAPPIN est le
  `tag == 3` d'i59. 0,361 impulsion par vie de propulseur contre 0,000 pour cinq autres
  equipements sur 4 films de famille A.
- Le REPULSEUR n'est pas dans ce canal-la (0,011 impulsion par vie sur 90 vies), ni dans les
  poses `deployed`, ni dans le canal i48 (`spent`), ni dans i54, ni dans le masque bipede
  balaye par portance.

**Controle de reproductibilite (fait en premier, avant toute mesure neuve).**
`TestR8I59Tags` rejoue sur `00ba2e1c` rend EXACTEMENT les chiffres publies par R8 :
tag=1 -> 10 lectures, med pic 6,72 ; par vie rang 5 = 0,533, rang 6 = 0,040, rangs 1/2/4/10/23
= 0,000. L'environnement de ce lot est donc le meme que celui de R8 (memes bornes de carte,
meme `WorldObjectPrecision`, meme decoupage par vie).

**Fait neuf apparu dans ce controle, non publie par R8** : la table d'i57 comporte une
cellule `tag=3` que le rapport R8 n'avait pas reprise —

```
i57 biped-spartan-ability
  tag=0            1572 lectures   medPic 3.41   [-1x478 4x218 6x215 23x164 1x159 10x138]
  tag=1/sub=2         9            6.72          [5x8 -1x1]
  tag=1/sub=3         1           12.09          [6x1]
  tag=2            1565            3.29          [-1x476 6x231 4x207 23x170 1x167 10x138]
  tag=3              40            3.80          [4x28 -1x10 6x2]      <-- LA PORTE (a)
```

**28 des 30 lectures `i57 tag=3` a rang connu tombent sur le rang 4, le GRAPPIN (93,3 %).**
C'est le premier element de la porte (a), et il joue contre elle : le tag 3 d'i57 est le
jumeau predit du tag 3 d'i59, c'est-a-dire le grappin, pas une capacite cachee. Mais le
raisonnement n'est pas fini : la branche `tag == 3` d'i57 se scinde en DEUX sur un bit `a`
lu juste apres, et seule la moitie `a == 0` est portee. Ce que fait `a == 1` reste inconnu, et
c'est precisement la question de la porte (a).

---

## 1. Pre-inscription des verdicts (ECRITE AVANT LES MESURES)

Regle du chantier : les seuils et les temoins s'ecrivent avant de regarder. Ce paragraphe est
fige ; les resultats viennent apres, et ils sont juges par ces criteres-la, pas par d'autres.

### 1.1 Porte (a) — la branche `tag == 3` d'i57, bit `a`

`consumeSpartanAbilityTag3` (miroir de FUN_142f262d4) lit `a = R(1)`. Si `a == 0`, la suite
est entierement portee (une porte de queue, puis le lecteur de queue-handle d'i60). Si
`a == 1`, le corps lit `R(6)` puis une branche gardee par un OCTET D'ETAT RUNTIME (`dst[2]`)
invisible du flux : le deser rend `false` et **le record entier est abandonne**.

Le bit `a` LUI-MEME est lisible sans rien deviner : il vient juste apres le tag, a une
position dont le cadrage est certain. **La mesure est donc gratuite.**

**Ce que j'attendrais si le repulseur etait la** (ecrit avant de regarder) :

1. **Concentration** — parmi les lectures `i57 tag==3, a==1` a rang connu, **>= 75 %** tombent
   sur le rang du repulseur (rang 6 en palette famille A). Meme seuil que celui que le tag 3
   d'i59 tient pour le grappin (93-95 % mesures).
2. **Non-confusion** — ces lectures ne doivent PAS se concentrer sur le rang 4 (grappin), qui
   possede deja tout le tag 3, ni sur le rang 5 (propulseur), qui possede le tag 1.
3. **Denominateur par vie** — le taux d'episodes `a==1` par vie de repulseur doit depasser
   d'un **facteur >= 5** le maximum des rangs temoins (detecteur, mur, ecran occultant, champ
   de reparation). C'est la forme de tableau qui a tranche le propulseur (par. 8.8 de R8) et
   c'est elle qui tranchera ici.
4. **Oracle du VOISIN** — a ces instants, un bipede a moins de 6 m doit montrer un pic de
   vitesse dont la MEDIANE depasse le P90 du temoin aleatoire apparie. Le repulseur pousse
   les autres : si le film date le geste, l'effet doit se voir sur la victime.

**TEMOIN POSITIF OBLIGATOIRE** : le meme instrument doit retrouver (i) le grappin sur le tag 3
d'i59 (>= 75 % rang 4) et (ii) le propulseur sur le tag 1 d'i57 (>= 0,3 impulsion par vie de
rang propulseur, 0,000 sur les rangs temoins). S'il ne les retrouve pas, aucun negatif ne
sera publie : c'est l'instrument qui sera en cause.

**Si les criteres 1 et 3 echouent**, la porte (a) est fermee cote FLUX, et Ghidra ne sert plus
qu'a documenter ce que `dst[2]` gouverne (et donc si une population entiere reste invisible).

### 1.2 Porte (b) — l'entite ti=37 PORTEE, jointe par les handles d'i26

R8 laisse la jointure d'identite des entites ti=37 a 13 % parce qu'elle passe par les records
de CREATION. La refaire par les HANDLES d'i26 du bipede (`ScanFilmUnitEquipment`, deja porte)
change la source d'identite : **ce n'est plus la pose de creation qui nomme l'objet, c'est le
RANG i48 du bipede qui le tient**. Le canal i48 couvre bien mieux (202 lectures de rang sur
`00ba2e1c`, une par vie environ), et il est deja le juge d'identite du propulseur.

**Ce que j'attendrais si le repulseur etait la** :

1. **Jointure** — la couverture d'identite des vies ti=37 passe de 13 % a > 40 %. En dessous,
   le negatif restera FAIBLE et sera publie comme tel.
2. **Charges** — sur les entites d'identite « repulseur », au moins une BAISSE de
   `charges-remaining` par vie porteuse en moyenne ( >= 0,5 ), contre ~0 sur les entites dont
   le rang est un equipement sans charge consommable.
3. **Activation** — ou, a defaut de baisse, une transition d'`equipment-activated-component`
   concentree sur ces memes entites.
4. **Oracle** — a l'instant de la baisse (ou de la transition), le VOISIN le plus proche du
   porteur montre un pic de vitesse au-dessus du P90 du temoin aleatoire.

**TEMOIN POSITIF OBLIGATOIRE** : les entites d'identite « grappin » doivent montrer leur
baisse de charge (ou leur transition d'activation) aux 1 101 instants certains de
`grappleLines[]`. Sans ce controle, un zero sur le repulseur ne dirait rien.

### 1.3 Porte (c) — i56 `biped-spartan-ability-energy`

A n'instruire que si (a) et (b) echouent. Critere : la SERIE d'energie d'un porteur de
repulseur doit montrer une DECROISSANCE discrete (marche) absente chez les temoins ; aucun
instant precis n'est attendu du canal (0,083 % des records).

### 1.4 La sortie de secours, et ce qu'elle exige pour etre publiee

Si les trois portes echouent, la conclusion candidate est : **le film porte l'EFFET du
repulseur (le deplacement replique de la victime) sans porter le GESTE (l'activation du
porteur)**, et le visionneur du jeu rejoue ce qu'il voit parce que la physique de la victime
est deja dans les positions.

**Cette conclusion ne sera PAS publiee comme une supposition.** Pour etre publiee elle exige
une mesure a deux faces, ecrite ici avant d'etre faite :

- **Face victime** : les bipedes situes a moins de 6 m d'un porteur de repulseur montrent, sur
  la duree de vie du porteur, un EXCES de discontinuites de vitesse (accelerations
  horizontales au-dela d'un seuil) par rapport aux bipedes voisins d'un porteur temoin
  (detecteur de menaces, champ de reparation — des equipements sans effet cinetique).
- **Face porteur** : le porteur de repulseur ne montre AUCUN exces symetrique (sinon on serait
  revenu a un geste lisible).
- **Temoin positif** : la meme mesure, appliquee au propulseur, doit montrer l'exces sur le
  PORTEUR et pas sur le voisin — la symetrie inverse. Si l'instrument ne separe pas ces deux
  cas connus, il ne separera rien.

---

## 2. [ETABLI — PORTE (a) FERMEE] La branche `tag == 3` d'i57 est le GRAPPIN

Instrument : `filmdec/r9_i57_tag3_research_test.go`, `TestR9I57Tag3`.

**Comment le bit `a` se lit sans rien deviner.** `consumeSpartanAbilityTag3` rend `false` SI ET
SEULEMENT SI `a != 0`, et `walkRecordTo` ne visite le composant que s'il est porte. Donc
`walkRecordTo == true` sur une lecture tag==3 equivaut a `a == 0`, et l'inverse a `a == 1`.
Aucune copie de la marche de production n'a ete faite (regle des <= 2 copies) ; le hook d'i57,
lui, publie le tag meme quand le record est abandonne. Reserve honnete : un debordement de
payload donnerait le meme `false` — il est marginal (les compteurs `masque` et `lu` sont
egaux a l'unite pres sur les cinq films).

### 2.1 Le resultat, sur cinq films

| film | tag=3 total | **`a == 0`** | `a == 1` | rangs i48 des `a == 1` | concentr. rang 4 |
|---|---|---|---|---|---|
| `00ba2e1c` | 40 | **0** | 40 | 4x28, inconnu x10, 6x2 | 93,3 % |
| `06dfe6d9` | 97 | **0** | 97 | 4x62, inconnu x28, 6x5, 2x2 | 89,9 % |
| `084a804d` | 83 | **0** | 83 | 4x70, inconnu x13 | 100 % |
| `11de8353` | 32 | **0** | 32 | 4x22, inconnu x10 | 100 % |

**PREMIER FAIT, et il etait inconnu : la moitie PORTEE de la branche n'est JAMAIS empruntee.**
`a == 0` rend zero lecture sur les quatre films. Tout le tag 3 d'i57 passe par la branche
gardee par l'octet d'etat runtime, donc **tout record dont le masque annonce i57 avec tag 3 est
abandonne apres i57**. C'est un angle mort reel du decodeur (2 a 3 % des records i57), a
consigner ; ce n'est pas une cachette pour le repulseur, comme la suite le montre.

**SECOND FAIT, decisif : le tag 3 d'i57 est le GRAPPIN.** 90 a 100 % des lectures a rang connu
tombent sur le rang 4. C'est le jumeau PREDIT du tag 3 d'i59 (l'ancre du grappin, deja en
production), et les deux comptes se recouvrent : sur `00ba2e1c`, i59 lu = 3 234 contre i57
lu = 3 187, ecart 47 — exactement le nombre de lectures tag=3 d'i59 (87) que le tag 3 d'i57 ne
rend pas (40). i59 precede i57 dans le masque, et quand le corps d'ancre d'i59 desynchronise,
i57 n'est plus atteint.

### 2.2 Les quatre criteres pre-inscrits, juges tels qu'ecrits

| critere pre-inscrit | seuil | mesure | verdict |
|---|---|---|---|
| 1 concentration sur le rang du repulseur | >= 75 % | 6,7 % / 7,2 % / 0 % / 0 % | **ECHOUE** |
| 2 non-confusion avec grappin/propulseur | — | 93-100 % sur le GRAPPIN | **ECHOUE** |
| 3 facteur par vie repulseur / meilleur temoin | >= 5 | voir ci-dessous | **ECHOUE** |
| 4 oracle du voisin > P90 aleatoire | — | voir ci-dessous | **ECHOUE** |

Episodes `tag==3, a==1` par VIE (le tableau qui a tranche le propulseur au lot R8) :

| film | rang 4 grappin | rang 6 repulseur | rang 5 propulseur | autres rangs |
|---|---|---|---|---|
| `00ba2e1c` | **0,800** (24/30) | 0,080 (2/25) | 0,000 (0/15) | 0,000 |
| `06dfe6d9` | **1,774** (55/31) | 0,185 (5/27) | 0,000 (0/16) | 0,133 (rang 2) |
| `084a804d` | **1,558** (67/43) | 0,000 (0/16) | 0,000 (0/9) | 0,000 |
| `11de8353` | **0,786** (22/28) | 0,000 (0/22) | 0,000 (0/28) | 0,000 |

Le facteur va dans le sens CONTRAIRE : le grappin emet 10 a 20 fois plus que le repulseur, et
sur deux films le repulseur emet zero alors qu'il est porte par 16 et 22 vies.

Oracle du voisin, aux instants `a == 1` (rayon 6 m, meme fenetre que R8) :

| film | voisins | med pic voisin | temoin aleatoire (med / p90) |
|---|---|---|---|
| `00ba2e1c` | 51 | 3,10 | 3,18 / 4,06 |
| `06dfe6d9` | 39 | 3,29 | 3,28 / 5,33 |
| `084a804d` | 37 | 3,14 | 3,22 / 4,02 |
| `11de8353` | 17 | 3,04 | — |

**Le voisin ne bouge pas plus qu'au hasard** — il est meme au-dessous de la mediane du temoin
sur trois films sur quatre. Et 18 a 70 de ces lectures n'ont AUCUN voisin a 6 m.

**TEMOINS POSITIFS : les deux passent.** (i) Le tag 3 d'i59 retrouve le grappin — 61/65,
114/127, 155/155, 55/56 lectures a rang connu sur le rang 4, soit 90 a 100 %, au-dessus des
75 % exiges. (ii) Le tag 1 d'i57 retrouve le propulseur — 0,533 / 0,438 / 0,444 impulsion par
vie de rang propulseur contre 0,000 sur tous les rangs temoins. **L'instrument voit donc bien
ce qu'il doit voir ; son zero sur le repulseur est un vrai zero.**

### 2.3 Ce que la porte (a) laisse derriere elle

- La branche `tag == 3 / a == 1` d'i57 est la charge PREDITE du grappin. Le report 3 du
  registre R8 est **solde**, et Ghidra n'y ajouterait qu'une description du contenu de la
  charge — plus une identite de capacite.
- **Decouverte laterale, a consigner** : la moitie portee (`a == 0`) est morte ; le decodeur
  perd donc 2 a 3 % des records i57 (et tout ce qui suit i57 dans ces records). Meme chose du
  cote d'i59 : 47 records sur `00ba2e1c` perdent i57 parce que l'ancre d'i59 desynchronise.
  Ces pertes touchent les vies de GRAPPIN, pas celles de repulseur.

---

## 3. [ETABLI — PORTE (b) DISQUALIFIEE PAR SON TEMOIN] L'entite ti=37 portee

Instrument : `filmdec/r9_ti37_identite_research_test.go`, `TestR9Ti37Identite`. Film
`00ba2e1c` (le film de reference de R8).

### 3.1 La jointure par les handles d'i26 est PIRE que celle qu'elle devait remplacer

| grandeur | mesure |
|---|---|
| emissions d'i26 dans le film | **170** |
| entrees de liste, dont presentes | 297, dont 291 |
| appartenance a la bande de slots ti=37 | 192 / 291 = **66,0 %** |
| entites distinctes vues en handle | 267 |
| entites distinctes vues en ETAT (charges/activated/...) | **1 641** |
| entites a la fois en handle et en etat | **52 (3,2 %)**, nommees 50 (3,0 %) |
| rappel : jointure par les POSES (canal de R8) | 217 / 1 641 = **13,2 %** |

**Le critere 1 pre-inscrit (couverture > 40 %) ECHOUE, et de loin : 3,2 % contre 13,2 % au
canal qu'il devait remplacer.** La cause est structurelle et se lit dans le premier chiffre :
i26 n'emet que **170 fois par film**. Ce n'est pas un canal continu de portage, c'est une
annonce d'inventaire. L'espoir du report 4 de R8 (« la couverture passerait a celle d'i26 »)
etait donc mal fonde : la couverture d'i26 est petite.

### 3.2 Le tableau par identite — et le temoin positif qui abat la piste

Identite par la POSE quand elle existe (canal de R8), a defaut par le rang i48 du porteur :

| identite | entites | records d'etat | lectures de charge | **BAISSES** | transitions `activated` |
|---|---|---|---|---|---|
| **grapple** (TEMOIN POSITIF) | 37 | 61 | 2 | **0** | **0** |
| **thruster** (TEMOIN POSITIF) | 19 | 23 | 1 | **0** | **0** |
| **repulsor** (CIBLE) | 11 | 29 | 19 | **0** | **0** |
| wall / wall_panel | 7 | 7 | 3 | 0 | 0 |
| sensor | 9 | 10 | 1 | 0 | 0 |
| repair_field | 9 | 12 | 3 | 0 | 0 |
| shroud_screen | 2 | 2 | 1 | 0 | 0 |
| grenades (4 familles) | 123 | 159 | 30 | 2 | 0 |
| `?` (identite inconnue) | 1 394 | 2 330 | 785 | 45 | 6 |

**LE TEMOIN POSITIF EST MUET, ET C'EST LUI QUI TRANCHE.** Le GRAPPIN a 1 101 instants d'usage
CERTAINS dans ce corpus (`grappleLines[]`, canal independant) et 37 entites ti=37 nommees :
il ne produit **aucune** baisse de charge et **aucune** transition d'activation. Le PROPULSEUR,
dont R8 a etabli les instants d'usage, est muet de la meme facon.
**Par la regle du chantier, un canal qui ne retrouve pas deux usages connus ne peut pas
prononcer de negatif sur un troisieme.** Le zero du repulseur (0 baisse sur 19 lectures de
charge) n'est donc PAS publie comme un negatif : c'est le CANAL qui est disqualifie.

**Pourquoi il l'est, mecaniquement.** Les composants d'etat de ti=37 sont lus dans les records
DELTA, et le decodage ne suit que les records qui portent une position (limite documentee par
`equipment_creation_width.go` : « un objet pose cesse d'emettre des qu'il s'immobilise »). Un
equipement PORTE ne bouge pas de son propre chef : 11 entites de repulseur ne rendent que
29 records d'etat sur tout un film. Le canal decrit des objets DANS LE MONDE, pas des objets
EN MAIN. C'est exactement la reserve que l'en-tete d'`equipment_state.go` ecrit depuis le
2026-08-17 (« ce que le balayage compte, ce sont des entites, pas des utilisations »).

### 3.3 [DECOUVERTE LATERALE] Les 45 « baisses de charge » ne sont pas des charges

Les baisses tombent presque toutes sur des entites sans identite, et **leurs valeurs excluent
la lecture « charges »** : une charge d'equipement vaut 0 a 3, or on lit 247->244, 251->96,
244->2, 240->223. Et elles descendent par paliers rapides — l'entite (1877,2) passe de 223 a
215 en 130 ms, en quatre lectures. C'est le profil d'un COMPTE A REBOURS de tics, pas d'un
compteur de charges. Deux lectures possibles, aucune tranchee ici : soit le champ R(8)
`equipment-charges-remaining-component` porte autre chose que son nom pour ces entites, soit
la marche atterrit sur un autre composant pour cet archetype.
**Hors perimetre de ce lot** (regle « zero fix opportuniste ») : consigne au registre des
reports, par. 7.

---

## 4. [ETABLI] Le tag d'i57/i59 est SATURE — le repulseur ne peut PAS y etre

Fait de cadrage, tire des tables du par. 2 et jamais publie ainsi : le tag externe est un
`R(2)`, il a QUATRE valeurs, et **les quatre sont maintenant attribuees** :

| tag | lectures (`00ba2e1c`) | rangs i48 | lecture |
|---|---|---|---|
| 0 | 1 572 | TOUS les rangs, a proportion de leur portage | etat de repos (A) |
| 1 | 10 | rang 5 a 8/9 | **impulsion — PROPULSEUR** (R8) |
| 2 | 1 565 | TOUS les rangs, memes proportions que le tag 0 | etat de repos (B) |
| 3 | 40 (i57) / 87 (i59) | rang 4 a 90-100 % | **ancre — GRAPPIN** (production) |

Les tags 0 et 2 comptent presque exactement le meme nombre de lectures (1 572 et 1 565) et
portent la MEME distribution de rangs : ce sont deux etats de repos, pas deux capacites.
**Il ne reste donc aucune valeur libre dans ce composant.** Chercher le repulseur dans
« spartan-ability » etait chercher dans un tiroir plein. Ce constat explique aussi, sans le
supposer, pourquoi le repulseur ne se comporte pas comme le propulseur : le propulseur EST une
capacite spartan (il pousse son porteur) ; le repulseur est un EQUIPEMENT qui agit sur autrui.

---

## 5. [ETABLI — NEGATIF] i54 rejuge par l'IDENTITE : toujours pas le repulseur

Instrument : `filmdec/r9_i54_identite_research_test.go`, `TestR9I54Identite`.

**Pourquoi rouvrir un canal ferme.** R8 (par. 7.2) a lu la charge utile d'i54
`biped-mobility-action` et l'a declaree muette **sur le seul oracle de la bouffee de vitesse
du porteur**. Sa conclusion ecrite — « i54 n'est pas le canal de l'usage du PROPULSEUR » — est
juste et ne dit rien du repulseur : **un canal juge a la vitesse ne peut pas trancher une
capacite qui n'en produit pas.** Ce lot le rejuge donc a l'IDENTITE, l'oracle qui a tranche le
propulseur.

`00ba2e1c` — 240 645 records, 1 021 avec i54, 978 corps lus, 53 episodes `flag1==1` :

| cellule | lectures | episodes | med pic | med pic VOISIN | rangs i48 |
|---|---|---|---|---|---|
| B7=0 | 700 | 37 | 3,05 | 3,30 | 6x112 4x110 10x39 1x37 5x19 (+383 inconnus) |
| B7=1 | 116 | 6 | 3,04 | 3,47 | 4x21 1x19 |
| B7=12 | 45 | 3 | 2,81 | — | 6x19 10x7 |
| B7=3 | 45 | 3 | 2,15 | — | 23x19 1x18 4x8 |
| *temoin aleatoire* | 8 320 | — | 3,18 (p90 4,06) | — | — |

`06dfe6d9` — 2 193 emissions, 14 valeurs de B7 : la plus grosse cellule (B7=0, 1 425 lectures)
porte de nouveau TOUS les rangs (6x224 4x168 5x114 23x111 10x58).

Episodes par VIE, valeur par valeur (le tableau du par. 8.8 de R8) — B7=0 sur `00ba2e1c` :
rang 6 (repulseur) 0,240 ; rang 4 0,200 ; rang -1 0,220 ; rang 10 0,111 ; rang 5 0,067. Sur
`06dfe6d9` : rang 6 0,444 ; rang 4 0,290 ; rang 5 0,375 ; rang 23 0,240 ; rang 12 0,667.

**VERDICT, juge sur les seuils pre-inscrits.** Aucune cellule d'i54 n'atteint 75 % de
concentration sur le rang du repulseur — la plus grosse en est a 27 % (112/407 a rang connu) —
et aucun facteur par vie n'approche 5 (le meilleur est 1,2 entre le rang 6 et le rang 4). Le
voisin ne bouge pas davantage (3,30 median contre 3,18 au temoin aleatoire). **i54 reste
ferme, et il l'est desormais sur les DEUX oracles.**

---

## 6. [ETABLI — NEGATIF, ET SOUS-PUISSANT : DIT COMME TEL] La face VICTIME

Instrument : `filmdec/r9_poussee_research_test.go`, `TestR9Poussee`. C'est la mesure de la
sortie de secours pre-inscrite au par. 1.4 : si le film ne porte pas le GESTE, porte-t-il au
moins l'EFFET — la projection de la victime, deja presente dans les positions repliquees ?

Detecteur (seuils ecrits avant, repris de l'oracle de R8) : une POUSSEE est un pas ou la
vitesse horizontale passe de <= 3,0 m/s a >= 6,0 m/s en <= 250 ms ; elle est attribuee a un
porteur situe a <= 6,0 m si la nouvelle vitesse s'ELOIGNE de lui. Denominateur : le temps
d'exposition (secondes-victime a <= 6 m de ce porteur).

| film | rang | expo (min) | poussees VOISIN | par min | vie (min) | poussees PROPRES | par min |
|---|---|---|---|---|---|---|---|
| `00ba2e1c` | **5 propulseur** | 7,82 | 1 | 0,128 | 9,84 | **15** | **1,524** |
| | **6 repulseur** | 11,06 | 10 | **0,904** | 16,83 | 14 | 0,832 |
| | 4 grappin | 14,72 | 9 | 0,611 | 17,72 | 8 | 0,451 |
| | 2 mur | 5,38 | 4 | 0,743 | 7,14 | 1 | 0,140 |
| | 23 champ de rep. | 6,89 | 5 | 0,725 | 10,48 | 3 | 0,286 |
| `06dfe6d9` | **5 propulseur** | 7,32 | 8 | 1,093 | 12,78 | **45** | **3,522** |
| | **6 repulseur** | 11,77 | 9 | **0,764** | 20,80 | 53 | 2,548 |
| | 4 grappin | 7,53 | 8 | 1,062 | 15,47 | 38 | 2,456 |
| | 23 champ de rep. | 8,32 | 9 | 1,082 | 19,62 | 34 | 1,733 |
| `084a804d` | **5 propulseur** | 7,56 | 3 | 0,397 | 11,15 | **9** | **0,807** |
| | **6 repulseur** | 5,80 | 1 | **0,172** | 14,21 | 9 | 0,633 |
| | 4 grappin | 14,38 | 7 | 0,487 | 24,81 | 14 | 0,564 |

**Critere 2 (temoin positif par symetrie inverse) : PASSE, mais faiblement.** Le rang du
propulseur mene la colonne « poussees PROPRES par minute » sur les trois films (1,524 / 3,522 /
0,807), au-dessus de tous les rangs bien peuples. L'ecart avec le second n'est que de 1,4 a
1,8 — l'instrument voit ce qu'il doit voir, mais il ne le voit pas de loin.

**Critere 1 (cible) : ECHOUE sur les trois films.** Le rang du repulseur mene la colonne
« voisin » sur `00ba2e1c` (0,904) mais avec un facteur de 1,22 sur le meilleur temoin, quand
2,0 etait exige ; et il est le DERNIER des rangs bien peuples sur les deux autres films (0,764
et 0,172). Il n'y a pas de signal reproductible.

**Critere 3 (non-retour) : PASSE.** Le porteur de repulseur ne montre aucun exces de poussee
propre (0,832 / 2,548 / 0,633, toujours sous le propulseur).

**CE QUE CETTE MESURE PERMET DE DIRE, ET CE QU'ELLE NE PERMET PAS.** Elle ne permet PAS de
conclure « le film porte l'effet » : la face victime ne se distingue pas du bruit. Elle ne
permet pas non plus de conclure l'inverse avec force, et **c'est ecrit** : son temoin positif
ne separe qu'a 1,4-1,8, les denominateurs sont de 2 a 12 minutes d'exposition et les comptes
de 1 a 10 — le bruit de Poisson y est du meme ordre que l'ecart cherche. **La sortie de
secours du par. 1.4 n'est donc PAS validee, et elle n'est pas refutee non plus.** Elle reste
une hypothese, et elle est publiee comme telle.

### 6.1 [CORRECTION A R8] Cette mesure affaiblit un argument de R8, et il faut le dire

R8 a ferme sa « piste A » (les 87 poses `repulsor/deployed` datent-elles un usage ?) sur
DEUX arguments : l'oracle physique (par. 4 de R8) et le canal i48 (par. 5, zero `spent` en
coincidence sur 76 poses). **Le premier des deux vient de perdre sa puissance** : la mesure du
present paragraphe etablit que la poussee du repulseur n'est PAS detectable dans les positions
repliquees, ni sur le porteur ni sur le voisin. Un oracle aveugle a l'effet ne peut pas refuter un canal qui
daterait le geste.

Le SECOND argument tient, lui, et il est independant : zero `spent` en coincidence, et un
quart des poses coincidant avec un `taken` de la meme famille (la signature de l'ECHANGE).
La piste A reste donc fermee — mais **sur un argument au lieu de deux**, et le rapport R8
gagnerait a porter cette nuance.

---

## 7. [ETABLI — GHIDRA, STATIQUE] Ce que l'exe dit, et ce qui reste ouvert

Instance headless lancee selon l'annexe C.1 de R6 (JDK 25, contournement
`-Djdk.net.unixdomain.tmpdir=Q:\nexistepas`), programme `/HaloInfinite.exe` du projet
`HI.gpr`, requetes HTTP `search_strings` / `get_xrefs_to` / `read_memory` /
`decompile_function`. **Serveur arrete en fin de lot, verrou libere, `HI.gpr` et `HI.rep`
intacts** (`HI.gpr` fait 0 octet depuis sa creation du 2026-06-04 : c'est un marqueur, pas
une donnee).

### 7.1 Ce que l'exe nomme

| chaine | adresse | lecture |
|---|---|---|
| `RepulsorField` | `143d106c0` | dans une TABLE DE NOMS d'equipement, voisine de `TreeOfLife`, `ActiveCamo`, `Transloc...`, `HealthPa...` — c'est le nom interne du repulseur dans l'enumere des equipements, pas une classe d'objet spawnable |
| `.?AVKnockbackTargetComponent@Objects@i343@@` | `1447d90c0` | RTTI d'un **KnockbackTargetComponent** — un composant existe pour la CIBLE d'une poussee |
| `EquipmentKnockbackPlayer` / `EquipmentKnockbackRequest` | `143c96710` / `143c96750` | les types d'evenement 104 et 119, absents des films (R7) |
| `IsInRepulsor`, `IsKnockedOffMapByRepulsor`, `Unit_IsInKnockback` | `1437xxxxx` | des accesseurs de SCRIPT (table de registration `FUN_140dd208c`), pas des champs repliques |

**`KnockbackTargetComponent` existe dans le moteur — et il n'est PAS dans la liste repliquee
du bipede.** L'archetype bipede des films compte exactement **64 composants (i0..i63)**, la
borne dure du masque (`worldObjectIndexBits` = 6), et la liste est PLEINE ; aucun de ses 64
noms ne contient « knockback ». Le composant de la victime existe donc cote moteur et n'est
pas transmis.

### 7.2 [DECOUVERTE] Cinq composants de ti=37 sont lus puis JETES — dont le SIGNAL DE COMMANDE

L'archetype ti=37 compte **31 composants** ; `equipment_state.go` n'en publie que six. Les
cinq autres sont lus par `consumeByName` et leurs valeurs jetees :

| composant | index | deser | grammaire |
|---|---|---|---|
| **`equipment-control-signal-component`** | i22 | `FUN_14101cd94` | **`R(4)` + `R(1)`[+quantStat]** |
| `equipment-being-hacked-component` | i25 | `FUN_142ed441c` | — |
| `equipment-tracked-object-handles-stack-component` | i28 | `FUN_140f72dec` | pile de handles d'objets SUIVIS |
| `equipment-command-tick-component` | i29 | `FUN_140e0a564` | — |
| `equipment-has-infinite-uses-component` | i30 | `FUN_142ed4640` | `R(1)` |

**`control-signal` est le seul champ de l'archetype dont le NOM annonce un ordre et non un
etat, et sa charge est un `R(4)` — seize valeurs, la forme d'un enumere de commande.**
Personne ne l'avait jamais lu. Le paragraphe suivant le lit.

### 7.3 [ETABLI — LE CANAL ti=37 EST DU BRUIT] Le recensement du masque tranche

Instrument : `filmdec/r9_i22_signal_research_test.go`, `TestR9I22Signal`. Il publie le
RECENSEMENT DU MASQUE de ti=37 sur `00ba2e1c` (91 948 records reconnus) — et **ce recensement
porte son propre etalon de bruit** : l'archetype n'a que 31 composants (i0..i30), donc **tout
comptage sur i31..i63 est une FAUSSE RECONNAISSANCE**.

| composant | annonces | lecture |
|---|---|---|
| i0 position | 91 948 | (definition du record) |
| i1 velocite | 42 322 | signal reel |
| i2 forward-and-up | 34 416 | signal reel |
| i3 velocite angulaire | 27 752 | signal reel |
| **i31..i63 — N'EXISTENT PAS** | **108 a 461, mediane ~210** | **PLANCHER DE BRUIT** |
| i27 charges-remaining | 856 | 4x le plancher |
| i26 energy-delay | 728 | 3,5x le plancher |
| i24 energy | 414 | 2x le plancher |
| i20 deployed | 325 | 1,5x le plancher |
| **i21 activated** | **204** | **AU PLANCHER** |
| **i22 control-signal** | **126** | **SOUS le plancher** |

**Les records delta de ti=37 ne portent, au-dessus du bruit, que la position, la vitesse et
l'orientation.** `equipment-activated` et `equipment-control-signal` y sont indiscernables de
la fausse reconnaissance.

Et la lecture des 126 valeurs d'i22 le confirme sans appel : elles se repartissent
**uniformement sur les seize valeurs du `R(4)`** (1 a 17 occurrences chacune) et **chacune sur
une entite differente**. C'est la signature exacte du bruit : un champ de 4 bits lu a une
position aleatoire rend une loi uniforme. Un enumere de commande reel aurait une ou deux
valeurs dominantes.

**Consequence : le par. 3 de ce rapport doit etre durci.** Le canal des composants d'etat de
ti=37 n'est pas seulement « disqualifie par son temoin positif » — il ne transmet
essentiellement rien dans les paquets delta. Les 45 « baisses de charge » du par. 3.3, dont
les valeurs (247, 251, 244) etaient deja incompatibles avec des charges d'equipement, sont
selon toute vraisemblance **du bruit de reconnaissance**. Le report 4 du registre R8 est donc
solde par la NEGATIVE : il n'y avait rien a joindre.

### 7.4 Ce que le visionneur du jeu utilise — ce qui est etabli, et ce qui ne l'est pas

**Ce qui est ETABLI :** le repulseur n'est ni dans le canal des evenements (R7 : 96,5 % des
listes marchees integralement, zero occurrence des types 104/119/42/43), ni dans le tag
d'i57/i59 (dont les quatre valeurs sont maintenant toutes attribuees : repos, propulseur,
grappin — par. 4), ni dans i54 sur l'un OU l'autre de ses deux oracles (par. 5), ni dans les
composants d'etat de ti=37, qui ne transmettent rien au-dessus du bruit (par. 7.3), ni dans
les poses `deployed` (R8, argument i48). Et son EFFET sur les positions repliquees ne se
distingue pas du bruit (par. 6).

**Ce qui N'EST PAS etabli, et qui est la reponse honnete a la question posee :** je ne peux
PAS nommer le canal que le visionneur Theater emploie. Les candidats restants, par ordre de
vraisemblance, sont ecrits au registre des reports (par. 9). Le plus serieux est le type
d'evenement **14 `PlayEffectOnObject`**, qui EXISTE dans les films (il bloque 48 listes sur le
parc de 12 films, R7 par. 3) et dont la grammaire n'est PAS portee : c'est un evenement
generique « joue cet effet sur cet objet », exactement la forme qu'aurait la souffle du
repulseur cote client. **Ce n'est pas une conclusion, c'est la premiere chose a instruire.**

Deux voisins ont ete elimines en chemin : le type **103 `EquipmentSpawnedObject`** (73 tetes
sur `00ba2e1c`) est celui des DEPLOYABLES et R5 l'a deja caracterise (il date les poses a
~100 ms, ne nomme pas le poseur) ; le type **105 `EquipmentObjectKnockedBack`** existe mais
est rarissime (8 tetes sur 12 films, R7) et concerne l'OBJET pousse, pas le joueur.

### 7.5 [ETABLI — PORTE (c) FERMEE] Le masque BIPEDE balaye SANS ANCRE, ses 64 composants

Instrument : `filmdec/r9_masque_research_test.go`, `TestR9Masque`. Il traite la porte (c) du
registre R8 (i56 `biped-spartan-ability-energy`) dans sa forme la plus forte : au lieu
d'instruire UN composant, il les interroge TOUS.

**Pourquoi le balayage de R8 devait etre refait.** Celui du par. 7.3 de R8 etait ancre sur des
BOUFFEES DE VITESSE — et le par. 6 de ce rapport etablit que le repulseur n'en produit pas.
L'ancrage etait aveugle a la cible. Un RECENSEMENT n'a besoin d'aucune ancre : si l'usage du
repulseur est transmis par un composant quelconque, ce composant doit etre ANNONCE plus
souvent quand un repulseur est porte. Le recensement ne lit AUCUN deser (en-tete et masque
seulement) : il ne souffre donc d'aucune desynchronisation.

`06dfe6d9`, taux d'annonce par record, par rang porte (extrait ; 11 rangs a >= 200 records) :

| composant | r1 (14 214) | r4 grappin (27 608) | r5 propul. (22 750) | **r6 repuls. (36 939)** | r9 surbouc. (1 743) | r10 (24 404) | r23 (34 987) |
|---|---|---|---|---|---|---|---|
| i5 `object-shield-vitality` | 0,046 | 0,101 | 0,121 | **0,187** | **0,561** | 0,152 | 0,097 |
| i28 `unit-active-camo-state` | 0,011 | **0,021** | 0,016 | 0,017 | 0,015 | 0,017 | 0,016 |
| i54 `biped-mobility-action` | 0,003 | 0,009 | 0,006 | 0,007 | 0,022 | 0,003 | 0,005 |
| **i56 `spartan-ability-energy`** | 0,001 | **0,004** | 0,001 | **0,001** | 0,000 | 0,001 | 0,001 |
| i57 `spartan-ability` | 0,010 | **0,019** | 0,014 | 0,015 | 0,012 | 0,016 | 0,014 |
| i59 `-non-predicted-state` | 0,010 | **0,023** | 0,014 | 0,015 | 0,012 | 0,016 | 0,014 |

**LE CONTROLE PRE-INSCRIT ECHOUE, ET IL FAUT LE DIRE.** J'avais pose i28
`unit-active-camo-state` sur le rang du camouflage comme temoin positif obligatoire. Il ne
ressort pas : le taux d'i28 est plat (0,007 a 0,021) et le rang 8 n'en est pas le maximum. La
regle du chantier veut alors qu'aucun negatif ne soit publie sur ce seul instrument.

**DEUX CONTROLES ONT NEANMOINS PASSE, et ils sont notes comme POSTERIEURS a la mesure** (ils
ne valent donc pas ce qu'aurait valu un temoin pre-inscrit, et le negatif ci-dessous est
publie avec cette reserve) :

1. **Le SURBOUCLIER** — i5 `object-shield-vitality` rend **0,561** sur le rang 9 contre 0,215
   au deuxieme rang, soit **2,6x**. Un equipement qui agit sur le bouclier rend bavard le
   composant du bouclier : le recensement le VOIT.
2. **Le GRAPPIN** — i59 rend 0,023 sur le rang 4 contre 0,010-0,016 ailleurs (1,4 a 2,3x), et
   i57 0,019 ; le grappin est le seul equipement dont l'usage passe par ces composants, et il
   y laisse sa trace. i56 le suit (0,004 contre 0,001 partout ailleurs).

**LE RESULTAT : le rang du REPULSEUR ne mene AUCUN des 64 composants.** Il est pourtant le
rang le MIEUX echantillonne du film — 36 939 records, plus que le grappin (27 608), le
propulseur (22 750) ou le champ de reparation (34 987). Le critere pre-inscrit demandait un
facteur >= 3 : le repulseur n'atteint 1,0 nulle part, c'est-a-dire qu'il n'est le maximum
d'aucune colonne.

**Porte (c) est fermee** : i56 `biped-spartan-ability-energy` est bien un composant qui reagit
a l'equipement — mais au GRAPPIN (0,004 contre 0,001), pas au repulseur.

**Ce que ce balayage ne peut PAS voir, et c'est dit** : un usage transmis comme une VALEUR
a l'interieur d'un composant annonce en permanence — le cas exact du propulseur, tag 1 d'i57 —
n'apparait pas dans un recensement d'ANNONCES. Le recensement ferme la question « un composant
REAGIT-il au port d'un repulseur », pas la question « une valeur le fait ». Cette seconde
question est fermee ailleurs, par la saturation du tag (par. 4).

---

## 8. VERDICT

### 8.1 Le repulseur — NON TROUVE, et l'inventaire est desormais complet

| canal | ce qui a ete mesure | verdict |
|---|---|---|
| Evenements 104 / 119 / 42 / 43 | R7 : 236 321 evenements, 96,5 % des listes marchees | ABSENT (acquis) |
| Poses `equipmentPlacements` `deployed` | R8 : oracle physique + canal i48 | ne datent aucun usage (argument i48 seul, cf. par. 6.1) |
| i57/i59 **tag == 1** | R8 : 22 films, 0,011 impulsion/vie sur 90 vies | ABSENT (acquis) |
| **i57 tag == 3, branche `a`** | **R9 : 4 films, 100 % des lectures en `a==1`, 90-100 % sur le rang du GRAPPIN, 0-7 % sur le repulseur, oracle du voisin sous le temoin** | **ABSENT — c'est le grappin predit** |
| **le TAG lui-meme** | **R9 : les 4 valeurs du `R(2)` sont attribuees (0 et 2 repos, 1 propulseur, 3 grappin)** | **AUCUNE PLACE LIBRE** |
| **i54 `biped-mobility-action`, par l'IDENTITE** | **R9 : 2 films, aucune cellule au-dela de 27 % sur le rang repulseur, facteur par vie 1,2 (5 exiges)** | **ABSENT — sur les DEUX oracles** |
| **ti=37 : jointure d'identite par les handles i26** | **R9 : couverture 3,2 %, pire que les 13,2 % des poses ; i26 n'emet que 170 fois par film** | **jointure IMPRATICABLE** |
| **ti=37 : composants d'etat (activated, charges, control-signal)** | **R9 : recensement du masque avec etalon de bruit — i21 au plancher, i22 sous le plancher ; les 126 valeurs d'i22 sont uniformes sur 16, une par entite** | **LE CANAL NE TRANSMET RIEN** |
| **la face VICTIME (positions repliquees)** | **R9 : 3 films, taux de poussee des voisins du repulseur au-dessus d'un seul temoin sur trois films ; temoin positif ne separant qu'a 1,4-1,8** | **PAS DE SIGNAL — mesure sous-puissante, dit** |
| **le MASQUE BIPEDE entier, SANS ANCRE (dont i56, porte c)** | **R9 : taux d'annonce des 64 composants par rang porte ; le recensement DETECTE le surbouclier (2,6x sur i5) et le grappin (1,4-2,3x sur i57/i59/i56) — le repulseur, rang le mieux echantillonne du film (36 939 records), ne mene AUCUN composant** | **ABSENT — porte (c) fermee** |

**Le negatif est maintenant STRUCTUREL et non plus circonstanciel** : ce n'est plus « on n'a
pas trouve », c'est « les tiroirs sont comptes et ils sont pleins ». Le composant de capacite
spartan a quatre valeurs de tag et les quatre sont attribuees ; l'archetype bipede a
exactement 64 composants (la borne dure du masque) et aucun ne porte de knockback ;
l'archetype d'equipement a 31 composants dont les records delta ne transmettent que la
position, la vitesse et l'orientation ; le canal des evenements est marche a 96,5 % et ne
porte aucun des types de poussee.

### 8.2 Ce qui a ete gagne en chemin (et qui vaut plus que le negatif)

1. **Le report 3 du registre R8 est solde** : la branche `tag == 3 / a == 1` d'i57 est la
   charge PREDITE du grappin. Et un angle mort a ete quantifie : la moitie PORTEE de cette
   branche (`a == 0`) n'est **jamais** empruntee, donc **tout** record annoncant i57 avec le
   tag 3 est abandonne apres i57 (2 a 3 % des records i57, plus 47 records par film ou
   l'ancre d'i59 desynchronise avant meme d'atteindre i57).
2. **Le report 4 du registre R8 est solde par la negative** : la jointure d'identite de ti=37
   ne se refait PAS par les handles d'i26 (170 emissions par film, couverture 3,2 %), et il
   n'y avait de toute facon rien a joindre — le canal d'etat ne transmet rien.
3. **Cinq composants de ti=37 sont lus-jetes**, dont `equipment-control-signal-component`
   (i22, `R(4)` — un enumere de commande) et
   `equipment-tracked-object-handles-stack-component` (i28). Ils ne portent rien dans les
   DELTAS ; ils restent a instruire dans les IMAGES-CLES.
4. **Le reconnaisseur de records ti=37 a un plancher de faux positifs mesurable**, revele par
   les composants i31..i63 qui n'existent pas : ~210 faux records par film et par index. Tout
   comptage ti=37 publie jusqu'ici (y compris les « 856 lectures de charges » et les
   « 45 baisses » de R8) doit etre relu avec ce plancher.
5. **i54 est desormais ferme sur les deux oracles**, pas seulement sur la vitesse.

### 8.3 Reponse a la question de l'utilisateur

L'utilisateur VOIT le repulseur rejoue dans le visionneur Theater. Ce lot etablit que le
GESTE n'est dans aucun des canaux decodes, et que l'EFFET n'est pas mesurable dans les
positions repliquees. **Il ne permet pas de dire ce que le visionneur emploie**, et c'est
ecrit tel quel plutot que comble par une hypothese.

La piste la plus serieuse pour le lot suivant est le type d'evenement **14
`PlayEffectOnObject`** : il est PRESENT dans les films, sa grammaire n'est PAS portee (il
bloque 48 listes sur 12 films, R7 par. 3), et c'est exactement la forme qu'aurait « joue cet
effet sur cet objet » — le souffle du repulseur. La seconde est le canal des IMAGES-CLES pour
ti=37, ou les composants d'etat sont peut-etre transmis alors qu'ils ne le sont pas dans les
deltas.

---

## 9. Registre des reports — ce qui reste, et ce que ca coute

| # | Ce qu'il faut faire | Pourquoi | Cout |
|---|---|---|---|
| 1 | **Porter la grammaire du type 14 `PlayEffectOnObject`** et croiser ses occurrences avec les porteurs de repulseur (rang i48, par vie, comme le par. 8.8 de R8) | c'est le seul canal PRESENT dans les films, non porte, dont la semantique correspond a un effet visuel rejouable | 1 lot |
| 2 | **Instruire les composants d'etat de ti=37 dans les IMAGES-CLES** (pas les deltas) : `activated` (i21), `control-signal` (i22, `R(4)`), `tracked-object-handles-stack` (i28) | ce lot etablit qu'ils ne sont PAS dans les deltas ; les images-cles n'ont pas ete regardees | 1 lot |
| 3 | **Etalonner le reconnaisseur de records ti=37** : publier son plancher de faux positifs (mesure ici : ~210 par index inexistant et par film) et relire tous les comptes ti=37 deja publies | plusieurs resultats de R8 reposent sur des comptes qui incluent ce plancher | 1 lot |
| 4 | **Elucider les 45 « baisses de charge »** : leurs valeurs (247->244, 251->96) excluent des charges d'equipement et leur cadence (4 paliers en 130 ms) evoque un compte a rebours de tics — bruit de reconnaissance, ou champ mal nomme ? | consigne au par. 3.3, non tranche | 1/2 lot |
| 5 | **Angle mort i57 tag 3** : tout record annoncant i57 avec le tag 3 est abandonne (la branche `a == 0` n'existe pas dans les films). Ghidra sur `FUN_142f262d4` pour porter la branche `a == 1` (elle lit `R(6)` puis une porte sur `dst[2]`, octet d'etat runtime) | 2 a 3 % des records i57 perdus, et tout ce qui suit i57 dans ces records | 1 lot Ghidra |
| 6 | **Detecteur de records a masque dense** (report 5 de R8, non traite) : `matchBipedHeader` ignore les masques > 7 composants | conditionne tout rappel futur | non chiffre |
| 7 | **Verite terrain Theater** (report 2 de R8, non traite) : relever au visionneur les usages de repulseur d'un joueur sur UN film, avec leurs instants | c'est le seul moyen de donner un ORACLE au repulseur ; sans lui, chaque nouveau canal se juge a l'aveugle | 1 releve utilisateur |
| 8 | Porter la nuance du par. 6.1 dans le rapport R8 (sa piste A tient sur un argument, pas deux) | un rapport qui garde un argument perime se relit mal | 10 min |

**Le report 7 est le plus rentable.** Les huit canaux instruits par R8 et R9 ont tous ete
juges sans jamais disposer d'UN SEUL instant d'usage certain du repulseur. Le grappin en a
1 101 et le propulseur a le sien depuis R8 — c'est pour cela qu'on les a trouves.

---

## 10. Instruments et commandes rejouables

Tous dans `apps/go-api/internal/analysis/filmdec/`, `*_research_test.go`, gardes par variables
d'environnement, sautes par defaut (CI comprise), `CGO_ENABLED=0`, `LockProcessDecode` tenu
pendant chaque decodage, `WorldObjectPrecision` pose depuis le layout du film et restaure en
sortie (le piege consigne par R8), aucune ecriture, aucune DuckDB ouverte, aucun code de
production touche. `gofmt -l` vide et `go vet` vert sur le paquet.

| Fichier | Role |
|---|---|
| `r9_i57_tag3_research_test.go` | porte (a) — le tag 3 d'i57 scinde par le bit `a`, x rang i48 x oracle, temoins positifs i59 tag 3 et i57 tag 1 |
| `r9_ti37_identite_research_test.go` | porte (b) — identite ti=37 par les handles i26 ET par les poses, baisses de charge, transitions d'activation |
| `r9_poussee_research_test.go` | la face VICTIME : poussees subies par les voisins, par rang, avec exposition et symetrie inverse |
| `r9_i54_identite_research_test.go` | i54 rejuge par l'IDENTITE (l'oracle que R8 n'avait pas applique) |
| `r9_i22_signal_research_test.go` | le signal de commande i22 + LE RECENSEMENT DU MASQUE ti=37 avec son etalon de bruit |
| `r9_masque_research_test.go` | porte (c) — LE RECENSEMENT DU MASQUE BIPEDE par rang porte, sans ancre, les 64 composants |

Chemins : `<F>` = `<repo>/data/cache/film_chunks`,
`<B>` = `<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json`
(ici `<repo>` = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`).

```
# depuis apps/go-api
CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=00ba2e1c,06dfe6d9,084a804d,11de8353 \
  go test ./internal/analysis/filmdec/ -run '^TestR9I57Tag3$' -count=1 -timeout 60m -v

CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=00ba2e1c \
  go test ./internal/analysis/filmdec/ \
  -run '^TestR9(Ti37Identite|I22Signal)$' -count=1 -timeout 120m -v

CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=00ba2e1c,06dfe6d9,084a804d \
  go test ./internal/analysis/filmdec/ -run '^TestR9Poussee$' -count=1 -timeout 120m -v

CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=00ba2e1c,06dfe6d9 \
  go test ./internal/analysis/filmdec/ -run '^TestR9I54Identite$' -count=1 -timeout 120m -v

CGO_ENABLED=0 R8_FILMS=<F> R8_BOUNDS=<B> R8_IDS=06dfe6d9 \
  go test ./internal/analysis/filmdec/ -run '^TestR9Masque$' -count=1 -timeout 60m -v

# recensement des types de tete (instrument R7, rejoue tel quel)
CGO_ENABLED=0 R7_ROOT=<F> R7_IDS=00ba2e1c \
  go test ./internal/analysis/filmdec/ -run '^TestR7Recensement$' -count=1 -timeout 30m -v
```

Cout mesure : `I57Tag3` 55 s/film, `Ti37Identite` 85 s/film, `I22Signal` 35 s/film,
`Poussee` 68 s/film, `I54Identite` 74 s/film, `Masque` 117 s/film.

Instance Ghidra : commande de l'annexe C.1 de R6, telle quelle (JDK 25 + contournement
`-Djdk.net.unixdomain.tmpdir=Q:\nexistepas`), puis
`POST /load_program_from_project {"path":"/HaloInfinite.exe"}`. Requetes utilisees :
`GET /search_strings?search_term=...&limit=...`, `/get_xrefs_to?address=...`,
`/read_memory?address=...&size=16`, `/decompile_function?address=...`.
**Serveur arrete en fin de lot ; aucun verrou residuel ; projet non modifie.**

---

## 11. Ce que ce lot NE dit pas

- Il ne dit PAS ce que le visionneur Theater emploie pour rejouer le repulseur (par. 7.4).
  C'est la question qui reste ouverte, et elle est ecrite comme telle.
- Il ne refute PAS l'hypothese « le film porte l'EFFET sans le GESTE » : la mesure du par. 6
  ne la confirme pas, et sa puissance est trop faible pour la refuter.
- Il n'etablit AUCUN rappel, ni pour le propulseur ni pour le repulseur. Le report 2 de R8
  (verite terrain Theater) reste entier et devient le report 7 d'ici.
- Il ne touche AUCUN code de production : les six fichiers du lot sont des
  `*_research_test.go` gardes par environnement. Aucun commit.
- Le recensement du masque bipede (par. 7.5) a vu son temoin positif PRE-INSCRIT echouer
  (i28 x rang camouflage) ; son negatif repose sur deux temoins POSTERIEURS a la mesure
  (surbouclier, grappin) et vaut donc moins qu'un negatif pre-inscrit. C'est ecrit sur place.
- Il ne rejoue ni R7 (canal des evenements, CLOS pour les types cibles) ni les acquis de R8
  sur le propulseur ; il reproduit le film de reference de R8 a l'identique en controle
  (par. 0).

