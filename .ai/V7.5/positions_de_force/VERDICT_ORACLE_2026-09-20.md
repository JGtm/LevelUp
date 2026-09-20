# VERDICT — positions de force contre l'oracle pro (etape 2, 2026-09-20)

> **NO-GO** — 1 carte(s) sur les 4 exigees tiennent rappel >= 0.70 ET precision >= 0.60 hors Live Fire (contaminee) : bazaar.

> Genere par `go test -tags research ./cmd/mappower-build/ -run Verdict` (`verdict_oracle_research_test.go`). Positions jugees : cuisson du 2026-09-20T08:28:09Z, reglage FIGE `powerpos.ReglageV1` (disque 2.0 m, plancher 3 matchs, 40 engagements, quantile 0.90, plancher de score 0.65, composante >= 12 cellules).

Regles appliquees, toutes ecrites avant la mesure (plan item 2.1-2.4, relecture du pilote en §7 de l'oracle) :

- **retrouvee** : le polygone de la position couvre >= 30 % de l'aire de la zone attendue, OU son barycentre tombe dans la zone ;
- **rappel** = zones `forte` retrouvees / zones `forte` RESOLUES dans le vocabulaire du depot ;
- **precision** = positions calculees qui retrouvent au moins une zone de l'oracle (`forte` ou `faible`) / positions calculees ;
- les lignes `zone_en = ?` de l'oracle sont exclues des deux (sa regle de lecture) ;
- les zones `forte` de raison « arme » SEULE sont rapportees a part ;
- **Live Fire ne compte pas comme preuve** (decalage mesure de 9,88 m entre ses variantes) ;
- sur une carte a moins de 3 zones `forte`, un rappel de 0,50 se lit « indetermine », pas « echec » ;
- le GO exige en plus ZERO contre-exemple PUR colore.

Les aires sont mesurees par echantillonnage au pas de 0.2 m (surfaces concaves, trouees et en plusieurs morceaux du catalogue de zones).

## 1. Rappel, precision, temoin — carte par carte

| Carte | Cle oracle | `forte` (resolues/total) | Rappel | Rappel temoin | Rappel hors « arme » seule | Positions | Precision | Precision temoin | Lecture |
|---|---|---|---|---|---|---|---|---|---|
| aquarius | `ctf_aquarius` | 2/2 | 0.00 | 1.00 | 0.00 (2) | 5 | 1.00 | 0.80 | ECHOUE |
| bazaar | `ctf_bazaar` | 1/1 | 1.00 | 0.00 | 1.00 (1) | 2 | 1.00 | 0.50 | TIENT |
| empyrean | `d035fc3e-f298-4c14-9487-465be2e1dc1f` | 0/0 | 0.00 | 0.00 | 0.00 (0) | 0 | 0.00 | 0.00 | hors verdict (aucune zone `forte` resolue) |
| forbidden | `ctf_forbidden` | 2/2 | 0.00 | 0.00 | 0.00 (2) | 1 | 0.00 | 1.00 | ECHOUE |
| live fire | `sgh_interlock` | 3/3 | 0.33 | 0.00 | 1.00 (1) | 3 | 1.00 | 0.00 | rapporte, NE COMPTE PAS (contamination) |
| recharge | `sgh_blueprint` | 6/6 | 0.17 | 0.33 | 0.20 (5) | 4 | 1.00 | 0.75 | ECHOUE |
| solitude | `f1cc3b4e-471c-4ec5-b855-1db7d9e6ce42` | 0/0 | 0.00 | 0.00 | 0.00 (0) | 1 | 0.00 | 0.00 | hors verdict (aucune zone `forte` resolue) |
| streets | `sgh_streets` | 2/2 | 0.00 | 0.00 | 0.00 (1) | 4 | 0.50 | 0.50 | ECHOUE |

## 2. Temoin negatif — permutation circulaire des zones

La zone attendue en Z_i est reputee en Z_i+1 dans la liste TRIEE des zones nommees de la carte. Si le rappel ne s'effondre pas, l'appariement ne mesure que la densite des zones, pas leur identite — et un GO ne vaudrait rien.

| Carte | Rappel reel | Rappel temoin | Delta | Precision reelle | Precision temoin | Delta | Decalage applique aux zones `forte` |
|---|---|---|---|---|---|---|---|
| aquarius | 0.00 | 1.00 | +1.00 | 1.00 | 0.80 | -0.20 | Hydro -> Planters ; Top Mid -> Yellow Base |
| bazaar | 1.00 | 0.00 | -1.00 | 1.00 | 0.50 | -0.50 | Market -> Palm Tree |
| forbidden | 0.00 | 0.00 | +0.00 | 0.00 | 1.00 | +1.00 | Dried Rat Hole -> Dried Rat Tunnel ; Overgrown Rat Hole -> Overgrown Rat Tunnel |
| live fire | 0.33 | 0.00 | -0.33 | 1.00 | 0.00 | -1.00 | Hallway -> House ; Platform -> Storage ; Tower -> Tunnel |
| recharge | 0.17 | 0.33 | +0.17 | 1.00 | 0.75 | -0.25 | Attic -> Batteries ; Hydro -> Long Hall ; Maintenance Bay -> Maintenance Elbow Stairs ; Orange Pipes -> Overhang ; Pit -> Platform ; Whirlpool Dam -> Whirlpool Ledge |
| streets | 0.00 | 0.00 | +0.00 | 0.50 | 0.50 | +0.00 | Main Street -> Main Street Alley ; Subway Balcony -> Subway Bend |
| **moyenne** | **0.25** | **0.22** | **-0.03** | **0.75** | **0.59** | **-0.16** | |

**Reserve sur la force de ce temoin, a lire dans la derniere colonne.** La liste des zones est TRIEE PAR NOM, et le vocabulaire des cartes est ainsi fait que le voisin alphabetique est souvent le voisin GEOGRAPHIQUE (`Dried Rat Hole` -> `Dried Rat Tunnel`, `Whirlpool Dam` -> `Whirlpool Ledge`, `Subway Balcony` -> `Subway Bend`, `Main Street` -> `Main Street Alley`). Le decalage deplace donc souvent l'attente de quelques metres seulement : le temoin est PLUS FAIBLE que ne le laisse croire son principe. Il reste concluant dans ce sens-ci — le rappel reel (0.25) n'est pas meilleur que le rappel temoin (0.22). L'appariement ne porte quasiment aucun signal, ce qui est le constat du NO-GO et non une objection a lui.

## 3. Contre-exemples (§3.1 de l'oracle)

Une position calculee est un contre-exemple quand son BARYCENTRE tombe dans une zone que les guides deconseillent. Deux colonnes, parce que la §4 de l'oracle (« desaccords entre sources ») nomme des zones qui sont a la fois position et piege dans le MEME article : les exiger a zero demanderait a l'algorithme de trancher un desaccord que l'oracle n'a pas tranche. **Le GO porte sur la colonne PURS.**

| Carte | Pieges PURS colores | Pieges aussi decrits comme positions |
|---|---|---|
| aquarius | — | aquarius__arene__1 dans Blue Base ; aquarius__arene__2 dans Yellow Base ; aquarius__arene__3 dans Yellow Base ; aquarius__arene__4 dans Blue Base ; aquarius__arene__5 dans Blue Base |
| bazaar | — | bazaar__arene__1 dans Market ; bazaar__arene__2 dans Market |
| empyrean | — | — |
| forbidden | — | — |
| live fire | live_fire__arene__1 dans Canal | — |
| recharge | recharge__arene__4 dans Storage | recharge__arene__4 dans Overhang |
| solitude | — | — |
| streets | — | — |

**Total des pieges PURS colores : 2** (dont 1 hors Live Fire — c'est ce compte-la qui engage le GO).

## 4. Faux positifs et faux negatifs, nommes

| Carte | Faux negatifs (`forte` manquees) | Faux positifs (positions sans zone de l'oracle) | `forte` NON RESOLUES au vocabulaire |
|---|---|---|---|
| aquarius | Hydro (arme + lignes de vue) ; Top Mid (hauteur + lignes de vue + arme) | — | — |
| bazaar | — | — | — |
| empyrean | — | — | — |
| forbidden | Dried Rat Hole (objectif + acces) ; Overgrown Rat Hole (objectif + acces) | forbidden__arene__1 (zone dominante : dried platform) | — |
| live fire | Hallway (arme) ; Platform (arme) | — | — |
| recharge | Hydro (arme + objectif + spawn) ; Maintenance Bay (hauteur + lignes de vue + acces) ; Orange Pipes (acces + arme) ; Pit (arme) ; Whirlpool Dam (hauteur + lignes de vue + arme) | — | — |
| solitude | — | solitude__arene__1 (zone dominante : ledge) | — |
| streets | Main Street (arme) ; Subway Balcony (lignes de vue + arme) | streets__arene__2 (zone dominante : station square) ; streets__arene__3 (zone dominante : subway stairs) | — |

Une zone `forte` NON RESOLUE est absente du catalogue de zones du depot pour cette carte : l'algorithme ne peut ni la trouver ni la manquer. Elle est hors du denominateur du rappel, et signalee ici pour que le defaut reste visible.

## 5. Chaque position calculee et sa zone nommee dominante

« Part » = fraction de la position couverte par la zone dominante. « Aire zone » = aire de la zone dominante (union des zones du meme libelle sur les cartes en miroir).

| Carte | Position | Score moyen | Aire pos. m2 | Zone dominante | Part | Aire zone m2 | Appariee a |
|---|---|---|---|---|---|---|---|
| aquarius | `aquarius__arene__1` | 0.714 | 11.2 | planters | 0.95 | 114.6 | Blue Base |
| aquarius | `aquarius__arene__2` | 0.712 | 13.6 | yellow base | 0.98 | 133.0 | Yellow Base |
| aquarius | `aquarius__arene__3` | 0.708 | 7.6 | yellow base | 0.59 | 133.0 | Yellow Base |
| aquarius | `aquarius__arene__4` | 0.703 | 11.3 | blue courtyard | 0.90 | 230.8 | Blue Base |
| aquarius | `aquarius__arene__5` | 0.702 | 9.3 | blue base | 0.94 | 218.4 | Blue Base |
| bazaar | `bazaar__arene__1` | 0.729 | 20.6 | market | 0.91 | 436.0 | East Market Bridge |
| bazaar | `bazaar__arene__2` | 0.706 | 17.4 | market | 0.96 | 436.0 | West Market Bridge |
| forbidden | `forbidden__arene__1` | 0.697 | 8.0 | dried platform | 0.94 | 31.0 | — |
| live fire | `live_fire__arene__1` | 0.702 | 31.1 | canal | 0.49 | 117.4 | Tower |
| live fire | `live_fire__arene__2` | 0.686 | 20.0 | tower | 0.61 | 101.7 | Tower |
| live fire | `live_fire__arene__3` | 0.678 | 20.9 | overlook | 0.93 | 36.8 | Overlook |
| recharge | `recharge__arene__1` | 0.725 | 19.6 | platform | 0.70 | 17.0 | Platform |
| recharge | `recharge__arene__2` | 0.724 | 17.1 | control room | 0.67 | 60.8 | Control Room |
| recharge | `recharge__arene__3` | 0.715 | 22.5 | batteries | 0.53 | 112.2 | Attic |
| recharge | `recharge__arene__4` | 0.692 | 6.0 | storage | 1.00 | 77.6 | Overhang |
| solitude | `solitude__arene__1` | 0.781 | 10.0 | ledge | 1.00 | 32.5 | — |
| streets | `streets__arene__1` | 0.734 | 17.7 | plaza stairs | 0.60 | 23.5 | Old Town |
| streets | `streets__arene__2` | 0.726 | 17.3 | station square | 0.76 | 85.1 | — |
| streets | `streets__arene__3` | 0.716 | 9.8 | subway stairs | 0.41 | 9.4 | — |
| streets | `streets__arene__4` | 0.705 | 25.5 | commercial district | 0.67 | 81.8 | Cafe |

## 6. Diagnostic — pourquoi chaque zone `forte` manquee a echappe

Le score FIGE est rejoue sur le CSV par cellule de l'etape 1, et seules les cellules dont le CENTRE tombe dans la zone sont regardees. « Cellules zone » est l'emprise de la zone en cellules de 0,5 m ; « scorables » compte celles qui passent le plancher de matchs ET les 40 engagements de leur disque.

| Carte | Zone manquee | Cellules zone | Scorables | Score p50 | Score max | Seuil carte | Au-dessus | Plus grande composante | Cause |
|---|---|---|---|---|---|---|---|---|---|
| live fire | Hallway | 629 | 143 | 0.562 | 0.660 | 0.667 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.660 < 0.667 (p90 de la carte) |
| live fire | Platform | 50 | 42 | 0.558 | 0.661 | 0.667 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.661 < 0.667 (p90 de la carte) |
| recharge | Maintenance Bay | 82 | 60 | 0.593 | 0.700 | 0.680 | 2 | 2 | COMPOSANTE TROP PETITE : 2 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 2 (minimum 12) |
| recharge | Pit | 376 | 265 | 0.568 | 0.737 | 0.680 | 15 | 10 | COMPOSANTE TROP PETITE : 15 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 10 (minimum 12) |
| recharge | Hydro | 1951 | 222 | 0.581 | 0.655 | 0.680 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.655 < 0.680 (p90 de la carte) |
| recharge | Whirlpool Dam | 352 | 276 | 0.622 | 0.700 | 0.680 | 9 | 5 | COMPOSANTE TROP PETITE : 9 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 5 (minimum 12) |
| recharge | Orange Pipes | 137 | 89 | 0.553 | 0.678 | 0.680 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.678 < 0.680 (p90 de la carte) |
| streets | Main Street | 157 | 114 | 0.542 | 0.636 | 0.691 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.636 < 0.691 (p90 de la carte) |
| streets | Subway Balcony | 33 | 25 | 0.660 | 0.726 | 0.691 | 5 | 5 | COMPOSANTE TROP PETITE : 5 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 5 (minimum 12) |
| aquarius | Top Mid | 316 | 142 | 0.614 | 0.705 | 0.687 | 7 | 2 | COMPOSANTE TROP PETITE : 7 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 2 (minimum 12) |
| aquarius | Hydro | 409 | 170 | 0.607 | 0.705 | 0.687 | 14 | 10 | COMPOSANTE TROP PETITE : 14 cellules au-dessus du seuil, la plus grande composante 4-connexe en fait 10 (minimum 12) |
| forbidden | Dried Rat Hole | 66 | 6 | 0.436 | 0.558 | 0.678 | 0 | 0 | SCORE SOUS LE SEUIL : max 0.558 < 0.678 (p90 de la carte) |
| forbidden | Overgrown Rat Hole | 66 | 0 | 0.000 | 0.000 | 0.678 | 0 | 0 | NON SCORABLE : aucune cellule de la zone n'atteint 3 matchs distincts ET 40 engagements dans son disque |

## 7. Ce que ces chiffres disent

Sur **13 zones `forte` manquees**, la repartition des causes est :

- **COMPOSANTE TROP PETITE** : 6
- **NON SCORABLE** : 1
- **SCORE SOUS LE SEUIL** : 6

Le signal qui manque n'est donc PAS le meme partout, et c'est le fait saillant de l'etape : une part des zones attendues EST detectee par le score (des cellules y passent le seuil) mais ne forme pas d'amas de 12 cellules 4-connexes, et une autre part tombe a quelques millemes sous le p90 de sa carte. Ni l'une ni l'autre ne se corrige en etape 2 : toute retouche de seuil apres le verdict l'invalide (protocole du plan). L'arbitrage revient au pilote.
