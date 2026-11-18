# Coup d'envoi : horloge de manche du film contre premier mouvement (2026-11-18)

Lecture seule des 1 455 rejeux du poste (schéma 125), un fichier à la fois, script Node jetable.

Deux datations du coup d'envoi coexistent dans un rejeu :
- `t0FilmMs` : le premier mouvement des joueurs (`replay.DetectT0Film` du décodeur), ou rien quand le
  détecteur refuse ;
- `roundClock[]` : l'horloge de manche du jeu, lue chaque seconde (état 1 arrêtée, 6 descend, 19 monte).
  Départ calculé ici : pour la première lecture où l'horloge court, l'image où elle valait sa limite (ou
  zéro quand elle monte).

| Grandeur | Valeur |
|---|---|
| Rejeux lus | 1 455 |
| Rejeux sans horloge de manche | 0 |
| Rejeux où le départ de l'horloge ne se calcule pas | 0 |
| Rejeux sans `t0FilmMs` (détecteur de mouvement refusé) | 8, tous datés par l'horloge |
| Rejeux comparés | 1 447 |
| Écart « premier mouvement moins horloge », médiane | +0,36 s |
| Écart, centiles 5 et 95 | +0,30 s et +0,53 s |
| Écart, minimum et maximum | −21,38 s et +1,49 s |
| À une demi-seconde près | 1 329 |
| À une seconde près | 1 422 |
| Au-delà de 3 s | 22, tous négatifs (environ −16,8 s) : le premier mouvement est daté avant le départ de l'horloge |

Première lecture de l'horloge : état 1 (arrêtée, valeur = limite) dans 1 439 rejeux ; déjà en marche dans
16 (le départ se calcule alors par la valeur lue).

Lecture : l'horloge du jeu date le coup d'envoi de tous les rejeux, sans refus. Les joueurs bougent un tiers
de seconde après son départ. Dans 22 rejeux le détecteur de mouvement date le coup d'envoi environ 17 s trop
tôt, et dans 8 il ne date rien. Exemples à 17 s : `00ba2e1c`, `06a883f7`, `087474d9`, `390b1de5`,
`43880965` (21,4 s).

À contrôler en T.0 : modes à plusieurs manches (seule la première manche date le coup d'envoi) ; modes où
l'horloge monte (état 19, 1 rejeu en première lecture) ; accord avec l'horloge affichée par la page du rejeu.
