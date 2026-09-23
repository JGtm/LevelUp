# PREPARATION M4 — la publication (lots 4.1, 4.2, 4.3, 4.4)

> Note d'ANALYSE SUR PIECES, sans production : aucun fichier Go modifie, aucun decodage, aucun
> artefact cuit. Elle prepare les quatre lots du jalon M4 du `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`
> (items 4.1.1 a 4.4.2) comme `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` a prepare les pas 4 a 6.
>
> Base : `24b67e339`, branche `feat/decfilm-m4prep`, worktree
> `C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-decfilm-m4prep`. Date : 2026-09-17.
>
> Sources lues en entier : le plan (section M4, §1.3 D6/D11/D12, §1.4 V1-V15), l'architecture
> cible §11 et §12 (`.ai/ARCHITECTURE_CIBLE_DECODEUR_FILM_2026-09-12.md`), l'ADR
> `docs/adr/0034-film-decoder-profile-and-layers.md` (D-6, D-7, D-8), la note de preparation M2
> (§3.2 types traversants, §3.3 `coverage.*`, §4 durees) et le brouillon
> `.ai/BROUILLON_CHRONIQUE_V61_2026-09-17.md`.
>
> Les chemins Go sont relatifs a `apps/go-api/`, les chemins web a `apps/web/src/`.
> Toutes les sorties de commande sont COLLEES telles quelles.

---

## 0. L'etat mesure a l'entree

### 0.1 Le parc, la cuisson, les fixtures

```
$ ls data/cache/film_chunks/ | wc -l
1386

$ ls -la data/cache/replays/halo_infinite/ | awk '{s+=$5; n++} END {print "n="n, "total="s, "moy="s/n}'
n=144 total=172768911 moy=1.19978e+06

$ ls data/cache/replays/halo_infinite/ | wc -l
141

$ ls -la apps/go-api/internal/games/halo_infinite/film/replay/testdata/inputs_*.bin.gz
-rw-r--r-- 1 Guillaume 197121  982629 inputs_000d5950.bin.gz
-rw-r--r-- 1 Guillaume 197121 1732074 inputs_111fa685.bin.gz
-rw-r--r-- 1 Guillaume 197121 1601980 inputs_11de8353.bin.gz
-rw-r--r-- 1 Guillaume 197121 1632276 inputs_60ae07c4.bin.gz
-rw-r--r-- 1 Guillaume 197121 1081872 inputs_a521164d.bin.gz
-rw-r--r-- 1 Guillaume 197121  612528 inputs_bcb6d393.bin.gz
-rw-r--r-- 1 Guillaume 197121 1802250 inputs_e5adf7b2.bin.gz
-rw-r--r-- 1 Guillaume 197121 1603591 inputs_fb1a1a72.bin.gz
```

| Grandeur | Mesure | Consequence pour M4 |
|---|---:|---|
| Films au cache local | **1 386** | le denominateur de toute recuisson et de tout parc de faits |
| Artefacts cuits au cache local | **141** (+3 `.derived`) | 172 768 911 o, **1,20 Mio en moyenne** |
| Jeu des 8 fixtures d'entrees | **11 049 200 o** | soit **1,38 Mio par film compresse** (le budget declare 11 044 446 o au 2026-09-14, plafond 12 Mio) |
| Un artefact reel (`01e1f945.json`) | 1 950 058 o | 36 cles racine servies, 113 traces, 29 553 points, 2 139 tirs, 251 armes au sol |

**Extrapolation a retenir avant d'ecrire une ligne de 4.1** : a 1,38 Mio de faits par film,
le parc entier pese **1 386 x 1,38 Mio ~ 1,9 Gio** de faits persistes, contre ~1,6 Gio
d'artefacts si le parc etait cuit en entier. La question de volumetrie n'est PAS theorique ;
elle est en §7, question D1.

### 0.2 Les revisions et versions en vigueur sur la base

| Revision / version | Valeur | Emplacement |
|---|---|---|
| `SchemaVersion` (document cuit) | **60** | `internal/games/halo_infinite/film/replay/document.go:48` |
| `GrammarRev` | golden `filmdec/testdata/grammar_rev.golden` | `filmdec/grammar_rev.go` |
| `KillSourceDecoderRev` | `killsource-2026-09-16.2` | `sync/killcollector/killsource_decoder_rev.go` |
| `MIN_RENDERABLE_SCHEMA_VERSION` (web) | **27** | `features/match-replay/model/replaySchemaStatusLogic.ts` |
| Fixtures de contrat web | schema **60**, 8 builds | `features/match-replay/test/fixtures/go/manifest.json` |
| Champs racine du document | **58** cuits, **58** servis | `film/replay/document.go`, `domain/replaydoc/document.go` |

Apres 2.6 : `SchemaVersion` = **61**, quatre revisions (`source.Rev`, `profile.Rev`,
`grammar.Rev`, `facts.Rev`) et le bloc publie `coverage.decoder = {grammarRev, factsRev, build}`
(arbitrages V15, questions 11, 12, 14, 15, 16). **4.2 herite donc d'une demi-marche deja
posee** : la note M2 §3.3 le dit explicitement, « 2.6.3 est la premiere moitie de 4.2.1 ».

---

## 1. Lot 4.1 — Les faits persistes par film avec leur revision

### 1.1 (a) Ce que `inputs_*.bin.gz` serialise AUJOURD'HUI

#### Le fichier, son format, son en-tete

