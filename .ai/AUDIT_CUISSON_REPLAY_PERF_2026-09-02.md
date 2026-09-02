# Audit performance — cuisson des artefacts de rejeu 2D — 2026-09-02

Objectif utilisateur : ramener la cuisson d'un film de ~3 min 30 (pire cas observe) vers
60-100 s, sachant qu'un bump de schema arrive (vehicules) et re-cuira le cache.

## Cadrage

Perimetre : `internal/replaybuild/`, `internal/analysis/replay/`, `internal/analysis/filmdec/`,
`internal/analysis/objectiveevents/`, `internal/games/halo_infinite/film/{killsource,filmcache}/`,
`internal/sync/replayartifacts/`, `internal/replaychild/`, `internal/filmproc/`,
`cmd/replay-build`, `cmd/replay-worker`, `cmd/levelup` (backfill-replay).

Axe : performance (temps et pression memoire d'une cuisson, debit d'une passe de masse).
Axe hors catalogue du skill `adversarial-audit`, cadre adapte : 4 auditeurs frais en
parallele + 3 verificateurs adverses (consigne « refuter ») sur les constats porteurs.

Doctrine de reference (NON remise en cause par cet audit) : protections memoire deliberees
apres 4 sinistres RAM (`internal/filmproc` : plafond dur, priorite basse, enfant borne) ;
« le VPS web ne decode jamais » ; verrou process `filmdec` (exclusion d'etat global mutable).
Dette assumee : bombe RAM `NamedEventsFrom`/`incrementTimes` (~26 Gio, registre 2026-08-24/25).

## Mesures de reference etablies

- Temoins CLI (2026-08-19, schema 18) : 208-284 s par film, **decodage pur** — la part
  compilation `go run` + demarrage est <= 1,2 % (bornee par 1er/dernier horodatage slog vs
  total watchdog). ATTENTION : HEAD est au schema 34 et fait PLUS de balayages que ces
  temoins (held_weapon, pickups, inventaire x2, equipment_changes, zoom absents des logs) —
  les 3 min 30 d'aujourd'hui sont coherentes avec ces 278 s d'hier plus les ajouts.
- Decomposition du temoin KOTH 278 s (`lotCter/cuisson_cli_01e1f945.log`, horodatages slog) :
  +84 s entre « actions d'objectif » et « capacites i48 » (fenetre = killsource + positions +
  tirs + loadouts + capacites), +94 s entre « ti=13 » et « pont slot->joueur » (fenetre =
  grenades + projectiles + morts + indices joueur, cinq balayages muets en cas de succes),
  ~98 s pour camo/grappin/poses/socles x2/zones ; assemblage + serialisation + ecriture < 1 s.
- Profil CPU du 2026-08-01 (`PLAN_BRANCHEMENT_KILLSOURCE.md:845-863`) : 78 % du decodage
  dans `filmdec.ReadBits` ; un correctif d'alignement avait deja divise certains films par 20.
- Corpus local (mesure 2026-09-02) : 1 380 films, 24,2 Mo/film compresses en moyenne
  (30,1 chunks), max 88 Mo ; artefacts ~1,84 Mo (max 6,63 Mo). Pics RSS films sains :
  48-256 Mio.

## Constats retenus

### [P1] C1 — ~36-40 relectures + decompressions zlib completes du film par artefact
- Ou : `filmdec.ReadFilmChunk` (`film_packets.go:42-49`) = `os.ReadFile` + inflate a CHAQUE
  appel, zero memoization ; chaque `ScanFilm*` de `replay/build.go:264-475` refait
  `CountFilmChunks` + une (souvent deux) boucles completes ; recompte independant : 34-38
  passes completes dans `BuildFromFilm` selon le mode + 1 statborg + 1 killsource
  (+ ~72 lectures partielles bornees a 6 chunks, + 10-13 re-parses du chunk_00 registre).
- Consequence : ~750 Mo lus et redecompresses pour ~20 Mo de donnees uniques sur le film
  temoin (amplification x37), en pur CPU zlib + churn d'allocation (`io.ReadAll` non
  pre-dimensionne, `film_packets.go:62`).
