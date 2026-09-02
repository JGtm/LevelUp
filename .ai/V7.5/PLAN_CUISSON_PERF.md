# Plan — Cuisson des artefacts de rejeu : decoder une fois, mesurer, et tenir 60-100 s

> Cree le 2026-09-02, REECRIT le meme jour apres revue adversariale (§9 : 9 constats bloquants,
> 12 recevables). Branche cible : `wt/cuisson-perf` (worktree `LevelUp-wt-cuisson-perf`, base
> `feat/v75` `900384f50`). Contrat d'execution : skill `plan-execution` (defaut) — ce plan ne le
> remplace pas, il s'y soumet. Source : `.ai/AUDIT_CUISSON_REPLAY_PERF_2026-09-02.md` (7 constats
> retenus, 3 verifications adverses).
>
> Cache film : le worktree monte `data/cache/{film_chunks,film_manifests,mvar,replays}` par
> JONCTION vers `LevelUp-go-migration/data/cache` (1 380 films). La seule lecture de base du plan
> est l'export des faits du lot 0 (§4, item 0.2), par `OpenReadForQuery`.

## 1. Objectif et critere de succes

**Probleme.** Un artefact de rejeu relit et redecompresse le film entier ~36-40 fois
(`filmdec.ReadFilmChunk` sans memoization, chaque `ScanFilm*` reboucle tout le film ; six
balayages recalculent bande de slots + layout + archetype ; trois implementations du meme inflate
et trois marcheurs de paquets DIVERGENTS). Temoins mesures : 208-284 s au schema 18, decodage
pur ; HEAD (schema 34) fait plus de balayages. Les boucles chaudes (consultation de map par bit
candidat, lecteur a 1 bit/iteration) gonflent chaque passe. Aucune phase n'est chronometree.

**Critere de succes (mesurable, gate du lot 6).**
1. **Equivalence** : sur le corpus d'equivalence (11 films, item 0.1), tous les lots de REFACTO
   PUR produisent des sorties IDENTIQUES A L'OCTET (digest par balayage et par sortie de
   `readFilmStats`, SHA-256 de l'artefact) a celles figees a `900384f50` AVEC LES FAITS DU MATCH.
   Les seuls ecarts admis sont ceux des lots de CORRECTION declares (lot 3 Live Fire, lot 4b
   films-bombes), chacun avec son diff chiffre au journal.
2. **Vitesse** : sur `01e1f945` (KOTH), `7344d24f` et `696a9d7c` (Strongholds), duree de cuisson
   totale <= 100 s chacun (protocole §6, meme machine que la reference du lot 0). Si l'objectif
   n'est pas atteint apres le lot 4, le plan CLOT en statuant l'ecart — il n'ouvre PAS de
   parallelisme intra-decodeur (refute par l'audit, §7).
3. **Protections intactes** : plafond memoire, priorite basse, enfant borne, verrou filmdec,
   « le VPS web ne decode jamais » — rien n'est retire ; le verrou solo est ETENDU (lot 5) ; le
   harnais d'equivalence lui-meme decode UN film par processus borne (D4).

## 2. Conception retenue

Trois responsabilites, la ou un balayage fait tout aujourd'hui :

| Responsabilite | Ou | Ce qui change |
|---|---|---|
| **Source du film** | NOUVEAU paquet FEUILLE `internal/analysis/filmsource` (aucun import du depot) : `Source`, `MemoryChunks`, `DirSource`, `ChunkMeta`, `Packet`, `Film`, `Load`, `LoadDir` | chunks decompresses UNE fois, decoupage en paquets fait UNE fois selon UNE grammaire (D3), metadonnees du manifeste FOURNIES par l'appelant. `filmdec`, `objectiveevents`, `killsource`, `filmcache`, `killcollector` en dependent tous ; aucun d'eux n'a plus d'inflate ni de marcheur |
| **Contexte du film** | `filmdec.FilmContext`, construit par `replay.BuildFromFilm` | bande de slots bipede, layout, archetypes du registre chunk_00 — calcules une fois, PASSES aux balayages |
| **Balayages** | les `Scan*` existants | signature `Scan*(film *filmsource.Film, ...)` : fonctions pures, zero E/S. Les `ScanFilm*(dir)` ne survivent qu'en enveloppes minces la ou des appelants NON production les utilisent (D2) |

Pourquoi un paquet feuille et pas `filmdec` (revue §9, constats 1, 2, 21) : `filmcache` importe
`objectiveevents` (`filmcache.go:32`), cinq tests INTERNES de `filmdec` importent `objectiveevents`
ou `filmcache` (`sonde_registre_verdicts_test.go:26-27`, `navpoint_ti12_radial_test.go:85`,
`objectif_ti11_minuteurs_test.go:97`, `ti47_annonces_test.go:24`, `zone_census_report_test.go:22`).
Toute solution qui fait importer `filmdec` par `objectiveevents`, ou `filmcache` par `filmdec`,
cree un cycle en production ou en test — c'est l'argument, et il est verifie (`go list -deps` :
`filmdec` n'importe aujourd'hui rien du depot). Le paquet feuille n'en cree aucun. (Note : la
couche `analysis/` importe deja `games/canonical` — `breakdown/counts.go:7`, `citations.go:17`,
`home_canonical*.go` — l'argument « analysis n'importe rien de games » serait faux ; il n'est pas
retenu.)

Ce qui ne bouge PAS (perimetre ferme, §7) : la grammaire ECS et les deserialiseurs, l'etat de
reglage global de `filmdec` (reste derriere `LockProcessDecode` ; un ratchet gele son compte,
item 1.10), l'ordre sequentiel des balayages, l'assemblage `BuildFromPositions`, l'emplacement des
paquets existants.

Memoire : le film decompresse tient en RAM — `killsource` le garde deja en entier et ses pics sur
films sains sont de 48 a 256 Mio (plafond enfant : 3 Gio). Le corpus inclut le plus gros film du
cache (`1c4c63c2`, 89 Mo compresses) pour tester cette hypothese au pire cas. Les films-bombes le
sont par `NamedEventsFrom`, pas par leur taille (lot 4b).

## 3. Decisions tranchees (defauts fermes — ne pas re-decider en cours d'execution)

- **D1 — La source du film est un paquet feuille `internal/analysis/filmsource`.** API :
  `type ChunkMeta struct{Index, ChunkType, StartMS int}` · `type Source interface{NumChunks() int;
  Chunk(i int) ([]byte, error)}` (octets BRUTS, compresses ou non — la forme de
  `killsource.ChunkSource`) · `type MemoryChunks [][]byte` · `func DirSource(dir) (Source, error)`
  (`chunk_NN.bin` tries, sans borne haute) · `type Packet struct{Chunk, Index, Type int; TS uint64;
  Payload []byte}` (payload = sous-tranche, jamais copie) · `type Film` avec `NumChunks()`,
  `Chunk(i) []byte`, `Packets(i) []Packet`, `Meta() []ChunkMeta` · `func Load(src Source, meta
  []ChunkMeta) (*Film, error)` · `func LoadDir(dir string, meta []ChunkMeta) (*Film, error)`.
  `meta` peut etre nil (enveloppes D2, tests) ; les consommateurs qui en ont besoin
  (`objectiveevents` : type de chunk, start_ms) le disent par une erreur explicite, pas par un
  resultat vide. Le paquet n'importe RIEN du depot — garde-rail archlint.
