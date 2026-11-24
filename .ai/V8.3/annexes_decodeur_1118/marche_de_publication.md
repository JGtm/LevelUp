# Publier une version du décodeur et la monter dans LevelUp : marche à suivre (rejouée à blanc le 2026-11-18)

Tiré des plans de montée v1.3.9 et v1.4.0 (`.ai/V8.3/`), de `docs/COMMANDS.md` (« Développer une brique »),
de l'ADR 0043 et des pièges notés au journal. `DEC` = clone du dépôt `JGtm/halo-infinite-film-decoder`
(PUBLIC). Les gestes 3 à 10 ont été REJOUÉS le 2026-11-18, sans aucun changement de code, sur le clone
`C:/t/dec/main` (branche `feat/v1.5.0`, tête = tag `v1.4.1`) et le dossier de travail `C:/t/dec-lu`, en
mode vérification (aucun drapeau de régénération). Chaque geste porte « vérifié le 2026-11-18 » (avec sa
durée et son résultat) ou « non rejoué : raison ». Sorties brutes sous `C:/t/dec/sorties/` (hors dépôt).

## 0. Conventions de ce relevé (à lire d'abord)

- `HOTE` = `<poste>/Downloads/Scripts/LevelUp` (dossier principal : films, bases, liste des
  originaux ; LECTURE SEULE). `LU` = `C:/t/dec-lu`. `DEC` = `C:/t/dec/main`.
- Cache Go dédié : `export GOCACHE="$LOCALAPPDATA/go-build-dec-0"` (convention : un cache par lot).
- Aucune jonction nulle part. La voie sans jonction, par commande :
  - tests du clone : `DECODEUR_FILM_RACINE_DONNEES=<HOTE>` (lecture seule ; `internal/depottest/depottest.go:30`) ;
  - équivalence : `-repo-root LU` ET une COPIE (pas une jonction) des films du corpus sous
    `LU/data/cache/film_chunks/<id>` et `LU/data/cache/film_manifests/<id>.json` (`data/cache/*` est ignoré
    par git ; la copie des 20 films pèse 676 Mo). Raison : `cmd/replay-equiv/child.go:87-108` bâtit son
    résolveur par `title.NewPathResolver(o.repoRoot)` et n'installe jamais les racines du processus ;
    `LEVELUP_DATA_DIR` et `film_archive_dir` n'y ont donc aucun effet, et l'archive est toujours
    `<repo-root>/data/cache`. Passer `-repo-root HOTE` marcherait SANS copie mais écrirait le verrou
    `film_decode.lock` et son `.json` sous `HOTE/data/cache` (`internal/filmrun/solo.go:92-114`) : refusé ;
  - gate du corpus : `--parc-root` désigne un « parc » que le gate ÉCRIT (cache des artefacts de la base :
    `cmd/replay-corpus-gate/base.go:216`, `basecache.go:46` = `<parc>/data/cache/replay_corpus_gate_base/`).
    Parc de travail sous `C:/t/dec/sorties/parc` : `config/` + `data/titles/halo_infinite/{reference,warehouse}`
    (copie de `metadata.duckdb`, son `.wal` et `shared_matches_v2.duckdb`) + `data/cache/film_chunks` et
    `film_manifests` des 19 témoins du manifeste (1,2 Go). Verrou : `--lock-root` hors de tout `data/`.
- Le clone se teste avec `GOWORK=off`, PAS avec l'espace de travail engendré : voir geste 1.

## A. Dans le dépôt du décodeur

