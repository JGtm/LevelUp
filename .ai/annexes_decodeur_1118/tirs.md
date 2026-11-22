# D.2 - Tirs, touches et dégâts : mesure à trois voix

Mesure sans code de production, faite sur 40 films : les 20 films de l'équivalence et 20 films choisis parmi
ceux où les armes se lisent mal. Dépôt public : aucun identifiant de joueur ici, les joueurs ne sont désignés
que par l'arme qu'ils tenaient le plus.

## 1. Résumé

**Verdict sur l'hypothèse (« le film n'écrit que les tirs qui ont atteint leur but ») : réfutée.**
Sur les événements de tir eux-mêmes, par arme :

- avec la lecture du témoin (la bibliothèque de Den, la plus complète : elle lit environ 18 % d'événements de
  plus que nous), le nombre d'événements de tir d'un joueur d'une arme à balles vaut 0,96 fois ses tirs tirés
  selon l'API (médiane ; fusil d'assaut MA40) et 2,24 fois ses tirs au but ; 78 % des joueurs sont à 10 % des tirs
  tirés, 2 % seulement à 10 % des tirs au but ; 88 % des joueurs sont plus proches des tirs tirés que des tirs au
  but ;
- pour les armes à énergie et à projectiles (Needler, Pulse Carbine, Disruptor, Commando, pistolet à plasma), le
  nombre d'événements est bas (0,36 à 0,52 fois les tirs tirés) mais n'est pas non plus égal aux tirs au but : le
  reste des tirs est écrit autrement, par des objets projectile. Événements plus objets projectile valent entre
  0,97 et 1,02 fois les tirs tirés (médiane) pour quatre de ces cinq armes, 1,21 pour le Pulse Carbine ;
- avec notre lecture, le rapport événements / tirs tirés est plus bas encore (médiane 0,49 sur tous les
  joueurs) pour une raison différente, qui n'a rien à voir avec l'hypothèse : notre lecture ne trouve que 85 %
  des événements que le témoin trouve (voir 5).

Conséquence pour la précision : pour une arme à balles, touches sur événements de tir est juste si le film est
bien lu (les événements sont les tirs tirés) ; pour une arme à énergie, elle ne se calcule pas par les seuls
événements : il faut y ajouter les objets projectile, sinon elle est surestimée d'un facteur 2 à 3.

**Les chiffres de départ sur les mêmes 262 joueurs (30 films que le témoin sait lire), part des joueurs à 10 %
près de l'API :**

| grandeur | notre lecture | témoin |
|---|---|---|
| tirs | 38,9 % (numérotation des tirs) | 38,5 % (événements + projectiles) ; 46,2 % si un tir écrit sous les deux formes n'est compté qu'une fois |
| touches | 42,0 % | 47,3 % |
| dégâts | 79,8 % | 83,6 % |

**Propositions pour D.3** (détaillées et chiffrées en 7) :

1. R1 : ajouter aux événements de tir les objets projectile (hors grenades) dont le créateur est le tireur, SAUF
   ceux nés à la même image que l'événement de tir du même tireur (identité par auteur et image, aucune fenêtre
   de temps). Plafond mesuré avec la lecture du témoin : part de joueurs à 10 % près des tirs de 38,5 % à 46,2 %
   (tous), et, pour les armes à énergie, de 0 à 13 % (événements seuls) à 28 à 39 % des joueurs.
2. R2 : estimer les tirs par le plus grand de deux comptes (numérotation ; événements plus projectiles non
   recoupés) : 50,4 % ; attention, deux armes à balles baissent de 6 points, ce qui touche le gate D : décision
   du user.
3. R3 : compter les projectiles dans la couverture du critère « données suffisantes » : joueurs suffisants de
   40,5 % à environ 50 %, au prix d'une part à 10 % près plus basse parmi eux.
4. R4 : pour les messages multiples d'une même touche, aucune règle à ajouter : la clé actuelle est déjà
   celle demandée.
5. Avant R1, une recherche bornée : notre grammaire lit 85 % des événements de tir que le témoin lit, et nos
   touches sont environ 18 % sous celles du témoin pour le MA40 sur 6 films de contrôle ; c'est plus gros que tout le reste.

## 2. Ce qui a été mesuré

- **Films.** 40 films : 20 de l'équivalence (dont trois sans aucun message de dégât lisible, builds anciens) et
  20 choisis parmi 1 455 par le nombre de joueurs dont l'arme dominante est mal lue (Needler, pistolet à plasma,
  Disruptor, Ravager, Shock Rifle, Pulse Carbine, Commando, Bandit, Avenger, Sniper). 37 films ont un bilan de
  tirs ; 404 joueurs humains sont joints à l'API.
- **Notre lecture.** Les documents de rejeu cuits avec le code de la piste D (tête intégrée plus D.1) : tirs =
  somme des sauts de la numérotation des tirs du joueur ; événements = événements de tir lus ; touches = messages
  de dégât d'arme comptés une fois par (auteur, unité blessée, image) ; dégâts = points retirés aux autres
  joueurs.
- **Le témoin.** `libfilmshell` de Den (copie locale en lecture seule, aucune récupération réseau), compilée
  avec une sonde de 80 lignes bâtie hors dépôt : par joueur et par arme, événements de tir (`Shot`), objets
  projectile hors grenades (ceux que sa précision compte), touches (dégâts d'arme, une fois par attaquant,
  victime et image) et dégâts. La bibliothèque refuse 10 des 40 films (en-tête de build inconnu : builds 1.4 à
  1.11, et deux films sans build lisible) ; il reste 30 films et 262 joueurs pour la comparaison à trois voix.
