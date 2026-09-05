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
  `go build ./...` exit 0 ; `golangci-lint run ./internal/analysis/filmdec/...` -> `0 issues.` ;
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
