# A.1 : cause des sept verdicts suspects du score (mesure, aucun code de production)

Date : 2026-11-23. Décodeur à la tête de la version 1.6.0 (branche de mesures, aucune modification de production).
Joueurs en libellés neutres (joueur A, B...), films par leur identifiant court de 8 caractères, aucune donnée d'identité.
« vérifié » = commande rejouée ou pièce relue ; « déduit » = raisonné sans preuve, avec ce qui la donnerait.

## Hypothèses de départ et identification des films

Les sept films ne sont pas nommés dans les plans. Je les ai retrouvés ainsi (vérifié) :

- Les annexes du plan 1.5.0 (rapport B.3 : 23 Grandes batailles du build `HI_1_11_0` ; relevé B : 44 rejeux « complets mais faux ») donnent la population : après la correction B.3, « verdict complet 12 → 18 films sur 23 » ; il reste donc 5 Grandes batailles « incomplètes ». Les 23 films ont été re-décodés à la tête de la 1.6.0, avec le même chemin que la production (kill-feed décodé, faits de match exportés de la copie de l'API) : 18 complets, 5 incomplets. Les 5 incomplets n'ont ni rang non confirmé ni manque (`scoreboardMissing` = 0) : ce sont bien les cinq du plan.
- Le rapport B.3 nomme `97b34406` comme seul contre-exemple (« un message d'élimination de trop »). C'est la Grande bataille « complète à tort ».
- Le relevé B nomme `44e14331` (CTF, 5 joueurs dont le tableau est sous la somme, « le tableau cesse d'être lu avant la fin »). C'est le film dont le tableau cesse de monter.

Supposition qui change le résultat : « tableau du film » = ce que publie `scoreTimeline` (courbe construite depuis les enregistrements d'entité du film), pas le tableau d'affichage du jeu.

## Ce que le verdict compare (vérifié, `replay/document_score_actions.go:327`)

`complete` est vrai si : au moins une trame, aucune trame arrêtée, vue A lisible, aucun rang non confirmé, toutes les actions à points publiées, et aucun joueur dont le tableau dépasse la somme de ses points positifs publiés (`scoreboardMissing`). Un tableau sous la somme (`scoreboardBehind`) n'accuse pas la lecture (`replay/document_score_actions_tableau.go:48`).

## Famille 1 : cinq Grandes batailles « incomplètes » (5 films)

Films (vérifié, `tableau_familles.tsv`) : `1aed1d45`, `9d293550`, `e318a8b0`, `e3a2c1e9`, `f6c0d615`. Tous `HI_1_11_0`, 24 à 28 joueurs.

Faits :
- Le seul motif d'« incomplet » est `stoppedFrames` > 0 : 8, 2, 4, 5 et 1 trames arrêtées sur 11 700 à 21 100 trames. Rang non confirmé 0, manque 0, actions non publiées 0.
- Pour les 5 films et tous leurs joueurs (128 joueurs au total) : somme des points publiés = score personnel de l'API, et tableau ≤ somme (vérifié). Les trames arrêtées n'ont donc coûté aucun point mesurable.

Cause prouvée (vérifié par une sonde sur la marche des trames, reproduisant le chemin de production : 1, 8, 2, 4, 5 trames arrêtées, comme le document) : la vue A d'une trame s'arrête sur un message dont le genre n'a pas de lecteur de charge (`internal/grammar/vue_a_lecture.go:207-209`, `chargeDuGenre` renvoie nil, `internal/grammar/vue_a_charges.go:33`). Les messages qui suivent dans la même trame sont perdus. Le genre qui arrête chaque trame :

| Film | Trames arrêtées | Genre d'arrêt (nombre) |
|---|---|---|
| `1aed1d45` | 8 | 13 `motor_system_interruption` (8) |
| `9d293550` | 2 | 117 `EquipmentTranslocatorTeleportEffects` (2) |
| `e318a8b0` | 4 | 13 (4) |
| `e3a2c1e9` | 5 | 13 (3), 117 (2) |
| `f6c0d615` | 1 | 27 `controlToggleResponse` (1) |

Le genre 93 `activate_spartan_ability` est dans le même cas (pas dans ces 5 films, mais dans 6 autres films du parc échantillonné). Ces genres sont arrêtants sur tous les films ; ils se répètent rarement, d'où 1 à 8 arrêts par film.

Ce que le verdict a de trop strict : il ne sait pas si les messages perdus portaient des points. Ici la somme égale l'API : le verdict est conservateur à raison tant que le genre n'est pas lu, mais il dit « incomplet » pour des films exacts.

Généralisation (déduit, échantillon vérifié) : sur 43 films du parc ayant des trames arrêtées (version 1.4.1), relus à la tête de la 1.6.0, 35 films n'ont d'arrêts que sur les genres 13, 27, 93, 117 (1 à 20 arrêts) ; 8 films (builds 1.10 et 1.8) ont des centaines à des milliers d'arrêts sur dizaines de genres (table des genres illisible sur ces builds : autre sujet). Sur les 219 rejeux du parc à trames arrêtées, cela ferait environ 180 films de plus, à confirmer par un décodage du parc entier.

Correction prescrite : écrire les quatre lecteurs de charge des genres 13, 27, 93, 117 (adresses des lecteurs dans `internal/grammar/vue_a_genres_test.go` lignes 30, 44, 110, 134 ; branche dans `chargeDuGenre`, nouveau fichier `vue_a_charges_*.go` ; révision de grammaire). Elle règle les 5 films (déduit : plus d'arrêt, donc `complete` vrai si les messages nouvellement lus ne montrent pas de rang non confirmé). Risque : les messages d'après l'arrêt entrent dans les listes de tous les films concernés (points, médailles, dégâts) ; il faut mesurer rang non confirmé et sommes contre l'API sur le parc avant de figer, puis régénérer (règle : une régénération du parc après la 1.6.0). Largeur des charges : non établie ici (désassemblage du jeu, outil indisponible dans cette session) ; c'est le point à lever.

## Famille 2 : une Grande bataille « complète » à tort (`97b34406`, 1 film)

Faits (vérifié) : `HI_1_11_0`, BTB Fiesta Slayer, 25 joueurs comparés, `complete` = vrai. Un seul joueur diffère : joueur A, rang 6 du tableau. Tableau 2 225 = API 2 225 ; somme des points publiés 2 325. Écart : 100 points, une action « élimination » de trop. Joueur A : 20 éliminations de l'API, 20 éliminations publiées dans `eliminations[]`, 21 actions `killPersonalScore`. L'action de trop est à la trame 15 831 du rejeu (chrono du rejeu 522 s ; 568,30 s d'horloge du film), même victime (joueur du film n° 6) que l'action de 15 759, alors que cette victime n'est morte qu'aux trames 15 759 et 16 761.

Cause prouvée (vérifié, sonde sur les messages bruts) : le jeu a renvoyé la trame entière 71 trames (2,365 s) plus tard. À 568,30 s, 35 messages (rangs 29, 59, 60, 62, 63, 65, 68, 104, 107, 108, 125-128, 161, 328, ...) répètent un à un, avec le même numéro de séquence, la même charge et les mêmes destinataires, les 35 messages de la trame de 565,94 s. La détection des copies n'a qu'une fenêtre d'une seconde (`internal/grammar/evenements_de_jeu.go:16` et `:71`), donc ce renvoi passe pour de nouveaux messages. Pourquoi le verdict ne l'attrape pas : le tableau est sous la somme (« retard »), cas que le verdict tolère pour la pénalité de -100 (`replay/document_score_actions_tableau.go:48`).

Une fenêtre plus large partout est dangereuse (vérifié) : sur 43 autres films du parc, 3 ont des messages isolés à clé identique (rang, texte, destinataires, séquence) à 1,0 à 1,95 s d'écart (rangs 581, 606, 587, non liés au score). Aucun autre film n'a de trame entière renvoyée dans cette fenêtre.

Correction prescrite : dans `internal/grammar/evenements_de_jeu.go` (`Tete`), écarter une trame dont tous les messages (au moins deux) répètent, à clé identique, ceux d'une trame précédente de moins de 3 s ; la règle d'une seconde reste pour les messages isolés ; révision de grammaire ; test avec les deux trames de ce film. Films réglés : 1 (somme 2 325 → 2 225, déduit par soustraction de l'action ; vérifier en décodant). Risque : aucun changement mesuré sur les 65 autres films relus ; un renvoi légitime de trame entière dans les 3 s serait lui aussi écarté, ce que montre le film. Alternative non retenue : exiger dans le verdict que le retard soit égal aux points négatifs du joueur ; elle ne corrige pas la somme et ne dit pas la cause.

## Famille 3 : le tableau cesse de monter dans les dernières secondes (`44e14331`, 1 film)

Faits (vérifié, `tableau_familles.tsv`) : `HI_1_13_0`, CTF Arena, 8 joueurs, API 2 à 3 (861 s). Le tableau des joueurs s'arrête entre les trames 21 579 et 23 104 sur 25 264 (soit 762 s sur 834 s du rejeu), la courbe d'équipe aussi : l'équipe gagnante reste à 2 pour 3 à l'API. 5 joueurs sur 8 ont un tableau sous la somme des actions, qui égale l'API pour les 8 joueurs. Total des scores personnels : API 18 805, tableau 17 680, écart 1 125.

Cause prouvée (vérifié) : le film comporte deux manches. La manche 1 (courte, de 803 s à 859 s d'horloge du film) est écrite dans le film : 74 enregistrements, dont 49 de joueurs, avec la remise à zéro simultanée des huit joueurs à 803 s. Les enregistrements du statborg vont jusqu'à 859 s (la lecture n'est pas tronquée). Mais la manche 1 n'est pas retenue : `ResolveRounds` rend écrites [0 1], retenues [0]. Les deux critères d'admission échouent : la suite cohérente du score de mode n'a qu'un point (seuil 3, `internal/facts/objectives/statborg.go:163`), et la part d'enregistrements de joueurs est de 49 sur 565, soit 8,67 %, sous le seuil de 10 % (`statborg.go:264` et `:303`, `rounds_decision.go:97`). Tout ce que la manche 1 apporte est donc absent du tableau.

Preuve par les valeurs (vérifié) : les fins de manche 1 lues par joueur sont 575, 300, 100, 100, 50, 0, 0, 0 (somme 1 125) ; les écarts API moins tableau par joueur sont 575, 300, 100, 100, 50, 0, 0, 0 (somme 1 125). L'appariement joueur à joueur n'est pas prouvé (les emplacements changent de joueur d'une manche à l'autre) : les multi-ensembles et la somme concordent.

Correction prescrite : abaisser `statMinRoundRecordShare` de 10 à 7 (`internal/facts/objectives/statborg.go:264`), à documenter sur le corpus ; ou admettre une manche dont tous les joueurs présents remettent leur score à zéro dans la même seconde. Films réglés : 1 (déduit : tableau = API ; vérifier par le décodage). Risque : la seule mesure large est mon échantillon de 220 films du parc (déduit pour le reste) : les manches non retenues y ont une part d'au plus 2,9 % (hors 4 films, voir découvertes) et les manches réelles retenues de 12,7 % à 32,9 % ; 7 % tombe dans le vide entre 2,9 et 8,67. Aucun autre film de l'échantillon ne changerait.

## Si la cause n'était pas établie

Les trois causes sont établies à la précision demandée. Ce qui reste non prouvé : (1) la largeur de charge des genres 13, 27, 93, 117 ; (2) l'effet de chaque correction, déduit, à vérifier en l'écrivant (cette mesure n'écrit pas de code).

## Découvertes hors périmètre (non traitées)

- 4 films sur 220 ont une manche écrite non retenue avec 29 à 38 enregistrements de joueurs (8,7 % à 13,1 %) et les manches [0 2] (la manche 1 manque) : `c6db0991`, `a4083bd2`, `983feabb`, `3e06a064`. La contiguïté les écarte. À examiner : manche réelle perdue ou numérotation.
- Les builds `HI_1_10_0` et `HI_1_8_0` ont des centaines à des milliers de trames arrêtées sur de nombreux genres : table des genres non lisible sur ces builds.
- Le verdict ne distingue pas un tableau en retard à cause de la pénalité de -100 d'un tableau en retard à cause d'une action de trop. Sur les rejeux du parc dits « complets » (version 1.4.1), 90 joueurs sur 762 en retard ne le sont pas du montant exact de leurs points négatifs (22 films sur 423 ; surtout des films `HI_1_11_0` d'avant B.3, donc peu parlant).

## Reproduction (outillage jetable, hors dépôt)

Faits de match exportés de la copie de l'API ; films décodés un à un par la commande de cuisson du plan 1.5.0 recompilée à la tête de la 1.6.0, base de faits persistés vidée avant chaque film ; sondes de recherche (marche des trames, statborg, copies de messages) sous l'étiquette `research`, jamais versionnées. Sorties : `tableau_familles.tsv`, `hist_copies.txt` (copies par fenêtre, 23 films), `echant_resultat.txt` et `echant_hist.txt` (43 films à trames arrêtées), `ech_rounds_res.txt` (220 films, manches), `parc_stops.tsv`, `parc_behind.tsv`.
