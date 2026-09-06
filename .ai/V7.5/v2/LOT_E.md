# Lot E — Decodeur (Go) — journal d'execution

> Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, section « Lot E ». Contrat : skill
> `plan-execution`. Worktree `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-v2-decodeur`,
> branche `feat/v2-decodeur`, base `a21fd77f4`.
> Statuts : `[x]` fait et verifie · `[~]` couvert ailleurs (reference) · `[!]` non traite
> (justification).
>
> Prefixe de TOUTE commande Go : depuis `apps/go-api`,
> `GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-decodeur CGO_ENABLED=1`,
> en SERIE, en avant-plan, `-p 1 -parallel 1 -count=1`.

## Tache E-I — refonte a comportement identique

### [x] E.1 — mesures de reference archivees AVANT tout changement

Archive complete : `.ai/V7.5/v2/LOT_E_digests_avant.md` (commandes exactes, verdicts, durees,
comptes, digests). Resume :

- Etage inconditionnel : `filmdec` / `film/*` / `replay` / `killcollector` / `archlint` VERTS ;
  `go build ./...` exit 0 ; `golangci-lint run ./internal/analysis/filmdec/...` -> **6 issues**
  (4 `goconst`, 2 `unparam` — dette anterieure ; la premiere mesure annoncait « 0 issues. » et
  c'etait un artefact de cache, cf. la correction datee du fichier de digests) ;
  les 7 digests de la mini-bobine (`minifilm.tsv`, grammaire 2) confirmes par
  `TestEquivalenceMiniFilm` PASS ; ratchet `filmdecVarsGeles = 118` vert.
- Etage films reels : goldens `killsource` sur 4 films, temoin de marche delta sur 3 films,
  empreinte du registre, controle G2 de la table ECS, integration `killcollector`.

VERIFICATION DEMANDEE PAR LE MANDAT — « les 49 etapes citees par le registre, ou ce qui existe
reellement » : les 49 etapes sont celles de `cmd/replay-equiv`
(`testdata/equivalence/<film>.tsv` = 1 ligne de grammaire + 49 lignes d'etape, 13 films figes).
Ce harnais **cuit un artefact par film** (`child.go:121-127`, `replaybuild.Builder.BuildBytes`),
ce que le mandat interdit (« Aucune cuisson d'artefacts »). Il n'a pas ete joue. Ce qui existe
sous garde de variable d'environnement, et qui a ete joue, est liste ci-dessus.
`REPLAY_FILM_DIR` n'est pas une garde de verification mais la porte de REGENERATION des goldens :
laissee vide a dessein.

DEUX ECHECS ANTERIEURS AU LOT, constates sur arbre propre — ils font partie de la reference :

1. `TestGoldenFilms/fccc61cd` : ligne 53 du golden, `3 propose(s)` fige contre `2 propose(s)`
   mesure ; le compte PUBLIE (2) est inchange. Preuve mesuree du constat P0-1 (traite au lot A).
2. `TestDeltaWalkWitness` sur les trois films figes : 000d5950 {38878,30080} -> {38883,30089},
   06dfe6d9 {10613,8502} -> {10610,8489}, 64e8adfa {39806,31973} -> {39818,31990}.

Aucune de ces references n'est regeneree par le lot E : elles servent de temoin d'invariance.

**Gate E.1** — `go build ./...` : exit 0. Tests : voir le tableau du fichier de digests.

### [x] E.2 — le code mort du decodeur, retire sur pieces (E1, E2, E6)

Commit `722424c7c`. 20 fichiers, -992 / +218 lignes.

| item du plan | statut | ce qui a ete fait, verifie sur pieces |
|---|---|---|
| `entity.go` + `entity_quant.go` supprimes sauf 3 helpers | `[x]` | 586 L supprimees. Cloture morte reverifiee symbole par symbole : hors des deux fichiers, seuls `bitLen`, `readQuantStat` et `quantStatDefaultWidth` ont un lecteur ; `Binding` et `decodeQuatBlock` n'apparaissent ailleurs QUE dans des commentaires. Les trois survivants sont deplaces dans `quantize.go` (a cote de `BitLenExport`, qui enveloppait deja `bitLen`, et de `ReadQuantizedVec3`, methode soeur de `readQuantStat`). |
| les 22 `Set*` sans appelant | `[x]` | Recensement REJOUE (pas recopie) : meme liste de 22 que le registre. Zero appelant verifie deux fois — par appel `\bSetX(` sur tout le module, puis par reference nue `\bSetX\b` pour attraper une valeur de fonction : les seules occurrences restantes etaient la declaration et des commentaires. Un seul module Go dans le depot (`apps/go-api/go.mod`), `cmd/` compris. |
| branche `if useLegacyAngularVel` + drapeau | `[x]` | Branche `traverse.go` supprimee, drapeau et setter supprimes. `consumeObjectAngularVelocity` **RESTE** : il est appele par `consumeObjectAngularVelocityDynPrec` (i3 correct de ti=40). |
| drapeau `useBipedDefaultStateDeser` + branches inatteignables | `[x]` | Drapeau supprime, `if useBipedDefaultStateDeser && ti == 35` devient `if ti == 35`. Les deux branches `defaultStateBitsByTI` (`traverse.go` et la sonde `TraverseKeyframeBipedAt`) supprimees, et la table avec elles : son seul peupleur etait `SetDefaultStateBitsForTI`, sans appelant. |
| `dynPrecHook` et `repTraceHook` + leurs 8 blocs | `[x]` | Les deux crochets supprimes. `repTraceHook` emporte `traceRep` et ses 9 appels. Les 8 blocs de sauvegarde : 3 disparaissent avec `frame_debug.go` et l'elagage de `probe_export.go`, les 5 autres perdent leur part `savedDP` (`frame_chain_infer.go` x2, `frame_records.go` x3) — les autres crochets sauves au meme endroit (`posCaptureHook`, `unitRefHook`, `accumWorld`) sont INTACTS. |
| `probe_export.go` (9 exports sans appelant) | `[x]` | 9 fonctions + la constante `VariantBitOffsetInWST` supprimees. `ConsumeComponentAt` et `TraverseKeyframeBipedAt` RESTENT : chacune a un appelant vivant dans `internal/analysis/replay` (hors perimetre du lot). Le fichier passe de 111 a 55 L. |
| `frame_debug.go` | `[x]` | Supprime (159 L, 2 fonctions + 3 types, zero appelant tests compris). Son exemption devenue morte est retiree de `archlint/no_local_longest_run_test.go` (le ratchet REDESCEND). |
| ratchet `filmdec_package_vars_test.go` redescendu | `[x]` | 118 -> **113**, avec la chronique datee : les cinq variables retirees sont nommees une par une, et la raison de garder les 17 autres (elles sont LUES par le decodage ; leur retrait serait la de-globalisation D10, hors plan) est ecrite. |

DECISION DE PERIMETRE, ecrite parce qu'elle sera contestee : le plan dit « supprimer les 22
`Set*` » — pas « et leurs variables ». Les 17 variables que ces setters ecrivaient sont LUES par
le decodage avec leur valeur de production (`DeltaQuantum`, `defaultReplRange`,
`inferRequireBoundSuccessor`, `absDequantMode`...). Les supprimer changerait le comportement ;
les figer en `const` serait une de-globalisation, explicitement hors plan (D10). Elles restent,
et chaque commentaire qui nommait le reglage disparu est corrige — aucune doc inversee.

DECOUVERTE (consignee, NON traitee) — LA CLOTURE MORTE QUE LES SETTERS LAISSENT DERRIERE EUX.
Quatorze de ces 17 variables n'ont plus AUCUN ecrivain et gardent leur valeur initiale ; les
branches qu'elles gardent sont donc desormais inatteignables : `absDequantMode`
(`position_capture.go`, une branche), `absPerIndexAxisW` (une branche), `accumWorld` (2 branches
+ `accumSlot` / `setAccumSlot`), `biDebug` (4 branches + `biCurSeq`, `BiBadSeqs`, `BiOkSeqs` —
bloc que le code lui-meme annote « a retirer apres »), `bipedActionLoop2Count` (boucle a 0 tour),
`bipedDefaultStateDecodeMovement`, `bipedDefaultStateTailBits`, `bipedMediaFramePresent`,
`deadStatePreSkip`, `deadStateVelocityPresent`, `inferRepair`, `inferResyncTargets`,
`vehicleMediaFrameBits`, et `inferRequireBoundSuccessor` (dont la disjonction est constante).
Trois gardent un ecrivain de PRODUCTION et sont donc vivantes : `posCaptureHook` (installe par
`scanForTargetDelta`), `DeltaQuantum` et `defaultReplRange` (lues telles quelles). Le retrait de
cette cloture n'est PAS dans le perimetre du lot E-I ; plusieurs de ces variables portent une
largeur MESUREE d'un chemin non nominal (`vehicleMediaFrameBits`, `bipedDefaultStateTailBits`),
c'est-a-dire de la connaissance de retro-ingenierie, pas du code mort ordinaire. A trancher par
le superviseur.

DECOUVERTE (traitee, parce qu'elle bloquait le gate) — `golangci-lint` signalait
`readQuantStat - param3 always receives 1` : le second lecteur, qui passait une autre valeur,
vivait dans `entity_quant.go`. `param3` MODELISE `param_3` de FUN_1406d3140, dont depend le cout
en bits (la sonde d'un bit n'est depensee que pour param3==1) : le figer effacerait une part de
la grammaire portee. `//nolint:unparam` avec la justification ecrite au-dessus, sur le patron
deja etabli du depot (`skill_rating.go`, `performance_score.go`, `handlers/helpers.go`).

**Gate E.2** — toutes les commandes rejouees, comparees LIGNE A LIGNE a l'archive E.1 :

```
go build ./...                                                    -> exit 0
go vet ./internal/analysis/filmdec/... ./internal/archlint/...     -> exit 0
gofmt -l ./internal/analysis/filmdec/ ./internal/archlint/         -> (vide)
go test ./internal/analysis/filmdec/... ./internal/games/halo_infinite/film/...
     ./internal/analysis/replay/... ./internal/sync/killcollector/...
     ./internal/archlint/... -p 1 -parallel 1 -count=1             -> les 10 paquets ok
KILLSOURCE_FIXTURES=<film_chunks> go test .../killsource/
     -run 'TestGoldenFilms|TestReferenceFilms|TestLigneDiscriminante...' -v
     -> sortie IDENTIQUE a l'avant, y compris l'echec anterieur fccc61cd
DELTA_WITNESS_FILM=<film> go test .../filmdec/
     -run 'TestDeltaWalkWitness|TestRegistryFingerprintOnFilm' -v   (x3 films, un par process)
     -> les trois sorties IDENTIQUES a l'avant
ECS_TABLE_FILM=<film> go test .../filmdec/ -run 'TestG1...|TestG2...|TestG3...' -v
     -> ok, sortie identique (50 blocs, 49 porteurs, 1067/1067 lignes, +14 alias)
KILLSOURCE_FIXTURES=<film_chunks> go test -tags=integration .../killcollector/ -v
     -> ok, 67 PASS / 2 SKIP / 0 FAIL, verdicts identiques (seule la duree change)
golangci-lint run --new-from-rev=a21fd77f4 .../filmdec/... .../archlint/...  -> 0 issues.
golangci-lint run --new-from-merge-base=origin/main .../filmdec/...          -> 0 issues.
```

DECOUVERTE METHODE — `golangci-lint` ment si on ne lui donne pas un cache propre. La mesure de
reference de E.1 annoncait « 0 issues. » sur `filmdec`. C'etait un artefact : le cache de
RESULTATS de golangci-lint est **global a la machine** (`%LocalAppData%\golangci-lint`,
independant de `GOCACHE`) et indexe par fichier, donc il sert des verdicts calcules dans un autre
jeu de fichiers du meme paquet — et `goconst` / `unparam` sont des analyses de PAQUET. Rejoue
avec `GOLANGCI_LINT_CACHE` propre au lot, l'etat de base rend **6 issues** (4 `goconst` +
2 `unparam`), et l'etat apres E.2 rend **exactement les 6 memes** (memes linters, memes comptes,
diff vide). Le fichier de digests est corrige. Tout lot qui mesure une reference de lint doit
isoler ce cache.

### [x] E.3 — un seul lecteur de preambule, une seule table de domaines (E3)

Commit `ff89b4625`. 10 fichiers, +145 / -50 lignes.

| item du plan | statut | ce qui a ete fait |
|---|---|---|
| les six lectures passent par le canonique | `[x]` | `readPacketHead(br)` (event_list.go) est le lecteur unique ; `PacketHeadEventType([]byte)` en est l'enveloppe. Les six sites migres : `event_list.go`, `zoom_events.go`, `transloc_events.go`, `biped_pickups.go`, `fire_aim_modal.go`, `weapon_hits.go` (deux balayages). |
| une seule table de domaines, `dom3 = 7` | `[x]` | `refDomWidth(dom)` (event_list.go), composee des quatre constantes nommees. Les deux copies de production disparaissent (`lot1RefDomWidths`, `zoomRefWidth`), ainsi que la largeur locale de `transloc_events.go` (devenue `refDomWidth(translocRefDomain)`). |
| test qui confronte les tables recopiees | `[x]` | `event_preamble_guard_test.go`, trois controles : la table rend les largeurs mesurees (dom3 = 7) et 0 hors table ; aucune source de production ne recopie une table domaine vers largeur ; aucune ne recopie la SEQUENCE saut-de-tete + R(7). |
| justification perimee de `biped_pickups.go` | `[x]` | Remplacee : elle disait que la constante « n'avait aucun lecteur et la CI l'a relevee », alors que `eventPayloadStartBit` en a deux depuis le portage des evenements vehicule. |

COMMENT LA FACTORISATION RESTE BIT-EXACTE, alors que les six sites suivaient DEUX conventions.
`readPacketHead` lit TOUJOURS les neuf bits et rend `{Config, More, Type}` ; c'est l'appelant qui
applique sa politique :

- TROIS sites testaient la continuation et sortaient SANS lire le type (`decodeZoomHead`,
  `decodeTranslocHead`, `decodeBipedPickup`). Ils abandonnent leur lecteur au meme instant : les
  7 bits lus en trop ne sont observes par personne.
- TROIS la SAUTAIENT (`Skip(2)`) et lisaient le type quoi qu'il arrive (`modalPostCountsBit`, les
  deux balayages de `weapon_hits.go`). Chacun est derriere un filtre sur l'OCTET DE TETE — 0xD2
  pour le type 36, 0xC0 pour le type 0 — dont le bit 1 vaut 1 par construction.

FAIRE TESTER LA CONTINUATION AUX TROIS DERNIERS AURAIT CHANGE LE COMPORTEMENT, et le journal doit
le dire : le harnais `writeModalHeader` (fire_aim_modal_test.go) ecrit `bits(0, 2)` en prefixe,
continuation comprise. Ses temoins synthetiques passeraient de « lus » a « refuses », et il aurait
fallu editer la fixture pour que le refacto passe — exactement ce que le paragraphe 6 du plan
master interdit. Le choix est donc de FACTORISER SANS UNIFORMISER, et d'ecrire la divergence a UN
endroit (l'en-tete de `readPacketHead`) au lieu de six.

VERIFICATION DEMANDEE — « la production ne lit pas le domaine 3 aujourd'hui, verifie que ca reste
vrai ». VERIFIE : les domaines effectivement lus sont 1, 2, 4, 7, 8 (`lot1RefDom(br, 7)` a
`weapon_hits.go:288`, `zoomRefDomains = {4, 8, 7}`, `boardRefs` en 2/3/7). Le SEUL lecteur du
domaine 3 est `boardRefs`, qui portait DEJA la valeur mesuree 7. Passer les deux copies de 8 a 7
ne change donc AUCUNE lecture servie — confirme par l'invariance des goldens.

DECOUVERTE (consignee, NON traitee) — DEUX INSTRUMENTS DE RECHERCHE gardent leur propre table
avec `3: 8` : `bpkDomWidths` (biped_pickup_research_test.go) et `r7DomWidth`
(r7_grammaire_research_test.go). Le second LIT le domaine 3 : `r7Domains[8] = {2, 3, 7}`,
l'embarquement. Les migrer changerait une largeur DANS UNE MESURE DATEE — hors du perimetre
« comportement strictement identique » de E-I. La portee du garde-rail le dit explicitement.

**Gate E.3** — mutation d'abord (`dom3RefWidth = 8`, un preambule recopie, une table recopiee :
les TROIS controles echouent, puis retour), puis :

```
go build ./...                                                     -> exit 0
go test ./internal/analysis/filmdec/... ./internal/games/halo_infinite/film/...
     ./internal/analysis/replay/... ./internal/sync/killcollector/...
     ./internal/archlint/... -p 1 -parallel 1 -count=1              -> les 10 paquets ok
goldens killsource, 4 films reels                                   -> IDENTIQUE a l'avant
temoin de marche delta, 3 films (un par process)                    -> IDENTIQUE a l'avant
table ECS G1/G2/G3 sur film reel                                    -> IDENTIQUE
integration killcollector (-tags=integration)                       -> IDENTIQUE
golangci-lint --new-from-merge-base=origin/main .../filmdec/...     -> 0 issues.
```

Ratchet `filmdecVarsGeles` resserre 113 vers **111** (les deux tables de grammaire deguisees en
`var` redeviennent du code).

### [x] E.4 — un seul marcheur de records delta bipede (E4)

Commit `5a9c86e82`. 11 fichiers, +109 / -274 lignes.

| item du plan | statut | ce qui a ete fait |
|---|---|---|
| `walkDeltaBipedRecords(fc, visit)` remplace les neuf triples boucles | `[x]` | `delta_biped_walk.go`, DEUX etages : `walkDeltaBipedPayload` (un payload en main, l'etage PUR — c'est celui que `ScanBipedRecords` utilise) et `walkDeltaBipedRecords` (chunks du contexte + paquets delta, delegue au premier). Les neuf sites migres : ability_charges, ability_impulses, ability_rank, camo_state, grapple_state, held_weapon_changes, inventory_delta, offline_biped, unit_equipment_scan. |
| garde-rail sur le litteral du seuil | `[x]` | `delta_biped_walk_guard_test.go` : aucune source de production hors du marcheur ne compose `bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt` ni n'appelle `matchBipedHeader` (`offline_biped.go` exempte : elle le DECLARE). |

POURQUOI DEUX ETAGES ET NON UN. Le plan cite `offline_biped.go` parmi les neuf, mais son balayage
(`ScanBipedRecords`) est PUR : il marche un payload deja en main, sans film ni chunk, et c'est le
coeur testable du decodeur. Un marcheur unique a un seul etage l'aurait force a inventer un
contexte de film. Les deux etages partagent la boucle de curseur, la borne et l'avance ; seules
les deux boucles externes appartiennent au second.

La borne (`deltaBipedMinRecord`), l'avance sans chevauchement (`i0 + lay.TotalBits()`) et le pas
d'echec d'un bit sont ceux d'origine, a la ligne pres. Le cas `DropSaturated` de
`ScanBipedRecords` devient un `return` du visiteur : l'avance qui le suivait etait deja celle du
marcheur.

DECOUVERTE (consignee, NON traitee) — une VINGTAINE d'instruments de recherche ancrent encore
leurs propres records (`i22_delta`, `i48_rank`, `i56_energy`, `r8_*`, `r9_*`, `r11_*`, `r12_*`...).
Plusieurs mesurent une grammaire candidate avec une porte deliberement differente ; les faire
passer par le marcheur les ferait mentir sur ce qu'ils mesurent. Le garde-rail est donc borne aux
sources de production, et le dit.

**Gate E.4** — CORRECTION DU 2026-09-06 (revue E-R1, constat C2, P1). Ce paragraphe disait :
« c'est l'item ou le TEMOIN DE MARCHE DELTA est l'oracle ». **C'ETAIT FAUX, ET MESURABLEMENT.**
`TestDeltaWalkWitness` (`delta_walk_witness_test.go:167`) chiffre `DecodeFrameRecords`
(`frame_records.go`), un marcheur DIFFERENT de `walkDeltaBiped*` : neutraliser entierement
`walkDeltaBipedPayload` laisse ses trois chiffres inchanges, au record pres. Ses mesures
identiques ne prouvaient donc rien sur E.4 — elles prouvaient l'invariance d'un autre chemin, ce
qui reste utile mais n'est pas l'oracle annonce.

L'ORACLE REEL DE E.4, ce sont :

- le **golden des familles** (E.6, `golden_minibobine_test.go`) — neutraliser le marcheur rougit
  13 familles (mutation M7 du verdict). Il n'existait pas quand E.4 a ete cloture, ce qui explique
  qu'un oracle ait ete cherche ailleurs ;
- le **temoin synthetique de l'avance**
  (`TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie`, correction 1 ci-dessous) — le seul qui
  attrape `p = i0 + 1`, que le golden laisse passer.

Le temoin de marche delta reste joue et compare a chaque item : il est un temoin d'INVARIANCE du
paquet, pas la preuve de cet item-la. Les commandes du gate, elles, sont inchangees :

```
gofmt -l ./internal/analysis/filmdec/                               -> (vide)
go build ./...                                                      -> exit 0
les 10 paquets du gate                                              -> ok
goldens killsource, 4 films reels                                   -> IDENTIQUE
temoin de marche delta, 3 films : 000d5950 {14350, 38883, 30089},
     06dfe6d9 {6606, 10610, 8489}, 64e8adfa {14357, 39818, 31990}   -> IDENTIQUE
table ECS, integration killcollector                                -> IDENTIQUE
golangci-lint --new-from-merge-base=origin/main .../filmdec/...      -> 0 issues.
```

### [x] E.5 — le verrou de decodage, ratchet AST (E5)

Commit `9d01c9733`.

| item du plan | statut | ce qui a ete fait |
|---|---|---|
| garde-rail du verrou generalise en ratchet AST dans `archlint` | `[x]` | `archlint/decode_lock_held_test.go`. Regle par POINT FIXE sur le graphe d'appel intra-paquet : une fonction est SOUS VERROU si elle prend `LockProcessDecode`, ou si TOUS ses appelants du paquet le sont ; toute fonction qui appelle un balayage filmdec (`Scan*`, `DecodeFrame*`, `TraverseEntity*`) doit etre sous verrou. Trois paquets mesures : `analysis/replay`, `film/killsource`, `sync/killcollector`. |
| `killcollector/positions.go` mis en conformite | `[x]` | `buildPositionRows` prend le verrou en tete. Non-reentrance VERIFIEE sur pieces : `killsource.Decode` (`decode.go:78-79`) le relache avant de rendre, et `collectPositions` est appele APRES lui dans `collect()` — jamais dedans. |

POURQUOI LA REGLE N'EXIGE PAS QUE CHAQUE BALAYEUR PRENNE LE VERROU. Le mutex n'est pas reentrant :
l'exiger de chaque fonction bloquerait le process des le premier chemin a deux niveaux (dans
`analysis/replay`, `BuildFromFilm` le prend et appelle une trentaine de balayeurs). La regle exige
donc la COUVERTURE, pas la prise — ce qui est exactement le contrat de `decode_gate.go` (« pour
TOUTE la duree du decodage d'un film — jamais par sous-appel »).

**Gate E.5** — mutation d'abord : le verrou retire de `buildPositionRows`, le ratchet designe
`internal/sync/killcollector/positions.go:182 buildPositionRows` et rien d'autre ; remis, il
passe. Puis les 10 paquets verts, `go build ./...` exit 0, gofmt propre, et les quatre temoins de
films reels IDENTIQUES a la reference E.1.

## Gate de cloture de la tache E-I — rejoue INTEGRALEMENT le 2026-09-06

Toutes les commandes ci-dessous ont tourne EN AVANT-PLAN, en serie, sur l'arbre a `9d01c9733`
(les cinq items commites), avec le prefixe du lot :
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-decodeur CGO_ENABLED=1`, depuis
`apps/go-api`, `-p 1 -parallel 1 -count=1`.

| commande | derniere ligne / verdict |
|---|---|
| `go build ./...` | exit 0 |
| `gofmt -l ./internal/ ./cmd/` | (aucune sortie) |
| `go test ./internal/analysis/filmdec/... ./internal/games/halo_infinite/film/... ./internal/analysis/replay/... ./internal/sync/killcollector/... ./internal/archlint/...` | `ok levelup/go-api/internal/archlint 5.891s` — **les 10 paquets ok**, exit 0 |
| `go test .../killsource/ -run TestGoldenMiniBobine + TestGoldenPhrasesJustes -v` | `ok ... 0.469s` |
| `go test .../replay/ -run <les 11 goldens inconditionnels> -v` | `ok ... 6.766s` — 20 PASS, 1 SKIP (la porte de REGENERATION `TestGoldenInputsRegenerate`, voulue) |
| `KILLSOURCE_FIXTURES=<film_chunks> go test .../killsource/ -run TestGoldenFilms + TestReferenceFilms + TestLigneDiscriminante... -v` | `FAIL ... 41.8s` — **identique a la reference E.1, echec anterieur `fccc61cd` compris** |
| `DELTA_WITNESS_FILM=<film> go test .../filmdec/ -run TestDeltaWalkWitness + TestRegistryFingerprintOnFilm -v` (x3, un film par process) | les trois sorties **identiques** a la reference E.1 |
| `ECS_TABLE_FILM=<film> go test .../filmdec/ -run TestG1... + TestG2... + TestG3... -v` | `ok` — **identique** (50 blocs, 49 porteurs, 1067/1067 lignes, +14 alias ; G3 : 27 references / 1479 champs) |
| `KILLSOURCE_FIXTURES=<film_chunks> go test -tags=integration .../killcollector/ -v` | `ok` — 67 PASS / 2 SKIP / 0 FAIL, **verdicts identiques** |
| `golangci-lint run --timeout 5m ./internal/analysis/filmdec/...` (cache propre au lot) | 6 issues — **memes linters, memes comptes que l'etat de base** (diff vide) |
| `golangci-lint run --timeout 5m --new-from-merge-base=origin/main ./...` (la commande EXACTE de la CI) | **`0 issues.`**, exit 0 |

### Le tableau des digests, AVANT et APRES

| temoin | reference E.1 (2026-09-05, avant tout changement) | apres E.2 a E.5 (2026-09-06) | verdict |
|---|---|---|---|
| mini-bobine `fire` | 519 / `f4923e82...aebaa6` | idem (`TestEquivalenceMiniFilm` PASS) | IDENTIQUE |
| mini-bobine `grenades` | 70 / `813221b5...326511` | idem | IDENTIQUE |
| mini-bobine `loadouts` | 150 / `e5dadc04...67a79e` | idem | IDENTIQUE |
| mini-bobine `inventory` | 184 / `a2b99e97...8df936` | idem | IDENTIQUE |
| mini-bobine `deaths` | 93 / `66ade085...4d1aa62e` | idem | IDENTIQUE |
| mini-bobine `playerIndices` | 1 / `b5cd498a...be4f2a` | idem | IDENTIQUE |
| mini-bobine `projectiles` | 22 / `f5c17800...a2eb61` | idem | IDENTIQUE |
| golden killsource `000d5950` | PASS | PASS | IDENTIQUE |
| golden killsource `9b191a7f` | PASS | PASS | IDENTIQUE |
| golden killsource `78919882` | PASS | PASS | IDENTIQUE |
| golden killsource `fccc61cd` | FAIL, 1 ligne (`3 propose(s)` fige contre `2` mesure) | FAIL, LA MEME ligne | IDENTIQUE |
| `TestReferenceFilms` x4 (calibration, couverture, gate (b), concordance) | 16 lignes chiffrees | les 16 memes | IDENTIQUE |
| temoin delta `000d5950` | paquets 14350, records 38883, aboutis 30089 (77,383 %) | idem | IDENTIQUE |
| temoin delta `06dfe6d9` | paquets 6606, records 10610, aboutis 8489 (80,009 %) | idem | IDENTIQUE |
| temoin delta `64e8adfa` | paquets 14357, records 39818, aboutis 31990 (80,341 %) | idem | IDENTIQUE |
| empreinte de registre x3 | `0x61e492dd4de7fd4e` (x2, concordantes) et `0x5827362c37d2adb3` | idem | IDENTIQUE |
| table ECS G2 sur `000d5950` | 50 blocs, 49 porteurs, 1067 = 1067 (+14 alias) | idem | IDENTIQUE |
| integration killcollector | 67 PASS / 2 SKIP / 0 FAIL | idem | IDENTIQUE |

Les empreintes sont tronquees dans ce tableau ; le fichier fige
`testdata/equivalence/minifilm.tsv` en porte la forme complete, et `TestEquivalenceMiniFilm` les
RECALCULE a chaque passe — son PASS est donc la preuve, pas le fichier.

**AUCUN CHIFFRE N'A BOUGE.** Les deux echecs anterieurs au lot (golden `fccc61cd`, temoin delta
sur les trois films) sont retrouves a l'identique : le lot E ne les repare pas — c'est le lot A
qui traite leur cause (P0-1) — et ne les masque pas non plus.

## Ratchets, apres le lot

| ratchet | avant | apres | sens |
|---|---|---|---|
| `archlint/filmdec_package_vars_test.go` | 118 | **111** | REDESCEND (E.2 : -5, E.3 : -2) |
| `archlint/no_local_longest_run_test.go` | 2 exemptions | **1** | REDESCEND (E.2 : `frame_debug.go` supprime) |
| `archlint/no_unbounded_film_loop_test.go` | vert | vert | inchange |
| `archlint/decode_lock_held_test.go` | (n'existait pas) | vert sur 3 paquets | NOUVEAU (E.5) |
| `filmdec/event_preamble_guard_test.go` | (n'existait pas) | vert, 3 controles | NOUVEAU (E.3) |
| `filmdec/delta_biped_walk_guard_test.go` | (n'existait pas) | vert, 2 controles | NOUVEAU (E.4) |

Aucun ratchet n'a monte. Aucune allowlist n'a grandi ; une a retreci.

## Chiffres du lot

| item | fichiers | lignes |
|---|---|---|
| E.2 code mort | 20 | +218 / -992 |
| E.3 preambule et domaines | 10 | +145 / -50 |
| E.4 marcheur delta | 11 | +109 / -274 |
| E.5 verrou | 2 | +45 / -1 |
| **total code** | **43 touches** | **+517 / -1317** |

## Decouvertes du lot — consignees, NON traitees

1. **Le golden `fccc61cd` et le temoin de marche delta etaient DEJA rouges** sur l'arbre
   d'arrivee. C'est la preuve mesuree du constat P0-1 (« `KillSourceDecoderRev` fige alors que le
   decodeur a change ») que le lot A traite. Le lot E les garde comme temoins d'invariance et ne
   regenere rien.
2. **La cloture morte laissee par les 22 reglages supprimes** : 14 variables de paquet n'ont plus
   aucun ecrivain et gardent leur valeur initiale ; les branches qu'elles gardent sont
   inatteignables. Plusieurs portent une largeur MESUREE d'un chemin non nominal
   (`vehicleMediaFrameBits`, `bipedDefaultStateTailBits`) : c'est de la connaissance de
   retro-ingenierie, pas du code mort ordinaire. A trancher par le superviseur.
3. **Deux instruments de recherche portent encore une table de domaines avec `3: 8`**, et l'un
   LIT le domaine 3 (`r7Domains[8] = {2, 3, 7}`). Les migrer changerait une mesure datee.
4. **Une vingtaine d'instruments de recherche ancrent leurs propres records delta bipede**,
   plusieurs avec une porte deliberement differente.
5. **`golangci-lint` ment sans cache propre.** Son cache de RESULTATS est global a la machine
   (`%LocalAppData%\golangci-lint`, independant de `GOCACHE`) et indexe par fichier : il sert des
   verdicts calcules dans un AUTRE jeu de fichiers du meme paquet, alors que `goconst` et
   `unparam` sont des analyses de PAQUET. Toute mesure de lint de reference doit isoler
   `GOLANGCI_LINT_CACHE` — sinon elle annonce « 0 issues. » sur un paquet qui en a 6.
6. **`TestKillSourcePositionsFilmReelEtRelitParLaVue` se SKIPPE** : son film code en dur
   (`9b191a7f`) est joue sur une carte absente du catalogue de bornes, donc 0 ligne de position.
   C'est le seul test de bout en bout de `buildPositionRows` sur film reel — celui que E.5 vient
   de mettre en conformite. Le fichier est hors du perimetre ferme du lot E (seul `positions.go`
   y entre) ; il releve du lot F (baseline de presence des tests).
7. **Les « 49 etapes d'equivalence en local » du registre appartiennent a `cmd/replay-equiv`**,
   qui cuit un artefact par film — interdit par le mandat du lot. Le gate de films reels du lot E
   repose donc sur ce qui existe sous garde de variable d'environnement, liste au fichier de
   digests.

## Statut de la tache E-I

Les cinq items sont `[x]`, faits et verifies sur pieces. Aucun `[~]`, aucun `[!]`.
Aucun test desactive, aucun skip ajoute, aucun golden regenere, aucune allowlist agrandie.

Prochaine etape : revue adversariale du superviseur, puis la tache E-II (E.6, E.7) — qui ne
demarre PAS sans son message.

---

# Tache E-II — preuves en CI (2026-09-06)

Meme worktree, meme branche, memes regles. Les temoins de E.1 restent le gate de chaque item ;
`GOLANGCI_LINT_CACHE=/c/Users/Guillaume/AppData/Local/golangci-v2-decodeur` isole desormais la
mesure de lint (lecon du lot E-I : ce cache est global a la machine et sert des verdicts perimes).

## [x] E.6 — golden inconditionnel des familles de balayage (F3, residu F1)

Commit `8a65d0969`. `filmdec/golden_minibobine_test.go` + `testdata/golden_minibobine_familles.tsv`.

| item | statut | ce qui a ete fait |
|---|---|---|
| golden inconditionnel sur la mini-bobine versionnee | `[x]` | 35 lignes figees, dont **33 familles** et 2 mesures derivees (compte corrige le 2026-09-06, correction C6 : ce journal ecrivait « 30 familles »). Il appelle les POINTS D'ENTREE (`ScanCamoStates`, `ScanAbilityCharges`, `ScanZoomEvents`, `ScanTranslocatorTeleports`, `ScanBipedPickups`, `ScanInventoryDeltas`, `ScanHeldWeaponChanges`, ...), JAMAIS les enveloppes `ScanFilm*(dir)`. Deux exceptions assumees, `weaponShots` et `weaponDamages` : leur point d'entree N'A PAS de forme `film`, il prend le repertoire — c'est donc bien le point d'entree, pas une enveloppe. |
| comptes + digest par famille | `[x]` | `nom \| compte \| digest \| premier`. |
| une valeur nommee lisible par famille | `[x]` | Le premier element, champs NOMMES, tronque a 140 caracteres. Exemple : `zoomEvents 37 ... {TimestampUS:4551203771 Slot:517 Level:1}`. Un rouge se lit dans le diff, sans outil. |
| `t.Fatal` si la bobine manque | `[x]` | Aucun `t.Skip` : la bobine est versionnee, son absence est une panne du depot. |
| assertions nommees i0/i4/i11/i43-46 + empreinte du registre | `[x]` | Dans `registry_test.go`, rassemblees en UN test (voir ci-dessous) plutot que dupliquees. |
| `registry_test.go` repointe, `t.Fatal` | `[x]` | Il pointait `c:/Users/Guillaume/.../data/cache/film_chunks/000d5950` — un chemin ABSOLU de la machine de l'auteur — et se `t.Skipf` ailleurs : il ne gardait rien en CI ni chez personne. Il lit maintenant la mini-bobine versionnee et `t.Fatal`. |

POURQUOI CETTE BOBINE-CI. La mini-bobine de `killsource` est un PREFIXE CONTIGU du film 000d5950
(chunks 00 a 05 + le chunk highlight), donc elle porte le REGISTRE et la continuite que le
decodeur exige pour construire son monde par accumulation. Celle du rejeu est faite de paquets
choisis : elle ne decode AUCUN record de canal delta. Mesure : cette bobine-ci rend 28 005 records
delta et 17 slots bipedes.

### Les 33 familles et les 2 mesures derivees, une par une

CORRECTION DU 2026-09-06 (revue E-R1, constat C6) — ce titre disait « les 30 familles » et
l'en-tete du fichier de test annoncait « une population non vide dans 25 familles sur 30 » et
« cinq familles rendent 0 ». Le fichier fige en realite **35 lignes** : 33 familles + 2 mesures
derivees, dont **29 peuplees**, **4 a zero** et **2 en erreur d'etat**. Le tableau ci-dessous
etait deja juste ligne par ligne — c'etaient les TOTAUX qui ne l'etaient pas.

| famille | population sur la mini-bobine |
|---|---|
| `abilityCharges` | 12 |
| `abilityImpulses` | 12 |
| `abilityRanks` | 15 |
| `bipedAim` | 398 |
| `camoStates` | 126 |
| `grappleReads` | 9 |
| `heldWeaponChanges` | 4 |
| `inventoryDeltas` | 187 |
| `unitEquipment` | 11 |
| `equipmentChanges` | 17 |
| `equipmentState` | 65 |
| `bipedPositions` | 28 004 |
| `projectiles` | 53 |
| `worldObjects_ti42` | 46 |
| `equipmentCreations` | 38 |
| `groundWeaponCreations` | 28 |
| `vehicleCreations` | **erreur d'etat** — « aucun slot d'archetype ti=40 dans les keyframes du film » |
| `equipmentPlacements` | 35 |
| `worldObjectKeyframes_ti35` | 17 (slots de bande) |
| `managedProperties` | 141 |
| `objectives_ti11` | **erreur d'etat** — « aucun slot d'archetype ti=11 dans les keyframes du film » |
| `navpointRadial` | **0 — population vide** |
| `fireEvents` | 45 |
| `grenadeThrows` | 10 |
| `bipedPickups` | 21 |
| `zoomEvents` | 37 |
| `translocatorTeleports` | **0 — population vide** |
| `vehicleEvents` | **0 — population vide** |
| `carrierMarks` | **0 — population vide** |
| `catalogueFamillesDuFilm` | 15 (mesure derivee, pas une famille) |
| `keyframeLoadouts` | 30 |
| `keyframeGroundWeapons` | 28 |
| `weaponShots` | 45 |
| `weaponDamages` | 46 |
| `weaponDamagesBaseSlot` | 512 (mesure derivee, pas une famille) |

CE QUE DISENT LES SIX LIGNES A ZERO OU EN ERREUR, ET C'EST UNE INFORMATION :

- `vehicleEvents` = 0 et `vehicleCreations` en erreur : **ce match n'a pas de vehicule**. Le film
  000d5950 est une partie Fiesta sur Cliffhanger — une arene sans vehicule. L'erreur est le refus
  PROPRE du balayage (« aucun slot ti=40 »), pas une panne.
- `objectives_ti11` en erreur : **pas d'objectif ti=11** — Fiesta est un mode a elimination.
- `translocatorTeleports` = 0 : **pas de translocateur** dans ce prefixe.
- `carrierMarks` = 0 : **pas de porteur** (ni drapeau, ni crane, ni bombe) — meme raison.
- `navpointRadial` = 0 : pas de point de navigation radial dans ce prefixe.

Si l'une de ces six se remplit un jour, le golden rougit — et c'est exactement ce qu'on veut
savoir.

### DEUX PIEGES TROUVES EN ECRIVANT CE GOLDEN, et corriges

1. **UN ZERO DE MAUVAIS APPEL N'EST PAS UNE POPULATION VIDE.** `ScanKeyframeLoadouts` et
   `ScanKeyframeGroundWeapons` filtrent sur un catalogue de familles d'arme ; appeles avec `nil`
   ils rendent 0 sans rien lire. Figer ce 0 aurait verrouille du vide. Le catalogue est donc
   DERIVE DU FILM lui-meme (identifiants d'arme des evenements de tir et des changements d'arme
   portee) : deterministe, sans fixture, 15 familles — et les deux balayages rendent alors 30 et
   28. De meme, `ScanFilmWeaponShots(dir, n)` balaie les chunks 1..n : avec `n=0` il rendait 0 ;
   avec `n=6`, 45 tirs.
2. **LE DIGEST NE PEUT PAS PASSER PAR `%+v`.** Plusieurs structures du decodeur portent des
   POINTEURS (`InventoryDelta.Ammo[].Mag`), dont `%+v` imprime l'ADRESSE : deux passes
   consecutives donnaient deux empreintes differentes pour `inventoryDeltas`. Un golden qui rougit
   au hasard ne verrouille rien et finit desactive. `rendreStable` descend par reflexion,
   dereference les pointeurs, trie les cles de carte et lit AUSSI les champs non exportes
   (`componentDirs`, `componentVitals` d'une position bipede), ou vit la moitie de ce qui
   distingue deux decodages. Stabilite verifiee sur TROIS passes independantes.

### Preuve par mutation

| mutation | familles rougies |
|---|---|
| `dom4RefWidth` 9 -> 10 (largeur du domaine 4 de la liste d'evenements) | **1** : `zoomEvents` — exactement la famille qui lit ce domaine |
| `bipedIndexBits` 6 -> 7 (largeur d'un index de composant) | **14** : `abilityCharges`, `abilityImpulses`, `abilityRanks`, `bipedAim`, `bipedPositions`, `camoStates`, `catalogueFamillesDuFilm`, `equipmentChanges`, `grappleReads`, `heldWeaponChanges`, `inventoryDeltas`, `keyframeGroundWeapons`, `keyframeLoadouts`, `unitEquipment` |

Les deux mutations ont ete annulees, et le golden repasse au vert.

## [x] E.7 — les 179 largeurs entieres de la table ECS, confrontees au code (F4)

Commit `25dbad2e6`. `filmdec/ecs_widths_guard_test.go` (controle G4).

Zero fixture, zero variable d'environnement, zero octet de film. Pour chacune des 179 lignes a
`bits_typ` ENTIER — toutes declarees « porte » —, le deser de PRODUCTION (`consumeByName`) tourne
sur trois tampons synthetiques (`0x00`, `0xFF`, `0xAA`) et on lit le nombre de bits consommes.

LES TROIS MOTIFS SONT LA CLE, parce que beaucoup de composants sont gardes :

| categorie | compte | regle |
|---|---|---|
| les trois motifs s'accordent -> largeur FIXE | **114** | elle DOIT egaler `bits_typ` (111 s'accordent, 3 sont des ecarts admis) |
| les trois divergent -> largeur GARDEE par le flux | **65** | l'entier de la table est NOMINAL ; ce controle ne peut pas le confronter |

Les deux comptes sont GELES : une ligne qui change de categorie est un signal, jamais un silence.

### Les trois ecarts, dates et justifies (`ecsEcartsAdmis`)

| ligne | table | mesure | pourquoi |
|---|---|---|---|
| ti=13 i=1 `managed-object-property-component` | 28 | 4 | largeur gardee par le TAG, **et la table le dit dans ses propres notes** (« largeur totale 4/5/8/28/36 selon le tag ») ; les trois motifs tombent sur des tags a charge nulle (t0, t10, t15), d'ou le R(4) du tag seul. `bits_typ` fige le cas t3. |
| ti=35 i=50 `biped-map-editor-flag-component` | 1 | 8 | **C'est la TABLE qui est perimee, pas le code.** `consumeBipedMapEditorFlag` lit un R(8) plat, « CONFIRMED bit-exact from the decompile » (FUN_142f02854). Corriger la colonne appartient a un lot qui revise la table ; la SIGNALER est le role de ce controle. |
| ti=37 i=14 `object-dissolver-component` | 4 | 113 | largeur gardee par la VALEUR lue, pas par un bit de porte : R(4) puis, si la valeur n'est pas 13, R(96)+R(12)+R(1). Les trois motifs manquent la valeur 13. La table fige le cas nominal. |

Le controle echoue aussi si un QUATRIEME ecart apparait, ou si l'un de ces trois se resorbe sans
qu'on retire sa ligne — une allowlist sans cible est du code mort.

**Preuve par mutation** : `tacmap-fasttravelstate` passe de `R(1)` a `R(3)` ; G4 designe
« ligne 678 (ti=34 i=6 tacmap-fasttravelstate) : la table annonce 1 bits, le deser de production
en consomme 3 (largeur FIXE : les trois motifs de tampon s'accordent) ». Retour, il passe.

DECOUVERTE (consignee, NON traitee) — `ti=43 i=0 object-position-component` porte `bits_typ = 15`
(« R(14) + R(1) ») alors que le code consomme 45 bits sur un tampon de zeros et 60 sur un tampon
de uns. La ligne tombe dans la categorie GARDEE, donc le controle ne la signale pas ; mais la
valeur de la table ne correspond a AUCUNE des trois mesures, ce qui est un cas different des trois
ecarts admis. A verifier par un lot qui revise la table.

## [x] E.8 — les 15 variables sans ecrivain (decouverte 2 de E-I)

Commit `0dd6e0fc8`.

| traitement | compte | detail |
|---|---|---|
| CONSTANTES NOMMEES, provenance ecrite | **10** | `absDequantMode`, `bipedActionLoop2Count`, `bipedDefaultStateDecodeMovement`, `bipedDefaultStateTailBits`, `bipedMediaFramePresent`, `deadStatePreSkip`, `deadStateVelocityPresent`, `inferRepair`, `inferRequireBoundSuccessor`, `vehicleMediaFrameBits` |
| SUPPRIMEES (aucune valeur mesuree) | **4** | l'instrumentation i63 : `biDebug`, `biCurSeq`, `BiBadSeqs`, `BiOkSeqs`, et ses quatre branches |
| SUPPRIMEE, MODELE GARDE | **1** | `absPerIndexAxisW` |
| GARDEES en `var`, deliberement | **2** | `accumWorld` (avec `accumSlot`) et `inferResyncTargets` |

Deux de ces constantes portent une LARGEUR MESUREE d'un chemin non nominal, et c'est la raison
d'etre du traitement : `vehicleMediaFrameBits` (bloc quaternion de la feuille 4 de ti=40, modelise
absent — bit-exact sur le chemin nominal, le cas d'un spawn) et `bipedDefaultStateTailBits` (queue
config-gardee de FUN_141f86704, 0 bit sur le corpus). Un savoir de retro-ingenierie ne se jette
pas : il se fige, et il se date.

`absPerIndexAxisW` etait une table nil, donc aucune valeur mesuree — mais son commentaire portait
le DESASSEMBLAGE des deux tables du moteur (`DAT_1445cc9e0` / `DAT_1445ccbe0`) et l'immediat
`LEVEL = 0x10` releve sur les trois sites d'appel. Ce modele est deplace sur `absAxisWFor`, la ou
un futur portage viendra le lire.

POURQUOI DEUX RESTENT. `accumWorld` et `inferResyncTargets` ne sont ni des largeurs ni des
valeurs : ce sont les INTERRUPTEURS de deux mecanismes entiers — l'accumulation de position par
World, la recuperation par resync valide. Les retirer supprimerait `setAccumSlot` et ses sites
d'appel dans les decodeurs, `validatedResync`, `scanForTargetDelta`, et l'unique installateur de
PRODUCTION de `posCaptureHook`. C'est une suppression de FONCTIONNALITE : elle se decide, elle ne
se glisse pas dans un lot a comportement identique. A trancher par le superviseur.

Ratchet `filmdecVarsGeles` resserre **111 -> 96**, chronique datee.

## Gate de cloture de la tache E-II

| commande | derniere ligne / verdict |
|---|---|
| `go build ./...` | exit 0 |
| `gofmt -l ./internal/ ./cmd/` | (aucune sortie) |
| `go test ./internal/analysis/filmdec/... ./internal/games/halo_infinite/film/... ./internal/analysis/replay/... ./internal/sync/killcollector/... ./internal/archlint/... -p 1 -parallel 1 -count=1` (AUCUNE variable d'environnement) | `ok levelup/go-api/internal/archlint 5.856s` — **les 10 paquets ok**, exit 0 |
| `KILLSOURCE_FIXTURES=<film_chunks> go test .../killsource/ -run TestGoldenFilms + TestReferenceFilms + TestLigneDiscriminante... -v` | **identique a la reference E.1**, echec anterieur `fccc61cd` compris |
| `DELTA_WITNESS_FILM=<film> go test .../filmdec/ -run TestDeltaWalkWitness + TestRegistryFingerprintOnFilm -v` (x3) | les trois **identiques** |
| `ECS_TABLE_FILM=<film> go test .../filmdec/ -run TestG1 + TestG2 + TestG3 + TestG4 -v` | les QUATRE controles PASS |
| `KILLSOURCE_FIXTURES=<film_chunks> go test -tags=integration .../killcollector/ -v` | `ok` — verdicts **identiques** |
| `golangci-lint run --timeout 5m ./internal/analysis/filmdec/...` (cache isole) | 6 issues — **memes linters, memes comptes que l'etat de base** |
| `golangci-lint run --timeout 5m --new-from-merge-base=origin/main ./...` (la commande EXACTE de la CI) | **`0 issues.`**, exit 0 |

AUCUN chiffre des temoins E.1 n'a bouge sur les trois items.

## Ratchets, apres la tache E-II

| ratchet | avant E-II | apres | sens |
|---|---|---|---|
| `archlint/filmdec_package_vars_test.go` | 111 | **96** | REDESCEND (E.8 : -15) |
| `filmdec/ecs_widths_guard_test.go` (G4) | (n'existait pas) | 114 fixes / 65 gardees, 3 ecarts admis | NOUVEAU (E.7) |
| `filmdec/golden_minibobine_test.go` | (n'existait pas) | 35 lignes figees : 33 familles + 2 mesures derivees | NOUVEAU (E.6) |
| `filmdec/registry_test.go` | chemin absolu + `t.Skipf` | mini-bobine versionnee + `t.Fatal` | DURCI (E.6) |

## Decouvertes de la tache E-II — consignees, NON traitees

1. **`ti=43 i=0 object-position-component`** : `bits_typ = 15` ne correspond a AUCUNE des trois
   mesures (45 / 60 / 60). La ligne est GARDEE, donc G4 ne la signale pas, mais l'ecart est d'une
   autre nature que les trois ecarts admis. A verifier par un lot qui revise la table.
2. **`biped-map-editor-flag-component`** : la table dit 1 bit, le code lit 8 et il est confirme
   bit-exact au decompile. C'est LA TABLE qui est a corriger — hors perimetre d'un lot a
   comportement identique.
3. **`accumWorld` / `accumSlot` et `inferResyncTargets`** restent sans ecrivain : ce sont les
   interrupteurs de deux mecanismes entiers, leur retrait est une decision produit.
4. **Les enveloppes `ScanFilm*(dir)`** restent hors du golden, comme le mandat le demande — sauf
   pour `weaponShots` et `weaponDamages`, qui n'ont PAS d'autre point d'entree. Leur forme `film`
   reste a ecrire ; c'est le lot 6 des enveloppes (dette datee, `archlint/no_film_reread_test.go`).

## Statut de la tache E-II

Les trois items sont `[x]`, faits et verifies sur pieces. Aucun `[~]`, aucun `[!]`.
Aucun test desactive, aucun skip ajoute, aucune allowlist agrandie sans justification datee.

---

# Corrections apres revue adversariale (E-R1) + item E.9 — 2026-09-06

Verdict source : `E-R1` (6 constats : 2 P1, 4 P2 ; 21 conditions qui tiennent ; 13 mutations).
Contrainte du mandat : **comportement du decodeur STRICTEMENT identique** — les temoins archives
de `LOT_E_digests_avant.md` restent le gate. Perimetre ferme :
`internal/analysis/filmdec/**`, `internal/archlint/decode_lock_held_test.go`,
`cmd/rdata_weapon_scan/main.go` (verrou seulement), ce journal, `.ai/thought_log.md`.

## [x] Correction 1 (C1, P1) — l'avance du marcheur delta, exercee pour de vrai

Commit `v2(E.fix-1)`. `delta_biped_walk_guard_test.go`, `offline_biped_test.go`.

LE TROU. `p = i0 + i0Bits` (`delta_biped_walk.go:83`) remplace par `p = i0 + 1` passait TOUT :
les 10 paquets du gate, le golden des familles, le temoin de marche delta. Le temoin synthetique
qui pretendait couvrir l'avance (`TestMarcheurDeltaBipedeAvanceCommeAvant`) marchait un payload
de 64 octets A ZERO : il ne publiait aucun record, donc l'instruction d'avance n'y etait JAMAIS
executee.

CE QUI EST FAIT.

1. `writeBipedHeaderEtMasque` extrait de `writeBipedRecord` (`offline_biped_test.go`) — l'ecrivain
   partage du paquet, pour que le nouveau temoin ne re-decrive pas la grammaire d'en-tete.
2. `TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie` : un payload de 21 octets porte DEUX
   records bipedes reconnus, colles l'un a l'autre, et le composant i0 du PREMIER porte un
   LEURRE — un en-tete de record valide sur un autre slot, plante au premier bit de l'axe X
   (`leurreDansLAxeDUnRecord`). Le marcheur doit publier exactement deux records, aux positions
   i0 = 39 et i0 = 123, tous deux sur le slot vrai ; le test verifie aussi qu'aucun couple publie
   ne se chevauche.
3. `TestMarcheurDeltaBipedeAvanceCommeAvant` renomme `TestMarcheurDeltaBipedeSArreteSurLaBorne`,
   et son en-tete dit desormais ce qu'il couvre reellement (la borne et la formule du seuil) et ce
   qu'il NE couvre pas (l'avance) — doc inversee corrigee a la source.
4. `TestMarcheurDeltaBipedeCompteSesRecordsSurLaMiniBobine` : le compte de records ANCRES par le
   marcheur, sur octets reels, pour les huit familles qui exposent ce denominateur plus le
   marcheur nu. 28 005 records pour les neuf entrees.

PREUVE PAR MUTATION (M6 du verdict, rejouee) — `p = i0 + i0Bits` -> `p = i0 + 1` :

```
--- FAIL: TestMarcheurDeltaBipedeNeRebalaiePasUnRecordPublie (0.00s)
    3 record(s) publie(s), attendu 2 :
      record 1 : i0=39  slot=517 masque=[0 1 2]
      record 2 : i0=77  slot=9   masque=[0 1]     <- le leurre, re-balaye en chevauchement
      record 3 : i0=123 slot=517 masque=[0 1 2]
```

Mutation annulee, le test repasse au vert.

MESURE QUI CONTREDIT LE MANDAT, ET QUI EST ECRITE PARCE QU'ELLE EST VRAIE — « un compte de
records par famille qui change si l'avance change » N'EXISTE PAS sur cette bobine. Les comptes ont
ete mesures SOUS la mutation avant d'etre figes : **28 005, identiques**, pour les neuf entrees.
Reprendre le balayage a l'interieur d'un record deja publie ne produit AUCUN ancrage de plus sur
la mini-bobine : la porte d'en-tete (prefixe, slot dans la bande de 17 slots sur 8 192, tag = 1,
deux bits nuls, masque strictement croissant depuis zero, gate d'i0 nul) est trop stricte pour
qu'un i0 de position la franchisse par hasard.

POURQUOI LE GOLDEN NE BOUGEAIT PAS, LA REPONSE COMPLETE. Deux raisons, et la seconde n'etait pas
soupconnee :

1. il n'y a AUCUN doublon a absorber — le jeu de records publie est le MEME avec et sans la
   mutation (mesure ci-dessus). Aucun dedoublonnage aval n'a donc rien masque, et rien n'est a
   corriger de ce cote ;
2. le golden des familles fige la LISTE de lectures de chaque famille et **jette les
   denominateurs** : `ajouterSlice(r, "camoStates", camo, err)` ignore le `st` du milieu, qui
   porte pourtant `Records`, le seul chiffre qui compte le travail du marcheur. Ce trou-la est
   ferme par le temoin de comptes, meme s'il ne suffit pas a attraper M6.

L'ORACLE DE L'AVANCE EST DONC LE TEMOIN SYNTHETIQUE, et le journal ne dit plus autre chose.

DECOUVERTE (consignee, NON traitee) — `ScanEquipmentState` figurait dans mon premier jet de la
liste des familles : il expose bien `Records`, mais il n'ancre PAS par le marcheur delta bipede
(`matchWorldObjectRecord`, `equipment_state.go:310`, sur la bande des slots d'equipement). Il a ete
retire : le compter ici aurait ete une doc inversee de plus. Son denominateur (5 282 sur la
bobine) reste non fige — il releve du marcheur d'objets du monde, pas de celui-ci.

## [x] Correction 2 (C2, P1) — le journal designait un oracle qui ne traverse pas E.4

Commit `v2(E.fix-2)`. `.ai/V7.5/v2/LOT_E.md` (gate E.4), `.ai/thought_log.md` (correctif date).

LE TROU. Le gate E.4 ecrivait « c'est l'item ou le TEMOIN DE MARCHE DELTA est l'oracle ».
`TestDeltaWalkWitness` (`delta_walk_witness_test.go:167`) appelle `DecodeFrameRecords`
(`frame_records.go`) : un marcheur DIFFERENT de `walkDeltaBiped*`. Mutation M7 du verdict —
`walkDeltaBipedPayload` neutralise (`return` en tete) — laisse ses trois mesures EXACTEMENT
identiques : `{paquets 14350, records 38883, aboutis 30089}`. La phrase du gate etait vraie et ne
prouvait rien.

CE QUI EST FAIT. Le paragraphe du gate E.4 porte desormais la correction datee, en clair : ce qu'il
disait, pourquoi c'etait faux, et quel est l'oracle REEL — le golden des familles (E.6, mutation M7
rougit 13 familles ; il n'existait pas quand E.4 a ete cloture, ce qui explique l'erreur) plus le
temoin synthetique de l'avance ajoute par la correction 1. Un correctif date est ajoute a l'entree
`[2026-09-06] Lot E-I` du thought_log, SANS reecrire l'entree : elle est datee, le correctif aussi.

CE QUI N'A PAS CHANGE, ET IL FAUT LE DIRE : l'item E.4 lui-meme reste bon. La revue a confirme sur
pieces que les neuf sites sont migres, que la borne, l'avance et le pas d'echec sont ceux d'origine
a la ligne pres, et que le golden des familles rougit quand le marcheur est neutralise. C'est la
DESIGNATION de la preuve qui etait fausse, pas le refacto.

## [x] Correction 3 (C3, P2) — le garde du preambule passe du grep a l'AST

Commit `v2(E.fix-3)`. `filmdec/event_preamble_guard_test.go`.

LE TROU. Le controle cherchait `\.Skip\(\s*[12]\s*\)` puis un `ReadBits(7)` dans les TROIS lignes
suivantes. Deux copies passaient :

- **M1** — la forme d'avant le lot, etalee sur 4-5 lignes. La prose du fichier affirmait que « les
  six copies d'origine tenaient toutes en trois lignes » : c'etait FAUX et mesurable — trois
  d'entre elles (`biped_pickups.go:212-216`, `transloc_events.go:168-172`, `zoom_events.go:136-140`
  a la base `a21fd77f4`) s'etalaient sur quatre ou cinq lignes.
- **M2** — la copie la plus probable de toutes, le copier-coller du corps du lecteur unique
  (`br.ReadBit(); br.ReadBit(); br.ReadBits(7)`), qui n'emploie ni `Skip(1)` ni `Skip(2)`.

CE QUI EST FAIT. Le controle ne regarde plus des lignes, il COMPTE DES BITS. Pour chaque lecteur,
dans chaque SUITE DE STATEMENTS, les operations de bits (`Skip(n)`, `ReadBit()`, `ReadBits(n)`)
sont ordonnees par position de source, puis on cherche « exactement deux bits consommes, puis une
lecture de sept » sur des operations CONSECUTIVES. Les trois conventions y tombent —
`Skip(2)+R(7)`, `Skip(1)+ReadBit()+R(7)`, `ReadBit()+ReadBit()+R(7)` — quel que soit le nombre de
lignes, une lecture placee dans la CONDITION d'un `if` appartenant a la suite qui porte ce `if`.
L'exemption est desormais la FONCTION `readPacketHead`, plus le fichier `event_list.go` entier.

POURQUOI « MEME SUITE DE STATEMENTS » ET PAS « MEME FONCTION » — MESURE. Sans cette borne, le
controle rend DEUX faux positifs, et ils ne sont pas des copies du preambule mais des grammaires de
composant que le flot de controle separe :

| site | ce que le controle voyait | ce que c'est vraiment |
|---|---|---|
| `components_object_state.go:253-255 consumeObjectLowFrequency` | `R(2)` puis `R(7)` | FUN_1407ef088 : le `R(7)` est DANS le `if f < 2`, il n'est pas consecutif |
| `traverse.go:577-585 consumeByName` | `ReadBit()` puis `R(7)` | DEUX branches exclusives du meme `switch` (`nav-cutscene-flag` et `player-desired-respawn-seat`) |

Un garde-rail qui exige une allowlist des le premier jour ne tient pas : la bonne borne est le
flot, pas une liste. Aucune allowlist n'a donc ete creee.

PREUVES PAR MUTATION (les deux, rejouees le 2026-09-06) :

```
M1  decodeZoomHead remis a sa forme d'avant le lot (Skip(1) + ReadBit + R(7), 5 lignes)
    --- FAIL: TestPreambuleNaQuUnSeulLecteur
        zoom_events.go:136-140 (decodeZoomHead) recopie le preambule d'evenement sur `br`

M2  septieme copie en convention CANONIQUE ajoutee a zoom_events.go
    --- FAIL: TestPreambuleNaQuUnSeulLecteur
        zoom_events.go:197-199 (copieDuPreambuleM2) recopie le preambule d'evenement sur `br`
```

Les deux mutations sont annulees (`git checkout -- zoom_events.go`), le controle repasse au vert.

CE QU'IL NE VOIT TOUJOURS PAS, ECRIT DANS L'EN-TETE DU FICHIER : (1) une copie dont les neuf bits
seraient repartis entre deux suites imbriquees — c'est le prix de la borne ci-dessus, et aucune des
six copies d'origine n'avait cette forme ; (2) une copie par arithmetique d'offset
(`readBitsAt(pay, p+2, 7)`), qui demanderait de suivre la valeur de `p`.

## [x] Correction 4 (C4, P2) — la liste des paquets qui decodent est DERIVEE de la regle

Commit `v2(E.fix-4)`. `archlint/decode_lock_held_test.go`, `cmd/rdata_weapon_scan/main.go`.

LE TROU. `paquetsQuiDecodent` etait une liste FERMEE de trois paquets, maintenue par une commande
documentee qui cherchait `filmdec.Scan` — alors que la regle CODEE (`balayageFilmdec`) couvre
`Scan*` **et** `DecodeFrame*` **et** `TraverseEntity*`. Un quatrieme paquet passait dans l'ecart
entre la liste et la commande censee l'entretenir : `cmd/rdata_weapon_scan/main.go:258` appelle
`filmdec.DecodeFrameRecords` et ne contient AUCUN `LockProcessDecode`. C'est un binaire que
`go build ./...` compile.

CE QUI EST FAIT.

1. La liste est **derivee** : `paquetsQuiDecodent(t, racine)` balaie `internal` et `cmd`, retient
   toute source non-test qui contient `filmdec.` ET dont l'AST porte un appel reconnu par
   `balayageFilmdec`, hors `internal/analysis/filmdec`. La regle et sa liste ne peuvent plus
   diverger : c'est la MEME fonction qui decide des deux.
2. Un PLANCHER date (`paquetsAttendus`, 4 entrees) fait echouer le test si la derivation cesse de
   trouver un paquet connu — une derivation cassee (mauvaise racine, marcheur renomme) ne peut pas
   rendre une liste vide en annoncant « vert ». Ce n'est pas un plafond : un paquet neuf entre
   tout seul.
3. `cmd/rdata_weapon_scan` mis en conformite : `release := filmdec.LockProcessDecode()` +
   `defer release()` en TETE de `main`, avec la justification ecrite au-dessus. C'est la forme qui
   satisfait le contrat « pour toute la duree du decodage, jamais par sous-appel » sur un binaire
   mono-tache ; `litLoc` est alors couvert par le point fixe (son seul appelant du paquet est
   `main`).
4. L'en-tete du fichier ne documente plus une commande de re-mesure : il explique que la liste est
   derivee, et pourquoi.

PREUVE, ROUGE PUIS VERT (2026-09-06) :

```
AVANT le verrou, liste derivee :
--- FAIL: TestBalayagesFilmdecSousVerrou/cmd/rdata_weapon_scan
      cmd/rdata_weapon_scan/main.go:214 litLoc
    (les trois autres paquets : PASS)

APRES le verrou :
--- PASS: TestBalayagesFilmdecSousVerrou (4 sous-tests PASS)
go build ./...  -> exit 0
go vet ./internal/archlint/ ./cmd/rdata_weapon_scan/  -> exit 0
```

CE QUE LA DERIVATION MESURE EN PLUS, ET QUI N'A RIEN CHANGE : le balayage couvre desormais les
fichiers a contrainte de compilation (`parser.ParseFile` ignore les tags). Aucun paquet
supplementaire n'en ressort sur l'arbre du jour — les quatre trouves sont exactement les quatre
que la commande de re-mesure corrigee rend.

## [x] Correction 5 (C5, P2) — deux sorties figees, deux portes de regeneration

Commit `v2(E.fix-5)`. `filmdec/fuzz_records_test.go`, `filmdec/golden_minibobine_test.go`.

LE TROU. Un seul drapeau `-update` (`updateGoldens`) pilotait DEUX sorties figees : le corpus de
graines du fuzz et le golden des familles. La commande qu'on tape pour regenerer les graines,
`go test ./internal/analysis/filmdec/ -update` SANS `-run`, reecrivait donc le golden au passage
avec ce que le decodeur rendait a cet instant — et le paquet repondait `ok`. Le fichier ecrit
pourtant lui-meme « Un golden ne s'edite JAMAIS a la main. Il ne se regenere qu'apres un changement
de decodage DECLARE ».

CE QUI EST FAIT. Deux portes NOMMEES, chacune ne regenerant que sa sortie :

| sortie figee | porte | ou |
|---|---|---|
| `testdata/fuzz/FuzzFilmRecordReaders/` | `-update` (`updateGrainesFuzz`) | `fuzz_records_test.go` |
| `testdata/golden_minibobine_familles.tsv` | `-update-golden-familles` (`updateGoldenFamilles`) | `golden_minibobine_test.go` |

La section « REGENERATION » de l'en-tete du golden porte la commande exacte et la note datee qui
explique le defaut d'avant ; la declaration du drapeau du fuzz porte la sienne.

PREUVE (M11 du verdict, rejouee le 2026-09-06). `br.Skip(2)` -> `br.Skip(3)` dans `readZoomRef`
(`zoom_events.go:163`, largeur de generation qu'aucun autre test n'epingle), puis la commande
exacte du constat :

```
go test ./internal/analysis/filmdec/ -count=1 -p 1 -parallel 1 -update
  AVANT : ok levelup/go-api/internal/analysis/filmdec 22.527s  + golden REECRIT
  APRES : FAIL levelup/go-api/internal/analysis/filmdec 3.784s + golden INTACT

go test ./internal/analysis/filmdec/ -run TestGoldenMiniBobineFamilles -update
  --- FAIL: TestGoldenMiniBobineFamilles
      famille 26 a change :
        fige  : zoomEvents 37 d2686333... {TimestampUS:4551203771 Slot:517 Level:1}
        obtenu: zoomEvents 37 2a1eb1fc... {TimestampUS:4551203771 Slot:517 Level:2}

md5 du golden, avant et apres les deux commandes : e396143919281fde5f92a56d0af03d86 (INCHANGE)
```

LES DEUX PORTES MARCHENT ENCORE, verifie apres retour de la mutation : `-update-golden-familles`
reecrit le golden a l'octet pres (meme md5, 35 familles) et `-update` reecrit les 5 graines sans
rien changer sur disque. Une porte qu'on ferme sans la reverifier est une porte cassee.

## [x] Correction 6 (C6, P2) — l'en-tete du golden decrivait un autre fichier

Commit `v2(E.fix-6)`. `filmdec/golden_minibobine_test.go`, section E.6 de ce journal.

LE TROU — TROIS AFFIRMATIONS FAUSSES, toutes verifiables en comptant :

| l'en-tete disait | le fichier fige / le code dit |
|---|---|
| « une population non vide dans 25 familles sur 30 » | 35 lignes : 33 familles + 2 mesures derivees, dont 29 peuplees |
| « Cinq familles rendent 0 sur ce prefixe » | QUATRE : `navpointRadial`, `translocatorTeleports`, `vehicleEvents`, `carrierMarks` |
| « `digest` est le sha256 d'un rendu `%+v` de TOUT le resultat » | c'est `rendreStable`, qui existe PRECISEMENT parce que `%+v` imprimait des ADRESSES |

La troisieme est la plus dangereuse : un mainteneur qui « simplifie » `rendreStable` en `%+v` sur
la foi de cette ligne reintroduit l'instabilite que le fichier venait d'eliminer — la
demonstration est ecrite 300 lignes plus bas dans le meme fichier.

CE QUI EST FAIT. L'en-tete est reecrit d'apres le TSV et d'apres `rendreStable` : les comptes
exacts (35 lignes, 33 familles + 2 mesures derivees, 29 peuplees, 4 a zero, 2 en erreur, chacune
nommee), le vrai rendu du digest avec la raison de ne pas revenir a `%+v`, et une note datee qui
dit ce que l'en-tete affirmait avant. S'y ajoute ce que la revue a mesure et que l'en-tete taisait :
**une population vide ne verrouille rien** — les deux lignes a zero d'une famille qui rend une
tranche portent le MEME digest de tranche vide (`4f53cda18c2baa0c0354...`, verifie sur le TSV) et
les deux lignes en erreur ne figent que la chaine du refus. La couverture reelle est de 29 lignes
sur 35, et c'est ecrit noir sur blanc.

LES COMPTES DE CE JOURNAL SONT CORRIGES AVEC : la section E.6 disait « 30 familles » aux trois
endroits ou elle les comptait (titre d'item, tableau, ligne de ratchet), alors que son propre
tableau ligne par ligne en listait 35. Les lignes du tableau etaient justes ; les totaux ne
l'etaient pas.

VERIFICATION : `TestGoldenMiniBobineFamilles` PASS, md5 du golden inchange
(`e396143919281fde5f92a56d0af03d86`) — cette correction ne touche que de la prose.

## [x] Correction 7 = item E.9 (decision utilisateur 10) — la table des largeurs, entree par entree

Commit `v2(E.9)`. `filmdec/testdata/ecs_table.tsv` (3 lignes), `filmdec/ecs_widths_guard_test.go`.

CE QUI REND CETTE CORRECTION SANS RISQUE, VERIFIE AVANT DE TOUCHER UNE SEULE LIGNE : **la table
n'a AUCUN lecteur applicatif.** Son unique lecteur est `loadECSTable` dans
`ecs_table_guard_test.go`, et l'en-tete de ce fichier l'ecrit deja (« un `ecs_table.go` serait du
code mort »). Aucune valeur de cette table n'entre dans un chemin de decodage : la corriger ne peut
pas changer un bit servi. **Zero ligne de code du decodeur n'a ete touchee.**

### La mesure d'abord : un controle G5 qui TIENT les largeurs citees

Une note dans une colonne de prose n'est pas une mesure — c'est une affirmation, et ce lot vient de
montrer trois fois ce que devient une affirmation que personne ne rejoue. Avant de corriger quoi
que ce soit, `TestG5MesuresCiteesParLaTable` (`ecs_widths_guard_test.go`) fige la largeur consommee
par le deser de production sur CHACUN des trois motifs, separement — la ou G4 ne retient que
l'accord ou le desaccord des trois :

| ligne | 0x00 | 0xFF | 0xAA | ce que ca dit |
|---|---|---|---|---|
| `ti=13 i=1` managed-object-property | 4 | 4 | 4 | les trois motifs tombent sur des tags a charge nulle |
| `ti=35 i=50` biped-map-editor-flag | 8 | 8 | 8 | R(8) PLAT, aucune porte |
| `ti=37 i=14` object-dissolver | 113 | 113 | 113 | les trois manquent la valeur 13 |
| `ti=43 i=0` object-position | **45** | **60** | **60** | garde par le bit precHigh de tete |

Les 45/60/60 sont donc RE-MESURES ici par le depot, pas repris de la sonde jetable de la revue.
Chaque note de la table cite ce controle par son nom : la note et la mesure se corrigent ensemble.

### Les trois entrees, une par une

**`ti=35 i=50 biped-map-editor-flag-component` — `bits_typ` 1 -> 8, CORRIGEE.**
`consumeBipedMapEditorFlag` (`components_biped_ability.go:145-147`) est un `br.ReadBits(8)` plat,
sans porte ni largeur de runtime, annote « CONFIRMED bit-exact from the decompile » (FUN_142f02854,
refill 8 bits unique). C'etait la TABLE qui etait perimee. La ligne **quitte l'allowlist**
`ecsEcartsAdmis` : elle est desormais tenue par G4 comme n'importe quelle largeur fixe.

**`ti=43 i=0 object-position-component` — NOTE, PAS DE NOMBRE.** `bits_typ = 15` (« R(14) + R(1) »)
ne correspond a AUCUN chemin du code. Aucun entier ne serait juste : la mesure donne 45 ou 60 selon
le motif, et le chemin dominant (45 = 3 de porte + AxisW + 2 de queue) est **PROPRE A LA CARTE** —
`AxisW` vaut 13/13/14 sur Cliffhanger et autre chose ailleurs (`WorldObjectPrecision`,
`traverse.go`). Le nombre est donc laisse tel quel et la colonne `notes` porte la mesure, sa
provenance et la raison pour laquelle aucun entier ne peut la remplacer — comme le fait deja
`ti=13 i=1`. La ligne est de categorie GARDEE : elle n'etait pas dans l'allowlist, elle n'y entre
pas.

**`ti=37 i=14 object-dissolver-component` — NOTE, PAS DE NOMBRE.** `consumeObjectDissolver` lit
R(4) puis, SI la valeur n'est pas 13, R(96)+R(12)+R(1) : 4 bits ou 113, garde par la VALEUR lue et
non par un bit de porte. `bits_typ = 4` fige le cas v==13 et reste tel quel ; la colonne `notes`
porte l'explication et la mesure. La ligne **reste dans l'allowlist**, comme le mandat le demande
pour une entree dont aucune valeur unique n'est etablie.

### ECART ASSUME AVEC LE MANDAT, ECRIT PARCE QU'IL EST VOLONTAIRE

Le mandat proposait la note « 45 ou 60 selon le motif (mesure 2026-09-06, **trois films**) ». La
mesure n'a PAS ete faite sur trois films : elle est faite sur les **trois motifs de tampon**
synthetiques du controle G4/G5 (`0x00`, `0xFF`, `0xAA`), sans un octet de film. Ecrire « trois
films » aurait invente une provenance — exactement la classe de defaut que les corrections 2 et 6
de cette meme session viennent de reparer. La note dit ce qui a ete mesure, et par quel controle.

### Ratchet

| ratchet | avant | apres | sens |
|---|---|---|---|
| `ecsEcartsAdmis` | 3 ecarts admis | **2** | REDESCEND (la ligne corrigee sort de l'allowlist) |
| `ecsLargeursFixes` / `ecsLargeursGardees` | 114 / 65 | 114 / 65 | inchange (aucune ligne ne change de categorie) |
| `filmdec/ecs_widths_guard_test.go` (G5) | (n'existait pas) | 4 lignes, 12 largeurs figees | NOUVEAU (E.9) |

### Preuves par mutation (les deux, rejouees le 2026-09-06)

```
1) la TABLE re-perimee : bits_typ remis a 1 sur ti=35 i=50, allowlist videe de cette ligne
   --- FAIL: TestG4LargeursEntieresSuiventLeCode
       G4 : ligne 739 (ti=35 i=50 biped-map-editor-flag-component) : la table annonce 1 bits,
       le deser de production en consomme 8 (largeur FIXE : les trois motifs s'accordent)

2) le CODE mute : consumeBipedMapEditorFlag R(8) -> R(9)
   --- FAIL: TestG4LargeursEntieresSuiventLeCode
       G4 : ... la table annonce 8 bits, le deser de production en consomme 9
   --- FAIL: TestG5MesuresCiteesParLaTable
       G5 : ti=35 i=50 ... consomme [9 9 9] bits sur les motifs [0 255 170], fige a [8 8 8]
```

Les deux mutations sont annulees ; G1, G3, G4, G5 passent, le paquet `filmdec` est vert.

## Gate de cloture des corrections + E.9 — rejoue INTEGRALEMENT le 2026-09-06

Meme worktree, meme branche, en SERIE, en avant-plan, `-p 1 -parallel 1 -count=1`, prefixe
`GOCACHE=/c/Users/Guillaume/AppData/Local/go-build-v2-decodeur-2`
`GOLANGCI_LINT_CACHE=/c/Users/Guillaume/AppData/Local/golangci-v2-decodeur CGO_ENABLED=1`.
Films lus en LECTURE SEULE dans le checkout principal, UN film par processus. Aucune cuisson
d'artefact, aucun tag `gamefiles`.

| commande | derniere ligne / verdict | contre la reference E.1 |
|---|---|---|
| `go build ./...` | exit 0 | identique |
| `go vet ./...` | exit 0 | identique |
| `gofmt -l ./internal/analysis/filmdec/ ./internal/archlint/ ./cmd/rdata_weapon_scan/` | (vide) | identique |
| les 10 paquets du gate inconditionnel | `ok levelup/go-api/internal/archlint 6.691s` — **10 ok**, exit 0 | identique |
| `TestEquivalenceMiniFilm` (7 digests de la mini-bobine) | `--- PASS (1.16s)` | identique |
| goldens killsource, 4 films reels | `FAIL ... 110.302s` — 000d5950 / 9b191a7f / 78919882 PASS, **`fccc61cd` FAIL sur LA MEME ligne** (`3 propose(s)` fige contre `2 propose(s)` mesure, 74 lignes figees / 74 obtenues) | **identique, echec anterieur compris** |
| `TestReferenceFilms` x4 | les quatre PASS | identique |
| `TestLigneDiscriminanteEstServieParLaMarche` | `--- PASS (17.25s)` | identique |
| temoin de marche delta `000d5950` | `{paquets 14350 records 38883 aboutis 30089}` (77.383 %), FAIL contre le fige `{38878, 30080}` | **identique, echec anterieur compris** |
| empreinte de registre `000d5950` | 50 blocs / 1067 slots / 49 porteurs · `0x61e492dd4de7fd4e` · concordance **true** · PASS | identique |
| table ECS G1 / G2 / G3 / G4 / **G5** sur `000d5950` | les CINQ PASS. G2 : 50 blocs, 49 porteurs, 1067 = 1067 (+14 alias). G3 : 27 references / 1479 champs | identique (+ G5, nouveau) |
| golden des familles, **2 passes independantes** | PASS x2, sortie identique, md5 du TSV `e396143919281fde5f92a56d0af03d86` inchange | identique |
| nouveaux temoins (marcheur x4, preambule) | PASS x2 passes | nouveau |
| integration `killcollector` (`-tags=integration`) | `ok ... 32.242s` — **67 PASS** | identique |
| `golangci-lint run --timeout 10m --new-from-merge-base=origin/main ./...` (la commande EXACTE de la CI) | **`0 issues.`**, exit 0 | vert |

AUCUN CHIFFRE DES TEMOINS N'A BOUGE. Les deux echecs ANTERIEURS au lot (golden `fccc61cd`, temoin
de marche delta) sont retrouves a l'identique : ces corrections ne les reparent pas — c'est le lot A
qui traite leur cause (P0-1) — et ne les masquent pas.

## Ratchets, apres les corrections

| ratchet | avant | apres | sens |
|---|---|---|---|
| `archlint/filmdec_package_vars_test.go` | 96 | 96 | inchange (fichier intact) |
| `archlint/no_local_longest_run_test.go` | 1 exemption | 1 | inchange (fichier intact) |
| `archlint/decode_lock_held_test.go` | liste FERMEE de 3 paquets | liste **DERIVEE** + plancher date de 4 | ELARGIT la couverture |
| `filmdec/event_preamble_guard_test.go` | exemption = le fichier `event_list.go` | exemption = la fonction `readPacketHead` | RESSERRE |
| `filmdec/ecs_widths_guard_test.go` — `ecsEcartsAdmis` | 3 ecarts admis | **2** | REDESCEND |
| `filmdec/ecs_widths_guard_test.go` — `ecsLargeursFixes` / `Gardees` | 114 / 65 | 114 / 65 | inchange |
| `filmdec/ecs_widths_guard_test.go` — G5 | (n'existait pas) | 4 lignes, 12 largeurs figees | NOUVEAU |
| `filmdec/delta_biped_walk_guard_test.go` | 2 controles | **4** (dont l'avance et les comptes reels) | ELARGIT |

Aucun ratchet n'a monte. Aucune allowlist n'a grandi ; deux ont retreci.

## Statut — les sept points

| # | constat | statut | preuve |
|---|---|---|---|
| 1 | C1 P1 — l'avance du marcheur n'etait couverte par rien | `[x]` | M6 rouge : 3 records au lieu de 2, le leurre publie a i0=77 |
| 2 | C2 P1 — le journal designait un faux oracle pour E.4 | `[x]` | correction datee au gate E.4 + correctif date au thought_log |
| 3 | C3 P2 — le garde du preambule ne voyait que la moitie des copies | `[x]` | M1 et M2 rouges, sites exacts designes |
| 4 | C4 P2 — liste fermee + un paquet qui decode sans verrou | `[x]` | ratchet rouge sur `cmd/rdata_weapon_scan/main.go:214 litLoc`, vert apres le verrou |
| 5 | C5 P2 — `-update` regenerait le golden des familles | `[x]` | M11 : `ok` -> `FAIL`, md5 du golden inchange |
| 6 | C6 P2 — l'en-tete du golden decrivait un autre fichier | `[x]` | comptes refaits sur le TSV, `rendreStable` cite a la place de `%+v` |
| 7 | E.9 — table des largeurs, decision utilisateur 10 | `[x]` | G5 mesure 8/8/8 et 45/60/60 ; table re-perimee -> G4 rouge ; code mute -> G4 + G5 rouges |

Aucun `[~]`, aucun `[!]`. Aucun test desactive, aucun skip ajoute, aucun golden regenere, aucune
allowlist agrandie. Zero ligne de code du DECODEUR touchee : le diff est fait de tests, de gardes,
de trois lignes de la table ECS (sans lecteur applicatif), d'un verrou dans un binaire d'outillage
et de journal.

## Decouvertes de cette session — consignees, NON traitees

1. **Le golden des familles jette les denominateurs.** `ajouterSlice(r, "camoStates", camo, err)`
   ignore le `st` du milieu de `ScanCamoStates`, qui porte pourtant `Records` — le seul chiffre qui
   compte le travail du marcheur. Ferme cote marcheur delta bipede par le temoin de comptes
   (correction 1) ; les autres denominateurs de chaque famille (`WithI28`, `Read`, `Unread`,
   `NoChannel`...) restent hors golden.
2. **`ScanEquipmentState` n'ancre PAS par le marcheur delta bipede** mais par
   `matchWorldObjectRecord` (`equipment_state.go:310`), sur la bande des slots d'equipement. Son
   denominateur (5 282 records sur la mini-bobine) n'est fige nulle part — il releve du marcheur
   d'objets du monde, qui n'a pas eu son item E.4.
3. **Deux formes de copie du preambule restent invisibles au garde AST** : les neuf bits repartis
   entre deux suites de statements imbriquees, et la copie par arithmetique d'offset
   (`readBitsAt(pay, p+2, 7)`). Les deux sont ecrites dans l'en-tete du fichier ; aucune des six
   copies d'origine n'avait ces formes.
4. **`bits_typ` melange deux semantiques** — une largeur FIXE et une largeur NOMINALE — et c'est ce
   melange qui oblige `ecsEcartsAdmis` a exister. Les distinguer (par exemple en prefixant les
   nominales) est une reecriture de la table, hors du perimetre de l'item E.9 qui corrige entree
   par entree sur mesure.

## Retouches apres ronde 2 (E-R2) — 2026-09-06

Verdict E-R2 : les six constats de E-R1 sont **FERMES** et l'item E.9 juge **prudent** (12 mutations
rejouees, tous les temoins retrouves au chiffre pres). Trois residus P3, traites ici en un commit.
Aucune ligne de code du decodeur touchee : le diff est fait de trois fichiers de garde-rail.

### [x] D-3 — un ecart admis resorbe par la TABLE n'etait pas detecte

`filmdec/ecs_widths_guard_test.go`. L'en-tete d'`ecsEcartsAdmis` promettait un echec « si l'un de
ceux-ci se resorbe sans qu'on retire sa ligne ». C'etait vrai dans UN sens seulement : la branche
`admis` ne comparait que `largeur != e.Mesure` (resorption par le CODE). Corriger `bits_typ` a la
valeur mesuree en gardant l'entree — **exactement la manoeuvre que E.9 vient d'executer sur
`ti=35 i=50`** — laissait G1/G3/G4/G5 tous verts, l'entree morte et son champ `Table` mensonger.

Le controle `e.Table != r.BitsTyp` est ajoute, avec son message : retirer la ligne si la table a ete
corrigee, mettre a jour `Table` avec une raison datee sinon. La prose dit desormais que **les deux
sens** sont controles.

PREUVE — `ecs_table.tsv` ligne 331 (`ti=13 i=1`), `bits_typ` 28 -> 4 (la mesure), entree conservee :

```
AVANT : G1, G3, G4, G5 -> les quatre PASS
APRES : --- FAIL: TestG4LargeursEntieresSuiventLeCode
        G4 : ligne 331 (ti=13 i=1 managed-object-property-component) est un ecart ADMIS declare
        contre une table a 28 bits, mais la table annonce maintenant 4.
```

Table restauree, `git diff` vide.

### [x] D-2 — un import ALIASE de `filmdec` echappait entierement a la liste derivee

`archlint/decode_lock_held_test.go`. `balayageFilmdec` comparait le qualificateur au litteral
« filmdec ». Un paquet ecrivant `fd "levelup/go-api/internal/analysis/filmdec"` puis
`fd.DecodeFrameRecords(...)` sans verrou laissait le ratchet VERT **et n'entrait pas dans la liste
derivee** — la derivation heritait du trou de la regle, et depuis qu'elle en est la SEULE source
(correction 4 de la ronde 1), plus rien ne le rattrapait a la main.

Le qualificateur est desormais RESOLU par l'import, jamais devine. `nomLocalDeFilmdec(f)` lit le
nom local du chemin `levelup/go-api/internal/analysis/filmdec` dans chaque fichier et couvre les
quatre formes : nom par defaut, alias, import point (appels NUS — traites), import muet (`_`, aucun
appel possible). La prise du verrou (`LockProcessDecode`) suit la meme resolution. Le prefiltre
textuel de la derivation cherche maintenant le CHEMIN d'import et non la chaine « filmdec. », qui
n'apparait nulle part dans un fichier a import aliase.

PREUVE — paquet jetable `cmd/sonde_alias_er2` appelant `DecodeFrameRecords` sans verrou :

```
import fd "…/filmdec"   AVANT : VERT, paquet absent de la liste
                        APRES : --- FAIL: .../cmd/sonde_alias_er2
                                cmd/sonde_alias_er2/main.go:5 main
import . "…/filmdec"    APRES : --- FAIL, meme site  (appels nus, forme couverte aussi)
```

Sonde supprimee, arbre propre. Les quatre paquets reels restent PASS.

### [x] D-1 — la liste des angles morts du garde AST se donnait pour une enumeration

`filmdec/event_preamble_guard_test.go`. La prose listait deux formes d'evasion ; la revue en a
exhibe deux autres. Traitement en deux temps, parce que les deux formes ne se valent pas :

- **`Skip(0)` intercale — FERME POUR DE BON.** `br.Skip(1); br.ReadBit(); br.Skip(0);
  br.ReadBits(7)` echappait parce qu'une operation de largeur nulle rompait la consecutivite. Elle
  ne consomme AUCUN bit : `opDeLAppel` ne la retient plus du tout. Aucun faux positif introduit
  (le paquet reste vert).
- **`ReadBits(9)` puis masque — ANGLE MORT ASSUME.** L'ancre du motif est la lecture de SEPT bits.
  Fermer cette forme demanderait de traiter toute lecture de 9 bits comme suspecte, alors que la
  largeur 9 n'a rien de propre au preambule : le controle deviendrait bruyant sur des grammaires de
  composant. C'est un arbitrage, il est ecrit comme tel.

La prose ne se donne plus pour complete : elle s'annonce « LISTE OUVERTE, TENUE A JOUR, JAMAIS
PRESENTEE COMME COMPLETE », enumere les **quatre** formes connues et marque chacune FERME ou NON
FERME, avec sa raison.

PREUVE — les deux sondes de la revue ajoutees a `zoom_events.go` :

```
copieM3ZeroLargeur  AVANT : VERT   APRES : FAIL — zoom_events.go:197-200 recopie le preambule
copieM3NeufBits     AVANT : VERT   APRES : VERT — angle mort n°4, assume et documente
```

Sondes retirees, `git diff` vide sur `zoom_events.go`.

### Gate des retouches

| commande | derniere ligne |
|---|---|
| `go test -count=1 ./internal/analysis/filmdec/... ./internal/archlint/... -p 1 -parallel 1` | `ok …/internal/archlint 18.912s` — les deux paquets **ok** |
| `go build ./...` | exit 0 |
| `gofmt -l ./internal/analysis/filmdec/ ./internal/archlint/` | (vide) |
| `golangci-lint run --timeout 10m --new-from-merge-base=origin/main ./...` | **`0 issues.`**, exit 0 |
| md5 du golden des familles | `e396143919281fde5f92a56d0af03d86` — **inchange** |
| `git diff --stat -- internal/analysis/filmdec/testdata/` | (vide) — aucune sortie figee touchee |

Aucun ratchet n'a monte. `ecsEcartsAdmis` reste a 2, `ecsLargeursFixes`/`Gardees` a 114/65, le
plancher `paquetsAttendus` a 4 : les trois retouches RESSERRENT les regles sans deplacer un seuil.
