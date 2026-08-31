# LE MODELE DE PAQUET EST PERCE : [configuration][liste d'evenements][trame de records]

Date : 2026-08-30, fin de soiree. Lot 1 du plan « percer la trame ». Cette note etablit LA
RECONCILIATION des lots D et E et ses consequences — dont la reouverture de la lunette.

## Le modele M, etabli par deux chaines sans etape commune

Un paquet delta de film s'ecrit :

```
[1 bit  drapeau de configuration]            (toujours 1 sur le corpus)
[liste d'evenements :
   ( 1  [R(7) type]  [3 references gardees]  [charge du type] )*  0 ]
[trame de records ECS, jusqu'a la fin du paquet]
```

La grammaire de la liste est EXACTEMENT celle prouvee au desassemblage par le lot E
(FUN_14076a1c4 + FUN_14080a9d4) : continuation, R(7) type < 123, trois references
[R(1) porte ; si 1 : R(w) index + R(2) generation] avec w par domaine (vtable+0x58), puis
la charge utile du type (vtable+0x68). LE POINT QUE TOUT LE MONDE AVAIT MANQUE : cette
liste vit APRES le bit de configuration et AVANT la trame de records — les deux lots
regardaient le meme flux avec un decalage d'un bit, et chacun voyait sa moitie.

**Chaine 1 (arithmetique, corpus entier)** : un evenement en tete de paquet donne
`octet0 = 0xC0 | (type >> 1)` et `bit8 = type & 1`. La table tombe juste sur TOUTES les
familles observees ; une liste vide donne `octet0 ∈ 0x80..0xBF` (bit 1 = 0) — 0xA0/0x80/
0x89 et leur cadrage k=2 prouve par l'oracle Rosette.

**Chaine 2 (decodage de bout en bout, 2 films)** : la famille 0xCA decodee ENTIEREMENT avec
des largeurs 100 % sourcees de l'exe (type 21 : refs domaines 4/8/7 -> R(9)/R(13)/R(13),
charge R(2)) — voir le verdict ci-dessous.

## La table famille -> evenement (annexe A du lot E appliquee)

| Octet | Volume corpus | Type(s) = bits 2..8 | Nom |
|---|---|---|---|
| `0xA0`, `0x80`, `0x89`, `0x88`… | ~35 M | — (bit 1 = 0) | liste vide : trame de records pure |
| `0xC0` | 983 883 | 0 / 1 | **damage_aftermath / damage_section_response** — LES DEGATS |
| `0xC2` | 458 938 | 4 / 5 | item_detonate_countdown / **projectile_detonate** |
| `0xC3` | 245 358 | 6 / 7 | projectile_impact_effect / projectile_object_impact_effect |
| `0xC7` | 1 023 286 | 14 / 15 | PlayEffectOnObject / Script |
| `0xCA` | 399 988 | 20 / **21** | incident / **unit_zoom — LA LUNETTE** (mesure : 100 % type 21 sur 2 films) |
| `0xD2` | 2 535 816 | **36** | **action_weapon_fire** — le record de tir/degat (bit8=0 constant mesure) |
| `0xD3` | 528 262 | 38 / 39 | **weapon_reload / biped_throw_initiate** (grenades) — le « gisement » du plan |
| `0xE5` | 195 824 | 74 / 75 | AISetMotorProgram / AIDialog |
| `0xE6` | 195 107 | 76 / 77 | Dialogue2D / DebugSendCameraPosition |
| `0xE9` | 922 724 | 82 / 83 | PlayerGameEventSmall / TeamGameEvent |
| `0xF3` | 31 228 | 102/103 | NetworkedActionRequest / EquipmentSpawnedObject |

(bit8 departage les deux types d'un octet ; a mesurer famille par famille — fait pour 0xCA
et 0xD2.)

## Le verdict 0xCA (unit_zoom) — bout en bout, deux films

