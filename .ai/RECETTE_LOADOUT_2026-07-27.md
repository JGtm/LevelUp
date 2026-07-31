# RECETTE DU LOADOUT — grenades, capacités, armes, munitions

> Établie le 2026-07-27. Remplace les hypothèses des documents antérieurs sur ces quatre points.
> Chaque table porte la mesure qui la soutient et le test qui aurait pu la casser.
>
> **Complétée le même jour par une lecture du binaire** (§7 et suivants) qui confirme la table des
> grenades par une voie totalement indépendante, ferme la question de la jauge d'énergie, et
> corrige trois erreurs de ce document. Les corrections sont marquées dans le texte.

## CORRECTIONS APPORTÉES PAR LA LECTURE DU BINAIRE — à lire avant le reste

1. **Les noms de composants étaient disponibles depuis toujours.** Ce document et les chantiers
   qui l'ont produit reposaient sur la prémisse « les libellés lisibles n'existent pas en build de
   release, l'identité viendra d'un ordre ». **C'est faux deux fois** : les noms sont dans
   l'exécutable (`vtable + 0x08`, thunk `LEA RAX,[chaîne] ; RET`) **et dans le film lui-même**
   (registre ECS du `chunk_00`), ce dernier étant déjà exploité par le dépôt depuis le 2026-07-01
   via `internal/analysis/filmdec/registry.go`. Une partie du travail de ces deux jours a
   re-dérivé une table qui existait déjà.
2. **`i57` est à `0x12E4`, pas `0x12E7`.** Défaut factuel publié dans une version antérieure.
3. **Vie et bouclier ne sont pas « non tranchés »** : `i4` = `object-body-vitality`, `i5` =
   `object-shield-vitality`, nommés dans le registre du film et déjà décodés dans
   `internal/analysis/filmdec/vitality.go`. Une passe antérieure avait prononcé « RÉFUTÉ » sur la
   bonne réponse, en écartant un champ sur une intuition sémantique alors que son nom était à un
   fichier de distance.

## 0. LA CAUSE RACINE — le POC mélange DEUX FILMS

C'est la découverte qui débloque tout, et elle explique l'intégralité des contradictions
accumulées les deux jours précédents.

Le document de rejeu `replay_demo.html` déclare `match = "000d5950"`, `date = "8 mars 2026"`.
Ses blocs `tracks`, `loadouts` et `roster` viennent bien de ce film. **Mais son bloc `inv` vient
d'un autre film** : `9e8fb31b` (Cliffhanger, 24 juillet), celui sur lequel la capture Cheat Engine
a été faite.

Mesures qui l'établissent, par deux chaînes indépendantes :

| test | résultat |
|---|---|
| paires d'armes des `loadouts` retrouvées dans les images-clés | **150/150** dans `000d5950` · 12/150 dans `9e8fb31b` |
| valeurs du bloc `inv` retrouvées dans la capture localisée | 17/17 pour i22, 54/54 pour i42, sur `9e8fb31b` |
| arme en main du relevé Theater confrontée à l'image-clé | **8/8** sur `000d5950` · **1/8** sur `9e8fb31b` |
| rosters comparés par xuid | **1 joueur commun sur 8** |
| structure des fichiers | `000d5950` : 496 s, 27 morceaux / 26 images-clés · `9e8fb31b` : 414 s, 22 / 22. Le POC déclare 4985 images, ce qui exige au moins 25 images-clés : impossible avec 22. |

**Conséquence** : la vérité terrain relevée en Theater décrit `000d5950`. La capture Cheat Engine
décrit `9e8fb31b`. Toute confrontation entre les deux comparait **deux matchs différents**. Il n'y
avait aucun défaut de décodage à chercher — ni horloge fausse, ni dotation de pré-match, ni
grammaire erronée. Ces trois explications ont été avancées puis abandonnées ; aucune n'était la
bonne.

**Règle qui en découle** : tout document de rejeu porte désormais l'identifiant du film de CHAQUE
bloc, et aucune confrontation ne se fait sans vérifier que les deux côtés portent le même.

## 1. LES GRENADES — bijection compteur → type

`i22` = `R(3)` valant toujours 4, puis **4 × R(8)** à valeurs 0/1/2.
`i47` = `[6 bits masque][3 bits sélection]`, le masque étant **exactement** le bitmap des compteurs
non nuls d'`i22` et la sélection l'index en base 1.

| compteur | type | appui |
|---|---|---|
| 0 | **Fragmentation** | 10 lancers, 6 entités |
| 1 | **Plasma** | 7 lancers, 4 entités |
| 2 | **Dynamo (choc)** | 6 lancers, 4 entités |
| 3 | **Spike** | 12 lancers, 8 entités |

