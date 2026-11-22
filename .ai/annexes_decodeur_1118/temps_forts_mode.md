# Decompte des temps forts « mode » (lecture seule)

Mesure faite le 2026-11-18 sur la copie `C:/t/verite/shared_matches_v2.duckdb`, rejeux du cache `data/cache/replays/halo_infinite` (lecture seule). Aucun fichier du depot touche.

## 1. Source et volume

- Table : `highlight_events` (colonnes : `id`, `match_id`, `event_type`, `time_ms`, `xuid`, `type_hint`, `raw_json`), jointure sur `match_registry` (`match_id`, `game_variant_name`, `mode_category`).
- Type « mode » : `event_type = 'mode'`. Le code est la colonne `type_hint` (octet 47 du bloc event ; `typeHintMode = 10` dans le decodeur, `internal/grammar/highlight_events.go`).
- Total : **23683 evenements** de type « mode », sur **527 matchs**.
- Codes presents : code 10 : 21353 evenements / 484 matchs ; code NULL : 2330 evenements / 76 matchs.
- Le code `NULL` (aucun `type_hint`, `raw_json` vide) : origine non etablie (voir section 5).
- `mode_category` du registre n'est PAS utilisable comme categorie de mode : il vaut « Assassin » pour Slayer, CTF et Strongholds. La categorie est donc derivee de `game_variant_name` : premier mot-cle trouve dans l'ordre Total Control, Extraction, Oddball, Strongholds, KOTH, CTF, Slayer ; sinon BTB (autres) si le nom contient BTB ; reste = autres. Les variantes BTB portant un mot-cle (« BTB:CTF », « BTB Heavies:Slayer ») sont rangees sous ce mode.

## 2. Par categorie de mode et par code

« Matchs » = matchs de la categorie ayant au moins un evenement de ce code. « Mediane » = mediane du nombre d'evenements par match, sur ces matchs seuls.

| Categorie | Code | Evenements | Matchs | Mediane par match |
|---|---|---:|---:|---:|
| Slayer (138 matchs au total) | 10 | 711 | 114 | 4 |
| Slayer (138 matchs au total) | NULL | 79 | 31 | 2 |
| Bases (Strongholds) (77 matchs au total) | 10 | 5241 | 75 | 60 |
| Bases (Strongholds) (77 matchs au total) | NULL | 550 | 10 | 48.5 |
| Roi de la colline (49 matchs au total) | 10 | 3357 | 49 | 67 |
| Drapeau (CTF) (236 matchs au total) | 10 | 10230 | 219 | 45 |
| Drapeau (CTF) (236 matchs au total) | NULL | 1671 | 34 | 47 |
| Balle (Oddball) (7 matchs au total) | 10 | 641 | 7 | 87 |
| Extraction (2 matchs au total) | 10 | 303 | 2 | 151.5 |
| Total Control (10 matchs au total) | 10 | 691 | 10 | 47.5 |
| Total Control (10 matchs au total) | NULL | 30 | 1 | 30 |
| BTB (autres) (6 matchs au total) | 10 | 150 | 6 | 15 |
| autres (2 matchs au total) | 10 | 29 | 2 | 14.5 |

## 3. Slayer : croisement avec le rejeu

Echantillon : 30 matchs tires au hasard (graine 20261118) parmi 137 matchs Slayer ayant un rejeu et au moins un evenement « mode » (variantes BTB a Slayer comprises ; ce sont des matchs a mort). Lus : 30 ; ecartes : 0 (identifiant de rejeu different), 0 (horloge absente). Evenements « mode » examines : 180.

