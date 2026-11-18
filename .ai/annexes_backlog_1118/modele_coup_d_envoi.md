# Coup d'envoi du match : modèle cible (item T.1 du plan du backlog du 2026-11-18)

Statut : **proposé par le pilote le 2026-11-18, à valider par le user avant tout code (porte de T.2).**
Demande du user : « faut remettre au propre, qu'on ait pas plusieurs paramètres et de fallback, je veux un
système cohérent ». Pièces : audit `coup_d_envoi.md` (inventaire fermé : 11 sources S1 à S11, 7 qualités,
22 replis R1 à R22, 4 durées), mesures `mesure_coup_d_envoi.md`.

## 1. Une définition

Le coup d'envoi d'un match est l'instant où l'horloge du jeu de la première manche démarre. Il n'y en a
qu'un par match, et toutes les pages le lisent au même endroit.

Preuve que l'horloge est la bonne lecture du film (audit B.3) : dans les 22 rejeux où l'horloge et le
« premier mouvement » diffèrent de 17 s, le troisième joueur à bouger réellement le fait 0,2 à 0,8 s après
le départ de l'horloge, et 17 à 22 s après le « premier mouvement » ; les arrivées des joueurs côté API
tombent 0,2 s après l'horloge. L'horloge date les 1 455 rejeux du poste ; le détecteur de mouvement en date
22 trop tôt et en refuse 8.

## 2. Une source par match, dans un ordre écrit une fois

| Rang | Source | Quand | Valeur |
|---|---|---|---|
| 1 | `film` | le match a un rejeu | départ de l'horloge, lu dans le rejeu (publié par le décodeur) |
| 2 | `api` | pas de rejeu, et l'API donne les arrivées des joueurs présents au début | l'estimation actuelle (première arrivée concordante), refusée si elle est négative ou au-delà de 120 s |
| 3 | inconnue | ni l'un ni l'autre (Halo 5, données absentes, estimation refusée) | aucune valeur |

L'ordre est décidé en UN point, à l'écriture. Les lecteurs ne choisissent jamais. La source est rangée à
côté de la valeur. Deux sources restent parce qu'un match sans film n'a que l'API : c'est un ordre de
priorité, pas un repli dispersé.

## 3. Ce qui est rangé

- Au registre du match : le coup d'envoi (colonne existante `real_start_time`) et sa source (`film`, `api`
  ou vide). La colonne `t0_quality` et ses sept valeurs disparaissent au profit de cette source : aucune page
  ne lisait la qualité (audit A.2). Le détail de l'estimation (une seule arrivée, arrivées dispersées, rejet)
  devient un message de journal et un compteur.
