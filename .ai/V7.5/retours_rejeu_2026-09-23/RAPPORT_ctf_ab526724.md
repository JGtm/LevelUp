# Rapport — lot `ctf_ab526724` : drapeau, score et frise absents + véhicules (CTF Starboard `ab526724`)

Enquête en lecture seule du 2026-09-23 sur `feat/v75` @ `43a01721e`. **Aucune commande `go`, aucun
worktree créé** : les faits persistés et les journaux ont suffi. Sondes Node :
`SP/sondes/ctf_ab526724/` (`sweep_ctf.mjs` → `sweep_ctf.tsv`, `manifests.mjs`, `score_track_park.mjs`,
`veh_n_dist.mjs`, `veh_static_park.mjs`, `bounds.mjs`, `cov.mjs`, `veh_samples_dt.mjs`), sorties `*.out.txt`.
Balises : **MESURÉ** (fichier:ligne, requête, sortie chiffrée) / **DÉDUIT** / **HYPOTHÈSE**.

## 0. En bref

- **Cause racine (drapeau, score, frise)** : le film a été archivé ~50 s après la fin du match, **avant
  que le serveur Halo l'ait finalisé**. Le manifeste local, qui sert de marqueur de commit du cache,
  ne liste que 34 chunks sur 37. Il manque les 2 derniers chunks de réplication (≈ 33 s de film, dont la
  3e capture) et **le chunk des temps forts** (type 3 : fil des morts + kill-feed). Les 3 chunks
  manquants sont arrivés sur le disque 11 s plus tard. Mais `filmcache.Write` ne réécrit jamais un
  manifeste existant, et la cuisson avait déjà chargé le film.
- **Conséquences** : sans fil des morts, les horloges ne se calent pas (`deathOffsetMatched` 0) et le pont
  statborg tombe à 3/8. Le triplet seul ne suffit pas sur un film tronqué. Côté drapeau : 19 prises hors
  fenêtre + 12 sans pont → 0 portage. Les 74 actions d'objectif sont toutes en `noSlot`, l'identité des
  camps est non résolue, et la 3e capture manque. Le killsource répond « sans kill-feed » **276 fois en
  13 h** : il lit le manifeste périmé, ce qui explique l'absence de lignes `kill_*` en base.
  **La publication ne dépend pas de ces lignes** : les deux symptômes ont la même cause amont, et
  « équipe = le film seul » tient.
- **Propre à ce match** : c'est le seul manifeste sur 1 625 dans ce cas. `d6918972` (CTF du même soir,
  même build, même registre) est nominal, et Starboard Team Slayer `f0220a96` aussi. **Mais la course
  est structurelle** : 23 matchs sur 96 sont détectés moins de 60 s après leur fin. Un précédent existe
  avec la même signature : `7b0d89c4`, 2026-09-02.
- **Deux défauts web génériques**, révélés par ce 0-3 :
  - le bandeau reste figé à **0 — 0** quand la seule série d'équipe n'a pas de camp (6 documents du parc) ;
  - la **piste Score n'est jamais dessinée sur un match à sens unique** (13 documents, même quand le
    camp est résolu).
- **Véhicules : du décor, pas une erreur.** Ce sont des objets de la carte Forge Starboard : 1 Scorpion,
  2 Wasp, 3 Warthog.
  - Ils sont identiques au centimètre dans les deux matchs Starboard du parc.
  - Ils se trouvent 19 à 24 m hors de l'emprise jouable.
  - Le film ne réplique leur position **qu'une fois, à la naissance**, et ils ne sont jamais occupés.
  - Deux Warthog sont à 1,8 m l'un de l'autre, donc se chevauchent : ce sont des objets non simulés.

## 1. Constat reproduit sur pièces

### 1.1 Le document `data/cache/replays/halo_infinite/ab526724.json` (MESURÉ, `cov.mjs`, `ctf.mjs`)

| Élément | ab526724 | 81c02726 (témoin sain) |
|---|---|---|
| construit | 2026-09-22 21:36:24 (+02:00), schéma 68 | 2026-09-22 14:19 |
| `coverage.bridge.deathOffsetMatched` / `RunnerUp` / `deathOffsetMs` | **0 / 0 / absent** | 39 / 7 / 440 475 |
| `coverage.bridge.livesNamed` / `indexReadings` | **0/144 / 0** | 39/45 / 17 |
| `coverage.verdict.bridge` | « partiel : la table d'index n'est confirmée par aucun second chunk » | nominal |
| `identity.filmTable` accord / silence | **0 / 8** | 8 / 0 |
| `identity.statborgSlot` | déduit 3 (`triplet_feuille` : slots 10, 16, 20 = Taiko900BPM, JGtm, Chocoboflor), **non résolus 5** | déduit 8 (`instants_de_mort` ×6 + `triplet_feuille` ×2) |
| `coverage.flagCarries` | flagFilm true, openings 31 = **noBridge 12 + outOfWindow 19**, carries 0 | (Strongholds) |
| `coverage.objectives` | available 74, **attached 0, noSlot 74** | 35 / 35 / 0 |
| `coverage.score.teamIdentity` | **unresolved** | a |
| `scoreTimeline.teams` | **une** série **sans `teamId`** : 1 à t=2021 (affiché 3:21.7), 2 à t=5184 (8:38.0) | 2 séries identifiées |
| `scoreTimeline.players` | 3 joueurs sur 8 (ceux du triplet) | 8 |
| fin du document | t=6300 → affiché **10:29.6** ; match API 670 s = **11:10** | — |

