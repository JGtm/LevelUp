# VERDICT — FUSION geometrie x empirique contre l'oracle pro v2 (item 2bis.D, 2026-09-20)

> **FUSION geometrie x empirique — STRICT : NO-GO** — 1 carte(s) de VALIDATION sur les 4 exigees tiennent rappel >= 0.70 ET precision >= 0.60 (bazaar) ; 1 piege(s) PUR(S) colore(s) sur les cartes de validation.
>
> **FUSION geometrie x empirique — ELARGI (validation + calibrage, N'EST PAS un GO strict) : NO-GO** — 1 carte(s) sur les 4 exigees tiennent (bazaar) ; 1 piege(s) PUR(S) sur ces cartes. Les cartes de calibrage ont servi a choisir les reglages : un GO elargi dit que la methode tient sur ce qu'elle a vu, pas qu'elle generalise.

> Genere par `go test -tags research ./cmd/mappower-build/ -run VerdictV2` (`verdict_v2_*_research_test.go`, `verdict_lignees_research_test.go`). Fichier juge : `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-power-positions\.ai\V7.5\positions_de_force\fusion_2026-09-20\positions_fusion.json` (cuisson du 2026-09-20T13:14:00Z). Reglage serialise dans le fichier : `{"PoidsGeo":0.6,"PoidsEmp":0.4,"EmpAbsent":0.25,"GeoAbsent":0,"Selection":{"RayonLissageM":0,"PlancherMatchs":0,"MinEngagementsDisque":0,"ForcePrior":0,"DeniveleReferenceM":0,"PoidsAvantage":0,"PoidsIntensite":0,"PoidsHauteur":0,"PoidsPortee":0,"PoidsCouverture":0,"PoidsAbri":0,"MinDirections":0,"UtiliseRangPondere":false,"QuantileSeuil":0,"SeuilScoreMin":0.7,"TailleMiniComposante":10,"MaxComposantes":8,"QuantileAmorce":0.9,"QuantileCroissance":0.8,"FermetureRayonCellules":1,"Connexite8":true}}`. Oracle : `ORACLE_PRO_V2_2026-09-20.md` (§6 positions, §7 contre-exemples).

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
| bazaar | validation | `ctf_bazaar` | 5/5 | 1.00 | 0.00 | 1.00 (4) | 1.00 (1) | 2 | 1.00 | 0.00 | TIENT |
| forbidden | validation | `ctf_forbidden` | 2/2 | 0.50 | 0.00 | 0.00 (0) | 0.50 (2) | 4 | 0.25 | 0.00 | indetermine (rappel 0,5 sur 2 fortes) |
| live fire | validation | `sgh_interlock` | 3/3 | 1.00 | 0.50 | 1.00 (1) | 1.00 (2) | 4 | 0.25 | 0.25 | ECHOUE |
| recharge | calibrage | `sgh_blueprint` | 9/9 | 0.67 | 0.60 | 1.00 (5) | 0.25 (4) | 8 | 0.75 | 0.25 | calibrage — rapportee, NON comptee (ECHOUE) |
| streets | calibrage | `sgh_streets` | 3/3 | 1.00 | 0.00 | 1.00 (1) | 1.00 (2) | 8 | 0.38 | 0.00 | calibrage — rapportee, NON comptee (ECHOUE) |


## 2. Temoin negatif GEOGRAPHIQUE — chaque forte remplacee par la zone la plus eloignee

Le temoin v1 (permutation alphabetique) deplacait souvent l'attente de quelques metres. Ici la zone attendue est remplacee par celle dont le centroide est LE PLUS ELOIGNE sur la carte : si rappel et precision ne s'effondrent pas, l'appariement ne mesure que la densite des zones, pas leur identite.

| Carte | Rappel reel | Rappel temoin | Delta | Precision reelle | Precision temoin | Delta | Remplacements des fortes (distance des centroides) |
|---|---|---|---|---|---|---|---|
| aquarius | 1.00 | 0.00 | -1.00 | 0.33 | 0.00 | -0.33 | Hydro -> Yellow Refrigeration (24 m) ; Top Mid -> Yellow Refrigeration (18 m) |
| bazaar | 1.00 | 0.00 | -1.00 | 1.00 | 0.00 | -1.00 | Cafe -> East Base (32 m) ; Den -> West Base (30 m) ; Market -> East Base (22 m) ; Palm Tree -> East Base (34 m) ; Tower -> East Base (24 m) |
| forbidden | 0.50 | 0.00 | -0.50 | 0.25 | 0.00 | -0.25 | Dried Rat Hole -> Overgrown Back Ledge (41 m) ; Overgrown Rat Hole -> Dried Back Ledge (41 m) |
| live fire | 1.00 | 0.50 | -0.50 | 0.25 | 0.25 | +0.00 | Hallway -> Nest (50 m) ; Platform -> Landing Pad (82 m) ; Tower -> Landing Pad (93 m) |
| recharge | 0.67 | 0.60 | -0.07 | 0.75 | 0.25 | -0.50 | Attic -> Hydro (29 m) ; Control Room -> Maintenance Bay (22 m) ; Hydro -> Orange Pipes (35 m) ; Long Hall -> Elevator (28 m) ; Maintenance Bay -> Hydro (33 m) ; Orange Pipes -> Hydro (35 m) ; Pit -> Hydro (19 m) ; Platform -> Whirlpool Ledge (25 m) ; Whirlpool Dam -> Hydro (28 m) |
| streets | 1.00 | 0.00 | -1.00 | 0.38 | 0.00 | -0.38 | Cafe -> Arc Street Hallway (31 m) ; Main Street -> East Alley (21 m) ; Subway Balcony -> East Alley (31 m) |
| **moyenne** | **0.86** | **0.18** | **-0.68** | **0.49** | **0.08** | **-0.41** | |

## 3. Contre-exemples (§7 de l'oracle v2)

Une position est un contre-exemple quand son BARYCENTRE tombe dans une zone deconseillee. PURS : l'oracle ne decrit la zone nulle part comme une position. PARTAGES : la zone est les deux a la fois dans l'oracle (Long Hall, Market, Pit, East / West Tower...). **Le GO porte sur les PURS des cartes de validation.**

| Carte | Role | Pieges PURS colores | Pieges aussi decrits comme positions |
|---|---|---|---|
| aquarius | calibrage | — | aquarius__arene__fusion__3 dans Blue Base ; aquarius__arene__fusion__4 dans Yellow Base |
| bazaar | validation | — | bazaar__arene__fusion__1 dans Market |
| forbidden | validation | — | — |
| live fire | validation | live_fire__arene__fusion__1 dans Canal | — |
| recharge | calibrage | — | recharge__arene__fusion__2 dans Batteries ; recharge__arene__fusion__6 dans Long Hall ; recharge__arene__fusion__7 dans Batteries |
| streets | calibrage | — | streets__arene__fusion__7 dans Main Street |

**Total des pieges PURS colores : 1**, dont **1 sur les cartes de validation** (c'est ce compte-la qui engage le GO).

## 4. Faux positifs et faux negatifs, nommes

| Carte | Faux negatifs HV (hauteur / lignes de vue) | Faux negatifs AO (arme / objectif) | Faux positifs (positions sans forte, avec leur zone dominante) | `forte` NON RESOLUES au vocabulaire |
|---|---|---|---|---|
| aquarius | — | — | aquarius__arene__fusion__3 (zone dominante : blue base, 55 %) ; aquarius__arene__fusion__4 (zone dominante : yellow base, 74 %) ; aquarius__arene__fusion__5 (zone dominante : yellow utility, 100 %) ; aquarius__arene__fusion__6 (zone dominante : blue utility, 99 %) | — |
| bazaar | — | — | — | — |
| forbidden | — | Overgrown Rat Hole (objectif + acces) | forbidden__arene__fusion__1 (zone dominante : dried bottom street, 36 %) ; forbidden__arene__fusion__3 (zone dominante : dried back base, 85 %) ; forbidden__arene__fusion__4 (zone dominante : overgrown base, 93 %) | — |
| live fire | — | — | live_fire__arene__fusion__1 (zone dominante : nest, 45 %) ; live_fire__arene__fusion__2 (zone dominante : green building, 97 %) ; live_fire__arene__fusion__4 (zone dominante : house, 66 %) | — |
| recharge | — | Hydro (arme + objectif + spawn) ; Orange Pipes (acces + arme) ; Pit (arme) | recharge__arene__fusion__1 (zone dominante : sneaky, 91 %) ; recharge__arene__fusion__7 (zone dominante : batteries, 82 %) | — |
| streets | — | — | streets__arene__fusion__1 (zone dominante : station tower 2, 64 %) ; streets__arene__fusion__2 (zone dominante : subway stairs, 83 %) ; streets__arene__fusion__3 (zone dominante : old town, 60 %) ; streets__arene__fusion__6 (zone dominante : old town stairs, 38 %) ; streets__arene__fusion__8 (zone dominante : old town alley, 85 %) | — |

Une forte NON RESOLUE est absente du catalogue de zones du depot pour cette carte : l'algorithme ne peut ni la trouver ni la manquer ; elle est hors du denominateur du rappel. Un faux positif est une position qui ne touche aucune forte — il peut toucher une zone `faible` de l'oracle, ce qui ne compte plus.

## 5. Chaque position calculee et sa zone nommee dominante

« Part » = fraction de la position couverte par la zone dominante. « Aire zone » = aire de la zone dominante (union des zones du meme libelle sur les cartes en miroir).

| Carte | Position | Score moyen | Aire pos. m2 | Zone dominante | Part | Aire zone m2 | Appariee a |
|---|---|---|---|---|---|---|---|
| aquarius | `aquarius__arene__fusion__1` | 0.794 | 55.6 | hydro | 0.52 | 102.4 | Top Mid |
| aquarius | `aquarius__arene__fusion__2` | 0.749 | 97.6 | planters | 0.73 | 114.6 | Top Mid |
| aquarius | `aquarius__arene__fusion__3` | 0.715 | 34.8 | blue base | 0.55 | 218.4 | — |
| aquarius | `aquarius__arene__fusion__4` | 0.713 | 56.9 | yellow base | 0.74 | 133.0 | — |
| aquarius | `aquarius__arene__fusion__5` | 0.690 | 6.6 | yellow utility | 1.00 | 76.6 | — |
| aquarius | `aquarius__arene__fusion__6` | 0.674 | 6.8 | blue utility | 0.99 | 78.9 | — |
| bazaar | `bazaar__arene__fusion__1` | 0.722 | 500.0 | market | 0.39 | 436.0 | Market |
| bazaar | `bazaar__arene__fusion__2` | 0.691 | 191.4 | west courtyard | 0.80 | 217.8 | Cafe |
| forbidden | `forbidden__arene__fusion__1` | 0.724 | 106.3 | dried bottom street | 0.36 | 42.6 | — |
| forbidden | `forbidden__arene__fusion__2` | 0.723 | 626.3 | overgrown courtyard | 0.09 | 101.6 | Dried Rat Hole |
| forbidden | `forbidden__arene__fusion__3` | 0.682 | 21.4 | dried back base | 0.85 | 166.9 | — |
| forbidden | `forbidden__arene__fusion__4` | 0.663 | 17.9 | overgrown base | 0.93 | 61.2 | — |
| live fire | `live_fire__arene__fusion__1` | 0.785 | 23.1 | nest | 0.45 | 44.4 | — |
| live fire | `live_fire__arene__fusion__2` | 0.781 | 9.8 | green building | 0.97 | 26.2 | — |
| live fire | `live_fire__arene__fusion__3` | 0.753 | 280.0 | tower | 0.30 | 101.7 | Tower |
| live fire | `live_fire__arene__fusion__4` | 0.704 | 34.1 | house | 0.66 | 128.6 | — |
| recharge | `recharge__arene__fusion__1` | 0.819 | 6.5 | sneaky | 0.91 | 8.8 | — |
| recharge | `recharge__arene__fusion__2` | 0.811 | 74.8 | batteries | 0.65 | 112.2 | Attic |
| recharge | `recharge__arene__fusion__3` | 0.785 | 92.0 | elevator | 0.44 | 70.1 | Whirlpool Dam |
| recharge | `recharge__arene__fusion__4` | 0.780 | 76.9 | hydro | 0.41 | 487.8 | Platform |
| recharge | `recharge__arene__fusion__5` | 0.751 | 9.2 | maintenance bay | 0.57 | 20.7 | Maintenance Bay |
| recharge | `recharge__arene__fusion__6` | 0.741 | 11.8 | long hall | 1.00 | 85.7 | Long Hall |
| recharge | `recharge__arene__fusion__7` | 0.735 | 14.4 | batteries | 0.82 | 112.2 | — |
| recharge | `recharge__arene__fusion__8` | 0.728 | 10.8 | control room | 0.96 | 60.8 | Control Room |
| streets | `streets__arene__fusion__1` | 0.853 | 25.4 | station tower 2 | 0.64 | 17.2 | — |
| streets | `streets__arene__fusion__2` | 0.805 | 7.6 | subway stairs | 0.83 | 9.4 | — |
| streets | `streets__arene__fusion__3` | 0.800 | 110.5 | old town | 0.60 | 79.3 | — |
| streets | `streets__arene__fusion__4` | 0.795 | 40.1 | subway bend | 0.34 | 36.5 | Subway Balcony |
| streets | `streets__arene__fusion__5` | 0.766 | 35.8 | commercial district | 0.46 | 81.8 | Cafe |
| streets | `streets__arene__fusion__6` | 0.724 | 42.5 | old town stairs | 0.38 | 27.0 | — |
| streets | `streets__arene__fusion__7` | 0.720 | 8.6 | main street | 0.44 | 39.4 | Main Street |
| streets | `streets__arene__fusion__8` | 0.706 | 11.0 | old town alley | 0.85 | 13.8 | — |

## 6. Couloirs — les positions de plus de 60 cellules : position ou salle ?

Le document de mesure v2 signale deux positions de 85 et 88 cellules retenues d'un bloc par la croissance au p90, et laisse au verdict de dire si c'est une position ou une salle. Lecture mecanique : SALLE ENTIERE si la zone dominante est couverte a plus de la moitie par la position ; COULOIR TRAVERSANT si au moins deux zones portent chacune un quart ou plus de la position (elle enjambe une frontiere) ; POSITION LARGE sinon (une partie d'une seule zone, ce que D1 appelle une position).

| Carte | Position | Cellules | Aire m2 | Zones traversees (part de la position / part de la zone) | Lecture |
|---|---|---|---|---|---|
| aquarius | `aquarius__arene__fusion__1` | 104 | 55.6 | hydro : 52 % de la position / 29 % de la zone ; top mid : 50 % de la position / 35 % de la zone ; blue courtyard : 18 % de la position / 11 % de la zone ; blue base : 16 % de la position / 4 % de la zone ; yellow courtyard : 12 % de la position / 4 % de la zone ; yellow utility : 11 % de la position / 8 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| aquarius | `aquarius__arene__fusion__2` | 256 | 97.6 | planters : 73 % de la position / 77 % de la zone ; pump : 48 % de la position / 81 % de la zone ; secret tunnel : 28 % de la position / 71 % de la zone ; top mid : 25 % de la position / 31 % de la zone ; blue courtyard : 20 % de la position / 17 % de la zone ; blue base : 19 % de la position / 9 % de la zone ; bottom mid : 11 % de la position / 96 % de la zone ; yellow courtyard : 10 % de la position / 6 % de la zone | SALLE ENTIERE : la zone dominante (planters) est couverte a 77 % |
| aquarius | `aquarius__arene__fusion__3` | 68 | 34.8 | blue base : 55 % de la position / 9 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| aquarius | `aquarius__arene__fusion__4` | 130 | 56.9 | yellow base : 74 % de la position / 39 % de la zone ; yellow courtyard : 20 % de la position / 7 % de la zone ; yellow utility : 17 % de la position / 12 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| bazaar | `bazaar__arene__fusion__1` | 884 | 500.0 | market : 39 % de la position / 77 % de la zone ; tower : 14 % de la position / 99 % de la zone ; antenna : 12 % de la position / 100 % de la zone ; tower basement : 12 % de la position / 95 % de la zone ; den : 10 % de la position / 100 % de la zone | SALLE ENTIERE : la zone dominante (market) est couverte a 77 % |
| bazaar | `bazaar__arene__fusion__2` | 274 | 191.4 | west courtyard : 80 % de la position / 70 % de la zone ; palm tree : 27 % de la position / 85 % de la zone ; cafe : 22 % de la position / 90 % de la zone ; market : 17 % de la position / 11 % de la zone ; west market bridge : 16 % de la position / 100 % de la zone ; west gate : 15 % de la position / 100 % de la zone | SALLE ENTIERE : la zone dominante (west courtyard) est couverte a 70 % |
| forbidden | `forbidden__arene__fusion__1` | 178 | 106.3 | dried bottom street : 36 % de la position / 90 % de la zone ; dried street : 29 % de la position / 92 % de la zone ; sad trombone : 24 % de la position / 23 % de la zone ; dried back nest : 15 % de la position / 22 % de la zone ; dried nest : 15 % de la position / 59 % de la zone ; overgrown pillars : 14 % de la position / 23 % de la zone ; overgrown hut : 12 % de la position / 63 % de la zone | SALLE ENTIERE : la zone dominante (dried bottom street) est couverte a 90 % |
| forbidden | `forbidden__arene__fusion__2` | 624 | 626.3 |  | POSITION LARGE : une partie d'une seule zone, pas la zone |
| live fire | `live_fire__arene__fusion__3` | 481 | 280.0 | tower : 30 % de la position / 92 % de la zone ; tunnel : 19 % de la position / 92 % de la zone ; hallway : 13 % de la position / 23 % de la zone ; brutes : 12 % de la position / 39 % de la zone ; canal : 11 % de la position / 39 % de la zone | SALLE ENTIERE : la zone dominante (tower) est couverte a 92 % |
| live fire | `live_fire__arene__fusion__4` | 72 | 34.1 | house : 66 % de la position / 18 % de la zone ; overlook : 25 % de la position / 23 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| recharge | `recharge__arene__fusion__2` | 119 | 74.8 | batteries : 65 % de la position / 43 % de la zone ; attic : 17 % de la position / 44 % de la zone ; maintenance elbow stairs : 11 % de la position / 43 % de la zone ; pit : 11 % de la position / 9 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| recharge | `recharge__arene__fusion__3` | 184 | 92.0 | elevator : 44 % de la position / 58 % de la zone ; whirlpool dam : 33 % de la position / 34 % de la zone ; whirlpool ledge : 19 % de la position / 91 % de la zone | SALLE ENTIERE : la zone dominante (elevator) est couverte a 58 % |
| recharge | `recharge__arene__fusion__4` | 165 | 76.9 | hydro : 41 % de la position / 7 % de la zone ; platform : 22 % de la position / 100 % de la zone ; pit : 20 % de la position / 16 % de la zone ; long hall : 13 % de la position / 12 % de la zone ; blue pipes : 11 % de la position / 34 % de la zone | POSITION LARGE : une partie d'une seule zone, pas la zone |
| streets | `streets__arene__fusion__3` | 233 | 110.5 | old town : 60 % de la position / 84 % de la zone ; old town bar : 21 % de la position / 53 % de la zone ; plaza stairs : 17 % de la position / 80 % de la zone | SALLE ENTIERE : la zone dominante (old town) est couverte a 84 % |
| streets | `streets__arene__fusion__4` | 80 | 40.1 | subway bend : 34 % de la position / 37 % de la zone ; subway : 28 % de la position / 19 % de la zone ; plaza : 23 % de la position / 11 % de la zone ; subway balcony : 21 % de la position / 100 % de la zone ; subway nest : 11 % de la position / 100 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| streets | `streets__arene__fusion__5` | 78 | 35.8 | commercial district : 46 % de la position / 20 % de la zone ; cafe : 45 % de la position / 96 % de la zone | COULOIR TRAVERSANT : a cheval sur 2 zones a 25 % ou plus |
| streets | `streets__arene__fusion__6` | 70 | 42.5 | old town stairs : 38 % de la position / 60 % de la zone ; station square : 32 % de la position / 16 % de la zone ; east alley : 12 % de la position / 4 % de la zone | SALLE ENTIERE : la zone dominante (old town stairs) est couverte a 60 % |

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

Chaque lignee a son document, genere par la meme commande avec son fichier de positions (depuis `apps/go-api`, `MAPPOWER_DATA_ROOT=<racine contenant data/>`) :

- **empirique v2** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/mesures_v2_2026-09-20/positions_v2.json MAPPOWER_VERDICT_SORTIE=VERDICT_V2_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`
- **geometrie seule** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/geometrie_2026-09-20/positions_geo.json MAPPOWER_VERDICT_SORTIE=VERDICT_GEO_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`
- **fusion** : `MAPPOWER_POSITIONS=.ai/V7.5/positions_de_force/fusion_2026-09-20/positions_fusion.json MAPPOWER_VERDICT_SORTIE=VERDICT_FUSION_2026-09-20.md go test -tags research ./cmd/mappower-build/ -run VerdictV2`

Le reglage de fusion (`fusion.ReglageFusionV1`) a ete choisi par balayage sur les cartes de calibrage seulement : `fusion_2026-09-20/_calibrage_fusion.md`. Le diagnostic v2 (section 7) ne se rejoue que sur un fichier empirique v2 ; le diagnostic geometrique ci-dessus, que sur le fichier geometrique.

## 10. Ce que ces chiffres disent

Aucune zone `forte` manquee : rien a expliquer.