- Reproduction : suivre `BuildBytes` (`replaybuild.go:194`) et compter les boucles
  `ReadFilmChunk` ; table complete dans le rapport de verification (agent V1).
- Verification adverse : TIENT — le recompte independant depasse legerement le chiffre
  initial. Deux redondances supplementaires trouvees : double `worldObjectSlotBand`
  (`equipment_placements.go:126` vs `projectiles.go:115`, une passe entiere gratuite) ;
  le motif de partage `...ForBand` existe deja pour les objets du monde
  (`build_ground_weapons.go:11-13` : « TROIS LECTURES PAR ARCHETYPE, PAS UNE DE PLUS »)
  mais pas pour les scanners bipedes.
- Traitement propose : charger les chunks decompresses UNE fois en memoire (patron
  `killsource.loadFilm`, `chunks.go:126-153`, deja dans le depot) et servir tous les
  scanners depuis la (RAM bornee ~ pics actuels 48-256 Mio, sous le plafond 3 Gio).
  Decision : a trancher.

### [P1] C2 — 6 scanners recalculent chacun bande de slots + layout (+ ~1,45 passe chacun)
- Ou : `held_weapon_changes.go:204`, `inventory_delta.go:196`, `ability_rank.go:123`
  (partage par `equipment_changes.go:129`), `camo_state.go:92`, `grapple_state.go:89`,
  `biped_pickups.go:163` — chacun refait `bipedSlotBand` (1 passe complete) et
  `DetectI0Layout` (borne a 6 chunks, ~0,45 passe) alors que `ScanFilmBipedPositions`
  sait deja RECEVOIR un layout (`offline_biped.go:178-186`).
- Consequence : ~9 passes completes evitables + 10-13 re-parses du chunk_00 (~1067
  allocations de chaines chacun).
- En prime, JUSTESSE : ces 6 scanners consomment le layout AUTO-DETECTE que le chemin des
  positions refuse par doctrine (`build.go:249-256` : sur une carte a >2 regions, l'index
  de region se lit comme un bit d'axe). Divergence de decodage a verifier sur un temoin
  multi-region (Live Fire) — potentiellement des attributions de slot fausses.
- Verification adverse : TIENT sur les 6 scanners et le principe ; chiffre corrige de
  2 passes a ~1,45 passe chacun (`detectMaxChunks = 6`, `i0_layout.go:128`).
- Traitement propose : calculer bande + layout + archetypes une fois dans `BuildFromFilm`
  et les passer (generaliser le motif `...ForBand`). Decision : a trancher.

### [P1] C3 — Boucles chaudes du decodeur : le cout par passe est lui-meme gonfle
- Ou / quoi :
  1. resynchronisation glissante bit-a-bit avec **consultation de map par bit candidat**
     (`offline_biped.go:272` + `:332` `slots[slot]` map[uint32]bool ; motif replique dans
     `ability_rank.go:159-162`, `camo_state.go:139-142`, `grapple_state.go:131-134`,
     `held_weapon_changes.go:120-123`, `i0_layout.go:194-197`) ;
  2. lecteurs de bits a 1 iteration par bit (`bitreader.go:33-44`, `offline_biped.go:395-402`
     — 78 % du CPU au profil du 2026-08-01) ;
  3. `killsource.locateStrict`/`locateFallback` : un decodage ECS complet tente par
     position de bit (`walk.go:83-111`), + `World.Snapshot()` copie la map des slots a
     CHAQUE paquet (`walk.go:154` -> `world.go:135-141`) ;
  4. 2 x `NewBitReader` par record alors que `SetBitPos` existe pour l'eviter — c'est ecrit
     dans bitreader.go (`offline_aim.go:248,263` vs `bitreader.go:52-53`) ;
  5. consultations de maps a cle CHAINE par composant par record (`traverse.go:1276,1293`)
     alors que l'index de composant est stable ;
  6. `ascendingFromZero` alloue avant de valider, jete sur chaque candidat rejete
     (`offline_biped.go:359-364`).
- Consequence : sur le temoin 278 s, une fois l'inflate x40 retire, ce sont ces boucles qui
  portent l'essentiel du temps restant (camo 16 s, grappin 14 s par balayage...).
- Verification adverse : sites verifies sur pieces (V1/V2) ; le profil de reference est
  celui du 2026-08-01.
- Traitement propose : slots en `[]bool`/bitset indexe par slot ; lecture de bits mot a mot ;
  reutiliser un lecteur via `SetBitPos` ; cles de composant par index. Decision : a trancher.

### [P1] C4 — `NamedEventsFrom` : ~22 re-balayages des StatRecords AVANT la bombe connue
- Ou : `objectiveevents/named.go:207-212` — pour chacune des ~11 cles de la table,
  `seriesBySlot` -> `rawSeriesByRound(recs)` rebalaie TOUT `recs` et realloue la map
  imbriquee, + `RealRounds(recs)` rebalaie encore.
- Consequence : multiplicateur de CPU et d'allocations en amont d'`incrementTimes` (la
  bombe ~26 Gio, dette assumee) ; distinct d'elle, non couvert par le registre existant.
- Verification adverse : non refute (site verifie par V1 sur la structure des appels).
- Traitement propose : une seule passe de regroupement par cle, et le plafond de prudence
  d'`incrementTimes` deja preconise au registre 2026-08-24 (le poser DEBLOQUE la re-cuisson
  de masse). Decision : a trancher.