- **L'API.** Copie de la base de l'API : par joueur et par match, `shots_fired` (tirs tirés), `shots_hit` (tirs
  au but), `damage_dealt`. L'API ne donne AUCUN détail par arme dans cette base (la table des précisions par arme
  est vide) : tous les rapports par arme sont donc faits **sur l'arme dominante du joueur** (celle de la plupart
  de ses tirs lus), comme la mesure de départ. Le groupe d'une arme mélange donc un peu d'autres armes ; la ligne
  « mono-arme » (au moins 90 % des tirs sur l'arme) limite cet effet.
- **« À 10 % près »** : valeur API non nulle : écart absolu au plus 10 % de la valeur API ; valeur API nulle :
  la lecture vaut aussi 0.
- Les rapports (sections 4 à 6) ne gardent que les joueurs à au moins 30 tirs tirés selon l'API (et 10 tirs au
  but pour la section 4) : en dessous le rapport est du bruit.
- **D.1.** Vérification de non-changement : 52 films (les 40 plus 12 pris au hasard), documents de rejeu cuits
  avant et après la modification, comparés octet à octet après décompression : 52 identiques sur 52. Aucun des
  1 455 rosters du parc de départ ne porte deux humains successifs sur un même index ; le test unitaire couvre ce
  cas.

## 3. Chiffres par arme : les deux lectures contre l'API

### T1 - memes joueurs pour les deux lectures (films que le temoin sait lire), part des joueurs a 10 % de l API

| arme dominante (notre lecture) | joueurs | tirs (nous : numerotation) | tirs (temoin : evenements + projectiles) | tirs (temoin : evenements + projectiles non recoupes) | touches (nous) | touches (temoin) | degats (nous) | degats (temoin) |
|---|---|---|---|---|---|---|---|---|
| MA40 AR | 64 | 79,7 % | 68,8 % | 73,4 % | 73,4 % | 84,4 % | 81,3 % | 79,7 % |
| Needler | 56 | 8,9 % | 28,6 % | 32,1 % | 14,3 % | 19,6 % | 75,0 % | 87,5 % |
| Mk50 Sidekick | 34 | 82,4 % | 64,7 % | 67,6 % | 70,6 % | 73,5 % | 91,2 % | 91,2 % |
| BR75 | 7 | 28,6 % | 28,6 % | 42,9 % | 57,1 % | 57,1 % | 85,7 % | 71,4 % |
| MA5K Avenger | 8 | 37,5 % | 12,5 % | 25,0 % | 50,0 % | 50,0 % | 100,0 % | 100,0 % |
| M392 Bandit | 0 | - | - | - | - | - | - | - |
| Pulse Carbine | 13 | 23,1 % | 23,1 % | 38,5 % | 30,8 % | 30,8 % | 53,8 % | 61,5 % |
| Disruptor | 18 | 11,1 % | 22,2 % | 27,8 % | 27,8 % | 27,8 % | 88,9 % | 83,3 % |
| VK78 Commando | 11 | 18,2 % | 9,1 % | 36,4 % | 9,1 % | 9,1 % | 72,7 % | 72,7 % |
| Plasma Pistol | 13 | 7,7 % | 23,1 % | 30,8 % | 38,5 % | 46,2 % | 69,2 % | 76,9 % |
| autres armes ou sans tir | 38 | 13,2 % | 13,2 % | 26,3 % | 21,1 % | 26,3 % | 78,9 % | 89,5 % |
| TOUS | 262 | 38,9 % | 38,5 % | 46,2 % | 42,0 % | 47,3 % | 79,8 % | 83,6 % |

Lecture : sur les 262 joueurs que le témoin sait lire, nous faisons aussi bien que lui sur les tirs (38,9 % contre
38,5 %) tant qu'on ne retire pas ses doublons, mieux sur les armes à balles (numérotation : MA40 79,7 %, Sidekick
82,4 %), et bien moins sur les armes à énergie (Needler 8,9 % contre 28,6 % à 32,1 %). Sur les touches nous sommes
5 points sous lui, sur les dégâts 4 points.

### T1bis - notre lecture sur TOUS les joueurs des 37 films lus (builds anciens compris)

| arme dominante | joueurs | suffisants | tirs (nous) | touches (nous) | degats (nous) |
|---|---|---|---|---|---|
| MA40 AR | 96 | 42,7 % | 70,8 % | 54,2 % | 56,3 % |
| Needler | 62 | 14,5 % | 9,7 % | 14,5 % | 69,4 % |
| Mk50 Sidekick | 37 | 62,2 % | 81,1 % | 64,9 % | 83,8 % |
| BR75 | 34 | 5,9 % | 64,7 % | 11,8 % | 17,6 % |
| MA5K Avenger | 25 | 4,0 % | 52,0 % | 20,0 % | 40,0 % |
| M392 Bandit | 25 | 0,0 % | 88,0 % | 0,0 % | 0,0 % |
| Pulse Carbine | 24 | 20,8 % | 41,7 % | 20,8 % | 37,5 % |
| Disruptor | 21 | 38,1 % | 9,5 % | 28,6 % | 76,2 % |
| VK78 Commando | 18 | 11,1 % | 33,3 % | 16,7 % | 55,6 % |
| Plasma Pistol | 13 | 23,1 % | 7,7 % | 38,5 % | 69,2 % |
| autres armes ou sans tir | 49 | 24,5 % | 22,4 % | 18,4 % | 65,3 % |
| TOUS | 404 | 26,2 % | 47,3 % | 30,2 % | 54,5 % |

Sur tous les joueurs lus, les touches et dégâts de notre lecture sont bas pour BR75, Bandit, Avenger : ce sont pour
l'essentiel des films de builds anciens (1.4 à 1.11), dont les messages de dégât ne se lisent pas ou mal.

## 4. L'hypothèse du user : le film n'écrit-il que les tirs au but ?

