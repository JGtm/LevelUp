# VERDICT — GEOMETRIE SEULE contre l'oracle pro v2 (item 2bis.D, 2026-09-20)

> **GEOMETRIE SEULE — STRICT : NO-GO** — 1 carte(s) de VALIDATION sur les 4 exigees tiennent rappel >= 0.70 ET precision >= 0.60 (bazaar) ; 1 piege(s) PUR(S) colore(s) sur les cartes de validation.
>
> **GEOMETRIE SEULE — ELARGI (validation + calibrage, N'EST PAS un GO strict) : NO-GO** — 1 carte(s) sur les 4 exigees tiennent (bazaar) ; 2 piege(s) PUR(S) sur ces cartes. Les cartes de calibrage ont servi a choisir les reglages : un GO elargi dit que la methode tient sur ce qu'elle a vu, pas qu'elle generalise.

> Genere par `go test -tags research ./cmd/mappower-build/ -run VerdictV2` (`verdict_v2_*_research_test.go`, `verdict_lignees_research_test.go`). Fichier juge : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-power-positions\.ai\V7.5\positions_de_force\geometrie_2026-09-20\positions_geo.json` (cuisson du 2026-09-20T12:48:45Z). Reglage serialise dans le fichier : `{"PoidsH":0.3,"PoidsV":0.2,"PoidsE":0.15,"PoidsR":0.2,"PoidsM":0.15,"PorteeRessourceM":20,"QuantileSeuil":0.9,"TailleMiniComposante":12,"MaxPositions":8,"RayonMaximumLocalM":3,"RayonPositionM":4}`. Oracle : `ORACLE_PRO_V2_2026-09-20.md` (§6 positions, §7 contre-exemples).

Regles appliquees, toutes ecrites avant la mesure (plan D11, oracle v2 §10) :

- **retrouvee** : le polygone de la position couvre >= 30 % de l'aire de la zone attendue, OU son barycentre tombe dans la zone ;
- **rappel** = zones `forte` retrouvees / zones `forte` RESOLUES au vocabulaire du depot ;
- **precision v2** = positions qui TOUCHENT une zone `forte` (meme relation que « retrouvee ») / positions calculees — les zones `faible` ne comptent plus ;
- rappel rapporte A PART sur les fortes de raison « hauteur » ou « lignes de vue » (HV), et sur celles de raison « arme » ou « objectif » sans hauteur ni vue (AO) ;
- les lignes `zone_en = ?` de l'oracle sont exclues ;
- **temoin GEOGRAPHIQUE** : chaque zone attendue est remplacee par la zone nommee de la carte dont le centroide est le plus eloigne ;
- **calibrage** (aquarius ; recharge ; streets) : rapportees, NON comptees ; **validation** : bazaar ; empyrean ; forbidden ; live fire ; solitude, plus toute carte ou l'oracle v2 porte une forte resolue ; sans forte : « hors oracle » ;
- sur une carte a 2 fortes, un rappel de 0,50 se lit « indetermine » ;
- contre-exemples (§7) : barycentre dans une zone deconseillee ; piege PUR si l'oracle ne decrit jamais la zone comme une position, PARTAGE sinon ; le GO exige ZERO piege PUR.

**Live Fire** : les positions v2 (`mesures_v2_2026-09-20/positions_v2.json`) ont ete mesurees SANS la variante classee « Live Fire - Ranked » (positions de kill fausses en base, decouverte 1 du plan ; `--exclure-variantes`, 3 854 kills ecartes selon le document de mesure v2). Elle compte donc comme carte de validation ici, contrairement au verdict v1 ou elle etait contaminee.

Les aires sont mesurees par echantillonnage au pas de 0.2 m.

## 1. Rappel, precision, temoin — carte par carte

| Carte | Role | Cle oracle | `forte` (resolues/total) | Rappel | Rappel temoin | Rappel HV (n) | Rappel AO (n) | Positions | Precision (fortes) | Precision temoin | Lecture |
|---|---|---|---|---|---|---|---|---|---|---|---|
| aquarius | calibrage | `ctf_aquarius` | 2/2 | 1.00 | 0.00 | 1.00 (2) | 0.00 (0) | 6 | 0.33 | 0.00 | calibrage — rapportee, NON comptee (ECHOUE) |
| bazaar | validation | `ctf_bazaar` | 5/5 | 1.00 | 0.00 | 1.00 (4) | 1.00 (1) | 8 | 0.88 | 0.00 | TIENT |
| forbidden | validation | `ctf_forbidden` | 2/2 | 0.50 | 0.00 | 0.00 (0) | 0.50 (2) | 8 | 0.12 | 0.00 | indetermine (rappel 0,5 sur 2 fortes) |
| live fire | validation | `sgh_interlock` | 3/3 | 0.67 | 0.00 | 1.00 (1) | 0.50 (2) | 7 | 0.71 | 0.00 | ECHOUE |
| recharge | calibrage | `sgh_blueprint` | 9/9 | 0.22 | 0.40 | 0.40 (5) | 0.00 (4) | 7 | 0.29 | 0.43 | calibrage — rapportee, NON comptee (ECHOUE) |
| streets | calibrage | `sgh_streets` | 3/3 | 0.67 | 0.00 | 1.00 (1) | 0.50 (2) | 5 | 0.60 | 0.00 | calibrage — rapportee, NON comptee (ECHOUE) |


## 2. Temoin negatif GEOGRAPHIQUE — chaque forte remplacee par la zone la plus eloignee

Le temoin v1 (permutation alphabetique) deplacait souvent l'attente de quelques metres. Ici la zone attendue est remplacee par celle dont le centroide est LE PLUS ELOIGNE sur la carte : si rappel et precision ne s'effondrent pas, l'appariement ne mesure que la densite des zones, pas leur identite.

| Carte | Rappel reel | Rappel temoin | Delta | Precision reelle | Precision temoin | Delta | Remplacements des fortes (distance des centroides) |
|---|---|---|---|---|---|---|---|
| aquarius | 1.00 | 0.00 | -1.00 | 0.33 | 0.00 | -0.33 | Hydro -> Yellow Refrigeration (24 m) ; Top Mid -> Yellow Refrigeration (18 m) |
| bazaar | 1.00 | 0.00 | -1.00 | 0.88 | 0.00 | -0.88 | Cafe -> East Base (32 m) ; Den -> West Base (30 m) ; Market -> East Base (22 m) ; Palm Tree -> East Base (34 m) ; Tower -> East Base (24 m) |
| forbidden | 0.50 | 0.00 | -0.50 | 0.12 | 0.00 | -0.12 | Dried Rat Hole -> Overgrown Back Ledge (41 m) ; Overgrown Rat Hole -> Dried Back Ledge (41 m) |
| live fire | 0.67 | 0.00 | -0.67 | 0.71 | 0.00 | -0.71 | Hallway -> Nest (50 m) ; Platform -> Landing Pad (82 m) ; Tower -> Landing Pad (93 m) |
| recharge | 0.22 | 0.40 | +0.18 | 0.29 | 0.43 | +0.14 | Attic -> Hydro (29 m) ; Control Room -> Maintenance Bay (22 m) ; Hydro -> Orange Pipes (35 m) ; Long Hall -> Elevator (28 m) ; Maintenance Bay -> Hydro (33 m) ; Orange Pipes -> Hydro (35 m) ; Pit -> Hydro (19 m) ; Platform -> Whirlpool Ledge (25 m) ; Whirlpool Dam -> Hydro (28 m) |
| streets | 0.67 | 0.00 | -0.67 | 0.60 | 0.00 | -0.60 | Cafe -> Arc Street Hallway (31 m) ; Main Street -> East Alley (21 m) ; Subway Balcony -> East Alley (31 m) |
| **moyenne** | **0.68** | **0.07** | **-0.61** | **0.49** | **0.07** | **-0.42** | |

## 3. Contre-exemples (§7 de l'oracle v2)

Une position est un contre-exemple quand son BARYCENTRE tombe dans une zone deconseillee. PURS : l'oracle ne decrit la zone nulle part comme une position. PARTAGES : la zone est les deux a la fois dans l'oracle (Long Hall, Market, Pit, East / West Tower...). **Le GO porte sur les PURS des cartes de validation.**

| Carte | Role | Pieges PURS colores | Pieges aussi decrits comme positions |
|---|---|---|---|
| aquarius | calibrage | aquarius__arene__geo__2 dans Blue Courtyard | aquarius__arene__geo__2 dans Blue Base ; aquarius__arene__geo__5 dans Yellow Base ; aquarius__arene__geo__6 dans Blue Base |
| bazaar | validation | — | bazaar__arene__geo__2 dans Market ; bazaar__arene__geo__6 dans Market ; bazaar__arene__geo__7 dans Market |
| forbidden | validation | — | — |
| live fire | validation | live_fire__arene__geo__2 dans Canal | — |
| recharge | calibrage | — | recharge__arene__geo__7 dans Batteries |
| streets | calibrage | — | — |

**Total des pieges PURS colores : 2**, dont **1 sur les cartes de validation** (c'est ce compte-la qui engage le GO).

## 4. Faux positifs et faux negatifs, nommes

| Carte | Faux negatifs HV (hauteur / lignes de vue) | Faux negatifs AO (arme / objectif) | Faux positifs (positions sans forte, avec leur zone dominante) | `forte` NON RESOLUES au vocabulaire |
|---|---|---|---|---|
| aquarius | — | — | aquarius__arene__geo__1 (zone dominante : pump, 94 %) ; aquarius__arene__geo__3 (zone dominante : planters, 94 %) ; aquarius__arene__geo__5 (zone dominante : yellow base, 95 %) ; aquarius__arene__geo__6 (zone dominante : blue base, 100 %) | — |
| bazaar | — | — | bazaar__arene__geo__5 (zone dominante : antenna, 49 %) | — |
| forbidden | — | Overgrown Rat Hole (objectif + acces) | forbidden__arene__geo__1 (zone dominante : overgrown street, 56 %) ; forbidden__arene__geo__2 (zone dominante : overgrown nest, 83 %) ; forbidden__arene__geo__3 (zone dominante : dried nest, 83 %) ; forbidden__arene__geo__4 (zone dominante : dried bottom street, 77 %) ; forbidden__arene__geo__5 (zone dominante : overgrown bottom street, 68 %) ; forbidden__arene__geo__6 (zone dominante : overgrown pillars, 70 %) ; forbidden__arene__geo__7 (zone dominante : sad trombone, 57 %) | — |
| live fire | — | Platform (arme + acces (position disputee)) | live_fire__arene__geo__5 (zone dominante : overlook, 79 %) ; live_fire__arene__geo__7 (zone dominante : nest, 18 %) | — |
| recharge | Control Room (hauteur + lignes de vue + acces) ; Maintenance Bay (hauteur + lignes de vue + acces) ; Whirlpool Dam (hauteur + lignes de vue + arme) | Hydro (arme + objectif + spawn) ; Long Hall (arme + acces) ; Orange Pipes (acces + arme) ; Pit (arme) | recharge__arene__geo__1 (zone dominante : sneaky, 91 %) ; recharge__arene__geo__4 (zone dominante : elevator, 96 %) ; recharge__arene__geo__5 (zone dominante : whirlpool ledge, 76 %) ; recharge__arene__geo__6 (zone dominante : whirlpool ledge, 85 %) ; recharge__arene__geo__7 (zone dominante : batteries, 77 %) | — |
| streets | — | Main Street (arme) | streets__arene__geo__1 (zone dominante : station tower 2, 69 %) ; streets__arene__geo__5 (zone dominante : old town, 87 %) | — |

Une forte NON RESOLUE est absente du catalogue de zones du depot pour cette carte : l'algorithme ne peut ni la trouver ni la manquer ; elle est hors du denominateur du rappel. Un faux positif est une position qui ne touche aucune forte — il peut toucher une zone `faible` de l'oracle, ce qui ne compte plus.

## 5. Chaque position calculee et sa zone nommee dominante

« Part » = fraction de la position couverte par la zone dominante. « Aire zone » = aire de la zone dominante (union des zones du meme libelle sur les cartes en miroir).

| Carte | Position | Score moyen | Aire pos. m2 | Zone dominante | Part | Aire zone m2 | Appariee a |
|---|---|---|---|---|---|---|---|
| aquarius | `aquarius__arene__geo__1` | 0.595 | 29.3 | pump | 0.94 | 58.0 | — |
| aquarius | `aquarius__arene__geo__2` | 0.585 | 16.6 | secret tunnel | 0.97 | 38.9 | Top Mid |
| aquarius | `aquarius__arene__geo__3` | 0.554 | 18.6 | planters | 0.94 | 114.6 | — |
| aquarius | `aquarius__arene__geo__4` | 0.547 | 17.1 | hydro | 0.94 | 102.4 | Top Mid |
| aquarius | `aquarius__arene__geo__5` | 0.545 | 36.8 | yellow base | 0.95 | 133.0 | — |
| aquarius | `aquarius__arene__geo__6` | 0.514 | 7.4 | blue base | 1.00 | 218.4 | — |
| bazaar | `bazaar__arene__geo__1` | 0.598 | 30.9 | tower | 0.92 | 72.6 | Tower |
| bazaar | `bazaar__arene__geo__2` | 0.584 | 21.1 | den | 0.67 | 50.6 | Market |
| bazaar | `bazaar__arene__geo__3` | 0.580 | 19.1 | west courtyard | 1.00 | 217.8 | Palm Tree |
| bazaar | `bazaar__arene__geo__4` | 0.568 | 15.3 | cafe | 0.73 | 46.9 | Cafe |
| bazaar | `bazaar__arene__geo__5` | 0.567 | 36.5 | antenna | 0.49 | 114.9 | — |
| bazaar | `bazaar__arene__geo__6` | 0.560 | 13.2 | market | 1.00 | 436.0 | Market |
| bazaar | `bazaar__arene__geo__7` | 0.559 | 5.2 | market | 1.00 | 436.0 | Market |
| bazaar | `bazaar__arene__geo__8` | 0.558 | 32.8 | den | 0.53 | 50.6 | Den |
| forbidden | `forbidden__arene__geo__1` | 0.651 | 25.2 | overgrown street | 0.56 | 28.8 | — |
| forbidden | `forbidden__arene__geo__2` | 0.651 | 22.7 | overgrown nest | 0.83 | 100.3 | — |
| forbidden | `forbidden__arene__geo__3` | 0.648 | 11.8 | dried nest | 0.83 | 26.3 | — |
| forbidden | `forbidden__arene__geo__4` | 0.646 | 34.3 | dried bottom street | 0.77 | 42.6 | — |
| forbidden | `forbidden__arene__geo__5` | 0.637 | 17.1 | overgrown bottom street | 0.68 | 41.3 | — |
| forbidden | `forbidden__arene__geo__6` | 0.635 | 16.8 | overgrown pillars | 0.70 | 64.5 | — |
| forbidden | `forbidden__arene__geo__7` | 0.622 | 6.6 | sad trombone | 0.57 | 188.3 | — |
| forbidden | `forbidden__arene__geo__8` | 0.617 | 19.4 | dried rat hole | 0.30 | 16.7 | Dried Rat Hole |
| live fire | `live_fire__arene__geo__1` | 0.618 | 22.9 | tower | 0.98 | 101.7 | Tower |
| live fire | `live_fire__arene__geo__2` | 0.612 | 15.9 | canal | 0.86 | 117.4 | Tower |
| live fire | `live_fire__arene__geo__3` | 0.609 | 28.3 | hallway | 0.81 | 157.4 | Hallway |
| live fire | `live_fire__arene__geo__4` | 0.597 | 18.1 | tunnel | 0.69 | 58.6 | Hallway |
| live fire | `live_fire__arene__geo__5` | 0.589 | 8.6 | overlook | 0.79 | 36.8 | — |
| live fire | `live_fire__arene__geo__6` | 0.585 | 10.0 | yard | 0.52 | 137.4 | Tower |
| live fire | `live_fire__arene__geo__7` | 0.584 | 12.9 | nest | 0.18 | 44.4 | — |
| recharge | `recharge__arene__geo__1` | 0.573 | 6.6 | sneaky | 0.91 | 8.8 | — |
| recharge | `recharge__arene__geo__2` | 0.551 | 13.2 | attic | 0.74 | 29.3 | Attic |
| recharge | `recharge__arene__geo__3` | 0.549 | 30.7 | platform | 0.54 | 17.0 | Platform |
| recharge | `recharge__arene__geo__4` | 0.547 | 19.9 | elevator | 0.96 | 70.1 | — |
| recharge | `recharge__arene__geo__5` | 0.524 | 7.4 | whirlpool ledge | 0.76 | 19.0 | — |
| recharge | `recharge__arene__geo__6` | 0.523 | 14.8 | whirlpool ledge | 0.85 | 19.0 | — |
| recharge | `recharge__arene__geo__7` | 0.520 | 20.7 | batteries | 0.77 | 112.2 | — |
| streets | `streets__arene__geo__1` | 0.630 | 21.0 | station tower 2 | 0.69 | 17.2 | — |
| streets | `streets__arene__geo__2` | 0.565 | 17.3 | subway | 0.52 | 60.6 | Subway Balcony |
| streets | `streets__arene__geo__3` | 0.539 | 14.4 | subway bend | 0.62 | 36.5 | Subway Balcony |
| streets | `streets__arene__geo__4` | 0.518 | 16.2 | cafe | 0.87 | 16.8 | Cafe |
| streets | `streets__arene__geo__5` | 0.510 | 28.7 | old town | 0.87 | 79.3 | — |

## 6. Couloirs — les positions de plus de 60 cellules : position ou salle ?

Le document de mesure v2 signale deux positions de 85 et 88 cellules retenues d'un bloc par la croissance au p90, et laisse au verdict de dire si c'est une position ou une salle. Lecture mecanique : SALLE ENTIERE si la zone dominante est couverte a plus de la moitie par la position ; COULOIR TRAVERSANT si au moins deux zones portent chacune un quart ou plus de la position (elle enjambe une frontiere) ; POSITION LARGE sinon (une partie d'une seule zone, ce que D1 appelle une position).

| Carte | Position | Cellules | Aire m2 | Zones traversees (part de la position / part de la zone) | Lecture |
|---|---|---|---|---|---|
| aquarius | `aquarius__arene__geo__1` | 64 | 29.3 | pump : 94 % de la position / 47 % de la zone ; planters : 68 % de la position / 31 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| aquarius | `aquarius__arene__geo__5` | 93 | 36.8 | yellow base : 95 % de la position / 33 % de la zone ; yellow courtyard : 15 % de la position / 4 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| bazaar | `bazaar__arene__geo__1` | 64 | 30.9 | tower : 92 % de la position / 39 % de la zone ; tower basement : 43 % de la position / 22 % de la zone ; market : 17 % de la position / 2 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| bazaar | `bazaar__arene__geo__5` | 67 | 36.5 | antenna : 49 % de la position / 31 % de la zone ; library : 30 % de la position / 41 % de la zone ; east market bridge : 27 % de la position / 34 % de la zone | COULOIR TRAVERSANT : a cheval sur 3 zones a 25 % ou plus |
| forbidden | `forbidden__arene__geo__1` | 65 | 25.2 | overgrown street : 56 % de la position / 49 % de la zone ; overgrown bottom street : 52 % de la position / 32 % de la zone ; dried hut : 49 % de la position / 59 % de la zone ; dried pillars : 31 % de la position / 12 % de la zone | COULOIR TRAVERSANT : a cheval sur 4 zones a 25 % ou plus |
| forbidden | `forbidden__arene__geo__4` | 65 | 34.3 | dried bottom street : 77 % de la position / 62 % de la zone ; dried street : 56 % de la position / 57 % de la zone ; sad trombone : 35 % de la position / 13 % de la zone ; dried nest : 17 % de la position / 23 % de la zone ; dried back nest : 13 % de la position / 6 % de la zone | SALLE ENTIERE : la zone dominante (dried bottom street) est couverte a 62 % |
| live fire | `live_fire__arene__geo__3` | 72 | 28.3 | hallway : 81 % de la position / 14 % de la zone ; tunnel : 30 % de la position / 15 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| recharge | `recharge__arene__geo__3` | 74 | 30.7 | platform : 54 % de la position / 97 % de la zone ; hydro : 30 % de la position / 2 % de la zone ; pit : 16 % de la position / 5 % de la zone | SALLE ENTIERE : la zone dominante (platform) est couverte a 97 % |
| streets | `streets__arene__geo__5` | 69 | 28.7 | old town : 87 % de la position / 32 % de la zone ; plaza stairs : 22 % de la position / 27 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |

## 7. Diagnostic — pourquoi chaque forte manquee a echappe a la v2

Diagnostic non rejoue : le fichier juge ne porte pas le reglage empirique v2 (pas d'hysteresis declaree). La geometrie et la fusion ont leurs propres outils.

## 8. Les lignees a regles egales

Chaque fichier de positions du chantier est juge ici avec les MEMES regles v2 (oracle v2, precision sur les fortes, temoin geographique) : c'est la seule lecture comparee qui vaille. Live Fire v1 etait CONTAMINEE (variante classee incluse) : sa colonne v1 se lit avec cette reserve. Une cellule « — » : la carte n'est pas dans le fichier de la lignee (la geometrie et la fusion ne couvrent que les six cartes cuites ; Empyrean et Solitude, cartes Forge sans geometrie, restent « empirique seul »).

| Carte | Role | Fortes | empirique v1 : rappel / precision / positions / pieges purs | empirique v2 : rappel / precision / positions / pieges purs | geometrie seule : rappel / precision / positions / pieges purs | fusion : rappel / precision / positions / pieges purs |
|---|---|---|---|---|---|---|
| aquarius | calibrage | 2 | 0.00 / 0.00 / 5 / 3 (echoue) | 1.00 / 0.25 / 4 / 1 (echoue) | 1.00 / 0.33 / 6 / 1 (echoue) | 1.00 / 0.33 / 6 / 0 (echoue) |
| bazaar | validation | 5 | 0.20 / 1.00 / 2 / 0 (echoue) | 0.20 / 0.33 / 3 / 0 (echoue) | 1.00 / 0.88 / 8 / 0 (TIENT) | 1.00 / 1.00 / 2 / 0 (TIENT) |
| empyrean | validation | 2 | 0.00 / 0.00 / 0 / 0 (aucune position) | 0.00 / 0.00 / 0 / 0 (aucune position) | — | — |
| forbidden | validation | 2 | 0.00 / 0.00 / 1 / 0 (echoue) | 0.50 / 0.50 / 2 / 0 (indetermine) | 0.50 / 0.12 / 8 / 0 (indetermine) | 0.50 / 0.25 / 4 / 0 (indetermine) |
| live fire | validation | 3 | 0.33 / 0.67 / 3 / 1 (echoue) | 0.33 / 0.40 / 5 / 0 (echoue) | 0.67 / 0.71 / 7 / 1 (echoue) | 1.00 / 0.25 / 4 / 1 (echoue) |
| recharge | calibrage | 9 | 0.33 / 0.75 / 4 / 1 (echoue) | 0.44 / 1.00 / 4 / 0 (echoue) | 0.22 / 0.29 / 7 / 0 (echoue) | 0.67 / 0.75 / 8 / 0 (echoue) |
| solitude | validation | 2 | 0.00 / 0.00 / 1 / 0 (echoue) | 0.50 / 0.33 / 3 / 0 (indetermine) | — | — |
| streets | calibrage | 3 | 0.33 / 0.25 / 4 / 0 (echoue) | 0.00 / 0.00 / 4 / 0 (echoue) | 0.67 / 0.60 / 5 / 0 (echoue) | 1.00 / 0.38 / 8 / 0 (echoue) |

- **empirique v1** : strict 0 carte(s) de validation tiennent (—), 1 piege(s) pur(s) ; elargi 0 carte(s) validation + calibrage (—), 5 piege(s) pur(s).
- **empirique v2** : strict 0 carte(s) de validation tiennent (—), 0 piege(s) pur(s) ; elargi 0 carte(s) validation + calibrage (—), 1 piege(s) pur(s).
- **geometrie seule** : strict 1 carte(s) de validation tiennent (bazaar), 1 piege(s) pur(s) ; elargi 1 carte(s) validation + calibrage (bazaar), 2 piege(s) pur(s).
- **fusion** : strict 1 carte(s) de validation tiennent (bazaar), 1 piege(s) pur(s) ; elargi 1 carte(s) validation + calibrage (bazaar), 1 piege(s) pur(s).

## 9. Geometrie et fusion

**Diagnostic geometrique.** Le score de `geo.ReglageGeoV1` est relu sur le CSV par noeud ; seuls les noeuds dont la cellule tombe dans la zone sont regardes. « Cellules zone » est l'emprise de la zone en cellules de 0,5 m, « avec noeud » celles que le sol derive couvre ; « maxima » compte les noeuds au-dessus du seuil qui dominent leur boule euclidienne 3D de 3 m (test plus strict que la dominance a 3 m de marche de la selection reelle).

| Carte | Zone manquee | Raison (oracle) | Cellules zone | Avec noeud | Noeuds | Score p50 | Score max | Seuil (p90) | Au-dessus | Maxima | Couverture par une position | Cause |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| forbidden | Overgrown Rat Hole | objectif + acces | 66 | 50 | 50 | 0.221 | 0.571 | 0.582 | 0 | 0 | 0 % | SOUS LE SEUIL : max 0.571 < 0.582 (p90 de la carte) |
| live fire | Platform | arme + acces (position disputee) | 50 | 40 | 44 | 0.518 | 0.613 | 0.563 | 5 | 0 | 11 % | RETENUE AILLEURS : une position touche la zone mais n'en couvre que 11 % (< 30 %) et son barycentre tombe dehors |
| recharge | Maintenance Bay | hauteur + lignes de vue + acces | 82 | 64 | 64 | 0.479 | 0.590 | 0.507 | 5 | 0 | 10 % | RETENUE AILLEURS : une position touche la zone mais n'en couvre que 10 % (< 30 %) et son barycentre tombe dehors |
| recharge | Pit | arme | 376 | 340 | 343 | 0.298 | 0.521 | 0.507 | 5 | 1 | 5 % | RETENUE AILLEURS : une position touche la zone mais n'en couvre que 5 % (< 30 %) et son barycentre tombe dehors |
| recharge | Hydro | arme + objectif + spawn | 1951 | 477 | 520 | 0.331 | 0.552 | 0.507 | 26 | 0 | 2 % | RETENUE AILLEURS : une position touche la zone mais n'en couvre que 2 % (< 30 %) et son barycentre tombe dehors |
| recharge | Whirlpool Dam | hauteur + lignes de vue + arme | 352 | 300 | 308 | 0.362 | 0.539 | 0.507 | 10 | 2 | 4 % | RETENUE AILLEURS : une position touche la zone mais n'en couvre que 4 % (< 30 %) et son barycentre tombe dehors |
| recharge | Orange Pipes | acces + arme | 137 | 114 | 117 | 0.333 | 0.529 | 0.507 | 3 | 0 | 0 % | SANS MAXIMUM LOCAL : 3 noeud(s) passe(nt) le seuil (max 0.529) mais aucun ne domine sa boule de 3 m (test euclidien, plus strict que la marche) |
| recharge | Control Room | hauteur + lignes de vue + acces | 243 | 193 | 242 | 0.328 | 0.533 | 0.507 | 2 | 1 | 0 % | TROP PETITE OU HORS PLAFOND : 1 maximum(s) local(aux) (max 0.533), aucune position ne touche la zone — composante < 12 noeuds a 4 m de marche, ou plafond de 8 (indecidable sans le graphe) |
| recharge | Long Hall | arme + acces | 342 | 216 | 216 | 0.397 | 0.497 | 0.507 | 0 | 0 | 1 % | SOUS LE SEUIL : max 0.497 < 0.507 (p90 de la carte) |
| streets | Main Street | arme | 157 | 130 | 130 | 0.344 | 0.434 | 0.468 | 0 | 0 | 0 % | SOUS LE SEUIL : max 0.434 < 0.468 (p90 de la carte) |

Chaque lignee a son document, genere par la meme commande avec son fichier de positions (depuis `apps/go-api`, `MAPPOWER_DATA_ROOT=<racine contenant data/>`) :

- **empirique v2** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/mesures_v2_2026-09-20/positions_v2.json MAPPOWER_VERDICT_SORTIE=VERDICT_V2_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`
- **geometrie seule** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/geometrie_2026-09-20/positions_geo.json MAPPOWER_VERDICT_SORTIE=VERDICT_GEO_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`
- **fusion** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/fusion_2026-09-20/positions_fusion.json MAPPOWER_VERDICT_SORTIE=VERDICT_FUSION_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`

Le reglage de fusion (`fusion.ReglageFusionV1`) a ete choisi par balayage sur les cartes de calibrage seulement : `fusion_2026-09-20/_calibrage_fusion.md`. Le diagnostic v2 (section 7) ne se rejoue que sur un fichier empirique v2 ; le diagnostic geometrique ci-dessus, que sur le fichier geometrique.

## 10. Ce que ces chiffres disent

Aucune zone `forte` manquee : rien a expliquer.
