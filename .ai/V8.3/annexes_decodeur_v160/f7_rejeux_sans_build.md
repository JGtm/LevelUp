# F.7 — cinq rejeux sans build : pourquoi aucun compteur nommé

Marque : [V] vérifié (relu sur le film ou le document) ; [D] déduit.

## Les cinq films

03af54c3, 13b00e35, 47d20b5d, 50247b26, a349fea8 (liste du journal des découvertes, recoupée avec le code :
`internal/grammar/film_identity.go` et la table des amorces). Les cinq sont dans l'archive. Décodage complet (rejeu entier)
de deux : 50247b26 et a349fea8. Lecture du seul `chunk_00` des trois autres.

## Verdict

Film sans section d'identification. Ni chunk d'en-tête absent ou tronqué, ni build inconnu du profil.

Preuves [V], chunk_00 décompressé des cinq films :

| Film | Version majeure | Format | Taille | Registre | Chaîne « HI_ » dans le chunk | Lecture de l'identité |
|---|---|---|---|---|---|---|
| 03af54c3 | 33 | 20 | 1 958 432 | 49 blocs, complet | 0 occurrence | `ErrNoFilmIdentity` |
| 13b00e35 | 31 | 20 | 1 955 336 | 49 blocs, complet | 0 | `ErrNoFilmIdentity` |
| 47d20b5d | 31 | 20 | 1 955 336 | 49 blocs, complet | 0 | `ErrNoFilmIdentity` |
| 50247b26 | 31 | 20 | 1 955 336 | 49 blocs, complet | 0 | `ErrNoFilmIdentity` |
| a349fea8 | 33 | 20 | 1 958 432 | 49 blocs, complet | 0 | `ErrNoFilmIdentity` |

- Le chunk d'en-tête existe, a la taille des autres films (≈ 1,95 Mo décompressé), son registre se lit en entier : pas de
  troncature (`ErrChunk00Truncated` n'est jamais rendue).
- La chaîne de build (`HI_...`) n'apparaît nulle part dans le chunk : la section n'existe pas, elle n'est pas
  illisible. Films de référence : e5adf7b2 (majeure 40, format 25) et a521164d (majeure 33, format 21) portent la chaîne
  (`HI_1_11_0`, `HI_1_4_1`) juste après le registre.
- Les cinq sont du format 20 ; les films de format 21 et plus ont la section (mesuré sur deux films de référence seulement) :
  la section arrive donc avec le format 21 [D ; à vérifier sur un film de format 21 d'une autre majeure].
- Le commentaire de `internal/profile/grenade.go` (zone « majeureSansSection31 ») range 03af54c3 en majeure 31 ; mesuré :
  33 [V]. Aucune conséquence de comportement (les deux amorces sont posées), mais l'affirmation est fausse.

## Pourquoi aucun compteur nommé n'y est lisible

Chaîne mesurée [V] sur 50247b26 et a349fea8 (document de rejeu) :

1. Sans section, pas de table des joueurs du film : refus nommé `sans_section` (`internal/grammar/film_player_table.go:75`).
   Journal : « table du film NON EMPLOYEE ... refus=sans_section liens=0 ».
2. Pas de table : `roster` vide (0 entrée), `bridge.slots` = 0, vies publiées toutes sans identité (244 vies sur
   50247b26, 413 sur a349fea8), verdict « bridge : non publiable : aucun pont ».
3. Pas d'identité : aucun joueur du tableau n'est nommé, `scoreTimeline` ne porte que les équipes (sans camp), donc aucune
   courbe de compteur par joueur (éliminations, tirs à la tête, etc.), aucune élimination, aucun lien avec un joueur de l'API.
4. Cause seconde, indépendante de l'identité : sur ces films anciens, les actions de score sont illisibles (`viewAUnreadable`
   vrai, aucun message lu, groupe de réponses introuvable) et les entrées du tableau ne prouvent pas leur disposition
   (résolveur absent sur les deux films, 122 et 198 enregistrements lus). Même a521164d, qui a pourtant une section
   (`HI_1_4_1`, 128 places au pont, 28 entrées de roster), ne publie aucun joueur du tableau [V] : retrouver la section
   ne suffirait pas à rendre les compteurs nommés sur ces anciens builds.

## Ce que le décodeur fait alors

[V] Le film n'est PAS mis de côté : positions, tirs, grenades, équipement, véhicules sont décodés et publiés
(« nominal » pour tirs et grenades). Ce qui dépend d'une identité est retenu et dit :

- Erreur typée `ErrNoFilmIdentity` (`internal/grammar/film_identity.go:91`), puis refus `sans_section` ; le profil du film
  est « INCOMPLET » (build vide) et jamais lu « au profil du build voisin » (ADR 0034 D-4).
- Repli nommés comptés dans `coverage.fallbacks` : `repli_controle_corruption_section_absente` (2),
  `repli_registre_inconnu_sans_lecteur_de_troncature` (1), `repli_fin_de_vie_sans_tableau_api` (170 et 246),
  `repli_enregistrement_statborg_abandonne` (3 262 et 4 718, fréquent aussi sur les films avec section : 4 709 sur
  e5adf7b2, donc pas propre à ces cinq).
- `coverage.decoder.build` vide et bloc `registry` absent : c'est le sens voulu du pointeur d'identité nul
  (`replay/build_from_facts.go:120`). La version majeure sert de clé de repli pour la table des grenades et l'empreinte du
  registre (`internal/profile/grenade.go:187`, `registre_empreintes.go:216`), statut « présumée ».

## Comportement voulu ?

Oui pour le décodage (erreur typée, repli nommés, rien d'inventé, document utilisable pour la position), c'est la règle D-4.
Deux réserves [D], à décider avant la clôture de la piste :

1. Compteur du refus : `UnknownBuildExpvarPairs` (compteur `filmdec_unknown_build_sans_section`) est exposée par la
   façade, mais je n'ai pas trouvé d'appelant de production dans le code lu (décodeur v1.6.0 en branche et LevelUp de la piste) :
   le refus `sans_section` n'est donc visible que par le journal et `coverage`, pas par un compteur de processus. À
   vérifier côté LevelUp avant d'en faire un défaut.
2. Un joueur ne verra jamais de compteur nommé sur ces cinq films tant que ni la table des joueurs ni la lecture des actions
   de score ne seront lues sans section : c'est une limite de grammaire, pas un bug. Le plan peut clore F.7 en `[!]`
   (non traité, justifié : cinq films de format 20, section inexistante) ; seule une lecture de la table des joueurs
   propre au format 20 changerait le résultat, et rien n'indique qu'elle existe.

## Effets de bord (arbre de travail de la piste de vérité, cache non versionné)

Les décodages passent par la chaîne de production, qui écrit dans le cache de l'arbre de données de la piste principale :

- fichiers de faits de film créés : 03b7bbb6, 221dd082, 0e809d10, 98dd71ff, dbb4cd56, 400bb628 (les autres existaient) ;
- manifestes copiés de l'archive du dossier principal (lecture seule de la source) vers le dossier des manifestes de
  l'arbre : 0e809d10, 98dd71ff, dbb4cd56, 400bb628 ;
- les documents de rejeu produits ont été déplacés dans `replays/` de ce dossier ; le dossier de rejeux de l'arbre est vide.
Aucune base ouverte hors de la copie de l'API (lecture seule). Aucun fichier de production modifié, rien de commité.