Pour chaque joueur, le nombre d'événements de tir est comparé aux DEUX colonnes de l'API. Les rapports sont donnés
sous la forme 10e / 25e / médiane / 75e / 90e percentile. Si le film n'écrivait que les tirs au but, le rapport
événements / tirs au but vaudrait 1 et la part « à 10 % des tirs au but » serait haute. La précision de l'API
(tirs au but / tirs tirés, dernière colonne) vaut 0,23 à 0,49 selon l'arme : les deux hypothèses sont bien
séparées (le rapport événements / tirs au but vaut 1 sous l'une, 1 divisé par la précision sous l'autre).

### T2 - hypothese, lecture du temoin : evenements de tir (Shot) (joueurs a au moins 30 tirs tires et 10 tirs au but selon l API)

| arme dominante | joueurs | a 10 % des tirs tires | a 10 % des tirs au but | rapport evenements / tirs tires : p10 / p25 / med / p75 / p90 | rapport evenements / tirs au but : p10 / p25 / med / p75 / p90 | plus proche des tirs tires que des tirs au but | precision API (tirs au but / tirs tires), mediane |
|---|---|---|---|---|---|---|---|
| MA40 AR | 60 | 78,3 % | 1,7 % | 0,67 / 0,93 / 0,96 / 0,98 / 1,05 | 1,71 / 1,92 / 2,24 / 2,43 / 2,86 | 88,3 % | 0,42 |
| Needler | 56 | 0,0 % | 12,5 % | 0,26 / 0,31 / 0,46 / 0,62 / 0,78 | 0,97 / 1,23 / 1,58 / 2,43 / 2,92 | 17,9 % | 0,28 |
| Mk50 Sidekick | 33 | 72,7 % | 6,1 % | 0,39 / 0,74 / 1,00 / 1,01 / 1,02 | 1,49 / 1,73 / 1,98 / 2,22 / 2,66 | 78,8 % | 0,44 |
| BR75 | 7 | 28,6 % | 42,9 % | 0,30 / 0,32 / 0,33 / 0,98 / 0,98 | 0,98 / 1,04 / 1,56 / 1,77 / 1,77 | 28,6 % | 0,30 |
| MA5K Avenger | 7 | 0,0 % | 0,0 % | 0,62 / 0,67 / 0,76 / 0,82 / 0,82 | 1,97 / 2,00 / 2,03 / 2,82 / 2,82 | 71,4 % | 0,39 |
| M392 Bandit | 0 | - | - | - | - | - | - |
| Pulse Carbine | 11 | 0,0 % | 18,2 % | 0,18 / 0,35 / 0,52 / 0,60 / 0,74 | 1,07 / 1,52 / 1,98 / 2,68 / 3,18 | 18,2 % | 0,26 |
| Disruptor | 16 | 12,5 % | 18,8 % | 0,16 / 0,24 / 0,45 / 0,66 / 0,94 | 0,72 / 1,02 / 1,25 / 1,64 / 2,40 | 18,8 % | 0,34 |
| VK78 Commando | 11 | 9,1 % | 18,2 % | 0,15 / 0,25 / 0,40 / 0,73 / 0,79 | 0,92 / 1,15 / 1,86 / 3,03 / 3,21 | 36,4 % | 0,23 |
| Plasma Pistol | 13 | 0,0 % | 0,0 % | 0,18 / 0,27 / 0,36 / 0,59 / 0,79 | 0,51 / 1,16 / 2,35 / 2,70 / 3,95 | 23,1 % | 0,23 |
| autres armes ou sans tir | 31 | 6,5 % | 19,4 % | 0,16 / 0,21 / 0,32 / 0,56 / 0,72 | 0,72 / 1,08 / 1,31 / 1,64 / 1,95 | 9,7 % | 0,28 |
| TOUS | 245 | 31,8 % | 10,6 % | 0,20 / 0,35 / 0,63 / 0,96 / 1,00 | 0,94 / 1,28 / 1,88 / 2,33 / 2,86 | 45,3 % | 0,34 |

### T2 - hypothese, notre lecture : evenements de tir ecrits (shotEvents) (joueurs a au moins 30 tirs tires et 10 tirs au but selon l API)

