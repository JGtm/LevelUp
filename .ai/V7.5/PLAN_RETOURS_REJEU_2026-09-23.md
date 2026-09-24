# Plan — retours du 2026-09-23 : tirs de véhicule, positions aberrantes, fiches, équipes, CTF, Tactique, tiroir des assets

> **Statut : GO utilisateur le 2026-09-23** (« Ok avec ton plan tu peux y aller », mode ultracode) :
> recommandations du §3 validées ; questions sans recommandation (Q7, Q8, Q10, Q11, info Q13) encore
> ouvertes — elles bloquent seulement les items qui en dépendent. Règle de l'utilisateur donnée avec le
> go, voir §3.0 (elle prime sur toute autre lecture des sièges).
> **Source** : enquête sur pièces du 2026-09-23 — six enquêteurs Opus en lecture seule, pilotés ;
> rapports, brief et instruments en annexe `retours_rejeu_2026-09-23/` (voir son `README.md`). Les
> affirmations structurantes ont été revérifiées par le superviseur sur le code et les documents
> (repères `[V]` ci-dessous).
> **Contrat d'exécution** : skill `plan-execution` (ordre strict dans un lot, aucun item sans statut
> `[x]` / `[~]` / `[!]`, zéro fix hors périmètre, découvertes consignées au §8, jamais différées en
> silence).
> **Base** : `feat/v75` @ `fe2106f4b` (= `origin/feat/v75` ; code identique à `43a01721e`, base de
> l'enquête, sur tous les fichiers concernés). Documents mesurés : schéma 68, les 111 fichiers de
> `data/cache/replays/halo_infinite/` ; bases : copies du 2026-09-23 14:11.
> **Branches** : campagne `feat/retours-rejeu` depuis `origin/feat/v75` ; un worktree
> `LevelUp-wt-rr-<lot>` + une branche `feat/rr-<lot>` par lot ; fusion dans la campagne par le
> superviseur ; fusion dans `feat/v75` sur go utilisateur après CI verte. Jamais de push sur `main`.

---

## 0. Verdict par point

| # | Point signalé | Verdict | Cause racine (prouvée) | Lots |
|---|---|---|---|---|
| 1 | Ghost : aucun tir, aucun son, aucun éclair (81c02726) | Défaut du **décodeur**, pas du rendu | Le décodeur des tirs ne lit que l'événement de TÊTE des paquets `0xD2`, à offsets fixes, variante longue seule `[V]`. Aucune arme à tir continu n'entre dans le document : Ghost, canons de la Banshee, Chopper, LAAG, LMG du Falcon, tourelle du Wraith — et, à pied, le Rayon de Sentinelle. Parc : **0 tir de Ghost publié** pour 1 522 s de pilotage (40 épisodes, 15 documents) et 63 frags au Ghost. | P1, L1.5, M4a, M4b |
| 2 | Mongoose qui part seul à 0:06 et finit hors zone jouable | Faux positif du balayage + interpolation sans borne | L'échantillon de t=983 (−201,16 ; 88,24 ; z −235) n'est pas une position : la même suite de bits revient sur des cartes aux quantifications différentes. Le client interpole 70 s vers lui, puis le plaque au bord de l'écran `[V]`. | M1 |
| 3 | Madina97294 réapparaît hors carte et « vole » jusqu'au spawn | Même famille | Son premier point (t=645, z −325) précède de 4,65 s le record de création de son corps ; la création coïncide avec le vrai placement (t=692) `[V]`. | M1 |
| 4 | Fiche de XxDaemonGamerxX sans armes à 2:08 | Défaut du décodeur (image-clé) + canal jamais branché | L'unique image-clé de sa vie (t 1494) a perdu le début de sa table : 1 joueur lu sur 8 `[V]`. Les armes de naissance ne sont lues nulle part. Parc : 213 images-clés trouées sur 2 868 (7,4 %, 69 documents), 18,9 % des vies sans aucune arme publiée. | P2, P3, M3 |
| 5 | b1ad85eb : Eagle à 3 au départ, à 5 à 2:14, les deux équipes à 5 à 6:24 | **Vous avez raison** : 4 contre 4 à tout instant | Web : un joueur parti reste affiché tant que personne ne s'assoit sur SON siège `[V]`. Go : siège = index de participant (un remplaçant n'en hérite presque jamais), équipe agrégée par index (les 3 bots partagent l'index 8, dans deux équipes → aucune équipe), corps de bots jamais nommés. Le bot « 343 Hundy », présent au coup d'envoi, n'a donc aucune fiche. | P4, M2 |
| 6 | « Sprint » sur les fiches — nouveau ? jamais demandé ? | Nouveau (21/09), **jamais demandé** | Affichage décidé par le pilote le 21/09 (« l'utilisateur les veut »), commit `b832320d4` étiqueté à tort « DECISION UTILISATEUR » ; annoncé deux fois comme « un libellé minimal sur la fiche », sans votre accord. | L1.1 |
| 7 | Tactique : tout se recharge quand on change de question ou de filtre | Défaut web | Le fond de carte est rendu SOUS la condition « données prêtes » (`TacticalAnalysisView.tsx:146`) et aucune des trois lectures ne garde sa donnée précédente (`queries.ts`, pas de `placeholderData`) `[V]` : chaque changement démonte et remonte tout, fond compris. | L2 |
| 8a | ab526724 : ni drapeau, ni score, ni frise | Film archivé **avant sa finalisation** + 2 défauts web génériques | Manifeste local à 34 morceaux sur 37, sans le morceau des temps forts (fil des morts) `[V]` ; `filmcache.Write` ne réécrit jamais un manifeste existant `[V]`. D'où : pas de calage d'horloge, 0 portage, 74/74 actions sans joueur, camps non résolus, 3e capture absente. Web : bandeau figé à 0 — 0 sur une série sans camp (7 documents), piste Score jamais dessinée sur un match à sens unique (13 documents). | O1, L3, L1.2, M5 |
| 8b | ab526724 : des véhicules — erreur ou décor ? | **Décor** de la carte Forge Starboard | 1 Scorpion, 2 Wasp, 3 Warthog, identiques au centimètre dans les deux matchs Starboard du parc, 19 à 24 m hors de l'arène, une seule position (à la naissance), jamais occupés. | L1.3 |
| 9 | Tiroir des assets : « Solution » en plusieurs exemplaires | Doublon d'**affichage** | Trois assets de carte distincts (la carte actuelle + deux copies Forge de 2023) portent le même nom anglais, donc la même image `[V]` ; aucune couche ne dédoublonne sur le visuel. Catalogue : 157 cartes pour 93 images. | L4 |

---

## 1. Constats et causes, avec les preuves

### 1.1 Tirs de véhicule (annexe `RAPPORT_tirs_vehicules.md`)

- 81c02726 : G MONEY 2123 pilote le Ghost de 0:08.4 à 1:26.1 puis de 3:03.7 à 4:24.2 (78 s et 80 s) ; ses
  6 frags au Ghost (`source_tag f712c64a` = `veh_cv_ghost`) tombent tous dans ces épisodes. Les faits
  persistés portent 1 063 événements de tir, **tous d'armes personnelles** ; zéro tir de l'index 2
  pendant ses épisodes, zéro dans les 2 s avant chacun des 6 frags. Épisodes, pont slot → joueur et
  seconde porte (`vehicle_shots.go`) sont justes : ils n'ont rien à rattacher.
- `film/internal/grammar/fire_events.go` `ScanFireEvents` : ne garde que les paquets delta dont
  l'octet 0 est le type 105 en variante longue, décode à offsets fixes (`decodeFireEvent`), jette la
  variante courte `[V]`.
- Partition qui explique tout (sons de tir du lot V3F) : les 6 armes à son de tir en BOUCLE (tir
  continu) ne sont jamais vues dans un film ; les 4 armes à son de COUP le sont toutes. Témoin
  indépendant : le Rayon de Sentinelle est ramassé 540 fois dans le parc et tiré 0 fois sur 190 615
  tirs publiés.
- Défauts aval, présents même une fois le tir lu :
  - 7 des 11 tags des tables client (`vehicleShotFx.ts`, `vehicleShotSound.ts`,
    `vehicleWeaponMounts.ts`) n'apparaissent dans aucun document ; 3 tags réellement publiés en sont
    absents : `0BB6976B` (51 tirs, tourelle latérale du Falcon), `49E40D17` (13, canon du Scorpion),
    `850902EF` (25, non identifié) ;
  - Rockethog (`C7D50912`, 127 tirs publiés) **muet depuis le lot 5.8.4 du 21/09** : il ne sonne que
    pour une famille `rockethog` qu'aucun châssis ne porte ;
  - tirs et artilleurs des tourelles posés au point de NAISSANCE de la tourelle : médiane 44,7 m du
    véhicule porteur (max 119 m) ;
  - Ghost en `plasma_cool` (bleu) ; `plasma_hot` (rouge, Ravageur) existe.
- Hors du Ghost, même décodeur : en BTB, les 39 joueurs d'index ≥ 16 n'ont aucun tir à pied publié
  (tireur lu sur 4 bits au lieu de 5, leurs tirs retombent sur l'index − 16) ; 1 205 tirs publiés
  (24 documents, presque tous avec bots) ont un identifiant d'arme décalé.
- Châssis « ? » fréquents = tourelles ENFANTS (LAAG `dd7f9102`, roquettes `bcfb852f`, tourelles du
  Falcon `1a043c29` / `f4c45d71`, pièces du Wraith `001b33fc` / `233c877d`, enfants du Scorpion),
  exclues de la table par un commentaire devenu faux (`vehicle_families.go:57-61`).

### 1.2 Positions aberrantes (annexe `RAPPORT_positions_limbe.md`)

- Ce ne sont ni des « limbes » ni des bornes de quantification : réécrites en bits, 22 des 64 valeurs
  aberrantes partagent la même suite de bits, au même décalage, sur des cartes différentes (Isolation,
  Behemoth, Launch Site…) ; témoin : 1 paire sur 49 600 chez les points normaux.
- Origine : les balayages ANCRÉS bit à bit (`ScanBipedPositionsForBand`, `ScanVehicleCreationsForBand`)
  acceptent un faux en-tête ; les filtres le laissent passer (voisin réel à moins de 15 s, premier
  point d'un slot accepté d'office, vitesse faible après un long silence).
- Le client interpole sans limite (`lib/replay/replayLogic.ts` `positionAt`) `[V]` et ne lit jamais
  `Point.g`, pourtant publié pour dire « ne pas interpoler au travers » `[V]` (974 lacunes au parc,
  222 déplacent le pion de plus de 10 m à travers la carte).
- Grammaire mesurée sur les faits : aucune position de corps n'est écrite avant sa création (sauf
  Madina, 2 cas sur 416 ouvertures) ; aucun véhicule ne se déplace pendant un silence de plus de 5 s
  (21 cas sur 565, tous aberrants).
- Parc : 33 documents sur 111 ; affichage fautif des véhicules 1 344 s (allers-retours) + 1 948 s (vies
  fantômes) + 3 684 s (faux spawns).

### 1.3 Fiche sans armes (annexe `RAPPORT_fiche_armes.md`)

- Vie slot 534 : 1:59.5 → 2:19.6 ; 0 `loadouts`, 0 `inventory`. Seule image-clé de la vie : t 1494
  (morceau 9) — 1 entrée sur 8 `[V]`, rien sous le slot 536 : perte de PRÉFIXE de la table.
- `WalkKeyframeWorld` (`keyframe_world.go`) n'enchaîne pas les records par leur grammaire : faute de
  voisin attendu, il élit un candidat lointain et perd tout ce qui est entre les deux. Preuve par
  absorption : à 4:46.8 le slot 541, mort, « porte » cinq armes, celles des vivants sautés `[V]`.
- Armes de naissance : jamais dans le delta (négatif étalonné : 0/106 naissances alors que
  l'équipement y est lu 94/106 en Super Fiesta) ; très probablement dans le record NEW du bipède, que
  la marche du dépôt traverse (`consumeWeaponStateTypeInfoVariant`) sans que personne ne branche la
  lecture.
- Même zone : `spawnSetFrom` prend une image-clé FUTURE quand la vie n'en a pas encore et retire de
  vraies prises (Hydra de JGtm à 1:16.0 classée `restated`).

### 1.4 Équipes b1ad85eb (annexe `RAPPORT_equipes_b1ad85eb.md`)

- Rejeu fidèle de la logique web : Eagle {MONEY x BUTTER, Namikidori, DRghie} au départ ;
  + « Joueur inconnu » (Hanover Cat, 2535449383340628, parti à 1:16) + Claudors à 2:14 ; Cobra à 6:24
  avec FairyNectar5788 (partie à 5:22) + 343 Brew Dog.
- C1 web : `seatOccupantAt` rend `present` un occupant fini sans successeur sur le même siège `[V]`.
- C2 Go : `seat = filmIndex` ; l'appariement de repli ne chaîne presque jamais (0 sur ce match).
- C3 Go : `ScanPlayerTeams` agrège le désignateur d'équipe par INDEX ; l'index 8 porte trois bots de
  deux équipes → divergent → non publié.
- C4 : corps de bots sur index partagé refusés (`index_hors_table`, libellé trompeur) ; l'heure API des
  relais retarde d'environ 21 s sur le film (médiane −22,3 s sur 45 relais).
- Le fait qui ouvre la solution : l'index de tireur des tirs est la PLACE, et le remplaçant en hérite
  (3 remplacements sur 3) — aujourd'hui les 84 tirs de Claudors sont perdus (`noSlot`).
- Parc : 14 documents dépassent la taille d'équipe (5 630 s), 76 affichent moins de joueurs que
  l'API, 22 montrent des joueurs partis (8 376 tuile-s).

### 1.5 « Sprint » (annexe `RAPPORT_web_tactique_assets_sprint.md` §3)

- Rendu : `ReplayPlayerCard.tsx:205-209` ; lecture `playerCardReadings.ts:136-139` ; priorité
  `stanceLogic.ts`. Introduit le 21/09 (`b832320d4` schéma 65, `a9fa54784` schéma 66 : « Sprint » et
  « Saut (dérivé) » ; `702e669b5` le 22/09 : « Escalade »).
- Trace : le 19/09 vous avez demandé si on avait les événements slide/crouch/sprint/saut, puis
  accepté « un petit lot de recherche ». L'affichage sur la fiche a été écrit dans le brief par le
  pilote (20/09), décidé par lui le 21/09 à 11:50 (« DECIDE (l utilisateur les veut) »), annoncé
  comme « un libellé minimal sur la fiche du joueur du rejeu » (21/09 11:51 et 13:02), sans réponse de
  votre part sur ce point. Aucun message de votre part ne demande l'affichage.
- Poids à l'écran : Sprint 73,6 % du temps étiqueté, « Saut (dérivé) » 18 % — alors que vous avez
  refusé le dérivé comme réponse finale (21/09 22:25) et abandonné la recherche du saut (22/09).
  Durées suspectes : glissade médiane 3,8 s (max 73 s), accroupi médiane 9,9 s.

### 1.6 Tactique (annexe `RAPPORT_web_tactique_assets_sprint.md` §1)

- Question/qui/spawn → nouvelle clé `tacticalRaster` sans `placeholderData` → `isPending` → le bloc
  entier est démonté (`TacticalAnalysisView.tsx:136-146`) `[V]`, fond compris (`TacticalPlanCard`),
  puis remonté : l'image n'est pas retéléchargée (cache infini) mais le DOM est reconstruit.
- Filtre L2 → `useTacticalMatchIDs` puis `useTacticalMaps` et `useTacticalRaster` repartent sans
  donnée précédente : deux démontages successifs possibles, et le titre retombe un instant sur le GUID
  de la carte.
- Même classe que l'Escouade du 20/09 ; les autres pages (Escouade, Explorateur, Sessions, filtres) ont
  déjà `placeholderData: keepPreviousData`.

### 1.7 CTF ab526724 (annexe `RAPPORT_ctf_ab526724.md`)

- Chronologie du 22/09 : fin du match 21:35:00 ; détection à +37 s ; les 34 premiers morceaux et le
  manifeste écrits à 21:35:50 ; les morceaux 34-36 (dont le 36, temps forts) écrits à 21:36:01 `[V]`,
  jamais ajoutés au manifeste ; cuisson lancée à 21:35:59 sur 34 morceaux.
- `ScanDeaths` prend « le dernier numéro » comme morceau des temps forts (ici un morceau de
  réplication) → fil des morts illisible → toute la chaîne d'identité tombe. Le killsource répond
  « sans kill-feed » 276 fois en 13 h. La publication ne dépend PAS de la base : même cause amont.
- Seul manifeste dans ce cas sur 1 625, mais la course est structurelle : 23 matchs sur 96 sont
  détectés moins de 60 s après leur fin ; précédent `7b0d89c4` (02/09), traité comme un risque de
  datation, jamais expliqué.
- Pas le build, pas Starboard : `d6918972` (même soir, même build) et `f0220a96` (Starboard) sont
  nominaux.

### 1.8 Tiroir des assets (annexe `RAPPORT_web_tactique_assets_sprint.md` §2)

- « Solution » = `ee43d273` (actuelle, 9 matchs) + `70a5226e` et `35c4680c` (copies de 2023, nom
  canonique = uuid, nom anglais par traduction) + « Absolution » `[V]`. Même image `Solution.jpg`,
  trois ids : le dédoublonnage client par id ne retire rien.
- Catalogue servi : 157 cartes, 93 images distinctes (64 en trop, pire cas ×4 : Rat's Nest, Immoler,
  Catalyst). Données saines (0 doublon de clé) : c'est un doublon d'affichage.

---

## 2. Errata sur des lots antérieurs (à corriger dans les lots désignés)

| Date | Affirmation | Réalité | Lot qui corrige |
|---|---|---|---|
| 2026-09-08, thought_log « Point 8 — CLOS » | Sur Isolation (81c02726), « deux Ghost courts » ; « le vrai facteur limitant est le nombre de chevauchées ARMÉES » | Deux épisodes de 78 et 80 s, 6 frags au Ghost ; le tir du Ghost n'est jamais décodé. Aucun lot n'a vérifié un tir de Ghost dans un document réel. | Entrée de thought_log du jour (erratum) ; M4b |
| 2026-09-21, `b832320d4` (+ plan du décodeur, note 5.3) | « DECISION UTILISATEUR » pour l'affichage des états sur la fiche | Décision du pilote, non adossée à un message utilisateur | Entrée de thought_log du jour (erratum) ; L1.1 |
| 2026-09-21, lot 5.8.4 | `c7d50912` « ne départage pas LAAG / Gauss / roquettes » | C'est le lance-roquettes du Rockethog ; le lot l'a rendu muet | L1.5 |
| `vehicle_families.go:57-61` | « aucune tourelle enfant observée au parc » | 150+ vies de tourelles enfants au parc | M4a |
| `facts/objectives/film.go:80-81`, `replaybuild/matchfacts.go:155-156` | `7b0d89c4` a des morceaux hors manifeste | Faux depuis la restauration du 16/09 ; la cause était un archivage avant finalisation | L3 |

---

## 3. Décisions utilisateur

### 3.0 Validé le 2026-09-23 (go) — et la règle des places

- Toutes les recommandations ci-dessous sont VALIDÉES (Q1-Q6, Q9, Q12, Q13 « masqués », Q14-Q22,
  Q24-Q27). Q2 (réparation d'ab526724) et Q1 (sondes) autorisées ; Q3 (deux republications) et Q4
  (vagues A et C avant le déploiement v7.5, D en v7.5.1) validées — chaque republication reste
  annoncée avant d'être lancée, serveur arrêté.
- **RÈGLE DE L'UTILISATEUR (citation, 23/09)** : « quand un joueur part, il libère la place de sa fiche
  de joueur pour son remplaçant ok ? Depuis le début on a du mal à se comprendre avec ça, le nombre de
  joueur dans un match est fini, il y a un maximum. » Conséquences, non négociables pour L1, M2 et M4b :
  une équipe a un nombre FINI de places (taille d'équipe du mode) ; une fiche = une place ; un partant
  LIBÈRE sa place et son remplaçant (bot ou humain) prend CETTE place ; jamais plus de fiches qu'il n'y a
  de places ; un joueur parti ne reste jamais affiché. Cela répond à Q23 (oui) et fixe Q20 (la place
  libérée reste visible, vide, jusqu'à l'arrivée du remplaçant).
- Réponses de l'utilisateur (23/09, même soir) aux questions ouvertes :
  - Q7 mortier du Wraith : **rouge** (`plasma_hot`) ;
  - Q8 canons du Chopper : **rouge** (`plasma_hot`, forme plasma) ;
  - Q10 : « ça dépend si c'est un Falcon LMG ou lance-grenades, le Falcon peut avoir l'un ou
    l'autre ». Lecture retenue (DÉDUITE, à confirmer par la mesure du lot) : `0BB6976B` tire au COUP
    et il est observé, alors que la LMG est une arme à tir continu (jamais décodée aujourd'hui) →
    `0BB6976B` = **lance-grenades** du Falcon ; la LMG reste `00015cd3`, attendue au décodage du tir
    continu (M4b). Le rendu suit la variante réellement montée (tourelle `1a043c29` vs `f4c45d71`) ;
  - Q11 Wasp : « il y a les deux modes de tir et normalement on sait les distinguer, j'avais complété
    la recherche dessus » → retrouver cette recherche (`.ai/V7.5/film_re/`, V3F : `11725dc4` = coup,
    450/min ; `d3c407ed` = son en boucle, 600/min ; Notion en lecture seule) avant toute modification ;
    sans pièce, l'entrée actuelle reste et l'écart est signalé ;
  - info Q13 : **oui**, les 6 véhicules garés sont visibles en jeu sur Starboard (décor réel). La
    décision validée reste « masqués » ; à reproposer au point d'étape (afficher en « élément de carte »
    neutre ?) — aucun changement sans ok ;
  - Q23 : « si un humain arrive oui il remplace celui du bot normalement ».

- **DÉCISIONS DU 23/09 (nuit) — elles PRIMENT sur toute consigne antérieure, y compris le brief d'un
  lot déjà lancé (M4a) :**
  - Wasp (Q11 tranchée) : « le coup par coup c'est le lance-missile, l'autre doit être la LMG de la
    Wasp (dont on a le son normalement déjà) » → `11725DC4` (son au coup, 450/min, observé) = LANCE-
    MISSILES ; `d3c407ed` (son en boucle, 600/min, tir continu) = LMG de la Wasp. Les libellés, styles
    et montages suivent ; le son actuel `vehicle_shot_wasp` (reconstruit depuis `11725dc4`) est donc
    celui des missiles ; le son de la LMG est à retrouver (V3E / manifeste V3) pour M4b.
  - Bombe de la Banshee : ROUGE et PLUS GROSSE (éclair et explosion). Obus du Scorpion : explosion
    comme la grenade. Roquettes du Rockethog : explosions comme le lance-roquettes SPNKR.
  - Lance-grenades du Falcon : l'utilisateur pensait avoir le son → le reconstruire depuis la banque
    `sb_010_veh_un_falcongrenadelauncher.pck` par la méthode V3E (gains des parents actor-mixer,
    `MakeUpGain` AkPropID 6, `InitialDelay` AkPropID 59, mode de cadence du conteneur, variantes,
    plafond −1 dBTP) ; désignation à l'oreille par l'utilisateur avant câblage.
  - Véhicules de décor (Starboard, Goliath) : l'utilisateur ne les a jamais vus en jeu → MASQUÉS
    (confirmé). Question de l'utilisateur : « on n'a pas moyen de savoir s'ils sont jouables avant ? »
    → la règle actuelle est générale (lue dans le comportement écrit par le film, pas au cas par
    cas) ; sonde C2 lancée pour trouver un champ de grammaire qui le dise dès la naissance.
  - `00007CA9` (3e emplacement au coup d'envoi) : inconnu de l'utilisateur → non affiché ; recherche
    de son identité (Ghidra / données du jeu).
  - Ghidra lancé, pont HTTP accessible → sonde P1-S3 lancée.
- **DÉCISIONS DU 24/09 (réponses aux verdicts Ghidra) :**
  1. Tir continu rendu COMME THEATER : début et fin de rafale lus dans le film (vue de contrôle),
     coups simulés à la cadence propre de l'arme lue dans le tag. VALIDÉ.
  2. PAS de solution temporaire quand la vue de contrôle n'est pas atteinte : on RÉPARE la lecture
     (clôture de la vue B, M3 puis M4b). En attendant, trou nommé et compté, rejeu muet sur ce passage ;
     le balayage par la fin (0,13 %) n'est PAS porté en production.
  3. Mains nues (`00007CA9`) : VALIDÉ les trois — exclu de la dotation affichée (règle nommée), remise du
     coup d'envoi classée comme telle (pas un ramassage, compteur dédié), « Mains nues » / « Unarmed »
     nommé s'il apparaît en cours de partie. L'utilisateur précise que c'est quasi impossible en jeu
     (jeter toutes ses armes) : cas extrêmement rare, peut-être jamais sur ~10 000 matchs.
  4. Catalogue : ajouter `E9E7FF79` = bobine à fusion de Forge (l'utilisateur croyait l'avoir : « on
     avait sorti les 4 variantes de la bobine » → vérifier lesquelles sont au catalogue et ajouter la
     variante manquante) ; vérifier les libellés `2AC9C2FF` (hotrod) et `230447B1` (proto_heatwave)
     contre Calcineur / Crémateur. VALIDÉ.
  5. Sons, désignés à l'oreille par l'utilisateur (« les premiers candidats sont tous bons ») :
     lance-grenades du Falcon = réutiliser `vehicle_shot_warthog_rocket_*` (même événement, aucun
     nouveau fichier) ; LMG de la Wasp = rendus V3E `wasp_lmg/` (boucle 3P tenue pendant le tir, coup
     3P en queue) pour M4b ; missiles de la Wasp = rendu V3E rééquilibré `wasp_missiles_controle/`
     (remplace `vehicle_shot_wasp_*`). Variantes lointaines A-E : non retenues (non câblées).
  6. Forge (information de l'utilisateur) : des petits véhicules (Mongoose, Gungoose) sont souvent POSÉS
     directement sur la carte, même officielle → « posé par la carte » n'est jamais « décor » à lui seul ;
     un véhicule posé jouable que personne ne touche pendant le match serait masqué à tort par L1.3 →
     proposition d'affinage en attente (voir §9). Takamanohara : l'objet posé `3a8060e2` = une des deux
     TOURELLES FIXES (gatling / mortier) symétriques en hauteur → famille « tourelle fixe », occupable.
- **DÉCISIONS DU 24/09 (suite) :**
  - Tir continu : l'utilisateur demande si seul le Ghost est concerné. Réponse lue dans Ghidra (P1-S3) :
    le mécanisme est GÉNÉRAL, par type de prédiction du barillet (`FUN_140de87fc`) — types 2 et 4 :
    événement de tir écrit à chaque coup ; types 1 et 3 : écrit seulement si le seau à jetons le
    permet, sinon bit « gâchette tenue » dans la vue de contrôle. Toute arme à tir continu en relève,
    véhicule ou non : Ghost, canons de la Banshee, Chopper, LAAG, LMG du Falcon, LMG de la Wasp, et à
    pied le Rayon de Sentinelle (sauts du compteur mesurés sur chacun en P1). M4b est donc générique :
    cadence lue dans le tag de CHAQUE arme, gate G1/G2 par FAMILLE (pas seulement le Ghost), Rayon de
    Sentinelle compris ; la réparation de la clôture de la vue B profite à tous les joueurs.
  - Variantes lointaines des sons : non retenues (confirmé).
  - Behemoth (hors Super Fiesta) : des Mongoose / Gungoose y sont toujours posés au départ → règle du
    décor AFFINÉE, VALIDÉE : « hors de la zone jouable » s'ajoute aux conditions de L1.3 (une seule pose,
    vie jusqu'à la fin, jamais occupé) — un véhicule jouable garé dans l'aire de jeu reste toujours
    affiché. Lot M7 (vague C), zone lue dans les références de carte du dépôt, jamais une liste de
    cartes.
- **EXIGENCE DE L'UTILISATEUR (citation, 23/09)** : « faut réparer les films mentionnés mais faut faire
  attention aux régressions et surtout que les correctifs soient propres, pérennes et solides et valides
  pour les autres films et les futurs films ». Conséquences pour TOUS les lots :
  - aucun correctif propre à un film (identifiant de match, slot, carte, instant en dur) : chaque règle
    est générale, lue dans le film ou dans le contrat du document, et les trois matchs signalés
    (81c02726, b1ad85eb, ab526724) ne sont que des TÉMOINS ;
  - validation sur TOUT le parc (111 documents) avant/après : la métrique visée bouge dans le bon sens ET
    les autres calques ne bougent pas (diff par calque, changements attendus déclarés d'avance) ;
  - lots Go de publication ou de décodage : `replay-equiv` + `replay-corpus-gate` (un témoin par build,
    donc valable pour les films anciens ET les builds futurs ; le film est autoportant, jamais de profil
    par build) ;
  - films futurs : toute règle qui dépend d'une constante mesurée est un repli NOMMÉ, compté en
    couverture, avec critère de retrait ; le compteur rend visible une dérive sur un film neuf au lieu
    de la taire ;
  - les films signalés sont réparés par la REPUBLICATION du parc (vagues C et D), pas à la main — sauf
    ab526724 dont les DONNÉES (film tronqué) sont réparées par O1, et dont le défaut amont (archivage
    avant finalisation) est corrigé pour tous les films futurs par L3.

Une décision non validée bloque les items qui en dépendent (colonne « bloque »), pas le reste.

### 3.1 Autorisations et calendrier

| # | Question | Recommandation | Bloque |
|---|---|---|---|
| Q1 | Sondes de film (lecture seule, films arène 4v4, une à la fois, aucune écriture d'artefact, quelques minutes chacune) : P1 tir continu (81c02726 puis 8a485699), P2 marche d'image-clé (morceau 9 de 81c02726, morceaux 1-2 de a0c36016), P3 armes de naissance (81c02726, b1f01a33, a0c36016), P4 entités d'équipe (b1ad85eb) | Oui, dans cet ordre (P1 d'abord : le Ghost) | P1-P4, vague D |
| Q2 | Réparer ab526724 (O1), serveur arrêté, prévenu avant | Oui (seul match touché ; le film est encore servi mais expirera) | O1 |
| Q3 | Deux montées de schéma et deux republications : 69 **depuis les faits** (sans décodage, rapide) puis 70 **avec re-décodage** des 111 films (et backfill killsource si la révision le chaîne), serveur arrêté | Oui aux deux | vagues C et D |
| Q4 | Calendrier v7.5 : ces correctifs avant ou après le déploiement ? | Vagues A et C avant (web + republication depuis les faits : une seule recuisson prod, au schéma 69) ; vague D (dépend de P1, re-décodage) en v7.5.1 | ordre des vagues |

### 3.2 Véhicules (la table de vérité complète est au §5)

| # | Question | Recommandation | Bloque |
|---|---|---|---|
| Q5 | Ghost, canons de la Banshee, Chopper, LAAG, LMG du Falcon tirent en continu tant que la gâchette est tenue : rendu en rafale à la cadence de l'arme + son PROLONGÉ pendant le tir (pas une boucle qui relance l'attaque) ? | Oui | M4b |
| Q6 | Ghost : éclair plasma ROUGE (`plasma_hot`, teinte du Ravageur) ; même teinte pour les canons de la Banshee, la tourelle du Wraith et le Shade ? | Oui | L1.5, M4a |
| Q7 | Mortier du Wraith : bleu (actuel) ou rouge ? | — (votre connaissance du jeu) | M4a |
| Q8 | Canons du Chopper : cinétiques ou plasma ? | — | M4a |
| Q9 | Rockethog : rejouer le son « roquettes » validé le 31/08 ? | Oui | L1.5 |
| Q10 | Falcon, tourelle latérale qui tire `0BB6976B` : lance-grenades ou canon Gauss ? | — | L1.5, M4a |
| Q11 | Wasp, arme `11725DC4` (coup, 450/min) : autocanon ou missiles ? | — | M4a |
| Q12 | Un artilleur (Warthog, Falcon, Wraith) est dessiné SUR le véhicule porteur, sans marqueur de tourelle séparé ? | Oui | M4a |
| Q13 | Véhicules de décor (Starboard, Goliath) : masqués (comme les Falcon/Pelican de décor) ou « élément de carte » neutre (comme les tourelles bannies) ? Et pour info : en jeu, voit-on 6 véhicules garés ~20 m au sud de l'arène de Starboard ? | Masqués | L1.3 |

Déjà tranché, non reposé : son de la LAAG du Warthog = son de la tourelle LMG (décision du 22/09).

### 3.3 Positions, fiche, équipes

| # | Question | Recommandation | Bloque |
|---|---|---|---|
| Q14 | Pendant une lacune de réplication d'un joueur : pion tenu à sa dernière position en pâlissant, ou masqué jusqu'au point suivant ? | Tenu, pâli | L1.4 |
| Q15 | Ne plus publier les vies de 1-2 points antérieures à la création du corps (20 au parc) ni les vies d'un point hors carte (22) : elles viennent du balayage, pas du film (revient sur « si le film le dit, on publie » du 14/09) ? | Oui (comptées en couverture) | M1 |
| Q16 | États de mouvement sur la fiche (Sprint, Saut (dérivé), Escalade, Accroupi, Glissade) : retirer, garder, ou derrière un réglage ? | Retirer de la fiche ; les données restent dans le document | L1.1 |
| Q17 | En attendant M3, arme inconnue sur une fiche vivante : (a) cellules vides « armes non lues » (actuel), (b) dotation du mode « présumée » avec marqueur, (c) seulement ce que le film a dit à l'instant (dernière prise, dernier tir) ? | (a) | — |
| Q18 | Une fois les armes de naissance lues (M3), supprimer la lecture « à venir » en début de vie (elle peut montrer une arme ramassée plus tard) ? | Oui | M3 |
| Q19 | Réparer la marche d'image-clé (M3) : défaut distinct du « résidu du film dense » clos le 22/09, touche aussi les films arène (7,4 % des images-clés) ? | Oui, après P2 | M3 |
| Q20 | Pendant un relais (partant sorti, remplaçant pas encore arrivé) : la place reste visible vide, ou disparaît ? | Visible vide (la grille ne bouge pas) | M2 |
| Q21 | Joueur présent sans corps (pas encore apparu, chargement) : tuile « pas encore apparu » ? | Oui | M2 |
| Q22 | Départ pendant qu'il est mort : la tuile sort à la fin de sa dernière vie, ou à la première image-clé où il n'est plus là (≤ 20 s, lu dans le film) ? | Première image-clé | M2 |
| Q23 | Mécanique : un bot remplace immédiatement le partant et prend SA place, puis un humain qui rejoint prend la place du bot — exact ? (c'est ce que lisent les tirs : 3 remplacements sur 3) | — (votre confirmation) | M2 |
| Q24 | Équipe et présence lues dans le film (option A, re-décodage) plutôt qu'un repli sur l'API (option B, contraire à l'ADR 0034 D-9) ? | A | M2 |

### 3.4 Score, Tactique, tiroir

| # | Question | Recommandation | Bloque |
|---|---|---|---|
| Q25 | Match à sens unique (3-0) : piste Score « égalité jusqu'à la 1re capture, puis le camp qui marque en tête » ? | Oui | L1.2 |
| Q26 | Tactique, pendant la mise à jour : ancien calque estompé sous « Mise à jour… », ou effacé (fond seul + indicateur) ? | Estompé | L2 |
| Q27 | Tiroir des assets, deux versions d'une carte à deux noms FR (Starboard/Tribord, Perilous/Périlleux, Curfew/Couvre-feu, Salvation/Salut) : nom traduit ou anglais ? | Traduit | L4 |

---

## 4. Vagues et lots

### 4.0 Règles communes

- Périmètre FERMÉ par lot (fichiers listés). Tout autre fichier = hors périmètre → §8, pas traité.
- Aucun agent ne lance serveur, Vite, navigateur, backfill ou décodage de film ; aucun n'ouvre les
  bases vivantes. Décodages (sondes P, clôture de vague D) : superviseur seul, un à la fois, sous le
  verrou `filmproc.AcquireSolo`, jamais un BTB en sonde, jamais en boucle.
- `go` : une commande à la fois par worktree, `GOCACHE` privé (`<worktree>/.gocache-<lot>`),
  `PATH=C:\msys64\ucrt64\bin;…` + `CGO_ENABLED=1` si DuckDB est tiré ; `GOLANGCI_LINT_CACHE` isolé.
- Web : jonction `node_modules` posée par le superviseur, délier AVANT tout `git worktree remove`
  (PowerShell `(Get-Item …).Delete()`, puis vérifier) ; `tsc -b` après purge de `node_modules/.tmp` ;
  vitest hors sandbox.
- Invariants : title-agnostic (capabilities), strings FR + EN, couleurs par jetons, fichiers ≤ 500 L,
  fonctions ≤ 80 L, lint 0 issue, tout test supprimé ou renommé retiré de la baseline JSONL dans le
  même commit, `openapi.yaml` + `make generate-types` régénérés à toute montée de schéma, aucun chemin
  d'installation du jeu en clair (ratchet himap), aucun heuristique sans nom : un repli est NOMMÉ,
  COMPTÉ en couverture, inscrit au registre `facts/fallback` avec date et critère de retrait.
- Clôture d'un lot : gate vert, items statués, section du lot mise à jour ici, rapport au superviseur
  (fait / non fait + justification / découvertes / mutations jouées). Commits sur la branche du lot,
  préfixe `rr(<lot>)`. Pas de push par les agents. Le thought_log est écrit par le superviseur à la
  fusion (les agents rendent le texte).
- Gates sur DOCUMENTS RÉELS : les instruments de l'annexe (`instruments/<lot>/`) sont portés en test
  `research` Go ou en script de gate du lot ; un chiffre « avant » est rejoué avant de coder (le parc
  a pu changer).

### 4.1 Vue d'ensemble

| Vague | Lots | Condition de départ | Republication |
|---|---|---|---|
| A | L1 rejeu web, L2 Tactique, L3 film non finalisé, L4 tiroir des assets (en parallèle) + P1 (superviseur) | go + Q1, Q6, Q9, Q10, Q13, Q14, Q16, Q25, Q26, Q27 | aucune |
| B | O1 réparation ab526724 ; P2, P3, P4 | Q1, Q2 ; L3 fusionné pour O1 (recommandé, non obligatoire) | ab526724 seul (`--one`) |
| C | M1 positions, M4a véhicules sans décodage, M5 score | vague A fusionnée ; Q3, Q12, Q15 (+ Q7, Q8, Q11 pour M4a) | schéma 69 depuis les faits |
| D | M2 équipes, M3 images-clés + armes de naissance, M4b tir des véhicules | verdicts P1-P4 ; Q3, Q5, Q18-Q24 | schéma 70, re-décodage des 111 films |

Tailles : L1 M, L2 S, L3 M, L4 S, O1 S, P1 M, P2-P4 S, M1 M, M4a M, M5 S, M2 L, M3 L, M4b L.

### 4.2 Vague A

#### L1 — Rejeu web, sans schéma (`LevelUp-wt-rr-l1`, `feat/rr-l1`)

Périmètre : `apps/web/src/features/match-replay/**`, `apps/web/src/lib/replay/**` (fichiers nommés
ci-dessous + leurs tests), i18n de la feature.

- [x] [`996f948f5`] **L1.1 États sur la fiche** (Q16). Si « retirer » : supprimer le mot (`ReplayPlayerCard.tsx`
  déstructuration + `:200-209`), le champ `stance` de `playerCardReadings.ts`, `stanceLogic.ts`
  (plus aucun appelant), la clé `stanceKind` FR/EN + contrat ; la couche de contrat du document
  (`replayDocumentSchema.ts`, `replayNormalize.ts`, `replayReadyTypes.ts`) reste. Test : « la fiche
  n'affiche aucun état de mouvement sur un document qui en porte ». Si « réglage » : bascule neuve
  « États de mouvement sur les fiches » (`useReplaySettings.ts`, `ReplaySettingsLayers.tsx`, FR + EN),
  défaut selon Q16, visible seulement si `coverage.stances.scanned`.
- [x] [`9ad997d1d`, commentaire des trois lecteurs `59d378a6f`] **L1.2 Score** (Q25). `scoreBannerLogic.ts:139` : bandeau `null` dès qu'une série publiée n'a
  pas de camp (plus de 0 — 0 inventé). `scoreTimeline.ts` `leaderStates` (l. 402-415) +
  `useReplayTimeline.ts:275-285` : quand toutes les séries publiées ont un camp, les camps absents
  valent 0 (règle déjà écrite en tête du module). Tests rouges avant : 3-0 à camp identifié → piste
  présente ; série sans camp → bandeau `null`.
- [x] [`98fe73fda` ; revue `23ad43977` ; contrôle de parc `59d378a6f` : le décor exige aussi une naissance à la frame 0] **L1.3 Véhicules de décor** (Q13). `vehiclesLayer.ts`, à côté de `FAMILLES_NON_JOUABLES` : une
  vie dont le film ne réplique la position qu'UNE fois, à la naissance (`samples.length === 1` et
  `samples[0].t === t0`), qui court jusqu'à la fin du film (`end === 'film_end'`) et n'a aucun
  occupant = objet de décor ; exclue du prédicat « embarqué ». Tests : fixtures ab526724 771-778 et
  d8b13ec2 768 (rouges : dessinés aujourd'hui) ; négatifs 0301037e 779 et tourelles de bfecd02b.
- [x] [`6f256a143` ; revue `23ad43977` (liseré, anneaux, halo pâlis pendant la lacune)] **L1.4 Lacunes de piste** (Q14). `lib/replay/replayLogic.ts` : `positionAt`, `trailAt`,
  `altitudeAt` respectent `Point.g` (aucune interpolation ni segment au travers) ; rendu du pion
  pendant la lacune selon Q14 (fichiers de dessin du pion dans `layers/`). Tests vitest au travers
  d'un point `g`.
- [x] [`03b1da54c` ; revue `cb36f3c31`, `cedc64cc8` (instrument de la fixture versionné) ; Wasp (Q11) laissée en l'état, écart signalé] **L1.5 Tables d'armes de véhicule sur les tags OBSERVÉS** (Q6, Q9, Q10). Ghost → `plasma_hot`
  (entrée gardée : elle servira dès M4b) ; Scorpion `49E40D17` (style + son `vehicle_shot_scorpion`,
  montage) ; Falcon `0BB6976B` selon Q10 ; Rockethog `C7D50912` audible sans dépendre d'une famille
  qu'aucun châssis ne porte (son « roquettes » validé le 31/08) ; `replaySound.ts` : prendre la vie
  du porteur qui COUVRE l'instant du tir, pas la première (l. ~532) ; commentaires faux corrigés
  (`vehicleShotSound.ts:42`, `vehicleShotFx.ts:49`). Garde-rail : fixture datée
  `vehicle_weapon_tags_observed.json` (sortie de l'instrument de parc) ; toute clé de table est
  observée OU inscrite dans une liste `attendusNonObserves` datée (source V3F, retrait à M4a) ; tout
  tag d'arme de véhicule observé ≥ 5 fois a une entrée ou une ligne « inconnu » motivée
  (`850902EF`).
- Gate L1 : `Remove-Item -Recurse -Force node_modules\.tmp; npm run typecheck; npm run lint;
  npm run lint:colors; npx vitest run src/features/match-replay src/lib/replay` ; knip ; verdict
  visuel utilisateur sur 81c02726 (pions sans vol au travers des lacunes, fiches), bc60b4d9 (piste
  Score d'un 3-0), fb1a1a72 (bandeau muet), ab526724 (décor absent), un match à Rockethog
  (4f77afc1 ou f2966f08 : son des roquettes).
- Clôture L1 : fusionné dans la campagne (`0a608ae13`, intégration de la vague A) ; mesure de parc
  `8161669d8` (111 documents : 0 régression). Verdict visuel utilisateur : EN ATTENTE.

#### L2 — Tactique : le fond ne bouge plus (`LevelUp-wt-rr-l2`, `feat/rr-l2`)

Périmètre : `apps/web/src/features/tactical/**` (+ leurs tests), i18n de la feature.

- [x] [`f1daed909` ; revue `d904fed2d` : `keepPreviousData` remplacé par `precedenteDuMemeJoueur` (réponse gardée à portée égale joueur/titre/carte)] L2.1 `queries.ts` : `placeholderData: keepPreviousData` sur `useTacticalMatchIDs`,
  `useTacticalMaps`, `useTacticalRaster` (pas sur `useTacticalCellule`) ; `useCoequipierOptions`
  mémoïsé.
- [x] [`5bb656d33`] L2.2 `TacticalPage.tsx` : `key={scope.carte}` sur `TacticalAnalysisView` ; `enChargement` =
  « aucune donnée encore » ; commentaire `:109-120` mis à jour dans le même commit.
- [x] [`5bb656d33` ; revue `d904fed2d` (échec du périmètre prime, tout le périmé estompé) ; parc `af8caff1e` (coéquipier introuvable nommé)] L2.3 `TacticalAnalysisView.tsx` : premier chargement → cadre + fond posés, indicateur
  par-dessus ; relecture → rien n'est démonté, `aria-busy`, calque et KPI selon Q26, mention
  « Mise à jour… » (FR + EN) ; légende, unité et source calculées depuis `raster.data.question`.
- [x] [`5bb656d33` : `TacticalPlanFond.tsx`] L2.4 `TacticalPlanCard.tsx` : cadre + `<img>` du fond extraits dans un composant qui ne reçoit
  que `mapId` et le calage ; le canvas du calque en enfant.
- [x] [`5bb656d33`, `d904fed2d`, `af8caff1e` ; mesure de parc rejouable `e06031d66` (109 cartes, 111 documents)] L2.5 Tests : `TacticalAnalysisView.fond.test.tsx` (vraie `useTacticalRaster`, `api.post`
  moqué, réponse différée : pendant l'attente, pas d'indicateur de remplacement, même nœud `<img>`
  connecté, `getBlob` appelé une fois — rouge aujourd'hui) ; variante filtre (`matchIds` →
  `null` → autre liste) ; `TacticalPage` : vignette et titre stables pendant l'attente ; réécriture du
  test « EN ATTENTE » (`TacticalAnalysisView.test.tsx:128-133`).
- [x] [`f1daed909` ; revue `d904fed2d` : clé lue multi-lignes] L2.6 Garde-rail `queriesPlaceholder.guard.test.ts` : toute lecture de `features/tactical/
  queries.ts` dont la clé embarque `hashFiltre(` déclare `placeholderData` ; allowlist datée
  (2026-09-23) d'une entrée, `useTacticalCellule`, avec sa raison.
- Gate L2 : typecheck après purge, lint, vitest `src/features/tactical` ; verdict visuel utilisateur
  (ouvrir une carte, passer « Où je meurs » → « Où je tue », cocher une session : le fond ne
  clignote pas).
- Clôture L2 : fusionné (`c51b45489`). Verdict visuel utilisateur : EN ATTENTE.

#### L3 — Un film n'est archivé, décodé ni cuit que FINALISÉ (`LevelUp-wt-rr-l3`, `feat/rr-l3`)

Principe (grammaire du film) : un film est finalisé quand son manifeste porte son morceau de temps
forts (`chunk_type 3`), par définition le dernier (1 624 manifestes sur 1 625 le portent).

Périmètre : `internal/games/halo_infinite/film/filmcache/write.go`, `internal/sync/haloclient/
halo_client_film.go`, `internal/sync/replayartifacts/cuisson.go`, `internal/sync/killcollector/
cache_films.go`, `internal/replaybuild/filmfacts_cuisson.go`, `internal/games/halo_infinite/film/
replay/deaths_source.go`, `internal/archlint/` (ratchet), commentaires de `facts/objectives/film.go`
et `replaybuild/matchfacts.go`, leurs tests.

- [x] [`2a5ba69c9` ; revue `8e4586b26` (sur-ensemble exact entièrement gardé)] L3.1 Prédicat unique `Finalise(chunks)` ; `Write` refuse de valider un manifeste sans type 3
  (`ErrFilmNonFinalise`) et REMPLACE un manifeste existant sans type 3 par une liste qui le complète
  (sur-ensemble exact par index) — seule réécriture permise.
- [x] [`2a5ba69c9` ; étendu à `GetFilmChunkURLs` et au manifeste du CACHE non finalisé (relu à l'API)] L3.2 `fetchFilmChunks` : manifeste API sans type 3 → `ErrFilmNonFinalise` (ni 404 ni panne).
- [x] [`2a5ba69c9` ; revue `73ad0138d` : poste `reportes` distinct de `sansFilm`, film non finalisé au-delà de `DelaiDeFinalisation` = 15 min → WARN + compteur distinct] L3.3 `persistFilmToCache` : cuisson reportée au cycle suivant, compteur + `slog.InfoContext`.
- [x] [`2a5ba69c9`, ÉCART DÉCLARÉ : erreur `ErrFilmNonFinalise` et non `found=false` (hors ligne, `found=false` poserait le marqueur TERMINAL `MBitFilmAbsent`) ; revue `e2d97d4ed` : le killsource sort « sans kill-feed » sans marqueur, INFO, jamais ERROR] L3.4 `LocalCacheFilms` : un manifeste local sans type 3 compte comme absent (le réseau le
  complète : ab526724 se répare seul côté killsource).
- [x] [`2a5ba69c9` ; revue `a0b05c3e2` : branchement testé, manifeste illisible refusé, repli « dernier numéro » inscrit au registre `facts/fallback`] L3.5 Chargement de la cuisson : refus d'un film sans type 3 ou dont des morceaux sont hors
  manifeste, erreur typée comptée « écartés » (jamais d'artefact dégradé silencieux).
- [x] [`2a5ba69c9` : `ErrFilSansTempsForts` ; parc : type 3 = dernier index sur 1 625 / 1 625, aucune sortie changée] L3.6 `ScanDeaths` : le morceau des temps forts choisi par son TYPE quand le manifeste est là
  (plus « le dernier numéro ») ; erreur typée distincte si le dernier morceau est de type 2.
- [x] [`a4e4615a3` : `facts.Rev` INCHANGÉE (killsource-2026-09-22.2), empreinte seule recopiée, puis de nouveau par `a0b05c3e2`] L3.7 Commentaires « doc inversée » sur `7b0d89c4` corrigés.
- [x] [`2a5ba69c9` : NON, `FilmFactsEntete` ne porte pas l'inventaire des morceaux — consigné §8.20] L3.8 Établir si l'en-tête des faits persistés porte l'inventaire des morceaux ; sinon, le
  consigner (§8) — O1 renomme les faits d'ab526724 à la main.
- [x] [`f89bae9d1` ; revue `8e4586b26` : comparaisons d'ordre, compte gelé par fichier, allowlist datée de 3 sites (§8.23)] L3.9 Ratchet archlint : aucune comparaison au type 3 hors du prédicat.
- Tests rouges avant : `Write` sans type 3 refusé ; `Write` qui complète un manifeste partiel ;
  `LocalCacheFilms` sur manifeste partiel → absent ; fetch d'un manifeste API partiel → erreur typée ;
  cuisson d'un répertoire « 34 au manifeste + 3 hors manifeste » refusée ; `ScanDeaths` sur un film
  dont le dernier morceau est de type 2.
- Gate L3 : `go build ./... && go vet ./...` ; `go test` des paquets touchés + `./internal/archlint/`
  + `./internal/himap/` (ratchet du chemin du jeu) ; `go test -tags=integration -p 1
  ./internal/sync/...` (sync touché) ; `make go-api-lint` ; parc : 0 manifeste sans type 3 hors
  ab526724 (`instruments/ctf_ab526724/manifests.mjs`).
- Clôture L3 : fusionné (`3748ca8e5`) ; découvertes D4-D7 consignées au §8 (20-23), D1-D3 corrigées
  dans le lot (`e2d97d4ed`, `73ad0138d`).

#### L4 — Tiroir des assets : une carte par visuel (`LevelUp-wt-rr-l4`, `feat/rr-l4`)

Périmètre : `apps/go-api/internal/service/asset_service.go` (+ un fichier voisin si besoin),
`asset_service_test.go`, `apps/web/src/features/asset-drawer/assetDrawerLogic.ts` (commentaire
périmé l. 18-23). Coordination : la campagne perf a touché `AssetDrawer.tsx` et `useAssetDrawer.ts`,
hors de ce périmètre.

- [x] [`df59448f5` : `asset_map_cards.go` ; mesure sur copie : 157 cartes / 93 images → 93 / 93] L4.1 `AssetService.ListMaps` : après résolution de l'image, UNE entrée par `ImageURL` ;
  représentant par ordre total — libellé FR traduit (Q27) d'abord, puis `NameFR` non vide, puis le
  plus petit `ID` ; sortie triée par `NameEN` puis `ID`. Le dépôt garde son grain « un asset par
  ligne » (test D15 vert).
- [x] [`df59448f5` ; revue `2262e2d2b` (tri anglais, recherche avant regroupement, entrée intacte)] L4.2 Tests rouges avant : les trois « Solution » + « Absolution » → 2 ;
  Starboard ×4 → une carte au libellé retenu ; tri.
- [x] [`df59448f5`] L4.3 Commentaire de `assetDrawerLogic.ts` corrigé.
- Gate L4 : `go test ./internal/service/...`, `make go-api-lint` ; après redémarrage (superviseur) :
  `GET /api/v1/assets/halo_infinite/maps?q=Solution` → 2 éléments ; catalogue complet → 93
  éléments, `image_url` toutes distinctes.
- Clôture L4 : fusionné (`62ad5ebf7`). Contrôle après redémarrage (superviseur) : EN ATTENTE.

#### P1 — Sonde : où le film porte le tir continu (superviseur, après Q1)

- Films : 81c02726 puis 8a485699 (arène), un à la fois, sous verrou, aucune écriture d'artefact.
- S1 : marcheur de liste complète existant (`r7_marche_liste_research_test.go`) sur les fenêtres de
  Ghost — tout événement (tête ou non) dont une référence désigne le Ghost (769/771) ou son pilote
  (514/541) ; tout type 36 hors tête ; tout type 36 de tête écarté par la garde de 113 bits, avec sa
  longueur.
- S2 : deltas des composants de l'entité Ghost (`ti=40`) et de son objet arme pendant l'épisode —
  quel composant bascule au début d'une rafale, à la cadence de l'arme ; témoin décalé de ±60 s.
- S3 si S1 et S2 sont négatifs : Ghidra en lecture seule (chemin d'émission d'`action_weapon_fire`
  pour un canon à son en boucle contre un canon à coup). Stop : au 2e négatif sans progrès, arrêt et
  retour à l'utilisateur (règle du 22/09).
- Gate de la sonde : un signal dont une occurrence précède chacun des 6 frags de G MONEY dans les 2 s,
  absent au témoin décalé, nul hors de ses épisodes. Verdict écrit ici (§9) avant M4b.
- [x] Verdict (§9, note `retours_rejeu_2026-09-23/SONDE_P1_tir_continu.md`, `fee612d28`) : S1 et S2
  NÉGATIFS pour le tir par tir, positifs sur les touches et le compteur ; S3 (Ghidra) POSITIF : le tir
  continu est dans la vue de contrôle (branche `feat/rr-ghidra`, pas encore fusionnée dans la campagne).

### 4.3 Vague B

#### O1 — Réparation des données d'ab526724 (superviseur, après Q2, serveur arrêté, prévenu avant)

- [x] [§9, nuit du 23/09] Renommer `data/cache/film_manifests/ab526724.json` → `ab526724.json.partiel-34` (pièce gardée).
- [x] [§9] `levelup archive-films --gamertag JGtm --match ab526724 --dry-run`, puis sans `--dry-run`.
- [x] [§9] Renommer `data/cache/film_facts/halo_infinite/ab526724.filmfacts.bin` (sinon la recuisson
  rejoue les faits du film tronqué).
- [x] [§9 : 35,6 s, pic 415 Mo] `levelup backfill-replay --one ab526724-3684-4335-b759-a18edcccc137` (jamais sans `--one`).
- [~] [§9 : killsource laissé au post-sync (le CLI n'a pas de `--match`), dérivés à la prochaine séquence habituelle ; les lignes `*_latest` du gate en dépendent] Killsource : laisser le post-sync le reprendre, sinon `backfill-killsource` ciblé ; dérivés
  (résumé d'usage, paliers de socles) selon la séquence habituelle.
- Gate O1 (`instruments/ctf_ab526724/sweep_ctf.mjs ab526724`) : manifeste 37 entrées dont un type 3 ;
  `coverage.bridge.deathOffsetMatched > 0` ; `identity.statborgSlot.non_resolu = 0` ;
  `coverage.flagCarries` `noBridge 0`, `outOfWindow 0`, `carries > 0` ; `flagReturnZone` présent ;
  `objectives.attached = available` ; série d'équipe finale à 3 avec camp ; fin affichée vers 11:0x ;
  lignes `kill_positions_latest`, `match_death_context_latest`, `kill_openings_latest` en base.

#### P2, P3, P4 — Sondes (superviseur, après Q1, une à la fois)

- P2 marche d'image-clé : `research` du paquet `grammar` sur le seul morceau 9 de 81c02726 et les
  morceaux 1-2 de a0c36016 — tracer `WalkKeyframeWorld` (départ, élections, arrêt), chercher les
  en-têtes EXACTS `[(1<<30)|slot][35]` des bipèdes vivants (519…536), compter les entrées sans
  archétype traversées. Verdict : présence prouvée + sous-mécanisme de la perte de préfixe.
- P3 armes de naissance : marche (`DecodeFrameRecords`) avec un `HeldWeaponHook` ; pour chaque record
  NEW `ti=35`, les familles lues. Oracles : première lecture d'image-clé sans prise intermédiaire,
  arme des tirs ; témoin : lecture d'une autre vie. Films : 81c02726, b1f01a33 (départs aléatoires),
  a0c36016. Seuil pour produire : ≥ 95 % des naissances lues, accord ≥ 98 %, témoin au hasard.
- P4 entités d'équipe : `TestSiegesDesRemplacants` pointé sur `data/cache/film_chunks/b1ad85eb` +
  horodatage des paquets BOT_METADATA. Attendu : trois entités d'index 8 (désignateurs 0, 0, 1),
  entité d'index 1 absente dès l'image-clé qui suit f3117, aucune entité d'index 5.
- [x] P2 (`772b02fe6`, `SONDE_P2_marche_image_cle.md`) : perte par élection d'une fausse ancre de slot
  bas, fenêtre de 120 000 bits qui coupe un suffixe → M3.1 réécrit.
- [x] P3 (`90381fc90`, `SONDE_P3_armes_naissance.md`) : le record NEW du bipède porte les armes de
  naissance (i43/i44/i46), aucun chemin de production ne les lit → M3.2 précisé.
- [x] P4 (`a49237d3d`, `SONDE_P4_equipes_places.md`) : CONFIRMÉ (trois entités d'index 8, index 1
  absent, aucune entité d'index 5 ; les tirs donnent la place 5 à Claudors) → M2 précisé.

### 4.4 Vague C — montée 69, republication DEPUIS LES FAITS (aucun décodage)

Une seule montée 68 → 69, posée par le superviseur à la première fusion de la vague ; chaque lot y
ajoute sa ligne de chronique (pas de 70 dans cette vague).

#### M1 — Positions : la grammaire de la vie, des replis nommés, un client qui respecte les silences

Périmètre : `film/replay/` (`build.go`, `identity_registry_creation.go`, `lives_decoupe.go`,
`vehicle_tracks.go`, `geometry.go`, couverture, chronique), registre `facts/fallback`, web
`model/vehiclesLayer.ts`, `lib/replay/replayNormalize.ts`, contrat (`openapi.yaml`, types générés).

- [ ] M1.1 R-B1 : une vie ouverte par un record de création commence à ce record ; les points
  antérieurs sont écartés et comptés (`coverage.tracks.avantCreation`). Filtrer AVANT `ouvrir`
  (sinon `originMs` bouge).
- [ ] M1.2 R-B2 (Q15) : aucune vie avant la première création de son slot quand le slot en porte une
  (`coverage.tracks.viesAvantPremiereCreation`).
- [ ] M1.3 Repli F-1 : position hors de l'emprise jouée (même règle et même constante que `boundsOf`,
  `boundsRejectSpreads = 12`), appliqué aux points de trace, échantillons de véhicule et `spawn` ;
  3e usage → centraliser `guardOf/axisGuard` + garde-rail grep ; nommé, compté, registre daté.
- [ ] M1.4 Repli F-2 : échantillon de véhicule atteint ou quitté à travers un silence > `lifeGapUS`
  avec déplacement ; une vie de véhicule sans position restante n'est plus publiée (`NoPosition++`).
- [ ] M1.5 `VehicleSample.g` publié (même sémantique que `Point.g`) ; `vehiclePositionAt` TIENT la
  dernière position au travers.
- Tests rouges avant : vie dont le premier point précède sa création ; vie entièrement avant la
  première création ; point, échantillon, spawn hors emprise ; aller-retour au travers d'un silence ;
  négatifs : chute continue de Behemoth 771 (t 3371-3384) conservée, véhicule garé 100 s conservé ;
  sur faits réels (`research`, ≤ 3 fichiers) : 81c02726 (770 sans échantillon à t=983, Madina démarre
  à t=692), ab526724 (769/770 non publiés), 879a4dba (excursions 770 et 774 retirées) ; web :
  maintien véhicule au travers de `g`.
- Gate parc après republication (`instruments/positions_limbe/census.mjs`, `bits.mjs`) : T1-T4 hors
  garde = 0, V1 = V2 = V3 = 0, paires de motifs transverses ≥ 24 bits = 0, chutes réelles de Behemoth
  identiques, points retirés = somme des nouveaux compteurs, véhicules déplacés pendant un silence
  > 5 s : 21 → 0.

#### M4a — Véhicules : pose, registre, familles (sans décodage)

Périmètre : `film/replay/` (`vehicle_shots.go`, `vehicle_tracks.go`, `vehicle_families.go`,
publication du registre), `config/titles/halo_infinite/mappings/vehicle_weapons.toml` (neuf), web
`model/vehicleShotFx.ts`, `sound/vehicleShotSound.ts`, `model/vehicleWeaponMounts.ts`,
`sound/replaySound.ts`, `model/vehiclesLayer.ts`, contrat.

- [ ] M4a.1 Tourelles enfants posées sur leur châssis porteur (repli nommé et compté « voisin
  `slot+1/+2` de même fenêtre », mesuré 44/45 sur la LAAG ; lecture du parent dans le film si P1-S2
  le trouve — alors en M4b). Tir, artilleur et cône posés sur le porteur ; plus de tourelle dessinée
  seule à sa naissance (Q12).
- [ ] M4a.2 La tourelle nomme la variante du châssis (`bcfb852f` → Rockethog, `64b925eb` → Gauss) ;
  Gungoose reconnu par son arme `0042678E` (dessiné Mongoose aujourd'hui) ; commentaire faux de
  `vehicle_families.go:57-61` corrigé.
- [ ] M4a.3 Registre `vehicle_weapons.toml`, clé = tag OBSERVÉ dans un film ; par entrée : véhicule,
  arme, tir continu ou coup, forme, teinte, son, montage, PREUVE (documents, nombre de tirs, tag de
  dégât co-occurrent) ; publié dans le document ; libellés FR + EN ; les trois tables client indexées
  par `vehicleWeapTag` supprimées (0 code mort), liste `attendusNonObserves` de L1.5 retirée.
- [ ] M4a.4 Garde-rails : test Go « toute clé du registre figure dans la fixture datée des tags
  observés », réciproque « tout tag de classe véhicule observé ≥ 5 fois a une entrée ou une ligne
  inconnu motivée » ; ratchet web : aucun littéral de tag ni `vehicleWeapTag(` hors du lecteur de
  registre.
- Gate (`instruments/tirs_vehicules/tirs_enfants.mjs`, `sweep_w.mjs`) : tirs de tourelle à ≤ 2 m du
  porteur en médiane (44,7 m aujourd'hui) ; 100 % des tirs d'arme de véhicule publiés ont style et son
  venus du registre (ou un silence DÉCIDÉ).

#### M5 — Score à sens unique et fil des morts illisible

Périmètre : `film/replay/score_team_identity.go`, couverture, `film_scan.go` (message l. ~404),
contrat.

- [ ] M5.1 Preuve (a′) : si un seul slot d'équipe porte une série, le score absent vaut 0 ; si le
  registre dit X-0 et que la série finit exactement à X, cette série est le camp X. Test rouge sur une
  fixture 3-0 au gabarit de bf5ced1b ; série à 2 contre un registre à 3 → reste `unresolved`.
- [ ] M5.2 Champ de couverture « fil des morts illisible » (aujourd'hui seulement déductible des
  journaux) ; message périmé de `film_scan.go` corrigé.
- Gate : `teamIdentity unresolved` sur les documents à une seule série 6 → 0 (ab526724 après O1).

#### M6 — Registre et catalogue : décisions du 24/09 (après M4a)

Périmètre : registre des armes de véhicule (M4a), catalogue des armes / objets (libellés FR + EN),
publication des ramassages (`film/replay/document_pickups.go` et voisins), sons statiques du rejeu.

- [ ] M6.1 Sons validés à l'oreille : lance-grenades du Falcon `0BB6976B` → stems
  `vehicle_shot_warthog_rocket_*` (même événement) ; missiles de la Wasp `11725DC4` → rendu V3E
  rééquilibré (`Downloads/Halo Infinite - Sons v75/rr_2026-09-23/wasp_missiles_controle/`, gain et
  plafond du pipeline des sons) ; stems de la LMG de la Wasp déposés pour M4b ; garde-rail des assets.
- [ ] M6.2 Si M4a ne l'a pas fait : libellés Wasp (coup = missiles, boucle = LMG), bombe de la Banshee
  rouge et plus grosse, obus du Scorpion = explosion de grenade, roquettes du Rockethog = explosion du
  SPNKR ; famille « tourelle fixe » pour `3a8060e2` (Takamanohara, occupable).
- [ ] M6.3 Mains nues `00007CA9` : exclu de la dotation affichée par une règle nommée, remise du coup
  d'envoi classée « remise mains nues » (compteur, hors `unknownFamilies`), entrée « Mains nues » /
  « Unarmed ».
- [ ] M6.4 Bobine à fusion `E9E7FF79` (vérifier les variantes déjà au catalogue) ; vérification sur
  pièces des libellés `2AC9C2FF` (hotrod) et `230447B1` (proto_heatwave) contre Calcineur / Crémateur.
- Gate : tests rouges/verts ; reconstruction depuis les faits base vs branche + `replay-diff` (seuls les
  calques visés bougent).

#### M7 — Décor : « hors de la zone jouable » (après M1)

- [ ] M7.1 La règle du décor (L1.3) exige EN PLUS que la vie soit hors de la zone jouable de la carte,
  lue dans les références de carte du dépôt (`data/titles/halo_infinite/reference/map_geometry/`,
  `map_positions_jouees.json`, calage des fonds — établir laquelle est la zone de JEU, en 3D si
  possible : le Wasp de Goliath est sous le sol) ; carte sans zone connue → la règle ne masque rien
  (repli nommé, compté). Commentaire de L1.3 corrigé (le film réplique la pose avant l'origine).
- [ ] M7.2 Tests : décors de Starboard et de Goliath toujours masqués ; un véhicule posé dans l'aire de
  jeu, jamais touché (fixture construite sur le modèle des Mongoose / Gungoose de Behemoth) reste
  affiché ; parc : 13 → 13 masqués, 0 véhicule en jeu masqué.

#### Clôture de la vague C

- [ ] Fusions M1, M4a, M5 dans la campagne ; `replay-equiv` et `replay-corpus-gate` (changements
  attendus déclarés) ; CI verte au niveau job.
- [ ] Republication des 111 artefacts DEPUIS LES FAITS (Q3), serveur arrêté, prévenu avant ;
  redémarrage ; gates parc M1/M4a/M5 rejoués ; verdict visuel utilisateur sur 81c02726 (Mongoose,
  Madina), 4f77afc1 (tourelles sur leur véhicule), ab526724.

### 4.5 Vague D — montée 70, re-décodage des 111 films

Conditions : verdicts P1-P4 écrits au §9 ; Q3, Q5, Q18-Q24. Ordre : M2 avant l'item de
rattachement de M4b (les tirs se rattachent par PLACE) ; M3 en parallèle.

#### M2 — Équipes, présence et place lues dans le film (option A)

Périmètre : `film/internal/grammar/player_teams.go` (+ fichier neuf < 500 L), `facts/killsource/
botmeta.go`, faits (`film_inputs.go`, `filmfacts_encode.go`, `filmfacts_decode.go`), `film/replay/`
(`roster_entry.go`, `player_teams.go`, `sieges.go`, `identity_registry_creation.go`,
`identity_registry_scoreboard.go`, `successions.go`), web `model/seatLogic.ts`, `ui/ReplayTeams.tsx`,
`lib/replay/replayNormalize.ts`, i18n.

- P4 CONFIRMÉ (`SONDE_P4_*.md`) : 3 entités d'index 8 (désignateurs 0/0/1 = Hundy, PardonMy, Brew
  Dog), FairyNectar absente dès f3213, aucune entité d'index 5 ; « présent au départ » = présent à la
  PREMIÈRE image-clé qui porte des entités (la toute première peut être vide) ; un trou au milieu de la
  fenêtre d'une entité n'est pas un départ (MONEY à f613 = perte de marche, cf. P2) ; BOT_METADATA doit
  garder l'instant de chaque paquet ; liens bot → entité par intersection avec la fenêtre STRICTE,
  corps → entité par création dans la fenêtre LARGE ; place de Claudors = 5 lue dans ses tirs.
- [ ] M2.1 Balayage `ti=9` par ENTITÉ (slot, index, désignateur, première/dernière image-clé) ; la
  table par index devient un contrôle ; BOT_METADATA garde l'instant de ses paquets pour lier chaque
  bot à SON entité.
- [ ] M2.2 Faits : `SchemaDesFaits` monte (verdict « redécoder »).
- [ ] M2.3 Publication : `roster[].presence [{from, to, toMax?}]` ; équipe PAR ENTRÉE via son
  entité ; `seat` = PLACE (siège de la table pour les occupants du départ, place lue dans les tirs pour
  un remplaçant qui tire, sinon chaînage par équipe — repli nommé, compté) ; corps d'index de bot
  partagé nommé par le bot dont l'entité vit à sa création ; `successions.go` reste un repli compté.
- [ ] M2.4 Web : `seatOccupantAt` lit `presence` ; la règle « présent sans successeur » est
  supprimée ; regroupement par `roster[].team` ; états Q20-Q22 (FR + EN).
- Tests : témoin au gabarit de b1ad85eb (8 sièges dont un jamais joué, 3 bots index 8 de désignateurs
  0/0/1, arrivants 9 et 10) → places Eagle 5 : Hundy → Hanover Cat → PardonMy → Claudors, Cobra 1 :
  FairyNectar → Brew Dog, ≤ 4 occupants par équipe à chaque frame (mutation : réagréger par index →
  rouge) ; réécriture de `seatLogic.test.ts:125` (qui fige `[3, 5]`) ; garde-rail de propriété « à
  aucune frame une équipe n'a plus d'occupants que de places » sur toutes les fixtures.
- Gate parc (`instruments/equipes_b1ad85eb/parc.mjs`) : 0 équipe au-delà de sa capacité (14
  documents aujourd'hui), 0 tuile d'un joueur sorti (22), « moins que l'API » limité aux fenêtres du
  retard API mesuré ; b1ad85eb exactement 4 + 4 aux trois instants signalés.

#### M3 — Marche d'image-clé réparée + armes de naissance (après P2, P3)

Périmètre : `film/internal/grammar/keyframe_world.go`, `grammar/birth_loadouts.go` (neuf) + marche,
`film/replay/` (`film_scan.go`, `film_inputs.go`, `filmfacts_*`, `loadouts.go`,
`document_weapon_changes.go`, `coverage.go`, `document_chronicle.go`), web `lib/replay/rosterLogic.ts`,
`changeRefine.ts`, `ui/ReplayWeaponsRow.tsx`, i18n, contrat.

- [ ] M3.1 Marche d'image-clé — RÉÉCRIT d'après P2 (note `retours_rejeu_2026-09-23/SONDE_P2_*.md`,
  branche `feat/rr-sondes`) : les records perdus SONT dans le film (8 en-têtes exacts à t 1494, dont
  532 absent du rapport initial) ; la cause n'est PAS l'entrée sans archétype (0 mesurée) mais la
  règle d'élection `kfCand.betterThan` (`keyframe_world.go:135`) qui préfère une fausse ancre de
  génération 1 et de slot plus BAS, placée plus loin. Corriger : élection « génération 1 puis le
  candidat le PLUS PROCHE » (V2) ; recalage sur l'en-tête EXACT `[(1<<30)|slot][35]` d'un eid connu
  (n1 = 152) ; l'élection reste un repli nommé et compté (`coverage.fallbacks`) ; publier le compteur
  des bipèdes absents encadrés. Fenêtre `kfScanFenetreBits` (120 000 bits, coupe un suffixe sur
  a0c36016) : la retirer SEULEMENT si une mesure V2-sans-fenêtre sur un film dense (dad793c7) et sur le
  parc ne déraille pas (la mesure 5.20.1 datait de l'ancienne règle) ; sinon la garder et le dire.
- [ ] M3.2 Canal « armes de naissance » — CONFIRMÉ par P3 (`SONDE_P3_*.md`) : record NEW `ti=35` de la
  naissance, emplacements i43 (arme 1), i44 (arme 2), i46 (3e emplacement), famille = moitié haute ;
  45/45, 106/107, 142/142 naissances ; Super Fiesta 98,8 % contre témoin 0 %. DEUX RÉPARATIONS DE
  GRAMMAIRE PRÉALABLES : (a) la vue B d'un paquet à liste démarre à la FIN de la liste d'événements,
  dont le premier record est ce NEW (aujourd'hui `marchLocate` le saute) ; (b) grammaire au bit près
  des composants i1..i42 d'un record NEW de bipède (désalignement mesuré −599 à +731 bits avant i43).
  Puis : faits → document : entrée `loadouts` à la frame de naissance avec provenance,
  `weaponChanges[].k` (emplacement 0/1) ; reclassement des premières émissions contre la dotation de
  naissance (retire le repli futur de `spawnSetFrom`, qui efface de vraies prises). La lecture « par
  catalogue » de l'instrument est une heuristique de mesure, JAMAIS portée en production ; un repli de
  lecture (si i1..i42 ne se ferme pas à 100 %) attend la décision de l'utilisateur. Objet `00007CA9`
  au 3e emplacement au coup d'envoi : IDENTIFIÉ (sonde CA9, 24/09, `SONDE_CA9_objet_00007CA9.md`,
  branche `feat/rr-ghidra` 82d553419) = l'arme « mains nues » (`WeaponTags.unarmed` du Lua global,
  tag weap du module globals) ; le jeu la remet au 3e emplacement de chaque bipède au coup d'envoi, et
  les ramassages de classe arme à t=0 qui la portent sont cette remise, pas des prises. Traitement
  (décision de l'utilisateur attendue, proposition) : l'exclure de la dotation affichée par une règle
  NOMMÉE (famille « mains nues », constante unique) ; classer ses remises à t=0 en « remise mains
  nues » avec compteur, hors `unknownFamilies` ; entrée de catalogue « Mains nues » / « Unarmed ».
- [ ] M3.3 Web : `loadoutAt` prend la dotation de naissance comme base ; `refineWeaponsReading`
  applique les prises avec `k` ; provenance en infobulle (FR + EN) ; lecture « à venir » retirée (Q18).
- Tests : payload synthétique qui fait sauter la marche aujourd'hui ; première émission avant toute
  image-clé classée `restated` à tort ; NEW `ti=35` synthétique portant i43/i44 ; web `loadoutAt`.
- Gate parc (`instruments/fiche_armes/sweep2.mjs`, `absorb.mjs`, `dist.mjs`) : images-clés touchées
  213/2 868 → 0 ; absorptions 6 → 0 ; vies sans arme 18,9 % → ≤ 1 % hors BTB ; fiche sans arme 6,6 %
  du temps de vie → < 0,5 % ; 81c02726 : slot 534 armé dès 1:59.5, sept bipèdes présents à t 1494.

#### M4b — Tir des véhicules décodé par la grammaire (après P1)

> **VERDICT P1-S3 (Ghidra, 24/09, `SONDE_P1S3_tir_continu_vue_controle.md`, branche `feat/rr-ghidra`
> d97b178aa) — REMPLACE la conclusion de P1 ci-dessous.** Le tir continu EST écrit dans le film, dans
> la VUE DE CONTRÔLE (vue C, rang 2 de chaque trame delta) : entrée kind 0 du joueur
> (`FUN_1406d0388`), bloc d'action (`FUN_1406d025c`) — m0 = R(3) gâchettes de la main 0 (0b100 =
> principale), m2 = R(3) main 1, m4/m5 = R(2) barillets ; le tireur = index de contrôle R(5) (= index
> du roster), le véhicule vient de la monture. Mécanisme lu : le numéro de tir avance à chaque tir
> (`FUN_14202f3a0`), mais `action_weapon_fire` n'est émis que si le seau à jetons du barillet
> (types de prédiction 1 et 3, `FUN_140de87fc`) l'autorise ; à défaut, l'écrivain de la vue C pose le
> bit « gâchette tenue ». Theater rejoue ce bit et simule la rafale à la cadence du tag : les instants
> de chaque coup ne sont pas dans le film, le DÉBUT et la FIN de rafale le sont, au tick. Mesuré sur
> 81c02726 : 1 050 entrées qui tirent, toutes de G MONEY, 6/6 frags précédés d'une entrée qui tire,
> témoin −60 s à 0. Limite : la vue C n'est lue que si la vue B se ferme (34 % des paquets) ;
> couverture des épisodes 29 %/55 % (marche) → 83 %/91 % avec un balayage par la fin étalonné (0,13 %
> d'erreur). La production refuse aujourd'hui trois branches de l'entrée et lit faux deux sous-lecteurs
> du bloc d'action. Découverte : une 3e monture de G MONEY sur le Ghost 771 (≈ 4:55-5:12) manque au
> document.
> M4b devient : (1) grammaire — porter l'entrée complète `FUN_1406cd860` et corriger le bloc d'action
> (catégories 1/2 de `FUN_140c9e990`, vecteur `FUN_1431a0cbc`), mesurer l'effet sur la clôture de la
> vue B ; (2) faits — par joueur et par tick m0/m2/m4/m5 et l'arme de main → intervalles de tir continu ;
> (3) publication — une rafale par intervalle posée sur le véhicule monté, cadence du tag (Ghost
> 7,5/s, lue dans le tag : barillet +0x70/+0x74/+0x78/+0x7c), son prolongé du début à la fin ; touches
> `damage_aftermath` gardées ; contrôle : nombre de coups ≈ saut du numéro de tir ; (4) trous — vue C
> non atteinte = trou NOMMÉ et compté, jamais « pas de tir » ; (5) gate G1 : entrée qui tire avant
> chacun des 6 frags, 0 au témoin, 0 hors montures après correction de la 3e monture. M4b démarre
> APRÈS M3 (qui répare le départ de la vue B, dont dépend la lecture de la vue C).
>
> **Verdict P1 (23/09, `SONDE_P1_tir_continu.md`, branche `feat/rr-sondes`) — historique.** Mesuré sur 81c02726 et 8a485699 : aucun record 36/35/37/10 du pilote
> ou du Ghost, en tête ou hors tête (marche de liste validée par un oracle indépendant) ; aucun
> composant `ti=40`/`ti=35` qui bascule avec le tir ; vue C réfutée par étalonnage. MAIS : (1) le film
> porte les TOUCHES des armes continues (`damage_aftermath` type 0 à toute position de liste, ref1 =
> corps du pilote, source = tag du véhicule : Ghost F712C64A, Banshee FA4FAD21) — gate 6/6 frags sur
> 81c02726, 0/12 au témoin, cadence 133 ms (7,5/s) ; (2) le record 36 porte un NUMÉRO DE TIR par
> joueur (8 bits, `n mod 256`) qui saute du nombre de tirs continus non écrits (Ghost +56, Banshee +37,
> Chopper +6, canon continu du Wasp, Rayon de Sentinelle) — nombre sans instant. S3 (Ghidra, lecture de
> l'émetteur d'`action_weapon_fire` pour un canon à son en boucle) NON jouée : Ghidra n'était pas
> lancé. Doctrine de l'utilisateur : Theater montre ces tirs, donc « pas dans un canal lu » ≠ « pas dans
> le film » tant que S3 n'est pas faite. Décisions attendues : (a) ouvrir Ghidra pour S3 ; (b) en
> attendant ou à défaut, rendre le tir continu depuis les touches (un éclair par touche, véhicule →
> victime) + une rafale du nombre de tirs numérotés en repli nommé et compté ; gate G1 redéfini en
> conséquence. Le même saut de compteur distingue les deux modes du Wasp (`11725DC4` écrit, mode
> continu seulement compté).

Périmètre : `film/internal/grammar/fire_events.go`, faits, `film/replay/` (`shots.go`,
`vehicle_shots.go`, `film_inputs.go`, `filmfacts_*`), web (rendu du tir continu), et tout consommateur
de `FireEvent` recensé à l'item M4b.1.

- [ ] M4b.1 Recenser les consommateurs de `FireEvent` (rejeu, précision par arme / `weaponscan`,
  killsource) et les révisions qui montent ; si `facts.Rev` (killsource) est chaîné, prévoir le
  backfill killsource dans la clôture (Q3).
- [ ] M4b.2 Record 36 lu par la GRAMMAIRE (plus d'offsets fixes) : type 36 seul (le `0xD2` couvre
  aussi le 37 `weapon_overheat`), les références (réf 0 = unité tireuse, bipède OU véhicule), tireur
  sur 5 bits (corrige les 39 joueurs BTB d'index ≥ 16 et les 1 205 identifiants décalés), liste
  complète si P1-S1 l'exige, canal du tir continu selon P1-S2/S3.
- [ ] M4b.3 Faits : indice 5 bits, slot de la réf 0, canal continu ; `SchemaDesFaits` et
  `grammar.Rev` montent.
- [ ] M4b.4 Publication : un tir dont la réf 0 est un véhicule se pose sur lui (l'épisode ne sert plus
  qu'à nommer l'occupant ; couvre les 25 frags au Ghost sans épisode du tueur) ; rattachement des tirs
  à pied par PLACE (M2).
- [ ] M4b.5 Web : rafale à la cadence de l'arme pendant l'intervalle de tir, son prolongé (Q5).
- Tests : mini-film extrait de 81c02726 (fenêtre d'un frag au Ghost) → un tir de Ghost décodé et posé
  sur le Ghost (rouge aujourd'hui) ; ratchet Go « aucune lecture à offset fixe du record 36 hors
  grammaire ».
- Gate G2 étendu (décision du 24/09) : par FAMILLE d'arme à tir continu — Ghost, canons de la Banshee,
  Chopper, LAAG, LMG du Falcon, LMG de la Wasp, Rayon de Sentinelle — rafales publiées là où le film
  les porte, cadence propre à chaque arme lue dans son tag.
- Gate parc (`instruments/tirs_vehicules/kills_vs_tirs.mjs`) : G1 81c02726 — un tir de Ghost publié
  dans les 2 s avant chacun des 6 frags de G MONEY (0/6 aujourd'hui), aucun hors de ses épisodes, posé
  à ≤ 3 m du sprite ; G2 — frags de classe véhicule précédés d'un tir de l'arme du tueur ≥ 90 % par
  famille lue (Ghost 1/63, LAAG 0/31, Banshee 3/26 aujourd'hui) ; G5 — joueurs d'index ≥ 16 avec des
  tirs (0/39), identifiants décalés 1 205 → 0 ; Rayon de Sentinelle tiré > 0.

#### Clôture de la vague D

- [ ] Fusions M2, M3, M4b ; `replay-equiv` + `replay-corpus-gate` (goldens re-figés sur la liste
  autorisée seulement) ; CI verte au niveau job.
- [ ] Re-décodage des 111 films + republication (Q3), un film à la fois, serveur arrêté, prévenu
  avant ; backfill killsource si M4b.1 l'exige ; redémarrage ; tous les gates parc rejoués ; verdict
  visuel utilisateur sur les trois matchs signalés.

---

## 5. Table de vérité des armes de véhicule (Q5-Q12)

Seules les colonnes « tag », « observé » et « porteur » sont MESURÉES ; effet, teinte et son sont des
propositions.

| Véhicule | Arme | Tag film | Observé (tirs / docs) | Porteur mesuré | Tir | Effet proposé | Teinte proposée | Son proposé |
|---|---|---|---|---|---|---|---|---|
| Ghost | canons plasma jumelés | `00015435` (V3F, jamais vu) | 0 (63 frags) | châssis `5b80c406` | continu | plasma, nez | `plasma_hot` (Q6) | `vehicle_shot_ghost` (validé 04/09) |
| Banshee | canons plasma | `0000aa68` (jamais vu) | 0 | `c6e79dcc` | continu | plasma, nez | `plasma_hot` (Q6) | `vehicle_shot_banshee_m1` (validé 05/09) |
| Banshee | bombe à combustible | `0000aa69` ; `850902EF` ? | 0 (`850902EF` : 25, dispersés) | `c6e79dcc` | coup | explosion | à définir | `vehicle_shot_banshee_m2` |
| Wraith | mortier plasma (conducteur) | `121B4009` | 142 / 4 | `ae845375` | coup | orbe plasma | `plasma_hot` (Q7 : rouge) | `vehicle_shot_wraith` |
| Wraith | tourelle plasma (artilleur) | inconnu | 0 (2 frags) | enfant `001b33fc` | ? | plasma | `plasma_hot` (Q6) | aucun |
| Chopper | canons avant | `b40e9618` (jamais vu) | 0 | `3d4a8a5a` | continu | plasma | `plasma_hot` (Q8 : rouge) | `vehicle_shot_chopper` (validé) |
| Warthog | LAAG (artilleur) | `0c6fd911` (V3F) | 0 (31 frags) | enfant `dd7f9102` | continu | balistique | `kinetic` | son de la LMG (décidé 22/09) |
| Rockethog | lance-roquettes | `C7D50912` | 127 / 2 | enfant `bcfb852f` | coup | roquette | à définir | `vehicle_shot_warthog_rocket` (validé 31/08, muet depuis 21/09, Q9) |
| Warthog Gauss | canon Gauss | `8647925a` | 0 | enfant `64b925eb` | ? | ? | ? | aucun |
| Scorpion | canon principal | `49E40D17` (tables : `00015cfa`, jamais vu) | 13 / 2 | `f6f54e56` | coup | obus | à définir | `vehicle_shot_scorpion` (mal clé) |
| Scorpion | mitrailleuse (artilleur) | inconnu | 0 | enfant `0000d500` | ? | balistique | `kinetic` | ? |
| Wasp | `11725DC4` (Q11) | `11725DC4` | 4 / 1 | `b65b3b4a` | coup | ? | `kinetic` | `vehicle_shot_wasp` |
| Wasp | `d3c407ed` | jamais vu | 0 | `b65b3b4a` | continu | ? | ? | aucun |
| Gungoose | mitrailleuses avant | `0042678E` | 15 / 3 | `af31ab1a` (dessiné Mongoose) | coup | balistique | `kinetic` | `vehicle_shot_gungoose` |
| Falcon | LMG latérale | `00015cd3` (jamais vu) | 0 (2 frags) | enfant `f4c45d71` | continu | balistique | `kinetic` | `vehicle_shot_falcon_lmg` (validé) |
| Falcon (variante lance-grenades) | lance-grenades latéral (Q10, déduit : coup, observé) | `0BB6976B` (absent des tables) | 51 / 3 (32 frags) | enfant `1a043c29` | coup | grenade (explosion à l'impact) | `kinetic` | à trouver sur les planches ; sinon silence décidé |
| Shade | tourelle plasma | inconnu | 0 | `000df0c4` | ? | plasma | `plasma_hot` (Q6) | aucun |

---

## 6. Gates sur documents réels — état de départ mesuré (2026-09-23)

| Mesure (instrument) | Aujourd'hui | Cible | Lot |
|---|---|---|---|
| Tirs de Ghost publiés au parc (`kills_vs_tirs.mjs`) | 0 (1 522 s de pilotage) | un tir avant chaque frag au Ghost | M4b |
| Frags au Ghost / LAAG / Banshee précédés d'un tir publié | 1/63 · 0/31 · 3/26 | ≥ 90 % par famille lue | M4b |
| Joueurs BTB d'index ≥ 16 avec des tirs | 0/39 | 39/39 | M4b |
| Tirs de tourelle, écart médian au porteur (`tirs_enfants.mjs`) | 44,7 m | ≤ 2 m | M4a |
| Documents à positions aberrantes (`census.mjs`) | 33/111 | 0 hors replis comptés | M1 |
| Véhicules déplacés pendant un silence > 5 s | 21 | 0 | M1 |
| Images-clés trouées (`sweep2.mjs`) | 213/2 868 (7,4 %) | 0 | M3 |
| Vies sans aucune arme publiée | 18,9 % | ≤ 1 % hors BTB | M3 |
| Documents dépassant la taille d'équipe (`parc.mjs`) | 14/111 | 0 | M2 |
| Documents avec tuile d'un joueur parti | 22/111 | 0 | M2 |
| CTF d'arène avec drapeau publié (`sweep_ctf.mjs`) | 15/16 | 16/16 | O1 |
| Piste Score possible sur CTF d'arène | 7/16 | 16/16 | L1.2, M5 |
| Bandeau faux à 0 — 0 | 7 documents | 0 | L1.2 |
| Manifestes sans morceau de temps forts (`manifests.mjs`) | 1/1 625 | 0, et plus jamais écrit | L3, O1 |
| Cartes du tiroir / images distinctes | 157 / 93 | 93 / 93 | L4 |

---

## 7. Protocole de reprise

Relire le skill `plan-execution`, puis ce fichier (§3 : décisions validées ; §9 : journal ; cases de
la vague en cours) et `git -C <worktree> log --oneline` de chaque lot ouvert. Reprendre à la première
case non statuée. Ne pas relancer un agent sans vérifier qu'il est mort. Les verdicts des sondes P
vivent au §9 ; aucun lot de la vague D ne démarre sans eux.

---

## 8. Découvertes hors périmètre (consignées, non traitées)

1. Boucle infinie du killsource « sans kill-feed » sans marqueur terminal : ab526724 276 fois en 13 h,
   7206e05b 764, d3fe5a96 717, 29206c7c 712, d672402a 702, 91b692a8 678 (causes à instruire hors
   ab526724).
2. Écritures LUSR v2 en échec répété sur ab526724 (« Duplicate key … violates primary key » sur
   `match_skill_rank`, 21:37 → 22:18) — même famille que l'index désynchronisé relevé en prod le 23/09.
3. BTB CTF (879a4dba, 4f77afc1) : ni drapeau ni actions, par la garde d'effectif du statborg
   (`refusedByRoster`) — voulu, à confirmer côté produit.
4. Rejeu depuis les faits ≢ rejeu depuis le film quand le fil des morts est illisible
   (`replaybuild/filmfacts_cuisson.go:96` reconstruit le fil sans l'erreur) : brèche de l'équivalence S8.
5. Durées d'états de mouvement anormales (glissade médiane 3,8 s, max 73 s ; accroupi médiane 9,9 s ;
   escalade max 183 s) : intervalles qui ne se ferment pas (hypothèse) — sans objet si Q16 = retirer,
   à instruire sinon.
6. `maps_catalog` : 169 lignes sur 288 ont un nom canonique = uuid (`ops/catalog_refresh.go` recopie
   `match_registry.map_name`).
7. Tiroir des assets : branche de recherche SQL morte en production ; catalogue chargé sur toute page,
   tiroir fermé.
8. Tactique : `useTacticalMaps` relit la grille sur l'écran d'analyse pour le seul nom de la carte ;
   `match_ids` `null` et `[]` hachent vers la même clé ; `hashFiltre` resérialise la liste à chaque
   rendu.
9. La requête Q12 exclut les participants « tout à zéro » (64 sur 31 documents) : le rejeu perd leur
   nom (Hanover Cat → « Joueur inconnu »).
10. Heure API des relais en retard d'environ 20 s sur le film : `successions.go` (fenêtre −2 / +20 s)
    et `presenceFeed.ts` (lignes « a rejoint / a quitté ») en héritent.
11. `43e96765` : index 0 divergent (deux entités d'index 0, désignateurs différents).
12. Bots présents moins de 20 s : aucune équipe lue (`ScanPlayerTeams` ne lit que les images-clés).
13. Trois documents affichent une 3e colonne « sans équipe » (`a6ae19fb`, `f2966f08`, `db1b00b3`).
14. `rosterLogic.groupByTeam` / `sideResolver` lisent toujours la feuille (D4 du lot 1.9.14).
15. Faux positifs du même balayage sur les objets du monde (projectiles Forge, lot B-bis, 612 pas
    impossibles) — non traités.
16. 222 lacunes de bipède avec déplacement > 10 m : combien tombent pendant un trajet en véhicule que le
    prédicat « embarqué » ne couvre pas — à mesurer.
17. Images-clés antérieures à l'origine (morceau 1) : 0 record sur trois films de faits — légitime ou
    même défaut, à vérifier.
18. Commentaire périmé `biped_creation.go` (« ~120 bits sur ~380 ») contredit `traverse.go:103-110`.
19. Option 2 des positions (porte grammaticale au décodage : `TryDeltaAt` à l'ancre + record suivant
    valide, refus de `gen=0`) : permettrait de retirer F-1 et F-2 ; sonde d'un film à prévoir après M1.
20. (L3.8) L'en-tête des faits persistés (`replay/filmfacts_fichier.go`, `FilmFactsEntete`) ne porte
    PAS l'inventaire des morceaux (seulement VersionCodec, Schema, les quatre révisions, build,
    registre et la clé de cuisson) : `Utilisable` ne peut pas voir que des faits viennent d'un film
    tronqué — d'où le renommage manuel d'O1. Résidu borné : depuis L3 aucun fait ne s'écrit depuis un
    film non finalisé ; 0 cas au parc après O1.
21. (L3 D4) Aucun plafond de tentatives ni marqueur non terminal pour un film qui ne serait JAMAIS
    finalisé : la reprise n'est bornée que par l'horizon de rattrapage de chaque étape
    (`replayartifacts.BacklogHorizon` = 64, `killcollector.PostSyncBacklogHorizon`). Même famille
    que 8.1. L3-R1 ajoute un WARN + compteur au-delà de 15 min, pas de quarantaine.
22. (L3 D5) `cmd/levelup/cmd_archive_films.go` `filmDejaEnCache` juge sur l'EXISTENCE du manifeste,
    pas sur `filmcache.Finalise` : `archive-films` ne répare pas seul un manifeste partiel d'avant L3.
23. (L3 D6) Trois sites comparent encore au type 3 (allowlist datée du ratchet
    `archlint/film_finalise_predicate_test.go`) : `cmd/levelup/cmd_backfill_medailles_feed.go`,
    `facts/objectives/extract.go` (`chunkTypePied` ; migrer recopie l'empreinte de `facts.Rev`),
    `testfixtures/jgtm_full_match.go`. Copies locales du type 2 : `cmd/fetch_film_chunks`,
    `grammar/positions`, `objectives`. Heuristique `nums[:len(nums)-1]` non gardée par le ratchet :
    `replay/player_index.go`, `facts/killsource/index_motif.go`.
24. (L4) `AssetService.ListMaps` écrit `ImageURL` dans la tranche rendue par le dépôt (préexistant) :
    la tranche du dépôt en amont n'est pas immuable.
25. (L1) Wasp (Q11) : V3F donne un son au coup 450/min contre un son en boucle 600/min, ce qui
    contredit peut-être M1 = autocanon ; ancres renommées par leur place, valeurs inchangées. Le
    commentaire « une seule position » de `vehiclesLayer.ts` est inexact (sonde C2, §9).
26. (intégration) Une soixantaine de COMMENTAIRES Go citent encore `.ai/<document>` à son ancien
    chemin (déplacés sous `.ai/V7.5/` par `fe2106f4b`, ou avant) ; seul un chemin LU par un test
    était cassé (corrigé, `8be964717`). Ceux de `film/` entrent dans les empreintes de révision :
    à corriger dans un lot qui assume la recopie d'empreinte.

---

## 9. Journal (superviseur)

- 2026-09-23 : enquête sur pièces (six enquêteurs Opus en lecture seule, un verrou « une commande go à
  la fois », trois sondes Go sur faits persistés, aucun film ouvert) ; rapports et instruments en
  annexe ; plan proposé. Aucun code modifié. En attente des décisions du §3.
- 2026-09-23 (soir) : GO utilisateur + règle des places (§3.0), mode ultracode. Campagne
  `feat/retours-rejeu` depuis `origin/feat/v75` fe2106f4b (worktree `LevelUp-wt-rr`) ; plan et annexes
  déplacés du checkout principal vers la campagne. Vague A lancée en workflow : L1, L2, L3, L4 en
  parallèle (un worktree + une branche `feat/rr-l<n>` chacun, implémentation → revue adverse →
  correction) ; en parallèle, la VOIE DE DÉCODAGE sérialisée : O1 (réparation d'ab526724) puis P1, P2,
  P3, P4 (worktree `LevelUp-wt-rr-sondes`, branche `feat/rr-sondes`), un film à la fois. Q10 ouverte :
  dans L1.5, `0BB6976B` reste une ligne « inconnu » motivée (ni style ni son) jusqu'à la réponse.
- 2026-09-23 (nuit) : workflow `wf_1bda9ef2-c42` terminé.
  - O1 [x] : ab526724 réparé (manifeste 37 entrées dont le type 3 ; faits et artefact reconstruits par
    `backfill-replay --one`, 35,6 s, pic 415 Mo). Vérifié par le superviseur sur le document : pont
    `nominal`, `flagCarries` 31 portages (noBridge 0, outOfWindow 0, 3 captures), 2 drapeaux,
    `flagReturnZone` présent, objectifs 76/76, statborg 8/8 déduits, série d'équipe du camp 1 finale à
    3, `teamIdentity` b. Killsource non ciblable par le CLI (aucun `--match`) : le post-sync le
    reprendra ; dérivés (`.derived.json`, usage-summary, paliers) à la prochaine séquence. Critère
    « frameCount ≈ 6 600 » de l'annexe mal calibré : 6 458 frames = fin de réplication − 6 s, comme les
    témoins sains.
  - L1 [x] (8 commits, revue adverse : 8 constats P2, corrigés ou classés) et L2 [x] (3 commits, revue :
    2 P1 + 7 P2, corrigés) — en attente du contrôle de parc indépendant (exigence du 23/09) puis de la
    fusion. L3 et L4 : l'agent d'implémentation a été interrompu par une erreur de l'API (faux positif
    du filtre de sécurité), aucun commit : relancés.
  - Sondes : P1 négatif sur le tir continu par tir, positif sur les touches et le compteur (M4b bloqué,
    décision utilisateur) ; P2 → M3.1 réécrit ; P3 positif → M3.2 précisé ; P4 confirmé → M2 précisé.
  - Workflow `wf_dd706355-a4e` (24/09) : P1-S3 POSITIF (tir continu dans la vue de contrôle, voir M4b) ;
    CA9 : `00007CA9` = arme « mains nues » (voir M3.2) ; découvertes CA9 : `E9E7FF79` =
    `forge_fusion_coil_mp` (bobine à fusion de Forge), libellés à vérifier `2AC9C2FF` (hotrod) et
    `230447B1` (proto_heatwave) ; C2 : le film dit « posé par la carte » (index de placement Forge,
    6e champ du bloc object-multiplayer-properties, `FUN_14080d524`), pas « non jouable » — 13/13 décors,
    0/100 véhicules en jeu, mais aussi des tourelles actives posées : la règle L1.3 reste (générale,
    identique sur le parc), seul son commentaire (« une seule position ») est inexact (le film réplique
    la pose avant l'origine) — à corriger au prochain lot qui touche `vehiclesLayer.ts` ; SONS : le
    lance-grenades du Falcon (`0BB6976B` → snd! 541792a4, banque falcongrenadelauncher, PAS le Gauss)
    joue le MÊME événement que le tir du Rockethog (18 médias identiques) → réutiliser
    `vehicle_shot_warthog_rocket_*` ; LMG de la Wasp reconstruite (V3E, boucle cadencée 0,100 s) ; le son
    actuel de la Wasp est bien celui des missiles (équilibre de couches de l'ancien rendu rev9). Rendus et
    page d'écoute hors dépôt : `Downloads/Halo Infinite - Sons v75/rr_2026-09-23/index.html`, en attente
    de l'oreille de l'utilisateur. Branches de recherche à fusionner à la prochaine intégration :
    `feat/rr-ghidra`, `feat/rr-c2`.
  - Workflow `wf_3add3bf4-008` terminé (24/09) : intégration A (b74c8f294, CI verte) ; M1, M4a, M5
    (vague C) et M2, M3 (vague D) livrés sur leurs branches, chacun relu par un relecteur adverse
    (M2 : 1 P0 + 4 P1 ; M3 : 4 P1 ; M4a : 3 P1 ; M1 : 1 P1 ; M5 : 1 P1) puis corrigé. Arbitrages
    TECHNIQUES du superviseur sur les questions des lots (tranchés, pas des décisions produit) :
    M3 — réparation (a) par l'ancre de signature du NEW de naissance (mesurée) plutôt que par le départ
    de la vue B, M3.1 par recalage sur l'en-tête exact + élection en repli (la règle V2 a été mesurée
    et écartée), Q18 appliquée PAR VIE (une vie sans dotation lue garde « à venir »), effet de bord sur
    la marche des états (+327 paquets non localisés) accepté et déclaré ; M5 — preuve (a0) avant (b)
    (8 documents b → a0, camp identique 8/8) ; la brèche d'équivalence « faits sans verdict du fil des
    morts » est fermée PROPREMENT en vague D (lot M8 : le verdict porté dans les faits, avec la montée
    de SchemaDesFaits de la vague D) ; M2 — place vide visible aussi avant le premier occupant, humain
    qui remplace un bot : présence lue dans le film (aucun repli), chaînage FIFO, kill-switch web daté
    retrait 2026-12-01, capacité d'équipe estimée en repli nommé (lecture de la taille d'équipe de la
    variante : découverte à instruire) ; M1 — constantes de continuité de F-1 acceptées, pertes
    collatérales déclarées (D1 grappin, D2 épisode de proximité, a349fea8, a521164d) consignées en
    découvertes. Question produit ouverte : les Falcon MOBILES de BTB (23 vies, 7 documents) sont
    masqués comme décor depuis la décision du 02/09 — les rendre visibles et pilotables quand le film
    les fait bouger ? `850902EF` (25 tirs) : identité à établir en M6 (bombe de la Banshee ?).
    Montées de schéma : chaque lot a posé 69 ; l'intégrateur réconcilie — vague C = 69, vague D = 70.
  - Découverte : `archlint/no_stale_fallback_target_test.go:59` lit `../../.ai/PLAN_DECODEUR_FILM_2026-09-13.md`,
    déplacé sous `.ai/V7.5/` par le commit d'archivage `fe2106f4b` : test ROUGE sur la base, donc sur la
    CI de `feat/v75` — corrigé à l'intégration de la campagne (bloque la CI).
- 2026-09-23 (nuit, intégration de la vague A) : `feat/rr-sondes`, `feat/rr-l3`, `feat/rr-l4`, `feat/rr-l2`,
  `feat/rr-l1` fusionnées `--no-ff` dans `feat/retours-rejeu` (`5e1e97be5`, `3748ca8e5`, `62ad5ebf7`,
  `c51b45489`, `0a608ae13`), aucun conflit ; casse de la base corrigée (`8be964717`, chemin du plan du
  décodeur archivé ; aucune autre lecture de document `.ai` déplacé). Gates sur la tête : build, vet,
  `go test ./...` (189 paquets ok, 0 FAIL), `-tags=integration -count=1 -p 1 -json ./...` (exit 0,
  17 103 pass / 786 skip / 0 fail), baseline JSONL OK (9 701 tests de la baseline tous présents),
  golangci-lint ratchet 0 issue ; web : `tsc -b`, eslint 0 erreur (26 avertissements antérieurs),
  lint:colors 0, vitest 792 fichiers / 8 510 tests ok, knip 0, manifestes i18n régénérés sans écart.
  Restent : verdicts visuels utilisateur L1 / L2, contrôle après redémarrage L4, killsource
  d'ab526724 au post-sync ; `feat/rr-ghidra` et `feat/rr-c2` non fusionnées (hors de cette intégration).