### [P1] C5 — Orchestration : couts fixes et serialisations evitables autour du decodage
- Ou / quoi (tous verifies sur pieces, V3) :
  1. ouvrier : telechargement des chunks STRICTEMENT sequentiel (`replay-worker/job.go:308-321`)
     alors que le serveur fait deja 8 en parallele (`haloclient/halo_client_film.go:241-267`) ;
  2. ouvrier : relit du disque l'artefact qu'il vient d'ecrire (`job.go:242`) parce qu'il
     appelle `BuildMatch` au lieu de `BuildBytes` — contraire a la doctrine ecrite
     (`replaybuild.go:277-279` reserve `BuildMatch` aux appelants in-processus) ;
  3. post-sync : `etatArtefact` (`artifacts.go:260-265`) fait 2 ReadFile+Unmarshal du meme
     artefact la ou `readArtifactDigest` rend deja les deux reponses en une lecture
     (`replaybuild.go:430-436`) ; meme motif x1380 candidats dans
     `cmd_backfill_replay_repair.go:73-76` (~2,5 Go lus avant le premier decodage d'une
     passe de reparation) ;
  4. `StoreArtifact` deserialise le blob 3 fois, dont `validateArtifact` qui materialise le
     `ReplayDocument` COMPLET pour 4 champs scalaires puis LE JETTE (`artifact_store.go:63,109`) ;
  5. backfill : enfant a priorite NORMALE (`backfill_child.go:171-202`, quasi-copie de
     `filmproc/runner.go` sans `lowPriority`) — le chemin qui tourne 8 h sur le poste ne
     cede pas le CPU, celui qui tourne 5 films par cycle le fait ;
  6. aucun timeout par enfant (`runner.go:182` herite du ctx du cycle) : un film pathologique
     sous le plafond memoire peut bloquer le post-sync bien au-dela du budget de 5 min
     (verifie ENTRE matchs seulement, `cuisson.go:141`) ;
  7. `NewBuilder` recharge ~133 Ko imperatifs + jusqu'a ~4,4 Mo paresseux de catalogues par
     ENFANT donc par match ; le telechargement du film N+1 ne recouvre jamais le decodage
     de N (deux instructions du meme corps de boucle, `cuisson.go:147,175` — rien d'autre
     ne l'empeche, le decodage est dans un autre processus).
- Consequence : latence par match et debit de masse ampute, sans aucun gain de securite.
- Traitement propose : appliquer R1/R3/R4/R8/R9/R2 (rapport orchestration) ; R5
  (alleger `validateArtifact` au digest) est un ALLEGEMENT DE GARDE a trancher explicitement.
  Decision : a trancher.

### [P1] C6 — Instrumentation : la duree est mesuree puis JETEE ; toutes les phases sont aveugles
- Ou : `filmproc/runner.go:205` produit `Result.Dur` ; `replaychild.go:173-183` consomme
  Issue/Code/Peak/Err et jamais Dur ; le log de succes post-sync (`cuisson.go:189-190`)
  porte tracks/bytes sans duree ; `cmd/replay-build` : zero `time.Since` ; aucune phase
  (telechargement agrege, passes de decodage, marshal, ecriture) n'a de chronometre en
  production ; le seul profil heap (`51101d1d`) est un item de backlog jamais execute.
- En prime : les pics RAM de `cout_machine.tsv` (temoins d'aout) mesurent le processus
  `go` LANCEUR, pas le decodeur (`run_replay_build.ps1:48` sur `go run`) — mesure fausse
  a ne plus citer.
- Consequence : impossible de verifier un gain ou d'attribuer une regression sans re-faire
  un harnais externe a chaque fois.
- Verification adverse : TIENT (V3).
- Traitement propose : garder `res.Dur` jusqu'au log de succes ; un `slog` de duree par
  balayage dans `BuildFromFilm` ; harnais pprof CPU+heap sur 2 temoins a HEAD + `51101d1d`.
  Decision : a trancher (phase 0 du chantier).

### [P2] C7 — Le verrou solo machine ne couvre qu'UN des quatre points d'entree
- Ou : `filmproc.AcquireSolo` (`solo.go:91`) n'a qu'un appelant de production
  (`cmd/replay-build/main.go:101`). Ni le post-sync (`replaychild.go:96`), ni le backfill
  (`cmd_backfill_replay_child.go:36`), ni l'ouvrier (`replay-worker/main.go:154-174`) ne le
  prennent ; aucune autre exclusion inter-process n'existe (V3, quatre candidats tombes).
- Consequence : une passe backfill et un post-sync (ou deux passes lancees a la main)
  peuvent decoder en meme temps sur la machine — la moitie inverse du scenario du
  2026-08-31 que `solo.go:20-22` documente.
- Verification adverse : TIENT.
- Traitement propose : escalade utilisateur — c'est une decision de protection (etendre le
  verrou aux trois chemins, ou le transformer en compteur a N creneaux si le debit de masse
  passe a 2 enfants). Decision : escalade.

## Constats ecartes

| Constat | Motif d'ecart |
|---|---|
| « Le decodage par chunk est parallelisable sur le chemin replay » | REFUTE (V2) pour 5 scanners : `held_weapon_changes` (automate prevFam/seen traverse les chunks, `:107-149`), `equipment_changes` (pas de compteur modulo 8 inter-emissions, `:127-158`), `inventory_delta` (verdict de film retroactif qui EFFACE des records deja emis, `inventory_delta_ammo.go:178-199`), `projectiles` (jointure en vies sur trou de 250 ms), `player_indices` (vote avec veto). Le corps de `ScanBipedRecords` tient, mais la bande de slots depend de la partition (`fillSlotBand`) et les post-filtres sont des replis ordonnes — resultat non decomposable par chunk. |
| « Les ~22 scanners forment un DAG presque sans aretes, parallelisable une fois l'etat de-globalise » | REFUTE (V2) : `MPPWidths` est ecrit dans la boucle la plus interne de la calibration (`equipment_creation_width.go:302`, des dizaines de milliers d'ecritures transitoires) et LU par le bloc default-state de toute traversee (`default_state.go:332,344`) ; hooks partages en protocole save/restore strictement LIFO mono-thread ; aretes manquees (zoom<-positions via `buildScopedLookup`, pads<-placements via les hooks). Surface reelle de de-globalisation : 106 var de paquet, 64 `Set*` exportes, >= 80 vars mutees, 28 hooks -> reecriture de la signature de traversee. |
| « Les temoins 208-284 s incluent la compilation go run » | REFUTE (V3) : part hors-decodage <= 1,2 % (bornage par horodatages slog vs total watchdog). |
| « +84 s = capacites i48 ; +94 s = pont slot->joueur » | Attribution corrigee (V3) : les horodatages ferment des FENETRES multi-balayages ; le pont et l'assemblage tiennent en < 1 s. |
| « StoreArtifact deserialise le blob 4 fois » | Ramene a 3 (V3) : la 4e lecture est le garde anti-regression qui lit l'artefact EN PLACE — legitime et justifiee sur 25 lignes (`artifact_store.go:140-164`). |
| « 6 scanners x 2 passes completes supplementaires » | Chiffre corrige a ~1,45 (V1) : `DetectI0Layout` est borne a 6 chunks. Les 6 scanners restent bien identifies. |
| `CountFilmChunks` dans la condition de boucle (`zoom_events.go:95`, n+1 x n os.Stat) | Vrai sur le fait, poids nul en performance (millisecondes) — defaut de style, pas un constat perf. |
| `io.ReadAll` « alloue ~2N » | Vrai sur l'absence de pre-dimensionnement ; quantification nuancee (2N cumule sur les doublements, pas un pic) et zlib ne porte pas de taille decompressee — heuristique de ratio seulement. Absorbe par C1 (le cache d'octets supprime la question). |
| Bombe RAM `NamedEventsFrom`/`incrementTimes` | Dette assumee (registre 2026-08-24/25) — seul le MULTIPLICATEUR amont (C4) est nouveau. |
| Existence du verrou filmdec, de la priorite basse, du plafond memoire, de l'enfant borne | Doctrine assumee — hors constat par cadrage. |