| arme dominante | joueurs | a 10 % des tirs tires | a 10 % des tirs au but | rapport evenements / tirs tires : p10 / p25 / med / p75 / p90 | rapport evenements / tirs au but : p10 / p25 / med / p75 / p90 | plus proche des tirs tires que des tirs au but | precision API (tirs au but / tirs tires), mediane |
|---|---|---|---|---|---|---|---|
| MA40 AR | 92 | 7,6 % | 4,3 % | 0,26 / 0,49 / 0,79 / 0,84 / 0,89 | 0,77 / 1,49 / 1,79 / 2,10 / 2,27 | 60,9 % | 0,40 |
| Needler | 62 | 1,6 % | 17,7 % | 0,19 / 0,26 / 0,37 / 0,50 / 0,65 | 0,74 / 0,99 / 1,36 / 1,96 / 2,35 | 11,3 % | 0,28 |
| Mk50 Sidekick | 36 | 8,3 % | 2,8 % | 0,32 / 0,52 / 0,83 / 0,85 / 0,89 | 1,31 / 1,42 / 1,63 / 1,83 / 2,31 | 66,7 % | 0,44 |
| BR75 | 31 | 0,0 % | 12,9 % | 0,24 / 0,39 / 0,49 / 0,60 / 0,65 | 0,81 / 0,88 / 1,23 / 1,67 / 1,91 | 6,5 % | 0,39 |
| MA5K Avenger | 24 | 0,0 % | 8,3 % | 0,25 / 0,38 / 0,47 / 0,56 / 0,65 | 0,76 / 1,00 / 1,32 / 1,63 / 2,08 | 0,0 % | 0,36 |
| M392 Bandit | 25 | 40,0 % | 8,0 % | 0,36 / 0,76 / 0,88 / 0,91 / 0,92 | 1,09 / 1,26 / 1,66 / 1,88 / 2,01 | 68,0 % | 0,49 |
| Pulse Carbine | 22 | 4,5 % | 13,6 % | 0,15 / 0,30 / 0,45 / 0,58 / 0,66 | 0,87 / 1,06 / 1,81 / 2,22 / 2,41 | 13,6 % | 0,29 |
| Disruptor | 19 | 0,0 % | 26,3 % | 0,13 / 0,20 / 0,40 / 0,52 / 0,73 | 0,59 / 0,87 / 1,04 / 1,29 / 1,57 | 10,5 % | 0,34 |
| VK78 Commando | 18 | 0,0 % | 27,8 % | 0,13 / 0,22 / 0,49 / 0,55 / 0,63 | 0,80 / 1,00 / 1,17 / 1,76 / 2,34 | 5,6 % | 0,28 |
| Plasma Pistol | 13 | 0,0 % | 7,7 % | 0,17 / 0,22 / 0,30 / 0,48 / 0,62 | 0,47 / 0,98 / 2,03 / 2,06 / 3,10 | 15,4 % | 0,23 |
| autres armes ou sans tir | 41 | 0,0 % | 19,5 % | 0,12 / 0,16 / 0,28 / 0,46 / 0,62 | 0,51 / 0,73 / 1,08 / 1,33 / 1,55 | 4,9 % | 0,29 |
| TOUS | 383 | 5,7 % | 12,0 % | 0,17 / 0,30 / 0,49 / 0,78 / 0,86 | 0,73 / 1,00 / 1,48 / 1,91 / 2,26 | 30,3 % | 0,35 |

### T2bis - joueurs mono-arme (arme dominante au moins 90 % de leurs tirs ecrits), evenements ecrits par notre lecture

| arme | joueurs | a 10 % des tirs tires | a 10 % des tirs au but | ev / tires : p10 / p25 / med / p75 / p90 | ev / au but : p10 / p25 / med / p75 / p90 |
|---|---|---|---|---|---|
| MA40 AR | 23 | 8,7 % | 4,3 % | 0,61 / 0,78 / 0,82 / 0,85 / 0,89 | 1,59 / 1,71 / 2,01 / 2,21 / 2,41 |
| Needler | 0 | - | - | - | - |
| Mk50 Sidekick | 6 | 0,0 % | 0,0 % | 0,83 / 0,83 / 0,85 / 0,86 / 0,88 | 1,63 / 1,63 / 1,90 / 1,92 / 2,32 |
| BR75 | 2 | 0,0 % | 50,0 % | 0,42 / 0,42 / 0,65 / 0,65 / 0,65 | 1,08 / 1,08 / 1,91 / 1,91 / 1,91 |
| MA5K Avenger | 0 | - | - | - | - |
| M392 Bandit | 9 | 66,7 % | 0,0 % | 0,86 / 0,89 / 0,91 / 0,92 / 0,92 | 1,36 / 1,57 / 1,67 / 1,78 / 1,85 |
| Pulse Carbine | 0 | - | - | - | - |
| Disruptor | 0 | - | - | - | - |
| VK78 Commando | 0 | - | - | - | - |
| Plasma Pistol | 0 | - | - | - | - |

**Verdict, écrit quel que soit le résultat : hypothèse réfutée sur les événements de tir.**