- Calques absents : `flagCarries`, `objectives`, `flagReturnZone` (ils sont vides, donc omis).
- `objectiveObjects` absent est **normal** : c'est le crâne d'Oddball
  (`apps/web/src/features/match-replay/layers/useReplayObjectiveObjects.ts:1-12`), avec 0 vie sur les
  18 CTF du parc.
- `zoneStates` absent est **normal** aussi : ce calque sert au KOTH et aux Bastions.
- **Faits persistés** (`data/cache/film_facts/halo_infinite/ab526724.filmfacts.bin`) : chaque gamertag
  n'y apparaît **qu'une fois**, dans la table des sièges. C'est 13 à 26 fois dans `81c02726` et 52 fois
  dans `d6918972` (fil des morts + kills). Les faits ne contiennent donc **ni fil des morts ni kills**
  (MESURÉ, `grep -a -o | wc -l`).

### 1.2 Le cache film (MESURÉ)

- `data/cache/film_manifests/ab526724.json`, écrit à 21:35:50.119 :
  - **34 entrées (0..33)** : chunk 0 de type 1, chunks 1..33 de type 2, **aucun type 3** ;
  - le chunk 33 commence à 640 105 ms et dure 20 010 ms, donc finit à **660,1 s de film**. C'est
    exactement la fin des positions du document (origine 30,017 s + 630,0 s).
- `data/cache/film_chunks/ab526724/` contient **37 fichiers** :
  - `chunk_00..33` écrits entre 21:35:50.093 et 21:35:50.118, avant le manifeste ;
  - **`chunk_34..36` écrits à 21:36:01.236–.238**, 11 s après ;
  - en-têtes : 34 et 35 commencent comme 33 (`01 00 00 00 eb 3b 05 00`, réplication), 36 comme le chunk
    de temps forts de `81c02726` (`09 00 …`).
- **Parc** (`manifests.out.txt`) : 1 625 manifestes (858 écrits par Go, 767 hérités). ab526724 est le
  **seul** sans chunk de type 3 et le **seul** avec des fichiers absents du manifeste.

### 1.3 Journaux (MESURÉ, `logs/sync.log`, `logs/auth.log`)

- Chronologie du 22/09 (heure locale, +02:00) :

| Heure | Événement |
|---|---|
| 21:35:00 | fin du match (19:23:50.4 UTC + 670 s = 19:35:00 UTC) |
| 21:35:36.97 | détection par `match_poller`, soit **37 s** après la fin |
| 21:35:43 → 21:35:50.38 | killsource « début » puis « sans kill-feed ». La durée de 7,2 s inclut le téléchargement ; c'est ce passage qui a archivé les 34 chunks via `RemoteFilms`. |
| 21:35:45 → 21:35:52.79 | second passage killsource, même résultat |
| 21:35:57.6 | verrou de décodage pris par le post-sync de Chocoboflor (pid 39176, décode « depuis 19:35:59Z ») |
| 21:36:24.29 | « artefact rejeu construit » (Chocoboflor, 144 pistes) |
| 21:36:01.239 | post-sync de JGtm : « artefact rejeu non construit … décodage déjà en cours ». Son `persistFilmToCache` (`films_persisted 5`) a écrit 34–36 à 21:36:01.236 (**DÉDUIT** : 3 ms d'écart ; le manifeste de l'API listait alors 37 chunks). |

- `match_kill_events_latest` pour ab526724 : 145 lignes `decoder_rev sync-kill-feed-2026-08-02`,
  `read_origin credit-seul`, écrites à 19:37:15 UTC. L'API servait donc le chunk de temps forts environ
  2 min après la fin.
- Killsource « sans kill-feed » sur ab526724 : **276 fois** entre le 22/09 21:35:50 et le 23/09 10:47:39.

### 1.4 Le parc CTF (MESURÉ, `sweep_ctf.tsv`, 18 documents)

