# C1 — Vies et présence : nos rejeux contre libfilmshell (2026-11-14, état avant tout changement)

Sources : documents du parc régénéré le 14/11 (schéma 116, `data/cache/replays/halo_infinite`, lus en
lecture seule) ; libfilmshell ad17521 par la sonde `C:\t\phasec1\sonde\` sur l'archive des films ;
comparateur `C:\t\phasec1\cmp\`. Sorties : `C:\t\phasec1\cmp_parc.tsv` (une ligne par film),
`cmp_parc_autres.tsv` (désaccords de propriétaire `AUTRE`, fins de vie écartées `FIN`).

Méthode : une piste de notre document = une vie. Temps d'une piste = `originMs + t x frameIntervalMs`.
Décalage de base de temps estimé par film (médiane des écarts de début des couples au même point de
départ à 1 m près). Appariement d'une vie à l'autre sur le début (après décalage) et la position de
départ. Le propriétaire est comparé par xuid ; les bots (sans xuid des deux côtés) ne sont pas comparés.

libfilmshell refuse les films d'avant HI_1_12_0 : la comparaison porte sur 1 345 films.

| Mesure | Valeur |
|---|---|
| Vies (nous / libfilmshell) | 131 171 / 131 757 |
| Vies appariées | 124 658 (95,0 % des nôtres) |
| Même propriétaire | 122 015 |
| Propriétaire différent | 268 (0,22 % des appariées ; 51 sur le seul film `58d09c44`) |
| Sans nom d'un seul côté | 6 chez nous, 11 chez libfilmshell |
| Fin de vie écartée de plus de 1,5 s | 3 465 (2,8 %) : la nôtre plus tard dans 3 297 cas ; écart médian 4,7 s, p90 15,9 s |
| Films à base de temps décalée de plus de 1 s | 13 (de -11,8 s à +6,1 s) |
| Joueurs (nous / libfilmshell / communs) | 12 386 / 11 705 / 11 696 |
| Arrivées en cours de match (libfilmshell / nous) | 1 732 / 1 680, 128 films en écart |
| Départs en cours de match (libfilmshell / nous) | 1 875 / 1 722, 160 films en écart |

Lecture : le nommage des vies est déjà presque toujours le même ; les écarts se concentrent sur les fins
de vie (notre piste dure plus longtemps), sur quelques films (`58d09c44`, `9fe88ec4`, `4f77afc1` : le
film du Warthog du backlog) et sur les arrivées et départs. Ni l'un ni l'autre côté n'est une vérité :
arrivées et départs se jugent contre l'API (`match_participants`, copie des bases en attente du signal
de levelup-7c) ; fins de vie et propriétaires contre le chunk des temps forts (morts horodatées) et le
fil des morts.

## Jugement des écarts contre le chunk des temps forts (morts horodatées, `C:\t\phasec1\morts_parc.tsv`)

Outils : `C:\t\phasec1\morts\` (morts lues par `decfilm.ParseHighlightEvents`), `C:\t\phasec1\juge\`.
Une mort « tient » une fin de vie si elle tombe à moins d'une seconde de cette fin.

| Écart | Verdict |
|---|---|
| Propriétaire différent (268) | notre joueur meurt à la fin de la vie : 239 ; les deux : 13 ; celui de libfilmshell seul : 0 ; aucune mort : 16 |
| Fin de vie écartée (3 465) | notre fin = la mort : 684 ; la fin de libfilmshell = la mort : 4 ; mort loin des deux : 144 ; pas de mort du joueur dans la vie : 2 633 |

Les 2 633 fins sans mort mêlent trois cas vus à la main : des appariements faux au départ du match
(apparitions simultanées : le comparateur croise deux vies), des vies que libfilmshell clôt quelques
secondes avant la fin du match, et des vies qu'il clôt en cours de match sans mort (passage dans un
véhicule probable, non vérifié). Aucun de ces cas ne désigne une erreur de notre côté.

**Conclusion pour C1.** Sur les écarts tranchables, notre nommage et nos fins de vie sont justes et ceux de
libfilmshell non. La lecture par `unit-control` n'apporte donc pas de justesse en moyenne ; son intérêt
reste de remplacer des récupérations (motif de xuid, pont « premier occupant ») par une lecture, et de
trancher les cas où ces récupérations se trompent (film du Warthog `4f77afc1`, C1.5). Q4 s'applique :
elle ne remplace que si elle nomme au moins aussi bien, ce qui se mesure film par film avec ces outils.
Arrivées et départs : à juger contre l'API (C0.1).
