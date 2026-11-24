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

## Troisième intégration (2026-11-22)

Mesures sur les 20 films du corpus d'équivalence, un passage par fusion.

| Champ | Changement | Films touchés |
|---|---|---|
| `coverage.scoreActions.scoreboardCompared`, `scoreboardMissing`, `scoreboardBehind` | champs neufs : joueurs comparés au tableau du film (202), dont le score du tableau est au-dessus de la somme des points publiés (71) ou en dessous (26) | 20 sur 20 (écrits) ; 16 comparent des joueurs, 5 en ont au-dessus, 10 en dessous |
| `coverage.scoreActions.complete` | devient faux quand le tableau montre des points que le calque ne publie pas (`scoreboardMissing` > 0) | 1 sur 20 (`9f57c612`) |
| `reloads[]`, `coverage.reloads`, `layers.reloads` | calque neuf : un rechargement par (image, auteur, références), 9 026 publiés sur 12 708 messages lus (3 653 copies écartées) ; les films dont la vue A s'arrête tôt n'en lisent aucun | 16 sur 20 portent des lignes ; 20 la couverture |
| `stances[]` (genre `moveCommand`), `coverage.stances.control`, `coverage.stances.byKind.moveCommand`, `coverage.stances.intervals` | commande de déplacement tenue : 27 166 commandes publiées en 30 541 intervalles (`stances[]` de 35 836 à 66 377 lignes) ; 159 commandes sans joueur | 17 sur 20 portent des intervalles ; 20 la couverture |
| `coverage.stances.lives` | une vie de plus par film touché (une commande couvre une vie que la posture seule ne couvrait pas) | 11 sur 20 |
| `shots[]`, `coverage.shots` (`available`, `attached`, `byUnit`, `noSlot`, visée) | les tirs se lisent à tout rang d'une trame : tirs publiés 52 348 → 66 406 (+14 058), disponibles 54 006 → 68 418 | 16 sur 20 (les 4 films dont la vue A s'arrête tôt ne changent pas) |
| `shotStats[].shots`, `shotEvents`, `coverage`, `sufficient` | un numéro de tir en retard n'ajoute rien (un numéro 117 écrit après le 121 ajoutait 252 tirs) ; les tirs comptés changent (`01e1f945` : 22 683 → 2 973) ; joueurs aux données suffisantes 64 → 81 (`coverage.hits.sufficient`) | 16 sur 20 |
| `hits[]`, `damage[]` (`attacker`, `victim`, `w`, `effect`, `shield`, `body`), `coverage.hits` | l'auteur d'un dégât est le joueur de l'unité responsable ; l'arme d'une touche est celle du joueur à l'instant (par l'index pour un bot) ; touches publiées 18 632 → 19 266, sans auteur 800 → 36 | 7 sur 20 (nombre et auteur), 11 sur 20 (arme) |
| `projectiles[].shot`, `coverage.projectiles.shotLinked`, `shotAmbiguous` | davantage de tirs à lier : liés 1 567 → 2 052, ambigus 132 → 214 | 15 sur 20 |
| `coverage.vehicles` (`shots`, `shotsByUnit`, `shotsVehicleWeapon`, `shotsNoRide`, `shotsOnCarrier`), `coverage.seats.tirsParPlace` | suivent les tirs lus : tirs de véhicule 497 → 821, tirs par place 1 306 → 1 866 | 7 sur 20 (`shotsNoRide` : 9) |
| `vehicles[].carrier`, `coverage.vehicles.turretsByRead`, `turretParentUnknown`, `turretsOnCarrier` | le porteur d'une pièce montée est lu à sa création : 142 pièces sur 142 lues, aucune au parent inconnu ; 3 pièces gagnent un porteur (`turretsOnCarrier` 139 → 142) | champs écrits sur 20 ; 9 portent des pièces ; 1 gagne 3 porteurs |
| `coverage.fallbacks[]` (`repli_tourelle_porteur_voisin_de_slot`) | le repli du voisinage de slot ne se déclenche plus (139 → 0) | 9 sur 20 |
| `coverage.decoder.grammarRev` | `grammar-2026-11-22` → `grammar-2026-11-22.2` | 20 sur 20 |
| résumés d'usage (`UsageSummaryRev` `us7` → `us8`) | un objet lâché à la mort est crédité à la vie qui s'achève, pas au repreneur de la place ; aucun champ du document ne change | 0 sur 20 (résumés à reprendre par `backfill-usage-summary`) |

## Quatrième intégration (2026-11-22)

# Décodeur v1.5.0 : changements de sortie ajoutés par la quatrième intégration (schéma 126)

À ajouter au registre `.ai/V8.3/annexes_decodeur_1118/changements_de_sortie.md` (même format). Mesures sur les 20 films du corpus d'équivalence,
un passage par état (`7f36cca` troisième intégration, `f443fc8` avec l'arbre décalé, puis avec l'émission jetée par la table du mode, puis l'arbre final) ;
« sur 20 » = films touchés. Le schéma du document reste 126 ; `SchemaDesFaits` reste 30 ; la révision de grammaire reste `grammar-2026-11-22.2` ;
`objectives.Rev` passe à `objectives-2026-11-22.2`.

| Champ | Changement | Films touchés |
|---|---|---|
| `coverage.scoreActions.treeShift` | champ neuf : messages de score dont le rang a été traduit par un décalage de l'arbre des réponses déclaré pour le build du film (zéro hors des builds et intervalles déclarés) | 20 sur 20 (écrit) ; non nul sur 1 (`e5adf7b2`, HI_1_11_0 Grande bataille : 101) |
| `scoreActions[]`, `coverage.scoreActions` (`scoreMessages`, `scored`, `published`, `outsideTree`, `scoreboardBehind`) | sur HI_1_11_0 en Grande bataille, le rang d'un message désigne la réponse de la table un ou deux rangs plus loin (rangs 179 à 263 : un cran ; 447 à 460 : deux) ; les points restent ceux de la table. `e5adf7b2` : actions publiées 268 → 261, `outsideTree` 6 190 → 6 212, `scoreboardBehind` 9 → 1. Sur 23 films du parc (592 joueurs) : messages non confirmés 278 → 0, joueurs au-dessus de leurs points publiés 41 → 0, verdict `complete` vrai 12 → 18 films | 1 sur 20 |
| `coverage.fallbacks[]` (`repli_emission_hors_domaine_jetee`) | une emission est jetée en entier quand la table du mode nomme ses deux canaux comme compteurs et qu'un des deux est négatif ou au-delà de 250 (Drapeau : `24 A` et `24 B`, relevé 12 / 7 034) ; un canal que la table ne nomme pas (durée `22 B` en zones, `23 B` en VIP) n'est jamais jugé. Le compte monte : `01e1f945` 11 → 515, `1c4c63c2` 9 → 10, `50247b26` 22 → 44, `60ae07c4` 0 → 1 952, `696a9d7c` 0 → 38, `7344d24f` 0 → 36, `a349fea8` 38 → 104, `a521164d` 17 → 34, `d9781168` 0 → 1 838. La passe du drapeau lit la table du Drapeau sur tout film : le compte monte aussi sur des films d'un autre mode, sans autre sortie changée | 9 sur 20 |
| `coverage.objectives` (`available`, `refusedByRoster`) | baisse des événements du Drapeau que la passe du drapeau ne lit plus : `a349fea8` 100 → 76, `a521164d` 104 → 83 (films sans roster publié pour les objectifs) | 2 sur 20 |
| `coverage.decoder.objectivesRev`, `layers.eliminations`, `layers.flagCarries`, `layers.flagReturnZone`, `layers.identity`, `layers.objectives`, `layers.scoreTimeline` | `objectives-2026-11-22` → `objectives-2026-11-22.2` | 20 sur 20 |

Mesures de la règle de l'émission jetée (aucun compteur ne s'éloigne de l'API) :

- aucun compteur d'objectif, aucune courbe du tableau, aucun calque ne change sur les 20 films ; contre les colonnes de l'API là où elles existent : 0 compteur plus loin, 0 plus proche ;
- `696a9d7c` et `7344d24f` (zones) : `zone_defensive_kills` de chaque joueur égal à l'état d'avant, égal à l'API (une première forme de la règle, sans la table du mode, le dégradait pour 14 joueurs sur 14) ;
- `390b1de5` (parc, instrument de recherche, lecture seule) : le faux portage de 94,3 s du joueur de rang 7 disparaît (API : 0 s), 25 portages → 24 ; `70a1c6c6` et `6a711138` ne changent pas.

Étapes de l'équivalence touchées par état (digests, 20 films) : arbre décalé : `artifact` seul, sur les 20 films (champ `treeShift`, 14 octets ; `e5adf7b2` : 562 octets de moins) ;
émission jetée par la table du mode : `artifact` sur 9 films, `flag` sur 1 (`1c4c63c2`), `skull` sur 2 (`60ae07c4`, `d9781168`), `objectives` sur aucun ; révision `objectives-2026-11-22.2` : `artifact` sur les 20 films.
Les étapes `scoreTimeline` et `eliminations` ne changent à aucun état.

Sans changement de sortie, mesuré sur les 20 films (digests de l'équivalence identiques à l'état précédent) : les cinq corrections de garde-rails de LevelUp (conversion booléenne unique, tri total des
manches figées, fenêtres des éliminations en millisecondes, replis nommés, noms de repli orphelins retirés) et la correction du roster d'appoint des fixtures (test seul).

Surface publique de `decfilm` : `ObjectiveTypeVip` ajoutée ; `NomPrecisionParArmePasseSautee` et `NomDistancesDeToucheDesactivees` retirées (le registre ne les déclare plus).

## Après le gate du corpus (2026-11-22)

| Champ | Changement | Films touchés |
|---|---|---|
| `coverage.fallbacks` (`repli_emission_hors_domaine_jetee`) | REMPLACE la ligne de la quatrième intégration : une émission jetée ne se compte que sur un film que la table du Drapeau reconnaît ; hors Drapeau le compte revient à zéro ou sous sa valeur de v1.4.1 (les valeurs négatives vues par le seul verdict de mode ne se comptent plus) | 6 des 20 films de l'équivalence (1 952 → 0, 1 838 → 0, 515 → 11, 44 → 4, 38 → 0, 36 → 0) |
| `coverage.fallbacks` (`repli_identite_premier_occupant_du_siege`) | baisse : correction des trajets de véhicule | 7 témoins du gate (par exemple 283 → 192) |
| `coverage.objectives.available`, `refusedByRoster` | CAUSE CORRIGÉE : sur deux films de zones d'anciens builds, les candidats à canal négatif ne sont plus comptés (100 → 76, 104 → 83) ; `objectives[]` y est vide avant comme après (garde d'effectif) | 2 témoins |
| `score` (courbe d'équipe) | un point initial à zéro d'une équipe disparaît avec le faux enregistrement qui le portait | 2 témoins |
| `coverage.eliminations.perfectOrphans`, `selfHits`, `selfDamages` | champs de couverture neufs ou recomptés, non nommés plus haut | 2 à 5 témoins |

À mesurer sur le parc régénéré : chaque ligne ci-dessus (gate de l'étape L).
