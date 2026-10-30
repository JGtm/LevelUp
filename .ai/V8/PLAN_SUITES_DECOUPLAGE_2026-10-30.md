# Plan — Suites du découplage (outillage local, ratchets muets, robustesse haloapi, backlog) — 2026-10-30

Source : `.ai/REGISTRE_REPORTS.md`, sections « découplage D1 », « D2 », « D3 » (items repris ci-dessous).
Exécution sous le contrat du skill `plan-execution` : ordre strict, un item à la fois, gate passé avant
l'item suivant, chaque item statué `[x]` / `[~]` / `[!]`, zéro fix hors périmètre (section « Découvertes »).

Branche `chore/suites-decouplage` depuis `release/v8.1.0` (799b1f648), worktree `../LevelUp-wt-suites`.
Caches : `GOCACHE=%LOCALAPPDATA%\go-build-suites`, `GOLANGCI_LINT_CACHE=%LOCALAPPDATA%\golangci-suites`.
Un commit par groupe (A, B, C, D). Zones réservées à un autre agent (ne pas toucher) : cible
`replay-equiv` du `Makefile`, `docs/COMMANDS.md` + FR (ligne `make replay-equiv`),
`libs/film/replay/testdata/equivalence`. Aucun jeton, aucun appel réseau réel (`httptest`).

## Objectif

Les gates locaux du poste Windows tournent (gate-push, go-api-test, baseline UTF-8, code 4 du gate de
corpus) ; `air` relance sur une brique ; aucun ratchet d'`archlint` ne se tait quand sa racine disparaît
ou que son compte baisse ; la brique `haloapi` échappe ses identifiants, ne masque plus une erreur de
lecture et ne prescrit plus de re-capture ; deux entrées de backlog consignent les suites décidées.

## A. Outillage local

- [x] A1 `Makefile` `gate-push` : `$(MAKE) libs-test` → `"$(MAKE)" libs-test` (D1-28/33).
      Gate : `make -n gate-push` (ou preuve équivalente si `make -n` échoue).
- [x] A2 `Makefile` `go-api-test` : délai des couches du décodeur (D2-L3-g). Gate : durée mesurée,
      `make go-api-test` code 0.
- [x] A3 `scripts/gate-push.ps1` : capture JSONL en UTF-8 (D2-L5-a). Gate : run partiel
      `./internal/service/...`, JSONL UTF-8, `check_test_baseline.sh` sans faux absents (avant/après).
- [x] A4 `apps/go-api/.air.toml` : surveiller `libs/film`, `libs/gamefiles`, `libs/himap` (D1-20).
      Gate : syntaxe (air ou lecture).
- [x] A5 `make replay-corpus-gate` : propager le code 4 (D2-L5-c). Gate : `--help` ou exécution courte.

## B. Ratchets muets (D2-L3-b/c)

- [ ] B1 Inventaire des ratchets d'`internal/archlint` qui tolèrent une racine absente ou un compte qui
      baisse.
- [ ] B2 Racine absente → `t.Fatalf` ; compte sous le plafond → rouge « baisse le plafond à N ».
      Gate : auto-preuve par mutation ; `go test ./internal/archlint/ -count=1` vert.

## C. Brique API (D3)

- [ ] C1 Échappement des identifiants dans les URL de `internal/haloapi/infinite` (D3-L3-j). Gate : test
      de parité des URL saines figé AVANT, tests d'identifiants hostiles rouges avant / verts après.
- [ ] C2 `xblauth/xbox_device_code.go` : erreur de lecture du corps rendue (D3-L2-e). Gate : test rouge
      avant, vert après.
- [ ] C3 `xblauth/oauth_error.go` : texte conforme à l'ADR 0023 (D3-L2-i). Gate : aucun test ne fixe
      l'ancien texte ; tests du paquet verts.

## D. Backlog

- [ ] D1 `.ai/BACKLOG.md` : entrée « Rafraîchissement OAuth : essayer d'abord la famille enregistrée ».
- [ ] D2 `.ai/BACKLOG.md` : entrée « Publication des briques : modules, tags, README ; dépôts séparés ».

