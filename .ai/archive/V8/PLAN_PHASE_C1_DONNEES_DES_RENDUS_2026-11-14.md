# Plan : phase C, premier temps — les données dont les rendus dépendent (2026-11-14)

Statut : **à valider par le user** (aucune étape lancée). Contrat d'exécution : skill `plan-execution`
(ordre strict, aucune case vide, découvertes notées et non traitées). Suite du §10 de
`PLAN_MARCHE_COMPLETE_ET_CANAUX_2026-11-10.md` (bloc 2). État des lieux sur pièces :
`annexes_phase_c1/inventaire_2026-11-14.md`.

## 1. Objectif et critère de succès

Le décodeur publie, mesurées contre l'API, les données que la phase B (rendus) consommera : présence
et propriétaire des vies, tableau des scores complet, tirs et touches, dégâts en points, médailles et
score par action, horloge et réapparition, zones et jauge du drapeau. **Succès** : chaque donnée
publiée l'est avec sa mesure contre une vérité écrite en annexe ; une donnée qui ne passe pas son seuil
n'est pas publiée (`[!]` avec la mesure), jamais publiée « à peu près ».

## 2. Où se fait le travail

- Code : dépôt du décodeur (clone `halo-infinite-film-decoder-wt-vehicules`), une branche par étape,
  PR, CI verte, fusion et release faites par la session (délégation du 13/11). Chaque release dit dans
  son message tout changement de ce que le décodeur rend.
- LevelUp : **une seule montée** à la fin (étape C9), sur la branche de la release qui suit la 8.2.0
  (décision du pilote du 14/11), avec la v1.3.8 et tout ce que ce plan publie. Rien sur
  `release/v8.2.0`.
- Vérités : lues sur une **copie** des bases (jamais celles que tient le serveur local), faite après le
  signal de fin de levelup-7c (passes en cours le 14/11). Aucune écriture en base dans ce plan.
- Équivalence : `replay-equiv` sur les 20 films depuis le worktree `LevelUp-wt-partage` par
  `GOWORK=<wt>/levelup-briques.work`, films lus dans l'archive du dépôt principal par variable, **aucune
  jonction** (incident du 13/11).

## 3. Décisions (tranchées par le user le 2026-11-14)

| # | Question | Décision |
|---|---|---|
| Q1 | Touches (C3) | Les touches sont publiées comme une **estimation**, avec, par match et par joueur, un indicateur de suffisance des données. Sous le seuil, l'écran affiche à la place un message concis et factuel disant que les données ne suffisent pas (le rendu relève de la phase B). Le seuil se calibre sur la mesure C3.2 : le niveau de l'indicateur au-dessus duquel au moins 7 joueurs sur 10 sont à 10 % près des touches de l'API. BTB : publié, marqué bêta (consigne du 10/11). |
| Q2 | Table des effets de dégât (C4) | Reprise de celle de libfilmshell, **bien créditée** : en-tête du fichier, `NOTICE.md` et README du décodeur, message de release. Confrontée aux fichiers du jeu installé par un test sous le tag `gamefiles`. |
| Q3 | Recuisson | Une seule, après la montée de LevelUp (C9), sur go du user et après accord des sessions paires ; aucune entre les étapes. |
| Q4 | Recherche de motif de xuid (C1) | Retirée si la lecture par les listes et `unit-control` nomme au moins aussi bien sur les 20 films et le parc ; sinon gardée en récupération comptée. |

## 4. Étapes

Ordre : C1 sert C2 à C4 (à qui est chaque ligne et chaque vie) ; C4 dépend de C2 (dégâts subis) et C3
(touches) ; C5 à C8 sont indépendantes et viennent ensuite ; C9 clôt.

### C0 — Préalables

