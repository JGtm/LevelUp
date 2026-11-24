# E.1 — éliminations dont l'auteur est un bot

Mesure, sans code de production. Décodeur v1.5.0 publié (le code lu est identique octet pour octet : `replay/eliminations.go`
de la branche de mesure = celui du module v1.5.0, vérifié par `diff`). Films : cinq films à bots dont les bots tuent, plus
trois films à bots sans élimination. Les films se désignent par leur identifiant court.

Marque : [V] vérifié (chiffre relu sur le document de rejeu, le tableau du film ou la copie de l'API) ; [D] déduit.

## Hypothèses de départ

- « Tableau du film » = les lignes du tableau des scores lues dans le film (compteur d'éliminations par ligne). Les lignes
  que personne ne tient (aucun joueur nommé par les entrées du tableau) sont les lignes des bots. [V] par recoupement
  ci-dessous : leur somme égale, film par film, les éliminations des bots de l'API et des actions de score.
- L'API donne les éliminations des bots (`match_participants`, lignes dont l'identifiant commence par `bid(`) pour quatre
  des cinq films ; pour `01e1f945` l'API ne liste pas le bot.
- Le calque `eliminations[]` est reconstruit à partir de `scoreTimeline.players` (joueurs nommés par un identifiant).

## Résultat : tableau par film

| Film | Mode | Bots au roster | Éliminations d'auteur bot dans `eliminations[]` | Actions de score `killPersonalScore` à destinataire bot | Éliminations des lignes de bots au tableau du film | Éliminations de bots à l'API | Écart calque contre tableau |
|---|---|---|---|---|---|---|---|
| 0e809d10 | Team Slayer (Arena) | 3 | 0 | 8 (jamais publiées) | 8 (3 lignes) | 8 | 8 absentes sur 78 (10,3 %) |
| 98dd71ff | Team Slayer (Arena) | 2 | 0 | 6 | 6 (2 lignes) | 6 | 6 sur 83 (7,2 %) |
| dbb4cd56 | Team Slayer (Arena) | 3 | 0 | 6 | 6 (3 lignes) | 6 | 6 sur 74 (8,1 %) |
| 400bb628 | CTF (Arena) | 6 | 0 | 14 | 14 (3 lignes) | 14 | 14 sur 111 (12,6 %) |
| 01e1f945 | KOTH (Arena) | 1 | 0 | 1 | 1 (1 ligne) | 0 (bot absent de l'API) | 1 sur 107 (0,9 %) |
| 51101d1d | CTF Neutral Flag | 3 | 0 | 0 | 0 | 0 | aucun |
| 9f57c612 | Assaut One Bomb | 2 | 0 | 0 | 0 | 0 | aucun |
| bcb6d393 | CTF (Arena) | 4 | 0 | 0 | 0 | 0 | aucun |

Recoupements [V] :

- Slayer par équipes : somme des scores d'équipe du film = éliminations des humains + éliminations des bots.
  0e809d10 : 28 + 50 = 78 = 70 + 8. 98dd71ff : 33 + 50 = 83 = 77 + 6. dbb4cd56 : 24 + 50 = 74 = 68 + 6.
- Dans les cinq films, le nombre de `killPersonalScore` du film (78, 83, 74, 111, 107) = éliminations publiées + éliminations
  de bots (70+8, 77+6, 68+6, 97+14, 106+1).
- Éliminations publiées = somme des éliminations des lignes nommées du tableau, dans les huit films (28, 72, 106, 49,
  70, 77, 68, 97) : les humains sont exacts, rien n'est attribué à tort.
- Aucune élimination du calque n'a un tueur vide ou un tueur de mauvaise identité : 0 sur 8 films.

## Côté victime (bots tués)

Les éliminations dont la victime est un bot perdent leur champ `victim` : `killPersonalScore` à cible sans humain au
roster = 20 (0e809d10), 18 (98dd71ff), 17 (dbb4cd56), 26 (400bb628), 2 (01e1f945), 1 (bcb6d393) [V]. Les morts de bots à
l'API : 20, 19, 20, 26 [V] ; les écarts de 1 et 3 (98dd71ff, dbb4cd56) sont des morts hors élimination (suicides, 1 et 4
actions `playerSuicidePersonalScore`) [D : non recoupé mort par mort].

## Cause : où le code décide de l'auteur

Verdict : les éliminations faites par un bot sont ABSENTES du calque, ni attribuées à tort ni présentes. La cause est une
règle d'identité, pas un défaut de lecture : le film nomme bien ces éliminations (tableau et actions de score), mais un bot
n'a pas d'identifiant, et tout le calque est indexé par identifiant.

1. `replay/eliminations.go:260` : le calque ne parcourt que `scoreTimeline.players` ; chaque ligne publiée prend
   `XUID: p.XUID` (`:225`). Les actions de score se rangent par identifiant du destinataire (`:250`).
2. `replay/score_timeline_players.go:166-169` (`playerRoundsByXUID`) : une série dont le propriétaire est vide (`""`) est
   sautée. Le propriétaire vient de `internal/facts/objectives/entry_identity.go:145-151` (`Split`) : une ligne sans titulaire
   nommé par les entrées du tableau garde son numéro de ligne et n'est pas nommée. Les bots n'y figurent pas [V par le
   résultat : les lignes non nommées sont exactement celles des bots] ; la raison exacte (les entrées ne listent que des
   identifiants) est [D].
3. `replay/document_score_actions.go:182-188` (`humainsParIndex`) exclut les bots (`!e.Bot && e.XUID != ""`) ;
   `:218-222` rend « sans humain » ; `:293-294` compte `noRecipient` et JETTE l'action. Même règle pour la cible
   (`:338-346`, `noTarget`).

Conséquence : `coverage.eliminations` et `coverage.scoreActions` sont silencieux sur ces pertes (le compte d'éliminations
est juste pour les humains ; `noRecipient` mélange éliminations et assistances de bots : 17 = 8 + 9 sur 0e809d10).

## Correction prescrite (non écrite)

Publier les éliminations d'un bot avec un tueur sans identifiant mais avec sa place : champ `seat` (index du joueur du film,
celui que porte déjà l'entrée de roster du bot, `roster[].filmIndex`) et `xuid` absent ; la place se lit sur l'index du
destinataire des actions `killPersonalScore` sans humain [D : l'index du destinataire = l'index de roster est établi par la
construction des actions, mais non rejoué ici pour les bots], et le nombre d'éliminations d'un bot se prend sur ces actions
(seule source qui porte déjà la place), contrôlé contre la somme des lignes non nommées du tableau. Même traitement pour
`victim` : une victime de bot se publie par sa place (`victimSeat`), jamais par un faux identifiant ; ajouter un compteur
de couverture « éliminations de bots publiées / lignes non nommées » pour que l'écart redevienne visible.

## Traçabilité

Documents de rejeu construits par la chaîne de production (`replay-build`) sur les faits de match exportés de la copie de
l'API, sondes dans `sondes/` (étiquette de test jetable, non commitée), sorties dans `replays/`. Effets de bord sur l'arbre
de travail de la piste (cache non versionné) : voir `f7.md`, section « Effets de bord ».