1. **Espace de travail.** vérifié le 2026-11-18 (1 s).
   `cd LU && bash scripts/briques-workspace.sh C:/t/dec/main` puis
   `export GOWORK="C:/t/dec-lu/levelup-briques.work"` : fichier engendré, ABSENT de `git status` (ignoré :
   `.gitignore:370`). ÉCART : depuis le dossier du clone, ce `GOWORK` fait échouer tout `go test` / `go list`
   d'un SOUS-PAQUET (`directory internal\profile is contained in a module that is not one of the workspace
   modules`), parce que le script écrit l'entrée `use C:/t/dec/main` avec des barres obliques (`pwd -W`) ;
   avec des barres inverses (`C:\t\dec\main`) tout passe (essayé). Donc : commandes DEPUIS `DEC` =
   `GOWORK=off` (le clone est un module autonome, c'est ce que joue sa CI) ; commandes DEPUIS
   `LU/apps/go-api` qui doivent voir le clone (`replay-equiv`, `replay-build`, `replay-verite`, `levelup`) =
   le `GOWORK` engendré (le module `apps/go-api` y est désigné en relatif, donc sans défaut). Piège confirmé
   et précisé : le gate du corpus ne retire pas tout seul `GOWORK` partout (geste 9).
   Pour une équivalence par piste : le script écrit toujours le MÊME fichier ; en garder une copie par piste
   hors dépôt (`GOWORK=<copie>`) et ne changer que le dossier du clone qu'elle désigne ; `-repo-root LU` et
   la copie des films servent à toutes les pistes (le clone est choisi par `GOWORK`, pas par `-repo-root`).
2. **Règles du dépôt public.** non rejoué : règles, pas commandes. Les deux gardes nommés existent :
   `internal/depottest/taille_test.go`, `internal/depottest/balayages_test.go` (joués par le geste 6 : `ok`).
   `replay/document.go` : 500 lignes exactement (`wc -l`, 2026-11-18). Contrôle des identifiants de joueur
   avant chaque commit : motif `25(33|35)[0-9]{12}` sur les fichiers modifiés, binaires décompressés compris.
3. **Révisions de couche.** vérifié le 2026-11-18 en mode vérification (20 s, code 0) :
   `cd DEC && GOWORK=off go test -count=1 ./internal/profile/ ./internal/source/ ./internal/grammar/
   ./internal/facts/killsource/ ./internal/facts/objectives/ -run 'RevSuit|ChroniqueDe|ChroniqueCouvre|ChroniqueDObjectives' -v`
   (10 tests `PASS`) et
   `go test -count=1 ./revision/ -run TestPerimetreDeChaqueCoucheEgaleSonGolden -v` (3 s, `PASS`).
   Ordre de régénération inchangé : profile et source, grammar, killsource, objectives. Régénération
   (NON rejouée : écrit) : drapeau ET variable, commandes imprimées par chaque test :
   `LEVELUP_UPDATE_PROFILE_REV=1 go test ./internal/profile/ -run TestProfileRevSuitLaCoucheProfil -update-profile-rev`
   (de même `SOURCE` / `TestSourceRevSuitLaCoucheSource`, `GRAMMAR` / `./internal/grammar/` /
   `TestGrammarRevSuitLaGrammaire`, `KILLSOURCE` / `./internal/facts/killsource/` / `TestKillsourceRevSuitLaSortie`,
   `OBJECTIVES` / `./internal/facts/objectives/` / `TestObjectivesRevSuitLaSortie`) ; périmètres :
   `LEVELUP_UPDATE_PERIMETRES=1 go test ./revision/ -run TestPerimetreDeChaqueCoucheEgaleSonGolden -update-perimetres`.
   Une porte de régénération ne rend jamais `ok` : relancer sans drapeaux. Entrées de chronique obligatoires.
   MANQUE DE L'ANNEXE D'ORIGINE : un sixième golden de révisions et de formes existe,
   `types/testdata/shapes.golden` : `LEVELUP_UPDATE_TYPES_SHAPES=1 go test ./types/ -run
   TestFormesDesTypesEgalentLeGolden -update-types-shapes` ; sa vérification : `go test -count=1 ./types/`
   (2 s, `ok`). D'autres portes de la grammaire se refigent à part quand le balayage change (drapeaux
   déclarés : `-update-ecs-table-level`, `-update-frame-closure`, `-update-keyframe-closure`,
   `-update-keyframe-closure-cuisson`, `-update-etat-complet-bobines`, `-update-structure-fermeture`,
   `-update-golden-familles`, `-update-graines-faits`) ; toutes jouées en vérification par le geste 6.
4. **Schéma.** vérifié le 2026-11-18 en mode vérification (14 s, code 0) :
   `GOWORK=off go test -count=1 ./replay/ -run DocumentShape -v` : `MatchesGolden` et `GoldenCarriesCurrentSchema`
   `PASS`, `Regenerate` `SKIP` (attendu). Régénération (NON rejouée) : le verrou de variable s'appelle
   `REPLAY_CONTRACT_UPDATE` (`replay/contract_fixtures_test.go:77`) :
   `REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run DocumentShape -update`. Canal de faits neuf :
   `filmFactsMagic` plus un. `SchemaVersion` courant : 125 (golden).
5. **Goldens.** vérifié le 2026-11-18 : `GOWORK=off go test -count=1 ./replay/ -run Golden -v` ; sans données
   de l'hôte 3 s, code 0 mais 30 `PASS` et le reste `SKIP` (les films d'appui manquent : le résultat est
   VIDE, ne pas le lire comme un témoin) ; AVEC `DECODEUR_FILM_RACINE_DONNEES=HOTE` : 30 s,
   `-run 'ContractFixtures|Golden|DocumentShape'`, 97 `PASS`, seuls les tests `...Regenerate` / `Fidelite` /
   `ReecrireIdentites` restent `SKIP`. Régénération (NON rejouée : écrit) : `GoldenInputs` (`go test ./replay/
   -run TestGoldenInputsRegenerate -update`), `GoldenBuildsRegenerate` (`-update`, variable `REPLAY_FILM_CACHE`),
   `GoldenBuildsAssemblyRegenerate -update-golden-builds-assembly`, portes nommées `-update-golden-fins-de-vie`
   et `-update-golden-equipement-origines`. Variables lues : `REPLAY_FILM_CACHE`, `REPLAY_FILM_DIR`,
   `REPLAY_IDENTITES_TABLE`.
6. **Tests du clone AVEC les données de l'hôte.** vérifié le 2026-11-18 (284 s, code 0, 33 paquets `ok`, aucun
   `FAIL`) :
   `cd DEC && GOWORK=off DECODEUR_FILM_RACINE_DONNEES="<poste>/Downloads/Scripts/LevelUp" go test ./... -count=1 -p 1 -timeout 3000s`.
   `-p 1` : un seul paquet à la fois, donc jamais deux films en mémoire (le défaut lance 16 paquets).
   La racine désigne l'HOTE (films du parc, `apps/web`, catalogues) : tout est lu ; aucun test n'écrit
   (statut git de l'hôte et dates de `data/cache` inchangés). Pour la RÉGÉNÉRATION des fixtures la racine doit
   être le worktree (geste 7). Sans la variable ces tests sautent ; une valeur qui ne désigne pas un dossier
   fait ÉCHOUER (jamais sauter).
7. **Fixtures de contrat du web.** vérifié le 2026-11-18 (2 s sans hôte = tout `SKIP` ; 30 s avec) :
   `DECODEUR_FILM_RACINE_DONNEES=HOTE GOWORK=off go test -count=1 ./replay/ -run ContractFixtures -v` : budget
   `PASS` (8 fixtures, 4 936 939 octets, plafond 4 980 736, MARGE 43 797 : l'annexe d'origine disait juste),
   `UnSeulJeuVivant`, `MatchCommitted` (8 sous-tests), `CarryCurrentSchema` : `PASS`. Régénération (NON
   rejouée : écrit dans `apps/web`) : `REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run ContractFixtures -update`
   avec la racine = le worktree, pas l'HOTE.
8. **Équivalence.** vérifié le 2026-11-18 (343 s = 5 min 43, et NON 12 min ; code 0) :
   `cd LU/apps/go-api && GOWORK=<engendré> go run ./cmd/replay-equiv -repo-root C:/t/dec-lu -originaux
   "<poste>/Downloads/Scripts/LevelUp/data/film_identites/xuids_originaux.txt" -out-dir C:/t/dec/sorties/equiv_tsv`
   (films copiés sous `LU/data/cache`, voir § 0) : `BILAN : 20 identique(s), 0 different(s), 0 ecarte(s),
   0 echec(s), 0 illisible(s)`. `-out-dir` garde un TSV de digests par film (une ligne par étape, première
   ligne `# digest-grammar: 2`). Le harnais ne lit que le clone des références `replay/testdata/equivalence/`.
   Attribuer CHAQUE écart, puis `-update` (écrit dans le clone), puis une passe sans `-update` : 20
   identiques. `-deux-passes` pour tout canal de faits neuf : vérifié sur `-films 000d5950 -deux-passes`
   (16 s, code 0) : « artefact IDENTIQUE à l'octet », mais « 1 écart(s) A CLASSER ... kill » déjà présent sur
   v1.4.1 sans aucun changement : c'est le bruit de base à connaître avant de lire une passe à deux voix. Un
   film HORS corpus (sans `<id>.facts.json` dans `testdata/equivalence`) rend `ECHEC (code 11)` : le harnais
   ne mesure que le corpus de 20 ; pour un autre film voir `instruments.md`.