**Horloge.** Les temps forts et le rejeu sont sur la meme horloge (ms depuis le debut du film, au pas de trame) : le temps d'un evenement « mode » est compare directement au temps de chaque genre publie par le rejeu, sans decalage.
- Trame vers ms : `originMs + trame * frameIntervalMs` pour les actions de score (`scoreActions[].t`), les debuts de reapparition (`respawnCountdowns[].from`) et la fin de vie des vehicules detruits (`vehicles[].tEnd`, repli `t1`).
- Medailles : `medals[].ms` directement (deja en ms).
- Controle de l'horloge : 1585 medailles dans les temps forts de ces matchs ; 1461 appariees a une medaille du rejeu (meme joueur, meme nom, a 1 s pres) ; 124 sans contrepartie. Ecart temps fort - rejeu : 0 a 0 ms, mediane 0 ms.

**Fenetre** : ±2 s. Un evenement peut coincider avec plusieurs genres ; les pourcentages ne s'additionnent donc pas. « Aucun » = aucun des quatre genres dans la fenetre.
**Temoin** : meme comptage avec chaque instant decale de 60 a 120 s (sens tire au hasard), pour mesurer les coincidences par pur hasard.

| Code | Evenements | Vehicule detruit | Score (scoreActions) | Medaille | Mort (reapparition) | Aucun des quatre |
|---|---:|---:|---:|---:|---:|---:|
| 10 | 160 | 110 (68.8 %) | 160 (100.0 %) | 79 (49.4 %) | 140 (87.5 %) | 0 (0.0 %) |
| NULL | 20 | 15 (75.0 %) | 20 (100.0 %) | 6 (30.0 %) | 16 (80.0 %) | 0 (0.0 %) |

Temoin (instants decales de 60 a 120 s) :

| Code | Evenements | Vehicule | Score | Medaille | Mort | Aucun |
|---|---:|---:|---:|---:|---:|---:|
| 10 | 160 | 8.1 % | 49.4 % | 30.0 % | 51.3 % | 46.9 % |
| NULL | 20 | 0.0 % | 20.0 % | 5.0 % | 20.0 % | 80.0 % |

## 4. Trois exemples par code (Slayer, echantillon de la section 3)

### Code 10

- Match 06c48289, instant 89824 ms (1:30) : coincide avec mort+score ; publie : score:killPersonalScore, score:ghostDestroyResponse, mort:reapparition.
- Match 06c48289, instant 204374 ms (3:24) : coincide avec medaille+mort+score ; publie : score:assistPersonalScore, score:markAssistPersonalScore, score:killPersonalScore, score:ghostDestroyResponse.
- Match 06c48289, instant 407246 ms (6:47) : coincide avec medaille+mort+score+vehicule ; publie : vehicule:ghost, score:ghostDestroyResponse, score:killPersonalScore, score:killPersonalScore.

### Code NULL

- Match 247cc763, instant 395022 ms (6:35) : coincide avec mort+score ; publie : score:scorpionDestroyResponse, score:killPersonalScore, mort:reapparition.
- Match 247cc763, instant 483118 ms (8:03) : coincide avec mort+score+vehicule ; publie : vehicule:wraith, score:empAssistPersonalScoreResponse, score:killPersonalScore, score:wraithDestroyResponse.
- Match 3028e64d, instant 181342 ms (3:01) : coincide avec medaille+mort+score+vehicule ; publie : vehicule:scorpion, score:scorpionDestroyResponse, score:killPersonalScore, medaille:Killjoy.

## 5. Ce qui n'a pas pu etre etabli

