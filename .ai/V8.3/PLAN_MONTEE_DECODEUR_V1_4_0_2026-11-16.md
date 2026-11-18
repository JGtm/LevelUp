# Plan — montée du décodeur de film v1.4.0 dans LevelUp (2026-11-16)

Décision de levelup-dd, pilote des releases (2026-11-16) : la montée v1.4.0 (schéma 125) n'entre pas dans la
8.2.1. Elle se prépare sur une branche à part, sans rien pousser sur `release/v8.2.1`, et part dans la release
suivante, créée depuis `main` après la 8.2.1 ; dd prévient pour le rebase. Notes rangées sous la 8.3.0 à
titre provisoire (dd : la 8.3.0 demande de toute façon une régénération, le schéma 125 y entre en même temps,
une seule régénération). Étape C9.2 du plan de la phase C (`PLAN_PHASE_C1_DONNEES_DES_RENDUS_2026-11-14.md`,
dépôt principal). Contrat : skill `plan-execution`.

Lieux : worktree `LevelUp-wt-decodeur140` (branche `feat/monte-decodeur-v140`, depuis
`origin/release/v8.2.1@c96242b8aa`, aucune jonction vers `data/`). Décodeur publié v1.4.0 (591a121), donc
aucun GOWORK : `GOWORK=off` partout. Clone jetable du décodeur au tag pour les fixtures de contrat, hors dépôt.

Aucune UI neuve. Règle pour la première UI qui les lira : tirs, touches et dégâts d'un joueur ne s'affichent
que sous `shotStats[].sufficient`.

## M1 — Module et miroir

- [x] `go get github.com/JGtm/halo-infinite-film-decoder@v1.4.0` (go.mod une ligne, go.sum deux lignes) ;
      `go build ./...` vert.
- [x] Fixtures de contrat 125 (`REPLAY_CONTRACT_UPDATE=1 go test ./replay/ -run ContractFixtures -update`
      dans le clone, racine de données = ce worktree), relance sans drapeaux verte ; 8 fixtures, 4 936 939
      octets (plafond 4 980 736). Aucun des 420 xuids d'origine dans les fixtures ni le manifeste (lus bruts et
      décompressés).
- [x] Instantané `film-source` (`go run ./cmd/film-web-snapshot`) : `document.go.txt`,
      `document_chronicle.entetes.txt`.
- [x] Miroir du schéma 125 : `domain/replaydoc` (`hits.go`, `damage.go`), `service/replayview`, contrat
      OpenAPI, types web, `replayDocumentSchema.ts` (66 clés), `replayNormalize.ts`, polarité de 24 feuilles
      neuves (247 échec, 219 succès, 198 neutre, 5 télémétrie), `contracttest`, plafond de surface 335,
      `damageeffects` et `internal/filmtest` rangés hors couches, exception datée pour l'aide de test
      `PermuterDeuxLignesDeLaTable`, couverture de `degats_*.bin.gz` par l'anonymiseur.
- [x] `resetSeconds` facultatif : sans délai écrit par la variante, ni zone ni jauge (`resolveFlagReturnRule`).
      La clé `reset_seconds` du manifeste du titre, que le décodeur v1.4.0 ne lit plus, est retirée avec son
      chargeur, sa projection et ses tests.
- [x] Décodeur v1.4.1 : l'instantané `golden_catalogue_titre.json` livré en v1.4.0 portait encore
      `"resetSeconds": 0` (l'hôte le vérifie octet par octet). Régénéré par la porte de l'hôte, PR #32 (CI
      verte, squash 037ec4d, arbre identique au commit testé), tag v1.4.1 et release latest, aucune sortie
      changée ; `go.mod` sur v1.4.1 (`go mod verify` vert), lignes intermédiaires v1.4.0 de `go.sum` retirées.
