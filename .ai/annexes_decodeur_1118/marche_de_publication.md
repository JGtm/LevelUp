# Publier une version du décodeur et la monter dans LevelUp : marche à suivre (relevé du 2026-11-18)

Tiré des plans de montée v1.3.9 et v1.4.0 (`.ai/V8.3/`), de `docs/COMMANDS.md` (« Développer une brique »),
de l'ADR 0043 et des pièges notés au journal. `DEC` = clone du dépôt `JGtm/halo-infinite-film-decoder`
(PUBLIC). À revérifier commande par commande à l'étape 0 du plan : ce relevé vient d'une lecture, aucune de
ces commandes n'a été rejouée le 2026-11-18.

## A. Dans le dépôt du décodeur

1. Espace de travail : `bash scripts/briques-workspace.sh ../<clone>` puis `GOWORK="$PWD/levelup-briques.work"`.
   Piège : le gate du corpus retire `GOWORK` et juge alors la version PUBLIÉE, pas le clone.
2. Règles du dépôt public : aucun vrai identifiant de joueur dans une fixture ou un golden (contrôle des
   suites de 16 chiffres commençant par 2533 ou 2535, binaires décompressés compris, avant chaque commit) ;
   plafond de 500 lignes par fichier (`internal/depottest/taille_test.go`) ; aucune boucle de balayage bit à
   bit hors de la liste nommée (`internal/depottest/balayages_test.go`) ; un repli neuf est un FAUX au banc.
   `replay/document.go` est à 500 lignes exactement : tout champ racine neuf demande d'abord une extraction.
3. Révisions de couche, dans l'ordre profile et source, grammar, killsource, objectives ; chaque porte exige
   son drapeau ET sa variable (`LEVELUP_UPDATE_GRAMMAR_REV=1 … -update-grammar-rev`, de même pour
   `KILLSOURCE` et `OBJECTIVES`) ; périmètres : `LEVELUP_UPDATE_PERIMETRES=1 go test ./revision/ -run
   TestPerimetreDeChaqueCoucheEgaleSonGolden -update-perimetres`. Une porte de régénération ne rend jamais
   `ok` : relancer sans drapeaux. Entrées de chronique obligatoires.
4. Schéma : `SchemaVersion` monté, entrée de `document_chronicle.go`, puis
   `REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run DocumentShape -update`. Canal de faits neuf :
   `filmFactsMagic` plus un.
5. Goldens : `GoldenInputs`, `GoldenAssembly`, `GoldenBuildsRegenerate`,
   `GoldenBuildsAssemblyRegenerate -update-golden-builds-assembly`, et les portes nommées
   (`-update-golden-fins-de-vie`, `-update-golden-equipement-origines`).
6. Tests du clone AVEC les données de l'hôte : `DECODEUR_FILM_RACINE_DONNEES=<worktree LevelUp> go test ./... -count=1`.
   Sans la variable ces tests sautent ; la CI du décodeur ne les joue pas, celle de LevelUp oui (deux
   versions de rattrapage publiées pour cet oubli).
7. Fixtures de contrat du web : `REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run ContractFixtures -update`,
   racine = le worktree, pas le dépôt principal. Budget : 4 980 736 octets, 43 797 de marge au 2026-11-18.
8. Équivalence : `cd apps/go-api && go run ./cmd/replay-equiv -repo-root <worktree> -originaux <LevelUp>/data/film_identites/xuids_originaux.txt`,
   20 films, 12 minutes par passe. Attribuer CHAQUE écart, puis `-update`, puis une passe sans `-update` :
   20 identiques. `-deux-passes` pour tout canal de faits neuf.
9. Gate du corpus : `go run ./cmd/replay-corpus-gate --reference=base --base <sha> --strict`, 25 minutes. Une
   PERTE bloque sans mécanisme d'acceptation ; une décision inscrite à `config/replay_corpus.toml` demande
   l'accord du user.
10. Registre des replis : `go run ./cmd/replay-verite -registre` ; décompte de `FORMAT.md`.
11. Documentation : `FORMAT.md`, `GLOSSARY.md`, `README.md`, `NOTICE.md` (crédit à Den pour toute table
    reprise de libfilmshell).
12. Une PR (code, version, notes), CI verte, squash, tag sur le commit de squash, release GitHub dont le corps
    dit en clair tout changement de sortie.

## B. Dans LevelUp

13. `cd apps/go-api && go get github.com/JGtm/halo-infinite-film-decoder@vX.Y.Z && make tidy` ; `go mod verify`.
14. Miroirs Go : `domain/replaydoc`, `service/replayview` (`convert_document.go`), gardes de parité et de
    formes jumelles, `make openapi-gen`, `make generate-types`, `contracttest`, plafonds de
    `archlint/film_facade_surface_test.go`.
15. Miroirs web : clés racine de `replayDocumentSchema.ts`, `replayNormalize.ts`, fixtures web,
    `go run ./cmd/film-web-snapshot` (instantané périmé = une version de rattrapage déjà payée).
16. Gates sans `GOWORK` : `go build`, `go vet`, `go test ./...` ; `-tags=integration -p 1` sur `replaybuild`,
    `sync/filmfetch`, `sync/killcollector`, `sync/replayartifacts`, `internal/api/wire` ; `make replay-equiv` ;
    `make gate-push` ; suite complète du web UNE fois par le pilote ; CI lue après chaque poussée.
17. Notes de version en français et en anglais, changelog (ligne des briques publiées), journal, backlog,
    registre des reports.

## C. Parc

18. Régénération au go du user, une fois, serveur arrêté, un film à la fois (`backfill-replay --only-existing`).
    Précédent : 1 455 films en 2 h 26. Puis les dérivés (25 minutes) : `backfill-usage-summary`,
    `backfill-pad-tiers --force`, `backfill-killsource --workers 1`, `backfill-vehicle-takes --force`,
    `backfill-flag-grabs-net --force`, `tactical-rasters --backfill`.
19. Prod : dossier d'attente, bascule simulée puis appliquée, après le déploiement de la version ; seuls les
    rejeux se copient, jamais une base.
