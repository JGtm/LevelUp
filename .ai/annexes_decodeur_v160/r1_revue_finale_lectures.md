# Relecture v1.6.0, lentille « justesse des lectures »

Diff relu : `git diff v1.5.0...feat/v1.6.0 -- internal/ types/ chattername/` (hors tests), plus le codec des faits sous `replay/` pour la seule symétrie.
Mesures : tests temporaires en superposition (`go test -overlay`, aucun fichier écrit dans le dépôt), sur un échantillon de films locaux, un film à la fois.

## Constats recevables

### C1 (P1) : un projectile collé hérite de la naissance d'une vie antérieure

- Endroit : `internal/grammar/canal_des_projectiles_colles.go:120` (la table `naissances` n'est écrite que par un record de création lu jusqu'au bout) et `:144` (toute lecture de collage reprend l'entrée de la clé, sans contrôle).
- Condition : la clé est (slot, génération sur deux bits), elle revient donc après quatre vies du slot. Quand la création de la vie courante n'est pas parmi les records lus, l'entrée d'une vie plus ancienne de la même clé reste en place. Rien ne la retire : ni une suppression, ni une création désalignée, ni un nouveau collage.
- Conséquence : la lecture sort avec `HasBirth` vrai et l'instant, le type (`Tag`) et le créateur (`Creator`) d'un autre projectile. La publication en tire `attach.tag`, `attach.by` et `attach.attacker` : un collage attribué au mauvais lanceur et au mauvais type. Aucun compteur ne le signale (`NoBirth` ne compte que l'absence totale).
- Mesure (vérifié) : sur 266 films, 2 238 lectures de collage ; 96 portent une naissance vieille de plus de 6 s, dont beaucoup au-delà de 100 s (maximum relevé 363,7 s) pour un projectile dont la vie se compte en secondes. Aucun record désaligné sur l'échantillon : la création manquante est une création jamais vue par le canal.
- Limite de la mesure : la tranche de 6 à 20 s peut contenir des cas justes ; au-delà de 100 s il n'y a pas de doute.

### C2 (P1) : la charge du genre 13 ne ferme pas les trames, contrairement à ce que la chronique affirme

- Endroit : `internal/grammar/vue_a_charges_effets.go:26-33` (`chargeInterruptionDuMoteur` : R(1), puis R(7) ou R(2), puis R(1)), branchée sans condition à `internal/grammar/vue_a_charges.go:47`.
- Condition : une trame dont la vue A porte un message de genre 13, sur les builds 1.12 et 1.13 surtout.
- Conséquence mesurée (vérifié) : sur 200 films, 1 228 trames s'arrêtent sur le message qui suit immédiatement un genre 13 lu, contre 40 plus loin. Le genre d'arrêt y est réparti presque uniformément sur 64 à 74 (70 à 138 occurrences chacun, 1 112 des 1 228), ce que la répartition générale des genres d'arrêt ne montre pas. Un « genre » uniforme sur onze valeurs voisines est la signature de bits de données lus comme un sélecteur : la charge lue est plus courte que celle de l'écrivain dans la majorité des cas.
- Par build (vérifié, 399 films) : après un genre 13 lu, la trame s'arrête dans 72 % des cas en 1.12 (344 contre 131 terminées) et 69 % en 1.13 (1 498 contre 662), pour un taux d'arrêt général de 2,5 %.
- Ce qui en découle (déduit) : quand le bit lu après la charge vaut 0, la vue A est déclarée terminée alors qu'elle est décalée ; sa fin sert à décider le début de la liste de records sur les films où la vue A décide. Les trames « terminées » après un genre 13 ne sont donc pas une preuve. Pour le vérifier : comparer, sur ces trames, la fermeture au bit près de la trame entière.
- Rapport avec la règle écrite : `vue_a_charges_mesurees.go:24-27` dit qu'une disposition mesurée ne vaut que sur les builds où elle est prouvée, derrière le profil. Les largeurs du genre 13 sont dites « bornées par la mesure » et ne passent pas par cette porte. Avant le diff, la trame s'arrêtait sur le genre 13, arrêt compté et sans risque.

### C3 (P2) : la manche admise par remise à zéro simultanée n'est ni nommée comme repli ni comptée

- Endroit : `internal/facts/objectives/round_reset.go:51-61` (fenêtre d'une seconde, deux joueurs au moins, moitié des sièges), versé dans `materialRounds` à `internal/facts/objectives/statborg.go:259`.
- Condition : une manche courte que seuls ces trois seuils admettent (le film témoin cité dans l'en-tête du fichier).
- Conséquence : le document publie cette manche comme les autres ; rien dans la couverture ne dit qu'elle tient à une règle à seuils, et le registre des replis n'en a pas d'entrée. Les deux critères d'admission antérieurs sont dans le même cas : c'est une dette de même nature, que ce diff étend.

## Regardé de près, non retenu

- Trame renvoyée en entier (fenêtre de 3 s) : mesuré sur 399 films, 3 trames écartées au-delà d'une seconde (2, 8 et 13 messages, séquences consécutives), aucune clé écartée deux fois. Pas de faux positif périodique observé.
- Genres 27, 93 et 117 : lus aussi sur le build 1.10, hors de la liste des builds à charges mesurées. Les taux d'arrêt après lecture (6 à 36 %) ne permettent pas de conclure à un décalage ; aucune preuve de défaut.
- Table des noms de répliques tenue hors du profil du build : un build inconnu ne reçoit aucun nom (abstention), et un précédent existe dans la table des réponses de jeu.

Conditions vérifiées qui tiennent : 21

1. Les trois rejets rapides par mot de 64 bits (en-tête delta, préfixe de création, entrée de table) n'écartent que des positions que le test complet de chaque appelant refuse, sans effet de bord ; bornes de fin identiques à l'ancienne boucle.
2. Le pré-test du localisateur lit la tête par le code même de l'essai complet ; le repli à largeur libre rend le même début qu'avant.
3. La liste récupérée mise en mémoire par trame est remise à zéro à chaque trame ; elle n'est demandée que quand la signature stricte a déjà échoué sous le même monde.
4. Les trois références d'un message sont remises à zéro à chaque message : aucun reste d'un message précédent dans un geste, un rechargement ou une réplique.
5. La charge d'un geste est rangée par chaque lecture ; le recueil ne se fait qu'après une charge lue sans débordement.
6. Répliques et annonces : mêmes bits et mêmes polarités de garde que les sauts d'avant ; rien n'est rangé sans lecteur inscrit.
7. Genre 93 : seule la charge inhibée se lit, le reste arrête la lecture comme avant.
8. Accroupi et saut : l'absence du champ est testée avant le masque ; les trois natures de commande passent par le même pliage et ne se mêlent pas aux rafales de tir.
9. Pas des entrées : rompu à chaque trou, index borné, écart nul ignoré, égalité des fenêtres tranchée de façon stable.
10. Jeu d'armes désiré : le crochet est éteint pendant les essais du localisateur ; slot non lié écarté et compté ; doublons comptés.
11. Parent d'un collage et créateur : même masque et même base que l'occupation des véhicules ; un seul créateur admis, sinon abstention comptée.
12. Relecture de l'état de création du projectile : même décalage d'en-tête que le précédent de l'équipement, lecteur et observation séparés.
13. Handle du projectile d'un lancer : lu seulement quand l'en-tête est dans le paquet.
14. Apparence : position du bloc cohérente avec le corps d'enregistrement (bloc de 44 octets fixe derrière), build inconnu = abstention dite, compteur de paires borné, plage hors paquet refusée, paquet débordé ou incomplet refusé et compté.
15. Essais du monde : toute écriture de la table passe par le journal ; un retour arrière pendant un essai est lui aussi journalisé.
16. Filtres de positions en place : les suites compactées sont neuves, aucune n'est tenue par un mémo.
17. Pénalité de score : confirmée seulement par la montée du compteur dans la même émission et une baisse exacte ; jamais reportée d'un occupant au suivant ; sans pénalité, la suite est celle d'avant.
18. Coupure par occupant : instants triés, borne basse incluse, appliquée aux seuls évènements nommés ; le garde par les éliminations ne s'applique au composant 22 que pour un compteur VIP demandé.
19. Révisions : grammaire et objectifs montées ; aucune source ne change sous `profile`, `source` ni `killsource` (leurs empreintes bougent par le paquet `types` et par la valeur de la révision amont).
20. Codec des faits (répliques, gestes, armes en main, projectiles collés, apparence, commandes, rechargements, lancers) : écriture et relecture dans le même ordre, coûts minimaux par élément exacts.
21. Table des noms : indice borné à douze bits, doublon et ligne mal formée refusés au chargement, fins de ligne tenues par `.gitattributes`.

## Vérification des corrections

Arbre d'origine : tête `4d0e9b7`. Arbre corrigé : mêmes sources plus les modifications non commitées de la piste de correction. Mesures par tests en superposition hors dépôt, un film à la fois ; la carte de chaque film vient de la liste film -> carte de la piste de correction et du catalogue de bornes indiqué par le coordinateur.

### 1. Ma mesure d'origine était prise sans la carte : mes chiffres C1 et C2 étaient faux

Vérifié : mes deux sondes ouvraient le film sans carte (contexte nu). Sans les largeurs de la carte, la lecture des records et de la vue A est décalée en amont, et elle fabrique de faux messages de genre 13 et de faux collages.

C2 rejoué AVEC la carte sur le code d'origine (365 films ; forme : lu puis trame terminée / lu puis trame arrêtée / arrêt sur le genre / arrêt juste après) :

| Build | Films | Genre 13 | Genre 27 | Genre 93 | Genre 117 |
|---|---|---|---|---|---|
| 1.10 | 10 | 92 / 41 / 0 / 33 | 73 / 19 / 0 / 13 | 34 / 15 / 59 / 14 | 92 / 21 / 0 / 19 |
| 1.11 | 18 | 5 / 0 / 0 / 0 | 26 / 0 / 0 / 0 | 4 / 0 / 0 / 0 | 13 / 0 / 0 / 0 |
| 1.12 | 31 | 0 / 0 / 0 / 0 | 1 / 0 / 0 / 0 | 5 / 0 / 0 / 0 | 6 / 0 / 0 / 0 |
| 1.13 | 303 | 1 / 0 / 0 / 0 | 5 / 0 / 0 / 0 | 24 / 0 / 0 / 0 | 63 / 0 / 0 / 0 |

- Mes 72 % et 69 % d'arrêts après un genre 13 en 1.12 et 1.13 n'existent pas : presque aucun message de genre 13 sur ces builds, aucun arrêt. Le correcteur avait raison.
- Le défaut réel est sur le build 1.10, et il touche les quatre genres, pas le seul genre 13 : 31 % d'arrêts après un genre 13 lu (41 sur 133), 21 % après un 27, 31 % après un 93, 19 % après un 117, pour 2 % de trames arrêtées en général. Le correcteur mesure 23 % sur 26 films : même ordre de grandeur.

C1 rejoué AVEC la carte sur le code d'origine (246 films, 21 665 collages lus) : 1 collage au-delà de 6 s, aucun au-delà de 20 s ni de 100 s, maximum 7,3 s. Mes 96 cas et mes 363,7 s n'existent pas avec la carte. Le correcteur avait raison.

### 2. Ce que donne le code corrigé

- C1, mêmes 246 films : 21 477 collages publiés (188 répétitions d'une même vie ne sont plus publiées, comptées), 1 au-delà de 6 s, maximum 7,3 s, aucun au-delà de 100 s. Aucune naissance ne change entre les deux arbres sur l'échantillon.
- Mes trois cas au-delà de 100 s (358,5 s, 278,2 s, 121,1 s, sur trois films) : avec la carte ils n'existent dans aucun des deux arbres. C'étaient des collages fantômes, lus sur des records décalés (parents absurdes, 70 à 141 listes récupérées à largeur libre par film). Le code corrigé n'a donc rien à retirer ni à remplacer pour eux : « inchangé, car absent ».
- C2, mêmes 365 films : plus aucun arrêt juste après l'un des quatre genres, sur aucun build. En 1.10 les quatre genres arrêtent de nouveau la trame sur le genre lui-même (133, 92, 108 et 112 arrêts comptés) ; en 1.11 le genre 117 arrête (13) ; les lectures gardées en 1.11, 1.12 et 1.13 sont toutes dans une trame terminée.

### 3. Relecture de la correction

Ce qui doit être vrai, et ce qui tient (vérifié par lecture, sauf mention) :

- Suivi de la vie par slot : la passe reçoit tous les records de la trame dans l'ordre ; une suppression ou une création sous le slot, de tout type, finit la vie ; une création désalignée ouvre une vie sans naissance ; un delta de génération différente ouvre une vie sans naissance. La naissance n'est reprise que de la vie qui l'a posée. Tient. Le reste déclaré (suppression et création cachées par des trames illisibles, puis même génération) est exact et non observé.
- La correction C1 protège contre un cas que la mesure avec carte ne montre pas : elle est juste, sans effet mesurable sur l'échantillon.
- Charges 13, 27, 93, 117 derrière le profil, une ligne par build et par genre, build inconnu = arrêt compté. Tient ; la mesure ci-dessus le confirme.
- À faire avant de figer, déjà annoncé par le correcteur : monter les révisions de la grammaire et du profil (leurs sources changent, et la sortie des collages change).

Constat recevable sur la correction :

**V1 (P1) : la couverture des collages sans piste n'atteint jamais le document**
- Endroit : `replay/projectiles_colles.go:161-162` (création du bloc de couverture quand il est absent), appelé à `replay/build.go:99`, alors que `replay/build_pistes.go:329` a déjà recopié le pointeur, encore nul, dans le document à `replay/build.go:92`.
- Condition : un film dont les collages sont lus et dont aucune trajectoire de projectile n'est lue.
- Conséquence (vérifié, test en superposition sur l'arbre corrigé, dans l'ordre réel des passes) : `attachments[]` est publié, mais `coverage.projectiles` reste absent du document ; `noTracks`, `repeats` et tous les comptes des collages sont perdus. Le test ajouté par le correcteur regarde le champ interne de l'assemblage, pas le document, et ne voit donc pas le défaut. La correction annoncée (« collages lus sans piste ») ne fait que la moitié de ce qu'elle dit.

Aucun autre constat recevable sur la correction.