**Comment elle est obtenue, sans aucune vérité terrain** : on apparie chaque lancer observé dans le
film à la transition qui décrémente d'exactement 1 un compteur d'`i22`, à la même image. Deux
chaînes de décodage indépendantes — balayage de motif pour le type du projectile, chaîne de
composants localisée par la capture pour l'index. **35 lancers sur 35** tombent sur une transition
unique. Zéro index portant deux types, zéro type porté par deux index : une seule des 24
bijections possibles survit.

**Les tests qui auraient pu la casser, tous passés** : index uniforme par entité, 0 sur 200 000
tirages ; index permutés à effectifs constants, 0 sur 200 000 ; décalage temporel de −120 à +120
images, 240 contrôles, moyenne 0,61/35 contre 35/35 réel, maximum 3/35. Sélectivité du marqueur
mesurée sur le flux réel : 1191 occurrences brutes, 35 passent le filtre, et ce sont exactement
les 35 qui coïncident.

Cette table est **interne à `9e8fb31b`** et ne dépend d'aucune donnée de l'autre film : elle
survit intacte à la découverte des deux films.

Accord `i22` ↔ `i47` : **194 sur 194**, zéro désaccord. Aucun masque supérieur à 15 sur 204
lectures, ce qui borne la palette à 4 types et réfute l'hypothèse d'un cinquième.

## 2. LES CAPACITÉS — table des index

> **CETTE TABLE EST À MOITIÉ FAUSSE, corrigée le 2026-07-28.** Le groupe de tags `sofd`, lu dans
> le binaire, donne la vraie palette. Il **confirme** les index 4 et 5, il **contredit** les
> index 3 et 6. Voir §13, qui fait foi.

Établie sur `000d5950`, le film de la vérité terrain. **Elle ne vient PAS d'`i48`** mais d'un champ
de 3 bits du record de biped des images-clés.

| index | capacité annoncée | appui | verdict du `sofd` |
|---|---|---|---|
| 3 | mur portatif | slot 517 — seul mur du relevé. 24 records | **CONTREDIT** — le mur est au rang 2 |
| 4 | **grappin** | slots 512, 513, 516 — **les trois grappins, valeur identique**. 44 records | **CONFIRMÉ** |
| 5 | **propulseur (dash)** | slots 514, 518, 519 — **les trois propulseurs**. 44 records | **CONFIRMÉ** |
| 6 | capteur de menace | slot 515 — seul capteur du relevé. 20 records | **CONTREDIT** — le rang 6 est le répulseur |

**Le motif de l'erreur est net, et il vaut leçon** : les deux index confirmés sont exactement ceux
adossés à un **triplet** de joueurs ; les deux contredits sont exactement ceux adossés à une
**observation unique**. Un contrôle de groupe vaut ; un témoignage isolé ne vaut pas.

**Le contrôle interne passe** : les deux triplets sortent groupés, identiques à l'intérieur,
mutuellement distincts, sans qu'aucune information de la table n'ait été injectée dans la lecture.

Appuis complémentaires : second canal indépendant — le record d'objet d'équipement porte le
même index, multiensembles identiques sur 24 images-clés sur 26. Persistance : 32 vies à index
constant sur au moins deux images-clés, contre 5 changeantes (ramassages au sol, normaux en
Fiesta) ; IKE ILYA garde l'index 4 sur six images-clés consécutives, soit 100 s d'une même vie.

### CETTE TABLE EST PARTIELLE — ne pas la généraliser

Le chantier a écrit « palette de **exactement** 4 valeurs sur 132 records ». C'est une
généralisation abusive : la mesure dit seulement « 4 valeurs **dans ce match** ». Sur une partie
d'arène en Fiesta, quatre équipements tirés au sort ne bornent pas la palette du jeu.

Le dossier `static/abilities-assets/halo_infinite/` contient **onze** capacités distinctes :

    ActiveCamouflage · DropWall · Grappleshot · Overshield · QuantumTranslocator
    RepairField · Repulsor · ShroudScreen · ThreatSeeker · ThreatSensor · Thruster

Les index mesurés sont 3, 4, 5, 6 — **consécutifs**, donc il existe très probablement des index
0 à 2 en dessous et 7 et au-delà au-dessus. L'ordre n'est pas alphabétique : alphabétiquement
`ThreatSensor` précède `Thruster`, or on mesure l'inverse (5 pour le propulseur, 6 pour le
capteur). C'est donc l'ordre d'une énumération interne.

**LA STABILITÉ DANS LE TEMPS EST PROBABLE — correction d'une alerte excessive.** J'avais écrit
qu'une énumération qui grandit après la sortie du jeu « n'a aucune raison d'être stable ». C'est
faux, et deux arguments convergent :

