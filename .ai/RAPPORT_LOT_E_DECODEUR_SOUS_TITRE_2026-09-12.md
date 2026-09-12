# Rapport — Lot E : le decodeur de film descend sous son titre (2026-09-12)

> Plan : `.ai/PLAN_FORK_ET_RELEASE_2026-09-11.md`, lot E (tache Notion 9, decision 5 de
> l'audit v2). Worktree `LevelUp-wt-decodeur-sous-titre`, branche `wt/decodeur-sous-titre`,
> base feat/v75 `2f5d165be`. Contrat : skill `plan-execution`. ADR 0012 (adapters Halo-only)
> et ADR 0025 (refactor title-agnostic).

## E.0 — Inventaire, avant tout code

### Ce qui descend

| Paquet | Fichiers `.go` | Lignes (dont hors test) |
|---|---|---|
| `internal/analysis/filmdec` | 439 | 115 101 (25 706) |
| `internal/analysis/replay` (+ `replay/mapvar`) | 499 | 130 827 (36 010) |
| Total | 938 | 245 928 (61 716) |

Trois paquets Go, pas deux : `filmdec`, `replay`, `replay/mapvar`. Aucun paquet de test
externe (`_test`) : tous les tests sont internes au paquet.

### Qui les importe, par couche

Arcs de dependance mesures par `go list -f '{{.Imports}}{{.TestImports}}{{.XTestImports}}'`
sur tout le module (85 arcs, 60 paquets distincts).

| Couche | Paquets importateurs |
|---|---|
| `internal/analysis/*` | `analysis` (test externe), `analysis/filmsource` (test externe), `analysis/sessionusage` (**production**) |
| `internal/api` | `api/handlers` (prod + test externe), `api/wire` (tests) |
| `internal/service` | `service` (prod + test), `service/replayview` (prod + test externe) |
| `internal/sync` | `sync/killcollector` (prod + test), `sync/replayartifacts` (prod + test) |
| `internal/replaybuild` | `replaybuild` (prod + test) |
| `internal/persist` | `persist` (prod + test) |
| `internal/games/*` | `games/halo_infinite/film/killsource`, `games/halo_infinite/ingest`, `games/halo_infinite/replayidentity`, `games/halo_infinite/replaylabels`, `games/mappings` |
| Autres `internal/` | `himap`, `mapcatalog`, `mapdecoupe` |
| `cmd/*` | 20 binaires : `levelup`, `killsource`, `backfill_t0_film`, `diag_deaths`, `replay-equiv`, `mapcallouts-build`, `mapfond-build`, `mapfond-inventaire`, `mapfond-webp`, `mapobj-build`, `mapopads-build`, `mapplafond-mesure`, `mapquant-build`, `mapstruct-build`, `oddball-terrain`, `rdata_weapon_scan`, `statnames-sweep`, `vehicle-sprite`, `vs-measure`, `weapon-icons-build`, `zone-attribution` |

`internal/analysis/replay` importe `internal/analysis/filmdec` et `replay/mapvar` : les trois
descendent ensemble, cet arc ne franchit jamais de frontiere.

### Violations « `analysis/` -> `games/{slug}` » que le ratchet voit

Cinq fichiers, cinq franchissements — deux existaient AVANT le lot, trois sont revelees par
le deplacement.

| Fichier (`apps/go-api/`) | Paquet de titre vise | Etat |
|---|---|---|
| `internal/analysis/objectiveevents/assaut_footer_research_test.go` | `games/halo_infinite/film/filmcache` | preexistant |
| `internal/analysis/objectiveevents/extract_test.go` | `games/halo_infinite/film/filmcache` | preexistant |
| `internal/analysis/filmsource/source_test.go` | `…/film/filmdec` | revele par E.2 |
| `internal/analysis/sessionusage/usage_outcomes.go` | `…/film/replay` | revele par E.2, **seul en production** |
| `internal/analysis/weapon_index_equivalence_test.go` | `…/film/filmdec` | revele par E.2 |

Quatre sur cinq sont des tests. Le seul franchissement de production, `sessionusage`, lit les
types de sortie d'usage d'equipement du decodeur : son portage est le meme geste que celui
deja fait au lot A pour le document de rejeu (`domain/replaydoc`).

A l'inverse, `internal/analysis/` importe deja `games/mappings`, `games/weapons` et
`games/canonical` en de nombreux endroits : ce sont des paquets INTER-TITRES, ils ne couplent
a aucun titre et le ratchet les tolere par construction.

Etat des lieux du repertoire : `internal/games/` porte 3 titres (`halo_infinite`, `halo_5`,
`synthetic_title_b`) et 4 paquets inter-titres (`canonical`, `classification`, `mappings`,
`weapons`). `internal/games/halo_infinite/film/` existait deja (`damagetag`, `filmcache`,
`killicon`, `killsource`, `medalname`) : le decodeur rejoint ses cinq voisins.

### Chemins non-Go qui citent les deux dossiers

| Fichier | Nature | Traitement E.2 |
|---|---|---|
| `.github/workflows/ci.yml` (2 lignes) | `go vet` + `go test` du job rapide portent `./internal/analysis/...` | les DEUX paquets nommes explicitement a cote, pour que le job garde exactement sa couverture |
| `Makefile` (`go-api-test`, repli de `go-api-lint`) | idem | idem |
| `apps/go-api/.golangci.yml` | exemption `gocyclo/funlen/lll` sur `internal/analysis/` dont le decodeur heritait | regle jumelle datee sur `internal/games/halo_infinite/film/(filmdec\|replay)/` |
| `apps/web/src/features/match-replay/layers/deltaLayersContract.guard.test.ts` | lit les sources Go par chemin absolu (`GO`, `FILMDEC`) | chemins reecrits |
| 10 fichiers `apps/web/src/**` | commentaires citant un fichier Go | chemins reecrits |
| `config/titles/halo_infinite/mappings/regulation.toml` (2) | commentaires | chemins reecrits |
| `docs/COMMANDS.md`, `docs/FR/COMMANDS.md` (6) | commandes et prose | chemins reecrits |
| `scripts/check_test_baseline.sh` (1) | commentaire | chemin reecrit |
| `cmd/replay-corpus-gate` / `config/replay_corpus.toml` | AUCUNE reference aux deux dossiers | rien |
| `internal/archlint/gamefiles_tag_test.go` | racines DERIVEES du systeme de fichiers, pas en dur | rien |
| `CLAUDE.md` | la table « tag gamefiles » ne cite que `internal/himap/` | rien |
| `lefthook.yml` | aucune reference | rien |

Cote Go, 6 chemins etaient ecrits en `filepath.Join("analysis", "replay")` — invisibles a une
reecriture de chemin d'import, traites un a un (voir E.2).

## E.1 — Ratchet, pose AVANT le deplacement

Commit `5a0d1ec91` — `test(archlint/E.1): ratchet analysis -> games/{slug}`.

`apps/go-api/internal/archlint/no_title_package_in_analysis_test.go`. Il PARSE les imports
(`go/parser`, `ImportsOnly`) de tous les `.go` d'`internal/analysis/`, tests compris — un grep
se ferait tromper par les chemins cites en commentaire, et ces paquets en citent des dizaines.
La frontiere n'avait jusqu'ici qu'un garde-rail LOCAL (`no_temporal_title_import_test.go`,
limite a `analysis/temporal`).