- [x] Gate sur v1.4.1 sans GOWORK : `go build`, `go vet` verts ; `go test ./...` 226 paquets ok, 71 sans test,
      un seul échec (`TestNoNewFrenchLabelLiteral` : le chargeur de la zone de retour tombe à 0 littéral, entrée
      retirée de l'allowlist, relance de `archlint` verte) ; `tsc -b --force` vert ; vitest du rejeu 363 fichiers,
      4 982 tests ; `eslint` des dossiers touchés 0 erreur (2 avertissements existants, `useReplayVehicles.ts`).
      (Sur v1.4.0 : tout vert sauf l'instantané ci-dessus et le titre du bloc des notes, corrigé en « v8.3 ».)

## M2 — Équivalence et gate du corpus

- [x] `replay-equiv` contre le module publié v1.4.1 (racine isolée : config et références du worktree,
      copies des 20 films, fixture des socles du harnais ; archive principale inchangée, 1 614 / 2 267) :
      20 identiques, 0 différent. Les références du module portent déjà les changements de la v1.4.0, aucune
      à refiger.
- [x] `replay-corpus-gate --reference=base --base c96242b8aa --strict` sur la copie du parc (18 témoins) :
      banc de vérité 18 ok sur 18, aucun FAUX, aucune décision à inscrire. Code 1 dû au seul filet, pertes
      instruites :
      - compteurs d'échec neufs du bloc `coverage.hits` (`noShooter`, `noVictim`, `stoppedFrames`,
        `vitalsUnread`) et `coverage.flagCarries.seatsBeyondBand` : absents en 124, ils apparaissent avec le
        schéma 125 ; rien de lu avant n'est perdu ;
      - 084a804d, jauge de retour 185 -> 183 points : les deux points retirés (20 559 et 20 596 ms) tombent
        exactement au milieu de leurs voisins sur une montée linéaire (0,666 / 0,686 / 0,706), aucune
        information perdue ; lecture trame par trame annoncée par la v1.4.0 ;
      - 11de8353, `spawnerByPointKind.unknown` 0 -> 1 avec `map_absent -> established` : la racine de cuisson
        de la base n'a pas le catalogue généré des socles, la tête l'a ; effet de l'environnement du gate, et
        le harnais d'équivalence (catalogue hermétique) rend ce film identique aux références.
      Zone de retour : les trois témoins CTF la gardent, avec le délai de leur variante (60 s, 24 s, 24 s) au
      lieu des 30 s du manifeste du titre. 28 décisions du manifeste (2026-11-15, v1.3.9) signalées « sans
      effet » contre cette base : elles relèvent de la 8.2.1, signalées à sa session. Aucun xuid réel dans le
      manifeste.

## M3 — Livraison de la branche

- [x] Notes de version et changelog FR/EN sous 8.3.0 (provisoire, renommable jusqu'à la sortie).
- [x] `.ai/BACKLOG.md` : entrées film de la phase C statuées (sièges à vacant intercalé et `5da6fd30`
      retirées, résolues ; ancien schéma réduit à deux films ; `ti=40` remplacée par l'anomalie mesurée ;
      Warthog gardée, non vérifiée).
- [x] Étapes de `make gate-push` : équivalence (M2), typecheck et lint web, tests Go (M1) ;
      `golangci-lint --new-from-merge-base=origin/main` 0 problème ; `.ai/thought_log.md`.
- [ ] Commit local (accord du user) ; rebase sur la release suivante au signal de dd ; poussée et CI alors.
- [ ] Régénération du parc : au go du user seulement, une fois, avec celle de la 8.3.0.

## Découvertes

- L'ancien appariement des tirs garde son entrée dans `decfilm`, appelée par le killcollector et l'ingestion
  de la précision de LevelUp (touches à 10 % de l'API : 11 %, contre 81 % pour les suffisants de la v1.4.0).
  Son retrait demande de faire passer ces deux lecteurs sur `shotStats[]` : changement de données
  persistées, hors de cette montée, à planifier.

## Journal

- 2026-11-16 : worktree et branche créés ; module, fixtures 125, instantané, notes et backlog faits.
- 2026-11-16 : miroir par un agent Sonnet, relu (garde-rails : seuls rangements datés des paquets neufs du
  décodeur, aucun affaiblissement) ; repli web sur `rules.flagReturnS` retiré (le décodeur ne publie
  `resetSeconds` que depuis lui, valeur identique) ; v1.4.1 publiée pour l'instantané périmé ; gates verts.
  `replay-equiv`, gate du corpus et `golangci-lint` attendent la fin de la régénération du parc de la 8.2.1
  (levelup-58, `backfill-replay` en cours depuis 08:54), pour la RAM et le verrou de décodage.
- 2026-11-16 : machine libérée par levelup-58 ; équivalence 20/20, lint 0, gate du corpus banc 18/18 ok,
  pertes du filet instruites (M2). Reste : commit (accord du user), rebase au signal de dd, régénération.
- 2026-11-16 : commit local 0c999a32c5 sur accord du user (hooks verts), non poussé. Reste : rebase au
  signal de dd, poussée et CI, régénération au go du user.
- 2026-11-16 : signal de dd (8.2.1 squashée dans `main`, fee334a039) : rebase sur `origin/main` (conflits
  résolus : `go.mod` garde la brique de cartes v1.3.1 de la 8.2.1, changelogs FR/EN). Manifeste du corpus :
  les 25 décisions de la v1.3.9 (« sans effet » contre une base au 124) retirées ; 8 décisions inscrites
  avec l'accord du user du 2026-11-16 (« oui ») : les quatre compteurs d'échec neufs du bloc des touches,
  `seatsBeyondBand`, et les trois mesures de la jauge de 084a804d. Gates refaits contre fee334a039 :
  `go mod verify`, build, vet, `go test ./...` 227 paquets ok ; `tsc -b --force`, vitest du rejeu 4 982,
  eslint 0 erreur ; équivalence 20/20 ; gate du corpus `--strict` code 0, 18 témoins ok, aucune décision
  sans effet (catalogue généré des socles posé dans les deux racines de cuisson). Reste : `release/v8.3.0`
  par dd, rebase dessus, poussée et CI ; régénération au go du user.