- Armes à balles (MA40, Sidekick ; Bandit n'est lu que par nous) : le rapport événements / tirs tirés a pour médiane 0,96 et 1,00 avec le
  témoin (0,79, 0,83 et 0,88 avec notre lecture, qui perd des événements) ; événements / tirs au but 1,9 à 2,2, soit 1
  divisé par la précision. Les événements suivent les tirs tirés, pas les tirs au but : 78 % (MA40) et 73 %
  (Sidekick) des joueurs sont à 10 % des tirs tirés avec le témoin, 2 % et 6 % à 10 % des tirs au but.
- Armes à énergie et à projectiles : le nombre d'événements (0,36 à 0,52 fois les tirs tirés) est plus proche des
  tirs au but que celui des armes à balles, ce qui peut faire croire à l'hypothèse ; mais il n'est à 10 % des tirs au but que pour 0 à 19 % des
  joueurs. Les tirs manquants sont les objets projectile : avec eux le total vaut 0,97 à 1,02 fois les tirs tirés (médiane ; 1,21 pour le Pulse). Exemple du Needler : 0,46 événement par tir tiré,
  la moitié du total en projectiles (médiane 0,50), et 0,28 tir au but par tir tiré (précision).
- Un cas à surveiller : le BR75 (rafale de trois, 7 joueurs lus par le témoin) a 0,33 événement par tir tiré, ce
  qui est voisin de sa précision (0,30) et pourrait passer pour l'hypothèse. Ce n'est pas elle : 43 % seulement des
  joueurs sont à 10 % des tirs au but, la numérotation des tirs donne 1,00 fois les tirs tirés (médiane) et événements
  plus objets projectile 0,98 fois ; le film écrit donc une partie des tirs en événements (un par rafale) et le reste
  en objets projectile, comme pour les armes à énergie. Échantillon petit : à confirmer sur plus de films si le
  user le veut.
- Une mesure sur les compteurs du tableau de score (« tirs » et « tirs au but » du tableau) avait déjà réfuté
  l'hypothèse ; elle est réfutée aussi sur les événements.
- Ce que cela change pour la précision : voir le résumé (arme à balles : juste ; arme à énergie : ajouter les
  projectiles).

## 5. Événements de tir et objets projectile, et ce qui manque à notre lecture

Compte « événements + objets projectile » du témoin, contre les tirs tirés de l'API :

### T3 - evenements + projectiles (temoin), evenements seuls (temoin), numerotation (nous), contre les tirs tires de l API

| arme dominante | joueurs sonde | ev+proj a 10 % | ev + proj non recoupes a 10 % | ev seuls a 10 % | numerotation a 10 % | (ev+proj) / tires : p10 / p25 / med / p75 / p90 | part des projectiles dans ev+proj : p10 / ... / p90 |
|---|---|---|---|---|---|---|---|
| MA40 AR | 60 | 68,3 % | 71,7 % | 78,3 % | 78,3 % | 0,79 / 0,96 / 0,98 / 1,06 / 1,18 | 0,00 / 0,00 / 0,02 / 0,11 / 0,21 |
| Needler | 56 | 28,6 % | 32,1 % | 0,0 % | 8,9 % | 0,55 / 0,78 / 0,97 / 1,13 / 1,21 | 0,15 / 0,25 / 0,50 / 0,63 / 0,74 |
| Mk50 Sidekick | 33 | 66,7 % | 69,7 % | 72,7 % | 81,8 % | 0,70 / 1,00 / 1,02 / 1,08 / 1,19 | 0,00 / 0,01 / 0,06 / 0,17 / 0,55 |
| BR75 | 7 | 28,6 % | 42,9 % | 28,6 % | 28,6 % | 0,81 / 0,81 / 0,98 / 1,11 / 1,11 | 0,08 / 0,28 / 0,51 / 0,74 / 0,74 |
| MA5K Avenger | 7 | 14,3 % | 28,6 % | 0,0 % | 28,6 % | 0,79 / 0,89 / 0,89 / 1,12 / 1,12 | 0,08 / 0,08 / 0,12 / 0,32 / 0,32 |
| M392 Bandit | 0 | - | - | - | - | - | - |
| Pulse Carbine | 11 | 18,2 % | 27,3 % | 0,0 % | 9,1 % | 0,69 / 0,93 / 1,21 / 1,32 / 1,36 | 0,20 / 0,50 / 0,58 / 0,66 / 0,76 |
| Disruptor | 16 | 25,0 % | 25,0 % | 12,5 % | 12,5 % | 0,68 / 0,85 / 1,02 / 1,18 / 1,22 | 0,13 / 0,29 / 0,48 / 0,65 / 0,87 |
| VK78 Commando | 11 | 9,1 % | 36,4 % | 9,1 % | 18,2 % | 0,63 / 0,84 / 1,00 / 1,19 / 1,22 | 0,36 / 0,38 / 0,46 / 0,81 / 0,84 |
| Plasma Pistol | 13 | 23,1 % | 30,8 % | 0,0 % | 7,7 % | 0,41 / 0,52 / 0,98 / 1,11 / 1,21 | 0,26 / 0,46 / 0,53 / 0,60 / 0,65 |

Les deux formes se recoupent pour certaines armes seulement. Mesure : pour chaque objet projectile du témoin (24 426
sur 30 films), existe-t-il un événement de tir du même joueur à la même image du film (même tick) ?

### T4 - recoupement : projectiles du temoin dont un evenement de tir du meme joueur tombe au meme tick, a 1 tick, a 3 ticks (60 ticks par seconde)

Joueurs avec projectiles : 231 ; projectiles : 24426 ; evenements de tir : 38530.
| ecart | projectiles recoupes | part | meme mesure, projectiles decales de 37 ticks (temoin du hasard) | part |
|---|---|---|---|---|
| 0 tick | 4960 | 20,3 % | 41 | 0,2 % |
| 1 tick | 4996 | 20,5 % | 222 | 0,9 % |
| 3 ticks | 5055 | 20,7 % | 594 | 2,4 % |

Par arme (pour chaque événement de tir, y a-t-il un projectile du même joueur à ±1 tick ?) :

projectiles 24426 sans evenement a +-1 tick 20813 85.2 %
| arme (temoin) | evenements | avec un projectile du meme joueur a +-1 tick | part |
|---|---|---|---|
| MA40 Assault Rifle | 18867 | 3 | 0,0 % |
| Mk50 Sidekick | 7812 | 12 | 0,2 % |
| Needler | 4244 | 0 | 0,0 % |
| BR75 Battle Rifle | 1855 | 1 | 0,1 % |
| Pulse Carbine | 1372 | 0 | 0,0 % |
| Plasma Pistol | 1275 | 112 | 8,8 % |
| Disruptor | 1269 | 0 | 0,0 % |
| Shock Rifle | 1059 | 0 | 0,0 % |
| Ravager | 1014 | 1014 | 100,0 % |
| VK78 Commando Rifle | 988 | 0 | 0,0 % |
| MA5K Avenger | 969 | 0 | 0,0 % |
| Vestige Carbine | 842 | 1 | 0,1 % |
| S7 Sniper Rifle | 805 | 0 | 0,0 % |
| Cindershot | 690 | 690 | 100,0 % |
| Mangler | 568 | 568 | 100,0 % |
| Stalker Rifle | 563 | 0 | 0,0 % |
| M41 Rocket Launcher | 548 | 548 | 100,0 % |
| Heatwave | 365 | 0 | 0,0 % |
| CQS48 Bulldog | 361 | 0 | 0,0 % |
| MLRS-2 Hydra | 319 | 319 | 100,0 % |
| M392 Bandit | 197 | 0 | 0,0 % |
| Skewer | 188 | 188 | 100,0 % |
| Fuel Rod SPNKr | 106 | 106 | 100,0 % |
| Bandit Evo | 49 | 0 | 0,0 % |
| tag:0bb6976b | 39 | 0 | 0,0 % |

Deux familles, sans zone grise : les lance-roquettes et armes à charge (Ravager, Cindershot, Mangler, M41,
MLRS-2 Hydra, Skewer, SPNKr) écrivent le tir sous les deux formes à la même image, à 100 % ; les autres armes
(MA40, Sidekick, Needler, BR75, Pulse, Disruptor, Shock, Commando, Avenger, Sniper, Stalker, Heatwave, Bulldog,
Bandit) n'ont aucun recoupement. Seul le pistolet à plasma est mixte (8,8 %, ses tirs chargés). Au total 85,2 % des
objets projectile n'ont aucun événement de tir du même joueur à ±1 tick : ce sont des tirs que seuls les objets
portent. Le recoupement se fait exactement à la même image : à 1 tick ou 3 ticks d'écart il n'ajoute presque rien.

**Écart de lecture : nos événements contre ceux du témoin, sur les 30 films lus par les deux.**

| arme (premier mot) | evenements notre lecture (publies) | evenements du temoin (joueurs humains) | rapport | 
|---|---|---|---|
| MA40 | 16042 | 18867 | 0,85 |
| Mk50 | 6568 | 7812 | 0,84 |
| Needler | 3508 | 4244 | 0,83 |
| BR75 | 1626 | 1855 | 0,88 |
| Pulse | 1149 | 1372 | 0,84 |
| Plasma | 1020 | 1275 | 0,80 |
| Disruptor | 1047 | 1269 | 0,83 |
| Shock | 856 | 1059 | 0,81 |
| Ravager | 863 | 1014 | 0,85 |
| VK78 | 826 | 988 | 0,84 |
| MA5K | 827 | 969 | 0,85 |
| Vestige | 699 | 842 | 0,83 |
| S7 | 633 | 805 | 0,79 |
| Cindershot | 565 | 690 | 0,82 |
| Mangler | 484 | 568 | 0,85 |
| Stalker | 481 | 563 | 0,85 |
| M41 | 471 | 548 | 0,86 |
| Heatwave | 304 | 365 | 0,83 |
| CQS48 | 325 | 361 | 0,90 |
| MLRS-2 | 292 | 319 | 0,92 |
| M392 | 204 | 197 | 1,04 |
| Skewer | 147 | 188 | 0,78 |

### T6 - evenements de tir : notre lecture / temoin, et numerotation (nous) / tirs tires de l API

| arme dominante | joueurs sonde | ev nous / ev temoin : p10 / p25 / med / p75 / p90 | numerotation / tires : p10 / p25 / med / p75 / p90 |
|---|---|---|---|
| MA40 AR | 61 | 0,82 / 0,85 / 0,87 / 0,89 / 0,90 | 0,88 / 1,00 / 1,01 / 1,06 / 1,29 |
| Needler | 56 | 0,78 / 0,82 / 0,85 / 0,88 / 0,89 | 0,36 / 0,42 / 0,65 / 0,87 / 1,09 |
| Mk50 Sidekick | 33 | 0,81 / 0,82 / 0,85 / 0,87 / 0,90 | 0,62 / 0,96 / 1,00 / 1,02 / 1,02 |
| BR75 | 7 | 0,85 / 0,88 / 0,88 / 0,89 / 0,89 | 0,63 / 0,94 / 1,00 / 1,03 / 1,17 |
| MA5K Avenger | 7 | 0,74 / 0,76 / 0,77 / 0,85 / 0,85 | 0,71 / 0,90 / 1,00 / 1,06 / 1,63 |
| M392 Bandit | 0 | - | 0,97 / 0,99 / 1,01 / 1,02 / 1,04 |
| Pulse Carbine | 13 | 0,76 / 0,82 / 0,85 / 0,87 / 0,90 | 0,47 / 0,54 / 0,89 / 0,97 / 1,00 |
| Disruptor | 17 | 0,73 / 0,79 / 0,82 / 0,86 / 0,91 | 0,19 / 0,46 / 0,68 / 0,75 / 0,87 |
| VK78 Commando | 11 | 0,73 / 0,82 / 0,87 / 0,88 / 0,89 | 0,43 / 0,56 / 0,76 / 0,93 / 1,03 |
| Plasma Pistol | 13 | 0,78 / 0,80 / 0,84 / 0,90 / 0,92 | 0,34 / 0,41 / 0,61 / 0,80 / 0,87 |
| autres armes ou sans tir | 33 | 0,77 / 0,82 / 0,85 / 0,89 / 0,92 | 0,14 / 0,37 / 0,56 / 0,86 / 0,97 |
| TOUS | 251 | 0,78 / 0,82 / 0,85 / 0,88 / 0,91 | 0,38 / 0,64 / 0,96 / 1,01 / 1,09 |

Notre grammaire lit 81 à 89 % des événements de tir que le témoin lit, sur chacun des 30 films ; sur l'ensemble des
joueurs le rapport va de 0,78 (10e percentile) à 0,91 (90e), sans arme épargnée (de 0,78 à 0,92 par arme ; une seule exception, à 1,04, sur 197 événements de Bandit).
Contrôle direct sur un film : notre balayage rend 519 événements de tir, tous avec leur tireur, le témoin 605 pour
les seuls humains. C'est un déficit de lecture, pas un effet d'arme ni de joueur. Il explique que notre « événements
seuls » ne soit à 10 % des tirs tirés que pour 5,7 % des joueurs (témoin : 32,3 %). La numérotation des tirs (colonne
de droite) le contourne : médiane 1,01 sur MA40, 1,00 sur Sidekick et BR75, mais 0,65 sur Needler.

## 6. Touches et dégâts

Notre compte de touches (un message d'arme par auteur, unité blessée et image) contre le compte du témoin et contre
les tirs au but de l'API :

### T5 - touches contre les tirs au but de l API, part a 10 % pres, et doublons de messages

| arme dominante | joueurs | touches (nous) | touches (temoin) | touches nous / API : p10 / p25 / med / p75 / p90 | messages / touches (nous) : p10 / ... / p90 |
|---|---|---|---|---|---|
| MA40 AR | 92 | 55,4 % | 85,0 % | 0,13 / 0,65 / 0,99 / 1,03 / 1,10 | 1,00 / 1,00 / 1,01 / 1,16 / 1,39 |
| Needler | 62 | 14,5 % | 19,6 % | 0,00 / 0,60 / 0,86 / 1,10 / 1,33 | 1,01 / 1,05 / 1,15 / 1,29 / 1,51 |
| Mk50 Sidekick | 36 | 63,9 % | 72,7 % | 0,70 / 0,98 / 1,00 / 1,06 / 1,15 | 1,00 / 1,00 / 1,00 / 1,17 / 1,35 |
| BR75 | 31 | 12,9 % | 57,1 % | 0,18 / 0,30 / 0,45 / 0,87 / 1,01 | 1,02 / 1,05 / 1,12 / 1,44 / 1,53 |
| MA5K Avenger | 24 | 20,8 % | 57,1 % | 0,35 / 0,42 / 0,84 / 1,08 / 1,47 | 1,01 / 1,05 / 1,20 / 1,41 / 1,62 |
| M392 Bandit | 25 | 0,0 % | - | 0,00 / 0,00 / 0,00 / 0,04 / 0,47 | 1,00 / 1,00 / 1,23 / 1,24 / 1,31 |
| Pulse Carbine | 22 | 18,2 % | 27,3 % | 0,00 / 0,60 / 0,86 / 1,03 / 1,28 | 1,02 / 1,06 / 1,18 / 1,27 / 1,54 |
| Disruptor | 20 | 30,0 % | 29,4 % | 0,46 / 0,65 / 0,95 / 1,14 / 2,60 | 1,00 / 1,03 / 1,15 / 1,29 / 1,56 |
| VK78 Commando | 18 | 16,7 % | 9,1 % | 0,71 / 0,73 / 0,82 / 0,86 / 1,00 | 1,04 / 1,11 / 1,24 / 1,57 / 2,73 |
| Plasma Pistol | 13 | 38,5 % | 46,2 % | 0,74 / 0,75 / 0,88 / 0,96 / 0,99 | 1,02 / 1,12 / 1,17 / 1,24 / 1,46 |
| autres armes ou sans tir | 41 | 14,6 % | 22,6 % | 0,00 / 0,53 / 0,80 / 0,99 / 1,35 | 1,05 / 1,09 / 1,26 / 1,52 / 2,07 |
| TOUS | 384 | 30,2 % | 47,2 % | 0,00 / 0,45 / 0,86 / 1,02 / 1,19 | 1,00 / 1,01 / 1,11 / 1,29 / 1,57 |

Identité de la touche. Variantes de la clé de fusion, mesurées sur nos lignes de dégât publiées (une par message qui
retire des points ; toutes les armes lues, builds anciens compris) : part des joueurs à 10 % des tirs au but. Colonnes :
(A) image + victime, (B) A + effet, (D) B + points, (C) chaque message pris tel quel.

| arme dominante | joueurs | (A) trame + victime | (B) + effet | (D) + effet + points | (C) chaque message | mediane (A)/API | mediane (C)/API | 
|---|---|---|---|---|---|---|---|
| (aucune) | 99 | 1,0 % | 0,0 % | 0,0 % | 1,0 % | 0.27 | 0.33 |
| MA40 AR | 62 | 67,7 % | 67,7 % | 67,7 % | 59,7 % | 1.00 | 1.01 |
| Needler | 37 | 16,2 % | 18,9 % | 18,9 % | 32,4 % | 0.76 | 0.95 |
| Disruptor | 31 | 12,9 % | 25,8 % | 25,8 % | 22,6 % | 0.86 | 1.07 |
| Mk50 Sidekick | 26 | 80,8 % | 80,8 % | 80,8 % | 80,8 % | 0.98 | 1.00 |
| BR75 | 14 | 14,3 % | 14,3 % | 14,3 % | 28,6 % | 0.28 | 0.36 |
| MA5K Avenger | 12 | 25,0 % | 16,7 % | 16,7 % | 25,0 % | 0.45 | 0.52 |
| TOUS | 334 | 24,9 % | 26,3 % | 26,3 % | 29,0 % | 0.65 | 0.81 |

- Notre clé actuelle (auteur, unité blessée, image du film) est déjà la clé que demande le plan : la mesure ne montre
  pas de messages d'une même touche répartis sur plusieurs images du film. Sur le document, 1 269 touches sur 27 899
  ont le même auteur et la même victime sur la même image du document : l'image du document dure 33 ms et celle du
  film environ 17 ms, ce sont deux images du film.
- Ajouter l'effet de dégât à la clé (B) ne change rien pour MA40 et Sidekick et gagne 3 à 13 points sur Needler et
  Disruptor ; ne rien fusionner (C) gagne sur le Needler (32,4 %) et perd sur MA40 (59,7 %). Aucune clé unique ne
  gagne partout : la fusion n'est donc pas la cause principale des touches manquantes.
- La cause principale est un déficit de messages lus : sur 6 films de contrôle, MA40 1 018 touches contre 1 241 au
  témoin, Needler 241 contre 377, Sidekick 308 contre 391. Il touche toutes les armes, comme les événements de tir.
- Les lignes de dégât entre joueurs qui ne font aucune touche (12 %, 4 301 sur 35 410) portent surtout des effets
  de mêlée, de grenade et de zone ; aucune arme à balles n'y perd de tirs.
- Dégâts : le total par joueur est le meilleur des trois chiffres (79,8 % nous, 83,6 % témoin) ; il ne demande ni
  arme ni fusion et n'est pas à retoucher par D.3.

## 7. Propositions chiffrées pour D.3

**R1. Les objets projectile s'ajoutent aux événements de tir, sauf ceux que l'événement de tir du même tireur à la
même image écrit déjà.** Identité : même tireur, même image du film ; aucune fenêtre de temps (le recoupement est
exact au tick : 4 960 objets sur 5 055 recoupés à 3 ticks). Grenades exclues. Effet, sur 262 joueurs avec la lecture du
témoin : part à 10 % près des tirs de 33,2 % (événements seuls) et 38,5 % (événements + projectiles sans la règle) à
46,2 % ; par arme, de événements + projectiles sans règle à avec la règle : Needler 28,6 -> 32,1 %, Commando 9,1 ->
36,4 %, Plasma 23,1 -> 30,8 %, Pulse 23,1 -> 38,5 %, Disruptor 22,2 -> 27,8 %, BR75 28,6 -> 42,9 %, MA5K 12,5 ->
25,0 %. Coût : lire le créateur des objets projectile (la grammaire du décodeur n'en lit pas le tireur aujourd'hui ;
le témoin le lit sur l'opération de création de l'objet) : une lecture neuve, à borner par la règle R-3 du plan. Le
document publie déjà 10 243 projectiles hors grenades sur ces 30 films, sans tireur, contre 24 426 au témoin.

**R2. L'estimation des tirs d'un joueur est le plus grand de deux bornes basses : la numérotation, et événements
plus projectiles non recoupés.** Les deux comptent des tirs vus, jamais des tirs inventés ; la numérotation rate les
armes dont les numéros ne sont pas par tir (Needler : 0,65 fois les tirs tirés), les événements ratent ce qui n'est
pas écrit en événement. Mesure sur les 262 joueurs : numérotation seule 38,9 %, événements + projectiles non recoupés
46,2 %, plus grand des deux 50,4 % ; par arme, le plus grand des deux gagne partout sauf MA40 (79,7 % -> 73,4 %) et
Sidekick (82,4 % -> 76,5 %), qui baisseraient d'environ 6 points. **Attention au gate D** (« pour aucune arme la part ne
baisse ») : décision à prendre par le user avec ces chiffres (garder la numérotation seule pour ces armes serait une
règle par arme, donc une table ; ou accepter la baisse de 6 points contre +11,5 points au total). R1 sans R2
(numérotation inchangée) ne fait baisser aucune arme mais ne fait pas monter l'estimation des tirs.

**R3. Les projectiles non recoupés comptent dans la couverture du critère « données suffisantes ».** Couverture =
(événements + projectiles non recoupés) / tirs numérotés, au lieu de événements / tirs numérotés (seuil 0,7
inchangé). Mesure sur les 262 joueurs (couverture calculée avec nos événements et les projectiles du témoin) :
joueurs suffisants 40,5 % -> 50,4 % (53,8 % avec les événements du témoin) ; Needler 16,1 % -> 35,7 %, Commando
18,2 % -> 36,4 %, Plasma 23,1 % -> 38,5 %. Prix : parmi les joueurs jugés suffisants, la part à 10 % des tirs descend de
64,2 % à 53,0 % (ceux qui entrent sont moins bien lus que ceux qui y étaient) ; touches 62,3 % -> 59,1 % ; dégâts 89,6 %
-> 89,4 %. Le critère « données suffisantes » ne change que sur ces chiffres, montrés au user.

**R4. Messages multiples d'une même touche : aucune règle à ajouter.** La clé actuelle (auteur, unité blessée, image
du film) est déjà celle demandée ; la mesure ne la montre pas comme cause du déficit de touches (section 6). Ajouter
l'effet de dégât à la clé est la seule variante qui ne baisse aucune arme à fort effectif (MA40 67,7 -> 67,7 %,
Sidekick 80,8 -> 80,8 %, Needler 16,2 -> 18,9 %, Disruptor 12,9 -> 25,8 %), mais elle baisse la MA5K (25,0 -> 16,7 %, 12
joueurs) : à ne pas faire sans une mesure sur le parc.