## Gate final

`go build ./... ; go vet ./...` ; `go test ./internal/archlint/ ./internal/haloapi/...
./internal/platform/netguard/ -count=1` ; `go test ./... -count=1` ; `gofmt -l` vide ;
`golangci-lint run --new-from-merge-base=origin/release/v8.1.0 ./...` 0 issue ; `make go-api-test`
code 0 ; `bash -n scripts/*.sh` ; syntaxe PowerShell de `scripts/gate-push.ps1`.

## Journal

### Groupe A (2026-10-30)

- A1 : `make -n gate-push` échoue AVANT et APRÈS (`CreateProcess(NULL, "")`, e=87) : sous `-n`, GnuWin32
  make 3.81 exécute quand même la ligne récursive et ne sait pas la lancer ; limite de l'outil, pas
  de la ligne. Preuve retenue : (1) reproduction minimale avec le même make (`"$(MAKE)" b` rend 0,
  `$(MAKE) b` rend « syntax error near unexpected token `(' », l'erreur du registre) ; (2)
  `make gate-push` réel avec `golangci-lint`, `go` et `npm` remplacés par des souches en tête de
  PATH : le sous-make `"C:/Program Files (x86)/GnuWin32/bin/make" libs-test` entre et sort
  (`== libs/film`, `== libs/gamefiles`, `== libs/himap`), la recette continue jusqu'à la baseline.
- A2 : AVANT, `make go-api-test` rouge : `libs/film/internal/grammar` panique au délai (`FAIL ...
  61.328s`, `-timeout 60s`), `replay` 45 s ; la première étape (go-api) passe en 50 s, à 10 s de
  son délai. `grammar` seul avec 600s : 151 s. Délai porté à 600s sur les trois lignes (valeur des
  pas miroirs de ci.yml). APRÈS : `make go-api-test` code 0 en 196 s (`grammar` 118,6 s,
  `replay` 78,5 s), 42 paquets `ok`, 0 FAIL.
- A3 : capture par `cmd /c "go test -json ... > fichier 2>> stderr"` dans une fonction
  `Invoke-GoTestJson` (deux sites : go-api et libs/film). Preuve sur `./internal/service/...`
  (`-tags=integration`, console forcée en page OEM 850 comme le terminal du poste) : AVANT
  (capture d'origine), 0 flèche dans le JSONL et `check_test_baseline.sh` rend
  `internal/service : 58/1076 absents (2457 présents)` (le compte du registre) ; APRÈS, 359 flèches,
  JSONL UTF-8 valide, `internal/service` absent du bilan des absents. Avec une console déjà en
  65001, les deux captures donnent les mêmes 2876 noms (le défaut dépend de la page de la console).
- A4 : `include_dir` hors de root = répertoires supplémentaires (air v1.65.1,
  `runner/config.go normalizeIncludeDirs` + `engine.go watchConfiguredDirs`), la surveillance de
  root reste entière. `exclude_dir` compare un chemin relatif exact : exclusion des `testdata/` et
  `research/` des briques par `exclude_regex` (`_test.go` déjà exclu). Syntaxe : `air -v -c
  .air.toml` imprime la ligne de version seule (config lue et motifs compilés) ; une copie cassée
  imprime la bannière de repli.
- A5 : `go build -o bin/replay-corpus-gate$(go env GOEXE)` puis exécution, `ARGS` transmis ;
  `apps/go-api/bin/` ignoré par git. Faux témoin `--temoins faux_temoin` (code 2, usage) : AVANT
  `go run` rend 1 (« exit status 2 » dans le journal seulement) ; APRÈS le binaire rend 2 et make
  imprime `Erreur 2` ; `--help` rend 0. Le commentaire de la cible disait « code 2 » pour un témoin
  absent : corrigé en code 4 (doc inversée, même cible).

## Découvertes

## Protocole de reprise

Relire `plan-execution`, puis ce journal ; `git log --oneline -6` sur `chore/suites-decouplage` ;
reprendre à la première case non statuée. Les commits portent le préfixe du groupe (A, B, C, D).