- Par joueur : son temps de jeu, calculé par la formule actuelle (présence du joueur, bornée par le coup
  d'envoi et la fin du match), toujours avec le coup d'envoi DU REGISTRE. Aujourd'hui il est calculé avec
  une estimation à part et n'est jamais refait (10 895 joueurs sur 1 345 matchs sont décalés, médiane 5 s).

## 4. Une résolution

Une seule fonction rend la chronologie d'un match : début, coup d'envoi, fin, durée de jeu (fin moins coup
d'envoi). Coup d'envoi inconnu : la chronologie le dit, et la durée de jeu vaut la durée du match ; cette
règle est écrite là et nulle part ailleurs. Côté SQL, un fragment pour le coup d'envoi (il existe) et un
fragment pour la durée de jeu (aujourd'hui recopiée).

Conséquences :
- les trois autres durées disparaissent des lecteurs : la durée jouable de l'API (fausse en Arène classée,
  égale à la durée dans 1 561 matchs sur 2 045), la durée « observée » par la médiane des temps de jeu (elle
  devient égale à la durée de jeu par construction), la durée moins coup d'envoi recalculée sur place ;
- la Dominance, l'en-tête, l'Escouade, les Sessions, la Tactique, le drapeau Prolongation et le LUSR lisent
  la même durée de jeu ;
- le champ servi à la page du match sous le nom `playable_duration_seconds` (qui est déjà la durée de jeu)
  prend son vrai nom, `gameplay_duration_seconds` ;
- la page du rejeu lit le coup d'envoi du rejeu, sans « sinon celui de l'API » : tous les rejeux en portent
  un dès la version du décodeur qui lit l'horloge.

## 5. Une écriture

Un seul persisteur écrit, pour un match et dans la même transaction, le coup d'envoi, sa source et le temps
de jeu de chaque joueur. Deux appelants : la synchronisation (source `api`, valeurs posées dans les lignes
avant leur insertion, comme aujourd'hui) et la publication d'un rejeu (source `film`), sur TOUS les chemins
de génération : aujourd'hui le report du film ne se fait que sur le chemin local, cinq rejeux par cycle, et
jamais sur le chemin de la prod (audit A.1, S2). Plus aucune écriture directe hors de `internal/persist`.

## 6. Un rattrapage

Une seule fonction recalcule coup d'envoi, source et temps de jeu de tous les matchs d'une base. Elle sert
à deux endroits :
- au démarrage, quand la révision rangée dans la base diffère de celle du code (même mécanique que la
  révision des notes, ADR 0047) : c'est ce qui met à jour l'application de bureau et la prod sans geste ;
- par une commande `levelup`, avec un mode de simulation qui liste les écarts sans écrire.
Les trois commandes `cmd/backfill_t0`, `cmd/backfill_t0_film` et `cmd/backfill_time_played` sont supprimées.

## 7. Les notes

La note de performance lit le temps de jeu ; le LUSR lit le temps de jeu ET le coup d'envoi. Les notes
rangées datent d'avant le report du film : elles sont déjà périmées (le nombre de joueurs pondérés sous 0,99
dans le LUSR passe de 2 065 à 4 048 avec les coups d'envoi du film). Les deux révisions de notation montent
donc avec ce chantier : au premier démarrage de la 8.3.0, le rattrapage passe, puis toutes les notes sont
rejouées une fois, joueur par joueur. La formule du temps de jeu ne change pas.

## 8. Le rejeu et le décodeur

Le décodeur publie UN coup d'envoi, lu sur l'horloge ; le détecteur de premier mouvement et ses refus sont
retirés (piste A du plan du décodeur). LevelUp lit le même champ avant comme après. Ordre : ce système
s'installe d'abord avec le champ actuel ; après la régénération du parc de la v1.5.0, la publication de
chaque rejeu réécrit son coup d'envoi par l'écriture unique. Le rattrapage du poste (T.6) se fait donc UNE
fois, après la régénération.

## 9. Sort de l'inventaire

| Élément de l'audit | Sort |
|---|---|
| S1, S1b estimation de l'API (deux appels) | gardée, UN appel, rang 2 |
| S2 report du film au fil de l'eau | remplacé par l'écriture unique, sur tous les chemins |
| S3, S4, S5 trois commandes | supprimées, remplacées par le rattrapage unique |
| S6 temps de jeu au sync | gardé, même formule, coup d'envoi du registre |
| S7 colonne `film_match_start_ms`, S8 `EstimateFilmMatchStartMS` et ses deux fonctions | code supprimé ; colonne retirée |
| S9 import OpenSpartan, S11 graine de démo | passent par la même écriture |
| Sept valeurs de `t0_quality` | remplacées par la source (`film`, `api`, vide) |
| Seuils de 2 s et de 120 s de l'estimation | gardés, privés à l'estimation, plus aucun lecteur |
| 11 matchs « negative » servis avec un coup d'envoi de 2 à 4 h | corrigés par le rattrapage (estimation refusée : source vide) |
| R1 à R8, R19 à R22 (coup d'envoi absent remplacé par 0 ; durées concurrentes) | une règle, dans la chronologie |
| R9, R10 (« temps de jeu, sinon durée jouable », « sinon durée ») | retirés : le temps de jeu vient toujours de l'écriture unique ; absent, il reste absent |
| R15 durée « observée » | retirée, remplacée par la durée de jeu |
| R16, R17 (valeur de l'API gardée quand le calcul échoue) | une règle, à l'écriture : présence inconnue = temps de jeu de l'API, source dite |
| R18 fin de session = début + temps de jeu | gardé (n'est pas un repli du coup d'envoi) |
| R11 à R14 (note de performance et LUSR quand le temps de jeu d'un joueur manque : 600 s, poids, ordre des quitteurs) | HORS de ce système : ce sont des paramètres des notes ; listés au backlog comme chantier à part s'il est voulu |
| Durée jouable de l'API (colonne) | plus écrite, plus lue ; colonne retirée |
| Halo 5 | source inconnue, durée de jeu = durée du match, inchangé à l'écran |

Les colonnes retirées (`t0_quality`, `film_match_start_ms`, `playable_duration_seconds`) le sont par la
recette de migration du dépôt ; si la recette impose de reconstruire la table du registre, la colonne reste
en base, sans écrivain ni lecteur, et son retrait est noté au backlog : aucun risque n'est pris sur la base
partagée pour un nom de colonne.

## 10. Garde-rails

- Le coup d'envoi et la durée de jeu ne se lisent que par la chronologie et ses deux fragments SQL.
- Aucune écriture du coup d'envoi ni du temps de jeu hors du persisteur unique.
- Aucune lecture de la durée jouable de l'API.
- Empreinte de la révision du coup d'envoi, comme pour les notes.

## 11. Limites connues, dites

- Un match sans rejeu garde une estimation. Pour 350 anciens matchs, elle place le coup d'envoi dans les 5
  premières secondes, alors qu'un vrai compte à rebours dure une vingtaine de secondes : leurs taux par
  minute restent sous-estimés d'environ 3 %. Rien de mieux n'existe sans le film, et aucune valeur
  « typique » n'est inventée.
- Le temps de jeu reste tronqué à la seconde (une seconde de moins en moyenne) : changer l'arrondi serait
  un changement de formule, écarté ici.
- 437 matchs ont une durée de l'API qui diffère de « fin moins début » de plus d'une seconde : non instruit,
  noté aux découvertes.

## 12. Ce que le user verra

- Le même coup d'envoi et la même durée sur toutes les pages d'un match (Dominance et en-tête ne peuvent
  plus diverger).
- Temps de jeu et taux par minute recalés de quelques secondes sur la plupart des matchs à rejeu, de 17 s ou
  plus sur une trentaine.
- Notes de performance et LUSR rejoués une fois à la mise à jour : des écarts faibles sont attendus, à
  mesurer et à te montrer avant la sortie (item T.6).