## Axes sans constat

- Assemblage du document (`BuildFromPositions`) : < 1 s au temoin, recherches deja
  logarithmiques (`sort.Search`), aucun quadratique trouve. Rien a faire la.
- Serialisation + ecriture atomique + fsync : < 1 s, ecriture atomique correcte.
- Chemins de repli d'inference (`inferChain`/`inferRepair`) : false par defaut, aucun
  appelant de production — leurs maps recreees ne tournent pas en prod.
- Pre-dimensionnements deja corrects : `fillSlotBand`, `cloneSlots`, `DropIsolated`,
  `coveredInstants`, etc. (liste V1/agent 2).

## Suite proposee (ordonnancement, PAS un plan execute)

- Phase 0 — mesurer (C6) : duree par balayage + `res.Dur` conserve + pprof CPU/heap sur
  2 temoins HEAD et `51101d1d`. Sans elle, aucun gain n'est verifiable.
- Phase 1 — le gros du gain, sans toucher aux protections : C1 (chunks decompresses une
  fois, patron killsource), C2 (bande/layout/archetypes partages + justesse layout),
  C3 (boucles chaudes), C4 (NamedEventsFrom + plafond incrementTimes — debloque aussi la
  re-cuisson de masse).
- Phase 2 — orchestration (C5) : R1 telechargements ouvrier x8, R3 BuildBytes chez
  l'ouvrier, R4 lecture unique, R8 priorite basse backfill, R9 deadline par enfant,
  R2 prefetch N+1 (profondeur 1).
- Phase 3 — debit de masse SI encore necessaire apres phases 1-2 : 2 creneaux backfill
  avec les quatre contreparties nommees (Job Object Windows, budget global divise, verrou
  a N creneaux — cf. C7, ordre de passe revu). PAS de multi-thread intra-decodeur : refute.
- Sequencage avec le schema vehicules : faire phases 0-1 AVANT le bump, pour que la
  re-cuisson des 1 380 films tourne au tarif optimise et une seule fois.

Escalades utilisateur : C7 (portee du verrou solo) ; R5 (allegement de `validateArtifact`) ;
la divergence layout de C2 si le temoin multi-region confirme des slots faux (justesse, pas
perf).

Audit realise par 4 auditeurs + 3 verificateurs adverses (sessions fraiches), 2026-09-02.
L'audit n'a modifie aucun code.