| mesure | 000d5950 (12 chunks) | 00502e52 (12 chunks) |
|---|---|---|
| type lu = 21 | **97 / 97** | **86 / 86** |
| ref0 (l'unite, domaine 4) presente | 97 / 97 (17 index distincts) | 86 / 86 (22 index) |
| charge R(2) = niveau + 1 | **1 x50 · 0 x47** (50 entrees / 47 sorties) | **1 x48 · 0 x38** |
| listes multiples (2e evenement) | 15 (types 38, 36, 5, 82, 1) | 16 (types 38, 0, 36, 6, 5) |
| trame apres l'evenement : fermeture | 37,8 % (temoin 0xA0 : 36,3 %) | 20,0 % (seuil 18 %) |
| masques 1..7 des deltas aboutis | **99,4 %** (n=168) | **99,3 %** (n=136) |
| verdict M3 (ecrit avant mesure) | **TENU** | **TENU** |

Les charges ne prennent QUE les valeurs 0 et 1 (= niveaux −1 et 0) : **des paires
entree/sortie de lunette**, quasi equilibrees — la semantique attendue, mesuree.

## Consequences

1. **LA LUNETTE EST DANS LA BOBINE.** La conclusion « aucun evenement de zoom »
   (phases 3-10 du chantier visee, negatif « triple-verrouille ») est REFUTEE : les trois
   chaines du negatif partageaient le meme decalage d'un bit (le bit de configuration
   ignore : type lu = octet & 0x7F au lieu de bits 2..8 ; octets attendus 0x95/0xA4 au lieu
   de 0xCA/0xD2). C'est exactement le piege que la methode du plan citait : « deux chaines
   concordantes compatibles avec une explication plus simple que personne n'avait
   cherchee ». ~400 000 evenements unit_zoom sur le corpus. RESTE pour le produit : associer
   ref0 (index domaine 4) aux joueurs, et croiser avec la verite terrain du chantier visee.
2. **0xD2 = action_weapon_fire (type 36), pas PlayerGameEventSmall.** L'affirmation
   inverse du lot E (E7, « PROUVE cote binaire ») portait sur le cadrage errone. La charge
   du type 36 (lecteur FUN_14080C1F8, variable, refs var-int internes) est la cible pour la
   visee complete et la victime — fire_events.go y lit deja des morceaux stables aux
   offsets 36..142, qui se relisent maintenant comme [3 refs de l'evenement + debut de
   charge].
3. **Le « gisement 0xD3 » = recharges d'arme (38) + amorces de lancer de grenade (39)** —
   moins glorieux que des kills, mais deux signaux produit neufs (economie de munitions,
   grenades par joueur) si la charge se decode (type 38 : 8 octets, lecteur 0x1407f0ff8).
4. **0xC0 = damage_aftermath / damage_section_response (983 k)** : le VRAI gisement de
   degats par coup, a instruire.
5. Les « k gagnants » des balayages etaient la longueur MODALE de l'evenement de tete
   (2 + 7 + refs + charge, variable par paquet) — le mystere de l'« en-tete par famille »
   est dissous : il n'y a pas d'en-tete, il y a un evenement.
6. Les recensements passes par premier octet restent des mesures valides du COUPLE
   (type>>1) ; leurs etiquettes se corrigent par la table ci-dessus.

## La distribution corpus entier (TestLot1TypesCorpus, 1 367 films, 3e chaine de validation)

34 600 422 paquets sans evenement de tete (bit 1 = 0, trames pures) ; principaux types de
tete : **action_weapon_fire 2 535 816** · Script 1 023 143 · PlayerGameEventSmall 922 722 ·
**damage_aftermath 872 495** · projectile_detonate 458 933 · **weapon_reload 404 027** ·
**unit_zoom 399 988** · **biped_pickup 236 860** (utile au chantier ramassage) · Dialogue2D
195 107 · AIDialog 193 857 · impacts de projectile 245 358 · **biped_throw_initiate
124 235** · damage_section_response 111 388 · … · PlayerKilledEvent 108. Tous les types
observes < 123, distribution semantiquement plausible partout. **Fermeture arithmetique
gratuite : 404 027 + 124 235 = 528 262 = le compte historique exact du « gisement 0xD3 »**
(methode §5 : trois mesures qui se ferment sans ajustement).