- **Code NULL** : 2330 evenements « mode » sans `type_hint` ni `raw_json`, sur 76 matchs ; le depot ne dit pas d'ou ils viennent (ecriture ancienne ou autre source). Non rattaches a un code du decodeur.
- **Destruction de vehicule** : le temps retenu est `tEnd` (repli `t1`) du vehicule qui porte un champ `destruction`. Cette correspondance est observee sur un rejeu (les `*DestroyResponse` tombent au meme instant), non documentee par le decodeur.
- **Mort** : le debut du compte a rebours de reapparition (`respawnCountdowns[].from`) sert de marque de mort ; ce n'est pas un champ « mort » publie.
- **Sens du code** : le decompte ne dit pas ce que le jeu fait a chaque code ; il dit seulement ce qui coincide dans le rejeu. Le code 10 est le seul code de type « mode » present dans la base ; d'autres codes n'ont pas ete observes.
- **Joueur** : un evenement « mode » porte un xuid, mais le decompte ne l'a pas exploite (pas de rattachement a un joueur, aucun identifiant ecrit).
- **Echantillon** : 30 matchs Slayer seulement ; les tailles par code (voir section 3) limitent la precision des parts.
- **Densite** : les actions de score sont presque continues en Slayer (plusieurs par dizaine de secondes) ; une fenetre de ±2 s coincide donc presque toujours avec une action de score. Le temoin decale (section 3) le montre ; la colonne score ne discrimine pas.
- **Non-exclusivite** : une meme action (ex. `*DestroyResponse`) compte a la fois comme score et, si elle coincide, comme destruction ; la colonne « aucun » ne le corrige pas.



---

# Decompte exact des temps forts « mode » par joueur (lecture seule)

Mesure du 2026-11-18. Source : `C:/t/verite/shared_matches_v2.duckdb` (copie) et rejeux `data/cache/replays/halo_infinite` (lecture seule). Aucun fichier du depot touche. Aucun identifiant de joueur ecrit ; les comparaisons par joueur se font en memoire.

## Methode et horloge

- Evenements « mode » : 23683 sur 527 matchs ; 521 matchs ont un fichier de rejeu ; 521 lus (ecartes : 0 identifiant different, 0 sans horloge).
- Trame vers ms : `originMs + trame * frameIntervalMs` pour `scoreActions[].t`, `vehicles[].tEnd` (repli `t1`), `zoneStates[].spans[].t0`, `flagCarries[].spans[].t0`, `skullCarries[].t0`. Ms directs : `medals[].ms`, `objectives[].timeMs`, `bombEvents[].timeMs`.
- Quantification du rejeu, mesuree sur les medailles (ms reel moins `origin + trame*pas`) : 28398 medailles ; min 0 ms, mediane 16 ms, max 32 ms ; 28398 dans ±33 ms. **Les actions de score n'ont pas de ms propre** : leur instant est au pas d'une trame (`frameIntervalMs`).
- Fenetre utilisee pour les etapes 2 et 3 : **trame signee [0, 40] ms** (instant de l'evenement « mode » moins instant de l'action de score). Pour les calques de l'etape 4 : ±33 ms.

## 1. Distribution de l'ecart au score le plus proche du MEME joueur

| Ecart | Evenements « mode » |
|---|---:|
| 0 ms | 12 (0.1 %) |
| 1-33 ms | 16117 (68.1 %) |
| 34-100 ms | 2469 (10.4 %) |
| 101-500 ms | 4 (0.0 %) |
| >500 ms | 71 (0.3 %) |
| aucune action de score de ce joueur | 4717 (19.9 %) |

Ecarts exacts (ms) entre 0 et 200, nombre d'evenements :

0:12 1:9 2:22 3:48 4:171 5:338 6:412 7:532 8:566 9:591 10:538 11:573 12:528 13:576 14:566 15:640 16:630 17:574 18:523 19:544 20:561 21:549 22:502 23:512 24:530 25:563 26:550 27:552 28:614 29:541 30:574 31:555 32:602 33:531 34:550 35:513 36:589 37:395 38:197 39:142 40:55 41:16 42:4 43:4 45:2 70:1 91:1 108:1

- Code 10 : 21353 evenements, dont 16238 apparies a la fenetre [0, 40] ms (76.0 %).
- Code NULL : 2330 evenements, dont 2258 apparies a la fenetre [0, 40] ms (96.9 %).

## 2. Par categorie de mode et nom de reponse de l'action appariee (fenetre de trame signee [0, 40] ms, tous les matchs avec rejeu)

### Code 10