| Question | Reponse mesuree |
|---|---|
| Nom | `internal/games/halo_infinite/film/replay/testdata/inputs_<short8>.bin.gz` |
| Nombre | **8**, un par build du jeu de reference (`goldenBuilds()`) |
| Compression | `compress/gzip` sur un flux binaire maison |
| Magie / version | `const goldenInputsMagic = "REPLAYINPUTS22\n"` (`golden_inputs_test.go:220`) — **v22**, montee au lot 1.9.1 le 2026-09-15 |
| Primitives | varint (`binary.AppendUvarint` / `AppendVarint`), `float32` LE, chaines prefixees, octet booleen — `gwriter` / `greader`, `golden_inputs_codec_test.go` |
| Delta-codage | horodatages globaux et coordonnees par slot ; **l'ordre d'origine est preserve** (`BuildFromPositions` numerote les traces dans l'ordre de premiere apparition des slots) |
| En-tete actuel | `film` (short8), **`mapModule`**, `axisW[3]`, `layoutDetected`, `inventoryDeltaAmmoRefused`, `filmMajorVersion` (presence + valeur), `filmClockOriginUS` |
| Erreurs typees | `errGoldenInputsCarte` (module de carte different), `errGoldenInputsDecoupage` (decoupage d'i0 en contradiction avec le catalogue) |
| Plafond | `goldenInputsBudget = 12 << 20`, teste par `TestGoldenInputsTiennentDansLeBudget` |

L'en-tete porte deja le principe que 4.1.1 demande : **une cle de relecture, et un refus
explicite quand elle ne correspond pas**. Le commentaire du fichier l'ecrit noir sur blanc :

> « goldenInputsMagic identifie le format et sa version. Un fixture d une autre version est une
> ERREUR, jamais une lecture "au mieux" : un decodage decale rendrait des chiffres plausibles. »

Et sur la carte :

> « LE MODULE DE LA CARTE OUVRE LE BLOB (lot 0.D.3 bis). Les positions y sont des QUANTA : sans
> l entree de catalogue qui les a produites, elles ne se dequantifient pas — et avec la MAUVAISE,
> elles se dequantifient en coordonnees FAUSSES, pas approximatives. »

#### Qui l'ecrit, qui le lit

**Aucun code de PRODUCTION ne le touche.** C'est le constat central de 4.1 : le codec entier vit
dans des fichiers `_test.go`.

```
$ wc -l film/replay/{film_inputs.go,golden_inputs_test.go,golden_inputs_{encode,decode,codec,budget,canaux,fidelite}_test.go,build_from_film.go}
   199 film_inputs.go                 <- PRODUCTION (le type FilmInputs, et rien d'autre)
   434 golden_inputs_test.go          <- test : harnais, magie, chargement, regeneration
   373 golden_inputs_encode_test.go   <- test : l'encodeur
   369 golden_inputs_decode_test.go   <- test : le decodeur
   473 golden_inputs_codec_test.go    <- test : les primitives (gwriter/greader, sections partagees)
    72 golden_inputs_budget_test.go   <- test : le plafond
   489 golden_inputs_canaux_test.go   <- test : les six canaux entres au lot 1.0
   118 golden_inputs_fidelite_test.go <- test : le codec transporte-t-il tout ?
   188 build_from_film.go             <- PRODUCTION (scanFilmInputs + BuildFromFilm)
```

| Role | Qui | Ou |
|---|---|---|
| PRODUIT le type | `scanFilmInputs` (39 etapes observees) | `film/replay/build_from_film.go` |
| CONSOMME le type | `FilmInputs.applyTo(*Options)` puis `BuildFromPositions` | `film/replay/film_inputs.go:applyTo` |
| SERIALISE | `encodeGoldenInputs` | `golden_inputs_encode_test.go` (TEST) |
| DESERIALISE | `decodeFilmInputs` | `golden_inputs_decode_test.go` (TEST) |
| REGENERE | `go test ./internal/games/halo_infinite/film/replay/ -run GoldenInputs -update` avec `REPLAY_FILM_DIR=<repo>/data/cache/film_chunks/<short8>` | seule porte d'ecriture |
| `replaybuild` | **ne le connait pas** : il appelle `replay.BuildFromFilm(matchID, slug, film, opts)` | `replaybuild/replaybuild.go:287` |
| `filmproc` | **ne le connait pas** : il tient le verrou de decodage, pas un format | `internal/filmproc/solo.go` |
| `no_second_artifact_sink_test` | **ne le connait pas** (cf. §1.5, c'est une erreur du plan) | `internal/archlint/no_second_artifact_sink_test.go` |

**Ce que la doctrine du fichier interdit d'y mettre**, et qu'il faut garder a M4 :

> « Il porte exactement les champs que l'assemblage CONSOMME. Il ne porte NI la geometrie NI la
> structure de carte : celles-ci ne viennent pas du film mais de catalogues versionnes a part, et
> les inclure ferait grossir le fixture d'un ordre de grandeur pour verrouiller un chargement de
> fichier, pas un decodage. »

C'est la regle qui donne sa frontiere au fichier de faits de 4.1 : **le film, tout le film, rien
que le film**. Les catalogues (bornes de carte, geometrie, structure, socles, zones) et les faits
de la base restent dehors.

#### La preuve deja acquise, et c'est elle qui vaut le plus

`golden_assembly_test.go` (« AUCUN OCTET DE FILM N EST LU ») rejoue l'assemblage complet depuis
`inputs_000d5950.bin.gz` et fige la sortie ; `golden_builds_test.go` le fait **sur les huit
builds** ; `TestGoldenInputsFidelite` verifie que le codec transporte tout ce que l'assemblage
lit ; `golden_inputs_test.go:356` compare deux documents construits, l'un depuis les entrees
fraiches et l'autre depuis le blob relu.

**Autrement dit, l'oracle de 4.1.3 (« document depuis les faits = document depuis le film, a
l'octet ») existe deja, sur 8 builds — mais seulement pour la part `FilmInputs`.** Le lot 4.1
n'invente pas cette preuve : il l'etend a la part manquante et la fait passer en production.

### 1.2 (b) Le contenu EXACT des faits a la sortie de la couche `facts` apres 2.5

Apres 2.5, `facts` = `killsource` (23 fichiers, 5 787 L) + `objectiveevents` (20 fichiers,
5 081 L) + `fallback` (deja feuille) — note M2 §2.3. **La couche `replay` garde ce qui PUBLIE.**
La question de 4.1 est donc : de quoi le publieur a-t-il besoin pour ne plus ouvrir le film ?

Reponse mesuree : de **trois** apports de film, et de trois seulement.

#### (i) `FilmInputs` — 39 champs, l'integralite de ce que le film donne aux calques

Liste collee de `film/replay/film_inputs.go`, dans l'ordre des balayages :

```
FilmMajorVersion *int                         Translocations []filmdec.TranslocatorTeleport
Positions []filmdec.BipedPosition             BipedCreations []filmdec.BipedCreation
Fire []filmdec.FireEvent                      Loadouts []filmdec.KeyframeLoadout
WeaponChanges []filmdec.HeldWeaponChange      Pickups []filmdec.BipedPickup
PickupStats filmdec.BipedPickupStats          Inventory []KeyframeInventory
InventoryDeltas []filmdec.InventoryDelta      InventoryDeltaAmmoRefused bool
AbilityRanks []filmdec.AbilityRank            EquipmentChanges []filmdec.EquipmentChange
EquipmentChangeStats filmdec.EquipmentChangeStats
CamoStates []filmdec.CamoRead                 GrappleReads []filmdec.GrappleRead
AbilityImpulses []filmdec.AbilityImpulse      AbilityImpulseStats filmdec.AbilityImpulseStats
AbilityCharges []filmdec.AbilityCharge        AbilityChargeStats filmdec.AbilityChargeStats
ZoomEvents []filmdec.ZoomEvent                Placements []filmdec.EquipmentPlacement
PlacementStats filmdec.EquipmentPlacementStats
SpawnEvents []filmdec.EquipmentSpawnEvent     SpawnStats filmdec.EquipmentSpawnStats
Pads PadScans                                 Vehicles VehicleScan
FlagMarks filmdec.CarrierMarkScan             ZoneReads []filmdec.ManagedPropertyRead
ZoneScanned bool                              BombReads []filmdec.NavpointRadialRead
Grenades []filmdec.GrenadeThrow               Projectiles []filmdec.ProjectileTrack
Deaths []Death                                PlayerIndices PlayerIndexTable
FilmTable FilmPlayerTable                     PlayerTeams map[int]int
TeamScan filmdec.TeamScanReport               FilmClockOriginUS uint64
```

Couverture par famille demandee au brief :
- **equipement** : `EquipmentChanges` + `EquipmentChangeStats`, `Placements` + `PlacementStats`,
  `SpawnEvents` + `SpawnStats`, `AbilityCharges`, `AbilityImpulses`, `Pads` ;
- **vehicules** : `Vehicles VehicleScan` (recensement, creations, nuage de positions, embarquements,
  visees des occupants — un SEUL champ, et il porte le nuage entier) ;
- **projectiles** : `Projectiles []filmdec.ProjectileTrack` (439 sur le film de reference) ;
- **grenades** : `Grenades []filmdec.GrenadeThrow` (70 sur le film de reference) + les lectures
  d'inventaire delta (`InventoryDeltas`, 184 etats).

**Poids** : c'est le bloc qui coute. Les 8 fixtures pesent 612 528 a 1 802 250 o compresses, et
c'est `Positions` (nuage NON decime) qui domine — un artefact reel publie 29 553 points APRES
decimation (`pointsStride` 1 a 5 au manifeste des fixtures de contrat).

Deux valeurs **NON serialisables telles quelles**, deja resolues par le fixture et dont 4.1 doit
reprendre la solution : `Options.Scoped` est une FERMETURE (le fixture porte `ZoomEvents` bruts et
`applyTo` reconstruit le palier) ; les statistiques de balayage qui ne servent qu'a l'observateur
ne sont pas dans le type (elles restent locales au balayage qui les produit).

#### (ii) `killsource.Result` — 6 types traversants, mais UN seul objet

`internal/games/halo_infinite/film/killsource/kill.go:309`. Champs :

```
Kills []Kill                     UnclaimedDeaths []UnclaimedDeath   Coverage Coverage
Health filmdec.KillSourceHealth  Stats Stats                        Roster Roster
Calibration string               BijectionMargin int                BijectionDetermined bool
Probe *RelaxedProbe
```

`Kill` (kill.go:19) : `TimeMS`, `Victim`, `Feed FeedTruth`, `Source SourceTruth`, `Diverges`,
`Read Provenance`, `Assist Assist`, `KillerDamage`, `AssistDamage`, **et `paquet paquetID` NON
EXPORTE**.

> **Point dur pour le format** : `paquet` est une coordonnee interne au decodeur (« ou l'octet a
> ete lu »), deliberement non exportee. Verification sur pieces : ses sites de lecture sont TOUS
> dans le decodage (`assist.go:371`, `feed_couples.go:137`, `hybrid.go:332` et `:484`,
> `match.go:34/48/131/185`). **Aucun consommateur post-decodage ne la lit** : elle n'a donc pas a
> entrer dans le fichier de faits — mais aucun encodeur generique (gob, reflexion) ne peut la
> porter, et une relecture qui la laisserait a zero doit etre PROUVEE inoffensive, pas supposee.

**Poids** : petit. Ordre de grandeur sur un match a 8 joueurs : ~100 `Kill`, 0 a 5
`UnclaimedDeath`, un `Roster` d'une dizaine d'entrees. Quelques dizaines de kio non compresses.

#### (iii) `objectiveevents` + le statborg — 14 types traversants, agreges en `filmStats`

Ils n'arrivent pas au publieur en vrac : `replaybuild/matchfacts.go:46` les agrege dans un
`filmStats` **non exporte**, issu de `readFilmStats(ctx, matchID, film, facts, deaths)` :

```
recs        []objectiveevents.StatRecord        (via StatRecordsCtx)
objectives  []objectiveevents.IdentifiedEvent   objectivesUnnamed int  objectivesRefused int
statborgIdentity objectiveevents.RoundIdentity
score / flag / vip / skull / bomb                (les entrees de calque par mode)
```

Les 14 types d'`objectiveevents` qui traversent (note M2 §3.2, collee) :

```
DeathInstant FlagFilmSignals FlagGrabsNetPlayer FlagSpan FlagTrack IdentifiedEvent NamedEvent
PlayerLine RoundBounds RoundIdentity RoundsDecision ScorePoint StatComponent StatRecord
```

> **Point dur d'architecture** : `filmStats` est un type **de `replaybuild`**, pas de `facts`. Il
> MELANGE du film (`recs`, `objectives`) et de la table de reglement du titre
> (`stats.score.TargetScore`, `HoldTicksPerPoint`, poses juste apres l'appel) et du catalogue de
> carte (`stats.flag.Spawns`, pose par `collecterEntreesCatalogue`). **Le fichier de faits ne doit
> porter que la part film.** La ligne de coupe est a tracer explicitement au lot (§7, A3).

#### (iv) Le registre d'identite — il est dans `replay`, pas dans `facts`

```
$ wc -l film/replay/identity*.go   (production seule, sans les _test.go)
   345 identity.go                      402 identity_registry.go
   109 identity_registry_bridge.go      406 identity_registry_creation.go
   147 identity_registry_elimination.go 224 identity_registry_exclusion.go
   221 identity_registry_film_table.go   98 identity_registry_health.go
   113 identity_registry_mutations.go   239 identity_registry_pont.go
   253 identity_registry_scoreboard.go  364 identity_registry_section.go
   = 2 921 L de production
```

Il est **derive**, pas lu : ses entrees sont `Deaths`, `FilmTable`, `PlayerIndices`,
`BipedCreations`, `PlayerTeams` (tous dans `FilmInputs`) plus le roster de la base
(`RosterXUIDs`, `Participants`, `ScoreboardTeams`, `StatborgIdentity`, poses par
`buildReplayOptions`). **Il n'a donc RIEN a faire dans le fichier de faits** : il se recalcule a la
publication, a cout pur.

C'est la reponse a la question implicite de 4.1 : **ou couper ?** La coupe est a la sortie des
balayages, pas apres l'assemblage. Deux raisons mesurees :

1. `golden_assembly_test.go` prouve deja que l'assemblage complet (registre d'identite compris)
   se rejoue depuis le seul `FilmInputs`, a l'octet, sur 8 builds. Couper plus tard obligerait a
   refaire cette preuve ;
2. couper apres l'assemblage ferait entrer le document dans le fichier de faits — c'est-a-dire
   persister deux fois la meme chose, et faire de tout changement de publication une invalidation
   des faits, exactement ce que D-7 interdit.

#### Recapitulatif : le contenu du fichier de faits

| Bloc | Source | Poids | Entre dans le fichier ? |
|---|---|---:|---|
| `FilmInputs` (39 champs) | film | 0,6 a 1,8 Mio gz | **oui**, c'est le gros |
| `killsource.Result` | film | des dizaines de kio | **oui** |
| Part FILM de `filmStats` (`recs`, `objectives`, `objectivesUnnamed`, `objectivesRefused`, `statborgIdentity`, entrees de calque par mode) | film | dizaines a centaines de kio | **oui**, apres coupe explicite |
| `deaths` (`lireMorts(film)`) | film | petit | **oui** (deja dans `FilmInputs.Deaths`) |
| Registre d'identite | derive | — | **non**, recalcule |
| Geometrie, structure, bornes, socles, zones, points d'apparition | catalogues versionnes | — | **non** (doctrine du fixture) |
| `port.MatchFacts` (lignes de match, scores, variante, map_id) | base | — | **non** — ils ont DEJA leur fichier, cf. §1.4 |
| `Options.Observe`, `Options.Scoped` | fonctions | — | **non** (reconstruites) |

### 1.3 (c) Le format : gob ou binaire maison ?

#### Le critere qui tranche

Le plan donne l'oracle : « document rejoue depuis les faits = document cuit, a l'octet ». Ce
critere ne porte PAS sur le fichier de faits, il porte sur le DOCUMENT. Le format est donc libre
tant qu'il est **fidele** (tout ce que l'assemblage lit revient), **refusant** (une cle qui ne
correspond pas est une erreur, pas une lecture au mieux) et **borne** (le parc tient sur le
disque).

#### Comparaison, sur pieces

| Critere | `encoding/gob` | Codec maison (celui qui existe) |
|---|---|---|
| Cout d'ecriture du lot | quasi nul | **1 215 L de codec + 434 L de harnais a faire passer de `_test.go` a la production**, plus les sections neuves (killsource, objectiveevents) |
| Champs non exportes (`Kill.paquet`) | silencieusement perdus | silencieusement perdus aussi — **egalite**, et dans les deux cas il faut la preuve que rien ne les lit apres coup (elle est faite, §1.2 (ii)) |
| Fermetures (`Options.Scoped`) | erreur a l'encodage | deja resolu : le type porte `ZoomEvents` bruts |
| Reproductibilite octet a octet | **NON** : `FilmInputs.PlayerTeams` est une `map[int]int`, et gob n'ordonne pas les entrees de map. Deux serialisations du meme objet peuvent differer | **OUI** : chaque section est ecrite dans un ordre decide (`sort` explicite dans l'encodeur) |
| Un champ ajoute / retire | **tolere en silence** (valeur zero). C'est exactement l'ambiguite « absence de champ » que D-7 declare etre la source des regressions « entre deux versions de schema » | **refuse** : la magie s'incremente dans le meme commit que la suite des sections, et `TestGoldenInputsVersionGuard` verrouille le refus explicite |
| Taille | descripteurs de types en tete, entiers non delta-codes : facteur 2 a 4 attendu sur des nuages de positions | delta + varint mesure : **1,38 Mio par film** ; a 1 386 films le facteur compte (1,9 Gio contre 4 a 8 Gio) |
| Preuve deja acquise | aucune | **8 builds, 22 versions de format, 4 tests de fidelite** |

#### Recommandation : le codec maison, promu en production

Trois raisons, dans l'ordre de poids :

1. **La preuve existe et elle est chere a refaire.** `golden_assembly_test.go` +
   `golden_builds_test.go` + `TestGoldenInputsFidelite` demontrent deja l'oracle de 4.1.3 sur la
   part `FilmInputs`. Passer a gob jetterait cette preuve et re-ouvrirait la question sur 8 builds.
2. **Gob tolere ce que D-7 interdit.** « La presence d'un calque se lit dans sa revision, jamais
   dans l'absence d'un champ » : un format qui rend zero pour un champ absent contredit la
   decision au niveau du transport, pas seulement du document.
3. **Le volume.** 1,9 Gio est deja une question ; 4 a 8 Gio en serait une autre.

Cout accepte et a annoncer : **le lot 4.1 est un lot L**, et sa plus grosse part est le passage
en production d'environ 1 650 L de code aujourd'hui sous `_test.go`, plus l'extension du codec a
`killsource.Result` et a la part film de `filmStats`. Les fixtures existantes doivent alors
**consommer le codec de production** (comme `golden_inputs_film_test.go` consomme deja
`scanFilmInputs` depuis le lot 1.0), sans quoi les deux copies redivergeraient — c'est la
decouverte D7 du plan, deja payee une fois.

#### L'en-tete propose

Le plan demande `{grammarRev, factsRev, build, schema de faits}`. L'en-tete actuel en porte deja
la moitie utile. Proposition, dans l'ordre d'ecriture :

```
magie          "REPLAYFACTS01\n"   le SCHEMA DE FAITS : s'incremente dans le meme commit que
                                   la suite des sections (regle heritee de goldenInputsMagic)
build          str                 la cle du profil, lue en clair dans chunk_00 section 2
                                   (ADR 0034 D-3) ; chaine VIDE = ErrUnknownBuild, et le
                                   fichier n'est alors PAS ecrit (D-4 : le film est mis de cote)
grammarRev     str                 la valeur de grammar.Rev au moment de la cuisson
factsRev       str                 la valeur de facts.Rev au moment de la cuisson
sourceRev      str  (option)       si les quatre revisions de 2.6 sont retenues
profileRev     str  (option)       idem
film           str                 le short8, deja present
mapModule      str                 deja present — sa verification est errGoldenInputsCarte
axisW[3]       uvarint x3          deja present
layoutDetected bool8               deja present
mapCatalogRev  str  (ajout)        l'empreinte du catalogue de bornes dont les quanta dependent
... puis les sections, inchangees, plus les sections neuves
```

**La regle de relecture, non negociable** : une seule des cles (`magie`, `build`, `grammarRev`,
`factsRev`, `mapModule`, `axisW`, `mapCatalogRev`) qui ne correspond pas a l'etat courant =
**le fichier est ignore et le film est redecode**. Jamais une lecture partielle, jamais un
« au mieux ». Le code de refus existe deja en trois formes (`errGoldenInputsCarte`,
`errGoldenInputsDecoupage`, la garde de magie) : 4.1 les generalise, il ne les invente pas.

### 1.4 (d) Le chemin par `PathResolver`

`PathResolver` (`internal/domain/title/registry.go`, 56 methodes) n'expose AUCUN chemin de cache
de film aujourd'hui : `film_chunks` et `film_manifests` sont des constantes de
`games/halo_infinite/film/filmcache/filmcache.go:47`, sous une racine passee par l'appelant, et
c'est le ratchet `archlint/no_hardcoded_film_cache_dirs_test.go` qui l'encadre.

Modele a copier, colle :

```go
// ReplayArtifactPath retourne le chemin de l'artefact de rejeu 2D d'un match ...
func (p *PathResolver) ReplayArtifactPath(titleSlug, matchID string) string {
	return filepath.Join(p.ReplayArtifactsDir(titleSlug), FilmShortMatchID(matchID)+".json")
}

func (p *PathResolver) ReplayArtifactsDir(titleSlug string) string {
	return filepath.Join(p.repoRoot, "data", "cache", "replays", titleSlug)
}
```

**Methodes a ajouter (deux, pas une)** :

```go
// FilmFactsPath : les faits lus dans le film d'un match, avec la revision de la couche qui
// les a produits. Cache regenerable (data/cache/*), jamais une donnee de reference.
func (p *PathResolver) FilmFactsPath(titleSlug, matchID string) string
    // -> data/cache/film_facts/{slug}/{short8}.<ext>

// FilmFactsDir : le dossier, pour la purge — meme role que ReplayArtifactsDir.
func (p *PathResolver) FilmFactsDir(titleSlug string) string
    // -> data/cache/film_facts/{slug}
```

La cle est `title.FilmShortMatchID(matchID)` : « la cle de tout ce qui derive d'un film », et
l'artefact l'emploie deja. Un chemin ecrit a la main est rouge par
`archlint/no_data_path_join_test.go`.

**Trois pieges, mesures** :

1. **Collision de vocabulaire.** `replaybuild/facts_file.go` definit deja `FactsFile`, lu et
   ecrit sous le nom `<short8>.facts.json`, et ce sont **les faits que LA BASE sait** (lignes de
   match, scores, variante, map_id, cartes candidates) — exactement le contraire du sens de 4.1.
   Trois programmes le manipulent (`levelup replay-facts-export`, `cmd/replay-equiv`,
   `cmd/replay-build --facts`). Nommer le fichier de 4.1 `{short8}.facts.bin` le poserait a cote
   de `{short8}.facts.json` avec un sens oppose. **Recommandation : `{short8}.filmfacts.bin`**,
   ou renommer explicitement l'existant — jamais les deux extensions voisines (§7, D2).
2. **La couche par titre.** Le plan ecrit `film_facts/{slug}/`, alors que `film_chunks/` et
   `film_manifests/` n'ont pas de couche par titre. La couche est juste (multi-titre, ADR 0008)
   mais elle cree une asymetrie avec les deux caches voisins : a assumer par ecrit, ou a aligner.
3. **D12 ne s'applique pas ici.** Le ratchet `no_runtime_versioned_catalog_write_test.go` interdit
   d'ECRIRE a l'execution un **catalogue versionne** (`data/titles/{slug}/reference/...`,
   cf. `FilmProfilesPath`, pose par le volet donnees de 3.1). Le fichier de faits est un **cache
   regenerable** sous `data/cache/` : il est ecrit a l'execution par construction. Le confondre
   avec un catalogue bloquerait le lot.

### 1.5 (e) Le verrou solo, et le « second puits d'artefact » — une correction du plan

#### Le verrou

`internal/filmproc/solo.go:91` : `AcquireSolo(cacheRoot, tool, matchID) (*SoloLock, error)` —
verrou de machine, fichier `data/cache/film_decode.lock`, refus immediat par `ErrDecodeBusy` ;
`AcquireSoloWait` attend jusqu'a `max`. Les garde-rails : `archlint/no_unbounded_film_loop_test.go`
exige que `bake.go` prenne `filmproc.AcquireSolo(Wait)` **avant le premier decodage**, et
`archlint/no_cuisson_depuis_tactique_test.go` interdit de cuire depuis la voie tactique.

Constat du gate 22 (colle) : le verrou est pris **28 fois** pour 14 temoins, soit **deux cuissons
par temoin** (base et tete) :

```
2026/09/16 10:09:53 INFO verrou de decodage pris outil=replay-corpus-gate match_id=bcb6d393-... verrou=...\data\cache\film_decode.lock
2026/09/16 10:09:53 INFO priorite CPU abaissee pour ce decodage outil=replay-corpus-gate classe=below_normal
```

**Regle a ecrire au lot 4.1.2** : le verrou protege **le DECODAGE**, pas la lecture des faits.
Une publication qui rejoue depuis les faits ne decode pas : elle ne doit **pas** prendre le
verrou, sinon toute recuisson de parc se re-serialise sur une exclusion dont elle n'a pas besoin
— et le gain de 4.1 est mange par l'attente. Le corollaire est qu'il faut deux chemins nommes
dans `replaybuild` (« depuis les faits », sans verrou ; « depuis le film », avec verrou pris avant
la premiere lecture), et que `no_unbounded_film_loop_test.go` doit continuer a prouver que le
second le prend en premier.

#### La correction

Le plan, item 4.1.2 : « aucun second puits d'artefact (`no_second_artifact_sink_test`) ».
**Sur pieces, ce ratchet ne garde pas ce que la phrase suppose.** Son en-tete, colle :

> « `replaybuild.SetArtifactStoredSink` cable le puits par lequel passent TOUTES les ecritures
> d'artefact du serveur. Il est fait pour etre appele UNE SEULE FOIS, au boot. Un second cablage
> n'ajouterait pas un second observateur : il REMPLACERAIT le premier. »

Il compte les appels a `SetArtifactStoredSink(` hors tests, avec deux sites autorises
(`api/wire/registry_replay_notify.go`, `replaybuild/artifact_events.go`). C'est le puits de
**NOTIFICATION** (Discord), pas le point d'ecriture.

Le point d'ecriture unique est ailleurs : `writeArtifactBytes` (`replaybuild/artifact_store.go:168`),
avec **quatre appelants repartis dans trois binaires** (dit par `artifact_events.go:8`), et la
regle de non-regression est `wouldDowngrade` (refus d'appauvrissement a schema EGAL) plus
`validateArtifact` (le refus de version, sur `StoreArtifact`) — l'ADR 0034 le corrige deja dans
sa section « Corrections », point 1.

**Consequence pour 4.1.2** : la garantie « un seul puits » qu'il faut poser pour les FAITS n'existe
pas encore. Elle demande soit un ratchet neuf sur le point d'ecriture du fichier de faits, soit la
reprise litterale de la doctrine de `writeArtifactBytes` (ecriture atomique, un point, refus de
regression). Le plan doit etre corrige dans le commit du lot, comme le 57 de 2.6 l'a ete.

### 1.6 (f) L'estimation de gain — mesuree, et ce que la mesure ne donne PAS

#### Ce que le scratchpad du gate contient, et ce qu'il ne contient pas

Le brief cite `scratchpad/gate_22.log` et des lignes `cuisson: phase phase=...`. **Mesure : ce
fichier n'en contient aucune.**

```
$ grep -c "cuisson" gate_22.log
0
```

Ce que `gate_22.log` porte est la table finale du gate, par temoin — c'est-a-dire le total de
DEUX cuissons (base et tete). Colle :

```
temoin       famille          base(ff80ec59c)   HEAD    gains   pertes   chang.      duree  statut
bcb6d393     ctf_mono_manche      60     60        0        0        0     11.82s  ok
bf15f7ab     slayer               60     60        0        0        0     13.94s  ok
c75f33b8     assaut_bombe         60     60        0        0        0     14.68s  ok
0797ce72     region_index_2_bits  60     60        0        0        0        18s  ok
51ebbc0f     deux_manches         60     60        0        0        0     19.39s  ok
bfecd02b     vehicules_v41_utilisateur 60  60      0        0        0      21.1s  ok
d9781168     oddball              60     60        0        0        0     24.82s  ok
60ae07c4     version_37           60     60        0        0        0     30.18s  ok
fb1a1a72     ctf_multi_manche     60     60        0        0        0     30.37s  ok
111fa685     version_39           60     60        0        0        0     40.57s  ok
e5adf7b2     version_40_build_1_11 60     60        0        0        0      45.2s  ok
4f77afc1     equipement_origine_utilisateur 60 60   0        0        0      1m51s  ok
084a804d     vehicules            60     60        0        0        0   2m24.61s  ok
a349fea8     version_33_sans_identification 60 60   0        0        0   3m36.67s  ok
```

Somme : **742,35 s pour 28 cuissons**, soit **26,5 s par cuisson en moyenne**, mediane ~22 s,
pire cas ~108 s (a349fea8). **Le « 15 s par film » du plan et de l'architecture §11 est perime**
(chiffre du 2026-09-12) : la valeur d'aujourd'hui, sur le corpus gate, est **26,5 s**.

#### Les phases, retrouvees ailleurs dans le scratchpad

Les lignes `cuisson: phase` existent, dans les journaux du lot 0.D. Sortie collee (extrait,
trie par film puis par phase) :

```
decodage 000d5950 12.8200241s      killsource 000d5950  2.3420855s   stats 000d5950  455.1582ms
decodage 000d5950 37.6465926s      killsource 000d5950  2.9553579s   stats 000d5950  550.0168ms
film     000d5950   107.836ms      marshal    000d5950     9.3358ms
decodage 01e1f945 16.9235831s      killsource 01e1f945  2.7087703s   stats 01e1f945  493.3446ms
decodage 01e1f945 17.9140935s      killsource 01e1f945  3.1330445s   stats 01e1f945  587.2316ms
film     01e1f945   135.1653ms     marshal    01e1f945     7.3416ms
decodage 084a804d 1m48.6274104s    killsource 084a804d 22.5037514s   stats 084a804d 1.8130679s
decodage 084a804d 3m1.8466433s     killsource 084a804d 52.6590701s   stats 084a804d 5.4117777s
film     084a804d  230.7078ms      marshal    084a804d    30.9572ms
decodage 111fa685 35.275656s       killsource 111fa685 10.666727s    stats 111fa685  741.5538ms
decodage 11de8353 32.6599569s      killsource 11de8353 34.861343s    stats 11de8353 2.0621869s
decodage a521164d 1m12.6258153s    killsource a521164d 16.1899381s   stats a521164d 1.0085463s
film     a521164d    71.3076ms     marshal    a521164d    74.422ms
```

Source : `internal/replaybuild/timing.go:31` (`logPhase`), cinq appels dans
`replaybuild/replaybuild.go` (l. 264 `film`, 267 `stats`, 287 `decodage`, 296 `marshal`,
356 `killsource`).

| Phase | Ce qu'elle fait | Plage mesuree | Disparait avec les faits ? |
|---|---|---:|---|
| `film` | ouvrir le manifeste, charger et decompresser le film une fois, lire le fil des morts | 0,071 a 2,19 s | **oui** |
| `stats` | `readFilmStats` : second decodage du statborg (`objectiveevents`) | 0,455 a 5,41 s | **oui** |
| `killsource` | `decodeKillSource` : la source de degat | 2,34 a 52,66 s | **oui** |
| `decodage` | `BuildFromFilm` = **39 balayages PUIS l'assemblage** | 12,82 s a **3 m 01,85 s** | **en partie seulement** |
| `marshal` | `json.Marshal(doc)` | 7,3 a 74,4 ms | non (0,01 a 0,07 s) |

#### Le trou de mesure, et comment le combler SANS decoder

`decodage` melange les balayages (qui disparaissent) et l'assemblage (qui reste). **Aucune mesure
du depot ne les separe aujourd'hui** : `film/replay/observe.go` chronometre les 39 etapes de
balayage en `slog.Debug` (`stepClock`, l. 82-86), mais l'assemblage (`BuildFromPositions`) n'est
pas observe — la liste fermee `BuildFromFilmSteps` s'arrete a `clockOrigin`.

Tant que ce trou existe, **« secondes contre 15 s » n'est pas verifiable**. Deux facons de le
combler, toutes deux **sans aucun decodage** :

1. **La moins chere, et elle est disponible tout de suite** : chronometrer
   `go test ./internal/games/halo_infinite/film/replay/ -run GoldenAssembly` et
   `-run GoldenBuilds`. Ces tests executent **l'assemblage complet depuis `inputs_*.bin.gz`,
   zero octet de film** : leur duree EST, a la lecture du blob pres, la duree de la publication
   depuis les faits. C'est la mesure a consigner en tete du lot 4.1, avant d'ecrire une ligne.
2. Un `logPhase("assemblage", ...)` de plus dans `BuildFromFilm` — mais il touche `film/replay`,
   paquet sequentiel de M2 (§6).

Attente raisonnee, a confirmer par (1) : le `marshal` mesure (10 a 75 ms) sur un document de
1,2 a 2,0 Mio et la decompression d'un blob de 1,4 Mio placent l'assemblage dans l'ordre de la
**seconde**, contre **26,5 s** de cuisson complete — soit **un facteur 20 a 30**, pas un
facteur 2. C'est ce qui justifie le lot ; ce n'est pas encore ce qui le prouve.

---

## 2. Lot 4.2 — La revision par calque portee par le document

### 2.1 Ou vivent les calques

| Role | Fichier | Taille |
|---|---|---:|
| Le type du document cuit | `film/replay/document.go` | 444 L, **58 champs racine** |
| Les calques, un fichier par famille | `film/replay/document_*.go` | 28 fichiers de production |
| La chronique | `film/replay/document_chronicle.go` | **1 569 L**, trois formes d'entete, v32 sautee, v51 restauree le 2026-09-14 |
| La couverture (le proto-`layers` d'aujourd'hui) | `film/replay/coverage.go` | 466 L, **47 tags JSON**, dont 34 blocs `*XxxCoverage` en `omitempty` |
| Le jumeau servi | `domain/replaydoc/document.go` + 10 fichiers | 1 783 L, **58 champs racine**, zero import du depot hors `time` |
| L'empreinte de forme (0.B.4) | `film/replay/document_shape_test.go` + `testdata/document_shape.golden` | 511 L + **989 L** |

Les calques nommes et leur producteur (les 23 `document_*.go` de production qui portent un
calque) :

```
document_ability_charges.go     document_ability_impulses.go   document_aim.go
document_bomb_armings.go        document_bomb_carries.go       document_equipment_changes.go
document_ground_weapon_items.go document_ground_weapons.go     document_labels.go
document_neutral_deaths.go      document_objective_objects.go  document_objectives_live.go
document_pickups.go             document_score.go              document_skull_carries.go
document_structure.go           document_tracks.go             document_translocations.go
document_vehicles.go            document_vehicles_coverage.go  document_vip_crown.go
document_weapon_changes.go      document_zones.go
```

### 2.2 Quels champs seraient `layers: {nom: revision}`

Trois nomenclatures existent deja et **il faut en choisir une, pas en creer une quatrieme** :

| Candidat | Cardinal | Pour | Contre |
|---|---:|---|---|
| Les etapes de `BuildFromFilmSteps` | 39 | ferme par `observe_test.go`, deja l'unite du harnais d'equivalence | ce sont des BALAYAGES, pas des calques : le web n'en lit aucun |
| Les blocs de `Coverage` | 34 | deja publies, deja lus par le web pour distinguer « absent » de « vide » | ils disent la COUVERTURE, pas la revision ; certains couvrent plusieurs champs |
| **Les champs optionnels du document que le web lit** | 16 (cf. §2.3) | **c'est le lecteur qui decide**, et le lecteur est `normalizeReplayDocument` | il faut ecrire la liste et la fermer par un test |

**Recommandation : la troisieme**, avec la regle que D-7 pose : « la presence d'un calque se lit
dans sa revision, jamais dans l'absence d'un champ ». Le nom d'un calque est donc celui de la
**cle JSON du document** que le web comble aujourd'hui, et le producteur de chaque nom est le
`document_*.go` qui l'ecrit. Forme :

```go
// dans ReplayDocument, a cote de SchemaVersion
Layers map[string]string `json:"layers,omitempty"`   // nom de calque -> revision de la couche qui l'a produit
```

La valeur d'une revision : **celle de la couche qui a produit le calque** (`grammar.Rev` pour un
calque lu dans le flux, `facts.Rev` pour un calque assemble), ce que `coverage.decoder` de 2.6
publie deja globalement. 4.2 le descend au calque.

### 2.3 Ce que `normalizeReplayDocument` devra lire

`apps/web/src/lib/replay/replayNormalize.ts`, **232 L**, une seule fonction exportee
(`normalizeReplayDocument(raw: ReplayDocument): ReplayDocumentReady`, l. 62).

Sa mecanique, aujourd'hui : elle **comble** chaque tableau absent par `[]`, et chaque site porte
en commentaire la meme phrase — « absent = artefact anterieur, ou film qui n'en porte aucun ;
`coverage.X` distingue les deux ». Extraits colles :

```
l.  76  // Absent = artefact anterieur, ou film hors famille bomb — `coverage.bombCarries`
l.  88  // n'en porte aucun — `coverage.equipmentChanges` distingue les deux.
l.  92  // anterieur, ou film qui n'en porte aucun (`coverage.weaponChanges` distingue les deux).
l.  96  // qui n'en porte aucun (`coverage.pickups` distingue les deux).
l. 100  // `coverage.groundWeaponItems` distingue les deux. Aucun tableau imbrique : l'objet est plat.
l. 114  // calque — `coverage.flagCarries` distingue les deux, et c'est pour cela qu'il est publie.
l. 122  // non reconnu VIP — `coverage.vipCrown` distingue les deux, et c'est pour cela qu'il existe.
l. 126  // reconnu Oddball — `coverage.skullCarries` distingue les deux.
l. 130  // au schema 38, ou film sans translocateur — `coverage.translocations` distingue les deux.
l. 134  // propulseur, ou palette non classee — `coverage.abilityImpulses` distingue les trois.
l. 139  // `coverage.abilityCharges` distingue les trois.
l. 143  // `coverage.objectiveObjects` distingue les trois, et c'est pour cela qu'il est publie.
l. 183  // aucun (`coverage.groundWeapons` distingue les deux, et c'est pour cela qu'il est publie).
l. 199  // (`coverage.vehicles` distingue les deux). LES DEUX TABLEAUX IMBRIQUES SE COMBLENT AUSSI
l. 225  // `coverage.zones` distingue les deux, et c'est pour cela qu'il est publie.
```

**C'est la liste des calques de 4.2.1, ecrite par le lecteur lui-meme** : bombArmings,
bombCarries, equipmentChanges, weaponChanges, pickups, groundWeaponItems, flagCarries, vipCrown,
skullCarries, translocations, abilityImpulses, abilityCharges, objectiveObjects, groundWeapons,
vehicles, zoneStates. Seize noms, tous deja documentes, tous deja apparies a un bloc de couverture.

Ce que 4.2.2 change : **la question « ce calque est-il present ? » cesse d'etre une jointure entre
un tableau absent et un bloc de couverture, et devient une lecture de `layers[nom]`.** C'est le
seul point web touche (D11), et le rendu ne bouge pas.

Precedent a citer dans le lot, il est deja dans le fichier (l. 160-163) :

```ts
    // LA PART DE REPLI du document (schema 58) : `coverage.fallbacks` liste les replis du
    // ... l'OBJET `coverage`, lui, garde le droit d'etre absent (meme regime qu'`identity`).
    coverage: raw.coverage ? { ...raw.coverage, fallbacks: raw.coverage.fallbacks ?? [] } : undefined,
```

`layers` suivra le meme regime : l'objet a le droit d'etre absent (artefact anterieur a 4.2) ;
**une entree absente dans un `layers` PRESENT veut dire « ce calque n'a pas ete produit »**, et
c'est une reponse, pas un trou.

### 2.4 L'empreinte de forme (0.B.4) et la matrice de compatibilite (0.B.2) — les fichiers

**Empreinte de forme, cote Go** :

| Fichier | Role |
|---|---|
| `film/replay/document_shape_test.go` (511 L) | `TestDocumentShapeMatchesGolden` (l. 59), `...TwinsAgree` (l. 76), `...GoldenCarriesCurrentSchema` (l. 91), `...SchemaHasChronicleEntry` (l. 101), `TestDocumentShapeRegenerate` (l. 124, porte unique et bruyante : exige `-update` ET une variable d'environnement, `t.Fatalf` meme en succes) |
| `film/replay/testdata/document_shape.golden` (989 L) | la forme figee en clair, a cote de son `SchemaVersion` |
| `film/replay/document_chronicle.go` (1 569 L) | une entree par version, **ecrite dans le commit qui monte la version** |
| `testutil/replay_chronicle.go` | l'extracteur partage (trois formes d'entete, ne comble aucun trou) |
| `contracttest/replay_contract_test.go:749` | `TestReplayDocumentFieldCountIsFrozen` : 58 champs servis, des deux cotes (type Go **et** schema genere) |
| `replaybuild/artifact_schema_history_test.go` | par version declaree par la chronique : `Digest` la classe perimee et le depot la refuse |

**Matrice de compatibilite, cote web** :

| Fichier | Role |
|---|---|
| `features/match-replay/model/replaySchemaStatusLogic.ts` | `MIN_RENDERABLE_SCHEMA_VERSION = 27`, justifiee par la chronique (v6 `Inventory.a`, v27 `weaponChanges[].until` : les deux seules montees qui RETIRENT un champ promis) ; `computeReplaySchemaStatus` a quatre etats |
| `features/match-replay/test/goFixtures.contract.test.ts` (222 L) | passe chaque fixture Go par `normalizeReplayDocument` ; l. 112 : aucune fixture sous le seuil |
| `features/match-replay/test/fixtures/go/manifest.json` | 8 fixtures, schema **60**, un build chacune, avec `pointsStride` |
| `features/match-replay/test/goFixtures.ts` (160 L) | le chargeur |
| `film/replay/contract_fixtures_test.go` | le producteur Go des fixtures, depuis les `inputs_*.bin.gz` |
| `features/match-replay/test/testDoc.guard.test.ts` | interdit tout litteral de version de schema hors `fixtures/go/` |

Les huit fixtures pesent 83 380 a 515 021 o compresses (2,49 Mio au total) : **etendre la matrice
aux calques (item 4.2.2) ne coute pas un octet de plus** tant que `layers` ne fait qu'ajouter une
table de chaines courtes ; ce qui coute, c'est une fixture de PLUS, et le plafond de
`contract_fixtures_budget_test.go` est la pour le rendre explicite.

---

## 3. Lot 4.3 — Un seul type publie

### 3.1 Les deux jumeaux, mesures

```
$ wc -l internal/domain/replaydoc/*.go internal/service/replayview/*.go
   381 domain/replaydoc/coverage.go            315 service/replayview/convert_coverage.go
   196 domain/replaydoc/coverage_objectives.go 169 service/replayview/convert_coverage_objectives.go
   128 domain/replaydoc/coverage_world.go      138 service/replayview/convert_coverage_world.go
    49 domain/replaydoc/doc.go                 208 service/replayview/convert_document.go
   238 domain/replaydoc/document.go             75 service/replayview/convert_ground_weapons.go
    74 domain/replaydoc/ground_weapons.go      101 service/replayview/convert_identity.go
   149 domain/replaydoc/identity.go            155 service/replayview/convert_inventory.go
   147 domain/replaydoc/inventory.go           165 service/replayview/convert_map.go
   177 domain/replaydoc/map.go                 210 service/replayview/convert_objectives.go
   188 domain/replaydoc/objectives.go           65 service/replayview/convert_vehicles.go
    56 domain/replaydoc/vehicles.go             76 service/replayview/replayview.go
                                               611 service/replayview/parity_test.go
  = 1 783 L (le type servi)                   = 2 288 L (la conversion + sa preuve)
```

### 3.2 Ce que la conversion fait, champ par champ

`toReplayDocument` (`convert_document.go:11`) : **58 affectations**, une par champ racine, chacune
etant l'un de quatre gestes seulement — copie directe (`SchemaVersion`, `MatchID`, `TitleSlug`,
`FrameCount`, `FrameIntervalMS`, `DurationMS`, `OriginMs`, `T0FilmMs`, `KillEffects`),
`sliceOf(v.X, toY)`, `ptrOf(v.X, toY)`, `mapOf(v.X, toY)`. Extrait colle :

