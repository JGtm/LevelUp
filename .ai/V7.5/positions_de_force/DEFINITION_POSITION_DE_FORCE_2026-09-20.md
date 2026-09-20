# Qu'est-ce qu'une position de force ? — définition, formule, inventaire des trous

> Écrit le 2026-09-20 à la demande de l'utilisateur, APRÈS la clôture de la recherche v1/v2
> (`PLAN_POSITIONS_DE_FORCE_2026-09-20.md`, quatre verdicts NO-GO). Le but est inverse de celui
> du chantier : partir de la DÉFINITION, indépendamment des données disponibles, en déduire une
> formule, puis dire pour chaque terme ce qu'on a, ce qu'on n'a pas, ce qu'il faudrait, et
> comment combler. Ce document ne mesure rien ; il fixe le socle d'un éventuel chantier suivant.

## 1. Définition

Une **position de force** est un emplacement de la carte depuis lequel un joueur, ou plus
souvent une équipe, **exerce un contrôle sur une surface bien plus grande que l'emplacement
lui-même, à un taux d'échange favorable, pendant un temps soutenu, et au service d'un enjeu
du mode joué**.

Chaque membre de phrase porte une composante, et aucune ne suffit seule :

| Membre | Composante | Ce que ça veut dire concrètement |
|---|---|---|
| « contrôle sur une surface plus grande » | **Domination visuelle** | On voit plusieurs voies, plusieurs pièces, les approches adverses. |
| « taux d'échange favorable » | **Asymétrie d'exposition** | On est vu de peu d'endroits, on a du couvert (hauteur, rebord, « head glitch »), l'adversaire arrive à découvert. |
| « pendant un temps soutenu » | **Tenabilité** | Peu d'accès, prévisibles ; une échappatoire ; on peut y être soutenu par un coéquipier (les pros parlent de *setup* : une position tient rarement seule). |
| « au service d'un enjeu du mode » | **Valeur stratégique** | Proximité ou vue sur les armes fortes, les bonus, l'objectif, et influence sur les réapparitions adverses. |
| « une équipe » | **Composition** | Une position de force est un ÉLÉMENT d'un dispositif d'équipe ; sa valeur dépend des autres positions tenues. |
| « du mode joué » | **Relativité au mode** | Le même lieu n'a pas la même valeur en Assassin, en Capture du drapeau ou en Roi de la colline ; et sa valeur change dans le temps (cycle des armes, score). |

Deux conséquences immédiates, qui expliquent les verdicts du chantier :

1. Une position de force n'est **ni un lieu où l'on tue beaucoup, ni un lieu où l'on gagne
   ses duels** : c'est un lieu d'où l'on CONTRÔLE. Le ratio de duel de la v1 mesurait la
   défense des bases, pas le contrôle ; l'exposition angulaire de la v2 était plus proche mais
   restait un effet, pas une cause.
2. Une position de force a une **cause géométrique** (elle est là parce que la carte est faite
   ainsi) et une **valeur contextuelle** (mode, armes, dispositif). La géométrie retrouve les
   causes ; elle ne peut pas deviner les valeurs. C'est exactement le partage observé : 10/13
   positions « hauteur / lignes de vue » retrouvées, 4/11 « arme / objectif ».

## 2. Une formule, en deux couches

Score d'un emplacement `p`, pour un mode `m` :

```
Force(p, m) = Potentiel(p) × Valeur(p, m) × Confirmation(p, m)
```