| Nom de reponse | Slayer | Bases (Strongholds) | Roi de la colline | Drapeau (CTF) | Balle (Oddball) | Extraction | Total Control | BTB (autres) | autres | Total |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| areaCapturedContributingPlayerResponse | 0 | 4197 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 4197 |
| onHillControlResponse | 0 | 0 | 2824 | 0 | 0 | 0 | 0 | 0 | 0 | 2824 |
| flagStealScore | 0 | 0 | 0 | 1809 | 0 | 0 | 0 | 0 | 7 | 1816 |
| flagPickupScore | 0 | 0 | 0 | 1335 | 0 | 0 | 0 | 0 | 2 | 1337 |
| flagReturnScore | 0 | 0 | 0 | 1186 | 0 | 0 | 0 | 0 | 2 | 1188 |
| flagRunnerKillScore | 0 | 0 | 0 | 1109 | 0 | 0 | 0 | 0 | 5 | 1114 |
| areaSecuredPlayerResponse | 0 | 917 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 917 |
| flagCaptureScore | 0 | 0 | 0 | 701 | 0 | 0 | 0 | 0 | 3 | 704 |
| flagCaptureAssist | 0 | 0 | 0 | 635 | 0 | 0 | 0 | 0 | 2 | 637 |
| ballControlResponse | 0 | 0 | 0 | 0 | 306 | 0 | 0 | 0 | 0 | 306 |
| onHillSoreResponse | 0 | 0 | 271 | 0 | 0 | 0 | 0 | 0 | 0 | 271 |
| ghostDestroyResponse | 93 | 11 | 4 | 15 | 0 | 0 | 0 | 0 | 0 | 123 |
| ballPickedUpCarrierPersonalScoreResponse | 0 | 0 | 0 | 0 | 140 | 0 | 0 | 0 | 0 | 140 |
| empAssistPersonalScoreResponse | 106 | 0 | 0 | 5 | 0 | 0 | 0 | 0 | 0 | 111 |
| killPersonalScore | 86 | 1 | 1 | 12 | 0 | 0 | 0 | 0 | 0 | 100 |
| bansheeDestroyResponse | 88 | 0 | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 90 |
| carrierKilledPlayerResponse | 0 | 0 | 0 | 0 | 84 | 0 | 0 | 0 | 0 | 84 |
| falconDestroyResponse | 55 | 0 | 0 | 6 | 0 | 0 | 0 | 0 | 3 | 64 |
| waspDestroyResponse | 45 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 45 |
| warthogDestroyResponse | 46 | 0 | 0 | 1 | 0 | 0 | 0 | 0 | 0 | 47 |
| razorbackDestroyResponse | 24 | 0 | 0 | 5 | 0 | 0 | 0 | 0 | 2 | 31 |
| gungooseDestroyResponse | 24 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 25 |
| scorpionDestroyResponse | 18 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 18 |
| wraithDestroyResponse | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 16 |
| chopperDestroyResponse | 14 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 14 |
| rockethogDestroyResponse | 9 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 9 |
| phantomDestroyResponse | 5 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 5 |
| assistPersonalScore | 1 | 1 | 1 | 1 | 0 | 0 | 0 | 0 | 0 | 4 |
| mongooseDestroyResponse | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 | 1 |
| **aucune action de score appariee** | 73 | 114 | 107 | 3410 | 111 | 303 | 691 | 12 | 1 | 4822 |
| (matchs avec rejeu de la categorie) | 137 | 77 | 48 | 236 | 7 | 2 | 10 | 2 | 2 | |

### Code NULL