9. **Gate du corpus.** vérifié le 2026-11-18 (1 299 s = 21 min 40, 19 témoins, code 0) :
   `cd LU/apps/go-api && CGO_ENABLED=0 go build -o C:/t/dec/sorties/replay-corpus-gate.exe ./cmd/replay-corpus-gate`
   puis, avec `GOWORK=off` : `replay-corpus-gate.exe --reference=base --base <sha de LU> --strict
   --source-root C:/t/dec-lu --parc-root C:/t/dec/sorties/parc --lock-root C:/t/dec/sorties/gate_lock
   --work-root C:/t/dec/sorties/gate_work2 --json C:/t/dec/sorties/g9_gate.json` (script :
   `C:/t/dec/sorties/lancer_gate.sh`). Résultat : 19 témoins `ok`, 0 gain, 0 perte, 0 changement, schéma 125
   des deux côtés, registre des replis de la base lu (129), banc de vérité `ok` partout. ÉCARTS :
   (a) l'annexe disait « il retire GOWORK » : FAUX pour la compilation du registre
   (`cmd/replay-corpus-gate/verite_registre.go:52` hérite l'environnement) ; avec le `GOWORK` engendré posé,
   le gate affiche « registre des replis d'avant INCONNU » (le worktree de base n'est pas dans l'espace de
   travail) et jugerait tout repli neuf FAUX : lancer avec `GOWORK=off`. (b) Ce que le gate juge
   RÉELLEMENT : `compilerReplayBuild` (`base.go`, `environnementSansGOWORK`) compile le `replay-build` de
   HEAD et celui de la base SANS espace de travail, donc contre le décodeur PUBLIÉ que `apps/go-api/go.mod`
   exige (v1.4.1). Le clone n'est JAMAIS jugé : sans changement de LevelUp le gate compare v1.4.1 à v1.4.1 et
   rend « ni gain ni perte » par construction. Pour juger une branche du décodeur il faut que `go.mod` du
   dossier source la désigne (pseudo-version ou `replace` local non commité, à vérifier à l'étape L.2) avec
   `--base` = le commit d'avant. (c) `--parc-root` doit être un parc que l'on accepte de voir écrit (cache
   de la base) : voir § 0. (d) Huit lignes `decision sans effet — a retirer du manifeste` (décisions de
   `config/replay_corpus.toml` datées du 2026-11-16, décodeur v1.4.1 : `flagCarries.spans.returnProgress/*`,
   `coverage.hits.{vitalsUnread,stoppedFrames,noVictim,noShooter}`, `coverage.flagCarries.{seatsBeyondBand,
   gaugePoints}`) : à nettoyer avec l'accord du user. (e) Le gate crée puis retire un worktree git détaché
   (`git worktree add --detach`) sous `--work-root` ; un `--work-root` qui garde un `base-worktree` d'une
   passe interrompue fait échouer la suivante (`exit 2`) : changer de `--work-root`, ou `git worktree remove
   --force` puis `git worktree prune`. Une passe tuée en cours laisse le worktree enregistré.
10. **Registre des replis.** vérifié le 2026-11-18 (20 s, code 0) : `cd LU/apps/go-api && GOWORK=<engendré> go
    run ./cmd/replay-verite -registre > registre.json` : 129 entrées (128 `true`, 1 `false` = compteur non
    câblé), égal au décompte de `FORMAT.md:641` (« 129 declarations »). La sortie de `go test ./internal/facts/fallback/`
    est `ok` (geste 6).
11. **Documentation.** non rejoué : aucune commande ; l'annexe d'origine reste juste (`FORMAT.md`,
    `GLOSSARY.md`, `README.md`, `NOTICE.md`).
12. **Une PR, tag, release.** non rejoué : publication (geste de pilote, hors de la marche à blanc).

## B. Dans LevelUp

13. `cd apps/go-api && go get github.com/JGtm/halo-infinite-film-decoder@vX.Y.Z && make tidy` ; `go mod verify`.
    non rejoué : modifie `go.mod`. Vérifié statiquement : `make tidy` est la recette de `apps/go-api/Makefile:78`
    (`GOWORK=off go mod tidy`), pas du `Makefile` racine.
14. Miroirs Go : `domain/replaydoc`, `service/replayview` (`convert_document.go`), gardes de parité et de
    formes jumelles, `make openapi-gen` (racine, `Makefile:135`, `CGO_ENABLED=1`), `make generate-types`
    (racine, `Makefile:57`, `npm run generate-types`), `contracttest`, plafonds de
    `archlint/film_facade_surface_test.go` (fichier présent). non rejoué : écrit.
15. Miroirs web : clés racine de `replayDocumentSchema.ts`, `replayNormalize.ts`, fixtures web,
    `go run ./cmd/film-web-snapshot` (dossier `cmd/film-web-snapshot/` présent). non rejoué : écrit.
16. Gates sans `GOWORK` : `go build`, `go vet`, `go test ./...` ; `-tags=integration -p 1` sur `replaybuild`,
    `sync/filmfetch`, `sync/killcollector`, `sync/replayartifacts`, `internal/api/wire` ; `make replay-equiv` ;
    `make gate-push` (`Makefile:420`, pose `GOWORK=off`) ; suite complète du web UNE fois par le pilote ; CI
    lue après chaque poussée. non rejoué : hors périmètre de la marche à blanc (gate-push = 25 min + 12 min).
    Rappel mesuré : `make replay-equiv` et `gate-push` lisent `<racine du dépôt>/data/cache` : dans un worktree
    il faut les films copiés (§ 0), sans jonction.
17. Notes de version en français et en anglais, changelog (ligne des briques publiées), journal, backlog,
    registre des reports. non rejoué : rédaction.

## C. Parc

18. Régénération au go du user, une fois, serveur arrêté, un film à la fois (`backfill-replay --only-existing`).
    Précédent : 1 455 films en 2 h 26. Puis les dérivés (25 minutes) : `backfill-usage-summary`,
    `backfill-pad-tiers --force`, `backfill-killsource --workers 1`, `backfill-vehicle-takes --force`,
    `backfill-flag-grabs-net --force`, `tactical-rasters --backfill`. non rejoué : écrit le parc ; présence
    des sous-commandes et des drapeaux `--only-existing`, `--force`, `--workers` constatée dans
    `apps/go-api/cmd/levelup/` le 2026-11-18.
19. Prod : dossier d'attente, bascule simulée puis appliquée, après le déploiement de la version ; seuls les
    rejeux se copient, jamais une base. non rejoué : geste de prod.
