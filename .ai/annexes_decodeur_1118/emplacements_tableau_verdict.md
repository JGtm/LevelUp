# C.5 : emplacements du tableau des scores, valeurs écrites par le film et accord avec l'API

Mesure du 2026-11-22, sans code de production. 58 films, 559 joueurs appariés (sur 777 lignes API des mêmes matchs).
Données et scripts : `<poste>/Downloads/Scripts/dec-v150-recup/c5/` (tableaux bruts : `verdict_tableau_1manche.tsv`,
`verdict_tableau_multi.tsv`, `rythme_par_nom.tsv`, `famille_par_nom.tsv`, `figees_*.tsv`).

## Réponse courte

1. Le nom d'un emplacement ne prouve pas sa valeur, mais Den a à moitié raison. Il y a TROIS régimes :
   - **vivant** : l'emplacement est réécrit à chaque événement, valeur finale égale à l'API (éliminations, morts, assistances, score
     personnel, tête, mêlée, grenade, arme lourde, série, et tous les compteurs d'objectif d'événement) ;
   - **traînant** : écrit, mais seulement quand son jumeau de la paire bouge ; la valeur est juste à l'instant de l'émission,
     périmée à la fin (tirs tirés, dégâts infligés, dégâts subis, KDA, durée de vie moyenne) ;
   - **jamais écrit en direct** : tirs au but et précision. Zéro émission réelle sur 58 films.