La forme multiplicative est voulue : un lieu sans potentiel géométrique n'est pas une
position de force quelle que soit sa valeur (un socle d'arme à découvert est un lieu où l'on
VA, pas un lieu que l'on TIENT) ; un lieu au fort potentiel mais sans valeur est un perchoir
inutile ; et un lieu que personne de compétent ne tient jamais est suspect.

### Couche 1 — Potentiel (propriété de la carte, indépendant des données de jeu)

```
Potentiel(p) = w_V·Vision(p) + w_X·Abri(p) + w_H·Hauteur(p) + w_A·Accès(p) + w_M·Échappatoire(p)
```

| Terme | Définition | Comment on le mesure en principe |
|---|---|---|
| **Vision** V | Étendue et DIVERSITÉ de ce que l'on voit à hauteur d'yeux (pas la surface brute : le nombre de voies distinctes couvertes). | Lancer de rayons depuis `p` vers le sol praticable, agrégé par voie d'approche. |
| **Abri** X | Inverse de l'exposition : fraction des directions d'où l'on peut être touché, à DEUX hauteurs (debout, accroupi derrière un rebord). | Rayons entrants, occlusion à 0,5 m et 1,2 m ; l'asymétrie debout/accroupi est le « head glitch ». |
| **Hauteur** H | Altitude relative au terrain que l'on domine. | Différence d'altitude avec les nœuds visibles (pas avec les voisins : un balcon domine la cour, pas le couloir derrière). |
| **Accès** A | Rareté et prévisibilité des chemins pour y arriver : nombre d'entrées, longueur des approches, et si ces approches sont VUES depuis `p`. | Graphe du sol praticable : degré, chemins disjoints, part des approches dans le champ de vision. |
| **Échappatoire** M | Coût pour rompre le contact : distance au premier point hors de vue, chute possible. | Plus court chemin vers un nœud non visible depuis les attaquants probables. |

### Couche 2 — Valeur (dépend du mode et du cycle du match)

```
Valeur(p, m) = w_R·Ressources(p, m) + w_O·Objectif(p, m) + w_S·Réapparitions(p) + w_C·Composition(p)
```

| Terme | Définition | Comment on le mesure en principe |
|---|---|---|
| **Ressources** R | Armes fortes et bonus contrôlés depuis `p` : vus, ou atteignables en peu de temps, pondérés par leur cycle de réapparition. | Distance de DÉPLACEMENT aux socles + visibilité du socle + période de réapparition. |
| **Objectif** O | Contrôle de l'objectif du mode : vue sur la colline, le drapeau, la zone, le trajet du porteur. | Visibilité des formes d'objectif et des trajets attendus. |
| **Réapparitions** S | Influence sur où l'adversaire réapparaît : tenir `p` pousse les réapparitions ennemies vers des zones prévisibles. | Règles de réapparition du jeu (zones, poids), ou mesure empirique des réapparitions en fonction des positions tenues. |
| **Composition** C | Compatibilité avec les autres positions d'un dispositif : tirs croisés, soutien mutuel, sans angle mort commun. | Intervisibilité entre positions candidates et couverture conjointe des approches. |

### Couche 3 — Confirmation (données de jeu, de joueurs COMPÉTENTS)

```
Confirmation(p, m) = f( tenue observée, avantage à la tenue, corrélation tenue → victoire )
```

Ce n'est pas un terme de plus : c'est l'oracle mesuré. Une position tenue longtemps par de
bonnes équipes, depuis laquelle elles tuent sans mourir, et dont la tenue précède la victoire,
est confirmée. Sans données de joueurs compétents, ce terme est muet, et c'est le mur
rencontré (nos joueurs suivis, sauf un, ne tiennent pas les positions pro).

## 3. Inventaire terme par terme — ce qu'on a, ce qui manque, comment combler

| Terme | On a | On n'a pas | Pour combler |
|---|---|---|---|
| **Vision** V | Prototype de rayons 2,5D sur un sol DÉRIVÉ du rendu (`powerpos/geo`, 6 cartes en 62 s). | La collision du jeu (le bloc de collision du `sbsp` n'est pas décodé) ; les voies d'approche comme unités (on compte des nœuds, pas des voies). | **Vérité terrain gratuite : chaque kill est une ligne de vue prouvée** entre la position du tueur et celle de la victime à l'instant (hors grenades, véhicules, mêlée) — 74 909 morts positionnées sur 951 films. Elles servent à VALIDER et CORRIGER les rayons géométriques (un rayon coupé là où un kill a eu lieu = occlusion fausse). Ensuite, découper les voies par le graphe du sol (entrées de pièce). |
| **Abri** X | Rayons symétriques (donc X ≈ 1 − V, corrélation 0,82-0,89 : le terme n'apporte rien tel quel). | L'asymétrie debout / accroupi ; les rebords partiels. | Rayons entrants à DEUX hauteurs contre les voxels déjà construits (0,25 m) ; c'est un ajout local au prototype, pas un nouveau chantier. |
| **Hauteur** H | Altitude relative au voisinage (rayon N m). | Altitude relative à ce que l'on VOIT. | Remplacer la moyenne de voisinage par la moyenne des nœuds visibles. Trivial une fois V juste. |
| **Accès** A | Rien de mesuré (le graphe du sol existe dans le prototype mais n'est pas exploité). Positions réellement courues (`map_positions_jouees.json`) et trajectoires de 173 rejeux. | Le maillage de navigation natif (aucune carte native n'en publie ; seules les cartes Forge ont un `navmesh.blob`). | Graphe du sol dérivé + trajectoires réelles pour élaguer les faux passages (un chemin jamais couru en 80 matchs n'existe probablement pas) ; degré, chemins disjoints, approches vues. |
| **Échappatoire** M | Une approximation dans le prototype. | Validation. | Vient avec V et A. |
| **Ressources** R | Socles d'armes et de bonus par carte (`map_weapon_pads.json`, socles de bonus), cycles de socle observés dans le film (`PadCycle`). | Les périodes de réapparition officielles par carte et par mode ; la distance de déplacement (on a le vol d'oiseau). | Les cycles mesurés dans le film donnent la période (déjà lus) ; la distance de déplacement vient du graphe (terme A). |
| **Objectif** O | Formes et positions des objectifs par carte et par variante (`map_objectives.json`, `mapvar`), états vivants des zones et trajets de drapeau dans le rejeu. | La liaison « position → objectif contrôlé » (visibilité des formes et des trajets). | Rayons vers les formes d'objectif ; trajets de porteur mesurés dans les rejeux CTF. |
| **Réapparitions** S | Détection des réapparitions dans le film (`spawn_detection`), positions de réapparition mesurées, points d'apparition Forge. | Les règles de réapparition du jeu (zones, poids, influence des positions tenues) — non décodées. | Mesure empirique : pour chaque réapparition observée, les positions des dix joueurs à cet instant sont dans le film ; on peut apprendre « qui tient quoi → l'ennemi réapparaît où » sans décoder les règles. Corpus nécessaire : des centaines de matchs par carte, tous niveaux confondus (les règles ne dépendent pas du niveau). |
| **Composition** C | Rien. | La notion même de dispositif d'équipe. | Vient après les positions candidates : intervisibilité entre candidats + couverture conjointe des approches. Puis confirmation par les dispositifs observés chez les bonnes équipes (voir ci-dessous). |
| **Confirmation** | 951 films de NOS joueurs (niveau modeste, sauf un), 173 rejeux complets ; oracle pro v2 : 28 positions attestées sur 8 cartes. | **Des données de joueurs compétents.** C'est LE trou. Et un vocabulaire de zones nommées sur les cartes Forge (Lattice, Argyle, Interference) pour pouvoir seulement NOMMER ce que les pros décrivent. | **L'API publique donne l'historique de match de n'importe quel gamertag, et un film se télécharge par identifiant de match** : le dépôt sait déjà lire le classement mondial et les stats de joueurs tiers (`snapshot-world-leaderboard`, `probe-world-stats`). Un corpus d'ÉLITE (joueurs Onyx du classement, parties classées récentes) se construit avec les outils existants : télécharger leurs films, décoder positions et kills, mesurer tenue et avantage. C'est la seule façon d'obtenir la couche 3 sans dépendre des guides. Coût : stockage des films, quota API, temps de décodage (déjà industrialisé : file de cuisson, ouvrier distant). |

## 4. Ce que ça change pour la méthode

1. **La géométrie est la couche première**, pas une voie de repli : elle mesure les causes.
   Mais elle doit être VALIDÉE par les lignes de vue prouvées des kills avant de servir — ce
   qui n'a pas été fait, et qui explique une partie des « retenues ailleurs » du verdict.
2. **La valeur est mode-dépendante** : un catalogue par carte ne suffit pas, il faut par carte
   ET par famille de mode (Assassin, objectif fixe, objectif mobile). Le calque du rejeu
   afficherait donc les positions DU MODE du match.
3. **La confirmation exige un corpus d'élite**, pas plus de nos matchs. Nos matchs servent à
   autre chose : les lignes de vue (validation de V), les réapparitions (terme S), les chemins
   réellement courus (terme A) — trois usages où le niveau des joueurs ne compte pas.
4. **L'oracle pro reste le juge**, et il faut d'abord combler son vocabulaire : sans zones
   nommées sur les cartes Forge du circuit, aucune position n'y est jugeable. C'est un chantier
   de données (callouts des `.mvar` Forge), pas de recherche.
5. **La sélection doit produire des LIEUX, pas des salles** : maxima locaux à rayon borné, aire
   plafonnée (les positions attestées font 5 à 30 m²). Le verdict de fusion l'a montré par
   l'absurde (pâtés de 500 m²).

## 5. Ordre de reprise proposé (si le sujet est rouvert)

1. Valider V par les kills (lignes de vue prouvées) ; corriger l'occlusion. Mesure : part des
   kills dont la ligne tueur → victime est déclarée visible par la géométrie (cible ≥ 95 %).
2. Ajouter X à deux hauteurs, H relatif au visible, A par le graphe élagué aux chemins courus.
3. Construire le corpus d'élite (films de joueurs Onyx) et mesurer tenue / avantage / victoire.
4. Compléter les zones nommées des cartes Forge du circuit (vocabulaire de l'oracle).
5. Sélection bornée, verdict par mode contre l'oracle v2, puis production selon le plan
   d'origine (étapes 3 à 7, inchangées).

## 6. Précisions de l'utilisateur (échange du 2026-09-20, après la première rédaction)

1. **La position de force est RELATIVE, pas absolue.** « Une power position peut être
   simplement une meilleure position que toutes les autres sur la map ; elle ne répond pas
   vraiment à tous nos critères, c'est juste la moins pire. » La formule doit donc produire un
   CLASSEMENT à l'intérieur d'une carte et d'un mode, et retenir les lieux qui se DÉTACHENT du
   reste de la carte (par un écart), jamais ceux qui dépassent une valeur fixe. Les réglages v1
   et v2 faisaient un quantile par carte mais y ajoutaient un plancher absolu et un plafond de
   huit positions : les deux contredisent cette définition. Une position peut aussi être « la
   meilleure pour une tâche » (surveiller un socle, tenir une colline) : la couche Valeur
   définit la tâche, le classement se fait parmi les candidats pour cette tâche.
2. **À niveau Onyx, peu de matchs suffisent.** Les bons joueurs convergent vers les mêmes
   lieux : le signal est la DENSITÉ D'OCCUPATION (temps passé), pas les frags, et la convergence
   elle-même se mesure (concentration des positions d'une équipe). Dix à vingt matchs Onyx par
   carte devraient suffire. Notre échec vient des joueurs, pas du nombre de matchs.
3. **Modes non classés et leurs cartes : pas de corpus Onyx.** Pistes, par ordre de solidité :
   (a) la géométrie, seul signal disponible partout — et les cartes Forge publient un vrai
   maillage de navigation, contrairement aux natives ; (b) le rang classé d'un joueur existe
   PAR IDENTITÉ ET PAR DATE même quand il joue en social (instantanés de CSR) — à vérifier sur
   pièces, mais ça pondérerait l'occupation des matchs sociaux par le niveau réel des joueurs
   présents ; (c) à défaut, la convergence des joueurs qui GAGNENT le match contre ceux qui le
   perdent, sur un corpus bien plus large que les 3-6 rejeux par carte mesurés en v1.

Socle corrigé : la géométrie CLASSE les lieux d'une carte, la valeur du mode dit POUR QUELLE
TÂCHE, la convergence des bons joueurs CONFIRME — avec peu de matchs quand ils sont bons, avec
plus de matchs pondérés par le niveau quand ils ne le sont pas.
