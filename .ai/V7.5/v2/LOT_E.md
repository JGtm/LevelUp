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