1. **Une énumération sérialisée grandit par ajout, pas par renumérotation.** Ajouter une entrée
   n'en déplace aucune autre, exactement comme ajouter une référence à un catalogue.
2. **Le jeu lui-même en dépend.** Theater doit rejouer un film enregistré avant une mise à jour.
   Si les index se renumérotaient, tout rejeu antérieur afficherait la mauvaise capacité. La
   contrainte de compatibilité du format de rejeu impose donc l'append-only.

Le `ThreatSeeker`, variante de mode Infection ajoutée après la sortie, occupe donc selon toute
vraisemblance un index **au-delà** de ceux du socle d'origine, sans avoir déplacé le grappin ni le
propulseur.

**Ce qui reste vrai, et qui est la seule vraie limite** : la table est **incomplète**, pas
instable. Quatre index sur onze capacités connues. Le manque n'est pas un risque de dérive, c'est
un trou de couverture — un index jamais observé apparaîtra un jour et on ne saura pas le nommer.

**Contrôle à garder, peu coûteux** : si un même index désignait deux capacités différentes sur
deux films d'époques différentes, l'hypothèse d'append-only tomberait. Vérifier au premier
recoupement multi-films disponible.

Réserve secondaire : les noms de fichiers d'assets ne sont pas une source fiable. Le dossier
voisin des armes est connu pour être mal nommé — `Cremator.png` y est en réalité le Cindershot.
Onze fichiers ne garantissent donc pas onze entrées d'énumération.

**Où lire** : paquet d'image-clé (type 2), record d'archétype 35 — ancre 28 bits `0x8CAC57A`, puis
dans les 60 bits suivants le motif 20 bits `00000000000000010010`, puis 3 bits = l'index. L'ancre
est présente exactement 2 fois par porteur (un biped + un objet), jamais plus.

**NON RÉSOLU** : l'index d'`i48` lui-même, et le compteur d'utilisations. `i48` n'est mesurable que
sur `9e8fb31b`, qui n'a pas de vérité terrain.

## 3. LE SÉLECTEUR D'ARME — `i42`

Grammaire réelle : `R(3)` puis, deux fois, `R(1)` porte active-bas suivi d'un `R(2)` optionnel.
Largeurs mesurées : 5 (101 fois) et 7 (134 fois), rien d'autre.

| valeur | signification |
|---|---|
| sel = 0 | emplacement 0 dégainé — arme `i43`, munitions `i30`/`i31` |
| sel = 1 | emplacement 1 dégainé — arme `i44`, munitions `i33`/`i34` |
| sel = 2 | aucune arme dégainée (état de spawn) |

**Le sens est établi par un oracle interne au film, sans vérité terrain** : l'emplacement dont les
munitions bougent est celui qui est dégainé.

| régime | émissions sur l'emplacement 0 | sur l'emplacement 1 |
|---|---|---|
| sel = 0 | **1921 (94,7 %)** | 107 |
| sel = 1 | 42 | **878 (95,4 %)** |
| référence tous régimes | 1963 (66,6 %) | 985 (33,4 %) |

Restreint aux huit slots initiaux : 98,9 % et 98,2 %. En régime sel = 2, **zéro** émission de
munition.

Ordre des armes dans une image-clé : l'ordre des bits **est** l'ordre des emplacements — premier
identifiant = emplacement 0, second (à environ +195 bits) = emplacement 1. Vérifié 8/8.

## 4. LES MUNITIONS

| composant | rôle | grammaire |
|---|---|---|
| `i30` / `i33` | chargeur | champ de 10 bits `[0][8 bits valeur][1]` ; valeur = 9 bits au curseur |
| `i31` / `i34` | réserve | 11 bits au curseur, valeur directe |

Invariant du chargeur : bit 0 = 0 implique bit 9 = 1 sur **1450 lectures sur 1450**.
Appariement : `i43` gouverne `i30`/`i31`, `i44` gouverne `i33`/`i34` — **31/31** sur les cas
discriminants, **0/31** pour l'appariement inverse.

Sémantique établie **sans terrain** : sur les seules entités au-delà du slot 519, le chargeur
baisse d'exactement 1 dans **913 baisses sur 927** (98,5 %) ; la réserve baisse par paquets.

**Table arme → chargeur / réserve**, 16 armes :

    BR75 30/90 · Bulldog 12/12 · Cindershot 6/6 · Disruptor 7/14 · Heatwave 8/8
    M41 SPNKr 2/2 · MA40 AR 25/75 · MLRS-2 Hydra 6/6 · Mangler 8/16 · Mk51 Sidekick 14/42
    Needler 30/30 · S7 Sniper 10/10 · Sentinel Beam 80/80 · Shock Rifle 15/30
    Skewer 1/3 · VK78 Commando 40/120