| doc | carte | API | `flagCarries` | actions att./dispo | teamIdentity | séries d'équipe | manifeste = fichiers, type 3 |
|---|---|---|---|---|---|---|---|
| 64e8adfa | Catalyst | 2-3 | 2 | 176/176 | a | 1:3 / 0:2 | 45=45, oui |
| fb1a1a72 | Banished Narrows | 0-1 | 2 | 72/72 | **unresolved** | ?:1 | 43=43, oui |
| 879a4dba | Fortitude (BTB) | 2-3 | 0 (flagFilm false) | 0/33 (`refusedByRoster`) | a | 1:3 / 0:2 | oui |
| 4f77afc1 | Flood Gulch (BTB) | 1-1 | 0 (flagFilm false) | 0/12 (`refusedByRoster`) | **unresolved** | ?:1 / ?:1 | oui |
| bf5ced1b | Illusion | 3-0 | 2 | 22/22 | **unresolved** | ?:3 | oui |
| bc60b4d9 | Illusion | 3-0 | 2 | 73/73 | b | 0:3 | oui |
| 7fce3219 | Takamanohara | 2-3 | 2 | 124/124 | a | 1:3 / 0:2 | oui |
| 16ea3668 | Aquarius | 3-1 | 2 | 56/56 | a | 0:3 / 1:1 | oui |
| b8a44fe8 | Forest | 0-1 | 2 | 129/129 | b | 1:1 | oui |
| a0c36016 | Forest | 0-1 | 2 | 95/95 | b | 1:1 | oui |
| cde26226 | Critical Dewpoint | 2-3 | 2 | 158/158 | a | 0:2 / 1:3 | oui |
| 4ecdf3e7 | High Ground (neutre) | 0-5 | 1 | 50/50 | **unresolved** | ?:5 | oui |
| 8bc6074f | Origin | 1-3 | 2 | 99/99 | a | 0:1 / 1:3 | oui |
| 58864b3c | Domicile | 3-0 | 2 | 42/42 | **unresolved** | ?:3 | oui |
| f8efc5ca | Absolution | 3-0 | 2 | 79/79 | b | 0:3 | oui |
| 294987fe | Aquarius Ranked | 4-2 | 2 | 139/139 | a | 0:4 / 1:2 | oui |
| **ab526724** | **Starboard** | **0-3** | **0** | **0/74** | **unresolved** | **?:2** | **34≠37, NON** |
| d6918972 | Origin (22/09, 19:46 UTC) | 1-3 | 2 | 58/58 | a | 1:3 / 0:1 | 37=37, oui |

- Sur les 16 CTF d'arène, ab526724 est le seul sans drapeau ni actions.
- Les deux BTB CTF sont écartés **volontairement** : l'effectif ne tient pas dans les 8 slots du statborg
  (`matchfacts.go:320-329`). Ce n'est pas le sujet de ce lot.
- Mêmes build (`HI_1_13_0`), empreinte de registre (`0x36ca8c3d2a2f9b88`) et révisions pour ab526724,
  d6918972 et f0220a96 : **ce n'est pas le build de jeu**.

## 2. Cause racine prouvée

### 2.1 La chaîne Go, dans l'ordre

1. **Acquisition d'un film non finalisé.**
   - `RemoteFilms` (`internal/sync/killcollector/remote_films.go:82-109`) et `persistFilmToCache`
     (`internal/sync/replayartifacts/cuisson.go:343`) prennent le manifeste de l'API tel qu'il est au
     moment de la requête. Vers 21:35:43–5x, il listait 34 chunks ; à 21:36:01, il en listait 37.
   - `filmcache.Write` écrit les chunks, puis le manifeste en dernier comme marqueur de commit, et
     **« un manifeste DÉJÀ PRÉSENT n'est jamais réécrit »** (`film/filmcache/write.go:79-82`).
   - Le second passage ajoute donc les fichiers 34–36, mais le manifeste reste à 34 entrées.