## Le squelette de la charge du type 36 (action_weapon_fire), lu au decompile

`FUN_14080C1F8(out 0x328 octets, flux, drapeau reseau p5)` — ordre de lecture (p5 = 0
suppose en mode film, A CONFIRMER ; sous-lecteurs marques ? a resoudre).

EN AMONT, l'en-tete d'evenement du type 36 (domaines lus a vtable+0x58 = 0x14080a048,
meme code froid que le type 21) :

```
[1 continuation][R(7) = 36]
ref0 : R(1) porte ; si 1 : R(1) sonde ; R(sonde ? 9 : 13) index ; R(2) gen  <- L'ATTAQUANT (domaine 1)
ref1 : R(1) ; si 1 : R(13) + R(2)                                            (domaine 8)
ref2 : R(1) ; si 1 : R(13) + R(2)                                            (domaine 7)
```

Puis la charge :

```
R(1) -> out[0]        LE CHEMIN COURT : si 1, quelques champs puis RETOUR PRECOCE
                      (la « variante courte » historique de fire_events — bit 7 sous
                      leur cadrage = ce bit-ci sous le modele M)
R(1) -> out[0x1c]     drapeau « bloc supplementaire » (garde plusieurs blocs plus bas)
R(7) + R(1)  RESOLU   -> out[1] (7 bits bas + 1 bit haut)
[R(1); si 1: R(5)]    -> out+0x0c
[R(1): -1 | R(2)] RESOLU -> out+0x08 (index -1 ou 0..3 — slot/cause ?)
[R(1); si 1: R(32)]   -> out+0x10
R(32) "variant_name"  -> out+0x14   <- L'ARME (id de variante) — NOMME dans l'exe
R(1) -> out[0x1d] · R(1) -> out[2] · [si 0x1c: R(1), R(1), si 1: horodatage 0x1431a0abc]
--- chemin long seulement ---
FUN_14080cc68 RESOLU  -> DEUX COMPTES en code a prefixe :
                        R(1) toutVide ; si 1 : cibles=0 ET composantes=0
                        sinon cibles      = [R(1) : 1 -> 1 | 0 -> R(4)]
                        puis  composantes = [R(1) : 1 -> 0 | 0 -> [R(1) : 1 -> 1 | 0 -> R(4)]]
                        (n1 = composantes -> out+0x34, n2 = cibles -> out+0xf8)
BOUCLE 1 x n1         composantes de degat : R(2) + R(1) + [p5==0: R(32) | var-int dom 1]
BOUCLE 2 x n2         LES CIBLES : R(4) + R(1) porte ; si 1 :
                        R(w) avec w = FUN_1406d310c(6)  <- LA VICTIME (ref domaine 6, ~9 bits)
                        + [n1<3 ? R(1) : R(4)] + R(16) + FUN_140c1e924(?)
FUN_1406cd5b8         lecture quantifiee composite -> out+0x2a0
FUN_1408eff64         ? -> out+0x2c8
R(30) -> out+0x28     LA VISEE (cubemap 30 bits) — TOUJOURS PRESENTE en p5==0
                      (le « 19 % » de fire_events = incapacite a traverser les boucles)
[si !0x2dd: [R(1); si 1: R(6) dequant] + R(6) dequant] -> out+0x2e0/0x2e4
[si 0x1c: horodatage + si drapeau build 0x24: R(n)]
[si !0x1c: R(2)-1 -> out+0x1e · [R(1); si 1: R(n)] + 2 sous-lecteurs]
R(6) -> out+0x30c · [R(1); si 1: R(n)] -> out+0x314 · [p5==0: R(1)] -> out[4]
[R(1); si 1: vecteur quantifie 3 composantes] -> out+0x318 (position ?)
```