Cohérence non construite, donc probante : la réserve est un **multiple entier** du chargeur pour
les 16 armes. Confrontée à la vérité terrain, la table reproduit **8/8** les couples relevés, y
compris le marteau du slot 513 qui n'a correctement **aucun** composant de munitions.

Hasard réfuté par mesure : 13 689 grammaires balayées, les 25 qui atteignent le score maximal
s'effondrent en une seule classe d'équivalence ; 0 sur 200 000 bijections aléatoires arme →
munitions reproduit le contrôle.

**RÉSERVE HONNÊTE** : aucune lecture de munition n'existe sur les huit slots initiaux entre les
images 35 et 296. La valeur à l'instant du relevé est donc **reconstruite** par la table indexée
par arme, pas lue directement. La table est solide, la lecture ponctuelle ne l'est pas encore.

## 4 bis. LE BIT DE TÊTE EST UN AIGUILLAGE — chargeur ou JAUGE D'ÉNERGIE

Le chantier avait laissé 1363 lectures sur 2813 « non décodées » et conclu qu'elles ne
concernaient aucune arme à chargeur, sur la foi de 2 lectures du Ravager. Remarque de
l'utilisateur, décisive : le Ravageur et le pistolet à plasma sont des **armes à charge**, dont la
jauge baisse à l'usage. Test refait en croisant chaque lecture avec l'arme de l'emplacement
correspondant, sur toutes les entités :

| arme | préfixe 0 | préfixe 1 | part préfixe 1 |
|---|---|---|---|
| **Ravager** | 1 | **597** | **99,8 %** |
| **Plasma Pistol** | 1 | **423** | **99,8 %** |
| **Pulse Carbine** | 0 | **112** | **100 %** |
| **Stalker Rifle** | 0 | **38** | **100 %** |
| Mk51 Sidekick | 394 | 24 | 5,7 % |
| Needler | 159 | 11 | 6,5 % |
| Sentinel Beam | 143 | 5 | 3,4 % |
| Shock Rifle | 96 | 10 | 9,4 % |
| BR75 | 97 | 8 | 7,6 % |
| Disruptor | 89 | 13 | 12,7 % |
| S7 Sniper | 90 | 15 | 14,3 % |
| toutes les autres | majoritaires | résiduel | < 40 % |

**Quatre armes se détachent à 99,8-100 %, et ce sont exactement les armes à charge.** Le bit de
tête d'`i30`/`i33` n'est donc pas un bit de cadrage mais un **aiguillage sémantique** : un même
champ porte le chargeur (préfixe 0) ou la jauge d'énergie (préfixe 1) selon l'arme tenue.

Cela **corrige** l'affirmation du chantier : ces quatre armes émettent bien des composants de
munitions, simplement dans l'autre branche. L'agent n'en avait vu que 2 là où il y en a 597.

Deux réserves :
- Le résidu de 5 à 40 % de préfixe 1 sur les armes à chargeur vient très probablement d'un
  décalage du suivi d'arme courante lors d'un échange d'arme. **Explication plausible, non
  mesurée** — à confirmer en réappariant sur l'arme réellement dégainée via `i42`.
- **CORRIGÉ le même jour** : ce paragraphe affirmait que le marteau et l'épée n'apparaissaient dans
  aucune des deux branches. **C'est faux** — mesure refaite sur `000d5950`, ils totalisent 22 des
  37 jauges. Voir §10, qui remplace cette affirmation.

La valeur exacte portée par la branche préfixe 1 (pourcentage, fraction, unités de charge) n'est
pas encore décodée — seul l'aiguillage l'est.

## 5. UN GAIN DE MÉTHODE — la localisation devient positionnelle

Le crochet journalise 16 octets lus à `[rdi+0x40]`, pointeur d'octet du lecteur, qui avance par
mots de 64 bits (cohérence mesurée sur 658 307 paires consécutives : 96,9 % à 8 octets).

    offset de la signature = paquet.Start + 8 * floor(curseur / 64) + 8

Balayage du décalage de −32 à +64 par pas de 8 : `+8` donne 623 lectures, `+0` en donne 8, tous
les autres zéro ou un.

- **5 677 localisations sur 5 770** pour les 91 entités d'archétype 35, soit 98,4 %.
- Multiplicité de 0 ou 1 paquet par lecture, **jamais plus** : zéro collision.
- **Faux positifs mesurés sur le flux réel : 0 sur 650 641 positions alignées testées.**

Le filtre d'entropie devient inutile — il rejetait 8 lectures valides sur 635.

## 6. CE QUI RESTE OUVERT