```go
func toReplayDocument(v replay.ReplayDocument) replaydoc.ReplayDocument {
	return replaydoc.ReplayDocument{
		SchemaVersion:       v.SchemaVersion,
		MatchID:             v.MatchID,
		...
		Coverage:            ptrOf(v.Coverage, toCoverage),
		Identity:            ptrOf(v.Identity, toIdentitySection),
	}
}
```

**Il n'y a AUCUNE transformation.** Preuve sur pieces, dans `parity_test.go` :

```go
var champsNonServis = map[string]string{}   // « La liste est vide au 2026-09-05 »
var typesNonServis  = map[string]string{}   // « Vide au 2026-09-06. »
```

Les deux echappatoires sont vides, et le troisieme test du fichier exige que « un document stocke
rempli de valeurs distinctes se serialise **a l'octet pres** comme sa projection ». Les deux
formes sont donc **identiques champ pour champ (58 = 58) et octet pour octet** : la conversion est
une recopie de 1 677 L, gardee par 611 L de preuve.

### 3.3 Ce qui empeche `film/replay` de produire `replaydoc` directement

**Rien, au compilateur.** Mesure :

```
$ grep -rn "levelup/go-api/internal/domain" --include=*.go internal/games/halo_infinite/film/replay/ | grep -v _test
(aucune sortie)
```