**Cible chiffrée proposée pour le journal** (part des joueurs à 10 % près de l'API, sur l'échantillon de 262 joueurs ;
à refaire sur le parc au moment de D.3) : tirs 38,9 % -> 46 % au moins (plafond mesuré par le témoin avec R1, 46,2 %,
jusqu'à 50,4 % avec R2) ; joueurs suffisants 40,5 % -> 50 % au moins ; touches et dégâts inchangés (aucune règle sur
eux). Le déficit de lecture de 15 % des événements et 18 % des messages (sections 5 et 6) vaut une recherche à part,
bornée, avant R1 : sans elle, la part « événements seuls » de notre lecture ne dépasse pas 5,7 %.

## 8. Ce qui n'a pas pu être fait ou reste fragile

- Le témoin ne lit que 30 des 40 films (builds antérieurs à 1.12) : trois voix sur 262 joueurs, deux voix sur 142
  joueurs de plus.
- Trois films de l'équivalence n'ont aucun message de dégât lisible : sans bilan de tirs, hors de toute colonne.
- Pas de détail par arme côté API : toute mesure « par arme » est par arme dominante du joueur, jamais par arme
  réellement tenue ; les touches par arme du témoin sont comptées une fois par attaquant, victime et demi-tick
  (approximation de « une fois par image »).
- L'échantillon des 20 films supplémentaires est choisi sur les armes mal lues : les parts « tous » ne se
  généralisent pas au parc entier (59,7 % de tirs à 10 % de l'API au départ sur 14 052 joueurs).
- Les objets projectile du témoin portent un tireur lu à la création ; les nôtres n'en portent aucun : R1 est un
  plafond mesuré avec sa lecture, pas un résultat de notre grammaire.
- Les 15 % d'événements et 18 % de messages que notre lecture ne trouve pas : cause non recherchée (hors de cette
  mesure).
