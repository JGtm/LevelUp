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
- **D3 — REVISION du 2026-09-02 sur la mesure 0.7 (1 378 films) et le diagnostic paquet par paquet
  (3 films, tous chunks).** FAITS : sur les chunks de DONNEES (01..N), l'unique paquet de taille 0
  est le marqueur CHUNK_END (type 7), en DERNIERE position, sans un octet apres — 27/27, 32/32,
  43/43 chunks sur `000d5950`, `7344d24f`, `60ae07c4`, et 1 378/1 378 films au meme motif ;
  `killsource` ne diverge jamais parce que sa regle `size <= 0` attrape ce terminateur ; aucun flux
  zlib tronque dans tout le cache ; `chunk_00` (le REGISTRE, pas un flux de paquets) porte 13-14
  en-tetes degeneres `size = 0` au MILIEU (types 0, et un type 29806 sur `60ae07c4`) — seul chunk
  ou `objectiveevents` diverge, et aucun consommateur legitime ne le marche comme des paquets.
  GRAMMAIRE RETENUE (remplace « arret sur size <= 0 ») : (1) en-tete de 16 octets, arret si
  `off+16+size > len` ; (2) le paquet est EMIS, taille 0 comprise ; (3) arret APRES un paquet de
  type 7 (le terminateur est emis, comme `filmdec` — qui ne trouvait rien apres — et filtre par
  type chez `killsource`) ; (4) arret AVANT EMISSION sur un paquet de taille 0 qui n'est PAS de type 7 — le paquet degenere n'est pas emis, comme chez killsource (en-tete
  degenere : `chunk_00` seulement, ou `killsource` s'arrete deja la). Effets attendus, A PROUVER
  par les digests des 11 films au lot 1 : `filmdec` identique sur tout chunk de donnees ;
  `killsource` voit un paquet de type 7 de plus par chunk (filtre par type en aval) et le meme
  `chunk_00` ; `objectiveevents` perd des trames VIDES de `chunk_00` (aucun record dedans). La
  regle d'ARRET de D3 reste : un digest du corpus qui bouge au lot 1 = arret et escalade.
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
  differe ; chaque TSV porte en tete `# digest-grammar: N` (`digest.GrammarVersion`), verifie avant toute comparaison — une version differente est une erreur d'INFRASTRUCTURE (re-figer par -update), jamais un ecart d'etape. Declare au ratchet `no_unbounded_film_loop_test.go` (« enfant borne, un film par
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

- [x] 0.1 Corpus FERME (verifie present au cache le 2026-09-02, chunks / Mo compresses) :
      `000d5950` Cliffhanger Fiesta (28 / 20,2) · `01e1f945` Catalyst KOTH (30 / 22,5) · `64e8adfa`
      CTF (45 / 33,7) · `7344d24f` Strongholds (33 / 25,8) · `696a9d7c` Strongholds (31 / 26,6) ·
      `084a804d` CTF, temoin historique 19 min (57 / 58,9) · `1c4c63c2` LE PLUS GROS film du cache
      (69 chunks / 89) · `53ce4390` CTF, ancre historique des bursts (41 / 31,8) · `d9781168`
      Oddball (39 / 29,8) · `9f57c612` Assaut One Bomb, temoin `navpoint_ti12` (27 / 19,8) ·
      `60ae07c4` Live Fire Ranked (44 / 32,0 — ecart attendu au lot 3 seulement). Total : 11 films.
      `BOMBES.txt` : `51101d1d` (13 / 9,1), `a349fea8` (51 / 65,9). Fichier
      `apps/go-api/internal/analysis/replay/testdata/equivalence/CORPUS.txt`, colonnes : short8,
      carte(s) candidate(s), variante, chunks, Mo, raison — cartes et variante REMPLIES PAR 0.2.
- [x] 0.2 Export des faits : `levelup replay-facts-export --out <dir> <short8...>` — lit, par
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
- [x] 0.3 Observateur (D4a) : `replay.Options.Observe` + appels dans `BuildFromFilm` apres chaque
      balayage, et dans `BuildBytes` pour chaque sortie de `readFilmStats`, `ksRes`, catalogues,
      blob. Test unitaire : sur la mini-bobine, la liste des etapes observees est exactement la
      liste FERMEE attendue (un balayage ajoute sans etape observee fait echouer le test).
- [x] 0.4 `filmproc.AcquireSoloWait(ctx, cacheRoot, tool, id, max)` (D7) dans `solo.go`, avec
      son test (deux acquisitions concurrentes sur la meme racine : la seconde attend, puis passe
      quand la premiere relache ; depasse `max`, elle refuse). PUIS `cmd/replay-equiv` (D4b) :
      parent + enfant (meme binaire, drapeau interne), `filmproc.Runner`, `AcquireSoloWait`,
      sentinelle `filmproc.Arm` ; `-update`, `-corpus`, `-films`, `-out` (enfant) ; sortie du
      parent : premiere etape qui differe, par film. Declare au ratchet avec sa justification
      datee. `TestEquivalenceMiniFilm` (D4c).
- [x] 0.4bis (ajoute le 2026-09-02, correction declaree) — NON-DETERMINISME du decodeur, revele par le
      harnais avant tout figeage : quatre sites batissaient une tranche en iterant une map puis la
      triaient sur une cle NON totale (ou rendaient le premier trouve) — `filmdec/projectiles.go`
      `lessTrack` (naissances des objets du monde), `filmdec/equipment_placements.go`
      `confirmPlacements`, `objectiveevents/slotidentity_rounds.go` `roundStartsOf`,
      `replay/ground_weapon_rules.go` `gwPadsClass` (le `dropper` publie d'une arme au sol etait
      TIRE AU SORT parmi les ex aequo : `groundWeapons[95-97].dropper` = 553 ou 556 selon
      l'execution sur `000d5950`). Ordre rendu TOTAL par des donnees de la piste (jamais l'adresse
      ni l'ordre d'iteration), test `filmdec/projectile_track_order_test.go` (4 ordres d'entree,
      1 sortie). Seul l'ordre des ex aequo change, et il etait aleatoire — aucune ancienne verite a
      preserver. Neuf sites de la meme famille a l`ASSEMBLAGE sont reperes et NON touches (§8) :
      le figeage du corpus dira s'ils bougent.
- [x] 0.5 Figer les digests a `900384f50` : `-update` sur les 11 films, puis UNE SECONDE
      EXECUTION sans `-update` qui doit passer — preuve de DETERMINISME du harnais (une iteration
      de map non triee residuelle se verrait ici). Les deux sorties au journal.
- [x] 0.6 Instrumentation (D5) : durees par balayage/phase ; `replaychild.Spawn` → `Built{Blob,
      Dur, Peak}` et `BuildOneFunc` ; log de succes post-sync avec `duration` ; `cmd/replay-build` :
      `logging.InstallCLILevel(repoRoot, niveau(LEVELUP_LOG_LEVEL))`, duree totale, `-cpuprofile`.
- [x] 0.7 Mesure de divergence des grammaires (D3) : `cmd/replay-equiv -walkers` — pour CHAQUE
      film du cache (1 380), inflate une fois par chunk, puis compare, chunk par chunk, le jeu de
      paquets (type, ts, taille) vu par les trois marcheurs de la chaine de cuisson et par la
      grammaire retenue — sur les QUATRE axes de D3 : `size == 0`, `CHUNK_END`, borne haute avec
      ou sans offset (et avance sur debordement), flux zlib tronque (brut compresse marche comme
      des paquets vs partiel). Un film par processus borne. Sortie :
      `.ai/V7.5/MESURES_CUISSON_PERF.md` §divergence — nombre de films sans divergence, liste des
      films divergents avec l'axe et le consommateur touches, nombre de films a flux tronque.
      Verdict ecrit : le lot 1 est-il un refacto pur sur le corpus (D3) ? La liste des films
      divergents hors corpus est soumise a l'utilisateur (D3, R-11).
- [x] 0.8 Mesure de reference HEAD (§6) sur `01e1f945`, `7344d24f`, `696a9d7c`, `1c4c63c2` :
      duree totale, duree par balayage, pic, profil CPU de `01e1f945` →
      `MESURES_CUISSON_PERF.md` §reference.
- [x] 0.9 Note dans `.ai/V7.5/replay2d/registre_film/LOTCTER_VOLET3.md` : les pics RAM de
      `cout_machine.tsv` mesurent le processus `go` lanceur, pas le decodeur (audit C6).
- Gate 0 : `cd apps/go-api && go test ./internal/analysis/replay/ ./internal/analysis/filmdec/ ./internal/replaybuild/ ./internal/sync/replayartifacts/ ./internal/replaychild/ ./internal/archlint/ ./cmd/levelup/... ./cmd/replay-equiv/...`
  vert ; `go run ./cmd/replay-equiv -corpus internal/analysis/replay/testdata/equivalence/CORPUS.txt`
  vert deux fois de suite (0.5) ; `MESURES_CUISSON_PERF.md` porte reference + divergence ;
  `make go-api-lint` sans nouvelle dette.

### Lot 1 — Source unique du film : decoder une fois — effort lourd

Prerequis : verdict 0.7 = zero divergence sur les 11 films du corpus (sinon : arret, escalade).

- [x] 1.1 `internal/analysis/filmsource` (D1) : `source.go` (Source, MemoryChunks, DirSource),
      `film.go` (ChunkMeta, Packet, Film, Load, LoadDir, inflate, framing D3), tests unitaires sur
      des chunks CONSTRUITS (zlib d'octets connus : un paquet, deux paquets, size 0, CHUNK_END,
      flux tronque — valeurs calculables, pas arbitraires) + test « aucun import du depot »
      (archlint).
- [x] 1.2 `filmdec` : `FilmPacket`/`WalkPackets`/`ReadFilmChunk`/`CountFilmChunks`/`inflateChunk`
      remplaces par `filmsource` ; `ParseRegistryChunk` recoit des octets deja decompresses
      (`registry.go:104` perd son inflate) ; les 17 `ScanFilm*` de `build.go:264-475` et les
      balayages hors plage (`bomb_armings.go:113`, `build_ground_weapons.go:97,109`,
      `build_objectives_live.go:91`, `build_zones.go:49`, `equipment_placements.go:318`) obtiennent
      leur forme `Scan*(film, ...)` ; `replay.BuildFromFilm(matchID, title, film *filmsource.Film,
      opt)` ; `deaths_source`, `origin`, `player_index`, `inventory_decode` idem ; le chunk
      highlight se lit UNE fois (`kills.go:67`, `matchfacts.go:196`, `build.go:449`).
      ECART STATUE (2026-09-02, consigne du pilote) : le chunk highlight n'est plus DECOMPRESSE
      qu'une fois, mais encore PARSE deux fois — une lecture partagee par `kills.go` et
      `matchfacts.go`, et celle de `BuildFromFilm`, qui RESTE pour ne pas deplacer l'etape
      observee `deaths` (note N-R du §8).
- [x] 1.3 `replaybuild.BuildBytes`/`BuildMatch` gardent leur signature (`filmDir string`) et
      chargent le film UNE fois (`filmsource.LoadDir(filmDir, meta)` avec `meta` du manifeste via
      `filmcache.OpenChunkDir`, deja ouvert en `matchfacts.go:72`) ; `readFilmStats`,
      `decodeKillSource`, `killRefs`, `BuildFromFilm` recoivent le `*Film`. Appelants inchanges :
      `api/wire/registry_replay_build.go:66`, `cmd/replay-build`, `cmd_backfill_replay_child.go`,
      `replaychild`, `replay-worker`.
- [x] 1.4 `killsource.Decode(ctx, matchID, film *filmsource.Film, ...)` ; `inflate`,
      `splitPackets`, `dirChunks`, `ChunkSource`, `MemoryChunks` supprimes ; `loadFilm` ne fait
      plus que trier `t0` depuis `film.Packets`. Appelants migres : `replaybuild/kills.go:33`,
      `sync/killcollector/collector.go:284` (+ `bridge.go:100` : `filmsource.MemoryChunks` et
      `ChunkMeta` construits depuis les `haloclient.FilmChunk`), `cmd/killsource/main.go:206`.
- [x] 1.5 `objectiveevents` : les NEUF points d'entree qui prennent un `FilmSource` passent a
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
- [x] 1.6 `sync/killcollector/positions.go:170-186` : un `Film` charge une fois depuis les chunks
      en memoire, les quatre balayages le recoivent (`:170`, `:175` `ScanFilmClockOrigin`, `:181`
      `ScanFilmDeaths`, `:186` `ScanFilmPlayerIndices`). `cmd/zone-attribution/measure.go:90,234`,
      `cmd/statnames-sweep/sweep.go:99` et `cmd/oddball-terrain/decode.go:64` (`ScanFilmDeaths(dir)`
      juste sous leur `StatRecordsCtx`) : charge le film UNE fois puis `BuildFromFilm(film)` /
      `ScanDeaths(film)` — jamais un `*Film` plus une enveloppe `dir` dans le meme CLI.
- [x] 1.7 Enveloppes D2 : inventaire par grep des appelants `ScanFilm*(dir` restants (tests de
      recherche, `cmd/tmp_*`, `cmd/*` non production) ; enveloppe conservee la ou un appelant
      existe, supprimee sinon.
      INVENTAIRE DU 2026-09-03 (tableau complet au §10) : 40 formes `dir` — 39 marquees
      « ENVELOPPE D2 » dans leur commentaire + `GroundWeaponPositions` (qui delegue aux deux
      autres, marquee depuis) ; CHACUNE a au moins un appelant, TOUS dans des `_test.go` de
      `analysis/filmdec` et `analysis/replay` (plus trois delegations internes a `filmdec`) ;
      ZERO appelant de production (aucun dans `replaybuild`, `sync`, `api`, `cmd`) ; donc AUCUNE
      suppression. La liste fermee d'`archlint` coincide exactement avec ces 40 noms.
- [x] 1.8 Test structurel « zero disque » : `BuildFromFilm` sur un `Film` charge par
      `Load(MemoryChunks)` depuis la mini-bobine, avec `opt.MapQuant` fourni (entree Cliffhanger du
      catalogue, comme `golden_assembly_test.go:87`) — tout acces disque residuel echoue le test
      (le repertoire n'existe pas).
      FAIT : `replay/zero_disque_test.go`, deux tests. Le film vient de trois `os.ReadFile` +
      `filmsource.Load(MemoryChunks, meta{Index:1,2,3})` — jamais `LoadDir` — et le decodage
      s'execute apres `t.Chdir` vers un repertoire temporaire VIDE (restaure par le nettoyage,
      avant la suppression du repertoire). Methode en trois pieces, ecrite dans l'en-tete du
      fichier : (1) CONTROLE que la bobine n'est plus atteignable par son chemin relatif depuis
      ce repertoire ; (2) `BuildFromFilm` doit rendre l'erreur EXACTE « aucun slot biped (ti=35)
      dans les keyframes du film » — toute autre, au premier chef une erreur d'ouverture, echoue
      le test ; (3) le repertoire courant est verifie VIDE apres coup. Second test : les SEPT
      familles de la liste fermee D4c decodent avec SUCCES depuis ce meme repertoire vide, sous
      leur forme FILM — c'est ce qui couvre le pipeline au-dela du premier balayage, puisque
      `BuildFromFilm` s'arrete aux positions sur cette bobine. Limite ECRITE : un acces par
      chemin ABSOLU echapperait a la mesure (bornee par `BuildFromFilm` qui ne recoit aucun
      chemin, et par la regle 2 d'`archlint`).
- [x] 1.9 Garde-rails `archlint` (allowlists datees) : `zlib.NewReader` — liste FERMEE :
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
      ETAT FINAL (2026-09-03) : les quatre regles existent. Ajouts de cet item aux acquis des
      agents 1.2-1.5 — (a) `killsource` rejoint `filmdec`/`objectiveevents`/`replay` dans la
      regle « zlib interdit » (l'item 1.4 lui a retire son inflate) ; (b) les paquets de
      PRODUCTION de la regle des enveloppes passent de 2 a 7, au site pres de D2
      (`objectiveevents`, `killsource`, `sync/killcollector`, `api/wire`, `cmd/zone-attribution`
      manquaient) ; (c) REGLE 4 NEUVE : l'allowlist des importateurs de `compress/zlib` de tout
      `apps/go-api` est fermee et verifiee DANS LES DEUX SENS (site en trop = echec ; entree
      morte = echec). Elle compte NEUF sites et non sept : le plan avait ecrit la liste avant que
      le lot 0 ne cree `cmd/replay-equiv/walkers.go`, qui porte en copie de mesure les trois
      marcheurs historiques et leur inflate (note N-AD — a trancher au lot 6). Les trois regles
      ajoutees/etendues sont verifiees DISCRIMINANTES par violation temporaire, restauree.
- [x] 1.10 Ratchet : `internal/archlint/filmdec_package_vars_test.go` compte les declarations
      `var` de niveau paquet dans `filmdec/*.go` non-test (compte gele a la valeur mesuree au
      moment du lot, justification datee) — le compte ne doit pas croitre (D10).
      MESURE DU 2026-09-03 : 113 NOMS declares par un `var` de niveau paquet (98 declarations,
      109 specs, 0 identifiant blanc) — c'est le compte gele, et la convention (un bloc
      `var (a; b)` vaut deux) est ecrite dans l'en-tete du test. Le test echoue si le compte
      CROIT, avec le detail par fichier ; une BAISSE passe et journalise le nouveau compte a
      inscrire, pour que le resserrage reste conscient. Discriminance verifiee dans les deux
      sens (var temporaire ajoutee -> echec a 115 ; constante portee a 120 -> passe en
      annoncant 113).
- Gate 1 : `go run ./cmd/replay-equiv -corpus ...` IDENTIQUE (11 films, toutes etapes, artefact) ;
  `go test ./internal/analysis/... ./internal/replaybuild/ ./internal/games/halo_infinite/film/... ./internal/sync/killcollector/ ./internal/archlint/ ./cmd/...`
  vert ; `make go-api-lint` sans nouvelle dette ; mesure des 4 temoins (§6) au journal.

### Lot 2 — Contexte du film partage — effort moyen

- [x] 2.1 `filmdec.FilmContext` construit UNE fois dans `BuildFromFilm` (layout : AUTO-DETECTE —
      D3bis) ; les six balayages delta (`held_weapon_changes.go`, `inventory_delta.go`,
      `ability_rank.go` partage par `equipment_changes.go`, `camo_state.go`, `grapple_state.go`) et
      `biped_pickups.go` le recoivent ; `bipedSlotBand`/`DetectI0LayoutOf`/`bipedArchetype` n'y
      sont plus appeles.
      FAIT (2026-09-03) : `filmdec/film_context.go`, `NewFilmContext(film) *FilmContext` +
      accesseurs `Film/ChunkNumbers/ChunkAt/BipedSlots/I0Layout/Registry`. DEUX ECARTS ASSUMES
      par rapport a la lettre de l'item, tous deux pour l'IDENTITE (details au §10) : le
      constructeur ne rend PAS d'erreur (les trois derivations echouent sur des films legitimes
      et chaque balayage doit rejouer SON message a SON etape), et les champs sont PRIVES derriere
      des accesseurs (un layout ou un registre se lit avec l'erreur qui va avec). Memoisation
      PARESSEUSE : le premier calcul a lieu exactement la ou il avait lieu avant.
- [x] 2.2 Double `worldObjectSlotBand` des poses (`equipment_placements.go` vs `projectiles.go`) :
      une seule bande, passee (`...ForBand`).
      FAIT : `ScanEquipmentPlacements` appelle `ScanWorldObjectsForBand(film, wr, band)` au lieu de
      `ScanWorldObjects(film, wr, ti=37)`, qui relevait la MEME bande une seconde fois. Les deux
      gardes que `ScanWorldObjects` posait avant de deleguer (aucun chunk, bande vide) sont deja
      passees trois lignes plus haut : la substitution est exacte.
- [x] 2.3 Registre chunk_00 parse une fois (`FilmContext.Registry`) ; les ~10 re-parses
      (`bipedArchetype`, `EquipmentArchetype`, `groundWeaponArchetype`,
      `managedPropertyArchetype`, `filmArchetype`) le consomment.
      FAIT : les SIX accesseurs d'archetype (les cinq nommes + `objectiveArchetype`, migre pour
      que l'allowlist de 2.4 soit fermee) deviennent des methodes de `*FilmContext` et lisent
      `c.Registry()` ; `filmdec.filmRegistry` est SUPPRIME. Mesure du re-parse supprime : 10 a 12
      analyses par cuisson -> 1.
- [x] 2.4 Garde-rail : `DetectI0Layout(`, `bipedSlotBand(`, `ParseRegistryChunk(` appeles
      uniquement depuis le constructeur de contexte (allowlist).
      FAIT : `internal/archlint/no_recomputed_film_context_test.go`, DEUX regles. (1) Dans
      `filmdec` hors _test, ces trois calculs (plus l'enveloppe `DetectI0Layout(dir)`) ne
      s'appellent que depuis une allowlist FERMEE de SEPT couples `fichier/fonction -> calcul` —
      le contexte (3), la bande REDUITE que `DetectI0LayoutOf` releve pour lui-meme sur six chunks
      et son enveloppe D2 (2), et `ScanBipedPositions` (2, hors perimetre — note N-AI). Le garde
      nomme la FONCTION ENGLOBANTE, pas le fichier : un second appel ajoute dans un fichier deja
      liste echoue. (2) Hors `filmdec`, aucun paquet de production de la cuisson n'analyse le
      registre lui-meme, allowlist fermee a UNE entree (`killsource/world.go`, D14). Les deux
      regles sont verifiees DANS LES DEUX SENS et DISCRIMINANTES par violation temporaire
      (appel `bipedSlotBand` reintroduit dans `camo_state.go`, `ParseRegistryChunk` dans
      `build_zones.go`, entree morte dans l'allowlist), toutes restaurees.
- Gate 2 : equivalence IDENTIQUE ; tests des paquets touches verts ; lint ; mesure §6 au journal.
  ETAT : gates de code verts (details au §10). L'EQUIVALENCE 9 FILMS ET LA MESURE §6 RESTENT A
  FAIRE — le pilote lance le harnais et les temoins ; aucune cuisson n'a ete lancee par l'agent.

### Lot 3 — Correction Live Fire : le layout du catalogue pour les six balayages — effort rapide

- [ ] 3.1 `FilmContext.Layout` = `opt.MapQuant.Layout()` quand valide, auto-detection en repli —
      la regle EXACTE de `build.go:257-259`, ecrite une fois pour les positions ET les six
      balayages.
- [ ] 3.2 Equivalence : identique sur les 10 films non-Live Fire ; sur `60ae07c4` les digests
      CHANGENT — `-update -films 60ae07c4` (le TSV garde son marqueur de grammaire courant), journal avec le diff des comptes par balayage, et une
      cuisson `replay-build` de `60ae07c4` dont le journal ne porte plus « decoupage i0 illisible ».
- [ ] 3.3 Test unitaire de la regle sur des entrees du VRAI catalogue (Live Fire : gate 6,
      region 1 ; Cliffhanger : gate 5) et une entree sans largeurs (repli).
- Gate 3 : equivalence IDENTIQUE hors `60ae07c4` ; `60ae07c4.tsv` regenere avec diff au journal ;
  tests verts.

### Lot 4 — Boucles chaudes (refacto pur) — effort moyen

- [x] 4.1 Bande de slots en tableau indexe (`bipedSlotBits = 13` → 8 192 entrees) a la place de
      `map[uint32]bool` dans `matchBipedHeaderRaw` (`offline_biped.go:332`) et ses repliques
      (`ability_rank.go:159-162`, `camo_state.go:139-142`, `grapple_state.go:131-134`,
      `held_weapon_changes.go:120-123`, `i0_layout.go:194-197`, `biped_pickups`).
      FAIT sur le chemin BIPEDE (type `SlotBand`, `filmdec/slot_band_dense.go`). Le chemin
      OBJET DU MONDE (`matchWorldObjectRecord`) reste sur la map — decouverte N-AP.
- [x] 4.2 `readBitsAt` et `BitReader.ReadBits` par mot ; test differentiel D6 (n = 0..64,
      frontieres, semantique hors tampon preservee par fonction).
      ELARGI PAR LE PROFIL a `kfReadBits` (58-61 % du CPU) et `PeekBits` : cf. journal.
- [x] 4.3 `offline_aim.go:248,263` : un lecteur reutilise par `SetBitPos` ; `ascendingFromZero`
      (`offline_biped.go:359-364`) valide AVANT d'allouer.
- [x] 4.4 `objectiveevents.NamedEventsFrom` : un seul balayage de `recs` regroupe par cle a la
      place des ~22 `rawSeriesByRound`/`RealRounds` (`named.go:207-212`) ; la cle de tri
      (`named.go:224-235`, aujourd'hui `TimeMS, Slot, Stat`) gagne `Comp` et `Side` — la sortie
      n'est deterministe qu'accidentellement (iteration de map + `Stat` unique par famille) et un
      second emplacement non redondant casserait tous les digests sans test pour le dire.
      Identique sur les 11 films (les faits du corpus rendent ce gate NON vacuant, cf. 0.2).
      Identite locale prouvee par differentiel contre la forme d'avant
      (`named_onepass_test.go`) ; l'identite 11 films reste a la charge du pilote.
- [x] 4.6 (AJOUTE au lot 4 par le pilote, sur lecture du profil) `replay.ScanPlayerIndices` /
      `weaponv3.ResolveXuidToPI` : le balayage du motif xuid de 64 bits lit un MOT par octet au
      lieu de 64 bits par position. C'etait le poste numero 1 mesure (38,5 s sur 96 s de cuisson).
- [!] 4.5 Mesure §6 des 4 temoins + profil CPU de `01e1f945` au journal, comparee a la reference.
      NON TRAITE : la cuisson est a la charge du pilote (aucune cuisson lancee par l'agent du
      lot 4). Mesures de substitution au journal : micro-mesures Go des primitives reecrites.
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

- 2026-09-02 (agent instrumentation, lot 0) : le meme vice de mesure que C6 existe pour lotBP —
  `LOTBP_PHASE0.md:376,395` cite `lotBP/cout_machine.tsv`, produit par `lotBP/run_one.ps1:41` qui
  echantillonne le pic du LANCEUR `go` (`go test`), pas du binaire qui decode. Non annote (0.9 vise
  lotCter). A traiter : une note datee identique.
- 2026-09-02 (agent instrumentation) — NOTE CORRIGEE LE 2026-09-02 APRES VERIFICATION SUR PIECES
  (revue adversariale, constat C3). La note d'origine disait « dette preexistante :
  `internal/replaybuild/replaybuild.go` depassait deja 500 lignes (535 avant, 547 apres les
  4 phases) ». C'EST FAUX : a la base du lot (`0c70e3bbc`), le fichier faisait **488 lignes** et
  `BuildBytes` **62 lignes** — les deux SOUS les seuils. Le depassement (549 lignes, `BuildBytes`
  a 92) a donc ete CREE PAR LE LOT 0, pas herite. Corrige dans le lot meme : les gardes de
  lecture d'artefact partent dans `internal/replaybuild/artifact_digest.go` (scission pure,
  commentaires intacts) -> 462 lignes, et les lectures de catalogue sortent de `BuildBytes` dans
  `collecterEntreesCatalogue` -> 66 lignes, ordre des etapes observees inchange
  (`replaybuild/observe_test.go` descend desormais dans la sous-fonction). Reste VRAI et
  preexistant : `cmd/replay-build/main.go` `main()` faisait bien 83 lignes a la base (ramene a 68
  par l'extraction d'`armerProtections`, puis 73 avec la fermeture du profil CPU sous la
  sentinelle, constat C7).
- 2026-09-02 (pilote) : `metadata.duckdb` est tenue en exclusivite par le serveur de dev (l'export
  0.2 a degrade les noms EN en `map_name` bruts, sans consequence ici) ; `shared_matches_v2.duckdb`
  s'ouvre en lecture pendant que le serveur tourne. Aucune action.

- 2026-09-02 (agent harnais, 0.4bis) : neuf sites de la MEME famille (tranche batie en iterant une
  map, cle de tri non totale) a l'ASSEMBLAGE du document, non touches parce qu'ils n'atteignent que
  l'etape `artifact` — stable sur `000d5950` : `replay/ground_weapon_objects.go:185` (`gwPadsLess`),
  `replay/grapple_lines.go:117`, `replay/build_objective_objects.go:95`, `replay/flag_objects.go:142`,
  `replay/score_timeline.go:338`, `replay/zone_states_hill.go:427`, `replay/zone_states_owner.go:112`
  (tri sur `gauge[ref]` seul), `filmdec/navpoint_radial_rises.go:65,68`, `filmdec/i0_layout.go:228`.
  Un film a drapeau, a zones ou a manches peut en reveiller un : le figeage du corpus (0.5, trois
  passes) le dira — si oui, correction declaree 0.4bis etendue, sinon dette a traiter hors lot.
  Verifies SURS : `killsource/feed.go:133`, `killsource/botmeta.go:150`,
  `replay/equipment_placements.go:439`. Dette : `filmdec/lessSample` n'est pas total (ignore
  `AtRest`, `Chunk`) mais son entree est d'ordre deterministe ; `sortedZoneSlots` (reutilise par
  `gwPadsClass`) porte un nom « zone » pour un helper generique — renommer toucherait 4 sites.

  **STATUE LE 2026-09-02 (agent 0.4bis etendu, apres le temoin `084a804d` — BTB Heavies CTF,
  26 joueurs : 44 etapes de balayage identiques mais artefact different, donc le residu est a
  l'assemblage).** Chaque site re-verifie sur pieces. CINQ CORRIGES, QUATRE SURS, UN SITE
  SUPPLEMENTAIRE TROUVE.

  | Site | Statut | Cle de departage ajoutee / preuve de surete |
  |---|---|---|
  | `replay/ground_weapon_objects.go:185` | `[x]` corrige | `gwPickupLess` : apparition, puis cle de vie, `FamilyID`, `Pos`, `Moved`, `Status`, `DropperSlot`, bornage et ramasseur — tous les champs de `gwPickupObject`. `gwPadsLess` gagne `Class` puis `HasDelta` (total sur `gwPadApparition`) ; sans effet sur `ground_weapon_rules.go:144`, ou `gwAtRestOf` ne passe QUE des apparitions `spawned` sans delta. Impact reel : `gwItemLinkPickups` parcourt cette tranche DANS SON ORDRE et prend le premier a `d < bestD` strict — le ramasseur publie changeait. |
  | `replay/grapple_lines.go:117` | `[~]` sur | `out` est bati en parcourant `slots` TRIE (`:102`), les lectures d'un slot viennent de la tranche `reads` et sont triees `SliceStable` ; le tri final est `SliceStable`. Entree deterministe + tri stable = sortie deterministe. Non touche. |
  | `replay/build_objective_objects.go:95` | `[x]` corrige | `objectiveObjectLess` : T0, famille, T1, En, Fr, longueur puis points. Le couple d'hier ne separait RIEN (une seule famille publiee) et le `SliceStable` reconduisait l'ordre d'entree — le commentaire affirmait une independance qu'il n'avait pas. |
  | `replay/flag_objects.go:142` | `[x]` corrige | `flagFreeLess` gagne T1US, ID, longueur puis les echantillons (`flagFreeSampleLess`). Le triplet de tete n'est pas total : une meme cle peut porter deux creations au MEME instant (la fin de vie est la creation suivante). |
  | `replay/score_timeline.go:338` | `[~]` sur | `out` est bati en parcourant `slots` issu de `sort.Ints` (`:325`) et le tri est `SliceStable`. Non touche. |
  | `replay/zone_states_hill.go:427` | `[~]` sur | `spans` vient de `byRef[ref]` bati en parcourant la tranche `periods` via `hillSpansOf` (pure, sur `runs` tranche) ; `refs` est `sort.Ints` ; tri `SliceStable`. Non touche. |
  | `replay/zone_states_owner.go:112` | `[x]` corrige | Tri sur `(gauge[ref], ref)`. Le slot de jauge N'EST PAS une cle : `pairGaugeSlots` garantit au plus une jauge PAR ZONE, jamais au plus une zone par jauge (contrairement a `electZoneOwners` et son `held`). L'alea decidait quelle zone s'appelle A. |
  | `filmdec/navpoint_radial_rises.go:65,68` | `[x]` corrige | `lessNavpointRise` : fin, slot, puis StartMS, QStart, QEnd, Samples. (EndMS, Slot) n'est pas total — deux lectures d'un meme slot a la meme ms font deux montees qui finissent sur la meme borne. Consomme par `replay/bomb_armings.go:152`. |
  | `filmdec/i0_layout.go:228` | `[~]` sur | Le parcours de `bySlot` n'alimente que des COMPTEURS (`rep.Pairs`, `flips[k]`) : l'addition est commutative, aucune tranche n'en sort. Le tri interne `(chunk, pkt)` recoit une entree d'ordre deterministe (la tranche `samples`). Non touche. |
  | `replay/equipment_placements.go:556` (`equipmentOwner`) | `[x]` **SITE SUPPLEMENTAIRE** | Argmin de distance sur la MAP `best` avec `<` STRICT : a egalite exacte — coordonnees quantifiees, 26 joueurs — le PREMIER TROUVE gagnait, donc un rang d'iteration. Departage par slot croissant. C'est le poseur publie de chaque equipement. |

  Six tests d'ordre ajoutes (un par site corrige), tous verifies DISCRIMINANTS (le comparateur
  d'avant, rejoue dans un fichier temporaire, rend bien deux sorties differentes) :
  `replay/{ground_weapon_object,flag_free_life,objective_object,zone_letter_ranks,equipment_owner}_order_test.go`
  et `filmdec/navpoint_rise_order_test.go` — les tests d'ordre passent de QUATRE a DIX.

- 2026-09-02 (agent 0.4bis etendu) — BALAYAGE COMPLET DU CHEMIN D'ASSEMBLAGE, ce qui reste
  NON CORRIGE (sur, mais pas pour la raison qu'on croit) :
  - N-I. **Deterministe par l'entree, pas par la cle** — quatre tris de la chaine des socles et
    du calque des armes portent une cle NON TOTALE mais recoivent une entree d'ordre
    deterministe : `ground_weapon_objects.go:276` et `ground_weapon_pads.go:331` (tri des membres
    sur `Appar.TUS` SEUL), `document_ground_weapon_items.go:180` (`(T0, W)`),
    `replay/equipment_placements.go:404` (`(T0, ID)`). Ils sont surs UNE FOIS le site 185 corrige,
    et pas avant — leur surete est un HERITAGE, pas une propriete. Le commentaire de
    `gwPadWeaponID` affirme « l'ordre des membres est total » : c'est faux au sens litteral
    (deterministe, oui ; total, non). Doc a nuancer, cle a fermer si un jour l'entree change.
  - N-J. `filmdec/i0_layout.go` (`profileI0`) — le tri `(chunk, pkt)` n'est pas total ET
    l'appariement qui suit (`b.pkt <= a.pkt` -> saut) rend le resultat SENSIBLE a l'ordre des ex
    aequo : deux echantillons du meme slot dans le MEME paquet decident lequel se compare au
    paquet suivant. Sur uniquement parce que la tranche `samples` arrive dans un ordre
    deterministe. Fragile — meme famille que `lessSample` (note precedente).
  - N-K. `replay/skull_carries.go:216` — tranche batie en iterant une map de maps ; les seuls ex
    aequo possibles sur `(t0MS, round, xuid)` sont des portages a xuid VIDE (deux slots non
    nommes), et ceux-la sont TOUS ecartes en aval (`buildSkullCarries`, `cov.NoBridge`). Sur par
    un filtre aval, pas par la cle.
  - N-L. `replay/equipment_placements.go` passe de 585 a 594 lignes (dette PREEXISTANTE, deja
    au-dessus du seuil de 500 avant ce lot) : +9 lignes par la correction du poseur. `golangci-lint
    run` rend 0 issue sur les deux paquets.
  - Verifies SURS au passage, sans changement : `modalZone` / `sortedZoneRefs` /
    `electZoneOwners` / `zoneRefsOf` (parcours tries, `>` strict), `clearModalZone`,
    `hillVotesInRamps` (somme), `slotFor` et `overlapsNamedLife` (predicats d'unicite),
    `extendSlotXUID` (injectivite garantie par `injectiveOrEmpty`), `playerRoundsByXUID`
    (unicite garantie par `withoutContestedXUID`), `buildRoster`, `applyFlagReturn`
    (« le SEUL au sol »), `flagOpenings` (les ex aequo fusionnent avant le tri),
    `trancheAmbiguites`, `pad_pickup_dating` (>= 2 candidats = abstention),
    `MatchEquipmentLife`, `placementEnds` (ecriture indexee), `AbsIndexHistogram`,
    `cloneSlots`, `bandeObserveeKeyframes`, `DropIsolated`, `fillSlotBand`.

- 2026-09-02 (seconde ronde) : N-F — les comparateurs `compareSample` (`filmdec/projectiles.go:234`)
  et `lessPlacement` affirment un ordre total sans reserve alors qu'un NaN casserait
  l'antisymetrie ; non atteignable (coordonnees dequantifiees d'entiers), doc a nuancer si un
  flottant nu entre un jour dans ces structs. N-G — `replay/build.go` passe de 875 a 922 L et
  `BuildFromFilm` de ~262 a 292 L par l'observateur (intrinseque a D4a, exempte funlen par la
  config lint) : dette CREEE par le lot 0, a resorber au lot 1 (la migration `Scan*(film)` scinde
  naturellement la fonction). N-H — `neutralDeaths`/`killRefs` evalues avant `structureFor`
  depuis la scission C3 (avant : dans le litteral Options) ; sans effet, `structureFor` est un
  memo idempotent. 0.4bis : les tests d'ordre sont au nombre de QUATRE
  (`projectile_track_order_test.go`, `equipment_placement_order_test.go`,
  `slotidentity_rounds_order_test.go`, `ground_weapon_dropper_order_test.go`).

- 2026-09-02 (agent 1.2/1.3) — CINQ NOTES, dont TROIS CHANGEMENTS DE COMPORTEMENT declares (aucun
  n'atteint les 9 films du corpus, tous atteignent des films PATHOLOGIQUES) :
  - **N-M. Un chunk ILLISIBLE (erreur d'E/S) fait desormais echouer le CHARGEMENT du film**, la ou
    chaque balayage le sautait et le comptait (`KeyframeInventoryStats.ChunksUnread`). Decoder une
    fois exige de charger une fois : `filmsource.Load` rend l'erreur du premier chunk illisible.
    Consequence VOULUE et plus sure : `replaybuild.chargerFilm` journalise, tous les balayages
    degradent, la cuisson finit sur `ErrNoTracks` — un refus bruyant remplace un artefact ampute
    indiscernable d'un film pauvre. `ChunksUnread` reste PUBLIE (il entre dans l'empreinte de
    l'etape `inventory.stats`, on ne touche pas a la forme des sorties) et vaut structurellement
    zero sur le chemin film. Les deux tests qui exercaient l'ancienne degradation
    (`replay/inventory_decode_test.go`) sont reecrits sur le nouveau contrat, avec l'historique.
  - **N-N. Le chunk highlight arrive DECOMPRESSE a `analysis.ParseHighlightEvents`.** Il accepte
    les deux formes depuis l'incident du 2026-05-22 (double tolerance : il tente un
    `zlib.NewReader` et retombe sur le clair), donc memes evenements — une decompression du plus
    gros chunk du film en moins. NUANCE MESURABLE : sur un flux zlib TRONQUE, l'ancien chemin
    rendait une ERREUR (`ParseHighlightEvents decompress:`) et le fil des morts etait vide ; le
    nouveau lit le PARTIEL rendu par `filmsource`. La mesure 0.7 n'a trouve aucun flux tronque sur
    1 378 films : le cas est theorique, mais il est ecrit.
  - **N-O. `ParseRegistryChunk` n'inflate plus** (garde-rail `zlib` oblige). Deux tests de
    `filmdec` lui passaient le chunk_00 BRUT par `os.ReadFile` — `ecs_table_guard_test.go` et
    `equipment_grammar_audit_research_test.go` — et rendaient desormais un registre VIDE en
    silence ; migres vers `ReadFilmChunk(dir, 0)`, qui decompresse. Les deux appelants de
    production hors `filmdec` passaient deja du decompresse (`killsource/world.go`
    `f.chunks[0]`, `cmd/rdata_weapon_scan` via son propre `inflate`) : verifie sur pieces.
  - N-P. DETTE DE TAILLE, resorbee par DEPLACEMENTS PURS (aucune ligne de logique changee) :
    `replay/build.go` 922 -> 415 L (`build_from_film.go` 331, `options.go` 226) — c'est la
    resorption que la note N-G annoncait ; `filmdec/projectiles.go` 525 -> 469 L
    (`slot_band_filled.go` 70, la regle COMBLEE rejoint sa jumelle observee). RESTE au-dessus du
    seuil, dette PREEXISTANTE : `replay/equipment_placements.go` 595 L (+1 ligne, l'import de
    `filmsource`). Consequence des deplacements : TROIS allowlists nommaient `replay/build.go` ou
    `projectiles.go` par leur chemin — `filmdec/world_object_precision_guard_test.go`,
    `archlint/no_unbounded_film_loop_test.go`, `archlint/no_rewritten_slot_band_test.go` — et une
    quatrieme, `replay/observe_test.go`, le PARSE par son nom. Toutes repointees.
  - N-Q. `replaybuild.collecterEntreesCatalogue` passe de 5 a 6 parametres (seuil CLAUDE.md n°5 :
    5 ; seuil lint effectif du depot : 7, decision mesuree documentee dans `.golangci.yml`).
    Le fil des morts partage est le 6e. `lint` vert.
  - N-R. Le fil des morts est encore PARSE deux fois par cuisson : une fois par `replaybuild`
    (partagee entre `identifiedEvents` et `killRefs` — c'etait deux avant) et une fois dans
    `BuildFromFilm`. Le chunk, lui, n'est plus decompresse qu'une fois. La troisieme reduction
    exigerait de passer `deaths` par `Options`, ce qui deplacerait l'etape observee `deaths` :
    hors mandat de ce lot, a rouvrir si la mesure montre que le parse compte.

- 2026-09-02 (agent 1.4 + part killcollector de 1.6) — CINQ NOTES, dont UNE DETTE ROUGE
  ANTERIEURE AU CHANTIER :
  - **N-S. LES GOLDENS `killsource` SUR FILMS REELS SONT DEJA ROUGES A LA BASE DU LOT**, et ce
    n'est PAS le lot 1 qui les a fait bouger. `TestGoldenFilms` (gate `KILLSOURCE_FIXTURES`, donc
    JAMAIS joue en CI) echoue sur 3 des 4 films de reference, avec les MEMES trois ecarts a
    l'octet aux trois points de mesure — apres l'item 1.4, a HEAD `c17f4941f`, et a `900384f50`
    (la base du plan), rejoues dans un worktree jetable (`git worktree add --detach`, supprime
    depuis) : `000d5950` calibration « score 346 » -> 347 ; `9b191a7f` « croissance x1.157 » ->
    x1.158 ; `fccc61cd` « source appartenant a la victime : 2 propose(s) » -> 3 (2 publiees dans
    les deux cas, donc AUCUNE ligne publiee ne change). `78919882` passe partout. La derive est
    donc anterieure au chantier et dormante : c'est EXACTEMENT le motif que
    `killsource/minibobine_test.go` denonce dans son propre en-tete (« un gate optionnel est un
    garde qui ne peut pas echouer »), retombe sur le golden qui l'avait inspire. NON TRAITE
    (hors perimetre, et regenerer un golden hors lot de correction est interdit par D4) : a
    dater et a trancher hors plan — soit on retrouve le commit qui a bouge, soit on refige avec
    le diff au journal.
  - **N-T. L'identite de la traduction `filmsource` -> `killsource` est MESUREE, pas affirmee.**
    Un test de diagnostic temporaire (recopie de l'ancien `DirChunks` + `inflate` (io.ReadAll) +
    `splitPackets`, compare au film charge) rend sur les quatre films de fixture : chunks
    IDENTIQUES a l'octet (position par position) et **0 ecart** sur 60 970 / 72 651 / 88 407 /
    69 607 paquets, sur `(chunk, idx, type, ts, taille)`. C'est la preuve directe que la regle
    (3) de D3 (terminateur type 7 EMIS) se neutralise exactement par le filtre de type pose dans
    `packetsOf`, et que le film n'a ni gagne ni perdu un octet. Test supprime apres mesure (il
    aurait ete un musee de l'ancienne grammaire).
  - N-U. `sync/killcollector/collector.go` passe de 537 a 546 L (dette PREEXISTANTE, deja
    au-dessus du seuil de 500 avant ce lot) : +9 lignes pour le chargement unique du film et son
    refus d'erreur (jamais avale). `golangci-lint run` rend 0 issue sur les quatre paquets
    touches. `positions.go`, lui, passe de 346 a 330 L (le pont disque supprime).
  - N-V. `.ai/V7.5/killweapon/GUIDE_KILLSOURCE.md` et `PLAN_BRANCHEMENT_KILLSOURCE.md` citent
    `killsource.DirChunks` / `MemoryChunks` dans leurs exemples de cablage : documentation
    devenue fausse par la suppression de ces symboles. NON TRAITE (hors perimetre du lot) —
    a reprendre a l'item 6.2 avec `docs/COMMANDS.md`.
  - N-W. DEUX CHANGEMENTS DE COMPORTEMENT declares, tous deux sur des chemins de DEGRADATION :
    (1) `replaybuild.decodeKillSource` journalisait en **Debug** « chunks illisibles » quand
    `DirChunks` echouait ; le film etant desormais charge en amont, ce cas arrive comme
    `ErrNoChunk` et se journalise en **Info** — la cause, elle, est deja journalisee en Warn par
    `chargerFilm`, donc l'information n'est pas perdue mais montee d'un niveau ;
    (2) cote positions, le compteur `killsource_positions_pont_echec` garde son NOM (un compteur
    publie est une interface) mais n'a plus qu'une cause : la sequence trouee. Les causes
    « disque plein / chemin illisible » disparaissent avec le repertoire temporaire.

- 2026-09-02 (agent 1.5 + part CLI de 1.6) — SEPT NOTES, dont UNE MESURE QUI A CHANGE UNE
  DECISION DU LOT :
  - **N-X. LE MANIFESTE LISTE BIEN LE CHUNK 00, donc `StatRecordsCtx` le consommait.** La
    question posee a l'entree de l'item (« les ChunkMeta du manifeste listent-elles le registre ? »)
    est tranchee sur pieces : 1 380 manifestes du cache sur 1 380 portent une entree `index: 0`,
    et son `chunk_type` vaut 1 partout. L'ancienne boucle iterait donc le registre comme les
    autres. LE JEU DE CHUNKS CONSOMME EST PRESERVE A L'IDENTIQUE, et le piege D3 sur ce chunk ne
    mord pas : les en-tetes degeneres du registre portent une TAILLE NULLE, donc des trames
    VIDES, dont `scanFrameForRecords` ne tire aucun enregistrement (`lim = 0*8 - statTailBits`
    est negatif). Que la nouvelle grammaire s'arrete au premier d'entre eux ne retire donc aucun
    record — mesure, pas raisonnement : cf. N-Z.
  - **N-Y. UN FILM DU CACHE PORTE DES CHUNKS HORS MANIFESTE, et sans filtre il aurait ete date
    faux.** `7b0d89c4` a `chunk_31.bin` et `chunk_32.bin` sur disque sans entree au manifeste
    (seul cas sur 1 380 ; deux autres films, `33b9fbe9` et `f8c067d7`, ont l'inverse : un
    manifeste complet et zero fichier). L'ancienne `FilmSource` iterait le MANIFESTE et ne les
    voyait jamais ; le film de `filmsource.LoadDir`, lui, porte tous les fichiers PRESENTS. D'ou
    `objectiveevents.manifestChunks` (et son jumeau `replaybuild.chunksDuManifeste` pour
    l'horloge de la bombe) : un chunk sans type de manifeste n'a pas de `start_ms`, donc rien
    d'y datable. LE CRITERE EST MESURE : sur les 1 380 manifestes, `chunk_type` ne prend que 1
    (en-tete), 2 (jeu) et 3 (pied) — JAMAIS 0, la valeur que `LoadDir` synthetise pour un fichier
    hors manifeste. La distinction compte aussi en aval : `filmdec/navpoint_radial_scan.go`
    teste `start, hasStart := startMS[c]`, donc inscrire un chunk inconnu a zero lui dirait
    « ce chunk commence a 0 » au lieu de « je ne sais pas ».
  - **N-Z. L'IDENTITE EST PROUVEE PAR EMPREINTES, avant ET apres.** Deux tests temporaires
    (supprimes) : le premier a compare, sur le CODE D'AVANT, `StatRecordsCtx(FilmSource)` a une
    reimplementation `filmsource` — 10 films (les 9 du corpus + `7b0d89c4`), empreintes SHA-256
    des enregistrements (temps, slot, manche, composants) IDENTIQUES, bursts identiques, pied
    identique a l'octet ; le second a rejoue les memes empreintes sur le CODE MIGRE et les a
    comparees aux valeurs relevees. 592 / 2 208 / 1 131 / 1 199 / 1 089 / 1 647 / 890 / 9 111 /
    619 / 638 enregistrements, meme empreinte des deux cotes. La variante SANS le filtre de N-Y
    a ete mesuree aussi : elle rend le meme resultat sur ces 10 films — le filtre est donc une
    garantie, pas un correctif. AUCUNE CUISSON COMPLETE LANCEE (chaine en vol) : le harnais des
    9 films reste au pilote.
  - **N-AA. LE GARDE-RAIL `filmcache_guard_test.go` EST SUPPRIME, ET SES TROIS DEROGATIONS AVEC.**
    Il cherchait par expression reguliere les implementations de `objectiveevents.FilmSource`
    (methode `Chunks() []ChunkMeta`), interface qui n'existe plus, et son allowlist portait trois
    entrees justifiees par le CYCLE D'IMPORT `filmcache -> objectiveevents` que ce lot supprime.
    Remplace par `var _ filmsource.Source = (*Source)(nil)` dans `filmcache.go`. La derogation
    tombee, les deux copies de la disposition du cache qu'elle abritait sont parties avec elle :
    `objectiveevents/extract_test.go` et `assaut_footer_research_test.go` passent par
    `filmcache.LoadFilm` (le troisieme fichier allowliste, `statborg_rounds_test.go`, ne lisait
    pas le disque — sa source etait un buffer repete, devenu `filmsource.Load(MemoryChunks)`).
  - N-AB. `filmcache.Source` change de forme : `Chunks()` devient `Meta() []filmsource.ChunkMeta`
    et `ChunkData(numero)` devient `Chunk(position)` — le contrat de `filmsource.Source`, ou
    l'indice est la POSITION dans le manifeste et non le numero de fichier. Deux nouveaux
    chargeurs, `LoadFilm(root, short)` et `LoadFilmDir(chunkDir)`, font manifeste + `LoadDir` en
    un appel : ils sont le chemin UNIQUE du cache vers un film, et ils prennent les FICHIERS
    presents (pas les entrees du manifeste), donc un cache partiel rend un film ampute plutot
    qu'une erreur — exactement la degradation d'avant, ou chaque chunk absent etait saute.
  - N-AC. CHANGEMENT DE COMPORTEMENT declare, sur un chemin de degradation : `readFilmStats`
    gardait sur « manifeste absent » et rendait alors un `filmStats` vide ; il garde desormais
    sur « film illisible OU manifeste absent ». Le cas neuf est le film dont le manifeste existe
    mais dont AUCUN chunk n'est sur disque (`33b9fbe9`, `f8c067d7`) : il produisait avant un
    `ScoreInput` a zero enregistrement, il produit maintenant un `filmStats` vide. INOBSERVABLE
    en sortie : sans chunk, `BuildFromFilm` echoue juste apres sur `ErrNoFilmChunk` et la cuisson
    ne rend aucun artefact dans les deux cas (verifie sur pieces : `filmdec.FilmChunkNumbers`
    rend nil sur un film nul).
  - N-AD. `cmd/replay-equiv/walkers.go` porte les TROIS marcheurs historiques en copie de mesure
    et cite leurs originaux par fichier et par ligne (`objectiveevents/film.go:120-140`, etc.).
    Apres les items 1.4 et 1.5, aucun de ces originaux n'existe plus : le mode `-walkers` mesure
    desormais de l'histoire, et ses commentaires pointent dans le vide. NON TRAITE (le harnais
    est au pilote, et la mesure 0.7 qu'il servait est close) — a trancher au lot 6 : le mode
    disparait avec sa raison d'etre, ou son en-tete dit qu'il fige une comparaison historique.

- 2026-09-03 (agent 1.7-1.10) — QUATRE NOTES, dont UNE LISTE DU PLAN PRISE EN DEFAUT :
  - **N-AE. LA LISTE « SEPT SITES ZLIB » DE L'ITEM 1.9 ETAIT INCOMPLETE — IL Y EN A NEUF (avec
    `filmsource`).** Le site manquant est `cmd/replay-equiv/walkers.go`, cree par le lot 0
    LUI-MEME (mode `-walkers` de la mesure 0.7) : il porte en copie les trois marcheurs
    historiques et leur inflate. Le plan avait ecrit sa liste fermee avant que ce fichier
    n'existe — la lecon de methode du §10 (« une liste dite fermee se verifie par un grep dont la
    commande est ecrite dans l'item ») vaut aussi pour les listes que le chantier fait BOUGER en
    cours de route. Le site est allowliste avec sa justification datee et le renvoi a N-AD ; la
    decision de fond (retirer le mode, ou declarer qu'il fige une comparaison historique) reste
    au lot 6, NON TRAITEE ici.
  - N-AF. `filmdec/keyframe_entity_queue.go` porte DEUX formes `dir` qui ne se declarent PAS
    enveloppes D2 : `FirstPacketOfType` et `AllPacketsOfType`, baties sur `ReadFilmChunk` +
    `CountFilmChunks` + `WalkPackets`. Quatre appelants, tous des tests de `filmdec`. Elles ne
    sont pas dans la liste fermee d'`archlint` (elles ne sont pas des balayages) mais elles
    maintiennent vivante l'entree `keyframe_entity_queue.go` de l'allowlist `os.*` — a replier
    au lot 6 avec les enveloppes. Meme famille : `GroundWeaponPositions` chargeait DEUX films par
    appel (un par enveloppe deleguee) ; sans consequence (instrument de mesure sous garde
    `GW_FILM`), et desormais marquee D2.
  - N-AG. LE CRITERE DE RETRAIT DES ENVELOPPES (lot 6, « le grep ne rend plus que leurs
    definitions ») EST PLUS CHER QUE PREVU, et c'est mesure : `ReadFilmChunk` compte 177 sites
    d'appel et `CountFilmChunks` 130, tous dans des tests de recherche de `filmdec` et `replay`
    (contre 1 a 6 pour la plupart des `ScanFilm*`). Retirer les enveloppes suppose donc de
    migrer ~300 appels de tests de recherche vers `filmcache.LoadFilm` / `FilmChunkAt` : a
    chiffrer au lot 6 avant de le promettre.
  - N-AH. COUT EN TEMPS DE TEST : `TestZeroDisqueBalayagesSupportes` (item 1.8) decode la
    mini-bobine par sept familles, soit ~15 s ; la suite `analysis/replay` passe a ~60 s. Le
    recouvrement avec `TestEquivalenceMiniFilm` (memes sept familles, par les enveloppes `dir`)
    est assume : l'un mesure l'IDENTITE des sorties, l'autre l'ABSENCE de disque, et fusionner
    les deux ferait un test qui echoue pour deux raisons distinctes. A revoir si la CI serre.
  - N-AI. `ScanBipedPositions` RELEVE ENCORE SA PROPRE BANDE (et detecte le decoupage quand
    `opt.Layout` est nil) : c'est la HUITIEME bande bipede de la cuisson, la seule que le lot 2
    n'a pas mutualisee. Raison ecrite : ses deux valeurs sont CONDITIONNELLES a des options —
    la bande porte sur `opt.Chunks` (une sous-liste quand un instrument la restreint) et la
    detection ne sert que de repli. Les brancher sur le contexte demanderait de trancher le cas
    de la sous-liste, ce qui n'est plus un refacto pur, et le plan ne le demande pas (item 2.1
    nomme les six delta + `biped_pickups`). Les deux appels sont ALLOWLISTES au garde-rail 2.4
    avec cette justification et un retrait cible au lot 4 (item 4.1 change la representation de
    la bande). Cout mesure : une marche de l'image-cle de tete de chaque chunk, une fois par film.
  - N-AJ. `killsource/world.go` ANALYSE ENCORE LE REGISTRE LUI-MEME (`filmdec.ParseRegistryChunk`
    sur `f.chunks[0]`) : c'est desormais la SEULE analyse de registre de la chaine de cuisson qui
    ne passe pas par `FilmContext`. Hors perimetre SANS CONDITION (D14) ; l'entree est isolee dans
    l'allowlist de la regle 2 du garde-rail 2.4, donc elle se verra le jour ou killsource entrera
    dans un plan.
  - N-AK. LA BANDE DE `DetectI0LayoutOf` N'EST PAS CELLE DU CONTEXTE, et ce n'est pas une dette :
    la detection releve la sienne sur les SIX PREMIERS chunks (`detectMaxChunks`), le contexte sur
    TOUS les chunks de donnees. Deux valeurs differentes ; les confondre changerait la detection —
    donc changerait la sortie. Ecrit dans l'allowlist du garde-rail pour qu'un futur « pourquoi
    deux bandes ? » trouve la reponse a cote du code.
  - N-AL. IL RESTE QUATRE MARCHES COMPLETES DES IMAGES-CLES pour les bandes d'OBJETS DU MONDE, et
    deux d'entre elles calculent la MEME valeur : `ScanEquipmentPlacements` releve
    `worldObjectSlotBand(ti=37)` tandis que `decodeFilmPadScan(ti=37)` releve
    `ScanWorldObjectKeyframes(ti=37).Band`, qui applique le MEME `slotBandExcluding` aux MEMES
    images-cles. S'y ajoutent `ScanWorldObjectKeyframes(ti=42)` et `worldObjectSlotBand(ti=41)`
    (projectiles), qui portent sur d'autres archetypes. L'item 2.2 ne nommait que la paire
    `equipment_placements`/`projectiles`, qui est traitee ; la paire ti=37 restante ne l'est PAS
    (perimetre ferme). A chiffrer au lot 4 : le recensement (`SeenUS`) et la bande sortent deja de
    la meme marche, un `FilmContext.WorldObjectKeyframes(ti)` memoise les rendrait tous les deux.
  - N-AM. LE CONTEXTE A ELARGI LE PERIMETRE DE SIGNATURES AU-DELA DES SEPT BALAYAGES NOMMES, et
    c'etait la condition pour que l'allowlist de 2.4 soit FERMEE : huit autres formes film
    (`ScanUnitEquipment`, `ScanEquipmentCreations(ForBand)`, `CalibrateMPPWidthsOf`,
    `ScanGroundWeaponCreations(ForBand)`, `ScanEquipmentState`, `ScanManagedProperties`,
    `ScanNavpointRadial`, `ScanObjectives`, `ScanEquipmentPlacements`) prennent desormais un
    `*FilmContext` a la place du `*filmsource.Film`, parce qu'elles lisent un archetype. AUCUNE
    n'a d'appelant hors `filmdec` et `replay` (verifie par grep avant migration) : le cout de
    l'elargissement est nul cote appelants, et le benefice est qu'aucun accesseur d'archetype ne
    peut plus re-analyser le registre.

- 2026-09-03 (agent lot 4) — decouvertes N-AN a N-AR.
  - N-AN. LE DOMAINE DE `BitReader.ReadBits` N'EST PAS 0..64, contrairement a ce qu'ecrit D6.
    `components_batch7.go:137-138` (`consumeTrackFrameComponent`, FUN_142ed740c) lit une largeur
    `w` sur 12 BITS DU FLUX puis fait `br.ReadBits(w)` : `w` peut donc valoir jusqu'a 4 095.
    Trois autres sites passent une largeur venue du flux ou d'une table
    (`components_managed_property.go:174`, `components_biped_ability.go:217`,
    `components_object.go:418`). La semantique d'origine pour `n > 64` — la valeur ne garde que
    les 64 DERNIERS bits lus, et le curseur avance quand meme de `n` — est donc REELLE, et elle a
    ete preservee (chemin bit a bit conserve, verrouille par
    `TestReadBitsWordWideMatchesReference`). D6 est a amender : « domaine reel 0..64 » vaut pour
    les appels a largeur LITTERALE, pas pour la fonction.
  - N-AO. LE PROFIL NE DESIGNE PAS LES PRIMITIVES QUE LE PLAN NOMMAIT. Au profil du lot 1
    (`tmp/L1_01e1f945.cpu.prof`, 152 s d'echantillons), `readBitsAt` pese 0,62 % et
    `BitReader.ReadBits` n'apparait pas du tout ; ce sont `filmdec.kfReadBits` (58,4 % flat) et
    `weaponv3.bitReader.bit` (20,8 % flat) qui portent le temps. Les deux primitives nommees par
    le plan ont ete reecrites quand meme — meme famille, meme test differentiel — mais le gain
    vient d'ailleurs. A retenir pour les prochains lots de perf : ce plan-ci nommait ses
    primitives par lecture de code, pas par mesure.
  - N-AP. LA BANDE D'OBJETS DU MONDE N'A PAS ETE DENSIFIEE, ET C'EST UN CHOIX CHIFFRE.
    `runtime.mapaccess1_fast32` pese 2,28 s sur 152 s (1,5 %), reparti en 1,18 s pour
    `matchWorldObjectRecord` et 0,92 s pour `matchBipedHeaderRaw`. Seul le second est traite :
    densifier le premier obligeait a changer le type de retour de `worldObjectSlotBand` /
    `observedSlotBand` / `slotBandExcluding`, donc DOUZE signatures exportees en `...ForBand`
    (`ScanWorldObjectsForBand`, `ScanEquipmentCreationsForBand`, `CalibrateMPPWidthsOf`,
    `WorldObjectPositionsForBand`, `GroundWeaponSlotBand`, le champ `WorldObjectCensus.Band`...),
    consommees hors `filmdec` par `replay` et par ~24 sites de tests de recherche — dont
    `build_ground_weapons.go:99` (`len(kf.Band) == 0`), ou une conversion silencieuse en tranche
    dense aurait CHANGE le comportement de production sans erreur de compilation. Pour 0,8 % du
    profil, dans un lot de refacto PUR dont l'identite est jugee par un harnais que l'agent
    n'execute pas, le rapport n'y etait pas. A rouvrir avec le harnais sous la main.
  - N-AQ. LES PRIMITIVES DE BITS NE SONT PLUS INLINEES, et le budget de l'inliner l'explique :
    `wordBitsAt` coute 111 pour un budget de 80, parce que `binary.BigEndian.Uint64` s'y inline
    (huit chargements). Un cout d'appel est donc paye par lecture ; il reste tres inferieur au
    gain (`kfScanNext` : 5,23 ms -> 0,79 ms puis 0,57 ms, soit 8,9x, micro-mesure sur tampon
    aleatoire de 256 Kio). LEVIER RESTANT, NON PRIS : une fenetre GLISSANTE dans `kfScanNext`
    (`id(q+1) = (id(q) << 1 | bit(q+31)) & 0xFFFFFFFF`) supprimerait la lecture par position ;
    estime a quelques secondes de plus sur la cuisson. Ce n'est plus une reecriture de primitive
    mais une reformulation de la boucle d'ancrage : a faire avec le harnais d'equivalence.
  - N-AR. `seriesBySlot` A FAILLI ETRE SUPPRIMEE A TORT. Apres le regroupement en une passe, elle
    n'a plus d'appelant dans `NamedEventsFrom` — mais `countsOf` (`named.go`, qui alimente
    `CrossCheckNamedEvents`, PRODUCTION) l'appelle toujours. Une suppression « code mort » fondee
    sur le seul appelant visible aurait casse le controle croise. Note ici parce que le meme piege
    guette toute factorisation de ce fichier : `named.go` a DEUX consommateurs de series, pas un.

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
- 2026-09-02 (lot 0, en cours) — Le pilote ne code plus : execution par agents (Opus) sous ce plan,
  revue adversariale par contexte frais, gates/statuts/journal/commits par le pilote (demande
  utilisateur). Fait avant le basculement : 0.1 (`CORPUS.txt`, `BOMBES.txt`), 0.2 (`levelup
  replay-facts-export`, 13 fichiers de faits — `metadata.duckdb` etait tenue par le serveur de
  dev : les cartes candidates sont les `map_name` BRUTS du registre, tous resolus au catalogue de
  bornes par `NormalizeMapName` ; la base partagee, elle, s'est ouverte en lecture sans arreter le
  serveur), 0.3 (observateur + `BuildFromFilmSteps` / `BuildBytesSteps*` + gardes structurels sur
  la SOURCE — la mini-bobine ne permet pas d'executer `BuildFromFilm` : aucune image-cle de
  bipede, `ScanFilmBipedPositions` refuse ; le test 0.3 lit donc le corps de la fonction, patron
  archlint), `AcquireSoloWait` (un test d'annulation a corriger : double `%w`). Confie aux agents :
  le reste de 0.4 (paquet `digest`, `FactsFile`, `cmd/replay-equiv` + mode `-walkers`,
  `TestEquivalenceMiniFilm`), 0.6, 0.9. Le pilote lancera 0.5, 0.7, 0.8.
- 2026-09-02 (lot 0, en cours) — Agents rendus : 0.6 et 0.9 (instrumentation) gate vert ; harnais
  (`digest`, `FactsFile`, `cmd/replay-equiv` + `-walkers`, `TestEquivalenceMiniFilm`) gate vert.
  DEUX DECOUVERTES DU HARNAIS, avant tout figeage : (1) NON-DETERMINISME en production —
  `filmdec/projectiles.go:159-176` (`ScanFilmWorldObjectsForBand`) itere une map puis trie par
  `sort.Slice` (instable) sur `lessTrack` (`:194-202`, cle `(Pts[0].TimestampUS, Slot, Gen)`) : sur
  `000d5950`, 3 pistes ex aequo sur `ti=42` (549) et 3 sur `ti=37` (477) — l'etape `pads` change
  d'empreinte d'une execution a l'autre, et le commentaire de `:169-175` dit que cet ordre decide des
  positions de naissance publiees. Correction PREALABLE au 0.5 (item 0.4bis, correction declaree :
  seul l'ordre des ex aequo change, et il etait aleatoire — aucune ancienne verite a preserver).
  (2) D3 : sur `000d5950`, 28/28 chunks divergent entre `filmdec.WalkPackets` et la grammaire
  unifiee, axe `taille_nulle` ; `killsource` s'accorde partout ; `objectiveevents` diverge sur
  1 chunk. Diagnostic demande (quels paquets : type, position) AVANT toute escalade.
- 2026-09-02 (lot 0, suite) — 0.7 EXECUTE par le pilote : 1 380 films en 6 min 55 s
  (`tmp/replay-equiv.exe -walkers`), resultats au §2 de `MESURES_CUISSON_PERF.md` : 0 flux tronque,
  `killsource` = grammaire unifiee partout, `filmdec` divergent sur TOUS les chunks de 1 378 films,
  `objectiveevents` sur exactement 1 chunk par film. Diagnostic paquet par paquet (agent, 3 films,
  tous chunks) : le paquet de taille 0 des chunks de donnees est le terminateur CHUNK_END (type 7),
  dernier, rien apres ; `chunk_00` (registre) porte des en-tetes degeneres au milieu. D3 REVISEE
  sur ces faits (§3) : la grammaire emet le terminateur puis s'arrete — plus « arret sur taille 0 » ;
  la regle d'arret sur digest divergent au lot 1 reste. 0.4bis : quatre non-determinismes corriges
  (dont un `dropper` d'arme au sol tire au sort), deux constructions identiques sur `000d5950`.
  0.5 lance par le pilote (trois passes : creation, puis deux verifications). Revue adversariale du
  lot lancee en parallele (contexte frais). 0.8 attend une machine au repos.
- 2026-09-02 (lot 0, suite) — SECONDE RONDE de revue (contexte frais, consigne refuter) : les 7
  corrections et 5 notes de la premiere ronde TIENNENT toutes (deplacement pur prouve par diff
  contre la base, lint 0 issue, gates verts). Deux reprises avant commit, dépechees : N-A (le tri
  des cles de map de `digest` n'etait pas deterministe quand deux cles distinctes ont le meme
  rendu — la famille 0.4bis logee dans l'instrument de certification ; departage par le rendu de
  la valeur) et N-B (marqueur `# digest-grammar: N` dans les TSV de reference, sans lequel un
  changement du rendu se lit comme une regression du decodeur — 6 TSV sur 9 etaient restes sous
  l'ancienne grammaire, indetectable). Notes portees au §8 : N-C a N-H. Regle (4) de D3
  desambiguisee ci-dessus (arret AVANT emission).
- 2026-09-02 (lot 0, suite) — MESURE 0.7 REJOUEE avec la grammaire D3 REVISEE (binaire corrige,
  6 min 40) : 1 378 films, 0 tronque, `filmdec` divergent sur EXACTEMENT 1 chunk par film (le
  registre `chunk_00`, jamais consomme comme flux de paquets), `killsource` divergent du seul
  terminateur emis (paquet type 7, filtre par type en aval) sur les chunks de donnees,
  `objectiveevents` idem registre. Les chunks de DONNEES sont bit-identiques entre `filmdec` et
  la grammaire retenue sur tout le cache : le prerequis « lot 1 = refacto pur » est acquis, a
  confirmer par les empreintes. Resultats au §2 de `MESURES_CUISSON_PERF.md` ; figeage des 11
  references relance apres les reprises N-A/N-B (les TSV mixtes ont ete detruits).
- 2026-09-02 (lot 0, figeage) — DEUX FILMS DU CORPUS SONT DES BOMBES MEMOIRE, reveles par le
  figeage (le harnais a fait son travail de sonde) : `1c4c63c2` (BTB One Flag, le plus gros du
  cache) tue au plafond 3 Gio a 3,96 Gio en 4,5 s ; `60ae07c4` (Live Fire, Ranked:Oddball —
  LE TEMOIN DU LOT 3) tue a 4,02 Gio en 2 min 12. Profil « gigaoctets en secondes » = famille
  `NamedEventsFrom` presumee (les deux modes portent des evenements nommes) ; a confirmer par le
  profil heap au lot 4b. CONSEQUENCES : corpus ACTIF des lots 1-2 = 9 films (les 9 references
  figees et verifiees) ; `1c4c63c2` et `60ae07c4` rejoignent le regime BOMBES (D11, empreintes
  figees au lot 4b) ; le LOT 3 depend desormais du plafond D13 — decision a trancher a l'entree
  du lot 3 : passer le lot 4b AVANT le lot 3, ou figer le temoin Live Fire avec un plafond
  releve (`-mem-gib 6`) si son pic reel le permet. `CORPUS.txt`/`BOMBES.txt` a realigner au
  commit du lot. La mesure 0.8 de `1c4c63c2` echouera au plafond : attendu, journalise.
- 2026-09-02 (lot 0, 0.4bis ETENDU) — Le temoin `084a804d` (BTB Heavies CTF, 26 joueurs) a
  tranche : 44 etapes de balayage IDENTIQUES entre deux cuissons, artefact different (meme taille
  7 249 908, sha different) — le residu est bien a l'ASSEMBLAGE, comme la note des neuf sites le
  redoutait. Les neuf sites ont ete re-verifies sur pieces un par un : CINQ CORRIGES
  (`ground_weapon_objects.go` + `gwPadsLess`, `flag_objects.go`, `build_objective_objects.go`,
  `zone_states_owner.go` — les lettres A/B/C de zone tiraient au sort quand deux zones partagent
  un slot de jauge —, `filmdec/navpoint_radial_rises.go`), QUATRE DECLARES SURS AVEC PREUVE
  (`grapple_lines.go`, `score_timeline.go`, `zone_states_hill.go` : entree triee + tri stable ;
  `filmdec/i0_layout.go` : compteurs commutatifs). Le balayage du reste du chemin d'assemblage a
  livre UN SITE QUE L'INVENTAIRE AVAIT MANQUE, de la meme famille « premier-trouve d'une
  iteration de map » : `replay/equipment_placements.go` `equipmentOwner` prenait l'argmin de
  distance par un `<` strict en iterant une map — a egalite exacte (coordonnees quantifiees,
  26 joueurs) le POSEUR PUBLIE de chaque equipement etait tire au sort. Corrige par le slot.
  Six tests d'ordre ajoutes, chacun verifie DISCRIMINANT en rejouant le comparateur d'avant
  (fichier temporaire, supprime) : les tests d'ordre passent de quatre a dix. Statut par site,
  cle de departage et preuves de surete au §8 ; decouvertes non traitees en N-I a N-L (dont une
  doc a nuancer et une fragilite d'appariement dans `profileI0`). Gate : `gofmt -l` vide,
  `go vet ./internal/analysis/...` vide, `go build ./...` vide, suites completes
  `filmdec` + `replay` vertes, `golangci-lint run` 0 issue sur les deux paquets. AUCUNE CUISSON
  LANCEE (chaine en vol) : la preuve est les tests d'ordre et la lecture. Rien committe.
- 2026-09-02 (lot 0, cloture) — Assemblage rendu deterministe (6 sites corriges dont
  `equipmentOwner`, manque de l'inventaire ; 4 prouves surs ; 10 tests d'ordre au total) apres que
  les passes de verification ont revele un ecart vivant sur `084a804d` (44 balayages identiques,
  artefact seul different, 3 sha distincts en 3 cuissons). Re-figeage puis DEUX verifications
  finales : 9/9 identiques, deux fois. 0.5 et 0.8 clos — reference HEAD au §1 de
  `MESURES_CUISSON_PERF.md` (temoins a 2 min 24 - 2 min 49, decodage = ~94 % du temps,
  `playerIndices` poste n°1 a 35-40 s). Gate de cloture lance (suite complete + integration
  -p 1) ; commit du lot a sa sortie verte. CI de branche : branche jamais poussee, aucun run
  distant a verifier — la CI jugera au premier push.
- 2026-09-02 (pilote, gate de cloture) : `go test ./...` complet echoue sur DEUX points HORS
  perimetre — (a) `internal/sync/v2` : erreur transitoire du cache de build Windows
  (« Ressources systeme insuffisantes ») pendant que la chaine de cuisson tournait en
  parallele ; VERT au rejeu isole ; (b) `internal/himap` `TestBalayageCoquille` : balayage des
  27 cartes depuis les FICHIERS DU JEU installes (garde `DeployRoot`, skip en CI), duree reelle
  > 25 min, donc > au timeout de 10 min de `go test ./...` — echec STRUCTUREL et PREEXISTANT
  sur toute machine ou le jeu est installe, quel que soit le commit (le lot 0 ne touche aucun
  fichier de `himap`). A traiter hors lot : borne `BALAYAGE_CARTES` par defaut, tag dedie, ou
  timeout declare. Les gates du lot (paquets touches + integration -p 1 + vet + lint) sont
  VERTS.
- 2026-09-02 (lot 1, item 1.1) — Le paquet FEUILLE `internal/analysis/filmsource` existe
  (`doc.go`, `source.go`, `film.go`, 2 fichiers de test) : stdlib seule, zero import du depot,
  grammaire D3 REVISEE appliquee dans l'ordre exact de ses quatre regles, inflate pre-dimensionne
  (`bytes.Buffer.Grow` ratio x6 borne a 64 Mio + `io.Copy`, pas d'`io.ReadAll` nu) rendant le
  PARTIEL sur flux tronque, payloads en SOUS-TRANCHES du chunk (documente : garder un paquet
  retient son chunk). 17 tests, tous sur des chunks CONSTRUITS sauf le dernier — et c'est celui-la
  qui verrouille D3 : sur la mini-bobine reelle, le chunk highlight rend le MEME jeu de paquets que
  `filmdec.WalkPackets` (index, type, ts, payload octet pour octet). Les trois variantes rejetees
  ont ete REJOUEES pour prouver les tests discriminants : sans la regle (4) un test rougit, sans la
  regle (3) deux rougissent, et la candidate abandonnee « arret sur taille <= 0 » fait rougir le
  test du film REEL — la mesure 0.7 est donc rejouee en CI, en une seconde, sur trois chunks.
  Garde-rail `internal/archlint/filmsource_leaf_test.go` : parse des imports (go/parser, pas un
  grep — le paquet CITE les chemins du cycle dans ses commentaires), `_test.go` exclus (le test
  externe importe `filmdec` pour la comparaison ci-dessus), verifie discriminant par un import
  temporaire. AUCUN consommateur touche (1.2+). Gate : `gofmt -l` vide, `go vet` vide,
  `go test ./internal/analysis/filmsource/ ./internal/archlint/` vert, `golangci-lint` 0 issue,
  `go build ./...` vert. Fichiers 61-331 L, fonctions < 60 L. Rien committe.
- 2026-09-02 (lot 1, items 1.2 et 1.3) — LES BALAYAGES NE RELISENT PLUS LE FILM. Les ~30
  `ScanFilm*(dir)` de `filmdec` et les quatre de `replay` ont chacun leur forme
  `Scan*(film *filmsource.Film, ...)` ; `BuildFromFilm(matchID, titleSlug, film, opt)` remplace
  `filmDir` ; `replaybuild.BuildBytes` charge le film UNE fois (manifeste `filmcache.OpenChunkDir`
  traduit en `[]filmsource.ChunkMeta` par la couche d'assemblage — `filmsource` reste FEUILLE) et
  le passe a tout le monde. `ParseRegistryChunk` perd son inflate ; `inflateChunk` meurt ;
  `filmdec` et `replay` n'importent plus `compress/zlib`. DEUX points de chargement en production,
  et deux seulement : `replaybuild/filmload.go` (la cuisson) et `cmd/zone-attribution/measure.go`
  (la mesure). REGLE D'INDEXATION POSEE (le piege n°1 des notes de l'agent 1.1) : `LoadDir`
  synthetise TOUJOURS `ChunkMeta.Index` depuis le nom du fichier et fusionne le manifeste PAR
  NUMERO, jamais par position — une bobine sans `chunk_00` (la mini-bobine) a son premier chunk de
  DONNEES a la position 0, et les confondre marcherait un chunk de donnees comme un registre.
  Cote consommateur, `filmdec.FilmChunkNumbers` s'arrete au premier trou de numerotation, comme
  l'ancien `CountFilmChunks` : refacto PUR, la regle est heritee sciemment et tombera avec les
  enveloppes. Le pont `filmdec.FilmChunkAt` est prouve EQUIVALENT a `ReadFilmChunk` +
  `WalkPackets` sur un vrai chunk, bornes de payload comprises
  (`filmdec/film_chunks_test.go`). ENVELOPPES D2 : les 40 formes `dir` survivent, TOUTES avec des
  appelants hors production (verifie fonction par fonction), aucune supprimee, aucune appelee
  depuis la cuisson. FIL DES MORTS : une seule lecture partagee par `identifiedEvents` et
  `killRefs` (elles ouvraient chacune le chunk highlight). GARDE-RAILS (item 1.9, partie 1.2/1.3) :
  `archlint/no_film_reread_test.go` — zlib interdit dans `filmdec`/`replay`, `os.*` interdit dans
  `filmdec` hors allowlist datee (3 fichiers), aucun appel d'enveloppe depuis `replay` et
  `replaybuild` (liste FERMEE de 40 noms, parse go/ast et non grep : ces paquets CITENT les noms
  interdits dans leurs commentaires). Les trois regles verifiees DISCRIMINANTES par violation
  temporaire. Gate : `gofmt -l` vide, `go vet ./...` vide, `go build ./...` vert, suites completes
  `filmsource`/`filmdec`/`replay`/`replaybuild`/`archlint`/`replay-equiv` vertes (dont
  `TestEquivalenceMiniFilm`, qui compare aux empreintes FIGEES : identite prouvee en CI sur les
  sept familles que la mini-bobine supporte), `replayartifacts`/`cmd/levelup`/`cmd/zone-attribution`
  vertes, `golangci-lint` 0 issue sur les six paquets touches. Decouvertes N-M a N-R au §8 (trois
  changements de comportement declares, tous hors corpus). AUCUNE CUISSON LANCEE, rien committe :
  le harnais des 9 films est au pilote.
- 2026-09-02 (lot 1, item 1.4 + la moitie killcollector de 1.6, agent) — LE DECODEUR DE LA SOURCE
  DE DEGAT NE LIT PLUS LE DISQUE. `killsource.Decode(ctx, name, film *filmsource.Film, opts)` :
  `ChunkSource`, `MemoryChunks`, `DirChunks`/`dirChunks`, `inflate` et `splitPackets` SUPPRIMES
  (aucun appelant restant, tests compris) ; `loadFilm` ne fait plus que traduire le film charge
  (`Chunk(i)` -> `f.chunks`, `AllPackets` -> `f.packets`) et trier `t0`. LE FILTRE DE TYPE 7 EST
  LA CLE DE L'IDENTITE et il est documente sur place (`packetsOf`) : `filmsource` EMET le
  terminateur CHUNK_END (regle 3 de D3 revisee), l'ancien `splitPackets` s'arretait sur
  `taille <= 0` et ne l'emettait jamais ; le terminateur etant le DERNIER paquet de son chunk, le
  filtrer restitue l'ancien jeu jusqu'au rang `idx`. Verifie que le type 7 ne traverse AUCUN
  filtre aval : les trois consommateurs de `f.packets` selectionnent deja par type (0 pour `t0` et
  le scan, 2 pour la timeline, 12 pour les bots) et `feed.go` lit des CHUNKS, pas des paquets. LE
  REGISTRE RESTE LU PAR OCTETS DE CHUNK, PAS PAR PAQUETS : `newTimeline` lit `f.chunks[0]`, qui
  vaut `film.Chunk(0)` — la POSITION 0 de la source, exactement comme avant (`DirChunks` triait
  `chunk_00.bin` en tete, `FilmOf` place l'en-tete a l'index 0) ; aucun `Packets(0)` n'est
  demande. VERROU PROCESS INCHANGE : `filmdec.LockProcessDecode()` + `resetGlobals()` au meme
  endroit, avant `prepare` — seul le CHARGEMENT est sorti du verrou (il ne touche aucun global).
  APPELANTS : `replaybuild/kills.go` recoit le film que `BuildBytes` charge deja (plus aucun
  second chargement dans la cuisson ; `collecterEntreesCatalogue` echange `filmDir` contre
  `film`, 6 parametres comme avant, ordre des etapes observees intact —
  `replaybuild/observe_test.go` vert) ; `killcollector` charge le film UNE fois par passe
  (`ChunkSourceOf` -> `FilmOf`, `ChunkSourceForMatch` -> `FilmForMatch`, metadonnees
  POSITIONNELLES construites depuis les `haloclient.FilmChunk`) et le partage entre les morts ET
  les positions ; `cmd/killsource` charge par `filmsource.LoadDir` (le chronometre ne mesure plus
  que le decodage) ; six fichiers de test migres vers `filmsource.LoadDir`/`Load`. MOITIE
  KILLCOLLECTOR DE 1.6 : le PONT DISQUE de `positions.go` est SUPPRIME (`writeChunksToTempDir` et
  ses six tests) — les quatre balayages prennent le film deja charge (`ScanBipedPositions`,
  `ScanClockOrigin`, `ScanDeaths`, `ScanPlayerIndices`), soit quatre ecritures de fichiers, quatre
  relectures et quatre decompressions du film entier en moins par match. Le seul controle qui
  protegeait d'une position FAUSSE survit tel quel, en memoire : `refuserSequenceTrouee` (meme
  regle que `CountFilmChunks(dir) != maxIdx` — chunks de donnees 1..N tous presents et non vides),
  avec ses tests. Item 1.6 reste `[ ]` : sa moitie `cmd/*` appartient a l'agent parallele.
  IDENTITE PROUVEE PAR MESURE, pas par raisonnement (note N-T au §8) : 0 ecart sur 291 635 paquets
  et sur tous les chunks des quatre films de fixture. GATE : `gofmt -l` vide sur les paquets
  touches, `go build ./...` vert, `go test` vert sur `killsource` (goldens de la mini-bobine
  compris, INCONDITIONNELS), `replaybuild`, `killcollector` (dont `-tags=integration` avec les
  films reels, 72 s) et `archlint` ; `golangci-lint run` 0 issue sur les quatre paquets. DEUX
  GATES NON RENDUS, ET POURQUOI : (1) `go vet ./...` et `go test ./internal/analysis/replay/` ne
  COMPILENT PAS, uniquement a cause de l'item 1.5 en vol chez l'agent parallele (toutes les
  erreurs sont des `objectiveevents.FilmSource` / `StatRecordsCtx` dans `replay/*_test.go`,
  `objectiveevents/*_test.go` et `filmdec/navpoint_ti12_radial_test.go` — aucune ne cite un
  symbole de ce lot ; `go build ./...`, lui, est vert) ; (2) `gofmt -l` signale
  `cmd/zone-attribution/measure.go`, fichier du meme agent. DECOUVERTE ROUGE ANTERIEURE AU
  CHANTIER : N-S au §8 — `TestGoldenFilms` (gate `KILLSOURCE_FIXTURES`) echoue DEJA sur
  `900384f50`, a l'identique. Rien committe.
- 2026-09-02 (lot 1, item 1.5 + part CLI de 1.6) — `objectiveevents` NE LIT PLUS DE FILM. Les
  NEUF points d'entree prennent un `*filmsource.Film` deja charge (`Extract`, `CaptureBurstTimes`,
  `StatRecords`, `StatRecordsCtx`, `NamedEvents`, `CrossCheckNamedEvents`, `SlotIdentity`,
  `SlotIdentityResolved`, `SlotIdentityFromDeaths`) ; `decompressChunk`, `walkFrames`,
  `framePacket`, `FilmSource` et `ChunkMeta` sont SUPPRIMES — le troisieme inflate et le
  troisieme marcheur du depot. Il ne reste que `framesOf` (filtre de type 0 sur les paquets deja
  decoupes) et `manifestChunks`. LA QUESTION DU CHUNK 00 EST TRANCHEE SUR PIECES ET NON SUPPOSEE
  (note N-X au §8) : le manifeste le LISTE (1 380/1 380, `chunk_type` 1), `StatRecordsCtx` le
  consommait, et il le consomme encore — les en-tetes degeneres que la nouvelle grammaire ne
  franchit pas ne portaient que des trames VIDES, d'ou zero enregistrement des deux cotes.
  LE JEU DE CHUNKS CONSOMME EST PRESERVE PAR CONSTRUCTION : `manifestChunks` ne rend que les
  chunks DECRITS PAR LE MANIFESTE, comme l'ancienne boucle sur `src.Chunks()` — un film du cache
  (`7b0d89c4`) porte deux fichiers hors manifeste que le film charge expose et que l'ancienne
  source ne voyait pas (note N-Y ; critere mesure : `chunk_type` vaut 1, 2 ou 3 sur les
  1 380 manifestes, jamais 0). IDENTITE MESUREE, DEUX FOIS, PAR EMPREINTES (note N-Z) : sur 10
  films (les 9 du corpus + `7b0d89c4`), les enregistrements, les bursts de capture et le pied de
  film sont identiques a l'octet entre le code d'avant et le code migre. `filmcache` N'IMPORTE
  PLUS `objectiveevents` : il expose `Meta() []filmsource.ChunkMeta`, implemente
  `filmsource.Source` (assertion de compilation, qui REMPLACE `filmcache_guard_test.go` et ses
  trois derogations devenues caduques — note N-AA) et gagne `LoadFilm`/`LoadFilmDir`, chemin
  unique du cache vers un film. PART CLI DE 1.6 : `cmd/zone-attribution`, `cmd/statnames-sweep`,
  `cmd/oddball-terrain` et `cmd/diag_weapons_v3` chargent le film UNE fois par film et le passent
  a tous leurs consommateurs (plus un seul `ScanFilmDeaths(dir)` sous un `StatRecordsCtx`) ; la
  moitie `killcollector` etant rendue par l'agent 1.4, l'item 1.6 est CLOS. ~40 fichiers de test
  migres (`replay`, `filmdec`, `objectiveevents`) : les producteurs de source deviennent des
  chargeurs de film (`p2aBobine`, `zcOpenFilm`, `newDiskFilm`, `objDiskFilm = filmsource.Film`,
  `filmRepete`), `p2aSource` supprime faute d'appelant. GARDE-RAIL (item 1.9) :
  `objectiveevents` rejoint `filmdec` et `replay` dans `archlint/no_film_reread_test.go` (zlib
  interdit) — regle verifiee DISCRIMINANTE par un import temporaire. Notes N-X a N-AD au §8, dont
  un changement de comportement declare et inobservable (N-AC) et une dette de doc laissee au
  lot 6 (N-AD, les copies de mesure de `cmd/replay-equiv` n'ont plus d'original). GATE :
  `gofmt -l` vide, `go vet ./...` vide, `go build ./...` vert, suites completes vertes sur
  `objectiveevents` (avec `FILM_CACHE_ROOT`, verite terrain rejouee), `filmcache`, `replaybuild`,
  `replay`, `filmdec`, `archlint` et `./cmd/...`, `golangci-lint run` 0 issue sur les dix paquets
  touches. Seuils : `statborg.go` 584 L (585 avant — la dette PREEXISTANTE ne croit pas),
  tous les autres fichiers touches sous 500 L. AUCUNE CUISSON LANCEE, rien committe.
- 2026-09-03 (lot 1, items 1.7 a 1.10, agent) — LES ENVELOPPES SONT INVENTORIEES, LE ZERO DISQUE
  EST PROUVE, LES GARDE-RAILS SONT A L'ETAT FINAL DU LOT. Rien committe.
  **1.7 — INVENTAIRE DES 40 FORMES `dir`.** Trente-neuf se declarent « ENVELOPPE D2 » dans leur
  commentaire (grep du marqueur, pas de la signature : cinq d'entre elles ont une signature
  multi-lignes et quatre nomment leur parametre `filmDir` — un grep sur `(dir string` en aurait
  manque neuf) ; la quarantieme, `GroundWeaponPositions`, est une forme `dir` qui DELEGUE aux
  deux autres et ne portait pas le marqueur : il lui a ete ajoute. CHACUNE a au moins un
  appelant : AUCUNE suppression. Tous les appelants sont des `_test.go` de `analysis/filmdec` ou
  `analysis/replay`, plus quatre delegations internes a `filmdec` (`keyframe_entity_queue.go` x3,
  `keyframe_ground_weapons.go` x1) ; ZERO appelant de production — rien dans `replaybuild`,
  `sync`, `api`, ni sous `cmd/`. La liste fermee d'`archlint`
  (`enveloppesInterditesEnProduction`) coincide EXACTEMENT avec ces 40 noms (verifie par
  comparaison d'ensembles, aucun nom en trop ni manquant). HORS MODULE, un dernier appelant
  existe et il est nomme pour que l'inventaire soit complet :
  `.ai/V7.5/outillage/precision_projectiles/tmp_projorig/scan.go` (`ScanFilmGrenadeThrows`,
  `ScanFilmProjectiles`) — outillage de recherche, hors `apps/go-api`, donc ni compile par
  `go build ./...` ni couvert par les garde-rails ; aucune consequence sur la production.

  | Enveloppe D2 | Appelants | tests `filmdec` | tests `replay` | Hors test |
  |---|---|---|---|---|
  | `ScanFilmAbilityRanks` | 6 | 2 | 4 | aucun |
  | `ScanFilmBipedPickups` | 19 | 2 | 17 | aucun |
  | `ScanFilmBipedPositions` | 47 | 10 | 37 | aucun |
  | `ScanFilmCamoStates` | 1 | 0 | 1 | aucun |
  | `ScanFilmCarrierMarks` | 1 | 0 | 1 | aucun |
  | `ScanFilmClockOrigin` | 17 | 0 | 17 | aucun |
  | `ScanFilmDeaths` | 38 | 0 | 38 | aucun |
  | `ScanFilmEquipmentChanges` | 6 | 2 | 4 | aucun |
  | `ScanFilmEquipmentCreations` | 2 | 0 | 2 | aucun |
  | `ScanFilmEquipmentCreationsForBand` | 6 | 4 | 2 | aucun |
  | `ScanFilmEquipmentPlacements` | 11 | 3 | 8 | aucun |
  | `ScanFilmEquipmentState` | 3 | 2 | 1 | aucun |
  | `ScanFilmFireEvents` | 13 | 2 | 11 | aucun |
  | `ScanFilmGrappleReads` | 2 | 0 | 2 | aucun |
  | `ScanFilmGrenadeThrows` | 5 | 0 | 5 | aucun |
  | `ScanFilmGroundWeaponCreations` | 1 | 0 | 1 | aucun |
  | `ScanFilmGroundWeaponCreationsForBand` | 6 | 0 | 6 | aucun |
  | `ScanFilmHeldWeaponChanges` | 13 | 5 | 8 | aucun |
  | `ScanFilmInventoryDeltas` | 4 | 1 | 3 | aucun |
  | `ScanFilmKeyframeGroundWeapons` | 7 | 2 | 5 | aucun |
  | `ScanFilmKeyframeInventory` | 11 | 0 | 11 | aucun |
  | `ScanFilmKeyframeLoadouts` | 14 | 4 | 10 | aucun |
  | `ScanFilmManagedProperties` | 11 | 0 | 11 | aucun |
  | `ScanFilmNavpointRadial` | 4 | 4 | 0 | aucun |
  | `ScanFilmObjectives` | 4 | 2 | 2 | aucun |
  | `ScanFilmPlayerIndices` | 21 | 0 | 21 | aucun |
  | `ScanFilmProjectiles` | 4 | 1 | 3 | aucun |
  | `ScanFilmUnitEquipment` | 3 | 1 | 2 | aucun |
  | `ScanFilmWorldObjectKeyframes` | 9 | 2 | 7 | aucun |
  | `ScanFilmWorldObjects` | 12 | 8 | 4 | aucun |
  | `ScanFilmWorldObjectsForBand` | 3 | 0 | 3 | aucun |
  | `ScanFilmZoomEvents` | 4 | 0 | 4 | aucun |
  | `CalibrateMPPWidths` | 3 | 3 | 0 | aucun |
  | `DetectI0Layout` | 28 | 24 | 3 | definition exclue |
  | `EquipmentArchetype` | 4 | 4 | 0 | aucun |
  | `GroundWeaponSlotBand` | 5 | 0 | 4 | +1 delegation `GroundWeaponPositions` |
  | `GroundWeaponPositions` | 1 | 1 | 0 | aucun |
  | `WorldObjectPositionsForBand` | 7 | 0 | 6 | +1 delegation `GroundWeaponPositions` |
  | `ReadFilmChunk` | 177 | 120 | 54 | +3 `keyframe_entity_queue.go` |
  | `CountFilmChunks` | 130 | 72 | 55 | +3 `keyframe_entity_queue.go` |

  **1.8 — TEST STRUCTUREL ZERO DISQUE** (`replay/zero_disque_test.go`, deux tests). Le film est
  charge par `os.ReadFile` des trois chunks puis `filmsource.Load(MemoryChunks, meta)` avec les
  Index REELS 1, 2, 3 — jamais `LoadDir` — et le decodage tourne apres `t.Chdir` vers un
  repertoire temporaire VIDE. METHODE, en trois pieces mesurees plutot qu'affirmees : (1) un
  CONTROLE verifie que la mini-bobine n'est plus atteignable par son chemin relatif depuis ce
  repertoire (sans lui, « repertoire vide » ne serait qu'une intention) ; (2) `BuildFromFilm`
  doit rendre l'erreur EXACTE « aucun slot biped (ti=35) dans les keyframes du film » — toute
  autre erreur, au premier chef une erreur d'ouverture, echoue le test ; (3) le repertoire est
  verifie vide apres coup. Comme `BuildFromFilm` s'arrete a son PREMIER balayage sur cette
  bobine, un SECOND test rejoue depuis le meme repertoire vide les SEPT familles de la liste
  fermee D4c sous leur forme FILM : elles doivent REUSSIR, et leur succes est la preuve qu'elles
  decodent entierement en memoire. DISCRIMINANCE VERIFIEE : une relecture residuelle inseree
  temporairement dans `BuildFromFilm` (chemin relatif) fait rougir le test 1, la meme dans
  `ScanDeaths` fait rougir le test 2 ; les deux sondes ont ete retirees (fichiers restaures a
  l'octet). LIMITE ECRITE dans l'en-tete : un acces par chemin ABSOLU echapperait a la mesure —
  elle est bornee par la signature de `BuildFromFilm` (aucun chemin) et par la regle 2
  d'`archlint`.
  **1.9 — CE QUI MANQUAIT AUX GARDE-RAILS.** Les agents 1.2-1.5 avaient pose : zlib interdit
  dans `filmdec`/`replay`/`objectiveevents`, `os.*` interdit dans `filmdec` hors allowlist
  datee, enveloppes interdites dans `replay`/`replaybuild`, `filmsource` feuille. TROIS AJOUTS :
  (a) `killsource` rejoint les paquets sans inflate — l'item 1.4 lui a retire le sien, et rien
  ne l'empechait de revenir ; (b) les paquets de PRODUCTION de la regle des enveloppes passent
  de 2 a 7, au site pres de D2 (`objectiveevents`, `killsource`, `sync/killcollector`,
  `api/wire`, `cmd/zone-attribution`) ; (c) REGLE 4 NEUVE, `TestAllowlistZlibFermee` : les
  importateurs de `compress/zlib` de tout `apps/go-api` (hors `_test.go`) sont une liste FERMEE,
  verifiee DANS LES DEUX SENS — un site en trop echoue, et une entree MORTE echoue aussi (une
  allowlist qui garde des entrees mortes ne mesure plus rien). Elle compte NEUF sites la ou le
  plan en annoncait sept : cf. note N-AE. Les trois regles sont verifiees DISCRIMINANTES par
  violation temporaire (import zlib dans `killsource`, appel d'enveloppe dans `killcollector`,
  entree morte dans l'allowlist), toutes restaurees.
  **1.10 — RATCHET DES VARIABLES DE PAQUET** (`archlint/filmdec_package_vars_test.go`).
  Comptage par `go/ast` des NOMS declares par un `var` de niveau paquet dans les fichiers
  non-test de `filmdec` : **113** (98 declarations, 109 specs, 0 identifiant blanc — un bloc
  `var (a; b)` vaut deux, c'est deux morceaux d'etat). Gele a 113, justification datee du
  2026-09-03 renvoyant a D10 : le chantier ne de-globalise pas, mais il ne laisse pas l'etat
  croitre. Le test echoue si le compte CROIT (avec le detail par fichier) ; une BAISSE passe et
  journalise le compte a inscrire, pour que le resserrage soit conscient. Discriminance verifiee
  dans les deux sens.
  GATE : `gofmt -l` vide (module entier), `go vet ./internal/analysis/... ./internal/archlint/`
  vide, `go build ./...` vert, `go test ./internal/analysis/replay/ ./internal/analysis/filmdec/
  ./internal/archlint/ -count=1` vert (59,6 s / 0,8 s / 6,5 s), `golangci-lint run` 0 issue sur
  `analysis/replay` et `archlint`. Decouvertes N-AE a N-AH au §8. AUCUNE CUISSON LANCEE.

### Lot 2 — Contexte du film partage (2026-09-03) — CLOS cote code, equivalence a la charge du pilote

**CE QUE LE LOT FERME.** Le lot 1 avait supprime les ~36 relectures du film ; restait le second
etage du meme defaut. Sur le film DEJA CHARGE, chaque balayage recalculait pour son compte les
trois memes derivations, qui ne dependent pourtant que du film. Comptage sur le chemin de
production de `BuildFromFilm`, AVANT / APRES :

| Derivation | Avant (par cuisson) | Apres | Qui recalculait |
|---|---|---|---|
| `bipedSlotBand` (tous chunks) | 8 | 2 | positions, ramassages natifs, + les 6 canaux delta |
| `DetectI0LayoutOf` (6 chunks bit a bit + sa bande) | 6 | 1 | held_weapon, inventory_delta, ability_rank, equipment_changes, camo, grapple |
| `ParseRegistryChunk` (chunk_00) | 10 a 12 | 1 | 6 x biped, calibration MPP, creations ti=37 (x2), ti=42, + ti=13 et ti=12 selon le mode |
| `worldObjectSlotBand(ti=37)` dans les poses | 2 | 1 | `ScanEquipmentPlacements` puis `ScanWorldObjects` (item 2.2) |

Le « 2 » de la bande bipede est `ScanBipedPositions`, hors perimetre et allowliste (note N-AI).

**PAR BALAYAGE (item 2.1).** `ScanHeldWeaponChanges`, `ScanInventoryDeltas`, `ScanAbilityRanks`,
`ScanEquipmentChanges` (par `walkAbilityEmissions`), `ScanCamoStates`, `ScanGrappleReads` :
chacun faisait `FilmChunkNumbers` + `bipedSlotBand` + `DetectI0LayoutOf` + `bipedArchetype` ;
chacun lit desormais `fc.ChunkNumbers()` + `fc.BipedSlots()` + `fc.I0Layout()` +
`fc.bipedArchetype()`. `ScanBipedPickups` : `bipedSlotBand` -> `fc.BipedSlots()`. L'ORDRE des
gardes et les MESSAGES d'erreur sont conserves mot pour mot — c'est ce qui fait que
`replay/zero_disque_test.go`, qui exige l'erreur EXACTE de la mini-bobine, passe sans modification.

**POURQUOI LA MEMOISATION EST PARESSEUSE.** Un constructeur qui calculerait tout d'avance
deplacerait le premier calcul AVANT le premier balayage — donc avant l'installation des largeurs
d'axe et avant le demarrage de l'horloge des etapes — et ferait travailler un film qui echoue des
les positions. Paresseux, le premier calcul a lieu exactement la ou il avait lieu avant (le
premier balayage qui en a besoin) ; les suivants le lisent. C'est aussi ce qui garantit « jamais
pire qu'avant » aux enveloppes D2, qui ouvrent leur propre contexte a chaque appel.

**POURQUOI `NewFilmContext` NE REND PAS D'ERREUR** (ecart assume par rapport a la signature
suggeree `(*FilmContext, error)`). Les trois derivations echouent sur des films LEGITIMES : une
bobine partielle n'a pas de `chunk_00`, un film trop court ne donne pas trois frontieres nettes
dans i0, un film nil arrive quand `replaybuild.chargerFilm` a echoue. Chaque balayage rend
aujourd'hui SON message a SON etape ; refuser au constructeur changerait ces messages ET l'etape
a laquelle la cuisson s'arrete. Le contexte MEMORISE donc l'echec et chaque accesseur le rejoue a
l'identique, autant de fois qu'on le lui demande. Meme raison pour les champs prives : un layout
et un registre se lisent avec l'erreur qui va avec.

**LA PREUVE D'IDENTITE, EN DEUX PIECES.**
  1. RAISONNEMENT VERIFIE SUR PIECES : les trois calculs sont des fonctions PURES des octets du
     film. `bipedSlotBand` -> `WalkKeyframeWorld` (maps locales, aucun global) ;
     `DetectI0LayoutOf` -> `matchBipedHeaderRaw` + `readBitsAt` + `profileI0` (constantes et
     arithmetique, aucun hook) ; `ParseRegistryChunk` -> `parseRegistry` (lecture d'octets ; son
     seul effet de bord est `warnUnknownRegistry`, qui journalise UNE fois par empreinte, a
     l'echelle du process). Aucun global mute entre les etapes 4 et 11 n'entre dans ces trois
     calculs — les balayages n'y installent que des hooks, restaures par `defer`. Les entrees
     etant identiques (`FilmChunkNumbers(film)` dans les deux mondes), les sorties le sont.
  2. TEST DIFFERENTIEL : `filmdec/film_context_test.go` compare, sur le MEME film, ce que le
     contexte rend et ce que le recalcul direct rend — numeros de chunks, bande (slot par slot),
     decoupage (valeur ET erreur), registre (empreinte + composants et drapeaux, archetype par
     archetype), plus le second appel (memoisation : meme valeur, et la MEME instance de
     registre). Deux etages : la MINI-BOBINE en CI, qui n'a ni registre ni slot bipede et prouve
     donc l'egalite des ECHECS (les six accesseurs d'archetype rendent tous `ErrNoRegistryChunk`),
     et un VRAI film sous `FILM_CONTEXT_FILM`, qui prouve l'egalite des valeurs NON VIDES et
     refuse de passer si le film est trop pauvre pour prouver quoi que ce soit. Passe sur
     `000d5950` (10,8 s), `7344d24f` (18,6 s) et `60ae07c4` (21,0 s) — dont Live Fire, ou le
     decoupage auto-detecte est justement celui que le lot 3 remplacera.

**SEUIL DE FICHIER TENU DANS LE LOT MEME.** Le regroupement de 4.4 portait
`objectiveevents/named.go` de 441 a 505 lignes — au-dela du seuil de 500. Depassement CREE par
ce lot, donc corrige dedans (lecon du lot 0, §8) : `named_series.go` recoit tout ce qui va d'une
liste d'enregistrements a une suite cumulee (`rawSeriesByKey`, `seriesBySlot`,
`rawSeriesByRound`, `cumulateRounds`, `sortedRounds`, `incrementTimes`), scission PURE,
commentaires intacts -> 339 + 178 lignes.

**GATE.** `gofmt -l .` vide (module entier) · `go vet ./...` vide · `go build ./...` vert ·
`go test ./internal/analysis/filmsource/ ./internal/analysis/filmdec/ ./internal/analysis/replay/
./internal/replaybuild/ ./internal/archlint/ -count=1` vert (0,25 s / 1,4 s / 39,1 s / 0,7 s /
17,1 s) · `go test ./internal/games/halo_infinite/film/killsource/ ./internal/sync/replayartifacts/
./cmd/replay-equiv/ -count=1` vert · `golangci-lint run ./internal/analysis/filmdec/...
./internal/analysis/replay/... ./internal/archlint/...` : 0 issue. Decouvertes N-AI a N-AM au §8.
AUCUNE CUISSON LANCEE — l'equivalence 9 films et les temoins §6 sont a la charge du pilote.

### Lot 4 — Boucles chaudes (2026-09-03) — CLOS cote code, equivalence a la charge du pilote

**LE PROFIL A ETE LU AVANT D'OPTIMISER, ET IL A DEPLACE LE LOT.** Top 10 de
`tmp/L1_01e1f945.cpu.prof` (binaire `tmp/replay-build.exe`, 152,35 s d'echantillons, lot 1) :

| flat | cum | fonction |
|---|---|---|
| 88,97 s (58,40 %) | 89,00 s | `filmdec.kfReadBits` |
| 31,75 s (20,84 %) | 31,88 s | `weaponv3.bitReader.bit` |
| 8,15 s (5,35 %) | 40,07 s | `weaponv3.bitReader.readBits` |
| 4,62 s (3,03 %) | 94,77 s | `filmdec.kfScanNext` |
| 2,11 s (1,38 %) | 2,11 s | `filmdec.PeekBits` |
| 1,14 s (0,75 %) | 27,93 s | `filmdec.kfValidAnchor` |
| 1,07 s (0,70 %) | 1,68 s | `replay.nearestFreeEnd` |
| 1,02 s (0,67 %) | 2,28 s | `runtime.mapaccess1_fast32` |
| 1,00 s (0,66 %) | 3,27 s | `filmdec.matchBipedHeader` |
| 0,94 s (0,62 %) | 0,94 s | `filmdec.readBitsAt` |

Deux enseignements, tous deux contraires a ce que le plan supposait (decouverte N-AO) :
`readBitsAt` et `BitReader.ReadBits` — les deux primitives NOMMEES par l'item 4.2 — pesent 0,62 %
et rien du tout ; le temps est chez `kfReadBits` (58 %, appelee pour CHAQUE position de bit du
payload d'image-cle par `kfScanNext`) et chez le lecteur de `weaponv3` (26 % en cumul, entierement
sous `ResolveXuidToPI`, c'est-a-dire le balayage `playerIndices` que `tmp/refL2_01e1f945.log`
chronometre a 38,47 s sur ~96 s de cuisson). Le lot a donc traite les quatre primitives, en
donnant la priorite a celles que la mesure designe.

**4.2 — LECTURE PAR MOT DE 64 BITS.** `filmdec/bits_word.go` porte `wordBitsAt(buf, pos, n)`
(chemin rapide : un `binary.BigEndian.Uint64` + deux decalages ; `wordBitsAtEdge` pour la queue de
tampon et le champ a cheval sur un neuvieme octet). Quatre primitives y branchent leur chemin
rapide — `BitReader.ReadBits`, `kfReadBits`, `readBitsAt`, `PeekBits` — et CHACUNE garde sa boucle
bit a bit d'origine pour le domaine ou les deux ne coincident pas. C'est la clause de D6 :
**la semantique hors tampon est preservee PAR FONCTION**, `readBitsAt` PANIQUE, les trois autres
rendent des zeros, et une position negative ou une largeur > 64 (N-AN : elle existe reellement)
repasse toujours par l'ancienne boucle.

Preuve : `filmdec/bits_word_test.go` et `weaponv3/bits_word_test.go` embarquent une COPIE DE
REFERENCE de chaque implementation d'avant et l'opposent a la nouvelle sur des tampons aleatoires
a graine fixee (tailles 0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 23, 31, 64, 65 octets), toutes les
largeurs 0..64, et les positions autour de chaque frontiere d'octet, de mot et de fin de tampon —
positions negatives et lectures entierement hors tampon comprises. Deux tests dedies verrouillent
les DEUX semantiques de bord : `TestReadBitsAtPanicsOutOfBufferLikeReference` (la panique tombe
exactement la ou la reference paniquait, ni plus tot ni plus tard) et
`TestReadBitsZeroPadsOutOfBuffer`. Un troisieme, `TestReadBitsWordWideMatchesReference`, couvre
les largeurs 65..200 (N-AN).

Un second geste, PUR, sur la boucle d'ancrage : `kfScanNext` lisait les 32 bits d'identifiant pour
son test de sentinelle puis `kfValidAnchor` LES RELISAIT. `kfAnchorFromID` prend l'identifiant
deja lu ; memes gardes, meme logique, `id` etant une fonction pure de (buf, q).

Micro-mesure (tampon aleatoire 256 Kio, fenetre 120 000 bits, Ryzen 7 9850X3D) :
`kfScanNext` **5,23 ms -> 0,57 ms (9,2x)** ; `readBitsAt` sur 13 bits 11,99 us -> 7,86 us (1,5x).

**4.6 — `ResolveXuidToPI` (poste n1).** Le balayage cherchait un motif de 64 bits en relisant 64
bits UN PAR UN a chaque position de bit du chunk, pour chaque xuid du roster, et
`ScanPlayerIndices` le fait sur TOUS les chunks de replication (28 sur `01e1f945`). `findPattern64`
lit desormais un mot big-endian par OCTET et en deduit les huit decalages avec le seul octet
suivant — meme ordre de parcours, donc la MEME premiere position, donc le meme index publie.
Micro-mesure sur un chunk de 1 Mio et 8 xuids absents (pire cas, balayage complet) :
**2 459,6 ms -> 25,3 ms, 97x**. Le differentiel `TestFindPattern64MatchesReference` implante le
motif a chaque decalage de bit 0..7 et a plusieurs profondeurs, dont la position 0 — celle ou la
relecture des 5 bits de `player_index` RECULE SOUS ZERO et doit y lire des zeros.

**4.1 — BANDE DE SLOTS DENSE.** `filmdec/slot_band_dense.go` : `SlotBand` = un booleen par slot
(domaine 8 192 = 2^13), construit UNE fois par balayage. `matchBipedHeaderRaw` l'interrogeait
par `map[uint32]bool` a CHAQUE bit candidat. Le type est un STRUCT et non un `[]bool` nu, et c'est
le point de methode : `len(bande)` veut dire « combien de slots » dans tout le depot, et vaudrait
8 192 sur une tranche dense — le struct fait ECHOUER LA COMPILATION sur chaque usage de forme
« map » (`len`, indexation, `range`) au lieu de le laisser mentir. C'est le compilateur qui a
trouve les 30 sites a convertir, pas une relecture. Perimetre : le chemin BIPEDE seul
(`bipedSlotBand`, `fillSlotBand`, `FilmContext.BipedSlots`, `matchBipedHeader(Raw)`,
`collectI0Samples`, `ScanBipedRecords` et les six repliques). Le chemin OBJET DU MONDE reste sur
la map : decouverte N-AP, avec le chiffre qui justifie l'abstention.

**4.3 — DEUX ALLOCATIONS PAR RECORD SUPPRIMEES.** `readBodyVitalityComponent` et
`readShieldVitalityComponent` allouaient chacun un `NewBitReader` par record bipede ; ils prennent
desormais le lecteur que `ScanBipedRecords` alloue UNE fois par payload et le repositionnent par
`SetBitPos`. `ascendingFromZero` valide la liste d'index dans un tampon de PILE
(`[bipedMaxMaskCnt]int`) et n'alloue la tranche qu'une fois la liste acceptee — le test echoue sur
l'immense majorite des positions candidates, donc l'allocation etait jetee a chaque rejet.

**4.4 — `NamedEventsFrom` EN UNE PASSE.** `rawSeriesByKey` groupe les emissions de TOUS les
emplacements non redondants en un seul parcours de `recs`, et `RealRounds` est calcule UNE fois :
la forme d'avant appelait `seriesBySlot` par emplacement, donc `rawSeriesByRound` ET `RealRounds`
une fois chacun par emplacement — seize marches completes pour la table CTF. Les filtres, leur
ordre et l'ordre d'insertion par (emplacement, slot, manche) sont conserves mot pour mot. La cle
de tri gagne `Comp` et `Side` (plan §9 R-8) : sur les tables actuelles cela ne change RIEN, parce
qu'un nom de statistique n'y vient que d'un emplacement — mais c'etait une coincidence, et
`TestSortNamedEventsOrdreTotal` la remplace par une garantie. Identite locale prouvee par
`TestNamedEventsFromOnePassMatchesReference` : la forme d'AVANT est recopiee dans le `_test` et
opposee a la nouvelle sur un corpus construit qui touche chaque filtre (deux manches, slots
d'equipe, emission negative a -115, score de mode hors domaine A=66/B=16635, manche 9 fortuite),
sur les cinq familles d'objectif.

**CE QUI N'A PAS ETE FAIT, ET POURQUOI.** L'item 4.5 (mesure des quatre temoins + profil) est
statue `[!]` : aucune cuisson n'a ete lancee par cet agent, le harnais est au pilote. Les
micro-mesures Go ci-dessus en tiennent lieu a titre indicatif et NE VALENT PAS la mesure §6.
La bande d'objets du monde n'a pas ete densifiee (N-AP, 0,8 % du profil contre douze signatures
exportees). La fenetre glissante de `kfScanNext` n'a pas ete posee (N-AQ).

**GATE.** `gofmt -l .` vide (module entier) · `go vet ./...` vide · `go build ./...` vert ·
`go test ./internal/analysis/filmsource/ ./internal/analysis/filmdec/ ./internal/analysis/replay/
./internal/analysis/objectiveevents/ ./internal/replaybuild/ ./internal/archlint/ -count=1` vert
(0,52 s / 0,91 s / 11,20 s / 0,39 s / 0,77 s / 6,57 s) · `go test
./internal/games/halo_infinite/film/killsource/ ./internal/analysis/weaponv3/ -count=1` vert ·
`go test ./...` (module entier) vert · `golangci-lint run` : 273 issues, soit la baseline —
ZERO nouvelle issue (une `unparam` introduite en cours de route a ete corrigee avant cloture).
Goldens inchanges. Decouvertes N-AN a N-AR au §8. AUCUNE CUISSON LANCEE : l'equivalence 11 films
et les temoins §6 restent a la charge du pilote, et c'est eux qui prononcent le refacto PUR.
