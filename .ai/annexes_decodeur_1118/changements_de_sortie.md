# Décodeur v1.5.0 : registre des changements de sortie

Tenu à chaque intégration de piste (règle R-6 du plan). Il devient le corps des notes de version du décodeur.
Mesures sur les 20 films du corpus d'équivalence, sauf mention.

| Champ | Changement | Films touchés |
|---|---|---|
| `schemaVersion` | 125 vers 126 | tous |
| `t0FilmMs` et sa couverture | coup d'envoi = départ de l'horloge de la première manche ; le détecteur de premier mouvement est retiré | tous les films datés |
| `vehicleRides[].xuid` | un trajet porte le joueur du corps que la création place sur le siège ; un trajet sans vie qui le couvre perd son nom | 2 films du parc (4 trajets sur 12 015) |
| `scoreTimeline.players[].stats` | une émission dont un compteur d'éliminations dépasse les éliminations du joueur est écartée, comptée dans `coverage.score.statsGuarded` | 50 couples joueur-compteur du parc |
| `scoreTimeline` | un faux enregistrement du tableau (composant 0 à 27 illisible) est abandonné en entier ; les joueurs à 16 grenades disparaissent | 18 films du parc |
| `medalLabels` | champ déclaré, calque à la requête ; aucune sortie cuite ne change | aucun |
| `objectives[]` | huit noms d'action neufs : `flag_returners_killed`, `flag_kills_as_returner`, `flag_kills_as_carrier`, `zone_offensive_kills`, `zone_defensive_kills`, `vip_kills`, `vip_assists`, `vip_kills_as_vip` ; `coverage.objectives` change | 6 films gagnent des actions (125 au total), 10 changent de couverture ; 2 films passent de « aucune donnée » à « partiel » |
| `eliminations[]`, `coverage.eliminations`, `layers.eliminations` | calque neuf : une ligne par élimination, marques `headshot` et `perfect` à trois états (vrai, faux, inconnu) | 17 films portent des lignes, 3 n'ont pas de courbe d'éliminations |
| `scoreTimeline.players[].frozenRounds[]`, `coverage.score.frozenRounds` | valeurs figées par manche terminée (21 valeurs par joueur, dont tirs, touches, précision, dégâts) ; la dernière manche n'est jamais figée par le jeu | 5 films à manches (58 joueurs, 152 couples joueur-manche) |
| `coverage.score` (`roundsUnplayed`, `roundsWritten`, `points`) et compteurs de replis du tableau | conséquence de la lecture des composants 28 à 55 | 11 films, 4 films, 20 films |
| `identity.statborgSlots` | un faux enregistrement disparaît (une équipe en trop, une ligne `non_resolu`) | 2 films sans joueur au tableau |
| marque `headshot` | passe de faux à inconnu quand la courbe de tirs à la tête manque | 2 films |

À mesurer sur le parc régénéré : chaque ligne ci-dessus (gate de l'étape L).
