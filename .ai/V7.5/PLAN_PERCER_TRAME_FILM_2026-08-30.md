# PLAN — Percer la trame du film (RE Ghidra + mesure)

> Ouvert le 2026-08-30, issu du chantier « visée à la lunette ». **Hors périmètre de la lunette.**
> Tâche destinée à une session Claude LOCALE : elle exige Ghidra (greffon HTTP 127.0.0.1:8089) et
> le corpus de films (`data/cache/film_chunks/`, 1369 films, hors git). Une routine cloud NE PEUT
> PAS l'exécuter.
>
> Miroir de suivi côté utilisateur (Notion) :
> <https://app.notion.com/p/3cc7ae87e7a3819290c5d3ae8a130ddd>
> Contexte de la découverte : page « Format du film Theater — le bit de continuation »,
> <https://app.notion.com/p/3cc7ae87e7a381ddb727c42eae16293d>

## Pourquoi maintenant

Trois verrous ont sauté le 2026-08-30 (détails : `.ai/thought_log.md`, entrées « Visee lunette »
phases 8 à 10, et `.ai/V7.5/film_re/NOTE_ENVELOPPE_EVENTS_*`, `NOTE_EMETTEUR_114_*`,
`GRAMMAIRE_EVENTS_FILM_*`, `NOTE_CARTE_CHUNK00_*`) :

1. Le premier octet d'un paquet delta **n'est pas un numéro de type d'événement** (160 paquets
   d'un seul octet le réfutent ; un en-tête d'événement exige au moins 11 bits). C'est une amorce
   suivie de l'identifiant du premier enregistrement.
2. Le film **déclare** une table de 123 entrées, une par type, dont le cardinal suit le build
   (119 → 123 de `HI_1_5_1` à `HI_1_13_0`). Artefact : `.ai/V7.5/film_re/chunk00_table_par_type.tsv`.
3. La chaîne Ghidra est rodée : table domaine → plage du var-int (`0x1451f98d0`), largeurs
   calculables statiquement, conventions des descripteurs (`+0x08` nom, `+0x58` domaine de la
   référence i, `+0x60` écrivain, `+0x68` lecteur, `+0x78` applicateur).

Mesure et binaire peuvent enfin se recouper champ par champ.

## Méthode imposée (non négociable)

- Seuils et témoins **écrits avant** chaque mesure (`.ai/V7.5/film_re/METHODE_RETRO_INGENIERIE_FILM.md`).
- Deux chaînes indépendantes valent une preuve ; une seule vaut une hypothèse — **et l'on cherche
  activement ce qui réfuterait**. Leçon payée le 30/08 : deux chaînes concordantes (désassemblage
  + régularité sur 41 M de paquets) se sont révélées compatibles avec une explication plus simple
  que personne n'avait cherchée. L'épreuve réfutante coûtait trois minutes.
- Faux positifs **mesurés sur le flux réel**, jamais calculés.
- Instruments sous garde d'environnement, sautés en CI, aucun effet sur la production.
- Fichiers ≤ 500 lignes, fonctions ≤ 80 lignes, `gofmt`, `go vet` propre.

## Lot 3 — Trancher le compte du registre (À FAIRE EN PREMIER, rapide)

**Le conflit.** Le dossier tient le registre de réplication pour **118 blocs d'archétype** ; la
relecture de `chunk_00` (lot D, 2026-08-30) en trouve **50**, et affirme que le 118 venait d'une
division de la taille du fichier par celle d'un bloc.

**Pourquoi c'est bloquant.** Tout l'inventaire des composants (325 noms, 1067 couples) en dépend,
donc toute affirmation du type « aucun composant ne porte X ». Le chantier lunette s'est appuyé
dessus plusieurs fois.

**Gate.** Le garde-rail G2 (`filmdec/ecs_table_guard_test.go`, variable `ECS_TABLE_FILM`) compare
déjà `testdata/ecs_table.tsv` au registre lu dans les films : il est VERT aujourd'hui. Soit il
valide 118 et le lot D se trompe, soit il ne vérifie pas ce qu'on croit. Commencer par là.

**Livrable.** Un verdict chiffré, et si le compte change : `ecs_table.tsv` corrigé, la doctrine
mise à jour, et la liste des conclusions antérieures à revérifier.

## Lot 1 — Les paquets écartés depuis toujours (plus forte valeur produit)

**Le gisement.** `528 262` paquets sur 1367 films sont ignorés par le décodeur, classés
« variante courte sans arme » (`fire_events.go`). Ce n'est pas une variante : la mesure donne
7 identifiants d'entité distincts d'un côté contre 50 de l'autre
(`replay/visee_octet0_research_test.go`, garde `OCTET0_FILM`). Dix fois le volume de tous les
kills du corpus.

**Ce qu'on vise.** Le décodeur avoue deux limites dures, toutes deux dues à l'incapacité de
traverser des boucles de longueur variable :
- la direction de visée n'est exposée que sur **19 %** des enregistrements ;
- **la victime n'est pas décodée du tout** (elle vit dans la liste des cibles).

**Garde-fou non négociable.** `killsource` fonctionne (59 kills vérifiés sur 59 en Theater). Tout
nouveau décodeur se valide **contre l'ancien** avant de le remplacer, jamais l'inverse. Le golden
existe (`replay/golden_*`, `killsource/golden_*`).

**Angle Ghidra.** Le lecteur d'un type est en `+0x68` de son descripteur ; les largeurs des
références se calculent depuis la table des domaines. C'est ce qui permet d'AVANCER dans un
enregistrement au lieu de deviner des décalages.