Etat des sous-lecteurs (31/08) : RESOLUS — FUN_141fcf670 = R(7)+R(1) ; FUN_1406d00ec =
[R(1):-1 | R(2)] ; FUN_14080cc68 = les deux comptes (code a prefixe, voir ci-dessus) ;
FUN_1407ef8e4 = R(3) ; FUN_1407f0278 = R(2) (un TAG). PARTIELS — FUN_1406cd5b8 et
FUN_1408eff64 sont des lecteurs COMPOSITES multi-champs (chacun : R(1) porte, un tag R(2)
via FUN_1407f0278, puis selon le tag un R(32) et/ou un R(6), plus des R(5) gardes) — la
forme est lue mais l'ordre exact reste a fixer champ par champ. FUN_140c1e924 (dans la
boucle des cibles) non lu. p5 (drapeau reseau) suppose 0 en mode film, A CONFIRMER.

## LES DEUX LECTEURS COMPOSITES SONT PERCES (workflow type36-subreaders, 31/08)

Un workflow multi-agents (11 agents, decompilation parallele + verification adverse au
DESASSEMBLAGE + synthese) a rendu la grammaire bit-exacte des deux composites, une
correction adverse integree (le verificateur de FUN_140c9eabc a rendu confirmed=false et
corrige). Grammaires cablees dans lot1SkipCd5b8 / lot1SkipEff64 :

- **FUN_1406cd5b8** : A=R(1) B=R(1) ; si B : sous-enreg FUN_140c9eabc [g0=R(1) ; si1 :
  tag=R(2) ; tag1 : R(32)+[R(1);si1:R(6)] · tag2 : R(32)] ; si A : R(4)+R(4) ; R(3) drapeaux ;
  si drapeaux&2 : g=R(1) ; si g==0 : R(20)+R(14) ; puis si A : C=R(1) ; si C : R(5).
- **FUN_1408eff64** (p5==0) : main=R(1) ; si main : tag=R(2) ; tag1 : R(32)+[R(1);si1:R(6)] ·
  tag2 : R(32).
- Autres sous-lecteurs resolus : FUN_140c1e924 (par-cible = 3*R(w), w appelant),
  FUN_1431a0abc (horodatage = R(1)+[si1:R(10)]), FUN_140c9e738 sel=1 (R(1)+[si0:R(20)+R(14)]),
  FUN_1406d84b4 = R(4) dans ce caller. Detail : journal du workflow.

## CE QUI EST PROUVE, ET CE QUI NE L'EST PAS (mesure discriminante, 2 films)

- **EN-TETE (attaquant + arme) : PROUVE.** Oracle discriminant = l'arme (variant_name R(32))
  est CATEGORIELLE : **11,0 % de valeurs distinctes** (27/245) sur 000d5950, **10,9 %**
  (30/276) sur 00502e52, contre un temoin de bruit a **79-81 %**. Type lu = 36 sur 100 %,
  attaquant present 100 %. Le tireur et son arme se decodent bit-exact pour les 2,5 M
  d'evenements action_weapon_fire.
- **VISEE R(30) : PLAUSIBLE, NON PROUVEE.** Le vecteur unitaire valide 240/240 mais l'oracle
  est NON DISCRIMINANT a 30 bits (6*gridSize^2 ~ 2^30 : un offset FAUX valide aussi 100 %).
  L'oracle categoriel du code de visee est faiblement favorable (50,8 % distinct vs bruit
  56,2 % ; 26,7 % vs 34,2 %) mais pas concluant — a 30 bits une vraie visee ne se repete pas
  exactement non plus. Les DEUX lecteurs composites ont une grammaire RE-verifiee (chaine
  independante), mais leur validation bout-en-bout sur film attend l'ORACLE DE TRAME.
- **CIBLES / composantes : le cadrage jusqu'a la visee est coherent** (les comptes 0/0 du cas
  modal ne desalignent pas l'en-tete, prouve par l'oracle arme en amont), mais le decodage
  des cibles elles-memes (victime) n'est pas fait.