- [x] C0.1 Copie de travail des bases de vérité (`shared_matches_v2`, bases joueurs pour
      `personal_score_awards`) dans `C:\t\phasec1\bases\`, après le signal de levelup-7c ; requêtes en
      lecture seule sur la copie. Fait le 14/11 : `shared_matches_v2` (15:00, aucun WAL ; 29 091 lignes
      de `match_participants` sur 2 043 matchs, `match_objective_stats`, `medals_earned`), neuf bases
      joueurs et `metadata` (les quatre tenues par le serveur copiées à froid par levelup-7c, arrêt
      propre). Requêtes : `C:\t\phasec1\q.exe <base> <sql>` (`cmd/diag_q`, ouverture `read_only`).
- [x] C0.2 Copie de libfilmshell à jour (vérifier laquelle est la plus récente ; rafraîchir si la
      fusion du 13/11 manque) ; sonde rebâtie si nécessaire. Fait : une seule copie, clone git
      (`Scripts\libfilmshell`), avancée de a13efb3 (10/11) à ad17521 (86 commits, fusions du 13/11
      comprises). Sonde neuve `C:\t\phasec1\sonde\` (API publique, chunks clairs, zlib ou zstd) :
      joueurs, arrivées, départs, vies avec propriétaire et positions de début et de fin.
- [x] C0.3 Témoins figés : documents des 20 films de l'équivalence tirés du parc régénéré le 14/11
      (schéma 116, décodeur v1.3.7, sortie identique à v1.3.8) copiés dans `C:\t\phasec1\temoins\`.
      Comparateur `C:\t\phasec1\cmp\` (vies appariées par temps et positions, propriétaire comparé).

Porte C0 : copie lisible, sonde qui tourne sur un film, témoins écrits.

### C1 — Présence et propriétaire des vies (D10)

- [x] C1.1 Brancher `DecodeRoster` (type 8) : premier et dernier listage de chaque joueur humain ;
      arrivées et départs datés dans les faits. Point d'étape de l'agent (non commité) :
      `grammar.ScanRosterListings`, `types.RosterListing`, faits persistés, `roster[].presence[]` daté par
      les listes. Mesure (`C:\t\phasec1\c1\mesure.md`) : 1 455 films, 49 625 paquets lus, 0 refusé ;
      contre `match_participants` (14 303 humains) : 950 arrivées et 1 511 départs confirmés, 0 arrivée
      vue par la liste seule ; erreur de temps 0,0 s (p50, p90) contre 10,4 / 18,6 s (arrivées) et
      4,3 / 13,7 s (départs) pour le document actuel (horloges alignées par un décalage médian par film :
      chiffre en partie auto-référent). unit-control sur 21 films : 3 230 vies au même humain, 0 à un
      autre. C1.4 à C1.6 renvoyés à l'agent (il les avait laissés de côté).
      Livré le 14/11 en un commit local (6012958, rebasé sur dd5e95d, schéma 119, `SchemaDesFaits` 24,
      `grammar.Rev` `grammar-2026-11-14.3`) : présence tirée des plages ; nommage corrigé par le joueur
      listé sous l'index à la création du corps (aucun humain listé = vie sans nom) ; corps sans création
      nommé par unit-control ; déclarations de bots lues par les plages ; places par la table du registre.
      `4f77afc1` : les 2 vies du bot perdent le faux nom humain ; le bot reste sans nom parce que l'hôte ne
      transmet pas les bots non épinglés à `Options.Bots` (à corriger à la montée de LevelUp, C9).
      `3abf0258` : rien ne bouge. `23ffd885` non traité. Équivalence : `artifact` et 4 étapes neuves seules.
      Revue adversariale (15/11) : deux P1 (nom juste retiré quand le corps précède le premier listage :
      `humainA` sans la règle de `tenuPar` ; siège repris par un listé hors table resté au nom de l'ancien
      occupant), quatre P2 (compteur gonflé, chemin sync sans unit-control, deux commentaires faux,
      champs persistés sans consommateur). Corrections renvoyées, avec mesure des vies renommées sur
      20 films et seconde passe d'équivalence sans `-update`.
      Corrigé (91075cc) et FUSIONNÉ dans main du décodeur le 15/11 (PR 23, 89c2559, schéma 119) : tests
      (hors `TestContractFixtures*`), lint, CI verts ; nommage sur 22 films : 0 vie gagne ou change de nom,
      2 perdent le faux nom d'un humain parti (`4f77afc1`) ; seconde passe d'équivalence : 20 identiques.
      Restent ouverts : `23ffd885` (C1.6) et le bot de `4f77afc1` (correctif côté hôte, C9).
- [x] C1.2 Observateur `unit-control` en production : le contrôleur du bipède créé donne le
      propriétaire de la vie ; fin de commande = fin de vie. Reprise d'un identifiant moteur par un
      arrivant traitée (porteur à l'instant, comme `holders.rs`). Fait : corps sans création nommé par
      `unit-control` (PR 23, 89c2559) ; fin de vie par lâcher de commande (PR 28 de d0, 52a72a8, schéma 123).
- [x] C1.3 Mesure : arrivées et départs contre `match_participants` (`joined_in_progress`,
      `left_in_progress`, `first_joined_time`, `last_leave_time`) ; propriétaire des vies contre le fil
      des morts et la lecture actuelle, sur les 20 films et le parc.
- [x] C1.4 Publication : `roster[].presence[]` tirée des listes ; `tracks[]` nommés par `unit-control`.
      Q4 appliquée. (PR 23, mesure `C:\t\phasec1\c1\mesure.md` §3-4.)
- [!] C1.5 (backlog, relevé le 10/11) Trajet de Warthog porté par l'humain alors qu'il suit un bot :
      film `4f77afc1`, siège 524, images 279 à 287 ; 8 pistes de bot rattachées à l'humain par le pont
      « premier occupant ». Le propriétaire par `unit-control` (C1.2) doit le trancher ; contrôle que les
      équipes ne bougent pas (joueur `3abf0258`, revenu sous deux index).
      [!] Partiel, justifié : les deux vies du bot de `4f77afc1` (emplacements 565 et 584) perdent le
      faux nom de l'humain parti (PR 23) et le bot les reprend par le correctif hôte de 58 (v1.3.9) ;
      `3abf0258` : aucune équipe ne bouge. Le trajet du siège 524 (images 279 à 287) n'a PAS été
      contrôlé à part : à vérifier au chantier suivant (sièges de véhicule, avec C8.1).
- [x] C1.6 (backlog, relevé le 10/11) Pose des places : 13 films à siège vacant intercalé lus sans
      table (`sieges.go`, `siegesDuDebut` lit encore `grammar.Lue()`) ; film `23ffd885` : siège 2 à
      l'index 8 par les chunks. Lecture par la table du chunk 0 et les listes de C1.1.
      Fait : `grammar.ScanFilmPlayerTableAuxListages` (31 films dont `23ffd885`), publié en v1.4.0 (PR 31).

Porte C1 : mesure C1.3 écrite en annexe ; accord avec l'API au moins égal à la lecture actuelle ;
`replay-equiv` : seuls présence et nommage des pistes changent.

### C2 — Tableau des scores complet

- [x] C2.1 Lever le plafond de 8 joueurs du statborg (`StatborgSlotEquipeMax`, `StatPlayerSlots`),
      index de joueur pris de C1. Écrit sur `feat/c2-statborg` (00e5b3e, e0480e7 ; schéma 117 local) :
      bande bornée par le dernier siège de la table de datums (composant 57 ; égale sur 1 608 films sur
      1 612), 158 films au-delà du slot 24 ; `scoreTimeline.players[].stats` (10 compteurs nommés).
      Accord parc entier : frags 99,7 %, morts 99,6 %, assistances 97,3 %, score 98,9 % ; `shotsFired`
      5,2 % et `damageTaken` 39,7 % (réplication rare). Équivalence : `artifact` sur 20 films, `score`
      et `flag` sur 5 films de plus de 8 sièges. En revue ; mesures demandées : accord au-delà de 8 sièges
      séparément (porte), films à plusieurs manches (56-78 % non établi), dernier point de `shotsFired` /
      `damageTaken`. Résultats (`C:\t\phasec1\c2\`) : au-delà de 8 sièges (158 films, 79 % des lignes API
      appariées) frags 99,2 %, morts 99,5 %, assistances 96,5 %, score 97,4 %, autres 97,4-98,3 % (lobbies
      de 8 : 99,2-99,7 %) ; plusieurs manches : 80-88 % (67-73 % par les morts seules) contre 99,6 % pour
      libfilmshell sur les mêmes films, écart dû à l'association (slot, manche) -> xuid ; `shotsFired` et
      `damageTaken` jamais finals, même au dernier point (15,7 % et 50,7 % dans la dernière minute).
      Décisions du pilote : (1) `shotsFired` et `damageTaken` retirés de la publication (valeurs jamais
      finales ; l'API porte les totaux) ; (2) compteurs nommés NON publiés sur les films à plusieurs
      manches tant que l'association par manche n'atteint pas le niveau d'une manche (suite : association
      par unit-control de C1) ; (3) au-delà de 8 sièges, écart résiduel à expliquer en annexe (porte).
      Revue adversariale : un P1 (le calque du drapeau change sur les CTF de plus de 8 sièges sans oracle :
      `084a804d` 23 -> 41 portages et jauge de retour qui ne s'apparie plus) -> le drapeau garde l'ancienne
      bande dans ce lot, au-delà de 8 sièges il passe à C7 ; cinq P2 (tests qui ne verrouillent pas la
      borne ni le retour à zéro, repli silencieux sans table de datums, émissions jetées non comptées,
      doc de `Total`). Corrections renvoyées à l'agent.
      User le 14/11 : « Pour les tirs et dégâts subis faut recroiser peut-être avec ce qu'a fait Den, pour
      les manches et le drapeau aussi si tu as du mal. Je ne supporte pas officiellement le BTB donc pas
      grave si là-dessus ça ne marche pas super. » Suite : tirs et dégâts subis viendront des événements de
      tir et de touche (C3, C4 ; libfilmshell les lit un par un, son tableau des scores a la même
      réplication rare que le nôtre) ; manches : reprendre la méthode d'association par manche de
      libfilmshell (99,6 %) comme témoin ; drapeau : `Film::carries` comme témoin en C7 ; au-delà de 8
      sièges (BTB), l'écart résiduel est accepté.
      Corrections faites (f243408, 7f84210 ; rebasée sur 57ea9fe, schéma 118) : `shotsFired` et
      `damageTaken` retirés ; `stats` absent sur les films à plusieurs manches, compté
      (`coverage.score.statsWithheld`) ; drapeau sur l'ancienne bande (`EntreeDuDrapeau`) ; repli
      `repli_bande_statborg_par_defaut` nommé et compté ; émissions jetées comptées ; tests mutants
      vérifiés. Écart au-delà de 8 sièges : 446 des 624 humains non appariés sont des arrivées ou départs en
      cours ; 21 désaccords de frags sur 2 783 appariés. Équivalence : `artifact` partout, `score` sur les
      5 films de plus de 8 sièges, et les empreintes de `flag` et `skull` sur les 20 parce que
      `ComptesDesReplis` (embarqué par le pont d'identité) gagne un champ ; sorties identiques (vérifié sur
      `51101d1d`). Manches : libfilmshell lit l'index de joueur de chaque entité du tableau (composant 57)
      et le registre des inscriptions (types 8 et 12) ; deux faits neufs, dont le type 8 que C1 apporte ->
      ajouté en C2.5.
- [x] C2.5 (ajouté le 14/11) Films à plusieurs manches : lire la valeur du composant 57 (index de joueur
      de chaque entité du tableau) et associer par les inscriptions de C1.1 ; publier `stats` sur ces films
      si l'accord atteint celui d'une manche (libfilmshell : 99,6 %). Dépend de la fusion de C1.
      Lancé le 15/11 après la fusion de C1 (branche `feat/c2-manches` depuis 89c2559).
      Premier rendu (rebasé sur 3ec1fb2, schéma 122 local) : entrées du tableau lues aux images-clés
      (`grammar.ScanStatborgEntries`, composant 57) résolues par les plages de type 8
      (`objectives.EntryResolver`) ; plusieurs manches : frags 85 -> 98 %, morts 80 -> 95 %, assistances
      88 -> 97 % (99,5 / 98,5 / 99,0 % là où le nombre de manches égale celui de libfilmshell) ; une
      manche : frags 99,7 -> 99,9 %, assistances 97,4 -> 98,7 % ; au-delà de 8 sièges : humains nommés
      82 -> 96 %, frags 99,2 -> 99,9 % mais morts 99,5 -> 98,7 % et score 97,4 -> 97,0 %. 9 films BTB
      comptent des manches en trop. Renvoyé : origine du recul et décision des manches par lecture.
      Point d'étape (rebasé sur 558af5f, schéma 122, faits 27, `grammar-2026-11-15.2`) : le recul BTB venait
      des joueurs nommés pour la première fois dont la dernière mort se perdait au vidage de la ligne au
      départ (41 sur 42) -> le zéro qui suit une valeur positive est écarté aussi sur les quatre compteurs
      de base, compté. Manches fantômes : l'horloge ne les distingue pas (18 vrais films à plusieurs manches
      sur 51 ont une horloge muette) ; les issues de manche (composant 56) oui -> une manche compte si son
      numéro ne dépasse pas la dernière décidée plus celle en cours (`coverage.score.roundsUnplayed`).
      Accord frags / morts / assistances / score : plusieurs manches 99,6 / 99,6 / 99,2 / 99,2 ; BTB 100 /
      100 / 99,9 / 98,4 ; une manche 100 / 100 / 99,9 / 99,8. Équivalence en cours ; revue à suivre.
      Commit 0fa753b ; revue adversariale : 0 P0/P1, cinq P2 (canal négatif non rejeté sur les compteurs
      de base : b1b62f38 publie 16 morts pour 7 ; `Unitary` sans condition et commentaire faux ; deux
      commentaires « faits 25 » ; `joinSegments` à plusieurs segments sans test ; l'étape `score` de
      `replay-equiv` ne couvre pas `scoreTimeline.players[]` ni `stats` : trou du harnais côté LevelUp,
      transmis à levelup-58). Corrections renvoyées, rebase sur 5a71271.
      Corrigé (4e1d957 : canal négatif écarté sur frags et morts seulement — les assistances ont un canal B
      légitimement négatif ; mesure refaite en égalité exacte : BTB morts 100 %, plusieurs manches 99,3 à
      99,7 %) ; goldens et graines refigés par le pilote avec les données de l'hôte (3efe72e : l'agent les
      avait passés sans la racine, les tests sautaient). FUSIONNÉ le 15/11 (PR 27, a36bf67, schéma 122).
      Signal donné à 58 et à d0.
- [x] C2.2 Nommer les composants que libfilmshell nomme (14) et que le film remplit, dont le composant
      10 A (dégâts subis) ; chacun confronté à son équivalent API quand il en a un. Fait : 10 compteurs
      nommés, confrontés (PR 22) ; `shotsFired` et `damageTaken` (10 A) retirés, jamais finals ; les
      dégâts viennent de C4.
- [x] C2.3 Mesure : frags, morts, assistances, score contre l'API au-delà de 8 joueurs (aujourd'hui 87 à
      94 %). Mesuré (`C:\t\phasec1\c2\`, puis C2.5) : au-delà de 8 sièges 100 / 100 / 99,9 / 98,4 %.
- [x] C2.4 Publication : séries par joueur des composants retenus dans `scoreTimeline.players[]`.
      (PR 22, puis PR 27 pour les films à plusieurs manches, PR 29 pour la disposition prouvée.)

Porte C2 : au-delà de 8 joueurs, accord au moins égal à celui des lobbies de 8 (99,7 %) ou écart
expliqué en annexe ; `replay-equiv` : seuls les champs de score changent.

### C3 — Tirs et touches, par arme et dans le temps

- [x] C3.1 Touche = effet de dégât d'arme par (tireur, victime, image), projectiles compris ; arme par
      le tag de dégât, sinon par le dernier tir.
- [x] C3.2 Mesure, nos deux méthodes ensemble (appariement actuel et touche par effet de dégât) contre
      `match_participants.shots_hit` et `accuracy` ; tirs contre `shots_fired`.
- [x] C3.3 Publication (Q1) : `hits[]` datés (tireur, victime, arme) au document, marqués
      estimation, avec l'indicateur de suffisance par joueur et le seuil calibré sur C3.2.

Porte C3 : seuil calibré et écrit en annexe avec sa mesure ; `replay-equiv` : seuls les champs ajoutés
changent.

Lancé le 15/11 avec C4 (worktree `hifd-c3`, branche `feat/c3-touches-degats`, depuis 558af5f) ; mesure
d'abord. Tirs (`shotsFired`) par les événements de tir, plus par le tableau des scores (C2).
Mesure (193 films, 2 559 joueurs ; `C:\t\phasec1\c3\`) : touches à 10 % de l'API : appariement actuel 11 %,
touche par effet de dégât 60 % (HI_1_13_0), 57 % (12), 38 % (11), 8 % (10) — égale à libfilmshell (59
contre 61 %) ; tirs : compteur `FireNumber` 58 %. Indicateur de suffisance par joueur : couverture des tirs
>= 0,7 et doublons < 1,2 -> 77 % à 10 % près (81 % en builds 12 et 13) pour 23 % des joueurs. C4 avec la
table de libfilmshell sans vie réelle : subis 0,1-1,4 % d'écart médian, infligés +15-19 % (porte non
tenue). Décision du pilote : C3 publié seul d'abord, puis C4 sur sa branche avec la vie du bouclier et du
corps (composants 4 et 5 du bipède).
C3 livré le 15/11 sur `feat/c3-touches-degats` (2 commits depuis 558af5f, schéma 122 local, à renuméroter
après le tag v1.3.9) : `hits[]` {t, shooter, victim, effect, w} marqués estimation, `shotStats[]` par joueur
(tirs, tirs écrits, touches, couverture, doublons, `sufficient`), seuils dans `coverage.hits` ; table des
effets de dégât de libfilmshell reprise et créditée. `sufficient` = couverture >= 0,7 et doublons < 1,2 et
trames arrêtées <= 0,5 % (troisième critère ajouté : builds 9 et 10 passaient à tort). Mesure (193 films,
2 560 joueurs) : suffisants 591 (23 %), touches à 10 % 80 %, tirs 86 % ; non suffisants 27 % et 54 %
(`C:\t\phasec1\c3\seuil.md`). Porte tenue. Appariement actuel gardé dans `decfilm` (appelé par le
killcollector et l'ingestion de la précision de LevelUp) : retrait avec la migration côté LevelUp. Revue
et fusion après le tag. C4 empilé sur C3 (`feat/c4-degats`).
Mesure C4 (15/11, vie réelle du bouclier et du corps, composants 4 et 5, comme libfilmshell) : médiane
des rapports infligés / subis : HI_1_13_0 1,002 / 1,022, HI_1_12_0 0,978 / 1,013, HI_1_11_0 0,749 / 0,959,
plus ancien 0,36-0,40 / 0,53-0,56 ; écart absolu médian par joueur 7,5-11,8 % (infligés) et 5,8-6,6 %
(subis) en builds 12 et 13 ; libfilmshell 1,02 / 1,06. Décision du pilote : la porte se lit en médiane
des rapports (référence du plan) ; publication en estimation sous `sufficient` de C3, avec la part des
suffisants à 10 % (cible 7 sur 10).
C4 livré le 15/11 sur `feat/c4-degats` (864e4ca, 5c21704, empilée sur C3 ; schéma 123 local, à renuméroter) :
`damage[]` par coup (moment, auteur, victime, effet, arme, points de bouclier et de corps, `measured`),
`shotStats[].damageDealt` / `damageTaken` sous `sufficient` ; table des montants de libfilmshell créditée.
Mesure (`C:\t\phasec1\c3\seuil_degats.md`) : suffisants (591) médiane des rapports 1,011 / 1,020, à 10 % :
76 % / 75 % (porte tenue), à 5 % : 59 % / 56 % ; builds 12 et 13 73-82 % / 75-76 %. Équivalence à faire
après le tag v1.3.9 ; revue et fusion avec C3 dans la version suivante.
Revue adversariale C3+C4 (15/11) : P1 le calcul des points est une traduction ligne à ligne de
`hit_damage.rs` (règle : le code de libfilmshell est un témoin) et NOTICE dit à tort « own code » ->
réécrit dans nos termes, méthode créditée ; P2 regroupement sans `AVictime`, touche sur soi comptée,
vitalités sans rattrapage des trames non localisées (compteur mort), totaux hors axe non documentés.
Renvoyé. Corrigé (43e7374, 2fb749d, 1e2253d, encore sur la base 558af5f) : calcul réécrit dans nos termes
(`damageeffects.Repartir`, scènes, registre des vies), méthode de `hit_damage.rs` créditée, NOTICE
corrigée ; 8 fixtures de dégâts re-rendues à l'identique ; clé de scène sans les messages sans victime ;
touches et dégâts sur soi exclus et comptés (`selfHits`, `selfDamages`) ; vitalités avec le rattrapage
des listes non localisées, trames illisibles dans `sufficient` (`vitalsUnread`, 0 sur les builds 11 à 13) ;
totaux et axe documentés. Mesure de contrôle (193 films), rebase, équivalence : après le message de fin
de 58.

### C4 — Dégâts infligés et subis en points

- [x] C4.1 Table des effets de dégât (Q2) ; dégâts par coup en points (bouclier, corps).
- [~] C4.2 Total réel par vie : composant 10 A à la réapparition ; parts du tueur et de l'assistant au
      message de mort ; l'estimation par coup répartit, le total réel recale (piste du user du 10/11).
      Couvert par C4.1 autrement : le composant 10 A n'est jamais final (C2 : 50,7 % dans la dernière
      minute) ; le recalage se fait par la vie réelle du bouclier et du corps (composants 4 et 5),
      comme libfilmshell ; porte tenue (C4.3).
- [x] C4.3 Mesure contre `damage_dealt` et `damage_taken` (référence du 10/11 : médianes 1,05 et 0,96).
      Contrôle sur la v1.4.0 (16/11, `C:\t\phasec1\c3\ana_dmg_d.txt`) : 1,012 / 1,020, à 10 % 76 / 75 %.
- [x] C4.4 Publication : dégâts infligés et subis par joueur dans le temps, et par arme. (`damage[]` par
      coup daté avec son arme, totaux par joueur dans `shotStats[]` sous `sufficient` ; v1.4.0.)

Porte C4 : mesure écrite ; écart médian à l'API sous 5 % ou `[!]` ; `replay-equiv` : seuls les champs
ajoutés changent.

### C5 — Médailles et score par action

- [x] C5.1 Publier les médailles horodatées au document (faits déjà lus), avec leur nom. FUSIONNÉ dans
      main du décodeur le 14/11 (PR 21, 57ea9fe, schéma 117) après corrections de la revue (identifiants
      fictifs, garde-rails étendus, `frameOfOriginMS` unique), tests (hors `TestContractFixtures*`), lint
      et CI verts ; worktree `hifd-c5` retiré, caches purgés. Écrit d'abord sur
      `feat/c5-medailles` (e3d4874, ca41743 ; schéma 117 local) : `medals[]` {t, ms, xuid, id, name},
      `coverage.medals`, couche `medals` ; une seule lecture du morceau des temps forts avec le fil des
      morts. Équivalence : seule l'étape `artifact` change, sur les 20 films. Tests verts hors les quatre
      `TestContractFixtures*` (fixtures web de LevelUp, à refaire à la montée). Revue adversariale : un
      P0 (vrais xuids dans les fixtures de médailles et les goldens ; branche supprimée du serveur public,
      commits encore joignables par SHA ; user le 14/11 : pas de demande au support GitHub), trois P2 (troisième copie du calcul de frame, deux
      commentaires faux). Correction demandée à l'agent, garde-rail d'identité étendu, historique
      réécrit ; agents sans droit de push désormais.
- [x] C5.2 (rouvert le 14/11 sur remarque du user : « c'était vraiment un événement en lecture pure ».
      La mesure ci-dessous appliquait un barème de l'extérieur ; le film porte les réponses de jeu
      `*PersonalScore*` par destinataire dans `PlayerGameEventSmall` (genre 82), que libfilmshell nomme
      avec leurs points (`events_data.rs`). Relancé sur `feat/c5-score-par-action`, mesure d'abord.)
      Mesure bis (`C:\t\phasec1\c5\mesure_c52b.md`) : le message 82 porte le rang de la réponse dans
      l'arbre d'événements du script de mode (`R(32)`), à lire avec le groupe de scripts du film (noms
      `*.lua` du chunk 0) ; aucun point sur le fil, ils viennent de la table de libfilmshell
      (`events_data.rs`, 739 réponses dont 65 à points). libfilmshell : somme = `personal_score` pour
      97 joueurs sur 104 (Assault seul en défaut) ; 65 clés de récompense sur 67 égales. Notre grammaire :
      identique au message près sur `000d5950` (3 073 messages, sommes = API), mais 1 209 événements à
      points contre 2 085 sur 11 films, parce que la marche de la vue A s'arrête au premier message dont
      la charge n'est pas lue (genres 127, 60, 50, 52, 66, 90, 111 à 117). Suite lancée le 14/11 :
      inventaire des genres qui arrêtent la marche, puis lecteurs de charge un par un (témoin
      libfilmshell). Reprise de la table des réponses : décision du user attendue.
      Rectification (`mesure_c52d.md`) : les arrêts venaient de l'outil de mesure (carte et découpage non
      posés, positions du tir lues 5 bits trop court), pas de la grammaire. Corrigé : messages 82
      identiques à libfilmshell sur ses 11 films, 2 085 événements à points contre 2 085 ; modes arène :
      somme = `personal_score` pour 92 joueurs sur 92 (10 films) ; Assault 5/12 (bombe sans réponse à
      points dans la table). Restent : 4 films à vue A illisible, groupe de scripts de `084a804d`, builds
      38-39 du BTB (tirs sans bloc de visée, 20 à 30 % des messages perdus, sans témoin). Suite : lecture
      des messages 82 et faits dédoublonnés commités ; publication après la décision sur la table.
      PR 22 (tableau des scores) fusionnée dans main du décodeur (dd5e95d, schéma 118).
      User le 14/11 : « oui tu peux reprendre la table des points de Den, je te l'ai déjà dit. Tu peux le
      créditer aussi » ; Assault est un mode personnalisé fait par des fans, sans barème : pas un défaut.
      Publication lancée (table créditée : en-tête, NOTICE.md, README ; couverture des modes sans points).
      Livré le 15/11 sur `feat/c5-score-par-action` (5 commits, schéma 119 local) : `scoreActions[]`
      {t, xuid, name, points, target}, `coverage.scoreActions`, table `reponsesdejeu/data/reponses.tsv`
      (423 lignes, 11 groupes) créditée avec la licence MIT ; garde `.luac` corrigé (groupe Slayer de
      `000d5950` inconnu avant). Mesure sur 193 films : builds HI_1_12_0 et HI_1_13_0, 963 joueurs sur 964
      exacts et 550 clés de récompense sur 550 ; HI_1_9_0 à HI_1_11_0 (BTB) 228 sur 587 (messages perdus
      après un tir sans bloc de visée) ; 47 films à groupe absent de la table ; 15 à vue A illisible.
      `betrayalPersonalScore` : `target` = le trahi (validé sur 4 films) -> les trahisons de C6.2 se
      lisent ici. Revue adversariale : table identique à libfilmshell (423/423), licence MIT conforme,
      dédoublonnage et codec justes ; P1 la couverture ne distingue pas « mode sans barème » de « rang
      inconnu » et aucun verdict ne dit qu'un film est complet (`stoppedFrames` seul, et `e5adf7b2` faux
      sans arrêt) -> compteurs séparés et verdict `complete` par film (sinon message « données
      insuffisantes » côté LevelUp, règle Q1) ; P2 messages d'un bot attribués à l'humain parti du même
      index -> attribution seulement à un joueur présent à l'instant. Renvoyé.
      Corrigé (cinq classes de messages, arbre complet des réponses de chaque groupe : 4 804 rangs dont 423
      à score ; `complete` ; attribution à la présence, `notPresent`) puis FUSIONNÉ le 15/11 (PR 25, 558af5f,
      schéma 121, faits 26). Sur 193 films, 80 complets et 765 joueurs sur 831 exacts parmi eux : `complete`
      est nécessaire, non suffisant (Assault hors de l'arbre ; points de véhicule de HI_1_11_0 différents de
      la table) ; documenté par le pilote (FORMAT.md, commentaire du champ) : le consommateur qui tient le
      score officiel compare la somme (montée de LevelUp, C9). Worktree `hifd-c5` désinscrit, dossier
      verrouillé à retirer plus tard.
      Score par action : barème par médaille et par action (`LabelPersonalScore`) ; mesure de la
      somme contre `personal_score` (tous les joueurs) et contre `personal_score_awards_latest` (joueur
      synchronisé). Mesuré : les médailles ne portent aucun point (`personal_score_awards_latest` n'a
      aucune ligne de médaille ; la somme des récompenses égale `personal_score` sur 22 couples sur 23).
      Reste le barème par action (frag, assistance, objectif) : non fait. Mesuré ensuite
      (`C:\t\phasec1\c5\mesure_c52.md`) : barème constant (frag 100, assistance 50, capture 300, saisie de
      zone 25 ; `zone_captured` 50 / 75 / 100 selon le mode) ; le film ne date ni marques, ni capteurs, ni
      EMP, ni véhicules, ni points de colline ; somme du barème sur les actions datées = `personal_score`
      pour 24 % des joueurs (53 % en modes sans objectif). Décision du pilote : pas de score par action
      publié dans ce lot (il serait faux trois fois sur quatre ; l'API porte le score). À reprendre si C3 à
      C7 datent les actions manquantes.
- [x] C5.3 Mesure : comptes de médailles contre `medals_earned`. Mesuré
      (`C:\t\phasec1\c5\mesure.md`) : 195 clés (film, joueur, médaille) sur 18 films, 188 égales, aucun
      compte différent, 7 absentes du film (un joueur de `7344d24f` que le morceau n'écrit pas) ; contre
      libfilmshell, 351 clés toutes égales. À cocher après la revue.

Porte C5 : mesures écrites ; `replay-equiv` : seuls les champs ajoutés changent.

### C6 — Horloge de manche, compte à rebours de réapparition, tirs alliés

- C6.1 Publier l'horloge de manche et le compte à rebours de réapparition (déjà décodés, sans
      appelant). (statut en fin de section)
- C6.2 Tirs alliés : drapeau `friendlyFire` et pénalité de trahison de la variante ; morts entre
      coéquipiers marquées au document. (statut en fin de section)
- C6.3 Mesure : compte à rebours contre le délai de la variante ; trahisons contre l'API (joueur
      synchronisé) et contre libfilmshell. (statut en fin de section)

Porte C6 : mesures écrites ; `replay-equiv` : seuls les champs ajoutés changent.

État au 14/11 : `feat/c6-horloges` (c2ea4ab, f3ed42b ; rebasée sur main 57ea9fe, schéma 118 local) :
`roundClock`, `respawnCountdowns`, `betrayals` (lu dans `player-last-betrayer` de la victime ; le
marquage par équipe au fil des morts abandonné : 38 trahisons, aucune en commun avec libfilmshell),
`rules`. Mesure (`C:\t\phasec1\c6\mesure.md`) : limite = durée de la variante à 0,55 s près (21 films),
pente 0,999-1,003 sur 19 films sur 23 (4 films à arrêts et relances, non expliqués) ; comptes à rebours =
délai de la variante dans 3 729 cas sur 3 931 ; variante égale à libfilmshell sur 1 345 films ;
trahisons : 72 (nous) contre 60 (libfilmshell), 17 couples chez nous seuls ; API 4 matchs sur 6.
Équivalence : étapes `horloges` (neuve) et `artifact` seules. Revue adversariale : P0 l'index lu (création
ou i15 - 1) n'est pas l'occupant courant quand un arrivant reprend un siège (trahisons et comptes à
rebours nommés sur le mauvais joueur ou un absent : `4f77afc1`, `a5eb7277`, `11de8353`) ; P1 comptes à
rebours en double quand une lecture manque (`084a804d` : 118 groupes sur 580) ; P1 tueur = victime sur
slot fantôme (`43390876`) ; faux positifs (`dd3095de`). Décisions du pilote : trahisons retirées du lot,
refaites par la lecture de la réponse `betrayalPersonalScore` des messages 82 (C5.2) ; comptes à rebours
publiés seulement pour un index à occupant unique, le reste retenu et compté, résolution à l'instant
après C1 ; doublons dédoublonnés ; `frameOf` existant réutilisé.
Corrigé le 15/11 (556009f, e4d26e9, schéma 119) : trahisons et lecture i15 retirées ; comptes retenus et
comptés (index inconnu, réutilisé, hors présence) ; doublons fusionnés ; horloge des 4 films à pente
atypique identique à libfilmshell sur les 3 qu'il lit (publiée). Relecture du pilote : une presence vide
(« présente à aucun instant ») publiait encore le compte, cas même du P0 de `4f77afc1` -> renvoyé.
Collision de numéros avec C1 (119, faits 24, `grammar-2026-11-14.3`) : le second fusionné renumérote.
FUSIONNÉ le 15/11 (PR 24, 3ec1fb2, schéma 120, faits 25) après correction de la presence vide (9 comptes
retirés sur `11de8353`) et rebase sur C1 (presence par les listes : 3 156 comptes publiés, 255 hors
présence) ; tests, lint, CI verts ; seconde passe d'équivalence : 20 identiques. Worktree `hifd-c6`
retiré.
- [x] C6.1 horloge de manche et comptes à rebours publiés.
- [~] C6.2 tirs alliés : `rules.friendlyFire` et `betrayalPenaltyS` publiés (C6.1) ; morts entre
      coéquipiers marquées par `scoreActions` (`betrayalPersonalScore`, `target` = le trahi), C5.2.
- [x] C6.3 mesures écrites (`C:\t\phasec1\c6\mesure.md`).

### C7 — Propriétés gérées : zones et jauge du drapeau (D11)

- [x] C7.1 Les deux lectures (forme éparse seule, aujourd'hui ; les deux formes, par la marche) sur le
      parc des modes à zones et à drapeau.
- [x] C7.2 Vérité : `match_objective_stats` (captures, sécurisations, tics de score, temps en zone,
      retours de drapeau) ; temps de retour du drapeau lu dans la variante du film (hachage
      `0x2b16b76f` : présence à vérifier).
- [x] C7.3 Bascule de la lecture retenue derrière la marche (12 à 20 % d'une cuisson sur ces modes),
      ou `[!]` si aucune des deux ne tient. Retenu : zones inchangées (la passe tient) ; jauge du drapeau
      choisie par lâcher, la marche quand elle a un point, sinon la passe (v1.4.0). Drapeau au-delà de
      8 sièges NON ouvert (mesure du 16/11 en Découvertes).

Porte C7 : mesure écrite, lecture retenue justifiée ; `replay-equiv` : seuls zones et jauge changent,
dans le sens de la mesure.

Lancé le 15/11 (worktree `hifd-c7`, branche `feat/c7-zones-drapeau`, depuis 558af5f), drapeau au-delà de
8 sièges compris (laissé par C2). User le 15/11 : « Pour les zones j'avais déjà pas mal travaillé ce sujet,
ce doit être de l'affinage mais c'est pas si nouveau » -> zones = affinage de la lecture publiée (colline
KOTH validée, bases pré-capturées de Strongholds), corrections seulement sur écart mesuré, Q4 ; le
nouveau de C7 est la jauge du drapeau, son temps de retour et le drapeau au-delà de 8 sièges.
Point du 15/11 (commit local 326abe4, `C:\t\phasec1\c7\mesure.md`) : zones inchangées (Bastions : 335
captures du tableau toutes à 3 frames d'un intervalle publié, la marche en perdrait 2 ; API 517/517
captures, 123/124 sécurisations ; KOTH identique) ; jauge de retour lue par la marche (132 lâchers sur 153
avec jauge contre 127) ; temps de retour lu dans la variante (`0x2b16b76f` : 15, 24, 40 ou 60 s sur les 46
films CTF ; le manifeste disait 30 s) -> `rules.flagReturnS` ; drapeau au-delà de 8 sièges : erreur de
portage 1 161 -> 282 s sur 1 384 s (9 films BTB), aucun film ne régresse ; 24 films BTB sans calque de
drapeau (expliqué, non traité). Reste équivalence et références : après le tag v1.3.9 (version suivante). C8 et C1.6 lancés le même jour (worktree `hifd-c1`, branche
`feat/c8-backlog`), mesure d'abord, aucune régénération sans le user.
Point C8 du 15/11 (commit local ceb43d3, `C:\t\phasec1\c1\mesure.md` §6) : C8.1 pas de défaut de bande
(un seul film à « autre archétype », `a12f267b` ; 3 parents hors bande, pas des véhicules) ; vraie
anomalie : 44 montées sur 9 533 à plus de 15 m du porteur, 43 par pièce montée dont le porteur est choisi
par voisinage de slot (`58d09c44`) -> à planifier. C8.2 : cinq films rafraîchis le 14/11 ; restent
`edc5daf6` (faits 15, pas de document) et `e96070e2` (septième, faits 19, document 107, pas de chunk sur le
poste). C8.3 : `5da6fd30` déjà nommé au document 116 (0 vie sans nom). C1.6 : `23ffd885` est un cas parmi
31 films (rang de table différent du siège listé) -> `SiegesParLesListages` : 44 entrées changent de siège,
38 pistes gagnent leur équipe, une vie gagne son nom ; version suivante. Incident : l'agent a lancé
`taskkill /IM go.exe` (peut avoir tué des go.exe d'autres sessions) ; sessions prévenues, règle ajoutée
aux consignes (arrêt par PID seulement).
Revue adversariale C7 + C8 (15/11) : C7 : P1 jauge par la marche qui supprime des jauges justes
(`eba1e63f` 25 lâchers, `7fce3219` 7) -> lecture par lâcher, la marche seulement où elle lit ; P1 porteur
faux par joueur au-delà de 8 sièges (`378659b1` 5,3 -> 41,6 s) -> cause à trouver, sinon sièges non
ouverts pour le drapeau ; P1 aucun golden de la jauge par la marche ; P2 `resetSeconds` à 30 s inventé,
chronique inexacte. C8 : P1 sortie changée sans montée de SchemaVersion ; P2 appel et garde sans test,
deux autres lecteurs de la table gardent le rang non corrigé, collision de révision. Renvoyé aux agents.
C8 corrigé (47bb86b) : SchemaVersion montée (numéro au rebase), tests sur la mini-bobine `000d5950`
(permutation de deux lignes de la table) qui échouent sans l'appel et sans le garde, helper unique
`grammar.ScanFilmPlayerTableAuxListages` pour la cuisson, `decfilm` et le kill-feed, garde-rail
`TestAucunLecteurNeLitLaTableSansLesListages`. En attente du tag, puis rebase et équivalence.
C7 corrigé (37d934f) : jauge choisie par lâcher (marche quand elle a un point, sinon la passe) : 493 séries
publiées contre 458 (passe seule) et 456 (marche seule), aucune perdue ; sièges au-delà de 8 NON ouverts
pour le drapeau (trois porteurs faux de plus de 5 s par joueur, dont `390b1de5` déjà faux sur huit sièges ;
`coverage.flagCarries.seatsBeyondBand`), à reprendre avec le composant 57 après rebase ; `resetSeconds`
absent sans délai lu dans la variante (champ optionnel côté LevelUp) ; chronique corrigée.

### C8 — Entrées du backlog film (transmises le 14/11 à la demande du user)

- [!] C8.1 Bande `ti=40` : 249 parents d'occupation exclus (slot ayant porté un autre archétype), 6
      porteurs de pièce lus à plus de 15 m à la montée. Mesure puis correction générale, ou `[!]`.
      Mesuré : pas de défaut de bande ; vraie anomalie, 44 montées sur 9 533 à plus de 15 m du porteur
      (porteur choisi par voisinage de slot, `58d09c44`). Correction générale non faite : chantier suivant.
- [~] C8.2 Sept films à faits d'un ancien schéma (`000d5950`, `51101d1d`, `9f57c612`, `a521164d`,
      `bcb6d393`, `edc5daf6`, un non identifié) : vérifier d'abord si la régénération du parc du 14/11
      (schéma 116) les a déjà rafraîchis ; ceux qui restent vont dans la régénération ciblée de C9.3.
      Vérifié : cinq rafraîchis le 14/11 ; `edc5daf6` (faits 15) et `e96070e2` (pas de chunk sur le
      poste) restent pour la régénération, au go du user (C9.3).
- [x] C8.3 Film `5da6fd30` : six vies sans nom tant que ses faits ne sont pas régénérés ; même
      vérification, puis régénération ciblée de C9.3 si nécessaire. Vérifié : déjà nommé au document 116,
      0 vie sans nom.

Porte C8 : chaque entrée statuée avec sa mesure ; les entrées sont retirées du backlog de la release à la
montée (C9.2), jamais par un push sur `release/v8.2.0`.

### C9 — Clôture

Changement du 15/11 (levelup-dd et levelup-58, choix du user rapporté) : la montée du décodeur entre dans
la 8.2.0, une seule régénération du parc ; levelup-58 porte le tag v1.3.9 et la montée de LevelUp, avec
le correctif hôte des bots non épinglés (détail transmis : `replayidentity.BotIdentities`). Gel proposé
par le pilote : main@5a71271 (dont #26, table des 157 médailles, autre session) + `feat/c2-manches`
(schéma 122) ; C3, C4, C7, C8 partent dans la version suivante. Rien d'autre poussé sur main du décodeur
tant que le gel n'est pas convenu. CONVENU avec levelup-58 le 15/11 : gel à main après la fusion de
`feat/c2-manches` (schéma 122) ; si la revue la bloque, gel à 5a71271. Après le gel, AUCUNE fusion sur
main du décodeur avant le tag v1.3.9 posé par levelup-58 ; le pilote relit son correctif des bots.
Ajout du 15/11 (levelup-d0, décision écrite du user rapportée) : les croix des morts sans type entrent dans
la 8.2.0 : publication par piste de la fin de vie par lâcher de commande `unit-control` ou suppression du
corps, avec sa frame. Développé par l'agent de d0 sur sa branche depuis main, relu et fusionné par le
pilote (schéma 123 après c2-manches). Gel proposé à 58 : après ce lot. ACCEPTÉ par levelup-58 le 15/11 :
gel = main après c2-manches (122) puis le lot de d0 (123) ; tag v1.3.9 au signal du pilote ; à chaque
fusion, donner à 58 le SHA et le changement de sortie en une phrase.
GEL ATTEINT le 15/11 : PR 27 (a36bf67, schéma 122) puis PR 28 du lot de d0 (52a72a8, schéma 123 : fin de
vie par commande, contrôles du pilote avec la racine de données). Signal donné à 58 ; plus aucune fusion
avant son tag v1.3.9.
Après le gel (15/11, sur demande du user « continue en te coordonnant avec les autres sessions ») : les
lots de la version suivante se préparent sur 52a72a8 sans fusion avant le tag. Ordre et numéros : C8
(124, `grammar-2026-11-15.4`) rebasé et passé à l'équivalence d'abord ; C7 (125) rebasé sans équivalence,
puis rebasé de nouveau après la fusion de C8 ; C3 + C4 (126) ensuite. 58 relancé sur l'état du tag.
58 (15/11) : rien ne le bloque ; il réserve le verrou de décodage pour replay-equiv et le gate du corpus,
tag prévu dans 2 à 3 h ; mes agents ont cessé tout décodage. C8 rebasé (97af2c1, schéma 124,
`grammar-2026-11-15.4`), goldens avec la racine, tests et lint verts ; équivalence après le message de
fin de 58 (passe d'attribution sans -update d'abord, puis -update, puis seconde passe).
Accélération (user le 15/11 : « parallélise au besoin, ça fait des jours qu'on est sur ce chantier ») : les
trois lots de la version suivante (C8, C7, C3 + C4) sont intégrés en UNE branche `feat/phase-c-suite`
(SchemaVersion 124, SchemaDesFaits 29, `grammar-2026-11-15.4`), une seule passe d'équivalence au lieu de
trois rebases successifs (chaque rebase changeait toutes les empreintes `artifact`). La revue cumulée C9.1
se fait sur cette branche, sans décodage, pendant les passes de 58 ; mesures de contrôle (C3 + C4 sur
193 films, porteurs au-delà de 8 sièges) en parallèle de l'équivalence après le message de fin de 58.
Demande de levelup-d5 (15/11) : `respawnCountdowns[].xuid` (même voie que le roster ; supprime les deux
jointures filmIndex -> xuid de d0 côté web et de d5 côté Tactique). Refusée dans le gel par levelup-58 ->
version suivante.
Correctif hôte de levelup-58 (bots non épinglés nommés passés à `Options.Bots`) : 21 films, « 343
Byrontron » gagne ses 2 vies, aucun nom humain ne change ; sièges de `4f77afc1` qui bougent (joueur_1379
1 -> 26, bots 24 -> 1) vérifiés par l'agent de présence : lecture juste (joueur_1379 listé au 26, la place 1
prend l'équipe du bot ; `replay/sieges_places.go`). Ancre périmée de `registre_objectifs.go:302`
(depuis #21) corrigée dans c2-manches.
Lot de d0 `feat/fin-de-vie-par-commande` (b4f5237, 459bbc6, base 5a71271) : `Track.EndReason` released |
deleted, `Track.ReleaseFrame`, `TrackCoverage.LifeEnds` ; 15 113 morts fraggées sur 15 290 avec un lâcher à
-2..+6 frames, 0 en fin de manche. Revue adversariale : 0 P0/P1, véhicule sans faux lâcher ; P2 : fin à la
frame même de la piste suivante comptée `open` ; `releaseFrame` d'un survivant hors de l'axe (recommandé :
pas de fin publiée pour un lâcher en fin de film) ; crédit de libfilmshell sans `lives.rs`. Renvoyé à d0.
- [x] C9.1 Revue adversariale par contexte neuf du diff cumulé du décodeur, une ronde. (Sur
      `feat/phase-c-suite` : 4 constats corrigés en 7325750, ronde 2 sans P0/P1 ; P2 en Découvertes.)
- [x] C9.2 Montée de LevelUp (branche de la release suivante) : `go.mod`, miroir du contrat du document
      (`replayDocumentSchema.ts`, fixtures), notes de version FR et EN, retrait des cinq entrées de
      `.ai/BACKLOG.md`, `make gate-push`, CI.
      v1.3.9 : montée faite par levelup-58 (`feat/monte-decodeur-v139`, 8.2.1). v1.4.0 : en attente.
      La 8.2.1 est figée (correctifs P0 seuls) ; 58 recommande la release suivante et a transmis la
      question à levelup-dd (16/11). Rien sur `release/v8.2.1` avant sa réponse.
      Réponse de dd (16/11) : release suivante, notes sous 8.3.0 à titre provisoire. Montée préparée sur
      v1.4.1 (instantané du catalogue régénéré, aucune sortie changée), branche `feat/monte-decodeur-v140`,
      rebasée sur `main` après la 8.2.1 (fee334a039) : commits locaux f2020acf26 et 1b6539ddd6, gates verts ;
      plan `PLAN_MONTEE_DECODEUR_V1_4_0_2026-11-16.md` sur la branche. Rebasée sur `release/v8.3.0`
      (2c3c646433), poussée ; première CI rouge (schéma `Hit` du contrat homonyme d'un type d'aide des
      garde-rails web, raté faute d'avoir lancé toute la suite vitest en local), corrigée par ef6f29b07f
      (renommage `ScanHit`, suite complète 12 939 tests) ; CI verte sur ef6f29b07f. `release/v8.3.0`
      avancée en rapide sur ef6f29b07f par dd (16/11).
- [!] C9.3 `replay-corpus-gate` ; régénération : celle du parc (Q3) si une sortie a changé, sinon
      régénération ciblée des films de C8.2 et C8.3 seulement ; sur go du user dans les deux cas.
      v1.3.9 : gate vert (58, 18/18, 0 FAUX, 0 PERTE, 25 décisions). v1.4.1 : équivalence 20/20, gate contre
      c96242b8aa banc 18/18 ok, 0 FAUX ; pertes du filet instruites (compteurs neufs du schéma 125, deux
      points de jauge interpolables, catalogue des socles absent de la racine de cuisson de la base).
      Après rebase, contre fee334a039 : `--strict` code 0, 18 ok ; 25 décisions v1.3.9 retirées, 8 inscrites
      (accord du user du 16/11). Régénération : [!] décision du user, portée par la 8.3.0 (une seule
      régénération avec le rejeu 3D, les temps forts et la refonte de l'interface) ; inscrite au registre.
- [x] C9.4 Plan, journal, `thought_log`, registre des reports, mémoire ; purge des caches du lot (16/11) :
      worktree `LevelUp-wt-decodeur140` retiré (aucune jonction ; archive 1 614 / 2 267 avant et après),
      branche locale supprimée (égale à `release/v8.3.0`), clone `C:\t\phasec1\m140` et racines de gate du
      scratchpad supprimés, caches `go-build-m140`, `golangci-m140`, `go-build-c1`, `golangci-c1` purgés.
      Restent, justifiés : `C:\t\phasec1\` (mesures citées par ce plan, bases copiées ; à supprimer sur
      décision du user) ; `Scripts\hifd-c5\replay\testdata` vide mais tenu ouvert par un autre processus ;
      `LevelUp-wt-partage`, racine d'hôte de levelup-58 (sans jonction, retouche `EQUIV_ARCHIVE` dans
      `child.go`), laissé à 58, prévenu ; `Scripts\hifd-medailles` et la branche distante
      `feat/monte-decodeur-v140` laissés à leurs propriétaires.

## 5. Gates communs à chaque étape (décodeur)

```
go vet ./... && go vet -tags=research ./...
DECODEUR_FILM_RACINE_DONNEES=<worktree LevelUp> go test ./...
golangci-lint run ./...
LEVELUP_UPDATE_GRAMMAR_REV / LEVELUP_UPDATE_KILLSOURCE_REV seulement à révision changée si la sortie change
replay-equiv -repo-root <wt> -originaux <xuids_originaux.txt> -out-dir C:\t\phasec1\<étape>\equiv
```

Les fixtures de contrat se testent contre un worktree LevelUp au même schéma.

## 6. Hors périmètre

Rendus (phase B), et tout ce que le §10 range en phase C second temps (véhicules, portages en plages,
munitions, entrées de déplacement, santé à chaque point, voix, téléporteurs, lecture sur deux fils).

## 7. Découvertes

- 2026-11-14 : libfilmshell ad17521 lit des données hors du périmètre de ce plan. Le user veut les
  traiter APRÈS la phase C (« on traitera ensuite les trucs hors de notre plan actuel ») : santé d'un
  véhicule au fil du match et cause de sa destruction (arme, détournement) ; trajectoires des projectiles
  et grenades en vol ; armes posées au sol ; état des zones à chaque instant ; visée verticale (`pitch`)
  et posture (`Input::stance`) ; textes du jeu (`game::text`, `game_response` : annonces et messages des
  modes). Chantier suivant à planifier à la clôture de C9, même méthode : mesure contre le film et
  l'API, crédit à libfilmshell si on reprend une table.
- 2026-11-14 (agent C5) : le `deathOffsetMs` du document s'écarte du premier paquet du film de 23 à
  161 ms selon le film (jusqu'à 5 images), alors que le zéro de l'horloge des temps forts tombe à
  -11 / +2 ms de ce paquet. Les calques posés par `matchClock.frameOfMatchMS` (drapeau, couronne, crâne,
  bombe, fin de vie par mort) héritent de cet écart. Non traité ici ; à mesurer puis trancher (C6 horloges
  ou chantier suivant).
- 2026-11-15 (agent C6) : `weaponPads` du film `a521164d` varie entre deux lancements du MÊME binaire
  (bornes `bounds` maxX 211,82 / 100,58 / 298,29, compteurs `hits`, attributions des morts) : sortie non
  déterministe, hors des champs de la phase C ; elle déplace l'empreinte `artifact` de ce film à
  l'équivalence. Non traité ; à instruire (ordre d'itération d'une map ou course probable, non vérifié)
  avant la release. Instruit le 15/11 (agent C1, `C:\t\phasec1\c1\mesure.md` §5) : NON reproduit sur main
  89c2559, ni par la cuisson seule (7 cuissons, 3 processus), ni par les faits, ni par `replay-equiv`
  (4 passes sur `a521164d`, `bcb6d393`, `1c4c63c2`, empreintes identiques) ; aucune goroutine ni map
  mutable dans le décodeur. Piste : entrées de l'hôte non figées que l'outil de mesure de l'agent lisait
  (`map_quant_bounds.json` et `film_profiles.json` modifiés dans le dépôt principal, catalogue de socles
  de l'overlay). À surveiller à la porte de C9 (deux passes d'équivalence identiques), pas de correctif.
- 2026-11-15 : libfilmshell avancé de ad17521 à 84ad5ec (10 commits du 14/11). Pour C7 (transmis à
  l'agent) : `navpoint::ZoneMarks` (équipe qui capture, contestation, compte à rebours, lettre d'une
  zone). Pour le chantier suivant : lecteur des variantes de carte `map.mvar` (objets Forge, points de
  réapparition, objectifs de chaque mode avec leur volume ; pas dans le film, utile côté cartes de
  LevelUp), catalogue des objets Forge (`forge_data.rs`, 4 305 lignes, table reprenable créditée),
  `GameMode::from_script_paths`. Le véhicule Komodo (`ab7eaefd`) est inconnu du décodeur hors tests de
  recherche ; Threat Seeker et Repair Field y sont déjà.
- 2026-11-15 (revue ronde 2 de `feat/phase-c-suite`, 7325750), deux P2 antérieurs que les corrections rendent
  visibles, non traités ici (chantier suivant) : (a) l'auteur d'un message de dégât (`replay/document_hits.go:285`)
  est lu sur l'index brut, qui vaut la place : après le départ du titulaire, les touches du remplaçant se
  résolvent « absent » (54 fois sur 11de8353 ; le remplaçant a des tirs, 0 message, 0 touche, jamais
  `sufficient`). (b) La numérotation des tirs d'une place dont les numéros s'entrelacent (11de8353, place 23,
  34, 35, 36, 34, 37...) produit des sauts de 252 à 254 : `shots` = 1 483 pour 90 tirs écrits, couverture
  0,06, verdict faux. La valeur `shots` publiée est fausse mais le verdict la déclare insuffisante : exigence pour
  la montée dans LevelUp, n'afficher tirs, touches et dégâts d'un joueur que sous `sufficient`. (c) `replay/golden_degats_test.go:115` permute les libellés « joueurs »
  et « suffisants » dans le golden.
- 2026-11-15 (agent C9, mesure du correctif du tableau sur 27 films) : 4 compteurs nommés faux sur des films à
  disposition prouvée, non traités : 11de8353 `grenadeKills` 16 pour 0 et `meleeKills` 0 pour 1, 51ebbc0f
  `grenadeKills` 6 pour 1, bcb6d393 `headshots` 4 pour 0. Les issues de manche (`DecidedRounds`) se lisent
  aussi sur une disposition non prouvée (tous les bits levés sur a521164d, sans effet observé).
- 2026-11-16 (mesure du drapeau au-delà de 8 sièges sur la v1.4.0, 9 films BTB, `C:\t\phasec1\c7\eval_entrees_5977aa7.txt`) :
  erreur absolue cumulée contre `time_as_flag_carrier_seconds` (1 382 s) : bande de 8 publiée 1 159 s, tous
  les sièges par le pont 282 s, tous les sièges par `objectives.EntryResolver` 210 s. Les entrées rattrapent
  des portages manqués par le pont (`b1bcbe24` 3686 et 5734, `58d09c44` 9502 et 6856, `443426df` 4457 en
  partie), mais trois porteurs restent faux de plus de 5 s, dont `390b1de5` 0494 (0 s à l'API, 94,3 s publiés)
  DÉJÀ dans la bande de 8 livrée. Restent manqués : `5f90df2e` 0296 et 3729, `b1bcbe24` 3463, `443426df` 5347,
  `378659b1` 4833. Bande non ouverte (inchangé) ; à reprendre au chantier suivant (BTB non supporté
  officiellement, mais le faux porteur sur la bande livrée est un défaut à instruire).
- 2026-11-16 : le harnais `replay-equiv` signale « ÉCART sur 52 étapes » quand des étapes nouvelles
  s'insèrent : il compare par position. La comparaison par nom d'étape (5 changent) est la bonne lecture ;
  à corriger côté LevelUp (comparaison par nom), chantier suivant.

## 8. Reprise de session

Lire ce fichier (cases, journal), l'annexe d'inventaire, les dernières entrées de `.ai/thought_log.md`,
`git log --oneline -10` du clone du décodeur, puis `C:\t\phasec1\` pour les mesures de l'étape en
cours.

## 9. Journal

- 2026-11-14 : plan écrit, état des lieux sur pièces en annexe. Attend la validation du user (Q1 à Q4).
- 2026-11-14 : cinq entrées du backlog film ajoutées à la demande du user, transmises par levelup-0b et
  levelup-dd (C1.5, C1.6, C8.1 à C8.3 ; texte d'origine dans `.ai/BACKLOG.md` de `release/v8.2.0`).
  Précision de levelup-dd : la régénération du parc est faite (schéma 116) ; pour les sept films et
  `5da6fd30`, régénération ciblée sur accord du user, pas une recuisson du parc. Rien sur
  `release/v8.2.0`.
- 2026-11-14 : user : Q3 validée (« Recuisson : ok »). Q1, Q2, Q4 reformulées en clair, réponse attendue.
- 2026-11-14 : user : Q1 « estimation si assez de données, sous un seuil un message concis et factuel
  qui dit qu'on n'a pas assez de données » ; Q2 « on reprend la sienne mais on le mentionne bien » ; Q4
  « oui, c'est le but recherché ». Plan validé, exécution lancée.
- 2026-11-14 : C0.2 et C0.3 faits ; C0.1 attend le signal de levelup-7c. Mesure « avant » de C1 sur 1 345
  films (`annexes_phase_c1/C1_mesure_vies_contre_libfilmshell_2026-11-14.md`) : 95,0 % des vies
  appariées, propriétaire différent sur 0,22 %, fin de vie plus tardive chez nous sur 2,8 % (médiane
  4,7 s), arrivées 1 680 contre 1 732 et départs 1 722 contre 1 875. Suite : juger ces écarts contre le
  chunk des temps forts et l'API, puis C1.1.
- 2026-11-14 : jugement contre les morts des temps forts (annexe C1) : sur les 268 propriétaires
  différents, le nôtre meurt à la fin de la vie 239 fois, celui de libfilmshell 0 fois ; nos fins de vie
  tombent sur la mort 684 fois, les siennes 4 fois. Sur accord du user (« mobilise des agents et
  parallélise »), quatre agents Sonnet lancés, un worktree du décodeur chacun (`Scripts\hifd-c1`, `-c2`,
  `-c5`, `-c6`, sans jonction), consignes communes `C:\t\phasec1\CONSIGNES_AGENTS.md` : C1 (présence et
  vies), C2 (tableau des scores complet), C5 (médailles), C6 (horloges, réapparition, tirs alliés).
  Écart à l'ordre du §4 : C5 et C6 sont indépendantes, C2 avance en parallèle de C1 sur l'association
  ligne -> joueur actuelle. Ils poussent leur branche sans PR ; fusion, renumérotation du schéma et
  release par le pilote. Équivalence sans jonction : `replay-equiv` du worktree `LevelUp-wt-partage`
  avec la retouche locale non versionnée `EQUIV_ARCHIVE` (child.go) et un `GOWORK` par agent.
- 2026-11-15 (soir) : gate du corpus de levelup-58 sur 52a72a8 (1 PERTE, 5 FAUX, puis 12 témoins de plus) instruit sur
  pièces. À accepter : bande statborg par défaut (50247b26, a349fea8 : rien de publié), émissions écartées du lot 122
  (aucun FP), score nul de l'API d'un partant (4f77afc1), objectifs refusés au-delà de 8 sièges, compteurs de couverture
  neufs, catalogue de carte absent du gate (artefact de la racine jetable, confirmé par 58). Défaut corrigé avant le
  tag : PR #29 (0c3cc96, schéma 124) ; entrées du tableau crues seulement si deux lignes d'équipe se lisent, émission
  d'assistances à canal A ou C négatif écartée ; mesure sur 27 films : 304 -> 303 joueurs publiés, tout exact 301 ->
  302, aucun exact retiré ; deux relectures. Suite (`feat/phase-c-suite`) : revue cumulée (4 constats corrigés en
  7325750, ronde 2 sans P0/P1), puis rebasée sur 0c3cc96 en un commit c89d7ca (schéma 125, faits 29,
  grammar-2026-11-15.4, objectives-2026-11-15.3 ; commits thématiques sous `backup-phase-c-suite`). Attend : tag
  v1.3.9 par 58, puis équivalence, mesures C3 et C7 et PR de la suite.
- 2026-11-15 (nuit) : PR #30 de 58 (références refigées, 20 lignes `artifact`) relue et fusionnée (fdd6f24). Trois
  PERTE du gate instruites : relais bornés à 0 = présences datées par les listes (chevauchements 0, dépassements
  0) ; `4f77afc1` sans présence, équipe non lue et vies passées au bot = une entrée humaine absente de
  `match_participants`. LevelUp est PUBLIC : pas de xuid au manifeste ; 58 a ajouté le motif `/*` aux décisions
  du gate. Gate vert (18/18), v1.3.9 taguée par 58 sur fdd6f24 ; montée v1.3.9 sur `feat/monte-decodeur-v139` (8.2.1).
- 2026-11-16 : verrou rendu par 58. Suite rebasée sur fdd6f24 (b5bd1f4). Équivalence : passe sans -update, par nom
  d'étape seuls `flag`, `flagGaugeWalk`, `damages`, `damages.stats`, `artifact` changent ; -update puis seconde
  passe : 20 identiques (5977aa7). Tests avec la racine : seuls les 4 `TestContractFixtures*`. Contrôle C3+C4 sur
  193 films à 5977aa7 (script corrigé : il mesurait l'ancienne copie) : identique à la mesure d'avant revue.
  Mesure C7 au-delà de 8 sièges (Découvertes). Fixtures : 46 fichiers lus bruts et décompressés contre les 420
  identités d'origine, 0 trouvée (outil vérifié sur un témoin positif). PR #31 fusionnée (591a121, arbre
  identique à 5977aa7), **v1.4.0** taguée et publiée avec ses changements de sortie et le crédit à libfilmshell.
  Montée de LevelUp : question posée par 58 à levelup-dd (8.2.1 figée, recommandation : release suivante).