| Nom de reponse | Slayer | Bases (Strongholds) | Roi de la colline | Drapeau (CTF) | Balle (Oddball) | Extraction | Total Control | BTB (autres) | autres | Total |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| areaCapturedContributingPlayerResponse | 0 | 441 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 441 |
| flagStealScore | 0 | 0 | 0 | 413 | 0 | 0 | 0 | 0 | 0 | 413 |
| flagPickupScore | 0 | 0 | 0 | 352 | 0 | 0 | 0 | 0 | 0 | 352 |
| flagReturnScore | 0 | 0 | 0 | 301 | 0 | 0 | 0 | 0 | 0 | 301 |
| flagRunnerKillScore | 0 | 0 | 0 | 281 | 0 | 0 | 0 | 0 | 0 | 281 |
| areaSecuredPlayerResponse | 0 | 106 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 106 |
| flagCaptureScore | 0 | 0 | 0 | 134 | 0 | 0 | 0 | 0 | 0 | 134 |
| flagCaptureAssist | 0 | 0 | 0 | 149 | 0 | 0 | 0 | 0 | 0 | 149 |
| ghostDestroyResponse | 20 | 0 | 0 | 3 | 0 | 0 | 0 | 0 | 0 | 23 |
| empAssistPersonalScoreResponse | 6 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 6 |
| killPersonalScore | 15 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 15 |
| bansheeDestroyResponse | 16 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 16 |
| waspDestroyResponse | 3 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 3 |
| gungooseDestroyResponse | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 2 |
| scorpionDestroyResponse | 9 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 9 |
| wraithDestroyResponse | 4 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 4 |
| chopperDestroyResponse | 2 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 2 |
| rockethogDestroyResponse | 1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 1 |
| **aucune action de score appariee** | 1 | 3 | 0 | 38 | 0 | 0 | 30 | 0 | 0 | 72 |
| (matchs avec rejeu de la categorie) | 137 | 77 | 48 | 236 | 7 | 2 | 10 | 2 | 2 | |

## 3. Sens inverse : pour chaque action de score, un evenement « mode » du meme joueur dans la fenetre [0, 40] ms apres son instant ?

| Nom de reponse | Actions de score (ces rejeux, tous joueurs) | Avec evenement « mode » du meme joueur | Part |
|---|---:|---:|---:|
| areaCapturedContributingPlayerResponse | 3960 | 3956 | 99.9 % |
| onHillControlResponse | 2796 | 2787 | 99.7 % |
| flagStealScore | 2073 | 1952 | 94.2 % |
| flagPickupScore | 1729 | 1472 | 85.1 % |
| flagReturnScore | 1402 | 1326 | 94.6 % |
| flagRunnerKillScore | 1353 | 1212 | 89.6 % |
| areaSecuredPlayerResponse | 886 | 878 | 99.1 % |
| flagCaptureScore | 729 | 726 | 99.6 % |
| flagCaptureAssist | 673 | 664 | 98.7 % |
| ballControlResponse | 306 | 306 | 100.0 % |

## 4. Evenements « mode » sans action de score appariee : coincidence (±33 ms pour les calques ; instants exacts en ms) avec d'autres calques

Calques utilises : `medals` (meme joueur), `vehicles[].destruction` (fin de vie, joueur = auteur de la destruction), `zoneStates` (changement d'etat de zone, SANS joueur), `flagCarries` (changement d'etat ou de porteur, sans joueur testé), `objectives` (meme joueur), `bombEvents` (meme joueur), `skullCarries` (debut de porteur, meme joueur).