- Tolere les 4 paquets inter-titres, verifie a chaque execution qu'ils existent encore.
- Tout autre repertoire d'`internal/games/` est un titre PAR DEFAUT : ajouter un titre ne
  demande aucune ligne ici, ajouter un paquet inter-titres en demande une.
- Allowlist datee 2026-09-12, une justification par ligne nommant le portage attendu ; une
  entree qui ne correspond plus a aucune violation fait rougir le test.
- Plancher de 300 fichiers parcourus (mesure : 1 337 avant, 399 apres) contre un ratchet muet.
- Clause TRANSITOIRE pour `analysis/filmdec` et `analysis/replay` au moment de la pose, avec
  son critere de retrait verifie par le test (une entree dont le repertoire n'existe plus fait
  rougir) — retiree par le commit E.2.
- Mutation de controle jouee : ajouter `games/halo_infinite/rankedplaylists` a
  `internal/analysis/xp_estimate.go` fait rougir le test, et il redevient vert une fois retire.

## E.2 — Deplacement pur

Commit unique — `refactor(film/E.2): deplacement pur de filmdec et replay sous
games/halo_infinite/film (ADR 0012)`.

- `git mv internal/analysis/filmdec internal/games/halo_infinite/film/filmdec`
- `git mv internal/analysis/replay  internal/games/halo_infinite/film/replay`
- Noms de paquets Go INCHANGES (`filmdec`, `replay`, `mapvar`) : aucun identifiant ne bouge.
- Reecriture de `analysis/(replay|filmdec)` en `games/halo_infinite/film/$1` sur 706 fichiers
  (Go, TS, MD, TOML, YML, SH) — imports, chaines et commentaires ensemble, pour qu'aucun
  commentaire ne pointe un chemin mort.