## damage_aftermath (type 0, 872 k) DECODE ET PROUVE — le vrai enregistrement de touche

Workflow `damage-aftermath-reader` (10 agents, decompilation parallele + verification adverse
+ synthese, corrections integrees : largeurs 19, victime = 15 bits). Grammaire complete cablee
(lot1DecodeDamageAftermath) ; 3 references d'en-tete du type 0 = domaines 1, 1, 7 (lus dans
l'exe, descripteur 0x144724f80 vtable+0x58). L'ORACLE DE TRAME DISCRIMINANT tranche : apres
avoir decode l'evenement EN ENTIER, on lit le bit de continuation puis la trame de records,
et on mesure sa PROFONDEUR (records/paquet). Au BON cadrage : **2,2-2,4 records/paquet** (une
vraie trame de tick, cf. 0xA0 ~2,9) ; a un offset FAUX (+3 bits) : **0,17 record/paquet** (la
trame tombe aussitot sur un faux marqueur de fin — fermeture triviale). **Facteur 13, TENU sur
000d5950 et 00502e52** (build de reference). Le film ancien 06dfe6d9 (HI_1_8_0, registre 49
blocs, grammaire de composants differente) rend une profondeur moindre : effet de build sur la
trame, hors sujet damage_aftermath.

PIEGE DE METHODE eprouve ici : la metrique DISCRIMINANTE est la PROFONDEUR, PAS le taux de
fermeture (au temoin, le taux de fermeture est trompeusement HAUT — 88-90 % — parce que la
trame desynchronisee se ferme en 0 record sur un faux marqueur ; le bon cadrage se ferme MOINS
souvent mais va LOIN, comme 0xA0). Et la victime n'a pas d'oracle geometrique : le juge est
structurel (la trame reprend).

