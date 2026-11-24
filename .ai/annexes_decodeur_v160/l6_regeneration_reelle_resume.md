# Régénération réelle du parc des rejeux (décodeur v1.6.1, schéma 127), 2026-11-24

Résumé des pièces de l'exécution (journal d'avancement du pilote). Binaire construit le 2026-11-24 à 11:32 depuis
la branche de montée, sans espace de travail local. Une racine de données réelle, un seul film décodé à la fois,
serveur arrêté. Fenêtre : 11:35 à 13:17, soit 1 h 42 (prévision : environ 3 h ; précédent : 2 h 26 pour 1 455 films).

## État de départ (vérifié)

- Aucun processus de LevelUp, aucun verrou tenu. Profil de développement, 4 fils pour le décodeur.
- 1 455 rejeux, tous au schéma 125 ; dossier des rejeux 2 149 245 711 octets ; archive des films 1 597 dossiers (33,5 Go) ;
  faits de film 7,2 Go.
- Base partagée des matchs 483 143 680 octets, sans fichier WAL. Disque : 123 Go libres.
- Sauvegarde faite avant : la base partagée et le dossier des rejeux (2,5 Go), identiques à l'original.

## Passes (durées mesurées)

| Passe | Résultat | Durée |
|---|---|---|
| `backfill-replay --only-existing` | 1 455 films construits, 0 erreur de décodage, 0 échec de préparation, 0 mort mémoire ; 135 hors registre, 7 sans artefact | 1 h 11 (11:35 à 12:46) |
| `backfill-usage-summary` | 1 455 écrits, 590 sans artefact, 0 échec | 3 min 50 |
| `backfill-pad-tiers --force` | 1 450 écrits, 5 sans prise nommable, 0 échec | 3 min 47 |
| `backfill-killsource --workers 1` | 1 454 films sautés (révision inchangée), 6 décodés : 3 écrits (509 morts), 2 sans film, 1 écarté (profil de build absent) ; crédit : 1 462 matchs, 146 445 morts (précédent 146 249) | 2 min 33 |
| `backfill-vehicle-takes --force` | 1 455 écrits, 0 échec | 3 min 43 |
| `backfill-flag-grabs-net --force` | 190 écrits (précédent 186), 0 échec | 3 min 34 |
| `tactical-rasters --backfill` | 1 455 lus, 1 455 écrits, 0 échec ; seconde passe : 0 écrit, 12 s | 4 min 34 |
| `backfill-kickoff` | 2 045 matchs écrits, 0 échec d'écriture ; révision rangée `kickoff-2026-11-22` ; seconde simulation : 0 changement | 7,8 s |

## Bilan (vérifié)

- 1 455 rejeux au schéma 127 (avant : 1 455 au schéma 125). Une seconde simulation de `backfill-replay` : 0 à construire.
- Pic de mémoire d'un enfant de décodage : 1,64 Gio au plus, pour un plafond de 3 Gio.
- Poids du dossier des rejeux : 2 149 245 711 -> 3 003 833 741 octets (+39,8 %). Faits de film : 7,2 -> 7,5 Go.
  Archive des films inchangée.
- Coup d'envoi : source film 1 455, API 579, inconnue 11 ; 1 466 matchs changent de valeur (11 deviennent inconnus) ;
  écart nouveau moins ancien : médiane -355 ms, 1 421 matchs à une seconde près ; temps de jeu : 9 795 joueurs sur
  1 108 matchs, médiane -7 s.
- Seule la base partagée des matchs a été modifiée (même taille, aucun WAL) ; référentiels, PvE et social intacts.

## Avertissements relevés (non bloquants)

- Une ligne d'erreur de diagnostic : équipe de bots non lue sur un film du build `HI_1_8_0`.
- Un film écarté du journal des morts : profil de build `HI_1_5_1` absent de la table des profils.
- Couverture des ramassages : 1 591 prises sans famille, 755 emplacements non rattachés, 1 119 consommations à chaîne
  trouée (mesure incomplète signalée). 83 matchs sans carte à la référence pour les paliers de socle.
- La jauge « sans tête » du journal des morts n'est pas imprimée par la commande : non vérifiée.

## Question : un film sauté pour « révision inchangée » donnerait-il le même résultat ?

- Code : le saut de `backfill-killsource` se décide sur la révision du décodeur rangée dans les vues `_latest` du
  journal des morts, des vies et du placement, plus la révision « sans journal de mort ». Les révisions de la
  grammaire et du profil n'y entrent pas. Pas de mode de comparaison.
- Comparaison sur une copie (30 films : 5 de mode grand, 5 anciens de 2023-2024, 5 avec bots, 15 variés ; reprise
  forcée) : 4 008 morts, 0 erreur. Vue `_latest` avant et après identique ligne à ligne, compte par film identique
  30 sur 30. Écarts au niveau de la table brute : 15 lignes d'un film (nom affiché), 8 lignes de crédit seul
  (étiquette du chemin de lecture) : aucune arme, aucun instant, aucune ligne en plus ou en moins.
- Non comparé : vies (`match_lives`), placement, tirs (`weapon_shots`).