- 6 chemins ecrits segment par segment (`filepath.Join("analysis", "replay")`) corriges a la
  main : `cmd/replay-equiv/main.go`, et quatre ratchets d'`archlint`
  (`no_flag_carry_end_outside_close`, `no_identity_bridge_outside_registry`,
  `no_player_index_identity`, `no_rewritten_slot_band`).
- Profondeur : le deplacement ajoute DEUX niveaux. Toutes les remontees relatives des deux
  paquets ont ete reajustees (19 sites : `..` x5 -> x7 vers la racine du depot, x6 -> x8 dans
  `mapvar`, x3 -> x5 vers `apps/go-api`, le couple de sondage de `map_bounds_test`, et
  `usage_summary_families_guard` qui atteint `games/mappings` depuis l'interieur de `games/`).
  `golden_minibobine_test.go` atteint desormais `killsource` comme un FRERE (`../killsource/`)
  au lieu de traverser l'arbre. `filmsource/source_test.go`, qui ne bouge pas, pointe vers la
  nouvelle adresse de la fixture du rejeu. Les racines deduites a l'execution (`go.mod`,
  racine du cache film) n'ont pas bouge.
- `gofmt -w` : le nouveau chemin ne se trie pas au meme endroit dans les blocs d'import.
  155 fichiers reformates, TOUS dans le perimetre du deplacement (verifie par difference avec
  la liste des fichiers modifies).
- Empreinte du decodeur de kills (`killSourceDecoderFingerprint`, `collector.go`) recopiee.
  `KillSourceDecoderRev` **ne bouge PAS**, et c'est une decision ecrite : seules les lignes
  d'import des sources de `killsource` changent, le decodage est identique au bit pres ;
  bumper la revision reinscrirait les 1 210 films du parc au backlog pour un renommage de
  repertoire.
- Allowlist du ratchet E.1 : clause transitoire retiree (son critere de retrait est atteint),
  3 entrees ajoutees pour les franchissements que le deplacement rend visibles.

### Statistiques du renommage