## Lot 2 — Les 538 ko jamais regardés (exploratoire)

`chunk_00` porte **trois** sections ; le dépôt n'en lisait qu'une. La troisième
(`0x0CB65C..0x14EB73`, ~538 ko, propre au match) porte les gamertags en UTF-16LE. Ni flux de
paquets, ni zlib imbriqué.

**Hypothèses à tester** : composition des équipes, réglages de partie, variante de mode, table des
entités du match, et surtout **correspondance identifiant ↔ joueur** — le pont slot → xuid est
aujourd'hui reconstruit indirectement par le fil des morts (`replay/owners.go`, `lives.go`), avec
des fragments de vie anonymes en début de match. Une table explicite vaudrait cher.

Carte de départ : `.ai/V7.5/film_re/NOTE_CARTE_CHUNK00_2026-08-30.md`.

## Ce que ce plan NE fait pas

Il ne rouvre pas la question de la lunette : elle a son propre fil (phases 1 à 10 du thought_log)
et six canaux y sont déjà mesurés négatifs. Si un lot ci-dessous fait tomber une brique dont la
lunette dépendait — typiquement le lot 3 si l'inventaire des composants était incomplet — le
signaler au thought_log et rouvrir explicitement, jamais en passant.

## Ordre et clôture

Lot 3, puis lot 1, puis lot 2. Chaque lot se clôt par : entrée `.ai/thought_log.md` (décision,
résultats chiffrés, prochaine étape), mise à jour du registre
(`.ai/V7.5/REGISTRE_REPORTS.md`) si quelque chose est reporté avec sa condition de reprise, et
commit sur une branche dédiée.

## Journal d'exécution

- **[2026-08-30] Lot 3 : CLOS `[x]`** — branche `wt/trame-film`, worktree dédié
  `LevelUp-wt-trame-film` (le worktree `wt-visee` était OCCUPÉ par une autre session en cours
  de travail — restitué intact). Verdict : le registre fait **50 blocs** (1 067 slots) sur les
  builds `HI_1_12/13`, **49 blocs** avant ; le « 118 » était `len(fichier)/taille_bloc`.
  G2 n'arbitrait rien : il tombait ROUGE sur `00162144` (slot fantôme bloc 71) dès qu'on le
  lui donnait. Correctif `parseRegistry` (fin structurelle), critères C1/C2'/C3 tenus à 100 %
  sur 1 367 films, C2 (prévalence < 1 %) RÉFUTÉ et publié tel quel (397 films pollués, 29 %).
  `ecs_table.tsv` : INCHANGÉ (déjà juste, ti ≤ 49). Goldens killsource : sortie publiée
  inchangée (98 kills, accord 85/0, ancres, contrôle négatif), 4 compteurs de diagnostic ±1,
  régénérés. Détail : `film_re/NOTE_COMPTE_REGISTRE_2026-08-30.md`.
- **[2026-08-30] Lot 1 : OUVERT `[ ]`, trois acquis mesurés** (même branche/worktree,
  instruments `lot1_familles_trame_research_test.go`, témoin `000d5950`) :
  (1) confirmation positive du modèle de trame — les familles se décodent au décodeur de
  production, L1-C1 TENU pour 0xD2, 0xD3 à 2 pts sous le seuil, publié ; les trames 0xD2/0xD3
  commencent TOUTES par un record **DEL d'une entité transitoire** (projectile probable), et
  0xD3 rend 47 slots distincts (recoupe les « 50 » du lot D sans étape commune) ;
  (2) les familles « vides » (0xC0…) portent leurs records dans les **vues 2/3** (prédiction
  tenue sur 0xC0 : 100 % ≥ 2 vues) — réserve : l'inférence multi-vues lit au-delà du payload ;
  (3) **l'amorce varie par famille** : k=2 pour 0xA0 (témoin), k=6 pour 0xC2 (99,3 %), k=8
  pour 0xD2 (86,2 %, NET) — un en-tête supplémentaire propre aux familles à bit 2 = 1.
  RESTE (ordre écrit dans la note) : sémantique de l'en-tête (piste Ghidra : 2e grammaire du
  record-loop, global `DAT_14474cd78`, largeur d'id dépendant du bit de configuration),
  re-balayage à plus de chunks pour les NON CONCLUANT, puis décodage des records suivants et
  confrontation au golden killsource. Détail : `film_re/NOTE_FAMILLES_TRAME_2026-08-30.md`.
- **[2026-08-30, suite] Lot 1 : cadrage ÉTABLI sur 2 films pour 0xC2 (k=6) et 0xD2 (k=8)** ;
  **RÉTRACTATION** du « record DEL de tête » (artefact du cadrage k=2 — sous le bon cadrage,
  le 1er record est un DELTA sur un transitoire non lié). Deux négatifs publiés : le cadrage
  par paquet est du bruit (k divergents à en-tête identique, 0xE9 : 0/255), l'inférence de
  chaîne NON CONCLUANTE par défaut d'instrument (trames vides gagnent, lecture au-delà du
  payload). Ghidra : lecture normale = branche B du record-loop (restauration force
  temporairement la branche A) ; largeur d'id runtime = table sélectionnée par le bit 0 du
  paquet — source du `IDLowBits` calibré. 0xD3 : k=6 reproductible mais faible (~40 %) —
  en-tête plus complexe, OUVERT. Table de synthèse publiée dans le miroir Notion.