| question | prochaine mesure |
|---|---|
| **La palette complète des capacités** | onze capacités connues, quatre index mesurés. Chercher l'énumération dans le binaire (chantier Ghidra), ou multiplier les relevés Theater sur des matchs aux équipements variés |
| La stabilité de l'énumération dans le temps | **probablement acquise** (append-only imposé par la compatibilité de Theater) — contrôle à garder : un même index désignant deux capacités sur deux films d'époques différentes la réfuterait |
| **La valeur de la jauge d'énergie** | l'aiguillage préfixe 0/1 est établi ; la grammaire de la branche 1 (pourcentage ? fraction ? unités ?) ne l'est pas |
| **La jauge du marteau et de l'épée** | elles n'émettent NI préfixe 0 NI préfixe 1 sur `i30`/`i33`. Chercher le composant porteur ailleurs |
| Le résidu de préfixe 1 sur les armes à chargeur | réapparier sur l'arme réellement dégainée via `i42` au lieu de la dernière vue |
| L'index d'`i48` et le compteur d'utilisations | exigent une vérité terrain sur `9e8fb31b`, ou une capture sur `000d5950` |
| Le cas discriminant du slot 513 pour `i42` | exige le flux `i42` de `000d5950` à son coup d'envoi |
| Prédiction falsifiable, à vérifier en Theater sur `9e8fb31b` | à 00:25,7 les huit joueurs tiennent la PREMIÈRE arme de leur paire ; `i42` émet sel = 0 sur les huit simultanément |

Loadouts réels de `9e8fb31b` à l'image 199, slots 512 à 519, si quelqu'un veut vérifier :
Gravity Hammer + Mk51 Sidekick · Skewer + Disruptor · M41 SPNKr + S7 Sniper · Plasma Pistol +
Disruptor · S7 Sniper + Energy Sword · S7 Sniper + Energy Sword · MA40 AR + Gravity Hammer ·
Shock Rifle + Plasma Pistol.

## 7. LA CARTE MÉMOIRE, LUE DANS LE BINAIRE

Chaque décalage ci-dessous a été relu instruction par instruction, pas reconstitué.

    tampon = *(void**)(enregistrement + 0x10)   -- tous les decalages sont relatifs a ce tampon

### Les emplacements d'arme — base 0x7F0, pas 0x90, quatre entrées

`0x1407F06BC` : `MOVSXD RAX,[RCX+8]` puis `LEA RCX,[RAX+RAX*8]` ; `SHL RCX,4` ;
`LEA RSI,[RBP+0x7f0]` ; `ADD RSI,RCX`. Fermeture arithmétique indépendante :
`0x7F0 + 4 x 0x90 = 0xA30`, qui est exactement le décalage d'`i47` lu ailleurs
(`0x140C6A628` : `ADD RCX,0xa30`). Trois mesures concordent sans ajustement.

| dans l'emplacement | contenu | preuve |
|---|---|---|
| +0x00 (`0x7F0`) | identité de l'arme | `i43`..`i46`, `weapon-state-type-info` |
| +0x7E (`0x86E`) | u16 chargeur, entier | `0x140EA1018` : `MOV word[RDI+0x86e],R9W` |
| +0x80 (`0x870`) | u16 réserve, `R(11)` inconditionnel | `0x140FE4E88` : `ADD [RDX+0x2c],0xB` ; `SHR R9,0x35` |
| +0x82 (`0x872`) | 2 bits de drapeaux | `i32`, `0x142F04C6C` |
| +0x84 (`0x874`) | **f32 jauge de charge**, 12 bits déquantifiés dans [0,1] | `i30`, branche fraction |
| +0x88 (`0x878`) | f32 surchauffe, 7 bits dans [0,1] | `i32`, `MOVSS [RDI+RSI*8+0x878],XMM0` |

### La grammaire d'i30/i33 est une UNION à deux branches

    b0 = R(1) ; si b0 == 0 : u16[+0x7E] = R(8)                        sinon 0
    b1 = R(1) ; si b1 == 0 : f32[+0x84] = dequant(R(12), 0.0f, 1.0f)  sinon 0.0f
    largeurs possibles : 2 (vide) / 10 (entier) / 14 (fraction)

**Cela ferme deux points ouverts de ce document.** L'invariant mesuré « bit 0 = 0 implique
bit 9 = 1, sur 1450 lectures sur 1450 » n'était pas une curiosité : c'est la structure elle-même.
Et la **jauge d'énergie** de la branche préfixe 1 — Ravageur, pistolet à plasma, carabine à
impulsions, fusil traqueur, à 99,8-100 % — n'est ni un pourcentage ni un compte d'unités : c'est
une **fraction de charge pleine dans [0,1] codée sur 12 bits**, soit 4096 niveaux, déquantifiée
entre `0.0f` et `1.0f`.