`git diff --stat -M` sur le code et la config : **1 240 fichiers, 1 055 insertions,
1 057 suppressions** (le commit porte en plus le plan, le journal et ce rapport, d'ou
1 243 fichiers au `git show`).
`git diff --name-status -M` : **981 renommages, 259 modifications, 0 ajout, 0 suppression**.
A 99 % de similarite, 525 renommages sont a 100 % (fichiers non-Go et Go sans import touche)
et 247 a 99 %. Le plus bas est a 81 % (`world_object_precision_guard_test.go`, petit fichier
dont l'allowlist entiere est faite de chemins).

## E.3 — Gates

| Gate | Resultat |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 (seuls les messages de contraintes de build CGO preexistants) |
| `go test ./...` (module complet, CGO) | 0 echec |
| `go test -tags=integration -p 1 ./...` | 0 echec sur le module ; re-joue sur `persist`/`sync`/`migration`/`api/wire` avec code de sortie verifie : **exit 0**, 13 paquets `ok` |
| `go vet -tags=gamefiles ./internal/himap/ ./cmd/mapstruct-build/ ./cmd/mapfond-build/` | exit 0 |
| `golangci-lint run --new-from-merge-base=origin/main` | **0 issue** |
| Goldens `go test ./internal/games/halo_infinite/film/replay/ -run Golden` | 8 PASS, 1 SKIP (regeneration), **aucun fichier golden modifie dans le diff** |
| `make generate-types` | aucune derive (`generated.ts` inchange) |
| `npm run typecheck` puis `tsc -b --force` | exit 0 (force, pour ecarter le faux vert incremental) |
| Garde-rail web des chemins Go (`deltaLayersContract.guard.test.ts`) | 8 tests verts |
| `npm run lint` | 0 erreur (31 avertissements preexistants) |
| `replay-corpus-gate --reference=base --base=feat/v75` | **`[!]` BLOQUE, pas joue** — voir ci-dessous |

### Le gate corpus n'a pas pu tourner : le serveur local tient la base partagee

`replay-corpus-gate --reference=base --base=feat/v75 --parc-root <LevelUp-go-migration>` sort
en **code 2 (couverture incomplete, 8/8 temoins ABSENT)** sans avoir compare quoi que ce soit.
La cause n'est pas le lot : l'export des faits (`levelup replay-facts-export`, la seule etape
qui lit la base) echoue pour les 8 temoins sur

    OpenReadOnly(shared_matches_v2.duckdb) : File is already open in
    …\LevelUp-go-migrationpps\go-apiin\levelup.exe (PID 40148)

c'est-a-dire le SERVEUR DE DEV qui tourne sur :8000 et tient la base en RW. C'est le modele
mono-process d'ADR 0013/0016, applique : RO et RW sur le meme fichier depuis deux process est
interdit. Les lots B.5 et B-bis avaient joue ce gate **serveur arrete** ; il l'exige.

Je n'arrete pas le serveur de l'utilisateur. Le gate est donc statue `[!]` : a rejouer par le
pilote, serveur arrete, avant la fusion. Attendu : 0 difference, puisque le lot ne change
aucun comportement — et les deux preuves qui ne dependent pas de la base sont deja au vert
(goldens du decodeur identiques, aucun fichier golden dans le diff ; suite Go complete et
integration vertes).

Aucune ecriture n'a ete faite dans le parc : le gate travaille dans un repertoire temporaire,
et il est mort avant toute cuisson.

## Decouvertes (non traitees)

1. **`no_analysis_type_in_http_body_test.go` perd de la portee.** Ce ratchet interdit qu'un
   type d'`internal/analysis/` atteigne un corps de route Huma ; les types du decodeur
   sortent de son perimetre avec le deplacement. Il reste vert (son unique entree d'allowlist
   vise `analysis/patterns`) et le lot A avait deja projete les trois corps du rejeu sur
   `domain/replaydoc`, mais la regle ne les surveille plus. A instruire : etendre le prefixe
   surveille aux paquets de titre, ou poser la meme regle cote `games/`.
2. **`analysis/sessionusage` est le dernier franchissement de production.** Tant qu'il tient,
   `internal/analysis/` n'est pas title-agnostic au sens strict de l'ADR 0025 ; c'est un
   portage de types vers `domain/` (ou `games/canonical/`), pas un deplacement.
3. **Le `node_modules` du worktree** est une jonction vers celui du worktree partage
   (`LevelUp-go-migration`), posee pour jouer vitest sans reinstaller les dependances. A
   supprimer avec le worktree (lot F).
