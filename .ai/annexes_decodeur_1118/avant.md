# Mesures AVANT, par piste (plan décodeur 2026-11-18, item 0.4)

Date des mesures : 2026-11-18. Parc : 1 455 rejeux du poste (schéma 125 pour tous), décodeur v1.4.1. Vérité :
export de l'API sur une copie de la base partagée (`C:/t/verite/`, hors dépôt). Aucun film décodé, aucune base
ouverte hormis la copie, aucun réseau. Aucun identifiant de joueur dans ce fichier (les noms de joueurs de la
liste E ne servent qu'à retrouver la ligne).

## Méthode et reproduction

Tout part de deux scripts, lancés un par un (Node `--max-old-space-size=1024`), sous `C:/t/dec/avant/` :

| Script | Rôle | Sorties |
|---|---|---|
| `pilote_mesure_horloge.js` (copie du script gardé, inchangé) | piste A | `horloge.out` |
| `extract.js` | UNE passe sur les 1 455 rejeux (un seul en mémoire) : écrit une ligne compacte par rejeu | `extract.jsonl` |
| `lib.js`, `analyse.js <join\|B\|C\|D\|E\|F\|G\|H\|J>` | lit `extract.jsonl` et l'export, sort les chiffres | `sortie_<section>.txt` |
| `analyse_I.js` | piste I, depuis `evenements_mode.tsv` | `sortie_I.txt`, `sortie_I_distinct.txt` |
| `b_variante.js`, `md_c.js` | variante de la mesure B (convention de comptage), tableau C par build | `sortie_B_variante.txt`, `sortie_C_md.txt` |
| `q_spree.sql` (via `C:/t/duckq`, copie en lecture seule) | colonne `max_killing_spree`, absente de l'export initial | `spree.tsv` |

Les rejeux ne sont jamais écrits. Les sondes gardées sous
`.ai/annexes_decodeur_1118/sondes/` sont copiées sous `C:/t/dec/avant/` ; seule `pilote_mesure_horloge` est
rejouée telle quelle, les autres mesures sont refaites par `extract.js` + `analyse.js` (une passe unique plutôt
que vingt sondes qui relisent le parc), et retrouvent les chiffres de la section 1 là où elle en donne (écarts
signalés plus bas).

### Jointures (taux avant tout autre chiffre)

- Rejeu -> match : clé `matchId` du document = `match_id` de l'export. 1 455 / 1 455 présents au registre ;
  1 450 rejeux ont au moins un joueur joint ; 5 n'en ont aucun (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`,
  `a349fea8` : ce sont exactement les 5 rejeux sans build, vérifié par `verif.js`).