## 8. LES GRENADES — CONFIRMÉES PAR LE BINAIRE, question close

`0x140F0DE00` : `MOV RCX,[R8+0x10]` ; `ADD RCX,0x758`. `i22` écrit N octets à `0x758`, six
emplacements existant (`0x758` à `0x75D`). **L'ordre est `typeId - 1`.**

Table `grenade_types` (base `0x1443E2AB0`, entrées de 16 octets), décompilée :

    id 1 grenade_frag « fragmentation grenade »   id 2 grenade_plasma
    id 3 grenade_lightning (le Dynamo)            id 4 grenade_spike
    id 5 grenade_sapper                           id 6 grenade_stasis

La boucle de dotation ne parcourt que 1 à 4, ce qui explique le `N = 4` constant observé.

| rang dans `i22` | grenade |
|---|---|
| 0 | **Fragmentation** |
| 1 | **Plasma** |
| 2 | **Dynamo** (nom interne `lightning`) |
| 3 | **Spike** |

**Identique à la table du §1, obtenue empiriquement par une voie sans aucun rapport** —
appariement de 35 lancers sur 35 aux décréments unitaires, 0 sur 200 000 permutations. Deux
chaînes totalement indépendantes donnent 4 rangs sur 4. **La question des grenades est close.**

**PIÈGE CONFIRMÉ, à ne pas utiliser** : l'énumération d'équipement (`FUN_140157770`) donne
FragGrenade = 9, **DynamoGrenade = 10, PlasmaGrenade = 11**, SpikeGrenade = 12. Elle **intervertit
Dynamo et Plasma** par rapport à `grenade_types`. Ne jamais s'en servir pour `i22`.

## 9. LES CAPACITÉS — le mécanisme est lu, le nommage n'est PAS dans l'exécutable

`0x1410F8FCC` : `ADD RCX,0xa34` puis saut vers `FUN_1406D0FF0`, qui écrit **deux octets par deux
lecteurs différents** :

| octet | rôle | grammaire |
|---|---|---|
| `0xA34` | **compteur de rotation**, PAS une identité | `R(3)` puis valeur - 1, donc -1 à 6 |
| `0xA35` | **l'identité** — un rang de palette | `R(1)` porte ; si 0, `R(6)` |

Que `0xA34` soit un compteur est établi, pas supposé : `FUN_140A225CC` fait
`entité[0x589] = (entité[0x589] + 1) % 7` par l'idiome de division magique par 7, uniquement quand
l'état change. L'intervalle -1 à 6 colle : une valeur d'absence plus sept états.

Le rang de `0xA35` est résolu à l'exécution par `FUN_1407E7648`, qui parcourt le bloc du groupe de
tags `sofd` (données à `tag+0x28`, nombre à `tag+0x38`, pas `0x20`), compare `entrée+0x18` au
handle de définition et rend le rang, -1 si absent. **6 bits, donc 64 entrées au maximum.**

**Conséquence, et elle est définitive** : la palette vit dans les fichiers de tags du jeu, pas dans
l'exécutable. Il est **inutile de continuer à chercher l'énumération des capacités dans Ghidra** —
cette ligne de l'inventaire ouvert du §6 est barrée. Ce n'est pas un abandon, c'est une mesure.

### Le compteur d'utilisations est trouvé

`0x140FC1410` : masque `R(3)`, puis pour chaque bit armé **7 bits**, écrits en `0x12EA + s`
(s = 0 à 2) ; emplacement non armé donne `0x7F`. Le consommateur `FUN_140F8F300` montre que `0x7F`
vaut **1.0f, plein par défaut**, et que deux encodages coexistent selon la capacité : continu
`v / 127.0f`, ou discret `(v >> 4) & 0xF` charges entières plus `(v & 0xF)` de recharge
fractionnaire. **Le « 5 utilisations » de la vérité terrain est le quartet de poids fort.**

Capacité active : `i57` à **`0x12E4`**, `R(2)` — corrige le `0x12E7` publié antérieurement.

### RÉSERVE SUR LA TABLE DU §2 — une seule mesure la tranche

Le §2 lit l'index de capacité sur **3 bits** dans le record de biped des images-clés, et obtient
3, 4, 5, 6 avec les deux triplets groupés. Le binaire dit que l'identité est un rang sur **6 bits
précédé d'une porte**. Les deux sont compatibles **si et seulement si** les rangs valent moins
de 8 — ce qui est le cas ici, mais n'a pas été vérifié.

