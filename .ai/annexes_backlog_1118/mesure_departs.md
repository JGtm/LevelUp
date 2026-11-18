# Mesure des départs en cours de match (2026-11-18)

Lecture seule des 1 455 rejeux du poste (`data/cache/replays/halo_infinite/*.json.gz`, schéma 125), un
fichier à la fois, par un script Node jetable (non versionné).

Méthode : pour chaque entrée du tableau (`roster[]`) qui n'est pas un bot et dont la présence publiée se
ferme plus de 10 s avant la dernière image du document, on prend la dernière vie du joueur (piste de plus
grand `endFrame`) et on relève son `endReason`, puis la présence d'un compte à rebours de réapparition du
joueur (`respawnCountdowns[]`) commençant à 10 images au plus de la fin de cette vie.

| Grandeur | Valeur |
|---|---|
| Rejeux lus | 1 455 (0 erreur) |
| Rejeux sans présence publiée | 5 |
| Rejeux avec au moins un départ | 729 |
| Départs | 1 386 |
| Dernière vie finie par un corps supprimé (`deleted`) | 875 |
| Dernière vie finie par un lâcher de commande (`released`, une mort) | 503 |
| Dernière vie sans raison publiée | 8 |
| Départs avec un compte à rebours à la fin de la dernière vie | 464 (au plus les 503 morts) |
| Pistes finies par `deleted`, tout le parc | 1 284 |
| dont dernière vie d'un partant | 875 |
| dont autres | 409 (35 dans les 10 dernières secondes du document) |

Les quantiles de l'écart entre la fin de la dernière vie et la fermeture de la présence sont à refaire
(étape 0.3 a du plan) : le script les a calculés sur une liste qui contenait des valeurs non numériques
(vies sans `endFrame`). Bornes fiables : aucun écart négatif, 62 écarts au-delà de 30 s.

## Match `38900a5e`, ELOsk11

Images de 33 ms ; coup d'envoi du film à 33 415 ms ; horloge de manche : 5:00 écoulées vers l'image 9790.

| Image | Fait lu dans le rejeu |
|---|---|
| 9539 | Fin de vie par lâcher de commande, compte à rebours de 9539 à 9782 : une mort |
| 9783 | Nouvelle vie (3 points) |
| 9785 | Corps supprimé (`endReason: "deleted"`), aucun lâcher, aucun compte à rebours |
| 9937 | Fermeture de la présence d'ELOsk11 à la place 2 |
| 9938 | Le bot 343 Mickey prend la place 2 ; sa première vie commence à l'image 10188 |

Le document ne publie aucune mort neutre (`neutralDeaths`) pour ce joueur.