`film/replay` n'importe aucun `domain/` aujourd'hui, et la direction `games/{slug}/... ->
domain/...` est la direction permise (l'inverse est celle qu'ADR 0012 et 0025 interdisent).
Le seul import interdit est ecrit dans `replaydoc/doc.go` et il est dans l'autre sens :

> « AUCUN IMPORT d'`internal/games/halo_infinite/film/replay` ICI, jamais : ce paquet est une
> feuille de `domain/`, et c'est ce qui garantit que le contrat ne suive pas le format de stockage
> par simple alias. »

**Ce qui empeche 4.3, c'est donc une DECISION ecrite, pas une contrainte technique** — et c'est
la decision du 2026-09-05, dont `doc.go` donne la mesure qui l'a motivee :

> « `SchemaVersion` a ete incremente **43 fois en cinq semaines** (v7.3.0 -> 2026-09-05), et
> chacune de ces montees regenerait le contrat public pour un champ que le client ne lisait pas
> encore ; inversement aucun champ ne pouvait etre renomme pour le client sans invalider le parc
> d'artefacts deja cuits. »

4.3 **renverse** cette decision. Il faut donc que le lot dise ce qui remplace les deux garanties
perdues :

| Garantie perdue | Ce que M4 propose a la place | Suffit-il ? |
|---|---|---|
| « ajouter un calque a la cuisson ne touche plus le contrat public » | `layers` : un calque neuf s'annonce par sa revision, le client qui ne le connait pas l'ignore | **oui**, si `layers` est additif et si le web ne lit jamais un champ inconnu |
| « renommer un champ pour le client sans invalider le parc » | **rien** | **non** : avec un type unique, renommer une cle JSON change le format d'artefact, donc perime le parc |

