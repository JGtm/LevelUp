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

- [x] B1 Inventaire des ratchets d'`internal/archlint` qui tolèrent une racine absente ou un compte qui
      baisse.
- [x] B2 Racine absente → `t.Fatalf` ; compte sous le plafond → rouge « baisse le plafond à N ».
      Gate : auto-preuve par mutation ; `go test ./internal/archlint/ -count=1` vert.

## C. Brique API (D3)

- [x] C1 Échappement des identifiants dans les URL de `internal/haloapi/infinite` (D3-L3-j). Gate : test
      de parité des URL saines figé AVANT, tests d'identifiants hostiles rouges avant / verts après.
- [x] C2 `xblauth/xbox_device_code.go` : erreur de lecture du corps rendue (D3-L2-e). Gate : test rouge
      avant, vert après.
- [x] C3 `xblauth/oauth_error.go` : texte conforme à l'ADR 0023 (D3-L2-i). Gate : aucun test ne fixe
      l'ancien texte ; tests du paquet verts.

## D. Backlog

- [x] D1 `.ai/BACKLOG.md` : entrée « Rafraîchissement OAuth : essayer d'abord la famille enregistrée ».
- [x] D2 `.ai/BACKLOG.md` : entrée « Publication des briques : modules, tags, README ; dépôts séparés ».

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


### Groupe B (2026-10-30)

- B1 inventaire (`grep os.Stat|IsNotExist|Skipf|Skip(` puis relecture de chaque site, et des
  comparaisons à un plafond `> X` sans pendant `< X`) :
  - racine absente tolérée : `no_identity_bridge_outside_registry` (`continue`),
    `no_player_index_identity` (`continue`), `no_rewritten_slot_band` (`t.Skipf`),
    `no_bare_duckdb_rw_open` (`continue`), `no_demo_forbidden_literal` (`continue`) ;
  - compte sous le plafond toléré : `no_french_label_literal` (`t.Logf`), `filmdec_package_vars`
    (`t.Logf`), `kill_feed_summaries_guard` (taille de l'allowlist), `pas_de_lecture_sous_ai`
    (taille de la liste), `i18n_tier_label_stored_not_served` (par fichier), `film_file_size`
    (`plafondsParFichier`), `film_function_length` (`plafondsParFonction`) ;
  - déjà conformes (rien à faire) : `briques_internes`, `briques_imports`, `film_facade_surface`,
    `film_finalise_predicate`, `film_tri_total`, `i18n_langue_suffixe`, `i18n_locale_constant`,
    `liste_liee_ratchet`, `no_discarded_remove`, `no_shared_writer_on_request_path`,
    `sync_root_freeze`, `title_slug_literal`, `unarmed_dotation_gate`, `un_seul_forcage_du_decodage`,
    `no_locale_literal` (plafond 0), `goSourcesUnder` (`Fatalf` sur racine absente) ;
  - hors règle : sauts sur OUTIL absent (`go`, `bash` : `ci_go_shards`, `film_lane_duckdb_free`,
    `titleseams_wired`, `briques_internes_fermeture`, `no_title_package_in_analysis`), `-short`
    (`i18n_text_sql`), découverte de dossiers par titre (`no_absolute_path_in_versioned_catalog`,
    plancher `Fatal` si aucun).
  Aucune racine déclarée n'a disparu : aucun ratchet mort à supprimer.
- B2 : helper `racineDeclaree(t, rel)` (`go_source_walk_test.go`, `t.Fatalf`) dans les cinq ;
  nouveau garde-rail `no_racine_absente_toleree_test.go` (AST des `*_test.go` d'archlint : `if` qui
  appelle `os.Stat` et dont le corps fait `continue` ou `X.Skip*`) avec auto-preuve (3 formes vues,
  0 faux positif, `exigerDossier`). Les sept ratchets de compte rougissent sous le plafond
  (« baisse le plafond à N »). Mise à la mesure : `frenchLabelAllowlist` (3 entrées à 0 retirées,
  `match_view_service.go` 3 -> 1), 18 entrées de `plafondsParFichier`, 8 de `plafondsParFonction`.
  Preuves par mutation : AVANT, les cinq racines renommées (`*_DISPARU`) laissent 3 tests verts et
  1 SKIP (le 5e rougit par hasard sur son allowlist) ; APRÈS, les cinq rougissent « racine déclarée
  ... » ; le nouveau garde-rail, nourri des cinq fichiers de HEAD (build tag jamais posé), en
  relève exactement cinq sites. Plafonds montés d'une marge (7 mutations) : AVANT 7 PASS ; APRÈS
  7 FAIL « baisse le plafond à N ». `go test ./internal/archlint/ -count=1` : ok (117,9 s) ;
  golangci-lint (base release) : 0 issue.


### Groupe C (2026-10-30)

- C1 : test `identifiants_url_test.go` écrit AVANT la modification (cible de requête reçue par un
  serveur `httptest`, `r.RequestURI`). AVANT : les 9 cibles saines vertes (UUID, `JGtm`,
  `xuid(NNN)`, xuid numérique, `csrseason13-3`), les 8 cas réservés rouges (`Foo Bar/Baz` coupé en
  deux segments, `Nom#1234` tronqué à `/hi/players/Nom`, `xuid(1#2)` tronqué, `?` pris pour la
  requête...), `FetchMatchStats("../x")` émis (`/matches/../x/stats`). APRÈS : tout vert, cibles
  saines identiques octet pour octet. Changements : `FetchMatchStats` refuse un non-UUID avant
  l'appel (`rexUUID`, comme `GetMatchStats`) et échappe ; service record et son URL de saison via
  `segmentJoueur` (gamertag `url.PathEscape`, forme `xuid(NNN)` telle quelle) ; `GetOperations`,
  `GetDecks`, `GetMatchPrivacy` échappent le xuid ; le classement échappe saison et playlist. Les
  autres sites du grep échappaient déjà (`GetMatchHistory`, `GetMatchStats`, film, skill, CSR,
  playlist, Discovery, apparence, rang de carrière) ou relèvent de S-3.
- C2 : `pollOnce` rend l'erreur de lecture (`lecture réponse poll (HTTP n): %w`). Test
  `httptest` à `Content-Length: 1000` pour 9 octets : AVANT « JSON poll: unexpected end of JSON
  input » (rouge), APRÈS `errors.Is(err, io.ErrUnexpectedEOF)` (vert).
- C3 : seul le commentaire d'en-tête portait le texte ; aucun test ne le fixait (grep `re-capture`
  et `refresh_token révoqué` dans `internal/haloapi` et tout `apps/go-api`). Texte conforme posé.