**Mesure décisive, à faire en priorité** : à la position de bit exacte où le §2 lit ses 3 bits,
élargir la fenêtre à 6 bits (trois bits plus tôt) et relire les mêmes records.
**Prédiction falsifiable** : les valeurs restent 3, 4, 5, 6 ; les trois bits ajoutés sont nuls sur
tous les records ; le bit immédiatement précédent vaut 0 partout (la porte « présent »).

- Si elle passe : la table du §2 **est** la table des rangs de la palette `sofd`, `i48` devient
  nommable sur le film de la vérité terrain, et le problème des deux films est contourné sans
  nouvelle capture.
- Si elle échoue : le champ des images-clés n'est pas `i48`, et les deux canaux doivent être
  traités séparément.

## 10. LA JAUGE DU MARTEAU ET DE L'ÉPÉE — RÉSOLU : elle est dans la branche jauge

**Correction d'une affirmation fausse de ce document.** Le §4 bis écrivait : « le marteau et l'épée
n'apparaissent dans aucune des deux branches, leur charge est portée par un autre composant ».
**C'est faux**, et l'erreur vient d'avoir mesuré sur `9e8fb31b` — un film où ces armes sont rares
et où le suivi d'arme courante décalait.

Mesure refaite sur `000d5950`, en croisant chaque lecture avec l'arme de l'emplacement, 37 jauges :

| arme | jauges lues |
|---|---|
| **Gravity Hammer** | **17** |
| Stalker Rifle | 9 |
| **Energy Sword** | **5** |
| Ravager | 3 |
| Pulse Carbine | 1 |
| Sentinel Beam | 1 |
| M41 SPNKr | 1 |

**Le marteau et l'épée totalisent 22 des 37 jauges.** Ils utilisent bien la branche préfixe 1,
comme les armes à batterie. La lecture correcte de l'aiguillage est donc :

> Un même champ porte **le chargeur** (branche préfixe 0, armes à munitions comptées) ou **une
> jauge de charge dans [0,1]** (branche préfixe 1, armes à énergie **et armes de mêlée**).

C'est cohérent avec le comportement en jeu, signalé par l'utilisateur : la réserve d'énergie du
marteau et de l'épée décroît à chaque coup porté. Le « charge 100 % » relevé en Theater pour le
marteau du slot 513 est donc une **valeur de jauge**, pas une absence de donnée.

**Réserve** : la lecture unique sur `M41 SPNKr`, un lance-roquettes à munitions comptées, est
très probablement un faux positif — soit un décalage du suivi d'arme, soit une lecture parasite.
Une jauge sur 1 lecture ne fait pas une catégorie ; à surveiller, pas à interpréter.

**Reste à mesurer** : que la valeur de jauge **décroisse effectivement aux coups portés**. La
présence du champ est établie, sa dynamique ne l'est pas.

**Le candidat `weapon-state-overheated` (`i32`/`i35`) reste ouvert pour autre chose** : f32
déquantifié sur 7 bits à emplacement+`0x88`, plus 2 bits de drapeaux à +`0x82`. Ce n'est pas la
jauge de charge, puisqu'elle est ailleurs — c'est vraisemblablement la surchauffe au sens propre
(Sentinel Beam, Plasma Pistol en tir chargé).

## 11. LE CONTRÔLE DES GROUPES N'A JAMAIS TOURNÉ

À consigner franchement : le contrôle interne annoncé comme passé dans le premier chantier
(« les trois grappins partagent la même valeur ») **n'a été exécuté dans aucune des quatre
passes**, et pour les capacités il **ne pouvait pas** l'être — la vérité terrain décrit
`000d5950` tandis que toutes les mesures d'`i48` et `i56` viennent de la capture sur `9e8fb31b`.

Les prédictions chiffrées qui en découlaient auraient échoué **quelle que soit la justesse de la
grammaire**. Un successeur qui les suivrait jetterait une grammaire correcte.

**Le premier contrôle réellement exécutable** : lire `i56` dans les records d'image-clé de
`000d5950` et comparer le quartet de poids fort de chaque octet d'emplacement armé à la vérité
terrain — **5, 5, 5, 4, 5, 1, 5, 5** pour les slots 512 à 519. Le 4 du capteur de menace et le 1
du mur portatif sont les deux points discriminants : deux valeurs aberrantes tombant au bon
endroit rendent un accord fortuit très improbable.

## 12. LA MÉTHODE DE NOMMAGE DES COMPOSANTS

Le nom d'un composant se lit à `vtable + 0x08`, thunk `LEA RAX,[chaîne] ; RET`. Vérifié par
lecture mémoire : `0x143C98830` = `weapon-state-ammo`, `0x143C98EC8` =
`biped-desired-ability-set-component`, `0x143C98EF0` = `weapon-state-type-info`,
`0x143C961C0` = `unit-grenade-counts-component`.