| Categorie | Sans score apparie | medaille | fin de vehicule | zone (sans joueur) | drapeau (changement, sans joueur) | objectif (stat) | bombe | tete (porteur) | Aucun calque |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Slayer | 74 | 24 (32.4 %) | 26 (35.1 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 38 (51.4 %) |
| Bases (Strongholds) | 117 | 5 (4.3 %) | 0 (0.0 %) | 74 (63.2 %) | 0 (0.0 %) | 114 (97.4 %) | 0 (0.0 %) | 0 (0.0 %) | 3 (2.6 %) |
| Roi de la colline | 107 | 0 (0.0 %) | 0 (0.0 %) | 5 (4.7 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 102 (95.3 %) |
| Drapeau (CTF) | 3448 | 511 (14.8 %) | 422 (12.2 %) | 0 (0.0 %) | 30 (0.9 %) | 18 (0.5 %) | 0 (0.0 %) | 0 (0.0 %) | 2693 (78.1 %) |
| Balle (Oddball) | 111 | 7 (6.3 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 104 (93.7 %) |
| Extraction | 303 | 195 (64.4 %) | 13 (4.3 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 103 (34.0 %) |
| Total Control | 721 | 34 (4.7 %) | 24 (3.3 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 674 (93.5 %) |
| BTB (autres) | 12 | 0 (0.0 %) | 2 (16.7 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 10 (83.3 %) |
| autres | 1 | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 0 (0.0 %) | 1 (100.0 %) |

## 5. Code NULL face au code 10

Matchs ayant les deux codes : 33. Evenements NULL de ces matchs : doublon d'un evenement code 10 du meme joueur a ±33 ms : 1284 ; autres : 0. Ecart des doublons : mediane 0 ms.

| Groupe de matchs | Matchs | Plus ancien | Mediane | Plus recent |
|---|---:|---|---|---|
| NULL seul (sans code 10) | 43 | 2026-06-08 19:52:27.703 | 2026-07-17 13:02:35.39 | 2026-09-01 19:43:14.496 |
| code 10 seul | 451 | 2023-04-15 21:09:54.446 | 2025-12-20 18:51:35.444 | 2026-11-10 21:23:42.27 |
| les deux | 33 | 2026-02-05 15:16:38.658 | 2026-04-23 19:32:08.865 | 2026-05-24 21:26:57.539 |

Les 1 046 evenements NULL des 43 matchs sans code 10 n'ont par construction aucun doublon ; ils ne sont compares a aucun code 10. Sur les 33 matchs a deux codes, 1 284 NULL sur 1 284 ont un code 10 du meme joueur a ±33 ms (pas de verification 1 pour 1 ici).

Dates : `match_registry.start_time` pour tous les matchs de la base ayant au moins un evenement « mode » (rejeu ou non).

## 1 bis. Test signe : l'ecart n'est pas un pic fixe, c'est une plage d'une trame

Ecart signe (temps « mode » moins instant de l'action de score du meme joueur le plus proche), par pas de 4 ms, entre -100 et +100 ms :

-92:1 -72:1 -44:1 -36:2 -32:9 -28:1 -24:1 -20:9 -16:7 -12:10 -8:17 -4:18 **0:76 4:1441 8:2250 12:2303 16:2268 20:2117 24:2194 28:2277 32:2192 36:1323 40:78** 44:2

Par code, ecart dans [0, 33[ ms (une trame) : code 10 : 13608 sur 16334 apparies a moins de 100 ms (83 %), 71 negatifs ; code NULL : 1918 sur 2264 (85 %), 6 negatifs. Il n'y a pas de valeur fixe unique : les evenements « mode » tombent 4 a 36 ms apres le debut de la trame de l'action, avec une coupure nette vers 40 ms. Une fenetre absolue de ±33 ms coupait donc la queue (34 a 40 ms, environ 2 400 evenements) ; la fenetre [0, 40] ms de la section 2 la couvre.

Le pas de quantification des medailles (ecart mediane 16 ms) explique le decalage : l'instant d'une action de score est connu a une trame pres.

## 6. Ce qui n'a pas pu etre etabli

- Une action de score n'a qu'un instant au pas de trame : l'ecart 0 ms n'est pas atteignable, la bonne mesure est la plage [0, 40] ms.
- La fenetre [0, 40] ms est choisie d'apres la distribution observee (plage d'une trame, coupure a 40 ms) ; elle n'a pas ete calibree sur une verite terrain.
- La liaison « une action de score / un evenement » est un appariement au plus proche : une action peut etre appariee a plusieurs evenements et inversement ; les doublons ne sont pas comptes a part.
- Un evenement « mode » peut avoir un joueur qui n'apparait pas dans le rejeu : il compte alors comme « aucune action de score de ce joueur ».
- Les calques de zone et de drapeau sont testes sans joueur.
- Le sens des codes et des noms de reponse n'est pas tranche par ce decompte.