- Gate C : `go test ./internal/haloapi/... ./internal/platform/netguard/ ./internal/platform/auth/
  ./internal/games/halo_infinite/... ./cmd/levelup/` vert ; golangci-lint (base release) 0 issue.


### Groupe D et gate final (2026-10-30)

- D1, D2 : deux entrées `###` ajoutées à la fin de la section « 📋 Backlog » de `.ai/BACKLOG.md`
  (format des voisines, datées, effort marqué « estimation »). Registre des reports : neuf lignes
  traitées retirées, la ligne du code 4 réduite au témoin `f75e7053`, la ligne D3-L1-f renvoie à
  l'entrée de backlog ; section du lot ajoutée (S-1, S-3, S-4).
- Gate final (depuis `apps/go-api`) : `go build ./...` 0 ; `go vet ./...` 0 ; `go test
  ./internal/archlint/ ./internal/haloapi/... ./internal/platform/netguard/ -count=1` 0 ; `gofmt -l`
  (go-api et libs) vide ; `golangci-lint run --new-from-merge-base=origin/release/v8.1.0 ./...`
  0 issue ; `go test ./... -count=1` code 0 (195 paquets ok, 0 FAIL) ; `make go-api-test` code 0
  (42 paquets ok, `grammar` 78 s) ; `bash -n scripts/*.sh` sans erreur ; `gate-push.ps1` : 0 erreur
  d'analyse PowerShell, `Get-Command -Syntax` rend le script.

## Découvertes

- S-1 (A1) : `make -n <cible>` est inutilisable sur le poste dès qu'une recette contient `$(MAKE)` :
  GnuWin32 make 3.81 exécute la ligne récursive même sous `-n` et échoue (`CreateProcess(NULL, "")`,
  e=87), guillemets ou non. Limite de l'outil ; non traité.
- S-2 (B) : D2-L3-c (les ratchets d'`apps/go-api` qui balaient `internal/` sans citer le film ne
  couvrent plus le décodeur) n'est pas dans le périmètre de B (racine absente, compte en baisse) :
  inventaire non fait, reste au registre.
- S-3 (C1) : les chemins CMS rendus par le service lui-même (`appearance.go` fichier de progression,
  `nameplate.go` emblème et plaques, `film.go` préfixe de blob) entrent sans échappement. Ce sont
  des chemins à plusieurs segments, pas des identifiants : les échapper segment par segment
  changerait les octets émis pour les caractères réservés (`(`, `)`, `,`...) que le service écrit
  lui-même, sans appel réel pour le vérifier. Non traité.
- S-4 (C2) : `peoplehub_resolver.go` et `xbox_profile_resolver.go` lisent `body, _ :=
  io.ReadAll(io.LimitReader(resp.Body, 512))` sur le chemin d'erreur (extrait pour le message) :
  même forme, hors périmètre de C2. Non traité.

## Protocole de reprise

Relire `plan-execution`, puis ce journal ; `git log --oneline -6` sur `chore/suites-decouplage` ;
reprendre à la première case non statuée. Les commits portent le préfixe du groupe (A, B, C, D).