**Piège** : `vtable + 0x58` est aussi un thunk de chaîne, mais rend le nom d'un **autre** composant
de la même famille.

**Réserve d'honnêteté** : le dépôt possède déjà la table complète des 64 noms, extraite en direct
le 2026-07-01 (`.ai/RECETTE_DECODAGE_FILM_CHUNKS.md` §6, et
`internal/analysis/filmdec/registry.go` qui analyse le registre ECS du `chunk_00` du film). La
méthode Ghidra ne fait que la redonner **hors ligne, sans jeu lancé**. Ce n'est pas une découverte
d'information, c'est une redondance utile.

## 13. LA PALETTE DES CAPACITES — le groupe de tags `sofd` fait foi

Etablie le 2026-07-28, en exploitant le croisement avec le worktree voisin : leur chaine
d'armement de vehicule (`vcdd -> sofd -> sofa -> uwfa -> weap`) traverse la meme structure ou se
resout l'identite de capacite. Noms obtenus par murmur3 des identifiants de chaine, recoupes avec
l'enumeration d'equipement de l'executable.

| rang | capacite |
|---|---|
| 0 | `mobility_sprint` — la course, categorie nulle, **pas une capacite** |
| 1 | **detecteur de menaces** (`ability_location_sensor`) |
| 2 | **mur de protection** (`ability_deployable_wall`) |
| 4 | **grappin** (`ability_grapple_hook`) |
| 5 | **propulseur** (`ability_evade`) |
| 6 | **repulseur** (`ability_knockback`) |
| 7 | `melee_default` — la melee, categorie nulle |
| 8 | **camouflage actif** |
| 9 | **surbouclier** |
| 11 | **translocateur quantique** |
| 12 | **traqueur de menaces** |
| 23 | **champ de reparation** — confirme par DEUX chaines : murmur3, et la banque sonore
       `sb_007_abl_repairfield` atteinte depuis ce rang a la distance la plus courte |

Les rangs 3, 10, 13 a 22 et 24 a 26 n'ont pas ete casses (identifiants de chaine non inverses).
Le rang 3 n'est de toute facon **pas une capacite** : categorie nulle, et son `eqip` ne reference
qu'un systeme de marquage.

### LA PALETTE N'EST PAS GLOBALE — correction d'une conclusion de la veille

Le 2026-07-27, ce document concluait que la stabilite de l'enumeration etait « probablement
acquise » par append-only. **C'est vrai a l'interieur d'une famille, faux entre familles.**

Mesure : sur les 46 equipements presents dans au moins deux des 12 `sofd` du `glpa`, **26 gardent
leur rang et 20 en changent**. Exemples verifies :

    ability_grapple_hook   rang 4 dans d91958af / 03137359 / 13c097ed, rang 8 dans 51e60c5a
    ability_evade          rang 5 dans la famille A, rang 6 dans 51e60c5a, rang 3 dans 758be0bc
    ability_knockback      rang 6 dans la famille A, rang 13 dans 51e60c5a
    powerup_overshield     rang 9 dans la famille A, rang 15 dans 51e60c5a

Le `sofd` employe est choisi **a l'execution** (composant a `unite+0x268`).

**Ce qui sauve l'exploitation** : les trois `sofd` de la « famille A » (d91958af 27 entrees,
03137359 19, 13c097ed 17) ont un prefixe **rigoureusement identique sur les rangs 0 a 9**, et
c'est la seule famille compatible avec un jeu de capacites de joueur.

**Consequence operationnelle** : une table de decodage doit etre **indexee par le `sofd` du
match**, pas gravee une fois pour toutes. Determiner quel `sofd` s'applique a un film donne est la
question ouverte qui reste.

### LES TROIS BRANCHES OUVERTES, non tranchees

Le controle contre les quatre index mesures echoue 2 sur 4 (voir §2). Trois explications restent
possibles, et aucune n'a pu etre departagee :

1. **Le champ de 3 bits n'est pas le rang de palette.** Le §9 de ce document le signalait deja :
   « le binaire dit 6 bits precedes d'une porte ; mesure decisive, a faire en priorite ». Cette
   mesure n'a toujours pas tourne. **C'est la premiere a faire** : relire les memes records sur
   6 bits plus le bit de porte, et verifier si le mur ressort au rang 2 et le capteur au rang 1.
2. **Le releve Theater a confondu** le repulseur et le detecteur de menaces sur ses deux
   observations uniques — deux boitiers tenus en main, visuellement proches.
3. **Le film n'emploie pas le `sofd` de la famille A.** Argument mesure contre : c'est le seul des
   douze ou le grappin soit au rang 4 et le propulseur au rang 5.