Ce que damage_aftermath rend (mesure, 2 films) :
- **SOURCE (tag du degat)** : 10-11 valeurs distinctes — categoriel = l'arme/effet responsable.
- **DEGAT (magnitude, code R(5) sur [0,16])** : distribue et groupe (1x81, 0x43, 2x20...).
- **PARTICIPANTS** (refs d'en-tete domaine 1) : ref0 22-24 distincts, ref1 15-21 — les entites
  blesse / responsable (plus que 8 joueurs : inclut projectiles/objets, attendu).
- **VICTIME domaine-0 finale** : presente 6-15 % — un objet secondaire (le blesse principal est
  vraisemblablement une des deux refs d'en-tete domaine 1, a departager).

C'EST LA VOIE PRECISION / TOUCHES : chaque damage_aftermath = un coup au but, avec l'arme
source, la magnitude, et les participants. 872 k sur le corpus.

**DEGAT EN CLAIR RESOLU (31/08)** : la magnitude = `dq(R(5), 0, DAT_143cd8454=16.0)` sur 32
niveaux (resolution 0.5), et la porte d'echelle (13) multiplie par `Kscale = DAT_143cd84ec =
-1.0` — c'est un BIT DE SIGNE : magnitude positive = degat, negative = SOIN (recharge). Mesure :
moyenne ~1.5 sur [0,16], ~10-19 % de valeurs negees (soins). La valeur de degat est donc
lisible en clair (cablee lot1Dequant + porte d'echelle). Reste (semantique) : departager
laquelle des deux refs d'en-tete domaine 1 est le blesse (croiser killsource).

**VISEE DU TYPE 36 — REELLE ET PROUVEE (31/08), correction d'un exces de prudence.**
L'utilisateur a rappele qu'on gere DEJA la visee : `fire_events.go` la lit a l'offset FIXE
bit 113 (largeur 30), gardee par flags[110]=1/[111]=0/[112]=0, pour 19 % des records. La
confrontation (`TestLot1ViseeCompare`) tranche par un ORACLE GEOMETRIQUE : une vraie visee est
concentree pres de l'horizontale (les joueurs visent peu en tangage), le bruit est uniforme sur
la sphere (E|composante| = 0.5 par axe). Mesure sur 2 films :
- **fire_events @113 : E|x|=0.27, E|y|=0.79, E|z|=0.34 · part x<0.3 = 70 %** — STRUCTURE NETTE,
  tres au-dessus de l'uniforme. VRAIE VISEE, PROUVEE.
- ma visee modele-M (bit ~80-81, variable) et le controle : E ~0.5 partout = BRUIT.

VERDICT : la visee est reelle et deja disponible en production (fire_events). Ce qui etait
INCONCLUANT n'etait pas la visee mais MA reconstruction : mon decodage modele-M du type 36 a un
BUG DE CADRAGE entre l'arme et la visee (il produit une position VARIABLE, ~80-81, la ou la
verite est ~FIXE a 113 — la structure arme->visee de ces paquets est constante et
`fire_events` la capture par offset fixe ; mon modele la sur-parse). L'ecart est de ~32 bits
(une largeur d'arme R(32) : fire_events lit l'arme sur 64 bits @44/76, mon modele sur 32).
CONSEQUENCE : attaquant + arme (prouves, oracle categoriel) et visee (prouvee, fire_events)
sont ACQUIS ; le seul residuel est le cadrage exact des champs post-arme de MON decodeur
type 36 (utile pour la victime dans la liste de cibles — mais cette victime est deja donnee
par damage_aftermath). fire_events fait foi pour la visee.

## LE JUGE DEFINITIF A POSER — l'oracle de trame (POSE pour damage_aftermath, cf. ci-dessus)

Le seul oracle DISCRIMINANT pour les composites+visee : decoder l'evenement 36 EN ENTIER
(y compris les champs post-visee : [R(1);si1:R(6)]+R(6) sous 0x2dd, R(2)+sous-lecteurs sous
0x1c, R(6), R(1)+[si1:R(w)], vec3), puis le bit de continuation, puis la TRAME de records —
et verifier qu'elle se ferme avec des masques 1..7 (99 % sous bon cadrage, 10 % au hasard,
oracle deja eprouve). Les quelques largeurs runtime post-visee se calibrent par balayage
contre cet oracle (technique idLowBits du depot). C'est le prochain geste, et il tranche.

## (archive) le premier essai du juge visee

Le controle decisif : dans le cas MODAL (0 cible, 0 composante, portes des sous-lecteurs a
0), la visee R(30) est a 3 bits de la fin du preambule ; un vecteur UNITAIRE valide
(DecodeAimVectorChecked) prouverait toute la chaine d'un coup. Mesure (TestLot1TirsEtCibles,
000d5950) : sur les 240 paquets a 0 cible, **229 ont au moins une porte non vide dans les
deux sous-lecteurs composites** (attendu : ces lecteurs portent des champs) et sont donc
SAUTES faute de leur ordre exact ; les 11 restants, portes toutes a zero, donnent
**11 vecteurs unitaires VALIDES sur 11, zero invalide**. C'est un signal POSITIF pour le
cadrage du preambule mais n=11 est trop faible pour un verdict — il faut fermer les deux
lecteurs composites (ci-dessus) pour lever les 229 sautes et porter n a quelques centaines.
NE PAS conclure sur la precision avant ce n.

A resoudre pour un decodeur bit-exact et le garde-fou : ordre exact de FUN_1406cd5b8 /
FUN_1408eff64 (champ par champ), FUN_140c1e924, p5, largeur runtime du domaine 6 — puis
validation du decodeur type 36 contre le golden killsource AVANT tout branchement.

## LA VOIE « PRECISION » (demande utilisateur du 30/08 au soir) — ou elle se calcule

La question : peut-on savoir si un tir/projectile TOUCHE un joueur ? Ce que le modele offre,
a noter en priorite pour un futur lot :

1. **Le numerateur (touches)** : le type 36 `action_weapon_fire` (2 535 816 sur le corpus)
   porte un **compte de cibles n2** (code a prefixe resolu, cf. squelette) puis la LISTE des
   cibles — chaque cible = R(4) + une reference **domaine 6** (l'entite touchee : joueur,
   vehicule, objet). Un evenement a n2 = 0 est un tir/degat sans cible. S'y ajoutent les
   familles `damage_aftermath` (872 k) / `damage_section_response` (111 k) et les impacts
   projectile (245 k) — plusieurs canaux de « touche » a recouper.
2. **Le denominateur (tirs tires)** : PAS d'evenement de tir manque dans le film (doctrine
   confirmee : type 35 request_weapon_fire = 0 occurrence sur le corpus). Le denominateur
   vient des **decrements de munitions** (`weapon-state-rounds-inventory`, i37 du bipede,
   deja lu par le traverseur — la methode du dossier l'utilise depuis juillet). Precision
   par arme = touches(36, par variant_name) / decrements(i37), les deux OFFLINE.
3. **Mesure preliminaire** (TestLot1TirsEtCibles, 000d5950, 12 chunks, en-tete du type 36
   decode bit-exact jusqu'aux comptes) : type=36 sur 245/245 ; attaquant (ref0, domaine 1)
   present 245/245 (31 index) ; chemin court 0 % ; **27 armes distinctes** (variant_name,
   repartitions plausibles — le champ est bien accroche) ; comptes lus : **98 % toutVide
   (0 cible)**, 5 evenements a (10 cibles, 5 composantes). RESERVE D'INTERPRETATION : le
   98 % peut etre reel (listes remplies seulement sur certains tirs) ou trahir un decalage
   residuel entre l'arme et les comptes — LE JUGE sera la visee R(30) en bout de chaine
   (vecteur unitaire verifiable par DecodeAimVectorChecked) une fois FUN_1406cd5b8 et
   FUN_1408eff64 resolus. NE PAS batir sur le 98 % avant ce controle.

## VEHICULES — entrees et sorties (demande utilisateur, mesure corpus 1 367 films)

Octet de tete = `0xC0 | (type>>1)`, bit8 dans l'octet suivant (PIEGE evite : base 0xC0, pas
0x80 — le bit de continuation est a 1). En-tete decode (attaquant + charge R(6) = siege) :

| Evenement | type | octet | corpus | films | siege R(6) dominant | unites (ref0) |
|---|---|---|---|---|---|---|
| biped_board_vehicle (embarquement) | 8 | 0xC4 | **374** | 154 | 16 (x197), 40, 0, 8, 41 | 243 distinctes |
| unit_enter_vehicle | 53 | 0xDA | **0** | 0 | — | — (absent des films arene) |
| unit_exit_vehicle (sortie) | 22 | 0xCB | **5 600** | 279 | 0 (x3 911), 1, 4, 8, 24 | 256 distinctes |

Lecture : **les sorties (5 600) et les embarquements (374) sont dans la bobine et se
decodent** ; le siege (charge R(6)) et l'unite qui monte/descend (ref0) sortent proprement.
L'asymetrie board/exit (374 vs 5 600) est a expliquer (ejections forcees / morts en
vehicule comptent en sortie ? embarquement limite a certains sieges ?) — non tranche. Le
type 53 `unit_enter_vehicle` est absent (0) : sur ce corpus arene, l'embarquement passe par
le type 8. C'est un signal produit neuf : temps passe en vehicule, par joueur, par siege.

## Ce qui reste ouvert

- La charge des types a grammaire variable (36 en tete) : largeurs R(n) sur pile a
  resoudre (Ghidra, lot E incertitude n4) avant de decoder tir par tir.
- bit8 par famille sur le corpus entier (0/1 par type) — une passe.
- Le pont index domaine 4 -> joueur (ref0 de unit_zoom) : mesurable par correlation avec
  les fils de vie ; gate produit final = verite terrain lunette du chantier visee.
- Les listes multiples (15/16 % des 0xCA) : decoder l'evenement suivant exige sa charge.
