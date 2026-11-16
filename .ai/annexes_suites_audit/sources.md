# Statistiques lues dans le film contre les sources déjà en place — mesure du 2026-11-16

Étape 0.5 du plan `PLAN_SUITES_AUDIT_2026-11-16.md`. Mesure en lecture seule sur le parc local (1 455 rejeux,
schéma 124) et sur une copie des bases ; aucun rejeu généré, aucun serveur lancé. Les champs du schéma 125
(tirs, touches, dégâts) sont absents du parc : ils sont mesurés sur les 8 fixtures du dépôt.

## Résultat par statistique

| Statistique (champ du rejeu) | Source déjà en place | Accord mesuré | Où l'une a raison contre l'autre | La plus fiable |
|---|---|---|---|---|
| Médailles datées (`medals`) | événements du match (`highlight_events`), tirés du MÊME morceau du film ; comptes de médailles de l'API | film = événements nommés à la milliseconde (44 988 triplets match, joueur, médaille, 0 écart) ; film = API pour 99,7 % des joueurs qui ont des lignes d'API | dans 173 matchs (11,9 %) les événements n'ont pas de nom et le rejeu les écarte : 6 200 médailles (9,9 %) que le film porte ; l'API a 83 médailles de plus pour 22 joueurs ; l'API n'a aucune ligne pour 46 % des joueurs | le champ du film : même origine que l'affichage actuel, sans les médailles perdues |
| Actions de score (`scoreActions`) | score personnel du match (API), total sans date | lecture dite complète (64,5 % des matchs) : somme = score personnel pour 98,5 % des joueurs ; lecture incomplète : 62,1 % | 15 matchs dits complets sont faux : 11 BTB (sur 16) et 4 Assaut ; 83 matchs sans aucune action lue | l'API pour le total ; le film est la seule source du détail daté, à n'afficher que si la lecture est complète, et pas en BTB ni en Assaut tant que le verdict y est faux |
| Horloge de manche (`roundClock`) | horloge calculée du rejeu (images écoulées depuis le coup d'envoi) | fin de manche à moins d'1 s pour 96,4 % des matchs à une manche ; fin réglementaire à 0,09 s | le film a raison sur les modes à manches (l'horloge du rejeu ne s'arrête ni ne repart entre deux manches : 13 à 48 s de pause comptées) ; en Roi de la colline et Oddball l'horloge du film est une minuterie d'objectif, pas le temps écoulé ; début décalé de plus de 3 s dans 26 matchs | l'horloge calculée pour le temps écoulé ; le film pour les manches et la minuterie du jeu : deux mesures différentes, pas un doublon |
| Règles de la variante (`rules`) | aucune | 4 champs écrits dans 99,7 % des matchs ; le délai de réapparition égale le minimum et la valeur la plus fréquente des comptes à rebours dans 1 450 matchs sur 1 450 ; tirs alliés vrais dans les 13 matchs classés, et eux seuls | sans objet ; `flagReturnS` n'est écrit dans aucun match ; les pénalités valent toujours 0 | le film, seule source (cohérence interne, pas de confirmation hors du film) |
| Tirs, touches, dégâts (`shotStats`, `hits`, `damage`) | totaux du match de l'API | joueurs à lecture jugée suffisante (13 sur 111) : écart médian 0,8 % (tirs), 1,3 % (touches), 2,3 % (dégâts) ; autres joueurs : 7 % (tirs), 33 % (dégâts) | un joueur « suffisant » à 69 tirs contre 189 pour l'API ; dégâts subis jusqu'à 15 % d'écart | l'API pour les totaux ; le film seulement pour les joueurs à lecture suffisante (12 % dans cet échantillon) |

## Non mesuré

- Tirs, touches, dégâts sur le parc : absents tant qu'il n'est pas régénéré au schéma 125 (3 matchs, 13 joueurs
  à lecture suffisante dans les fixtures).
- Horloge en prolongation : le film ne donne pas d'horloge de prolongation continue.
- Assaut et Husky Raid : minuteries multiples.
- Pourquoi l'API n'a pas de lignes de médailles pour 46 % des joueurs.

## Découvertes (hors périmètre du plan, soumises au user)

1. Médailles absentes du rejeu dans 11,9 % des matchs (événements sans nom écartés par `collectMedalEvents`) ;
   le champ `medals` du document les porte.
2. Verdict « lecture complète » des actions de score faux en BTB (11 matchs sur 16) et en Assaut.
3. `flagReturnS` jamais écrit par le décodeur sur ce parc.
4. Lecture des tirs suffisante pour 12 % des joueurs seulement (échantillon de 111 joueurs).
5. Coup d'envoi estimé par l'API à 5,3 s de l'horloge du film en médiane (jusqu'à 155 s) ; celui lu dans le film
   à 0,36 s.