2. **Cuisson sur 34 chunks** (enfant démarré à 19:35:59Z, avant l'arrivée des fichiers 34–36).
   - `ScanDeaths` prend pour chunk de temps forts **« le DERNIER numéro »**
     (`film/replay/deaths_source.go:62-64`), ici le 33, qui est un chunk de réplication. La lecture
     échoue.
   - Résultat : `film_scan.go:402-407` émet « fil des morts illisible » et pose `deaths = nil`.
   - La table d'index des chunks n'est même pas lue, faute de morts (`film_scan.go:424`). D'où
     `indexReadings 0`, le verdict « partiel » et `filmTable` silence 8.
   - **Preuve que le film chargé n'avait que 34 chunks** : le balayage des positions parcourt tous les
     numéros présents (`grammar/offline_biped_band.go:111-114`), et les positions s'arrêtent pile à la
     fin du chunk 33.
3. **Pas de calage d'horloge.** `bestDeathOffset` rend (0, 0, 0) quand le fil est vide
   (`film/replay/lives_death_offset.go:87-88`), donc `DeathOffsetMS` vaut 0.
4. **Drapeau.**
   - Le pont par manche combine les morts (vide) et le triplet de la feuille
     (`replaybuild/matchfacts.go:393-399`). Le triplet exige l'égalité exacte avec les totaux de l'API
     (`facts/objectives/slotidentity.go:89`). Or le film s'arrête environ 40 s de match avant la fin :
     seuls 3 slots sur 8 concordent (**DÉDUIT** : 5 joueurs ont bougé un compteur dans ces 40 s).
   - Les 12 prises des slots non nommés comptent en `NoBridge` (`flag_carries.go:247`).
   - Les 19 prises nommées sont placées sur l'axe par `frameOfMatchMS` avec un calage de 0. L'instant
     du match tombe alors avant l'origine du moteur, donc à −1, et compte en `OutOfWindow`
     (`match_clock.go:34-43`, `flag_carries.go:402-405`).
   - Résultat : 0 portage, donc ni `flagCarries` ni `flagReturnZone` (`build_objectives_live.go:234-236`).
5. **Actions d'objectif.** `identifiedEvents` : `deaths.err != nil` met toute la famille en `noSlot`
   (`replaybuild/matchfacts.go:331-334`), soit 74/74. Si le fil avait seulement été vide, les 3 slots du
   triplet auraient rattaché leurs actions ; `attached = 0` prouve la branche « illisible ».
6. **Score.**
   - La 3e capture est dans les chunks 34–35, absents : la série finit à 2.
   - La preuve (a), le score final, exige **les deux** slots d'équipe (`score_team_identity.go:59-63`)
     et l'égalité (2 ≠ 3).
   - La preuve (b), les frags, exige le triplet des deux camps.
   - Les deux échouent : `unresolved`, et la série est publiée sans `teamId`.
7. **Killsource.**
   - `LocalCacheFilms` sert **les entrées du manifeste** (`sync/killcollector/cache_films.go:57-92`),
     toujours le disque d'abord (`remote_films.go:88-97`).
   - `loadKillFeed` ne trouve aucun chunk lisible comme kill-feed (`facts/killsource/feed.go:91-111`) :
     `ErrNoKillFeed`, rien n'est écrit, et c'est retenté à chaque cycle, indéfiniment.

**Dépendance à la base killsource : NON.**
- `port.MatchFacts` ne contient que les lignes de match, les scores, la variante et la carte
  (`internal/domain/match_facts.go:60-81`).
- Les équipes des joueurs sont celles du film (`coverage.teams` : film 8, accord 8).
- Rejeu et base partagent la même cause amont : le manifeste sans chunk de temps forts.

### 2.2 La chaîne web

- **Drapeau** : `useReplayFlagCarries.ts:130` ne dessine que `doc.flagCarries`. L'état « au socle »
  (`home`) fait partie de ce calque : un calque vide, c'est aucun drapeau nulle part.
- **Bandeau** : `readScoreBanner` passe la garde, puisqu'il y a une série (`scoreBannerLogic.ts:139`).
  Les deux camps viennent du tableau de score, et `teamSeriesFor` cherche `t.teamId === camp`
  (`lib/replay/scoreTimeline.ts:261`). La série sans `teamId` n'est trouvée pour aucun camp :
  **0 — 0 affiché tout le match**, une fausse mesure. Même défaut sur fb1a1a72, bf5ced1b, 58864b3c,
  4ecdf3e7, c259789d et sur 4f77afc1 (deux séries sans camp).
- **Frise** : `scoreTrack` (`hooks/useReplayTimeline.ts:282`) appelle `leaderStates`, qui exige
  **au moins 2 séries avec `teamId`** (`scoreTimeline.ts:405-406`). Résultat vide, donc pas de piste Score.
  Même cause sur les **13** documents du parc à une seule série, tous des matchs à sens unique
  (`score_track_park.out.txt`), dont 7 ont pourtant un camp résolu (bc60b4d9, f8efc5ca, b8a44fe8,
  a0c36016, 606d9844, 8076f97f, ee90570b). La règle du module — « une équipe sans série vaut zéro »,
  mesurée sur le témoin 530820e5 — est appliquée au bandeau, mais pas à la piste.

### 2.3 Propre au match, au build récent, ou à Starboard ?

- **Pas le build** : d6918972 (même soir, même build, même registre) a 37 entrées pour 37 fichiers,
  `deathOffsetMatched` 129 et 25 portages.
- **Pas Starboard** : sur f0220a96 (Team Slayer Starboard), le pont est nominal (`deathOffsetMatched` 88).
- **Une course au téléchargement.**
  - Délai de détection après la fin du match, mesuré sur 96 matchs de `logs/auth.log` : 23 matchs
    sous 60 s, dont `7b0d89c4` (+22 s) et `ab526724` (+37 s) ; 72 au-delà de 30 min.
  - Le serveur finalise le film environ 1 min après la fin (**DÉDUIT** de 21:35:5x → 21:36:01).
  - Deux manifestes partiels sont connus.

## 3. Historique — ce qui a été fait, et pourquoi ça n'a pas suffi

- **`7b0d89c4`, même signature, déjà vue.**
  - `.ai/V7.5/PLAN_CUISSON_PERF.md:877-888` (N-Y, 2026-09-02) : « `chunk_31.bin` et `chunk_32.bin` sur
    disque sans entrée au manifeste (seul cas sur 1 380) ». Il manquait le dernier chunk de réplication
    et le chunk de temps forts.
  - Le problème a été traité comme un risque de **datation** : `manifestChunks` et `chunksDuManifeste`
    ignorent les chunks non décrits. La cause n'a pas été cherchée.
  - La restauration du cache du 16/09 a remplacé ce manifeste par un manifeste complet (33 entrées,
    type 3 en 32). Le défaut a disparu sans être compris.
  - `facts/objectives/film.go:80-81` et `replaybuild/matchfacts.go:155-156` le citent encore au présent.
- **Schéma 42 (2026-09-06)** : le pont du drapeau est complété par le triplet (cas `c0a82e88`). Inopérant
  ici, parce que le triplet porte sur un film tronqué.
- **Schéma 48, lot « pont muet » (2026-09-07)** : le calage par vote. Hors sujet ici, le fil des morts
  étant vide.
- **Web** : la piste Score (2026-08-28, garde D1 du 2026-09-02) exige deux séries identifiées, alors que
  la règle du 3-0 était déjà écrite en tête de `scoreTimeline.ts`.
- **Véhicules.**
  - Décor par **famille** (Falcon, Pelican, Phantom, Skiff, 2026-09-02) et `map_element` pour les
    tourelles bannies (2026-09-14) : aucun des deux ne couvre un décor de famille pilotable.
  - `.ai/V7.5/film_re/CADRAGE_VEHICULES_2026-08-31.md:737` avait noté « ti=40 sur des cartes sans
    véhicule (… Starboard …) : entités statiques », sans traitement.

## 4. Solutions proposées

### 4.A Le match ab526724 et la course d'acquisition

**Option A1 — réparation des données seule.** Aucun code. **Décision de l'utilisateur, serveur arrêté,
prévenir avant.**
1. Renommer `data/cache/film_manifests/ab526724.json` en `ab526724.json.partiel-34` (on garde la pièce).
2. `levelup archive-films --gamertag JGtm --match ab526724`, avec `--dry-run` d'abord.
   - La présence d'un film se juge sur son manifeste (`cmd/levelup/cmd_archive_films.go:291`).
   - Les fichiers déjà présents ne sont pas réécrits (`write.go:71-73`) ; le manifeste complet est écrit.
   - Il faut que le film soit encore servi : c'est le cas aujourd'hui, mais il expirera.
3. Renommer `data/cache/film_facts/halo_infinite/ab526724.filmfacts.bin`. **Sans cela, la recuisson
   rejoue les faits du film tronqué** : `Utilisable` (`film/replay/filmfacts_fichier.go:323-335`) ne
   compare que les révisions et la clé de carte, pas l'inventaire des chunks.
4. `levelup backfill-replay --one ab526724-3684-4335-b759-a18edcccc137`.
   - C'est la forme dite « interne », mais elle est autonome : verrou solo, faits de match lus en base,
     noms de carte résolus (`cmd_backfill_replay_child.go:80-107`).
   - **Jamais `backfill-replay` sans `--one`** : la passe cuirait tous les films sans artefact.
5. Killsource : le post-sync le retente à chaque cycle et réussira une fois le manifeste réparé ; sinon
   `backfill-killsource` (passe films). Ensuite, les dérivés (résumé d'usage, paliers de socles) selon la
   séquence habituelle.

- Montée de schéma : non. Republication : ab526724 seul, par redécodage. Taille : S.
- Risque : la course se reproduira sur un autre match détecté tôt.

**Option A2 — A1, plus le code « un film n'est archivé, décodé ni cuit que finalisé ».** **Recommandée.**

Le principe vient de la grammaire du film : un film est finalisé quand son manifeste porte son chunk de
temps forts (`chunk_type 3`, `haloclient.FilmChunkTypeHighlightEvents`). Ce chunk est, par définition,
le dernier du film (`deaths_source.go:62`). Mesure du parc : 1 624 manifestes sur 1 625 le portent ;
l'exception est ab526724.

Fichiers touchés :
- `internal/games/halo_infinite/film/filmcache/write.go` :
  - ne pas valider (commit) un manifeste sans type 3 : erreur typée `ErrFilmNonFinalise` ;
  - **remplacer** un manifeste existant **sans type 3** par une liste qui le complète (sur-ensemble exact
    par index). C'est la seule réécriture permise ;
  - un prédicat unique, `Finalise(chunks)`.
- `internal/sync/haloclient/halo_client_film.go` (`fetchFilmChunks`) : un manifeste de l'API sans type 3
  renvoie `ErrFilmNonFinalise`. Ce n'est ni un 404 ni une panne.
- `internal/sync/replayartifacts/cuisson.go` (`persistFilmToCache`, l.343) : reporter la cuisson
  (compteur + journal INFO) ; le cycle suivant retente.
- `internal/sync/killcollector/cache_films.go:57` : un manifeste local sans type 3 compte comme absent.
  `RemoteFilms` repasse alors par le réseau et complète le manifeste : **ab526724 se répare seul côté
  killsource**.
- `internal/replaybuild/filmfacts_cuisson.go` (chargement du film) : refuser de cuire un film sans type 3,
  ou dont des fichiers sont absents du manifeste (la signature d'ab526724). L'erreur typée est comptée en
  « écartés » : jamais d'artefact dégradé silencieux.
- `internal/games/halo_infinite/film/replay/deaths_source.go` : choisir le chunk de temps forts **par son
  type** quand le manifeste est là, et non plus comme « le dernier numéro ».

Tests (rouges avant, verts après) :
- `Write` d'une liste sans type 3 : le commit est refusé ;
- `Write` qui complète un manifeste partiel ;
- `LocalCacheFilms` sur un manifeste partiel renvoie `found=false` ;
- fetch d'un manifeste API partiel : erreur typée ;
- cuisson d'un répertoire « 34 chunks au manifeste + 3 fichiers hors manifeste » : refusée ;
- `ScanDeaths` sur un film dont le dernier chunk est de type 2 : erreur typée distincte.

Garde-rail : un ratchet `archlint` interdit toute comparaison au type 3 hors du prédicat (règle des
≤ 2 copies).

Bilan :
- Montée de schéma : **non**. Republication : ab526724 seul (étapes A1). Taille : **M**.
- Risques :
  - un film légitimement sans chunk de temps forts : 0 mesuré sur 1 625 manifestes ;
  - un match très frais n'aura son artefact qu'au cycle suivant.

### 4.B Score et frise des matchs à sens unique

Ce défaut est générique ; il ne se répare pas en republiant ab526724.

**Option B1 — web seul.** Taille S, sans schéma.
- `features/match-replay/model/scoreBannerLogic.ts:139` : renvoyer `null` dès qu'une série publiée n'a
  pas de `teamId`. Plus de 0 — 0 non mesuré (7 documents concernés).
- `lib/replay/scoreTimeline.ts:402-415` (`leaderStates`) et `hooks/useReplayTimeline.ts:275-285` : quand
  **toutes** les séries publiées ont un camp, compléter les camps absents par 0. Les camps sont ceux du
  document (`roster[].team`, lus dans le film) : c'est la règle déjà écrite en tête du module.
  `leadChanges` ne change pas (un 3-0 n'a pas de retournement ; le test existant reste vert).
- Tests vitest :
  - 3-0 avec un camp identifié : piste « égalité, puis le meneur » (rouge aujourd'hui : `[]`) ;
  - série sans camp : bandeau `null` (rouge aujourd'hui : 0 — 0).

**Option B2 — B1, plus la preuve (a′) côté Go.** **Recommandée.**
- `film/replay/score_team_identity.go:55-71` : si un seul slot d'équipe porte une série de score, le score
  absent vaut 0 (la même règle mesurée).
- Si le registre dit X-0 (X > 0) et que la série finit **exactement** à X, cette série est le camp X.
- Effet : 5 documents sains sont résolus (fb1a1a72, bf5ced1b, 58864b3c, 4ecdf3e7, c259789d).
- ab526724 reste `unresolved` tant qu'il est tronqué (2 ≠ 3) : le garde-fou tient.
- Test Go : rouge sur une fixture 3-0 qui a la forme de bf5ced1b, et série à 2 contre un registre à 3
  (doit rester `unresolved`).
- Montée de schéma : **oui**, puisque la règle de publication du calque score change. Republication
  **depuis les faits**, sans redécodage (verdict `republier`). **Décision de l'utilisateur.** Taille S.

### 4.C Véhicules : décor ou erreur ?

**Réponse : du décor de carte, pas une erreur de décodage.**
- **Même objet dans deux matchs** : les 6 vies (771 à 778, mêmes châssis) sont aux mêmes positions au
  centimètre dans ab526724 (CTF:Arena) et dans f0220a96 (Team Slayer:Arena, 2026-09-01). C'est donc un
  placement fixe de la carte.
- **Carte Forge** : Starboard est une carte Forge sur le canevas « espace » (`fo03_space`,
  `internal/himap/cartes_forge.go:76-82`). Son `.mvar` pose banshee, scorpion, warthog,
  warthog_razorback et wasp (`.ai/V7.5/ETAT_VEHICULES_2026-08-31.md:14,79`). Les deux variantes, CTF et
  Team Slayer, les laissent exister.
- **Hors de l'emprise jouable** :
  - joueurs : y ∈ [−110,5 ; −76,5], sur 27 305 échantillons des deux matchs (rasters `ab526724.json` et
    `f0220a96.json`) ;
  - fond de carte : y ≥ −117,8 (calibration
    `data/titles/halo_infinite/reference/map_backgrounds/7a9265af-….json`) ;
  - véhicules : y ∈ [−133,8 ; −129,7], soit **19,4 à 23,5 m** au sud de l'arène, hors du fond dessiné ;
  - socles de drapeau : (−8,55 ; −93,17) et (16,3 ; −93,17), bien dans l'arène.
- **Jamais simulés** : une seule position, relevée à t=0 (affiché 0:00), pour une vie qui court jusqu'à la
  fin du film (`end film_end`), sans aucun occupant. Deux Warthog sont à 1,8 m l'un de l'autre, les
  2 Wasp à 3,3 m : ils se chevaucheraient physiquement (**DÉDUIT** de leur gabarit).
- **Pas d'emprise de jeu Starboard au dépôt pour trancher « hangar visible ou caché »** :
  `data/cache/mvar/` n'a pas d'entrée Starboard, et les bornes de quantification sont celles du canevas
  (±231 m). Le « hangar visible derrière une vitre » reste une **HYPOTHÈSE** (question Q3).
- Les vies 769/770 (famille inconnue, positions de limbes) relèvent d'un autre enquêteur.

**Option C1 — règle d'affichage lue dans le film.** **Recommandée.** Taille S, sans schéma.
- Dans `apps/web/src/features/match-replay/model/vehiclesLayer.ts` (à côté de `FAMILLES_NON_JOUABLES`,
  l.97), une vie devient un **objet de décor** quand le film ne réplique sa position **qu'une fois, à la
  naissance** (échantillon unique à t = t0), pour une vie qui court jusqu'à la fin (`end = film_end`),
  **sans aucun occupant**.
- Ce n'est pas un seuil : c'est ce que le film écrit.
- Mesure sur le parc (`veh_n_dist.out.txt`, 278 vies de familles pilotables) :
  - **13 vies** tombent dans ce cas : 12 à Starboard, et 1 à Goliath (`d8b13ec2` slot 768 : un Wasp 3 m
    **sous** le sol de jeu, sur une carte Forge dont le `.mvar` pose un Wasp) ;
  - **0 des 232 vies en jeu** (au moins 2 positions), dont 90 garées jamais occupées : Refuge `0301037e`
    en a 52 à 76 positions chacune ;
  - la vie 770 (une seule position, mais à t=3601) n'est pas capturée.
- Rendu : masqué comme les Falcon, ou « élément de carte » neutre comme les tourelles bannies
  (**décision de l'utilisateur**, Q2).
- Aussi exclu du prédicat « embarqué ».
- Tests vitest :
  - fixtures ab526724 771–778 et d8b13ec2 768 : rouges aujourd'hui (dessinées) ;
  - négatifs : `0301037e` 779 et les tourelles de `bfecd02b`, qui restent inchangées.

**Option C2 — confirmation par la grammaire.**
- Une sonde de recherche sur 2 films 4v4 non BTB (f0220a96 Starboard, 0301037e Refuge).
- Elle compare, entre les vies de décor et les vies garées en jeu, les composants de l'archétype ti=40 et
  les bits de leur record de création. Piste : `i34 vehicle-type-physics`, « composant conditionnel non
  expliqué » selon `CADRAGE_VEHICULES` §6.5.
- Coût : 2 décodages de films, environ 1 à 2 min chacun, RAM modérée, sous le verrou.
- Si un champ tranche, le Go le publie par vie (montée de schéma et republication) et C1 en devient la
  lecture.

**Repli écarté.** La distance à l'emprise n'est pas recommandée : le Wasp de Goliath est dans l'emprise
en XY, et les tourelles bannies, qu'on doit dessiner, sont jusqu'à 4,9 m hors emprise.

### 4.D Ordre et gates

1. **A1** (données, après accord). Gate document ab526724 après recuisson :
   - manifeste à 37 entrées, dont un type 3 (`manifests.mjs` : 0 suspect) ;
   - `coverage.bridge.deathOffsetMatched > 0` et `deathOffsetMs` présent ;
   - `identity.statborgSlot.non_resolu = 0` ;
   - `coverage.flagCarries` : `noBridge 0`, `outOfWindow 0`, `carries > 0` ;
   - 2 drapeaux dans `flagCarries`, `flagReturnZone` présent ;
   - `objectives.attached = available` ;
   - série d'équipe finale à 3 avec `teamId` 1 ;
   - `frameCount` ≈ 6 600 ou plus (fin affichée vers 11:0x) ;
   - verdict du pont `nominal` ;
   - en base : lignes `kill_positions_latest`, `match_death_context_latest` et `kill_openings_latest`
     pour le match.

   Script : `node SP/sondes/ctf_ab526724/sweep_ctf.mjs ab526724`.
2. **A2** (code). Gate :
   - 0 manifeste sans type 3 dans le parc ;
   - compteur « films non finalisés reportés » en journal ;
   - aucun artefact cuit depuis un film non finalisé.
3. **B1** (web). Gate visuel :
   - bc60b4d9 (3-0) : piste Score présente ;
   - fb1a1a72 : bandeau muet ;
   - ab526724 réparé : bandeau et piste présents.
4. **B2** (Go + montée de schéma, après accord). Gate : `teamIdentity unresolved` sur les documents à une
   seule série passe de 6 à 0 (ab526724 réparé).
5. **C1** (web), puis **C2** (sonde, après accord). Gate : 13 vies de décor, 0 véhicule en jeu masqué.

**Gate parc** (16 CTF d'arène, avant → après A1 + B1 + B2) :

| Critère | Avant | Après |
|---|---|---|
| drapeau publié | 15 | 16 |
| actions complètes | 15 | 16 |
| piste Score possible | 7 | 16 |
| bandeau faux à 0 — 0 | 5 | 0 |

## 5. Questions pour l'utilisateur (une ligne chacune)

- **Q1.** Réparer ab526724 maintenant, serveur arrêté : manifeste renommé, `archive-films --match`, faits
  renommés, `backfill-replay --one`, puis killsource ? (oui / non)
- **Q2.** Les véhicules de décor (Starboard, Goliath) : les masquer comme les Falcon, ou les dessiner comme
  « élément de carte » neutre comme les tourelles bannies ?
- **Q3.** En jeu sur Starboard, voit-on 6 véhicules garés (1 Scorpion, 2 Wasp, 3 Warthog) à environ 20 m au
  sud de l'arène ? (oui / non / inaccessibles)
- **Q4.** Sur un match à sens unique (3-0), afficher la piste Score : égalité jusqu'à la 1re capture, puis
  le camp qui marque en tête ? (oui / non)
- **Q5.** B2 impose une montée de schéma, avec republication depuis les faits (sans redécodage) : d'accord,
  et groupée avec la prochaine montée ?

## 6. Hors périmètre découvert (noté, non traité)

1. **Boucle infinie de « sans kill-feed »** : le post-sync retente sans marqueur terminal. ab526724 a été
   retenté 276 fois en 13 h ; 7206e05b 764 fois, d3fe5a96 717, 29206c7c 712, d672402a 702, 91b692a8 678
   (`logs/sync.log`, `logs/general.log`).
2. **Rejeu depuis les faits ≢ rejeu depuis le film quand le fil des morts est illisible.**
   `replaybuild/filmfacts_cuisson.go:96` reconstruit `filmDeaths{list: f.Facts.Deaths}` sans l'erreur.
   Republié depuis ses faits, ab526724 prendrait donc la branche « fil vide » au lieu de « illisible » :
   `coverage.objectives` différent. C'est une brèche de l'équivalence S8.
3. **Deux règles pour trouver le même chunk** : `ScanDeaths` le prend par position (le dernier numéro),
   le killsource par contenu (`feed.go:91`).
4. **Message périmé** : `film_scan.go:404`, « aucun tir ni lancer ne sera publié », est faux depuis la
   table des sièges du film (2 838 tirs publiés sur ab526724).
5. **Commentaires qui citent un état disparu** : `facts/objectives/film.go:80-81` et
   `replaybuild/matchfacts.go:155-156` présentent `7b0d89c4` comme ayant des chunks hors manifeste. C'est
   faux depuis la restauration du 16/09 (« doc inversée »).
6. **Le document ne dit pas « fil des morts illisible »**. La seule trace est le verdict « partiel : table
   d'index », et le diagnostic a exigé journaux et code. Un champ de couverture dédié (montée de schéma)
   le rendrait lisible.
7. **Écritures LUSR v2 en échec répété sur ab526724** : « Duplicate key id: N violates primary key » sur
   `match_skill_rank`, à 21:37, 21:48, 22:00, 22:09, 22:18… (`logs/sync.log`). Même famille que l'index
   désynchronisé noté en prod le 23/09.
8. **BTB CTF** (879a4dba, 4f77afc1) : pas de drapeau ni d'actions, par la garde d'effectif du statborg
   (`refusedByRoster`). C'est voulu et documenté ; à confirmer côté produit.