- Joueur : son xuid du rejeu (`roster[].xuid`, `scoreTimeline.players[].xuid`) = `xuid` de l'export (décimal,
  même forme). Jamais recopié ici.
  - Humains du roster joints : 14 303 / 14 346 (99,70 %) ; entrées de roster sans xuid (bots) : 1 124, non
    jointes par construction.
  - Joueurs du tableau du film (`scoreTimeline.players`) : 14 034 / 14 034 (100 %).
  - Bilan de tirs (`shotStats`) : 14 052 / 14 053 (99,99 %).
  - Sens inverse : lignes de l'export pour les matchs couverts 15 489, retrouvées dans le roster 14 303
    (92,3 % ; le reste = bots de l'API et humains que le roster ne porte pas).
- Le registre n'a pas de colonne de build : le build vient du rejeu (`coverage.decoder.build`). Répartition :
  `HI_1_13_0` 1 200 ; `HI_1_12_0` 145 ; `HI_1_11_0` 57 ; `HI_1_10_0` 31 ; `HI_1_8_0` 13 ; sans build 5 ;
  `HI_1_9_0` 3 ; `HI_1_4_1` 1.
- Le « mode » est `mode_category` du registre (clé match) ; Grande bataille = `BTB`.

## Piste A — coup d'envoi

Script : `pilote_mesure_horloge.js` sur le dossier des rejeux (`horloge.out`). Question tranchée ailleurs
(l'horloge a raison) ; seuls les nombres.

| Grandeur | Valeur |
|---|---|
| Rejeux lus | 1 455 |
| Datés par l'horloge de manche (un départ établi) | 1 455 (aucun sans horloge, aucun sans départ) |
| Datés par le premier mouvement (`t0FilmMs` présent) | 1 447 ; 8 sans (refus du détecteur ; ils ne sont datés que par l'horloge) |
| Comparés (les deux dates) | 1 447 |
| Écart « premier mouvement moins horloge », secondes : min / p05 / p25 / médiane / p75 / p95 / max | -21,38 / 0,30 / 0,33 / 0,36 / 0,40 / 0,53 / 1,49 |
| Dans une demi-seconde | 1 329 |
| Dans une seconde | 1 422 |
| Entre 1 s et 3 s | 3 |
| Au-delà de 3 s | 22 (exemples relevés à environ -16,8 s : `00ba2e1c`, `06a883f7`, `087474d9`, `0f34365a`) |

La première lecture de l'horloge est dans l'état 1 pour 1 439 rejeux, déjà dans l'état 6 pour 15, dans l'état
19 (horloge qui monte) pour 1.

## Piste B — verdict du score

Script : `extract.js` + `analyse.js B` (`sortie_B.txt`), `b_variante.js`. Somme des points = somme de
`scoreActions[].points` par joueur ; score du tableau = dernière valeur de `scoreTimeline.players[].score.total`.

| Grandeur | Valeur |
|---|---|
| Rejeux avec `scoreActions` publié | 1 367 |
| Rejeux dits « complets » (`coverage.scoreActions.complete`) | 944 (511 incomplets) |
| Complets où la somme d'un joueur diffère du tableau, joueurs SANS score de tableau exclus | 43 rejeux, 163 joueurs sur 8 017 |
| Idem, joueur sans score de tableau compté 0 (convention de la sonde de l'instruction) | 44 rejeux (le 44e est `96cd58c8` : un joueur sans ligne de score) |
| Joueurs sans score de tableau, dans les complets / sur le parc | 79 / 236 |

Les 43 par groupe de scripts : `slayerevents.lua` 24, `bigteambattle.lua+btbslayerevents.lua` 12,
`arenaflagevents.lua+flageventstubs.lua` 5, `areacaptureeventstubs` (Bastion, Bases) 2. Par build :
`HI_1_13_0` 29, `HI_1_11_0` 12, `HI_1_12_0` 2. Par variante : BTB (Slayer, Fiesta Slayer, Heavies) 12 ; Assaut
(One Bomb 3, Neutral Bomb 1) 4 ; Slayer Arena / Super Fiesta / Team Slayer 20 ; CTF et dérivés 5 ; Bases et
KOTH 2. Ce sont les chiffres de la section 1 (12 / 4 / 28 autres). Écarts « tableau moins somme » les plus
fréquents (joueurs) : -25 : 55 ; +100 : 29 ; -50 : 16 ; +50 : 15 ; +25 : 5 ; +75 : 4 ; -75 : 4. Rejeux dont tous
les joueurs en écart manquent exactement 100 points : 23.

### Contre l'API (somme des points publiés contre `personal_score`, par joueur humain joint)

| Population | Joueurs comparés | Accord exact | Répartition de l'écart (somme moins API) |
|---|---|---|---|
| Rejeux « complets » | 8 194 (18 non joints) | 8 065 (98,42 %) | exact 8 065 ; \|écart\| <= 25 : 63 ; 26 à 100 : 45 ; > 100 : 21 ; somme < API 55, somme > API 74 |
| Tous les rejeux avec `scoreActions` | 12 302 (32 non joints) | 11 722 (95,29 %) | exact 11 722 ; <= 25 : 135 ; 26 à 100 : 154 ; > 100 : 291 ; somme < API 397, somme > API 183 |

Complets, par mode (accord exact / comparés) : Super Fiesta 2 877/2 877 ; Assassin (Arena) 4 653/4 682
(99,38 %) ; BTB 356/456 (78,07 %) ; Ranked 56/56 ; Husky Raid 91/91 ; Fiesta 19/19 ; Other 13/13.
Complets, par build : `HI_1_13_0` 6 971/6 991 (99,71 %) ; `HI_1_12_0` 868/877 (98,97 %) ; `HI_1_11_0` 226/326
(69,33 %).
Tous, par mode : BTB 645/1 139 (56,63 %) ; Assassin 7 049/7 120 (99,00 %) ; Ranked 64/72 ; Husky Raid 116/123 ;
Super Fiesta 3 781/3 781 ; Fiesta 46/46 ; Other 21/21. Tous, par build : `HI_1_13_0` 10 008/10 060 (99,48 %) ;
`HI_1_12_0` 1 224/1 233 (99,27 %) ; `HI_1_11_0` 426/610 (69,84 %) ; `HI_1_10_0` 40/315 (12,70 %) ; `HI_1_9_0`
24/84 (28,57 %). Rejeux « complets » par build (script `verif.js`, `sortie_verif.txt`) : `HI_1_13_0` 835 / 1 200 ;
`HI_1_12_0` 97 / 145 ; `HI_1_11_0` 12 / 57 ; aucun pour `HI_1_10_0`, `HI_1_9_0`, `HI_1_8_0`, `HI_1_4_1` ni les
rejeux sans build : les sommes de ces builds sous-estiment l'API et sont déjà dits incomplets.

## Piste C — compteurs nommés

Script : `extract.js` + `analyse.js C` (`sortie_C.txt`) et `md_c.js` (`sortie_C_md.txt`). Valeur du film = dernier
point de `total` de la série ; `maxSpree` = maximum de ses points par manche (la série n'a pas de total).
L'équivalent API de `maxSpree` existe : `match_participants.max_killing_spree` (export `spree.tsv`).
Joueurs du tableau : 14 034, tous joints.

Accord EXACT avec l'API, parmi les joueurs dont le film porte la valeur (exact / lus ; entre parenthèses les
joueurs dont le film n'a pas de valeur, tous à 0 côté API sauf un cas de série) :

| build | éliminations | morts | assistances | tirs à la tête | corps à corps | grenades | armes lourdes | série `maxSpree` |
|---|---|---|---|---|---|---|---|---|
| sans build | 1/1 | 1/1 | 1/1 | n/a (+1) | n/a (+1) | n/a (+1) | n/a (+1) | 0/1 |
| HI_1_8_0 | 227/227 | 227/227 | 189/189 (+38) | 157/157 (+70) | 147/148 (+79) | 148/148 (+79) | 125/125 (+102) | 210/211 (+16) |
| HI_1_9_0 | 67/67 | 67/67 | 57/57 (+10) | 34/34 (+33) | 36/38 (+29) | 37/38 (+29) | 38/38 (+29) | 60/61 (+6) |
| HI_1_10_0 | 759/759 | 759/759 | 650/650 (+109) | 457/457 (+302) | 413/413 (+346) | 407/413 (+346) | 488/488 (+271) | 709/715 (+44) |
| HI_1_11_0 | 1 468/1 468 (+4) | 1 468/1 468 (+4) | 1 235/1 235 (+237) | 906/908 (+564) | 744/748 (+724) | 747/748 (+724) | 854/855 (+617) | 1 352/1 361 (+111) |
| HI_1_12_0 | 1 420/1 420 | 1 420/1 420 | 1 247/1 247 (+173) | 1 040/1 042 (+378) | 951/953 (+467) | 952/953 (+467) | 641/641 (+779) | 1 380/1 383 (+37) |
| HI_1_13_0 | 10 078/10 084 (+4) | 10 074/10 084 (+4) | 9 141/9 151 (+937) | 6 977/6 987 (+3 101) | 7 193/7 207 (+2 881) | 7 181/7 207 (+2 881) | 6 430/6 432 (+3 656) | 9 869/9 879 (+209) |
| Tous | 14 020/14 026 (99,96 %) | 14 016/14 026 (99,93 %) | 12 520/12 530 (99,92 %) | 9 571/9 585 (99,85 %) | 9 484/9 507 (99,76 %) | 9 472/9 507 (99,63 %) | 8 576/8 579 (99,97 %) | 13 580/13 611 (99,77 %) |

Un compteur « absent » du film vaut 0 à l'API pour 100 % des cas (assistances 1 504, tête 4 449, corps à corps
4 527, grenades 4 527, armes lourdes 5 455 ; série : 422 sur 423 ; éliminations et morts : 8). Comptés comme 0,
l'accord devient : tête 99,9 %, corps à corps 99,8 %, grenades 99,8 %, armes lourdes 100 %.

Défauts de la section 1, mesurés :

| Grandeur | Valeur |
|---|---|
| Couples joueur-manche où un compteur nommé dépasse les éliminations de la manche | 27, dans 26 rejeux (sur 14 491 couples joueur-manche lus ; la section 1 divise par les 14 026 joueurs) |
| Par compteur | tête 2 ; corps à corps 7 ; grenades 18 ; armes lourdes 0 |
| Valeurs de série (`maxSpree`) supérieures aux éliminations du joueur | 23 joueurs dans 22 rejeux ; maximum 29 812 |
| Séries du film supérieures à celle de l'API | 24 joueurs (sur 13 611 lus) |
| Joueurs dont les éliminations à la grenade valent au moins 12 | 19, TOUS à exactement 16 (dans 18 rejeux) |
| Parmi ces 19, l'API dit aussi 16 | 0 sur 19 |
| Rejeux avec au moins un défaut (couple, série, 16) | 51 |

20 rejeux témoins (8 premiers caractères ; classés par nombre de défauts, puis ordre alphabétique) : `643ad899`
(4), `11de8353` (3), `153a82d6`, `17ba02e3`, `32acd9bf`, `34bb3bc8`, `81019bcb`, `8a726369`, `a51e8d62`,
`bd4a7521`, `c744aa29`, `f1ae8345`, `f2e14485`, `f5443e17`, `f58b857e` (2 chacun), `0d0bc019`, `12999e9f`,
`13b00e35`, `145908d1`, `1550b5c0` (1 chacun).

## Piste D — tirs et touches

Script : `extract.js` + `analyse.js D` (`sortie_D.txt`). Entrées : `shotStats[]` (tirs, touches `hits`, dégâts
infligés `damageDealt`) contre `shots_fired`, `shots_hit`, `damage_dealt`. « À 10 % près » : valeur API > 0 :
`|film - API| <= 0,1 x API` ; valeur API = 0 : le film vaut 0 (87 cas de tirs, 134 de touches, 90 de dégâts à
zéro côté API). Arme dominante : l'arme (famille, libellé `en`) qui porte le plus de tirs du joueur dans
`shots[]`, tirs rattachés au joueur par la vie de leur slot.

| Grandeur | Valeur |
|---|---|
| Joueurs du bilan de tirs | 14 053 (14 052 joints) |
| Joueurs « suffisants » | 6 469 (46,03 %) |
| À 10 % de l'API, TOUS les joueurs | tirs 59,74 % ; touches 58,41 % ; dégâts 64,77 % (n = 14 052) |
| À 10 %, joueurs suffisants (6 469) | tirs 81,98 % ; touches 83,89 % ; dégâts 82,64 % |
| À 10 %, joueurs non suffisants (7 583) | tirs 40,76 % ; touches 36,67 % ; dégâts 49,53 % |

Par arme dominante (armes portées par au moins 100 joueurs ; part de joueurs à 10 % près de l'API) :

| Arme | Joueurs | Suffisants | Tirs | Touches | Dégâts |
|---|---|---|---|---|---|
| MA40 AR | 4 429 | 55,43 % | 71,01 % | 66,27 % | 64,08 % |
| BR75 | 3 261 | 56,15 % | 73,63 % | 67,86 % | 59,95 % |
| Mk50 Sidekick | 2 023 | 65,45 % | 71,58 % | 75,73 % | 81,76 % |
| Needler | 1 323 | 23,58 % | 19,27 % | 40,06 % | 77,48 % |
| M392 Bandit | 433 | 21,02 % | 71,13 % | 36,95 % | 30,25 % |
| VK78 Commando | 392 | 19,13 % | 27,55 % | 33,16 % | 59,69 % |
| Pulse Carbine | 359 | 21,17 % | 31,75 % | 35,65 % | 62,95 % |
| Disruptor | 233 | 22,32 % | 15,02 % | 26,61 % | 65,67 % |
| Plasma Pistol | 204 | 16,67 % | 19,61 % | 29,90 % | 76,96 % |
| MA5K Avenger | 185 | 5,95 % | 55,14 % | 39,46 % | 38,38 % |
| Ravager | 184 | 22,83 % | 23,37 % | 25,54 % | 69,57 % |
| Shock Rifle | 183 | 15,30 % | 16,94 % | 29,51 % | 55,19 % |
| S7 Sniper | 181 | 27,07 % | 54,70 % | 37,02 % | 52,49 % |
| (aucun tir rattaché) | 108 | 1,85 % | 77,78 % | 65,74 % | 70,37 % |
| autres armes (< 100 joueurs) | 554 | 15,70 % | 32,67 % | 26,17 % | 46,57 % |

## Piste E — drapeau

Script : `extract.js` + `analyse.js E` (`sortie_E.txt`). Portage d'un joueur = somme des intervalles de
`flagCarries[].spans[]` portant son identifiant, états `carried` (fermé) et `carried_open` (ouvert), en secondes ;
variante « fermé seul » donnée aussi. API : `time_as_flag_carrier_seconds` (aucune valeur manquante sur les
joueurs joints). Taille du match = longueur du roster du rejeu (bots compris).

| Grandeur | Valeur |
|---|---|
| Rejeux à drapeau (`flagCarries` publié ; `coverage.flagCarries.flagFilm` vrai) | 186 ; dont mode BTB : 11 ; dont roster de plus de 8 : 94 |
| Joueurs humains comparés | 1 750 (2 non joints) |
| Accord à 2 s près, tous (portage publié, fermé + ouvert) | 1 542 (88,11 %) ; \|écart\| : <= 2 s 1 542 ; 2 à 10 s 132 ; 10 à 30 s 54 ; > 30 s 22 ; film > API 73, film < API 135 |
| Matchs de 8 joueurs ou moins | n = 736 ; accord 675 (91,71 %) ; 2 à 10 s 46 ; 10 à 30 s 13 ; > 30 s 2 ; film > API 46, film < API 15 |
| Matchs de plus de 8 joueurs | n = 1 014 ; accord 867 (85,50 %) ; 2 à 10 s 86 ; 10 à 30 s 41 ; > 30 s 20 ; film > API 27, film < API 120 |
| Variante fermé seul : tous / <= 8 / > 8 | 88,23 % (1 544) / 91,98 % (677) / 85,50 % (867) |

Au-delà de 8 joueurs le film sous-estime le plus souvent (120 cas de portage manquant ou trop court contre 27
trop long). Le film `390b1de5` donne le faux porteur de la section 1 : 94,3 s publiées pour un joueur dont l'API
dit 0.

20 plus gros écarts (match, joueur, taille du roster, film en s, API en s) :

| Match | Joueur | Roster | Film | API |
|---|---|---|---|---|
| `cdb0ffcf` | DissidentJack | 26 | 0,0 | 101,3 |
| `390b1de5` | Lenweeee | 25 | 94,3 | 0 |
| `084a804d` | Madina97294 | 26 | 0,0 | 82,9 |
| `378659b1` | DonaldJTrump894 | 24 | 0,0 | 80,8 |
| `cdb0ffcf` | Sass2208 | 26 | 0,0 | 77,3 |
| `b1bcbe24` | RedBaron7928 | 24 | 0,0 | 73,3 |
| `5f90df2e` | PR3DTR | 29 | 0,0 | 60,2 |
| `390b1de5` | bAlThAzAr1104 | 25 | 0,0 | 46,3 |
| `dc7bbf5b` | Mangoshooter449 | 8 | 129,5 | 83,6 |
| `378659b1` | Samc Mtz | 24 | 0,0 | 43,2 |
| `ade271ff` | Chocoboflor | 9 | 21,0 | 63,9 |
| `7b60cff5` | Sirol213 | 9 | 0,6 | 41,3 |
| `be758198` | skriccy | 24 | 0,0 | 40,2 |
| `fc1465f2` | RamenRiot2399 | 10 | 54,6 | 92,8 |
| `be758198` | Madina97294 | 24 | 0,0 | 37,5 |
| `ade271ff` | Just JamJar | 9 | 0,0 | 33,1 |
| `760fb768` | MrBeefDinner | 11 | 0,0 | 32,8 |
| `9ea96468` | The lion3603496 | 8 | 53,5 | 21,2 |
| `60320538` | Jackofcool777 | 15 | 0,0 | 31,5 |
| `e96c390b` | OwOMiguelOwO585 | 9 | 0,0 | 31,3 |

## Piste F — départs et résumés d'usage

Script : `extract.js` + `analyse.js F` (`sortie_F.txt`). Partant = humain joint dont la présence publiée se
ferme au moins 10 s avant la dernière image du document. A = somme des intervalles de `roster[].presence`
(secondes) ; B = (fin de la dernière vie publiée du joueur moins début de sa première présence) en secondes ;
T = `time_played_seconds` de l'API.

| Grandeur | Valeur |
|---|---|
| Humains avec une présence publiée | 14 270 (4 non joints) |
| Partants joints | 1 452 (dont 67 sans vie publiée, écartés de la comparaison A/B : 1 385 comparés) |
| Partants que l'API dit partis (`left_in_progress`) parmi les 1 385 | 1 385 sur 1 385 |
| Écart A - T (s) : p5 / p25 / médiane / p75 / p95 | -67,1 / -30,0 / -16,5 / -6,7 / +1,7 ; \|écart\| médian 17,9, p90 48,1 ; à 5 s : 14,08 % ; à 30 s : 73,00 % ; au-delà de 30 s : 374 |
| Écart B - T (s) : p5 / p25 / médiane / p75 / p95 | -99,9 / -39,8 / -23,9 / -13,3 / -0,2 ; \|écart\| médian 25,0, p90 67,3 ; à 5 s : 6,14 % ; à 30 s : 58,19 % ; au-delà de 30 s : 579 |
| Lequel est le plus proche de l'API | présence A : 1 272 (91,84 %) ; dernier corps B : 78 (5,63 %) ; égalité 35 |
| Poses déployées (hors grenades, poseur nommé) | 9 178 |
| Poses déployées hors de toute vie de leur poseur, après la fin d'une de ses vies | 7, dans 7 rejeux, toutes des murs ; écart 1 à 3 images (médiane 1) ; 0 sans vie du poseur |

Les deux estimations sont plus courtes que l'API pour les partants (écart médian négatif) ; la présence publiée est
la plus proche pour neuf partants sur dix.

## Piste G — zones

Script : `extract.js` + `analyse.js G` (`sortie_G.txt`).

| Grandeur | Valeur |
|---|---|
| Rejeux avec `zoneStates` | 125 : méthode `captures+geometry` 77 ; `designator+geometry` 48 |
| Rejeux de Bases (zones + variante Strongholds au registre) | 77 (Strongholds:Arena 68, Arena:Strongholds 8, Ranked:Strongholds 1) ; les 48 autres sont des collines (KOTH) |
| Zones de ces 77 rejeux | 231 (trois par rejeu) |
| Répartition de `letterRank` | 0 : 77 ; 1 : 77 ; 2 : 77 (chaque rejeu porte exactement les rangs 0, 1, 2) |
| `coverage.zones.letters` | 3 pour les 77 |

## Piste H — canaux

Script : `extract.js` + `analyse.js H` (`sortie_H.txt`). Genres publiés dans `stances[]` sur tout le parc :
`sprint`, `jumpDerived`, `clamber`, `crouch`, `slide` ; `mobility` : aucun.

| Genre | Total publié | Médiane par rejeu | p90 | Rejeux à zéro |
|---|---|---|---|---|
| Sauts dérivés (`jumpDerived`) | 655 702 | 371 | 631 | 0 / 1 455 |
| Sprints (`sprint`) | 1 029 888 | 583 | 989 | 1 |
| Accroupissements (`crouch`) | 1 129 | 0 | 0 | 1 448 (donc 7 rejeux en portent) |
| Glissades (`slide`) | 1 039 | 0 | 0 | 1 448 (7 rejeux) |
| Franchissements (`clamber`) | 65 404 | 40 | 81 | 3 |
| Actions de mobilité (`mobility`) | 0 | 0 | 0 | 1 455 |

Balayage des états lancé sur 1 455 rejeux ; composants déclarés absents : 0 ; épisodes de saut lus
2 499 582 ; sauts dérivés (compteur de couverture) 656 151.

Couverture de la vue de contrôle (`coverage.continuousFire`), par build : part de paquets fermés et part de trous
(sommes de paquets).

| Build | Rejeux | Paquets | Part fermée | Part de trous |
|---|---|---|---|---|
| HI_1_13_0 | 1 200 | 38 534 227 | 99,94 % | 0,06 % |
| HI_1_12_0 | 145 | 4 190 208 | 99,97 % | 0,03 % |
| HI_1_11_0 | 57 | 1 275 251 | 99,77 % | 0,23 % |
| HI_1_10_0 | 31 | 797 686 | 96,13 % | 3,87 % |
| HI_1_8_0 | 13 | 372 667 | 90,46 % | 9,54 % |
| HI_1_9_0 | 3 | 68 000 | 98,68 % | 1,32 % |
| sans build | 5 | 106 011 | 1,04 % | 98,96 % |
| HI_1_4_1 | 1 | 11 130 | 6,93 % | 93,07 % |

Rechargements, bouton de saut, commande de déplacement, répliques : aucun champ publié aujourd'hui (rien à
mesurer « avant » que zéro).

## Piste I — temps forts de type « mode »

Script : `analyse_I.js` sur `evenements_mode.tsv` ; distinct par `q` sur la copie (`sortie_I_distinct.txt`).
Événements `highlight_events.event_type = 'mode'`, code = `type_hint`.

| Grandeur | Valeur |
|---|---|
| Événements de type mode / matchs distincts | 23 683 / 527 |
| Code 10 / sans code | 21 353 / 2 330 |
| `medal_value` 2 / sans valeur | 16 925 / 6 758 |

Par catégorie de mode : Assassin (Arena) 17 089 ; BTB 5 224 ; Ranked 463 ; Other 300 ; Husky Raid 255 ; Super
Fiesta 180 ; Firefight 147 ; Fiesta 25. Par famille de variante (lecture du nom) : CTF 11 901 ; Bases 5 791 ; KOTH
3 357 ; Slayer 790 ; Total Control 721 ; Oddball 641 ; autres 482. Variantes les plus fournies : CTF:Arena 6 655 ;
Strongholds:Arena 5 127 ; KOTH:Arena 3 066 ; CTF:Arena Neutral Flag 1 467 ; BTB Heavies:CTF 1 172 ; BTB:CTF
1 159 ; BTB:Fiesta CTF 674. Le détail complet (58 lignes mode x variante x code x valeur) est dans
`C:/t/verite/evenements_mode.tsv`.

## Piste J — poids et clés racine

Script : `extract.js` + `analyse.js J` (`sortie_J.txt`).

| Grandeur | Valeur |
|---|---|
| Poids compressé d'un rejeu : médiane | 1 105 992 octets (environ 1,05 Mio) ; moyenne 1 363 350 ; p90 1 664 945 ; min 41 044 ; max 8 579 753 ; total 1 983 674 539 |
| Clés racine du document : médiane / min / max | 47 / 32 / 57 (le premier rejeu en porte 50) ; union sur le parc : 69 clés |

## Récapitulatif

| Grandeur | Valeur avant | Piste qui doit la faire bouger | Sens attendu |
|---|---|---|---|
| Rejeux datés par l'horloge / par le premier mouvement | 1 455 / 1 447 | A | 1 455 / 1 455 (mouvement : retiré ou sans refus) |
| Rejeux sans coup d'envoi du mouvement | 8 | A | 0 |
| Rejeux à plus de 3 s d'écart entre mouvement et horloge | 22 | A | 0 (une seule source) |
| Rejeux « complets » (score) | 944 | B | baisse (les rejeux en écart passent « incomplets ») |
| Complets dont un joueur diffère du tableau | 43 (44 avec joueur sans score compté 0) | B | 0 |
| Accord exact somme des points / `personal_score`, complets | 98,42 % (8 065 / 8 194) | B (B.3) | hausse (BTB build 1.11 et 100 points) |
| Idem, BTB complets ; build 1.11 complets | 78,07 % ; 69,33 % | B (B.3) | hausse |
| Accord exact, tous les rejeux (build 1.10 / 1.9) | 95,29 % (12,70 % / 28,57 %) | B (B.1 : verdict) | inchangé (le verdict dit l'incomplet) ; hausse seulement si B.3 aboutit |
| Accord exact des compteurs nommés à l'API (lus) : éliminations / morts / assistances | 99,96 % / 99,93 % / 99,92 % | C | inchangé, aucune baisse |
| Idem : tête / corps à corps / grenades / armes lourdes / série | 99,85 % / 99,76 % / 99,63 % / 99,97 % / 99,77 % | C | hausse (grenades surtout) |
| Couples joueur-manche, compteur nommé > éliminations | 27 (26 rejeux) | C (C.2) | 0 |
| Séries `maxSpree` > éliminations | 23 joueurs (max 29 812) | C (C.2) | 0 |
| Joueurs à exactement 16 grenades (API : jamais 16) | 19 | C (C.3) | 0 |
| Joueurs « suffisants » (tirs) | 46,03 % (6 469 / 14 053) | D (D.3, DD-2) | hausse |
| Tirs / touches / dégâts à 10 % de l'API, tous joueurs | 59,74 % / 58,41 % / 64,77 % | D (D.3) | hausse |
| Idem pour les armes à plasma et projectiles (Needler : tirs / touches) | 19,27 % / 40,06 % | D (D.3) | hausse |
| Portage du drapeau à 2 s de l'API, tous / <= 8 / > 8 joueurs | 88,11 % / 91,71 % / 85,50 % | E | hausse, surtout au-delà de 8 |
| Faux porteur de `390b1de5` | 94,3 s publiées, API 0 | E (E.1, E.2) | 0 |
| Portages manquants au-delà de 8 joueurs (film < API de plus de 2 s) | 120 | E (E.3) | baisse |
| Présence publiée : écart médian à l'API (partants) / part à 5 s | -16,5 s / 14,08 % | F (F.2, F.3) | rapprochement si un fait date mieux le départ |
| Fin du dernier corps : écart médian à l'API / part à 5 s | -23,9 s / 6,14 % | F | sans objet (pas publié) |
| Poses de mur déployées après la fin de vie de leur poseur | 7 | F (F.1, DD-3) | 7 poses changent de crédit au plus |
| Rejeux de Bases ; rangs de `letterRank` | 77 ; {0, 1, 2} (un chacun) | G | champ `letter` ajouté ; rang inchangé |
| Sauts dérivés / sprints publiés (total ; médiane par rejeu) | 655 702 (371) / 1 029 888 (583) | H | inchangé (nouveaux genres en plus) |
| Accroupissements / glissades publiés (rejeux concernés) | 1 129 / 1 039 (7 rejeux) | H (H.4) | hausse si le diagnostic trouve une lecture |
| Rechargements, bouton de saut, commande de déplacement publiés | 0 | H (H.1, H.3) | > 0 |
| Part fermée de la vue de contrôle, `HI_1_13_0` | 99,94 % | H (H.2) | inchangé |
| Événements de mode / matchs / code 10 | 23 683 / 527 / 21 353 | I | inchangé (décompte, aucun code de production) |
| Poids compressé médian d'un rejeu | 1 105 992 octets | H (Gate H : < 2 % de plus), J | hausse limitée à 2 % |
| Clés racine du document : médiane / union | 47 / 69 | H, J (J.1, J.2) | hausse (champs neufs) |

## Écarts avec la section 1 du plan et réserves

- Piste B : 43 ou 44 rejeux selon la convention. La section 1 compte un joueur sans score de tableau pour 0 (44) ;
  les écarts de fond (12 BTB 1.11, 4 Assaut, 28 autres) se retrouvent.
- Piste C : le « 27 sur 14 026 » de la section 1 rapporte des couples joueur-manche aux joueurs ; les couples
  lus sont 14 491. 27, 23 et 19 se retrouvent exactement. Fait nouveau : l'API n'a jamais 16 pour ces 19 joueurs.
- Piste D : 46,03 % retrouvé (6 469 / 14 053).
- Piste E : 186 rejeux à drapeau retrouvés ; les « 9 films de Grande bataille à drapeau » de la section 1
  deviennent 11 par `mode_category = BTB` (définition différente) ; 94 ont un roster de plus de 8.
- Piste F : 1 385 partants comparés (la section 1 en annonce 1 383) ; 7 murs, 9 178 poses : retrouvés.
- Piste H : « aucun accroupissement ni glissade sur 200 rejeux » : sur les 1 455, 7 rejeux en portent (1 129 et
  1 039 entrées) ; aucune action de mobilité.
- Piste G : 77 rejeux de Bases confirmés ; le rang n'a que trois valeurs (une par zone).
- Cinq rejeux sans build n'ont aucune jointure joueur (identiques aux « 5 rejeux sans build » du journal de
  découvertes).
- Non mesuré : coup d'envoi contre le premier événement daté de l'API sur les 22 rejeux en désaccord et 50
  autres (demandé par l'item 0.4 du plan, mais déjà tranché ailleurs et écarté de cette consigne) ; rechargements,
  bouton de saut, commande de déplacement et répliques (pas encore publiés : valeur avant = 0, pas de mesure) ;
  « tirs et touches du témoin libfilmshell » (D.2 : hors de cet item) ; les 14 films de Grande bataille de la
  section 1, remplacés par `mode_category = BTB` (11).