C'est LA question d'architecture de 4.3 (§7, A5). Le rythme de montee mesure (60 au 2026-09-17,
43 montees en cinq semaines a l'ete) dit que le probleme n'est pas theorique.

### 3.4 L'en-tete `X-Replay-Latest-Schema-Version` : ou il vit

```
internal/api/middleware/cors.go:16    const ReplayLatestSchemaHeader = "X-Replay-Latest-Schema-Version"   <- la constante
internal/api/handlers/replay.go:82    LatestSchemaVersion int `header:"X-Replay-Latest-Schema-Version"`    <- le champ Huma qui l'ecrit
internal/api/handlers/replay_test.go:177                                                                   <- la preuve (= replay.SchemaVersion)
api/openapi.yaml:3630                 X-Replay-Latest-Schema-Version:                                      <- le contrat
apps/web/src/lib/api/client.ts:322                                                                         <- l'exposition hors corps JSON
apps/web/src/lib/replay/queries.ts:46 'X-Replay-Latest-Schema-Version'                                     <- la lecture web
apps/web/src/lib/replay/replayReadyTypes.ts:381                                                            <- le champ de la frontiere
apps/web/src/lib/api/generated.ts:17673                                                                    <- le type genere
```

Il est dans le middleware CORS parce qu'il doit etre expose
(`Access-Control-Expose-Headers`). 4.3.1 dit qu'il « ne porte que la version du producteur » :
**c'est deja le cas** (`replay_test.go:177` le prouve contre `replay.SchemaVersion`). Ce que 4.3
change est l'autre moitie : aujourd'hui l'en-tete est la SEULE source de cette valeur parce que
`replaydoc` ne porte pas de numero de producteur (`doc.go` : « PAS DE NUMERO DE VERSION DANS CE
PAQUET, et c'est une decision », le `ContractVersion = 39` ayant ete retire en revue comme
tautologique). Avec un type unique, le corps porte `schemaVersion` (de l'artefact lu) ET le
producteur pourrait poser le sien ; **il ne doit pas** — sinon `computeReplaySchemaStatus`
comparerait deux nombres du meme corps et ne distinguerait plus l'artefact lu du producteur
courant. La regle a ecrire : **l'en-tete reste la seule source de la version du producteur**, et
c'est exactement ce que 4.3.1 dit.

### 3.5 Les commandes du contrat

```
make openapi-gen     -> cd apps/go-api && CGO_ENABLED=1 go run ./cmd/openapi-gen
make generate-types  -> openapi.yaml -> apps/web/src/lib/api/generated.ts
make openapi-check   -> cd apps/go-api && CGO_ENABLED=1 go run ./cmd/openapi-gen -check
                        node tools/check-generated-types-fresh.mjs
gate                 -> go test ./internal/api/ -run TestOpenAPIYAMLIsUpToDate -count=1   (CGO)
contrats 0.B         -> go test ./contracttest/ -run 'TestReplayDocumentFieldCountIsFrozen|TestReplayContract'
                        npx vitest run src/features/match-replay/test/goFixtures.contract.test.ts
```

Ordre impose (heritage de la checklist 2.6.3) : **`openapi.yaml` se regenere EN DERNIER**, apres
que la forme Go est figee et que la chronique porte l'entree.

---

## 4. Lot 4.4 — La recuisson selective par couche

### 4.1 `replaybuild.Digest` aujourd'hui

`internal/replaybuild/artifact_digest.go`. Le type, colle :

```go
type Digest struct {
	MatchID       string
	SchemaVersion int
	Players       int
	Tracks        int
	Bytes         int
}

func (d Digest) UpToDate() bool { return d.SchemaVersion == replay.SchemaVersion }
func (d Digest) HasPlayerCounters() bool { return d.Players > 0 }
```

Il est rempli par `digestFromBytes`, qui deserialise **quatre cles seulement** de l'artefact
(`matchId`, `schemaVersion`, `tracks`, `scoreTimeline.players`) — c'est ce qui rend la lecture
bon marche sur un document de 1,2 a 2,0 Mio. `ArtifactDigest(path)` incremente
`replay_artifact_digest_reads_total` (expvar), pour prouver qu'un cycle n'ouvre qu'une fois.

### 4.2 Comment « a recuire » se decide aujourd'hui

**Par une EGALITE de `SchemaVersion`, et rien d'autre.** Les appelants de production, colles :

```
internal/api/wire/registry_build_queue.go:395   if replaybuild.ArtifactUpToDate(path) { ... }
internal/sync/replayartifacts/artifacts.go:280  d, ok := replaybuild.ArtifactDigest(path)
internal/sync/replayartifacts/artifacts.go:281  if !ok || !d.UpToDate() { ... }
cmd/levelup/cmd_backfill_replay.go:232          if !o.force && replaybuild.ArtifactUpToDate(path) { ... }
cmd/levelup/cmd_backfill_replay_repair.go:78    d, ok := replaybuild.ArtifactDigest(path)
cmd/levelup/cmd_backfill_replay_repair.go:85    if !d.UpToDate() { ... }
```

Cote ECRITURE, la garde symetrique est `wouldDowngrade` puis `writeArtifactBytes`
(`artifact_store.go:88` et `:168`) ; `StoreArtifact` ajoute `validateArtifact` (le seul ecrivain
qui peut porter une autre version).

Cote WEB, la meme decision est reprise, en quatre etats, par `computeReplaySchemaStatus` :
`invalid` (contrat) puis `stale` si sous `MIN_RENDERABLE_SCHEMA_VERSION`, puis `unknown` si
l'en-tete manque, puis la comparaison ordinaire.

**Consequence mesurable de l'egalite unique** : un correctif sur UNE carte monte
`SchemaVersion`, donc **les 1 386 films du cache** deviennent « a recuire », a 26,5 s par film,
soit **~10 h de machine** — et la machine ne decode qu'un film a la fois (verrou solo). C'est
exactement le constat de l'architecture §11.

### 4.3 Ce qu'il faudrait pour decider par couche

Quatre changements, tous petits, et leur ordre compte :

1. **Le document porte `layers`** (4.2.1) — sans lui, il n'y a rien a lire.
2. **`digestFromBytes` lit `layers`** : une cinquieme cle dans la structure anonyme, au meme cout
   (le parseur n'ouvre toujours pas le reste du document).
3. **`Digest` gagne `Layers map[string]string`** et **`UpToDate()` cesse d'etre une egalite
   unique**. La decision devient une fonction a trois sorties, pas un booleen :

   | Verdict | Condition | Conduite |
   |---|---|---|
   | `aJour` | `SchemaVersion` egal ET toutes les revisions de calque egales | rien |
   | `republier` | revisions de `grammar` et de `facts` egales, seule la publication a bouge | **rejouer depuis les faits** (4.1), pas de verrou, ~1 s |
   | `redecoder` | une revision de `grammar` ou de `facts` a bouge | verrou solo, decodage complet, ~26,5 s |

   C'est la seule forme qui tienne les deux promesses de D-7 : « un changement de publication ne
   redecode pas » et « un changement de grammaire ne recuit que ce qui en depend ».
4. **Les cinq appelants passent du booleen au verdict.** `ArtifactUpToDate` peut rester (elle a
   des appelants qui n'ont besoin que d'elle) mais devient `verdict == aJour`.

Deux pieges a ecrire au lot :

- **Le verdict `republier` ne doit pas exister sans faits sur le disque.** Si le fichier de faits
  manque (purge, film arrive avant 4.1), le verdict retombe sur `redecoder`. Un troisieme etat
  « je republierais si j'avais les faits » serait un piege a diagnostic.
- **La regression reste interdite dans les deux sens.** `wouldDowngrade` compare aujourd'hui a
  schema EGAL ; avec des revisions par calque, il faut aussi refuser d'ECRASER un calque par une
  version produite sous une revision plus ancienne. Sans cela, un `republier` joue avec un binaire
  perime appauvrirait un artefact que le parc avait deja mieux.

### 4.4 Le badge admin — ou vit l'etat, et les chaines

| Role | Fichier | Detail |
|---|---|---|
| L'etat, pur et teste sans React | `features/match-replay/model/replaySchemaStatusLogic.ts` | `ReplaySchemaStatus` = `unknown` \| `upToDate` \| `stale` \| `invalid` ; `MIN_RENDERABLE_SCHEMA_VERSION = 27` |
| Le rendu | `features/match-replay/ui/ReplaySchemaBadge.tsx` | `if (!isAdmin) return null` — **rien, pas meme un element vide, pour un non-admin** ; tons `success` / `warning` / `destructive` / neutre |
| Les chaines FR | `features/match-replay/i18n/i18n.ts` l. **26-32** | 6 cles |
| Les chaines EN | `features/match-replay/i18n/i18n.ts` l. **465-471** | 6 cles, parite par typage |
| Le contrat i18n | `features/match-replay/i18n/i18nContract.ts` | |
| Les tests | `model/replaySchemaStatusLogic.test.ts`, `test/testDoc.guard.test.ts` | |

Les cles, collees (FR puis EN) :

```
schemaBadgeUpToDateFmt      `Schéma ${schemaVersion} · à jour`
schemaBadgeStaleFmt         `Schéma ${schemaVersion} · à recuire (dernier : ${latestSchemaVersion})`
schemaBadgeStaleNoTargetFmt `Schéma ${schemaVersion} · à recuire`
schemaBadgeUnknownFmt       `Schéma ${schemaVersion}`
schemaBadgeInvalidFmt       `Schéma ${schemaVersion} · contrat non respecté (${detail})`
contractUnknownKeysFmt / contractInvalidFieldFmt / contractMalformed   (les manquements, mis en mots ici)

schemaBadgeUpToDateFmt      `Schema ${schemaVersion} · up to date`
schemaBadgeStaleFmt         `Schema ${schemaVersion} · rebuild needed (latest: ${latestSchemaVersion})`
schemaBadgeStaleNoTargetFmt `Schema ${schemaVersion} · rebuild needed`
schemaBadgeUnknownFmt       `Schema ${schemaVersion}`
schemaBadgeInvalidFmt       `Schema ${schemaVersion} · contract violated (${detail})`
```

**Ce que 4.4.2 ajoute** : un cinquieme etat, ou un detail au `stale` — l'etat **par couche**.
Deux formes possibles, la premiere recommandee :

- (i) enrichir `stale` d'une liste de calques perimes : `Schéma 61 · à recuire (calques :
  véhicules, équipement)` / `Schema 61 · rebuild needed (layers: vehicles, equipment)`. Une cle de
  plus par langue, aucun etat neuf, aucun test d'etat a reecrire ;
- (ii) un etat `layersStale` distinct de `stale` : plus explicite, mais il faut alors trancher ce
  que dit le badge quand `SchemaVersion` ET des calques sont perimes.

Regle deja posee par le fichier et a respecter : **le manquement est une DONNEE, pas une phrase**
(« ce module ne decide que du STATUT ; c'est le badge qui met le manquement en mots, par
`i18n.ts`, dans les deux langues », constat R2-2 de la revue). La liste des calques perimes sort
donc de `computeReplaySchemaStatus` en tableau de noms, jamais en chaine.

Le libelle d'un calque doit etre en FR sans anglicisme (regle 1 du depot) : `vehicles` cote
donnee, « véhicules » a l'ecran.

---

## 5. Ordre conseille et durees

| Rang | Lot | Depend de | Regime de gate (V2) | Taille mesuree | Duree estimee |
|---:|---|---|---|---|---|
| 0 | **Preparation sans mutation** (matrice web, badge, note de format, ratchet du puits de faits) | rien | aucun decodage | 3 fichiers web + 1 ratchet `archlint/` | **S** — 0,5 jour, **en parallele de M2** (§6) |
| 1 | **4.2** revision par calque | 2.6 fusionne (`coverage.decoder` pose) | court + fixtures de contrat regenerees | 1 champ Go, 16 calques nommes, 232 L web, 8 fixtures | **M** — 1 a 1,5 jour |
| 2 | **4.1** faits persistes | **2.5** (frontiere `facts`) + 4.2 | **COMPLET** (lot risque : nouveau format, nouvel oracle) | ~1 650 L de `_test.go` a promouvoir + 2 sections neuves + 2 methodes `PathResolver` + 2 chemins dans `replaybuild` | **L** — 2,5 a 3,5 jours |
| 3 | **4.4** recuisson selective | 4.1 et 4.2 | court, plus une passe de parc sur signal (D6) | 5 appelants, `Digest` a 6 champs, 1 a 2 cles i18n par langue | **M** — 1 jour |
| 4 | **4.3** un seul type publie | 4.2 (et de preference apres 4.4) | court + `openapi-check` + contrats 0.B | **2 288 L a supprimer**, 58 champs a reprendre, 1 arbitrage produit | **M vers L** — 1,5 a 2 jours |

**Total estime : 6,5 a 9 jours-agent**, dont 4.1 fait la moitie.

**Pourquoi 4.2 AVANT 4.1**, contre l'ordre du plan : `layers` est ce qui donne a 4.4 de quoi
decider, et surtout ce qui donne a 4.1 sa cle de relecture cote document (« sous quelles revisions
cet artefact a-t-il ete cuit ? »). Le poser d'abord coute une montee de schema de plus, mais evite
que 4.1 doive inventer sa propre marque de revision puis la re-pointer. La note M2 §3.3 va deja
dans ce sens : « 2.6.3 est la premiere moitie de 4.2.1, posee tot pour que la recuisson selective
par couche (4.4) ait de quoi decider ».

**Pourquoi 4.3 EN DERNIER** : il supprime la conversion jumelle, donc `parity_test.go` — le garde
qui prouve aujourd'hui que rien ne se perd entre le cuit et le servi. Le supprimer avant que
`layers` et la recuisson par couche soient poses reviendrait a retirer le filet au milieu de la
traversee. Et c'est le seul lot qui porte un arbitrage produit (§7, A5) : il peut etre le point
d'arret propre du jalon si l'arbitrage tarde.

**Ordre a NE PAS suivre** : 4.1 avant 4.2. Le fichier de faits porterait alors ses revisions dans
un en-tete que le document ne sait pas nommer, et 4.2 devrait aligner les deux nomenclatures apres
coup — la meme erreur que « 2.6 avant 2.5 » dans la note M2.

---

## 6. Ce qui depend de 2.5 / 2.6

### 6.1 Ce qui DOIT attendre

| Item | Depend de | Pourquoi, mesure |
|---|---|---|
| **4.1 en entier** | **2.5 fusionne** | La couche `facts` n'existe pas avant 2.5 : `killsource` (5 787 L) et `objectiveevents` (5 081 L) sont encore deux paquets separes, et `FilmInputs` vit dans `replay`. Un fichier de faits ecrit avant la frontiere figerait une nomenclature que 2.5 deplacerait aussitot |
| **4.1.1, l'en-tete** | **2.6** | `grammarRev` et `factsRev` n'existent pas avant 2.6 (aujourd'hui : `GrammarRev` dans `filmdec` et `KillSourceDecoderRev` dans `killcollector`, deux paquets que 2.5 deplace) |
| **4.2.1, le champ `Layers`** | **2.6** | Il mute `film/replay/document.go`, paquet a un seul muteur (V12), et il monte `SchemaVersion` : deux montees dans la meme fenetre (61 en 2.6, 62 en 4.2) melangeraient les preuves d'equivalence |
| **4.3 en entier** | 2.5, 2.6, et 4.2 | Il touche `film/replay`, `domain/replaydoc`, `service/replayview`, `api/openapi.yaml` |
| **4.4.1, `Digest`** | 4.1 et 4.2 | Il ne peut pas lire des revisions que le document ne porte pas |
| Toute mesure de type `logPhase("assemblage", ...)` | 2.5 / 2.6 | `replaybuild` et `film/replay` sont sequentiels |

### 6.2 Ce qui PEUT commencer maintenant, sans muter les paquets du film

Sous frontiere de fichiers ecrite, aucun conflit avec 2.4 / 2.5 / 2.6 ni avec 2.8 / 2.9 :

1. **La matrice de compatibilite web etendue aux calques (item 4.2.2, volet web seul).**
   `features/match-replay/test/goFixtures.contract.test.ts` (222 L) et
   `model/replaySchemaStatusLogic.ts` peuvent gagner **le cas de test d'un `layers` absent** —
   c'est-a-dire la preuve que le web, sur un artefact de schema 60 sans `layers`, continue de
   rendre et n'affirme rien. Zero fichier Go touche, zero fixture a regenerer (les 8 fixtures de
   schema 60 sont deja des artefacts sans `layers`).
2. **Le badge admin, volet i18n et etats (item 4.4.2, volet web seul).** Les deux blocs de cles,
   la liste de calques en DONNEE, et les tests de `libelleDe`. Les chaines peuvent etre ecrites et
   testees avant que Go ne produise la donnee : `computeReplaySchemaStatus` accepte deja un
   parametre optionnel sans casser ses appelants.
3. **Le format de faits, en NOTE.** La presente section §1.3 est cette note ; ce qui reste est
   l'inventaire exhaustif des sections a ajouter au codec (killsource, part film de `filmStats`),
   fait en lecture seule sur `golden_inputs_encode_test.go` et `kill.go`. Aucune ligne de Go.
4. **Le ratchet du puits de faits** (§1.5) : un fichier neuf sous `internal/archlint/`, qui ne
   touche aucun paquet du film. Il peut etre ecrit avant son sujet et rester inerte (allowlist
   vide sur un motif qui n'existe pas encore) — c'est le regime des ratchets 2.4.3 et 2.5.0.
5. **La correction du plan** : l'item 4.1.2 cite `no_second_artifact_sink_test` pour une garantie
   qu'il ne donne pas, et la section M4 herite du « 15 s » perime (mesure : 26,5 s). Les deux se
   corrigent dans le commit du lot qui les applique, comme le 57 de 2.6.
6. **La mesure de l'assemblage seul** (§1.6) : chronometrer `-run GoldenAssembly` /
   `-run GoldenBuilds` ne mute rien et ne decode rien. **C'est le premier geste du lot 4.1**, et
   il peut etre fait aujourd'hui.

### 6.3 Ce que 2.5 changera aux chemins cites ici

Apres 2.5, les chemins de cette note deviennent : `film/internal/facts/` pour `killsource` et
`objectiveevents` ; `film/internal/grammar/` pour `filmdec` ; `film/internal/source/` pour
`filmsource` ; `film/internal/types/` pour les ~70 types de contrat (arbitrage V15, question 13 :
un golden unique `types/testdata/shapes.golden`). **`film/replay/` ne bouge pas** — c'est la
couche `replay`, et c'est elle que M4 travaille. Les references de cette note a
`film/replay/*.go`, `domain/replaydoc/`, `service/replayview/`, `replaybuild/` et au web sont donc
**stables a travers 2.5**, et c'est ce qui rend la preparation utilisable telle quelle.

---

## 7. Questions ouvertes

Une decision par ligne. `(A)` architecture, `(P)` produit, `(D)` donnee. `(U)` = a remonter a
l'utilisateur ; les autres peuvent etre tranchees par le pilote dans le cadre de l'ADR 0034.

### Architecture

| # | Question | Recommandation |
|---|---|---|
| **A1** | **(BLOQUANTE)** Le fichier de faits coupe-t-il a la sortie des balayages (`FilmInputs` + `killsource.Result` + part film de `filmStats`) ou apres l'assemblage ? | **A la sortie des balayages.** L'oracle existe deja sur 8 builds (`golden_assembly_test.go`), et couper apres l'assemblage persisterait deux fois la meme chose, contre D-7 |
| **A2** | **(BLOQUANTE)** Format : gob ou promotion du codec maison ? | **Le codec maison** (§1.3) : gob n'est pas reproductible sur les maps, tolere le champ absent en silence (contre D-7) et jette une preuve acquise sur 22 versions |
| **A3** | **(BLOQUANTE)** Ou passe la ligne de coupe dans `filmStats` (type de `replaybuild`, qui melange film, catalogue de carte et table de reglement) ? | Ecrire la coupe champ par champ AVANT de coder : `recs`, `objectives`, `objectivesUnnamed`, `objectivesRefused`, `statborgIdentity` et les entrees de calque **dans leur etat AVANT** `stats.score.TargetScore`, `HoldTicksPerPoint` et `stats.flag.Spawns` |
| **A4** | Le registre d'identite (2 921 L) reste-t-il dans `replay` (recalcule a la publication) ou descend-il en `facts` (persiste) ? | **Reste dans `replay`, recalcule.** Il est derive, pas lu ; le persister ferait de tout changement de registre une invalidation du parc de faits |
| **A5** | **(U)** 4.3 fusionne le format de cuisson et le contrat public, ce que la decision du 2026-09-05 avait explicitement separe (43 montees de schema en 5 semaines). Que devient la garantie « renommer un champ pour le client sans invalider le parc » ? | Aucune reponse dans le plan. Trois options : (i) l'abandonner par ecrit ; (ii) garder un alias de cle JSON par version ; (iii) reporter 4.3 apres M4 (arret propre, V6) |
| **A6** | La lecture des faits prend-elle le verrou `filmproc.AcquireSolo` ? | **Non** — le verrou protege le decodage. Deux chemins nommes dans `replaybuild`, et `no_unbounded_film_loop_test.go` continue de prouver que le chemin decodant le prend en premier |
| **A7** | La garantie « un seul puits » pour les faits : ratchet neuf, ou reprise litterale de la doctrine `writeArtifactBytes` ? | Ratchet neuf sous `archlint/`, allowlist vide des le premier jour (regime de D-2) ; le plan cite un ratchet qui garde autre chose (§1.5) |
| **A8** | `layers` : nomenclature des 16 cles du document, des 34 blocs de couverture, ou des 39 etapes de balayage ? | **Les cles du document**, parce que c'est le lecteur qui decide (D11 : le web n'est touche qu'a la normalisation) |

### Produit

| # | Question | Recommandation |
|---|---|---|
| **P1** | **(U)** Le badge admin gagne-t-il un cinquieme etat (`layersStale`) ou un detail au `stale` existant ? | Un **detail au `stale`** : une cle de plus par langue, aucun etat neuf, aucun test d'etat a reecrire |
| **P2** | Les noms de calque affiches : donnee en anglais (`vehicles`) et libelle FR/EN a l'ecran, ou libelle partout ? | Donnee en anglais, libelle par `i18n.ts` dans les deux langues (regle 1 : « véhicules », pas `vehicles`) |
| **P3** | `MIN_RENDERABLE_SCHEMA_VERSION` (27) bouge-t-il a M4 ? | **Non** : `layers` est additif, et aucune montee de M4 ne retire un champ promis au client. Le seuil ne se deplace que ce jour-la |
| **P4** | **(U)** La recuisson du parc a la cloture de M4 : sur signal, comme D6 l'exige — mais avec quel verdict ? | Une seule passe, et **en `republier` quand c'est possible** : si M4 ne change que la publication, 1 386 films se rejouent en ~25 min au lieu de ~10 h. C'est la demonstration meme du jalon |

### Donnee

| # | Question | Recommandation |
|---|---|---|
| **D1** | **(U)** Volumetrie : 1 386 films x 1,38 Mio ~ **1,9 Gio** de faits persistes, sur un disque qui porte deja les chunks, les manifests et les artefacts. Plafond, purge, ou sous-ensemble ? | Un **plafond declare et teste**, sur le modele de `goldenInputsBudget` (« un test qui rougit dit qu'une question se pose, pas qu'il faut changer sa constante »), plus une purge par age au modele de `replay_purge_cron`. Decision utilisateur avant d'ecrire le premier fichier |
| **D2** | **(BLOQUANTE)** Nommage : `{short8}.facts.bin` se poserait a cote de `{short8}.facts.json`, qui designe **les faits de la BASE** (`replaybuild/facts_file.go`, 3 programmes) — sens oppose | **`{short8}.filmfacts.bin`**, ou renommer l'existant. Jamais les deux extensions voisines |
| **D3** | `film_facts/{slug}/` introduit une couche par titre que `film_chunks/` et `film_manifests/` n'ont pas | Garder la couche (ADR 0008) et **ecrire l'asymetrie** dans le godoc de `FilmFactsDir` |
| **D4** | Le fichier de faits est-il un cache (`data/cache/`) ou une donnee de reference (`data/titles/{slug}/reference/`) ? | **Un cache** : regenerable depuis le film, ecrit a l'execution. D12 et `no_runtime_versioned_catalog_write_test.go` visent les catalogues, pas lui — a ne pas confondre, sous peine de bloquer le lot |
| **D5** | La cle de relecture inclut-elle l'empreinte du catalogue de bornes de carte ? | **Oui** : les positions sont des quanta, et `errGoldenInputsCarte` explique pourquoi une mauvaise entree de catalogue donne « des coordonnees FAUSSES, pas approximatives » |
| **D6** | Un film dont le build est inconnu (`ErrUnknownBuild`, D-4) : ecrit-on un fichier de faits ? | **Non.** Le film est mis de cote ; ecrire des faits sous une cle vide creerait un fichier qu'aucune relecture ne pourrait valider |