2. Le film porte pourtant une SECONDE copie du tableau, que notre décodeur ne lit pas (il s'arrête à la 28e ligne) et que le lecteur d'instantané de Den ne lit pas non plus (28 lignes) : les valeurs FIGÉES des manches terminées (composants 28 à 55 de l'archétype du tableau). Elles contiennent
   les tirs au but et la précision, exacts à la manche. Mais elles n'existent que pour les manches 0 à N-2 : la dernière manche d'un
   match, donc le match entier dans 48 films sur 58, n'est jamais figée dans le film.
3. Il n'existe donc, dans les 58 films, aucune valeur de fin de match pour tirs tirés, tirs au but, précision, dégâts infligés,
   dégâts subis, KDA ou durée de vie moyenne. Les instantanés du tableau (une ligne complète tous les 20 s environ) répètent les mêmes
   valeurs traînantes, jamais plus récentes que le flux de deltas.

## Étalonnage de la chaîne (matchs à une seule manche, 520 joueurs, 49 films)

Valeur finale d'un emplacement = plus longue suite non décroissante des émissions de la manche (les dernières émissions
sont parfois des zéros de remise à l'état initial : la « dernière valeur » brute donne 90 %, la suite filtrée 99 %).

| emplacement | joueurs à API > 0 | accord exact | observation |
|---|---|---|---|
| 6A tirs à la tête | 346 | **100,0 %** | |
| 8B éliminations à la grenade | 171 | **100,0 %** | |
| 10B réapparitions | 520 | pas de colonne API ; `réapparitions - morts API` vaut 0 (136 joueurs) ou 1 (382) : **518/520 = 99,6 %** | cohérent : une réapparition de départ de plus que de morts, sauf si le joueur est mort sans réapparaître avant la fin |
| 2A éliminations | 510 | 99,8 % | |
| 2B morts | 520 | 99,8 % | |
| 3A assistances | 470 | 99,4 % | |
| 8A mêlée | 270 | 99,6 % | |
| 9A arme lourde | 285 | 99,6 % | |
| 5B meilleure série | 510 | 99,2 % | |
| 1B score personnel | 517 | 99,0 % | |

La chaîne est bonne (99 % et plus). Les 0,2 à 1 % de désaccords viennent de joueurs mal appariés ou d'émissions manquantes.
Un témoin indépendant (le lecteur du flux de deltas de libfilmshell, Den, MIT ; code lu comme témoin, non recopié), passé sur 10
films (115 joueurs), donne exactement les mêmes valeurs finales traînantes et la même absence de tirs au but, et ses instantanés
ne dépassent presque jamais ses deltas (instantané > delta : 2 cas sur 1 610 couples joueur-emplacement, tous sur le KDA ; instantané plus ancien que le delta : 13 cas sur 115 pour le score personnel, 7 pour les éliminations).

## Tableau de verdict, par emplacement (matchs à une seule manche)

Légende : « écrit » = part des joueurs avec au moins une émission non nulle ; « API > 0 » = nombre de joueurs de valeur API non nulle ;
« exact » = égalité à la précision stockée sur ces joueurs ; « <= API » = part des joueurs où la valeur du film ne dépasse jamais l'API ;
« jumeau » = part des changements qui arrivent dans la même émission que le changement de l'autre valeur du composant.

### Emplacements communs (index 0 à 22)

| nom (index) | écrit | API > 0 | exact | <= API | jumeau | rythme mesuré |
|---|---|---|---|---|---|---|
| Score (0A) | 52 % | pas de colonne par joueur (`participants.score` = score personnel) | égal aux éliminations en Slayer 188/190 ; aux prises de drapeau en CTF 115/116 ; 14/16 aux explosions en Assaut ; zéro en Bases / Contrôle total (score d'équipe) | | 2 % | à chaque frag (94 %) |
| RoundsWon (0B) | 5 % : **artefact** (valeurs 16, 26, 9216, -23552) | colonne d'équipe seulement | **jamais écrit en direct** | | | voir valeurs figées |
| RoundsTied (1A) | 1 % : artefact | aucune | jamais écrit | | | |
| PersonalScore (1B) | 99,6 % | 517 | 99,0 % | 98,8 % | 0 % | à chaque récompense (59 % co-datées avec un frag) |
| Kills (2A) | 98 % | 510 | 99,8 % | 99,8 % | 5 % | à chaque frag |
| Deaths (2B) | 100 % | 520 | 99,8 % | 99,8 % | 5 % | à chaque mort (97 %) |
| Assists (3A) | 91 % | 470 | 99,4 % | 99,2 % | 100 % (avec KDA) | à chaque assistance |
| **KDA (3B)** | 91 % | 258 | **26,4 %** | 57,5 % | 100 % | **seulement à chaque assistance** ; valeur = éliminations - morts + assistances/3 à cet instant (2 015 émissions sur 2 017 à 0,01 près de la formule recalculée sur la même lecture) ; périmé dès qu'une élimination ou une mort suit la dernière assistance. Rapport film / API : p10 0,37, médiane 1,00, p90 1,35 |
| Suicides (4A) | 9 % | aucune colonne | non vérifiable | | | à la mort (90 %) |
| Betrayals (4B) | 2,5 % | aucune colonne | non vérifiable | | | |
| **AverageLifeDuration (5A)** | 84 % | 520 | **7,7 %** | 69,8 % | 100 % (avec 5B) | recalculée seulement quand la série bouge, donc à un frag ; rapport médian 0,91 |
| BestKillingSpree (5B) | 98 % | 510 | 99,2 % | 99,2 % | 39 % | à chaque frag |
| Headshots (6A) | 67 % | 346 | **100 %** | | 100 % | à chaque frag à la tête |
| **ShotsFired (6B)** | 67 % | 520 | **2,3 %** | **100 %** | 100 % (avec 6A) | **seulement à chaque frag à la tête (100 % co-daté avec un frag)** ; séries monotones 503/520 ; rapport médian 0,83 (p10 0,22) |
| **ShotsHit (7A)** | **0,2 %** (1 joueur, valeur négative : artefact) | 520 | **0** | | | **250 émissions du composant 7 sur 58 films, 244 à zéro, 6 négatives ou incohérentes ; aucune valeur réelle** |
| **AccuracyPercentage (7B)** | **0,2 %** (même artefact) | 520 | **0** | | | idem |
| MeleeKills (8A) | 52 % | 270 | 99,6 % | | 1 % | à chaque frag de mêlée |
| GrenadeKills (8B) | 33 % | 171 | 100 % | | 3 % | à chaque frag à la grenade |
| PowerWeaponKills (9A) | 55 % | 285 | 99,6 % | | 100 % (avec 9B) | à chaque frag à l'arme lourde |
| **DamageDone (9B)** | 55 % | 520 | **9,2 %** | **99,8 %** | 100 % | **seulement à chaque frag à l'arme lourde** ; rapport médian 0,80 (p10 0,25) ; zéro écrit pour 45 % des joueurs (pas d'arme lourde) |
| **DamageTaken (10A)** | 100 % | 520 | **42,7 %** | **99,6 %** | 100 % (avec 10B) | **seulement à chaque réapparition (100 % co-daté), 10 s en médiane après la mort** ; rapport médian 0,97 : seule la dernière vie manque |
| Spawns (10B) | 100 % | pas de colonne | voir étalonnage | | 90 % | à chaque réapparition |
| LoadoutXP (11A) | 98 % | pas de colonne | **égal à 100 x éliminations API : 518/520 (99,6 %)** | | 0 % | à chaque frag : redondant avec Kills |
| ObjectivesCompleted (11B) | 0,3 % : artefact | aucune | jamais écrit | | | |
| SpartanKills / Assists / Deaths (12A, 12B, 13A) | 99 / 92 / 100 % | aucune | égaux à Kills / Assists / Deaths : 353/354 | | | doublons |
| SpartanKDA (13B) | 100 % | aucune | recalculée aux morts (et non aux assistances) : égale à KDA seulement 37/354 | | 100 % | à chaque mort |
| AI*, Marine, Grunt, Jackal, Elite, Brute, Hunter, Skimmer, Sentinel, Boss (14A à 20A) | 0 à 1,3 % : artefacts | aucune | **jamais écrits en multijoueur** | | | |

### Emplacements propres à un mode (index 41 à 51) : le nom change avec le mode, l'index reste

Accord mesuré sur les matchs à une seule manche. « n » = joueurs de valeur API non nulle.

| mode | nom (index = 2 x composant + côté) | n | exact | rythme |
|---|---|---|---|---|
| CTF | FlagCaptureAssists (20B) | 16 | 100 % | événement |
| CTF | FlagCaptures (21A) | 25 | 96 % (2 écarts sur 126) | événement |
| CTF | FlagCarriersKilled (21B) | 41 | 97,6 % | événement, 98 % co-daté avec un frag |
| CTF | FlagGrabs (22A) | 42 | 100 % | événement |
| CTF | FlagReturnersKilled (22B) | 14 | 100 % | événement, 100 % avec un frag |
| CTF | FlagReturns (23A) | 29 | 100 % | événement |
| CTF | FlagSecures (23B) | 43 | 100 % | événement |
| CTF | FlagSteals (24A) | 44 | 100 % | événement |
| CTF | KillsAsFlagCarrier (24B) | 1 | 100 % (échantillon insuffisant) | frag |
| CTF | KillsAsFlagReturner (25A) | 14 | 100 % | frag |
| CTF | **FlagCarryTime (25B)** | 65 | **10,8 %** | **traînant : 8 joueurs écrits sur 126, seulement avec KillsAsFlagReturner** |
| Bases, Contrôle total | StrongholdCaptures (20B) | 114 | 100 % | événement |
| Bases, Contrôle total | StrongholdSecures (21A) | 40 | 100 % | événement |
| Bases, Contrôle total | StrongholdOffensiveKills (21B) | 68 | 100 % | frag |
| Bases, Contrôle total | StrongholdDefensiveKills (22A) | 59 | 100 % | frag |
| Bases, Contrôle total | **StrongholdOccupationTime (22B)** | 85 | **25,9 %** | écrit chaque seconde (intervalle médian 1 001 ms) mais la valeur reste sous l'API (médiane 0,93, jamais au-dessus) : autre définition ou arrêt avant la fin |
| Bases, Contrôle total | StrongholdScoringTicks (23A) | 22 | 100 % | chaque seconde pendant le contrôle |
| VIP | VipKills (20B) | 14 | 100 % | frag |
| VIP | VipAssists (21A) | 13 | 100 % | assistance |
| VIP | KillsAsVip (21B) | 12 | 100 % | frag |
| VIP | TimesSelectedAsVip (22A) | 16 | 100 % | à la sélection |
| VIP | BestKillingSpreeAsVip (22B) | 12 | 100 % | frag |
| VIP | LongestTimeAsVip (23A) | 16 | 93,8 % (68,6 contre 68,7 : 0,1 s) | toutes les 100 ms |
| VIP | TimeAsVip (23B) | 16 | 100 % | toutes les 100 ms |
| Assaut | Score (0A) | 4 | 14/16 joueurs | à l'explosion |
| Extraction (noms SUPPOSÉS par la famille : définition illisible, builds <= 1.11) | SuccessfulExtractions, ConversionsDenied, ConversionsCompleted, InitiationsDenied, InitiationsCompleted (20B à 22B) | 16 / 4 / 7 / 1 / 16 | 100 % | événement |
| Balle (Oddball) | Skull* (20B à 23A) | | **non établi** : tous les films Oddball du parc sont à plusieurs manches ; en multi-manches la chaîne tombe à 50-65 % (voir ci-dessous) | |
| KOTH, HuskyRaid, Total Control | pas d'emplacement de mode propre lu, hors ceux ci-dessus | | | |

### « Écrit ou jamais écrit » par mode, pour les emplacements cités

Cellule : part des joueurs écrits (au moins une émission non nulle) / accord exact avec l'API quand une colonne existe. Matchs à une manche.

| emplacement | Slayer | CTF | Bases | KOTH | Contrôle total | Assaut | VIP | Extraction | Husky Raid |
|---|---|---|---|---|---|---|---|---|---|
| ShotsFired | 71 / 4 | 65 / 0 | 67 / 3 | 65 / 0 | 63 / 3 | 87 / 0 | 75 / 0 | 59 / 4 | 50 / 0 |
| ShotsHit | 1 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| AccuracyPercentage | 1 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 | 0 / 0 |
| DamageDone | 59 / 11 | 59 / 7 | 54 / 8 | 22 / 0 | 62 / 14 | 53 / 0 | 31 / 13 | 39 / 2 | 80 / 40 |
| DamageTaken | 100 / 36 | 100 / 47 | 100 / 41 | 100 / 39 | 100 / 58 | 100 / 27 | 100 / 38 | 100 / 52 | 100 / 30 |
| KDA | 90 / 29 | 90 / 20 | 92 / 15 | 100 / 52 | 91 / 22 | 100 / 20 | 94 / 38 | 87 / 33 | 80 / 0 |
| AverageLifeDuration | 85 / 8 | 81 / 3 | 90 / 15 | 91 / 4 | 80 / 11 | 93 / 0 | 81 / 0 | 78 / 9 | 90 / 20 |
| RoundsWon | 7 (artefact) | 4 | 0 | 0 | 8 | 0 | 0 | 4 | 10 |
| LoadoutXP | 99 | 97 | 100 | 100 | 95 | 100 | 100 | 96 | 100 |

Constance par version du jeu (builds 1.8 à 1.13, 6 builds) : ShotsHit et Accuracy à 0 dans tous ; ShotsFired exact de 0 à 4 % partout ;
DamageTaken exact de 25 à 63 % selon le build (même mécanisme). Le comportement ne dépend pas du build.

## Réponse explicite pour chaque emplacement cité

- **ShotsFired (6B)** : écrit pour les joueurs qui ont au moins un frag à la tête (67 %) ; il n'est réécrit qu'à ces frags. Valeur finale = tirs
  tirés au dernier frag à la tête : jamais au-dessus de l'API (100 %), égale dans 2,3 % des cas. **Non juste en fin de match.**
- **ShotsHit (7A)** : **jamais écrit en direct** (zéro valeur réelle, 0 sur 58 films et 559 joueurs ; 6 émissions hors domaine ou négatives sur 250).
  Confirme Den. Mais il est écrit dans les valeurs figées (voir plus bas).
- **AccuracyPercentage (7B)** : idem ShotsHit. Dans les valeurs figées, précision = touches / tirs au centième : 118 couples joueur-manche
  sur 118 à 0,01 point près (69 à l'arrondi exact), touches <= tirs 118/118. Stockée en centièmes (précision 2).
- **DamageDone (9B)** : écrit seulement avec les frags à l'arme lourde. Jamais au-dessus de l'API (99,8 %), égal dans 9,2 % des cas. **Non juste en fin de match.**
- **DamageTaken (10A)** : écrit à chaque réapparition. Médiane de rapport 0,97 (seule la dernière vie est perdue) ; égal à l'API dans 42,7 % des cas
  (joueurs dont la dernière vie ne subit rien, ou qui ne meurent pas). **Non juste en fin de match.**
- **KDA (3B)** : écrit seulement quand les assistances changent ; la valeur est exacte à l'instant de l'émission (formule d'ADR 0006 recalculée : 99,9 %),
  périmée ensuite. Exact en fin de match dans 26 %. **Ne pas lire : recalculer à partir de 2A, 2B, 3A, qui sont exacts à 99 %.**
- **AverageLifeDuration (5A)** : écrit seulement avec la série (5B). Exact 7,7 % ; rapport médian 0,91, parfois au-dessus de l'API (30 %). **Non juste.**
- **RoundsWon (0B)** : **jamais écrit en direct**. Écrit dans les valeurs figées : 1 pour le vainqueur de chaque manche terminée. La victoire de la dernière
  manche n'y est jamais, donc le total de manches gagnées est faux d'une unité chez le vainqueur (1 + 0 contre 2 de l'API, constaté sur 5 films).
- **LoadoutXP (11A)** : écrit à chaque frag, vaut 100 x éliminations (518/520). Redondant, aucune colonne API propre.
- Emplacements de mode non nommés par notre décodeur : voir la liste des candidats plus bas.

## Valeurs figées des manches terminées (item 4)

Format (lu dans `components.rs` de libfilmshell, Den, MIT ; mon lecteur est réécrit, non recopié) : par composant de 28 à 55, un masque de 32 bits (un bit par
manche terminée), puis pour chaque manche deux valeurs A et B, chacune précédée d'un indicateur « valeur par défaut ». Ordre mesuré : (A manche 0, B manche 0, A manche 1, B manche 1...).
Le composant 28 + k répète l'emplacement 2k / 2k+1.

- Livrées dans un enregistrement dense de 34 à 58 composants, **environ 50 ms après le début de la manche suivante**, et reprises dans les instantanés suivants.
- Présentes dans les **9 films à plusieurs manches sur 9** (60ae07c4, 92f18088, 43716616, c88ec007, d9781168, c75f33b8, 443426df, 51ebbc0f, 24dbb67d),
  jamais dans les 48 films à une seule manche (zéro enregistrement, zéro valeur ; seul 47d20b5d, au pont d'identité illisible, en montre un, faux positif). Masques observés : N-1 bits pour N manches, jamais N.
- Contenu : tirs au but, précision, tirs tirés, dégâts, KDA, durée de vie moyenne, manches gagnées, XP de kit, tout le tableau, par manche. Contrôle d'ensemble : somme des manches figées <= API dans 36 à 38 cas sur 39 (les 1 à 3 écarts par emplacement ne sont pas expliqués ; probablement le pont d'identité).
- Valeurs figées contre valeurs vivantes de la même manche : ShotsFired figé > vivant 50/61, DamageDone 51/61, DamageTaken 37/61, jamais l'inverse sur les films propres :
  les valeurs vivantes traînantes sont bien périmées, les figées les complètent.
- Totaux de match sur les films à manches (compteurs exacts : éliminations, morts, assistances, tête, mêlée, grenade, arme lourde, score personnel) :
  figées des manches 0 à N-2 + vivante de la dernière manche = API **100 % (12/12 par compteur) sur les 4 films propres** (60ae07c4, c88ec007, 51ebbc0f, 24dbb67d),
  contre 21/39 pour la somme des manches vivantes seule sur les 9 films à manches (éliminations : 29/39 avec les figées, 21/39 sans ; sur les 4 films propres, la somme vivante seule donne 12/12 partout sauf l'arme lourde 11/12).
  Les écarts restants sont des films dont le pont d'identité ou l'attribution de manche est faux, pas des valeurs figées fausses.

**Il n'y a donc pas d'instantané final du tableau** : ni la dernière manche, ni le match. Les instantanés répètent les valeurs traînantes (instantané > flux de deltas : 2 cas sur 1 610, tous sur le KDA ; instantané plus ancien : 13 cas sur 115 pour le score personnel).

## Rythme des émissions, en résumé

| classe | emplacements | rythme |
|---|---|---|
| à chaque événement | éliminations, morts, tête, mêlée, grenade, arme lourde, série, score personnel, objectifs d'événement (drapeau, bases, VIP, extraction) | une émission par événement ; 1 à 9 % seulement des émissions dans des paquets de 8 composants ou plus |
| avec le jumeau | KDA (avec assistances), durée de vie moyenne (avec série), tirs tirés (avec tête), dégâts infligés (avec arme lourde), dégâts subis (avec réapparition) | une seule émission par événement du jumeau, jamais seuls (100 % de co-datation) |
| chronométré | TimeAsVip, LongestTimeAsVip | toutes les 100 ms |
| chronométré | StrongholdOccupationTime, StrongholdScoringTicks | toutes les secondes (1 001 ms) |
| à la manche | valeurs figées | au début de la manche suivante |
| jamais | tirs au but, précision, manches égalisées, objectifs terminés, ennemis de campagne | |

## Candidats à publier (écrits ET justes, non publiés aujourd'hui)

Publiés aujourd'hui : suicides, trahisons, série, tête, mêlée, grenade, arme lourde, réapparitions, éliminations, assistances, et les événements de drapeau,
bases (prises, sécurisations), VIP (sélections), explosions de bombe, plus les courbes de score et de score personnel.

À une manche, exacts >= 96 % sur 12 à 114 joueurs chacun (noms lus dans la définition du film pour les builds 1.12 et 1.13 ; supposés par la famille pour 1.8 à 1.11) :
1. CTF : FlagReturnersKilled (22B), KillsAsFlagReturner (25A), KillsAsFlagCarrier (24B, échantillon insuffisant : 1 joueur).
2. Bases et Contrôle total : StrongholdOffensiveKills (21B), StrongholdDefensiveKills (22A), StrongholdScoringTicks (23A).
3. VIP : VipKills (20B), VipAssists (21A), KillsAsVip (21B), BestKillingSpreeAsVip (22B), TimeAsVip (23B), LongestTimeAsVip (23A, à 0,1 s).
4. Extraction (noms supposés) : les cinq emplacements, 100 %, petits effectifs (1 à 16 joueurs).
5. Hors tableau de statistique mais sans coût : Deaths (2B) est exact à 99,8 % (déjà connu par le fil des morts).
6. Valeurs figées, pour les matchs à plusieurs manches : tirs tirés, tirs au but, précision, dégâts, durée de vie, KDA, manches gagnées, par manche terminée (exact en interne) ;
   et elles corrigent les totaux de compteurs des matchs à manches.

Écrits mais à ne pas publier comme totaux : ShotsFired, DamageDone, DamageTaken, KDA, AverageLifeDuration (traînants), FlagCarryTime et StrongholdOccupationTime (traînant et autre définition), LoadoutXP et Spartan* (doublons).

## Ce que je n'ai pas pu établir

- **Balle (Oddball)** : aucun film à une manche dans le parc. Sur les 7 films Oddball (2 à 3 manches), les emplacements du crâne ne passent pas la chaîne (37 à 65 %), par défauts
  du pont d'identité et de l'attribution de manche, pas par une valeur fausse prouvée. Non tranché.
- **Exactitude des valeurs figées de ShotsHit, Accuracy, ShotsFired, DamageDone par rapport à l'API manche par manche** : l'API n'a que des totaux. Prouvé seulement :
  cohérence interne (précision = touches / tirs), bornes (somme <= API, 36 à 39 films), et égalité exacte pour les compteurs de combat (12/12). Pour un match de 2 manches,
  les tirs au but de la manche 2 ne sont dans aucune source.
- Exactitude des valeurs traînantes à l'instant de l'émission, sauf KDA (formule recalculée, 99,9 %) ; pour les autres, seul le rapport au total de l'API (jamais au-dessus : 99,6 à 100 %).
- Valeurs de Score par joueur, Suicides, Betrayals, RoundsTied, AI* et Spartan* contre l'API : aucune colonne par joueur dans la copie.
- Spawns : pas de colonne API, seulement la relation à la colonne des morts.
- Extraction et builds <= 1.11 : définition illisible chez Den (12 films sur 58 : 1.8 à 1.11, plus 2 sans section d'identification). Noms posés par la famille de mode, soutenus par un accord à 100 %.
  Le lecteur de Den refuse ces builds (UnknownHeader) : mon témoin de l'instantané ne couvre que 1.12 et 1.13 (10 films).
- Pont d'identité : 559 joueurs appariés sur 777 lignes API ; les films à beaucoup de joueurs ou à manches multiples en perdent (c349da69 : 2 sur 13 ; c88ec007 : 2 sur 8).
- Modes absents du parc de la sélection (Stockpile, Sentry, Husky Raid autres que CTF) : non mesurés.

## Films employés (8 premiers caractères, 58)

0c0b1e2a e44bfaaa acda46c8 89377a9f fac07bec 684ca0cc 7cce111d 38900a5e 9669e5a2 aa415da6 3ff3d6c1 13d92593 7b60cff5 f37771d1 2228a19a c349da69 418eee76 8d827e90 76e6623c 572e236b
d55f2475 da2fd554 21ece4d8 3685373c 4ecdf3e7 47d20b5d 4f77afc1 11de8353 d5bbf57c b4035a67 9f9b19e5 50247b26 9ac5f43c 66aa5f0b 5676a9ba d2c64f8c 4ce480be 11899699 b6336eeb d4ddf054
60ae07c4 92f18088 43716616 c88ec007 d9781168 35b75a31 69b16f5d c75f33b8 99553e4a 00761d27 41722c72 be565134 e157a672 52fc79ef c5094b31 443426df 51ebbc0f 24dbb67d
(plus 000d5950 pour l'essai de la chaîne, sans API.) Builds : 1.8 (2), 1.9 (1), 1.10 (2), 1.11 (5), 1.12 (13), 1.13 (33), 2 films sans section d'identification. Modes : Slayer et variantes 19,
CTF 12, Bases 5, KOTH 3, Contrôle total 3, Balle 7, Assaut 3, VIP 2, Extraction 2, Husky Raid 2.
Témoin de Den (instantanés) : 3ff3d6c1 e44bfaaa 7b60cff5 8d827e90 4f77afc1 99553e4a 35b75a31 21ece4d8 d5bbf57c 9f9b19e5.
Valeurs figées lues dans les 58 films.

## Crédit

Den (OpenSpartan, libfilmshell, MIT) : la présence et le format des définitions d'emplacements (`variant_stats.rs`), la lecture des instantanés du tableau (`gamestate.rs`, `snapshot.rs`)
et le format du composant des valeurs figées (`components.rs`), lus comme témoins ; les définitions ont été relues avec son lecteur (sonde hors dépôt), le lecteur des composants figés dans le flux
de deltas a été écrit à part. Son constat « les emplacements Shots Hit et Accuracy du tableau ne contiennent rien dans le film » est confirmé pour le flux et les instantanés,
et complété : ils sont dans les valeurs figées des manches terminées.