- **D2 — Les enveloppes `ScanFilm*(dir)` sont TOLEREES hors production, INTERDITES en production.**
  Elles valent `filmsource.LoadDir(dir, nil)` + `Scan*(film)` : une decompression par appel, jamais
  pire qu'aujourd'hui. Sites de PRODUCTION (enveloppe interdite, migration obligatoire au lot 1) :
  `replay/build.go`, `replay/bomb_armings.go:113`, `replay/build_ground_weapons.go:97,109`,
  `replay/build_objectives_live.go:91`, `replay/build_zones.go:49`, `replay/equipment_placements.go:318`,
  `replaybuild/`, `killsource/`, `objectiveevents/`, `sync/killcollector/positions.go:170-186`,
  `cmd/zone-attribution/measure.go:90,234`, `api/wire/registry_replay_build.go:66`. Une enveloppe
  sans appelant est supprimee (regle 0 code mort).
- **D3 — UNE grammaire de decoupage, et sa purete se MESURE avant d'etre affirmee.** Les trois
  marcheurs divergent (revue §9, constat 8) : `filmdec.WalkPackets` accepte `size == 0` et continue
  (`film_packets.go:71-89`) ; `killsource.splitPackets` s'arrete sur `size <= 0` (`chunks.go:158-171`) ;
  `objectiveevents.walkFrames` n'emet que le type 0, S'ARRETE sur `CHUNK_END = 7`
  (`film.go:47,134`), et borne par `size > len(data)` SANS l'offset (`film.go:127`) en avancant
  meme quand le payload deborde. Les inflates divergent sur un flux zlib tronque
  (`objectiveevents.decompressChunk` rend le BRUT compresse — que `walkFrames` marche alors comme
  des paquets —, les deux autres rendent le partiel). Deux AUTRES marcheurs existent HORS de la
  chaine de cuisson : `analysis/weaponv3/timing.go:33-56` et `analysis/positions/positions.go:124-141`
  (celui-ci sert la vue de match cote serveur : `api/handlers/match_view_positions.go:18`) — ils ne
  participent pas a la construction d'artefact, ne sont PAS migres par ce plan (§7), et sont
  nommes pour que « une grammaire » soit lu comme « une grammaire pour la chaine de cuisson ».
  Grammaire retenue pour `filmsource` : en-tete 16 octets LE `[u16 type][2][u32 size][u64 ts]` ;
  arret sur `size <= 0` (un paquet vide ne porte rien et un `size == 0` en boucle est le symptome
  d'un tampon corrompu) ; arret sur `off+16+size > len` ; arret sur `type == 7` (CHUNK_END) ;
  inflate rendant le PARTIEL sur flux tronque (« un film Theater se termine parfois net »,
  `killsource/chunks.go:84-85`). Le lot 0 MESURE sur les 1 380 films, pour chaque consommateur,
  si cette grammaire change le jeu de paquets qu'il voit (item 0.7). Zero divergence sur le
  corpus d'equivalence = le lot 1 est un refacto pur (gate : identique) ; une divergence sur un
  film du corpus = ARRET et escalade utilisateur (le plan ne choisit pas seul entre deux
  grammaires). Les films HORS corpus qui divergent sont listes au journal, avec le consommateur
  touche, et la liste est SOUMISE A L'UTILISATEUR AVANT LE LOT 1 : la mesure 0.7 compare des
  jeux de PAQUETS, pas des sorties — sur ces films, la sortie de cuisson changera sans etre
  mesuree par `replay-equiv`. Recommandation ecrite ici, a confirmer : accepter ces changements
  comme corrections declarees (sur ces films les trois marcheurs actuels se contredisent DEJA
  entre eux — il n'existe pas d'ancienne verite unique a preserver), plutot que d'etendre le
  corpus a des films que les gates ne pourraient de toute facon pas declarer « identiques ».
- **D3bis — Refacto pur et correction ne se melangent jamais dans un lot.** Lot 2 passe aux six
  balayages delta le layout AUTO-DETECTE calcule une fois (sortie identique) ; c'est le lot 3,
  seul, qui bascule sur le layout du CATALOGUE (sortie differente sur Live Fire, et seulement la).
  Meme decoupage pour `NamedEventsFrom` : regroupement en une passe (identique, lot 4) puis
  plafond (films-bombes seuls, lot 4b).
- **D4 — L'equivalence se prouve par DIGESTS figes depuis l'ANCIEN code, AVEC les faits du match,
  UN film par processus borne.** Trois pieces :
  (a) un OBSERVATEUR dans le code de production, pas une copie de l'orchestration :
  `replay.Options.Observe func(step string, v any)` appele apres chaque balayage de
  `BuildFromFilm` (nom du balayage, valeur), et `replaybuild.BuildBytes` l'appelle pour chaque
  sortie de `readFilmStats` (`statRecords`, `namedEvents`, `objectives`, `score`, `flag`, `vip`,
  `skull`, `bomb`), pour `killsource` (`ksRes`), pour les catalogues resolus (`flagSpawns`,
  `zones`, `spawnPoints`) et pour le blob final. Nil = aucun cout. Si `BuildFromFilm` gagne un
  balayage, le digest le voit ; une copie de la sequence dans un harnais ne l'aurait pas vu.
  Le digest d'une etape est le SHA-256 du JSON canonique de sa valeur : il ne voit que les
  champs EXPORTES (ex. le struct embarque non exporte `componentDirs` de `filmdec.BipedPosition`,
  `offline_biped.go:79`, est invisible) — c'est le digest de l'ARTEFACT qui couvre le reste ; et
  `encoding/json` refuse NaN/Inf : l'encodeur du harnais remplace un flottant non fini par la
  chaine `"NaN"`/`"+Inf"`/`"-Inf"` AVANT le hachage, pour qu'un film pathologique rende un digest
  et non une erreur ;
  (b) `cmd/replay-equiv` : PARENT qui lit `CORPUS.txt` et lance UN ENFANT PAR FILM par
  `filmproc.Runner` (plafond 3 Gio, priorite basse, verrou solo en attente bornee — D7).
  L'ENFANT ecrit lui-meme le fichier de digests que le parent lui designe (`-out <chemin>`) :
  une ligne `etape  compte  sha256(JSON canonique)` par etape, une ligne `artifact  octets  sha256`.
  Le tube du runner fusionne stdout et stderr et relaie le journal (`runner.go:190-192`) : il ne
  transporte AUCUNE donnee de digest. Le PARENT compare ensuite le fichier de l'enfant a
  `testdata/equivalence/<film>.tsv` (ou l'y copie avec `-update`) et nomme la PREMIERE etape qui
  differe. Declare au ratchet `no_unbounded_film_loop_test.go` (« enfant borne, un film par
  processus ») et arme sa sentinelle (second ratchet). Le harnais N'ENCHAINE JAMAIS deux films
  dans un processus — c'est le motif des quatre sinistres RAM ;
  (c) `TestEquivalenceMiniFilm` (Go test, CI) : digests des balayages que la mini-bobine
  `replay/testdata/minifilm_000d5950` supporte — liste FERMEE : tirs, lancers de grenade,
  loadouts d'image-cle, inventaire d'image-cle, morts (highlight), indices joueur, projectiles
  (fenetre) — SANS `BuildFromFilm` ni `BuildBytes` (la bobine n'a ni chunk_00, ni manifeste, ni
  positions valides : `world_object_precision_guard_test.go:26-29,76-81`, `PROVENANCE.txt`). La
  CI ne couvre donc PAS le registre ni les positions : c'est le corpus local qui les couvre, et
  c'est ecrit.
  Regeneration des digests : uniquement par `-update`, uniquement aux lots 0 (creation), 3 et 4b
  (ecarts declares) ; chaque regeneration porte au journal le diff des comptes. Un digest qui
  bouge dans un lot de refacto pur = le lot echoue.
- **D5 — Instrumentation en production.** `slog.Debug` par balayage (`"duration"`), `slog.Info`
  par phase de `BuildBytes` ; `replaychild.Spawn` rend `(Built{Blob, Dur, Peak}, error)` au lieu
  de `([]byte, error)` et `BuildOneFunc` suit — c'est le seul chemin par lequel `res.Dur`
  (`filmproc/runner.go:205`) atteint le log de succes de `cuisson.go:189` ; duree totale et
  `-cpuprofile` dans `cmd/replay-build`, qui installe `logging.InstallCLILevel` avec le niveau de
  `LEVELUP_LOG_LEVEL` (aujourd'hui : aucun handler, les Debug sont perdus). Aucun test de temps.
- **D6 — Primitives chaudes : test differentiel contre une COPIE DE REFERENCE en `_test.go`**, sur
  le DOMAINE REEL : `ReadBits` n = 0..64 (appels a 64 : `components_world.go:90,92`,
  `default_state.go:260`, `equipment_creation.go:401`, `traverse.go:1360` ; 59 : `traverse.go:288`),
  positions autour des frontieres d'octet, de mot et de fin de tampon. Semantique hors tampon
  PRESERVEE par fonction : `ReadBits` rend 0 (`bitreader.go:36-39`), `readBitsAt` panique
  (`offline_biped.go:395-402`, aucune garde) — la reecriture ne change ni l'une ni l'autre, et le
  test le verifie. Le code de production ne garde aucune copie.
- **D7 — Le verrou solo devient a DEUX regimes, et le second nait au lot 0.** `AcquireSolo`
  (refus immediat, inchange) pour le post-sync et `cmd/replay-build` ; `AcquireSoloWait(ctx,
  cacheRoot, tool, id, max)` (attente bornee puis refus) CREE A L'ITEM 0.4 avec son test, parce
  que `cmd/replay-equiv` (gate de tous les lots) en a besoin des le lot 0 ; le lot 5 (item 5.7)
  ne fait que le CABLER dans l'enfant de backfill et l'ouvrier. Portee inchangee : la racine de
  cache.
- **D8 — L'ouvrier n'ecrit plus d'artefact local.** `BuildBytes` + envoi des octets ; l'artefact
  canonique est celui que le serveur range. Sa relecture disque (`job.go:242`) disparait.
- **D9 — `validateArtifact` n'est PAS allege** (R5 de l'audit : decision utilisateur, §7).
- **D10 — Pas de parallelisme intra-decodeur, pas de de-globalisation, pas de deplacement de
  paquet existant.** Le ratchet de l'item 1.10 gele le nombre de `var` de paquet de `filmdec`.
- **D11 — Films-bombes exclus du corpus d'equivalence jusqu'au lot 4b.** `51101d1d` et `a349fea8`
  entrent au corpus au 4b, premier digest fige alors.
- **D12 — Ordre des lots = risque croissant, gain decroissant.**
- **D13 — Bornes du plafond (lot 4b), ECRITES, sur la BONNE grandeur.** `incrementTimes`
  (`objectiveevents/named.go:350-359`) deroule `p.Value - prev` avec `prev` PARTANT DE ZERO et
  jamais reinitialise (`named.go:340-342` : « la premiere valeur observee est comptee depuis
  zero ») : la grandeur qui explose est ce deroulage, PREMIER TERME COMPRIS (`pts[0].Value - 0`),
  pas l'ecart entre deux echantillons consecutifs. Bornes : (1) dans `incrementTimes`, un
  deroulage `p.Value - prev > 10 000` est REJETE (le point est saute, `prev` avance a `p.Value`
  sans emission), compte et journalise (`slog.Warn`, comp, cote, valeur) ; (2) dans
  `NamedEventsFrom` — et pas dans `incrementTimes`, appele dans une double boucle cles x slots
  (`named.go:207-212`) ou un plafond serait PAR APPEL —, le total d'evenements emis par film est
  plafonne a 1 000 000 (au-dela : arret du deroulage, `slog.Warn`). 10 000 est un dernier
  rempart, pas un filtre : la pire anomalie connue est un saut de 66 (`named.go:280-291`), la
  bombe vaut des milliards d'entiers. Le lot 4b mesure d'abord le deroulage maximal reel
  (premier terme compris) sur les 11 films sains : s'il depasse 1 000, escalade utilisateur
  avant de poser la borne.
- **D14 — Sites `killsource` hors periмetre, SANS condition.** `locateStrict`/`World.Snapshot`
  ne sont pas dans ce plan. Une mesure du lot 0 qui montrerait killsource dominant est une
  Decouverte (§8) a escalader, pas un item qui s'ajoute.

## 4. Lots

Statuts : `[x]` fait et verifie · `[~]` couvert par un autre item (reference) · `[!]` non traite
(justification au journal). Aucune case vide a la cloture d'un lot. Un lot est CLOS quand son gate
a tourne vert dans la session, ses items sont statues, ce fichier et `thought_log.md` sont a jour,
et le commit du lot est fait (prefixe `cuisson-perf(L<n>)`).

### Lot 0 — Reference, instrumentation, mesure de divergence — effort moyen

- [ ] 0.1 Corpus FERME (verifie present au cache le 2026-09-02, chunks / Mo compresses) :
      `000d5950` Cliffhanger Fiesta (28 / 20,2) · `01e1f945` Catalyst KOTH (30 / 22,5) · `64e8adfa`
      CTF (45 / 33,7) · `7344d24f` Strongholds (33 / 25,8) · `696a9d7c` Strongholds (31 / 26,6) ·
      `084a804d` CTF, temoin historique 19 min (57 / 58,9) · `1c4c63c2` LE PLUS GROS film du cache
      (69 chunks / 89) · `53ce4390` CTF, ancre historique des bursts (41 / 31,8) · `d9781168`
      Oddball (39 / 29,8) · `9f57c612` Assaut One Bomb, temoin `navpoint_ti12` (27 / 19,8) ·
      `60ae07c4` Live Fire Ranked (44 / 32,0 — ecart attendu au lot 3 seulement). Total : 11 films.
      `BOMBES.txt` : `51101d1d` (13 / 9,1), `a349fea8` (51 / 65,9). Fichier
      `apps/go-api/internal/analysis/replay/testdata/equivalence/CORPUS.txt`, colonnes : short8,
      carte(s) candidate(s), variante, chunks, Mo, raison — cartes et variante REMPLIES PAR 0.2.
- [ ] 0.2 Export des faits : `levelup replay-facts-export --out <dir> <short8...>` — lit, par
      `OpenReadForQuery` (lecture seule, sure meme si le serveur tient la base — il ecoute sur
      :8000 au moment ou ce plan est ecrit), pour chaque match : `port.MatchFacts`
      (`ReplayFactsRepo.FactsForMatch`, deja appele depuis `cmd_backfill_replay_child.go:95`) et
      les identites de carte candidates par `mapNamesForOne` (`cmd_backfill_replay.go:350-369`,
      MEME paquet `main` — jamais une troisieme copie des requetes de `selectBuildWork`), et ecrit
      `<short8>.facts.json` (forme deja lue par `cmd/replay-build --facts`, `loadFacts`) enrichie
      de `mapNames` (champ ignore par les lecteurs existants). Les 13 fichiers (corpus +
      bombes) vont dans `testdata/equivalence/` et `CORPUS.txt` est complete depuis eux. Sans les
      faits, zones, actions d'objectif, VIP/crane/bombe, socles et points d'apparition sont
      COURT-CIRCUITES (`zones.go:78-82`, `matchfacts.go:103-107,192`) et l'equivalence ne les
      verrait pas (revue §9, constat 6).
- [ ] 0.3 Observateur (D4a) : `replay.Options.Observe` + appels dans `BuildFromFilm` apres chaque
      balayage, et dans `BuildBytes` pour chaque sortie de `readFilmStats`, `ksRes`, catalogues,
      blob. Test unitaire : sur la mini-bobine, la liste des etapes observees est exactement la
      liste FERMEE attendue (un balayage ajoute sans etape observee fait echouer le test).
- [ ] 0.4 `filmproc.AcquireSoloWait(ctx, cacheRoot, tool, id, max)` (D7) dans `solo.go`, avec
      son test (deux acquisitions concurrentes sur la meme racine : la seconde attend, puis passe
      quand la premiere relache ; depasse `max`, elle refuse). PUIS `cmd/replay-equiv` (D4b) :
      parent + enfant (meme binaire, drapeau interne), `filmproc.Runner`, `AcquireSoloWait`,
      sentinelle `filmproc.Arm` ; `-update`, `-corpus`, `-films`, `-out` (enfant) ; sortie du
      parent : premiere etape qui differe, par film. Declare au ratchet avec sa justification
      datee. `TestEquivalenceMiniFilm` (D4c).
- [ ] 0.5 Figer les digests a `900384f50` : `-update` sur les 11 films, puis UNE SECONDE
      EXECUTION sans `-update` qui doit passer — preuve de DETERMINISME du harnais (une iteration
      de map non triee residuelle se verrait ici). Les deux sorties au journal.
- [ ] 0.6 Instrumentation (D5) : durees par balayage/phase ; `replaychild.Spawn` → `Built{Blob,
      Dur, Peak}` et `BuildOneFunc` ; log de succes post-sync avec `duration` ; `cmd/replay-build` :
      `logging.InstallCLILevel(repoRoot, niveau(LEVELUP_LOG_LEVEL))`, duree totale, `-cpuprofile`.
- [ ] 0.7 Mesure de divergence des grammaires (D3) : `cmd/replay-equiv -walkers` — pour CHAQUE
      film du cache (1 380), inflate une fois par chunk, puis compare, chunk par chunk, le jeu de
      paquets (type, ts, taille) vu par les trois marcheurs de la chaine de cuisson et par la
      grammaire retenue — sur les QUATRE axes de D3 : `size == 0`, `CHUNK_END`, borne haute avec
      ou sans offset (et avance sur debordement), flux zlib tronque (brut compresse marche comme
      des paquets vs partiel). Un film par processus borne. Sortie :
      `.ai/V7.5/MESURES_CUISSON_PERF.md` §divergence — nombre de films sans divergence, liste des
      films divergents avec l'axe et le consommateur touches, nombre de films a flux tronque.
      Verdict ecrit : le lot 1 est-il un refacto pur sur le corpus (D3) ? La liste des films
      divergents hors corpus est soumise a l'utilisateur (D3, R-11).
- [ ] 0.8 Mesure de reference HEAD (§6) sur `01e1f945`, `7344d24f`, `696a9d7c`, `1c4c63c2` :
      duree totale, duree par balayage, pic, profil CPU de `01e1f945` →
      `MESURES_CUISSON_PERF.md` §reference.
- [ ] 0.9 Note dans `.ai/V7.5/replay2d/registre_film/LOTCTER_VOLET3.md` : les pics RAM de
      `cout_machine.tsv` mesurent le processus `go` lanceur, pas le decodeur (audit C6).
- Gate 0 : `cd apps/go-api && go test ./internal/analysis/replay/ ./internal/analysis/filmdec/ ./internal/replaybuild/ ./internal/sync/replayartifacts/ ./internal/replaychild/ ./internal/archlint/ ./cmd/levelup/... ./cmd/replay-equiv/...`
  vert ; `go run ./cmd/replay-equiv -corpus internal/analysis/replay/testdata/equivalence/CORPUS.txt`
  vert deux fois de suite (0.5) ; `MESURES_CUISSON_PERF.md` porte reference + divergence ;
  `make go-api-lint` sans nouvelle dette.

### Lot 1 — Source unique du film : decoder une fois — effort lourd

Prerequis : verdict 0.7 = zero divergence sur les 11 films du corpus (sinon : arret, escalade).

- [ ] 1.1 `internal/analysis/filmsource` (D1) : `source.go` (Source, MemoryChunks, DirSource),
      `film.go` (ChunkMeta, Packet, Film, Load, LoadDir, inflate, framing D3), tests unitaires sur
      des chunks CONSTRUITS (zlib d'octets connus : un paquet, deux paquets, size 0, CHUNK_END,
      flux tronque — valeurs calculables, pas arbitraires) + test « aucun import du depot »
      (archlint).
- [ ] 1.2 `filmdec` : `FilmPacket`/`WalkPackets`/`ReadFilmChunk`/`CountFilmChunks`/`inflateChunk`
      remplaces par `filmsource` ; `ParseRegistryChunk` recoit des octets deja decompresses
      (`registry.go:104` perd son inflate) ; les 17 `ScanFilm*` de `build.go:264-475` et les
      balayages hors plage (`bomb_armings.go:113`, `build_ground_weapons.go:97,109`,
      `build_objectives_live.go:91`, `build_zones.go:49`, `equipment_placements.go:318`) obtiennent
      leur forme `Scan*(film, ...)` ; `replay.BuildFromFilm(matchID, title, film *filmsource.Film,
      opt)` ; `deaths_source`, `origin`, `player_index`, `inventory_decode` idem ; le chunk
      highlight se lit UNE fois (`kills.go:67`, `matchfacts.go:196`, `build.go:449`).
- [ ] 1.3 `replaybuild.BuildBytes`/`BuildMatch` gardent leur signature (`filmDir string`) et
      chargent le film UNE fois (`filmsource.LoadDir(filmDir, meta)` avec `meta` du manifeste via
      `filmcache.OpenChunkDir`, deja ouvert en `matchfacts.go:72`) ; `readFilmStats`,
      `decodeKillSource`, `killRefs`, `BuildFromFilm` recoivent le `*Film`. Appelants inchanges :
      `api/wire/registry_replay_build.go:66`, `cmd/replay-build`, `cmd_backfill_replay_child.go`,
      `replaychild`, `replay-worker`.
- [ ] 1.4 `killsource.Decode(ctx, matchID, film *filmsource.Film, ...)` ; `inflate`,
      `splitPackets`, `dirChunks`, `ChunkSource`, `MemoryChunks` supprimes ; `loadFilm` ne fait
      plus que trier `t0` depuis `film.Packets`. Appelants migres : `replaybuild/kills.go:33`,
      `sync/killcollector/collector.go:284` (+ `bridge.go:100` : `filmsource.MemoryChunks` et
      `ChunkMeta` construits depuis les `haloclient.FilmChunk`), `cmd/killsource/main.go:206`.
- [ ] 1.5 `objectiveevents` : les NEUF points d'entree qui prennent un `FilmSource` passent a
      `*filmsource.Film` — `Extract` (`extract.go:103`), `CaptureBurstTimes` (`extract.go:312`),
      `StatRecords` (`statborg.go:176`), `StatRecordsCtx` (`statborg.go:186`), `NamedEvents`
      (`named.go:190`), `CrossCheckNamedEvents` (`named.go:393`), `SlotIdentity`
      (`slotidentity.go:69`), `SlotIdentityResolved` (`slotidentity_deaths.go:95`),
      `SlotIdentityFromDeaths` (`slotidentity_deaths.go:123`) ; `decompressChunk`, `walkFrames`,
      `FilmSource`, `ChunkMeta` supprimes. Appelants (grep de reference, a rejouer au moment du
      lot : `objectiveevents\.(Extract|CaptureBurstTimes|StatRecords|StatRecordsCtx|NamedEvents|CrossCheckNamedEvents|SlotIdentity|SlotIdentityResolved|SlotIdentityFromDeaths)\(`
      sur `apps/go-api` — 57 sites le 2026-09-02) : 7 en PRODUCTION — `replaybuild/matchfacts.go:83`
      (`StatRecordsCtx`) et `:170` (`CaptureBurstTimes`), plus les signatures `bombInput`/`flagInput`
      (`:121`, `:166`) ; `cmd/zone-attribution/measure.go:200,201` ; `cmd/diag_weapons_v3/process.go:35` ;
      `cmd/statnames-sweep/sweep.go:98` ; `cmd/oddball-terrain/decode.go:63` — et ~50 en TESTS
      (fichiers `replay/*_test.go` assaut_*, bombe_*, attachement_*, colline_*, lettres_*,
      objectifs_*, oddball_*, skull_*, totalcontrol_*, vip_*, zone_state_*, et `filmdec/
      sonde_registre_verdicts_test.go:158`, `zone_census_report_test.go:111`). Les tests ne changent
      que par leur PRODUCTEUR de source : `filmcache.OpenChunkDir` (4 sites : `matchfacts.go:72`,
      `lettres_ordre_research_test.go:308`, `zone_state_p2a_corpus_test.go:184`,
      `filmdec/zone_census_report_test.go:124` `zcOpenSource`), les helpers `p2aSource`, et les
      trois doubles de test d'`objectiveevents` (`extract_test.go:48-68` `diskFilmSource`,
      `statborg_rounds_test.go:170` `repeatingSource`, `assaut_footer_research_test.go:78`
      `afSource`) qui deviennent des `filmsource.Load(MemoryChunks, meta)`. Nouveau helper
      `filmcache.LoadFilm(root, short) (*filmsource.Film, error)` = manifeste + `LoadDir` avec
      `meta`, pour les tests et CLI. `filmcache.Source` implemente `filmsource.Source` et expose
      `Meta() []filmsource.ChunkMeta` (n'importe plus `objectiveevents`) ; `filmcache_guard_test.go`
      (regex ancree sur le NOM `Chunks()` et allowlist de fichiers implementant `FilmSource`, dont
      la justification « cycle d'import » devient caduque) est SUPPRIME et remplace par
      l'assertion de compilation `var _ filmsource.Source = (*Source)(nil)`. `sync/comeback.go:98`
      et `replaybuild/zones.go` n'utilisent que `ObjectiveTypeOf` : intacts.
- [ ] 1.6 `sync/killcollector/positions.go:170-186` : un `Film` charge une fois depuis les chunks
      en memoire, les quatre balayages le recoivent (`:170`, `:175` `ScanFilmClockOrigin`, `:181`
      `ScanFilmDeaths`, `:186` `ScanFilmPlayerIndices`). `cmd/zone-attribution/measure.go:90,234`,
      `cmd/statnames-sweep/sweep.go:99` et `cmd/oddball-terrain/decode.go:64` (`ScanFilmDeaths(dir)`
      juste sous leur `StatRecordsCtx`) : charge le film UNE fois puis `BuildFromFilm(film)` /
      `ScanDeaths(film)` — jamais un `*Film` plus une enveloppe `dir` dans le meme CLI.
- [ ] 1.7 Enveloppes D2 : inventaire par grep des appelants `ScanFilm*(dir` restants (tests de
      recherche, `cmd/tmp_*`, `cmd/*` non production) ; enveloppe conservee la ou un appelant
      existe, supprimee sinon.
- [ ] 1.8 Test structurel « zero disque » : `BuildFromFilm` sur un `Film` charge par
      `Load(MemoryChunks)` depuis la mini-bobine, avec `opt.MapQuant` fourni (entree Cliffhanger du
      catalogue, comme `golden_assembly_test.go:87`) — tout acces disque residuel echoue le test
      (le repertoire n'existe pas).
- [ ] 1.9 Garde-rails `archlint` (allowlists datees) : `zlib.NewReader` — liste FERMEE :
      `filmsource/film.go` · `analysis/highlight_event_parser.go:107` (parseur autonome appele par
      `sync/collect.go:111`, `engine_highlight_events.go:203`, `convergence_backfill_events.go:281`
      sur des blobs bruts ou zlib) · `sync/haloclient/halo_client_http.go:59` et
      `cmd/replay-worker/job.go:342` (validation au telechargement) · `hinavmesh/conteneur.go:65`
      (autre domaine) · `cmd/fetch_film_chunks/main.go:233`, `cmd/diag_weapons_v3/positions.go:90`,
      `cmd/rdata_weapon_scan/main.go:33` (recherche) — SEPT sites non-test survivants (les trois
      de la chaine de cuisson, `filmdec/film_packets.go:57`, `filmdec/registry.go:104`,
      `objectiveevents/film.go:100`, `killsource/chunks.go:90`, disparaissent au lot 1) ; le
      garde-rail EXCLUT les `_test.go` (meme filtre que `no_unbounded_film_loop_test.go:94` ;
      `minifilm_test.go` n'appelle d'ailleurs que `zlib.NewWriterLevel`) ; `os.ReadFile` dans
      `filmdec/*.go` hors chargeurs de catalogue (`map_bounds.go`) ; `ScanFilm[A-Za-z]*\(dir`
      interdit dans les sites de production de D2 ; `filmsource` n'importe rien du depot.
- [ ] 1.10 Ratchet : `internal/archlint/filmdec_package_vars_test.go` compte les declarations
      `var` de niveau paquet dans `filmdec/*.go` non-test (compte gele a la valeur mesuree au
      moment du lot, justification datee) — le compte ne doit pas croitre (D10).
- Gate 1 : `go run ./cmd/replay-equiv -corpus ...` IDENTIQUE (11 films, toutes etapes, artefact) ;
  `go test ./internal/analysis/... ./internal/replaybuild/ ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/ ./internal/archlint/ ./cmd/...`
  vert ; `make go-api-lint` sans nouvelle dette ; mesure des 4 temoins (§6) au journal.

### Lot 2 — Contexte du film partage — effort moyen

- [ ] 2.1 `filmdec.FilmContext{BipedSlots, Layout, BipedArch, EquipmentArch, GroundWeaponArch,
      Registry}` construit UNE fois dans `BuildFromFilm` (layout : AUTO-DETECTE — D3bis) ; les six
      balayages delta (`held_weapon_changes.go:204`, `inventory_delta.go:196`, `ability_rank.go:123`
      partage par `equipment_changes.go:129`, `camo_state.go:92`, `grapple_state.go:89`) et
      `biped_pickups.go:163` le recoivent ; `bipedSlotBand`/`DetectI0Layout`/`bipedArchetype` n'y
      sont plus appeles.
- [ ] 2.2 Double `worldObjectSlotBand` des poses (`equipment_placements.go:126` vs
      `projectiles.go:115`) : une seule bande, passee (`...ForBand`).
- [ ] 2.3 Registre chunk_00 parse une fois (`FilmContext.Registry`) ; les ~10 re-parses
      (`bipedArchetype`, `EquipmentArchetype`, `groundWeaponArchetype`,
      `managedPropertyArchetype`, `filmArchetype`) le consomment.
- [ ] 2.4 Garde-rail : `DetectI0Layout(`, `bipedSlotBand(`, `ParseRegistryChunk(` appeles
      uniquement depuis le constructeur de contexte (allowlist).
- Gate 2 : equivalence IDENTIQUE ; tests des paquets touches verts ; lint ; mesure §6 au journal.

### Lot 3 — Correction Live Fire : le layout du catalogue pour les six balayages — effort rapide

- [ ] 3.1 `FilmContext.Layout` = `opt.MapQuant.Layout()` quand valide, auto-detection en repli —
      la regle EXACTE de `build.go:257-259`, ecrite une fois pour les positions ET les six
      balayages.
- [ ] 3.2 Equivalence : identique sur les 10 films non-Live Fire ; sur `60ae07c4` les digests
      CHANGENT — `-update -films 60ae07c4`, journal avec le diff des comptes par balayage, et une
      cuisson `replay-build` de `60ae07c4` dont le journal ne porte plus « decoupage i0 illisible ».
- [ ] 3.3 Test unitaire de la regle sur des entrees du VRAI catalogue (Live Fire : gate 6,
      region 1 ; Cliffhanger : gate 5) et une entree sans largeurs (repli).
- Gate 3 : equivalence IDENTIQUE hors `60ae07c4` ; `60ae07c4.tsv` regenere avec diff au journal ;
  tests verts.

### Lot 4 — Boucles chaudes (refacto pur) — effort moyen

- [ ] 4.1 Bande de slots en tableau indexe (`bipedSlotBits = 13` → 8 192 entrees) a la place de
      `map[uint32]bool` dans `matchBipedHeaderRaw` (`offline_biped.go:332`) et ses repliques
      (`ability_rank.go:159-162`, `camo_state.go:139-142`, `grapple_state.go:131-134`,
      `held_weapon_changes.go:120-123`, `i0_layout.go:194-197`, `biped_pickups`).
- [ ] 4.2 `readBitsAt` et `BitReader.ReadBits` par mot ; test differentiel D6 (n = 0..64,
      frontieres, semantique hors tampon preservee par fonction).
- [ ] 4.3 `offline_aim.go:248,263` : un lecteur reutilise par `SetBitPos` ; `ascendingFromZero`
      (`offline_biped.go:359-364`) valide AVANT d'allouer.
- [ ] 4.4 `objectiveevents.NamedEventsFrom` : un seul balayage de `recs` regroupe par cle a la
      place des ~22 `rawSeriesByRound`/`RealRounds` (`named.go:207-212`) ; la cle de tri
      (`named.go:224-235`, aujourd'hui `TimeMS, Slot, Stat`) gagne `Comp` et `Side` — la sortie
      n'est deterministe qu'accidentellement (iteration de map + `Stat` unique par famille) et un
      second emplacement non redondant casserait tous les digests sans test pour le dire.
      Identique sur les 11 films (les faits du corpus rendent ce gate NON vacuant, cf. 0.2).
- [ ] 4.5 Mesure §6 des 4 temoins + profil CPU de `01e1f945` au journal, comparee a la reference.
- Gate 4 : equivalence IDENTIQUE (11 films) ; tests differentiels verts ; lint ; mesure au
  journal ; verdict §1.2 ecrit (atteint / ecart statue) — le lot 5 s'execute dans les deux cas.

### Lot 4b — Films-bombes : plafond de prudence (correction declaree) — effort rapide

- [ ] 4b.1 Mesure du deroulage maximal reel `p.Value - prev` (PREMIER TERME COMPRIS, `prev`
      partant de zero) sur les 11 films sains, par comp et cote — si > 1 000 : escalade avant
      d'aller plus loin (D13).
- [ ] 4b.2 Plafond D13 : rejet du deroulage > 10 000 dans `incrementTimes` ; plafond total
      1 000 000 par film dans `NamedEventsFrom` (pas dans `incrementTimes`) ; compteurs de rejet
      journalises ; tests unitaires sur des series construites (premier terme enorme ; saut
      intermediaire ; total depasse).
- [ ] 4b.3 `51101d1d` et `a349fea8` cuisent sous 3 Gio via `replay-equiv` (pic + duree au
      journal) ; premier digest fige (`-update -films ...`), entree au `CORPUS.txt` « fige au lot
      4b » ; les 11 films sains restent IDENTIQUES.
- [ ] 4b.4 `REGISTRE_REPORTS.md` (bombe RAM `NamedEventsFrom`) : cause confirmee ou infirmee par
      le profil heap (`-memprofile` de l'enfant) pris au 4b.3, avec le chiffre.
- Gate 4b : equivalence IDENTIQUE sur les 11 films sains ; les deux bombes cuisent sous plafond ;
  tests verts.

### Lot 5 — Orchestration et protections — effort moyen

- [ ] 5.1 Ouvrier : telechargement des chunks en parallele borne (8), sur le modele de
      `haloclient/halo_client_film.go:241-267` ; test `httptest` : 30 chunks, ordre restitue, une
      erreur = echec du job, aucune goroutine qui fuit.
- [ ] 5.2 Ouvrier : `BuildBytes` + envoi des octets, plus d'ecriture ni de relecture locale (D8) ;
      test : aucun fichier ecrit sous `replays/`.
- [ ] 5.3 `replaybuild.ArtifactDigest(path)` exporte ; `etatArtefact` (`artifacts.go:260-265`) et
      `cmd_backfill_replay_repair.go:73-76` lisent UNE fois ; `ArtifactUpToDate`/
      `ArtifactHasPlayerCounters` deviennent des vues du digest ; test : une ouverture par artefact.
- [ ] 5.4 Backfill : `runnerEnfant` (`cmd/levelup/backfill_child.go`) migre sur `filmproc.Runner`
      (priorite basse + fin du doublon codes/marqueur/relais) ; test du runner etendu.
- [ ] 5.5 Post-sync : deadline par enfant = min(budget de cycle restant, 15 min) autour de
      `BuildOne` (`cuisson.go:175`) ; test : un `BuildOne` bloquant est coupe, compte en `echecs`,
      le cycle continue.
- [ ] 5.6 Post-sync : prechargement du film N+1 (profondeur 1) pendant la cuisson de N, abandonne
      si le budget est epuise ; test avec doubles (ordre des ecritures, budget respecte).
- [ ] 5.7 Verrou solo (D7) : CABLAGE de l'existant (cree au 0.4) — `replaychild.Spawn` prend
      `AcquireSolo` (immediat), l'enfant de backfill et l'ouvrier prennent `AcquireSoloWait`
      (attente 10 min) ; allowlist du ratchet mise a jour (regime de chaque site) ; tests : le
      post-sync refuse et compte en `echecs` quand le verrou est tenu ; l'enfant de backfill
      attend puis passe.
- Gate 5 : `go test ./internal/sync/... ./internal/filmproc/ ./internal/replaychild/ ./internal/replaybuild/ ./cmd/replay-worker/... ./cmd/levelup/...`
  vert ; `go test -tags=integration ./...` vert (obligatoire : `internal/sync` touche) ;
  `go test ./internal/archlint/` vert ; lint.

### Lot 6 — Cloture — effort rapide

- [ ] 6.1 Mesure finale §6 + profil, tableau reference/final dans `MESURES_CUISSON_PERF.md` ;
      verdict §1.2 ecrit.
- [ ] 6.2 Docs : `docs/COMMANDS.md` (FR+EN : `replay-equiv`, `replay-facts-export`,
      `-cpuprofile`) ; `AUDIT_CUISSON_REPLAY_PERF_2026-09-02.md` : colonne « Decision » renseignee.
- [ ] 6.3 `delivery-checklist` ; `adversarial-review` du diff complet (lots a risque : 1 et 5).
- [ ] 6.4 Proposition de merge dans `feat/v75` (pas de push sans accord ; la re-cuisson de masse
      n'est PAS dans ce plan : elle suit le schema vehicules).
- Gate 6 : `make gate-push` vert.

## 5. Tests — ce qui verrouille quoi

| Ce qu'on protege | Test | Regime |
|---|---|---|
| Sortie identique (refacto pur), faits compris | `cmd/replay-equiv` (11 films, un processus borne par film, digests par etape + artefact) | local ; gate de chaque lot |
| Sortie identique (CI) | `TestEquivalenceMiniFilm` (liste fermee de balayages) + goldens existants (`GoldenAssembly`, `GoldenInputs`) | toujours |
| Liste des etapes observees | test unitaire de l'observateur (0.3) | toujours |
| Decodage unique / aucune E/S dans les balayages | test « zero disque » (1.8) + garde-rails (1.9, 2.4) | toujours |
| Grammaire de decoupage | tests unitaires `filmsource` sur chunks construits (1.1) + mesure 0.7 | toujours / lot 0 |
| Primitives reecrites | tests differentiels D6 (domaine 0..64) | toujours |
| Correction Live Fire | test de la regle (3.3) + diff des comptes au journal | toujours + local |
| Orchestration | tests unitaires par item du lot 5, `-tags=integration` | toujours / gate 5 |
| Etat global de filmdec | ratchet 1.10 | toujours |
| Vitesse | mesure protocolisee (§6), pas un test | journal |

Ce qu'on NE fait pas : tests unitaires de deserialiseurs sur des motifs de bits fabriques ; tests
de temps en CI ; copie de la sequence des balayages dans un harnais (l'observateur vit dans le
code de production).

## 6. Protocole de mesure (0.8, repete aux lots 1, 2, 4, 6)

```
cd apps/go-api && go build -o ../../tmp/replay-build.exe ./cmd/replay-build
LEVELUP_LOG_LEVEL=debug tmp/replay-build.exe --map "<carte du CORPUS.txt>" --facts internal/analysis/replay/testdata/equivalence/<f>.facts.json --cpuprofile tmp/<f>.prof <f> data/cache/film_chunks/<f>
```
pour `f` dans `01e1f945`, `7344d24f`, `696a9d7c`, `1c4c63c2` — la carte vient de `CORPUS.txt`
(rempli par 0.2), jamais devinee. Releve : duree totale, duree par balayage (Debug, a partir de
0.6), pic (sentinelle). Machine et date au tableau. Un seul decodage a la fois (verrou solo). La
cuisson ecrit l'artefact a sa place canonique (jonction vers le cache partage) au schema courant :
identique a celui en place hors lots de correction, ou le garde anti-regression fait foi.

## 7. Hors perimetre (explicite — ne pas traiter, noter en Decouvertes si rencontre)

- Parallelisme intra-decodeur (par chunk ou entre balayages) : REFUTE par l'audit.
- De-globalisation de `filmdec` (>= 80 vars, 28 hooks) ; deplacement de `filmdec` sous `games/`.
- Allegement de `validateArtifact` (R5) — decision utilisateur.
- Deux enfants backfill simultanes — nouvelle decision apres mesure du lot 6.
- `killsource.locateStrict`/`World.Snapshot` (D14, sans condition).
- Les deux marcheurs de paquets HORS chaine de cuisson (`analysis/weaponv3/timing.go:33-56`,
  `analysis/positions/positions.go:124-141`, ce dernier en production cote serveur pour la vue de
  match) : non migres — un chantier separe s'ils doivent rejoindre `filmsource`.
- Enrichissement de la mini-bobine (chunk_00, manifeste) — la CI ne couvre pas le registre ni les
  positions, c'est ecrit en D4c ; a rouvrir hors plan si le corpus local ne suffit plus.
- Re-cuisson de masse du cache ; schema vehicules.

## 8. Decouvertes (a noter, ne pas traiter)

_(vide a la creation)_

## 9. Revue du plan (plan-review, 2026-09-02)

Relecteur frais, grille `plan-review` + verification sur pieces. Verdict initial : « a refaire
partiellement » (§2, D1, D3, D4, lots 0 et 1). Constats et traitement :

| # | Constat | Traitement dans cette version |
|---|---|---|
| 1, 2, 21 | Cycle d'import `filmdec -> filmcache -> objectiveevents -> filmdec` ; cinq tests internes de `filmdec` importent `objectiveevents`/`filmcache` ; premier import `analysis/ -> games/` | D1 reecrite : paquet feuille `filmsource` sans import ; `filmdec` n'importe rien de plus |
| 3 | Supprimer `FilmSource`/`ChunkMeta` casse 6 sites + `filmcache_guard_test.go` | item 1.5 : liste fermee des sites et du garde-rail |
| 4 | `BuildFromFilm` casse `cmd/zone-attribution/measure.go:90,234` ; `registry_replay_build.go:66` | items 1.3 (signatures `BuildBytes`/`BuildMatch` inchangees) et 1.6 |
| 5 | `killsource.Decode` : appelants `killcollector/collector.go:284`, `cmd/killsource/main.go:206` | item 1.4 |
| 6 | Faits vides = equivalence vacuante (zones, objectifs, VIP/crane/bombe, socles) | item 0.2 : export des faits par film ; D4a observe chaque sortie de `readFilmStats` |
| 7 | Mini-bobine : pas de chunk_00/manifeste, `BuildFromFilm` refuse sans `MapQuant` | D4c : etage CI = digests des balayages supportes, liste fermee ; limites ecrites |
| 8 | Trois marcheurs divergents (size 0, CHUNK_END, flux tronque) : le lot 1 n'est pas pur par construction | D3 : grammaire unique ecrite + mesure 0.7 sur 1 380 films ; prerequis du lot 1 ; divergence sur le corpus = escalade |
| 9 | Harnais enchainant 12 films dans un processus de test non borne | D4b : `cmd/replay-equiv`, un enfant borne par film, `filmproc.Runner` + sentinelle + verrou, declare au ratchet |
| 10 | Allowlist `zlib` incomplete | item 1.9 : liste fermee (10 sites) |
| 11 | Item 2.4 inapplicable (`FindPackets` boucle sur des films) et ecarte par l'audit | retire ; `zoom_events.go:95` absorbe par 1.2 |
| 12 | 17 appels (pas 22) ; 6 sites hors plage + `killcollector/positions.go` | D2 et items 1.2, 1.6 : inventaire exhaustif |
| 13 | `084a804d` n'est pas le plus gros (max = `1c4c63c2`, 89 Mo) | 0.1 : `1c4c63c2` ajoute (11 films) |
| 14 | Regle de selection non fermee | 0.1 : short8 nommes |
| 15 | Borne du plafond `incrementTimes` non ecrite | D13 : 10 000 / 1 000 000 + mesure prealable |
| 16 | §7 laissait une porte « si > 15 % » | D14 : hors perimetre sans condition |
| 17 | Test differentiel limite a 1..32 ; `readBitsAt` sans garde vs `ReadBits` rend 0 | D6 : domaine 0..64, semantique hors tampon preservee par fonction |
| 18 | `res.Dur` ne peut pas atteindre `cuisson.go` sans changer `Spawn` | D5 : `Spawn` rend `Built{Blob, Dur, Peak}` |
| 19 | Ratchet des `var` promis sans item | item 1.10 |
| 20 | `--map` sans nom de carte | §6 : carte lue dans `CORPUS.txt`, rempli par l'export 0.2 |
| — | Pas de section Blockers | §9bis ci-dessous |

Points de la grille satisfaits (inchanges) : structure et criteres mesurables, branche verifiee
(`wt/cuisson-perf` = `900384f50`), multi-titre (aucun `filepath.Join` sur `data/`, `no_data_path_join_test`
ne rougira pas), determinisme de `BuildBytes` (aucun `time.Now`/`rand` hors tests, `GeneratedAt`
lus des catalogues), sites des lots 2-5 exacts sur pieces, logging (`"duration"` = convention du
depot), done-definition par lot, renvoi a `plan-execution`.

### Seconde lecture (2026-09-02, relecteur frais, points bloquants corriges) — verdict :
« executable apres corrections listees ». D1 demontre sans cycle (`go list -deps`), D4a/D4b
ancres sur des mecanismes verifies, inventaires des `ScanFilm*` exacts au site pres, 0.2
executable. Constats et traitement :

| # | Constat | Traitement |
|---|---|---|
| BL-1 | 0.4 exigeait `AcquireSoloWait` que 5.7 creait : gate 0 ne compilait pas | D7 et 0.4 : cree au lot 0 ; 5.7 ne fait que cabler |
| BL-2 | « liste fermee » de 1.5 : 3 sites sur 9 points d'entree `FilmSource`, ~57 appelants | 1.5 reecrit : 9 fonctions, 7 sites de production nommes, producteurs de source, grep de reference, garde-rail `filmcache` remplace par une assertion de compilation |
| BL-3 | D13 bornait l'ecart entre echantillons ; la grandeur qui explose est `p.Value - prev` premier terme compris ; un total dans `incrementTimes` serait par appel | D13, 4b.1, 4b.2 reecrits |
| R-1 | Cinq marcheurs, pas trois (`weaponv3/timing.go`, `positions/positions.go`) | D3 et §7 : nommes, hors chaine de cuisson, hors perimetre |
| R-2 | Quatrieme axe de divergence (borne haute sans offset + brut compresse marche) | D3 et 0.7 |
| R-3 | `minifilm_test.go` n'appelle pas `zlib.NewReader` ; tests non listes | 1.9 : liste des 7 sites non-test, `_test.go` exclus |
| R-4 | « analysis n'importe rien de games » est faux (`games/canonical`) | §2 corrige : seul le cycle de test est l'argument |
| R-5 | `Runner` ne rend pas stdout ; stdout/stderr fusionnes | D4b : l'enfant ecrit son fichier de digests |
| R-6 | Garde-rail `filmcache` ancre sur `Chunks()` + allowlist a justification caduque | 1.5 : supprime, assertion de compilation |
| R-7 | Appels `matchfacts.go:83,170` non cites ; `ScanFilmDeaths(dir)` sous 2 CLI | 1.5, 1.6 |
| R-8 | Tri de `NamedEventsFrom` sans `Comp`/`Side` : determinisme accidentel | 4.4 |
| R-9 | NaN/Inf font echouer `json.Marshal` ; champs non exportes invisibles | D4a |
| R-10 | 0.2 recreait les requetes de `selectBuildWork` | 0.2 : `mapNamesForOne` |
| R-11 | Films hors corpus divergents = changements de sortie non mesures | D3 : escalade explicite avant le lot 1, recommandation ecrite |

Points qui tiennent (seconde lecture) : D1 sans cycle, CHUNK_END = type 7 confirme par trois
sources RE et deux marcheurs independants, `size == 0` presente comme hypothese a mesurer,
observateur compatible (`Options` deja non comparable, aucun `DeepEqual`/`Marshal` sur elle),
`Runner` re-execute le meme binaire, ratchets satisfiables, 0.2 sans cycle ni import de `sync`,
inventaires 1.2-1.6 exacts, D6 exact (`readBitsAt` rend `uint32`, domaine 0..64 = `ReadBits`
seul), 10 000 coherent comme dernier rempart.

### 9bis. Blockers connus

- L'export des faits (0.2) lit la base partagee : `OpenReadForQuery` est la voie autorisee meme si
  le serveur local la tient ; si l'ouverture echoue, arreter le serveur local le temps de l'export
  (13 lectures courtes), jamais forcer un `OpenReadOnly`.
- Le corpus exige le cache local (jonctions en place) ; sans lui, seul l'etage CI (D4c) tourne.
- Le verdict 0.7 conditionne le lot 1 : une divergence sur le corpus arrete le plan jusqu'a la
  decision utilisateur (D3).

## 10. Journal

- 2026-09-02 — Plan cree depuis l'audit. Worktree et jonctions du cache en place ; registre
  d'audit deplace dans ce worktree. Aucun code modifie.
- 2026-09-02 — Revue adversariale (§9) : 9 bloquants, 12 recevables. Plan REECRIT : paquet
  feuille `filmsource` (D1), grammaire unique mesuree (D3, item 0.7), faits exportes par film
  (0.2), observateur en production (D4a), harnais un-film-par-processus (D4b), etage CI borne a la
  mini-bobine (D4c), inventaires exhaustifs (D2, 1.2-1.6), bornes ecrites (D13), portes fermees
  (D14).
- 2026-09-02 — Seconde lecture (§9) : « executable apres corrections » — 3 bloquants (ordre de
  creation du verrou d'attente, 9 points d'entree `FilmSource` et non 3, grandeur du plafond) et
  11 recevables, tous integres. Lecon de methode consignee : une liste dite « fermee » se
  verifie par un grep dont la commande est ecrite dans l'item. Decisions soumises a
  l'utilisateur avant execution : D7 (verrou a deux regimes), D8 (l'ouvrier n'ecrit plus en
  local), D9 (pas d'allegement de `validateArtifact`), D13 (bornes 10 000 / 1 000 000), R-11
  (films hors corpus divergents acceptes comme corrections). Prochaine etape : lot 0.
